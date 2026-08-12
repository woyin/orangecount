// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package quickentry

import (
	"strings"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

func evalFromText(t *testing.T, text string) *ledger.Evaluation {
	t.Helper()
	file, diagnostics := ledger.ParseText("test.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("parse errors: %+v", diagnostics.All())
	}
	evaluation := ledger.EvaluateFiles(map[source.FileID]*ledger.File{1: file}, []source.FileID{1}, ledger.EvalOptions{})
	if !evaluation.Valid {
		t.Fatalf("evaluation invalid: %+v", evaluation.Diagnostics)
	}
	return evaluation
}

func TestExplicitFormBasic(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Expenses:Food CNY
2000-01-01 open Equity:Opening CNY
2000-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat
2000-01-01 custom "orangecount.quick-account.v1" "餐饮" Expenses:Food
`)
	results := Compile(CompileRequest{
		Text:              `28 CNY @微信 -> @餐饮 : 工作午餐 #trip`,
		Date:              "2026-08-12",
		OperatingCurrency: "CNY",
		Evaluation:        evaluation,
	})
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d: %+v", len(results), results)
	}
	r := results[0]
	if len(r.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", r.Errors)
	}
	if r.Entry == nil {
		t.Fatalf("missing entry")
	}
	if r.Entry.Date != "2026-08-12" {
		t.Errorf("date=%s", r.Entry.Date)
	}
	if r.Entry.Flag != "*" {
		t.Errorf("flag=%s", r.Entry.Flag)
	}
	if r.Entry.Narration != "工作午餐" {
		t.Errorf("narration=%s", r.Entry.Narration)
	}
	if len(r.Entry.Tags) != 1 || r.Entry.Tags[0] != "trip" {
		t.Errorf("tags=%v", r.Entry.Tags)
	}
	if len(r.Entry.Postings) != 2 {
		t.Fatalf("postings=%v", r.Entry.Postings)
	}
	if r.Entry.Postings[0].Account != "Assets:WeChat" || r.Entry.Postings[0].Amount != "-28" || r.Entry.Postings[0].Currency != "CNY" {
		t.Errorf("source posting=%+v", r.Entry.Postings[0])
	}
	if r.Entry.Postings[1].Account != "Expenses:Food" || r.Entry.Postings[1].Amount != "28" || r.Entry.Postings[1].Currency != "CNY" {
		t.Errorf("dest posting=%+v", r.Entry.Postings[1])
	}
	// Preview should be canonical Beancount.
	if !strings.Contains(r.Preview, "2026-08-12 *") {
		t.Errorf("preview missing date line: %q", r.Preview)
	}
	if !strings.Contains(r.Preview, "Assets:WeChat") {
		t.Errorf("preview missing source account: %q", r.Preview)
	}
}

func TestTemplateInvocation(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Expenses:Food CNY
2000-01-01 open Equity:Opening CNY
2000-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat
2000-01-01 custom "orangecount.quick-template.v1" "午餐"
  destination: "Expenses:Food"
  source: "微信"
  currency: "CNY"
  narration: "午餐"
`)
	results := Compile(CompileRequest{
		Text:       "午餐 28",
		Date:       "2026-08-12",
		Evaluation: evaluation,
	})
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d: %+v", len(results), results)
	}
	r := results[0]
	if len(r.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", r.Errors)
	}
	if r.Entry.Narration != "午餐" {
		t.Errorf("narration=%s", r.Entry.Narration)
	}
	if r.Entry.Postings[0].Account != "Assets:WeChat" {
		t.Errorf("source=%s", r.Entry.Postings[0].Account)
	}
	if r.Entry.Postings[1].Account != "Expenses:Food" {
		t.Errorf("dest=%s", r.Entry.Postings[1].Account)
	}
	if r.Entry.Postings[0].Currency != "CNY" {
		t.Errorf("currency=%s", r.Entry.Postings[0].Currency)
	}
}

func TestBatchWithMultipleLines(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Expenses:Food CNY
2000-01-01 open Income:Salary CNY
2000-01-01 open Assets:Bank CNY
2000-01-01 open Equity:Opening CNY
2000-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat
2000-01-01 custom "orangecount.quick-account.v1" "餐饮" Expenses:Food
2000-01-01 custom "orangecount.quick-account.v1" "工资" Income:Salary
2000-01-01 custom "orangecount.quick-account.v1" "银行卡" Assets:Bank
2000-01-01 custom "orangecount.quick-template.v1" "午餐"
  source: "微信"
  destination: "餐饮"
  currency: "CNY"
2000-01-01 custom "orangecount.quick-template.v1" "咖啡"
  source: "微信"
  destination: "餐饮"
  currency: "CNY"
`)
	results := Compile(CompileRequest{
		Text: `午餐 28 @微信
咖啡 18 @微信

10000 CNY @工资 -> @银行卡 : 八月工资`,
		Date:              "2026-08-12",
		OperatingCurrency: "CNY",
		Evaluation:        evaluation,
	})
	if len(results) != 3 {
		t.Fatalf("want 3 results (empty lines skipped), got %d: %+v", len(results), results)
	}
	for i, r := range results {
		if len(r.Errors) != 0 {
			t.Fatalf("line %d errors: %+v", i+1, r.Errors)
		}
	}
	// Line 3: salary — direction puts Income as source (negative), Bank as dest.
	salary := results[2].Entry
	if salary.Postings[0].Account != "Income:Salary" {
		t.Errorf("salary source=%s", salary.Postings[0].Account)
	}
	if salary.Postings[1].Account != "Assets:Bank" {
		t.Errorf("salary dest=%s", salary.Postings[1].Account)
	}
}

