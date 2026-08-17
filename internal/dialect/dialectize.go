// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dialect

import (
	"bytes"
	"math/big"
	"sort"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

// Dialectize rewrites one standard v3 source file into the dialect: every
// transaction that is exactly representable as a dialect line is replaced
// (span-level, byte-preserving for everything else), and every other entry
// stays untouched. The filter never loses information — eligibility requires
// the transaction to survive the round trip bit-for-bit in meaning.
func Dialectize(file *ledger.File) ([]Edit, bool) {
	if file == nil {
		return nil, false
	}
	var edits []Edit
	changed := false
	for _, directive := range file.Directives {
		txn, ok := transactionOf(directive)
		if !ok || !eligibleForDialect(txn) {
			continue
		}
		todos, hasOther := spanTodos(file.Source, txn.At)
		if hasOther {
			// Free-text comments have no metadata form; converting the
			// block would delete them, so it stays standard.
			continue
		}
		if len(todos) > 0 {
			if metaHasKey(txn.Meta, "todo") {
				continue
			}
			cp := *txn
			cp.Meta = append(append([]ledger.Metadata(nil), txn.Meta...),
				ledger.Metadata{Key: "todo", Value: ledger.Value{Kind: ledger.ValueString, String: strings.Join(todos, "; ")}})
			txn = &cp
		}
		line := SerializeDialect(txn)
		if line == "" {
			continue
		}
		edits = append(edits, Edit{Span: txn.At, Text: line})
		changed = true
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].Span.Start < edits[j].Span.Start })
	return edits, changed
}

func transactionOf(directive ledger.Directive) (*ledger.Transaction, bool) {
	switch d := directive.(type) {
	case *ledger.Transaction:
		return d, true
	case ledger.Transaction:
		return &d, true
	default:
		return nil, false
	}
}

// eligibleForDialect applies the ADR-0045 conversion filter: two or more
// postings forming one source with many destinations or vice versa, no
// cost, price, posting flag, or metadata on plain legs, no transaction
// metadata, flag * or !, and a non-empty narration and payee. Investment
// buys and sells are recognized structurally and always convert: their
// headers quote the narration, so lexer-significant characters such as
// parentheses are safe. Only the bare single-line form requires
// reparse-safe text.
func eligibleForDialect(txn *ledger.Transaction) bool {
	if !txn.Date.Valid() || len(txn.Postings) < 2 {
		return false
	}
	if txn.Flag != "" && txn.Flag != "*" && txn.Flag != "!" {
		return false
	}
	if txn.Narration == "" && txn.Payee == "" {
		// An empty narration would be re-materialized as the 消费 default.
		return false
	}
	if classifyInvestment(txn.Postings) != nil {
		return true
	}
	return plainBlockEligible(txn)
}

// plainBlockEligible applies the plain-block filter: balanced single-
// currency plain legs, one side singleton or exact amount pairing, and the
// single-line form's reparse-safe text.
func plainBlockEligible(txn *ledger.Transaction) bool {
	negative, positive, balanced := plainLegShape(txn.Postings)
	if !balanced || negative == 0 || positive == 0 {
		return false
	}
	if negative > 1 && positive > 1 {
		return splitAmountsPairExactly(txn)
	}
	if negative+positive == 2 {
		// The single-line form leaves narration and payee unquoted.
		return textIsReparseSafe(txn.Narration) && textIsReparseSafe(txn.Payee)
	}
	// Block headers quote narration and payee.
	return true
}

// splitAmountsPairExactly reports whether every source posting of a
// both-sides-split transaction pairs with a destination of the exact same
// amount, so each leg stays one faithful record.
func splitAmountsPairExactly(txn *ledger.Transaction) bool {
	negatives, positives, ok := splitLegs(txn)
	if !ok {
		return false
	}
	_, matched := matchLegPairs(negatives, positives)
	return matched
}

// investmentTxn classifies the postings of an investment transaction:
// securities lot, cash leg, optional fee, optional elided gain.
type investmentTxn struct {
	cash       *ledger.Posting
	cashExtra  []*ledger.Posting
	securities []*ledger.Posting
	fee        *ledger.Posting
	gain       *ledger.Posting
}

// lot returns the single securities lot; multi-asset shapes are validated
// apart and never reach the single-lot paths.
func (inv *investmentTxn) lot() *ledger.Posting {
	return inv.securities[0]
}

