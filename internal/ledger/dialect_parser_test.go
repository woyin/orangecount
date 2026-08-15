// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"strings"
	"testing"
)

// parseOneDialect parses text and returns the last Dialect directive plus
// the diagnostic codes produced (last, so anchor chains are assertable).
func parseOneDialect(t *testing.T, text string) (Dialect, []string) {
	t.Helper()
	file, bag := ParseText("main.bean", []byte(text))
	var codes []string
	for _, d := range bag.All() {
		codes = append(codes, d.Code)
	}
	var found Dialect
	saw := false
	for _, directive := range file.Directives {
		if d, ok := directive.(Dialect); ok {
			found, saw = d, true
		}
	}
	if !saw {
		return Dialect{}, codes
	}
	return found, codes
}

func TestParseDialectLineFullGrammar(t *testing.T) {
	d, codes := parseOneDialect(t,
		"2026-08-12 ! 1,234.56 CNY @Assets:WeChat -> @Food \"美团\" : 工作午餐 #a #b ^l\n")
	if len(codes) != 0 {
		t.Fatalf("codes=%v", codes)
	}
	if !d.HasDate || d.Date.Raw != "2026-08-12" || d.Flag != "!" {
		t.Fatalf("head=%+v", d)
	}
	if d.Amount.Raw != "1234.56" || d.Currency != "CNY" {
		t.Fatalf("amount=%q currency=%q", d.Amount.Raw, d.Currency)
	}
	if d.SourceRef != "Assets:WeChat" || d.DestRef != "Food" {
		t.Fatalf("endpoints=%q %q", d.SourceRef, d.DestRef)
	}
	if d.Payee != "美团" || d.Narration != "工作午餐" || !d.HasNarration {
		t.Fatalf("payee=%q narration=%q has=%v", d.Payee, d.Narration, d.HasNarration)
	}
	if len(d.Tags) != 2 || len(d.Links) != 1 {
		t.Fatalf("tags=%v links=%v", d.Tags, d.Links)
	}
}

func TestParseDialectLineOmissionsAndAnchors(t *testing.T) {
	full, codes := parseOneDialect(t, "2026-08-12 28 USD @A -> @B\n")
	if len(codes) != 0 || full.Flag != "*" || full.HasNarration || full.Currency != "USD" {
		t.Fatalf("full=%+v codes=%v", full, codes)
	}
	anchored, codes := parseOneDialect(t, "2026-08-12 1 USD @A -> @B\n2 USD @A -> @B\n")
	if len(codes) != 0 || !anchored.Anchored || anchored.Date.Raw != "2026-08-12" {
		t.Fatalf("anchored=%+v codes=%v", anchored, codes)
	}
	// A dated standard transaction between dialect lines does not become the
	// anchor: the undated line still inherits the earlier dialect date.
	anchored, codes = parseOneDialect(t,
		"2026-08-12 1 USD @A -> @B\n2026-08-13 * \"p\" \"n\"\n  A -1 USD\n  B 1 USD\n2 USD @A -> @B\n")
	if len(codes) != 0 || anchored.Date.Raw != "2026-08-12" {
		t.Fatalf("anchor=%q codes=%v", anchored.Date.Raw, codes)
	}
}

func TestParseDialectLineDatelessWithFlag(t *testing.T) {
	d, codes := parseOneDialect(t, "2026-08-12 1 USD @A -> @B\n! 2 USD @A -> @B\n")
	if len(codes) != 0 {
		t.Fatalf("codes=%v", codes)
	}
	if d.Flag != "!" || !d.Anchored {
		t.Fatalf("dateless flag=%q anchored=%v", d.Flag, d.Anchored)
	}
}

func TestParseDialectLineSyntaxFailures(t *testing.T) {
	cases := []struct{ code, line string }{
		{"E-DIALECT-AMOUNT", "2026-08-12 -5 USD @A -> @B"},            // negative
		{"E-DIALECT-AMOUNT", "2026-08-12 1.2.3 USD @A -> @B"},         // malformed
		{"E-PARSE-EXPECTED", "2026-08-12"},                            // bare date stays a v3 error
		{"E-DIALECT-DATE", "5 USD @A -> @B"},                          // no anchor in file
		{"E-DIALECT-SYNTAX", "2026-08-12 5 USD A -> @B"},              // missing @ on source
		{"E-DIALECT-ARROW", "2026-08-12 5 USD @A @B"},                 // missing arrow
		{"E-DIALECT-SYNTAX", "2026-08-12 5 USD @A -> @B \"x\" junk"},  // bare word without ':'
		{"E-DIALECT-SYNTAX", "2026-08-12 5 USD @A -> @B \"x\" \"y\""}, // second payee string
		{"E-DIALECT-SYNTAX", "2026-08-12 5 USD @@"},                   // total-price sigil
	}
	for _, tc := range cases {
		_, codes := parseOneDialect(t, tc.line+"\n")
		if !containsCode(codes, tc.code) {
			t.Errorf("line=%q want %s got %v", tc.line, tc.code, codes)
		}
	}
}

func TestParseDialectNarrationJoinsWords(t *testing.T) {
	d, codes := parseOneDialect(t, "2026-08-12 5 USD @A -> @B : a b c\n")
	if len(codes) != 0 || d.Narration != "a b c" {
		t.Fatalf("narration=%q codes=%v", d.Narration, codes)
	}
	// Tags before the narration colon still parse.
	d, codes = parseOneDialect(t, "2026-08-12 5 USD @A -> @B \"p\" #t : n\n")
	if len(codes) != 0 || d.Payee != "p" || len(d.Tags) != 1 || d.Narration != "n" {
		t.Fatalf("d=%+v codes=%v", d, codes)
	}
}

func TestIsDialectStartNaturalGap(t *testing.T) {
	// Standard shapes stay out of the dialect path even when they carry
	// numbers or flags in unusual spots.
	standard := []string{
		"2026-08-12 open Assets:Cash USD",
		"2026-08-12 * \"p\" \"n\"",
		"2026-08-12 balance Assets:Cash 5 USD",
		"option \"title\" \"t\"",
		"include \"x.bean\"",
	}
	for _, line := range standard {
		file, bag := ParseText("m.bean", []byte(line+"\n"))
		for _, directive := range file.Directives {
			if _, ok := directive.(Dialect); ok {
				t.Errorf("standard line %q claimed as dialect", line)
			}
		}
		if bag.HasErrors() {
			t.Errorf("standard line %q stopped parsing: %v", line, bag.All())
		}
	}
}

func containsCode(list []string, want string) bool {
	for _, value := range list {
		if value == want || strings.HasPrefix(value, want) {
			return true
		}
	}
	return false
}
