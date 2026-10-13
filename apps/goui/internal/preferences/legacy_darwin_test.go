// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

//go:build darwin

package preferences

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"orangecount/apps/goui/internal/platform"
)

func TestOpenLegacyDefaultsReadsFoundationValues(t *testing.T) {
	if err := loadLegacyFoundation(); err != nil {
		t.Fatal(err)
	}
	domain := fmt.Sprintf("org.orangecount.goui.test.%d", os.Getpid())
	var defaults objc.ID
	var suiteName objc.ID
	var suiteText string
	withLegacyAutoreleasePool(func() {
		suiteName = testLegacyString(domain)
		suiteText = legacyString(suiteName)
		defaults = objc.ID(objc.GetClass("NSUserDefaults")).Send(legacySelector("alloc")).Send(
			legacySelector("initWithSuiteName:"), suiteName,
		)
		if defaults == 0 {
			t.Fatalf("could not create isolated defaults suite (suiteName=%#x text=%q)", suiteName, suiteText)
		}

		recents := objc.ID(objc.GetClass("NSMutableArray")).Send(legacySelector("new"))
		recents.Send(legacySelector("addObject:"), testLegacyString("/tmp/first.bean"))
		recents.Send(legacySelector("addObject:"), objc.ID(objc.GetClass("NSNumber")).Send(legacySelector("numberWithInt:"), 7))
		recents.Send(legacySelector("addObject:"), testLegacyString("/tmp/second.bean"))
		defaults.Send(legacySelector("setObject:forKey:"), recents, testLegacyString(LegacyRecentLedgersKey))
		recents.Send(legacySelector("release"))

		defaults.Send(legacySelector("setObject:forKey:"), testLegacyString("v3"), testLegacyString("org.orangecount.semantics./tmp/first.bean"))
		legacyBookmarkBytes := []byte{0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b}
		legacyBookmarkData := objc.ID(objc.GetClass("NSData")).Send(legacySelector("dataWithBytes:length:"), unsafe.Pointer(unsafe.SliceData(legacyBookmarkBytes)), uintptr(len(legacyBookmarkBytes)))
		defaults.Send(legacySelector("setObject:forKey:"), legacyBookmarkData, testLegacyString("org.orangecount.goui.test.bookmark-data"))
		reading := objc.ID(objc.GetClass("NSMutableDictionary")).Send(legacySelector("new"))
		reading.Send(legacySelector("setObject:forKey:"), testLegacyString("USD"), testLegacyString("currency"))
		reading.Send(legacySelector("setObject:forKey:"), testLegacyString("market-value"), testLegacyString("valuation"))
		reading.Send(legacySelector("setObject:forKey:"), objc.ID(objc.GetClass("NSNumber")).Send(legacySelector("numberWithInt:"), 2), testLegacyString("ignored-number"))
		defaults.Send(legacySelector("setObject:forKey:"), reading, testLegacyString("org.orangecount.reading./tmp/first.bean"))
		reading.Send(legacySelector("release"))
		objc.Send[bool](defaults, legacySelector("synchronize"))
	})
	if defaults == 0 {
		t.Fatal("isolated defaults suite was not initialized")
	}
	defer func() {
		withLegacyAutoreleasePool(func() {
			defaults.Send(legacySelector("removePersistentDomainForName:"), testLegacyString(domain))
			objc.Send[bool](defaults, legacySelector("synchronize"))
		})
		defaults.Send(legacySelector("release"))
	}()

	reader, err := OpenLegacyDefaults(domain)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.(*darwinLegacyDefaults).Close()

	paths, found, err := reader.StringArray(LegacyRecentLedgersKey)
	if err != nil || !found || !reflect.DeepEqual(paths, []string{"/tmp/first.bean", "/tmp/second.bean"}) {
		t.Fatalf("StringArray = %v, %v, %v", paths, found, err)
	}
	profile, found, err := reader.String("org.orangecount.semantics./tmp/first.bean")
	if err != nil || !found || profile != "v3" {
		t.Fatalf("String = %q, %v, %v", profile, found, err)
	}
	reading, found, err := reader.StringDictionary("org.orangecount.reading./tmp/first.bean")
	if err != nil || !found || !reflect.DeepEqual(reading, map[string]string{"currency": "USD", "valuation": "market-value"}) {
		t.Fatalf("StringDictionary = %v, %v, %v", reading, found, err)
	}
	bookmarkData, found, err := reader.Data("org.orangecount.goui.test.bookmark-data")
	if err != nil || !found || !reflect.DeepEqual(bookmarkData, []byte{0x62, 0x6f, 0x6f, 0x6b, 0x6d, 0x61, 0x72, 0x6b}) {
		t.Fatalf("Data = %v, %v, %v", bookmarkData, found, err)
	}

	// Reading through the adapter must leave the source suite unchanged.
	withLegacyAutoreleasePool(func() {
		got := objc.Send[objc.ID](defaults, legacySelector("objectForKey:"), testLegacyString("org.orangecount.semantics./tmp/first.bean"))
		if legacyString(got) != "v3" {
			t.Errorf("legacy value changed after read: %q", legacyString(got))
		}
	})
}

