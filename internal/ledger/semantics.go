// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package ledger

import "fmt"

// SemanticsProfile is fixed for the lifetime of an evaluation/session.
// The empty profile deliberately retains the pre-migration behavior.
type SemanticsProfile string

const (
	Legacy SemanticsProfile = "legacy"
	V3     SemanticsProfile = "v3"
)

func ParseSemanticsProfile(value string) (SemanticsProfile, error) {
	switch SemanticsProfile(value) {
	case "", Legacy:
		return Legacy, nil
	case V3:
		return V3, nil
	default:
		return "", fmt.Errorf("unsupported semantics profile %q (expected legacy or v3)", value)
	}
}

// EvaluationRevision must change when a profile's accounting rules change.
func (p SemanticsProfile) EvaluationRevision() string {
	if p == V3 {
		return "v3-core-4"
	}
	return "legacy-1"
}

func (o EvalOptions) Normalized() (EvalOptions, error) {
	profile, err := ParseSemanticsProfile(string(o.SemanticsProfile))
	if err != nil {
		return o, err
	}
	o.SemanticsProfile = profile
	// Historically evaluation always inferred decimal tolerance. Retain that
	// behavior; source options configure the inference multiplier in v3.
	o.InferDecimalTolerance = true
	return o, nil
}

func validBooking(value string) bool {
	switch value {
	case "STRICT", "STRICT_WITH_SIZE", "FIFO", "LIFO", "HIFO", "NONE", "AVERAGE":
		return true
	default:
		return false
	}
}

func (e *evaluator) activateAverage() {
	if !contains(e.result.ActiveExtensions, "average-booking") {
		e.result.ActiveExtensions = append(e.result.ActiveExtensions, "average-booking")
	}
}
