// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package diagnostic

import (
	"bytes"
	"strings"
	"testing"

	"orangecount/internal/source"
)

func TestBagSortAndLocale(t *testing.T) {
	var bag Bag
	late := source.Span{Start: 10, StartLine: 2, StartColumn: 1}
	first := source.Span{Start: 1, StartLine: 1, StartColumn: 1}
	bag.Add(New("E-PARSE-DATE", Error, late).WithPath("z.bean"))
	bag.Add(New("E-PARSE-DATE", Error, first).WithPath("a.bean"))
	got := bag.All()
	if got[0].Path != "a.bean" || !bag.HasErrors() {
		t.Fatalf("sorted=%+v errors=%v", got, bag.HasErrors())
	}
	var out bytes.Buffer
	if err := RenderHuman(&out, got, "zh-CN"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "日期无效") {
		t.Fatalf("locale output=%q", out.String())
	}
}

func TestCustomMessageIsRedactedAndStable(t *testing.T) {
	d := New("E-CUSTOM", Error, source.Span{}, "raw\nledger")
	if d.Message != "raw ledger" || d.Redacted == false {
		t.Fatalf("diagnostic=%+v", d)
	}
	if got := Localize(d, "zh-CN"); got.Message != "raw ledger" {
		t.Fatalf("localized=%+v", got)
	}
}

func TestBagHandlesNilAndOrdersSameLocationBySeverityCodeAndSequence(t *testing.T) {
	var nilBag *Bag
	nilBag.Add(New("E-PARSE-DATE", Error, source.Span{}))
	nilBag.Extend(New("E-PARSE-DATE", Error, source.Span{}))
	if nilBag.Len() != 0 || !nilBag.Empty() || nilBag.HasErrors() || nilBag.All() != nil {
		t.Fatal("nil bag did not behave as an empty collection")
	}

	span := source.Span{Start: 3}
	var bag Bag
	bag.Add(New("Z", Info, span).WithPath("same.bean"))
	bag.Add(New("A", Warning, span).WithPath("same.bean"))
	bag.Extend(New("B", Error, span).WithPath("same.bean"), New("A", Error, span).WithPath("same.bean"))
	got := bag.All()
	if bag.Len() != 4 || bag.Empty() || !bag.HasErrors() {
		t.Fatalf("bag state len=%d empty=%v errors=%v", bag.Len(), bag.Empty(), bag.HasErrors())
	}
	order := []string{got[0].Code, got[1].Code, got[2].Code, got[3].Code}
	if want := []string{"A", "B", "A", "Z"}; strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("sorted order=%v want=%v", order, want)
	}
}

func TestLocalizationAndRenderersHaveSafeFallbacks(t *testing.T) {
	if got := LocalizeCode("E-PARSE-DATE", ""); got != "invalid date" {
		t.Fatalf("empty locale=%q", got)
	}
	if got := LocalizeCode("E-PARSE-DATE", "fr"); got != "invalid date" {
		t.Fatalf("unknown locale=%q", got)
	}
	if got := LocalizeCode("E-UNKNOWN", "zh-CN"); got != "E-UNKNOWN" {
		t.Fatalf("unknown code=%q", got)
	}
	catalogue := New("E-PARSE-DATE", Error, source.Span{StartLine: 2, StartColumn: 3})
	if catalogue.MessageKeyOrCode() != "E-PARSE-DATE" {
		t.Fatalf("catalogue message key=%q", catalogue.MessageKeyOrCode())
	}
	custom := New("E-CUSTOM", Warning, source.Span{}, "literal")
	if custom.MessageKeyOrCode() != "E-CUSTOM" || Localize(custom, "zh-CN").Message != "literal" {
		t.Fatalf("custom diagnostic=%+v", custom)
	}
	var human bytes.Buffer
	if err := RenderHuman(&human, []Diagnostic{catalogue}, "en"); err != nil {
		t.Fatal(err)
	}
	if got := human.String(); got != "<source>:2:3: error E-PARSE-DATE: invalid date\n" {
		t.Fatalf("human=%q", got)
	}
	var jsonOut bytes.Buffer
	withRelated := catalogue
	withRelated.Related = []Related{{Message: "context"}}
	if err := RenderJSON(&jsonOut, []Diagnostic{withRelated}, "zh-CN"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(jsonOut.String(), "日期无效") || !strings.Contains(jsonOut.String(), "context") {
		t.Fatalf("json=%q", jsonOut.String())
	}
}

func TestRedactMessageNormalizesWhitespaceAndSensitiveFragments(t *testing.T) {
	if got := RedactMessage("  alpha\r\nbeta\tsecret  ", "secret", ""); got != "alpha  beta [redacted]" {
		t.Fatalf("redacted=%q", got)
	}
}
