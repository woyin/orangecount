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

// Valid reports whether the numeric fields form a real calendar date.
func (d Date) Valid() bool {
	return d.Year > 0 && d.Month >= 1 && d.Month <= 12 && d.Day >= 1 && d.Day <= daysInMonth(d.Year, d.Month)
}

// String returns the original source text of the date, keeping output lossless.
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

// DirectiveKind names the concrete directive a syntax node represents.
type DirectiveKind string

// The Kind* constants enumerate every directive the parser can produce.
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
	KindDialect   DirectiveKind = "dialect"
)

// Directive is implemented by every core directive. Span and Raw are kept on
// the interface so callers can provide source navigation without type switches.
type Directive interface {
	Kind() DirectiveKind
	Span() source.Span
	RawText() string
}

// DirectiveBase carries the span, raw text, and trailing metadata shared by
// every directive; embedding it satisfies the Directive interface.
type DirectiveBase struct {
	At   source.Span
	Raw  string
	Meta []Metadata
}

// Span implements Directive with the embedded source span.
func (d DirectiveBase) Span() source.Span { return d.At }

// RawText implements Directive with the verbatim source line.
func (d DirectiveBase) RawText() string { return d.Raw }

// Metadata is one key/value pair attached to a directive or posting.
type Metadata struct {
	Key   string
	Value Value
	Span  source.Span
}

// Option is an `option` directive: key/value configuration for the evaluator
// (for example tolerance or operating_currency).
type Option struct {
	DirectiveBase
	Key, Value string
}

// Kind implements Directive.
func (Option) Kind() DirectiveKind { return KindOption }

// Plugin is a `plugin` directive naming an extension module and its config.
type Plugin struct {
	DirectiveBase
	Module string
	Config string
}

// Kind implements Directive.
func (Plugin) Kind() DirectiveKind { return KindPlugin }

// Include is an `include` directive referencing another ledger file.
type Include struct {
	DirectiveBase
	Path string
}

// Kind implements Directive.
func (Include) Kind() DirectiveKind { return KindInclude }

// TagDirective is a `pushtag`/`poptag` pair controlling the ambient tag stack.
type TagDirective struct {
	DirectiveBase
	Tag string
}

// Kind implements Directive.
func (d TagDirective) Kind() DirectiveKind {
	fields := strings.Fields(d.Raw)
	if len(fields) > 0 && strings.EqualFold(fields[0], "poptag") {
		return KindPopTag
	}
	return KindPushTag
}

// Open is an `open` directive starting an account, optionally constraining
// its currencies and booking method.
type Open struct {
	DirectiveBase
	Date       Date
	Account    string
	Currencies []string
	Booking    string
}

// Kind implements Directive.
func (Open) Kind() DirectiveKind { return KindOpen }

// Close is a `close` directive ending an account.
type Close struct {
	DirectiveBase
	Date    Date
	Account string
}

// Kind implements Directive.
func (Close) Kind() DirectiveKind { return KindClose }

// Commodity is a `commodity` directive declaring a currency.
type Commodity struct {
	DirectiveBase
	Date     Date
	Currency string
}

// Kind implements Directive.
func (Commodity) Kind() DirectiveKind { return KindCommodity }

// Balance is a `balance` directive asserting an account amount at a date,
// with an optional explicit tolerance.
type Balance struct {
	DirectiveBase
	Date      Date
	Account   string
	Amount    Amount
	Tolerance *Number
}

// Kind implements Directive.
func (Balance) Kind() DirectiveKind { return KindBalance }

// Pad is a `pad` directive: balance an account by importing from another.
type Pad struct {
	DirectiveBase
	Date          Date
	Account       string
	SourceAccount string
}

// Kind implements Directive.
func (Pad) Kind() DirectiveKind { return KindPad }

// Event is an `event` directive recording a dated key/value fact.
type Event struct {
	DirectiveBase
	Date  Date
	Type  string
	Value string
}

// Kind implements Directive.
func (Event) Kind() DirectiveKind { return KindEvent }

