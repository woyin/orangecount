// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"math/big"
	"regexp"
	"strings"
	"time"

	"orangecount/internal/ledger"
)

var integerCastSyntax = regexp.MustCompile(`^[+-]?[0-9]+(?:_[0-9]+)*$`)

func compileDatePart(node functionExpr, arg typedExpr) (typedExpr, error) {
	if arg.kind != "date" {
		return typedExpr{}, typedFailure("type-mismatch", "no typed signature for %s(%s)", node.name, arg.kind)
	}
	result := typedExpr{name: node.String(), kind: "integer", nullable: arg.nullable, grouped: arg.grouped}
	result.value = func(row Row, rows []Row) (any, error) {
		value, err := arg.eval(row, rows)
		if err != nil || value == nil {
			return nil, err
		}
		date, err := time.Parse("2006-01-02", value.(string))
		if err != nil {
			return nil, typedFailure("type-mismatch", "invalid date value for %s", node.name)
		}
		return datePartDecimal(ledger.Date{Year: date.Year(), Month: int(date.Month()), Day: date.Day()}, node.name), nil
	}
	return result, nil
}

func castInteger(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch original := unwrapMetadataNumber(value).(type) {
	case ledger.Decimal:
		rational := original.Rat()
		integer := new(big.Int).Quo(rational.Num(), rational.Denom())
		return ledger.NewDecimal(new(big.Rat).SetInt(integer)), nil
	case bool:
		if original {
			return ledger.NewDecimal(big.NewRat(1, 1)), nil
		}
		return ledger.Zero(), nil
	case string:
		text := normalizeNumericDigits(strings.TrimSpace(original))
		if !integerCastSyntax.MatchString(text) {
			return nil, nil
		}
		integer, ok := new(big.Int).SetString(strings.ReplaceAll(text, "_", ""), 10)
		if !ok {
			return nil, nil
		}
		return ledger.NewDecimal(new(big.Rat).SetInt(integer)), nil
	default:
		return nil, nil
	}
}

func (compiler typedCompiler) compileDateConstructor(node functionExpr) (typedExpr, error) {
	args := make([]typedExpr, len(node.args))
	result := typedExpr{name: node.String(), kind: "date", nullable: true}
	for i, expr := range node.args {
		compiled, err := compiler.compile(expr)
		if err != nil {
			return typedExpr{}, err
		}
		args[i] = compiled
		result.grouped = result.grouped || compiled.grouped
	}
	for _, arg := range args {
		if arg.kind != "integer" && arg.kind != "bool" {
			return typedExpr{}, typedFailure("type-mismatch", "date(year, month, day) requires integer or bool arguments")
		}
	}
	result.value = func(row Row, rows []Row) (any, error) {
		values := make([]int, len(args))
		for i, arg := range args {
			value, err := arg.eval(row, rows)
			if err != nil || value == nil {
				return nil, err
			}
			number := new(big.Int)
			if boolean, ok := value.(bool); ok {
				if boolean {
					number.SetInt64(1)
				}
			} else {
				number.Set(value.(ledger.Decimal).Rat().Num())
			}
			// The reference date constructor consumes C-sized signed ints.
			if !number.IsInt64() || number.Int64() < -2147483648 || number.Int64() > 2147483647 {
				return nil, typedFailure("unsupported-capability", "date argument exceeds the constructor integer range")
			}
			values[i] = int(number.Int64())
		}
		year, month, day := values[0], values[1], values[2]
		if year < 1 || year > 9999 || month < 1 || month > 12 || day < 1 || day > 31 {
			return nil, nil
		}
		date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if date.Day() != day {
			return nil, nil
		}
		return date.Format("2006-01-02"), nil
	}
	return result, nil
}
