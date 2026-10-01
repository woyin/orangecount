// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package compat

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/snapshot"
)

type queryColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type queryResult struct {
	Columns []queryColumn  `json:"columns"`
	Rows    [][]query.Cell `json:"rows"`
}

type queryGoldenCase struct {
	ID            string            `json:"id"`
	Query         string            `json:"query"`
	Result        *queryResult      `json:"result,omitempty"`
	SourceColumns []int             `json:"source_columns,omitempty"`
	Error         *queryGoldenError `json:"error,omitempty"`
}

type queryGoldenError struct {
	Class string `json:"class"`
	Code  string `json:"code"`
}

type queryGolden struct {
	SchemaVersion int `json:"schema_version"`
	Oracle        struct {
		Beancount string `json:"beancount"`
		Beanquery string `json:"beanquery"`
	} `json:"oracle"`
	Fixture    string            `json:"fixture"`
	InputSHA   string            `json:"input_sha256"`
	QueriesSHA string            `json:"queries_sha256"`
	SourcesSHA map[string]string `json:"sources_sha256"`
	Cases      []queryGoldenCase `json:"cases"`
}

// QueryParity reports only public case IDs and mismatch categories.
type QueryParity struct {
	Checked  int            `json:"checked"`
	Failed   int            `json:"failed"`
	Failures []QueryFailure `json:"failures"`
}

type QueryFailure struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

// CheckQueryCorpus requires every registered public corpus. A missing golden
// cannot silently reduce coverage. CheckQueryGolden remains the explicit
// single-file entry for inspection and detector fault injection.
func CheckQueryCorpus(root string) (QueryParity, error) {
	result := QueryParity{Failures: make([]QueryFailure, 0)}
	for _, name := range []string{"default-source", "compound-values", "price-weight", "inventory-order", "metadata-source", "metadata-stack", "metadata-pad", "dynamic-values", "casts", "compound-casts", "aggregates"} {
		part, err := CheckQueryGolden(root, filepath.Join(root, "testdata/golden/query-v1", name+".json"))
		if err != nil {
			return result, fmt.Errorf("%s: %w", name, err)
		}
		result.Checked += part.Checked
		result.Failed += part.Failed
		for _, failure := range part.Failures {
			failure.ID = name + "/" + failure.ID
			result.Failures = append(result.Failures, failure)
		}
	}
	return result, nil
}

// CheckQueryGolden checks the reviewed public fixture without Python or writes.
// The fixture must reside under testdata/fixtures/query, including symlink resolution.
func CheckQueryGolden(root, goldenPath string) (QueryParity, error) {
	result := QueryParity{Failures: make([]QueryFailure, 0)}
	encoded, err := os.ReadFile(goldenPath)
	if err != nil {
		return result, err
	}
	var golden queryGolden
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&golden); err != nil {
		return result, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return result, fmt.Errorf("query golden must contain exactly one JSON document")
	}
	if golden.SchemaVersion != 2 || golden.Oracle.Beancount != "3.2.3" || golden.Oracle.Beanquery != "0.2.0" {
		return result, fmt.Errorf("unsupported query oracle identity")
	}
	allowed, err := filepath.EvalSymlinks(filepath.Join(root, "testdata/fixtures/query"))
	if err != nil {
		return result, err
	}
	allowed, err = filepath.Abs(allowed)
	if err != nil {
		return result, err
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, golden.Fixture))
	if err != nil {
		return result, err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return result, err
	}
	if !strings.HasPrefix(path, allowed+string(filepath.Separator)) {
		return result, fmt.Errorf("query oracle fixture outside public query corpus")
	}
	input, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	if hashQueryInput(input) != golden.InputSHA {
		return result, fmt.Errorf("query fixture identity mismatch")
	}
	manifestData, err := os.ReadFile(strings.TrimSuffix(path, filepath.Ext(path)) + ".queries.json")
	if err != nil {
		return result, err
	}
	if hashQueryInput(manifestData) != golden.QueriesSHA {
		return result, fmt.Errorf("query manifest identity mismatch")
	}
	var manifest []queryGoldenCase
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return result, err
	}
	if len(manifest) == 0 || len(manifest) != len(golden.Cases) {
		return result, fmt.Errorf("query oracle case coverage mismatch")
	}
	seen := make(map[string]bool)
	for i, item := range golden.Cases {
		if item.ID == "" || seen[item.ID] || manifest[i].ID != item.ID || manifest[i].Query != item.Query || !reflect.DeepEqual(manifest[i].SourceColumns, item.SourceColumns) || !reflect.DeepEqual(manifest[i].Error, item.Error) {
			return result, fmt.Errorf("query oracle case identity mismatch")
		}
		if (item.Result == nil) == (item.Error == nil) {
			return result, fmt.Errorf("query oracle must specify exactly one result or error")
		}
		seen[item.ID] = true
	}
	build := snapshot.BuildWithOptions(path, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	if build.Err != nil || build.Snapshot == nil {
		return result, fmt.Errorf("public query fixture build failed")
	}
	sources := make(map[string]string)
	publicRoot := filepath.Dir(filepath.Dir(filepath.Dir(allowed)))
	for _, file := range build.Graph.Files {
		path, err := filepath.EvalSymlinks(file.Path)
		if err != nil || !strings.HasPrefix(path, allowed+string(filepath.Separator)) {
			return result, fmt.Errorf("query include source outside public corpus")
		}
		rel, err := filepath.Rel(publicRoot, path)
		if err != nil {
			return result, err
		}
		sources[filepath.ToSlash(rel)] = hashQueryInput(file.Data)
	}
	if !reflect.DeepEqual(sources, golden.SourcesSHA) {
		return result, fmt.Errorf("query source graph identity mismatch")
	}
	for _, item := range golden.Cases {
		result.Checked++
		value, err := query.EvaluateTyped(item.Query, build.Snapshot.Evaluation())
		if item.Error != nil {
			var typed query.TypedError
			if !errors.As(err, &typed) || typed.Code != item.Error.Code {
				result.Failures = append(result.Failures, QueryFailure{item.ID, "error-category-mismatch"})
				result.Failed++
			}
			continue
		}
		if err != nil {
			result.Failures = append(result.Failures, QueryFailure{item.ID, "execution-error"})
			result.Failed++
			continue
		}
		actual, err := normalizeQueryResult(value)
		if err == nil {
			err = normalizeQuerySources(&actual, item.SourceColumns, allowed)
		}
		if err != nil || !reflect.DeepEqual(actual, *item.Result) {
			result.Failures = append(result.Failures, QueryFailure{item.ID, "typed-result-mismatch"})
			result.Failed++
		}
	}
	return result, nil
}

