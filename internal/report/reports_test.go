// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package report

import (
	"math/big"
	"strings"
	"testing"

	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/source"
)

func TestPresentationFormatsApproximateDecimalsWithoutLosingExactValue(t *testing.T) {
	terminating, err := ledger.ParseDecimal("1.25")
	if err != nil {
		t.Fatal(err)
	}
	nonTerminating := ledger.NewDecimal(new(big.Rat).SetFrac64(1, 3))
	result := query.Result{Columns: []string{"balance"}, Rows: []query.Row{
		{"balance": terminating},
		{"balance": nonTerminating},
	}}
	presented := Present(result)
	first, ok := presented.Rows[0]["balance"].(PresentedDecimal)
	if !ok || first.Display != "1.25" || first.Exact != "1.25" || first.Approximate {
		t.Fatalf("terminating presentation=%#v", presented.Rows[0]["balance"])
	}
	second, ok := presented.Rows[1]["balance"].(PresentedDecimal)
	if !ok || second.Display != "0.333333" || second.Exact != "1/3" || !second.Approximate {
		t.Fatalf("non-terminating presentation=%#v", presented.Rows[1]["balance"])
	}
	if strings.Contains(second.Display, "/") {
		t.Fatalf("display leaked rational=%q", second.Display)
	}
	if original, ok := result.Rows[1]["balance"].(ledger.Decimal); !ok || original.String() != "1/3" {
		t.Fatalf("exact result was mutated=%#v", result.Rows[1]["balance"])
	}
}

func TestCoreReportsAreDeterministic(t *testing.T) {
	file, diagnostics := ledger.ParseText("report.bean", []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}
	accounts := Accounts(*evaluation)
	if len(accounts.Rows) != 2 || accounts.Columns[0] != "account" {
		t.Fatalf("accounts=%+v", accounts)
	}
	if len(Journal(*evaluation).Rows) != 2 || len(TrialBalance(*evaluation).Rows) != 2 {
		t.Fatalf("journal/trial balance=%+v/%+v", Journal(*evaluation), TrialBalance(*evaluation))
	}
	trial := TrialBalance(*evaluation)
	for _, row := range trial.Rows {
		balance, ok := row["balance"].(ledger.Decimal)
		if !ok {
			t.Fatalf("trial balance type=%T row=%+v", row["balance"], row)
		}
		account, _ := row["account"].(string)
		want := "1"
		if account == "Equity:Opening" {
			want = "-1"
		}
		if balance.String() != want {
			t.Fatalf("trial balance account=%q balance=%s want=%s rows=%+v", account, balance.String(), want, trial.Rows)
		}
	}
}

func TestErrorsRedactsAbsolutePaths(t *testing.T) {
	evaluation := ledger.Evaluation{Diagnostics: []diagnostic.Diagnostic{
		diagnostic.New("E-CUSTOM", diagnostic.Error, source.Span{StartLine: 1, StartColumn: 1}).WithPath("/private/ledger/main.bean"),
	}}
	result := Errors(evaluation)
	if len(result.Rows) != 1 || result.Rows[0]["path"] != "main.bean" {
		t.Fatalf("errors=%+v", result)
	}
}

func TestBalanceSheetAndTrialBalanceDoNotProjectOpened(t *testing.T) {
	file, diagnostics := ledger.ParseText("opened-report.bean", []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n"))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	for name, result := range map[string]query.Result{"trial": TrialBalance(*evaluation), "balance": BalanceSheet(*evaluation)} {
		for _, column := range result.Columns {
			if column == "opened" {
				t.Fatalf("%s report exposes opened column: %v", name, result.Columns)
			}
		}
	}
}

func TestBalanceSheetBuildsExplicitAncestorTree(t *testing.T) {
	file, diagnostics := ledger.ParseText("tree-report.bean", []byte("2000-01-01 open Assets:Bank:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Bank:Cash 1 USD\n  Equity:Opening -1 USD\n"))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}
	result := BalanceSheet(*evaluation)
	byAccount := map[string]query.Row{}
	for _, row := range result.Rows {
		byAccount[row["account"].(string)] = row
	}
	for account, depth := range map[string]int{"Assets": 0, "Assets:Bank": 1, "Assets:Bank:Cash": 2} {
		row, ok := byAccount[account]
		if !ok || row["_tree_depth"] != depth {
			t.Fatalf("missing or incorrect tree node %q: %+v", account, row)
		}
	}
	if byAccount["Assets"]["_tree_role"] != "aggregate" || byAccount["Assets:Bank"]["_tree_role"] != "aggregate" || byAccount["Assets:Bank:Cash"]["_tree_role"] != "direct" {
		t.Fatalf("tree node roles=%+v", byAccount)
	}
	if byAccount["Assets:Bank"]["_tree_parent"] != "Assets" || byAccount["Assets"]["_tree_has_child"] != true {
		t.Fatalf("tree relationship=%+v", byAccount)
	}
}

