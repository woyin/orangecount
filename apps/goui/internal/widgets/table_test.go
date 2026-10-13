// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package widgets

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"orangecount/internal/report"
)

type tableFixture struct {
	id    string
	value int
}

func fixtureColumns() []TableColumn[tableFixture] {
	return []TableColumn[tableFixture]{
		{
			Column: ui.TableColumn{Title: "Value", ID: "value", Sortable: true},
			Compare: func(a, b tableFixture) int {
				switch {
				case a.value < b.value:
					return -1
				case a.value > b.value:
					return 1
				default:
					return 0
				}
			},
			Cell: func(c *ui.Context, row tableFixture) { ui.Text(c, fmt.Sprint(row.value)) },
		},
	}
}

func TestStableTableSortsTypedValuesAndKeepsStableIDs(t *testing.T) {
	rows := []tableFixture{{id: "ten", value: 10}, {id: "two", value: 2}, {id: "one", value: 1}}
	order := ui.SortOrder{Column: "value"}
	state := ui.ListState{Sort: &order}
	key := func(row tableFixture) string { return row.id }
	table, err := NewStableTable(&state, rows, key, nil, fixtureColumns())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := []string{table.IDAt(0), table.IDAt(1), table.IDAt(2)}, []string{"one", "two", "ten"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ascending IDs = %v, want %v", got, want)
	}
	if got := table.IndexOf("two"); got != 1 {
		t.Fatalf("stable row index = %d, want 1", got)
	}
	order.Descending = true
	table, err = NewStableTable(&state, rows, key, nil, fixtureColumns())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := []string{table.IDAt(0), table.IDAt(1), table.IDAt(2)}, []string{"ten", "two", "one"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("descending numeric IDs = %v, want %v", got, want)
	}
	if state.Key(1) != "two" {
		t.Fatalf("ListState key after sorting = %v, want stable row ID two", state.Key(1))
	}
}

func TestStableTableRejectsAmbiguousIdentityAndSchema(t *testing.T) {
	key := func(row tableFixture) string { return row.id }
	state := ui.ListState{}
	columns := []TableColumn[tableFixture]{
		{Column: ui.TableColumn{Title: "First", ID: "same"}, Cell: func(*ui.Context, tableFixture) {}},
		{Column: ui.TableColumn{Title: "Second", ID: "same"}, Cell: func(*ui.Context, tableFixture) {}},
	}
	if _, err := NewStableTable(&state, nil, key, nil, columns); err == nil {
		t.Fatal("duplicate column IDs were accepted")
	}
	columns = []TableColumn[tableFixture]{{Column: ui.TableColumn{Title: "Value", ID: "value"}, Cell: func(*ui.Context, tableFixture) {}}}
	if _, err := NewStableTable(&state, []tableFixture{{id: "same"}, {id: "same"}}, key, nil, columns); err == nil {
		t.Fatal("duplicate row IDs were accepted")
	}
	columns = []TableColumn[tableFixture]{
		{Column: ui.TableColumn{Title: "Value", ID: "value-primary"}, Cell: func(*ui.Context, tableFixture) {}},
		{Column: ui.TableColumn{Title: "Value", ID: "value-secondary"}, Cell: func(*ui.Context, tableFixture) {}},
	}
	if _, err := NewStableTable(&state, []tableFixture{{id: "one"}}, key, nil, columns); err != nil {
		t.Fatalf("repeated visible titles with stable distinct IDs were rejected: %v", err)
	}
}

func TestStableTableSortHeaderUsesPersistentTypedSort(t *testing.T) {
	rows := []tableFixture{{id: "ten", value: 10}, {id: "two", value: 2}}
	var sortOrder ui.SortOrder
	selected := -1
	state := ui.ListState{Sort: &sortOrder, Selected: &selected}
	key := func(row tableFixture) string { return row.id }
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(10).Children(func() {
			table, err := NewStableTable(&state, rows, key, nil, fixtureColumns())
			if err != nil {
				panic(err)
			}
			table.Render(c).Grow(1)
		})
	}, 500, 160)
	if err := tester.Click("10"); err != nil {
		t.Fatalf("select first row: %v; UI=%v", err, tester.Texts())
	}
	if selected != 0 {
		t.Fatalf("initial selected row = %d, want row ten", selected)
	}
	if got := state.Key(selected); got != "ten" {
		t.Fatalf("initial selected key = %v, want ten", got)
	}
	if err := tester.Click("Value"); err != nil {
		t.Fatal(err)
	}
	if sortOrder != (ui.SortOrder{Column: "value"}) {
		t.Fatalf("first header click sort = %+v", sortOrder)
	}
	tester.Frame()
	first, firstOK := tester.Find("2")
	second, secondOK := tester.Find("10")
	if !firstOK || !secondOK || first.Y >= second.Y {
		t.Fatalf("typed ascending order was not rendered: texts=%v first=%+v second=%+v", tester.Texts(), first, second)
	}
	if selected != 1 {
		t.Fatalf("selection did not follow row identity after sort: row=%d, want 1", selected)
	}
	if got := state.Key(selected); got != "ten" {
		t.Fatalf("selected key after sort = %v, want ten", got)
	}
}

func TestCompareAmountsUsesExactRationals(t *testing.T) {
	third := report.PresentedDecimal{Display: "0.333333", Exact: "1/3", Approximate: true}
	pointFour := report.PresentedDecimal{Display: "0.4", Exact: "0.4"}
	if got := CompareAmounts(third, pointFour); got >= 0 {
		t.Fatalf("exact 1/3 should sort before 0.4, got comparison %d", got)
	}
	if got := CompareAmounts(report.PresentedDecimal{Exact: "-2"}, report.PresentedDecimal{Exact: "-1"}); got >= 0 {
		t.Fatalf("exact -2 should sort before -1, got comparison %d", got)
	}
	if got := CompareOptionalAmounts(nil, &pointFour); got <= 0 {
		t.Fatalf("missing amount should sort after a numeric value, got comparison %d", got)
	}
}
