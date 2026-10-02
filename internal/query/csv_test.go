// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
	"testing"
)

func TestTypedCSVPreservesOrdinalHeadersAndExactNumbers(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT 1 AS v, 2 AS v, decimal('12345678901234567890.0001') AS precise, 'a,b' AS text, NULL AS empty LIMIT 1", typedEvaluation(t))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := result.WriteCSV(&output); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0][0] != "v" || rows[0][1] != "v" || rows[1][0] != "1" || rows[1][1] != "2" || rows[1][2] != "12345678901234567890.0001" || rows[1][3] != "a,b" || rows[1][4] != "" {
		t.Fatalf("invalid CSV: %+v", rows)
	}
}

func TestTypedCSVPreservesFullInventoryCosts(t *testing.T) {
	built := snapshot.BuildWithOptions("../../testdata/fixtures/query/compound-casts.bean", snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if built.Err != nil || built.Snapshot == nil {
		t.Fatal("public fixture failed")
	}
	result, err := query.EvaluateTyped("SELECT sum(position) WHERE account = 'Assets:Precision'", built.Snapshot.Evaluation())
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := result.WriteCSV(&output); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var positions []query.PresentedPosition
	if err := json.Unmarshal([]byte(rows[1][0]), &positions); err != nil {
		t.Fatal(err)
	}
	if len(positions) != 1 || positions[0].Units.Number.Exact != "1" || positions[0].Cost == nil || positions[0].Cost.Number.Exact != "10" || positions[0].Cost.Label == nil || *positions[0].Cost.Label != "a\"b" {
		t.Fatalf("full cost identity lost: %+v", positions)
	}
}

type failedCSVWriter struct{}

func (failedCSVWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestTypedCSVReturnsWriterFailure(t *testing.T) {
	result, err := query.EvaluateTyped("SELECT number LIMIT 1", typedEvaluation(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := result.WriteCSV(failedCSVWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("lost write failure: %v", err)
	}
}
