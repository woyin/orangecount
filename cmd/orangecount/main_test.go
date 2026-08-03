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

func TestRunCheckHumanAndJSON(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte("plugin \"migration\"\n2000-02-30 open Assets:Cash USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := run([]string{"check", "--locale", "zh-CN", entry}, &out, &errOut)
	if code != 1 || !strings.Contains(out.String(), "日期无效") || errOut.Len() != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	code = run([]string{"check", "--json", entry}, &out, &errOut)
	if code != 1 || !strings.Contains(out.String(), `"code":"E-PARSE-DATE"`) {
		t.Fatalf("json code=%d output=%q", code, out.String())
	}
}

func TestRunQueryJSONAndCSV(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := run([]string{"query", entry, "SELECT account, balance FROM accounts ORDER BY account"}, &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), `"columns"`) || errOut.Len() != 0 {
		t.Fatalf("json code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run([]string{"query", "--csv", entry, "SELECT account FROM accounts"}, &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), "account\n") {
		t.Fatalf("csv code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}
