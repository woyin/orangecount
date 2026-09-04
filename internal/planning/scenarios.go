// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

package planning

// ScenarioOptions makes every relaxation of the primary calculation explicit.
// It never modifies Input or source-ledger configuration.
type ScenarioOptions struct {
	IncludeFutureInflows bool
	ReleaseAdjustable    bool
	Draft                *Plan
}

// ScenarioResult presents an ephemeral affordability comparison. Result uses
// the same exact arithmetic and low-point semantics as the primary result.
type ScenarioResult struct {
	Result                Result
	IncludesFutureInflows bool
	ReleasesAdjustable    bool
	Draft                 *Plan
}

// EvaluateScenario calculates an explicit what-if view. A draft is appended
// only to a copy of the input and therefore cannot persist or affect the
// primary safe-to-spend result.
func EvaluateScenario(input Input, options ScenarioOptions) (ScenarioResult, error) {
	scenario := input
	scenario.Plans = append([]Plan(nil), input.Plans...)
	if options.ReleaseAdjustable {
		kept := scenario.Plans[:0]
		for _, plan := range scenario.Plans {
			if plan.Direction == Outflow && plan.Commitment == Adjustable {
				continue
			}
			kept = append(kept, plan)
		}
		scenario.Plans = kept
	}
	if options.Draft != nil {
		draft := *options.Draft
		scenario.Plans = append(scenario.Plans, draft)
	}
	result, err := calculate(scenario, options.IncludeFutureInflows)
	if err != nil {
		return ScenarioResult{}, err
	}
	return ScenarioResult{Result: result, IncludesFutureInflows: options.IncludeFutureInflows, ReleasesAdjustable: options.ReleaseAdjustable, Draft: options.Draft}, nil
}
