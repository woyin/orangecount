// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
)

func TestNativePricesPreservesSourceHistoryAndUsesCutoffSelection(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/prices.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var request desktop.Request
	if err := json.Unmarshal([]byte(`{"view":"prices","price_section":"history","price_base":"STK","price_quote":"USD","date_from":"2000-01-05","date_to":"2000-01-08"}`), &request); err != nil {
		t.Fatal(err)
	}
	view, err := s.Read(request)
	if err != nil {
		t.Fatalf("price view must be a real native route: %v", err)
	}
	data, _ := json.Marshal(view)
	var response struct {
		Prices struct {
			Records  []any `json:"records"`
			Selected *struct {
				Rate    report.PresentedDecimal `json:"rate"`
				Date    string                  `json:"date"`
				AgeDays int                     `json:"age_days"`
				Inverse bool                    `json:"inverse"`
				Source  report.ProjectionSource `json:"source"`
			} `json:"selected"`
		} `json:"prices"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	selected := response.Prices.Selected
	if selected == nil || selected.Rate.Exact != "11" || selected.Date != "2000-01-04" || selected.AgeDays != 4 || selected.Inverse || selected.Source.Line != 18 || len(response.Prices.Records) != 0 {
		t.Fatal("date_from limits actual history, not as-of selection; direct last source price beats a newer inverse and future price")
	}
}

func TestNativePricesDistinguishesInverseZeroAndMissingQuotes(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/prices.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, want := range []struct {
		base, quote, cutoff, exact, reason string
		inverse                            bool
		sourceBase, sourceQuote            string
		line, age                          int
	}{
		{"STK", "EUR", "2000-01-08", "0.5", "", true, "EUR", "STK", 20, 2},
		{"ZERO", "USD", "2000-01-08", "0", "", false, "ZERO", "USD", 22, 2},
		{"STK", "GBP", "2000-01-08", "", "missing-local-quote", false, "", "", 0, 0},
		{"STK", "USD", "2000-01-03", "", "missing-local-quote", false, "", "", 0, 0},
		{"STK", "AUD", "2000-01-08", "", "missing-local-quote", false, "", "", 0, 0},
		{"UNQUOTED", "USD", "2000-01-08", "", "missing-local-quote", false, "", "", 0, 0},
		{"STK", "", "2000-01-08", "", "unspecified-pair", false, "", "", 0, 0},
	} {
		t.Run(want.base+"/"+want.quote+"/"+want.cutoff, func(t *testing.T) {
			view, err := s.Read(desktop.Request{View: "prices", PriceBase: want.base, PriceQuote: want.quote, DateTo: want.cutoff, Currency: "EUR"})
			if err != nil {
				t.Fatal(err)
			}
			p := view.Prices
			if p == nil || p.Commodity == nil || p.Commodity.Currency != want.base || p.SelectedReason != want.reason {
				t.Fatal("selection context or reason missing")
			}
			if want.exact == "" {
				if p.Selected != nil {
					t.Fatal("missing price may not become zero or multi-hop conversion")
				}
				return
			}
			q := p.Selected
			if q == nil || q.Rate.Exact != want.exact || q.Inverse != want.inverse || q.SourceBase != want.sourceBase || q.SourceQuote != want.sourceQuote || q.Source.Line != want.line || q.AgeDays != want.age {
				t.Fatalf("incorrect independent selected quote: %+v", q)
			}
			if want.inverse && len(p.Records) != 0 {
				t.Fatal("inverse selection may not manufacture source history")
			}
			if want.base == "ZERO" && (len(p.Records) != 1 || p.Records[0].Value == nil || *p.Records[0].Value != 0) {
				t.Fatal("direct zero is a real history point")
			}
		})
	}
}

func TestNativePricesPagesSourceMultiplicityAndRejectsStaleSnapshots(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"prices.bean", "prices-included.bean"} {
		data, err := os.ReadFile("../../testdata/fixtures/native/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "prices.bean")
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3, Watch: snapshot.WatchOptions{PollInterval: time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	r := desktop.Request{View: "prices", PriceBase: "STK", PriceQuote: "USD", DateFrom: "2000-01-04", DateTo: "2000-01-04", Limit: 1}
	first, err := s.Read(r)
	if err != nil {
		t.Fatal(err)
	}
	if first.Prices.RecordsPage.TotalCount != 3 || !first.Prices.RecordsPage.HasMore || first.Prices.Records[0].Rate.Exact != "10" || !strings.HasSuffix(first.Prices.Records[0].Source.Path, "prices-included.bean") {
		t.Fatal("first included source price must survive pagination")
	}
	ids := map[string]bool{first.Prices.Records[0].ID: true}
	r.SnapshotID = first.State.SnapshotID
	for offset := 1; offset < 3; offset++ {
		r.Offset = offset
		page, err := s.Read(r)
		if err != nil {
			t.Fatal(err)
		}
		p := page.Prices
		if len(p.Records) != 1 || p.Records[0].Rate.Exact != "11" || ids[p.Records[0].ID] || p.Selected.Rate.Exact != "11" || p.RecordsPage.Offset != offset {
			t.Fatal("duplicate prices must retain distinct identity and independent full selection")
		}
		ids[p.Records[0].ID] = true
	}
	r.Offset = 0
	r.Sort = "descending"
	desc, err := s.Read(r)
	if err != nil {
		t.Fatal(err)
	}
	if desc.Prices.Records[0].Source.Line != 18 {
		t.Fatal("same-day descending order must reverse source order")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte("\n2000-01-21 price STK 100 USD\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(r); !errors.Is(err, desktop.ErrStale) {
		t.Fatal("price pages may not cross snapshots")
	}
	r.SnapshotID = ""
	current, err := s.Read(r)
	if err != nil {
		t.Fatal(err)
	}
	if current.Prices.Records[0].ID == desc.Prices.Records[0].ID {
		t.Fatal("new snapshot must produce new source record IDs")
	}
}

func TestNativePricesSelectedCommodityIsCompleteBeyondCatalogPreview(t *testing.T) {
	var input strings.Builder
	for i := 0; i < 105; i++ {
		fmt.Fprintf(&input, "2000-01-01 commodity C%03d\n", i)
	}
	for i := 0; i < 505; i++ {
		fmt.Fprintf(&input, "2000-01-02 price STK %d USD\n", i)
	}
	path := filepath.Join(t.TempDir(), "main.bean")
	if err := os.WriteFile(path, []byte(input.String()), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "prices", PriceBase: "STK", PriceQuote: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	p := view.Prices
	if len(p.Commodities) != 100 || p.CommoditiesPage.TotalCount != 107 || !p.CommoditiesPage.Truncated || p.Commodity == nil || p.Commodity.Currency != "STK" || p.Commodity.PriceCount != 505 || len(p.Commodity.QuoteCurrencies) != 1 || p.Commodity.QuoteCurrencies[0] != "USD" || p.RecordsPage.TotalCount != 505 || len(p.Records) != 100 || p.Selected.Rate.Exact != "504" {
		t.Fatal("selected commodity and selected quote must remain complete beyond both previews")
	}
	last, err := s.Read(desktop.Request{View: "prices", PriceBase: "STK", PriceQuote: "USD", SnapshotID: view.State.SnapshotID, Offset: 500, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(last.Prices.Records) != 5 || last.Prices.RecordsPage.HasMore || last.Prices.Selected.Rate.Exact != "504" {
		t.Fatal("history tail is independently readable")
	}
}

func TestNativePricesEmptyAndUnplottableValuesPreserveExactMeaning(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty.bean")
	if err := os.WriteFile(empty, []byte("; No declarations, prices or account use.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(empty, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "prices"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Prices.Commodities == nil || view.Prices.Records == nil || len(view.Prices.Commodities) != 0 || view.Prices.Selected != nil || view.Prices.SelectedReason != "unspecified-pair" {
		t.Fatal("empty catalog requires arrays and explicit unselected state")
	}
	path := filepath.Join(t.TempDir(), "main.bean")
	huge := "1" + strings.Repeat("0", 400)
	if err := os.WriteFile(path, []byte("2000-01-01 price STK "+huge+" USD\n2000-01-02 price STK 2 USD\n"), 0600); err != nil {
		t.Fatal(err)
	}
	large, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer large.Close()
	projected, err := large.Read(desktop.Request{View: "prices", PriceBase: "STK", PriceQuote: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	points := projected.Prices.Records
	if len(points) != 2 || points[0].Rate.Exact != huge || points[0].Value != nil || points[1].Segment != 1 || points[1].Value == nil || *points[1].Value != 2 {
		t.Fatal("nonfinite chart coordinates must retain exact amounts and break the segment")
	}
}

func TestNativePricesRejectsOversizedCompleteResponse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	if err := os.WriteFile(path, []byte("2000-01-01 commodity "+strings.Repeat("X", 17<<20)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state, err := s.Poll()
	if err != nil || !state.Valid {
		t.Fatal("budget input must be valid")
	}
	if _, err := s.Read(desktop.Request{View: "prices"}); !errors.Is(err, desktop.ErrResultBudget) {
		t.Fatal("prices must enforce the full bridge response budget")
	}
}

func TestNativeChartsRetainExactUnplottableHistoryWithoutJoiningAcrossGap(t *testing.T) {
	huge := "1" + strings.Repeat("0", 400)
	input := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"finite\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n2000-02-02 * \"huge\"\n  Assets:Cash " + huge + " USD\n  Equity:Opening -" + huge + " USD\n2000-03-02 * \"finite again\"\n  Assets:Cash -" + huge + " USD\n  Equity:Opening " + huge + " USD\n"
	path := filepath.Join(t.TempDir(), "main.bean")
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, route := range []string{"dashboard", "account"} {
		r := desktop.Request{View: route, Currency: "native"}
		if route == "account" {
			r.Account = "Assets:Cash"
		}
		view, err := s.Read(r)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := json.Marshal(view); err != nil {
			t.Fatal("nullable coordinates must serialize without Infinity")
		}
		var points []report.HistoryPoint
		if route == "dashboard" {
			points = view.Trends.CurrencySeries[0].NetWorthPoints
			if len(view.Trends.NetWorthPoints) != 0 {
				t.Fatal("legacy nonnullable flat chart must not omit the gap and connect finite ends")
			}
		} else {
			points = view.Account.History[0].Points
		}
		if len(points) != 3 || points[0].Value == nil || points[1].Value != nil || points[1].Valuation.CompleteTotal == nil || points[2].Value == nil || *points[2].Value != 1 || points[2].Segment == points[0].Segment {
			t.Fatal("financially complete values may be unplottable, and later coordinates must start a new segment")
		}
	}
}

func TestNativePricesRejectsUnrelatedValuationAndRequestParameters(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/prices.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, request := range []desktop.Request{
		{View: "prices", Valuation: "units"},
		{View: "prices", Valuation: "market-value", Currency: "USD"},
		{View: "prices", PriceSection: "unknown"},
		{View: "prices", PriceSection: "history"},
		{View: "prices", PriceSection: "commodities", PriceBase: "STK"},
		{View: "prices", PriceQuote: "USD"},
		{View: "prices", Account: "Assets:Stock"},
		{View: "prices", GroupBy: "account"},
		{View: "prices", Text: "STK"},
		{View: "journal", PriceBase: "STK"},
		{View: "prices", Offset: 1},
		{View: "prices", Limit: 501},
	} {
		if _, err := s.Read(request); !errors.Is(err, desktop.ErrRequest) {
			t.Fatalf("unrelated or invalid price parameters must be rejected: %+v", request)
		}
	}
	if _, err := s.Read(desktop.Request{View: "prices", PriceBase: "MISSING", PriceQuote: "USD"}); !errors.Is(err, report.ErrCommodityUnknown) {
		t.Fatal("unknown commodity must be a target error")
	}
}

func TestNativePricesCatalogRetainsClearedUnquotedAndDeclaredCommodities(t *testing.T) {
	s, err := desktop.OpenWithOptions("../../testdata/fixtures/native/prices.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "prices", DateTo: "2000-01-08"})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(view)
	var response struct {
		Prices struct {
			Section     string `json:"section"`
			Commodities []struct {
				Currency        string   `json:"currency"`
				Accounts        []string `json:"accounts"`
				QuoteCurrencies []string `json:"quote_currencies"`
				PriceCount      int      `json:"price_count"`
				Declarations    []struct {
					Source report.ProjectionSource `json:"source"`
				} `json:"declarations"`
			} `json:"commodities"`
		} `json:"prices"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if response.Prices.Section != "commodities" || len(response.Prices.Commodities) != 10 {
		t.Fatal("price catalog must include declarations, historical account use and both ends of local pairs")
	}
	checked := 0
	for _, item := range response.Prices.Commodities {
		switch item.Currency {
		case "STK":
			if len(item.Accounts) != 1 || item.Accounts[0] != "Assets:Stock" || item.PriceCount != 8 || len(item.Declarations) != 1 || item.Declarations[0].Source.Line != 3 || len(item.QuoteCurrencies) != 4 {
				t.Fatal("cleared stock must retain source price multiplicity, account use and its declaration")
			}
			checked++
		case "UNQUOTED":
			if item.PriceCount != 0 || len(item.Accounts) != 1 || len(item.QuoteCurrencies) != 0 {
				t.Fatal("a used unquoted commodity may not disappear")
			}
			checked++
		case "DECLAREDONLY":
			if len(item.Declarations) != 1 || len(item.Accounts) != 0 || item.PriceCount != 0 {
				t.Fatal("declaration-only commodity may not disappear")
			}
			checked++
		case "JPY":
			if item.PriceCount != 1 || len(item.QuoteCurrencies) != 1 || item.QuoteCurrencies[0] != "STK" {
				t.Fatal("quote-only currency retains its real local pair")
			}
			checked++
		}
	}
	if checked != 4 {
		t.Fatal("catalog omitted a required commodity class")
	}
}
