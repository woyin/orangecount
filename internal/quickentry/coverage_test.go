// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package quickentry

import (
	"strings"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/web/favaadapter"
)

// The tests in this file exercise the compiler's and profile resolver's
// error and edge branches directly, complementing the happy-path tests in
// compiler_test.go.

func compileOne(t *testing.T, req CompileRequest) LineResult {
	t.Helper()
	results := Compile(req)
	if len(results) != 1 {
		t.Fatalf("want exactly 1 result, got %d: %+v", len(results), results)
	}
	return results[0]
}

func TestCompileRejectsMalformedBatchHeaders(t *testing.T) {
	cases := []struct {
		name    string
		req     CompileRequest
		wantErr string
	}{
		{
			name:    "date not ISO shaped",
			req:     CompileRequest{Text: "28 CNY @a -> @b", Date: "2026/08/12"},
			wantErr: "E-QUICK-DATE",
		},
		{
			name:    "date calendar invalid",
			req:     CompileRequest{Text: "28 CNY @a -> @b", Date: "2026-02-30"},
			wantErr: "E-QUICK-DATE",
		},
		{
			name:    "flag neither * nor !",
			req:     CompileRequest{Text: "28 CNY @a -> @b", Date: "2026-08-12", Flag: "X"},
			wantErr: "E-QUICK-FLAG",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := compileOne(t, tc.req)
			if len(result.Errors) == 0 || result.Errors[0].Code != tc.wantErr {
				t.Fatalf("errors=%+v want code %s", result.Errors, tc.wantErr)
			}
		})
	}
}

func TestCompileFlagBangIsAccepted(t *testing.T) {
	result := compileOne(t, CompileRequest{
		Text: "28 CNY @Assets:WeChat -> @Expenses:Food", Date: "2026-08-12", Flag: "!",
	})
	if len(result.Errors) != 0 || result.Entry == nil {
		t.Fatalf("errors=%+v entry=%v", result.Errors, result.Entry)
	}
	if result.Entry.Flag != "!" {
		t.Fatalf("flag=%s want !", result.Entry.Flag)
	}
}

func TestCompileEmptyBatchReportsEmptyLine(t *testing.T) {
	result := compileOne(t, CompileRequest{Text: "  \n\t\n", Date: "2026-08-12"})
	if len(result.Errors) != 1 || result.Errors[0].Code != "E-QUICK-EMPTY" {
		t.Fatalf("errors=%+v", result.Errors)
	}
}

const profileLedger = `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Expenses:Food CNY
2000-01-01 open Equity:Opening CNY
2000-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat
2000-01-01 custom "orangecount.quick-account.v1" "餐饮" Expenses:Food
2000-01-01 custom "orangecount.quick-template.v1" "午餐"
  destination: "Expenses:Food"
  currency: "CNY"
  payee: "食堂"
  narration: "默认描述"
  tags: "food noon"
  links: "lunch-1"
`

func TestTemplateInvocationOverridesAndDefaults(t *testing.T) {
	evaluation := evalFromText(t, profileLedger)
	result := compileOne(t, CompileRequest{
		Text: "午餐 12 @微信 : 自定义 #extra ^extra-link", Date: "2026-08-12", Evaluation: evaluation,
	})
	if len(result.Errors) != 0 || result.Entry == nil {
		t.Fatalf("errors=%+v entry=%v", result.Errors, result.Entry)
	}
	entry := result.Entry
	if entry.Payee != "食堂" {
		t.Errorf("payee=%s (template default lost)", entry.Payee)
	}
	if entry.Narration != "自定义" {
		t.Errorf("narration=%s (override ignored)", entry.Narration)
	}
	wantTags := []string{"food", "noon", "extra"}
	if strings.Join(entry.Tags, ",") != strings.Join(wantTags, ",") {
		t.Errorf("tags=%v want %v", entry.Tags, wantTags)
	}
	wantLinks := []string{"lunch-1", "extra-link"}
	if strings.Join(entry.Links, ",") != strings.Join(wantLinks, ",") {
		t.Errorf("links=%v want %v", entry.Links, wantLinks)
	}
	// @微信 fills the template's unfilled source role.
	if entry.Postings[0].Account != "Assets:WeChat" || entry.Postings[1].Account != "Expenses:Food" {
		t.Errorf("postings=%+v", entry.Postings)
	}
}

