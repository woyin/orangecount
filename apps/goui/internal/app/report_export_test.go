// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/localfiles"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/analysis"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

func TestReportSaveCoordinatorWritesCSVFromCurrentProjection(t *testing.T) {
	model, controller, response := readyReportModel(t)
	defer closeWorkspaceController(t, controller)
	model.statementSearch = "Assets:Cash"
	if current := model.currentReportResponse(); current == nil || current.ReportExport.ProjectionID != response.ReportExport.ProjectionID {
		t.Fatal("local tree search changed the frozen complete-projection identity")
	}
	texts := strings.Join(ui.NewTester(model.View, 1180, 780).Texts(), "\n")
	if !strings.Contains(texts, "保存报表 CSV…") || !strings.Contains(texts, "保存报表分析文件…") {
		t.Fatalf("report export controls are missing from the matching report page: %q", texts)
	}
	target := filepath.Join(t.TempDir(), "balance-sheet.csv")
	options := make(chan mygo.SaveDialogOptions, 1)
	model.querySaveDialog = func(_ context.Context, got mygo.SaveDialogOptions) (string, error) {
		options <- got
		return target, nil
	}
	model.beginReportSave("csv")
	captured := waitWorkspaceState(t, controller, func(state workspace.State) bool { return !state.ReportCaptureBusy && state.ReportCapture != nil })
	if captured.ReportCapture.Identity != *response.ReportExport || captured.ReportCapture.Format != "csv" {
		t.Fatalf("report capture identity changed: %+v", captured.ReportCapture)
	}
	model.workspaceState = captured
	model.finishPendingReportSave(captured)
	select {
	case got := <-options:
		if !strings.HasSuffix(got.DefaultPath, ".csv") || len(got.Filters) != 1 || got.Filters[0].Extensions[0] != "csv" || !strings.Contains(got.Message, "完整投影") || !strings.Contains(got.Message, response.ReportExport.SemanticsProfile) {
			t.Fatalf("Save Panel does not describe report projection: %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("report Save Panel workflow did not start")
	}
	waitForFile(t, target)
	content, err := os.ReadFile(target)
	if err != nil || !strings.Contains(strings.ToLower(string(content)), "account") || len(content) == 0 {
		t.Fatalf("report CSV = %q, %v", content, err)
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("report export mode = %v, %v", info, err)
	}
}

func TestReportSaveCoordinatorWritesAnalysisProjection(t *testing.T) {
	model, controller, response := readyReportModel(t)
	defer closeWorkspaceController(t, controller)
	target := filepath.Join(t.TempDir(), "balance-sheet.ocanalysis")
	options := make(chan mygo.SaveDialogOptions, 1)
	model.querySaveDialog = func(_ context.Context, got mygo.SaveDialogOptions) (string, error) {
		options <- got
		return target, nil
	}
	model.beginReportSave("analysis")
	captured := waitWorkspaceState(t, controller, func(state workspace.State) bool { return !state.ReportCaptureBusy && state.ReportCapture != nil })
	model.workspaceState = captured
	model.finishPendingReportSave(captured)
	select {
	case got := <-options:
		if len(got.Filters) != 1 || got.Filters[0].Extensions[0] != "ocanalysis" || !strings.Contains(got.Message, "schema") {
			t.Fatalf("analysis Save Panel options = %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("analysis Save Panel workflow did not start")
	}
	waitForFile(t, target)
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	document, err := analysis.ReadReport(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("decode saved report analysis: %v", err)
	}
	if document.Context.SnapshotID != response.ReportExport.SnapshotID || document.Context.Section != response.ReportExport.Section || len(document.Rows) == 0 {
		t.Fatalf("saved analysis lost report identity or rows: %+v", document.Context)
	}
}

func TestStatementSavePanelShowsCoreCurrencyAndValuationIdentity(t *testing.T) {
	path, err := filepath.Abs("../../../../testdata/fixtures/native/valuation.bean")
	if err != nil {
		t.Fatal(err)
	}
	model, controller, _ := readyReportModelAt(t, path)
	defer closeWorkspaceController(t, controller)
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: desktop.Request{
		View: "balance_sheet", Currency: "USD", Valuation: "market-value", Locale: "zh-CN",
	}}); err != nil {
		t.Fatal(err)
	}
	state := waitWorkspaceState(t, controller, func(state workspace.State) bool {
		return !state.Busy && state.Response != nil && state.Response.Request.Valuation == "market-value" && state.Response.Request.Currency == "USD"
	})
	model.workspaceState = state
	if got := state.Response.ReportExport; got == nil || got.Currency != "USD" || got.Valuation != "market-value" {
		t.Fatalf("workspace report identity lost core options: %+v", got)
	}
	message := make(chan string, 1)
	model.querySaveDialog = func(_ context.Context, options mygo.SaveDialogOptions) (string, error) {
		message <- options.Message
		return "", nil
	}
	model.beginReportSave("analysis")
	captured := waitWorkspaceState(t, controller, func(state workspace.State) bool { return !state.ReportCaptureBusy && state.ReportCapture != nil })
	model.workspaceState = captured
	model.finishPendingReportSave(captured)
	select {
	case got := <-message:
		if !strings.Contains(got, "报表口径 USD / 按市价") {
			t.Fatalf("Save Panel did not show the core report currency and valuation: %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("report Save Panel message was not produced")
	}
}

func TestReportSaveCoordinatorCancellationDoesNotWrite(t *testing.T) {
	model, controller, _ := readyReportModel(t)
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
	model.beginReportSave("csv")
	captured := waitWorkspaceState(t, controller, func(state workspace.State) bool { return !state.ReportCaptureBusy && state.ReportCapture != nil })
	model.workspaceState = captured
	model.finishPendingReportSave(captured)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("report Save Panel workflow did not start")
	}
	model.cancelPendingExport()
	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("report save dialog did not stop after cancellation")
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled report save created a file: %v", err)
	}
}

func TestReportSaveRefreshesProtectionAfterReloadDuringSavePanel(t *testing.T) {
	root := t.TempDir()
	mainLedger := filepath.Join(root, "main.bean")
	initial := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"old source\"\n  Assets:Cash 10 USD\n  Equity:Opening -10 USD\n"
	if err := os.WriteFile(mainLedger, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	model, controller, _ := readyReportModelAt(t, mainLedger)
	defer closeWorkspaceController(t, controller)
	initialRevision := controller.State().Ledger.Revision
	newSource := filepath.Join(root, "new-source.bean")
	newSourceContent := "2000-01-03 * \"new source\"\n  Assets:Cash -1 USD\n  Equity:Opening 1 USD\n"
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
			return !state.Busy && state.Ledger.Revision > initialRevision
		}); !ok {
			return "", errors.New("report workspace reload did not finish before protection refresh")
		}
		return newSource, nil
	}
	model.beginReportSave("csv")
	captured := waitWorkspaceState(t, controller, func(state workspace.State) bool { return !state.ReportCaptureBusy && state.ReportCapture != nil })
	model.workspaceState = captured
	model.finishPendingReportSave(captured)
	select {
	case err := <-result:
		if !errors.Is(err, localfiles.ErrProtectedTarget) {
			t.Fatalf("report save after protection refresh = %v, want protected target", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("report save did not finish after protection refresh")
	}
	if got, err := os.ReadFile(newSource); err != nil || string(got) != newSourceContent {
		t.Fatalf("newly protected report source was changed: %q, %v", got, err)
	}
	if capture := controller.State().ReportCapture; capture == nil || !strings.Contains(string(capture.Content), "Assets:Cash") {
		t.Fatal("reload changed or discarded the report bytes frozen before the Save Panel")
	}
}

func TestReportExportControlsCoverSupportedReportPages(t *testing.T) {
	cases := []struct {
		name, fixture, view  string
		page                 page
		prefetchHoldingGroup bool
		configure            func(*Model)
	}{
		{name: "dashboard", fixture: "history.bean", view: "dashboard", page: pageDashboard},
		{name: "balance sheet", fixture: "valuation.bean", view: "balance_sheet", page: pageBalance},
		{name: "income statement", fixture: "cashflow.bean", view: "income_statement", page: pageIncome},
		{name: "journal", fixture: "cashflow.bean", view: "journal", page: pageJournal},
		{name: "account", fixture: "account-detail.bean", view: "account", page: pageAccount, configure: func(model *Model) {
			model.accountName = "Assets:Test"
			model.accountScope = "subtree"
		}},
		{name: "prices", fixture: "prices.bean", view: "prices", page: pagePrices},
		{name: "price history", fixture: "prices.bean", view: "prices", page: pagePrices, configure: func(model *Model) {
			model.priceSection, model.priceBase, model.priceQuote = "history", "STK", "USD"
		}},
		{name: "holdings", fixture: "holdings-trades.bean", view: "holdings", page: pageHoldings},
		{name: "holding events", fixture: "holdings-trades.bean", view: "holdings", page: pageHoldings, configure: func(model *Model) { model.holdingSection = "events" }},
		{name: "holding history", fixture: "average-history.bean", view: "holdings", page: pageHoldings, configure: func(model *Model) { model.holdingSection = "history" }},
		{name: "holding lots", fixture: "holdings-trades.bean", view: "holdings", page: pageHoldings, prefetchHoldingGroup: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, err := filepath.Abs("../../../../testdata/fixtures/native/" + tc.fixture)
			if err != nil {
				t.Fatal(err)
			}
			controller := workspace.NewController("report-page-"+tc.name, workspace.DesktopEngine{}, nil)
			defer closeWorkspaceController(t, controller)
			if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: path, Options: workspace.OpenOptions{
				SemanticsProfile: ledger.V3,
				Watch:            snapshot.WatchOptions{PollInterval: time.Hour},
			}}); err != nil {
				t.Fatal(err)
			}
			state := waitWorkspaceState(t, controller, func(state workspace.State) bool { return state.Phase == workspace.PhaseReady && !state.Busy })
			model := NewModel(true)
			model.page, model.workspace, model.workspaceState = tc.page, controller, state
			if tc.configure != nil {
				tc.configure(model)
			}
			if tc.prefetchHoldingGroup {
				if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: model.requestForView("holdings")}); err != nil {
					t.Fatal(err)
				}
				state = waitWorkspaceState(t, controller, func(state workspace.State) bool {
					return !state.Busy && state.Response != nil && state.Response.Holdings != nil
				})
				if len(state.Response.Holdings.Groups) == 0 {
					t.Fatal("holdings fixture has no group for its lots projection")
				}
				model.workspaceState = state
				model.holdingSection = "lots"
				model.holdingGroupID = state.Response.Holdings.Groups[0].ID
			}
			if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: model.requestForView(tc.view)}); err != nil {
				t.Fatal(err)
			}
			state = waitWorkspaceState(t, controller, func(state workspace.State) bool {
				return !state.Busy && state.Response != nil && state.Response.ReportExport != nil
			})
			model.workspaceState = state
			if model.currentReportResponse() == nil {
				t.Fatalf("current report page %q did not expose its export identity: %+v", tc.name, state.Response.ReportExport)
			}
			texts := strings.Join(ui.NewTester(model.View, 1180, 780).Texts(), "\n")
			if !strings.Contains(texts, "保存报表 CSV…") || !strings.Contains(texts, "保存报表分析文件…") || !strings.Contains(texts, "打开源文件") {
				t.Fatalf("report actions are missing on %q: %s", tc.name, texts)
			}
		})
	}
}

