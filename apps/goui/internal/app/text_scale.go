// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import "fmt"

var supportedTextScales = [...]float32{1, 1.25, 1.5}

func normalizedTextScale(scale float32) float32 {
	for _, supported := range supportedTextScales {
		if scale == supported {
			return supported
		}
	}
	return 1
}

func (m *Model) increaseTextScale() { m.updateTextScale(1) }
func (m *Model) decreaseTextScale() { m.updateTextScale(-1) }
func (m *Model) resetTextScale()    { m.updateTextScale(0) }

func (m *Model) updateTextScale(direction int) {
	apply := func() {
		if m.closed.Load() {
			return
		}
		current := normalizedTextScale(m.textScale)
		index := 0
		for i, scale := range supportedTextScales {
			if scale == current {
				index = i
				break
			}
		}
		switch {
		case direction > 0:
			index = min(index+1, len(supportedTextScales)-1)
		case direction < 0:
			index = max(index-1, 0)
		case direction == 0:
			index = 0
		}
		next := supportedTextScales[index]
		if next == current {
			return
		}
		m.textScale = next
		m.preferenceError = ""
		m.status = fmt.Sprintf(m.tr("文字大小：%d%%"), int(next*100))
		m.persistReadingPreferences()
	}
	if m.window != nil {
		m.window.Update(apply)
		return
	}
	apply()
}