func TestTemplateInvocationPayeeAliasFillsDestination(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Expenses:Food CNY
2000-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat
2000-01-01 custom "orangecount.quick-account.v1" "餐饮" Expenses:Food
2000-01-01 custom "orangecount.quick-template.v1" "转"
  source: "Assets:WeChat"
`)
	result := compileOne(t, CompileRequest{
		Text: "转 5 CNY @餐饮", Date: "2026-08-12", Evaluation: evaluation,
	})
	if len(result.Errors) != 0 || result.Entry == nil {
		t.Fatalf("errors=%+v entry=%v", result.Errors, result.Entry)
	}
	if result.Entry.Postings[0].Account != "Assets:WeChat" || result.Entry.Postings[1].Account != "Expenses:Food" {
		t.Errorf("postings=%+v (alias should fill destination)", result.Entry.Postings)
	}
}

func TestTemplateInvocationErrorBranches(t *testing.T) {
	evaluation := evalFromText(t, profileLedger)
	cases := []struct {
		name    string
		text    string
		wantErr string
	}{
		{"missing amount", "午餐", "E-QUICK-AMOUNT"},
		{"unexpected word", "午餐 3 abc", "E-QUICK-TEMPLATE-TOKEN"},
		{"surprise token in rest", "午餐 3 CNY CNY", "E-QUICK-TEMPLATE-TOKEN"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := compileOne(t, CompileRequest{Text: tc.text, Date: "2026-08-12", Evaluation: evaluation})
			if len(result.Errors) == 0 || result.Errors[0].Code != tc.wantErr {
				t.Fatalf("errors=%+v want %s", result.Errors, tc.wantErr)
			}
		})
	}
}

func TestTemplateCurrencyResolutionChain(t *testing.T) {
	// Template without currency: falls back to operating currency, then to
	// an error when neither exists.
	evaluation := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Expenses:Food CNY
2000-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat
2000-01-01 custom "orangecount.quick-account.v1" "餐饮" Expenses:Food
2000-01-01 custom "orangecount.quick-template.v1" "无币种"
  source: "Assets:WeChat"
  destination: "Expenses:Food"
`)
	withOperating := compileOne(t, CompileRequest{Text: "无币种 5", Date: "2026-08-12", Evaluation: evaluation, OperatingCurrency: "CNY"})
	if len(withOperating.Errors) != 0 || withOperating.Entry.Postings[0].Currency != "CNY" {
		t.Fatalf("operating currency fallback errors=%+v", withOperating.Errors)
	}
	without := compileOne(t, CompileRequest{Text: "无币种 5", Date: "2026-08-12", Evaluation: evaluation})
	if len(without.Errors) == 0 || without.Errors[0].Code != "E-QUICK-CURRENCY" {
		t.Fatalf("errors=%+v want E-QUICK-CURRENCY", without.Errors)
	}
	// A template may itself declare an invalid currency; the resolved value
	// then fails the currency shape check.
	badCurrency := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Expenses:Food CNY
2000-01-01 custom "orangecount.quick-template.v1" "坏币种"
  source: "Assets:WeChat"
  destination: "Expenses:Food"
  currency: "x"
`)
	invalid := compileOne(t, CompileRequest{Text: "坏币种 5", Date: "2026-08-12", Evaluation: badCurrency})
	if len(invalid.Errors) == 0 || invalid.Errors[0].Code != "E-QUICK-CURRENCY" {
		t.Fatalf("errors=%+v want E-QUICK-CURRENCY (invalid shape)", invalid.Errors)
	}
}

func TestTemplateWithoutEndpointsCannotResolve(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 custom "orangecount.quick-template.v1" "半成品"
  currency: "CNY"
`)
	result := compileOne(t, CompileRequest{Text: "半成品 5", Date: "2026-08-12", Evaluation: evaluation})
	if len(result.Errors) == 0 || result.Errors[0].Code != "E-QUICK-SOURCE" {
		t.Fatalf("errors=%+v want E-QUICK-SOURCE", result.Errors)
	}
	withSource := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 custom "orangecount.quick-template.v1" "半成品"
  source: "Assets:WeChat"
  currency: "CNY"
