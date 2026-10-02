// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/source"
	"path/filepath"
	"slices"
)

type QueryExportRequest struct {
	SnapshotID  string `json:"snapshot_id"`
	RequestID   uint64 `json:"request_id"`
	Fingerprint string `json:"fingerprint"`
	Format      string `json:"format"`
}

type QueryExportCapture struct {
	Context QueryContext `json:"context"`
	Format  string       `json:"format"`
	Content []byte       `json:"content"`
	// Local write protection, deliberately excluded from archive members.
	ProtectedPaths []string `json:"protected_paths"`
}

type ExportProtection struct {
	Revision       uint64   `json:"revision"`
	SnapshotID     string   `json:"snapshot_id"`
	ProtectedPaths []string `json:"protected_paths"`
}

// ExportProtection refreshes local write guards without changing previously
// captured analysis bytes. Failed input graphs also remain protected.
func (s *Session) ExportProtection() (ExportProtection, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ExportProtection{}, ErrClosed
	}
	if s.state.Reloading {
		s.mu.RUnlock()
		return ExportProtection{}, ErrStale
	}
	state, current, graph := s.state, s.current, s.latest.Graph
	documents := s.latest.DocumentDeclarations()
	s.mu.RUnlock()
	paths := exportProtectedPaths(graph, ledger.Evaluation{})
	for _, document := range documents {
		path := document.Filename
		if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(document.SourcePath), path)
		}
		paths = append(paths, filepath.Clean(path))
		if real, err := filepath.EvalSymlinks(path); err == nil {
			paths = append(paths, real)
		}
	}
	if current != nil {
		paths = append(paths, current.protectedPaths...)
	}
	slices.Sort(paths)
	paths = slices.Compact(paths)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return ExportProtection{}, ErrClosed
	}
	if s.state.Revision != state.Revision || s.state.Reloading {
		return ExportProtection{}, ErrStale
	}
	return ExportProtection{Revision: state.Revision, SnapshotID: state.SnapshotID, ProtectedPaths: paths}, nil
}

// CaptureQueryExport freezes bytes before a save panel is opened. It accepts
// only the current successful generation; already returned bytes remain valid
// after reload, later execution, or session close.
func (s *Session) CaptureQueryExport(request QueryExportRequest) (QueryExportCapture, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return QueryExportCapture{}, ErrClosed
	}
	response := s.queryExport
	if request.SnapshotID != s.state.SnapshotID {
		s.mu.RUnlock()
		return QueryExportCapture{}, ErrStale
	}
	if response == nil || request.RequestID != response.Context.RequestID || request.Fingerprint != response.Context.Fingerprint || (request.Format != "csv" && request.Format != "analysis") {
		s.mu.RUnlock()
		return QueryExportCapture{}, ErrRequest
	}
	protectedPaths := append([]string{}, s.queryExportProtectedPaths...)
	s.mu.RUnlock()
	var output bytes.Buffer
	writer := &queryExportWriter{output: &output, remaining: query.QueryByteLimit}
	var err error
	if request.Format == "csv" {
		err = response.Execution.Result.WriteCSV(writer)
	} else {
		err = response.WriteAnalysis(writer)
	}
	if err != nil {
		return QueryExportCapture{}, err
	}
	return QueryExportCapture{Context: response.Context, Format: request.Format, Content: output.Bytes(), ProtectedPaths: protectedPaths}, nil
}

func exportProtectedPaths(graph *source.Graph, evaluation ledger.Evaluation) []string {
	paths := map[string]bool{}
	add := func(path string) {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return
		}
		paths[filepath.Clean(absolute)] = true
		if real, err := filepath.EvalSymlinks(absolute); err == nil {
			paths[real] = true
		}
	}
	if graph != nil {
		for path := range graph.ByPath {
			add(path)
		}
	}
	for _, record := range evaluation.Entries {
		var filenames []string
		switch document := record.Directive.(type) {
		case ledger.Document:
			filenames = document.Filenames
		case *ledger.Document:
			if document != nil {
				filenames = document.Filenames
			}
		}
		for _, name := range filenames {
			if !filepath.IsAbs(name) {
				name = filepath.Join(filepath.Dir(record.File), name)
			}
			add(name)
		}
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	slices.Sort(result)
	return result
}

type queryExportWriter struct {
	output    io.Writer
	remaining int
}

func (w *queryExportWriter) Write(data []byte) (int, error) {
	if len(data) > w.remaining {
		return 0, ErrResultBudget
	}
	n, err := w.output.Write(data)
	w.remaining -= n
	return n, err
}

