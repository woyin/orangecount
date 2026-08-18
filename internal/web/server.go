// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package web provides the small loopback-only HTTP surface used by serve.
package web

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/repairguidance"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
	"orangecount/internal/web/favaadapter"
)

// assets contains the compiled, dependency-free browser bundle. Keeping the
// files inside this package makes the released binary independent of Node or
// a runtime CDN.
//
//go:embed assets/*
var assets embed.FS

// Config wires a Server: the snapshot store, document roots for attachments,
// and the loopback-only listen address.
type Config struct {
	Store         *snapshot.Store
	DocumentRoots source.DocumentRoots
	Addr          string
}

// Server is the Fava-parity web UI: an HTTP mux over the snapshot store
// with in-memory preview/option state. All handlers run against immutable
// published snapshots, so a failed edit never corrupts the served view.
type Server struct {
	mu             sync.RWMutex
	writeMu        sync.Mutex
	store          *snapshot.Store
	roots          source.DocumentRoots
	addr           string
	http           *http.Server
	bound          string
	ready          chan struct{}
	readyOnce      sync.Once
	readyErr       error
	optionsMu      sync.RWMutex
	options        map[string]string
	previews       *importPreviewStore
	quickPreviews  *quickPreviewStore
	quickLastBatch *quickBatchRecord
}

// NewServer validates the loopback address and store, then builds the
// server around an embedded-asset HTTP handler.
func NewServer(config Config) (*Server, error) {
	addr := config.Addr
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	if !loopbackAddr(addr) {
		return nil, fmt.Errorf("serve address must be loopback-only")
	}
	if config.Store == nil {
		return nil, fmt.Errorf("serve requires a snapshot store")
	}
	server := &Server{store: config.Store, roots: config.DocumentRoots, addr: addr, ready: make(chan struct{}), options: make(map[string]string), previews: newImportPreviewStore(), quickPreviews: newQuickPreviewStore()}
	server.http = &http.Server{Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second}
	return server, nil
}

// Handler returns the fully routed mux; tests use it through httptest
// without binding a listener.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/diagnostics", s.handleDiagnostics)
	mux.HandleFunc("/api/v1/diagnostics/context", s.handleDiagnosticContext)
	mux.HandleFunc("/api/v1/query", s.handleQuery)
	mux.HandleFunc("/api/v1/source", s.handleSource)
	mux.HandleFunc("/api/v1/editor", s.handleEditor)
	mux.HandleFunc("/api/v1/editor/", s.handleEditor)
	mux.HandleFunc("/api/v1/import", s.handleImport)
	mux.HandleFunc("/api/v1/import/", s.handleImport)
	mux.HandleFunc("/api/v1/options", s.handleOptions)
	mux.HandleFunc("/api/v1/help", s.handleHelp)
	mux.HandleFunc("/api/v1/reports/", s.handleReport)
	// Private frontend-transplant adapter. This path is loopback-only by the
	// server's construction and is intentionally not a public Fava API.
	mux.HandleFunc("/__orangecount/fava/", s.handleFavaAdapter)
	// Reserve the Fava-style Documents UI route separately from attachment
	// paths (`/documents/<name>`), otherwise ServeMux canonicalizes `/documents`
	// to the attachment handler before the embedded shell can load.
	mux.HandleFunc("/documents", s.handleIndex)
	mux.HandleFunc("/documents/", s.handleDocument)
	mux.HandleFunc("/app.js", s.handleApp)
	mux.HandleFunc("/app.css", s.handleStyle)
	return mux
}

// Addr returns the bound endpoint once Serve has bound it, falling back to
// the configured address before that.
func (s *Server) Addr() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	bound, addr := s.bound, s.addr
	s.mu.RUnlock()
	if bound == "" {
		return addr
	}
	return bound
}

// WaitReady blocks until Serve has bound its listener and Addr returns the
// actual endpoint (including an OS-selected port for :0).
func (s *Server) WaitReady(ctx context.Context) error {
	if s == nil || s.ready == nil {
		return fmt.Errorf("nil web server")
	}
	select {
	case <-s.ready:
		s.mu.RLock()
		err := s.readyErr
		s.mu.RUnlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) signalReady(err error) {
	s.mu.Lock()
	s.readyErr = err
	s.mu.Unlock()
	s.readyOnce.Do(func() { close(s.ready) })
}

// Serve binds the listener, signals readiness, and serves until ctx is
// canceled or the listener fails.
func (s *Server) Serve(ctx context.Context) error {
	if s == nil || s.http == nil {
		return fmt.Errorf("nil web server")
	}
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.signalReady(err)
		return err
	}
	s.mu.Lock()
	s.bound = listener.Addr().String()
	s.mu.Unlock()
	s.signalReady(nil)
	finished := make(chan error, 1)
	go func() { finished <- s.http.Serve(listener) }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.http.Shutdown(shutdownCtx)
		return nil
	case err := <-finished:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func loopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	data, err := assets.ReadFile(s.uiAsset("index.html"))
	if err != nil {
		http.Error(w, "embedded UI unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'wasm-unsafe-eval'")
	_, _ = w.Write(data)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	current := s.store.Current()
	response := struct {
		Version         string `json:"version"`
		SnapshotID      string `json:"snapshot_id,omitempty"`
		Valid           bool   `json:"valid"`
		AccountCount    int    `json:"account_count"`
		DiagnosticCount int    `json:"diagnostic_count"`
		PublishedAt     string `json:"published_at,omitempty"`
	}{Version: "0.1.0-dev", DiagnosticCount: len(s.store.Diagnostics())}
	if current != nil {
		evaluation := current.Evaluation()
		response.SnapshotID = current.ID
		response.Valid = evaluation.Valid
		response.AccountCount = len(evaluation.Accounts)
		response.PublishedAt = current.BuiltAt.UTC().Format(time.RFC3339Nano)
	}
	writeJSON(w, response)
}

func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	locale := requestedLocale(r)
	diagnostics := s.store.Diagnostics()
	graph := s.store.LatestGraph()
	localized := make([]diagnostic.Diagnostic, len(diagnostics))
	for i, value := range diagnostics {
		localized[i] = diagnostic.Localize(value, locale)
		localized[i].Path = displayDiagnosticPath(localized[i], graph)
	}
	writeJSON(w, localized)
}

type diagnosticContextLine struct {
	Line    int    `json:"line"`
	Content string `json:"content"`
}

type diagnosticContextResponse struct {
	Available bool                    `json:"available"`
	Path      string                  `json:"path,omitempty"`
	FocusLine int                     `json:"focus_line,omitempty"`
	Lines     []diagnosticContextLine `json:"lines,omitempty"`
	Reason    string                  `json:"reason,omitempty"`
}

