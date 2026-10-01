// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package report

import (
	"orangecount/internal/ledger"
	"sort"
)

type AverageCostPoint struct {
	Date        string            `json:"date"`
	Quantity    PresentedDecimal  `json:"quantity"`
	CostBasis   PresentedDecimal  `json:"cost_basis"`
	AverageCost *PresentedDecimal `json:"average_cost"`
	Value       *float64          `json:"value"`
	Segment     int               `json:"segment"`
}

type AverageCostSeries struct {
	ID           string             `json:"id"`
	Account      string             `json:"account"`
	Accounts     []string           `json:"accounts"`
	Commodity    string             `json:"commodity"`
	CostCurrency string             `json:"cost_currency"`
	Points       []AverageCostPoint `json:"points"`
	PointsPage   ProjectionPage     `json:"points_page"`
}

// Signed M changes conserve aggregate basis, but checkpoints are emitted only
// after a complete transaction. Liquidation emits an explicit missing average;
// a later reacquisition starts a new line segment, with its own actual basis.
func averageInventoryHistory(e ledger.Evaluation, changes []ledger.BookedInventoryChange, account, scope, from, to, groupBy string) []AverageCostSeries {
	result := []AverageCostSeries{}
	series := map[string]*AverageCostSeries{}
	quantities, bases := map[string]ledger.Decimal{}, map[string]ledger.Decimal{}
	segments := map[string]int{}
	appendPoint := func(key, date string) {
		if date < from {
			return
		}
		item := series[key]
		point := AverageCostPoint{Date: date, Quantity: FormatDecimal(quantities[key]), CostBasis: FormatDecimal(bases[key]), Segment: segments[key]}
		if !quantities[key].IsZero() {
			average := FormatDecimal(bases[key].Quo(quantities[key]))
			point.AverageCost, point.Value = &average, chartCoordinate(&average)
		}
		if n := len(item.Points); n > 0 && item.Points[n-1].Date == date {
			// Preserve transaction-level liquidation even when the display
			// coalesces all activity to one end-of-day point.
			item.Points[n-1] = point
		} else {
			item.Points = append(item.Points, point)
		}
		if point.AverageCost != nil && point.Value == nil {
			segments[key]++
		}
	}
	for cursor := 0; cursor < len(changes); {
		index := changes[cursor].EntryIndex
		date := e.Entries[index].Date.Raw
		changed := map[string]bool{}
		before := map[string]ledger.Decimal{}
		for cursor < len(changes) && changes[cursor].EntryIndex == index {
			change := changes[cursor]
			cursor++
			cost := change.Position.Cost
			if cost == nil || !holdingAccountMatches(change.Account, account, scope) {
				continue
			}
			currency := cost.Currency
			key := holdingGroupIdentity(change.Account, change.Position.Currency, &currency, groupBy)
			if series[key] == nil {
				series[key] = &AverageCostSeries{ID: key, Account: change.Account, Accounts: []string{}, Commodity: change.Position.Currency, CostCurrency: currency, Points: []AverageCostPoint{}}
				if groupBy != "account" {
					series[key].Account = ""
				}
			}
			series[key].Accounts = appendUniqueString(series[key].Accounts, change.Account)
			if !changed[key] {
				before[key] = quantities[key]
			}
			quantities[key] = quantities[key].Add(change.Position.Units)
			bases[key] = bases[key].Add(change.Position.Units.Mul(cost.Number))
			changed[key] = true
		}
		for key := range changed {
			appendPoint(key, date)
			if !before[key].IsZero() && quantities[key].IsZero() {
				segments[key]++
			}
		}
	}
	for _, key := range sortedKeys(series) {
		sort.Strings(series[key].Accounts)
		if points := series[key].Points; len(points) == 0 || points[len(points)-1].Date != to {
			appendPoint(key, to)
		}
		result = append(result, *series[key])
	}
	return result
}
