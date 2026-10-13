// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

//go:build darwin

package preferences

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestRealLegacyProductionDefaultsMigrationIsReadOnly is opt-in because it
// reads the current user's old app preferences. It never prints ledger paths
// or values and writes only to t.TempDir.
func TestRealLegacyProductionDefaultsMigrationIsReadOnly(t *testing.T) {
	if os.Getenv("ORANGECOUNT_TEST_REAL_LEGACY_DEFAULTS") != "1" {
		t.Skip("set ORANGECOUNT_TEST_REAL_LEGACY_DEFAULTS=1 to validate the real production defaults domain read-only")
	}
	defaults, err := OpenLegacyDefaults(LegacyApplicationDomain)
	if err != nil {
		t.Fatal(err)
	}
	defer defaults.(*darwinLegacyDefaults).Close()

	before, err := captureLegacyDefaultsSnapshot(defaults)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(filepath.Join(t.TempDir(), "preferences-v1.json"))
	if err := store.MigrateLegacy(defaults); err != nil {
		t.Fatalf("migrate production defaults into temporary Go preferences: %v", err)
	}
	doc, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if doc.LegacyMigrationVersion != CurrentLegacyMigrationVersion {
		t.Fatalf("migration version = %d, want %d", doc.LegacyMigrationVersion, CurrentLegacyMigrationVersion)
	}
	if len(doc.RecentLedgerPaths) > 5 || len(doc.Ledgers) > 5 {
		t.Fatalf("migration exceeded the bounded recent/profile counts: %d/%d", len(doc.RecentLedgerPaths), len(doc.Ledgers))
	}
	legacyRootConfigurations := 0
	for _, saved := range before.Roots {
		if saved.Found {
			legacyRootConfigurations++
		}
	}
	verifiedEntries := 0
	seenPaths := make(map[string]bool)
	for _, rawPath := range before.Recents.Value {
		path, err := normalizeLegacyPath(rawPath)
		if err != nil || seenPaths[path] {
			continue
		}
		seenPaths[path] = true
		if _, err := Identity(path); err != nil {
			continue // A missing legacy recent remains visible but has no ledger profile to import.
		}
		item, err := store.ForLedger(path)
		if err != nil {
			t.Fatalf("read migrated preference record %d: %v", verifiedEntries, err)
		}
		profileKey := "org.orangecount.semantics." + path
		wantProfile := "legacy"
		if saved := before.Profiles[profileKey]; saved.Found && (saved.Value == "legacy" || saved.Value == "v3") {
			wantProfile = saved.Value
		}
		if item.SemanticsProfile != wantProfile {
			t.Fatalf("migrated semantics profile mismatch for existing entry %d", verifiedEntries)
		}
		readingKey := "org.orangecount.reading." + path
		if saved := before.Readings[readingKey]; saved.Found {
			if currency := strings.TrimSpace(saved.Value["currency"]); validCurrency(currency) && item.Currency != currency {
				t.Fatalf("migrated currency mismatch for existing entry %d", verifiedEntries)
			}
			if valuation := saved.Value["valuation"]; validValuation(valuation) && item.Valuation != valuation {
				t.Fatalf("migrated valuation mismatch for existing entry %d", verifiedEntries)
			}
		}
		canonical, err := canonicalLedgerPath(path)
		if err != nil {
			t.Fatalf("canonicalize existing ledger %d: %v", verifiedEntries, err)
		}
		rootsKey := LegacyAttachmentRootsKeyPrefix + canonical
		if saved := before.Roots[rootsKey]; saved.Found {
			config, err := decodeLegacyAttachmentRoots(saved.Value)
			if err != nil || item.AttachmentRootsGeneration != config.Generation || !reflect.DeepEqual(item.AttachmentRoots, config.Directories) {
				t.Fatalf("migrated attachment roots mismatch for existing entry %d: %v", verifiedEntries, err)
			}
		}
		verifiedEntries++
	}
	after, err := captureLegacyDefaultsSnapshot(defaults)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("migration changed recognized values in the old production defaults domain")
	}
	t.Logf("real production defaults domain was read-only; recognized recent paths=%d, migrated ledger preferences=%d, verified existing entries=%d, attachment configurations=%d", len(doc.RecentLedgerPaths), len(doc.Ledgers), verifiedEntries, legacyRootConfigurations)
}

type capturedLegacyValue[T any] struct {
	Found bool
	Value T
}

type legacyDefaultsSnapshot struct {
	Recents  capturedLegacyValue[[]string]
	Profiles map[string]capturedLegacyValue[string]
	Readings map[string]capturedLegacyValue[map[string]string]
	Roots    map[string]capturedLegacyValue[[]byte]
}

func captureLegacyDefaultsSnapshot(defaults LegacyDefaults) (legacyDefaultsSnapshot, error) {
	snapshot := legacyDefaultsSnapshot{
		Profiles: make(map[string]capturedLegacyValue[string]),
		Readings: make(map[string]capturedLegacyValue[map[string]string]),
		Roots:    make(map[string]capturedLegacyValue[[]byte]),
	}
	paths, found, err := defaults.StringArray(LegacyRecentLedgersKey)
	if err != nil {
		return legacyDefaultsSnapshot{}, err
	}
	snapshot.Recents = capturedLegacyValue[[]string]{Found: found, Value: paths}
	for _, rawPath := range paths {
		path, err := normalizeLegacyPath(rawPath)
		if err != nil {
			continue
		}
		profileKey := "org.orangecount.semantics." + path
		profile, found, err := defaults.String(profileKey)
		if err != nil {
			return legacyDefaultsSnapshot{}, err
		}
		snapshot.Profiles[profileKey] = capturedLegacyValue[string]{Found: found, Value: profile}

		readingKey := "org.orangecount.reading." + path
		reading, found, err := defaults.StringDictionary(readingKey)
		if err != nil {
			return legacyDefaultsSnapshot{}, err
		}
		snapshot.Readings[readingKey] = capturedLegacyValue[map[string]string]{Found: found, Value: reading}

		canonical, err := canonicalLedgerPath(path)
		if err != nil {
			continue
		}
		rootsKey := LegacyAttachmentRootsKeyPrefix + canonical
		roots, found, err := defaults.Data(rootsKey)
		if err != nil {
			return legacyDefaultsSnapshot{}, err
		}
		snapshot.Roots[rootsKey] = capturedLegacyValue[[]byte]{Found: found, Value: roots}
	}
	return snapshot, nil
}
