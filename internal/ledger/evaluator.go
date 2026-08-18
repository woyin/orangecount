// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"orangecount/internal/diagnostic"
	"orangecount/internal/source"
)

// flagMerging marks the internal legs an AVERAGE booking generates to remove
// old lots and refill the merged lot, mirroring Beancount's FLAG_MERGING. It is
// diagnostic/display metadata only; booking logic is unaffected by flag.
const (
	flagMerging = "M"
)

// EvalOptions controls deterministic compatibility behavior. A zero tolerance
// requests exact balancing; when InferDecimalTolerance is true, transaction
// tolerances are inferred from the decimal places present in their postings.
type EvalOptions struct {
	DefaultTolerance      Decimal
	InferDecimalTolerance bool
}

// AccountState is the evaluated lifecycle and balance of one account.
type AccountState struct {
	Name       string
	Opened     Date
	Closed     *Date
	Currencies []string
	Booking    string
	Balances   map[string]Decimal
	Positions  []Position
}

// Cost is the normalized subset of a Beancount lot cost that the evaluator
// can use for FIFO inventory matching. Additional cost components remain in
// the source AST and are never silently discarded from source navigation.
type Cost struct {
	Number   Decimal
	Currency string
	Date     *Date
	Label    string
}

// Position tracks units and their optional lot cost.
type Position struct {
	Units    Decimal
	Currency string
	Cost     *Cost
	Span     source.Span
}

// PriceQuote is an exact price-map entry in source order.
type PriceQuote struct {
	Date     Date
	Base     string
	Amount   Decimal
	Currency string
	Span     source.Span
}

// EntryRecord retains evaluated source order for later reports and journal
// views without exposing mutable evaluator internals.
type EntryRecord struct {
	Date      Date
	Directive Directive
	Span      source.Span
	File      string
}

// Evaluation is the result of evaluating an include graph. A result may be
// partial when diagnostics contain errors; snapshot builders publish only
// error-free evaluations.
type Evaluation struct {
	Accounts    map[string]AccountState
	Prices      map[string][]PriceQuote
	Entries     []EntryRecord
	Options     map[string]string
	Diagnostics []diagnostic.Diagnostic
	Valid       bool
}

// Evaluate evaluates parsed files in include traversal order. It is pure with
// respect to source files and creates fresh maps for every invocation.
func Evaluate(graph *source.Graph, parsed map[source.FileID]*File, options EvalOptions) *Evaluation {
	return evaluateOrder(graph, parsed, options, sourceOrder(graph, parsed))
}

func evaluateOrder(graph *source.Graph, parsed map[source.FileID]*File, options EvalOptions, order []source.FileID) *Evaluation {
	e := &evaluator{
		graph:    graph,
		result:   &Evaluation{Accounts: make(map[string]AccountState), Prices: make(map[string][]PriceQuote), Options: make(map[string]string)},
		options:  options,
		accounts: make(map[string]*accountWork),
		pads:     make(map[string]Pad),
		visited:  make(map[source.FileID]bool),
	}
	if graph != nil && graph.Entry != 0 {
		e.evaluateFile(graph.Entry, parsed)
	} else {
		for _, fileID := range order {
			e.evaluateFile(fileID, parsed)
		}
	}
	for account, pad := range e.pads {
		e.add("W-EVAL-PAD-UNUSED", diagnostic.Warning, pad.Span(), e.pathFor(pad.Span()))
		_ = account
	}
	e.finish()
	return e.result
}

// EvaluateFiles is a graph-independent helper for tests and embedders that
// already have deterministic source order.
func EvaluateFiles(files map[source.FileID]*File, order []source.FileID, options EvalOptions) *Evaluation {
	return evaluateOrder(nil, files, options, append([]source.FileID(nil), order...))
}

