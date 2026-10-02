// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"orangecount/internal/analysis"
	"reflect"
	"strings"
	"unicode/utf8"

	"orangecount/internal/query"
)

// QueryAnalysisDocument is a captured result; reading it never executes a query.
type QueryAnalysisDocument struct {
	Context QueryAnalysisContext
	Result  query.TypedResult
}

// ReadQueryAnalysis validates the bounded archive and restores concrete typed
// values. CSV must agree byte-for-byte with the canonical captured projection.
func ReadQueryAnalysis(input io.ReaderAt, size int64) (QueryAnalysisDocument, error) {
	var result QueryAnalysisDocument
	fail := func() (QueryAnalysisDocument, error) {
		return QueryAnalysisDocument{}, fmt.Errorf("invalid query analysis container")
	}
	if size <= 0 || size > query.QueryByteLimit {
		return fail()
	}
	archive, err := zip.NewReader(input, size)
	if err != nil || len(archive.File) != 4 {
		return fail()
	}
	members := map[string][]byte{}
	var total uint64
	for _, file := range archive.File {
		switch file.Name {
		case "context.json", "schema.json", "cells.ndjson", "data.csv":
		default:
			return fail()
		}
		if _, exists := members[file.Name]; exists || !file.Mode().IsRegular() || file.UncompressedSize64 > 2*query.QueryByteLimit {
			return fail()
		}
		total += file.UncompressedSize64
		if total > 4*query.QueryByteLimit {
			return fail()
		}
		reader, err := file.Open()
		if err != nil {
			return fail()
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, 2*query.QueryByteLimit+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || len(data) > 2*query.QueryByteLimit || !utf8.Valid(data) {
			return fail()
		}
		members[file.Name] = data
	}
	if analysis.StrictJSON(members["context.json"], &result.Context) != nil || analysis.StrictJSON(members["schema.json"], &result.Result.Columns) != nil {
		return fail()
	}
	c := result.Context
	q := c.NormalizedRequest
	if c.FormatVersion != 1 || c.Producer == "" || c.View != "query" || c.CaptureScope != "captured-result" || c.Page != nil || c.Contract != "beanquery-0.2.0-typed-v1" || c.RowLimit < 1 || c.RowLimit > query.QueryRowLimit || c.ReturnedRows < 0 || c.ReturnedRows > c.RowLimit || q.RequestID == 0 || strings.TrimSpace(q.Text) == "" || len(q.Text) > 65536 || c.SnapshotID == "" || c.EvaluationRevision == "" || c.SnapshotID != q.SnapshotID || c.SemanticsProfile != q.SemanticsProfile || c.EvaluationRevision != q.EvaluationRevision {
		return fail()
	}
	if (c.SemanticsProfile != "legacy" && c.SemanticsProfile != "v3") || (c.Truncated && (c.ReturnedRows != c.RowLimit || c.Completeness != "truncated")) || (!c.Truncated && c.Completeness != "complete") {
		return fail()
	}
	fingerprint := q.Fingerprint
	q.RequestID = 0
	q.Fingerprint = ""
	identity, _ := json.Marshal(q)
	hash := sha256.Sum256(identity)
	if fingerprint != hex.EncodeToString(hash[:]) {
		return fail()
	}
	ids := map[string]bool{}
	if result.Result.Columns == nil || len(result.Result.Columns) > 4096 {
		return fail()
	}
	for _, column := range result.Result.Columns {
		if column.ID == "" || ids[column.ID] || !analysisType(column.Type, true) || (column.Type == "null" && !column.Nullable) {
			return fail()
		}
		ids[column.ID] = true
	}
	result.Result.Contract = c.Contract
	result.Result.Rows = make([][]query.Cell, 0)
	scanner := bufio.NewScanner(bytes.NewReader(members["cells.ndjson"]))
	scanner.Buffer(make([]byte, 4096), 2*query.QueryByteLimit)
	for scanner.Scan() {
		if len(result.Result.Rows) >= c.ReturnedRows {
			return fail()
		}
		var row []struct {
			Type  string          `json:"type"`
			Value json.RawMessage `json:"value"`
		}
		if analysis.StrictJSON(scanner.Bytes(), &row) != nil || row == nil || len(row) != len(result.Result.Columns) {
			return fail()
		}
		restored := make([]query.Cell, len(row))
		for i, cell := range row {
			column := result.Result.Columns[i]
			if !analysisType(cell.Type, false) || (cell.Type == "null" && !column.Nullable) || (cell.Type != "null" && column.Type != "object" && cell.Type != column.Type) {
				return fail()
			}
			value, err := analysisCell(cell.Type, cell.Value)
			if err != nil {
				return fail()
			}
			restored[i] = query.Cell{Type: cell.Type, Value: value}
		}
		result.Result.Rows = append(result.Result.Rows, restored)
	}
	if scanner.Err() != nil || len(result.Result.Rows) != c.ReturnedRows {
		return fail()
	}
	var csv bytes.Buffer
	if result.Result.WriteCSV(&csv) != nil || !bytes.Equal(csv.Bytes(), members["data.csv"]) {
		return fail()
	}
	return result, nil
}

func analysisType(kind string, schema bool) bool {
	switch kind {
	case "null", "string", "date", "integer", "bool", "decimal", "amount", "position", "inventory":
		return true
	case "object":
		return schema
	}
	return false
}

func analysisCell(kind string, raw json.RawMessage) (any, error) {
	invalid := fmt.Errorf("invalid typed cell")
	if kind == "null" {
		if len(raw) != 0 {
			return nil, invalid
		}
		return nil, nil
	}
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, invalid
	}
	var target any
	switch kind {
	case "string", "date", "integer":
		target = new(string)
	case "bool":
		target = new(bool)
	case "decimal":
		target = new(query.PresentedDecimal)
	case "amount":
		target = new(query.PresentedAmount)
	case "position":
		target = new(query.PresentedPosition)
	case "inventory":
		target = new([]query.PresentedPosition)
	default:
		return nil, invalid
	}
	if analysis.StrictJSON(raw, target) != nil {
		return nil, invalid
	}
	value := reflect.ValueOf(target).Elem().Interface()
	if !analysis.ValidValue(value, kind) {
		return nil, invalid
	}
	return value, nil
}
