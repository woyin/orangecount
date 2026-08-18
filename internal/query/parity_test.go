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

package query

import (
	"strings"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

// balancedEvaluation parses and evaluates raw ledger text, failing the test
// on any error.
func balancedEvaluation(t *testing.T, content string) ledger.Evaluation {
	t.Helper()
	file, diagnostics := ledger.ParseText("parity.bean", []byte(content))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation invalid: %+v", evaluation.Diagnostics)
	}
	return *evaluation
}

// multiCurrencyLedger interleaves files so chronological order differs from
// source order (2026 rows precede 2025 rows): the balance column must still
// accumulate by date.
const multiCurrencyLedger = `
option "operating_currency" "CNY"
2025-01-01 open Assets:Bank CNY
2025-01-01 open Assets:Broker SH, CNY
2025-01-01 open Expenses:Food CNY
2025-01-01 open Expenses:Tech USD
2025-01-01 open Assets:Cash CNY, USD
2026-02-01 * "late buy" "file order first"
  Assets:Broker  2 SH {10 CNY}
  Assets:Cash   -20 CNY
2025-03-01 * "groceries"
  Expenses:Food  30 CNY
  Assets:Cash
2025-02-01 * "usd sub"
  Expenses:Tech  3 USD
  Assets:Cash   -3 USD
`

func TestUnknownColumnFailsLoudlyInsteadOfZero(t *testing.T) {
	evaluation := balancedEvaluation(t, multiCurrencyLedger)
	for _, q := range []string{
		"SELECT account, sum(amount) FROM postings GROUP BY account",
		"SELECT payee FROM postings WHERE amounts > 0",
		"SELECT payee FROM postings GROUP BY weekday",
	} {
		if _, err := Evaluate(q, evaluation); err == nil || !strings.Contains(err.Error(), "unknown column") {
			t.Errorf("%q should fail with unknown column, got %v", q, err)
		}
	}
	if _, err := Evaluate("SELECT account FROM nosuchtable", evaluation); err == nil {
		t.Error("unknown table should fail")
	}
}

func TestRegexOperatorFiltersAccounts(t *testing.T) {
	evaluation := balancedEvaluation(t, multiCurrencyLedger)
	result, err := Evaluate(`SELECT account FROM postings WHERE account ~ 'Expenses:' AND number > 0`, evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("regex should match both expense accounts, got %+v", result.Rows)
	}
	if _, err := Evaluate(`SELECT account FROM postings WHERE account ~ '['`, evaluation); err == nil || !strings.Contains(err.Error(), "invalid regular expression") {
		t.Errorf("invalid pattern should fail, got %v", err)
	}
	// A non-string left operand never matches instead of erroring (NULL-ish).
	if result, err := Evaluate(`SELECT account FROM postings WHERE number ~ 'Expenses:'`, evaluation); err != nil || len(result.Rows) != 0 {
		t.Errorf("numeric ~ pattern should match nothing, got %+v err=%v", result.Rows, err)
	}
}

func TestBareDateColumnsGroupByYearAndMonth(t *testing.T) {
	evaluation := balancedEvaluation(t, multiCurrencyLedger)
	result, err := Evaluate(`SELECT year, month, sum(number) AS total FROM postings WHERE account ~ 'Expenses' GROUP BY year, month`, evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected one row per (year, month), got %+v", result.Rows)
	}
	first := result.Rows[0]
	if first["year"] == nil || first["month"] == nil || first["total"] == nil {
		t.Fatalf("bare date columns must resolve, got %+v", first)
	}
}

func TestSelectAliasUsableInGroupBy(t *testing.T) {
	evaluation := balancedEvaluation(t, multiCurrencyLedger)
	result, err := Evaluate(`SELECT year(date) AS y, sum(number) AS total FROM postings WHERE account ~ 'Expenses' GROUP BY y`, evaluation)
	if err != nil {
		t.Fatal(err)
	}
	// Only 2025 carries Expenses postings (the 2026 buy hits Assets), and the
	// two expense currencies sum to 30 + 3.
	if len(result.Rows) != 1 || result.Rows[0]["y"] == nil || formatValue(result.Rows[0]["total"]) != "33" {
		t.Fatalf("alias group keys should partition by year, got %+v", result.Rows)
	}
}

func TestBalanceColumnAccumulatesChronologically(t *testing.T) {
	evaluation := balancedEvaluation(t, multiCurrencyLedger)
	result, err := Evaluate(`SELECT date, number, balance FROM postings WHERE account = 'Assets:Cash'`, evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 3 {
		t.Fatalf("expected 3 cash postings (usd sub, groceries, late buy), got %+v", result.Rows)
	}
	// Source order lists the 2026 buy first; the balance column must still
	// reflect date order: -3 USD (Feb), 30 CNY (Mar), then the 2026 buy.
	if got := result.Rows[0]["balance"]; got != "-3 USD" {
		t.Fatalf("first chronological balance=%v, want -3 USD", got)
	}
	last := result.Rows[len(result.Rows)-1]["balance"].(string)
	if !strings.Contains(last, "-50 CNY") || !strings.Contains(last, "-3 USD") {
		t.Fatalf("final balance=%q should hold -50 CNY and -3 USD", last)
	}
}

func TestWeightColumnValuesCostPriceAndPlain(t *testing.T) {
	evaluation := balancedEvaluation(t, multiCurrencyLedger)
	result, err := Evaluate(`SELECT account, weight FROM postings WHERE account ~ 'Broker|Cash' ORDER BY date`, evaluation)
	if err != nil {
		t.Fatal(err)
	}
	weights := map[string]string{}
	for _, row := range result.Rows {
		weights[row["account"].(string)+formatValue(row["weight"])] = formatValue(row["weight"])
	}
	// The costed SH buy weighs 20 CNY; the cash legs weigh their own units.
	if weights["Assets:Broker20"] == "" {
		t.Fatalf("cost weight missing, got %+v", result.Rows)
	}
}

func TestRootFunctionTruncatesAccountComponents(t *testing.T) {
	evaluation := balancedEvaluation(t, multiCurrencyLedger)
	result, err := Evaluate(`SELECT root(account, 1) AS top, sum(number) AS total FROM postings WHERE account ~ 'Expenses' GROUP BY top`, evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0]["top"] != "Expenses" {
		t.Fatalf("root(1) should collapse to Expenses, got %+v", result.Rows)
	}
	deep, err := Evaluate(`SELECT root('Assets:Bank:Checking', 2) FROM postings LIMIT 1`, evaluation)
	if err != nil || len(deep.Rows) == 0 || deep.Rows[0][deep.Columns[0]] != "Assets:Bank" {
		t.Fatalf("root(2)=%+v err=%v", deep, err)
	}
}

func TestAggregatesRejectNonNumericColumns(t *testing.T) {
	evaluation := balancedEvaluation(t, multiCurrencyLedger)
	if _, err := Evaluate(`SELECT sum(balance) FROM postings`, evaluation); err == nil || !strings.Contains(err.Error(), "cannot aggregate") {
		t.Errorf("sum(balance) should fail honestly, got %v", err)
	}
	if _, err := Evaluate(`SELECT sum(narration) FROM postings`, evaluation); err == nil {
		t.Error("sum(narration) should fail")
	}
}