// Scalars and amounts are values. Costs and inventory slices need independent
// copies because the public execution response belongs to its caller.
func cloneQueryResponse(response QueryResponse) QueryResponse {
	execution := *response.Execution
	execution.PhaseSteps = nil
	execution.Result.Columns = append([]query.Column{}, execution.Result.Columns...)
	execution.Result.Rows = make([][]query.Cell, len(response.Execution.Result.Rows))
	clonePosition := func(value query.PresentedPosition) query.PresentedPosition {
		if value.Cost != nil {
			cost := *value.Cost
			if cost.Date != nil {
				date := *cost.Date
				cost.Date = &date
			}
			if cost.Label != nil {
				label := *cost.Label
				cost.Label = &label
			}
			value.Cost = &cost
		}
		return value
	}
	for i, row := range response.Execution.Result.Rows {
		execution.Result.Rows[i] = append([]query.Cell{}, row...)
		for j, cell := range row {
			switch value := cell.Value.(type) {
			case query.PresentedPosition:
				execution.Result.Rows[i][j].Value = clonePosition(value)
			case []query.PresentedPosition:
				positions := make([]query.PresentedPosition, len(value))
				for k, position := range value {
					positions[k] = clonePosition(position)
				}
				execution.Result.Rows[i][j].Value = positions
			}
		}
	}
	response.Execution = &execution
	return response
}

// QueryAnalysisContext describes the captured result, never a new evaluation
// of the editor draft or an assertion that all matching source rows were read.
type QueryAnalysisContext struct {
	FormatVersion      int                     `json:"format_version"`
	Producer           string                  `json:"producer"`
	View               string                  `json:"view"`
	SnapshotID         string                  `json:"snapshot_id"`
	SemanticsProfile   ledger.SemanticsProfile `json:"semantics_profile"`
	EvaluationRevision string                  `json:"evaluation_revision"`
	NormalizedRequest  QueryContext            `json:"normalized_request"`
	CaptureScope       string                  `json:"capture_scope"`
	ReturnedRows       int                     `json:"returned_rows"`
	Truncated          bool                    `json:"truncated"`
	Completeness       string                  `json:"completeness"`
	Page               *struct{}               `json:"page"`
	Contract           string                  `json:"contract"`
	RowLimit           int                     `json:"row_limit"`
}

// WriteAnalysis serializes a successful captured response. File replacement
// and target protection are the caller's responsibility. It never rereads a
// session or copies a source graph, attachment, or diagnostic into the archive.
func (response QueryResponse) WriteAnalysis(output io.Writer) error {
	if response.Error != nil || response.Execution == nil || response.Context.SnapshotID == "" || response.Context.RequestID == 0 {
		return fmt.Errorf("analysis export requires a captured successful query")
	}
	execution := response.Execution
	if execution.ReturnedRows != len(execution.Result.Rows) || execution.ReturnedRows > query.QueryRowLimit {
		return fmt.Errorf("analysis export has inconsistent row metadata")
	}
	completeness := "complete"
	if execution.Truncated {
		completeness = "truncated"
	}
	capture := QueryAnalysisContext{
		FormatVersion: 1, Producer: "OrangeCount", View: "query", SnapshotID: response.Context.SnapshotID,
		SemanticsProfile: response.Context.SemanticsProfile, EvaluationRevision: response.Context.EvaluationRevision,
		NormalizedRequest: response.Context, CaptureScope: "captured-result", ReturnedRows: execution.ReturnedRows,
		Truncated: execution.Truncated, Completeness: completeness, Contract: execution.Result.Contract, RowLimit: execution.RowLimit,
	}
	archive := zip.NewWriter(output)
	writeJSON := func(name string, value any) error {
		member, err := archive.Create(name)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(member)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(value)
	}
	if err := writeJSON("context.json", capture); err != nil {
		return err
	}
	if err := writeJSON("schema.json", execution.Result.Columns); err != nil {
		return err
	}
	cells, err := archive.Create("cells.ndjson")
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(cells)
	encoder.SetEscapeHTML(false)
	for _, row := range execution.Result.Rows {
		if len(row) != len(execution.Result.Columns) {
			return fmt.Errorf("analysis export row width does not match schema")
		}
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	csv, err := archive.Create("data.csv")
	if err != nil {
		return err
	}
	if err := execution.Result.WriteCSV(csv); err != nil {
		return err
	}
	return archive.Close()
}
