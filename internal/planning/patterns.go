// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

package planning

import (
	"sort"
	"strings"
	"time"

	"orangecount/internal/ledger"
)

// Occurrence is the normalized historical evidence used for local pattern and
// match suggestions. It deliberately contains no probabilistic score.
type Occurrence struct {
	ID      string
	Date    ledger.Date
	Account string
	Name    string
	Amount  ledger.Decimal
}

// CashFlowPattern exposes exactly why a recurring candidate was suggested.
type CashFlowPattern struct {
	Key           string
	Name          string
	Account       string
	Occurrences   []Occurrence
	CadenceDays   int
	MinimumAmount ledger.Decimal
	MaximumAmount ledger.Decimal
}

// DetectPatterns groups matching normalized name/account occurrences. A group
// needs at least minOccurrences and an exactly repeatable calendar-day gap to
// become a candidate; the strictness is intentional for the first deterministic
// version and avoids calling merely similar spending a future obligation.
func DetectPatterns(occurrences []Occurrence, minOccurrences int) []CashFlowPattern {
	if minOccurrences < 2 {
		minOccurrences = 2
	}
	groups := map[string][]Occurrence{}
	for _, occurrence := range occurrences {
		if !occurrence.Date.Valid() || occurrence.Amount.Sign() <= 0 {
			continue
		}
		name := normalizeName(occurrence.Name)
		if name == "" || strings.TrimSpace(occurrence.Account) == "" {
			continue
		}
		key := occurrence.Account + "\x00" + name
		groups[key] = append(groups[key], occurrence)
	}
	var patterns []CashFlowPattern
	for key, group := range groups {
		if len(group) < minOccurrences {
			continue
		}
		sort.Slice(group, func(i, j int) bool { return compareDate(group[i].Date, group[j].Date) < 0 })
		cadence := daysBetween(group[0].Date, group[1].Date)
		if cadence <= 0 {
			continue
		}
		consistent := true
		for index := 2; index < len(group); index++ {
			if daysBetween(group[index-1].Date, group[index].Date) != cadence {
				consistent = false
				break
			}
		}
		if !consistent {
			continue
		}
		minimum, maximum := group[0].Amount, group[0].Amount
		for _, occurrence := range group[1:] {
			if occurrence.Amount.Cmp(minimum) < 0 {
				minimum = occurrence.Amount
			}
			if occurrence.Amount.Cmp(maximum) > 0 {
				maximum = occurrence.Amount
			}
		}
		patterns = append(patterns, CashFlowPattern{Key: key, Name: normalizeName(group[0].Name), Account: group[0].Account, Occurrences: append([]Occurrence(nil), group...), CadenceDays: cadence, MinimumAmount: minimum, MaximumAmount: maximum})
	}
	sort.Slice(patterns, func(i, j int) bool { return patterns[i].Key < patterns[j].Key })
	return patterns
}

func normalizeName(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func daysBetween(left, right ledger.Date) int {
	leftTime := timeAtMidnight(left)
	rightTime := timeAtMidnight(right)
	return int(rightTime.Sub(leftTime).Hours() / 24)
}

func timeAtMidnight(date ledger.Date) time.Time {
	return time.Date(date.Year, time.Month(date.Month), date.Day, 0, 0, 0, 0, time.UTC)
}
