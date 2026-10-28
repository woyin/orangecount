// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package ledger

import "testing"

// The five root names are configurable, so account classification must follow
// the configured names rather than hard-coded English literals. These tests pin
// that contract because a regression here silently empties every report.
func TestAccountTypesDefaultToUpstreamRoots(t *testing.T) {
	types := Evaluation{Options: map[string]string{}}.AccountTypes()
	want := DefaultAccountTypes()
	if types != want {
		t.Fatalf("default types=%+v want=%+v", types, want)
	}
	for _, kind := range types.Keys() {
		if got := types.TypeOf(types.RootOf(kind) + ":Leaf"); got != kind {
			t.Fatalf("classify %s leaf: got %q want %q", kind, got, kind)
		}
	}
}

func TestAccountTypesFollowNameOptions(t *testing.T) {
	evaluation := Evaluation{Options: map[string]string{
		"name_assets":      "资产",
		"name_liabilities": "负债",
		"name_equity":      "权益",
		"name_income":      "收入",
		"name_expenses":    "支出",
	}}
	types := evaluation.AccountTypes()
	cases := map[string]string{
		"资产:钱包:微信": "Assets",
		"负债:信用卡":   "Liabilities",
		"权益:期初":    "Equity",
		"收入:工资":    "Income",
		"支出:餐饮":    "Expenses",
		// The default English roots are no longer configured, so they must not
		// classify: guessing would move balances into the wrong statement.
		"Assets:Cash": "",
		"Acme:Cash":   "",
	}
	for account, want := range cases {
		if got := types.TypeOf(account); got != want {
			t.Errorf("TypeOf(%q)=%q want %q", account, got, want)
		}
	}
}

func TestAccountTypesIgnoreInvalidRootValues(t *testing.T) {
	// A value that is not a valid root name must not silently rename a
	// statement; the evaluator reports it as an option error instead.
	for _, value := range []string{"", "a:B", "assets", "1Assets", "As sets"} {
		evaluation := Evaluation{Options: map[string]string{"name_assets": value}}
		if got := evaluation.AccountTypes().Assets; got != "Assets" {
			t.Errorf("invalid name_assets %q changed the root to %q", value, got)
		}
	}
}

func TestValidRootNameMirrorsUpstreamAccountType(t *testing.T) {
	valid := []string{"Assets", "A", "A-b", "资产", "資産", "Actifs", "A资产"}
	for _, name := range valid {
		if !ValidRootName(name) {
			t.Errorf("valid root rejected: %q", name)
		}
	}
	invalid := []string{"", ":", "assets", "1Assets", "A:B", "A B", "A_B", "-A"}
	for _, name := range invalid {
		if ValidRootName(name) {
			t.Errorf("invalid root accepted: %q", name)
		}
	}
}
