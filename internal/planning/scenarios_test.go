// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

package planning

import "testing"

func TestEvaluateScenarioMakesRelaxationsAndDraftExplicit(t *testing.T) {
	input := Input{Today: date("2026-08-12"), HorizonEnd: date("2026-08-20"), RecordedThrough: date("2026-08-12"), Currency: "CNY", SpendableFunds: []AccountBalance{{Account: "Assets:Cash", Amount: decimal(t, "10000")}}, MinimumReserve: decimal(t, "1000"), Plans: []Plan{
		{ID: "food", Date: date("2026-08-13"), Amount: decimal(t, "2000"), Direction: Outflow, Commitment: Adjustable},
		{ID: "salary", Date: date("2026-08-14"), Amount: decimal(t, "5000"), Direction: Inflow},
	}}
	primary, err := Calculate(input)
	if err != nil {
		t.Fatal(err)
	}
	if got := primary.PrimarySafeToSpend.String(); got != "7000" {
		t.Fatalf("primary=%s", got)
	}
	draft := &Plan{ID: "laptop", Name: "laptop", Date: date("2026-08-15"), Amount: decimal(t, "9000"), Direction: Outflow, Commitment: Adjustable}
	scenario, err := EvaluateScenario(input, ScenarioOptions{IncludeFutureInflows: true, ReleaseAdjustable: true, Draft: draft})
	if err != nil {
		t.Fatal(err)
	}
	if got := scenario.Result.PrimarySafeToSpend.String(); got != "5000" {
		t.Fatalf("scenario=%s", got)
	}
	if !scenario.IncludesFutureInflows || !scenario.ReleasesAdjustable || scenario.Draft != draft {
		t.Fatalf("scenario flags=%+v", scenario)
	}
	if len(input.Plans) != 2 {
		t.Fatal("scenario mutated primary input")
	}
}
