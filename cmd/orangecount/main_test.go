// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orangecount/internal/diagnostic"
	"orangecount/internal/source"
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

func TestRunDispatchAndArgumentValidation(t *testing.T) {
	oldVersion := version
	version = "test-version"
	defer func() { version = oldVersion }()
	cases := []struct {
		name   string
		args   []string
		want   int
		output string
		stderr string
	}{
		{name: "empty help", want: 0, output: "orangecount check"},
		{name: "version", args: []string{"version"}, want: 0, output: "test-version"},
		{name: "unknown", args: []string{"unknown"}, want: 2, stderr: "unknown command"},
		{name: "check invalid locale", args: []string{"check", "--locale", "fr", "ledger.bean"}, want: 2, stderr: "unsupported locale"},
		{name: "check missing entry", args: []string{"check"}, want: 2, stderr: "expected exactly one"},
		{name: "query invalid locale", args: []string{"query", "--locale", "fr"}, want: 2, stderr: "unsupported locale"},
		{name: "query conflicting shortcuts", args: []string{"query", "--json", "--csv"}, want: 2, stderr: "cannot be combined"},
		{name: "query invalid format", args: []string{"query", "--format", "text"}, want: 2, stderr: "unsupported format"},
		{name: "query missing arguments", args: []string{"query"}, want: 2, stderr: "expected an entry"},
		{name: "serve invalid locale", args: []string{"serve", "--locale", "fr"}, want: 2, stderr: "unsupported locale"},
		{name: "serve missing entry", args: []string{"serve"}, want: 2, stderr: "expected exactly one"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if got := run(tc.args, &out, &errOut); got != tc.want {
				t.Fatalf("run(%v)=%d, want %d; stdout=%q stderr=%q", tc.args, got, tc.want, out.String(), errOut.String())
			}
			if tc.output != "" && !strings.Contains(out.String(), tc.output) {
				t.Fatalf("stdout=%q does not contain %q", out.String(), tc.output)
			}
			if tc.stderr != "" && !strings.Contains(errOut.String(), tc.stderr) {
				t.Fatalf("stderr=%q does not contain %q", errOut.String(), tc.stderr)
			}
		})
	}
}

