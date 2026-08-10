// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package query

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

func queryEvaluation(t *testing.T) ledger.Evaluation {
	t.Helper()
	text := `2000-01-01 open Assets:Cash USD,CNY
2000-01-01 open Expenses:Food USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "Grocer" "first" #food ^receipt
  Assets:Cash -2 USD
  Expenses:Food 2 USD
2000-01-03 * "Grocer" "second" #food #weekly ^receipt
  Assets:Cash -3 USD
  Expenses:Food 3 USD
2000-01-04 price USD 1.5 CNY
`
	file, diagnostics := ledger.ParseText("query.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation diagnostics=%+v", evaluation.Diagnostics)
	}
	return *evaluation
}

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

func TestEvaluateSupportsArithmeticPredicatesAndAggregateFunctions(t *testing.T) {
	evaluation := queryEvaluation(t)
	result, err := Evaluate(`SELECT account, sum(number) AS total, avg(number) AS average, min(number) AS minimum, max(number) AS maximum, first(number) AS first, last(number) AS last, count(number) AS count FROM postings WHERE has_tag(tags, 'food') AND has_link(links, 'receipt') GROUP BY account ORDER BY total DESC`, evaluation)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 2 || result.Rows[0]["account"] != "Expenses:Food" {
		t.Fatalf("aggregate result=%+v", result)
	}
	for _, name := range []string{"total", "average", "minimum", "maximum", "first", "last", "count"} {
		if got := formatValue(result.Rows[0][name]); got == "" || got == "0" {
			t.Fatalf("aggregate %s=%q result=%+v", name, got, result)
		}
	}
	calendared, err := Evaluate(`SELECT year(date) AS year, month(date) AS month, day(date) AS day FROM entries WHERE year(date) >= 2000 ORDER BY day DESC LIMIT 2`, evaluation)
	if err != nil || len(calendared.Rows) != 2 || formatValue(calendared.Rows[0]["day"]) != "4" {
		t.Fatalf("calendar result=%+v err=%v", calendared, err)
	}
	arithmetic, err := Evaluate(`SELECT account, -number AS negated, number + 2 AS plus_two, number / 2 AS half FROM postings WHERE not (account = 'Equity:Opening') AND number < 0 ORDER BY negated LIMIT 1`, evaluation)
	if err != nil || len(arithmetic.Rows) != 1 || formatValue(arithmetic.Rows[0]["negated"]) != "2" || formatValue(arithmetic.Rows[0]["half"]) != "-1" {
		t.Fatalf("arithmetic result=%+v err=%v", arithmetic, err)
	}
}

func TestEvaluateCoversTablesStarAndEmptyAggregates(t *testing.T) {
	evaluation := queryEvaluation(t)
	for _, query := range []string{
		`SELECT * FROM entries ORDER BY date`,
		`SELECT account, currency, balance, opened FROM accounts ORDER BY account`,
		`SELECT date, currency, amount, quote_currency FROM prices`,
	} {
		result, err := Evaluate(query, evaluation)
		if err != nil || len(result.Columns) == 0 || len(result.Rows) == 0 {
			t.Fatalf("query=%q result=%+v err=%v", query, result, err)
		}
	}
	empty, err := Evaluate(`SELECT sum(number) AS total, count(*) AS count FROM postings WHERE account = 'Assets:Missing'`, evaluation)
	if err != nil || len(empty.Rows) != 0 {
		t.Fatalf("empty aggregate=%+v err=%v", empty, err)
	}
	accounts, err := EvaluateQuery(Query{Select: []SelectItem{{Expr: fieldExpr{name: "account"}}, {Expr: fieldExpr{name: "balance"}}}, From: "accounts"}, ledger.Evaluation{Accounts: map[string]ledger.AccountState{"Assets:Empty": {Opened: ledger.Date{Raw: "2000-01-01"}}}})
	if err != nil || len(accounts.Rows) != 1 || formatValue(accounts.Rows[0]["balance"]) != "0" {
		t.Fatalf("zero-balance account=%+v err=%v", accounts, err)
	}
}

func TestQueryParsingAndEvaluationRejectInvalidExpressions(t *testing.T) {
	for _, text := range []string{
		"", "SELECT FROM postings", "SELECT account postings", "SELECT account FROM postings GROUP account", "SELECT account FROM postings ORDER account", "SELECT account FROM postings LIMIT -1", "SELECT account FROM postings trailing",
	} {
		if _, err := Parse(text); err == nil {
			t.Errorf("Parse(%q) succeeded", text)
		}
	}
	evaluation := queryEvaluation(t)
	for _, text := range []string{
		`SELECT number / 0 FROM postings`, `SELECT unknown(number) FROM postings`, `SELECT sum() FROM postings`, `SELECT sum(number, 1) FROM postings`, `SELECT has_tag(tags) FROM postings`, `SELECT -account FROM postings`,
	} {
		if _, err := Evaluate(text, evaluation); err == nil {
			t.Errorf("Evaluate(%q) succeeded", text)
		}
	}
}

func TestResultCSVAndValueHelpersHandleAllSupportedShapes(t *testing.T) {
	result := Result{Columns: []string{"none", "decimal", "date", "kind", "strings", "boolean", "other"}, Rows: []Row{{
		"none": nil, "decimal": mustDecimal(t, "1.25"), "date": ledger.Date{Raw: "2000-01-02"}, "kind": ledger.KindOpen, "strings": []string{"a", "b"}, "boolean": true, "other": 42,
	}}}
	var out bytes.Buffer
	if err := result.WriteCSV(&out); err != nil || !strings.Contains(out.String(), "1.25,2000-01-02,open,\"a,b\",true,42") {
		t.Fatalf("csv=%q err=%v", out.String(), err)
	}
	if err := result.WriteCSV(errorWriter{}); !errors.Is(err, errWrite) {
		t.Fatalf("write error=%v", err)
	}
	if got := parseDate("not-a-date"); got.Raw != "not-a-date" || got.Year != 0 {
		t.Fatalf("invalid date=%+v", got)
	}
	if got := compareValues("2", 10); got >= 0 {
		t.Fatalf("numeric comparison=%d", got)
	}
	if !truthy(1) || truthy(ledger.Zero()) || truthy("") || truthy(nil) {
		t.Fatal("truthiness is inconsistent")
	}
}

var errWrite = errors.New("write failed")

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errWrite }

func mustDecimal(t *testing.T, raw string) ledger.Decimal {
	t.Helper()
	value, err := ledger.ParseDecimal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