func sourceOrder(graph *source.Graph, parsed map[source.FileID]*File) []source.FileID {
	if graph != nil && len(graph.Order) != 0 {
		return append([]source.FileID(nil), graph.Order...)
	}
	order := make([]source.FileID, 0, len(parsed))
	for id := range parsed {
		order = append(order, id)
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	return order
}

type accountWork struct {
	state    AccountState
	lastDate Date
}

type evaluator struct {
	graph    *source.Graph
	result   *Evaluation
	options  EvalOptions
	accounts map[string]*accountWork
	pads     map[string]Pad
	visited  map[source.FileID]bool
	events   []balanceEvent
	pending  []pendingBalance
}

type balanceEvent struct {
	account  string
	currency string
	date     Date
	amount   Decimal
	span     source.Span
}

type pendingBalance struct {
	account   string
	currency  string
	date      Date
	expected  Decimal
	tolerance Decimal
	span      source.Span
	path      string
}

func (e *evaluator) evaluateFile(fileID source.FileID, parsed map[source.FileID]*File) {
	if e.visited[fileID] {
		return
	}
	file := parsed[fileID]
	if file == nil {
		return
	}
	e.visited[fileID] = true
	for _, directive := range file.Directives {
		if include, ok := directive.(Include); ok && e.graph != nil {
			for _, edge := range e.graph.Edges[fileID] {
				if edge.Literal == include.Path {
					e.evaluateFile(edge.To, parsed)
					break
				}
			}
			continue
		}
		e.evaluateDirective(file, directive)
	}
}

// evaluateDirective folds one parsed directive into the evaluation: entry
// bookkeeping first, then the per-type state machine. Options configure
// loading and are retained by the parsed source and the evaluation Options
// map, but Beancount does not expose them in its loaded entry stream;
// isEntryDirective keeps Evaluation.Entries aligned with that stream while
// plugin/tag declarations stay retained for source-preserving diagnostics.
func (e *evaluator) evaluateDirective(file *File, directive Directive) {
	span := directive.Span()
	path := sourcePath(file)
	if isEntryDirective(directive) {
		e.result.Entries = append(e.result.Entries, EntryRecord{Span: span, Directive: directive, File: path, Date: directiveDate(directive)})
	}
	switch d := directive.(type) {
	case Option:
		e.applyOption(d)
	case Plugin:
		// The parser already emits the migration warning. Plugin code is never
		// executed in this implementation.
	case Include, TagDirective, Query, Event, Note, Document, Custom:
		// These directives are source-preserved and consumed by later report
		// layers; they do not mutate account state in the core evaluator.
	case Dialect:
		// Source-only shorthand: the dialect pass replaces it before
		// evaluation. Reaching here means a caller bypassed snapshot's
		// expansion, so warn loudly instead of dropping it silently.
		e.add("W-DIALECT-UNEXPANDED", diagnostic.Warning, span, path)
	case Open:
		e.open(d)
	case Close:
		e.close(d)
	case Commodity:
		// Commodity declarations establish no balance by themselves.
	case Price:
		e.price(d)
	case Pad:
		e.pads[d.Account] = d
	case Balance:
		e.balance(d)
	case *Transaction:
		e.evaluateTransaction(d)
	case Transaction:
		copy := d
		e.evaluateTransaction(&copy)
	default:
		e.add("W-EVAL-UNSUPPORTED", diagnostic.Warning, span, path)
	}
}

// evaluateTransaction books a transaction and folds its postings in.
func (e *evaluator) evaluateTransaction(tx *Transaction) {
	booked := e.bookTransaction(tx)
	e.replaceLastEntry(booked)
	e.transaction(booked)
}

// sourcePath returns the display path of the file a directive came from.
func sourcePath(file *File) string {
	if file != nil && file.Source != nil {
		return file.Source.Path
	}
	return ""
}

// applyOption records one option declaration. Repeated operating_currency
// values accumulate (see appendOperatingCurrency); every other key is
// single-valued and overwrites. The tolerance option also seeds the
// transaction-balancing tolerance after validation.
func (e *evaluator) applyOption(d Option) {
	if strings.EqualFold(d.Key, "operating_currency") {
		e.result.Options[d.Key] = appendOperatingCurrency(e.result.Options[d.Key], d.Value)
	} else {
		e.result.Options[d.Key] = d.Value
	}
	if !strings.EqualFold(d.Key, "tolerance") {
		return
	}
	tolerance, err := ParseDecimal(d.Value)
	switch {
	case err != nil:
		e.add("E-EVAL-OPTION", diagnostic.Error, d.Span(), e.pathFor(d.Span()))
	case tolerance.Sign() < 0:
		e.add("E-EVAL-TOLERANCE", diagnostic.Error, d.Span(), e.pathFor(d.Span()))
	default:
		e.options.DefaultTolerance = tolerance
	}
}

// appendOperatingCurrency joins repeated operating_currency declarations into
// a space-separated list in declaration order, keeping the first-declared
// currency (the ledger's primary operating currency) first and skipping
// duplicates.
func appendOperatingCurrency(existing, value string) string {
	if existing == "" {
		return value
	}
	for _, part := range strings.Fields(existing) {
		if part == value {
			return existing
		}
	}
	return existing + " " + value
}

func isEntryDirective(d Directive) bool {
	switch d.(type) {
	case Option, *Option:
		return false
	default:
		return true
	}
}

func (e *evaluator) replaceLastEntry(d Directive) {
	if d == nil || len(e.result.Entries) == 0 {
		return
	}
	e.result.Entries[len(e.result.Entries)-1].Directive = d
}

// directiveDate returns the date of any directive type for chronological
// ordering; transactions carry it in DirectiveBase so they fall through to
// the default.
func directiveDate(d Directive) Date {
	switch x := d.(type) {
	case Open:
		return x.Date
	case Close:
		return x.Date
	case Commodity:
		return x.Date
	case Balance:
		return x.Date
	case Pad:
		return x.Date
	case Event:
		return x.Date
	case Query:
		return x.Date
	case Price:
		return x.Date
	case Document:
		return x.Date
	case Note:
		return x.Date
	case Custom:
		return x.Date
	case Transaction:
		return x.Date
	case *Transaction:
		return x.Date
	default:
		return Date{}
	}
}

func (e *evaluator) open(d Open) {
	if d.Account == "" {
		e.add("E-EVAL-OPEN", diagnostic.Error, d.Span(), e.pathFor(d.Span()))
		return
	}
	if existing, ok := e.accounts[d.Account]; ok {
		// Reopening a closed account and opening an already-open account are
		// distinct lifecycle errors with distinct diagnostics.
		if existing.state.Closed != nil {
			e.add("E-EVAL-REOPEN", diagnostic.Error, d.Span(), e.pathFor(d.Span()))
		} else {
			e.add("E-EVAL-OPEN", diagnostic.Error, d.Span(), e.pathFor(d.Span()))
		}
		return
	}
	e.accounts[d.Account] = &accountWork{state: AccountState{Name: d.Account, Opened: d.Date, Currencies: append([]string(nil), d.Currencies...), Booking: d.Booking, Balances: make(map[string]Decimal)}}
}

func (e *evaluator) close(d Close) {
	account, ok := e.accounts[d.Account]
	if !ok || account.state.Closed != nil || !dateAtLeast(d.Date, account.state.Opened) {
		e.add("E-EVAL-CLOSE", diagnostic.Error, d.Span(), e.pathFor(d.Span()))
		return
	}
	closed := d.Date
	account.state.Closed = &closed
}

func (e *evaluator) price(d Price) {
	if d.Currency == "" || d.Amount.Currency == "" || d.Amount.Number.Raw == "" {
		e.add("E-EVAL-OPTION", diagnostic.Error, d.Span(), e.pathFor(d.Span()))
		return
	}
	quote := PriceQuote{Date: d.Date, Base: d.Currency, Amount: DecimalFromNumber(d.Amount.Number), Currency: d.Amount.Currency, Span: d.Span()}
	e.result.Prices[d.Currency] = append(e.result.Prices[d.Currency], quote)
}

// resolvedPosting is one posting of a transaction with its units fully
// interpolated: amount and currency are the values the balancing and state
// updates must use.
type resolvedPosting struct {
	posting  Posting
	amount   Decimal
	currency string
}

// transaction balances one booked transaction and folds its postings into
// account state. At most one posting may elide its amount (Beancount's
// interpolation); the elided leg is inferred from the imbalance, completed
// in the entry stream itself (see inferElision), and every known leg is then
// applied. A residual imbalance beyond the configured tolerance is reported
// once per transaction.
func (e *evaluator) transaction(tx *Transaction) {
	if tx == nil || len(tx.Postings) == 0 {
		if tx != nil {
			e.add("E-EVAL-UNBALANCED", diagnostic.Error, tx.Span(), e.pathFor(tx.Span()))
		}
		return
	}
	known, missing, totals := collectResolvedPostings(tx, e.addContribution)
	if len(missing) > 1 {
		for _, index := range missing {
			posting := tx.Postings[index]
			e.add("E-EVAL-INFER", diagnostic.Error, posting.Span(), e.pathFor(posting.Span()))
		}
	} else if len(missing) == 1 {
		if resolved, ok := e.inferElision(tx, missing[0], totals); ok {
			known = append(known, resolved)
			e.addContribution(totals, resolved.posting, resolved.amount, resolved.currency)
		}
	}
	tolerance := e.options.DefaultTolerance
	if tolerance.IsZero() && e.options.InferDecimalTolerance {
		tolerance = inferredTolerance(tx.Postings)
	}
	for _, total := range totals {
		if !within(total, tolerance) {
			e.add("E-EVAL-UNBALANCED", diagnostic.Error, tx.Span(), e.pathFor(tx.Span()))
			break
		}
	}
	for _, item := range known {
		e.applyPosting(tx.Date, item.posting, item.amount, item.currency)
	}
}

// addContribution folds one posting's amount into the per-currency totals.
type contributionFunc func(totals map[string]Decimal, posting Posting, amount Decimal, currency string)

// collectResolvedPostings partitions the transaction's postings into fully
// specified legs and legs with a missing amount side, accumulating the
// per-currency running totals of the known legs.
func collectResolvedPostings(tx *Transaction, addContribution contributionFunc) (known []resolvedPosting, missing []int, totals map[string]Decimal) {
	totals = make(map[string]Decimal)
	for index, posting := range tx.Postings {
		if posting.Units == nil || posting.Units.Currency == "" || posting.Units.Number.Raw == "" {
			missing = append(missing, index)
			continue
		}
		amount := DecimalFromNumber(posting.Units.Number)
		known = append(known, resolvedPosting{posting: posting, amount: amount, currency: posting.Units.Currency})
		addContribution(totals, posting, amount, posting.Units.Currency)
	}
	return known, missing, totals
}

// inferElision completes the single amount-less posting: it infers the
// currency, computes the balancing amount against the other legs, and writes
// the completed units back into the entry stream. Completing the posting in
// the stream itself (rather than only in local state) keeps every consumer
// that reads Entries - journal rows, query results, report charts - from
// silently skipping the interpolated leg, so a purchase counts both the
// shares bought and the cash that paid for them.
func (e *evaluator) inferElision(tx *Transaction, index int, totals map[string]Decimal) (resolvedPosting, bool) {
	posting := tx.Postings[index]
	currency := e.inferPostingCurrency(posting, totals)
	balanceCurrency, factor, unitsInBalance, ok := e.inferenceTarget(posting, currency, totals)
	if !ok || factor.IsZero() {
		e.add("E-EVAL-INFER", diagnostic.Error, posting.Span(), e.pathFor(posting.Span()))
		return resolvedPosting{}, false
	}
	inferred := divideDecimal(totals[balanceCurrency].Neg(), factor)
	if unitsInBalance {
		// The residual being absorbed lives in balanceCurrency, so the
		// completed units must be denominated there; the account's standing
		// balance may otherwise suggest a different currency entirely.
		currency = balanceCurrency
	}
	units := Amount{Number: numberFromDecimal(inferred), Currency: currency}
	tx.Postings[index].Units = &units
	return resolvedPosting{posting: tx.Postings[index], amount: inferred, currency: currency}, true
}

// bookTransaction resolves reducing cost specifications against the account's
// existing lots before interpolation and evaluation. Beancount may split one
// source reduction across several matching lots; the booked transaction keeps
// those derived postings tied to the original source span while exposing the
// same normalized view used by holdings and query reports.
func (e *evaluator) bookTransaction(tx *Transaction) *Transaction {
	if tx == nil || len(tx.Postings) == 0 {
		return tx
	}
	booked := *tx
	booked.Postings = make([]Posting, 0, len(tx.Postings))
	working := make(map[string][]Position)
	for account, state := range e.accounts {
		working[account] = append([]Position(nil), state.state.Positions...)
	}
	for _, posting := range tx.Postings {
		resolved := e.bookPosting(posting, working)
		booked.Postings = append(booked.Postings, resolved...)
		for _, item := range resolved {
			updateWorkingPositions(working, item)
		}
	}
	return &booked
}

// bookPosting is the inventory-resolution pre-pass for one reducing posting
// (a posting with both units and an explicit cost whose number is negative).
// It matches the reduction against the account's current lots and splits it
// into per-lot legs under the account's booking method. Non-reducing or
// unmatched postings pass through unchanged. When the inventory cannot
// satisfy the reduction it deliberately returns the original posting
// unchanged: applyPosting later attempts the same reduction against the
// authoritative account state and emits E-EVAL-INVENTORY exactly once.
// Reporting in both places produced a duplicate diagnostic for one oversell.
func (e *evaluator) bookPosting(posting Posting, working map[string][]Position) []Posting {
	if !isReducingPosting(posting) {
		return []Posting{posting}
	}
	amount := DecimalFromNumber(posting.Units.Number)
	account := e.accounts[posting.Account]
	if account == nil || strings.EqualFold(account.state.Booking, "NONE") {
		return []Posting{posting}
	}
	constraints := deriveCostConstraints(*posting.Cost)
	ordered := orderBookingMatches(bookingMatches(working[posting.Account], posting.Units.Currency, constraints), account.state.Booking)
	if len(ordered) == 0 {
		return []Posting{posting}
	}
	requested := amount.Neg()
	var allocations []Position
	switch {
	case isOrderedBooking(account.state.Booking):
		allocations = allocateOrderedReduction(ordered, requested)
	case strings.EqualFold(account.state.Booking, "AVERAGE"):
		return e.bookAverageReduction(posting, ordered, requested, constraints)
	default:
		allocations = allocateExactReduction(ordered, requested)
	}
	if allocations == nil {
		return []Posting{posting}
	}
	return allocationPostings(posting, allocations)
}

// isReducingPosting reports whether the posting is a negative-amount posting
// with the units and explicit cost inventory booking requires.
func isReducingPosting(posting Posting) bool {
	if posting.Cost == nil || posting.Units == nil || posting.Units.Currency == "" || posting.Units.Number.Raw == "" {
		return false
	}
	return DecimalFromNumber(posting.Units.Number).Sign() < 0
}

// isOrderedBooking reports whether the method consumes lots in a fixed order.
func isOrderedBooking(booking string) bool {
	switch strings.ToLower(booking) {
	case "fifo", "lifo", "hifo":
		return true
	default:
		return false
	}
}

// bookingMatches selects the positive lots of one currency whose cost
// satisfies the reduction's cost constraints.
func bookingMatches(positions []Position, currency string, constraints costConstraints) []Position {
	matches := make([]Position, 0)
	for _, position := range positions {
		if position.Units.Sign() <= 0 || position.Currency != currency || position.Cost == nil {
			continue
		}
		if costMatchesPosition(constraints, *position.Cost) {
			matches = append(matches, position)
		}
	}
	return matches
}

// allocateOrderedReduction serves FIFO/LIFO/HIFO bookings: lots are consumed
// in method order until the reduction is covered. A shortfall returns nil so
// the caller can leave the posting to applyPosting's single diagnostic.
func allocateOrderedReduction(ordered []Position, requested Decimal) []Position {
	remaining := requested
	allocations := make([]Position, 0, len(ordered))
	for _, match := range ordered {
		if remaining.IsZero() {
			break
		}
		units := match.Units
		if units.Cmp(remaining) > 0 {
			units = remaining
		}
		match.Units = units
		allocations = append(allocations, match)
		remaining = remaining.Sub(units)
	}
	if !remaining.IsZero() {
		return nil
	}
	return allocations
}

// allocateExactReduction serves STRICT (and unknown) bookings: the full
// reduction must land on the matching lots — all of them together, or the
// single oversized lot trimmed to size. Anything else is a shortfall (nil).
func allocateExactReduction(ordered []Position, requested Decimal) []Position {
	total := Zero()
	for _, match := range ordered {
		total = total.Add(match.Units)
	}
	if total.Equal(requested) {
		return ordered
	}
	if len(ordered) != 1 || ordered[0].Units.Cmp(requested) < 0 {
		return nil
	}
	match := ordered[0]
	match.Units = requested
	return []Position{match}
}

// allocationPostings renders the per-lot allocations back as postings tied to
// the original source span: negated per-lot units plus the lot's cost.
func allocationPostings(posting Posting, allocations []Position) []Posting {
	resolved := make([]Posting, 0, len(allocations))
	for _, allocation := range allocations {
		item := posting
		units := *posting.Units
		units.Number = numberFromDecimal(allocation.Units.Neg())
		item.Units = &units
		cost := costSpecFromCost(*allocation.Cost, posting.Cost)
		item.Cost = &cost
		resolved = append(resolved, item)
	}
	return resolved
}

// bookAverageReduction implements Beancount's Booking.AVERAGE (which v3 defines
// but leaves disabled, returning "AVERAGE method is not supported"). It is the
// first working implementation, per ADR-0042. Matching lots are merged lazily
// at reduction time into a single weighted-average lot (date = earliest
// contributing lot, label cleared), then the reduction is applied to that lot.
// The merge is encoded as multiple legs so both the working-copy and
// authoritative applyPosting paths consume it through the normal
// costMatchesPosition logic without special-casing: legs that remove each old
// lot (flagged "M", mirroring Beancount's FLAG_MERGING), one leg that refills
// the merged lot, and the reduction leg. The remove and refill legs carry the
// same total cost value in opposite directions, so transaction balancing is
// unaffected. Reductions with an explicit cost and cross-cost-currency merges
// are converted to an internal rejected leg. applyPosting then emits
// E-EVAL-INVENTORY exactly once without mutating inventory, consistent with
// how the other booking methods signal failure (no duplicate diagnostic).
func (e *evaluator) bookAverageReduction(posting Posting, matches []Position, requested Decimal, constraints costConstraints) []Posting {
	if constraints.number != nil {
		// AVERAGE's contract is that the engine owns the cost; an explicit
		// reduction cost is rejected. applyPosting reports it once.
		return []Posting{rejectAverageReduction(posting)}
	}
	if len(matches) == 0 {
		return []Posting{posting}
	}
	costCurrency := matches[0].Cost.Currency
	for _, match := range matches[1:] {
		if match.Cost.Currency != costCurrency {
			// Cross-cost-currency lots cannot be averaged together.
			return []Posting{rejectAverageReduction(posting)}
		}
	}
	totalUnits := Zero()
	totalCostValue := Zero()
	var earliest *Date
	for _, match := range matches {
		totalUnits = totalUnits.Add(match.Units)
		totalCostValue = totalCostValue.Add(match.Units.Mul(match.Cost.Number))
		if match.Cost.Date != nil && (earliest == nil || dateKey(*match.Cost.Date) < dateKey(*earliest)) {
			d := *match.Cost.Date
			earliest = &d
		}
	}
	if totalUnits.IsZero() {
		return []Posting{posting}
	}
	avgCost := Cost{
		Number:   totalCostValue.Quo(totalUnits),
		Currency: costCurrency,
		Date:     earliest,
		Label:    "",
	}
	legs := make([]Posting, 0, len(matches)+2)
	// 1. Remove each matching old lot at its own cost.
	for _, match := range matches {
		item := posting
		units := *posting.Units
		units.Number = numberFromDecimal(match.Units.Neg())
		item.Units = &units
		cost := costSpecFromCost(*match.Cost, posting.Cost)
		item.Cost = &cost
		item.Flag = flagMerging
		legs = append(legs, item)
	}
	// 2. Refill the merged lot at the weighted-average cost.
	refill := posting
	refillUnits := *posting.Units
	refillUnits.Number = numberFromDecimal(totalUnits)
	refill.Units = &refillUnits
	refillCost := costSpecFromCost(avgCost, posting.Cost)
	refill.Cost = &refillCost
	refill.Flag = flagMerging
	legs = append(legs, refill)
	// 3. Apply the reduction against the merged lot.
	reduction := posting
	reductionUnits := *posting.Units
	reductionUnits.Number = numberFromDecimal(requested.Neg())
	reduction.Units = &reductionUnits
	reductionCost := costSpecFromCost(avgCost, posting.Cost)
	reduction.Cost = &reductionCost
	legs = append(legs, reduction)
	return legs
}

func rejectAverageReduction(posting Posting) Posting {
	posting.averageRejected = true
	return posting
}

type costConstraints struct {
	number   *Decimal
	currency string
	date     *Date
	label    string
}

func deriveCostConstraints(spec CostSpec) costConstraints {
	constraints := costConstraints{}
	for _, value := range spec.Components {
		switch value.Kind {
		case ValueAmount:
			number := DecimalFromNumber(value.Amount.Number)
			constraints.number = &number
			constraints.currency = value.Amount.Currency
		case ValueCurrency:
			if constraints.currency == "" {
				constraints.currency = value.String
			}
		case ValueDate:
			date := value.Date
			constraints.date = &date
		case ValueString:
			constraints.label = value.String
		}
	}
	return constraints
}

// costMatchesPosition reports whether a lot's cost satisfies the booking
// constraints; nil constraints are wildcards per beancount's exact-match
// semantics.
func costMatchesPosition(constraints costConstraints, cost Cost) bool {
	if constraints.number != nil && cost.Number.Cmp(*constraints.number) != 0 {
		return false
	}
	if constraints.currency != "" && cost.Currency != constraints.currency {
		return false
	}
	if constraints.date != nil && (cost.Date == nil || dateKey(*cost.Date) != dateKey(*constraints.date)) {
		return false
	}
	if constraints.label != "" && cost.Label != constraints.label {
		return false
	}
	return true
}

// orderBookingMatches sorts candidate lots per the account's booking
// method — FIFO ascending and LIFO descending by lot date.
func orderBookingMatches(matches []Position, booking string) []Position {
	ordered := append([]Position(nil), matches...)
	if strings.EqualFold(booking, "FIFO") || strings.EqualFold(booking, "LIFO") {
		sort.SliceStable(ordered, func(i, j int) bool {
			left, right := ordered[i].Cost, ordered[j].Cost
			if left == nil || right == nil || left.Date == nil || right.Date == nil {
				return false
			}
			if strings.EqualFold(booking, "LIFO") {
				return dateKey(*left.Date) > dateKey(*right.Date)
			}
			return dateKey(*left.Date) < dateKey(*right.Date)
		})
	}
	if strings.EqualFold(booking, "HIFO") {
		sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Cost.Number.Cmp(ordered[j].Cost.Number) > 0 })
	}
	return ordered
}

