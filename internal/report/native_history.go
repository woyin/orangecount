// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package report

import (
	"math"
	"orangecount/internal/ledger"
	"sort"
	"time"
)

// HistoryPoint retains unavailable dates. Segment prevents clients from drawing
// a line through a missing valuation; coordinates are derived only from totals.
type HistoryPoint struct {
	Date      string            `json:"date"`
	Month     string            `json:"month"`
	Segment   int               `json:"segment"`
	Value     *float64          `json:"value"`
	Valuation *AccountValuation `json:"valuation"`
}

type HistoryCashFlow struct {
	Month         string            `json:"month"`
	Date          string            `json:"date"`
	Income        *AccountValuation `json:"income"`
	Expenses      *AccountValuation `json:"expenses"`
	Net           *AccountValuation `json:"net"`
	IncomeValue   *float64          `json:"income_value"`
	ExpensesValue *float64          `json:"expenses_value"`
}

type CurrencyHistorySeries struct {
	Currency             string            `json:"currency"`
	Mode                 string            `json:"mode"`
	Valuation            string            `json:"valuation"`
	NetWorthPoints       []HistoryPoint    `json:"net_worth_points"`
	MonthlyCashFlows     []HistoryCashFlow `json:"monthly_cash_flows"`
	IncomeTotal          *PresentedDecimal `json:"income_total"`
	ExpensesTotal        *PresentedDecimal `json:"expenses_total"`
	NetTotal             *PresentedDecimal `json:"net_total"`
	NetWorthAvailability string            `json:"net_worth_availability"`
	CashFlowAvailability string            `json:"cash_flow_availability"`
}

type historyMonth struct {
	date                  string
	net, income, expenses map[string]ledger.Decimal
	netActive             bool
}