func TestAccountTreeSeparatesOwnAndTotalBalance(t *testing.T) {
	// Assets:Bank has a direct posting and Assets:Bank:Cash is a child with its
	// own posting. The parent's own balance must stay separate from the
	// aggregate that includes the child, so the same balance is never reported
	// twice as an ordinary balance row.
	text := `2000-01-01 open Assets:Bank USD
2000-01-01 open Assets:Bank:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "seed"
  Assets:Bank 10 USD
  Equity:Opening -10 USD
2000-01-03 * "cash deposit"
  Assets:Bank:Cash 5 USD
  Assets:Bank -5 USD
`
	file, diagnostics := ledger.ParseText("own-total.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}
	result := BalanceSheet(*evaluation)
	byAccount := map[string]query.Row{}
	for _, row := range result.Rows {
		byAccount[row["account"].(string)] = row
	}
	own := func(name string) string { return byAccount[name]["own_balance"].(ledger.Decimal).String() }
	total := func(name string) string { return byAccount[name]["total_balance"].(ledger.Decimal).String() }
	balance := func(name string) string { return byAccount[name]["balance"].(ledger.Decimal).String() }
	// Assets:Bank carries only its direct balance (10 - 5 = 5); its aggregate
	// includes the child Cash (5 + 5 = 10).
	if own("Assets:Bank") != "5" || total("Assets:Bank") != "10" || balance("Assets:Bank") != "10" {
		t.Fatalf("Assets:Bank own=%s total=%s balance=%s rows=%+v", own("Assets:Bank"), total("Assets:Bank"), balance("Assets:Bank"), byAccount)
	}
	// Assets:Bank:Cash is a leaf: own equals total.
	if own("Assets:Bank:Cash") != "5" || total("Assets:Bank:Cash") != "5" {
		t.Fatalf("Assets:Bank:Cash own=%s total=%s", own("Assets:Bank:Cash"), total("Assets:Bank:Cash"))
	}
	// The synthetic root Assets has no direct posting: own zero, total aggregate.
	if own("Assets") != "0" || total("Assets") != "10" {
		t.Fatalf("Assets own=%s total=%s", own("Assets"), total("Assets"))
	}
}

func TestTrialBalanceTreeIncludesOnlyRealAncestors(t *testing.T) {
	file, diagnostics := ledger.ParseText("trial-tree.bean", []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Income:Salary USD\n2000-01-02 * \"seed\"\n  Assets:Cash 2 USD\n  Income:Salary -2 USD\n"))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	result := TrialBalanceTree(*evaluation)
	byAccount := map[string]query.Row{}
	for _, row := range result.Rows {
		byAccount[row["account"].(string)] = row
	}
	for account, role := range map[string]string{"Assets": "aggregate", "Assets:Cash": "direct", "Income": "aggregate", "Income:Salary": "direct"} {
		if got := byAccount[account]["_tree_role"]; got != role {
			t.Fatalf("account=%q role=%v want=%q row=%+v", account, got, role, byAccount[account])
		}
	}
	if len(TrialBalance(*evaluation).Rows) != 2 {
		t.Fatalf("flat trial balance changed shape: %+v", TrialBalance(*evaluation))
	}
}

