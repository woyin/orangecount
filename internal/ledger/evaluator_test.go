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

func TestEvaluateCompletesInterpolatedPostingInEntryStream(t *testing.T) {
	// Beancount hands back a fully interpolated transaction, so the elided
	// posting must carry its computed amount in Entries. Consumers that walk
	// the entry stream (journal rows, query results, report charts) otherwise
	// skip it: a purchase would count the shares it received but not the cash
	// it paid, silently inflating asset charts.
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH
2000-01-01 open Expenses:Fees USD
2000-01-02 * "buy with an elided cash posting"
  Assets:Cash
  Assets:Shares 10 SH {7 USD}
  Expenses:Fees 1 USD
`
	file, parseDiagnostics := ParseText("interpolated.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%s diagnostics=%+v", evaluation, evaluation.Diagnostics)
	}
	var booked *Transaction
	for index := len(evaluation.Entries) - 1; index >= 0; index-- {
		if tx, ok := evaluation.Entries[index].Directive.(*Transaction); ok {
			booked = tx
			break
		}
	}
	if booked == nil {
		t.Fatal("no booked transaction in the entry stream")
	}
	var cash *Posting
	for index := range booked.Postings {
		if booked.Postings[index].Account == "Assets:Cash" {
			cash = &booked.Postings[index]
		}
	}
	if cash == nil || cash.Units == nil {
		t.Fatalf("interpolated posting has no units: %+v", booked.Postings)
	}
	if got := DecimalFromNumber(cash.Units.Number).String(); got != "-71" {
		t.Fatalf("interpolated cash units=%s want -71", got)
	}
	if cash.Units.Currency != "USD" {
		t.Fatalf("interpolated cash currency=%q want USD", cash.Units.Currency)
	}
}

func TestEvaluateWeighsCostOverPriceWhenBothPresent(t *testing.T) {
	// A posting carrying both a cost and a price balances at its cost; the
	// price is reporting information. Weighting at the price instead makes the
	// proceeds cancel the reduction exactly, so the realized gain silently
	// collapses to zero instead of landing on the inferred income posting.
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH
2000-01-01 open Income:Gains USD
2000-01-02 * "buy"
  Assets:Shares 10 SH {10 USD}
  Assets:Cash -100 USD
2000-01-03 * "sell at a gain"
  Assets:Shares -10 SH {10 USD} @ 15 USD
  Assets:Cash 150 USD
  Income:Gains
`
	file, parseDiagnostics := ParseText("cost-over-price.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%s diagnostics=%+v", evaluation, evaluation.Diagnostics)
	}
	gains, ok := evaluation.Account("Income:Gains")
	if !ok || gains.Balances["USD"].String() != "-50" {
		t.Fatalf("realized gain=%+v want -50 USD", gains)
	}
}

