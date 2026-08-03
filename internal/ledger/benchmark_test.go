// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"strconv"
	"strings"
	"testing"

	"orangecount/internal/source"
)

func benchmarkLedgerText() []byte {
	var builder strings.Builder
	builder.WriteString("2000-01-01 open Assets:Cash USD\n2000-01-01 open Equity:Opening USD\n")
	for i := 0; i < 100; i++ {
		builder.WriteString("2000-01-02 * \"fixture ")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteString("\"\n  Assets:Cash 1 USD\n  Equity:Opening -1 USD\n")
	}
	return []byte(builder.String())
}

func BenchmarkParse(b *testing.B) {
	data := benchmarkLedgerText()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		_, diagnostics := ParseText("benchmark.bean", data)
		if diagnostics.HasErrors() {
			b.Fatal(diagnostics.All())
		}
	}
}

func BenchmarkEvaluate(b *testing.B) {
	data := benchmarkLedgerText()
	file, diagnostics := ParseText("benchmark.bean", data)
	if diagnostics.HasErrors() {
		b.Fatal(diagnostics.All())
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		evaluation := EvaluateFiles(map[source.FileID]*File{1: file}, []source.FileID{1}, EvalOptions{})
		if !evaluation.Valid {
			b.Fatal(evaluation.Diagnostics)
		}
	}
}
