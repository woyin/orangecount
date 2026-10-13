// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package charts

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/i18n"
)

func TestMissingSplitsLineWhileZeroRemainsAvailable(t *testing.T) {
	points := []Point{
		{Label: "Jan", Value: 10, Exact: "10.00 USD", Available: true},
		{Label: "Feb", Value: 0, Exact: "0.00 USD", Available: true},
		{Label: "Mar", Exact: "No rate", Available: false},
		{Label: "Apr", Value: -5, Exact: "-5.00 USD", Available: true},
	}
	segments := splitSegments(points)
	if len(segments) != 2 || len(segments[0]) != 2 || len(segments[1]) != 1 {
		t.Fatalf("line segments = %#v", segments)
	}
	if !segments[0][1].Available || segments[0][1].Value != 0 {
		t.Fatalf("available zero was lost: %#v", segments[0][1])
	}
	got := pointRange(points)
	if got.min >= -5 || got.max <= 10 {
		t.Fatalf("range %v does not include padded negative and positive values", got)
	}
}

func TestExplicitSourceSegmentStartsANewLine(t *testing.T) {
	points := []Point{
		{Label: "First source run", Value: 10, Exact: "10 USD", Available: true, Segment: 0},
		{Label: "Next source run", Value: 20, Exact: "20 USD", Available: true, Segment: 1},
	}
	if got := splitSegments(points); len(got) != 2 || len(got[0]) != 1 || len(got[1]) != 1 {
		t.Fatalf("explicit source segments were connected: %#v", got)
	}
	if got := decimatedLineSegments(points, 10); len(got) != 2 || len(got[0]) != 1 || len(got[1]) != 1 {
		t.Fatalf("decimated rendering connected explicit source segments: %#v", got)
	}
}

func TestLayoutsReturnPlotGeometryRangesAndAlignedSamples(t *testing.T) {
	viewport := ui.Rect{X: 10, Y: 20, W: 200, H: 100}
	line := layoutLine([]Point{
		{Label: "first", Value: 100, Available: true},
		{Label: "missing", Available: false},
		{Label: "last", Value: -50, Available: true},
	}, viewport)
	if line.plot != (ui.Rect{X: 22, Y: 32, W: 176, H: 74}) {
		t.Fatalf("line plot = %#v", line.plot)
	}
	if line.values != (valueRange{min: -62, max: 112}) {
		t.Fatalf("line range = %#v", line.values)
	}
	if len(line.segments) != 2 || len(line.segments[0]) != 1 || line.segments[0][0] != 0 || line.segments[1][0] != 2 {
		t.Fatalf("line layout lost the missing-value break: %#v", line.segments)
	}

	grouped := layoutGroupedBars([]Series{
		{Label: "Income", Points: []Point{
			{Label: "B", Value: 20, Available: true},
			{Label: "A", Value: 0, Available: true},
		}},
		{Label: "Expenses", Points: []Point{
			{Label: "A", Value: -10, Available: true},
		}},
	}, viewport)
	if len(grouped.labels) != 2 || grouped.labels[0] != "A" || grouped.labels[1] != "B" {
		t.Fatalf("grouped labels = %#v", grouped.labels)
	}
	if grouped.labelIndex["A"] != 0 || grouped.labelIndex["B"] != 1 {
		t.Fatalf("grouped label index = %#v", grouped.labelIndex)
	}
	if len(grouped.samples) != 2 || grouped.samples[0] != 0 || grouped.samples[1] != 1 {
		t.Fatalf("grouped layout omitted a period: %#v", grouped.samples)
	}
	if grouped.values.min != -12.4 || grouped.values.max != 22.4 {
		t.Fatalf("grouped range = %#v", grouped.values)
	}
	if grouped.groupWidth != 88 || grouped.barWidth < 27.27 || grouped.barWidth > 27.29 {
		t.Fatalf("group/bar width = %v/%v", grouped.groupWidth, grouped.barWidth)
	}
}

