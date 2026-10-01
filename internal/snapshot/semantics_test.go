// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package snapshot

import (
	"orangecount/internal/ledger"
	"testing"
)

func TestProfilesHaveDistinctStableIdentities(t *testing.T) {
	path := "../../testdata/fixtures/semantics/lifo-explicit.bean"
	implicit := Build(path)
	legacy := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.Legacy}})
	v3 := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	for _, result := range []BuildResult{implicit, legacy, v3} {
		if result.Snapshot == nil {
			t.Fatalf("build failed: %+v", result.Diagnostics)
		}
	}
	if implicit.Snapshot.ID != legacy.Snapshot.ID || legacy.Snapshot.ID == v3.Snapshot.ID {
		t.Fatalf("profile identities: implicit=%s legacy=%s v3=%s", implicit.Snapshot.ID, legacy.Snapshot.ID, v3.Snapshot.ID)
	}
	if got := v3.Snapshot.Evaluation(); got.SemanticsProfile != ledger.V3 || got.EvaluationRevision == "" {
		t.Fatalf("profile metadata missing: %+v", got)
	}
	bad := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: "unknown"}})
	if bad.Snapshot != nil || len(bad.Diagnostics) == 0 {
		t.Fatal("unknown profile published")
	}
}

func TestStoreKeepsProfileAcrossImplicitReloadAndRejectsSwitch(t *testing.T) {
	path := "../../testdata/fixtures/semantics/lifo-explicit.bean"
	v3 := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}).Snapshot
	store := NewStore(v3)
	if result := store.Reload(path, BuildOptions{}); result.Snapshot == nil || result.Snapshot.ID != v3.ID {
		t.Fatalf("implicit reload changed profile: %+v", result)
	}
	if result := store.Reload(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.Legacy}}); result.Err == nil || store.Current().ID != v3.ID {
		t.Fatalf("profile switched in a live store: %+v", result)
	}
	if store.Publish(Build(path).Snapshot, nil) {
		t.Fatal("published another profile")
	}
}

func TestPublishedCostAndBookedPostingsAreDefensiveCopies(t *testing.T) {
	s := BuildWithOptions("../../testdata/fixtures/semantics/lifo-implicit.bean", BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}).Snapshot
	e := s.Evaluation()
	e.Accounts["Assets:Stock"].Positions[0].Cost.Date.Raw = "changed"
	for _, record := range e.Entries {
		if tx, ok := record.Directive.(*ledger.Transaction); ok {
			tx.Postings[0].Units.Number.Rat.SetInt64(999)
			tx.Postings[0].Account = "changed"
		}
	}
	after := s.Evaluation()
	if after.Accounts["Assets:Stock"].Positions[0].Cost.Date.Raw == "changed" {
		t.Fatal("published lot mutated through a copy")
	}
	for _, record := range after.Entries {
		if tx, ok := record.Directive.(*ledger.Transaction); ok && (tx.Postings[0].Account == "changed" || tx.Postings[0].Units.Number.Rat.Num().Int64() == 999) {
			t.Fatal("published posting mutated through a copy")
		}
	}
}
