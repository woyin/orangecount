// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
// sourceprobe observes loading only; accounting and private inputs are not used
// by its public reference acceptance harness.
package main

import (
	"encoding/json"
	"flag"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
	"os"
)

func main() {
	entry := flag.String("entry", "", "public source entry")
	evaluate := flag.Bool("evaluate", false, "observe standard transaction order")
	flag.Parse()
	graph, err := source.LoadGraph(*entry)
	if err != nil {
		os.Exit(1)
	}
	type edge struct{ Literal, Path string }
	result := struct {
		Edges      []edge   `json:"edges"`
		Errors     []string `json:"errors"`
		Narrations []string `json:"narrations,omitempty"`
	}{Edges: []edge{}, Errors: []string{}}
	for _, item := range graph.Edges[graph.Entry] {
		result.Edges = append(result.Edges, edge{item.Literal, graph.DisplayPath(item.To)})
	}
	for _, issue := range graph.Diagnostics {
		result.Errors = append(result.Errors, issue.Code)
	}
	if *evaluate {
		loaded := snapshot.BuildWithOptions(*entry, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
		if loaded.Snapshot == nil {
			for _, d := range loaded.Diagnostics {
				result.Errors = append(result.Errors, d.Code)
			}
		} else {
			for _, record := range loaded.Snapshot.Evaluation().Entries {
				switch txn := record.Directive.(type) {
				case *ledger.Transaction:
					result.Narrations = append(result.Narrations, txn.Narration)
				case ledger.Transaction:
					result.Narrations = append(result.Narrations, txn.Narration)
				}
			}
		}
	}
	if err = json.NewEncoder(os.Stdout).Encode(result); err != nil {
		os.Exit(1)
	}
}
