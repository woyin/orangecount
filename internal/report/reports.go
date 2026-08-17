// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package report contains deterministic core-derived reports over an evaluated
// ledger. Reports do not read source files or execute plugins.
package report

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/source"
)

// Set bundles every standard report result for one evaluation, in the shape
// the web layer serializes.
type Set struct {
	Accounts        query.Result `json:"accounts"`
	Journal         query.Result `json:"journal"`
	TrialBalance    query.Result `json:"trial_balance"`
	BalanceSheet    query.Result `json:"balance_sheet"`
	IncomeStatement query.Result `json:"income_statement"`
	Holdings        query.Result `json:"holdings"`
	Prices          query.Result `json:"prices"`
	Events          query.Result `json:"events"`
	Documents       query.Result `json:"documents"`
	Statistics      query.Result `json:"statistics"`
	Errors          query.Result `json:"errors"`
}

// Filters are the URL-backed global filters shared by report views. TimePrefix
// is an ISO year or year-month prefix; empty means no time restriction.
type Filters struct {
	Account    string
	Text       string
	TimePrefix string
	// TimeBegin/TimeEnd form a half-open ISO date range; they carry time
	// filters that a plain prefix cannot express (e.g. Fava's "2025-Q2").
	TimeBegin string
	TimeEnd   string
	Period    string
}

// MatchesDate reports whether an ISO date string passes the filter's time
// constraints. Rows without a parseable date are kept, matching the
// prefix-only behaviour these filters replaced.
func (filters Filters) MatchesDate(date string) bool {
	if date == "" {
		return true
	}
	if prefix := strings.TrimSpace(filters.TimePrefix); prefix != "" && !strings.HasPrefix(date, prefix) {
		return false
	}
	if begin := strings.TrimSpace(filters.TimeBegin); begin != "" && date < begin {
		return false
	}
	if end := strings.TrimSpace(filters.TimeEnd); end != "" && date >= end {
		return false
	}
	return true
}

// JournalFilters are the directive-level controls exposed by the journal
// view. Empty fields leave the corresponding dimension unfiltered.
type JournalFilters struct {
	Flag      string
	Tag       string
	Link      string
	Payee     string
	Narration string
	Kind      string
}

// Filter applies global account, FQL text, and ISO time-prefix filters while
// preserving the exact values and deterministic row order of the report. The
// text filter is parsed as Fava's filter query; rows are derived values, so
// the predicate evaluates against the row's own columns (see FQLTargetFromRow)
// and unparseable input degrades to the legacy case-insensitive substring so
// a caller that skipped validation never loses rows silently.
// filterContext is the parsed, per-dimension form of Filters: a dimension is
// inactive when its zero value makes the corresponding check pass trivially.
// Building it once keeps the row loop a single predicate call.
type filterContext struct {
	account string
	// dateCheck applies the time prefix/range; nil means no date filtering.
	dateCheck func(string) bool
	period    string
	anchor    string
	// textMatch applies the text filter; nil means no text filtering.
	textMatch func(query.Row) bool
}

// matches reports whether one row passes every active filter dimension.
func (c filterContext) matches(row query.Row) bool {
	if c.account != "" {
		value, _ := row["account"].(string)
		if value != c.account && !strings.HasPrefix(value, c.account+":") {
			return false
		}
	}
	if c.dateCheck != nil {
		if date, ok := row["date"].(string); ok && !c.dateCheck(date) {
			return false
		}
	}
	if c.period != "" {
		date, hasDate := row["date"].(string)
		if hasDate && !samePeriod(date, c.anchor, c.period) {
			return false
		}
	}
	return c.textMatch == nil || c.textMatch(row)
}

// Filter applies global account, FQL text, and ISO time-prefix filters while
// preserving the exact values and deterministic row order of the report. The
// text filter is parsed as Fava's filter query; rows are derived values, so
// the predicate evaluates against the row's own columns (see FQLTargetFromRow)
// and unparseable input degrades to the legacy case-insensitive substring so
// a caller that skipped validation never loses rows silently.
func Filter(result query.Result, filters Filters) query.Result {
	context, active := filterContextFor(result, filters)
	if !active {
		return result
	}
	filtered := query.Result{Columns: append([]string(nil), result.Columns...), Rows: make([]query.Row, 0, len(result.Rows))}
	for _, row := range result.Rows {
		if context.matches(row) {
			filtered.Rows = append(filtered.Rows, row)
		}
	}
	return filtered
}

