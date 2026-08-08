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
func AccountIntervals(e ledger.Evaluation, account, mode, interval string, filters Filters) query.Result {
	columns := []string{"interval"}
	account = strings.TrimSpace(account)
	if (mode != "changes" && mode != "balances") || account == "" {
		return query.Result{Columns: columns}
	}
	interval = normalizeChartPeriod(interval)
	textFilter, textErr := ParseFQL(strings.TrimSpace(filters.Text))
	text := strings.ToLower(strings.TrimSpace(filters.Text))
	totals := map[string]map[string]ledger.Decimal{}
	currencySet := map[string]bool{}
	for _, posting := range transactionPostings(e) {
		if !filters.MatchesDate(posting.date) {
			continue
		}
		if posting.account != account && !strings.HasPrefix(posting.account, account+":") {
			continue
		}
		// Apply the advanced text filter to each transaction posting, matching
		// Fava's filtered ledger view that interval tables aggregate over.
		if textErr == nil {
			if text != "" && !textFilter.Match(fqlTargetFromChartPosting(posting)) {
				continue
			}
		} else if text != "" && !strings.Contains(strings.ToLower(posting.narration+" "+posting.payee+" "+posting.account), text) {
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
	if len(totals) == 0 {
		return query.Result{Columns: columns}
	}
	currencies := make([]string, 0, len(currencySet))
	for currency := range currencySet {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	keys := make([]string, 0, len(totals))
	for key := range totals {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	columns = append(columns, currencies...)
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
	return query.Result{Columns: columns, Rows: rows}
}

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
