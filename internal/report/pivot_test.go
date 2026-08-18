// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package report

import (
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/source"
)

// pivotEvaluation builds the multi-currency fixture the pivot tests use:
// expenses in two currencies and two months, a quiet month between them, and
// a cash leg the account filter must exclude.
func pivotEvaluation(t *testing.T) ledger.Evaluation {
	t.Helper()
	text := `option "operating_currency" "CNY"
2025-01-01 open Assets:Cash CNY, USD
2025-01-01 open Expenses:Food CNY
2025-01-01 open Expenses:Food:USD USD
2025-01-10 * "买菜"
  Expenses:Food      30 CNY
  Assets:Cash
2025-01-20 * "usd snack"
  Expenses:Food:USD  3 USD
  Assets:Cash
2025-03-05 * "三月的饭"
  Expenses:Food      45 CNY
  Assets:Cash
`
	file, diagnostics := ledger.ParseText("pivot.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}
	return *evaluation
}

func pivotDecimal(t *testing.T, raw string) ledger.Decimal {
	t.Helper()
	value, err := ledger.ParseDecimal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

// pivotAmount reads one cell and renders it as text so comparisons stay
// exact-deactimal instead of struct identity (computed Decimals carry no Raw).
func pivotAmount(row query.Row, column string) string {
	value, _ := row[column].(ledger.Decimal)
	return value.String()
}

func TestPivotTableSumsByMonthPerCurrency(t *testing.T) {
	result := PivotTable(pivotEvaluation(t), PivotSpec{Rows: "month", Values: "sum", Account: "Expenses"})
	if len(result.Columns) != 3 || result.Columns[0] != "interval" || result.Columns[1] != "CNY" || result.Columns[2] != "USD" {
		t.Fatalf("columns=%v want interval + CNY + USD", result.Columns)
	}
	// Quiet February still renders as a dense zero row, matching the
	// interval tables' full-grid presentation.
	if len(result.Rows) != 3 || result.Rows[1]["interval"] != "2025-02" {
		t.Fatalf("jan feb mar dense rows, got %+v", result.Rows)
	}
	if pivotAmount(result.Rows[0], "CNY") != "30" || pivotAmount(result.Rows[0], "USD") != "3" {
		t.Fatalf("january row=%+v", result.Rows[0])
	}
	if pivotAmount(result.Rows[2], "CNY") != "45" {
		t.Fatalf("march row=%+v", result.Rows[2])
	}
}

func TestPivotTableBalanceCarriesQuietMonths(t *testing.T) {
	result := PivotTable(pivotEvaluation(t), PivotSpec{Rows: "month", Values: "balance", Account: "Expenses"})
	if len(result.Rows) != 3 {
		t.Fatalf("february must appear carrying january's balance, got %+v", result.Rows)
	}
	if pivotAmount(result.Rows[1], "CNY") != "30" {
		t.Fatalf("february carried CNY=%v want 30", result.Rows[1]["CNY"])
	}
	if pivotAmount(result.Rows[2], "CNY") != "75" {
		t.Fatalf("march ending CNY=%v want 75", result.Rows[2]["CNY"])
	}
}

func TestPivotTableGroupsColumnsByAccountRoot(t *testing.T) {
	result := PivotTable(pivotEvaluation(t), PivotSpec{Rows: "month", Columns: "root1", Values: "sum", Account: "Expenses"})
	want := []string{"interval", "Expenses CNY", "Expenses USD"}
	if len(result.Columns) != len(want) || result.Columns[0] != want[0] || result.Columns[1] != want[1] || result.Columns[2] != want[2] {
		t.Fatalf("columns=%v want %v", result.Columns, want)
	}
	if pivotAmount(result.Rows[0], "Expenses CNY") != "30" {
		t.Fatalf("root1 grouped row=%+v", result.Rows[0])
	}
}

func TestPivotTableDeeperRootSplitsSubaccounts(t *testing.T) {
	result := PivotTable(pivotEvaluation(t), PivotSpec{Rows: "year", Columns: "root2", Values: "sum", Account: "Expenses"})
	columns := map[string]bool{}
	for _, name := range result.Columns {
		columns[name] = true
	}
	if !columns["Expenses:Food CNY"] || !columns["Expenses:Food USD"] {
		t.Fatalf("root2 groups by the first two components, got %v", result.Columns)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("one year row, got %+v", result.Rows)
	}
}

func TestPivotTableEmptyAccountReturnsHeaderOnly(t *testing.T) {
	result := PivotTable(pivotEvaluation(t), PivotSpec{Rows: "month", Values: "sum", Account: "NoSuch"})
	if len(result.Columns) != 1 || result.Columns[0] != "interval" || len(result.Rows) != 0 {
		t.Fatalf("empty pivot=%+v", result)
	}
}
