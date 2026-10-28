// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package spreadsheet

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// XLSX is an OPC (ZIP) package of XML parts. This writer emits the smallest
// part set Excel and LibreOffice both accept: content types, package
// relationships, the workbook, its relationship file, one worksheet per sheet,
// and a minimal style table. Strings are written inline (t="inlineStr"), which
// removes the shared-string table and its index bookkeeping.
const (
	xlsxMainNS     = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	xlsxRelNS      = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	xlsxPackageNS  = "http://schemas.openxmlformats.org/package/2006/relationships"
	xlsxContentNS  = "http://schemas.openxmlformats.org/package/2006/content-types"
	xlsxSheetType  = "application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"
	xlsxStyleType  = "application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"
	xlsxBookType   = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"
	xlsxXMLDeclare = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`
)

// WriteXLSX writes sheets as a minimal but valid OOXML workbook. Sheet order
// is preserved and becomes the workbook's tab order.
func WriteXLSX(output io.Writer, sheets []Sheet) error {
	archive := zip.NewWriter(output)
	parts := []struct {
		name string
		body string
	}{
		{"[Content_Types].xml", xlsxContentTypes(sheets)},
		{"_rels/.rels", xlsxPackageRels()},
		{"xl/workbook.xml", xlsxWorkbook(sheets)},
		{"xl/_rels/workbook.xml.rels", xlsxWorkbookRels(sheets)},
		{"xl/styles.xml", xlsxStyles()},
	}
	for _, part := range parts {
		if err := writeZipEntry(archive, part.name, part.body); err != nil {
			return err
		}
	}
	for index, sheet := range sheets {
		if err := writeZipEntry(archive, fmt.Sprintf("xl/worksheets/sheet%d.xml", index+1), xlsxWorksheet(sheet)); err != nil {
			return err
		}
	}
	return archive.Close()
}

func xlsxContentTypes(sheets []Sheet) string {
	var builder strings.Builder
	builder.WriteString(xlsxXMLDeclare)
	builder.WriteString(`<Types xmlns="` + xlsxContentNS + `">`)
	builder.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	builder.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	builder.WriteString(`<Override PartName="/xl/workbook.xml" ContentType="` + xlsxBookType + `"/>`)
	builder.WriteString(`<Override PartName="/xl/styles.xml" ContentType="` + xlsxStyleType + `"/>`)
	for index := range sheets {
		builder.WriteString(fmt.Sprintf(`<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="%s"/>`, index+1, xlsxSheetType))
	}
	builder.WriteString(`</Types>`)
	return builder.String()
}

func xlsxPackageRels() string {
	return xlsxXMLDeclare + `<Relationships xmlns="` + xlsxPackageNS + `">` +
		`<Relationship Id="rId1" Type="` + xlsxRelNS + `/officeDocument" Target="xl/workbook.xml"/>` +
		`</Relationships>`
}

func xlsxWorkbook(sheets []Sheet) string {
	var builder strings.Builder
	builder.WriteString(xlsxXMLDeclare)
	builder.WriteString(`<workbook xmlns="` + xlsxMainNS + `" xmlns:r="` + xlsxRelNS + `"><sheets>`)
	used := map[string]bool{}
	for index, sheet := range sheets {
		name := uniqueSheetName(sheet.Name, index, used)
		builder.WriteString(fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, xmlEscape(name), index+1, index+1))
	}
	builder.WriteString(`</sheets></workbook>`)
	return builder.String()
}

func xlsxWorkbookRels(sheets []Sheet) string {
	var builder strings.Builder
	builder.WriteString(xlsxXMLDeclare)
	builder.WriteString(`<Relationships xmlns="` + xlsxPackageNS + `">`)
	for index := range sheets {
		builder.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="`+xlsxRelNS+`/worksheet" Target="worksheets/sheet%d.xml"/>`, index+1, index+1))
	}
	builder.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="`+xlsxRelNS+`/styles" Target="styles.xml"/>`, len(sheets)+1))
	builder.WriteString(`</Relationships>`)
	return builder.String()
}

// xlsxStyles is the smallest style table that still satisfies readers: one
// font, two fills, one border, and one cell format record. No cell in this
// writer sets an explicit style index, so the part exists only because some
// consumers refuse a workbook without it.
func xlsxStyles() string {
	return xlsxXMLDeclare + `<styleSheet xmlns="` + xlsxMainNS + `">` +
		`<fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts>` +
		`<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>` +
		`<borders count="1"><border/></borders>` +
		`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
		`<cellXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/></cellXfs>` +
		`</styleSheet>`
}

func xlsxWorksheet(sheet Sheet) string {
	var builder strings.Builder
	builder.WriteString(xlsxXMLDeclare)
	builder.WriteString(`<worksheet xmlns="` + xlsxMainNS + `"><sheetData>`)
	for rowIndex, row := range sheet.Rows {
		builder.WriteString(fmt.Sprintf(`<row r="%d">`, rowIndex+1))
		for columnIndex, cell := range row {
			builder.WriteString(xlsxCell(cellRef(rowIndex, columnIndex), cell))
		}
		builder.WriteString(`</row>`)
	}
	builder.WriteString(`</sheetData></worksheet>`)
	return builder.String()
}

// xlsxCell writes one cell. A number is stored numerically only when its exact
// text parses as a plain decimal; a rational such as 1/3 stays text so the
// workbook never carries a value a spreadsheet would reinterpret or drop.
func xlsxCell(reference string, cell Cell) string {
	switch cell.Kind {
	case CellEmpty:
		return ""
	case CellNumber:
		if isNumericText(cell.Text) {
			return fmt.Sprintf(`<c r="%s"><v>%s</v></c>`, reference, cell.Text)
		}
		return xlsxInlineString(reference, cell.Text)
	case CellDate, CellString:
		return xlsxInlineString(reference, cell.Text)
	default:
		return xlsxInlineString(reference, cell.Text)
	}
}

func xlsxInlineString(reference, text string) string {
	if text == "" {
		return fmt.Sprintf(`<c r="%s"/>`, reference)
	}
	return fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, reference, xmlEscape(text))
}

// isNumericText reports whether text is a plain decimal literal a spreadsheet
// can store as a number. Rational forms, thousand separators, and padding are
// rejected so the exported cell matches the exact text shown elsewhere.
func isNumericText(text string) bool {
	if text == "" || strings.TrimSpace(text) != text {
		return false
	}
	if strings.ContainsAny(text, "/, ") {
		return false
	}
	_, err := strconv.ParseFloat(text, 64)
	return err == nil
}

// xmlEscape escapes one text value for an XML attribute or text node.
func xmlEscape(text string) string {
	var buffer bytes.Buffer
	if err := xml.EscapeText(&buffer, []byte(text)); err != nil {
		return ""
	}
	return buffer.String()
}
