// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

// testEvaluation builds a deterministic sanitized evaluation. All names and
// amounts are synthetic: no private ledger data is ever used in adapter
// tests.
func testEvaluation(t *testing.T) ledger.Evaluation {
	t.Helper()
	text := `2000-01-01 open Assets:Bank:Cash USD
2000-01-01 open Assets:Bank:Euro EUR
2000-01-01 open Equity:Opening USD
2000-01-01 open Expenses:Food USD EUR
2000-01-01 commodity AAPL
2000-06-01 price AAPL 150.25 USD
2000-06-01 price EUR 1.10 USD
2002-02-03 * "Cafe" "Lunch" #meal ^trip
  Assets:Bank:Cash -2.50 USD
  Expenses:Food 2.50 USD
2002-02-03 * "Cafe" "Dinner" #meal
  Assets:Bank:Euro -1.15 EUR
  Expenses:Food 1.15 EUR
`
	file, diagnostics := ledger.ParseText("report.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}
	return *evaluation
}

// buildSnapshot writes a sanitized ledger to a temp file and builds a real
// snapshot from it, so the projection exercises the real snapshot boundary
// (including private snapshot fields we cannot set by hand).
func buildSnapshot(t *testing.T, text string) *snapshot.Snapshot {
	t.Helper()
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	result := snapshot.Build(entry)
	if result.Snapshot == nil {
		t.Fatalf("snapshot build failed=%+v", result.Diagnostics)
	}
	return result.Snapshot
}

func TestEnvelopeHasStableMtimeFromSnapshot(t *testing.T) {
	built := time.Date(2026, 3, 4, 5, 6, 7, 123000000, time.UTC)
	first := NewEnvelope("payload", built)
	second := NewEnvelope("payload", built)
	if first.Mtime != second.Mtime || first.Data != "payload" {
		t.Fatalf("envelope not deterministic: %+v vs %+v", first, second)
	}
}

func TestAdapterErrorMarshalShape(t *testing.T) {
	data, err := json.Marshal(AdapterError{Error: "no valid snapshot"})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 || decoded["error"] != "no valid snapshot" {
		t.Fatalf("adapter error shape=%v", decoded)
	}
}

func TestRegistryCoversExpectedRoutesAndStaysInternal(t *testing.T) {
	expected := []RouteName{
		RouteLedgerData, RouteStatus, RouteErrors, RouteMetadata1, RouteJournal,
		RouteTreeReport, RouteQueryShell, RouteEditorSource, RouteEditorSave,
		RouteImportPreview, RouteOptions,
	}
	for _, route := range expected {
		contract, ok := Lookup(route)
		if !ok {
			t.Fatalf("route %s not in registry", route)
		}
		if contract.Authority != AuthorityLedger && contract.Authority != AuthorityPresentation {
			t.Fatalf("route %s lacks an authority", route)
		}
		if contract.Owner == "" || contract.Errors != ErrorFull {
			t.Fatalf("route %s incomplete contract %+v", route, contract)
		}
	}
	if _, ok := Lookup(RouteName("not-a-route")); ok {
		t.Fatal("registry accepted an unknown route")
	}
	for _, contract := range Registry {
		if strings.Contains(string(contract.Route), "/") || strings.Contains(string(contract.Route), "http") {
			t.Fatalf("route %q must be an internal identifier, not a URL", contract.Route)
		}
	}
}

func TestBootstrapDeterministicOrderingAndExactTypes(t *testing.T) {
	snap := buildSnapshot(t, `2000-01-01 open Assets:Bank:Cash USD
2000-01-01 open Assets:Bank:Euro EUR
2000-01-01 open Equity:Opening USD
2000-01-01 open Expenses:Food USD
2000-01-01 commodity AAPL
2000-06-01 price AAPL 150.25 USD
2002-02-03 * "Cafe" "Lunch" #meal ^trip
  Assets:Bank:Cash -2.50 USD
  Expenses:Food 2.50 USD
`)
	proj := BootstrapProjection(BootstrapOptions{Snapshot: snap, BaseURL: "/"})
	if proj.BaseURL != "/" || !proj.Valid {
		t.Fatalf("bootstrap base/valid=%+v", proj)
	}
	if !isSorted(proj.Accounts) || !isSorted(proj.Currencies) || !isSorted(proj.Payees) || !isSorted(proj.Years) {
		t.Fatalf("bootstrap slices not sorted: %+v", proj)
	}
	if len(proj.Accounts) < 3 || len(proj.Currencies) < 2 {
		t.Fatalf("bootstrap too small: %+v", proj)
	}
	result, err := json.Marshal(proj)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"accounts", "currencies", "payees", "tags", "links", "years", "extensions", "other_ledgers"} {
		if _, ok := decoded[key].([]any); !ok {
			t.Fatalf("bootstrap field %q not an array: %v", key, decoded[key])
		}
	}
	if decoded["incognito"] != false || decoded["have_excel"] != false {
		t.Fatalf("bootstrap out-of-scope flags wrong: %v", decoded)
	}
}

