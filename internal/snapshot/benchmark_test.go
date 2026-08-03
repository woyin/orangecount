// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package snapshot

import (
	"os"
	"path/filepath"
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
