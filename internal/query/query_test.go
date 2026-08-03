// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package query

import (
	"strings"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

func TestParseAndEvaluateGroupedQuery(t *testing.T) {
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "one"
  Assets:Cash 2 USD
  Equity:Opening -2 USD
2000-01-03 * "two"
  Assets:Cash 3 USD
  Equity:Opening -3 USD
`
	file, diagnostics := ledger.ParseText("query.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}
	result, err := Evaluate("SELECT account, sum(number) AS total FROM postings WHERE currency = 'USD' GROUP BY account ORDER BY total DESC", *evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 2 || formatValue(result.Rows[0]["account"]) != "Assets:Cash" || formatValue(result.Rows[0]["total"]) != "5" {
		t.Fatalf("result=%+v", result)
	}
	var csv strings.Builder
	if err := result.WriteCSV(&csv); err != nil || !strings.Contains(csv.String(), "account,total") {
		t.Fatalf("csv=%q err=%v", csv.String(), err)
	}
}

func TestQueryFunctionsAndLimit(t *testing.T) {
	file, diagnostics := ledger.ParseText("query.bean", []byte("2000-01-01 open Assets:Cash USD\n"))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	result, err := Evaluate("SELECT count(*) AS count, year(date) AS year FROM entries LIMIT 1", *evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || formatValue(result.Rows[0]["count"]) != "1" || formatValue(result.Rows[0]["year"]) != "2000" {
		t.Fatalf("result=%+v", result)
	}
}

func TestQueryRejectsUnknownTable(t *testing.T) {
	if _, err := Evaluate("SELECT * FROM private_table", ledger.Evaluation{}); err == nil {
		t.Fatal("unknown table accepted")
	}
}

// TestLexerHandlesNonASCIIWhitespace verifies that the query lexer treats
// multi-byte Unicode whitespace (e.g. U+00A0 no-break space and U+3000 ideo-
// graphic space) as a token separator, not as part of an identifier. The old
// byte-walking lexer inspected rune(text[i]) on a single byte and misclassified
// these separators.
func TestLexerHandlesNonASCIIWhitespace(t *testing.T) {
	for _, separator := range []string{"\u00a0", "\u3000", "\t", " "} {
		query := "SELECT account FROM accounts WHERE account = 'Assets:Cash'" + separator + "ORDER BY account"
		result, err := Evaluate(query, ledger.Evaluation{})
		if err != nil {
			t.Fatalf("separator %q err=%v", separator, err)
		}
		if result.Columns == nil {
			t.Fatalf("separator %q produced no result", separator)
		}
	}
}
