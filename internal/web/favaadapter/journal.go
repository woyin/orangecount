// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

import (
	"sort"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"orangecount/internal/source"
)

// JournalReport is the entry-shaped projection the transplanted Fava journal
// renders. Fava's journal is a list of typed ledger entries, not a flat list of
// postings: a transaction owns its postings, and balance/open/close/note/…
// directives are siblings of it. Projecting only transaction postings, as the
// table-shaped report does, cannot express that structure.
type JournalReport struct {
	Entries []JournalEntry `json:"entries"`
}

// JournalEntry is one ledger directive. Type drives both the row class and the
// entry-type filter; the remaining fields are populated per type and omitted
// when they do not apply.
type JournalEntry struct {
	Type      string                   `json:"type"`
	Date      string                   `json:"date"`
	Flag      string                   `json:"flag,omitempty"`
	Payee     string                   `json:"payee,omitempty"`
	Narration string                   `json:"narration,omitempty"`
	Account   string                   `json:"account,omitempty"`
	Amount    *JournalAmount           `json:"amount,omitempty"`
	Tags      []string                 `json:"tags,omitempty"`
	Links     []string                 `json:"links,omitempty"`
	Metadata  []JournalMeta            `json:"metadata,omitempty"`
	Postings  []JournalPosting         `json:"postings,omitempty"`
	Filenames []string                 `json:"filenames,omitempty"`
	File      string                   `json:"file,omitempty"`
	Span      string                   `json:"span,omitempty"`
	Extra     map[string]string        `json:"extra,omitempty"`
	Balance   map[string]JournalAmount `json:"balance,omitempty"`
	// Change is the per-currency sum of the entry's postings inside the
	// account filter, populated only when the journal is scoped to an
	// account: it is the column Fava's account journal renders as "Change".
	Change []JournalAmount `json:"change,omitempty"`
}

type JournalPosting struct {
	Account  string         `json:"account"`
	Flag     string         `json:"flag,omitempty"`
	Units    *JournalAmount `json:"units,omitempty"`
	Cost     *JournalAmount `json:"cost,omitempty"`
	Price    *JournalAmount `json:"price,omitempty"`
	Metadata []JournalMeta  `json:"metadata,omitempty"`
}

type JournalAmount struct {
	Number   report.PresentedDecimal `json:"number"`
	Currency string                  `json:"currency"`
}

type JournalMeta struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// journalFlagClass maps a transaction flag to the CSS modifier Fava uses to
// show or hide it with the cleared/pending/other chips.
func journalFlagClass(flag string) string {
	switch flag {
	case "*", "txn":
		return "cleared"
	case "!":
		return "pending"
	default:
		return "other"
	}
}

func journalAmount(amount *ledger.Amount) *JournalAmount {
	if amount == nil || amount.Currency == "" || amount.Number.Raw == "" {
		return nil
	}
	return &JournalAmount{Number: report.FormatDecimal(ledger.DecimalFromNumber(amount.Number)), Currency: amount.Currency}
}

// journalCost renders a booked lot cost as a per-unit amount. The evaluator
// resolves `{}` against the matched lot before publishing the entry stream, so
// by this point a reduction carries a concrete cost.
func journalCost(spec *ledger.CostSpec) *JournalAmount {
	if spec == nil {
		return nil
	}
	for _, component := range spec.Components {
		if component.Kind == ledger.ValueAmount && component.Amount.Currency != "" && component.Amount.Number.Raw != "" {
			return &JournalAmount{Number: report.FormatDecimal(ledger.DecimalFromNumber(component.Amount.Number)), Currency: component.Amount.Currency}
		}
	}
	return nil
}

func journalPrice(spec *ledger.PriceSpec) *JournalAmount {
	if spec == nil {
		return nil
	}
	return journalAmount(&spec.Amount)
}

func journalMeta(values []ledger.Metadata) []JournalMeta {
	if len(values) == 0 {
		return nil
	}
	out := make([]JournalMeta, 0, len(values))
	for _, value := range values {
		out = append(out, JournalMeta{Key: value.Key, Value: strings.TrimSpace(value.Value.Raw)})
	}
	return out
}

