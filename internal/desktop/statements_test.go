// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop_test

import (
	"encoding/json"
	"orangecount/internal/desktop"
	"orangecount/internal/snapshot"
	"os"
	"path/filepath"
	"testing"
)

func TestStatementsUseAsOfBalancesAndInclusivePeriodFlows(t *testing.T) {
	s, err := desktop.Open("../../testdata/fixtures/native/cashflow.bean", snapshot.WatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	bs, err := s.Read(desktop.Request{View: "balance_sheet", DateFrom: "2026-01-02", DateTo: "2026-01-02"})
	if err != nil {
		t.Fatal(err)
	}
	if bs.Tree.RootNodes[0].SubtreeBalances["USD"] != "1000" {
		t.Fatalf("as-of Jan 2 must exclude Jan 3 spending: %+v", bs.Tree)
	}
	is, err := s.Read(desktop.Request{View: "income_statement", DateFrom: "2026-01-03", DateTo: "2026-01-03"})
	if err != nil {
		t.Fatal(err)
	}
	if is.Tree.RootNodes[0].SubtreeBalances["USD"] != "" || is.Tree.RootNodes[1].SubtreeBalances["USD"] != "300" {
		t.Fatalf("Jan 3 must include only spending: %+v", is.Tree)
	}
	view, err := s.Read(desktop.Request{View: "dashboard", DateFrom: "2026-01-03", DateTo: "2026-01-03"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Trends.NetWorthPoints[0].Amount.Exact != "700" || view.Trends.MonthlyCashFlows[0].NetAmount.Exact != "-300" {
		t.Fatal("net worth carries prior balance; selected-period flow does not")
	}
	quiet, err := s.Read(desktop.Request{View: "dashboard", DateFrom: "2026-02-01", DateTo: "2026-02-28"})
	if err != nil {
		t.Fatal(err)
	}
	if len(quiet.Trends.NetWorthPoints) != 1 || quiet.Trends.NetWorthPoints[0].Amount.Exact != "700" {
		t.Fatal("quiet periods must carry the prior net worth without inventing cash flow")
	}
	if len(quiet.Trends.MonthlyCashFlows) != 0 {
		t.Fatal("quiet periods have no cash-flow records")
	}
	if _, err := s.Read(desktop.Request{View: "dashboard", DateFrom: "2030-01-01"}); err != desktop.ErrRequest {
		t.Fatal("a start date after the ledger-anchored end must reject the inverted range")
	}
}

func TestStatementDefaultsToCompleteNativeBalances(t *testing.T) {
	s, err := desktop.Open("../../testdata/fixtures/v3-parity/options.bean", snapshot.WatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "balance_sheet"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Tree.Currency != "" || len(view.Tree.RootNodes) == 0 || view.Tree.RootNodes[0].SubtreeBalances["EUR"] != "2" {
		t.Fatalf("unconfigured tables must retain all native balances: %+v", view.Tree)
	}
	native, err := s.Read(desktop.Request{View: "balance_sheet", Currency: "native"})
	if err != nil {
		t.Fatal(err)
	}
	if len(native.Tree.RootNodes) == 0 || native.Tree.RootNodes[0].SubtreeBalances["USD"] != "1" || native.Tree.RootNodes[0].SubtreeBalances["EUR"] != "2" {
		t.Fatalf("native selection must retain both commodities: %+v", native.Tree)
	}
	if native.Tree.DateTo != "2000-01-03" {
		t.Fatalf("default as-of must use the latest ledger date, including its balance assertion; got %s", native.Tree.DateTo)
	}
}

func TestNetWorthDoesNotRequireConversionOfEquityOnlyTransfers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	data := `option "operating_currency" "USD"
2026-01-01 open Assets:Cash USD
2026-01-01 open Income:Salary USD
2026-01-01 open Equity:One EUR
2026-01-01 open Equity:Two EUR
2026-01-02 * "Salary"
  Assets:Cash 100 USD
  Income:Salary -100 USD
2026-01-03 * "Equity transfer"
  Equity:One 1 EUR
  Equity:Two -1 EUR
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.Open(path, snapshot.WatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "dashboard"})
	if err != nil {
		t.Fatal(err)
	}
	if !view.State.Valid || view.Trends.NetWorthAvailability != "priced" || len(view.Trends.NetWorthPoints) != 1 || view.Trends.NetWorthPoints[0].Amount.Exact != "100" {
		t.Fatalf("equity-only EUR activity must not hide USD net worth: %+v", view.Trends)
	}
	tree, err := s.Read(desktop.Request{View: "balance_sheet"})
	if err != nil {
		t.Fatal(err)
	}
	if tree.Tree.Currency != "" || len(tree.Tree.RootNodes) == 0 {
		t.Fatal("default native table must remain readable even with unpriced Equity")
	}
}

func TestConvertedStatementPreservesNativeTreeAndSeparatesPartialValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	data := `option "operating_currency" "USD"
2000-01-01 open Assets:Cash USD,EUR
2000-01-01 open Equity:Opening USD,EUR
2000-01-02 * "mixed cash"
  Assets:Cash 100 USD
  Assets:Cash 20 EUR
  Equity:Opening -100 USD
  Equity:Opening -20 EUR
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.Open(path, snapshot.WatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "balance_sheet", Currency: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Tree.RootNodes) == 0 {
		t.Fatal("missing quote must not erase the native tree")
	}
	assets := view.Tree.RootNodes[0]
	if assets.SubtreeBalances["USD"] != "100" || assets.SubtreeBalances["EUR"] != "20" {
		t.Fatal("conversion must not replace or omit native balances")
	}
	encoded, _ := json.Marshal(assets)
	var node map[string]any
	if err := json.Unmarshal(encoded, &node); err != nil {
		t.Fatal(err)
	}
	conversion, ok := node["subtree_valuation"].(map[string]any)
	if !ok || conversion["completeness"] != "partial" || conversion["complete_total"] != nil {
		t.Fatalf("incomplete valuation must have no complete total: %s", encoded)
	}
	portion, ok := conversion["valued_portion"].(map[string]any)
	if !ok || portion["exact"] != "100" {
		t.Fatalf("only the USD portion is valued: %s", encoded)
	}
}

func TestStatementValuationUsesOnlyMatchingAsOfQuotesAndDistinguishesZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	data := `2000-01-01 open Assets:Cash USD,EUR,GBP
2000-01-01 open Assets:Zero USD
2000-01-01 open Assets:Empty USD
2000-01-01 open Assets:Foreign EUR
2000-01-01 open Equity:Opening USD,EUR,GBP
2000-01-02 * "balances"
  Assets:Cash 100 USD
  Assets:Cash 20 EUR
  Assets:Cash 10 GBP
  Assets:Foreign 4 EUR
  Assets:Zero 1 USD
  Assets:Zero -1 USD
  Equity:Opening -100 USD
  Equity:Opening -24 EUR
  Equity:Opening -10 GBP
2000-01-02 price EUR 2 USD
2000-01-02 price USD 0.5 GBP
2000-01-03 price EUR 9 JPY
2000-01-04 price EUR 10 USD
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.Open(path, snapshot.WatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "balance_sheet", Currency: "USD", DateTo: "2000-01-03"})
	if err != nil {
		t.Fatal(err)
	}
	assets := view.Tree.RootNodes[0]
	total := assets.SubtreeValuation
	if total.Completeness != "complete" || total.CompleteTotal == nil || total.CompleteTotal.Exact != "168" || view.Tree.Availability != "priced" {
		t.Fatalf("100 USD + 24 EUR at 2 + 10 GBP at inverse 2 = 168, no future or JPY quote: %+v %+v", total, view.Tree)
	}
	for _, quote := range total.Quotes {
		if quote.Date != "2000-01-02" {
			t.Fatal("future quote used")
		}
	}
	for _, child := range assets.Children {
		if child.Name == "Zero" && (child.DirectValuation.Completeness != "complete" || child.DirectValuation.CompleteTotal == nil || child.DirectValuation.CompleteTotal.Exact != "0") {
			t.Fatal("legitimate zero was confused with empty/unavailable")
		}
		if child.Name == "Empty" && (child.DirectValuation.Completeness != "empty" || child.DirectValuation.CompleteTotal != nil) {
			t.Fatal("empty account acquired a manufactured total")
		}
	}
	early, err := s.Read(desktop.Request{View: "balance_sheet", Currency: "USD", DateTo: "2000-01-02"})
	if err != nil {
		t.Fatal(err)
	}
	if early.Tree.RootNodes[0].SubtreeValuation.CompleteTotal.Exact != "168" {
		t.Fatal("matching pair must be used at the inclusive cutoff")
	}
	unavailable, err := s.Read(desktop.Request{View: "balance_sheet", Currency: "CHF", DateTo: "2000-01-03"})
	if err != nil {
		t.Fatal(err)
	}
	if got := unavailable.Tree.RootNodes[0].SubtreeValuation; got.Completeness != "unavailable" || got.CompleteTotal != nil || got.ValuedPortion != nil {
		t.Fatal("no coverage must not manufacture a zero-valued portion")
	}
}

func TestDashboardNativeSeriesNeverAddsDifferentCurrencies(t *testing.T) {
	s, err := desktop.Open("../../testdata/fixtures/v3-parity/options.bean", snapshot.WatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "dashboard", Window: "all"})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(view.Trends)
	var trends map[string]any
	if err := json.Unmarshal(encoded, &trends); err != nil {
		t.Fatal(err)
	}
	series, ok := trends["currency_series"].([]any)
	if !ok || len(series) != 2 {
		t.Fatalf("native dashboard must expose separate USD and EUR series: %s", encoded)
	}
	wanted := map[string]string{"USD": "1", "EUR": "2"}
	for _, raw := range series {
		item := raw.(map[string]any)
		points := item["net_worth_points"].([]any)
		if len(points) != 1 {
			t.Fatal("expected one point per currency")
		}
		valuation := points[0].(map[string]any)["valuation"].(map[string]any)
		total := valuation["complete_total"].(map[string]any)
		if total["exact"] != wanted[item["currency"].(string)] {
			t.Fatalf("different currencies were added: %s", encoded)
		}
	}
}

func TestHistoryCarriesInventoryAndKeepsMissingQuotePoints(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	data := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Stock STK "FIFO"
2000-01-01 open Equity:Opening USD
2000-01-01 open Income:Gains USD
2000-01-02 * "opening"
  Assets:Cash 1000 USD
  Equity:Opening -1000 USD
2000-01-03 * "purchase"
  Assets:Stock 10 STK {10 USD}
  Assets:Cash -100 USD
2000-02-15 price STK 20 USD
2000-03-04 * "full liquidation"
  Assets:Stock -10 STK {} @ 20 USD
  Assets:Cash 200 USD
  Income:Gains
2000-03-30 price STK 99 USD
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.Open(path, snapshot.WatchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "dashboard", Currency: "USD", Valuation: "market-value", DateTo: "2000-03-31"})
	if err != nil {
		t.Fatal(err)
	}
	series := view.Trends.CurrencySeries
	if len(series) != 1 || len(series[0].NetWorthPoints) != 3 {
		t.Fatalf("three monthly inventory states required: %+v", series)
	}
	points := series[0].NetWorthPoints
	if points[0].Date != "2000-01-31" || points[0].Valuation.CompleteTotal != nil || points[0].Value != nil || points[0].Valuation.ValuedPortion.Exact != "900" {
		t.Fatal("January must remain a gap with only 900 USD valued, not a future quote or current empty stock inventory")
	}
	if points[1].Valuation.CompleteTotal.Exact != "1100" || points[1].Valuation.Quotes[0].Date != "2000-02-15" {
		t.Fatal("quiet February must revalue the held 10 STK using February's quote")
	}
	if points[2].Valuation.CompleteTotal.Exact != "1100" || len(points[2].Valuation.Quotes) != 0 {
		t.Fatal("liquidated inventory must remain zero regardless of March's quote")
	}
	if points[1].Segment == points[0].Segment {
		t.Fatal("a missing valuation must start a new line segment")
	}
	if series[0].NetWorthAvailability != "partial" || len(view.Trends.NetWorthPoints) != 0 {
		t.Fatal("legacy flat points must not connect across a missing valuation")
	}
}
