// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package snapshot

import (
	"orangecount/internal/ledger"
	"orangecount/internal/source"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func buildV3Fixture(t *testing.T, name string) BuildResult {
	t.Helper()
	return BuildWithOptions("../../testdata/fixtures/semantics/"+name+".bean", BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
}

func TestV3StrictAmbiguityDoesNotMutateInventoryOrInterpolate(t *testing.T) {
	for _, name := range []string{"strict-explicit", "strict-default"} {
		result := buildV3Fixture(t, name)
		if result.Snapshot != nil {
			t.Fatalf("%s: ambiguous reduction published", name)
		}
		graph, err := source.LoadGraph("../../testdata/fixtures/semantics/" + name + ".bean")
		if err != nil {
			t.Fatal(err)
		}
		parsed, _ := ledger.ParseGraph(graph)
		e := ledger.Evaluate(graph, parsed, ledger.EvalOptions{SemanticsProfile: ledger.V3})
		if e.Accounts["Assets:Stock"].Balances["STK"].String() != "30" || e.Accounts["Assets:Cash"].Balances["USD"].String() != "-500" {
			t.Fatalf("%s: rejected transaction mutated state", name)
		}
		last := e.Entries[len(e.Entries)-1].Directive.(*ledger.Transaction)
		if last.Postings[len(last.Postings)-1].Units != nil {
			t.Fatal("failed booking fell through to interpolation")
		}
	}
}

func TestV3StrictSizeSelectsSizedLot(t *testing.T) {
	result := buildV3Fixture(t, "strict-size")
	if result.Snapshot == nil {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	if got := result.Snapshot.Evaluation().Accounts["Income:Gains"].Balances["USD"].String(); got != "-200" {
		t.Fatalf("gain=%s, want -200", got)
	}
}

func TestV3ToleranceMultiplier(t *testing.T) {
	if result := buildV3Fixture(t, "tolerance-1"); result.Snapshot == nil {
		t.Fatalf("multiplier 1 rejected: %+v", result.Diagnostics)
	}
	if result := buildV3Fixture(t, "tolerance-half"); result.Snapshot != nil {
		t.Fatal("multiplier 0.5 accepted a 0.01 residual")
	}
	path := filepath.Join(t.TempDir(), "main.bean")
	text := `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
2000-01-02 * "mixed precision"
  Assets:Cash 1.0 USD
  Equity:Opening -1.04 USD
2000-01-03 balance Assets:Cash 1.01 USD
`
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	result := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if result.Snapshot == nil {
		t.Fatalf("coarsest decimal or balance tolerance rejected: %+v", result.Diagnostics)
	}
}

func TestV3NoneMaintainsSignedLotsWithoutMatching(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	text := `2000-01-01 open Assets:Stock STK "NONE"
2000-01-01 open Assets:Cash USD
2000-01-01 open Income:Gains USD
2000-01-02 * "buy"
  Assets:Stock 1 STK {10 USD}
  Assets:Cash -10 USD
2000-01-03 * "sell"
  Assets:Stock -2 STK {10 USD} @ 30 USD
  Assets:Cash 60 USD
  Income:Gains
`
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	options := BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}}
	result := BuildWithOptions(path, options)
	if result.Snapshot == nil {
		t.Fatalf("NONE oversell rejected: %+v", result.Diagnostics)
	}
	lots := result.Snapshot.Evaluation().Accounts["Assets:Stock"].Positions
	if len(lots) != 2 || lots[1].Units.String() != "-2" || lots[1].Cost.Date.Raw != "2000-01-03" {
		t.Fatalf("NONE matched or lost signed lot: %+v", lots)
	}
	text = strings.Replace(text, "-2 STK {10 USD}", "-2 STK {}", 1)
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	if result := BuildWithOptions(path, options); result.Snapshot != nil {
		t.Fatal("NONE silently accepted unresolved cost")
	}
}

func TestV3SizedUnlabeledLotDoesNotConsumeLabeledInventory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	text := `2000-01-01 open Assets:Stock STK "STRICT_WITH_SIZE"
2000-01-01 open Assets:Cash USD
2000-01-02 * "buy"
  Assets:Stock 2 STK {10 USD, "a"}
  Assets:Stock 3 STK {10 USD, ""}
  Assets:Cash -50 USD
2000-01-03 * "sell only unlabeled"
  Assets:Stock -3 STK {}
  Assets:Cash 30 USD
`
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	result := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if result.Snapshot == nil {
		t.Fatalf("sized unlabeled lot was not resolved: %+v", result.Diagnostics)
	}
	lots := result.Snapshot.Evaluation().Accounts["Assets:Stock"].Positions
	if len(lots) != 1 || lots[0].Units.String() != "2" || lots[0].Cost.Label != "a" {
		t.Fatalf("wrong label consumed: %+v", lots)
	}
}

