// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"math/big"
	"strings"
	"testing"

	"orangecount/internal/source"
)

// codesFor parses and evaluates text and joins every diagnostic code.
func codesFor(t *testing.T, text string) string {
	t.Helper()
	file, parseBag := ParseText("edge.bean", []byte(text))
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	var codes []string
	for _, diagnostic := range append(parseBag.All(), evaluation.Diagnostics...) {
		codes = append(codes, diagnostic.Code)
	}
	return strings.Join(codes, ",")
}

// TestParseAndEvalGrammarEdges pins bounded errors for malformed constructs.
func TestParseAndEvalGrammarEdges(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{"unterminated string", "2000-01-01 note Assets:Cash \"oops\n", "E-PARSE-STRING"},
		{"metadata missing colon", "2000-01-01 open Assets:A USD\n  meta value\n", "E-PARSE-EXPECTED"},
		{"bare date line", "2000-01-01 open Assets:A USD\n2000-01-02\n", "E-PARSE-EXPECTED"},
		{"dialect leg order", "2000-01-01 open Assets:A USD\n2000-01-01 open Expenses:F USD\n2026-08-20 * \"我\" \"吃饭\"\n  Assets:A -100 USD\n  50 USD @F -> @A\n", "E-DIALECT-LEG-ORDER"},
		{"dialect bad security", "2000-01-01 open Assets:A USD\n2000-01-01 open Assets:B AAPL\n2026-01-02 10 STK {} @A -> @B 手续费 -1.00 CNY @Expenses:Fees\n", "E-DIALECT-SYNTAX"},
		{"dialect missing amount", "2000-01-01 open Assets:A USD\n2000-01-01 open Expenses:F USD\n2026-01-02 1.2.3 USD @A -> @F\n", "E-DIALECT-AMOUNT"},
		{"elided posting stays unbalanced when nothing can infer it", "2000-01-01 open Assets:A USD\n2000-01-01 open Expenses:F USD\n2026-01-02 * \"x\"\n  Assets:A -7 USD\n  Expenses:F @ 7 CNY\n", "E-EVAL-UNBALANCED"},
		{"dialect without date anchor", "5 USD @A -> @F\n", "E-DIALECT-DATE"},
		{"negative balance tolerance", "2000-01-01 open Assets:A USD\n2000-01-02 balance Assets:A 1 USD ~ -1\n", "E-EVAL-TOLERANCE"},
		{"balance short fails", "2000-01-01 open Assets:A USD\n2000-01-01 open Equity:E USD\n2000-01-02 * \"t\"\n  Assets:A 5 USD\n  Equity:E -5 USD\n2000-01-03 balance Assets:A 9 USD\n", "E-EVAL-BALANCE"},
		{"posting currency rejected", "2000-01-01 open Assets:A USD\n2000-01-01 open Equity:E USD\n2000-01-02 * \"t\"\n  Assets:A 5 EUR\n  Equity:E -5 USD\n", "E-EVAL-CURRENCY"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			codes := codesFor(t, testCase.text)
			if !strings.Contains(codes, testCase.want) {
				t.Fatalf("codes=%q want %q", codes, testCase.want)
			}
		})
	}
}

// TestEvaluatePluginCommodityAndUnknownFileID walks the evaluator's
// no-op directive branches and the tolerate-missing-file guard.
func TestEvaluatePluginCommodityAndUnknownFileID(t *testing.T) {
	text := `plugin "beancount.plugins.odd"
2000-01-01 commodity AAPL
2000-01-01 open Assets:A USD
`
	file, parseBag := ParseText("noop.bean", []byte(text))
	if parseBag.HasErrors() {
		t.Fatalf("parse=%+v", parseBag.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("diagnostics=%+v", evaluation.Diagnostics)
	}
	// An order entry whose file never parsed is skipped, not a panic.
	empty := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{2}, EvalOptions{})
	if empty == nil {
		t.Fatal("missing file id must yield an evaluation")
	}
}

// amountValue builds an Amount for direct struct tests.
func amountValue(raw, currency string) Amount {
	value, ok := new(big.Rat).SetString(raw)
	if !ok {
		panic("bad rat " + raw)
	}
	return Amount{Number: Number{Raw: raw, Rat: value}, Currency: currency}
}

// TestPostingWeightCoversCostPriceAndBareUnits pins the weight hierarchy:
// cost beats price beats bare units.
func TestPostingWeightCoversCostPriceAndBareUnits(t *testing.T) {
	costRat, okCost := new(big.Rat).SetString("100")
	if !okCost {
		t.Fatal("cost rat")
	}
	units := amountValue("2", "AAPL")
	withCost := Posting{Units: &units, Cost: &CostSpec{Components: []Value{{Kind: ValueAmount, Amount: Amount{Number: Number{Raw: "100", Rat: costRat}, Currency: "USD"}}}}}
	if got := PostingWeight(withCost).String(); got != "200" {
		t.Fatalf("cost weight=%s", got)
	}
	priceRat, okPrice := new(big.Rat).SetString("10")
	if !okPrice {
		t.Fatal("price rat")
	}
	withPrice := Posting{Units: &units, Price: &PriceSpec{Amount: Amount{Number: Number{Raw: "10", Rat: priceRat}, Currency: "USD"}}}
	if got := PostingWeight(withPrice).String(); got != "20" {
		t.Fatalf("price weight=%s", got)
	}
	bare := amountValue("3", "USD")
	if got := PostingWeight(Posting{Units: &bare}).String(); got != "3" {
		t.Fatalf("bare weight=%s", got)
	}
}