// StatementHistory walks booked postings once, carrying Assets+Liabilities
// through every requested calendar month. Income/Expenses are inclusive-period
// activity. Local quotes are selected at each month's endpoint, never at the
// final report date for an earlier point. No snapshot rebuild or file read.
func StatementHistory(e ledger.Evaluation, from, to, target, valuation string) []CurrencyHistorySeries {
	result := []CurrencyHistorySeries{}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return result
	}
	types := e.AccountTypes()
	postings := transactionPostings(e)
	first := ""
	currencies := map[string]bool{}
	nativeValuation := valuation
	if nativeValuation == "" {
		nativeValuation = "at-cost"
	}
	if nativeValuation == "market-value" {
		nativeValuation = "units"
	}
	for _, posting := range postings {
		if posting.date > to {
			break
		}
		kind := types.TypeOf(posting.account)
		if kind != "Assets" && kind != "Liabilities" && kind != "Income" && kind != "Expenses" {
			continue
		}
		if first == "" {
			first = posting.date
		}
		_, commodity := nativeChartAmount(posting, nativeValuation)
		currencies[commodity] = true
	}
	if first == "" {
		return result
	}
	if from > first {
		first = from
	}
	start, err := time.Parse("2006-01-02", first)
	if err != nil || start.After(end) {
		return result
	}
	cursor := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	net := map[string]ledger.Decimal{}
	netActive := false
	months := []historyMonth{}
	index := 0
	for !cursor.After(end) {
		cutoff := cursor.AddDate(0, 1, 0).AddDate(0, 0, -1)
		if cutoff.After(end) {
			cutoff = end
		}
		date := cutoff.Format("2006-01-02")
		income, expenses := map[string]ledger.Decimal{}, map[string]ledger.Decimal{}
		for index < len(postings) && postings[index].date <= date {
			posting := postings[index]
			index++
			kind := types.TypeOf(posting.account)
			amount, commodity := nativeChartAmount(posting, nativeValuation)
			switch kind {
			case "Assets", "Liabilities":
				net[commodity] = net[commodity].Add(amount)
				netActive = true
			case "Income", "Expenses":
				if (from != "" && posting.date < from) || posting.date[:7] != cursor.Format("2006-01") {
					continue
				}
				if kind == "Income" {
					income[commodity] = income[commodity].Add(amount.Neg())
				} else {
					expenses[commodity] = expenses[commodity].Add(amount)
				}
			}
		}
		copyNet := map[string]ledger.Decimal{}
		for cur, amount := range net {
			copyNet[cur] = amount
		}
		months = append(months, historyMonth{date: date, net: copyNet, income: income, expenses: expenses, netActive: netActive})
		cursor = cursor.AddDate(0, 1, 0)
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
	for _, cur := range keys {
		series := CurrencyHistorySeries{Currency: cur, Mode: "native", Valuation: valuation, NetWorthPoints: []HistoryPoint{}, MonthlyCashFlows: []HistoryCashFlow{}, NetWorthAvailability: "empty", CashFlowAvailability: "empty"}
		if target != "" {
			series.Mode = "converted"
		}
		if series.Valuation == "" {
			series.Valuation = "at-cost"
		}
		totalIncome, totalExpenses := ledger.Zero(), ledger.Zero()
		flowsComplete := true
		segment := 0
		netCovered, netMissing := 0, 0
		flowCovered, flowMissing := 0, 0
		convert := func(values map[string]ledger.Decimal, active bool, date string) *AccountValuation {
			selected := map[string]string{}
			if target == "" {
				if amount, ok := values[cur]; ok {
					selected[cur] = amount.String()
				}
			} else {
				for commodity, amount := range values {
					selected[commodity] = amount.String()
				}
			}
			value := valueBalances(e, selected, active, cur, date)
			value.Mode = series.Mode
			return value
		}
		for _, month := range months {
			_, nativeNetActive := month.net[cur]
			if month.netActive && (target != "" || nativeNetActive) {
				value := convert(month.net, true, month.date)
				point := HistoryPoint{Date: month.date, Month: month.date[:7], Segment: segment, Valuation: value}
				if value.CompleteTotal != nil {
					point.Value = chartCoordinate(value.CompleteTotal)
					netCovered++
				} else {
					netMissing++
				}
				if point.Value == nil {
					segment++
				}
				series.NetWorthPoints = append(series.NetWorthPoints, point)
			}
			flowActive := len(month.income) > 0 || len(month.expenses) > 0
			if target == "" {
				_, hasIncome := month.income[cur]
				_, hasExpenses := month.expenses[cur]
				flowActive = hasIncome || hasExpenses
			}
			if !flowActive {
				continue
			}
			inc := convert(month.income, true, month.date)
			exp := convert(month.expenses, true, month.date)
			netFlows := map[string]ledger.Decimal{}
			for commodity, amount := range month.income {
				netFlows[commodity] = amount
			}
			for commodity, amount := range month.expenses {
				netFlows[commodity] = netFlows[commodity].Sub(amount)
			}
			netFlow := convert(netFlows, true, month.date)
			flow := HistoryCashFlow{Month: month.date[:7], Date: month.date, Income: inc, Expenses: exp, Net: netFlow, IncomeValue: chartCoordinate(inc.CompleteTotal), ExpensesValue: chartCoordinate(exp.CompleteTotal)}
			series.MonthlyCashFlows = append(series.MonthlyCashFlows, flow)
			if inc.CompleteTotal != nil && exp.CompleteTotal != nil {
				incomeAmount, _ := ledger.ParseDecimal(inc.CompleteTotal.Exact)
				expenseAmount, _ := ledger.ParseDecimal(exp.CompleteTotal.Exact)
				totalIncome = totalIncome.Add(incomeAmount)
				totalExpenses = totalExpenses.Add(expenseAmount)
				flowCovered++
			} else {
				flowsComplete = false
				flowMissing++
			}
		}
		series.NetWorthAvailability = historyAvailability(netCovered, netMissing)
		series.CashFlowAvailability = historyAvailability(flowCovered, flowMissing)
		if flowsComplete && flowCovered > 0 {
			incomeAmount, expenseAmount, netAmount := FormatDecimal(totalIncome), FormatDecimal(totalExpenses), FormatDecimal(totalIncome.Sub(totalExpenses))
			series.IncomeTotal, series.ExpensesTotal, series.NetTotal = &incomeAmount, &expenseAmount, &netAmount
		}
		result = append(result, series)
	}
	return result
}

func chartCoordinate(value *PresentedDecimal) *float64 {
	if value == nil {
		return nil
	}
	number, err := ledger.ParseDecimal(value.Exact)
	if err != nil {
		panic("invalid exact chart value")
	}
	coordinate, _ := number.Rat().Float64()
	if math.IsInf(coordinate, 0) || math.IsNaN(coordinate) {
		return nil
	}
	return &coordinate
}

func historyAvailability(covered, missing int) string {
	if missing > 0 {
		if covered > 0 {
			return "partial"
		}
		return "unavailable-currency"
	}
	if covered > 0 {
		return "priced"
	}
	return "empty"
}
