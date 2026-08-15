// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package report

import (
	"math/big"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
)

// The tests in this file cover the filter, FQL, and holdings helper branches
// the primary report tests skip.

func decimal(t *testing.T, raw string) ledger.Decimal {
	t.Helper()
	value, err := ledger.ParseDecimal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestFilterPredicateDimensions(t *testing.T) {
	rows := query.Result{
		Columns: []string{"date", "account", "narration", "tags"},
		Rows: []query.Row{
			{"date": "2024-01-01", "account": "Assets:Cash", "narration": "groceries", "tags": []string{"food"}},
			{"date": "2024-02-01", "account": "Assets:Cash:Wallet", "narration": "rent", "tags": []string{"home"}},
			{"date": "2024-03-01", "account": "Expenses:Food", "narration": "cafe", "tags": []string{}},
		},
	}
	if got := Filter(rows, Filters{}); len(got.Rows) != 3 {
		t.Fatal("empty filters must be a no-op")
	}
	if got := Filter(rows, Filters{Account: "Assets:Cash"}); len(got.Rows) != 2 {
		t.Fatalf("subtree account filter rows=%d", len(got.Rows))
	}
	if got := Filter(rows, Filters{Account: "Expenses"}); len(got.Rows) != 1 {
		t.Fatalf("non-matching account rows=%d", len(got.Rows))
	}
	if got := Filter(rows, Filters{Text: "cafe"}); len(got.Rows) != 1 {
		t.Fatalf("substring text filter rows=%d", len(got.Rows))
	}
	// The parsed FQL path handles anchored regex text; a plain substring
	// search could never match "^rent", so one row proves the FQL predicate
	// ran. Tag terms cannot match rows (tags are not string columns).
	if got := Filter(rows, Filters{Text: "\"^rent\""}); len(got.Rows) != 1 {
		t.Fatalf("fql anchored filter rows=%d", len(got.Rows))
	}
	if got := Filter(rows, Filters{Text: "#food"}); len(got.Rows) != 0 {
		t.Fatalf("tag filter against rows rows=%d", len(got.Rows))
	}
	// Unparseable FQL degrades to the substring match: the raw pattern is
	// unlikely to appear verbatim, but rows are compared rather than dropped.
	if got := Filter(rows, Filters{Text: "((^rent"}); len(got.Rows) != 0 {
		t.Fatalf("degraded filter rows=%d", len(got.Rows))
	}

	if got := Filter(rows, Filters{TimePrefix: "2024-01"}); len(got.Rows) != 1 {
		t.Fatalf("time prefix rows=%d", len(got.Rows))
	}
	if got := Filter(rows, Filters{TimeBegin: "2024-02-01", TimeEnd: "2024-03-01"}); len(got.Rows) != 1 {
		t.Fatalf("time range rows=%d", len(got.Rows))
	}
	if got := Filter(rows, Filters{Period: "year"}); len(got.Rows) != 3 {
		t.Fatalf("same-year period rows=%d", len(got.Rows))
	}
	// A row without a date column passes date/period checks by design.
	noDate := query.Result{Columns: []string{"account"}, Rows: []query.Row{{"account": "Assets:Cash"}}}
	if got := Filter(noDate, Filters{Period: "month"}); len(got.Rows) != 1 {
		t.Fatalf("dateless rows=%d", len(got.Rows))
	}
}

func TestSamePeriodBoundaries(t *testing.T) {
	cases := []struct {
		date, anchor, period string
		want                 bool
	}{
		{"2024-03-05", "2024-03-20", "month", true},
		{"2024-04-05", "2024-03-20", "month", false},
		{"2024-03-05", "2024-03-20", "quarter", true},
		{"2024-06-30", "2024-04-01", "quarter", true},
		{"2024-07-01", "2024-04-01", "quarter", false},
		{"", "2024-04-01", "year", true},
		{"2024-01-01", "", "year", true},
		{"2024-01-01", "2024-12-31", "decade", true},
		{"2024", "2024", "month", false}, // too-short dates compare false
	}
	for _, tc := range cases {
		if got := samePeriod(tc.date, tc.anchor, tc.period); got != tc.want {
			t.Fatalf("samePeriod(%q,%q,%q)=%v want %v", tc.date, tc.anchor, tc.period, got, tc.want)
		}
	}
}

func TestFilterJournalPredicateDimensions(t *testing.T) {
	rows := query.Result{
		Columns: []string{"flag", "kind", "tags", "links", "payee", "narration"},
		Rows: []query.Row{
			{"flag": "*", "kind": "transaction", "tags": []string{"food"}, "links": []any{"l1"}, "payee": "Market", "narration": "weekend"},
			{"flag": "!", "kind": "balance", "tags": []any{"home"}, "links": []string{"l2"}, "payee": "Landlord", "narration": "monthly"},
		},
	}
	if got := FilterJournal(rows, JournalFilters{}); len(got.Rows) != 2 {
		t.Fatal("empty filters must be a no-op")
	}
	if got := FilterJournal(rows, JournalFilters{Flag: "*"}); len(got.Rows) != 1 {
		t.Fatalf("flag rows=%d", len(got.Rows))
	}
	if got := FilterJournal(rows, JournalFilters{Kind: "TRANSACTION"}); len(got.Rows) != 1 {
		t.Fatalf("kind rows=%d", len(got.Rows))
	}
	if got := FilterJournal(rows, JournalFilters{Tag: "FOOD"}); len(got.Rows) != 1 {
		t.Fatalf("tag rows=%d", len(got.Rows))
	}
	if got := FilterJournal(rows, JournalFilters{Link: "l2"}); len(got.Rows) != 1 {
		t.Fatalf("link rows=%d", len(got.Rows))
	}
	if got := FilterJournal(rows, JournalFilters{Payee: "market"}); len(got.Rows) != 1 {
		t.Fatalf("payee rows=%d", len(got.Rows))
	}
	if got := FilterJournal(rows, JournalFilters{Narration: "month"}); len(got.Rows) != 1 {
		t.Fatalf("narration rows=%d", len(got.Rows))
	}
	if got := FilterJournal(rows, JournalFilters{Tag: "nope"}); len(got.Rows) != 0 {
		t.Fatalf("unmatched rows=%d", len(got.Rows))
	}
}

func TestFQLFieldAndComparisonHelpers(t *testing.T) {
	target := FQLTarget{Payee: "Market", Narration: "weekend", Metadata: map[string]string{"note": "abc"}}
	if value, ok := target.field("payee"); !ok || value != "Market" {
		t.Fatalf("payee field=%q ok=%v", value, ok)
	}
	if _, ok := target.field("missing"); ok {
		t.Fatal("unknown field must not resolve")
	}
	if value, ok := target.field("note"); !ok || value != "abc" {
		t.Fatalf("metadata field=%q ok=%v", value, ok)
	}
	plain := FQLTarget{}
	if _, ok := plain.field("note"); ok {
		t.Fatal("nil metadata must not resolve")
	}
	for _, tc := range []struct {
		operator string
		want     bool
	}{
		{"=", false}, {">", true}, {">=", true}, {"<", false}, {"<=", false},
	} {
		if got := compareAbs(tc.operator, big.NewFloat(10), big.NewFloat(5)); got != tc.want {
			t.Fatalf("compareAbs(%q)=%v want %v", tc.operator, got, tc.want)
		}
	}
	// Invalid regex patterns degrade to exact equality.
	matcher := fqlStringMatcher("(unclosed")
	if !matcher("(unclosed") || matcher("other") {
		t.Fatal("equality fallback broken")
	}
	// The compiled pattern is case-insensitive, mirroring Fava's Match.
	valid := fqlStringMatcher("^Mar.*")
	if !valid("Market") || !valid("market") {
		t.Fatal("case-insensitive anchor regex broken")
	}
	var nilFilter *FQL
	if !nilFilter.Match(FQLTarget{}) {
		t.Fatal("nil filter matches everything")
	}
}

func TestHoldingValueAtCostBranches(t *testing.T) {
	cost := decimal(t, "10")
	units := decimal(t, "2")
	position := ledger.Position{Units: units, Currency: "SH", Cost: &ledger.Cost{Number: cost, Currency: "USD"}}
	evaluation := ledger.Evaluation{
		Prices: map[string][]ledger.PriceQuote{
			"SH":  {{Date: ledger.Date{Raw: "2024-01-01"}, Base: "SH", Amount: decimal(t, "15"), Currency: "USD"}},
			"USD": {{Date: ledger.Date{Raw: "2024-01-01"}, Base: "USD", Amount: decimal(t, "7"), Currency: "CNY"}},
		},
	}
	// Plain at-cost: value = units * cost.
	value, currency, status, include := holdingValue(evaluation, position, "", "at-cost", "")
	if !include || status != "at-cost" || currency != "USD" || value.(ledger.Decimal).String() != "20" {
		t.Fatalf("at-cost value=%v currency=%s status=%s include=%v", value, currency, status, include)
	}
	// At-cost with a display currency that has an exact quote converts.
	value, currency, status, include = holdingValue(evaluation, position, "", "at-cost", "CNY")
	if !include || status != "at-cost" || currency != "CNY" || value.(ledger.Decimal).String() != "140" {
		t.Fatalf("converted at-cost value=%v currency=%s status=%s", value, currency, status)
	}
	// At-cost with an unavailable display currency stays in its currency.
	value, currency, status, include = holdingValue(evaluation, position, "", "at-cost", "EUR")
	if !include || currency != "USD" {
		t.Fatalf("unconverted at-cost currency=%s", currency)
	}
	// A position without cost contributes no value columns at cost.
	bare := ledger.Position{Units: units, Currency: "SH"}
	if _, _, _, include := holdingValue(evaluation, bare, "", "at-cost", ""); include {
		t.Fatal("costless position must not include value")
	}
	// Market value without any quote reports unavailable-price.
	empty := ledger.Evaluation{}
	if _, _, status, include := holdingValue(empty, position, "", "market-value", ""); !include || status != "unavailable-price" {
		t.Fatalf("missing quote status=%s include=%v", status, include)
	}
}

func TestPositionSurvivesAsOfBoundaries(t *testing.T) {
	date := ledger.Date{Raw: "2024-01-01"}
	cost := ledger.Cost{Number: decimal(t, "1"), Currency: "USD", Date: &date}
	units := decimal(t, "1")
	if !positionSurvivesAsOf(ledger.Position{Units: units, Currency: "SH", Cost: &cost}, "2024-01-01") {
		t.Fatal("as-of equal to cost date must survive")
	}
	if !positionSurvivesAsOf(ledger.Position{Units: units, Currency: "SH", Cost: &cost}, "") {
		t.Fatal("empty as-of keeps every lot")
	}
	if positionSurvivesAsOf(ledger.Position{Units: units, Currency: "SH", Cost: &cost}, "2023-12-31") {
		t.Fatal("earlier as-of must drop the lot")
	}
	if !positionSurvivesAsOf(ledger.Position{Units: units, Currency: "SH"}, "2023-12-31") {
		t.Fatal("costless lot always survives")
	}
}

func TestLatestDateAndTextMatcherHelpers(t *testing.T) {
	rows := []query.Row{{"date": "2024-01-01"}, {"date": "2024-05-05"}, {"account": "x"}}
	if latestDate(rows) != "2024-05-05" {
		t.Fatalf("latestDate=%q", latestDate(rows))
	}
	if textMatcher("", "") != nil {
		t.Fatal("empty text yields no matcher")
	}
	if textMatcher("plain", "plain") == nil {
		t.Fatal("plain text yields a matcher")
	}
}