func TestDuplicateDetection(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Expenses:Food CNY
2000-01-01 open Equity:Opening CNY
2000-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat
2000-01-01 custom "orangecount.quick-account.v1" "餐饮" Expenses:Food
2026-08-12 * "工作午餐"
  Assets:WeChat -28 CNY
  Expenses:Food 28 CNY
`)
	results := Compile(CompileRequest{
		Text: `28 CNY @微信 -> @餐饮 : 工作午餐`,
		Date: "2026-08-12",
		OperatingCurrency: "CNY",
		Evaluation:        evaluation,
	})
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	DetectDuplicates(results, evaluation)
	if !results[0].Duplicate {
		t.Errorf("expected duplicate flag on equivalent transaction")
	}
}

func TestHistoricalDateProfileResolution(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Assets:OldBank CNY
2000-01-01 open Assets:NewBank CNY
2000-01-01 open Equity:Opening CNY
2026-06-01 custom "orangecount.quick-account.v1" "银行卡" Assets:NewBank
`)
	// Transaction dated 2026-05-15 should resolve to... no earlier rule, so fails.
	results := Compile(CompileRequest{
		Text: `100 CNY @银行卡 -> @Assets:OldBank`,
		Date: "2026-05-15",
		OperatingCurrency: "CNY",
		Evaluation:        evaluation,
	})
	if len(results) != 1 {
		t.Fatalf("want 1, got %d", len(results))
	}
	if len(results[0].Errors) == 0 {
		t.Errorf("expected error for future-dated alias; the 2026-06-01 rule should not apply to a May transaction")
	}
	// Transaction dated 2026-07-01 should resolve to NewBank.
	results = Compile(CompileRequest{
		Text: `100 CNY @银行卡 -> @Assets:OldBank`,
		Date: "2026-07-01",
		OperatingCurrency: "CNY",
		Evaluation:        evaluation,
	})
	if len(results[0].Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", results[0].Errors)
	}
	if results[0].Entry.Postings[0].Account != "Assets:NewBank" {
		t.Errorf("expected NewBank, got %s", results[0].Entry.Postings[0].Account)
	}
}

