// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

// Package queryeditor owns source offsets shared by the query field and the
// desktop diagnostics. Parser locations are UTF-8 byte offsets; MyGo's editor
// positions are rune offsets.
package queryeditor

import (
	"errors"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

var ErrInvalidRange = errors.New("query source range is not on UTF-8 rune boundaries")

// ByteRangeToRune converts a half-open UTF-8 byte range to editor rune offsets.
// It rejects invalid UTF-8, out-of-range offsets and offsets inside a code
// point, leaving the caller free to show a diagnostic without moving a caret.
func ByteRangeToRune(source string, start, end int) (runeStart, runeEnd int, err error) {
	if start < 0 || end < start || end > len(source) || !utf8.ValidString(source) {
		return 0, 0, ErrInvalidRange
	}
	if !runeBoundary(source, start) || !runeBoundary(source, end) {
		return 0, 0, ErrInvalidRange
	}
	return utf8.RuneCountInString(source[:start]), utf8.RuneCountInString(source[:end]), nil
}

// SelectByteRange adapts the parser's UTF-8 byte range to MyGo's public rune
// selection API. Invalid ranges leave the editor selection and caret alone.
func SelectByteRange(editor *ui.Element, source string, startByte, endByte int) bool {
	start, end, err := ByteRangeToRune(source, startByte, endByte)
	if err != nil || editor == nil {
		return false
	}
	return editor.SetTextSelection(start, end)
}

func runeBoundary(source string, offset int) bool {
	return offset == len(source) || offset >= 0 && offset < len(source) && utf8.RuneStart(source[offset])
}