`)
	result = compileOne(t, CompileRequest{Text: "半成品 5", Date: "2026-08-12", Evaluation: withSource})
	if len(result.Errors) == 0 || result.Errors[0].Code != "E-QUICK-DEST" {
		t.Fatalf("errors=%+v want E-QUICK-DEST", result.Errors)
	}
}

func TestExplicitFormTokenErrors(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		wantErr string
	}{
		{"missing arrow", "28 CNY @Assets:WeChat @Expenses:Food", "E-QUICK-ARROW"},
		{"too many left tokens", "28 CNY EXTRA @Assets:WeChat -> @Expenses:Food", "E-QUICK-LEFT-TOKENS"},
		{"two source aliases", "28 @Assets:WeChat @Assets:Cash -> @Expenses:Food", "E-QUICK-SOURCE"},
		{"unexpected left token", "#tag -> @Expenses:Food", "E-QUICK-LEFT-TOKENS"},
		{"missing amount", "@Assets:WeChat -> @Expenses:Food", "E-QUICK-AMOUNT"},
		{"non decimal amount", "abc CNY @Assets:WeChat -> @Expenses:Food", "E-QUICK-AMOUNT"},
		{"two destinations", "28 @Assets:WeChat -> @Expenses:Food @Expenses:More", "E-QUICK-DEST"},
		{"unexpected right token", "28 @Assets:WeChat -> @Expenses:Food bogus", "E-QUICK-RIGHT-TOKENS"},
		{"missing source alias", "28 -> @Expenses:Food", "E-QUICK-SOURCE"},
		{"missing dest alias", "28 @Assets:WeChat ->", "E-QUICK-DEST"},
		{"unknown source alias", "28 CNY @nope -> @Expenses:Food", "E-QUICK-SOURCE"},
		{"unknown dest alias", "28 CNY @Assets:WeChat -> @nope", "E-QUICK-DEST"},
		{"invalid currency", "28 x! @Assets:WeChat -> @Expenses:Food", "E-QUICK-CURRENCY"},
		{"no currency inferable", "28 @Assets:WeChat -> @Expenses:Food", "E-QUICK-CURRENCY"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := compileOne(t, CompileRequest{Text: tc.text, Date: "2026-08-12"})
			if len(result.Errors) == 0 || result.Errors[0].Code != tc.wantErr {
				t.Fatalf("errors=%+v want %s", result.Errors, tc.wantErr)
			}
		})
	}
}

func TestExplicitFormAcceptsFullAccountNames(t *testing.T) {
	result := compileOne(t, CompileRequest{
		Text: "28 CNY @Assets:WeChat -> @Expenses:Food ^link-9 : 便签",
		Date: "2026-08-12",
	})
	if len(result.Errors) != 0 || result.Entry == nil {
		t.Fatalf("errors=%+v", result.Errors)
	}
	if len(result.Entry.Links) != 1 || result.Entry.Links[0] != "link-9" {
		t.Errorf("links=%v", result.Entry.Links)
	}
	if result.Entry.Narration != "便签" {
		t.Errorf("narration=%s", result.Entry.Narration)
	}
}

func TestExplicitFormInvalidAccountShapedAliasRejected(t *testing.T) {
	result := compileOne(t, CompileRequest{Text: "28 CNY @lower:case -> @Expenses:Food", Date: "2026-08-12"})
	if len(result.Errors) == 0 || result.Errors[0].Code != "E-QUICK-SOURCE" {
		t.Fatalf("errors=%+v want E-QUICK-SOURCE", result.Errors)
	}
}

func TestDetectDuplicatesNilEvaluationAndErrorRows(t *testing.T) {
	results := []LineResult{{Line: 1, Source: "28 CNY @a -> @b", Errors: []LineError{{Code: "X"}}}}
	DetectDuplicates(results, nil)
	if results[0].Duplicate {
		t.Fatal("nil evaluation must not mark duplicates")
	}
	evaluation := evalFromText(t, profileLedger)
	DetectDuplicates(results, evaluation)
	if results[0].Duplicate {
		t.Fatal("error rows are never duplicates")
	}
}

func TestHasEquivalentTransactionMismatchDimensions(t *testing.T) {
	// Build one candidate entry; each ledger variant differs in exactly one
	// dimension, so only the identical variant may be flagged.
	entry := compiledEntry(t)
	ledgerText := func(mutation string) string {
		return `2000-01-01 open Assets:WeChat CNY,USD
