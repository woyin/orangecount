// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dialect

import (
	"math/big"
	"strconv"
	"strings"

	"orangecount/internal/ledger"
)

// SerializeTransaction renders a compiled dialect transaction as the
// canonical three-line Beancount v3 block used by export. Date and flag are
// always written; payee appears only when present; tags and links follow the
// narration; both posting amounts are explicit.
func SerializeTransaction(txn *ledger.Transaction) string {
	var head strings.Builder
	head.WriteString(txn.Date.Raw)
	head.WriteString(" ")
	head.WriteString(flagOrDefault(txn.Flag))
	if txn.Payee != "" {
		head.WriteString(" ")
		head.WriteString(strconv.Quote(txn.Payee))
	}
	head.WriteString(" ")
	head.WriteString(strconv.Quote(txn.Narration))
	for _, tag := range txn.Tags {
		head.WriteString(" #")
		head.WriteString(tag)
	}
	for _, link := range txn.Links {
		head.WriteString(" ^")
		head.WriteString(link)
	}
	var block strings.Builder
	block.WriteString(head.String())
	writeMetaLines(&block, txn.Meta)
	for _, posting := range txn.Postings {
		block.WriteString("\n  ")
		block.WriteString(posting.Account)
		block.WriteString(" ")
		if posting.Units != nil {
			block.WriteString(posting.Units.Number.Raw)
			if posting.Units.Currency != "" {
				block.WriteString(" ")
				block.WriteString(posting.Units.Currency)
			}
		}
		if posting.Cost != nil {
			block.WriteString(" ")
			block.WriteString(posting.Cost.Raw)
		}
		if posting.Price != nil {
			if posting.Price.Total {
				block.WriteString(" @@")
			} else {
				block.WriteString(" @")
			}
			block.WriteString(" ")
			block.WriteString(posting.Price.Amount.Number.Raw)
			if posting.Price.Amount.Currency != "" {
				block.WriteString(" ")
				block.WriteString(posting.Price.Amount.Currency)
			}
		}
	}
	return block.String()
}

func flagOrDefault(flag string) string {
	if flag == "" {
		return "*"
	}
	return flag
}

// SerializeDialect renders the dialect shorthand line for dialectize. It is
// the inverse of the parser's dialect grammar; only reparse-safe text may be
// produced (eligible transactions are filtered before this runs).
func SerializeDialect(txn *ledger.Transaction) string {
	if inv := classifyInvestment(txn.Postings); inv != nil {
		if inv.sell() {
			return serializeSellLeg(txn, inv)
		}
		return serializeBuyLeg(txn, inv)
	}
	negatives, positives, ok := splitLegs(txn)
	if !ok || len(txn.Postings) < 2 {
		return ""
	}
	if len(txn.Postings) == 2 && len(negatives) == 1 && len(positives) == 1 && len(txn.Meta) == 0 {
		// The single-line form has no room for metadata; a transaction
		// with metadata takes the block form so nothing is lost.
		return serializeSingleLine(txn, negatives[0], positives[0])
	}
	// Block form: a standard header plus one indented leg per counterparty.
	if len(negatives) != 1 && len(positives) != 1 {
		// Multiple sources and destinations cannot decompose into legs.
		return ""
	}
	var block strings.Builder
	writeBlockHeader(&block, txn)
	writeMetaLines(&block, txn.Meta)
	if len(negatives) == 1 {
		// One source, many destinations: each destination contributes its
		// own amount.
		for _, p := range positives {
			writeLeg(&block, negatives[0], p, p.Units)
		}
	} else {
		// Many sources, one destination: each source contributes its own
		// amount (written as a positive magnitude).
		for _, n := range negatives {
			writeLeg(&block, n, positives[0], absUnits(n.Units))
		}
	}
	return block.String()
}

// splitLegs separates a transaction's postings into negative and positive
// legs. It returns ok=false when any posting lacks plain units.
func splitLegs(txn *ledger.Transaction) (negatives, positives []ledger.Posting, ok bool) {
	for i := range txn.Postings {
		p := &txn.Postings[i]
		if p.Units == nil || p.Units.Number.Rat == nil {
			return nil, nil, false
		}
		if p.Units.Number.Rat.Sign() < 0 {
			negatives = append(negatives, *p)
		} else {
			positives = append(positives, *p)
		}
	}
	return negatives, positives, true
}

// serializeSingleLine renders the compact two-posting dialect line.
func serializeSingleLine(txn *ledger.Transaction, negative, positive ledger.Posting) string {
	var line strings.Builder
	line.WriteString(canonicalDate(txn.Date))
	if txn.Flag == "!" {
		line.WriteString(" !")
	}
	line.WriteString(" ")
	line.WriteString(strings.TrimPrefix(positive.Units.Number.Raw, "+"))
	line.WriteString(" ")
	line.WriteString(positive.Units.Currency)
	line.WriteString(" @")
	line.WriteString(negative.Account)
	line.WriteString(" -> @")
	line.WriteString(positive.Account)
	if txn.Payee != "" {
		line.WriteString(" ")
		line.WriteString(strconv.Quote(txn.Payee))
	}
	line.WriteString(" : ")
	line.WriteString(txn.Narration)
	for _, tag := range txn.Tags {
		line.WriteString(" #")
		line.WriteString(tag)
	}
	for _, link := range txn.Links {
		line.WriteString(" ^")
		line.WriteString(link)
	}
	return line.String()
}

