// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package authoring

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"orangecount/internal/snapshot"
)

const validLedger = "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n"

func writerFixture(t *testing.T) (*Writer, *snapshot.Store, string, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(path, []byte(validLedger), 0o640); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(path)
	if built.Snapshot == nil {
		t.Fatalf("build failed: %+v", built.Diagnostics)
	}
	store := snapshot.NewStore(built.Snapshot)
	writer, err := NewWriter(store)
	if err != nil {
		t.Fatal(err)
	}
	return writer, store, path, built.Snapshot.ID
}

func TestWriterPublishesAndRevertsAppend(t *testing.T) {
	writer, store, path, id := writerFixture(t)
	change, _ := Append("main.bean", `2000-01-02 note Assets:Cash "x"`)
	result, err := writer.Publish(id, change)
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt == nil || result.Build.Snapshot == nil || result.Build.Snapshot.ID != store.Current().ID {
		t.Fatalf("result=%+v current=%+v", result, store.Current())
	}
	if result.Backup != "main.bean.orangecount.bak" {
		t.Fatalf("backup=%q", result.Backup)
	}
	backup, err := os.ReadFile(path + ".orangecount.bak")
	if err != nil || !bytes.Equal(backup, []byte(validLedger)) {
		t.Fatalf("backup=%q err=%v", backup, err)
	}
	reverted, err := writer.Revert(result.Build.Snapshot.ID, result.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, []byte(validLedger)) || reverted.Build.Snapshot.ID != store.Current().ID {
		t.Fatalf("file=%q current=%+v", got, store.Current())
	}
}

func TestWriterRejectsStaleSnapshot(t *testing.T) {
	writer, _, _, _ := writerFixture(t)
	change, _ := Append("main.bean", `2000-01-02 note Assets:Cash "x"`)
	if _, err := writer.Publish("stale", change); !errors.Is(err, ErrSnapshotChanged) {
		t.Fatalf("err=%v", err)
	}
}

func TestWriterRejectsDiskDrift(t *testing.T) {
	writer, _, path, id := writerFixture(t)
	if err := os.WriteFile(path, []byte(validLedger+"; external\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	change, _ := Append("main.bean", `2000-01-02 note Assets:Cash "x"`)
	if _, err := writer.Publish(id, change); !errors.Is(err, ErrSourceChanged) {
		t.Fatalf("err=%v", err)
	}
}

func TestWriterClassifiesReadFailureAsIO(t *testing.T) {
	writer, _, path, id := writerFixture(t)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	change, _ := Append("main.bean", `2000-01-02 note Assets:Cash "x"`)
	if _, err := writer.Publish(id, change); !errors.Is(err, ErrIO) {
		t.Fatalf("err=%v", err)
	}
}

func TestWriterRollsBackInvalidLedger(t *testing.T) {
	writer, store, path, id := writerFixture(t)
	change, _ := Replace("main.bean", []byte("not valid ledger\n"))
	result, err := writer.Publish(id, change)
	if !errors.Is(err, ErrValidation) || len(result.Build.Diagnostics) == 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, []byte(validLedger)) || store.Current().ID != id {
		t.Fatalf("file=%q id=%q", got, store.Current().ID)
	}
}

func TestWriterPublishesFullReplacement(t *testing.T) {
	writer, store, path, id := writerFixture(t)
	replacement := validLedger + "2000-01-02 note Assets:Cash \"replacement\"\n"
	change, _ := Replace("main.bean", []byte(replacement))
	result, err := writer.Publish(id, change)
	if err != nil {
		t.Fatal(err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil || string(got) != replacement {
		t.Fatalf("file=%q err=%v", got, readErr)
	}
	if result.Build.Snapshot == nil || result.Build.Snapshot.ID != store.Current().ID {
		t.Fatalf("result=%+v current=%+v", result, store.Current())
	}
}

func TestWriterRejectsTargetOutsideGraph(t *testing.T) {
	writer, _, _, id := writerFixture(t)
	change, _ := Replace("missing.bean", nil)
	if _, err := writer.Publish(id, change); !errors.Is(err, ErrTargetMissing) {
		t.Fatalf("err=%v", err)
	}
}

func TestWriterRejectsRevertAfterLaterPublication(t *testing.T) {
	writer, _, _, id := writerFixture(t)
	first, _ := Append("main.bean", `2000-01-02 note Assets:Cash "one"`)
	one, err := writer.Publish(id, first)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := Append("main.bean", `2000-01-03 note Assets:Cash "two"`)
	two, err := writer.Publish(one.Build.Snapshot.ID, second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Revert(two.Build.Snapshot.ID, one.Receipt); !errors.Is(err, ErrSnapshotChanged) {
		t.Fatalf("err=%v", err)
	}
}

func TestWriterSerializesConcurrentPublications(t *testing.T) {
	writer, _, _, id := writerFixture(t)
	changes := make([]Change, 2)
	changes[0], _ = Append("main.bean", `2000-01-02 note Assets:Cash "one"`)
	changes[1], _ = Append("main.bean", `2000-01-03 note Assets:Cash "two"`)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, change := range changes {
		wg.Add(1)
		go func(change Change) {
			defer wg.Done()
			_, err := writer.Publish(id, change)
			errs <- err
		}(change)
	}
	wg.Wait()
	close(errs)
	var success, stale int
	for err := range errs {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrSnapshotChanged):
			stale++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatalf("success=%d stale=%d", success, stale)
	}
}
