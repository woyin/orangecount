// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/platform"
	"orangecount/apps/goui/internal/preferences"
	"orangecount/apps/goui/internal/querypresets"
	"orangecount/apps/goui/internal/widgets"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/desktop"
	"orangecount/internal/documents"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
)

func TestWindowOptionsUseNativeContentAndNoPage(t *testing.T) {
	options := NewModel(true).WindowOptions()
	if options.Content == nil {
		t.Fatal("window has no native Content")
	}
	if options.URL != "" {
		t.Fatalf("native Content window also configures a page URL: %q", options.URL)
	}
	if options.MinWidth != 900 || options.MinHeight != 600 {
		t.Fatalf("minimum window size = %dx%d, want 900x600", options.MinWidth, options.MinHeight)
	}
}

func TestWelcomeDoesNotRestoreLedgerAndSeparatesQAIdentity(t *testing.T) {
	qa := NewModel(true)
	tester := ui.NewTester(qa.View, 1180, 780)
	if !tester.HasText("欢迎使用 OrangeCount") || !tester.HasText("不会自动恢复最近账本") {
		t.Fatalf("QA welcome does not explain its startup isolation: %q", tester.Texts())
	}
	if qa.workspaceState.ActivePath != "" || qa.workspace != nil {
		t.Fatal("new QA window unexpectedly contains a restored ledger")
	}
}

func TestWelcomeShowsPreferenceStorageStartupError(t *testing.T) {
	m := NewModel(false)
	m.SetPreferences(nil)
	m.SetPreferenceStartupError(errors.New("permission denied"))

	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("应用偏好存储不可用：permission denied") {
		t.Fatalf("welcome page omitted the preference storage error: %q", tester.Texts())
	}

	m.locale = "en"
	tester.Frame()
	if !tester.HasText("Application preferences are unavailable: permission denied") {
		t.Fatalf("welcome page did not localize the preference storage error: %q", tester.Texts())
	}
}

func TestWelcomeOffersExplicitlyReopenableRecentLedger(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	store := preferences.NewStore(filepath.Join(t.TempDir(), "support", "preferences.json"))
	if err := store.RecordRecentLedger(path); err != nil {
		t.Fatal(err)
	}
	controller := workspace.NewController("recent-window", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("recent ledger workspace did not close")
		}
	})
	m := NewModel(true)
	m.SetPreferences(store)
	m.workspace = controller
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("最近账本") || !tester.HasText("cashflow.bean") {
		t.Fatalf("welcome did not expose the saved recent entry: %q", tester.Texts())
	}
	if state := controller.State(); state.ActivePath != "" {
		t.Fatalf("recent entry opened without an explicit selection: %+v", state)
	}
	if err := tester.Click("cashflow.bean"); err != nil {
		t.Fatal(err)
	}
	state := waitWorkspace(t, controller, func(s workspace.State) bool { return s.Phase == workspace.PhaseReady })
	if state.ActivePath != filepath.Clean(path) || state.Ledger.SemanticsProfile != ledger.Legacy {
		t.Fatalf("explicit recent open = path:%q profile:%q", state.ActivePath, state.Ledger.SemanticsProfile)
	}
}

func TestAttachmentRootsPageShowsDefaultAndInvalidBookmarks(t *testing.T) {
	m := NewModel(true)
	m.page = pageAttachmentRoots
	m.workspaceState.ActivePath = "/tmp/example.bean"
	m.workspaceState.AttachmentRootsLoaded = true
	m.workspaceState.AttachmentRoots = desktop.AttachmentRoots{DefaultRoot: "/tmp", AdditionalRoots: []string{}}
	m.attachmentBookmarks = []platform.DirectoryBookmark{{Path: "/tmp/attachments", Data: []byte{1}, Device: 2, Inode: 3}}
	m.invalidAttachmentRoots = []string{"/tmp/attachments"}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"附件目录", "默认目录（不可移除）", "/tmp", "/tmp/attachments", "当前未加入策略", "添加目录…"} {
		if !tester.HasText(want) {
			t.Errorf("attachment roots page does not show %q: %q", want, tester.Texts())
		}
	}
}

func TestMergeAttachmentRootBookmarkReplacesOnlyInvalidDuplicates(t *testing.T) {
	const attachmentPath = "/tmp/shared-attachments"
	old := platform.DirectoryBookmark{Path: attachmentPath, Data: []byte("old-bookmark"), Device: 1, Inode: 2}
	selected := platform.DirectoryBookmark{Path: attachmentPath, Data: []byte("new-bookmark"), Device: 1, Inode: 3}
	other := platform.DirectoryBookmark{Path: "/tmp/other-attachments", Data: []byte("other-bookmark"), Device: 1, Inode: 4}
	tests := []struct {
		name          string
		existing      []platform.DirectoryBookmark
		invalidPaths  []string
		defaultRoot   string
		selected      platform.DirectoryBookmark
		wantResult    attachmentRootBookmarkMergeResult
		wantPaths     []string
		wantDataAtEnd []byte
	}{
		{
			name:          "reselecting an invalid bookmark replaces its saved identity",
			existing:      []platform.DirectoryBookmark{old},
			invalidPaths:  []string{attachmentPath},
			selected:      selected,
			wantResult:    attachmentRootBookmarkReplaced,
			wantPaths:     []string{attachmentPath},
			wantDataAtEnd: selected.Data,
		},
		{
			name:          "an already valid bookmark remains a duplicate",
			existing:      []platform.DirectoryBookmark{old},
			selected:      selected,
			wantResult:    attachmentRootBookmarkDuplicate,
			wantPaths:     []string{attachmentPath},
			wantDataAtEnd: old.Data,
		},
		{
			name:          "a new directory is appended",
			existing:      []platform.DirectoryBookmark{old},
			selected:      other,
			wantResult:    attachmentRootBookmarkAdded,
			wantPaths:     []string{attachmentPath, other.Path},
			wantDataAtEnd: other.Data,
		},
		{
			name:          "the default root cannot be added as an extra root",
			defaultRoot:   attachmentPath,
			selected:      selected,
			wantResult:    attachmentRootBookmarkDefault,
			wantPaths:     nil,
			wantDataAtEnd: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, result := mergeAttachmentRootBookmark(tt.existing, tt.invalidPaths, tt.defaultRoot, tt.selected)
			if result != tt.wantResult {
				t.Fatalf("merge result = %d, want %d", result, tt.wantResult)
			}
			if len(updated) != len(tt.wantPaths) {
				t.Fatalf("updated roots = %+v, want paths %v", updated, tt.wantPaths)
			}
			for i, wantPath := range tt.wantPaths {
				if updated[i].Path != wantPath {
					t.Fatalf("updated root %d path = %q, want %q", i, updated[i].Path, wantPath)
				}
			}
			if len(updated) > 0 && !bytes.Equal(updated[len(updated)-1].Data, tt.wantDataAtEnd) {
				t.Fatalf("last bookmark identity = %q, want %q", updated[len(updated)-1].Data, tt.wantDataAtEnd)
			}
			if len(tt.existing) > 0 && !bytes.Equal(tt.existing[0].Data, old.Data) {
				t.Fatal("merging modified the caller's existing bookmark")
			}
		})
	}
}

func TestAttachmentsPageShowsStatusWithoutExposingResolvedPath(t *testing.T) {
	ledgerPath, attachmentPath := writeDocumentLedger(t, false)
	session, err := desktop.OpenWithOptions(ledgerPath, desktop.OpenOptions{SemanticsProfile: ledger.Legacy})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "journal", JournalMode: "all", DirectiveKind: "document", Sort: "descending", Limit: 100, Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Directives == nil || len(response.Directives.Records) != 1 || len(response.Directives.Records[0].Attachments) != 1 {
		t.Fatalf("document declarations = %+v", response.Directives)
	}
	attachment := response.Directives.Records[0].Attachments[0]
	if attachment.Status != documents.Available {
		t.Fatalf("attachment status = %q, want available", attachment.Status)
	}
	m := NewModel(true)
	m.page = pageAttachments
	m.workspaceState = workspace.State{
		Phase: workspace.PhaseReady, ActivePath: ledgerPath, Ledger: response.State,
		Response: &response, AttachmentRootsLoaded: true,
		AttachmentRoots: desktop.AttachmentRoots{PolicyRevision: attachment.PolicyRevision, DefaultRoot: filepath.Dir(ledgerPath)},
	}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"附件", "receipt.pdf", "可用", "打开", "在 Finder 中显示"} {
		if !tester.HasText(want) {
			t.Errorf("attachment view does not show %q: %q", want, tester.Texts())
		}
	}
	if tester.HasText(attachmentPath) {
		t.Fatal("attachment view exposed the resolved filesystem path")
	}
}

func TestAttachmentOpenUsesSessionRevalidatedAction(t *testing.T) {
	ledgerPath, attachmentPath := writeDocumentLedger(t, false)
	controller := workspace.NewController("attachment-action-test", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("attachment action workspace did not close")
		}
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: ledgerPath}); err != nil {
		t.Fatal(err)
	}
	state := waitWorkspace(t, controller, func(s workspace.State) bool { return s.Phase == workspace.PhaseReady })
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: desktop.Request{
		View: "journal", JournalMode: "all", DirectiveKind: "document", Sort: "descending", Limit: 100, Locale: "zh-CN",
	}}); err != nil {
		t.Fatal(err)
	}
	state = waitWorkspace(t, controller, func(s workspace.State) bool { return !s.Busy && s.Response != nil && s.Response.Directives != nil })
	attachment := state.Response.Directives.Records[0].Attachments[0]
	opened := ""
	m := NewModel(true)
	m.workspace = controller
	m.workspaceState = state
	m.openPath = func(path string) error { opened = path; return nil }
	m.requestAttachmentAction(attachment, false)
	requestState := waitWorkspace(t, controller, func(s workspace.State) bool {
		return !s.Busy && s.Response != nil && s.Response.DocumentAction != nil
	})
	m.finishAttachmentAction(requestState)
	if opened != attachmentPath {
		t.Fatalf("opened path = %q, want session-authorized %q", opened, attachmentPath)
	}
	if m.pendingAttachmentAction != nil || m.attachmentActionNotice != "已交给系统默认应用打开。" {
		t.Fatalf("attachment action state = pending:%v notice:%q", m.pendingAttachmentAction, m.attachmentActionNotice)
	}

	revealed := ""
	m.revealPath = func(path string) { revealed = path }
	m.requestAttachmentAction(attachment, true)
	revealState := waitWorkspace(t, controller, func(s workspace.State) bool {
		return !s.Busy && s.Response != nil && s.Response.DocumentAction != nil
	})
	m.finishAttachmentAction(revealState)
	if revealed != attachmentPath {
		t.Fatalf("revealed path = %q, want session-authorized %q", revealed, attachmentPath)
	}

	opened = ""
	m.requestAttachmentAction(attachment, false)
	staleActionState := waitWorkspace(t, controller, func(s workspace.State) bool {
		return !s.Busy && s.Response != nil && s.Response.DocumentAction != nil
	})
	if _, err := controller.ConfigureAttachmentRoots(desktop.AttachmentRootsRequest{
		PolicyRevision: staleActionState.AttachmentRoots.PolicyRevision, AdditionalRoots: []string{},
	}); err != nil {
		t.Fatal(err)
	}
	m.finishAttachmentAction(staleActionState)
	if opened != "" {
		t.Fatalf("stale action opened %q after attachment policy changed", opened)
	}
}

func TestDirectivePageRendersTypedFieldsMetadataAndGeneratedLineage(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "directives", "all.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "journal", JournalMode: "all", Sort: "ascending", Limit: 100, Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page = pageDirectives
	m.directiveMode = "all"
	m.directiveDescending = false
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: response.State, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"全部指令", "所有指令", "配置", "交易", "由 方言行 生成", "关联源指令", "#explicit", "^public-link", "NULL", "TRUE", "显示来源"} {
		if !tester.HasText(want) {
			t.Errorf("directive page does not show %q: %q", want, tester.Texts())
		}
	}
	expectedAllKinds := map[string]bool{
		"transaction": false, "open": false, "close": false, "commodity": false,
		"balance": false, "pad": false, "event": false, "query": false,
		"price": false, "document": false, "note": false, "custom": false,
	}
	for _, record := range response.Directives.Records {
		if _, expected := expectedAllKinds[record.Kind]; expected {
			expectedAllKinds[record.Kind] = true
		}
		label := directiveTypeLabel(record.Kind)
		if !tester.HasText(label) && !tester.HasText("由 "+label+" 生成") {
			t.Errorf("directive page did not render core record kind %q: %q", record.Kind, tester.Texts())
		}
	}
	for kind, found := range expectedAllKinds {
		if !found {
			t.Errorf("public all-directives fixture has no dated %q record", kind)
		}
	}

	config, err := session.Read(desktop.Request{View: "journal", JournalMode: "config", DateFrom: "2099-01-01", Sort: "ascending", Limit: 100, Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	m.directiveMode = "config"
	m.workspaceState.Response = &config
	configTester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"配置记录按源码顺序读取", "不应用日期", "选项", "All directive types"} {
		if !configTester.HasText(want) {
			t.Errorf("configuration page does not show %q: %q", want, configTester.Texts())
		}
	}
	expectedConfigKinds := map[string]bool{
		"option": false, "include": false, "pushtag": false, "poptag": false, "pushmeta": false, "popmeta": false,
	}
	for _, record := range config.Directives.Records {
		if _, expected := expectedConfigKinds[record.Kind]; expected {
			expectedConfigKinds[record.Kind] = true
		}
		if !configTester.HasText(directiveTypeLabel(record.Kind)) {
			t.Errorf("configuration page did not render core record kind %q: %q", record.Kind, configTester.Texts())
		}
	}
	for kind, found := range expectedConfigKinds {
		if !found {
			t.Errorf("public all-directives fixture has no configuration %q record", kind)
		}
	}
	if configTester.HasText("开始日期 YYYY-MM-DD") {
		t.Fatal("configuration directives displayed date controls that do not apply to their source order")
	}
}

func TestDirectivePageExplainsMissingSnapshot(t *testing.T) {
	m := NewModel(true)
	m.page = pageDirectives
	m.directiveMode = "all"
	m.workspaceState = workspace.State{
		Phase:    workspace.PhaseInvalid,
		Response: &desktop.ViewResponse{State: desktop.State{Valid: false, HasSnapshot: false}},
	}
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("账本没有可用快照") || !tester.HasText("修复账本后可重新读取") {
		t.Fatalf("directive page did not explain the missing snapshot: %q", tester.Texts())
	}
}

func TestDirectiveRequestsKeepConfigSourceOrderAndBoundSearch(t *testing.T) {
	m := NewModel(true)
	m.directiveMode = "all"
	m.directiveKind = "note"
	m.directiveSearch = "duplicate"
	m.directiveOffset = 200
	request := m.requestForView("directives")
	if request.View != "journal" || request.JournalMode != "all" || request.DirectiveKind != "note" || request.Text != "duplicate" || request.Offset != 200 || request.Limit != 100 || request.Sort != "descending" {
		t.Fatalf("all-directive request = %+v", request)
	}
	m.directiveMode = "config"
	m.directiveDescending = true
	request = m.requestForView("directives")
	if request.JournalMode != "config" || request.Sort != "ascending" || request.DateFrom != "" || request.DateTo != "" {
		t.Fatalf("configuration request acquired date scope or lost source order: %+v", request)
	}
	m.directiveSearchDraft = strings.Repeat("x", 65537)
	m.applyDirectiveSearch()
	if m.directiveError == "" || m.workspace != nil || m.directiveSearch != "duplicate" {
		t.Fatalf("oversized directive search was not rejected locally: error:%q workspace:%v", m.directiveError, m.workspace)
	}
	response := &desktop.ViewResponse{Request: desktop.Request{
		View: "journal", JournalMode: "config", DirectiveKind: "note", Text: "duplicate", Sort: "ascending", Offset: 200, Limit: 100,
	}, Directives: &desktop.DirectivesPayload{Mode: "config"}}
	if !m.directiveResponseMatchesFilters(response) {
		t.Fatal("matching response did not preserve the applied filters")
	}
	m.directiveSearchDraft = "unapplied draft"
	if !m.directiveResponseMatchesFilters(response) {
		t.Fatal("editing the search draft changed the currently applied result identity")
	}
}

func TestDirectiveValueFormatterKeepsNestedTypesAndExactAmounts(t *testing.T) {
	value := desktop.DirectiveValue{Type: "list", Value: []desktop.DirectiveValue{
		{Type: "decimal", Value: query.PresentedDecimal{Display: "1.25000000000000000001", Exact: "1.25000000000000000001"}},
		{Type: "null"},
		{Type: "bool", Value: true},
		{Type: "map", Value: []desktop.DirectiveMetadata{{
			Key: "cost", Value: desktop.DirectiveValue{Type: "amount", Value: query.PresentedAmount{
				Number: query.PresentedDecimal{Exact: "10.00000000000000000003"}, Currency: "USD",
			}},
		}}},
	}}
	got := formatDirectiveValue(value)
	want := "[1.25000000000000000001, NULL, TRUE, {cost: 10.00000000000000000003 USD}]"
	if got != want {
		t.Fatalf("formatted directive value = %q, want %q", got, want)
	}
}

