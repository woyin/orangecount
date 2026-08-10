// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"math/big"
	"reflect"
	"testing"

	"orangecount/internal/source"
)

func testNumber(t *testing.T, raw string) Number {
	t.Helper()
	value, err := ParseDecimal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return Number{Raw: raw, Rat: value.Rat()}
}

func testCost(t *testing.T, raw, currency, date, label string) Cost {
	t.Helper()
	value, err := ParseDecimal(raw)
	if err != nil {
		t.Fatal(err)
	}
	cost := Cost{Number: value, Currency: currency, Label: label}
	if date != "" {
		cost.Date = &Date{Raw: date, Year: 2000, Month: 1, Day: 1}
	}
	return cost
}

func TestEvaluatorCostConstraintsAndBookingOrderRemainDeterministic(t *testing.T) {
	date := Date{Raw: "2000-01-01", Year: 2000, Month: 1, Day: 1}
	spec := CostSpec{Components: []Value{
		{Kind: ValueAmount, Amount: Amount{Number: testNumber(t, "10"), Currency: "USD"}},
		{Kind: ValueDate, Date: date},
		{Kind: ValueString, String: "lot"},
	}}
	constraints := deriveCostConstraints(spec)
	if constraints.number == nil || constraints.number.String() != "10" || constraints.currency != "USD" || constraints.date == nil || constraints.label != "lot" {
		t.Fatalf("constraints=%+v", constraints)
	}
	matching := testCost(t, "10", "USD", "2000-01-01", "lot")
	if !costMatchesPosition(constraints, matching) {
		t.Fatal("matching cost was rejected")
	}
	for _, cost := range []Cost{testCost(t, "11", "USD", "2000-01-01", "lot"), testCost(t, "10", "EUR", "2000-01-01", "lot"), testCost(t, "10", "USD", "2000-01-01", "other")} {
		if costMatchesPosition(constraints, cost) {
			t.Errorf("mismatched cost accepted: %+v", cost)
		}
	}
	positions := []Position{
		{Units: mustDecimal(t, "1"), Currency: "SH", Cost: &Cost{Number: mustDecimal(t, "10"), Currency: "USD", Date: &Date{Raw: "2000-01-01", Year: 2000, Month: 1, Day: 1}}},
		{Units: mustDecimal(t, "1"), Currency: "SH", Cost: &Cost{Number: mustDecimal(t, "12"), Currency: "USD", Date: &Date{Raw: "2000-01-02", Year: 2000, Month: 1, Day: 2}}},
	}
	for booking, want := range map[string]string{"FIFO": "10", "LIFO": "12", "HIFO": "12"} {
		if got := orderBookingMatches(positions, booking)[0].Cost.Number.String(); got != want {
			t.Errorf("%s first cost=%s, want %s", booking, got, want)
		}
	}
	generated := costSpecFromCost(matching, &CostSpec{Total: true, Components: []Value{{Kind: ValueCurrency, String: "ignored"}}})
	if generated.Total || len(generated.Components) != 3 || generated.Components[0].Amount.Currency != "USD" {
		t.Fatalf("generated cost=%+v", generated)
	}
	if normalized, ok := normalizeCost(generated); !ok || normalized.Currency != "USD" || normalized.Label != "lot" {
		t.Fatalf("normalized=%+v ok=%v", normalized, ok)
	}
	if _, ok := normalizeCost(CostSpec{}); ok {
		t.Fatal("empty cost was normalized")
	}
}

