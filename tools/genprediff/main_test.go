// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orangecount/internal/compat"
)

// TestPromotedCorpusReplaysAgainstRecordedOracle keeps the promoted
// generative-differential corpus alive: every paired .bean/.json in
// testdata/fixtures/gen must still evaluate to the recorded oracle state, so a
// regression can never sit in the tree as dead data.
func TestPromotedCorpusReplaysAgainstRecordedOracle(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "testdata", "fixtures", "gen")
	inputs, err := filepath.Glob(filepath.Join(dir, "*.bean"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatalf("no promoted corpus in %s; run go run ./tools/genprediff -promote-all", dir)
	}
	for _, input := range inputs {
		stem := strings.TrimSuffix(input, ".bean")
		encoded, err := os.ReadFile(stem + ".json")
		if err != nil {
			t.Fatalf("%s: missing recorded difference: %v", filepath.Base(input), err)
		}
		var recorded struct {
			Oracle compat.FullGolden `json:"oracle"`
		}
		if err := json.Unmarshal(encoded, &recorded); err != nil {
			t.Fatalf("%s: %v", filepath.Base(stem), err)
		}
		have, buildErr := compat.BuildFullSide(input)
		if buildErr != nil && have.Valid {
			t.Fatalf("%s: build: %v", filepath.Base(input), buildErr)
		}
		differences := compat.FullGoldenDifferences(normalizeApproved(recorded.Oracle), normalizeApproved(have))
		differences = append(differences, prefixDifferences(recorded.Oracle, have)...)
		if len(differences) != 0 {
			t.Fatalf("%s: %s", filepath.Base(input), strings.Join(differences, ","))
		}
	}
}
