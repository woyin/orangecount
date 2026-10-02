// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"testing"

	"orangecount/internal/query"
)

func TestRunQueryTypedCSVPreservesExactValuesAndDuplicateHeaders(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQuery([]string{"--format", "typed-csv", "--semantics", "v3", "../../testdata/fixtures/query/default-source.bean", "SELECT 1 AS v, 2 AS v, decimal('12345678901234567890.0001') AS precise, 'a,b' AS text LIMIT 1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	rows, err := csv.NewReader(&stdout).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0][0] != "v" || rows[0][1] != "v" || rows[1][0] != "1" || rows[1][1] != "2" || rows[1][2] != "12345678901234567890.0001" || rows[1][3] != "a,b" {
		t.Fatalf("invalid CSV: %+v", rows)
	}
}

func TestRunQueryTypedJSONPreservesColumnPositions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQuery([]string{"--format", "typed-json", "--semantics", "v3", "../../testdata/fixtures/query/default-source.bean", "SELECT 1 AS v, 2 AS v, number = 0 AS zero, '' AS blank LIMIT 1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	var result query.TypedResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Contract != query.CurrentContract || len(result.Columns) != 4 || result.Columns[0].ID == result.Columns[1].ID || result.Rows[0][0].Value != "1" || result.Rows[0][1].Value != "2" || result.Rows[0][2].Value != false || result.Rows[0][3].Value != "" {
		t.Fatalf("result=%s", stdout.String())
	}
}

func TestRunQueryBuiltInPresets(t *testing.T) {
	for _, preset := range query.Presets() {
		t.Run(preset.ID, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runQuery([]string{"--format", "typed-json", "--semantics", "v3", "../../testdata/fixtures/query/presets.bean", preset.Text}, &stdout, &stderr); code != 0 {
				t.Fatalf("exit=%d: %s", code, stderr.String())
			}
			var result query.TypedResult
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if len(result.Rows) == 0 || len(result.Columns) == 0 {
				t.Fatal("preset returned no rows or schema")
			}
			if preset.ID == "monthly-income" || preset.ID == "monthly-expenses" {
				if len(result.Rows) != 2 || result.Rows[0][0].Value != "2000" || result.Rows[0][1].Value != "1" || result.Rows[1][1].Value != "2" {
					t.Fatal("monthly grouping failed")
				}
			}
		})
	}
}

func TestRunQueryExplicitPeriodPresets(t *testing.T) {
	for _, bounds := range [][2]string{{"2000-02-01", "2000-02-21"}, {"2000-02-01", ""}, {"", "2000-01-31"}} {
		presets, err := query.PresetsForPeriod(bounds[0], bounds[1])
		if err != nil {
			t.Fatal(err)
		}
		for _, preset := range presets {
			var stdout, stderr bytes.Buffer
			if code := runQuery([]string{"--format", "typed-json", "--semantics", "v3", "../../testdata/fixtures/query/presets.bean", preset.Text}, &stdout, &stderr); code != 0 {
				t.Fatalf("exit=%d: %s", code, stderr.String())
			}
			var result query.TypedResult
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if preset.ID == "monthly-income" || preset.ID == "monthly-expenses" {
				month := "2"
				if bounds[0] == "" {
					month = "1"
				}
				if len(result.Rows) != 1 || result.Rows[0][1].Value != month {
					t.Fatal("inclusive monthly period failed")
				}
			}
		}
	}
}
