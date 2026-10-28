// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Interval is the query language's relativedelta equivalent: a calendar-aware
// period with independent year, month, day, and clock components. It is never
// a database value; only date arithmetic consumes it.
type Interval struct {
	Years   int `json:"years"`
	Months  int `json:"months"`
	Days    int `json:"days"`
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
}

var intervalUnits = map[string]func(*Interval, int){
	"year":    func(i *Interval, n int) { i.Years += n },
	"years":   func(i *Interval, n int) { i.Years += n },
	"month":   func(i *Interval, n int) { i.Months += n },
	"months":  func(i *Interval, n int) { i.Months += n },
	"day":     func(i *Interval, n int) { i.Days += n },
	"days":    func(i *Interval, n int) { i.Days += n },
	"hour":    func(i *Interval, n int) { i.Hours += n },
	"hours":   func(i *Interval, n int) { i.Hours += n },
	"minute":  func(i *Interval, n int) { i.Minutes += n },
	"minutes": func(i *Interval, n int) { i.Minutes += n },
	"second":  func(i *Interval, n int) { i.Seconds += n },
	"seconds": func(i *Interval, n int) { i.Seconds += n },
}

// ParseInterval reads a relaxed duration such as "1 month" or
// "1 year 2 months". An unparseable period is an error, never a zero period.
func ParseInterval(text string) (Interval, error) {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(text)))
	if len(fields) == 0 {
		return Interval{}, fmt.Errorf("interval requires a period")
	}
	result := Interval{}
	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			return Interval{}, fmt.Errorf("interval requires a number followed by a unit")
		}
		number, err := strconv.Atoi(fields[i])
		if err != nil {
			return Interval{}, fmt.Errorf("interval requires an integer count")
		}
		apply, ok := intervalUnits[strings.TrimSuffix(fields[i+1], "s")]
		if !ok {
			apply, ok = intervalUnits[fields[i+1]]
		}
		if !ok {
			return Interval{}, fmt.Errorf("interval unit %q is unsupported", fields[i+1])
		}
		apply(&result, number)
	}
	return result, nil
}

func parseInterval(text string) (Interval, error) { return ParseInterval(text) }

// addInterval shifts a date by whole calendar units, retaining the day of
// month where the target month is long enough.
func addInterval(text string, interval Interval) string {
	parsed, err := time.Parse(isoDateLayout, text)
	if err != nil {
		return text
	}
	parsed = parsed.AddDate(interval.Years, interval.Months, interval.Days)
	parsed = parsed.Add(time.Duration(interval.Hours)*time.Hour + time.Duration(interval.Minutes)*time.Minute + time.Duration(interval.Seconds)*time.Second)
	return parsed.Format(isoDateLayout)
}
