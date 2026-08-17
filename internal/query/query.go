// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package query implements a deterministic, read-only BeanQuery-shaped
// workbench over evaluated ledger rows.
package query

import (
	"encoding/csv"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"orangecount/internal/ledger"
)

// Row is one result record keyed by output column name.
type Row map[string]any

// Result is a materialized query result: ordered columns plus rows.
type Result struct {
	Columns []string `json:"columns"`
	Rows    []Row    `json:"rows"`
}

// WriteCSV serializes the result as CSV, header first, missing values as
// empty cells.
func (r Result) WriteCSV(w io.Writer) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(r.Columns); err != nil {
		return err
	}
	for _, row := range r.Rows {
		values := make([]string, len(r.Columns))
		for i, column := range r.Columns {
			values[i] = formatValue(row[column])
		}
		if err := writer.Write(values); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

// Query is a parsed BeanQuery: projection, source table, filters, grouping,
// ordering, and limit.
type Query struct {
	Select  []SelectItem
	From    string
	Where   Expr
	GroupBy []Expr
	OrderBy []OrderItem
	Limit   int
}

// SelectItem is one projected expression with an optional output alias.
type SelectItem struct {
	Expr  Expr
	Alias string
}

// OrderItem is one sort key; Desc reverses its direction.
type OrderItem struct {
	Expr Expr
	Desc bool
}

// Expr is a query expression node. eval receives the current row and the
// full row set (window aggregates); aggregate marks nodes that must run
// once per group rather than per row.
type Expr interface {
	String() string
	eval(row Row, rows []Row) (any, error)
	aggregate() bool
}

// ParseError reports a syntax error in the query text.
type ParseError struct{ Message string }

func (e ParseError) Error() string { return e.Message }

// Parse lexes and parses BeanQuery text into a Query.
func Parse(text string) (Query, error) {
	parser := &queryParser{tokens: lex(text)}
	return parser.parse()
}

// Evaluate parses and runs one query against the evaluated ledger.
func Evaluate(text string, evaluation ledger.Evaluation) (Result, error) {
	query, err := Parse(text)
	if err != nil {
		return Result{}, err
	}
	return EvaluateQuery(query, evaluation)
}

// EvaluateQuery runs a parsed query over an evaluation: resolve the table,
// apply WHERE, group, project the select list, then order and limit. Each
// stage is a helper so the pipeline stays readable in order.
func EvaluateQuery(query Query, evaluation ledger.Evaluation) (Result, error) {
	rows, err := rowsForTable(query.From, evaluation)
	if err != nil {
		return Result{}, err
	}
	rows, err = applyWhere(query.Where, rows)
	if err != nil {
		return Result{}, err
	}
	groups := groupRows(rows, query.GroupBy, query.Select)
	selectItems := expandStarSelect(rows, query.Select)
	result, err := projectGroups(selectItems, groups)
	if err != nil {
		return Result{}, err
	}
	sortResultRows(result, query.OrderBy)
	if query.Limit > 0 && len(result.Rows) > query.Limit {
		result.Rows = result.Rows[:query.Limit]
	}
	return result, nil
}

// applyWhere filters rows by the WHERE predicate, if present.
func applyWhere(where Expr, rows []Row) ([]Row, error) {
	if where == nil {
		return rows, nil
	}
	filtered := make([]Row, 0, len(rows))
	for _, row := range rows {
		value, err := where.eval(row, []Row{row})
		if err != nil {
			return nil, err
		}
		if truthy(value) {
			filtered = append(filtered, row)
		}
	}
	return filtered, nil
}

// expandStarSelect rewrites a lone `SELECT *` into one field item per column
// of the underlying rows, alphabetically so the output is deterministic.
func expandStarSelect(rows []Row, selectItems []SelectItem) []SelectItem {
	if len(selectItems) != 1 {
		return selectItems
	}
	if _, ok := selectItems[0].Expr.(starExpr); !ok {
		return selectItems
	}
	keys := make(map[string]struct{})
	for _, row := range rows {
		for key := range row {
			keys[key] = struct{}{}
		}
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	expanded := make([]SelectItem, 0, len(ordered))
	for _, key := range ordered {
		expanded = append(expanded, SelectItem{Expr: fieldExpr{name: key}, Alias: key})
	}
	return expanded
}

// projectGroups evaluates the select list once per group, naming each output
// column by its alias (or expression text when unaliased).
func projectGroups(selectItems []SelectItem, groups [][]Row) (Result, error) {
	result := Result{}
	for _, item := range selectItems {
		alias := item.Alias
		if alias == "" {
			alias = item.Expr.String()
		}
		result.Columns = append(result.Columns, alias)
	}
	for _, group := range groups {
		projected := Row{}
		for _, item := range selectItems {
			alias := item.Alias
			if alias == "" {
				alias = item.Expr.String()
			}
			value, err := item.Expr.eval(firstRow(group), group)
			if err != nil {
				return Result{}, err
			}
			projected[alias] = value
		}
		result.Rows = append(result.Rows, projected)
	}
	return result, nil
}

// sortResultRows applies the ORDER BY list; ties keep their prior (stable)
// order, and rows missing the sort key compare via formatValue.
func sortResultRows(result Result, orderBy []OrderItem) {
	if len(orderBy) == 0 {
		return
	}
	sort.SliceStable(result.Rows, func(i, j int) bool {
		left, right := result.Rows[i], result.Rows[j]
		for _, item := range orderBy {
			lv := left[item.Expr.String()]
			rv := right[item.Expr.String()]
			comparison := compareValues(lv, rv)
			if comparison == 0 {
				continue
			}
			if item.Desc {
				return comparison > 0
			}
			return comparison < 0
		}
		return false
	})
}

// groupRows partitions the joined rows for aggregation: no GROUP BY and no
// aggregate keeps every row its own group, no GROUP BY with aggregates makes
// one group, and otherwise rows group by their formatted key values.
func groupRows(rows []Row, groupBy []Expr, selectItems []SelectItem) [][]Row {
	aggregate := false
	for _, item := range selectItems {
		aggregate = aggregate || item.Expr.aggregate()
	}
	if len(groupBy) == 0 && !aggregate {
		groups := make([][]Row, 0, len(rows))
		for _, row := range rows {
			groups = append(groups, []Row{row})
		}
		return groups
	}
	if len(groupBy) == 0 {
		if len(rows) == 0 {
			return nil
		}
		return [][]Row{rows}
	}
	groups := make([][]Row, 0)
	indexes := make(map[string]int)
	for _, row := range rows {
		parts := make([]string, len(groupBy))
		for i, expr := range groupBy {
			value, _ := expr.eval(row, []Row{row})
			parts[i] = formatValue(value)
		}
		key := strings.Join(parts, "\x00")
		index, ok := indexes[key]
		if !ok {
			index = len(groups)
			indexes[key] = index
			groups = append(groups, nil)
		}
		groups[index] = append(groups[index], row)
	}
	return groups
}

func firstRow(rows []Row) Row {
	if len(rows) == 0 {
		return Row{}
	}
	return rows[0]
}

func rowsForTable(table string, evaluation ledger.Evaluation) ([]Row, error) {
	switch strings.ToLower(table) {
	case "postings", "posting":
		return postingRows(evaluation), nil
	case "entries", "entry", "directives":
		return entryRows(evaluation), nil
	case "accounts", "account":
		return accountRows(evaluation), nil
	case "prices", "price":
		return priceRows(evaluation), nil
	default:
		return nil, fmt.Errorf("unknown query table %q", table)
	}
}

func postingRows(evaluation ledger.Evaluation) []Row {
	rows := make([]Row, 0)
	for _, entry := range evaluation.Entries {
		var transaction *ledger.Transaction
		switch value := entry.Directive.(type) {
		case *ledger.Transaction:
			transaction = value
		case ledger.Transaction:
			copy := value
			transaction = &copy
		}
		if transaction == nil {
			continue
		}
		for _, posting := range transaction.Postings {
			// Journal directive filters use the transaction flag (the same flag
			// shown in Fava's journal). Preserve a separate posting_flag for
			// callers that need the lower-level posting marker.
			row := Row{"date": transaction.Date.Raw, "account": posting.Account, "flag": transaction.Flag, "posting_flag": posting.Flag, "narration": transaction.Narration, "payee": transaction.Payee, "tags": append([]string(nil), transaction.Tags...), "links": append([]string(nil), transaction.Links...), "file": entry.File, "span": entry.Span.String(), "kind": string(transaction.Kind())}
			if posting.Units != nil {
				row["currency"] = posting.Units.Currency
				row["units"] = ledger.DecimalFromNumber(posting.Units.Number)
				row["number"] = ledger.DecimalFromNumber(posting.Units.Number)
			}
			if posting.Cost != nil {
				row["cost"] = posting.Cost.Raw
			}
			rows = append(rows, row)
		}
	}
	return rows
}

func entryRows(evaluation ledger.Evaluation) []Row {
	rows := make([]Row, 0, len(evaluation.Entries))
	for _, entry := range evaluation.Entries {
		rows = append(rows, Row{"date": entry.Date.Raw, "kind": entry.Directive.Kind(), "span": entry.Span.String(), "file": entry.File})
	}
	return rows
}

func accountRows(evaluation ledger.Evaluation) []Row {
	accounts := make([]string, 0, len(evaluation.Accounts))
	for name := range evaluation.Accounts {
		accounts = append(accounts, name)
	}
	sort.Strings(accounts)
	rows := make([]Row, 0)
	for _, name := range accounts {
		account := evaluation.Accounts[name]
		currencies := make([]string, 0, len(account.Balances))
		for currency := range account.Balances {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)
		for _, currency := range currencies {
			rows = append(rows, Row{"account": name, "currency": currency, "balance": account.Balances[currency], "opened": account.Opened.Raw})
		}
		if len(currencies) == 0 {
			rows = append(rows, Row{"account": name, "currency": "", "balance": ledger.Zero(), "opened": account.Opened.Raw})
		}
	}
	return rows
}

func priceRows(evaluation ledger.Evaluation) []Row {
	bases := make([]string, 0, len(evaluation.Prices))
	for base := range evaluation.Prices {
		bases = append(bases, base)
	}
	sort.Strings(bases)
	rows := make([]Row, 0)
	for _, base := range bases {
		for _, quote := range evaluation.Prices[base] {
			rows = append(rows, Row{"date": quote.Date.Raw, "currency": quote.Base, "amount": quote.Amount, "quote_currency": quote.Currency})
		}
	}
	return rows
}

type queryTokenKind uint8

const (
	tokenWord queryTokenKind = iota
	tokenString
	tokenNumber
	tokenOperator
	tokenComma
	tokenLParen
	tokenRParen
	tokenStar
)

type queryToken struct {
	kind queryTokenKind
	text string
}

// lex splits a query string into words, numbers, strings, operators, and
// delimiters. Each token family has its own scanner; the loop only routes on
// the leading rune.
func lex(text string) []queryToken {
	tokens := make([]queryToken, 0)
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		if r == '\'' || r == '"' {
			var token queryToken
			token, i = lexString(text, i, byte(r), size)
			tokens = append(tokens, token)
			continue
		}
		if kind, ok := delimiterKind(r); ok {
			tokens = append(tokens, queryToken{kind: kind, text: string(r)})
			i += size
			continue
		}
		if strings.ContainsRune("=<>!+-/", r) {
			var token queryToken
			token, i = lexOperator(text, i, size)
			tokens = append(tokens, token)
			continue
		}
		var token queryToken
		token, i = lexWord(text, i)
		tokens = append(tokens, token)
	}
	return tokens
}

// delimiterKind maps one-rune delimiters to their token kinds.
func delimiterKind(r rune) (queryTokenKind, bool) {
	switch r {
	case ',':
		return tokenComma, true
	case '(':
		return tokenLParen, true
	case ')':
		return tokenRParen, true
	case '*':
		return tokenStar, true
	default:
		return tokenWord, false
	}
}

// lexString consumes a quoted string starting at text[i]; backslash escapes
// the next character, and an unterminated string ends at end-of-input.
func lexString(text string, i int, quote byte, size int) (queryToken, int) {
	i += size
	var value strings.Builder
	for i < len(text) && text[i] != quote {
		if text[i] == '\\' && i+1 < len(text) {
			i++
		}
		value.WriteByte(text[i])
		i++
	}
	if i < len(text) {
		i++
	}
	return queryToken{kind: tokenString, text: value.String()}, i
}

// lexOperator consumes an operator starting at text[i]: two-character forms
// (<=, >=, !=, <>) are glued, everything else is the single rune.
func lexOperator(text string, i int, size int) (queryToken, int) {
	start := i
	i += size
	if i < len(text) && (text[i] == '=' || (text[start] == '<' && text[i] == '>')) {
		i++
	}
	return queryToken{kind: tokenOperator, text: text[start:i]}, i
}

// lexWord consumes a bare word and classifies it as a number when it parses
// as one, mirroring how the primary parser treats numeric literals.
func lexWord(text string, i int) (queryToken, int) {
	start := i
	for i < len(text) {
		r, size := utf8.DecodeRuneInString(text[i:])
		if unicode.IsSpace(r) || strings.ContainsRune("(),*<>=!+-/\"", r) {
			break
		}
		i += size
	}
	word := text[start:i]
	kind := tokenWord
	if _, err := strconv.ParseFloat(word, 64); err == nil {
		kind = tokenNumber
	}
	return queryToken{kind: kind, text: word}, i
}

type queryParser struct {
	tokens []queryToken
	index  int
}

// parse reads one SELECT statement clause by clause. Each clause parser
// consumes its keywords so this loop only sequences them and verifies the
// statement is fully consumed at the end.
func (p *queryParser) parse() (Query, error) {
	query := Query{}
	if !p.acceptWord("select") {
		return query, p.errorf("query must start with SELECT")
	}
	if err := p.parseSelectList(&query); err != nil {
		return query, err
	}
	if len(query.Select) == 0 {
		return query, p.errorf("SELECT requires at least one expression")
	}
	if !p.acceptWord("from") {
		return query, p.errorf("query requires FROM")
	}
	from, err := p.expectWord("table name")
	if err != nil {
		return query, err
	}
	query.From = from
	if err := p.parseWhere(&query); err != nil {
		return query, err
	}
	if err := p.parseGroupBy(&query); err != nil {
		return query, err
	}
	if err := p.parseOrderBy(&query); err != nil {
		return query, err
	}
	if err := p.parseLimit(&query); err != nil {
		return query, err
	}
	if p.peek().kind != 0 || p.index < len(p.tokens) {
		return query, p.errorf("unexpected token %q", p.peek().text)
	}
	return query, nil
}

// parseSelectList consumes the comma-separated projection expressions. A
// bare word after an expression (not a clause keyword) is its implicit alias.
func (p *queryParser) parseSelectList(query *Query) error {
	for {
		if p.accept(tokenStar) {
			query.Select = append(query.Select, SelectItem{Expr: starExpr{}})
		} else {
			expr, err := p.expression(0)
			if err != nil {
				return err
			}
			item := SelectItem{Expr: expr}
			if p.acceptWord("as") {
				alias, aliasErr := p.expectWord("column alias")
				if aliasErr != nil {
					return aliasErr
				}
				item.Alias = alias
			} else if p.peek().kind == tokenWord && !isClauseWord(p.peek().text) {
				item.Alias = p.next().text
			}
			query.Select = append(query.Select, item)
		}
		if !p.accept(tokenComma) {
			return nil
		}
	}
}

// parseWhere consumes the optional WHERE predicate.
func (p *queryParser) parseWhere(query *Query) error {
	if !p.acceptWord("where") {
		return nil
	}
	where, err := p.expression(0)
	if err != nil {
		return err
	}
	query.Where = where
	return nil
}

// parseGroupBy consumes the optional GROUP BY expression list.
func (p *queryParser) parseGroupBy(query *Query) error {
	if !p.acceptWord("group") {
		return nil
	}
	if !p.acceptWord("by") {
		return p.errorf("GROUP requires BY")
	}
	groupBy, err := p.expressionList()
	if err != nil {
		return err
	}
	query.GroupBy = groupBy
	return nil
}

// parseOrderBy consumes the optional ORDER BY list with per-item ASC/DESC.
func (p *queryParser) parseOrderBy(query *Query) error {
	if !p.acceptWord("order") {
		return nil
	}
	if !p.acceptWord("by") {
		return p.errorf("ORDER requires BY")
	}
	for {
		expr, err := p.expression(0)
		if err != nil {
			return err
		}
		item := OrderItem{Expr: expr}
		if p.acceptWord("desc") {
			item.Desc = true
		} else {
			p.acceptWord("asc")
		}
		query.OrderBy = append(query.OrderBy, item)
		if !p.accept(tokenComma) {
			return nil
		}
	}
}

// parseLimit consumes the optional non-negative LIMIT n.
func (p *queryParser) parseLimit(query *Query) error {
	if !p.acceptWord("limit") {
		return nil
	}
	token := p.next()
	if token.kind != tokenNumber {
		return p.errorf("LIMIT requires a number")
	}
	limit, err := strconv.Atoi(token.text)
	if err != nil || limit < 0 {
		return p.errorf("invalid LIMIT")
	}
	query.Limit = limit
	return nil
}

func (p *queryParser) expression(minPrecedence int) (Expr, error) {
	left, err := p.primary()
	if err != nil {
		return nil, err
	}
	for {
		token := p.peek()
		precedence := operatorPrecedence(token)
		if precedence < minPrecedence {
			break
		}
		p.next()
		right, rightErr := p.expression(precedence + 1)
		if rightErr != nil {
			return nil, rightErr
		}
		left = binaryExpr{operator: strings.ToLower(token.text), left: left, right: right}
	}
	return left, nil
}

// primary parses one primary expression: literals, parenthesized groups,
// unary operators, function calls, and field references.
func (p *queryParser) primary() (Expr, error) {
	token := p.next()
	switch token.kind {
	case tokenString:
		return literalExpr{value: token.text, display: strconv.Quote(token.text)}, nil
	case tokenNumber:
		decimal, err := ledger.ParseDecimal(token.text)
		if err != nil {
			return nil, p.errorf("invalid number %q", token.text)
		}
		return literalExpr{value: decimal, display: token.text}, nil
	case tokenLParen:
		return p.parenthesized()
	case tokenOperator:
		if token.text == "-" {
			inner, err := p.primary()
			if err != nil {
				return nil, err
			}
			return unaryExpr{operator: "-", inner: inner}, nil
		}
		return nil, p.errorf("unexpected operator %q", token.text)
	case tokenWord:
		return p.wordPrimary(token)
	case tokenStar:
		return starExpr{}, nil
	default:
		return nil, p.errorf("unexpected token %q", token.text)
	}
}

// parenthesized parses "( expression )".
func (p *queryParser) parenthesized() (Expr, error) {
	expr, err := p.expression(0)
	if err != nil {
		return nil, err
	}
	if !p.accept(tokenRParen) {
		return nil, p.errorf("missing closing parenthesis")
	}
	return expr, nil
}

// wordPrimary parses a leading word: NOT, a function call, or a field name.
func (p *queryParser) wordPrimary(token queryToken) (Expr, error) {
	if strings.EqualFold(token.text, "not") {
		inner, err := p.primary()
		if err != nil {
			return nil, err
		}
		return unaryExpr{operator: "not", inner: inner}, nil
	}
	if p.accept(tokenLParen) {
		args, err := p.callArguments()
		if err != nil {
			return nil, err
		}
		return functionExpr{name: strings.ToLower(token.text), args: args}, nil
	}
	return fieldExpr{name: token.text}, nil
}

// callArguments parses a function argument list up to the closing
// parenthesis; a bare * inside the list (as in count(*)) is an argument.
func (p *queryParser) callArguments() ([]Expr, error) {
	args := make([]Expr, 0)
	if p.accept(tokenRParen) {
		return args, nil
	}
	for {
		if p.accept(tokenStar) {
			args = append(args, starExpr{})
		} else {
			expr, err := p.expression(0)
			if err != nil {
				return nil, err
			}
			args = append(args, expr)
		}
		if p.accept(tokenRParen) {
			return args, nil
		}
		if !p.accept(tokenComma) {
			return nil, p.errorf("function arguments require commas")
		}
	}
}

func (p *queryParser) expressionList() ([]Expr, error) {
	list := make([]Expr, 0)
	for {
		expr, err := p.expression(0)
		if err != nil {
			return nil, err
		}
		list = append(list, expr)
		if !p.accept(tokenComma) {
			return list, nil
		}
	}
}

func (p *queryParser) accept(kind queryTokenKind) bool {
	if p.peek().kind != kind {
		return false
	}
	p.index++
	return true
}

func (p *queryParser) acceptWord(value string) bool {
	if p.peek().kind != tokenWord || !strings.EqualFold(p.peek().text, value) {
		return false
	}
	p.index++
	return true
}

func (p *queryParser) expectWord(label string) (string, error) {
	token := p.next()
	if token.kind != tokenWord {
		return "", p.errorf("expected %s", label)
	}
	return token.text, nil
}

func (p *queryParser) peek() queryToken {
	if p.index >= len(p.tokens) {
		return queryToken{}
	}
	return p.tokens[p.index]
}

func (p *queryParser) next() queryToken {
	token := p.peek()
	if p.index < len(p.tokens) {
		p.index++
	}
	return token
}

func (p *queryParser) errorf(format string, args ...any) error {
	return ParseError{Message: fmt.Sprintf(format, args...)}
}

// operatorPrecedence ranks query operators for expression parsing:
// OR < AND < comparisons < + - < * /.
func operatorPrecedence(token queryToken) int {
	if token.kind == tokenStar {
		return 5
	}
	if token.kind == tokenWord {
		switch strings.ToLower(token.text) {
		case "or":
			return 1
		case "and":
			return 2
		}
	}
	if token.kind != tokenOperator {
		return -1
	}
	switch token.text {
	case "=", "!=", "<>", "<", "<=", ">", ">=":
		return 3
	case "+", "-":
		return 4
	case "*", "/":
		return 5
	default:
		return -1
	}
}

func isClauseWord(value string) bool {
	switch strings.ToLower(value) {
	case "from", "where", "group", "order", "limit", "asc", "desc", "by":
		return true
	default:
		return false
	}
}

type fieldExpr struct{ name string }

func (e fieldExpr) String() string  { return e.name }
func (e fieldExpr) aggregate() bool { return false }
func (e fieldExpr) eval(row Row, _ []Row) (any, error) {
	if value, ok := row[e.name]; ok {
		return value, nil
	}
	for key, value := range row {
		if strings.EqualFold(key, e.name) {
			return value, nil
		}
	}
	return nil, nil
}

type literalExpr struct {
	value   any
	display string
}

func (e literalExpr) String() string               { return e.display }
func (e literalExpr) aggregate() bool              { return false }
func (e literalExpr) eval(Row, []Row) (any, error) { return e.value, nil }

type starExpr struct{}

func (starExpr) String() string                     { return "*" }
func (starExpr) aggregate() bool                    { return false }
func (starExpr) eval(row Row, _ []Row) (any, error) { return row, nil }

type unaryExpr struct {
	operator string
	inner    Expr
}

func (e unaryExpr) String() string  { return e.operator + " " + e.inner.String() }
func (e unaryExpr) aggregate() bool { return e.inner.aggregate() }
func (e unaryExpr) eval(row Row, rows []Row) (any, error) {
	value, err := e.inner.eval(row, rows)
	if err != nil {
		return nil, err
	}
	if e.operator == "not" {
		return !truthy(value), nil
	}
	decimal, ok := decimalValue(value)
	if !ok {
		return nil, fmt.Errorf("unary minus requires a number")
	}
	return decimal.Neg(), nil
}

type binaryExpr struct {
	operator string
	left     Expr
	right    Expr
}

func (e binaryExpr) String() string {
	return e.left.String() + " " + e.operator + " " + e.right.String()
}
func (e binaryExpr) aggregate() bool { return e.left.aggregate() || e.right.aggregate() }

// eval evaluates a binary operator: boolean connectives, comparisons, and
// arithmetic. Arithmetic requires numbers on both sides; comparisons fall
// back to formatted string order for non-numeric operands.
func (e binaryExpr) eval(row Row, rows []Row) (any, error) {
	left, err := e.left.eval(row, rows)
	if err != nil {
		return nil, err
	}
	right, err := e.right.eval(row, rows)
	if err != nil {
		return nil, err
	}
	switch e.operator {
	case "and":
		return truthy(left) && truthy(right), nil
	case "or":
		return truthy(left) || truthy(right), nil
	default:
		return e.evalComparison(left, right)
	}
}

// evalComparison evaluates comparisons and arithmetic on already-evaluated
// operands.
func (e binaryExpr) evalComparison(left, right any) (any, error) {
	switch e.operator {
	case "=":
		return compareValues(left, right) == 0, nil
	case "!=", "<>":
		return compareValues(left, right) != 0, nil
	case "<":
		return compareValues(left, right) < 0, nil
	case "<=":
		return compareValues(left, right) <= 0, nil
	case ">":
		return compareValues(left, right) > 0, nil
	case ">=":
		return compareValues(left, right) >= 0, nil
	case "+", "-", "*", "/":
		return e.evalArithmetic(left, right)
	default:
		return nil, fmt.Errorf("unsupported operator %q", e.operator)
	}
}

// evalArithmetic applies +, -, *, / to numeric operands.
func (e binaryExpr) evalArithmetic(left, right any) (any, error) {
	leftNumber, leftOK := decimalValue(left)
	rightNumber, rightOK := decimalValue(right)
	if !leftOK || !rightOK {
		return nil, fmt.Errorf("arithmetic requires numbers")
	}
	switch e.operator {
	case "+":
		return leftNumber.Add(rightNumber), nil
	case "-":
		return leftNumber.Sub(rightNumber), nil
	case "*":
		return leftNumber.Mul(rightNumber), nil
	default: // "/"
		if rightNumber.IsZero() {
			return nil, fmt.Errorf("division by zero")
		}
		return ledger.NewDecimal(new(big.Rat).Quo(leftNumber.Rat(), rightNumber.Rat())), nil
	}
}

type functionExpr struct {
	name string
	args []Expr
}

func (e functionExpr) String() string {
	args := make([]string, len(e.args))
	for i, arg := range e.args {
		args[i] = arg.String()
	}
	return e.name + "(" + strings.Join(args, ", ") + ")"
}

func (e functionExpr) aggregate() bool {
	switch e.name {
	case "sum", "count", "min", "max", "first", "last", "avg":
		return true
	default:
		for _, arg := range e.args {
			if arg.aggregate() {
				return true
			}
		}
		return false
	}
}

// eval dispatches a function call to its evaluation family. Counting and the
// numeric aggregates fold over the group's rows; date parts and membership
// predicates evaluate their single argument against the current row.
func (e functionExpr) eval(row Row, rows []Row) (any, error) {
	switch e.name {
	case "count":
		return e.evalCount(rows)
	case "sum", "avg", "min", "max", "first", "last":
		return e.evalAggregate(row, rows)
	case "year", "month", "day":
		return e.evalDatePart(row, rows)
	case "has_tag", "has_link":
		return e.evalMembership(row, rows)
	default:
		return nil, fmt.Errorf("unsupported function %q", e.name)
	}
}

// evalCount implements count() and count(expr): the bare form counts rows,
// the argument form counts rows where the argument is non-NULL.
func (e functionExpr) evalCount(rows []Row) (any, error) {
	if len(e.args) == 0 || (len(e.args) == 1 && e.args[0].String() == "*") {
		return ledger.NewDecimal(big.NewRat(int64(len(rows)), 1)), nil
	}
	count := 0
	for _, candidate := range rows {
		value, err := e.args[0].eval(candidate, []Row{candidate})
		if err != nil {
			return nil, err
		}
		if value != nil {
			count++
		}
	}
	return ledger.NewDecimal(big.NewRat(int64(count), 1)), nil
}

// evalAggregate implements sum/avg/min/max/first/last over the group's rows;
// rows where the argument is missing or non-numeric are skipped. An empty
// input yields 0 (matching the workbench's deterministic presentation).
func (e functionExpr) evalAggregate(row Row, rows []Row) (any, error) {
	if len(e.args) != 1 {
		return nil, fmt.Errorf("%s requires one argument", e.name)
	}
	values := make([]ledger.Decimal, 0, len(rows))
	for _, candidate := range rows {
		value, err := e.args[0].eval(candidate, []Row{candidate})
		if err != nil {
			return nil, err
		}
		if decimal, ok := decimalValue(value); ok {
			values = append(values, decimal)
		}
	}
	if len(values) == 0 {
		return ledger.Zero(), nil
	}
	switch e.name {
	case "sum":
		return sumDecimals(values), nil
	case "avg":
		sum, _ := functionExpr{name: "sum", args: e.args}.eval(row, rows)
		return ledger.NewDecimal(new(big.Rat).Quo(sum.(ledger.Decimal).Rat(), big.NewRat(int64(len(values)), 1))), nil
	case "min", "max":
		return extremalDecimal(values, e.name == "min"), nil
	case "first":
		return values[0], nil
	default: // "last"
		return values[len(values)-1], nil
	}
}

// evalDatePart implements year()/month()/day() over a date (or date string).
func (e functionExpr) evalDatePart(row Row, rows []Row) (any, error) {
	if len(e.args) != 1 {
		return nil, fmt.Errorf("%s requires one argument", e.name)
	}
	value, err := e.args[0].eval(row, rows)
	if err != nil {
		return nil, err
	}
	date, ok := value.(ledger.Date)
	if !ok {
		if raw, isString := value.(string); isString {
			date = parseDate(raw)
		}
	}
	var part int
	switch e.name {
	case "year":
		part = date.Year
	case "month":
		part = date.Month
	default: // "day"
		part = date.Day
	}
	return ledger.NewDecimal(big.NewRat(int64(part), 1)), nil
}

// evalMembership implements has_tag/has_link: exact membership of a value in
// a collection such as the tags or links column.
func (e functionExpr) evalMembership(row Row, rows []Row) (any, error) {
	if len(e.args) != 2 {
		return nil, fmt.Errorf("%s requires a collection and value", e.name)
	}
	collection, err := e.args[0].eval(row, rows)
	if err != nil {
		return nil, err
	}
	needle, err := e.args[1].eval(row, rows)
	if err != nil {
		return nil, err
	}
	needleString := formatValue(needle)
	if values, ok := collection.([]string); ok {
		for _, value := range values {
			if value == needleString {
				return true, nil
			}
		}
	}
	return false, nil
}

// sumDecimals adds a list of decimals.
func sumDecimals(values []ledger.Decimal) ledger.Decimal {
	result := ledger.Zero()
	for _, value := range values {
		result = result.Add(value)
	}
	return result
}

// extremalDecimal picks the minimum or maximum of a non-empty list.
func extremalDecimal(values []ledger.Decimal, min bool) ledger.Decimal {
	result := values[0]
	for _, value := range values[1:] {
		if (min && value.Cmp(result) < 0) || (!min && value.Cmp(result) > 0) {
			result = value
		}
	}
	return result
}

func decimalValue(value any) (ledger.Decimal, bool) {
	switch typed := value.(type) {
	case ledger.Decimal:
		return typed, true
	case ledger.Number:
		return ledger.DecimalFromNumber(typed), true
	case int:
		return ledger.NewDecimal(big.NewRat(int64(typed), 1)), true
	case int64:
		return ledger.NewDecimal(big.NewRat(typed, 1)), true
	case float64:
		decimal, err := ledger.ParseDecimal(strconv.FormatFloat(typed, 'f', -1, 64))
		return decimal, err == nil
	case string:
		decimal, err := ledger.ParseDecimal(typed)
		return decimal, err == nil
	default:
		return ledger.Zero(), false
	}
}

func truthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case ledger.Decimal:
		return !typed.IsZero()
	case string:
		return typed != ""
	default:
		return true
	}
}

func compareValues(left, right any) int {
	if leftNumber, ok := decimalValue(left); ok {
		if rightNumber, ok := decimalValue(right); ok {
			return leftNumber.Cmp(rightNumber)
		}
	}
	leftString, rightString := formatValue(left), formatValue(right)
	if leftString < rightString {
		return -1
	}
	if leftString > rightString {
		return 1
	}
	return 0
}

func formatValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case ledger.Decimal:
		return typed.String()
	case ledger.Date:
		return typed.Raw
	case ledger.DirectiveKind:
		return string(typed)
	case []string:
		return strings.Join(typed, ",")
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprint(typed)
	}
}

func parseDate(raw string) ledger.Date {
	parts := strings.Split(raw, "-")
	if len(parts) != 3 {
		return ledger.Date{Raw: raw}
	}
	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])
	return ledger.Date{Year: year, Month: month, Day: day, Raw: raw}
}