func TestAmbiguousSameDayAccount(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Assets:BankA CNY
2000-01-01 open Assets:BankB CNY
2000-01-01 open Equity:Opening CNY
2026-01-01 custom "orangecount.quick-account.v1" "银行" Assets:BankA
2026-01-01 custom "orangecount.quick-account.v1" "银行" Assets:BankB
`)
	profile := EffectiveProfile(evaluation, ledger.Date{Year: 2026, Month: 6, Day: 1, Raw: "2026-06-01"})
	if len(profile.Accounts) != 0 {
		t.Errorf("expected no resolved accounts due to ambiguity, got %d", len(profile.Accounts))
	}
	found := false
	for _, p := range profile.Problems {
		if p.Code == "W-QUICK-ACCOUNT-AMBIGUOUS" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ambiguity problem in %+v", profile.Problems)
	}
}

func TestUnsupportedVersionIsIgnored(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Equity:Opening CNY
2000-01-01 custom "orangecount.quick-account.v2" "微信" Assets:WeChat
`)
	profile := EffectiveProfile(evaluation, ledger.Date{Year: 2026, Month: 6, Day: 1, Raw: "2026-06-01"})
	if len(profile.Accounts) != 0 {
		t.Errorf("unsupported version must not resolve, got %d accounts", len(profile.Accounts))
	}
	if len(profile.Problems) != 1 || profile.Problems[0].Code != "W-QUICK-SCHEMA-UNSUPPORTED" {
		t.Errorf("expected one unsupported-schema problem, got %+v", profile.Problems)
	}
}

func TestNegativeAmountRejected(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Expenses:Food CNY
2000-01-01 open Equity:Opening CNY
2000-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat
2000-01-01 custom "orangecount.quick-account.v1" "餐饮" Expenses:Food
`)
	results := Compile(CompileRequest{
		Text: `-28 CNY @微信 -> @餐饮`,
		Date: "2026-08-12",
		OperatingCurrency: "CNY",
		Evaluation:        evaluation,
	})
	if len(results[0].Errors) == 0 {
		t.Errorf("expected error for negative amount; quick-entry amounts must be positive")
	}
}

func TestIncomeDirectionEmitsBothAmounts(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Income:Salary CNY
2000-01-01 open Assets:Bank CNY
2000-01-01 open Equity:Opening CNY
2000-01-01 custom "orangecount.quick-account.v1" "工资" Income:Salary
2000-01-01 custom "orangecount.quick-account.v1" "银行卡" Assets:Bank
`)
	results := Compile(CompileRequest{
		Text: `10000 CNY @工资 -> @银行卡 : 八月工资`,
		Date: "2026-08-12",
		OperatingCurrency: "CNY",
		Evaluation:        evaluation,
	})
	r := results[0]
	if len(r.Errors) != 0 {
		t.Fatalf("unexpected errors: %+v", r.Errors)
	}
	// Both postings always carry explicit amounts.
	if r.Entry.Postings[0].Amount == "" || r.Entry.Postings[1].Amount == "" {
		t.Errorf("explicit output requires both posting amounts")
	}
	// Source is Income (value flows from salary to bank).
	if r.Entry.Postings[0].Account != "Income:Salary" {
		t.Errorf("income must be source: %s", r.Entry.Postings[0].Account)
	}
}

func TestInvalidDate(t *testing.T) {
	results := Compile(CompileRequest{Text: "28 CNY @a -> @b", Date: "not-a-date"})
	if len(results[0].Errors) == 0 || results[0].Errors[0].Code != "E-QUICK-DATE" {
		t.Errorf("expected E-QUICK-DATE, got %+v", results[0].Errors)
	}
}

func TestMissingCurrencyRejects(t *testing.T) {
	evaluation := evalFromText(t, `2000-01-01 open Assets:WeChat CNY
2000-01-01 open Expenses:Food CNY
2000-01-01 open Equity:Opening CNY
2000-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat
2000-01-01 custom "orangecount.quick-account.v1" "餐饮" Expenses:Food
`)
	// No operating currency, no per-line currency.
	results := Compile(CompileRequest{
		Text:       `28 @微信 -> @餐饮`,
		Date:       "2026-08-12",
		Evaluation: evaluation,
	})
	if len(results[0].Errors) == 0 {
		t.Errorf("expected currency error when no currency can be resolved")
	}
}

func TestProfileProblemsDoNotInvalidateEvaluation(t *testing.T) {
	// A profile with ambiguous rules should still let the ledger evaluate.
	evaluation := evalFromText(t, `2000-01-01 open Assets:BankA CNY
2000-01-01 open Assets:BankB CNY
2000-01-01 open Equity:Opening CNY
2026-01-01 custom "orangecount.quick-account.v1" "银行" Assets:BankA
2026-01-01 custom "orangecount.quick-account.v1" "银行" Assets:BankB
`)
	if !evaluation.Valid {
		t.Fatalf("profile problems must not invalidate ledger evaluation: %+v", evaluation.Diagnostics)
	}
}
