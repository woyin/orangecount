// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

// Decimal is an immutable exact rational used for ledger arithmetic. Beancount
// numbers are decimal, but rational storage avoids floating-point rounding at
// every intermediate step; String preserves a terminating decimal whenever
// one exists and otherwise emits a deterministic fraction.
type Decimal struct{ rat *big.Rat }

// Zero returns the additive identity, safe to use without construction.
func Zero() Decimal { return Decimal{rat: new(big.Rat)} }

// NewDecimal wraps a big.Rat, copying it; nil yields Zero.
func NewDecimal(r *big.Rat) Decimal {
	if r == nil {
		return Zero()
	}
	return Decimal{rat: new(big.Rat).Set(r)}
}

// DecimalFromNumber converts a parsed Number literal into a Decimal.
func DecimalFromNumber(number Number) Decimal { return NewDecimal(number.Rat) }

// ParseDecimal parses source text (commas allowed) into an exact Decimal.
func ParseDecimal(raw string) (Decimal, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ",", ""))
	if raw == "" {
		return Zero(), fmt.Errorf("empty decimal")
	}
	rat, ok := new(big.Rat).SetString(raw)
	if !ok {
		return Zero(), fmt.Errorf("invalid decimal %q", raw)
	}
	return NewDecimal(rat), nil
}

// Rat returns a defensive copy of the underlying rational.
func (d Decimal) Rat() *big.Rat {
	if d.rat == nil {
		return new(big.Rat)
	}
	return new(big.Rat).Set(d.rat)
}

// IsZero reports whether the value equals zero.
func (d Decimal) IsZero() bool { return d.rat == nil || d.rat.Sign() == 0 }

// Sign reports -1, 0, or +1 for negative, zero, and positive values.
func (d Decimal) Sign() int {
	if d.rat == nil {
		return 0
	}
	return d.rat.Sign()
}

// Add returns the exact sum; receivers are never mutated.
func (d Decimal) Add(other Decimal) Decimal {
	return Decimal{rat: new(big.Rat).Add(d.Rat(), other.Rat())}
}

// Sub returns the exact difference.
func (d Decimal) Sub(other Decimal) Decimal {
	return Decimal{rat: new(big.Rat).Sub(d.Rat(), other.Rat())}
}

// Mul returns the exact product.
func (d Decimal) Mul(other Decimal) Decimal {
	return Decimal{rat: new(big.Rat).Mul(d.Rat(), other.Rat())}
}

// Quo returns the exact quotient; division by zero follows big.Rat semantics.
func (d Decimal) Quo(other Decimal) Decimal {
	return Decimal{rat: new(big.Rat).Quo(d.Rat(), other.Rat())}
}

// Neg returns the additive inverse.
func (d Decimal) Neg() Decimal { return Decimal{rat: new(big.Rat).Neg(d.Rat())} }

// Cmp compares magnitudes: -1, 0, or +1.
func (d Decimal) Cmp(other Decimal) int { return d.Rat().Cmp(other.Rat()) }

// Equal reports exact equality.
func (d Decimal) Equal(other Decimal) bool { return d.Cmp(other) == 0 }

func (d Decimal) String() string {
	r := d.Rat()
	if r.Sign() == 0 {
		return "0"
	}
	// big.Rat.FloatString is deterministic. Choose enough places for the
	// decimal inputs accepted by the parser while falling back to a fraction
	// for non-terminating values rather than silently rounding.
	if decimalScale, ok := terminatingScale(r.Denom()); ok {
		return r.FloatString(decimalScale)
	}
	return r.RatString()
}

// MarshalJSON keeps exact decimal values lossless in API and report payloads.
// Numbers that are not terminating decimals are encoded as strings containing
// their canonical fraction instead of being rounded through a JSON float.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func terminatingScale(denom *big.Int) (int, bool) {
	if denom == nil || denom.Sign() == 0 {
		return 0, false
	}
	d := new(big.Int).Set(denom)
	two, five := big.NewInt(2), big.NewInt(5)
	scale := 0
	for new(big.Int).Mod(d, two).Sign() == 0 {
		d.Div(d, two)
		scale++
	}
	fiveScale := 0
	for new(big.Int).Mod(d, five).Sign() == 0 {
		d.Div(d, five)
		fiveScale++
	}
	if d.Cmp(big.NewInt(1)) != 0 {
		return 0, false
	}
	if fiveScale > scale {
		scale = fiveScale
	}
	return scale, true
}
