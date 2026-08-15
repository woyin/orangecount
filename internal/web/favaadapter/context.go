// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

import (
	"crypto/sha256"
	"encoding/hex"

	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"orangecount/internal/source"
)

// EntryContext is what Fava's context modal shows for one entry: where it
// lives, its source exactly as written, and (for Transaction/Balance) the
// balances of the affected accounts immediately before and after the entry.
// The editable CodeMirror slice belongs to a later phase (H1); this is the
// read-only projection.
type EntryContext struct {
	Entry       JournalEntry `json:"entry"`
	SourceSlice string       `json:"source_slice"`
	SHA256Sum   string       `json:"sha256sum"`
	// BalancesBefore/After are the per-account balances (grouped by currency)
	// of the accounts the entry touches, immediately before and after the
	// entry. Only Transaction and Balance entries carry them; other kinds are
	// nil (mirrors Fava's context modal).
	BalancesBefore map[string][]JournalAmount `json:"balances_before,omitempty"`
	BalancesAfter  map[string][]JournalAmount `json:"balances_after,omitempty"`
}

// entryHash identifies one ledger entry by its source position. The position
// is immutable within a snapshot, so the hash is a stable address for the
// context modal without exposing file paths in URLs.
func entryHash(record ledger.EntryRecord) string {
	digest := sha256.New()
	digest.Write([]byte(record.File))
	digest.Write([]byte{0})
	digest.Write([]byte(record.Span.String()))
	return hex.EncodeToString(digest.Sum(nil))
}

// ProjectEntryContext resolves an entry hash to its projection, source slice,
// and (for Transaction/Balance) the affected accounts' balances immediately
// before and after the entry. It scans the published entry stream; hashes are
// position-derived, so the same snapshot always answers the same way.
func ProjectEntryContext(e ledger.Evaluation, graph *source.Graph, hash string) (EntryContext, bool) {
	if hash == "" {
		return EntryContext{}, false
	}
	for index, record := range e.Entries {
		if entryHash(record) != hash {
			continue
		}
		entry, ok := projectJournalEntry(record)
		if !ok {
			return EntryContext{}, false
		}
		entry.EntryHash = hash
		entry.File = journalDisplayPath(record.File, graph)
		entry.Span = record.Span.String()
		slice := entrySourceBlock(record, graph)
		ctx := EntryContext{Entry: entry, SourceSlice: slice, SHA256Sum: sha256Hex(slice)}
		if before, after, ok := journalContextBalances(e.Entries, index); ok {
			ctx.BalancesBefore = before
			ctx.BalancesAfter = after
		}
		return ctx, true
	}
	return EntryContext{}, false
}

// journalContextBalances computes the balances of the accounts an entry
// touches, immediately before and after it, mirroring Fava's context modal.
// It returns (before, after, true) only for Transaction and Balance entries;
// other kinds yield (nil, nil, false). The running balance accumulates every
// transaction posting up to (but not including) the target entry for
// "before", then adds the target's own postings for "after" (a Balance has no
// after). Accounts are grouped by currency, one JournalAmount per currency.
// journalContextBalances computes the balances of the accounts an entry
// touches, immediately before and after it, mirroring Fava's context modal.
// It returns (before, after, true) only for Transaction and Balance entries;
// other kinds yield (nil, nil, false). The running balance accumulates every
// transaction posting up to (but not including) the target entry for
// "before", then adds the target's own postings for "after" (a Balance has no
// after). Accounts are grouped by currency, one JournalAmount per currency.
func journalContextBalances(entries []ledger.EntryRecord, target int) (map[string][]JournalAmount, map[string][]JournalAmount, bool) {
	accounts, targetTransaction, isBalance, ok := contextTouchedAccounts(entries[target].Directive)
	if !ok {
		return nil, nil, false
	}
	running := map[string]map[string]ledger.Decimal{}
	for i := 0; i < target; i++ {
		for _, posting := range transactionPostingsOf(entries[i].Directive) {
			accumulateContextPosting(running, accounts, posting.Account, posting.Units)
		}
	}
	before := presentContextBalances(running)
	if isBalance {
		return before, nil, true
	}
	for _, posting := range targetTransaction.Postings {
		accumulateContextPosting(running, accounts, posting.Account, posting.Units)
	}
	return before, presentContextBalances(running), true
}

// contextTouchedAccounts collects the accounts the target directive touches
// and, for transactions, the directive itself. ok is false for entry kinds
// the context modal does not balance.
func contextTouchedAccounts(directive ledger.Directive) (accounts map[string]bool, tx *ledger.Transaction, isBalance bool, ok bool) {
	accounts = map[string]bool{}
	switch value := directive.(type) {
	case *ledger.Transaction:
		for _, posting := range value.Postings {
			accounts[posting.Account] = true
		}
		return accounts, value, false, true
	case ledger.Transaction:
		for _, posting := range value.Postings {
			accounts[posting.Account] = true
		}
		return accounts, &value, false, true
	case ledger.Balance:
		accounts[value.Account] = true
	case *ledger.Balance:
		accounts[value.Account] = true
	default:
		return nil, nil, false, false
	}
	return accounts, nil, true, true
}

// transactionPostingsOf returns a directive's postings when it is a
// transaction (either representation), nil otherwise.
func transactionPostingsOf(directive ledger.Directive) []ledger.Posting {
	switch value := directive.(type) {
	case *ledger.Transaction:
		return value.Postings
	case ledger.Transaction:
		return value.Postings
	default:
		return nil
	}
}

// accumulateContextPosting adds one posting's units into the running
// per-account, per-currency totals, skipping postings the target does not
// touch or that carry no complete amount.
func accumulateContextPosting(running map[string]map[string]ledger.Decimal, accounts map[string]bool, account string, units *ledger.Amount) {
	if units == nil || units.Currency == "" || units.Number.Raw == "" || !accounts[account] {
		return
	}
	byCurrency := running[account]
	if byCurrency == nil {
		byCurrency = map[string]ledger.Decimal{}
		running[account] = byCurrency
	}
	value := ledger.DecimalFromNumber(units.Number)
	if existing, ok := byCurrency[units.Currency]; ok {
		value = existing.Add(value)
	}
	byCurrency[units.Currency] = value
}

// presentContextBalances turns the running per-account/per-currency decimal
// map into the wire form Fava shows (one JournalAmount per currency).
func presentContextBalances(running map[string]map[string]ledger.Decimal) map[string][]JournalAmount {
	if len(running) == 0 {
		return nil
	}
	out := make(map[string][]JournalAmount, len(running))
	for account, byCurrency := range running {
		amounts := make([]JournalAmount, 0, len(byCurrency))
		for currency, value := range byCurrency {
			if value.IsZero() {
				continue
			}
			amounts = append(amounts, JournalAmount{Number: report.FormatDecimal(value), Currency: currency})
		}
		if len(amounts) > 0 {
			out[account] = amounts
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sha256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
