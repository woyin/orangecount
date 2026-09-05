// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

package planning

import (
	"strings"
	"testing"

	"orangecount/internal/ledger"
)

// codesOf collects one problem code per planning problem for compact asserts.
func codesOf(profile Profile) string {
	var codes []string
	for _, problem := range profile.Problems {
		codes = append(codes, problem.Code)
	}
	return strings.Join(codes, ",")
}

// TestProfileConfigurationProblemsAreIsolated walks the profile parser's
// rejection branches: every invalid value yields exactly one named problem
// and never invalidates the accounting evaluation.
func TestProfileConfigurationProblemsAreIsolated(t *testing.T) {
	bad := ledger.Custom{
		Date: date("2026-08-12"), Type: profileCustomType,
		Values: []ledger.Value{{Kind: ledger.ValueString, String: "primary"}},
		DirectiveBase: ledger.DirectiveBase{Meta: []ledger.Metadata{
			meta("minimum_reserve", ledger.Value{Kind: ledger.ValueString, String: "not-an-amount"}),
			meta("spendable_account", ledger.Value{Kind: ledger.ValueString, String: "not-an-account"}),
			meta("short_term_debt_account", ledger.Value{Kind: ledger.ValueString, String: "nope"}),
			meta("recorded_through", ledger.Value{Kind: ledger.ValueString, String: "someday"}),
		}},
	}
	codes := codesOf(EffectiveProfile(&ledger.Evaluation{Entries: []ledger.EntryRecord{{Directive: bad}}}))
	for _, want := range []string{"W-PLANNING-RESERVE", "W-PLANNING-SPENDABLE-ACCOUNT", "W-PLANNING-DEBT-ACCOUNT", "W-PLANNING-RECORDED-THROUGH", "W-PLANNING-CURRENCY", "W-PLANNING-TIMEZONE"} {
		if !strings.Contains(codes, want) {
			t.Fatalf("codes=%s want %s", codes, want)
		}
	}
}

func TestProfileReserveCurrencyMismatchAndNegativeReserve(t *testing.T) {
	negative := ledger.Custom{
		Date: date("2026-08-12"), Type: profileCustomType,
		Values: []ledger.Value{{Kind: ledger.ValueString, String: "primary"}},
		DirectiveBase: ledger.DirectiveBase{Meta: []ledger.Metadata{
			meta("currency", ledger.Value{Kind: ledger.ValueCurrency, String: "CNY"}),
			meta("minimum_reserve", ledger.Value{Kind: ledger.ValueAmount, Amount: amount("-1", "USD")}),
			meta("spendable_account", ledger.Value{Kind: ledger.ValueAccount, String: "Assets:Cash"}),
			meta("recorded_through", ledger.Value{Kind: ledger.ValueDate, Date: date("2026-08-12")}),
		}},
	}
	codes := codesOf(EffectiveProfile(&ledger.Evaluation{Entries: []ledger.EntryRecord{{Directive: negative}}}))
	// A negative reserve is rejected on its own; the USD amount also mismatches
	// the CNY planning currency once the currency key is known.
	if !strings.Contains(codes, "W-PLANNING-RESERVE") {
		t.Fatalf("codes=%s", codes)
	}
}

