// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package report_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"orangecount/internal/compat"
	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
)

// The expected inventory is an independent pinned Python result, rather than
// a second view of today's evaluator positions. Includes generated pad entries.
func TestHoldingsReplayFinalInventoryMatchesPinnedOracle(t *testing.T) {
	paths, err := filepath.Glob("../../testdata/golden/v3-full/*.golden.json")
	if err != nil || len(paths) < 46 {
		t.Fatal("pinned full inventory corpus missing")
	}
	valid := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var want compat.FullGolden
		if err := json.Unmarshal(data, &want); err != nil {
			t.Fatal(err)
		}
		if !want.Valid {
			continue
		}
		valid++
		t.Run(filepath.Base(path), func(t *testing.T) {
			fixture := filepath.Join("../../testdata/fixtures/v3-parity", want.Fixture)
			if _, err := os.Stat(fixture); err != nil {
				fixture = filepath.Join("../../testdata/fixtures/semantics", want.Fixture)
			}
			built := snapshot.BuildWithOptions(fixture, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
			if built.Snapshot == nil || !built.Snapshot.Valid() {
				t.Fatal("oracle-valid fixture must publish a snapshot")
			}
			e := built.Snapshot.Evaluation()
			for account := range e.Accounts {
				view, err := report.HoldingsView(e, account, "direct", "", "")
				if err != nil {
					t.Fatal(err)
				}
				have := compat.FullInventory{Currencies: map[string]string{}, Lots: []string{}}
				for _, group := range view.Groups {
					for _, lot := range group.Lots {
						if lot.CostCurrency == nil {
							have.Currencies[lot.Commodity] = lot.Quantity.Exact
							continue
						}
						encoded, err := json.Marshal(map[string]any{"number": lot.Quantity.Exact, "currency": lot.Commodity, "cost": map[string]any{"number": lot.CostNumber.Exact, "currency": *lot.CostCurrency, "date": lot.CostDate, "label": lot.CostLabel}})
						if err != nil {
							t.Fatal(err)
						}
						have.Lots = append(have.Lots, string(encoded))
					}
				}
				sort.Strings(have.Lots)
				expected, ok := want.State.Balances[account]
				if !ok {
					expected = compat.FullInventory{Currencies: map[string]string{}, Lots: []string{}}
				}
				if !reflect.DeepEqual(expected, have) {
					t.Fatalf("account %s replay differs from fixed external oracle\nwant %+v\nhave %+v", account, expected, have)
				}
			}
		})
	}
	if valid < 20 {
		t.Fatal("insufficient oracle-valid replay cases")
	}
}
