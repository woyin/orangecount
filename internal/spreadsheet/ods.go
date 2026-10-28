// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package spreadsheet

import (
	"archive/zip"
	"fmt"
	"io"
	"strings"
)

// ODS is an ODF package: a ZIP whose first entry is the uncompressed
// `mimetype` string, followed by content.xml, styles.xml and the manifest. The
// mimetype entry must be first and stored (not deflated) or readers fall back
// to sniffing, so it is written before the archive body starts.
const (
	odsMimetype   = "application/vnd.oasis.opendocument.spreadsheet"
	odsOfficeNS   = "urn:oasis:names:tc:opendocument:xmlns:office:1.0"
	odsTableNS    = "urn:oasis:names:tc:opendocument:xmlns:table:1.0"
	odsTextNS     = "urn:oasis:names:tc:opendocument:xmlns:text:1.0"
	odsManifestNS = "urn:oasis:names:tc:opendocument:xmlns:manifest:1.0"
	odsStyleNS    = "urn:oasis:names:tc:opendocument:xmlns:style:1.0"
	odsFoNS       = "urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0"
	odsVersion    = "1.3"
	odsXMLDeclare = `<?xml version="1.0" encoding="UTF-8"?>`
)

// WriteODS writes sheets as a minimal but valid OpenDocument spreadsheet.
func WriteODS(output io.Writer, sheets []Sheet) error {
	archive := zip.NewWriter(output)
	if err := writeMimetypeEntry(archive); err != nil {
		return err
	}
	if err := writeZipEntry(archive, "content.xml", odsContent(sheets)); err != nil {
		return err
	}
	if err := writeZipEntry(archive, "styles.xml", odsStyles()); err != nil {
		return err
	}
	if err := writeZipEntry(archive, "META-INF/manifest.xml", odsManifest()); err != nil {
		return err
	}
	return archive.Close()
}

// writeMimetypeEntry stores the media type uncompressed and first in the
// archive, which is what the ODF specification requires.
func writeMimetypeEntry(archive *zip.Writer) error {
	header := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
	header.SetModTime(zipEpoch)
	entry, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.WriteString(entry, odsMimetype)
	return err
}

func odsContent(sheets []Sheet) string {
	var builder strings.Builder
	builder.WriteString(odsXMLDeclare)
	builder.WriteString(`<office:document-content`)
	builder.WriteString(` xmlns:office="` + odsOfficeNS + `"`)
	builder.WriteString(` xmlns:table="` + odsTableNS + `"`)
	builder.WriteString(` xmlns:text="` + odsTextNS + `"`)
	builder.WriteString(` office:version="` + odsVersion + `">`)
	builder.WriteString(`<office:body><office:spreadsheet>`)
	used := map[string]bool{}
	for index, sheet := range sheets {
		name := uniqueSheetName(sheet.Name, index, used)
		builder.WriteString(`<table:table table:name="` + xmlEscape(name) + `">`)
		for _, row := range sheet.Rows {
			builder.WriteString(`<table:table-row>`)
			for _, cell := range row {
				builder.WriteString(odsCell(cell))
			}
			builder.WriteString(`</table:table-row>`)
		}
		builder.WriteString(`</table:table>`)
	}
	builder.WriteString(`</office:spreadsheet></office:body></office:document-content>`)
	return builder.String()
}

// odsCell writes one cell with its office:value-type and matching typed
// attribute, so a numeric cell carries office:value and a date carries
// office:date-value instead of an opaque string.
func odsCell(cell Cell) string {
	switch cell.Kind {
	case CellEmpty:
		return `<table:table-cell/>`
	case CellNumber:
		if isNumericText(cell.Text) {
			return `<table:table-cell office:value-type="float" office:value="` + cell.Text + `"><text:p>` + xmlEscape(cell.Text) + `</text:p></table:table-cell>`
		}
		return odsStringCell(cell.Text)
	case CellDate:
		if isISODate(cell.Text) {
			return `<table:table-cell office:value-type="date" office:date-value="` + cell.Text + `"><text:p>` + xmlEscape(cell.Text) + `</text:p></table:table-cell>`
		}
		return odsStringCell(cell.Text)
	default:
		return odsStringCell(cell.Text)
	}
}

func odsStringCell(text string) string {
	if text == "" {
		return `<table:table-cell office:value-type="string"><text:p/></table:table-cell>`
	}
	return `<table:table-cell office:value-type="string"><text:p>` + xmlEscape(text) + `</text:p></table:table-cell>`
}

// isISODate reports whether text is a calendar date in YYYY-MM-DD form, the
// only shape office:date-value accepts.
func isISODate(text string) bool {
	if len(text) != 10 || text[4] != '-' || text[7] != '-' {
		return false
	}
	for index, char := range text {
		if index == 4 || index == 7 {
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func odsStyles() string {
	return odsXMLDeclare + `<office:document-styles` +
		` xmlns:office="` + odsOfficeNS + `"` +
		` xmlns:style="` + odsStyleNS + `"` +
		` xmlns:fo="` + odsFoNS + `"` +
		` office:version="` + odsVersion + `">` +
		`<office:styles><style:default-style style:family="table-cell"><style:text-properties fo:font-size="11pt"/></style:default-style></office:styles>` +
		`</office:document-styles>`
}

func odsManifest() string {
	return odsXMLDeclare + `<manifest:manifest xmlns:manifest="` + odsManifestNS + `" manifest:version="` + odsVersion + `">` +
		fmt.Sprintf(`<manifest:file-entry manifest:full-path="/" manifest:media-type="%s" manifest:version="%s"/>`, odsMimetype, odsVersion) +
		`<manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>` +
		`<manifest:file-entry manifest:full-path="styles.xml" manifest:media-type="text/xml"/>` +
		`<manifest:file-entry manifest:full-path="META-INF/manifest.xml" manifest:media-type="text/xml"/>` +
		`</manifest:manifest>`
}
