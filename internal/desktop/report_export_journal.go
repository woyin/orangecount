// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop

import (
	"orangecount/internal/analysis"
	"orangecount/internal/query"
	"orangecount/internal/report"
	"reflect"
	"strconv"
)

func journalAnalysis(response ViewResponse) (analysis.Document, error) {
	p := response.Journal
	d := baseAnalysis(response, "transactions", "posting", p.DateFrom, p.DateTo)
	d.Context.Completeness = "not-applicable"
	d.Schema.PageUnit = "transaction"
	d.Context.NormalizedRequest["text"] = response.Request.Text
	d.Context.NormalizedRequest["account"] = response.Request.Account
	column, err := recordSchema(reflect.TypeOf(JournalPostingDTO{}), "posting")
	if err != nil {
		return d, err
	}
	c := analysis.NewColumn
	d.Schema.Columns = []analysis.Column{c("transaction_id", "Transaction", "string", false), c("posting_index", "Posting index", "integer", false), c("date", "Date", "date", false), c("flag", "Flag", "string", false), c("payee", "Payee", "string", false), c("narration", "Narration", "string", false), c("tags", "Tags", "set<string>", false), c("links", "Links", "set<string>", false), column}
	for _, t := range p.Transactions {
		for i, posting := range t.Postings {
			cell, err := recordValue(reflect.ValueOf(posting), column, response.State.SnapshotID)
			if err != nil {
				return d, err
			}
			d.Rows = append(d.Rows, []query.Cell{stringCell(t.ID), {Type: "integer", Value: strconv.Itoa(i)}, {Type: "date", Value: t.Date}, stringCell(t.Flag), stringCell(t.Payee), stringCell(t.Narration), {Type: "set<string>", Value: analysis.NormalizeSet(t.Tags)}, {Type: "set<string>", Value: analysis.NormalizeSet(t.Links)}, cell})
		}
	}
	d.Context.ReturnedRows = len(d.Rows)
	setAnalysisPage(&d, report.ProjectionPage{TotalCount: p.TotalCount, Offset: p.Offset, Limit: p.Limit, ReturnedCount: len(p.Transactions), HasMore: p.HasMore, Truncated: len(p.Transactions) < p.TotalCount})
	return d, nil
}
