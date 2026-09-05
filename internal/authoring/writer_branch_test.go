// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package authoring

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"orangecount/internal/snapshot"
)

func TestNewWriterRejectsNilStore(t *testing.T) {
	if _, err := NewWriter(nil); err == nil {
		t.Fatal("nil store must be rejected")
	}
}

func TestChangeTargetReportsAddressedFile(t *testing.T) {
	change, err := Append("main.bean", "x\n")
	if err != nil {
		t.Fatal(err)
	}
	if change.Target() != "main.bean" {
		t.Fatalf("target=%q", change.Target())
	}
}

func TestPublishRejectsUnknownTargetAndRevertWithoutReceipt(t *testing.T) {
	writer, store, _, id := writerFixture(t)
	change, _ := Replace("outside.bean", []byte("x"))
	if _, err := writer.Publish(id, change); err == nil {
		t.Fatal("unknown target must fail")
	}
	// A revert without the publication receipt can never know what to restore.
	if _, err := writer.Revert(id, nil); err == nil {
		t.Fatal("revert without receipt must fail")
	}
	_ = store
}

func TestPublishValidationFailureRestoresSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.bean")
	valid := "2000-01-01 open Assets:Cash USD\n"
	if err := os.WriteFile(path, []byte(valid), 0o640); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(path)
	if built.Snapshot == nil {
		t.Fatalf("build=%+v", built.Diagnostics)
	}
	store := snapshot.NewStore(built.Snapshot)
	writer, err := NewWriter(store)
	if err != nil {
		t.Fatal(err)
	}
	// Appending a directive that breaks the ledger must fail validation, put
	// the original bytes back, and leave the previous snapshot published.
	change, _ := Append("main.bean", "this is not beancount\n")
	result, err := writer.Publish(built.Snapshot.ID, change)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err=%v", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil || string(got) != valid {
		t.Fatalf("source not restored: %q err=%v", got, readErr)
	}
	if store.Current().ID != built.Snapshot.ID {
		t.Fatal("previous snapshot must stay published")
	}
	if result.Backup == "" {
		t.Fatal("backup path must be reported")
	}
}