func TestLayoutsUseTheCurrentViewportForSampling(t *testing.T) {
	points := make([]Point, 1_000)
	labels := make([]string, len(points))
	series := make([]Point, len(points))
	for index := range points {
		points[index] = Point{Label: fmt.Sprintf("period-%04d", index), Value: float64(index), Available: true}
		labels[index] = points[index].Label
		series[index] = points[index]
	}
	narrow := layoutLine(points, ui.Rect{W: 100, H: 100})
	wide := layoutLine(points, ui.Rect{W: 300, H: 100})
	if len(wide.segments[0]) <= len(narrow.segments[0]) {
		t.Fatalf("line sampling did not respond to viewport width: narrow=%d wide=%d", len(narrow.segments[0]), len(wide.segments[0]))
	}
	narrowBars := layoutGroupedBars([]Series{{Points: series}}, ui.Rect{W: 100, H: 100})
	wideBars := layoutGroupedBars([]Series{{Points: series}}, ui.Rect{W: 300, H: 100})
	if len(wideBars.samples) <= len(narrowBars.samples) {
		t.Fatalf("grouped sampling did not respond to viewport width: narrow=%d wide=%d", len(narrowBars.samples), len(wideBars.samples))
	}
}

func TestLineAndGroupedHitUseTheirPlotGeometry(t *testing.T) {
	plot := ui.Rect{X: 12, Y: 8, W: 100, H: 50}
	for _, test := range []struct {
		name string
		got  int
		want int
	}{
		{name: "line first", got: lineIndexAt(plot, 3, plot.X), want: 0},
		{name: "line middle", got: lineIndexAt(plot, 3, plot.X+plot.W/2), want: 1},
		{name: "line last", got: lineIndexAt(plot, 3, plot.X+plot.W), want: 2},
		{name: "line clamps", got: lineIndexAt(plot, 3, plot.X+plot.W*2), want: 2},
		{name: "group first cell", got: groupedIndexAt(plot, 4, plot.X+12.5), want: 0},
		{name: "group second cell", got: groupedIndexAt(plot, 4, plot.X+37.5), want: 1},
		{name: "group final edge", got: groupedIndexAt(plot, 4, plot.X+plot.W), want: 3},
		{name: "empty line", got: lineIndexAt(plot, 0, plot.X), want: -1},
		{name: "empty group", got: groupedIndexAt(plot, 0, plot.X), want: -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("hit index = %d, want %d", test.got, test.want)
			}
		})
	}
}

func TestGroupedBarsRendersSeriesLegend(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		GroupedBars(c, []Series{
			{Label: "Income", Color: ui.RGB(40, 145, 100), Points: []Point{{Label: "Jan", Value: 10, Exact: "10 USD", Available: true}}},
			{Label: "Expenses", Color: ui.RGB(210, 80, 76), Points: []Point{{Label: "Jan", Value: -3, Exact: "-3 USD", Available: true}}},
		})
	}, 500, 300)
	for _, label := range []string{"Income", "Expenses", "分组柱状图"} {
		if !tester.HasText(label) {
			t.Errorf("grouped chart legend/identity omits %q: %q", label, tester.Texts())
		}
	}
}

func TestLocalizedChartsAndExactTableUseEnglishLabels(t *testing.T) {
	translate := func(source string) string { return i18n.Text(i18n.English, source) }
	series := []Series{{Label: "Net worth", Points: []Point{
		{Label: "2026-01", Value: 0, Exact: "0 USD", Available: true},
		{Label: "2026-02", Exact: "", Available: false},
	}}}
	var tableState ui.ListState
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			LineLocalized(c, series[0], translate)
			GroupedBarsLocalized(c, []Series{{Label: "Income", Points: series[0].Points}}, translate)
			ExactTableLocalized(c, &tableState, series, translate)
		})
	}, 760, 900)
	for _, label := range []string{"Net worth line chart", "Grouped bar chart"} {
		if _, ok := tester.Find(label); !ok {
			t.Errorf("localized chart label %q not found in %q", label, tester.Texts())
		}
	}
	for _, label := range []string{"Period", "Missing"} {
		if !tester.HasText(label) {
			t.Errorf("localized exact table omitted %q: %q", label, tester.Texts())
		}
	}
	if tester.HasText("期间") || tester.HasText("分组柱状图") {
		t.Fatalf("localized chart/table leaked Chinese labels: %q", tester.Texts())
	}
}

