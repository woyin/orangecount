// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dialect_test

import (
	"fmt"
	"strings"
	"testing"
)

// exoticV3Ledger drives the dialectize classifiers with shapes the main
// fixtures do not cover: several metadata value kinds on an eligible block,
// a fee buy, and a multi-currency split. Every entry balances so evaluation
// stays valid; which shapes the filter can express is the behavior under test.
const exoticV3Ledger = `2000-01-01 open Assets:WeChat USD
2000-01-01 open Assets:Cash USD,CNY
2000-01-01 open Assets:Broker:AAPL AAPL
2000-01-01 open Assets:Broker:Cash USD
2000-01-01 open Expenses:Food USD,CNY
2000-01-01 open Expenses:Fees USD

2026-08-12 * "美团" "带全类元数据"
  receipt_date: 2026-08-12
  attachment: "receipt.pdf"
  Assets:WeChat -28 USD
  Expenses:Food 28 USD

2026-08-13 * "买入" "带手续费"
  Assets:Broker:AAPL 10 AAPL {101.00 USD}
  Assets:Broker:Cash -1011 USD
  Expenses:Fees 1 USD

2026-08-15 * "多币种" "拆分"
  Assets:Cash -30 USD
  Assets:Cash -20 CNY
  Expenses:Food 30 USD
  Expenses:Food 20 CNY

2026-08-16 * "余额式" "插值"
  Assets:Cash -7 USD
  Expenses:Food
`

// TestDialectizeExoticShapesPreservesBalances pins that none of the exotic
// shapes corrupt balances, and that the convertible shapes convert while the
// over-specified ones stay plain v3.
func TestDialectizeExoticShapesPreservesBalances(t *testing.T) {
	original := writeFile(t, "exotic.bean", exoticV3Ledger)
	dialectVersion := dialectizeFile(t, original)

	before := balancesOf(t, original)
	after := balancesOf(t, dialectVersion)
	if fmt.Sprint(before) != fmt.Sprint(after) {
		t.Fatalf("balances diverged:\nbefore=%v\nafter=%v", before, after)
	}
	out := readFile(t, dialectVersion)
	// Metadata rides along on the compiled block form.
	if !strings.Contains(out, "28 USD @Assets:WeChat -> @Expenses:Food") {
		t.Fatalf("metadata transaction not dialectized to block form:\n%s", out)
	}
	// A fee buy takes the fee dialect form.
	if !strings.Contains(out, "手续费 1 USD @Expenses:Fees") {
		t.Fatalf("fee buy not dialectized to fee form:\n%s", out)
	}
	// The four-leg multi-currency split exceeds the dialect's expressive
	// power and must stay plain v3.
	if !strings.Contains(out, "  Assets:Cash -20 CNY\n") {
		t.Fatalf("multi-currency split must stay plain v3:\n%s", out)
	}
	// The elided-leg interpolation has no dialect spelling either.
	if !strings.Contains(out, "  Expenses:Food\n") {
		t.Fatalf("elided-leg transaction must stay plain v3:\n%s", out)
	}
}

// TestDialectizeExportFixpointOnExoticShapes extends ADR-0045's fixpoint
// property to the exotic corpus: exporting, dialectizing that export, and
// exporting again is byte-stable.
func TestDialectizeExportFixpointOnExoticShapes(t *testing.T) {
	original := writeFile(t, "exotic2.bean", exoticV3Ledger)
	first := exportFile(t, dialectizeFile(t, original))
	dialectized := dialectizeFile(t, writeFile(t, "exotic3.bean", readFile(t, first)))
	second := exportFile(t, dialectized)
	if readFile(t, first) != readFile(t, second) {
		t.Fatalf("round trip is not a fixpoint:\n--- first ---\n%s\n--- second ---\n%s", readFile(t, first), readFile(t, second))
	}
}
