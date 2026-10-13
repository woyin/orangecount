// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

// Package charts draws reusable finance views. Exact ledger text stays beside
// the approximate coordinates used only to plot geometry.
package charts

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// Translate maps application-owned chart labels into the active UI locale.
// Ledger values and period labels are data and must not be passed through it.
type Translate func(string) string

func Line(c *ui.Context, series Series) *ui.Element {
	return LineLocalized(c, series, nil)
}

func LineLocalized(c *ui.Context, series Series, tr Translate) *ui.Element {
	color := series.Color
	if color == ui.Transparent {
		color = ui.RGB(42, 119, 214)
	}
	element := ui.Box(c).FillWidth().Height(220).Label(series.Label + chartText(tr, "折线图")).Role(ui.RoleImage).Focusable()
	selected := selectedPoint(element, len(series.Points), false)
	description := series.Label + chartText(tr, "；精确值见下方表格；聚焦后可用左右箭头切换，Home/End跳至首尾")
	tip := chartText(tr, "悬停查看精确期间值")
	if selected >= 0 {
		point := series.Points[selected]
		description += chartText(tr, "；当前：") + point.Label + " · " + tooltipTextLocalized(point, tr)
		tip = point.Label + " · " + tooltipTextLocalized(point, tr)
	}
	element.Description(description)
	element.Tooltip(tip)
	return element.Draw(func(p *ui.Painter, r ui.Rect) {
		layout := layoutLine(series.Points, r)
		renderLine(p, series, layout, selected, color, c.Theme().Border)
	})
}
func GroupedBars(c *ui.Context, series []Series) *ui.Element {
	return GroupedBarsLocalized(c, series, nil)
}

func GroupedBarsLocalized(c *ui.Context, series []Series, tr Translate) *ui.Element {
	container := ui.Column(c).FillWidth().Gap(4)
	container.Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			for _, item := range series {
				color := item.Color
				if color == ui.Transparent {
					color = ui.RGB(42, 119, 214)
				}
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(9, 9).Background(color).Radius(2).Shrink(0)
					ui.Text(c, item.Label).FontSize(11).TextColor(c.Theme().TextMuted)
				})
			}
		})
		groupedBarsPlotLocalized(c, series, tr)
	})
	return container
}
func groupedBarsPlot(c *ui.Context, series []Series) *ui.Element {
	return groupedBarsPlotLocalized(c, series, nil)
}

func groupedBarsPlotLocalized(c *ui.Context, series []Series, tr Translate) *ui.Element {
	labels := pointLabels(series)
	element := ui.Box(c).FillWidth().Height(220).Label(chartText(tr, "分组柱状图")).Role(ui.RoleImage).Focusable()
	selected := selectedPoint(element, len(labels), true)
	description := chartText(tr, "各期间精确金额见下方表格；聚焦后可用左右箭头切换，Home/End跳至首尾")
	tip := chartText(tr, "悬停查看精确期间金额")
	if selected >= 0 {
		label := labels[selected]
		var lines []string
		for _, item := range series {
			point, ok := pointForLabel(item.Points, label)
			if !ok {
				lines = append(lines, item.Label+chartSeparator(tr)+chartText(tr, "缺失"))
				continue
			}
			lines = append(lines, item.Label+chartSeparator(tr)+tooltipTextLocalized(point, tr))
		}
		description += chartText(tr, "；当前：") + label + " · " + strings.Join(lines, chartText(tr, "；"))
		tip = label + "\n" + strings.Join(lines, "\n")
	}
	element.Description(description)
	element.Tooltip(tip)
	return element.Draw(func(p *ui.Painter, r ui.Rect) {
		layout := layoutGroupedBars(series, r)
		renderGroupedBars(p, series, layout, selected, c.Theme().Border, c.Theme().Accent)
	})
}
