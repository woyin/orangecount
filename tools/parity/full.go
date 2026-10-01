// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"encoding/json"
	"orangecount/internal/compat"
	"os"
	"path/filepath"
	"sort"
)

type fullRecord struct {
	Fixture    string   `json:"fixture"`
	Dimensions []string `json:"dimensions"`
}

type fullReport struct {
	SchemaVersion     int          `json:"schema_version"`
	FixtureCount      int          `json:"fixture_count"`
	CleanCount        int          `json:"clean_count"`
	FailedCount       int          `json:"failed_count"`
	ExtensionFixtures []string     `json:"extension_fixtures"`
	Records           []fullRecord `json:"records"`
}

func buildFullReport(directories []string, goldenDir string) fullReport {
	result := fullReport{SchemaVersion: compat.FullSchemaVersion, ExtensionFixtures: []string{}, Records: []fullRecord{}}
	paths := []string{}
	for _, dir := range directories {
		found, err := filepath.Glob(filepath.Join(dir, "*.bean"))
		if err != nil || len(found) == 0 {
			result.FailedCount++
			result.Records = append(result.Records, fullRecord{filepath.Base(dir), []string{"fixtures-missing"}})
		}
		paths = append(paths, found...)
	}
	sort.Strings(paths)
	for _, path := range paths {
		name := filepath.Base(path)
		if name == "average-explicit.bean" || name == "global-average.bean" {
			result.ExtensionFixtures = append(result.ExtensionFixtures, name)
			continue
		}
		result.FixtureCount++
		var golden compat.FullGolden
		encoded, err := os.ReadFile(filepath.Join(goldenDir, name[:len(name)-5]+".golden.json"))
		dimensions := []string{}
		if err != nil || json.Unmarshal(encoded, &golden) != nil {
			dimensions = append(dimensions, "golden-missing-or-invalid")
		} else {
			if golden.OracleEngine != "beancount-python-3.2.3" || golden.Fixture != name {
				dimensions = append(dimensions, "oracle-identity")
			}
			have, err := compat.BuildFullSide(path)
			if err != nil {
				dimensions = append(dimensions, "build-failed")
			} else {
				dimensions = append(dimensions, compat.FullGoldenDifferences(golden, have)...)
			}
		}
		if len(dimensions) == 0 {
			result.CleanCount++
		} else {
			result.FailedCount++
			result.Records = append(result.Records, fullRecord{name, dimensions})
		}
	}
	return result
}

func writeReport(value any, path string) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fail("encode report")
	}
	encoded = append(encoded, '\n')
	if path == "" {
		_, _ = os.Stdout.Write(encoded)
	} else if err := os.WriteFile(path, encoded, 0600); err != nil {
		fail("write report")
	}
}