func writeDocumentLedger(t *testing.T, outsideRoot bool) (string, string) {
	t.Helper()
	root := t.TempDir()
	attachmentRoot := root
	if outsideRoot {
		attachmentRoot = t.TempDir()
	}
	attachmentPath := filepath.Join(attachmentRoot, "receipt.pdf")
	if err := os.WriteFile(attachmentPath, []byte("synthetic receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(root, "ledger.bean")
	contents := "option \"operating_currency\" \"USD\"\n2024-01-01 open Assets:Cash USD\n2024-01-01 document Assets:Cash \"" + attachmentPath + "\"\n"
	if err := os.WriteFile(ledgerPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	canonicalAttachmentPath, err := filepath.EvalSymlinks(attachmentPath)
	if err != nil {
		t.Fatal(err)
	}
	return ledgerPath, canonicalAttachmentPath
}

func TestVirtualTableRendersVisibleRowsFromLargeDataset(t *testing.T) {
	m := NewModel(true)
	tester := ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("虚拟表格 · 50,000 行"); err != nil {
		t.Fatal(err)
	}
	if !tester.HasText("Entry 000001") {
		t.Fatalf("table did not render its first visible row: %q", tester.Texts())
	}
	if tester.HasText("Entry 050000") {
		t.Fatal("offscreen row was rendered in the initial viewport")
	}
	visible := 0
	for _, text := range tester.Texts() {
		if strings.HasPrefix(text, "Entry ") {
			visible++
		}
	}
	if visible < 1 || visible > 100 {
		t.Fatalf("rendered %d synthetic table rows in a 780 DIP viewport", visible)
	}
}

func TestChartValidationPageKeepsExactValuesAndMissingStatus(t *testing.T) {
	m := NewModel(true)
	tester := ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("图表绘制试验"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"收入与支出", "月度收支分组柱", "1000.00 USD", "0.00 USD", "不可用"} {
		if !tester.HasText(want) {
			t.Errorf("chart page does not expose %q: %q", want, tester.Texts())
		}
	}
	if rect, ok := tester.Find("1000.00 USD"); !ok || rect.Y+rect.H > 780 {
		t.Errorf("first exact amount is outside the chart page viewport: %+v, found=%v", rect, ok)
	}
	m.locale = "en"
	tester.Frame()
	for _, want := range []string{"Net cash flow line chart", "Grouped bar chart", "Period", "Unavailable"} {
		if !tester.HasText(want) {
			t.Errorf("English chart page does not expose %q: %q", want, tester.Texts())
		}
	}
	if tester.HasText("期间") || tester.HasText("分组柱状图") {
		t.Errorf("English chart page still exposes Chinese chart labels: %q", tester.Texts())
	}
}

func TestNestedModuleReadsPublicCashflowFixtureThroughDesktopSession(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "dashboard", Currency: "USD", Valuation: "at-cost"})
	if err != nil {
		t.Fatal(err)
	}
	if !response.Summary.Valid || response.Summary.TransactionsCount != 2 {
		t.Fatalf("cashflow summary = valid:%v transactions:%d diagnostics:%d", response.Summary.Valid, response.Summary.TransactionsCount, response.Summary.ErrorCount)
	}
}

func TestDashboardRendersCashflowFromCoreProjection(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "dashboard", Valuation: "at-cost", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}

	m := NewModel(true)
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"账本已读取", "财务概览", "净资产变化", "月度收支变化", "1000 USD", "300 USD"} {
		if !tester.HasText(want) {
			t.Errorf("core-backed dashboard does not render %q: %q", want, tester.Texts())
		}
	}
	dashboard := NewModel(true)
	dashboard.page = pageDashboard
	dashboard.workspaceState = m.workspaceState
	tester = ui.NewTester(dashboard.View, 1180, 780)
	if tester.HasText("账本已读取") || !tester.HasText("净资产变化") || !tester.HasText("1000 USD") {
		t.Fatalf("dedicated dashboard page did not show the core projection independently: %q", tester.Texts())
	}
}

func TestDashboardKeepsNativeCurrenciesSeparate(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "v3-parity", "options.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "dashboard", Valuation: "at-cost", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Trends.CurrencySeries) != 2 {
		t.Fatalf("native dashboard currency series = %d, want USD and EUR", len(response.Trends.CurrencySeries))
	}
	m := NewModel(true)
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"USD · at-cost", "EUR · at-cost", "1 USD", "2 EUR"} {
		if !tester.HasText(want) {
			t.Errorf("dashboard did not keep native amount %q visible: %q", want, tester.Texts())
		}
	}
}

func TestDashboardExplainsUnavailableChartAndCurrencyStates(t *testing.T) {
	trends := desktop.HistoricalTrendsPayload{
		CurrencySeries: []report.CurrencyHistorySeries{{
			Currency: "USD", Valuation: "market-value",
			NetWorthAvailability: "unavailable-currency", CashFlowAvailability: "partial",
		}},
	}
	m := NewModel(true)
	tester := ui.NewTester(func(c *ui.Context) { m.dashboardPage(c, trends) }, 900, 600)
	if !tester.HasText("缺少本地报价，估值不完整") || !tester.HasText("部分估值缺少本地报价") || tester.HasText("unavailable-currency") || tester.HasText("partial") {
		t.Fatalf("dashboard exposed internal availability codes: %q", tester.Texts())
	}

	noCurrency := NewModel(true)
	selectCurrencyTester := ui.NewTester(func(c *ui.Context) {
		noCurrency.dashboardPage(c, desktop.HistoricalTrendsPayload{NetWorthAvailability: "select-currency"})
	}, 900, 600)
	if !selectCurrencyTester.HasText("请先选择估值币种") || selectCurrencyTester.HasText("账本尚无可绘制的月度收支记录") {
		t.Fatalf("dashboard conflated missing currency with empty history: %q", selectCurrencyTester.Texts())
	}
}

func TestDashboardUnavailableNetWorthExactStatusUsesCurrentLocale(t *testing.T) {
	trends := desktop.HistoricalTrendsPayload{CurrencySeries: []report.CurrencyHistorySeries{{
		Currency: "USD", Valuation: "market-value", NetWorthAvailability: "partial",
		NetWorthPoints: []report.HistoryPoint{{Month: "2026-01"}},
	}}}
	m := NewModel(true)
	tester := ui.NewTester(func(c *ui.Context) { m.dashboardPage(c, trends) }, 900, 700)
	if !tester.HasText("不可用") || tester.HasText("Unavailable") {
		t.Fatalf("Chinese dashboard did not localize the missing net-worth point: %q", tester.Texts())
	}
	m.locale = "en"
	tester.Frame()
	if !tester.HasText("Unavailable") || tester.HasText("不可用") {
		t.Fatalf("English dashboard did not localize the missing net-worth point: %q", tester.Texts())
	}
}

func TestDashboardShowsMissingQuoteAndOmitsQuietMonthFlows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dashboard.bean")
	data := `option "operating_currency" "USD"
2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Foreign EUR
2000-01-01 open Income:Salary USD
2000-01-01 open Equity:Opening USD, EUR
2000-01-02 * "salary"
  Assets:Cash 100 USD
  Income:Salary -100 USD
2000-01-03 * "foreign balance"
  Assets:Foreign 2 EUR
  Equity:Opening -2 EUR
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "dashboard", Currency: "USD", Valuation: "market-value", DateTo: "2000-03-31", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Trends == nil || len(response.Trends.CurrencySeries) != 1 {
		t.Fatalf("missing dashboard currency series: %+v", response.Trends)
	}
	series := response.Trends.CurrencySeries[0]
	if series.NetWorthAvailability != "unavailable-currency" || len(series.NetWorthPoints) != 3 || series.NetWorthPoints[0].Value != nil {
		t.Fatalf("missing local quote was not retained across the requested range: %+v", series)
	}
	if series.CashFlowAvailability != "priced" || len(series.MonthlyCashFlows) != 1 || series.MonthlyCashFlows[0].Month != "2000-01" {
		t.Fatalf("quiet February/March invented cash flows: %+v", series.MonthlyCashFlows)
	}
	m := NewModel(true)
	tester := ui.NewTester(func(c *ui.Context) { m.dashboardPage(c, *response.Trends) }, 1180, 780)
	if !tester.HasText("缺少本地报价，估值不完整") || tester.HasText("unavailable-currency") {
		t.Fatalf("dashboard did not explain the missing quote: %q", tester.Texts())
	}
}

func TestDashboardChartTooltipShowsMissingReasonAndUsedQuoteSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dashboard-quotes.bean")
	data := `option "operating_currency" "USD"
2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Foreign EUR
2000-01-01 open Income:Salary USD
2000-01-01 open Equity:Opening USD, EUR
2000-01-02 * "salary"
  Assets:Cash 100 USD
  Income:Salary -100 USD
2000-01-03 * "foreign balance"
  Assets:Foreign 2 EUR
  Equity:Opening -2 EUR
2000-02-15 price EUR 2 USD
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "dashboard", Currency: "USD", Valuation: "market-value", DateTo: "2000-03-31", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Trends == nil || len(response.Trends.CurrencySeries) != 1 {
		t.Fatalf("dashboard trends = %+v", response.Trends)
	}
	points := response.Trends.CurrencySeries[0].NetWorthPoints
	if len(points) != 3 || len(points[0].Valuation.Missing) != 1 || len(points[1].Valuation.Quotes) != 1 {
		t.Fatalf("expected an unpriced January point and the February quote: %+v", points)
	}
	quote := points[1].Valuation.Quotes[0]
	wantSourcePath, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Trends.SourceFiles[quote.Source.File]; got != wantSourcePath {
		t.Fatalf("quote source file = %q, want canonical path %q", got, wantSourcePath)
	}

	model := NewModel(true)
	tester := ui.NewTester(func(c *ui.Context) { model.dashboardPage(c, *response.Trends) }, 1180, 780)
	chart, ok := tester.Find("净资产折线图")
	if !ok {
		t.Fatalf("dashboard line chart is missing: %q", tester.Texts())
	}
	tester.Move(chart.X+chart.W/2, chart.Y+chart.H/2)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	quoteLocation := fmt.Sprintf("%s:%d:%d", filepath.Base(path), quote.Source.StartLine, quote.Source.StartColumn)
	for _, want := range []string{"2000-02-15", "EUR/USD 2", quoteLocation} {
		if !tester.HasText(want) {
			t.Errorf("available valuation tooltip omitted %q: %q", want, tester.Texts())
		}
	}
	tester.Move(chart.X+12, chart.Y+chart.H/2)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	if !tester.HasText("EUR：缺少本地报价") {
		t.Fatalf("unavailable valuation tooltip omitted the missing commodity and reason: %q", tester.Texts())
	}
}

