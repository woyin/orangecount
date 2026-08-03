// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"os"
	"path/filepath"
	"testing"

	"orangecount/internal/diagnostic"
	"orangecount/internal/source"
)

func TestEvaluateBalancedLifecycleAndAssertion(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "opening"
  Assets:Cash 100 USD
  Equity:Opening -100 USD
2000-01-03 balance Assets:Cash 100 USD
2000-01-04 close Assets:Cash
`
	file, parseDiagnostics := ParseText("valid.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid || hasCode(evaluation.Diagnostics, "E-") {
		t.Fatalf("evaluation=%s diagnostics=%+v", evaluation, evaluation.Diagnostics)
	}
	account, ok := evaluation.Account("Assets:Cash")
	if !ok || account.Balances["USD"].String() != "100" || account.Closed == nil {
		t.Fatalf("account=%+v", account)
	}
	equity, ok := evaluation.Account("Equity:Opening")
	if !ok || equity.Balances["USD"].String() != "-100" {
		t.Fatalf("equity balance=%+v", equity)
	}
}

func TestEvaluateExcludesOptionsFromEntries(t *testing.T) {
	text := `option "title" "entry stream"
2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "opening"
  Assets:Cash 1 USD
  Equity:Opening -1 USD
`
	file, parseDiagnostics := ParseText("option-entry-stream.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid || evaluation.Options["title"] != "entry stream" {
		t.Fatalf("evaluation=%s options=%+v diagnostics=%+v", evaluation, evaluation.Options, evaluation.Diagnostics)
	}
	if len(evaluation.Entries) != 3 {
		t.Fatalf("entries=%d want 3", len(evaluation.Entries))
	}
	for _, entry := range evaluation.Entries {
		if _, ok := entry.Directive.(Option); ok {
			t.Fatalf("option leaked into entries: %+v", entry)
		}
	}
}

func TestEvaluateBalanceAssertionSortsBeforeSameDayTransactions(t *testing.T) {
	before := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "before assertion"
  Assets:Cash 10 USD
  Equity:Opening -10 USD
2000-01-02 balance Assets:Cash 0 USD
2000-01-02 * "after assertion"
  Assets:Cash 5 USD
  Equity:Opening -5 USD
`
	file, diagnostics := ParseText("same-day-before.bean", []byte(before))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", diagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid || hasCode(evaluation.Diagnostics, "E-EVAL-BALANCE") {
		t.Fatalf("same-day transaction diagnostics=%+v", evaluation.Diagnostics)
	}

	after := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 balance Assets:Cash 0 USD
2000-01-02 * "after assertion"
  Assets:Cash 5 USD
  Equity:Opening -5 USD
`
	file, diagnostics = ParseText("same-day-after.bean", []byte(after))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", diagnostics.All())
	}
	evaluation = EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid || hasCode(evaluation.Diagnostics, "E-EVAL-BALANCE") {
		t.Fatalf("after assertion diagnostics=%+v", evaluation.Diagnostics)
	}
}

func TestEvaluateBalanceAssertionIncludesSubaccounts(t *testing.T) {
	text := `2000-01-01 open Assets:Portfolio USD
2000-01-01 open Assets:Portfolio:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "subaccount posting"
  Assets:Portfolio:Cash 10 USD
  Equity:Opening -10 USD
2000-01-03 balance Assets:Portfolio 10 USD
`
	file, diagnostics := ParseText("subaccount-balance.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", diagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid || hasCode(evaluation.Diagnostics, "E-EVAL-BALANCE") {
		t.Fatalf("evaluation diagnostics=%+v", evaluation.Diagnostics)
	}
}

func TestEvaluateAccumulatedErrors(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-02 * "bad"
  Assets:Cash 10 USD
2000-01-03 balance Assets:Cash 0 USD
2000-01-04 close Assets:Cash
2000-01-05 * "after close"
  Assets:Cash 1 USD
`
	file, _ := ParseText("invalid.bean", []byte(text))
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if evaluation.Valid || !hasCode(evaluation.Diagnostics, "E-EVAL-UNBALANCED") || !hasCode(evaluation.Diagnostics, "E-EVAL-BALANCE") || !hasCode(evaluation.Diagnostics, "E-EVAL-POSTING") {
		t.Fatalf("evaluation=%s diagnostics=%+v", evaluation, evaluation.Diagnostics)
	}
}

