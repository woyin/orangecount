// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dialect_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

// buildLedger writes text to a temp ledger and builds a snapshot. It returns
// the build result so tests can assert both balances and diagnostics.
func buildLedger(t *testing.T, text string) snapshot.BuildResult {
	t.Helper()
	entry := filepath.Join(t.TempDir(), "main.bean")
	if err := os.WriteFile(entry, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return snapshot.Build(entry)
}

func diagCodes(result snapshot.BuildResult) []string {
	var codes []string
	for _, d := range result.Diagnostics {
		codes = append(codes, d.Code)
	}
	return codes
}

func hasCode(result snapshot.BuildResult, code string) bool {
	for _, c := range diagCodes(result) {
		if c == code {
			return true
		}
	}
	return false
}

const baseOpens = `2000-01-01 open Assets:WeChat USD
2000-01-01 open Assets:Cash USD
2000-01-01 open Expenses:Food USD
2000-01-01 open Income:Salary USD
2000-01-01 open Assets:Bank:Checking USD
option "operating_currency" "USD"
`

func TestDialectLineCompilesIntoBalancedTransaction(t *testing.T) {
	result := buildLedger(t,
		"2000-01-01 open Assets:WeChat USD,CNY\n2000-01-01 open Assets:Cash USD,CNY\n"+
			"option \"operating_currency\" \"USD\"\n"+
			"2026-08-12 28 CNY @Assets:WeChat -> @Assets:Cash : 午餐 #food ^link1\n")
	if result.Snapshot == nil {
		t.Fatalf("snapshot nil: %v", diagCodes(result))
	}
	accounts := result.Snapshot.Evaluation().Accounts
	if got := accounts["Assets:WeChat"].Balances["CNY"].String(); got != "-28" {
		t.Fatalf("WeChat CNY=%s", got)
	}
	if got := accounts["Assets:Cash"].Balances["CNY"].String(); got != "28" {
		t.Fatalf("Cash CNY=%s", got)
	}
	entries := result.Snapshot.Evaluation().Entries
	if len(entries) == 0 {
		t.Fatal("no entries")
	}
	found := false
	for _, entry := range entries {
		if strings.Contains(filepath.ToSlash(entry.File), "main.bean") {
			found = true
		}
	}
	if !found {
		t.Fatal("dialect transaction missing from entries")
	}
}

func TestDialectDetectionRules(t *testing.T) {
	// A standard transaction stays standard; the natural-gap shapes become
	// dialect lines without disturbing keyword directives.
	result := buildLedger(t, baseOpens+
		"2026-01-01 * \"标准\" \"交易\"\n  Assets:WeChat -5 USD\n  Assets:Cash 5 USD\n"+
		"2026-01-02 10 USD @Assets:Cash -> @Assets:WeChat\n"+
		"15 USD @Assets:Cash -> @Assets:WeChat\n"+
		"2026-01-03 ! 20 USD @Assets:Cash -> @Assets:WeChat\n")
	if result.Snapshot == nil {
		t.Fatalf("snapshot nil: %v", diagCodes(result))
	}
	accounts := result.Snapshot.Evaluation().Accounts
	// -5 +10 +15 +20 = 40 on WeChat; Cash mirrors.
	if got := accounts["Assets:WeChat"].Balances["USD"].String(); got != "40" {
		t.Fatalf("WeChat USD=%s", got)
	}
	if got := accounts["Assets:Cash"].Balances["USD"].String(); got != "-40" {
		t.Fatalf("Cash USD=%s", got)
	}
}

func TestBlockAnchoring(t *testing.T) {
	result := buildLedger(t, baseOpens+
		"2026-08-12 28 USD @Assets:Cash -> @Expenses:Food : a\n"+
		"5 USD @Assets:Cash -> @Expenses:Food : b\n"+
		"2026-08-13 7 USD @Assets:Cash -> @Expenses:Food : c\n"+
		"3 USD @Assets:Cash -> @Expenses:Food : d\n")
	if result.Snapshot == nil {
		t.Fatalf("snapshot nil: %v", diagCodes(result))
	}
	// 5 anchors to 2026-08-12 and 3 anchors to 2026-08-13: the evaluation
	// exposes dates only via entries; assert via journal order stability by
	// counting entries at each date.
	byDate := map[string]int{}
	for _, entry := range result.Snapshot.Evaluation().Entries {
		byDate[entry.Date.Raw]++
	}
	if byDate["2026-08-12"] != 2 || byDate["2026-08-13"] != 2 {
		t.Fatalf("anchor dates wrong: %v", byDate)
	}
}

func TestBlockAnchoringRequiresDialectAnchor(t *testing.T) {
	// A dated standard transaction/open between dialect lines must not become
	// an anchor; the undated line below has no dialect anchor above it.
	result := buildLedger(t, baseOpens+
		"2026-08-12 * \"p\" \"n\"\n  Assets:Cash -1 USD\n  Expenses:Food 1 USD\n"+
		"5 USD @Assets:Cash -> @Expenses:Food : late\n")
	if result.Snapshot != nil {
		t.Fatal("unanchored dialect line published a snapshot")
	}
	if !hasCode(result, "E-DIALECT-DATE") {
		t.Fatalf("codes=%v", diagCodes(result))
	}
}

func TestThreeLevelEndpointResolution(t *testing.T) {
	// Level 1 full name, level 2 declared alias (date-effective), level 3
	// unique tail segment.
	result := buildLedger(t, baseOpens+
		"2001-01-01 custom \"orangecount.quick-account.v1\" \"工资\" Income:Salary\n"+
		"2026-08-12 100 USD @工资 -> @Assets:Bank:Checking\n"+ // alias
		"2026-08-13 4 USD @Checking -> @Assets:Cash\n"+ // tail: Assets:Bank:Checking
		"5 USD @Cash -> @Assets:WeChat\n") // tail: Assets:Cash
	if result.Snapshot == nil {
		t.Fatalf("snapshot nil: %v", diagCodes(result))
	}
	accounts := result.Snapshot.Evaluation().Accounts
	if got := accounts["Income:Salary"].Balances["USD"].String(); got != "-100" {
		t.Fatalf("alias Income:Salary=%s", got)
	}
	// Balances are cumulative: +100 then -4 on Checking; +4 then -5 on Cash.
	if got := accounts["Assets:Bank:Checking"].Balances["USD"].String(); got != "96" {
		t.Fatalf("alias dest=%s", got)
	}
	if got := accounts["Assets:Cash"].Balances["USD"].String(); got != "-1" {
		t.Fatalf("tail Cash=%s", got)
	}
}

func TestAliasIsDateEffective(t *testing.T) {
	// The alias directive is dated after the transaction, so it must not
	// apply; with no full-name or unique-tail fallback either, the line is an
	// honest endpoint error rather than a retroactive mapping.
	result := buildLedger(t, baseOpens+
		"2027-01-01 custom \"orangecount.quick-account.v1\" \"工资\" Income:Salary\n"+
		"2026-08-12 100 USD @工资 -> @Assets:Cash\n")
	if result.Snapshot != nil || !hasCode(result, "E-DIALECT-ACCOUNT") {
		t.Fatalf("codes=%v snapshot=%v", diagCodes(result), result.Snapshot != nil)
	}
}

func TestAmbiguousTailReportsCandidates(t *testing.T) {
	result := buildLedger(t, baseOpens+
		"2000-01-01 open Expenses:Food:Lunch USD\n"+
		"2000-01-01 open Expenses:Travel:Lunch USD\n"+
		"2026-08-12 5 USD @Assets:Cash -> @Lunch\n")
	if result.Snapshot != nil {
		t.Fatal("ambiguous tail published a snapshot")
	}
	if !hasCode(result, "E-DIALECT-AMBIGUOUS") {
		t.Fatalf("codes=%v", diagCodes(result))
	}
	joined := ""
	for _, d := range result.Diagnostics {
		if d.Code == "E-DIALECT-AMBIGUOUS" {
			joined = d.Message
		}
	}
	if !strings.Contains(joined, "Expenses:Food") {
		t.Fatalf("candidates not listed: %q", joined)
	}
}

func TestUnknownEndpointFails(t *testing.T) {
	result := buildLedger(t, baseOpens+
		"2026-08-12 5 USD @Assets:Cash -> @Nowhere\n")
	if result.Snapshot != nil || !hasCode(result, "E-DIALECT-ACCOUNT") {
		t.Fatalf("codes=%v snapshot=%v", diagCodes(result), result.Snapshot != nil)
	}
}

func TestCurrencyAndNarrationDefaults(t *testing.T) {
	// Omitted currency uses the single operating_currency; omitted narration
	// defaults to 消费; flag defaults to *.
	result := buildLedger(t, baseOpens+
		"2026-08-12 12 @Assets:Cash -> @Expenses:Food\n")
	if result.Snapshot == nil {
		t.Fatalf("snapshot nil: %v", diagCodes(result))
	}
	if got := result.Snapshot.Evaluation().Accounts["Expenses:Food"].Balances["USD"].String(); got != "12" {
		t.Fatalf("Food USD=%s", got)
	}
	var narration, flag string
	for _, entry := range result.Snapshot.Evaluation().Entries {
		if tx, ok := entry.Directive.(ledger.Transaction); ok {
			narration, flag = tx.Narration, tx.Flag
		} else if tx, ok := entry.Directive.(*ledger.Transaction); ok {
			narration, flag = tx.Narration, tx.Flag
		}
	}
	if narration != "消费" {
		t.Fatalf("narration=%q", narration)
	}
	if flag != "*" {
		t.Fatalf("flag=%q", flag)
	}
}

func TestCurrencyOmissionWithoutOperatingCurrencyFails(t *testing.T) {
	result := buildLedger(t,
		"2000-01-01 open Assets:Cash USD\n2000-01-01 open Expenses:Food USD\n"+
			"2026-08-12 5 @Assets:Cash -> @Expenses:Food\n")
	if result.Snapshot != nil || !hasCode(result, "E-DIALECT-CURRENCY") {
		t.Fatalf("codes=%v", diagCodes(result))
	}
}

func TestDialectSyntaxErrors(t *testing.T) {
	cases := []struct{ code, line string }{
		{"E-DIALECT-ARROW", "2026-08-12 5 USD @Assets:Cash Expenses:Food"},
		{"E-DIALECT-AMOUNT", "2026-08-12 -5 USD @Assets:Cash -> @Expenses:Food"},
		// A non-amount word after the date is a standard v3 syntax error,
		// not a dialect line: detection only claims amount-shaped starts.
		{"E-PARSE-DIRECTIVE", "2026-08-12 abc USD @Assets:Cash -> @Expenses:Food"},
		{"E-DIALECT-SYNTAX", "2026-08-12 5 USD @Assets:Cash -> @Expenses:Food \"p1\" \"p2\""},
		{"E-DIALECT-SYNTAX", "2026-08-12 5 USD Assets:Cash -> Expenses:Food"},
	}
	for _, tc := range cases {
		result := buildLedger(t, baseOpens+tc.line+"\n")
		if result.Snapshot != nil || !hasCode(result, tc.code) {
			t.Errorf("line %q: want %s, codes=%v", tc.line, tc.code, diagCodes(result))
		}
	}
}

func TestDialectErrorsBlockSnapshot(t *testing.T) {
	// FD-0004 semantics: E-DIALECT-* are error diagnostics and must prevent
	// publication.
	result := buildLedger(t, baseOpens+
		"2026-08-12 5 USD @Assets:Cash -> @Missing\n")
	if result.Snapshot != nil {
		t.Fatal("dialect error published a snapshot")
	}
}

// TestDialectBlockCompilesIntoBalancedTransaction exercises the block form:
// a standard header followed by indented amount-first legs. Each leg becomes
// a source/destination posting pair, and same-account postings merge so the
// standard export shows one source amount.
func TestDialectBlockCompilesIntoBalancedTransaction(t *testing.T) {
	entry := writeFile(t, "loan.bean", `2000-01-01 open Assets:Bank:华夏0139 CNY
2000-01-01 open Liabilities:Loan:房贷:营苑东村 CNY
2000-01-01 open Expenses:Interest:营苑东村 CNY
option "operating_currency" "CNY"
2021-05-20 * "我" "还房贷"
  1,472.22 CNY @Assets:Bank:华夏0139 -> @Liabilities:Loan:房贷:营苑东村
  1,275.69 CNY @Assets:Bank:华夏0139 -> @Expenses:Interest:营苑东村
`)
	result := snapshot.Build(entry)
	if result.Err != nil {
		t.Fatalf("build: %v", result.Err)
	}
	if result.Snapshot == nil {
		t.Fatalf("snapshot nil: %v", result.Diagnostics)
	}
	file := result.Snapshot.Parsed()[source.FileID(1)]
	var txns []*ledger.Transaction
	for _, d := range file.Directives {
		if tx, ok := d.(*ledger.Transaction); ok {
			txns = append(txns, tx)
		}
	}
	if len(txns) != 1 {
		t.Fatalf("compiled transactions=%d", len(txns))
	}
	tx := txns[0]
	if tx.Narration != "还房贷" || tx.Payee != "我" || len(tx.Postings) != 3 {
		t.Fatalf("txn=%+v postings=%d", tx, len(tx.Postings))
	}
	byAccount := map[string]string{}
	for _, p := range tx.Postings {
		byAccount[p.Account] = p.Units.Number.Raw
	}
	if byAccount["Assets:Bank:华夏0139"] != "-2747.91" {
		t.Fatalf("source merged amount=%q want -2747.91", byAccount["Assets:Bank:华夏0139"])
	}
	if byAccount["Liabilities:Loan:房贷:营苑东村"] != "1472.22" {
		t.Fatalf("principal=%q", byAccount["Liabilities:Loan:房贷:营苑东村"])
	}
	if byAccount["Expenses:Interest:营苑东村"] != "1275.69" {
		t.Fatalf("interest=%q", byAccount["Expenses:Interest:营苑东村"])
	}
}

// TestDialectBlockRoundTrip exercises the 房贷 shape through dialectize and
// export: balances are preserved and the fixpoint is byte-stable.
func TestDialectBlockRoundTrip(t *testing.T) {
	const v3 = `2000-01-01 open Assets:Bank:华夏0139 CNY
2000-01-01 open Liabilities:Loan:房贷:营苑东村 CNY
2000-01-01 open Expenses:Interest:营苑东村 CNY
option "operating_currency" "CNY"

2021-05-20 * "我" "还房贷"
  Assets:Bank:华夏0139 -2,747.91 CNY
  Liabilities:Loan:房贷:营苑东村 1,472.22 CNY
  Expenses:Interest:营苑东村 1,275.69 CNY
`
	original := writeFile(t, "loan.bean", v3)
	blockLedger := dialectizeFile(t, original)
	blockText := readFile(t, blockLedger)
	if !strings.Contains(blockText, `* "我" "还房贷"`) {
		t.Fatalf("block header missing:\n%s", blockText)
	}
	if !strings.Contains(blockText, "1472.22 CNY @Assets:Bank:华夏0139 -> @Liabilities:Loan:房贷:营苑东村") {
		t.Fatalf("principal leg missing:\n%s", blockText)
	}
	// export the block back to v3 and confirm balance equality.
	exported := exportFile(t, blockLedger)
	exportedText := readFile(t, exported)
	if !strings.Contains(exportedText, "Assets:Bank:华夏0139 -2747.91 CNY") {
		t.Fatalf("merged source not restored:\n%s", exportedText)
	}
	assertBalancesEqual(t, original, exported)
}

func assertBalancesEqual(t *testing.T, a, b string) {
	t.Helper()
	ba := balancesOf(t, a)
	bb := balancesOf(t, b)
	if !reflect.DeepEqual(ba, bb) {
		t.Fatalf("balances diverged:\nbefore=%v\nafter=%v", ba, bb)
	}
}

// TestDialectBlockManySourcesUsesEachSourceAmount guards the inverse shape:
// several negative postings into one positive posting must emit one leg per
// source with that source's own positive magnitude, never the shared target
// amount and never a negative sign.
func TestDialectBlockManySourcesUsesEachSourceAmount(t *testing.T) {
	const v3 = `2000-01-01 open Assets:Bank:工行4515 CNY
2000-01-01 open Expenses:Social:资助朋友 CNY
option "operating_currency" "CNY"

2025-08-15 * "小黑" "亲属卡消费"
  Assets:Bank:工行4515 -59.80 CNY
  Assets:Bank:工行4515 -24.66 CNY
  Expenses:Social:资助朋友 84.46 CNY
`
	original := writeFile(t, "many-src.bean", v3)
	blockLedger := dialectizeFile(t, original)
	blockText := readFile(t, blockLedger)
	if strings.Contains(blockText, "-59.80") || strings.Contains(blockText, "-24.66") {
		t.Fatalf("legs must be positive magnitudes:\n%s", blockText)
	}
	for _, want := range []string{
		" 59.8 CNY @Assets:Bank:工行4515 -> @Expenses:Social:资助朋友",
		" 24.66 CNY @Assets:Bank:工行4515 -> @Expenses:Social:资助朋友",
	} {
		if !strings.Contains(blockText, want) {
			t.Fatalf("leg %q missing:\n%s", want, blockText)
		}
	}
	assertBalancesEqual(t, original, blockLedger)
	exported := exportFile(t, blockLedger)
	assertBalancesEqual(t, original, exported)
}
