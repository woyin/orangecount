// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package repairguidance

import (
	"os"
	"path/filepath"
	"testing"

	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

// These are intentionally tiny, synthetic inputs. They exercise the same
// parser/evaluator/source seams used by the application without containing a
// real ledger or a ledger-specific amount that guidance could accidentally
// repeat.
func TestEveryGuidanceCodeHasARepresentativeTrigger(t *testing.T) {
	tests := []struct {
		code string
		text string
	}{
		{code: "E-SOURCE-UTF8"},
		{code: "E-PARSE-DATE", text: "2000-02-30 open Assets:Cash USD\n"},
		{code: "E-PARSE-DIRECTIVE", text: "2000-01-01 opne Assets:Cash USD\n"},
		{code: "E-PARSE-EXPECTED", text: "2000-01-01 open\n"},
		{code: "E-PARSE-TOKEN", text: "2000-01-01 * \"x\" ???\n"},
		{code: "E-PARSE-STRING", text: "2000-01-01 * \"x\n"},
		{code: "E-EVAL-OPEN", text: "2000-01-01 open Assets:Cash USD\n2000-01-02 open Assets:Cash USD\n"},
		{code: "E-EVAL-REOPEN", text: "2000-01-01 open Assets:Cash USD\n2000-01-02 close Assets:Cash\n2000-01-03 open Assets:Cash USD\n"},
		{code: "E-EVAL-CLOSE", text: "2000-01-01 close Assets:Cash\n"},
		{code: "E-EVAL-POSTING", text: "2000-01-02 * \"too early\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n2000-01-03 open Assets:Cash USD\n2000-01-03 open Equity:Opening USD\n"},
		{code: "E-EVAL-CURRENCY", text: "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening EUR\n2000-01-02 * \"wrong currency\"\n  Assets:Cash 1 EUR\n  Equity:Opening -1 EUR\n"},
		{code: "E-EVAL-UNBALANCED", text: "2000-01-01 open Assets:Cash USD\n2000-01-02 * \"missing leg\"\n  Assets:Cash 1 USD\n"},
		{code: "E-EVAL-INFER", text: "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"ambiguous\"\n  Assets:Cash\n  Equity:Opening\n"},
		{code: "E-EVAL-BALANCE", text: "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n2000-01-03 balance Assets:Cash 2 USD\n"},
		{code: "E-EVAL-PAD", text: "2000-01-01 open Assets:Cash USD\n2000-01-02 pad Assets:Cash Equity:Missing\n2000-01-03 balance Assets:Cash 1 USD\n"},
		{code: "E-EVAL-TOLERANCE", text: "2000-01-01 open Assets:Cash USD\n2000-01-02 balance Assets:Cash 0 USD ~ -0.01 USD\n"},
		{code: "E-EVAL-OPTION", text: "option \"tolerance\" \"not-a-number\"\n"},
		{code: "E-EVAL-INVENTORY", text: "2000-01-01 open Assets:Shares SH\n2000-01-01 open Assets:Cash USD\n2000-01-02 * \"buy\"\n  Assets:Shares 1 SH {10 USD}\n  Assets:Cash -10 USD\n2000-01-03 * \"sell too much\"\n  Assets:Shares -2 SH {10 USD}\n  Assets:Cash 20 USD\n"},
	}

	for _, fixture := range tests {
		t.Run(fixture.code, func(t *testing.T) {
			var diagnostics []diagnostic.Diagnostic
			if fixture.code == "E-SOURCE-UTF8" {
				_, bag := ledger.ParseText("fixture.bean", []byte{0xff, '\n'})
				diagnostics = bag.All()
			} else {
				file, bag := ledger.ParseText("fixture.bean", []byte(fixture.text))
				diagnostics = append(diagnostics, bag.All()...)
				if !bag.HasErrors() {
					evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
					diagnostics = append(diagnostics, evaluation.Diagnostics...)
				}
			}
			if !hasDiagnosticCode(diagnostics, fixture.code) {
				t.Fatalf("fixture did not trigger %s: %+v", fixture.code, diagnostics)
			}
			guide, ok := Lookup(fixture.code, LocaleEnglish)
			if !ok || guide.Phase != Order(fixture.code) || guide.Topic != "diagnostics/"+fixture.code {
				t.Fatalf("fixture guidance=%+v ok=%v", guide, ok)
			}
		})
	}
}

func TestSourceGraphGuidanceFixtures(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte("include \"child.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if graph, err := source.LoadGraph(entry); err != nil || !hasGraphCode(graph, "E-INCLUDE-READ") {
		t.Fatalf("missing include graph=%+v err=%v", graph, err)
	}
	cycleA := filepath.Join(dir, "cycle-a.bean")
	cycleB := filepath.Join(dir, "cycle-b.bean")
	if err := os.WriteFile(cycleA, []byte("include \"cycle-b.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cycleB, []byte("include \"cycle-a.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	graph, err := source.LoadGraph(cycleA)
	if err != nil || !hasGraphCode(graph, "E-INCLUDE-CYCLE") {
		t.Fatalf("cycle graph=%+v err=%v", graph, err)
	}
}

func hasDiagnosticCode(values []diagnostic.Diagnostic, want string) bool {
	for _, value := range values {
		if value.Code == want {
			return true
		}
	}
	return false
}

func hasGraphCode(graph *source.Graph, want string) bool {
	if graph == nil {
		return false
	}
	for _, issue := range graph.Diagnostics {
		if issue.Code == want {
			return true
		}
	}
	return false
}
