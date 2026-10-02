// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"orangecount/internal/desktop"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestQueryExportCaptureProtectsIncludedSourcesAndDeclaredAttachments(t *testing.T) {
	directory := t.TempDir()
	entry := filepath.Join(directory, "main.bean")
	child := filepath.Join(directory, "child.bean")
	attachment := filepath.Join(directory, "receipt.pdf")
	if err := os.WriteFile(entry, []byte("include \"child.bean\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, []byte("2000-01-01 open Assets:Cash USD\n2000-01-02 document Assets:Cash \"receipt.pdf\"\n2000-01-03 document Assets:Cash \"missing.pdf\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(attachment, []byte("public attachment"), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(entry, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: "SELECT account LIMIT 0"})
	if err != nil {
		t.Fatal(err)
	}
	capture, err := session.CaptureQueryExport(desktop.QueryExportRequest{SnapshotID: state.SnapshotID, RequestID: 1, Fingerprint: response.Context.Fingerprint, Format: "analysis"})
	if err != nil {
		t.Fatal(err)
	}
	realDirectory, err := filepath.EvalSymlinks(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(realDirectory, "main.bean"), filepath.Join(realDirectory, "child.bean"), filepath.Join(realDirectory, "receipt.pdf"), filepath.Join(realDirectory, "missing.pdf")} {
		if !slices.Contains(capture.ProtectedPaths, path) {
			t.Fatalf("missing protected public path: %s", path)
		}
	}
	archive, err := zip.NewReader(bytes.NewReader(capture.Content), int64(len(capture.Content)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(directory)) || bytes.Contains(data, []byte("protected_paths")) {
			t.Fatal("local target protection leaked into saved analysis")
		}
	}
}

func TestExportProtectionRefreshIncludesReloadedAndFailedSourceGraphs(t *testing.T) {
	directory := t.TempDir()
	entry := filepath.Join(directory, "main.bean")
	if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\n"), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(entry, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: "SELECT account LIMIT 0"})
	if err != nil {
		t.Fatal(err)
	}
	capture, err := session.CaptureQueryExport(desktop.QueryExportRequest{SnapshotID: state.SnapshotID, RequestID: 1, Fingerprint: response.Context.Fingerprint, Format: "csv"})
	if err != nil {
		t.Fatal(err)
	}
	for _, childName := range []string{"new.bean", "invalid.bean"} {
		content := "2000-01-02 price USD 1 EUR\n"
		if childName == "invalid.bean" {
			content = "2000-01-03 document Assets:Cash \"new-attachment.csv\"\nnot a valid directive\n"
		}
		if err := os.WriteFile(filepath.Join(directory, childName), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(entry, []byte("2000-01-01 open Assets:Cash USD\ninclude \""+childName+"\"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := session.Reload(); err != nil {
			t.Fatal(err)
		}
		protection, err := session.ExportProtection()
		if err != nil {
			t.Fatal(err)
		}
		realDirectory, _ := filepath.EvalSymlinks(directory)
		if !slices.Contains(protection.ProtectedPaths, filepath.Join(realDirectory, childName)) {
			t.Fatal("latest include graph unprotected")
		}
		if childName == "invalid.bean" {
			if !slices.Contains(protection.ProtectedPaths, filepath.Join(realDirectory, "new-attachment.csv")) {
				t.Fatal("new attachment in failed build unprotected")
			}
			current, _ := session.Poll()
			if current.Valid || current.SnapshotID != protection.SnapshotID {
				t.Fatal("failed input published an accounting snapshot")
			}
		}
		if capture.Context.SnapshotID != state.SnapshotID || string(capture.Content) != "account\n" {
			t.Fatal("protection refresh replaced captured content")
		}
	}
	session.Close()
	if _, err := session.ExportProtection(); !errors.Is(err, desktop.ErrClosed) {
		t.Fatalf("closed session returned protection: %v", err)
	}
}

type brokenAnalysisWriter struct{}

func (brokenAnalysisWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestAnalysisRejectsFailedExecutionAndPropagatesWriteFailure(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/query/default-source.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	success, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: "SELECT number LIMIT 1"})
	if err != nil {
		t.Fatal(err)
	}
	failed, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 2, Text: "SELECT unknown(number) LIMIT 0"})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := failed.WriteAnalysis(&output); err == nil || output.Len() != 0 {
		t.Fatal("failed execution exported or wrote bytes")
	}
	if err := success.WriteAnalysis(brokenAnalysisWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write failure lost: %v", err)
	}
	if err := success.WriteAnalysis(&output); err != nil {
		t.Fatal(err)
	}
	if success.Context.RequestID != 1 || success.Context.Text != "SELECT number LIMIT 1" {
		t.Fatal("later failure changed captured execution")
	}
}