func numberFromDecimal(value Decimal) Number {
	raw := value.String()
	rat := value.Rat()
	return Number{Raw: raw, Rat: rat}
}

func costSpecFromCost(cost Cost, original *CostSpec) CostSpec {
	spec := CostSpec{}
	if original != nil {
		spec = *original
		spec.Total = false
		spec.Components = nil
	}
	amount := Amount{Number: numberFromDecimal(cost.Number), Currency: cost.Currency}
	spec.Components = append(spec.Components, Value{Kind: ValueAmount, Raw: amount.Number.Raw + " " + amount.Currency, Amount: amount})
	if cost.Date != nil {
		spec.Components = append(spec.Components, Value{Kind: ValueDate, Raw: cost.Date.Raw, Date: *cost.Date})
	}
	if cost.Label != "" {
		spec.Components = append(spec.Components, Value{Kind: ValueString, Raw: cost.Label, String: cost.Label})
	}
	return spec
}

// updateWorkingPositions folds an interpolated posting into the per-account
// working inventory used by balance assertions. Additions append a lot at the
// recorded cost; reductions consume matching lots in order without going
// negative — the booking step has already decided the authoritative
// matching, so this only tracks quantities.
func updateWorkingPositions(working map[string][]Position, posting Posting) {
	if posting.averageRejected {
		return
	}
	if posting.Units == nil || posting.Units.Currency == "" || posting.Units.Number.Raw == "" || posting.Cost == nil {
		return
	}
	cost, ok := normalizeCost(*posting.Cost)
	if !ok {
		return
	}
	amount := DecimalFromNumber(posting.Units.Number)
	positions := working[posting.Account]
	if amount.Sign() > 0 {
		working[posting.Account] = append(positions, Position{Units: amount, Currency: posting.Units.Currency, Cost: &cost})
		return
	}
	remaining := amount.Neg()
	for index := 0; index < len(positions) && remaining.Sign() > 0; {
		position := &positions[index]
		if position.Currency != posting.Units.Currency || position.Cost == nil || !costMatchesPosition(costConstraints{number: &cost.Number, currency: cost.Currency, date: cost.Date, label: cost.Label}, *position.Cost) {
			index++
			continue
		}
		consumed := position.Units
		if consumed.Cmp(remaining) > 0 {
			consumed = remaining
		}
		position.Units = position.Units.Sub(consumed)
		remaining = remaining.Sub(consumed)
		if position.Units.IsZero() {
			positions = append(positions[:index], positions[index+1:]...)
			continue
		}
		index++
	}
	working[posting.Account] = positions
}

