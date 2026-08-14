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

func TestSerializeProfileDefinitionProducesParseableV1Directive(t *testing.T) {
	text, err := SerializeProfileDefinition(ProfileDefinition{
		Date: date("2026-08-12"), ID: "primary", Currency: "CNY", Timezone: "Asia/Singapore", MinimumReserve: decimal(t, "20000"),
		SpendableAccounts: []string{"Assets:CMB:Checking", "Assets:WeChat"}, ShortTermDebtAccounts: []string{"Liabilities:CMB:CreditCard"}, RecordedThrough: date("2026-08-12"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, `custom "orangecount.planning-profile.v1" "primary"`) || !strings.Contains(text, "minimum_reserve: 20000 CNY") {
		t.Fatalf("text=%s", text)
	}
	file, bag := ledger.ParseText("planning.bean", []byte(text))
	if bag.HasErrors() || len(file.Directives) != 1 {
		t.Fatalf("diagnostics=%v directives=%d", bag.All(), len(file.Directives))
	}
}

func TestSerializePlanRevisionPreservesTerminalRevisionMinimality(t *testing.T) {
	active, err := SerializePlanRevision(PlanRevisionDefinition{Date: date("2026-08-12"), Revision: 1, Status: "active", Plan: Plan{ID: "rent", Name: "房租", Date: date("2026-09-01"), Amount: decimal(t, "5000"), Currency: "CNY", Direction: Outflow, Commitment: Committed, Account: "Assets:CMB:Checking"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(active, `commitment: "committed"`) || !strings.Contains(active, "amount: 5000") {
		t.Fatalf("active=%s", active)
	}
	cancelled, err := SerializePlanRevision(PlanRevisionDefinition{Date: date("2026-08-13"), Revision: 2, Status: "cancelled", Plan: Plan{ID: "rent"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cancelled, "amount:") || !strings.Contains(cancelled, `status: "cancelled"`) {
		t.Fatalf("cancelled=%s", cancelled)
	}
	file, bag := ledger.ParseText("planning.bean", []byte(active+"\n\n"+cancelled))
	if bag.HasErrors() || len(file.Directives) != 2 {
		t.Fatalf("diagnostics=%v directives=%d", bag.All(), len(file.Directives))
	}
}

func TestSerializePlanRevisionRoundTripsQuotedNames(t *testing.T) {
	name := `Rent "A" \ B`
	text, err := SerializePlanRevision(PlanRevisionDefinition{Date: date("2026-08-12"), Revision: 1, Status: "active", Plan: Plan{ID: "rent", Name: name, Date: date("2026-09-01"), Amount: decimal(t, "5000"), Currency: "CNY", Direction: Outflow, Commitment: Committed}})
	if err != nil {
		t.Fatal(err)
	}
	file, bag := ledger.ParseText("planning.bean", []byte(text))
	if bag.HasErrors() {
		t.Fatalf("diagnostics=%v", bag.All())
	}
	evaluation := &ledger.Evaluation{}
	for _, directive := range file.Directives {
		evaluation.Entries = append(evaluation.Entries, ledger.EntryRecord{Directive: directive})
	}
	profile := EffectiveProfile(evaluation)
	if len(profile.Plans) != 1 {
		t.Fatalf("plans=%+v problems=%+v", profile.Plans, profile.Problems)
	}
	if got := profile.Plans[0].Name; got != name {
		t.Fatalf("round-tripped name=%q, want %q", got, name)
	}
}

func TestSerializedPlanningDirectivesRoundTripThroughEffectiveProfile(t *testing.T) {
	profileText, err := SerializeProfileDefinition(ProfileDefinition{
		Date:              date("2026-08-12"),
		ID:                "primary",
		Currency:          "CNY",
		Timezone:          "Asia/Singapore",
		MinimumReserve:    decimal(t, "1000"),
		SpendableAccounts: []string{"Assets:CMB:Checking"},
		RecordedThrough:   date("2026-08-12"),
	})
	if err != nil {
		t.Fatal(err)
	}
	planText, err := SerializePlanRevision(PlanRevisionDefinition{Date: date("2026-08-12"), Revision: 1, Plan: Plan{ID: "rent", Name: "房租", Date: date("2026-09-01"), Amount: decimal(t, "5000"), Currency: "CNY", Direction: Outflow, Commitment: Committed}})
	if err != nil {
		t.Fatal(err)
	}
	file, bag := ledger.ParseText("planning.bean", []byte(profileText+"\n\n"+planText))
	if bag.HasErrors() {
		t.Fatalf("parse errors=%v", bag.All())
	}
	evaluation := &ledger.Evaluation{}
	for _, directive := range file.Directives {
		evaluation.Entries = append(evaluation.Entries, ledger.EntryRecord{Directive: directive})
	}
	profile := EffectiveProfile(evaluation)
	if len(profile.Problems) != 0 || len(profile.Plans) != 1 {
		t.Fatalf("profile=%+v", profile)
	}
	if got := profile.Plans[0].Currency; got != "CNY" {
		t.Fatalf("currency=%q", got)
	}
}

func TestSerializePlanningDefinitionsRejectUnsafeValues(t *testing.T) {
	if _, err := SerializeProfileDefinition(ProfileDefinition{Date: date("2026-08-12"), ID: "p", Currency: "CNY", Timezone: "Asia/Singapore", SpendableAccounts: []string{"Assets:Cash"}, RecordedThrough: date("2026-08-12"), MinimumReserve: decimal(t, "-1")}); err == nil {
		t.Fatal("negative reserve accepted")
	}
	if _, err := SerializePlanRevision(PlanRevisionDefinition{Date: date("2026-08-12"), Revision: 1, Status: "active", Plan: Plan{ID: "x", Name: "x", Date: date("2026-08-13"), Amount: decimal(t, "0"), Currency: "CNY", Direction: Outflow, Commitment: Committed}}); err == nil {
		t.Fatal("zero plan amount accepted")
	}
}
