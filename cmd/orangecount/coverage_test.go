// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests in this file exercise the CLI's dispatch, help, query, and serve
// edge branches that the primary command tests skip.

func TestRunWithInputDispatch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runWithInput([]string{"bogus"}, nil, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	if code := runWithInput([]string{"version"}, nil, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "0.") {
		t.Fatalf("version code=%d stdout=%q", code, stdout.String())
	}
	stdout.Reset()
	if code := runWithInput([]string{"--version"}, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("--version code=%d", code)
	}
	stdout.Reset()
	if code := runWithInput(nil, nil, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "orangecount check") {
		t.Fatalf("no args code=%d stdout=%q", code, stdout.String())
	}
	stdout.Reset()
	if code := runWithInput([]string{"--help"}, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("--help code=%d", code)
	}
	stdout.Reset()
	if code := runWithInput([]string{"help"}, nil, &stdout, &stderr); code != 0 {
		t.Fatalf("help-only code=%d", code)
	}
}

func TestRunHelpBadLocaleAndUnknownTopic(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runHelp([]string{"--locale", "fr", "diagnostics/E-PARSE-DATE"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "locale") {
		t.Fatalf("bad locale code=%d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runHelp([]string{"diagnostics/E-NOT-A-CODE"}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "not found") {
		t.Fatalf("unknown topic code=%d stderr=%q", code, stderr.String())
	}
	stderr.Reset()
	if code := runHelp([]string{"--bogus-flag"}, &stdout, &stderr); code != 2 {
		t.Fatalf("bad flag code=%d", code)
	}
	stdout.Reset()
	if code := runHelp([]string{}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "expected a topic") {
		t.Fatalf("topic-less help code=%d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runHelp([]string{"--locale=zh-CN", "diagnostics/E-PARSE-DATE"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "主题") {
		t.Fatalf("zh-CN help code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := runHelp([]string{"plain-topic"}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "unknown topic") {
		t.Fatalf("non-diagnostics topic code=%d stderr=%q", code, stderr.String())
	}
}

func TestNormalizeHelpArgsOrders(t *testing.T) {
	// The documented `help diagnostics/<CODE> --locale ...` spelling is
	// normalized to Go flag order: flags first, positional topic last.
	got := normalizeHelpArgs([]string{"diagnostics/E-PARSE-DATE", "--locale", "zh-CN"})
	if strings.Join(got, " ") != "--locale zh-CN diagnostics/E-PARSE-DATE" {
		t.Fatalf("reorder=%v", got)
	}
	if got := normalizeHelpArgs([]string{"--locale=zh-CN", "topic"}); strings.Join(got, " ") != "--locale=zh-CN topic" {
		t.Fatalf("eq-form=%v", got)
	}
	already := []string{"diagnostics/E-PARSE-DATE"}
	if got := normalizeHelpArgs(already); strings.Join(got, " ") != strings.Join(already, " ") {
		t.Fatalf("topic-first passthrough=%v", got)
	}
}

func writeLedger(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "entry.bean")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunQueryValidationAndFormats(t *testing.T) {
	entry := writeLedger(t, "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n")
	var stdout, stderr bytes.Buffer
	if code := runQuery([]string{"--locale", "fr", entry, "SELECT account FROM accounts"}, &stdout, &stderr); code != 2 {
		t.Fatalf("bad locale code=%d", code)
	}
	if code := runQuery([]string{"--json", "--csv", entry, "SELECT 1"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "cannot be combined") {
		t.Fatalf("combined flags code=%d stderr=%q", code, stderr.String())
	}
	if code := runQuery([]string{"--format", "xml", entry, "SELECT 1"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "unsupported format") {
		t.Fatalf("bad format code=%d stderr=%q", code, stderr.String())
	}
	if code := runQuery([]string{entry}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "expected an entry ledger path") {
		t.Fatalf("missing args code=%d stderr=%q", code, stderr.String())
	}
	if code := runQuery([]string{"--bad", entry, "SELECT 1"}, &stdout, &stderr); code != 2 {
		t.Fatalf("bad flag code=%d", code)
	}
	missing := writeLedger(t, "2000-01-01 open Assets:Cash USD\n")
	if code := runQuery([]string{missing, "SELECT broken ("}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "orangecount query") {
		t.Fatalf("bad query code=%d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	if code := runQuery([]string{"--json", entry, "SELECT account FROM accounts"}, &stdout, &stderr); code != 0 {
		t.Fatalf("json alias code=%d", code)
	}
	var result struct {
		Columns []string `json:"columns"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || len(result.Columns) != 1 {
		t.Fatalf("json output=%q err=%v columns=%v", stdout.String(), err, result.Columns)
	}
	stdout.Reset()
	if code := runQuery([]string{"--csv", entry, "SELECT account FROM accounts"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "account") {
		t.Fatalf("csv alias code=%d stdout=%q", code, stdout.String())
	}
	// The CSV writer's error path: a failing writer surfaces as exit 1.
	failing := &failingWriter{}
	if code := runQuery([]string{"--csv", entry, "SELECT account FROM accounts"}, failing, &stderr); code != 1 {
		t.Fatalf("failing csv writer code=%d", code)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errDiskFull }

var errDiskFull = &diskFullError{}

type diskFullError struct{}

func (*diskFullError) Error() string { return "disk full" }

func TestRunCheckBranches(t *testing.T) {
	entry := writeLedger(t, "2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Cash USD\n")
	var stdout, stderr bytes.Buffer
	if code := runCheck([]string{"--locale", "fr", entry}, &stdout, &stderr); code != 2 {
		t.Fatalf("bad locale code=%d", code)
	}
	if code := runCheck([]string{}, &stdout, &stderr); code != 2 {
		t.Fatalf("missing path code=%d", code)
	}
	if code := runCheck([]string{"--bad", entry}, &stdout, &stderr); code != 2 {
		t.Fatalf("bad flag code=%d", code)
	}
	stdout.Reset()
	if code := runCheck([]string{"--json", entry}, &stdout, &stderr); code != 1 || !strings.Contains(stdout.String(), "E-EVAL-OPEN") {
		t.Fatalf("json check code=%d stdout=%q", code, stdout.String())
	}
	stdout.Reset()
	if code := runCheck([]string{entry}, &stdout, &stderr); code != 1 || !strings.Contains(stdout.String(), "E-EVAL-OPEN") || !strings.Contains(stdout.String(), "-> ") {
		t.Fatalf("human check code=%d stdout=%q", code, stdout.String())
	}
	valid := writeLedger(t, "2000-01-01 open Assets:Cash USD\n")
	stdout.Reset()
	if code := runCheck([]string{valid}, &stdout, &stderr); code != 0 {
		t.Fatalf("valid check code=%d stdout=%q", code, stdout.String())
	}
}

func TestRunServeFlagValidationAndSnapshotErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runServe([]string{"--locale", "fr", "x.bean"}, nil, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "locale") {
		t.Fatalf("bad locale code=%d stderr=%q", code, stderr.String())
	}
	if code := runServe([]string{}, nil, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "exactly one") {
		t.Fatalf("missing path code=%d stderr=%q", code, stderr.String())
	}
	if code := runServe([]string{"--bad", "x.bean"}, nil, &stdout, &stderr); code != 2 {
		t.Fatalf("bad flag code=%d", code)
	}
	stdout.Reset()
	stderr.Reset()
	invalid := writeLedger(t, "2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Cash USD\n")
	if code := runServe([]string{invalid}, nil, &stdout, &stderr); code != 1 || !strings.Contains(stdout.String(), "E-EVAL-OPEN") {
		t.Fatalf("invalid ledger serve code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestServeHelpersDirectly(t *testing.T) {
	if id := snapshotID(nil); id != "" {
		t.Fatalf("nil snapshot id=%q", id)
	}
	if waitForPortRelease("127.0.0.1:1", 0) == nil {
		t.Fatal("port 1 wait should time out or fail")
	}
}

func TestServeInitialSnapshotAndPortPrecheck(t *testing.T) {
	valid := writeLedger(t, "2000-01-01 open Assets:Cash USD\n")
	var stdout, stderr bytes.Buffer
	if _, initial, code := serveInitialSnapshot(valid, "en", &stdout, &stderr); initial == nil || code != 0 {
		t.Fatalf("valid initial code=%d stderr=%q", code, stderr.String())
	}
	invalid := writeLedger(t, "2000-01-01 open Assets:Cash USD\n2000-01-01 open Assets:Cash USD\n")
	if _, initial, code := serveInitialSnapshot(invalid, "en", &stdout, &stderr); initial != nil || code != 1 {
		t.Fatalf("invalid initial code=%d initial=%v", code, initial)
	}
	stdout.Reset()
	stderr.Reset()
	// No owners: the pre-check is a no-op that lets serving continue.
	inspectPortOwners = func(string) ([]portOwner, error) { return nil, nil }
	if code := resolveExistingPortConflict("127.0.0.1:1", nil, &stderr); code != 0 {
		t.Fatalf("ownerless precheck code=%d stderr=%q", code, stderr.String())
	}
	// Owners plus a declined prompt aborts with code 1.
	inspectPortOwners = func(string) ([]portOwner, error) {
		return []portOwner{{PID: os.Getpid(), Command: "self"}}, nil
	}
	if code := resolveExistingPortConflict("127.0.0.1:1", strings.NewReader("n\n"), &stderr); code != 1 || !strings.Contains(stderr.String(), "was not started") {
		t.Fatalf("declined precheck code=%d stderr=%q", code, stderr.String())
	}
}

func TestParsePortOwnersRecords(t *testing.T) {
	owners := parsePortOwners("p123\nccommand a\np123\ncsecond\n")
	if len(owners) != 1 || owners[0].PID != 123 || owners[0].Command != "command a" {
		t.Fatalf("owners=%+v", owners)
	}
	blank := parsePortOwners("x\n\np9\n")
	if len(blank) != 1 || blank[0].Command != "unknown process" {
		t.Fatalf("blank=%+v", blank)
	}
}
