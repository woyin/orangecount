// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package charts

import (
	"math"
	"sort"

	"github.com/egoist/mygo/ui"
)

type chartIndexRange struct{ start, end int }

const chartSamplesPerDIP = 2

type lineLayout struct {
	plot     ui.Rect
	values   valueRange
	segments [][]int
}

type groupedLayout struct {
	plot       ui.Rect
	values     valueRange
	labels     []string
	labelIndex map[string]int
	samples    []int
	groupWidth float32
	barWidth   float32
}

// layoutLine converts the current viewport into a small Painter-ready result.
// It is pure: callers may retain the result only for this data and viewport.
func layoutLine(points []Point, viewport ui.Rect) lineLayout {
	plot := inset(viewport, 12, 12, 12, 14)
	return lineLayout{
		plot:     plot,
		values:   pointRange(points),
		segments: decimatedLineSegments(points, chartPointLimit(viewport.W)),
	}
}

// layoutGroupedBars aligns each series by its period and samples whole periods
// so rendering and hit testing continue to use the same original labels.
func layoutGroupedBars(series []Series, viewport ui.Rect) groupedLayout {
	labels := pointLabels(series)
	labelIndex := make(map[string]int, len(labels))
	for index, label := range labels {
		labelIndex[label] = index
	}
	all := make([]Point, 0)
	for _, item := range series {
		all = append(all, item.Points...)
	}
	plot := inset(viewport, 12, 12, 12, 14)
	groupWidth := float32(0)
	if len(labels) > 0 {
		groupWidth = plot.W / float32(len(labels))
	}
	return groupedLayout{
		plot:       plot,
		values:     pointRange(all),
		labels:     labels,
		labelIndex: labelIndex,
		samples:    decimatedGroupIndicesWithLabels(labels, series, chartPointLimit(viewport.W), labelIndex),
		groupWidth: groupWidth,
		barWidth:   groupedBarWidth(groupWidth, len(series)),
	}
}

func chartPointLimit(width float32) int {
	return max(4, int(math.Ceil(float64(max(width, 1)*chartSamplesPerDIP))))
}
func groupedBarWidth(groupWidth float32, seriesCount int) float32 {
	if seriesCount <= 0 {
		return 0
	}
	return max(groupWidth*0.62/float32(seriesCount), 0.5)
}

