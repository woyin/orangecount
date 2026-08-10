// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"testing"

	"orangecount/internal/source"
)

func TestDirectiveInterfaceKeepsKindsSpansAndRawTextUniform(t *testing.T) {
	base := DirectiveBase{At: source.Span{File: 1, Start: 2, End: 3}, Raw: "raw"}
	cases := []struct {
		name  string
		value Directive
		kind  DirectiveKind
	}{
		{"option", Option{DirectiveBase: base}, KindOption},
		{"plugin", Plugin{DirectiveBase: base}, KindPlugin},
		{"include", Include{DirectiveBase: base}, KindInclude},
		{"push tag", TagDirective{DirectiveBase: base, Tag: "tag"}, KindPushTag},
		{"pop tag", TagDirective{DirectiveBase: DirectiveBase{Raw: "poptag #tag"}}, KindPopTag},
		{"open", Open{DirectiveBase: base}, KindOpen},
		{"close", Close{DirectiveBase: base}, KindClose},
		{"commodity", Commodity{DirectiveBase: base}, KindCommodity},
		{"balance", Balance{DirectiveBase: base}, KindBalance},
		{"pad", Pad{DirectiveBase: base}, KindPad},
		{"event", Event{DirectiveBase: base}, KindEvent},
		{"query", Query{DirectiveBase: base}, KindQuery},
		{"price", Price{DirectiveBase: base}, KindPrice},
		{"document", Document{DirectiveBase: base}, KindDocument},
		{"note", Note{DirectiveBase: base}, KindNote},
		{"custom", Custom{DirectiveBase: base}, KindCustom},
		{"transaction", Transaction{DirectiveBase: base}, KindTxn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.value.Kind(); got != tc.kind {
				t.Fatalf("Kind()=%q, want %q", got, tc.kind)
			}
			if tc.name != "pop tag" && (tc.value.Span() != base.At || tc.value.RawText() != "raw") {
				t.Fatalf("span=%+v raw=%q", tc.value.Span(), tc.value.RawText())
			}
		})
	}
	posting := Posting{At: base.At, Raw: "posting"}
	if posting.Span() != base.At || posting.RawText() != "posting" {
		t.Fatalf("posting interface=%+v", posting)
	}
}

func TestDateValidationHandlesLeapYearsAndMonthLengths(t *testing.T) {
	cases := []struct {
		date  Date
		valid bool
	}{
		{Date{Year: 2024, Month: 2, Day: 29, Raw: "2024-02-29"}, true},
		{Date{Year: 1900, Month: 2, Day: 29}, false},
		{Date{Year: 2000, Month: 2, Day: 29}, true},
		{Date{Year: 2025, Month: 4, Day: 31}, false},
		{Date{Year: 2025, Month: 12, Day: 31}, true},
		{Date{Year: 0, Month: 1, Day: 1}, false},
		{Date{Year: 2025, Month: 13, Day: 1}, false},
	}
	for _, tc := range cases {
		if got := tc.date.Valid(); got != tc.valid {
			t.Errorf("date=%+v valid=%v, want %v", tc.date, got, tc.valid)
		}
	}
	if got := (Date{Raw: "2025-12-31"}).String(); got != "2025-12-31" {
		t.Fatalf("Date.String()=%q", got)
	}
}