func TestAccountPageExplainsMissingValuationInUserLanguage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "account-valuation.bean")
	data := `option "operating_currency" "USD"
2000-01-01 open Assets:Foreign EUR
2000-01-01 open Assets:Quoted STK
2000-01-01 open Equity:Opening EUR
2000-01-01 open Equity:Stock STK
2000-01-02 * "foreign balance"
  Assets:Foreign 2 EUR
  Equity:Opening -2 EUR
2000-01-03 * "quoted balance"
  Assets:Quoted 3 STK
  Equity:Stock -3 STK
2000-01-04 price STK 10 USD
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	request := desktop.Request{
		View: "account", Account: "Assets:Foreign", Scope: "direct", Sort: "descending", Limit: 100,
		SnapshotID: state.SnapshotID, Window: "all", Currency: "USD", Valuation: "market-value", Locale: "zh-CN",
	}
	response, err := session.Read(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Account == nil || response.Account.AccountDetailProjection == nil || response.Account.AccountDetailProjection.DirectValuation == nil {
		t.Fatalf("missing account valuation projection: %+v", response.Account)
	}
	m := NewModel(true)
	m.page, m.accountName, m.accountScope = pageAccount, "Assets:Foreign", "direct"
	m.filterCurrency, m.filterValuation = "USD", "market-value"
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: response.State, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"估值不可用", "EUR：缺少本地报价"} {
		if !tester.HasText(want) {
			t.Errorf("account page does not explain missing valuation %q: %q", want, tester.Texts())
		}
	}
	if tester.HasText("unavailable") || tester.HasText("missing-local-quote") {
		t.Fatalf("account page exposed internal valuation codes: %q", tester.Texts())
	}

	request.Account = "Assets:Quoted"
	response, err = session.Read(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Account == nil || response.Account.DirectValuation == nil || len(response.Account.DirectValuation.Quotes) != 1 {
		t.Fatalf("account quote source missing from projection: %+v", response.Account)
	}
	m = NewModel(true)
	m.page, m.accountName, m.accountScope = pageAccount, "Assets:Quoted", "direct"
	m.filterCurrency, m.filterValuation = "USD", "market-value"
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: response.State, Response: &response}
	var revealed string
	m.revealPath = func(value string) { revealed = value }
	tester = ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("2000-01-04 10 USD") || !tester.HasText("显示估值报价来源") {
		t.Fatalf("account page omitted applied local quote/source: %q", tester.Texts())
	}
	if err := tester.Click("显示估值报价来源"); err != nil {
		t.Fatalf("reveal account valuation quote: %v", err)
	}
	expectedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	if revealed != expectedPath {
		t.Fatalf("account quote Finder path = %q, want %q", revealed, expectedPath)
	}
}

func TestAccountPageKeepsDirectoryAndQuietPeriodStatesHonest(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "account-detail.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	response, err := session.Read(desktop.Request{
		View: "account", Account: "Assets", Scope: "subtree", Sort: "descending", Limit: 100,
		SnapshotID: state.SnapshotID, DateFrom: "2000-02-01", DateTo: "2000-02-29", Window: "all", Valuation: "at-cost", Locale: "zh-CN",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Account == nil || response.Account.Explicit || response.Journal == nil || len(response.Journal.Transactions) != 0 {
		t.Fatalf("directory/quiet account projection = account:%+v journal:%+v", response.Account, response.Journal)
	}
	m := NewModel(true)
	m.page, m.accountName, m.accountScope = pageAccount, "Assets", "subtree"
	m.filterDateFrom, m.filterDateTo = "2000-02-01", "2000-02-29"
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: response.State, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"Assets", "目录账户 · 无直接开户资料", "当前筛选没有账户交易。"} {
		if !tester.HasText(want) {
			t.Errorf("directory/quiet account page omits %q: %q", want, tester.Texts())
		}
	}
	if tester.HasText("明确开户") || tester.HasText("显示 1–0") {
		t.Fatalf("directory account or quiet period was misrepresented: %q", tester.Texts())
	}
}

func TestValuationLabelsHideUnknownInternalCodes(t *testing.T) {
	if got := valuationCompletenessText("partial"); got != "部分估值" {
		t.Errorf("partial valuation label = %q", got)
	}
	if got := valuationReasonText("missing-local-quote"); got != "缺少本地报价" {
		t.Errorf("missing quote label = %q", got)
	}
	if got := valuationReasonText("future-internal-code"); got != "估值资料不完整" {
		t.Errorf("unknown valuation reason leaked internal code: %q", got)
	}
}

func TestStatementPageRendersExactAccountTreeFromCore(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "balance_sheet", Valuation: "at-cost", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page = pageBalance
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"资产负债表", "Assets", "Bank", "700"} {
		if !tester.HasText(want) {
			t.Errorf("statement page does not render %q: %q", want, tester.Texts())
		}
	}
	if m.reportTree.List.Key == nil || m.reportTree.List.Label == nil || m.reportTree.Rows() == 0 {
		t.Fatal("statement outline lacks stable identities, accessible labels, or visible account rows")
	}
	for row := 0; row < m.reportTree.Rows(); row++ {
		name := m.reportTree.Item(row)
		if key, label := m.reportTree.List.Key(row), m.reportTree.List.Label(row); key != name || label != name {
			t.Fatalf("statement row %d identity/label = %v/%q, want full account name %q", row, key, label, name)
		}
	}
	if m.reportTree.Open.Len() == 0 {
		t.Fatal("statement tree should start expanded to preserve the existing full report view")
	}
	if err := tester.Click("全部折叠"); err != nil || m.reportTree.Open.Len() != 0 {
		t.Fatalf("collapse all did not clear open accounts: err=%v open=%d", err, m.reportTree.Open.Len())
	}
	if err := tester.Click("全部展开"); err != nil || m.reportTree.Open.Len() == 0 {
		t.Fatalf("expand all did not restore open accounts: err=%v open=%d", err, m.reportTree.Open.Len())
	}
}

func TestStatementPageExplainsNativeMissingQuoteAndEmptyStates(t *testing.T) {
	optionsPath := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "v3-parity", "options.bean")
	options, err := desktop.OpenWithOptions(optionsPath, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer options.Close()

	converted, err := options.Read(desktop.Request{View: "balance_sheet", Currency: "USD", Valuation: "at-cost", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if converted.Tree.Availability != "unavailable-currency" || converted.Tree.DateTo != "2000-01-03" {
		t.Fatalf("USD conversion status/date = %q/%q", converted.Tree.Availability, converted.Tree.DateTo)
	}
	m := NewModel(true)
	m.page = pageBalance
	m.filterCurrency = "USD"
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: optionsPath, Response: &converted}
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("缺少本地报价，估值不完整") || !tester.HasText("估值币种 USD") || tester.HasText("unavailable-currency") {
		t.Fatalf("statement page did not explain unavailable conversion: %q", tester.Texts())
	}
	english := NewModel(true)
	english.page = pageBalance
	english.locale = "en"
	english.filterCurrency = "USD"
	english.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: optionsPath, Response: &converted}
	englishTester := ui.NewTester(english.View, 1180, 780)
	if !englishTester.HasText("A local quote is missing; valuation is incomplete") {
		t.Fatalf("English statement status was not translated: %q", englishTester.Texts())
	}

	native, err := options.Read(desktop.Request{View: "balance_sheet", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	m = NewModel(true)
	m.page = pageBalance
	m.filterCurrency = ""
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: optionsPath, Response: &native}
	tester = ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("各原币种分别显示") || !tester.HasText("1 USD") || !tester.HasText("2 EUR") {
		t.Fatalf("native multi-currency balances were not explained and kept separate: %q", tester.Texts())
	}

	emptyPath := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "empty.bean")
	emptySession, err := desktop.OpenWithOptions(emptyPath, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer emptySession.Close()
	empty, err := emptySession.Read(desktop.Request{View: "income_statement", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	m = NewModel(true)
	m.page = pageIncome
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: emptyPath, Response: &empty}
	tester = ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("当前期间没有活动") || tester.HasText("empty") {
		t.Fatalf("empty statement state was not explained: %q", tester.Texts())
	}
}

func TestStatementPageShowsOwnSubtreeAndConvertedBalancesForBothProfiles(t *testing.T) {
	valuationPath := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "valuation.bean")
	marketPath := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "holdings-include.bean")
	for _, profile := range []ledger.SemanticsProfile{ledger.Legacy, ledger.V3} {
		t.Run(string(profile), func(t *testing.T) {
			session, err := desktop.OpenWithOptions(valuationPath, desktop.OpenOptions{SemanticsProfile: profile})
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()

			native, err := session.Read(desktop.Request{View: "balance_sheet", DateTo: "2000-01-03", Locale: "zh-CN"})
			if err != nil {
				t.Fatal(err)
			}
			model := NewModel(true)
			model.page = pageBalance
			model.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: valuationPath, Response: &native}
			tester := ui.NewTester(model.View, 1180, 780)
			for _, want := range []string{"直属余额", "汇总余额", "20 EUR"} {
				if !tester.HasText(want) {
					t.Errorf("native statement omitted %q for profile %s: %q", want, profile, tester.Texts())
				}
			}
			// Assets:Cash contributes 20 EUR to the fixed category total and has
			// the same amount in both the direct and subtree account columns.
			eurCells := 0
			for _, text := range tester.Texts() {
				if text == "20 EUR" {
					eurCells++
				}
			}
			if eurCells < 3 {
				t.Errorf("statement rendered %d matching EUR cells; want category, direct, and subtree values: %q", eurCells, tester.Texts())
			}

			converted, err := session.Read(desktop.Request{View: "balance_sheet", Currency: "USD", Valuation: "at-cost", DateTo: "2000-01-03", Locale: "zh-CN"})
			if err != nil {
				t.Fatal(err)
			}
			model.workspaceState.Response = &converted
			tester.Frame()
			if !tester.HasText("折算总计：140 USD") {
				t.Errorf("dated local EUR/USD quote was not shown as an exact converted total for profile %s: %q", profile, tester.Texts())
			}
			model.locale = "en"
			tester.Frame()
			for _, want := range []string{"Own balance", "Subtree total", "Converted total: 140 USD"} {
				if !tester.HasText(want) {
					t.Errorf("English statement omitted %q for profile %s: %q", want, profile, tester.Texts())
				}
			}

			partial, err := session.Read(desktop.Request{View: "balance_sheet", Currency: "USD", Valuation: "at-cost", DateTo: "2000-01-02", Locale: "zh-CN"})
			if err != nil {
				t.Fatal(err)
			}
			model.workspaceState.Response = &partial
			model.locale = "zh-CN"
			tester.Frame()
			for _, want := range []string{"已折算部分：100 USD", "缺报价：EUR"} {
				if !tester.HasText(want) {
					t.Errorf("partial valuation omitted %q for profile %s: %q", want, profile, tester.Texts())
				}
			}
			model.locale = "en"
			tester.Frame()
			for _, want := range []string{"Valued portion: 100 USD", "Missing quotes: EUR"} {
				if !tester.HasText(want) {
					t.Errorf("English partial valuation omitted %q for profile %s: %q", want, profile, tester.Texts())
				}
			}

			marketSession, err := desktop.OpenWithOptions(marketPath, desktop.OpenOptions{SemanticsProfile: profile})
			if err != nil {
				t.Fatal(err)
			}
			defer marketSession.Close()
			for _, test := range []struct {
				valuation string
				amount    string
				total     string
			}{
				{valuation: "at-cost", amount: "20 USD", total: "折算总计：20 USD"},
				{valuation: "market-value", amount: "2 STK", total: "折算总计：30 USD"},
			} {
				response, err := marketSession.Read(desktop.Request{View: "balance_sheet", Currency: "USD", Valuation: test.valuation, DateTo: "2000-01-03", Locale: "zh-CN"})
				if err != nil {
					t.Fatal(err)
				}
				model = NewModel(true)
				model.page = pageBalance
				model.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: marketPath, Response: &response}
				tester = ui.NewTester(model.View, 1180, 780)
				if !tester.HasText(test.amount) || !tester.HasText(test.total) {
					t.Errorf("%s statement did not show source amount %q and converted total %q for profile %s: %q", test.valuation, test.amount, test.total, profile, tester.Texts())
				}
			}
		})
	}
}

func TestStatementSearchFiltersRowsWithoutChangingCoreTotals(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "balance_sheet", Valuation: "at-cost", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page = pageBalance
	m.statementSearchDraft = "Bank"
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	allRows := m.reportTree.Rows()
	allTree := widgets.NewAccountTree(response.Tree.RootNodes, "")
	allRoots := allTree.RootKeys()
	allAssets, hasAssets := allTree.Node("Assets")
	if len(allRoots) == 0 || !hasAssets || allAssets.SubtreeAmounts["USD"].Exact == "" {
		t.Fatal("fixture does not contain a report total to preserve")
	}
	assetsTotal := allAssets.SubtreeAmounts["USD"].Exact
	if err := tester.Click("筛选账户"); err != nil {
		t.Fatalf("apply account search: %v; UI=%q", err, tester.Texts())
	}
	if !tester.HasText("Assets") || !tester.HasText("Bank") || tester.HasText("Liabilities") {
		t.Fatalf("account search did not keep matches and their ancestors only: %q", tester.Texts())
	}
	if m.reportTree.Rows() != 2 {
		t.Fatalf("search-visible tree rows = %d, want Assets + Bank", m.reportTree.Rows())
	}
	filteredTree := widgets.NewAccountTree(response.Tree.RootNodes, m.statementSearch)
	filteredAssets, ok := filteredTree.Node("Assets")
	if !ok || filteredAssets.SubtreeAmounts["USD"].Exact != assetsTotal {
		var got string
		if ok {
			got = filteredAssets.SubtreeAmounts["USD"].Exact
		}
		t.Fatalf("search changed the core-provided parent total: got %q want %q", got, assetsTotal)
	}
	m.statementSearchDraft = "no matching account"
	tester = ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("筛选账户"); err != nil {
		t.Fatalf("apply empty account search: %v", err)
	}
	if m.reportTree.Rows() != 0 || !tester.HasText("分类汇总") || !tester.HasText(widgets.ExactAmountText(allAssets.SubtreeAmounts["USD"], "USD")) {
		t.Fatalf("no-match search hid the core-provided summary: rows=%d UI=%q", m.reportTree.Rows(), tester.Texts())
	}
	if err := tester.Click("清除搜索"); err != nil {
		t.Fatalf("clear account search: %v", err)
	}
	if m.statementSearch != "" || m.reportTree.Rows() != allRows {
		t.Fatalf("clear search did not restore all stable account rows: search=%q rows=%d want=%d", m.statementSearch, m.reportTree.Rows(), allRows)
	}
}

func TestStatementSearchTemporarilyExpandsMatchesAndRestoresUserCollapse(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "account-detail.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "balance_sheet", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page = pageBalance
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("全部折叠"); err != nil || m.reportTree.Open.Len() != 0 {
		t.Fatalf("collapse account tree before search: err=%v open=%v", err, slices.Collect(m.reportTree.Open.All()))
	}

	m.statementSearchDraft = "Assets:Test:Child"
	tester = ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("筛选账户"); err != nil {
		t.Fatalf("apply deep account search: %v", err)
	}
	if m.reportTree.Rows() != 3 || !m.reportTree.Open.Has("Assets") || !m.reportTree.Open.Has("Assets:Test") {
		t.Fatalf("search did not temporarily expand the matching path: rows=%d open=%v", m.reportTree.Rows(), slices.Collect(m.reportTree.Open.All()))
	}
	if got := m.reportTree.Item(m.reportTree.Rows() - 1); got != "Assets:Test:Child" {
		t.Fatalf("deep search last visible account = %q, want Assets:Test:Child", got)
	}

	tester = ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("清除搜索"); err != nil {
		t.Fatalf("clear account search: %v", err)
	}
	if m.statementSearch != "" || m.reportTree.Open.Len() != 0 || m.reportTree.Rows() != len(response.Tree.RootNodes) {
		t.Fatalf("clearing search did not restore the user's collapsed tree: search=%q rows=%d roots=%d open=%v", m.statementSearch, m.reportTree.Rows(), len(response.Tree.RootNodes), slices.Collect(m.reportTree.Open.All()))
	}
}

func TestStatementExpandAllWhileSearchingUpdatesRestoredState(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "account-detail.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "balance_sheet", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page = pageBalance
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("全部折叠"); err != nil {
		t.Fatal(err)
	}
	m.statementSearchDraft = "Assets:Test:Child"
	tester = ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("筛选账户"); err != nil {
		t.Fatal(err)
	}
	tester = ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("全部展开"); err != nil {
		t.Fatalf("expand the report while searching: %v", err)
	}
	if m.reportTree.Open.Len() < 2 {
		t.Fatalf("expand all did not open the full tree during search: open=%d", m.reportTree.Open.Len())
	}
	tester = ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("清除搜索"); err != nil {
		t.Fatalf("clear search after expand all: %v", err)
	}
	if m.statementSearch != "" || m.reportTree.Open.Len() < 2 || m.reportTree.Rows() <= len(response.Tree.RootNodes) {
		t.Fatalf("clearing search lost the explicit expand-all action: search=%q open=%d rows=%d roots=%d", m.statementSearch, m.reportTree.Open.Len(), m.reportTree.Rows(), len(response.Tree.RootNodes))
	}
}

func TestStatementTreeExpansionResetsWhenReportContextChanges(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	balance, err := session.Read(desktop.Request{View: "balance_sheet", Valuation: "at-cost", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	income, err := session.Read(desktop.Request{View: "income_statement", Valuation: "at-cost", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page = pageBalance
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Response: &balance}
	balanceTester := ui.NewTester(m.View, 1180, 780)
	if err := balanceTester.Click("全部折叠"); err != nil || m.reportTree.Open.Len() != 0 {
		t.Fatalf("collapse balance tree: err=%v open=%d", err, m.reportTree.Open.Len())
	}
	m.page = pageIncome
	m.workspaceState.Response = &income
	incomeTester := ui.NewTester(m.View, 1180, 780)
	if m.reportTree.Open.Len() == 0 || !incomeTester.HasText("Salary") || !incomeTester.HasText("Food") {
		t.Fatalf("income tree did not initialize its own expanded state: open=%d UI=%q", m.reportTree.Open.Len(), incomeTester.Texts())
	}
}

func TestStatementTreePreservesExpansionAndSelectionOnSameLedgerReload(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "account-detail.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "balance_sheet", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	response.State.SnapshotID = "snapshot-before"
	m := NewModel(true)
	m.page = pageBalance
	m.workspaceState = workspace.State{
		Phase: workspace.PhaseReady, ActivePath: path,
		Ledger: desktop.State{SnapshotID: "snapshot-before"}, Response: &response,
	}
	tester := ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("全部折叠"); err != nil || m.reportTree.Open.Len() != 0 {
		t.Fatalf("collapse account tree before reload: err=%v open=%d", err, m.reportTree.Open.Len())
	}
	m.statementSelectedAccount = "Assets:Test:Child"

	reloaded := response
	reloaded.State.SnapshotID = "snapshot-after"
	m.workspaceState.Ledger.SnapshotID = "snapshot-after"
	m.workspaceState.Response = &reloaded
	ui.NewTester(m.View, 1180, 780)
	if m.reportTree.Open.Len() != 0 || m.statementSelectedAccount != "Assets:Test:Child" {
		t.Fatalf("same-ledger reload changed account-tree state: open=%d selected=%q", m.reportTree.Open.Len(), m.statementSelectedAccount)
	}
}

func TestAccountDetailPageRendersCoreLifecycleHistoryAndTransactions(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "account-detail.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	response, err := session.Read(desktop.Request{
		View: "account", Account: "Assets:Test:Child", Scope: "direct", Sort: "descending", Limit: 100,
		SnapshotID: state.SnapshotID, DateFrom: "2000-01-03", DateTo: "2000-01-06", Window: "all", Valuation: "at-cost", Locale: "zh-CN",
	})
	if err != nil {
		t.Fatal(err)
	}
	var revealed string
	m := NewModel(true)
	m.page = pageAccount
	m.accountName = "Assets:Test:Child"
	m.accountScope = "direct"
	m.filterDateFrom, m.filterDateTo = "2000-01-03", "2000-01-06"
	m.workspaceState = workspace.State{
		Phase: workspace.PhaseReady, ActivePath: path, Ledger: desktop.State{SnapshotID: state.SnapshotID}, Response: &response,
	}
	m.revealPath = func(value string) { revealed = value }
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"Assets:Test:Child", "已关闭", "关闭日期 2000-01-05", "记账方式 FIFO", "账户余额", "期间变化", "15", "-5", "历史余额", "15 USD", "账户生命周期来源", "Child reduction", "1 笔账户交易"} {
		if !tester.HasText(want) {
			t.Errorf("account detail page does not render %q: %q", want, tester.Texts())
		}
	}
	if err := tester.Click("· Child reduction"); err != nil {
		t.Fatalf("select account transaction: %v; UI=%q", err, tester.Texts())
	}
	tester.Frame()
	for _, want := range []string{"manual-review-7", "posting_note", "交易来源 ·", "显示交易来源", "元数据", "来源"} {
		if !tester.HasText(want) {
			t.Errorf("selected account transaction does not render %q: %q", want, tester.Texts())
		}
	}
	if err := tester.Click("在 Finder 中显示"); err != nil {
		t.Fatal(err)
	}
	if revealed != path {
		t.Fatalf("lifecycle source Finder path = %q, want %q; UI=%q", revealed, path, tester.Texts())
	}
	if err := tester.Click("显示交易来源"); err != nil {
		t.Fatalf("click account transaction source: %v; UI=%q", err, tester.Texts())
	}
	if revealed != path {
		t.Fatalf("transaction source Finder path = %q, want %q", revealed, path)
	}
}

func TestAccountRequestCarriesScopeFiltersAndSnapshot(t *testing.T) {
	m := NewModel(true)
	m.accountName, m.accountScope = "Assets:Cash", "direct"
	m.accountJournalSearch, m.accountJournalOffset = "lunch", 100
	m.workspaceState.Ledger.SnapshotID = "account-snapshot"
	m.filterDateFrom, m.filterDateTo, m.filterWindow = "2026-01-01", "2026-01-31", "all"
	m.filterCurrency, m.filterValuation = "USD", "market-value"
	request := m.requestForView("account")
	if request.View != "account" || request.Account != "Assets:Cash" || request.Scope != "direct" || request.Text != "lunch" || request.Sort != "descending" || request.Offset != 100 || request.Limit != 100 || request.SnapshotID != "account-snapshot" || request.DateFrom != "2026-01-01" || request.DateTo != "2026-01-31" || request.Window != "all" || request.Currency != "USD" || request.Valuation != "market-value" {
		t.Fatalf("account request omitted its identity, scope, filters, or snapshot: %+v", request)
	}
}

func TestAccountDetailNavigationReturnsToFilteredStatement(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	controller := workspace.NewController("account-detail-navigation", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("account-detail workspace did not close")
		}
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: path, Options: workspace.OpenOptions{SemanticsProfile: ledger.V3}}); err != nil {
		t.Fatal(err)
	}
	opened := waitWorkspace(t, controller, func(state workspace.State) bool { return state.Phase == workspace.PhaseReady && !state.Busy })
	m := NewModel(true)
	m.workspace = controller
	m.workspaceState = opened
	m.page = pageBalance
	m.statementSearch, m.statementSearchDraft = "assets:bank", "assets:bank"
	m.requestView("balance_sheet")
	statement := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Tree != nil && state.Response.Request.View == "balance_sheet" && !state.Busy
	})
	m.workspaceState = statement
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("查看账户 Assets:Bank") {
		t.Fatalf("filtered statement has no account-detail action: %q", tester.Texts())
	}
	if err := tester.Click("Assets:Bank"); err != nil {
		t.Fatalf("select account row: %v; UI=%q", err, tester.Texts())
	}
	if m.statementSelectedAccount != "Assets:Bank" {
		t.Fatalf("statement selection did not capture the clicked account: %q", m.statementSelectedAccount)
	}
	if !tester.HasText("查看账户详情 Assets:Bank") {
		t.Fatalf("selected account has no explicit detail action: %q", tester.Texts())
	}
	if err := tester.Click("查看账户详情 Assets:Bank"); err != nil {
		t.Fatalf("open selected account detail: %v; UI=%q", err, tester.Texts())
	}
	if m.page != pageAccount || m.accountName != "Assets:Bank" {
		t.Fatalf("account action selected page=%d account=%q", m.page, m.accountName)
	}
	account := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Account != nil && !state.Busy
	})
	m.workspaceState = account
	tester = ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("Assets:Bank") || !tester.HasText("账户流水") {
		t.Fatalf("account detail did not load from the core: %q", tester.Texts())
	}
	if err := tester.Click("返回报表"); err != nil {
		t.Fatal(err)
	}
	if m.page != pageBalance || m.statementSearch != "assets:bank" || m.statementSearchDraft != "assets:bank" {
		t.Fatalf("return lost report page or local search: page=%d search=%q draft=%q", m.page, m.statementSearch, m.statementSearchDraft)
	}
	returned := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Tree != nil && state.Response.Request.View == "balance_sheet" && !state.Busy
	})
	m.workspaceState = returned
	tester = ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("查看账户 Assets:Bank") || tester.HasText("查看账户 Expenses:Food") {
		t.Fatalf("return did not restore the filtered report: %q", tester.Texts())
	}
	if m.reportTree.List.Selected == nil || m.statementSelection < 0 || m.statementSelection >= m.reportTree.Rows() || m.reportTree.Item(m.statementSelection) != "Assets:Bank" {
		t.Fatalf("return did not restore selected account row: selected=%d rows=%d identity=%q", m.statementSelection, m.reportTree.Rows(), m.statementSelectedAccount)
	}
	if err := tester.Click("流水"); err != nil {
		t.Fatalf("navigate to journal: %v; UI=%q", err, tester.Texts())
	}
	journal := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Journal != nil && state.Response.Request.View == "journal" && !state.Busy
	})
	m.workspaceState = journal
	tester.Frame()
	if err := tester.Click("返回"); err != nil {
		t.Fatalf("navigate back to the statement: %v; UI=%q", err, tester.Texts())
	}
	backed := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Tree != nil && state.Response.Request.View == "balance_sheet" && !state.Busy
	})
	m.workspaceState = backed
	tester.Frame()
	if m.page != pageBalance || m.statementSearch != "assets:bank" || !tester.HasText("查看账户 Assets:Bank") {
		t.Fatalf("back navigation lost the page or local search: page=%d search=%q UI=%q", m.page, m.statementSearch, tester.Texts())
	}
	if err := tester.Click("前进"); err != nil {
		t.Fatalf("navigate forward to journal: %v; UI=%q", err, tester.Texts())
	}
	forwardedJournal := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Journal != nil && state.Response.Request.View == "journal" && !state.Busy
	})
	m.workspaceState = forwardedJournal
	tester.Frame()
	if m.page != pageJournal {
		t.Fatalf("forward navigation selected page %d, want journal", m.page)
	}
	if err := tester.Click("返回"); err != nil {
		t.Fatalf("navigate back before choosing another page: %v", err)
	}
	backed = waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Tree != nil && state.Response.Request.View == "balance_sheet" && !state.Busy
	})
	m.workspaceState = backed
	tester.Frame()
	if err := tester.Click("财务概览"); err != nil {
		t.Fatalf("navigate to overview: %v", err)
	}
	if m.page != pageDashboard || len(m.pageHistoryForward) != 0 {
		t.Fatalf("new navigation did not replace the forward branch: page=%d forward=%v", m.page, m.pageHistoryForward)
	}
}

func TestJournalPageSelectsTransactionAndShowsExactPostings(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "journal", JournalMode: "transactions", Sort: "descending", Limit: 100, Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page = pageJournal
	m.journalSelection = -1
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("Salary"); err != nil {
		t.Fatalf("select Salary transaction: %v; UI=%q", err, tester.Texts())
	}
	for _, want := range []string{"Assets:Bank", "Income:Salary", "1000 USD", "-1000 USD"} {
		if !tester.HasText(want) {
			t.Errorf("journal details do not render %q: %q", want, tester.Texts())
		}
	}
}

func TestJournalPageShowsExactCostPriceMetadataAndSources(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "holdings-trades.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	response, err := session.Read(desktop.Request{
		View: "journal", JournalMode: "transactions", Sort: "ascending", Limit: 100,
		SnapshotID: state.SnapshotID, Window: "all", Locale: "zh-CN",
	})
	if err != nil {
		t.Fatal(err)
	}
	selectTransaction := func(narration string) *Model {
		t.Helper()
		for _, tx := range response.Journal.Transactions {
			if tx.Narration != narration {
				continue
			}
			m := NewModel(true)
			m.page = pageJournal
			m.journalDescending = false
			m.journalSelectedID = tx.ID
			m.workspaceState = workspace.State{
				Phase: workspace.PhaseReady, ActivePath: path,
				Ledger: desktop.State{SnapshotID: state.SnapshotID}, Response: &response,
			}
			return m
		}
		t.Fatalf("journal transaction %q was not projected", narration)
		return nil
	}

	lots := ui.NewTester(selectTransaction("first lots").View, 1180, 780)
	if !lots.HasText("{10 USD, 2000-01-02}") || !lots.HasText("Assets:Fifo") {
		t.Fatalf("exact lot cost is missing from the journal detail: %q", lots.Texts())
	}

	reduction := ui.NewTester(selectTransaction("two total-price reductions").View, 1180, 780)
	if !reduction.HasText("@@ 100 USD") {
		t.Fatalf("exact total price is missing from the journal detail: %q", reduction.Texts())
	}

	fee := ui.NewTester(selectTransaction("attributed fee").View, 1180, 780)
	if !fee.HasText("@ 20 USD") || !fee.HasText(`trade_fee_for："Assets:Fees"`) || !fee.HasText("Expenses:TradeFees") || !fee.HasText("holdings-trades.bean:") {
		t.Fatalf("exact price, typed metadata, or source location is missing from the journal detail: %q", fee.Texts())
	}
}

func TestJournalSearchAndPaginationUseSnapshotBoundCorePages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.bean")
	var source strings.Builder
	source.WriteString("option \"operating_currency\" \"USD\"\n2026-01-01 open Assets:Cash USD\n2026-01-01 open Income:Salary USD\n")
	for index := 0; index < 105; index++ {
		fmt.Fprintf(&source, "2026-01-02 * \"Txn %03d\"\n  Assets:Cash 1 USD\n  Income:Salary -1 USD\n", index)
	}
	if err := os.WriteFile(path, []byte(source.String()), 0600); err != nil {
		t.Fatal(err)
	}
	controller := workspace.NewController("journal-pagination-ui", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("journal workspace did not close")
		}
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: path, Options: workspace.OpenOptions{
		SemanticsProfile: ledger.V3, Watch: snapshot.WatchOptions{PollInterval: time.Hour},
	}}); err != nil {
		t.Fatal(err)
	}
	opened := waitWorkspace(t, controller, func(state workspace.State) bool { return state.Phase == workspace.PhaseReady && !state.Busy })
	m := NewModel(true)
	m.workspace = controller
	m.workspaceState = opened
	m.page = pageJournal
	m.requestView("journal")
	firstPage := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Journal != nil && state.Response.Journal.Offset == 0 && !state.Busy
	})
	m.workspaceState = firstPage
	tester := ui.NewTester(m.View, 1180, 780)
	if firstPage.Response.Journal.TotalCount != 105 || len(firstPage.Response.Journal.Transactions) != 100 || !firstPage.Response.Journal.HasMore || !tester.HasText("Txn 104") {
		t.Fatalf("first journal page is incorrect: payload=%+v UI=%q", firstPage.Response.Journal, tester.Texts())
	}
	if err := tester.Click("下一页"); err != nil {
		t.Fatalf("open next journal page: %v", err)
	}
	secondPage := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Journal != nil && state.Response.Journal.Offset == 100 && !state.Busy
	})
	m.workspaceState = secondPage
	tester = ui.NewTester(m.View, 1180, 780)
	if secondPage.Response.Request.SnapshotID != opened.Ledger.SnapshotID || secondPage.Response.Journal.TotalCount != 105 || len(secondPage.Response.Journal.Transactions) != 5 || secondPage.Response.Journal.HasMore {
		t.Fatalf("second journal page lost its snapshot or bounds: %+v", secondPage.Response.Journal)
	}
	if err := tester.Click("上一页"); err != nil {
		t.Fatalf("return to first journal page: %v", err)
	}
	firstPage = waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Journal != nil && state.Response.Journal.Offset == 0 && !state.Busy
	})
	m.workspaceState = firstPage
	m.journalSearchDraft = "Txn 002"
	tester = ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("筛选流水"); err != nil {
		t.Fatalf("apply journal search: %v", err)
	}
	filtered := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Response != nil && state.Response.Journal != nil && state.Response.Request.Text == "Txn 002" && !state.Busy
	})
	m.workspaceState = filtered
	tester = ui.NewTester(m.View, 1180, 780)
	if filtered.Response.Journal.TotalCount != 1 || len(filtered.Response.Journal.Transactions) != 1 || !tester.HasText("Txn 002") || tester.HasText("Txn 104") {
		t.Fatalf("journal search did not replace the prior page with its match: payload=%+v UI=%q", filtered.Response.Journal, tester.Texts())
	}
}

func TestJournalRequestCarriesSearchAccountDateSortAndSnapshot(t *testing.T) {
	m := NewModel(true)
	m.workspaceState.Ledger.SnapshotID = "profile-snapshot"
	m.filterDateFrom, m.filterDateTo, m.filterWindow = "2026-01-01", "2026-01-31", "all"
	m.filterCurrency, m.filterValuation = "USD", "market-value"
	m.journalSearch = "salary"
	m.journalAccount = "Assets:Cash"
	m.journalDescending = false
	m.journalOffset = 100
	request := m.requestForView("journal")
	if request.View != "journal" || request.JournalMode != "transactions" || request.Text != "salary" || request.Account != "Assets:Cash" || request.Sort != "ascending" || request.Offset != 100 || request.Limit != 100 || request.SnapshotID != "profile-snapshot" || request.DateFrom != "2026-01-01" || request.DateTo != "2026-01-31" || request.Window != "all" {
		t.Fatalf("journal request omitted an applied filter or snapshot boundary: %+v", request)
	}
	if request.Currency != "" || request.Valuation != "" {
		t.Fatalf("journal request implies unsupported currency valuation controls: %+v", request)
	}
}

func TestQueryPageShowsQueryCancellationProgress(t *testing.T) {
	m := NewModel(true)
	m.page = pageQuery
	m.workspaceState = workspace.State{QueryBusy: true, QueryCancelling: true}
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("正在取消…") || tester.HasText("正在执行…") {
		t.Fatalf("query page did not distinguish cancellation from execution: %q", tester.Texts())
	}
	if !tester.HasText("取消查询") {
		t.Fatal("query page removed its cancel action while cancellation is pending")
	}
}

func TestQueryPageExecutesExplicitDraftAndRendersTypedResults(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "query", "default-source.bean")
	controller := workspace.NewController("query-ui", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("query workspace did not close")
		}
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: path, Options: workspace.OpenOptions{SemanticsProfile: ledger.V3}}); err != nil {
		t.Fatal(err)
	}
	state := waitWorkspace(t, controller, func(s workspace.State) bool { return s.Phase == workspace.PhaseReady })

	m := NewModel(true)
	m.page = pageQuery
	m.workspace = controller
	m.workspaceState = state
	tester := ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("BQL 查询"); err != nil {
		t.Fatal(err)
	}
	tester.Type("SELECT 1 AS v, 2 AS v, NULL AS n, '' AS blank LIMIT 1")
	if m.queryDraft != "SELECT 1 AS v, 2 AS v, NULL AS n, '' AS blank LIMIT 1" {
		t.Fatalf("typing did not update query draft: %q", m.queryDraft)
	}
	if got := controller.State(); got.QueryBusy || got.QueryResponse != nil {
		t.Fatalf("editing a draft ran it implicitly: busy=%v response=%+v", got.QueryBusy, got.QueryResponse)
	}
	if err := tester.Click("执行查询"); err != nil {
		t.Fatal(err)
	}
	state = waitWorkspace(t, controller, func(s workspace.State) bool { return s.QueryResponse != nil && !s.QueryBusy })
	if response := state.QueryResponse; response.Error != nil || response.Execution == nil {
		t.Fatalf("explicit query did not produce a result: %+v", response)
	}
	if execution := state.QueryResponse.Execution; execution.ReturnedRows != 1 || len(execution.Result.Columns) != 4 || len(execution.Result.Rows) != 1 {
		t.Fatalf("typed result shape = rows:%d columns:%d data:%v", execution.ReturnedRows, len(execution.Result.Columns), execution.Result.Rows)
	}
	m.workspaceState = state
	tester.Frame()
	for _, want := range []string{"执行文本", "SELECT 1 AS v, 2 AS v, NULL AS n, '' AS blank LIMIT 1", "1 行", "v", "1", "2", "NULL", `""`} {
		if !tester.HasText(want) {
			t.Errorf("query page does not render %q: %q", want, tester.Texts())
		}
	}
	if !tester.HasText("查询指纹") || !tester.HasText("工作步数") {
		t.Fatalf("query execution context or budget counters are missing: %q", tester.Texts())
	}
	if err := tester.Click("BQL 查询"); err != nil {
		t.Fatal(err)
	}
	tester.Key(ui.Cmd, ui.KeyA)
	tester.Type("SELECT 3 AS v LIMIT 1")
	if got := controller.State(); got.QueryBusy || got.QueryResponse == nil || got.QueryResponse.Context.Text != "SELECT 1 AS v, 2 AS v, NULL AS n, '' AS blank LIMIT 1" {
		t.Fatalf("editing a new draft disturbed the previous execution: %+v", got)
	}
	tester.Frame()
	if !tester.HasText("SELECT 1 AS v, 2 AS v, NULL AS n, '' AS blank LIMIT 1") {
		t.Fatalf("editing the draft hid the prior execution context: %q", tester.Texts())
	}
	tester.Key(ui.Cmd, ui.KeyEnter)
	state = waitWorkspace(t, controller, func(s workspace.State) bool {
		return s.QueryResponse != nil && !s.QueryBusy && s.QueryResponse.Context.RequestID > 1
	})
	if response := state.QueryResponse; response.Error != nil || response.Execution == nil || response.Context.Text != "SELECT 3 AS v LIMIT 1" {
		t.Fatalf("Command-Enter did not run the latest draft: %+v", response)
	}
}

func TestQueryPageSelectionCopiesTypedContextAndFullPositionCost(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "query", "compound-values.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	text := "SELECT account, position WHERE account = 'Assets:Stock' LIMIT 1"
	response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 42, Text: text})
	if err != nil {
		t.Fatal(err)
	}
	if response.Error != nil || response.Execution == nil || len(response.Execution.Result.Rows) != 1 {
		t.Fatalf("compound query did not produce one typed row: %+v", response)
	}

	var copied string
	m := NewModel(true)
	m.page = pageQuery
	m.queryDraft = text
	m.writeClipboard = func(value string) { copied = value }
	m.workspaceState = workspace.State{
		Phase: workspace.PhaseReady, ActivePath: path,
		Ledger: desktop.State{SnapshotID: state.SnapshotID}, QueryResponse: &response,
	}
	tester := ui.NewTester(m.View, 1180, 1180)
	if err := tester.Click("Assets:Stock"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !tester.HasText("所选结果行 · 1") || !tester.HasText("position · position") || !tester.HasText("alpha") {
		t.Fatalf("selected compound row details are missing: %q", tester.Texts())
	}
	if err := tester.Click("复制类型化结果行"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	var payload struct {
		Context  desktop.QueryContext `json:"context"`
		Contract string               `json:"contract"`
		Columns  []query.Column       `json:"columns"`
		Cells    []query.Cell         `json:"cells"`
	}
	if err := json.Unmarshal([]byte(copied), &payload); err != nil {
		t.Fatalf("copied row is not typed JSON: %v (%s)", err, copied)
	}
	if payload.Context != response.Context || payload.Contract != query.CurrentContract || len(payload.Columns) != 2 || len(payload.Cells) != 2 {
		t.Fatalf("copied row lost execution context or ordered schema: %+v", payload)
	}
	if payload.Columns[0].Name != "account" || payload.Columns[1].Name != "position" || payload.Cells[1].Type != "position" {
		t.Fatalf("copied row changed its positional typed schema: %+v", payload)
	}
	for _, want := range []string{`"currency":"USD"`, `"exact":"10"`, `"label":"alpha"`} {
		if !strings.Contains(copied, want) {
			t.Errorf("copied position lost full cost component %q: %s", want, copied)
		}
	}
}

func TestQueryPageKeepsTenThousandResultRowsVirtualized(t *testing.T) {
	rows := make([][]query.Cell, 10_000)
	for index := range rows {
		rows[index] = []query.Cell{{Type: "integer", Value: fmt.Sprint(index + 1)}}
	}
	response := desktop.QueryResponse{
		Context: desktop.QueryContext{SnapshotID: "query-snapshot", RequestID: 9, Text: "SELECT number"},
		Execution: &query.Execution{
			Result:       query.TypedResult{Contract: query.CurrentContract, Columns: []query.Column{{ID: "c0", Name: "number", Type: "integer"}}, Rows: rows},
			ReturnedRows: len(rows), RowLimit: len(rows),
		},
	}
	m := NewModel(true)
	m.page = pageQuery
	m.workspaceState = workspace.State{
		Phase:         workspace.PhaseReady,
		Ledger:        desktop.State{SnapshotID: response.Context.SnapshotID},
		QueryResponse: &response,
	}
	tester := ui.NewTester(m.View, 1180, 780)
	visibleCells := 0
	for _, text := range tester.Texts() {
		if text != "" && strings.Trim(text, "0123456789") == "" {
			visibleCells++
		}
	}
	if visibleCells == 0 || visibleCells >= 100 {
		t.Fatalf("10,000-row query eagerly built %d numeric cell labels (all rows should remain virtualized)", visibleCells)
	}
	firstCell, ok := tester.Find("1")
	if !ok {
		t.Fatal("the first virtual row is not present before scrolling")
	}
	if err := tester.Click("1"); err != nil {
		t.Fatal(err)
	}
	if m.querySelection != 0 {
		t.Fatalf("mouse click did not select the first virtual row: %d", m.querySelection)
	}
	for attempt := 0; attempt < 10 && !m.queryTable.AtEnd(); attempt++ {
		tester.Scroll(firstCell.X+firstCell.W/2, firstCell.Y+firstCell.H/2, 0, 100_000)
	}
	if !m.queryTable.AtEnd() || !tester.HasText("10000") {
		t.Fatalf("mouse scrolling did not reach the final virtual rows: atEnd=%v labels=%q", m.queryTable.AtEnd(), tester.Texts())
	}
	tester.Key(0, ui.KeyHome)
	if m.querySelection != 0 {
		t.Fatalf("Home did not select the first virtual row: %d", m.querySelection)
	}
	tester.Key(0, ui.KeyEnd)
	if m.querySelection != len(rows)-1 || !m.queryTable.AtEnd() || !tester.HasText("所选结果行 · 10000") {
		t.Fatalf("End did not select and scroll to the final typed row: selection=%d atEnd=%v labels=%q", m.querySelection, m.queryTable.AtEnd(), tester.Texts())
	}
}

func TestQueryTableSortsTypedIntegersAndKeepsSelectedRowForCopy(t *testing.T) {
	response := desktop.QueryResponse{
		Context: desktop.QueryContext{SnapshotID: "sort-snapshot", RequestID: 7, Text: "SELECT number", Fingerprint: "sort-query"},
		Execution: &query.Execution{
			Result: query.TypedResult{
				Contract: query.CurrentContract,
				Columns:  []query.Column{{ID: "c0", Name: "rank", Type: "integer", Nullable: true}},
				Rows: [][]query.Cell{
					{{Type: "integer", Value: "10"}},
					{{Type: "integer", Value: "2"}},
					{{Type: "integer", Value: "1"}},
				},
			},
			ReturnedRows: 3,
			RowLimit:     100,
		},
	}
	var copied string
	m := NewModel(true)
	m.page = pageQuery
	m.workspaceState = workspace.State{
		Phase:         workspace.PhaseReady,
		Ledger:        desktop.State{SnapshotID: response.Context.SnapshotID},
		QueryResponse: &response,
	}
	m.writeClipboard = func(value string) { copied = value }
	tester := ui.NewTester(m.View, 1180, 1180)
	if err := tester.Click("rank"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if m.querySort != (ui.SortOrder{Column: "c0"}) || !tester.HasText("1") {
		t.Fatalf("clicking the typed integer header did not sort ascending: sort=%+v, labels=%q", m.querySort, tester.Texts())
	}
	if err := tester.Click("10"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if m.querySelection != 2 {
		t.Fatalf("selected display row = %d, want the third row in ascending order", m.querySelection)
	}
	if err := tester.Click("rank"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if m.querySort != (ui.SortOrder{Column: "c0", Descending: true}) || m.querySelection != 0 {
		t.Fatalf("descending sort did not preserve the selected result identity: sort=%+v, selection=%d", m.querySort, m.querySelection)
	}
	if err := tester.Click("复制类型化结果行"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	var payload struct {
		Cells []query.Cell `json:"cells"`
	}
	if err := json.Unmarshal([]byte(copied), &payload); err != nil {
		t.Fatalf("copied sorted row is not typed JSON: %v (%q)", err, copied)
	}
	if len(payload.Cells) != 1 || payload.Cells[0].Type != "integer" || payload.Cells[0].Value != "10" {
		t.Fatalf("sorting changed the selected source row: %+v", payload.Cells)
	}
}

func TestQueryPageRendersZeroRowTypedResultSchema(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "query", "default-source.bean")
	controller := workspace.NewController("query-empty-ui", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("empty-result query workspace did not close")
		}
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: path, Options: workspace.OpenOptions{SemanticsProfile: ledger.V3}}); err != nil {
		t.Fatal(err)
	}
	state := waitWorkspace(t, controller, func(state workspace.State) bool { return state.Phase == workspace.PhaseReady })
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentExecuteQuery, QueryText: "SELECT account, position WHERE account = 'Assets:Missing' LIMIT 1"}); err != nil {
		t.Fatal(err)
	}
	state = waitWorkspace(t, controller, func(state workspace.State) bool { return state.QueryResponse != nil && !state.QueryBusy })
	if response := state.QueryResponse; response.Error != nil || response.Execution == nil || response.Execution.ReturnedRows != 0 || len(response.Execution.Result.Columns) != 2 || len(response.Execution.Result.Rows) != 0 {
		t.Fatalf("zero-row typed query shape = %+v", response)
	}
	m := NewModel(true)
	m.page = pageQuery
	m.workspaceState = state
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"0/10000 行", "account", "position"} {
		if !tester.HasText(want) {
			t.Errorf("zero-row query view omitted %q: %q", want, tester.Texts())
		}
	}
	if tester.HasText("所选结果行") || m.querySelection != -1 {
		t.Fatalf("zero-row result showed a selection: selected=%d, labels=%q", m.querySelection, tester.Texts())
	}
	if err := tester.Click("account"); err != nil {
		t.Fatal(err)
	}
	if m.querySort.Column != "c0" || m.querySelection != -1 {
		t.Fatalf("empty typed table did not handle sortable schema safely: sort=%+v selection=%d", m.querySort, m.querySelection)
	}
}

func TestQueryCellDetailsPreserveNestedTypesExactIntegersAndBoundExpansion(t *testing.T) {
	largeInteger := "900719925474099312345678901234567890"
	cell := query.Cell{Type: "transaction", Value: query.PresentedDictionary{
		"blank":   query.Cell{Type: "string", Value: ""},
		"nothing": query.Cell{Type: "null"},
		"count":   query.Cell{Type: "integer", Value: largeInteger},
		"amount": query.Cell{Type: "amount", Value: query.PresentedAmount{
			Number: query.PresentedDecimal{Display: "0.333333", Exact: "1/3", Approximate: true}, Currency: "USD",
		}},
	}}
	joined := strings.Join(queryCellDetails(cell), "\n")
	for _, want := range []string{`"blank" · ""`, `"nothing" · NULL`, largeInteger, "近似显示；精确值 1/3", "USD"} {
		if !strings.Contains(joined, want) {
			t.Errorf("recursive details lost %q: %s", want, joined)
		}
	}

	large := query.PresentedDictionary{}
	for index := 0; index < 300; index++ {
		large[fmt.Sprintf("field-%03d", index)] = query.Cell{Type: "integer", Value: fmt.Sprint(index)}
	}
	lines := queryCellDetails(query.Cell{Type: "dictionary", Value: large})
	if len(lines) != 200 || !strings.Contains(lines[len(lines)-1], "其余详情已折叠") {
		t.Fatalf("large metadata detail was not bounded: %d lines, last=%q", len(lines), lines[len(lines)-1])
	}
}

func TestOfflineQueryAnalysisLoadsTypedResultWithoutTouchingActiveLedger(t *testing.T) {
	archivePath, document := writeQueryAnalysisFixture(t,
		filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "query", "compound-values.bean"),
		"SELECT account, position WHERE account = 'Assets:Stock' LIMIT 1",
	)
	activePath := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "query", "default-source.bean")
	controller := workspace.NewController("offline-analysis-ui", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("offline analysis workspace did not close")
		}
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: activePath, Options: workspace.OpenOptions{SemanticsProfile: ledger.V3}}); err != nil {
		t.Fatal(err)
	}
	state := waitWorkspace(t, controller, func(s workspace.State) bool { return s.Phase == workspace.PhaseReady })
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentExecuteQuery, QueryText: "SELECT account LIMIT 1"}); err != nil {
		t.Fatal(err)
	}
	state = waitWorkspace(t, controller, func(s workspace.State) bool { return s.QueryResponse != nil && !s.QueryBusy })
	liveContext := state.QueryResponse.Context

	m := NewModel(true)
	m.page = pageQuery
	m.workspace = controller
	m.workspaceState = state
	m.queryDraft = "SELECT 7 AS draft LIMIT 1"
	m.offlineAnalysis = &document
	m.offlineAnalysisPath = archivePath
	m.querySelection = 0
	m.querySelectionContext = queryResultSelectionContext(document.Context.NormalizedRequest, archivePath)
	var copied string
	m.writeClipboard = func(value string) { copied = value }
	tester := ui.NewTester(m.View, 1180, 1180)
	for _, want := range []string{
		"离线分析捕获", "不会重新执行原查询", "capture scope captured-result", "query-result.ocanalysis",
		document.Context.NormalizedRequest.Text, "数量 2 STK", "成本 10 USD", "成本日期 2000-01-02", `标签 "alpha"`,
	} {
		if !tester.HasText(want) {
			t.Errorf("offline capture view omitted %q: %q", want, tester.Texts())
		}
	}
	if err := tester.Click("复制类型化结果行"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	result := document.Result
	wantCopy, err := json.Marshal(struct {
		Context  desktop.QueryContext `json:"context"`
		Contract string               `json:"contract"`
		Columns  []query.Column       `json:"columns"`
		Cells    []query.Cell         `json:"cells"`
	}{Context: document.Context.NormalizedRequest, Contract: result.Contract, Columns: result.Columns, Cells: result.Rows[0]})
	if err != nil {
		t.Fatal(err)
	}
	if copied != string(wantCopy) {
		t.Fatalf("offline typed row copy changed its captured context/schema/cells:\ngot:  %s\nwant: %s", copied, wantCopy)
	}
	if m.queryDraft != "SELECT 7 AS draft LIMIT 1" {
		t.Fatalf("opening the capture replaced the user's query draft: %q", m.queryDraft)
	}
	if err := tester.Click("填入捕获查询草稿"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if m.queryDraft != document.Context.NormalizedRequest.Text {
		t.Fatalf("explicit fill did not copy only the captured query text: %q", m.queryDraft)
	}
	current := controller.State()
	if current.ActivePath != state.ActivePath || current.QueryResponse == nil || current.QueryBusy || current.QueryResponse.Context != liveContext {
		t.Fatalf("offline archive changed the active session or reran its query: %+v", current)
	}
	if err := tester.Click("关闭离线捕获"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if tester.HasText("离线分析捕获") || !tester.HasText(liveContext.Text) {
		t.Fatalf("closing the capture did not return to the untouched live result: %q", tester.Texts())
	}

	corruptPath := filepath.Join(t.TempDir(), "bad.ocanalysis")
	if err := os.WriteFile(corruptPath, []byte("not a query archive"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readQueryAnalysisFile(corruptPath); err == nil {
		t.Fatal("corrupt offline analysis was accepted")
	}
	oversizedPath := filepath.Join(t.TempDir(), "oversized.ocanalysis")
	oversized, err := os.Create(oversizedPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := oversized.Truncate(int64(query.QueryByteLimit) + 1); err != nil {
		oversized.Close()
		t.Fatal(err)
	}
	if err := oversized.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readQueryAnalysisFile(oversizedPath); err == nil {
		t.Fatal("over-budget offline analysis was accepted")
	}
}

func writeQueryAnalysisFixture(t *testing.T, fixture, text string) (string, desktop.QueryAnalysisDocument) {
	t.Helper()
	session, err := desktop.OpenWithOptions(fixture, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: text})
	if err != nil {
		t.Fatal(err)
	}
	if response.Error != nil || response.Execution == nil {
		t.Fatalf("fixture query failed: %+v", response)
	}
	var archive bytes.Buffer
	if err := response.WriteAnalysis(&archive); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "query-result.ocanalysis")
	if err := os.WriteFile(path, archive.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	document, err := readQueryAnalysisFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, document
}

func TestQueryRunRejectsOversizedDraftAndMarkedComposition(t *testing.T) {
	for _, tc := range []struct {
		name        string
		draft       string
		composition string
		wantStatus  string
	}{
		{name: "64 KiB UTF-8 limit", draft: strings.Repeat("x", 64*1024+1), wantStatus: "查询文本超过 64 KiB UTF-8 限制"},
		{name: "marked IME composition", draft: "", composition: "SELECT 1", wantStatus: "请先完成输入法组合，再执行查询"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			controller := workspace.NewController("query-guard", workspace.DesktopEngine{}, nil)
			t.Cleanup(controller.Close)
			m := NewModel(true)
			m.page = pageQuery
			m.workspace = controller
			m.queryDraft = tc.draft
			m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: "/tmp/query-guard.bean"}
			tester := ui.NewTester(m.View, 1180, 780)
			if tc.composition != "" {
				if err := tester.Click("BQL 查询"); err != nil {
					t.Fatal(err)
				}
				tester.Compose(tc.composition, len(tc.composition))
				tester.Key(ui.Cmd, ui.KeyEnter)
			} else {
				if err := tester.Click("执行查询"); err != nil {
					t.Fatal(err)
				}
			}
			if m.status != tc.wantStatus {
				t.Fatalf("guard status = %q, want %q", m.status, tc.wantStatus)
			}
			if state := controller.State(); state.QueryBusy || state.QueryResponse != nil {
				t.Fatalf("rejected query was dispatched: %+v", state)
			}
		})
	}
}

func TestQueryPageLocatesUnicodeParserRangeInEditor(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "query", "default-source.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	text := "SELECT '💰', missing LIMIT 0"
	response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: text})
	if err != nil {
		t.Fatal(err)
	}
	if response.Error == nil || response.Error.Range == nil {
		t.Fatalf("query did not return a source range: %+v", response)
	}
	start := strings.Index(text, "missing")
	if response.Error.Range.Start != start || response.Error.Range.End != start+len("missing") {
		t.Fatalf("parser byte range = %+v, want [%d,%d)", response.Error.Range, start, start+len("missing"))
	}

	m := NewModel(true)
	m.page = pageQuery
	m.queryDraft = text
	m.workspaceState = workspace.State{
		Phase: workspace.PhaseReady, ActivePath: path,
		QueryResponse: &response,
	}
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("unknown-column") {
		t.Fatalf("typed parser error is not visible: %q", tester.Texts())
	}
	if err := tester.Click("定位到查询错误"); err != nil {
		t.Fatal(err)
	}
	tester.Type("account")
	want := "SELECT '💰', account LIMIT 0"
	if m.queryDraft != want {
		t.Fatalf("typing after error navigation produced %q, want %q", m.queryDraft, want)
	}
}

func TestQueryErrorNavigationDoesNotMoveMarkedInputComposition(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "query", "default-source.bean")
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	text := "SELECT '💰', missing LIMIT 0"
	response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: text})
	if err != nil {
		t.Fatal(err)
	}
	if response.Error == nil || response.Error.Range == nil {
		t.Fatalf("query did not return a source range: %+v", response)
	}

	model := NewModel(true)
	model.queryDraft = text
	navigateToError := false
	tester := ui.NewTester(func(c *ui.Context) {
		editor := ui.TextArea(c, &model.queryDraft).Label("BQL 查询").MinHeight(220).FillWidth()
		if navigateToError {
			model.locateQueryError(editor, &response)
		}
	}, 1180, 780)
	if err := tester.Click("BQL 查询"); err != nil {
		t.Fatal(err)
	}
	tester.Compose("組", 1)
	navigateToError = true
	tester.Frame()
	if model.queryDraft != text {
		t.Fatalf("error navigation replaced the marked query text: %q", model.queryDraft)
	}
	if model.status != "当前输入法组合尚未结束，错误定位已保留" {
		t.Fatalf("marked composition navigation status = %q", model.status)
	}
}

func TestReadingControlsValidateDatesAndApplyCoreRequest(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	controller := workspace.NewController("filter-window", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("workspace did not close")
		}
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: path, Options: workspace.OpenOptions{SemanticsProfile: ledger.V3}}); err != nil {
		t.Fatal(err)
	}
	waitWorkspace(t, controller, func(s workspace.State) bool { return s.Phase == workspace.PhaseReady })
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: requestForView("dashboard")}); err != nil {
		t.Fatal(err)
	}
	state := waitWorkspace(t, controller, func(s workspace.State) bool { return s.Response != nil })

	m := NewModel(true)
	m.workspace = controller
	m.workspaceState = state
	tester := ui.NewTester(m.View, 900, 600)
	for _, want := range []string{"开始日期", "结束日期", "应用筛选", "币种", "估值", "近1月"} {
		if !tester.HasText(want) {
			t.Errorf("reading controls do not render %q: %q", want, tester.Texts())
		}
	}
	for _, want := range []string{"应用筛选", "原币", "按成本", "近1月"} {
		if rect, ok := tester.Find(want); !ok || rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 900 || rect.Y+rect.H > 600 {
			t.Errorf("reading control %q is outside the 900×600 minimum window: %+v, found=%v", want, rect, ok)
		}
	}
	operation := controller.State().Operation
	if err := tester.Click("开始日期 YYYY-MM-DD"); err != nil {
		t.Fatal(err)
	}
	tester.Type("not-a-date")
	if err := tester.Click("应用筛选"); err != nil {
		t.Fatal(err)
	}
	if !tester.HasText("开始日期格式应为 YYYY-MM-DD") {
		t.Fatalf("invalid date was not explained: %q", tester.Texts())
	}
	if got := controller.State().Operation; got != operation {
		t.Fatalf("invalid date started a read operation: %d -> %d", operation, got)
	}

	m.filterDateFrom, m.filterDateTo = "2000-01-01", "2000-01-31"
	tester.Frame()
	if err := tester.Click("原币"); err != nil {
		t.Fatal(err)
	}
	if err := tester.Click("USD"); err != nil {
		t.Fatal(err)
	}
	if err := tester.Click("按成本"); err != nil {
		t.Fatal(err)
	}
	if err := tester.Click("按市价"); err != nil {
		t.Fatal(err)
	}
	if err := tester.Click("应用筛选"); err != nil {
		t.Fatal(err)
	}
	filtered := waitWorkspace(t, controller, func(s workspace.State) bool {
		return s.Response != nil && s.Response.Request.Currency == "USD" && s.Response.Request.Valuation == "market-value"
	})
	request := filtered.Response.Request
	if request.View != "dashboard" || request.DateFrom != "2000-01-01" || request.DateTo != "2000-01-31" || request.Currency != "USD" || request.Valuation != "market-value" {
		t.Fatalf("applied filter request = %+v", request)
	}
	if err := tester.Click("近1月"); err != nil {
		t.Fatal(err)
	}
	if err := tester.Click("应用筛选"); err != nil {
		t.Fatal(err)
	}
	period := waitWorkspace(t, controller, func(s workspace.State) bool {
		return s.Response != nil && s.Response.Request.Window == "1m" && s.Response.Request.Currency == "USD"
	})
	if request := period.Response.Request; request.DateFrom != "" || request.DateTo != "" {
		t.Fatalf("preset period did not clear custom dates: %+v", request)
	}
}

func TestDiagnosticSourceActionRevealsSourceInFinder(t *testing.T) {
	const sourcePath = "/tmp/ledger/include.bean"
	var revealed string
	m := NewModel(true)
	m.revealPath = func(path string) { revealed = path }
	m.workspaceState = workspace.State{
		Phase:      workspace.PhaseInvalid,
		ActivePath: "/tmp/ledger/main.bean",
		Response: &desktop.ViewResponse{Summary: desktop.SummaryPayload{Diagnostics: []desktop.DiagnosticDTO{{
			Code: "parse-error", Severity: "error", Message: "未知指令", Source: desktop.SourceLocation{Path: sourcePath, Line: 4, Column: 2},
		}}}},
	}
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("未知指令") || !tester.HasText("include.bean:4:2") {
		t.Fatalf("source diagnostic is missing its message or location: %q", tester.Texts())
	}
	if err := tester.Click("在 Finder 中显示来源"); err != nil {
		t.Fatal(err)
	}
	if revealed != sourcePath {
		t.Fatalf("Finder reveal path = %q, want %q", revealed, sourcePath)
	}
}

func TestDiagnosticsPageDistinguishesStaleAndCopiesCurrentLocation(t *testing.T) {
	const sourcePath = "/tmp/ledger/include.bean"
	var copied, revealed string
	m := NewModel(true)
	m.page = pageDiagnostics
	m.writeClipboard = func(value string) { copied = value }
	m.revealPath = func(path string) { revealed = path }
	m.workspaceState = workspace.State{
		Phase: workspace.PhaseStale, ActivePath: "/tmp/ledger/main.bean",
		Response: &desktop.ViewResponse{Summary: desktop.SummaryPayload{Diagnostics: []desktop.DiagnosticDTO{{
			Code: "missing-include", Severity: "error", Message: "找不到 include 文件",
			Source: desktop.SourceLocation{Path: sourcePath, Line: 4, Column: 2},
			Related: []desktop.RelatedLocation{{
				Source:  desktop.SourceLocation{Path: "/tmp/ledger/accounts.bean", Line: 9, Column: 4},
				Message: "由此处引用",
			}},
		}}}},
	}
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("上次有效快照") || !tester.HasText("找不到 include 文件") || !tester.HasText("include.bean:4:2") || !tester.HasText("accounts.bean:9:4") {
		t.Fatalf("stale diagnostic page lacks its phase or source details: %q", tester.Texts())
	}
	if err := tester.Click("复制位置"); err != nil {
		t.Fatal(err)
	}
	if copied != sourcePath+":4:2" {
		t.Fatalf("copied diagnostic location = %q", copied)
	}
	if err := tester.Click("复制关联位置"); err != nil {
		t.Fatal(err)
	}
	if copied != "/tmp/ledger/accounts.bean:9:4" {
		t.Fatalf("copied related diagnostic location = %q", copied)
	}
	if err := tester.Click("在 Finder 中显示来源"); err != nil {
		t.Fatal(err)
	}
	if revealed != sourcePath {
		t.Fatalf("Finder reveal path = %q", revealed)
	}
}

func TestGoUIRendersCurrentInvalidDiagnosticsAndSameWindowRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	valid := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Equity:Opening USD
2026-01-02 * "Opening"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`
	if err := os.WriteFile(path, []byte(valid), 0600); err != nil {
		t.Fatal(err)
	}
	controller := workspace.NewController("diagnostic-recovery-ui", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(2 * time.Second):
			t.Error("diagnostic recovery workspace did not close")
		}
	})
	var copiedLocation string
	newModel := func(state workspace.State, currentPage page) *Model {
		model := NewModel(true)
		model.workspace = controller
		model.workspaceState = state
		model.page = currentPage
		model.writeClipboard = func(value string) { copiedLocation = value }
		return model
	}
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentOpen, Path: path, Options: workspace.OpenOptions{
		SemanticsProfile: ledger.V3,
		Watch:            snapshot.WatchOptions{PollInterval: 10 * time.Millisecond, Debounce: 15 * time.Millisecond},
	}}); err != nil {
		t.Fatal(err)
	}
	opened := waitWorkspace(t, controller, func(state workspace.State) bool { return state.Phase == workspace.PhaseReady && !state.Busy })
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: requestForView("dashboard")}); err != nil {
		t.Fatal(err)
	}
	initial := waitWorkspace(t, controller, func(state workspace.State) bool {
		return !state.Busy && state.Response != nil && state.Response.State.Valid && state.Response.Summary.TransactionsCount == 1
	})
	model := newModel(initial, pageDashboard)
	initialUI := ui.NewTester(model.View, 1180, 780)
	if !initialUI.HasText("财务概览") || !initialUI.HasText("10 USD") {
		t.Fatalf("initial report was not rendered: %q", initialUI.Texts())
	}

	if err := os.WriteFile(path, []byte("this is not valid Beancount\n"), 0600); err != nil {
		t.Fatal(err)
	}
	stale := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Phase == workspace.PhaseStale && !state.Ledger.Reloading
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: requestForView("dashboard")}); err != nil {
		t.Fatal(err)
	}
	invalid := waitWorkspace(t, controller, func(state workspace.State) bool {
		return !state.Busy && state.Response != nil && !state.Response.State.Valid && state.Response.Summary.ErrorCount > 0
	})
	if invalid.Ledger.Revision < stale.Ledger.Revision || invalid.Response.State.SnapshotID != stale.Ledger.SnapshotID {
		t.Fatal("invalid diagnostics do not describe the current stale snapshot")
	}
	model = newModel(invalid, pageDiagnostics)
	diagnosticUI := ui.NewTester(model.View, 1180, 780)
	if !diagnosticUI.HasText("诊断") || !diagnosticUI.HasText("账本文件已有变化") || !diagnosticUI.HasText("main.bean:1:1") {
		t.Fatalf("current parse failure was not rendered with its source location: %q", diagnosticUI.Texts())
	}
	firstDiagnostic := invalid.Response.Summary.Diagnostics[0]
	if err := diagnosticUI.Click("复制位置"); err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("%s:%d:%d", firstDiagnostic.Source.Path, firstDiagnostic.Source.Line, firstDiagnostic.Source.Column); copiedLocation != want {
		t.Fatalf("copied first diagnostic location = %q, want %q", copiedLocation, want)
	}

	if err := os.WriteFile(path, []byte(valid+"\n\nthis is a different invalid ledger\n"), 0600); err != nil {
		t.Fatal(err)
	}
	secondStale := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Phase == workspace.PhaseStale && state.Ledger.Revision > invalid.Ledger.Revision && !state.Ledger.Reloading
	})
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: requestForView("dashboard")}); err != nil {
		t.Fatal(err)
	}
	secondInvalid := waitWorkspace(t, controller, func(state workspace.State) bool {
		return !state.Busy && state.Response != nil && !state.Response.State.Valid && state.Response.Summary.ErrorCount > 0 && state.Ledger.Revision >= secondStale.Ledger.Revision
	})
	secondDiagnostic := secondInvalid.Response.Summary.Diagnostics[0]
	if secondDiagnostic.Source.Line <= firstDiagnostic.Source.Line {
		t.Fatalf("second invalid revision did not move the source location: first=%+v second=%+v", firstDiagnostic, secondDiagnostic)
	}
	model = newModel(secondInvalid, pageDiagnostics)
	updatedDiagnosticUI := ui.NewTester(model.View, 1180, 780)
	newLocation := fmt.Sprintf("main.bean:%d:%d", secondDiagnostic.Source.Line, secondDiagnostic.Source.Column)
	if !updatedDiagnosticUI.HasText(newLocation) || updatedDiagnosticUI.HasText("main.bean:1:1") {
		t.Fatalf("diagnostics page did not replace the prior location with the newest revision: %q", updatedDiagnosticUI.Texts())
	}
	if err := updatedDiagnosticUI.Click("复制位置"); err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("%s:%d:%d", secondDiagnostic.Source.Path, secondDiagnostic.Source.Line, secondDiagnostic.Source.Column); copiedLocation != want {
		t.Fatalf("copied updated diagnostic location = %q, want %q", copiedLocation, want)
	}

	repaired := valid + `2026-01-03 * "Deposit"
  Assets:Cash 20 USD
  Equity:Opening -20 USD
`
	if err := os.WriteFile(path, []byte(repaired), 0600); err != nil {
		t.Fatal(err)
	}
	model = newModel(secondInvalid, pageDashboard)
	reloadUI := ui.NewTester(model.View, 1180, 780)
	if err := reloadUI.Click("重新读取"); err != nil {
		t.Fatalf("same-window reload action could not be used: %v", err)
	}
	readyAgain := waitWorkspace(t, controller, func(state workspace.State) bool {
		return state.Phase == workspace.PhaseReady && !state.Busy && !state.Ledger.Reloading && state.Response == nil
	})
	if readyAgain.ActivePath != opened.ActivePath || readyAgain.Epoch != opened.Epoch {
		t.Fatal("repair recovery replaced or detached the existing workspace")
	}
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: requestForView("dashboard")}); err != nil {
		t.Fatal(err)
	}
	recovered := waitWorkspace(t, controller, func(state workspace.State) bool {
		return !state.Busy && state.Response != nil && state.Response.State.Valid && state.Response.Summary.TransactionsCount == 2
	})
	model = newModel(recovered, pageDashboard)
	recoveredUI := ui.NewTester(model.View, 1180, 780)
	if !recoveredUI.HasText("财务概览") || !recoveredUI.HasText("30 USD") {
		t.Fatalf("repaired report was not rendered in the same window: %q", recoveredUI.Texts())
	}
}