// addContribution records the value a posting contributes to transaction
// balancing. Costed and priced inventory is balanced in its cost/price
// currency; the lot's unit currency remains in account state but is not a
// second unconvertible balancing leg.
func (e *evaluator) addContribution(totals map[string]Decimal, posting Posting, amount Decimal, unitsCurrency string) {
	currency, value, ok := balancingContribution(posting, amount, unitsCurrency)
	if !ok || currency == "" {
		currency, value = unitsCurrency, amount
	}
	totals[currency] = decimalAdd(totals[currency], value)
}

// balancingContribution returns the posting's balancing weight. Cost takes
// precedence over price: when a posting carries both — the shape every lot
// reduction like `-2200 SHARES {} @ 21.46 CNY` takes — Beancount balances it at
// the cost of the lots removed and treats the price as reporting information
// only. Weighting such a posting at its price instead silently moved the whole
// realized gain out of the inferred residual posting.
func balancingContribution(posting Posting, amount Decimal, unitsCurrency string) (string, Decimal, bool) {
	if posting.Cost != nil {
		if cost, ok := normalizeCost(*posting.Cost); ok && cost.Currency != "" && !cost.Number.IsZero() {
			if posting.Cost.Total {
				return cost.Currency, signedTotal(amount, cost.Number), true
			}
			return cost.Currency, amount.Mul(cost.Number), true
		}
	}
	if posting.Price != nil && posting.Price.Amount.Currency != "" && posting.Price.Amount.Number.Raw != "" {
		price := DecimalFromNumber(posting.Price.Amount.Number)
		if posting.Price.Total {
			return posting.Price.Amount.Currency, signedTotal(amount, price), true
		}
		return posting.Price.Amount.Currency, amount.Mul(price), true
	}
	return unitsCurrency, amount, unitsCurrency != ""
}

