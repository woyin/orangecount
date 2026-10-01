// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

// Command statecapture writes a sensitive, short-lived verification capture
// outside the repository. It never prints ledger content or source paths.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"orangecount/internal/compat"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
	"os"
	"path/filepath"
	"strings"
)

type capture struct {
	InputIdentity string                                 `json:"input_identity"`
	Entry         string                                 `json:"entry"`
	Files         []*source.SourceFile                   `json:"files"`
	Side          compat.FullGolden                      `json:"side"`
	Periods       map[string]query.Result                `json:"periods"`
	Edges         map[source.FileID][]source.IncludeEdge `json:"edges"`
}

func main() {
	path := flag.String("ledger", "", "ledger input")
	out := flag.String("out", "", "sensitive temporary capture outside repository")
	flag.Parse()
	if err := run(*path, *out); err != nil {
		fmt.Fprintln(os.Stderr, "statecapture: unable to capture a stable ledger input")
		os.Exit(1)
	}
}

func run(path, out string) error {
	root, err := filepath.Abs(".")
	if err != nil {
		return err
	}
	target, err := filepath.Abs(out)
	if err != nil || out == "" || target == root || strings.HasPrefix(target, root+string(os.PathSeparator)) {
		return fmt.Errorf("capture must be outside repository")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(target))
	if err != nil {
		return err
	}
	if parent == root || strings.HasPrefix(parent, root+string(os.PathSeparator)) {
		return fmt.Errorf("capture parent is inside repository")
	}
	graph, err := source.LoadGraph(path)
	if err != nil {
		return err
	}
	result := snapshot.BuildGraphWithOptions(graph, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	value := capture{Entry: graph.Path(graph.Entry), InputIdentity: snapshot.InputIdentity(graph), Side: compat.FullSideFromBuild(path, result), Periods: map[string]query.Result{}, Edges: graph.Edges}
	for _, id := range graph.Order {
		value.Files = append(value.Files, graph.File(id))
	}
	if result.Snapshot != nil {
		e := result.Snapshot.Evaluation()
		for _, account := range []string{"Expenses", "Income"} {
			sql := "SELECT year(date) AS year, month(date) AS month, currency, sum(number) AS total FROM postings WHERE account ~ '^" + account + "' GROUP BY year(date), month(date), currency ORDER BY year(date), month(date), currency"
			rows, err := query.Evaluate(sql, e)
			if err != nil {
				return err
			}
			value.Periods[account] = rows
		}
	}
	if graph.ChangedOnDisk() {
		return fmt.Errorf("input changed")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(encoded)
	return err
}