// ProjectJournal walks the published entry stream and emits every directive the
// Fava journal can display, newest first. It never re-evaluates: the entries
// already carry booked costs and interpolated amounts.
func ProjectJournal(e ledger.Evaluation, graph *source.Graph, filters report.Filters, journalFilters report.JournalFilters) JournalReport {
	account := strings.TrimSpace(filters.Account)
	entries := make([]JournalEntry, 0, len(e.Entries))
	for _, record := range e.Entries {
		entry, ok := projectJournalEntry(record)
		if !ok {
			continue
		}
		entry.File = journalDisplayPath(record.File, graph)
		entry.Span = record.Span.String()
		if account != "" {
			entry.Change = journalChange(record, account)
		}
		entries = append(entries, entry)
	}
	entries = filterJournalEntries(entries, filters, journalFilters)
	// Fava lists the journal newest first, and reverses within a day too: the
	// last entry written for a date appears at the top. Reversing the
	// date-ascending stream produces that, and the stable sort afterwards only
	// has to fix up any input that was not already in date order.
	for left, right := 0, len(entries)-1; left < right; left, right = left+1, right-1 {
		entries[left], entries[right] = entries[right], entries[left]
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Date > entries[j].Date })
	return JournalReport{Entries: entries}
}

func journalDisplayPath(path string, graph *source.Graph) string {
	if graph != nil {
		if id, ok := graph.ByPath[path]; ok {
			return graph.DisplayPath(id)
		}
	}
	return source.SafeDisplayPath(path)
}

func projectJournalEntry(record ledger.EntryRecord) (JournalEntry, bool) {
	switch directive := record.Directive.(type) {
	case *ledger.Transaction:
		return journalTransaction(directive), true
	case ledger.Transaction:
		return journalTransaction(&directive), true
	case ledger.Balance:
		entry := JournalEntry{Type: "balance", Date: directive.Date.Raw, Flag: "Bal", Account: directive.Account}
		entry.Amount = journalAmount(&directive.Amount)
		return entry, true
	case ledger.Open:
		entry := JournalEntry{Type: "open", Date: directive.Date.Raw, Flag: "Open", Account: directive.Account}
		if len(directive.Currencies) > 0 {
			entry.Extra = map[string]string{"currencies": strings.Join(directive.Currencies, ", ")}
		}
		return entry, true
	case ledger.Close:
		return JournalEntry{Type: "close", Date: directive.Date.Raw, Flag: "Close", Account: directive.Account}, true
	case ledger.Note:
		return JournalEntry{Type: "note", Date: directive.Date.Raw, Flag: "Note", Account: directive.Account, Narration: directive.Comment}, true
	case ledger.Document:
		entry := JournalEntry{Type: "document", Date: directive.Date.Raw, Flag: "Doc", Account: directive.Account, Tags: directive.Tags, Links: directive.Links}
		entry.Filenames = directive.Filenames
		return entry, true
	case ledger.Pad:
		entry := JournalEntry{Type: "pad", Date: directive.Date.Raw, Flag: "Pad", Account: directive.Account}
		entry.Extra = map[string]string{"source_account": directive.SourceAccount}
		return entry, true
	case ledger.Query:
		entry := JournalEntry{Type: "query", Date: directive.Date.Raw, Flag: "Query", Narration: directive.Name}
		entry.Extra = map[string]string{"query": directive.Query}
		return entry, true
	case ledger.Custom:
		entry := JournalEntry{Type: "custom", Date: directive.Date.Raw, Flag: "Custom", Narration: directive.Type}
		values := make([]string, 0, len(directive.Values))
		for _, value := range directive.Values {
			values = append(values, strings.TrimSpace(value.Raw))
		}
		entry.Extra = map[string]string{"values": strings.Join(values, " ")}
		return entry, true
	default:
		// Price and event directives are deliberately absent: Fava's journal
		// does not list them, giving each its own report page instead.
		return JournalEntry{}, false
	}
}

func journalTransaction(transaction *ledger.Transaction) JournalEntry {
	entry := JournalEntry{
		Type:      "transaction",
		Date:      transaction.Date.Raw,
		Flag:      transaction.Flag,
		Payee:     transaction.Payee,
		Narration: transaction.Narration,
		Tags:      transaction.Tags,
		Links:     transaction.Links,
		Metadata:  journalMeta(transaction.Meta),
	}
	for _, posting := range transaction.Postings {
		entry.Postings = append(entry.Postings, JournalPosting{
			Account:  posting.Account,
			Flag:     posting.Flag,
			Units:    journalAmount(posting.Units),
			Cost:     journalCost(posting.Cost),
			Price:    journalPrice(posting.Price),
			Metadata: journalMeta(posting.Meta),
		})
	}
	return entry
}

