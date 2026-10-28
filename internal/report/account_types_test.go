// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package report

import (
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/source"
)

func evaluateReportText(t *testing.T, name, text string) *ledger.Evaluation {
	t.Helper()
	file, diagnostics := ledger.ParseText(name, []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics: %+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{SemanticsProfile: ledger.V3})
	if !evaluation.Valid {
		t.Fatalf("ledger invalid: %+v", evaluation.Diagnostics)
	}
	return evaluation
}

func accountsIn(rows []query.Row) map[string]bool {
	accounts := map[string]bool{}
	for _, row := range rows {
		if account, ok := row["account"].(string); ok {
			accounts[account] = true
		}
	}
	return accounts
}

const cnSchemeLedger = `option "operating_currency" "CNY"
option "account_root_scheme" "cn"
2020-01-01 open 资产:钱包:微信 CNY
2020-01-01 open 权益:期初 CNY
2020-01-01 open 收入:工资 CNY
2020-01-01 open 支出:餐饮 CNY
2020-01-02 * "期初"
  资产:钱包:微信  1000 CNY
  权益:期初  -1000 CNY
2020-01-03 * "午饭"
  支出:餐饮  30 CNY
  资产:钱包:微信  -30 CNY
2020-01-04 * "工资"
  资产:钱包:微信  5000 CNY
  收入:工资  -5000 CNY
`

// The cn scheme makes the five Chinese roots exact equivalents of the English
// ones with no per-type option, so the statements must populate without any
// name_* line. Before classification followed the accepted vocabulary, such a
// ledger was accepted and then silently dropped from every report.
func TestCNSchemePopulatesStatements(t *testing.T) {
	evaluation := evaluateReportText(t, "cn.bean", cnSchemeLedger)

	balance := BalanceSheet(*evaluation)
	if len(balance.Rows) == 0 {
		t.Fatal("balance sheet is empty for a cn-scheme ledger")
	}
	accounts := accountsIn(balance.Rows)
	for _, want := range []string{"资产", "资产:钱包:微信", "权益", "权益:期初"} {
		if !accounts[want] {
			t.Errorf("balance sheet missing %q; got %v", want, accounts)
		}
	}

	income := accountsIn(IncomeStatement(*evaluation).Rows)
	for _, want := range []string{"收入:工资", "支出:餐饮"} {
		if !income[want] {
			t.Errorf("income statement missing %q; got %v", want, income)
		}
	}

	tree := BuildAccountTree(*evaluation, "Assets", "Equity")
	if len(tree) != 2 || tree[0].FullName != "资产" || tree[1].FullName != "权益" {
		t.Fatalf("tree roots=%+v", tree)
	}
}

// Both vocabularies are accepted inside one cn-scheme ledger, so the statements
// must carry both subtrees instead of one spelling replacing the other.
func TestCNSchemeKeepsBothSpellings(t *testing.T) {
	evaluation := evaluateReportText(t, "mixed.bean", `option "operating_currency" "CNY"
option "account_root_scheme" "cn"
2020-01-01 open 资产:钱包 CNY
2020-01-01 open Assets:Cash CNY
2020-01-01 open 权益:期初 CNY
2020-01-02 * "seed"
  资产:钱包  10 CNY
  Assets:Cash  20 CNY
  权益:期初  -30 CNY
`)
	accounts := accountsIn(BalanceSheet(*evaluation).Rows)
	for _, want := range []string{"资产", "资产:钱包", "Assets", "Assets:Cash", "权益"} {
		if !accounts[want] {
			t.Errorf("balance sheet missing %q; got %v", want, accounts)
		}
	}
	tree := BuildAccountTree(*evaluation, "Assets")
	if len(tree) != 2 {
		t.Fatalf("expected one subtree per spelling, got %+v", tree)
	}
}

// A Chinese-scheme ledger written in English keeps English chart labels, so the
// label follows the spelling the ledger uses rather than the scheme.
func TestDisplayRootFollowsLedgerSpelling(t *testing.T) {
	cn := evaluateReportText(t, "cn.bean", cnSchemeLedger)
	if got := cn.DisplayRoot("Assets"); got != "资产" {
		t.Fatalf("cn ledger label=%q want 资产", got)
	}
	english := evaluateReportText(t, "en.bean", `option "account_root_scheme" "cn"
2020-01-01 open Assets:Cash CNY
`)
	if got := english.DisplayRoot("Assets"); got != "Assets" {
		t.Fatalf("english ledger label=%q want Assets", got)
	}
}

// An account whose root is neither the configured name nor an accepted alias
// must be reported instead of vanishing from reports.
func TestUnclassifiedRootIsReported(t *testing.T) {
	file, diagnostics := ledger.ParseText("unclassified.bean", []byte(`2020-01-01 open Acme:Cash CNY
2020-01-01 open Equity:Opening CNY
2020-01-02 * "seed"
  Acme:Cash  1 CNY
  Equity:Opening  -1 CNY
`))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics: %+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{SemanticsProfile: ledger.V3})
	found := false
	for _, d := range evaluation.Diagnostics {
		if d.Code == "E-EVAL-ACCOUNT" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unclassified root produced no E-EVAL-ACCOUNT: %+v", evaluation.Diagnostics)
	}
	if evaluation.Valid {
		t.Fatal("an unclassifiable root must invalidate the ledger")
	}
}

// The global scheme must not accept a Chinese root, so the account error fires
// with no cn option present.
func TestGlobalSchemeRejectsChineseRootAtEvaluation(t *testing.T) {
	file, diagnostics := ledger.ParseText("global.bean", []byte("2020-01-01 open 资产:钱包 CNY\n"))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics: %+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{SemanticsProfile: ledger.V3})
	found := false
	for _, d := range evaluation.Diagnostics {
		if d.Code == "E-EVAL-ACCOUNT" {
			found = true
		}
	}
	if !found {
		t.Fatalf("global scheme accepted a Chinese root: %+v", evaluation.Diagnostics)
	}
}