func TestEvaluatorArithmeticInferenceAndOrderingHelpers(t *testing.T) {
	posting := Posting{Units: &Amount{Number: testNumber(t, "2"), Currency: "SH"}, Cost: &CostSpec{Components: []Value{{Kind: ValueAmount, Amount: Amount{Number: testNumber(t, "3"), Currency: "USD"}}}}}
	if currency, value, ok := balancingContribution(posting, mustDecimal(t, "2"), "SH"); !ok || currency != "USD" || value.String() != "6" {
		t.Fatalf("cost contribution currency=%q value=%s ok=%v", currency, value, ok)
	}
	posting.Cost.Total = true
	if _, value, _ := balancingContribution(posting, mustDecimal(t, "-2"), "SH"); value.String() != "-3" {
		t.Fatalf("total contribution=%s", value)
	}
	posting.Cost = nil
	posting.Price = &PriceSpec{Amount: Amount{Number: testNumber(t, "4"), Currency: "EUR"}}
	if currency, value, ok := balancingContribution(posting, mustDecimal(t, "2"), "SH"); !ok || currency != "EUR" || value.String() != "8" {
		t.Fatalf("price contribution currency=%q value=%s ok=%v", currency, value, ok)
	}
	if got := signedTotal(mustDecimal(t, "-2"), mustDecimal(t, "5")); got.String() != "-5" || divideDecimal(mustDecimal(t, "1"), Zero()).String() != "0" {
		t.Fatalf("arithmetic helpers=%s", got)
	}
	if got := inferredTolerance([]Posting{{Units: &Amount{Number: testNumber(t, "1.23")}}, {Units: &Amount{Number: testNumber(t, "1.234")}}}); got.String() != "0.0005" {
		t.Fatalf("inferred tolerance=%s", got)
	}

	e := &evaluator{accounts: map[string]*accountWork{"Assets:Cash": {state: AccountState{Currencies: []string{"USD"}}}}}
	if got := e.inferPostingCurrency(Posting{Account: "Assets:Cash"}, map[string]Decimal{}); got != "USD" {
		t.Fatalf("account inference=%q", got)
	}
	if got := e.inferPostingCurrency(Posting{}, map[string]Decimal{"EUR": Zero()}); got != "EUR" {
		t.Fatalf("total inference=%q", got)
	}
	if currency, factor, ok := e.inferenceTarget(Posting{Price: &PriceSpec{Amount: Amount{Number: testNumber(t, "2"), Currency: "EUR"}}}, "USD", map[string]Decimal{}); !ok || currency != "EUR" || factor.String() != "2" {
		t.Fatalf("price target currency=%q factor=%s ok=%v", currency, factor, ok)
	}
	if currency, _, ok := e.inferenceTarget(Posting{}, "", map[string]Decimal{}); ok || currency != "" {
		t.Fatalf("unexpected inference target currency=%q ok=%v", currency, ok)
	}
	if !reflect.DeepEqual(sourceOrder(nil, map[source.FileID]*File{2: {}, 1: {}}), []source.FileID{1, 2}) {
		t.Fatal("graph-independent source order was not sorted")
	}
	graph := &source.Graph{Order: []source.FileID{3, 1}}
	if !reflect.DeepEqual(sourceOrder(graph, nil), []source.FileID{3, 1}) {
		t.Fatal("graph order was not preserved")
	}
}

func mustDecimal(t *testing.T, raw string) Decimal {
	t.Helper()
	value, err := ParseDecimal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestEvaluationAccountAccessorAndStringDefendState(t *testing.T) {
	evaluation := &Evaluation{Valid: true, Accounts: map[string]AccountState{"Assets:Cash": {Balances: map[string]Decimal{"USD": NewDecimal(big.NewRat(1, 1))}, Currencies: []string{"USD"}, Positions: []Position{{Currency: "USD"}}}}, Prices: map[string][]PriceQuote{"USD": {{Currency: "EUR"}}}}
	account, ok := evaluation.Account("Assets:Cash")
	if !ok || account.Balances["USD"].String() != "1" || evaluation.String() != "accounts=1 prices=1 valid=true" {
		t.Fatalf("account=%+v ok=%v string=%q", account, ok, evaluation.String())
	}
	delete(account.Balances, "USD")
	account.Currencies[0] = "EUR"
	if _, ok := evaluation.Account("missing"); ok || evaluation.Accounts["Assets:Cash"].Balances["USD"].String() != "1" || evaluation.Accounts["Assets:Cash"].Currencies[0] != "USD" {
		t.Fatal("Account accessor leaked mutable state")
	}
	var nilEvaluation *Evaluation
	if _, ok := nilEvaluation.Account("Assets:Cash"); ok || nilEvaluation.String() != "<nil evaluation>" {
		t.Fatal("nil evaluation accessors are inconsistent")
	}
}
