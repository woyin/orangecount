// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package querypresets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orangecount/internal/query"
)

func TestStoreSaveReloadDeleteAndLedgerIsolation(t *testing.T) {
	root := t.TempDir()
	first := writeLedger(t, root, "first.bean")
	second := writeLedger(t, root, "second.bean")
	storePath := filepath.Join(root, "support", "query-presets-v1.json")
	store := NewStore(storePath)

	firstPresets, err := store.Save(first, "  Quarterly cash flow  ", "SELECT date, amount\nFROM postings")
	if err != nil {
		t.Fatal(err)
	}
	if len(firstPresets) != 1 || firstPresets[0].Name != "Quarterly cash flow" || len(firstPresets[0].ID) != 32 {
		t.Fatalf("saved presets = %+v", firstPresets)
	}
	if _, err := store.Save(first, "quarterly CASH FLOW", "SELECT 1"); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("duplicate name error = %v, want ErrDuplicateName", err)
	}
	if got, err := NewStore(storePath).LoadLedger(second); err != nil || len(got) != 0 {
		t.Fatalf("other ledger presets = %+v, err = %v", got, err)
	}
	reloaded, err := NewStore(storePath).LoadLedger(first)
	if err != nil || len(reloaded) != 1 || reloaded[0].Text != "SELECT date, amount\nFROM postings" {
		t.Fatalf("reloaded presets = %+v, err = %v", reloaded, err)
	}
	reloaded[0].Text = "mutated"
	again, err := NewStore(storePath).LoadLedger(first)
	if err != nil || again[0].Text != "SELECT date, amount\nFROM postings" {
		t.Fatalf("returned presets alias stored values: %+v, err = %v", again, err)
	}

	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), first) || strings.Contains(string(data), second) {
		t.Fatalf("preset store contains a raw ledger path: %s", data)
	}
	fileInfo, err := os.Stat(storePath)
	if err != nil || fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("store mode = %v, err = %v; want 0600", fileInfo, err)
	}
	dirInfo, err := os.Stat(filepath.Dir(storePath))
	if err != nil || dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("store directory mode = %v, err = %v; want 0700", dirInfo, err)
	}

	remaining, err := store.Delete(first, firstPresets[0].ID)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("delete result = %+v, err = %v", remaining, err)
	}
	if _, err := store.Delete(first, firstPresets[0].ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete error = %v, want ErrNotFound", err)
	}
	if got, err := store.LoadLedger(first); err != nil || len(got) != 0 {
		t.Fatalf("presets after delete = %+v, err = %v", got, err)
	}
}

func TestStoreRejectsInvalidValuesWithoutWriting(t *testing.T) {
	root := t.TempDir()
	ledger := writeLedger(t, root, "ledger.bean")
	path := filepath.Join(root, "query-presets.json")
	store := NewStore(path)
	invalid := []struct {
		name string
		text string
	}{
		{name: "", text: "SELECT 1"},
		{name: "name", text: " \n "},
		{name: "bad\x00name", text: "SELECT 1"},
		{name: strings.Repeat("界", 81), text: "SELECT 1"},
		{name: "too much", text: strings.Repeat("x", query.QueryTextLimit+1)},
		{name: "bad text", text: "SELECT\x00 1"},
	}
	for _, item := range invalid {
		if _, err := store.Save(ledger, item.name, item.text); !errors.Is(err, ErrInvalid) {
			t.Errorf("Save(%q, text bytes %d) error = %v, want ErrInvalid", item.name, len(item.text), err)
		}
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid save created store: %v", err)
	}
}

func TestStoreUnknownSchemaAndFailedWritePreserveExistingFile(t *testing.T) {
	root := t.TempDir()
	ledger := writeLedger(t, root, "ledger.bean")
	path := filepath.Join(root, "query-presets.json")
	store := NewStore(path)
	if _, err := store.Save(ledger, "Original", "SELECT 1"); err != nil {
		t.Fatal(err)
	}
	previous, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	store.writeFile = func(string, []byte) error { return errors.New("disk full") }
	if _, err := store.Save(ledger, "Second", "SELECT 2"); err == nil {
		t.Fatal("failed write reported success")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(previous) {
		t.Fatalf("failed write changed durable file: err=%v", err)
	}

	unknown := []byte(`{"schema_version":99,"ledgers":[]}`)
	if err := os.WriteFile(path, unknown, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(path).LoadLedger(ledger); !errors.Is(err, ErrSchema) {
		t.Fatalf("unknown schema error = %v, want ErrSchema", err)
	}
	if _, err := NewStore(path).Save(ledger, "Must not replace", "SELECT 3"); !errors.Is(err, ErrSchema) {
		t.Fatalf("unknown schema Save error = %v, want ErrSchema", err)
	}
	stillUnknown, err := os.ReadFile(path)
	if err != nil || string(stillUnknown) != string(unknown) {
		t.Fatalf("unknown schema was overwritten: %s, err=%v", stillUnknown, err)
	}
}

func TestConcurrentSavesPreserveBothLedgers(t *testing.T) {
	root := t.TempDir()
	first := writeLedger(t, root, "first.bean")
	second := writeLedger(t, root, "second.bean")
	store := NewStore(filepath.Join(root, "query-presets.json"))
	errs := make(chan error, 2)
	for _, item := range []struct{ path, name, text string }{
		{first, "First", "SELECT 1"}, {second, "Second", "SELECT 2"},
	} {
		item := item
		go func() {
			_, err := store.Save(item.path, item.name, item.text)
			errs <- err
		}()
	}
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{first, second} {
		got, err := store.LoadLedger(path)
		if err != nil || len(got) != 1 {
			t.Fatalf("presets for %s = %+v, err = %v", filepath.Base(path), got, err)
		}
	}
}

func writeLedger(t *testing.T, root, name string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte("option \"title\" \"test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
