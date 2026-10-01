// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package desktop

import (
	"fmt"
	"math"
	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"sort"
	"strings"
)

type SummaryPayload struct {
	Currencies        []string        `json:"currencies"`
	Valid             bool            `json:"valid"`
	Title             string          `json:"title"`
	OperatingCur      []string        `json:"operating_currencies"`
	Accounts          []string        `json:"accounts"`
	EntriesCount      int             `json:"entries_count"`
	TransactionsCount int             `json:"transactions_count"`
	EntryPath         string          `json:"entry_path"`
	Diagnostics       []DiagnosticDTO `json:"diagnostics"`
	ErrorCount        int             `json:"error_count"`
	Errors            []string        `json:"errors"`
}

type JournalPostingDTO struct {
	Account       string                   `json:"account"`
	UnitsNumber   string                   `json:"units_number"`
	UnitsCurrency string                   `json:"units_currency"`
	Cost          *PostingCost             `json:"cost,omitempty"`
	Price         *PostingPrice            `json:"price,omitempty"`
	Source        SourceLocation           `json:"source"`
	Amount        *report.PresentedDecimal `json:"amount,omitempty"`
}

type JournalTransactionDTO struct {
	ID        string              `json:"id"`
	Date      string              `json:"date"`
	Flag      string              `json:"flag"`
	Payee     string              `json:"payee"`
	Narration string              `json:"narration"`
	Tags      []string            `json:"tags"`
	Links     []string            `json:"links"`
	Postings  []JournalPostingDTO `json:"postings"`
	Source    SourceLocation      `json:"source"`
}

type JournalPayload struct {
	TotalCount   int                     `json:"total_count"`
	Offset       int                     `json:"offset"`
	Limit        int                     `json:"limit"`
	HasMore      bool                    `json:"has_more"`
	DateFrom     string                  `json:"date_from"`
	DateTo       string                  `json:"date_to"`
	Transactions []JournalTransactionDTO `json:"transactions"`
	Error        string                  `json:"error,omitempty"`
}

type TreeReportPayload struct {
	ValuationStatus     report.ValuationStatus   `json:"valuation_status"`
	Currency            string                   `json:"currency"`
	DateFrom            string                   `json:"date_from"`
	DateTo              string                   `json:"date_to"`
	Availability        string                   `json:"availability"`
	ReportType          string                   `json:"report_type"`
	OperatingCurrencies []string                 `json:"operating_currencies"`
	RootNodes           []report.AccountTreeNode `json:"root_nodes"`
	Error               string                   `json:"error,omitempty"`
}

type TrendPointDTO struct {
	Date     string                  `json:"date"`
	Value    float64                 `json:"value"`
	Currency string                  `json:"currency"`
	Amount   report.PresentedDecimal `json:"amount"`
}

type CashFlowBarDTO struct {
	Month          string                  `json:"month"`
	Income         float64                 `json:"income"`
	Expenses       float64                 `json:"expenses"`
	Net            float64                 `json:"net"`
	Currency       string                  `json:"currency"`
	IncomeAmount   report.PresentedDecimal `json:"income_amount"`
	ExpensesAmount report.PresentedDecimal `json:"expenses_amount"`
	NetAmount      report.PresentedDecimal `json:"net_amount"`
}

type HistoricalTrendsPayload struct {
	CurrencySeries       []report.CurrencyHistorySeries `json:"currency_series"`
	IncomeAmount         report.PresentedDecimal        `json:"income_amount"`
	ExpensesAmount       report.PresentedDecimal        `json:"expenses_amount"`
	NetAmount            report.PresentedDecimal        `json:"net_amount"`
	DateFrom             string                         `json:"date_from"`
	DateTo               string                         `json:"date_to"`
	OperatingCurrency    string                         `json:"operating_currency"`
	NetWorthAvailability string                         `json:"net_worth_availability"`
	CashFlowAvailability string                         `json:"cash_flow_availability"`
	NetWorthPoints       []TrendPointDTO                `json:"net_worth_points"`
	MonthlyCashFlows     []CashFlowBarDTO               `json:"monthly_cash_flows"`
	Error                string                         `json:"error,omitempty"`
}

