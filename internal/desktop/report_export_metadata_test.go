// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"orangecount/internal/analysis"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
)

func TestJournalAnalysisRetainsTypedMetadataWithoutSourcePaths(t *testing.T) {
	path, err := filepath.Abs("../../testdata/fixtures/query/metadata-source.bean")
	if err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, err := session.Poll()
	if err != nil || !state.Valid || !state.HasSnapshot {
		t.Fatalf("open metadata fixture: valid=%t snapshot=%t err=%v", state.Valid, state.HasSnapshot, err)
	}
	response, err := session.Read(desktop.Request{View: "journal", JournalMode: "transactions", Limit: 100, SnapshotID: state.SnapshotID})
	if err != nil || response.Journal == nil || response.ReportExport == nil || len(response.Journal.Transactions) != 1 {
		t.Fatalf("read metadata journal: transactions=%d capture=%t err=%v", func() int {
			if response.Journal == nil {
				return 0
			}
			return len(response.Journal.Transactions)
		}(), response.ReportExport != nil, err)
	}

	request := desktop.ReportExportRequest{
		ProjectionID: response.ReportExport.ProjectionID,
		SnapshotID:   state.SnapshotID,
		Format:       "analysis",
		CaptureScope: response.ReportExport.CaptureScope,
	}
	capture, err := session.CaptureReportExport(request)
	if err != nil {
		t.Fatal(err)
	}
	document, err := analysis.ReadReport(bytes.NewReader(capture.Content), int64(len(capture.Content)))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(capture.Content, []byte(path)) {
		t.Fatal("analysis capture exposed an absolute metadata source path")
	}
	transactionMetadataColumn, postingMetadataColumn := -1, -1
	for i, column := range document.Schema.Columns {
		switch column.ID {
		case "transaction_metadata":
			transactionMetadataColumn = i
		case "posting_metadata":
			postingMetadataColumn = i
		}
	}
	if transactionMetadataColumn < 0 || postingMetadataColumn < 0 || len(document.Rows) != 2 {
		t.Fatalf("metadata columns/rows = %d/%d/%d", transactionMetadataColumn, postingMetadataColumn, len(document.Rows))
	}
	transactionMetadata := metadataEntries(t, document.Rows[0][transactionMetadataColumn])
	if got := transactionMetadata["common"]; got.kind != "string" || got.value != `{"type":"string","value":"entry"}` {
		t.Fatalf("transaction metadata was not retained: kind=%q value=%q", got.kind, got.value)
	}
	postingMetadata := metadataEntries(t, document.Rows[0][postingMetadataColumn])
	if got := postingMetadata["common"]; got.kind != "null" || got.value != `{"type":"null","value":null}` {
		t.Fatalf("posting null metadata was not retained: kind=%q value=%q", got.kind, got.value)
	}
	if got := postingMetadata["switch"]; got.kind != "bool" || got.value != `{"type":"bool","value":false}` {
		t.Fatalf("posting false metadata was not retained: kind=%q value=%q", got.kind, got.value)
	}

	request.Format = "csv"
	csvCapture, err := session.CaptureReportExport(request)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(csvCapture.Content, []byte(path)) || !bytes.Contains(csvCapture.Content, []byte("Transaction metadata")) || !bytes.Contains(csvCapture.Content, []byte("Posting metadata")) || !bytes.Contains(csvCapture.Content, []byte("entry")) {
		t.Fatal("CSV did not retain typed metadata or exposed an absolute source path")
	}
}

type exportedMetadata struct {
	kind  string
	value string
}

func metadataEntries(t *testing.T, cell query.Cell) map[string]exportedMetadata {
	t.Helper()
	if cell.Type != "list<record>" {
		t.Fatalf("metadata cell type = %q", cell.Type)
	}
	rows, ok := cell.Value.([][]query.Cell)
	if !ok {
		t.Fatalf("metadata cell value has type %T", cell.Value)
	}
	entries := make(map[string]exportedMetadata, len(rows))
	for _, row := range rows {
		if len(row) != 3 {
			t.Fatalf("metadata row has %d fields", len(row))
		}
		key, keyOK := row[0].Value.(string)
		kind, kindOK := row[1].Value.(string)
		value, valueOK := row[2].Value.(string)
		if !keyOK || !kindOK || !valueOK {
			t.Fatalf("metadata row contains an unexpected type: key=%T kind=%T value=%T", row[0].Value, row[1].Value, row[2].Value)
		}
		var typed struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal([]byte(value), &typed); err != nil || typed.Type != kind {
			t.Fatalf("metadata typed JSON does not match its kind: %v", err)
		}
		entries[key] = exportedMetadata{kind: kind, value: value}
	}
	return entries
}
