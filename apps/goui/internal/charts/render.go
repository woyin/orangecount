// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package charts

import "github.com/egoist/mygo/ui"

func renderLine(p *ui.Painter, series Series, layout lineLayout, selected int, color, guideColor ui.Color) {
	drawGuides(p, layout.plot, layout.values, guideColor)
	for _, indices := range layout.segments {
		var path ui.Path
		lastX, lastY := float32(0), float32(0)
		for sample, index := range indices {
			point := series.Points[index]
			x := xAt(layout.plot, index, len(series.Points))
			y := yAt(layout.plot, layout.values, point.Value)
			if sample == 0 {
				path.MoveTo(x, y)
			} else {
				path.LineTo(x, y)
			}
			lastX, lastY = x, y
		}
		if len(indices) == 1 {
			var dot ui.Path
			dot.Circle(lastX, lastY, 3)
			p.FillPath(&dot, color)
		} else if len(indices) > 1 {
			p.StrokePath(&path, 2.5, color)
		}
	}
	if selected >= 0 && selected < len(series.Points) {
		point := series.Points[selected]
		selectedX := xAt(layout.plot, selected, len(series.Points))
		p.Line(selectedX, layout.plot.Y, selectedX, layout.plot.Y+layout.plot.H, 1, color)
		if valid(point) {
			var marker ui.Path
			marker.Circle(selectedX, yAt(layout.plot, layout.values, point.Value), 4)
			p.FillPath(&marker, color)
		}
	}
}

func renderGroupedBars(p *ui.Painter, series []Series, layout groupedLayout, selected int, guideColor, selectionColor ui.Color) {
	drawGuides(p, layout.plot, layout.values, guideColor)
	if len(layout.labels) == 0 || len(series) == 0 {
		return
	}
	zeroY := yAt(layout.plot, layout.values, 0)
	sampled := make([]bool, len(layout.labels))
	for _, index := range layout.samples {
		sampled[index] = true
	}
	for seriesIndex, item := range series {
		color := item.Color
		if color == ui.Transparent {
			color = ui.RGB(42, 119, 214)
		}
		for _, point := range item.Points {
			pointIndex, ok := layout.labelIndex[point.Label]
			if !ok || !sampled[pointIndex] || !valid(point) {
				continue
			}
			center := layout.plot.X + (float32(pointIndex)+0.5)*layout.groupWidth
			x := center - (float32(len(series))*layout.barWidth)/2 + (float32(seriesIndex)+0.5)*layout.barWidth
			y := yAt(layout.plot, layout.values, point.Value)
			if point.Value == 0 {
				var dot ui.Path
				dot.Circle(x, zeroY, max(2, layout.barWidth/3))
				p.FillPath(&dot, color)
				continue
			}
			top := min(y, zeroY)
			height := max(abs32(y-zeroY), 1)
			p.Fill(ui.Rect{X: x - layout.barWidth/2, Y: top, W: layout.barWidth, H: height}, color, 2)
		}
	}
	if selected >= 0 && selected < len(layout.labels) {
		center := layout.plot.X + (float32(selected)+0.5)*layout.groupWidth
		p.Line(center, layout.plot.Y, center, layout.plot.Y+layout.plot.H, 1, selectionColor)
	}
}

func drawGuides(p *ui.Painter, r ui.Rect, bounds valueRange, color ui.Color) {
	zeroY := yAt(r, bounds, 0)
	p.Line(r.X, zeroY, r.X+r.W, zeroY, 1, color)
	p.Line(r.X, r.Y, r.X+r.W, r.Y, 1, color)
	p.Line(r.X, r.Y+r.H, r.X+r.W, r.Y+r.H, 1, color)
}
func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
