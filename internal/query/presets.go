// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Preset uses tested BQL and fills an editable draft without executing it.
type Preset struct {
	ID     string `json:"id"`
	NameZH string `json:"name_zh"`
	NameEN string `json:"name_en"`
	Text   string `json:"text"`
}

//go:embed presets.json
var presetJSON []byte

// Presets returns an independent copy of the built-in catalogue.
func Presets() []Preset {
	var result []Preset
	if err := json.Unmarshal(presetJSON, &result); err != nil {
		panic("invalid built-in query presets")
	}
	return result
}

// PresetsForPeriod renders validated inclusive bounds into the built-in BQL.
// It only handles our fixed templates; it never rewrites an editor draft.
func PresetsForPeriod(from, to string) ([]Preset, error) {
	if from == "" && to == "" {
		return nil, fmt.Errorf("period requires an explicit date bound")
	}
	conditions := []string{}
	for i, date := range []string{from, to} {
		if date == "" {
			continue
		}
		parsed, err := time.Parse("2006-01-02", date)
		if err != nil || parsed.Format("2006-01-02") != date {
			return nil, fmt.Errorf("invalid preset period")
		}
		operator := ">="
		if i == 1 {
			operator = "<="
		}
		conditions = append(conditions, "date "+operator+" "+date)
	}
	if from != "" && to != "" && from > to {
		return nil, fmt.Errorf("invalid preset period")
	}
	presets := Presets()
	for i, preset := range presets {
		boundary := len(preset.Text)
		for _, clause := range []string{" GROUP BY ", " ORDER BY "} {
			if index := strings.Index(preset.Text, clause); index >= 0 && index < boundary {
				boundary = index
			}
		}
		head, tail := preset.Text[:boundary], preset.Text[boundary:]
		conjunction := " WHERE "
		if strings.Contains(head, " WHERE ") {
			conjunction = " AND "
		}
		presets[i].Text = head + conjunction + strings.Join(conditions, " AND ") + tail
	}
	return presets, nil
}
