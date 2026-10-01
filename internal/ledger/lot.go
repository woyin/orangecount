// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package ledger

// sameLot compares inventory identity, unlike reduction constraints where an
// omitted date or label is a wildcard. Source span is provenance, not identity.
func sameLot(a, b Position) bool {
	if a.Currency != b.Currency || (a.Cost == nil) != (b.Cost == nil) {
		return false
	}
	if a.Cost == nil {
		return true
	}
	x, y := a.Cost, b.Cost
	if !x.Number.Equal(y.Number) || x.Currency != y.Currency || x.Label != y.Label || x.LabelPresent != y.LabelPresent || (x.Date == nil) != (y.Date == nil) {
		return false
	}
	return x.Date == nil || dateKey(*x.Date) == dateKey(*y.Date)
}

func appendOrMergeLot(positions []Position, added Position) []Position {
	for i := range positions {
		if sameLot(positions[i], added) {
			positions[i].Units = positions[i].Units.Add(added.Units)
			if positions[i].Units.IsZero() {
				return append(positions[:i], positions[i+1:]...)
			}
			return positions
		}
	}
	if added.Units.IsZero() {
		return positions
	}
	return append(positions, added)
}

// Acquisition normalization is performed on booked postings, so inventory,
// journal, query and exported evaluated views all share the same cost date.
func acquisitionPosting(posting Posting, date Date, none ...bool) Posting {
	if posting.Cost == nil || posting.Units == nil || posting.Units.Number.Raw == "" {
		return posting
	}
	units := DecimalFromNumber(posting.Units.Number)
	if units.Sign() <= 0 && !(len(none) > 0 && none[0]) {
		return posting
	}
	cost, ok := normalizeCost(*posting.Cost)
	if !ok {
		return posting
	}
	if posting.Cost.Total && !units.IsZero() {
		if units.Sign() < 0 {
			units = units.Neg()
		}
		cost.Number = cost.Number.Quo(units)
	}
	if cost.Date == nil {
		acquired := date
		cost.Date = &acquired
	}
	spec := costSpecFromCost(cost, posting.Cost)
	posting.Cost = &spec
	return posting
}

func completeCost(spec CostSpec) bool {
	for _, v := range spec.Components {
		if v.Kind == ValueAmount && v.Amount.Number.Raw != "" && v.Amount.Currency != "" {
			return true
		}
	}
	return false
}
