// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

import (
	"sort"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/report"
)

// TreeReport is the private wire projection consumed by the transplanted Fava
// tree-report components. It keeps the Fava tree vocabulary while preserving
// OrangeCount's exact decimal presentation objects instead of converting
// ledger values to binary floating point.
type TreeReport struct {
	DateRange *DateRange                  `json:"date_range"`
	Charts    []report.PresentedChartSpec `json:"charts"`
	Trees     []TreeNode                  `json:"trees"`
}

type DateRange struct {
	Begin string `json:"begin"`
	End   string `json:"end"`
}

// TreeNode is an adapted Fava SerialisedTreeNode. Balance is the node's own
// amount and BalanceChildren is the aggregate amount when the node is
// collapsed. That distinction mirrors report.accountRootReport's explicit
// own/total projections.
type TreeNode struct {
	Account         string                             `json:"account"`
	Balance         map[string]report.PresentedDecimal `json:"balance"`
	BalanceChildren map[string]report.PresentedDecimal `json:"balance_children"`
	Cost            map[string]report.PresentedDecimal `json:"cost"`
	CostChildren    map[string]report.PresentedDecimal `json:"cost_children"`
	Children        []TreeNode                         `json:"children"`
	HasTxns         bool                               `json:"has_txns"`
}

type treeNode struct {
	account         string
	balance         map[string]ledger.Decimal
	balanceChildren map[string]ledger.Decimal
	hasTxns         bool
	children        map[string]*treeNode
}

// ProjectTreeReport adapts the semantic report result to the narrow wire
// contract used by the transplanted frontend. It never evaluates entries or
// mutates the supplied evaluation.
func ProjectTreeReport(e ledger.Evaluation, name string, filters report.Filters, period, currency, valuation string) TreeReport {
	result, roots := treeResult(e, name)
	result = report.Filter(result, filters)
	root := buildTree(result)
	revalueTree(root, e, valuation)
	trees := make([]TreeNode, 0, len(roots))
	for _, name := range roots {
		// The trial balance is rooted at the synthetic "" account, which is the
		// tree root itself rather than one of its children. Looking it up as a
		// child found nothing and substituted an empty placeholder, leaving the
		// whole trial-balance table blank.
		node := root
		if name != "" {
			node = root.children[name]
		}
		if node == nil {
			node = &treeNode{account: name, balance: map[string]ledger.Decimal{}, balanceChildren: map[string]ledger.Decimal{}, children: map[string]*treeNode{}}
		}
		trees = append(trees, toTreeNode(node))
	}
	if name == "income_statement" {
		trees = insertNetProfit(trees)
	}
	charts := []report.PresentedChartSpec{}
	// Prefer the per-measure, per-currency chart set: it plots every currency
	// the ledger actually holds instead of discarding whatever cannot be
	// converted into one display currency.
	for _, chart := range report.ReportCharts(e, chartRoute(name), period, valuation) {
		charts = append(charts, report.PresentChart(chart))
	}
	if len(charts) == 0 {
		chartCurrency := strings.TrimSpace(currency)
		if chartCurrency == "" {
			chartCurrency = firstCurrency(e)
		}
		chart := report.ReportChart(e, chartRoute(name), period, chartCurrency, valuation, filters.Account)
		if chart.Kind != "" {
			charts = append(charts, report.PresentChart(chart))
		}
	}
	return TreeReport{DateRange: evaluationDateRange(e), Charts: charts, Trees: trees}
}

func treeResult(e ledger.Evaluation, name string) (query.Result, []string) {
	switch name {
	case "income_statement":
		return report.IncomeStatement(e), []string{"Income", "Net Profit", "Expenses"}
	case "balance_sheet":
		return report.BalanceSheet(e), []string{"Assets", "Liabilities", "Equity"}
	case "trial_balance":
		return report.TrialBalanceTree(e), []string{""}
	default:
		return query.Result{}, nil
	}
}

func chartRoute(name string) string {
	switch name {
	case "income_statement":
		return "income-statement"
	case "balance_sheet":
		return "balance-sheet"
	case "trial_balance":
		return "trial-balance"
	default:
		return ""
	}
}

// revalueTree restates every node's own balance under the requested valuation
// and then rebuilds the subtree totals from those restated amounts.
//
// The semantic report layer reports each account in the commodity it holds, so
// without this pass a ledger holding funds or shares shows raw lot units
// ("6433.906476 FUND_017436") where Fava shows their cost in the operating
// currency. Subtree totals must be recomputed rather than converted, because a
// parent's total is the sum of its descendants' own balances and conversion
// changes which currency each of those lands in.
func revalueTree(node *treeNode, e ledger.Evaluation, valuation string) map[string]ledger.Decimal {
	if node == nil {
		return map[string]ledger.Decimal{}
	}
	if node.account != "" {
		node.balance = report.ValuedBalances(e, node.account, valuation)
	}
	totals := make(map[string]ledger.Decimal, len(node.balance))
	for currency, amount := range node.balance {
		totals[currency] = amount
	}
	for _, child := range node.children {
		for currency, amount := range revalueTree(child, e, valuation) {
			totals[currency] = totals[currency].Add(amount)
		}
	}
	for currency, amount := range totals {
		if amount.IsZero() {
			delete(totals, currency)
		}
	}
	node.balanceChildren = totals
	result := make(map[string]ledger.Decimal, len(totals))
	for currency, amount := range totals {
		result[currency] = amount
	}
	return result
}

