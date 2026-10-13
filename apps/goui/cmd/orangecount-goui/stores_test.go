// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/egoist/mygo"
	"orangecount/apps/goui/internal/preferences"
)

type startupLegacyDefaults struct {
	recent       []string
	strings      map[string]string
	dictionaries map[string]map[string]string
	arrayReads   int
	closed       bool
}

func (d *startupLegacyDefaults) StringArray(key string) ([]string, bool, error) {
	d.arrayReads++
	if key != preferences.LegacyRecentLedgersKey {
		return nil, false, nil
	}
	return append([]string(nil), d.recent...), true, nil
}

func (d *startupLegacyDefaults) String(key string) (string, bool, error) {
	value, ok := d.strings[key]
	return value, ok, nil
}

func (d *startupLegacyDefaults) StringDictionary(key string) (map[string]string, bool, error) {
	value, ok := d.dictionaries[key]
	return value, ok, nil
}

func (*startupLegacyDefaults) Data(string) ([]byte, bool, error) { return nil, false, nil }

func (d *startupLegacyDefaults) Close() { d.closed = true }

func TestProductionStartupStoresMigrateLegacyAndRemainUnderUserData(t *testing.T) {
	ledgerPath := startupFixture(t, "native/cashflow.bean")
	userData := filepath.Join(t.TempDir(), "OrangeCount Go UI Development")
	legacy := &startupLegacyDefaults{
		recent: []string{ledgerPath},
		strings: map[string]string{
			"org.orangecount.semantics." + ledgerPath: "v3",
		},
		dictionaries: map[string]map[string]string{
			"org.orangecount.reading." + ledgerPath: {"currency": "USD", "valuation": "market-value"},
		},
	}
	var openedDomain string
	preferenceStore, presetStore := openApplicationStores(userData, false, func(domain string) (preferences.LegacyDefaults, error) {
		openedDomain = domain
		return legacy, nil
	}, nil)
	if preferenceStore == nil || presetStore == nil {
		t.Fatal("production startup returned nil stores")
	}
	if openedDomain != preferences.LegacyApplicationDomain || !legacy.closed {
		t.Fatalf("legacy defaults lifecycle = domain:%q closed:%t", openedDomain, legacy.closed)
	}
	paths, err := preferenceStore.RecentLedgerPaths()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths, []string{ledgerPath}) {
		t.Fatalf("migrated recent paths = %q, want only %q", paths, ledgerPath)
	}
	item, err := preferenceStore.ForLedger(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if item.SemanticsProfile != "v3" || item.Currency != "USD" || item.Valuation != "market-value" {
		t.Fatalf("migrated ledger reading preferences = %+v", item)
	}
	document, err := preferenceStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if document.LegacyMigrationVersion != preferences.CurrentLegacyMigrationVersion {
		t.Fatalf("migration marker = %d", document.LegacyMigrationVersion)
	}
	if _, err := presetStore.Save(ledgerPath, "Production only", "SELECT account LIMIT 1"); err != nil {
		t.Fatal(err)
	}
	checkPrivateFile(t, filepath.Join(userData, "preferences-v1.json"))
	checkPrivateFile(t, filepath.Join(userData, "query-presets-v1.json"))
	checkPrivateDirectory(t, userData)
}

func TestQAStartupStoresDoNotOpenLegacyDefaults(t *testing.T) {
	userData := filepath.Join(t.TempDir(), "OrangeCount Go UI QA")
	legacyCalls := 0
	preferenceStore, presetStore := openApplicationStores(userData, true, func(string) (preferences.LegacyDefaults, error) {
		legacyCalls++
		return nil, errors.New("QA must not access legacy defaults")
	}, nil)
	if preferenceStore == nil || presetStore == nil {
		t.Fatal("QA startup returned nil stores")
	}
	if legacyCalls != 0 {
		t.Fatalf("QA startup opened legacy defaults %d times", legacyCalls)
	}
	document, err := preferenceStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if document.LegacyMigrationVersion != 0 || len(document.RecentLedgerPaths) != 0 {
		t.Fatalf("new QA preference document contains migrated user data: %+v", document)
	}
	checkPrivateDirectory(t, userData)
}