func TestBootstrapDiagnosticsNeverLeakAbsolutePaths(t *testing.T) {
	snap := buildSnapshot(t, `2000-01-01 open Assets:Cash USD
2000-01-01 open Equity:Opening USD
`)
	// Inject a diagnostic with an absolute path directly into the evaluation
	// boundary is not possible via the public snapshot API; instead verify the
	// projection's redaction helper on a synthetic path.
	adapter := projectDiagnostics([]diagnostic.Diagnostic{
		diagnostic.New("E-CUSTOM", diagnostic.Error, source.Span{StartLine: 2, StartColumn: 1}).
			WithPath("/private/ledger/main.bean"),
	}, snap.Graph())
	if len(adapter) != 1 || adapter[0].Source == nil {
		t.Fatalf("expected one diagnostic with source: %+v", adapter)
	}
	if strings.Contains(adapter[0].Source.Filename, "/private/") {
		t.Fatalf("diagnostic source leaked absolute path: %+v", adapter[0].Source)
	}
	if adapter[0].Source.Filename == "" {
		t.Fatalf("diagnostic source filename empty: %+v", adapter[0].Source)
	}
}

func TestBootstrapNilSnapshotIsSafe(t *testing.T) {
	proj := BootstrapProjection(BootstrapOptions{})
	if proj.Valid || len(proj.Accounts) != 0 || len(proj.Currencies) != 0 {
		t.Fatalf("nil snapshot projection not safe/empty: %+v", proj)
	}
	if _, err := json.Marshal(proj); err != nil {
		t.Fatal(err)
	}
}

func isSorted(values []string) bool {
	for i := 1; i < len(values); i++ {
		if values[i-1] > values[i] {
			return false
		}
	}
	return true
}

func TestMetadataKeepsExactValuesAndStableOrder(t *testing.T) {
	value := testEvaluation(t)
	proj := MetadataProjectionOptions(MetadataOptions{Evaluation: value, Root: "Assets"})
	if len(proj.Currencies) == 0 {
		t.Fatalf("no currencies in projection: %+v", proj)
	}
	data, err := json.Marshal(proj)
	if err != nil {
		t.Fatal(err)
	}
	// Exact-value preservation: the -2.50 / 2.50 amounts must survive JSON as
	// "-2.5"/"2.5" (canonical exact strings), never as floats, and the
	// AAPL/USD price pair must be present.
	if strings.Contains(string(data), `"-2.5"`) == false || strings.Contains(string(data), `"2.5"`) == false {
		t.Fatalf("exact amounts missing from JSON: %s", data)
	}
	if strings.Contains(string(data), `"base":"AAPL"`) == false {
		t.Fatalf("price pair missing from JSON: %s", data)
	}
	if len(proj.AccountTotal.Totals) != 2 || proj.AccountTotal.Totals[0].Currency != "EUR" || proj.AccountTotal.Totals[1].Currency != "USD" {
		t.Fatalf("assets total=%+v", proj.AccountTotal)
	}
	euro := proj.AccountTotal.Totals[0].Amount
	dollar := proj.AccountTotal.Totals[1].Amount
	if euro.String() != "-1.15" || dollar.String() != "-2.5" {
		t.Fatalf("assets total exact values wrong: %s / %s", euro.String(), dollar.String())
	}
	for i := 1; i < len(proj.Commodities); i++ {
		prev := proj.Commodities[i-1]
		cur := proj.Commodities[i]
		if prev.Currency > cur.Currency || (prev.Currency == cur.Currency && prev.Account > cur.Account) {
			t.Fatalf("commodity summaries not sorted: %+v", proj.Commodities)
		}
	}
}