func TestOpeningLedgerRestoresItsPreferencesBeforeFirstRead(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	store := preferences.NewStore(filepath.Join(t.TempDir(), "support", "preferences.json"))
	if err := store.UpdateLedger(path, func(item *preferences.LedgerPreferences) error {
		item.SemanticsProfile = "legacy"
		item.DateFrom, item.DateTo = "2000-01-01", "2000-01-31"
		item.Currency = "USD"
		item.Valuation = "market-value"
		item.Locale = "en"
		item.Window = "all"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	controller := workspace.NewController("restore-window", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("workspace did not close")
		}
	})
	m := NewModel(true)
	m.workspace = controller
	m.SetPreferences(store)
	m.openLedger(path)
	state := waitWorkspace(t, controller, func(s workspace.State) bool { return s.Phase == workspace.PhaseReady })
	m.applyPendingOpenState(state)
	m.workspaceState = state
	if m.pendingOpen || m.filterCurrency != "USD" || m.filterValuation != "market-value" || m.filterDateFrom != "2000-01-01" || m.filterDateTo != "2000-01-31" || m.locale != "en" {
		t.Fatalf("opening ledger did not restore its reading controls: pending=%v from=%s to=%s currency=%s valuation=%s locale=%s", m.pendingOpen, m.filterDateFrom, m.filterDateTo, m.filterCurrency, m.filterValuation, m.locale)
	}
	if state.Ledger.SemanticsProfile != ledger.Legacy {
		t.Fatalf("open used semantics profile %q, want legacy", state.Ledger.SemanticsProfile)
	}
	if err := controller.Dispatch(workspace.Intent{Kind: workspace.IntentRead, Request: m.requestForView("dashboard")}); err != nil {
		t.Fatal(err)
	}
	read := waitWorkspace(t, controller, func(s workspace.State) bool { return s.Response != nil })
	request := read.Response.Request
	if request.DateFrom != "2000-01-01" || request.DateTo != "2000-01-31" || request.Currency != "USD" || request.Valuation != "market-value" || request.Locale != "en" {
		t.Fatalf("first read did not use restored preferences: %+v", request)
	}
}

