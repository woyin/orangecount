// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

import (
	"math/big"
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
	Type      string           `json:"type"`
	Date      string           `json:"date"`
	Flag      string           `json:"flag,omitempty"`
	Payee     string           `json:"payee,omitempty"`
	Narration string           `json:"narration,omitempty"`
	Account   string           `json:"account,omitempty"`
	Amount    *JournalAmount   `json:"amount,omitempty"`
	Tags      []string         `json:"tags,omitempty"`
	Links     []string         `json:"links,omitempty"`
	Metadata  []JournalMeta    `json:"metadata,omitempty"`
	Postings  []JournalPosting `json:"postings,omitempty"`
	Filenames []string         `json:"filenames,omitempty"`
	File      string           `json:"file,omitempty"`
	Span      string           `json:"span,omitempty"`
	// EntryHash is the position-derived identity used by the context modal
	// (#context-<hash>), stable within a snapshot.
	EntryHash string                   `json:"entry_hash,omitempty"`
	Extra     map[string]string        `json:"extra,omitempty"`
	Balance   map[string]JournalAmount `json:"balance,omitempty"`
	// Change is the per-currency sum of the entry's postings inside the
	// account filter, populated only when the journal is scoped to an
	// account: it is the column Fava's account journal renders as "Change".
	Change []JournalAmount `json:"change,omitempty"`
	// CustomValues is the typed value list of a custom directive, projected so
	// the journal can render each value by its data type the way Fava does
	// (accounts as links, amounts as formatted amounts, strings quoted, etc.).
	CustomValues []JournalCustomValue `json:"custom_values,omitempty"`
}

