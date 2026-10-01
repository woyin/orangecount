// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
)

func TestBoundedQueryCapsReturnedRowsAfterOrdering(t *testing.T) {
	evaluation := typedEvaluation(t)
	tx := evaluation.Entries[len(evaluation.Entries)-1]
	evaluation.Entries = make([]ledger.EntryRecord, 5001)
	for i := range evaluation.Entries {
		evaluation.Entries[i] = tx
	}
	got, err := query.ExecuteTyped(context.Background(), "SELECT number ORDER BY number DESC", evaluation, query.ExecutionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if got.ReturnedRows != 10000 || len(got.Result.Rows) != 10000 || !got.Truncated || got.RowLimit != 10000 {
		t.Fatalf("rows=%d returned=%d truncated=%t limit=%d", len(got.Result.Rows), got.ReturnedRows, got.Truncated, got.RowLimit)
	}
	if got.Result.Rows[0][0].Value.(query.PresentedDecimal).Exact != "2" || got.Result.Rows[5001][0].Value.(query.PresentedDecimal).Exact != "-2" {
		t.Fatal("row budget was applied before ORDER BY")
	}
	limited, err := query.ExecuteTyped(context.Background(), "SELECT number LIMIT 10", evaluation, query.ExecutionLimits{})
	if err != nil || limited.Truncated || limited.ReturnedRows != 10 {
		t.Fatalf("explicit LIMIT: %+v, %v", limited, err)
	}
}

func TestBoundedQueryInterruptsDeepDecimalExpression(t *testing.T) {
	expression := "number"
	for i := 0; i < 18; i++ {
		expression = "(" + expression + " + 0.0)"
	}
	_, err := query.ExecuteTyped(context.Background(), "SELECT str("+expression+") LIMIT 1", typedEvaluation(t), query.ExecutionLimits{Steps: 300})
	var failure query.TypedError
	if !errors.As(err, &failure) || failure.Code != "work-budget-exceeded" {
		t.Fatalf("deep repeated calculation was not stopped: %v", err)
	}
}

func TestBoundedQueryBudgetsActualWhereGroupingAggregationAndSorting(t *testing.T) {
	text := "SELECT account, sum(number) AS total WHERE number != 0 GROUP BY account ORDER BY total DESC"
	good, err := query.ExecuteTyped(context.Background(), text, typedEvaluation(t), query.ExecutionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	var before uint64
	for _, phase := range []string{"parse", "source", "where", "group", "project", "sort", "encode"} {
		count := good.PhaseSteps[phase]
		if count == 0 {
			t.Fatalf("stage %s did not execute", phase)
		}
		if phase == "where" || phase == "group" || phase == "project" || phase == "sort" {
			_, err := query.ExecuteTyped(context.Background(), text, typedEvaluation(t), query.ExecutionLimits{Steps: before + count - 1})
			var failure query.TypedError
			if !errors.As(err, &failure) || failure.Code != "work-budget-exceeded" || failure.Phase != phase {
				t.Fatalf("%s was not interrupted: %v", phase, err)
			}
		}
		before += count
	}
}

func TestBoundedQueryTextBytesAndNumericComplexity(t *testing.T) {
	text := "SELECT 1 LIMIT 0"
	remaining := query.QueryTextLimit - len(text)
	text += strings.Repeat("\u3000", remaining/3) + strings.Repeat(" ", remaining%3)
	if _, err := query.ExecuteTyped(context.Background(), text, typedEvaluation(t), query.ExecutionLimits{}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		text, code string
		limits     query.ExecutionLimits
	}{
		{text + " ", "text-budget-exceeded", query.ExecutionLimits{}},
		{"SELECT narration", "result-budget-exceeded", query.ExecutionLimits{Bytes: 64}},
		{"SELECT " + strings.Repeat("(", 80) + "1" + strings.Repeat(")", 80), "complexity-budget-exceeded", query.ExecutionLimits{}},
		{"SELECT decimal('1e100000000') LIMIT 1", "numeric-budget-exceeded", query.ExecutionLimits{}},
	} {
		_, err := query.ExecuteTyped(context.Background(), tc.text, typedEvaluation(t), tc.limits)
		var failure query.TypedError
		if !errors.As(err, &failure) || failure.Code != tc.code {
			t.Fatalf("wanted %s: %v", tc.code, err)
		}
	}
}

type activeCancellationContext struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *activeCancellationContext) Err() error {
	c.checks++
	if c.checks == 100 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestBoundedQueryChecksCancellationWhileRunning(t *testing.T) {
	evaluation := typedEvaluation(t)
	tx := evaluation.Entries[len(evaluation.Entries)-1]
	evaluation.Entries = make([]ledger.EntryRecord, 1000)
	for i := range evaluation.Entries {
		evaluation.Entries[i] = tx
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	active := &activeCancellationContext{Context: ctx, cancel: cancel}
	_, err := query.ExecuteTyped(active, "SELECT sum(number) WHERE number != 0", evaluation, query.ExecutionLimits{})
	if !errors.Is(err, context.Canceled) || active.checks != 100 {
		t.Fatalf("cancel not checked during execution: checks=%d, error=%v", active.checks, err)
	}
}

func TestBoundedQueryByteBudgetCountsReturnedRowsNotTruncationWitness(t *testing.T) {
	text := "SELECT '" + strings.Repeat("x", 1000) + "'"
	good, err := query.ExecuteTyped(context.Background(), text, typedEvaluation(t), query.ExecutionLimits{Rows: 1})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(good)
	if err != nil {
		t.Fatal(err)
	}
	same, err := query.ExecuteTyped(context.Background(), text, typedEvaluation(t), query.ExecutionLimits{Rows: 1, Bytes: len(encoded)})
	if err != nil || same.ReturnedRows != 1 || !same.Truncated {
		t.Fatalf("discarded witness consumed response budget: bytes=%d, result=%+v, error=%v", len(encoded), same, err)
	}
}

func TestBoundedQueryRejectsGrowingIntermediateNumbers(t *testing.T) {
	_, err := query.ExecuteTyped(context.Background(), "SELECT decimal('1e10000') * decimal('1e10000') LIMIT 1", typedEvaluation(t), query.ExecutionLimits{})
	var failure query.TypedError
	if !errors.As(err, &failure) || failure.Code != "numeric-budget-exceeded" {
		t.Fatalf("large computed number escaped the numeric budget: %v", err)
	}
}
