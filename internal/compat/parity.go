// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package compat

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
)

// This file implements the ADR-0008 differential dimensions against golden
// snapshots produced by the Beancount v3 oracle (tools/reference). Every
// normalized value is sanitized corpus data, never private ledger content.

// ParityBalance is the normalized per-account state shared by both engines:
// costless per-currency sums plus sorted costed lot strings.
type ParityBalance struct {
	Currencies map[string]string `json:"currencies"`
	Lots       []string          `json:"lots"`
}

// GoldenError is one oracle diagnostic reduced to its class and line.
type GoldenError struct {
	Type string `json:"type"`
	Line int    `json:"line"`
}

// Golden mirrors one testdata/golden/v3-parity snapshot.
type Golden struct {
	Fixture      string                     `json:"fixture"`
	OracleEngine string                     `json:"oracle_engine"`
	Valid        bool                       `json:"valid"`
	Errors       []GoldenError              `json:"errors"`
	Entries      []string                   `json:"entries"`
	Balances     map[string]ParityBalance   `json:"balances"`
	Queries      map[string]json.RawMessage `json:"queries"`
}

// OCSide is OrangeCount's normalized view of the same fixture.
type OCSide struct {
	Valid    bool
	Errors   []GoldenError // reuse: Type holds the OC diagnostic code
	Entries  []string
	Balances map[string]ParityBalance
	Queries  map[string][][]any
}

// DiffRecord is one redacted difference between the engines.
type DiffRecord struct {
	Fixture   string         `json:"fixture"`
	Dimension string         `json:"dimension"`
	Kind      string         `json:"kind"`
	Detail    map[string]any `json:"detail,omitempty"`
	Boundary  string         `json:"boundary,omitempty"`
}

// cashEntriesSQL is the M4 query. The identical text must run on both
// engines and produce identical rows.
const cashEntriesSQL = "SELECT date, narration FROM postings WHERE account ~ 'Cash' ORDER BY date"

// canonicalNumber normalizes a decimal source spelling the same way on both
// sides: plain notation with trailing fractional zeros removed.
func canonicalNumber(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "0"
	}
	if strings.ContainsAny(s, "/") { // non-terminating fraction form
		return s
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	if s == "" || s == "-" || s == "+0" || s == "-0" {
		return "0"
	}
	return s
}

func canonicalAmountNumber(n ledger.Number) string { return canonicalNumber(n.Raw) }

func canonicalDecimal(d ledger.Decimal) string { return canonicalNumber(d.String()) }

func formatAmount(number, currency string) string { return number + " " + currency }

// perUnitPrice normalizes a whole-position `@@` price to the per-unit
// spelling the oracle uses on booked entries.
func perUnitPrice(p ledger.Posting, unitCount string) string {
	if !p.Price.Total {
		return canonicalAmountNumber(p.Price.Amount.Number)
	}
	units, ok := new(big.Rat).SetString(unitCount)
	if !ok || units.Sign() == 0 {
		return canonicalAmountNumber(p.Price.Amount.Number)
	}
	total := p.Price.Amount.Number.Rat
	if total == nil {
		return canonicalAmountNumber(p.Price.Amount.Number)
	}
	perUnit := new(big.Rat).Quo(total, units)
	return canonicalRat(perUnit)
}

// canonicalRat renders an exact rational with trailing zeros trimmed.
func canonicalRat(r *big.Rat) string {
	s := r.FloatString(12)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	return s
}

func formatDate(d ledger.Date) string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

func sortedCopy(values []string) []string {
	out := []string{}
	out = append(out, values...)
	sort.Strings(out)
	return out
}

// normCostSpec reduces an OC cost specification to "number currency", the
// same subset the oracle exposes on booked entries.
func normCostSpec(spec *ledger.CostSpec) any {
	if spec == nil {
		return nil
	}
	var number, currency string
	for _, value := range spec.Components {
		switch value.Kind {
		case ledger.ValueAmount:
			number = canonicalAmountNumber(value.Amount.Number)
			currency = value.Amount.Currency
		case ledger.ValueNumber:
			if number == "" {
				number = canonicalAmountNumber(value.Number)
			}
		case ledger.ValueCurrency:
			if currency == "" {
				currency = value.String
			}
		}
	}
	if number == "" || currency == "" {
		return nil
	}
	return formatAmount(number, currency)
}

