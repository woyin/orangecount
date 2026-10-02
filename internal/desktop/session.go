// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package desktop owns one read-only ledger session per native window.
package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime/trace"
	"sync"
	"time"

	"orangecount/internal/documents"
	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
	"strings"
)

var (
	ErrClosed       = errors.New("ledger session is closed")
	ErrRequest      = errors.New("invalid view request")
	ErrStale        = errors.New("ledger snapshot changed; request the current view")
	ErrResultBudget = errors.New("ledger result exceeds the response budget")
)

type State struct {
	SemanticsProfile   ledger.SemanticsProfile `json:"semantics_profile"`
	EvaluationRevision string                  `json:"evaluation_revision"`
	ActiveExtensions   []string                `json:"active_extensions"`
	Revision           uint64                  `json:"revision"`
	SnapshotID         string                  `json:"snapshot_id"`
	EntryPath          string                  `json:"entry_path"`
	Valid              bool                    `json:"valid"`
	HasSnapshot        bool                    `json:"has_snapshot"`
	Reloading          bool                    `json:"reloading"`
	BuildCount         uint64                  `json:"build_count"`
}

type Request struct {
	DocumentID       string `json:"document_id,omitempty"`
	JournalMode      string `json:"journal_mode,omitempty"`
	DirectiveKind    string `json:"directive_kind,omitempty"`
	MigrationDetails bool   `json:"migration_details"`
	Valuation        string `json:"valuation"`
	View             string `json:"view"`
	Currency         string `json:"currency"`
	Locale           string `json:"locale"`
	SnapshotID       string `json:"snapshot_id,omitempty"`
	DateFrom         string `json:"date_from"`
	DateTo           string `json:"date_to"`
	Window           string `json:"window"`
	Account          string `json:"account"`
	Scope            string `json:"scope,omitempty"`
	GroupBy          string `json:"group_by,omitempty"`
	HoldingSection   string `json:"holding_section,omitempty"`
	GroupID          string `json:"group_id,omitempty"`
	EventID          string `json:"event_id,omitempty"`
	LotKind          string `json:"lot_kind,omitempty"`
	PriceSection     string `json:"price_section,omitempty"`
	PriceBase        string `json:"price_base,omitempty"`
	PriceQuote       string `json:"price_quote,omitempty"`
	Text             string `json:"text"`
	Sort             string `json:"sort"`
	Offset           int    `json:"offset"`
	Limit            int    `json:"limit"`
}

type ViewResponse struct {
	ReportExport   *ReportExportIdentity         `json:"report_export,omitempty"`
	DocumentAction *DocumentAction               `json:"document_action,omitempty"`
	Directives     *DirectivesPayload            `json:"directives,omitempty"`
	QueryCatalogue *QueryCatalogue               `json:"query_catalogue,omitempty"`
	Prices         *report.PricesProjection      `json:"prices,omitempty"`
	Holdings       *report.HoldingsProjection    `json:"holdings,omitempty"`
	Account        *AccountDetailPayload         `json:"account_detail,omitempty"`
	State          State                         `json:"state"`
	Request        Request                       `json:"request"`
	Summary        SummaryPayload                `json:"summary"`
	Tree           *TreeReportPayload            `json:"tree,omitempty"`
	Journal        *JournalPayload               `json:"journal,omitempty"`
	Trends         *HistoricalTrendsPayload      `json:"trends,omitempty"`
	Migration      *snapshot.MigrationComparison `json:"migration,omitempty"`
}

type publication struct {
	documentMu         sync.Mutex
	documentResolver   *documents.Resolver
	documentReferences map[string]documents.Reference
	protectedPaths     []string
	snapshot           *snapshot.Snapshot
	evaluation         ledger.Evaluation
	journal            []journalEntry
	latestDate         string
}

