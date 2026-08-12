// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

// Package planning evaluates owner-confirmed liquidity plans. It deliberately
// has no HTTP or source-file dependency: callers supply already-resolved
// balances and plans, while this package returns a deterministic explanation
// of the conservative safe-to-spend result.
package planning

import (
	"fmt"
	"sort"
	"strings"

	"orangecount/internal/ledger"
)

// Commitment expresses whether an outflow is protected in the current plan or
// may be released by an explicit alternative scenario. Both kinds reduce the
// primary safe-to-spend amount.
type Commitment string

const (
	Committed  Commitment = "committed"
	Adjustable Commitment = "adjustable"
)

// Direction identifies the economic direction of a planned cash flow.
type Direction string

const (
	Inflow  Direction = "inflow"
	Outflow Direction = "outflow"
)

// AccountBalance is a selected account balance in the planning currency.
// Spendable balances may be negative (for example, an overdraft); liability
// balances are represented as positive amounts owed by the profile layer.
type AccountBalance struct {
	Account string
	Amount  ledger.Decimal
}

// Plan is an owner-confirmed future assumption. It is intentionally not a
// ledger transaction. Amount is always positive; Direction carries its sign.
type Plan struct {
	ID         string
	Name       string
	Date       ledger.Date
	Amount     ledger.Decimal
	Currency   string // required by the source-ledger representation
	Direction  Direction
	Commitment Commitment // required for outflows
	Account    string     // optional expected funding account
}

// Input supplies the resolved facts needed for one primary calculation. Today
// and RecordedThrough are calendar dates in the owner-confirmed planning
// timezone; timezone conversion belongs to the profile/UI boundary.
type Input struct {
	Today           ledger.Date
	HorizonEnd      ledger.Date
	RecordedThrough ledger.Date
	Currency        string
	SpendableFunds  []AccountBalance
	ShortTermDebts  []AccountBalance
	MinimumReserve  ledger.Decimal
	Plans           []Plan
}

// Point explains primary headroom at a dated timeline step. Inflows are
// retained in IgnoredInflows for transparency but never increase the primary
// result.
type Point struct {
	Date              ledger.Date
	CommittedOutflow  ledger.Decimal
	AdjustableOutflow ledger.Decimal
	IgnoredInflows    ledger.Decimal
	Headroom          ledger.Decimal
}

// Result is the exact, explanation-rich output of Calculate.
type Result struct {
	Currency             string
	CurrentFunds         ledger.Decimal
	CurrentShortTermDebt ledger.Decimal
	MinimumReserve       ledger.Decimal
	Timeline             []Point
	LiquidityLowPoint    Point
	PrimarySafeToSpend   ledger.Decimal
	FundingShortfall     ledger.Decimal
	CurrentPlanning      bool
}

// Calculate applies the primary conservative algorithm. It only spends money
// already held: future inflows are visible in timeline points but are not added
// to headroom. All confirmed outflows through each date are subtracted,
// including adjustable ones; the lowest dated headroom determines the result.
func Calculate(input Input) (Result, error) {
	return calculate(input, false)
}

