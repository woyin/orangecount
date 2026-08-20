// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"flag"
	"fmt"
	"io"

	"orangecount/internal/diagnostic"
	"orangecount/internal/repairguidance"
	"orangecount/internal/snapshot"
)

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
	result := snapshot.Build(fs.Arg(0))
	ds := result.Diagnostics
	if *jsonOutput {
		if err := diagnostic.RenderJSON(stdout, ds, *locale); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else if err := renderCheckHuman(stdout, ds, *locale); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if result.Snapshot == nil || result.Err != nil || hasDiagnosticErrors(ds) {
		return 1
	}
	return 0
}

func renderCheckHuman(w io.Writer, ds []diagnostic.Diagnostic, locale string) error {
	if err := diagnostic.RenderHuman(w, ds, locale); err != nil {
		return err
	}
	for _, value := range ds {
		if value.Severity != diagnostic.Error {
			continue
		}
		guide, ok := repairguidance.Lookup(value.Code, locale)
		if !ok {
			continue
		}
		if _, err := fmt.Fprintf(w, "  -> %s (help: %s)\n", guide.ShortAction, guide.Topic); err != nil {
			return err
		}
	}
	return nil
}
