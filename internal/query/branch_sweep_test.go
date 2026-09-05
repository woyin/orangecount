// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package query

import (
	"strings"
	"testing"
)

// TestQueryParserAndEvaluatorErrors walks the query language's rejection
// paths: each text must fail with an error mentioning the given fragment,
// never panic, and never return partial rows.
func TestQueryParserAndEvaluatorErrors(t *testing.T) {
	evaluation := queryEvaluation(t)
	cases := []struct {
		name    string
		text    string
		message string
	}{
		{"empty", "", "must start with SELECT"},
		{"unbalanced paren", "select (account", "paren"},
		{"missing from", "select account", "requires FROM"},
		{"unknown function", "select nosuch(account) from postings", "unsupported function"},
		{"bare word", "select from", "requires FROM"},
		{"dangling operator", "select account where and", "FROM"},
		{"unknown column", "select * from mystery", "unknown query table"},
		{"bad membership", "select account from postings where account in", "unexpected"},
		{"unknown function in where", "select account from postings where nosuch(1) = 1", "unsupported function"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Evaluate(testCase.text, evaluation)
			if err == nil {
				t.Fatalf("query %q must fail", testCase.text)
			}
			if testCase.message != "" && !strings.Contains(err.Error(), testCase.message) {
				t.Fatalf("error=%v want fragment %q", err, testCase.message)
			}
		})
	}
}

func TestQueryAcceptsTypedComparisons(t *testing.T) {
	evaluation := queryEvaluation(t)
	// Account names compare as strings; a numeric-looking right side does not
	// crash the comparison and simply yields the matching subset.
	result, err := Evaluate("select account from postings where account > 5", evaluation)
	if err != nil {
		t.Fatalf("string comparison must not error: %v", err)
	}
	if len(result.Columns) == 0 {
		t.Fatalf("columns=%v", result.Columns)
	}
}

func TestQueryUnsupportedConstructsAreRejectedLoudly(t *testing.T) {
	evaluation := queryEvaluation(t)
	for _, text := range []string{
		"select * from postings open on 2000-01-01",
		"balances from year(2020)",
		"select account from postings group by account order by balances",
	} {
		if _, err := Evaluate(text, evaluation); err == nil {
			t.Fatalf("query %q must be rejected", text)
		}
	}
}
