// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package compat

import (
	"os"
	"path/filepath"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

func TestFavaVisualFixtureCoversReadOnlySurface(t *testing.T) {
	root := fixturePath("fava-visual")
	built := snapshot.Build(filepath.Join(root, "main.bean"))
	if built.Snapshot == nil || built.Err != nil {
		t.Fatalf("visual fixture snapshot unavailable: err=%v diagnostics=%+v", built.Err, built.Diagnostics)
	}
	evaluation := built.Snapshot.Evaluation()
	if !evaluation.Valid || len(evaluation.Accounts) < 7 || len(evaluation.Entries) < 20 {
		t.Fatalf("visual fixture is too small: valid=%v accounts=%d entries=%d diagnostics=%+v", evaluation.Valid, len(evaluation.Accounts), len(evaluation.Entries), evaluation.Diagnostics)
	}
	for _, currency := range []string{"USD", "EUR", "JPY", "CAD", "SH"} {
		if _, ok := evaluation.Prices[currency]; currency == "SH" && !ok {
			t.Fatalf("missing synthetic price series for %s", currency)
		}
	}
	for name, result := range map[string]struct {
		columns int
		rows    int
	}{
		"accounts":         {len(report.Accounts(evaluation).Columns), len(report.Accounts(evaluation).Rows)},
		"journal":          {len(report.Journal(evaluation).Columns), len(report.Journal(evaluation).Rows)},
		"statistics":       {len(report.Statistics(evaluation).Columns), len(report.Statistics(evaluation).Rows)},
		"trial-balance":    {len(report.TrialBalanceTree(evaluation).Columns), len(report.TrialBalanceTree(evaluation).Rows)},
		"balance-sheet":    {len(report.BalanceSheet(evaluation).Columns), len(report.BalanceSheet(evaluation).Rows)},
		"income-statement": {len(report.IncomeStatement(evaluation).Columns), len(report.IncomeStatement(evaluation).Rows)},
		"holdings":         {len(report.Holdings(evaluation).Columns), len(report.Holdings(evaluation).Rows)},
		"events":           {len(report.Events(evaluation).Columns), len(report.Events(evaluation).Rows)},
		"documents":        {len(report.Documents(evaluation).Columns), len(report.Documents(evaluation).Rows)},
	} {
		if result.columns == 0 || result.rows == 0 {
			t.Errorf("%s report lacks fixture coverage: columns=%d rows=%d", name, result.columns, result.rows)
		}
	}
	if _, err := query.Evaluate("SELECT account, balance FROM accounts ORDER BY account", evaluation); err != nil {
		t.Fatalf("saved-query shape failed: %v", err)
	}
	if _, err := query.Evaluate("SELECT FROM", evaluation); err == nil {
		t.Fatal("invalid query unexpectedly succeeded")
	}
}

func TestFavaReferenceFixtureIsDenseAndDeterministic(t *testing.T) {
	root := fixturePath("fava-reference")
	built := snapshot.Build(filepath.Join(root, "main.bean"))
	if built.Snapshot == nil || built.Err != nil {
		t.Fatalf("dense reference snapshot unavailable: err=%v diagnostics=%+v", built.Err, built.Diagnostics)
	}
	evaluation := built.Snapshot.Evaluation()
	if !evaluation.Valid || len(evaluation.Accounts) < 80 || len(evaluation.Entries) < 280 {
		t.Fatalf("dense reference coverage too small: valid=%v accounts=%d entries=%d diagnostics=%+v", evaluation.Valid, len(evaluation.Accounts), len(evaluation.Entries), evaluation.Diagnostics)
	}
	if got := len(collectFixtureCurrencies(evaluation)); got < 10 {
		t.Fatalf("dense reference currencies=%d want at least 10", got)
	}
	for name, result := range map[string]query.Result{
		"journal":          report.Journal(evaluation),
		"trial-balance":    report.TrialBalanceTree(evaluation),
		"balance-sheet":    report.BalanceSheet(evaluation),
		"income-statement": report.IncomeStatement(evaluation),
		"holdings":         report.Holdings(evaluation),
		"documents":        report.Documents(evaluation),
		"events":           report.Events(evaluation),
	} {
		if len(result.Rows) == 0 || len(result.Columns) == 0 {
			t.Errorf("dense reference %s report is empty: columns=%d rows=%d", name, len(result.Columns), len(result.Rows))
		}
	}
	if _, err := query.Evaluate("SELECT account, balance FROM accounts ORDER BY account", evaluation); err != nil {
		t.Fatalf("dense reference saved query failed: %v", err)
	}
}

func collectFixtureCurrencies(evaluation ledger.Evaluation) map[string]bool {
	currencies := make(map[string]bool)
	for _, account := range evaluation.Accounts {
		for currency := range account.Balances {
			currencies[currency] = true
		}
		for _, position := range account.Positions {
			currencies[position.Currency] = true
		}
	}
	for base, quotes := range evaluation.Prices {
		currencies[base] = true
		for _, quote := range quotes {
			currencies[quote.Currency] = true
		}
	}
	return currencies
}

func TestFavaVisualFixtureHasSafeDocumentsImportAndEditorStates(t *testing.T) {
	root := fixturePath("fava-visual")
	roots, err := source.NewDocumentRoots([]string{filepath.Join(root, "documents")})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"receipt-a.txt", "nested/receipt-b.txt"} {
		if _, err := roots.Resolve(name); err != nil {
			t.Fatalf("safe document %q did not resolve: %v", name, err)
		}
	}
	if _, err := roots.Resolve("../import/import-candidate.csv"); err == nil {
		t.Fatal("document root accepted traversal")
	}

	importData, err := os.ReadFile(filepath.Join(root, "import", "import-candidate.csv"))
	if err != nil || len(importData) == 0 {
		t.Fatalf("import candidate unavailable: err=%v", err)
	}
	editorData, err := os.ReadFile(filepath.Join(root, "editor", "invalid.bean"))
	if err != nil {
		t.Fatal(err)
	}
	_, diagnostics := ledger.ParseText("fava-visual/editor/invalid.bean", editorData)
	if !diagnostics.HasErrors() {
		t.Fatal("editor diagnostic fixture has no parse errors")
	}
}