func TestDiagnosticLocaleSelectorPersistsAndFeedsEveryRead(t *testing.T) {
	ledgerPath := filepath.Join(t.TempDir(), "ledger.bean")
	if err := os.WriteFile(ledgerPath, []byte("option \"title\" \"test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := preferences.NewStore(filepath.Join(t.TempDir(), "support", "preferences.json"))
	if err := store.UpdateLedger(ledgerPath, func(item *preferences.LedgerPreferences) error {
		item.Locale = "zh-CN"
		item.DateFrom = "2000-01-01"
		item.Currency = "EUR"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	defer m.Close()
	m.SetPreferences(store)
	m.workspaceState.ActivePath = ledgerPath
	m.page = pageDiagnostics
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("界面语言") {
		t.Fatalf("active ledger window has no interface locale control: %q", tester.Texts())
	}
	if err := tester.Click("简体中文"); err != nil {
		t.Fatal(err)
	}
	if err := tester.Click("English"); err != nil {
		t.Fatal(err)
	}
	if m.requestLocale() != "en" {
		t.Fatalf("selected interface locale = %q, want en", m.requestLocale())
	}
	if !tester.HasText("Interface Language") || !tester.HasText("Workspace") || tester.HasText("界面语言") || tester.HasText("工作区") {
		t.Fatalf("English interface labels were not refreshed after changing locale: %q", tester.Texts())
	}
	m.preferenceWork.Wait()
	item, err := store.ForLedger(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if item.Locale != "en" || item.DateFrom != "2000-01-01" || item.Currency != "EUR" {
		t.Fatalf("persisted diagnostic locale overwrote another reading preference: %+v", item)
	}
	for _, view := range []string{
		"dashboard", "balance_sheet", "income_statement", "journal", "account",
		"prices", "holdings", "directives", "attachments", "migration", "query",
	} {
		request := m.requestForView(view)
		if request.Locale != "en" {
			t.Errorf("%s request locale = %q, want en", view, request.Locale)
		}
	}
}

func TestEnglishPresentationTranslatesApplicationLabelsAndPreservesLedgerValues(t *testing.T) {
	m := NewModel(true)
	defer m.Close()
	m.locale = "en"
	page := report.ProjectionPage{TotalCount: 101, Offset: 0, ReturnedCount: 100}
	if got, want := m.pageDescription(page, "个持仓分组"), "101 holding groups · Showing 1–100"; got != want {
		t.Fatalf("English page description = %q, want %q", got, want)
	}

	ledgerText := "月度收支"
	cell := query.Cell{Type: "string", Value: ledgerText}
	if got := queryCellTextWithLocale(cell, m.tr); got != ledgerText {
		t.Fatalf("query value was translated: %q", got)
	}
	if got, want := queryCellTextWithLocale(query.Cell{Type: "positions", Value: []query.PresentedPosition{}}, m.tr), "Inventory · 0 items"; got != want {
		t.Fatalf("English typed-cell summary = %q, want %q", got, want)
	}
	if got := strings.Join(queryCellDetailsWithLocale(query.Cell{Type: "positions", Value: []query.PresentedPosition{}}, m.tr), "\n"); got != "Empty Inventory" {
		t.Fatalf("English typed-cell details = %q, want Empty Inventory", got)
	}

	identity := desktop.ReportExportIdentity{
		ReturnedRows: 20, SourceRecordCount: 30, Truncated: true,
		DateFrom: "2024-01-01", DateTo: "2024-12-31",
		Currency: "USD", Valuation: "market-value",
		SemanticsProfile: "v3", Completeness: "typed", CaptureScope: "complete-projection",
	}
	message := reportExportPanelMessage(identity, "analysis", m.tr)
	for _, want := range []string{"Analysis file saves", "Complete Projection", "Result reached the item limit", "2024-01-01 to 2024-12-31", "Report basis USD / At Market Value", "v3"} {
		if !strings.Contains(message, want) {
			t.Errorf("English report save message %q does not contain %q", message, want)
		}
	}
	if strings.Contains(message, "完整投影") || strings.Contains(message, "保存") {
		t.Fatalf("English report save message contains untranslated app copy: %q", message)
	}

	chinese := NewModel(true)
	defer chinese.Close()
	message = reportExportPanelMessage(desktop.ReportExportIdentity{Valuation: "at-cost"}, "analysis", chinese.tr)
	if !strings.Contains(message, "报表口径 原币 / 按成本") {
		t.Errorf("native-currency statement context is missing from the save message: %q", message)
	}
	priceIdentity := desktop.ReportExportIdentity{
		Section: "history", PriceBase: "STK", PriceQuote: "USD",
		DateFrom: "2000-01-01", DateTo: "2000-01-31",
	}
	englishPriceMessage := reportExportPanelMessage(priceIdentity, "analysis", m.tr)
	if !strings.Contains(englishPriceMessage, "Quote pair STK/USD") || strings.Contains(englishPriceMessage, "Report basis") {
		t.Errorf("English price save message does not clearly identify its quote pair: %q", englishPriceMessage)
	}
	chinesePriceMessage := reportExportPanelMessage(priceIdentity, "analysis", chinese.tr)
	if !strings.Contains(chinesePriceMessage, "报价对 STK/USD") || strings.Contains(chinesePriceMessage, "报表口径") {
		t.Errorf("Chinese price save message does not clearly identify its quote pair: %q", chinesePriceMessage)
	}
	pricePath := reportExportDefaultPath(filepath.Join(t.TempDir(), "ledger.bean"), priceIdentity)
	if !strings.Contains(filepath.Base(pricePath), "STK-USD") {
		t.Errorf("price export default filename omits the selected pair: %q", pricePath)
	}
	otherPricePath := reportExportDefaultPath(filepath.Join(t.TempDir(), "ledger.bean"), desktop.ReportExportIdentity{
		Section: "history", PriceBase: "EUR", PriceQuote: "USD",
		DateFrom: "2000-01-01", DateTo: "2000-01-31",
	})
	if filepath.Base(pricePath) == filepath.Base(otherPricePath) {
		t.Errorf("different price pairs share a default export filename: %q", pricePath)
	}
	queryMessage := queryExportPanelMessage(querySaveRequest{
		format: "analysis", returnedRows: 10, rowLimit: 100, truncated: true,
		context: desktop.QueryContext{SnapshotID: "snapshot-42", SemanticsProfile: "v3"},
	}, m.tr)
	for _, want := range []string{"Analysis file saves", "10/100 rows", "Result reached the row limit", "snapshot-42", "v3"} {
		if !strings.Contains(queryMessage, want) {
			t.Errorf("English query save message %q does not contain %q", queryMessage, want)
		}
	}
}

func TestFindFocusMatrixAtMinimumWindowAndLargeTextScale(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "account-detail.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}

	m := NewModel(true)
	defer m.Close()
	m.locale = "en"
	m.accountName, m.accountScope = "Assets:Test:Child", "direct"
	m.filterDateFrom, m.filterDateTo = "2000-01-03", "2000-01-06"
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: state}
	tests := []struct {
		page  page
		view  string
		focus string
	}{
		{pageBalance, "balance_sheet", "Search Accounts"},
		{pageIncome, "income_statement", "Search Accounts"},
		{pageJournal, "journal", "Search Journal"},
		{pageAccount, "account", "Search Account Journal"},
		{pageHoldings, "holdings", "Holdings Account"},
		{pageDirectives, "directives", "Search Directives"},
		{pageQuery, "query", "Search Ledger Queries"},
	}
	for _, test := range tests {
		m.page = test.page
		response, err := session.Read(m.requestForView(test.view))
		if err != nil {
			t.Fatalf("read %s page: %v", test.view, err)
		}
		m.workspaceState.Response = &response
		m.searchFocusPending = false
		m.focusCurrentSearch()
		tester := ui.NewTester(m.View, 900, 600)
		tester.SetPreferences(ui.Preferences{TextScale: 1.5})
		tester.Frame()
		if !tester.Focused(test.focus) {
			t.Errorf("Find did not focus %q on %s at 900×600/1.5: %q", test.focus, test.view, tester.Texts())
		}
	}
}

func TestLocalizedSelectPreservesDynamicValuesThatMatchUIStrings(t *testing.T) {
	m := NewModel(true)
	m.locale = "en"
	selected := "财务概览"
	options := []string{"全部账户", "财务概览"}
	tester := ui.NewTester(func(c *ui.Context) {
		m.selectOptions(c, &selected, options, map[string]string{"全部账户": m.tr("全部账户")}).Label("Account").Changed()
	}, 420, 300)
	if !tester.HasText("财务概览") || tester.HasText("Overview") {
		t.Fatalf("dynamic ledger option was translated: %q", tester.Texts())
	}
	if err := tester.Click("财务概览"); err != nil {
		t.Fatal(err)
	}
	if err := tester.Click("All Accounts"); err != nil {
		t.Fatalf("select translated sentinel option: %v; UI=%q", err, tester.Texts())
	}
	if selected != "全部账户" {
		t.Fatalf("selected option stored presentation label %q instead of source value", selected)
	}
}

func TestOpeningLedgerKeepsLegacyDefaultSemantics(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	controller := workspace.NewController("legacy-default-window", workspace.DesktopEngine{}, nil)
	t.Cleanup(func() {
		controller.Close()
		select {
		case <-controller.Done():
		case <-time.After(time.Second):
			t.Error("workspace did not close")
		}
	})
	m := NewModel(true)
	m.workspace = controller
	m.openLedger(path)
	state := waitWorkspace(t, controller, func(s workspace.State) bool { return s.Phase == workspace.PhaseReady })
	if state.Ledger.SemanticsProfile != ledger.Legacy {
		t.Fatalf("new Go UI ledger profile = %q, want unchanged SwiftUI default %q", state.Ledger.SemanticsProfile, ledger.Legacy)
	}
}

func TestWindowCloseWaitsForReadingPreferenceWrite(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "cashflow.bean")
	store := preferences.NewStore(filepath.Join(t.TempDir(), "support", "preferences.json"))
	m := NewModel(true)
	m.SetPreferences(store)
	m.workspaceState = workspace.State{ActivePath: path, Ledger: desktop.State{SemanticsProfile: ledger.V3}}
	m.filterWindow = "1y"
	m.filterCurrency = "USD"
	m.filterValuation = "market-value"
	m.persistReadingPreferences()
	m.Close()
	item, err := store.ForLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	if item.Window != "1y" || item.Currency != "USD" || item.Valuation != "market-value" {
		t.Fatalf("window closed before its reading preferences were durable: %+v", item)
	}
}

func TestPricesPageRendersCatalogAndExactSourceOrderedHistory(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "prices.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page = pagePrices
	m.priceBase = "STK"
	m.workspaceState = workspace.State{
		Phase: workspace.PhaseReady, ActivePath: path, Ledger: desktop.State{SnapshotID: state.SnapshotID},
	}
	catalog, err := session.Read(m.requestForView("prices"))
	if err != nil {
		t.Fatal(err)
	}
	m.workspaceState.Response = &catalog
	var revealed string
	m.revealPath = func(value string) { revealed = value }
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"商品目录", "STK", "UNQUOTED", "DECLAREDONLY", "报价对", "价格数", "商品声明", "显示商品声明来源"} {
		if !tester.HasText(want) {
			t.Errorf("commodity catalog does not render %q: %q", want, tester.Texts())
		}
	}
	if err := tester.Click("显示商品声明来源"); err != nil {
		t.Fatalf("click commodity declaration source: %v; UI=%q", err, tester.Texts())
	}
	if revealed != path {
		t.Fatalf("commodity declaration source path = %q, want %q", revealed, path)
	}
	m.commoditySearchDraft = "DECLAREDONLY"
	tester = ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("筛选商品"); err != nil {
		t.Fatalf("apply commodity catalog search: %v; UI=%q", err, tester.Texts())
	}
	if m.commoditySearch != "DECLAREDONLY" {
		t.Fatalf("applied commodity search = %q", m.commoditySearch)
	}
	searchResponse, err := session.Read(m.requestForView("prices"))
	if err != nil {
		t.Fatal(err)
	}
	if searchResponse.Prices.CommoditiesPage.TotalCount != 1 || len(searchResponse.Prices.Commodities) != 1 || searchResponse.Prices.Commodities[0].Currency != "DECLAREDONLY" {
		t.Fatalf("catalog search did not filter the core projection: %+v", searchResponse.Prices)
	}
	m.workspaceState.Response = &searchResponse
	tester = ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("DECLAREDONLY") || tester.HasText("UNQUOTED") {
		t.Fatalf("catalog search did not update visible rows: %q", tester.Texts())
	}
	if err := tester.Click("清除商品搜索"); err != nil || m.commoditySearch != "" {
		t.Fatalf("clear commodity search: err=%v search=%q", err, m.commoditySearch)
	}
	catalog, err = session.Read(m.requestForView("prices"))
	if err != nil {
		t.Fatal(err)
	}
	m.workspaceState.Response = &catalog
	tester = ui.NewTester(m.View, 1180, 780)
	selectedCommodity := m.commoditySelectedID
	baseAfterClear := m.priceBase
	if err := tester.Click("价格数"); err != nil {
		t.Fatalf("sort commodity catalog by typed price count: %v; UI=%q", err, tester.Texts())
	}
	tester.Frame()
	if m.commoditySort.Column != "prices" || m.commoditySelectedID != selectedCommodity || m.priceBase != baseAfterClear {
		t.Fatalf("sorting changed commodity selection: sort=%+v selected=%q base=%q", m.commoditySort, m.commoditySelectedID, m.priceBase)
	}

	m.priceSection = "history"
	m.priceBase, m.priceQuote = "STK", "USD"
	m.priceDescending = false
	m.filterDateTo = "2000-01-20"
	history, err := session.Read(m.requestForView("prices"))
	if err != nil {
		t.Fatal(err)
	}
	if history.Prices == nil || len(history.Prices.Records) != 4 {
		t.Fatalf("STK/USD source history has %d records, want four: %+v", len(history.Prices.Records), history.Prices)
	}
	for i := 1; i < len(history.Prices.Records); i++ {
		if history.Prices.Records[i-1].SourceOrder >= history.Prices.Records[i].SourceOrder {
			t.Fatalf("source order changed in ascending history: %+v", history.Prices.Records)
		}
	}
	if !strings.HasSuffix(history.Prices.Records[0].Source.Path, "prices-included.bean") || history.Prices.Selected == nil || history.Prices.Selected.Rate.Exact != "99" {
		t.Fatalf("history lost included-source ordering or cutoff quote: records=%+v selected=%+v", history.Prices.Records, history.Prices.Selected)
	}
	m.workspaceState.Response = &history
	tester = ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"价格历史", "截至 2000-01-20", "99 USD", "10 USD", "11 USD", "prices-included.bean"} {
		if !tester.HasText(want) {
			t.Errorf("price history does not render %q: %q", want, tester.Texts())
		}
	}
	if err := tester.Click("显示所选报价来源"); err != nil {
		t.Fatalf("click selected quote source: %v; UI=%q", err, tester.Texts())
	}
	if revealed != path {
		t.Fatalf("selected quote source path = %q, want %q", revealed, path)
	}
}

func TestPricesRequestKeepsCatalogAndHistoryPagingIndependent(t *testing.T) {
	m := NewModel(true)
	m.priceSection = "commodities"
	m.commoditySearch = "STK"
	m.commodityOffset, m.priceOffset = 200, 400
	request := m.requestForView("prices")
	if request.PriceSection != "commodities" || request.Offset != 200 || request.PriceBase != "" || request.PriceQuote != "" || request.CommoditySearch != "STK" {
		t.Fatalf("commodity request inherited history selection or offset: %+v", request)
	}
	m.priceSection = "history"
	m.priceBase, m.priceQuote = "STK", "USD"
	request = m.requestForView("prices")
	if request.PriceSection != "history" || request.Offset != 400 || request.PriceBase != "STK" || request.PriceQuote != "USD" || request.CommoditySearch != "" {
		t.Fatalf("history request lost pair or used catalog offset: %+v", request)
	}
}

func TestPriceChartPointsRetainCoreSegmentsAndExactUnplottableRecords(t *testing.T) {
	first, last := 10.0, 20.0
	points := priceChartPoints([]report.PriceRecord{
		{Date: "2000-01-01", Quote: "USD", Rate: report.PresentedDecimal{Exact: "10"}, Value: &first, Segment: 0},
		{Date: "2000-01-02", Quote: "USD", Rate: report.PresentedDecimal{Exact: "1e400"}, Value: nil, Segment: 0},
		{Date: "2000-01-03", Quote: "USD", Rate: report.PresentedDecimal{Exact: "20"}, Value: &last, Segment: 1},
	})
	if len(points) != 3 || points[0].Segment != 0 || points[1].Available || points[1].Exact != "1e400 USD" || points[2].Segment != 1 || !points[2].Available {
		t.Fatalf("price chart lost source segments or exact unplottable value: %+v", points)
	}
}

func TestPricesPageKeepsZeroAndUnplottableValuesVisible(t *testing.T) {
	huge := "1" + strings.Repeat("0", 400)
	path := filepath.Join(t.TempDir(), "price-values.bean")
	data := "2000-01-01 commodity STK\n2000-01-02 price STK " + huge + " USD\n2000-01-03 price STK 0 USD\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page, m.priceSection, m.priceBase, m.priceQuote = pagePrices, "history", "STK", "USD"
	m.filterDateTo = "2000-01-03"
	m.priceDescending = false
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: state}
	response, err := session.Read(m.requestForView("prices"))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Prices.Records) != 2 || response.Prices.Records[0].Value != nil || response.Prices.Records[1].Value == nil || *response.Prices.Records[1].Value != 0 || response.Prices.Records[0].Segment == response.Prices.Records[1].Segment {
		t.Fatalf("zero/unplottable source points = %+v", response.Prices.Records)
	}
	m.workspaceState.Response = &response
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{huge[:32], "0 USD", "2000-01-02", "2000-01-03", "来源段", "1", "2"} {
		if !tester.HasText(want) {
			t.Errorf("price history UI lost %q: %q", want, tester.Texts())
		}
	}
}

