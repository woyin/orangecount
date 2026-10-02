// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"orangecount/internal/ledger"
	"orangecount/internal/source"
	"reflect"
	"strconv"
	"strings"
)

// DirectiveRecord keeps effective data separate from its original source kind.
// Generated identities are snapshot-bound; origin IDs identify their declaration.
type DirectiveRecord struct {
	ID         string
	Kind       ledger.DirectiveKind
	SourceKind ledger.DirectiveKind
	Date       string
	File       string
	Span       source.Span
	Directive  ledger.Directive
	Generated  bool
	OriginID   string
	search     string
	accounts   []string
}

type DirectiveCatalogue struct {
	Dated         []DirectiveRecord
	Configuration []DirectiveRecord
}

type sourceDirectiveIndex struct {
	configuration []DirectiveRecord
	origins       map[source.Span]DirectiveRecord
}

// DirectiveCatalogue returns independent typed records from a valid published
// snapshot. Configuration follows loader traversal, dated entries follow the
// evaluator's effective ordering; it never merges a failed attempt's source.
func (s *Snapshot) DirectiveCatalogue() DirectiveCatalogue {
	if s == nil {
		return DirectiveCatalogue{Dated: []DirectiveRecord{}, Configuration: []DirectiveRecord{}}
	}
	return cloneValue(reflect.ValueOf(s.directives)).Interface().(DirectiveCatalogue)
}

func indexSourceDirectives(graph *source.Graph, parsed map[source.FileID]*ledger.File) sourceDirectiveIndex {
	index := sourceDirectiveIndex{configuration: []DirectiveRecord{}, origins: map[source.Span]DirectiveRecord{}}
	if graph == nil {
		return index
	}
	visited := map[source.FileID]bool{}
	var walk func(source.FileID)
	walk = func(id source.FileID) {
		if visited[id] {
			return
		}
		visited[id] = true
		file := parsed[id]
		if file == nil {
			return
		}
		for _, directive := range file.Directives {
			path := graph.Path(id)
			kind := directive.Kind()
			hash := sha256.Sum256([]byte(path + "\x00" + string(kind) + "\x00" + directive.RawText()))
			record := DirectiveRecord{ID: hex.EncodeToString(hash[:]), Kind: kind, SourceKind: kind, File: path, Span: directive.Span(), Directive: directive}
			index.origins[record.Span] = record
			if configurationKind(kind) {
				index.configuration = append(index.configuration, record)
			}
			if include, ok := directive.(ledger.Include); ok {
				for _, edge := range graph.Edges[id] {
					if edge.Literal == include.Path {
						walk(edge.To)
						break
					}
				}
			}
		}
	}
	if graph.Entry != 0 {
		walk(graph.Entry)
	} else {
		for _, id := range graph.Order {
			walk(id)
		}
	}
	return index
}

func configurationKind(kind ledger.DirectiveKind) bool {
	switch kind {
	case ledger.KindOption, ledger.KindPlugin, ledger.KindInclude, ledger.KindPushTag, ledger.KindPopTag, ledger.KindPushMeta, ledger.KindPopMeta:
		return true
	}
	return false
}

func buildDirectiveCatalogue(index sourceDirectiveIndex, evaluation *ledger.Evaluation, snapshotID string) DirectiveCatalogue {
	catalogue := DirectiveCatalogue{Dated: []DirectiveRecord{}, Configuration: append([]DirectiveRecord{}, index.configuration...)}
	counts := map[string]int{}
	for _, record := range index.origins {
		counts[record.ID]++
	}
	// Identical declarations cannot safely restore by content across revisions.
	for span, record := range index.origins {
		if counts[record.ID] > 1 {
			record.ID += ":" + snapshotID + ":" + strconv.Itoa(int(span.File)) + ":" + strconv.Itoa(span.Start)
			index.origins[span] = record
		}
	}
	for i, record := range catalogue.Configuration {
		catalogue.Configuration[i].ID = index.origins[record.Span].ID
		prepareDirectiveSearch(&catalogue.Configuration[i])
	}
	if evaluation == nil {
		return catalogue
	}
	for ordinal, entry := range evaluation.Entries {
		if !entry.Date.Valid() {
			continue
		}
		origin, exists := index.origins[entry.Span]
		record := DirectiveRecord{ID: origin.ID, Kind: entry.Directive.Kind(), SourceKind: origin.Kind, Date: entry.Date.Raw, File: entry.File, Span: entry.Span, Directive: entry.Directive}
		record.Generated = !exists || record.Kind != origin.Kind
		if record.Generated {
			record.OriginID = origin.ID
			record.ID = origin.ID + ":generated:" + snapshotID + ":" + strconv.Itoa(ordinal)
		}
		prepareDirectiveSearch(&record)
		catalogue.Dated = append(catalogue.Dated, record)
	}
	return catalogue
}