func normalizeQuerySources(result *queryResult, indices []int, root string) error {
	for _, index := range indices {
		if index < 0 || index >= len(result.Columns) || (result.Columns[index].Type != "string" && result.Columns[index].Type != "object") {
			return fmt.Errorf("invalid declared source column")
		}
		for _, row := range result.Rows {
			if row[index].Type == "null" {
				continue
			}
			path, ok := row[index].Value.(string)
			if !ok || !filepath.IsAbs(path) {
				return fmt.Errorf("invalid source path")
			}
			rel, err := filepath.Rel(root, path)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return fmt.Errorf("source outside public corpus")
			}
			row[index].Value = filepath.ToSlash(rel)
		}
	}
	return nil
}

func hashQueryInput(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func normalizeQueryResult(value query.TypedResult) (queryResult, error) {
	result := queryResult{Columns: make([]queryColumn, 0, len(value.Columns)), Rows: make([][]query.Cell, 0, len(value.Rows))}
	for i, col := range value.Columns {
		if col.ID != fmt.Sprintf("c%d", i) {
			return result, fmt.Errorf("invalid column position identity")
		}
		result.Columns = append(result.Columns, queryColumn{col.Name, col.Type})
	}
	for _, row := range value.Rows {
		if len(row) != len(value.Columns) {
			return result, fmt.Errorf("invalid cell count")
		}
		cells := make([]query.Cell, len(row))
		for i, cell := range row {
			if cell.Type == "null" {
				if !value.Columns[i].Nullable {
					return result, fmt.Errorf("unexpected null")
				}
			} else if cell.Type != value.Columns[i].Type && value.Columns[i].Type != "object" {
				return result, fmt.Errorf("schema type mismatch")
			}
			cell.Value = normalizeQueryCellValue(cell.Value)
			cells[i] = cell
		}
		result.Rows = append(result.Rows, cells)
	}
	// Decode the normalized wire shape exactly as reviewed golden cells are
	// decoded. This preserves required nulls and empty arrays, not Go pointers.
	encoded, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	var wire queryResult
	err = json.Unmarshal(encoded, &wire)
	return wire, err
}

func normalizeQueryCellValue(value any) any {
	switch value := value.(type) {
	case query.PresentedDecimal:
		return value.Exact
	case query.PresentedAmount:
		return map[string]any{"number": value.Number.Exact, "currency": value.Currency}
	case query.PresentedPosition:
		result := map[string]any{"units": normalizeQueryCellValue(value.Units), "cost": nil}
		if cost := value.Cost; cost != nil {
			result["cost"] = map[string]any{"number": cost.Number.Exact, "currency": cost.Currency, "date": cost.Date, "label": cost.Label}
		}
		return result
	case []query.PresentedPosition:
		result := make([]any, 0, len(value))
		for _, position := range value {
			result = append(result, normalizeQueryCellValue(position))
		}
		return result
	default:
		return value
	}
}
