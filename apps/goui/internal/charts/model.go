// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package charts

import (
	"math"
	"sort"

	"github.com/egoist/mygo/ui"
)

// Point distinguishes an available zero from a missing measurement. Value is
// for drawing only; Exact is the text shown in the standard table.
type Point struct {
	Label     string
	Value     float64
	Exact     string
	Detail    string
	Available bool
	Segment   int
}

type Series struct {
	Label  string
	Color  ui.Color
	Points []Point
}

type valueRange struct{ min, max float64 }

func exactTextLocalized(point Point, tr Translate) string {
	if point.Exact != "" {
		return point.Exact
	}
	if point.Available {
		return chartText(tr, "可用")
	}
	return chartText(tr, "缺失")
}

func tooltipTextLocalized(point Point, tr Translate) string {
	text := exactTextLocalized(point, tr)
	if point.Detail != "" {
		text += " · " + point.Detail
	}
	return text
}

func chartText(tr Translate, source string) string {
	if tr == nil {
		return source
	}
	return tr(source)
}

func chartSeparator(tr Translate) string {
	separator := chartText(tr, "：")
	if separator == ":" {
		return ": "
	}
	return separator
}

func pointForLabel(points []Point, label string) (Point, bool) {
	for _, point := range points {
		if point.Label == label {
			return point, true
		}
	}
	return Point{}, false
}

func pointRange(points []Point) valueRange {
	lo, hi, found := 0.0, 0.0, false
	for _, point := range points {
		if !valid(point) {
			continue
		}
		if !found {
			lo, hi, found = point.Value, point.Value, true
		} else {
			lo, hi = min(lo, point.Value), max(hi, point.Value)
		}
	}
	if !found {
		return valueRange{-1, 1}
	}
	lo, hi = min(lo, 0), max(hi, 0)
	if lo == hi {
		return valueRange{-1, 1}
	}
	pad := (hi - lo) * 0.08
	return valueRange{lo - pad, hi + pad}
}
func valid(point Point) bool {
	return point.Available && !math.IsNaN(point.Value) && !math.IsInf(point.Value, 0)
}
func pointLabels(series []Series) []string {
	set := make(map[string]struct{})
	for _, item := range series {
		for _, point := range item.Points {
			set[point.Label] = struct{}{}
		}
	}
	labels := make([]string, 0, len(set))
	for label := range set {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels
}
func inset(r ui.Rect, left, top, right, bottom float32) ui.Rect {
	return ui.Rect{X: r.X + left, Y: r.Y + top, W: max(r.W-left-right, 1), H: max(r.H-top-bottom, 1)}
}
func xAt(r ui.Rect, index, count int) float32 {
	if count < 2 {
		return r.X + r.W/2
	}
	return r.X + float32(index)*r.W/float32(count-1)
}
func yAt(r ui.Rect, bounds valueRange, value float64) float32 {
	if bounds.max == bounds.min {
		return r.Y + r.H/2
	}
	normalized := (value - bounds.min) / (bounds.max - bounds.min)
	return r.Y + r.H - float32(normalized)*r.H
}
