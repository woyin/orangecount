// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"strings"
	"testing"

	"orangecount/internal/source"
)

// evaluateText parses and evaluates one file, returning the evaluation and a
// joined string of all diagnostic codes for compact branch assertions.
func evaluateText(t *testing.T, text string) (*Evaluation, string) {
	t.Helper()
	file, parseDiagnostics := ParseText("branch.bean", []byte(text))
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	var codes []string
	for _, diagnostic := range append(parseDiagnostics.All(), evaluation.Diagnostics...) {
		codes = append(codes, diagnostic.Code)
	}
	return evaluation, strings.Join(codes, ",")
}

func TestEvaluateUnexpandedDialectWarns(t *testing.T) {
	// A Dialect directive reaching evaluation means a caller bypassed the
	// snapshot's expansion pass; the evaluator must warn, not drop it.
	file, parseDiagnostics := ParseText("dialect.bean", []byte(`2000-01-01 10 USD @Assets:Cash -> @Expenses:Food "午餐"`))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !hasCode(evaluation.Diagnostics, "W-DIALECT-UNEXPANDED") {
		t.Fatalf("diagnostics=%+v", evaluation.Diagnostics)
	}
}

func TestEvaluateValueFormTransactionAndEmptyPostings(t *testing.T) {
	// Entries may hold the transaction value form; the evaluator must book it
	// exactly like the pointer form the parser emits.
	file, _ := ParseText("value.bean", []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"entry\"\n  Assets:Cash 5 USD\n  Equity:Opening -5 USD\n"))
	valueFile := &File{}
	*valueFile = *file
	for index, directive := range valueFile.Directives {
		if transaction, ok := directive.(*Transaction); ok {
			valueFile.Directives[index] = *transaction
		}
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: valueFile}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid || hasCode(evaluation.Diagnostics, "E-") {
		t.Fatalf("value-form evaluation=%+v", evaluation.Diagnostics)
	}
	if account, _ := evaluation.Account("Assets:Cash"); account.Balances["USD"].String() != "5" {
		t.Fatalf("balance=%+v", account)
	}
	// A transaction without postings cannot balance and must error.
	_, codes := evaluateText(t, "2000-01-01 open Assets:Cash USD\n2000-01-02 * \"empty\"\n")
	if !strings.Contains(codes, "E-EVAL-UNBALANCED") {
		t.Fatalf("empty postings codes=%s", codes)
	}
}

func TestEvaluatePadWithoutMissingBalanceWarns(t *testing.T) {
	// A pad is only consumed by a balance assertion for its account. With no
	// assertion at all it can never fire, and the warning keeps that visible.
	evaluation, codes := evaluateText(t, `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-01 pad Assets:Cash Equity:Opening
2000-01-02 * "entry"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
`)
	if !evaluation.Valid {
		t.Fatalf("evaluation must stay valid: %+v", evaluation.Diagnostics)
	}
	if !strings.Contains(codes, "W-EVAL-PAD-UNUSED") {
		t.Fatalf("codes=%s", codes)
	}
}

func TestEvaluateBookingNoneAccountPassesReductionThrough(t *testing.T) {
	// A NONE-booking account does no lot matching: reductions post as given
	// even when they reference a cost that was never opened.
	_, codes := evaluateText(t, `2000-01-01 open Assets:Inv STOCK "NONE"
2000-01-01 open Equity:Opening USD
2000-01-02 * "buy"
  Assets:Inv 5 STOCK {10.00 USD}
  Equity:Opening -50.00 USD
2000-01-03 * "reduce unknown lot"
  Assets:Inv -5 STOCK {99.00 USD}
  Equity:Opening 495.00 USD
`)
	if !strings.Contains(codes, "E-") {
		t.Fatalf("expected the unmatched-cost reduction to fail loudly: %s", codes)
	}
}

