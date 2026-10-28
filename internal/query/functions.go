// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package query

import (
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"orangecount/internal/ledger"
)

// queryContext carries the evaluated ledger so scalar functions can read
// account lifecycle, commodity metadata, and the price map. It is passed to the
// typed compiler once per statement.
type queryContext struct {
	evaluation ledger.Evaluation
}

// These helpers implement the scalar-function surface of the pinned BeanQuery
// release. Each returns the exact Go value the typed encoder expects, so the
// inventory comparison observes both type and value.

func (context queryContext) accountState(name string) (ledger.AccountState, bool) {
	state, ok := context.evaluation.Accounts[name]
	return state, ok
}

// openDirective finds the first Open directive for the account. Upstream keeps
// the same first-wins rule in get_account_open_close.
func (context queryContext) openDirective(name string) (ledger.EntryRecord, bool) {
	for _, entry := range context.evaluation.Entries {
		open, ok := entry.Directive.(ledger.Open)
		if ok && strings.EqualFold(open.Account, name) {
			return entry, true
		}
	}
	return ledger.EntryRecord{}, false
}

func (context queryContext) commodityDirective(currency string) (ledger.EntryRecord, bool) {
	for _, entry := range context.evaluation.Entries {
		commodity, ok := entry.Directive.(ledger.Commodity)
		if ok && commodity.Currency == currency {
			return entry, true
		}
	}
	return ledger.EntryRecord{}, false
}

// price looks up the latest quote at or before date. An empty date means the
// latest available quote, matching prices.get_price upstream.
func (context queryContext) price(base, quote string, date *ledger.Date) (ledger.Decimal, bool) {
	quotes := context.evaluation.Prices[base]
	var best *ledger.PriceQuote
	for i := range quotes {
		candidate := quotes[i]
		if !strings.EqualFold(candidate.Currency, quote) {
			continue
		}
		if date != nil && candidate.Date.Raw > date.Raw {
			continue
		}
		if best == nil || candidate.Date.Raw >= best.Date.Raw {
			chosen := candidate
			best = &chosen
		}
	}
	if best == nil {
		return ledger.Zero(), false
	}
	return best.Amount, true
}

var accountTypeOrder = []string{"Assets", "Liabilities", "Equity", "Income", "Expenses"}

// accountSortKey reproduces get_account_sort_key: the configured type index
// followed by the account name, joined by a dash. The index comes from the
// account's classified type, so a ledger that renames its roots through the
// name_* options sorts exactly as the default-root ledger it mirrors.
func accountSortKey(types ledger.AccountTypes, name string) string {
	index := len(accountTypeOrder)
	if kind := types.TypeOf(name); kind != "" {
		for i, key := range accountTypeOrder {
			if key == kind {
				index = i
				break
			}
		}
	}
	return fmt.Sprintf("%d-%s", index, name)
}

// accountSign is +1 for assets and expenses, -1 for the other three types.
func accountSign(types ledger.AccountTypes, name string) int {
	switch types.TypeOf(name) {
	case "Assets", "Expenses":
		return 1
	default:
		return -1
	}
}

func accountRoot(name string, depth int) string {
	components := strings.Split(name, ":")
	if depth < 1 {
		depth = 1
	}
	if depth > len(components) {
		depth = len(components)
	}
	return strings.Join(components[:depth], ":")
}

func accountParent(name string) string {
	at := strings.LastIndexByte(name, ':')
	if at < 0 {
		return ""
	}
	return name[:at]
}

func accountLeaf(name string) string {
	at := strings.LastIndexByte(name, ':')
	if at < 0 {
		return name
	}
	return name[at+1:]
}

var queryPatternCache = map[string]*regexp.Regexp{}

// boundedPattern compiles and caches a pattern. Invalid patterns are an error,
// never a silent no-match.
func boundedPattern(pattern string) (*regexp.Regexp, error) {
	if compiled, ok := queryPatternCache[pattern]; ok {
		return compiled, nil
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression %q", pattern)
	}
	queryPatternCache[pattern] = compiled
	return compiled, nil
}

// shorten reproduces textwrap.shorten: collapse whitespace, then cut at a word
// boundary and append an ellipsis so the result fits the width.
func shorten(text string, width int) (string, error) {
	if width <= 0 {
		return "", fmt.Errorf("placeholder too large for max width")
	}
	words := strings.Fields(text)
	joined := strings.Join(words, " ")
	if utf8.RuneCountInString(joined) <= width {
		return joined, nil
	}
	// The ellipsis counts toward the width; drop whole words from the end.
	ellipsis := "..."
	limit := width - len(ellipsis)
	if limit <= 0 {
		return "", fmt.Errorf("placeholder too large for max width")
	}
	current := ""
	for _, word := range words {
		next := word
		if current != "" {
			next = current + " " + word
		}
		if utf8.RuneCountInString(next) > limit {
			break
		}
		current = next
	}
	if current == "" {
		return "", nil
	}
	return current + ellipsis, nil
}

func substring(text string, start, end int) string {
	runes := []rune(text)
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start > end {
		return ""
	}
	return string(runes[start:end])
}

