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

// Each localized vocabulary uses the statutory terms of its jurisdiction and
// makes them exact equivalents of the English roots without any per-type
// option. The English names stay valid in the same ledger, so a localized
// ledger can adopt the scheme without renaming anything.
func TestLocalizedSchemesClassifyTheirRoots(t *testing.T) {
	tests := []struct {
		scheme AccountRootScheme
		roots  map[string]string
	}{
		{SchemeZhHans, map[string]string{
			"资产:钱包": "Assets", "负债:卡": "Liabilities", "权益:期初": "Equity",
			"收入:工资": "Income", "支出:餐饮": "Expenses",
		}},
		{SchemeZhHant, map[string]string{
			"資產:錢包": "Assets", "負債:卡": "Liabilities", "權益:期初": "Equity",
			"收入:薪水": "Income", "支出:餐飲": "Expenses",
		}},
		{SchemeJapanese, map[string]string{
			"資産:財布": "Assets", "負債:カード": "Liabilities", "純資産:期首": "Equity",
			"収益:給与": "Income", "費用:食費": "Expenses",
		}},
		{SchemeKorean, map[string]string{
			"자산:지갑": "Assets", "부채:카드": "Liabilities", "자본:기초": "Equity",
			"수익:급여": "Income", "비용:식비": "Expenses",
		}},
	}
	for _, test := range tests {
		types := Evaluation{Options: map[string]string{AccountRootSchemeOption: string(test.scheme)}}.AccountTypes()
		if types.Scheme != test.scheme {
			t.Errorf("%s: resolved scheme=%q", test.scheme, types.Scheme)
		}
		for account, want := range test.roots {
			if got := types.TypeOf(account); got != want {
				t.Errorf("%s: TypeOf(%q)=%q want %q", test.scheme, account, got, want)
			}
		}
		// The upstream English vocabulary stays valid under every scheme.
		for _, kind := range types.Keys() {
			if got := types.TypeOf(types.RootOf(kind) + ":Leaf"); got != kind {
				t.Errorf("%s: English root %s misclassified as %q", test.scheme, kind, got)
			}
			roots := types.RootsOf(kind)
			if len(roots) != 2 {
				t.Errorf("%s: %s roots=%v want 2 spellings", test.scheme, kind, roots)
			}
			if roots[0] != types.RootOf(kind) || roots[1] != localizedRoots[test.scheme][kind] {
				t.Errorf("%s: %s roots=%v order or alias wrong", test.scheme, kind, roots)
			}
		}
	}
}

// A localized vocabulary must not leak across schemes: 資産 is Japanese and 資產
// is Traditional Chinese, and neither belongs to the other scheme.
func TestLocalizedSchemesDoNotLeak(t *testing.T) {
	zhHant := Evaluation{Options: map[string]string{AccountRootSchemeOption: "zh-Hant"}}.AccountTypes()
	if got := zhHant.TypeOf("資産:財布"); got != "" {
		t.Errorf("zh-Hant accepted the Japanese root as %q", got)
	}
	japanese := Evaluation{Options: map[string]string{AccountRootSchemeOption: "ja"}}.AccountTypes()
	if got := japanese.TypeOf("資產:錢包"); got != "" {
		t.Errorf("ja accepted the Traditional root as %q", got)
	}
	if got := japanese.TypeOf("자산:지갑"); got != "" {
		t.Errorf("ja accepted the Korean root as %q", got)
	}
}

// The global scheme must stay exactly the upstream vocabulary: a localized root
// is an unclassified account, not a silently accepted alias.
func TestGlobalSchemeRejectsLocalizedRoots(t *testing.T) {
	types := (Evaluation{Options: map[string]string{AccountRootSchemeOption: "global"}}).AccountTypes()
	if types.Scheme != SchemeGlobal {
		t.Fatalf("scheme=%q want %q", types.Scheme, SchemeGlobal)
	}
	for _, account := range []string{"资产:钱包", "資產:錢包", "資産:財布", "자산:지갑"} {
		if got := types.TypeOf(account); got != "" {
			t.Errorf("global scheme classified %q as %q", account, got)
		}
	}
	if got := types.TypeOf("Assets:Cash"); got != "Assets" {
		t.Fatalf("global scheme lost the English root: %q", got)
	}
}

// Each scheme must supply five distinct names. A duplicated spelling inside one
// table would make classification ambiguous before the evaluator could report
// it, so the table itself is pinned here.
func TestLocalizedRootTablesAreDistinct(t *testing.T) {
	for scheme, roots := range localizedRoots {
		if len(roots) != 5 {
			t.Fatalf("%s table has %d entries", scheme, len(roots))
		}
		seen := map[string]string{}
		for kind, name := range roots {
			if name == "" {
				t.Errorf("%s: %s has an empty name", scheme, kind)
			}
			if !ValidRootName(name) {
				t.Errorf("%s: %s=%q is not a valid root name", scheme, kind, name)
			}
			if previous, ok := seen[name]; ok {
				t.Errorf("%s: %q maps to both %s and %s", scheme, name, previous, kind)
			}
			seen[name] = kind
			// A localized name must not equal its English canonical key, or the
			// scheme would silently reuse the upstream spelling.
			if name == kind {
				t.Errorf("%s: %s reuses the English name", scheme, kind)
			}
		}
	}
}

// The option value is typed by hand, so common spellings resolve to the same
// vocabulary while an unrecognized value is rejected rather than guessed.
func TestResolveRootSchemeAliases(t *testing.T) {
	cases := map[string]AccountRootScheme{
		"":                    SchemeGlobal,
		"global":              SchemeGlobal,
		"GLOBAL":              SchemeGlobal,
		" global ":            SchemeGlobal,
		"cn":                  SchemeZhHans,
		"zh-Hans":             SchemeZhHans,
		"zh":                  SchemeZhHans,
		"simplified":          SchemeZhHans,
		"简体中文":                SchemeZhHans,
		"zh-Hant":             SchemeZhHant,
		"zh-TW":               SchemeZhHant,
		"tw":                  SchemeZhHant,
		"traditional-chinese": SchemeZhHant,
		"繁體中文":                SchemeZhHant,
		"ja":                  SchemeJapanese,
		"jp":                  SchemeJapanese,
		"japanese":            SchemeJapanese,
		"日本語":                 SchemeJapanese,
		"ko":                  SchemeKorean,
		"kr":                  SchemeKorean,
		"korean":              SchemeKorean,
		"한국어":                 SchemeKorean,
	}
	for value, want := range cases {
		if got, ok := resolveRootScheme(value); !ok || got != want {
			t.Errorf("resolveRootScheme(%q)=%q,%v want %q,true", value, got, ok, want)
		}
	}
	for _, value := range []string{"xx", "zh-Hansx", "klingon", "1"} {
		if got, ok := resolveRootScheme(value); ok || got != SchemeGlobal {
			t.Errorf("resolveRootScheme(%q)=%q,%v want global,false", value, got, ok)
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
	valid := []string{"Assets", "A", "A-b", "资产", "資產", "資産", "자산", "Actifs", "A资产"}
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