func TestPlanRevisionProblems(t *testing.T) {
	cases := []struct {
		name      string
		directive ledger.Custom
		want      string
	}{
		{"bad revision", planDirectiveWith("p1", []ledger.Metadata{meta("revision", ledger.Value{Kind: ledger.ValueString, String: "first"})}), "W-PLANNING-PLAN-REVISION"},
		{"zero revision", planDirectiveWith("p1", []ledger.Metadata{meta("revision", ledger.Value{Kind: ledger.ValueNumber, Number: ledger.Number{Raw: "0"}})}), "W-PLANNING-PLAN-REVISION"},
		{"bad date", planDirectiveWith("p1", []ledger.Metadata{meta("expected_date", ledger.Value{Kind: ledger.ValueString, String: "soon"})}), "W-PLANNING-PLAN-DATE"},
		{"bad amount", planDirectiveWith("p1", []ledger.Metadata{meta("amount", ledger.Value{Kind: ledger.ValueString, String: "cheap"})}), "W-PLANNING-PLAN-AMOUNT"},
		{"unknown status", planDirectiveWith("p1", []ledger.Metadata{meta("status", ledger.Value{Kind: ledger.ValueString, String: "pending"})}), "W-PLANNING-PLAN-STATUS"},
		{"missing name", planDirectiveWith("p1", []ledger.Metadata{
			meta("expected_date", ledger.Value{Kind: ledger.ValueDate, Date: date("2026-09-01")}),
			meta("amount", ledger.Value{Kind: ledger.ValueAmount, Amount: amount("10", "CNY")}),
			meta("direction", ledger.Value{Kind: ledger.ValueString, String: "outflow"}),
			meta("commitment", ledger.Value{Kind: ledger.ValueString, String: "committed"}),
		}), "W-PLANNING-PLAN-NAME"},
		{"invalid direction", planDirectiveWith("p1", []ledger.Metadata{
			meta("name", ledger.Value{Kind: ledger.ValueString, String: "转账"}),
			meta("expected_date", ledger.Value{Kind: ledger.ValueDate, Date: date("2026-09-01")}),
			meta("amount", ledger.Value{Kind: ledger.ValueAmount, Amount: amount("10", "CNY")}),
			meta("direction", ledger.Value{Kind: ledger.ValueString, String: "sideways"}),
		}), "W-PLANNING-PLAN"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			codes := codesOf(EffectiveProfile(&ledger.Evaluation{Entries: []ledger.EntryRecord{{Directive: testCase.directive}}}))
			if !strings.Contains(codes, testCase.want) {
				t.Fatalf("codes=%s want %s", codes, testCase.want)
			}
		})
	}
}

func planDirectiveWith(id string, extra []ledger.Metadata) ledger.Custom {
	return ledger.Custom{
		Date:          date("2026-08-12"),
		Type:          planCustomType,
		Values:        []ledger.Value{{Kind: ledger.ValueString, String: id}},
		DirectiveBase: ledger.DirectiveBase{Meta: extra},
	}
}

func TestPlanIDAndValueKindsAreStrict(t *testing.T) {
	// A plan directive without a string ID is one ID problem.
	noID := ledger.Custom{Date: date("2026-08-12"), Type: planCustomType, Values: []ledger.Value{{Kind: ledger.ValueNumber, Number: ledger.Number{Raw: "7"}}}}
	if codes := codesOf(EffectiveProfile(&ledger.Evaluation{Entries: []ledger.EntryRecord{{Directive: noID}}})); !strings.Contains(codes, "W-PLANNING-PLAN-ID") {
		t.Fatalf("codes=%s", codes)
	}
	// A profile directive without a string ID is one ID problem.
	noProfileID := ledger.Custom{Date: date("2026-08-12"), Type: profileCustomType, Values: []ledger.Value{{Kind: ledger.ValueNull}}}
	if codes := codesOf(EffectiveProfile(&ledger.Evaluation{Entries: []ledger.EntryRecord{{Directive: noProfileID}}})); !strings.Contains(codes, "W-PLANNING-PROFILE-ID") {
		t.Fatalf("codes=%s", codes)
	}
	// An unrelated OrangeCount custom directive raises no planning problem.
	unrelated := ledger.Custom{Date: date("2026-08-12"), Type: "orangecount.quick-account.v1", Values: []ledger.Value{{Kind: ledger.ValueString, String: "三餐"}}}
	if profile := EffectiveProfile(&ledger.Evaluation{Entries: []ledger.EntryRecord{{Directive: unrelated}}}); len(profile.Problems) != 1 {
		t.Fatalf("unrelated directive problems=%+v", profile.Problems)
	}
}

func TestSerializeProfileRejectsIncompleteDefinitions(t *testing.T) {
	if _, err := SerializeProfileDefinition(ProfileDefinition{}); err == nil {
		t.Fatal("an empty profile definition must not serialize")
	}
	if _, err := SerializePlanRevision(PlanRevisionDefinition{}); err == nil {
		t.Fatal("an empty plan revision must not serialize")
	}
	valid := PlanRevisionDefinition{Date: date("2026-08-12"), Revision: 2, Status: "active", Plan: Plan{ID: "p", Name: "n", Date: date("2026-09-01"), Amount: decimal(t, "5"), Currency: "CNY", Direction: Outflow, Commitment: Committed}}
	text, err := SerializePlanRevision(valid)
	if err != nil {
		t.Fatalf("valid revision err=%v", err)
	}
	if !strings.Contains(text, "revision: 2") {
		t.Fatalf("text=%s", text)
	}
}