func TestEvaluateLotsPricesAndPad(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH
2000-01-01 open Equity:Opening USD
2000-01-01 * "seed"
  Equity:Opening -100 USD
  Assets:Cash 100 USD
2000-01-02 pad Assets:Cash Equity:Opening
2000-01-03 balance Assets:Cash 150 USD
2000-01-04 price SH 10 USD
2000-01-05 * "buy"
  Assets:Shares 2 SH {10 USD}
  Assets:Cash -20 USD
2000-01-06 * "sell"
  Assets:Shares -1 SH {10 USD}
  Assets:Cash 10 USD
`
	file, parseDiagnostics := ParseText("lots.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if hasCode(evaluation.Diagnostics, "E-EVAL-INVENTORY") || len(evaluation.Prices["SH"]) != 1 {
		t.Fatalf("evaluation=%s diagnostics=%+v prices=%+v", evaluation, evaluation.Diagnostics, evaluation.Prices)
	}
	shares, ok := evaluation.Account("Assets:Shares")
	if !ok || len(shares.Positions) != 1 || shares.Positions[0].Units.String() != "1" {
		t.Fatalf("shares=%+v", shares)
	}
}

func TestEvaluateBooksReductionAcrossLots(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH
2000-01-01 open Equity:Opening USD
2000-01-02 * "buy one"
  Assets:Shares 1 SH {10 USD}
  Assets:Cash -10 USD
2000-01-03 * "buy two"
  Assets:Shares 2 SH {12 USD}
  Assets:Cash -24 USD
2000-01-04 * "sell both lots"
  Assets:Shares -3 SH {} @ 15 USD
  Assets:Cash 45 USD
`
	file, parseDiagnostics := ParseText("book-lots.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid || hasCode(evaluation.Diagnostics, "E-EVAL-INVENTORY") {
		t.Fatalf("evaluation diagnostics=%+v", evaluation.Diagnostics)
	}
	shares, ok := evaluation.Account("Assets:Shares")
	if !ok || len(shares.Positions) != 0 || shares.Balances["SH"].String() != "0" {
		t.Fatalf("shares=%+v", shares)
	}
	var booked *Transaction
	for index := len(evaluation.Entries) - 1; index >= 0; index-- {
		if tx, ok := evaluation.Entries[index].Directive.(*Transaction); ok {
			booked = tx
			break
		}
	}
	if booked == nil || len(booked.Postings) != 3 {
		t.Fatalf("booked transaction=%+v", booked)
	}
	if booked.Postings[0].Units == nil || booked.Postings[0].Units.Number.String() != "-1" || booked.Postings[1].Units == nil || booked.Postings[1].Units.Number.String() != "-2" {
		t.Fatalf("booked reduction postings=%+v", booked.Postings)
	}
	if booked.Postings[0].Cost == nil || booked.Postings[1].Cost == nil || len(booked.Postings[0].Cost.Components) == 0 || len(booked.Postings[1].Cost.Components) == 0 || booked.Postings[0].Cost.Components[0].Amount.Number.String() != "10" || booked.Postings[1].Cost.Components[0].Amount.Number.String() != "12" {
		t.Fatalf("booked lot costs=%+v", booked.Postings)
	}
}

func TestEvaluateFollowsTextualIncludeOrder(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	child := filepath.Join(dir, "child.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\ninclude \"child.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, []byte("2000-01-02 * \"included\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	graph, err := source.LoadGraph(entry)
	if err != nil {
		t.Fatal(err)
	}
	parsed, parseDiagnostics := ParseGraph(graph)
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := Evaluate(graph, parsed, EvalOptions{})
	if !hasCode(evaluation.Diagnostics, "E-EVAL-POSTING") {
		t.Fatalf("expected missing Equity lifecycle diagnostic: %+v", evaluation.Diagnostics)
	}
}

func TestEvaluateCurrencyOnlyCostInference(t *testing.T) {
	text := `2000-01-01 open Assets:Shares SH
2000-01-01 open Assets:Cash USD
2000-01-01 * "buy"
  Assets:Shares SH {2 USD}
  Assets:Cash -2 USD
`
	file, parseDiagnostics := ParseText("currency-only.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation diagnostics=%+v", evaluation.Diagnostics)
	}
	shares, ok := evaluation.Account("Assets:Shares")
	if !ok || shares.Balances["SH"].String() != "1" {
		t.Fatalf("shares=%+v", shares)
	}
}

