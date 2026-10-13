// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package datefmt_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"orangecount/internal/datefmt"
	"orangecount/internal/ledger"
)

// weekCases maps each layout/value pair from the tracked CSV week-date
// evidence (docs/evidence/csv-week-date-2026-10-12.json) to the date every
// supplied field agrees on, or to the failure class when no such date exists.
// Valid expectations were re-derived from the pinned Python reference
// (datetime.strptime plus the candidate's round-trip rule); the rejected rows
// are the candidate's documented safety refusals.
var weekCases = []struct {
	name    string
	format  string
	value   string
	date    string
	code    datefmt.Code
	message string
}{
	{
		name: "iso-cross-year-week-one", format: "%G-W%V-%u", value: "2020-W01-1", date: "2019-12-30",
	},
	{
		name: "iso-cross-year-week-fifty-three", format: "%G-W%V-%u", value: "2020-W53-7", date: "2021-01-03",
	},
	{
		name: "iso-week-53-that-does-not-exist", format: "%G-W%V-%u", value: "2021-W53-1", code: datefmt.CodeRange,
		message: "Invalid week: 53",
	},
	{
		name: "iso-week-zero", format: "%G-W%V-%u", value: "2020-W00-1", code: datefmt.CodeMismatch,
	},
	{
		name: "sunday-week-zero-rolls-to-previous-year", format: "%Y %U %w", value: "2021 00 0", code: datefmt.CodeRange,
	},
	{
		name: "sunday-week-one", format: "%Y %U %w", value: "2021 01 0", date: "2021-01-03",
	},
	{
		name: "sunday-week-53-rolls-past-declared-year", format: "%Y %U %w", value: "2021 53 6", code: datefmt.CodeRange,
	},
	{
		name: "monday-week-zero-rolls-to-previous-year", format: "%Y %W %w", value: "2021 00 1", code: datefmt.CodeRange,
	},
	{
		name: "monday-week-one", format: "%Y %W %w", value: "2021 01 1", date: "2021-01-04",
	},
	{
		name: "sunday-week-without-weekday", format: "%Y %U", value: "2024 40", code: datefmt.CodeSyntax,
	},
	{
		name: "week-without-year", format: "%U %w", value: "40 1", code: datefmt.CodeSyntax,
	},
	{
		name: "calendar-and-week-disagree", format: "%Y-%m-%d %U %w", value: "2024-01-01 02 0", code: datefmt.CodeConflict,
	},
	{
		name: "iso-and-calendar-disagree", format: "%G-W%V-%u %Y-%m-%d", value: "2020-W01-1 2024-05-01", code: datefmt.CodeConflict,
	},
	{
		name: "iso-week-without-weekday", format: "%G-W%V", value: "2024-W20", code: datefmt.CodeSyntax,
	},
	{
		name: "iso-week-with-calendar-year", format: "%Y-W%V-%u", value: "2024-W20-1", code: datefmt.CodeSyntax,
	},
	{
		name: "plain-calendar-date", format: "%Y-%m-%d", value: "2024-05-01", date: "2024-05-01",
	},
	{
		name: "iso-weekday-seven-is-sunday", format: "%G-W%V-%u", value: "2024-W01-7", date: "2024-01-07",
	},
	{
		name: "sunday-week-00-belongs-to-previous-year", format: "%Y %U %w", value: "2023 00 0", code: datefmt.CodeRange,
	},
	{
		name: "two-digit-year", format: "%y-%m-%d", value: "24-05-01", date: "2024-05-01",
	},
	{
		name: "impossible-calendar-day", format: "%Y-%m-%d", value: "2023-02-29", code: datefmt.CodeRange,
	},
	{
		name: "month-out-of-range", format: "%Y-%m-%d", value: "2023-13-01", code: datefmt.CodeRange,
	},
	{
		name: "unsupported-directive", format: "%Y-%j", value: "2024-100", code: datefmt.CodeSyntax,
	},
	{
		name: "value-does-not-match-layout", format: "%Y-%m-%d", value: "01/05/2024", code: datefmt.CodeMismatch,
	},
	{
		name: "repeated-field", format: "%Y %Y-%m-%d", value: "2024 2024-05-01", code: datefmt.CodeSyntax,
	},
	{
		name: "trailing-value-text", format: "%Y-%m-%d", value: "2024-05-01T00:00", code: datefmt.CodeMismatch,
	},
}

// TestParseMatchesCapturedCSVWeekEvidence pins the parsed dates and the
// rejection classes for the captured CSV week-date cases. The date field is
// the exact calendar date; a rejected value must never return one.
func TestParseMatchesCapturedCSVWeekEvidence(t *testing.T) {
	for _, test := range weekCases {
		t.Run(test.name, func(t *testing.T) {
			date, err := datefmt.Parse(test.format, test.value)
			if test.code == "" {
				if err != nil {
					t.Fatalf("Parse(%q, %q) err=%v want %s", test.format, test.value, err, test.date)
				}
				if got := formatDate(date); got != test.date {
					t.Fatalf("Parse(%q, %q)=%s want %s", test.format, test.value, got, test.date)
				}
				if date.Year <= 0 || !date.Valid() {
					t.Fatalf("parsed date %+v is not a real calendar date", date)
				}
				return
			}
			if err == nil {
				t.Fatalf("Parse(%q, %q)=%s want failure %s", test.format, test.value, formatDate(date), test.code)
			}
			var failure datefmt.Error
			if !errors.As(err, &failure) {
				t.Fatalf("Parse(%q, %q) err=%v is not a datefmt.Error", test.format, test.value, err)
			}
			if failure.Code != test.code {
				t.Fatalf("Parse(%q, %q) code=%s want %s (message %q)", test.format, test.value, failure.Code, test.code, failure.Message)
			}
			if test.message != "" && failure.Message != test.message {
				t.Fatalf("Parse(%q, %q) message=%q want %q", test.format, test.value, failure.Message, test.message)
			}
			if date.Year != 0 || date.Month != 0 || date.Day != 0 || date.Raw != "" {
				t.Fatalf("a failed parse returned a partial date: %+v", date)
			}
		})
	}
}

