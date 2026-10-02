// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"

	"orangecount/internal/analysis"
	"orangecount/internal/query"
	"orangecount/internal/report"
)

var ErrExpiredCapture = errors.New("report projection expired; read the view again")

type ReportExportIdentity struct {
	ReturnedRows      int    `json:"returned_rows"`
	SourceRecordCount int    `json:"source_record_count"`
	Truncated         bool   `json:"truncated"`
	DateFrom          string `json:"date_from"`
	DateTo            string `json:"date_to"`
	SemanticsProfile  string `json:"semantics_profile"`
	Completeness      string `json:"completeness"`
	ProjectionID      string `json:"projection_id"`
	SnapshotID        string `json:"snapshot_id"`
	Section           string `json:"section"`
	CaptureScope      string `json:"capture_scope"`
}
type ReportExportRequest struct {
	CaptureID    string `json:"capture_id,omitempty"`
	ProjectionID string `json:"projection_id"`
	SnapshotID   string `json:"snapshot_id"`
	Format       string `json:"format"`
	CaptureScope string `json:"capture_scope"`
}
type ReportExportCapture struct {
	Identity       ReportExportIdentity `json:"identity"`
	Format         string               `json:"format"`
	Content        []byte               `json:"content"`
	ProtectedPaths []string             `json:"protected_paths"`
}
type cachedReport struct {
	document []byte
	identity ReportExportIdentity
	paths    []string
}

