// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package query

import (
	"encoding/json"
	"math/big"
	"orangecount/internal/ledger"
	"sort"
)

// Compound wire values recursively retain a child's actual type, including
// NULL versus empty collections. Dictionaries always have string keys.
type dictionaryValue map[string]any
type listValue []any
type stringSet []string
type recordValue struct {
	kind   string
	fields dictionaryValue
}
type PresentedDictionary map[string]Cell

// An ordered entry array preserves distinct Unicode spellings on Swift, whose
// native Dictionary considers canonically equivalent Strings equal.
type DictionaryEntry struct {
	Key   string `json:"key"`
	Value Cell   `json:"value"`
}

func (value PresentedDictionary) MarshalJSON() ([]byte, error) {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	entries := make([]DictionaryEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, DictionaryEntry{key, value[key]})
	}
	return json.Marshal(entries)
}
func (value stringSet) MarshalJSON() ([]byte, error) {
	values := append([]string{}, value...)
	sort.Strings(values)
	result := values[:0]
	for _, v := range values {
		if len(result) == 0 || result[len(result)-1] != v {
			result = append(result, v)
		}
	}
	return json.Marshal(result)
}

type PresentedList []Cell

func collectionCell(kind string, value any, depth int, budget *executionBudget) (Cell, error) {
	switch kind {
	case "dictionary", "open", "close", "transaction":
		var fields dictionaryValue
		if kind == "dictionary" {
			fields, _ = value.(dictionaryValue)
		} else {
			record, ok := value.(recordValue)
			if !ok || record.kind != kind {
				return Cell{}, typedFailure("type-mismatch", "invalid %s value", kind)
			}
			fields = record.fields
		}
		if fields == nil {
			return Cell{}, typedFailure("type-mismatch", "invalid dictionary value")
		}
		if len(fields) > 10000 {
			return Cell{}, typedFailure("result-budget-exceeded", "Dictionary exceeds 10000 fields")
		}
		result := PresentedDictionary{}
		for key, child := range fields {
			if len(key) > 65536 {
				return Cell{}, typedFailure("result-budget-exceeded", "Dictionary key exceeds byte budget")
			}
			cell, err := encodeTypedCell("object", child, depth+1, budget)
			if err != nil {
				return Cell{}, err
			}
			result[key] = cell
		}
		return Cell{Type: kind, Value: result}, nil
	case "set<string>":
		if list, ok := value.(listValue); ok {
			return collectionCell("list", list, depth, budget)
		}
		source, ok := value.(stringSet)
		if !ok || len(source) > 10000 {
			return Cell{}, typedFailure("type-mismatch", "invalid or oversized string set")
		}
		values := append([]string{}, source...)
		sort.Strings(values)
		unique := values[:0]
		for _, v := range values {
			if len(v) > 65536 {
				return Cell{}, typedFailure("result-budget-exceeded", "Set element exceeds byte budget")
			}
			if len(unique) == 0 || unique[len(unique)-1] != v {
				unique = append(unique, v)
			}
		}
		return Cell{Type: kind, Value: unique}, nil
	case "list":
		values, ok := value.(listValue)
		if !ok || len(values) > 10000 {
			return Cell{}, typedFailure("result-budget-exceeded", "Invalid or oversized list")
		}
		result := PresentedList{}
		for _, v := range values {
			cell, err := encodeTypedCell("object", v, depth+1, budget)
			if err != nil {
				return Cell{}, err
			}
			result = append(result, cell)
		}
		return Cell{Type: kind, Value: result}, nil
	}
	return Cell{}, typedFailure("type-mismatch", "invalid collection type")
}
func metadataCollection(value ledger.Value) (any, error) {
	switch value.Kind {
	case ledger.ValueNull:
		return nil, nil
	case ledger.ValueString, ledger.ValueAccount, ledger.ValueCurrency, ledger.ValueTag, ledger.ValueLink:
		return value.String, nil
	case ledger.ValueBool:
		return value.Bool, nil
	case ledger.ValueDate:
		return value.Date, nil
	case ledger.ValueNumber:
		return metadataDecimalValue(value.Number), nil
	case ledger.ValueAmount:
		n := metadataDecimalValue(value.Amount.Number)
		return amountValue{Number: n.Value, Currency: value.Amount.Currency, Text: n.Text}, nil
	case ledger.ValueList:
		list := listValue{}
		for _, child := range value.List {
			v, err := metadataCollection(child)
			if err != nil {
				return nil, err
			}
			list = append(list, v)
		}
		return list, nil
	case ledger.ValueMap:
		dict := dictionaryValue{}
		for _, child := range value.Map {
			v, err := metadataCollection(child.Value)
			if err != nil {
				return nil, err
			}
			dict[child.Key] = v
		}
		return dict, nil
	}
	return nil, typedFailure("unsupported-capability", "unsupported metadata value kind %s", value.Kind)
}
func metadataDictionary(scope queryMetadata) (dictionaryValue, error) {
	values := dictionaryValue{}
	if scope.Filename != "" {
		values["filename"] = scope.Filename
		values["lineno"] = metadataInteger{dateInteger(scope.Line)}
	}
	for _, metadata := range scope.Values {
		value, err := metadataCollection(metadata.Value)
		if err != nil {
			return nil, err
		}
		values[metadata.Key] = value
	}
	if scope.HasTolerances {
		tolerances := dictionaryValue{}
		for currency, amount := range scope.Tolerances {
			tolerances[currency] = amount
		}
		values["__tolerances__"] = tolerances
	}
	return values, nil
}

func dateInteger(value int) ledger.Decimal {
	return ledger.NewDecimal(new(big.Rat).SetInt64(int64(value)))
}

// ValidRecord checks the declared accessible structure, including required
// nullable fields. It does not reinterpret a captured record as a live ledger.
func ValidRecord(kind string, fields PresentedDictionary) bool {
	schemas := map[string]map[string]sourceColumn{
		"open":        {"meta": {"dictionary", false}, "date": {"date", false}, "account": {"string", false}, "currencies": {"list", true}, "booking": {"string", true}},
		"close":       {"meta": {"dictionary", false}, "date": {"date", false}, "account": {"string", false}},
		"transaction": {"meta": {"dictionary", false}, "date": {"date", false}, "flag": {"string", false}, "payee": {"string", true}, "narration": {"string", false}, "tags": {"set<string>", false}, "links": {"set<string>", false}, "accounts": {"set<string>", false}},
	}
	schema, ok := schemas[kind]
	if !ok || len(schema) != len(fields) {
		return false
	}
	for key, column := range schema {
		cell, exists := fields[key]
		if !exists {
			return false
		}
		if cell.Type == "null" {
			if !column.nullable {
				return false
			}
		} else if cell.Type != column.kind {
			return false
		}
	}
	return true
}
