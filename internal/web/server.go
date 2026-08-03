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
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

// assets contains the compiled, dependency-free browser bundle. Keeping the
// files inside this package makes the released binary independent of Node or
// a runtime CDN.
//
//go:embed assets/*
var assets embed.FS

type Config struct {
	Store         *snapshot.Store
	DocumentRoots source.DocumentRoots
	Addr          string
}

type Server struct {
	mu        sync.RWMutex
	writeMu   sync.Mutex
	store     *snapshot.Store
	roots     source.DocumentRoots
	addr      string
	http      *http.Server
	bound     string
	ready     chan struct{}
	readyOnce sync.Once
	optionsMu sync.RWMutex
	options   map[string]string
	pendingMu sync.Mutex
	pending   map[string]importPreview
}

type importPreview struct {
	Path    string
	Content string
	expires int64
}

// importPreviewTTL bounds how long a preview may wait before being committed.
// Previews are anonymous server-side state; an abandoned preview must not
// accumulate forever in memory.
const importPreviewTTL = 30 * time.Minute

// maxImportPreviews caps the number of previews retained simultaneously. A
// client that repeatedly previews hash-distinct content cannot grow server
// memory without bound.
const maxImportPreviews = 32

func (p importPreview) live(unix int64) bool { return p.expires > unix }

func (s *Server) storePreview(id string, preview importPreview) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	now := time.Now().Unix()
	for id, existing := range s.pending {
		if !existing.live(now) {
			delete(s.pending, id)
		}
	}
	preview.expires = now + int64(importPreviewTTL/time.Second)
	if len(s.pending) >= maxImportPreviews {
		// Overflow: drop the oldest live preview to make room, keeping the
		// most recent previews usable. Idempotent previews share an id.
		var oldestID string
		var oldest int64
		for id, existing := range s.pending {
			if oldestID == "" || existing.expires < oldest {
				oldestID, oldest = id, existing.expires
			}
		}
		delete(s.pending, oldestID)
	}
	s.pending[id] = preview
}

