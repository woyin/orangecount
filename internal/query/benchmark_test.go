// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package query

import (
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

func BenchmarkEvaluateQuery(b *testing.B) {
	file, diagnostics := ledger.ParseText("query-benchmark.bean", []byte("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n2000-01-02 * \"seed\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n"))
	if diagnostics.HasErrors() {
		b.Fatal(diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		b.Fatal(evaluation.Diagnostics)
	}
	queryText := "SELECT account, sum(balance) AS balance FROM accounts GROUP BY account ORDER BY account"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Evaluate(queryText, *evaluation); err != nil {
			b.Fatal(err)
		}
	}
}
