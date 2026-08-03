// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package report

import (
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
)

// PresentedDecimal is the display-safe representation used by normal report
// views. Exact remains the canonical ledger value; Display is rounded only
// when the exact value has a non-terminating decimal expansion.
type PresentedDecimal struct {
	Display     string `json:"display"`
	Exact       string `json:"exact"`
	Approximate bool   `json:"approximate"`
}

// Present converts a machine-exact report result into a UI/API presentation
// result. Query results and CSV exports continue to use the original exact
// Decimal values; this function never mutates its input.
func Present(result query.Result) query.Result {
	presented := query.Result{Columns: append([]string(nil), result.Columns...), Rows: make([]query.Row, len(result.Rows))}
	for index, row := range result.Rows {
		copyRow := make(query.Row, len(row))
		for column, value := range row {
			copyRow[column] = presentValue(value)
		}
		presented.Rows[index] = copyRow
	}
	return presented
}

func presentValue(value any) any {
	switch typed := value.(type) {
	case ledger.Decimal:
		return FormatDecimal(typed)
	case []ledger.Decimal:
		values := make([]PresentedDecimal, len(typed))
		for index, decimal := range typed {
			values[index] = FormatDecimal(decimal)
		}
		return values
	default:
		return value
	}
}

// FormatDecimal keeps terminating decimals unchanged and rounds only
// non-terminating values to six fractional digits for human display.
func FormatDecimal(value ledger.Decimal) PresentedDecimal {
	exact := value.String()
	presented := PresentedDecimal{Display: exact, Exact: exact}
	if !isRationalString(exact) {
		return presented
	}
	presented.Display = value.Rat().FloatString(6)
	presented.Approximate = true
	return presented
}

func isRationalString(value string) bool {
	// Decimal.String emits a slash only for a non-terminating rational. Keep
	// this check local to the presentation layer so ledger arithmetic remains
	// independent of display policy.
	return strings.ContainsRune(value, '/')
}
