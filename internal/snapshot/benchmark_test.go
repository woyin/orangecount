// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func BenchmarkBuildAndReload(b *testing.B) {
	dir := b.TempDir()
	entry := filepath.Join(dir, "main.bean")
	data := []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n")
	if err := os.WriteFile(entry, data, 0o600); err != nil {
		b.Fatal(err)
	}
	initial := Build(entry)
	if initial.Snapshot == nil {
		b.Fatal(initial.Diagnostics)
	}
	store := NewStore(initial.Snapshot)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if result := store.Reload(entry, BuildOptions{}); result.Snapshot == nil {
			b.Fatal(result.Diagnostics)
		}
	}
}

// BenchmarkWatchSignature measures one stat-only patrol tick over a single
// ~5.6 MB ledger file (the ADR-0044 reference shape). The pre-ADR patrol
// re-read and hashed all contents on every tick: ~6.5 ms on this shape.
func BenchmarkWatchSignature(b *testing.B) {
	dir := b.TempDir()
	entry := filepath.Join(dir, "main.bean")
	line := "2010-01-01 * \"payee\" \"narration\"\n  Assets:Cash -1 USD\n  Expenses:Food 1 USD\n"
	if err := os.WriteFile(entry, []byte(strings.Repeat(line, 82000)), 0o600); err != nil {
		b.Fatal(err)
	}
	patrol := newGraphPatrol(entry)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = patrol.signature()
	}
}
