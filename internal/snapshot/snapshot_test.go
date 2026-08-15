// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package snapshot

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

func TestBuildAndFailedReloadRetainsPreviousSnapshot(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	valid := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"ok\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"
	if err := os.WriteFile(entry, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	first := Build(entry)
	if first.Snapshot == nil {
		t.Fatalf("initial diagnostics=%+v err=%v", first.Diagnostics, first.Err)
	}
	store := NewStore(first.Snapshot)
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n2000-01-02 * \"broken\"\n  Assets:Cash 1 USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reloaded := store.Reload(entry, BuildOptions{})
	if reloaded.Snapshot != nil || store.Current() != first.Snapshot || len(store.Diagnostics()) == 0 {
		t.Fatalf("reload=%+v current=%p first=%p diagnostics=%+v", reloaded, store.Current(), first.Snapshot, store.Diagnostics())
	}
}

func TestWatchDebouncesIncludeGraphChanges(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first := Build(entry)
	if first.Snapshot == nil {
		t.Fatalf("initial diagnostics=%+v", first.Diagnostics)
	}
	store := NewStore(first.Snapshot)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	count := 0
	done := make(chan struct{})
	go func() {
		_ = store.Watch(ctx, entry, BuildOptions{}, WatchOptions{PollInterval: 10 * time.Millisecond, Debounce: 20 * time.Millisecond}, func(result ReloadResult) {
			mu.Lock()
			count++
			mu.Unlock()
			close(done)
		})
	}()
	time.Sleep(30 * time.Millisecond)
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n; changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("watcher did not reload")
	}
	cancel()
	mu.Lock()
	got := count
	mu.Unlock()
	if got != 1 {
		t.Fatalf("reload count=%d", got)
	}
}

func TestSnapshotPreservesEvaluationOptions(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte(`option "title" "Probe Ledger"
option "operating_currency" "CNY"
2020-01-01 open Assets:Cash CNY
2020-01-01 open Equity:Opening CNY
2020-01-01 * "opening"
  Assets:Cash 1 CNY
  Equity:Opening
`), 0o644); err != nil {
		t.Fatal(err)
	}
	res := Build(entry)
	if res.Snapshot == nil {
		t.Fatalf("snapshot nil: %v", res.Err)
	}
	opts := res.Snapshot.Evaluation().Options
	if opts["title"] != "Probe Ledger" {
		t.Fatalf("title option lost: %v", opts)
	}
	if opts["operating_currency"] != "CNY" {
		t.Fatalf("operating_currency option lost: %v", opts)
	}
}