func TestRunQueryAndCheckFailuresRemainReported(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.bean")
	if err := os.WriteFile(valid, []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"query", valid, "not a query"}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "orangecount query:") {
		t.Fatalf("invalid query code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := run([]string{"check", filepath.Join(dir, "missing.bean")}, &out, &errOut); code != 1 || !strings.Contains(out.String(), "E-INCLUDE-READ") {
		t.Fatalf("missing check code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}

func TestParsePortOwnersAndPortHelpers(t *testing.T) {
	owners := parsePortOwners("p12\ncfirst\np12\ncduplicate\npinvalid\ncignored\np42\n\np77\ncthird\n")
	want := []portOwner{{PID: 12, Command: "first"}, {PID: 42, Command: "unknown process"}, {PID: 77, Command: "third"}}
	if len(owners) != len(want) {
		t.Fatalf("owners=%+v", owners)
	}
	for i := range want {
		if owners[i] != want[i] {
			t.Fatalf("owner[%d]=%+v want=%+v", i, owners[i], want[i])
		}
	}
	originalInspect := inspectPortOwners
	defer func() { inspectPortOwners = originalInspect }()
	inspectPortOwners = func(port string) ([]portOwner, error) {
		if port != "5001" {
			t.Fatalf("port=%q", port)
		}
		return want[:1], nil
	}
	if got, err := portOwnersAt("127.0.0.1:5001"); err != nil || len(got) != 1 || got[0] != want[0] {
		t.Fatalf("owners=%+v err=%v", got, err)
	}
	if _, err := portOwnersAt("not-an-address"); err == nil {
		t.Fatal("invalid address was accepted")
	}
}

func TestLsofPortOwnersDelegatesProcessExecutionAndPreservesErrors(t *testing.T) {
	original := runLsof
	defer func() { runLsof = original }()
	runLsof = func(port string) ([]byte, error) {
		if port != "5000" {
			t.Fatalf("port=%q", port)
		}
		return []byte("p77\nclocal-server\n"), nil
	}
	owners, err := lsofPortOwners("5000")
	if err != nil || len(owners) != 1 || owners[0] != (portOwner{PID: 77, Command: "local-server"}) {
		t.Fatalf("owners=%+v err=%v", owners, err)
	}
	runLsof = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	if _, err := lsofPortOwners("5000"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lsof error=%v", err)
	}
}

func TestResolvePortConflictDeclinesNonInteractiveAndInspectionErrors(t *testing.T) {
	originalInspect := inspectPortOwners
	defer func() { inspectPortOwners = originalInspect }()
	inspectPortOwners = func(string) ([]portOwner, error) { return nil, os.ErrPermission }
	var output bytes.Buffer
	retry, err := resolvePortConflict(defaultServeAddr, nil, &output)
	if err != nil || retry || !strings.Contains(output.String(), "unable to identify") {
		t.Fatalf("retry=%v err=%v output=%q", retry, err, output.String())
	}
	inspectPortOwners = func(string) ([]portOwner, error) { return []portOwner{{PID: 1, Command: "busy"}}, nil }
	output.Reset()
	retry, err = resolvePortConflict(defaultServeAddr, nil, &output)
	if err != nil || retry || !strings.Contains(output.String(), "run interactively") {
		t.Fatalf("retry=%v err=%v output=%q", retry, err, output.String())
	}
}

func TestRunServeValidatesDocumentRootsAndPortConflictsBeforeServing(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "valid.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := runWithInput([]string{"serve", "--document-root", filepath.Join(dir, "missing"), entry}, strings.NewReader(""), &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "document root") {
		t.Fatalf("document root code=%d out=%q err=%q", code, out.String(), errOut.String())
	}

	originalInspect := inspectPortOwners
	defer func() { inspectPortOwners = originalInspect }()
	inspectPortOwners = func(port string) ([]portOwner, error) {
		if port != "5000" {
			t.Fatalf("port=%q", port)
		}
		return []portOwner{{PID: 123, Command: "existing"}}, nil
	}
	out.Reset()
	errOut.Reset()
	if code := runWithInput([]string{"serve", entry}, nil, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "currently used") || !strings.Contains(errOut.String(), "run interactively") {
		t.Fatalf("conflict code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}

func TestCLIHelpersRenderAndWaitWithoutSideEffects(t *testing.T) {
	if snapshotID(nil) != "" {
		t.Fatal("nil snapshot ID was non-empty")
	}
	if hasDiagnosticErrors(nil) {
		t.Fatal("empty diagnostics reported an error")
	}
	if !hasDiagnosticErrors([]diagnostic.Diagnostic{{Severity: diagnostic.Warning}, {Severity: diagnostic.Error}}) {
		t.Fatal("error diagnostic was missed")
	}
	var output bytes.Buffer
	if err := renderDiagnostics(&output, "en", true, []diagnostic.Diagnostic{diagnostic.New("E-PARSE-DATE", diagnostic.Error, source.Span{})}); err != nil || !strings.Contains(output.String(), `"code":"E-PARSE-DATE"`) {
		t.Fatalf("json diagnostics=%q err=%v", output.String(), err)
	}
	output.Reset()
	if err := renderDiagnostics(&output, "en", false, []diagnostic.Diagnostic{diagnostic.New("E-PARSE-DATE", diagnostic.Error, source.Span{})}); err != nil || !strings.Contains(output.String(), "invalid date") {
		t.Fatalf("human diagnostics=%q err=%v", output.String(), err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if err := waitForPortRelease(addr, time.Second); err != nil {
		t.Fatalf("free port was not reusable: %v", err)
	}
}

func TestPortConflictFailureAndServeAddressContracts(t *testing.T) {
	originalInspect, originalStop, originalWait := inspectPortOwners, stopPortOwner, waitForPort
	defer func() { inspectPortOwners, stopPortOwner, waitForPort = originalInspect, originalStop, originalWait }()
	inspectPortOwners = func(string) ([]portOwner, error) { return []portOwner{{PID: 77, Command: "busy"}}, nil }
	stopPortOwner = func(int) error { return os.ErrPermission }
	var output bytes.Buffer
	if retry, err := resolvePortConflict(defaultServeAddr, strings.NewReader("y\n"), &output); retry || !errors.Is(err, os.ErrPermission) || !strings.Contains(err.Error(), "PID 77") {
		t.Fatalf("stop failure retry=%v err=%v output=%q", retry, err, output.String())
	}
	stopPortOwner = func(int) error { return nil }
	waitForPort = func(string, time.Duration) error { return errors.New("still busy") }
	if retry, err := resolvePortConflict(defaultServeAddr, strings.NewReader("是\n"), &output); retry || err == nil || !strings.Contains(err.Error(), "still busy") {
		t.Fatalf("wait failure retry=%v err=%v", retry, err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := waitForPortRelease(listener.Addr().String(), 0); err == nil || !strings.Contains(err.Error(), "still in use") {
		t.Fatalf("occupied port wait error=%v", err)
	}
	if err := terminatePortOwner(99999999); err == nil {
		t.Fatal("signalling a non-existent process unexpectedly succeeded")
	}

	dir := t.TempDir()
	entry := filepath.Join(dir, "valid.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	inspectPortOwners = func(string) ([]portOwner, error) { return nil, nil }
	var stdout, stderr bytes.Buffer
	if code := runWithInput([]string{"serve", "--addr", "not-an-address", entry}, nil, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "loopback-only") {
		t.Fatalf("invalid addr code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	inspectionCalls := 0
	inspectPortOwners = func(string) ([]portOwner, error) {
		inspectionCalls++
		if inspectionCalls == 1 {
			return nil, nil
		}
		return []portOwner{{PID: 88, Command: "occupied"}}, nil
	}
	stdout.Reset()
	stderr.Reset()
	if code := runWithInput([]string{"serve", "--addr", occupied.Addr().String(), entry}, strings.NewReader("no\n"), &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "OrangeCount was not started") {
		t.Fatalf("late conflict code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
