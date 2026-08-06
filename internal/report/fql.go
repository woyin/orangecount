// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package report

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"orangecount/internal/query"
)

// FQL is a parsed filter query in Fava's filter syntax: #tag and ^link
// membership, bare or quoted strings matched against narration/payee/comment,
// key:"value" regex matches, key=number amount comparisons, posting amount
// comparisons (>10), all(...)/any(...) posting quantifiers, and , / - /
// juxtaposition for or / not / and. The grammar mirrors the pinned Fava
// reference's FilterSyntaxLexer/FilterSyntaxParser.
type FQL struct {
	root *fqlNode
}

// FQLTarget is the entry-shaped view an FQL predicate evaluates against.
// Fields that do not apply to an entry stay zero; tag/link/posting terms then
// simply do not match, mirroring Fava's getattr-based semantics.
type FQLTarget struct {
	Tags      []string
	Links     []string
	Payee     string
	Narration string
	Comment   string
	Account   string
	Flag      string
	Date      string
	Metadata  map[string]string
	// Amount carries an entry-level amount such as a balance assertion's
	// amount; it backs number=… comparisons.
	Amount   *big.Float
	Postings []FQLPosting
}

type FQLPosting struct {
	Account string
	Units   *big.Float
}

// Match reports whether the filter selects the target entry.
func (f *FQL) Match(target FQLTarget) bool {
	if f == nil || f.root == nil {
		return true
	}
	return f.root.match(target)
}

type fqlNode struct {
	kind     string // tag, link, text, key, keyNumber, postingNumber, and, or, not, all, any
	value    string
	operator string
	number   *big.Float
	matcher  func(string) bool
	left     *fqlNode
	right    *fqlNode
	child    *fqlNode
}

func (node *fqlNode) match(target FQLTarget) bool {
	switch node.kind {
	case "tag":
		return containsExact(target.Tags, node.value)
	case "link":
		return containsExact(target.Links, node.value)
	case "text":
		for _, value := range []string{target.Narration, target.Payee, target.Comment} {
			if value != "" && node.matcher(value) {
				return true
			}
		}
		return false
	case "key":
		if value, ok := target.field(node.value); ok {
			return node.matcher(value)
		}
		return false
	case "keyNumber":
		// Fava compares the absolute value of an amount-like attribute; the
		// only entry-level amount this projection carries is number.
		if node.value == "number" && target.Amount != nil {
			return compareAbs(node.operator, target.Amount, node.number)
		}
		return false
	case "postingNumber":
		for _, posting := range target.Postings {
			if posting.Units != nil && compareAbs(node.operator, posting.Units, node.number) {
				return true
			}
		}
		return false
	case "and":
		return node.left.match(target) && node.right.match(target)
	case "or":
		return node.left.match(target) || node.right.match(target)
	case "not":
		return !node.child.match(target)
	case "all":
		for _, posting := range target.Postings {
			if !node.child.match(postingTarget(posting)) {
				return false
			}
		}
		return true
	case "any":
		for _, posting := range target.Postings {
			if node.child.match(postingTarget(posting)) {
				return true
			}
		}
		return false
	}
	return false
}

func postingTarget(posting FQLPosting) FQLTarget {
	return FQLTarget{Account: posting.Account}
}

func (target FQLTarget) field(name string) (string, bool) {
	switch name {
	case "payee":
		return target.Payee, true
	case "narration":
		return target.Narration, true
	case "comment":
		return target.Comment, true
	case "account":
		return target.Account, true
	case "flag":
		return target.Flag, true
	case "date":
		return target.Date, true
	}
	if target.Metadata != nil {
		if value, ok := target.Metadata[name]; ok {
			return value, true
		}
	}
	return "", false
}

func containsExact(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func compareAbs(operator string, left *big.Float, right *big.Float) bool {
	magnitude := new(big.Float).Abs(left)
	switch operator {
	case "=":
		return magnitude.Cmp(right) == 0
	case ">=":
		return magnitude.Cmp(right) >= 0
	case "<=":
		return magnitude.Cmp(right) <= 0
	case ">":
		return magnitude.Cmp(right) > 0
	default: // "<"
		return magnitude.Cmp(right) < 0
	}
}

// fqlStringMatcher mirrors Fava's Match helper: a case-insensitive regex
// search, degrading to exact equality when the pattern is not a valid regex.
// Go's RE2 rejects some Python patterns (e.g. backreferences); those take the
// equality fallback too.
func fqlStringMatcher(pattern string) func(string) bool {
	compiled, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return func(value string) bool { return value == pattern }
	}
	return func(value string) bool { return compiled.MatchString(value) }
}

type fqlToken struct {
	kind  string // link, tag, all, any, key, eq, cmp, number, string, literal
	value string
}