func TestExactTableRowsAlignByPeriodAcrossSparseSeries(t *testing.T) {
	series := []Series{
		{Label: "USD", Points: []Point{
			{Label: "2026-01", Value: 1, Exact: "1 USD", Available: true},
			{Label: "2026-03", Value: 3, Exact: "3 USD", Available: true},
		}},
		{Label: "EUR", Points: []Point{
			{Label: "2026-02", Value: 2, Exact: "2 EUR", Available: true},
			{Label: "2026-03", Value: 4, Exact: "4 EUR", Available: true},
		}},
	}
	labels := pointLabels(series)
	if len(labels) != 3 || labels[0] != "2026-01" || labels[1] != "2026-02" || labels[2] != "2026-03" {
		t.Fatalf("grouped chart period labels = %v", labels)
	}
	rows := exactRows(series)
	if len(rows) != 3 || rows[0].label != "2026-01" || rows[1].label != "2026-02" || rows[2].label != "2026-03" {
		t.Fatalf("union of periods = %+v", rows)
	}
	if _, ok := rows[0].points[1]; ok {
		t.Fatal("EUR February value was incorrectly aligned to January")
	}
	if got := rows[1].points[1].Exact; got != "2 EUR" {
		t.Fatalf("EUR February value = %q", got)
	}
}

func TestExactTableRendersSparseSeriesBySharedPeriod(t *testing.T) {
	series := []Series{
		{Label: "USD", Points: []Point{
			{Label: "2026-01", Exact: "1 USD", Available: true},
			{Label: "2026-03", Exact: "3 USD", Available: true},
		}},
		{Label: "EUR", Points: []Point{
			{Label: "2026-02", Exact: "2 EUR", Available: true},
			{Label: "2026-03", Exact: "4 EUR", Available: true},
		}},
	}
	var state ui.ListState
	tester := ui.NewTester(func(c *ui.Context) {
		ExactTable(c, &state, series)
	}, 700, 300)
	for _, want := range []string{"2026-01", "2026-02", "2026-03", "1 USD", "2 EUR", "3 USD", "4 EUR"} {
		if !tester.HasText(want) {
			t.Errorf("exact table did not render %q in its period-aligned rows: %q", want, tester.Texts())
		}
	}
}

