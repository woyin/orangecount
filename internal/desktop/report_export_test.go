// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"orangecount/internal/analysis"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"os"
	"path/filepath"
	"testing"
)

func TestReportCaptureCancellationIsRememberedBeforeCaptureStarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	text := "2024-01-01 open Assets:Cash USD\n2024-01-01 open Equity:Opening USD\n2024-01-02 * \"public\"\n  Assets:Cash 12 USD\n  Equity:Opening -12 USD\n"
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	view, err := session.Read(desktop.Request{View: "balance_sheet", Currency: "USD"})
	if err != nil || view.ReportExport == nil {
		t.Fatalf("create report projection: %+v, %v", view.ReportExport, err)
	}
	request := desktop.ReportExportRequest{
		CaptureID: "cancel-before-start", ProjectionID: view.ReportExport.ProjectionID,
		SnapshotID: view.State.SnapshotID, Format: "analysis", CaptureScope: view.ReportExport.CaptureScope,
	}
	if err := session.CancelReportExport(request.CaptureID); err != nil {
		t.Fatal(err)
	}
	if _, err := session.CaptureReportExport(request); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-cancelled report capture = %v, want context.Canceled", err)
	}
}

func TestTreeCaptureHasExactValuesAndFixedIdentity(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.bean")
	text := "2024-01-01 open Assets:Cash USD\n2024-01-01 open Equity:Opening USD\n2024-01-02 * \"public\"\n  Assets:Cash  12.34 USD\n  Equity:Opening\n"
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v, err := s.Read(desktop.Request{View: "balance_sheet", Currency: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	if v.ReportExport == nil {
		t.Fatal("missing capture identity")
	}
	req := desktop.ReportExportRequest{ProjectionID: v.ReportExport.ProjectionID, SnapshotID: v.State.SnapshotID, Format: "analysis", CaptureScope: "complete-projection"}
	capture, err := s.CaptureReportExport(req)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := analysis.ReadReport(bytes.NewReader(capture.Content), int64(len(capture.Content)))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Context.SourceRecordCount != 4 || doc.Context.ReturnedRows != 4 {
		t.Fatalf("expected Assets/Cash and Equity/Opening rows, got %d", len(doc.Rows))
	}
	if bytes.Contains(capture.Content, []byte(path)) {
		t.Fatal("absolute source path in archive")
	}
	found := false
	for _, row := range doc.Rows {
		if row[0].Value == "Assets:Cash" {
			raw, _ := json.Marshal(row[4].Value)
			if !bytes.Contains(raw, []byte(`"exact":"12.34"`)) {
				t.Fatalf("incorrect exact inventory: %s", raw)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("missing Cash")
	}
	req.Format = "csv"
	csv, err := s.CaptureReportExport(req)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(csv.Content, []byte("Assets:Cash")) {
		t.Fatal("missing CSV row")
	}
	bad := req
	bad.ProjectionID = "unknown"
	if _, err = s.CaptureReportExport(bad); !errors.Is(err, desktop.ErrExpiredCapture) {
		t.Fatalf("unknown capture: %v", err)
	}
	for i := 1; i <= 17; i++ {
		_, err = s.Read(desktop.Request{View: "balance_sheet", DateTo: fmt.Sprintf("2024-02-%02d", i), Currency: "USD"})
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.CaptureReportExport(req); !errors.Is(err, desktop.ErrExpiredCapture) {
		t.Fatalf("capture not evicted: %v", err)
	}
}

func TestReportProjectionSectionsRoundTrip(t *testing.T) {
	cases := []struct {
		file    string
		request desktop.Request
	}{
		{"valuation.bean", desktop.Request{View: "balance_sheet", Currency: "USD", Valuation: "market-value"}},
		{"cashflow.bean", desktop.Request{View: "income_statement"}},
		{"history.bean", desktop.Request{View: "dashboard"}},
		{"account-detail.bean", desktop.Request{View: "account", Account: "Assets:Test"}},
		{"cashflow.bean", desktop.Request{View: "journal", Offset: 1, Limit: 1}},
		{"prices.bean", desktop.Request{View: "prices", Limit: 1}},
		{"prices.bean", desktop.Request{View: "prices", PriceBase: "STK", PriceQuote: "USD", Offset: 1, Limit: 1}},
		{"holdings-trades.bean", desktop.Request{View: "holdings", HoldingSection: "groups", Limit: 1}},
		{"holdings-trades.bean", desktop.Request{View: "holdings", HoldingSection: "events", Limit: 1}},
		{"average-history.bean", desktop.Request{View: "holdings", HoldingSection: "history", Limit: 1}},
		{"holdings-unknowns.bean", desktop.Request{View: "holdings"}},
		{"empty.bean", desktop.Request{View: "journal"}},
	}
	for _, tc := range cases {
		t.Run(tc.file+"/"+tc.request.View+"/"+tc.request.HoldingSection+"/"+tc.request.PriceBase, func(t *testing.T) {
			path, err := filepath.Abs("../../testdata/fixtures/native/" + tc.file)
			if err != nil {
				t.Fatal(err)
			}
			s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			state, _ := s.Poll()
			request := tc.request
			request.SnapshotID = state.SnapshotID
			v, err := s.Read(request)
			if err != nil {
				t.Fatal(err)
			}
			if v.ReportExport == nil {
				t.Fatal("missing export")
			}
			capture, err := s.CaptureReportExport(desktop.ReportExportRequest{SnapshotID: v.State.SnapshotID, ProjectionID: v.ReportExport.ProjectionID, Format: "analysis", CaptureScope: v.ReportExport.CaptureScope})
			if err != nil {
				t.Fatal(err)
			}
			d, err := analysis.ReadReport(bytes.NewReader(capture.Content), int64(len(capture.Content)))
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(d)
			if bytes.Contains(raw, []byte(path)) || bytes.Contains(raw, []byte(`"path":`)) {
				t.Fatal("filesystem source path leaked into capture")
			}
			if d.Context.ReturnedRows != len(d.Rows) {
				t.Fatal("incorrect row count")
			}
			if v.Journal != nil && v.Account == nil {
				if d.Context.SourceRecordCount != len(v.Journal.Transactions) || d.Schema.RowUnit != "posting" {
					t.Fatal("lost transaction/posting counts")
				}
			}
			if v.Holdings != nil && len(v.Holdings.Groups) > 0 {
				g := v.Holdings.Groups[0]
				view, err := s.Read(desktop.Request{View: "holdings", HoldingSection: "lots", GroupID: g.ID, Limit: 1, SnapshotID: v.State.SnapshotID})
				if err != nil {
					t.Fatal(err)
				}
				if view.ReportExport == nil {
					t.Fatal("missing lot export")
				}
				cap, err := s.CaptureReportExport(desktop.ReportExportRequest{SnapshotID: view.State.SnapshotID, ProjectionID: view.ReportExport.ProjectionID, Format: "analysis", CaptureScope: view.ReportExport.CaptureScope})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = analysis.ReadReport(bytes.NewReader(cap.Content), int64(len(cap.Content))); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestStatementReportAnalysisPreservesCurrencyAndValuationRequest(t *testing.T) {
	path, err := filepath.Abs("../../testdata/fixtures/native/valuation.bean")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, currency, valuation, wantValuation string
	}{
		{name: "native currencies at default cost", wantValuation: "at-cost"},
		{name: "USD market value", currency: "USD", valuation: "market-value", wantValuation: "market-value"},
	}
	profileProjectionIDs := map[ledger.SemanticsProfile]map[string]string{}
	for _, profile := range []ledger.SemanticsProfile{ledger.Legacy, ledger.V3} {
		t.Run(string(profile), func(t *testing.T) {
			s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: profile})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			projectionIDs := map[string]bool{}
			profileProjectionIDs[profile] = map[string]string{}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					view, err := s.Read(desktop.Request{View: "balance_sheet", Currency: tc.currency, Valuation: tc.valuation})
					if err != nil {
						t.Fatal(err)
					}
					if view.ReportExport == nil {
						t.Fatal("statement omitted report export identity")
					}
					if view.ReportExport.SemanticsProfile != string(profile) || view.ReportExport.Currency != tc.currency || view.ReportExport.Valuation != tc.wantValuation {
						t.Errorf("report identity profile/currency/valuation = %q/%q/%q, want %q/%q/%q", view.ReportExport.SemanticsProfile, view.ReportExport.Currency, view.ReportExport.Valuation, profile, tc.currency, tc.wantValuation)
					}
					projectionIDs[view.ReportExport.ProjectionID] = true
					profileProjectionIDs[profile][tc.currency+"\x00"+tc.wantValuation] = view.ReportExport.ProjectionID
					capture, err := s.CaptureReportExport(desktop.ReportExportRequest{
						ProjectionID: view.ReportExport.ProjectionID, SnapshotID: view.State.SnapshotID,
						Format: "analysis", CaptureScope: view.ReportExport.CaptureScope,
					})
					if err != nil {
						t.Fatal(err)
					}
					doc, err := analysis.ReadReport(bytes.NewReader(capture.Content), int64(len(capture.Content)))
					if err != nil {
						t.Fatal(err)
					}
					if doc.Context.SemanticsProfile != string(profile) {
						t.Errorf("analysis profile = %q, want %q", doc.Context.SemanticsProfile, profile)
					}
					if got := doc.Context.NormalizedRequest["currency"]; got != tc.currency {
						t.Errorf("analysis currency = %q, want %q", got, tc.currency)
					}
					if got := doc.Context.NormalizedRequest["valuation"]; got != tc.wantValuation {
						t.Errorf("analysis valuation = %q, want %q", got, tc.wantValuation)
					}
				})
			}
			if len(projectionIDs) != len(cases) {
				t.Fatalf("different currency/valuation requests shared a projection ID: %v", projectionIDs)
			}
		})
	}
	for _, tc := range cases {
		key := tc.currency + "\x00" + tc.wantValuation
		if profileProjectionIDs[ledger.Legacy][key] == profileProjectionIDs[ledger.V3][key] {
			t.Errorf("legacy and v3 %s reports share a projection ID %q", tc.name, profileProjectionIDs[ledger.V3][key])
		}
	}
}

func TestDashboardReportAnalysisNormalizesNativeCurrencyAndDefaultValuation(t *testing.T) {
	path, err := filepath.Abs("../../testdata/fixtures/native/history.bean")
	if err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "dashboard", Currency: "native", Locale: "zh-CN"})
	if err != nil {
		t.Fatal(err)
	}
	if view.ReportExport == nil {
		t.Fatal("dashboard omitted report export identity")
	}
	if view.ReportExport.Currency != "" || view.ReportExport.Valuation != "at-cost" {
		t.Fatalf("dashboard report identity currency/valuation = %q/%q, want native currencies at at-cost", view.ReportExport.Currency, view.ReportExport.Valuation)
	}
	capture, err := s.CaptureReportExport(desktop.ReportExportRequest{
		ProjectionID: view.ReportExport.ProjectionID, SnapshotID: view.State.SnapshotID,
		Format: "analysis", CaptureScope: view.ReportExport.CaptureScope,
	})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := analysis.ReadReport(bytes.NewReader(capture.Content), int64(len(capture.Content)))
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.Context.NormalizedRequest["currency"]; got != "" {
		t.Errorf("dashboard analysis currency = %q, want native currencies", got)
	}
	if got := doc.Context.NormalizedRequest["valuation"]; got != "at-cost" {
		t.Errorf("dashboard analysis valuation = %q, want at-cost", got)
	}
}

func TestAccountAndHoldingsReportAnalysisUseEffectiveCurrencyAndValuation(t *testing.T) {
	cases := []struct {
		name, fixture, wantCurrency, wantValuation string
		request                                    desktop.Request
	}{
		{
			name: "account native at default cost", fixture: "account-detail.bean",
			request:       desktop.Request{View: "account", Account: "Assets:Test", Currency: "native"},
			wantValuation: "at-cost",
		},
		{
			name: "holdings native at default cost", fixture: "average-history.bean",
			request:       desktop.Request{View: "holdings", Account: "Assets:Stock", Currency: "native", HoldingSection: "groups"},
			wantValuation: "at-cost",
		},
		{
			name: "holdings USD market value", fixture: "average-history.bean",
			request:      desktop.Request{View: "holdings", Account: "Assets:Stock", Currency: "USD", Valuation: "market-value", HoldingSection: "groups"},
			wantCurrency: "USD", wantValuation: "market-value",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, err := filepath.Abs("../../testdata/fixtures/native/" + tc.fixture)
			if err != nil {
				t.Fatal(err)
			}
			s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			view, err := s.Read(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			if view.ReportExport == nil {
				t.Fatal("missing report export identity")
			}
			if view.ReportExport.Currency != tc.wantCurrency || view.ReportExport.Valuation != tc.wantValuation {
				t.Errorf("report identity currency/valuation = %q/%q, want %q/%q", view.ReportExport.Currency, view.ReportExport.Valuation, tc.wantCurrency, tc.wantValuation)
			}
			capture, err := s.CaptureReportExport(desktop.ReportExportRequest{
				ProjectionID: view.ReportExport.ProjectionID, SnapshotID: view.State.SnapshotID,
				Format: "analysis", CaptureScope: view.ReportExport.CaptureScope,
			})
			if err != nil {
				t.Fatal(err)
			}
			doc, err := analysis.ReadReport(bytes.NewReader(capture.Content), int64(len(capture.Content)))
			if err != nil {
				t.Fatal(err)
			}
			if got := doc.Context.NormalizedRequest["currency"]; got != tc.wantCurrency {
				t.Errorf("analysis currency = %q, want %q", got, tc.wantCurrency)
			}
			if got := doc.Context.NormalizedRequest["valuation"]; got != tc.wantValuation {
				t.Errorf("analysis valuation = %q, want %q", got, tc.wantValuation)
			}
		})
	}
}

func TestPriceReportIdentityRetainsSelectedQuotePair(t *testing.T) {
	path, err := filepath.Abs("../../testdata/fixtures/native/prices.bean")
	if err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{
		View: "prices", PriceSection: "history", PriceBase: "STK", PriceQuote: "USD",
		DateFrom: "2000-01-01", DateTo: "2000-01-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.ReportExport == nil {
		t.Fatal("price history omitted report export identity")
	}
	identityJSON, err := json.Marshal(view.ReportExport)
	if err != nil {
		t.Fatal(err)
	}
	var identity map[string]any
	if err := json.Unmarshal(identityJSON, &identity); err != nil {
		t.Fatal(err)
	}
	if identity["price_base"] != "STK" || identity["price_quote"] != "USD" {
		t.Fatalf("price report identity pair = %v/%v, want STK/USD", identity["price_base"], identity["price_quote"])
	}
	capture, err := s.CaptureReportExport(desktop.ReportExportRequest{
		ProjectionID: view.ReportExport.ProjectionID, SnapshotID: view.State.SnapshotID,
		Format: "analysis", CaptureScope: view.ReportExport.CaptureScope,
	})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := analysis.ReadReport(bytes.NewReader(capture.Content), int64(len(capture.Content)))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Context.NormalizedRequest["price_base"] != "STK" || doc.Context.NormalizedRequest["price_quote"] != "USD" {
		t.Fatalf("captured price request pair = %q/%q, want STK/USD", doc.Context.NormalizedRequest["price_base"], doc.Context.NormalizedRequest["price_quote"])
	}
}

func TestReportPagesRetainDifferentIdentities(t *testing.T) {
	path, err := filepath.Abs("../../testdata/fixtures/native/cashflow.bean")
	if err != nil {
		t.Fatal(err)
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state, _ := s.Poll()
	first, err := s.Read(desktop.Request{View: "journal", SnapshotID: state.SnapshotID, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Read(desktop.Request{View: "journal", SnapshotID: state.SnapshotID, Limit: 1, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	if first.ReportExport.ProjectionID == second.ReportExport.ProjectionID {
		t.Fatal("different pages share export identity")
	}
	for _, view := range []desktop.ViewResponse{first, second} {
		cap, err := s.CaptureReportExport(desktop.ReportExportRequest{SnapshotID: state.SnapshotID, ProjectionID: view.ReportExport.ProjectionID, CaptureScope: "current-page", Format: "analysis"})
		if err != nil {
			t.Fatal(err)
		}
		d, err := analysis.ReadReport(bytes.NewReader(cap.Content), int64(len(cap.Content)))
		if err != nil {
			t.Fatal(err)
		}
		if d.Context.Page.Offset != view.Journal.Offset || d.Rows[0][0].Value != view.Journal.Transactions[0].ID {
			t.Fatal("another page replaced capture contents")
		}
	}
}
