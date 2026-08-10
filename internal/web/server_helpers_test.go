// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"net/http/httptest"
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