func TestLineBarsAndExactTableRenderAtOneAndTwoScale(t *testing.T) {
	series := []Series{
		{Label: "Income", Color: ui.RGB(30, 145, 90), Points: []Point{
			{Label: "Jan", Value: 1000, Exact: "1000.00 USD", Available: true},
			{Label: "Feb", Value: 0, Exact: "0.00 USD", Available: true},
		}},
		{Label: "Expenses", Color: ui.RGB(205, 65, 70), Points: []Point{
			{Label: "Jan", Value: -300, Exact: "-300.00 USD", Available: true},
			{Label: "Feb", Exact: "Missing price", Available: false},
		}},
	}
	for _, scale := range []float32{1, 2} {
		state := ui.ListState{}
		tester := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				Line(c, Series{Label: "Net", Color: ui.RGB(42, 119, 214), Points: []Point{
					{Label: "Jan", Value: 700, Exact: "700.00 USD", Available: true},
					{Label: "Feb", Value: -300, Exact: "-300.00 USD", Available: true},
					{Label: "Mar", Value: 0, Exact: "0.00 USD", Available: true},
					{Label: "Apr", Available: false},
				}})
				GroupedBars(c, series)
				ExactTable(c, &state, series)
			})
		}, 640, 720)
		tester.SetScale(scale)
		if !tester.HasText("1000.00 USD") || !tester.HasText("0.00 USD") || !tester.HasText("Missing price") {
			t.Fatalf("scale %.0f exact values not available in standard table: %q", scale, tester.Texts())
		}
		light := tester.Image()
		if len(light.Pix) == 0 {
			t.Fatalf("scale %.0f produced no rendered image", scale)
		}
		lightPixels := append([]byte(nil), light.Pix...)
		tester.SetDark(true)
		if !tester.HasText("1000.00 USD") || !tester.HasText("Missing price") {
			t.Fatalf("scale %.0f dark appearance lost exact table values: %q", scale, tester.Texts())
		}
		darkPixels := tester.Image().Pix
		if len(darkPixels) == 0 || bytes.Equal(lightPixels, darkPixels) {
			t.Fatalf("scale %.0f did not repaint charts for dark appearance", scale)
		}
	}
}

func TestLineHoverShowsTheNearestExactMeasurement(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		Line(c, Series{Label: "净资产", Points: []Point{
			{Label: "2026-01", Value: 10, Exact: "10.00 USD", Available: true},
			{Label: "2026-02", Value: 0, Exact: "0.00 USD", Available: true},
			{Label: "2026-03", Value: -5, Exact: "-5.00 USD", Available: true},
		}})
	}, 640, 360)
	chart, ok := tester.Find("净资产折线图")
	if !ok {
		t.Fatal("line chart has no accessible label")
	}
	tester.Move(chart.X+chart.W/2, chart.Y+chart.H/2)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	if !tester.HasText("2026-02 · 0.00 USD") {
		t.Fatalf("line hover did not show the exact zero measurement: %q", tester.Texts())
	}
}

func TestLineHoverKeepsQuoteDetailWithExactValue(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		Line(c, Series{Label: "净资产", Points: []Point{{
			Label: "2000-02", Value: 104, Exact: "104 USD", Detail: "采用报价 EUR/USD 2 · 2000-02-15 · quote.bean:13", Available: true,
		}}})
	}, 640, 360)
	chart, ok := tester.Find("净资产折线图")
	if !ok {
		t.Fatal("line chart has no accessible label")
	}
	tester.Move(chart.X+chart.W/2, chart.Y+chart.H/2)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	for _, want := range []string{"2000-02 · 104 USD · 采用报价 EUR/USD 2", "2000-02-15", "quote.bean:13"} {
		if !tester.HasText(want) {
			t.Errorf("line tooltip omitted quote provenance %q: %q", want, tester.Texts())
		}
	}
}

func TestGroupedBarsHoverShowsExactAndMissingMeasurements(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		GroupedBars(c, []Series{
			{Label: "收入", Points: []Point{
				{Label: "2026-01", Value: 1000, Exact: "1000.00 USD", Available: true},
				{Label: "2026-02", Value: 0, Exact: "0.00 USD", Available: true},
			}},
			{Label: "支出", Points: []Point{
				{Label: "2026-01", Value: -300, Exact: "300.00 USD", Available: true},
				{Label: "2026-02", Exact: "缺少报价", Available: false},
			}},
		})
	}, 640, 360)
	chart, ok := tester.Find("分组柱状图")
	if !ok {
		t.Fatal("grouped bar chart has no accessible label")
	}
	tester.Move(chart.X+chart.W-12, chart.Y+chart.H/2)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	if !tester.HasText("2026-02") || !tester.HasText("收入：0.00 USD") || !tester.HasText("支出：缺少报价") {
		t.Fatalf("grouped bar hover did not preserve exact or unavailable values: %q", tester.Texts())
	}
}