// sell reports whether the classified transaction is a sale (securities
// reduction at a price).
func (inv *investmentTxn) sell() bool {
	return len(inv.securities) == 1 && inv.securities[0].Price != nil
}

// classifyInvestment recognizes the investment shapes the dialect can
// express. The loop sorts postings into a securities lot, an elided gain or
// cash leg, and explicit plain legs (cash or fee by account prefix); the
// shape validators then decide whether the mixture converts.
func classifyInvestment(postings []ledger.Posting) *investmentTxn {
	var inv investmentTxn
	var plains []*ledger.Posting
	for i := range postings {
		p := &postings[i]
		if len(p.Meta) > 0 {
			return nil
		}
		switch {
		case isSecuritiesLeg(p):
			inv.securities = append(inv.securities, p)
		case p.Units == nil:
			if !assignElidedLeg(&inv, p) {
				return nil
			}
		case plainAmountLeg(*p):
			plains = append(plains, p)
		default:
			return nil
		}
	}
	if len(inv.securities) == 0 {
		return nil
	}
	for _, p := range plains {
		if !assignPlainLeg(&inv, p) {
			return nil
		}
	}
	if len(inv.securities) > 1 {
		if validMultiBuy(&inv) {
			return &inv
		}
		return nil
	}
	if len(inv.cashExtra) > 0 {
		return nil
	}
	if validSingleLot(&inv) {
		return &inv
	}
	return nil
}

// validSingleLot validates a one-lot classification: a sell checks the
// sell shape, anything else must be a convertible buy.
func validSingleLot(inv *investmentTxn) bool {
	if inv.sell() {
		return validSell(inv)
	}
	return validBuy(inv)
}

// isSecuritiesLeg reports whether the posting is the securities lot: units
// in a lot currency with a cost batch (and a price marking a reduction).
func isSecuritiesLeg(p *ledger.Posting) bool {
	if p.Cost == nil || p.Units == nil || p.Units.Currency == "" {
		return false
	}
	if p.Price != nil {
		return p.Units.Number.Rat != nil && p.Units.Number.Rat.Sign() < 0
	}
	return true
}

// assignElidedLeg files a nil-units posting as the gain (Income accounts)
// or the elided cash. It reports false on duplicates.
func assignElidedLeg(inv *investmentTxn, p *ledger.Posting) bool {
	if strings.HasPrefix(p.Account, "Income:") {
		if inv.gain != nil {
			return false
		}
		inv.gain = p
		return true
	}
	if inv.cash != nil {
		if p.Account != inv.cash.Account {
			return false
		}
		// A multi-asset export writes one cash line per leg; the extras
		// validate as a sum. Single-lot paths reject them (two payments
		// are two records and must not merge into one leg).
		inv.cashExtra = append(inv.cashExtra, p)
		return true
	}
	inv.cash = p
	return true
}

// assignPlainLeg files an explicit plain posting as the fee (Expenses
// accounts, positive) or the cash. It reports false on duplicates or an
// invalid fee.
func assignPlainLeg(inv *investmentTxn, p *ledger.Posting) bool {
	if strings.HasPrefix(p.Account, "Expenses:") {
		if inv.fee != nil || p.Units.Number.Rat.Sign() <= 0 {
			return false
		}
		inv.fee = p
		return true
	}
	if inv.cash != nil {
		if p.Account != inv.cash.Account {
			return false
		}
		// A multi-asset export writes one cash line per leg; the extras
		// validate as a sum. Single-lot paths reject them (two payments
		// are two records and must not merge into one leg).
		inv.cashExtra = append(inv.cashExtra, p)
		return true
	}
	inv.cash = p
	return true
}

// validBuy reports whether a buy shape converts: cash negative or elided;
// with a fee the cash may be elided (derived) or explicit (amount form,
// which needs an explicit quantity); a bonus share has an elided income
// source instead of cash; the clean shape needs cash = quantity × cost.
func validBuy(inv *investmentTxn) bool {
	if inv.fee != nil {
		return validFeeBuy(inv)
	}
	if inv.gain != nil {
		return inv.cash == nil && inv.lot().Units.Number.Rat != nil
	}
	switch {
	case inv.cash == nil:
		return false
	case inv.cash.Units == nil:
		return true // elided cash: the derived form
	case inv.cash.Units.Number.Rat.Sign() >= 0:
		return false
	case inv.lot().Units.Currency == inv.cash.Units.Currency:
		return false
	default:
		return buyCashMatchesCost(*inv.cash, *inv.lot())
	}
}

