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
