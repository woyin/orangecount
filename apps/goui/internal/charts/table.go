// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package charts

import (
	"sort"

	"github.com/egoist/mygo/ui"
)

// ExactTable keeps exact ledger values and missing states beside plotted
// approximations; it never reads financial values from plot geometry.
func ExactTable(c *ui.Context, state *ui.ListState, series []Series) *ui.Element {
	return ExactTableLocalized(c, state, series, nil)
}

func ExactTableLocalized(c *ui.Context, state *ui.ListState, series []Series, tr Translate) *ui.Element {
	columns := []ui.TableColumn{{Title: chartText(tr, "期间"), ID: "period", Width: 130, Fixed: true}}
	for i, item := range series {
		columns = append(columns, ui.TableColumn{Title: item.Label, ID: "series-" + itoa(i), Width: 160})
	}
	labels := pointLabels(series)
	pointsByLabel := make([]map[string]Point, len(series))
	for seriesIndex, item := range series {
		points := make(map[string]Point, len(item.Points))
		for _, point := range item.Points {
			points[point.Label] = point
		}
		pointsByLabel[seriesIndex] = points
	}
	return ui.Table(c, state, columns, len(labels), func(row, col int) {
		if col == 0 {
			ui.Text(c, labels[row])
			return
		}
		point, ok := pointsByLabel[col-1][labels[row]]
		if !ok {
			ui.Text(c, "—")
			return
		}
		if point.Available {
			ui.Text(c, point.Exact)
			return
		}
		label := point.Exact
		if label == "" {
			label = chartText(tr, "缺失")
		}
		ui.Text(c, label).TextColor(c.Theme().TextMuted)
	}).FillWidth().Height(210)
}

type exactRow struct {
	label  string
	points map[int]Point
}

func exactRows(series []Series) []exactRow {
	rowsByLabel := make(map[string]map[int]Point)
	for seriesIndex, item := range series {
		for _, point := range item.Points {
			if rowsByLabel[point.Label] == nil {
				rowsByLabel[point.Label] = make(map[int]Point)
			}
			rowsByLabel[point.Label][seriesIndex] = point
		}
	}
	labels := make([]string, 0, len(rowsByLabel))
	for label := range rowsByLabel {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	rows := make([]exactRow, 0, len(labels))
	for _, label := range labels {
		rows = append(rows, exactRow{label: label, points: rowsByLabel[label]})
	}
	return rows
}
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	n := len(buf)
	for i > 0 {
		n--
		buf[n] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[n:])
}
