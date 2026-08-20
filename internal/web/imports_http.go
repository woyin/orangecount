// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"orangecount/internal/authoring"
	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	current := s.store.Current()
	if current == nil || current.Graph() == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	suffix := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/import"), "/")
	if r.Method == http.MethodGet && (suffix == "" || suffix == "targets") {
		writeJSON(w, struct {
			Paths      []string `json:"paths"`
			Entry      string   `json:"entry"`
			SnapshotID string   `json:"snapshot_id"`
		}{Paths: current.Graph().DisplayPaths(), Entry: current.Graph().DisplayPath(current.Graph().Entry), SnapshotID: current.ID})
		return
	}
	if r.Method == http.MethodGet && suffix == "adapters" {
		writeJSON(w, struct {
			Adapters []map[string]any `json:"adapters"`
		}{Adapters: []map[string]any{
			{"id": "beancount", "label": "Beancount source", "extensions": []string{".bean", ".beancount"}},
			{"id": "csv", "label": "Generic CSV", "extensions": []string{".csv"}, "columns": []string{"date", "payee", "account", "amount", "currency", "narration"}},
		}})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.NotFound(w, r)
		return
	}
	switch suffix {
	case "preview":
		s.handleImportPreview(w, r, current)
	case "commit":
		s.handleImportCommit(w, r, current)
	default:
		http.NotFound(w, r)
	}
}

type importPreviewRequest struct {
	Path    string            `json:"path"`
	Content string            `json:"content"`
	Adapter string            `json:"adapter"`
	Mapping map[string]string `json:"mapping"`
}

