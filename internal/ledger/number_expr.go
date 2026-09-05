// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package ledger

import (
	"math/big"
	"strings"

	"orangecount/internal/source"
)

// parseNumberExpr consumes a Beancount v3 number expression starting at from.
// It supports +, -, *, /, parentheses, and unary signs, matching the upstream
// number_expr grammar. The returned Number carries the evaluated exact value;
// Raw is the source text (comma-stripped, like tryNumber) when the expression
// is a single literal and the canonical decimal form otherwise so diagnostics
// can point back at the input.
func parseNumberExpr(ts []token, from int) (Number, int, bool) {
	toks, next, ok := splitNumberExpr(ts, from)
	if !ok {
		return Number{}, from, false
	}
	p := &numberExprParser{tokens: toks}
	value, ok := p.parseExpression(0)
	if !ok || p.pos != len(toks) {
		return Number{}, from, false
	}
	raw := NewDecimal(value).String()
	if len(toks) == 1 {
		raw = strings.ReplaceAll(toks[0].text, ",", "")
	}
	return Number{Raw: raw, Rat: value, At: toks[0].span}, next, true
}

type exprToken struct {
	text string
	span source.Span
}

// numberExprSplitter accumulates expression sub-tokens while tracking whether
// the grammar expects an operand next. Its append method is the single gate
// that rejects malformed token sequences.
type numberExprSplitter struct {
	out           []exprToken
	expectOperand bool
}

func (s *numberExprSplitter) appendSub(sub exprToken) bool {
	if isExprDelimiter(sub.text) {
		s.out = append(s.out, sub)
		switch sub.text {
		case "(", "+", "-", "*", "/":
			s.expectOperand = true
		case ")":
			s.expectOperand = false
		}
		return true
	}
	if !s.expectOperand || !looksLikeNumberStart(sub.text) {
		return false
	}
	s.out = append(s.out, sub)
	s.expectOperand = false
	return true
}

func splitNumberExpr(ts []token, from int) ([]exprToken, int, bool) {
	if from >= len(ts) {
		return nil, from, false
	}
	splitter := numberExprSplitter{expectOperand: true}
	i := from
	for i < len(ts) {
		t := ts[i]
		if isExprDelimiter(t.text) {
			if !splitter.appendSub(exprToken{t.text, t.span}) {
				break
			}
			i++
			continue
		}
		if t.kind == tokString || t.kind == tokPunct {
			break
		}
		subs := splitExprWord(t)
		if len(subs) == 0 {
			break
		}
		keepWord := true
		for _, sub := range subs {
			if !splitter.appendSub(sub) {
				keepWord = false
				break
			}
		}
		if !keepWord {
			break
		}
		i++
	}
	if len(splitter.out) == 0 {
		return nil, from, false
	}
	return splitter.out, i, true
}

func isExprDelimiter(s string) bool {
	switch s {
	case "+", "-", "*", "/", "(", ")":
		return true
	default:
		return false
	}
}

func splitExprWord(t token) []exprToken {
	if tryNumber(t).Raw != "" {
		return []exprToken{{t.text, t.span}}
	}
	text := t.text
	var out []exprToken
	runes := []rune(text)
	start := 0
	flush := func(end int) {
		if end > start {
			sub := string(runes[start:end])
			span := source.Span{File: t.span.File, Start: t.span.Start + start, End: t.span.Start + end, StartLine: t.span.StartLine, StartColumn: t.span.StartColumn + start, EndLine: t.span.EndLine, EndColumn: t.span.StartColumn + end}
			out = append(out, exprToken{sub, span})
			start = end
		}
	}
	for i, r := range runes {
		if strings.ContainsRune("+-*/()", r) {
			flush(i)
			out = append(out, exprToken{string(r), subSpan(t.span, i, i+1)})
			start = i + 1
		}
	}
	flush(len(runes))
	return out
}

func subSpan(s source.Span, start, end int) source.Span {
	return source.Span{File: s.File, Start: s.Start + start, End: s.Start + end, StartLine: s.StartLine, StartColumn: s.StartColumn + start, EndLine: s.EndLine, EndColumn: s.StartColumn + end}
}

func looksLikeNumberStart(s string) bool {
	_, ok := new(big.Rat).SetString(strings.ReplaceAll(s, ",", ""))
	return ok
}

type numberExprParser struct {
	tokens []exprToken
	pos    int
}

func (p *numberExprParser) peek() (exprToken, bool) {
	if p.pos >= len(p.tokens) {
		return exprToken{}, false
	}
	return p.tokens[p.pos], true
}

func (p *numberExprParser) consume() exprToken {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *numberExprParser) parseExpression(minPrec int) (*big.Rat, bool) {
	left, ok := p.parseUnary()
	if !ok {
		return nil, false
	}
	for {
		t, ok := p.peek()
		if !ok {
			break
		}
		prec, op, binary := binaryPrecedence(t.text)
		if !binary || prec < minPrec {
			break
		}
		p.consume()
		right, ok := p.parseExpression(prec + 1)
		if !ok {
			return nil, false
		}
		switch op {
		case "+":
			left = new(big.Rat).Add(left, right)
		case "-":
			left = new(big.Rat).Sub(left, right)
		case "*":
			left = new(big.Rat).Mul(left, right)
		case "/":
			if right.Sign() == 0 {
				return nil, false
			}
			left = new(big.Rat).Quo(left, right)
		}
	}
	return left, true
}

func binaryPrecedence(text string) (int, string, bool) {
	switch text {
	case "+", "-":
		return 1, text, true
	case "*", "/":
		return 2, text, true
	default:
		return 0, "", false
	}
}

func (p *numberExprParser) parseUnary() (*big.Rat, bool) {
	t, ok := p.peek()
	if !ok {
		return nil, false
	}
	if t.text == "+" || t.text == "-" {
		p.consume()
		value, ok := p.parseUnary()
		if !ok {
			return nil, false
		}
		if t.text == "-" {
			value = new(big.Rat).Neg(value)
		}
		return value, true
	}
	if t.text == "(" {
		p.consume()
		value, ok := p.parseExpression(0)
		if !ok {
			return nil, false
		}
		closeTok, ok := p.peek()
		if !ok || closeTok.text != ")" {
			return nil, false
		}
		p.consume()
		return value, true
	}
	rat, ok := new(big.Rat).SetString(strings.ReplaceAll(t.text, ",", ""))
	if !ok {
		return nil, false
	}
	p.consume()
	return rat, true
}
