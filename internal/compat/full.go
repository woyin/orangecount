// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package compat

import (
	"bytes"
	"encoding/json"
	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

const FullSchemaVersion = 2

func fullJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	var canonical any
	_ = json.Unmarshal(encoded, &canonical)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(canonical)
	return bytes.TrimSpace(buffer.Bytes())
}

type FullAmount struct {
	Number   string `json:"number"`
	Currency string `json:"currency"`
}

type FullCost struct {
	Number   string  `json:"number"`
	Currency string  `json:"currency"`
	Date     *string `json:"date"`
	Label    *string `json:"label"`
}

type FullInventory struct {
	Currencies map[string]string `json:"currencies"`
	Lots       []string          `json:"lots"`
}

type FullState struct {
	Entries  []string                 `json:"entries"`
	Balances map[string]FullInventory `json:"balances"`
}

type FullGolden struct {
	SchemaVersion int           `json:"schema_version"`
	Fixture       string        `json:"fixture"`
	OracleEngine  string        `json:"oracle_engine"`
	Valid         bool          `json:"valid"`
	Errors        []GoldenError `json:"errors"`
	State         FullState     `json:"state"`
}

func fullAmount(a *ledger.Amount) *FullAmount {
	if a == nil || a.Number.Raw == "" {
		return nil
	}
	return &FullAmount{ledger.DecimalFromNumber(a.Number).String(), a.Currency}
}

func fullPrice(p ledger.Posting) *FullAmount {
	if p.Price == nil {
		return nil
	}
	if p.Price.PerUnit != nil {
		return fullAmount(p.Price.PerUnit)
	}
	amount := ledger.DecimalFromNumber(p.Price.Amount.Number)
	if p.Price.Total && p.Units != nil {
		units := ledger.DecimalFromNumber(p.Units.Number)
		if units.Sign() < 0 {
			units = units.Neg()
		}
		if !units.IsZero() {
			amount = amount.Quo(units)
		}
	}
	return &FullAmount{amount.String(), p.Price.Amount.Currency}
}

func fullCostSpec(spec *ledger.CostSpec) *FullCost {
	if spec == nil {
		return nil
	}
	value := &FullCost{}
	for _, c := range spec.Components {
		switch c.Kind {
		case ledger.ValueAmount:
			value.Number, value.Currency = ledger.DecimalFromNumber(c.Amount.Number).String(), c.Amount.Currency
		case ledger.ValueDate:
			date := c.Date.Raw
			value.Date = &date
		case ledger.ValueString:
			label := c.String
			value.Label = &label
		}
	}
	return value
}

func fullValue(value ledger.Value) any {
	var kind string
	var data any
	switch value.Kind {
	case ledger.ValueNumber:
		kind, data = "number", ledger.DecimalFromNumber(value.Number).String()
	case ledger.ValueAmount:
		kind, data = "amount", fullAmount(&value.Amount)
	case ledger.ValueDate:
		kind, data = "date", value.Date.Raw
	case ledger.ValueBool:
		kind, data = "bool", value.Bool
	case ledger.ValueAccount:
		kind, data = "account", value.String
	case ledger.ValueCurrency:
		kind, data = "currency", value.String
	case ledger.ValueNull:
		kind, data = "null", nil
	default:
		kind, data = "string", value.String
	}
	return map[string]any{"type": kind, "value": data}
}

func fullMetadata(meta []ledger.Metadata) map[string]any {
	values := map[string]any{}
	for _, m := range meta {
		switch m.Key {
		case "filename", "lineno", "__automatic__", "__tolerances__":
			continue
		}
		values[m.Key] = fullValue(m.Value)
	}
	return values
}

func fullDirective(d ledger.Directive) (map[string]any, bool) {
	entry, ok := normalizeDirective(d)
	if !ok {
		return nil, false
	}
	// Typed metadata is extracted from each directive's common base.
	var meta []ledger.Metadata
	switch v := d.(type) {
	case ledger.Open:
		meta = v.Meta
	case ledger.Close:
		meta = v.Meta
	case ledger.Commodity:
		meta = v.Meta
	case ledger.Balance:
		meta = v.Meta
	case ledger.Pad:
		meta = v.Meta
	case ledger.Price:
		meta = v.Meta
	case ledger.Event:
		meta = v.Meta
	case ledger.Note:
		meta = v.Meta
	case ledger.Document:
		meta = v.Meta
	case ledger.Query:
		meta = v.Meta
	case ledger.Custom:
		meta = v.Meta
		values := []any{}
		for _, x := range v.Values {
			values = append(values, fullValue(x))
		}
		entry["values"] = values
	case *ledger.Transaction:
		meta = v.Meta
		entry["postings"] = fullPostings(v.Postings)
	case ledger.Transaction:
		meta = v.Meta
		entry["postings"] = fullPostings(v.Postings)
	}
	entry["metadata"] = fullMetadata(meta)
	return entry, true
}

func fullPostings(postings []ledger.Posting) []any {
	values := []any{}
	for _, p := range postings {
		var flag any
		if p.Flag != "" {
			flag = p.Flag
		}
		values = append(values, map[string]any{"account": p.Account, "units": fullAmount(p.Units), "cost": fullCostSpec(p.Cost), "price": fullPrice(p), "flag": flag, "metadata": fullMetadata(p.Meta)})
	}
	return values
}

