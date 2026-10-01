// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"encoding/json"
	"orangecount/internal/compat"
	"os"
	"path/filepath"
	"testing"
)

func TestFullGateRejectsMissingAndMutatedGolden(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "txn-basic.bean")
	data, err := os.ReadFile("../../testdata/fixtures/v3-parity/txn-basic.bean")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture, data, 0600); err != nil {
		t.Fatal(err)
	}
	golden := t.TempDir()
	if result := buildFullReport([]string{dir}, golden); result.FailedCount != 1 {
		t.Fatal("missing golden passed")
	}
	data, err = os.ReadFile("../../testdata/golden/v3-full/txn-basic.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var value compat.FullGolden
	_ = json.Unmarshal(data, &value)
	value.State.Entries = append(value.State.Entries, value.State.Entries[0])
	data, _ = json.Marshal(value)
	if err := os.WriteFile(filepath.Join(golden, "txn-basic.golden.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if result := buildFullReport([]string{dir}, golden); result.FailedCount != 1 || result.CleanCount != 0 {
		t.Fatal("changed multiplicity passed")
	}
}
