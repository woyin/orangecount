// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package report

import (
	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

type ReductionEvent struct {
	ID               string            `json:"id"`
	Date             string            `json:"date"`
	Account          string            `json:"account"`
	Commodity        string            `json:"commodity"`
	Kind             string            `json:"kind"`
	Quantity         PresentedDecimal  `json:"quantity"`
	ConsumedQuantity PresentedDecimal  `json:"consumed_quantity"`
	CostCurrency     *string           `json:"cost_currency"`
	CostBasis        *PresentedDecimal `json:"cost_basis"`
	PriceCurrency    string            `json:"price_currency"`
	Proceeds         *PresentedDecimal `json:"proceeds"`
	ProceedsAtCost   *PresentedDecimal `json:"proceeds_at_cost"`
	Conversion       *QuoteUse         `json:"conversion"`
	GrossGain        *PresentedDecimal `json:"gross_gain"`
	Fees             *PresentedDecimal `json:"fees"`
	NetGain          *PresentedDecimal `json:"net_gain"`
	GrossReason      string            `json:"gross_reason"`
	NetReason        string            `json:"net_reason"`
	Lots             []HoldingLot      `json:"lots"`
	LotsPage         ProjectionPage    `json:"lots_page"`
	MergeLots        []HoldingLot      `json:"merge_lots"`
	MergeLotsPage    ProjectionPage    `json:"merge_lots_page"`
	Source           ProjectionSource  `json:"source"`
	Narration        string            `json:"narration"`
}

type reductionWork struct {
	item                      ReductionEvent
	entryIndex                int
	price                     *ledger.PriceSpec
	quantity, consumed, basis ledger.Decimal
	missingCost, mixedCost    bool
}

func evaluatedTransaction(record ledger.EntryRecord) *ledger.Transaction {
	switch value := record.Directive.(type) {
	case *ledger.Transaction:
		return value
	case ledger.Transaction:
		return &value
	}
	return nil
}

func reductionEvents(e ledger.Evaluation, changes []ledger.BookedInventoryChange, account, scope, from string, paths map[source.FileID]string, origins map[source.Span]bool) []ReductionEvent {
	result := []ReductionEvent{}
	works := map[string]*reductionWork{}
	order := []string{}
	counts := map[int]map[string]bool{}
	merges := map[string][]HoldingLot{}
	for _, change := range changes {
		tx := evaluatedTransaction(e.Entries[change.EntryIndex])
		posting := tx.Postings[change.PostingIndex]
		if change.Position.Units.Sign() >= 0 {
			continue
		}
		identity := any(posting.Span())
		if !posting.Span().Valid() {
			identity = change.PostingIndex
		}
		key := holdingIdentity(change.EntryIndex, change.Account, change.Position.Currency, identity)
		if posting.Flag == "M" {
			for _, consumed := range change.ConsumedLots {
				merges[key] = append(merges[key], holdingLot(change.Account, consumed, paths, origins))
			}
			continue
		}
		if counts[change.EntryIndex] == nil {
			counts[change.EntryIndex] = map[string]bool{}
		}
		counts[change.EntryIndex][key] = true
		if tx.Date.Raw < from || !holdingAccountMatches(change.Account, account, scope) {
			continue
		}
		work := works[key]
		if work == nil {
			work = &reductionWork{entryIndex: change.EntryIndex, price: posting.Price, item: ReductionEvent{ID: key, Date: tx.Date.Raw, Account: change.Account, Commodity: change.Position.Currency, Kind: "reduction", Lots: []HoldingLot{}, MergeLots: []HoldingLot{}, Source: projectionLocation(e.Entries[change.EntryIndex].File, posting.Span()), Narration: tx.Narration}}
			works[key] = work
			order = append(order, key)
		}
		position := change.Position
		position.Units = position.Units.Neg()
		work.quantity = work.quantity.Add(position.Units)
		work.consumed = work.consumed.Add(change.Consumed)
		for _, consumed := range change.ConsumedLots {
			work.item.Lots = append(work.item.Lots, holdingLot(change.Account, consumed, paths, origins))
		}
		if cost := position.Cost; cost != nil {
			if work.item.CostCurrency == nil {
				currency := cost.Currency
				work.item.CostCurrency = &currency
			} else if *work.item.CostCurrency != cost.Currency {
				work.mixedCost = true
			}
			work.basis = work.basis.Add(position.Units.Mul(cost.Number))
		} else {
			work.missingCost = true
		}
	}
	for _, key := range order {
		work := works[key]
		item := &work.item
		item.MergeLots = append(item.MergeLots, merges[key]...)
		item.Quantity, item.ConsumedQuantity = FormatDecimal(work.quantity), FormatDecimal(work.consumed)
		tx := evaluatedTransaction(e.Entries[work.entryIndex])
		transfer := false
		for _, posting := range tx.Postings {
			if posting.Flag != "M" && posting.Units != nil && posting.Units.Currency == item.Commodity && ledger.DecimalFromNumber(posting.Units.Number).Sign() > 0 && (accountRoot(posting.Account) == "Assets" || accountRoot(posting.Account) == "Liabilities") {
				transfer = true
			}
		}
		if transfer {
			item.Kind = "transfer-or-exchange"
		} else if work.consumed.Cmp(work.quantity) < 0 {
			item.Kind = "short-or-mixed"
		}
		if !work.missingCost && !work.mixedCost && work.consumed.Equal(work.quantity) {
			value := FormatDecimal(work.basis)
			item.CostBasis = &value
		}
		if work.price != nil {
			item.PriceCurrency = work.price.Amount.Currency
			proceeds := ledger.DecimalFromNumber(work.price.Amount.Number)
			if !work.price.Total {
				proceeds = proceeds.Mul(work.quantity)
			}
			value := FormatDecimal(proceeds)
			item.Proceeds = &value
		}
		switch {
		case transfer:
			item.GrossReason = "transfer-or-exchange"
		case work.missingCost:
			item.GrossReason = "missing-cost"
		case work.mixedCost:
			item.GrossReason = "mixed-cost-currencies"
		case work.consumed.Cmp(work.quantity) < 0:
			item.GrossReason = "short-position-opening"
		case item.Proceeds == nil:
			item.GrossReason = "missing-sale-price"
		case ledger.DecimalFromNumber(work.price.Amount.Number).Sign() < 0:
			item.GrossReason = "negative-sale-price"
		default:
			proceeds, _ := ledger.ParseDecimal(item.Proceeds.Exact)
			if item.PriceCurrency != *item.CostCurrency {
				if rate, used, ok := PriceAt(e, item.PriceCurrency, *item.CostCurrency, item.Date); ok {
					proceeds = proceeds.Mul(rate)
					item.Conversion = &used
				} else {
					item.GrossReason = "missing-proceeds-quote"
				}
			}
			if item.GrossReason == "" {
				value, gain := FormatDecimal(proceeds), FormatDecimal(proceeds.Sub(work.basis))
				item.ProceedsAtCost, item.GrossGain = &value, &gain
				item.Kind = "sale"
			}
		}
		item.NetReason = item.GrossReason
		if item.GrossGain != nil {
			fees, reason := attributedTradeFees(e, tx, item, len(counts[work.entryIndex]))
			item.NetReason = reason
			if reason == "" {
				gross, _ := ledger.ParseDecimal(item.GrossGain.Exact)
				feeValue, netValue := FormatDecimal(fees), FormatDecimal(gross.Sub(fees))
				item.Fees, item.NetGain = &feeValue, &netValue
			}
		}
		result = append(result, *item)
	}
	return result
}

// An expense becomes an attributed fee only with explicit posting metadata
// trade_fee_for: "Account". A single reduction, fully reconciled cash receipts
// and dated local conversions are required; a label/narration never assigns a fee.
func attributedTradeFees(e ledger.Evaluation, tx *ledger.Transaction, item *ReductionEvent, count int) (ledger.Decimal, string) {
	if count != 1 {
		return ledger.Zero(), "ambiguous-trade-fees"
	}
	fees, cash := ledger.Zero(), ledger.Zero()
	currency := *item.CostCurrency
	for _, posting := range tx.Postings {
		if posting.Flag == "M" || posting.Units == nil || posting.Cost != nil {
			continue
		}
		amount := ledger.DecimalFromNumber(posting.Units.Number)
		if amount.IsZero() {
			continue
		}
		root := accountRoot(posting.Account)
		if root == "Income" {
			continue
		}
		if root != "Assets" && root != "Liabilities" && root != "Expenses" {
			return ledger.Zero(), "unsupported-trade-counterpart"
		}
		if root == "Expenses" {
			assigned := false
			for _, meta := range posting.Meta {
				if meta.Key == "trade_fee_for" && meta.Value.String == item.Account && (meta.Value.Kind == ledger.ValueString || meta.Value.Kind == ledger.ValueAccount) {
					assigned = true
				}
			}
			if !assigned {
				return ledger.Zero(), "unattributed-expenses"
			}
		}
		if posting.Units.Currency != currency {
			if rate, _, ok := PriceAt(e, posting.Units.Currency, currency, item.Date); ok {
				amount = amount.Mul(rate)
			} else {
				return ledger.Zero(), "missing-fee-or-settlement-quote"
			}
		}
		if root == "Expenses" {
			fees = fees.Add(amount)
		} else {
			cash = cash.Add(amount)
		}
	}
	proceeds, _ := ledger.ParseDecimal(item.ProceedsAtCost.Exact)
	if !cash.Equal(proceeds.Sub(fees)) {
		return ledger.Zero(), "proceeds-not-reconciled"
	}
	return fees, ""
}
