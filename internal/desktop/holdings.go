// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"

	"orangecount/internal/ledger"
	"orangecount/internal/report"
)

const holdingsByteLimit = 16 << 20

func projectHoldings(e ledger.Evaluation, r Request, snapshotID, from, to string) (*report.HoldingsProjection, error) {
	value, err := report.HoldingsViewWithOptions(e, r.Account, r.Scope, from, to, report.HoldingsOptions{Currency: r.Currency, Valuation: r.Valuation, GroupBy: r.GroupBy})
	if err != nil {
		return nil, err
	}
	id := func(logical string) string {
		sum := sha256.Sum256([]byte(snapshotID + "\x00" + logical))
		return hex.EncodeToString(sum[:])
	}
	for i := range value.Groups {
		value.Groups[i].ID = id(value.Groups[i].ID)
		for j := range value.Groups[i].Lots {
			value.Groups[i].Lots[j].ID = id(value.Groups[i].Lots[j].ID)
		}
	}
	for i := range value.Events {
		value.Events[i].ID = id(value.Events[i].ID)
		for j := range value.Events[i].Lots {
			value.Events[i].Lots[j].ID = id(value.Events[i].Lots[j].ID)
		}
		for j := range value.Events[i].MergeLots {
			value.Events[i].MergeLots[j].ID = id(value.Events[i].MergeLots[j].ID)
		}
	}
	for i := range value.AverageHistory {
		value.AverageHistory[i].ID = id(value.AverageHistory[i].ID)
	}
	if r.Sort == "descending" {
		slices.Reverse(value.Groups)
		slices.Reverse(value.Events)
		slices.Reverse(value.AverageHistory)
	}
	limit := r.Limit
	if limit == 0 {
		limit = 100
	}
	section := r.HoldingSection
	if section == "" {
		section = "groups"
	}
	value.Section = section
	// The primary collection honors offset/limit. Secondary previews have their
	// own explicit metadata and remain capped at 100 independently of its offset.
	switch section {
	case "groups":
		value.Groups, value.GroupsPage = holdingPage(value.Groups, r.Offset, limit)
		value.Events, value.EventsPage = holdingPage(value.Events, 0, 100)
		value.AverageHistory, value.HistoryPage = holdingPage(value.AverageHistory, 0, 100)
	case "events":
		value.Groups, value.GroupsPage = holdingPage(value.Groups, 0, 0)
		value.AverageHistory, value.HistoryPage = holdingPage(value.AverageHistory, 0, 0)
		value.Events, value.EventsPage = holdingPage(value.Events, r.Offset, limit)
	case "lots":
		if r.EventID != "" {
			selected := []report.ReductionEvent{}
			for _, event := range value.Events {
				if event.ID == r.EventID {
					selected = append(selected, event)
				}
			}
			if len(selected) == 0 {
				return nil, ErrRequest
			}
			value.Events = selected
			value.EventsPage = report.ProjectionPage{TotalCount: 1, Limit: 1, ReturnedCount: 1}
			value.Groups, value.GroupsPage = holdingPage(value.Groups, 0, 0)
			value.AverageHistory, value.HistoryPage = holdingPage(value.AverageHistory, 0, 0)
			break
		}
		selected := []report.HoldingGroup{}
		for _, group := range value.Groups {
			if group.ID == r.GroupID {
				selected = append(selected, group)
			}
		}
		if len(selected) == 0 {
			return nil, ErrRequest
		}
		value.Groups = selected
		value.GroupsPage = report.ProjectionPage{TotalCount: 1, Limit: 1, ReturnedCount: 1}
		value.Events, value.EventsPage = holdingPage(value.Events, 0, 0)
		value.AverageHistory, value.HistoryPage = holdingPage(value.AverageHistory, 0, 0)
	case "history":
		value.Groups, value.GroupsPage = holdingPage(value.Groups, 0, 0)
		value.Events, value.EventsPage = holdingPage(value.Events, 0, 0)
		if r.GroupID != "" {
			selected := []report.AverageCostSeries{}
			for _, series := range value.AverageHistory {
				if series.ID == r.GroupID {
					selected = append(selected, series)
				}
			}
			if len(selected) == 0 {
				return nil, ErrRequest
			}
			value.AverageHistory = selected
			value.HistoryPage = report.ProjectionPage{TotalCount: 1, Limit: 1, ReturnedCount: 1}
		} else {
			value.AverageHistory, value.HistoryPage = holdingPage(value.AverageHistory, r.Offset, limit)
		}
	}
	for i := range value.Groups {
		group := &value.Groups[i]
		offset, cap := 0, 100
		if section == "lots" {
			offset, cap = r.Offset, limit
			if r.Sort == "descending" {
				slices.Reverse(group.Lots)
			}
		}
		group.Lots, group.LotsPage = holdingPage(group.Lots, offset, cap)
	}
	for i := range value.AverageHistory {
		series := &value.AverageHistory[i]
		offset, cap := max(0, len(series.Points)-100), 100
		if section == "history" && r.GroupID != "" {
			offset, cap = r.Offset, limit
			if r.Sort == "descending" {
				slices.Reverse(series.Points)
			}
		}
		series.Points, series.PointsPage = holdingPage(series.Points, offset, cap)
	}
	// A source sale may itself split into many lots. The metadata makes the
	// preview explicit; consumed quantity and basis still describe the full sale.
	for i := range value.Events {
		event := &value.Events[i]
		offset, cap := 0, 100
		if section == "lots" && r.EventID != "" && r.LotKind != "merged" {
			offset, cap = r.Offset, limit
			if r.Sort == "descending" {
				slices.Reverse(event.Lots)
			}
		}
		event.Lots, event.LotsPage = holdingPage(event.Lots, offset, cap)
		offset, cap = 0, 100
		if section == "lots" && r.EventID != "" && r.LotKind == "merged" {
			offset, cap = r.Offset, limit
			if r.Sort == "descending" {
				slices.Reverse(event.MergeLots)
			}
		}
		event.MergeLots, event.MergeLotsPage = holdingPage(event.MergeLots, offset, cap)
	}
	return value, nil
}

func holdingPage[T any](items []T, offset, limit int) ([]T, report.ProjectionPage) {
	total := len(items)
	start := min(offset, total)
	end := start + min(limit, total-start)
	result := make([]T, end-start)
	copy(result, items[start:end])
	return result, report.ProjectionPage{TotalCount: total, Offset: offset, Limit: limit, ReturnedCount: len(result), HasMore: end < total, Truncated: len(result) < total}
}