func TestReportExportControlsAreHiddenOnDirectivePage(t *testing.T) {
	model := NewModel(true)
	model.page = pageDirectives
	model.workspaceState.Ledger.SnapshotID = "snapshot-directives"
	model.workspaceState.Response = &desktop.ViewResponse{
		State: desktop.State{SnapshotID: "snapshot-directives"},
		ReportExport: &desktop.ReportExportIdentity{
			ProjectionID: "not-a-directive-export", SnapshotID: "snapshot-directives", CaptureScope: "current-page",
		},
	}
	texts := strings.Join(ui.NewTester(model.View, 1180, 780).Texts(), "\n")
	if strings.Contains(texts, "保存报表 CSV…") || strings.Contains(texts, "保存报表分析文件…") {
		t.Fatalf("unsupported directive page advertised a report export: %s", texts)
	}
}

func TestSourceOpenRequiresCurrentWorkspaceResponse(t *testing.T) {
	model, controller, _ := readyReportModel(t)
	defer closeWorkspaceController(t, controller)
	opened := ""
	model.openSourcePath = func(path string) error { opened = path; return nil }
	identity := model.workspaceState
	model.openSourceText(identity.ActivePath, identity.Epoch, identity.Operation, identity.Ledger.SnapshotID)
	if opened != identity.ActivePath {
		t.Fatalf("current source open path = %q, want %q", opened, identity.ActivePath)
	}
	opened = ""
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: desktop.Request{View: "income_statement", Currency: "USD", Valuation: "at-cost", Locale: "zh-CN"}}); err != nil {
		t.Fatal(err)
	}
	waitWorkspaceState(t, controller, func(state workspace.State) bool {
		return !state.Busy && state.Response != nil && state.Response.Request.View == "income_statement"
	})
	model.openSourceText(identity.ActivePath, identity.Epoch, identity.Operation, identity.Ledger.SnapshotID)
	if opened != "" || !strings.Contains(model.status, "没有打开来源") {
		t.Fatalf("stale source action opened %q or did not report rejection: %q", opened, model.status)
	}
}

