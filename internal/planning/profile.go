// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

package planning

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"orangecount/internal/ledger"
)

// SchemaVersion is the frozen public source-ledger schema version. Changing a
// field's meaning requires a new directive suffix rather than reinterpretation.
const SchemaVersion = "v1"

const (
	profileCustomType = "orangecount.planning-profile." + SchemaVersion
	planCustomType    = "orangecount.planned-flow." + SchemaVersion
)

// Problem is a planning-only configuration diagnostic. A problem disables the
// affected planning rule but never makes the accounting evaluation invalid.
type Problem struct {
	Code    string
	Message string
	Source  string
}

// Profile is the effective owner-confirmed planning configuration. The
// profile parser does not infer any of these values from transaction history.
type Profile struct {
	ID                    string
	Currency              string
	Timezone              string
	MinimumReserve        ledger.Decimal
	SpendableAccounts     []string
	ShortTermDebtAccounts []string
	RecordedThrough       ledger.Date
	Plans                 []Plan
	Problems              []Problem
}

// EffectiveProfile parses OrangeCount planning directives from an evaluated
// ledger. Only the selected, versioned schema is used; unknown planning schema
// versions are surfaced as isolated problems.
func EffectiveProfile(evaluation *ledger.Evaluation) Profile {
	var result Profile
	if evaluation == nil {
		return result
	}
	var profileCandidates []ledger.Custom
	plansByID := map[string][]planRevision{}
	for _, entry := range evaluation.Entries {
		custom, ok := entry.Directive.(ledger.Custom)
		if !ok {
			continue
		}
		switch custom.Type {
		case profileCustomType:
			profileCandidates = append(profileCandidates, custom)
		case planCustomType:
			revision, problem, ok := parsePlanCustom(custom)
			if problem != nil {
				result.Problems = append(result.Problems, *problem)
			}
			if ok {
				plansByID[revision.plan.ID] = append(plansByID[revision.plan.ID], revision)
			}
		default:
			if strings.HasPrefix(custom.Type, "orangecount.planning-profile.") || strings.HasPrefix(custom.Type, "orangecount.planned-flow.") {
				result.Problems = append(result.Problems, Problem{Code: "W-PLANNING-SCHEMA-UNSUPPORTED", Message: fmt.Sprintf("unsupported planning schema %q; directive ignored", custom.Type), Source: custom.Type})
			}
		}
	}
	if len(profileCandidates) == 0 {
		result.Problems = append(result.Problems, Problem{Code: "W-PLANNING-PROFILE-MISSING", Message: "no planning profile is configured", Source: profileCustomType})
	} else if len(profileCandidates) > 1 {
		result.Problems = append(result.Problems, Problem{Code: "W-PLANNING-PROFILE-AMBIGUOUS", Message: "multiple planning profiles are configured; planning is disabled", Source: profileCustomType})
	} else {
		profile, problems := parseProfileCustom(profileCandidates[0])
		result.ID, result.Currency, result.Timezone = profile.ID, profile.Currency, profile.Timezone
		result.MinimumReserve, result.SpendableAccounts, result.ShortTermDebtAccounts, result.RecordedThrough = profile.MinimumReserve, profile.SpendableAccounts, profile.ShortTermDebtAccounts, profile.RecordedThrough
		result.Problems = append(result.Problems, problems...)
	}
	for _, revisions := range plansByID {
		plan, problem, ok := effectivePlan(revisions)
		if problem != nil {
			result.Problems = append(result.Problems, *problem)
		}
		if ok {
			result.Plans = append(result.Plans, plan)
		}
	}
	sort.Slice(result.Plans, func(i, j int) bool {
		if result.Plans[i].Date.Raw == result.Plans[j].Date.Raw {
			return result.Plans[i].ID < result.Plans[j].ID
		}
		return compareDate(result.Plans[i].Date, result.Plans[j].Date) < 0
	})
	return result
}

