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

func TestJournalBetweenHonorsEachInclusiveRangeBoundary(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "first"
  Assets:Cash 1 USD
  Equity:Opening -1 USD
2000-01-03 * "second"
  Assets:Cash 2 USD
  Equity:Opening -2 USD
2000-01-04 * "third"
  Assets:Cash 3 USD
  Equity:Opening -3 USD
`
	file, diagnostics := ledger.ParseText("journal-range.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	date := func(raw string) *ledger.Date { return &ledger.Date{Raw: raw} }
	for _, test := range []struct {
		from, to *ledger.Date
		want     int
	}{
		{date("2000-01-03"), nil, 4}, {nil, date("2000-01-03"), 4}, {date("2000-01-03"), date("2000-01-03"), 2}, {date("2000-02-01"), nil, 0},
	} {
		if result := JournalBetween(*evaluation, test.from, test.to); len(result.Rows) != test.want {
			t.Errorf("JournalBetween(%+v, %+v) rows=%d want=%d", test.from, test.to, len(result.Rows), test.want)
		}
	}
}

func TestReportSetIncludesEveryCoreProjectionAndSafeGraphPaths(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "seed"
  Assets:Cash 1 USD
  Equity:Opening -1 USD
2000-01-03 price USD 1.5 EUR
2000-01-04 event "status" "ok"
2000-01-05 document Assets:Cash "receipt.pdf" #proof ^doc
`
	file, diagnostics := ledger.ParseText("report-set.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}
	all := All(*evaluation)
	for name, result := range map[string]query.Result{
		"accounts": all.Accounts, "journal": all.Journal, "trial": all.TrialBalance, "balance": all.BalanceSheet, "income": all.IncomeStatement, "holdings": all.Holdings, "prices": all.Prices, "events": all.Events, "documents": all.Documents, "statistics": all.Statistics, "errors": all.Errors,
	} {
		if result.Columns == nil {
			t.Fatalf("%s returned nil columns", name)
		}
	}
	if len(all.Prices.Rows) != 1 || len(all.Events.Rows) != 1 || len(all.Documents.Rows) != 1 {
		t.Fatalf("special reports prices=%+v events=%+v documents=%+v", all.Prices, all.Events, all.Documents)
	}
	graphFile := source.NewSourceFile(1, "/private/ledger/main.bean", []byte(text))
	graph := &source.Graph{Entry: 1, Files: map[source.FileID]*source.SourceFile{1: graphFile}, ByPath: map[string]source.FileID{graphFile.Path: 1}, Order: []source.FileID{1}}
	withDiagnostic := *evaluation
	withDiagnostic.Diagnostics = []diagnostic.Diagnostic{diagnostic.New("E-CUSTOM", diagnostic.Error, source.Span{File: 1, StartLine: 2, StartColumn: 1}).WithPath(graphFile.Path)}
	if got := ErrorsWithGraph(withDiagnostic, graph); len(got.Rows) != 1 || got.Rows[0]["path"] != "main.bean" {
		t.Fatalf("graph errors=%+v", got)
	}
}

func TestReportDateAndPresentationHelpersHandleEdgeValues(t *testing.T) {
	for _, tc := range []struct {
		date, anchor, period string
		want                 bool
	}{
		{"2024-02-03", "2024-12-01", "year", true}, {"2024-02-03", "2024-03-01", "month", false}, {"2024-02-03", "2024-03-01", "quarter", true}, {"2024-02-03", "2024-04-01", "quarter", false}, {"bad", "2024-04-01", "quarter", true},
	} {
		if got := samePeriod(tc.date, tc.anchor, tc.period); got != tc.want {
			t.Errorf("samePeriod(%q, %q, %q)=%v", tc.date, tc.anchor, tc.period, got)
		}
	}
	values := []ledger.Decimal{ledger.NewDecimal(big.NewRat(1, 3))}
	presented, ok := presentValue(values).([]PresentedDecimal)
	if !ok || len(presented) != 1 || !presented[0].Approximate {
		t.Fatalf("presented decimals=%+v", presented)
	}
	if unchanged := presentValue("text"); unchanged != "text" {
		t.Fatalf("non-decimal presentation=%v", unchanged)
	}
}

func TestNativeReportChartsKeepIncomeAndExpenseCurrenciesSeparate(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Income:Salary USD
2000-01-01 open Expenses:Food USD
2000-01-15 * "salary"
  Assets:Cash 10 USD
  Income:Salary -10 USD
2000-02-15 * "food"
  Assets:Cash -3 USD
  Expenses:Food 3 USD
`
	file, diagnostics := ledger.ParseText("native-charts.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	charts := ReportCharts(*evaluation, "income-statement", "month", "units")
	if len(charts) != 3 || charts[0].Kind != ChartBar || charts[0].Measure != "flow" {
		t.Fatalf("native flow charts=%+v", charts)
	}
	if len(charts[0].Series) != 1 || len(charts[0].Series[0].Points) != 2 {
		t.Fatalf("net profit series=%+v", charts[0].Series)
	}
	if unknown := ReportCharts(*evaluation, "unknown", "month", "units"); unknown != nil {
		t.Fatalf("unknown route charts=%+v", unknown)
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

func TestTimeRangeFilterUsesHalfOpenBounds(t *testing.T) {
	result := query.Result{Columns: []string{"date", "narration"}, Rows: []query.Row{
		{"date": "2024-03-31", "narration": "Before Q2"},
		{"date": "2024-04-01", "narration": "Q2 start"},
		{"date": "2024-06-30", "narration": "Q2 end"},
		{"date": "2024-07-01", "narration": "After Q2"},
	}}
	filtered := Filter(result, Filters{TimeBegin: "2024-04-01", TimeEnd: "2024-07-01"})
	if len(filtered.Rows) != 2 || filtered.Rows[0]["narration"] != "Q2 start" || filtered.Rows[1]["narration"] != "Q2 end" {
		t.Fatalf("range filtered=%+v", filtered.Rows)
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

func TestPostingsPerAccountSortsByCountThenAccount(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Bank USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "seed"
  Assets:Cash 1 USD
  Equity:Opening -1 USD
2000-01-03 * "transfer"
  Assets:Cash 2 USD
  Assets:Bank -2 USD
`
	file, diagnostics := ledger.ParseText("postings-stats.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}
	result := PostingsPerAccount(*evaluation)
	if len(result.Columns) != 2 || result.Columns[0] != "account" || result.Columns[1] != "postings" {
		t.Fatalf("columns=%+v", result.Columns)
	}
	want := []struct {
		account string
		count   int
	}{
		{"Assets:Cash", 2},
		{"Assets:Bank", 1},
		{"Equity:Opening", 1},
	}
	if len(result.Rows) != len(want) {
		t.Fatalf("rows=%+v", result.Rows)
	}
	for i, expected := range want {
		if result.Rows[i]["account"] != expected.account || result.Rows[i]["postings"] != expected.count {
			t.Fatalf("row %d=%+v want %+v", i, result.Rows[i], expected)
		}
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

func TestValuedBalancesConvertsLotsWithoutTouchingPlainCurrency(t *testing.T) {
	dec := func(raw string) ledger.Decimal {
		value, err := ledger.ParseDecimal(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		return value
	}
	date := ledger.Date{Raw: "2024-01-01", Year: 2024, Month: 1, Day: 1}
	// The account holds two share lots at different costs plus a plain cash
	// balance in the same currency the lots cost out to.
	evaluation := ledger.Evaluation{
		Accounts: map[string]ledger.AccountState{"Assets:Broker": {
			Name:     "Assets:Broker",
			Balances: map[string]ledger.Decimal{"SH": dec("30"), "USD": dec("5")},
			Positions: []ledger.Position{
				{Units: dec("10"), Currency: "SH", Cost: &ledger.Cost{Number: dec("10"), Currency: "USD", Date: &date}},
				{Units: dec("20"), Currency: "SH", Cost: &ledger.Cost{Number: dec("12"), Currency: "USD", Date: &date}},
			},
		}},
		Prices: map[string][]ledger.PriceQuote{"SH": {{Date: date, Base: "SH", Amount: dec("15"), Currency: "USD"}}},
	}

	units := ValuedBalances(evaluation, "Assets:Broker", "units")
	if units["SH"].String() != "30" || units["USD"].String() != "5" {
		t.Fatalf("units valuation must not convert: %v", display(units))
	}

	// 10*10 + 20*12 = 340, plus the untouched 5 USD of cash.
	atCost := ValuedBalances(evaluation, "Assets:Broker", "at-cost")
	if atCost["USD"].String() != "345" {
		t.Fatalf("at-cost USD=%v want 345", display(atCost))
	}
	if _, ok := atCost["SH"]; ok {
		t.Fatalf("fully converted commodity must not linger: %v", display(atCost))
	}

	// 30*15 = 450 at the latest quote, plus the same 5 USD of cash.
	atMarket := ValuedBalances(evaluation, "Assets:Broker", "market-value")
	if atMarket["USD"].String() != "455" {
		t.Fatalf("market-value USD=%v want 455", display(atMarket))
	}
}

func TestValuedBalancesKeepsUnquotedLotsInTheirOwnCommodity(t *testing.T) {
	dec := func(raw string) ledger.Decimal {
		value, err := ledger.ParseDecimal(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		return value
	}
	date := ledger.Date{Raw: "2024-01-01", Year: 2024, Month: 1, Day: 1}
	// No price quote exists for FUND, so market-value must leave the lot in
	// its own commodity rather than inventing a conversion.
	evaluation := ledger.Evaluation{
		Accounts: map[string]ledger.AccountState{"Assets:Fund": {
			Name:      "Assets:Fund",
			Balances:  map[string]ledger.Decimal{"FUND": dec("100")},
			Positions: []ledger.Position{{Units: dec("100"), Currency: "FUND", Cost: &ledger.Cost{Number: dec("2"), Currency: "CNY", Date: &date}}},
		}},
	}
	atMarket := ValuedBalances(evaluation, "Assets:Fund", "market-value")
	if atMarket["FUND"].String() != "100" {
		t.Fatalf("unquoted lot must stay in its commodity: %v", display(atMarket))
	}
	if _, ok := atMarket["CNY"]; ok {
		t.Fatalf("unquoted lot must not produce a converted amount: %v", display(atMarket))
	}
	atCost := ValuedBalances(evaluation, "Assets:Fund", "at-cost")
	if atCost["CNY"].String() != "200" {
		t.Fatalf("at-cost CNY=%v want 200", display(atCost))
	}
}

func display(values map[string]ledger.Decimal) map[string]string {
	out := make(map[string]string, len(values))
	for currency, amount := range values {
		out[currency] = amount.String()
	}
	return out
}

func TestReportChartsPlotEveryCurrencyWithoutConversion(t *testing.T) {
	// Two currencies with no price quote between them. A single-display-currency
	// chart has to drop one of them; the per-currency chart set must keep both.
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Yen JPY
2000-01-01 open Equity:Opening
2000-01-02 * "usd opening"
  Assets:Cash 100 USD
  Equity:Opening -100 USD
2000-01-03 * "jpy opening"
  Assets:Yen 5000 JPY
  Equity:Opening -5000 JPY
`
	file, parseDiagnostics := ledger.ParseText("multi-currency.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%s diagnostics=%+v", evaluation, evaluation.Diagnostics)
	}

	charts := ReportCharts(*evaluation, "balance-sheet", "month", "at-cost")
	if len(charts) == 0 {
		t.Fatal("expected at least one balance-sheet chart")
	}
	byTitle := map[string]ChartSpec{}
	for _, chart := range charts {
		byTitle[chart.Title] = chart
	}
	assets, ok := byTitle["Assets"]
	if !ok {
		t.Fatalf("no Assets chart in %v", byTitle)
	}
	labels := map[string]ChartSeries{}
	for _, series := range assets.Series {
		labels[series.Label] = series
	}
	if len(labels) != 2 || labels["USD"].Label == "" || labels["JPY"].Label == "" {
		t.Fatalf("want one series per currency, got %+v", assets.Series)
	}
	if assets.Availability != "" {
		t.Fatalf("per-currency charts never need a conversion quote, got %q", assets.Availability)
	}
	// Series must be dense and equal length: the renderer positions points by
	// index, so a short series would be drawn against the wrong dates.
	if len(labels["USD"].Points) != len(labels["JPY"].Points) {
		t.Fatalf("series lengths differ: USD=%d JPY=%d", len(labels["USD"].Points), len(labels["JPY"].Points))
	}
	for _, series := range assets.Series {
		last := series.Points[len(series.Points)-1]
		want := map[string]string{"USD": "100", "JPY": "5000"}[series.Label]
		if last.Value.String() != want {
			t.Fatalf("%s final balance=%s want %s", series.Label, last.Value.String(), want)
		}
	}
}

func TestReportChartsValueLotsAtCost(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH
2000-01-02 * "buy"
  Assets:Shares 10 SH {7 USD}
  Assets:Cash -70 USD
`
	file, parseDiagnostics := ledger.ParseText("lots.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	charts := ReportCharts(*evaluation, "balance-sheet", "month", "at-cost")
	for _, chart := range charts {
		if chart.Title != "Assets" {
			continue
		}
		// The shares cost exactly what the cash posting paid, so at cost the
		// account nets to zero USD and no SH series remains.
		for _, series := range chart.Series {
			if series.Label == "SH" {
				t.Fatalf("at-cost chart must not keep raw lot units: %+v", series)
			}
		}
	}
	units := ReportCharts(*evaluation, "balance-sheet", "month", "units")
	found := false
	for _, chart := range units {
		if chart.Title != "Assets" {
			continue
		}
		for _, series := range chart.Series {
			if series.Label == "SH" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("units chart must keep the raw lot units series")
	}
}

func TestTrialBalanceChartEmitsEachAccountOnce(t *testing.T) {
	// One account holding several commodities produces one report row per
	// (account, currency). Registering a child once per row repeated it under
	// its parent for every currency, and because that repetition compounds at
	// each level the emitted hierarchy grew multiplicatively rather than
	// linearly with the account count.
	text := `2000-01-01 open Assets:Broker:Shares
2000-01-01 open Assets:Broker:Cash USD
2000-01-01 open Equity:Opening
2000-01-02 * "three commodities in one account"
  Assets:Broker:Shares 1 AAA {1 USD}
  Assets:Broker:Shares 1 BBB {1 USD}
  Assets:Broker:Shares 1 CCC {1 USD}
  Equity:Opening -3 USD
2000-01-03 * "cash"
  Assets:Broker:Cash 5 USD
  Equity:Opening -5 USD
`
	file, parseDiagnostics := ledger.ParseText("dup.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	chart := ReportChart(*evaluation, "trial-balance", "month", "USD", "at-cost", "")

	counts := map[string]int{}
	var walk func(nodes []ChartNode)
	walk = func(nodes []ChartNode) {
		for _, node := range nodes {
			counts[node.Name]++
			walk(node.Children)
		}
	}
	walk(chart.Nodes)
	if len(counts) == 0 {
		t.Fatal("hierarchy chart produced no nodes")
	}
	for name, seen := range counts {
		if seen != 1 {
			t.Fatalf("account %q appears %d times in the hierarchy; each account must appear once", name, seen)
		}
	}
}

func TestHoldingsAggregateGroupsAndSumsWithinCostCurrency(t *testing.T) {
	decimal := func(raw string) ledger.Decimal {
		value, err := ledger.ParseDecimal(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		return value
	}
	flat := query.Result{
		Columns: []string{"account", "currency", "units", "cost_currency", "cost"},
		Rows: []query.Row{
			{"account": "Assets:Cash:Reserve01", "currency": "FUND", "units": decimal("2"), "cost_currency": "USD", "cost": decimal("10")},
			{"account": "Assets:Cash:Reserve01", "currency": "FUND", "units": decimal("3"), "cost_currency": "USD", "cost": decimal("12")},
			{"account": "Assets:Broker:Index01", "currency": "FUND", "units": decimal("1"), "cost_currency": "EUR", "cost": decimal("7")},
			{"account": "Assets:Broker:Index01", "currency": "USD", "units": decimal("5")},
		},
	}

	byAccount := HoldingsAggregate(flat, "by_account")
	if !strings.Contains(strings.Join(byAccount.Columns, ","), "average_cost") {
		t.Fatalf("by_account columns=%v, want average_cost", byAccount.Columns)
	}
	if len(byAccount.Rows) != 3 {
		t.Fatalf("by_account produced %d rows, want 3 groups", len(byAccount.Rows))
	}
	first := byAccount.Rows[0]
	if first["account"] != "Assets:Broker:Index01" || first["currency"] != "FUND" {
		t.Fatalf("expected sorted first group Assets:Broker:Index01/FUND, got %v/%v", first["account"], first["currency"])
	}
	reserve := byAccount.Rows[2]
	if got := reserve["units"].(ledger.Decimal).String(); got != decimal("5").String() {
		t.Fatalf("reserve units = %s, want 5", got)
	}
	if got := reserve["book_value"].(ledger.Decimal).String(); got != decimal("56").String() {
		t.Fatalf("reserve book_value = %s, want 56 (2*10 + 3*12)", got)
	}
	if average := reserve["average_cost"].(ledger.Decimal); average.Mul(decimal("5")).Cmp(decimal("56")) != 0 {
		t.Fatalf("reserve average_cost*units = %s, want 56", average.Mul(decimal("5")).String())
	}

	byCurrency := HoldingsAggregate(flat, "by_currency")
	var fundUSD query.Row
	for _, row := range byCurrency.Rows {
		if row["currency"] == "FUND" && row["cost_currency"] == "USD" {
			fundUSD = row
		}
	}
	if fundUSD == nil {
		t.Fatal("by_currency missing FUND/USD group")
	}
	average := fundUSD["average_cost"].(ledger.Decimal)
	if average.Mul(decimal("5")).Cmp(decimal("56")) != 0 {
		t.Fatalf("average_cost*units = %s, want 56", average.Mul(decimal("5")).String())
	}

	byRoot := HoldingsAggregate(flat, "by_root_account")
	if !strings.Contains(strings.Join(byRoot.Columns, ","), "average_cost") {
		t.Fatalf("by_root_account columns=%v, want average_cost", byRoot.Columns)
	}
	roots := map[string]bool{}
	var rootFundUSD query.Row
	for _, row := range byRoot.Rows {
		roots[row["root_account"].(string)] = true
		if row["currency"] == "FUND" && row["cost_currency"] == "USD" {
			rootFundUSD = row
		}
	}
	if len(byRoot.Rows) != 3 || !roots["Assets"] {
		t.Fatalf("by_root_account rows=%d roots=%v, want 3 rows under Assets", len(byRoot.Rows), roots)
	}
	if rootFundUSD == nil || rootFundUSD["average_cost"].(ledger.Decimal).Mul(decimal("5")).Cmp(decimal("56")) != 0 {
		t.Fatalf("by_root_account FUND/USD average_cost=%v, want 56/5", rootFundUSD["average_cost"])
	}

	byCommodity := HoldingsAggregate(flat, "by_commodity")
	if !strings.Contains(strings.Join(byCommodity.Columns, ","), "average_cost") {
		t.Fatalf("by_commodity columns=%v, want average_cost", byCommodity.Columns)
	}
	for _, row := range byCommodity.Rows {
		if row["currency"] == "FUND" {
			if _, hasBook := row["book_value"]; hasBook {
				t.Fatal("by_commodity FUND group must omit book_value when cost currencies are mixed")
			}
			if _, hasAverage := row["average_cost"]; hasAverage {
				t.Fatal("by_commodity FUND group must omit average_cost when cost currencies are mixed")
			}
			if got := row["units"].(ledger.Decimal).String(); got != decimal("6").String() {
				t.Fatalf("by_commodity FUND units = %s, want 6", got)
			}
		}
	}

	byCostCurrency := HoldingsAggregate(flat, "by_cost_currency")
	if !strings.Contains(strings.Join(byCostCurrency.Columns, ","), "average_cost") {
		t.Fatalf("by_cost_currency columns=%v, want average_cost", byCostCurrency.Columns)
	}
	if len(byCostCurrency.Rows) != 3 {
		t.Fatalf("by_cost_currency produced %d rows, want 3 groups", len(byCostCurrency.Rows))
	}
	byCost := map[string]query.Row{}
	for _, row := range byCostCurrency.Rows {
		byCost[row["cost_currency"].(string)] = row
	}
	if got := byCost["USD"]["units"].(ledger.Decimal).String(); got != decimal("5").String() {
		t.Fatalf("by_cost_currency USD units = %s, want 5", got)
	}
	if got := byCost["USD"]["book_value"].(ledger.Decimal).String(); got != decimal("56").String() {
		t.Fatalf("by_cost_currency USD book_value = %s, want 56", got)
	}
	if average := byCost["USD"]["average_cost"].(ledger.Decimal); average.Mul(decimal("5")).Cmp(decimal("56")) != 0 {
		t.Fatalf("by_cost_currency USD average_cost*units = %s, want 56", average.Mul(decimal("5")).String())
	}
	if got := byCost["EUR"]["book_value"].(ledger.Decimal).String(); got != decimal("7").String() {
		t.Fatalf("by_cost_currency EUR book_value = %s, want 7", got)
	}
	if _, hasBook := byCost[""]["book_value"]; hasBook {
		t.Fatal("by_cost_currency group without costs must omit book_value")
	}

	if passthrough := HoldingsAggregate(flat, ""); len(passthrough.Rows) != len(flat.Rows) {
		t.Fatal("empty aggregation must return the flat result untouched")
	}
}

func TestAccountAverageCostChartReplaysBookedLots(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH "AVERAGE"
2000-01-01 open Income:Gains USD
2000-01-05 * "buy first lot"
  Assets:Shares 100 SH {10 USD}
  Assets:Cash -1000 USD
2000-02-05 * "buy second lot"
  Assets:Shares 200 SH {12 USD}
  Assets:Cash -2400 USD
2000-03-05 * "sell part"
  Assets:Shares -100 SH {} @ 15 USD
  Assets:Cash 1500 USD
  Income:Gains
`
	file, diagnostics := ledger.ParseText("average-cost-history.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}
	chart := AccountAverageCostChart(*evaluation, "month", "Assets:Shares")
	if chart.Kind != ChartLine || chart.Measure != "average-cost" || chart.Title != "Average cost evolution" || len(chart.Series) != 1 {
		t.Fatalf("average-cost chart=%+v", chart)
	}
	series := chart.Series[0]
	if series.Label != "SH (USD)" || len(series.Points) != 3 {
		t.Fatalf("average-cost series=%+v", series)
	}
	expected, err := ledger.ParseDecimal("34/3")
	if err != nil {
		t.Fatal(err)
	}
	if series.Points[0].Date != "2000-01" || series.Points[0].Value.String() != "10" || !series.Points[1].Value.Equal(expected) || !series.Points[2].Value.Equal(expected) {
		t.Fatalf("average-cost points=%+v, want 10, 34/3, 34/3", series.Points)
	}
}
