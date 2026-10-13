// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"orangecount/apps/goui/internal/preferences"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/ledger"
)

func TestTextScaleMenuPersistsPerLedgerAndRestoresOnOpen(t *testing.T) {
	root := t.TempDir()
	ledgerPath := filepath.Join(root, "ledger.bean")
	otherLedgerPath := filepath.Join(root, "other.bean")
	contents := []byte("option \"operating_currency\" \"USD\"\n")
	for _, path := range []string{ledgerPath, otherLedgerPath} {
		if err := os.WriteFile(path, contents, 0600); err != nil {
			t.Fatal(err)
		}
	}
	store := preferences.NewStore(filepath.Join(root, "preferences-v1.json"))
	model := NewModel(true)
	model.SetPreferences(store)
	model.workspaceState.ActivePath = ledgerPath
	model.workspaceState.Ledger.SemanticsProfile = ledger.Legacy
	menu := NewApplicationMenu(func(*mygo.Window) *Model { return model }, nil)
	for _, step := range []struct {
		id   string
		want float32
	}{{"increase-text-size", 1.25}, {"increase-text-size", 1.5}, {"decrease-text-size", 1.25}} {
		item := menu.ItemByID(step.id)
		if item == nil || item.Click == nil {
			t.Fatalf("text-size menu item %q is unavailable", step.id)
		}
		item.Click(item, nil)
		if model.textScale != step.want {
			t.Fatalf("menu %q set scale to %v, want %v", step.id, model.textScale, step.want)
		}
		waitForStoredTextScale(t, store, ledgerPath, step.want)
	}
	model.Close()

	stored, err := store.ForLedger(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	restored := NewModel(true)
	restored.SetPreferences(store)
	restored.pendingOpen = true
	restored.pendingPath = ledgerPath
	restored.pendingPreferences = stored
	restored.applyPendingOpenState(workspace.State{Phase: workspace.PhaseReady, ActivePath: ledgerPath})
	if restored.textScale != 1.25 || restored.pendingOpen {
		t.Fatalf("reopened ledger text scale = %v (pending=%t), want 1.25 and ready", restored.textScale, restored.pendingOpen)
	}
	restored.Close()

	other, err := store.ForLedger(otherLedgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if other.TextScale != 1 {
		t.Fatalf("text scale leaked into another ledger: %v", other.TextScale)
	}
}

func waitForStoredTextScale(t *testing.T, store *preferences.Store, path string, want float32) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		item, err := store.ForLedger(path)
		if err == nil && item.TextScale == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	item, err := store.ForLedger(path)
	t.Fatalf("stored text scale = %v, %v; want %v", item.TextScale, err, want)
}
