// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package source

import (
	"os"
	"path/filepath"
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
