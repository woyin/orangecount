// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dialect

import (
	"math/big"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

func TestParseQuickAccountCustomAcceptsAccountAndStringValues(t *testing.T) {
	base := ledger.Custom{Type: "orangecount.quick-account.v1"}
	cases := []struct {
		name    string
		values  []ledger.Value
		alias   string
		account string
		ok      bool
	}{
		{"account value", []ledger.Value{
			{Kind: ledger.ValueString, String: "微信"},
			{Kind: ledger.ValueAccount, String: "Assets:WeChat"},
		}, "微信", "Assets:WeChat", true},
		{"quoted account", []ledger.Value{
			{Kind: ledger.ValueString, String: "微信"},
			{Kind: ledger.ValueString, String: "Assets:WeChat"},
		}, "微信", "Assets:WeChat", true},
		{"non-string alias", []ledger.Value{
			{Kind: ledger.ValueAccount, String: "WeChat"},
			{Kind: ledger.ValueAccount, String: "Assets:WeChat"},
		}, "", "", false},
		{"currency account", []ledger.Value{
			{Kind: ledger.ValueString, String: "微信"},
			{Kind: ledger.ValueCurrency, String: "USD"},
		}, "", "", false},
		{"too few values", []ledger.Value{{Kind: ledger.ValueString, String: "微信"}}, "", "", false},
		{"empty alias", []ledger.Value{
			{Kind: ledger.ValueString, String: ""},
			{Kind: ledger.ValueAccount, String: "Assets:WeChat"},
		}, "", "", false},
	}
	for _, tc := range cases {
		custom := base
		custom.Values = tc.values
		alias, account, ok := parseQuickAccountCustom(custom)
		if ok != tc.ok || alias != tc.alias || account != tc.account {
			t.Errorf("%s: got (%q,%q,%v) want (%q,%q,%v)", tc.name, alias, account, ok, tc.alias, tc.account, tc.ok)
		}
	}
}

func TestAccountNameValidAndTail(t *testing.T) {
	valid := []string{"Assets:Cash", "Expenses:Food:Lunch", "Assets:Wallet:微信", "Assets:", "Assets::Cash"}
	invalid := []string{"Assets", "assets:cash", ""}
	for _, name := range valid {
		if !accountNameValid(name) {
			t.Errorf("%q should be valid", name)
		}
	}
	for _, name := range invalid {
		if accountNameValid(name) {
			t.Errorf("%q should be invalid", name)
		}
	}
	if got := accountTail("Expenses:Food:Lunch"); got != "Lunch" {
		t.Errorf("tail=%q", got)
	}
	if got := accountTail("Assets"); got != "Assets" {
		t.Errorf("colonless tail=%q", got)
	}
}

func TestResolveAliasPrefersLatestEffectiveDate(t *testing.T) {
	d := func(year, month, day int) ledger.Date {
		return ledger.Date{Year: year, Month: month, Day: day}
	}
	idx := &index{aliases: []aliasRule{
		{date: d(2026, 1, 1), alias: "微信", account: "Assets:WeChat"},
		{date: d(2026, 6, 1), alias: "微信", account: "Assets:WeChatNew"},
	}}
	if account, ok := resolveAlias(idx, "微信", d(2026, 3, 1)); !ok || account != "Assets:WeChat" {
		t.Fatalf("march resolution=%q,%v", account, ok)
	}
	if account, ok := resolveAlias(idx, "微信", d(2026, 9, 1)); !ok || account != "Assets:WeChatNew" {
		t.Fatalf("september resolution=%q,%v", account, ok)
	}
	if _, ok := resolveAlias(idx, "微信", d(2025, 1, 1)); ok {
		t.Fatal("alias resolved before its directive date")
	}
	// Same-day competing definitions disable the alias (ADR-0043).
	idx.aliases = append(idx.aliases, aliasRule{date: d(2026, 6, 1), alias: "微信", account: "Assets:Other"})
	if _, ok := resolveAlias(idx, "微信", d(2026, 9, 1)); ok {
		t.Fatal("same-day competing alias resolved")
	}
}

func TestApplyEditsSkipsOverlappingAndOutOfRange(t *testing.T) {
	data := []byte("0123456789")
	edits := []Edit{
		{Span: source.Span{Start: 2, End: 4}, Text: "XY"},
		{Span: source.Span{Start: 3, End: 5}, Text: "Z"},   // overlapping: skipped
		{Span: source.Span{Start: 20, End: 25}, Text: "!"}, // out of range: skipped
	}
	if got := string(ApplyEdits(data, edits)); got != "01XY456789" {
		t.Fatalf("apply=%q", got)
	}
}

func TestEligibleForDialectRejectsEachIneligibleShape(t *testing.T) {
	date := ledger.Date{Year: 2026, Month: 8, Day: 12}
	amount := func(raw string, neg bool) *ledger.Amount {
		rat, ok := new(big.Rat).SetString(raw)
		if !ok {
			t.Fatalf("bad test amount %q", raw)
		}
		if neg {
			rat = new(big.Rat).Neg(rat)
		}
		return &ledger.Amount{Number: ledger.Number{Raw: raw, Rat: rat}, Currency: "USD"}
	}
	base := func() ledger.Transaction {
		return ledger.Transaction{Date: date, Flag: "*", Narration: "午餐", Postings: []ledger.Posting{
			{Account: "A", Units: amount("5", true)},
			{Account: "B", Units: amount("5", false)},
		}}
	}
	if !eligibleForDialect(ptr(base())) {
		t.Fatal("baseline should be eligible")
	}
	shapes := map[string]func(*ledger.Transaction){
		"three legs":     func(tx *ledger.Transaction) { tx.Postings = append(tx.Postings, ledger.Posting{Account: "C"}) },
		"posting flag":   func(tx *ledger.Transaction) { tx.Postings[0].Flag = "!" },
		"posting meta":   func(tx *ledger.Transaction) { tx.Postings[0].Meta = []ledger.Metadata{{Key: "k"}} },
		"cost":           func(tx *ledger.Transaction) { tx.Postings[0].Cost = &ledger.CostSpec{Raw: "{5 USD}"} },
		"price":          func(tx *ledger.Transaction) { tx.Postings[0].Price = &ledger.PriceSpec{Amount: *amount("5", false)} },
		"mixed currency": func(tx *ledger.Transaction) { tx.Postings[0].Units.Currency = "CNY" },
		"unbalanced":     func(tx *ledger.Transaction) { tx.Postings[1].Units = amount("6", false) },
		"zero amount": func(tx *ledger.Transaction) {
			tx.Postings[0].Units = amount("0", true)
			tx.Postings[1].Units = amount("0", false)
		},
		"no narration": func(tx *ledger.Transaction) { tx.Narration = ""; tx.Payee = "" },
		"unsafe narr":  func(tx *ledger.Transaction) { tx.Narration = "a #b" },
		"padded narr":  func(tx *ledger.Transaction) { tx.Narration = " x" },
		"unsafe payee": func(tx *ledger.Transaction) { tx.Payee = `a"b` },
		"odd flag":     func(tx *ledger.Transaction) { tx.Flag = "X" },
		"invalid date": func(tx *ledger.Transaction) { tx.Date = ledger.Date{Year: 2026, Month: 13, Day: 1} },
	}
	for name, mutate := range shapes {
		tx := ptr(base())
		mutate(tx)
		if eligibleForDialect(tx) {
			t.Errorf("%s: unexpectedly eligible", name)
		}
	}
}

func ptr(tx ledger.Transaction) *ledger.Transaction { return &tx }

func TestSpanHasCommentAndTransactionOf(t *testing.T) {
	file := ledger.File{Source: source.NewSourceFile(1, "x.bean", []byte("abc ; TODO: fix nav\n"))}
	todos, hasOther := spanTodos(file.Source, file.Source.Span(0, 20))
	if hasOther || len(todos) != 1 || todos[0] != "fix nav" {
		t.Fatalf("todo not extracted: %v %v", todos, hasOther)
	}
	free := source.NewSourceFile(1, "x.bean", []byte("abc ; note"))
	if todos, hasOther := spanTodos(free, free.Span(0, 9)); len(todos) != 0 || !hasOther {
		t.Fatalf("free comment misread: %v %v", todos, hasOther)
	}
	quoted := source.NewSourceFile(1, "x.bean", []byte(`"a;TODO: x"`))
	if todos, hasOther := spanTodos(quoted, quoted.Span(0, 11)); len(todos) != 0 || hasOther {
		t.Fatal("semicolon inside string misread as comment")
	}
	if todos, hasOther := spanTodos(nil, source.Span{}); todos != nil || hasOther {
		t.Fatal("nil file should report nothing")
	}
	asValue := ledger.Transaction{}
	if _, ok := transactionOf(asValue); !ok {
		t.Fatal("value transaction not recognized")
	}
	if _, ok := transactionOf(ledger.Open{}); ok {
		t.Fatal("open misread as transaction")
	}
}

func TestSerializeHelpers(t *testing.T) {
	if got := flagOrDefault(""); got != "*" {
		t.Fatalf("default flag=%q", got)
	}
	if got := flagOrDefault("!"); got != "!" {
		t.Fatalf("flag=%q", got)
	}
	if got := canonicalDate(ledger.Date{}); got != "" {
		t.Fatalf("invalid date serialized=%q", got)
	}
	rat := func(text string) *big.Rat {
		value, ok := new(big.Rat).SetString(text)
		if !ok {
			t.Fatalf("bad test number %q", text)
		}
		return value
	}
	txn := &ledger.Transaction{Postings: []ledger.Posting{
		{Account: "A", Units: &ledger.Amount{Number: ledger.Number{Raw: "-5", Rat: rat("-5")}, Currency: "USD"}},
		{Account: "B", Units: &ledger.Amount{Number: ledger.Number{Raw: "5", Rat: rat("5")}, Currency: "USD"}},
	}}
	if positivePosting(txn).Account != "B" {
		t.Fatal("positive leg not found")
	}
	if otherPosting(txn, &txn.Postings[1]).Account != "A" {
		t.Fatal("other leg not found")
	}
	noLegs := &ledger.Transaction{}
	if positivePosting(noLegs) != nil {
		t.Fatal("empty transaction yielded a positive leg")
	}
	if SerializeDialect(noLegs) != "" {
		t.Fatal("unserializable transaction produced text")
	}
}

// TestChineseAccountFullNameEndpoints guards the ADR resolution level 1:
// a full account name with CJK segments (valid v3, upstream beancount
// accepts it) must resolve verbatim instead of falling through to the
// tail/alias levels.
func TestChineseAccountFullNameEndpoints(t *testing.T) {
	file, bag := ledger.ParseText("m.bean", []byte(`2026-08-01 13.00 CNY @Assets:Wallet:微信 -> @Expenses:Living:吃的:一日三餐 "我" : 吃早饭`+"\n"))
	if bag.HasErrors() {
		t.Fatalf("parse errors: %v", bag.All())
	}
	if len(file.Directives) != 1 {
		t.Fatalf("directives=%d", len(file.Directives))
	}
	// Endpoint text is carried through verbatim; resolution happens in
	// the dialect package against the real account index.
	d := file.Directives[0].(ledger.Dialect)
	if d.SourceRef != "Assets:Wallet:微信" || d.DestRef != "Expenses:Living:吃的:一日三餐" {
		t.Fatalf("refs=%q %q", d.SourceRef, d.DestRef)
	}
}
