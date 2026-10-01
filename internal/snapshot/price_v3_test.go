// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package snapshot

import (
	"orangecount/internal/ledger"
	"os"
	"path/filepath"
	"testing"
)

func TestV3TotalPriceRetainsSourceAndPinnedPerUnitRepresentation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	text := `2000-01-01 open Assets:Stock STK
2000-01-01 open Assets:Cash USD
2000-01-02 * "total quote"
  Assets:Stock -3 STK @@ 2 USD
  Assets:Cash 2 USD
`
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	result := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if result.Snapshot == nil {
		t.Fatal(result.Diagnostics)
	}
	tx := result.Snapshot.Evaluation().Entries[2].Directive.(*ledger.Transaction)
	p := tx.Postings[0]
	if !p.Price.Total || p.Price.Amount.Number.Raw != "2" {
		t.Fatal("source total price lost")
	}
	if p.Price.PerUnit == nil || p.Price.PerUnit.Number.Raw != "0.6666666666666666666666666667" {
		t.Fatal("pinned price normalization absent")
	}
}

func TestV3GroupsPostingOrderByWeightCurrency(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	text := `2000-01-01 open Assets:Cash USD,EUR
2000-01-01 open Equity:Opening USD,EUR
2000-01-02 * "interleaved currencies"
  Assets:Cash 1 USD
  Assets:Cash 2 EUR
  Equity:Opening -1 USD
  Equity:Opening -2 EUR
`
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	result := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if result.Snapshot == nil {
		t.Fatal(result.Diagnostics)
	}
	tx := result.Snapshot.Evaluation().Entries[2].Directive.(*ledger.Transaction)
	if tx.Postings[1].Account != "Equity:Opening" || tx.Postings[1].Units.Currency != "USD" {
		t.Fatal("booked posting order does not follow first-seen weight currency groups")
	}
}

func TestV3ZeroTotalQuoteIsValidAndPreservesSource(t *testing.T) {
	path := "../../testdata/fixtures/semantics/zero-total-price.bean"
	result := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if result.Snapshot == nil || !result.Snapshot.Valid() {
		t.Fatal(result.Diagnostics)
	}
	evaluation := result.Snapshot.Evaluation()
	transaction := evaluation.Entries[2].Directive.(*ledger.Transaction)
	posting := transaction.Postings[0]
	if posting.Price.Amount.Number.Raw != "1.00" || !posting.Price.Total || posting.Price.PerUnit == nil || !ledger.DecimalFromNumber(posting.Price.PerUnit.Number).IsZero() {
		t.Fatal("zero quantity requires a zero booked quote without replacing the source total")
	}
	if !ledger.PostingWeight(posting).IsZero() || !ledger.BookedPriceWeight(posting).IsZero() {
		t.Fatal("zero units contributed a nonzero weight")
	}
	foundSource := false
	for _, file := range result.Snapshot.Parsed() {
		for _, directive := range file.Directives {
			if tx, ok := directive.(*ledger.Transaction); ok && tx.Narration == "zero quantity nonzero total quote" {
				foundSource = true
				if tx.Postings[0].Price.PerUnit != nil || tx.Postings[0].Price.Amount.Number.Raw != "1.00" {
					t.Fatal("source AST was normalized in place")
				}
			}
		}
	}
	if !foundSource {
		t.Fatal("source transaction absent from parsed snapshot")
	}
	legacy := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.Legacy}})
	if legacy.Snapshot != nil && legacy.Snapshot.Valid() {
		t.Fatal("legacy accounting changed implicitly")
	}
}
