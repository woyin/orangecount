// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package queryeditor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestByteRangeToRuneHandlesUnicodeAndLineEndings(t *testing.T) {
	source := "select 现金💰e\u0301\r\nfrom Expenses"
	cases := []struct {
		name       string
		needle     string
		byteStart  int
		byteLength int
		wantStart  int
		wantEnd    int
	}{
		{name: "Chinese", needle: "现金", byteStart: strings.Index(source, "现金"), byteLength: len("现金"), wantStart: 7, wantEnd: 9},
		{name: "emoji", needle: "💰", byteStart: strings.Index(source, "💰"), byteLength: len("💰"), wantStart: 9, wantEnd: 10},
		{name: "combining sequence", needle: "e\u0301", byteStart: strings.Index(source, "e\u0301"), byteLength: len("e\u0301"), wantStart: 10, wantEnd: 12},
		{name: "CRLF", needle: "\r\n", byteStart: strings.Index(source, "\r\n"), byteLength: len("\r\n"), wantStart: 12, wantEnd: 14},
		{name: "tail", byteStart: len(source), byteLength: 0, wantStart: 27, wantEnd: 27},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, end, err := ByteRangeToRune(source, tc.byteStart, tc.byteStart+tc.byteLength)
			if err != nil {
				t.Fatal(err)
			}
			if start != tc.wantStart || end != tc.wantEnd {
				t.Fatalf("rune range = [%d,%d), want [%d,%d) for %q", start, end, tc.wantStart, tc.wantEnd, tc.needle)
			}
		})
	}
}

func TestSelectByteRangeSelectsTheCoreDiagnosticToken(t *testing.T) {
	source := "SELECT '💰', bad_column LIMIT 0"
	startByte := strings.Index(source, "bad_column")
	apply, selected := false, false
	tester := ui.NewTester(func(c *ui.Context) {
		e := ui.TextArea(c, &source).Fill().Focus()
		if apply {
			selected = SelectByteRange(e, source, startByte, startByte+len("bad_column"))
		}
	}, 460, 180)
	apply = true
	tester.Frame()
	if !selected {
		t.Fatal("valid core diagnostic range was not selected")
	}
	tester.Type("fixed")
	if want := "SELECT '💰', fixed LIMIT 0"; source != want {
		t.Fatalf("selected core token replacement = %q, want %q", source, want)
	}
}

func TestSelectByteRangeLeavesEditorUnchangedForInvalidBytes(t *testing.T) {
	source := "现金💰"
	apply, selected := false, true
	tester := ui.NewTester(func(c *ui.Context) {
		e := ui.TextArea(c, &source).Fill().Focus()
		if apply {
			selected = SelectByteRange(e, source, 1, 4)
		}
	}, 240, 120)
	apply = true
	tester.Frame()
	if selected {
		t.Fatal("range inside a multibyte rune was accepted")
	}
	tester.Type("!")
	if want := "现金💰!"; source != want {
		t.Fatalf("invalid diagnostic range moved the caret: %q, want %q", source, want)
	}
}

func TestByteRangeToRuneRejectsInvalidBoundaries(t *testing.T) {
	source := "账户💰"
	for _, bounds := range [][2]int{{-1, 0}, {4, 5}, {0, len(source) + 1}, {5, 4}} {
		if _, _, err := ByteRangeToRune(source, bounds[0], bounds[1]); err == nil {
			t.Errorf("range [%d,%d) was accepted", bounds[0], bounds[1])
		}
	}
	if _, _, err := ByteRangeToRune("\xff", 0, 1); err == nil {
		t.Error("invalid UTF-8 source was accepted")
	}
}
