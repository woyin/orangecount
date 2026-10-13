// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package preferences

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"orangecount/apps/goui/internal/platform"
)

func TestIdentityCanonicalizesSymlinksWithoutPersistingLedgerPath(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "private.bean")
	if err := os.WriteFile(ledger, []byte("option \"title\" \"private\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias.bean")
	if err := os.Symlink(ledger, alias); err != nil {
		t.Fatal(err)
	}
	first, err := Identity(ledger)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Identity(alias)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || strings.Contains(first, "private") {
		t.Fatalf("ledger identities differ or contain the path: %q / %q", first, second)
	}

	path := filepath.Join(root, "app", "preferences.json")
	store := NewStore(path)
	defaultPreferences, err := store.ForLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if defaultPreferences.SemanticsProfile != "legacy" {
		t.Fatalf("new ledger semantics = %q, want existing app's legacy default", defaultPreferences.SemanticsProfile)
	}
	if defaultPreferences.Locale != "zh-CN" {
		t.Fatalf("new ledger locale = %q, want simplified Chinese", defaultPreferences.Locale)
	}
	if err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error {
		item.Currency = "USD"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), ledger) || strings.Contains(string(data), alias) {
		t.Fatalf("preference file contains a source path: %s", data)
	}
}

func TestLedgerLocaleIsValidatedAndSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.bean")
	writeLedger(t, ledger)
	path := filepath.Join(root, "support", "preferences.json")
	store := NewStore(path)
	if err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error {
		item.Locale = "en"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewStore(path).ForLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Locale != "en" {
		t.Fatalf("reloaded locale = %q, want en", reloaded.Locale)
	}
	if err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error {
		item.Locale = "fr-FR"
		return nil
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unsupported locale update error = %v, want ErrInvalid", err)
	}
}

func TestLedgerTextScaleIsValidatedAndSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.bean")
	writeLedger(t, ledger)
	path := filepath.Join(root, "support", "preferences.json")
	store := NewStore(path)
	defaults, err := store.ForLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if defaults.TextScale != 1 {
		t.Fatalf("new ledger text scale = %v, want 1", defaults.TextScale)
	}
	if err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error {
		item.TextScale = 1.25
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewStore(path).ForLedger(ledger)
	if err != nil || reloaded.TextScale != 1.25 {
		t.Fatalf("reloaded text scale = %v, %v; want 1.25", reloaded.TextScale, err)
	}
	if err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error {
		item.TextScale = 1.3
		return nil
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unsupported text scale error = %v, want ErrInvalid", err)
	}

	// Version-1 preferences written before text_scale was added remain valid.
	if err := validateLedger(LedgerPreferences{Identity: reloaded.Identity, SemanticsProfile: "legacy", Window: "all", Valuation: "at-cost"}); err != nil {
		t.Fatalf("legacy preference record without text_scale became invalid: %v", err)
	}
}

