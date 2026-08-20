// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"orangecount/internal/diagnostic"
	"orangecount/internal/dialect"
	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

func runExport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "output file (single-file ledger) or directory (include graph)")
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
	return writeConverted(fs.Arg(0), *out, "export", stdout, stderr, false)
}

func runDialectize(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dialectize", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "output file (single-file ledger) or directory (include graph)")
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

func writeConverted(entry, out, label string, stdout, stderr io.Writer, reverse bool) int {
	graph, err := source.LoadGraph(entry)
	if err != nil {
		fmt.Fprintf(stderr, "orangecount %s: %v\n", label, err)
		return 1
	}
	parsed, bag := ledger.ParseGraph(graph)
	if bag.HasErrors() {
		renderBag(stderr, bag, label)
		return 1
	}
	outputs := make(map[source.FileID][]byte)
	if reverse {
		for fileID, file := range parsed {
			if file == nil || file.Source == nil {
				continue
			}
			edits, _ := dialect.Dialectize(file)
			outputs[fileID] = dialect.ApplyEdits(file.Source.Data, edits)
		}
	} else {
		rendered, diags := dialect.ExportText(graph, parsed)
		for _, d := range diags {
			if d.Severity == diagnostic.Error {
				fmt.Fprintf(stderr, "orangecount %s: %s %s\n", label, d.Code, d.Message)
				return 1
			}
		}
		outputs = rendered
	}
	return writeConvertedFiles(graph, outputs, out, label, stdout, stderr)
}

func writeConvertedFiles(graph *source.Graph, outputs map[source.FileID][]byte, out, label string, stdout, stderr io.Writer) int {
	fail := func(err error) int { fmt.Fprintf(stderr, "orangecount %s: %v\n", label, err); return 1 }
	if len(graph.Order) == 1 {
		if dir := filepath.Dir(out); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fail(err)
			}
		}
		if err := os.WriteFile(out, outputs[graph.Entry], 0o600); err != nil {
			return fail(err)
		}
		fmt.Fprintf(stdout, "orangecount %s: wrote %s\n", label, out)
		return 0
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fail(err)
	}
	entryDir := filepath.Dir(graph.Path(graph.Entry))
	for fileID, data := range outputs {
		rel, err := filepath.Rel(entryDir, graph.Path(fileID))
		if err != nil {
			return fail(err)
		}
		target := filepath.Join(out, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			return fail(err)
		}
	}
	fmt.Fprintf(stdout, "orangecount %s: wrote %d files under %s\n", label, len(outputs), out)
	return 0
}

func renderBag(w io.Writer, bag *diagnostic.Bag, label string) {
	for _, d := range bag.All() {
		fmt.Fprintf(w, "orangecount %s: %s %s\n", label, d.Code, d.Message)
	}
}
