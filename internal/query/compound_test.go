// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query_test

import (
	"encoding/json"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
)

func compoundEvaluation(t *testing.T) ledger.Evaluation {
	t.Helper()
	result := snapshot.BuildWithOptions("../../testdata/fixtures/query/compound-values.bean", snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if result.Err != nil || result.Snapshot == nil {
		t.Fatalf("fixture: %v %v", result.Err, result.Diagnostics)
	}
	return result.Snapshot.Evaluation()
}

func TestTypedQueryPositionPreservesFinancialValue(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT account, position", compoundEvaluation(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Columns[1].Type != "position" || len(result.Rows) != 19 {
		t.Fatalf("result=%+v", result)
	}
	encoded, err := json.Marshal(result.Rows)
	if err != nil {
		t.Fatal(err)
	}
	var rows [][]struct {
		Type  string
		Value json.RawMessage
	}
	if err := json.Unmarshal(encoded, &rows); err != nil {
		t.Fatal(err)
	}
	wants := map[int]string{
		0:  `{"units":{"number":{"display":"2","exact":"2","approximate":false},"currency":"STK"},"cost":{"number":{"display":"10","exact":"10","approximate":false},"currency":"USD","date":"2000-01-02","label":"alpha"}}`,
		1:  `{"units":{"number":{"display":"-20","exact":"-20","approximate":false},"currency":"USD"},"cost":null}`,
		8:  `{"units":{"number":{"display":"3","exact":"3","approximate":false},"currency":"STK"},"cost":{"number":{"display":"10","exact":"10","approximate":false},"currency":"USD","date":"2000-01-03","label":""}}`,
		12: `{"units":{"number":{"display":"4","exact":"4","approximate":false},"currency":"STK"},"cost":{"number":{"display":"0","exact":"0","approximate":false},"currency":"USD","date":"2000-01-04","label":null}}`,
		16: `{"units":{"number":{"display":"-1","exact":"-1","approximate":false},"currency":"STK"},"cost":{"number":{"display":"10","exact":"10","approximate":false},"currency":"USD","date":"2000-01-02","label":"alpha"}}`,
	}
	for index, want := range wants {
		if string(rows[index][1].Value) != want {
			t.Fatalf("row %d: %s, want %s", index, rows[index][1].Value, want)
		}
	}
}