func calculate(input Input, includeFutureInflows bool) (Result, error) {
	if err := validateInput(input); err != nil {
		return Result{}, err
	}

	funds, err := sumBalances("spendable funds", input.SpendableFunds, true)
	if err != nil {
		return Result{}, err
	}
	debts, err := sumBalances("short-term debts", input.ShortTermDebts, false)
	if err != nil {
		return Result{}, err
	}

	byDate := map[string][]Plan{}
	for _, plan := range input.Plans {
		if err := validatePlan(plan); err != nil {
			return Result{}, err
		}
		if compareDate(plan.Date, input.Today) < 0 || compareDate(plan.Date, input.HorizonEnd) > 0 {
			continue
		}
		byDate[plan.Date.Raw] = append(byDate[plan.Date.Raw], plan)
	}

	datesByRaw := map[string]ledger.Date{input.Today.Raw: input.Today}
	for _, plans := range byDate {
		datesByRaw[plans[0].Date.Raw] = plans[0].Date
	}
	dates := make([]ledger.Date, 0, len(datesByRaw))
	for _, date := range datesByRaw {
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool { return compareDate(dates[i], dates[j]) < 0 })

	committed, adjustable, ignoredInflows := ledger.Zero(), ledger.Zero(), ledger.Zero()
	base := funds.Sub(debts).Sub(input.MinimumReserve)
	result := Result{
		Currency:             strings.TrimSpace(input.Currency),
		CurrentFunds:         funds,
		CurrentShortTermDebt: debts,
		MinimumReserve:       input.MinimumReserve,
		CurrentPlanning:      compareDate(input.RecordedThrough, input.Today) == 0,
	}
	for _, date := range dates {
		// Outflows always precede inflows conceptually. Primary headroom ignores
		// inflows entirely, but keeping their sum makes the conservative choice
		// visible to callers and future scenario evaluation.
		for _, plan := range byDate[date.Raw] {
			switch plan.Direction {
			case Outflow:
				if plan.Commitment == Committed {
					committed = committed.Add(plan.Amount)
				} else {
					adjustable = adjustable.Add(plan.Amount)
				}
			case Inflow:
				ignoredInflows = ignoredInflows.Add(plan.Amount)
			}
		}
		point := Point{
			Date:              date,
			CommittedOutflow:  committed,
			AdjustableOutflow: adjustable,
			IgnoredInflows:    ignoredInflows,
			Headroom:          base.Sub(committed).Sub(adjustable),
		}
		if includeFutureInflows {
			point.Headroom = point.Headroom.Add(ignoredInflows)
		}
		result.Timeline = append(result.Timeline, point)
	}

	result.LiquidityLowPoint = result.Timeline[0]
	for _, point := range result.Timeline[1:] {
		if point.Headroom.Cmp(result.LiquidityLowPoint.Headroom) < 0 {
			result.LiquidityLowPoint = point
		}
	}
	if result.LiquidityLowPoint.Headroom.Sign() < 0 {
		result.FundingShortfall = result.LiquidityLowPoint.Headroom.Neg()
		result.PrimarySafeToSpend = ledger.Zero()
	} else {
		result.PrimarySafeToSpend = result.LiquidityLowPoint.Headroom
		result.FundingShortfall = ledger.Zero()
	}
	return result, nil
}

func validateInput(input Input) error {
	if !input.Today.Valid() || !input.HorizonEnd.Valid() || !input.RecordedThrough.Valid() {
		return fmt.Errorf("planning dates must be valid")
	}
	if compareDate(input.HorizonEnd, input.Today) < 0 {
		return fmt.Errorf("planning horizon ends before today")
	}
	if strings.TrimSpace(input.Currency) == "" {
		return fmt.Errorf("planning currency is required")
	}
	if input.MinimumReserve.Sign() < 0 {
		return fmt.Errorf("minimum reserve cannot be negative")
	}
	return nil
}

func validatePlan(plan Plan) error {
	if !plan.Date.Valid() {
		return fmt.Errorf("plan %q has invalid date", plan.ID)
	}
	if plan.Amount.Sign() <= 0 {
		return fmt.Errorf("plan %q amount must be positive", plan.ID)
	}
	switch plan.Direction {
	case Inflow:
		return nil
	case Outflow:
		if plan.Commitment != Committed && plan.Commitment != Adjustable {
			return fmt.Errorf("outflow plan %q must be committed or adjustable", plan.ID)
		}
		return nil
	default:
		return fmt.Errorf("plan %q has invalid direction", plan.ID)
	}
}

func sumBalances(kind string, balances []AccountBalance, allowNegative bool) (ledger.Decimal, error) {
	total := ledger.Zero()
	for _, balance := range balances {
		if strings.TrimSpace(balance.Account) == "" {
			return ledger.Zero(), fmt.Errorf("%s contain an unnamed account", kind)
		}
		if !allowNegative && balance.Amount.Sign() < 0 {
			return ledger.Zero(), fmt.Errorf("%s account %q has a negative amount", kind, balance.Account)
		}
		total = total.Add(balance.Amount)
	}
	return total, nil
}

func compareDate(left, right ledger.Date) int {
	if left.Year != right.Year {
		if left.Year < right.Year {
			return -1
		}
		return 1
	}
	if left.Month != right.Month {
		if left.Month < right.Month {
			return -1
		}
		return 1
	}
	if left.Day < right.Day {
		return -1
	}
	if left.Day > right.Day {
		return 1
	}
	return 0
}
