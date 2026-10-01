// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

type aggregateSelection struct {
	value any
	row   Row
}

// The source row belongs to the selected value: equal financial values may
// have different Decimal spellings. Ties retain the first representation.
// FIRST skips leading NULL; LAST includes a trailing NULL; extrema skip NULL.
func selectAggregateValue(name string, arg typedExpr, rows []Row) (aggregateSelection, error) {
	var selected aggregateSelection
	for _, row := range rows {
		value, err := arg.eval(row, []Row{row})
		if err != nil {
			return aggregateSelection{}, err
		}
		if name == "last" {
			selected = aggregateSelection{value, row}
			continue
		}
		if value == nil {
			continue
		}
		if selected.value == nil {
			selected = aggregateSelection{value, row}
			if name == "first" {
				return selected, nil
			}
			continue
		}
		comparison, err := compareTypedValues(value, selected.value)
		if err != nil {
			return aggregateSelection{}, err
		}
		if name == "min" && comparison < 0 || name == "max" && comparison > 0 {
			selected = aggregateSelection{value, row}
		}
	}
	return selected, nil
}
