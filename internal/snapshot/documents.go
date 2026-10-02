// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package snapshot

import (
	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

// DocumentDeclaration is source evidence, independent of accounting validity.
// A valid parsed document in an otherwise failed attempt still identifies a
// user-owned path that an analysis export must not replace.
type DocumentDeclaration struct {
	SourcePath string
	Filename   string
}

func (result BuildResult) DocumentDeclarations() []DocumentDeclaration {
	return append([]DocumentDeclaration{}, result.documents...)
}

func documentDeclarations(graph *source.Graph, parsed map[source.FileID]*ledger.File) []DocumentDeclaration {
	result := []DocumentDeclaration{}
	if graph == nil {
		return result
	}
	for _, id := range graph.Order {
		file := parsed[id]
		if file == nil {
			continue
		}
		for _, directive := range file.Directives {
			var names []string
			switch document := directive.(type) {
			case ledger.Document:
				names = document.Filenames
			case *ledger.Document:
				if document != nil {
					names = document.Filenames
				}
			}
			for _, name := range names {
				result = append(result, DocumentDeclaration{SourcePath: graph.Path(id), Filename: name})
			}
		}
	}
	return result
}
