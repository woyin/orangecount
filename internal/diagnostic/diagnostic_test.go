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
