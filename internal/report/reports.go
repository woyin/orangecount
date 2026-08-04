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
	Period     string
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

// Filter applies global account, free-text, and ISO time-prefix filters while
// preserving the exact values and deterministic row order of the report.
func Filter(result query.Result, filters Filters) query.Result {
	account := strings.TrimSpace(filters.Account)
	text := strings.ToLower(strings.TrimSpace(filters.Text))
	prefix := strings.TrimSpace(filters.TimePrefix)
	period := strings.ToLower(strings.TrimSpace(filters.Period))
	if account == "" && text == "" && prefix == "" && period == "" {
		return result
	}
	anchor := ""
	if period != "" {
		for _, row := range result.Rows {
			if date, ok := row["date"].(string); ok && date > anchor {
				anchor = date
			}
		}
	}
	filtered := query.Result{Columns: append([]string(nil), result.Columns...), Rows: make([]query.Row, 0, len(result.Rows))}
	for _, row := range result.Rows {
		if account != "" {
			value, _ := row["account"].(string)
			if value != account && !strings.HasPrefix(value, account+":") {
				continue
			}
		}
		if prefix != "" {
			if date, ok := row["date"].(string); ok && !strings.HasPrefix(date, prefix) {
				continue
			}
		}
		if period != "" {
			date, hasDate := row["date"].(string)
			if hasDate && !samePeriod(date, anchor, period) {
				continue
			}
		}
		if text != "" && !strings.Contains(strings.ToLower(fmt.Sprint(row)), text) {
			continue
		}
		filtered.Rows = append(filtered.Rows, row)
	}
	return filtered
}

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
	flag := strings.TrimSpace(filters.Flag)
	tag := strings.ToLower(strings.TrimSpace(filters.Tag))
	link := strings.ToLower(strings.TrimSpace(filters.Link))
	payee := strings.ToLower(strings.TrimSpace(filters.Payee))
	narration := strings.ToLower(strings.TrimSpace(filters.Narration))
	kind := strings.TrimSpace(filters.Kind)
	if flag == "" && tag == "" && link == "" && payee == "" && narration == "" && kind == "" {
		return result
	}
	filtered := query.Result{Columns: append([]string(nil), result.Columns...), Rows: make([]query.Row, 0, len(result.Rows))}
	for _, row := range result.Rows {
		if flag != "" {
			value, _ := row["flag"].(string)
			if value != flag {
				continue
			}
		}
		if kind != "" {
			value := fmt.Sprint(row["kind"])
			if !strings.EqualFold(value, kind) {
				continue
			}
		}
		if tag != "" && !containsString(row["tags"], tag) {
			continue
		}
		if link != "" && !containsString(row["links"], link) {
			continue
		}
		if payee != "" && !strings.Contains(strings.ToLower(fmt.Sprint(row["payee"])), payee) {
			continue
		}
		if narration != "" && !strings.Contains(strings.ToLower(fmt.Sprint(row["narration"])), narration) {
			continue
		}
		filtered.Rows = append(filtered.Rows, row)
	}
	return filtered
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

func Accounts(e ledger.Evaluation) query.Result {
	return evaluate("SELECT account, currency, balance, opened FROM accounts ORDER BY account, currency", e)
}

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

func BalanceSheet(e ledger.Evaluation) query.Result {
	return accountRootReport(e, "Assets", "Liabilities", "Equity")
}

func IncomeStatement(e ledger.Evaluation) query.Result {
	return accountRootReport(e, "Income", "Expenses")
}

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
			if asOf != "" && position.Cost != nil && position.Cost.Date != nil && position.Cost.Date.Raw > asOf {
				continue
			}
			row := query.Row{"account": name, "currency": position.Currency, "units": position.Units}
			if asOf != "" {
				row["as_of_basis"] = "surviving-lots"
			}
			if position.Cost != nil {
				row["cost_currency"] = position.Cost.Currency
				row["cost"] = position.Cost.Number
				row["cost_label"] = position.Cost.Label
			}
			if strings.EqualFold(valuation, "market-value") {
				if quote, ok := latestQuote(e, position.Currency, asOf); ok {
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
					row["value"] = value
					row["value_currency"] = valueCurrency
					row["valuation_status"] = status
				} else {
					row["value"] = nil
					row["valuation_status"] = "unavailable-price"
				}
			} else if position.Cost != nil {
				value := position.Units.Mul(position.Cost.Number)
				valueCurrency := position.Cost.Currency
				if displayCurrency != "" && !strings.EqualFold(valueCurrency, displayCurrency) {
					if conversion, found := latestQuote(e, valueCurrency, asOf); found && strings.EqualFold(conversion.Currency, displayCurrency) {
						value = value.Mul(conversion.Amount)
						valueCurrency = conversion.Currency
					}
				}
				row["value"] = value
				row["value_currency"] = valueCurrency
				row["valuation_status"] = "at-cost"
			}
			rows = append(rows, row)
		}
	}
	columns := []string{"account", "currency", "units", "cost_currency", "cost", "cost_label"}
	if asOf != "" || !strings.EqualFold(valuation, "at-cost") {
		columns = append(columns, "value", "value_currency", "valuation_status")
		if asOf != "" {
			columns = append(columns, "as_of_basis")
		}
	}
	return query.Result{Columns: columns, Rows: rows}
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

