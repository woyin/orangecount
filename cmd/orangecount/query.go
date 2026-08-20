// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"orangecount/internal/diagnostic"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
)

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
	options, ok := validateQueryOptions(queryOptions{locale: *locale, format: *format}, *jsonOutput, *csvOutput, fs.NArg(), stderr)
	if !ok {
		return 2
	}
	result := snapshot.Build(fs.Arg(0))
	if result.Snapshot == nil || result.Err != nil || hasDiagnosticErrors(result.Diagnostics) {
		if err := diagnostic.RenderHuman(stderr, result.Diagnostics, options.locale); err != nil {
			return 1
		}
		return 1
	}
	value, err := query.Evaluate(fs.Arg(1), result.Snapshot.Evaluation())
	if err != nil {
		fmt.Fprintf(stderr, "orangecount query: %s\n", err)
		return 1
	}
	return writeQueryResult(stdout, stderr, value, options.format)
}

type queryOptions struct{ locale, format string }

func validateQueryOptions(options queryOptions, jsonOutput, csvOutput bool, argc int, stderr io.Writer) (queryOptions, bool) {
	if options.locale != "en" && options.locale != "zh-CN" {
		fmt.Fprintf(stderr, "orangecount: unsupported locale %q\n", options.locale)
		return options, false
	}
	if jsonOutput && csvOutput {
		fmt.Fprintln(stderr, "orangecount query: --json and --csv cannot be combined")
		return options, false
	}
	switch {
	case jsonOutput:
		options.format = "json"
	case csvOutput:
		options.format = "csv"
	}
	if options.format != "json" && options.format != "csv" {
		fmt.Fprintf(stderr, "orangecount query: unsupported format %q\n", options.format)
		return options, false
	}
	if argc != 2 {
		fmt.Fprintln(stderr, "orangecount query: expected an entry ledger path and query text")
		return options, false
	}
	return options, true
}

func writeQueryResult(stdout, stderr io.Writer, value query.Result, format string) int {
	if format == "csv" {
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
