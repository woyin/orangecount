// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/platform"
	"orangecount/apps/goui/internal/preferences"
)

func TestApplicationMenuRoutesCommandsAndExposesMacShortcuts(t *testing.T) {
	model := NewModel(true)
	model.page = pageQuery
	created := 0
	menu := NewApplicationMenu(func(*mygo.Window) *Model { return model }, func() *Model {
		created++
		return model
	})
	for id, accelerator := range map[string]string{
		"open-ledger":        "CmdOrCtrl+O",
		"new-window":         "CmdOrCtrl+N",
		"reload-ledger":      "CmdOrCtrl+R",
		"find":               "CmdOrCtrl+F",
		"increase-text-size": "CmdOrCtrl+Plus",
		"decrease-text-size": "CmdOrCtrl+-",
		"reset-text-size":    "CmdOrCtrl+0",
	} {
		item := menu.ItemByID(id)
		if item == nil || item.Accelerator != accelerator || item.Click == nil {
			t.Fatalf("menu item %q = %+v, want accelerator %q and a command", id, item, accelerator)
		}
	}
	for id, want := range map[string]string{
		"source-editor":             "选择来源编辑器…",
		"source-editor-default":     "使用系统默认应用打开来源",
		"export-preferences-backup": "导出偏好备份…",
		"increase-text-size":        "放大文字",
		"decrease-text-size":        "缩小文字",
		"reset-text-size":           "恢复默认文字大小",
	} {
		item := menu.ItemByID(id)
		if item == nil || item.Label != want || item.Click == nil {
			t.Errorf("Chinese menu item %q = %+v, want label %q and a command", id, item, want)
		}
	}
	menu.ItemByID("find").Click(menu.ItemByID("find"), nil)
	if !model.searchFocusPending || model.searchFocusPage != pageQuery {
		t.Fatalf("Find did not target the focused query window: pending=%t page=%d", model.searchFocusPending, model.searchFocusPage)
	}
	menu.ItemByID("new-window").Click(menu.ItemByID("new-window"), nil)
	if created != 1 {
		t.Fatalf("New Window calls = %d, want 1", created)
	}
	menu.ItemByID("reload-ledger").Click(menu.ItemByID("reload-ledger"), nil)
	if model.status != "请先打开账本。" {
		t.Fatalf("Reload without an active ledger status = %q", model.status)
	}
	menu.ItemByID("increase-text-size").Click(menu.ItemByID("increase-text-size"), nil)
	if model.textScale != 1.25 {
		t.Fatalf("Increase Text Size set scale to %v, want 1.25", model.textScale)
	}
	menu.ItemByID("increase-text-size").Click(menu.ItemByID("increase-text-size"), nil)
	if model.textScale != 1.5 {
		t.Fatalf("second Increase Text Size set scale to %v, want 1.5", model.textScale)
	}
	menu.ItemByID("increase-text-size").Click(menu.ItemByID("increase-text-size"), nil)
	if model.textScale != 1.5 {
		t.Fatalf("Increase Text Size exceeded the supported maximum: %v", model.textScale)
	}
	menu.ItemByID("decrease-text-size").Click(menu.ItemByID("decrease-text-size"), nil)
	if model.textScale != 1.25 {
		t.Fatalf("Decrease Text Size set scale to %v, want 1.25", model.textScale)
	}
	menu.ItemByID("reset-text-size").Click(menu.ItemByID("reset-text-size"), nil)
	if model.textScale != 1 {
		t.Fatalf("Reset Text Size set scale to %v, want 1", model.textScale)
	}
}

func TestApplicationMenuLocalizesForFocusedLedger(t *testing.T) {
	menu := NewApplicationMenu(nil, nil, "en")
	for id, want := range map[string]string{
		"open-ledger":               "Open Ledger…",
		"new-window":                "New Window",
		"reload-ledger":             "Reload Ledger",
		"source-editor":             "Choose Source Editor…",
		"source-editor-default":     "Use System Default for Sources",
		"export-preferences-backup": "Export Preferences Backup…",
		"find":                      "Find",
		"increase-text-size":        "Increase Text Size",
		"decrease-text-size":        "Decrease Text Size",
		"reset-text-size":           "Reset Text Size",
	} {
		item := menu.ItemByID(id)
		if item == nil || item.Label != want {
			t.Errorf("English menu item %q = %+v, want label %q", id, item, want)
		}
	}
}

