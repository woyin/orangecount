// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

func TestServerWaitReadyReturnsListenFailure(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()

	server, err := NewServer(Config{Store: snapshot.NewStore(nil), Addr: occupied.Addr().String()})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(context.Background()) }()
	if err := server.WaitReady(context.Background()); !errors.Is(err, syscall.EADDRINUSE) {
		t.Fatalf("WaitReady error=%v, want address-in-use", err)
	}
	if err := <-done; !errors.Is(err, syscall.EADDRINUSE) {
		t.Fatalf("Serve error=%v, want address-in-use", err)
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

func TestRepairGuidanceHelpAndBoundedContext(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	valid := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"
	if err := os.WriteFile(entry, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	store := snapshot.NewStore(nil)
	if result := store.Reload(entry, snapshot.BuildOptions{}); result.Snapshot == nil {
		t.Fatalf("initial build diagnostics=%+v", result.Diagnostics)
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
	help := request("/api/v1/help?topic=diagnostics%2FE-EVAL-UNBALANCED&locale=zh-CN")
	if help.Code != http.StatusOK || !strings.Contains(help.Body.String(), `"topic":"diagnostics/E-EVAL-UNBALANCED"`) || !strings.Contains(help.Body.String(), "检查交易") {
		t.Fatalf("help status=%d body=%q", help.Code, help.Body.String())
	}
	helpIndex := request("/api/v1/help?locale=zh-CN")
	if helpIndex.Code != http.StatusOK || !strings.Contains(helpIndex.Body.String(), `"title":"诊断"`) {
		t.Fatalf("localized help index status=%d body=%q", helpIndex.Code, helpIndex.Body.String())
	}
	missing := request("/api/v1/help?topic=diagnostics%2FE-NOT-RELEASED")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing topic status=%d body=%q", missing.Code, missing.Body.String())
	}
	missingChinese := request("/api/v1/help?topic=diagnostics%2FE-NOT-RELEASED&locale=zh-CN")
	if missingChinese.Code != http.StatusNotFound || !strings.Contains(missingChinese.Body.String(), "找不到本地帮助主题") {
		t.Fatalf("missing localized topic status=%d body=%q", missingChinese.Code, missingChinese.Body.String())
	}
	context := request("/api/v1/diagnostics/context?path=main.bean&line=2")
	if context.Code != http.StatusOK || strings.Contains(context.Body.String(), dir) || !strings.Contains(context.Body.String(), `"focus_line":2`) || !strings.Contains(context.Body.String(), "seed") {
		t.Fatalf("context status=%d body=%q", context.Code, context.Body.String())
	}
	if strings.Contains(context.Body.String(), "2000-01-01") == false {
		t.Fatalf("context did not include adjacent line: %q", context.Body.String())
	}
	outside := request("/api/v1/diagnostics/context?path=" + url.QueryEscape(entry) + "&line=2")
	if outside.Code != http.StatusBadRequest {
		t.Fatalf("absolute context status=%d body=%q", outside.Code, outside.Body.String())
	}

	invalid := "2000-01-01 open Assets:Cash USD\n2000-02-30 * \"broken\"\n"
	if err := os.WriteFile(entry, []byte(invalid), 0o600); err != nil {
		t.Fatal(err)
	}
	failed := store.Reload(entry, snapshot.BuildOptions{})
	if failed.Snapshot != nil || store.Current() == nil || len(store.Diagnostics()) == 0 {
		t.Fatalf("failed reload did not retain valid snapshot and diagnostics: result=%+v", failed)
	}
	failedContext := request("/api/v1/diagnostics/context?path=main.bean&line=2")
	if failedContext.Code != http.StatusOK || !strings.Contains(failedContext.Body.String(), "2000-02-30") {
		t.Fatalf("failed-reload context status=%d body=%q", failedContext.Code, failedContext.Body.String())
	}
	diagnostics := request("/api/v1/diagnostics")
	if strings.Contains(diagnostics.Body.String(), "content") || strings.Contains(diagnostics.Body.String(), "broken") {
		t.Fatalf("diagnostics leaked source context: %q", diagnostics.Body.String())
	}
}

func TestDiagnosticContextExplainsUnavailableInitialBuild(t *testing.T) {
	server, err := NewServer(Config{Store: snapshot.NewStore(nil), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/diagnostics/context?path=main.bean&line=1&locale=zh-CN", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"available":false`) || !strings.Contains(recorder.Body.String(), "源文件上下文") {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
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
	guideResponse := request("/__orangecount/fava/help?topic=diagnostics%2FE-PARSE-DATE&locale=zh-CN")
	if guideResponse.Code != http.StatusOK || !strings.Contains(guideResponse.Body.String(), `"topic":"diagnostics/E-PARSE-DATE"`) || !strings.Contains(guideResponse.Body.String(), "修正日期") {
		t.Fatalf("private diagnostic guide status=%d body=%q", guideResponse.Code, guideResponse.Body.String())
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

func TestServerFavaAccountIncludesAverageCostChart(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Shares SH "AVERAGE"
2000-01-01 open Income:Gains USD
2000-01-05 * "buy first lot"
  Assets:Shares 100 SH {10 USD}
  Assets:Cash -1000 USD
2000-02-05 * "buy second lot"
  Assets:Shares 200 SH {12 USD}
  Assets:Cash -2400 USD
2000-03-05 * "sell part"
  Assets:Shares -100 SH {} @ 15 USD
  Assets:Cash 1500 USD
  Income:Gains
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
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/__orangecount/fava/reports/account?account=Assets%3AShares&period=month", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			AverageCostChart struct {
				Measure string `json:"measure"`
				Series  []struct {
					Label  string `json:"label"`
					Points []struct {
						Value struct {
							Exact string `json:"exact"`
						} `json:"value"`
					} `json:"points"`
				} `json:"series"`
			} `json:"average_cost_chart"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	chart := payload.Data.AverageCostChart
	if chart.Measure != "average-cost" || len(chart.Series) != 1 || chart.Series[0].Label != "SH (USD)" || len(chart.Series[0].Points) != 3 || chart.Series[0].Points[2].Value.Exact != "34/3" {
		t.Fatalf("average-cost chart=%+v", chart)
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
	guide := request(http.MethodGet, "/__orangecount/fava/help?topic=diagnostics%2FE-EVAL-UNBALANCED&locale=zh-CN", "")
	if guide.Code != http.StatusOK || !strings.Contains(guide.Body.String(), `"topic":"diagnostics/E-EVAL-UNBALANCED"`) || !strings.Contains(guide.Body.String(), "检查交易") {
		t.Fatalf("adapter guide status=%d body=%q", guide.Code, guide.Body.String())
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
	previews := newImportPreviewStore()
	for i := 0; i < maxImportPreviews*2; i++ {
		id := fmt.Sprintf("preview-%d", i)
		previews.Store(id, importPreview{Path: "p.bean", Content: "content"})
	}
	count := previews.len()
	if count > maxImportPreviews {
		t.Fatalf("pending map grew to %d entries, cap is %d", count, maxImportPreviews)
	}
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

// TestFavaAdapterDocumentUpload mirrors Fava's put_document layout: uploads
// land in <root>/<account parts>/<filename>, the target is served back by the
// /documents/ route, and invalid accounts or existing targets are rejected.
func TestFavaAdapterDocumentUpload(t *testing.T) {
	ledgerDir := t.TempDir()
	entry := filepath.Join(ledgerDir, "main.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	root := t.TempDir()
	roots, err := source.NewDocumentRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), DocumentRoots: roots, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	upload := func(account, folder, filename, content string) *httptest.ResponseRecorder {
		var body strings.Builder
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("account", account)
		if folder != "" {
			_ = writer.WriteField("folder", folder)
		}
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/__orangecount/fava/document", strings.NewReader(body.String()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	// A traversal-styled filename is reduced to its basename.
	success := upload("Assets:Cash", "", "../receipt.pdf", "pdf-bytes")
	if success.Code != http.StatusOK || !strings.Contains(success.Body.String(), `"filename":"Assets/Cash/receipt.pdf"`) {
		t.Fatalf("upload status=%d body=%q", success.Code, success.Body.String())
	}
	saved, err := os.ReadFile(filepath.Join(root, "Assets", "Cash", "receipt.pdf"))
	if err != nil || string(saved) != "pdf-bytes" {
		t.Fatalf("saved file err=%v content=%q", err, saved)
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/documents/Assets%2FCash%2Freceipt.pdf", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "pdf-bytes" {
		t.Fatalf("served document status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	duplicate := upload("Assets:Cash", "", "receipt.pdf", "other")
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d body=%q", duplicate.Code, duplicate.Body.String())
	}
	badAccount := upload("Assets:Missing", "", "other.pdf", "x")
	if badAccount.Code != http.StatusBadRequest || !strings.Contains(badAccount.Body.String(), "Not a valid account") {
		t.Fatalf("bad account status=%d body=%q", badAccount.Code, badAccount.Body.String())
	}
	badFolder := upload("Assets:Cash", "/tmp/not-a-root", "other.pdf", "x")
	if badFolder.Code != http.StatusBadRequest || !strings.Contains(badFolder.Body.String(), "Not a documents folder") {
		t.Fatalf("bad folder status=%d body=%q", badFolder.Code, badFolder.Body.String())
	}
	// An explicit configured folder is accepted. Roots are symlink-resolved
	// (e.g. macOS /var -> /private/var), so send the resolved path.
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	explicit := upload("Assets:Cash", resolved, "second.pdf", "second")
	if explicit.Code != http.StatusOK || !strings.Contains(explicit.Body.String(), `"filename":"Assets/Cash/second.pdf"`) {
		t.Fatalf("explicit folder status=%d body=%q", explicit.Code, explicit.Body.String())
	}
	// Uploads are same-origin write paths.
	cross := httptest.NewRequest(http.MethodPost, "/__orangecount/fava/document", strings.NewReader(""))
	cross.Header.Set("Origin", "http://evil.example")
	cross.Host = "127.0.0.1:9"
	crossRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(crossRecorder, cross)
	if crossRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-origin upload status=%d", crossRecorder.Code)
	}
}

// TestFavaAdapterDocumentUploadRequiresRoot ensures uploads are rejected with
// a clear message when no document root was configured.
func TestFavaAdapterDocumentUploadRequiresRoot(t *testing.T) {
	ledgerDir := t.TempDir()
	entry := filepath.Join(ledgerDir, "main.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n"), 0o600); err != nil {
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
	var body strings.Builder
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("account", "Assets:Cash")
	part, _ := writer.CreateFormFile("file", "x.pdf")
	_, _ = part.Write([]byte("x"))
	_ = writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/__orangecount/fava/document", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "No document root") {
		t.Fatalf("no-root status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

// TestFavaAdapterDocumentMove mirrors Fava's move_document endpoint: an
// attachment is relocated into another account's subfolder chain (optionally
// renamed) within the same document root, without overwriting targets.
func TestFavaAdapterDocumentMove(t *testing.T) {
	ledgerDir := t.TempDir()
	entry := filepath.Join(ledgerDir, "main.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Bank USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	root := t.TempDir()
	roots, err := source.NewDocumentRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), DocumentRoots: roots, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "Assets", "Cash"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Assets", "Cash", "doc.pdf"), []byte("doc-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	move := func(filename, account, newName string) *httptest.ResponseRecorder {
		body := fmt.Sprintf(`{"filename":%q,"account":%q,"new_name":%q}`, filename, account, newName)
		req := httptest.NewRequest(http.MethodPost, "/__orangecount/fava/move-document", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	success := move("Assets/Cash/doc.pdf", "Assets:Bank", "moved.pdf")
	if success.Code != http.StatusOK || !strings.Contains(success.Body.String(), `"filename":"Assets/Bank/moved.pdf"`) {
		t.Fatalf("move status=%d body=%q", success.Code, success.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "Assets", "Cash", "doc.pdf")); !os.IsNotExist(err) {
		t.Fatalf("source still present after move, err=%v", err)
	}
	saved, err := os.ReadFile(filepath.Join(root, "Assets", "Bank", "moved.pdf"))
	if err != nil || string(saved) != "doc-bytes" {
		t.Fatalf("moved file err=%v content=%q", err, saved)
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/documents/Assets%2FBank%2Fmoved.pdf", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "doc-bytes" {
		t.Fatalf("served moved document status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	missing := move("Assets/Cash/nope.pdf", "Assets:Bank", "")
	if missing.Code != http.StatusBadRequest || !strings.Contains(missing.Body.String(), "could not be found") {
		t.Fatalf("missing source status=%d body=%q", missing.Code, missing.Body.String())
	}
	badAccount := move("Assets/Bank/moved.pdf", "Assets:Missing", "")
	if badAccount.Code != http.StatusBadRequest || !strings.Contains(badAccount.Body.String(), "Not a valid account") {
		t.Fatalf("bad account status=%d body=%q", badAccount.Code, badAccount.Body.String())
	}
	if err := os.WriteFile(filepath.Join(root, "Assets", "Cash", "clash.pdf"), []byte("clash"), 0o600); err != nil {
		t.Fatal(err)
	}
	overwrite := move("Assets/Bank/moved.pdf", "Assets:Cash", "clash.pdf")
	if overwrite.Code != http.StatusConflict {
		t.Fatalf("overwrite status=%d body=%q", overwrite.Code, overwrite.Body.String())
	}
	cross := httptest.NewRequest(http.MethodPost, "/__orangecount/fava/move-document", strings.NewReader(`{"filename":"x","account":"y"}`))
	cross.Header.Set("Origin", "http://evil.example")
	cross.Host = "127.0.0.1:9"
	crossRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(crossRecorder, cross)
	if crossRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-origin move status=%d", crossRecorder.Code)
	}
}

func TestHTTPHandlersRejectUnsupportedMethodsAndProtectEmptySnapshots(t *testing.T) {
	server, err := NewServer(Config{Store: snapshot.NewStore(nil), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/api/v1/status", "/api/v1/diagnostics", "/api/v1/reports/accounts", "/api/v1/query", "/api/v1/source", "/api/v1/help", "/app.js", "/app.css", "/documents/file.txt"} {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s status=%d body=%q", path, recorder.Code, recorder.Body.String())
		}
	}
	for _, path := range []string{"/api/v1/editor", "/api/v1/import/adapters"} {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))
		if recorder.Code != http.StatusServiceUnavailable {
			t.Errorf("POST %s status=%d body=%q", path, recorder.Code, recorder.Body.String())
		}
	}
	for _, path := range []string{"/api/v1/reports/accounts", "/api/v1/query?q=SELECT+account+FROM+accounts", "/api/v1/source", "/api/v1/editor", "/__orangecount/fava/ledger_data"} {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusServiceUnavailable {
			t.Errorf("GET %s status=%d body=%q", path, recorder.Code, recorder.Body.String())
		}
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/__orangecount/fava/ledger_data", nil))
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != "GET" {
		t.Fatalf("private adapter method status=%d allow=%q", recorder.Code, recorder.Header().Get("Allow"))
	}
}

func TestEditorImportAndOptionsFailureContracts(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	content := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n"
	if err := os.WriteFile(entry, []byte(content), 0o600); err != nil {
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
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{http.MethodGet, "/api/v1/options", "", http.StatusOK},
		{http.MethodPut, "/api/v1/options", "", http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/v1/options", "not-json", http.StatusBadRequest},
		{http.MethodPost, "/api/v1/options", `{"currency":"usd"}`, http.StatusBadRequest},
		{http.MethodGet, "/api/v1/editor/file", "", http.StatusBadRequest},
		{http.MethodPost, "/api/v1/editor/validate", "not-json", http.StatusBadRequest},
		{http.MethodPost, "/api/v1/editor/validate", `{"path":"missing.bean","content":""}`, http.StatusNotFound},
		{http.MethodPost, "/api/v1/editor/validate", `{"path":"main.bean","content":"2000-01-01 open"}`, http.StatusOK},
		{http.MethodPost, "/api/v1/editor/save", `{"path":"main.bean","content":"` + `2000-01-01 open Assets:Cash USD` + `","expected_snapshot_id":"stale"}`, http.StatusConflict},
		{http.MethodPost, "/api/v1/editor/save", `{"path":"missing.bean","content":""}`, http.StatusNotFound},
		{http.MethodGet, "/api/v1/import", "", http.StatusOK},
		{http.MethodGet, "/api/v1/import/unknown", "", http.StatusNotFound},
		{http.MethodPut, "/api/v1/import", "", http.StatusNotFound},
		{http.MethodPost, "/api/v1/import/preview", `{}`, http.StatusBadRequest},
		{http.MethodPost, "/api/v1/import/preview", `{"path":"bank.txt","adapter":"text","content":""}`, http.StatusBadRequest},
		{http.MethodPost, "/api/v1/import/preview", `{"path":"bank.bean","content":"2000-01-01 open"}`, http.StatusOK},
		{http.MethodPost, "/api/v1/import/preview", `{"path":"bank.bean","content":"include \"other.bean\""}`, http.StatusBadRequest},
		{http.MethodPost, "/api/v1/import/preview", `{"path":"bank.csv","adapter":"csv","content":"date,account,amount\n"}`, http.StatusBadRequest},
	} {
		response := request(tc.method, tc.path, tc.body)
		if response.Code != tc.status {
			t.Errorf("%s %s status=%d want=%d body=%q", tc.method, tc.path, response.Code, tc.status, response.Body.String())
		}
	}
	// JSON decoders must reject a second object: otherwise a proxy could make
	// two callers disagree about which state-changing body was accepted.
	if response := request(http.MethodPost, "/api/v1/options", `{} {}`); response.Code != http.StatusBadRequest {
		t.Fatalf("multiple JSON documents status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestReportEndpointMatrixKeepsAllViewsReachable(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	content := `option "operating_currency" "USD"
2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Broker SH
2000-01-01 open Income:Salary USD
2000-01-01 open Expenses:Food USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "seed" #tag
  Assets:Cash 100 USD
  Equity:Opening -100 USD
2000-02-02 * "buy"
  Assets:Broker 2 SH {10 USD}
  Assets:Cash -20 USD
2000-03-02 * "payday"
  Assets:Cash 5 USD
  Income:Salary -5 USD
2000-03-03 * "groceries"
  Assets:Cash -2 USD
  Expenses:Food 2 USD
2000-03-04 price SH 11 USD
2000-03-05 event "status" "ok"
2000-03-06 document Assets:Cash "receipt.pdf"
`
	if err := os.WriteFile(entry, []byte(content), 0o600); err != nil {
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
	for _, path := range []string{
		"/api/v1/status",
		"/api/v1/reports/accounts?account=Assets%3ACash&r=changes&interval=month",
		"/api/v1/reports/accounts?account=Assets%3ACash&r=balances&interval=quarter",
		"/api/v1/reports/journal?flag=*&tag=tag&payee=seed",
		"/api/v1/reports/trial-balance?currency=USD&period=year",
		"/api/v1/reports/balance-sheet?currency=USD&valuation=market-value",
		"/api/v1/reports/income-statement?currency=USD&period=quarter",
		"/api/v1/reports/holdings?as_of=2000-03-01&currency=USD&aggregation=currency",
		"/api/v1/reports/prices",
		"/api/v1/reports/events",
		"/api/v1/reports/documents",
		"/api/v1/reports/statistics",
		"/api/v1/reports/errors",
	} {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
			t.Errorf("GET %s status=%d content-type=%q body=%q", path, recorder.Code, recorder.Header().Get("Content-Type"), recorder.Body.String())
		}
	}
	for _, path := range []string{
		"/api/v1/reports/accounts?format=xml",
		"/api/v1/reports/holdings?as_of=not-a-date",
		"/api/v1/reports/unknown",
		"/api/v1/query?q=SELECT+account+FROM+accounts&format=xml",
		"/api/v1/query?q=not+a+query",
		"/api/v1/source?path=missing.bean",
	} {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code < http.StatusBadRequest {
			t.Errorf("GET %s status=%d should reject request", path, recorder.Code)
		}
	}
}

func TestServerConstructionAndDocumentWriteEdgeContracts(t *testing.T) {
	if _, err := NewServer(Config{Addr: "127.0.0.1:0"}); err == nil {
		t.Fatal("server accepted a nil snapshot store")
	}
	for address, want := range map[string]bool{"localhost:5000": true, "[::1]:5000": true, "127.0.0.1:5000": true, "0.0.0.0:5000": false, "missing-port": false} {
		if got := loopbackAddr(address); got != want {
			t.Errorf("loopbackAddr(%q)=%v, want %v", address, got, want)
		}
	}
	var nilServer *Server
	if nilServer.Addr() != "" || nilServer.Serve(context.Background()) == nil {
		t.Fatal("nil server helpers accepted an unusable server")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	server, err := NewServer(Config{Store: snapshot.NewStore(nil), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.WaitReady(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled WaitReady error=%v", err)
	}

	ledgerDir := t.TempDir()
	entry := filepath.Join(ledgerDir, "main.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Bank USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	root := t.TempDir()
	roots, err := source.NewDocumentRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	server, err = NewServer(Config{Store: snapshot.NewStore(built.Snapshot), DocumentRoots: roots, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body, contentType string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", contentType)
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	if response := call(http.MethodPost, "/__orangecount/fava/document", "not-a-multipart-body", "multipart/form-data; boundary=bad"); response.Code != http.StatusBadRequest {
		t.Fatalf("malformed upload status=%d body=%q", response.Code, response.Body.String())
	}
	var multipartBody strings.Builder
	multipartWriter := multipart.NewWriter(&multipartBody)
	if err := multipartWriter.WriteField("account", "Assets:Cash"); err != nil {
		t.Fatal(err)
	}
	if err := multipartWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if response := call(http.MethodPost, "/__orangecount/fava/document", multipartBody.String(), multipartWriter.FormDataContentType()); response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "No file") {
		t.Fatalf("empty upload status=%d body=%q", response.Code, response.Body.String())
	}
	if response := call(http.MethodGet, "/__orangecount/fava/move-document", "", ""); response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "POST" {
		t.Fatalf("move method status=%d allow=%q", response.Code, response.Header().Get("Allow"))
	}
	if response := call(http.MethodPost, "/__orangecount/fava/move-document", "not-json", "application/json"); response.Code != http.StatusBadRequest {
		t.Fatalf("malformed move status=%d body=%q", response.Code, response.Body.String())
	}
	if err := os.MkdirAll(filepath.Join(root, "Assets", "Cash"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Assets", "Cash", "same.pdf"), []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}
	if response := call(http.MethodPost, "/__orangecount/fava/move-document", `{"filename":"Assets/Cash/same.pdf","account":"Assets:Cash","new_name":"."}`, "application/json"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Document unchanged") {
		t.Fatalf("unchanged move status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestEditorFileAndStaticAssetEndpointsPreserveHTTPContracts(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	content := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n"
	if err := os.WriteFile(entry, []byte(content), 0o600); err != nil {
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
	request := func(method, path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
		return recorder
	}
	file := request(http.MethodGet, "/api/v1/editor/file?path=main.bean")
	if file.Code != http.StatusOK || !strings.Contains(file.Body.String(), `"content":"2000-01-01 open`) || strings.Contains(file.Body.String(), dir) {
		t.Fatalf("editor file status=%d body=%q", file.Code, file.Body.String())
	}
	for _, path := range []string{"/api/v1/editor/file?path=missing.bean"} {
		if response := request(http.MethodGet, path); response.Code != http.StatusNotFound {
			t.Errorf("editor path %s status=%d", path, response.Code)
		}
	}
	for _, tc := range []struct {
		path        string
		contentType string
	}{
		{"/app.js", "javascript"},
	} {
		response := request(http.MethodHead, tc.path)
		if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Type"), tc.contentType) {
			t.Errorf("HEAD %s status=%d content-type=%q body=%d", tc.path, response.Code, response.Header().Get("Content-Type"), response.Body.Len())
		}
	}
	if response := request(http.MethodGet, "/missing-route"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "/app.js") {
		t.Fatalf("SPA route status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestPrivateAdapterReadOnlyResourcesShareSnapshotContract(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	text := `option "title" "Adapter resources"
2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "Seed" "Narration" #tag ^link
  Assets:Cash 1 USD
  Equity:Opening -1 USD
2000-01-03 event "status" "ok"
2000-01-04 query "named" "SELECT account FROM accounts"
2000-01-05 document Assets:Cash "receipt.pdf" #proof ^doc
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
	for _, path := range []string{
		"/__orangecount/fava/changed?mtime=stale",
		"/__orangecount/fava/options",
		"/__orangecount/fava/help",
		"/__orangecount/fava/diagnostics?locale=zh-CN",
		"/__orangecount/fava/editor?path=main.bean",
		"/__orangecount/fava/import?kind=adapters",
		"/__orangecount/fava/source?path=main.bean",
		"/__orangecount/fava/journal?tag=tag&payee=seed",
		"/__orangecount/fava/download-journal?account=Assets%3ACash",
		"/__orangecount/fava/income_statement?conversion=units&period=month",
		"/__orangecount/fava/reports/statistics",
		"/__orangecount/fava/reports/events",
	} {
		response := request(path)
		if response.Code != http.StatusOK {
			t.Errorf("GET %s status=%d body=%q", path, response.Code, response.Body.String())
		}
	}
	journal := request("/__orangecount/fava/journal")
	var journalPayload struct {
		Data struct {
			Entries []struct {
				EntryHash string `json:"entry_hash"`
			} `json:"entries"`
		} `json:"data"`
	}
	if err := json.Unmarshal(journal.Body.Bytes(), &journalPayload); err != nil || len(journalPayload.Data.Entries) == 0 || journalPayload.Data.Entries[0].EntryHash == "" {
		t.Fatalf("journal payload=%q err=%v", journal.Body.String(), err)
	}
	contextResponse := request("/__orangecount/fava/entry-context?entry_hash=" + url.QueryEscape(journalPayload.Data.Entries[0].EntryHash))
	if contextResponse.Code != http.StatusOK || !strings.Contains(contextResponse.Body.String(), `"entry"`) {
		t.Fatalf("entry context status=%d body=%q", contextResponse.Code, contextResponse.Body.String())
	}
	for _, path := range []string{
		"/__orangecount/fava/entry-context",
		"/__orangecount/fava/entry-context?entry_hash=missing",
		"/__orangecount/fava/reports/query",
		"/__orangecount/fava/reports/query?query_string=not+a+query",
		"/__orangecount/fava/reports/no-such-report",
	} {
		response := request(path)
		if response.Code < http.StatusBadRequest {
			t.Errorf("GET %s unexpectedly succeeded: %d body=%q", path, response.Code, response.Body.String())
		}
	}
}

func TestPrivateAdapterAddEntriesUsesReviewedWriteWorkflow(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	initial := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n"
	if err := os.WriteFile(entry, []byte(initial), 0o600); err != nil {
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
	request := func(body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/__orangecount/fava/add-entries", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	invalid := request(`{"entries":[]}`)
	if invalid.Code != http.StatusBadRequest || !strings.Contains(invalid.Body.String(), "no entries") {
		t.Fatalf("invalid add status=%d body=%q", invalid.Code, invalid.Body.String())
	}
	added := request(`{"entries":[{"type":"note","date":"2000-01-02","account":"Assets:Cash","comment":"reviewed note"}]}`)
	if added.Code != http.StatusOK || !strings.Contains(added.Body.String(), `"published":true`) {
		t.Fatalf("add status=%d body=%q", added.Code, added.Body.String())
	}
	contents, err := os.ReadFile(entry)
	if err != nil || !strings.Contains(string(contents), `note Assets:Cash "reviewed note"`) {
		t.Fatalf("entry contents=%q err=%v", contents, err)
	}
	backup, err := os.ReadFile(entry + ".orangecount.bak")
	if err != nil || string(backup) != initial {
		t.Fatalf("backup=%q err=%v", backup, err)
	}
}

func TestImportCommitRejectsMissingStaleAndUnknownTargets(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n"), 0o600); err != nil {
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
	post := func(path, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	if response := post("/api/v1/import/commit", `{"preview_id":"missing"}`); response.Code != http.StatusNotFound {
		t.Fatalf("missing preview status=%d body=%q", response.Code, response.Body.String())
	}
	preview := post("/api/v1/import/preview", `{"path":"import.bean","content":"2000-01-02 * \"import\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"}`)
	var payload struct {
		PreviewID string `json:"preview_id"`
	}
	if preview.Code != http.StatusOK || json.Unmarshal(preview.Body.Bytes(), &payload) != nil || payload.PreviewID == "" {
		t.Fatalf("preview status=%d body=%q", preview.Code, preview.Body.String())
	}
	stale := post("/api/v1/import/commit", `{"preview_id":`+jsonString(payload.PreviewID)+`,"expected_snapshot_id":"stale"}`)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale commit status=%d body=%q", stale.Code, stale.Body.String())
	}
	unknown := post("/api/v1/import/commit", `{"preview_id":`+jsonString(payload.PreviewID)+`,"target":"unknown.bean"}`)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown target status=%d body=%q", unknown.Code, unknown.Body.String())
	}
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