// JournalCustomValue is one typed value of a custom directive.
type JournalCustomValue struct {
	Dtype string `json:"dtype"`
	Value string `json:"value"`
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
		entry.EntryHash = entryHash(record)
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
		entry.Metadata = journalMeta(directive.Meta)
		return entry, true
	case ledger.Open:
		entry := JournalEntry{Type: "open", Date: directive.Date.Raw, Flag: "Open", Account: directive.Account}
		if len(directive.Currencies) > 0 {
			entry.Extra = map[string]string{"currencies": strings.Join(directive.Currencies, ", ")}
		}
		entry.Metadata = journalMeta(directive.Meta)
		return entry, true
	case ledger.Close:
		entry := JournalEntry{Type: "close", Date: directive.Date.Raw, Flag: "Close", Account: directive.Account}
		entry.Metadata = journalMeta(directive.Meta)
		return entry, true
	case ledger.Note:
		entry := JournalEntry{Type: "note", Date: directive.Date.Raw, Flag: "Note", Account: directive.Account, Narration: directive.Comment}
		entry.Metadata = journalMeta(directive.Meta)
		return entry, true
	case ledger.Document:
		entry := JournalEntry{Type: "document", Date: directive.Date.Raw, Flag: "Doc", Account: directive.Account, Tags: directive.Tags, Links: directive.Links}
		entry.Filenames = directive.Filenames
		entry.Metadata = journalMeta(directive.Meta)
		return entry, true
	case ledger.Pad:
		entry := JournalEntry{Type: "pad", Date: directive.Date.Raw, Flag: "Pad", Account: directive.Account}
		entry.Extra = map[string]string{"source_account": directive.SourceAccount}
		entry.Metadata = journalMeta(directive.Meta)
		return entry, true
	case ledger.Query:
		entry := JournalEntry{Type: "query", Date: directive.Date.Raw, Flag: "Query", Narration: directive.Name}
		entry.Extra = map[string]string{"query": directive.Query}
		entry.Metadata = journalMeta(directive.Meta)
		return entry, true
	case ledger.Custom:
		entry := JournalEntry{Type: "custom", Date: directive.Date.Raw, Flag: "Custom", Narration: directive.Type}
		values := make([]string, 0, len(directive.Values))
		for _, value := range directive.Values {
			values = append(values, strings.TrimSpace(value.Raw))
		}
		entry.Extra = map[string]string{"values": strings.Join(values, " ")}
		entry.CustomValues = journalCustomValues(directive.Values)
		entry.Metadata = journalMeta(directive.Meta)
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

// journalCustomValues projects a custom directive's typed values the way
// Fava renders them: accounts verbatim (the frontend links them), amounts
// as formatted amounts, strings quoted, booleans and dates literally,
// numbers as-is, and everything else by its raw text.
func journalCustomValues(values []ledger.Value) []JournalCustomValue {
	if len(values) == 0 {
		return nil
	}
	out := make([]JournalCustomValue, 0, len(values))
	for _, value := range values {
		var dtype, display string
		switch value.Kind {
		case ledger.ValueAccount, ledger.ValueCurrency:
			dtype, display = "account", strings.TrimSpace(value.Raw)
		case ledger.ValueAmount:
			dtype = "amount"
			display = report.FormatDecimal(ledger.DecimalFromNumber(value.Amount.Number)).Display + " " + value.Amount.Currency
		case ledger.ValueString:
			dtype, display = "string", "\""+value.String+"\""
		case ledger.ValueBool:
			dtype, display = "bool", fmtBool(value.Bool)
		case ledger.ValueDate:
			dtype, display = "date", value.Date.Raw
		case ledger.ValueNumber:
			dtype, display = "number", strings.TrimSpace(value.Raw)
		case ledger.ValueTag:
			dtype, display = "tag", "#"+strings.TrimSpace(value.Raw)
		case ledger.ValueLink:
			dtype, display = "link", "^"+strings.TrimSpace(value.Raw)
		default:
			dtype, display = "text", strings.TrimSpace(value.Raw)
		}
		out = append(out, JournalCustomValue{Dtype: dtype, Value: display})
	}
	return out
}

func fmtBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}
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
// entries a bookmarked URL selects. The text filter is Fava's filter query
// (tags, links, key:value, and/or/-); unparseable input degrades to the
// legacy substring match so a caller that skipped validation never loses
// entries silently.
func filterJournalEntries(entries []JournalEntry, filters report.Filters, journal report.JournalFilters) []JournalEntry {
	state := newJournalFilterState(filters, journal)
	out := make([]JournalEntry, 0, len(entries))
	for _, entry := range entries {
		if state.excluded(entry) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

// journalFilterState precomputes the per-request filter inputs so entry
// projections (journal, export) can apply identical predicates per entry.
type journalFilterState struct {
	filters    report.Filters
	account    string
	text       string
	textFilter *report.FQL
	textErr    error
	hasTime    bool
	flag       string
	tag        string
	link       string
	payee      string
	narration  string
	kind       string
}

func newJournalFilterState(filters report.Filters, journal report.JournalFilters) journalFilterState {
	rawText := strings.TrimSpace(filters.Text)
	textFilter, textErr := report.ParseFQL(rawText)
	return journalFilterState{
		filters:    filters,
		account:    strings.TrimSpace(filters.Account),
		text:       strings.ToLower(rawText),
		textFilter: textFilter,
		textErr:    textErr,
		hasTime:    strings.TrimSpace(filters.TimePrefix) != "" || strings.TrimSpace(filters.TimeBegin) != "" || strings.TrimSpace(filters.TimeEnd) != "",
		flag:       strings.TrimSpace(journal.Flag),
		tag:        strings.ToLower(strings.TrimSpace(journal.Tag)),
		link:       strings.ToLower(strings.TrimSpace(journal.Link)),
		payee:      strings.ToLower(strings.TrimSpace(journal.Payee)),
		narration:  strings.ToLower(strings.TrimSpace(journal.Narration)),
		kind:       strings.TrimSpace(journal.Kind),
	}
}

func (state journalFilterState) excluded(entry JournalEntry) bool {
	if state.hasTime && !state.filters.MatchesDate(entry.Date) {
		return true
	}
	if state.account != "" && !entryTouchesAccount(entry, state.account) {
		return true
	}
	if state.flag != "" && entry.Flag != state.flag {
		return true
	}
	if state.kind != "" && !strings.EqualFold(entry.Type, state.kind) {
		return true
	}
	if state.tag != "" && !containsFold(entry.Tags, state.tag) {
		return true
	}
	if state.link != "" && !containsFold(entry.Links, state.link) {
		return true
	}
	if state.payee != "" && !strings.Contains(strings.ToLower(entry.Payee), state.payee) {
		return true
	}
	if state.narration != "" && !strings.Contains(strings.ToLower(entry.Narration), state.narration) {
		return true
	}
	if state.text != "" {
		if state.textFilter != nil && state.textErr == nil {
			return !state.textFilter.Match(journalFQLTarget(entry))
		}
		return !entryMatchesText(entry, state.text)
	}
	return false
}

// journalFQLTarget maps a projected entry onto the shape Fava's filter query
// evaluates: tags and links are exact, metadata keys are reachable as
// key:value terms, and balance assertions expose their amount as "number".
func journalFQLTarget(entry JournalEntry) report.FQLTarget {
	target := report.FQLTarget{
		Tags:      entry.Tags,
		Links:     entry.Links,
		Payee:     entry.Payee,
		Narration: entry.Narration,
		Account:   entry.Account,
		Flag:      entry.Flag,
		Date:      entry.Date,
		Metadata:  map[string]string{},
	}
	for _, meta := range entry.Metadata {
		target.Metadata[meta.Key] = meta.Value
	}
	if entry.Type == "balance" {
		target.Amount = presentedNumber(entry.Amount)
	}
	for _, posting := range entry.Postings {
		target.Postings = append(target.Postings, report.FQLPosting{
			Account: posting.Account,
			Units:   presentedNumber(posting.Units),
		})
	}
	return target
}

// presentedNumber recovers the machine value behind a presented amount so
// amount comparisons can run; it returns nil when nothing parseable exists.
func presentedNumber(amount *JournalAmount) *big.Float {
	if amount == nil {
		return nil
	}
	for _, candidate := range []string{amount.Number.Exact, strings.ReplaceAll(amount.Number.Display, ",", "")} {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if number, _, err := big.ParseFloat(strings.TrimSpace(candidate), 10, 128, big.ToNearestEven); err == nil {
			return number
		}
	}
	return nil
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
