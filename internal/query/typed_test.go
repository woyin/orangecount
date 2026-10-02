// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package query_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
)

func typedEvaluation(t *testing.T) ledger.Evaluation {
	t.Helper()
	result := snapshot.BuildWithOptions("../../testdata/fixtures/query/default-source.bean", snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if result.Err != nil || result.Snapshot == nil {
		t.Fatalf("fixture build failed: %v, %v", result.Err, result.Diagnostics)
	}
	return result.Snapshot.Evaluation()
}

func TestTypedQuerySourceColumnsAndScalarFunctions(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT abs(number) AS magnitude, length(account) AS size, filename, lineno LIMIT 1", typedEvaluation(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0][0].Value.(query.PresentedDecimal).Exact != "2" || result.Rows[0][1].Value != "11" || result.Rows[0][3].Value != "5" {
		t.Fatalf("result=%+v", result)
	}
	if name := filepath.Base(result.Rows[0][2].Value.(string)); name != "default-source.bean" {
		t.Fatalf("source=%s", name)
	}
}

func TestTypedQueryPreservesDuplicateTitlesAndEmptyString(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT 1 AS v, 2 AS v, '' AS blank LIMIT 1", typedEvaluation(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Columns) != 3 || result.Columns[0].ID == result.Columns[1].ID || result.Columns[0].Name != "v" || result.Columns[1].Name != "v" || result.Columns[0].Type != "integer" {
		t.Fatalf("columns=%+v", result.Columns)
	}
	if len(result.Rows) != 1 || result.Rows[0][0].Value != "1" || result.Rows[0][1].Value != "2" || result.Rows[0][2].Value != "" {
		t.Fatalf("rows=%+v", result.Rows)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct{ Rows [][]map[string]any }
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if v, exists := decoded.Rows[0][2]["value"]; !exists || v != "" {
		t.Fatalf("empty string was lost: %s", encoded)
	}
}

func TestTypedQueryDefaultsToPostingsWithDeclaredSchema(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT account, number", typedEvaluation(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Columns) != 2 || result.Columns[0].Type != "string" || result.Columns[1].Type != "decimal" || len(result.Rows) != 2 {
		t.Fatalf("result=%+v", result)
	}
	if result.Rows[0][0].Value != "Assets:Cash" || result.Rows[1][0].Value != "Expenses:Food" {
		t.Fatalf("rows=%+v", result.Rows)
	}
	if got := result.Rows[0][1].Value.(query.PresentedDecimal).Exact; got != "-2" {
		t.Fatalf("number=%s", got)
	}
}

func TestTypedQueryRejectsInvalidWhereFunctionEvenOnEmptyInput(t *testing.T) {
	if _, err := query.EvaluateTyped("SELECT account WHERE unknown(number) = 0", ledger.Evaluation{}); err == nil {
		t.Fatal("empty input suppressed invalid function")
	}
}

func TestTypedQueryLimitZeroRetainsSchemaWithNoRows(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT account, number LIMIT 0", typedEvaluation(t))
	if err != nil || len(result.Columns) != 2 || result.Rows == nil || len(result.Rows) != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestTypedQueryDivisionByZeroReturnsNullableCell(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT number / 0 AS ratio LIMIT 1", typedEvaluation(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Columns[0].Type != "decimal" || !result.Columns[0].Nullable || result.Rows[0][0].Type != "null" {
		t.Fatalf("result=%+v", result)
	}
	encoded, err := json.Marshal(result.Rows[0][0])
	if err != nil || string(encoded) != `{"type":"null"}` {
		t.Fatalf("null=%s err=%v", encoded, err)
	}
}

func TestTypedQueryGroupedTotalsUseAliasesAndIntegerCounts(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT account, sum(number) AS total, count(*) AS count GROUP BY account ORDER BY total DESC", typedEvaluation(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 2 || result.Rows[0][0].Value != "Expenses:Food" || result.Rows[0][1].Value.(query.PresentedDecimal).Exact != "2" || result.Rows[0][2].Value != "1" || result.Columns[2].Type != "integer" {
		t.Fatalf("result=%+v", result)
	}
}

func TestTypedQuerySeparatesPostingAndTransactionMetadata(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT meta('txmeta'), meta('pmeta'), entry_meta('txmeta'), any_meta('txmeta') LIMIT 1", typedEvaluation(t))
	if err != nil {
		t.Fatal(err)
	}
	row := result.Rows[0]
	if row[0].Type != "null" || row[1].Value != "bar" || row[2].Value != "foo" || row[3].Value != "foo" || result.Columns[0].Type != "object" || result.Columns[0].Name != "meta('txmeta')" {
		t.Fatalf("result=%+v", result)
	}
}

func TestTypedQueryRejectsUnrepresentedSystemToleranceObject(t *testing.T) {
	// The pinned loader supplies a dict on ordinary transactions, including
	// an empty dict. An unavailable wire type must not masquerade as NULL.
	for _, fn := range []string{"entry_meta", "any_meta"} {
		_, err := query.EvaluateTyped("SELECT "+fn+"('__tolerances__') LIMIT 1", typedEvaluation(t))
		var detail query.TypedError
		if !errors.As(err, &detail) || detail.Code != "unsupported-capability" {
			t.Errorf("%s lost the system object: %v", fn, err)
		}
	}
	result, err := query.EvaluateTyped("SELECT meta('__tolerances__') LIMIT 1", typedEvaluation(t))
	if err != nil || len(result.Rows) != 1 || result.Rows[0][0].Type != "null" {
		t.Fatalf("posting scope: %+v, %v", result, err)
	}
}