// filterContextFor parses Filters into a filterContext and reports whether
// any dimension is active at all. The period dimension needs the latest date
// across the result rows as its comparison anchor.
// filterContextFor parses Filters into a filterContext and reports whether
// any dimension is active at all.
func filterContextFor(result query.Result, filters Filters) (filterContext, bool) {
	account := strings.TrimSpace(filters.Account)
	rawText := strings.TrimSpace(filters.Text)
	text := strings.ToLower(rawText)
	period := strings.ToLower(strings.TrimSpace(filters.Period))
	active := account != "" || text != "" || filters.TimePrefix != "" || filters.TimeBegin != "" || filters.TimeEnd != "" || period != ""
	context := filterContext{account: account, period: period, textMatch: textMatcher(rawText, text)}
	if active && (filters.TimePrefix != "" || filters.TimeBegin != "" || filters.TimeEnd != "") {
		context.dateCheck = filters.MatchesDate
	}
	if period != "" {
		context.anchor = latestDate(result.Rows)
	}
	return context, active
}

// textMatcher builds the row predicate for the text filter: the parsed FQL
// query when it parses, the legacy case-insensitive substring over the whole
// row otherwise, so unparseable input never silently loses rows.
func textMatcher(rawText, foldedText string) func(query.Row) bool {
	if foldedText == "" {
		return nil
	}
	textFilter, textErr := ParseFQL(rawText)
	if textFilter != nil && textErr == nil {
		return func(row query.Row) bool { return textFilter.Match(FQLTargetFromRow(row)) }
	}
	return func(row query.Row) bool { return strings.Contains(strings.ToLower(fmt.Sprint(row)), foldedText) }
}

// latestDate finds the maximal date column across rows ("" when absent);
// period filters compare against this anchor.
func latestDate(rows []query.Row) string {
	anchor := ""
	for _, row := range rows {
		if date, ok := row["date"].(string); ok && date > anchor {
			anchor = date
		}
	}
	return anchor
}

// samePeriod reports whether two dates fall in the same year, month, or
// quarter; missing dates never exclude a row.
func samePeriod(date, anchor, period string) bool {
	if date == "" || anchor == "" {
		return true
	}
	switch period {
	case "year":
		return len(date) >= 4 && len(anchor) >= 4 && date[:4] == anchor[:4]
	case "month":
		return len(date) >= 7 && len(anchor) >= 7 && date[:7] == anchor[:7]
	case "quarter":
		if len(date) < 7 || len(anchor) < 7 {
			return true
		}
		left, leftErr := strconv.Atoi(date[5:7])
		right, rightErr := strconv.Atoi(anchor[5:7])
		return leftErr != nil || rightErr != nil || (date[:4] == anchor[:4] && (left-1)/3 == (right-1)/3)
	default:
		return true
	}
}

// FilterJournal applies case-insensitive metadata filters to posting rows.
// It preserves row order and exact numeric values for CSV/export consumers.
func FilterJournal(result query.Result, filters JournalFilters) query.Result {
	context := newJournalFilterContext(filters)
	if !context.active {
		return result
	}
	filtered := query.Result{Columns: append([]string(nil), result.Columns...), Rows: make([]query.Row, 0, len(result.Rows))}
	for _, row := range result.Rows {
		if context.matches(row) {
			filtered.Rows = append(filtered.Rows, row)
		}
	}
	return filtered
}

// journalFilterContext is the lowercased, active-flagged form of
// JournalFilters.
type journalFilterContext struct {
	flag, kind, payee, narration, tag, link string
	active                                  bool
}

func newJournalFilterContext(filters JournalFilters) journalFilterContext {
	context := journalFilterContext{
		flag:      strings.TrimSpace(filters.Flag),
		kind:      strings.TrimSpace(filters.Kind),
		payee:     strings.ToLower(strings.TrimSpace(filters.Payee)),
		narration: strings.ToLower(strings.TrimSpace(filters.Narration)),
		tag:       strings.ToLower(strings.TrimSpace(filters.Tag)),
		link:      strings.ToLower(strings.TrimSpace(filters.Link)),
	}
	context.active = context.flag != "" || context.kind != "" || context.payee != "" || context.narration != "" || context.tag != "" || context.link != ""
	return context
}