type Session struct {
	reportWork                map[string]context.CancelFunc
	cancelledReports          map[string]bool
	cancelledReportOrder      []string
	reportCaptures            map[string]cachedReport
	reportCaptureOrder        []string
	reportCaptureBytes        int
	mu                        sync.RWMutex
	closed                    bool
	store                     *snapshot.Store
	state                     State
	latest                    snapshot.BuildResult
	current                   *publication
	reloads                   chan chan error
	stop                      chan struct{}
	done                      chan struct{}
	closeOnce                 sync.Once
	options                   snapshot.BuildOptions
	queryCancel               context.CancelFunc
	queryRequestID            uint64
	queryLastID               uint64
	queryCancelledThrough     uint64
	queryGate                 chan struct{}
	queryExport               *QueryResponse
	queryExportProtectedPaths []string
}

type OpenOptions struct {
	SemanticsProfile ledger.SemanticsProfile `json:"semantics_profile"`
	Watch            snapshot.WatchOptions   `json:"-"`
	InputIdentity    string                  `json:"input_identity,omitempty"`
}

// Open performs one initial build. Invalid ledgers still produce a session
// so their diagnostics remain readable and a later repair can recover it.
func Open(path string, options snapshot.WatchOptions) (*Session, error) {
	return OpenWithOptions(path, OpenOptions{Watch: options})
}

func OpenWithOptions(path string, selected OpenOptions) (*Session, error) {
	profile, err := ledger.ParseSemanticsProfile(string(selected.SemanticsProfile))
	if err != nil {
		return nil, ErrRequest
	}
	options := selected.Watch
	entry, err := filepath.Abs(path)
	if err != nil || path == "" {
		return nil, ErrRequest
	}
	s := &Session{store: snapshot.NewStore(nil), state: State{EntryPath: entry}, reloads: make(chan chan error), stop: make(chan struct{}), done: make(chan struct{}), queryGate: make(chan struct{}, 1)}
	s.options.Evaluation.SemanticsProfile = profile
	s.state.SemanticsProfile, s.state.EvaluationRevision = profile, profile.EvaluationRevision()
	s.state.ActiveExtensions = []string{}
	initialDetector := snapshot.NewChangeDetector(entry, nil)
	initialSignature := initialDetector.Signature()
	first := s.build()
	if selected.InputIdentity != "" && snapshot.InputIdentity(first.Graph) != selected.InputIdentity {
		return nil, ErrStale
	}
	s.publish(first)
	if options.PollInterval <= 0 {
		options.PollInterval = 250 * time.Millisecond
	}
	if options.Debounce <= 0 {
		options.Debounce = 150 * time.Millisecond
	}
	detector := snapshot.NewChangeDetector(entry, s.latest.Graph)
	signature := detector.Signature()
	changedDuringBuild := s.latest.Graph.ChangedOnDisk() || initialDetector.Signature() != initialSignature
	go s.run(options, detector, signature, changedDuringBuild)
	return s, nil
}

func (s *Session) build() snapshot.BuildResult {
	region := trace.StartRegion(context.Background(), "desktop.build")
	defer region.End()
	return s.store.Reload(s.state.EntryPath, s.options)
}

