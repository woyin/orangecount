// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

// TestPrivateLedgerUIReadOnlyProjection is opt-in. It intentionally logs only
// the selected profile, a one-way snapshot digest, view IDs, and node counts.
// It never logs the ledger path, projected text, values, or diagnostics.
func TestPrivateLedgerUIReadOnlyProjection(t *testing.T) {
	path := os.Getenv("ORANGECOUNT_PRIVATE_LEDGER")
	if path == "" {
		t.Skip("set ORANGECOUNT_PRIVATE_LEDGER to run the opt-in read-only UI comparison")
	}
	profile := ledger.Legacy
	switch os.Getenv("ORANGECOUNT_PRIVATE_PROFILE") {
	case "", "legacy":
	case "v3":
		profile = ledger.V3
	default:
		t.Fatal("ORANGECOUNT_PRIVATE_PROFILE must be legacy or v3; source details omitted")
	}

	if !filepath.IsAbs(path) {
		t.Fatal("ORANGECOUNT_PRIVATE_LEDGER must be an absolute path; source details omitted")
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatal("private ledger entry is unavailable; source details omitted")
	}

	options := desktop.OpenOptions{
		SemanticsProfile: profile,
		Watch:            snapshot.WatchOptions{PollInterval: time.Hour},
	}
	coreSession, err := desktop.OpenWithOptions(path, options)
	if err != nil {
		t.Fatalf("core read-only open failed (%T); source details omitted", err)
	}
	t.Cleanup(coreSession.Close)
	coreState, err := coreSession.Poll()
	if err != nil || !coreState.Valid || !coreState.HasSnapshot {
		t.Fatal("core did not publish a valid snapshot; source details omitted")
	}

	controller := workspace.NewController("private-read", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(10 * time.Second):
			t.Error("Go UI workspace did not close; source details omitted")
		}
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: path, Options: workspace.OpenOptions{
		SemanticsProfile: profile,
		Watch:            options.Watch,
	}}); err != nil {
		t.Fatalf("Go UI workspace open failed (%T); source details omitted", err)
	}
	workspaceState := waitPrivateLedgerState(t, controller, func(state workspace.State) bool {
		return state.Phase == workspace.PhaseReady && !state.Busy
	})
	if workspaceState.Ledger.SnapshotID != coreState.SnapshotID || workspaceState.Ledger.SemanticsProfile != profile {
		t.Fatalf("core and Go UI opened different snapshot/profile identities (core=%s, ui=%s; profile=%s)", privateSnapshotDigest(coreState.SnapshotID), privateSnapshotDigest(workspaceState.Ledger.SnapshotID), profile)
	}

	views := []struct {
		id, heading string
		page        page
	}{
		{"dashboard", "财务概览", pageDashboard},
		{"balance_sheet", "资产负债表", pageBalance},
		{"income_statement", "损益表", pageIncome},
		{"journal", "流水", pageJournal},
		{"prices", "商品与价格", pagePrices},
		{"holdings", "持仓与成本", pageHoldings},
		{"directives", "全部指令", pageDirectives},
		{"attachments", "附件", pageAttachments},
		{"migration", "语义配置", pageSemantics},
		{"query", "查询工作台", pageQuery},
	}
	for _, view := range views {
		model := NewModel(true)
		model.workspace = controller
		model.workspaceState = workspaceState
		model.page = view.page
		request := model.requestForView(view.id)
		request.SnapshotID = coreState.SnapshotID
		coreResponse, err := coreSession.Read(request)
		if err != nil {
			t.Fatalf("core read failed for view %s (%T); source details omitted", view.id, err)
		}
		if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: request}); err != nil {
			t.Fatalf("Go UI read failed for view %s (%T); source details omitted", view.id, err)
		}
		workspaceState = waitPrivateLedgerState(t, controller, func(state workspace.State) bool {
			return !state.Busy && state.Response != nil && state.Response.Request == coreResponse.Request
		})
		workspaceResponse := *workspaceState.Response
		comparableCore := privateComparableResponse(coreResponse)
		comparableWorkspace := privateComparableResponse(workspaceResponse)
		if !reflect.DeepEqual(comparableCore, comparableWorkspace) {
			t.Errorf("core and Go UI projections differ for view %s (fields=%v state_fields=%v); private values omitted", view.id, privateResponseDifferenceFields(comparableCore, comparableWorkspace), privateDesktopStateDifferenceFields(comparableCore.State, comparableWorkspace.State))
			continue
		}

		model.workspaceState = workspaceState
		tester := ui.NewTester(model.View, 1180, 780)
		if !tester.HasText(view.heading) {
			t.Errorf("Go UI did not render the expected heading for view %s; private values omitted", view.id)
			continue
		}
		t.Logf("view=%s core_workspace=match ui_heading=present rendered_nodes=%d", view.id, len(tester.Texts()))
	}
	privateCompareQueryAndExports(t, coreSession, controller, coreState)
	t.Logf("profile=%s snapshot_sha256_12=%s views=%d read_only=true", profile, privateSnapshotDigest(coreState.SnapshotID), len(views))
}