2000-01-01 open Expenses:Food CNY,USD
2000-01-01 open Assets:Cash CNY
2000-01-01 open Equity:Opening CNY
` + mutation
	}
	variants := []struct {
		name    string
		ledger  string
		dupWant bool
	}{
		{"identical", ledgerText("2026-08-12 *\n  Assets:WeChat -12 CNY\n  Expenses:Food 12 CNY\n"), true},
		{"date differs", ledgerText("2026-08-13 *\n  Assets:WeChat -12 CNY\n  Expenses:Food 12 CNY\n"), false},
		{"flag differs", ledgerText("2026-08-12 !\n  Assets:WeChat -12 CNY\n  Expenses:Food 12 CNY\n"), false},
		{"payee differs", ledgerText("2026-08-12 * \"别家\" \"\"\n  Assets:WeChat -12 CNY\n  Expenses:Food 12 CNY\n"), false},
		{"narration differs", ledgerText("2026-08-12 * \"\" \"另一件事\"\n  Assets:WeChat -12 CNY\n  Expenses:Food 12 CNY\n"), false},
		{"posting count differs", ledgerText("2026-08-12 *\n  Assets:WeChat -12 CNY\n  Expenses:Food 6 CNY\n  Assets:Cash 6 CNY\n"), false},
		{"account differs", ledgerText("2026-08-12 *\n  Assets:Cash -12 CNY\n  Expenses:Food 12 CNY\n"), false},
		{"amount differs", ledgerText("2026-08-12 *\n  Assets:WeChat -13 CNY\n  Expenses:Food 13 CNY\n"), false},
		{"currency differs", ledgerText("2026-08-12 *\n  Assets:WeChat -12 USD\n  Expenses:Food 12 USD\n"), false},
	}
	for _, tc := range variants {
		t.Run(tc.name, func(t *testing.T) {
			results := []LineResult{{Line: 1, Source: "x", Entry: entry}}
			DetectDuplicates(results, evalFromText(t, tc.ledger))
			if results[0].Duplicate != tc.dupWant {
				t.Fatalf("duplicate=%v want %v", results[0].Duplicate, tc.dupWant)
			}
		})
	}
}

func compiledEntry(t *testing.T) *favaadapter.NewEntry {
	t.Helper()
	results := Compile(CompileRequest{
		Text:       "12 CNY @Assets:WeChat -> @Expenses:Food",
		Date:       "2026-08-12",
		Flag:       "*",
		Evaluation: evalFromText(t, profileLedger),
	})
	if len(results) != 1 || results[0].Entry == nil {
		t.Fatalf("compile failed: %+v", results)
	}
	return results[0].Entry
}

func TestTokenizeShapes(t *testing.T) {
	tokens := tokenize("12\tCNY\t@微信\t->\t@餐饮 : 备注 #tag ^link")
	kinds := make([]tokenKind, 0, len(tokens))
	for _, tk := range tokens {
		kinds = append(kinds, tk.kind)
	}
	want := []tokenKind{tokWord, tokWord, tokAt, tokArrow, tokAt, tokColon, tokHash, tokLink}
	if len(kinds) != len(want) {
		t.Fatalf("kinds=%v want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds=%v want %v", kinds, want)
		}
	}
	// Narration runs until the next sigil, so "#tag" stays a tag.
	if tokens[5].text != ":备注" {
		t.Errorf("narration token=%q", tokens[5].text)
	}
	if tokens[6].text != "#tag" {
		t.Errorf("tag token=%q", tokens[6].text)
	}
	if sigilKind('#') != tokHash || sigilKind('^') != tokLink || sigilKind('@') != tokAt {
		t.Error("sigilKind mapping broken")
	}
	// A lone dash that is not an arrow is a word.
	if tk := tokenize("-")[0]; tk.kind != tokWord || tk.text != "-" {
		t.Errorf("lone dash token=%+v", tk)
	}
}

func TestAtoiOrZeroRejectsNonDigits(t *testing.T) {
	if atoiOrZero("12a") != 0 {
		t.Fatal("non-digit input must yield 0")
	}
	if atoiOrZero("12") != 12 {
		t.Fatal("digits must parse")
	}
}

func TestProfileHelpers(t *testing.T) {
	if got := splitSpaceList("a b,c\td"); strings.Join(got, ",") != "a,b,c,d" {
		t.Errorf("splitSpaceList=%v", got)
	}
	if got := splitSpaceList("   "); len(got) != 0 {
		t.Errorf("blank input=%v", got)
	}
	if accountString(ledger.Value{Kind: ledger.ValueAccount, String: "Assets:Cash"}) != "Assets:Cash" {
		t.Error("account-typed value should pass through")
	}
	if accountString(ledger.Value{Kind: ledger.ValueString, String: " Assets:Cash "}) != "Assets:Cash" {
		t.Error("string-typed account should validate and trim")
	}
	if accountString(ledger.Value{Kind: ledger.ValueString, String: "nope"}) != "" {
		t.Error("invalid account string should be rejected")
	}
	if accountString(ledger.Value{Kind: ledger.ValueNumber}) != "" {
		t.Error("non-account typed value should be rejected")
	}
	if isValidAccount("Assets:Cash") != true || isValidAccount("assets:cash") || isValidAccount("Assets:") || isValidAccount("Assets") {
		t.Error("isValidAccount contract broken")
	}
}

func TestEffectiveProfileNilAndDateWindow(t *testing.T) {
	if profile := EffectiveProfile(nil, ledger.Date{Year: 2026}); len(profile.Accounts) != 0 || len(profile.Templates) != 0 {
		t.Fatal("nil evaluation must yield an empty profile")
	}
	// A rule dated after the capture date must not apply.
	evaluation := evalFromText(t, `2026-09-01 custom "orangecount.quick-account.v1" "晚到" Assets:WeChat