func signedTotal(units, total Decimal) Decimal {
	if units.Sign() < 0 {
		return total.Neg()
	}
	return total
}

func divideDecimal(left, right Decimal) Decimal {
	if right.IsZero() {
		return Zero()
	}
	return NewDecimal(new(big.Rat).Quo(left.Rat(), right.Rat()))
}

// inferPostingCurrency resolves a posting's effective currency for balance
// totals: explicit units win, then the cost currency, then the price.
func (e *evaluator) inferPostingCurrency(posting Posting, totals map[string]Decimal) string {
	if posting.Units != nil && posting.Units.Currency != "" {
		return posting.Units.Currency
	}
	if account := e.accounts[posting.Account]; account != nil {
		if len(account.state.Currencies) == 1 {
			return account.state.Currencies[0]
		}
		if len(account.state.Balances) == 1 {
			for currency := range account.state.Balances {
				return currency
			}
		}
	}
	if len(totals) == 1 {
		for currency := range totals {
			return currency
		}
	}
	return ""
}

// inferenceTarget picks the currency an amount-less posting converts into
// and the conversion factor: an explicit price first, then the cost, then the
// posting's own units currency when it already has a balance, and finally a
// sole existing total. unitsInBalance reports that the balancing amount
// itself is denominated in the target currency (the last two branches), so
// the completed posting's units carry that currency — not merely the
// account's standing-balance currency, which a multi-currency account may
// have picked up from an earlier transaction. False means the posting cannot
// be inferred.
func (e *evaluator) inferenceTarget(posting Posting, unitsCurrency string, totals map[string]Decimal) (string, Decimal, bool, bool) {
	if posting.Price != nil && posting.Price.Amount.Currency != "" && posting.Price.Amount.Number.Raw != "" {
		factor := DecimalFromNumber(posting.Price.Amount.Number)
		if posting.Price.Total {
			factor = NewDecimal(big.NewRat(1, 1))
		}
		return posting.Price.Amount.Currency, factor, false, true
	}
	if posting.Cost != nil {
		if cost, ok := normalizeCost(*posting.Cost); ok && cost.Currency != "" && !cost.Number.IsZero() {
			factor := cost.Number
			if posting.Cost.Total {
				factor = NewDecimal(big.NewRat(1, 1))
			}
			return cost.Currency, factor, false, true
		}
	}
	if unitsCurrency != "" {
		if _, ok := totals[unitsCurrency]; ok {
			return unitsCurrency, NewDecimal(big.NewRat(1, 1)), true, true
		}
	}
	if len(totals) == 1 {
		for currency := range totals {
			return currency, NewDecimal(big.NewRat(1, 1)), true, true
		}
	}
	return "", Zero(), false, false
}