// BuildInput resolves the effective profile's selected ledger accounts into a
// calculation input. Liability balances use Beancount's normal negative sign;
// a positive liability balance is a credit and does not count as debt owed.
func BuildInput(evaluation *ledger.Evaluation, profile Profile, today, horizonEnd ledger.Date) (Input, error) {
	if evaluation == nil {
		return Input{}, fmt.Errorf("planning requires a ledger evaluation")
	}
	if profile.Currency == "" || !profile.RecordedThrough.Valid() {
		return Input{}, fmt.Errorf("planning profile is incomplete")
	}
	input := Input{Today: today, HorizonEnd: horizonEnd, RecordedThrough: profile.RecordedThrough, Currency: profile.Currency, MinimumReserve: profile.MinimumReserve, Plans: append([]Plan(nil), profile.Plans...)}
	for _, account := range profile.SpendableAccounts {
		state, ok := evaluation.Account(account)
		if !ok {
			return Input{}, fmt.Errorf("configured spendable account %q does not exist", account)
		}
		input.SpendableFunds = append(input.SpendableFunds, AccountBalance{Account: account, Amount: state.Balances[profile.Currency]})
	}
	for _, account := range profile.ShortTermDebtAccounts {
		state, ok := evaluation.Account(account)
		if !ok {
			return Input{}, fmt.Errorf("configured short-term debt account %q does not exist", account)
		}
		balance := state.Balances[profile.Currency]
		if balance.Sign() < 0 {
			balance = balance.Neg()
		} else {
			balance = ledger.Zero()
		}
		input.ShortTermDebts = append(input.ShortTermDebts, AccountBalance{Account: account, Amount: balance})
	}
	return input, nil
}

type profileData struct {
	ID                    string
	Currency              string
	Timezone              string
	MinimumReserve        ledger.Decimal
	SpendableAccounts     []string
	ShortTermDebtAccounts []string
	RecordedThrough       ledger.Date
}

func parseProfileCustom(custom ledger.Custom) (profileData, []Problem) {
	var profile profileData
	var problems []Problem
	if len(custom.Values) != 1 || custom.Values[0].Kind != ledger.ValueString || strings.TrimSpace(custom.Values[0].String) == "" {
		return profile, []Problem{{Code: "W-PLANNING-PROFILE-ID", Message: "planning profile requires one non-empty string ID", Source: custom.Type}}
	}
	profile.ID = strings.TrimSpace(custom.Values[0].String)
	for _, meta := range custom.Meta {
		switch meta.Key {
		case "currency":
			profile.Currency = stringValue(meta.Value)
		case "timezone":
			profile.Timezone = stringValue(meta.Value)
		case "minimum_reserve":
			amount, ok := amountValue(meta.Value)
			if !ok || amount.Currency == "" || amount.Number.Raw == "" {
				problems = append(problems, problem("W-PLANNING-RESERVE", "minimum_reserve must be an amount", custom))
				continue
			}
			profile.MinimumReserve = ledger.DecimalFromNumber(amount.Number)
			if profile.MinimumReserve.Sign() < 0 {
				problems = append(problems, problem("W-PLANNING-RESERVE", "minimum_reserve cannot be negative", custom))
			}
		case "spendable_account":
			if account := accountValue(meta.Value); account != "" {
				profile.SpendableAccounts = append(profile.SpendableAccounts, account)
			} else {
				problems = append(problems, problem("W-PLANNING-SPENDABLE-ACCOUNT", "spendable_account must be an account", custom))
			}
		case "short_term_debt_account":
			if account := accountValue(meta.Value); account != "" {
				profile.ShortTermDebtAccounts = append(profile.ShortTermDebtAccounts, account)
			} else {
				problems = append(problems, problem("W-PLANNING-DEBT-ACCOUNT", "short_term_debt_account must be an account", custom))
			}
		case "recorded_through":
			if meta.Value.Kind == ledger.ValueDate && meta.Value.Date.Valid() {
				profile.RecordedThrough = meta.Value.Date
			} else {
				problems = append(problems, problem("W-PLANNING-RECORDED-THROUGH", "recorded_through must be a date", custom))
			}
		}
	}
	if profile.Currency == "" {
		problems = append(problems, problem("W-PLANNING-CURRENCY", "planning profile requires currency", custom))
	}
	if profile.Timezone == "" {
		problems = append(problems, problem("W-PLANNING-TIMEZONE", "planning profile requires timezone", custom))
	}
	if !profile.RecordedThrough.Valid() {
		problems = append(problems, problem("W-PLANNING-RECORDED-THROUGH", "planning profile requires recorded_through date", custom))
	}
	if len(profile.SpendableAccounts) == 0 {
		problems = append(problems, problem("W-PLANNING-SPENDABLE-ACCOUNT", "planning profile requires at least one spendable account", custom))
	}
	return profile, problems
}

