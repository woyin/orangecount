// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package report

import (
	"orangecount/internal/ledger"
	"strings"
)

type StatementProjection struct {
	ValuationStatus ValuationStatus
	Nodes           []AccountTreeNode
	Availability    string
	Chart           ChartSpec
}

// StatementView projects already evaluated postings. Balance sheets include
// all activity through to; income statements include only [from, to]. The
// same exact posting amounts feed the table and monthly chart. No evaluation
// or filesystem read is performed, and the input snapshot stays immutable.
func StatementView(e ledger.Evaluation, route, from, to, currency, valuation string) StatementProjection {
	if valuation == "" {
		valuation = "at-cost"
	}
	roots := []string{"Assets", "Liabilities", "Equity"}
	labels := []string{"Net worth"}
	if route == "income_statement" {
		roots = []string{"Income", "Expenses"}
		labels = roots
	}
	allowed := map[string]bool{}
	for _, root := range roots {
		allowed[root] = true
	}
	projected := ledger.Evaluation{Accounts: map[string]ledger.AccountState{}}
	for name, state := range e.Accounts {
		if to != "" && state.Opened.Raw > to {
			continue
		}
		state.Balances = map[string]ledger.Decimal{}
		state.Positions = nil
		projected.Accounts[name] = state
	}
	months := map[string]map[string]ledger.Decimal{}
	seriesAvailability := chartAvailability{}
	hasPostings := false
	activity := map[string]bool{}
	for _, posting := range transactionPostings(e) {
		if to != "" && posting.date > to {
			continue
		}
		if route == "income_statement" && from != "" && posting.date < from {
			continue
		}
		root := accountRoot(posting.account)
		if !allowed[root] {
			continue
		}
		hasPostings = true
		inChart := root != "Equity"
		month := posting.date[:7]
		if inChart && months[month] == nil {
			months[month] = map[string]ledger.Decimal{}
		}
		nativeValuation := valuation
		if nativeValuation == "market-value" {
			nativeValuation = "units"
		}
		amount, commodity := nativeChartAmount(posting, nativeValuation)
		state, exists := projected.Accounts[posting.account]
		if !exists {
			continue
		}
		state.Balances[commodity] = state.Balances[commodity].Add(amount)
		activity[posting.account] = true
		if currency != "" {
			var status chartAmountStatus
			amount, status = chartAmount(e, posting, to, currency, valuation)
			if inChart {
				seriesAvailability.observe(status)
			}
			commodity = currency
			if status != amountOK {
				continue
			}
		}
		if inChart {
			months[month][root] = months[month][root].Add(amount)
		}
	}
	nodes := accountTree(projected, roots, true)
	if nodes == nil {
		nodes = []AccountTreeNode{}
	}
	values := map[string][]ChartPoint{}
	for _, label := range labels {
		values[label] = []ChartPoint{}
	}
	running := ledger.Zero()
	for _, month := range sortedKeys(months) {
		if route == "balance_sheet" {
			running = running.Add(months[month]["Assets"]).Add(months[month]["Liabilities"])
			if from == "" || month >= from[:7] {
				values["Net worth"] = append(values["Net worth"], ChartPoint{Date: month, Value: running})
			}
		} else {
			for _, root := range roots {
				values[root] = append(values[root], ChartPoint{Date: month, Value: months[month][root]})
			}
		}
	}
	if route == "balance_sheet" && len(months) > 0 && len(values["Net worth"]) == 0 && len(to) >= 7 {
		values["Net worth"] = append(values["Net worth"], ChartPoint{Date: to[:7], Value: running})
	}
	chart := chartFromSeries(chartKind(strings.ReplaceAll(route, "_", "-")), "", currency, currency, valuation, "month", labels, values)
	chart.Availability = seriesAvailability.resolve(currency)
	availability := "priced"
	if currency == "" {
		chart.Availability = "native-multi"
		availability = "native-multi"
	}
	if len(months) == 0 {
		chart.Availability = "empty"
	}
	if !hasPostings {
		availability = "empty"
	}
	status := annotateStatement(e, nodes, activity, currency, to)
	if currency != "" && (status.Completeness == "partial" || status.Completeness == "unavailable") {
		availability = "unavailable-currency"
	}
	return StatementProjection{Nodes: nodes, Availability: availability, Chart: chart, ValuationStatus: status}
}
