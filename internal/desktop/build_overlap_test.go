//go:build !windows

// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orangecount/internal/desktop"
	"orangecount/internal/snapshot"
)

// A FIFO makes the source-load overlap deterministic: the child has already
// been read when the last include's writer opens, but the build cannot finish
// until that writer closes. No build hooks or timing guesses are needed.
func TestEditDuringInitialBuildIsNotLost(t *testing.T) {
	dir := t.TempDir()
	entry, child, gate := filepath.Join(dir, "main.bean"), filepath.Join(dir, "child.bean"), filepath.Join(dir, "gate.bean")
	if err := exec.Command("mkfifo", gate).Run(); err != nil {
		t.Skipf("FIFO unavailable: %v", err)
	}
	base, err := os.ReadFile("../../testdata/fixtures/native/cashflow.bean")
	if err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{entry: "include \"child.bean\"\ninclude \"gate.bean\"\n", child: string(base)} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	opened := make(chan *desktop.Session, 1)
	go func() {
		s, _ := desktop.Open(entry, snapshot.WatchOptions{PollInterval: 10 * time.Millisecond, Debounce: 10 * time.Millisecond})
		opened <- s
	}()
	writerReady := make(chan *os.File, 1)
	go func() { file, _ := os.OpenFile(gate, os.O_WRONLY, 0600); writerReady <- file }()
	var writer *os.File
	select {
	case writer = <-writerReady:
	case <-time.After(3 * time.Second):
		t.Fatal("build did not reach final include")
	}
	if writer == nil {
		t.Fatal("cannot open FIFO writer")
	}
	defer writer.Close()
	if err := os.WriteFile(child, []byte(strings.ReplaceAll(string(base), "Salary\"", "Latest\"")), 0600); err != nil {
		t.Fatal(err)
	}
	// Replace the FIFO path with an empty ordinary file for subsequent reloads.
	if err := os.WriteFile(gate+".new", nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(gate+".new", gate); err != nil {
		t.Fatal(err)
	}
	// Send a real comment through the pipe and check teardown errors. A
	// stalled FIFO must fail this test promptly rather than hang the suite.
	if _, err := writer.WriteString("; initial build gate released\n"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	var s *desktop.Session
	select {
	case s = <-opened:
	case <-time.After(3 * time.Second):
		t.Fatal("initial build did not finish after releasing the FIFO")
	}
	if s == nil {
		t.Fatal("opening failed")
	}
	defer s.Close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		view, err := s.Read(desktop.Request{View: "journal", Text: "Latest"})
		if err == nil && view.Journal != nil && view.Journal.TotalCount == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("an include edit made during the initial build was lost")
}