func TestHoldingsPagesRenderExactInventoryEventsAverageHistoryAndLots(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "average-history.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	workspaceState := workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: desktop.State{SnapshotID: state.SnapshotID}}
	newModel := func(section string) *Model {
		model := NewModel(true)
		model.page = pageHoldings
		model.holdingSection = section
		model.holdingAccount, model.holdingAccountDraft = "Assets:Stock", "Assets:Stock"
		model.holdingScope, model.holdingGroupBy, model.holdingDescending = "subtree", "account", false
		model.filterDateTo = "2000-01-08"
		model.workspaceState = workspaceState
		return model
	}
	m := newModel("groups")
	read := func() *desktop.ViewResponse {
		t.Helper()
		response, err := session.Read(m.requestForView("holdings"))
		if err != nil {
			t.Fatalf("read %s holdings: %v", m.holdingSection, err)
		}
		if response.Holdings == nil {
			t.Fatalf("missing %s holdings projection", m.holdingSection)
		}
		m.workspaceState.Response = &response
		return &response
	}

	groups := read()
	if len(groups.Holdings.Groups) != 1 || groups.Holdings.Groups[0].Quantity.Exact != "5" || groups.Holdings.Groups[0].CostBasis.Exact != "40" || groups.Holdings.Groups[0].AverageCost.Exact != "8" {
		t.Fatalf("exact current inventory = %+v", groups.Holdings.Groups)
	}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"持仓与成本", "5 STK", "40 USD", "8 USD", "不可用", "缺少本地报价", "查看当前批次"} {
		if !tester.HasText(want) {
			t.Errorf("holding groups do not render %q: %q", want, tester.Texts())
		}
	}
	if tester.HasText("unavailable") || tester.HasText("missing-local-quote") {
		t.Fatalf("holding valuation exposed internal codes: %q", tester.Texts())
	}
	if err := tester.Click("查看当前批次"); err != nil {
		t.Fatalf("open current lots: %v; UI=%q", err, tester.Texts())
	}
	groupID := m.holdingGroupID
	m = newModel("lots")
	m.holdingGroupID = groupID
	lots := read()
	if len(lots.Holdings.Groups) != 1 || len(lots.Holdings.Groups[0].Lots) != 1 || lots.Holdings.Groups[0].Lots[0].Quantity.Exact != "5" {
		t.Fatalf("current lot detail = %+v", lots.Holdings.Groups)
	}
	tester = ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"当前批次", "2000-01-08", "5 STK", "显示批次来源"} {
		if !tester.HasText(want) {
			t.Errorf("current lot detail does not render %q: %q", want, tester.Texts())
		}
	}

	m = newModel("events")
	events := read()
	if len(events.Holdings.Events) != 3 || events.Holdings.Events[0].GrossGain == nil || events.Holdings.Events[0].GrossGain.Exact != "400/3" {
		t.Fatalf("realized events lost exact sale economics: %+v", events.Holdings.Events)
	}
	tester = ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"已实现", "First sale", "400/3 USD", "查看消耗批次"} {
		if !tester.HasText(want) {
			t.Errorf("realized events do not render %q: %q", want, tester.Texts())
		}
	}
	if err := tester.Click("查看消耗批次"); err != nil {
		t.Fatalf("open consumed lots: %v; UI=%q", err, tester.Texts())
	}
	eventID := m.holdingEventID
	m = newModel("lots")
	m.holdingEventID, m.holdingLotKind = eventID, "consumed"
	eventLots := read()
	if len(eventLots.Holdings.Events) != 1 || eventLots.Holdings.Events[0].LotsPage.TotalCount == 0 {
		t.Fatalf("reduction lot details missing: %+v", eventLots.Holdings.Events)
	}
	tester = ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("First sale") || !tester.HasText("被消耗批次") {
		t.Fatalf("reduction lot detail missing event identity: %q", tester.Texts())
	}

	m = newModel("events")
	events = read()
	if events.Holdings.Events[0].MergeLotsPage.TotalCount != 2 {
		t.Fatalf("expected the AVERAGE source event to retain two merge lots: %+v", events.Holdings.Events[0].MergeLotsPage)
	}
	tester = ui.NewTester(m.View, 1180, 780)
	if err := tester.Click("查看合并来源"); err != nil {
		t.Fatalf("open merged contribution lots: %v; UI=%q", err, tester.Texts())
	}
	eventID = m.holdingEventID
	m = newModel("lots")
	m.holdingEventID, m.holdingLotKind = eventID, "merged"
	merged := read()
	if len(merged.Holdings.Events) != 1 || len(merged.Holdings.Events[0].MergeLots) != 2 {
		t.Fatalf("merged source lots = %+v", merged.Holdings.Events)
	}
	tester = ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"合并前来源", "10 USD", "20 USD", "posting"} {
		if !tester.HasText(want) {
			t.Errorf("merged source lots do not render %q: %q", want, tester.Texts())
		}
	}

	m = newModel("history")
	history := read()
	if len(history.Holdings.AverageHistory) != 1 || history.Holdings.AverageHistory[0].PointsPage.TotalCount != 7 {
		t.Fatalf("average history points = %+v", history.Holdings.AverageHistory)
	}
	tester = ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("查看完整曲线") || !tester.HasText("8 USD") {
		t.Fatalf("average history list missing: %q", tester.Texts())
	}
	if err := tester.Click("查看完整曲线"); err != nil {
		t.Fatalf("open average history: %v; UI=%q", err, tester.Texts())
	}
	historyID := m.holdingGroupID
	m = newModel("history")
	m.holdingGroupID = historyID
	series := read()
	if len(series.Holdings.AverageHistory) != 1 || len(series.Holdings.AverageHistory[0].Points) != 7 {
		t.Fatalf("selected average history = %+v", series.Holdings.AverageHistory)
	}
	tester = ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"STK", "10 USD", "50/3 USD", "空仓", "8 USD"} {
		if !tester.HasText(want) {
			t.Errorf("average history curve does not render %q: %q", want, tester.Texts())
		}
	}
}

