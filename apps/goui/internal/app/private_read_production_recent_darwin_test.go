// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

//go:build darwin

package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orangecount/apps/goui/internal/preferences"
)

// TestPrivateLedgerUIReadOnlyProjectionFromProductionRecent is separately
// opt-in. It selects the first readable ledger from the existing production
// defaults recents, then delegates to the privacy-safe parity harness. Neither
// the path nor ledger values are emitted by this wrapper or the harness.
func TestPrivateLedgerUIReadOnlyProjectionFromProductionRecent(t *testing.T) {
	if os.Getenv("ORANGECOUNT_TEST_PRIVATE_RECENT_LEDGER") != "1" {
		t.Skip("set ORANGECOUNT_TEST_PRIVATE_RECENT_LEDGER=1 to compare one production recent ledger read-only")
	}
	defaults, err := preferences.OpenLegacyDefaults(preferences.LegacyApplicationDomain)
	if err != nil {
		t.Fatalf("open production defaults: %v", err)
	}
	defer defaults.(interface{ Close() }).Close()

	recents, found, err := defaults.StringArray(preferences.LegacyRecentLedgersKey)
	if err != nil {
		t.Fatal(err)
	}
	if !found || len(recents) == 0 {
		t.Skip("production defaults have no recent ledger; source details omitted")
	}
	selected := ""
	for _, recent := range recents {
		candidate := strings.TrimSpace(recent)
		if candidate == "~" || strings.HasPrefix(candidate, "~/") {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				continue
			}
			if candidate == "~" {
				candidate = home
			} else {
				candidate = filepath.Join(home, strings.TrimPrefix(candidate, "~/"))
			}
		}
		if candidate == "" || !filepath.IsAbs(candidate) || filepath.Clean(candidate) != candidate {
			continue
		}
		info, statErr := os.Stat(candidate)
		if statErr == nil && info.Mode().IsRegular() {
			selected = candidate
			break
		}
	}
	if selected == "" {
		t.Skip("production defaults have no readable recent ledger; source details omitted")
	}

	profile := "legacy"
	if saved, ok, readErr := defaults.String("org.orangecount.semantics." + selected); readErr != nil {
		t.Fatal(readErr)
	} else if ok && saved == "v3" {
		profile = "v3"
	}
	t.Run("read-only-core-workspace-ui-parity", func(t *testing.T) {
		t.Setenv("ORANGECOUNT_PRIVATE_LEDGER", selected)
		t.Setenv("ORANGECOUNT_PRIVATE_PROFILE", profile)
		TestPrivateLedgerUIReadOnlyProjection(t)
	})
}