// writeBlockHeader writes the standard v3 header line of a dialect block:
// date, flag, payee, narration, tags, links.
func writeBlockHeader(block *strings.Builder, txn *ledger.Transaction) {
	block.WriteString(canonicalDate(txn.Date))
	block.WriteString(" ")
	block.WriteString(flagOrDefault(txn.Flag))
	if txn.Payee != "" {
		block.WriteString(" ")
		block.WriteString(strconv.Quote(txn.Payee))
	}
	block.WriteString(" ")
	block.WriteString(strconv.Quote(txn.Narration))
	for _, tag := range txn.Tags {
		block.WriteString(" #")
		block.WriteString(tag)
	}
	for _, link := range txn.Links {
		block.WriteString(" ^")
		block.WriteString(link)
	}
}

// writeMetaLines renders transaction metadata as indented v3 pairs after
// the header line.
func writeMetaLines(block *strings.Builder, meta []ledger.Metadata) {
	for _, m := range meta {
		block.WriteString("\n  ")
		block.WriteString(m.Key)
		block.WriteString(": ")
		block.WriteString(metaValueText(m.Value))
	}
}

// metaValueText renders one metadata value: strings quote their content,
// everything else keeps its raw source form.
func metaValueText(v ledger.Value) string {
	if v.Kind == ledger.ValueString {
		return strconv.Quote(v.String)
	}
	if v.Raw != "" {
		return v.Raw
	}
	return strconv.Quote(v.String)
}

// absUnits returns a copy of units with the number's sign normalized to
// positive, so a negative posting renders as a positive dialect amount.
func absUnits(units *ledger.Amount) *ledger.Amount {
	if units == nil {
		return nil
	}
	rat := units.Number.Rat
	if rat == nil {
		return units
	}
	abs := new(big.Rat).Abs(rat)
	return &ledger.Amount{
		Number:   ledger.Number{Raw: ledger.NewDecimal(abs).String(), Rat: abs},
		Currency: units.Currency,
	}
}

// writeLeg appends one indented dialect leg "amount @source -> @destination".
func writeLeg(block *strings.Builder, source, destination ledger.Posting, units *ledger.Amount) {
	block.WriteString("\n  ")
	block.WriteString(strings.TrimPrefix(units.Number.Raw, "+"))
	block.WriteString(" ")
	block.WriteString(units.Currency)
	block.WriteString(" @")
	block.WriteString(source.Account)
	block.WriteString(" -> @")
	block.WriteString(destination.Account)
}

// serializeBuyLeg renders a buy transaction as a single investment dialect
// leg. An explicit-quantity buy becomes "QUANTITY SECURITY {COST}
// @cash -> @securities"; an auto-quantity buy (empty securities quantity)
// becomes "AMOUNT CURRENCY SECURITY {COST} @cash -> @securities" so the
// share count is derivable from the cash side.
func serializeBuyLeg(txn *ledger.Transaction, inv *investmentTxn) string {
	securities := inv.lot()
	if len(inv.securities) > 1 {
		return serializeMultiBuyLeg(txn, inv)
	}
	var block strings.Builder
	writeInvestmentHeader(&block, txn)
	writeMetaLines(&block, txn.Meta)
	block.WriteString("\n  ")
	explicitCash := inv.cash != nil && inv.cash.Units != nil
	switch {
	case inv.gain != nil:
		// Bonus share: the income account is the source.
		block.WriteString(strings.TrimPrefix(securities.Units.Number.Raw, "+"))
		block.WriteString(" ")
		block.WriteString(securities.Units.Currency)
		block.WriteString(" ")
		block.WriteString(securities.Cost.Raw)
		block.WriteString(" @")
		block.WriteString(inv.gain.Account)
		block.WriteString(" -> @")
		block.WriteString(securities.Account)
		return block.String()
	case explicitCash && securities.Units.Number.Raw != "":
		// Explicit cash and quantity (with or without fee): the amount
		// form carries both numbers exactly as written.
		cashAbs := absUnits(inv.cash.Units)
		block.WriteString(cashAbs.Number.Raw)
		block.WriteString(" ")
		block.WriteString(cashAbs.Currency)
		block.WriteString(" ")
		block.WriteString(strings.TrimPrefix(securities.Units.Number.Raw, "+"))
		block.WriteString(" ")
	default:
		if securities.Units.Number.Raw != "" {
			block.WriteString(strings.TrimPrefix(securities.Units.Number.Raw, "+"))
			block.WriteString(" ")
		} else if explicitCash {
			// Auto-quantity: carry the cash amount so the share count stays
			// derivable from quantity × unit cost.
			cashAbs := absUnits(inv.cash.Units)
			block.WriteString(cashAbs.Number.Raw)
			block.WriteString(" ")
			block.WriteString(cashAbs.Currency)
			block.WriteString(" ")
		}
	}
	block.WriteString(securities.Units.Currency)
	block.WriteString(" ")
	block.WriteString(securities.Cost.Raw)
	block.WriteString(" @")
	if inv.cash != nil {
		block.WriteString(inv.cash.Account)
	}
	block.WriteString(" -> @")
	block.WriteString(securities.Account)
	writeFeeSuffix(&block, inv)
	return block.String()
}