// validFeeBuy reports whether a buy with a fee converts.
func validFeeBuy(inv *investmentTxn) bool {
	if inv.gain != nil {
		return false
	}
	if inv.cash == nil || inv.cash.Units == nil {
		// Elided cash: the leg derives cost plus fee from the residual.
		return inv.cash != nil
	}
	// Explicit cash: the amount form requires an explicit quantity and a
	// negative cash leg.
	return inv.lot().Units.Number.Rat != nil && inv.cash.Units.Number.Rat.Sign() < 0
}

// validMultiBuy reports whether a multi-asset buy converts: two or more
// securities lots into the same account, no price, fee, or gain, every lot
// a plain single-cost lot in one currency, and the explicit cash (when
// present) equal to the sum of quantity × unit cost.
func validMultiBuy(inv *investmentTxn) bool {
	if inv.fee != nil || inv.gain != nil || inv.cash == nil {
		return false
	}
	sum, currency, ok := multiLotSum(inv)
	if !ok {
		return false
	}
	if inv.cash.Units == nil {
		return true // elided cash: the residual equals the sum
	}
	total, ok := multiCashTotal(inv, currency)
	return ok && total.Cmp(sum) == 0
}

// multiLotSum sums quantity × unit cost across the lots and reports their
// single cost currency; it fails on priced lots, mixed destination
// accounts, missing quantities, or non-single costs.
func multiLotSum(inv *investmentTxn) (*big.Rat, string, bool) {
	sum := new(big.Rat)
	currency := ""
	for _, s := range inv.securities {
		if s.Price != nil || s.Account != inv.securities[0].Account || s.Units.Number.Rat == nil {
			return nil, "", false
		}
		unit, costCurrency, ok := singleUnitCost(s)
		if !ok {
			return nil, "", false
		}
		if currency == "" {
			currency = costCurrency
		} else if currency != costCurrency {
			return nil, "", false
		}
		sum.Add(sum, new(big.Rat).Mul(s.Units.Number.Rat, unit))
	}
	return sum, currency, true
}

// multiCashTotal sums the absolute cash postings and reports whether every
// one is explicit, negative, and in the given currency.
func multiCashTotal(inv *investmentTxn, currency string) (*big.Rat, bool) {
	total := new(big.Rat)
	for _, p := range append([]*ledger.Posting{inv.cash}, inv.cashExtra...) {
		if p.Units == nil || p.Units.Number.Rat == nil || p.Units.Number.Rat.Sign() >= 0 || p.Units.Currency != currency {
			return nil, false
		}
		total.Add(total, new(big.Rat).Abs(p.Units.Number.Rat))
	}
	return total, true
}

// singleUnitCost returns the unit cost and its currency when the cost batch
// holds exactly one amount component.
func singleUnitCost(s *ledger.Posting) (*big.Rat, string, bool) {
	var found bool
	var unit *big.Rat
	currency := ""
	for _, value := range s.Cost.Components {
		if value.Kind != ledger.ValueAmount {
			continue
		}
		if found {
			return nil, "", false
		}
		found = true
		unit = ledger.DecimalFromNumber(value.Amount.Number).Rat()
		currency = value.Amount.Currency
	}
	return unit, currency, found
}

// validSell reports whether a sell shape converts: explicit cash (positive)
// may pair with an elided gain; elided cash needs the fee to balance; a
// gain with elided cash cannot compile (two elided postings).
func validSell(inv *investmentTxn) bool {
	cashElided := inv.cash != nil && inv.cash.Units == nil
	if inv.cash != nil && !cashElided && inv.cash.Units.Number.Rat.Sign() <= 0 {
		return false
	}
	if inv.gain != nil && (inv.cash == nil || cashElided) {
		return false
	}
	if inv.cash == nil {
		return false
	}
	return !cashElided || inv.fee != nil
}

// buyCashMatchesCost reports whether the cash posting equals quantity × the
// cost batch's unit price, so the auto-calc direction reproduces it exactly.
// An empty quantity is always consistent (the cash drives the share count).
func buyCashMatchesCost(cash, securities ledger.Posting) bool {
	cashRat := cash.Units.Number.Rat
	if cashRat == nil || securities.Cost == nil {
		return false
	}
	if securities.Units.Number.Rat == nil || securities.Units.Number.Rat.Sign() == 0 {
		return true
	}
	for _, value := range securities.Cost.Components {
		if value.Kind != ledger.ValueAmount {
			continue
		}
		qty := ledger.NewDecimal(securities.Units.Number.Rat)
		unit := ledger.DecimalFromNumber(value.Amount.Number)
		want := qty.Mul(unit).Rat()
		got := new(big.Rat).Abs(cashRat)
		if got.Cmp(want) == 0 {
			return true
		}
	}
	return false
}

