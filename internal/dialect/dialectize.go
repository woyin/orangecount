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
		line := SerializeDialect(txn)
		if line == "" {
			continue
		}
		if spanHasComment(file.Source, txn.At) {
			// A comment inside the transaction block would be deleted by a
			// span replacement; keep the block standard instead.
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

// eligibleForDialect applies the ADR-0045 conversion filter: two postings,
// opposite signed equal-magnitude amounts in one currency, no cost, price,
// posting flag, or metadata on either leg, no transaction metadata, flag *
// or !, and a non-empty reparse-safe narration and payee.
func eligibleForDialect(txn *ledger.Transaction) bool {
	if !txn.Date.Valid() {
		return false
	}
	if txn.Flag != "" && txn.Flag != "*" && txn.Flag != "!" {
		return false
	}
	if len(txn.Meta) > 0 || len(txn.Postings) < 2 {
		return false
	}
	if txn.Narration == "" && txn.Payee == "" {
		// An empty narration would be re-materialized as the 消费 default.
		return false
	}
	if !textIsReparseSafe(txn.Narration) || !textIsReparseSafe(txn.Payee) {
		return false
	}
	negative, positive, balanced := plainLegShape(txn.Postings)
	// A dialect block expresses one source with many destinations, or many
	// sources with one destination.
	return balanced && (negative == 1 || positive == 1)
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

// spanHasComment reports whether the source bytes covered by span contain a
// semicolon comment outside a string.
func spanHasComment(file *source.SourceFile, span source.Span) bool {
	if file == nil {
		return false
	}
	text := file.Text(span)
	quoted := false
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '"':
			quoted = !quoted
		case ';':
			if !quoted {
				return true
			}
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