2026-01-01 custom "orangecount.quick-account.v1" "早到" Assets:Cash
`)
	profile := EffectiveProfile(evaluation, ledger.Date{Year: 2026, Month: 8, Day: 12})
	if len(profile.Accounts) != 1 || profile.Accounts[0].Alias != "早到" {
		t.Fatalf("accounts=%+v", profile.Accounts)
	}
}

func TestParseAccountCustomProblems(t *testing.T) {
	custom := ledger.Custom{Type: accountCustomType}
	if _, problem, ok := parseAccountCustom(custom); ok || problem == nil || problem.Code != "W-QUICK-ACCOUNT-ARITY" {
		t.Fatalf("arity problem=%+v ok=%v", problem, ok)
	}
	custom.Values = []ledger.Value{{Kind: ledger.ValueNumber}, {Kind: ledger.ValueString, String: "Assets:Cash"}}
	if _, problem, ok := parseAccountCustom(custom); ok || problem == nil || problem.Code != "W-QUICK-ACCOUNT-ALIAS" {
		t.Fatalf("alias problem=%+v ok=%v", problem, ok)
	}
	custom.Values = []ledger.Value{{Kind: ledger.ValueString, String: " "}, {Kind: ledger.ValueString, String: "Assets:Cash"}}
	if _, problem, ok := parseAccountCustom(custom); ok || problem == nil || problem.Code != "W-QUICK-ACCOUNT-ALIAS" {
		t.Fatalf("blank alias problem=%+v ok=%v", problem, ok)
	}
	custom.Values = []ledger.Value{{Kind: ledger.ValueString, String: "别名"}, {Kind: ledger.ValueString, String: "not-an-account"}}
	if _, problem, ok := parseAccountCustom(custom); ok || problem == nil || problem.Code != "W-QUICK-ACCOUNT-ACCOUNT" {
		t.Fatalf("account problem=%+v ok=%v", problem, ok)
	}
}

func TestParseTemplateCustomNameAndMeta(t *testing.T) {
	custom := ledger.Custom{Type: templateCustomType}
	if _, problem, ok := parseTemplateCustom(custom); ok || problem == nil || problem.Code != "W-QUICK-TEMPLATE-NAME" {
		t.Fatalf("name problem=%+v ok=%v", problem, ok)
	}
	custom.Values = []ledger.Value{{Kind: ledger.ValueString, String: "模"}}
	rule, problem, ok := parseTemplateCustom(custom)
	if !ok || problem != nil {
		t.Fatalf("ok=%v problem=%+v", ok, problem)
	}
	if rule.Name != "模" {
		t.Fatalf("name=%s", rule.Name)
	}
	custom.Meta = []ledger.Metadata{
		{Key: "source", Value: ledger.Value{Kind: ledger.ValueString, String: " Assets:WeChat "}},
		{Key: "destination", Value: ledger.Value{Kind: ledger.ValueString, String: "Expenses:Food"}},
		{Key: "currency", Value: ledger.Value{Kind: ledger.ValueString, String: "CNY"}},
		{Key: "payee", Value: ledger.Value{Kind: ledger.ValueString, String: "商家"}},
		{Key: "narration", Value: ledger.Value{Kind: ledger.ValueString, String: "说明"}},
		{Key: "tags", Value: ledger.Value{Kind: ledger.ValueString, String: "a b"}},
		{Key: "links", Value: ledger.Value{Kind: ledger.ValueString, String: "x,y"}},
		{Key: "ignored", Value: ledger.Value{Kind: ledger.ValueString, String: "z"}},
	}
	rule, problem, ok = parseTemplateCustom(custom)
	if !ok || problem != nil {
		t.Fatalf("ok=%v problem=%+v", ok, problem)
	}
	if rule.Source != "Assets:WeChat" || rule.Destination != "Expenses:Food" || rule.Currency != "CNY" || rule.Payee != "商家" || rule.Narration != "说明" {
		t.Fatalf("rule=%+v", rule)
	}
	if strings.Join(rule.Tags, ",") != "a,b" || strings.Join(rule.Links, ",") != "x,y" {
		t.Fatalf("tags=%v links=%v", rule.Tags, rule.Links)
	}
	// Blank tag/link lists stay empty rather than producing one empty item.
	custom.Meta = []ledger.Metadata{{Key: "tags", Value: ledger.Value{Kind: ledger.ValueString, String: "  "}}}
	rule, _, _ = parseTemplateCustom(custom)
	if len(rule.Tags) != 0 {
		t.Fatalf("blank tags=%v", rule.Tags)
	}
}

func TestLatestAccountAndTemplateEmptyGroups(t *testing.T) {
	if rule, ambiguous := latestAccount(nil); ambiguous || rule.Alias != "" {
		t.Fatalf("empty account group rule=%+v ambiguous=%v", rule, ambiguous)
	}
	if rule, ambiguous := latestTemplate(nil); ambiguous || rule.Name != "" {
		t.Fatalf("empty template group rule=%+v ambiguous=%v", rule, ambiguous)
	}
}

func TestProfileUnsupportedAndAmbiguousTemplates(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 custom "orangecount.quick-template.v9" "旧版"
2000-01-01 custom "orangecount.quick-template.v1" "双胞胎"
2000-01-01 custom "orangecount.quick-template.v1" "双胞胎"
`)
	profile := EffectiveProfile(evaluation, ledger.Date{Year: 2026})
	if len(profile.Templates) != 0 {
		t.Fatalf("ambiguous templates must be disabled: %+v", profile.Templates)
	}
	codes := map[string]bool{}
	for _, problem := range profile.Problems {
		codes[problem.Code] = true
	}
	if !codes["W-QUICK-SCHEMA-UNSUPPORTED"] || !codes["W-QUICK-TEMPLATE-AMBIGUOUS"] {
		t.Fatalf("problems=%+v", profile.Problems)
	}
}

