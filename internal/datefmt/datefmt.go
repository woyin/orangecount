// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package datefmt parses one date string against a strftime-style layout.
// It exists so a CSV import can declare the bank's date format, including the
// ISO (%G/%V/%u) and Sunday/Monday (%U/%W/%w) week forms, without the
// permissive field normalisation Python's strptime applies.
//
// The contract is deliberately strict: a value is accepted only when every
// supplied field identifies the same real calendar date. A missing field, a
// field that conflicts with another field, or a week number that cannot be
// written back into its declared year is an error. Failure never returns a
// date and never returns a partially filled result, so a caller cannot commit
// a silently rolled entry.
package datefmt

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"orangecount/internal/ledger"
)

// Code names a failure class so an importer can map it to a diagnostic without
// matching on message text.
type Code string

const (
	// CodeSyntax means the layout itself is unusable: an unknown directive, a
	// repeated field, or an unsupported combination of directives.
	CodeSyntax Code = "date-format-syntax"
	// CodeMismatch means the value does not match the layout.
	CodeMismatch Code = "date-value-mismatch"
	// CodeRange means a matched number is outside the range its directive
	// allows (for example ISO week 53 in a 52-week ISO year).
	CodeRange Code = "date-value-out-of-range"
	// CodeConflict means two directive groups identify different dates.
	CodeConflict Code = "date-value-conflict"
)

// Error reports a parse failure with its class. Value and Format are echoed so
// an import diagnostic can point at the offending cell without re-parsing.
type Error struct {
	Code    Code
	Message string
	Format  string
	Value   string
}

func (e Error) Error() string { return e.Message }

func failure(code Code, format, value, message string) error {
	return Error{Code: code, Message: message, Format: format, Value: value}
}

// ISOWeekday returns the ISO weekday number (Monday 1 .. Sunday 7) for a
// calendar date.
func ISOWeekday(date time.Time) int {
	return int(date.Weekday()+6)%7 + 1
}

// SundayWeek returns the %U week number (00-53, weeks start on Sunday) and the
// %w weekday number (Sunday 0 .. Saturday 6) for a calendar date.
func SundayWeek(date time.Time) (week, weekday int) {
	day := date.YearDay() - 1
	weekday = int(date.Weekday())
	return (day + 7 - weekday) / 7, weekday
}

// MondayWeek returns the %W week number (00-53, weeks start on Monday) and the
// %w weekday number (Sunday 0 .. Saturday 6) for a calendar date.
func MondayWeek(date time.Time) (week, weekday int) {
	day := date.YearDay() - 1
	monday := (int(date.Weekday()) + 6) % 7
	return (day + 7 - monday) / 7, int(date.Weekday())
}

