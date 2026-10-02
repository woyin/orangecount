// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop

import (
	"encoding/json"
	"orangecount/internal/query"
	"strings"
	"testing"
)

func TestRecursiveAnalysisCellValidation(t *testing.T) {
	for _, raw := range []string{
		`[{"key":"a","value":{"type":"null"}},{"key":"a","value":{"type":"null"}}]`,
		`[{"key":"z","value":{"type":"null"}},{"key":"a","value":{"type":"null"}}]`,
		`[{"key":"a","value":{"type":"null","value":""}}]`,
		`[{"key":"a","value":{"type":"bool","value":"true"}}]`,
		`[{"key":"a","value":{"type":"future","value":{}}}]`,
		`null`,
	} {
		if _, err := analysisCell("dictionary", json.RawMessage(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	for _, raw := range []string{`["a","a"]`, `["z","a"]`, `null`} {
		if _, err := analysisCell("set<string>", json.RawMessage(raw)); err == nil {
			t.Errorf("accepted set %s", raw)
		}
	}
	raw := `[{"key":"e\u0301","value":{"type":"integer","value":"1"}},{"key":"é","value":{"type":"integer","value":"2"}}]`
	value, err := analysisCell("dictionary", json.RawMessage(raw))
	if err != nil || len(value.(query.PresentedDictionary)) != 2 {
		t.Fatalf("Unicode keys lost: %v %v", value, err)
	}
	if _, err := analysisCell("open", json.RawMessage(`[]`)); err == nil {
		t.Fatal("missing record fields accepted")
	}
	deep := `{"type":"null"}`
	for range 35 {
		deep = `{"type":"list","value":[` + deep + `]}`
	}
	raw = `[{"key":"nested","value":` + deep + `}]`
	if _, err := analysisCell("dictionary", json.RawMessage(raw)); err == nil {
		t.Fatal("deep nesting accepted")
	}
	if _, err := analysisCell("dictionary", json.RawMessage(`[{"key":"`+strings.Repeat("a", 65537)+`","value":{"type":"null"}}]`)); err == nil {
		t.Fatal("oversized key accepted")
	}
}
