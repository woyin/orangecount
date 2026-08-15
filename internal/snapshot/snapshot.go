// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package snapshot builds and publishes immutable, fully evaluated ledger
// views. A failed build never replaces the last valid view.
package snapshot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

// BuildOptions parameterizes evaluation when building a snapshot.
type BuildOptions struct {
	Evaluation ledger.EvalOptions
}

// Snapshot is an immutable published ledger view. Its fields are read-only by
// convention; callers should use Evaluation() and Diagnostics() to receive
// defensive copies of mutable collections.
type Snapshot struct {
	ID        string
	BuiltAt   time.Time
	EntryPath string

	graph       *source.Graph
	parsed      map[source.FileID]*ledger.File
	evaluation  *ledger.Evaluation
	diagnostics []diagnostic.Diagnostic
}

// BuildResult is the outcome of one build attempt.
type BuildResult struct {
	Snapshot *Snapshot
	// Graph is the source graph from the latest attempted build. It is kept
	// even when Snapshot is nil so local diagnostics can explain a failed
	// reload without publishing an invalid accounting view.
	Graph       *source.Graph
	Diagnostics []diagnostic.Diagnostic
	Err         error
}

// Build reads, parses, evaluates, and validates an include graph. Snapshot is
// nil whenever an error diagnostic prevents safe publication.
func Build(entry string) BuildResult { return BuildWithOptions(entry, BuildOptions{}) }

// BuildWithOptions runs Build with explicit evaluation options.
func BuildWithOptions(entry string, options BuildOptions) BuildResult {
	graph, err := source.LoadGraph(entry)
	if err != nil {
		bag := diagnostic.Bag{}
		bag.Add(diagnostic.New("E-INCLUDE-READ", diagnostic.Error, source.Span{}).WithPath(entry))
		// LoadGraph can still return a partial graph (for example, when an
		// included file is unreadable). Keep it available for bounded source
		// context even though no accounting snapshot may be published.
		return BuildResult{Graph: graph, Diagnostics: bag.All(), Err: err}
	}
	parsed, parseBag := ledger.ParseGraph(graph)
	evaluation := ledger.Evaluate(graph, parsed, options.Evaluation)
	bag := diagnostic.Bag{}
	bag.Extend(parseBag.All()...)
	for _, d := range evaluation.Diagnostics {
		bag.Add(d)
	}
	diagnostics := bag.All()
	if bag.HasErrors() || !evaluation.Valid {
		return BuildResult{Graph: graph, Diagnostics: diagnostics}
	}
	id := snapshotID(graph)
	return BuildResult{Snapshot: &Snapshot{ID: id, BuiltAt: time.Now().UTC(), EntryPath: graph.Path(graph.Entry), graph: graph, parsed: parsed, evaluation: evaluation, diagnostics: append([]diagnostic.Diagnostic(nil), diagnostics...)}, Graph: graph, Diagnostics: diagnostics}
}

func snapshotID(graph *source.Graph) string {
	hash := sha256.New()
	if graph == nil {
		return ""
	}
	for _, id := range graph.Order {
		file := graph.File(id)
		if file == nil {
			continue
		}
		hash.Write([]byte(file.Path))
		hash.Write([]byte{0})
		hash.Write(file.Data)
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))[:16]
}

// Graph exposes the parsed include graph; nil when the snapshot is nil.
func (s *Snapshot) Graph() *source.Graph {
	if s == nil {
		return nil
	}
	return s.graph
}

// Valid reports whether the snapshot carries an evaluation safe to publish.
func (s *Snapshot) Valid() bool { return s != nil && s.evaluation != nil && s.evaluation.Valid }

// Parsed returns a defensive copy of the per-file parsed ASTs.
func (s *Snapshot) Parsed() map[source.FileID]*ledger.File {
	if s == nil {
		return nil
	}
	return cloneParsed(s.parsed)
}

