// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package report

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

type ProjectionSource struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"end_line"`
	EndColumn int    `json:"end_column"`
}

type HoldingLot struct {
	ID           string            `json:"id"`
	Account      string            `json:"account"`
	Commodity    string            `json:"commodity"`
	Quantity     PresentedDecimal  `json:"quantity"`
	CostCurrency *string           `json:"cost_currency"`
	CostNumber   *PresentedDecimal `json:"cost_number"`
	CostBasis    *PresentedDecimal `json:"cost_basis"`
	CostDate     *string           `json:"cost_date"`
	CostLabel    *string           `json:"cost_label"`
	Source       ProjectionSource  `json:"source"`
	Origin       string            `json:"origin"`
}

type HoldingGroup struct {
	ID              string            `json:"id"`
	Account         string            `json:"account"`
	Accounts        []string          `json:"accounts"`
	Commodity       string            `json:"commodity"`
	CostCurrency    *string           `json:"cost_currency"`
	Quantity        PresentedDecimal  `json:"quantity"`
	CostBasis       *PresentedDecimal `json:"cost_basis"`
	AverageCost     *PresentedDecimal `json:"average_cost"`
	Lots            []HoldingLot      `json:"lots"`
	InventoryKind   string            `json:"inventory_kind"`
	CostValuation   *AccountValuation `json:"cost_valuation"`
	MarketValuation *AccountValuation `json:"market_valuation"`
	UnrealizedGain  *PresentedDecimal `json:"unrealized_gain"`
	GainReason      string            `json:"gain_reason"`
	LotsPage        ProjectionPage    `json:"lots_page"`
}

type ProjectionPage struct {
	TotalCount    int  `json:"total_count"`
	Offset        int  `json:"offset"`
	Limit         int  `json:"limit"`
	ReturnedCount int  `json:"returned_count"`
	HasMore       bool `json:"has_more"`
	Truncated     bool `json:"truncated"`
}

type HoldingsProjection struct {
	Events         []ReductionEvent         `json:"events"`
	AverageHistory []AverageCostSeries      `json:"average_history"`
	DateFrom       string                   `json:"date_from"`
	DateTo         string                   `json:"date_to"`
	Account        string                   `json:"account"`
	Scope          string                   `json:"scope"`
	Groups         []HoldingGroup           `json:"groups"`
	Currency       string                   `json:"currency"`
	Valuation      string                   `json:"valuation"`
	GroupBy        string                   `json:"group_by"`
	Section        string                   `json:"section"`
	GroupsPage     ProjectionPage           `json:"groups_page"`
	EventsPage     ProjectionPage           `json:"events_page"`
	HistoryPage    ProjectionPage           `json:"history_page"`
	SourceFiles    map[source.FileID]string `json:"source_files"`
}

type HoldingsOptions struct{ Currency, Valuation, GroupBy string }

func HoldingsView(e ledger.Evaluation, account, scope, from, to string) (*HoldingsProjection, error) {
	return HoldingsViewWithOptions(e, account, scope, from, to, HoldingsOptions{})
}

func HoldingsViewWithOptions(e ledger.Evaluation, account, scope, from, to string, options HoldingsOptions) (*HoldingsProjection, error) {
	if account != "" {
		exists := false
		for name := range e.Accounts {
			exists = exists || name == account || strings.HasPrefix(name, account+":")
		}
		if !exists {
			return nil, ErrAccountUnknown
		}
	}
	if scope == "" {
		scope = "subtree"
	}
	result := &HoldingsProjection{DateFrom: from, DateTo: to, Account: account, Scope: scope, Groups: []HoldingGroup{}}
	result.Currency = options.Currency
	if result.Currency == "native" {
		result.Currency = ""
	}
	result.Valuation = options.Valuation
	if result.Valuation == "" {
		result.Valuation = "at-cost"
	}
	result.GroupBy = options.GroupBy
	if result.GroupBy == "" {
		result.GroupBy = "account"
	}
	if account == "" {
		result.Scope = "assets-and-liabilities"
	}
	replayed := ledger.ReplayBookedInventory(e, to)
	paths := map[source.FileID]string{}
	origins := map[source.Span]bool{}
	for _, entry := range e.Entries {
		paths[entry.Span.File] = entry.File
		if tx := evaluatedTransaction(entry); tx != nil {
			for _, posting := range tx.Postings {
				if posting.Flag == "M" {
					origins[posting.Span()] = true
				}
			}
		}
	}
	result.SourceFiles = paths
	result.Events = reductionEvents(e, replayed.Changes, account, scope, from, paths, origins)
	result.AverageHistory = averageInventoryHistory(e, replayed.Changes, account, scope, from, to, result.GroupBy)
	groups := map[string]*HoldingGroup{}
	quantities, bases := map[string]ledger.Decimal{}, map[string]ledger.Decimal{}
	types := e.AccountTypes()
	for name, positions := range replayed.Accounts {
		if !holdingAccountMatches(types, name, account, scope) {
			continue
		}
		for _, position := range positions {
			lot := holdingLot(name, position, paths, origins)
			key := holdingGroupIdentity(name, position.Currency, lot.CostCurrency, result.GroupBy)
			group := groups[key]
			if group == nil {
				group = &HoldingGroup{ID: key, Account: name, Accounts: []string{}, Commodity: position.Currency, CostCurrency: lot.CostCurrency, Lots: []HoldingLot{}}
				if result.GroupBy != "account" {
					group.Account = ""
				}
				groups[key] = group
			}
			group.Lots = append(group.Lots, lot)
			group.Accounts = appendUniqueString(group.Accounts, name)
			quantities[key] = quantities[key].Add(position.Units)
			if position.Cost != nil {
				bases[key] = bases[key].Add(position.Units.Mul(position.Cost.Number))
			}
		}
	}
	for _, key := range sortedKeys(groups) {
		group := groups[key]
		group.Quantity = FormatDecimal(quantities[key])
		if group.CostCurrency != nil {
			basis := FormatDecimal(bases[key])
			group.CostBasis = &basis
			if !quantities[key].IsZero() {
				average := FormatDecimal(bases[key].Quo(quantities[key]))
				group.AverageCost = &average
			}
		}
		sort.Slice(group.Lots, func(i, j int) bool { return group.Lots[i].ID < group.Lots[j].ID })
		sort.Strings(group.Accounts)
		valueHoldingGroup(e, group, result.Currency, to)
		result.Groups = append(result.Groups, *group)
	}
	sort.Slice(result.Groups, func(i, j int) bool {
		x, y := result.Groups[i], result.Groups[j]
		if result.GroupBy == "cost_currency" {
			left, right := "", ""
			if x.CostCurrency != nil {
				left = *x.CostCurrency
			}
			if y.CostCurrency != nil {
				right = *y.CostCurrency
			}
			if left != right {
				return left < right
			}
		}
		if x.Account != y.Account {
			return x.Account < y.Account
		}
		if x.Commodity != y.Commodity {
			return x.Commodity < y.Commodity
		}
		return x.ID < y.ID
	})
	return result, nil
}

func holdingGroupIdentity(account, commodity string, costCurrency *string, groupBy string) string {
	if groupBy != "account" {
		account = ""
	}
	return holdingIdentity(groupBy, account, commodity, costCurrency)
}

func appendUniqueString(values []string, value string) []string {
	for _, old := range values {
		if old == value {
			return values
		}
	}
	return append(values, value)
}

func valueHoldingGroup(e ledger.Evaluation, group *HoldingGroup, target, cutoff string) {
	positive, negative := false, false
	for _, lot := range group.Lots {
		amount, _ := ledger.ParseDecimal(lot.Quantity.Exact)
		positive = positive || amount.Sign() > 0
		negative = negative || amount.Sign() < 0
	}
	group.InventoryKind = "long"
	if group.CostCurrency == nil {
		group.InventoryKind = "plain"
	} else if positive && negative {
		group.InventoryKind = "mixed"
	} else if negative {
		group.InventoryKind = "short"
	}
	quote := target
	if quote == "" && group.CostCurrency != nil {
		quote = *group.CostCurrency
	}
	group.MarketValuation = valueBalances(e, map[string]string{group.Commodity: group.Quantity.Exact}, true, quote, cutoff)
	if quote == "" {
		group.MarketValuation = unknownHoldingValue(group.Commodity, "unspecified-quote-currency", quote, cutoff)
	}
	if group.CostCurrency == nil {
		group.CostValuation = unknownHoldingValue(group.Commodity, "missing-cost", quote, cutoff)
		group.GainReason = "missing-cost"
		return
	}
	group.CostValuation = valueBalances(e, map[string]string{*group.CostCurrency: group.CostBasis.Exact}, true, quote, cutoff)
	switch {
	case group.MarketValuation.CompleteTotal == nil:
		group.GainReason = "missing-price"
	case group.CostValuation.CompleteTotal == nil:
		group.GainReason = "missing-cost-quote"
	default:
		market, _ := ledger.ParseDecimal(group.MarketValuation.CompleteTotal.Exact)
		cost, _ := ledger.ParseDecimal(group.CostValuation.CompleteTotal.Exact)
		gain := FormatDecimal(market.Sub(cost))
		group.UnrealizedGain = &gain
	}
}

func unknownHoldingValue(commodity, reason, currency, cutoff string) *AccountValuation {
	mode := "native"
	if currency != "" {
		mode = "converted"
	}
	return &AccountValuation{ValuationStatus: ValuationStatus{Mode: mode, TargetCurrency: currency, AsOf: cutoff, Completeness: "unavailable", MissingItemCount: 1, Missing: []MissingValuation{{Commodity: commodity, Reason: reason}}, Quotes: []QuoteUse{}}}
}

func holdingAccountMatches(types ledger.AccountTypes, name, account, scope string) bool {
	if account == "" {
		kind := types.TypeOf(name)
		return kind == "Assets" || kind == "Liabilities"
	}
	return name == account || (scope != "direct" && strings.HasPrefix(name, account+":"))
}

func holdingLot(account string, position ledger.Position, paths map[source.FileID]string, origins map[source.Span]bool) HoldingLot {
	lot := HoldingLot{Account: account, Commodity: position.Currency, Quantity: FormatDecimal(position.Units), Source: projectionLocation(paths[position.Span.File], position.Span), Origin: "posting"}
	if origins[position.Span] {
		lot.Origin = "average-merge"
	}
	if cost := position.Cost; cost != nil {
		currency, number, basis := cost.Currency, FormatDecimal(cost.Number), FormatDecimal(position.Units.Mul(cost.Number))
		lot.CostCurrency, lot.CostNumber, lot.CostBasis = &currency, &number, &basis
		if cost.Date != nil {
			date := cost.Date.Raw
			lot.CostDate = &date
		}
		if cost.LabelPresent {
			label := cost.Label
			lot.CostLabel = &label
		}
	}
	var exactCost *string
	if lot.CostNumber != nil {
		exact := lot.CostNumber.Exact
		exactCost = &exact
	}
	lot.ID = holdingIdentity(account, position.Currency, lot.CostCurrency, exactCost, lot.CostDate, lot.CostLabel)
	return lot
}

func projectionLocation(path string, span source.Span) ProjectionSource {
	return ProjectionSource{Path: path, Line: span.StartLine, Column: span.StartColumn, EndLine: span.EndLine, EndColumn: span.EndColumn}
}

func holdingIdentity(parts ...any) string {
	data, err := json.Marshal(parts)
	if err != nil {
		panic("invalid immutable holding identity")
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
