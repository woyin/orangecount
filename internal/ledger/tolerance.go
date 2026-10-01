// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package ledger

import (
	"math/big"
	"strings"
)

// decimalQuantum retains source precision including trailing zeros. Integer
// literals do not infer a tolerance. Exact arithmetic never rounds residuals.
func decimalQuantum(number Number) Decimal {
	raw := strings.ReplaceAll(number.Raw, ",", "")
	dot := strings.IndexByte(raw, '.')
	if dot < 0 || dot == len(raw)-1 {
		return Zero()
	}
	scale := len(raw) - dot - 1
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	return NewDecimal(new(big.Rat).SetFrac(big.NewInt(1), denom))
}

// transactionTolerances uses source postings, before splitting lots or
// interpolation can remove decimal precision or introduce automatic legs.
func (e *evaluator) transactionTolerances(postings []Posting) map[string]Decimal {
	tolerances := make(map[string]Decimal)
	for cur, tol := range e.inferredToleranceDefault {
		tolerances[cur] = tol
	}
	costBounds := make(map[string]Decimal)
	maximum, _ := ParseDecimal("0.5")
	for _, p := range postings {
		if p.Units == nil || p.Units.Number.Raw == "" {
			continue
		}
		quantum := decimalQuantum(p.Units.Number)
		if quantum.IsZero() {
			continue
		}
		tol := quantum.Mul(e.toleranceMultiplier)
		cur := p.Units.Currency
		if previous, ok := tolerances[cur]; !ok || tol.Cmp(previous) > 0 {
			tolerances[cur] = tol
		}
		if !e.inferToleranceFromCost {
			continue
		}
		addBound := func(currency string, price Decimal) {
			bound := tol.Mul(price)
			if bound.Cmp(maximum) > 0 {
				bound = maximum
			}
			costBounds[currency] = costBounds[currency].Add(bound)
		}
		if p.Cost != nil {
			if cost, ok := normalizeCost(*p.Cost); ok {
				if p.Cost.Total {
					units := DecimalFromNumber(p.Units.Number)
					if units.Sign() < 0 {
						units = units.Neg()
					}
					if !units.IsZero() {
						cost.Number = cost.Number.Quo(units)
					}
				}
				addBound(cost.Currency, cost.Number)
			}
		}
		if p.Price != nil {
			price := DecimalFromNumber(p.Price.Amount.Number)
			if p.Price.Total {
				units := DecimalFromNumber(p.Units.Number)
				if units.Sign() < 0 {
					units = units.Neg()
				}
				if !units.IsZero() {
					price = price.Quo(units)
				}
			}
			addBound(p.Price.Amount.Currency, price)
		}
	}
	for cur, tol := range costBounds {
		if previous, ok := tolerances[cur]; !ok || tol.Cmp(previous) > 0 {
			tolerances[cur] = tol
		}
	}
	return tolerances
}

// Keep interpolation's representable decimal spelling separate from the exact
// inferred value and Raw. A spelling may not introduce quantization of money.
func inferredNumberText(value Decimal, currency string, postings []Posting) string {
	scale := -1
	for _, posting := range postings {
		if posting.Units == nil || posting.Units.Currency != currency || posting.Units.Number.Raw == "" {
			continue
		}
		text := strings.ReplaceAll(posting.Units.Number.Raw, ",", "")
		if at := strings.IndexByte(text, '.'); at >= 0 {
			precision := len(text) - at - 1
			if scale < 0 || precision < scale {
				scale = precision
			}
		}
	}
	if scale < 0 {
		return ""
	}
	text := value.Rat().FloatString(scale)
	if represented, err := ParseDecimal(text); err == nil && represented.Equal(value) {
		return text
	}
	return ""
}