func TestMetadataNoFloatInFractionDecimal(t *testing.T) {
	third := ledger.NewDecimal(new(big.Rat).SetFrac64(1, 3))
	data, err := third.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"1/3"` {
		t.Fatalf("fraction marshalled as %s", data)
	}
}

func TestSupportedFavaOptionsAreOptIn(t *testing.T) {
	options := map[string]string{"locale": "en", "plugin-only-option": "secret"}
	projected := projectFavaOptions(options)
	if _, ok := projected["plugin-only-option"]; ok {
		t.Fatalf("plugin option leaked: %+v", projected)
	}
	if _, ok := projected["locale"]; !ok {
		t.Fatalf("allowlisted option missing: %+v", projected)
	}
}

func TestBootstrapExactValuesSurviveJSON(t *testing.T) {
	snap := buildSnapshot(t, `2000-01-01 open Assets:Bank:Cash USD
2000-01-01 open Equity:Opening USD
2002-02-03 * "Cafe" "Lunch"
  Assets:Bank:Cash -2.50 USD
  Equity:Opening 2.50 USD
`)
	proj := BootstrapProjection(BootstrapOptions{Snapshot: snap})
	data, err := json.Marshal(proj)
	if err != nil {
		t.Fatal(err)
	}
	// The bootstrap payload itself carries no ledger amounts (only options,
	// accounts, currencies, diagnostics), so this test asserts the slice
	// fields are present and the currency set is exact; exact-value
	// preservation is covered by TestMetadataKeepsExactValuesAndStableOrder.
	if !reflect.DeepEqual(proj.Currencies, []string{"USD"}) {
		t.Fatalf("bootstrap currencies=%v", proj.Currencies)
	}
	_ = data
}

func TestProjectJournalCarriesAccountChange(t *testing.T) {
	evaluation := testEvaluation(t)
	projection := ProjectJournal(evaluation, nil, report.Filters{Account: "Assets:Bank"}, report.JournalFilters{})
	wantChange := map[string]string{"USD": "-2.5", "EUR": "-1.15"}
	seen := 0
	for _, entry := range projection.Entries {
		if entry.Type != "transaction" {
			if entry.Change != nil {
				t.Fatalf("non-transaction carries change: %+v", entry)
			}
			continue
		}
		seen++
		if len(entry.Change) != 1 {
			t.Fatalf("change=%+v", entry.Change)
		}
		currency := entry.Change[0].Currency
		expected, ok := wantChange[currency]
		if !ok || entry.Change[0].Number.Display != expected {
			t.Fatalf("change=%+v want %q for %s", entry.Change, expected, currency)
		}
	}
	if seen != 2 {
		t.Fatalf("transactions=%d entries=%+v", seen, projection.Entries)
	}
	unfiltered := ProjectJournal(evaluation, nil, report.Filters{}, report.JournalFilters{})
	for _, entry := range unfiltered.Entries {
		if entry.Change != nil {
			t.Fatalf("unfiltered entry carries change: %+v", entry)
		}
	}
}

func TestProjectJournalHonorsFQLFilters(t *testing.T) {
	evaluation := testEvaluation(t)
	countTransactions := func(projection JournalReport) int {
		count := 0
		for _, entry := range projection.Entries {
			if entry.Type == "transaction" {
				count++
			}
		}
		return count
	}
	// Juxtaposition is AND: only the Lunch transaction carries both terms.
	tagged := ProjectJournal(evaluation, nil, report.Filters{Text: "#meal ^trip"}, report.JournalFilters{})
	if got := countTransactions(tagged); got != 1 || tagged.Entries[0].Narration != "Lunch" {
		t.Fatalf("tagged=%+v", tagged.Entries)
	}
	// Comma is OR and string terms are case-insensitive regexes.
	alternation := ProjectJournal(evaluation, nil, report.Filters{Text: "dinner, LUNCH"}, report.JournalFilters{})
	if got := countTransactions(alternation); got != 2 {
		t.Fatalf("alternation transactions=%d entries=%+v", got, alternation.Entries)
	}
	// Posting-amount comparison uses magnitudes.
	amount := ProjectJournal(evaluation, nil, report.Filters{Text: ">2"}, report.JournalFilters{})
	if got := countTransactions(amount); got != 1 {
		t.Fatalf("amount transactions=%d entries=%+v", got, amount.Entries)
	}
	// Negation keeps the directives that lack the tag (the open directives).
	negated := ProjectJournal(evaluation, nil, report.Filters{Text: "-#meal"}, report.JournalFilters{})
	if got := countTransactions(negated); got != 0 || len(negated.Entries) == 0 {
		t.Fatalf("negated entries=%+v", negated.Entries)
	}
}

func TestProjectJournalProjectsDirectiveMetadata(t *testing.T) {
	snap := buildSnapshot(t, `2000-01-01 open Assets:Bank:Cash USD
2000-01-01 open Equity:Opening USD
2002-02-05 note Assets:Bank "Review statement"
  auditor: kim
2002-02-03 * "Cafe" "Lunch"
  reference: INV-7
  Assets:Bank:Cash -2.50 USD
    cleared-by: bank
  Equity:Opening 2.50 USD
`)
	projection := ProjectJournal(snap.Evaluation(), snap.Graph(), report.Filters{}, report.JournalFilters{})
	var note *JournalEntry
	var transaction *JournalEntry
	for i := range projection.Entries {
		switch projection.Entries[i].Type {
		case "note":
			note = &projection.Entries[i]
		case "transaction":
			transaction = &projection.Entries[i]
		}
	}
	if note == nil || len(note.Metadata) != 1 || note.Metadata[0].Key != "auditor" || note.Metadata[0].Value != "kim" {
		t.Fatalf("note metadata=%+v", note)
	}
	if transaction == nil || len(transaction.Metadata) != 1 || transaction.Metadata[0].Key != "reference" || transaction.Metadata[0].Value != "INV-7" {
		t.Fatalf("transaction metadata=%+v", transaction)
	}
	posting := transaction.Postings[0]
	if len(posting.Metadata) != 1 || posting.Metadata[0].Key != "cleared-by" || posting.Metadata[0].Value != "bank" {
		t.Fatalf("posting metadata=%+v", posting)
	}
}

func TestExportEntriesSlicesFilteredSource(t *testing.T) {
	snap := buildSnapshot(t, `2000-01-01 open Assets:Bank:Cash USD
2000-01-01 open Equity:Opening USD
2002-02-03 * "Cafe" "Lunch" #meal
  Assets:Bank:Cash -2.50 USD
  Equity:Opening 2.50 USD
2002-02-04 * "Cafe" "Dinner"
  Assets:Bank:Cash -1.15 USD
  Equity:Opening 1.15 USD
`)
	exported := ExportEntries(snap.Evaluation(), snap.Graph(), report.Filters{Text: "#meal"}, report.JournalFilters{})
	if !strings.Contains(exported, `2002-02-03 * "Cafe" "Lunch" #meal`) || !strings.Contains(exported, "Assets:Bank:Cash -2.50 USD") {
		t.Fatalf("export missing the tagged transaction:\n%s", exported)
	}
	if strings.Contains(exported, "Dinner") || strings.Contains(exported, "open") {
		t.Fatalf("export leaks unfiltered entries:\n%s", exported)
	}
	if !strings.HasSuffix(exported, "\n") || strings.Contains(exported, "\n\n\n") {
		t.Fatalf("export separators unexpected:\n%q", exported)
	}
	everything := ExportEntries(snap.Evaluation(), snap.Graph(), report.Filters{}, report.JournalFilters{})
	if !strings.Contains(everything, "Dinner") || !strings.Contains(everything, "open Assets:Bank:Cash") {
		t.Fatalf("unfiltered export incomplete:\n%s", everything)
	}
}