func ProjectSummary(path string, evaluation *ledger.Evaluation, diagnostics []DiagnosticDTO, valid bool) SummaryPayload {
	result := SummaryPayload{Valid: valid, EntryPath: path, Currencies: []string{}, Accounts: []string{}, OperatingCur: []string{}, Errors: []string{}, Diagnostics: diagnostics}
	for _, d := range diagnostics {
		if d.Severity == diagnostic.Error {
			result.ErrorCount++
			result.Errors = append(result.Errors, d.Message)
		}
	}
	if evaluation == nil {
		return result
	}
	result.Title = evaluation.Options["title"]
	result.OperatingCur = append(result.OperatingCur, strings.Fields(evaluation.Options["operating_currency"])...)
	result.EntriesCount = len(evaluation.Entries)
	for _, entry := range evaluation.Entries {
		if entry.Directive.Kind() == ledger.KindTxn {
			result.TransactionsCount++
		}
	}
	for name := range evaluation.Accounts {
		result.Accounts = append(result.Accounts, name)
	}
	sort.Strings(result.Accounts)
	currencies := map[string]bool{}
	for _, cur := range result.OperatingCur {
		currencies[cur] = true
	}
	for _, state := range evaluation.Accounts {
		for _, cur := range state.Currencies {
			currencies[cur] = true
		}
		for cur := range state.Balances {
			currencies[cur] = true
		}
	}
	result.Currencies = []string{}
	for cur := range currencies {
		result.Currencies = append(result.Currencies, cur)
	}
	sort.Strings(result.Currencies)
	return result
}

func ProjectJournal(eval ledger.Evaluation) JournalPayload {
	txs := make([]JournalTransactionDTO, 0)

	for i, record := range eval.Entries {
		var tx *ledger.Transaction
		switch t := record.Directive.(type) {
		case *ledger.Transaction:
			tx = t
		case ledger.Transaction:
			tx = &t
		}
		if tx == nil {
			continue
		}

		postings := make([]JournalPostingDTO, 0, len(tx.Postings))
		for _, p := range tx.Postings {
			num := ""
			cur := ""
			var amount *report.PresentedDecimal
			if p.Units != nil {
				num = p.Units.Number.Raw
				cur = p.Units.Currency
				value := report.FormatDecimal(ledger.DecimalFromNumber(p.Units.Number))
				amount = &value
			}
			postings = append(postings, JournalPostingDTO{
				Account:       p.Account,
				UnitsNumber:   num,
				UnitsCurrency: cur,
				Amount:        amount,
				Cost:          postingCost(p.Cost),
				Price:         postingPrice(p.Price),
				Source:        location(record.File, p.Span()),
			})
		}

		tags := tx.Tags
		if tags == nil {
			tags = []string{}
		}
		links := tx.Links
		if links == nil {
			links = []string{}
		}

		txs = append(txs, JournalTransactionDTO{
			ID:        fmt.Sprintf("tx_%04d_%s", i+1, record.Date.Raw),
			Date:      record.Date.Raw,
			Flag:      tx.Flag,
			Payee:     tx.Payee,
			Narration: tx.Narration,
			Tags:      tags,
			Links:     links,
			Postings:  postings,
			Source:    location(record.File, record.Span),
		})
	}

	return JournalPayload{
		TotalCount:   len(txs),
		Transactions: txs,
	}
}

func ProjectTree(eval ledger.Evaluation, reportType string) TreeReportPayload {
	opCurs := make([]string, 0)
	if opCur, ok := eval.Options["operating_currency"]; ok {
		opCurs = append(opCurs, strings.Fields(opCur)...)
	}

	var roots []string
	if reportType == "income_statement" {
		roots = []string{"Income", "Expenses"}
	} else {
		reportType = "balance_sheet"
		roots = []string{"Assets", "Liabilities", "Equity"}
	}

	nodes := report.BuildAccountTree(eval, roots...)
	if nodes == nil {
		nodes = []report.AccountTreeNode{}
	}
	return TreeReportPayload{
		ReportType:          reportType,
		OperatingCurrencies: opCurs,
		RootNodes:           nodes,
	}
}

func ProjectTrends(eval ledger.Evaluation, cur string) HistoricalTrendsPayload {
	if currencies := strings.Fields(eval.Options["operating_currency"]); cur == "" && len(currencies) > 0 {
		cur = currencies[0]
	}
	if cur == "" {
		return emptyTrends("select-currency")
	}

	bsChart := report.ReportChart(eval, "balance-sheet", "month", cur, "at-cost", "")
	isChart := report.ReportChart(eval, "income-statement", "month", cur, "at-cost", "")

	return projectChartTrends(bsChart, isChart, cur)
}

