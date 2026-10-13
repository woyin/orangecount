// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package query

import "testing"

func TestCompareCellsUsesExactCoreOrdering(t *testing.T) {
	tests := []struct {
		name       string
		left       Cell
		right      Cell
		comparison int
	}{
		{name: "null first", left: Cell{Type: "null"}, right: Cell{Type: "integer", Value: "0"}, comparison: -1},
		{name: "arbitrary precision integer", left: Cell{Type: "integer", Value: "900719925474099312345678901234567890"}, right: Cell{Type: "integer", Value: "900719925474099312345678901234567891"}, comparison: -1},
		{name: "exact fraction", left: Cell{Type: "decimal", Value: PresentedDecimal{Exact: "1/3"}}, right: Cell{Type: "decimal", Value: PresentedDecimal{Exact: "0.5"}}, comparison: -1},
		{name: "amount currency then value", left: Cell{Type: "amount", Value: PresentedAmount{Number: PresentedDecimal{Exact: "2"}, Currency: "USD"}}, right: Cell{Type: "amount", Value: PresentedAmount{Number: PresentedDecimal{Exact: "10"}, Currency: "USD"}}, comparison: -1},
		{name: "position exact units", left: Cell{Type: "position", Value: PresentedPosition{Units: PresentedAmount{Number: PresentedDecimal{Exact: "2"}, Currency: "USD"}}}, right: Cell{Type: "position", Value: PresentedPosition{Units: PresentedAmount{Number: PresentedDecimal{Exact: "10"}, Currency: "USD"}}}, comparison: -1},
		{name: "inventory by core position order", left: Cell{Type: "inventory", Value: []PresentedPosition{{Units: PresentedAmount{Number: PresentedDecimal{Exact: "2"}, Currency: "USD"}}}}, right: Cell{Type: "inventory", Value: []PresentedPosition{{Units: PresentedAmount{Number: PresentedDecimal{Exact: "10"}, Currency: "USD"}}}}, comparison: -1},
		{name: "bool", left: Cell{Type: "bool", Value: false}, right: Cell{Type: "bool", Value: true}, comparison: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := CompareCells(test.left, test.right)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.comparison {
				t.Fatalf("CompareCells() = %d, want %d", got, test.comparison)
			}
		})
	}
}

func TestCompareCellsRejectsInventedCollectionOrdering(t *testing.T) {
	if _, err := CompareCells(Cell{Type: "list", Value: PresentedList{}}, Cell{Type: "list", Value: PresentedList{}}); err == nil {
		t.Fatal("collection values without core query ordering were accepted")
	}
}
