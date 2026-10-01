// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package desktop_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"orangecount/internal/desktop"
	"orangecount/internal/snapshot"
)

func TestSessionIdleDoesNotRebuild(t *testing.T) {
	duration := 120 * time.Millisecond
	if seconds, err := strconv.Atoi(os.Getenv("ORANGECOUNT_IDLE_SECONDS")); err == nil && seconds > 0 {
		duration = time.Duration(seconds) * time.Second
	}
	s, err := desktop.Open("../../testdata/fixtures/native/cashflow.bean", snapshot.WatchOptions{PollInterval: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	initial, _ := s.Poll()
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		state, err := s.Poll()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(state, initial) || state.BuildCount != 1 {
			t.Fatalf("idle session changed or rebuilt: %+v", state)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Logf("idle duration %s: one initial build; no reloads", duration)
}

func awaitState(t *testing.T, s *desktop.Session, match func(desktop.State) bool) desktop.State {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		state, err := s.Poll()
		if err != nil {
			t.Fatal(err)
		}
		if match(state) {
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("session state did not converge")
	return desktop.State{}
}

func TestMissingIncludeRecoveryAndManualReload(t *testing.T) {
	dir := t.TempDir()
	entry, child := filepath.Join(dir, "main.bean"), filepath.Join(dir, "included.bean")
	if err := os.WriteFile(entry, []byte("include \"included.bean\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := desktop.Open(entry, snapshot.WatchOptions{PollInterval: 10 * time.Millisecond, Debounce: 15 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	view, err := s.Read(desktop.Request{View: "journal"})
	if err != nil {
		t.Fatal(err)
	}
	if view.State.HasSnapshot || view.Journal != nil || view.Summary.ErrorCount == 0 {
		t.Fatal("first failure must expose diagnostics without another ledger's data")
	}
	data, err := os.ReadFile("../../testdata/fixtures/native/cashflow.bean")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, data, 0600); err != nil {
		t.Fatal(err)
	}
	valid := awaitState(t, s, func(state desktop.State) bool { return state.Valid && !state.Reloading })
	if err := os.Remove(child); err != nil {
		t.Fatal(err)
	}
	awaitState(t, s, func(state desktop.State) bool { return !state.Valid && !state.Reloading })
	if err := os.WriteFile(child, data, 0600); err != nil {
		t.Fatal(err)
	}
	awaitState(t, s, func(state desktop.State) bool { return state.Valid && !state.Reloading })
	if _, err := s.Read(desktop.Request{View: "journal", SnapshotID: "obsolete"}); !errors.Is(err, desktop.ErrStale) {
		t.Fatal("old snapshot requests must be rejected")
	}
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	after, err := s.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if after.SnapshotID != valid.SnapshotID || after.Revision <= valid.Revision {
		t.Fatal("manual validation must update state while preserving content identity")
	}
}

func TestConcurrentReadReloadAndClose(t *testing.T) {
	for iteration := 0; iteration < 20; iteration++ {
		s, err := desktop.Open("../../testdata/fixtures/native/cashflow.bean", snapshot.WatchOptions{PollInterval: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		var group sync.WaitGroup
		for worker := 0; worker < 4; worker++ {
			group.Add(1)
			go func() {
				defer group.Done()
				_, readErr := s.Read(desktop.Request{View: "journal"})
				if readErr != nil && !errors.Is(readErr, desktop.ErrClosed) {
					t.Error(readErr)
				}
				if err := s.Reload(); err != nil && !errors.Is(err, desktop.ErrClosed) {
					t.Error(err)
				}
			}()
		}
		s.Close()
		group.Wait()
		if _, err := s.Poll(); !errors.Is(err, desktop.ErrClosed) {
			t.Fatal("closed session still responds")
		}
	}
}

func TestWatchRetainsLastValidSnapshotAndUpdatesEveryFailure(t *testing.T) {
	contents, err := os.ReadFile("../../testdata/fixtures/native/cashflow.bean")
	if err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(t.TempDir(), "main.bean")
	write := func(data []byte) {
		t.Helper()
		if err := os.WriteFile(entry, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(contents)
	s, err := desktop.Open(entry, snapshot.WatchOptions{PollInterval: 10 * time.Millisecond, Debounce: 15 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	initial, _ := s.Poll()
	write([]byte("2026-01-01 * \"unterminated\n"))
	failed := awaitState(t, s, func(state desktop.State) bool { return !state.Valid && !state.Reloading })
	view, err := s.Read(desktop.Request{View: "journal"})
	if err != nil {
		t.Fatal(err)
	}
	if failed.SnapshotID != initial.SnapshotID || view.Journal.TotalCount != 2 || view.Summary.ErrorCount == 0 {
		t.Fatal("failed reload must retain the valid journal and publish current diagnostics")
	}
	codes := func(view desktop.ViewResponse) string {
		values := []string{}
		for _, d := range view.Summary.Diagnostics {
			values = append(values, d.Code)
		}
		return strings.Join(values, ",")
	}
	if !strings.Contains(codes(view), "E-PARSE-STRING") {
		t.Fatal("missing first syntax diagnostic")
	}
	secondError, err := os.ReadFile("../../testdata/fixtures/v3-parity/err-unbalanced.bean")
	if err != nil {
		t.Fatal(err)
	}
	write(secondError)
	awaitState(t, s, func(state desktop.State) bool {
		return !state.Reloading && state.Revision > failed.Revision && !state.Valid
	})
	view, err = s.Read(desktop.Request{View: "journal"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(codes(view), "E-PARSE-STRING") || !strings.Contains(codes(view), "E-EVAL-UNBALANCED") {
		t.Fatalf("the second error must replace the first diagnostics: %s", codes(view))
	}
	write(contents)
	repaired := awaitState(t, s, func(state desktop.State) bool { return state.Valid && !state.Reloading })
	if repaired.BuildCount != 4 {
		t.Fatalf("each stable edit must build once; got %d", repaired.BuildCount)
	}
}

func TestSessionSharesOneSnapshotAcrossViews(t *testing.T) {
	path, _ := filepath.Abs("../../testdata/fixtures/native/cashflow.bean")
	s, err := desktop.Open(path, snapshot.WatchOptions{PollInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tree, err := s.Read(desktop.Request{View: "balance_sheet"})
	if err != nil {
		t.Fatal(err)
	}
	journal, err := s.Read(desktop.Request{View: "journal"})
	if err != nil {
		t.Fatal(err)
	}
	if tree.State.SnapshotID == "" || tree.State.SnapshotID != journal.State.SnapshotID {
		t.Fatal("views must use the same published snapshot")
	}
	if journal.Journal.TotalCount != 2 || journal.Summary.TransactionsCount != 2 {
		t.Fatalf("unexpected transaction count: %+v", journal)
	}
	state, err := s.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if state.BuildCount != 1 {
		t.Fatalf("read and poll rebuilt the ledger: %d builds", state.BuildCount)
	}
	s.Close()
	if _, err := s.Read(desktop.Request{View: "journal"}); err == nil {
		t.Fatal("closed sessions must reject reads")
	}
}