func TestV3ImplicitAcquisitionDatesResolveLIFO(t *testing.T) {
	result := buildV3Fixture(t, "lifo-implicit")
	if result.Snapshot == nil {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	e := result.Snapshot.Evaluation()
	if got := e.Accounts["Income:Gains"].Balances["USD"].String(); got != "-100" {
		t.Fatalf("gain=%s, want -100", got)
	}
	lots := e.Accounts["Assets:Stock"].Positions
	if len(lots) != 2 || lots[0].Cost.Date == nil || lots[0].Cost.Date.Raw != "2000-01-02" || lots[1].Cost.Date == nil || lots[1].Cost.Date.Raw != "2000-01-03" {
		t.Fatalf("missing acquisition dates: %+v", lots)
	}
	// Legacy is still a reproducible migration baseline.
	legacy := Build("../../testdata/fixtures/semantics/lifo-implicit.bean").Snapshot.Evaluation()
	if legacy.Accounts["Income:Gains"].Balances["USD"].String() != "-200" {
		t.Fatal("legacy behavior changed")
	}
}

func TestV3FullLotIdentityAggregatesOnlyIdenticalLots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	text := `2000-01-01 open Assets:Stock STK "FIFO"
2000-01-01 open Assets:Cash USD
2000-01-02 * "same lot"
  Assets:Stock 2 STK {10 USD, "a"}
  Assets:Stock 3 STK {10 USD, "a"}
  Assets:Stock 4 STK {10 USD, "b"}
  Assets:Cash -90 USD
2000-01-03 * "new date"
  Assets:Stock 1 STK {10 USD, "a"}
  Assets:Cash -10 USD
`
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	result := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if result.Snapshot == nil {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	lots := result.Snapshot.Evaluation().Accounts["Assets:Stock"].Positions
	if len(lots) != 3 || lots[0].Units.String() != "5" || lots[1].Cost.Label != "b" || lots[2].Cost.Date.Raw != "2000-01-03" {
		t.Fatalf("lot identities: %+v", lots)
	}
}

func TestV3GlobalBookingAndAccountOverride(t *testing.T) {
	for _, tc := range []struct{ name, gain, booking string }{
		{"global-lifo", "-100", "LIFO"},
		{"global-average", "-400/3", "AVERAGE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := buildV3Fixture(t, tc.name)
			if result.Snapshot == nil {
				t.Fatalf("diagnostics: %+v", result.Diagnostics)
			}
			e := result.Snapshot.Evaluation()
			if got := e.Accounts["Income:Gains"].Balances["USD"].String(); got != tc.gain {
				t.Fatalf("gain=%s, want %s", got, tc.gain)
			}
			if got := e.Accounts["Assets:Stock"].Booking; got != tc.booking {
				t.Fatalf("effective booking=%s", got)
			}
			if tc.booking == "AVERAGE" && len(e.ActiveExtensions) == 0 {
				t.Fatal("average extension was not declared")
			}
		})
	}
	path := filepath.Join(t.TempDir(), "main.bean")
	for _, tc := range []struct {
		global, account, want string
		valid                 bool
	}{
		{"LIFO", "FIFO", "FIFO", true},
		{"LIFO", "", "LIFO", true},
		{"", "", "STRICT", true},
		{"BAD", "FIFO", "", false},
		{"FIFO", "BAD", "", false},
	} {
		text := ""
		if tc.global != "" {
			text += "option \"booking_method\" \"" + tc.global + "\"\n"
		}
		text += "2000-01-01 open Assets:Stock STK"
		if tc.account != "" {
			text += " \"" + tc.account + "\""
		}
		text += "\n"
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		result := BuildWithOptions(path, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
		if (result.Snapshot != nil) != tc.valid {
			t.Fatalf("%+v: diagnostics=%+v", tc, result.Diagnostics)
		}
		if tc.valid && result.Snapshot.Evaluation().Accounts["Assets:Stock"].Booking != tc.want {
			t.Fatalf("effective booking: %+v", tc)
		}
	}
}
