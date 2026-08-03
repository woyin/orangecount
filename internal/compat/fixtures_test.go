// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package compat contains sanitized compatibility and regression fixtures.
// It intentionally never references the owner-provided private ledger.
package compat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

func fixturePath(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	base := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "fixtures")
	return filepath.Join(append([]string{base}, parts...)...)
}

func TestSanitizedCoreFixtureParsesAndEvaluates(t *testing.T) {
	entry := fixturePath("include", "main.bean")
	result := snapshot.Build(entry)
	if result.Snapshot == nil || result.Err != nil {
		t.Fatalf("fixture snapshot unavailable: err=%v diagnostics=%+v", result.Err, result.Diagnostics)
	}
	evaluation := result.Snapshot.Evaluation()
	if !evaluation.Valid || len(evaluation.Accounts) != 2 || len(evaluation.Entries) < 3 {
		t.Fatalf("unexpected fixture evaluation: valid=%v accounts=%d entries=%d diagnostics=%+v", evaluation.Valid, len(evaluation.Accounts), len(evaluation.Entries), evaluation.Diagnostics)
	}
	first, err := query.Evaluate("SELECT account, sum(balance) AS balance FROM accounts GROUP BY account ORDER BY account", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	second, err := query.Evaluate("SELECT account, sum(balance) AS balance FROM accounts GROUP BY account ORDER BY account", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := formatRows(first), formatRows(second); got != want {
		t.Fatalf("query is not deterministic: %q != %q", got, want)
	}
	if len(report.Journal(evaluation).Rows) != 2 {
		t.Fatalf("journal rows=%d", len(report.Journal(evaluation).Rows))
	}

	core := snapshot.Build(fixturePath("core.bean"))
	if core.Snapshot == nil {
		t.Fatalf("core fixture snapshot unavailable: err=%v diagnostics=%+v", core.Err, core.Diagnostics)
	}
	coreEvaluation := core.Snapshot.Evaluation()
	wantBalances := map[string]string{
		"Assets:Cash|USD":    "100",
		"Assets:Shares|":     "0",
		"Equity:Opening|USD": "-100",
	}
	assertReportBalances(t, "accounts", report.Accounts(coreEvaluation), wantBalances)
	assertReportBalances(t, "trial balance", report.TrialBalance(coreEvaluation), wantBalances)
	balanceSheet := report.BalanceSheet(coreEvaluation)
	assertReportBalances(t, "balance sheet", balanceSheet, wantBalances)
	assertAggregateBalance(t, "balance sheet", balanceSheet, "Assets", "USD", "100")
	assertAggregateBalance(t, "balance sheet", balanceSheet, "Equity", "USD", "-100")
}

func assertReportBalances(t *testing.T, name string, result query.Result, want map[string]string) {
	t.Helper()
	leafRows := make([]query.Row, 0, len(result.Rows))
	for _, row := range result.Rows {
		// Tree reports include synthetic ancestor rows for the UI. Their
		// totals are checked separately; this assertion remains focused on
		// the exact account/currency balances used by the compatibility
		// fixture.
		if role, ok := row["_tree_role"].(string); ok && role == "aggregate" {
			continue
		}
		leafRows = append(leafRows, row)
	}
	if len(leafRows) != len(want) {
		t.Fatalf("%s rows=%d want=%d: %+v", name, len(leafRows), len(want), result.Rows)
	}
	seen := make(map[string]bool, len(leafRows))
	for _, row := range leafRows {
		account, _ := row["account"].(string)
		currency, _ := row["currency"].(string)
		key := account + "|" + currency
		balance, ok := row["balance"].(ledger.Decimal)
		if !ok {
			t.Fatalf("%s row %q has balance type %T", name, key, row["balance"])
		}
		if got, expected := balance.String(), want[key]; got != expected {
			t.Fatalf("%s %q balance=%s want=%s", name, key, got, expected)
		}
		seen[key] = true
	}
	for key := range want {
		if !seen[key] {
			t.Fatalf("%s missing row %q: %+v", name, key, result.Rows)
		}
	}
}

func assertAggregateBalance(t *testing.T, name string, result query.Result, account, currency, want string) {
	t.Helper()
	for _, row := range result.Rows {
		if row["account"] != account || row["currency"] != currency || row["_tree_role"] != "aggregate" {
			continue
		}
		balance, ok := row["balance"].(ledger.Decimal)
		if !ok {
			t.Fatalf("%s aggregate %q has balance type %T", name, account, row["balance"])
		}
		if got := balance.String(); got != want {
			t.Fatalf("%s aggregate %q balance=%s want=%s", name, account, got, want)
		}
		return
	}
	t.Fatalf("%s missing aggregate %q/%s: %+v", name, account, currency, result.Rows)
}

func TestSanitizedSyntaxAndUnicodeFixtures(t *testing.T) {
	for _, name := range []string{"core.bean", "unicode.bean"} {
		data, err := os.ReadFile(fixturePath(name))
		if err != nil {
			t.Fatal(err)
		}
		_, diagnostics := ledger.ParseText(name, data)
		if diagnostics.HasErrors() {
			t.Fatalf("%s diagnostics=%+v", name, diagnostics.All())
		}
	}
	data, err := os.ReadFile(fixturePath("invalid.bean"))
	if err != nil {
		t.Fatal(err)
	}
	_, diagnostics := ledger.ParseText("invalid.bean", data)
	if !diagnostics.HasErrors() || diagnostics.Len() < 2 {
		t.Fatalf("invalid fixture did not accumulate diagnostics: %+v", diagnostics.All())
	}
}

func TestSanitizedIncludeFailuresAreStable(t *testing.T) {
	for _, name := range []string{"missing.bean", "cycle-a.bean"} {
		graph, err := source.LoadGraph(fixturePath("include", name))
		if err != nil {
			t.Fatal(err)
		}
		if len(graph.Diagnostics) == 0 {
			t.Fatalf("%s has no graph diagnostics", name)
		}
		first := graph.Diagnostics[0].Code
		graphAgain, err := source.LoadGraph(fixturePath("include", name))
		if err != nil || len(graphAgain.Diagnostics) == 0 || graphAgain.Diagnostics[0].Code != first {
			t.Fatalf("%s diagnostics are unstable: first=%+v second=%+v err=%v", name, graph.Diagnostics, graphAgain.Diagnostics, err)
		}
	}
}

func TestSanitizedFixturesContainNoPrivateLedgerMarker(t *testing.T) {
	root := fixturePath()
	var paths []string
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(string(data)), "financebook") {
			t.Fatalf("private-ledger marker found in fixture %s", path)
		}
	}
}

func TestCompatibilityLedgerIsMachineReadable(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "docs", "compatibility-ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ledger struct {
		Contract            string           `json:"contract"`
		ApprovedDifferences []map[string]any `json:"approved_differences"`
		Fixtures            []string         `json:"sanitized_fixtures"`
	}
	if err := json.Unmarshal(data, &ledger); err != nil {
		t.Fatal(err)
	}
	if ledger.Contract != "beancount-v3-core" || len(ledger.ApprovedDifferences) == 0 || len(ledger.Fixtures) == 0 {
		t.Fatalf("incomplete compatibility ledger: %+v", ledger)
	}
}

func formatRows(result query.Result) string {
	var builder strings.Builder
	for _, row := range result.Rows {
		for _, column := range result.Columns {
			builder.WriteString(column)
			builder.WriteByte('=')
			builder.WriteString(queryValue(row[column]))
			builder.WriteByte(';')
		}
	}
	return builder.String()
}

func queryValue(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(toString(value), "\n", " "), "\r", " "))
}

func toString(value any) string {
	if decimal, ok := value.(ledger.Decimal); ok {
		return decimal.String()
	}
	return fmt.Sprint(value)
}
