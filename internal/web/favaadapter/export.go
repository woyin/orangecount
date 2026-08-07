// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

import (
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"orangecount/internal/source"
)

// ExportEntries renders the currently filtered entries as Beancount source,
// the way Fava's download-journal endpoint does. Each surviving entry is
// sliced out of its original source file, so the export is source-preserving:
// formatting, comments inside a directive, and metadata all come through
// exactly as written. Entries are separated by blank lines.
func ExportEntries(e ledger.Evaluation, graph *source.Graph, filters report.Filters, journal report.JournalFilters) string {
	state := newJournalFilterState(filters, journal)
	blocks := make([]string, 0)
	for _, record := range e.Entries {
		entry, ok := projectJournalEntry(record)
		if !ok {
			continue
		}
		entry.File = journalDisplayPath(record.File, graph)
		entry.Span = record.Span.String()
		if state.excluded(entry) {
			continue
		}
		if block := entrySourceBlock(record, graph); block != "" {
			blocks = append(blocks, block)
		}
	}
	if len(blocks) == 0 {
		return ""
	}
	return strings.Join(blocks, "\n\n") + "\n"
}

// entrySourceBlock cuts the byte span of one entry out of its source file.
func entrySourceBlock(record ledger.EntryRecord, graph *source.Graph) string {
	if graph == nil || !record.Span.Valid() || record.Span.Empty() {
		return ""
	}
	file := graph.File(record.Span.File)
	if file == nil {
		return ""
	}
	start, end := record.Span.Start, record.Span.End
	if start < 0 || end > len(file.Data) || start >= end {
		return ""
	}
	return strings.TrimRight(string(file.Data[start:end]), " \t\r\n")
}