func normValue(v ledger.Value) any {
	switch v.Kind {
	case ledger.ValueNumber:
		return canonicalAmountNumber(v.Number)
	case ledger.ValueBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case ledger.ValueDate:
		return formatDate(v.Date)
	case ledger.ValueAmount:
		return formatAmount(canonicalAmountNumber(v.Amount.Number), v.Amount.Currency)
	default:
		return v.String
	}
}

// normalizeTransaction maps one booked transaction to the golden entry shape.
func normalizeTransaction(t ledger.Transaction) (map[string]any, bool) {
	postings := make([]map[string]any, 0, len(t.Postings))
	for _, p := range t.Postings {
		posting := map[string]any{"account": p.Account}
		unitCount := ""
		if p.Units != nil && p.Units.Currency != "" && p.Units.Number.Raw != "" {
			unitCount = canonicalAmountNumber(p.Units.Number)
			posting["units"] = formatAmount(unitCount, p.Units.Currency)
		} else {
			posting["units"] = nil
		}
		posting["cost"] = normCostSpec(p.Cost)
		if p.Price != nil && p.Price.Amount.Currency != "" {
			posting["price"] = formatAmount(perUnitPrice(p, unitCount), p.Price.Amount.Currency)
		} else {
			posting["price"] = nil
		}
		postings = append(postings, posting)
	}
	var payee any
	if t.Payee != "" {
		payee = t.Payee
	}
	entry := map[string]any{
		"kind":      "transaction",
		"date":      formatDate(t.Date),
		"flag":      t.Flag,
		"payee":     payee,
		"narration": t.Narration,
		"tags":      sortedCopy(t.Tags),
		"links":     sortedCopy(t.Links),
		"postings":  postings,
	}
	return entry, true
}

// normalizeDirective maps one OC AST directive to the golden entry shape.
// The result marshals with sorted keys so both sides compare as strings.
func normalizeDirective(d ledger.Directive) (map[string]any, bool) {
	switch t := d.(type) {
	case *ledger.Transaction:
		return normalizeTransaction(*t)
	case ledger.Transaction:
		return normalizeTransaction(t)
	}
	entry := map[string]any{}
	switch t := d.(type) {
	case ledger.Open:
		var booking any
		if t.Booking != "" {
			booking = t.Booking
		}
		entry["kind"] = "open"
		entry["date"] = formatDate(t.Date)
		entry["account"] = t.Account
		entry["currencies"] = sortedCopy(t.Currencies)
		entry["booking"] = booking
	case ledger.Close:
		entry["kind"] = "close"
		entry["date"] = formatDate(t.Date)
		entry["account"] = t.Account
	case ledger.Balance:
		entry["kind"] = "balance"
		entry["date"] = formatDate(t.Date)
		entry["account"] = t.Account
		entry["amount"] = formatAmount(canonicalAmountNumber(t.Amount.Number), t.Amount.Currency)
		if t.Tolerance != nil {
			entry["tolerance"] = canonicalAmountNumber(*t.Tolerance)
		} else {
			entry["tolerance"] = nil
		}
	case ledger.Commodity:
		entry["kind"] = "commodity"
		entry["date"] = formatDate(t.Date)
		entry["currency"] = t.Currency
	case ledger.Pad:
		entry["kind"] = "pad"
		entry["date"] = formatDate(t.Date)
		entry["account"] = t.Account
		entry["source_account"] = t.SourceAccount
	case ledger.Event:
		entry["kind"] = "event"
		entry["date"] = formatDate(t.Date)
		entry["event_type"] = t.Type
		entry["value"] = t.Value
	case ledger.Query:
		entry["kind"] = "query"
		entry["date"] = formatDate(t.Date)
		entry["name"] = t.Name
		entry["query"] = t.Query
	case ledger.Price:
		entry["kind"] = "price"
		entry["date"] = formatDate(t.Date)
		entry["currency"] = t.Currency
		entry["amount"] = formatAmount(canonicalAmountNumber(t.Amount.Number), t.Amount.Currency)
	case ledger.Document:
		entry["kind"] = "document"
		entry["date"] = formatDate(t.Date)
		entry["account"] = t.Account
		if len(t.Filenames) > 0 {
			entry["filename"] = filepath.Base(t.Filenames[0])
		}
		entry["tags"] = sortedCopy(t.Tags)
		entry["links"] = sortedCopy(t.Links)
	case ledger.Note:
		entry["kind"] = "note"
		entry["date"] = formatDate(t.Date)
		entry["account"] = t.Account
		entry["comment"] = t.Comment
	case ledger.Custom:
		values := make([]any, 0, len(t.Values))
		for _, v := range t.Values {
			values = append(values, normValue(v))
		}
		entry["kind"] = "custom"
		entry["date"] = formatDate(t.Date)
		entry["custom_type"] = t.Type
		entry["values"] = values
	default:
		return nil, false
	}
	return entry, true
}

