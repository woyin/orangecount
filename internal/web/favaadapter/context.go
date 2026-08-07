// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

import (
	"crypto/sha256"
	"encoding/hex"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

// EntryContext is what Fava's context modal shows for one entry: where it
// lives and its source exactly as written. Balances before/after and the
// editable CodeMirror slice belong to later phases (H1); this is the
// read-only projection.
type EntryContext struct {
	Entry       JournalEntry `json:"entry"`
	SourceSlice string       `json:"source_slice"`
	SHA256Sum   string       `json:"sha256sum"`
}

// entryHash identifies one ledger entry by its source position. The position
// is immutable within a snapshot, so the hash is a stable address for the
// context modal without exposing file paths in URLs.
func entryHash(record ledger.EntryRecord) string {
	digest := sha256.New()
	digest.Write([]byte(record.File))
	digest.Write([]byte{0})
	digest.Write([]byte(record.Span.String()))
	return hex.EncodeToString(digest.Sum(nil))
}

// ProjectEntryContext resolves an entry hash to its projection and source
// slice. It scans the published entry stream; hashes are position-derived,
// so the same snapshot always answers the same way.
func ProjectEntryContext(e ledger.Evaluation, graph *source.Graph, hash string) (EntryContext, bool) {
	if hash == "" {
		return EntryContext{}, false
	}
	for _, record := range e.Entries {
		if entryHash(record) != hash {
			continue
		}
		entry, ok := projectJournalEntry(record)
		if !ok {
			return EntryContext{}, false
		}
		entry.EntryHash = hash
		entry.File = journalDisplayPath(record.File, graph)
		entry.Span = record.Span.String()
		slice := entrySourceBlock(record, graph)
		return EntryContext{Entry: entry, SourceSlice: slice, SHA256Sum: sha256Hex(slice)}, true
	}
	return EntryContext{}, false
}

func sha256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