func TestIsolatedUserDataOverrideIsPrivateAndExplicit(t *testing.T) {
	root := t.TempDir()
	requested := filepath.Join(root, "isolated", "qa-data")
	var setPathCalls int
	var actualName mygo.PathName
	var actualPath string
	setPath := func(name mygo.PathName, path string) {
		setPathCalls++
		actualName, actualPath = name, path
	}
	isolated, err := configureIsolatedUserDataPath(setPath, requested)
	if err != nil {
		t.Fatal(err)
	}
	if !isolated || setPathCalls != 1 || actualName != mygo.PathUserData || actualPath != requested {
		t.Fatalf("isolated user-data override = isolated:%t calls:%d name:%q path:%q", isolated, setPathCalls, actualName, actualPath)
	}
	checkPrivateDirectory(t, requested)

	isolated, err = configureIsolatedUserDataPath(setPath, " ")
	if err != nil {
		t.Fatal(err)
	}
	if isolated || setPathCalls != 1 {
		t.Fatalf("empty override unexpectedly isolated the app: isolated:%t calls:%d", isolated, setPathCalls)
	}

	productionPath := filepath.Join(root, "must-not-exist")
	if _, err := configureIsolatedUserDataPath(setPath, "relative/path"); err == nil {
		t.Fatal("relative isolated user-data override was accepted")
	}
	if _, err := os.Stat(productionPath); !os.IsNotExist(err) {
		t.Fatalf("invalid override created an unexpected directory: %v", err)
	}

	sharedPath := filepath.Join(root, "shared")
	if err := os.Mkdir(sharedPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sharedPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := configureIsolatedUserDataPath(setPath, sharedPath); err == nil {
		t.Fatal("non-private existing isolated directory was accepted")
	}
	info, err := os.Stat(sharedPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("override validation changed existing directory permissions to %o", info.Mode().Perm())
	}

	targetPath := filepath.Join(root, "private-target")
	if err := os.Mkdir(targetPath, 0o700); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(root, "isolated-link")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatal(err)
	}
	if _, err := configureIsolatedUserDataPath(setPath, linkPath); err == nil {
		t.Fatal("symlinked isolated user-data directory was accepted")
	}
	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if targetInfo.Mode().Perm() != 0o700 {
		t.Fatalf("symlink rejection changed target directory permissions to %o", targetInfo.Mode().Perm())
	}
}

func TestIsolatedDevelopmentStoresSkipLegacyDefaults(t *testing.T) {
	userData := filepath.Join(t.TempDir(), "OrangeCount Go UI Isolated Development")
	legacyCalls := 0
	preferenceStore, presetStore := openApplicationStores(userData, true, func(string) (preferences.LegacyDefaults, error) {
		legacyCalls++
		return nil, errors.New("isolated development must not access production defaults")
	}, nil)
	if preferenceStore == nil || presetStore == nil || legacyCalls != 0 {
		t.Fatalf("isolated startup stores=%v/%v legacy calls=%d", preferenceStore, presetStore, legacyCalls)
	}
	paths, err := preferenceStore.RecentLedgerPaths()
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("isolated development imported recent paths: %q", paths)
	}
	checkPrivateDirectory(t, userData)
}

func startupFixture(t *testing.T, relative string) string {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	path, err := filepath.Abs(filepath.Join(root, "testdata", "fixtures", filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func checkPrivateFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("startup file %s permissions = %o, want private regular file", path, info.Mode().Perm())
	}
}

func checkPrivateDirectory(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("startup data directory %s permissions = %o, want private directory", path, info.Mode().Perm())
	}
}