// Query is a `query` directive storing a named BeanQuery for the UI.
type Query struct {
	DirectiveBase
	Date  Date
	Name  string
	Query string
}

// Kind implements Directive.
func (Query) Kind() DirectiveKind { return KindQuery }

// Price is a `price` directive declaring a commodity exchange rate at a date.
type Price struct {
	DirectiveBase
	Date     Date
	Currency string
	Amount   Amount
}

// Kind implements Directive.
func (Price) Kind() DirectiveKind { return KindPrice }

// Document is a `document` directive attaching files (and tags/links) to an
// account at a date.
type Document struct {
	DirectiveBase
	Date      Date
	Account   string
	Filenames []string
	Tags      []string
	Links     []string
}

// Kind implements Directive.
func (Document) Kind() DirectiveKind { return KindDocument }

// Note is a `note` directive attaching a comment to an account.
type Note struct {
	DirectiveBase
	Date    Date
	Account string
	Comment string
}

// Kind implements Directive.
func (Note) Kind() DirectiveKind { return KindNote }

// Custom is a `custom` directive with a type name and arbitrary values.
type Custom struct {
	DirectiveBase
	Date   Date
	Type   string
	Values []Value
}

// Kind implements Directive.
func (Custom) Kind() DirectiveKind { return KindCustom }

// Transaction is a dated transaction with flag, payee/narration, tags/links,
// metadata, and postings.
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

// Kind implements Directive.
func (Transaction) Kind() DirectiveKind { return KindTxn }

// Dialect is one OrangeCount dialect shorthand line (ADR-0045): a terse
// two-posting transaction the dialect pass replaces with a Transaction
// before evaluation. It is source-only; the evaluator never consumes it.
// When HasDate is false the parser resolved the date by block anchoring
// (Anchored) and Date holds the anchor value; a missing anchor is diagnosed
// at parse time and leaves Date zero.
type Dialect struct {
	DirectiveBase
	Date         Date
	HasDate      bool
	Anchored     bool
	Flag         string
	Amount       Number
	Currency     string
	SourceRef    string
	DestRef      string
	Payee        string
	Narration    string
	HasNarration bool
	Tags         []string
	Links        []string
	// Investment legs carry a securities quantity with a cost batch instead
	// of (or alongside) a plain cash amount.
	HasQuantity bool
	Quantity    Number
	Security    string
	Cost        *CostSpec
}

// Kind implements Directive.
func (Dialect) Kind() DirectiveKind { return KindDialect }

// Posting is one leg of a transaction: account, optional flag, units, cost
// and price specs, and metadata.
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

// Span returns the posting's source span for navigation and diagnostics.
func (p Posting) Span() source.Span { return p.At }

// RawText returns the posting's verbatim source text.
func (p Posting) RawText() string { return p.Raw }

// CostSpec is the `{...}` cost annotation on a posting, kept as raw
// components until the evaluator normalizes them.
type CostSpec struct {
	At         source.Span
	Raw        string
	Total      bool
	Components []Value
}

// PriceSpec is the `@`/`@@` price annotation on a posting; Total marks the
// `@@` whole-position spelling.
type PriceSpec struct {
	At     source.Span
	Total  bool
	Amount Amount
}

// Number is an exact decimal backed by a big.Rat plus the source text.
type Number struct {
	Raw string
	Rat *big.Rat
	At  source.Span
}

// String returns the original source spelling of the number.
func (n Number) String() string { return n.Raw }

// Amount pairs a Number with its currency.
type Amount struct {
	Number   Number
	Currency string
	At       source.Span
}

// ValueKind discriminates the Value union for metadata and custom values.
type ValueKind string

// The Value* constants enumerate every value shape the parser can produce in
// metadata, custom directives, and expression positions.
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

// Value is a tagged union covering every scalar/list/map the grammar accepts
// outside of postings.
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

// Comment is a standalone or trailing source comment.
type Comment struct {
	At   source.Span
	Text string
}

// File is one parsed source file: its directives in order and its comments.
type File struct {
	Source     *source.SourceFile
	Directives []Directive
	Comments   []Comment
}
