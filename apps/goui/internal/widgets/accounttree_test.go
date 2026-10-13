// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package widgets

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"orangecount/internal/report"
)

func TestAccountTreeFilterKeepsAncestorsAndCoreTotals(t *testing.T) {
	roots := []report.AccountTreeNode{{
		Name: "Assets", FullName: "Assets",
		SubtreeAmounts: map[string]report.PresentedDecimal{"USD": {Display: "125.00", Exact: "125"}},
		Children: []report.AccountTreeNode{
			{
				Name: "Bank", FullName: "Assets:Bank",
				SubtreeAmounts: map[string]report.PresentedDecimal{"USD": {Display: "100.00", Exact: "100"}},
				Children:       []report.AccountTreeNode{{Name: "Checking", FullName: "Assets:Bank:Checking"}},
			},
			{Name: "Cash", FullName: "Assets:Cash"},
		},
	}, {Name: "Expenses", FullName: "Expenses"}}

	tree := NewAccountTree(roots, " BANK:checking ")
	if got, want := tree.RootKeys(), []string{"Assets"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("filtered roots = %v, want %v", got, want)
	}
	if got, want := tree.Children("Assets"), []string{"Assets:Bank"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("visible Assets children = %v, want %v", got, want)
	}
	if got, want := tree.Children("Assets:Bank"), []string{"Assets:Bank:Checking"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("visible Bank children = %v, want %v", got, want)
	}
	assets, ok := tree.Node("Assets")
	if !ok || assets.SubtreeAmounts["USD"].Exact != "125" {
		t.Fatalf("filter removed or recalculated the core parent total: %+v, found=%v", assets, ok)
	}
	if _, ok := tree.Node("Assets:Cash"); ok {
		t.Fatal("non-matching sibling remained in the visible tree")
	}
	if got, want := tree.Currencies(), []string{"USD"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("currency columns = %v, want unfiltered currencies %v", got, want)
	}
}

func TestAccountTreeNoMatchAndExpansionUseStableFullPaths(t *testing.T) {
	roots := []report.AccountTreeNode{{
		Name: "Assets", FullName: "Assets",
		Children: []report.AccountTreeNode{{
			Name: "Bank", FullName: "Assets:Bank",
			Children: []report.AccountTreeNode{{Name: "Checking", FullName: "Assets:Bank:Checking"}},
		}},
	}}
	tree := NewAccountTree(roots, "no match")
	if len(tree.RootKeys()) != 0 {
		t.Fatalf("no-match query returned roots %v", tree.RootKeys())
	}
	state := ui.OutlineState[string]{}
	tree.SetExpanded(&state, true)
	for _, key := range []string{"Assets", "Assets:Bank"} {
		if !state.Open.Has(key) {
			t.Errorf("expand all omitted full-path branch %q", key)
		}
	}
	if state.Open.Len() != 2 {
		t.Fatalf("expanded branch count = %d, want 2", state.Open.Len())
	}
	tree.SetExpanded(&state, false)
	if state.Open.Len() != 0 {
		t.Fatalf("collapse all kept %d open branches", state.Open.Len())
	}
}

func TestAmountCellKeepsExactValueAccessibleWithoutChangingDisplay(t *testing.T) {
	value := report.PresentedDecimal{Display: "0.166667", Exact: "1/6", Approximate: true}
	tester := ui.NewTester(func(c *ui.Context) {
		AmountCell(c, value, "USD")
	}, 260, 80)
	if got := AmountText(value); got != "≈0.166667" {
		t.Fatalf("rounded amount display = %q, want ≈0.166667", got)
	}
	if _, ok := tester.Find("1/6 USD"); !ok {
		t.Fatalf("exact amount was not exposed as the accessible name: %q", tester.Texts())
	}
	if err := tester.RightClick("1/6 USD"); err != nil {
		t.Fatal(err)
	}
	if err := tester.ChooseMenuItem("复制精确金额"); err != nil {
		t.Fatalf("exact amount context action missing: %v; menu=%v", err, tester.Menu())
	}
	if got := tester.Clipboard(); got != "1/6 USD" {
		t.Fatalf("copied amount = %q, want exact 1/6 USD", got)
	}
	if got := ExactAmountText(value, "USD"); got != "1/6 USD" {
		t.Fatalf("exact amount text = %q, want 1/6 USD", got)
	}
}