func TestProjectEntryContextResolvesSourceSlice(t *testing.T) {
	snap := buildSnapshot(t, `2000-01-01 open Assets:Bank:Cash USD
2000-01-01 open Equity:Opening USD
2002-02-03 * "Cafe" "Lunch" #meal
  Assets:Bank:Cash -2.50 USD
  Equity:Opening 2.50 USD
`)
	journal := ProjectJournal(snap.Evaluation(), snap.Graph(), report.Filters{Text: "#meal"}, report.JournalFilters{})
	if len(journal.Entries) != 1 || journal.Entries[0].EntryHash == "" {
		t.Fatalf("journal entry hash missing: %+v", journal.Entries)
	}
	hash := journal.Entries[0].EntryHash
	context, ok := ProjectEntryContext(snap.Evaluation(), snap.Graph(), hash)
	if !ok {
		t.Fatal("entry hash did not resolve")
	}
	if !strings.Contains(context.SourceSlice, `2002-02-03 * "Cafe" "Lunch" #meal`) || !strings.Contains(context.SourceSlice, "Assets:Bank:Cash -2.50 USD") {
		t.Fatalf("source slice does not reproduce the entry:\n%s", context.SourceSlice)
	}
	if context.SHA256Sum == "" || len(context.SHA256Sum) != 64 {
		t.Fatalf("sha256sum malformed: %q", context.SHA256Sum)
	}
	if context.Entry.Type != "transaction" || context.Entry.Span == "" {
		t.Fatalf("context entry incomplete: %+v", context.Entry)
	}
	if _, ok := ProjectEntryContext(snap.Evaluation(), snap.Graph(), "deadbeef"); ok {
		t.Fatal("unknown hash resolved")
	}
}

