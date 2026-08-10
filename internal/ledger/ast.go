// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package ledger contains the lossless-enough Beancount v3 source AST and
// parser. Evaluation deliberately lives in later layers.

package ledger

import (
	"math/big"
	"strings"

	"orangecount/internal/source"
)

// Date is a calendar date in the ledger's proleptic Gregorian calendar.
// Keeping the original text allows diagnostics and source navigation to stay
// lossless while the numeric fields support later evaluation.
type Date struct {
	Year, Month, Day int
	Raw              string
	Span             source.Span
}

func (d Date) Valid() bool {
	return d.Year > 0 && d.Month >= 1 && d.Month <= 12 && d.Day >= 1 && d.Day <= daysInMonth(d.Year, d.Month)
}

func (d Date) String() string { return d.Raw }

func daysInMonth(year, month int) int {
	if month == 2 {
		leap := year%4 == 0 && (year%100 != 0 || year%400 == 0)
		if leap {
			return 29
		}
		return 28
	}
	if month == 4 || month == 6 || month == 9 || month == 11 {
		return 30
	}
	return 31
}

type DirectiveKind string

const (
	KindOption    DirectiveKind = "option"
	KindPlugin    DirectiveKind = "plugin"
	KindInclude   DirectiveKind = "include"
	KindPushTag   DirectiveKind = "pushtag"
	KindPopTag    DirectiveKind = "poptag"
	KindOpen      DirectiveKind = "open"
	KindClose     DirectiveKind = "close"
	KindCommodity DirectiveKind = "commodity"
	KindBalance   DirectiveKind = "balance"
	KindPad       DirectiveKind = "pad"
	KindEvent     DirectiveKind = "event"
	KindQuery     DirectiveKind = "query"
	KindPrice     DirectiveKind = "price"
	KindDocument  DirectiveKind = "document"
	KindNote      DirectiveKind = "note"
	KindCustom    DirectiveKind = "custom"
	KindTxn       DirectiveKind = "transaction"
)

// Directive is implemented by every core directive. Span and Raw are kept on
// the interface so callers can provide source navigation without type switches.
type Directive interface {
	Kind() DirectiveKind
	Span() source.Span
	RawText() string
}

type DirectiveBase struct {
	At   source.Span
	Raw  string
	Meta []Metadata
}

func (d DirectiveBase) Span() source.Span { return d.At }
func (d DirectiveBase) RawText() string   { return d.Raw }

type Metadata struct {
	Key   string
	Value Value
	Span  source.Span
}

type Option struct {
	DirectiveBase
	Key, Value string
}

func (Option) Kind() DirectiveKind { return KindOption }

type Plugin struct {
	DirectiveBase
	Module string
	Config string
}

func (Plugin) Kind() DirectiveKind { return KindPlugin }

type Include struct {
	DirectiveBase
	Path string
}

func (Include) Kind() DirectiveKind { return KindInclude }

type TagDirective struct {
	DirectiveBase
	Tag string
}

func (d TagDirective) Kind() DirectiveKind {
	fields := strings.Fields(d.Raw)
	if len(fields) > 0 && strings.EqualFold(fields[0], "poptag") {
		return KindPopTag
	}
	return KindPushTag
}

type Open struct {
	DirectiveBase
	Date       Date
	Account    string
	Currencies []string
	Booking    string
}

func (Open) Kind() DirectiveKind { return KindOpen }

type Close struct {
	DirectiveBase
	Date    Date
	Account string
}

func (Close) Kind() DirectiveKind { return KindClose }

type Commodity struct {
	DirectiveBase
	Date     Date
	Currency string
}

func (Commodity) Kind() DirectiveKind { return KindCommodity }

type Balance struct {
	DirectiveBase
	Date      Date
	Account   string
	Amount    Amount
	Tolerance *Number
}

func (Balance) Kind() DirectiveKind { return KindBalance }

type Pad struct {
	DirectiveBase
	Date          Date
	Account       string
	SourceAccount string
}

func (Pad) Kind() DirectiveKind { return KindPad }

type Event struct {
	DirectiveBase
	Date  Date
	Type  string
	Value string
}

func (Event) Kind() DirectiveKind { return KindEvent }

type Query struct {
	DirectiveBase
	Date  Date
	Name  string
	Query string
}

func (Query) Kind() DirectiveKind { return KindQuery }

type Price struct {
	DirectiveBase
	Date     Date
	Currency string
	Amount   Amount
}

func (Price) Kind() DirectiveKind { return KindPrice }

type Document struct {
	DirectiveBase
	Date      Date
	Account   string
	Filenames []string
	Tags      []string
	Links     []string
}

func (Document) Kind() DirectiveKind { return KindDocument }

type Note struct {
	DirectiveBase
	Date    Date
	Account string
	Comment string
}

func (Note) Kind() DirectiveKind { return KindNote }

type Custom struct {
	DirectiveBase
	Date   Date
	Type   string
	Values []Value
}

func (Custom) Kind() DirectiveKind { return KindCustom }

type Transaction struct {
	DirectiveBase
	Date      Date
	Flag      string
	Payee     string
	Narration string
	Tags      []string
	Links     []string
	Meta      []Metadata
	Postings  []Posting
}

func (Transaction) Kind() DirectiveKind { return KindTxn }

type Posting struct {
	At      source.Span
	Raw     string
	Account string
	Flag    string
	Units   *Amount
	Cost    *CostSpec
	Price   *PriceSpec
	Meta    []Metadata
	// averageRejected is evaluator-only booking state. It is never parsed from
	// or emitted to source/report consumers; applyPosting uses it to diagnose a
	// disallowed AVERAGE reduction without changing inventory or balances.
	averageRejected bool
}

func (p Posting) Span() source.Span { return p.At }
func (p Posting) RawText() string   { return p.Raw }

type CostSpec struct {
	At         source.Span
	Raw        string
	Total      bool
	Components []Value
}

type PriceSpec struct {
	At     source.Span
	Total  bool
	Amount Amount
}

type Number struct {
	Raw string
	Rat *big.Rat
	At  source.Span
}

func (n Number) String() string { return n.Raw }

type Amount struct {
	Number   Number
	Currency string
	At       source.Span
}

type ValueKind string

const (
	ValueInvalid  ValueKind = "invalid"
	ValueString   ValueKind = "string"
	ValueNumber   ValueKind = "number"
	ValueBool     ValueKind = "bool"
	ValueDate     ValueKind = "date"
	ValueAccount  ValueKind = "account"
	ValueCurrency ValueKind = "currency"
	ValueTag      ValueKind = "tag"
	ValueLink     ValueKind = "link"
	ValueAmount   ValueKind = "amount"
	ValueList     ValueKind = "list"
	ValueMap      ValueKind = "map"
	ValueNull     ValueKind = "null"
)

type Value struct {
	Kind   ValueKind
	Raw    string
	At     source.Span
	String string
	Number Number
	Bool   bool
	Date   Date
	Amount Amount
	List   []Value
	Map    []Metadata
}

type Comment struct {
	At   source.Span
	Text string
}

type File struct {
	Source     *source.SourceFile
	Directives []Directive
	Comments   []Comment
}
