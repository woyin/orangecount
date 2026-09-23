// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateCoversEveryAccountFamily runs the generator with every account
// family enabled so the emitted ledger exercises all account classes.
func TestGenerateCoversEveryAccountFamily(t *testing.T) {
	root := t.TempDir()
	result, err := Generate(root, Config{
		Wallets: 2, Reserves: 1, Receivables: 1, Investments: 2,
		Cards: 1, Loans: 1, SalesChannels: 1, ExpenseCategories: 3,
		ActivityEntries: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) == 0 || result.Transactions == 0 {
		t.Fatalf("result=%+v", result)
	}
	var contents []string
	for _, name := range result.Files {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		contents = append(contents, string(data))
	}
	text := strings.Join(contents, "\n")
	for _, family := range []string{"Assets:", "Expenses:", "Income:", "Equity:", "Liabilities:"} {
		if !strings.Contains(text, family) {
			t.Fatalf("family %s missing from generated ledger", family)
		}
	}
	if !strings.Contains(text, "balance ") {
		t.Fatal("generated ledger should carry balance assertions")
	}
}

// TestNormalizeConfigAppliesDefaults pins the option fill: zero/negative
// counts fall back to the deterministic defaults.
func TestNormalizeConfigAppliesDefaults(t *testing.T) {
	filled := normalizeConfig(Config{ActivityEntries: 7})
	if filled.ActivityEntries != 7 {
		t.Fatalf("explicit entries=%d", filled.ActivityEntries)
	}
	if filled.Wallets != DefaultConfig.Wallets || filled.Investments != DefaultConfig.Investments {
		t.Fatalf("defaults not applied: %+v", filled)
	}
}
