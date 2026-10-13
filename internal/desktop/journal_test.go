// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orangecount/internal/desktop"
	"orangecount/internal/snapshot"
)

func TestJournalPagesFilterBeforeCountingAndStayOnOneSnapshot(t *testing.T) {
	s, err := desktop.Open("../../testdata/fixtures/native/cashflow.bean", snapshot.WatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	read := func(raw string) desktop.ViewResponse {
		t.Helper()
		var request desktop.Request
		if err := json.Unmarshal([]byte(raw), &request); err != nil {
			t.Fatal(err)
		}
		view, err := s.Read(request)
		if err != nil {
			t.Fatal(err)
		}
		return view
	}
	first := read(`{"view":"journal","limit":1,"sort":"descending"}`)
	if first.Journal.TotalCount != 2 || len(first.Journal.Transactions) != 1 || first.Journal.Transactions[0].Date != "2026-01-03" {
		t.Fatalf("first page must be the latest of two transactions: %+v", first.Journal)
	}
	second := read(`{"view":"journal","limit":1,"offset":1,"snapshot_id":"` + first.State.SnapshotID + `"}`)
	if len(second.Journal.Transactions) != 1 || second.Journal.Transactions[0].Date != "2026-01-02" {
		t.Fatal("second page must show the older transaction")
	}
	filtered := read(`{"view":"journal","account":"Expenses","text":"food","date_from":"2026-01-03","date_to":"2026-01-03"}`)
	if filtered.Journal.TotalCount != 1 || len(filtered.Journal.Transactions[0].Postings) != 2 {
		t.Fatal("filters must count matching whole transactions")
	}
	if _, err := s.Read(desktop.Request{View: "journal", SnapshotID: "old"}); err != desktop.ErrStale {
		t.Fatal("pages cannot mix snapshots")
	}
	state, _ := s.Poll()
	if state.BuildCount != 1 {
		t.Fatal("filters must not rebuild")
	}
}

func TestJournalPaginationScalesTo52500Transactions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large-journal.bean")
	var source strings.Builder
	source.Grow(5_250_000)
	source.WriteString("option \"operating_currency\" \"USD\"\n2026-01-01 open Assets:Cash USD\n2026-01-01 open Expenses:Food USD\n")
	for index := 0; index < 52_500; index++ {
		fmt.Fprintf(&source, "2026-01-02 * \"Txn %05d\"\n  Assets:Cash 1 USD\n  Expenses:Food -1 USD\n", index)
	}
	if err := os.WriteFile(path, []byte(source.String()), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.Open(path, snapshot.WatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first, err := s.Read(desktop.Request{View: "journal", JournalMode: "transactions", Sort: "descending", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if first.Journal.TotalCount != 52_500 || len(first.Journal.Transactions) != 100 || !first.Journal.HasMore || first.Journal.Transactions[0].Narration != "Txn 52499" {
		t.Fatalf("large first page has wrong total, bounds, or order: total=%d returned=%d more=%v first=%q", first.Journal.TotalCount, len(first.Journal.Transactions), first.Journal.HasMore, first.Journal.Transactions[0].Narration)
	}
	second, err := s.Read(desktop.Request{View: "journal", JournalMode: "transactions", Sort: "descending", Offset: 100, Limit: 100, SnapshotID: first.State.SnapshotID})
	if err != nil {
		t.Fatal(err)
	}
	if second.State.SnapshotID != first.State.SnapshotID || second.Journal.TotalCount != 52_500 || len(second.Journal.Transactions) != 100 || !second.Journal.HasMore || second.Journal.Transactions[0].Narration != "Txn 52399" {
		t.Fatalf("large second page changed snapshot, total, bounds, or order: snapshot=%q total=%d returned=%d more=%v first=%q", second.State.SnapshotID, second.Journal.TotalCount, len(second.Journal.Transactions), second.Journal.HasMore, second.Journal.Transactions[0].Narration)
	}
	state, err := s.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if state.BuildCount != 1 {
		t.Fatalf("journal paging rebuilt the ledger: %d builds", state.BuildCount)
	}
}

func TestJournalIdentitySurvivesLineShiftsButNotAmbiguousDuplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	base, err := os.ReadFile("../../testdata/fixtures/native/cashflow.bean")
	if err != nil {
		t.Fatal(err)
	}
	write := func(text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(string(base))
	s, err := desktop.Open(path, snapshot.WatchOptions{PollInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	read := func() desktop.ViewResponse {
		t.Helper()
		v, err := s.Read(desktop.Request{View: "journal"})
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	first := read()
	write("; inserted comment\n" + string(base))
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	shifted := read()
	if shifted.Journal.Transactions[0].ID != first.Journal.Transactions[0].ID || shifted.Journal.Transactions[0].Source.Line != first.Journal.Transactions[0].Source.Line+1 {
		t.Fatal("unique transaction must survive a source line shift")
	}
	duplicate := "2026-01-03 * \"Food\"\n  Expenses:Food 300 USD\n  Assets:Bank -300 USD\n"
	write(string(base) + duplicate)
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	a := read()
	if a.Journal.Transactions[0].ID == a.Journal.Transactions[1].ID {
		t.Fatal("duplicate transactions need distinct snapshot-local IDs")
	}
	write("; new revision\n" + string(base) + duplicate)
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	b := read()
	if a.Journal.Transactions[0].ID == b.Journal.Transactions[0].ID {
		t.Fatal("ambiguous duplicate expansion must not transfer across revisions")
	}
}

func TestJournalProjectionIncludesExactCostsPricesAndPostingMetadata(t *testing.T) {
	path := "../../testdata/fixtures/native/holdings-trades.bean"
	s, err := desktop.Open(path, snapshot.WatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state, err := s.Poll()
	if err != nil {
		t.Fatal(err)
	}
	response, err := s.Read(desktop.Request{
		View: "journal", JournalMode: "transactions", Sort: "ascending", Limit: 100,
		SnapshotID: state.SnapshotID, Window: "all", Locale: "zh-CN",
	})
	if err != nil {
		t.Fatal(err)
	}
	find := func(narration string) *desktop.JournalTransactionDTO {
		t.Helper()
		for index := range response.Journal.Transactions {
			if response.Journal.Transactions[index].Narration == narration {
				return &response.Journal.Transactions[index]
			}
		}
		t.Fatalf("journal transaction %q is missing", narration)
		return nil
	}

	lots := find("first lots")
	var exactCost string
	for _, posting := range lots.Postings {
		if posting.Account == "Assets:Fifo" && posting.Cost != nil {
			exactCost = posting.Cost.Display
		}
	}
	if exactCost != "{10 USD}" {
		t.Fatalf("journal lost the exact posting cost: %q", exactCost)
	}

	fee := find("attributed fee")
	var exactPrice string
	var postingMetadata *desktop.DirectiveMetadata
	var metadataSource desktop.SourceLocation
	for _, posting := range fee.Postings {
		if posting.Account == "Assets:Fees" && posting.Price != nil {
			exactPrice = posting.Price.Number.Exact + " " + posting.Price.Currency
		}
		if posting.Account == "Expenses:TradeFees" && len(posting.Metadata) > 0 {
			postingMetadata = &posting.Metadata[0]
			metadataSource = posting.Metadata[0].Source
		}
	}
	if exactPrice != "20 USD" || postingMetadata == nil || postingMetadata.Key != "trade_fee_for" || postingMetadata.Value.Value != "Assets:Fees" || metadataSource.Path == "" || metadataSource.Line == 0 {
		t.Fatalf("journal lost exact price or typed posting metadata/source: price=%q metadata=%+v source=%+v transaction=%+v", exactPrice, postingMetadata, metadataSource, fee)
	}
}