func TestTypedQueryRejectsNonFiniteMetadataDecimalCasts(t *testing.T) {
	// The fixed reference accepts these strings as Decimal values. Our exact
	// rational wire contract cannot represent them; they must not become NULL.
	for _, value := range []string{"NaN", "Infinity"} {
		t.Run(value, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "finite-contract.bean")
			text := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"finite contract\"\n  Assets:Cash 1 USD\n    value: \"" + value + "\"\n  Equity:Opening -1 USD\n"
			if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
				t.Fatal(err)
			}
			build := snapshot.BuildWithOptions(path, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
			if build.Err != nil || build.Snapshot == nil {
				t.Fatalf("build: %v %v", build.Err, build.Diagnostics)
			}
			_, err := query.EvaluateTyped("SELECT meta('value') + 1 LIMIT 1", build.Snapshot.Evaluation())
			var detail query.TypedError
			if !errors.As(err, &detail) || detail.Code != "unsupported-capability" {
				t.Fatalf("non-finite value masqueraded as NULL: %v", err)
			}
		})
	}
}

func TestLegacyQueryRejectsDuplicateTitlesInsteadOfDiscardingValues(t *testing.T) {
	_, err := query.Evaluate("SELECT 1 AS v, 2 AS v FROM postings LIMIT 1", typedEvaluation(t))
	if err == nil {
		t.Fatal("legacy map silently discarded a duplicate output column")
	}
}

func TestTypedQueryErrorsRemainSpecificOnEmptyInput(t *testing.T) {
	for _, test := range []struct{ text, code string }{
		{"SELECT unknown(number)", "unknown-function"},
		{"SELECT abs(account) WHERE account = 'NO'", "type-mismatch"},
		{"SELECT missing", "unknown-column"},
		{"SELECT account FROM unknown", "unknown-table"},
		{"SELECT", "syntax"},
	} {
		_, err := query.EvaluateTyped(test.text, ledger.Evaluation{})
		var detail query.TypedError
		if !errors.As(err, &detail) || detail.Code != test.code {
			t.Errorf("query=%q err=%v code=%s", test.text, err, detail.Code)
		}
	}
}

func TestTypedQueryDoesNotSuppressGroupKeyErrors(t *testing.T) {
	evaluation := typedEvaluation(t)
	for _, entry := range evaluation.Entries {
		if tx, ok := entry.Directive.(*ledger.Transaction); ok {
			tx.Meta = append(tx.Meta, ledger.Metadata{Key: "unrepresentable", Value: ledger.Value{Kind: ledger.ValueList}})
		}
	}
	_, err := query.EvaluateTyped("SELECT entry_meta('unrepresentable') GROUP BY entry_meta('unrepresentable')", evaluation)
	var detail query.TypedError
	if !errors.As(err, &detail) || detail.Code != "unsupported-capability" {
		t.Fatalf("group key error suppressed: %v", err)
	}
}

func TestTypedQueryGeneratedPadPostingsUseBalanceMetadataSource(t *testing.T) {
	build := snapshot.BuildWithOptions("../../testdata/fixtures/query/pad-source.bean", snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if build.Err != nil || build.Snapshot == nil {
		t.Fatalf("build=%+v", build)
	}
	result, err := query.EvaluateTyped("SELECT filename, lineno", build.Snapshot.Evaluation())
	if err != nil || len(result.Rows) != 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	for _, row := range result.Rows {
		if row[0].Type != "string" || row[1].Value != "4" {
			t.Fatalf("generated source must match balance metadata: %+v", row)
		}
	}
}

func TestTypedQueryGroupingRetainsMetadataTypes(t *testing.T) {
	build := snapshot.BuildWithOptions("../../testdata/fixtures/query/typed-grouping.bean", snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if build.Err != nil || build.Snapshot == nil {
		t.Fatalf("build=%+v", build)
	}
	result, err := query.EvaluateTyped("SELECT meta('key') AS key, count(*) AS n WHERE account = 'Expenses:Food' GROUP BY key", build.Snapshot.Evaluation())
	if err != nil || len(result.Rows) != 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if result.Rows[0][0].Type != "string" || result.Rows[1][0].Type != "decimal" || result.Rows[0][1].Value != "1" || result.Rows[1][1].Value != "1" {
		t.Fatalf("rows=%+v", result.Rows)
	}
}

func TestTypedQueryDateLiteralCannotBecomeIntegerSubtraction(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT 2000-01-02 AS day LIMIT 1", typedEvaluation(t))
	if err != nil || result.Columns[0].Type != "date" || result.Rows[0][0].Value != "2000-01-02" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestTypedQueryDoesNotCoerceNumericStringsInComparison(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT '2' < '10' AS ordered LIMIT 1", typedEvaluation(t))
	if err != nil || result.Rows[0][0].Value != false {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestTypedQueryRejectsInvalidRegexTypesBeforeFiltering(t *testing.T) {
	_, err := query.EvaluateTyped("SELECT number ~ number WHERE account = 'NO'", typedEvaluation(t))
	var detail query.TypedError
	if !errors.As(err, &detail) || detail.Code != "type-mismatch" {
		t.Fatalf("invalid regex signature accepted: %v", err)
	}
}

func TestTypedQuerySortsNumericStringsLexically(t *testing.T) {
	build := snapshot.BuildWithOptions("../../testdata/fixtures/query/scalar-order.bean", snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if build.Err != nil || build.Snapshot == nil {
		t.Fatalf("build=%+v", build)
	}
	result, err := query.EvaluateTyped("SELECT narration ORDER BY narration", build.Snapshot.Evaluation())
	if err != nil || len(result.Rows) != 4 || result.Rows[0][0].Value != "10" || result.Rows[2][0].Value != "2" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
