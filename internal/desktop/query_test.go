// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

func TestQueryExecutionCapturesSnapshotTextAndDuplicateColumns(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/query/default-source.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: "SELECT 1 AS v, 2 AS v LIMIT 1"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Error != nil || response.Execution == nil || response.Context.SnapshotID != state.SnapshotID || response.Context.EvaluationRevision != state.EvaluationRevision || response.Context.Text != "SELECT 1 AS v, 2 AS v LIMIT 1" || response.Context.RequestID != 1 || len(response.Context.Fingerprint) != 64 {
		t.Fatalf("invalid execution: %+v", response)
	}
	result := response.Execution.Result
	if len(result.Columns) != 2 || result.Columns[0].Name != "v" || result.Columns[1].Name != "v" || result.Columns[0].ID == result.Columns[1].ID || result.Rows[0][0].Value != "1" || result.Rows[0][1].Value != "2" {
		t.Fatalf("typed positions lost: %+v", result)
	}
}

func TestQueryCancellationIsRememberedBeforeExecutionStarts(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/query/default-source.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	if err := session.CancelQuery(1); err != nil {
		t.Fatal(err)
	}
	response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: "SELECT number"})
	if err != nil || response.Execution != nil || response.Error == nil || response.Error.Code != "cancelled" {
		t.Fatalf("queued cancellation lost: %+v, %v", response, err)
	}
}

func TestQueryExecutionKeepsSuccessFailureAndSnapshotChangesSeparate(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "main.bean")
	original, err := os.ReadFile("../../testdata/fixtures/query/default-source.bean")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3, Watch: snapshot.WatchOptions{PollInterval: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	request := desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: "SELECT account LIMIT 0"}
	first, err := session.ExecuteQuery(context.Background(), request)
	if err != nil || first.Execution == nil || first.Execution.ReturnedRows != 0 || len(first.Execution.Result.Columns) != 1 {
		t.Fatalf("zero rows lost schema: %+v, %v", first, err)
	}
	request.RequestID = 2
	same, err := session.ExecuteQuery(context.Background(), request)
	if err != nil || same.Context.Fingerprint != first.Context.Fingerprint {
		t.Fatal("delivery generation changed the captured query identity")
	}
	request.RequestID = 3
	request.Text = "SELECT invalid_function(number) LIMIT 0"
	failed, err := session.ExecuteQuery(context.Background(), request)
	if err != nil || failed.Error == nil || failed.Error.Code != "unknown-function" || failed.Execution != nil {
		t.Fatalf("failure reused old success: %+v, %v", failed, err)
	}
	if err := os.WriteFile(path, append(original, []byte("\n2000-01-03 price USD 1 EUR\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if err := session.Reload(); err != nil {
		t.Fatal(err)
	}
	request.RequestID = 4
	if _, err := session.ExecuteQuery(context.Background(), request); !errors.Is(err, desktop.ErrStale) {
		t.Fatalf("old snapshot accepted: %v", err)
	}
}

func TestQueryCancelStopsRunningGoWorkAndIsolatesLaterGenerations(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/query/default-source.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	expression := "number"
	for i := 0; i < 18; i++ {
		expression = "(" + expression + " + 0.0)"
	}
	type outcome struct {
		response desktop.QueryResponse
		err      error
	}
	done := make(chan outcome, 1)
	go func() {
		response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: "SELECT str(" + expression + ") LIMIT 1"})
		done <- outcome{response, err}
	}()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case got := <-done:
			if got.err != nil || got.response.Execution != nil || got.response.Error == nil || got.response.Error.Code != "cancelled" {
				t.Fatalf("real execution did not cancel: %+v, %v", got.response, got.err)
			}
			response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 2, Text: "SELECT 1 LIMIT 1"})
			if err != nil || response.Execution == nil {
				t.Fatalf("later generation remained cancelled: %+v, %v", response, err)
			}
			late, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: "SELECT 2 LIMIT 1"})
			if err != nil || late.Error == nil || late.Error.Code != "cancelled" || late.Execution != nil {
				t.Fatalf("late older generation executed: %+v, %v", late, err)
			}
			return
		case <-ticker.C:
			if err := session.CancelQuery(1); err != nil {
				t.Fatal(err)
			}
		case <-timer.C:
			t.Fatal("cancelled query kept running")
		}
	}
}
