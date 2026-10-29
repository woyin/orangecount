// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package compat_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"orangecount/internal/compat"
)

// today() cannot be frozen, so its golden must carry the dynamic marker. A
// golden without it is compared literally and fails the day after capture, which
// is how this file's marker went missing: the comparator supported the marker
// from the start, but the generator never emitted it.
func TestTodayGoldenIsMarkedDynamic(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "golden", "inventory", "function.today.0.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("golden not present: %v", err)
	}
	var golden struct {
		Dynamic string `json:"dynamic"`
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("unparseable golden: %v", err)
	}
	if golden.Dynamic != "today" {
		t.Fatalf("function.today.0 must be marked dynamic=today, got %q", golden.Dynamic)
	}
}

func TestInventoryGoldenCapabilities(t *testing.T) {
	report, err := compat.CheckInventoryGolden("../..")
	if err != nil {
		t.Fatalf("inventory golden: %v", err)
	}
	encoded, _ := json.MarshalIndent(report, "", "  ")
	if path := os.Getenv("INVENTORY_REPORT"); path != "" {
		if err := os.WriteFile(path, encoded, 0o644); err != nil {
			t.Fatalf("write inventory report: %v", err)
		}
	}
	t.Logf("inventory parity: checked=%d verified=%d failed=%d", report.Checked, report.Verified, report.Failed)
	for _, failure := range report.Failures {
		t.Errorf("inventory-mismatch %s: %s %s", failure.ID, failure.Reason, failure.Detail)
	}
	if report.Failed != 0 || report.Verified != report.Checked {
		t.Fatalf("inventory parity failed: %d of %d verified", report.Verified, report.Checked)
	}
}