func readyReportModel(t *testing.T) (*Model, *workspace.Controller, desktop.ViewResponse) {
	t.Helper()
	path, err := filepath.Abs("../../../../testdata/fixtures/query/default-source.bean")
	if err != nil {
		t.Fatal(err)
	}
	return readyReportModelAt(t, path)
}

func readyReportModelAt(t *testing.T, path string) (*Model, *workspace.Controller, desktop.ViewResponse) {
	t.Helper()
	controller := workspace.NewController("report-export-test", workspace.DesktopEngine{}, nil)
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: path, Options: workspace.OpenOptions{
		SemanticsProfile: ledger.V3,
		Watch:            snapshot.WatchOptions{PollInterval: time.Hour},
	}}); err != nil {
		t.Fatal(err)
	}
	waitWorkspaceState(t, controller, func(state workspace.State) bool { return state.Phase == workspace.PhaseReady && !state.Busy })
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: desktop.Request{
		View: "balance_sheet", Currency: "USD", Valuation: "at-cost", Locale: "zh-CN",
	}}); err != nil {
		t.Fatal(err)
	}
	state := waitWorkspaceState(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Tree != nil && !state.Busy
	})
	if state.Response.ReportExport == nil {
		t.Fatal("core did not return an export identity for balance sheet")
	}
	model := NewModel(true)
	model.page = pageBalance
	model.workspace = controller
	model.workspaceState = state
	return model, controller, *state.Response
}
