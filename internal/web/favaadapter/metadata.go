// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

import (
	"sort"
	"strings"

	"orangecount/internal/ledger"
)

// MetadataProjection is the second P3 phase-1 projection: a deterministic
// metadata view over evaluated balances and price quotes. It is an
// independent Go mapping of parts of the Fava "ledger_data"/"commodities"
// contract (docs/fava-contract-map.md rows "ledger_data" (currencies,
// precisions) and "commodities"); the frontend readers consume sorted arrays.
//
// It is presentation-only: it never converts a currency, never reads source
// files, and never changes any ledger amount. Every emitted value is either
// a sorted string slice or an exact ledger.Decimal marshalled as its exact
// string (never a float).
type MetadataProjection struct {
	// Currencies is the sorted, deduplicated commodity list derived from
	// balances, positions, and price quotes.
	Currencies []string `json:"currencies"`
	// Commodities is the sorted per-commodity summary. Accounts with a direct
	// balance in the currency and total (aggregate) scene per currency are
	// deterministic across snapshots.
	Commodities []CommoditySummary `json:"commodities"`
	// PricePairs is the sorted list of base/quote pairs that have at least one
	// exact price quote in the evaluation.
	PricePairs []PricePair `json:"price_pairs"`
	// AccountTotal is the total balance of the given root account in its
	// natural currencies, exact values. This is the "one simple report
	// projection" required by P3 phase 1; it exercises aggregate ordering and
	// exact math without changing accounting semantics.
	AccountTotal AccountTotal `json:"account_total"`
}

// CommoditySummary summarizes where and in what exact amounts a commodity
// appears. Owned is the direct balance in accounts holding the commodity;
// Aggregate is the same total recomputed at every ancestor level, so it is
// always equal to Owned for a commodity. Sorted by account then currency.
type CommoditySummary struct {
	Currency  string `json:"currency"`
	Account   string `json:"account"`
	Owned     string `json:"owned"`
	Aggregate string `json:"aggregate"`
}

// PricePair is one base/quote pair with price-map presence. Only pairs with
// at least one exact quote are emitted.
type PricePair struct {
	Base  string `json:"base"`
	Quote string `json:"quote"`
}

// AccountTotal is a deterministic, exact per-currency total for one root
// account subtree (or the whole ledger when root is empty). The root is the
// presentation anchor; the balances themselves are never modified.
type AccountTotal struct {
	Root   string           `json:"root"`
	Totals []CurrencyAmount `json:"totals"`
}

// CurrencyAmount is an exact amount for one currency inside an AccountTotal.
type CurrencyAmount struct {
	Currency string         `json:"currency"`
	Amount   ledger.Decimal `json:"amount"`
}

// MetadataOptions are the projection inputs.
type MetadataOptions struct {
	// Evaluation is the cloned immutable evaluation (never mutated here).
	Evaluation ledger.Evaluation
	// Root filters the account total to one root account subtree. Empty means
	// the whole ledger.
	Root string
	// BaseURL is reserved for phase-2 wiring; unused by the pure projection.
	BaseURL string
}

// MetadataProjectionOptions builds the projection. It is pure and
// deterministic: same inputs, same JSON bytes.
func MetadataProjectionOptions(opts MetadataOptions) MetadataProjection {
	proj := MetadataProjection{
		Currencies: collectCurrencies(opts.Evaluation),
		PricePairs: collectPricePairs(opts.Evaluation),
	}
	proj.Commodities = collectCommoditySummaries(opts.Evaluation)
	proj.AccountTotal = projectAccountTotal(opts.Evaluation, opts.Root)
	return proj
}

// collectPricePairs returns sorted base/quote pairs that have at least one
// exact quote. Sorted by base then quote for deterministic output.
func collectPricePairs(evaluation ledger.Evaluation) []PricePair {
	set := map[string]bool{}
	for base, quotes := range evaluation.Prices {
		if len(quotes) == 0 {
			continue
		}
		for _, quote := range quotes {
			if quote.Currency == "" {
				continue
			}
			key := base + "\x00" + quote.Currency
			set[key] = true
		}
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]PricePair, 0, len(keys))
	for _, key := range keys {
		parts := strings.SplitN(key, "\x00", 2)
		pairs = append(pairs, PricePair{Base: parts[0], Quote: parts[1]})
	}
	return pairs
}

// collectCommoditySummaries groups exact balances by currency and account,
// then sorts. Aggregate uses the same exact decimals as Owned; the two are
// separate fields so the frontend can render "natural units" per account
// without ambiguity. No float arithmetic occurs.
func collectCommoditySummaries(evaluation ledger.Evaluation) []CommoditySummary {
	type key struct {
		currency string
		account  string
	}
	summary := map[key]ledger.Decimal{}
	for name, state := range evaluation.Accounts {
		for currency, balance := range state.Balances {
			summary[key{currency: currency, account: name}] = balance
		}
	}
	keys := make([]key, 0, len(summary))
	for k := range summary {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].currency != keys[j].currency {
			return keys[i].currency < keys[j].currency
		}
		return keys[i].account < keys[j].account
	})
	result := make([]CommoditySummary, 0, len(keys))
	for _, k := range keys {
		amount := summary[k]
		result = append(result, CommoditySummary{
			Currency:  k.currency,
			Account:   k.account,
			Owned:     amount.String(),
			Aggregate: amount.String(),
		})
	}
	return result
}

// projectAccountTotal sums exact balances over the root-subtree accounts (or
// the whole ledger) grouped by currency, then sorts by currency. The
// presentation anchor (root) is never part of arithmetic; a missing root is
// the whole ledger.
func projectAccountTotal(evaluation ledger.Evaluation, root string) AccountTotal {
	totals := map[string]ledger.Decimal{}
	for name, state := range evaluation.Accounts {
		if root != "" && !strings.HasPrefix(name, root+":") && name != root {
			continue
		}
		for currency, balance := range state.Balances {
			totals[currency] = totals[currency].Add(balance)
		}
	}
	currencies := make([]string, 0, len(totals))
	for currency := range totals {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	amounts := make([]CurrencyAmount, 0, len(currencies))
	for _, currency := range currencies {
		amounts = append(amounts, CurrencyAmount{Currency: currency, Amount: totals[currency]})
	}
	total := AccountTotal{Root: root, Totals: amounts}
	return total
}