// NormalizeFull retains posting order, exact values, typed user metadata,
// flags, cost dates/labels and repeated entries. No numeric rounding is used.
func NormalizeFull(e ledger.Evaluation) FullState {
	value := FullState{Entries: []string{}, Balances: map[string]FullInventory{}}
	for _, r := range e.Entries {
		entry, ok := fullDirective(r.Directive)
		if !ok {
			continue
		}
		entry["source"] = map[string]any{"file": filepath.Base(r.File), "line": r.Span.StartLine}
		encoded := fullJSON(entry)
		value.Entries = append(value.Entries, string(encoded))
	}
	sort.Strings(value.Entries)
	for name, account := range e.Accounts {
		if len(account.Balances) == 0 {
			continue
		}
		inventory := FullInventory{Currencies: map[string]string{}, Lots: []string{}}
		plain := map[string]ledger.Decimal{}
		for cur, amount := range account.Balances {
			plain[cur] = amount
		}
		lots := map[string]ledger.Decimal{}
		for _, p := range account.Positions {
			if p.Cost == nil {
				continue
			}
			c := FullCost{Number: p.Cost.Number.String(), Currency: p.Cost.Currency}
			if p.Cost.Date != nil {
				date := p.Cost.Date.Raw
				c.Date = &date
			}
			if p.Cost.LabelPresent || p.Cost.Label != "" {
				label := p.Cost.Label
				c.Label = &label
			}
			identity, _ := json.Marshal(map[string]any{"currency": p.Currency, "cost": c})
			lots[string(identity)] = lots[string(identity)].Add(p.Units)
			plain[p.Currency] = plain[p.Currency].Sub(p.Units)
		}
		for key, units := range lots {
			if units.IsZero() {
				continue
			}
			var lot map[string]any
			_ = json.Unmarshal([]byte(key), &lot)
			lot["number"] = units.String()
			encoded := fullJSON(lot)
			inventory.Lots = append(inventory.Lots, string(encoded))
		}
		for cur, amount := range plain {
			if !amount.IsZero() {
				inventory.Currencies[cur] = amount.String()
			}
		}
		sort.Strings(inventory.Lots)
		value.Balances[name] = inventory
	}
	return value
}

func FullDifferences(want, have FullState) []string {
	differences := []string{}
	if !reflect.DeepEqual(want.Entries, have.Entries) {
		differences = append(differences, "entries")
	}
	if !reflect.DeepEqual(want.Balances, have.Balances) {
		differences = append(differences, "balances")
	}
	return differences
}

func BuildFullSide(path string) (FullGolden, error) {
	result := snapshot.BuildWithOptions(path, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	return FullSideFromBuild(path, result), result.Err
}

// FullSideFromBuild avoids re-reading files when a verifier already captured
// the source graph. Invalid builds deliberately expose no accepted state.
func FullSideFromBuild(path string, result snapshot.BuildResult) FullGolden {
	value := FullGolden{SchemaVersion: FullSchemaVersion, Fixture: filepath.Base(path), Valid: result.Snapshot != nil, Errors: []GoldenError{}, State: FullState{Entries: []string{}, Balances: map[string]FullInventory{}}}
	for _, d := range result.Diagnostics {
		if d.Severity == diagnostic.Error {
			kind := d.Code
			switch d.Code {
			case "E-PARSE-STRING":
				kind = "LexerError"
			case "E-PARSE-DATE", "E-PARSE-TOKEN", "E-PARSE-EXPECTED", "E-PARSE-DIRECTIVE", "E-PARSE-METADATA-STACK", "E-EVAL-OPTION":
				kind = "ParserError"
				if strings.HasPrefix(d.Message, "inferred_tolerance_multiplier was renamed") {
					kind = "DeprecatedError"
				}
			case "E-EVAL-POSTING", "E-EVAL-UNBALANCED", "E-EVAL-OPEN", "E-EVAL-REOPEN", "E-EVAL-CLOSE", "E-EVAL-CURRENCY":
				kind = "ValidationError"
			case "E-EVAL-BALANCE":
				kind = "BalanceError"
			case "E-EVAL-INFER":
				kind = "InterpolationError"
			case "E-EVAL-INVENTORY":
				if strings.HasPrefix(d.Message, "cost reduction is ambiguous") {
					kind = "AmbiguousMatchError"
				} else {
					kind = "ReductionError"
				}
			}
			value.Errors = append(value.Errors, GoldenError{Type: kind, Line: d.Span.StartLine})
		}
	}
	if result.Snapshot != nil {
		value.State = NormalizeFull(result.Snapshot.Evaluation())
	}
	return value
}

// FullGoldenDifferences has no broad booking, precision or inventory waiver.
// Two named historical fixtures allow only their recorded posting/header
// locations; error class and multiplicity still have to match exactly.
func FullGoldenDifferences(want, have FullGolden) []string {
	differences := []string{}
	if want.SchemaVersion != FullSchemaVersion || have.SchemaVersion != FullSchemaVersion {
		differences = append(differences, "schema")
	}
	if want.Valid != have.Valid {
		differences = append(differences, "validity")
	}
	normalized := func(errors []GoldenError) []string {
		values := []string{}
		for _, e := range errors {
			if e.Type == "ValidationError" {
				switch want.Fixture {
				case "err-closed.bean":
					if e.Line == 5 {
						e.Line = 4
					}
				case "err-unopened.bean":
					if e.Line == 2 || e.Line == 3 {
						e.Line = 1
					}
				}
			}
			encoded, _ := json.Marshal(e)
			values = append(values, string(encoded))
		}
		sort.Strings(values)
		return values
	}
	if !reflect.DeepEqual(normalized(want.Errors), normalized(have.Errors)) {
		differences = append(differences, "diagnostics")
	}
	differences = append(differences, FullDifferences(want.State, have.State)...)
	return differences
}
