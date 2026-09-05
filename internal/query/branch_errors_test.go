// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package query

import (
	"strings"
	"testing"
)

// TestQueryFunctionAndSyntaxErrorMessages walks the engine's rejection
// branches so every operator and builtin reports a specific, bounded error.
func TestQueryFunctionAndSyntaxErrorMessages(t *testing.T) {
	evaluation := queryEvaluation(t)
	cases := []struct {
		text    string
		message string
	}{
		{"select root(account) from postings", "root requires two arguments"},
		{"select root(number, 2) from postings", "root requires an account column"},
		{"select root(account, 1.5) from postings", "root requires an integer depth"},
		{"select root(account, account) from postings", "numeric depth"},
		{"select account from postings where account ~ 5", "string pattern"},
		{"select account from postings limit x", "LIMIT requires a number"},
		{"select sum(number number) from postings", "commas"},
		{"select account from postings where number % 2", "unexpected"},
		{"select account from postings where", "unknown column"},
		{"select account, nosuch from postings", "unknown column"},
		{"select account from postings where date ~ 2020", "string pattern"},
	}
	for _, testCase := range cases {
		t.Run(testCase.text, func(t *testing.T) {
			_, err := Evaluate(testCase.text, evaluation)
			if err == nil {
				t.Fatalf("query %q must fail", testCase.text)
			}
			if !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("err=%v want %q", err, testCase.message)
			}
		})
	}
}

// TestQueryArithmeticAndRootQueries pins the valid forms of the operators the
// rejection tests bound: subtraction and root account grouping.
func TestQueryArithmeticAndRootQueries(t *testing.T) {
	evaluation := queryEvaluation(t)
	result, err := Evaluate("select root(account, 1) as top, sum(number) from postings group by top", evaluation)
	if err != nil {
		t.Fatalf("root query err=%v", err)
	}
	if len(result.Columns) == 0 || len(result.Rows) == 0 {
		t.Fatalf("result=%+v", result)
	}
	if _, err := Evaluate("select number - 1 from postings", evaluation); err != nil {
		t.Fatalf("subtraction err=%v", err)
	}
	// root(account, n<=0) is defined as the empty string, not an error.
	empty, err := Evaluate("select root(account, -2) as r from postings", evaluation)
	if err != nil || len(empty.Rows) == 0 {
		t.Fatalf("negative root err=%v rows=%d", err, len(empty.Rows))
	}
	for _, row := range empty.Rows {
		if row["r"] != "" {
			t.Fatalf("negative root must yield empty string, got %v", row["r"])
		}
	}
}