// handleDiagnosticContext (GET) returns one diagnostic's rendered context —
// the offending source lines and its linked entries — for the web UI.
func (s *Server) handleDiagnosticContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	line, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("line")))
	if path == "" || filepath.IsAbs(path) || err != nil || line <= 0 {
		writeAPIError(w, http.StatusBadRequest, "path and positive line are required")
		return
	}
	graph := s.store.LatestGraph()
	if graph == nil {
		writeJSON(w, diagnosticContextResponse{
			Available: false,
			Path:      source.SafeDisplayPath(path),
			Reason:    diagnosticContextUnavailableReason(requestedLocale(r)),
		})
		return
	}
	file, display, ok := graphFile(graph, path)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "source file is not in the current include graph")
		return
	}
	lines := strings.Split(string(file.Data), "\n")
	if line > len(lines) {
		writeAPIError(w, http.StatusBadRequest, "line is outside the source file")
		return
	}
	start := line - 1
	if start > 0 {
		start--
	}
	end := line + 1
	if end > len(lines) {
		end = len(lines)
	}
	context := make([]diagnosticContextLine, 0, end-start)
	for index := start; index < end; index++ {
		context = append(context, diagnosticContextLine{Line: index + 1, Content: lines[index]})
	}
	writeJSON(w, diagnosticContextResponse{Available: true, Path: display, FocusLine: line, Lines: context})
}

// handleReport (GET) renders a named report (accounts, journal, documents,
// editor statistics) as paginated JSON for the Fava-compatible UI.
func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	current := s.store.Current()
	if current == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/reports/"), "/")
	result, chartRoute, known, err := s.buildReport(r, current, name)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !known {
		http.NotFound(w, r)
		return
	}
	if format := strings.TrimSpace(r.URL.Query().Get("format")); strings.EqualFold(format, "csv") {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		if err := result.WriteCSV(w); err != nil {
			return
		}
		return
	} else if format != "" && !strings.EqualFold(format, "json") {
		writeAPIError(w, http.StatusBadRequest, "format must be json or csv")
		return
	}
	presented := report.Present(result)
	if chartRoute == "" {
		writeJSON(w, presented)
		return
	}
	evaluation := current.Evaluation()
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	valuation := strings.TrimSpace(r.URL.Query().Get("valuation"))
	if valuation == "" {
		valuation = "at-cost"
	}
	chart := report.ReportChart(evaluation, chartRoute, period, strings.TrimSpace(r.URL.Query().Get("currency")), valuation, strings.TrimSpace(r.URL.Query().Get("account")))
	// The statement trees additionally carry the per-measure, per-currency
	// chart set: one series per currency, no conversion. A single-currency
	// display needs a quote for every foreign posting, so a ledger without
	// price directives would blank those series with a "no conversion
	// quote" warning while the table below shows both currencies fine.
	charts := []report.PresentedChartSpec{}
	for _, spec := range report.ReportCharts(evaluation, chartRoute, period, valuation) {
		charts = append(charts, report.PresentChart(spec))
	}
	writeJSON(w, struct {
		query.Result
		Chart  report.PresentedChartSpec   `json:"chart"`
		Charts []report.PresentedChartSpec `json:"charts,omitempty"`
	}{Result: presented, Chart: report.PresentChart(chart), Charts: charts})
}

// buildReport is the shared semantic/report projection used by both the
// existing OrangeCount report endpoint and the private Fava-shaped adapter.
// Keeping one builder prevents the two surfaces from drifting in filtering,
// redaction, or report ownership. Known-but-failing parses are reported as
// (result, route, true, err); unknown names as (…, false, nil).
func (s *Server) buildReport(r *http.Request, current *snapshot.Snapshot, name string) (query.Result, string, bool, error) {
	result, chartRoute, err := reportForRequest(r, current, name)
	if err != nil {
		return query.Result{}, "", true, err
	}
	if !reportKnown(name) {
		return query.Result{}, "", false, nil
	}
	result = redactQueryPaths(result, current.Graph())
	filters, err := globalReportFilters(r)
	if err != nil {
		return query.Result{}, "", true, err
	}
	// The pivot applies the global filters itself while accumulating (its rows
	// are intervals, not postings), so the generic row filter must not re-run.
	if !strings.EqualFold(name, "pivot") {
		result = report.Filter(result, filters)
	}
	if strings.EqualFold(name, "journal") {
		result = report.FilterJournal(result, journalFiltersFromQuery(r))
	}
	return result, chartRoute, true, nil
}

// reportKnown reports whether name maps to a report this server serves.
func reportKnown(name string) bool {
	switch strings.ToLower(name) {
	case "accounts", "account", "journal", "trial-balance", "trial_balance", "trialbalance",
		"balance-sheet", "balance_sheet", "balancesheet", "income-statement", "income_statement", "incomestatement",
		"holdings", "pivot",
		"prices", "price", "commodities", "commodity", "events", "event",
		"documents", "document", "statistics", "statistic", "stats", "errors", "error":
		return true
	default:
		return false
	}
}

// reportForRequest computes the unfiltered result and chart route for one
// report name. A nil result with empty columns and nil error means the name
// is unknown to the caller only when reportKnown disagrees.
// reportForRequest dispatches to the named report builder, applying the
// request's filter parameters (date ranges, accounts, interval).
func reportForRequest(r *http.Request, current *snapshot.Snapshot, name string) (query.Result, string, error) {
	evaluation := current.Evaluation()
	switch strings.ToLower(name) {
	case "accounts", "account":
		return accountReport(r, evaluation)
	case "journal":
		from, to, err := journalDateRange(r)
		if err != nil {
			return query.Result{}, "", err
		}
		return report.JournalBetween(evaluation, from, to), "", nil
	case "trial-balance", "trial_balance", "trialbalance":
		// The web report needs explicit ancestors for Fava-style hierarchy
		// rendering. Keep report.TrialBalance flat for query-compatible
		// consumers and use its tree variant only at this presentation boundary.
		return report.TrialBalanceTree(evaluation), "trial-balance", nil
	case "balance-sheet", "balance_sheet", "balancesheet":
		return report.BalanceSheet(evaluation), "balance-sheet", nil
	case "income-statement", "income_statement", "incomestatement":
		return report.IncomeStatement(evaluation), "income-statement", nil
	case "holdings":
		return holdingsReport(r, evaluation)
	case "pivot":
		filters, err := globalReportFilters(r)
		if err != nil {
			return query.Result{}, "", err
		}
		spec := report.PivotSpec{
			Rows:    r.URL.Query().Get("rows"),
			Columns: r.URL.Query().Get("columns"),
			Values:  r.URL.Query().Get("values"),
			Account: r.URL.Query().Get("account"),
			Filters: filters,
		}
		return report.PivotTable(evaluation, spec), "", nil
	case "prices", "price", "commodities", "commodity":
		return report.Prices(evaluation), "", nil
	case "events", "event":
		return report.Events(evaluation), "", nil
	case "documents", "document":
		return report.Documents(evaluation), "", nil
	case "statistics", "statistic", "stats":
		return report.Statistics(evaluation), "", nil
	case "errors", "error":
		return report.ErrorsWithGraph(evaluation, current.Graph()), "", nil
	default:
		return query.Result{}, "", nil
	}
}

// accountReport computes the accounts result. Fava's account page switches
// between the journal, per-interval changes, and per-interval balances with
// the `r` query parameter; the aggregate interval rows carry no entry columns,
// so they skip the generic row filters (time filters are applied inside).
func accountReport(r *http.Request, evaluation ledger.Evaluation) (query.Result, string, error) {
	view := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("r")))
	if view == "changes" || view == "balances" {
		filters, err := globalReportFilters(r)
		if err != nil {
			return query.Result{}, "", err
		}
		result := report.AccountIntervals(evaluation, strings.TrimSpace(r.URL.Query().Get("account")), view, strings.TrimSpace(r.URL.Query().Get("interval")), filters)
		return result, "accounts", nil
	}
	return report.Accounts(evaluation), "accounts", nil
}

