// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"github.com/egoist/mygo/ui"
	"orangecount/apps/goui/internal/charts"
)

func cashflowLineSample(tr func(string) string) charts.Series {
	return charts.Series{Label: "Net cash flow", Color: ui.RGB(42, 119, 214), Points: []charts.Point{
		{Label: "Jan", Value: 700, Exact: "700.00 USD", Available: true},
		{Label: "Feb", Value: -300, Exact: "-300.00 USD", Available: true},
		{Label: "Mar", Value: 0, Exact: "0.00 USD", Available: true},
		{Label: "Apr", Exact: tr("不可用"), Available: false},
		{Label: "May", Value: 450, Exact: "450.00 USD", Available: true},
	}}
}

func cashflowBarSample(tr func(string) string) []charts.Series {
	return []charts.Series{
		{Label: "Income", Color: ui.RGB(40, 145, 100), Points: []charts.Point{
			{Label: "Jan", Value: 1000, Exact: "1000.00 USD", Available: true},
			{Label: "Feb", Value: 0, Exact: "0.00 USD", Available: true},
			{Label: "Mar", Value: 650, Exact: "650.00 USD", Available: true},
			{Label: "Apr", Exact: tr("不可用"), Available: false},
		}},
		{Label: "Expenses", Color: ui.RGB(210, 80, 76), Points: []charts.Point{
			{Label: "Jan", Value: -300, Exact: "-300.00 USD", Available: true},
			{Label: "Feb", Value: -250, Exact: "-250.00 USD", Available: true},
			{Label: "Mar", Value: 0, Exact: "0.00 USD", Available: true},
			{Label: "Apr", Exact: tr("不可用"), Available: false},
		}},
	}
}

func LineChart(c *ui.Context, series charts.Series, tr charts.Translate) *ui.Element {
	return charts.LineLocalized(c, series, tr)
}

func GroupedBars(c *ui.Context, series []charts.Series, tr charts.Translate) *ui.Element {
	return charts.GroupedBarsLocalized(c, series, tr)
}

func ExactTable(c *ui.Context, state *ui.ListState, series []charts.Series, tr charts.Translate) *ui.Element {
	return charts.ExactTableLocalized(c, state, series, tr)
}
