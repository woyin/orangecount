// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package report

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
)

// AccountIntervals aggregates the postings of one account's subtree into
// per-interval totals. Mode "changes" reports each interval's posting sum;
// mode "balances" reports the running total carried across intervals. Rows
// cover every interval from the first activity to the last so quiet periods
// keep their carried balance visible, matching Fava's interval tables.
// AccountIntervals aggregates the postings of one account's subtree into
// per-interval totals. Mode "changes" reports each interval's posting sum;
// mode "balances" reports the running total carried across intervals. Rows
// cover every interval from the first activity to the last so quiet periods
// keep their carried balance visible, matching Fava's interval tables.
func AccountIntervals(e ledger.Evaluation, account, mode, interval string, filters Filters) query.Result {
	columns := []string{"interval"}
	account = strings.TrimSpace(account)
	if (mode != "changes" && mode != "balances") || account == "" {
		return query.Result{Columns: columns}
	}
	interval = normalizeChartPeriod(interval)
	totals, currencies := intervalTotals(e, account, interval, filters)
	if len(totals) == 0 {
		return query.Result{Columns: columns}
	}
	keys := sortedKeys(totals)
	return query.Result{
		Columns: append(columns, currencies...),
		Rows:    intervalTableRows(totals, keys, currencies, mode, interval),
	}
}

// intervalTotals sums the account subtree's postings per interval key and
// collects the currencies seen. The advanced text filter is applied to each
// transaction posting, matching Fava's filtered ledger view that interval
// tables aggregate over; unparseable text degrades to a substring match.
func intervalTotals(e ledger.Evaluation, account, interval string, filters Filters) (map[string]map[string]ledger.Decimal, []string) {
	textFilter, textErr := ParseFQL(strings.TrimSpace(filters.Text))
	text := strings.ToLower(strings.TrimSpace(filters.Text))
	totals := map[string]map[string]ledger.Decimal{}
	currencySet := map[string]bool{}
	for _, posting := range transactionPostings(e) {
		if !filters.MatchesDate(posting.date) {
			continue
		}
		if !withinAccountSubtree(posting.account, account) || !intervalPostingMatchesText(posting, text, textFilter, textErr) {
			continue
		}
		key := chartPeriodKey(posting.date, interval)
		if key == "" {
			continue
		}
		if totals[key] == nil {
			totals[key] = map[string]ledger.Decimal{}
		}
		totals[key][posting.currency] = totals[key][posting.currency].Add(posting.amount)
		currencySet[posting.currency] = true
	}
	currencies := make([]string, 0, len(currencySet))
	for currency := range currencySet {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	return totals, currencies
}

// intervalPostingMatchesText applies the text filter to one posting: the
// parsed FQL predicate when available, the legacy substring otherwise.
func intervalPostingMatchesText(posting chartPosting, text string, textFilter *FQL, textErr error) bool {
	if text == "" {
		return true
	}
	if textErr == nil {
		return textFilter.Match(fqlTargetFromChartPosting(posting))
	}
	return strings.Contains(strings.ToLower(posting.narration+" "+posting.payee+" "+posting.account), text)
}

// withinAccountSubtree reports whether an account is the node itself or a
// descendant of it (colon-separated name hierarchy).
func withinAccountSubtree(name, account string) bool {
	return name == account || strings.HasPrefix(name, account+":")
}

// intervalRows renders one row per interval from the first activity to the
// last, filling quiet periods with zero changes (balances mode carries the
// running total across them).
func intervalTableRows(totals map[string]map[string]ledger.Decimal, keys, currencies []string, mode, interval string) []query.Row {
	rows := make([]query.Row, 0, len(keys))
	running := map[string]ledger.Decimal{}
	for key := keys[0]; ; key = nextPeriodKey(key, interval) {
		row := query.Row{"interval": key}
		for _, currency := range currencies {
			amount := ledger.Zero()
			if totals[key] != nil {
				amount = totals[key][currency]
			}
			if mode == "balances" {
				running[currency] = running[currency].Add(amount)
				amount = running[currency]
			}
			row[currency] = amount
		}
		rows = append(rows, row)
		if key == keys[len(keys)-1] {
			break
		}
	}
	return rows
}

// nextPeriodKey advances a period bucket key ("2026", "2026-Q2", "2026-03")
// by one interval, passing malformed keys through unchanged.
func nextPeriodKey(key, interval string) string {
	switch interval {
	case "year":
		year, err := strconv.Atoi(key)
		if err != nil {
			return key
		}
		return strconv.Itoa(year + 1)
	case "quarter":
		parts := strings.Split(key, "-Q")
		if len(parts) != 2 {
			return key
		}
		year, yearErr := strconv.Atoi(parts[0])
		quarter, quarterErr := strconv.Atoi(parts[1])
		if yearErr != nil || quarterErr != nil {
			return key
		}
		if quarter >= 4 {
			return strconv.Itoa(year+1) + "-Q1"
		}
		return parts[0] + "-Q" + strconv.Itoa(quarter+1)
	default:
		parts := strings.Split(key, "-")
		if len(parts) != 2 {
			return key
		}
		year, yearErr := strconv.Atoi(parts[0])
		month, monthErr := strconv.Atoi(parts[1])
		if yearErr != nil || monthErr != nil {
			return key
		}
		if month >= 12 {
			return strconv.Itoa(year+1) + "-01"
		}
		return fmt.Sprintf("%s-%02d", parts[0], month+1)
	}
}

// fqlTargetFromChartPosting builds an FQL target from a chart posting so the
// advanced text filter can match its transaction attributes (payee, narration,
// tags, links, account, flag, date).
func fqlTargetFromChartPosting(posting chartPosting) FQLTarget {
	return FQLTarget{
		Tags:      posting.tags,
		Links:     posting.links,
		Payee:     posting.payee,
		Narration: posting.narration,
		Account:   posting.account,
		Flag:      posting.flag,
		Date:      posting.date,
		Metadata:  map[string]string{},
	}
}