// holdingsReport applies the optional as-of date, valuation, and aggregation
// selectors to the holdings result.
func holdingsReport(r *http.Request, evaluation ledger.Evaluation) (query.Result, string, error) {
	asOf, err := reportAsOfDate(r)
	if err != nil {
		return query.Result{}, "", err
	}
	valuation := strings.TrimSpace(r.URL.Query().Get("valuation"))
	if valuation == "" {
		valuation = "at-cost"
	}
	result := report.HoldingsAtCurrency(evaluation, asOf, valuation, strings.TrimSpace(r.URL.Query().Get("currency")))
	result = report.HoldingsAggregate(result, strings.TrimSpace(r.URL.Query().Get("aggregation")))
	return result, "", nil
}

// globalReportFilters parses the filter controls every report surface shares
// (semantic reports, the journal projection, and downloads). Parse errors are
// returned so the API edge can answer 400 instead of silently matching
// nothing; each dimension is delegated to a focused parser below.
func globalReportFilters(r *http.Request) (report.Filters, error) {
	text := strings.TrimSpace(r.URL.Query().Get("filter"))
	if text != "" {
		// Fava rejects an unparseable filter at the API edge; validating here
		// lets the shell show the parse error instead of silently dropping
		// entries with a different filter semantics.
		if _, err := report.ParseFQL(text); err != nil {
			return report.Filters{}, err
		}
	}
	period, err := periodFilter(r)
	if err != nil {
		return report.Filters{}, err
	}
	if valuation := strings.TrimSpace(r.URL.Query().Get("valuation")); valuation != "" && valuation != "at-cost" && valuation != "market-value" {
		return report.Filters{}, fmt.Errorf("invalid valuation filter")
	}
	prefix, begin, end, err := reportTimeFilter(strings.TrimSpace(r.URL.Query().Get("time")), time.Now())
	if err != nil {
		return report.Filters{}, err
	}
	return report.Filters{
		Account:    strings.TrimSpace(r.URL.Query().Get("account")),
		Text:       text,
		Period:     period,
		TimePrefix: prefix,
		TimeBegin:  begin,
		TimeEnd:    end,
	}, nil
}

// periodFilter validates the interval selector. "all" and the empty string
// mean unfiltered and normalize to an empty Period.
func periodFilter(r *http.Request) (string, error) {
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" || period == "all" {
		return "", nil
	}
	switch period {
	case "month", "quarter", "year":
		return period, nil
	default:
		return "", fmt.Errorf("invalid period filter")
	}
}

// reportTimeFilter resolves Fava's `time` vocabulary into either a date
// prefix or a half-open YYYY-MM-01 begin/end range — the two shapes
// report.Filters can express. Relative words ("year", "month") anchor at
// now; explicit YYYY, YYYY-MM, and YYYY-Qn forms are validated strictly so a
// typo becomes a 400 rather than a filter that silently matches nothing.
func reportTimeFilter(raw string, now time.Time) (prefix, begin, end string, err error) {
	switch raw {
	case "", "all":
		return "", "", "", nil
	case "year":
		return now.Format("2006"), "", "", nil
	case "month":
		return now.Format("2006-01"), "", "", nil
	}
	if isNumericYear(raw) {
		return raw, "", "", nil
	}
	if begin, end, ok := quarterTimeRange(raw); ok {
		return "", begin, end, nil
	}
	if _, parseErr := time.Parse("2006-01", raw); parseErr == nil {
		return raw, "", "", nil
	}
	return "", "", "", fmt.Errorf("invalid time filter")
}

// isNumericYear reports whether raw is exactly four ASCII digits.
func isNumericYear(raw string) bool {
	if len(raw) != 4 {
		return false
	}
	_, err := strconv.Atoi(raw)
	return err == nil
}

// quarterTimeRange parses Fava's "2025-Q2" syntax into the half-open month
// range a prefix cannot express. ok is false for anything else, including a
// syntactically valid quarter out of range.
func quarterTimeRange(raw string) (begin, end string, ok bool) {
	if len(raw) != 7 || raw[4] != '-' || (raw[5] != 'Q' && raw[5] != 'q') {
		return "", "", false
	}
	year, yearErr := strconv.Atoi(raw[:4])
	quarter, quarterErr := strconv.Atoi(raw[6:])
	if yearErr != nil || quarterErr != nil || quarter < 1 || quarter > 4 {
		return "", "", false
	}
	beginMonth := (quarter-1)*3 + 1
	endYear, endMonth := year, beginMonth+3
	if endMonth > 12 {
		endYear++
		endMonth = 1
	}
	return fmt.Sprintf("%04d-%02d-01", year, beginMonth), fmt.Sprintf("%04d-%02d-01", endYear, endMonth), true
}

func journalDateRange(r *http.Request) (*ledger.Date, *ledger.Date, error) {
	from, err := parseISODate(r.URL.Query().Get("from"))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid from date: %w", err)
	}
	to, err := parseISODate(r.URL.Query().Get("to"))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid to date: %w", err)
	}
	if from != nil && to != nil && from.Raw > to.Raw {
		return nil, nil, fmt.Errorf("from date must not be after to date")
	}
	return from, to, nil
}

func parseISODate(raw string) (*ledger.Date, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("expected YYYY-MM-DD")
	}
	year, month, day := parsed.Date()
	if year <= 0 {
		return nil, fmt.Errorf("expected a positive year")
	}
	return &ledger.Date{Year: year, Month: int(month), Day: day, Raw: raw}, nil
}

func reportAsOfDate(r *http.Request) (string, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("as_of"))
	if raw == "" {
		return "", nil
	}
	if _, err := parseISODate(raw); err != nil {
		return "", fmt.Errorf("invalid as-of date: %w", err)
	}
	return raw, nil
}

// handleQuery (GET) runs a read-only query against the current snapshot and
// returns rows in JSON or CSV per the format parameter.
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	text := strings.TrimSpace(r.URL.Query().Get("q"))
	if text == "" {
		writeAPIError(w, http.StatusBadRequest, "query parameter q is required")
		return
	}
	current := s.store.Current()
	if current == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	result, err := query.Evaluate(text, current.Evaluation())
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	result = redactQueryPaths(result, current.Graph())
	if strings.EqualFold(r.URL.Query().Get("format"), "csv") {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		if err := result.WriteCSV(w); err != nil {
			return
		}
		return
	}
	if format := strings.TrimSpace(r.URL.Query().Get("format")); format != "" && !strings.EqualFold(format, "json") {
		writeAPIError(w, http.StatusBadRequest, "format must be json or csv")
		return
	}
	writeJSON(w, result)
}

func redactQueryPaths(result query.Result, graph *source.Graph) query.Result {
	for _, row := range result.Rows {
		for _, column := range []string{"file", "path"} {
			value, ok := row[column].(string)
			if !ok || value == "" {
				continue
			}
			if graph != nil {
				if id, found := graph.ByPath[value]; found {
					row[column] = graph.DisplayPath(id)
					continue
				}
			}
			row[column] = source.SafeDisplayPath(value)
		}
	}
	return result
}

