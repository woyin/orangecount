// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package main

// The generative differential compares six dimensions of accounting state. All
// six are derived from the same fixture file, so a difference always names the
// engine and never the harness:
//
//  1. validity            does the ledger evaluate at all;
//  2. diagnostics         which error classes, and with what multiplicity;
//  3. ordered entries     the normalized entry stream in source order, so a
//                         reordering is visible (a sorted multiset hides it);
//  4. prefix inventories  the complete account inventory after *every*
//                         transaction, not only the final balances;
//  5. prices              every price directive as date/base/quote/rate;
//  6. period income       income and expense posting totals per month and
//                         currency.
//
// Dimensions 1-2 come from compat.FullGoldenDifferences. Dimensions 3-6 are
// produced here on the Go side and by genprediff_oracle.py on the oracle side.

import (
	"encoding/json"
	"fmt"
	"sort"

	"orangecount/internal/compat"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

// oracleRecord is one decoded oracle entry with every dimension available. The
// schema-2 fields are decoded by compat.FullGolden; these extra fields carry the
// dimensions the schema cannot express.
type oracleRecord struct {
	Fixture           string                         `json:"fixture"`
	Valid             bool                           `json:"valid"`
	OrderedEntries    []string                       `json:"ordered_entries"`
	PrefixInventories []map[string]map[string]string `json:"prefix_inventories"`
	Prices            []string                       `json:"prices"`
	PeriodIncome      map[string]string              `json:"period_income"`
}

func readOracleRecords(encoded []byte) (map[string]oracleRecord, error) {
	var raw []oracleRecord
	if err := json.Unmarshal(encoded, &raw); err != nil {
		return nil, fmt.Errorf("decode oracle: %w", err)
	}
	result := map[string]oracleRecord{}
	for _, item := range raw {
		result[item.Fixture] = item
	}
	return result, nil
}

// goDimensions is the Go engine side of dimensions 3-6.
type goDimensions struct {
	OrderedEntries    []string
	PrefixInventories []map[string]map[string]string
	Prices            []string
	PeriodIncome      map[string]string
}

// buildGoDimensions derives dimensions 3-6 from one evaluated ledger.
func buildGoDimensions(path string) (goDimensions, error) {
	build := snapshot.BuildWithOptions(path, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	golden := compat.FullSideFromBuild(path, build)
	result := goDimensions{PeriodIncome: map[string]string{}}
	if golden.Valid == false || build.Snapshot == nil {
		// An invalid ledger has no accepted state on either side.
		result.OrderedEntries = []string{}
		result.PrefixInventories = []map[string]map[string]string{}
		result.Prices = []string{}
		return result, nil
	}

	result.OrderedEntries = orderedEntries(golden.State.Entries)

	evaluation := build.Snapshot.Evaluation()
	running := map[string]map[string]ledger.Decimal{}
	for _, record := range evaluation.Entries {
		tx, ok := transactionDirective(record.Directive)
		if !ok {
			continue
		}
		for _, posting := range tx.Postings {
			if posting.Units == nil {
				continue
			}
			currency := posting.Units.Currency
			if currency == "" {
				continue
			}
			amount := ledger.DecimalFromNumber(posting.Units.Number)
			if running[posting.Account] == nil {
				running[posting.Account] = map[string]ledger.Decimal{}
			}
			running[posting.Account][currency] = running[posting.Account][currency].Add(amount)
		}
		snapshotInventory := map[string]map[string]string{}
		for account, currencies := range running {
			trimmed := map[string]string{}
			for currency, amount := range currencies {
				if !amount.IsZero() {
					trimmed[currency] = amount.String()
				}
			}
			if len(trimmed) != 0 {
				snapshotInventory[account] = trimmed
			}
		}
		result.PrefixInventories = append(result.PrefixInventories, snapshotInventory)
		if result.PrefixInventories == nil {
			result.PrefixInventories = []map[string]map[string]string{}
		}
	}
	if result.PrefixInventories == nil {
		result.PrefixInventories = []map[string]map[string]string{}
	}

	result.Prices = priceRows(&evaluation)
	result.PeriodIncome = periodIncome(&evaluation)
	return result, nil
}

func transactionDirective(directive ledger.Directive) (*ledger.Transaction, bool) {
	switch value := directive.(type) {
	case ledger.Transaction:
		return &value, true
	case *ledger.Transaction:
		return value, true
	default:
		return nil, false
	}
}

// orderedEntries re-sorts the normalized entry multiset into source order.
// compat's normalization embeds each entry's source line, so the line recovers
// the order the entry stream was evaluated in. A stable sort keeps entries that
// share a line (a directive and the transaction a `pad` synthesizes) in the
// evaluator's own order.
type entryWithLine struct {
	line    int
	content string
	seq     int
}

func orderedEntries(entries []string) []string {
	rows := make([]entryWithLine, 0, len(entries))
	for index, entry := range entries {
		line := 0
		var probe struct {
			Source struct {
				Line int `json:"line"`
			} `json:"source"`
		}
		if err := json.Unmarshal([]byte(entry), &probe); err == nil {
			line = probe.Source.Line
		}
		rows = append(rows, entryWithLine{line: line, content: entry, seq: index})
	}
	sort.SliceStable(rows, func(left, right int) bool {
		if rows[left].line != rows[right].line {
			return rows[left].line < rows[right].line
		}
		return rows[left].seq < rows[right].seq
	})
	ordered := make([]string, 0, len(rows))
	for _, row := range rows {
		ordered = append(ordered, row.content)
	}
	return ordered
}

// priceRows flattens the evaluator's price map into sorted JSON rows so the
// comparison is order independent but content exact.
func priceRows(evaluation *ledger.Evaluation) []string {
	rows := []string{}
	for _, quotes := range evaluation.Prices {
		for _, quote := range quotes {
			encoded, err := json.Marshal(map[string]string{
				"date":  quote.Date.Raw,
				"base":  quote.Base,
				"quote": quote.Currency,
				"rate":  quote.Amount.String(),
			})
			if err != nil {
				continue
			}
			rows = append(rows, string(encoded))
		}
	}
	sort.Strings(rows)
	return rows
}

// periodIncome totals income and expense postings per month and currency.
func periodIncome(evaluation *ledger.Evaluation) map[string]string {
	totals := map[string]ledger.Decimal{}
	for _, record := range evaluation.Entries {
		tx, ok := transactionDirective(record.Directive)
		if !ok {
			continue
		}
		period := fmt.Sprintf("%04d-%02d", record.Date.Year, record.Date.Month)
		for _, posting := range tx.Postings {
			if posting.Units == nil {
				continue
			}
			root := posting.Account
			if index := indexOf(root, ':'); index > 0 {
				root = root[:index]
			}
			if root != "Income" && root != "Expenses" {
				continue
			}
			key := period + "|" + posting.Units.Currency
			totals[key] = totals[key].Add(ledger.DecimalFromNumber(posting.Units.Number))
		}
	}
	result := map[string]string{}
	for key, total := range totals {
		if !total.IsZero() {
			result[key] = total.String()
		}
	}
	return result
}

func indexOf(value string, target byte) int {
	for index := 0; index < len(value); index++ {
		if value[index] == target {
			return index
		}
	}
	return -1
}

// dimensionDifferences compares dimensions 3-6 and names the first offending
// element, which is the actionable part of an ordered comparison.
func dimensionDifferences(want oracleRecord, have goDimensions) []string {
	differences := []string{}
	if !equalStrings(want.OrderedEntries, have.OrderedEntries) {
		differences = append(differences, "ordered-entries"+firstMismatch(want.OrderedEntries, have.OrderedEntries))
	}
	if len(want.PrefixInventories) != len(have.PrefixInventories) {
		differences = append(differences, fmt.Sprintf("prefix-count(want=%d,have=%d)", len(want.PrefixInventories), len(have.PrefixInventories)))
	} else {
		for index := range want.PrefixInventories {
			if canonicalInventory(want.PrefixInventories[index]) != canonicalInventory(have.PrefixInventories[index]) {
				differences = append(differences, fmt.Sprintf("prefix-inventory[%d]", index))
				break
			}
		}
	}
	if !equalStrings(want.Prices, have.Prices) {
		differences = append(differences, "prices"+firstMismatch(want.Prices, have.Prices))
	}
	if !equalMaps(want.PeriodIncome, have.PeriodIncome) {
		differences = append(differences, "period-income"+firstMapMismatch(want.PeriodIncome, have.PeriodIncome))
	}
	return differences
}

func canonicalInventory(value map[string]map[string]string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(encoded)
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalMaps(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func firstMismatch(want, have []string) string {
	limit := len(want)
	if len(have) < limit {
		limit = len(have)
	}
	for index := 0; index < limit; index++ {
		if want[index] != have[index] {
			return fmt.Sprintf("[%d:want=%q,have=%q]", index, truncate(want[index]), truncate(have[index]))
		}
	}
	return fmt.Sprintf("[length:want=%d,have=%d]", len(want), len(have))
}

func firstMapMismatch(want, have map[string]string) string {
	keys := make([]string, 0, len(want))
	for key := range want {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if have[key] != want[key] {
			return fmt.Sprintf("[%s:want=%q,have=%q]", key, want[key], have[key])
		}
	}
	for key, value := range have {
		if _, ok := want[key]; !ok {
			return fmt.Sprintf("[%s:want=<missing>,have=%q]", key, value)
		}
	}
	return ""
}

func truncate(value string) string {
	if len(value) <= 120 {
		return value
	}
	return value[:120] + "..."
}
