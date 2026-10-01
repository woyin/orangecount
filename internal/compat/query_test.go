// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package compat_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orangecount/internal/compat"
)

func TestQueryOracleGateChecksTypedAggregateEndpoints(t *testing.T) {
	result, err := compat.CheckQueryGolden("../..", "../../testdata/golden/query-v1/aggregates.json")
	if err != nil || result.Checked != 11 || result.Failed != 0 {
		t.Fatalf("report=%+v error=%v", result, err)
	}
}

func TestQueryOracleGateChecksCompoundCasts(t *testing.T) {
	result, err := compat.CheckQueryGolden("../..", "../../testdata/golden/query-v1/compound-casts.json")
	if err != nil || result.Checked == 0 || result.Failed != 0 {
		t.Fatalf("report=%+v error=%v", result, err)
	}
}

func TestQueryOracleGateChecksExplicitCasts(t *testing.T) {
	result, err := compat.CheckQueryGolden("../..", "../../testdata/golden/query-v1/casts.json")
	if err != nil || result.Checked == 0 || result.Failed != 0 {
		t.Fatalf("report=%+v error=%v", result, err)
	}
}

func TestQueryOracleGateChecksDynamicValues(t *testing.T) {
	result, err := compat.CheckQueryGolden("../..", "../../testdata/golden/query-v1/dynamic-values.json")
	if err != nil || result.Checked == 0 || result.Failed != 0 {
		t.Fatalf("report=%+v error=%v", result, err)
	}
}

func TestQueryOracleGateChecksReviewedCases(t *testing.T) {
	result, err := compat.CheckQueryGolden("../..", "../../testdata/golden/query-v1/default-source.json")
	if err != nil || result.Checked < 2 || result.Failed != 0 {
		t.Fatalf("report=%+v error=%v", result, err)
	}
}

func TestQueryOracleGateChecksCompoundValues(t *testing.T) {
	result, err := compat.CheckQueryGolden("../..", "../../testdata/golden/query-v1/compound-values.json")
	if err != nil || result.Checked == 0 || result.Failed != 0 {
		t.Fatalf("report=%+v error=%v", result, err)
	}
}

func TestQueryOracleGateChecksNormalizedQuoteWeights(t *testing.T) {
	result, err := compat.CheckQueryGolden("../..", "../../testdata/golden/query-v1/price-weight.json")
	if err != nil || result.Checked == 0 || result.Failed != 0 {
		t.Fatalf("report=%+v error=%v", result, err)
	}
}

func TestQueryOracleGateChecksInventorySortingTies(t *testing.T) {
	result, err := compat.CheckQueryGolden("../..", "../../testdata/golden/query-v1/inventory-order.json")
	if err != nil || result.Checked == 0 || result.Failed != 0 {
		t.Fatalf("report=%+v error=%v", result, err)
	}
}

func TestQueryOracleGateChecksMetadataSourcesAndNullPrecedence(t *testing.T) {
	result, err := compat.CheckQueryGolden("../..", "../../testdata/golden/query-v1/metadata-source.json")
	if err != nil || result.Checked == 0 || result.Failed != 0 {
		t.Fatalf("report=%+v error=%v", result, err)
	}
}

func TestQueryOracleGateChecksLexicalMetadataStack(t *testing.T) {
	result, err := compat.CheckQueryGolden("../..", "../../testdata/golden/query-v1/metadata-stack.json")
	if err != nil || result.Checked == 0 || result.Failed != 0 {
		t.Fatalf("report=%+v error=%v", result, err)
	}
}

func TestQueryOracleGateChecksGeneratedMetadata(t *testing.T) {
	result, err := compat.CheckQueryGolden("../..", "../../testdata/golden/query-v1/metadata-pad.json")
	if err != nil || result.Checked == 0 || result.Failed != 0 {
		t.Fatalf("report=%+v error=%v", result, err)
	}
}

func TestQueryOracleGateDetectsChangedIncludedBytes(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"testdata/fixtures/query/metadata-stack.bean", "testdata/fixtures/query/metadata-stack-child.bean", "testdata/fixtures/query/metadata-stack.queries.json", "testdata/golden/query-v1/metadata-stack.json"} {
		data, err := os.ReadFile(filepath.Join("../..", name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, "metadata-stack-child.bean") {
			data = append(data, []byte("; changed bytes with identical accounting values\n")...)
		}
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, err := compat.CheckQueryGolden(root, filepath.Join(root, "testdata/golden/query-v1/metadata-stack.json"))
	if err == nil || !strings.Contains(err.Error(), "source graph identity mismatch") {
		t.Fatalf("changed include must invalidate reviewed input identity: %v", err)
	}
}

func TestQueryOracleGateDetectsChangedCostIdentity(t *testing.T) {
	data, err := os.ReadFile("../../testdata/golden/query-v1/compound-values.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"number", "currency", "date", "label"} {
		t.Run(field, func(t *testing.T) {
			var golden map[string]any
			if err := json.Unmarshal(data, &golden); err != nil {
				t.Fatal(err)
			}
			rows := golden["cases"].([]any)[0].(map[string]any)["result"].(map[string]any)["rows"].([]any)
			cost := rows[0].([]any)[1].(map[string]any)["value"].(map[string]any)["cost"].(map[string]any)
			cost[field] = "changed"
			changed, err := json.Marshal(golden)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "changed.json")
			if err := os.WriteFile(path, changed, 0600); err != nil {
				t.Fatal(err)
			}
			report, err := compat.CheckQueryGolden("../..", path)
			if err != nil || report.Failed != 1 || report.Failures[0].ID != "position-full-cost" {
				t.Fatalf("cost identity corruption escaped: %+v %v", report, err)
			}
		})
	}
}

func TestQueryOracleCorpusRejectsMissingRegisteredGolden(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"testdata/fixtures/query/default-source.bean", "testdata/fixtures/query/default-source.queries.json", "testdata/golden/query-v1/default-source.json"} {
		data, err := os.ReadFile(filepath.Join("../..", name))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := compat.CheckQueryCorpus(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing registered corpus must fail: %v", err)
	}
}

func TestQueryOracleGateDetectsChangedExpectedCell(t *testing.T) {
	data, err := os.ReadFile("../../testdata/golden/query-v1/default-source.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden map[string]any
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	cases := golden["cases"].([]any)
	result := cases[0].(map[string]any)["result"].(map[string]any)
	rows := result["rows"].([]any)
	rows[0].([]any)[1].(map[string]any)["value"] = "123456789"
	changed, err := json.Marshal(golden)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "changed.json")
	if err := os.WriteFile(path, changed, 0600); err != nil {
		t.Fatal(err)
	}
	report, err := compat.CheckQueryGolden("../..", path)
	if err != nil || report.Failed != 1 || len(report.Failures) != 1 || report.Failures[0].ID != "default-postings" {
		t.Fatalf("corruption escaped gate: report=%+v err=%v", report, err)
	}
}

func TestQueryOracleGateRejectsTrailingJSONDocument(t *testing.T) {
	data, err := os.ReadFile("../../testdata/golden/query-v1/default-source.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "trailing.json")
	if err := os.WriteFile(path, append(data, []byte("{}")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := compat.CheckQueryGolden("../..", path); err == nil {
		t.Fatal("trailing document silently accepted")
	}
}