func TestHoldingsRequestCarriesSelectedIdentityFiltersAndSnapshot(t *testing.T) {
	m := NewModel(true)
	m.holdingSection, m.holdingAccount, m.holdingScope = "lots", "Assets:Stock", "direct"
	m.holdingGroupBy, m.holdingGroupID, m.holdingLotKind = "cost_currency", "group-id", "merged"
	m.holdingOffset, m.holdingDescending = 100, true
	m.workspaceState.Ledger.SnapshotID = "holding-snapshot"
	m.filterDateFrom, m.filterDateTo, m.filterWindow = "2000-01-01", "2000-01-31", "all"
	m.filterCurrency, m.filterValuation = "USD", "market-value"
	request := m.requestForView("holdings")
	if request.View != "holdings" || request.HoldingSection != "lots" || request.Account != "Assets:Stock" || request.Scope != "direct" || request.GroupBy != "cost_currency" || request.GroupID != "group-id" || request.EventID != "" || request.LotKind != "" || request.Sort != "descending" || request.Offset != 100 || request.Limit != 100 || request.SnapshotID != "holding-snapshot" || request.DateFrom != "2000-01-01" || request.DateTo != "2000-01-31" || request.Currency != "USD" || request.Valuation != "market-value" {
		t.Fatalf("holding request omitted scope, identity, filter, paging, or snapshot: %+v", request)
	}
	m.holdingGroupID, m.holdingEventID, m.holdingLotKind = "", "event-id", "consumed"
	request = m.requestForView("holdings")
	if request.GroupID != "" || request.EventID != "event-id" || request.LotKind != "consumed" {
		t.Fatalf("event lot request mixed group and event identities: %+v", request)
	}
}

func TestHoldingsPageKeepsDirectoryAndEmptyAverageHistoryHonest(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "account-detail.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	newModel := func(section, account, scope string) *Model {
		model := NewModel(true)
		model.page, model.holdingSection = pageHoldings, section
		model.holdingAccount, model.holdingAccountDraft = account, account
		model.holdingScope, model.holdingGroupBy = scope, "account"
		model.holdingDescending = false
		model.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: state}
		return model
	}

	directoryModel := newModel("groups", "Assets", "direct")
	directory, err := session.Read(directoryModel.requestForView("holdings"))
	if err != nil {
		t.Fatal(err)
	}
	if len(directory.Holdings.Groups) != 0 {
		t.Fatalf("directory account direct holdings included child postings: %+v", directory.Holdings.Groups)
	}
	directoryModel.workspaceState.Response = &directory
	directoryTester := ui.NewTester(directoryModel.View, 1180, 780)
	if !directoryTester.HasText("当前日期与账户范围内没有剩余持仓") || directoryTester.HasText("Assets:Test") {
		t.Fatalf("directory account empty state invented direct inventory: %q", directoryTester.Texts())
	}

	historyModel := newModel("history", "Assets:Test", "subtree")
	history, err := session.Read(historyModel.requestForView("holdings"))
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Holdings.AverageHistory) != 0 {
		t.Fatalf("non-AVERAGE account invented an average-cost series: %+v", history.Holdings.AverageHistory)
	}
	historyModel.workspaceState.Response = &history
	historyTester := ui.NewTester(historyModel.View, 1180, 780)
	if !historyTester.HasText("当前范围没有按成本计价的平均成本历史") || historyTester.HasText("查看完整曲线") || historyTester.HasText("所选平均成本序列已不在当前快照中") {
		t.Fatalf("empty average history state is misleading: %q", historyTester.Texts())
	}
}

func TestHoldingsPageShowsAttributedFeesAndConservativeMissingFees(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "holdings-trades.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		account string
		texts   []string
	}{{"Assets:Fees", []string{"attributed fee", "100 USD", "5 USD", "95 USD"}}, {"Assets:Unassigned", []string{"unassigned fee", "100 USD", "费用没有明确关联到该账户"}}} {
		response, err := session.Read(desktop.Request{View: "holdings", HoldingSection: "events", Account: want.account, Scope: "subtree", GroupBy: "account", Sort: "ascending", Limit: 100, SnapshotID: state.SnapshotID, Window: "all", Valuation: "at-cost", Locale: "zh-CN"})
		if err != nil {
			t.Fatal(err)
		}
		if len(response.Holdings.Events) != 1 {
			t.Fatalf("%s events = %+v", want.account, response.Holdings.Events)
		}
		m := NewModel(true)
		m.page, m.holdingSection = pageHoldings, "events"
		m.holdingAccount, m.holdingAccountDraft, m.holdingScope = want.account, want.account, "subtree"
		m.holdingGroupBy, m.holdingDescending = "account", false
		m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: response.State, Response: &response}
		tester := ui.NewTester(m.View, 1180, 780)
		for _, text := range want.texts {
			if !tester.HasText(text) {
				t.Errorf("%s event page does not render %q: %q", want.account, text, tester.Texts())
			}
		}
		if want.account == "Assets:Fees" && response.Holdings.Events[0].Fees.Exact != "5" {
			t.Fatal("explicit fee should be reported as an exact expense")
		}
		if want.account == "Assets:Unassigned" && (response.Holdings.Events[0].Fees != nil || response.Holdings.Events[0].NetGain != nil) {
			t.Fatal("unassigned expense must not become an attributed fee")
		}
	}
}