// BuildOCSide evaluates one fixture with OrangeCount and normalizes it into
// the golden comparison shape.
func BuildOCSide(fixturePath string) (OCSide, []diagnostic.Diagnostic, error) {
	result := snapshot.Build(fixturePath)
	side := OCSide{Valid: false, Errors: nil}
	var diags []diagnostic.Diagnostic
	if result.Snapshot != nil {
		diags = result.Snapshot.Diagnostics()
	} else {
		diags = result.Diagnostics
	}
	for _, item := range diags {
		if item.Severity != diagnostic.Error {
			continue
		}
		side.Errors = append(side.Errors, GoldenError{Type: item.Code, Line: item.Span.StartLine})
	}
	side.Valid = len(side.Errors) == 0
	if !side.Valid {
		return side, diags, nil
	}
	evaluation := result.Snapshot.Evaluation()
	for _, record := range evaluation.Entries {
		entry, ok := normalizeDirective(record.Directive)
		if !ok {
			continue
		}
		encoded, err := json.Marshal(entry)
		if err != nil {
			return side, diags, err
		}
		side.Entries = append(side.Entries, string(encoded))
	}
	sort.Strings(side.Entries)
	side.Balances = normalizeBalances(evaluation)
	rows, err := evaluateCashEntries(evaluation)
	if err != nil {
		return side, diags, err
	}
	side.Queries = map[string][][]any{"cash_entries": rows}
	return side, diags, nil
}

// normalizeBalances derives the oracle-shaped per-account state: lot units
// appear only in the lots list, so per-currency sums exclude them.
func normalizeBalances(evaluation ledger.Evaluation) map[string]ParityBalance {
	out := map[string]ParityBalance{}
	for name, state := range evaluation.Accounts {
		balance := ParityBalance{Currencies: map[string]string{}, Lots: []string{}}
		lotUnits := map[string]ledger.Decimal{}
		lotKeys := []string{}
		for _, position := range state.Positions {
			if position.Cost == nil {
				continue
			}
			units := canonicalDecimal(position.Units)
			key := fmt.Sprintf("%s %s {%s %s}", units, position.Currency,
				canonicalDecimal(position.Cost.Number), position.Cost.Currency)
			lotKeys = append(lotKeys, key)
			previous, ok := lotUnits[position.Currency]
			if !ok {
				previous = ledger.Zero()
			}
			lotUnits[position.Currency] = previous.Add(position.Units)
		}
		sort.Strings(lotKeys)
		balance.Lots = lotKeys
		currencies := []string{}
		for currency := range state.Balances {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)
		for _, currency := range currencies {
			net := state.Balances[currency]
			if held, ok := lotUnits[currency]; ok {
				net = net.Sub(held)
			}
			if net.Rat().Sign() == 0 {
				continue
			}
			balance.Currencies[currency] = canonicalDecimal(net)
		}
		if len(balance.Currencies) == 0 && len(balance.Lots) == 0 {
			continue
		}
		out[name] = balance
	}
	return out
}

