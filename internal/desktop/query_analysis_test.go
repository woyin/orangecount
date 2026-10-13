// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
	"os"
	"testing"
)

func publicAnalysis(t *testing.T, fixture, text string) []byte {
	t.Helper()
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/query/"+fixture, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: text})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := response.WriteAnalysis(&output); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func rewriteAnalysisMember(t *testing.T, data []byte, name string, edit func([]byte) []byte) []byte {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("read archive member %s: %v %v", file.Name, err, closeErr)
		}
		if file.Name == name {
			contents = edit(contents)
		}
		member, err := writer.Create(file.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestAnalysisReaderRestoresExactPositionalValuesAndZeroRows(t *testing.T) {
	for _, test := range []struct {
		fixture, text string
		rows          int
	}{
		{"default-source.bean", "SELECT 1 AS v, 2 AS v, decimal('1.00000000000000000001') AS precise, NULL AS n, '' AS blank LIMIT 1", 1},
		{"default-source.bean", "SELECT account, position LIMIT 0", 0},
		{"compound-casts.bean", "SELECT sum(position) WHERE account = 'Assets:Precision'", 1},
	} {
		t.Run(test.text, func(t *testing.T) {
			data := publicAnalysis(t, test.fixture, test.text)
			restored, err := desktop.ReadQueryAnalysis(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatal(err)
			}
			if restored.Context.NormalizedRequest.Text != test.text || len(restored.Result.Rows) != test.rows {
				t.Fatal("capture context or rows changed")
			}
			if test.rows == 0 {
				if len(restored.Result.Columns) != 2 {
					t.Fatal("zero-row schema lost")
				}
				return
			}
			if test.fixture == "default-source.bean" {
				row := restored.Result.Rows[0]
				if restored.Result.Columns[0].Name != "v" || restored.Result.Columns[1].Name != "v" || row[0].Value != "1" || row[1].Value != "2" || row[2].Value.(query.PresentedDecimal).Exact != "1.00000000000000000001" || row[3].Type != "null" || row[4].Value != "" {
					t.Fatal("exact typed values lost")
				}
			} else {
				positions := restored.Result.Rows[0][0].Value.([]query.PresentedPosition)
				if len(positions) != 1 || positions[0].Units.Number.Exact != "1" || positions[0].Cost == nil || positions[0].Cost.Number.Exact != "10" || positions[0].Cost.Label == nil || *positions[0].Cost.Label != "a\"b" {
					t.Fatal("inventory full cost identity lost")
				}
			}
		})
	}
}

func TestAnalysisReaderAcceptsSupportedScalarContracts(t *testing.T) {
	original := publicAnalysis(t, "default-source.bean", "SELECT account, number LIMIT 1")
	for _, contract := range []string{query.LegacyContract, query.PreviousContract, query.CurrentContract} {
		t.Run(contract, func(t *testing.T) {
			data := rewriteAnalysisMember(t, original, "context.json", func(contents []byte) []byte {
				return bytes.Replace(contents, []byte(query.CurrentContract), []byte(contract), 1)
			})
			document, err := desktop.ReadQueryAnalysis(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatal(err)
			}
			if document.Result.Contract != contract || len(document.Result.Columns) != 2 || len(document.Result.Rows) != 1 {
				t.Fatalf("contract %s did not restore its scalar result: %+v", contract, document)
			}
		})
	}
}

func TestAnalysisReaderRejectsTamperedArchives(t *testing.T) {
	original := publicAnalysis(t, "default-source.bean", "SELECT 1 AS v LIMIT 1")
	for _, scenario := range []string{"extra", "duplicate", "missing", "csv", "context", "schema", "cell", "duplicate-key", "case-alias", "unknown-contract", "future-format"} {
		t.Run(scenario, func(t *testing.T) {
			source, err := zip.NewReader(bytes.NewReader(original), int64(len(original)))
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			writer := zip.NewWriter(&output)
			for _, file := range source.File {
				if scenario == "missing" && file.Name == "data.csv" {
					continue
				}
				reader, _ := file.Open()
				data, _ := io.ReadAll(reader)
				reader.Close()
				switch {
				case scenario == "csv" && file.Name == "data.csv":
					data = []byte("v\n2\n")
				case scenario == "context" && file.Name == "context.json":
					data = bytes.Replace(data, []byte(`"returned_rows":1`), []byte(`"returned_rows":0`), 1)
				case scenario == "schema" && file.Name == "schema.json":
					data = bytes.Replace(data, []byte(`"nullable":false`), []byte(`"nullable":null`), 1)
				case scenario == "cell" && file.Name == "cells.ndjson":
					data = bytes.Replace(data, []byte(`"value":"1"`), []byte(`"value":"1e1000000"`), 1)
				case scenario == "case-alias" && file.Name == "context.json":
					data = bytes.Replace(data, []byte(`"view":"query"`), []byte(`"view":"query","VIEW":"query"`), 1)
				case scenario == "duplicate-key" && file.Name == "context.json":
					data = bytes.Replace(data, []byte(`"format_version":1`), []byte(`"format_version":1,"format_version":1`), 1)
				case scenario == "unknown-contract" && file.Name == "context.json":
					data = bytes.Replace(data, []byte(query.CurrentContract), []byte("beanquery-0.2.0-typed-v99"), 1)
				case scenario == "future-format" && file.Name == "context.json":
					data = bytes.Replace(data, []byte(`"format_version":1`), []byte(`"format_version":2`), 1)
				}
				member, _ := writer.Create(file.Name)
				member.Write(data)
				if scenario == "duplicate" && file.Name == "data.csv" {
					member, _ = writer.Create(file.Name)
					member.Write(data)
				}
			}
			if scenario == "extra" {
				member, _ := writer.Create("../source.bean")
				member.Write([]byte("unexpected"))
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := desktop.ReadQueryAnalysis(bytes.NewReader(output.Bytes()), int64(output.Len())); err == nil {
				t.Fatal("tampered archive accepted")
			}
		})
	}
}

func TestAnalysisReaderFinancialSampleGrid(t *testing.T) {
	for _, scenario := range []string{"multicurrency", "empty", "labels", "ratio"} {
		t.Run(scenario, func(t *testing.T) {
			text := "SELECT sum(position) AS inventory"
			switch scenario {
			case "empty":
				text += " WHERE account = 'Assets:Plain'"
			case "labels":
				text += " WHERE account = 'Assets:Stock'"
			case "ratio":
				text = "SELECT 1.0 / 3 AS ratio LIMIT 1"
			}
			data := publicAnalysis(t, "compound-values.bean", text)
			document, err := desktop.ReadQueryAnalysis(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatal(err)
			}
			if len(document.Result.Rows) != 1 {
				t.Fatal("expected aggregate row")
			}
			value := document.Result.Rows[0][0].Value
			if scenario == "ratio" {
				d := value.(query.PresentedDecimal)
				if d.Exact != "0.3333333333333333333333333333" || d.Display != d.Exact || d.Approximate {
					t.Fatalf("ratio lost: %#v", d)
				}
				return
			}
			positions := value.([]query.PresentedPosition)
			switch scenario {
			case "empty":
				if len(positions) != 0 {
					t.Fatal("cancelled inventory not empty")
				}
			case "multicurrency":
				currencies := map[string]bool{}
				for _, p := range positions {
					currencies[p.Units.Currency] = true
				}
				for _, c := range []string{"USD", "EUR", "STK", "ABC"} {
					if !currencies[c] {
						t.Fatal("missing currency", c)
					}
				}
			case "labels":
				var absent, empty, alpha, zero bool
				for _, p := range positions {
					if p.Cost == nil {
						t.Fatal("stock missing cost")
					}
					c := p.Cost
					absent = absent || c.Label == nil
					empty = empty || (c.Label != nil && *c.Label == "")
					alpha = alpha || (c.Label != nil && *c.Label == "alpha")
					zero = zero || c.Number.Exact == "0"
				}
				if !absent || !empty || !alpha || !zero {
					t.Fatal("cost identities collapsed")
				}
			}
		})
	}
}

func TestAnalysisReaderRationalCostAndTruncation(t *testing.T) {
	data := publicAnalysis(t, "analysis-rational.bean", "SELECT position WHERE account = 'Assets:Stock'")
	document, err := desktop.ReadQueryAnalysis(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	cost := document.Result.Rows[0][0].Value.(query.PresentedPosition).Cost
	if cost == nil || cost.Number.Exact != "1/3" || cost.Number.Display != "0.333333" || !cost.Number.Approximate {
		t.Fatal("rational total cost lost")
	}
	const text = "SELECT account"
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/query/compound-values.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: text})
	if err != nil {
		t.Fatal(err)
	}
	built := snapshot.BuildWithOptions("../../testdata/fixtures/query/compound-values.bean", snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if built.Snapshot == nil {
		t.Fatal("fixture failed")
	}
	execution, err := query.ExecuteTyped(context.Background(), text, built.Snapshot.Evaluation(), query.ExecutionLimits{Rows: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !execution.Truncated || execution.ReturnedRows != 2 {
		t.Fatal("real execution did not truncate")
	}
	response.Execution = &execution
	var output bytes.Buffer
	if err := response.WriteAnalysis(&output); err != nil {
		t.Fatal(err)
	}
	document, err = desktop.ReadQueryAnalysis(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if !document.Context.Truncated || document.Context.Completeness != "truncated" || document.Context.RowLimit != 2 || len(document.Result.Rows) != 2 {
		t.Fatal("truncation metadata lost")
	}
}

func TestAnalysisReaderRejectsArchiveResourceAndIntegrityFailures(t *testing.T) {
	original := publicAnalysis(t, "default-source.bean", "SELECT 1 LIMIT 1")
	for _, scenario := range []string{"expanded-budget", "crc", "symlink", "utf8", "compressed-budget"} {
		t.Run(scenario, func(t *testing.T) {
			if scenario == "compressed-budget" {
				if _, err := desktop.ReadQueryAnalysis(bytes.NewReader(original), query.QueryByteLimit+1); err == nil {
					t.Fatal("oversized input accepted")
				}
				return
			}
			source, _ := zip.NewReader(bytes.NewReader(original), int64(len(original)))
			var output bytes.Buffer
			writer := zip.NewWriter(&output)
			for _, file := range source.File {
				reader, _ := file.Open()
				data, _ := io.ReadAll(reader)
				reader.Close()
				header := &zip.FileHeader{Name: file.Name, Method: zip.Store}
				if scenario == "expanded-budget" {
					header.Method = zip.Deflate
					if file.Name == "data.csv" {
						data = bytes.Repeat([]byte("x"), 2*query.QueryByteLimit+1)
					}
				}
				if scenario == "symlink" && file.Name == "data.csv" {
					header.SetMode(os.ModeSymlink | 0600)
				}
				if scenario == "utf8" && file.Name == "data.csv" {
					data = []byte{0xff}
				}
				member, err := writer.CreateHeader(header)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := member.Write(data); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			data := output.Bytes()
			if scenario == "crc" {
				archive, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
				offset, err := archive.File[0].DataOffset()
				if err != nil {
					t.Fatal(err)
				}
				data[offset] ^= 1
			}
			if _, err := desktop.ReadQueryAnalysis(bytes.NewReader(data), int64(len(data))); err == nil {
				t.Fatal("invalid archive accepted")
			}
		})
	}
}

func TestAnalysisReaderRejectsMisleadingDecimalDisplay(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/query/default-source.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: "SELECT decimal('1.25') LIMIT 1"})
	if err != nil {
		t.Fatal(err)
	}
	original := response.Execution.Result.Rows[0][0].Value.(query.PresentedDecimal)
	for _, scenario := range []string{"display", "approximate"} {
		t.Run(scenario, func(t *testing.T) {
			changed := original
			if scenario == "display" {
				changed.Display = "999999"
			} else {
				changed.Approximate = true
			}
			response.Execution.Result.Rows[0][0].Value = changed
			var output bytes.Buffer
			if err := response.WriteAnalysis(&output); err != nil {
				t.Fatal(err)
			}
			if _, err := desktop.ReadQueryAnalysis(bytes.NewReader(output.Bytes()), int64(output.Len())); err == nil {
				t.Fatal("misleading display accepted despite exact CSV agreement")
			}
		})
	}
}

func TestAnalysisStructuredValuesRoundTrip(t *testing.T) {
	for _, read := range []string{
		"SELECT tags,meta,entry,entry_meta('__tolerances__'),other_accounts WHERE date = 2000-01-05 LIMIT 1",
		"SELECT open,close FROM #accounts LIMIT 1",
	} {
		data := publicAnalysis(t, "standard-tables.bean", read)
		document, err := desktop.ReadQueryAnalysis(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			t.Fatal(err)
		}
		if document.Result.Contract != query.CurrentContract || len(document.Result.Rows) != 1 {
			t.Fatalf("%+v", document)
		}
		if read[7:11] == "tags" {
			cells := document.Result.Rows[0]
			if cells[0].Type != "set<string>" || cells[1].Type != "dictionary" || cells[2].Type != "transaction" || cells[3].Type != "dictionary" || cells[4].Type != "list" {
				t.Fatalf("%+v", cells)
			}
			metadata := cells[1].Value.(query.PresentedDictionary)
			if metadata["absent"].Type != "null" || metadata["detail"].Value.(query.PresentedDecimal).Exact != "2.5" {
				t.Fatal("nested scalar types changed")
			}
			tolerance := cells[3].Value.(query.PresentedDictionary)
			if tolerance["USD"].Value.(query.PresentedDecimal).Exact != "0.005" {
				t.Fatal("system metadata changed")
			}
		}
	}
}

// TestAnalysisCaptureRoundTripsStandardForms locks the four standard BQL forms
// (DISTINCT, HAVING, derived table, PIVOT BY) through desktop.Session and the
// analysis archive. The expected values are the captured BeanQuery 0.2.0
// observations for testdata/fixtures/query/standard-forms.bean; the archive
// must restore the pivoted schema, NULL pivot holes and exact decimals.
func TestAnalysisCaptureRoundTripsStandardForms(t *testing.T) {
	for _, test := range []struct {
		name    string
		text    string
		columns []string
		rows    int
	}{
		{"distinct", "SELECT DISTINCT account ORDER BY account", []string{"account"}, 3},
		{"having", "SELECT account, sum(number) AS total GROUP BY account HAVING sum(number) > 0 ORDER BY account", []string{"account", "total"}, 2},
		{"subquery", "SELECT account FROM (SELECT account WHERE number > 0) ORDER BY account", []string{"account"}, 2},
		{"pivot", "SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY 1, 2", []string{"account/currency", "USD"}, 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := publicAnalysis(t, "standard-forms.bean", test.text)
			restored, err := desktop.ReadQueryAnalysis(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatal(err)
			}
			if len(restored.Result.Columns) != len(test.columns) || len(restored.Result.Rows) != test.rows {
				t.Fatalf("restored=%+v", restored.Result)
			}
			for i, name := range test.columns {
				if restored.Result.Columns[i].Name != name {
					t.Fatalf("column %d=%s want %s", i, restored.Result.Columns[i].Name, name)
				}
			}
			if test.name == "distinct" && restored.Result.Rows[0][0].Value != "Assets:Cash" {
				t.Fatalf("distinct rows=%+v", restored.Result.Rows)
			}
			if test.name == "having" {
				decimal, ok := restored.Result.Rows[0][1].Value.(query.PresentedDecimal)
				if !ok || decimal.Exact != "90" {
					t.Fatalf("having rows=%+v", restored.Result.Rows)
				}
			}
			if test.name == "pivot" {
				if restored.Result.Columns[1].Nullable != true || restored.Result.Rows[0][1].Type != "decimal" {
					t.Fatalf("pivot schema=%+v", restored.Result.Columns)
				}
			}
		})
	}
}