func TestSourceEditorSelectionPersistsRoutesSourcesAndRestoresSystemDefault(t *testing.T) {
	store := preferences.NewStore(filepath.Join(t.TempDir(), "support", "preferences.json"))
	model := NewModel(true)
	var openedPath, openedBundleID string
	model.openSourceWithEditor = func(path, bundleID string) error {
		openedPath, openedBundleID = path, bundleID
		return nil
	}
	model.sourceEditorPicker = func(*mygo.Window) (string, error) { return "/Applications/Code.app", nil }
	model.sourceEditorInspector = func(path string) (platform.SourceEditor, error) {
		if path != "/Applications/Code.app" {
			t.Fatalf("editor picker path = %q", path)
		}
		return platform.SourceEditor{BundleID: "com.example.code-editor", Name: "Code"}, nil
	}
	model.SetPreferences(store)
	menu := NewApplicationMenu(func(*mygo.Window) *Model { return model }, nil)
	choose := menu.ItemByID("source-editor")
	if choose == nil || choose.Click == nil {
		t.Fatal("File menu has no source-editor chooser")
	}
	reset := menu.ItemByID("source-editor-default")
	if reset == nil || reset.Click == nil {
		t.Fatal("File menu has no system-default source-editor command")
	}
	if err := <-model.chooseSourceEditor(); err != nil {
		t.Fatal(err)
	}
	waitForSourceEditor(t, store, "com.example.code-editor")
	if err := model.openSourcePath("/tmp/ledger/include.bean"); err != nil {
		t.Fatal(err)
	}
	if openedPath != "/tmp/ledger/include.bean" || openedBundleID != "com.example.code-editor" {
		t.Fatalf("source route = %q via %q", openedPath, openedBundleID)
	}
	model.sourceEditorPicker = func(*mygo.Window) (string, error) { return "", nil }
	if err := <-model.chooseSourceEditor(); err != nil {
		t.Fatal(err)
	}
	if got, err := store.SourceEditorBundleID(); err != nil || got != "com.example.code-editor" {
		t.Fatalf("cancelled editor choice changed preference to %q, %v", got, err)
	}
	if err := <-model.restoreSystemSourceEditor(); err != nil {
		t.Fatal(err)
	}
	waitForSourceEditor(t, store, "")
	if err := model.openSourcePath("/tmp/ledger/include.bean"); err != nil {
		t.Fatal(err)
	}
	if openedPath != "/tmp/ledger/include.bean" || openedBundleID != "" {
		t.Fatalf("system-default source route = %q via %q", openedPath, openedBundleID)
	}
}

func TestSourceEditorPreferenceRoutesAfterModelRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "support", "preferences.json")
	firstStore := preferences.NewStore(path)
	const bundleID = "com.example.restart-editor"
	if err := firstStore.SetSourceEditorBundleID(bundleID); err != nil {
		t.Fatal(err)
	}

	model := NewModel(true)
	var openedPath, openedBundleID string
	model.openSourceWithEditor = func(path, editorBundleID string) error {
		openedPath, openedBundleID = path, editorBundleID
		return nil
	}
	model.SetPreferences(preferences.NewStore(path))
	const sourcePath = "/tmp/ledger/include.bean"
	if err := model.openSourcePath(sourcePath); err != nil {
		t.Fatal(err)
	}
	if openedPath != sourcePath || openedBundleID != bundleID {
		t.Fatalf("new model source route = %q via %q, want %q via %q", openedPath, openedBundleID, sourcePath, bundleID)
	}
}

func waitForSourceEditor(t *testing.T, store *preferences.Store, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := store.SourceEditorBundleID()
		if err != nil {
			t.Fatal(err)
		}
		if got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	got, err := store.SourceEditorBundleID()
	t.Fatalf("source editor bundle ID = %q, %v; want %q", got, err, want)
}

func TestFindFocusesHoldingsAccountSearch(t *testing.T) {
	model := NewModel(true)
	model.page = pageHoldings
	model.focusCurrentSearch()

	tester := ui.NewTester(model.holdingsFilters, 900, 600)
	if !tester.Focused("持仓账户") || model.searchFocusPending {
		t.Fatalf("Find did not focus the holdings account search: focused=%t pending=%t", tester.Focused("持仓账户"), model.searchFocusPending)
	}
}
