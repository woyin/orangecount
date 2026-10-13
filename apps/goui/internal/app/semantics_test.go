// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/preferences"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

func TestSemanticsPreviewIsSnapshotBoundAndApplyPersistsOnlyAfterOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	input := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening-Balances USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening-Balances -10 USD
`
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	profileOpenStarted := make(chan struct{}, 1)
	releaseProfileOpen := make(chan struct{})
	controller := workspace.NewController("semantics-profile-window", profileOpenGateEngine{
		started: profileOpenStarted, release: releaseProfileOpen,
	}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("semantics workspace did not close")
		}
	})
	watch := snapshot.WatchOptions{PollInterval: time.Hour}
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: path, Options: workspace.OpenOptions{SemanticsProfile: ledger.Legacy, Watch: watch}}); err != nil {
		t.Fatal(err)
	}
	opened := waitWorkspace(t, controller, func(state workspace.State) bool { return state.Phase == workspace.PhaseReady && !state.Busy })
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: desktop.Request{View: "migration"}}); err != nil {
		t.Fatal(err)
	}
	previewState := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Migration != nil && !state.Busy
	})
	if previewState.Response.Migration.InputIdentity == "" || previewState.Response.Request.SnapshotID != opened.Ledger.SnapshotID {
		t.Fatalf("migration preview is not bound to the current graph and profile snapshot: input=%q request=%q state=%q", previewState.Response.Migration.InputIdentity, previewState.Response.Request.SnapshotID, opened.Ledger.SnapshotID)
	}

	store := preferences.NewStore(filepath.Join(t.TempDir(), "support", "preferences.json"))
	m := NewModel(true)
	m.workspace = controller
	m.workspaceState = previewState
	m.page = pageSemantics
	m.pageHistoryBack = []page{pageBalance}
	m.pageHistoryForward = []page{pageJournal}
	m.queryDraft = "SELECT * FROM entries"
	m.SetPreferences(store)
	defer m.Close()
	request := m.requestForView("migration")
	if request.SnapshotID != opened.Ledger.SnapshotID || request.MigrationDetails {
		t.Fatalf("migration request is not snapshot-bound or asks for private detail: %+v", request)
	}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"语义配置", "Legacy", "V3", "预览只显示差异数量", "应用 V3"} {
		if !tester.HasText(want) {
			t.Errorf("semantics page is missing %q: %q", want, tester.Texts())
		}
	}

	m.applySemanticsProfile(ledger.V3, previewState.Response.Migration.InputIdentity)
	select {
	case <-profileOpenStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("profile reopen did not reach the gated open")
	}
	if len(m.pageHistoryBack) != 0 || len(m.pageHistoryForward) != 0 {
		t.Fatalf("accepted profile reopen retained stale navigation history: back=%v forward=%v", m.pageHistoryBack, m.pageHistoryForward)
	}
	m.workspaceState = controller.State()
	pending := ui.NewTester(m.View, 1180, 780)
	operationWhileOpening := controller.State().Operation
	if err := pending.Click("返回"); err != nil {
		t.Fatalf("click disabled Back while the profile session is opening: %v", err)
	}
	if m.page != pageSemantics || controller.State().Operation != operationWhileOpening {
		t.Fatalf("stale navigation changed pages during profile reopen: page=%d state=%+v", m.page, controller.State())
	}
	close(releaseProfileOpen)
	applied := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Phase == workspace.PhaseReady && !state.Busy && state.Ledger.SemanticsProfile == ledger.V3
	})
	if applied.Ledger.SnapshotID != previewState.Response.Migration.V3.SnapshotID || applied.Response != nil {
		t.Fatalf("profile apply did not reopen the same input as a fresh session: %+v", applied)
	}
	m.applyPendingOpenState(applied)
	m.workspaceState = applied
	m.preferenceWork.Wait()
	item, err := store.ForLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	if item.SemanticsProfile != string(ledger.V3) || m.queryDraft != "" {
		t.Fatalf("successful profile apply did not clear old query state and persist the selected profile: profile=%q query=%q", item.SemanticsProfile, m.queryDraft)
	}
	unchanged, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(unchanged) != input {
		t.Fatal("applying a semantics profile modified the ledger file")
	}
}

type profileOpenGateEngine struct {
	started chan<- struct{}
	release <-chan struct{}
}

func (e profileOpenGateEngine) Open(ctx context.Context, path string, options workspace.OpenOptions) (workspace.Session, error) {
	if options.SemanticsProfile == ledger.V3 {
		select {
		case e.started <- struct{}{}:
		default:
		}
		select {
		case <-e.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return (workspace.DesktopEngine{}).Open(ctx, path, options)
}

func TestSemanticsApplyRejectsPreviewFromDifferentWorkspaceSnapshot(t *testing.T) {
	m := NewModel(true)
	m.workspaceState = workspace.State{
		Phase: workspace.PhaseReady, ActivePath: "/tmp/current.bean",
		Ledger: desktop.State{SemanticsProfile: ledger.Legacy, SnapshotID: "current"},
		Response: &desktop.ViewResponse{
			State:     desktop.State{SnapshotID: "preview"},
			Request:   desktop.Request{View: "migration", SnapshotID: "preview"},
			Migration: &snapshot.MigrationComparison{InputIdentity: "preview"},
		},
	}
	if m.migrationResponseMatchesCurrent(m.workspaceState.Response) {
		t.Fatal("preview response from an older profile-specific snapshot was accepted")
	}
}

func TestSemanticsApplyRejectsPreviewFromOlderInvalidLedgerRevision(t *testing.T) {
	const currentPath = "/tmp/current-invalid.bean"
	m := NewModel(true)
	m.workspaceState = workspace.State{
		Phase: workspace.PhaseInvalid, ActivePath: currentPath,
		Ledger: desktop.State{Revision: 2, EntryPath: currentPath},
		Response: &desktop.ViewResponse{
			State:     desktop.State{Revision: 1, EntryPath: currentPath},
			Request:   desktop.Request{View: "migration"},
			Migration: &snapshot.MigrationComparison{InputIdentity: "older-input"},
		},
	}
	if m.migrationResponseMatchesCurrent(m.workspaceState.Response) {
		t.Fatal("migration preview from an older invalid-ledger revision was accepted")
	}
}

func TestSemanticsPageCanSwitchAwayFromInvalidLegacyEvaluation(t *testing.T) {
	fixture := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "holdings-unknowns.bean")
	fixture, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}
	controller := workspace.NewController("invalid-legacy-semantics", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("invalid semantics workspace did not close")
		}
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: fixture, Options: workspace.OpenOptions{
		SemanticsProfile: ledger.Legacy,
		Watch:            snapshot.WatchOptions{PollInterval: time.Hour},
	}}); err != nil {
		t.Fatal(err)
	}
	invalid := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Phase == workspace.PhaseInvalid && !state.Busy
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: desktop.Request{
		View:       "migration",
		SnapshotID: invalid.Ledger.SnapshotID,
	}}); err != nil {
		t.Fatal(err)
	}
	preview := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Operation > invalid.Operation && !state.Busy
	})
	if preview.Response == nil || preview.Response.Migration == nil || !preview.Response.Migration.V3.Valid || preview.Response.Migration.InputIdentity == "" {
		t.Fatalf("fixture does not provide a valid V3 comparison bound to its source: response=%+v", preview.Response)
	}

	model := NewModel(true)
	model.workspace = controller
	model.workspaceState = preview
	model.page = pageSemantics
	if !model.migrationResponseMatchesCurrent(preview.Response) {
		t.Fatal("migration preview for the current invalid ledger was rejected as stale")
	}
	tester := ui.NewTester(model.View, 1180, 780)
	if !tester.HasText("应用 V3") {
		t.Fatalf("invalid Legacy evaluation leaves no usable V3 recovery action: current_preview=%t phase=%s ledger_snapshot=%q migration_input=%q v3_valid=%t text=%q",
			model.migrationResponseMatchesCurrent(preview.Response), preview.Phase, preview.Ledger.SnapshotID,
			preview.Response.Migration.InputIdentity, preview.Response.Migration.V3.Valid, tester.Texts())
	}
	if err := tester.Click("应用 V3"); err != nil {
		t.Fatalf("click V3 recovery action: %v", err)
	}
	openedV3 := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Epoch > preview.Epoch && state.Phase == workspace.PhaseReady && !state.Busy
	})
	if openedV3.ActivePath != fixture || openedV3.Ledger.SemanticsProfile != ledger.V3 || !openedV3.Ledger.Valid {
		t.Fatalf("V3 recovery did not open the valid interpretation of the same source: %+v", openedV3)
	}
}
