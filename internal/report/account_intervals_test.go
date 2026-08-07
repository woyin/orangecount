// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package report

import (
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/source"
)

func evaluateIntervalLedger(t *testing.T) *ledger.Evaluation {
	t.Helper()
	text := `2000-01-01 open Assets:Bank USD
2000-01-01 open Assets:Bank:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-05 * "seed"
  Assets:Bank 10 USD
  Equity:Opening -10 USD
2000-02-07 * "child deposit"
  Assets:Bank:Cash 2 USD
  Equity:Opening -2 USD
2000-03-09 * "top up"
  Assets:Bank 5 USD
  Equity:Opening -5 USD
`
	file, diagnostics := ledger.ParseText("intervals.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}
	return evaluation
}

func intervalRows(t *testing.T, result query.Result) []map[string]string {
	t.Helper()
	rows := make([]map[string]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		rendered := map[string]string{}
		for column, value := range row {
			if decimal, ok := value.(ledger.Decimal); ok {
				rendered[column] = decimal.String()
				continue
			}
			rendered[column] = value.(string)
		}
		rows = append(rows, rendered)
	}
	return rows
}

func TestAccountIntervalsChangesAndBalancesCarryQuietMonths(t *testing.T) {
	evaluation := evaluateIntervalLedger(t)

	changes := AccountIntervals(*evaluation, "Assets:Bank", "changes", "month", Filters{})
	if want := []string{"interval", "USD"}; len(changes.Columns) != 2 || changes.Columns[0] != want[0] || changes.Columns[1] != want[1] {
		t.Fatalf("columns=%v", changes.Columns)
	}
	changeRows := intervalRows(t, changes)
	if len(changeRows) != 3 {
		t.Fatalf("changes rows=%+v", changeRows)
	}
	for index, want := range []string{"10", "2", "5"} {
		if changeRows[index]["USD"] != want {
			t.Fatalf("changes[%d]=%s want=%s rows=%+v", index, changeRows[index]["USD"], want, changeRows)
		}
	}
	if changeRows[1]["interval"] != "2000-02" {
		t.Fatalf("quiet month missing: %+v", changeRows)
	}

	balances := AccountIntervals(*evaluation, "Assets:Bank", "balances", "month", Filters{})
	balanceRows := intervalRows(t, balances)
	for index, want := range []string{"10", "12", "17"} {
		if balanceRows[index]["USD"] != want {
			t.Fatalf("balances[%d]=%s want=%s rows=%+v", index, balanceRows[index]["USD"], want, balanceRows)
		}
	}
}

func TestAccountIntervalsHonorTimeFilterAndRejectBadMode(t *testing.T) {
	evaluation := evaluateIntervalLedger(t)

	filtered := AccountIntervals(*evaluation, "Assets:Bank", "balances", "month", Filters{TimeBegin: "2000-02-01"})
	rows := intervalRows(t, filtered)
	if len(rows) != 2 || rows[0]["interval"] != "2000-02" || rows[0]["USD"] != "2" || rows[1]["USD"] != "7" {
		t.Fatalf("filtered rows=%+v", rows)
	}

	if got := AccountIntervals(*evaluation, "Assets:Bank", "bogus", "month", Filters{}); len(got.Rows) != 0 || len(got.Columns) != 1 {
		t.Fatalf("bad mode should stay empty: %+v", got)
	}
}
