// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

package planning

import (
	"strconv"
	"testing"

	"orangecount/internal/ledger"
)

func TestEffectiveProfileAndBuildInputResolveLedgerBalances(t *testing.T) {
	evaluation := &ledger.Evaluation{
		Entries: []ledger.EntryRecord{
			{Directive: profileDirective()},
			{Directive: planDirective("rent-2026-09", 1, "active", "2026-09-01", "5000")},
		},
		Accounts: map[string]ledger.AccountState{
			"Assets:CMB:Checking":        {Balances: map[string]ledger.Decimal{"CNY": decimal(t, "20000")}},
			"Assets:WeChat":              {Balances: map[string]ledger.Decimal{"CNY": decimal(t, "3000")}},
			"Liabilities:CMB:CreditCard": {Balances: map[string]ledger.Decimal{"CNY": decimal(t, "-8000")}},
		},
	}
	profile := EffectiveProfile(evaluation)
	if len(profile.Problems) != 0 {
		t.Fatalf("problems=%+v", profile.Problems)
	}
	if profile.Currency != "CNY" || profile.Timezone != "Asia/Singapore" || profile.MinimumReserve.String() != "5000" {
		t.Fatalf("profile=%+v", profile)
	}
	if len(profile.Plans) != 1 || profile.Plans[0].ID != "rent-2026-09" {
		t.Fatalf("plans=%+v", profile.Plans)
	}
	input, err := BuildInput(evaluation, profile, date("2026-08-12"), date("2026-09-01"))
	if err != nil {
		t.Fatal(err)
	}
	if got := input.SpendableFunds[0].Amount.String(); got != "20000" {
		t.Fatalf("fund=%s", got)
	}
	if got := input.ShortTermDebts[0].Amount.String(); got != "8000" {
		t.Fatalf("debt=%s", got)
	}
	result, err := Calculate(input)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.PrimarySafeToSpend.String(); got != "5000" {
		t.Fatalf("primary=%s", got)
	}
}

func TestEffectiveProfileUsesLatestPlanRevisionAndExcludesCancelledPlan(t *testing.T) {
	evaluation := &ledger.Evaluation{Entries: []ledger.EntryRecord{
		{Directive: profileDirective()},
		{Directive: planDirective("rent", 1, "active", "2026-09-01", "5000")},
		{Directive: planDirective("rent", 2, "active", "2026-09-02", "5500")},
		{Directive: planDirective("trip", 1, "active", "2026-09-03", "1000")},
		{Directive: planStatusDirective("trip", 2, "cancelled")},
	}}
	profile := EffectiveProfile(evaluation)
	if len(profile.Plans) != 1 {
		t.Fatalf("plans=%+v", profile.Plans)
	}
	if got := profile.Plans[0].Amount.String(); got != "5500" {
		t.Fatalf("amount=%s", got)
	}
	if got := profile.Plans[0].Date.String(); got != "2026-09-02" {
		t.Fatalf("date=%s", got)
	}
}

func TestEffectiveProfileKeepsConfigurationProblemsIsolated(t *testing.T) {
	evaluation := &ledger.Evaluation{Entries: []ledger.EntryRecord{
		{Directive: profileDirective()},
		{Directive: ledger.Custom{Date: date("2026-08-12"), Type: "orangecount.planned-flow.v2"}},
		{Directive: planDirective("rent", 1, "active", "2026-09-01", "5000")},
		{Directive: planDirective("rent", 1, "active", "2026-09-02", "6000")},
	}}
	profile := EffectiveProfile(evaluation)
	if len(profile.Plans) != 0 {
		t.Fatalf("ambiguous plan was accepted: %+v", profile.Plans)
	}
	if !hasProblem(profile.Problems, "W-PLANNING-SCHEMA-UNSUPPORTED") || !hasProblem(profile.Problems, "W-PLANNING-PLAN-AMBIGUOUS") {
		t.Fatalf("problems=%+v", profile.Problems)
	}
}

