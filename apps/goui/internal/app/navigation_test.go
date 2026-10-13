// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestPageHistoryIsBoundedAndAvoidsAdjacentDuplicates(t *testing.T) {
	history := pushPage(nil, pageWelcome)
	history = pushPage(history, pageWelcome)
	if len(history) != 1 {
		t.Fatalf("repeated current page grew history to %d entries", len(history))
	}
	for i := 0; i < maxPageHistory+20; i++ {
		history = pushPage(history, page(i%int(pageCharts+1)))
	}
	if len(history) != maxPageHistory {
		t.Fatalf("history length = %d, want its %d-entry cap", len(history), maxPageHistory)
	}
}

func TestViewNavigationMapsAreBidirectional(t *testing.T) {
	views := []string{"dashboard", "balance_sheet", "income_statement", "journal", "account", "prices", "holdings", "directives", "migration", "query", "attachments"}
	for _, view := range views {
		page, ok := pageForView(view)
		if !ok {
			t.Errorf("page mapping is missing view %q", view)
			continue
		}
		got, ok := viewForPage(page)
		if !ok || got != view {
			t.Errorf("view mapping for page %d = %q, %t; want %q", page, got, ok, view)
		}
	}
	if _, ok := pageForView("unsupported"); ok {
		t.Fatal("unsupported core view was accepted")
	}
}

func TestNavigationToolbarFitsMinimumWindowAtLargeTextScale(t *testing.T) {
	m := NewModel(true)
	m.locale = "en"
	m.page = pageDashboard
	m.pageHistoryBack = []page{pageWelcome}
	m.pageHistoryForward = []page{pageBalance}
	m.workspaceState.ActivePath = "/tmp/example.bean"
	tester := ui.NewTester(m.View, 900, 600)
	tester.SetPreferences(ui.Preferences{TextScale: 1.5})
	tester.Frame()
	back, hasBack := tester.Find("Back")
	forward, hasForward := tester.Find("Forward")
	if !hasBack || !hasForward {
		t.Fatalf("scaled minimum window hid a navigation action: %q", tester.Texts())
	}
	for label, rect := range map[string]ui.Rect{"Back": back, "Forward": forward} {
		if rect.X < 224 || rect.Y < 0 || rect.X+rect.W > 900 || rect.Y+rect.H > 600 {
			t.Errorf("%s button escaped the 900×600 content viewport: %+v", label, rect)
		}
	}
	if back.X >= forward.X {
		t.Fatalf("navigation buttons are not in reading order: Back=%+v Forward=%+v", back, forward)
	}
}

func TestOfflineAnalysisPageRestoresThroughBackForwardHistory(t *testing.T) {
	archivePath, document := writeQueryAnalysisFixture(t,
		filepath.Join("..", "..", "..", "..", "testdata", "fixtures", "query", "compound-values.bean"),
		"SELECT account, position WHERE account = 'Assets:Stock' LIMIT 1",
	)
	m := NewModel(true)
	m.page = pageQuery
	m.offlineAnalysis = &document
	m.offlineAnalysisPath = archivePath
	m.pageHistoryBack = []page{pageWelcome}
	tester := ui.NewTester(m.View, 1180, 780)
	if !tester.HasText("离线分析捕获") {
		t.Fatalf("fixture offline capture did not render: %q", tester.Texts())
	}
	m.navigateBack()
	tester.Frame()
	if m.page != pageWelcome || !tester.HasText("欢迎使用 OrangeCount") {
		t.Fatalf("back navigation did not restore the previous page: page=%d UI=%q", m.page, tester.Texts())
	}
	m.navigateForward()
	tester.Frame()
	if m.page != pageQuery || !tester.HasText("离线分析捕获") || m.status != "" || m.workspace != nil {
		t.Fatalf("forward navigation did not restore the offline capture without opening a ledger: page=%d status=%q workspace=%v UI=%q", m.page, m.status, m.workspace, tester.Texts())
	}
}
