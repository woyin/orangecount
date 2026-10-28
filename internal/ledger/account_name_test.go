// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package ledger

import "testing"

func TestValidAccountNameMirrorsUpstreamLexer(t *testing.T) {
	valid := []string{
		"Assets:Cash",
		"Assets:Wallet:微信",
		"Expenses:Living:吃的:一日三餐",
		"Liabilities:CreditCard:招行",
		"Assets:Bank:工银亚洲1350",
		"Equity:Opening-Balances",
		"Assets:2024:Q1",
		// Upstream ACCOUNTNAME may start with a digit, so this is valid there
		// too; the first segment keeps the [A-Z] rule.
		"Assets:1Cash",
	}
	for _, name := range valid {
		if !ValidAccountName(name) {
			t.Errorf("upstream-valid account rejected: %q", name)
		}
	}
	invalid := []string{
		"Assets",
		"",
		":Cash",
		"Assets:",
		"Assets::Cash",
		"assets:Cash",
		"Assets:cash",
		"Assets:Cash_Money",
		"Assets:Cash Money",
		"1Assets:Cash",
	}
	for _, name := range invalid {
		if ValidAccountName(name) {
			t.Errorf("invalid account accepted: %q", name)
		}
	}
}