// applyPosting folds one fully-interpolated posting into the authoritative
// account state: lifecycle checks, balance totals, and inventory movement.
// The lifecycle and currency checks are advisory (only inventory exhaustion
// and rejected average reductions are hard failures here).
func (e *evaluator) applyPosting(date Date, posting Posting, amount Decimal, currency string) {
	account, ok := e.accounts[posting.Account]
	if !ok || !dateAtLeast(date, account.state.Opened) || (account.state.Closed != nil && !dateBefore(date, *account.state.Closed)) {
		e.add("E-EVAL-POSTING", diagnostic.Error, posting.Span(), e.pathFor(posting.Span()))
		return
	}
	if posting.averageRejected {
		e.add("E-EVAL-INVENTORY", diagnostic.Error, posting.Span(), e.pathFor(posting.Span()))
		return
	}
	if len(account.state.Currencies) != 0 && !contains(account.state.Currencies, currency) {
		e.add("E-EVAL-CURRENCY", diagnostic.Error, posting.Span(), e.pathFor(posting.Span()))
	}
	// Entries are evaluated in deterministic source/include order. Beancount
	// permits that source order to differ from chronological order (the core
	// sorts entries by date before applying state), so a posting date that moves
	// backwards in the source must not be reported as a lifecycle failure.
	if !account.lastDate.Valid() || dateAtLeast(date, account.lastDate) {
		account.lastDate = date
	}
	account.state.Balances[currency] = decimalAdd(account.state.Balances[currency], amount)
	e.events = append(e.events, balanceEvent{account: posting.Account, currency: currency, date: date, amount: amount, span: posting.Span()})
	if posting.Cost == nil {
		return
	}
	cost, ok := normalizeCost(*posting.Cost)
	if !ok {
		return
	}
	e.applyInventory(date, posting, account, amount, currency, cost)
}