func (s *Session) retainReport(result *ViewResponse) error {
	var d analysis.Document
	switch {
	case result.Tree != nil:
		d = treeAnalysis(*result)
	case result.Trends != nil:
		d = historyAnalysis(*result)
	case result.Journal != nil && result.Account == nil:
		var err error
		d, err = journalAnalysis(*result)
		if err != nil {
			return err
		}
	case result.Account != nil || result.Prices != nil || result.Holdings != nil:
		var err error
		d, err = recordAnalysis(*result)
		if err != nil {
			return err
		}
	default:
		return nil
	}
	d.Context.NormalizedRequest["sort"] = result.Request.Sort
	if d.Context.Page != nil {
		d.Context.NormalizedRequest["offset"] = strconv.Itoa(d.Context.Page.Offset)
		d.Context.NormalizedRequest["limit"] = strconv.Itoa(d.Context.Page.Limit)
	}
	request, _ := json.Marshal(d.Context.NormalizedRequest)
	hash := sha256.Sum256(append([]byte(result.State.SnapshotID+"\x00"), request...))
	id := hex.EncodeToString(hash[:])
	d.Context.ProjectionID = id
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	if len(raw) > analysis.ByteLimit {
		return nil
	}
	if err = d.Validate(); err != nil {
		if errors.Is(err, analysis.ErrBudget) {
			return nil
		}
		return err
	}
	identity := ReportExportIdentity{ReturnedRows: d.Context.ReturnedRows, SourceRecordCount: d.Context.SourceRecordCount, Truncated: d.Context.Truncated, DateFrom: d.Context.NormalizedRequest["date_from"], DateTo: d.Context.NormalizedRequest["date_to"], SemanticsProfile: d.Context.SemanticsProfile, Completeness: d.Context.Completeness, ProjectionID: id, SnapshotID: result.State.SnapshotID, Section: d.Context.Section, CaptureScope: d.Context.CaptureScope}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.state.SnapshotID != result.State.SnapshotID {
		return ErrStale
	}
	if s.reportCaptures == nil {
		s.reportCaptures = make(map[string]cachedReport)
	}
	if old, ok := s.reportCaptures[id]; ok {
		s.reportCaptureBytes -= len(old.document)
		for i, key := range s.reportCaptureOrder {
			if key == id {
				s.reportCaptureOrder = append(s.reportCaptureOrder[:i], s.reportCaptureOrder[i+1:]...)
				break
			}
		}
	}
	for len(s.reportCaptureOrder) > 0 && (len(s.reportCaptureOrder) >= 16 || s.reportCaptureBytes+len(raw) > analysis.ByteLimit) {
		key := s.reportCaptureOrder[0]
		s.reportCaptureOrder = s.reportCaptureOrder[1:]
		s.reportCaptureBytes -= len(s.reportCaptures[key].document)
		delete(s.reportCaptures, key)
	}
	s.reportCaptures[id] = cachedReport{document: raw, identity: identity, paths: append([]string{}, s.current.protectedPaths...)}
	s.reportCaptureOrder = append(s.reportCaptureOrder, id)
	s.reportCaptureBytes += len(raw)
	result.ReportExport = &identity
	return nil
}
func (s *Session) CaptureReportExport(request ReportExportRequest) (ReportExportCapture, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if request.CaptureID != "" {
		if len(request.CaptureID) > 128 {
			return ReportExportCapture{}, ErrRequest
		}
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return ReportExportCapture{}, ErrClosed
		}
		if s.reportWork == nil {
			s.reportWork = make(map[string]context.CancelFunc)
		}
		if s.cancelledReports[request.CaptureID] {
			s.mu.Unlock()
			return ReportExportCapture{}, context.Canceled
		}
		if _, exists := s.reportWork[request.CaptureID]; exists || len(s.reportWork) >= 16 {
			s.mu.Unlock()
			return ReportExportCapture{}, ErrRequest
		}
		s.reportWork[request.CaptureID] = cancel
		s.mu.Unlock()
		defer func() { s.mu.Lock(); delete(s.reportWork, request.CaptureID); s.mu.Unlock() }()
	}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ReportExportCapture{}, ErrClosed
	}
	if request.SnapshotID != s.state.SnapshotID {
		s.mu.RUnlock()
		return ReportExportCapture{}, ErrStale
	}
	if request.Format != "analysis" && request.Format != "csv" {
		s.mu.RUnlock()
		return ReportExportCapture{}, ErrRequest
	}
	capture, ok := s.reportCaptures[request.ProjectionID]
	s.mu.RUnlock()
	if !ok {
		return ReportExportCapture{}, ErrExpiredCapture
	}
	if capture.identity.CaptureScope != request.CaptureScope || capture.identity.SnapshotID != request.SnapshotID {
		return ReportExportCapture{}, ErrRequest
	}
	d, decodeErr := analysis.DecodeDocument(capture.document)
	if decodeErr != nil {
		return ReportExportCapture{}, decodeErr
	}
	var b bytes.Buffer
	var err error
	if request.Format == "analysis" {
		err = d.WriteContext(ctx, &b)
	} else {
		err = d.WriteCSV(&cancelledReportWriter{ctx: ctx, out: &queryExportWriter{output: &b, remaining: analysis.ByteLimit}})
	}
	if err != nil {
		return ReportExportCapture{}, err
	}
	if err := ctx.Err(); err != nil {
		return ReportExportCapture{}, err
	}
	return ReportExportCapture{Identity: capture.identity, Format: request.Format, Content: b.Bytes(), ProtectedPaths: append([]string{}, capture.paths...)}, nil
}
func treeAnalysis(response ViewResponse) analysis.Document {
	tree := response.Tree
	r := response.Request
	normalized := map[string]string{"view": r.View, "section": "tree", "date_from": tree.DateFrom, "date_to": tree.DateTo, "currency": tree.Currency, "valuation": tree.ValuationStatus.Mode}
	d := analysis.Document{Context: analysis.Context{FormatVersion: 2, Producer: "OrangeCount", Contract: "native-report-analysis-v1", View: r.View, Section: "tree", SnapshotID: response.State.SnapshotID, SemanticsProfile: string(response.State.SemanticsProfile), EvaluationRevision: response.State.EvaluationRevision, NormalizedRequest: normalized, CaptureScope: "complete-projection", Completeness: tree.ValuationStatus.Completeness, ValuationStatus: &tree.ValuationStatus, ValuationDetails: map[string]report.ValuationStatus{}}, Schema: analysis.Schema{RowUnit: "account", PageUnit: "account"}, Rows: [][]query.Cell{}}
	c := analysis.NewColumn
	statusColumns := []analysis.Column{c("mode", "Mode", "string", false), c("currency", "Currency", "string", false), c("as_of", "As of", "string", false), c("completeness", "Completeness", "string", false), c("covered", "Covered", "integer", false), c("missing", "Missing", "integer", false)}
	d.Schema.Columns = []analysis.Column{c("account", "Account", "string", false), c("parent", "Parent", "string", false), c("depth", "Depth", "integer", false), c("explicit", "Explicit", "bool", false), c("direct_inventory", "Direct inventory", "inventory", false), c("subtree_inventory", "Subtree inventory", "inventory", false), c("direct_valued_portion", "Direct valued portion", "decimal", true), c("direct_complete_total", "Direct complete total", "decimal", true), c("subtree_valued_portion", "Subtree valued portion", "decimal", true), c("subtree_complete_total", "Subtree complete total", "decimal", true), analysis.RecordColumn("direct_status", "Direct status", statusColumns, true), analysis.RecordColumn("subtree_status", "Subtree status", statusColumns, true)}
	amountCell := func(v *report.PresentedDecimal) query.Cell {
		if v == nil {
			return query.Cell{Type: "null"}
		}
		return query.Cell{Type: "decimal", Value: *v}
	}
	valuationCells := func(v *report.AccountValuation) []query.Cell {
		if v == nil {
			return []query.Cell{{Type: "null"}, {Type: "null"}, {Type: "null"}}
		}
		status := []query.Cell{{Type: "string", Value: v.Mode}, {Type: "string", Value: v.TargetCurrency}, {Type: "string", Value: v.AsOf}, {Type: "string", Value: v.Completeness}, {Type: "integer", Value: strconv.Itoa(v.CoveredItemCount)}, {Type: "integer", Value: strconv.Itoa(v.MissingItemCount)}}
		return []query.Cell{amountCell(v.ValuedPortion), amountCell(v.CompleteTotal), {Type: "record", Value: status}}
	}
	inventory := func(amounts map[string]report.PresentedDecimal) []query.PresentedPosition {
		currencies := make([]string, 0, len(amounts))
		for currency := range amounts {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)
		positions := []query.PresentedPosition{}
		for _, currency := range currencies {
			positions = append(positions, query.PresentedPosition{Units: query.PresentedAmount{Number: amounts[currency], Currency: currency}})
		}
		return positions
	}
	var walk func([]report.AccountTreeNode, string)
	walk = func(nodes []report.AccountTreeNode, parent string) {
		for _, n := range nodes {
			direct, subtree := valuationCells(n.DirectValuation), valuationCells(n.SubtreeValuation)
			row := []query.Cell{{Type: "string", Value: n.FullName}, {Type: "string", Value: parent}, {Type: "integer", Value: strconv.Itoa(n.Depth)}, {Type: "bool", Value: n.Explicit}, {Type: "inventory", Value: inventory(n.DirectAmounts)}, {Type: "inventory", Value: inventory(n.SubtreeAmounts)}, direct[0], direct[1], subtree[0], subtree[1], direct[2], subtree[2]}
			d.Rows = append(d.Rows, row)
			if n.DirectValuation != nil {
				d.Context.ValuationDetails[n.FullName+"/direct"] = n.DirectValuation.ValuationStatus
			}
			if n.SubtreeValuation != nil {
				d.Context.ValuationDetails[n.FullName+"/subtree"] = n.SubtreeValuation.ValuationStatus
			}
			walk(n.Children, n.FullName)
		}
	}
	walk(tree.RootNodes, "")
	d.Context.ReturnedRows = len(d.Rows)
	d.Context.SourceRecordCount = len(d.Rows)
	return d
}

func (s *Session) CancelReportExport(id string) error {
	if id == "" || len(id) > 128 {
		return ErrRequest
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.cancelledReports == nil {
		s.cancelledReports = map[string]bool{}
	}
	if !s.cancelledReports[id] {
		s.cancelledReports[id] = true
		s.cancelledReportOrder = append(s.cancelledReportOrder, id)
	}
	if len(s.cancelledReportOrder) > 32 {
		delete(s.cancelledReports, s.cancelledReportOrder[0])
		s.cancelledReportOrder = s.cancelledReportOrder[1:]
	}
	if cancel := s.reportWork[id]; cancel != nil {
		cancel()
	}
	return nil
}

type cancelledReportWriter struct {
	ctx context.Context
	out interface{ Write([]byte) (int, error) }
}

func (w *cancelledReportWriter) Write(data []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.out.Write(data)
}
