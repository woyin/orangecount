// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package desktop

import (
	"fmt"
	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/report"
	"slices"
)

type SavedQuery struct {
	ID     string         `json:"id"`
	Date   string         `json:"date"`
	Name   string         `json:"name"`
	Text   string         `json:"text"`
	Source SourceLocation `json:"source"`
}

type QueryPresetPeriod struct {
	DateFrom string         `json:"date_from"`
	DateTo   string         `json:"date_to"`
	Presets  []query.Preset `json:"presets"`
}

type QueryCatalogue struct {
	SnapshotID string                `json:"snapshot_id"`
	Period     *QueryPresetPeriod    `json:"period"`
	Presets    []query.Preset        `json:"presets"`
	Queries    []SavedQuery          `json:"queries"`
	Page       report.ProjectionPage `json:"page"`
}

func projectQueryCatalogue(e ledger.Evaluation, request Request, snapshotID, from, to string) *QueryCatalogue {
	queries := make([]SavedQuery, 0)
	for index, record := range e.Entries {
		var saved *ledger.Query
		switch directive := record.Directive.(type) {
		case ledger.Query:
			saved = &directive
		case *ledger.Query:
			saved = directive
		}
		if saved == nil {
			continue
		}
		queries = append(queries, SavedQuery{ID: fmt.Sprintf("%s:q%d", snapshotID, index), Date: saved.Date.Raw, Name: saved.Name, Text: saved.Query, Source: location(record.File, record.Span)})
	}
	if request.Sort == "descending" {
		slices.Reverse(queries)
	}
	limit := request.Limit
	if limit == 0 {
		limit = 100
	}
	rows, page := holdingPage(queries, request.Offset, limit)
	catalogue := &QueryCatalogue{SnapshotID: snapshotID, Queries: rows, Page: page, Presets: query.Presets()}
	if presets, err := query.PresetsForPeriod(from, to); err == nil {
		catalogue.Period = &QueryPresetPeriod{DateFrom: from, DateTo: to, Presets: presets}
	}
	return catalogue
}
