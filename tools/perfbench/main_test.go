// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orangecount/internal/snapshot"
)

func TestGenerateLedgerIsDeterministic(t *testing.T) {
	first := generateLedger(50000)
	second := generateLedger(50000)
	if !bytes.Equal(first, second) {
		t.Fatal("generator produced different ledgers for identical arguments")
	}
}

func TestGeneratedLedgerBuildsCleanly(t *testing.T) {
	entry, cleanup, err := prepareLedger("", 2000, t.TempDir()+"/perfbench.bean")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	result := snapshot.Build(entry)
	if result.Snapshot == nil {
		t.Fatalf("generated ledger failed to build: %v", result.Diagnostics)
	}
}

func TestGeneratedLedgerShapeScalesWithTransactions(t *testing.T) {
	small := generateLedger(100)
	if got := strings.Count(string(small), `"payee `); got != 100 {
		t.Fatalf("payee transactions=%d", got)
	}
	if got := strings.Count(string(small), `"buy `); got != 4 { // transactions/25
		t.Fatalf("buy transactions=%d", got)
	}
}

func TestDetectBeancountReportsInterpreterFailure(t *testing.T) {
	if _, ok := detectBeancount("definitely-not-an-interpreter"); ok {
		t.Fatal("a missing interpreter must not report beancount availability")
	}
	// Any command that exits zero counts as a successful probe; the version is
	// whatever the interpreter printed on stdout.
	if version, ok := detectBeancount("true"); !ok {
		t.Fatal("a zero-exit probe must report availability")
	} else if strings.TrimSpace(version) != "" {
		t.Fatalf("version=%q", version)
	}
}

func TestRunBeancountParsesColdWarmOutput(t *testing.T) {
	fake := filepath.Join(t.TempDir(), "fake-python.sh")
	script := "#!/bin/sh\necho 'interpreter noise'\necho '2.5 1.5'\n"
	if err := os.WriteFile(fake, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	cold, warm, err := runBeancount(fake, "ledger.bean")
	if err != nil {
		t.Fatalf("runBeancount=%v", err)
	}
	if cold != 2500*time.Millisecond || warm != 1500*time.Millisecond {
		t.Fatalf("cold=%s warm=%s", cold, warm)
	}
	// Short or non-numeric output is a hard error, not a zero duration.
	broken := filepath.Join(t.TempDir(), "broken-python.sh")
	if err := os.WriteFile(broken, []byte("#!/bin/sh\necho 'only one line'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runBeancount(broken, "ledger.bean"); err == nil {
		t.Fatal("unparsable output must fail")
	}
	if _, _, err := runBeancount("definitely-not-an-interpreter", "ledger.bean"); err == nil {
		t.Fatal("a failed run must surface the interpreter error")
	}
}

func TestRunOrangeCountTimesValidLedger(t *testing.T) {
	ledger := generateLedger(10)
	entry := filepath.Join(t.TempDir(), "bench.bean")
	if err := os.WriteFile(entry, ledger, 0o600); err != nil {
		t.Fatal(err)
	}
	cold, warm := runOrangeCount(entry)
	if cold <= 0 || warm <= 0 || warm > time.Hour {
		t.Fatalf("cold=%s warm=%s", cold, warm)
	}
}

func TestPrepareLedgerUsesExistingAndWritesOut(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "mine.bean")
	if err := os.WriteFile(existing, []byte("2000-01-01 open Assets:Cash USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, cleanup, err := prepareLedger(existing, 5, "")
	if err != nil || path != existing {
		t.Fatalf("path=%s err=%v", path, err)
	}
	cleanup()
	out := filepath.Join(t.TempDir(), "out.bean")
	path, cleanup, err = prepareLedger("", 5, out)
	if err != nil || path != out {
		t.Fatalf("out path=%s err=%v", path, err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("out file missing: %v", err)
	}
	// An explicit -out path belongs to the caller: cleanup deliberately keeps it.
	cleanup()
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("explicit -out must survive cleanup: %v", err)
	}
}

func TestRunSkipsCompareAndRunsFullPipeline(t *testing.T) {
	out := filepath.Join(t.TempDir(), "gen.bean")
	// Generated ledger, comparison skipped.
	run("", 20, true, "python3", out)
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("generated out missing: %v", err)
	}
	// An existing ledger path is benchmarked untouched.
	existing := filepath.Join(t.TempDir(), "mine.bean")
	if err := os.WriteFile(existing, []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"x\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(existing, 0, true, "python3", "")
}

func TestRunBeancountComparisonWithFakeInterpreter(t *testing.T) {
	fake := filepath.Join(t.TempDir(), "fake-python.sh")
	script := "#!/bin/sh\necho 'beancount 3.2.0 fake'\necho '0.5 0.25'\n"
	if err := os.WriteFile(fake, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	// The comparison leg runs against the fake interpreter end to end.
	run("", 10, false, fake, "")
}