// Evaluation returns a deep copy of the evaluated ledger state.
func (s *Snapshot) Evaluation() ledger.Evaluation {
	if s == nil || s.evaluation == nil {
		return ledger.Evaluation{}
	}
	return cloneEvaluation(*s.evaluation)
}

// Diagnostics returns a copy of the build's diagnostics.
func (s *Snapshot) Diagnostics() []diagnostic.Diagnostic {
	if s == nil {
		return nil
	}
	return append([]diagnostic.Diagnostic(nil), s.diagnostics...)
}

func cloneParsed(parsed map[source.FileID]*ledger.File) map[source.FileID]*ledger.File {
	copyFiles := make(map[source.FileID]*ledger.File, len(parsed))
	for id, file := range parsed {
		copyFiles[id] = file
	}
	return copyFiles
}

func cloneEvaluation(value ledger.Evaluation) ledger.Evaluation {
	accounts := value.Accounts
	value.Accounts = make(map[string]ledger.AccountState, len(accounts))
	for name, account := range accounts {
		balances := account.Balances
		account.Balances = make(map[string]ledger.Decimal, len(account.Balances))
		for currency, amount := range balances {
			account.Balances[currency] = ledger.NewDecimal(amount.Rat())
		}
		account.Currencies = append([]string(nil), account.Currencies...)
		account.Positions = append([]ledger.Position(nil), account.Positions...)
		value.Accounts[name] = account
	}
	prices := value.Prices
	value.Prices = make(map[string][]ledger.PriceQuote, len(prices))
	for base, quotes := range prices {
		value.Prices[base] = append([]ledger.PriceQuote(nil), quotes...)
	}
	value.Entries = append([]ledger.EntryRecord(nil), value.Entries...)
	options := value.Options
	value.Options = make(map[string]string, len(options))
	for key, option := range options {
		value.Options[key] = option
	}
	value.Diagnostics = append([]diagnostic.Diagnostic(nil), value.Diagnostics...)
	return value
}

// Store atomically retains the most recently valid snapshot and separately
// exposes diagnostics from the latest attempted reload.
type Store struct {
	mu          sync.RWMutex
	current     *Snapshot
	attempted   *source.Graph
	diagnostics []diagnostic.Diagnostic
}

// NewStore creates a store seeded with an optional initial snapshot.
func NewStore(initial *Snapshot) *Store {
	store := &Store{current: initial}
	if initial != nil {
		store.attempted = initial.Graph()
	}
	return store
}

// Current returns the latest published snapshot, or nil.
func (s *Store) Current() *Snapshot {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// Diagnostics returns the diagnostics of the latest attempted reload.
func (s *Store) Diagnostics() []diagnostic.Diagnostic {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]diagnostic.Diagnostic(nil), s.diagnostics...)
}

// LatestGraph returns the immutable source graph from the latest attempted
// build. It may describe an invalid build and therefore must not be used as a
// published accounting snapshot.
func (s *Store) LatestGraph() *source.Graph {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.attempted
}

// Publish atomically installs a candidate snapshot plus its diagnostics;
// false means nothing changed (nil receiver or candidate).
func (s *Store) Publish(candidate *Snapshot, diagnostics []diagnostic.Diagnostic) bool {
	if s == nil || candidate == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = candidate
	s.attempted = candidate.Graph()
	s.diagnostics = append([]diagnostic.Diagnostic(nil), diagnostics...)
	return true
}

// Reload rebuilds from entry, keeps the attempted graph and diagnostics, and
// publishes the snapshot only when the build succeeded.
func (s *Store) Reload(entry string, options BuildOptions) BuildResult {
	result := BuildWithOptions(entry, options)
	s.mu.Lock()
	s.attempted = result.Graph
	s.diagnostics = append([]diagnostic.Diagnostic(nil), result.Diagnostics...)
	if result.Snapshot != nil {
		s.current = result.Snapshot
	}
	s.mu.Unlock()
	return result
}

