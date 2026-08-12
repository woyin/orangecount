// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

package planning

// MatchSuggestion is evidence for an owner to confirm a plan fulfillment. It
// never mutates a plan, an occurrence, or a safe-to-spend result.
type MatchSuggestion struct {
	PlanID       string
	OccurrenceID string
	SameName     bool
	SameAccount  bool
	SameAmount   bool
	DateDistance int
}

// SuggestMatches returns deterministic, explainable candidates. It requires
// exact amount plus at least name or account agreement, and keeps the date
// window deliberately small so the review flow asks rather than guesses.
func SuggestMatches(plans []Plan, occurrences []Occurrence, dateWindowDays int) []MatchSuggestion {
	if dateWindowDays < 0 {
		dateWindowDays = 0
	}
	var suggestions []MatchSuggestion
	for _, plan := range plans {
		if plan.Direction != Outflow || !plan.Date.Valid() || plan.Amount.Sign() <= 0 {
			continue
		}
		for _, occurrence := range occurrences {
			if !occurrence.Date.Valid() || occurrence.Amount.Sign() <= 0 {
				continue
			}
			distance := daysBetween(plan.Date, occurrence.Date)
			if distance < 0 {
				distance = -distance
			}
			if distance > dateWindowDays {
				continue
			}
			sameName := normalizeName(plan.Name) != "" && normalizeName(plan.Name) == normalizeName(occurrence.Name)
			sameAccount := plan.Account != "" && plan.Account == occurrence.Account
			sameAmount := plan.Amount.Equal(occurrence.Amount)
			if sameAmount && (sameName || sameAccount) {
				suggestions = append(suggestions, MatchSuggestion{PlanID: plan.ID, OccurrenceID: occurrence.ID, SameName: sameName, SameAccount: sameAccount, SameAmount: sameAmount, DateDistance: distance})
			}
		}
	}
	return suggestions
}
