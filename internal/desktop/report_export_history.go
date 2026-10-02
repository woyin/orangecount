// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop

import (
	"orangecount/internal/analysis"
	"orangecount/internal/query"
	"orangecount/internal/report"
	"strconv"
)

func baseAnalysis(response ViewResponse, section, unit, from, to string) analysis.Document {
	return analysis.Document{Context: analysis.Context{FormatVersion: 2, Producer: "OrangeCount", Contract: "native-report-analysis-v1", View: response.Request.View, Section: section, SnapshotID: response.State.SnapshotID, SemanticsProfile: string(response.State.SemanticsProfile), EvaluationRevision: response.State.EvaluationRevision, NormalizedRequest: map[string]string{"view": response.Request.View, "section": section, "date_from": from, "date_to": to, "currency": response.Request.Currency, "valuation": response.Request.Valuation}, CaptureScope: "complete-projection", Completeness: "empty", ValuationDetails: map[string]report.ValuationStatus{}}, Schema: analysis.Schema{RowUnit: unit, PageUnit: unit, Columns: []analysis.Column{}}, Rows: [][]query.Cell{}}
}
func decimalCell(v *report.PresentedDecimal) query.Cell {
	if v == nil {
		return query.Cell{Type: "null"}
	}
	return query.Cell{Type: "decimal", Value: *v}
}
func stringCell(v string) query.Cell { return query.Cell{Type: "string", Value: v} }
func historyAnalysis(response ViewResponse) analysis.Document {
	t := response.Trends
	d := baseAnalysis(response, "history", "history-point", t.DateFrom, t.DateTo)
	c := analysis.NewColumn
	d.Schema.Columns = []analysis.Column{c("kind", "Kind", "string", false), c("currency", "Currency", "string", false), c("date", "Date", "date", false), c("income", "Income", "decimal", true), c("expenses", "Expenses", "decimal", true), c("net", "Net", "decimal", true), c("income_portion", "Income portion", "decimal", true), c("expenses_portion", "Expenses portion", "decimal", true), c("net_portion", "Net portion", "decimal", true), c("segment", "Segment", "integer", true), c("valuation_key", "Valuation details", "string", false)}
	statuses := map[string]int{}
	add := func(kind, currency, date string, income, expenses, net *report.AccountValuation, segment *int) {
		key := strconv.Itoa(len(d.Rows))
		val := func(name string, v *report.AccountValuation) (query.Cell, query.Cell) {
			if v == nil {
				return query.Cell{Type: "null"}, query.Cell{Type: "null"}
			}
			d.Context.ValuationDetails[key+"/"+name] = v.ValuationStatus
			statuses[v.Completeness]++
			return decimalCell(v.CompleteTotal), decimalCell(v.ValuedPortion)
		}
		i, ip := val("income", income)
		e, ep := val("expenses", expenses)
		n, np := val("net", net)
		seg := query.Cell{Type: "null"}
		if segment != nil {
			seg = query.Cell{Type: "integer", Value: strconv.Itoa(*segment)}
		}
		d.Rows = append(d.Rows, []query.Cell{stringCell(kind), stringCell(currency), {Type: "date", Value: date}, i, e, n, ip, ep, np, seg, stringCell(key)})
	}
	for _, series := range t.CurrencySeries {
		for _, point := range series.NetWorthPoints {
			add("net-worth", series.Currency, point.Date, nil, nil, point.Valuation, &point.Segment)
		}
		for _, flow := range series.MonthlyCashFlows {
			add("cash-flow", series.Currency, flow.Date, flow.Income, flow.Expenses, flow.Net, nil)
		}
	}
	if statuses["partial"] > 0 || (statuses["unavailable"] > 0 && statuses["complete"] > 0) {
		d.Context.Completeness = "partial"
	} else if statuses["unavailable"] > 0 {
		d.Context.Completeness = "unavailable"
	} else if statuses["complete"] > 0 {
		d.Context.Completeness = "complete"
	}
	d.Context.ReturnedRows = len(d.Rows)
	d.Context.SourceRecordCount = len(d.Rows)
	return d
}