func TestSourceEditorPreferencePersistsAndRejectsUnsafeBundleIDs(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "support", "preferences.json")
	store := NewStore(path)
	if got, err := store.SourceEditorBundleID(); err != nil || got != "" {
		t.Fatalf("default source editor = %q, %v; want system default", got, err)
	}
	if err := store.SetSourceEditorBundleID("com.example.code-editor"); err != nil {
		t.Fatal(err)
	}
	ledger := filepath.Join(root, "ledger.bean")
	writeLedger(t, ledger)
	if err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error {
		item.Currency = "USD"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"source_editor_bundle_id": "com.example.code-editor"`) || strings.Contains(string(data), filepath.Dir(path)) {
		t.Fatalf("source editor setting was not stored as a bundle ID: %s", data)
	}
	reloaded := NewStore(path)
	if got, err := reloaded.SourceEditorBundleID(); err != nil || got != "com.example.code-editor" {
		t.Fatalf("reloaded source editor = %q, %v", got, err)
	}
	for _, invalid := range []string{"org.orangecount.goui", "com.example/bad", "com.example.", "single"} {
		if err := reloaded.SetSourceEditorBundleID(invalid); !errors.Is(err, ErrInvalid) {
			t.Errorf("setting source editor to %q error = %v, want ErrInvalid", invalid, err)
		}
	}
	if got, err := reloaded.SourceEditorBundleID(); err != nil || got != "com.example.code-editor" {
		t.Fatalf("invalid setting replaced configured editor = %q, %v", got, err)
	}
	if err := reloaded.SetSourceEditorBundleID(""); err != nil {
		t.Fatal(err)
	}
	if got, err := reloaded.SourceEditorBundleID(); err != nil || got != "" {
		t.Fatalf("cleared source editor = %q, %v", got, err)
	}
	invalidDocument := []byte(`{"schema_version":1,"ledgers":[],"source_editor_bundle_id":"org.orangecount.goui"}`)
	if err := os.WriteFile(path, invalidDocument, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(path).Load(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed persisted source editor error = %v, want ErrInvalid", err)
	}
}

func TestStoreUpdateRestartAndPermissions(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.bean")
	writeLedger(t, ledger)
	path := filepath.Join(root, "support", "preferences.json")
	store := NewStore(path)
	err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error {
		item.SemanticsProfile = "legacy"
		item.DateFrom = "2026-01-01"
		item.DateTo = "2026-01-31"
		item.Window = "all"
		item.Currency = "EUR"
		item.Valuation = "market-value"
		item.Columns = map[string][]string{"journal": {"date", "payee", "amount"}}
		item.ColumnWidths = map[string]map[string]float32{"journal": {"date": 128, "payee": 260}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("preference file mode = %04o, want 0600", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("preference directory mode = %04o, want 0700", dirInfo.Mode().Perm())
	}

	item, err := NewStore(path).ForLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if item.SemanticsProfile != "legacy" || item.DateFrom != "2026-01-01" || item.Currency != "EUR" || item.Valuation != "market-value" {
		t.Fatalf("reloaded ledger preferences = %+v", item)
	}
	item.Columns["journal"][0] = "mutated"
	again, err := NewStore(path).ForLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if again.Columns["journal"][0] != "date" {
		t.Fatalf("preference columns share mutable memory: %+v", again.Columns)
	}
	item.ColumnWidths["journal"]["date"] = 999
	again, err = NewStore(path).ForLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if again.ColumnWidths["journal"]["date"] != 128 {
		t.Fatalf("preference column widths share mutable memory: %+v", again.ColumnWidths)
	}
}

func TestColumnWidthsRejectInvalidSchema(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.bean")
	writeLedger(t, ledger)
	store := NewStore(filepath.Join(root, "preferences.json"))
	for _, test := range []struct {
		name  string
		width float32
	}{
		{name: "too narrow", width: 15.9},
		{name: "too wide", width: 10001},
		{name: "infinite", width: float32(math.Inf(1))},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error {
				item.ColumnWidths = map[string]map[string]float32{"journal": {"date": test.width}}
				return nil
			})
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("UpdateLedger error = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestConcurrentWindowUpdatesPreserveDifferentLedgers(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.bean")
	second := filepath.Join(root, "second.bean")
	writeLedger(t, first)
	writeLedger(t, second)
	store := NewStore(filepath.Join(root, "support", "preferences.json"))
	var wait sync.WaitGroup
	for _, item := range []struct{ path, currency string }{{first, "USD"}, {second, "EUR"}} {
		item := item
		wait.Add(1)
		go func() {
			defer wait.Done()
			if err := store.UpdateLedger(item.path, func(preferences *LedgerPreferences) error {
				preferences.Currency = item.currency
				return nil
			}); err != nil {
				t.Errorf("update %s: %v", item.currency, err)
			}
		}()
	}
	wait.Wait()
	gotFirst, err := store.ForLedger(first)
	if err != nil {
		t.Fatal(err)
	}
	gotSecond, err := store.ForLedger(second)
	if err != nil {
		t.Fatal(err)
	}
	if gotFirst.Currency != "USD" || gotSecond.Currency != "EUR" {
		t.Fatalf("concurrent update results = %q, %q", gotFirst.Currency, gotSecond.Currency)
	}
}

func TestInvalidSchemaIsNotSilentlyOverwritten(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.bean")
	writeLedger(t, ledger)
	path := filepath.Join(root, "preferences.json")
	original := []byte(`{"schema_version":99,"ledgers":[]}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(path)
	if _, err := store.Load(); !errors.Is(err, ErrSchema) {
		t.Fatalf("Load error = %v, want ErrSchema", err)
	}
	if err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error { item.Currency = "USD"; return nil }); !errors.Is(err, ErrSchema) {
		t.Fatalf("UpdateLedger error = %v, want ErrSchema", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatalf("invalid preferences were overwritten: %s", data)
	}
}