// matches reports whether one posting row passes every active dimension.
func (c journalFilterContext) matches(row query.Row) bool {
	if c.flag != "" {
		if value, _ := row["flag"].(string); value != c.flag {
			return false
		}
	}
	if c.kind != "" && !strings.EqualFold(fmt.Sprint(row["kind"]), c.kind) {
		return false
	}
	if c.tag != "" && !containsString(row["tags"], c.tag) {
		return false
	}
	if c.link != "" && !containsString(row["links"], c.link) {
		return false
	}
	if c.payee != "" && !strings.Contains(strings.ToLower(fmt.Sprint(row["payee"])), c.payee) {
		return false
	}
	if c.narration != "" && !strings.Contains(strings.ToLower(fmt.Sprint(row["narration"])), c.narration) {
		return false
	}
	return true
}

func containsString(value any, needle string) bool {
	switch values := value.(type) {
	case []string:
		for _, item := range values {
			if strings.Contains(strings.ToLower(item), needle) {
				return true
			}
		}
	case []any:
		for _, item := range values {
			if strings.Contains(strings.ToLower(fmt.Sprint(item)), needle) {
				return true
			}
		}
	}
	return false
}

// All builds the full report set for one evaluation.
func All(evaluation ledger.Evaluation) Set {
	return Set{
		Accounts:        Accounts(evaluation),
		Journal:         Journal(evaluation),
		TrialBalance:    TrialBalance(evaluation),
		BalanceSheet:    BalanceSheet(evaluation),
		IncomeStatement: IncomeStatement(evaluation),
		Holdings:        Holdings(evaluation),
		Prices:          Prices(evaluation),
		Events:          Events(evaluation),
		Documents:       Documents(evaluation),
		Statistics:      Statistics(evaluation),
		Errors:          Errors(evaluation),
	}
}

// Accounts lists evaluated accounts with currencies, balances, and opening
// dates.
func Accounts(e ledger.Evaluation) query.Result {
	return evaluate("SELECT account, currency, balance, opened FROM accounts ORDER BY account, currency", e)
}

// Journal returns all journal postings in entry order.
func Journal(e ledger.Evaluation) query.Result {
	return JournalBetween(e, nil, nil)
}

// JournalBetween returns journal postings in the inclusive date range. A nil
// endpoint leaves that side unbounded; dates are already normalized by the
// API layer before reaching this function.
func JournalBetween(e ledger.Evaluation, from, to *ledger.Date) query.Result {
	// Keep posting metadata in the journal result so the browser can implement
	// Fava-style flag/tag/link filters and expandable transaction details
	// without reading source files or duplicating evaluator state.
	result := evaluate("SELECT date, account, units, currency, flag, payee, narration, tags, links, file, span, kind FROM postings ORDER BY date, account", e)
	if from == nil && to == nil {
		return result
	}
	rows := make([]query.Row, 0, len(result.Rows))
	for _, row := range result.Rows {
		value, _ := row["date"].(string)
		if from != nil && value < from.Raw {
			continue
		}
		if to != nil && value > to.Raw {
			continue
		}
		rows = append(rows, row)
	}
	result.Rows = rows
	return result
}

// Statistics summarizes the immutable evaluation for the statistics route.
// The rows are intentionally aggregate and deterministic; no source paths or
// ledger values are copied into the report metadata.
func Statistics(e ledger.Evaluation) query.Result {
	counts := map[string]int{}
	for _, entry := range e.Entries {
		if entry.Directive == nil {
			continue
		}
		counts[string(entry.Directive.Kind())]++
	}
	keys := make([]string, 0, len(counts))
	for kind := range counts {
		keys = append(keys, kind)
	}
	sort.Strings(keys)
	rows := make([]query.Row, 0, len(keys))
	for _, kind := range keys {
		rows = append(rows, query.Row{"directive": kind, "count": counts[kind]})
	}
	return query.Result{Columns: []string{"directive", "count"}, Rows: rows}
}

// PostingsPerAccount counts how many postings each account received, the
// counterpart of Fava's "Postings per Account" statistics section. Rows are
// ordered by count descending, then account, so the busiest accounts lead.
func PostingsPerAccount(e ledger.Evaluation) query.Result {
	counts := map[string]int{}
	for _, entry := range e.Entries {
		var transaction *ledger.Transaction
		switch value := entry.Directive.(type) {
		case ledger.Transaction:
			copy := value
			transaction = &copy
		case *ledger.Transaction:
			transaction = value
		}
		if transaction == nil {
			continue
		}
		for _, posting := range transaction.Postings {
			counts[posting.Account]++
		}
	}
	rows := make([]query.Row, 0, len(counts))
	for account, count := range counts {
		rows = append(rows, query.Row{"account": account, "postings": count})
	}
	sort.Slice(rows, func(i, j int) bool {
		left, right := rows[i]["postings"].(int), rows[j]["postings"].(int)
		if left != right {
			return left > right
		}
		return rows[i]["account"].(string) < rows[j]["account"].(string)
	})
	return query.Result{Columns: []string{"account", "postings"}, Rows: rows}
}

