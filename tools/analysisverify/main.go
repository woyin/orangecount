// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

// analysisverify validates a captured query without opening its source ledger.
package main

import (
	"encoding/json"
	"fmt"
	"orangecount/internal/desktop"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: analysisverify FILE.ocanalysis")
		os.Exit(2)
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot open analysis")
		os.Exit(1)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot inspect analysis")
		os.Exit(1)
	}
	document, err := desktop.ReadQueryAnalysis(file, info.Size())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	summary := struct {
		Version   int    `json:"format_version"`
		View      string `json:"view"`
		Rows      int    `json:"rows"`
		Columns   int    `json:"columns"`
		Truncated bool   `json:"truncated"`
	}{document.Context.FormatVersion, document.Context.View, len(document.Result.Rows), len(document.Result.Columns), document.Context.Truncated}
	if json.NewEncoder(os.Stdout).Encode(summary) != nil {
		os.Exit(1)
	}
}