func TestFailedWriteLeavesExistingPreferencesUntouched(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.bean")
	writeLedger(t, ledger)
	path := filepath.Join(root, "preferences.json")
	store := NewStore(path)
	if err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error { item.Currency = "USD"; return nil }); err != nil {
		t.Fatal(err)
	}
	previous, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	store.writeFile = func(string, []byte) error { return errors.New("disk full") }
	if err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error { item.Currency = "EUR"; return nil }); err == nil {
		t.Fatal("failed write was reported as successful")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(previous) != string(after) {
		t.Fatal("failed write changed the previous durable document")
	}
}

func TestStoreRejectsInvalidReadingRanges(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.bean")
	writeLedger(t, ledger)
	store := NewStore(filepath.Join(root, "preferences.json"))
	for _, dates := range [][2]string{{"2026-02-30", "2026-03-01"}, {"2026-04-01", "2026-03-31"}} {
		err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error {
			item.DateFrom, item.DateTo = dates[0], dates[1]
			return nil
		})
		if !errors.Is(err, ErrInvalidDate) {
			t.Errorf("update %q..%q error = %v, want ErrInvalidDate", dates[0], dates[1], err)
		}
	}
}

func TestRecentLedgersAreCanonicalBoundedAndExplicit(t *testing.T) {
	root := t.TempDir()
	paths := make([]string, 6)
	for i := range paths {
		paths[i] = filepath.Join(root, "ledger-"+string(rune('a'+i))+".bean")
		writeLedger(t, paths[i])
	}
	store := NewStore(filepath.Join(root, "preferences.json"))
	for _, path := range paths {
		if err := store.RecordRecentLedger(path); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.RecentLedgerPaths()
	if err != nil {
		t.Fatal(err)
	}
	canonical := func(path string) string {
		t.Helper()
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			t.Fatal(err)
		}
		return resolved
	}
	if len(got) != 5 || got[0] != canonical(paths[5]) || got[4] != canonical(paths[1]) {
		t.Fatalf("recent paths did not keep the five most recent in order: %v", got)
	}
	if err := store.RecordRecentLedger(paths[3]); err != nil {
		t.Fatal(err)
	}
	got, err = store.RecentLedgerPaths()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 || got[0] != canonical(paths[3]) || got[1] != canonical(paths[5]) {
		t.Fatalf("reopening an entry did not move it to the front: %v", got)
	}
	if _, err := store.RecentLedgerPaths(); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyMigrationImportsOnlyRecognizedPreferencesOnce(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.bean")
	second := filepath.Join(root, "second.bean")
	missing := filepath.Join(root, "missing.bean")
	writeLedger(t, first)
	writeLedger(t, second)
	legacy := &fakeLegacyDefaults{
		arrays:       map[string][]string{LegacyRecentLedgersKey: {first, missing, second}},
		strings:      map[string]string{"org.orangecount.semantics." + first: "v3"},
		dictionaries: map[string]map[string]string{"org.orangecount.reading." + first: {"currency": "USD", "valuation": "market-value", "unknown": "ignored"}},
	}
	store := NewStore(filepath.Join(root, "support", "preferences.json"))
	if err := store.MigrateLegacy(legacy); err != nil {
		t.Fatal(err)
	}
	paths, err := store.RecentLedgerPaths()
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 || paths[0] != first || paths[1] != missing || paths[2] != second {
		t.Fatalf("legacy recents = %v", paths)
	}
	item, err := store.ForLedger(first)
	if err != nil {
		t.Fatal(err)
	}
	if item.SemanticsProfile != "v3" || item.Currency != "USD" || item.Valuation != "market-value" {
		t.Fatalf("recognized legacy preferences were not imported: %+v", item)
	}
	defaultItem, err := store.ForLedger(second)
	if err != nil {
		t.Fatal(err)
	}
	if defaultItem.SemanticsProfile != "legacy" || defaultItem.Currency != "" {
		t.Fatalf("missing legacy settings changed the existing default: %+v", defaultItem)
	}
	legacy.strings["org.orangecount.semantics."+first] = "legacy"
	if err := store.MigrateLegacy(legacy); err != nil {
		t.Fatal(err)
	}
	again, err := store.ForLedger(first)
	if err != nil {
		t.Fatal(err)
	}
	if again.SemanticsProfile != "v3" {
		t.Fatalf("completed migration ran a second time: %+v", again)
	}
	doc, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if doc.LegacyMigrationVersion != CurrentLegacyMigrationVersion {
		t.Fatalf("legacy migration marker = %d", doc.LegacyMigrationVersion)
	}
}

func TestLegacyMigrationDoesNotOverwriteExistingGoPreferences(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ledger.bean")
	writeLedger(t, path)
	existingDirectory := filepath.Join(root, "current-attachments")
	legacyDirectory := filepath.Join(root, "legacy-attachments")
	if err := os.Mkdir(existingDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(legacyDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	store := NewStore(filepath.Join(root, "preferences.json"))
	if err := store.UpdateLedger(path, func(item *LedgerPreferences) error {
		item.SemanticsProfile, item.Currency = "legacy", "EUR"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	currentRoot := platform.DirectoryBookmark{Path: existingDirectory, Data: []byte{9}, Device: 2, Inode: 4}
	if _, err := store.UpdateAttachmentRoots(path, []platform.DirectoryBookmark{currentRoot}, "writer"); err != nil {
		t.Fatal(err)
	}
	legacyRoot := platform.DirectoryBookmark{Path: legacyDirectory, Data: []byte{1}, Device: 3, Inode: 5}
	legacyRoots, err := json.Marshal(struct {
		Generation  uint64                       `json:"generation"`
		Directories []platform.DirectoryBookmark `json:"directories"`
	}{Generation: 2, Directories: []platform.DirectoryBookmark{legacyRoot}})
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicalLedgerPath(path)
	if err != nil {
		t.Fatal(err)
	}
	legacy := &fakeLegacyDefaults{
		arrays:  map[string][]string{LegacyRecentLedgersKey: {path}},
		strings: map[string]string{"org.orangecount.semantics." + path: "v3"},
		data:    map[string][]byte{LegacyAttachmentRootsKeyPrefix + canonical: legacyRoots},
	}
	if err := store.MigrateLegacy(legacy); err != nil {
		t.Fatal(err)
	}
	got, err := store.ForLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.SemanticsProfile != "legacy" || got.Currency != "EUR" {
		t.Fatalf("legacy migration replaced current settings: %+v", got)
	}
	if len(got.AttachmentRoots) != 1 || got.AttachmentRoots[0].Path != existingDirectory || string(got.AttachmentRoots[0].Data) != string(currentRoot.Data) || got.AttachmentRootsGeneration != 1 {
		t.Fatalf("legacy migration replaced current attachment roots: %+v", got.AttachmentRoots)
	}
}

func TestLegacyRecentMigrationPreservesNewerGoRecentsFirst(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, "current.bean")
	legacy := filepath.Join(root, "legacy.bean")
	writeLedger(t, current)
	writeLedger(t, legacy)
	store := NewStore(filepath.Join(root, "preferences.json"))
	if err := store.RecordRecentLedger(current); err != nil {
		t.Fatal(err)
	}
	if err := store.MigrateLegacy(&fakeLegacyDefaults{
		arrays: map[string][]string{LegacyRecentLedgersKey: {legacy}},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := store.RecentLedgerPaths()
	if err != nil {
		t.Fatal(err)
	}
	canonicalCurrent, err := filepath.EvalSymlinks(current)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != canonicalCurrent || got[1] != legacy {
		t.Fatalf("migration reordered newer Go recents: %v", got)
	}
}

func TestLegacyMigrationImportsValidatedAttachmentBookmarks(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.bean")
	attachments := filepath.Join(root, "attachments")
	writeLedger(t, ledger)
	if err := os.Mkdir(attachments, 0o700); err != nil {
		t.Fatal(err)
	}
	bookmark := platform.DirectoryBookmark{Path: attachments, Data: []byte{1, 2, 3}, Device: 11, Inode: 29}
	legacyConfig := struct {
		Generation  uint64                       `json:"generation"`
		Directories []platform.DirectoryBookmark `json:"directories"`
	}{Generation: 7, Directories: []platform.DirectoryBookmark{bookmark}}
	encoded, err := json.Marshal(legacyConfig)
	if err != nil {
		t.Fatal(err)
	}
	canonicalLedger, err := canonicalLedgerPath(ledger)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(filepath.Join(root, "preferences.json"))
	if err := store.UpdateLedger(ledger, func(item *LedgerPreferences) error {
		item.Currency = "EUR"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	doc, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	doc.LegacyMigrationVersion = 1 // Upgrade a preview written before bookmark migration.
	previous, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.path, previous, 0o600); err != nil {
		t.Fatal(err)
	}
	defaults := &fakeLegacyDefaults{
		arrays: map[string][]string{LegacyRecentLedgersKey: {ledger}},
		data:   map[string][]byte{LegacyAttachmentRootsKeyPrefix + canonicalLedger: encoded},
	}
	if err := store.MigrateLegacy(defaults); err != nil {
		t.Fatal(err)
	}
	item, err := store.ForLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if item.Currency != "EUR" || item.AttachmentRootsGeneration != 7 || len(item.AttachmentRoots) != 1 || item.AttachmentRoots[0].Path != bookmark.Path || string(item.AttachmentRoots[0].Data) != string(bookmark.Data) {
		t.Fatalf("legacy attachment roots were not migrated: %+v", item)
	}
	doc, err = store.Load()
	if err != nil || doc.LegacyMigrationVersion != CurrentLegacyMigrationVersion {
		t.Fatalf("legacy migration version = %d, %v", doc.LegacyMigrationVersion, err)
	}

	bookmark.Data[0] = 9
	item.AttachmentRoots[0].Data[0] = 8
	reloaded, generation, err := store.AttachmentRoots(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if generation != 7 || len(reloaded) != 1 || string(reloaded[0].Data) != string([]byte{1, 2, 3}) {
		t.Fatalf("attachment bookmark bytes were not defensively copied: %v", reloaded)
	}
	nextGeneration, err := store.UpdateAttachmentRoots(ledger, reloaded, "writer")
	if err != nil || nextGeneration != 8 {
		t.Fatalf("UpdateAttachmentRoots generation = %d, %v; want 8", nextGeneration, err)
	}
	if _, err := store.UpdateAttachmentRoots(ledger, make([]platform.DirectoryBookmark, 33), "writer"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("too many attachment roots error = %v, want %v", err, ErrInvalid)
	}
}

func TestClearedAttachmentRootsSurviveRestartAndLegacyMigration(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.bean")
	legacyDirectory := filepath.Join(root, "legacy-attachments")
	writeLedger(t, ledger)
	if err := os.Mkdir(legacyDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "preferences.json")
	store := NewStore(path)
	generation, err := store.UpdateAttachmentRoots(ledger, []platform.DirectoryBookmark{}, "writer")
	if err != nil || generation != 1 {
		t.Fatalf("clearing attachment roots: generation=%d err=%v", generation, err)
	}
	store = NewStore(path)
	legacyRoot := platform.DirectoryBookmark{Path: legacyDirectory, Data: []byte{1}, Device: 2, Inode: 3}
	legacyData, err := json.Marshal(struct {
		Generation  uint64                       `json:"generation"`
		Directories []platform.DirectoryBookmark `json:"directories"`
	}{Generation: 4, Directories: []platform.DirectoryBookmark{legacyRoot}})
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicalLedgerPath(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MigrateLegacy(&fakeLegacyDefaults{
		arrays: map[string][]string{LegacyRecentLedgersKey: {ledger}},
		data:   map[string][]byte{LegacyAttachmentRootsKeyPrefix + canonical: legacyData},
	}); err != nil {
		t.Fatal(err)
	}
	item, err := store.ForLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if item.AttachmentRoots == nil || len(item.AttachmentRoots) != 0 || item.AttachmentRootsGeneration != 1 {
		t.Fatalf("cleared roots were restored from legacy defaults: %+v", item)
	}
}

func TestAttachmentRootsUpdatesNotifyOtherWindowsWithCopiedRecords(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.bean")
	writeLedger(t, ledger)
	store := NewStore(filepath.Join(root, "preferences.json"))
	writerEvents := make(chan AttachmentRootsChange, 1)
	readerEvents := make(chan AttachmentRootsChange, 1)
	stopWriter := store.SubscribeAttachmentRoots("writer", func(change AttachmentRootsChange) { writerEvents <- change })
	stopReader := store.SubscribeAttachmentRoots("reader", func(change AttachmentRootsChange) { readerEvents <- change })
	defer stopWriter()
	defer stopReader()

	bookmark := platform.DirectoryBookmark{Path: root, Data: []byte{1, 2}, Device: 3, Inode: 4}
	generation, err := store.UpdateAttachmentRoots(ledger, []platform.DirectoryBookmark{bookmark}, "writer")
	if err != nil || generation != 1 {
		t.Fatalf("UpdateAttachmentRoots generation = %d, %v", generation, err)
	}
	select {
	case <-writerEvents:
		t.Fatal("originating window received its own policy update")
	case change := <-readerEvents:
		if change.Generation != 1 || len(change.Roots) != 1 || change.Identity == "" || change.Origin != "writer" {
			t.Fatalf("shared root event = %+v", change)
		}
		change.Roots[0].Data[0] = 9
	case <-time.After(time.Second):
		t.Fatal("another window did not receive the roots update")
	}
	stored, _, err := store.AttachmentRoots(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored[0].Data) != string(bookmark.Data) {
		t.Fatalf("subscriber mutation changed stored bookmark: %v", stored[0].Data)
	}

	stopReader()
	if _, err := store.UpdateAttachmentRoots(ledger, []platform.DirectoryBookmark{}, "writer"); err != nil {
		t.Fatal(err)
	}
	select {
	case change := <-readerEvents:
		t.Fatalf("unsubscribed window received a policy update: %+v", change)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestLegacyPathExpansionAndValidation(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	for input, want := range map[string]string{
		"~":                     filepath.Clean(home),
		"~/ledger/../main.bean": filepath.Join(home, "main.bean"),
	} {
		got, err := normalizeLegacyPath(input)
		if err != nil || got != want {
			t.Errorf("normalizeLegacyPath(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	for _, input := range []string{"", "relative.bean", "/tmp/a\x00bean"} {
		if _, err := normalizeLegacyPath(input); err == nil {
			t.Errorf("normalizeLegacyPath(%q) unexpectedly succeeded", input)
		}
	}
}

type fakeLegacyDefaults struct {
	arrays       map[string][]string
	strings      map[string]string
	dictionaries map[string]map[string]string
	data         map[string][]byte
}

func (f *fakeLegacyDefaults) StringArray(key string) ([]string, bool, error) {
	value, ok := f.arrays[key]
	return append([]string(nil), value...), ok, nil
}

func (f *fakeLegacyDefaults) String(key string) (string, bool, error) {
	value, ok := f.strings[key]
	return value, ok, nil
}

func (f *fakeLegacyDefaults) StringDictionary(key string) (map[string]string, bool, error) {
	value, ok := f.dictionaries[key]
	out := make(map[string]string, len(value))
	for name, item := range value {
		out[name] = item
	}
	return out, ok, nil
}

func (f *fakeLegacyDefaults) Data(key string) ([]byte, bool, error) {
	value, ok := f.data[key]
	return append([]byte(nil), value...), ok, nil
}
func writeLedger(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("option \"title\" \"test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
