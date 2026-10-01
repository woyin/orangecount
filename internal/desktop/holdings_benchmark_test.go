// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop_test

import (
	"os"
	"testing"

	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"time"
)

// Generate the fixed public input with make mac-benchmark first. No private
// input override is accepted; setup is excluded from per-read timing.
func BenchmarkNativeHoldingsProjection(b *testing.B) {
	path := "../../build/macos-benchmark/usd.bean"
	if _, err := os.Stat(path); err != nil {
		b.Skip("generate the public benchmark fixture first")
	}
	s, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3, Watch: snapshot.WatchOptions{PollInterval: time.Hour}})
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	request := desktop.Request{View: "holdings", Account: "Assets:Broker:AAPL", Currency: "native"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		view, err := s.Read(request)
		if err != nil || !view.State.Valid || len(view.Holdings.Groups) != 1 || view.Holdings.Groups[0].LotsPage.TotalCount != 1500 {
			b.Fatal("benchmark must read the complete valid 1500-lot domain")
		}
	}
}