// TrialBalance sums balances per account/currency, one row each.
func TrialBalance(e ledger.Evaluation) query.Result {
	return evaluate("SELECT account, currency, sum(balance) AS balance FROM accounts GROUP BY account, currency ORDER BY account, currency", e)
}

// TrialBalanceTree returns the trial-balance account hierarchy used by the
// web report. TrialBalance intentionally remains the flat, query-compatible
// result for callers that need one row per evaluated account/currency; this
// variant adds explicit ancestors so indentation and collapse state are
// backed by real parent nodes.
func TrialBalanceTree(e ledger.Evaluation) query.Result {
	return accountRootReport(e)
}

// BalanceSheet returns the Assets/Liabilities/Equity hierarchy report.
func BalanceSheet(e ledger.Evaluation) query.Result {
	return accountRootReport(e, "Assets", "Liabilities", "Equity")
}

// IncomeStatement returns the Income/Expenses hierarchy report.
func IncomeStatement(e ledger.Evaluation) query.Result {
	return accountRootReport(e, "Income", "Expenses")
}

// Holdings returns every lot at cost with no as-of filtering.
func Holdings(e ledger.Evaluation) query.Result {
	return HoldingsAt(e, "", "at-cost")
}

// HoldingsAt applies an as-of date to surviving lots and optionally values
// them using the latest price quote at or before that date. The evaluator's
// immutable positions remain the source of truth; no floating-point or market
// data lookup is introduced.
func HoldingsAt(e ledger.Evaluation, asOf, valuation string) query.Result {
	return HoldingsAtCurrency(e, asOf, valuation, "")
}

// ValuedBalances returns one account's own balances converted according to
// valuation, the way Fava's account-tree reports present a commodity holding.
//
// "units" keeps every balance in its own commodity. "at-cost" replaces each lot
// with units*cost in the lot's cost currency. "market-value" uses the latest
// price-map quote instead, and leaves a lot in its own commodity when no quote
// exists rather than inventing one. Amounts that are not held at cost (plain
// currency balances) are never touched.
//
// Zero results are dropped so a fully converted commodity does not linger as a
// "0 FUND" column, matching Beancount inventory behavior.
func ValuedBalances(e ledger.Evaluation, account, valuation string) map[string]ledger.Decimal {
	values := make(map[string]ledger.Decimal)
	state, ok := e.Accounts[account]
	if !ok {
		return values
	}
	for currency, amount := range state.Balances {
		values[currency] = amount
	}
	if !strings.EqualFold(valuation, "units") {
		for _, position := range state.Positions {
			if position.Cost == nil || position.Cost.Currency == "" {
				continue
			}
			value, currency := position.Units.Mul(position.Cost.Number), position.Cost.Currency
			if strings.EqualFold(valuation, "market-value") {
				quote, found := latestQuote(e, position.Currency, "")
				if !found {
					continue
				}
				value, currency = position.Units.Mul(quote.Amount), quote.Currency
			}
			values[position.Currency] = values[position.Currency].Sub(position.Units)
			values[currency] = values[currency].Add(value)
		}
	}
	for currency, amount := range values {
		if amount.IsZero() {
			delete(values, currency)
		}
	}
	return values
}

// HoldingsAtCurrency is HoldingsAt with an optional presentation currency.
// Conversion uses only exact price-map quotes already present in the
// evaluation; a missing quote leaves the value in its native currency.
// HoldingsAtCurrency is HoldingsAt with an optional presentation currency.
// Conversion uses only exact price-map quotes already present in the
// evaluation; a missing quote leaves the value in its native currency.
func HoldingsAtCurrency(e ledger.Evaluation, asOf, valuation, displayCurrency string) query.Result {
	rows := make([]query.Row, 0)
	accounts := make([]string, 0, len(e.Accounts))
	for name := range e.Accounts {
		accounts = append(accounts, name)
	}
	sort.Strings(accounts)
	for _, name := range accounts {
		for _, position := range e.Accounts[name].Positions {
			if !positionSurvivesAsOf(position, asOf) {
				continue
			}
			rows = append(rows, holdingRow(e, name, position, asOf, valuation, displayCurrency))
		}
	}
	return query.Result{Columns: holdingsColumns(asOf, valuation), Rows: rows}
}

