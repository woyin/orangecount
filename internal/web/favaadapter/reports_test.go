// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

import (
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/report"
)

func TestProjectTreeReportPreservesFavaTreeShapeAndExactNetProfit(t *testing.T) {
	income, err := ledger.ParseDecimal("100.25")
	if err != nil {
		t.Fatal(err)
	}
	expenses, err := ledger.ParseDecimal("-40.10")
	if err != nil {
		t.Fatal(err)
	}
	evaluation := ledger.Evaluation{
		Accounts: map[string]ledger.AccountState{
			"Income:Salary": {Name: "Income:Salary", Balances: map[string]ledger.Decimal{"USD": income}},
			"Expenses:Food": {Name: "Expenses:Food", Balances: map[string]ledger.Decimal{"USD": expenses}},
			"Assets:Cash":   {Name: "Assets:Cash", Balances: map[string]ledger.Decimal{"USD": ledger.Zero()}},
		},
		Prices:  map[string][]ledger.PriceQuote{},
		Options: map[string]string{},
		Valid:   true,
	}

	projected := ProjectTreeReport(evaluation, "income_statement", report.Filters{}, "month", "USD", "at-cost")
	if len(projected.Trees) != 3 {
		t.Fatalf("trees=%d want 3: %+v", len(projected.Trees), projected.Trees)
	}
	if projected.Trees[0].Account != "Income" || projected.Trees[1].Account != "Net Profit" || projected.Trees[2].Account != "Expenses" {
		t.Fatalf("tree order=%q/%q/%q", projected.Trees[0].Account, projected.Trees[1].Account, projected.Trees[2].Account)
	}
	if got := projected.Trees[1].BalanceChildren["USD"].Exact; got != "60.15" {
		t.Fatalf("net profit=%q want 60.15", got)
	}
	if len(projected.Trees[0].Children) != 1 || projected.Trees[0].Children[0].Account != "Income:Salary" {
		t.Fatalf("income children=%+v", projected.Trees[0].Children)
	}
	if projected.Trees[1].Balance["USD"].Exact != "60.15" {
		t.Fatalf("net own balance=%q", projected.Trees[1].Balance["USD"].Exact)
	}
}

func TestProjectTreeReportValuesLotsAndRollsSubtreeTotalsUp(t *testing.T) {
	dec := func(raw string) ledger.Decimal {
		value, err := ledger.ParseDecimal(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		return value
	}
	date := ledger.Date{Raw: "2024-01-01", Year: 2024, Month: 1, Day: 1}
	// One sibling holds shares at cost, the other plain cash. Under at-cost the
	// parent total must be the sum of the converted amounts in one currency,
	// not a mix of cash plus raw share units.
	evaluation := ledger.Evaluation{
		Accounts: map[string]ledger.AccountState{
			"Assets:Broker:Shares": {
				Name:      "Assets:Broker:Shares",
				Balances:  map[string]ledger.Decimal{"SH": dec("10")},
				Positions: []ledger.Position{{Units: dec("10"), Currency: "SH", Cost: &ledger.Cost{Number: dec("7"), Currency: "USD", Date: &date}}},
			},
			"Assets:Broker:Cash": {Name: "Assets:Broker:Cash", Balances: map[string]ledger.Decimal{"USD": dec("30")}},
		},
		Prices:  map[string][]ledger.PriceQuote{},
		Options: map[string]string{},
		Valid:   true,
	}

	atCost := ProjectTreeReport(evaluation, "balance_sheet", report.Filters{}, "month", "USD", "at-cost")
	assets := atCost.Trees[0]
	if assets.Account != "Assets" {
		t.Fatalf("first tree=%q", assets.Account)
	}
	// 10 shares * 7 USD + 30 USD cash.
	if got := assets.BalanceChildren["USD"].Exact; got != "100" {
		t.Fatalf("at-cost subtree total=%q want 100", got)
	}
	if _, ok := assets.BalanceChildren["SH"]; ok {
		t.Fatalf("converted commodity must not remain in the subtree total: %+v", assets.BalanceChildren)
	}

	units := ProjectTreeReport(evaluation, "balance_sheet", report.Filters{}, "month", "USD", "units")
	unitsAssets := units.Trees[0]
	if got := unitsAssets.BalanceChildren["SH"].Exact; got != "10" {
		t.Fatalf("units subtree SH=%q want 10", got)
	}
	if got := unitsAssets.BalanceChildren["USD"].Exact; got != "30" {
		t.Fatalf("units subtree USD=%q want 30", got)
	}
}

func TestProjectTreeReportRootsTrialBalanceAtTheTreeRoot(t *testing.T) {
	dec := func(raw string) ledger.Decimal {
		value, err := ledger.ParseDecimal(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		return value
	}
	// The trial balance is rooted at the synthetic "" account. Looking that up
	// as a child of the tree root found nothing and returned an empty
	// placeholder, so the whole table rendered blank.
	evaluation := ledger.Evaluation{
		Accounts: map[string]ledger.AccountState{
			"Assets:Cash":      {Name: "Assets:Cash", Balances: map[string]ledger.Decimal{"USD": dec("10")}},
			"Income:Salary":    {Name: "Income:Salary", Balances: map[string]ledger.Decimal{"USD": dec("-10")}},
			"Expenses:Grocery": {Name: "Expenses:Grocery", Balances: map[string]ledger.Decimal{"USD": dec("4")}},
		},
		Prices:  map[string][]ledger.PriceQuote{},
		Options: map[string]string{},
		Valid:   true,
	}
	projected := ProjectTreeReport(evaluation, "trial_balance", report.Filters{}, "month", "USD", "at-cost")
	if len(projected.Trees) != 1 {
		t.Fatalf("trees=%d want 1", len(projected.Trees))
	}
	root := projected.Trees[0]
	if root.Account != "" {
		t.Fatalf("trial balance root account=%q want empty", root.Account)
	}
	if len(root.Children) == 0 {
		t.Fatal("trial balance tree must carry the account roots, not an empty placeholder")
	}
	seen := map[string]bool{}
	for _, child := range root.Children {
		seen[child.Account] = true
	}
	for _, want := range []string{"Assets", "Income", "Expenses"} {
		if !seen[want] {
			t.Fatalf("missing root %q in %v", want, seen)
		}
	}
}
