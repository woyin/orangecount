// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package compat

import (
	"encoding/json"
	"orangecount/internal/ledger"
	"orangecount/internal/source"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFullComparisonRetainsDateLabelFlagAndMultiplicity(t *testing.T) {
	text := `2000-01-01 open Assets:Stock STK "FIFO"
2000-01-01 open Assets:Cash USD
2000-01-02 * "buy"
  ! Assets:Stock 1 STK {10 USD, "tag"}
    origin: "synthetic"
  Assets:Cash -10 USD
`
	file, bag := ledger.ParseText("synthetic.bean", []byte(text))
	if bag.HasErrors() {
		t.Fatal(bag.All())
	}
	e := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{SemanticsProfile: ledger.V3})
	before := NormalizeFull(*e)
	joined := strings.Join(before.Entries, "\n")
	for _, value := range []string{`"date":"2000-01-02"`, `"label":"tag"`, `"flag":"!"`, `"origin"`, `"synthetic"`} {
		if !strings.Contains(joined, value) {
			t.Fatalf("full normalized state omits %s", value)
		}
	}
	e.Entries = append(e.Entries, e.Entries[len(e.Entries)-1])
	after := NormalizeFull(*e)
	if len(FullDifferences(before, after)) == 0 {
		t.Fatal("duplicate transaction lost in comparison")
	}
}

func TestPinnedFullGoldenCorpus(t *testing.T) {
	paths, err := filepath.Glob("../../testdata/golden/v3-full/*.golden.json")
	if err != nil || len(paths) < 27 {
		t.Fatalf("full corpus missing: %v", err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var want FullGolden
			if err := json.Unmarshal(data, &want); err != nil {
				t.Fatal(err)
			}
			if want.OracleEngine != "beancount-python-3.2.3" {
				t.Fatal("unpinned oracle")
			}
			fixture := filepath.Join("../../testdata/fixtures/v3-parity", want.Fixture)
			if _, err := os.Stat(fixture); err != nil {
				fixture = filepath.Join("../../testdata/fixtures/semantics", want.Fixture)
			}
			have, err := BuildFullSide(fixture)
			if err != nil {
				t.Fatal(err)
			}
			if differences := FullGoldenDifferences(want, have); len(differences) > 0 {
				t.Errorf("dimensions=%v; diagnostics want=%v have=%v", differences, want.Errors, have.Errors)
				if len(want.State.Entries) != len(have.State.Entries) {
					t.Errorf("entry count want=%d have=%d", len(want.State.Entries), len(have.State.Entries))
				}
				for i := range want.State.Entries {
					if i < len(have.State.Entries) && want.State.Entries[i] != have.State.Entries[i] {
						t.Errorf("first entry difference\nwant %s\nhave %s", want.State.Entries[i], have.State.Entries[i])
						break
					}
				}
			}
		})
	}
}

func TestFullPriceIsExactAndTotalSignUsesAbsoluteUnits(t *testing.T) {
	amount, _ := ledger.ParseDecimal("-3")
	price, _ := ledger.ParseDecimal("2")
	p := ledger.Posting{Units: &ledger.Amount{Number: ledger.Number{Raw: amount.String(), Rat: amount.Rat()}, Currency: "STK"}, Price: &ledger.PriceSpec{Total: true, Amount: ledger.Amount{Number: ledger.Number{Raw: price.String(), Rat: price.Rat()}, Currency: "USD"}}}
	if got := fullPrice(p); got == nil || got.Number != "2/3" {
		t.Fatalf("price was signed or rounded: %+v", got)
	}
}
