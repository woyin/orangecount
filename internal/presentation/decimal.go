// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

// Package presentation formats exact amounts without changing their values.
package presentation

import (
	"strings"

	"orangecount/internal/ledger"
)

// Decimal retains the canonical value alongside its readable representation.
type Decimal struct {
	Display     string `json:"display"`
	Exact       string `json:"exact"`
	Approximate bool   `json:"approximate"`
}

// FormatDecimal shares the established report policy: terminating values are
// unchanged; only a non-terminating display is rounded to six fractional digits.
func FormatDecimal(value ledger.Decimal) Decimal {
	exact := value.String()
	result := Decimal{Display: exact, Exact: exact}
	if strings.ContainsRune(exact, '/') {
		result.Display = value.Rat().FloatString(6)
		result.Approximate = true
	}
	return result
}
