// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package source

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPositionAndSpanUnicode(t *testing.T) {
	f := NewSourceFile(1, "x.bean", []byte("é\n第二行\n"))
	got := f.Position(3)
	if got.Line != 2 || got.Column != 1 || got.Offset != 3 {
		t.Fatalf("position = %+v", got)
	}
	sp := f.Span(3, len(f.Data))
	if sp.StartLine != 2 || sp.StartColumn != 1 || sp.EndLine != 3 || sp.EndColumn != 1 {
		t.Fatalf("span = %+v", sp)
	}
}

func TestLoadGraphIncludesAndCycle(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	child := filepath.Join(dir, "child.bean")
	if err := os.WriteFile(entry, []byte("include \"child.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, []byte("include \"main.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGraph(entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Order) != 2 || len(g.Diagnostics) != 1 || g.Diagnostics[0].Code != "E-INCLUDE-CYCLE" {
		t.Fatalf("graph order=%v diagnostics=%+v", g.Order, g.Diagnostics)
	}
	if len(g.Edges[1]) != 1 || len(g.Edges[2]) != 1 {
		t.Fatalf("edges=%+v", g.Edges)
	}
}

func TestLoadGraphResolvesAbsoluteIncludesVerbatim(t *testing.T) {
	dir := t.TempDir()
	inner := filepath.Join(dir, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(inner, "child.bean")
	if err := os.WriteFile(child, []byte("2026-01-01 open Assets:Cash CNY\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The entry lives outside inner/ and includes the child by absolute path,
	// as beancount does: absolute includes are used verbatim rather than
	// joined onto the including file's directory.
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte(fmt.Sprintf("include %q\n", child)), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGraph(entry)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Order) != 2 || len(g.Diagnostics) != 0 {
		t.Fatalf("order=%v diagnostics=%+v", g.Order, g.Diagnostics)
	}
}

func TestLoadGraphAccumulatesMissingInclude(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte("include \"missing.bean\"\ninclude \"missing.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGraph(entry)
	if err != nil {
		t.Fatal(err)
	}
	resolved, _ := filepath.EvalSymlinks(entry)
	if len(g.Order) != 1 || len(g.Diagnostics) != 2 || g.Diagnostics[0].Code != "E-INCLUDE-READ" || g.Diagnostics[0].SourcePath != resolved {
		t.Fatalf("order=%v diagnostics=%+v", g.Order, g.Diagnostics)
	}
}

func TestGraphDisplayPathsAreRelativeAndConstrained(t *testing.T) {
	root := t.TempDir()
	ledgerDir := filepath.Join(root, "ledger")
	if err := os.Mkdir(ledgerDir, 0o700); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(ledgerDir, "main.bean")
	childDir := filepath.Join(ledgerDir, "nested")
	if err := os.Mkdir(childDir, 0o700); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(childDir, "child.bean")
	outside := filepath.Join(root, "shared.bean")
	if err := os.WriteFile(entry, []byte("include \"nested/child.bean\"\ninclude \"../shared.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	graph, err := LoadGraph(entry)
	if err != nil {
		t.Fatal(err)
	}
	paths := graph.DisplayPaths()
	if len(paths) != 3 || paths[0] != "main.bean" || paths[1] != "nested/child.bean" || paths[2] != "include/3/shared.bean" {
		t.Fatalf("display paths=%v", paths)
	}
	if _, ok := graph.FileIDForDisplayPath("/private/ledger/main.bean"); ok {
		t.Fatal("absolute display path accepted")
	}
	if id, ok := graph.FileIDForDisplayPath("nested/child.bean"); !ok || id != graph.Order[1] {
		t.Fatalf("child lookup id=%d ok=%v order=%v", id, ok, graph.Order)
	}
	if got := SafeDisplayPath("/private/ledger/main.bean"); got != "main.bean" {
		t.Fatalf("safe absolute path=%q", got)
	}
	if got := SafeDisplayPath("../private/ledger/main.bean"); got != "main.bean" {
		t.Fatalf("safe traversal path=%q", got)
	}
}

func TestSourceFileDefensivelyIndexesAndClampsRanges(t *testing.T) {
	data := []byte("aé\r\nlast")
	f := NewSourceFile(7, "ledger.bean", data)
	data[0] = 'x'
	if got := string(f.Data); got != "aé\r\nlast" {
		t.Fatalf("source data was not copied: %q", got)
	}
	if got := f.Position(-1); got != (Position{Line: 1, Column: 1}) {
		t.Fatalf("negative position=%+v", got)
	}
	if got := f.Position(2); got != (Position{Offset: 2, Line: 1, Column: 2}) {
		t.Fatalf("continuation-byte position=%+v", got)
	}
	if got := f.Position(100); got != (Position{Offset: len(f.Data), Line: 2, Column: 5}) {
		t.Fatalf("clamped position=%+v", got)
	}
	span := f.Span(-4, 100)
	if span.Start != 0 || span.End != len(f.Data) || f.Text(span) != "aé\r\nlast" {
		t.Fatalf("clamped span=%+v text=%q", span, f.Text(span))
	}
	if got := f.Text(Span{File: 99}); got != "" {
		t.Fatalf("foreign span text=%q", got)
	}
	if got := f.LineText(1); got != "aé" {
		t.Fatalf("line 1=%q", got)
	}
	if got := f.LineText(2); got != "last" {
		t.Fatalf("line 2=%q", got)
	}
	if got := f.LineText(3); got != "" {
		t.Fatalf("out-of-range line=%q", got)
	}
	if f.LineCount() != 2 || f.String() != "ledger.bean" {
		t.Fatalf("lines=%d string=%q", f.LineCount(), f.String())
	}
}

func TestSourceNilAndSpanFormatting(t *testing.T) {
	var f *SourceFile
	if f.String() != "<nil>" || f.Position(1) != (Position{}) || f.LineCount() != 0 || f.Text(Span{}) != "" {
		t.Fatalf("nil source behavior is inconsistent")
	}
	if got := (Span{}).String(); got != "<unknown>" {
		t.Fatalf("unknown span=%q", got)
	}
	oneLine := Span{File: 2, Start: 4, End: 6, StartLine: 3, StartColumn: 5, EndLine: 3, EndColumn: 7}
	if !oneLine.Valid() || oneLine.Empty() || oneLine.String() != "file#2:3:5-7" {
		t.Fatalf("one-line span=%+v string=%q", oneLine, oneLine.String())
	}
	multiLine := oneLine
	multiLine.End = multiLine.Start
	multiLine.EndLine, multiLine.EndColumn = 4, 2
	if !multiLine.Empty() || multiLine.String() != "file#2:3:5-4:2" {
		t.Fatalf("multi-line span=%+v string=%q", multiLine, multiLine.String())
	}
}

func TestLoadGraphRecognizesEscapedIncludesAndKeepsTraversalDeterministic(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	child := filepath.Join(dir, "child name.bean")
	if err := os.WriteFile(entry, []byte("; include \"ignored.bean\"\ninclude\t\"child name.bean\"\ninclude no-quote\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	graph, err := Load(entry)
	if err != nil {
		t.Fatal(err)
	}
	if got := graph.DisplayPaths(); !reflect.DeepEqual(got, []string{"main.bean", "child name.bean"}) {
		t.Fatalf("display paths=%v", got)
	}
	if id, ok := graph.FileIDForDisplayPath("../main.bean"); ok || id != 0 {
		t.Fatalf("traversal lookup id=%v ok=%v", id, ok)
	}
	if _, err := LoadGraph(filepath.Join(dir, "missing.bean")); err == nil {
		t.Fatal("missing entry did not fail")
	}
}

func TestGraphPathHelpersRejectUnsafeValues(t *testing.T) {
	if (*Graph)(nil).File(1) != nil || (*Graph)(nil).Path(1) != "" || (*Graph)(nil).DisplayPath(1) != "" || (*Graph)(nil).DisplayPaths() != nil {
		t.Fatal("nil graph accessors returned data")
	}
	if _, ok := (*Graph)(nil).FileIDForDisplayPath("main.bean"); ok {
		t.Fatal("nil graph resolved a path")
	}
	cases := map[string]string{
		"": "", ".": "", "nested/file.bean": "nested/file.bean", "../secret.bean": "secret.bean", "/tmp/secret.bean": "secret.bean",
	}
	for input, want := range cases {
		if got := SafeDisplayPath(input); got != want {
			t.Errorf("SafeDisplayPath(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestIncludeScannerAndSpanAliasesHandleEscapesAndInvalidRanges(t *testing.T) {
	file := NewSourceFile(1, "fixture.bean", []byte("include \"child\\\\name\\\".bean\"\ninclude \"line\\nfeed.bean\"\ninclude \"unterminated\n"))
	matches := scanIncludes(file)
	if len(matches) != 2 || matches[0].path != `child\name".bean` || matches[1].path != "line\nfeed.bean" {
		t.Fatalf("include matches=%+v", matches)
	}
	if scanIncludes(nil) != nil || closingQuote(`unterminated\\`) != -1 || closingQuote(`a\\\"b`) != -1 || unescapeString(`a\\qb`) != `a\qb` {
		t.Fatal("scanner edge cases are inconsistent")
	}
	span := file.SpanAt(4, 2)
	if span.Start != 4 || span.End != 4 || file.Text(Span{File: file.ID, Start: -1, End: 2}) == "" {
		t.Fatalf("span alias/range handling span=%+v", span)
	}
}