func TestReportChartsUsePeriodSeriesAndExactValues(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-01 open Income:Salary USD
2000-01-01 open Expenses:Food USD
2000-01-02 * "seed"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
2000-01-15 * "salary"
  Assets:Cash 100 USD
  Income:Salary -100 USD
2000-02-01 * "food"
  Expenses:Food 25 USD
  Assets:Cash -25 USD
`
	file, diagnostics := ledger.ParseText("charts.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	balances := ReportChart(*evaluation, "balance-sheet", "month", "USD", "at-cost", "")
	if balances.Kind != "line" || balances.Period != "month" || balances.Currency != "USD" || len(balances.Series) != 4 {
		t.Fatalf("balance chart metadata=%+v", balances)
	}
	if balances.Interval != "month" || balances.Measure != "balance" || balances.Availability != "priced" {
		t.Fatalf("balance chart semantics=%+v", balances)
	}
	if got := balances.Series[0].Points[0].Value.String(); got != "110" {
		t.Fatalf("jan assets=%s want=110 points=%+v", got, balances.Series[0].Points)
	}
	if got := balances.Series[0].Points[1].Value.String(); got != "85" {
		t.Fatalf("feb assets=%s want=85 points=%+v", got, balances.Series[0].Points)
	}
	income := ReportChart(*evaluation, "income-statement", "month", "USD", "at-cost", "")
	if income.Kind != "stacked-bar" || len(income.Series) != 3 || income.Series[0].Points[0].Value.String() != "-100" || income.Series[1].Points[1].Value.String() != "25" {
		t.Fatalf("income chart=%+v", income)
	}
	if income.Measure != "flow" || !income.Series[0].Stacked || !income.Series[1].Stacked {
		t.Fatalf("income chart flow/stacked=%+v", income)
	}
	account := ReportChart(*evaluation, "accounts", "month", "USD", "at-cost", "Assets:Cash")
	if len(account.Series) != 1 || len(account.Series[0].Points) != 2 || account.Series[0].Points[1].Value.String() != "85" {
		t.Fatalf("account chart=%+v", account)
	}
	if displayed := PresentChart(balances).Series[0].Points[0].Value; displayed.Exact != "110" || displayed.Display != "110" {
		t.Fatalf("presented chart value=%+v", displayed)
	}
}

func TestReportChartUsesLocalPriceForMarketValue(t *testing.T) {
	text := `2000-01-01 open Assets:Shares SH
2000-01-01 open Equity:Opening USD
2000-01-01 price SH 15 USD
2000-01-02 * "buy"
  Assets:Shares 2 SH {10 USD}
  Equity:Opening -20 USD
`
	file, diagnostics := ledger.ParseText("market-chart.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}
	chart := ReportChart(*evaluation, "balance-sheet", "month", "USD", "market-value", "")
	if got := chart.Series[0].Points[0].Value.String(); got != "30" {
		t.Fatalf("market assets=%s want=30 chart=%+v", got, chart)
	}
	if got := chart.Series[3].Points[0].Value.String(); got != "30" {
		t.Fatalf("market net worth=%s want=30 chart=%+v", got, chart)
	}
	costChart := ReportChart(*evaluation, "balance-sheet", "month", "USD", "at-cost", "")
	if got := costChart.Series[0].Points[0].Value.String(); got != "20" {
		t.Fatalf("cost assets=%s want=20 chart=%+v", got, costChart)
	}
}

func TestTrialBalanceHierarchyChartPresentsLeafAndAggregate(t *testing.T) {
	text := `2000-01-01 open Assets:Bank USD
2000-01-01 open Assets:Bank:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "seed"
  Assets:Bank 10 USD
  Equity:Opening -10 USD
2000-01-03 * "cash"
  Assets:Bank:Cash 5 USD
  Assets:Bank -5 USD
`
	file, diagnostics := ledger.ParseText("trial-chart.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	chart := ReportChart(*evaluation, "trial-balance", "month", "USD", "at-cost", "")
	if chart.Kind != "hierarchy" || len(chart.Nodes) == 0 {
		t.Fatalf("trial chart kind/nodes=%+v", chart)
	}
	if chart.Availability != "priced" || chart.Measure != "balance" {
		t.Fatalf("trial chart semantics=%+v", chart)
	}
	nodeByName := map[string]ChartNode{}
	var walk func(node ChartNode)
	walk = func(node ChartNode) {
		nodeByName[node.Name] = node
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, node := range chart.Nodes {
		walk(node)
	}
	// Aggregate node Assets:Bank carries its subtree total (10); leaf
	// Assets:Bank:Cash carries its own balance (5).
	if got := nodeByName["Assets:Bank"].Value.String(); got != "10" {
		t.Fatalf("aggregate Assets:Bank value=%s want=10", got)
	}
	if got := nodeByName["Assets:Bank:Cash"].Value.String(); got != "5" {
		t.Fatalf("leaf Assets:Bank:Cash value=%s want=5", got)
	}
	presented := PresentChart(chart)
	if len(presented.Nodes) == 0 || presented.Nodes[0].Value.Exact == "" {
		t.Fatalf("presented trial chart=%+v", presented)
	}
}

func TestChartMultiCurrencyUnavailableStatus(t *testing.T) {
	// One postable in a native currency with no local price directive: the
	// requested display currency cannot be valued.
	text := `2000-01-01 open Assets:Gold GOLD
2000-01-01 open Equity:Opening USD
2000-01-02 * "buy gold"
  Assets:Gold 2 GOLD
  Equity:Opening -2 USD
`
	file, diagnostics := ledger.ParseText("multi-chart.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	chart := ReportChart(*evaluation, "balance-sheet", "month", "USD", "market-value", "")
	if chart.Availability != "unavailable-price" {
		t.Fatalf("missing price availability=%q want=unavailable-price chart=%+v", chart.Availability, chart)
	}
	// A price exists but only to another currency, so no conversion to USD.
	withPrice := `2000-01-01 open Assets:Gold GOLD
2000-01-01 open Equity:Opening USD
2000-01-01 price GOLD 5 CNY
2000-01-02 * "buy gold"
  Assets:Gold 2 GOLD
  Equity:Opening -2 USD
`
	file2, diag2 := ledger.ParseText("multi-chart2.bean", []byte(withPrice))
	if diag2.HasErrors() {
		t.Fatalf("parse=%+v", diag2.All())
	}
	evaluation2 := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file2}, []source.FileID{1}, ledger.EvalOptions{})
	chart2 := ReportChart(*evaluation2, "balance-sheet", "month", "USD", "market-value", "")
	if chart2.Availability != "unavailable-currency" {
		t.Fatalf("no-conversion availability=%q want=unavailable-currency chart=%+v", chart2.Availability, chart2)
	}
}

func TestChartPeriodIntervalQuarterYear(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "jan"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
2000-03-02 * "mar"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
2000-06-02 * "jun"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	file, diagnostics := ledger.ParseText("period.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	quarterly := ReportChart(*evaluation, "balance-sheet", "quarter", "USD", "at-cost", "")
	if quarterly.Interval != "quarter" || len(quarterly.Series[0].Points) != 2 {
		t.Fatalf("quarterly interval=%s points=%d", quarterly.Interval, len(quarterly.Series[0].Points))
	}
	yearly := ReportChart(*evaluation, "balance-sheet", "year", "USD", "at-cost", "")
	if yearly.Interval != "year" || len(yearly.Series[0].Points) != 1 {
		t.Fatalf("yearly interval=%s points=%d", yearly.Interval, len(yearly.Series[0].Points))
	}
}

func TestAccountChartNativeModeEmitsPerCurrencySeries(t *testing.T) {
	// A multi-currency account with no display currency requested must emit one
	// running series per native currency instead of silently dropping all data.
	text := `2000-01-01 open Assets:Bank:Main USD
2000-01-01 open Assets:Bank:Main CNY
2000-01-01 open Equity:Opening USD
2000-01-01 open Equity:Opening CNY
2000-01-02 * "usd"
  Assets:Bank:Main 10 USD
  Equity:Opening -10 USD
2000-03-02 * "cny"
  Assets:Bank:Main 100 CNY
  Equity:Opening -100 CNY
`
	file, diagnostics := ledger.ParseText("native-account.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	chart := ReportChart(*evaluation, "accounts", "month", "", "at-cost", "Assets:Bank:Main")
	if chart.Availability != "native-multi" {
		t.Fatalf("native availability=%q want=native-multi", chart.Availability)
	}
	if len(chart.Series) != 2 {
		t.Fatalf("native series len=%d want=2 (USD, CNY), chart=%+v", len(chart.Series), chart)
	}
	labels := map[string]bool{}
	for _, series := range chart.Series {
		labels[series.Label] = true
		if len(series.Points) == 0 {
			t.Fatalf("native series %q has no points", series.Label)
		}
	}
	if !labels["USD"] || !labels["CNY"] {
		t.Fatalf("native series labels=%v want USD and CNY", labels)
	}
}

func TestGlobalReportFiltersKeepStableRows(t *testing.T) {
	result := query.Result{Columns: []string{"date", "account", "narration"}, Rows: []query.Row{
		{"date": "2024-01-02", "account": "Assets:Cash", "narration": "Coffee"},
		{"date": "2024-01-02", "account": "Expenses:Food", "narration": "Coffee"},
		{"date": "2024-02-02", "account": "Assets:Cash", "narration": "Rent"},
	}}
	filtered := Filter(result, Filters{Account: "Assets", Text: "coffee", TimePrefix: "2024-01"})
	if len(filtered.Rows) != 1 || filtered.Rows[0]["account"] != "Assets:Cash" {
		t.Fatalf("filtered=%+v", filtered.Rows)
	}
}

func TestPeriodFilterUsesLatestReportPeriod(t *testing.T) {
	result := query.Result{Columns: []string{"date", "value"}, Rows: []query.Row{
		{"date": "2024-01-01", "value": ledger.Zero()},
		{"date": "2024-02-01", "value": ledger.Zero()},
		{"date": "2024-02-15", "value": ledger.Zero()},
	}}
	filtered := Filter(result, Filters{Period: "month"})
	if len(filtered.Rows) != 2 || filtered.Rows[0]["date"] != "2024-02-01" {
		t.Fatalf("period filtered=%+v", filtered.Rows)
	}
}

func TestJournalFiltersMatchDirectiveMetadata(t *testing.T) {
	result := query.Result{Columns: []string{"flag", "tags", "links", "payee", "narration"}, Rows: []query.Row{
		{"flag": "*", "tags": []string{"food"}, "links": []string{"receipt"}, "payee": "Cafe", "narration": "Lunch"},
		{"flag": "!", "tags": []string{"travel"}, "links": []string{"trip"}, "payee": "Rail", "narration": "Ticket"},
	}}
	filtered := FilterJournal(result, JournalFilters{Flag: "*", Tag: "FOOD", Link: "receipt", Payee: "caf", Narration: "lunch"})
	if len(filtered.Rows) != 1 || filtered.Rows[0]["payee"] != "Cafe" {
		t.Fatalf("filtered=%+v", filtered.Rows)
	}
}

func TestJournalKindFilterMatchesDirectiveKind(t *testing.T) {
	result := query.Result{Columns: []string{"kind", "account"}, Rows: []query.Row{
		{"kind": "transaction", "account": "Assets:Cash"},
		{"kind": "transaction", "account": "Expenses:Food"},
	}}
	filtered := FilterJournal(result, JournalFilters{Kind: "transaction"})
	if len(filtered.Rows) != 2 {
		t.Fatalf("kind=transaction filtered=%+v", filtered.Rows)
	}
	nonMatching := FilterJournal(result, JournalFilters{Kind: "document"})
	if len(nonMatching.Rows) != 0 {
		t.Fatalf("kind=document filtered=%+v", nonMatching.Rows)
	}
	// The kind value must compare even when the filter target does not match.
	flagged := FilterJournal(result, JournalFilters{Kind: "note"})
	if len(flagged.Rows) != 0 {
		t.Fatalf("kind=note filtered=%+v", flagged.Rows)
	}
}

func TestStatisticsHasStableDirectiveCounts(t *testing.T) {
	file, diagnostics := ledger.ParseText("stats.bean", []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	result := Statistics(*evaluation)
	if len(result.Rows) == 0 || result.Columns[0] != "directive" || result.Columns[1] != "count" {
		t.Fatalf("statistics=%+v", result)
	}
}

func TestHoldingsAsOfAndValuationControlsAreDeterministic(t *testing.T) {
	cost, err := ledger.ParseDecimal("10")
	if err != nil {
		t.Fatal(err)
	}
	units, err := ledger.ParseDecimal("2")
	if err != nil {
		t.Fatal(err)
	}
	date := ledger.Date{Raw: "2024-01-01", Year: 2024, Month: 1, Day: 1}
	evaluation := ledger.Evaluation{
		Accounts: map[string]ledger.AccountState{"Assets:Shares": {Name: "Assets:Shares", Positions: []ledger.Position{{Units: units, Currency: "SH", Cost: &ledger.Cost{Number: cost, Currency: "USD", Date: &date}}}}},
		Prices:   map[string][]ledger.PriceQuote{"SH": {{Date: date, Base: "SH", Amount: ledger.NewDecimal(new(big.Rat).SetInt64(15)), Currency: "USD"}}},
	}
	result := HoldingsAt(evaluation, "2024-01-02", "market-value")
	if len(result.Rows) != 1 || result.Rows[0]["value_currency"] != "USD" {
		t.Fatalf("holdings=%+v", result)
	}
	value, ok := result.Rows[0]["value"].(ledger.Decimal)
	if !ok || value.String() != "30" {
		t.Fatalf("value=%#v", result.Rows[0]["value"])
	}
	if len(HoldingsAt(evaluation, "2023-12-31", "market-value").Rows) != 0 {
		t.Fatal("as-of date did not filter future lot")
	}
	unavailable := HoldingsAtCurrency(evaluation, "2024-01-02", "market-value", "CNY")
	if unavailable.Rows[0]["valuation_status"] != "unavailable-currency" {
		t.Fatalf("unsupported currency status=%+v", unavailable.Rows[0])
	}
}