func TestHoldingsPageRevealsActualMarketQuoteSource(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "history.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	request := desktop.Request{View: "holdings", HoldingSection: "groups", Account: "Assets:Stock", Scope: "subtree", GroupBy: "account", Sort: "ascending", Limit: 100, SnapshotID: state.SnapshotID, DateTo: "2000-02-29", Window: "all", Currency: "USD", Valuation: "market-value", Locale: "zh-CN"}
	response, err := session.Read(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Holdings.Groups) != 1 || response.Holdings.Groups[0].UnrealizedGain == nil || response.Holdings.Groups[0].UnrealizedGain.Exact != "100" || len(response.Holdings.Groups[0].MarketValuation.Quotes) != 1 {
		t.Fatalf("market valuation is not the expected local quote: %+v", response.Holdings.Groups)
	}
	m := NewModel(true)
	m.page, m.holdingSection = pageHoldings, "groups"
	m.holdingAccount, m.holdingAccountDraft, m.holdingScope, m.holdingGroupBy = "Assets:Stock", "Assets:Stock", "subtree", "account"
	m.holdingDescending = false
	m.filterDateTo, m.filterCurrency, m.filterValuation = "2000-02-29", "USD", "market-value"
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: response.State, Response: &response}
	var revealed string
	m.revealPath = func(value string) { revealed = value }
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("2000-02-15 20 USD") || !tester.HasText("显示估值报价来源") {
		t.Fatalf("market quote source missing: %q", tester.Texts())
	}
	if err := tester.Click("显示估值报价来源"); err != nil {
		t.Fatalf("reveal market quote source: %v; UI=%q", err, tester.Texts())
	}
	if revealed != path {
		t.Fatalf("market quote Finder path = %q, want %q", revealed, path)
	}
}

func TestHoldingsPageShowsPlainAndShortInventoryWithoutInventedCost(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "holdings-identities.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	request := desktop.Request{View: "holdings", HoldingSection: "groups", Scope: "subtree", GroupBy: "account", Sort: "ascending", Limit: 100, SnapshotID: state.SnapshotID, Window: "all", Valuation: "at-cost", Locale: "zh-CN"}
	response, err := session.Read(request)
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page, m.holdingSection = pageHoldings, "groups"
	m.holdingGroupBy, m.holdingScope, m.holdingDescending = "account", "subtree", false
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: response.State, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"Assets:Plain", "Assets:Short", "7 STK", "-3 STK", "无成本库存", "空头", "—"} {
		if !tester.HasText(want) {
			t.Errorf("plain/short inventory view does not render %q: %q", want, tester.Texts())
		}
	}
}

func TestHoldingsPageKeepsCostCurrenciesInSeparateExactGroups(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "holdings-unknowns.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	request := desktop.Request{
		View: "holdings", HoldingSection: "groups", Account: "Assets", Scope: "subtree", GroupBy: "cost_currency", Sort: "ascending",
		Limit: 100, SnapshotID: state.SnapshotID, DateTo: "2000-01-02", Window: "all", Valuation: "at-cost", Locale: "zh-CN",
	}
	response, err := session.Read(request)
	if err != nil {
		t.Fatal(err)
	}
	basisByCurrency := map[string]string{}
	for _, group := range response.Holdings.Groups {
		if group.Commodity != "STK" {
			continue
		}
		if group.CostCurrency == nil || group.CostBasis == nil {
			t.Fatalf("cost currency group lost its exact basis: %+v", group)
		}
		basisByCurrency[*group.CostCurrency] = group.CostBasis.Exact
	}
	if len(basisByCurrency) != 2 || basisByCurrency["USD"] != "100" || basisByCurrency["EUR"] != "60" {
		t.Fatalf("cost bases were mixed or changed: %v", basisByCurrency)
	}
	m := NewModel(true)
	m.page, m.holdingSection, m.holdingGroupBy = pageHoldings, "groups", "cost_currency"
	m.holdingAccount, m.holdingAccountDraft = "Assets", "Assets"
	m.holdingScope, m.holdingDescending = "subtree", false
	m.filterDateTo = "2000-01-02"
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: response.State, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"10 STK", "100 USD", "3 STK", "60 EUR"} {
		if !tester.HasText(want) {
			t.Errorf("cost currency page does not show %q: %q", want, tester.Texts())
		}
	}
	if tester.HasText("160") {
		t.Fatalf("cost currencies were visually totaled as if they shared a unit: %q", tester.Texts())
	}
}

func TestHoldingsPageShowsInverseQuoteAndRejectsCrossRateGuess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "holdings-quotes.bean")
	data := `option "operating_currency" "USD"
2000-01-01 commodity STK
2000-01-01 commodity GLD
2000-01-01 open Assets:Inverse STK "FIFO"
2000-01-01 open Assets:Cross GLD "FIFO"
2000-01-01 open Assets:Cash EUR, USD
2000-01-01 open Equity:Opening EUR, USD
2000-01-02 * "inverse quote holding"
  Assets:Inverse 3 STK {10 EUR}
  Assets:Cash -30 EUR
2000-01-02 * "cross rate only"
  Assets:Cross 4 GLD {5 USD}
  Equity:Opening -20 USD
2000-01-02 price USD 0.5 STK
2000-01-02 price GLD 2 EUR
2000-01-02 price EUR 2 USD
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	request := desktop.Request{
		View: "holdings", HoldingSection: "groups", Scope: "subtree", GroupBy: "account", Sort: "ascending",
		Limit: 100, SnapshotID: state.SnapshotID, DateTo: "2000-01-02", Window: "all", Currency: "USD", Valuation: "market-value", Locale: "zh-CN",
	}
	response, err := session.Read(request)
	if err != nil {
		t.Fatal(err)
	}
	var inverse, cross *report.HoldingGroup
	for i := range response.Holdings.Groups {
		group := &response.Holdings.Groups[i]
		switch group.Commodity {
		case "STK":
			inverse = group
		case "GLD":
			cross = group
		}
	}
	if inverse == nil || inverse.MarketValuation == nil || inverse.MarketValuation.CompleteTotal == nil || inverse.MarketValuation.CompleteTotal.Exact != "6" || len(inverse.MarketValuation.Quotes) != 1 || !inverse.MarketValuation.Quotes[0].Inverse {
		t.Fatalf("inverse local quote was not applied exactly: %+v", inverse)
	}
	if inverse.CostValuation == nil || inverse.CostValuation.CompleteTotal == nil || inverse.CostValuation.CompleteTotal.Exact != "60" {
		t.Fatalf("local EUR/USD cost conversion is not exact: %+v", inverse.CostValuation)
	}
	if cross == nil || cross.MarketValuation == nil || cross.MarketValuation.Completeness != "unavailable" || len(cross.MarketValuation.Missing) != 1 || len(cross.MarketValuation.Quotes) != 0 {
		t.Fatalf("multi-hop cross rate was guessed or not explained: %+v", cross)
	}

	m := NewModel(true)
	m.page, m.holdingSection, m.holdingGroupBy = pageHoldings, "groups", "account"
	m.holdingScope, m.holdingDescending, m.holdingSelectedID = "subtree", false, inverse.ID
	m.filterDateTo, m.filterCurrency, m.filterValuation = "2000-01-02", "USD", "market-value"
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: response.State, Response: &response}
	var revealed string
	m.revealPath = func(value string) { revealed = value }
	tester := ui.NewTester(m.View, 1180, 780)
	for _, want := range []string{"60 USD", "6 USD", "2000-01-02 2 USD", "反向报价", "显示估值报价来源"} {
		if !tester.HasText(want) {
			t.Errorf("inverse market quote detail omits %q: %q", want, tester.Texts())
		}
	}
	expectedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := tester.Click("显示估值报价来源"); err != nil || revealed != expectedPath {
		t.Fatalf("inverse quote source action = %q, err=%v, want %q", revealed, err, expectedPath)
	}
	m.holdingSelectedID = cross.ID
	tester = ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("缺少本地报价") || tester.HasText("16 USD") {
		t.Fatalf("UI displayed a fabricated cross-rate valuation: %q", tester.Texts())
	}
}

func TestHoldingsLotsPageReadsBeyondTheHundredLotPreview(t *testing.T) {
	var input strings.Builder
	input.WriteString("option \"operating_currency\" \"USD\"\n2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Stock STK \"FIFO\"\n2000-01-01 open Income:Gains USD\n2000-01-02 * \"many lots\"\n")
	for i := 0; i < 101; i++ {
		fmt.Fprintf(&input, "  Assets:Stock 1 STK {1 USD, \"lot-%03d\"}\n", i)
	}
	input.WriteString("  Assets:Cash -101 USD\n")
	path := filepath.Join(t.TempDir(), "lots.bean")
	if err := os.WriteFile(path, []byte(input.String()), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	groups, err := session.Read(desktop.Request{View: "holdings", HoldingSection: "groups", Account: "Assets:Stock", Scope: "subtree", GroupBy: "account", Sort: "ascending", Limit: 100, SnapshotID: state.SnapshotID, Window: "all", Valuation: "at-cost", Locale: "zh-CN"})
	if err != nil || groups.Holdings == nil || len(groups.Holdings.Groups) != 1 {
		t.Fatalf("read large holding group: err=%v projection=%+v", err, groups.Holdings)
	}
	groupID := groups.Holdings.Groups[0].ID
	m := NewModel(true)
	m.page, m.holdingSection = pageHoldings, "lots"
	m.holdingAccount, m.holdingAccountDraft, m.holdingScope, m.holdingGroupBy = "Assets:Stock", "Assets:Stock", "subtree", "account"
	m.holdingGroupID = groupID
	m.holdingDescending = false
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: state}
	first, err := session.Read(m.requestForView("holdings"))
	if err != nil || first.Holdings.Groups[0].LotsPage.TotalCount != 101 || len(first.Holdings.Groups[0].Lots) != 100 || !first.Holdings.Groups[0].LotsPage.HasMore {
		t.Fatalf("first lot page: err=%v projection=%+v", err, first.Holdings)
	}
	m.workspaceState.Response = &first
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("101个批次 · 显示 1–100") {
		t.Fatalf("first page count missing: %q", tester.Texts())
	}
	if err := tester.Click("下一页"); err != nil {
		t.Fatalf("advance lot page: %v; UI=%q", err, tester.Texts())
	}
	if m.holdingOffset != 100 {
		t.Fatalf("next lot action selected offset %d, want 100", m.holdingOffset)
	}
	last, err := session.Read(m.requestForView("holdings"))
	if err != nil || len(last.Holdings.Groups[0].Lots) != 1 || last.Holdings.Groups[0].LotsPage.Offset != 100 || last.Holdings.Groups[0].LotsPage.HasMore {
		t.Fatalf("last lot page: err=%v projection=%+v", err, last.Holdings)
	}
	label := *last.Holdings.Groups[0].Lots[0].CostLabel
	m = NewModel(true)
	m.page, m.holdingSection = pageHoldings, "lots"
	m.holdingAccount, m.holdingAccountDraft, m.holdingScope, m.holdingGroupBy = "Assets:Stock", "Assets:Stock", "subtree", "account"
	m.holdingGroupID, m.holdingOffset, m.holdingDescending = groupID, 100, false
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: last.State, Response: &last}
	tester = ui.NewTester(m.View, 1180, 780)
	if !tester.HasText(label) || !tester.HasText("101个批次 · 显示 101–101") {
		t.Fatalf("last source lot is not shown in the UI: label=%q texts=%q", label, tester.Texts())
	}
}

func TestQueryCatalogueSelectionOnlyFillsTheDraft(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "queries.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page = pageQuery
	m.queryDraft = "SELECT keep_me"
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: desktop.State{SnapshotID: state.SnapshotID}}
	response, err := session.Read(m.requestForView("query"))
	if err != nil {
		t.Fatal(err)
	}
	if response.QueryCatalogue == nil || response.QueryCatalogue.SnapshotID != state.SnapshotID || len(response.QueryCatalogue.Queries) != 3 || len(response.QueryCatalogue.Presets) != 4 {
		t.Fatalf("query catalogue = %+v", response.QueryCatalogue)
	}
	m.workspaceState.Response = &response
	tester := ui.NewTester(m.View, 1180, 780)
	m.focusCurrentSearch()
	tester.Frame()
	if !tester.Focused("搜索账本查询") {
		t.Fatalf("Find did not focus the query catalogue search field: %q", tester.Texts())
	}
	for _, want := range []string{"查询目录", "选择只填入草稿，不会运行查询", "填入预设 · 按月收入", "账本保存的查询 · 3 条", "Included", "同名"} {
		if !tester.HasText(want) {
			t.Errorf("query catalogue does not render %q: %q", want, tester.Texts())
		}
	}
	item := response.QueryCatalogue.Queries[0]
	button := fmt.Sprintf(m.tr("填入草稿 · %s · %s · %d"), item.Name, item.Date, item.Source.Line)
	if err := tester.Click(button); err != nil {
		t.Fatalf("fill saved query draft: %v; UI=%q", err, tester.Texts())
	}
	if m.queryDraft != item.Text || m.workspaceState.QueryResponse != nil || m.status != "已填入草稿；尚未执行。" {
		t.Fatalf("saved query selection ran instead of filling only the draft: draft=%q status=%q response=%+v", m.queryDraft, m.status, m.workspaceState.QueryResponse)
	}
	preset := query.Presets()[0]
	if err := tester.Click("填入预设 · " + preset.NameZH); err != nil {
		t.Fatalf("fill built-in preset: %v; UI=%q", err, tester.Texts())
	}
	if m.queryDraft != preset.Text || m.workspaceState.QueryResponse != nil {
		t.Fatalf("preset selection did not stay an unexecuted draft: %q", m.queryDraft)
	}
	m.queryPresetPeriodFrom, m.queryPresetPeriodTo = "2000-02-01", "2000-02-21"
	m.generateQueryPeriodPresets(state.SnapshotID)
	if len(m.queryPresetPeriodPresets) != 4 || m.queryPresetPeriodError != "" {
		t.Fatalf("period presets = %d, error = %q", len(m.queryPresetPeriodPresets), m.queryPresetPeriodError)
	}
	tester.Frame()
	if err := tester.Click("填入期间 · 按月收入"); err != nil {
		t.Fatalf("fill period preset: %v; UI=%q", err, tester.Texts())
	}
	if m.queryDraft != m.queryPresetPeriodPresets[0].Text || m.workspaceState.QueryResponse != nil {
		t.Fatalf("period preset did not remain an unexecuted draft: draft=%q response=%+v", m.queryDraft, m.workspaceState.QueryResponse)
	}
	m.queryPresetPeriodFrom = "not-a-date"
	m.generateQueryPeriodPresets(state.SnapshotID)
	if len(m.queryPresetPeriodPresets) != 0 || m.queryPresetPeriodError == "" {
		t.Fatalf("invalid period preset was accepted: presets=%d error=%q", len(m.queryPresetPeriodPresets), m.queryPresetPeriodError)
	}
	tester.SetSize(900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 1.5})
	tester.Frame()
	if !tester.HasText("查询工作台") || !tester.HasText("搜索账本查询") {
		t.Fatalf("small scaled query window lost its page/search controls: %q", tester.Texts())
	}
}

func TestSavedUserPresetSelectionOnlyFillsTheDraft(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.bean")
	if err := os.WriteFile(path, []byte("option \"title\" \"test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := NewModel(true)
	m.page = pageQuery
	m.queryDraft = "SELECT keep_me"
	m.queryPresetStore = querypresets.NewStore(filepath.Join(t.TempDir(), "query-presets.json"))
	m.queryPresetLedger = path
	m.queryPresets = []querypresets.Preset{{ID: "8d9c01e0c14b4f95b75ea8a77f84f269", Name: "Saved sample", Text: "SELECT date, amount FROM postings"}}
	m.workspaceState = workspace.State{
		ActivePath: path,
		Ledger:     desktop.State{SnapshotID: "current-snapshot"},
	}
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("我的查询预设 · 1 条") {
		t.Fatalf("saved user preset is not shown: %q", tester.Texts())
	}
	if err := tester.Click("填入草稿 · Saved sample"); err != nil {
		t.Fatalf("fill saved user preset: %v; UI=%q", err, tester.Texts())
	}
	if m.queryDraft != "SELECT date, amount FROM postings" || m.workspaceState.QueryResponse != nil || m.status != "已填入草稿；尚未执行。" {
		t.Fatalf("saved preset ran instead of filling only the draft: draft=%q status=%q response=%+v", m.queryDraft, m.status, m.workspaceState.QueryResponse)
	}
}

func TestQueryCatalogueRejectsStaleSnapshotAndIgnoresSharedFilters(t *testing.T) {
	m := NewModel(true)
	m.queryCatalogueOffset, m.queryCatalogueDescending = 100, false
	m.queryCatalogueSearch = "cash flow"
	m.workspaceState.Ledger.SnapshotID = "query-snapshot"
	m.filterDateFrom, m.filterDateTo, m.filterWindow = "2000-01-01", "2000-01-31", "1m"
	m.filterCurrency, m.filterValuation = "USD", "market-value"
	request := m.requestForView("query")
	if request.View != "query" || request.SnapshotID != "query-snapshot" || request.QuerySearch != "cash flow" || request.Offset != 100 || request.Sort != "ascending" || request.Limit != 100 || request.DateFrom != "" || request.DateTo != "" || request.Window != "" || request.Currency != "" || request.Valuation != "" {
		t.Fatalf("query catalogue inherited another page's filters: %+v", request)
	}
	path := filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "native", "queries.bean")
	path, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	response, err := session.Read(desktop.Request{View: "query", Limit: 100, Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	m.page = pageQuery
	m.workspaceState = workspace.State{Phase: workspace.PhaseReady, ActivePath: path, Ledger: desktop.State{SnapshotID: "newer-snapshot"}, Response: &response}
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("查询目录正在读取或已过期") || tester.HasText("填入预设 · 按月收入") {
		t.Fatalf("stale catalogue remained selectable: %q", tester.Texts())
	}
}

func waitWorkspace(t *testing.T, controller *workspace.Controller, predicate func(workspace.State) bool) workspace.State {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := controller.State()
		if predicate(state) {
			return state
		}
		time.Sleep(time.Millisecond)
	}
	state := controller.State()
	t.Fatalf("timed out waiting for workspace state: phase=%s operation=%d busy=%v error=%s", state.Phase, state.Operation, state.Busy, state.Error)
	return state
}