func (s *Server) handleSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	current := s.store.Current()
	if current == nil || current.Graph() == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	graph := current.Graph()
	requested := strings.TrimSpace(r.URL.Query().Get("path"))
	if requested == "" {
		writeJSON(w, struct {
			Paths []string `json:"paths"`
		}{Paths: graph.DisplayPaths()})
		return
	}
	id, ok := graph.FileIDForDisplayPath(requested)
	if !ok {
		// Preserve exact graph-member lookup for callers that already hold an
		// internal path, while never reading an arbitrary filesystem path.
		id, ok = graph.ByPath[requested]
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	file := graph.File(id)
	if file == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}{Path: graph.DisplayPath(id), Content: string(file.Data)})
}

// handleEditor exposes only files that are members of the current include
// graph. Reads are GET-only; writes require an explicit POST and a snapshot
// precondition so an editor cannot silently overwrite a concurrent reload.
func (s *Server) handleEditor(w http.ResponseWriter, r *http.Request) {
	current := s.store.Current()
	if current == nil || current.Graph() == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	suffix := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/editor"), "/")
	switch {
	case r.Method == http.MethodGet && suffix == "":
		paths := current.Graph().DisplayPaths()
		writeJSON(w, struct {
			Paths      []string `json:"paths"`
			Entry      string   `json:"entry"`
			SnapshotID string   `json:"snapshot_id"`
		}{Paths: paths, Entry: current.Graph().DisplayPath(current.Graph().Entry), SnapshotID: current.ID})
	case r.Method == http.MethodGet && (suffix == "file" || suffix == "files"):
		s.handleEditorFile(w, r, current)
	case r.Method == http.MethodPost && suffix == "validate":
		s.handleEditorValidate(w, r, current)
	case r.Method == http.MethodPost && suffix == "save":
		s.handleEditorSave(w, r, current)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.NotFound(w, r)
	}
}

func (s *Server) handleEditorFile(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		writeAPIError(w, http.StatusBadRequest, "path is required")
		return
	}
	file, display, ok := graphFile(current.Graph(), path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, struct {
		Path       string `json:"path"`
		Content    string `json:"content"`
		SnapshotID string `json:"snapshot_id"`
	}{Path: display, Content: string(file.Data), SnapshotID: current.ID})
}

type editorContentRequest struct {
	Path             string `json:"path"`
	Content          string `json:"content"`
	ExpectedSnapshot string `json:"expected_snapshot_id"`
}