func buildTree(result query.Result) *treeNode {
	root := &treeNode{account: "", balance: map[string]ledger.Decimal{}, balanceChildren: map[string]ledger.Decimal{}, children: map[string]*treeNode{}}
	for _, row := range result.Rows {
		account, _ := row["account"].(string)
		node := ensureTreeNode(root, account)
		currency, _ := row["currency"].(string)
		if currency != "" {
			node.balance[currency] = decimalValue(row["own_balance"], row["balance"])
			node.balanceChildren[currency] = decimalValue(row["total_balance"], row["balance"])
		}
		if direct, ok := row["_tree_has_direct"].(bool); ok {
			node.hasTxns = node.hasTxns || direct
		}
	}
	return root
}

func ensureTreeNode(root *treeNode, account string) *treeNode {
	if account == "" {
		return root
	}
	parts := strings.Split(account, ":")
	parent := root
	name := ""
	for _, part := range parts {
		if name == "" {
			name = part
		} else {
			name += ":" + part
		}
		node := parent.children[name]
		if node == nil {
			node = &treeNode{account: name, balance: map[string]ledger.Decimal{}, balanceChildren: map[string]ledger.Decimal{}, children: map[string]*treeNode{}}
			parent.children[name] = node
		}
		parent = node
	}
	return parent
}

func decimalValue(values ...any) ledger.Decimal {
	for _, value := range values {
		if decimal, ok := value.(ledger.Decimal); ok {
			return decimal
		}
	}
	return ledger.Zero()
}

func toTreeNode(node *treeNode) TreeNode {
	children := make([]*treeNode, 0, len(node.children))
	for _, child := range node.children {
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool { return children[i].account < children[j].account })
	result := TreeNode{
		Account:         node.account,
		Balance:         presentedMap(node.balance),
		BalanceChildren: presentedMap(node.balanceChildren),
		Cost:            nil,
		CostChildren:    nil,
		Children:        make([]TreeNode, 0, len(children)),
		HasTxns:         node.hasTxns,
	}
	for _, child := range children {
		result.Children = append(result.Children, toTreeNode(child))
	}
	return result
}

func presentedMap(values map[string]ledger.Decimal) map[string]report.PresentedDecimal {
	result := make(map[string]report.PresentedDecimal, len(values))
	for currency, value := range values {
		result[currency] = report.FormatDecimal(value)
	}
	return result
}

func insertNetProfit(trees []TreeNode) []TreeNode {
	if len(trees) < 3 {
		return trees
	}
	netBalance := sumPresented(trees[0].BalanceChildren, trees[2].BalanceChildren)
	net := TreeNode{
		Account:         "Net Profit",
		Balance:         netBalance,
		BalanceChildren: netBalance,
		Children:        []TreeNode{},
		HasTxns:         trees[0].HasTxns || trees[2].HasTxns,
	}
	return []TreeNode{trees[0], net, trees[2]}
}

func sumPresented(left, right map[string]report.PresentedDecimal) map[string]report.PresentedDecimal {
	result := make(map[string]report.PresentedDecimal, len(left)+len(right))
	for currency, value := range left {
		result[currency] = value
	}
	for currency, value := range right {
		if existing, ok := result[currency]; ok {
			// The exact form is canonical for terminating values. For the
			// display-only net row, retain exact rational arithmetic by parsing
			// through the ledger decimal constructor rather than floats.
			result[currency] = addPresented(existing, value)
		} else {
			result[currency] = value
		}
	}
	return result
}

func addPresented(left, right report.PresentedDecimal) report.PresentedDecimal {
	leftDecimal, leftOK := ledger.ParseDecimal(left.Exact)
	rightDecimal, rightOK := ledger.ParseDecimal(right.Exact)
	if leftOK == nil && rightOK == nil {
		return report.FormatDecimal(leftDecimal.Add(rightDecimal))
	}
	return report.PresentedDecimal{Display: left.Display + " + " + right.Display, Exact: left.Exact + " + " + right.Exact, Approximate: true}
}

func evaluationDateRange(e ledger.Evaluation) *DateRange {
	begin, end := "", ""
	for _, entry := range e.Entries {
		date := entry.Date.Raw
		if date == "" {
			continue
		}
		if begin == "" || date < begin {
			begin = date
		}
		if date > end {
			end = date
		}
	}
	if begin == "" && end == "" {
		return nil
	}
	return &DateRange{Begin: begin, End: end}
}

func firstCurrency(e ledger.Evaluation) string {
	if configured := strings.TrimSpace(e.Options["operating_currency"]); configured != "" {
		for _, value := range strings.FieldsFunc(configured, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' }) {
			if value != "" {
				return value
			}
		}
	}
	currencies := map[string]bool{}
	for _, state := range e.Accounts {
		for currency := range state.Balances {
			if currency != "" {
				currencies[currency] = true
			}
		}
	}
	values := make([]string, 0, len(currencies))
	for currency := range currencies {
		values = append(values, currency)
	}
	sort.Strings(values)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
