// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"flag"
	"fmt"
	"io"

	"orangecount/internal/diagnostic"
	"orangecount/internal/snapshot"
	"orangecount/internal/sourceexport"
)

func runExport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	buildOptions := semanticsFlag(fs)
	out := fs.String("out", "", "atomic output outside source directory; existing parent required; include graph requires a new directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "orangecount export: expected exactly one entry ledger path")
		return 2
	}
	if *out == "" {
		fmt.Fprintln(stderr, "orangecount export: -out is required")
		return 2
	}
	return writeConverted(fs.Arg(0), *out, "export", stdout, stderr, false, *buildOptions)
}

func runDialectize(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dialectize", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "atomic output outside source directory; existing parent required; include graph requires a new directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "orangecount dialectize: expected exactly one entry ledger path")
		return 2
	}
	if *out == "" {
		fmt.Fprintln(stderr, "orangecount dialectize: -out is required")
		return 2
	}
	return writeConverted(fs.Arg(0), *out, "dialectize", stdout, stderr, true)
}

func writeConverted(entry, out, label string, stdout, stderr io.Writer, reverse bool, options ...snapshot.BuildOptions) int {
	config := sourceexport.Options{Reverse: reverse}
	if len(options) > 0 {
		config.Build = options[0]
	}
	plan, err := sourceexport.Prepare(entry, config)
	if err == nil {
		err = plan.Write(out)
	}
	if err != nil {
		fmt.Fprintf(stderr, "orangecount %s: %v\n", label, err)
		return 1
	}
	fmt.Fprintf(stdout, "orangecount %s: wrote %s\n", label, out)
	return 0
}

func renderBag(w io.Writer, bag *diagnostic.Bag, label string) {
	for _, d := range bag.All() {
		fmt.Fprintf(w, "orangecount %s: %s %s\n", label, d.Code, d.Message)
	}
}