func (s *Server) handleImportPreview(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request importPreviewRequest
	if err := decodeJSONBody(w, r, &request, 4<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	adapter := strings.ToLower(strings.TrimSpace(request.Adapter))
	if adapter == "" {
		adapter = "beancount"
	}
	name, normalized, err := normalizedImportContent(request, adapter)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	file, bag := ledger.ParseText(name, []byte(normalized))
	diagnostics := bag.All()
	if bag.HasErrors() {
		writeJSON(w, struct {
			Valid       bool                 `json:"valid"`
			Diagnostics []diagnosticResponse `json:"diagnostics"`
		}{Valid: false, Diagnostics: diagnosticsPayload(diagnostics, nil)})
		return
	}
	if err := rejectImportDirectives(file); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Evaluate the imported file against the merged include graph rather than
	// in isolation (see evaluateImportMerged); this mirrors what the commit
	// path revalidates, so a preview that says "valid" will commit.
	evaluation := evaluateImportMerged(current, file)
	for _, value := range evaluation.Diagnostics {
		diagnostics = append(diagnostics, value)
	}
	previewID := importPreviewID(name, normalized)
	s.previews.Store(previewID, importPreview{Path: name, Content: normalized})
	rows := importedRowsOnly(evaluation, name)
	writeJSON(w, struct {
		PreviewID   string               `json:"preview_id"`
		Path        string               `json:"path"`
		Valid       bool                 `json:"valid"`
		Diagnostics []diagnosticResponse `json:"diagnostics"`
		Rows        query.Result         `json:"rows"`
		Diff        importDiff           `json:"diff"`
	}{PreviewID: previewID, Path: name, Valid: importEvaluationValid(evaluation), Diagnostics: diagnosticsPayload(diagnostics, nil), Rows: report.Present(rows), Diff: importDiff{AddedLines: strings.Count(normalized, "\n") + 1, Bytes: len([]byte(normalized))}})
}

// normalizedImportContent validates the request's file name against its
// adapter and returns the Beancount text the preview should evaluate. CSV
// input is converted and the file name re-pointed at a .bean extension.
func normalizedImportContent(request importPreviewRequest, adapter string) (name, content string, err error) {
	name, err = safeImportName(request.Path, adapter)
	if err != nil {
		return "", "", err
	}
	content = request.Content
	if adapter == "csv" {
		content, err = csvToBeancount(request.Content, request.Mapping)
		if err != nil {
			return "", "", err
		}
		name = strings.TrimSuffix(name, filepath.Ext(name)) + ".bean"
	}
	return name, content, nil
}

// rejectImportDirectives refuses imports that would grow the include graph
// or activate plugins; an import may only add entries.
func rejectImportDirectives(file *ledger.File) error {
	for _, directive := range file.Directives {
		switch directive.(type) {
		case ledger.Include, *ledger.Include, ledger.Plugin, *ledger.Plugin:
			return fmt.Errorf("imports cannot add include or plugin directives")
		}
	}
	return nil
}

// evaluateImportMerged evaluates the imported file against the published
// include graph rather than in isolation. The full graph knows which accounts
// are open and which currencies are permitted, so a posting to an account
// opened only in the main ledger (the common case for an import) is not
// misreported as an E-EVAL-POSTING lifecycle error. The imported file is not
// yet a member of the published graph, so it is assigned a fresh FileID that
// cannot collide with the existing members.
func evaluateImportMerged(current *snapshot.Snapshot, file *ledger.File) *ledger.Evaluation {
	order := append([]source.FileID(nil), current.Graph().Order...)
	parsed := current.Parsed()
	importID := source.FileID(len(parsed) + 1)
	parsed[importID] = file
	order = append(order, importID)
	return ledger.EvaluateFiles(parsed, order, ledger.EvalOptions{})
}

// importedRowsOnly projects the merged evaluation's postings and keeps only
// the imported file's rows: the merged graph necessarily includes the
// existing ledger's postings, but the preview table should show only what
// this import would add.
func importedRowsOnly(evaluation *ledger.Evaluation, name string) query.Result {
	rows, err := query.Evaluate("SELECT date, account, units, currency, flag, payee, narration FROM postings ORDER BY date, account", *evaluation)
	if err != nil {
		return query.Result{}
	}
	imported := rows.Rows[:0]
	for _, row := range rows.Rows {
		if row["file"] == name {
			imported = append(imported, row)
		}
	}
	rows.Rows = imported
	return rows
}

// importEvaluationValid reports whether the whole merged graph, including the
// proposed import, is free of error diagnostics. The imported file itself is
// always valid by the time this runs (parse errors returned earlier).
func importEvaluationValid(evaluation *ledger.Evaluation) bool {
	for _, value := range evaluation.Diagnostics {
		if value.Severity == diagnostic.Error {
			return false
		}
	}
	return true
}

type importDiff struct {
	AddedLines int `json:"added_lines"`
	Bytes      int `json:"bytes"`
}

type importCommitRequest struct {
	PreviewID        string `json:"preview_id"`
	Target           string `json:"target"`
	ExpectedSnapshot string `json:"expected_snapshot_id"`
}

// handleImportCommit (POST) applies a staged import: re-validates the
// previewed entry against the expected snapshot, applies the file edits, and
// rebuilds; a mismatch aborts with a conflict instead of writing.
func (s *Server) handleImportCommit(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request importCommitRequest
	if err := decodeJSONBody(w, r, &request, 64<<10); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	preview, ok := s.previews.Take(request.PreviewID)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "import preview not found or expired")
		return
	}
	if request.ExpectedSnapshot != "" && request.ExpectedSnapshot != current.ID {
		writeAPIError(w, http.StatusConflict, "snapshot changed; refresh import targets")
		return
	}
	target := strings.TrimSpace(request.Target)
	if target == "" {
		target = current.Graph().DisplayPath(current.Graph().Entry)
	}
	_, display, ok := graphFile(current.Graph(), target)
	if !ok {
		http.NotFound(w, r)
		return
	}
	change, err := authoring.Append(display, preview.Content)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.authoring.Publish(current.ID, change)
	if err != nil {
		status := authoringStatus(err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		writeJSON(w, struct {
			Published   bool                 `json:"published"`
			Backup      string               `json:"backup,omitempty"`
			Diagnostics []diagnosticResponse `json:"diagnostics"`
		}{Backup: result.Backup, Diagnostics: diagnosticsPayload(result.Build.Diagnostics, current.Graph())})
		return
	}
	s.previews.Discard(request.PreviewID)
	writeJSON(w, struct {
		Published  bool   `json:"published"`
		SnapshotID string `json:"snapshot_id"`
		Backup     string `json:"backup"`
	}{Published: true, SnapshotID: result.Build.Snapshot.ID, Backup: result.Backup})
}

