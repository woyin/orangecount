// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orangecount/internal/snapshot"
)

const exportQuery = "SELECT account, balance FROM accounts ORDER BY account"

func newExportServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	ledgerText := "2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 12.34 USD\n  Equity:Opening -12.34 USD\n"
	if err := os.WriteFile(entry, []byte(ledgerText), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("snapshot unavailable: %+v", built.Diagnostics)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func exportRequest(t *testing.T, server *Server, format string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/query?q="+url.QueryEscape(exportQuery)+"&format="+format, nil))
	return recorder
}

// zipParts decodes every entry of one generated document.
func zipParts(t *testing.T, data []byte) map[string]string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("archive is not a valid ZIP: %v", err)
	}
	parts := map[string]string{}
	for _, file := range reader.File {
		opened, err := file.Open()
		if err != nil {
			t.Fatalf("entry %q cannot be opened: %v", file.Name, err)
		}
		body, err := io.ReadAll(opened)
		_ = opened.Close()
		if err != nil {
			t.Fatalf("entry %q cannot be read: %v", file.Name, err)
		}
		parts[file.Name] = string(body)
	}
	return parts
}

// xlsxRows decodes the first worksheet into row/cell text pairs.
func xlsxRows(t *testing.T, sheet string) [][]string {
	t.Helper()
	type cell struct {
		Type   string `xml:"t,attr"`
		Value  string `xml:"v"`
		Inline struct {
			Text string `xml:"t"`
		} `xml:"is"`
	}
	type row struct {
		Cells []cell `xml:"c"`
	}
	var document struct {
		Rows []row `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal([]byte(sheet), &document); err != nil {
		t.Fatalf("worksheet is not well-formed: %v", err)
	}
	rows := make([][]string, 0, len(document.Rows))
	for _, entry := range document.Rows {
		values := make([]string, 0, len(entry.Cells))
		for _, item := range entry.Cells {
			if item.Type == "inlineStr" {
				values = append(values, item.Inline.Text)
				continue
			}
			values = append(values, item.Value)
		}
		rows = append(rows, values)
	}
	return rows
}

// odsRows decodes the first table into row/cell text pairs.
func odsRows(t *testing.T, content string) [][]string {
	t.Helper()
	type cell struct {
		ValueType string `xml:"value-type,attr"`
		Value     string `xml:"value,attr"`
		Paragraph []struct {
			Text string `xml:",chardata"`
		} `xml:"p"`
	}
	type row struct {
		Cells []cell `xml:"table-cell"`
	}
	type table struct {
		Name string `xml:"name,attr"`
		Rows []row  `xml:"table-row"`
	}
	var document struct {
		Tables []table `xml:"body>spreadsheet>table"`
	}
	if err := xml.Unmarshal([]byte(content), &document); err != nil {
		t.Fatalf("content.xml is not well-formed: %v", err)
	}
	if len(document.Tables) == 0 {
		t.Fatal("content.xml declares no table")
	}
	rows := make([][]string, 0, len(document.Tables[0].Rows))
	for _, entry := range document.Tables[0].Rows {
		values := make([]string, 0, len(entry.Cells))
		for _, item := range entry.Cells {
			text := ""
			for _, paragraph := range item.Paragraph {
				text += paragraph.Text
			}
			if text == "" {
				text = item.Value
			}
			values = append(values, text)
		}
		rows = append(rows, values)
	}
	return rows
}

func TestQueryExportFormatsCarryTheSameValues(t *testing.T) {
	server := newExportServer(t)

	csvResponse := exportRequest(t, server, "csv")
	if csvResponse.Code != http.StatusOK || !strings.HasPrefix(csvResponse.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("csv status=%d content-type=%q", csvResponse.Code, csvResponse.Header().Get("Content-Type"))
	}
	csvRows, err := csv.NewReader(strings.NewReader(csvResponse.Body.String())).ReadAll()
	if err != nil {
		t.Fatalf("csv is not parseable: %v", err)
	}
	if len(csvRows) < 2 {
		t.Fatalf("csv has no data rows: %v", csvRows)
	}

	xlsxResponse := exportRequest(t, server, "xlsx")
	if xlsxResponse.Code != http.StatusOK {
		t.Fatalf("xlsx status=%d body=%s", xlsxResponse.Code, xlsxResponse.Body.String())
	}
	if got := xlsxResponse.Header().Get("Content-Type"); got != xlsxContentType {
		t.Fatalf("xlsx content-type=%q", got)
	}
	if got := xlsxResponse.Header().Get("Content-Disposition"); got != `attachment; filename="query_result.xlsx"` {
		t.Fatalf("xlsx disposition=%q", got)
	}
	xlsxParts := zipParts(t, xlsxResponse.Body.Bytes())
	xlsxTable := xlsxRows(t, xlsxParts["xl/worksheets/sheet1.xml"])
	if !strings.Contains(xlsxParts["xl/worksheets/sheet2.xml"], "SELECT account, balance FROM accounts ORDER BY account") {
		t.Fatalf("xlsx Query sheet lacks the query text: %s", xlsxParts["xl/worksheets/sheet2.xml"])
	}

	odsResponse := exportRequest(t, server, "ods")
	if odsResponse.Code != http.StatusOK {
		t.Fatalf("ods status=%d body=%s", odsResponse.Code, odsResponse.Body.String())
	}
	if got := odsResponse.Header().Get("Content-Type"); got != odsContentType {
		t.Fatalf("ods content-type=%q", got)
	}
	if got := odsResponse.Header().Get("Content-Disposition"); got != `attachment; filename="query_result.ods"` {
		t.Fatalf("ods disposition=%q", got)
	}
	odsParts := zipParts(t, odsResponse.Body.Bytes())
	odsTable := odsRows(t, odsParts["content.xml"])
	if !strings.Contains(odsParts["content.xml"], "SELECT account, balance FROM accounts ORDER BY account") {
		t.Fatalf("ods Query sheet lacks the query text: %s", odsParts["content.xml"])
	}

	if len(xlsxTable) != len(csvRows) || len(odsTable) != len(csvRows) {
		t.Fatalf("row counts differ: csv=%d xlsx=%d ods=%d", len(csvRows), len(xlsxTable), len(odsTable))
	}
	for index, csvRow := range csvRows {
		for column, want := range csvRow {
			if column >= len(xlsxTable[index]) {
				t.Fatalf("row %d xlsx is missing column %d: %v", index, column, xlsxTable[index])
			}
			if column >= len(odsTable[index]) {
				t.Fatalf("row %d ods is missing column %d: %v", index, column, odsTable[index])
			}
			if xlsxTable[index][column] != want {
				t.Fatalf("row %d column %d xlsx=%q csv=%q", index, column, xlsxTable[index][column], want)
			}
			if odsTable[index][column] != want {
				t.Fatalf("row %d column %d ods=%q csv=%q", index, column, odsTable[index][column], want)
			}
		}
	}
}

func TestQueryExportRejectsUnknownFormat(t *testing.T) {
	server := newExportServer(t)
	recorder := exportRequest(t, server, "pdf")
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "xlsx") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
