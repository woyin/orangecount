// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package repairguidance

import (
	"reflect"
	"testing"

	"orangecount/internal/diagnostic"
)

func TestCoverageMatchesReleasedDiagnostics(t *testing.T) {
	if err := ValidateCoverage(diagnostic.ReleasedErrorCodes()); err != nil {
		t.Fatal(err)
	}
	if got, want := Codes(), diagnostic.ReleasedErrorCodes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("codes=%v want=%v", got, want)
	}
}

func TestLookupLocalizesWithoutSharingMutableSlices(t *testing.T) {
	english, ok := Lookup("E-EVAL-UNBALANCED", LocaleEnglish)
	if !ok || english.Topic != "diagnostics/E-EVAL-UNBALANCED" || english.Phase != PhaseSemantic {
		t.Fatalf("english guide=%+v ok=%v", english, ok)
	}
	chinese, ok := Lookup("E-EVAL-UNBALANCED", LocaleChinese)
	if !ok || chinese.ShortAction == english.ShortAction {
		t.Fatalf("localized guide=%+v", chinese)
	}
	english.Inspect[0] = "changed"
	again, _ := Lookup("E-EVAL-UNBALANCED", LocaleEnglish)
	if again.Inspect[0] == "changed" {
		t.Fatal("lookup exposed mutable catalogue slice")
	}
}

func TestUnknownLookupAndOrderAreSafe(t *testing.T) {
	if _, ok := Lookup("E-NOT-RELEASED", LocaleEnglish); ok {
		t.Fatal("unknown code unexpectedly had guidance")
	}
	if got := Order("E-NOT-RELEASED"); got != PhaseSemantic {
		t.Fatalf("unknown phase=%q", got)
	}
	if got := Order(" E-PARSE-DATE "); got != PhaseSyntax {
		t.Fatalf("trimmed phase=%q", got)
	}
}
