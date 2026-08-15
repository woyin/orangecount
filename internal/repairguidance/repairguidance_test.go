// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package repairguidance

import (
	"reflect"
	"strings"
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

func TestValidateCoverageContracts(t *testing.T) {
	released := diagnostic.ReleasedErrorCodes()
	if err := ValidateCoverage(released); err != nil {
		t.Fatalf("released codes must validate: %v", err)
	}
	if err := ValidateCoverage([]string{"not-a-code"}); err == nil || !strings.Contains(err.Error(), "invalid released") {
		t.Fatalf("invalid code err=%v", err)
	}
	if err := ValidateCoverage([]string{"E-NO-SUCH-CODE"}); err == nil || !strings.Contains(err.Error(), "missing repair guidance") {
		t.Fatalf("missing guidance err=%v", err)
	}
	// A catalogue entry whose code was never released is stale guidance.
	var pruned []string
	for _, code := range released {
		if code != "E-EVAL-OPEN" {
			pruned = append(pruned, code)
		}
	}
	if err := ValidateCoverage(pruned); err == nil || !strings.Contains(err.Error(), "no released diagnostic code") {
		t.Fatalf("stale guidance err=%v", err)
	}
}

func TestValidateGuideFieldContracts(t *testing.T) {
	guide, ok := catalogue["E-EVAL-OPEN"]
	if !ok {
		t.Fatal("fixture guide missing")
	}
	if err := validateGuide("E-EVAL-OPEN", LocaleEnglish, guide.English); err != nil {
		t.Fatalf("english guide err=%v", err)
	}
	broken := guide.English
	broken.What = " "
	if err := validateGuide("E-EVAL-OPEN", LocaleEnglish, broken); err == nil || !strings.Contains(err.Error(), "missing what") {
		t.Fatalf("blank field err=%v", err)
	}
	mismatched := guide.English
	mismatched.Code = "E-OTHER"
	if err := validateGuide("E-EVAL-OPEN", LocaleEnglish, mismatched); err == nil || !strings.Contains(err.Error(), "code mismatch") {
		t.Fatalf("code mismatch err=%v", err)
	}
	unstable := guide.English
	unstable.Topic = "wrong/topic"
	if err := validateGuide("E-EVAL-OPEN", LocaleEnglish, unstable); err == nil || !strings.Contains(err.Error(), "unstable topic") {
		t.Fatalf("unstable topic err=%v", err)
	}
	noSteps := guide.English
	noSteps.Inspect = nil
	if err := validateGuide("E-EVAL-OPEN", LocaleEnglish, noSteps); err == nil || !strings.Contains(err.Error(), "inspect or safe steps") {
		t.Fatalf("missing steps err=%v", err)
	}
}
