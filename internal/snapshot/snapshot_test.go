// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package snapshot

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
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
