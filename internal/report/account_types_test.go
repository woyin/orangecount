// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package report

import (
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

// A ledger that renames its root accounts through the name_* options must
// produce the same statements as the default-root ledger it mirrors. Before
// classification followed the configured names, such a ledger was accepted and
// then silently dropped from every statement, which is worse than an error.
func TestRenamedRootsStillPopulateStatements(t *testing.T) {
	const text = `option "operating_currency" "CNY"
option "name_assets" "资产"
option "name_liabilities" "负债"
option "name_equity" "权益"
option "name_income" "收入"
option "name_expenses" "支出"
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
	file, diagnostics := ledger.ParseText("renamed.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse diagnostics: %+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{SemanticsProfile: ledger.V3})
	if !evaluation.Valid {
		t.Fatalf("ledger invalid: %+v", evaluation.Diagnostics)
	}

	balance := BalanceSheet(*evaluation)
	if len(balance.Rows) == 0 {
		t.Fatal("balance sheet is empty for a renamed-root ledger")
	}
	accounts := map[string]bool{}
	for _, row := range balance.Rows {
		accounts[row["account"].(string)] = true
	}
	for _, want := range []string{"资产", "资产:钱包:微信", "权益"} {
		if !accounts[want] {
			t.Errorf("balance sheet missing %q; got %v", want, accounts)
		}
	}

	income := IncomeStatement(*evaluation)
	names := map[string]bool{}
	for _, row := range income.Rows {
		names[row["account"].(string)] = true
	}
	for _, want := range []string{"收入:工资", "支出:餐饮"} {
		if !names[want] {
			t.Errorf("income statement missing %q; got %v", want, names)
		}
	}

	// The tree roots must be the configured names, not the canonical keys.
	tree := BuildAccountTree(*evaluation, "Assets", "Equity")
	if len(tree) != 2 || tree[0].FullName != "资产" || tree[1].FullName != "权益" {
		t.Fatalf("tree roots=%+v", tree)
	}
}

// An account whose root is not one of the five configured types cannot be
// classified, so it must be reported instead of vanishing from reports.
func TestUnclassifiedRootIsReported(t *testing.T) {
	const text = `2020-01-01 open Acme:Cash CNY
2020-01-01 open Equity:Opening CNY
2020-01-02 * "seed"
  Acme:Cash  1 CNY
  Equity:Opening  -1 CNY
`
	file, diagnostics := ledger.ParseText("unclassified.bean", []byte(text))
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
