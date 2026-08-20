// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"fmt"
	"io"
	"os"

	"orangecount/internal/diagnostic"
	"orangecount/internal/snapshot"
)

var version = "0.1.3-dev"

const defaultServeAddr = "127.0.0.1:5000"

func main() { os.Exit(runWithInput(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int { return runWithInput(args, nil, stdout, stderr) }

func runWithInput(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || (args[0] == "help" && len(args) == 1) {
		fmt.Fprintln(stdout, "orangecount check [--locale en|zh-CN] [--json] <entry.bean>")
		fmt.Fprintf(stdout, "orangecount serve [--addr %s] [--document-root DIR] <entry.bean>\n", defaultServeAddr)
		fmt.Fprintln(stdout, "orangecount query [--locale en|zh-CN] [--format json|csv] <entry.bean> <query>")
		fmt.Fprintln(stdout, "orangecount export -out FILE|DIR <entry.bean>")
		fmt.Fprintln(stdout, "orangecount dialectize -out FILE|DIR <entry.bean>")
		fmt.Fprintln(stdout, "orangecount help [--locale en|zh-CN] diagnostics/<CODE>")
		return 0
	}
	if args[0] == "version" || args[0] == "--version" {
		fmt.Fprintln(stdout, version)
		return 0
	}
	switch args[0] {
	case "help":
		return runHelp(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "serve":
		return runServe(args[1:], stdin, stdout, stderr)
	case "query":
		return runQuery(args[1:], stdout, stderr)
	case "export":
		return runExport(args[1:], stdout, stderr)
	case "dialectize":
		return runDialectize(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "orangecount: unknown command %q\n", args[0])
		return 2
	}
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
