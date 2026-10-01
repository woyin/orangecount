// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

func semanticsFlag(fs *flag.FlagSet) *snapshot.BuildOptions {
	options := &snapshot.BuildOptions{}
	fs.Func("semantics", "accounting semantics (legacy or v3; default legacy)", func(value string) error {
		profile, err := ledger.ParseSemanticsProfile(value)
		if err == nil {
			options.Evaluation.SemanticsProfile = profile
		}
		return err
	})
	return options
}

func runCompareSemantics(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("compare-semantics", flag.ContinueOnError)
	fs.SetOutput(stderr)
	details := fs.Bool("details", false, "include private ledger details for local review")
	failOnChange := fs.Bool("fail-on-change", false, "exit 1 if accounting results change")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "orangecount compare-semantics: expected one ledger path")
		return 2
	}
	graph, err := source.LoadGraph(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, "orangecount compare-semantics: unable to load input graph")
		return 1
	}
	comparison := snapshot.CompareSemantics(graph, *details)
	if err := json.NewEncoder(stdout).Encode(comparison); err != nil {
		return 1
	}
	if *failOnChange {
		for _, count := range comparison.ChangeCounts {
			if count > 0 {
				return 1
			}
		}
	}
	return 0
}
