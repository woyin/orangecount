// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"strings"
	"testing"

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
