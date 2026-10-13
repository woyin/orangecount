// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package widgets

import (
	"fmt"
	"sort"

	"github.com/egoist/mygo/ui"
)

// TableColumn binds a stable native column ID to typed sorting and cell
// rendering. Compare must use the value's domain representation, not display
// text, whenever Column.Sortable is true.
type TableColumn[T any] struct {
	Column  ui.TableColumn
	Compare func(T, T) int
	Cell    func(*ui.Context, T)
}

// StableTable binds MyGo's virtualized table to unique row IDs and column IDs,
// and sorts a private row view through the typed comparators supplied by the
// caller. Selection, focus, and scroll anchors therefore follow row identity
// when filtering or sorting changes visible indexes.
type StableTable[T any] struct {
	state   *ui.ListState
	rows    []T
	key     func(T) string
	label   func(T) string
	columns []TableColumn[T]
	specs   []ui.TableColumn
}

// NewStableTable validates the table schema and row identities, then builds
// the visible row order requested by state.Sort. A state with sortable
// columns must provide a persistent SortOrder pointer.
func NewStableTable[T any](state *ui.ListState, rows []T, key func(T) string, label func(T) string, columns []TableColumn[T]) (*StableTable[T], error) {
	if state == nil {
		return nil, fmt.Errorf("stable table requires persistent ListState")
	}
	if key == nil {
		return nil, fmt.Errorf("stable table requires a row key")
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("stable table requires at least one column")
	}
	table := &StableTable[T]{
		state:   state,
		rows:    append([]T(nil), rows...),
		key:     key,
		label:   label,
		columns: append([]TableColumn[T](nil), columns...),
		specs:   make([]ui.TableColumn, len(columns)),
	}
	columnIDs := make(map[string]struct{}, len(columns))
	sortable := false
	for i, column := range table.columns {
		id := column.Column.ID
		if id == "" {
			return nil, fmt.Errorf("table column %d has no stable ID", i)
		}
		if _, exists := columnIDs[id]; exists {
			return nil, fmt.Errorf("duplicate table column ID %q", id)
		}
		columnIDs[id] = struct{}{}
		if column.Cell == nil {
			return nil, fmt.Errorf("table column %q has no cell renderer", id)
		}
		if column.Column.Sortable {
			sortable = true
			if column.Compare == nil {
				return nil, fmt.Errorf("sortable table column %q has no typed comparator", id)
			}
		}
		table.specs[i] = column.Column
	}
	if sortable && state.Sort == nil {
		return nil, fmt.Errorf("sortable table requires persistent SortOrder")
	}
	keys := make(map[string]struct{}, len(rows))
	for i, row := range table.rows {
		id := key(row)
		if id == "" {
			return nil, fmt.Errorf("table row %d has an empty stable ID", i)
		}
		if _, exists := keys[id]; exists {
			return nil, fmt.Errorf("duplicate table row ID %q", id)
		}
		keys[id] = struct{}{}
	}
	if state.Sort != nil && state.Sort.Column != "" {
		columnIndex := -1
		for i := range table.columns {
			if table.columns[i].Column.ID == state.Sort.Column {
				columnIndex = i
				break
			}
		}
		if columnIndex < 0 || !table.columns[columnIndex].Column.Sortable {
			return nil, fmt.Errorf("sort column %q is not present and sortable", state.Sort.Column)
		}
		column := table.columns[columnIndex]
		sort.SliceStable(table.rows, func(i, j int) bool {
			comparison := column.Compare(table.rows[i], table.rows[j])
			if comparison != 0 {
				if state.Sort.Descending {
					return comparison > 0
				}
				return comparison < 0
			}
			return key(table.rows[i]) < key(table.rows[j])
		})
	}
	state.Key = func(row int) any { return key(table.rows[row]) }
	state.Label = func(row int) string {
		if label != nil {
			return label(table.rows[row])
		}
		return key(table.rows[row])
	}
	return table, nil
}

// Rows returns a copy of the row order presented to the UI.
func (table *StableTable[T]) Rows() []T { return append([]T(nil), table.rows...) }

// At returns the row at a visible table index.
func (table *StableTable[T]) At(index int) (T, bool) {
	if index < 0 || index >= len(table.rows) {
		var zero T
		return zero, false
	}
	return table.rows[index], true
}

// IndexOf returns the current visible index for a stable row ID.
func (table *StableTable[T]) IndexOf(id string) int {
	for index, row := range table.rows {
		if table.key(row) == id {
			return index
		}
	}
	return -1
}

// IDAt returns the stable ID at a visible table index.
func (table *StableTable[T]) IDAt(index int) string {
	row, ok := table.At(index)
	if !ok {
		return ""
	}
	return table.key(row)
}

// Render builds the native virtualized table using the validated schema.
func (table *StableTable[T]) Render(c *ui.Context) *ui.Element {
	return ui.Table(c, table.state, table.specs, len(table.rows), func(row, col int) {
		table.columns[col].Cell(c, table.rows[row])
	})
}
