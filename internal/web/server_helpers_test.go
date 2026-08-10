// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportFilterParsingAndOptionValidationCoverSupportedVariants(t *testing.T) {
	filterRequest := func(raw string) (*httptest.ResponseRecorder, error) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("GET", "/api/v1/reports/journal?"+raw, nil)
		filters, err := globalReportFilters(request)
		if err == nil {
			recorder.Header().Set("account", filters.Account)
			recorder.Header().Set("period", filters.Period)
			recorder.Header().Set("begin", filters.TimeBegin)
			recorder.Header().Set("end", filters.TimeEnd)
			recorder.Header().Set("prefix", filters.TimePrefix)
		}
		return recorder, err
	}
	for raw, expected := range map[string]map[string]string{
		"account=Assets%3ACash&filter=tag%3Afood&period=quarter&time=2024-Q4": {"account": "Assets:Cash", "period": "quarter", "begin": "2024-10-01", "end": "2025-01-01"},
		"time=2024-02": {"prefix": "2024-02"},
		"time=2024":    {"prefix": "2024"},
		"time=all":     {},
	} {
		recorder, err := filterRequest(raw)
		if err != nil {
			t.Errorf("filters %q err=%v", raw, err)
			continue
		}
		for key, value := range expected {
			if got := recorder.Header().Get(key); got != value {
				t.Errorf("filters %q %s=%q, want %q", raw, key, got, value)
			}
		}
	}
	for _, raw := range []string{"filter=unclosed%3A%28", "period=week", "valuation=units", "time=2024-Q5", "time=bad"} {
		if _, err := filterRequest(raw); err == nil {
			t.Errorf("filters %q succeeded", raw)
		}
	}
	request := httptest.NewRequest("GET", "/api/v1/reports/journal?from=2024-02-02&to=2024-02-01", nil)
	if _, _, err := journalDateRange(request); err == nil {
		t.Fatal("reversed journal range succeeded")
	}
	if date, err := parseISODate("2024-02-29"); err != nil || date == nil || date.Raw != "2024-02-29" {
		t.Fatalf("parsed date=%+v err=%v", date, err)
	}
	for _, raw := range []string{"invalid", "0000-01-01"} {
		if _, err := parseISODate(raw); err == nil {
			t.Errorf("parseISODate(%q) succeeded", raw)
		}
	}
	if _, err := reportAsOfDate(httptest.NewRequest("GET", "/?as_of=invalid", nil)); err == nil {
		t.Fatal("invalid as-of date succeeded")
	}
	for key, value := range map[string]string{"locale": "zh-CN", "currency": "USD", "time": "month"} {
		if err := validateLocalOption(key, value); err != nil {
			t.Errorf("valid option %s=%s: %v", key, value, err)
		}
	}
	for key, value := range map[string]string{"locale": "fr", "currency": "usd", "time": "quarter", "unknown": "value"} {
		if err := validateLocalOption(key, value); err == nil {
			t.Errorf("invalid option %s=%s succeeded", key, value)
		}
	}
}

func TestImportAndAtomicWriteHelpersRejectUnsafeInputAndPreserveContent(t *testing.T) {
	for _, tc := range []struct {
		raw, adapter, want string
	}{
		{"ledger.bean", "beancount", "ledger.bean"}, {"ledger.beancount", "beancount", "ledger.beancount"}, {"bank.csv", "csv", "bank.csv"},
	} {
		got, err := safeImportName(tc.raw, tc.adapter)
		if err != nil || got != tc.want {
			t.Errorf("safeImportName(%q, %q)=%q, %v", tc.raw, tc.adapter, got, err)
		}
	}
	for _, tc := range [][2]string{{"", "beancount"}, {"../ledger.bean", "beancount"}, {"ledger.csv", "beancount"}, {"ledger.bean", "csv"}} {
		if _, err := safeImportName(tc[0], tc[1]); err == nil {
			t.Errorf("unsafe import %q/%q accepted", tc[0], tc[1])
		}
	}
	converted, err := csvToBeancount("date,payee,account,amount,currency,narration\n2000-01-02,Cafe,Assets:Cash,1.50,USD,\"note \"\"quoted\"\"\"\n", map[string]string{"offset_account": "Equity:Opening"})
	if err != nil || !strings.Contains(converted, `"note \"quoted\""`) || !strings.Contains(converted, "Equity:Opening -1.5 USD") {
		t.Fatalf("converted=%q err=%v", converted, err)
	}
	for _, content := range []string{"date,account,amount\n", "date,account\n2000-01-01,Assets:Cash\n", "date,account,amount\ninvalid,Assets:Cash,1\n", "date,account,amount\n2000-01-01,Assets:Cash,nope\n"} {
		if _, err := csvToBeancount(content, nil); err == nil {
			t.Errorf("invalid CSV accepted: %q", content)
		}
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.txt")
	if err := atomicWrite(path, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(path, []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "second" {
		t.Fatalf("atomic contents=%q err=%v", data, err)
	}
	if err := atomicWrite(filepath.Join(dir, "missing", "file"), []byte("nope"), 0o600); err == nil {
		t.Fatal("atomic write to missing directory succeeded")
	}
}