// PostingWeight values one posting in its quote currency the way BeanQuery's
// weight column does: units x lot cost when costed, units x price (or the
// @@ total, signed by the units) when priced, plain units otherwise. It is
// the honest per-posting value when no conversion quote exists.
func PostingWeight(posting Posting) Decimal {
	units := DecimalFromNumber(posting.Units.Number)
	if posting.Cost != nil {
		if cost, ok := normalizeCost(*posting.Cost); ok && cost.Currency != "" {
			return NewDecimal(new(big.Rat).Mul(units.Rat(), cost.Number.Rat()))
		}
	}
	if posting.Price != nil && posting.Price.Amount.Currency != "" {
		total := DecimalFromNumber(posting.Price.Amount.Number)
		if posting.Price.Total {
			if units.Rat().Sign() < 0 {
				return NewDecimal(new(big.Rat).Neg(total.Rat()))
			}
			return total
		}
		return NewDecimal(new(big.Rat).Mul(units.Rat(), total.Rat()))
	}
	return units
}

// applyInventory moves the posting's cost-basis inventory: acquisitions
// append a lot; reductions consume matching lots and report E-EVAL-INVENTORY
// when the inventory cannot cover the full amount.
func (e *evaluator) applyInventory(date Date, posting Posting, account *accountWork, amount Decimal, currency string, cost Cost) {
	if amount.Sign() >= 0 {
		account.state.Positions = append(account.state.Positions, Position{Units: amount, Currency: currency, Cost: &cost, Span: posting.Span()})
		return
	}
	remaining := consumePositions(&account.state.Positions, amount.Neg(), currency, costConstraints{number: &cost.Number, currency: cost.Currency, date: cost.Date, label: cost.Label})
	if remaining.Sign() > 0 {
		e.add("E-EVAL-INVENTORY", diagnostic.Error, posting.Span(), e.pathFor(posting.Span()))
	}
}

// consumePositions removes up to requested units of one currency from the
// lots whose cost matches the constraints, deleting emptied lots. It returns
// the part of the request the inventory could not cover.
func consumePositions(positions *[]Position, requested Decimal, currency string, constraints costConstraints) Decimal {
	list := *positions
	remaining := requested
	for i := 0; i < len(list) && remaining.Sign() > 0; {
		position := &list[i]
		if position.Currency != currency || position.Cost == nil || !costMatchesPosition(constraints, *position.Cost) {
			i++
			continue
		}
		consumed := position.Units
		if consumed.Cmp(remaining) > 0 {
			consumed = remaining
		}
		position.Units = position.Units.Sub(consumed)
		remaining = remaining.Sub(consumed)
		if position.Units.IsZero() {
			list = append(list[:i], list[i+1:]...)
			continue
		}
		i++
	}
	*positions = list
	return remaining
}

// balance evaluates a balance directive against the working inventory: the
// expected amount is compared with the accumulated positions of that
// currency within tolerance, and a mismatch or unusable amount is an error.
func (e *evaluator) balance(d Balance) {
	currency := d.Amount.Currency
	if currency == "" || d.Amount.Number.Raw == "" {
		e.add("E-EVAL-BALANCE", diagnostic.Error, d.Span(), e.pathFor(d.Span()))
		return
	}
	if account, ok := e.accounts[d.Account]; ok {
		if pad, hasPad := e.pads[d.Account]; hasPad {
			e.applyPad(d, pad, account)
			delete(e.pads, d.Account)
		}
	}
	tolerance := e.options.DefaultTolerance
	if d.Tolerance != nil {
		tolerance = DecimalFromNumber(*d.Tolerance)
		if tolerance.Sign() < 0 {
			e.add("E-EVAL-TOLERANCE", diagnostic.Error, d.Span(), e.pathFor(d.Span()))
			return
		}
	}
	if tolerance.IsZero() && e.options.InferDecimalTolerance {
		tolerance = inferredTolerance([]Posting{{Units: &d.Amount}})
	}
	e.pending = append(e.pending, pendingBalance{account: d.Account, currency: currency, date: d.Date, expected: DecimalFromNumber(d.Amount.Number), tolerance: tolerance, span: d.Span(), path: e.pathFor(d.Span())})
}

func (e *evaluator) applyPad(balance Balance, pad Pad, target *accountWork) {
	sourceAccount, ok := e.accounts[pad.SourceAccount]
	if !ok || sourceAccount.state.Closed != nil {
		e.add("E-EVAL-PAD", diagnostic.Error, pad.Span(), e.pathFor(pad.Span()))
		return
	}
	currency := balance.Amount.Currency
	expected := DecimalFromNumber(balance.Amount.Number)
	actual := target.state.Balances[currency]
	difference := expected.Sub(actual)
	if difference.IsZero() {
		return
	}
	target.state.Balances[currency] = decimalAdd(target.state.Balances[currency], difference)
	sourceAccount.state.Balances[currency] = decimalAdd(sourceAccount.state.Balances[currency], difference.Neg())
	// The pad adjustment is a synthetic entry dated at the pad directive's date,
	// not at the balance assertion that consumes it. Booking the events at the
	// balance date would make them same-day with the assertion and therefore
	// excluded by eventBeforeBalance, so a pad that brings the account to the
	// expected balance would incorrectly fail the assertion.
	e.events = append(e.events,
		balanceEvent{account: balance.Account, currency: currency, date: pad.Date, amount: difference, span: pad.Span()},
		balanceEvent{account: pad.SourceAccount, currency: currency, date: pad.Date, amount: difference.Neg(), span: pad.Span()},
	)
}

