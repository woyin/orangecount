// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"fmt"
	"net/http"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/presentation"
	"orangecount/internal/query"
	"orangecount/internal/spreadsheet"
)

// The spreadsheet exports mirror Fava's download-query documents: one
// Results sheet with the column headers and rows, and one Query sheet holding
// the query text. Fava delegates this to pyexcel; here the workbook is written
// by internal/spreadsheet, which keeps the release binary dependency-free.
const (
	xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	odsContentType  = "application/vnd.oasis.opendocument.spreadsheet"
)

// querySheets builds the two-sheet document for one query result.
func querySheets(result query.Result, queryText string) []spreadsheet.Sheet {
	rows := make([][]spreadsheet.Cell, 0, len(result.Rows)+1)
	header := make([]spreadsheet.Cell, len(result.Columns))
	for index, column := range result.Columns {
		header[index] = spreadsheet.StringCell(column)
	}
	rows = append(rows, header)
	for _, row := range result.Rows {
		cells := make([]spreadsheet.Cell, len(result.Columns))
		for index, column := range result.Columns {
			cells[index] = spreadsheetCell(row[column])
		}
		rows = append(rows, cells)
	}
	return []spreadsheet.Sheet{
		{Name: "Results", Rows: rows},
		{Name: "Query", Rows: [][]spreadsheet.Cell{{spreadsheet.StringCell("Query")}, {spreadsheet.StringCell(queryText)}}},
	}
}

// spreadsheetCell maps one query value onto a typed cell. The text of a cell is
// the same text the CSV export writes, so all three formats carry identical
// values; only the cell type differs.
func spreadsheetCell(value any) spreadsheet.Cell {
	switch typed := value.(type) {
	case nil:
		return spreadsheet.EmptyCell()
	case ledger.Decimal:
		return spreadsheet.NumberCell(typed.String())
	case presentation.Decimal:
		return spreadsheet.NumberCell(typed.Exact)
	case ledger.Date:
		return spreadsheet.DateCell(typed.Raw)
	case ledger.DirectiveKind:
		return spreadsheet.StringCell(string(typed))
	case []string:
		return spreadsheet.StringCell(strings.Join(typed, ","))
	case bool:
		return spreadsheet.StringCell(fmt.Sprintf("%t", typed))
	case string:
		return spreadsheet.StringCell(typed)
	case int:
		return spreadsheet.NumberCell(fmt.Sprintf("%d", typed))
	default:
		return spreadsheet.StringCell(fmt.Sprint(typed))
	}
}

// writeQuerySpreadsheet streams one query result as XLSX or ODS with the
// download headers Fava uses for its spreadsheet exports.
func writeQuerySpreadsheet(w http.ResponseWriter, format string, result query.Result, queryText string) {
	sheets := querySheets(result, queryText)
	switch format {
	case "xlsx":
		w.Header().Set("Content-Type", xlsxContentType)
		w.Header().Set("Content-Disposition", `attachment; filename="query_result.xlsx"`)
		_ = spreadsheet.WriteXLSX(w, sheets)
	case "ods":
		w.Header().Set("Content-Type", odsContentType)
		w.Header().Set("Content-Disposition", `attachment; filename="query_result.ods"`)
		_ = spreadsheet.WriteODS(w, sheets)
	}
}
