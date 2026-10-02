// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package query_test

import (
	"errors"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
	"os"
	"path/filepath"
	"testing"
)

func TestStandardScalarTablesAndSourceBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tables.bean")
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-01 commodity USD
2000-01-02 * "shop" "lunch"
  Assets:Cash 2 USD
  Equity:Opening -2 USD
2000-01-03 balance Assets:Cash 2 USD
2000-01-03 note Assets:Cash "hello"
2000-01-03 event "location" "home"
2000-01-03 document Assets:Cash "receipt.pdf"
2000-01-03 price USD 7 CNY
`
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	build := snapshot.BuildWithOptions(path, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if build.Snapshot == nil {
		t.Fatalf("%+v", build.Diagnostics)
	}
	canonicalDir, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		text        string
		rows        int
		kind, value string
	}{
		{"SELECT payee, narration FROM transactions", 1, "string", "shop"},
		{"SELECT type FROM entries WHERE type = 'note'", 1, "string", "note"},
		{"SELECT account FROM accounts", 2, "string", "Assets:Cash"},
		{"SELECT name FROM commodities", 1, "string", "USD"},
		{"SELECT account FROM #balances", 1, "string", "Assets:Cash"},
		{"SELECT comment FROM notes", 1, "string", "hello"},
		{"SELECT type, description FROM events", 1, "string", "location"},
		{"SELECT filename FROM documents", 1, "string", filepath.Join(canonicalDir, "receipt.pdf")},
		{"SELECT currency FROM prices", 1, "string", "USD"},
	} {
		t.Run(test.text, func(t *testing.T) {
			r, err := query.EvaluateTyped(test.text, build.Snapshot.Evaluation())
			if err != nil {
				t.Fatal(err)
			}
			if len(r.Rows) != test.rows || r.Columns[0].Type != test.kind || r.Rows[0][0].Value != test.value {
				t.Fatalf("%+v", r)
			}
		})
	}
	for _, text := range []string{"SELECT account FROM events LIMIT 0", "SELECT abs(account) FROM events", "SELECT count(narration) FROM prices"} {
		_, err := query.EvaluateTyped(text, ledger.Evaluation{})
		var detail query.TypedError
		if !errors.As(err, &detail) || detail.Code != "unknown-column" {
			t.Errorf("%s: %v", text, err)
		}
	}
	for _, table := range []string{"entries", "accounts", "commodities", "balances", "notes", "events", "documents", "prices", "transactions"} {
		r, err := query.EvaluateTyped("SELECT count(*) FROM "+table, ledger.Evaluation{})
		if err != nil || len(r.Columns) != 1 || r.Columns[0].Type != "integer" || len(r.Rows) != 0 {
			t.Errorf("%s %+v %v", table, r, err)
		}
	}
}
