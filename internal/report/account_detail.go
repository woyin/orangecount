// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package report

import (
	stderrors "errors"
	"sort"
	"strings"
	"time"

	"orangecount/internal/ledger"
)

var ErrAccountUnknown = stderrors.New("account does not exist in this snapshot")

// AccountDetailProjection retains both scopes; scope chooses only period
// changes, history and the associated journal. Search never changes totals.
type AccountDetailProjection struct {
	History          []AccountHistorySeries      `json:"history"`
	Account          string                      `json:"account"`
	Scope            string                      `json:"scope"`
	Explicit         bool                        `json:"explicit"`
	Opened           string                      `json:"opened"`
	Closed           *string                     `json:"closed"`
	Lifecycle        string                      `json:"lifecycle"`
	Booking          string                      `json:"booking"`
	Currencies       []string                    `json:"currencies"`
	DateFrom         string                      `json:"date_from"`
	DateTo           string                      `json:"date_to"`
	Valuation        string                      `json:"valuation"`
	DirectAmounts    map[string]PresentedDecimal `json:"direct_amounts"`
	SubtreeAmounts   map[string]PresentedDecimal `json:"subtree_amounts"`
	DirectValuation  *AccountValuation           `json:"direct_valuation"`
	SubtreeValuation *AccountValuation           `json:"subtree_valuation"`
	PeriodChange     map[string]PresentedDecimal `json:"period_change"`
	PeriodValuation  *AccountValuation           `json:"period_valuation"`
}

// AccountDetailView reads booked postings, including opening inventory before
// from. Prefix-only directory accounts are valid, but similar names are not.
func AccountDetailView(e ledger.Evaluation, account, scope, from, to, target, valuation string) (*AccountDetailProjection, error) {
	state, explicit := e.Accounts[account]
	exists := explicit
	for name := range e.Accounts {
		exists = exists || strings.HasPrefix(name, account+":")
	}
	if account == "" || !exists {
		return nil, ErrAccountUnknown
	}
	if scope == "" {
		scope = "subtree"
	}
	if valuation == "" {
		valuation = "at-cost"
	}
	if target == "native" {
		target = ""
	}
	detail := &AccountDetailProjection{Account: account, Scope: scope, Explicit: explicit, Lifecycle: "implicit", Currencies: []string{}, DateFrom: from, DateTo: to, Valuation: valuation}
	if explicit {
		detail.Opened, detail.Booking, detail.Lifecycle = state.Opened.Raw, state.Booking, "open"
		detail.Currencies = append(detail.Currencies, state.Currencies...)
		if detail.Booking == "" && e.SemanticsProfile != ledger.V3 {
			detail.Booking = "FIFO"
		}
		if state.Closed != nil {
			closed := state.Closed.Raw
			detail.Closed = &closed
			if closed <= to {
				detail.Lifecycle = "closed"
			}
		}
		if detail.Opened > to {
			detail.Lifecycle = "not-opened"
		}
	}
	direct, subtree, changes := map[string]ledger.Decimal{}, map[string]ledger.Decimal{}, map[string]ledger.Decimal{}
	directActive, subtreeActive, changeActive := false, false, false
	nativeMode := valuation
	if nativeMode == "market-value" {
		nativeMode = "units"
	}
	for _, posting := range transactionPostings(e) {
		if posting.date > to {
			break
		}
		own := posting.account == account
		if !own && !strings.HasPrefix(posting.account, account+":") {
			continue
		}
		amount, cur := nativeChartAmount(posting, nativeMode)
		subtree[cur] = subtree[cur].Add(amount)
		subtreeActive = true
		if own {
			direct[cur] = direct[cur].Add(amount)
			directActive = true
		}
		if posting.date >= from && (scope != "direct" || own) {
			changes[cur] = changes[cur].Add(amount)
			changeActive = true
		}
	}
	detail.DirectAmounts, detail.SubtreeAmounts, detail.PeriodChange = presentedBalances(direct), presentedBalances(subtree), presentedBalances(changes)
	detail.DirectValuation = valueBalances(e, exactBalances(direct), directActive, target, to)
	detail.SubtreeValuation = valueBalances(e, exactBalances(subtree), subtreeActive, target, to)
	detail.PeriodValuation = valueBalances(e, exactBalances(changes), changeActive, target, to)
	detail.History = accountHistory(e, account, scope, from, to, target, nativeMode)
	return detail, nil
}

type AccountHistorySeries struct {
	Currency string         `json:"currency"`
	Mode     string         `json:"mode"`
	Points   []HistoryPoint `json:"points"`
}

func accountHistory(e ledger.Evaluation, account, scope, from, to, target, valuation string) []AccountHistorySeries {
	result := []AccountHistorySeries{}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return result
	}
	postings := []chartPosting{}
	currencies := map[string]bool{}
	for _, posting := range transactionPostings(e) {
		if posting.date > to {
			break
		}
		if posting.account != account && (scope == "direct" || !strings.HasPrefix(posting.account, account+":")) {
			continue
		}
		postings = append(postings, posting)
		_, cur := nativeChartAmount(posting, valuation)
		currencies[cur] = true
	}
	if len(postings) == 0 {
		return result
	}
	first := postings[0].date
	if from > first {
		first = from
	}
	start, err := time.Parse("2006-01-02", first)
	if err != nil || start.After(end) {
		return result
	}
	keys := []string{}
	if target != "" {
		keys = append(keys, target)
	} else {
		for cur := range currencies {
			keys = append(keys, cur)
		}
		sort.Strings(keys)
	}
	segments := make([]int, len(keys))
	for _, cur := range keys {
		mode := "native"
		if target != "" {
			mode = "converted"
		}
		result = append(result, AccountHistorySeries{Currency: cur, Mode: mode, Points: []HistoryPoint{}})
	}
	balances, index := map[string]ledger.Decimal{}, 0
	for cursor := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC); !cursor.After(end); cursor = cursor.AddDate(0, 1, 0) {
		cutoff := cursor.AddDate(0, 1, 0).AddDate(0, 0, -1)
		if cutoff.After(end) {
			cutoff = end
		}
		date := cutoff.Format("2006-01-02")
		for index < len(postings) && postings[index].date <= date {
			amount, cur := nativeChartAmount(postings[index], valuation)
			balances[cur] = balances[cur].Add(amount)
			index++
		}
		for i, cur := range keys {
			selected := exactBalances(balances)
			if target == "" {
				amount, active := selected[cur]
				if !active {
					continue
				}
				selected = map[string]string{cur: amount}
			}
			value := valueBalances(e, selected, index > 0, cur, date)
			value.Mode = result[i].Mode
			point := HistoryPoint{Date: date, Month: date[:7], Segment: segments[i], Valuation: value, Value: chartCoordinate(value.CompleteTotal)}
			result[i].Points = append(result[i].Points, point)
			if point.Value == nil {
				segments[i]++
			}
		}
	}
	return result
}

func exactBalances(values map[string]ledger.Decimal) map[string]string {
	result := make(map[string]string, len(values))
	for cur, amount := range values {
		result[cur] = amount.String()
	}
	return result
}

func presentedBalances(values map[string]ledger.Decimal) map[string]PresentedDecimal {
	result := make(map[string]PresentedDecimal, len(values))
	for cur, amount := range values {
		result[cur] = FormatDecimal(amount)
	}
	return result
}