// TestParseRejectsEveryFieldOfAMultiRowFileIndependently proves the strict
// contract is per value: one accepted row in the captured case does not make
// its rejected sibling acceptable, and no failure leaves state behind.
func TestParseRejectsEveryFieldOfAMultiRowFileIndependently(t *testing.T) {
	// Captured case "sunday_cross_year" mixes an accepted row with two that
	// cannot round-trip in 2021.
	accepted, err := datefmt.Parse("%Y %U %w", "2021 01 0")
	if err != nil || formatDate(accepted) != "2021-01-03" {
		t.Fatalf("accepted row=%s err=%v", formatDate(accepted), err)
	}
	for _, value := range []string{"2021 00 0", "2021 53 6"} {
		date, err := datefmt.Parse("%Y %U %w", value)
		if err == nil {
			t.Fatalf("rejected row %q returned %s", value, formatDate(date))
		}
	}
}

// TestParseIsIndependentOfCallOrder proves a failed parse does not poison the
// next call, so a batch importer can report every bad row.
func TestParseIsIndependentOfCallOrder(t *testing.T) {
	for i := 0; i < 3; i++ {
		if _, err := datefmt.Parse("%G-W%V-%u", "2021-W53-1"); err == nil {
			t.Fatal("invalid ISO week accepted")
		}
		date, err := datefmt.Parse("%G-W%V-%u", "2020-W01-1")
		if err != nil || formatDate(date) != "2019-12-30" {
			t.Fatalf("valid ISO week after a failure = %s err=%v", formatDate(date), err)
		}
	}
}

// TestWeekNumberHelpersMatchTheReferenceFormulas keeps the %U/%W/%w writers
// aligned with the pinned Python reference for a full year boundary.
func TestWeekNumberHelpersMatchTheReferenceFormulas(t *testing.T) {
	for _, test := range []struct {
		date             string
		sundayWeek       int
		sundayWeekday    int
		mondayWeek       int
		mondayWeekday    int
		isoWeekday       int
		isoWeek, isoYear int
	}{
		{"2024-01-01", 0, 1, 1, 1, 1, 1, 2024},
		{"2024-01-07", 1, 0, 1, 0, 7, 1, 2024},
		{"2024-01-08", 1, 1, 2, 1, 1, 2, 2024},
		{"2024-12-31", 52, 2, 53, 2, 2, 1, 2025},
		{"2021-01-01", 0, 5, 0, 5, 5, 53, 2020},
		{"2023-12-31", 53, 0, 52, 0, 7, 52, 2023},
	} {
		date, err := datefmt.Parse("%Y-%m-%d", test.date)
		if err != nil {
			t.Fatalf("Parse(%s) err=%v", test.date, err)
		}
		parsed, err := parseISO(test.date)
		if err != nil {
			t.Fatal(err)
		}
		sundayWeek, sundayWeekday := datefmt.SundayWeek(parsed)
		if sundayWeek != test.sundayWeek || sundayWeekday != test.sundayWeekday {
			t.Errorf("%s SundayWeek=%d/%d want %d/%d", test.date, sundayWeek, sundayWeekday, test.sundayWeek, test.sundayWeekday)
		}
		mondayWeek, mondayWeekday := datefmt.MondayWeek(parsed)
		if mondayWeek != test.mondayWeek || mondayWeekday != test.mondayWeekday {
			t.Errorf("%s MondayWeek=%d/%d want %d/%d", test.date, mondayWeek, mondayWeekday, test.mondayWeek, test.mondayWeekday)
		}
		if got := datefmt.ISOWeekday(parsed); got != test.isoWeekday {
			t.Errorf("%s ISOWeekday=%d want %d", test.date, got, test.isoWeekday)
		}
		isoYear, isoWeek := parsed.ISOWeek()
		if isoWeek != test.isoWeek || isoYear != test.isoYear {
			t.Errorf("%s ISO=%d/%d want %d/%d", test.date, isoWeek, isoYear, test.isoWeek, test.isoYear)
		}
		if date.Year != parsed.Year() {
			t.Errorf("%s Parse year=%d want %d", test.date, date.Year, parsed.Year())
		}
	}
}

// TestParseNeverReturnsADateWithAnError is the contract guard a CSV importer
// depends on: the date and the error are mutually exclusive, so a caller that
// checks only err can never commit a partially filled entry.
func TestParseNeverReturnsADateWithAnError(t *testing.T) {
	for _, test := range weekCases {
		date, err := datefmt.Parse(test.format, test.value)
		if err != nil && (date.Year != 0 || date.Month != 0 || date.Day != 0 || date.Raw != "") {
			t.Fatalf("%s: error %v came with date %+v", test.name, err, date)
		}
		if err == nil && date.Raw != test.value {
			t.Fatalf("%s: parsed date lost its source text: %+v", test.name, date)
		}
	}
}

// formatDate renders the numeric fields so an assertion compares the parsed
// calendar date, not the original source text.
func formatDate(date ledger.Date) string {
	return fmt.Sprintf("%04d-%02d-%02d", date.Year, date.Month, date.Day)
}

// parseISO is a test-only helper that builds the reference date so the week
// number assertions do not depend on the package under test.
func parseISO(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}