func TestHoverHitUsesElementLocalCoordinatesForOffsetCharts(t *testing.T) {
	points := []Point{
		{Label: "2026-01", Value: 10, Exact: "10 USD", Available: true},
		{Label: "2026-02", Value: 20, Exact: "20 USD", Available: true},
		{Label: "2026-03", Value: 30, Exact: "30 USD", Available: true},
	}
	t.Run("line", func(t *testing.T) {
		tester := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Padding(48).Children(func() {
				Line(c, Series{Label: "Net worth", Points: points})
			})
		}, 760, 360)
		chart, ok := tester.Find("Net worth折线图")
		if !ok || chart.X < 40 {
			t.Fatalf("offset line chart was not laid out as expected: %+v, found=%v", chart, ok)
		}
		tester.Move(chart.X+chart.W/2, chart.Y+chart.H/2)
		time.Sleep(650 * time.Millisecond)
		tester.Frame()
		if !tester.HasText("2026-02 · 20 USD") {
			t.Fatalf("offset line chart did not hit its middle point: %q", tester.Texts())
		}
	})
	t.Run("grouped bars", func(t *testing.T) {
		tester := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Padding(48).Children(func() {
				GroupedBars(c, []Series{
					{Label: "Income", Points: points},
					{Label: "Expenses", Points: points},
				})
			})
		}, 760, 360)
		chart, ok := tester.Find("分组柱状图")
		if !ok || chart.X < 40 {
			t.Fatalf("offset grouped-bar chart was not laid out as expected: %+v, found=%v", chart, ok)
		}
		tester.Move(chart.X+chart.W/2, chart.Y+chart.H/2)
		time.Sleep(650 * time.Millisecond)
		tester.Frame()
		if !tester.HasText("2026-02") || !tester.HasText("Income：20 USD") {
			t.Fatalf("offset grouped-bar chart did not hit its middle point: %q", tester.Texts())
		}
	})
}

func TestLineKeyboardSelectsExactPeriod(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		Line(c, Series{Label: "净资产", Points: []Point{
			{Label: "2026-01", Value: 10, Exact: "10.00 USD", Available: true},
			{Label: "2026-02", Value: 0, Exact: "0.00 USD", Available: true},
			{Label: "2026-03", Value: -5, Exact: "-5.00 USD", Available: true},
		}})
	}, 640, 360)
	tester.Key(0, ui.KeyTab)
	if !tester.Focused("净资产折线图") {
		t.Fatal("Tab did not focus the line chart")
	}
	tester.Key(0, ui.KeyHome)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	if !tester.HasText("2026-01 · 10.00 USD") {
		t.Fatalf("Home did not select the first exact line measurement: %q", tester.Texts())
	}
	tester.Key(0, ui.KeyRight)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	if !tester.HasText("2026-02 · 0.00 USD") {
		t.Fatalf("Right did not select the next exact line measurement: %q", tester.Texts())
	}
	tester.Key(0, ui.KeyEnd)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	if !tester.HasText("2026-03 · -5.00 USD") {
		t.Fatalf("End did not select the last exact line measurement: %q", tester.Texts())
	}
}

func TestGroupedBarsKeyboardSelectsExactAndMissingPeriod(t *testing.T) {
	tester := ui.NewTester(func(c *ui.Context) {
		GroupedBars(c, []Series{
			{Label: "收入", Points: []Point{
				{Label: "2026-01", Value: 1000, Exact: "1000.00 USD", Available: true},
				{Label: "2026-02", Value: 0, Exact: "0.00 USD", Available: true},
			}},
			{Label: "支出", Points: []Point{
				{Label: "2026-01", Value: -300, Exact: "300.00 USD", Available: true},
				{Label: "2026-02", Exact: "缺少报价", Available: false},
			}},
		})
	}, 640, 360)
	tester.Key(0, ui.KeyTab)
	if !tester.Focused("分组柱状图") {
		t.Fatal("Tab did not focus the grouped bar chart")
	}
	tester.Key(0, ui.KeyHome)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	if !tester.HasText("2026-01") || !tester.HasText("收入：1000.00 USD") || !tester.HasText("支出：300.00 USD") {
		t.Fatalf("Home did not select the first exact grouped-bar measurements: %q", tester.Texts())
	}
	tester.Key(0, ui.KeyEnd)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	if !tester.HasText("2026-02") || !tester.HasText("收入：0.00 USD") || !tester.HasText("支出：缺少报价") {
		t.Fatalf("End did not preserve the last exact and missing grouped-bar measurements: %q", tester.Texts())
	}
}

