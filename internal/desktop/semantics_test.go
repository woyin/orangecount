// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package desktop

import (
	"orangecount/internal/ledger"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionOptionsRemainFixedAcrossReload(t *testing.T) {
	path := "../../testdata/fixtures/semantics/lifo-explicit.bean"
	s, err := OpenWithOptions(path, OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	before, _ := s.Poll()
	if before.SemanticsProfile != ledger.V3 || before.EvaluationRevision == "" {
		t.Fatalf("missing identity: %+v", before)
	}
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	after, _ := s.Poll()
	if before.SnapshotID != after.SnapshotID || after.SemanticsProfile != ledger.V3 {
		t.Fatalf("reload changed semantics: %+v", after)
	}
	if _, err := OpenWithOptions(path, OpenOptions{SemanticsProfile: "bogus"}); err == nil {
		t.Fatal("unknown profile opened")
	}
}

func TestMigrationPreviewCannotSwitchChangedInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	text, err := os.ReadFile("../../testdata/fixtures/semantics/lifo-implicit.bean")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, text, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := OpenWithOptions(path, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	preview, err := s.Read(Request{View: "migration"})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Migration == nil || preview.Migration.ChangeCounts["balances"] == 0 {
		t.Fatal("migration preview missing")
	}
	if err := os.WriteFile(path, append(text, []byte("\n; changed\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if next, err := OpenWithOptions(path, OpenOptions{SemanticsProfile: ledger.V3, InputIdentity: preview.Migration.InputIdentity}); err != ErrStale {
		if next != nil {
			next.Close()
		}
		t.Fatalf("stale preview accepted: %v", err)
	}
}