func TestLegacyDefaultsMigrationEndToEndIsReadOnly(t *testing.T) {
	root := t.TempDir()
	ledgerPath := filepath.Join(root, "legacy.bean")
	writeLedger(t, ledgerPath)
	canonicalLedger, err := canonicalLedgerPath(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	attachmentsPath := filepath.Join(root, "attachments")
	if err := os.Mkdir(attachmentsPath, 0o700); err != nil {
		t.Fatal(err)
	}
	bookmark, err := platform.CreateDirectoryBookmark(attachmentsPath)
	if err != nil {
		t.Fatal(err)
	}
	legacyRoots, err := json.Marshal(legacyAttachmentRootConfiguration{
		Generation: 9, Directories: []platform.DirectoryBookmark{bookmark},
	})
	if err != nil {
		t.Fatal(err)
	}

	domain := fmt.Sprintf("org.orangecount.goui.test.migration.%d.%d", os.Getpid(), time.Now().UnixNano())
	var defaults objc.ID
	withLegacyAutoreleasePool(func() {
		suiteName := testLegacyString(domain)
		defaults = objc.ID(objc.GetClass("NSUserDefaults")).Send(legacySelector("alloc")).Send(
			legacySelector("initWithSuiteName:"), suiteName,
		)
		if defaults == 0 {
			t.Fatal("could not create isolated defaults suite")
		}

		recents := objc.ID(objc.GetClass("NSMutableArray")).Send(legacySelector("new"))
		recents.Send(legacySelector("addObject:"), testLegacyString(canonicalLedger))
		defaults.Send(legacySelector("setObject:forKey:"), recents, testLegacyString(LegacyRecentLedgersKey))
		recents.Send(legacySelector("release"))
		defaults.Send(legacySelector("setObject:forKey:"), testLegacyString("v3"), testLegacyString("org.orangecount.semantics."+canonicalLedger))

		reading := objc.ID(objc.GetClass("NSMutableDictionary")).Send(legacySelector("new"))
		reading.Send(legacySelector("setObject:forKey:"), testLegacyString("USD"), testLegacyString("currency"))
		reading.Send(legacySelector("setObject:forKey:"), testLegacyString("market-value"), testLegacyString("valuation"))
		defaults.Send(legacySelector("setObject:forKey:"), reading, testLegacyString("org.orangecount.reading."+canonicalLedger))
		reading.Send(legacySelector("release"))

		rootsData := objc.ID(objc.GetClass("NSData")).Send(legacySelector("dataWithBytes:length:"), unsafe.Pointer(unsafe.SliceData(legacyRoots)), uintptr(len(legacyRoots)))
		defaults.Send(legacySelector("setObject:forKey:"), rootsData, testLegacyString(LegacyAttachmentRootsKeyPrefix+canonicalLedger))
		defaults.Send(legacySelector("setObject:forKey:"), testLegacyString("preserved"), testLegacyString("org.orangecount.unknown.migration-test"))
		objc.Send[bool](defaults, legacySelector("synchronize"))
	})
	if defaults == 0 {
		t.Fatal("isolated defaults suite was not initialized")
	}
	defer func() {
		withLegacyAutoreleasePool(func() {
			defaults.Send(legacySelector("removePersistentDomainForName:"), testLegacyString(domain))
			objc.Send[bool](defaults, legacySelector("synchronize"))
		})
		defaults.Send(legacySelector("release"))
	}()

	reader, err := OpenLegacyDefaults(domain)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.(*darwinLegacyDefaults).Close()

	store := NewStore(filepath.Join(root, "go-support", "preferences.json"))
	if err := store.MigrateLegacy(reader); err != nil {
		t.Fatal(err)
	}
	recents, err := store.RecentLedgerPaths()
	if err != nil || !reflect.DeepEqual(recents, []string{canonicalLedger}) {
		t.Fatalf("migrated recents = %v, %v", recents, err)
	}
	item, err := store.ForLedger(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if item.SemanticsProfile != "v3" || item.Currency != "USD" || item.Valuation != "market-value" {
		t.Fatalf("migrated reading preferences = %+v", item)
	}
	roots, generation, err := store.AttachmentRoots(ledgerPath)
	if err != nil || generation != 9 || len(roots) != 1 || roots[0].Path != bookmark.Path || !reflect.DeepEqual(roots[0].Data, bookmark.Data) {
		t.Fatalf("migrated attachment roots = %+v, generation %d, err %v", roots, generation, err)
	}
	resolved, err := platform.ResolveDirectoryBookmark(roots[0])
	if err != nil || resolved != bookmark.Path {
		t.Fatalf("migrated bookmark resolves to %q, %v; want %q", resolved, err, bookmark.Path)
	}
	doc, err := store.Load()
	if err != nil || doc.LegacyMigrationVersion != CurrentLegacyMigrationVersion {
		t.Fatalf("migration version = %d, %v", doc.LegacyMigrationVersion, err)
	}

	// The adapter and migration read the old domain but never write to it.
	oldRoots, found, err := reader.Data(LegacyAttachmentRootsKeyPrefix + canonicalLedger)
	if err != nil || !found || !reflect.DeepEqual(oldRoots, legacyRoots) {
		t.Fatalf("legacy bookmark data changed: found=%v err=%v", found, err)
	}
	oldProfile, found, err := reader.String("org.orangecount.semantics." + canonicalLedger)
	if err != nil || !found || oldProfile != "v3" {
		t.Fatalf("legacy semantics changed: profile=%q found=%v err=%v", oldProfile, found, err)
	}
	oldRecents, found, err := reader.StringArray(LegacyRecentLedgersKey)
	if err != nil || !found || !reflect.DeepEqual(oldRecents, []string{canonicalLedger}) {
		t.Fatalf("legacy recents changed: recents=%v found=%v err=%v", oldRecents, found, err)
	}
	oldReading, found, err := reader.StringDictionary("org.orangecount.reading." + canonicalLedger)
	if err != nil || !found || !reflect.DeepEqual(oldReading, map[string]string{"currency": "USD", "valuation": "market-value"}) {
		t.Fatalf("legacy reading preferences changed: reading=%v found=%v err=%v", oldReading, found, err)
	}
	withLegacyAutoreleasePool(func() {
		unknown := objc.Send[objc.ID](defaults, legacySelector("objectForKey:"), testLegacyString("org.orangecount.unknown.migration-test"))
		if got := legacyString(unknown); got != "preserved" {
			t.Errorf("unknown legacy key changed: %q", got)
		}
	})

	// A completed migration ignores later edits to the old suite.
	withLegacyAutoreleasePool(func() {
		defaults.Send(legacySelector("setObject:forKey:"), testLegacyString("legacy"), testLegacyString("org.orangecount.semantics."+canonicalLedger))
		objc.Send[bool](defaults, legacySelector("synchronize"))
	})
	if err := store.MigrateLegacy(reader); err != nil {
		t.Fatal(err)
	}
	item, err = store.ForLedger(ledgerPath)
	if err != nil || item.SemanticsProfile != "v3" {
		t.Fatalf("completed migration re-imported old preferences: %+v, %v", item, err)
	}
}

func testLegacyString(value string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(legacySelector("stringWithUTF8String:"), value+"\x00")
}
