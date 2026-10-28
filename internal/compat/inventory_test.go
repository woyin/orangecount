// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package compat_test

import (
	"encoding/json"
	"os"
	"testing"

	"orangecount/internal/compat"
)

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