func evaluateCashEntries(evaluation ledger.Evaluation) ([][]any, error) {
	result, err := query.Evaluate(cashEntriesSQL, evaluation)
	if err != nil {
		return nil, err
	}
	rows := make([][]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		cells := make([]any, 0, len(result.Columns))
		for _, column := range result.Columns {
			cells = append(cells, fmt.Sprintf("%v", row[column]))
		}
		rows = append(rows, cells)
	}
	return rows, nil
}

// LoadGolden reads one oracle snapshot from disk.
func LoadGolden(path string) (Golden, error) {
	var golden Golden
	data, err := os.ReadFile(path)
	if err != nil {
		return golden, err
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		return golden, fmt.Errorf("golden %s: %w", filepath.Base(path), err)
	}
	return golden, nil
}

// lineSet returns the sorted unique diagnostic lines.
func lineSet(errors []GoldenError) []int {
	seen := map[int]bool{}
	lines := []int{}
	for _, item := range errors {
		if !seen[item.Line] {
			seen[item.Line] = true
			lines = append(lines, item.Line)
		}
	}
	sort.Ints(lines)
	return lines
}

func missingStrings(want, have []string) []string {
	haveSet := map[string]bool{}
	for _, item := range have {
		haveSet[item] = true
	}
	missing := []string{}
	for _, item := range want {
		if !haveSet[item] {
			missing = append(missing, item)
		}
	}
	return missing
}

// DiffFixture compares one fixture against its oracle snapshot and returns
// the redacted difference records. Every record is evidence for triage;
// records whose Kind maps to an approved boundary are tagged.
func DiffFixture(fixture string, golden Golden, oc OCSide) []DiffRecord {
	records := []DiffRecord{}
	add := func(dimension, kind string, detail map[string]any) {
		records = append(records, DiffRecord{
			Fixture: fixture, Dimension: dimension, Kind: kind, Detail: detail,
			Boundary: approvedBoundary(kind),
		})
	}

	// M6 validity.
	if golden.Valid != oc.Valid {
		add("M6-validity", "validity-mismatch", map[string]any{
			"oracle_valid": golden.Valid, "orangecount_valid": oc.Valid,
		})
	}

	if !golden.Valid || !oc.Valid {
		// M2 diagnostics: compare error counts and error lines only.
		if len(golden.Errors) != len(oc.Errors) {
			add("M2-diagnostics", "diagnostic-count", map[string]any{
				"oracle": len(golden.Errors), "orangecount": len(oc.Errors),
				"oracle_types": errorTypes(golden.Errors), "orangecount_codes": errorTypes(oc.Errors),
			})
		}
		oracleLines := lineSet(golden.Errors)
		ocLines := lineSet(oc.Errors)
		if fmt.Sprint(oracleLines) != fmt.Sprint(ocLines) {
			add("M2-diagnostics", "diagnostic-lines", map[string]any{
				"oracle_lines": oracleLines, "orangecount_lines": ocLines,
			})
		}
		return records
	}

	// M1 entries.
	missingEntries := missingStrings(golden.Entries, oc.Entries)
	extraEntries := missingStrings(oc.Entries, golden.Entries)
	if len(missingEntries) > 0 || len(extraEntries) > 0 {
		add("M1-entries", "entries-diverge", map[string]any{
			"oracle_only": missingEntries, "orangecount_only": extraEntries,
		})
	}

	// M3 accounting state.
	if !sameBalances(golden.Balances, oc.Balances) {
		add("M3-state", "balances-diverge", map[string]any{
			"oracle_only_accounts":      missingBalanceKeys(golden.Balances, oc.Balances),
			"orangecount_only_accounts": missingBalanceKeys(oc.Balances, golden.Balances),
			"differing_accounts":        differingAccounts(golden.Balances, oc.Balances),
		})
	}

	// M4 queries.
	oracleRows := decodeGoldenQuery(golden.Queries, "cash_entries")
	if oracleRows == nil {
		add("M4-queries", "query-unsupported", map[string]any{"engine": "oracle"})
	} else if oc.Queries == nil {
		add("M4-queries", "query-unsupported", map[string]any{"engine": "orangecount"})
	} else if ocRows := oc.Queries["cash_entries"]; !sameQueryRows(oracleRows, ocRows) {
		add("M4-queries", "query-rows-diverge", map[string]any{
			"oracle": oracleRows, "orangecount": ocRows,
		})
	}

	// M5 report: the balance table derived from each side's state.
	oracleReport := renderBalanceReport(golden.Balances)
	ocReport := renderBalanceReport(oc.Balances)
	if oracleReport != ocReport {
		add("M5-reports", "report-diverge", map[string]any{
			"oracle_rows": strings.Count(oracleReport, "\n"),
			"equal":       false,
		})
	}
	return records
}

