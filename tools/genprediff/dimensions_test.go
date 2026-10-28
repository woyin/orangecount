// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDimensionsAreActuallyCompared proves the four generative dimensions are
// real comparisons, not decoration: each one must report a difference when the
// oracle record is perturbed. A test that only asserts "no differences" would
// pass just as well with the dimensions deleted.
func TestDimensionsAreActuallyCompared(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cases := []string{
		"424242-plain.bean",
		"424242-costs.bean",
		"424242-multicurrency.bean",
	}
	var selected string
	for _, name := range cases {
		path := filepath.Join(root, "testdata", "fixtures", "gen", name)
		if _, statErr := os.Stat(path); statErr == nil {
			selected = path
			break
		}
	}
	if selected == "" {
		t.Skip("no promoted corpus; run go run ./tools/genprediff -promote-all")
	}

	have, err := buildGoDimensions(selected)
	if err != nil {
		t.Fatal(err)
	}
	base := oracleRecord{
		Fixture:           filepath.Base(selected),
		Valid:             true,
		OrderedEntries:    append([]string(nil), have.OrderedEntries...),
		PrefixInventories: append([]map[string]map[string]string(nil), have.PrefixInventories...),
		Prices:            append([]string(nil), have.Prices...),
		PeriodIncome:      map[string]string{},
	}
	for key, value := range have.PeriodIncome {
		base.PeriodIncome[key] = value
	}
	if differences := dimensionDifferences(base, have); len(differences) != 0 {
		t.Fatalf("baseline must match: %v", differences)
	}

	// 1) ordered entries: swap the first two entries (the multiset is unchanged,
	// so only an order-sensitive comparison can see this).
	if len(have.OrderedEntries) >= 2 {
		swapped := append([]string(nil), base.OrderedEntries...)
		swapped[0], swapped[1] = swapped[1], swapped[0]
		perturbed := base
		perturbed.OrderedEntries = swapped
		if differences := dimensionDifferences(perturbed, have); !containsPrefix(differences, "ordered-entries") {
			t.Fatalf("reordering must be detected, got %v", differences)
		}
	}

	// 2) prefix inventories: perturb one transaction's inventory.
	if len(have.PrefixInventories) != 0 {
		perturbed := base
		perturbed.PrefixInventories = deepCopyInventories(base.PrefixInventories)
		perturbed.PrefixInventories[0]["__probe__"] = map[string]string{"USD": "1"}
		if differences := dimensionDifferences(perturbed, have); !containsPrefix(differences, "prefix-inventory") {
			t.Fatalf("prefix inventory change must be detected, got %v", differences)
		}
	}

	// 3) prices.
	perturbed := base
	perturbed.Prices = append(append([]string(nil), base.Prices...), `{"date":"2020-01-02","base":"AAA","quote":"USD","rate":"1"}`)
	if differences := dimensionDifferences(perturbed, have); !containsPrefix(differences, "prices") {
		t.Fatalf("price change must be detected, got %v", differences)
	}

	// 4) period income.
	perturbed = base
	perturbed.PeriodIncome = map[string]string{}
	for key, value := range base.PeriodIncome {
		perturbed.PeriodIncome[key] = value
	}
	perturbed.PeriodIncome["1999-01|USD"] = "1"
	if differences := dimensionDifferences(perturbed, have); !containsPrefix(differences, "period-income") {
		t.Fatalf("period income change must be detected, got %v", differences)
	}
}

// TestOrderedEntriesFollowSourceOrder checks the Go side really orders entries
// by source line rather than returning the sorted multiset it was handed.
func TestOrderedEntriesFollowSourceOrder(t *testing.T) {
	entries := []string{
		`{"source":{"file":"a.bean","line":9},"kind":"open","account":"Assets:B"}`,
		`{"source":{"file":"a.bean","line":3},"kind":"open","account":"Assets:A"}`,
		`{"source":{"file":"a.bean","line":5},"kind":"balance","account":"Assets:A"}`,
	}
	ordered := orderedEntries(entries)
	want := []string{entries[1], entries[2], entries[0]}
	if len(ordered) != len(want) {
		t.Fatalf("length %d != %d", len(ordered), len(want))
	}
	for index := range want {
		if ordered[index] != want[index] {
			t.Fatalf("entry %d = %q, want %q", index, ordered[index], want[index])
		}
	}
}

func containsPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func deepCopyInventories(source []map[string]map[string]string) []map[string]map[string]string {
	copied := make([]map[string]map[string]string, len(source))
	for index, inventory := range source {
		inner := make(map[string]map[string]string, len(inventory))
		for account, currencies := range inventory {
			row := make(map[string]string, len(currencies))
			for currency, amount := range currencies {
				row[currency] = amount
			}
			inner[account] = row
		}
		copied[index] = inner
	}
	return copied
}
