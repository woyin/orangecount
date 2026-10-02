// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package query_test

import (
	"orangecount/internal/query"
	"testing"
)

func TestStructuredStandardColumnsRetainEmptyAndNullValues(t *testing.T) {
	for _, read := range []struct{ text, kind string }{
		{"SELECT tags FROM #postings LIMIT 1", "set<string>"},
		{"SELECT accounts FROM #transactions LIMIT 1", "set<string>"},
		{"SELECT meta FROM #postings LIMIT 1", "dictionary"},
		{"SELECT entry_meta('__tolerances__') LIMIT 1", "dictionary"},
		{"SELECT entry FROM #postings LIMIT 1", "transaction"},
		{"SELECT open FROM #accounts LIMIT 1", "open"},
	} {
		t.Run(read.text, func(t *testing.T) {
			result, err := query.EvaluateTyped(read.text, typedEvaluation(t))
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Rows) != 1 || result.Rows[0][0].Type != read.kind {
				t.Fatalf("%+v", result)
			}
		})
	}
}
