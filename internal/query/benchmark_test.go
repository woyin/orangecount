// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package query

import (
	"context"
	"fmt"
	"strings"
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

// benchmarkEvaluation parses and evaluates one generated ledger once per
// benchmark, so the source cost stays outside the timed loop and every case
// pays the same one-time setup.
func benchmarkEvaluation(b *testing.B, text string) ledger.Evaluation {
	b.Helper()
	file, diagnostics := ledger.ParseText("query-benchmark.bean", []byte(text))
	if diagnostics.HasErrors() {
		b.Fatal(diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		b.Fatal(evaluation.Diagnostics)
	}
	return *evaluation
}

// benchmarkNarrationLedger repeats one narration over every posting of
// `transactions` balanced two-posting transactions. DISTINCT therefore sees
// 2*transactions rows that all carry the same narration.
func benchmarkNarrationLedger(transactions int) string {
	var builder strings.Builder
	builder.WriteString("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n")
	for i := 0; i < transactions; i++ {
		fmt.Fprintf(&builder, "2000-01-02 * \"payee-%d\" \"shared narration\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n", i)
	}
	return builder.String()
}

// benchmarkAccountLedger spreads `accounts` accounts across `commodities`
// commodities with one balanced transaction per pair, so the same ledger backs
// the pivot, HAVING and derived-subquery benchmarks. `open` lines are declared
// for every generated account to keep the evaluation free of account errors.
func benchmarkAccountLedger(accounts, commodities int) string {
	var builder strings.Builder
	builder.WriteString("2000-01-01 open Equity:Opening\n")
	for account := 0; account < accounts; account++ {
		fmt.Fprintf(&builder, "2000-01-01 open Assets:Item%03d\n", account)
	}
	for account := 0; account < accounts; account++ {
		for commodity := 0; commodity < commodities; commodity++ {
			name := fmt.Sprintf("C%03d", commodity)
			amount := commodity + 1
			fmt.Fprintf(&builder, "2000-01-02 * \"trade-%d-%d\"\n  Assets:Item%03d %d %s\n  Equity:Opening -%d %s\n",
				account, commodity, account, amount, name, amount, name)
		}
	}
	return builder.String()
}

// BenchmarkDistinct measures the DISTINCT dedup stage of the bounded
// execution path. The duplicate cases collapse 1k/10k pre-distinct rows to a
// single row; the unique case keeps 10000 distinct line numbers, one key per
// posting.
func BenchmarkDistinct(b *testing.B) {
	for _, size := range []struct {
		name         string
		transactions int
		text         string
	}{
		{name: "duplicates-1k", transactions: 500, text: "SELECT DISTINCT narration FROM postings"},
		{name: "duplicates-10k", transactions: 5000, text: "SELECT DISTINCT narration FROM postings"},
		{name: "unique-10k-postings", transactions: 5000, text: "SELECT DISTINCT lineno FROM postings ORDER BY lineno"},
	} {
		b.Run(size.name, func(b *testing.B) {
			evaluation := benchmarkEvaluation(b, benchmarkNarrationLedger(size.transactions))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := ExecuteTyped(context.Background(), size.text, evaluation, ExecutionLimits{}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPivot measures GROUP BY plus the PIVOT BY reshape. The fixture
// produces 201 row keys (200 generated accounts plus Equity:Opening) x 8 pivot
// values x 1 measure = 1608 cells, well below QueryPivotCellLimit (1048576
// cells).
func BenchmarkPivot(b *testing.B) {
	evaluation := benchmarkEvaluation(b, benchmarkAccountLedger(200, 8))
	const text = "SELECT account, currency, sum(number) GROUP BY account, currency PIVOT BY account, currency"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ExecuteTyped(context.Background(), text, evaluation, ExecutionLimits{}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDerivedSubquery measures an inner WHERE feeding an outer ORDER BY
// over the same 200-account fixture: the inner select keeps the postings above
// 6 of 8 commodities and the outer query sorts the derived rows.
func BenchmarkDerivedSubquery(b *testing.B) {
	evaluation := benchmarkEvaluation(b, benchmarkAccountLedger(200, 8))
	const text = "SELECT account, number FROM (SELECT account, number FROM postings WHERE number > 6) ORDER BY account"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ExecuteTyped(context.Background(), text, evaluation, ExecutionLimits{}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHaving measures the HAVING predicate once per group plus the
// aggregate projection over 201 groups.
func BenchmarkHaving(b *testing.B) {
	evaluation := benchmarkEvaluation(b, benchmarkAccountLedger(200, 8))
	const text = "SELECT account, sum(number) AS total GROUP BY account HAVING sum(number) > 0 ORDER BY account"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ExecuteTyped(context.Background(), text, evaluation, ExecutionLimits{}); err != nil {
			b.Fatal(err)
		}
	}
}
