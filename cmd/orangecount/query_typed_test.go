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

// TestRunQueryStandardForms covers the four standard BQL forms end to end
// through the packaged CLI. Expected values are the captured BeanQuery 0.2.0
// observations for testdata/fixtures/query/standard-forms.bean, mapped into
// the typed contract this CLI exposes.
func TestRunQueryStandardForms(t *testing.T) {
	for _, test := range []struct {
		name    string
		text    string
		columns []string
		rows    int
		extra   func(t *testing.T, result query.TypedResult)
	}{
		{
			name:    "distinct",
			text:    "SELECT DISTINCT account ORDER BY account",
			columns: []string{"account"},
			rows:    3,
			extra: func(t *testing.T, result query.TypedResult) {
				if result.Rows[0][0].Value != "Assets:Cash" || result.Rows[2][0].Value != "Expenses:Food" {
					t.Fatalf("distinct rows=%+v", result.Rows)
				}
			},
		},
		{
			name:    "having",
			text:    "SELECT account, sum(number) AS total GROUP BY account HAVING sum(number) > 0 ORDER BY account",
			columns: []string{"account", "total"},
			rows:    2,
			extra: func(t *testing.T, result query.TypedResult) {
				if decimalExact(t, result.Rows[0][1]) != "90" {
					t.Fatalf("having rows=%+v", result.Rows)
				}
			},
		},
		{
			name:    "subquery",
			text:    "SELECT account FROM (SELECT account WHERE number > 0) ORDER BY account",
			columns: []string{"account"},
			rows:    2,
			extra: func(t *testing.T, result query.TypedResult) {
				if result.Rows[0][0].Value != "Assets:Cash" || result.Rows[1][0].Value != "Expenses:Food" {
					t.Fatalf("subquery rows=%+v", result.Rows)
				}
			},
		},
		{
			name:    "pivot",
			text:    "SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY 1, 2",
			columns: []string{"account/currency", "USD"},
			rows:    3,
			extra: func(t *testing.T, result query.TypedResult) {
				if !result.Columns[1].Nullable || decimalExact(t, result.Rows[0][1]) != "90" {
					t.Fatalf("pivot result=%+v", result)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runQuery([]string{"--format", "typed-json", "--semantics", "v3", "../../testdata/fixtures/query/standard-forms.bean", test.text}, &stdout, &stderr); code != 0 {
				t.Fatalf("exit=%d stderr=%s", code, stderr.String())
			}
			var result query.TypedResult
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Contract != query.CurrentContract || len(result.Rows) != test.rows || len(result.Columns) != len(test.columns) {
				t.Fatalf("result=%+v", result)
			}
			for i, name := range test.columns {
				if result.Columns[i].Name != name {
					t.Fatalf("column %d=%s want %s", i, result.Columns[i].Name, name)
				}
			}
			test.extra(t, result)
		})
	}
}

// decimalExact reads the exact spelling of a decimal cell as it arrives from
// the CLI wire format, where the value decodes to a generic map.
func decimalExact(t *testing.T, cell query.Cell) string {
	t.Helper()
	if cell.Type != "decimal" {
		t.Fatalf("cell %+v is not decimal", cell)
	}
	value, ok := cell.Value.(map[string]any)
	if !ok {
		t.Fatalf("decimal cell %+v has unexpected wire shape", cell)
	}
	exact, _ := value["exact"].(string)
	return exact
}
