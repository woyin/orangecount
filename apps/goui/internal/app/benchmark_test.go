// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/charts"
	"orangecount/apps/goui/internal/workspace"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
)

const benchmarkWindowWidth, benchmarkWindowHeight = 1180, 780

func BenchmarkGouiHeadlessWelcomeFrame(b *testing.B) {
	model := NewModel(true)
	tester := ui.NewTester(model.View, benchmarkWindowWidth, benchmarkWindowHeight)
	benchmarkFrames(b, tester)
}

func BenchmarkGouiHeadlessJournalPageFrame52500(b *testing.B) {
	path := benchmarkLedgerPath(b)
	session, err := (workspace.DesktopEngine{}).Open(context.Background(), path, workspace.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		b.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		b.Fatal(err)
	}
	response, err := session.Read(desktop.Request{
		View: "journal", JournalMode: "transactions", SnapshotID: state.SnapshotID,
		Window: "all", Sort: "descending", Limit: 100, Locale: "zh-CN",
	})
	if err != nil {
		b.Fatal(err)
	}
	if response.Journal == nil || response.Journal.TotalCount < 52_500 {
		b.Fatalf("benchmark ledger has %d journal rows, want at least 52,500", journalRowCount(response))
	}
	model := NewModel(true)
	model.page = pageJournal
	model.journalSelection = -1
	model.workspaceState = workspace.State{
		Phase: workspace.PhaseReady, ActivePath: path,
		Ledger: state, Response: &response,
	}
	tester := ui.NewTester(model.View, benchmarkWindowWidth, benchmarkWindowHeight)
	benchmarkFrames(b, tester)
	b.ReportMetric(float64(response.Journal.TotalCount), "journal-rows")
}

func BenchmarkGouiHeadlessChartFrame10k(b *testing.B) {
	points := make([]charts.Point, 10_000)
	for i := range points {
		points[i] = charts.Point{
			Label: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, i, 0).Format("2006-01"),
			Value: float64((i%97)-48) * 123.45,
			Exact: "12,345.67 USD", Available: i%31 != 0,
		}
	}
	income := charts.Series{Label: "收入", Color: ui.RGB(40, 145, 100), Points: points}
	expenses := charts.Series{Label: "支出", Color: ui.RGB(210, 80, 76), Points: points}
	var tableState ui.ListState
	view := func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			charts.Line(c, charts.Series{Label: "净资产", Color: ui.RGB(42, 119, 214), Points: points})
			charts.GroupedBars(c, []charts.Series{income, expenses})
			charts.ExactTable(c, &tableState, []charts.Series{income, expenses})
		})
	}
	tester := ui.NewTester(view, 900, 600)
	b.ReportMetric(float64(len(points)), "raw-points")
	benchmarkFrames(b, tester)
}

func BenchmarkGouiCoreOpenLedger52500(b *testing.B) {
	path := benchmarkLedgerPath(b)
	b.ReportAllocs()
	samples := make([]time.Duration, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		started := time.Now()
		session, err := (workspace.DesktopEngine{}).Open(context.Background(), path, workspace.OpenOptions{SemanticsProfile: ledger.V3})
		if err != nil {
			b.Fatal(err)
		}
		samples[i] = time.Since(started)
		b.StopTimer()
		session.Close()
		b.StartTimer()
	}
	b.StopTimer()
	reportDurationSamples(b, samples, "open")
}

func BenchmarkGouiCoreReadJournalPage10052500(b *testing.B) {
	path := benchmarkLedgerPath(b)
	session, err := (workspace.DesktopEngine{}).Open(context.Background(), path, workspace.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		b.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil {
		b.Fatal(err)
	}
	request := desktop.Request{
		View: "journal", JournalMode: "transactions", SnapshotID: state.SnapshotID,
		Window: "all", Sort: "descending", Limit: 100, Locale: "zh-CN",
	}
	b.ReportAllocs()
	samples := make([]time.Duration, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		started := time.Now()
		if _, err := session.Read(request); err != nil {
			b.Fatal(err)
		}
		samples[i] = time.Since(started)
	}
	b.StopTimer()
	reportDurationSamples(b, samples, "read")
}

func benchmarkLedgerPath(b *testing.B) string {
	b.Helper()
	path := os.Getenv("GOUI_BENCH_LEDGER")
	if path == "" {
		b.Skip("set GOUI_BENCH_LEDGER to a 52,500-transaction public benchmark ledger")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		b.Fatal(err)
	}
	return absolute
}

func benchmarkFrames(b *testing.B, tester *ui.Tester) {
	samples := make([]time.Duration, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		started := time.Now()
		tester.Frame()
		samples[i] = time.Since(started)
	}
	b.StopTimer()
	reportDurationSamples(b, samples, "frame")
}

func reportDurationSamples(b *testing.B, samples []time.Duration, unit string) {
	if len(samples) == 0 {
		return
	}
	ordered := append([]time.Duration(nil), samples...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	b.ReportMetric(float64(len(ordered)), "samples/"+unit)
	b.ReportMetric(float64(ordered[0]), "min-ns/"+unit)
	b.ReportMetric(float64(percentileDuration(ordered, 0.95)), "p95-ns/"+unit)
	b.ReportMetric(float64(percentileDuration(ordered, 0.99)), "p99-ns/"+unit)
	b.ReportMetric(float64(ordered[len(ordered)-1]), "max-ns/"+unit)
}

func percentileDuration(samples []time.Duration, quantile float64) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	index := int(math.Ceil(quantile*float64(len(samples)))) - 1
	index = max(0, min(index, len(samples)-1))
	return samples[index]
}

func journalRowCount(response desktop.ViewResponse) int {
	if response.Journal == nil {
		return 0
	}
	return response.Journal.TotalCount
}
