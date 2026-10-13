// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package widgets

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSourceTextKeepsStableLocationFormat(t *testing.T) {
	tests := []struct {
		name string
		in   SourceLocation
		want string
	}{
		{name: "empty", want: "—"},
		{name: "path only", in: SourceLocation{Path: "ledger.bean"}, want: "ledger.bean"},
		{name: "line", in: SourceLocation{Path: "ledger.bean", Line: 12}, want: "ledger.bean:12"},
		{name: "column", in: SourceLocation{Path: "ledger.bean", Line: 12, Column: 4}, want: "ledger.bean:12:4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SourceText(tt.in); got != tt.want {
				t.Fatalf("SourceText(%+v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSourceRowActionsAndDisabledOpen(t *testing.T) {
	var actions []string
	location := SourceLocation{Path: "/tmp/book.bean", Line: 12, Column: 4}
	tester := ui.NewTester(func(c *ui.Context) {
		SourceRow(c, location, SourceActions{
			CopyLabel: "复制位置", RevealLabel: "显示来源", OpenLabel: "打开源文件",
			Copy:   func() { actions = append(actions, "copy") },
			Reveal: func() { actions = append(actions, "reveal") },
			Open:   func() { actions = append(actions, "open") },
		})
	}, 900, 80)
	if !tester.HasText("/tmp/book.bean:12:4") {
		t.Fatalf("source location missing from row: %v", tester.Texts())
	}
	for _, label := range []string{
		"复制位置 · /tmp/book.bean:12:4",
		"显示来源 · /tmp/book.bean:12:4",
		"打开源文件 · /tmp/book.bean:12:4",
	} {
		if _, ok := tester.Find(label); !ok {
			t.Errorf("source action lacks a location-specific accessibility label %q", label)
		}
	}
	for _, label := range []string{"复制位置", "显示来源", "打开源文件"} {
		if err := tester.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if want := []string{"copy", "reveal", "open"}; !reflect.DeepEqual(actions, want) {
		t.Fatalf("source actions = %v, want %v", actions, want)
	}

	blocked := 0
	disabled := ui.NewTester(func(c *ui.Context) {
		SourceRow(c, location, SourceActions{
			OpenLabel: "打开源文件", OpenDisabled: true,
			Open: func() { blocked++ },
		})
	}, 900, 80)
	if err := disabled.Click("打开源文件"); err != nil {
		t.Fatal(err)
	}
	if blocked != 0 {
		t.Fatalf("disabled open action ran %d times", blocked)
	}
}

func TestSourceRowEmptyLocationHasNoActions(t *testing.T) {
	called := false
	tester := ui.NewTester(func(c *ui.Context) {
		SourceRow(c, SourceLocation{}, SourceActions{
			CopyLabel: "复制位置",
			Copy:      func() { called = true },
		})
	}, 400, 60)
	if !tester.HasText("—") {
		t.Fatalf("empty source placeholder missing: %v", tester.Texts())
	}
	if _, ok := tester.Find("复制位置"); ok {
		t.Fatal("empty source rendered an actionable copy button")
	}
	if called {
		t.Fatal("empty source unexpectedly invoked a callback")
	}
}
