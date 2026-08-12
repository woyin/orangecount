// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

package planning

import (
	"testing"

	"orangecount/internal/ledger"
)

func TestCalculateUsesLowestConservativeHeadroom(t *testing.T) {
	result, err := Calculate(Input{
		Today: date("2026-08-12"), HorizonEnd: date("2026-09-01"), RecordedThrough: date("2026-08-12"), Currency: "CNY",
		SpendableFunds: []AccountBalance{{Account: "Assets:CMB", Amount: decimal(t, "20000")}, {Account: "Assets:WeChat", Amount: decimal(t, "3000")}},
		ShortTermDebts: []AccountBalance{{Account: "Liabilities:CMB:Credit", Amount: decimal(t, "8000")}},
		MinimumReserve: decimal(t, "5000"),
		Plans: []Plan{
			{ID: "rent", Date: date("2026-08-15"), Amount: decimal(t, "5000"), Direction: Outflow, Commitment: Committed},
			{ID: "fun", Date: date("2026-08-20"), Amount: decimal(t, "1000"), Direction: Outflow, Commitment: Adjustable},
			{ID: "salary", Date: date("2026-09-01"), Amount: decimal(t, "10000"), Direction: Inflow},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.PrimarySafeToSpend.String(); got != "4000" {
		t.Fatalf("primary=%s", got)
	}
	if result.FundingShortfall.Sign() != 0 {
		t.Fatalf("shortfall=%s", result.FundingShortfall)
	}
	if got := result.LiquidityLowPoint.Date.String(); got != "2026-08-20" {
		t.Fatalf("low date=%s", got)
	}
	if got := result.LiquidityLowPoint.Headroom.String(); got != "4000" {
		t.Fatalf("low=%s", got)
	}
	if got := result.Timeline[len(result.Timeline)-1].IgnoredInflows.String(); got != "10000" {
		t.Fatalf("ignored inflows=%s", got)
	}
	if !result.CurrentPlanning {
		t.Fatal("today-recorded plan should be current")
	}
}

func TestCalculateShowsShortfallAndFirstLowPoint(t *testing.T) {
	result, err := Calculate(Input{
		Today: date("2026-08-12"), HorizonEnd: date("2026-08-20"), RecordedThrough: date("2026-08-11"), Currency: "CNY",
		SpendableFunds: []AccountBalance{{Account: "Assets:CMB", Amount: decimal(t, "10000")}},
		MinimumReserve: decimal(t, "5000"),
		Plans:          []Plan{{ID: "rent", Date: date("2026-08-13"), Amount: decimal(t, "8000"), Direction: Outflow, Commitment: Committed}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.PrimarySafeToSpend.String(); got != "0" {
		t.Fatalf("primary=%s", got)
	}
	if got := result.FundingShortfall.String(); got != "3000" {
		t.Fatalf("shortfall=%s", got)
	}
	if got := result.LiquidityLowPoint.Date.String(); got != "2026-08-13" {
		t.Fatalf("low date=%s", got)
	}
	if result.CurrentPlanning {
		t.Fatal("stale recorded-through date must not be current")
	}
}

func TestCalculateCountsDebtFundedPurchaseOnceAndIgnoresFutureIncome(t *testing.T) {
	result, err := Calculate(Input{
		Today: date("2026-08-12"), HorizonEnd: date("2026-08-20"), RecordedThrough: date("2026-08-12"), Currency: "CNY",
		SpendableFunds: []AccountBalance{{Account: "Assets:CMB", Amount: decimal(t, "10000")}},
		ShortTermDebts: []AccountBalance{{Account: "Liabilities:Credit", Amount: decimal(t, "2000")}},
		MinimumReserve: decimal(t, "1000"),
		Plans: []Plan{
			{ID: "laptop", Date: date("2026-08-13"), Amount: decimal(t, "3000"), Direction: Outflow, Commitment: Adjustable},
			{ID: "salary", Date: date("2026-08-14"), Amount: decimal(t, "9000"), Direction: Inflow},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 10000 - 2000 current card debt - 1000 reserve - 3000 proposed card
	// purchase. The later salary cannot increase the primary amount.
	if got := result.PrimarySafeToSpend.String(); got != "4000" {
		t.Fatalf("primary=%s", got)
	}
	if got := result.Timeline[2].Headroom.String(); got != "4000" {
		t.Fatalf("income changed primary headroom: %s", got)
	}
}

func TestCalculateAppliesTodayPlansOnce(t *testing.T) {
	result, err := Calculate(Input{
		Today:           date("2026-08-12"),
		HorizonEnd:      date("2026-08-20"),
		RecordedThrough: date("2026-08-12"),
		Currency:        "CNY",
		SpendableFunds:  []AccountBalance{{Account: "Assets:CMB", Amount: decimal(t, "1000")}},
		Plans: []Plan{
			{ID: "today", Date: date("2026-08-12"), Amount: decimal(t, "300"), Direction: Outflow, Commitment: Committed},
			{ID: "income", Date: date("2026-08-12"), Amount: decimal(t, "900"), Direction: Inflow},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Timeline) != 1 {
		t.Fatalf("timeline steps=%d", len(result.Timeline))
	}
	if got := result.PrimarySafeToSpend.String(); got != "700" {
		t.Fatalf("primary=%s", got)
	}
	if got := result.Timeline[0].IgnoredInflows.String(); got != "900" {
		t.Fatalf("ignored inflows=%s", got)
	}
}

func TestCalculateValidatesUnsafeInputs(t *testing.T) {
	base := Input{Today: date("2026-08-12"), HorizonEnd: date("2026-08-20"), RecordedThrough: date("2026-08-12"), Currency: "CNY"}
	for name, mutate := range map[string]func(*Input){
		"negative reserve":     func(in *Input) { in.MinimumReserve = decimal(t, "-1") },
		"horizon before today": func(in *Input) { in.HorizonEnd = date("2026-08-11") },
		"negative debt": func(in *Input) {
			in.ShortTermDebts = []AccountBalance{{Account: "Liabilities:CMB", Amount: decimal(t, "-1")}}
		},
		"invalid outflow commitment": func(in *Input) {
			in.Plans = []Plan{{ID: "p", Date: date("2026-08-13"), Amount: decimal(t, "1"), Direction: Outflow}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			in := base
			mutate(&in)
			if _, err := Calculate(in); err == nil {
				t.Fatal("Calculate accepted invalid input")
			}
		})
	}
}

func TestCalculateIncludesOverdrawnSpendableAccount(t *testing.T) {
	result, err := Calculate(Input{
		Today:           date("2026-08-12"),
		HorizonEnd:      date("2026-08-20"),
		RecordedThrough: date("2026-08-12"),
		Currency:        "CNY",
		SpendableFunds: []AccountBalance{
			{Account: "Assets:CMB", Amount: decimal(t, "100")},
			{Account: "Assets:Alipay", Amount: decimal(t, "-30")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.CurrentFunds.String(); got != "70" {
		t.Fatalf("funds=%s", got)
	}
}

func date(raw string) ledger.Date {
	return ledger.Date{Year: int(raw[0]-'0')*1000 + int(raw[1]-'0')*100 + int(raw[2]-'0')*10 + int(raw[3]-'0'), Month: int(raw[5]-'0')*10 + int(raw[6]-'0'), Day: int(raw[8]-'0')*10 + int(raw[9]-'0'), Raw: raw}
}

func decimal(t *testing.T, raw string) ledger.Decimal {
	t.Helper()
	value, err := ledger.ParseDecimal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
