// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package preferences

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupSnapshotFreezesValidatedPreferenceDocument(t *testing.T) {
	root := t.TempDir()
	ledgerPath := filepath.Join(root, "ledger.bean")
	writeLedger(t, ledgerPath)
	storePath := filepath.Join(root, "support", "preferences-v1.json")
	store := NewStore(storePath)
	legacy := &fakeLegacyDefaults{
		arrays:  map[string][]string{LegacyRecentLedgersKey: {ledgerPath}},
		strings: map[string]string{"org.orangecount.semantics." + ledgerPath: "v3"},
		dictionaries: map[string]map[string]string{
			"org.orangecount.reading." + ledgerPath: {"currency": "USD", "valuation": "market-value"},
		},
	}
	if err := store.MigrateLegacy(legacy); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateLedger(ledgerPath, func(item *LedgerPreferences) error {
		item.Locale = "en"
		item.TextScale = 1.5
		if item.Columns == nil {
			item.Columns = make(map[string][]string)
		}
		item.Columns["journal"] = []string{"date", "amount"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.BackupSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.ProtectedPaths) != 1 || snapshot.ProtectedPaths[0] != storePath {
		t.Fatalf("backup protected paths = %v, want live preference file %q", snapshot.ProtectedPaths, storePath)
	}
	var exported Document
	if err := json.Unmarshal(snapshot.Content, &exported); err != nil {
		t.Fatalf("backup is not valid JSON: %v", err)
	}
	if exported.SchemaVersion != SchemaVersion || exported.LegacyMigrationVersion != CurrentLegacyMigrationVersion || len(exported.RecentLedgerPaths) != 1 || exported.RecentLedgerPaths[0] != ledgerPath {
		t.Fatalf("backup lost document identity or migration state: %+v", exported)
	}
	if len(exported.Ledgers) != 1 || exported.Ledgers[0].SemanticsProfile != "v3" || exported.Ledgers[0].Locale != "en" || exported.Ledgers[0].TextScale != 1.5 || exported.Ledgers[0].Currency != "USD" || exported.Ledgers[0].Valuation != "market-value" || strings.Join(exported.Ledgers[0].Columns["journal"], ",") != "date,amount" {
		t.Fatalf("backup lost migrated or current reading preferences: %+v", exported.Ledgers)
	}
	if strings.Contains(string(snapshot.Content), "query_draft") || strings.Contains(string(snapshot.Content), "QueryResponse") {
		t.Fatal("backup contains query or rendered-result state outside the preferences schema")
	}

	if err := store.UpdateLedger(ledgerPath, func(item *LedgerPreferences) error {
		item.Currency = "EUR"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(snapshot.Content), `"currency": "USD"`) || strings.Contains(string(snapshot.Content), `"currency": "EUR"`) {
		t.Fatal("backup snapshot changed after later preference updates")
	}
	current, err := store.BackupSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(current.Content), `"currency": "EUR"`) {
		t.Fatal("a new backup snapshot did not include the latest preference update")
	}
	if _, err := os.Stat(storePath); err != nil {
		t.Fatalf("backup snapshot unexpectedly moved or removed the live preferences: %v", err)
	}
}