// finish closes out evaluation: each deferred balance assertion is replayed
// against the pad adjustments recorded before it (within tolerance) and its
// account's lifecycle window, then the final account states are snapshotted.
func (e *evaluator) finish() {
	for _, pending := range e.pending {
		account, ok := e.accounts[pending.account]
		if !ok || !dateAtLeast(pending.date, account.state.Opened) || (account.state.Closed != nil && !dateBefore(pending.date, *account.state.Closed)) {
			e.add("E-EVAL-BALANCE", diagnostic.Error, pending.span, pending.path)
			continue
		}
		actual := Zero()
		for _, event := range e.events {
			if accountWithin(pending.account, event.account) && event.currency == pending.currency && e.eventBeforeBalance(event, pending) {
				actual = actual.Add(event.amount)
			}
		}
		if !within(actual.Sub(pending.expected), pending.tolerance) {
			e.add("E-EVAL-BALANCE", diagnostic.Error, pending.span, pending.path)
		}
	}
	for name, account := range e.accounts {
		state := account.state
		state.Currencies = append([]string(nil), state.Currencies...)
		state.Balances = cloneDecimals(state.Balances)
		state.Positions = append([]Position(nil), state.Positions...)
		if state.Closed != nil {
			closed := *state.Closed
			state.Closed = &closed
		}
		e.result.Accounts[name] = state
	}
	e.result.Valid = true
	for _, d := range e.result.Diagnostics {
		if d.Severity == diagnostic.Error {
			e.result.Valid = false
			break
		}
	}
	e.result.Diagnostics = append([]diagnostic.Diagnostic(nil), e.result.Diagnostics...)
}

func accountWithin(parent, account string) bool {
	return account == parent || strings.HasPrefix(account, parent+":")
}

// eventBeforeBalance reports whether an account event contributes to a
// balance assertion. Assertions include all earlier dates. Beancount sorts
// Balance directives ahead of ordinary same-day transactions, so same-day
// postings are deliberately excluded regardless of textual source position.
func (e *evaluator) eventBeforeBalance(event balanceEvent, balance pendingBalance) bool {
	eventDate, balanceDate := dateKey(event.date), dateKey(balance.date)
	if eventDate < balanceDate {
		return true
	}
	// Beancount's entry sort key places Balance directives before all ordinary
	// same-day transactions (regardless of their textual position). A posting
	// dated on the assertion day therefore cannot satisfy that assertion until
	// the following day.
	return false
}

func (e *evaluator) add(code string, severity diagnostic.Severity, span source.Span, path string) {
	d := diagnostic.New(code, severity, span).WithPath(path)
	e.result.Diagnostics = append(e.result.Diagnostics, d)
}

func (e *evaluator) pathFor(span source.Span) string {
	if e.graph != nil {
		return e.graph.Path(span.File)
	}
	return ""
}

// normalizeCost collapses a cost spec's component values into one Cost:
// number-per-currency pairs merge additively (the beancount "{number #
// number}" form), and a date and label attach when present. ok is false when
// the components do not resolve to a single amount.
func normalizeCost(spec CostSpec) (Cost, bool) {
	cost := Cost{}
	for _, value := range spec.Components {
		switch value.Kind {
		case ValueAmount:
			if cost.Currency == "" {
				cost.Number = DecimalFromNumber(value.Amount.Number)
				cost.Currency = value.Amount.Currency
			}
		case ValueCurrency:
			if cost.Currency == "" {
				cost.Currency = value.String
			}
		case ValueDate:
			date := value.Date
			cost.Date = &date
		case ValueString:
			if cost.Label == "" {
				cost.Label = value.String
			}
		}
	}
	if cost.Currency == "" {
		return Cost{}, false
	}
	return cost, true
}

func inferredTolerance(postings []Posting) Decimal {
	maxScale := 0
	for _, posting := range postings {
		if posting.Units == nil {
			continue
		}
		raw := posting.Units.Number.Raw
		if dot := strings.IndexByte(raw, '.'); dot >= 0 {
			scale := len(raw) - dot - 1
			if scale > maxScale {
				maxScale = scale
			}
		}
	}
	if maxScale == 0 {
		return Zero()
	}
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(maxScale)), nil)
	return NewDecimal(new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Mul(denom, big.NewInt(2))))
}

func decimalAdd(left, right Decimal) Decimal {
	return left.Add(right)
}

func within(value, tolerance Decimal) bool {
	if value.Sign() < 0 {
		value = value.Neg()
	}
	return value.Cmp(tolerance) <= 0
}

func cloneDecimals(values map[string]Decimal) map[string]Decimal {
	copyValues := make(map[string]Decimal, len(values))
	for key, value := range values {
		copyValues[key] = NewDecimal(value.Rat())
	}
	return copyValues
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func dateAtLeast(left, right Date) bool { return dateKey(left) >= dateKey(right) }
func dateBefore(left, right Date) bool  { return dateKey(left) < dateKey(right) }

func dateKey(date Date) int {
	return date.Year*10000 + date.Month*100 + date.Day
}

// Account returns a deep copy of the account's evaluated state so callers
// cannot mutate the evaluation's internal balances.
func (e *Evaluation) Account(name string) (AccountState, bool) {
	if e == nil {
		return AccountState{}, false
	}
	state, ok := e.Accounts[name]
	if !ok {
		return AccountState{}, false
	}
	state.Balances = cloneDecimals(state.Balances)
	state.Currencies = append([]string(nil), state.Currencies...)
	state.Positions = append([]Position(nil), state.Positions...)
	return state, true
}

func (e *Evaluation) String() string {
	if e == nil {
		return "<nil evaluation>"
	}
	return fmt.Sprintf("accounts=%d prices=%d valid=%t", len(e.Accounts), len(e.Prices), e.Valid)
}
