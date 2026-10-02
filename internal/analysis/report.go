// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

// Package analysis reads and writes immutable, bounded accounting captures.
package analysis

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"unicode/utf8"

	"orangecount/internal/query"
	"orangecount/internal/report"
)

const ByteLimit = 16 * 1024 * 1024
const RowLimit = 10000

var ErrInvalid = errors.New("invalid report analysis")
var ErrBudget = errors.New("analysis resource limit exceeded")

type Page struct {
	TotalCount    int  `json:"total_count"`
	Offset        int  `json:"offset"`
	Limit         int  `json:"limit"`
	ReturnedCount int  `json:"returned_count"`
	HasMore       bool `json:"has_more"`
}
type Supplement struct {
	Columns []Column     `json:"columns"`
	Cells   []query.Cell `json:"cells"`
}
type Context struct {
	Supplement         *Supplement                       `json:"supplement,omitempty"`
	FormatVersion      int                               `json:"format_version"`
	Producer           string                            `json:"producer"`
	Contract           string                            `json:"contract"`
	View               string                            `json:"view"`
	Section            string                            `json:"section"`
	SnapshotID         string                            `json:"snapshot_id"`
	ProjectionID       string                            `json:"projection_id"`
	SemanticsProfile   string                            `json:"semantics_profile"`
	EvaluationRevision string                            `json:"evaluation_revision"`
	NormalizedRequest  map[string]string                 `json:"normalized_request"`
	CaptureScope       string                            `json:"capture_scope"`
	ReturnedRows       int                               `json:"returned_rows"`
	SourceRecordCount  int                               `json:"source_record_count"`
	Truncated          bool                              `json:"truncated"`
	Completeness       string                            `json:"completeness"`
	Page               *Page                             `json:"page"`
	ValuationDetails   map[string]report.ValuationStatus `json:"valuation_details"`
	ValuationStatus    *report.ValuationStatus           `json:"valuation_status"`
}
type Column struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Nullable bool      `json:"nullable"`
	Fields   *[]Column `json:"fields"`
}
type Schema struct {
	RowUnit  string   `json:"row_unit"`
	PageUnit string   `json:"page_unit"`
	Columns  []Column `json:"columns"`
}
type Document struct {
	Context Context        `json:"context"`
	Schema  Schema         `json:"schema"`
	Rows    [][]query.Cell `json:"rows"`
}

