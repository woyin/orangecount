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

func (s *Snapshot) Graph() *source.Graph {
	if s == nil {
		return nil
	}
	return s.graph
}

func (s *Snapshot) Valid() bool { return s != nil && s.evaluation != nil && s.evaluation.Valid }

func (s *Snapshot) Parsed() map[source.FileID]*ledger.File {
	if s == nil {
		return nil
	}
	return cloneParsed(s.parsed)
}

func (s *Snapshot) Evaluation() ledger.Evaluation {
	if s == nil || s.evaluation == nil {
		return ledger.Evaluation{}
	}
	return cloneEvaluation(*s.evaluation)
}

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

func NewStore(initial *Snapshot) *Store {
	store := &Store{current: initial}
	if initial != nil {
		store.attempted = initial.Graph()
	}
	return store
}

func (s *Store) Current() *Snapshot {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

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
	lastSignature := graphSignature(entry)
	ticker := time.NewTicker(watch.PollInterval)
	defer ticker.Stop()
	var changedAt time.Time
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			signature := graphSignature(entry)
			if signature != lastSignature {
				lastSignature = signature
				changedAt = now
			}
			if !changedAt.IsZero() && now.Sub(changedAt) >= watch.Debounce {
				result := s.Reload(entry, options)
				published := result.Snapshot != nil
				if callback != nil {
					callback(ReloadResult{BuildResult: result, Published: published})
				}
				changedAt = time.Time{}
			}
		}
	}
}

func graphSignature(entry string) string {
	graph, err := source.LoadGraph(entry)
	if err != nil {
		return "error:" + err.Error()
	}
	type item struct {
		path string
		size int64
		mod  int64
	}
	items := make([]item, 0, len(graph.Order))
	for _, id := range graph.Order {
		file := graph.File(id)
		if file == nil {
			continue
		}
		info, statErr := os.Stat(file.Path)
		if statErr != nil {
			items = append(items, item{path: file.Path})
			continue
		}
		items = append(items, item{path: file.Path, size: info.Size(), mod: info.ModTime().UnixNano()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].path < items[j].path })
	var builder strings.Builder
	for _, value := range items {
		fmt.Fprintf(&builder, "%s:%d:%d\x00", value.path, value.size, value.mod)
	}
	// Include content hashes so rapid writes with identical size and timestamp
	// granularity still trigger a reload.
	for _, id := range graph.Order {
		file := graph.File(id)
		if file == nil {
			continue
		}
		sum := sha256.Sum256(file.Data)
		fmt.Fprintf(&builder, "%x\x00", sum[:])
	}
	return builder.String()
}