var (
	fqlReLink   = regexp.MustCompile(`\A\^[A-Za-z0-9\-_/.]+`)
	fqlReTag    = regexp.MustCompile(`\A#[A-Za-z0-9\-_/.]+`)
	fqlReAll    = regexp.MustCompile(`\Aall\(`)
	fqlReAny    = regexp.MustCompile(`\Aany\(`)
	fqlReWord   = regexp.MustCompile(`\A[a-z][a-zA-Z0-9\-_]+`)
	fqlReCmp    = regexp.MustCompile(`\A(>=|<=|=|<|>)`)
	fqlReNumber = regexp.MustCompile(`\A\d*\.?\d+`)
	fqlReString = regexp.MustCompile(`\A(\w[-\w]*|"[^"]*"|'[^']*')`)
)

// fqlKeyAt reports whether rest starts a KEY token: a lowercase-led word run
// of at least two characters followed (after optional blanks) by an operator.
// Fava expresses this with a lookahead, which RE2 does not support.
func fqlKeyAt(text string, position int) string {
	match := fqlReWord.FindString(text[position:])
	if match == "" {
		return ""
	}
	lookahead := position + len(match)
	for lookahead < len(text) && (text[lookahead] == ' ' || text[lookahead] == '\t') {
		lookahead++
	}
	if lookahead < len(text) {
		switch text[lookahead] {
		case ':', '=', '<', '>':
			return match
		}
	}
	return ""
}

func fqlLex(text string) ([]fqlToken, error) {
	var tokens []fqlToken
	for position := 0; position < len(text); {
		char := text[position]
		if char == ' ' || char == '\t' {
			position++
			continue
		}
		rest := text[position:]
		switch {
		case strings.HasPrefix(rest, "^"):
			match := fqlReLink.FindString(rest)
			if match == "" {
				return nil, fmt.Errorf("failed to parse filter: %s", text)
			}
			tokens = append(tokens, fqlToken{kind: "link", value: match[1:]})
			position += len(match)
		case strings.HasPrefix(rest, "#"):
			match := fqlReTag.FindString(rest)
			if match == "" {
				return nil, fmt.Errorf("failed to parse filter: %s", text)
			}
			tokens = append(tokens, fqlToken{kind: "tag", value: match[1:]})
			position += len(match)
		default:
			switch match := fqlReAll.FindString(rest); {
			case match != "":
				tokens = append(tokens, fqlToken{kind: "all"})
				position += len(match)
				continue
			}
			if match := fqlReAny.FindString(rest); match != "" {
				tokens = append(tokens, fqlToken{kind: "any"})
				position += len(match)
				continue
			}
			if match := fqlKeyAt(text, position); match != "" {
				tokens = append(tokens, fqlToken{kind: "key", value: match})
				position += len(match)
				continue
			}
			if strings.HasPrefix(rest, ":") {
				tokens = append(tokens, fqlToken{kind: "eq"})
				position++
				continue
			}
			if match := fqlReCmp.FindString(rest); match != "" {
				tokens = append(tokens, fqlToken{kind: "cmp", value: match})
				position += len(match)
				continue
			}
			if match := fqlReNumber.FindString(rest); match != "" {
				tokens = append(tokens, fqlToken{kind: "number", value: match})
				position += len(match)
				continue
			}
			if match := fqlReString.FindString(rest); match != "" {
				value := match
				if strings.HasPrefix(match, `"`) || strings.HasPrefix(match, "'") {
					value = match[1 : len(match)-1]
				}
				tokens = append(tokens, fqlToken{kind: "string", value: value})
				position += len(match)
				continue
			}
			if char == '-' || char == ',' || char == '(' || char == ')' {
				tokens = append(tokens, fqlToken{kind: "literal", value: string(char)})
				position++
				continue
			}
			return nil, fmt.Errorf("illegal character %q in filter", string(char))
		}
	}
	return tokens, nil
}

type fqlParser struct {
	tokens []fqlToken
	pos    int
	text   string
}

// ParseFQL parses a Fava-style filter query. An empty input yields a nil
// filter that matches everything. Parse failures describe the problem the way
// Fava's FilterError types do, so the message can be shown verbatim.
func ParseFQL(text string) (*FQL, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	tokens, err := fqlLex(text)
	if err != nil {
		return nil, err
	}
	parser := &fqlParser{tokens: tokens, text: text}
	root, err := parser.parseOr()
	if err != nil {
		return nil, err
	}
	if parser.pos != len(parser.tokens) {
		return nil, fmt.Errorf("failed to parse filter: %s", text)
	}
	return &FQL{root: root}, nil
}

func (parser *fqlParser) peek() *fqlToken {
	if parser.pos >= len(parser.tokens) {
		return nil
	}
	return &parser.tokens[parser.pos]
}

