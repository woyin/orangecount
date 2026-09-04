// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package web provides the small loopback-only HTTP surface used by serve.
package web

import (
	"context"
	"embed"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"orangecount/internal/authoring"
	"orangecount/internal/diagnostic"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
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
	mu               sync.RWMutex
	authoring        *authoring.Writer
	quickMu          sync.Mutex
	store            *snapshot.Store
	roots            source.DocumentRoots
	addr             string
	http             *http.Server
	bound            string
	ready            chan struct{}
	readyOnce        sync.Once
	readyErr         error
	optionsMu        sync.RWMutex
	options          map[string]string
	previews         *importPreviewStore
	quickPreviews    *quickPreviewStore
	quickLastBatch   *quickBatchRecord
	planningPreviews *planningPreviewStore
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
	writer, err := authoring.NewWriter(config.Store)
	if err != nil {
		return nil, err
	}
	server := &Server{store: config.Store, authoring: writer, roots: config.DocumentRoots, addr: addr, ready: make(chan struct{}), options: make(map[string]string), previews: newImportPreviewStore(), quickPreviews: newQuickPreviewStore(), planningPreviews: newPlanningPreviewStore()}
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