func TestEvaluateAccumulatesRepeatedOperatingCurrency(t *testing.T) {
	text := `option "operating_currency" "CNY"
option "operating_currency" "USD"
option "operating_currency" "CNY"
2000-01-01 open Assets:Cash CNY
2000-01-01 open Equity:Opening CNY
2000-01-02 * "opening"
  Assets:Cash 1 CNY
  Equity:Opening -1 CNY
`
	file, parseDiagnostics := ParseText("operating-currency.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%s diagnostics=%+v", evaluation, evaluation.Diagnostics)
	}
	// Beancount accumulates repeated operating_currency declarations rather
	// than letting the last one win; the first declared currency is the
	// ledger's primary one and must stay first, with duplicates collapsed.
	if got := evaluation.Options["operating_currency"]; got != "CNY USD" {
		t.Fatalf("operating_currency=%q want %q", got, "CNY USD")
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
	// The reduction is weighted at the cost of the lots removed (1x10 + 2x12 =
	// 34 USD), not at the 15 USD sale price, so the 45 USD proceeds leave an
	// 11 USD realized gain for the inferred posting to absorb. Beancount
	// rejects this transaction outright without that gain posting.
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH
2000-01-01 open Equity:Opening USD
2000-01-01 open Income:Gains USD
2000-01-02 * "buy one"
  Assets:Shares 1 SH {10 USD}
  Assets:Cash -10 USD
2000-01-03 * "buy two"
  Assets:Shares 2 SH {12 USD}
  Assets:Cash -24 USD
2000-01-04 * "sell both lots"
  Assets:Shares -3 SH {} @ 15 USD
  Assets:Cash 45 USD
  Income:Gains
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
	if booked == nil || len(booked.Postings) != 4 {
		t.Fatalf("booked transaction=%+v", booked)
	}
	gains, ok := evaluation.Account("Income:Gains")
	if !ok || gains.Balances["USD"].String() != "-11" {
		t.Fatalf("realized gain=%+v", gains)
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

func TestEvaluateAverageBookingMergesLotsBeforePartialReduction(t *testing.T) {
	// The independent worked basis is (100×10 + 200×12) / 300 = 34/3 USD per
	// share. Selling 100 shares must leave one 200-share lot at that basis.
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH "AVERAGE"
2000-01-01 open Income:Gains USD
2000-01-02 * "buy first lot"
  Assets:Shares 100 SH {10 USD, 2000-01-02, "first"}
  Assets:Cash -1000 USD
2000-01-03 * "buy second lot"
  Assets:Shares 200 SH {12 USD, 2000-01-03, "second"}
  Assets:Cash -2400 USD
2000-01-04 * "sell part"
  Assets:Shares -100 SH {} @ 15 USD
  Assets:Cash 1500 USD
  Income:Gains
`
	file, parseDiagnostics := ParseText("average-partial.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid || hasCode(evaluation.Diagnostics, "E-EVAL-INVENTORY") {
		t.Fatalf("evaluation diagnostics=%+v", evaluation.Diagnostics)
	}
	shares, ok := evaluation.Account("Assets:Shares")
	if !ok || shares.Booking != "AVERAGE" || len(shares.Positions) != 1 {
		t.Fatalf("shares=%+v", shares)
	}
	position := shares.Positions[0]
	expectedCost, err := ParseDecimal("34/3")
	if err != nil {
		t.Fatal(err)
	}
	if position.Units.String() != "200" || position.Cost == nil || position.Cost.Currency != "USD" || !position.Cost.Number.Equal(expectedCost) || position.Cost.Date == nil || position.Cost.Date.Raw != "2000-01-02" || position.Cost.Label != "" {
		t.Fatalf("remaining position=%+v, want 200 SH at 34/3 USD", position)
	}
}

func TestEvaluateAverageBookingRejectsExplicitReductionCost(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH "AVERAGE"
2000-01-01 open Income:Gains USD
2000-01-02 * "buy"
  Assets:Shares 100 SH {10 USD}
  Assets:Cash -1000 USD
2000-01-03 * "explicit-cost sell"
  Assets:Shares -50 SH {10 USD} @ 15 USD
  Assets:Cash 750 USD
  Income:Gains
`
	file, parseDiagnostics := ParseText("average-explicit-cost.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if count := countCode(evaluation.Diagnostics, "E-EVAL-INVENTORY"); count != 1 {
		t.Fatalf("E-EVAL-INVENTORY emitted %d times, want once: %+v", count, evaluation.Diagnostics)
	}
	shares, ok := evaluation.Account("Assets:Shares")
	if !ok || len(shares.Positions) != 1 || shares.Positions[0].Units.String() != "100" || shares.Positions[0].Cost == nil || shares.Positions[0].Cost.Number.String() != "10" {
		t.Fatalf("explicit-cost reduction must leave the lot unbooked, shares=%+v", shares)
	}
	if got := shares.Balances["SH"].String(); got != "100" {
		t.Fatalf("explicit-cost reduction must not change the account balance, got %s", got)
	}
}

func TestEvaluateAverageBookingLiquidatesMergedLots(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH "AVERAGE"
2000-01-01 open Income:Gains USD
2000-01-02 * "buy first lot"
  Assets:Shares 100 SH {10 USD}
  Assets:Cash -1000 USD
2000-01-03 * "buy second lot"
  Assets:Shares 200 SH {12 USD}
  Assets:Cash -2400 USD
2000-01-04 * "sell all"
  Assets:Shares -300 SH {} @ 15 USD
  Assets:Cash 4500 USD
  Income:Gains
`
	file, parseDiagnostics := ParseText("average-liquidation.bean", []byte(text))
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
}

func TestEvaluateAverageBookingRejectsCrossCostCurrencyReduction(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD, HKD
2000-01-01 open Assets:Shares SH "AVERAGE"
2000-01-01 open Income:Gains USD
2000-01-02 * "buy USD lot"
  Assets:Shares 100 SH {10 USD}
  Assets:Cash -1000 USD
2000-01-03 * "buy HKD lot"
  Assets:Shares 100 SH {80 HKD}
  Assets:Cash -8000 HKD
2000-01-04 * "sell"
  Assets:Shares -100 SH {} @ 15 USD
  Assets:Cash 1500 USD
  Income:Gains
`
	file, parseDiagnostics := ParseText("average-cross-currency.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if count := countCode(evaluation.Diagnostics, "E-EVAL-INVENTORY"); count != 1 {
		t.Fatalf("E-EVAL-INVENTORY emitted %d times, want once: %+v", count, evaluation.Diagnostics)
	}
	shares, ok := evaluation.Account("Assets:Shares")
	if !ok || len(shares.Positions) != 2 || shares.Positions[0].Units.String() != "100" || shares.Positions[1].Units.String() != "100" {
		t.Fatalf("cross-currency reduction must leave lots unbooked, shares=%+v", shares)
	}
	if got := shares.Balances["SH"].String(); got != "200" {
		t.Fatalf("cross-currency reduction must not change the account balance, got %s", got)
	}
}

func TestEvaluateAverageBookingRejectsOversell(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH "AVERAGE"
2000-01-01 open Income:Gains USD
2000-01-02 * "buy"
  Assets:Shares 100 SH {10 USD}
  Assets:Cash -1000 USD
2000-01-03 * "oversell"
  Assets:Shares -101 SH {} @ 15 USD
  Assets:Cash 1515 USD
  Income:Gains
`
	file, parseDiagnostics := ParseText("average-oversell.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if count := countCode(evaluation.Diagnostics, "E-EVAL-INVENTORY"); count != 1 {
		t.Fatalf("E-EVAL-INVENTORY emitted %d times, want once: %+v", count, evaluation.Diagnostics)
	}
}

func TestEvaluateAverageBookingReducesSingleLot(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH "AVERAGE"
2000-01-01 open Income:Gains USD
2000-01-02 * "buy"
  Assets:Shares 100 SH {10 USD}
  Assets:Cash -1000 USD
2000-01-03 * "sell part"
  Assets:Shares -40 SH {} @ 15 USD
  Assets:Cash 600 USD
  Income:Gains
`
	file, parseDiagnostics := ParseText("average-single-lot.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid || hasCode(evaluation.Diagnostics, "E-EVAL-INVENTORY") {
		t.Fatalf("evaluation diagnostics=%+v", evaluation.Diagnostics)
	}
	shares, ok := evaluation.Account("Assets:Shares")
	if !ok || len(shares.Positions) != 1 || shares.Positions[0].Units.String() != "60" || shares.Positions[0].Cost == nil || shares.Positions[0].Cost.Number.String() != "10" {
		t.Fatalf("shares=%+v", shares)
	}
}

func TestEvaluateFIFOReductionRemainsUnchangedWithExplicitBooking(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH "FIFO"
2000-01-01 open Income:Gains USD
2000-01-02 * "buy first lot"
  Assets:Shares 100 SH {10 USD}
  Assets:Cash -1000 USD
2000-01-03 * "buy second lot"
  Assets:Shares 200 SH {12 USD}
  Assets:Cash -2400 USD
2000-01-04 * "sell part"
  Assets:Shares -100 SH {} @ 15 USD
  Assets:Cash 1500 USD
  Income:Gains
`
	file, parseDiagnostics := ParseText("fifo-regression.bean", []byte(text))
	if parseDiagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", parseDiagnostics.All())
	}
	evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
	if !evaluation.Valid || hasCode(evaluation.Diagnostics, "E-EVAL-INVENTORY") {
		t.Fatalf("evaluation diagnostics=%+v", evaluation.Diagnostics)
	}
	shares, ok := evaluation.Account("Assets:Shares")
	if !ok || shares.Booking != "FIFO" || len(shares.Positions) != 1 || shares.Positions[0].Units.String() != "200" || shares.Positions[0].Cost == nil || shares.Positions[0].Cost.Number.String() != "12" {
		t.Fatalf("FIFO booking changed unexpectedly, shares=%+v", shares)
	}
}
