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

// FQLPosting carries the posting fields an FQL predicate can match: its
// account and a float view of its units (display-oriented matching).
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

// match evaluates the predicate tree against one target: the boolean
// combinators recurse, the posting quantifiers delegate to matchQuantified,
// and every leaf shape to matchTerm.
func (node *fqlNode) match(target FQLTarget) bool {
	switch node.kind {
	case "and":
		return node.left.match(target) && node.right.match(target)
	case "or":
		return node.left.match(target) || node.right.match(target)
	case "not":
		return !node.child.match(target)
	case "all", "any":
		return node.matchQuantified(target)
	default:
		return node.matchTerm(target)
	}
}

// matchTerm evaluates one leaf predicate against the entry-shaped target.
func (node *fqlNode) matchTerm(target FQLTarget) bool {
	switch node.kind {
	case "tag":
		return containsExact(target.Tags, node.value)
	case "link":
		return containsExact(target.Links, node.value)
	case "text":
		return node.matchText(target)
	case "key":
		if value, ok := target.field(node.value); ok {
			return node.matcher(value)
		}
		return false
	case "keyNumber":
		// Fava compares the absolute value of an amount-like attribute; the
		// only entry-level amount this projection carries is number.
		return node.value == "number" && target.Amount != nil && compareAbs(node.operator, target.Amount, node.number)
	case "postingNumber":
		for _, posting := range target.Postings {
			if posting.Units != nil && compareAbs(node.operator, posting.Units, node.number) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// matchText applies the text pattern to the entry's free-text fields,
// skipping empty ones the way Fava's getattr walk does.
func (node *fqlNode) matchText(target FQLTarget) bool {
	for _, value := range []string{target.Narration, target.Payee, target.Comment} {
		if value != "" && node.matcher(value) {
			return true
		}
	}
	return false
}

// matchQuantified evaluates all(...)/any(...) over the target's postings:
// all requires every posting to match, any at least one. Quantifiers over
// the (empty) posting list are vacuously true for all, false for any.
func (node *fqlNode) matchQuantified(target FQLTarget) bool {
	for _, posting := range target.Postings {
		matched := node.child.match(postingTarget(posting))
		if node.kind == "all" && !matched {
			return false
		}
		if node.kind == "any" && matched {
			return true
		}
	}
	return node.kind == "all"
}

func postingTarget(posting FQLPosting) FQLTarget {
	return FQLTarget{Account: posting.Account}
}

// field resolves a filter field name against the target's text attributes
// and metadata; unknown names report ok=false.
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

// fqlLex splits a filter query into tokens. Sigil-led tokens (tags, links)
// and the default token ladder are delegated so the loop only advances.
func fqlLex(text string) ([]fqlToken, error) {
	var tokens []fqlToken
	for position := 0; position < len(text); {
		char := text[position]
		if char == ' ' || char == '\t' {
			position++
			continue
		}
		if char == '^' || char == '#' {
			token, next, err := fqlLexSigil(text, position)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token)
			position = next
			continue
		}
		token, next, ok := fqlLexDefault(text, position)
		if !ok {
			return nil, fmt.Errorf("illegal character %q in filter", string(char))
		}
		tokens = append(tokens, token)
		position = next
	}
	return tokens, nil
}

// fqlLexSigil lexes a ^link or #tag starting at position. A sigil with no
// following word characters is a parse error, mirroring Fava's lexer.
func fqlLexSigil(text string, position int) (fqlToken, int, error) {
	match := fqlReTag.FindString(text[position:])
	kind := "tag"
	if text[position] == '^' {
		match = fqlReLink.FindString(text[position:])
		kind = "link"
	}
	if match == "" {
		return fqlToken{}, position, fmt.Errorf("failed to parse filter: %s", text)
	}
	return fqlToken{kind: kind, value: match[1:]}, position + len(match), nil
}

// fqlLexDefault tries the remaining token shapes in Fava's lexer order —
// all(/any( quantifiers, lookahead-detected keys, ":"/comparators, numbers,
// strings, and the literal punctuation — and reports whether one matched.
func fqlLexDefault(text string, position int) (fqlToken, int, bool) {
	rest := text[position:]
	if match := fqlReAll.FindString(rest); match != "" {
		return fqlToken{kind: "all"}, position + len(match), true
	}
	if match := fqlReAny.FindString(rest); match != "" {
		return fqlToken{kind: "any"}, position + len(match), true
	}
	if match := fqlKeyAt(text, position); match != "" {
		return fqlToken{kind: "key", value: match}, position + len(match), true
	}
	if strings.HasPrefix(rest, ":") {
		return fqlToken{kind: "eq"}, position + 1, true
	}
	if match := fqlReCmp.FindString(rest); match != "" {
		return fqlToken{kind: "cmp", value: match}, position + len(match), true
	}
	if match := fqlReNumber.FindString(rest); match != "" {
		return fqlToken{kind: "number", value: match}, position + len(match), true
	}
	if match := fqlReString.FindString(rest); match != "" {
		value := match
		if strings.HasPrefix(match, `"`) || strings.HasPrefix(match, "'") {
			value = match[1 : len(match)-1]
		}
		return fqlToken{kind: "string", value: value}, position + len(match), true
	}
	if char := text[position]; char == '-' || char == ',' || char == '(' || char == ')' {
		return fqlToken{kind: "literal", value: string(char)}, position + 1, true
	}
	return fqlToken{}, position, false
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

// parsePrimary parses the smallest FQL unit — a parenthesized group, a
// quoted string, a bare word, or a wildcard — recursing into the operator
// precedence climb above it.
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
