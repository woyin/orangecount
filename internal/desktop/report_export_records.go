// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"orangecount/internal/analysis"
	"orangecount/internal/ledger"
	"orangecount/internal/presentation"
	"orangecount/internal/query"
	"orangecount/internal/report"
)

// Records use declared Go types, not first-row inference. Presentation floats
// and filesystem paths are excluded at this seam; financial values stay typed.
func recordSchema(t reflect.Type, id string) (analysis.Column, error) {
	nullable := t.Kind() == reflect.Pointer
	if nullable {
		t = t.Elem()
	}
	c := analysis.NewColumn(id, id, "", nullable)
	if t == reflect.TypeOf(report.PresentedDecimal{}) {
		c.Type = "decimal"
		return c, nil
	}
	if t == reflect.TypeOf(map[string]report.PresentedDecimal{}) {
		c.Type = "inventory"
		return c, nil
	}
	switch t.Kind() {
	case reflect.String:
		c.Type = "string"
		if id == "units_number" {
			c.Type = "decimal"
		}
	case reflect.Bool:
		c.Type = "bool"
	case reflect.Int, reflect.Int64, reflect.Uint32, reflect.Uint64:
		c.Type = "integer"
	case reflect.Slice:
		if t.Elem().Kind() == reflect.String {
			c.Type = "set<string>"
		} else {
			element, err := recordSchema(t.Elem(), "item")
			if err != nil || element.Type != "record" {
				return c, fmt.Errorf("unsupported analysis list type %s", t)
			}
			c.Type = "list<record>"
			c.Fields = element.Fields
		}
	case reflect.Struct:
		fields := []analysis.Column{}
		if _, ok := t.FieldByName("Path"); ok {
			fields = append(fields, analysis.NewColumn("source_id", "Source ID", "string", false))
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if excludedAnalysisField(f) {
				continue
			}
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "" {
				name = f.Name
			}
			column, err := recordSchema(f.Type, name)
			if err != nil {
				return c, err
			}
			fields = append(fields, column)
		}
		c.Type = "record"
		c.Fields = &fields
	default:
		return c, fmt.Errorf("unsupported analysis field type %s", t)
	}
	return c, nil
}
func excludedAnalysisField(f reflect.StructField) bool {
	return f.PkgPath != "" || f.Name == "Path" || f.Name == "SourceFiles" || f.Name == "DirectBalances" || f.Name == "SubtreeBalances" || f.Type.Kind() == reflect.Float64 || (f.Type.Kind() == reflect.Pointer && f.Type.Elem().Kind() == reflect.Float64) || f.Tag.Get("json") == "-"
}
func recordValue(v reflect.Value, column analysis.Column, snapshot string) (query.Cell, error) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return query.Cell{Type: "null"}, nil
		}
		v = v.Elem()
	}
	cell := query.Cell{Type: column.Type}
	if column.Type == "decimal" {
		if v.Kind() == reflect.String {
			exact, err := ledger.ParseDecimal(v.String())
			if err != nil {
				return cell, err
			}
			cell.Value = presentation.FormatDecimal(exact)
			return cell, nil
		}
		cell.Value = v.Interface()
		return cell, nil
	}
	if column.Type == "inventory" {
		amounts := v.Interface().(map[string]report.PresentedDecimal)
		keys := make([]string, 0, len(amounts))
		for key := range amounts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		values := []query.PresentedPosition{}
		for _, key := range keys {
			values = append(values, query.PresentedPosition{Units: query.PresentedAmount{Number: amounts[key], Currency: key}})
		}
		cell.Value = values
		return cell, nil
	}
	switch column.Type {
	case "string":
		cell.Value = v.String()
	case "bool":
		cell.Value = v.Bool()
	case "integer":
		if v.Kind() >= reflect.Uint && v.Kind() <= reflect.Uint64 {
			cell.Value = strconv.FormatUint(v.Uint(), 10)
		} else {
			cell.Value = strconv.FormatInt(v.Int(), 10)
		}
	case "set<string>":
		strings := []string{}
		for i := 0; i < v.Len(); i++ {
			strings = append(strings, v.Index(i).String())
		}
		cell.Value = analysis.NormalizeSet(strings)
	case "list<record>":
		items := make([][]query.Cell, 0, v.Len())
		element := column
		element.Type = "record"
		for i := 0; i < v.Len(); i++ {
			item, err := recordValue(v.Index(i), element, snapshot)
			if err != nil {
				return cell, err
			}
			items = append(items, item.Value.([]query.Cell))
		}
		cell.Value = items
	case "record":
		fields := []query.Cell{}
		offset := 0
		if path := v.FieldByName("Path"); path.IsValid() {
			hash := sha256.Sum256([]byte(snapshot + "\x00" + fmt.Sprint(v.Interface())))
			fields = append(fields, stringCell(hex.EncodeToString(hash[:])))
			offset++
		}
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if excludedAnalysisField(f) {
				continue
			}
			item, err := recordValue(v.Field(i), (*column.Fields)[offset], snapshot)
			if err != nil {
				return cell, err
			}
			fields = append(fields, item)
			offset++
		}
		cell.Value = fields
	default:
		return cell, analysis.ErrInvalid
	}
	return cell, nil
}
func addRecordRows(d *analysis.Document, items any) error {
	collectAnalysisValuations(d, reflect.ValueOf(items), "rows")
	v := reflect.ValueOf(items)
	column, err := recordSchema(v.Type().Elem(), "record")
	if err != nil {
		return err
	}
	d.Schema.Columns = []analysis.Column{column}
	for i := 0; i < v.Len(); i++ {
		cell, err := recordValue(v.Index(i), column, d.Context.SnapshotID)
		if err != nil {
			return err
		}
		d.Rows = append(d.Rows, []query.Cell{cell})
	}
	return nil
}
func addSupplement(d *analysis.Document, value any) error {
	v := reflect.ValueOf(value)
	column, err := recordSchema(v.Type(), "selection")
	if err != nil {
		return err
	}
	cell, err := recordValue(v, column, d.Context.SnapshotID)
	if err != nil {
		return err
	}
	d.Context.Supplement = &analysis.Supplement{Columns: []analysis.Column{column}, Cells: []query.Cell{cell}}
	return nil
}
func setAnalysisPage(d *analysis.Document, page report.ProjectionPage) {
	d.Context.CaptureScope = "current-page"
	d.Context.Page = &analysis.Page{TotalCount: page.TotalCount, Offset: page.Offset, Limit: page.Limit, ReturnedCount: page.ReturnedCount, HasMore: page.HasMore}
	d.Context.Truncated = page.Truncated
	d.Context.SourceRecordCount = page.ReturnedCount
}
func recordAnalysis(response ViewResponse) (analysis.Document, error) {
	var d analysis.Document
	var err error
	r := response.Request
	switch {
	case response.Account != nil:
		a := response.Account
		d = baseAnalysis(response, "overview", "account", a.DateFrom, a.DateTo)
		d.Context.NormalizedRequest["account"] = a.Account
		d.Context.NormalizedRequest["scope"] = a.Scope
		err = addRecordRows(&d, []*AccountDetailPayload{a})
		d.Context.Completeness = "not-applicable"
	case response.Prices != nil:
		p := response.Prices
		d = baseAnalysis(response, p.Section, "price", p.DateFrom, p.DateTo)
		d.Context.Completeness = "not-applicable"
		d.Context.NormalizedRequest["price_base"] = p.Base
		d.Context.NormalizedRequest["price_quote"] = p.Quote
		if p.Section == "commodities" {
			d.Schema.RowUnit = "commodity"
			d.Schema.PageUnit = "commodity"
			err = addRecordRows(&d, p.Commodities)
			setAnalysisPage(&d, p.CommoditiesPage)
		} else {
			err = addRecordRows(&d, p.Records)
			setAnalysisPage(&d, p.RecordsPage)
			if err == nil {
				err = addSupplement(&d, struct {
					Selected *report.SelectedPrice `json:"selected"`
					Reason   string                `json:"reason"`
				}{p.Selected, p.SelectedReason})
			}
		}
	case response.Holdings != nil:
		h := response.Holdings
		d = baseAnalysis(response, h.Section, "holding", h.DateFrom, h.DateTo)
		d.Context.Completeness = "not-applicable"
		d.Context.NormalizedRequest["account"] = h.Account
		d.Context.NormalizedRequest["scope"] = h.Scope
		d.Context.NormalizedRequest["group_by"] = h.GroupBy
		d.Context.NormalizedRequest["group_id"] = r.GroupID
		d.Context.NormalizedRequest["event_id"] = r.EventID
		d.Context.NormalizedRequest["lot_kind"] = r.LotKind
		switch h.Section {
		case "groups":
			err = addRecordRows(&d, h.Groups)
			setAnalysisPage(&d, h.GroupsPage)
		case "events":
			err = addRecordRows(&d, h.Events)
			setAnalysisPage(&d, h.EventsPage)
		case "history":
			if r.GroupID != "" && len(h.AverageHistory) > 0 {
				series := h.AverageHistory[0]
				err = addRecordRows(&d, series.Points)
				setAnalysisPage(&d, series.PointsPage)
				series.Points = []report.AverageCostPoint{}
				if err == nil {
					err = addSupplement(&d, series)
				}
			} else {
				err = addRecordRows(&d, h.AverageHistory)
				setAnalysisPage(&d, h.HistoryPage)
			}
		case "lots":
			if len(h.Groups) > 0 {
				err = addRecordRows(&d, h.Groups[0].Lots)
				setAnalysisPage(&d, h.Groups[0].LotsPage)
				if err == nil {
					selected := h.Groups[0]
					selected.Lots = []report.HoldingLot{}
					err = addSupplement(&d, selected)
				}
			} else if len(h.Events) > 0 {
				event := h.Events[0]
				if r.LotKind == "merged" {
					err = addRecordRows(&d, event.MergeLots)
					setAnalysisPage(&d, event.MergeLotsPage)
				} else {
					err = addRecordRows(&d, event.Lots)
					setAnalysisPage(&d, event.LotsPage)
				}
				if err == nil {
					event.Lots = []report.HoldingLot{}
					event.MergeLots = []report.HoldingLot{}
					err = addSupplement(&d, event)
				}
			} else {
				return d, ErrRequest
			}
		default:
			return d, ErrRequest
		}
	default:
		return d, ErrRequest
	}
	if err != nil {
		return d, err
	}
	statuses := map[string]int{}
	for _, v := range d.Context.ValuationDetails {
		statuses[v.Completeness]++
	}
	if statuses["partial"] > 0 || (statuses["unavailable"] > 0 && statuses["complete"] > 0) {
		d.Context.Completeness = "partial"
	} else if statuses["unavailable"] > 0 {
		d.Context.Completeness = "unavailable"
	} else if statuses["complete"] > 0 {
		d.Context.Completeness = "complete"
	} else if statuses["empty"] > 0 {
		d.Context.Completeness = "empty"
	}
	d.Context.ReturnedRows = len(d.Rows)
	if d.Context.Page == nil {
		d.Context.SourceRecordCount = len(d.Rows)
	}
	return d, nil
}

func collectAnalysisValuations(d *analysis.Document, v reflect.Value, path string) {
	if !v.IsValid() {
		return
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Type() == reflect.TypeOf(report.AccountValuation{}) {
		d.Context.ValuationDetails[path] = v.Interface().(report.AccountValuation).ValuationStatus
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if !excludedAnalysisField(f) {
				collectAnalysisValuations(d, v.Field(i), path+"/"+f.Name)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			collectAnalysisValuations(d, v.Index(i), path+"/"+strconv.Itoa(i))
		}
	}
}
