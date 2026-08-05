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

func (e *evaluator) evaluateDirective(file *File, directive Directive) {
	span := directive.Span()
	path := ""
	if file != nil && file.Source != nil {
		path = file.Source.Path
	}
	// Options configure loading and are retained by the parsed source and the
	// evaluation Options map, but Beancount does not expose them in its loaded
	// entry stream. Keep Evaluation.Entries aligned with that stream while
	// retaining plugin/tag declarations for source-preserving diagnostics.
	if isEntryDirective(directive) {
		e.result.Entries = append(e.result.Entries, EntryRecord{Span: span, Directive: directive, File: path, Date: directiveDate(directive)})
	}
	switch d := directive.(type) {
	case Option:
		// Beancount accumulates repeated operating_currency declarations
		// instead of letting the last one win (every other option key is
		// single-valued and overwrites). Losing all but the last declared
		// operating currency previously made downstream consumers - the web
		// UI's default report/chart currency in particular - pick a currency
		// that was never the ledger's primary one.
		if strings.EqualFold(d.Key, "operating_currency") {
			e.result.Options[d.Key] = appendOperatingCurrency(e.result.Options[d.Key], d.Value)
		} else {
			e.result.Options[d.Key] = d.Value
		}
		if strings.EqualFold(d.Key, "tolerance") {
			tolerance, err := ParseDecimal(d.Value)
			if err != nil {
				e.add("E-EVAL-OPTION", diagnostic.Error, d.Span(), e.pathFor(d.Span()))
			} else {
				e.options.DefaultTolerance = tolerance
			}
		}
	case Plugin:
		// The parser already emits the migration warning. Plugin code is never
		// executed in this implementation.
	case Include, TagDirective, Query, Event, Note, Document, Custom:
		// These directives are source-preserved and consumed by later report
		// layers; they do not mutate account state in the core evaluator.
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
		booked := e.bookTransaction(d)
		e.replaceLastEntry(booked)
		e.transaction(booked)
	case Transaction:
		copy := d
		booked := e.bookTransaction(&copy)
		e.replaceLastEntry(booked)
		e.transaction(booked)
	default:
		e.add("W-EVAL-UNSUPPORTED", diagnostic.Warning, span, path)
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

func (e *evaluator) transaction(tx *Transaction) {
	if tx == nil || len(tx.Postings) == 0 {
		if tx != nil {
			e.add("E-EVAL-UNBALANCED", diagnostic.Error, tx.Span(), e.pathFor(tx.Span()))
		}
		return
	}
	type knownPosting struct {
		posting  Posting
		amount   Decimal
		currency string
	}
	known := make([]knownPosting, 0, len(tx.Postings))
	missing := make([]int, 0, len(tx.Postings))
	totals := make(map[string]Decimal)
	for index, posting := range tx.Postings {
		if posting.Units == nil || posting.Units.Currency == "" || posting.Units.Number.Raw == "" {
			missing = append(missing, index)
			continue
		}
		amount := DecimalFromNumber(posting.Units.Number)
		known = append(known, knownPosting{posting: posting, amount: amount, currency: posting.Units.Currency})
		e.addContribution(totals, posting, amount, posting.Units.Currency)
	}
	if len(missing) > 1 {
		for _, index := range missing {
			posting := tx.Postings[index]
			e.add("E-EVAL-INFER", diagnostic.Error, posting.Span(), e.pathFor(posting.Span()))
		}
	} else if len(missing) == 1 {
		index := missing[0]
		posting := tx.Postings[index]
		currency := e.inferPostingCurrency(posting, totals)
		balanceCurrency, factor, ok := e.inferenceTarget(posting, currency, totals)
		if !ok || factor.IsZero() {
			e.add("E-EVAL-INFER", diagnostic.Error, posting.Span(), e.pathFor(posting.Span()))
		} else {
			inferred := totals[balanceCurrency].Neg()
			inferred = divideDecimal(inferred, factor)
			// Complete the posting in the entry stream, the way Beancount's
			// loader hands back a fully interpolated transaction. Leaving the
			// amount only in local evaluator state made every consumer that
			// reads Entries - journal rows, query results, and the report
			// charts - silently skip the interpolated posting, so a purchase
			// counted its shares but never the cash that paid for them.
			units := Amount{Number: numberFromDecimal(inferred), Currency: currency}
			tx.Postings[index].Units = &units
			posting = tx.Postings[index]
			known = append(known, knownPosting{posting: posting, amount: inferred, currency: currency})
			e.addContribution(totals, posting, inferred, currency)
		}
	}
	tolerance := e.options.DefaultTolerance
	if tolerance.IsZero() && e.options.InferDecimalTolerance {
		tolerance = inferredTolerance(tx.Postings)
	}
	for currency, total := range totals {
		if !within(total, tolerance) {
			e.add("E-EVAL-UNBALANCED", diagnostic.Error, tx.Span(), e.pathFor(tx.Span()))
			_ = currency
			break
		}
	}
	for _, item := range known {
		e.applyPosting(tx.Date, item.posting, item.amount, item.currency)
	}
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

func (e *evaluator) bookPosting(posting Posting, working map[string][]Position) []Posting {
	if posting.Cost == nil || posting.Units == nil || posting.Units.Currency == "" || posting.Units.Number.Raw == "" {
		return []Posting{posting}
	}
	amount := DecimalFromNumber(posting.Units.Number)
	if amount.Sign() >= 0 {
		return []Posting{posting}
	}
	account := e.accounts[posting.Account]
	if account == nil || strings.EqualFold(account.state.Booking, "NONE") {
		return []Posting{posting}
	}
	constraints := deriveCostConstraints(*posting.Cost)
	matches := make([]Position, 0)
	for _, position := range working[posting.Account] {
		if position.Units.Sign() <= 0 || position.Currency != posting.Units.Currency || position.Cost == nil {
			continue
		}
		if costMatchesPosition(constraints, *position.Cost) {
			matches = append(matches, position)
		}
	}
	if len(matches) == 0 {
		return []Posting{posting}
	}
	ordered := orderBookingMatches(matches, account.state.Booking)
	requested := amount.Neg()
	allocations := make([]Position, 0, len(ordered))
	if strings.EqualFold(account.state.Booking, "FIFO") || strings.EqualFold(account.state.Booking, "LIFO") || strings.EqualFold(account.state.Booking, "HIFO") {
		remaining := requested
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
		// bookPosting is a resolution pre-pass; it deliberately does not report
		// inventory exhaustion here. When it cannot satisfy the reduction it
		// returns the original posting unchanged, and applyPosting later
		// attempts the same reduction against the authoritative account state
		// and emits E-EVAL-INVENTORY exactly once if it fails. Reporting in both
		// places produced a duplicate diagnostic for a single oversell.
		if !remaining.IsZero() {
			return []Posting{posting}
		}
	} else {
		total := Zero()
		for _, match := range ordered {
			total = total.Add(match.Units)
		}
		if !total.Equal(requested) {
			if len(ordered) != 1 {
				return []Posting{posting}
			}
			match := ordered[0]
			if match.Units.Cmp(requested) < 0 {
				return []Posting{posting}
			}
			match.Units = requested
			allocations = append(allocations, match)
		} else {
			allocations = ordered
		}
	}
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

func updateWorkingPositions(working map[string][]Position, posting Posting) {
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

func (e *evaluator) inferenceTarget(posting Posting, unitsCurrency string, totals map[string]Decimal) (string, Decimal, bool) {
	if posting.Price != nil && posting.Price.Amount.Currency != "" && posting.Price.Amount.Number.Raw != "" {
		factor := DecimalFromNumber(posting.Price.Amount.Number)
		if posting.Price.Total {
			factor = NewDecimal(big.NewRat(1, 1))
		}
		return posting.Price.Amount.Currency, factor, true
	}
	if posting.Cost != nil {
		if cost, ok := normalizeCost(*posting.Cost); ok && cost.Currency != "" && !cost.Number.IsZero() {
			factor := cost.Number
			if posting.Cost.Total {
				factor = NewDecimal(big.NewRat(1, 1))
			}
			return cost.Currency, factor, true
		}
	}
	if unitsCurrency != "" {
		if _, ok := totals[unitsCurrency]; ok {
			return unitsCurrency, NewDecimal(big.NewRat(1, 1)), true
		}
	}
	if len(totals) == 1 {
		for currency := range totals {
			return currency, NewDecimal(big.NewRat(1, 1)), true
		}
	}
	return "", Zero(), false
}

func (e *evaluator) applyPosting(date Date, posting Posting, amount Decimal, currency string) {
	account, ok := e.accounts[posting.Account]
	if !ok || !dateAtLeast(date, account.state.Opened) || (account.state.Closed != nil && !dateBefore(date, *account.state.Closed)) {
		e.add("E-EVAL-POSTING", diagnostic.Error, posting.Span(), e.pathFor(posting.Span()))
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
	if amount.Sign() >= 0 {
		account.state.Positions = append(account.state.Positions, Position{Units: amount, Currency: currency, Cost: &cost, Span: posting.Span()})
		return
	}
	remaining := amount.Neg()
	constraints := costConstraints{number: &cost.Number, currency: cost.Currency, date: cost.Date, label: cost.Label}
	for i := 0; i < len(account.state.Positions) && remaining.Sign() > 0; {
		position := &account.state.Positions[i]
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
			account.state.Positions = append(account.state.Positions[:i], account.state.Positions[i+1:]...)
			continue
		}
		i++
	}
	if remaining.Sign() > 0 {
		e.add("E-EVAL-INVENTORY", diagnostic.Error, posting.Span(), e.pathFor(posting.Span()))
	}
}

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
