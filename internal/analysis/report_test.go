// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package analysis_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"orangecount/internal/analysis"
	"orangecount/internal/query"
	"os"
	"strings"
	"testing"
)

// The fixture states its expected numbers independently of the production writer.
func publicArchive(t *testing.T, completeness string, rows string, csv string, page string) []byte {
	t.Helper()
	context := `{"format_version":2,"producer":"OrangeCount","contract":"native-report-analysis-v1","view":"balance_sheet","section":"tree","snapshot_id":"public-snapshot","projection_id":"public-projection","semantics_profile":"v3","evaluation_revision":"v3-core-3","normalized_request":{"date_to":"2024-01-31"},"capture_scope":"complete-projection","returned_rows":1,"source_record_count":1,"truncated":false,"completeness":"` + completeness + `","page":` + page + `,"valuation_status":null,"valuation_details":{}}`
	if rows == "" {
		context = strings.ReplaceAll(context, `"returned_rows":1,"source_record_count":1`, `"returned_rows":0,"source_record_count":0`)
	}
	schema := `{"row_unit":"account","page_unit":"account","columns":[{"id":"account","name":"Account","type":"string","nullable":false,"fields":null},{"id":"total","name":"Total","type":"decimal","nullable":true,"fields":null}]}`
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, member := range []struct{ name, data string }{{"context.json", context}, {"schema.json", schema}, {"cells.ndjson", rows}, {"data.csv", csv}} {
		f, err := w.Create(member.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write([]byte(member.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestIndependentReportFixturesAndRoundTrip(t *testing.T) {
	cases := []struct{ name, status, rows, csv string }{
		{"empty", "empty", "", "Account,Total\r\n"},
		{"exact", "complete", "[{\"type\":\"string\",\"value\":\"Assets:Cash\"},{\"type\":\"decimal\",\"value\":{\"display\":\"0.333333\",\"exact\":\"1/3\",\"approximate\":true}}]\n", "Account,Total\r\nAssets:Cash,1/3\r\n"},
		{"unknown", "partial", "[{\"type\":\"string\",\"value\":\"Assets:Cash\"},{\"type\":\"null\"}]\n", "Account,Total\r\nAssets:Cash,\r\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := publicArchive(t, tc.status, tc.rows, tc.csv, "null")
			doc, err := analysis.ReadReport(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatal(err)
			}
			if doc.Context.Completeness != tc.status {
				t.Fatal("lost status")
			}
			var out bytes.Buffer
			if err = doc.Write(&out); err != nil {
				t.Fatal(err)
			}
			again, err := analysis.ReadReport(bytes.NewReader(out.Bytes()), int64(out.Len()))
			if err != nil {
				t.Fatal(err)
			}
			a, _ := json.Marshal(doc)
			b, _ := json.Marshal(again)
			if !bytes.Equal(a, b) {
				t.Fatal("roundtrip changed typed values")
			}
		})
	}
}
func TestReportRejectsMismatchAndDuplicateMember(t *testing.T) {
	data := publicArchive(t, "empty", "", "Account,Total\n", "null")
	if _, err := analysis.ReadReport(bytes.NewReader(data), int64(len(data))); err == nil {
		t.Fatal("accepted noncanonical CSV")
	}
	data = publicArchive(t, "empty", "", "Account,Total\r\n", `{"total_count":5,"offset":2,"limit":2,"returned_count":0,"has_more":true}`)
	if _, err := analysis.ReadReport(bytes.NewReader(data), int64(len(data))); err == nil {
		t.Fatal("accepted complete projection with page")
	}
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for i := 0; i < 4; i++ {
		f, _ := w.Create("context.json")
		f.Write([]byte("{}"))
	}
	w.Close()
	if _, err := analysis.ReadReport(bytes.NewReader(b.Bytes()), int64(b.Len())); err == nil {
		t.Fatal("accepted duplicate members")
	}
}

func TestReportRejectsMalformedTypesAndContext(t *testing.T) {
	valid := publicArchive(t, "complete", "[{\"type\":\"string\",\"value\":\"Assets:Cash\"},{\"type\":\"decimal\",\"value\":{\"display\":\"1\",\"exact\":\"1\",\"approximate\":false}}]\n", "Account,Total\r\nAssets:Cash,1\r\n", "null")
	mutations := []struct{ name, member, old, new string }{
		{"version", "context.json", `"format_version":2`, `"format_version":99`},
		{"duplicate key", "context.json", `"producer":"OrangeCount"`, `"producer":"OrangeCount","producer":"duplicate"`},
		{"missing required", "context.json", `"projection_id":"public-projection",`, ``},
		{"unknown request", "context.json", `"date_to":"2024-01-31"`, `"source_path":"/private/main.bean"`},
		{"row mismatch", "context.json", `"returned_rows":1`, `"returned_rows":0`},
		{"invalid exact", "cells.ndjson", `"exact":"1"`, `"exact":"1/0"`},
		{"display mismatch", "cells.ndjson", `"display":"1"`, `"display":"2"`},
		{"wrong type", "cells.ndjson", `"type":"decimal"`, `"type":"bool"`},
		{"nullable violation", "cells.ndjson", `{"type":"string","value":"Assets:Cash"}`, `{"type":"null"}`},
		{"unknown field", "schema.json", `"fields":null`, `"fields":null,"extra":true`},
		{"duplicate column", "schema.json", `"id":"total"`, `"id":"account"`},
	}
	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			data := mutateMember(t, valid, m.member, m.old, m.new)
			if _, err := analysis.ReadReport(bytes.NewReader(data), int64(len(data))); err == nil {
				t.Fatal("accepted malformed archive")
			}
		})
	}
}
func mutateMember(t *testing.T, data []byte, name, old, new string) []byte {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, f := range z.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if f.Name == name {
			raw = []byte(strings.ReplaceAll(string(raw), old, new))
		}
		out, err := w.Create(f.Name)
		if err != nil {
			t.Fatal(err)
		}
		out.Write(raw)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestRecordSetAndInventoryRemainTyped(t *testing.T) {
	raw := `{"context":{"format_version":2,"producer":"OrangeCount","contract":"native-report-analysis-v1","view":"holdings","section":"lots","snapshot_id":"public","projection_id":"public-lot","semantics_profile":"v3","evaluation_revision":"v3-core-3","normalized_request":{},"capture_scope":"current-page","returned_rows":1,"source_record_count":1,"truncated":false,"completeness":"not-applicable","page":null,"valuation_status":null,"valuation_details":{}},"schema":{"row_unit":"lot","page_unit":"lot","columns":[{"id":"lot","name":"Lot","type":"inventory","nullable":false,"fields":null},{"id":"tags","name":"Tags","type":"set<string>","nullable":false,"fields":null},{"id":"details","name":"Details","type":"record","nullable":false,"fields":[{"id":"reason","name":"Reason","type":"string","nullable":true,"fields":null}]}]},"rows":[[{"type":"inventory","value":[{"units":{"number":{"exact":"2","display":"2","approximate":false},"currency":"HOOL"},"cost":{"number":{"exact":"0","display":"0","approximate":false},"currency":"USD","date":null,"label":""}},{"units":{"number":{"exact":"1","display":"1","approximate":false},"currency":"EUR"},"cost":null}]},{"type":"set<string>","value":["a","b"]},{"type":"record","value":[{"type":"null"}]}]]}`
	d, err := analysis.DecodeDocument([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err = d.Write(&b); err != nil {
		t.Fatal(err)
	}
	again, err := analysis.ReadReport(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err != nil {
		t.Fatal(err)
	}
	positions, ok := again.Rows[0][0].Value.([]query.PresentedPosition)
	if !ok || len(positions) != 2 || positions[0].Cost == nil || positions[0].Cost.Date != nil || positions[0].Cost.Label == nil || *positions[0].Cost.Label != "" || positions[1].Cost != nil {
		t.Fatal("lost full cost key or typed inventory")
	}
	invalid := strings.Replace(raw, `["a","b"]`, `["b","a"]`, 1)
	if _, err = analysis.DecodeDocument([]byte(invalid)); err == nil {
		t.Fatal("accepted unordered set")
	}
	invalid = strings.Replace(raw, `"fields":[{"id":"reason"`, `"fields":null,"extra":[{"id":"reason"`, 1)
	if _, err = analysis.DecodeDocument([]byte(invalid)); err == nil {
		t.Fatal("accepted malformed record schema")
	}
}

func TestPublishedV2Samples(t *testing.T) {
	for _, name := range []string{"empty", "multicurrency", "partial", "paged-postings"} {
		t.Run(name, func(t *testing.T) {
			var b bytes.Buffer
			w := zip.NewWriter(&b)
			for _, member := range []string{"context.json", "schema.json", "cells.ndjson", "data.csv"} {
				raw, err := os.ReadFile("../../testdata/fixtures/analysis-v2/" + name + "/" + member)
				if err != nil {
					t.Fatal(err)
				}
				out, err := w.Create(member)
				if err != nil {
					t.Fatal(err)
				}
				out.Write(raw)
			}
			w.Close()
			d, err := analysis.ReadReport(bytes.NewReader(b.Bytes()), int64(b.Len()))
			if err != nil {
				t.Fatal(err)
			}
			if name == "paged-postings" && (len(d.Rows) != 3 || d.Context.SourceRecordCount != 1) {
				t.Fatal("lost distinct posting/transaction units")
			}
			if name == "partial" && (d.Context.Completeness != "partial" || d.Rows[0][2].Type != "null") {
				t.Fatal("partial turned into complete")
			}
		})
	}
}

func TestReportContainerCRCAndFilesystemMembers(t *testing.T) {
	valid := publicArchive(t, "empty", "", "Account,Total\r\n", "null")
	members, err := analysis.Members(bytes.NewReader(valid), int64(len(valid)))
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []os.FileMode{0600, os.ModeSymlink | 0777} {
		var b bytes.Buffer
		w := zip.NewWriter(&b)
		for _, name := range []string{"context.json", "schema.json", "cells.ndjson", "data.csv"} {
			h := &zip.FileHeader{Name: name, Method: zip.Store}
			if name == "context.json" {
				h.SetMode(mode)
			}
			f, err := w.CreateHeader(h)
			if err != nil {
				t.Fatal(err)
			}
			f.Write(members[name])
		}
		w.Close()
		data := b.Bytes()
		if mode == 0600 {
			i := bytes.Index(data, []byte("OrangeCount"))
			if i < 0 {
				t.Fatal("missing uncompressed marker")
			}
			data[i] = 'X'
		}
		if _, err = analysis.ReadReport(bytes.NewReader(data), int64(len(data))); err == nil {
			t.Fatal("accepted bad CRC or symlink member")
		}
	}
	if _, err = analysis.ReadReport(bytes.NewReader(valid), analysis.ByteLimit+1); !errors.Is(err, analysis.ErrBudget) {
		t.Fatal("missing compressed byte budget")
	}
}

type cancelWriter struct{ cancel context.CancelFunc }

func (w cancelWriter) Write(p []byte) (int, error) { w.cancel(); return len(p), nil }
func TestReportWritingObservesActualCancellation(t *testing.T) {
	data := publicArchive(t, "empty", "", "Account,Total\r\n", "null")
	d, err := analysis.ReadReport(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err = d.WriteContext(ctx, cancelWriter{cancel}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation during actual write: %v", err)
	}
	var out bytes.Buffer
	if err = d.WriteContext(ctx, &out); !errors.Is(err, context.Canceled) || out.Len() != 0 {
		t.Fatal("pre-cancelled capture wrote bytes")
	}
}