type planRevision struct {
	plan     Plan
	revision int
	status   string
}

func parsePlanCustom(custom ledger.Custom) (planRevision, *Problem, bool) {
	if len(custom.Values) != 1 || custom.Values[0].Kind != ledger.ValueString || strings.TrimSpace(custom.Values[0].String) == "" {
		return planRevision{}, ptr(problem("W-PLANNING-PLAN-ID", "planned flow requires one non-empty string ID", custom)), false
	}
	revision := planRevision{plan: Plan{ID: strings.TrimSpace(custom.Values[0].String)}, revision: 1, status: "active"}
	for _, meta := range custom.Meta {
		switch meta.Key {
		case "revision":
			if meta.Value.Kind != ledger.ValueNumber {
				return planRevision{}, ptr(problem("W-PLANNING-PLAN-REVISION", "revision must be a positive integer", custom)), false
			}
			value, err := strconv.Atoi(meta.Value.Number.Raw)
			if err != nil || value < 1 {
				return planRevision{}, ptr(problem("W-PLANNING-PLAN-REVISION", "revision must be a positive integer", custom)), false
			}
			revision.revision = value
		case "name":
			revision.plan.Name = stringValue(meta.Value)
		case "expected_date":
			if meta.Value.Kind == ledger.ValueDate && meta.Value.Date.Valid() {
				revision.plan.Date = meta.Value.Date
			} else {
				return planRevision{}, ptr(problem("W-PLANNING-PLAN-DATE", "expected_date must be a date", custom)), false
			}
		case "amount":
			amount, ok := amountValue(meta.Value)
			if !ok {
				return planRevision{}, ptr(problem("W-PLANNING-PLAN-AMOUNT", "amount must be an amount", custom)), false
			}
			revision.plan.Amount = ledger.DecimalFromNumber(amount.Number)
		case "direction":
			revision.plan.Direction = Direction(stringValue(meta.Value))
		case "commitment":
			revision.plan.Commitment = Commitment(stringValue(meta.Value))
		case "account":
			revision.plan.Account = accountValue(meta.Value)
		case "status":
			revision.status = stringValue(meta.Value)
		}
	}
	if revision.status == "cancelled" || revision.status == "fulfilled" {
		return revision, nil, true
	}
	if revision.status != "active" {
		return planRevision{}, ptr(problem("W-PLANNING-PLAN-STATUS", "status must be active, cancelled, or fulfilled", custom)), false
	}
	if revision.plan.Name == "" {
		return planRevision{}, ptr(problem("W-PLANNING-PLAN-NAME", "planned flow requires name", custom)), false
	}
	if err := validatePlan(revision.plan); err != nil {
		return planRevision{}, ptr(problem("W-PLANNING-PLAN", err.Error(), custom)), false
	}
	return revision, nil, true
}

func effectivePlan(revisions []planRevision) (Plan, *Problem, bool) {
	sort.Slice(revisions, func(i, j int) bool { return revisions[i].revision < revisions[j].revision })
	seen := make(map[int]bool, len(revisions))
	for _, revision := range revisions {
		if seen[revision.revision] {
			return Plan{}, &Problem{Code: "W-PLANNING-PLAN-AMBIGUOUS", Message: fmt.Sprintf("multiple revisions for plan %q have the same revision", revision.plan.ID), Source: revision.plan.ID}, false
		}
		seen[revision.revision] = true
	}
	latest := revisions[len(revisions)-1]
	if latest.status != "active" {
		return Plan{}, nil, false
	}
	return latest.plan, nil, true
}

func stringValue(value ledger.Value) string {
	if value.Kind == ledger.ValueString || value.Kind == ledger.ValueCurrency {
		return strings.TrimSpace(value.String)
	}
	return ""
}

func accountValue(value ledger.Value) string {
	if value.Kind == ledger.ValueAccount {
		return strings.TrimSpace(value.String)
	}
	return ""
}

func amountValue(value ledger.Value) (ledger.Amount, bool) {
	if value.Kind != ledger.ValueAmount || value.Amount.Number.Raw == "" {
		return ledger.Amount{}, false
	}
	return value.Amount, true
}

func problem(code, message string, custom ledger.Custom) Problem {
	return Problem{Code: code, Message: message, Source: custom.Type}
}
func ptr(value Problem) *Problem { return &value }