func (parser *fqlParser) parseError() error {
	return fmt.Errorf("failed to parse filter: %s", parser.text)
}

func (parser *fqlParser) parseOr() (*fqlNode, error) {
	left, err := parser.parseAnd()
	if err != nil {
		return nil, err
	}
	for token := parser.peek(); token != nil && token.kind == "literal" && token.value == ","; token = parser.peek() {
		parser.pos++
		right, err := parser.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &fqlNode{kind: "or", left: left, right: right}
	}
	return left, nil
}

func (parser *fqlParser) parseAnd() (*fqlNode, error) {
	left, err := parser.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		token := parser.peek()
		if token == nil || (token.kind == "literal" && (token.value == "," || token.value == ")")) {
			return left, nil
		}
		right, err := parser.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &fqlNode{kind: "and", left: left, right: right}
	}
}

func (parser *fqlParser) parseUnary() (*fqlNode, error) {
	token := parser.peek()
	if token != nil && token.kind == "literal" && token.value == "-" {
		parser.pos++
		child, err := parser.parseUnary()
		if err != nil {
			return nil, err
		}
		return &fqlNode{kind: "not", child: child}, nil
	}
	return parser.parsePrimary()
}

func (parser *fqlParser) parsePrimary() (*fqlNode, error) {
	token := parser.peek()
	if token == nil {
		return nil, parser.parseError()
	}
	switch token.kind {
	case "literal":
		if token.value != "(" {
			return nil, parser.parseError()
		}
		parser.pos++
		child, err := parser.parseOr()
		if err != nil {
			return nil, err
		}
		return child, parser.expectCloseParen()
	case "all", "any":
		parser.pos++
		child, err := parser.parseOr()
		if err != nil {
			return nil, err
		}
		return &fqlNode{kind: token.kind, child: child}, parser.expectCloseParen()
	case "tag":
		parser.pos++
		return &fqlNode{kind: "tag", value: token.value}, nil
	case "link":
		parser.pos++
		return &fqlNode{kind: "link", value: token.value}, nil
	case "string":
		parser.pos++
		return &fqlNode{kind: "text", matcher: fqlStringMatcher(token.value)}, nil
	case "key":
		parser.pos++
		return parser.parseKeyTail(token.value)
	case "cmp":
		parser.pos++
		number, err := parser.parseNumber()
		if err != nil {
			return nil, err
		}
		return &fqlNode{kind: "postingNumber", operator: token.value, number: number}, nil
	}
	return nil, parser.parseError()
}

func (parser *fqlParser) parseKeyTail(key string) (*fqlNode, error) {
	token := parser.peek()
	if token == nil {
		return nil, parser.parseError()
	}
	if token.kind == "eq" {
		parser.pos++
		value := parser.peek()
		if value != nil && value.kind == "string" {
			parser.pos++
			return &fqlNode{kind: "key", value: key, matcher: fqlStringMatcher(value.value)}, nil
		}
		return nil, parser.parseError()
	}
	if token.kind == "cmp" {
		parser.pos++
		number, err := parser.parseNumber()
		if err != nil {
			return nil, err
		}
		return &fqlNode{kind: "keyNumber", value: key, operator: token.value, number: number}, nil
	}
	return nil, parser.parseError()
}

func (parser *fqlParser) parseNumber() (*big.Float, error) {
	token := parser.peek()
	if token == nil || token.kind != "number" {
		return nil, parser.parseError()
	}
	parser.pos++
	number, _, err := big.ParseFloat(token.value, 10, 128, big.ToNearestEven)
	if err != nil {
		return nil, parser.parseError()
	}
	return number, nil
}

func (parser *fqlParser) expectCloseParen() error {
	token := parser.peek()
	if token == nil || token.kind != "literal" || token.value != ")" {
		return parser.parseError()
	}
	parser.pos++
	return nil
}

// FQLTargetFromRow maps a table row onto the entry-shaped view FQL evaluates,
// using the row's own columns as metadata so key:value terms can reach any
// column present. Rows are derived values, so posting- and tag-shaped terms
// simply do not match them.
func FQLTargetFromRow(row query.Row) FQLTarget {
	target := FQLTarget{Metadata: map[string]string{}}
	for column, value := range row {
		if text, ok := value.(string); ok {
			target.Metadata[column] = text
		}
	}
	target.Payee = rowString(row, "payee")
	target.Narration = rowString(row, "narration")
	target.Comment = rowString(row, "comment")
	target.Account = rowString(row, "account")
	target.Flag = rowString(row, "flag")
	target.Date = rowString(row, "date")
	return target
}

func rowString(row query.Row, key string) string {
	value, _ := row[key].(string)
	return value
}