// plainLegShape counts negative and positive plain legs and reports whether
// the transaction is balanced with a single currency and no zero amounts.
func plainLegShape(postings []ledger.Posting) (negative, positive int, balanced bool) {
	var sum *big.Rat
	currency := ""
	for _, posting := range postings {
		if !plainAmountLeg(posting) {
			return 0, 0, false
		}
		rat := posting.Units.Number.Rat
		if rat.Sign() == 0 {
			return 0, 0, false
		}
		if currency == "" {
			currency = posting.Units.Currency
		} else if posting.Units.Currency != currency {
			return 0, 0, false
		}
		if rat.Sign() < 0 {
			negative++
		} else {
			positive++
		}
		if sum == nil {
			sum = new(big.Rat)
		}
		sum.Add(sum, rat)
	}
	return negative, positive, sum != nil && sum.Sign() == 0
}

// plainAmountLeg reports whether a posting is a bare `Account Number
// Currency` leg: units present, no cost, price, flag, or metadata.
func plainAmountLeg(posting ledger.Posting) bool {
	return posting.Units != nil && posting.Cost == nil && posting.Price == nil && posting.Flag == "" && len(posting.Meta) == 0
}

// textIsReparseSafe reports whether a narration or payee survives the dialect
// grammar unchanged: the line form leaves it unquoted, so sigils that the
// tokenizer would reinterpret forbid conversion.
func textIsReparseSafe(text string) bool {
	if text != strings.TrimSpace(text) {
		return false
	}
	// Mirror the ledger lexer's word-stop set (parser.go scanWordToken):
	// any of these inside bare narration/payee would split into extra
	// tokens and fail to re-parse as one word.
	for _, r := range text {
		if strings.ContainsRune("{}[](),@~=*;\"#^:", r) {
			return false
		}
	}
	return true
}

// spanTodos scans the source bytes covered by span for TODO and FIXME
// comments. It returns the extracted task texts (with the marker prefix
// stripped) and whether any other free-text comment survives there.
func spanTodos(file *source.SourceFile, span source.Span) (todos []string, hasOther bool) {
	if file == nil {
		return nil, false
	}
	text := file.Text(span)
	for _, line := range strings.Split(text, "\n") {
		idx := indexUnquoted(line, ';')
		if idx < 0 {
			continue
		}
		comment := strings.TrimSpace(line[idx+1:])
		trimmed := strings.TrimPrefix(strings.TrimPrefix(comment, "TODO:"), "FIXME:")
		if trimmed != comment {
			if task := strings.TrimSpace(trimmed); task != "" {
				todos = append(todos, task)
			}
			continue
		}
		hasOther = true
	}
	return todos, hasOther
}

// indexUnquoted returns the first occurrence of b outside double-quoted
// strings, or -1.
func indexUnquoted(line string, b byte) int {
	quoted := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			quoted = !quoted
		case b:
			if !quoted {
				return i
			}
		}
	}
	return -1
}

// metaHasKey reports whether the metadata list already defines key.
func metaHasKey(meta []ledger.Metadata, key string) bool {
	for _, m := range meta {
		if m.Key == key {
			return true
		}
	}
	return false
}

// ApplyEdits returns the source text with every edit applied. Edits must be
// non-overlapping; they are applied in span order and all bytes outside the
// edited spans are preserved exactly.
func ApplyEdits(data []byte, edits []Edit) []byte {
	if len(edits) == 0 {
		return data
	}
	sorted := append([]Edit(nil), edits...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Span.Start < sorted[j].Span.Start })
	var out bytes.Buffer
	cursor := 0
	for _, edit := range sorted {
		start, end := edit.Span.Start, edit.Span.End
		if start < cursor || end > len(data) || start > end {
			// Defensive: overlapping or out-of-range edits are skipped so a
			// bug can never corrupt an export.
			continue
		}
		out.Write(data[cursor:start])
		out.WriteString(edit.Text)
		cursor = end
	}
	out.Write(data[cursor:])
	return out.Bytes()
}
