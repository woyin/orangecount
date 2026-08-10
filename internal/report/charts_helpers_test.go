// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package report

import (
	"testing"

	"orangecount/internal/ledger"
)

func TestChartHelperContractsCoverUnavailableAndNativeMeasurements(t *testing.T) {
	for route, want := range map[string]string{"income-statement": ChartStackedBar, "balance-sheet": ChartLine, "accounts": ChartLine, "trial-balance": ChartHierarchy, "unknown": ""} {
		if got := chartKind(route); got != want {
			t.Errorf("chartKind(%q)=%q, want %q", route, got, want)
		}
	}
	for route, want := range map[string]string{"balance-sheet": "Balance sheet", "income-statement": "Income statement", "accounts": "Account balance", "trial-balance": "Trial balance", "unknown": ""} {
		if got := chartTitle(route); got != want {
			t.Errorf("chartTitle(%q)=%q, want %q", route, got, want)
		}
	}
	for _, test := range []struct{ raw, interval, want string }{{"", "month", ""}, {"2000-13-01", "quarter", ""}, {"2000-01", "quarter", "2000-Q1"}, {"2000-10", "quarter", "2000-Q4"}, {"2000-01", "year", "2000"}} {
		if got := chartPeriodKey(test.raw, test.interval); got != test.want {
			t.Errorf("chartPeriodKey(%q, %q)=%q, want %q", test.raw, test.interval, got, test.want)
		}
	}
	cost := &ledger.CostSpec{Components: []ledger.Value{{Kind: ledger.ValueAmount, Amount: ledger.Amount{Number: chartNumber(t, "3"), Currency: "USD"}}}}
	posting := chartPosting{currency: "SH", amount: chartDecimal(t, "2"), cost: cost}
	if amount, status := chartAmount(ledger.Evaluation{}, posting, "2000-01-01", "USD", "at-cost"); status != amountOK || amount.String() != "6" {
		t.Fatalf("cost chart amount=%s status=%d", amount, status)
	}
	if amount, status := chartAmount(ledger.Evaluation{}, posting, "2000-01-01", "", "at-cost"); status != amountNativeMulti || amount.String() != "2" {
		t.Fatalf("native chart amount=%s status=%d", amount, status)
	}
	if _, status := chartAmount(ledger.Evaluation{}, posting, "2000-01-01", "EUR", "market-value"); status != amountUnavailablePrice {
		t.Fatalf("missing price status=%d", status)
	}
	priced := ledger.Evaluation{Prices: map[string][]ledger.PriceQuote{"SH": {{Date: ledger.Date{Raw: "2000-01-01"}, Base: "SH", Amount: chartDecimal(t, "4"), Currency: "EUR"}}}}
	if amount, status := chartAmount(priced, posting, "2000-01-01", "EUR", "market-value"); status != amountOK || amount.String() != "8" {
		t.Fatalf("market chart amount=%s status=%d", amount, status)
	}
	if _, status := chartAmount(priced, posting, "2000-01-01", "USD", "market-value"); status != amountUnavailableCurrency {
		t.Fatalf("wrong quote currency status=%d", status)
	}
	if _, _, _, ok := chartCost(&ledger.CostSpec{Components: []ledger.Value{{Kind: ledger.ValueString, String: "no amount"}}}); ok {
		t.Fatal("non-amount lot component became a chart cost")
	}
	for _, test := range []struct {
		status         chartAmountStatus
		currency, want string
	}{
		{amountUnavailablePrice, "USD", "unavailable-price"}, {amountUnavailableCurrency, "USD", "unavailable-currency"}, {amountNativeMulti, "", "native-multi"}, {amountOK, "", "at-cost"},
	} {
		availability := chartAvailability{}
		availability.observe(test.status)
		if got := availability.resolve(test.currency); got != test.want {
			t.Errorf("availability status=%d currency=%q got=%q want=%q", test.status, test.currency, got, test.want)
		}
	}
	if value, currency := nativeChartAmount(chartPosting{currency: "SH", amount: chartDecimal(t, "2"), cost: cost}, "at-cost"); value.String() != "6" || currency != "USD" {
		t.Fatalf("native at-cost=%s %s", value, currency)
	}
	if value, currency := nativeChartAmount(chartPosting{currency: "SH", amount: chartDecimal(t, "2"), cost: cost}, "units"); value.String() != "2" || currency != "SH" {
		t.Fatalf("native units=%s %s", value, currency)
	}
	totalCost := &ledger.CostSpec{Total: true, Components: cost.Components}
	if value, currency := nativeChartAmount(chartPosting{currency: "SH", amount: chartDecimal(t, "-2"), cost: totalCost}, "at-cost"); value.String() != "-3" || currency != "USD" {
		t.Fatalf("native total reduction=%s %s", value, currency)
	}
	if accountRoot("Assets:Cash") != "Assets" || accountRoot("Assets") != "Assets" || !accountWithin("Assets", "Assets:Cash") || accountWithin("Assets:Cash", "Assets:Cashflow") {
		t.Fatal("account hierarchy helpers changed")
	}
	availability := chartAvailability{unavailablePrice: true, unavailableCurrency: true}
	if availability.resolve("USD") != "unavailable-price" || availability.resolve("") != "unavailable-price" {
		t.Fatalf("availability priority=%+v", availability)
	}
	if charts := ReportCharts(ledger.Evaluation{}, "unknown", "month", "at-cost"); charts != nil {
		t.Fatalf("unknown native charts=%+v", charts)
	}
	if chart := trialBalanceChart(ledger.Evaluation{}, []string{"2000-01"}, map[string]string{"2000-01": "2000-01-01"}, "month", "EUR", "at-cost"); chart.Availability != "unavailable-currency" {
		t.Fatalf("missing trial balance currency=%+v", chart)
	}
	if empty := ReportChart(ledger.Evaluation{}, "accounts", "year", "USD", "at-cost", ""); empty.Kind != ChartLine || empty.Measure != "" {
		t.Fatalf("empty report chart=%+v", empty)
	}
	if average := AccountAverageCostChart(ledger.Evaluation{}, "month", ""); average.Measure != "average-cost" || len(average.Series) != 0 {
		t.Fatalf("empty average chart=%+v", average)
	}
}

func chartNumber(t *testing.T, raw string) ledger.Number {
	t.Helper()
	value, err := ledger.ParseDecimal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return ledger.Number{Raw: raw, Rat: value.Rat()}
}

func chartDecimal(t *testing.T, raw string) ledger.Decimal {
	t.Helper()
	value, err := ledger.ParseDecimal(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
