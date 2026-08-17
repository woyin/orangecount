// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dialect_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orangecount/internal/dialect"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

// mixedV3Ledger exercises both sides of the dialectize filter: trivially
// representable daily transactions and entries the dialect cannot express.
const mixedV3Ledger = `2000-01-01 open Assets:WeChat USD
2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Broker:AAPL AAPL
2000-01-01 open Assets:Broker:Cash USD
2000-01-01 open Expenses:Food USD
2000-01-01 open Expenses:Travel USD
2000-01-01 open Expenses:Travel:Rent USD
2000-01-01 open Income:Salary USD
option "operating_currency" "USD"

2026-08-12 * "美团" "工作午餐"
  Assets:WeChat -28 USD
  Expenses:Food 28 USD

2026-08-13 * "地铁" "通勤"
  Assets:Cash -6 USD
  Expenses:Travel 6 USD

2026-08-14 * "" "空收款人有注释可转换"
  Assets:Cash -1 USD
  Expenses:Food 1 USD

2026-08-15 * "三腿" "拆分记账"
  Assets:Cash -30 USD
  Expenses:Food 20 USD
  Expenses:Travel 10 USD

2026-08-16 * "买入" "带成本批次"
  Assets:Broker:AAPL 10 AAPL {101.00 USD}
  Assets:Broker:Cash -1010 USD

2026-08-17 ! "待核对" "旗标保留"
  Assets:WeChat -15 USD
  Expenses:Food 15 USD

2026-08-18 * "房贷" "等额反向"
  Assets:Cash -3000 USD
  Expenses:Travel:Rent 3000 USD

2026-08-19 price AAPL 150.00 USD
`

func writeFile(t *testing.T, name, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// dialectizeFile rewrites one ledger file through the public filter.
func dialectizeFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, bag := ledger.ParseText(path, data)
	if bag.HasErrors() {
		t.Fatalf("input does not parse: %v", bag.All())
	}
	edits, _ := dialect.Dialectize(file)
	out := dialect.ApplyEdits(data, edits)
	return writeFile(t, "dialect.bean", string(out))
}

