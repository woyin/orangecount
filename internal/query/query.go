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

type Row map[string]any

type Result struct {
	Columns []string `json:"columns"`
	Rows    []Row    `json:"rows"`
}

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

type Query struct {
	Select  []SelectItem
	From    string
	Where   Expr
	GroupBy []Expr
	OrderBy []OrderItem
	Limit   int
}

type SelectItem struct {
	Expr  Expr
	Alias string
}

type OrderItem struct {
	Expr Expr
	Desc bool
}

type Expr interface {
	String() string
	eval(row Row, rows []Row) (any, error)
	aggregate() bool
}

type ParseError struct{ Message string }

func (e ParseError) Error() string { return e.Message }

func Parse(text string) (Query, error) {
	parser := &queryParser{tokens: lex(text)}
	return parser.parse()
}

func Evaluate(text string, evaluation ledger.Evaluation) (Result, error) {
	query, err := Parse(text)
	if err != nil {
		return Result{}, err
	}
	return EvaluateQuery(query, evaluation)
}

func EvaluateQuery(query Query, evaluation ledger.Evaluation) (Result, error) {
	rows, err := rowsForTable(query.From, evaluation)
	if err != nil {
		return Result{}, err
	}
	if query.Where != nil {
		filtered := make([]Row, 0, len(rows))
		for _, row := range rows {
			value, evalErr := query.Where.eval(row, []Row{row})
			if evalErr != nil {
				return Result{}, evalErr
			}
			if truthy(value) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	groups := groupRows(rows, query.GroupBy, query.Select)
	selectItems := query.Select
	if len(selectItems) == 1 {
		if _, ok := selectItems[0].Expr.(starExpr); ok {
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
			selectItems = make([]SelectItem, 0, len(ordered))
			for _, key := range ordered {
				selectItems = append(selectItems, SelectItem{Expr: fieldExpr{name: key}, Alias: key})
			}
		}
	}
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
			value, evalErr := item.Expr.eval(firstRow(group), group)
			if evalErr != nil {
				return Result{}, evalErr
			}
			projected[alias] = value
		}
		result.Rows = append(result.Rows, projected)
	}
	if len(query.OrderBy) != 0 {
		sort.SliceStable(result.Rows, func(i, j int) bool {
			left, right := result.Rows[i], result.Rows[j]
			for _, item := range query.OrderBy {
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
	if query.Limit > 0 && len(result.Rows) > query.Limit {
		result.Rows = result.Rows[:query.Limit]
	}
	return result, nil
}

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
			row := Row{"date": transaction.Date.Raw, "account": posting.Account, "flag": transaction.Flag, "posting_flag": posting.Flag, "narration": transaction.Narration, "payee": transaction.Payee, "tags": append([]string(nil), transaction.Tags...), "links": append([]string(nil), transaction.Links...), "file": entry.File, "span": entry.Span.String()}
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

func lex(text string) []queryToken {
	tokens := make([]queryToken, 0)
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		if r == '\'' || r == '"' {
			quote := byte(r)
			i += size
			start := i
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
			if value.Len() == 0 && start < i-1 {
				value.WriteString("")
			}
			tokens = append(tokens, queryToken{kind: tokenString, text: value.String()})
			continue
		}
		if strings.ContainsRune("(),", r) {
			kind := tokenComma
			if r == '(' {
				kind = tokenLParen
			} else if r == ')' {
				kind = tokenRParen
			}
			tokens = append(tokens, queryToken{kind: kind, text: string(r)})
			i += size
			continue
		}
		if r == '*' {
			tokens = append(tokens, queryToken{kind: tokenStar, text: "*"})
			i += size
			continue
		}
		if strings.ContainsRune("=<>!+-/", r) {
			start := i
			i += size
			if i < len(text) && (text[i] == '=' || (text[start] == '<' && text[i] == '>')) {
				i++
			}
			tokens = append(tokens, queryToken{kind: tokenOperator, text: text[start:i]})
			continue
		}
		start := i
		for i < len(text) {
			r, size = utf8.DecodeRuneInString(text[i:])
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
		tokens = append(tokens, queryToken{kind: kind, text: word})
	}
	return tokens
}

type queryParser struct {
	tokens []queryToken
	index  int
}

func (p *queryParser) parse() (Query, error) {
	query := Query{}
	if !p.acceptWord("select") {
		return query, p.errorf("query must start with SELECT")
	}
	for {
		if p.accept(tokenStar) {
			query.Select = append(query.Select, SelectItem{Expr: starExpr{}})
		} else {
			expr, err := p.expression(0)
			if err != nil {
				return query, err
			}
			item := SelectItem{Expr: expr}
			if p.acceptWord("as") {
				alias, err := p.expectWord("column alias")
				if err != nil {
					return query, err
				}
				item.Alias = alias
			} else if p.peek().kind == tokenWord && !isClauseWord(p.peek().text) {
				item.Alias = p.next().text
			}
			query.Select = append(query.Select, item)
		}
		if !p.accept(tokenComma) {
			break
		}
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
	if p.acceptWord("where") {
		query.Where, err = p.expression(0)
		if err != nil {
			return query, err
		}
	}
	if p.acceptWord("group") {
		if !p.acceptWord("by") {
			return query, p.errorf("GROUP requires BY")
		}
		query.GroupBy, err = p.expressionList()
		if err != nil {
			return query, err
		}
	}
	if p.acceptWord("order") {
		if !p.acceptWord("by") {
			return query, p.errorf("ORDER requires BY")
		}
		for {
			expr, parseErr := p.expression(0)
			if parseErr != nil {
				return query, parseErr
			}
			item := OrderItem{Expr: expr}
			if p.acceptWord("desc") {
				item.Desc = true
			} else {
				p.acceptWord("asc")
			}
			query.OrderBy = append(query.OrderBy, item)
			if !p.accept(tokenComma) {
				break
			}
		}
	}
	if p.acceptWord("limit") {
		token := p.next()
		if token.kind != tokenNumber {
			return query, p.errorf("LIMIT requires a number")
		}
		limit, parseErr := strconv.Atoi(token.text)
		if parseErr != nil || limit < 0 {
			return query, p.errorf("invalid LIMIT")
		}
		query.Limit = limit
	}
	if p.peek().kind != 0 || p.index < len(p.tokens) {
		return query, p.errorf("unexpected token %q", p.peek().text)
	}
	return query, nil
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
		expr, err := p.expression(0)
		if err != nil {
			return nil, err
		}
		if !p.accept(tokenRParen) {
			return nil, p.errorf("missing closing parenthesis")
		}
		return expr, nil
	case tokenOperator:
		if token.text == "-" {
			expr, err := p.primary()
			if err != nil {
				return nil, err
			}
			return unaryExpr{operator: "-", inner: expr}, nil
		}
		return nil, p.errorf("unexpected operator %q", token.text)
	case tokenWord:
		if strings.EqualFold(token.text, "not") {
			expr, err := p.primary()
			if err != nil {
				return nil, err
			}
			return unaryExpr{operator: "not", inner: expr}, nil
		}
		if p.accept(tokenLParen) {
			args := make([]Expr, 0)
			if !p.accept(tokenRParen) {
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
						break
					}
					if !p.accept(tokenComma) {
						return nil, p.errorf("function arguments require commas")
					}
				}
			}
			return functionExpr{name: strings.ToLower(token.text), args: args}, nil
		}
		return fieldExpr{name: token.text}, nil
	case tokenStar:
		return starExpr{}, nil
	default:
		return nil, p.errorf("unexpected token %q", token.text)
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
		default:
			if rightNumber.IsZero() {
				return nil, fmt.Errorf("division by zero")
			}
			return ledger.NewDecimal(new(big.Rat).Quo(leftNumber.Rat(), rightNumber.Rat())), nil
		}
	default:
		return nil, fmt.Errorf("unsupported operator %q", e.operator)
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

func (e functionExpr) eval(row Row, rows []Row) (any, error) {
	switch e.name {
	case "count":
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
	case "sum", "avg", "min", "max", "first", "last":
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
			result := ledger.Zero()
			for _, value := range values {
				result = result.Add(value)
			}
			return result, nil
		case "avg":
			sum, _ := functionExpr{name: "sum", args: e.args}.eval(row, rows)
			return ledger.NewDecimal(new(big.Rat).Quo(sum.(ledger.Decimal).Rat(), big.NewRat(int64(len(values)), 1))), nil
		case "min", "max":
			result := values[0]
			for _, value := range values[1:] {
				if (e.name == "min" && value.Cmp(result) < 0) || (e.name == "max" && value.Cmp(result) > 0) {
					result = value
				}
			}
			return result, nil
		case "first":
			return values[0], nil
		default:
			return values[len(values)-1], nil
		}
	case "year", "month", "day":
		if len(e.args) != 1 {
			return nil, fmt.Errorf("%s requires one argument", e.name)
		}
		value, err := e.args[0].eval(row, rows)
		if err != nil {
			return nil, err
		}
		date, ok := value.(ledger.Date)
		if !ok {
			if raw, ok := value.(string); ok {
				date = parseDate(raw)
			}
		}
		switch e.name {
		case "year":
			return ledger.NewDecimal(big.NewRat(int64(date.Year), 1)), nil
		case "month":
			return ledger.NewDecimal(big.NewRat(int64(date.Month), 1)), nil
		default:
			return ledger.NewDecimal(big.NewRat(int64(date.Day), 1)), nil
		}
	case "has_tag", "has_link":
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
	default:
		return nil, fmt.Errorf("unsupported function %q", e.name)
	}
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