func privateCompareQueryAndExports(t *testing.T, coreSession *desktop.Session, controller *workspace.Controller, coreState desktop.State) {
	t.Helper()

	const queryText = "SELECT account, number FROM postings LIMIT 10"
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentExecuteQuery, QueryText: queryText}); err != nil {
		t.Fatalf("Go UI query execution could not start (%T); private values omitted", err)
	}
	workspaceState := waitPrivateLedgerState(t, controller, func(state workspace.State) bool {
		return !state.QueryBusy && state.QueryResponse != nil
	})
	coreQuery, err := coreSession.ExecuteQuery(t.Context(), desktop.QueryRequest{
		SnapshotID: coreState.SnapshotID,
		RequestID:  workspaceState.QueryResponse.Context.RequestID,
		Text:       queryText,
	})
	if err != nil {
		t.Fatalf("core query execution failed (%T); private values omitted", err)
	}
	if !reflect.DeepEqual(coreQuery, *workspaceState.QueryResponse) {
		t.Error("core and Go UI query execution results differ; private values omitted")
		return
	}
	if coreQuery.Error != nil || coreQuery.Execution == nil {
		t.Fatal("fixed query did not produce a typed result; private values omitted")
	}

	model := NewModel(true)
	model.workspace = controller
	model.workspaceState = workspaceState
	model.page = pageQuery
	tester := ui.NewTester(model.View, 1180, 780)
	if !tester.HasText("查询工作台") || !tester.HasText("account") || !tester.HasText("number") {
		t.Error("Go UI did not render the fixed query result column; private values omitted")
	} else {
		t.Logf("view=query_result core_workspace=match ui_result=present rendered_nodes=%d", len(tester.Texts()))
	}

	for _, format := range []string{"csv", "analysis"} {
		request := desktop.QueryExportRequest{
			SnapshotID:  coreQuery.Context.SnapshotID,
			RequestID:   coreQuery.Context.RequestID,
			Fingerprint: coreQuery.Context.Fingerprint,
			Format:      format,
		}
		coreCapture, err := coreSession.CaptureQueryExport(request)
		if err != nil {
			t.Fatalf("core query %s capture failed (%T); private bytes omitted", format, err)
		}
		if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentCaptureQuery, QueryExport: request}); err != nil {
			t.Fatalf("Go UI query %s capture could not start (%T); private bytes omitted", format, err)
		}
		workspaceState = waitPrivateLedgerState(t, controller, func(state workspace.State) bool {
			return !state.QueryCaptureBusy && state.QueryCapture != nil
		})
		workspaceCapture := workspaceState.QueryCapture
		if workspaceCapture.Format != coreCapture.Format || workspaceCapture.Context != coreCapture.Context || !reflect.DeepEqual(workspaceCapture.Content, coreCapture.Content) {
			t.Errorf("core and Go UI query %s export bytes differ; private bytes omitted", format)
			continue
		}
		t.Logf("export=query format=%s core_workspace=match bytes=%d", format, len(coreCapture.Content))
	}

	request := requestForView("journal")
	request.SnapshotID = coreState.SnapshotID
	coreResponse, err := coreSession.Read(request)
	if err != nil || coreResponse.ReportExport == nil {
		t.Fatalf("core report projection could not be prepared (%T); private values omitted", err)
	}
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: request}); err != nil {
		t.Fatalf("Go UI report projection could not be read (%T); private values omitted", err)
	}
	workspaceState = waitPrivateLedgerState(t, controller, func(state workspace.State) bool {
		return !state.Busy && state.Response != nil && state.Response.Request == coreResponse.Request && state.Response.ReportExport != nil
	})
	identity := coreResponse.ReportExport
	if !reflect.DeepEqual(identity, workspaceState.Response.ReportExport) {
		t.Fatal("core and Go UI report export identities differ; private values omitted")
	}
	reportRequest := desktop.ReportExportRequest{
		ProjectionID: identity.ProjectionID,
		SnapshotID:   identity.SnapshotID,
		Format:       "csv",
		CaptureScope: identity.CaptureScope,
	}
	coreReport, err := coreSession.CaptureReportExport(reportRequest)
	if err != nil {
		t.Fatalf("core report capture failed (%T); private bytes omitted", err)
	}
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentCaptureReport, Report: reportRequest}); err != nil {
		t.Fatalf("Go UI report capture could not start (%T); private bytes omitted", err)
	}
	workspaceState = waitPrivateLedgerState(t, controller, func(state workspace.State) bool {
		return !state.ReportCaptureBusy && state.ReportCapture != nil
	})
	workspaceReport := workspaceState.ReportCapture
	if !reflect.DeepEqual(workspaceReport.Identity, coreReport.Identity) || workspaceReport.Format != coreReport.Format || !reflect.DeepEqual(workspaceReport.Content, coreReport.Content) {
		t.Error("core and Go UI report CSV export bytes differ; private bytes omitted")
		return
	}
	t.Logf("export=journal_report format=csv core_workspace=match bytes=%d", len(coreReport.Content))

}