// Members validates ZIP structure before handing any contents to a decoder.
// It is also used for version dispatch; it never extracts filesystem paths.
func Members(input io.ReaderAt, size int64) (map[string][]byte, error) {
	if size <= 0 || size > ByteLimit {
		return nil, ErrBudget
	}
	z, err := zip.NewReader(input, size)
	if err != nil || len(z.File) != 4 {
		return nil, ErrInvalid
	}
	result := map[string][]byte{}
	var total uint64
	for _, f := range z.File {
		if !slices.Contains([]string{"context.json", "schema.json", "cells.ndjson", "data.csv"}, f.Name) || !f.Mode().IsRegular() {
			return nil, ErrInvalid
		}
		if _, ok := result[f.Name]; ok {
			return nil, ErrInvalid
		}
		total += f.UncompressedSize64
		if f.UncompressedSize64 > 2*ByteLimit || total > 4*ByteLimit {
			return nil, ErrBudget
		}
		r, err := f.Open()
		if err != nil {
			return nil, ErrInvalid
		}
		data, err := io.ReadAll(io.LimitReader(r, 2*ByteLimit+1))
		closeErr := r.Close()
		if err != nil || closeErr != nil || !utf8.Valid(data) {
			return nil, ErrInvalid
		}
		if len(data) > 2*ByteLimit {
			return nil, ErrBudget
		}
		result[f.Name] = data
	}
	return result, nil
}
func Version(input io.ReaderAt, size int64) (int, error) {
	members, err := Members(input, size)
	if err != nil {
		return 0, err
	}
	var c map[string]json.RawMessage
	if err := StrictJSON(members["context.json"], &c); err != nil {
		return 0, ErrInvalid
	}
	var version int
	if json.Unmarshal(c["format_version"], &version) != nil {
		return 0, ErrInvalid
	}
	return version, nil
}
func ReadReport(input io.ReaderAt, size int64) (Document, error) {
	var d Document
	members, err := Members(input, size)
	if err != nil {
		return d, err
	}
	if StrictJSON(members["context.json"], &d.Context) != nil || StrictJSON(members["schema.json"], &d.Schema) != nil {
		return Document{}, ErrInvalid
	}
	if err := d.validateHeader(); err != nil {
		return Document{}, err
	}
	if err := restoreSupplement(&d.Context); err != nil {
		return Document{}, err
	}
	d.Rows = make([][]query.Cell, 0)
	scanner := bufio.NewScanner(bytes.NewReader(members["cells.ndjson"]))
	scanner.Buffer(make([]byte, 4096), 2*ByteLimit)
	for scanner.Scan() {
		if len(d.Rows) >= d.Context.ReturnedRows {
			return Document{}, ErrInvalid
		}
		var wire []json.RawMessage
		if StrictJSON(scanner.Bytes(), &wire) != nil || len(wire) != len(d.Schema.Columns) {
			return Document{}, ErrInvalid
		}
		row := make([]query.Cell, len(wire))
		for i, raw := range wire {
			cell, err := decodeCell(raw, d.Schema.Columns[i], 0)
			if err != nil {
				return Document{}, fmt.Errorf("%w: row %d column %d", err, len(d.Rows), i)
			}
			row[i] = cell
		}
		d.Rows = append(d.Rows, row)
	}
	if scanner.Err() != nil || len(d.Rows) != d.Context.ReturnedRows {
		return Document{}, ErrInvalid
	}
	if err = d.Validate(); err != nil {
		return Document{}, err
	}
	var canonical bytes.Buffer
	if err = d.WriteCSV(&canonical); err != nil || !bytes.Equal(canonical.Bytes(), members["data.csv"]) {
		return Document{}, ErrInvalid
	}
	return d, nil
}
func (d Document) validateHeader() error {
	c := d.Context
	if c.ReturnedRows > RowLimit || len(d.Schema.Columns) > 4096 {
		return ErrBudget
	}
	if c.FormatVersion != 2 || c.Contract != "native-report-analysis-v1" || c.Producer == "" || c.SnapshotID == "" || c.ProjectionID == "" || c.Section == "" || c.EvaluationRevision == "" || !slices.Contains([]string{"legacy", "v3"}, c.SemanticsProfile) {
		return ErrInvalid
	}
	if !slices.Contains([]string{"balance_sheet", "income_statement", "dashboard", "account", "journal", "holdings", "prices"}, c.View) || !slices.Contains([]string{"complete", "partial", "unavailable", "empty", "not-applicable"}, c.Completeness) {
		return ErrInvalid
	}
	if !slices.Contains([]string{"complete-projection", "current-page"}, c.CaptureScope) || c.ReturnedRows < 0 || c.SourceRecordCount < 0 {
		return ErrInvalid
	}
	if c.NormalizedRequest == nil || d.Schema.Columns == nil || len(d.Schema.Columns) > 4096 || d.Schema.RowUnit == "" || d.Schema.PageUnit == "" {
		return ErrInvalid
	}
	allowed := []string{"view", "section", "date_from", "date_to", "currency", "valuation", "account", "scope", "group_by", "text", "sort", "offset", "limit", "price_base", "price_quote", "holding_section", "group_id", "event_id", "lot_kind"}
	for key, value := range c.NormalizedRequest {
		if !slices.Contains(allowed, key) || len(value) > 65536 || !utf8.ValidString(value) {
			return ErrInvalid
		}
	}
	if c.ValuationStatus != nil {
		if err := validateValuation(*c.ValuationStatus); err != nil {
			return err
		}
	}
	for _, v := range c.ValuationDetails {
		if err := validateValuation(v); err != nil {
			return err
		}
	}
	if c.Page != nil {
		p := c.Page
		if c.CaptureScope != "current-page" || p.TotalCount < 0 || p.Offset < 0 || p.Limit < 1 || p.ReturnedCount < 0 || p.ReturnedCount > p.Limit || p.ReturnedCount > p.TotalCount || p.HasMore != (p.Offset < p.TotalCount && p.ReturnedCount < p.TotalCount-p.Offset) || (p.Offset >= p.TotalCount && p.ReturnedCount > 0) || (p.Offset < p.TotalCount && p.ReturnedCount > p.TotalCount-p.Offset) {
			return ErrInvalid
		}
		if p.ReturnedCount != c.SourceRecordCount || c.Truncated != (p.ReturnedCount < p.TotalCount) {
			return ErrInvalid
		}
	} else if c.Truncated || c.SourceRecordCount != c.ReturnedRows {
		return ErrInvalid
	}
	if d.Schema.RowUnit == d.Schema.PageUnit && c.SourceRecordCount != c.ReturnedRows {
		return ErrInvalid
	}
	if d.Schema.RowUnit != d.Schema.PageUnit && (d.Schema.RowUnit != "posting" || d.Schema.PageUnit != "transaction") {
		return ErrInvalid
	}
	if c.ValuationDetails == nil {
		return ErrInvalid
	}
	if c.Supplement != nil {
		if err := validateColumns(c.Supplement.Columns, 0); err != nil {
			return err
		}
		if len(c.Supplement.Columns) != len(c.Supplement.Cells) {
			return ErrInvalid
		}
		for i, cell := range c.Supplement.Cells {
			raw, err := json.Marshal(cell)
			if err != nil {
				return ErrInvalid
			}
			if _, err = decodeCell(raw, c.Supplement.Columns[i], 0); err != nil {
				return err
			}
		}
	}
	return validateColumns(d.Schema.Columns, 0)
}
func validateColumns(columns []Column, depth int) error {
	if depth > 8 || len(columns) > 4096 {
		return ErrBudget
	}
	ids := map[string]bool{}
	for _, c := range columns {
		if c.ID == "" || len(c.ID) > 65536 || len(c.Name) > 65536 || ids[c.ID] {
			return ErrInvalid
		}
		ids[c.ID] = true
		if c.Type == "record" || c.Type == "list<record>" {
			if c.Fields == nil {
				return ErrInvalid
			}
			if err := validateColumns(*c.Fields, depth+1); err != nil {
				return err
			}
		} else {
			if c.Fields != nil || !slices.Contains([]string{"string", "integer", "bool", "date", "decimal", "amount", "position", "inventory", "set<string>"}, c.Type) {
				return ErrInvalid
			}
		}
	}
	return nil
}
func decodeCell(raw []byte, column Column, depth int) (query.Cell, error) {
	if depth > 8 {
		return query.Cell{}, ErrBudget
	}
	if len(raw) > 2*ByteLimit {
		return query.Cell{}, ErrBudget
	}
	var wire struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	}
	if StrictJSON(raw, &wire) != nil {
		return query.Cell{}, ErrInvalid
	}
	if wire.Type == "null" {
		if !column.Nullable || len(wire.Value) != 0 {
			return query.Cell{}, ErrInvalid
		}
		return query.Cell{Type: "null"}, nil
	}
	if wire.Type != column.Type || len(wire.Value) == 0 || bytes.Equal(wire.Value, []byte("null")) {
		return query.Cell{}, ErrInvalid
	}
	cell := query.Cell{Type: wire.Type}
	if wire.Type == "list<record>" {
		var values []json.RawMessage
		if StrictJSON(wire.Value, &values) != nil || values == nil || len(values) > RowLimit {
			return cell, ErrInvalid
		}
		items := make([][]query.Cell, len(values))
		element := column
		element.Type = "record"
		for i, value := range values {
			wrapped, _ := json.Marshal(struct {
				Type  string          `json:"type"`
				Value json.RawMessage `json:"value"`
			}{"record", value})
			decoded, err := decodeCell(wrapped, element, depth+1)
			if err != nil {
				return cell, err
			}
			items[i] = decoded.Value.([]query.Cell)
		}
		cell.Value = items
		return cell, nil
	}
	if wire.Type == "record" {
		var values []json.RawMessage
		if StrictJSON(wire.Value, &values) != nil || column.Fields == nil || len(values) != len(*column.Fields) {
			return cell, ErrInvalid
		}
		children := make([]query.Cell, len(values))
		for i, value := range values {
			v, err := decodeCell(value, (*column.Fields)[i], depth+1)
			if err != nil {
				return cell, err
			}
			children[i] = v
		}
		cell.Value = children
		if err := validateValuationRecord(column, children); err != nil {
			return cell, err
		}
		return cell, nil
	}
	if wire.Type == "set<string>" {
		var values []string
		if StrictJSON(wire.Value, &values) != nil || values == nil || !slices.IsSorted(values) {
			return cell, ErrInvalid
		}
		for i, v := range values {
			if len(v) > 65536 || (i > 0 && values[i-1] == v) {
				return cell, ErrInvalid
			}
		}
		cell.Value = values
		return cell, nil
	}
	var target any
	switch wire.Type {
	case "string", "integer", "date":
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
		return cell, ErrInvalid
	}
	if StrictJSON(wire.Value, target) != nil {
		return cell, ErrInvalid
	}
	value := reflect.ValueOf(target).Elem().Interface()
	if text, ok := value.(string); ok && len(text) > 65536 {
		return cell, ErrBudget
	}
	if !ValidValue(value, wire.Type) {
		return cell, ErrInvalid
	}
	cell.Value = value
	return cell, nil
}
func (d Document) Validate() error { return d.ValidateContext(context.Background()) }
func (d Document) ValidateContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.validateHeader(); err != nil {
		return err
	}
	if d.Rows == nil || len(d.Rows) != d.Context.ReturnedRows {
		return ErrInvalid
	}
	transactions := map[string]bool{}
	for _, row := range d.Rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(row) != len(d.Schema.Columns) {
			return ErrInvalid
		}
		for i, cell := range row {
			raw, err := json.Marshal(cell)
			if err != nil {
				return ErrInvalid
			}
			if _, err = decodeCell(raw, d.Schema.Columns[i], 0); err != nil {
				return err
			}
			if d.Schema.RowUnit == "posting" && d.Schema.Columns[i].ID == "transaction_id" {
				if cell.Type != "string" {
					return ErrInvalid
				}
				transactions[cell.Value.(string)] = true
			}
		}
	}
	if d.Schema.RowUnit == "posting" && len(transactions) != d.Context.SourceRecordCount {
		return ErrInvalid
	}
	return nil
}
func (d Document) WriteCSV(out io.Writer) error {
	w := csv.NewWriter(out)
	w.UseCRLF = true
	headers := make([]string, len(d.Schema.Columns))
	for i, c := range d.Schema.Columns {
		headers[i] = c.Name
	}
	if err := w.Write(headers); err != nil {
		return err
	}
	for _, row := range d.Rows {
		values := make([]string, len(row))
		for i, c := range row {
			switch v := c.Value.(type) {
			case nil:
				values[i] = ""
			case string:
				values[i] = v
			case bool:
				if v {
					values[i] = "true"
				} else {
					values[i] = "false"
				}
			case query.PresentedDecimal:
				values[i] = v.Exact
			default:
				raw, err := json.Marshal(v)
				if err != nil {
					return err
				}
				values[i] = string(raw)
			}
		}
		if err := w.Write(values); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

type limitWriter struct {
	out  io.Writer
	left int
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if len(p) > w.left {
		return 0, ErrBudget
	}
	n, err := w.out.Write(p)
	w.left -= n
	return n, err
}
func (d Document) Write(out io.Writer) error { return d.WriteContext(context.Background(), out) }
func (d Document) WriteContext(ctx context.Context, out io.Writer) error {
	if err := d.ValidateContext(ctx); err != nil {
		return err
	}
	limited := &limitWriter{out: contextWriter{ctx: ctx, out: out}, left: ByteLimit}
	archive := zip.NewWriter(limited)
	total := 4 * ByteLimit
	member := func(name string, write func(io.Writer) error) error {
		w, err := archive.Create(name)
		if err != nil {
			return err
		}
		limit := &limitWriter{out: w, left: min(2*ByteLimit, total)}
		before := limit.left
		err = write(limit)
		total -= before - limit.left
		return err
	}
	jsonMember := func(name string, value any) error {
		return member(name, func(w io.Writer) error { e := json.NewEncoder(w); e.SetEscapeHTML(false); return e.Encode(value) })
	}
	if err := jsonMember("context.json", d.Context); err != nil {
		return err
	}
	if err := jsonMember("schema.json", d.Schema); err != nil {
		return err
	}
	if err := member("cells.ndjson", func(w io.Writer) error {
		e := json.NewEncoder(w)
		e.SetEscapeHTML(false)
		for _, row := range d.Rows {
			if err := e.Encode(row); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if err := member("data.csv", d.WriteCSV); err != nil {
		return err
	}
	if err := archive.Close(); err != nil {
		return err
	}
	return ctx.Err()
}

// NewColumn constructs a primitive column with an explicitly null field schema.
func NewColumn(id, name, kind string, nullable bool) Column {
	return Column{ID: id, Name: name, Type: kind, Nullable: nullable}
}
func RecordColumn(id, name string, fields []Column, nullable bool) Column {
	return Column{ID: id, Name: name, Type: "record", Nullable: nullable, Fields: &fields}
}
func ErrorCode(err error) string {
	if errors.Is(err, ErrBudget) {
		return "resource-limit"
	}
	return "invalid-analysis"
}

// NormalizeSet keeps a set distinct from an ordered sequence.
func NormalizeSet(values []string) []string {
	out := append([]string{}, values...)
	slices.Sort(out)
	return slices.Compact(out)
}

// DecodeDocument restores concrete typed cells from a frozen JSON capture.
func DecodeDocument(raw []byte) (Document, error) {
	var wire struct {
		Context Context             `json:"context"`
		Schema  Schema              `json:"schema"`
		Rows    [][]json.RawMessage `json:"rows"`
	}
	if err := StrictJSON(raw, &wire); err != nil {
		return Document{}, ErrInvalid
	}
	d := Document{Context: wire.Context, Schema: wire.Schema, Rows: make([][]query.Cell, 0, len(wire.Rows))}
	if err := restoreSupplement(&d.Context); err != nil {
		return Document{}, err
	}
	if err := d.validateHeader(); err != nil {
		return Document{}, err
	}
	for _, values := range wire.Rows {
		if len(values) != len(d.Schema.Columns) {
			return Document{}, ErrInvalid
		}
		row := make([]query.Cell, len(values))
		for i, value := range values {
			cell, err := decodeCell(value, d.Schema.Columns[i], 0)
			if err != nil {
				return Document{}, err
			}
			row[i] = cell
		}
		d.Rows = append(d.Rows, row)
	}
	return d, d.Validate()
}

func restoreSupplement(c *Context) error {
	if c.Supplement == nil {
		return nil
	}
	if len(c.Supplement.Columns) != len(c.Supplement.Cells) {
		return ErrInvalid
	}
	for i, cell := range c.Supplement.Cells {
		raw, err := json.Marshal(cell)
		if err != nil {
			return ErrInvalid
		}
		restored, err := decodeCell(raw, c.Supplement.Columns[i], 0)
		if err != nil {
			return err
		}
		c.Supplement.Cells[i] = restored
	}
	return nil
}

func validateValuation(v report.ValuationStatus) error {
	if !slices.Contains([]string{"complete", "partial", "unavailable", "empty"}, v.Completeness) || v.CoveredItemCount < 0 || v.MissingItemCount < 0 || v.Missing == nil || v.Quotes == nil {
		return ErrInvalid
	}
	switch v.Completeness {
	case "partial":
		if v.CoveredItemCount == 0 || v.MissingItemCount == 0 {
			return ErrInvalid
		}
	case "unavailable":
		if v.CoveredItemCount != 0 || v.MissingItemCount == 0 {
			return ErrInvalid
		}
	case "complete", "empty":
		if v.MissingItemCount != 0 {
			return ErrInvalid
		}
	}
	for _, q := range v.Quotes {
		if q.Base == "" || q.Quote == "" || !ValidValue(q.Rate, "decimal") || !ValidValue(q.Date, "date") {
			return ErrInvalid
		}
	}
	return nil
}
func validateValuationRecord(column Column, cells []query.Cell) error {
	status := ""
	total := query.Cell{Type: "null"}
	hasTotal := false
	for i, c := range *column.Fields {
		if c.ID == "complete_total" {
			total = cells[i]
			hasTotal = true
		}
		if c.ID == "ValuationStatus" && cells[i].Type == "record" {
			children := cells[i].Value.([]query.Cell)
			for j, f := range *c.Fields {
				if f.ID == "completeness" {
					status, _ = children[j].Value.(string)
				}
			}
		}
	}
	if hasTotal && (status == "partial" || status == "unavailable" || status == "empty") && total.Type != "null" {
		return ErrInvalid
	}
	return nil
}

type contextWriter struct {
	ctx context.Context
	out io.Writer
}

func (w contextWriter) Write(data []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.out.Write(data)
}
