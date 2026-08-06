// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orangecount/internal/snapshot"
	"orangecount/internal/source"
	"orangecount/internal/web/favaadapter"
)

func TestServerStatusAndDocumentContainment(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "safe.txt"), []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	roots, err := source.NewDocumentRoots([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	store := snapshot.NewStore(nil)
	server, err := NewServer(Config{Store: store, DocumentRoots: roots, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))
	var status map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &status); err != nil || status["valid"] != false {
		t.Fatalf("status=%s err=%v", recorder.Body.String(), err)
	}
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/documents/safe.txt", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "safe" {
		t.Fatalf("document status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/documents/%2e%2e%2fsecret.txt", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("traversal status=%d", recorder.Code)
	}
	// Fava links the Documents report with a trailing slash, so "/documents/"
	// is a UI route rather than a request for an attachment named "". It must
	// serve the shell so the page stays bookmarkable and survives a refresh.
	for _, path := range []string{"/documents", "/documents/"} {
		recorder = httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("documents route %q status=%d", path, recorder.Code)
		}
		if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
			t.Fatalf("documents route %q content-type=%q", path, contentType)
		}
	}
}

func TestServerRejectsNonLoopbackAndServesLoopback(t *testing.T) {
	store := snapshot.NewStore(nil)
	if _, err := NewServer(Config{Store: store, Addr: "0.0.0.0:0"}); err == nil {
		t.Fatal("non-loopback address accepted")
	}
	server, err := NewServer(Config{Store: store, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()
	deadline := time.Now().Add(time.Second)
	for server.Addr() == "127.0.0.1:0" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !strings.HasPrefix(server.Addr(), "127.0.0.1:") {
		t.Fatalf("bound address=%q", server.Addr())
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestServerPortZeroReportsBoundAddressAndServesAPI(t *testing.T) {
	server, err := NewServer(Config{Store: snapshot.NewStore(nil), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()
	if err := server.WaitReady(ctx); err != nil {
		t.Fatal(err)
	}
	addr := server.Addr()
	if !strings.HasPrefix(addr, "127.0.0.1:") || strings.HasSuffix(addr, ":0") {
		t.Fatalf("expected an OS-selected loopback port, got %q", addr)
	}
	response, err := http.Get("http://" + addr + "/api/v1/status")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status endpoint returned HTTP %d", response.StatusCode)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestServerSourceAndErrorPathsUseDisplayIdentifiers(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	childDir := filepath.Join(dir, "nested")
	if err := os.Mkdir(childDir, 0o700); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(childDir, "child.bean")
	if err := os.WriteFile(entry, []byte("plugin \"example.plugin\"\ninclude \"nested/child.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 pad Assets:Cash Equity:Opening\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := snapshot.NewStore(nil)
	result := store.Reload(entry, snapshot.BuildOptions{})
	if result.Snapshot == nil {
		t.Fatalf("snapshot diagnostics=%+v", result.Diagnostics)
	}
	server, err := NewServer(Config{Store: store, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	request := func(path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		return recorder
	}
	list := request("/api/v1/source")
	if list.Code != http.StatusOK || strings.Contains(list.Body.String(), dir) {
		t.Fatalf("source list status=%d body=%q", list.Code, list.Body.String())
	}
	var paths struct {
		Paths []string `json:"paths"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &paths); err != nil {
		t.Fatal(err)
	}
	if len(paths.Paths) != 2 || paths.Paths[0] != "main.bean" || paths.Paths[1] != "nested/child.bean" {
		t.Fatalf("source paths=%v", paths.Paths)
	}
	childResponse := request("/api/v1/source?path=nested%2Fchild.bean")
	if childResponse.Code != http.StatusOK || strings.Contains(childResponse.Body.String(), dir) || !strings.Contains(childResponse.Body.String(), `"path":"nested/child.bean"`) {
		t.Fatalf("child source status=%d body=%q", childResponse.Code, childResponse.Body.String())
	}
	resolvedEntry, err := filepath.EvalSymlinks(entry)
	if err != nil {
		t.Fatal(err)
	}
	absResponse := request("/api/v1/source?path=" + url.QueryEscape(resolvedEntry))
	if absResponse.Code != http.StatusOK || strings.Contains(absResponse.Body.String(), dir) || !strings.Contains(absResponse.Body.String(), `"path":"main.bean"`) {
		t.Fatalf("absolute source lookup status=%d body=%q", absResponse.Code, absResponse.Body.String())
	}
	errors := request("/api/v1/reports/errors")
	if errors.Code != http.StatusOK || strings.Contains(errors.Body.String(), dir) || !strings.Contains(errors.Body.String(), `"path":"nested/child.bean"`) {
		t.Fatalf("error report status=%d body=%q", errors.Code, errors.Body.String())
	}
	diagnostics := request("/api/v1/diagnostics")
	if diagnostics.Code != http.StatusOK || strings.Contains(diagnostics.Body.String(), dir) || !strings.Contains(diagnostics.Body.String(), `"path":"main.bean"`) {
		t.Fatalf("diagnostics status=%d body=%q", diagnostics.Code, diagnostics.Body.String())
	}
	entries := request("/api/v1/query?q=SELECT%20file%20FROM%20entries")
	if entries.Code != http.StatusOK || strings.Contains(entries.Body.String(), dir) || !strings.Contains(entries.Body.String(), `"file":"main.bean"`) {
		t.Fatalf("entry query status=%d body=%q", entries.Code, entries.Body.String())
	}
}

func TestPrivateFavaAdapterBootstrapAndMetadata(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	text := "option \"title\" \"Adapter Fixture\"\n" +
		"2000-01-01 open Assets:Cash USD\n" +
		"2000-01-01 open Equity:Opening USD\n" +
		"2000-01-02 * \"Seed\"\n" +
		"  Assets:Cash 2.50 USD\n" +
		"  Equity:Opening -2.50 USD\n"
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
	request := func(path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		return recorder
	}
	changedResponse := request("/__orangecount/fava/changed")
	if changedResponse.Code != http.StatusOK || !strings.Contains(changedResponse.Body.String(), `"data":false`) {
		t.Fatalf("changed status=%d body=%q", changedResponse.Code, changedResponse.Body.String())
	}
	bootstrapResponse := request("/__orangecount/fava/ledger_data")
	if bootstrapResponse.Code != http.StatusOK {
		t.Fatalf("bootstrap status=%d body=%q", bootstrapResponse.Code, bootstrapResponse.Body.String())
	}
	var bootstrap favaadapter.Envelope
	if err := json.Unmarshal(bootstrapResponse.Body.Bytes(), &bootstrap); err != nil {
		t.Fatal(err)
	}
	data, ok := bootstrap.Data.(map[string]any)
	options, optionsOK := data["options"].(map[string]any)
	if !ok || !optionsOK || options["title"] != "Adapter Fixture" {
		t.Fatalf("bootstrap data=%v", bootstrap.Data)
	}
	if bootstrap.Mtime == "" {
		t.Fatal("bootstrap omitted snapshot mtime")
	}
	metadataResponse := request("/__orangecount/fava/metadata?root=Assets")
	if metadataResponse.Code != http.StatusOK || !strings.Contains(metadataResponse.Body.String(), "Assets:Cash") {
		t.Fatalf("metadata status=%d body=%q", metadataResponse.Code, metadataResponse.Body.String())
	}
	incomeResponse := request("/__orangecount/fava/income_statement?period=month")
	if incomeResponse.Code != http.StatusOK || !strings.Contains(incomeResponse.Body.String(), `"trees"`) || !strings.Contains(incomeResponse.Body.String(), "Net Profit") {
		t.Fatalf("income statement status=%d body=%q", incomeResponse.Code, incomeResponse.Body.String())
	}
	for _, route := range []string{"balance_sheet", "trial_balance"} {
		response := request("/__orangecount/fava/" + route)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"trees"`) {
			t.Fatalf("%s status=%d body=%q", route, response.Code, response.Body.String())
		}
	}
	generic := request("/__orangecount/fava/reports/events")
	if generic.Code != http.StatusOK || !strings.Contains(generic.Body.String(), `"columns"`) || !strings.Contains(generic.Body.String(), `"rows"`) {
		t.Fatalf("generic report status=%d body=%q", generic.Code, generic.Body.String())
	}
	queryResponse := request("/__orangecount/fava/reports/query?query_string=SELECT%20account%20FROM%20accounts")
	if queryResponse.Code != http.StatusOK || !strings.Contains(queryResponse.Body.String(), `"columns"`) {
		t.Fatalf("private query status=%d body=%q", queryResponse.Code, queryResponse.Body.String())
	}
	for _, route := range []string{"options", "help", "editor", "import", "source"} {
		response := request("/__orangecount/fava/" + route)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"data"`) {
			t.Fatalf("private %s status=%d body=%q", route, response.Code, response.Body.String())
		}
	}
	unknown := request("/__orangecount/fava/not-a-route")
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown private adapter route status=%d", unknown.Code)
	}
}

func TestServerQueryReportsAndEmbeddedUI(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	request := func(path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		return recorder
	}
	page := request("/")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "/app.js") {
		t.Fatalf("index status=%d body=%q", page.Code, page.Body.String())
	}
	routePage := request("/journal?view=journal")
	if routePage.Code != http.StatusOK || !strings.Contains(routePage.Body.String(), "global-time") || !strings.Contains(routePage.Body.String(), "sidebar") {
		t.Fatalf("route shell status=%d body=%q", routePage.Code, routePage.Body.String())
	}
	app := request("/app.js")
	if app.Code != http.StatusOK || !strings.Contains(app.Body.String(), "zh-CN") {
		t.Fatalf("app status=%d body prefix=%q", app.Code, app.Body.String()[:min(80, app.Body.Len())])
	}
	reportResponse := request("/api/v1/reports/accounts")
	var reportPayload map[string]any
	if err := json.Unmarshal(reportResponse.Body.Bytes(), &reportPayload); err != nil || reportResponse.Code != http.StatusOK {
		t.Fatalf("report status=%d body=%q err=%v", reportResponse.Code, reportResponse.Body.String(), err)
	}
	queryResponse := request("/api/v1/query?q=SELECT%20account%2C%20balance%20FROM%20accounts%20ORDER%20BY%20account")
	if queryResponse.Code != http.StatusOK || !strings.Contains(queryResponse.Body.String(), "columns") {
		t.Fatalf("query status=%d body=%q", queryResponse.Code, queryResponse.Body.String())
	}
	statisticsResponse := request("/api/v1/reports/statistics")
	if statisticsResponse.Code != http.StatusOK || !strings.Contains(statisticsResponse.Body.String(), "directive") {
		t.Fatalf("statistics status=%d body=%q", statisticsResponse.Code, statisticsResponse.Body.String())
	}
	chartResponse := request("/api/v1/reports/balance-sheet?period=month&currency=USD")
	if chartResponse.Code != http.StatusOK || !strings.Contains(chartResponse.Body.String(), `"chart"`) || !strings.Contains(chartResponse.Body.String(), `"kind":"line"`) {
		t.Fatalf("balance chart status=%d body=%q", chartResponse.Code, chartResponse.Body.String())
	}
	csvResponse := request("/api/v1/query?format=csv&q=SELECT%20account%20FROM%20accounts")
	if csvResponse.Code != http.StatusOK || !strings.HasPrefix(csvResponse.Body.String(), "account\n") {
		t.Fatalf("csv status=%d body=%q", csvResponse.Code, csvResponse.Body.String())
	}
	missingQuery := request("/api/v1/query")
	if missingQuery.Code != http.StatusBadRequest || !strings.Contains(missingQuery.Body.String(), "q is required") {
		t.Fatalf("missing query status=%d body=%q", missingQuery.Code, missingQuery.Body.String())
	}
}

func TestJournalReportDateFiltersAreInclusiveAndValidated(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "same day"
  Assets:Cash 1 USD
  Equity:Opening -1 USD
2000-01-03 * "next day"
  Assets:Cash 2 USD
  Equity:Opening -2 USD
`
	if err := os.WriteFile(entry, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	request := func(path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		return recorder
	}
	filtered := request("/api/v1/reports/journal?from=2000-01-02&to=2000-01-02")
	if filtered.Code != http.StatusOK {
		t.Fatalf("filtered status=%d body=%q", filtered.Code, filtered.Body.String())
	}
	var payload struct {
		Rows []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal(filtered.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Rows) != 2 {
		t.Fatalf("filtered rows=%d body=%q", len(payload.Rows), filtered.Body.String())
	}
	for _, row := range payload.Rows {
		if row["date"] != "2000-01-02" {
			t.Fatalf("filtered row=%v", row)
		}
	}
	global := request("/api/v1/reports/journal?account=Assets%3ACash&filter=same%20day")
	if global.Code != http.StatusOK {
		t.Fatalf("global status=%d body=%q", global.Code, global.Body.String())
	}
	payload.Rows = nil
	if err := json.Unmarshal(global.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Rows) != 1 || payload.Rows[0]["account"] != "Assets:Cash" {
		t.Fatalf("global rows=%v", payload.Rows)
	}
	flagged := request("/api/v1/reports/journal?flag=*")
	if flagged.Code != http.StatusOK {
		t.Fatalf("flag status=%d body=%q", flagged.Code, flagged.Body.String())
	}
	payload.Rows = nil
	if err := json.Unmarshal(flagged.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Rows) != 4 {
		t.Fatalf("flagged rows=%d body=%q", len(payload.Rows), flagged.Body.String())
	}
	csv := request("/api/v1/reports/journal?from=2000-01-02&to=2000-01-02&format=csv")
	if csv.Code != http.StatusOK || !strings.HasPrefix(csv.Body.String(), "date,account") || strings.Contains(csv.Body.String(), "display") {
		t.Fatalf("csv status=%d body=%q", csv.Code, csv.Body.String())
	}
	invalid := request("/api/v1/reports/journal?from=not-a-date")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid filter status=%d body=%q", invalid.Code, invalid.Body.String())
	}
	invalidPeriod := request("/api/v1/reports/balance-sheet?period=fortnight")
	if invalidPeriod.Code != http.StatusBadRequest {
		t.Fatalf("invalid period status=%d body=%q", invalidPeriod.Code, invalidPeriod.Body.String())
	}
	invalidAsOf := request("/api/v1/reports/holdings?as_of=not-a-date")
	if invalidAsOf.Code != http.StatusBadRequest {
		t.Fatalf("invalid as-of status=%d body=%q", invalidAsOf.Code, invalidAsOf.Body.String())
	}
}

func TestFavaAdapterJournalAcceptsQuarterTimeFilter(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-15 * "q1 txn"
  Assets:Cash 1 USD
  Equity:Opening -1 USD
2000-04-15 * "q2 txn"
  Assets:Cash 2 USD
  Equity:Opening -2 USD
`
	if err := os.WriteFile(entry, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	request := func(path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		return recorder
	}
	filtered := request("/__orangecount/fava/journal?time=2000-Q2")
	if filtered.Code != http.StatusOK {
		t.Fatalf("quarter status=%d body=%q", filtered.Code, filtered.Body.String())
	}
	body := filtered.Body.String()
	if !strings.Contains(body, "q2 txn") || strings.Contains(body, "q1 txn") {
		t.Fatalf("quarter body=%q", body)
	}
	invalid := request("/__orangecount/fava/journal?time=2000-Q5")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid quarter status=%d body=%q", invalid.Code, invalid.Body.String())
	}
}

func TestEmbeddedUIProvidesSortableTablesAndJournalDateControls(t *testing.T) {
	server, err := NewServer(Config{Store: snapshot.NewStore(nil), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("app status=%d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, marker := range []string{"aria-sort", "table-sort", "data-sort-value", "ascending", "descending", "journal-from", "journal-to", "journal-apply", "journal-reset", "journal-flag", "journal-tag", "journal-payee", "report-period", "report-valuation", "report-as-of", "report-chart", "selected", "table-pagination", "query-csv", "pathViews", "statistics", "editor-buffer", "editor-lines", "editor-validate", "editor-save", "import-adapter", "import-preview", "import-commit", "import-diff", "options-save", "help-search", "global-time", "global-account", "global-filter", "menu-toggle", "currency-switch", "own_balance", "total_balance", "tree-aggregate-badge", "chart-hierarchy-layout", "report-tree-mode", "chart-grid", "chart-tick", "chart-tooltip", "data-series-index", "hierarchy-node", "buildTooltipHtml", "samplePoints", "legendToggleStates"} {
		if !strings.Contains(body, marker) {
			t.Fatalf("embedded app missing %q", marker)
		}
	}
}

func TestTransplantedUIAssetSelection(t *testing.T) {
	t.Setenv("ORANGECOUNT_TRANSPLANTED_UI", "1")
	server, err := NewServer(Config{Store: snapshot.NewStore(nil), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "app.css") || !strings.Contains(recorder.Body.String(), "display: contents") {
		t.Fatalf("transplanted index status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/app.css", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Header().Get("Content-Type"), "text/css") || !strings.Contains(recorder.Body.String(), "--font-family") {
		t.Fatalf("transplanted css status=%d content-type=%q", recorder.Code, recorder.Header().Get("Content-Type"))
	}
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Fava-aligned shell") {
		t.Fatalf("transplanted app status=%d", recorder.Code)
	}
}

func TestEditorWriteWorkflowIsAtomicAndRevalidated(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	valid := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"
	if err := os.WriteFile(entry, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(recorder, request)
		return recorder
	}
	files := request(http.MethodGet, "/api/v1/editor", "")
	if files.Code != http.StatusOK || !strings.Contains(files.Body.String(), "main.bean") || strings.Contains(files.Body.String(), dir) {
		t.Fatalf("editor files status=%d body=%q", files.Code, files.Body.String())
	}
	invalidPreview := request(http.MethodPost, "/api/v1/editor/validate", `{"path":"main.bean","content":"2000-01-01 open"}`)
	if invalidPreview.Code != http.StatusOK || strings.Contains(invalidPreview.Body.String(), `"valid":true`) {
		t.Fatalf("invalid preview status=%d body=%q", invalidPreview.Code, invalidPreview.Body.String())
	}
	updated := strings.Replace(valid, "seed", "updated", 1)
	save := request(http.MethodPost, "/api/v1/editor/save", `{"path":"main.bean","content":`+jsonString(updated)+`}`)
	if save.Code != http.StatusOK || !strings.Contains(save.Body.String(), `"published":true`) {
		t.Fatalf("save status=%d body=%q", save.Code, save.Body.String())
	}
	backupPath := filepath.Join(dir, "main.bean.orangecount.bak")
	backup, err := os.ReadFile(backupPath)
	if err != nil || string(backup) != valid {
		t.Fatalf("backup err=%v content=%q", err, backup)
	}
	if content, err := os.ReadFile(entry); err != nil || !strings.Contains(string(content), "updated") {
		t.Fatalf("saved entry err=%v content=%q", err, content)
	}
	invalidSave := request(http.MethodPost, "/api/v1/editor/save", `{"path":"main.bean","content":"not-a-date open Assets:Cash USD\n"}`)
	if invalidSave.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidSave.Body.String(), `"published":false`) {
		t.Fatalf("invalid save status=%d body=%q", invalidSave.Code, invalidSave.Body.String())
	}
	if content, err := os.ReadFile(entry); err != nil || !strings.Contains(string(content), "updated") {
		t.Fatalf("invalid save changed entry err=%v content=%q", err, content)
	}
}

func TestImportOptionsAndHelpWorkflows(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	valid := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"
	if err := os.WriteFile(entry, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	adapters := request(http.MethodGet, "/api/v1/import/adapters", "")
	if adapters.Code != http.StatusOK || !strings.Contains(adapters.Body.String(), `"id":"csv"`) {
		t.Fatalf("adapters status=%d body=%q", adapters.Code, adapters.Body.String())
	}
	preview := request(http.MethodPost, "/api/v1/import/preview", `{"path":"bank.bean","content":"2000-01-03 * \"imported\"\n  Assets:Cash 2 USD\n  Equity:Opening -2 USD\n"}`)
	if preview.Code != http.StatusOK || !strings.Contains(preview.Body.String(), "preview_id") {
		t.Fatalf("preview status=%d body=%q", preview.Code, preview.Body.String())
	}
	var previewPayload struct {
		PreviewID string `json:"preview_id"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &previewPayload); err != nil || previewPayload.PreviewID == "" {
		t.Fatalf("preview payload=%q err=%v", preview.Body.String(), err)
	}
	csvPreview := request(http.MethodPost, "/api/v1/import/preview", `{"path":"bank.csv","adapter":"csv","mapping":{"offset_account":"Equity:Opening","currency":"USD"},"content":"date,payee,account,amount,currency,narration\n2000-01-03,Cafe,Assets:Cash,2,USD,Lunch\n"}`)
	if csvPreview.Code != http.StatusOK || !strings.Contains(csvPreview.Body.String(), "preview_id") || !strings.Contains(csvPreview.Body.String(), "added_lines") {
		t.Fatalf("csv preview status=%d body=%q", csvPreview.Code, csvPreview.Body.String())
	}
	commitBody := `{"preview_id":` + jsonString(previewPayload.PreviewID) + `,"target":"main.bean"}`
	commit := request(http.MethodPost, "/api/v1/import/commit", commitBody)
	if commit.Code != http.StatusOK || !strings.Contains(commit.Body.String(), `"published":true`) {
		t.Fatalf("commit status=%d body=%q", commit.Code, commit.Body.String())
	}
	options := request(http.MethodPost, "/api/v1/options", `{"locale":"zh-CN","currency":"CNY","time":"month"}`)
	if options.Code != http.StatusOK || !strings.Contains(options.Body.String(), `"saved":true`) {
		t.Fatalf("options status=%d body=%q", options.Code, options.Body.String())
	}
	badOptions := request(http.MethodPost, "/api/v1/options", `{"locale":"fr"}`)
	if badOptions.Code != http.StatusBadRequest {
		t.Fatalf("bad options status=%d body=%q", badOptions.Code, badOptions.Body.String())
	}
	help := request(http.MethodGet, "/api/v1/help", "")
	if help.Code != http.StatusOK || !strings.Contains(help.Body.String(), "Editor safety") {
		t.Fatalf("help status=%d body=%q", help.Code, help.Body.String())
	}
}

// TestImportPreviewEvaluatesAgainstMergedGraph ensures a preview of an import
// that posts to accounts opened only in the main ledger (not within the
// imported file) is reported as valid. Previously the preview evaluated the
// imported file in isolation, producing false E-EVAL-POSTING lifecycle errors
// for every realistic import and showing "Request failed" in the UI even
// though the commit path would succeed.
func TestImportPreviewEvaluatesAgainstMergedGraph(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	valid := "2000-01-01 open Assets:Food USD\n2000-01-01 open Equity:Imported USD\n2000-01-02 * \"seed\"\n  Assets:Food 1 USD\n  Equity:Imported -1 USD\n"
	if err := os.WriteFile(entry, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Host = "127.0.0.1:0"
		req.Header.Set("Origin", "http://127.0.0.1:0")
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	// The import posts to Assets:Food and Equity:Imported, which are opened only
	// in the main ledger. Under the old isolated evaluation this produced
	// E-EVAL-POSTING errors and valid=false.
	preview := request(http.MethodPost, "/api/v1/import/preview", `{"path":"bank.bean","content":"2000-01-03 * \"imported\"\n  Assets:Food 2 USD\n  Equity:Imported -2 USD\n"}`)
	if preview.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%q", preview.Code, preview.Body.String())
	}
	var payload struct {
		PreviewID   string `json:"preview_id"`
		Valid       bool   `json:"valid"`
		Path        string `json:"path"`
		Diagnostics []struct {
			Code string `json:"code"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &payload); err != nil {
		t.Fatalf("preview payload=%q err=%v", preview.Body.String(), err)
	}
	if !payload.Valid {
		codes := make([]string, 0, len(payload.Diagnostics))
		for _, d := range payload.Diagnostics {
			codes = append(codes, d.Code)
		}
		t.Fatalf("preview valid=false path=%q diagnostics=%v", payload.Path, codes)
	}
	if payload.PreviewID == "" {
		t.Fatalf("preview id missing: %q", preview.Body.String())
	}
}

// TestWriteEndpointsRejectCrossOrigin verifies that state-changing POST
// endpoints reject requests carrying a cross-origin Origin header (a drive-by
// CSRF vector against a co-resident browser) while still accepting requests
// with no Origin header (non-browser clients).
func TestWriteEndpointsRejectCrossOrigin(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	valid := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n"
	if err := os.WriteFile(entry, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body, origin, host string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if host != "" {
			req.Host = host
		}
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	// Cross-origin Origin must be rejected on write endpoints.
	for _, path := range []string{"/api/v1/editor/save", "/api/v1/import/preview", "/api/v1/import/commit", "/api/v1/options"} {
		body := "{}"
		recorder := request(http.MethodPost, path, body, "http://evil.example", "127.0.0.1:9")
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("%s cross-origin status=%d body=%q", path, recorder.Code, recorder.Body.String())
		}
	}
	// Same-origin Origin must be accepted.
	same := request(http.MethodPost, "/api/v1/options", `{"locale":"en"}`, "http://127.0.0.1:9", "127.0.0.1:9")
	if same.Code != http.StatusOK {
		t.Fatalf("same-origin options status=%d body=%q", same.Code, same.Body.String())
	}
}

// TestImportPreviewMapIsBounded verifies that storing more previews than the
// cap retains at most maxImportPreviews entries and that expired previews are
// evicted, so the server cannot grow its pending map without bound.
func TestImportPreviewMapIsBounded(t *testing.T) {
	server := &Server{pending: make(map[string]importPreview)}
	for i := 0; i < maxImportPreviews*2; i++ {
		id := fmt.Sprintf("preview-%d", i)
		server.storePreview(id, importPreview{Path: "p.bean", Content: "content"})
	}
	server.pendingMu.Lock()
	count := len(server.pending)
	server.pendingMu.Unlock()
	if count > maxImportPreviews {
		t.Fatalf("pending map grew to %d entries, cap is %d", count, maxImportPreviews)
	}
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
