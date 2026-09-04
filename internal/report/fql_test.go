// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package report

import (
	"math/big"
	"strings"
	"testing"

	"orangecount/internal/query"
)

func fqlTestTarget() FQLTarget {
	return FQLTarget{
		Tags:      []string{"trip", "food"},
		Links:     []string{"invoice-1"},
		Payee:     "Edeka",
		Narration: "Groceries and coffee",
		Account:   "Expenses:Food",
		Flag:      "*",
		Date:      "2026-01-05",
		Metadata:  map[string]string{"purpose": "office snacks"},
		Postings: []FQLPosting{
			{Account: "Expenses:Food", Units: big.NewFloat(12.5)},
			{Account: "Assets:Cash", Units: big.NewFloat(-25)},
		},
	}
}

func TestFQLMatchesTagsLinksAndStrings(t *testing.T) {
	target := fqlTestTarget()
	cases := []struct {
		text string
		want bool
	}{
		{"#trip", true},
		{"#trips", false}, // exact membership, not substring
		{"^invoice-1", true},
		{"^invoice", false},
		{"coffee", true},      // case-insensitive regex over narration/payee/comment
		{"COFFEE", true},      // the pattern itself is case-insensitive too
		{`"Edeka"`, true},     // quoted string still matches payee
		{"Rent", false},       // no field carries it
		{"payee:edeka", true}, // key term matches its field
		{"payee:rewe", false},
		{"purpose:snacks", true}, // metadata keys are reachable
		{`purpose:"^office"`, true},
		{"missing:value", false},
		{"account:Expenses", true},
		{"flag:*", false}, // "*" is not a valid lexeme for a value: parse error below
	}
	for _, testCase := range cases {
		filter, err := ParseFQL(testCase.text)
		if testCase.text == "flag:*" {
			if err == nil {
				t.Fatalf("flag:* should fail to parse")
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseFQL(%q) error: %v", testCase.text, err)
		}
		if got := filter.Match(target); got != testCase.want {
			t.Fatalf("Match(%q)=%v want=%v", testCase.text, got, testCase.want)
		}
	}
}

func TestFQLBooleanCombinators(t *testing.T) {
	target := fqlTestTarget()
	cases := []struct {
		text string
		want bool
	}{
		{"#trip coffee", true}, // juxtaposition is AND
		{"#trip rent", false},
		{"#trip, rent", true},   // comma is OR
		{"rent, -#trip", false}, // negation
		{"-rent", true},
		{"(#trip, #nope) coffee", true},  // parentheses group
		{"any(account:Assets)", true},    // one posting qualifies
		{"all(account:Expenses)", false}, // the cash posting does not
		{"any(account:Liabilities)", false},
		{">20", true}, // posting units magnitude
		{">100", false},
		{"=12.5", true},
		{"number=5", false}, // no entry-level amount here
		{"#trip and", true}, // "and" lexes as a plain string, narration contains it
	}
	for _, testCase := range cases {
		filter, err := ParseFQL(testCase.text)
		if err != nil {
			t.Fatalf("ParseFQL(%q) error: %v", testCase.text, err)
		}
		if got := filter.Match(target); got != testCase.want {
			t.Fatalf("Match(%q)=%v want=%v", testCase.text, got, testCase.want)
		}
	}
}

func TestFQLParseErrors(t *testing.T) {
	cases := []struct {
		text    string
		message string
	}{
		{"payee:", "failed to parse filter"},
		{"#trip &", `illegal character "&" in filter`},
		{")", "failed to parse filter"},
		{"(#trip", "failed to parse filter"},
		{">", "failed to parse filter"},
	}
	for _, testCase := range cases {
		_, err := ParseFQL(testCase.text)
		if err == nil {
			t.Fatalf("ParseFQL(%q) should fail", testCase.text)
		}
		if !strings.Contains(err.Error(), testCase.message) {
			t.Fatalf("ParseFQL(%q)=%q want substring %q", testCase.text, err.Error(), testCase.message)
		}
	}
	if filter, err := ParseFQL("   "); err != nil || filter != nil {
		t.Fatalf("blank filter should be nil with no error, got %v %v", filter, err)
	}
}

func TestFQLNumberComparisonOnBalanceAmount(t *testing.T) {
	target := FQLTarget{Amount: big.NewFloat(100)}
	filter, err := ParseFQL("number>=100")
	if err != nil {
		t.Fatalf("ParseFQL error: %v", err)
	}
	if !filter.Match(target) {
		t.Fatalf("number>=100 should match amount 100")
	}
	if filter.Match(FQLTarget{}) {
		t.Fatalf("number>=100 should not match an entry without an amount")
	}
}

func TestFQLTargetFromRowUsesColumns(t *testing.T) {
	row := query.Row{"date": "2026-01-05", "account": "Expenses:Food", "narration": "Coffee run", "change": "1.00"}
	target := FQLTargetFromRow(row)
	filter, err := ParseFQL("coffee, #nowhere")
	if err != nil {
		t.Fatalf("ParseFQL error: %v", err)
	}
	if !filter.Match(target) {
		t.Fatalf("narration term should match a row target")
	}
	column, err := ParseFQL(`change:"1.00"`)
	if err != nil {
		t.Fatalf("ParseFQL error: %v", err)
	}
	if !column.Match(target) {
		t.Fatalf("row columns should be reachable as key:value terms")
	}
}