// positionSurvivesAsOf reports whether a cost-basis lot acquired after the
// as-of date is excluded from the holdings view (lots without cost dates
// always survive).
func positionSurvivesAsOf(position ledger.Position, asOf string) bool {
	if asOf == "" || position.Cost == nil || position.Cost.Date == nil {
		return true
	}
	return position.Cost.Date.Raw <= asOf
}

// holdingRow projects one position into a holdings row. Value columns are
// filled by the valuation selector: market-value via price quotes, at-cost
// via the lot's cost basis.
func holdingRow(e ledger.Evaluation, account string, position ledger.Position, asOf, valuation, displayCurrency string) query.Row {
	row := query.Row{"account": account, "currency": position.Currency, "units": position.Units}
	if asOf != "" {
		row["as_of_basis"] = "surviving-lots"
	}
	if position.Cost != nil {
		row["cost_currency"] = position.Cost.Currency
		row["cost"] = position.Cost.Number
		row["cost_label"] = position.Cost.Label
	}
	if value, currency, status, ok := holdingValue(e, position, asOf, valuation, displayCurrency); ok {
		row["value"] = value
		row["value_currency"] = currency
		row["valuation_status"] = status
	}
	return row
}

// holdingValue computes the (value, value currency, status) triple for one
// position under the valuation selector, converting into the display
// currency when an exact quote exists. ok is false when the valuation adds
// no value columns (plain at-cost without display conversion input).
func holdingValue(e ledger.Evaluation, position ledger.Position, asOf, valuation, displayCurrency string) (any, string, string, bool) {
	if strings.EqualFold(valuation, "market-value") {
		// Market valuation always contributes the value triple; a missing
		// quote is expressed as a nil value with an unavailable-price status
		// rather than by omitting the columns.
		quote, ok := latestQuote(e, position.Currency, asOf)
		if !ok {
			return nil, "", "unavailable-price", true
		}
		return marketHoldingValue(e, position, quote, asOf, displayCurrency)
	}
	if position.Cost == nil {
		return nil, "", "", false
	}
	value := position.Units.Mul(position.Cost.Number)
	currency := position.Cost.Currency
	if displayCurrency != "" && !strings.EqualFold(currency, displayCurrency) {
		if conversion, found := latestQuote(e, currency, asOf); found && strings.EqualFold(conversion.Currency, displayCurrency) {
			value = value.Mul(conversion.Amount)
			currency = conversion.Currency
		}
	}
	return value, currency, "at-cost", true
}

// marketHoldingValue values a position at a known quote and converts into
// the display currency when another exact quote exists; a missing
// conversion quote is reported as a status, never as an invented number.
func marketHoldingValue(e ledger.Evaluation, position ledger.Position, quote ledger.PriceQuote, asOf, displayCurrency string) (any, string, string, bool) {
	value := position.Units.Mul(quote.Amount)
	valueCurrency := quote.Currency
	status := "market"
	if displayCurrency != "" && !strings.EqualFold(valueCurrency, displayCurrency) {
		if conversion, found := latestQuote(e, valueCurrency, asOf); found && strings.EqualFold(conversion.Currency, displayCurrency) {
			value = value.Mul(conversion.Amount)
			valueCurrency = conversion.Currency
		} else {
			status = "unavailable-currency"
		}
	}
	return value, valueCurrency, status, true
}

// holdingsColumns picks the result columns: the value triple appears once a
// valuation or as-of view is requested, and the as-of basis column only in
// as-of views.
func holdingsColumns(asOf, valuation string) []string {
	columns := []string{"account", "currency", "units", "cost_currency", "cost", "cost_label"}
	if asOf != "" || !strings.EqualFold(valuation, "at-cost") {
		columns = append(columns, "value", "value_currency", "valuation_status")
		if asOf != "" {
			columns = append(columns, "as_of_basis")
		}
	}
	return columns
}

// HoldingsAggregate collapses flat holdings rows the way Fava's holdings-by
// views do: units are summed per group, and book value is summed only within
// one cost currency so cross-currency totals are never invented. "all" (or an
// unknown key) returns the input untouched.
// holdingsGroup accumulates one aggregation bucket: summed units and summed
// book value. Book value is only summed within a single cost currency;
// bookMixed marks buckets whose cost currencies differ so no cross-currency
// total is ever invented.
type holdingsGroup struct {
	units        ledger.Decimal
	book         ledger.Decimal
	bookCurrency string
	bookMixed    bool
	hasCost      bool
}

