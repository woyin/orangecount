// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

package planning

import "testing"

func TestDetectPatternsRequiresExplainableExactCadence(t *testing.T) {
	patterns := DetectPatterns([]Occurrence{
		{ID: "1", Date: date("2026-01-01"), Account: "Assets:CMB", Name: " Netflix ", Amount: decimal(t, "50")},
		{ID: "2", Date: date("2026-01-31"), Account: "Assets:CMB", Name: "netflix", Amount: decimal(t, "55")},
		{ID: "3", Date: date("2026-03-02"), Account: "Assets:CMB", Name: "NETFLIX", Amount: decimal(t, "50")},
		{ID: "4", Date: date("2026-01-01"), Account: "Assets:CMB", Name: "irregular", Amount: decimal(t, "1")},
		{ID: "5", Date: date("2026-02-05"), Account: "Assets:CMB", Name: "irregular", Amount: decimal(t, "1")},
		{ID: "6", Date: date("2026-03-07"), Account: "Assets:CMB", Name: "irregular", Amount: decimal(t, "1")},
	}, 3)
	if len(patterns) != 1 {
		t.Fatalf("patterns=%+v", patterns)
	}
	pattern := patterns[0]
	if pattern.Name != "netflix" || pattern.CadenceDays != 30 || pattern.MinimumAmount.String() != "50" || pattern.MaximumAmount.String() != "55" {
		t.Fatalf("pattern=%+v", pattern)
	}
}
