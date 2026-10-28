// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package spreadsheet writes spreadsheet documents without third-party
// dependencies. XLSX and ODS are both ZIP archives of XML parts, so the
// standard library's archive/zip and encoding/xml are enough; keeping the
// writer here means the release binary still needs only Go.
//
// The surface is deliberately small: a document is an ordered list of sheets,
// and a sheet is an ordered list of rows of typed cells. That is exactly what
// a query result needs, and it keeps the two format writers independent of
// any ledger or HTTP type.
package spreadsheet

import (
	"strings"
)

// CellKind is the data type of one cell. The type decides how the value is
// encoded in each format, so a decimal stays numeric in Excel and a date stays
// a date instead of an opaque string.
type CellKind string

const (
	// CellString is text; it is never interpreted as a number.
	CellString CellKind = "string"
	// CellNumber is a machine-parsable number, kept as its exact text.
	CellNumber CellKind = "number"
	// CellDate is a calendar date rendered as an ISO-8601 string.
	CellDate CellKind = "date"
	// CellEmpty is an absent value (SQL NULL). It has no text and produces an
	// empty cell rather than an empty string.
	CellEmpty CellKind = "empty"
)

// Cell is one table cell.
type Cell struct {
	Text string
	Kind CellKind
}

// StringCell builds a text cell.
func StringCell(text string) Cell { return Cell{Text: text, Kind: CellString} }

// NumberCell builds a numeric cell from its exact decimal text.
func NumberCell(text string) Cell { return Cell{Text: text, Kind: CellNumber} }

// DateCell builds a date cell from an ISO-8601 date.
func DateCell(text string) Cell { return Cell{Text: text, Kind: CellDate} }

// EmptyCell builds an absent value cell.
func EmptyCell() Cell { return Cell{Kind: CellEmpty} }

// Sheet is one named worksheet.
type Sheet struct {
	Name string
	Rows [][]Cell
}

// normalizeSheetName returns a format-safe sheet name. Spreadsheet sheet names
// cannot be empty, longer than 31 characters, or contain []:*?/, and both
// formats reject a duplicate name; the fallback keeps the document valid
// instead of failing the whole export.
func normalizeSheetName(name string, index int) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "Sheet" + itoa(index+1)
	}
	trimmed = strings.Map(func(r rune) rune {
		switch r {
		case '[', ']', ':', '*', '?', '/', '\\':
			return '_'
		}
		return r
	}, trimmed)
	if len(trimmed) > 31 {
		trimmed = trimmed[:31]
	}
	return trimmed
}

// uniqueSheetName returns a format-safe, document-unique sheet name. Both
// formats reject a duplicate tab name, and a query export repeats the same
// header names, so the suffix is added before the name is truncated.
func uniqueSheetName(name string, index int, used map[string]bool) string {
	base := normalizeSheetName(name, index)
	candidate := base
	for suffix := 2; used[strings.ToLower(candidate)]; suffix++ {
		trimmed := base
		tail := " (" + itoa(suffix) + ")"
		if len(trimmed)+len(tail) > 31 {
			trimmed = trimmed[:31-len(tail)]
		}
		candidate = trimmed + tail
	}
	used[strings.ToLower(candidate)] = true
	return candidate
}

// itoa is a tiny positive-int formatter; the package avoids pulling strconv
// into two files for one call site.
func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 4)
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

// columnName renders a zero-based column index as a spreadsheet column label
// (A, B, …, Z, AA, AB, …).
func columnName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+index%26)) + name
		index = index/26 - 1
	}
	return name
}

// cellRef is the A1-style reference of one cell.
func cellRef(row, column int) string {
	return columnName(column) + itoa(row+1)
}