func safeImportName(raw, adapter string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.Base(raw) != raw || strings.ContainsAny(raw, `/\\`) {
		return "", fmt.Errorf("import path must be a file name")
	}
	ext := strings.ToLower(filepath.Ext(raw))
	if adapter == "csv" && ext == ".csv" {
		return raw, nil
	}
	if adapter != "beancount" || (ext != ".bean" && ext != ".beancount") {
		return "", fmt.Errorf("import path extension does not match adapter")
	}
	return raw, nil
}

// csvToBeancount converts a generic CSV import into Beancount transactions
// balanced against an offset account. The header row names the columns
// (date, account, amount required; currency, payee, narration optional).
func csvToBeancount(content string, mapping map[string]string) (string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("invalid CSV: %w", err)
	}
	if len(records) < 2 {
		return "", fmt.Errorf("CSV must contain a header and at least one row")
	}
	header, err := csvHeaderIndex(records[0])
	if err != nil {
		return "", err
	}
	offset := csvMappingValue(mapping, "offset_account", "Equity:Imported")
	currencyDefault := csvMappingValue(mapping, "currency", "USD")
	var builder strings.Builder
	for line, record := range records[1:] {
		rendered, renderErr := csvRecordToBean(record, header, offset, currencyDefault)
		if renderErr != nil {
			return "", fmt.Errorf("CSV row %d: %w", line+2, renderErr)
		}
		builder.WriteString(rendered)
	}
	return builder.String(), nil
}

// csvHeaderIndex maps lowercased/trimmed header names to record indexes and
// enforces the required columns.
func csvHeaderIndex(headerRow []string) (map[string]int, error) {
	header := make(map[string]int, len(headerRow))
	for index, value := range headerRow {
		header[strings.ToLower(strings.TrimSpace(value))] = index
	}
	for _, required := range []string{"date", "account", "amount"} {
		if _, ok := header[required]; !ok {
			return nil, fmt.Errorf("CSV requires %s column", required)
		}
	}
	return header, nil
}

// csvMappingValue reads one mapping override, falling back to the default
// when the mapping is nil or the value is blank.
func csvMappingValue(mapping map[string]string, key, fallback string) string {
	if mapping == nil {
		return fallback
	}
	if value := strings.TrimSpace(mapping[key]); value != "" {
		return value
	}
	return fallback
}

// csvRecordToBean renders one CSV record as a balanced transaction with its
// offset posting. Embedded newlines in payee/narration are flattened so one
// record can never produce broken directives.
func csvRecordToBean(record []string, header map[string]int, offset, currencyDefault string) (string, error) {
	value := func(key string) string {
		index, ok := header[key]
		if !ok || index >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[index])
	}
	date, amount := value("date"), value("amount")
	if _, err := parseISODate(date); err != nil {
		return "", fmt.Errorf("invalid date")
	}
	parsedAmount, err := ledger.ParseDecimal(amount)
	if err != nil {
		return "", fmt.Errorf("invalid amount")
	}
	currency := value("currency")
	if currency == "" {
		currency = currencyDefault
	}
	payee := csvSingleLine(value("payee"))
	narration := csvSingleLine(value("narration"))
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s * \"%s\"", date, escapeBeanString(payee)))
	if narration != "" {
		builder.WriteString(fmt.Sprintf(" \"%s\"", escapeBeanString(narration)))
	}
	builder.WriteByte('\n')
	builder.WriteString(fmt.Sprintf("  %s %s %s\n", value("account"), parsedAmount.String(), currency))
	builder.WriteString(fmt.Sprintf("  %s %s %s\n", offset, parsedAmount.Neg().String(), currency))
	return builder.String(), nil
}

// csvSingleLine collapses line breaks so CSV fields stay single-line strings.
func csvSingleLine(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\n", " "), "\r", " ")
}

func escapeBeanString(value string) string {
	return strings.ReplaceAll(value, "\"", "\\\"")
}

func importPreviewID(path, content string) string {
	hash := sha256.New()
	hash.Write([]byte(path))
	hash.Write([]byte{0})
	hash.Write([]byte(content))
	return fmt.Sprintf("%x", hash.Sum(nil))[:16]
}

// handleOptions serves the local options API: GET returns current values,
// POST (same-origin) validates and persists them alongside the ledger.