func TestEvaluateNumberOnlyPostingInference(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-01 * "inferred currency"
  Assets:Cash 10
  Equity:Opening -10 USD
`
	file, parseDiagnostics := ParseText("number-only.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation diagnostics=%+v", evaluation.Diagnostics)
	}
	account, ok := evaluation.Account("Assets:Cash")
	if !ok || account.Balances["USD"].String() != "10" {
		t.Fatalf("account=%+v", account)
	}
}

func TestEvaluateTotalPriceAndDeferredBalance(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	child := filepath.Join(dir, "child.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n2000-01-02 balance Assets:Cash 10 USD\ninclude \"child.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, []byte("2000-01-01 open Equity:Opening USD\n2000-01-01 * \"buy\"\n  Assets:Cash 10 USD\n  Equity:Opening -10 USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	graph, err := source.LoadGraph(entry)
	if err != nil {
		t.Fatal(err)
	}
	parsed, parseDiagnostics := ParseGraph(graph)
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := Evaluate(graph, parsed, EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("deferred balance diagnostics=%+v", evaluation.Diagnostics)
	}

	text := `2000-01-01 open Assets:Shares SH
2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-01 * "buy"
  Assets:Shares 1 SH {10 USD}
  Assets:Cash -10 USD
2000-01-02 * "sell"
  Assets:Shares -1 SH @@ 10 USD
  Assets:Cash 10 USD
2000-01-03 balance Assets:Cash 0 USD ~ 0.01 USD
`
	file, parseDiagnostics := ParseText("price-total.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("price parse diagnostics=%+v", parseDiagnostics.All())
	}
	priceEvaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !priceEvaluation.Valid {
		t.Fatalf("price evaluation diagnostics=%+v", priceEvaluation.Diagnostics)
	}
}

func hasCode(ds []diagnostic.Diagnostic, prefix string) bool {
	for _, d := range ds {
		if len(prefix) <= len(d.Code) && d.Code[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func countCode(ds []diagnostic.Diagnostic, code string) int {
	count := 0
	for _, d := range ds {
		if d.Code == code {
			count++
		}
	}
	return count
}

// TestEvaluateReopenAfterCloseDiagnostic verifies that reopening a closed
// account is reported with the distinct E-EVAL-REOPEN code rather than the
// generic E-EVAL-OPEN duplicate-open diagnostic.
func TestEvaluateReopenAfterCloseDiagnostic(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-02 close Assets:Cash
2000-01-03 open Assets:Cash USD
`
	file, parseDiagnostics := ParseText("reopen.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !hasCode(evaluation.Diagnostics, "E-EVAL-REOPEN") {
		t.Fatalf("expected E-EVAL-REOPEN, diagnostics=%+v", evaluation.Diagnostics)
	}
	if hasCode(evaluation.Diagnostics, "E-EVAL-OPEN") {
		t.Fatalf("reopen should not emit E-EVAL-OPEN, diagnostics=%+v", evaluation.Diagnostics)
	}
}

// TestEvaluateInventoryDiagnosticEmittedOnce verifies that an oversell posting
// produces exactly one E-EVAL-INVENTORY diagnostic rather than a duplicate from
// both bookPosting and applyPosting.
func TestEvaluateInventoryDiagnosticEmittedOnce(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH
2000-01-01 open Equity:Opening USD
2000-01-02 * "buy"
  Assets:Shares 2 SH {10 USD}
  Assets:Cash -20 USD
2000-01-03 * "oversell"
  Assets:Shares -5 SH {10 USD}
  Assets:Cash 50 USD
`
	file, parseDiagnostics := ParseText("oversell.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !hasCode(evaluation.Diagnostics, "E-EVAL-INVENTORY") {
		t.Fatalf("expected E-EVAL-INVENTORY, diagnostics=%+v", evaluation.Diagnostics)
	}
	if count := countCode(evaluation.Diagnostics, "E-EVAL-INVENTORY"); count != 1 {
		t.Fatalf("E-EVAL-INVENTORY emitted %d times, want once: %+v", count, evaluation.Diagnostics)
	}
}
