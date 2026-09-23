// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Command parity emits redacted differential records between OrangeCount and
// the committed Beancount v3 oracle snapshots (ADR-0008). Its JSON output is
// the input for tools/reference/jev_triage.py; it never carries private
// ledger content, only sanitized corpus differences.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"orangecount/internal/compat"
)

type report struct {
	FixtureCount int                 `json:"fixture_count"`
	CleanCount   int                 `json:"clean_count"`
	Records      []compat.DiffRecord `json:"records"`
}

func main() {
	fixturesDir := flag.String("fixtures", "testdata/fixtures/v3-parity", "fixture directory")
	goldenDir := flag.String("golden", "testdata/golden/v3-parity", "golden snapshot directory")
	out := flag.String("out", "", "write the report JSON here instead of stdout")
	flag.Parse()

	paths, err := filepath.Glob(filepath.Join(*fixturesDir, "*.bean"))
	if err != nil || len(paths) == 0 {
		fail("no fixtures under %s", *fixturesDir)
	}
	sort.Strings(paths)

	result := report{Records: []compat.DiffRecord{}}
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".bean")
		golden, err := compat.LoadGolden(filepath.Join(*goldenDir, name+".golden.json"))
		if err != nil {
			result.Records = append(result.Records, compat.DiffRecord{
				Fixture: name, Dimension: "harness", Kind: "golden-missing",
				Detail: map[string]any{"error": "run tools/reference/generate_golden.py"},
			})
			continue
		}
		oc, _, err := compat.BuildOCSide(path)
		if err != nil {
			fail("fixture %s: %v", name, err)
		}
		records := compat.DiffFixture(golden.Fixture, golden, oc)
		unregistered := 0
		for _, r := range records {
			if r.Boundary == "" {
				unregistered++
			}
		}
		if unregistered == 0 {
			result.CleanCount++
		}
		result.Records = append(result.Records, records...)
	}
	result.FixtureCount = len(paths)

	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fail("encode: %v", err)
	}
	if *out == "" {
		os.Stdout.Write(append(encoded, '\n'))
		return
	}
	if err := os.WriteFile(*out, append(encoded, '\n'), 0o644); err != nil {
		fail("write %s: %v", *out, err)
	}
	fmt.Printf("parity: %d fixtures, %d clean, %d difference records -> %s\n",
		result.FixtureCount, result.CleanCount, len(result.Records), *out)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "parity: "+format+"\n", args...)
	os.Exit(2)
}