// isoWeeksInYear reports whether an ISO year has 52 or 53 ISO weeks. A year
// has 53 ISO weeks when its 1 January is a Thursday, or when it is a leap year
// whose 1 January is a Wednesday.
func isoWeeksInYear(year int) int {
	jan1 := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	switch jan1.Weekday() {
	case time.Thursday:
		return 53
	case time.Wednesday:
		if isLeap(year) {
			return 53
		}
	}
	return 52
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// fields collects the numbers a layout and value supplied.
type fields struct {
	calendarYear  int
	hasCalYear    bool
	isoYear       int
	hasISOYear    bool
	month         int
	hasMonth      bool
	day           int
	hasDay        bool
	isoWeek       int
	hasISOWeek    bool
	weekU         int
	hasWeekU      bool
	weekW         int
	hasWeekW      bool
	weekday       int
	hasWeekday    bool
	isoWeekday    int
	hasISOWeekday bool
}

type token struct {
	literal   string
	directive byte
	hasDir    bool
}

// Parse reads value according to a strftime-style format and returns the
// calendar date every supplied field agrees on.
func Parse(format, value string) (ledger.Date, error) {
	tokens, err := tokenize(format, value)
	if err != nil {
		return ledger.Date{}, err
	}
	var parsed fields
	offset := 0
	for _, item := range tokens {
		if !item.hasDir {
			if !strings.HasPrefix(value[offset:], item.literal) {
				return ledger.Date{}, failure(CodeMismatch, format, value,
					fmt.Sprintf("time data %q does not match format %q", value, format))
			}
			offset += len(item.literal)
			continue
		}
		width := directiveWidth(item.directive)
		text, ok := takeDigits(value[offset:], width)
		if !ok {
			return ledger.Date{}, failure(CodeMismatch, format, value,
				fmt.Sprintf("time data %q does not match format %q", value, format))
		}
		offset += len(text)
		number, convErr := strconv.Atoi(text)
		if convErr != nil {
			return ledger.Date{}, failure(CodeMismatch, format, value,
				fmt.Sprintf("time data %q does not match format %q", value, format))
		}
		if err := parsed.assign(item.directive, number, format, value); err != nil {
			return ledger.Date{}, err
		}
	}
	if offset != len(value) {
		return ledger.Date{}, failure(CodeMismatch, format, value,
			fmt.Sprintf("time data %q does not match format %q", value, format))
	}
	return parsed.resolve(format, value)
}

// assign stores one directive's number, rejecting a repeated field and an
// out-of-range value before any date is computed.
func (f *fields) assign(directive byte, number int, format, value string) error {
	duplicate := func() error {
		return failure(CodeSyntax, format, value,
			fmt.Sprintf("date format %q supplies the %%%c field more than once", format, directive))
	}
	switch directive {
	case 'Y':
		if f.hasCalYear {
			return duplicate()
		}
		f.calendarYear, f.hasCalYear = number, true
	case 'y':
		if f.hasCalYear {
			return duplicate()
		}
		if number < 69 {
			number += 2000
		} else {
			number += 1900
		}
		f.calendarYear, f.hasCalYear = number, true
	case 'G':
		if f.hasISOYear {
			return duplicate()
		}
		f.isoYear, f.hasISOYear = number, true
	case 'm':
		if f.hasMonth {
			return duplicate()
		}
		if number < 1 || number > 12 {
			return failure(CodeRange, format, value, fmt.Sprintf("month must be in 1..12, got %d", number))
		}
		f.month, f.hasMonth = number, true
	case 'd':
		if f.hasDay {
			return duplicate()
		}
		if number < 1 || number > 31 {
			return failure(CodeRange, format, value, fmt.Sprintf("day must be in 1..31, got %d", number))
		}
		f.day, f.hasDay = number, true
	case 'V':
		if f.hasISOWeek {
			return duplicate()
		}
		if number < 1 || number > 53 {
			// Python rejects week 00 as a value mismatch (the %V field is
			// one-origin), so keep that class rather than a range error.
			return failure(CodeMismatch, format, value,
				fmt.Sprintf("time data %q does not match format %q", value, format))
		}
		f.isoWeek, f.hasISOWeek = number, true
	case 'U':
		if f.hasWeekU {
			return duplicate()
		}
		if number < 0 || number > 53 {
			return failure(CodeRange, format, value, fmt.Sprintf("week must be in 0..53, got %d", number))
		}
		f.weekU, f.hasWeekU = number, true
	case 'W':
		if f.hasWeekW {
			return duplicate()
		}
		if number < 0 || number > 53 {
			return failure(CodeRange, format, value, fmt.Sprintf("week must be in 0..53, got %d", number))
		}
		f.weekW, f.hasWeekW = number, true
	case 'w':
		if f.hasWeekday {
			return duplicate()
		}
		if number < 0 || number > 6 {
			return failure(CodeRange, format, value, fmt.Sprintf("weekday must be in 0..6, got %d", number))
		}
		f.weekday, f.hasWeekday = number, true
	case 'u':
		if f.hasISOWeekday {
			return duplicate()
		}
		if number < 1 || number > 7 {
			return failure(CodeRange, format, value, fmt.Sprintf("ISO weekday must be in 1..7, got %d", number))
		}
		f.isoWeekday, f.hasISOWeekday = number, true
	}
	return nil
}

// resolve turns the collected fields into exactly one date, or fails when the
// fields are incomplete, conflicting, or cannot be written back into the year
// and week they declare.
func (f fields) resolve(format, value string) (ledger.Date, error) {
	hasWeek := f.hasWeekU || f.hasWeekW
	hasCalendarDate := f.hasMonth || f.hasDay
	hasISODate := f.hasISOWeek || f.hasISOYear

	if hasISODate {
		// Without an ISO year, %V would have to borrow the calendar year.
		// Python refuses that combination instead of guessing a year.
		if f.hasISOWeek && !f.hasISOYear && f.hasCalYear {
			return ledger.Date{}, failure(CodeSyntax, format, value,
				"ISO week directive '%V' is incompatible with the year directive '%Y'. Use the ISO year '%G' instead.")
		}
		if !f.hasISOYear || !f.hasISOWeek || !f.hasWeekday && !f.hasISOWeekday {
			return ledger.Date{}, failure(CodeSyntax, format, value,
				"ISO year directive '%G' must be used with the ISO week directive '%V' and a weekday directive ('%A', '%a', '%w', or '%u').")
		}
		if f.hasWeekU || f.hasWeekW {
			return ledger.Date{}, failure(CodeSyntax, format, value,
				"ISO week directive '%V' cannot be combined with the %U/%W week directives")
		}
	}
	if hasWeek && (!f.hasCalYear || !f.hasWeekday) {
		return ledger.Date{}, failure(CodeSyntax, format, value,
			"CSV week date: %U/%W require an explicit calendar year (%Y or %y) and a weekday; refusing Python's defaulting or ignored week fields")
	}
	if hasCalendarDate && (!f.hasCalYear || !f.hasMonth || !f.hasDay) {
		return ledger.Date{}, failure(CodeSyntax, format, value,
			"CSV calendar date requires a year (%Y or %y), a month (%m) and a day (%d)")
	}
	if !hasWeek && !hasCalendarDate && !hasISODate {
		return ledger.Date{}, failure(CodeSyntax, format, value,
			fmt.Sprintf("date format %q does not name a calendar date", format))
	}

	candidates := make([]time.Time, 0, 3)
	if f.hasCalYear && f.hasMonth && f.hasDay {
		date, err := buildDate(f.calendarYear, f.month, f.day, format, value)
		if err != nil {
			return ledger.Date{}, err
		}
		candidates = append(candidates, date)
	}
	if hasISODate {
		date, err := f.isoWeekDate(format, value)
		if err != nil {
			return ledger.Date{}, err
		}
		candidates = append(candidates, date)
	}
	if hasWeek {
		date, err := f.weekDate(format, value)
		if err != nil {
			return ledger.Date{}, err
		}
		candidates = append(candidates, date)
	}
	date := candidates[0]
	for _, candidate := range candidates[1:] {
		if !candidate.Equal(date) {
			return ledger.Date{}, failure(CodeConflict, format, value,
				"CSV week date: supplied fields identify different dates; refusing Python's precedence or overwrite rules")
		}
	}
	return ledger.Date{Year: date.Year(), Month: int(date.Month()), Day: date.Day(), Raw: value}, nil
}

func (f fields) isoWeekDate(format, value string) (time.Time, error) {
	weekday := f.isoWeekday
	if !f.hasISOWeekday {
		// A %w weekday is Sunday 0 .. Saturday 6; ISO numbers Monday 1 .. Sunday 7.
		weekday = f.weekday
		if weekday == 0 {
			weekday = 7
		}
	}
	if f.isoWeek > isoWeeksInYear(f.isoYear) {
		return time.Time{}, failure(CodeRange, format, value, fmt.Sprintf("Invalid week: %d", f.isoWeek))
	}
	// ISO week 1 is the week containing 4 January.
	jan4 := time.Date(f.isoYear, time.January, 4, 0, 0, 0, 0, time.UTC)
	monday := jan4.AddDate(0, 0, -(ISOWeekday(jan4) - 1))
	date := monday.AddDate(0, 0, (f.isoWeek-1)*7+(weekday-1))
	// Re-derive the declared ISO year and week: a value that rolls into
	// another year is a real error, not a normalisation opportunity.
	isoYear, isoWeek := date.ISOWeek()
	if isoYear != f.isoYear || isoWeek != f.isoWeek || ISOWeekday(date) != weekday {
		return time.Time{}, failure(CodeRange, format, value,
			fmt.Sprintf("Invalid week: %d", f.isoWeek))
	}
	return date, nil
}

func (f fields) weekDate(format, value string) (time.Time, error) {
	sundayFirst := f.hasWeekU
	week := f.weekU
	if !sundayFirst {
		week = f.weekW
	}
	date, err := buildWeekDate(f.calendarYear, week, f.weekday, sundayFirst, format, value)
	if err != nil {
		return time.Time{}, err
	}
	// Python normalises a week field that cannot exist in the declared year
	// (week 00 of a year whose 1 January is already in week 01, or a week 53
	// that only exists in a neighbouring year). Refuse that normalisation so a
	// bank export cannot silently move a transaction to another year.
	gotWeek, gotWeekday := SundayWeek(date)
	if !sundayFirst {
		gotWeek, gotWeekday = MondayWeek(date)
	}
	if date.Year() != f.calendarYear || gotWeek != week || gotWeekday != f.weekday {
		return time.Time{}, failure(CodeRange, format, value,
			"CSV week date: year/week/weekday does not round-trip in the declared calendar year; refusing Python's permissive week-field normalization")
	}
	return date, nil
}

// buildWeekDate resolves a (%U or %W) week plus a %w weekday inside a declared
// calendar year. Week 0 is the partial week before the first Sunday (%U) or
// Monday (%W); week 1 starts on that first Sunday or Monday. The %w weekday is
// Sunday 0 .. Saturday 6, so a Monday-first week maps it to an offset of
// (weekday + 6) % 7.
func buildWeekDate(year, week, weekday int, sundayFirst bool, format, value string) (time.Time, error) {
	jan1 := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	offsetToFirst := 0
	dayOffset := weekday
	if sundayFirst {
		offsetToFirst = (7 - int(jan1.Weekday())) % 7
	} else {
		offsetToFirst = (8 - int(jan1.Weekday())) % 7
		dayOffset = (weekday + 6) % 7
	}
	firstWeekStart := jan1.AddDate(0, 0, offsetToFirst)
	if week == 0 {
		// Only days before the first week start belong to week 0.
		date := firstWeekStart.AddDate(0, 0, dayOffset-7)
		if !date.Before(firstWeekStart) {
			return time.Time{}, failure(CodeRange, format, value,
				"CSV week date: year/week/weekday does not round-trip in the declared calendar year; refusing Python's permissive week-field normalization")
		}
		return date, nil
	}
	return firstWeekStart.AddDate(0, 0, (week-1)*7+dayOffset), nil
}

func buildDate(year, month, day int, format, value string) (time.Time, error) {
	if year < 1 {
		return time.Time{}, failure(CodeRange, format, value, fmt.Sprintf("year must be positive, got %d", year))
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return time.Time{}, failure(CodeRange, format, value,
			fmt.Sprintf("day %d is out of range for %04d-%02d", day, year, month))
	}
	return date, nil
}

// directiveWidth reports how many characters a directive consumes. A %w
// weekday is one digit; every other supported directive is two digits wide
// except %Y and %G, which are four.
func directiveWidth(directive byte) int {
	switch directive {
	case 'Y', 'G':
		return 4
	case 'w', 'u':
		return 1
	default:
		return 2
	}
}

func takeDigits(text string, width int) (string, bool) {
	if len(text) < width {
		return "", false
	}
	for _, char := range text[:width] {
		if char < '0' || char > '9' {
			return "", false
		}
	}
	return text[:width], true
}

// tokenize splits the layout into literal runs and directives and rejects a
// directive the package does not implement, so an unsupported layout fails
// before any value is read.
func tokenize(format, value string) ([]token, error) {
	tokens := make([]token, 0, len(format))
	var literal strings.Builder
	flush := func() {
		if literal.Len() != 0 {
			tokens = append(tokens, token{literal: literal.String()})
			literal.Reset()
		}
	}
	for i := 0; i < len(format); i++ {
		char := format[i]
		if char != '%' {
			literal.WriteByte(char)
			continue
		}
		if i+1 >= len(format) {
			return nil, failure(CodeSyntax, format, value, "date format ends with a bare '%'")
		}
		i++
		directive := format[i]
		if directive == '%' {
			literal.WriteByte('%')
			continue
		}
		if !supportedDirective(directive) {
			return nil, failure(CodeSyntax, format, value,
				fmt.Sprintf("unsupported CSV DateFormat directive '%%%c'", directive))
		}
		flush()
		tokens = append(tokens, token{directive: directive, hasDir: true})
	}
	flush()
	return tokens, nil
}

func supportedDirective(directive byte) bool {
	switch directive {
	case 'Y', 'y', 'G', 'm', 'd', 'V', 'U', 'W', 'w', 'u':
		return true
	default:
		return false
	}
}
