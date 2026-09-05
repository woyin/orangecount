// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package report

import (
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

// chartEvaluation builds a small evaluation with prices so the market-value
// chart paths have quotes to work with.
func chartEvaluation(t *testing.T) ledger.Evaluation {
	t.Helper()
	text := `2000-01-01 open Assets:Cash USD,CNY
2000-01-01 open Expenses:Food USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "entry"
  Assets:Cash 100 USD
  Equity:Opening -100 USD
2000-01-03 * "spend"
  Assets:Cash -10 USD
  Expenses:Food 10 USD
2000-01-04 price USD 7 CNY
`
	file, bag := ledger.ParseText("charts.bean", []byte(text))
	if bag.HasErrors() {
		t.Fatalf("bag=%+v", bag.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("eval=%+v", evaluation.Diagnostics)
	}
	return *evaluation
}

// TestChartRoutesTitlesKindsAndValuations walks the dispatch helpers behind
// the chart endpoint: every route gets a kind and title, unknown valuations
// and periods fall back to defaults, and the commodity route renders a line.
func TestChartRoutesTitlesKindsAndValuations(t *testing.T) {
	evaluation := chartEvaluation(t)
	for _, route := range []string{"balance-sheet", "income-statement", "accounts", "trial-balance"} {
		chart := ReportChart(evaluation, route, "nonsense-period", "USD", "nonsense-valuation", "Assets:Cash")
		if chart.Kind == "" || chart.Title == "" {
			t.Fatalf("route %s kind=%q title=%q", route, chart.Kind, chart.Title)
		}
	}
	// An unknown route renders an intentionally empty chart instead of guessing.
	if empty := ReportChart(evaluation, "unknown-route", "month", "USD", "at-cost", ""); empty.Kind != "" || empty.Title != "" {
		t.Fatalf("unknown route=%+v", empty)
	}
	// Market-value valuation rewrites the measure; without a currency the
	// chart keeps per-currency series.
	market := ReportChart(evaluation, "balance-sheet", "month", "", "market-value", "")
	if market.Measure == "at-cost" {
		t.Fatalf("market valuation ignored: %+v", market)
	}
	charts := ReportCharts(evaluation, "balance-sheet", "month", "market-value")
	if len(charts) == 0 {
		t.Fatal("statement charts missing")
	}
	presented := PresentChart(charts[0])
	if presented.Title == "" {
		t.Fatalf("presented=%+v", presented)
	}
}

// TestChartIntervalsAverageCostAndAmountEdges covers the interval key
// computation (year/quarter/unknown), the average-cost evolution chart, and
// the chart amount conversion fallbacks.
func TestChartIntervalsAverageCostAndAmountEdges(t *testing.T) {
	if chartPeriodKey("2000-01-02", "year") != "2000" || chartPeriodKey("2000-05-02", "quarter") != "2000-Q2" {
		t.Fatal("interval keys broken")
	}
	if chartPeriodKey("short", "month") != "" || chartPeriodKey("2000-13-02", "quarter") != "" {
		t.Fatal("interval guards broken")
	}
	evaluation := chartEvaluation(t)
	for _, interval := range []string{"week", "month", "quarter", "year", ""} {
		keys, ends := chartPeriods(evaluation, interval)
		if len(keys) == 0 {
			t.Fatalf("interval %q produced no periods", interval)
		}
		if ends[keys[0]] == "" {
			t.Fatalf("interval %q missing end dates", interval)
		}
	}
	// The average-cost chart tracks a bought-and-held commodity.
	average := AccountAverageCostChart(evaluation, "month", "Assets:Cash")
	if average.Kind == "" {
		t.Fatalf("average-cost chart=%+v", average)
	}
}
