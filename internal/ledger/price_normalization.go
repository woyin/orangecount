// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package ledger

import "math/big"

// roundedSignificant implements half-even significant-digit rounding for
// v3's booked @@ unit price and the typed query's booked-price weight. Balances,
// cost basis, AVERAGE and the source total quote remain exact.
func roundedSignificant(value Decimal, precision int) Decimal {
	if value.IsZero() {
		return Zero()
	}
	rat := value.Rat()
	sign := rat.Sign()
	rat.Abs(rat)
	n, d := rat.Num(), rat.Denom()
	exponent := len(n.String()) - len(d.String())
	power := func(n int) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil) }
	if exponent >= 0 {
		if n.Cmp(new(big.Int).Mul(d, power(exponent))) < 0 {
			exponent--
		}
	} else if new(big.Int).Mul(n, power(-exponent)).Cmp(d) < 0 {
		exponent--
	}
	scale := precision - 1 - exponent
	numerator, denominator := new(big.Int).Set(n), new(big.Int).Set(d)
	if scale >= 0 {
		numerator.Mul(numerator, power(scale))
	} else {
		denominator.Mul(denominator, power(-scale))
	}
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	tie := new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(denominator)
	if tie > 0 || (tie == 0 && quotient.Bit(0) != 0) {
		quotient.Add(quotient, big.NewInt(1))
	}
	if sign < 0 {
		quotient.Neg(quotient)
	}
	if scale >= 0 {
		return NewDecimal(new(big.Rat).SetFrac(quotient, power(scale)))
	}
	return NewDecimal(new(big.Rat).SetInt(new(big.Int).Mul(quotient, power(-scale))))
}

func bookedPrice(posting Posting) Posting {
	if posting.Price == nil || !posting.Price.Total || posting.Units == nil || posting.Units.Number.Raw == "" {
		return posting
	}
	units := DecimalFromNumber(posting.Units.Number)
	if units.Sign() < 0 {
		units = units.Neg()
	}
	price := *posting.Price
	normalized := Zero()
	if !units.IsZero() {
		normalized = roundedSignificant(DecimalFromNumber(price.Amount.Number).Quo(units), 28)
	}
	price.PerUnit = &Amount{Number: numberFromDecimal(normalized), Currency: price.Amount.Currency, At: price.Amount.At}
	posting.Price = &price
	return posting
}

// BookedPriceWeight returns the typed reference query's price-valued units:
// normalized unit quote times signed units in its fixed Decimal context. It
// does not change the posting or replace PostingWeight's exact source total.
// Costed callers use their cost basis instead of this price-only operation.
func BookedPriceWeight(posting Posting) Decimal {
	units := DecimalFromNumber(posting.Units.Number)
	if posting.Price == nil || units.IsZero() {
		return units
	}
	if posting.Price.Total && posting.Price.PerUnit == nil {
		posting = bookedPrice(posting)
	}
	quote := posting.Price.Amount
	if posting.Price.PerUnit != nil {
		quote = *posting.Price.PerUnit
	}
	return roundedSignificant(units.Mul(DecimalFromNumber(quote.Number)), 28)
}

func groupBookedPostings(postings []Posting) []Posting {
	order := []string{}
	groups := map[string][]Posting{}
	for _, p := range postings {
		if p.Units == nil {
			return postings
		}
		currency := p.Units.Currency
		if p.Cost != nil {
			if cost, ok := normalizeCost(*p.Cost); ok {
				currency = cost.Currency
			}
		} else if p.Price != nil {
			currency = p.Price.Amount.Currency
		}
		if _, exists := groups[currency]; !exists {
			order = append(order, currency)
		}
		groups[currency] = append(groups[currency], p)
	}
	result := make([]Posting, 0, len(postings))
	for _, currency := range order {
		result = append(result, groups[currency]...)
	}
	return result
}
