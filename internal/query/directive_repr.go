// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"fmt"
	"sort"
	"strings"

	"orangecount/internal/ledger"
)

// The helpers below render Go values with Python's str()/repr() spelling. The
// pinned oracle hashes those exact bytes, so the spellings are part of the
// contract rather than cosmetic formatting.

// pyReprString renders a Python str repr: single quotes with escaping.
func pyReprString(value string) string {
	var builder strings.Builder
	builder.WriteByte('\'')
	for _, r := range value {
		switch r {
		case '\\':
			builder.WriteString("\\\\")
		case '\'':
			builder.WriteString("\\'")
		case '\n':
			builder.WriteString("\\n")
		case '\r':
			builder.WriteString("\\r")
		case '\t':
			builder.WriteString("\\t")
		default:
			builder.WriteRune(r)
		}
	}
	builder.WriteByte('\'')
	return builder.String()
}

// pyAmount renders an Amount the way its __str__ does: "<number> <currency>".
func pyAmount(value ledger.Amount) string {
	return value.Number.Raw + " " + value.Currency
}

// pyDecimal renders a Decimal through str(), which keeps the source quantum.
func pyDecimal(value ledger.Decimal) string { return value.String() }

// pyDate renders datetime.date(...) the way repr() does.
func pyDate(value ledger.Date) string {
	return fmt.Sprintf("datetime.date(%d, %d, %d)", value.Year, value.Month, value.Day)
}

// pyMetadata renders the directive metadata dict with Python dict ordering:
// filename, lineno, then declaration order, then the system tolerance map.
func pyMetadata(entry ledger.EntryRecord, root string, tolerances map[string]ledger.Decimal) pyField {
	var values []ledger.Metadata
	switch d := entry.Directive.(type) {
	case ledger.Open:
		values = d.Meta
	case ledger.Close:
		values = d.Meta
	case ledger.Commodity:
		values = d.Meta
	case ledger.Pad:
		values = d.Meta
	case ledger.Balance:
		values = d.Meta
	case ledger.Transaction:
		values = d.Meta
	case ledger.Note:
		values = d.Meta
	case ledger.Event:
		values = d.Meta
	case ledger.Price:
		values = d.Meta
	case ledger.Document:
		values = d.Meta
	case ledger.Query:
		values = d.Meta
	case ledger.Custom:
		values = d.Meta
	}
	var builder strings.Builder
	builder.WriteByte('{')
	first := true
	writeEntry := func(key, rendered string) {
		if !first {
			builder.WriteString(", ")
		}
		first = false
		builder.WriteString(pyReprString(key))
		builder.WriteString(": ")
		builder.WriteString(rendered)
	}
	if entry.File != "" {
		writeEntry("filename", pyReprString(relativeSource(entry.File, root)))
		writeEntry("lineno", fmt.Sprint(entry.Span.StartLine))
	}
	for _, metadata := range values {
		writeEntry(metadata.Key, pyMetadataValue(metadata.Value, root))
	}
	if len(tolerances) > 0 {
		writeEntry("__tolerances__", pyToleranceDict(tolerances))
	}
	builder.WriteByte('}')
	return scalarPy(builder.String())
}

func mustMeta(entry ledger.EntryRecord, root string, tolerances map[string]ledger.Decimal) pyField {
	return pyMetadata(entry, root, tolerances)
}

