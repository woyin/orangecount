// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"bytes"
	"testing"
)

func TestCLIExplicitSemantics(t *testing.T) {
	path := "../../testdata/fixtures/semantics/lifo-explicit.bean"
	for _, command := range [][]string{
		{"check", "--semantics", "v3", path},
		{"query", "--semantics", "v3", path, "SELECT account FROM accounts"},
	} {
		var out, err bytes.Buffer
		if code := run(command, &out, &err); code != 0 {
			t.Fatalf("%v: code=%d %s", command, code, err.String())
		}
	}
	var out, err bytes.Buffer
	if code := run([]string{"check", "--semantics", "bogus", path}, &out, &err); code != 2 {
		t.Fatalf("unknown semantics exit=%d", code)
	}
}

func TestCLICompareReportsChangesAndCanGate(t *testing.T) {
	var out, err bytes.Buffer
	path := "../../testdata/fixtures/semantics/lifo-implicit.bean"
	if code := run([]string{"compare-semantics", "--fail-on-change", path}, &out, &err); code != 1 {
		t.Fatalf("migration gate code=%d: %s", code, err.String())
	}
	if bytes.Contains(out.Bytes(), []byte("Income:Gains")) {
		t.Fatal("default comparison leaked account names")
	}
}