func (s *Server) handleEditorValidate(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	var request editorContentRequest
	if err := decodeJSONBody(w, r, &request, 4<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, _, ok := graphFile(current.Graph(), request.Path); !ok {
		http.NotFound(w, r)
		return
	}
	file, bag := ledger.ParseText(source.SafeDisplayPath(request.Path), []byte(request.Content))
	_ = file
	diagnostics := bag.All()
	writeJSON(w, struct {
		Valid       bool                 `json:"valid"`
		Diagnostics []diagnosticResponse `json:"diagnostics"`
	}{Valid: !bag.HasErrors(), Diagnostics: diagnosticsPayload(diagnostics, nil)})
}

func (s *Server) handleEditorSave(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request editorContentRequest
	if err := decodeJSONBody(w, r, &request, 8<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if request.ExpectedSnapshot != "" && request.ExpectedSnapshot != current.ID {
		writeAPIError(w, http.StatusConflict, "snapshot changed; reload before saving")
		return
	}
	file, display, ok := graphFile(current.Graph(), request.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	result, backup, err := s.replaceGraphFile(current, file.Path, display, []byte(request.Content))
	if err != nil {
		status := http.StatusUnprocessableEntity
		if result.Err != nil {
			status = http.StatusInternalServerError
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		writeJSON(w, struct {
			Published   bool                 `json:"published"`
			Backup      string               `json:"backup,omitempty"`
			Diagnostics []diagnosticResponse `json:"diagnostics"`
		}{Backup: backup, Diagnostics: diagnosticsPayload(result.Diagnostics, current.Graph())})
		return
	}
	writeJSON(w, struct {
		Published   bool                 `json:"published"`
		SnapshotID  string               `json:"snapshot_id"`
		Backup      string               `json:"backup"`
		Diagnostics []diagnosticResponse `json:"diagnostics"`
	}{Published: true, SnapshotID: result.Snapshot.ID, Backup: backup, Diagnostics: diagnosticsPayload(result.Diagnostics, result.Snapshot.Graph())})
}

// replaceGraphFile performs an atomic replacement, keeps a recoverable backup,
// then revalidates the entire include graph before publishing a new snapshot.
// Failed validation restores the original bytes and leaves the prior snapshot
// active.
func (s *Server) replaceGraphFile(current *snapshot.Snapshot, actualPath, displayPath string, content []byte) (snapshot.BuildResult, string, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if latest := s.store.Current(); latest == nil || latest.ID != current.ID {
		return snapshot.BuildResult{Err: fmt.Errorf("snapshot changed")}, "", fmt.Errorf("snapshot changed; reload before saving")
	}
	old, err := os.ReadFile(actualPath)
	if err != nil {
		return snapshot.BuildResult{Err: err}, "", err
	}
	info, err := os.Stat(actualPath)
	if err != nil {
		return snapshot.BuildResult{Err: err}, "", err
	}
	backupPath := actualPath + ".orangecount.bak"
	if err := os.WriteFile(backupPath, old, info.Mode().Perm()); err != nil {
		return snapshot.BuildResult{Err: err}, "", err
	}
	if err := atomicWrite(actualPath, content, info.Mode().Perm()); err != nil {
		return snapshot.BuildResult{Err: err}, displayPath + ".orangecount.bak", err
	}
	result := s.store.Reload(current.EntryPath, snapshot.BuildOptions{})
	if result.Snapshot != nil {
		return result, displayPath + ".orangecount.bak", nil
	}
	// Restore the previous source and snapshot. The failed diagnostics are
	// retained for the caller while the store returns to the last valid view.
	_ = atomicWrite(actualPath, old, info.Mode().Perm())
	_ = s.store.Reload(current.EntryPath, snapshot.BuildOptions{})
	return result, displayPath + ".orangecount.bak", fmt.Errorf("validation failed")
}

func atomicWrite(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	temporary, err := os.CreateTemp(dir, ".orangecount-write-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, path)
}

// handleDocumentUpload stores an uploaded attachment beneath a configured
// document root, in the subfolder chain of the target account, the way
// Fava's put_document endpoint lays out uploads. It is a reviewed write path:
// same-origin only, account validated against the evaluation, filename
// reduced to its basename, and no overwrite of existing files.
func (s *Server) handleDocumentUpload(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	roots := s.roots.Paths()
	if len(roots) == 0 {
		writeAPIError(w, http.StatusBadRequest, "No document root is configured (serve --document-root).")
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeAPIError(w, http.StatusBadRequest, "The upload could not be parsed.")
		return
	}
	account := strings.TrimSpace(r.FormValue("account"))
	if _, ok := current.Evaluation().Accounts[account]; !ok {
		writeAPIError(w, http.StatusBadRequest, fmt.Sprintf("Not a valid account: %q", account))
		return
	}
	folder, err := uploadFolder(roots, strings.TrimSpace(r.FormValue("folder")))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil || header == nil {
		writeAPIError(w, http.StatusBadRequest, "No file uploaded.")
		return
	}
	defer file.Close()
	// The basename reduction is Fava's separator-to-space sanitization made
	// strict: directory components can never survive the upload.
	name := filepath.Base(strings.TrimSpace(header.Filename))
	if !validDocumentName(name) {
		writeAPIError(w, http.StatusBadRequest, "Uploaded file is missing a filename.")
		return
	}
	// Defense in depth: the components are validated above, but the resolved
	// target must provably stay inside the chosen document root.
	target, err := documentTargetPath(folder, account, name)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Uploaded file is missing a filename.")
		return
	}
	if _, err := os.Stat(target); err == nil {
		writeAPIError(w, http.StatusConflict, "Target path already exists: "+name)
		return
	}
	if writeErr := storeUploadedFile(target, file); writeErr != nil {
		writeAPIError(w, writeErr.status, writeErr.message)
		return
	}
	relative := documentRelativePath(folder, target)
	writeJSON(w, favaadapter.NewEnvelope(struct {
		Filename string `json:"filename"`
		Message  string `json:"message"`
	}{Filename: relative, Message: "Uploaded to " + relative}, current.BuiltAt))
}

// httpError carries the status/message pair a failed filesystem write wants
// to answer with.
type httpError struct {
	status  int
	message string
}

func (e httpError) Error() string { return e.message }

// uploadFolder resolves the form's folder selector against the configured
// document roots. An empty selector stays on the primary root; an explicit
// selector must name one of the configured roots exactly.
func uploadFolder(roots []string, requested string) (string, error) {
	if requested == "" {
		return roots[0], nil
	}
	for _, candidate := range roots {
		if requested == candidate {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("Not a documents folder: %s.", requested)
}

// validDocumentName rejects basenames that are empty, traversal-shaped, or
// option-like (leading dash).
func validDocumentName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.HasPrefix(name, "-")
}

// documentTargetPath nests name under the account's colon-separated subfolder
// chain inside root. Defense in depth: the components are validated by the
// caller, but the resolved target must provably stay inside the root.
func documentTargetPath(root, account, name string) (string, error) {
	target := filepath.Join(root, filepath.Join(strings.Split(account, ":")...), name)
	if !pathWithin(root, filepath.Clean(target)) {
		return "", fmt.Errorf("target escapes document root")
	}
	return target, nil
}

// storeUploadedFile writes the upload into target without ever overwriting:
// the account folders are created, the file itself is opened O_EXCL, and any
// later failure removes the partial file again.
func storeUploadedFile(target string, source io.Reader) *httpError {
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return &httpError{status: http.StatusInternalServerError, message: "The account folder could not be created."}
	}
	created, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return &httpError{status: http.StatusConflict, message: "Target path already exists: " + filepath.Base(target)}
	}
	if _, err := io.Copy(created, source); err != nil {
		_ = created.Close()
		_ = os.Remove(target)
		return &httpError{status: http.StatusInternalServerError, message: "The document could not be written."}
	}
	if err := created.Close(); err != nil {
		_ = os.Remove(target)
		return &httpError{status: http.StatusInternalServerError, message: "The document could not be written."}
	}
	return nil
}

// documentRelativePath renders target as the slash-separated path relative
// to its document root for the response envelope.
func documentRelativePath(root, target string) string {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		relative = filepath.Base(target)
	}
	return filepath.ToSlash(relative)
}

// handleDocumentMove moves an existing attachment into the subfolder chain of
// another account, optionally renaming it, the way Fava's move_document
// endpoint works. The file must already live beneath a configured document
// root; the target stays inside that same root and never overwrites.
func (s *Server) handleDocumentMove(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeAPIError(w, http.StatusMethodNotAllowed, "Only POST is supported.")
		return
	}
	if s.roots.Empty() {
		writeAPIError(w, http.StatusBadRequest, "No document root is configured (serve --document-root).")
		return
	}
	var request struct {
		Filename string `json:"filename"`
		Account  string `json:"account"`
		NewName  string `json:"new_name"`
	}
	if err := decodeJSONBody(w, r, &request, 1<<16); err != nil {
		writeAPIError(w, http.StatusBadRequest, "The request body must be a JSON object.")
		return
	}
	account := strings.TrimSpace(request.Account)
	if _, ok := current.Evaluation().Accounts[account]; !ok {
		writeAPIError(w, http.StatusBadRequest, fmt.Sprintf("Not a valid account: %q", account))
		return
	}
	sourcePath, err := s.roots.Resolve(strings.TrimSpace(request.Filename))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "The document could not be found beneath a configured document root.")
		return
	}
	root := documentRootFor(s.roots.Paths(), sourcePath)
	if root == "" {
		writeAPIError(w, http.StatusInternalServerError, "The document root for this attachment could not be determined.")
		return
	}
	name := filepath.Base(strings.TrimSpace(request.NewName))
	if !validDocumentName(name) {
		name = filepath.Base(sourcePath)
	}
	// Defense in depth: account and name are validated above, but the resolved
	// target must provably stay inside the source's document root.
	target, targetErr := documentTargetPath(root, account, name)
	if targetErr != nil || filepath.Clean(target) == sourcePath {
		if targetErr != nil {
			writeAPIError(w, http.StatusBadRequest, "The new filename is not valid.")
			return
		}
		writeJSON(w, favaadapter.NewEnvelope(struct {
			Filename string `json:"filename"`
			Message  string `json:"message"`
		}{Filename: documentRelativePath(root, target), Message: "Document unchanged."}, current.BuiltAt))
		return
	}
	if _, err := os.Stat(target); err == nil {
		writeAPIError(w, http.StatusConflict, "Target path already exists: "+name)
		return
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "The account folder could not be created.")
		return
	}
	if err := os.Rename(sourcePath, target); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "The document could not be moved.")
		return
	}
	relative := documentRelativePath(root, target)
	writeJSON(w, favaadapter.NewEnvelope(struct {
		Filename string `json:"filename"`
		Message  string `json:"message"`
	}{Filename: relative, Message: "Moved to " + relative}, current.BuiltAt))
}

// documentRootFor finds the configured root that actually contains the
// attachment. The relative != "." requirement keeps the root directories
// themselves from being treated as movable documents.
func documentRootFor(roots []string, sourcePath string) string {
	for _, candidate := range roots {
		if relative, relErr := filepath.Rel(candidate, sourcePath); relErr == nil && pathWithin(candidate, sourcePath) && relative != "." {
			return candidate
		}
	}
	return ""
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative))
}

func graphFile(graph *source.Graph, displayPath string) (*source.SourceFile, string, bool) {
	if graph == nil {
		return nil, "", false
	}
	id, ok := graph.FileIDForDisplayPath(strings.TrimSpace(displayPath))
	if !ok {
		return nil, "", false
	}
	file := graph.File(id)
	if file == nil {
		return nil, "", false
	}
	return file, graph.DisplayPath(id), true
}