// journalChange sums the posting units that fall inside the filtered account
// so the account journal can show the per-entry change the way Fava does.
// Only transactions carry postings; every other directive yields no change.
func journalChange(record ledger.EntryRecord, account string) []JournalAmount {
	var transaction *ledger.Transaction
	switch directive := record.Directive.(type) {
	case *ledger.Transaction:
		transaction = directive
	case ledger.Transaction:
		transaction = &directive
	default:
		return nil
	}
	totals := map[string]ledger.Decimal{}
	for _, posting := range transaction.Postings {
		if !accountWithinPrefix(posting.Account, account) {
			continue
		}
		if posting.Units == nil || posting.Units.Currency == "" || posting.Units.Number.Raw == "" {
			continue
		}
		value := ledger.DecimalFromNumber(posting.Units.Number)
		if existing, ok := totals[posting.Units.Currency]; ok {
			value = existing.Add(value)
		}
		totals[posting.Units.Currency] = value
	}
	if len(totals) == 0 {
		return nil
	}
	currencies := make([]string, 0, len(totals))
	for currency := range totals {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	change := make([]JournalAmount, 0, len(totals))
	for _, currency := range currencies {
		change = append(change, JournalAmount{Number: report.FormatDecimal(totals[currency]), Currency: currency})
	}
	return change
}

// filterJournalEntries applies the same global and journal-specific filters the
// table report honours, so switching representation does not change which
// entries a bookmarked URL selects.
func filterJournalEntries(entries []JournalEntry, filters report.Filters, journal report.JournalFilters) []JournalEntry {
	account := strings.TrimSpace(filters.Account)
	text := strings.ToLower(strings.TrimSpace(filters.Text))
	hasTime := strings.TrimSpace(filters.TimePrefix) != "" || strings.TrimSpace(filters.TimeBegin) != "" || strings.TrimSpace(filters.TimeEnd) != ""
	flag := strings.TrimSpace(journal.Flag)
	tag := strings.ToLower(strings.TrimSpace(journal.Tag))
	link := strings.ToLower(strings.TrimSpace(journal.Link))
	payee := strings.ToLower(strings.TrimSpace(journal.Payee))
	narration := strings.ToLower(strings.TrimSpace(journal.Narration))
	kind := strings.TrimSpace(journal.Kind)

	out := make([]JournalEntry, 0, len(entries))
	for _, entry := range entries {
		if hasTime && !filters.MatchesDate(entry.Date) {
			continue
		}
		if account != "" && !entryTouchesAccount(entry, account) {
			continue
		}
		if flag != "" && entry.Flag != flag {
			continue
		}
		if kind != "" && !strings.EqualFold(entry.Type, kind) {
			continue
		}
		if tag != "" && !containsFold(entry.Tags, tag) {
			continue
		}
		if link != "" && !containsFold(entry.Links, link) {
			continue
		}
		if payee != "" && !strings.Contains(strings.ToLower(entry.Payee), payee) {
			continue
		}
		if narration != "" && !strings.Contains(strings.ToLower(entry.Narration), narration) {
			continue
		}
		if text != "" && !entryMatchesText(entry, text) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func entryTouchesAccount(entry JournalEntry, account string) bool {
	if accountWithinPrefix(entry.Account, account) {
		return true
	}
	for _, posting := range entry.Postings {
		if accountWithinPrefix(posting.Account, account) {
			return true
		}
	}
	return false
}

func accountWithinPrefix(name, prefix string) bool {
	return name != "" && (name == prefix || strings.HasPrefix(name, prefix+":"))
}

func entryMatchesText(entry JournalEntry, needle string) bool {
	fields := []string{entry.Payee, entry.Narration, entry.Account, entry.Flag}
	for _, value := range fields {
		if value != "" && strings.Contains(strings.ToLower(value), needle) {
			return true
		}
	}
	if containsFold(entry.Tags, needle) || containsFold(entry.Links, needle) {
		return true
	}
	for _, posting := range entry.Postings {
		if strings.Contains(strings.ToLower(posting.Account), needle) {
			return true
		}
	}
	return false
}

func containsFold(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), needle) {
			return true
		}
	}
	return false
}