// DirectiveSelection filters the immutable index before copying a bounded page.
type DirectiveSelection struct {
	Mode, Kind, Text, Account, DateFrom, DateTo string
	Offset, Limit                               int
	Descending                                  bool
}
type DirectiveRecordPage struct {
	Records                   []DirectiveRecord
	TotalCount, Offset, Limit int
	HasMore                   bool
}

func (s *Snapshot) SelectDirectives(selection DirectiveSelection) (DirectiveRecordPage, error) {
	page := DirectiveRecordPage{Records: []DirectiveRecord{}, Offset: selection.Offset, Limit: selection.Limit}
	if selection.Mode != "all" && selection.Mode != "config" || selection.Offset < 0 || selection.Limit < 0 || selection.Limit > 500 {
		return page, fmt.Errorf("invalid directive selection")
	}
	if page.Limit == 0 {
		page.Limit = 100
	}
	if s == nil {
		return page, nil
	}
	records := s.directives.Dated
	if selection.Mode == "config" {
		records = s.directives.Configuration
	}
	text := strings.ToLower(strings.TrimSpace(selection.Text))
	for step := range records {
		i := step
		if selection.Descending {
			i = len(records) - 1 - step
		}
		record := records[i]
		if selection.Kind != "" && string(record.Kind) != selection.Kind {
			continue
		}
		if selection.Mode == "all" && (selection.DateFrom != "" && record.Date < selection.DateFrom || selection.DateTo != "" && record.Date > selection.DateTo) {
			continue
		}
		if text != "" && !strings.Contains(record.search, text) {
			continue
		}
		if selection.Account != "" {
			match := false
			for _, account := range record.accounts {
				if account == selection.Account || strings.HasPrefix(account, selection.Account+":") {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if page.TotalCount >= page.Offset && len(page.Records) < page.Limit {
			page.Records = append(page.Records, record)
		}
		page.TotalCount++
	}
	page.HasMore = page.Offset+len(page.Records) < page.TotalCount
	page.Records = cloneValue(reflect.ValueOf(page.Records)).Interface().([]DirectiveRecord)
	return page, nil
}

func prepareDirectiveSearch(record *DirectiveRecord) {
	parts := []string{string(record.Kind), string(record.SourceKind), record.Date, record.File, record.Directive.RawText()}
	value := reflect.Indirect(reflect.ValueOf(record.Directive))
	if field := value.FieldByName("Meta"); field.IsValid() {
		for _, meta := range field.Interface().([]ledger.Metadata) {
			parts = append(parts, meta.Key, meta.Value.Raw, meta.Value.String)
		}
	}
	switch directive := value.Interface().(type) {
	case ledger.Open:
		record.accounts = []string{directive.Account}
	case ledger.Close:
		record.accounts = []string{directive.Account}
	case ledger.Balance:
		record.accounts = []string{directive.Account}
	case ledger.Pad:
		record.accounts = []string{directive.Account, directive.SourceAccount}
	case ledger.Document:
		record.accounts = []string{directive.Account}
	case ledger.Note:
		record.accounts = []string{directive.Account}
	case ledger.Transaction:
		parts = append(parts, directive.Payee, directive.Narration, strings.Join(directive.Tags, " "), strings.Join(directive.Links, " "))
		for _, posting := range directive.Postings {
			record.accounts = append(record.accounts, posting.Account)
		}
	}
	record.search = strings.ToLower(strings.Join(append(parts, record.accounts...), " "))
}
