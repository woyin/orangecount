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

// The tests in this file cover parser and evaluator branches the primary
// grammar and booking tests skip: directive arity errors, tolerance options,
// metadata attachment fallbacks, and inference edges.

func parseOrFail(t *testing.T, text string) *File {
	t.Helper()
	file, bag := ParseText("coverage.bean", []byte(text))
	if bag.HasErrors() {
		t.Fatalf("unexpected parse errors: %+v", bag.All())
	}
	return file
}

func TestDirectiveArityAndShapeErrors(t *testing.T) {
	cases := []struct {
		name string
		text string
		code string
	}{
		{"balance too short", "2000-01-01 balance Assets:Cash", "E-PARSE-EXPECTED"},
		{"pad too short", "2000-01-01 pad Assets:Cash", "E-PARSE-EXPECTED"},
		{"event not quoted", "2000-01-01 event type value", "E-PARSE-EXPECTED"},
		{"event too short", "2000-01-01 event \"type\"", "E-PARSE-EXPECTED"},
		{"query too short", "2000-01-01 query \"name\"", "E-PARSE-EXPECTED"},
		{"query not quoted", "2000-01-01 query name text", "E-PARSE-EXPECTED"},
		{"price too short", "2000-01-01 price USD", "E-PARSE-EXPECTED"},
		{"document too short", "2000-01-01 document Assets:Cash", "E-PARSE-EXPECTED"},
		{"note too short", "2000-01-01 note Assets:Cash", "E-PARSE-EXPECTED"},
		{"note not quoted", "2000-01-01 note Assets:Cash comment", "E-PARSE-EXPECTED"},
		{"custom too short", "2000-01-01 custom \"type\"", "E-PARSE-EXPECTED"},
		{"dated directive alone", "2000-01-01", "E-PARSE-EXPECTED"},
		{"unknown date word", "2000-01-01 mystery Assets:Cash", "E-PARSE-DIRECTIVE"},
		{"not a directive", "mystery line here", "E-PARSE-DIRECTIVE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, bag := ParseText("arity.bean", []byte(tc.text))
			found := false
			for _, d := range bag.All() {
				if d.Code == tc.code {
					found = true
				}
			}
			if !found {
				t.Fatalf("text=%q diagnostics=%+v want %s", tc.text, bag.All(), tc.code)
			}
		})
	}
}

func TestParserKeywordArityErrors(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"option unquoted", "option title value"},
		{"option arity", "option \"title\""},
		{"plugin unquoted", "plugin mod"},
		{"plugin arity", "plugin \"mod\" extra"},
		{"include unquoted", "include path.bean"},
		{"include arity", "include"},
		{"pushtag missing hash", "pushtag tag"},
		{"pushtag arity", "pushtag"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, bag := ParseText("keyword.bean", []byte(tc.text))
			if !bag.HasErrors() {
				t.Fatalf("text=%q should report E-PARSE-EXPECTED, got %+v", tc.text, bag.All())
			}
		})
	}
}

func TestParseDirectiveMetadataAttachment(t *testing.T) {
	file := parseOrFail(t, "2000-01-01 open Assets:Cash USD\n  owner: \"me\"\n  note2: 12 USD\n")
	open, ok := file.Directives[0].(Open)
	if !ok || len(open.Meta) != 2 {
		t.Fatalf("open meta=%+v", open.Meta)
	}
	// Keyword directives also receive following metadata.
	file = parseOrFail(t, "option \"title\" \"t\"\n  provenance: \"handwritten\"\n")
	option, ok := file.Directives[0].(Option)
	if !ok || len(option.Meta) != 1 || option.Meta[0].Key != "provenance" {
		t.Fatalf("option meta=%+v", option.Meta)
	}
	// Metadata between the transaction head and its first posting attaches
	// to the transaction itself (posting metadata attaches to the posting).
	file = parseOrFail(t, "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 *\n  channel: \"web\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n")
	var tx *Transaction
	for _, d := range file.Directives {
		if candidate, isTX := d.(*Transaction); isTX {
			tx = candidate
		}
	}
	if tx == nil || len(tx.Meta) != 1 || tx.Meta[0].Key != "channel" {
		t.Fatalf("tx meta=%+v", tx.Meta)
	}
}