func (s *Session) publish(result snapshot.BuildResult) {
	region := trace.StartRegion(context.Background(), "desktop.publish")
	defer region.End()
	var next *publication
	if result.Snapshot != nil {
		next = &publication{evaluation: result.Snapshot.Evaluation(), snapshot: result.Snapshot}
		next.protectedPaths = exportProtectedPaths(result.Graph, next.evaluation)
		next.journal = indexJournal(next.evaluation, result.Snapshot.ID)
		for _, entry := range next.evaluation.Entries {
			if entry.Date.Raw > next.latestDate {
				next.latestDate = entry.Date.Raw
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.latest = result
	s.state.BuildCount++
	s.state.Revision++
	s.state.Valid = result.Snapshot != nil
	s.state.Reloading = false
	if next != nil {
		if s.queryCancel != nil && result.Snapshot.ID != s.state.SnapshotID {
			s.queryCancel()
		}
		if result.Snapshot.ID != s.state.SnapshotID {
			s.reportCaptures = nil
			s.reportCaptureOrder = nil
			s.reportCaptureBytes = 0
			s.queryExport = nil
			s.queryExportProtectedPaths = nil
		}
		s.current = next
		s.state.HasSnapshot = true
		s.state.SnapshotID = result.Snapshot.ID
		s.state.ActiveExtensions = append([]string{}, next.evaluation.ActiveExtensions...)
	}
}

// Poll is an in-memory state read: no filesystem operations or projections.
func (s *Session) Poll() (State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return State{}, ErrClosed
	}
	state := s.state
	state.ActiveExtensions = append([]string{}, state.ActiveExtensions...)
	return state, nil
}

// Read captures one coherent publication before projecting any view. The
// published evaluation is privately retained and never mutated by callers.
func (s *Session) Read(request Request) (ViewResponse, error) {
	if request.View == "" {
		request.View = "balance_sheet"
	}
	if request.Locale == "" {
		request.Locale = "zh-CN"
	}
	if request.Locale != "zh-CN" && request.Locale != "en" {
		return ViewResponse{}, ErrRequest
	}
	switch request.View {
	case "balance_sheet", "income_statement", "journal", "dashboard", "migration", "account", "holdings", "prices", "query":
	default:
		return ViewResponse{}, ErrRequest
	}
	if !validJournalRequest(request) {
		return ViewResponse{}, ErrRequest
	}
	if request.View == "account" {
		if request.Account == "" {
			return ViewResponse{}, ErrRequest
		}
		if request.Scope == "" {
			request.Scope = "subtree"
		}
	}
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ViewResponse{}, ErrClosed
	}
	state, current, latest := s.state, s.current, s.latest
	s.mu.RUnlock()
	if request.SnapshotID != "" && request.SnapshotID != state.SnapshotID {
		return ViewResponse{}, ErrStale
	}
	var evaluation *ledger.Evaluation
	if current != nil {
		evaluation = &current.evaluation
	}
	result := ViewResponse{State: state, Request: request, Summary: ProjectSummary(state.EntryPath, evaluation, ProjectDiagnostics(latest.Diagnostics, latest.Graph, request.Locale), state.Valid)}
	if request.View == "migration" {
		comparison := snapshot.CompareSemantics(latest.Graph, request.MigrationDetails)
		result.Migration = &comparison
		return result, nil
	}
	if current == nil {
		return result, nil
	}
	if request.DocumentID != "" {
		action, err := current.authorizeDocument(request.DocumentID)
		if err != nil {
			return ViewResponse{}, err
		}
		result.DocumentAction = action
		return result, nil
	}
	if request.View == "query" {
		from, to := current.dateRange(request)
		result.QueryCatalogue = projectQueryCatalogue(*evaluation, request, state.SnapshotID, from, to)
		data, err := json.Marshal(result)
		if err != nil {
			return ViewResponse{}, err
		}
		if len(data) > holdingsByteLimit-64 {
			return ViewResponse{}, ErrResultBudget
		}
		return result, nil
	}
	from, to := current.dateRange(request)
	if request.View != "balance_sheet" && !(request.View == "journal" && request.JournalMode == "config") && from != "" && to != "" && from > to {
		return ViewResponse{}, ErrRequest
	}
	switch request.View {
	case "prices":
		prices, err := projectPrices(*evaluation, request, state.SnapshotID, from, to)
		if err != nil {
			return ViewResponse{}, err
		}
		result.Prices = prices
	case "holdings":
		holdings, err := projectHoldings(*evaluation, request, state.SnapshotID, from, to)
		if err != nil {
			return ViewResponse{}, err
		}
		result.Holdings = holdings
	case "account":
		account, err := projectAccount(*evaluation, request, from, to)
		if err != nil {
			return ViewResponse{}, err
		}
		result.Account = account
		journal := current.readJournal(request)
		result.Journal = &journal
	case "balance_sheet", "income_statement":
		cur := request.Currency
		if cur == "native" {
			cur = ""
		}
		statement := report.StatementView(*evaluation, request.View, from, to, cur, request.Valuation)
		tree := TreeReportPayload{Currency: cur, ReportType: request.View, OperatingCurrencies: strings.Fields(evaluation.Options["operating_currency"]), RootNodes: statement.Nodes, DateFrom: from, DateTo: to, Availability: statement.Availability, ValuationStatus: statement.ValuationStatus}
		result.Tree = &tree
	case "journal":
		if request.JournalMode == "all" || request.JournalMode == "config" {
			payload, err := current.readDirectives(request, state.SnapshotID, from, to)
			if err != nil {
				return ViewResponse{}, err
			}
			result.Directives = payload
		} else {
			journal := current.readJournal(request)
			result.Journal = &journal
		}
	case "dashboard":
		cur := request.Currency
		if cur == "native" {
			cur = ""
		}
		trends := projectHistory(*evaluation, from, to, cur, request.Valuation)
		result.Trends = &trends
	}
	if request.View == "holdings" || request.View == "prices" || result.Directives != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return ViewResponse{}, err
		}
		// Reserve the small fixed C envelope in addition to state/summary.
		if len(data) > holdingsByteLimit-64 {
			return ViewResponse{}, ErrResultBudget
		}
	}
	if err := s.retainReport(&result); err != nil {
		return ViewResponse{}, err
	}
	return result, nil
}