func TestSerializeNewEntries(t *testing.T) {
	text, err := SerializeNewEntries([]NewEntry{
		{
			Type: "transaction", Date: "2026-08-06", Payee: `Bob's "Cafe"`, Narration: "Lunch",
			Tags:  []string{"meal"},
			Links: []string{"trip"},
			Postings: []NewPosting{
				{Account: "Assets:Bank:Cash", Amount: "-2.50", Currency: "USD"},
				{Account: "Expenses:Food"},
			},
		},
		{Type: "balance", Date: "2026-08-06", Account: "Assets:Bank:Cash", Amount: "100.00", Currency: "USD"},
		{Type: "note", Date: "2026-08-06", Account: "Assets:Bank:Cash", Comment: "checked"},
	})
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	wantLines := []string{
		`2026-08-06 * "Bob's \"Cafe\"" "Lunch" #meal ^trip`,
		"  Assets:Bank:Cash -2.50 USD",
		"  Expenses:Food",
		"2026-08-06 balance Assets:Bank:Cash 100.00 USD",
		`2026-08-06 note Assets:Bank:Cash "checked"`,
	}
	for _, line := range wantLines {
		if !strings.Contains(text, line) {
			t.Fatalf("missing line %q in:\n%s", line, text)
		}
	}
	if strings.Count(text, "\n\n") != 2 {
		t.Fatalf("expected three blank-line-separated blocks:\n%q", text)
	}

	rejections := []NewEntry{
		{Type: "transaction", Date: "2026-8-6", Narration: "x", Postings: []NewPosting{{Account: "Assets:A:B", Amount: "1", Currency: "USD"}}},
		{Type: "transaction", Date: "2026-08-06", Flag: "?", Narration: "x", Postings: []NewPosting{{Account: "Assets:A:B"}}},
		{Type: "transaction", Date: "2026-08-06", Narration: "a\nb", Postings: []NewPosting{{Account: "Assets:A:B"}}},
		{Type: "transaction", Date: "2026-08-06", Narration: "x"},
		{Type: "balance", Date: "2026-08-06", Account: "assets:bad", Amount: "1", Currency: "USD"},
		{Type: "note", Date: "2026-08-06", Comment: "x"},
		{Type: "price", Date: "2026-08-06"},
	}
	for _, entry := range rejections {
		if _, err := SerializeNewEntries([]NewEntry{entry}); err == nil {
			t.Fatalf("entry %+v should be rejected", entry)
		}
	}
}

func TestUpdateActivityTracksLastEntryPerAccount(t *testing.T) {
	text := `2000-01-01 open Assets:Bank:Cash USD
2000-01-01 open Liabilities:Card USD
2000-01-01 open Income:Salary USD
2000-01-01 open Expenses:Food USD
2000-02-01 * "Pay"
  Assets:Bank:Cash 100 USD
  Income:Salary -100 USD
2000-03-01 * "Card"
  Liabilities:Card -20 USD
  Expenses:Food 20 USD
2000-04-01 * "Refund"
  Assets:Bank:Cash 5 USD
  Liabilities:Card -5 USD
`
	file, diagnostics := ledger.ParseText("activity.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse=%+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation=%+v", evaluation.Diagnostics)
	}

	rows := UpdateActivity(*evaluation)
	if len(rows) != 2 {
		t.Fatalf("expected 2 Assets/Liabilities rows, got %+v", rows)
	}
	if rows[0].Account != "Assets:Bank:Cash" || rows[1].Account != "Liabilities:Card" {
		t.Fatalf("rows must be sorted by account: %+v", rows)
	}
	if rows[0].LastEntryDate != "2000-04-01" || rows[1].LastEntryDate != "2000-04-01" {
		t.Fatalf("last entry must win: %+v", rows)
	}
	if rows[0].EntryHash == "" || rows[1].EntryHash == "" {
		t.Fatalf("entry hash must be populated: %+v", rows)
	}
	if rows[0].EntryHash != rows[1].EntryHash {
		t.Fatalf("both accounts share the same last entry: %+v", rows)
	}
	if got := rows[0].Balances["USD"]; got != "105" {
		t.Fatalf("Assets balance = %q, want 105", got)
	}
	if got := rows[1].Balances["USD"]; got != "-25" {
		t.Fatalf("Liabilities balance = %q, want -25", got)
	}
	for _, row := range rows {
		if strings.HasPrefix(row.Account, "Income") || strings.HasPrefix(row.Account, "Expenses") {
			t.Fatalf("non balance-sheet account leaked into update activity: %+v", row)
		}
	}
}