func errorTypes(errors []GoldenError) []string {
	out := []string{}
	for _, item := range errors {
		out = append(out, item.Type)
	}
	sort.Strings(out)
	return out
}

func sameBalances(want, have map[string]ParityBalance) bool {
	if len(want) != len(have) {
		return false
	}
	for account, wantBalance := range want {
		haveBalance, ok := have[account]
		if !ok {
			return false
		}
		if len(wantBalance.Currencies) != len(haveBalance.Currencies) {
			return false
		}
		for currency, amount := range wantBalance.Currencies {
			if haveBalance.Currencies[currency] != amount {
				return false
			}
		}
		if strings.Join(wantBalance.Lots, "|") != strings.Join(haveBalance.Lots, "|") {
			return false
		}
	}
	return true
}

func missingBalanceKeys(want, have map[string]ParityBalance) []string {
	keys := []string{}
	for account := range want {
		if _, ok := have[account]; !ok {
			keys = append(keys, account)
		}
	}
	sort.Strings(keys)
	return keys
}

func differingAccounts(want, have map[string]ParityBalance) []string {
	keys := []string{}
	for account, wantBalance := range want {
		haveBalance, ok := have[account]
		if !ok {
			continue
		}
		equal := len(wantBalance.Currencies) == len(haveBalance.Currencies)
		if equal {
			for currency, amount := range wantBalance.Currencies {
				if haveBalance.Currencies[currency] != amount {
					equal = false
					break
				}
			}
		}
		if equal {
			equal = strings.Join(wantBalance.Lots, "|") == strings.Join(haveBalance.Lots, "|")
		}
		if !equal {
			keys = append(keys, account)
		}
	}
	sort.Strings(keys)
	return keys
}

func decodeGoldenQuery(queries map[string]json.RawMessage, name string) [][]any {
	raw, ok := queries[name]
	if !ok {
		return nil
	}
	var rows [][]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil
	}
	return rows
}

func sameQueryRows(want, have [][]any) bool {
	if len(want) != len(have) {
		return false
	}
	for i, wantRow := range want {
		haveRow := have[i]
		if len(wantRow) != len(haveRow) {
			return false
		}
		for j, wantCell := range wantRow {
			if fmt.Sprint(wantCell) != fmt.Sprint(haveRow[j]) {
				return false
			}
		}
	}
	return true
}

// renderBalanceReport renders the M5 balance table: one
// "account currency amount" line per currency plus lot lines, sorted.
func renderBalanceReport(balances map[string]ParityBalance) string {
	lines := []string{}
	accounts := []string{}
	for account := range balances {
		accounts = append(accounts, account)
	}
	sort.Strings(accounts)
	for _, account := range accounts {
		balance := balances[account]
		currencies := []string{}
		for currency := range balance.Currencies {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)
		for _, currency := range currencies {
			lines = append(lines, fmt.Sprintf("%s %s %s", account, currency, balance.Currencies[currency]))
		}
		for _, lot := range balance.Lots {
			lines = append(lines, fmt.Sprintf("%s %s", account, lot))
		}
	}
	return strings.Join(lines, "\n")
}
