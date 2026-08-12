// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

package planning

import (
	"fmt"
	"regexp"
	"strings"

	"orangecount/internal/ledger"
)

// ProfileDefinition is the write-time form of the planning profile. It keeps
// the source date separate from RecordedThrough: the former dates a source
// directive, while the latter declares how current the owner's ledger is.
type ProfileDefinition struct {
	Date                  ledger.Date
	ID                    string
	Currency              string
	Timezone              string
	MinimumReserve        ledger.Decimal
	SpendableAccounts     []string
	ShortTermDebtAccounts []string
	RecordedThrough       ledger.Date
}

// PlanRevisionDefinition is an append-only revision of a stable plan ID.
// Cancelled and fulfilled revisions intentionally require only ID, revision,
// and status; the preceding active revision remains the historical detail.
type PlanRevisionDefinition struct {
	Date     ledger.Date
	Plan     Plan
	Revision int
	Status   string // active, cancelled, or fulfilled
}

// SerializeProfileDefinition emits the frozen v1 planning-profile directive.
func SerializeProfileDefinition(definition ProfileDefinition) (string, error) {
	if !definition.Date.Valid() || !definition.RecordedThrough.Valid() {
		return "", fmt.Errorf("profile dates must be valid")
	}
	if strings.TrimSpace(definition.ID) == "" {
		return "", fmt.Errorf("profile ID is required")
	}
	if !currencyPattern.MatchString(strings.TrimSpace(definition.Currency)) {
		return "", fmt.Errorf("invalid planning currency %q", definition.Currency)
	}
	if strings.TrimSpace(definition.Timezone) == "" {
		return "", fmt.Errorf("planning timezone is required")
	}
	if definition.MinimumReserve.Sign() < 0 {
		return "", fmt.Errorf("minimum reserve cannot be negative")
	}
	if len(definition.SpendableAccounts) == 0 {
		return "", fmt.Errorf("at least one spendable account is required")
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("%s custom %q %q", definition.Date.Raw, profileCustomType, definition.ID))
	lines = append(lines, fmt.Sprintf("  currency: %s", strings.TrimSpace(definition.Currency)))
	lines = append(lines, fmt.Sprintf("  timezone: %q", escapeString(definition.Timezone)))
	lines = append(lines, fmt.Sprintf("  minimum_reserve: %s %s", definition.MinimumReserve.String(), strings.TrimSpace(definition.Currency)))
	for _, account := range definition.SpendableAccounts {
		if !accountPattern.MatchString(account) {
			return "", fmt.Errorf("invalid spendable account %q", account)
		}
		lines = append(lines, fmt.Sprintf("  spendable_account: %s", account))
	}
	for _, account := range definition.ShortTermDebtAccounts {
		if !accountPattern.MatchString(account) {
			return "", fmt.Errorf("invalid short-term debt account %q", account)
		}
		lines = append(lines, fmt.Sprintf("  short_term_debt_account: %s", account))
	}
	lines = append(lines, fmt.Sprintf("  recorded_through: %s", definition.RecordedThrough.Raw))
	return strings.Join(lines, "\n"), nil
}

// SerializePlanRevision emits one append-only plan revision. Active revisions
// carry all calculation fields; terminal status revisions deliberately do not
// duplicate them, preventing a cancellation from silently changing history.
func SerializePlanRevision(definition PlanRevisionDefinition) (string, error) {
	if !definition.Date.Valid() {
		return "", fmt.Errorf("revision date must be valid")
	}
	if strings.TrimSpace(definition.Plan.ID) == "" {
		return "", fmt.Errorf("plan ID is required")
	}
	if definition.Revision < 1 {
		return "", fmt.Errorf("revision must be positive")
	}
	status := strings.TrimSpace(definition.Status)
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "cancelled" && status != "fulfilled" {
		return "", fmt.Errorf("invalid plan status %q", status)
	}
	lines := []string{fmt.Sprintf("%s custom %q %q", definition.Date.Raw, planCustomType, definition.Plan.ID), fmt.Sprintf("  revision: %d", definition.Revision), fmt.Sprintf("  status: %q", status)}
	if status != "active" {
		return strings.Join(lines, "\n"), nil
	}
	if err := validatePlan(definition.Plan); err != nil {
		return "", err
	}
	if strings.TrimSpace(definition.Plan.Name) == "" {
		return "", fmt.Errorf("active plan requires name")
	}
	if !currencyPattern.MatchString(strings.TrimSpace(definition.Plan.Currency)) {
		return "", fmt.Errorf("active plan requires a valid currency")
	}
	lines = append(lines,
		fmt.Sprintf("  name: %q", escapeString(definition.Plan.Name)),
		fmt.Sprintf("  expected_date: %s", definition.Plan.Date.Raw),
		fmt.Sprintf("  amount: %s %s", definition.Plan.Amount.String(), definition.Plan.Currency),
		fmt.Sprintf("  direction: %q", definition.Plan.Direction),
	)
	if definition.Plan.Direction == Outflow {
		lines = append(lines, fmt.Sprintf("  commitment: %q", definition.Plan.Commitment))
	}
	if account := strings.TrimSpace(definition.Plan.Account); account != "" {
		if !accountPattern.MatchString(account) {
			return "", fmt.Errorf("invalid plan account %q", account)
		}
		lines = append(lines, fmt.Sprintf("  account: %s", account))
	}
	return strings.Join(lines, "\n"), nil
}

var (
	currencyPattern = regexp.MustCompile(`\A[A-Z][A-Z0-9'._-]*\z`)
	accountPattern  = regexp.MustCompile(`\A[A-Z][A-Za-z0-9\-]*(?::[A-Z][A-Za-z0-9\-]*)+\z`)
)

func escapeString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}