func (s *Server) takePreview(id string) (importPreview, bool) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	preview, ok := s.pending[id]
	if !ok {
		return importPreview{}, false
	}
	if !preview.live(time.Now().Unix()) {
		delete(s.pending, id)
		return importPreview{}, false
	}
	return preview, true
}

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
	server := &Server{store: config.Store, roots: config.DocumentRoots, addr: addr, ready: make(chan struct{}), options: make(map[string]string), pending: make(map[string]importPreview)}
	server.http = &http.Server{Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second}
	return server, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/diagnostics", s.handleDiagnostics)
	mux.HandleFunc("/api/v1/query", s.handleQuery)
	mux.HandleFunc("/api/v1/source", s.handleSource)
	mux.HandleFunc("/api/v1/editor", s.handleEditor)
	mux.HandleFunc("/api/v1/editor/", s.handleEditor)
	mux.HandleFunc("/api/v1/import", s.handleImport)
	mux.HandleFunc("/api/v1/import/", s.handleImport)
	mux.HandleFunc("/api/v1/options", s.handleOptions)
	mux.HandleFunc("/api/v1/help", s.handleHelp)
	mux.HandleFunc("/api/v1/reports/", s.handleReport)
	// Reserve the Fava-style Documents UI route separately from attachment
	// paths (`/documents/<name>`), otherwise ServeMux canonicalizes `/documents`
	// to the attachment handler before the embedded shell can load.
	mux.HandleFunc("/documents", s.handleIndex)
	mux.HandleFunc("/documents/", s.handleDocument)
	mux.HandleFunc("/app.js", s.handleApp)
	return mux
}

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
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) Serve(ctx context.Context) error {
	if s == nil || s.http == nil {
		return fmt.Errorf("nil web server")
	}
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.readyOnce.Do(func() { close(s.ready) })
		return err
	}
	s.mu.Lock()
	s.bound = listener.Addr().String()
	s.mu.Unlock()
	s.readyOnce.Do(func() { close(s.ready) })
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
	data, err := assets.ReadFile("assets/index.html")
	if err != nil {
		http.Error(w, "embedded UI unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'")
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
	var graph *source.Graph
	if current := s.store.Current(); current != nil {
		graph = current.Graph()
	}
	localized := make([]diagnostic.Diagnostic, len(diagnostics))
	for i, value := range diagnostics {
		localized[i] = diagnostic.Localize(value, locale)
		localized[i].Path = displayDiagnosticPath(localized[i], graph)
	}
	writeJSON(w, localized)
}

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
	evaluation := current.Evaluation()
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/reports/")
	name = strings.TrimSuffix(name, "/")
	chartRoute := ""
	var result query.Result
	switch strings.ToLower(name) {
	case "accounts", "account":
		result = report.Accounts(evaluation)
		chartRoute = "accounts"
	case "journal":
		from, to, err := journalDateRange(r)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		result = report.JournalBetween(evaluation, from, to)
	case "trial-balance", "trial_balance", "trialbalance":
		// The web report needs explicit ancestors for Fava-style hierarchy
		// rendering. Keep report.TrialBalance flat for query-compatible
		// consumers and use its tree variant only at this presentation boundary.
		result = report.TrialBalanceTree(evaluation)
		chartRoute = "trial-balance"
	case "balance-sheet", "balance_sheet", "balancesheet":
		result = report.BalanceSheet(evaluation)
		chartRoute = "balance-sheet"
	case "income-statement", "income_statement", "incomestatement":
		result = report.IncomeStatement(evaluation)
		chartRoute = "income-statement"
	case "holdings":
		asOf, err := reportAsOfDate(r)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		valuation := strings.TrimSpace(r.URL.Query().Get("valuation"))
		if valuation == "" {
			valuation = "at-cost"
		}
		result = report.HoldingsAtCurrency(evaluation, asOf, valuation, strings.TrimSpace(r.URL.Query().Get("currency")))
	case "prices", "price", "commodities", "commodity":
		result = report.Prices(evaluation)
	case "events", "event":
		result = report.Events(evaluation)
	case "documents", "document":
		result = report.Documents(evaluation)
	case "statistics", "statistic", "stats":
		result = report.Statistics(evaluation)
	case "errors", "error":
		result = report.ErrorsWithGraph(evaluation, current.Graph())
	default:
		http.NotFound(w, r)
		return
	}
	result = redactQueryPaths(result, current.Graph())
	filters, err := globalReportFilters(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	result = report.Filter(result, filters)
	if strings.EqualFold(name, "journal") {
		result = report.FilterJournal(result, report.JournalFilters{
			Flag:      r.URL.Query().Get("flag"),
			Tag:       r.URL.Query().Get("tag"),
			Link:      r.URL.Query().Get("link"),
			Payee:     r.URL.Query().Get("payee"),
			Narration: r.URL.Query().Get("narration"),
		})
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
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	valuation := strings.TrimSpace(r.URL.Query().Get("valuation"))
	if valuation == "" {
		valuation = "at-cost"
	}
	chart := report.ReportChart(evaluation, chartRoute, period, strings.TrimSpace(r.URL.Query().Get("currency")), valuation, strings.TrimSpace(r.URL.Query().Get("account")))
	writeJSON(w, struct {
		query.Result
		Chart report.PresentedChartSpec `json:"chart"`
	}{Result: presented, Chart: report.PresentChart(chart)})
}

func globalReportFilters(r *http.Request) (report.Filters, error) {
	filters := report.Filters{Account: strings.TrimSpace(r.URL.Query().Get("account")), Text: strings.TrimSpace(r.URL.Query().Get("filter"))}
	if period := strings.TrimSpace(r.URL.Query().Get("period")); period != "" {
		switch period {
		case "all", "month", "quarter", "year":
			if period != "all" {
				filters.Period = period
			}
		default:
			return report.Filters{}, fmt.Errorf("invalid period filter")
		}
	}
	if valuation := strings.TrimSpace(r.URL.Query().Get("valuation")); valuation != "" && valuation != "at-cost" && valuation != "market-value" {
		return report.Filters{}, fmt.Errorf("invalid valuation filter")
	}
	rawTime := strings.TrimSpace(r.URL.Query().Get("time"))
	switch rawTime {
	case "", "all":
		return filters, nil
	case "year":
		filters.TimePrefix = time.Now().Format("2006")
	case "month":
		filters.TimePrefix = time.Now().Format("2006-01")
	default:
		if len(rawTime) == 4 {
			if _, err := strconv.Atoi(rawTime); err != nil {
				return report.Filters{}, fmt.Errorf("invalid time filter")
			}
			filters.TimePrefix = rawTime
			return filters, nil
		}
		if len(rawTime) == 7 {
			if _, err := time.Parse("2006-01", rawTime); err != nil {
				return report.Filters{}, fmt.Errorf("invalid time filter")
			}
			filters.TimePrefix = rawTime
			return filters, nil
		}
		return report.Filters{}, fmt.Errorf("invalid time filter")
	}
	return filters, nil
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
	name, err := safeImportName(request.Path, adapter)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	normalized := request.Content
	if adapter == "csv" {
		normalized, err = csvToBeancount(request.Content, request.Mapping)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		name = strings.TrimSuffix(name, filepath.Ext(name)) + ".bean"
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
	for _, directive := range file.Directives {
		switch directive.(type) {
		case ledger.Include, *ledger.Include, ledger.Plugin, *ledger.Plugin:
			writeAPIError(w, http.StatusBadRequest, "imports cannot add include or plugin directives")
			return
		}
	}
	// Evaluate the imported file against the merged include graph rather than
	// in isolation. The full graph knows which accounts are open and which
	// currencies are permitted, so a posting to an account opened only in the
	// main ledger (the common case for an import) is not misreported as an
	// E-EVAL-POSTING lifecycle error. This mirrors what the commit path
	// revalidates, so a preview that says "valid" will commit successfully.
	order := current.Graph().Order
	parsed := current.Parsed()
	// The imported file is not yet a member of the published graph, so it is
	// assigned a fresh FileID that cannot collide with the existing members.
	importID := source.FileID(len(parsed) + 1)
	parsed[importID] = file
	order = append(order, importID)
	evaluation := ledger.EvaluateFiles(parsed, order, ledger.EvalOptions{})
	for _, value := range evaluation.Diagnostics {
		diagnostics = append(diagnostics, value)
	}
	previewID := importPreviewID(name, normalized)
	s.storePreview(previewID, importPreview{Path: name, Content: normalized})
	rows, queryErr := query.Evaluate("SELECT date, account, units, currency, flag, payee, narration FROM postings ORDER BY date, account", *evaluation)
	if queryErr != nil {
		rows = query.Result{}
	} else {
		// The merged graph necessarily includes the existing ledger's postings.
		// Keep the preview focused on the imported file so the table shows only
		// what this import would add.
		imported := rows.Rows[:0]
		for _, row := range rows.Rows {
			if row["file"] == name {
				imported = append(imported, row)
			}
		}
		rows.Rows = imported
	}
	// The imported file itself is always valid here (parse errors already
	// returned); the merged evaluation is valid only when the whole graph,
	// including the proposed import, is free of error diagnostics.
	valid := true
	for _, value := range evaluation.Diagnostics {
		if value.Severity == diagnostic.Error {
			valid = false
			break
		}
	}
	writeJSON(w, struct {
		PreviewID   string               `json:"preview_id"`
		Path        string               `json:"path"`
		Valid       bool                 `json:"valid"`
		Diagnostics []diagnosticResponse `json:"diagnostics"`
		Rows        query.Result         `json:"rows"`
		Diff        importDiff           `json:"diff"`
	}{PreviewID: previewID, Path: name, Valid: valid, Diagnostics: diagnosticsPayload(diagnostics, nil), Rows: report.Present(rows), Diff: importDiff{AddedLines: strings.Count(normalized, "\n") + 1, Bytes: len([]byte(normalized))}})
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

func (s *Server) handleImportCommit(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request importCommitRequest
	if err := decodeJSONBody(w, r, &request, 64<<10); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	preview, ok := s.takePreview(request.PreviewID)
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
	s.pendingMu.Lock()
	delete(s.pending, request.PreviewID)
	s.pendingMu.Unlock()
	writeJSON(w, struct {
		Published  bool   `json:"published"`
		SnapshotID string `json:"snapshot_id"`
		Backup     string `json:"backup"`
	}{Published: true, SnapshotID: result.Snapshot.ID, Backup: backup})
}

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

func csvToBeancount(content string, mapping map[string]string) (string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("invalid CSV: %w", err)
	}
	if len(records) < 2 {
		return "", fmt.Errorf("CSV must contain a header and at least one row")
	}
	header := make(map[string]int, len(records[0]))
	for index, value := range records[0] {
		header[strings.ToLower(strings.TrimSpace(value))] = index
	}
	for _, required := range []string{"date", "account", "amount"} {
		if _, ok := header[required]; !ok {
			return "", fmt.Errorf("CSV requires %s column", required)
		}
	}
	offset := "Equity:Imported"
	if mapping != nil && strings.TrimSpace(mapping["offset_account"]) != "" {
		offset = strings.TrimSpace(mapping["offset_account"])
	}
	currencyDefault := "USD"
	if mapping != nil && strings.TrimSpace(mapping["currency"]) != "" {
		currencyDefault = strings.TrimSpace(mapping["currency"])
	}
	var builder strings.Builder
	for line, record := range records[1:] {
		value := func(key string) string {
			index, ok := header[key]
			if !ok || index >= len(record) {
				return ""
			}
			return strings.TrimSpace(record[index])
		}
		date, amount := value("date"), value("amount")
		if _, err := parseISODate(date); err != nil {
			return "", fmt.Errorf("CSV row %d: invalid date", line+2)
		}
		parsedAmount, err := ledger.ParseDecimal(amount)
		if err != nil {
			return "", fmt.Errorf("CSV row %d: invalid amount", line+2)
		}
		currency := value("currency")
		if currency == "" {
			currency = currencyDefault
		}
		payee := strings.ReplaceAll(strings.ReplaceAll(value("payee"), "\n", " "), "\r", " ")
		narration := strings.ReplaceAll(strings.ReplaceAll(value("narration"), "\n", " "), "\r", " ")
		account := value("account")
		builder.WriteString(fmt.Sprintf("%s * \"%s\"", date, escapeBeanString(payee)))
		if narration != "" {
			builder.WriteString(fmt.Sprintf(" \"%s\"", escapeBeanString(narration)))
		}
		builder.WriteByte('\n')
		builder.WriteString(fmt.Sprintf("  %s %s %s\n", account, parsedAmount.String(), currency))
		builder.WriteString(fmt.Sprintf("  %s %s %s\n", offset, parsedAmount.Neg().String(), currency))
	}
	return builder.String(), nil
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

func (s *Server) handleHelp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, struct {
		Sections []map[string]string `json:"sections"`
	}{Sections: []map[string]string{
		{"id": "navigation", "title": "Navigation", "body": "Use the sidebar or the menu button to move between reports."},
		{"id": "filters", "title": "Filters", "body": "Global time, account, and text filters are bookmarkable URL state."},
		{"id": "editor", "title": "Editor safety", "body": "Validate before saving. Saves are atomic, backed up, and revalidated before publication."},
		{"id": "import", "title": "Import review", "body": "Preview imported postings and explicitly commit them to a selected ledger file."},
		{"id": "prices", "title": "Local prices", "body": "Market valuation uses only price directives in the local ledger. Missing quotes are shown as unavailable; no external provider is contacted."},
		{"id": "plugins", "title": "Plugin migration", "body": "Python plugins are never executed. Plugin directives remain visible as diagnostics so they can be migrated explicitly."},
		{"id": "shortcuts", "title": "Keyboard", "body": "Tab reaches controls; Enter applies filters and runs queries."},
	}})
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

func (s *Server) handleApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	data, err := assets.ReadFile("assets/app.js")
	if err != nil {
		http.Error(w, "embedded UI unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
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
