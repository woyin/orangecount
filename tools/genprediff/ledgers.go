// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"fmt"
	"math/rand"
	"strings"
)

const ledgerHeader = `option "operating_currency" "USD"

2020-01-01 open Assets:Cash USD
2020-01-01 open Assets:Bank USD
2020-01-01 open Assets:Broker USD
2020-01-01 open Assets:Broker:Stock
2020-01-01 open Expenses:Food USD
2020-01-01 open Expenses:Travel USD
2020-01-01 open Expenses:Fees USD
2020-01-01 open Income:Salary USD
2020-01-01 open Equity:Opening USD
2020-01-01 open Liabilities:Card USD
`

func plainLedger() string {
	return ledgerHeader + `
2020-01-02 * "Opening"
  Assets:Cash   1000.00 USD
  Equity:Opening

2020-01-03 * "Lunch"
  Expenses:Food  12.50 USD
  Assets:Cash   -12.50 USD
`
}

func multicurrencyLedger() string {
	return ledgerHeader + `2020-01-01 open Assets:Euro EUR
2020-01-01 open Expenses:Euro EUR

2020-01-02 * "USD opening"
  Assets:Cash  100.00 USD
  Equity:Opening  -100.00 USD

2020-01-03 * "EUR opening"
  Assets:Euro  200.00 EUR
  Equity:Opening  -200.00 EUR

2020-01-04 * "Mixed"
  Expenses:Euro  10.00 EUR
  Assets:Euro   -10.00 EUR
`
}

func costsLedger() string {
	return ledgerHeader + `
2020-01-05 * "Buy shares"
  Assets:Broker:Stock  10 AAA {12.00 USD}
  Assets:Broker  -120.00 USD

2020-01-06 * "Sell shares"
  Assets:Broker:Stock  -4 AAA {12.00 USD} @ 15.00 USD
  Assets:Broker  60.00 USD
  Income:Salary  -12.00 USD
`
}

func balanceLedger() string {
	return ledgerHeader + `
2020-01-02 * "Opening"
  Assets:Cash  500.00 USD
  Equity:Opening  -500.00 USD

2020-01-31 balance Assets:Cash  500.00 USD

2020-02-01 * "Spend"
  Expenses:Food  25.00 USD
  Assets:Cash  -25.00 USD

2020-02-29 balance Assets:Cash  475.00 USD
`
}

func metadataLedger() string {
	return ledgerHeader + `
2020-01-02 * "Tagged" #food ^receipt
  category: "eating"
  Assets:Cash  -9.00 USD
  Expenses:Food  9.00 USD
    note: "single"
`
}

func flagsLedger() string {
	return ledgerHeader + `
2020-01-02 ! "Pending"
  Assets:Cash  -3.00 USD
  Expenses:Food  3.00 USD

2020-01-03 * "Cleared"
  Assets:Cash  -4.00 USD
  Expenses:Food  4.00 USD
`
}

func padLedger() string {
	return ledgerHeader + `
2020-01-31 pad Assets:Cash Equity:Opening

2020-02-01 balance Assets:Cash  0.00 USD

2020-02-02 * "Spend"
  Expenses:Food  15.00 USD
  Assets:Cash  -15.00 USD
`
}

func unbalancedLedger() string {
	return ledgerHeader + `
2020-01-02 * "Broken"
  Assets:Cash  -5.00 USD
  Expenses:Food  6.00 USD
`
}

func closedLedger() string {
	return ledgerHeader + `2020-06-01 close Assets:Cash

2020-07-01 * "After close"
  Assets:Cash  -1.00 USD
  Expenses:Food  1.00 USD
`
}

func pricesLedger() string {
	// The equity leg needs an account that accepts AAA; the shared header opens
	// Equity:Opening for USD only, which would make this case invalid and hide
	// the price history from the differential.
	return ledgerHeader + `2020-01-01 open Equity:Prices AAA

2020-01-02 price AAA 12.00 USD
2020-01-03 price AAA 13.50 USD

2020-01-04 * "Holding"
  Assets:Broker:Stock  3 AAA
  Equity:Prices
`
}

// randomLedger builds a seeded pseudo-random but fully deterministic ledger so
// the generator explores combinations the hand-written cases do not cover.
func randomLedger(random *rand.Rand, index int) fixtureCase {
	accounts := []string{"Expenses:Food", "Expenses:Travel", "Expenses:Fees"}
	var builder strings.Builder
	builder.WriteString(ledgerHeader)
	builder.WriteString("\n2020-01-02 * \"Opening\"\n  Assets:Cash  1000.00 USD\n  Equity:Opening  -1000.00 USD\n")
	transactions := 3 + random.Intn(5)
	for i := 0; i < transactions; i++ {
		account := accounts[random.Intn(len(accounts))]
		amount := random.Intn(9000) + 100
		day := 3 + random.Intn(25)
		fmt.Fprintf(&builder, "\n2020-01-%02d * \"Generated %d\"\n  %s  %d.%02d USD\n  Assets:Cash  -%d.%02d USD\n",
			day, i, account, amount/100, amount%100, amount/100, amount%100)
	}
	return fixtureCase{ID: fmt.Sprintf("random-%02d", index), Ledger: builder.String()}
}