func waitPrivateLedgerState(t *testing.T, controller *workspace.Controller, predicate func(workspace.State) bool) workspace.State {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		state := controller.State()
		if predicate(state) {
			return state
		}
		time.Sleep(2 * time.Millisecond)
	}
	state := controller.State()
	t.Fatalf("Go UI workspace timed out (phase=%s busy=%t response_ready=%t); source details omitted", state.Phase, state.Busy, state.Response != nil)
	return state
}

func privateSnapshotDigest(snapshotID string) string {
	digest := sha256.Sum256([]byte(snapshotID))
	return hex.EncodeToString(digest[:6])
}

func privateResponseDifferenceFields(core, workspace desktop.ViewResponse) []string {
	coreValue, workspaceValue := reflect.ValueOf(core), reflect.ValueOf(workspace)
	responseType := coreValue.Type()
	different := make([]string, 0)
	for i := 0; i < responseType.NumField(); i++ {
		if !reflect.DeepEqual(coreValue.Field(i).Interface(), workspaceValue.Field(i).Interface()) {
			different = append(different, responseType.Field(i).Name)
		}
	}
	return different
}

func privateComparableResponse(response desktop.ViewResponse) desktop.ViewResponse {
	if len(response.State.ActiveExtensions) == 0 {
		response.State.ActiveExtensions = nil
	}
	return response
}

func privateDesktopStateDifferenceFields(core, workspace desktop.State) []string {
	coreValue, workspaceValue := reflect.ValueOf(core), reflect.ValueOf(workspace)
	stateType := coreValue.Type()
	different := make([]string, 0)
	for i := 0; i < stateType.NumField(); i++ {
		if !reflect.DeepEqual(coreValue.Field(i).Interface(), workspaceValue.Field(i).Interface()) {
			different = append(different, stateType.Field(i).Name)
		}
	}
	return different
}
