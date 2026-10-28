// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDivergenceIsDetectedEndToEnd runs the whole pipeline for one fixture: the
// oracle sees the pristine ledger and the Go engine sees a ledger with one extra
// transaction. The comparison must name the divergence in the ordered entries,
// the per-transaction inventories and the period income. Without a test like
// this, a comparison that silently returned no differences would look identical
// to a passing engine.
func TestDivergenceIsDetectedEndToEnd(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	original, err := os.ReadFile(filepath.Join(root, "testdata", "fixtures", "gen", "424242-plain.bean"))
	if err != nil {
		t.Skip("no promoted corpus; run go run ./tools/genprediff -promote-all")
	}
	if err := os.WriteFile(filepath.Join(work, "424242-plain.bean"), original, 0o644); err != nil {
		t.Fatal(err)
	}
	runOracle(root, work, filepath.Join(work, "oracle.json"))
	encoded, err := os.ReadFile(filepath.Join(work, "oracle.json"))
	if err != nil {
		t.Fatal(err)
	}
	records, err := readOracleRecords(encoded)
	if err != nil {
		t.Fatal(err)
	}
	record, ok := records["424242-plain.bean"]
	if !ok {
		t.Fatalf("oracle record missing for the corpus fixture")
	}

	mutated := string(original) + "\n2020-01-09 * \"Extra\"\n  Expenses:Food  7.00 USD\n  Assets:Cash  -7.00 USD\n"
	mutatedPath := filepath.Join(work, "424242-plain.bean")
	if err := os.WriteFile(mutatedPath, []byte(mutated), 0o644); err != nil {
		t.Fatal(err)
	}
	have, err := buildGoDimensions(mutatedPath)
	if err != nil {
		t.Fatal(err)
	}
	differences := dimensionDifferences(record, have)
	for _, want := range []string{"ordered-entries", "prefix", "period-income"} {
		if !containsPrefix(differences, want) {
			t.Fatalf("expected a %q difference, got %v", want, differences)
		}
	}
}