func TestBuildFailureAndSnapshotAccessorsAreSafeAndDefensive(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.bean")
	failed := Build(missing)
	if failed.Snapshot != nil || failed.Err == nil || len(failed.Diagnostics) != 1 || failed.Diagnostics[0].Code != "E-INCLUDE-READ" {
		t.Fatalf("failed build=%+v", failed)
	}
	if snapshotID(nil) != "" {
		t.Fatal("nil graph had a snapshot ID")
	}

	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	valid := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"opening\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"
	if err := os.WriteFile(entry, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	result := Build(entry)
	if result.Snapshot == nil || !result.Snapshot.Valid() || result.Snapshot.Graph() == nil {
		t.Fatalf("snapshot=%+v", result)
	}
	parsed := result.Snapshot.Parsed()
	if len(parsed) != 1 {
		t.Fatalf("parsed=%v", parsed)
	}
	delete(parsed, result.Snapshot.Graph().Entry)
	if len(result.Snapshot.Parsed()) != 1 {
		t.Fatal("parsed accessor leaked its map")
	}
	evaluation := result.Snapshot.Evaluation()
	evaluation.Accounts["Assets:Cash"] = ledger.AccountState{}
	evaluation.Options["title"] = "changed"
	if result.Snapshot.Evaluation().Options["title"] == "changed" || result.Snapshot.Evaluation().Accounts["Assets:Cash"].Balances["USD"].String() != "1" {
		t.Fatal("evaluation accessor leaked mutable collections")
	}
	diagnostics := result.Snapshot.Diagnostics()
	diagnostics = append(diagnostics, diagnostic.New("E-TEST", diagnostic.Error, source.Span{}))
	if len(result.Snapshot.Diagnostics()) != len(result.Diagnostics) {
		t.Fatal("diagnostics accessor leaked its slice")
	}

	var nilSnapshot *Snapshot
	if nilSnapshot.Valid() || nilSnapshot.Graph() != nil || nilSnapshot.Parsed() != nil || nilSnapshot.Diagnostics() != nil || nilSnapshot.Evaluation().Valid {
		t.Fatal("nil snapshot accessors returned state")
	}
}

func TestStorePublishAndWatchOptionsRespectPublicationInvariants(t *testing.T) {
	var nilStore *Store
	if nilStore.Current() != nil || nilStore.Diagnostics() != nil || nilStore.Publish(nil, nil) {
		t.Fatal("nil store mutated or exposed state")
	}
	if err := nilStore.Watch(context.Background(), "ignored", BuildOptions{}, WatchOptions{}, nil); err == nil {
		t.Fatal("nil store watch did not fail")
	}
	if got := (WatchOptions{}).normalized(); got.PollInterval != 250*time.Millisecond || got.Debounce != 150*time.Millisecond {
		t.Fatalf("normalized defaults=%+v", got)
	}
	custom := WatchOptions{PollInterval: time.Millisecond, Debounce: 2 * time.Millisecond}.normalized()
	if custom.PollInterval != time.Millisecond || custom.Debounce != 2*time.Millisecond {
		t.Fatalf("custom watch options=%+v", custom)
	}

	store := NewStore(nil)
	if store.Publish(nil, []diagnostic.Diagnostic{diagnostic.New("E-TEST", diagnostic.Error, source.Span{})}) || store.Current() != nil {
		t.Fatal("nil candidate was published")
	}
	candidate := &Snapshot{ID: "candidate", evaluation: &ledger.Evaluation{Valid: true}}
	inputDiagnostics := []diagnostic.Diagnostic{diagnostic.New("W-TEST", diagnostic.Warning, source.Span{})}
	if !store.Publish(candidate, inputDiagnostics) || store.Current() != candidate {
		t.Fatal("candidate was not published")
	}
	inputDiagnostics[0].Code = "mutated"
	if got := store.Diagnostics()[0].Code; got != "W-TEST" {
		t.Fatalf("store diagnostics leaked input slice: %q", got)
	}
}

func TestGraphPatrolSignatureTracksStatIdentity(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	included := filepath.Join(dir, "extra.bean")
	mainLedger := "2000-01-01 open Assets:Cash USD\ninclude \"extra.bean\"\n"
	if err := os.WriteFile(entry, []byte(mainLedger), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(included, []byte("2000-01-01 open Equity:Opening USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	patrol := newGraphPatrol(entry)
	first := patrol.signature()
	if second := patrol.signature(); second != first {
		t.Fatal("stable ledger produced unstable signature")
	}

	// Content edit on the entry changes the signature.
	if err := os.WriteFile(entry, []byte(mainLedger+"; changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if second := patrol.signature(); second == first {
		t.Fatal("content change did not alter patrol signature")
	}

	// Deletion and re-appearance of an included file both change it.
	if err := os.Remove(included); err != nil {
		t.Fatal(err)
	}
	deleted := newGraphPatrol(entry).signature()
	if err := os.WriteFile(included, []byte("2000-01-01 open Equity:Opening USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	restored := newGraphPatrol(entry).signature()
	if deleted == restored {
		t.Fatal("included file deletion/recreation kept identical signature")
	}

	// A patrol over a missing entry still stats the entry path itself.
	missing := newGraphPatrol(filepath.Join(dir, "absent.bean"))
	if !strings.Contains(missing.signature(), ":missing\x00") {
		t.Fatalf("missing entry signature=%q", missing.signature())
	}
}

func TestGraphPatrolWatchesPreviouslyMissingIncludeTargets(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	target := filepath.Join(dir, "late.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\ninclude \"late.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	patrol := newGraphPatrol(entry)
	before := patrol.signature()
	if err := os.WriteFile(target, []byte("2000-01-01 open Equity:Opening USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if after := patrol.signature(); after == before {
		t.Fatal("appearance of a previously missing include target was not detected")
	}
	// After a rebuild the patrol refreshes: the target is now a graph file
	// instead of a diagnostic path, and the signature is stable again.
	refreshed := patrolFromGraph(entry, Build(entry).Graph)
	if refreshed.signature() != refreshed.signature() {
		t.Fatal("refreshed patrol signature is unstable")
	}
}

func TestWatchReloadsWhenMissingIncludeTargetAppears(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\ninclude \"late.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first := Build(entry)
	// FD-0004: a missing include target is an error-severity diagnostic, so
	// the initial build publishes no snapshot; the store still watches.
	store := NewStore(first.Snapshot)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan ReloadResult, 1)
	go func() {
		_ = store.Watch(ctx, entry, BuildOptions{}, WatchOptions{PollInterval: 10 * time.Millisecond, Debounce: 20 * time.Millisecond}, func(result ReloadResult) {
			select {
			case done <- result:
			default:
			}
		})
	}()
	time.Sleep(30 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(dir, "late.bean"), []byte("2000-01-01 open Equity:Opening USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-done:
		if !result.Published {
			t.Fatalf("reload did not publish: diagnostics=%+v", result.Diagnostics)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("watcher ignored a missing include target appearing")
	}
}