func TestLineSamplingPreservesExtremaEndpointsAndMissingBreaks(t *testing.T) {
	points := make([]Point, 10_000)
	for index := range points {
		points[index] = Point{Label: fmt.Sprintf("period-%05d", index), Value: float64(index % 7), Available: true}
	}
	points[4_321].Value = 10_000
	points[4_324].Value = -9_000
	points[5_000].Available = false
	segments := decimatedLineSegments(points, 400)
	if len(segments) != 2 {
		t.Fatalf("sampled line segments = %d, want the two sides of the unavailable point", len(segments))
	}
	if len(segments[0])+len(segments[1]) > 400 {
		t.Fatalf("sampled %d points, limit is 400", len(segments[0])+len(segments[1]))
	}
	for _, index := range []int{0, 4_321, 4_324, 4_999, 5_001, 9_999} {
		if !containsIndex(segments, index) {
			t.Errorf("sample omitted endpoint or local extreme %d", index)
		}
	}
	if containsIndex(segments, 5_000) {
		t.Fatal("sampled line included the unavailable measurement")
	}
}

func TestLineSamplingKeepsManyGapsWithoutPaintingEveryPoint(t *testing.T) {
	const segmentCount, segmentLength = 20, 100
	points := make([]Point, segmentCount*(segmentLength+1)-1)
	for segment := 0; segment < segmentCount; segment++ {
		start := segment * (segmentLength + 1)
		for offset := 0; offset < segmentLength; offset++ {
			points[start+offset] = Point{Label: fmt.Sprintf("%d-%d", segment, offset), Value: float64(offset % 5), Available: true}
		}
		points[start+43].Value = 1_000 + float64(segment)
		points[start+47].Value = -1_000 - float64(segment)
	}
	segments := decimatedLineSegments(points, 40)
	if len(segments) != segmentCount {
		t.Fatalf("sampled %d segments, want %d", len(segments), segmentCount)
	}
	count := 0
	for segment, indices := range segments {
		count += len(indices)
		if len(indices) > 4 {
			t.Errorf("segment %d retained %d points, want at most its endpoint and extrema samples", segment, len(indices))
		}
		start := segment * (segmentLength + 1)
		for _, index := range []int{start, start + 43, start + 47, start + segmentLength - 1} {
			if !containsIndex([][]int{indices}, index) {
				t.Errorf("segment %d omitted endpoint or extreme %d", segment, index)
			}
		}
	}
	if count >= len(points)/2 {
		t.Fatalf("dense-gap sampling retained %d of %d points", count, len(points))
	}
}

func TestGroupedSamplingPreservesPerSeriesExtrema(t *testing.T) {
	labels := make([]string, 10_000)
	income := make([]Point, len(labels))
	expenses := make([]Point, len(labels))
	for index := range labels {
		labels[index] = fmt.Sprintf("period-%05d", index)
		income[index] = Point{Label: labels[index], Value: float64(index % 3), Available: true}
		expenses[index] = Point{Label: labels[index], Value: -float64(index % 5), Available: true}
	}
	income[3_777].Value = 12_000
	expenses[8_888].Value = -15_000
	indices := decimatedGroupIndices(labels, []Series{{Points: income}, {Points: expenses}}, 400)
	if len(indices) > 400 {
		t.Fatalf("sampled %d grouped periods, limit is 400", len(indices))
	}
	for _, index := range []int{0, 3_777, 8_888, len(labels) - 1} {
		if !containsValue(indices, index) {
			t.Errorf("grouped sample omitted endpoint or series extreme %d", index)
		}
	}
}

