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

func Zero() Decimal { return Decimal{rat: new(big.Rat)} }

func NewDecimal(r *big.Rat) Decimal {
	if r == nil {
		return Zero()
	}
	return Decimal{rat: new(big.Rat).Set(r)}
}

func DecimalFromNumber(number Number) Decimal { return NewDecimal(number.Rat) }

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

func (d Decimal) Rat() *big.Rat {
	if d.rat == nil {
		return new(big.Rat)
	}
	return new(big.Rat).Set(d.rat)
}

func (d Decimal) IsZero() bool { return d.rat == nil || d.rat.Sign() == 0 }

func (d Decimal) Sign() int {
	if d.rat == nil {
		return 0
	}
	return d.rat.Sign()
}

func (d Decimal) Add(other Decimal) Decimal {
	return Decimal{rat: new(big.Rat).Add(d.Rat(), other.Rat())}
}

func (d Decimal) Sub(other Decimal) Decimal {
	return Decimal{rat: new(big.Rat).Sub(d.Rat(), other.Rat())}
}

func (d Decimal) Mul(other Decimal) Decimal {
	return Decimal{rat: new(big.Rat).Mul(d.Rat(), other.Rat())}
}

func (d Decimal) Quo(other Decimal) Decimal {
	return Decimal{rat: new(big.Rat).Quo(d.Rat(), other.Rat())}
}

func (d Decimal) Neg() Decimal { return Decimal{rat: new(big.Rat).Neg(d.Rat())} }

func (d Decimal) Cmp(other Decimal) int { return d.Rat().Cmp(other.Rat()) }

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
