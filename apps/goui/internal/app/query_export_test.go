// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"orangecount/apps/goui/internal/localfiles"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

func TestQuerySaveCoordinatorWritesFrozenCSVAfterProtectionRefresh(t *testing.T) {
	model, controller, success := readyQueryModel(t)
	defer closeWorkspaceController(t, controller)
	target := filepath.Join(t.TempDir(), "query.csv")
	options := make(chan mygo.SaveDialogOptions, 1)
	model.querySaveDialog = func(_ context.Context, got mygo.SaveDialogOptions) (string, error) {
		options <- got
		return target, nil
	}
	model.beginQuerySave("csv")
	model.queryDraft = "SELECT 99 AS unrelated_draft"
	captured := waitWorkspaceState(t, controller, func(state workspace.State) bool { return !state.QueryCaptureBusy && state.QueryCapture != nil })
	if captured.QueryCapture.Context != success.QueryResponse.Context || captured.QueryCapture.Format != "csv" {
		t.Fatalf("capture identity changed before Save Panel: %+v", captured.QueryCapture)
	}
	model.workspaceState = captured
	model.finishPendingQuerySave(captured)
	select {
	case got := <-options:
		if !strings.HasSuffix(got.DefaultPath, "-query.csv") || !strings.Contains(got.Message, success.QueryResponse.Context.SnapshotID) || len(got.Filters) != 1 || got.Filters[0].Extensions[0] != "csv" {
			t.Fatalf("Save Panel options do not describe the frozen CSV: %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("query Save Panel workflow did not start")
	}
	waitForFile(t, target)
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "v,v\n1,2\n" {
		t.Fatalf("query export = %q, %v", got, err)
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("query export mode = %v, %v", info, err)
	}
}

func TestQuerySaveCoordinatorCancellationDoesNotWrite(t *testing.T) {
	model, controller, _ := readyQueryModel(t)
	defer closeWorkspaceController(t, controller)
	target := filepath.Join(t.TempDir(), "cancelled.csv")
	started := make(chan struct{}, 1)
	returned := make(chan struct{}, 1)
	model.querySaveDialog = func(ctx context.Context, _ mygo.SaveDialogOptions) (string, error) {
		started <- struct{}{}
		<-ctx.Done()
		returned <- struct{}{}
		return "", ctx.Err()
	}
	model.beginQuerySave("csv")
	captured := waitWorkspaceState(t, controller, func(state workspace.State) bool { return !state.QueryCaptureBusy && state.QueryCapture != nil })
	model.workspaceState = captured
	model.finishPendingQuerySave(captured)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("query Save Panel workflow did not start")
	}
	model.CancelPendingExport()
	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("save dialog did not stop after cancellation")
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled query save created a file: %v", err)
	}
}

func TestQuerySaveCoordinatorRejectsResultAfterQueryChanges(t *testing.T) {
	model, controller, success := readyQueryModel(t)
	defer closeWorkspaceController(t, controller)
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentExecuteQuery, QueryText: "NOT A QUERY"}); err != nil {
		t.Fatal(err)
	}
	state := waitWorkspaceState(t, controller, func(state workspace.State) bool {
		return !state.QueryBusy && state.QueryResponse != nil && state.QueryResponse.Context.RequestID > success.QueryResponse.Context.RequestID
	})
	if state.QueryResponse.Error == nil {
		t.Fatalf("invalid follow-up query unexpectedly succeeded: %+v", state.QueryResponse)
	}
	started := make(chan struct{}, 1)
	model.querySaveDialog = func(context.Context, mygo.SaveDialogOptions) (string, error) {
		started <- struct{}{}
		return "", nil
	}
	model.beginQuerySave("csv")
	if model.pendingQuerySave != nil || model.workspace.State().QueryCaptureBusy {
		t.Fatal("stale successful response was accepted for capture after a newer query failed")
	}
	if !strings.Contains(model.status, "无法准备查询结果保存") {
		t.Fatalf("stale query capture status = %q", model.status)
	}
	select {
	case <-started:
		t.Fatal("Save Panel opened for a stale query response")
	default:
	}
}

func TestQuerySaveCoordinatorCancelsWhenLeavingQueryPage(t *testing.T) {
	model, controller, _ := readyQueryModel(t)
	defer closeWorkspaceController(t, controller)
	started := make(chan struct{}, 1)
	returned := make(chan struct{}, 1)
	model.querySaveDialog = func(ctx context.Context, _ mygo.SaveDialogOptions) (string, error) {
		started <- struct{}{}
		<-ctx.Done()
		returned <- struct{}{}
		return "", ctx.Err()
	}
	model.beginQuerySave("csv")
	captured := waitWorkspaceState(t, controller, func(state workspace.State) bool { return !state.QueryCaptureBusy && state.QueryCapture != nil })
	model.workspaceState = captured
	model.finishPendingQuerySave(captured)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("query Save Panel workflow did not start")
	}
	model.requestView("balance_sheet")
	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("leaving the query page did not cancel the Save Panel")
	}
}

