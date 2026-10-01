// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package snapshot_test

import (
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

func TestV3EffectiveMetadataPreservesParsedSourceAndLegacy(t *testing.T) {
	graph, err := source.LoadGraph("../../testdata/fixtures/query/metadata-stack.bean")
	if err != nil {
		t.Fatal(err)
	}
	v3 := snapshot.BuildGraphWithOptions(graph, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	legacy := snapshot.BuildGraphWithOptions(graph, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.Legacy}})
	if v3.Snapshot == nil || legacy.Snapshot == nil {
		t.Fatalf("builds: %v %v", v3.Diagnostics, legacy.Diagnostics)
	}
	if got := len(v3.Snapshot.Evaluation().Entries); got != 11 {
		t.Fatalf("source stack commands leaked into loaded entries: %d, want 11", got)
	}
	for _, file := range v3.Snapshot.Parsed() {
		for _, directive := range file.Directives {
			if tx, ok := directive.(*ledger.Transaction); ok && tx.Narration == "late outer" && len(tx.Meta) != 0 {
				t.Fatal("effective metadata changed parsed source")
			}
		}
	}
	for _, entry := range legacy.Snapshot.Evaluation().Entries {
		if tx, ok := entry.Directive.(*ledger.Transaction); ok && tx.Narration == "late outer" && len(tx.Meta) != 0 {
			t.Fatal("v3 correction changed legacy metadata")
		}
	}
}