func TestBuildInputRejectsUnknownConfiguredAccount(t *testing.T) {
	profile := EffectiveProfile(&ledger.Evaluation{Entries: []ledger.EntryRecord{{Directive: profileDirective()}}, Accounts: map[string]ledger.AccountState{}})
	if _, err := BuildInput(&ledger.Evaluation{Accounts: map[string]ledger.AccountState{}}, profile, date("2026-08-12"), date("2026-08-20")); err == nil {
		t.Fatal("BuildInput accepted a missing account")
	}
}

func profileDirective() ledger.Custom {
	return ledger.Custom{
		Date: date("2026-08-12"), Type: profileCustomType,
		Values: []ledger.Value{{Kind: ledger.ValueString, String: "primary"}},
		DirectiveBase: ledger.DirectiveBase{Meta: []ledger.Metadata{
			meta("currency", ledger.Value{Kind: ledger.ValueCurrency, String: "CNY"}),
			meta("timezone", ledger.Value{Kind: ledger.ValueString, String: "Asia/Singapore"}),
			meta("minimum_reserve", ledger.Value{Kind: ledger.ValueAmount, Amount: amount("5000", "CNY")}),
			meta("spendable_account", ledger.Value{Kind: ledger.ValueAccount, String: "Assets:CMB:Checking"}),
			meta("spendable_account", ledger.Value{Kind: ledger.ValueAccount, String: "Assets:WeChat"}),
			meta("short_term_debt_account", ledger.Value{Kind: ledger.ValueAccount, String: "Liabilities:CMB:CreditCard"}),
			meta("recorded_through", ledger.Value{Kind: ledger.ValueDate, Date: date("2026-08-12")}),
		}},
	}
}

func planDirective(id string, revision int, status, expectedDate, rawAmount string) ledger.Custom {
	return ledger.Custom{
		Date: date("2026-08-12"), Type: planCustomType,
		Values: []ledger.Value{{Kind: ledger.ValueString, String: id}},
		DirectiveBase: ledger.DirectiveBase{Meta: []ledger.Metadata{
			meta("revision", ledger.Value{Kind: ledger.ValueNumber, Number: ledger.Number{Raw: strconv.Itoa(revision)}}),
			meta("name", ledger.Value{Kind: ledger.ValueString, String: "房租"}),
			meta("expected_date", ledger.Value{Kind: ledger.ValueDate, Date: date(expectedDate)}),
			meta("amount", ledger.Value{Kind: ledger.ValueAmount, Amount: amount(rawAmount, "CNY")}),
			meta("direction", ledger.Value{Kind: ledger.ValueString, String: "outflow"}),
			meta("commitment", ledger.Value{Kind: ledger.ValueString, String: "committed"}),
			meta("status", ledger.Value{Kind: ledger.ValueString, String: status}),
		}},
	}
}

func planStatusDirective(id string, revision int, status string) ledger.Custom {
	return ledger.Custom{
		Date:   date("2026-08-12"),
		Type:   planCustomType,
		Values: []ledger.Value{{Kind: ledger.ValueString, String: id}},
		DirectiveBase: ledger.DirectiveBase{Meta: []ledger.Metadata{
			meta("revision", ledger.Value{Kind: ledger.ValueNumber, Number: ledger.Number{Raw: strconv.Itoa(revision)}}),
			meta("status", ledger.Value{Kind: ledger.ValueString, String: status}),
		}},
	}
}

func meta(key string, value ledger.Value) ledger.Metadata {
	return ledger.Metadata{Key: key, Value: value}
}

func amount(raw, currency string) ledger.Amount {
	value, _ := ledger.ParseDecimal(raw)
	return ledger.Amount{Number: ledger.Number{Raw: raw, Rat: value.Rat()}, Currency: currency}
}

func hasProblem(problems []Problem, code string) bool {
	for _, problem := range problems {
		if problem.Code == code {
			return true
		}
	}
	return false
}
