// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"orangecount/apps/goui/internal/localfiles"
	"orangecount/apps/goui/internal/preferences"
)

func TestPreferencesBackupMenuExportsFrozenPrivatePreferenceSnapshot(t *testing.T) {
	root := t.TempDir()
	ledgerPath := filepath.Join(root, "ledger.bean")
	if err := os.WriteFile(ledgerPath, []byte("option \"title\" \"backup-test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := preferences.NewStore(filepath.Join(root, "support", "preferences-v1.json"))
	if err := store.RecordRecentLedger(ledgerPath); err != nil {
		t.Fatal(err)
	}
	recentPaths, err := store.RecentLedgerPaths()
	if err != nil || len(recentPaths) != 1 {
		t.Fatalf("recent preference paths = %v, %v", recentPaths, err)
	}
	if err := store.UpdateLedger(ledgerPath, func(item *preferences.LedgerPreferences) error {
		item.Currency = "USD"
		item.Locale = "en"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "chosen", "preferences-backup.json")
	if err := os.Mkdir(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	model := NewModel(true)
	model.SetPreferences(store)
	result := make(chan struct {
		path string
		err  error
	}, 1)
	model.preferenceBackupSaveDialog = func(ctx context.Context, options mygo.SaveDialogOptions) (string, error) {
		if options.Title != "导出偏好备份" || options.ButtonLabel != "导出" || len(options.Filters) != 1 || options.Filters[0].Extensions[0] != "json" {
			t.Errorf("preferences backup save-panel options = %+v", options)
		}
		for _, want := range []string{"最近账本路径", "附件目录书签", "不包含账本内容", "查询草稿"} {
			if !strings.Contains(options.Message, want) {
				t.Errorf("backup save-panel scope omitted %q: %q", want, options.Message)
			}
		}
		if err := store.UpdateLedger(ledgerPath, func(item *preferences.LedgerPreferences) error {
			item.Currency = "EUR"
			return nil
		}); err != nil {
			return "", err
		}
		return destination, ctx.Err()
	}
	model.onPreferenceBackupResult = func(path string, err error) {
		result <- struct {
			path string
			err  error
		}{path: path, err: err}
	}
	menu := NewApplicationMenu(func(*mygo.Window) *Model { return model }, nil)
	item := menu.ItemByID("export-preferences-backup")
	if item == nil || item.Click == nil {
		t.Fatal("File menu has no preferences backup command")
	}
	item.Click(item, nil)
	select {
	case got := <-result:
		if got.err != nil || got.path != destination {
			t.Fatalf("backup result = %q, %v", got.path, got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("preferences backup did not finish")
	}
	model.preferenceWork.Wait()
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	var exported preferences.Document
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("exported backup is not valid JSON: %v", err)
	}
	if len(exported.RecentLedgerPaths) != 1 || exported.RecentLedgerPaths[0] != recentPaths[0] || len(exported.Ledgers) != 1 || exported.Ledgers[0].Currency != "USD" || exported.Ledgers[0].Locale != "en" {
		t.Fatalf("exported backup did not preserve the pre-panel preference snapshot: %+v", exported)
	}
	info, err := os.Stat(destination)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("backup file mode = %v, %v; want 0600", info, err)
	}
	if strings.Contains(string(data), "query_draft") || strings.Contains(string(data), "QueryResponse") {
		t.Fatal("backup contains query or rendered-result state")
	}
	current, err := store.ForLedger(ledgerPath)
	if err != nil || current.Currency != "EUR" {
		t.Fatalf("export changed the live preferences: %+v, %v", current, err)
	}
}

func TestPreferencesBackupCannotReplaceTheLiveStore(t *testing.T) {
	root := t.TempDir()
	ledgerPath := filepath.Join(root, "ledger.bean")
	if err := os.WriteFile(ledgerPath, []byte("option \"title\" \"backup-protection-test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	storePath := filepath.Join(root, "support", "preferences-v1.json")
	store := preferences.NewStore(storePath)
	if err := store.UpdateLedger(ledgerPath, func(item *preferences.LedgerPreferences) error {
		item.Currency = "USD"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	model := NewModel(true)
	model.SetPreferences(store)
	result := make(chan error, 1)
	model.preferenceBackupSaveDialog = func(context.Context, mygo.SaveDialogOptions) (string, error) {
		return storePath, nil
	}
	model.onPreferenceBackupResult = func(_ string, err error) { result <- err }
	model.exportPreferencesBackup()
	select {
	case err := <-result:
		if !errors.Is(err, localfiles.ErrProtectedTarget) {
			t.Fatalf("backup target protection error = %v, want %v", err, localfiles.ErrProtectedTarget)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("protected backup attempt did not finish")
	}
	model.preferenceWork.Wait()
	after, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("backup export replaced or modified the live preferences file")
	}
}
