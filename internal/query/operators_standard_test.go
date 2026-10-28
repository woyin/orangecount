// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import "testing"

// TestStandardOperatorForms covers the BeanQuery operator surface that is
// exercised by the inventory probes: BETWEEN, IN/NOT IN, modulo, regular
// expression variants, string concatenation, and date arithmetic.
func TestStandardOperatorForms(t *testing.T) {
	evaluation := queryEvaluation(t)
	cases := []struct {
		query string
		kind  string
		value any
	}{
		{"SELECT 2 BETWEEN 1 AND 3 AS v FROM #entries LIMIT 1", "bool", true},
		{"SELECT 'abc' BETWEEN 'abc' AND 'abc' AS v FROM #entries LIMIT 1", "bool", true},
		{"SELECT 2 IN (1, 2) AS v FROM #entries LIMIT 1", "bool", true},
		{"SELECT 2 NOT IN (1, 2) AS v FROM #entries LIMIT 1", "bool", false},
		{"SELECT 'food' IN tags AS v FROM #transactions LIMIT 1", "bool", true},
		{"SELECT 'filename' IN meta AS v FROM #entries LIMIT 1", "bool", true},
		{"SELECT 2 % 2 AS v FROM #entries LIMIT 1", "integer", "0"},
		{"SELECT 'abc' + 'abc' AS v FROM #entries LIMIT 1", "string", "abcabc"},
		{"SELECT date('2020-01-02') + 2 AS v FROM #entries LIMIT 1", "date", "2020-01-04"},
		{"SELECT 2 + date('2020-01-02') AS v FROM #entries LIMIT 1", "date", "2020-01-04"},
		{"SELECT 'abc' !~ 'b' AS v FROM #entries LIMIT 1", "bool", false},
		{"SELECT 'abc' ?~ 'abc' AS v FROM #entries LIMIT 1", "bool", true},
	}
	for _, testCase := range cases {
		result, err := EvaluateTyped(testCase.query, evaluation)
		if err != nil {
			t.Fatalf("%s: %v", testCase.query, err)
		}
		if len(result.Rows) != 1 || result.Rows[0][0].Type != testCase.kind {
			t.Fatalf("%s: result=%+v", testCase.query, result)
		}
		if got := result.Rows[0][0].Value; got != testCase.value {
			t.Fatalf("%s: value=%v want %v", testCase.query, got, testCase.value)
		}
	}
	division, err := EvaluateTyped("SELECT 2 / 2 AS v FROM #entries LIMIT 1", evaluation)
	if err != nil || division.Rows[0][0].Type != "decimal" || division.Rows[0][0].Value.(PresentedDecimal).Exact != "1" {
		t.Fatalf("integer division result=%+v err=%v", division, err)
	}
}
