// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package query_test

import (
	"encoding/json"
	"orangecount/internal/query"
	"os"
	"strings"
	"testing"
)

func TestPresetsMatchReviewedOracleInputsAndDoNotShareStorage(t *testing.T) {
	data, err := os.ReadFile("../../testdata/fixtures/query/presets.queries.json")
	if err != nil {
		t.Fatal(err)
	}
	var oracle []struct {
		ID    string `json:"id"`
		Query string `json:"query"`
	}
	if err := json.Unmarshal(data, &oracle); err != nil {
		t.Fatal(err)
	}
	presets := query.Presets()
	if len(presets) != 4 || len(oracle) != 16 {
		t.Fatal("preset coverage differs")
	}
	for i, preset := range presets {
		if preset.ID != oracle[i].ID || preset.Text != oracle[i].Query || preset.NameZH == "" || preset.NameEN == "" {
			t.Fatal("preset differs from reviewed executable oracle input")
		}
	}
	for n, bounds := range [][2]string{{"2000-02-01", "2000-02-21"}, {"2000-02-01", ""}, {"", "2000-01-31"}} {
		generated, err := query.PresetsForPeriod(bounds[0], bounds[1])
		if err != nil {
			t.Fatal(err)
		}
		for i, preset := range generated {
			if preset.Text != oracle[4+n*4+i].Query {
				t.Fatal("period text differs from oracle input")
			}
		}
	}
	presets[0].Text = "changed"
	if query.Presets()[0].Text == "changed" {
		t.Fatal("shared catalogue changed")
	}
}

func TestPeriodPresetsValidateAndExposeInclusiveDateBounds(t *testing.T) {
	for _, bounds := range [][2]string{{"2000-02-01", "2000-02-21"}, {"2000-02-01", ""}, {"", "2000-01-31"}} {
		presets, err := query.PresetsForPeriod(bounds[0], bounds[1])
		if err != nil {
			t.Fatal(err)
		}
		if len(presets) != 4 {
			t.Fatal("period presets missing")
		}
		for _, preset := range presets {
			if bounds[0] != "" && !strings.Contains(preset.Text, "date >= "+bounds[0]) {
				t.Fatal("lower bound missing")
			}
			if bounds[1] != "" && !strings.Contains(preset.Text, "date <= "+bounds[1]) {
				t.Fatal("upper bound missing")
			}
		}
	}
	for _, bounds := range [][2]string{{"2000-02-30", ""}, {"2000-02-01 OR TRUE", ""}, {"2000-03-01", "2000-02-01"}, {"", ""}} {
		if _, err := query.PresetsForPeriod(bounds[0], bounds[1]); err == nil {
			t.Fatal("invalid or empty period accepted")
		}
	}
}