func TestGroupedBarWidthRemainsVisibleAfterDenseSampling(t *testing.T) {
	if got := groupedBarWidth(0.05, 2); got < 0.5 {
		t.Fatalf("dense grouped bar width = %v, want at least a half DIP", got)
	}
	if got := groupedBarWidth(100, 2); got != 31 {
		t.Fatalf("ordinary grouped bar width = %v, want 31", got)
	}
}

func TestDenseLineHoverHitsOriginalUnsampledPoint(t *testing.T) {
	points := make([]Point, 10_000)
	for index := range points {
		points[index] = Point{Label: fmt.Sprintf("period-%05d", index), Value: float64(index), Exact: fmt.Sprintf("exact-%05d", index), Available: true}
	}
	tester := ui.NewTester(func(c *ui.Context) {
		Line(c, Series{Label: "密集序列", Points: points})
	}, 640, 360)
	chart, ok := tester.Find("密集序列折线图")
	if !ok {
		t.Fatal("dense line chart was not built")
	}
	tester.Move(chart.X+chart.W/2, chart.Y+chart.H/2)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	if !tester.HasText("period-05000 · exact-05000") {
		t.Fatalf("dense line hit was based on the sampled points instead of the original series: %q", tester.Texts())
	}
	if len(tester.Image().Pix) == 0 {
		t.Fatal("dense line chart produced no rendered image")
	}
}

func TestDenseGroupedBarsHoverHitsOriginalUnsampledPeriod(t *testing.T) {
	labels := make([]string, 10_000)
	income := make([]Point, len(labels))
	expenses := make([]Point, len(labels))
	for index := range labels {
		labels[index] = fmt.Sprintf("period-%05d", index)
		income[index] = Point{Label: labels[index], Value: 1, Exact: fmt.Sprintf("income-%05d", index), Available: true}
		expenses[index] = Point{Label: labels[index], Value: -1, Exact: fmt.Sprintf("expense-%05d", index), Available: true}
	}
	series := []Series{{Label: "收入", Points: income}, {Label: "支出", Points: expenses}}
	sampled := decimatedGroupIndices(labels, series, 1_200)
	isSampled := make([]bool, len(labels))
	for _, index := range sampled {
		isSampled[index] = true
	}
	target := 5_000
	for target < len(labels)-1 && isSampled[target] {
		target++
	}
	if isSampled[target] {
		t.Fatal("test could not find an unsampled period")
	}
	tester := ui.NewTester(func(c *ui.Context) {
		GroupedBars(c, series)
	}, 640, 360)
	chart, ok := tester.Find("分组柱状图")
	if !ok {
		t.Fatal("dense grouped bar chart was not built")
	}
	plotWidth := chart.W - 24
	x := chart.X + 12 + float32(target)*plotWidth/float32(len(labels)-1)
	tester.Move(x, chart.Y+chart.H/2)
	time.Sleep(650 * time.Millisecond)
	tester.Frame()
	want := labels[target] + "\n收入：income-" + fmt.Sprintf("%05d", target) + "\n支出：expense-" + fmt.Sprintf("%05d", target)
	if !tester.HasText(want) {
		t.Fatalf("dense grouped hit was based on sampled periods instead of the original series: %q; want %q", tester.Texts(), want)
	}
}

func containsIndex(segments [][]int, want int) bool {
	for _, segment := range segments {
		if containsValue(segment, want) {
			return true
		}
	}
	return false
}

func containsValue(values []int, want int) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
