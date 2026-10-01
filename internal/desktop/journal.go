// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"orangecount/internal/ledger"
)

// Keep only source references and a search index, not a second complete JSON
// projection. Exact amount/cost formatting runs for the requested page only.
type journalEntry struct {
	index            int
	id, date, search string
	accounts         []string
}

func indexJournal(e ledger.Evaluation, snapshotID string) []journalEntry {
	entries := make([]journalEntry, 0)
	counts := map[string]int{}
	for index, record := range e.Entries {
		var tx *ledger.Transaction
		switch value := record.Directive.(type) {
		case *ledger.Transaction:
			tx = value
		case ledger.Transaction:
			tx = &value
		}
		if tx == nil {
			continue
		}
		parts := []string{record.File, tx.RawText(), tx.Date.Raw, tx.Flag, tx.Payee, tx.Narration, strings.Join(tx.Tags, " "), strings.Join(tx.Links, " ")}
		search := []string{tx.Date.Raw, tx.Payee, tx.Narration, strings.Join(tx.Tags, " "), strings.Join(tx.Links, " ")}
		accounts := make([]string, 0, len(tx.Postings))
		for _, posting := range tx.Postings {
			parts = append(parts, posting.RawText())
			search = append(search, posting.Account)
			if posting.Units != nil {
				search = append(search, posting.Units.Number.Raw, posting.Units.Currency)
			}
			accounts = append(accounts, posting.Account)
		}
		sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
		id := hex.EncodeToString(sum[:])
		counts[id]++
		entries = append(entries, journalEntry{index: index, id: id, date: tx.Date.Raw, search: strings.ToLower(strings.Join(search, " ")), accounts: accounts})
	}
	// Ambiguous duplicates deliberately cannot restore expansion across content
	// revisions. Unique source content survives unrelated insertions/line shifts.
	for i := range entries {
		if counts[entries[i].id] > 1 {
			entries[i].id += ":" + snapshotID + ":" + strconv.Itoa(i)
		}
	}
	return entries
}

func validJournalRequest(r Request) bool {
	if r.View != "prices" && (r.PriceSection != "" || r.PriceBase != "" || r.PriceQuote != "") {
		return false
	}
	if r.View == "prices" {
		if r.Valuation != "" && r.Valuation != "at-cost" {
			return false
		}
		if r.Account != "" || r.Text != "" || (r.PriceSection != "" && r.PriceSection != "history" && r.PriceSection != "commodities") || (r.PriceQuote != "" && r.PriceBase == "") {
			return false
		}
		if (r.PriceSection == "history" && r.PriceBase == "") || (r.PriceSection == "commodities" && (r.PriceBase != "" || r.PriceQuote != "")) {
			return false
		}
	}
	if r.LotKind != "" && (r.View != "holdings" || r.HoldingSection != "lots" || r.EventID == "" || (r.LotKind != "consumed" && r.LotKind != "merged")) {
		return false
	}
	if r.View != "holdings" && (r.HoldingSection != "" || r.GroupID != "" || r.EventID != "") {
		return false
	}
	if r.View == "holdings" {
		if r.Scope == "direct" && r.Account == "" {
			return false
		}
		if r.Text != "" {
			return false
		}
		switch r.HoldingSection {
		case "", "groups", "events", "history", "lots":
		default:
			return false
		}
		if r.HoldingSection == "lots" && (r.GroupID == "") == (r.EventID == "") {
			return false
		}
		if r.GroupID != "" && r.HoldingSection != "lots" && r.HoldingSection != "history" {
			return false
		}
		if r.GroupID != "" && r.SnapshotID == "" {
			return false
		}
		if r.EventID != "" && (r.HoldingSection != "lots" || r.SnapshotID == "") {
			return false
		}
	}
	if r.GroupBy != "" && (r.View != "holdings" || (r.GroupBy != "account" && r.GroupBy != "commodity" && r.GroupBy != "cost_currency")) {
		return false
	}
	if r.Scope != "" && ((r.View != "account" && r.View != "holdings") || (r.Scope != "direct" && r.Scope != "subtree")) {
		return false
	}
	if r.Valuation == "market-value" {
		if r.Currency == "" || r.Currency == "native" || (r.View != "dashboard" && r.View != "balance_sheet" && r.View != "income_statement" && r.View != "account" && r.View != "holdings") {
			return false
		}
	} else if r.Valuation != "" && r.Valuation != "units" && r.Valuation != "at-cost" {
		return false
	}
	for _, date := range []string{r.DateFrom, r.DateTo} {
		if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return false
			}
		}
	}
	if r.DateFrom != "" && r.DateTo != "" && r.DateFrom > r.DateTo {
		return false
	}
	if r.Offset < 0 || r.Limit < 0 || r.Limit > 500 {
		return false
	}
	if r.Offset > 0 && r.SnapshotID == "" {
		return false
	}
	if r.Sort != "" && r.Sort != "ascending" && r.Sort != "descending" {
		return false
	}
	switch r.Window {
	case "", "all", "1m", "1y":
	default:
		return false
	}
	return true
}

func (p *publication) dateRange(r Request) (string, string) {
	from, to := r.DateFrom, r.DateTo
	if to == "" {
		to = p.latestDate
	}
	if from == "" && to != "" && (r.Window == "1m" || r.Window == "1y") {
		anchor, _ := time.Parse("2006-01-02", to)
		months := -1
		if r.Window == "1y" {
			months = -12
		}
		// Month start avoids normalization surprises on the 29th–31st.
		from = time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, months+1, 0).Format("2006-01-02")
	}
	return from, to
}

func (p *publication) readJournal(r Request) JournalPayload {
	from, to := p.dateRange(r)
	limit := r.Limit
	if limit == 0 {
		limit = 100
	}
	result := JournalPayload{Transactions: []JournalTransactionDTO{}, Offset: r.Offset, Limit: limit, DateFrom: from, DateTo: to}
	query := strings.ToLower(strings.TrimSpace(r.Text))
	selected := make([]journalEntry, 0, limit)
	for step := range p.journal {
		i := step
		if r.Sort != "ascending" {
			i = len(p.journal) - 1 - step
		}
		entry := p.journal[i]
		if from != "" && entry.date < from || to != "" && entry.date > to || query != "" && !strings.Contains(entry.search, query) {
			continue
		}
		if r.Account != "" {
			matches := false
			for _, account := range entry.accounts {
				if account == r.Account || (r.Scope != "direct" && strings.HasPrefix(account, r.Account+":")) {
					matches = true
					break
				}
			}
			if !matches {
				continue
			}
		}
		if result.TotalCount >= r.Offset && len(selected) < limit {
			selected = append(selected, entry)
		}
		result.TotalCount++
	}
	evaluation := p.evaluation
	evaluation.Entries = make([]ledger.EntryRecord, 0, len(selected))
	for _, entry := range selected {
		evaluation.Entries = append(evaluation.Entries, p.evaluation.Entries[entry.index])
	}
	result.Transactions = ProjectJournal(evaluation).Transactions
	for i := range result.Transactions {
		result.Transactions[i].ID = selected[i].id
	}
	result.HasMore = r.Offset+len(selected) < result.TotalCount
	return result
}
