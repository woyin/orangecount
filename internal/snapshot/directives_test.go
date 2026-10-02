// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package snapshot_test

import (
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectiveCatalogueKeepsLexicalConfigurationAndEffectivePadOrigin(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "main.bean")
	child := filepath.Join(root, "child.bean")
	main := `option "title" "before"
include "child.bean"
option "title" "after"
include "child.bean"
pushmeta project: "parent"
2000-01-02 * "source transaction"
  Assets:Padded 1 USD
  Equity:Padding -1 USD
popmeta project:
2000-01-03 pad Assets:Padded Equity:Padding
2000-01-04 balance Assets:Padded 3 USD
`
	included := `option "operating_currency" "USD"
2000-01-01 open Assets:Padded USD
2000-01-01 open Equity:Padding USD
`
	if err := os.WriteFile(entry, []byte(main), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, []byte(included), 0600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.BuildWithOptions(entry, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if built.Snapshot == nil {
		t.Fatal(built.Diagnostics)
	}
	catalogue := built.Snapshot.DirectiveCatalogue()
	expected := []ledger.DirectiveKind{ledger.KindOption, ledger.KindInclude, ledger.KindOption, ledger.KindOption, ledger.KindInclude, ledger.KindPushMeta, ledger.KindPopMeta}
	if len(catalogue.Configuration) != len(expected) {
		t.Fatal("configuration records missing or duplicate include expanded")
	}
	for i, kind := range expected {
		if catalogue.Configuration[i].Kind != kind || catalogue.Configuration[i].Date != "" {
			t.Fatal("lexical configuration order or undated identity changed")
		}
	}
	var padID string
	var generated *snapshot.DirectiveRecord
	var txCount int
	for i, record := range catalogue.Dated {
		if record.Kind == ledger.KindPad {
			padID = record.ID
		}
		if record.Kind == ledger.KindTxn {
			txCount++
			if record.Generated {
				generated = &catalogue.Dated[i]
			} else {
				tx := record.Directive.(*ledger.Transaction)
				if len(tx.Meta) != 1 || tx.Meta[0].Key != "project" {
					t.Fatal("effective metadata missing")
				}
			}
		}
	}
	if txCount != 2 || generated == nil || generated.OriginID != padID || generated.SourceKind != ledger.KindPad || generated.Date != "2000-01-03" {
		t.Fatal("pad generation/source relationship lost")
	}
	catalogue.Configuration[0].Directive = ledger.Option{}
	tx := generated.Directive.(*ledger.Transaction)
	tx.Narration = "mutated"
	fresh := built.Snapshot.DirectiveCatalogue()
	if fresh.Configuration[0].Directive.(ledger.Option).Value != "before" {
		t.Fatal("configuration copy changed snapshot")
	}
	for _, record := range fresh.Dated {
		if record.Generated && record.Directive.(*ledger.Transaction).Narration == "mutated" {
			t.Fatal("dated deep copy changed snapshot")
		}
	}
}

func TestDirectiveCatalogueCoversEverySourceKindAndPreservesDuplicateIdentity(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"all.bean", "included.bean"} {
		data, err := os.ReadFile("../../testdata/fixtures/directives/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	entry := filepath.Join(root, "all.bean")
	build := func() *snapshot.Snapshot {
		result := snapshot.BuildWithOptions(entry, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
		if result.Snapshot == nil {
			t.Fatal(result.Diagnostics)
		}
		return result.Snapshot
	}
	before := build().DirectiveCatalogue()
	kinds := map[ledger.DirectiveKind]bool{}
	ids := map[string]bool{}
	stable := map[string]string{}
	var duplicateIDs []string
	for _, record := range append(before.Configuration, before.Dated...) {
		if record.ID == "" || ids[record.ID] {
			t.Fatal("record identity missing or collided")
		}
		ids[record.ID] = true
		kinds[record.SourceKind] = true
		if record.Kind == ledger.KindNote {
			duplicateIDs = append(duplicateIDs, record.ID)
		} else if !record.Generated {
			stable[record.File+record.Directive.RawText()] = record.ID
		}
	}
	for _, kind := range []ledger.DirectiveKind{ledger.KindOption, ledger.KindPlugin, ledger.KindInclude, ledger.KindPushTag, ledger.KindPopTag, ledger.KindPushMeta, ledger.KindPopMeta, ledger.KindOpen, ledger.KindClose, ledger.KindCommodity, ledger.KindBalance, ledger.KindPad, ledger.KindEvent, ledger.KindQuery, ledger.KindPrice, ledger.KindDocument, ledger.KindNote, ledger.KindCustom, ledger.KindTxn, ledger.KindDialect} {
		if !kinds[kind] {
			t.Fatal("source kind omitted", kind)
		}
	}
	if len(duplicateIDs) != 2 || duplicateIDs[0] == duplicateIDs[1] {
		t.Fatal("duplicate source records collapsed")
	}
	bytes, err := os.ReadFile(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry, append([]byte("; unrelated inserted line\n"), bytes...), 0600); err != nil {
		t.Fatal(err)
	}
	after := build().DirectiveCatalogue()
	for _, record := range append(after.Configuration, after.Dated...) {
		if record.Kind == ledger.KindNote {
			if record.ID == duplicateIDs[0] || record.ID == duplicateIDs[1] {
				t.Fatal("ambiguous duplicates restored across changed snapshot")
			}
		} else if !record.Generated {
			if record.ID != stable[record.File+record.Directive.RawText()] {
				t.Fatal("unambiguous source identity changed after inserted line")
			}
		}
	}
}

func TestFailedReloadCannotMixNewConfigurationIntoPublishedCatalogue(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "main.bean")
	initial := []byte("option \"title\" \"published\"\n2000-01-01 open Assets:Cash USD\n")
	if err := os.WriteFile(entry, initial, 0600); err != nil {
		t.Fatal(err)
	}
	result := snapshot.BuildWithOptions(entry, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if result.Snapshot == nil {
		t.Fatal(result.Diagnostics)
	}
	store := snapshot.NewStore(result.Snapshot)
	if err := os.WriteFile(entry, []byte("option \"title\" \"unpublished\"\nnot a directive\n"), 0600); err != nil {
		t.Fatal(err)
	}
	failed := store.Reload(entry, snapshot.BuildOptions{})
	if failed.Snapshot != nil || store.Current().ID != result.Snapshot.ID || len(store.Diagnostics()) == 0 {
		t.Fatal("failed reload replaced snapshot or lost diagnostics")
	}
	catalogue := store.Current().DirectiveCatalogue()
	if len(catalogue.Configuration) != 1 || catalogue.Configuration[0].Directive.(ledger.Option).Value != "published" || len(catalogue.Dated) != 1 {
		t.Fatal("new invalid configuration mixed into published records")
	}
}

func TestDirectiveCatalogueCopiesExactNumbersMetadataAndTags(t *testing.T) {
	built := snapshot.BuildWithOptions("../../testdata/fixtures/directives/all.bean", snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if built.Snapshot == nil {
		t.Fatal(built.Diagnostics)
	}
	catalogue := built.Snapshot.DirectiveCatalogue()
	for _, record := range catalogue.Dated {
		if record.Kind == ledger.KindCustom {
			custom := record.Directive.(ledger.Custom)
			custom.Values[0].Amount.Number.Rat.SetInt64(999)
		}
		if record.Kind == ledger.KindTxn && !record.Generated {
			tx := record.Directive.(*ledger.Transaction)
			tx.Postings[0].Units.Number.Rat.SetInt64(999)
			tx.Meta[0].Value.String = "changed"
			tx.Tags[0] = "changed"
		}
	}
	for _, record := range built.Snapshot.DirectiveCatalogue().Dated {
		if record.Kind == ledger.KindCustom {
			if record.Directive.(ledger.Custom).Values[0].Amount.Number.Rat.RatString() != "1" {
				t.Fatal("exact custom amount changed")
			}
		}
		if record.Kind == ledger.KindTxn && !record.Generated {
			tx := record.Directive.(*ledger.Transaction)
			if tx.Postings[0].Units.Number.Rat.RatString() != "1" || tx.Meta[0].Value.String != "public" || tx.Tags[0] == "changed" {
				t.Fatal("exact posting or metadata/tag copy changed source")
			}
		}
	}
}

func TestBlockDialectTransactionsKeepHeaderOrigin(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "block.bean")
	input := `2000-01-01 open Assets:Cash USD
2000-01-01 open Expenses:One USD
2000-01-01 open Expenses:Two USD
2000-01-02 * "block"
  1 USD @Assets:Cash -> @Expenses:One
  2 USD @Assets:Cash -> @Expenses:Two
2000-01-02 * "block"
  1 USD @Assets:Cash -> @Expenses:One
  2 USD @Assets:Cash -> @Expenses:Two
`
	if err := os.WriteFile(entry, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.BuildWithOptions(entry, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if built.Snapshot == nil {
		t.Fatal(built.Diagnostics)
	}
	page, err := built.Snapshot.SelectDirectives(snapshot.DirectiveSelection{Mode: "all", Kind: "transaction"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Records) != 2 {
		t.Fatal("lost block transactions")
	}
	for _, record := range page.Records {
		if !record.Generated || record.SourceKind != ledger.KindTxn || record.OriginID == "" {
			t.Fatal("block header origin missing", record)
		}
	}
	if page.Records[0].OriginID == page.Records[1].OriginID {
		t.Fatal("duplicate source headers merged")
	}
}
