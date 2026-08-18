// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package report

import (
	"sort"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
)

// PivotSpec selects one pivot report: how rows bucket time (month, quarter,
// year), how columns group postings (none means one column per currency,
// rootN groups by the first N account components), which value to show
// (period sums or carried ending balances), and the account subtree plus
// filters every posting must match.
type PivotSpec struct {
	Rows     string // "month" | "quarter" | "year"
	Columns  string // "" | "root1" | "root2" | "root3"
	Values   string // "sum" | "balance"
	Account  string
	Filters  Filters
}

// PivotTable renders the Excel-style cross-tab: one row per interval, one
// column per (column-group, currency) pair. Sums are per-interval posting
// totals; balances carry across intervals like the interval tables, so quiet
// periods repeat the standing balance instead of reading as zero. Nothing is
// converted across currencies — each currency keeps its own column, the
// honest presentation for a ledger without price directives.
func PivotTable(e ledger.Evaluation, spec PivotSpec) query.Result {
	interval := normalizeChartPeriod(mapPivotRows(spec.Rows))
	if interval == "" {
		interval = "month"
	}
	textFilter, textErr := ParseFQL(strings.TrimSpace(spec.Filters.Text))
	text := strings.ToLower(strings.TrimSpace(spec.Filters.Text))
	totals := map[string]map[string]ledger.Decimal{}
	keys := map[string]bool{}
	denominator := columnCurrencyLabeler(spec.Columns)
	for _, posting := range transactionPostings(e) {
		if !spec.Filters.MatchesDate(posting.date) {
			continue
		}
		// An empty account prefix means every account: the pivot is the whole
		// ledger unless the user narrows it (withinAccountSubtree alone treats
		// "" as matching nothing).
		if spec.Account != "" && !withinAccountSubtree(posting.account, spec.Account) {
			continue
		}
		if !intervalPostingMatchesText(posting, text, textFilter, textErr) {
			continue
		}
		key := chartPeriodKey(posting.date, interval)
		if key == "" {
			continue
		}
		column := pivotColumnLabel(denominator(posting.account), posting.currency)
		if totals[key] == nil {
			totals[key] = map[string]ledger.Decimal{}
		}
		totals[key][column] = totals[key][column].Add(posting.amount)
		keys[key] = true
	}
	if len(keys) == 0 {
		return query.Result{Columns: []string{"interval"}}
	}
	ordered := sortedKeys(totals)
	columns := columnSet(totals)
	result := query.Result{Columns: append([]string{"interval"}, columns...)}
	running := map[string]ledger.Decimal{}
	for key := ordered[0]; ; key = nextPeriodKey(key, interval) {
		row := query.Row{"interval": key}
		for _, column := range columns {
			amount := ledger.Zero()
			if totals[key] != nil {
				amount = totals[key][column]
			}
			if spec.Values == "balance" {
				running[column] = running[column].Add(amount)
				amount = running[column]
			}
			row[column] = amount
		}
		result.Rows = append(result.Rows, row)
		if key == ordered[len(ordered)-1] {
			break
		}
	}
	return result
}

// pivotColumnLabel joins a column-group prefix and a currency; without a
// prefix the currency alone is the label.
func pivotColumnLabel(group, currency string) string {
	if group == "" {
		return currency
	}
	return group + " " + currency
}

// mapPivotRows normalizes a row selector to an interval name, defaulting to
// months for anything unrecognized.
func mapPivotRows(rows string) string {
	switch strings.TrimSpace(rows) {
	case "quarter":
		return "quarter"
	case "year":
		return "year"
	default:
		return "month"
	}
}

// columnCurrencyLabeler builds the column key prefix for an account: the
// empty prefix yields pure per-currency columns; rootN truncates the account
// to its first N components.
func columnCurrencyLabeler(columns string) func(string) string {
	depth := 0
	switch strings.TrimSpace(columns) {
	case "root1":
		depth = 1
	case "root2":
		depth = 2
	case "root3":
		depth = 3
	}
	if depth == 0 {
		return func(string) string { return "" }
	}
	return func(account string) string {
		parts := strings.Split(account, ":")
		if depth >= len(parts) {
			return account
		}
		return strings.Join(parts[:depth], ":")
	}
}

// columnSet collects the sorted (column-group, currency) labels seen across
// intervals so every row carries the full cross-tab shape.
func columnSet(totals map[string]map[string]ledger.Decimal) []string {
	seen := map[string]bool{}
	for _, byColumn := range totals {
		for column := range byColumn {
			seen[column] = true
		}
	}
	columns := make([]string, 0, len(seen))
	for column := range seen {
		columns = append(columns, column)
	}
	sort.Strings(columns)
	return columns
}