func TestEvaluateElisionWithoutTargetErrors(t *testing.T) {
	// Two elided postings cannot both be inferred unambiguously.
	_, codes := evaluateText(t, `2000-01-01 open Assets:Cash USD
2000-01-01 open Expenses:Food USD
2000-01-02 * "ambiguous"
  Assets:Cash 10 USD
  Expenses:Food
  Expenses:Food
`)
	if !strings.Contains(codes, "E-EVAL") {
		t.Fatalf("codes=%s", codes)
	}
}

func TestParseMetadataStackKeywordErrors(t *testing.T) {
	_, codes := evaluateText(t, "popmeta owner\n")
	if !strings.Contains(codes, "E-PARSE-EXPECTED") {
		t.Fatalf("popmeta without colon codes=%s", codes)
	}
	_, codes = evaluateText(t, "pushmeta owner\n")
	if !strings.Contains(codes, "E-PARSE-EXPECTED") {
		t.Fatalf("pushmeta without colon codes=%s", codes)
	}
	_, codes = evaluateText(t, "pushmeta owner: 20 USD\n2000-01-01 open Assets:Cash USD\n")
	if strings.Contains(codes, "E-PARSE-EXPECTED") {
		t.Fatalf("valid pushmeta rejected: %s", codes)
	}
}

func TestParseBalanceToleranceExpressionErrors(t *testing.T) {
	_, codes := evaluateText(t, "2000-01-01 open Assets:Cash USD\n2000-01-02 balance Assets:Cash 1 USD ~ (2\n")
	if !strings.Contains(codes, "E-PARSE-TOKEN") {
		t.Fatalf("unterminated expression codes=%s", codes)
	}
}

func TestParseNumberExpressionEdges(t *testing.T) {
	// Unary signs, division, and parens evaluate exactly.
	file, bag := ParseText("expr.bean", []byte(`2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "expr"
  Assets:Cash (2 + 1) * 2 / 3 USD
  Equity:Opening -2 USD
`))
	if bag.HasErrors() {
		t.Fatalf("bag=%+v", bag.All())
	}
	directive := file.Directives[2].(*Transaction)
	if got := directive.Postings[0].Units.Number.Rat.RatString(); got != "2" {
		t.Fatalf("expression=%s", got)
	}
	// A word that only looks like an expression start stays a word: an
	// operator without a following operand must not swallow the line.
	_, codes := evaluateText(t, "2000-01-01 open Assets:Cash USD\n2000-01-02 * \"expr\"\n  Assets:Cash 3 + USD\n  Equity:Opening -3 USD\n")
	if !strings.Contains(codes, "E-PARSE-TOKEN") {
		t.Fatalf("trailing operator codes=%s", codes)
	}
}

func TestParseMultibyteWordTokens(t *testing.T) {
	// Bare non-ASCII words (aliases, Chinese endpoints) tokenize as words.
	file, bag := ParseText("words.bean", []byte("2000-01-01 commodity 米酒\n"))
	if bag.HasErrors() {
		t.Fatalf("bag=%+v", bag.All())
	}
	commodity, ok := file.Directives[0].(Commodity)
	if !ok || commodity.Currency != "米酒" {
		t.Fatalf("commodity=%+v", file.Directives[0])
	}
}

func TestParseOpenWithoutAccountErrors(t *testing.T) {
	_, codes := evaluateText(t, "2000-01-01 open\n")
	if !strings.Contains(codes, "E-PARSE-EXPECTED") {
		t.Fatalf("codes=%s", codes)
	}
}

func TestReplaceLastEntryIgnoresNilAndEmptyStream(t *testing.T) {
	// Neither a nil directive nor an empty entry stream may panic.
	state := &evaluator{result: &Evaluation{}}
	state.replaceLastEntry(nil)
	state.replaceLastEntry(Open{})
	if len(state.result.Entries) != 0 {
		t.Fatalf("entries=%+v", state.result.Entries)
	}
}
