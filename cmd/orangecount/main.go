// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"orangecount/internal/diagnostic"
	"orangecount/internal/logging"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
	"orangecount/internal/web"
)

// version is overridable at build time via -ldflags "-X main.version=<tag>" so
// release binaries report their release tag; local builds keep the dev marker.
var version = "0.1.0-dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(stdout, "orangecount check [--locale en|zh-CN] [--json] <entry.bean>")
		fmt.Fprintln(stdout, "orangecount serve [--addr 127.0.0.1:0] [--document-root DIR] <entry.bean>")
		fmt.Fprintln(stdout, "orangecount query [--locale en|zh-CN] [--format json|csv] <entry.bean> <query>")
		return 0
	}
	if args[0] == "version" || args[0] == "--version" {
		fmt.Fprintln(stdout, version)
		return 0
	}
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "serve":
		return runServe(args[1:], stdout, stderr)
	case "query":
		return runQuery(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "orangecount: unknown command %q\n", args[0])
		return 2
	}
}

func runQuery(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(stderr)
	locale := fs.String("locale", "en", "diagnostic display locale (en or zh-CN)")
	format := fs.String("format", "json", "result format (json or csv)")
	jsonOutput := fs.Bool("json", false, "alias for --format json")
	csvOutput := fs.Bool("csv", false, "alias for --format csv")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *locale != "en" && *locale != "zh-CN" {
		fmt.Fprintf(stderr, "orangecount: unsupported locale %q\n", *locale)
		return 2
	}
	if *jsonOutput && *csvOutput {
		fmt.Fprintln(stderr, "orangecount query: --json and --csv cannot be combined")
		return 2
	}
	if *jsonOutput {
		*format = "json"
	}
	if *csvOutput {
		*format = "csv"
	}
	if *format != "json" && *format != "csv" {
		fmt.Fprintf(stderr, "orangecount query: unsupported format %q\n", *format)
		return 2
	}
	if fs.NArg() != 2 {
		fmt.Fprintln(stderr, "orangecount query: expected an entry ledger path and query text")
		return 2
	}
	result := snapshot.Build(fs.Arg(0))
	if result.Snapshot == nil || result.Err != nil || hasDiagnosticErrors(result.Diagnostics) {
		if err := diagnostic.RenderHuman(stderr, result.Diagnostics, *locale); err != nil {
			return 1
		}
		return 1
	}
	value, err := query.Evaluate(fs.Arg(1), result.Snapshot.Evaluation())
	if err != nil {
		fmt.Fprintf(stderr, "orangecount query: %s\n", err)
		return 1
	}
	if *format == "csv" {
		if err := value.WriteCSV(stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	locale := fs.String("locale", "en", "diagnostic display locale (en or zh-CN)")
	jsonOutput := fs.Bool("json", false, "render diagnostics as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *locale != "en" && *locale != "zh-CN" {
		fmt.Fprintf(stderr, "orangecount: unsupported locale %q\n", *locale)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "orangecount check: expected exactly one entry ledger path")
		return 2
	}
	entry := fs.Arg(0)
	result := snapshot.Build(entry)
	ds := result.Diagnostics
	if *jsonOutput {
		if err := diagnostic.RenderJSON(stdout, ds, *locale); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else if err := diagnostic.RenderHuman(stdout, ds, *locale); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if result.Snapshot == nil || result.Err != nil || hasDiagnosticErrors(ds) {
		return 1
	}
	return 0
}

func runServe(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	locale := fs.String("locale", "en", "display locale (en or zh-CN)")
	addr := fs.String("addr", "127.0.0.1:0", "loopback listen address")
	sensitiveLogs := fs.Bool("sensitive-logs", false, "temporarily include sensitive diagnostic fields in local logs")
	pollInterval := fs.Duration("poll", 250*time.Millisecond, "include graph polling interval")
	debounce := fs.Duration("debounce", 150*time.Millisecond, "reload debounce duration")
	var documentRoots []string
	fs.Func("document-root", "explicit document attachment root (repeatable)", func(value string) error {
		documentRoots = append(documentRoots, value)
		return nil
	})
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *locale != "en" && *locale != "zh-CN" {
		fmt.Fprintf(stderr, "orangecount: unsupported locale %q\n", *locale)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "orangecount serve: expected exactly one entry ledger path")
		return 2
	}
	entry := fs.Arg(0)
	initial := snapshot.Build(entry)
	if err := renderDiagnostics(stdout, *locale, false, initial.Diagnostics); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if initial.Snapshot == nil || initial.Err != nil || hasDiagnosticErrors(initial.Diagnostics) {
		return 1
	}
	roots, err := source.NewDocumentRoots(documentRoots)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	store := snapshot.NewStore(initial.Snapshot)
	server, err := web.NewServer(web.Config{Store: store, DocumentRoots: roots, Addr: *addr})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	logger := logging.New(stderr, logging.Options{Sensitive: *sensitiveLogs})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		_ = store.Watch(ctx, entry, snapshot.BuildOptions{}, snapshot.WatchOptions{PollInterval: *pollInterval, Debounce: *debounce}, func(result snapshot.ReloadResult) {
			_ = logger.Event("reload", map[string]any{"reload": true, "published": result.Published, "snapshot_id": snapshotID(result.Snapshot)})
		})
	}()
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(ctx) }()
	if err := server.WaitReady(ctx); err != nil {
		if ctx.Err() != nil {
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "serving on http://%s\n", server.Addr())
	if err := <-serveDone; err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func snapshotID(value *snapshot.Snapshot) string {
	if value == nil {
		return ""
	}
	return value.ID
}

func hasDiagnosticErrors(ds []diagnostic.Diagnostic) bool {
	for _, d := range ds {
		if d.Severity == diagnostic.Error {
			return true
		}
	}
	return false
}

func renderDiagnostics(w io.Writer, locale string, jsonOutput bool, ds []diagnostic.Diagnostic) error {
	if jsonOutput {
		return diagnostic.RenderJSON(w, ds, locale)
	}
	return diagnostic.RenderHuman(w, ds, locale)
}