// absorb folds one holdings row into the bucket.
func (g *holdingsGroup) absorb(row query.Row) {
	units, ok := row["units"].(ledger.Decimal)
	if !ok {
		return
	}
	g.units = g.units.Add(units)
	cost, hasCost := row["cost"].(ledger.Decimal)
	if !hasCost {
		return
	}
	g.book = g.book.Add(units.Mul(cost))
	g.hasCost = true
	costCurrency := asString(row["cost_currency"])
	switch {
	case g.bookCurrency == "":
		g.bookCurrency = costCurrency
	case g.bookCurrency != costCurrency:
		g.bookMixed = true
	}
}

// row renders the bucket as one output row: the key columns lead, followed
// by units, average cost, and book value (the latter two only for unmixed
// cost bases with a non-zero unit count).
func (g *holdingsGroup) row(columns []string, keyParts []string) query.Row {
	row := query.Row{}
	for index, column := range keyParts {
		row[columns[index]] = column
	}
	row["units"] = g.units
	if g.hasCost && !g.bookMixed {
		row["book_value"] = g.book
		if !g.units.IsZero() {
			row["average_cost"] = g.book.Quo(g.units)
		}
	}
	return row
}

// HoldingsAggregate collapses flat holdings rows the way Fava's holdings-by
// views do: units are summed per group, and book value is summed only within
// one cost currency so cross-currency totals are never invented. "all" (or an
// unknown key) returns the input untouched.
func HoldingsAggregate(result query.Result, aggregation string) query.Result {
	columns, keyOf, ok := aggregationPlan(aggregation)
	if !ok {
		return result
	}
	groups := map[string]*holdingsGroup{}
	keys := map[string][]string{}
	order := []string{}
	for _, row := range result.Rows {
		keyParts := keyOf(row)
		id := strings.Join(keyParts, "\x00")
		aggregate, found := groups[id]
		if !found {
			aggregate = &holdingsGroup{}
			groups[id] = aggregate
			keys[id] = keyParts
			order = append(order, id)
		}
		aggregate.absorb(row)
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	rows := make([]query.Row, 0, len(order))
	for _, id := range order {
		rows = append(rows, groups[id].row(columns, keys[id]))
	}
	return query.Result{Columns: columns, Rows: rows}
}

// aggregationPlan maps an aggregation key to its output columns and group
// key extractor; ok is false for "all" and unknown keys.
func aggregationPlan(aggregation string) (columns []string, keyOf func(query.Row) []string, ok bool) {
	switch aggregation {
	case "by_account":
		return []string{"account", "currency", "cost_currency", "units", "average_cost", "book_value"},
			func(row query.Row) []string {
				return []string{asString(row["account"]), asString(row["currency"]), asString(row["cost_currency"])}
			}, true
	case "by_currency":
		return []string{"currency", "cost_currency", "units", "average_cost", "book_value"},
			func(row query.Row) []string {
				return []string{asString(row["currency"]), asString(row["cost_currency"])}
			}, true
	case "by_root_account":
		return []string{"root_account", "currency", "cost_currency", "units", "average_cost", "book_value"},
			func(row query.Row) []string {
				account := asString(row["account"])
				if index := strings.Index(account, ":"); index > 0 {
					account = account[:index]
				}
				return []string{account, asString(row["currency"]), asString(row["cost_currency"])}
			}, true
	case "by_cost_currency":
		return []string{"cost_currency", "units", "average_cost", "book_value"},
			func(row query.Row) []string { return []string{asString(row["cost_currency"])} }, true
	case "by_commodity":
		return []string{"currency", "units", "average_cost", "book_value"},
			func(row query.Row) []string { return []string{asString(row["currency"])} }, true
	default:
		return nil, nil, false
	}
}

func asString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func latestQuote(e ledger.Evaluation, base, asOf string) (ledger.PriceQuote, bool) {
	quotes := e.Prices[base]
	var latest ledger.PriceQuote
	found := false
	for _, quote := range quotes {
		if asOf != "" && quote.Date.Raw > asOf {
			continue
		}
		if !found || quote.Date.Raw > latest.Date.Raw {
			latest, found = quote, true
		}
	}
	return latest, found
}

// Prices lists price directives in date order.
func Prices(e ledger.Evaluation) query.Result {
	return evaluate("SELECT date, currency, amount, quote_currency FROM prices ORDER BY date, currency", e)
}

// Events lists event directives in entry order.
func Events(e ledger.Evaluation) query.Result {
	rows := make([]query.Row, 0)
	for _, entry := range e.Entries {
		if event, ok := entry.Directive.(ledger.Event); ok {
			rows = append(rows, query.Row{"date": event.Date.Raw, "type": event.Type, "value": event.Value, "file": entry.File, "span": entry.Span.String()})
		}
	}
	return query.Result{Columns: []string{"date", "type", "value", "file", "span"}, Rows: rows}
}

// Documents lists document attachments expanded to one row per file.
func Documents(e ledger.Evaluation) query.Result {
	rows := make([]query.Row, 0)
	for _, entry := range e.Entries {
		if document, ok := entry.Directive.(ledger.Document); ok {
			for _, filename := range document.Filenames {
				rows = append(rows, query.Row{"date": document.Date.Raw, "account": document.Account, "filename": filename, "tags": append([]string(nil), document.Tags...), "links": append([]string(nil), document.Links...), "file": entry.File, "span": entry.Span.String()})
			}
		}
	}
	return query.Result{Columns: []string{"date", "account", "filename", "tags", "links", "file", "span"}, Rows: rows}
}

// Errors lists the evaluation's error diagnostics as rows.
func Errors(e ledger.Evaluation) query.Result {
	return errors(e, nil)
}

// ErrorsWithGraph renders diagnostic paths using the safe display identifiers
// of the current include graph. It is the API-facing variant; Errors remains
// useful for graph-independent callers and still strips absolute paths.
func ErrorsWithGraph(e ledger.Evaluation, graph *source.Graph) query.Result {
	return errors(e, graph)
}

func errors(e ledger.Evaluation, graph *source.Graph) query.Result {
	rows := make([]query.Row, 0, len(e.Diagnostics))
	for _, diagnostic := range e.Diagnostics {
		path := source.SafeDisplayPath(diagnostic.Path)
		if graph != nil {
			if graph.File(diagnostic.Span.File) != nil {
				path = graph.DisplayPath(diagnostic.Span.File)
			} else if id, ok := graph.ByPath[diagnostic.Path]; ok {
				path = graph.DisplayPath(id)
			}
		}
		rows = append(rows, query.Row{"code": diagnostic.Code, "severity": diagnostic.Severity, "path": path, "line": diagnostic.Span.StartLine, "column": diagnostic.Span.StartColumn, "message": diagnostic.Message})
	}
	return query.Result{Columns: []string{"code", "severity", "path", "line", "column", "message"}, Rows: rows}
}

// accountRootReport renders one account subtree as report tree rows. The
// index construction (see accountTreeIndex) and the row rendering are split
// so each stays readable; the invariants are documented there.
func accountRootReport(e ledger.Evaluation, roots ...string) query.Result {
	index := buildAccountTreeIndex(e, roots)
	names := make([]string, 0, len(index.nodes))
	for name := range index.nodes {
		names = append(names, name)
	}
	sort.Strings(names)
	rows := make([]query.Row, 0, len(names))
	for _, name := range names {
		rows = append(rows, accountTreeRows(name, index)...)
	}
	return query.Result{Columns: []string{"account", "currency", "balance", "own_balance", "total_balance", "_tree_depth", "_tree_parent", "_tree_has_child", "_tree_role", "_tree_has_direct", "_tree_explicit"}, Rows: rows}
}

// accountTreeIndex aggregates the accounts of the (optionally root-filtered)
// evaluation into tree-shaped totals.
//
// Reports are trees, not a flat collection of account names: every ancestor
// is built explicitly and aggregates its descendants by commodity, so
// indentation is backed by an actual parent node even when a ledger only
// ever posts to a deep leaf account. Whether a node corresponds to an opened
// account is kept separate from whether its balance is an aggregate:
// synthetic ancestors are useful for presentation but must never be mistaken
// for account state (or included in account/CSV leaf assertions). totals
// holds each node's aggregate balance (itself plus all descendants) while
// ownTotals holds only the balance posted directly to that node; keeping the
// two separate means a parent-with-direct-postings is still distinguishable
// from its leaf children and the parent's own balance is never re-counted as
// a separate ordinary balance.
type accountTreeIndex struct {
	totals    map[string]map[string]ledger.Decimal
	ownTotals map[string]map[string]ledger.Decimal
	explicit  map[string]bool
	direct    map[string]bool
	nodes     map[string]bool
	children  map[string]map[string]bool
}

// buildAccountTreeIndex builds the subtree index for the accounts whose root is
// listed in roots (all accounts when roots is empty).
func buildAccountTreeIndex(e ledger.Evaluation, roots []string) accountTreeIndex {
	allowed := make(map[string]bool, len(roots))
	for _, root := range roots {
		allowed[root] = true
	}
	index := accountTreeIndex{
		totals:    map[string]map[string]ledger.Decimal{},
		ownTotals: map[string]map[string]ledger.Decimal{},
		explicit:  make(map[string]bool, len(e.Accounts)),
		direct:    make(map[string]bool, len(e.Accounts)),
		nodes:     map[string]bool{},
		children:  map[string]map[string]bool{},
	}
	for name, state := range e.Accounts {
		if !accountAllowed(name, allowed) {
			continue
		}
		index.explicit[name] = true
		index.direct[name] = len(state.Balances) != 0
		for node := name; node != ""; node = accountParent(node) {
			index.nodes[node] = true
			if parent := accountParent(node); parent != "" {
				if index.children[parent] == nil {
					index.children[parent] = map[string]bool{}
				}
				index.children[parent][node] = true
			}
		}
		for currency, balance := range state.Balances {
			addTreeBalance(index.totals, name, currency, balance, true)
			addTreeBalance(index.ownTotals, name, currency, balance, false)
		}
	}
	return index
}

// accountAllowed checks the account's root against the allowed set (an
// empty set allows everything).
func accountAllowed(name string, allowed map[string]bool) bool {
	if len(allowed) == 0 {
		return true
	}
	root := name
	if colon := strings.IndexByte(name, ':'); colon >= 0 {
		root = name[:colon]
	}
	return allowed[root]
}

// addTreeBalance adds one balance to a node's own totals and, when
// propagate is set, to every ancestor's aggregate totals.
func addTreeBalance(totals map[string]map[string]ledger.Decimal, name, currency string, balance ledger.Decimal, propagate bool) {
	for node := name; node != ""; node = accountParent(node) {
		if totals[node] == nil {
			totals[node] = map[string]ledger.Decimal{}
		}
		totals[node][currency] = totals[node][currency].Add(balance)
		if !propagate {
			return
		}
	}
}

// accountTreeRows renders one tree node as one row per currency. An empty
// currency is the zero marker used by accountRows and avoids inventing a
// commodity that was never present in the balance; explicitly opened
// accounts (and synthetic ancestors with no ending balance) stay visible.
func accountTreeRows(name string, index accountTreeIndex) []query.Row {
	currencies := make([]string, 0, len(index.totals[name]))
	for currency := range index.totals[name] {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	if len(currencies) == 0 {
		currencies = []string{""}
	}
	rows := make([]query.Row, 0, len(currencies))
	for _, currency := range currencies {
		role := "direct"
		if len(index.children[name]) > 0 {
			role = "aggregate"
		}
		rows = append(rows, query.Row{
			"account":          name,
			"currency":         currency,
			"balance":          treeTotal(index.totals, name, currency),
			"own_balance":      treeTotal(index.ownTotals, name, currency),
			"total_balance":    treeTotal(index.totals, name, currency),
			"_tree_depth":      strings.Count(name, ":"),
			"_tree_parent":     accountParent(name),
			"_tree_has_child":  len(index.children[name]) > 0,
			"_tree_role":       role,
			"_tree_has_direct": index.direct[name],
			"_tree_explicit":   index.explicit[name],
		})
	}
	return rows
}

// treeTotal reads one node's total for a currency, defaulting to zero.
func treeTotal(totals map[string]map[string]ledger.Decimal, name, currency string) ledger.Decimal {
	if value, ok := totals[name][currency]; ok {
		return value
	}
	return ledger.Zero()
}

func accountParent(name string) string {
	if index := strings.LastIndexByte(name, ':'); index >= 0 {
		return name[:index]
	}
	return ""
}

func evaluate(text string, evaluation ledger.Evaluation) query.Result {
	result, err := query.Evaluate(text, evaluation)
	if err != nil {
		return query.Result{Columns: []string{"error"}, Rows: []query.Row{{"error": err.Error()}}}
	}
	return result
}
