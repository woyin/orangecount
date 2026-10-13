// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package widgets

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"orangecount/internal/ledger"
	"orangecount/internal/report"
)

// AmountCell renders the readable value without discarding its exact ledger
// representation. Approximate values are marked visibly; the accessible name
// and tooltip retain the exact value and optional currency code. The function
// does not parse, round, or otherwise recalculate financial values.
func AmountCell(c *ui.Context, value report.PresentedDecimal, currency string) *ui.Element {
	display := AmountText(value)
	if currency != "" {
		display = currency + " " + display
	}

	exact := value.Exact
	if exact == "" {
		exact = value.Display
	}
	if exact == "" {
		exact = "金额不可用"
	}
	accessible := exact
	if currency != "" {
		accessible += " " + currency
	}
	cell := ui.Text(c, display).Label(accessible).Tooltip(accessible)
	cell.ContextMenu(func(menu *ui.Menu) {
		if menu.Item("复制精确金额").Chosen() {
			c.WriteClipboard(accessible)
		}
	})
	return cell
}

// AmountText returns the report's readable amount, adding an approximation
// marker when Display differs from the exact canonical value.
func AmountText(value report.PresentedDecimal) string {
	display := value.Display
	if display == "" {
		display = value.Exact
	}
	if display == "" {
		return "—"
	}
	if value.Approximate {
		return "≈" + display
	}
	return display
}

// ExactAmountText returns the canonical value with an optional currency code.
// It is used where a financial detail is intentionally shown without the
// rounded display representation.
func ExactAmountText(value report.PresentedDecimal, currency string) string {
	exact := value.Exact
	if exact == "" {
		exact = value.Display
	}
	if exact == "" {
		return "—"
	}
	if currency == "" {
		return exact
	}
	return strings.TrimSpace(exact + " " + currency)
}

// CompareAmounts compares canonical exact values without parsing display text
// through float64. Values without a numeric representation sort after valid
// amounts in ascending order; two unavailable values use their exact text as a
// deterministic tie-breaker.
func CompareAmounts(a, b report.PresentedDecimal) int {
	aText, bText := a.Exact, b.Exact
	if aText == "" {
		aText = a.Display
	}
	if bText == "" {
		bText = b.Display
	}
	aValue, aErr := ledger.ParseDecimal(aText)
	bValue, bErr := ledger.ParseDecimal(bText)
	switch {
	case aErr == nil && bErr == nil:
		return aValue.Cmp(bValue)
	case aErr == nil:
		return -1
	case bErr == nil:
		return 1
	default:
		return strings.Compare(aText, bText)
	}
}

// CompareOptionalAmounts orders a missing value after a present value in
// ascending order, then compares present values by their exact rationals.
func CompareOptionalAmounts(a, b *report.PresentedDecimal) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	default:
		return CompareAmounts(*a, *b)
	}
}
