// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package spreadsheet

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func sampleSheets() []Sheet {
	return []Sheet{
		{Name: "Results", Rows: [][]Cell{
			{StringCell("account"), StringCell("balance")},
			{StringCell("Assets:Bank"), NumberCell("12.34")},
			{StringCell("Expenses:Food"), NumberCell("1/3")},
			{StringCell("note"), EmptyCell()},
		}},
		{Name: "Query", Rows: [][]Cell{
			{StringCell("Query")},
			{StringCell("SELECT account, balance FROM accounts")},
		}},
	}
}

// readZipParts returns every entry name in archive order and the decoded body.
func readZipParts(t *testing.T, data []byte) ([]string, map[string]string) {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("archive is not a valid ZIP: %v", err)
	}
	names := make([]string, 0, len(reader.File))
	bodies := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
		opened, err := file.Open()
		if err != nil {
			t.Fatalf("entry %q cannot be opened: %v", file.Name, err)
		}
		body, err := io.ReadAll(opened)
		_ = opened.Close()
		if err != nil {
			t.Fatalf("entry %q cannot be read: %v", file.Name, err)
		}
		bodies[file.Name] = string(body)
	}
	return names, bodies
}

func TestWriteXLSXEmitsRequiredParts(t *testing.T) {
	var buffer bytes.Buffer
	if err := WriteXLSX(&buffer, sampleSheets()); err != nil {
		t.Fatal(err)
	}
	names, bodies := readZipParts(t, buffer.Bytes())
	for _, required := range []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"xl/workbook.xml",
		"xl/_rels/workbook.xml.rels",
		"xl/styles.xml",
		"xl/worksheets/sheet1.xml",
		"xl/worksheets/sheet2.xml",
	} {
		if _, ok := bodies[required]; !ok {
			t.Fatalf("required part %q missing; parts=%v", required, names)
		}
	}
	for _, name := range names {
		if strings.HasSuffix(name, ".xml") || strings.HasSuffix(name, ".rels") {
			var document any
			if err := xml.Unmarshal([]byte(bodies[name]), &document); err != nil {
				t.Fatalf("part %q is not well-formed XML: %v", name, err)
			}
		}
	}
	if !strings.Contains(bodies["xl/workbook.xml"], `name="Results"`) || !strings.Contains(bodies["xl/workbook.xml"], `name="Query"`) {
		t.Fatalf("workbook does not declare both sheets: %s", bodies["xl/workbook.xml"])
	}
	results := bodies["xl/worksheets/sheet1.xml"]
	if !strings.Contains(results, "Assets:Bank") || !strings.Contains(results, "<v>12.34</v>") {
		t.Fatalf("sheet1 lacks the typed cells: %s", results)
	}
	// A rational is not a spreadsheet number, so it stays text and keeps its
	// exact spelling instead of being rounded.
	if !strings.Contains(results, "1/3") {
		t.Fatalf("sheet1 dropped the exact rational text: %s", results)
	}
	// A NULL value is an absent cell, which is how OOXML spells an empty
	// cell: row 4 ends after A4.
	if !strings.Contains(results, `<c r="A4" t="inlineStr"><is><t xml:space="preserve">note</t></is></c></row>`) {
		t.Fatalf("sheet1 does not leave the NULL cell empty: %s", results)
	}
}

func TestWriteODSStoresMimetypeFirst(t *testing.T) {
	var buffer bytes.Buffer
	if err := WriteODS(&buffer, sampleSheets()); err != nil {
		t.Fatal(err)
	}
	names, bodies := readZipParts(t, buffer.Bytes())
	if len(names) == 0 || names[0] != "mimetype" {
		t.Fatalf("first entry is %v, want mimetype first", names)
	}
	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if reader.File[0].Method != zip.Store {
		t.Fatalf("mimetype method=%d, want zip.Store", reader.File[0].Method)
	}
	if bodies["mimetype"] != "application/vnd.oasis.opendocument.spreadsheet" {
		t.Fatalf("mimetype=%q", bodies["mimetype"])
	}
	for _, required := range []string{"content.xml", "styles.xml", "META-INF/manifest.xml"} {
		if _, ok := bodies[required]; !ok {
			t.Fatalf("required part %q missing; parts=%v", required, names)
		}
	}
	for _, name := range []string{"content.xml", "styles.xml", "META-INF/manifest.xml"} {
		var document any
		if err := xml.Unmarshal([]byte(bodies[name]), &document); err != nil {
			t.Fatalf("part %q is not well-formed XML: %v", name, err)
		}
	}
	content := bodies["content.xml"]
	if !strings.Contains(content, `table:name="Results"`) || !strings.Contains(content, `table:name="Query"`) {
		t.Fatalf("content.xml does not declare both sheets: %s", content)
	}
	if !strings.Contains(content, `office:value-type="float" office:value="12.34"`) {
		t.Fatalf("content.xml lost the numeric cell: %s", content)
	}
	if !strings.Contains(content, `office:value-type="string"`) || !strings.Contains(content, "SELECT account, balance FROM accounts") {
		t.Fatalf("content.xml lost the query sheet: %s", content)
	}
	if !strings.Contains(content, "<table:table-cell/>") {
		t.Fatalf("content.xml does not encode the NULL cell as an empty cell: %s", content)
	}
	if !strings.Contains(bodies["META-INF/manifest.xml"], "application/vnd.oasis.opendocument.spreadsheet") {
		t.Fatalf("manifest lacks the media type: %s", bodies["META-INF/manifest.xml"])
	}
}

func TestWriteXLSXAndODSAreDeterministic(t *testing.T) {
	for _, write := range []func(io.Writer, []Sheet) error{WriteXLSX, WriteODS} {
		var first, second bytes.Buffer
		if err := write(&first, sampleSheets()); err != nil {
			t.Fatal(err)
		}
		if err := write(&second, sampleSheets()); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first.Bytes(), second.Bytes()) {
			t.Fatalf("repeated exports differ: %d vs %d bytes", first.Len(), second.Len())
		}
	}
}

func TestSheetNamesStayValidAndUnique(t *testing.T) {
	used := map[string]bool{}
	if got := uniqueSheetName("", 0, used); got == "" {
		t.Fatal("empty name was not replaced")
	}
	long := strings.Repeat("x", 40)
	if got := uniqueSheetName(long, 0, used); len(got) > 31 {
		t.Fatalf("name %q is longer than 31 characters", got)
	}
	first := uniqueSheetName("Results", 0, used)
	second := uniqueSheetName("Results", 1, used)
	if first == second {
		t.Fatalf("duplicate sheet names %q", first)
	}
	if strings.ContainsAny(uniqueSheetName("a/b:c", 0, map[string]bool{}), "/:") {
		t.Fatal("invalid sheet name characters survived")
	}
}

func TestColumnReferences(t *testing.T) {
	cases := map[int]string{0: "A", 25: "Z", 26: "AA", 27: "AB", 51: "AZ", 52: "BA"}
	for index, want := range cases {
		if got := columnName(index); got != want {
			t.Fatalf("columnName(%d)=%q want %q", index, got, want)
		}
	}
	if got := cellRef(0, 0); got != "A1" {
		t.Fatalf("cellRef(0,0)=%q", got)
	}
	if got := cellRef(9, 27); got != "AB10" {
		t.Fatalf("cellRef(9,27)=%q", got)
	}
}
