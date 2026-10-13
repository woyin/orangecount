// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package desktop

import (
	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

func evaluationSourceFiles(e ledger.Evaluation) map[source.FileID]string {
	files := make(map[source.FileID]string)
	for _, entry := range e.Entries {
		if entry.File != "" {
			files[entry.Span.File] = entry.File
		}
	}
	return files
}
