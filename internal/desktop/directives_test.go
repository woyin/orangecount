// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop_test

import (
	"errors"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"testing"
)

func TestJournalDirectiveModesFilterBeforePagingAndKeepTransactionCount(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/directives/all.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	view, err := session.Read(desktop.Request{View: "journal", JournalMode: "all", Limit: 1, Sort: "ascending"})
	if err != nil {
		t.Fatal(err)
	}
	page := view.Directives
	if page == nil || page.DateTo != "2000-01-07" {
		t.Fatal("missing captured directive cutoff")
	}
	if page == nil || len(page.Records) != 1 || page.Page.TotalCount != 18 || !page.Page.HasMore || view.Journal != nil || view.Summary.TransactionsCount != 3 {
		t.Fatal("directive pagination changed transaction counts")
	}
	filtered, err := session.Read(desktop.Request{View: "journal", JournalMode: "all", DirectiveKind: "note", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Directives.Page.TotalCount != 2 || len(filtered.Directives.Records) != 1 {
		t.Fatal("kind filtering after page")
	}
	next, err := session.Read(desktop.Request{View: "journal", JournalMode: "all", DirectiveKind: "note", SnapshotID: view.State.SnapshotID, Offset: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if next.Directives.Records[0].ID == filtered.Directives.Records[0].ID {
		t.Fatal("duplicate records collapsed")
	}
	config, err := session.Read(desktop.Request{View: "journal", JournalMode: "config", DateFrom: "2099-01-01", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if config.Directives.DateFrom != "" || config.Directives.DateTo != "" {
		t.Fatal("configuration acquired a date scope")
	}
	if config.Directives.Page.TotalCount != 8 || config.Directives.Records[0].Date != nil || config.Directives.Records[0].Kind != "option" {
		t.Fatal("config lexical order or date scope changed")
	}
	if _, err := session.Read(desktop.Request{View: "journal", JournalMode: "all", Offset: 1}); !errors.Is(err, desktop.ErrRequest) {
		t.Fatal("unbound page accepted")
	}
	for _, request := range []desktop.Request{{View: "journal", JournalMode: "wrong"}, {View: "query", JournalMode: "all"}, {View: "journal", JournalMode: "transactions", DirectiveKind: "note"}, {View: "journal", JournalMode: "all", DirectiveKind: "wrong"}} {
		if _, err := session.Read(request); !errors.Is(err, desktop.ErrRequest) {
			t.Fatal("invalid directive request accepted")
		}
	}
}

func TestDirectiveFieldsPreserveExactMetadataAndGeneratedOrigins(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/directives/all.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	view, err := session.Read(desktop.Request{View: "journal", JournalMode: "all", Sort: "ascending"})
	if err != nil {
		t.Fatal(err)
	}
	var padID string
	var generated bool
	var custom bool
	var document bool
	for _, record := range view.Directives.Records {
		if record.Kind == "pad" {
			padID = record.ID
		}
		if record.Generated && record.SourceKind == "pad" {
			generated = true
			if record.OriginID == nil || *record.OriginID != padID || len(record.Postings) != 2 || record.Postings[0].Posting.Amount.Exact != "2" || record.Metadata[0].Value.Value != "pad" || record.Postings[0].Metadata[0].Value.Value != "balance" {
				t.Fatal("pad provenance or effective metadata lost")
			}
		}
		if record.Kind == "custom" {
			custom = true
			if record.Fields[1].Value.Value.(query.PresentedAmount).Number.Exact != "1" || record.Fields[2].Value.Value != true || record.Fields[3].Value.Value != "2000-01-01" {
				t.Fatal("custom typed values lost")
			}
		}
		if record.Kind == "document" {
			document = true
			if len(record.Tags) != 1 || len(record.Links) != 1 || len(record.Metadata) != 1 {
				t.Fatal("document metadata/tags/links lost")
			}
		}
	}
	if !generated || !custom || !document {
		t.Fatal("directive payload omitted types")
	}
	filtered, err := session.Read(desktop.Request{View: "journal", JournalMode: "all", Text: "public", DirectiveKind: "transaction"})
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Directives.Page.TotalCount != 1 {
		t.Fatal("effective stack metadata not searchable")
	}
}
