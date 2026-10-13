// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package charts

import (
	"math"

	"github.com/egoist/mygo/ui"
)

type pointSelection struct {
	index                  int
	pointerX, pointerY     float32
	pointerSeen, pointerOn bool
	focused                bool
}

func selectedPoint(element *ui.Element, count int, grouped bool) int {
	state := ui.Local(element, "selected-point", func() pointSelection {
		return pointSelection{index: -1}
	})
	if count == 0 {
		state.index = -1
		return -1
	}
	x, y, over := element.PointerPosition()
	focused := element.Focused()
	if over && (!state.pointerSeen || !state.pointerOn || x != state.pointerX || y != state.pointerY) {
		state.index = hoveredIndex(element, count, x, grouped)
	}
	if state.focused && !focused {
		state.index = -1
		if over {
			state.index = hoveredIndex(element, count, x, grouped)
		}
	}
	state.focused = focused
	state.pointerSeen, state.pointerOn = true, over
	state.pointerX, state.pointerY = x, y
	if !over && !focused {
		state.index = -1
	}

	left := element.Shortcut(0, ui.KeyLeft)
	right := element.Shortcut(0, ui.KeyRight)
	first := element.Shortcut(0, ui.KeyHome)
	last := element.Shortcut(0, ui.KeyEnd)
	switch {
	case first:
		state.index = 0
	case last:
		state.index = count - 1
	case left:
		if state.index < 0 {
			state.index = count - 1
		} else if state.index > 0 {
			state.index--
		}
	case right:
		if state.index < 0 {
			state.index = 0
		} else if state.index < count-1 {
			state.index++
		}
	}
	if state.index < 0 || state.index >= count {
		return -1
	}
	return state.index
}
func hoveredIndex(element *ui.Element, count int, x float32, grouped bool) int {
	// PointerPosition returns coordinates local to this element, so hit testing
	// must use a plot rectangle with the same local origin. Bounds is relative
	// to the window and would shift the selected point for charts laid out away
	// from the window's top-left corner.
	bounds := element.Bounds()
	plot := inset(ui.Rect{W: bounds.W, H: bounds.H}, 12, 12, 12, 14)
	if grouped {
		return groupedIndexAt(plot, count, x)
	}
	return lineIndexAt(plot, count, x)
}

func lineIndexAt(plot ui.Rect, count int, x float32) int {
	if count == 0 {
		return -1
	}
	if count == 1 {
		return 0
	}
	position := min(max(x-plot.X, 0), plot.W)
	return min(max(int(math.Round(float64(position/plot.W*float32(count-1)))), 0), count-1)
}

func groupedIndexAt(plot ui.Rect, count int, x float32) int {
	if count == 0 {
		return -1
	}
	position := min(max(x-plot.X, 0), plot.W)
	return min(max(int(position/plot.W*float32(count)), 0), count-1)
}
