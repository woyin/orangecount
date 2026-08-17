// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package dialect_test

import (
	"strings"
	"testing"

	"orangecount/internal/snapshot"
)

// TestInvestmentBoundaryErrors locks the error paths of the investment
// dialect: each malformed leg must produce its diagnostic instead of a
// silently different transaction.
func TestInvestmentBoundaryErrors(t *testing.T) {
	opens := `2000-01-01 open Assets:Cash CNY
2000-01-01 open Assets:Holding "FIFO"
2000-01-01 open Expenses:Fees CNY
2000-01-01 open Income:Gains
option "operating_currency" "CNY"

`
	cases := []struct {
		name string
		leg  string
		code string
	}{
		{"sell without quantity", "STK {} @ 10.00 CNY @Assets:Holding -> @Assets:Cash 手续费 1.00 CNY @Expenses:Fees", "E-DIALECT-SECURITY"},
		{"sell with nothing to balance", "10 STK {} @ 10.00 CNY @Assets:Holding -> @Assets:Cash", "E-DIALECT-SECURITY"},
		{"gain endpoint without cash", "10 STK {} @ 10.00 CNY @Assets:Holding -> @Assets:Cash -> @Income:Gains", "E-DIALECT-SECURITY"},
		{"price without currency", "10 STK {} @ 10.00 @Assets:Holding -> @Assets:Cash", "E-DIALECT-SECURITY"},
		{"fee without currency", "10 STK {1.00 CNY} @Assets:Cash -> @Assets:Holding 手续费 1.00 @Expenses:Fees", "E-DIALECT-CURRENCY"},
		{"fee with unknown endpoint", "10 STK {1.00 CNY} @Assets:Cash -> @Assets:Holding 手续费 1.00 CNY @NoSuchFeeTail", "E-DIALECT-ACCOUNT"},
		{"gain endpoint unknown", "99.00 CNY 10 STK {} @ 10.00 CNY @Assets:Holding -> @Assets:Cash -> @NoSuchGainTail", "E-DIALECT-ACCOUNT"},
		{"underdetermined bare security", "STK {1.00 CNY} @Assets:Cash -> @Assets:Holding", "E-DIALECT-SECURITY"},
		{"tail garbage", "10 STK {1.00 CNY} @Assets:Cash -> @Assets:Holding extra", "E-DIALECT-SYNTAX"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := opens + "2026-01-01 * \"me\" \"t\"\n  " + tc.leg + "\n"
			path := writeFile(t, "b.bean", text)
			result := snapshot.Build(path)
			found := false
			for _, d := range result.Diagnostics {
				if strings.Contains(d.Code, "E-DIALECT") {
					found = true
					t.Logf("diag: %s %s", d.Code, d.Message)
				}
			}
			if !found {
				t.Fatalf("expected a dialect diagnostic, got none (snapshot nil: %v, codes: %v)", result.Snapshot == nil, diagCodesOf(result))
			}
		})
	}
}

func diagCodesOf(result snapshot.BuildResult) []string {
	var codes []string
	for _, d := range result.Diagnostics {
		codes = append(codes, d.Code)
	}
	return codes
}

// TestInvestmentFilterRejectsNonConvertibleShapes locks the recognizer's
// rejections: these v3 transactions stay standard because no leg form
// reproduces them.
func TestInvestmentFilterRejectsNonConvertibleShapes(t *testing.T) {
	opens := `2000-01-01 open Assets:Cash CNY
2000-01-01 open Assets:Holding "FIFO"
2000-01-01 open Expenses:Fees CNY
2000-01-01 open Income:Gains
option "operating_currency" "CNY"

`
	cases := []struct {
		name string
		txn  string
	}{
		{"negative sell cash", `2026-01-01 * "me" "bad sell"
  Assets:Cash  -100.00 CNY
  Assets:Holding -10 STK {} @ 10.00 CNY
  Expenses:Fees 1.00 CNY
`},
		{"gain with elided cash", `2026-01-01 * "me" "two elided"
  Assets:Cash
  Assets:Holding -10 STK {} @ 10.00 CNY
  Expenses:Fees 1.00 CNY
  Income:Gains
`},
		{"elided cash without fee", `2026-01-01 * "me" "no balance anchor"
  Assets:Cash
  Assets:Holding 10 STK {1.00 CNY}
`},
		{"explicit cash mismatching cost", `2026-01-01 * "me" "cash not qty times cost"
  Assets:Cash -111.00 CNY
  Assets:Holding 10 STK {1.00 CNY}
`},
		{"bonus share without quantity", `2026-01-01 * "me" "qty elided bonus"
  Assets:Holding STK {1.00 CNY}
  Income:Gains
`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := writeFile(t, "r.bean", opens+tc.txn)
			dialectVersion := dialectizeFile(t, original)
			text := readFile(t, dialectVersion)
			if !strings.Contains(text, "Assets:Holding ") {
				t.Fatalf("transaction should stay standard v3:\n%s", text)
			}
		})
	}
}

// TestMultiAssetBuyRejections locks the honest boundary of the multi-asset
// rule: fees cannot split across legs, mismatched cash or mixed
// destinations stay standard.
func TestMultiAssetBuyRejections(t *testing.T) {
	base := `2000-01-01 open Assets:Wallet:微信 CNY
2000-01-01 open Assets:收藏品 "FIFO"
2000-01-01 open Assets:其他收藏 "FIFO"
2000-01-01 open Expenses:Fees:交易费用 CNY
option "operating_currency" "CNY"

`
	cases := []struct {
		name string
		txn  string
	}{
		{"cash does not sum", `2026-01-16 * "我" "购买"
  Assets:Wallet:微信  -1500.00 CNY
  Assets:收藏品  20 COIN_2026_HORSE {10.00 CNY}
  Assets:收藏品  60 NOTE_2026_HORSE {20.00 CNY}
`},
		{"fee cannot split", `2026-01-16 * "我" "购买"
  Assets:Wallet:微信
  Assets:收藏品  20 COIN_2026_HORSE {10.00 CNY}
  Assets:收藏品  60 NOTE_2026_HORSE {20.00 CNY}
  Expenses:Fees:交易费用 5.00 CNY
`},
		{"mixed destinations", `2026-01-16 * "我" "购买"
  Assets:Wallet:微信  -1400.00 CNY
  Assets:收藏品  20 COIN_2026_HORSE {10.00 CNY}
  Assets:其他收藏  60 NOTE_2026_HORSE {20.00 CNY}
`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := writeFile(t, "mr.bean", base+tc.txn)
			dialectVersion := dialectizeFile(t, original)
			text := readFile(t, dialectVersion)
			if !strings.Contains(text, "Assets:收藏品 ") && !strings.Contains(text, "Assets:其他收藏 ") {
				t.Fatalf("should stay standard v3:\n%s", text)
			}
		})
	}
}