func projectChartTrends(bsChart, isChart report.ChartSpec, cur string) HistoricalTrendsPayload {
	netWorthPoints := make([]TrendPointDTO, 0)
	for _, series := range bsChart.Series {
		if series.Label == "Net worth" || series.Label == cur || len(bsChart.Series) == 1 {
			for _, pt := range series.Points {
				val, _ := pt.Value.Rat().Float64()
				netWorthPoints = append(netWorthPoints, TrendPointDTO{
					Date:     pt.Date,
					Value:    val,
					Currency: cur,
					Amount:   report.FormatDecimal(pt.Value),
				})
			}
			break
		}
	}

	monthIncome := map[string]ledger.Decimal{}
	monthExpense := map[string]ledger.Decimal{}
	seenMonths := map[string]bool{}
	var months []string

	for _, series := range isChart.Series {
		if series.Label != "Income" && series.Label != "Expenses" {
			continue
		}
		for _, pt := range series.Points {
			if !seenMonths[pt.Date] {
				months = append(months, pt.Date)
				seenMonths[pt.Date] = true
			}
			if series.Label == "Income" {
				monthIncome[pt.Date] = monthIncome[pt.Date].Add(pt.Value.Neg())
			} else {
				monthExpense[pt.Date] = monthExpense[pt.Date].Add(pt.Value)
			}
		}
	}
	sort.Strings(months)

	cashFlows := make([]CashFlowBarDTO, 0)
	totalIncome, totalExpenses := ledger.Zero(), ledger.Zero()
	for _, m := range months {
		inc := monthIncome[m]
		exp := monthExpense[m]
		totalIncome = totalIncome.Add(inc)
		totalExpenses = totalExpenses.Add(exp)
		incomeValue, _ := inc.Rat().Float64()
		expenseValue, _ := exp.Rat().Float64()
		netValue, _ := inc.Sub(exp).Rat().Float64()
		cashFlows = append(cashFlows, CashFlowBarDTO{
			Month:          m,
			Income:         incomeValue,
			Expenses:       expenseValue,
			Net:            netValue,
			Currency:       cur,
			IncomeAmount:   report.FormatDecimal(inc),
			ExpensesAmount: report.FormatDecimal(exp),
			NetAmount:      report.FormatDecimal(inc.Sub(exp)),
		})
	}

	if bsChart.Availability != "priced" {
		netWorthPoints = []TrendPointDTO{}
	}
	if isChart.Availability != "priced" {
		cashFlows = []CashFlowBarDTO{}
	}
	return HistoricalTrendsPayload{
		OperatingCurrency: cur,
		IncomeAmount:      report.FormatDecimal(totalIncome), ExpensesAmount: report.FormatDecimal(totalExpenses), NetAmount: report.FormatDecimal(totalIncome.Sub(totalExpenses)),
		NetWorthAvailability: bsChart.Availability,
		CashFlowAvailability: isChart.Availability,
		NetWorthPoints:       netWorthPoints,
		MonthlyCashFlows:     cashFlows,
	}
}

func emptyTrends(availability string) HistoricalTrendsPayload {
	zero := report.FormatDecimal(ledger.Zero())
	return HistoricalTrendsPayload{IncomeAmount: zero, ExpensesAmount: zero, NetAmount: zero, NetWorthAvailability: availability, CashFlowAvailability: availability, NetWorthPoints: []TrendPointDTO{}, MonthlyCashFlows: []CashFlowBarDTO{}}
}

// projectHistory preserves structured unavailable points in CurrencySeries.
// The legacy flat fields are only populated for one complete currency series.
func projectHistory(e ledger.Evaluation, from, to, currency, valuation string) HistoricalTrendsPayload {
	series := report.StatementHistory(e, from, to, currency, valuation)
	result := emptyTrends("empty")
	result.DateFrom, result.DateTo = from, to
	result.CurrencySeries = series
	if len(series) > 1 {
		result.NetWorthAvailability, result.CashFlowAvailability = "native-multi", "native-multi"
		return result
	}
	if len(series) == 0 {
		return result
	}
	item := series[0]
	result.OperatingCurrency = item.Currency
	result.NetWorthAvailability, result.CashFlowAvailability = item.NetWorthAvailability, item.CashFlowAvailability
	if item.IncomeTotal != nil {
		result.IncomeAmount = *item.IncomeTotal
		result.ExpensesAmount = *item.ExpensesTotal
		result.NetAmount = *item.NetTotal
	}
	if item.NetWorthAvailability == "priced" {
		for _, point := range item.NetWorthPoints {
			if point.Value == nil {
				// The old flat contract cannot represent a missing coordinate.
				// Keep the complete nullable series rather than join across it.
				result.NetWorthPoints = []TrendPointDTO{}
				break
			}
			result.NetWorthPoints = append(result.NetWorthPoints, TrendPointDTO{Date: point.Month, Value: *point.Value, Currency: item.Currency, Amount: *point.Valuation.CompleteTotal})
		}
	}
	if item.CashFlowAvailability == "priced" {
		for _, flow := range item.MonthlyCashFlows {
			net, _ := ledger.ParseDecimal(flow.Net.CompleteTotal.Exact)
			netCoordinate, _ := net.Rat().Float64()
			if flow.IncomeValue == nil || flow.ExpensesValue == nil || math.IsInf(netCoordinate, 0) || math.IsNaN(netCoordinate) {
				result.MonthlyCashFlows = []CashFlowBarDTO{}
				break
			}
			result.MonthlyCashFlows = append(result.MonthlyCashFlows, CashFlowBarDTO{Month: flow.Month, Income: *flow.IncomeValue, Expenses: *flow.ExpensesValue, Net: netCoordinate, Currency: item.Currency, IncomeAmount: *flow.Income.CompleteTotal, ExpensesAmount: *flow.Expenses.CompleteTotal, NetAmount: *flow.Net.CompleteTotal})
		}
	}
	return result
}