// decimatedLineSegments reduces Painter work while keeping each missing-value
// break independent. Indices remain in the original series so hit testing,
// selection and exact text never use the sampled data as their source.
func decimatedLineSegments(points []Point, limit int) [][]int {
	if len(points) == 0 {
		return nil
	}
	limit = max(limit, 4)
	var ranges []chartIndexRange
	validCount := 0
	for index := 0; index < len(points); {
		if !valid(points[index]) {
			index++
			continue
		}
		start := index
		segment := points[index].Segment
		for index < len(points) && valid(points[index]) && points[index].Segment == segment {
			index++
		}
		ranges = append(ranges, chartIndexRange{start: start, end: index})
		validCount += index - start
	}
	if len(ranges) == 0 {
		return nil
	}
	if validCount <= limit {
		return allLineSegmentIndices(ranges)
	}

	segments := make([][]int, 0, len(ranges))
	for _, segment := range ranges {
		budget := max(4, limit*(segment.end-segment.start)/validCount)
		segments = append(segments, sampleLineSegment(points, segment.start, segment.end, budget))
	}
	return segments
}
func allLineSegmentIndices(ranges []chartIndexRange) [][]int {
	segments := make([][]int, 0, len(ranges))
	for _, segment := range ranges {
		indices := make([]int, segment.end-segment.start)
		for offset := range indices {
			indices[offset] = segment.start + offset
		}
		segments = append(segments, indices)
	}
	return segments
}
func sampleLineSegment(points []Point, start, end, budget int) []int {
	length := end - start
	if length <= budget {
		indices := make([]int, length)
		for offset := range indices {
			indices[offset] = start + offset
		}
		return indices
	}
	bucketCount := max(1, budget/4)
	indices := make([]int, 0, bucketCount*4)
	for bucket := 0; bucket < bucketCount; bucket++ {
		first := start + bucket*length/bucketCount
		last := start + (bucket+1)*length/bucketCount - 1
		minimum, maximum := first, first
		for index := first + 1; index <= last; index++ {
			if points[index].Value < points[minimum].Value {
				minimum = index
			}
			if points[index].Value > points[maximum].Value {
				maximum = index
			}
		}
		candidates := []int{first, minimum, maximum, last}
		sort.Ints(candidates)
		for _, index := range candidates {
			if len(indices) == 0 || indices[len(indices)-1] != index {
				indices = append(indices, index)
			}
		}
	}
	return indices
}
func decimatedGroupIndices(labels []string, series []Series, limit int) []int {
	if len(labels) == 0 {
		return nil
	}
	limit = max(limit, 1)
	if len(labels) <= limit {
		indices := make([]int, len(labels))
		for index := range indices {
			indices[index] = index
		}
		return indices
	}
	labelIndices := make(map[string]int, len(labels))
	for index, label := range labels {
		labelIndices[label] = index
	}
	return decimatedGroupIndicesWithLabels(labels, series, limit, labelIndices)
}
func decimatedGroupIndicesWithLabels(labels []string, series []Series, limit int, labelIndices map[string]int) []int {
	if len(labels) == 0 {
		return nil
	}
	limit = max(limit, 1)
	if len(labels) <= limit {
		indices := make([]int, len(labels))
		for index := range indices {
			indices[index] = index
		}
		return indices
	}
	bucketCount := max(1, limit/(2+2*len(series)))
	type extrema struct {
		minValue, maxValue float64
		minIndex, maxIndex int
		found              bool
	}
	extremes := make([][]extrema, len(series))
	for seriesIndex, item := range series {
		extremes[seriesIndex] = make([]extrema, bucketCount)
		for _, point := range item.Points {
			if !valid(point) {
				continue
			}
			index, ok := labelIndices[point.Label]
			if !ok {
				continue
			}
			bucket := min(index*bucketCount/len(labels), bucketCount-1)
			current := &extremes[seriesIndex][bucket]
			if !current.found {
				*current = extrema{minValue: point.Value, maxValue: point.Value, minIndex: index, maxIndex: index, found: true}
				continue
			}
			if point.Value < current.minValue {
				current.minValue, current.minIndex = point.Value, index
			}
			if point.Value > current.maxValue {
				current.maxValue, current.maxIndex = point.Value, index
			}
		}
	}

	indices := make([]int, 0, bucketCount*(2+2*len(series)))
	for bucket := 0; bucket < bucketCount; bucket++ {
		first := bucket * len(labels) / bucketCount
		last := (bucket+1)*len(labels)/bucketCount - 1
		indices = append(indices, first, last)
		for seriesIndex := range extremes {
			current := extremes[seriesIndex][bucket]
			if current.found {
				indices = append(indices, current.minIndex, current.maxIndex)
			}
		}
	}
	sort.Ints(indices)
	unique := indices[:0]
	for _, index := range indices {
		if len(unique) == 0 || unique[len(unique)-1] != index {
			unique = append(unique, index)
		}
	}
	return unique
}
func splitSegments(points []Point) [][]Point {
	var segments [][]Point
	var current []Point
	currentSegment := 0
	flush := func() {
		if len(current) > 0 {
			segments = append(segments, current)
			current = nil
		}
	}
	for _, point := range points {
		if !valid(point) {
			flush()
			continue
		}
		if len(current) > 0 && point.Segment != currentSegment {
			flush()
		}
		currentSegment = point.Segment
		current = append(current, point)
	}
	flush()
	return segments
}