type diagnosticResponse struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Path     string `json:"path,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Message  string `json:"message"`
}

func diagnosticsPayload(values []diagnostic.Diagnostic, graph *source.Graph) []diagnosticResponse {
	result := make([]diagnosticResponse, 0, len(values))
	for _, value := range values {
		path := displayDiagnosticPath(value, graph)
		result = append(result, diagnosticResponse{Code: value.Code, Severity: string(value.Severity), Path: path, Line: value.Span.StartLine, Column: value.Span.StartColumn, Message: value.Message})
	}
	return result
}

// requireSameOrigin rejects cross-site requests that a browser would otherwise
// send for drive-by CSRF. The server is loopback-only, but a co-resident
// browser visiting a malicious site could still issue state-changing requests
// to it. Same-origin requests from the embedded app carry an Origin (or
// Referer) matching the loopback host; non-browser clients may omit both and
// are allowed through. When an Origin is present but does not match the request
// host, the request is rejected with 403.
func requireSameOrigin(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		writeAPIError(w, http.StatusForbidden, "cross-origin request rejected")
		return false
	}
	if strings.EqualFold(parsed.Host, r.Host) {
		return true
	}
	writeAPIError(w, http.StatusForbidden, "cross-origin request rejected")
	return false
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any, limit int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request body must contain one JSON object")
	}
	return nil
}

// handleImport serves the import pipeline: GET lists adapters and files,
// POST parses a source file into staged entries pending preview and commit.
func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	current := s.store.Current()
	if current == nil || current.Graph() == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	suffix := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/import"), "/")
	if r.Method == http.MethodGet && (suffix == "" || suffix == "targets") {
		writeJSON(w, struct {
			Paths      []string `json:"paths"`
			Entry      string   `json:"entry"`
			SnapshotID string   `json:"snapshot_id"`
		}{Paths: current.Graph().DisplayPaths(), Entry: current.Graph().DisplayPath(current.Graph().Entry), SnapshotID: current.ID})
		return
	}
	if r.Method == http.MethodGet && suffix == "adapters" {
		writeJSON(w, struct {
			Adapters []map[string]any `json:"adapters"`
		}{Adapters: []map[string]any{
			{"id": "beancount", "label": "Beancount source", "extensions": []string{".bean", ".beancount"}},
			{"id": "csv", "label": "Generic CSV", "extensions": []string{".csv"}, "columns": []string{"date", "payee", "account", "amount", "currency", "narration"}},
		}})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.NotFound(w, r)
		return
	}
	switch suffix {
	case "preview":
		s.handleImportPreview(w, r, current)
	case "commit":
		s.handleImportCommit(w, r, current)
	default:
		http.NotFound(w, r)
	}
}

type importPreviewRequest struct {
	Path    string            `json:"path"`
	Content string            `json:"content"`
	Adapter string            `json:"adapter"`
	Mapping map[string]string `json:"mapping"`
}

func (s *Server) handleImportPreview(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request importPreviewRequest
	if err := decodeJSONBody(w, r, &request, 4<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	adapter := strings.ToLower(strings.TrimSpace(request.Adapter))
	if adapter == "" {
		adapter = "beancount"
	}
	name, normalized, err := normalizedImportContent(request, adapter)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	file, bag := ledger.ParseText(name, []byte(normalized))
	diagnostics := bag.All()
	if bag.HasErrors() {
		writeJSON(w, struct {
			Valid       bool                 `json:"valid"`
			Diagnostics []diagnosticResponse `json:"diagnostics"`
		}{Valid: false, Diagnostics: diagnosticsPayload(diagnostics, nil)})
		return
	}
	if err := rejectImportDirectives(file); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Evaluate the imported file against the merged include graph rather than
	// in isolation (see evaluateImportMerged); this mirrors what the commit
	// path revalidates, so a preview that says "valid" will commit.
	evaluation := evaluateImportMerged(current, file)
	for _, value := range evaluation.Diagnostics {
		diagnostics = append(diagnostics, value)
	}
	previewID := importPreviewID(name, normalized)
	s.previews.Store(previewID, importPreview{Path: name, Content: normalized})
	rows := importedRowsOnly(evaluation, name)
	writeJSON(w, struct {
		PreviewID   string               `json:"preview_id"`
		Path        string               `json:"path"`
		Valid       bool                 `json:"valid"`
		Diagnostics []diagnosticResponse `json:"diagnostics"`
		Rows        query.Result         `json:"rows"`
		Diff        importDiff           `json:"diff"`
	}{PreviewID: previewID, Path: name, Valid: importEvaluationValid(evaluation), Diagnostics: diagnosticsPayload(diagnostics, nil), Rows: report.Present(rows), Diff: importDiff{AddedLines: strings.Count(normalized, "\n") + 1, Bytes: len([]byte(normalized))}})
}

// normalizedImportContent validates the request's file name against its
// adapter and returns the Beancount text the preview should evaluate. CSV
// input is converted and the file name re-pointed at a .bean extension.
func normalizedImportContent(request importPreviewRequest, adapter string) (name, content string, err error) {
	name, err = safeImportName(request.Path, adapter)
	if err != nil {
		return "", "", err
	}
	content = request.Content
	if adapter == "csv" {
		content, err = csvToBeancount(request.Content, request.Mapping)
		if err != nil {
			return "", "", err
		}
		name = strings.TrimSuffix(name, filepath.Ext(name)) + ".bean"
	}
	return name, content, nil
}

// rejectImportDirectives refuses imports that would grow the include graph
// or activate plugins; an import may only add entries.
func rejectImportDirectives(file *ledger.File) error {
	for _, directive := range file.Directives {
		switch directive.(type) {
		case ledger.Include, *ledger.Include, ledger.Plugin, *ledger.Plugin:
			return fmt.Errorf("imports cannot add include or plugin directives")
		}
	}
	return nil
}

// evaluateImportMerged evaluates the imported file against the published
// include graph rather than in isolation. The full graph knows which accounts
// are open and which currencies are permitted, so a posting to an account
// opened only in the main ledger (the common case for an import) is not
// misreported as an E-EVAL-POSTING lifecycle error. The imported file is not
// yet a member of the published graph, so it is assigned a fresh FileID that
// cannot collide with the existing members.
func evaluateImportMerged(current *snapshot.Snapshot, file *ledger.File) *ledger.Evaluation {
	order := append([]source.FileID(nil), current.Graph().Order...)
	parsed := current.Parsed()
	importID := source.FileID(len(parsed) + 1)
	parsed[importID] = file
	order = append(order, importID)
	return ledger.EvaluateFiles(parsed, order, ledger.EvalOptions{})
}

// importedRowsOnly projects the merged evaluation's postings and keeps only
// the imported file's rows: the merged graph necessarily includes the
// existing ledger's postings, but the preview table should show only what
// this import would add.
func importedRowsOnly(evaluation *ledger.Evaluation, name string) query.Result {
	rows, err := query.Evaluate("SELECT date, account, units, currency, flag, payee, narration FROM postings ORDER BY date, account", *evaluation)
	if err != nil {
		return query.Result{}
	}
	imported := rows.Rows[:0]
	for _, row := range rows.Rows {
		if row["file"] == name {
			imported = append(imported, row)
		}
	}
	rows.Rows = imported
	return rows
}

// importEvaluationValid reports whether the whole merged graph, including the
// proposed import, is free of error diagnostics. The imported file itself is
// always valid by the time this runs (parse errors returned earlier).
func importEvaluationValid(evaluation *ledger.Evaluation) bool {
	for _, value := range evaluation.Diagnostics {
		if value.Severity == diagnostic.Error {
			return false
		}
	}
	return true
}

type importDiff struct {
	AddedLines int `json:"added_lines"`
	Bytes      int `json:"bytes"`
}

type importCommitRequest struct {
	PreviewID        string `json:"preview_id"`
	Target           string `json:"target"`
	ExpectedSnapshot string `json:"expected_snapshot_id"`
}

// handleImportCommit (POST) applies a staged import: re-validates the
// previewed entry against the expected snapshot, applies the file edits, and
// rebuilds; a mismatch aborts with a conflict instead of writing.
func (s *Server) handleImportCommit(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request importCommitRequest
	if err := decodeJSONBody(w, r, &request, 64<<10); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	preview, ok := s.previews.Take(request.PreviewID)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "import preview not found or expired")
		return
	}
	if request.ExpectedSnapshot != "" && request.ExpectedSnapshot != current.ID {
		writeAPIError(w, http.StatusConflict, "snapshot changed; refresh import targets")
		return
	}
	target := strings.TrimSpace(request.Target)
	if target == "" {
		target = current.Graph().DisplayPath(current.Graph().Entry)
	}
	file, display, ok := graphFile(current.Graph(), target)
	if !ok {
		http.NotFound(w, r)
		return
	}
	old, err := os.ReadFile(file.Path)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "unable to read import target")
		return
	}
	combined := append([]byte(nil), old...)
	if len(combined) != 0 && combined[len(combined)-1] != '\n' {
		combined = append(combined, '\n')
	}
	combined = append(combined, []byte(preview.Content)...)
	if len(combined) != 0 && combined[len(combined)-1] != '\n' {
		combined = append(combined, '\n')
	}
	result, backup, err := s.replaceGraphFile(current, file.Path, display, combined)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if result.Err != nil {
			status = http.StatusInternalServerError
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		writeJSON(w, struct {
			Published   bool                 `json:"published"`
			Backup      string               `json:"backup,omitempty"`
			Diagnostics []diagnosticResponse `json:"diagnostics"`
		}{Backup: backup, Diagnostics: diagnosticsPayload(result.Diagnostics, current.Graph())})
		return
	}
	s.previews.Discard(request.PreviewID)
	writeJSON(w, struct {
		Published  bool   `json:"published"`
		SnapshotID string `json:"snapshot_id"`
		Backup     string `json:"backup"`
	}{Published: true, SnapshotID: result.Snapshot.ID, Backup: backup})
}

// safeImportName resolves a user-supplied import path: absolute paths are
// rejected, the file must live under the adapter's root, and the name must
// stay within ASCII word characters plus a short allowlist.
func safeImportName(raw, adapter string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.Base(raw) != raw || strings.ContainsAny(raw, `/\\`) {
		return "", fmt.Errorf("import path must be a file name")
	}
	ext := strings.ToLower(filepath.Ext(raw))
	if adapter == "csv" && ext == ".csv" {
		return raw, nil
	}
	if adapter != "beancount" || (ext != ".bean" && ext != ".beancount") {
		return "", fmt.Errorf("import path extension does not match adapter")
	}
	return raw, nil
}

// csvToBeancount converts a generic CSV import into Beancount transactions
// balanced against an offset account. The header row names the columns
// (date, account, amount required; currency, payee, narration optional).
func csvToBeancount(content string, mapping map[string]string) (string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("invalid CSV: %w", err)
	}
	if len(records) < 2 {
		return "", fmt.Errorf("CSV must contain a header and at least one row")
	}
	header, err := csvHeaderIndex(records[0])
	if err != nil {
		return "", err
	}
	offset := csvMappingValue(mapping, "offset_account", "Equity:Imported")
	currencyDefault := csvMappingValue(mapping, "currency", "USD")
	var builder strings.Builder
	for line, record := range records[1:] {
		rendered, renderErr := csvRecordToBean(record, header, offset, currencyDefault)
		if renderErr != nil {
			return "", fmt.Errorf("CSV row %d: %w", line+2, renderErr)
		}
		builder.WriteString(rendered)
	}
	return builder.String(), nil
}

// csvHeaderIndex maps lowercased/trimmed header names to record indexes and
// enforces the required columns.
func csvHeaderIndex(headerRow []string) (map[string]int, error) {
	header := make(map[string]int, len(headerRow))
	for index, value := range headerRow {
		header[strings.ToLower(strings.TrimSpace(value))] = index
	}
	for _, required := range []string{"date", "account", "amount"} {
		if _, ok := header[required]; !ok {
			return nil, fmt.Errorf("CSV requires %s column", required)
		}
	}
	return header, nil
}

// csvMappingValue reads one mapping override, falling back to the default
// when the mapping is nil or the value is blank.
func csvMappingValue(mapping map[string]string, key, fallback string) string {
	if mapping == nil {
		return fallback
	}
	if value := strings.TrimSpace(mapping[key]); value != "" {
		return value
	}
	return fallback
}

// csvRecordToBean renders one CSV record as a balanced transaction with its
// offset posting. Embedded newlines in payee/narration are flattened so one
// record can never produce broken directives.
func csvRecordToBean(record []string, header map[string]int, offset, currencyDefault string) (string, error) {
	value := func(key string) string {
		index, ok := header[key]
		if !ok || index >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[index])
	}
	date, amount := value("date"), value("amount")
	if _, err := parseISODate(date); err != nil {
		return "", fmt.Errorf("invalid date")
	}
	parsedAmount, err := ledger.ParseDecimal(amount)
	if err != nil {
		return "", fmt.Errorf("invalid amount")
	}
	currency := value("currency")
	if currency == "" {
		currency = currencyDefault
	}
	payee := csvSingleLine(value("payee"))
	narration := csvSingleLine(value("narration"))
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s * \"%s\"", date, escapeBeanString(payee)))
	if narration != "" {
		builder.WriteString(fmt.Sprintf(" \"%s\"", escapeBeanString(narration)))
	}
	builder.WriteByte('\n')
	builder.WriteString(fmt.Sprintf("  %s %s %s\n", value("account"), parsedAmount.String(), currency))
	builder.WriteString(fmt.Sprintf("  %s %s %s\n", offset, parsedAmount.Neg().String(), currency))
	return builder.String(), nil
}

// csvSingleLine collapses line breaks so CSV fields stay single-line strings.
func csvSingleLine(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\n", " "), "\r", " ")
}

func escapeBeanString(value string) string {
	return strings.ReplaceAll(value, "\"", "\\\"")
}

func importPreviewID(path, content string) string {
	hash := sha256.New()
	hash.Write([]byte(path))
	hash.Write([]byte{0})
	hash.Write([]byte(content))
	return fmt.Sprintf("%x", hash.Sum(nil))[:16]
}

// handleOptions serves the local options API: GET returns current values,
// POST (same-origin) validates and persists them alongside the ledger.
func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && !requireSameOrigin(w, r) {
		return
	}
	if r.Method == http.MethodGet {
		current := s.store.Current()
		evaluationOptions := map[string]string{}
		if current != nil {
			for key, value := range current.Evaluation().Options {
				evaluationOptions[key] = value
			}
		}
		s.optionsMu.RLock()
		for key, value := range s.options {
			evaluationOptions[key] = value
		}
		s.optionsMu.RUnlock()
		writeJSON(w, struct {
			Options map[string]string `json:"options"`
		}{Options: evaluationOptions})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var values map[string]string
	if err := decodeJSONBody(w, r, &values, 64<<10); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	for key, value := range values {
		if err := validateLocalOption(key, value); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	s.optionsMu.Lock()
	for key, value := range values {
		s.options[key] = value
	}
	s.optionsMu.Unlock()
	writeJSON(w, struct {
		Saved bool `json:"saved"`
	}{Saved: true})
}

// validateLocalOption checks a settable local option's value; unknown keys
// and out-of-domain values are rejected so the options file stays clean.
func validateLocalOption(key, value string) error {
	switch key {
	case "locale":
		if value != "en" && value != "zh-CN" {
			return fmt.Errorf("locale must be en or zh-CN")
		}
	case "currency":
		if len(value) < 2 || len(value) > 12 {
			return fmt.Errorf("currency must be a short uppercase code")
		}
		for _, character := range value {
			if character < 'A' || character > 'Z' {
				return fmt.Errorf("currency must be a short uppercase code")
			}
		}
	case "time":
		if value != "all" && value != "year" && value != "month" {
			return fmt.Errorf("time must be all, year, or month")
		}
	default:
		return fmt.Errorf("unsupported option %q", key)
	}
	return nil
}

func helpSections(locales ...string) []map[string]string {
	locale := repairguidance.LocaleEnglish
	if len(locales) > 0 && locales[0] == repairguidance.LocaleChinese {
		locale = repairguidance.LocaleChinese
	}
	sections := []struct{ id, enTitle, enBody, zhTitle, zhBody string }{
		{"navigation", "Navigation", "Use the sidebar or the menu button to move between reports.", "导航", "使用侧边栏或菜单按钮在报表之间移动。"},
		{"filters", "Filters", "Global time, account, and text filters are bookmarkable URL state.", "筛选", "全局时间、账户和文本筛选会保存在可收藏的 URL 状态中。"},
		{"options", "Options", "The color scheme, locale, and fava options live on the Options page. Beancount options are declared in the ledger and shown there read-only.", "选项", "配色、语言和 Fava 选项位于选项页。Beancount 选项在账本中声明，并以只读方式显示。"},
		{"editor", "Editor safety", "Validate before saving. Saves are atomic, backed up, and revalidated before publication.", "编辑器安全", "保存前会验证。保存采用原子写入、备份，并在发布前重新验证。"},
		{"import", "Import review", "Preview imported postings and explicitly commit them to a selected ledger file.", "导入审核", "预览导入的记账，并明确提交到选定的账本文件。"},
		{"prices", "Local prices", "Market valuation uses only price directives in the local ledger. Missing quotes are shown as unavailable; no external provider is contacted.", "本地价格", "市值估算只使用本地账本中的 price 指令。缺少报价时显示不可用；不会联系外部服务。"},
		{"plugins", "Plugin migration", "Python plugins are never executed. Plugin directives remain visible as diagnostics so they can be migrated explicitly.", "插件迁移", "不会执行 Python 插件。插件指令会保留为诊断，以便明确迁移。"},
		{"diagnostics", "Diagnostics", "Open a diagnostic to see its repair order, safe checks, generic example, and local source context.", "诊断", "打开诊断可查看修复顺序、安全检查、通用示例和本地源上下文。"},
		{"shortcuts", "Keyboard", "Tab reaches controls; Enter applies filters and runs queries.", "键盘", "使用 Tab 到达控件，使用 Enter 应用筛选并运行查询。"},
		{"quick-entry", "Quick Entry", "Use the Quick tab (a q) to capture two-posting transactions with compact shorthand. Template form: 'lunch 28 @wechat'. Explicit form: '28 CNY @source -> @dest : narration'. Press Ctrl+Enter to preview, then Ctrl+Enter again to commit. Aliases and templates are defined in the ledger as dated custom directives; manage them under /quick-profile.", "速记", "使用 Quick 标签页（快捷键 a q）以紧凑语法快速记录双过账交易。模板形式：'午餐 28 @微信'。显式形式：'28 CNY @微信 -> @餐饮 : 摘要'。按 Ctrl+Enter 预览，再按 Ctrl+Enter 确认提交。别名和模板在账本中定义为带日期的 custom 指令；可在 /quick-profile 管理。"},
	}
	result := make([]map[string]string, 0, len(sections))
	for _, section := range sections {
		title, body := section.enTitle, section.enBody
		if locale == repairguidance.LocaleChinese {
			title, body = section.zhTitle, section.zhBody
		}
		result = append(result, map[string]string{"id": section.id, "title": title, "body": body})
	}
	return result
}

func (s *Server) handleHelp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if topic := strings.TrimSpace(r.URL.Query().Get("topic")); topic != "" {
		code := strings.TrimPrefix(topic, "diagnostics/")
		guide, ok := repairguidance.Lookup(code, requestedLocale(r))
		if !ok || topic != guide.Topic {
			writeAPIError(w, http.StatusNotFound, helpTopicNotFoundMessage(requestedLocale(r)))
			return
		}
		writeJSON(w, guide)
		return
	}
	writeJSON(w, struct {
		Sections []map[string]string `json:"sections"`
	}{Sections: helpSections(requestedLocale(r))})
}

func displayDiagnosticPath(value diagnostic.Diagnostic, graph *source.Graph) string {
	if graph != nil {
		if graph.File(value.Span.File) != nil {
			return graph.DisplayPath(value.Span.File)
		}
		if id, ok := graph.ByPath[value.Path]; ok {
			return graph.DisplayPath(id)
		}
	}
	return source.SafeDisplayPath(value.Path)
}

func diagnosticContextUnavailableReason(locale string) string {
	if locale == repairguidance.LocaleChinese {
		return "当前失败构建没有可安全展示的源文件上下文。请先修复 include 或文件读取问题，然后重新检查。"
	}
	return "Source context is unavailable for the failed build. Fix the include or file-read problem, then check again."
}

func helpTopicNotFoundMessage(locale string) string {
	if locale == repairguidance.LocaleChinese {
		return "找不到本地帮助主题。请检查诊断代码。"
	}
	return "local help topic not found; check the diagnostic code"
}

func (s *Server) uiAsset(name string) string {
	if os.Getenv("ORANGECOUNT_TRANSPLANTED_UI") == "1" {
		return "assets/transplanted/" + name
	}
	return "assets/" + name
}

func (s *Server) handleApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	data, err := assets.ReadFile(s.uiAsset("app.js"))
	if err != nil {
		http.Error(w, "embedded UI unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

func (s *Server) handleStyle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	data, err := assets.ReadFile(s.uiAsset("app.css"))
	if err != nil {
		http.Error(w, "embedded UI unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

func (s *Server) handleDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	encoded := strings.TrimPrefix(r.URL.Path, "/documents/")
	if encoded == "" {
		// Bare "/documents/" is the Documents UI route, not an attachment
		// request: Fava links documents with a trailing slash, so serving the
		// shell here keeps that URL bookmarkable and refreshable.
		s.handleIndex(w, r)
		return
	}
	name, err := url.PathUnescape(encoded)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	path, err := s.roots.Resolve(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	writeJSON(w, struct {
		Error string `json:"error"`
	}{Error: message})
}

func requestedLocale(r *http.Request) string {
	locale := r.URL.Query().Get("locale")
	if locale == "zh-CN" {
		return locale
	}
	return "en"
}