func splitComponent(text, delimiter string, index int) (string, error) {
	parts := strings.Split(text, delimiter)
	if index < 0 || index >= len(parts) {
		return "", fmt.Errorf("list index out of range")
	}
	return parts[index], nil
}

func joinStrings(values []string) string {
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}

func findFirst(pattern string, values []string) (any, error) {
	if len(values) == 0 {
		return nil, nil
	}
	compiled, err := boundedPattern(pattern)
	if err != nil {
		return nil, err
	}
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	for _, value := range sorted {
		if compiled.MatchString(value) && compiled.FindString(value) == value {
			return value, nil
		}
	}
	return nil, nil
}

func yearMonth(text string) (string, error) {
	parsed, err := time.Parse(isoDateLayout, text)
	if err != nil {
		return "", fmt.Errorf("invalid date")
	}
	return fmt.Sprintf("%04d-%02d-01", parsed.Year(), int(parsed.Month())), nil
}

func quarterOf(text string) (string, error) {
	parsed, err := time.Parse(isoDateLayout, text)
	if err != nil {
		return "", fmt.Errorf("invalid date")
	}
	return fmt.Sprintf("%04d-Q%d", parsed.Year(), (int(parsed.Month())-1)/3+1), nil
}

func weekdayOf(text string) (string, error) {
	parsed, err := time.Parse(isoDateLayout, text)
	if err != nil {
		return "", fmt.Errorf("invalid date")
	}
	return parsed.Format("Mon"), nil
}

func dateAdd(text string, days int) (string, error) {
	parsed, err := time.Parse(isoDateLayout, text)
	if err != nil {
		return "", fmt.Errorf("invalid date")
	}
	return parsed.AddDate(0, 0, days).Format(isoDateLayout), nil
}

func dateDifference(left, right string) (ledger.Decimal, error) {
	leftDate, leftErr := time.Parse(isoDateLayout, left)
	rightDate, rightErr := time.Parse(isoDateLayout, right)
	if leftErr != nil || rightErr != nil {
		return ledger.Zero(), fmt.Errorf("invalid date")
	}
	return ledger.NewDecimal(new(big.Rat).SetInt64(int64(leftDate.Sub(rightDate).Hours() / 24))), nil
}

func datePart(part, text string) (ledger.Decimal, error) {
	parsed, err := time.Parse(isoDateLayout, text)
	if err != nil {
		return ledger.Zero(), fmt.Errorf("invalid date")
	}
	var value int
	switch strings.ToLower(part) {
	case "year":
		value = parsed.Year()
	case "month":
		value = int(parsed.Month())
	case "day":
		value = parsed.Day()
	case "quarter":
		value = (int(parsed.Month())-1)/3 + 1
	default:
		return ledger.Zero(), fmt.Errorf("unsupported date part %q", part)
	}
	return ledger.NewDecimal(new(big.Rat).SetInt64(int64(value))), nil
}

func dateTrunc(part, text string) (string, error) {
	parsed, err := time.Parse(isoDateLayout, text)
	if err != nil {
		return "", fmt.Errorf("invalid date")
	}
	switch strings.ToLower(part) {
	case "year":
		return fmt.Sprintf("%04d-01-01", parsed.Year()), nil
	case "quarter":
		month := (int(parsed.Month())-1)/3*3 + 1
		return fmt.Sprintf("%04d-%02d-01", parsed.Year(), month), nil
	case "month":
		return fmt.Sprintf("%04d-%02d-01", parsed.Year(), int(parsed.Month())), nil
	case "day":
		return parsed.Format(isoDateLayout), nil
	default:
		return "", fmt.Errorf("unsupported date part %q", part)
	}
}

// dateBin aligns a date to the boundary of an interval relative to an origin.
func dateBin(interval Interval, text, origin string) (string, error) {
	parsed, err := time.Parse(isoDateLayout, text)
	if err != nil {
		return "", fmt.Errorf("invalid date")
	}
	originDate, err := time.Parse(isoDateLayout, origin)
	if err != nil {
		return "", fmt.Errorf("invalid date")
	}
	if interval.Years == 0 && interval.Months == 0 {
		if interval.Days <= 0 {
			return "", fmt.Errorf("interval must be positive")
		}
		days := int(parsed.Sub(originDate).Hours() / 24)
		steps := days / interval.Days
		if days < 0 && days%interval.Days != 0 {
			steps--
		}
		return originDate.AddDate(0, 0, steps*interval.Days).Format(isoDateLayout), nil
	}
	months := (parsed.Year()-originDate.Year())*12 + int(parsed.Month()) - int(originDate.Month())
	span := interval.Years*12 + interval.Months
	if span <= 0 {
		return "", fmt.Errorf("interval must be positive")
	}
	steps := months / span
	if months < 0 && months%span != 0 {
		steps--
	}
	return originDate.AddDate(0, steps*span, 0).Format(isoDateLayout), nil
}

func roundDecimal(value ledger.Decimal, digits int) ledger.Decimal {
	// Rounding uses the query context's default 28-significant-digit precision.
	text := value.Rat().FloatString(digits)
	if digits < 0 {
		text = value.Rat().FloatString(0)
	}
	rounded, err := ledger.ParseDecimal(text)
	if err != nil {
		return value
	}
	return rounded
}
