// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

package planning

import "testing"

func TestSuggestMatchesOnlyReturnsExplainableCandidates(t *testing.T) {
	plans := []Plan{{ID: "rent", Name: "房租", Date: date("2026-08-01"), Amount: decimal(t, "5000"), Direction: Outflow, Commitment: Committed, Account: "Assets:CMB"}}
	occurrences := []Occurrence{
		{ID: "match", Date: date("2026-08-02"), Name: "房租", Account: "Assets:CMB", Amount: decimal(t, "5000")},
		{ID: "wrong-amount", Date: date("2026-08-02"), Name: "房租", Account: "Assets:CMB", Amount: decimal(t, "5001")},
		{ID: "too-late", Date: date("2026-08-10"), Name: "房租", Account: "Assets:CMB", Amount: decimal(t, "5000")},
	}
	suggestions := SuggestMatches(plans, occurrences, 3)
	if len(suggestions) != 1 {
		t.Fatalf("suggestions=%+v", suggestions)
	}
	if suggestion := suggestions[0]; !suggestion.SameName || !suggestion.SameAccount || !suggestion.SameAmount || suggestion.DateDistance != 1 {
		t.Fatalf("suggestion=%+v", suggestion)
	}
}