func TestSessionQueryExportFreezesSuccessAndRejectsOtherExecutions(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/query/compound-casts.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	success, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 1, Text: "SELECT sum(position) WHERE account = 'Assets:Precision'"})
	if err != nil {
		t.Fatal(err)
	}
	request := desktop.QueryExportRequest{SnapshotID: state.SnapshotID, RequestID: 1, Fingerprint: success.Context.Fingerprint, Format: "csv"}
	var expected bytes.Buffer
	if err := success.Execution.Result.WriteCSV(&expected); err != nil {
		t.Fatal(err)
	}
	// The caller owns its returned response; changing it must not change export.
	positions := success.Execution.Result.Rows[0][0].Value.([]query.PresentedPosition)
	*positions[0].Cost.Label = "changed label"
	positions[0].Cost.Number.Exact = "999"
	success.Execution.Result.Columns[0].Name = "changed"
	success.Execution.Result.Rows[0][0].Value = "changed"
	captured, err := session.CaptureQueryExport(request)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(captured.Content, expected.Bytes()) || captured.Context.RequestID != 1 {
		t.Fatal("export read caller mutations")
	}
	wrong := request
	wrong.Fingerprint = "other"
	if _, err := session.CaptureQueryExport(wrong); !errors.Is(err, desktop.ErrRequest) {
		t.Fatalf("wrong identity: %v", err)
	}
	_, err = session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: 2, Text: "SELECT unknown(number) LIMIT 0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.CaptureQueryExport(request); !errors.Is(err, desktop.ErrRequest) {
		t.Fatalf("old success presented as current: %v", err)
	}
	if !bytes.Equal(captured.Content, expected.Bytes()) {
		t.Fatal("already captured bytes changed")
	}
}

func TestCapturedQueryAnalysisRetainsContextSchemaAndTypedCells(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/query/default-source.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state, _ := session.Poll()
	for i, text := range []string{"SELECT 1 AS v, 2 AS v, decimal('1.00000000000000000001') AS exact, NULL AS n, '' AS blank LIMIT 1", "SELECT account, position LIMIT 0"} {
		response, err := session.ExecuteQuery(context.Background(), desktop.QueryRequest{SnapshotID: state.SnapshotID, RequestID: uint64(i + 1), Text: text})
		if err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		if err := response.WriteAnalysis(&output); err != nil {
			t.Fatal(err)
		}
		archive, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
		if err != nil {
			t.Fatal(err)
		}
		members := map[string][]byte{}
		for _, file := range archive.File {
			reader, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(reader)
			reader.Close()
			if err != nil {
				t.Fatal(err)
			}
			members[file.Name] = data
		}
		if len(members) != 4 {
			t.Fatalf("unexpected members: %v", members)
		}
		var capture desktop.QueryAnalysisContext
		if err := json.Unmarshal(members["context.json"], &capture); err != nil {
			t.Fatal(err)
		}
		if capture.FormatVersion != 1 || capture.CaptureScope != "captured-result" || capture.SnapshotID != state.SnapshotID || capture.NormalizedRequest.Text != text || capture.ReturnedRows != response.Execution.ReturnedRows {
			t.Fatalf("lost capture: %+v", capture)
		}
		schema, _ := json.Marshal(response.Execution.Result.Columns)
		if !bytes.Equal(bytes.TrimSpace(members["schema.json"]), schema) {
			t.Fatal("schema changed")
		}
		rows, err := csv.NewReader(bytes.NewReader(members["data.csv"])).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if len(rows) != 2 || rows[0][0] != "v" || rows[0][1] != "v" || rows[1][2] != "1.00000000000000000001" {
				t.Fatal("exact CSV lost")
			}
			cells, _ := json.Marshal(response.Execution.Result.Rows[0])
			if !bytes.Equal(bytes.TrimSpace(members["cells.ndjson"]), cells) {
				t.Fatal("typed NULL/empty or cell positions changed")
			}
		} else if len(members["cells.ndjson"]) != 0 || len(rows) != 1 {
			t.Fatal("zero rows lost schema or acquired data")
		}
	}
}
