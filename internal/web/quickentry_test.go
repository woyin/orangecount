// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orangecount/internal/snapshot"
)

func setupQuickEntryServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	content := `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Expenses:Food CNY
2000-01-01 open Equity:Opening CNY
2000-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat
2000-01-01 custom "orangecount.quick-account.v1" "餐饮" Expenses:Food
option "operating_currency" "CNY"
`
	if err := os.WriteFile(entry, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	return server, entry
}

func TestQuickEntryPreviewCommitRoundTrip(t *testing.T) {
	server, entry := setupQuickEntryServer(t)
	post := func(path, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	// Preview
	previewBody := `{"text":"28 CNY @微信 -> @餐饮 : 工作午餐","date":"2026-08-12"}`
	preview := post("/__orangecount/fava/quick-preview", previewBody)
	if preview.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%q", preview.Code, preview.Body.String())
	}
	var previewPayload quickPreviewResponse
	if err := json.Unmarshal(preview.Body.Bytes(), &previewPayload); err != nil {
		t.Fatal(err)
	}
	if previewPayload.HasErrors {
		t.Fatalf("preview has errors: %+v", previewPayload.Lines)
	}
	if previewPayload.Token == "" {
		t.Fatal("preview token is empty")
	}
	if len(previewPayload.Lines) != 1 || previewPayload.Lines[0].Entry == nil {
		t.Fatalf("expected 1 compiled line, got %+v", previewPayload.Lines)
	}
	// Commit
	commitBody := `{"token":"` + previewPayload.Token + `","expected_snapshot_id":"` + previewPayload.SnapshotID + `"}`
	commit := post("/__orangecount/fava/quick-commit", commitBody)
	if commit.Code != http.StatusOK || !strings.Contains(commit.Body.String(), `"published":true`) {
		t.Fatalf("commit status=%d body=%q", commit.Code, commit.Body.String())
	}
	// Verify the entry was written
	contents, err := os.ReadFile(entry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "2026-08-12 *") || !strings.Contains(string(contents), "Assets:WeChat") || !strings.Contains(string(contents), "Expenses:Food") {
		t.Fatalf("entry file missing published content: %q", contents)
	}
}

func TestQuickEntryCommitRejectsReplay(t *testing.T) {
	server, _ := setupQuickEntryServer(t)
	post := func(path, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	previewBody := `{"text":"28 CNY @微信 -> @餐饮 : 测试","date":"2026-08-12"}`
	preview := post("/__orangecount/fava/quick-preview", previewBody)
	var previewPayload quickPreviewResponse
	json.Unmarshal(preview.Body.Bytes(), &previewPayload)
	commitBody := `{"token":"` + previewPayload.Token + `","expected_snapshot_id":"` + previewPayload.SnapshotID + `"}`
	first := post("/__orangecount/fava/quick-commit", commitBody)
	if first.Code != http.StatusOK {
		t.Fatalf("first commit status=%d body=%q", first.Code, first.Body.String())
	}
	// Replay should fail: single-use token already consumed.
	replay := post("/__orangecount/fava/quick-commit", commitBody)
	if replay.Code != http.StatusNotFound {
		t.Fatalf("replay should be rejected, status=%d body=%q", replay.Code, replay.Body.String())
	}
}

func TestQuickEntryPreviewRejectsInvalidLine(t *testing.T) {
	server, _ := setupQuickEntryServer(t)
	post := func(path, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	// Missing arrow in explicit form.
	previewBody := `{"text":"28 CNY @微信","date":"2026-08-12"}`
	preview := post("/__orangecount/fava/quick-preview", previewBody)
	if preview.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%q", preview.Code, preview.Body.String())
	}
	var payload quickPreviewResponse
	json.Unmarshal(preview.Body.Bytes(), &payload)
	if !payload.HasErrors {
		t.Fatalf("expected errors for invalid line, got %+v", payload.Lines)
	}
}

func TestQuickEntryCommitRejectsStaleSnapshot(t *testing.T) {
	server, _ := setupQuickEntryServer(t)
	post := func(path, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	previewBody := `{"text":"28 CNY @微信 -> @餐饮 : test","date":"2026-08-12"}`
	preview := post("/__orangecount/fava/quick-preview", previewBody)
	var previewPayload quickPreviewResponse
	json.Unmarshal(preview.Body.Bytes(), &previewPayload)
	commitBody := `{"token":"` + previewPayload.Token + `","expected_snapshot_id":"stale-snapshot-id"}`
	commit := post("/__orangecount/fava/quick-commit", commitBody)
	if commit.Code != http.StatusConflict {
		t.Fatalf("stale snapshot should be rejected, status=%d body=%q", commit.Code, commit.Body.String())
	}
}

func TestQuickEntryUndoRemovesBatch(t *testing.T) {
	server, entry := setupQuickEntryServer(t)
	post := func(path, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	// Preview + Commit
	previewBody := `{"text":"28 CNY @微信 -> @餐饮 : undo-test","date":"2026-08-12"}`
	preview := post("/__orangecount/fava/quick-preview", previewBody)
	var previewPayload quickPreviewResponse
	json.Unmarshal(preview.Body.Bytes(), &previewPayload)
	commitBody := `{"token":"` + previewPayload.Token + `","expected_snapshot_id":"` + previewPayload.SnapshotID + `"}`
	post("/__orangecount/fava/quick-commit", commitBody)
	// Verify content was written
	before, _ := os.ReadFile(entry)
	if !strings.Contains(string(before), "undo-test") {
		t.Fatalf("content not written: %q", before)
	}
	// Undo
	undo := post("/__orangecount/fava/quick-undo", `{}`)
	if undo.Code != http.StatusOK || !strings.Contains(undo.Body.String(), `"undone":true`) {
		t.Fatalf("undo status=%d body=%q", undo.Code, undo.Body.String())
	}
	// Verify content was removed
	after, _ := os.ReadFile(entry)
	if strings.Contains(string(after), "undo-test") {
		t.Fatalf("undo should remove the batch, but content remains: %q", after)
	}
}

func TestQuickEntryProfileListing(t *testing.T) {
	server, _ := setupQuickEntryServer(t)
	get := func(path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}
	resp := get("/__orangecount/fava/quick-profile")
	if resp.Code != http.StatusOK {
		t.Fatalf("profile status=%d body=%q", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"type":"account"`) || !strings.Contains(body, "微信") {
		t.Fatalf("profile listing missing account rules: %q", body)
	}
}
