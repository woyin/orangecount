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
	"time"
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

func TestResolvePortConflictReportsOwnerAndRequiresConfirmation(t *testing.T) {
	originalInspect, originalStop, originalWait := inspectPortOwners, stopPortOwner, waitForPort
	defer func() {
		inspectPortOwners, stopPortOwner, waitForPort = originalInspect, originalStop, originalWait
	}()
	inspectPortOwners = func(port string) ([]portOwner, error) {
		if port != "5000" {
			t.Fatalf("port=%q", port)
		}
		return []portOwner{{PID: 4242, Command: "example-server"}}, nil
	}
	stopped := []int{}
	stopPortOwner = func(pid int) error {
		stopped = append(stopped, pid)
		return nil
	}
	waitForPort = func(addr string, _ time.Duration) error {
		if addr != defaultServeAddr {
			t.Fatalf("addr=%q", addr)
		}
		return nil
	}

	var output bytes.Buffer
	retry, err := resolvePortConflict(defaultServeAddr, strings.NewReader("yes\n"), &output)
	if err != nil || !retry || len(stopped) != 1 || stopped[0] != 4242 {
		t.Fatalf("retry=%v err=%v stopped=%v", retry, err, stopped)
	}
	if text := output.String(); !strings.Contains(text, "PID 4242: example-server") || !strings.Contains(text, "Close the listed process") {
		t.Fatalf("output=%q", text)
	}

	output.Reset()
	stopped = nil
	retry, err = resolvePortConflict(defaultServeAddr, strings.NewReader("no\n"), &output)
	if err != nil || retry || len(stopped) != 0 || !strings.Contains(output.String(), "OrangeCount was not started.") {
		t.Fatalf("decline retry=%v err=%v stopped=%v output=%q", retry, err, stopped, output.String())
	}
}

func TestServeHelpUsesFixedDefaultPort(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"help"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), defaultServeAddr) || errOut.Len() != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}
