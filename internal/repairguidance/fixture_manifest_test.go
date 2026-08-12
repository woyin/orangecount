// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package repairguidance

import (
	"strings"
	"testing"

	"orangecount/internal/diagnostic"
)

// The manifest is deliberately small and source-free: it records the
// diagnostic families that the release gate must exercise without turning a
// user's ledger into a committed fixture. Parser/evaluator packages own the
// triggering ledgers; this package verifies that every released error has a
// corresponding guidance fixture slot and repair phase.
var guidanceFixtureManifest = map[string]struct {
	family string
	phase  string
}{
	"E-INCLUDE-CYCLE": {"source graph", PhaseSource}, "E-INCLUDE-READ": {"source graph", PhaseSource},
	"E-SOURCE-UTF8": {"encoding", PhaseSyntax},
	"E-PARSE-DATE":  {"parser", PhaseSyntax}, "E-PARSE-DIRECTIVE": {"parser", PhaseSyntax},
	"E-PARSE-EXPECTED": {"parser", PhaseSyntax}, "E-PARSE-TOKEN": {"parser", PhaseSyntax}, "E-PARSE-STRING": {"parser", PhaseSyntax},
	"E-EVAL-OPEN": {"lifecycle", PhaseSemantic}, "E-EVAL-REOPEN": {"lifecycle", PhaseSemantic},
	"E-EVAL-CLOSE": {"lifecycle", PhaseSemantic}, "E-EVAL-POSTING": {"lifecycle", PhaseSemantic}, "E-EVAL-CURRENCY": {"currency", PhaseSemantic},
	"E-EVAL-UNBALANCED": {"transaction", PhaseSemantic}, "E-EVAL-INFER": {"transaction", PhaseSemantic},
	"E-EVAL-BALANCE": {"assertion", PhaseSemantic}, "E-EVAL-PAD": {"assertion", PhaseSemantic},
	"E-EVAL-TOLERANCE": {"assertion", PhaseSemantic}, "E-EVAL-INVENTORY": {"inventory", PhaseSemantic},
	"E-EVAL-OPTION": {"configuration", PhaseSemantic},
}

func TestGuidanceFixtureManifestCoversEveryReleasedError(t *testing.T) {
	released := diagnostic.ReleasedErrorCodes()
	for _, code := range released {
		fixture, ok := guidanceFixtureManifest[code]
		if !ok {
			t.Fatalf("released code %s has no fixture slot", code)
		}
		if fixture.family == "" || fixture.phase != Order(code) {
			t.Fatalf("fixture %s=%+v does not match guidance phase %q", code, fixture, Order(code))
		}
	}
	if got, want := len(guidanceFixtureManifest), len(released); got != want {
		t.Fatalf("fixture count=%d released error count=%d", got, want)
	}
	for code := range guidanceFixtureManifest {
		found := false
		for _, releasedCode := range released {
			if releasedCode == code {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("fixture has unreleased code %s", code)
		}
		if _, ok := Lookup(code, LocaleEnglish); !ok {
			t.Fatalf("fixture has unknown guidance code %s", code)
		}
	}
}

func TestGuidanceExamplesRemainGeneric(t *testing.T) {
	for _, code := range Codes() {
		for _, locale := range []string{LocaleEnglish, LocaleChinese} {
			guide, ok := Lookup(code, locale)
			if !ok {
				t.Fatalf("missing %s/%s", code, locale)
			}
			for _, value := range []string{guide.What, guide.Why, guide.Example.Before, guide.Example.After, guide.Example.Note, guide.Revalidate} {
				if strings.Contains(value, "/Users/") || strings.Contains(value, "\\Users\\") || strings.Contains(value, "127.0.0.1") {
					t.Fatalf("%s/%s contains a private fixture value: %q", code, locale, value)
				}
			}
		}
	}
}