// exportFile renders the pure v3 export of a (possibly dialect) ledger.
func exportFile(t *testing.T, path string) string {
	t.Helper()
	graph, err := source.LoadGraph(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, bag := ledger.ParseGraph(graph)
	if bag.HasErrors() {
		t.Fatalf("export input does not parse: %v", bag.All())
	}
	output, diags := dialect.ExportText(graph, parsed)
	for _, d := range diags {
		if d.Severity == "error" {
			t.Fatalf("dialect error in export: %s %s", d.Code, d.Message)
		}
	}
	data, ok := output[graph.Entry]
	if !ok {
		t.Fatal("export missing entry file")
	}
	return writeFile(t, "export.bean", string(data))
}

// balancesOf builds a snapshot and flattens account balances for comparison.
func balancesOf(t *testing.T, path string) map[string]string {
	t.Helper()
	result := snapshot.Build(path)
	if result.Snapshot == nil {
		var codes []string
		for _, d := range result.Diagnostics {
			codes = append(codes, d.Code)
		}
		t.Fatalf("snapshot nil: %v", codes)
	}
	out := map[string]string{}
	for account, state := range result.Snapshot.Evaluation().Accounts {
		for currency, balance := range state.Balances {
			out[account+"/"+currency] = balance.String()
		}
	}
	return out
}

// TestDialectizePreservesBalances is ADR-0045 property one: for any v3
// ledger, dialectize then rebuild must yield identical account balances.
func TestDialectizePreservesBalances(t *testing.T) {
	original := writeFile(t, "v3.bean", mixedV3Ledger)
	dialectVersion := dialectizeFile(t, original)

	before := balancesOf(t, original)
	after := balancesOf(t, dialectVersion)
	if len(before) == 0 {
		t.Fatal("fixture produced no balances")
	}
	if fmt.Sprint(before) != fmt.Sprint(after) {
		t.Fatalf("balances diverged:\nbefore=%v\nafter=%v", before, after)
	}
}

// TestRoundTripReachesFixpoint is ADR-0045 property two: exporting a dialect
// ledger, dialectizing that export, and exporting again must be byte-stable.
func TestRoundTripReachesFixpoint(t *testing.T) {
	dialectSource := writeFile(t, "d0.bean", `2000-01-01 open Assets:WeChat USD
2000-01-01 open Assets:Cash USD
2000-01-01 open Expenses:Food USD
option "operating_currency" "USD"
2026-08-12 28 USD @WeChat -> @Food : 工作午餐
2026-08-13 ! 15 USD @Cash -> @Food "地铁"
2026-08-14 3000 USD @Cash -> @Food "租金" : 月租 #home
6 USD @Cash -> @Food "地铁" : 通勤
`)
	first := exportFile(t, dialectSource)
	dialectized := dialectizeFile(t, first)
	second := exportFile(t, dialectized)
	firstText, secondText := readFile(t, first), readFile(t, second)
	if firstText != secondText {
		t.Fatalf("round trip is not a fixpoint:\n--- first ---\n%s\n--- second ---\n%s", firstText, secondText)
	}
	if !strings.Contains(firstText, `"工作午餐"`) {
		t.Fatalf("narration lost in export:\n%s", firstText)
	}
	if !strings.Contains(firstText, `! "地铁"`) {
		t.Fatalf("flag or payee lost in export:\n%s", firstText)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestDialectizeFilterPreservesIneligibleByteForByte checks the filter side:
// multi-leg, cost-carrying, empty-narration, and metadata transactions stay
// untouched, and only trivially-representable ones are rewritten.
func TestDialectizeFilterPreservesIneligibleByteForByte(t *testing.T) {
	original := writeFile(t, "v3.bean", mixedV3Ledger)
	rewritten := readFile(t, dialectizeFile(t, original))

	mustContain := []string{
		"{101.00 USD}",          // cost lot stays standard
		`* "买入" "带成本批次"`,        // cost txn header stays verbatim
		"2026-08-19 price AAPL", // price directive stays
	}
	for _, want := range mustContain {
		if !strings.Contains(rewritten, want) {
			t.Errorf("rewritten output lost %q:\n%s", want, rewritten)
		}
	}
	// The three-leg split is expressible as a block now: header plus one
	// leg per destination, so its amounts survive as leg text.
	for _, want := range []string{
		`* "三腿" "拆分记账"`,
		" 20 USD @Assets:Cash -> @Expenses:Food",
		" 10 USD @Assets:Cash -> @Expenses:Travel",
	} {
		if !strings.Contains(rewritten, want) {
			t.Errorf("three-leg split not converted to block (lost %q):\n%s", want, rewritten)
		}
	}
	if !strings.Contains(rewritten, "@Assets:WeChat -> @Expenses:Food") {
		t.Errorf("eligible transaction was not converted:\n%s", rewritten)
	}
	if !strings.Contains(rewritten, "! 15 USD @Assets:WeChat -> @Expenses:Food") {
		t.Errorf("flag ! not carried into dialect line:\n%s", rewritten)
	}
}

// TestExportReplacesDialectLines verifies the export path produces pure v3:
// every dialect line is gone and its compiled block is present.
func TestExportReplacesDialectLines(t *testing.T) {
	dialectSource := writeFile(t, "d.bean", `2000-01-01 open Assets:Cash USD
2000-01-01 open Expenses:Food USD
option "operating_currency" "USD"
2026-08-12 28 @Cash -> @Food "美团" : 工作午餐 #food
`)
	exported := readFile(t, exportFile(t, dialectSource))
	if strings.Contains(exported, "->") {
		t.Fatalf("dialect arrow survived export:\n%s", exported)
	}
	for _, want := range []string{
		`2026-08-12 * "美团" "工作午餐" #food`,
		"Assets:Cash -28 USD",
		"Expenses:Food 28 USD",
	} {
		if !strings.Contains(exported, want) {
			t.Errorf("export missing %q:\n%s", want, exported)
		}
	}
	// The export must itself be a valid standard ledger.
	balancesOf(t, exportFile(t, dialectSource))
}
