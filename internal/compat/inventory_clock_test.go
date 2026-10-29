// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package compat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// A golden whose query reads the wall clock must carry a clock dynamic marker,
// and a clock marker must point at a query that actually reads one. Without this
// pairing a new time-varying function is compared literally, passes on its
// capture day, and fails every day after — which is what happened to
// function.today.0 before the marker was emitted.
func TestClockDependentGoldensCarryDynamicMarkers(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "testdata", "golden", "inventory", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Skip("inventory goldens not present")
	}
	clockPatterns := make([]*regexp.Regexp, 0, len(clockFunctions))
	for _, name := range clockFunctions {
		clockPatterns = append(clockPatterns, regexp.MustCompile(`\b`+name+`\s*\(`))
	}
	checked := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var golden struct {
			ID      string `json:"id"`
			Query   string `json:"query"`
			Dynamic string `json:"dynamic"`
		}
		if err := json.Unmarshal(data, &golden); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if golden.ID == "" {
			continue // index.json carries no capability
		}
		checked++
		readClock := false
		for _, pattern := range clockPatterns {
			if pattern.MatchString(golden.Query) {
				readClock = true
				break
			}
		}
		rule, marked := dynamicRules[golden.Dynamic]
		if readClock && !(marked && rule.clock) {
			t.Errorf("%s calls a clock function but carries dynamic=%q; the comparison would freeze a time-varying value", golden.ID, golden.Dynamic)
		}
		if marked && rule.clock && !readClock {
			t.Errorf("%s carries a clock dynamic marker but calls no clock function", golden.ID)
		}
	}
	if checked == 0 {
		t.Fatal("no inventory goldens were checked")
	}
}