func TestQuerySaveCoordinatorWritesTypedAnalysisDocument(t *testing.T) {
	model, controller, success := readyQueryModel(t)
	defer closeWorkspaceController(t, controller)
	target := filepath.Join(t.TempDir(), "query.ocanalysis")
	options := make(chan mygo.SaveDialogOptions, 1)
	model.querySaveDialog = func(_ context.Context, got mygo.SaveDialogOptions) (string, error) {
		options <- got
		return target, nil
	}
	model.beginQuerySave("analysis")
	captured := waitWorkspaceState(t, controller, func(state workspace.State) bool { return !state.QueryCaptureBusy && state.QueryCapture != nil })
	model.workspaceState = captured
	model.finishPendingQuerySave(captured)
	select {
	case got := <-options:
		if !strings.HasSuffix(got.DefaultPath, "-query.ocanalysis") || len(got.Filters) != 1 || got.Filters[0].Extensions[0] != "ocanalysis" || !strings.Contains(got.Message, "typed cells") {
			t.Fatalf("analysis Save Panel options = %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("analysis Save Panel workflow did not start")
	}
	waitForFile(t, target)
	file, err := os.Open(target)
	if err != nil {
		t.Fatal(err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		t.Fatal(err)
	}
	document, err := desktop.ReadQueryAnalysis(file, info.Size())
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("read saved query analysis: %v, close %v", err, closeErr)
	}
	if document.Context.NormalizedRequest.Text != success.QueryResponse.Context.Text || len(document.Result.Columns) != 2 || len(document.Result.Rows) != 1 {
		t.Fatalf("saved analysis lost typed query result: %+v", document)
	}
}

func TestQuerySaveRefreshesProtectionAfterReloadDuringSavePanel(t *testing.T) {
	root := t.TempDir()
	mainLedger := filepath.Join(root, "main.bean")
	initial := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Expenses:Food USD\n2000-01-02 * \"old source\"\n  Assets:Cash -2 USD\n  Expenses:Food 2 USD\n"
	if err := os.WriteFile(mainLedger, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	model, controller, _ := readyQueryModelAt(t, mainLedger)
	defer closeWorkspaceController(t, controller)
	initialCaptureState := controller.State()
	newSource := filepath.Join(root, "new-source.bean")
	newSourceContent := "2000-01-03 * \"new source\"\n  Assets:Cash -1 USD\n  Expenses:Food 1 USD\n"
	result := make(chan error, 1)
	model.onExportSaveResult = func(_ string, err error) { result <- err }
	model.querySaveDialog = func(_ context.Context, _ mygo.SaveDialogOptions) (string, error) {
		if err := os.WriteFile(newSource, []byte(newSourceContent), 0o600); err != nil {
			return "", err
		}
		if err := os.WriteFile(mainLedger, []byte(initial+"include \"new-source.bean\"\n"), 0o600); err != nil {
			return "", err
		}
		if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentReload}); err != nil {
			return "", err
		}
		if _, ok := pollWorkspaceState(controller, func(state workspace.State) bool {
			return !state.Busy && state.Ledger.Revision > initialCaptureState.Ledger.Revision
		}); !ok {
			return "", errors.New("workspace reload did not finish before protection refresh")
		}
		return newSource, nil
	}
	model.beginQuerySave("csv")
	captured := waitWorkspaceState(t, controller, func(state workspace.State) bool { return !state.QueryCaptureBusy && state.QueryCapture != nil })
	model.workspaceState = captured
	model.finishPendingQuerySave(captured)
	select {
	case err := <-result:
		if !errors.Is(err, localfiles.ErrProtectedTarget) {
			t.Fatalf("save after refreshed source graph = %v, want protected target", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("query save did not finish after protection refresh")
	}
	got, err := os.ReadFile(newSource)
	if err != nil || string(got) != newSourceContent {
		t.Fatalf("newly protected source was changed: %q, %v", got, err)
	}
	final := controller.State()
	if final.QueryCapture == nil || string(final.QueryCapture.Content) != "v,v\n1,2\n" {
		t.Fatal("reload changed the bytes frozen before the Save Panel")
	}
}

func readyQueryModel(t *testing.T) (*Model, *workspace.Controller, workspace.State) {
	t.Helper()
	path, err := filepath.Abs("../../../../testdata/fixtures/query/default-source.bean")
	if err != nil {
		t.Fatal(err)
	}
	return readyQueryModelAt(t, path)
}

func readyQueryModelAt(t *testing.T, path string) (*Model, *workspace.Controller, workspace.State) {
	t.Helper()
	controller := workspace.NewController("query-export-test", workspace.DesktopEngine{}, nil)
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: path, Options: workspace.OpenOptions{
		SemanticsProfile: ledger.V3,
		Watch:            snapshot.WatchOptions{PollInterval: time.Hour},
	}}); err != nil {
		t.Fatal(err)
	}
	state := waitWorkspaceState(t, controller, func(state workspace.State) bool { return state.Phase == workspace.PhaseReady && !state.Busy })
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentExecuteQuery, QueryText: "SELECT 1 AS v, 2 AS v LIMIT 1"}); err != nil {
		t.Fatal(err)
	}
	state = waitWorkspaceState(t, controller, func(state workspace.State) bool { return state.QueryResponse != nil && !state.QueryBusy })
	if state.QueryResponse.Error != nil || state.QueryResponse.Execution == nil {
		t.Fatalf("query fixture did not produce a successful result: %+v", state.QueryResponse)
	}
	model := NewModel(true)
	model.workspace = controller
	model.workspaceState = state
	return model, controller, state
}

func waitWorkspaceState(t *testing.T, controller *workspace.Controller, predicate func(workspace.State) bool) workspace.State {
	t.Helper()
	state, ok := pollWorkspaceState(controller, predicate)
	if !ok {
		t.Fatalf("workspace state did not reach expected condition: %+v", state)
	}
	return state
}

func pollWorkspaceState(controller *workspace.Controller, predicate func(workspace.State) bool) (workspace.State, bool) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := controller.State()
		if predicate(state) {
			return state, true
		}
		time.Sleep(time.Millisecond)
	}
	return controller.State(), false
}

func closeWorkspaceController(t *testing.T, controller *workspace.Controller) {
	t.Helper()
	controller.Close()
	select {
	case <-controller.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("workspace controller did not close")
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("query export was not written to %s", path)
}