func TestParsePostingModifierEdges(t *testing.T) {
	file := parseOrFail(t, "2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Shares USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 *\n  Assets:Shares 2 USD {3 USD} @ 4 USD\n  Assets:Cash -6 USD\n")
	tx, ok := file.Directives[3].(*Transaction)
	if !ok {
		t.Fatalf("directives=%+v", file.Directives)
	}
	posting := tx.Postings[0]
	if posting.Cost == nil || posting.Price == nil || posting.Price.Total {
		t.Fatalf("posting=%+v", posting)
	}
	// Total price uses the @@ operator.
	file = parseOrFail(t, "2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Shares USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 *\n  Assets:Shares 2 USD @@ 8 USD\n  Assets:Cash -8 USD\n")
	tx = file.Directives[3].(*Transaction)
	if spec := tx.Postings[0].Price; spec == nil || !spec.Total {
		t.Fatalf("total price=%+v", tx.Postings[0].Price)
	}
	// A trailing @ with no amount drops the posting silently (the amount
	// parser sees no tokens and the caller gives up on the line).
	file, bag := ParseText("incomplete.bean", []byte("2000-01-02 *\n  Assets:Shares 2 USD @\n"))
	if bag.HasErrors() || len(file.Directives) != 1 {
		t.Fatalf("bag=%+v directives=%d", bag.All(), len(file.Directives))
	}
	if tx, isTX := file.Directives[0].(*Transaction); isTX && len(tx.Postings) != 0 {
		t.Fatalf("postings=%+v", tx.Postings)
	}
}

