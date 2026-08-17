// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package ledger_test

import (
	"strings"
	"testing"

	"orangecount/internal/ledger"
)

// parseLeg parses one block leg under a header and returns the Dialect.
func parseLeg(t *testing.T, leg string) (ledger.Dialect, []string) {
	t.Helper()
	text := "2026-01-01 * \"me\" \"t\"\n  " + leg + "\n"
	file, bag := ledger.ParseText("m.bean", []byte(text))
	var codes []string
	for _, d := range bag.All() {
		codes = append(codes, d.Code)
	}
	for _, d := range file.Directives {
		if dd, ok := d.(ledger.Dialect); ok {
			return dd, codes
		}
	}
	return ledger.Dialect{}, codes
}

// TestInvestmentLegGrammar locks the parsed shapes of every investment leg
// form.
func TestInvestmentLegGrammar(t *testing.T) {
	cases := []struct {
		name string
		leg  string
		want func(d ledger.Dialect) bool
	}{
		{"explicit quantity buy", "500 STK {31.12 CNY} @Assets:Cash -> @Assets:Holding", func(d ledger.Dialect) bool {
			return d.HasQuantity && d.Quantity.Raw == "500" && d.Security == "STK" && d.Amount.Raw == "" && d.Cost != nil && d.Price == nil && d.FeeAmount.Raw == ""
		}},
		{"auto quantity buy", "1000 CNY FUND {1.50 CNY} @Assets:Cash -> @Assets:Fund", func(d ledger.Dialect) bool {
			return !d.HasQuantity && d.Amount.Raw == "1000" && d.Currency == "CNY" && d.Security == "FUND" && d.Cost != nil
		}},
		{"explicit cash sell", "47212 CNY 2200 STK {} @ 21.46 CNY @Assets:Holding -> @Assets:Cash -> @Income:Gains 手续费 55.23 CNY @Expenses:Fees", func(d ledger.Dialect) bool {
			return d.Amount.Raw == "47212" && d.Quantity.Raw == "2200" && d.Price != nil && d.Price.Amount.Currency == "CNY" && d.GainRef == "Income:Gains" && d.FeeAmount.Raw == "55.23" && d.FeeCurrency == "CNY" && d.FeeRef == "Expenses:Fees"
		}},
		{"elided cash sell", "2000 STK {} @12.60 CNY @Assets:Holding -> @Assets:Cash 手续费 6.39 CNY @Expenses:Fees", func(d ledger.Dialect) bool {
			return d.Amount.Raw == "" && d.Quantity.Raw == "2000" && d.Price != nil && d.GainRef == "" && d.FeeAmount.Raw == "6.39"
		}},
		{"bonus share", "240 STK {37.62 CNY} @Income:Gains -> @Assets:Holding", func(d ledger.Dialect) bool {
			return d.SourceRef == "Income:Gains" && d.DestRef == "Assets:Holding" && d.Price == nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, codes := parseLeg(t, tc.leg)
			if len(codes) > 0 {
				t.Fatalf("unexpected diagnostics: %v", codes)
			}
			if !tc.want(d) {
				t.Fatalf("shape mismatch: %+v", d)
			}
		})
	}
}

// TestInvestmentLegGrammarErrors locks the parse diagnostics of malformed
// investment legs.
func TestInvestmentLegGrammarErrors(t *testing.T) {
	cases := []struct {
		name string
		leg  string
		code string
	}{
		{"price not a number reads as endpoint", "10 STK {} @ abc @A -> @B", "E-DIALECT-ARROW"},
		{"fee amount negative", "10 STK {1.00 CNY} @A -> @B 手续费 -1.00 CNY @Expenses:Fees", "E-DIALECT-AMOUNT"},
		{"fee missing currency", "10 STK {1.00 CNY} @A -> @B 手续费 1.00 @Expenses:Fees", "E-DIALECT-CURRENCY"},
		{"fee missing endpoint", "10 STK {1.00 CNY} @A -> @B 手续费 1.00 CNY", "E-DIALECT-SYNTAX"},
		{"gain endpoint malformed", "99 CNY 10 STK {} @ 1.00 CNY @A -> @B -> nowhere", "E-DIALECT-SYNTAX"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, codes := parseLeg(t, tc.leg)
			found := false
			for _, c := range codes {
				if strings.Contains(c, tc.code) {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %s, got %v", tc.code, codes)
			}
		})
	}
}