func Prices(e ledger.Evaluation) query.Result {
	return evaluate("SELECT date, currency, amount, quote_currency FROM prices ORDER BY date, currency", e)
}

func Events(e ledger.Evaluation) query.Result {
	rows := make([]query.Row, 0)
	for _, entry := range e.Entries {
		if event, ok := entry.Directive.(ledger.Event); ok {
			rows = append(rows, query.Row{"date": event.Date.Raw, "type": event.Type, "value": event.Value, "file": entry.File, "span": entry.Span.String()})
		}
	}
	return query.Result{Columns: []string{"date", "type", "value", "file", "span"}, Rows: rows}
}

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

func accountRootReport(e ledger.Evaluation, roots ...string) query.Result {
	allowed := make(map[string]bool, len(roots))
	for _, root := range roots {
		allowed[root] = true
	}

	// Reports are trees, not a flat collection of account names. Build every
	// ancestor explicitly and aggregate its descendants by commodity. This
	// means indentation is backed by an actual parent node even when a ledger
	// only ever posts to a deep leaf account. Keep whether a node corresponds
	// to an opened account separate from whether its balance is an aggregate:
	// synthetic ancestors are useful for presentation, but must never be
	// mistaken for account state (or included in account/CSV leaf assertions).
	// totals holds each node's aggregate balance (itself plus all descendants);
	// ownTotals holds only the balance posted directly to that node. Keeping the
	// two separate means a synthetic ancestor or a parent-with-direct-postings is
	// still distinguishable from its leaf children: the parent's own balance must
	// not be re-counted as if it were a separate ordinary balance.
	totals := map[string]map[string]ledger.Decimal{}
	ownTotals := map[string]map[string]ledger.Decimal{}
	explicit := make(map[string]bool, len(e.Accounts))
	direct := make(map[string]bool, len(e.Accounts))
	nodes := make(map[string]bool)
	children := map[string]map[string]bool{}
	for name, state := range e.Accounts {
		root := name
		if colon := strings.IndexByte(name, ':'); colon >= 0 {
			root = name[:colon]
		}
		if len(allowed) != 0 && !allowed[root] {
			continue
		}
		explicit[name] = true
		direct[name] = len(state.Balances) != 0
		for node := name; node != ""; node = accountParent(node) {
			nodes[node] = true
			parent := accountParent(node)
			if parent != "" {
				if children[parent] == nil {
					children[parent] = map[string]bool{}
				}
				children[parent][node] = true
			}
		}
		for currency, balance := range state.Balances {
			for node := name; node != ""; node = accountParent(node) {
				if totals[node] == nil {
					totals[node] = map[string]ledger.Decimal{}
				}
				totals[node][currency] = totals[node][currency].Add(balance)
			}
			if ownTotals[name] == nil {
				ownTotals[name] = map[string]ledger.Decimal{}
			}
			ownTotals[name][currency] = ownTotals[name][currency].Add(balance)
		}
	}
	names := make([]string, 0, len(nodes))
	for name := range nodes {
		names = append(names, name)
	}
	sort.Strings(names)
	rows := make([]query.Row, 0, len(names))
	for _, name := range names {
		currencies := make([]string, 0, len(totals[name]))
		for currency := range totals[name] {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)
		if len(currencies) == 0 {
			// Keep explicitly opened accounts (and synthetic ancestors whose
			// descendants have no ending balance) in the tree. An empty
			// currency is the same zero marker used by accountRows and avoids
			// inventing a commodity that was never present in the balance.
			currencies = []string{""}
		}
		for _, currency := range currencies {
			role := "direct"
			if len(children[name]) > 0 {
				role = "aggregate"
			}
			balance := ledger.Zero()
			if value, ok := totals[name][currency]; ok {
				balance = value
			}
			own := ledger.Zero()
			if value, ok := ownTotals[name][currency]; ok {
				own = value
			}
			rows = append(rows, query.Row{
				"account":          name,
				"currency":         currency,
				"balance":          balance,
				"own_balance":      own,
				"total_balance":    balance,
				"_tree_depth":      strings.Count(name, ":"),
				"_tree_parent":     accountParent(name),
				"_tree_has_child":  len(children[name]) > 0,
				"_tree_role":       role,
				"_tree_has_direct": direct[name],
				"_tree_explicit":   explicit[name],
			})
		}
	}
	return query.Result{Columns: []string{"account", "currency", "balance", "own_balance", "total_balance", "_tree_depth", "_tree_parent", "_tree_has_child", "_tree_role", "_tree_has_direct", "_tree_explicit"}, Rows: rows}
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