// Close is idempotent. Already captured immutable reads can safely finish.
func (s *Session) Close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		for _, cancel := range s.reportWork {
			cancel()
		}
		s.reportCaptures = nil
		s.reportCaptureOrder = nil
		s.reportCaptureBytes = 0
		s.queryExport = nil
		s.queryExportProtectedPaths = nil
		if s.queryCancel != nil {
			s.queryCancel()
		}
		s.mu.Unlock()
		close(s.stop)
		<-s.done
		s.mu.Lock()
		s.current = nil
		s.latest = snapshot.BuildResult{}
		s.store = nil
		s.mu.Unlock()
	})
}

// Reload waits for a queued attempt, not necessarily a valid publication.
// Simultaneous pending requests share an attempt; watch and manual work never
// run two builders concurrently.
func (s *Session) Reload() error {
	answer := make(chan error, 1)
	select {
	case <-s.stop:
		return ErrClosed
	case s.reloads <- answer:
	}
	select {
	case <-s.stop:
		return ErrClosed
	case err := <-answer:
		return err
	}
}

func (s *Session) run(options snapshot.WatchOptions, detector *snapshot.ChangeDetector, lastSignature string, pending bool) {
	defer close(s.done)
	ticker := time.NewTicker(options.PollInterval)
	defer ticker.Stop()
	results := make(chan snapshot.BuildResult, 1)
	var waiting, active []chan error
	var changedAt time.Time
	building := false
	notify := func(waiters []chan error, err error) {
		for _, waiter := range waiters {
			waiter <- err
		}
	}
	for {
		select {
		case <-s.stop:
			if building {
				<-results
			}
			notify(active, ErrClosed)
			notify(waiting, ErrClosed)
			return
		case waiter := <-s.reloads:
			waiting = append(waiting, waiter)
			pending, changedAt = true, time.Time{}
		case now := <-ticker.C:
			if signature := detector.Signature(); signature != lastSignature {
				trace.Log(context.Background(), "desktop.watch", "change detected")
				lastSignature = signature
				pending, changedAt = true, now
			}
		case result := <-results:
			s.publish(result)
			building = false
			notify(active, nil)
			active = nil
			// A change near the end of a build must not disappear when the
			// watch set is refreshed from the attempted include graph.
			if detector.Signature() != lastSignature {
				pending, changedAt = true, time.Now()
			}
			detector = snapshot.NewChangeDetector(s.state.EntryPath, result.Graph)
			lastSignature = detector.Signature()
			if result.Graph.ChangedOnDisk() {
				pending, changedAt = true, time.Now()
			}
		}
		if pending && !building && (changedAt.IsZero() || time.Since(changedAt) >= options.Debounce) {
			pending, building = false, true
			active, waiting = waiting, nil
			changedAt = time.Time{}
			s.mu.Lock()
			s.state.Reloading = true
			s.state.Revision++
			s.mu.Unlock()
			go func() { results <- s.build() }()
		}
	}
}

// Empty means the ledger's first operating currency; native deliberately keeps
// each commodity separate. The echoed request remains unchanged for callers.
func statementCurrency(e ledger.Evaluation, requested string) string {
	if requested == "native" {
		return ""
	}
	if requested == "" {
		if currencies := strings.Fields(e.Options["operating_currency"]); len(currencies) > 0 {
			return currencies[0]
		}
	}
	return requested
}
