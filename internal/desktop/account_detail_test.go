// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
)

func TestAccountDetailDistinguishesScopeAndKeepsOpeningInventory(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/account-detail.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3, Watch: snapshot.WatchOptions{PollInterval: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "account", Account: "Assets:Test", DateFrom: "2000-01-03", DateTo: "2000-01-03", Text: "no matching transaction"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Account *struct {
			Name    string                             `json:"account"`
			Scope   string                             `json:"scope"`
			Booking string                             `json:"booking"`
			Direct  map[string]report.PresentedDecimal `json:"direct_amounts"`
			Subtree map[string]report.PresentedDecimal `json:"subtree_amounts"`
			Change  map[string]report.PresentedDecimal `json:"period_change"`
		} `json:"account_detail"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	detail := decoded.Account
	if detail == nil || detail.Name != "Assets:Test" || detail.Scope != "subtree" || detail.Booking != "LIFO" {
		t.Fatalf("wrong account identity/effective booking: %s", data)
	}
	if detail.Direct["USD"].Exact != "10" || detail.Subtree["USD"].Exact != "25" || detail.Change["USD"].Exact != "-5" {
		t.Fatalf("period must not erase opening inventory or search change totals: %s", data)
	}
	if view.Journal == nil || view.Journal.TotalCount != 0 {
		t.Fatal("text search should apply only to the account journal")
	}
}

func TestAccountDetailDirectLifecycleAndParentDirectories(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/account-detail.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	direct, err := s.Read(desktop.Request{View: "account", Account: "Assets:Test", Scope: "direct", DateFrom: "2000-01-03", DateTo: "2000-01-06"})
	if err != nil {
		t.Fatal(err)
	}
	if len(direct.Account.PeriodChange) != 0 || direct.Journal.TotalCount != 0 || direct.Account.DirectAmounts["USD"].Exact != "10" {
		t.Fatal("direct scope must exclude child activity")
	}
	child, err := s.Read(desktop.Request{View: "account", Account: "Assets:Test:Child", DateTo: "2000-01-06"})
	if err != nil {
		t.Fatal(err)
	}
	if child.Account.Lifecycle != "closed" || *child.Account.Closed != "2000-01-05" || child.Account.Booking != "FIFO" || child.Account.DirectAmounts["USD"].Exact != "15" || len(child.Account.Sources) != 2 {
		t.Fatal("close must retain history, override booking and original lifecycle sources")
	}
	if child.Account.Sources[0].Source.Line != 5 || child.Account.Sources[0].Kind != "open" || child.Account.Sources[1].Kind != "close" {
		t.Fatal("lifecycle sources must be original directives")
	}
	parent, err := s.Read(desktop.Request{View: "account", Account: "Assets", DateTo: "2000-01-02"})
	if err != nil {
		t.Fatal(err)
	}
	if parent.Account.Explicit || parent.Account.Lifecycle != "implicit" || parent.Account.SubtreeAmounts["USD"].Exact != "1029" {
		t.Fatal("tree directories must have truthful subtree totals")
	}
	if _, err := s.Read(desktop.Request{View: "account", Account: "Assets:Testi"}); !errors.Is(err, report.ErrAccountUnknown) {
		t.Fatal("similar prefix must not select another account")
	}
	for _, request := range []desktop.Request{{View: "account"}, {View: "account", Account: "Assets:Test", Scope: "all"}, {View: "journal", Scope: "direct"}} {
		if _, err := s.Read(request); !errors.Is(err, desktop.ErrRequest) {
			t.Fatal("unknown account request combinations must fail")
		}
	}
}

func TestAccountDetailHistoryCarriesBalanceAcrossQuietMonths(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/account-detail.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "account", Account: "Assets:Test", DateFrom: "2000-01-03", DateTo: "2000-02-29"})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(view.Account)
	var detail struct {
		History []struct {
			Currency string                `json:"currency"`
			Points   []report.HistoryPoint `json:"points"`
		} `json:"history"`
	}
	if err := json.Unmarshal(data, &detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.History) != 1 || detail.History[0].Currency != "USD" || len(detail.History[0].Points) != 2 {
		t.Fatalf("one native currency and two real endpoints expected: %s", data)
	}
	points := detail.History[0].Points
	if points[0].Date != "2000-01-31" || points[1].Date != "2000-02-29" || points[0].Valuation.CompleteTotal.Exact != "25" || points[1].Valuation.CompleteTotal.Exact != "25" {
		t.Fatal("history must preserve opening inventory and quiet-month carry")
	}
}

func TestAccountDetailHistoryUsesEachCutoffQuoteAndKeepsGaps(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/history.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "account", Account: "Assets:Stock", Valuation: "market-value", Currency: "USD", DateTo: "2000-03-31"})
	if err != nil {
		t.Fatal(err)
	}
	points := view.Account.History[0].Points
	if len(points) != 3 || points[0].Date != "2000-01-31" || points[0].Value != nil || points[0].Valuation.CompleteTotal != nil {
		t.Fatal("January must remain a missing valuation, not use February quote")
	}
	if points[1].Valuation.CompleteTotal.Exact != "200" || points[1].Valuation.Quotes[0].Date != "2000-02-15" || points[1].Segment == points[0].Segment {
		t.Fatal("quiet February must use its own quote and start a new segment")
	}
	if points[2].Valuation.CompleteTotal.Exact != "0" || len(points[2].Valuation.Quotes) != 0 {
		t.Fatal("fully sold March inventory needs no quote and is a valid zero")
	}
	if view.Account.SubtreeAmounts["STK"].Exact != "0" || view.Account.SubtreeValuation.CompleteTotal.Exact != "0" {
		t.Fatal("current zero inventory must retain native commodity identity")
	}
}