func TestCompileSerializeBaseline(t *testing.T) {
	// Sanity check for the duplicate-detection fixture: the canonical
	// explicit form compiles cleanly and previews as Beancount source.
	result := compileOne(t, CompileRequest{
		Text: "28 CNY @Assets:WeChat -> @Expenses:Food : 合法",
		Date: "2026-08-12",
	})
	if len(result.Errors) != 0 {
		t.Fatalf("baseline should compile: %+v", result.Errors)
	}
	if !strings.Contains(result.Preview, "2026-08-12 *") {
		t.Fatalf("preview=%q", result.Preview)
	}
}

func TestCompileLineEmptyTokenList(t *testing.T) {
	// A line of only sigils still tokenizes, but a genuinely empty token
	// list (whitespace-only line) is skipped by Compile; reach compileLine's
	// guard directly for the empty-line branch.
	result := compileLine("   ", 1, ledger.Date{Year: 2026}, "*", "CNY", Profile{})
	if len(result.Errors) != 1 || result.Errors[0].Code != "E-QUICK-EMPTY-LINE" {
		t.Fatalf("errors=%+v", result.Errors)
	}
}

func TestResolveAccountAliasChain(t *testing.T) {
	profile := Profile{Accounts: []AccountRule{{Alias: "别名", Account: "Assets:Cash"}}}
	if got := resolveAccount("别名", profile); got != "Assets:Cash" {
		t.Errorf("alias resolution=%s", got)
	}
	if got := resolveAccount("Assets:Cash", profile); got != "Assets:Cash" {
		t.Errorf("full account should pass through, got %s", got)
	}
	if got := resolveAccount("missing", profile); got != "" {
		t.Errorf("unknown alias should be empty, got %s", got)
	}
	if got := resolveAccount("", profile); got != "" {
		t.Errorf("empty alias should be empty, got %s", got)
	}
	if _, ok := findTemplate(profile, "nope"); ok {
		t.Error("unknown template must not be found")
	}
}

var _ = favaadapter.NewEntry{}