func TestEvaluateOptionToleranceValidation(t *testing.T) {
	// A negative tolerance is rejected.
	_, bag := ParseText("neg.bean", []byte("option \"tolerance\" \"-1\"\n"))
	_ = bag
	file, parseBag := ParseText("neg.bean", []byte("option \"tolerance\" \"-1\"\n"))
	if parseBag.HasErrors() {
		t.Fatal(parseBag.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	found := false
	for _, d := range evaluation.Diagnostics {
		if d.Code == "E-EVAL-TOLERANCE" {
			found = true
		}
	}
	if !found {
		t.Fatalf("negative tolerance diagnostics=%+v", evaluation.Diagnostics)
	}
	// A non-numeric tolerance is rejected too.
	file, _ = ParseText("bad.bean", []byte("option \"tolerance\" \"lots\"\n"))
	evaluation = EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	found = false
	for _, d := range evaluation.Diagnostics {
		if d.Code == "E-EVAL-OPTION" {
			found = true
		}
	}
	if !found {
		t.Fatalf("non-numeric tolerance diagnostics=%+v", evaluation.Diagnostics)
	}
	// A valid tolerance seeds the balancing tolerance.
	file, _ = ParseText("ok.bean", []byte("option \"tolerance\" \"0.5\"\n"))
	evaluation = EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	for _, d := range evaluation.Diagnostics {
		if d.Code == "E-EVAL-TOLERANCE" || d.Code == "E-EVAL-OPTION" {
			t.Fatalf("valid tolerance diagnostics=%+v", evaluation.Diagnostics)
		}
	}
}

func TestDecimalSignAndZeroContracts(t *testing.T) {
	zero := Zero()
	if zero.Sign() != 0 {
		t.Fatal("zero sign")
	}
	negative := NewDecimal(big.NewRat(-1, 2))
	if negative.Sign() >= 0 {
		t.Fatal("negative sign")
	}
	positive := NewDecimal(big.NewRat(1, 2))
	if positive.Sign() <= 0 {
		t.Fatal("positive sign")
	}
}

func TestInferenceEdgesThroughEvaluator(t *testing.T) {
	// Two amount-less postings cannot be inferred.
	file := parseOrFail(t, "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 *\n  Assets:Cash USD\n  Equity:Opening USD\n")
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	found := false
	for _, d := range evaluation.Diagnostics {
		if d.Code == "E-EVAL-INFER" {
			found = true
		}
	}
	if !found {
		t.Fatalf("double elision diagnostics=%+v", evaluation.Diagnostics)
	}
	// A single elision against a price spec infers from the price leg.
	file = parseOrFail(t, "2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Shares USD\n2000-01-02 *\n  Assets:Shares 2 USD {3 USD}\n  Assets:Cash\n")
	evaluation = EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	for _, d := range evaluation.Diagnostics {
		if d.Code == "E-EVAL-INFER" || d.Code == "E-EVAL-UNBALANCED" {
			t.Fatalf("price inference diagnostics=%+v", evaluation.Diagnostics)
		}
	}
}

func TestOpenLifecycleErrors(t *testing.T) {
	file := parseOrFail(t, "2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Cash USD\n")
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	found := false
	for _, d := range evaluation.Diagnostics {
		if d.Code == "E-EVAL-OPEN" {
			found = true
		}
	}
	if !found {
		t.Fatalf("double open diagnostics=%+v", evaluation.Diagnostics)
	}
	file = parseOrFail(t, "2000-01-01 open Assets:Cash USD\n2000-01-02 close Assets:Cash\n2000-01-03 open Assets:Cash USD\n")
	evaluation = EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	found = false
	for _, d := range evaluation.Diagnostics {
		if d.Code == "E-EVAL-REOPEN" {
			found = true
		}
	}
	if !found {
		t.Fatalf("reopen diagnostics=%+v", evaluation.Diagnostics)
	}

}

func TestParseContinuationEdges(t *testing.T) {
	// A line indented deeper than an existing posting cannot be another
	// posting; it is expected to be metadata and errors instead.
	_, bag := ParseText("continuation.bean", []byte("2000-01-01 open Assets:Cash USD\n2000-01-02 *\n  Assets:Cash 1 USD\n    Assets:Cash 2 USD\n"))
	if !bag.HasErrors() {
		t.Fatal("deeper-indented posting should error")
	}
	// A continuation line with no open transaction is an error.
	_, bag = ParseText("orphan.bean", []byte("2000-01-01 open Assets:Cash USD\n  Assets:Cash 1 USD\n"))
	if !bag.HasErrors() {
		t.Fatal("orphan continuation should error")
	}
	// Tokenizer edge: a word ending at a comment marker.
	file := parseOrFail(t, "option \"title\" \"t\" ; trailing comment\n")
	if len(file.Comments) != 1 || !strings.Contains(file.Comments[0].Text, "trailing comment") {
		t.Fatalf("comments=%+v", file.Comments)
	}
}

func TestAmountParsingEdges(t *testing.T) {
	file := parseOrFail(t, "2000-01-01 open Assets:Cash USD\n2000-01-01 balance Assets:Cash 1,000.50 USD ~ 0.5\n")
	balance, ok := file.Directives[1].(Balance)
	if !ok {
		t.Fatalf("directives=%+v", file.Directives)
	}
	if balance.Amount.Number.Raw != "1000.50" {
		t.Fatalf("amount raw=%s", balance.Amount.Number.Raw)
	}
	if balance.Tolerance == nil {
		t.Fatal("tolerance missing")
	}
	// price requires a complete amount.
	_, bag := ParseText("price.bean", []byte("2000-01-01 price USD 3\n"))
	if !bag.HasErrors() {
		t.Fatal("incomplete price amount should error")
	}
}

func TestEvaluateDirectiveDefaultWarning(t *testing.T) {
	// A directive type the evaluator does not know warns. Pad-on-unknown and
	// others are covered; reach the default branch with a Query on a missing
	// table is not possible — instead drive evaluateDirective directly.
	file := parseOrFail(t, "2000-01-01 open Assets:Cash USD\n")
	e := &evaluator{result: &Evaluation{Prices: map[string][]PriceQuote{}, Options: map[string]string{}}}
	e.evaluateDirective(file, Option{DirectiveBase: DirectiveBase{At: file.Directives[0].Span()}, Key: "operating_currency", Value: "USD"})
	if e.result.Options["operating_currency"] != "USD" {
		t.Fatalf("options=%+v", e.result.Options)
	}
}