// serializeMultiBuyLeg renders a multi-asset buy as parallel derived legs:
// "QUANTITY SECURITY {COST} @cash -> @securities" per lot, all into the
// same account. Each leg derives its own cash side; their sum reproduces
// the original explicit or elided cash posting.
func serializeMultiBuyLeg(txn *ledger.Transaction, inv *investmentTxn) string {
	var block strings.Builder
	writeInvestmentHeader(&block, txn)
	writeMetaLines(&block, txn.Meta)
	for _, s := range inv.securities {
		block.WriteString("\n  ")
		block.WriteString(strings.TrimPrefix(s.Units.Number.Raw, "+"))
		block.WriteString(" ")
		block.WriteString(s.Units.Currency)
		block.WriteString(" ")
		block.WriteString(s.Cost.Raw)
		block.WriteString(" @")
		block.WriteString(inv.cash.Account)
		block.WriteString(" -> @")
		block.WriteString(s.Account)
	}
	return block.String()
}

// serializeSellLeg renders a sale as a single dialect leg: "[CASH CURRENCY]
// QUANTITY SECURITY {COST} @ PRICE CURRENCY @securities -> @cash [-> @gain]
// [手续费 FEE CURRENCY @fee]". The cash amount stays exactly as written
// (gross or net of the fee); an elided cash leg omits it so the residual
// reproduces the original inference.
func serializeSellLeg(txn *ledger.Transaction, inv *investmentTxn) string {
	securities := inv.lot()
	qty := absUnits(securities.Units)
	var block strings.Builder
	writeInvestmentHeader(&block, txn)
	writeMetaLines(&block, txn.Meta)
	block.WriteString("\n  ")
	if inv.cash != nil && inv.cash.Units != nil {
		cashAbs := absUnits(inv.cash.Units)
		block.WriteString(cashAbs.Number.Raw)
		block.WriteString(" ")
		block.WriteString(cashAbs.Currency)
		block.WriteString(" ")
	}
	block.WriteString(qty.Number.Raw)
	block.WriteString(" ")
	block.WriteString(qty.Currency)
	block.WriteString(" ")
	block.WriteString(securities.Cost.Raw)
	block.WriteString(" @ ")
	block.WriteString(securities.Price.Amount.Number.Raw)
	block.WriteString(" ")
	block.WriteString(securities.Price.Amount.Currency)
	block.WriteString(" @")
	block.WriteString(securities.Account)
	block.WriteString(" -> @")
	if inv.cash != nil {
		block.WriteString(inv.cash.Account)
	}
	if inv.gain != nil {
		block.WriteString(" -> @")
		block.WriteString(inv.gain.Account)
	}
	writeFeeSuffix(&block, inv)
	return block.String()
}

// writeInvestmentHeader writes the shared quoted transaction header.
func writeInvestmentHeader(block *strings.Builder, txn *ledger.Transaction) {
	block.WriteString(canonicalDate(txn.Date))
	block.WriteString(" ")
	block.WriteString(flagOrDefault(txn.Flag))
	if txn.Payee != "" {
		block.WriteString(" ")
		block.WriteString(strconv.Quote(txn.Payee))
	}
	block.WriteString(" ")
	block.WriteString(strconv.Quote(txn.Narration))
	for _, tag := range txn.Tags {
		block.WriteString(" #")
		block.WriteString(tag)
	}
	for _, link := range txn.Links {
		block.WriteString(" ^")
		block.WriteString(link)
	}
}

// writeFeeSuffix appends "手续费 AMOUNT CURRENCY @account" when a fee leg
// exists.
func writeFeeSuffix(block *strings.Builder, inv *investmentTxn) {
	if inv.fee == nil {
		return
	}
	feeAbs := absUnits(inv.fee.Units)
	block.WriteString(" 手续费 ")
	block.WriteString(feeAbs.Number.Raw)
	block.WriteString(" ")
	block.WriteString(feeAbs.Currency)
	block.WriteString(" @")
	block.WriteString(inv.fee.Account)
}

func canonicalDate(date ledger.Date) string {
	if date.Raw != "" && date.Valid() {
		return date.Raw
	}
	return ""
}

func positivePosting(txn *ledger.Transaction) *ledger.Posting {
	for i := range txn.Postings {
		units := txn.Postings[i].Units
		if units != nil && units.Number.Rat != nil && units.Number.Rat.Sign() > 0 {
			return &txn.Postings[i]
		}
	}
	return nil
}

func otherPosting(txn *ledger.Transaction, posting *ledger.Posting) *ledger.Posting {
	for i := range txn.Postings {
		if &txn.Postings[i] != posting {
			return &txn.Postings[i]
		}
	}
	return posting
}
