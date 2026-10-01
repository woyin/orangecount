// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package report

import (
	"orangecount/internal/ledger"
	"orangecount/internal/source"
	"sort"
)

// QuoteUse identifies the local quote actually used, including an inverse rate.
// All arithmetic remains exact; an empty lookup never manufactures a zero.
type QuoteUse struct {
	Base    string           `json:"base"`
	Quote   string           `json:"quote"`
	Date    string           `json:"date"`
	Rate    PresentedDecimal `json:"rate"`
	Inverse bool             `json:"inverse"`
	Source  source.Span      `json:"source"`
}

type MissingValuation struct {
	Commodity string `json:"commodity"`
	Reason    string `json:"reason"`
}

type ValuationStatus struct {
	Mode             string             `json:"mode"`
	TargetCurrency   string             `json:"target_currency"`
	AsOf             string             `json:"as_of"`
	Completeness     string             `json:"completeness"`
	CoveredItemCount int                `json:"covered_item_count"`
	MissingItemCount int                `json:"missing_item_count"`
	Missing          []MissingValuation `json:"missing"`
	Quotes           []QuoteUse         `json:"quotes"`
}

// AccountValuation accompanies, rather than replaces, native balance columns.
// CompleteTotal is explicitly null for partial/unavailable/native/empty views.
type AccountValuation struct {
	ValuationStatus
	ValuedPortion *PresentedDecimal `json:"valued_portion"`
	CompleteTotal *PresentedDecimal `json:"complete_total"`
}

// PriceAt selects a matching local pair at or before asOf. Direct quotes take
// precedence; otherwise a nonzero inverse quote is used. No cross-rate or
// network lookup is implicit. Equal-date quotes use the last source-order item.
func PriceAt(e ledger.Evaluation, base, quote, asOf string) (ledger.Decimal, QuoteUse, bool) {
	find := func(from, to string) (ledger.PriceQuote, bool) {
		var selected ledger.PriceQuote
		found := false
		for _, candidate := range e.Prices[from] {
			if candidate.Currency != to || (asOf != "" && candidate.Date.Raw > asOf) {
				continue
			}
			if !found || candidate.Date.Raw >= selected.Date.Raw {
				selected, found = candidate, true
			}
		}
		return selected, found
	}
	if selected, ok := find(base, quote); ok {
		return selected.Amount, QuoteUse{Base: base, Quote: quote, Date: selected.Date.Raw, Rate: FormatDecimal(selected.Amount), Source: selected.Span}, true
	}
	if selected, ok := find(quote, base); ok && !selected.Amount.IsZero() {
		one, _ := ledger.ParseDecimal("1")
		rate := one.Quo(selected.Amount)
		return rate, QuoteUse{Base: base, Quote: quote, Date: selected.Date.Raw, Rate: FormatDecimal(rate), Inverse: true, Source: selected.Span}, true
	}
	return ledger.Zero(), QuoteUse{}, false
}

func valueBalances(e ledger.Evaluation, balances map[string]string, active bool, currency, asOf string) *AccountValuation {
	result := &AccountValuation{ValuationStatus: ValuationStatus{Mode: "native", TargetCurrency: currency, AsOf: asOf, Completeness: "empty", Missing: []MissingValuation{}, Quotes: []QuoteUse{}}}
	if currency != "" {
		result.Mode = "converted"
	}
	if !active {
		return result
	}
	result.Completeness = "complete"
	keys := make([]string, 0, len(balances))
	for key := range balances {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	portion := ledger.Zero()
	for _, commodity := range keys {
		amount, err := ledger.ParseDecimal(balances[commodity])
		if err != nil {
			panic("invalid exact balance in immutable projection")
		}
		if amount.IsZero() {
			continue
		}
		if currency == "" || commodity == currency {
			result.CoveredItemCount++
			if currency != "" {
				portion = portion.Add(amount)
			}
			continue
		}
		if rate, used, ok := PriceAt(e, commodity, currency, asOf); ok {
			portion = portion.Add(amount.Mul(rate))
			result.CoveredItemCount++
			result.Quotes = append(result.Quotes, used)
		} else {
			result.MissingItemCount++
			result.Missing = append(result.Missing, MissingValuation{Commodity: commodity, Reason: "missing-local-quote"})
		}
	}
	if result.MissingItemCount > 0 {
		result.Completeness = "unavailable"
		if result.CoveredItemCount > 0 {
			result.Completeness = "partial"
		}
	}
	if currency != "" && (result.CoveredItemCount > 0 || result.MissingItemCount == 0) {
		formatted := FormatDecimal(portion)
		result.ValuedPortion = &formatted
		if result.MissingItemCount == 0 {
			result.CompleteTotal = &formatted
		}
	}
	return result
}

func annotateStatement(e ledger.Evaluation, nodes []AccountTreeNode, activity map[string]bool, currency, asOf string) ValuationStatus {
	summary := ValuationStatus{Mode: "native", TargetCurrency: currency, AsOf: asOf, Completeness: "empty", Missing: []MissingValuation{}, Quotes: []QuoteUse{}}
	if currency != "" {
		summary.Mode = "converted"
	}
	hasActivity := false
	seenMissing := map[string]bool{}
	seenQuotes := map[string]bool{}
	var visit func(*AccountTreeNode) bool
	visit = func(node *AccountTreeNode) bool {
		active := activity[node.FullName]
		node.DirectValuation = valueBalances(e, node.DirectBalances, active, currency, asOf)
		direct := node.DirectValuation
		hasActivity = hasActivity || active
		summary.CoveredItemCount += direct.CoveredItemCount
		summary.MissingItemCount += direct.MissingItemCount
		for _, missing := range direct.Missing {
			if !seenMissing[missing.Commodity] {
				summary.Missing = append(summary.Missing, missing)
				seenMissing[missing.Commodity] = true
			}
		}
		for _, used := range direct.Quotes {
			key := used.Base + "\x00" + used.Quote + "\x00" + used.Date
			if !seenQuotes[key] {
				summary.Quotes = append(summary.Quotes, used)
				seenQuotes[key] = true
			}
		}
		for i := range node.Children {
			active = visit(&node.Children[i]) || active
		}
		node.SubtreeValuation = valueBalances(e, node.SubtreeBalances, active, currency, asOf)
		return active
	}
	for i := range nodes {
		visit(&nodes[i])
	}
	if hasActivity {
		summary.Completeness = "complete"
	}
	if summary.MissingItemCount > 0 {
		summary.Completeness = "unavailable"
		if summary.CoveredItemCount > 0 {
			summary.Completeness = "partial"
		}
	}
	sort.Slice(summary.Missing, func(i, j int) bool { return summary.Missing[i].Commodity < summary.Missing[j].Commodity })
	return summary
}
