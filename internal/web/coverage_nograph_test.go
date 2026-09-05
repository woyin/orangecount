// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

// adapterFixtureWithFood extends the fixture with the Food account so a
// document can move onto it.
const adapterFixtureWithFood = adapterFixture + "2000-01-01 open Expenses:Food USD\n"

// buildSnapshot builds a valid snapshot for the given ledger text.
func buildSnapshot(t *testing.T, text string) *snapshot.Snapshot {
	t.Helper()
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	return built.Snapshot
}

// TestAdapterRoutesReportMissingGraph serves a snapshot built from a missing
// entry (no source graph) and pins that every graph-backed adapter route
// degrades to a single, explicit 503 instead of panicking.
func TestAdapterRoutesReportMissingGraph(t *testing.T) {
	built := snapshot.Build(filepath.Join(t.TempDir(), "missing.bean"))
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"/__orangecount/fava/editor",
		"/__orangecount/fava/source",
		"/__orangecount/fava/import",
		"/__orangecount/fava/journal",
		"/__orangecount/fava/download-journal",
		"/__orangecount/fava/entry-context?hash=x",
		"/__orangecount/fava/reports/balance_sheet",
		"/api/v1/reports/balance-sheet",
	} {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}

// TestDocumentUploadAndMoveLifecycle drives the attachment upload and move
// endpoints through multipart bodies: a valid round trip plus the account
// and file guards.
func TestDocumentUploadAndMoveLifecycle(t *testing.T) {
	docRoot := t.TempDir()
	roots, err := source.NewDocumentRoots([]string{docRoot})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(buildSnapshot(t, adapterFixtureWithFood)), DocumentRoots: roots, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}

	multipartBody := func(account string, includeFile bool, filename string) (*strings.Reader, string) {
		var body strings.Builder
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("account", account)
		if includeFile {
			part, _ := writer.CreateFormFile("file", filename)
			_, _ = part.Write([]byte("attachment bytes"))
		}
		writer.Close()
		return strings.NewReader(body.String()), writer.FormDataContentType()
	}

	upload := func(account string, includeFile bool, filename string) *httptest.ResponseRecorder {
		body, contentType := multipartBody(account, includeFile, filename)
		request := httptest.NewRequest(http.MethodPost, "/__orangecount/fava/document", body)
		request.Header.Set("Content-Type", contentType)
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, request)
		return recorder
	}

	// Upload requires a declared account and an attached file.
	if bad := upload("NoSuch", true, "a.txt"); bad.Code != http.StatusBadRequest {
		t.Fatalf("unknown account status=%d body=%s", bad.Code, bad.Body.String())
	}
	if noFile := upload("Assets:Cash", false, "a.txt"); noFile.Code != http.StatusBadRequest {
		t.Fatalf("missing file status=%d body=%s", noFile.Code, noFile.Body.String())
	}
	if ok := upload("Assets:Cash", true, "invoice.txt"); ok.Code != http.StatusOK {
		t.Fatalf("valid upload status=%d body=%s", ok.Code, ok.Body.String())
	}
	entries, err := os.ReadDir(filepath.Join(docRoot, "Assets", "Cash"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("uploaded entries=%v err=%v", entries, err)
	}

	// Moving it to another account renames the file beneath the same root.
	move := post(t, server, "/__orangecount/fava/move-document", `{"filename":"Assets/Cash/invoice.txt","account":"Expenses:Food","new_name":"archived.txt"}`)
	if move.Code != http.StatusOK {
		t.Fatalf("move status=%d body=%s", move.Code, move.Body.String())
	}
	if _, err := os.Stat(filepath.Join(docRoot, "Assets", "Cash", "invoice.txt")); !os.IsNotExist(err) {
		t.Fatal("moved file still at source")
	}
	if _, err := os.Stat(filepath.Join(docRoot, "Expenses", "Food", "archived.txt")); err != nil {
		t.Fatalf("moved file missing: %v", err)
	}
}
