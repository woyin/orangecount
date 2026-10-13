// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package query

import (
	"fmt"

	"orangecount/internal/ledger"
)

// CompareCells orders non-NULL result cells with the same typed ordering used
// by query ORDER BY. NULL sorts before every non-NULL value. Cells whose type
// has no query ordering return an error rather than inventing a display order.
func CompareCells(left, right Cell) (int, error) {
	if left.Type == "null" || right.Type == "null" {
		if left.Type == right.Type {
			return 0, nil
		}
		if left.Type == "null" {
			return -1, nil
		}
		return 1, nil
	}
	if left.Type != right.Type {
		return 0, fmt.Errorf("cannot compare %q and %q cells", left.Type, right.Type)
	}
	leftValue, err := comparableCellValue(left)
	if err != nil {
		return 0, err
	}
	rightValue, err := comparableCellValue(right)
	if err != nil {
		return 0, err
	}
	return compareTypedValues(leftValue, rightValue)
}

func comparableCellValue(cell Cell) (any, error) {
	invalid := func() (any, error) { return nil, fmt.Errorf("invalid %q cell value %T", cell.Type, cell.Value) }
	switch cell.Type {
	case "string", "date":
		if value, ok := cell.Value.(string); ok {
			return value, nil
		}
	case "integer":
		text, ok := cell.Value.(string)
		if !ok {
			return invalid()
		}
		value, err := ledger.ParseDecimal(text)
		if err != nil || !value.Rat().IsInt() {
			return invalid()
		}
		return value, nil
	case "decimal":
		value, ok := cell.Value.(PresentedDecimal)
		if !ok {
			return invalid()
		}
		number, err := ledger.ParseDecimal(value.Exact)
		if err != nil {
			return invalid()
		}
		return number, nil
	case "bool":
		if value, ok := cell.Value.(bool); ok {
			return value, nil
		}
	case "amount":
		value, ok := cell.Value.(PresentedAmount)
		if !ok {
			return invalid()
		}
		number, err := ledger.ParseDecimal(value.Number.Exact)
		if err != nil {
			return invalid()
		}
		return amountValue{Number: number, Currency: value.Currency}, nil
	case "position":
		value, ok := cell.Value.(PresentedPosition)
		if !ok {
			return invalid()
		}
		return comparablePosition(value)
	case "inventory":
		values, ok := cell.Value.([]PresentedPosition)
		if !ok {
			return invalid()
		}
		positions := make(inventoryValue, 0, len(values))
		for _, value := range values {
			position, err := comparablePosition(value)
			if err != nil {
				return nil, err
			}
			positions = append(positions, position)
		}
		return positions, nil
	}
	return invalid()
}

func comparablePosition(value PresentedPosition) (positionValue, error) {
	units, err := comparableAmount(value.Units)
	if err != nil {
		return positionValue{}, err
	}
	position := positionValue{Units: units}
	if value.Cost != nil {
		number, err := ledger.ParseDecimal(value.Cost.Number.Exact)
		if err != nil {
			return positionValue{}, fmt.Errorf("invalid position cost")
		}
		position.Cost = &costValue{Number: number, Currency: value.Cost.Currency, Date: value.Cost.Date, Label: value.Cost.Label}
	}
	return position, nil
}

func comparableAmount(value PresentedAmount) (amountValue, error) {
	number, err := ledger.ParseDecimal(value.Number.Exact)
	if err != nil {
		return amountValue{}, fmt.Errorf("invalid amount")
	}
	return amountValue{Number: number, Currency: value.Currency}, nil
}
