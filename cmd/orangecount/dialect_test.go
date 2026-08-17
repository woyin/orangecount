// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const dialectLedger = `2000-01-01 open Assets:WeChat USD
2000-01-01 open Assets:Cash USD
2000-01-01 open Expenses:Food USD
option "operating_currency" "USD"
2026-08-12 28 @WeChat -> @Food "美团" : 工作午餐 #food
15 @Cash -> @Food : 通勤
`

func writeEntry(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "main.bean")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExportAndDialectizeRoundTripThroughCLI(t *testing.T) {
	entry := writeEntry(t, dialectLedger)
	dir := t.TempDir()
	exported := filepath.Join(dir, "export.bean")
	dialectized := filepath.Join(dir, "roundtrip.bean")
	fixpoint := filepath.Join(dir, "fixpoint.bean")

	var out bytes.Buffer
	if code := runExport([]string{"-out", exported, entry}, &out, &out); code != 0 {
		t.Fatalf("export failed: %s", out.String())
	}
	exportText := readFileText(t, exported)
	if strings.Contains(exportText, "->") {
		t.Fatalf("dialect shorthand survived export:\n%s", exportText)
	}
	if !strings.Contains(exportText, `2026-08-12 * "美团" "工作午餐" #food`) {
		t.Fatalf("compiled header missing:\n%s", exportText)
	}

	out.Reset()
	if code := runDialectize([]string{"-out", dialectized, exported}, &out, &out); code != 0 {
		t.Fatalf("dialectize failed: %s", out.String())
	}

	out.Reset()
	if code := runExport([]string{"-out", fixpoint, dialectized}, &out, &out); code != 0 {
		t.Fatalf("second export failed: %s", out.String())
	}
	if exportText != readFileText(t, fixpoint) {
		t.Fatalf("fixpoint broken:\nfirst:\n%s\nsecond:\n%s", exportText, readFileText(t, fixpoint))
	}
}

func TestExportRequiresOutAndEntry(t *testing.T) {
	var out bytes.Buffer
	if code := runExport([]string{}, &out, &out); code != 2 {
		t.Fatalf("export without args exit=%d", code)
	}
	if code := runExport([]string{"-out", "x.bean"}, &out, &out); code != 2 {
		t.Fatalf("export without entry exit=%d", code)
	}
	if code := runDialectize([]string{}, &out, &out); code != 2 {
		t.Fatalf("dialectize without args exit=%d", code)
	}
}

func TestExportRejectsBrokenDialect(t *testing.T) {
	entry := writeEntry(t, dialectLedger+"2026-08-14 5 @Cash -> @Nowhere\n")
	var out bytes.Buffer
	code := runExport([]string{"-out", filepath.Join(t.TempDir(), "x.bean"), entry}, &out, &out)
	if code != 1 {
		t.Fatalf("export of failing dialect exit=%d out=%s", code, out.String())
	}
	if !strings.Contains(out.String(), "E-DIALECT") {
		t.Fatalf("expected E-DIALECT diagnostic, got: %s", out.String())
	}
}

func readFileText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
