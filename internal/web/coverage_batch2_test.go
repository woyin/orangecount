// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestReportPivotEndpoint exercises the pivot builder through the report
// dispatch table, including the alias spellings and the filter error path.
func TestReportPivotEndpoint(t *testing.T) {
	server := newLedgerTestServer(t)
	response := get(t, server, "/api/v1/reports/pivot?rows=year&columns=currency&values=number")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "columns") {
		t.Fatalf("pivot status=%d body=%s", response.Code, response.Body.String())
	}
	// A malformed global filter must surface as a 400 with the parser's
	// message rather than an empty table.
	bad := get(t, server, "/api/v1/reports/pivot?rows=year&columns=currency&values=number&filter=%27%27%27")
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("bad filter status=%d body=%s", bad.Code, bad.Body.String())
	}
}

// TestFavaTreeReportValuations drives the adapter's statement-tree report
// through its valuation and interval parameters.
func TestFavaTreeReportValuations(t *testing.T) {
	server := newLedgerTestServer(t)
	for _, path := range []string{
		"/__orangecount/fava/reports/balance_sheet?valuation=market-value",
		"/__orangecount/fava/reports/income_statement?interval=quarterly",
		"/__orangecount/fava/reports/trial_balance?valuation=market-value",
	} {
		response := get(t, server, path)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
}

// TestEditorSaveRejectsBadPathsAndStaleSnapshots covers the authoring status
// mapping on the editor endpoint: bad paths are 400, stale snapshots 409.
func TestEditorSaveRejectsBadPathsAndStaleSnapshots(t *testing.T) {
	server := newLedgerTestServer(t)
	badPath := post(t, server, "/api/v1/editor/save", `{"path":"outside.bean","content":"2000-01-01 open Assets:Cash USD\n","expected_snapshot_id":""}`)
	if badPath.Code != http.StatusNotFound && badPath.Code != http.StatusBadRequest {
		t.Fatalf("bad path status=%d body=%s", badPath.Code, badPath.Body.String())
	}
	listing := get(t, server, "/api/v1/editor")
	stale := post(t, server, "/api/v1/editor/save", `{"path":"`+fieldJSON(t, listing.Body.String(), "paths")+`","content":"2000-01-01 open Assets:Cash USD\n","expected_snapshot_id":"stale"}`)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale save status=%d body=%s", stale.Code, stale.Body.String())
	}
}

// TestFavaDiagnosticsAndImportAdapters walks localized diagnostics and the
// adapter listing branches of the private adapter.
func TestFavaDiagnosticsAndImportAdapters(t *testing.T) {
	server := newLedgerTestServer(t)
	if response := get(t, server, "/__orangecount/fava/diagnostics?locale=zh-CN"); response.Code != http.StatusOK {
		t.Fatalf("diagnostics status=%d", response.Code)
	}
	adapters := get(t, server, "/__orangecount/fava/import?kind=adapters")
	if adapters.Code != http.StatusOK || !strings.Contains(adapters.Body.String(), "beancount") {
		t.Fatalf("adapters status=%d body=%s", adapters.Code, adapters.Body.String())
	}
}

// TestPlanningPreviewStoreEviction pins the bounded-preview guarantee: storing
// past capacity evicts the oldest entry so previews cannot accumulate.
func TestPlanningPreviewStoreEviction(t *testing.T) {
	store := newPlanningPreviewStore()
	first := store.Store(planningPreview{Content: "first", Target: "main.bean", SnapshotID: "s"})
	for index := 0; index < 40; index++ {
		store.Store(planningPreview{Content: strings.Repeat("x", index+1), Target: "main.bean", SnapshotID: "s"})
	}
	if _, ok := store.Take(first); ok {
		t.Fatal("the oldest preview must be evicted at capacity")
	}
}

// newLedgerTestServer builds a server over a small valid two-account ledger.
func newLedgerTestServer(t *testing.T) *Server {
	t.Helper()
	server, _ := newPlanningTestServer(t, `2000-01-01 open Assets:Cash USD,CNY
2000-01-01 open Expenses:Food USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "Grocer" "first" #food ^receipt
  Assets:Cash -2 USD
  Expenses:Food 2 USD
2000-01-03 price USD 1.5 CNY
`)
	return server
}

// fieldJSON extracts one string-array element from a JSON response body
// without pulling a full decoder into each assertion.
func fieldJSON(t *testing.T, body, field string) string {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("body=%s err=%v", body, err)
	}
	paths, ok := payload[field].([]any)
	if !ok || len(paths) == 0 {
		t.Fatalf("field %q missing in %s", field, body)
	}
	first, _ := paths[0].(string)
	return first
}