// WatchOptions tunes the polling loop: how often to stat the graph and how
// long to debounce write bursts.
type WatchOptions struct {
	PollInterval time.Duration
	Debounce     time.Duration
}

func (o WatchOptions) normalized() WatchOptions {
	if o.PollInterval <= 0 {
		o.PollInterval = 250 * time.Millisecond
	}
	if o.Debounce <= 0 {
		o.Debounce = 150 * time.Millisecond
	}
	return o
}

// ReloadResult reports a watched reload's build outcome and whether a valid
// snapshot was published.
type ReloadResult struct {
	BuildResult
	Published bool
}

// Watch polls the resolved include graph without external dependencies. It
// debounces bursts of file writes and invokes callback after each attempted
// reload. The loop ends cleanly when ctx is canceled.
func (s *Store) Watch(ctx context.Context, entry string, options BuildOptions, watch WatchOptions, callback func(ReloadResult)) error {
	if s == nil {
		return fmt.Errorf("nil snapshot store")
	}
	watch = watch.normalized()
	// Seed the patrol with one full graph load; every later tick stats the
	// known paths instead of re-reading file contents (ADR-0044).
	patrol := newGraphPatrol(entry)
	lastSignature := patrol.signature()
	ticker := time.NewTicker(watch.PollInterval)
	defer ticker.Stop()
	var changedAt time.Time
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			signature := patrol.signature()
			if signature != lastSignature {
				lastSignature = signature
				changedAt = now
			}
			if !changedAt.IsZero() && now.Sub(changedAt) >= watch.Debounce {
				result := s.Reload(entry, options)
				// Refresh the patrol from the attempted graph so newly included
				// files are watched and resolved misses stop being statted.
				patrol = patrolFromGraph(entry, result.Graph)
				lastSignature = patrol.signature()
				published := result.Snapshot != nil
				if callback != nil {
					callback(ReloadResult{BuildResult: result, Published: published})
				}
				changedAt = time.Time{}
			}
		}
	}
}

// graphPatrol is the stat-only change detector for the watch loop
// (ADR-0044). It never re-reads file contents: a change is a differing size
// or modification time on any watched path.
type graphPatrol struct {
	paths []string // sorted: entry, graph files, and recorded missing targets
}

// newGraphPatrol seeds the watched paths with one full graph load. A missing
// entry still yields a patrol that stats the entry itself, so the ledger
// appearing later is detected.
func newGraphPatrol(entry string) *graphPatrol {
	graph, _ := source.LoadGraph(entry)
	return patrolFromGraph(entry, graph)
}

// patrolFromGraph derives the watched paths from an attempted graph: the
// entry, every loaded file, and every path reported by E-INCLUDE-READ so a
// previously missing include target is picked up when it appears.
func patrolFromGraph(entry string, graph *source.Graph) *graphPatrol {
	seen := map[string]bool{entry: true}
	paths := []string{entry}
	if graph != nil {
		for _, id := range graph.Order {
			if file := graph.File(id); file != nil && !seen[file.Path] {
				seen[file.Path] = true
				paths = append(paths, file.Path)
			}
		}
		for _, issue := range graph.Diagnostics {
			if issue.Code == "E-INCLUDE-READ" && !seen[issue.Path] {
				seen[issue.Path] = true
				paths = append(paths, issue.Path)
			}
		}
	}
	sort.Strings(paths)
	return &graphPatrol{paths: paths}
}

// signature renders the stat identity of every watched path. Missing paths
// are part of the signature, so both deletions and appearances change it.
func (p *graphPatrol) signature() string {
	var builder strings.Builder
	for _, path := range p.paths {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(&builder, "%s:missing\x00", path)
			continue
		}
		fmt.Fprintf(&builder, "%s:%d:%d\x00", path, info.Size(), info.ModTime().UnixNano())
	}
	return builder.String()
}