func pyToleranceDict(tolerances map[string]ledger.Decimal) string {
	currencies := make([]string, 0, len(tolerances))
	for currency := range tolerances {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	var builder strings.Builder
	builder.WriteByte('{')
	for i, currency := range currencies {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(pyReprString(currency))
		builder.WriteString(": Decimal(")
		builder.WriteString(pyReprString(tolerances[currency].String()))
		builder.WriteString(")")
	}
	builder.WriteByte('}')
	return builder.String()
}

// pyMetadataValue renders one parsed value with repr(), since dict reprs show
// their members through repr.
func pyMetadataValue(value ledger.Value, root string) string {
	switch value.Kind {
	case ledger.ValueString, ledger.ValueAccount, ledger.ValueCurrency, ledger.ValueTag, ledger.ValueLink:
		return pyReprString(value.String)
	case ledger.ValueNumber:
		return "Decimal(" + pyReprString(value.Number.Raw) + ")"
	case ledger.ValueBool:
		if value.Bool {
			return "True"
		}
		return "False"
	case ledger.ValueDate:
		return pyDate(value.Date)
	case ledger.ValueAmount:
		return pyAmount(value.Amount)
	case ledger.ValueNull:
		return "None"
	case ledger.ValueList:
		members := make([]string, 0, len(value.List))
		for _, child := range value.List {
			members = append(members, pyMetadataValue(child, root))
		}
		return "[" + strings.Join(members, ", ") + "]"
	case ledger.ValueMap:
		members := make([]string, 0, len(value.Map))
		for _, child := range value.Map {
			members = append(members, pyReprString(child.Key)+": "+pyMetadataValue(child.Value, root))
		}
		return "{" + strings.Join(members, ", ") + "}"
	default:
		return pyReprString(value.Raw)
	}
}

// pyPosting renders a Posting namedtuple field-by-field so the hash recursion
// matches stable_hash_namedtuple instead of hashing its repr.
func pyPosting(posting ledger.Posting, entry ledger.EntryRecord, root string) ([]pyField, error) {
	units := "None"
	if posting.Units != nil {
		units = pyAmount(*posting.Units)
	}
	cost := "None"
	if posting.Cost != nil {
		cost = pyCost(posting.Cost)
	}
	price := "None"
	if posting.Price != nil {
		price = pyAmount(posting.Price.Amount)
	}
	flag := "None"
	if posting.Flag != "" {
		flag = pyReprString(posting.Flag)
	}
	file := posting.SourceMetadata.File
	line := posting.SourceMetadata.Line
	if !posting.SourceMetadata.Present {
		file = entry.File
		line = posting.At.StartLine
	}
	meta, err := pyPostingMeta(posting, file, line, root)
	if err != nil {
		return nil, err
	}
	return []pyField{
		scalarPy(posting.Account),
		scalarPy(units),
		scalarPy(cost),
		scalarPy(price),
		scalarPy(flag),
		meta,
	}, nil
}

// pyPostingMeta renders the posting's metadata dict in upstream order.
func pyPostingMeta(posting ledger.Posting, file string, line int, root string) (pyField, error) {
	var builder strings.Builder
	builder.WriteByte('{')
	builder.WriteString(pyReprString("filename"))
	builder.WriteString(": ")
	builder.WriteString(pyReprString(relativeSource(file, root)))
	builder.WriteString(", ")
	builder.WriteString(pyReprString("lineno"))
	builder.WriteString(": ")
	builder.WriteString(fmt.Sprint(line))
	for _, metadata := range posting.Meta {
		builder.WriteString(", ")
		builder.WriteString(pyReprString(metadata.Key))
		builder.WriteString(": ")
		builder.WriteString(pyMetadataValue(metadata.Value, root))
	}
	builder.WriteByte('}')
	return scalarPy(builder.String()), nil
}

// pyCost renders a Cost tuple the way its repr does.
func pyCost(cost *ledger.CostSpec) string {
	number := "None"
	currency := "None"
	date := "None"
	label := "None"
	for _, component := range cost.Components {
		switch component.Kind {
		case ledger.ValueNumber:
			number = "Decimal(" + pyReprString(component.Number.Raw) + ")"
		case ledger.ValueAmount:
			// The evaluator records a booked cost as one amount component. Its
			// DecimalText keeps the source quantum, which the oracle hashes.
			text := component.Amount.Number.DecimalText
			if text == "" {
				text = component.Amount.Number.Raw
			}
			number = "Decimal(" + pyReprString(text) + ")"
			currency = pyReprString(component.Amount.Currency)
		case ledger.ValueCurrency:
			currency = pyReprString(component.String)
		case ledger.ValueDate:
			date = pyDate(component.Date)
		case ledger.ValueString:
			label = pyReprString(component.String)
		}
	}
	if cost.Total {
		// A total cost divides by the position units upstream; the component
		// spelling is unchanged, so the total flag stays out of the repr.
		_ = cost.Total
	}
	return "Cost(number=" + number + ", currency=" + currency + ", date=" + date + ", label=" + label + ")"
}

// pyCustomValue renders one Custom directive value.
func pyCustomValue(value ledger.Value) string {
	return "ValueType(value=" + pyMetadataValue(value, "") + ", dtype=<class " + pyCustomDtype(value) + ">)"
}

func pyCustomDtype(value ledger.Value) string {
	switch value.Kind {
	case ledger.ValueString, ledger.ValueAccount, ledger.ValueCurrency, ledger.ValueTag, ledger.ValueLink:
		return "'str'"
	case ledger.ValueNumber:
		return "'decimal.Decimal'"
	case ledger.ValueBool:
		return "'bool'"
	case ledger.ValueDate:
		return "'datetime.date'"
	case ledger.ValueAmount:
		return "'beancount.core.amount.Amount'"
	default:
		return "'str'"
	}
}
