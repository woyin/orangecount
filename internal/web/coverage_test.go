// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"context"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

// The tests in this file walk the adapter and API edge branches (method
// gating, validation errors, fallbacks) that the happy-path tests skip.

const adapterFixture = "option \"title\" \"Fixture\"\n" +
	"2000-01-01 open Assets:Cash USD\n" +
	"2000-01-01 open Equity:Opening USD\n" +
	"2000-01-02 * \"Seed\"\n" +
	"  Assets:Cash 2.50 USD\n" +
	"  Equity:Opening -2.50 USD\n"

func newAdapterServer(t *testing.T, text string) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil || built.Err != nil {
		t.Fatalf("build diagnostics=%+v err=%v", built.Diagnostics, built.Err)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	return server, dir
}

func get(t *testing.T, server *Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

func post(t *testing.T, server *Server, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(recorder, request)
	return recorder
}

func TestFavaAdapterMethodGating(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/__orangecount/fava/changed", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT read-only route status=%d", recorder.Code)
	}
	if allow := recorder.Header().Get("Allow"); allow != "GET" {
		t.Fatalf("allow=%q", allow)
	}
}

func TestFavaAdapterNoSnapshot(t *testing.T) {
	server, err := NewServer(Config{Store: snapshot.NewStore(nil), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	recorder := get(t, server, "/__orangecount/fava/ledger_data")
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestFavaAdapterChangedMtimeComparison(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	response := get(t, server, "/__orangecount/fava/changed")
	var envelope struct {
		Data bool `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	// An unknown mtime reports changed; the current one reports unchanged.
	response = get(t, server, "/__orangecount/fava/changed?mtime=2000-01-01T00:00:00Z")
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Data {
		t.Fatalf("stale mtime should report changed: %s", response.Body.String())
	}
}

func TestFavaAdapterHelpAndDiagnostics(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	index := get(t, server, "/__orangecount/fava/help")
	if index.Code != http.StatusOK || !strings.Contains(index.Body.String(), "sections") {
		t.Fatalf("help index status=%d body=%q", index.Code, index.Body.String())
	}
	unknown := get(t, server, "/__orangecount/fava/help?topic=diagnostics%2Fnope")
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown topic status=%d", unknown.Code)
	}
	diagnostics := get(t, server, "/__orangecount/fava/diagnostics")
	if diagnostics.Code != http.StatusOK {
		t.Fatalf("diagnostics status=%d", diagnostics.Code)
	}
}

func TestFavaAdapterEditorSourceImportRoutes(t *testing.T) {
	server, dir := newAdapterServer(t, adapterFixture)
	editor := get(t, server, "/__orangecount/fava/editor")
	if editor.Code != http.StatusOK || !strings.Contains(editor.Body.String(), "main.bean") {
		t.Fatalf("editor index status=%d body=%q", editor.Code, editor.Body.String())
	}
	file := get(t, server, "/__orangecount/fava/editor?path=main.bean")
	if file.Code != http.StatusOK || !strings.Contains(file.Body.String(), "option") {
		t.Fatalf("editor file status=%d body=%q", file.Code, file.Body.String())
	}
	missing := get(t, server, "/__orangecount/fava/editor?path=nope.bean")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("editor missing status=%d", missing.Code)
	}
	sourceIndex := get(t, server, "/__orangecount/fava/source")
	if sourceIndex.Code != http.StatusOK || !strings.Contains(sourceIndex.Body.String(), "paths") {
		t.Fatalf("source index status=%d body=%q", sourceIndex.Code, sourceIndex.Body.String())
	}
	sourceFile := get(t, server, "/__orangecount/fava/source?path=main.bean")
	if sourceFile.Code != http.StatusOK || !strings.Contains(sourceFile.Body.String(), "option") {
		t.Fatalf("source file status=%d body=%q", sourceFile.Code, sourceFile.Body.String())
	}
	sourceMissing := get(t, server, "/__orangecount/fava/source?path=nope.bean")
	if sourceMissing.Code != http.StatusNotFound {
		t.Fatalf("source missing status=%d", sourceMissing.Code)
	}
	adapters := get(t, server, "/__orangecount/fava/import?kind=adapters")
	if adapters.Code != http.StatusOK || !strings.Contains(adapters.Body.String(), "beancount") {
		t.Fatalf("adapters status=%d body=%q", adapters.Code, adapters.Body.String())
	}
	importIndex := get(t, server, "/__orangecount/fava/import")
	if importIndex.Code != http.StatusOK || !strings.Contains(importIndex.Body.String(), "main.bean") {
		t.Fatalf("import index status=%d body=%q", importIndex.Code, importIndex.Body.String())
	}
	_ = dir
}

func TestFavaAdapterDownloadJournalAndEntryContext(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	download := get(t, server, "/__orangecount/fava/download-journal")
	if download.Code != http.StatusOK || !strings.Contains(download.Header().Get("Content-Disposition"), "journal.bean") {
		t.Fatalf("download status=%d disposition=%q", download.Code, download.Header().Get("Content-Disposition"))
	}
	if !strings.Contains(download.Body.String(), "Assets:Cash") {
		t.Fatalf("download body=%q", download.Body.String())
	}
	missing := get(t, server, "/__orangecount/fava/entry-context")
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("entry-context without hash status=%d", missing.Code)
	}
	unknown := get(t, server, "/__orangecount/fava/entry-context?entry_hash=zz")
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("entry-context unknown status=%d", unknown.Code)
	}
	invalidFilter := get(t, server, "/__orangecount/fava/journal?filter=%23")
	if invalidFilter.Code != http.StatusBadRequest {
		t.Fatalf("journal invalid filter status=%d", invalidFilter.Code)
	}
}

func TestFavaAdapterReportsEdges(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	queryMissing := get(t, server, "/__orangecount/fava/reports/query")
	if queryMissing.Code != http.StatusBadRequest {
		t.Fatalf("query without text status=%d", queryMissing.Code)
	}
	queryInvalid := get(t, server, "/__orangecount/fava/reports/query?query_string=NOT+SELECT")
	if queryInvalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid query status=%d", queryInvalid.Code)
	}
	statistics := get(t, server, "/__orangecount/fava/reports/statistics")
	if statistics.Code != http.StatusOK || !strings.Contains(statistics.Body.String(), "update_activity") {
		t.Fatalf("statistics status=%d body=%q", statistics.Code, statistics.Body.String())
	}
	// Conversion selector vocabulary maps onto valuation names.
	for _, conversion := range []string{"market_value", "units"} {
		response := get(t, server, "/__orangecount/fava/balance_sheet?conversion="+conversion)
		if response.Code != http.StatusOK {
			t.Fatalf("balance_sheet conversion=%s status=%d", conversion, response.Code)
		}
	}
	response := get(t, server, "/__orangecount/fava/balance_sheet?valuation=market-value")
	if response.Code != http.StatusOK {
		t.Fatalf("explicit valuation status=%d", response.Code)
	}
	changes := get(t, server, "/__orangecount/fava/reports/accounts?r=changes&interval=year&time=2000")
	if changes.Code != http.StatusOK {
		t.Fatalf("accounts changes status=%d", changes.Code)
	}
	badReport := get(t, server, "/__orangecount/fava/reports/not-a-report")
	if badReport.Code != http.StatusNotFound {
		t.Fatalf("unknown report status=%d", badReport.Code)
	}
	invalidInterval := get(t, server, "/__orangecount/fava/reports/holdings?as_of=not-a-date")
	if invalidInterval.Code != http.StatusBadRequest {
		t.Fatalf("invalid as-of status=%d", invalidInterval.Code)
	}
}

func TestIndexMethodGating(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST / status=%d", recorder.Code)
	}
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("HEAD / status=%d", recorder.Code)
	}
}

func TestQueryEndpointEdges(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/query?q=SELECT+1", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST query status=%d", recorder.Code)
	}
	if get(t, server, "/api/v1/query").Code != http.StatusBadRequest {
		t.Fatal("missing q should 400")
	}
	if get(t, server, "/api/v1/query?q=broken+(").Code != http.StatusBadRequest {
		t.Fatal("parse error should 400")
	}
	csv := get(t, server, "/api/v1/query?q=SELECT+account+FROM+accounts&format=csv")
	if csv.Code != http.StatusOK || !strings.Contains(csv.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("csv status=%d type=%q", csv.Code, csv.Header().Get("Content-Type"))
	}
	if get(t, server, "/api/v1/query?q=SELECT+account+FROM+accounts&format=xml").Code != http.StatusBadRequest {
		t.Fatal("unknown format should 400")
	}
	jsonResponse := get(t, server, "/api/v1/query?q=SELECT+account+FROM+accounts&format=json")
	if jsonResponse.Code != http.StatusOK {
		t.Fatalf("json alias status=%d", jsonResponse.Code)
	}
}

func TestEditorEndpointsValidation(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/api/v1/editor", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("bad method status=%d", recorder.Code)
	}
	if get(t, server, "/api/v1/editor/file?path=nope.bean").Code != http.StatusNotFound {
		t.Fatal("missing file should 404")
	}
	if get(t, server, "/api/v1/editor/file").Code != http.StatusBadRequest {
		t.Fatal("missing path should 400")
	}
	// Validate parses proposed content without writing.
	validate := post(t, server, "/api/v1/editor/validate", `{"path":"main.bean","content":"2000-01-01 balance Assets:Cash 1 USD\n"}`)
	if validate.Code != http.StatusOK || !strings.Contains(validate.Body.String(), "diagnostics") {
		t.Fatalf("validate status=%d body=%q", validate.Code, validate.Body.String())
	}
	validateMissing := post(t, server, "/api/v1/editor/validate", `{"path":"nope.bean"}`)
	if validateMissing.Code != http.StatusNotFound {
		t.Fatalf("validate missing status=%d", validateMissing.Code)
	}
	// Save with a stale snapshot precondition conflicts without writing.
	stale := post(t, server, "/api/v1/editor/save", `{"path":"main.bean","content":"x","expected_snapshot_id":"other"}`)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale save status=%d", stale.Code)
	}
	badJSON := post(t, server, "/api/v1/editor/save", `{`)
	if badJSON.Code != http.StatusBadRequest {
		t.Fatalf("bad json status=%d", badJSON.Code)
	}
}

func TestJournalDateRangeValidation(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	if get(t, server, "/api/v1/reports/journal?from=2020-13-01").Code != http.StatusBadRequest {
		t.Fatal("invalid from should 400")
	}
	if get(t, server, "/api/v1/reports/journal?to=2020-01-xx").Code != http.StatusBadRequest {
		t.Fatal("invalid to should 400")
	}
	if get(t, server, "/api/v1/reports/journal?from=2020-02-01&to=2020-01-01").Code != http.StatusBadRequest {
		t.Fatal("reversed range should 400")
	}
	if get(t, server, "/api/v1/reports/journal?from=2000-01-01&to=2000-12-31").Code != http.StatusOK {
		t.Fatal("valid range should 200")
	}
}

func TestReportEndpointUnknownName(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	if get(t, server, "/api/v1/reports/not-a-report").Code != http.StatusNotFound {
		t.Fatal("unknown report should 404")
	}
	if get(t, server, "/api/v1/reports/holdings?as_of=bad").Code != http.StatusBadRequest {
		t.Fatal("bad as-of should 400")
	}
}

func TestTimeFilterVocabulary(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		raw      string
		prefix   string
		begin    string
		end      string
		wantErrs bool
	}{
		{raw: "", prefix: ""},
		{raw: "all", prefix: ""},
		{raw: "year", prefix: "2026"},
		{raw: "month", prefix: "2026-08"},
		{raw: "2025", prefix: "2025"},
		{raw: "2025-04", prefix: "2025-04"},
		{raw: "2025-Q2", begin: "2025-04-01", end: "2025-07-01"},
		{raw: "2025-q4", begin: "2025-10-01", end: "2026-01-01"},
		{raw: "2025-Q5", wantErrs: true},
		{raw: "2025-13", wantErrs: true},
		{raw: "20x5", wantErrs: true},
		{raw: "whenever", wantErrs: true},
	}
	for _, tc := range cases {
		prefix, begin, end, err := reportTimeFilter(tc.raw, now)
		if tc.wantErrs {
			if err == nil {
				t.Fatalf("raw=%q expected error", tc.raw)
			}
			continue
		}
		if err != nil || prefix != tc.prefix || begin != tc.begin || end != tc.end {
			t.Fatalf("raw=%q prefix=%q begin=%q end=%q err=%v", tc.raw, prefix, begin, end, err)
		}
	}
}

func TestPeriodAndValuationFilterValidation(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	if get(t, server, "/__orangecount/fava/journal?period=century").Code != http.StatusBadRequest {
		t.Fatal("invalid period should 400")
	}
	if get(t, server, "/__orangecount/fava/journal?valuation=fair").Code != http.StatusBadRequest {
		t.Fatal("invalid valuation should 400")
	}
	if get(t, server, "/__orangecount/fava/journal?period=all").Code != http.StatusOK {
		t.Fatal("period=all should 200")
	}
}

func TestOptionsEndpointGetPostAndValidation(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	merged := get(t, server, "/api/v1/options")
	if merged.Code != http.StatusOK || !strings.Contains(merged.Body.String(), "title") {
		t.Fatalf("options status=%d body=%q", merged.Code, merged.Body.String())
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/v1/options", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT options status=%d", recorder.Code)
	}
	saved := post(t, server, "/api/v1/options", `{"locale":"zh-CN"}`)
	if saved.Code != http.StatusOK || !strings.Contains(saved.Body.String(), "saved") {
		t.Fatalf("save status=%d body=%q", saved.Code, saved.Body.String())
	}
	for _, invalid := range []string{`{"locale":"fr"}`, `{"currency":"usd"}`, `{"currency":"TOOLONGCURRENCY"}`, `{"time":"decade"}`, `{"unknown":"x"}`} {
		if response := post(t, server, "/api/v1/options", invalid); response.Code != http.StatusBadRequest {
			t.Fatalf("invalid option %s should 400, got %d", invalid, response.Code)
		}
	}
	// The saved value is now merged into GET.
	after := get(t, server, "/api/v1/options")
	if !strings.Contains(after.Body.String(), "zh-CN") {
		t.Fatalf("saved option missing: %s", after.Body.String())
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestUploadHelpersDirectly(t *testing.T) {
	dir := t.TempDir()
	if folder, err := uploadFolder([]string{dir}, ""); err != nil || folder != dir {
		t.Fatalf("default folder=%q err=%v", folder, err)
	}
	if folder, err := uploadFolder([]string{dir}, dir); err != nil || folder != dir {
		t.Fatalf("explicit folder=%q err=%v", folder, err)
	}
	if _, err := uploadFolder([]string{dir}, "/elsewhere"); err == nil {
		t.Fatal("unknown folder should error")
	}
	if validDocumentName("ok.txt") != true || validDocumentName("") || validDocumentName("..") || validDocumentName("-flag") {
		t.Fatal("validDocumentName contract broken")
	}
	if _, err := documentTargetPath(dir, "Assets:Cash", "doc.pdf"); err != nil {
		t.Fatalf("in-root target err=%v", err)
	}
	if _, err := documentTargetPath(dir, "A:../../escape", "doc.pdf"); err == nil {
		t.Fatal("escaping account should error")
	}
	if relative := documentRelativePath(dir, filepath.Join(dir, "Assets", "doc.pdf")); relative != "Assets/doc.pdf" {
		t.Fatalf("relative=%q", relative)
	}
	target := filepath.Join(dir, "doc.pdf")
	if writeErr := storeUploadedFile(target, strings.NewReader("x")); writeErr != nil {
		t.Fatalf("store err=%v", writeErr)
	}
	if content, err := os.ReadFile(target); err != nil || string(content) != "x" {
		t.Fatalf("content=%q err=%v", content, err)
	}
	if writeErr := storeUploadedFile(target, strings.NewReader("y")); writeErr == nil || writeErr.status != http.StatusConflict {
		t.Fatalf("existing target should conflict: %+v", writeErr)
	}
	failing := storeUploadedFile(filepath.Join(dir, "fail.txt"), failingReader{})
	if failing == nil || failing.status != http.StatusInternalServerError {
		t.Fatalf("copy failure should 500: %+v", failing)
	}
	if _, err := os.Stat(filepath.Join(dir, "fail.txt")); !os.IsNotExist(err) {
		t.Fatal("failed upload should be removed")
	}
	if err := atomicWrite(filepath.Join(dir, "missing-child", "f"), []byte("x"), 0o600); err == nil {
		t.Fatal("write into missing directory should fail")
	}
}

func TestCSVImportConversionBranches(t *testing.T) {
	basic := "date,account,amount\n2000-01-01,Assets:Cash,5\n"
	if out, err := csvToBeancount(basic, nil); err != nil || !strings.Contains(out, "Equity:Imported") {
		t.Fatalf("basic out=%q err=%v", out, err)
	}
	if _, err := csvToBeancount("", nil); err == nil {
		t.Fatal("empty CSV should fail")
	}
	if _, err := csvToBeancount("date,account\n2000-01-01,Assets:Cash\n", nil); err == nil || !strings.Contains(err.Error(), "amount") {
		t.Fatalf("missing column err=%v", err)
	}
	if _, err := csvToBeancount("date,account,amount\n2000-13-01,Assets:Cash,5\n", nil); err == nil || !strings.Contains(err.Error(), "row 2") {
		t.Fatalf("invalid date err=%v", err)
	}
	if _, err := csvToBeancount("date,account,amount\n2000-01-01,Assets:Cash,abc\n", nil); err == nil || !strings.Contains(err.Error(), "invalid amount") {
		t.Fatalf("invalid amount err=%v", err)
	}
	mapped, err := csvToBeancount("date,account,amount,currency,payee,narration\n2000-01-01,Assets:Cash,5,EUR,\"Pay,ee\",Note\n",
		map[string]string{"offset_account": "Equity:Custom", "currency": "USD"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mapped, "Equity:Custom") || !strings.Contains(mapped, "5 EUR") || !strings.Contains(mapped, "Pay,ee") {
		t.Fatalf("mapped output=%q", mapped)
	}
}

func TestSafeImportNameBranches(t *testing.T) {
	if _, err := safeImportName("  ", "beancount"); err == nil {
		t.Fatal("blank name should fail")
	}
	if _, err := safeImportName("sub/dir.bean", "beancount"); err == nil {
		t.Fatal("directory component should fail")
	}
	if _, err := safeImportName("data.csv", "beancount"); err == nil {
		t.Fatal("extension mismatch should fail")
	}
	if _, err := safeImportName("data.bean", "csv"); err == nil {
		t.Fatal("bean with csv adapter should fail")
	}
	if name, err := safeImportName("data.csv", "csv"); err != nil || name != "data.csv" {
		t.Fatalf("csv name=%q err=%v", name, err)
	}
	if name, err := safeImportName("data.beancount", "beancount"); err != nil || name != "data.beancount" {
		t.Fatalf("beancount name=%q err=%v", name, err)
	}
}

func TestImportPreviewRejectsIncludeAndPlugin(t *testing.T) {
	importFixture := "2000-01-01 open Assets:Cash USD\n" +
		"2000-01-01 open Equity:Imported USD\n"
	server, _ := newAdapterServer(t, importFixture)
	include := post(t, server, "/api/v1/import/preview", `{"path":"include.bean","content":"include \"other.bean\"\n"}`)
	if include.Code != http.StatusBadRequest || !strings.Contains(include.Body.String(), "include or plugin") {
		t.Fatalf("include status=%d body=%q", include.Code, include.Body.String())
	}
	plugin := post(t, server, "/api/v1/import/preview", `{"path":"plugin.bean","content":"plugin \"x\"\n"}`)
	if plugin.Code != http.StatusBadRequest {
		t.Fatalf("plugin status=%d", plugin.Code)
	}
	invalidPath := post(t, server, "/api/v1/import/preview", `{"path":"sub/x.bean","content":""}`)
	if invalidPath.Code != http.StatusBadRequest {
		t.Fatalf("invalid path status=%d", invalidPath.Code)
	}
	badCSV := post(t, server, "/api/v1/import/preview", `{"path":"rows.csv","adapter":"csv","content":"date,account\n"}`)
	if badCSV.Code != http.StatusBadRequest {
		t.Fatalf("bad csv status=%d", badCSV.Code)
	}
	// A valid CSV preview must round-trip through evaluation.
	good := post(t, server, "/api/v1/import/preview", `{"path":"rows.csv","adapter":"csv","content":"date,account,amount\n2000-01-03,Assets:Cash,1\n"}`)
	if good.Code != http.StatusOK || !strings.Contains(good.Body.String(), "preview_id") {
		t.Fatalf("good csv status=%d body=%q", good.Code, good.Body.String())
	}
	var payload struct {
		PreviewID string `json:"preview_id"`
	}
	if err := json.Unmarshal(good.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	unknown := post(t, server, "/api/v1/import/commit", `{"preview_id":"nope"}`)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown preview status=%d", unknown.Code)
	}
	stale := post(t, server, "/api/v1/import/commit", `{"preview_id":"`+payload.PreviewID+`","expected_snapshot_id":"other"}`)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale commit status=%d", stale.Code)
	}
	commit := post(t, server, "/api/v1/import/commit", `{"preview_id":"`+payload.PreviewID+`"}`)
	if commit.Code != http.StatusOK {
		t.Fatalf("commit status=%d body=%q", commit.Code, commit.Body.String())
	}
}

func TestImportEndpointRouting(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	if get(t, server, "/api/v1/import/adapters").Code != http.StatusOK {
		t.Fatal("adapters should 200")
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/api/v1/import", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("bad method status=%d", recorder.Code)
	}
	if get(t, server, "/api/v1/import/unknown").Code != http.StatusNotFound {
		t.Fatal("unknown suffix should 404")
	}
}

func TestRedactQueryPathsFallbacks(t *testing.T) {
	server, dir := newAdapterServer(t, adapterFixture)
	_ = dir
	// Rows carry internal paths; the API must answer with display paths.
	response := get(t, server, "/api/v1/query?q=SELECT+file+FROM+postings")
	if response.Code != http.StatusOK && response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestReplaceGraphFileFailurePaths(t *testing.T) {
	server, dir := newAdapterServer(t, adapterFixture)
	current := server.store.Current()
	if current == nil {
		t.Fatal("no snapshot")
	}
	file, display, ok := graphFile(current.Graph(), "main.bean")
	if !ok {
		t.Fatal("entry file missing")
	}
	// A vanished file cannot be replaced.
	if err := os.Remove(file.Path); err != nil {
		t.Fatal(err)
	}
	if _, _, err := server.replaceGraphFile(current, file.Path, display, []byte("x")); err == nil {
		t.Fatal("missing file should fail")
	}
	_ = dir
}

func TestSourceRootHelpersAndDocumentRootErrors(t *testing.T) {
	// No document roots configured: upload and move both refuse early.
	built := snapshot.Build(filepath.Join(t.TempDir(), "missing.bean"))
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	upload := post(t, server, "/__orangecount/fava/document", "")
	if upload.Code != http.StatusBadRequest && upload.Code != http.StatusServiceUnavailable && upload.Code != http.StatusForbidden {
		t.Fatalf("upload without roots status=%d", upload.Code)
	}
}

func TestServeWaitReadyAndAddrContracts(t *testing.T) {
	server, _ := newAdapterServer(t, adapterFixture)
	if server.Addr() == "" {
		t.Fatal("addr should not be empty")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// A cancelled context returns promptly from WaitReady.
	if err := server.WaitReady(ctx); err == nil {
		t.Fatal("cancelled context should error")
	}
}

func TestDocumentRootsWithinAndResolve(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	roots, err := source.NewDocumentRoots([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := roots.Resolve("no-such-file.txt"); err == nil {
		t.Fatal("unresolved path should error")
	}
	if !pathWithin(dir, filepath.Join(dir, "x")) || pathWithin(dir, filepath.Dir(dir)) {
		t.Fatal("pathWithin contract broken")
	}
	if root := documentRootFor([]string{dir}, nested); root != dir {
		t.Fatalf("root=%q", root)
	}
	if root := documentRootFor([]string{dir}, dir); root != "" {
		t.Fatal("root itself is not a document")
	}
}

const quickFixture = "option \"operating_currency\" \"USD\"\n" +
	"2000-01-01 open Assets:Cash USD\n" +
	"2000-01-01 open Expenses:Food USD\n" +
	"2000-01-01 open Equity:Opening USD\n"

func quickRoundTrip(t *testing.T, server *Server) string {
	t.Helper()
	preview := post(t, server, "/__orangecount/fava/quick-preview", `{"text":"8 USD @Assets:Cash -> @Expenses:Food : Coffee","date":"2000-01-03"}`)
	if preview.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%q", preview.Code, preview.Body.String())
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &payload); err != nil || payload.Token == "" {
		t.Fatalf("preview body=%s err=%v", preview.Body.String(), err)
	}
	commit := post(t, server, "/__orangecount/fava/quick-commit", `{"token":"`+payload.Token+`"}`)
	if commit.Code != http.StatusOK {
		t.Fatalf("commit status=%d body=%q", commit.Code, commit.Body.String())
	}
	return payload.Token
}

func TestQuickEntryPreviewCommitUndo(t *testing.T) {
	server, _ := newAdapterServer(t, quickFixture)
	// Preview with compile errors reports them without a token.
	badPreview := post(t, server, "/__orangecount/fava/quick-preview", `{"text":"8 USD @nope -> @Expenses:Food","date":"2000-01-03"}`)
	if badPreview.Code != http.StatusOK || !strings.Contains(badPreview.Body.String(), "E-QUICK-SOURCE") {
		t.Fatalf("bad preview status=%d body=%q", badPreview.Code, badPreview.Body.String())
	}
	if !strings.Contains(badPreview.Body.String(), `"has_errors":true`) {
		t.Fatalf("bad preview should flag errors: %s", badPreview.Body.String())
	}
	unknownTarget := post(t, server, "/__orangecount/fava/quick-preview", `{"text":"8 USD @Assets:Cash -> @Expenses:Food","date":"2000-01-03","target":"nope.bean"}`)
	if unknownTarget.Code != http.StatusBadRequest {
		t.Fatalf("unknown target status=%d", unknownTarget.Code)
	}
	quickRoundTrip(t, server)
	// Undo restores the previous file content and snapshot.
	undo := post(t, server, "/__orangecount/fava/quick-undo", "")
	if undo.Code != http.StatusOK || !strings.Contains(undo.Body.String(), `"undone":true`) {
		t.Fatalf("undo status=%d body=%q", undo.Code, undo.Body.String())
	}
	// A second undo finds no batch.
	again := post(t, server, "/__orangecount/fava/quick-undo", "")
	if again.Code != http.StatusNotFound {
		t.Fatalf("second undo status=%d", again.Code)
	}
}

func TestQuickEntryCommitErrorBranches(t *testing.T) {
	server, _ := newAdapterServer(t, quickFixture)
	unknown := post(t, server, "/__orangecount/fava/quick-commit", `{"token":"nope"}`)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown token status=%d", unknown.Code)
	}
	// Take a token, then expire the precondition by passing a stale snapshot.
	preview := post(t, server, "/__orangecount/fava/quick-preview", `{"text":"8 USD @Assets:Cash -> @Expenses:Food","date":"2000-01-03"}`)
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	stale := post(t, server, "/__orangecount/fava/quick-commit", `{"token":"`+payload.Token+`","expected_snapshot_id":"other"}`)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale commit status=%d", stale.Code)
	}
	// A compile-error preview yields no token at all.
	broken := post(t, server, "/__orangecount/fava/quick-preview", `{"text":"","date":"2000-01-03"}`)
	if broken.Code != http.StatusOK {
		t.Fatalf("broken preview status=%d", broken.Code)
	}
	var brokenPayload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(broken.Body.Bytes(), &brokenPayload); err != nil || brokenPayload.Token != "" {
		t.Fatalf("broken preview must not carry a token: %s err=%v", broken.Body.String(), err)
	}
}

func TestQuickEntryUndoConflictBranches(t *testing.T) {
	server, _ := newAdapterServer(t, quickFixture)
	quickRoundTrip(t, server)
	// Simulate the ledger moving on: patch the recorded snapshot id.
	server.writeMu.Lock()
	batch := *server.quickLastBatch
	batch.SnapshotID = "different"
	server.quickLastBatch = &batch
	server.writeMu.Unlock()
	moved := post(t, server, "/__orangecount/fava/quick-undo", "")
	if moved.Code != http.StatusConflict {
		t.Fatalf("moved-on undo status=%d", moved.Code)
	}
	// Restore the record, but make the file end differ so the suffix check
	// conflicts.
	server.writeMu.Lock()
	batch = *server.quickLastBatch
	batch.SnapshotID = server.store.Current().ID
	server.quickLastBatch = &batch
	server.writeMu.Unlock()
	target := batch.TargetFile
	handle, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handle.WriteString("\n2000-01-05 note Assets:Cash \"later edit\"\n"); err != nil {
		t.Fatal(err)
	}
	handle.Close()
	// The reload from the append makes the snapshot differ again; refresh id.
	if result := server.store.Reload(server.store.Current().EntryPath, snapshot.BuildOptions{}); result.Snapshot != nil {
		server.writeMu.Lock()
		batch.SnapshotID = result.Snapshot.ID
		server.quickLastBatch = &batch
		server.writeMu.Unlock()
	}
	conflict := post(t, server, "/__orangecount/fava/quick-undo", "")
	if conflict.Code != http.StatusConflict {
		t.Fatalf("suffix-mismatch undo status=%d body=%q", conflict.Code, conflict.Body.String())
	}
	// A vanished target file surfaces the read error.
	server.writeMu.Lock()
	batch = *server.quickLastBatch
	previous := batch.TargetFile
	batch.TargetFile = filepath.Join(filepath.Dir(previous), "gone.bean")
	server.quickLastBatch = &batch
	server.writeMu.Unlock()
	missing := post(t, server, "/__orangecount/fava/quick-undo", "")
	if missing.Code != http.StatusInternalServerError {
		t.Fatalf("missing-file undo status=%d", missing.Code)
	}
}

func TestQuickProfileListingAndSave(t *testing.T) {
	server, _ := newAdapterServer(t, quickFixture+
		"2000-01-01 custom \"orangecount.quick-account.v1\" \"cash\" Assets:Cash\n")
	listing := get(t, server, "/__orangecount/fava/quick-profile")
	if listing.Code != http.StatusOK || !strings.Contains(listing.Body.String(), "Assets:Cash") {
		t.Fatalf("profile listing status=%d body=%q", listing.Code, listing.Body.String())
	}
	save := post(t, server, "/__orangecount/fava/quick-profile-save", `{"date":"2000-01-02","type":"account","name":"food","account":"Expenses:Food"}`)
	if save.Code != http.StatusOK || !strings.Contains(save.Body.String(), `"published":true`) {
		t.Fatalf("profile save status=%d body=%q", save.Code, save.Body.String())
	}
	// The saved rule now appears in the effective profile.
	after := get(t, server, "/__orangecount/fava/quick-profile")
	if !strings.Contains(after.Body.String(), "Expenses:Food") {
		t.Fatalf("saved rule missing: %s", after.Body.String())
	}
	templateSave := post(t, server, "/__orangecount/fava/quick-profile-save", `{"date":"2000-01-03","type":"template","name":"lunch","source":"Assets:Cash","destination":"Expenses:Food","currency":"USD","payee":"Deli","narration":"Lunch"}`)
	if templateSave.Code != http.StatusOK {
		t.Fatalf("template save status=%d body=%q", templateSave.Code, templateSave.Body.String())
	}
}

func TestQuickProfileSaveValidation(t *testing.T) {
	server, _ := newAdapterServer(t, quickFixture)
	cases := []struct {
		name string
		body string
	}{
		{"bad date", `{"date":"x","type":"account","name":"n","account":"Assets:Cash"}`},
		{"blank name", `{"date":"2000-01-02","type":"account","name":" ","account":"Assets:Cash"}`},
		{"blank account", `{"date":"2000-01-02","type":"account","name":"n","account":" "}`},
		{"bad account", `{"date":"2000-01-02","type":"account","name":"n","account":"lower"}`},
		{"bad type", `{"date":"2000-01-02","type":"other","name":"n"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if response := post(t, server, "/__orangecount/fava/quick-profile-save", tc.body); response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
			}
		})
	}
}

func TestSerializeQuickProfileDirectiveShapes(t *testing.T) {
	account, err := serializeQuickProfileDirective("2000-01-02", "account", "nam\"e", "Assets:Cash", "", "", "", "", "")
	if err != nil || !strings.Contains(account, `nam\"e`) || !strings.Contains(account, "Assets:Cash") {
		t.Fatalf("account directive=%q err=%v", account, err)
	}
	template, err := serializeQuickProfileDirective("2000-01-02", "template", "lunch", "", " Assets:Cash ", "Expenses:Food", "USD", "Deli", "Lunch")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"quick-template", `source: "Assets:Cash"`, `destination: "Expenses:Food"`, `currency: "USD"`, `payee: "Deli"`, `narration: "Lunch"`} {
		if !strings.Contains(template, want) {
			t.Fatalf("template directive missing %s: %q", want, template)
		}
	}
	bare, err := serializeQuickProfileDirective("2000-01-02", "template", "minimal", "", "", "", "", "", "")
	if err != nil || strings.Contains(bare, "source") {
		t.Fatalf("minimal template=%q err=%v", bare, err)
	}
	if escapeBeancountString(`a\b "c"`) != `a\\b \"c\"` {
		t.Fatal("escape contract broken")
	}
	if firstOperatingCurrency(map[string]string{"operating_currency": " USD , EUR "}) != "USD" {
		t.Fatal("first operating currency contract broken")
	}
	if firstOperatingCurrency(map[string]string{}) != "" {
		t.Fatal("missing operating currency should be empty")
	}
}

func TestFavaAddEntriesRoundTrip(t *testing.T) {
	server, _ := newAdapterServer(t, quickFixture)
	body := `{"entries":[{"type":"transaction","date":"2000-01-04","flag":"*","narration":"Added","postings":[{"account":"Assets:Cash","amount":"-3","currency":"USD"},{"account":"Expenses:Food","amount":"3","currency":"USD"}]}]}`
	added := post(t, server, "/__orangecount/fava/add-entries", body)
	if added.Code != http.StatusOK || !strings.Contains(added.Body.String(), `"published":true`) {
		t.Fatalf("add-entries status=%d body=%q", added.Code, added.Body.String())
	}
	invalid := post(t, server, "/__orangecount/fava/add-entries", `{"entries":[{"type":"bogus"}]}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid entry status=%d", invalid.Code)
	}
	badJSON := post(t, server, "/__orangecount/fava/add-entries", `{`)
	if badJSON.Code != http.StatusBadRequest {
		t.Fatalf("bad json status=%d", badJSON.Code)
	}
	// An unbalanced entry is rejected by graph revalidation.
	unbalanced := post(t, server, "/__orangecount/fava/add-entries", `{"entries":[{"type":"transaction","date":"2000-01-05","flag":"*","narration":"Broken","postings":[{"account":"Assets:Cash","amount":"-3","currency":"USD"}]}]}`)
	if unbalanced.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unbalanced status=%d body=%q", unbalanced.Code, unbalanced.Body.String())
	}
}

func TestEditorSaveValidatesContent(t *testing.T) {
	server, _ := newAdapterServer(t, quickFixture)
	broken := post(t, server, "/api/v1/editor/save", `{"path":"main.bean","content":"2000-01-01 open Assets:Cash USD\n2000-01-02 close Assets:Cash\n2000-01-03 * \"x\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"}`)
	if broken.Code != http.StatusUnprocessableEntity {
		t.Fatalf("closed-account save status=%d body=%q", broken.Code, broken.Body.String())
	}
	// The failed validation restored the original bytes.
	current := server.store.Current()
	file, _, ok := graphFile(current.Graph(), "main.bean")
	if !ok || !strings.Contains(string(file.Data), "operating_currency") {
		t.Fatalf("file was not restored: %q", file.Data)
	}
	missing := post(t, server, "/api/v1/editor/save", `{"path":"nope.bean","content":"x"}`)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing save status=%d", missing.Code)
	}
}
