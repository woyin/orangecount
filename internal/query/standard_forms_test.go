// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package query_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

// standardFormsEvaluation builds the fixed synthetic ledger whose expected
// values were captured from BeanQuery 0.2.0 ("the oracle"). Every expectation
// in this file is an oracle observation mapped into the typed contract, except
// where a comment names a deliberate OrangeCount boundary: the oracle reports
// JSON rows with a dtype per column, which becomes the current cell type and
// value.
func standardFormsEvaluation(t *testing.T) ledger.Evaluation {
	t.Helper()
	build := snapshot.BuildWithOptions("../../testdata/fixtures/query/standard-forms.bean", snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if build.Err != nil || build.Snapshot == nil {
		t.Fatalf("fixture build failed: %v, %v", build.Err, build.Diagnostics)
	}
	return build.Snapshot.Evaluation()
}

// standardFormsMultiCurrencyEvaluation builds the two-currency fixture whose
// pivot has a missing (row key, pivot value) combination. The expectations are
// again captured BeanQuery 0.2.0 observations.
func standardFormsMultiCurrencyEvaluation(t *testing.T) ledger.Evaluation {
	t.Helper()
	build := snapshot.BuildWithOptions("../../testdata/fixtures/query/standard-forms-multicurrency.bean", snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if build.Err != nil || build.Snapshot == nil {
		t.Fatalf("fixture build failed: %v, %v", build.Err, build.Diagnostics)
	}
	return build.Snapshot.Evaluation()
}

// typedRow renders one result row as comparable text so the assertions read
// like the oracle output (NULL as "<null>", exact decimals without display
// rounding, and the cell type prefixed so NULL, empty string and zero stay
// distinguishable).
func typedRow(t *testing.T, result query.TypedResult, row int) string {
	t.Helper()
	if row >= len(result.Rows) {
		t.Fatalf("row %d missing from %+v", row, result.Rows)
	}
	parts := make([]string, len(result.Rows[row]))
	for i, cell := range result.Rows[row] {
		parts[i] = typedCellText(cell)
	}
	return strings.Join(parts, " | ")
}

func typedCellText(cell query.Cell) string {
	switch cell.Type {
	case "null":
		return "<null>"
	case "decimal":
		return "decimal(" + cell.Value.(query.PresentedDecimal).Exact + ")"
	case "integer":
		return "integer(" + cell.Value.(string) + ")"
	case "string":
		return "string(" + cell.Value.(string) + ")"
	case "bool":
		if cell.Value.(bool) {
			return "bool(true)"
		}
		return "bool(false)"
	default:
		return cell.Type + "(?)"
	}
}

func requireTypedCells(t *testing.T, result query.TypedResult, want []string) {
	t.Helper()
	if len(result.Rows) != len(want) {
		t.Fatalf("rows=%d want=%d\n got=%v", len(result.Rows), len(want), result.Rows)
	}
	for i := range want {
		if got := typedRow(t, result, i); got != want[i] {
			t.Errorf("row %d = %q want %q", i, got, want[i])
		}
	}
}

func requireTypedColumns(t *testing.T, result query.TypedResult, names, types []string) {
	t.Helper()
	if len(result.Columns) != len(names) {
		t.Fatalf("columns=%+v want names %v", result.Columns, names)
	}
	for i, column := range result.Columns {
		if column.Name != names[i] || column.Type != types[i] {
			t.Errorf("column %d = %s/%s want %s/%s", i, column.Name, column.Type, names[i], types[i])
		}
		if column.ID != "c"+itoa(i) {
			t.Errorf("column %d id=%q", i, column.ID)
		}
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}

// TestStandardFormsMatchBeanQueryOracle pins the four external oracle cases
// recorded in docs/evidence/query-oracle-2026-10-12.json (the review-time
// capture of the same run lives under fix/review-evidence-2026-10-12/ and is
// not part of a normal checkout). Row values, column names and column types
// are the oracle's, mapped to the typed contract; the JSON shape differs, so
// the expectations are compared as typed cells rather than as raw JSON.
func TestStandardFormsMatchBeanQueryOracle(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	for _, test := range []struct {
		name  string
		text  string
		names []string
		types []string
		want  []string
	}{
		{
			name:  "distinct",
			text:  "SELECT DISTINCT account ORDER BY account",
			names: []string{"account"},
			types: []string{"string"},
			want: []string{
				"string(Assets:Cash)",
				"string(Equity:Opening)",
				"string(Expenses:Food)",
			},
		},
		{
			name:  "having",
			text:  "SELECT account, sum(number) AS total GROUP BY account HAVING sum(number) > 0 ORDER BY account",
			names: []string{"account", "total"},
			types: []string{"string", "decimal"},
			want: []string{
				"string(Assets:Cash) | decimal(90)",
				"string(Expenses:Food) | decimal(10)",
			},
		},
		{
			name:  "subquery",
			text:  "SELECT account FROM (SELECT account WHERE number > 0) ORDER BY account",
			names: []string{"account"},
			types: []string{"string"},
			want: []string{
				"string(Assets:Cash)",
				"string(Expenses:Food)",
			},
		},
		{
			name:  "pivot",
			text:  "SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY 1, 2",
			names: []string{"account/currency", "USD"},
			types: []string{"string", "decimal"},
			want: []string{
				"string(Assets:Cash) | decimal(90)",
				"string(Equity:Opening) | decimal(-100)",
				"string(Expenses:Food) | decimal(10)",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := query.EvaluateTyped(test.text, evaluation)
			if err != nil {
				t.Fatal(err)
			}
			if result.Contract != query.CurrentContract {
				t.Fatalf("contract=%s", result.Contract)
			}
			requireTypedColumns(t, result, test.names, test.types)
			requireTypedCells(t, result, test.want)
		})
	}
}

// TestDistinctCollapsesCanonicalValuesButKeepsTypesAndNulls verifies the
// documented DISTINCT semantics: equal typed values collapse, NULLs collapse
// among themselves, and an empty string never equals NULL.
func TestDistinctCollapsesCanonicalValuesButKeepsTypesAndNulls(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	result, err := query.EvaluateTyped("SELECT DISTINCT account, number ORDER BY account, number", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedCells(t, result, []string{
		"string(Assets:Cash) | decimal(-10)",
		"string(Assets:Cash) | decimal(100)",
		"string(Equity:Opening) | decimal(-100)",
		"string(Expenses:Food) | decimal(10)",
	})
	// The unpriced cost column is entirely NULL; DISTINCT keeps exactly one row.
	blanks, err := query.EvaluateTyped("SELECT DISTINCT payee FROM postings", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedColumns(t, blanks, []string{"payee"}, []string{"string"})
	requireTypedCells(t, blanks, []string{"<null>"})
	// An empty string is a value, not NULL, so it survives as its own row.
	empty, err := query.EvaluateTyped("SELECT DISTINCT '' AS blank FROM postings", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedCells(t, empty, []string{"string()"})
}

// TestDistinctPreservesOrderLimitAndBigIntegers checks that DISTINCT runs
// before ORDER BY and LIMIT and does not disturb the existing lossless integer
// contract (the oracle's bigint control case).
func TestDistinctPreservesOrderLimitAndBigIntegers(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	result, err := query.EvaluateTyped("SELECT DISTINCT account ORDER BY account DESC LIMIT 2", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedCells(t, result, []string{"string(Expenses:Food)", "string(Equity:Opening)"})
	big, err := query.EvaluateTyped("SELECT 9223372036854775807 + 1 LIMIT 1", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedColumns(t, big, []string{"9223372036854775807 + 1"}, []string{"integer"})
	requireTypedCells(t, big, []string{"integer(9223372036854775808)"})
}

// TestHavingFiltersGroupsAfterAggregation covers the aggregate predicate, a
// source-column predicate combined with an aggregate, NOT, and the alias rule:
// HAVING may not reference a projected alias. A bare source column in HAVING
// resolves against the reference's leftover scan context (the last row of the
// unfiltered source), so it is an aggregate-level test that keeps or drops
// every group at once. The observations below reproduce the pinned oracle
// cell-for-cell instead of reinterpreting the predicate per group.
func TestHavingFiltersGroupsAfterAggregation(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	for _, test := range []struct {
		name string
		text string
		want []string
	}{
		{
			name: "source-predicate-matching-context-row",
			// The last source row is Assets:Cash, so this predicate holds for
			// every group and only the aggregate filters.
			text: "SELECT account, count(*) AS n GROUP BY account HAVING count(*) > 0 AND account = 'Assets:Cash' ORDER BY account",
			want: []string{
				"string(Assets:Cash) | integer(2)",
				"string(Equity:Opening) | integer(1)",
				"string(Expenses:Food) | integer(1)",
			},
		},
		{
			name: "source-predicate-rejecting-context-row",
			text: "SELECT account, count(*) AS n GROUP BY account HAVING count(*) > 0 AND account ~ 'Exp'",
			want: nil,
		},
		{
			name: "source-predicate-still-combines-with-aggregate",
			text: "SELECT account, count(*) AS n GROUP BY account HAVING count(*) = 1 AND account = 'Assets:Cash' ORDER BY account",
			want: []string{
				"string(Equity:Opening) | integer(1)",
				"string(Expenses:Food) | integer(1)",
			},
		},
		{
			// WHERE does not change the context row: the last source row is
			// still Assets:Cash -10 USD, so this predicate holds for every
			// surviving group even though WHERE removed that account.
			name: "source-predicate-with-where",
			text: "SELECT account, count(*) AS n WHERE account != 'Assets:Cash' GROUP BY account HAVING count(*) > 0 AND account = 'Assets:Cash' ORDER BY account",
			want: []string{
				"string(Equity:Opening) | integer(1)",
				"string(Expenses:Food) | integer(1)",
			},
		},
		{
			name: "source-predicate-with-where-rejecting",
			text: "SELECT account, count(*) AS n WHERE account != 'Assets:Cash' GROUP BY account HAVING count(*) > 0 AND account = 'Expenses:Food'",
			want: nil,
		},
		{
			name: "negated-aggregate",
			text: "SELECT account, sum(number) AS t GROUP BY account HAVING NOT sum(number) > 0 ORDER BY account",
			want: []string{"string(Equity:Opening) | decimal(-100)"},
		},
		{
			name: "aggregate-only-projection",
			text: "SELECT account FROM postings GROUP BY account HAVING count(*) > 1",
			want: []string{"string(Assets:Cash)"},
		},
		{
			name: "limit-after-having",
			text: "SELECT account, sum(number) AS t GROUP BY account HAVING sum(number) > 0 ORDER BY t DESC LIMIT 1",
			want: []string{"string(Assets:Cash) | decimal(90)"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := query.EvaluateTyped(test.text, evaluation)
			if err != nil {
				t.Fatal(err)
			}
			requireTypedCells(t, result, test.want)
		})
	}
	// The reference resolves HAVING names against source columns only, so a
	// projected alias stays a compile error even when an aggregate appears.
	for _, text := range []string{
		"SELECT account, sum(number) AS total GROUP BY account HAVING total > 0",
	} {
		_, err := query.EvaluateTyped(text, evaluation)
		var detail query.TypedError
		if !errors.As(err, &detail) || detail.Code != "unknown-column" && detail.Code != "invalid-group" {
			t.Errorf("query %q should be rejected, got %v", text, err)
		}
	}
}

// TestDerivedSubqueryExposesInnerColumnsAndTypes checks that the outer query
// reads the inner projection: names, declared types, WHERE over inner columns,
// aliases, nesting, aggregates over a derived table and a star select.
func TestDerivedSubqueryExposesInnerColumnsAndTypes(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	for _, test := range []struct {
		name  string
		text  string
		names []string
		types []string
		want  []string
	}{
		{
			name:  "outer-filter-on-inner-column",
			text:  "SELECT account, total FROM (SELECT account, sum(number) AS total GROUP BY account) WHERE total > 0 ORDER BY account",
			names: []string{"account", "total"},
			types: []string{"string", "decimal"},
			want:  []string{"string(Assets:Cash) | decimal(90)", "string(Expenses:Food) | decimal(10)"},
		},
		{
			name:  "inner-order-and-limit",
			text:  "SELECT account FROM (SELECT account FROM postings ORDER BY account DESC LIMIT 1)",
			names: []string{"account"},
			types: []string{"string"},
			want:  []string{"string(Expenses:Food)"},
		},
		{
			name:  "inner-distinct",
			text:  "SELECT account FROM (SELECT DISTINCT account FROM postings) ORDER BY account",
			names: []string{"account"},
			types: []string{"string"},
			want:  []string{"string(Assets:Cash)", "string(Equity:Opening)", "string(Expenses:Food)"},
		},
		{
			name:  "inner-having",
			text:  "SELECT account, n FROM (SELECT account, sum(number) AS n GROUP BY account HAVING sum(number) > 0) ORDER BY account",
			names: []string{"account", "n"},
			types: []string{"string", "decimal"},
			want:  []string{"string(Assets:Cash) | decimal(90)", "string(Expenses:Food) | decimal(10)"},
		},
		{
			name:  "inner-alias",
			text:  "SELECT x FROM (SELECT account AS x FROM postings) WHERE x ~ 'Cash' ORDER BY x",
			names: []string{"x"},
			types: []string{"string"},
			want:  []string{"string(Assets:Cash)", "string(Assets:Cash)"},
		},
		{
			name:  "nested-derived",
			text:  "SELECT account FROM (SELECT account FROM (SELECT account FROM postings) WHERE account ~ 'Cash') ORDER BY account",
			names: []string{"account"},
			types: []string{"string"},
			want:  []string{"string(Assets:Cash)", "string(Assets:Cash)"},
		},
		{
			name:  "aggregate-over-derived",
			text:  "SELECT sum(number) AS total FROM (SELECT number WHERE number > 0)",
			names: []string{"total"},
			types: []string{"decimal"},
			want:  []string{"decimal(110)"},
		},
		{
			name:  "count-over-derived",
			text:  "SELECT count(*) AS n FROM (SELECT account FROM postings)",
			names: []string{"n"},
			types: []string{"integer"},
			want:  []string{"integer(4)"},
		},
		{
			name:  "star-over-derived",
			text:  "SELECT * FROM (SELECT account FROM postings) WHERE account ~ 'Cash' ORDER BY account",
			names: []string{"account"},
			types: []string{"string"},
			want:  []string{"string(Assets:Cash)", "string(Assets:Cash)"},
		},
		{
			name:  "empty-derived-keeps-schema",
			text:  "SELECT DISTINCT account FROM (SELECT account FROM postings WHERE number > 1000)",
			names: []string{"account"},
			types: []string{"string"},
			want:  nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := query.EvaluateTyped(test.text, evaluation)
			if err != nil {
				t.Fatal(err)
			}
			requireTypedColumns(t, result, test.names, test.types)
			requireTypedCells(t, result, test.want)
		})
	}
}

// TestDerivedSubqueryRejectsUnknownOuterColumns keeps the honest-filtering
// contract for derived tables: a name the inner projection does not carry is a
// compile error, not a silent NULL.
func TestDerivedSubqueryRejectsUnknownOuterColumns(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	for _, text := range []string{
		"SELECT bogus FROM (SELECT account FROM postings)",
		"SELECT account FROM (SELECT account FROM postings) WHERE bogus = 1",
		"SELECT account FROM (SELECT account FROM postings) GROUP BY bogus",
	} {
		_, err := query.EvaluateTyped(text, evaluation)
		var detail query.TypedError
		if !errors.As(err, &detail) || detail.Code != "unknown-column" {
			t.Errorf("query %q err=%v", text, err)
		}
	}
}

// TestPivotBySpreadsMeasureAcrossValueColumns covers the pivot contract: the
// first pivot column stays the row key, the second becomes one column per
// distinct value, remaining projected columns repeat as measures, and absent
// combinations are NULL.
func TestPivotBySpreadsMeasureAcrossValueColumns(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	result, err := query.EvaluateTyped("SELECT account, currency, sum(number) AS s, count(*) AS n GROUP BY account, currency PIVOT BY account, currency", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedColumns(t, result, []string{"account/currency", "USD/s", "USD/n"}, []string{"string", "decimal", "integer"})
	requireTypedCells(t, result, []string{
		"string(Assets:Cash) | decimal(90) | integer(2)",
		"string(Equity:Opening) | decimal(-100) | integer(1)",
		"string(Expenses:Food) | decimal(10) | integer(1)",
	})
	// A NULL combination must stay NULL, and the measure column must declare
	// itself nullable so the empty cell is representable. The multi-currency
	// fixture has a (Equity:Opening, CNY) hole, so the oracle leaves it NULL.
	multicurrency := standardFormsMultiCurrencyEvaluation(t)
	nulls, err := query.EvaluateTyped("SELECT account, currency, sum(number) AS s, count(*) AS n GROUP BY account, currency PIVOT BY account, currency", multicurrency)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedColumns(t, nulls, []string{"account/currency", "CNY/s", "CNY/n", "USD/s", "USD/n"}, []string{"string", "decimal", "integer", "decimal", "integer"})
	requireTypedCells(t, nulls, []string{
		"string(Assets:Cash) | decimal(-70) | integer(1) | decimal(90) | integer(2)",
		"string(Equity:Opening) | <null> | <null> | decimal(-100) | integer(1)",
		"string(Expenses:Food) | decimal(70) | integer(1) | decimal(10) | integer(1)",
	})
	// Every column after the row key is a measure column, and a measure can be
	// absent for some pivot value, so it must declare itself nullable.
	for _, column := range nulls.Columns[1:] {
		if !column.Nullable {
			t.Fatalf("pivot measure columns are always nullable: %+v", nulls.Columns)
		}
	}
	// Pivoting the other way keeps the same holes in a different shape.
	reversed, err := query.EvaluateTyped("SELECT account, currency, sum(number) AS s, count(*) AS n GROUP BY account, currency PIVOT BY currency, account", multicurrency)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedColumns(t, reversed, []string{"currency/account", "Assets:Cash/s", "Assets:Cash/n", "Equity:Opening/s", "Equity:Opening/n", "Expenses:Food/s", "Expenses:Food/n"}, []string{"string", "decimal", "integer", "decimal", "integer", "decimal", "integer"})
	requireTypedCells(t, reversed, []string{
		"string(CNY) | decimal(-70) | integer(1) | <null> | <null> | decimal(70) | integer(1)",
		"string(USD) | decimal(90) | integer(2) | decimal(-100) | integer(1) | decimal(10) | integer(1)",
	})
}

// TestPivotByHonorsOrderByHavingAndLimit places PIVOT BY in the reference's
// clause order (after ORDER BY, before LIMIT) and shows HAVING still filters
// the groups that feed the pivot.
func TestPivotByHonorsOrderByHavingAndLimit(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	limited, err := query.EvaluateTyped("SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY account, currency LIMIT 2", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedCells(t, limited, []string{"string(Assets:Cash) | decimal(90)", "string(Equity:Opening) | decimal(-100)"})
	filtered, err := query.EvaluateTyped("SELECT account, currency, sum(number) GROUP BY account, currency HAVING sum(number) > 0 PIVOT BY account, currency", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedCells(t, filtered, []string{"string(Assets:Cash) | decimal(90)", "string(Expenses:Food) | decimal(10)"})
	// LIMIT 0 keeps the pivoted schema and returns no rows.
	empty, err := query.EvaluateTyped("SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY account, currency LIMIT 0", evaluation)
	// The oracle reports no measure columns for LIMIT 0: with no rows there is
	// no pivot value to spread, so only the row key survives.
	if err != nil || len(empty.Rows) != 0 || len(empty.Columns) != 1 || empty.Columns[0].Name != "account/currency" {
		t.Fatalf("LIMIT 0 result=%+v err=%v", empty, err)
	}
}

// TestPivotByValidationMatchesReference rejects the same PIVOT BY shapes the
// reference rejects: a repeated column, a pivot column outside the projection,
// and a value column that is not a GROUP BY key. A pivot whose projected
// columns are all pivot keys is accepted: with no measure to repeat the
// reference keeps only the row-key column (an oracle observation, not a
// deliberate boundary).
func TestPivotByValidationMatchesReference(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	for _, test := range []struct {
		text, code string
	}{
		{"SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY account, account", "invalid-pivot"},
		{"SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY 1, 1", "invalid-pivot"},
		{"SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY 1, 9", "invalid-pivot"},
		{"SELECT account, sum(number) AS total GROUP BY account PIVOT BY 1, 2", "invalid-pivot"},
		{"SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY 1", "syntax"},
	} {
		_, err := query.EvaluateTyped(test.text, evaluation)
		var detail query.TypedError
		if !errors.As(err, &detail) || detail.Code != test.code {
			t.Errorf("query %q err=%v want code %s", test.text, err, test.code)
		}
	}
	noMeasure, err := query.EvaluateTyped("SELECT account, sum(number) AS s GROUP BY account PIVOT BY s, account", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedColumns(t, noMeasure, []string{"s/account"}, []string{"decimal"})
	requireTypedCells(t, noMeasure, []string{"decimal(-100)", "decimal(10)", "decimal(90)"})
	keysOnly, err := query.EvaluateTyped("SELECT account, currency GROUP BY account, currency PIVOT BY account, currency", evaluation)
	if err != nil {
		t.Fatal(err)
	}
	requireTypedColumns(t, keysOnly, []string{"account/currency"}, []string{"string"})
	requireTypedCells(t, keysOnly, []string{"string(Assets:Cash)", "string(Equity:Opening)", "string(Expenses:Food)"})
}

// TestStandardFormsRespectExecutionBudgetAndCancellation proves every new
// stage charges the shared step budget and stops on cancellation. The per-phase
// assertions reuse the established bounded-execution pattern.
func TestStandardFormsRespectExecutionBudgetAndCancellation(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	texts := map[string]string{
		"distinct": "SELECT DISTINCT account FROM postings",
		"having":   "SELECT account, sum(number) AS total GROUP BY account HAVING sum(number) > 0",
		"subquery": "SELECT account FROM (SELECT account FROM postings WHERE number > 0)",
		"pivot":    "SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY account, currency",
	}
	for name, text := range texts {
		t.Run(name, func(t *testing.T) {
			good, err := query.ExecuteTyped(context.Background(), text, evaluation, query.ExecutionLimits{})
			if err != nil {
				t.Fatal(err)
			}
			if good.ReturnedRows == 0 || len(good.PhaseSteps) == 0 {
				t.Fatalf("execution reported no work: %+v", good)
			}
			var steps uint64
			for _, count := range good.PhaseSteps {
				steps += count
			}
			_, err = query.ExecuteTyped(context.Background(), text, evaluation, query.ExecutionLimits{Steps: steps - 1})
			var failure query.TypedError
			if !errors.As(err, &failure) || failure.Code != "work-budget-exceeded" {
				t.Fatalf("step budget did not stop %s: %v", name, err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := query.ExecuteTyped(ctx, text, evaluation, query.ExecutionLimits{}); !errors.Is(err, context.Canceled) {
				t.Fatalf("%s ignored cancellation: %v", name, err)
			}
		})
	}
	// The pivot reshape itself must charge steps: a limit that only covers
	// everything before it still fails.
	text := "SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY account, currency"
	good, err := query.ExecuteTyped(context.Background(), text, evaluation, query.ExecutionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if good.PhaseSteps["pivot"] == 0 {
		t.Fatal("pivot reshape did not charge the budget")
	}
}

// TestStandardFormsRemainTypedJSONEncodable is the contract guard for the
// saved-analysis path: the new column shapes must encode and decode through
// the declared typed wire format, including NULL pivot cells.
func TestStandardFormsRemainTypedJSONEncodable(t *testing.T) {
	evaluation := standardFormsMultiCurrencyEvaluation(t)
	result, err := query.ExecuteTyped(context.Background(), "SELECT DISTINCT account, currency, sum(number) AS s GROUP BY account, currency PIVOT BY account, currency", evaluation, query.ExecutionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"name":"account/currency"`) || !strings.Contains(string(encoded), `{"type":"null"}`) {
		t.Fatalf("pivot wire shape is not representable: %s", encoded)
	}
	var decoded query.Execution
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Result.Columns) != 3 || len(decoded.Result.Rows) != 3 || decoded.Result.Rows[1][1].Type != "null" {
		t.Fatalf("decoded=%+v", decoded)
	}
}

// TestLegacyEntryAcceptsStandardForms keeps the compatibility entry honest:
// Parse/Evaluate accept the same four forms, and the legacy map result keeps
// its documented behavior (no declared cell types, duplicate titles rejected).
func TestLegacyEntryAcceptsStandardForms(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	distinct, err := query.Evaluate("SELECT DISTINCT account FROM postings ORDER BY account", evaluation)
	if err != nil || len(distinct.Rows) != 3 || distinct.Rows[0]["account"] != "Assets:Cash" {
		t.Fatalf("legacy DISTINCT result=%+v err=%v", distinct, err)
	}
	having, err := query.Evaluate("SELECT account, sum(number) AS total FROM postings GROUP BY account HAVING sum(number) > 0 ORDER BY account", evaluation)
	if err != nil || len(having.Rows) != 2 || formatLegacy(having.Rows[0]["total"]) != "90" {
		t.Fatalf("legacy HAVING result=%+v err=%v", having, err)
	}
	derived, err := query.Evaluate("SELECT account FROM (SELECT account FROM postings WHERE number > 0) ORDER BY account", evaluation)
	if err != nil || len(derived.Rows) != 2 {
		t.Fatalf("legacy derived result=%+v err=%v", derived, err)
	}
	pivot, err := query.Evaluate("SELECT account, currency, sum(number) FROM postings GROUP BY account, currency PIVOT BY 1, 2", evaluation)
	if err != nil || len(pivot.Columns) != 2 || pivot.Columns[0] != "account/currency" || pivot.Columns[1] != "USD" || len(pivot.Rows) != 3 {
		t.Fatalf("legacy pivot result=%+v err=%v", pivot, err)
	}
	// A non-aggregate HAVING is rejected by the reference and by this entry,
	// including through a derived table.
	for _, text := range []string{
		"SELECT account, sum(number) AS total FROM postings GROUP BY account HAVING 1 = 1",
		"SELECT account, sum(number) AS total FROM (SELECT account, number FROM postings) GROUP BY account HAVING 1 = 1",
	} {
		if _, err := query.Evaluate(text, evaluation); err == nil {
			t.Fatalf("legacy non-aggregate HAVING accepted: %s", text)
		}
	}
	// A derived table that references a column it does not carry fails loudly.
	if _, err := query.Evaluate("SELECT bogus FROM (SELECT account FROM postings)", evaluation); err == nil {
		t.Fatal("legacy derived table accepted an unknown column")
	}
	// A pivot whose projected columns are all pivot keys keeps only the
	// row-key column, matching the typed entry and the pinned oracle.
	keysOnly, err := query.Evaluate("SELECT account, sum(number) AS s FROM postings GROUP BY account PIVOT BY s, account", evaluation)
	if err != nil || len(keysOnly.Columns) != 1 || keysOnly.Columns[0] != "s/account" || len(keysOnly.Rows) != 3 {
		t.Fatalf("legacy no-measure pivot result=%+v err=%v", keysOnly, err)
	}
}

// TestLegacyDerivedStarStillValidatesEveryClause keeps the honest-filtering
// contract for `SELECT * FROM (SELECT ...)`: the star only skips its own field
// check, so an unknown name in WHERE, GROUP BY, HAVING or ORDER BY is an
// error instead of a plausible zero-row answer. The pinned oracle rejects all
// four shapes.
func TestLegacyDerivedStarStillValidatesEveryClause(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	for _, text := range []string{
		"SELECT * FROM (SELECT account FROM postings) WHERE bogus = 1",
		"SELECT * FROM (SELECT account FROM postings) GROUP BY bogus",
		"SELECT * FROM (SELECT account FROM postings) ORDER BY bogus",
		"SELECT account FROM (SELECT account FROM postings) ORDER BY bogus",
		"SELECT * FROM (SELECT account FROM postings) WHERE account ~ 'Cash' ORDER BY bogus",
	} {
		if _, err := query.Evaluate(text, evaluation); err == nil {
			t.Errorf("legacy derived query %q returned a plausible result", text)
		}
	}
	// The valid star forms keep working: ORDER BY may name an output title or
	// any inner projection column, even one the outer select does not project.
	star, err := query.Evaluate("SELECT * FROM (SELECT account FROM postings) WHERE account ~ 'Cash' ORDER BY account", evaluation)
	if err != nil || len(star.Rows) != 2 {
		t.Fatalf("valid star derived query result=%+v err=%v", star, err)
	}
	inner, err := query.Evaluate("SELECT account FROM (SELECT account, number FROM postings) ORDER BY number", evaluation)
	if err != nil || len(inner.Rows) != 4 {
		t.Fatalf("ORDER BY inner column result=%+v err=%v", inner, err)
	}
}

// TestPivotCellBudgetBoundsTheDenseMatrixBeforeAllocation proves the pivot
// reshape checks its cell ceiling before allocating the dense matrix. Before
// the check existed a one-row query could reserve a multi-gigabyte matrix and
// only then hit the byte budget.
func TestPivotCellBudgetBoundsTheDenseMatrixBeforeAllocation(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	const text = "SELECT account, currency, sum(number) AS s GROUP BY account, currency PIVOT BY account, currency"
	// The fixture needs 3 row keys x 1 pivot value x 1 measure = 3 cells, so a
	// budget of 2 fails loudly instead of allocating past the ceiling.
	_, err := query.ExecuteTyped(context.Background(), text, evaluation, query.ExecutionLimits{PivotCells: 2})
	var detail query.TypedError
	if !errors.As(err, &detail) || detail.Code != "result-budget-exceeded" {
		t.Fatalf("pivot cell budget was not enforced: %v", err)
	}
	// A budget that covers the matrix keeps the pivot working.
	execution, err := query.ExecuteTyped(context.Background(), text, evaluation, query.ExecutionLimits{PivotCells: 16})
	if err != nil || len(execution.Result.Columns) != 2 || len(execution.Result.Rows) != 3 {
		t.Fatalf("pivot with an adequate cell budget result=%+v err=%v", execution.Result, err)
	}
	if query.QueryPivotCellLimit <= 0 || query.QueryPivotCellLimit > query.QueryByteLimit {
		t.Fatalf("pivot cell limit %d is not a usable ceiling", query.QueryPivotCellLimit)
	}
}

// TestPivotResultWidthRespectsTheRowBudget proves the typed entry cannot return
// a pivot wider than the row ceiling. A PIVOT BY turns rows into columns, so
// checking only len(rows) let a one-row result carry more columns than the
// declared limit while reporting truncated=false.
func TestPivotResultWidthRespectsTheRowBudget(t *testing.T) {
	evaluation := standardFormsEvaluation(t)
	const text = "SELECT account, currency, sum(number) AS s GROUP BY account, currency PIVOT BY account, currency"
	_, err := query.ExecuteTyped(context.Background(), text, evaluation, query.ExecutionLimits{Rows: 1})
	var detail query.TypedError
	if !errors.As(err, &detail) || detail.Code != "result-budget-exceeded" {
		t.Fatalf("over-wide pivot was not rejected: %v", err)
	}
	// A limit that covers both the pivoted width and the row count still
	// returns the whole result without a truncation claim.
	execution, err := query.ExecuteTyped(context.Background(), text, evaluation, query.ExecutionLimits{Rows: 3})
	if err != nil || execution.Truncated || len(execution.Result.Columns) != 2 || len(execution.Result.Rows) != 3 {
		t.Fatalf("pivot within the row budget result=%+v err=%v", execution, err)
	}
}

// TestDistinctFailsLoudlyAtUniqueKeyBudget pins the SELECT DISTINCT
// deduplication ceiling. DISTINCT runs before sort and LIMIT, so its map
// grows with unique input rows, not result rows; a high-cardinality input
// must fail loudly on both entries instead of growing without bound (the
// legacy entry runs without a step budget at all).
func TestDistinctFailsLoudlyAtUniqueKeyBudget(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n")
	for i := 0; i <= query.QueryDistinctKeyLimit; i++ {
		fmt.Fprintf(&builder, "2000-01-02 * \"distinct key %05d\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n", i)
	}
	file, diagnostics := ledger.ParseText("distinct-budget.bean", []byte(builder.String()))
	if diagnostics.HasErrors() {
		t.Fatalf("generated ledger has errors: %v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("generated ledger did not evaluate: %v", evaluation.Diagnostics)
	}
	const text = "SELECT DISTINCT narration FROM postings ORDER BY narration"
	_, err := query.EvaluateTyped(text, *evaluation)
	var detail query.TypedError
	if !errors.As(err, &detail) || detail.Code != "result-budget-exceeded" || !strings.Contains(err.Error(), "unique-key budget") {
		t.Fatalf("typed DISTINCT over the key budget was not rejected: %v", err)
	}
	if _, err := query.Evaluate(text, *evaluation); err == nil || !strings.Contains(err.Error(), "unique-key budget") {
		t.Fatalf("legacy DISTINCT over the key budget was not rejected: %v", err)
	}
	// A DISTINCT well under the ceiling keeps working: the standard fixture
	// has a handful of accounts, so the same query shape still succeeds.
	small, err := query.EvaluateTyped("SELECT DISTINCT account FROM postings", standardFormsEvaluation(t))
	if err != nil || len(small.Rows) == 0 {
		t.Fatalf("DISTINCT below the key budget failed: %v", err)
	}
}

// formatLegacy renders a legacy result value the way Result.WriteCSV does, so
// the assertion does not depend on the concrete numeric type.
func formatLegacy(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
