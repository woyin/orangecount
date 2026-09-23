// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testFixturePath(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	base := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "fixtures", "v3-parity")
	return filepath.Join(append([]string{base}, parts...)...)
}

func TestBridgeLoadSnapshot(t *testing.T) {
	fixture := testFixturePath("txn-basic.bean")
	payload := loadSnapshot(fixture)
	if !payload.Valid {
		t.Fatalf("expected valid snapshot, errors: %v", payload.Errors)
	}
	if payload.EntriesCount != 4 {
		t.Fatalf("expected 4 entries, got %d", payload.EntriesCount)
	}
	if len(payload.Accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(payload.Accounts))
	}
}

func TestBridgeGetTreeReport(t *testing.T) {
	fixture := testFixturePath("txn-basic.bean")
	payload := loadTreeReport(fixture, "balance_sheet")
	if payload.Error != "" {
		t.Fatalf("unexpected error: %s", payload.Error)
	}
	if payload.ReportType != "balance_sheet" {
		t.Fatalf("unexpected report_type: %s", payload.ReportType)
	}
	if len(payload.RootNodes) != 2 { // Assets, Equity
		t.Fatalf("expected 2 root nodes, got %d", len(payload.RootNodes))
	}
	assets := payload.RootNodes[0]
	if assets.Name != "Assets" || assets.SubtreeBalances["USD"] != "100" {
		t.Fatalf("unexpected Assets node: %+v", assets)
	}
}

func TestBridgeGetJournal(t *testing.T) {
	fixture := testFixturePath("txn-basic.bean")
	payload := loadJournal(fixture, "")
	if payload.Error != "" {
		t.Fatalf("unexpected error: %s", payload.Error)
	}
	if payload.TotalCount != 1 { // txn-basic has 1 transaction
		t.Fatalf("expected 1 transaction, got %d", payload.TotalCount)
	}
	tx := payload.Transactions[0]
	if tx.Date != "2000-01-02" || tx.Payee != "payee" || tx.Narration != "narration" {
		t.Fatalf("unexpected transaction fields: %+v", tx)
	}
	if len(tx.Postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(tx.Postings))
	}
	if tx.Postings[0].Account != "Assets:Cash" || tx.Postings[0].UnitsNumber != "100.00" || tx.Postings[0].UnitsCurrency != "USD" {
		t.Fatalf("unexpected posting 0: %+v", tx.Postings[0])
	}
}

func TestBridgeGetHistoricalTrendsAndRevision(t *testing.T) {
	fixture := testFixturePath("txn-basic.bean")
	trends := loadHistoricalTrends(fixture)
	if trends.Error != "" {
		t.Fatalf("unexpected trends error: %s", trends.Error)
	}
	if trends.OperatingCurrency != "USD" {
		t.Fatalf("expected USD operating currency, got %s", trends.OperatingCurrency)
	}

	rev := getLedgerRevision(fixture)
	if len(rev) < 5 || rev[:4] != "rev:" {
		t.Fatalf("expected valid revision string, got %s", rev)
	}
}
