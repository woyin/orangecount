// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"orangecount/internal/diagnostic"
	"orangecount/internal/source"
)

// Parse parses one source file. It never stops at the first malformed line:
// recovery occurs at the next top-level directive, allowing one check run to
// report all independent syntax problems.
func Parse(file *source.SourceFile) (*File, *diagnostic.Bag) {
	p := &parser{file: file, bag: new(diagnostic.Bag)}
	p.parse()
	return p.out, p.bag
}

// ParseText is a convenience for tests and callers that have not built an
// include graph yet.
func ParseText(path string, text []byte) (*File, *diagnostic.Bag) {
	return Parse(source.NewSourceFile(1, path, text))
}

// ParseGraph parses every source in include order and accumulates both source
// graph and syntax diagnostics. The returned map is keyed by source.FileID so
// spans can be resolved without relying on paths that may contain private
// ledger information.
func ParseGraph(graph *source.Graph) (map[source.FileID]*File, *diagnostic.Bag) {
	parsed := make(map[source.FileID]*File)
	bag := new(diagnostic.Bag)
	if graph == nil {
		return parsed, bag
	}
	for _, issue := range graph.Diagnostics {
		severity := diagnostic.Error
		if strings.HasPrefix(issue.Code, "W-") {
			severity = diagnostic.Warning
		}
		path := issue.SourcePath
		if path == "" {
			path = issue.Path
		}
		d := diagnostic.New(issue.Code, severity, issue.Span).WithPath(path)
		for _, related := range issue.Related {
			d.Related = append(d.Related, diagnostic.Related{Span: related})
		}
		bag.Add(d)
	}
	for _, id := range graph.Order {
		file := graph.File(id)
		if file == nil {
			continue
		}
		ast, diagnostics := Parse(file)
		parsed[id] = ast
		bag.Extend(diagnostics.All()...)
	}
	return parsed, bag
}

type parser struct {
	file          *source.SourceFile
	bag           *diagnostic.Bag
	out           *File
	lines         []line
	tx            *Transaction
	posting       *Posting
	lastDirective int
	// dialectAnchor is the most recent explicit date on a dialect line in
	// this file; undated dialect lines inherit it (ADR-0045 block anchoring).
	dialectAnchor *Date
}

type line struct {
	number int
	start  int
	text   string
	indent int
}

type tokenKind uint8

const (
	tokWord tokenKind = iota
	tokString
	tokPunct
)

type token struct {
	kind  tokenKind
	text  string
	value string
	span  source.Span
}

// parse drives the line-oriented scan: read lines, skip blanks and
// standalone comments, hand each directive line to its parser, and recurse
// into includes.
func (p *parser) parse() {
	p.out = &File{Source: p.file}
	p.lastDirective = -1
	if p.file == nil {
		return
	}
	if !utf8.Valid(p.file.Data) {
		p.add("E-SOURCE-UTF8", diagnostic.Error, p.file.Span(0, len(p.file.Data)))
	}
	p.lines = splitLines(p.file)
	for _, ln := range p.lines {
		ts := tokenize(p.file, ln)
		if comment := commentOnLine(p.file, ln); comment != nil {
			p.out.Comments = append(p.out.Comments, *comment)
		}
		for _, t := range ts {
			if t.text == "<unterminated>" {
				p.add("E-PARSE-STRING", diagnostic.Error, t.span)
			}
		}
		if len(ts) == 0 {
			continue
		}
		if ln.indent > 0 {
			p.parseContinuation(ln, ts)
			continue
		}
		p.tx, p.posting = nil, nil
		p.lastDirective = -1
		p.parseDirective(ln, ts)
	}
}

func splitLines(f *source.SourceFile) []line {
	var lines []line
	start, number := 0, 1
	for start <= len(f.Data) {
		end := start
		for end < len(f.Data) && f.Data[end] != '\n' {
			end++
		}
		text := string(f.Data[start:end])
		indent := 0
		for indent < len(text) && (text[indent] == ' ' || text[indent] == '\t') {
			indent++
		}
		lines = append(lines, line{number: number, start: start, text: text, indent: indent})
		number++
		if end == len(f.Data) {
			break
		}
		start = end + 1
	}
	return lines
}

// commentOnLine extracts the trailing "; ..." comment of a line, if any,
// keeping its source span for diagnostics.
func commentOnLine(f *source.SourceFile, ln line) *Comment {
	idx := strings.Index(ln.text, ";")
	if idx < 0 {
		return nil
	}
	// Ignore semicolons inside strings.
	quoted, escaped := false, false
	for i := 0; i < len(ln.text); i++ {
		c := ln.text[i]
		if escaped {
			escaped = false
			continue
		}
		if quoted && c == '\\' {
			escaped = true
			continue
		}
		if c == '"' {
			quoted = !quoted
		}
		if c == ';' && !quoted {
			idx = i
			sp := f.Span(ln.start+idx, ln.start+len(ln.text))
			return &Comment{At: sp, Text: strings.TrimSpace(ln.text[idx+1:])}
		}
	}
	return nil
}

// tokenize splits one physical line into parser tokens. Strings keep both
// their raw text and their unquoted value; sigils are single tokens except
// the "@@" total-price operator. A semicolon starts a comment and ends the
// scan. Each token shape is delegated to a scan* helper so the loop itself
// only decides which scanner applies.
func tokenize(f *source.SourceFile, ln line) []token {
	text := ln.text
	var out []token
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		if r == ';' {
			break
		}
		if r == '"' {
			var tokens []token
			tokens, i = scanStringToken(f, ln, text, i)
			out = append(out, tokens...)
			continue
		}
		if isSigilRune(r, text, i) {
			i = scanSigilToken(f, ln, text, i, &out)
			continue
		}
		i = scanWordToken(f, ln, text, i, &out)
	}
	return out
}

// isSigilRune reports whether the rune at text[i] starts a punctuation token.
// A comma between two ASCII digits is a thousands separator and therefore
// belongs to the surrounding word, not a sigil.
func isSigilRune(r rune, text string, i int) bool {
	if !strings.ContainsRune("{}[](),@~=*", r) {
		return false
	}
	return !(r == ',' && i > 0 && i+1 < len(text) && isASCIIDigit(text[i-1]) && isASCIIDigit(text[i+1]))
}

// scanSigilToken appends the sigil token starting at text[i] and returns the
// index after it. "@@" is one operator; the cost delimiters are represented
// by two adjacent braces and interpreted by the posting parser.
func scanSigilToken(f *source.SourceFile, ln line, text string, i int, out *[]token) int {
	start := i
	if text[i] == '@' && i+1 < len(text) && text[i+1] == '@' {
		i += 2
	} else {
		i++
	}
	*out = append(*out, token{kind: tokPunct, text: text[start:i], span: f.Span(ln.start+start, ln.start+i)})
	return i
}

// scanStringToken consumes the double-quoted string starting at text[i] and
// returns its token plus, when the string is unterminated, a synthetic
// "<unterminated>" marker token that parse() turns into E-PARSE-STRING.
func scanStringToken(f *source.SourceFile, ln line, text string, i int) ([]token, int) {
	start := i
	i++
	escaped, closed := false, false
	for i < len(text) {
		if escaped {
			escaped = false
			i++
			continue
		}
		if text[i] == '\\' {
			escaped = true
			i++
			continue
		}
		if text[i] == '"' {
			i++
			closed = true
			break
		}
		i++
	}
	raw := text[start:i]
	value := raw
	if len(raw) >= 2 && closed {
		if unquoted, err := strconv.Unquote(raw); err == nil {
			value = unquoted
		} else {
			value = raw[1 : len(raw)-1]
		}
	}
	tokens := []token{{kind: tokString, text: raw, value: value, span: f.Span(ln.start+start, ln.start+i)}}
	if !closed {
		tokens = append(tokens, token{kind: tokPunct, text: "<unterminated>", span: f.Span(ln.start+i, ln.start+len(text))})
	}
	return tokens, i
}

// scanWordToken appends the word token starting at text[i]: the maximal run
// that is neither whitespace, a sigil, nor the comment/string openers; a
// comma between digits is consumed as part of the word. Invalid UTF-8 is
// diagnosed by Parse, but tokenization must still make progress over every
// byte so malformed input cannot hang the loop.
func scanWordToken(f *source.SourceFile, ln line, text string, i int, out *[]token) int {
	start := i
	for i < len(text) {
		r, size := utf8.DecodeRuneInString(text[i:])
		if r == ',' && i > start && i+1 < len(text) && isASCIIDigit(text[i-1]) && isASCIIDigit(text[i+1]) {
			i++
			continue
		}
		if unicode.IsSpace(r) || strings.ContainsRune("{}[](),@~=*;\"", r) {
			break
		}
		i += size
	}
	if i == start {
		_, size := utf8.DecodeRuneInString(text[i:])
		if size <= 0 {
			size = 1
		}
		i += size
	}
	*out = append(*out, token{kind: tokWord, text: text[start:i], value: text[start:i], span: f.Span(ln.start+start, ln.start+i)})
	return i
}

func isASCIIDigit(value byte) bool { return value >= '0' && value <= '9' }

// parseContinuation handles one indented line inside a directive: metadata
// pairs, postings, and the dialect block forms.
func (p *parser) parseContinuation(ln line, ts []token) {
	if isMetadataLine(ts) {
		meta, ok := p.parseMetadata(ts)
		if !ok {
			return
		}
		if p.posting != nil {
			p.posting.Meta = append(p.posting.Meta, meta)
			p.posting.At.End = meta.Span.End
			p.posting.At.EndLine, p.posting.At.EndColumn = meta.Span.EndLine, meta.Span.EndColumn
			p.posting.Raw = p.file.Text(p.posting.At)
			p.tx.At.End = meta.Span.End
			p.tx.At.EndLine, p.tx.At.EndColumn = meta.Span.EndLine, meta.Span.EndColumn
			p.tx.Raw = p.file.Text(p.tx.At)
		} else if p.tx != nil {
			p.tx.Meta = append(p.tx.Meta, meta)
			p.tx.At.End = meta.Span.End
			p.tx.At.EndLine, p.tx.At.EndColumn = meta.Span.EndLine, meta.Span.EndColumn
			p.tx.Raw = p.file.Text(p.tx.At)
		} else if !p.attachMetadata(meta) {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, meta.Span)
		}
		return
	}
	if p.tx == nil {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, ts[0].span)
		return
	}
	if p.posting != nil && ln.indent >= 4 {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, ts[0].span)
		return
	}
	// A dialect leg inside a transaction block: an amount-first indented
	// line after a transaction header (or the investment form that starts
	// with a security word and cost batch). It carries no date of its own;
	// the header owns date/flag/payee/narration/tags. Standard postings are
	// always account-first, so the shape is unambiguous.
	if isDialectStart(ts) || isInvestmentHead(ts) {
		p.parseDialectLeg(ln, ts)
		return
	}
	posting, ok := p.parsePosting(ts)
	if !ok {
		return
	}
	p.tx.Postings = append(p.tx.Postings, posting)
	p.posting = &p.tx.Postings[len(p.tx.Postings)-1]
	p.tx.At.End = posting.At.End
	p.tx.At.EndLine, p.tx.At.EndColumn = posting.At.EndLine, posting.At.EndColumn
	p.tx.Raw = p.file.Text(p.tx.At)
}

// parseDirective dispatches one top-level line to its directive parser
// based on the leading keyword.
func (p *parser) parseDirective(ln line, ts []token) {
	if len(ts) == 0 {
		return
	}
	if isDialectStart(ts) {
		p.parseDialectLine(ln, ts)
		return
	}
	first := strings.ToLower(ts[0].text)
	if isNonDateKeyword(first) {
		p.parseKeyword(ts)
		return
	}
	if !looksDate(ts[0].text) {
		p.add("E-PARSE-DIRECTIVE", diagnostic.Error, ts[0].span)
		return
	}
	date := p.parseDate(ts[0])
	if len(ts) < 2 {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, ts[0].span)
		return
	}
	keyword := strings.ToLower(ts[1].text)
	if isDateDirective(keyword) {
		p.parseDateDirective(keyword, date, ts[2:], ts[0].span)
		return
	}
	if !isFlag(ts[1].text) && ts[1].kind != tokString && !strings.HasPrefix(ts[1].text, "#") && !strings.HasPrefix(ts[1].text, "^") {
		p.add("E-PARSE-DIRECTIVE", diagnostic.Error, ts[1].span)
		return
	}
	// A date followed by a flag/string is a transaction. Unknown one-character
	// flags are retained so future v3 flags can be diagnosed by validation.
	p.parseTransaction(date, ts)
}

func isNonDateKeyword(k string) bool {
	switch k {
	case "option", "plugin", "include", "pushtag", "poptag":
		return true
	default:
		return false
	}
}

func isDateDirective(k string) bool {
	switch k {
	case "open", "close", "commodity", "balance", "pad", "event", "query", "price", "document", "note", "custom":
		return true
	default:
		return false
	}
}

// parseKeyword parses the keywords that may start a line without a date:
// option, plugin, include, pushtag, and poptag. Each keyword's token-shape
// validation lives in its own parser.
func (p *parser) parseKeyword(ts []token) {
	if len(ts) == 0 {
		return
	}
	base := p.base(ts)
	switch strings.ToLower(ts[0].text) {
	case "option":
		p.parseOptionKeyword(base, ts)
	case "plugin":
		p.parsePluginKeyword(base, ts)
	case "include":
		p.parseIncludeKeyword(base, ts)
	case "pushtag", "poptag":
		p.parseTagKeyword(base, ts)
	}
}

// parseOptionKeyword reads `option "key" "value"`.
func (p *parser) parseOptionKeyword(base DirectiveBase, ts []token) {
	if len(ts) != 3 || ts[1].kind != tokString || ts[2].kind != tokString {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	p.appendDirective(Option{DirectiveBase: base, Key: ts[1].value, Value: ts[2].value})
}

// parsePluginKeyword reads `plugin "module" ["config"]` and warns that v3
// migrates plugins into the core engine.
func (p *parser) parsePluginKeyword(base DirectiveBase, ts []token) {
	if (len(ts) != 2 && len(ts) != 3) || ts[1].kind != tokString || (len(ts) == 3 && ts[2].kind != tokString) {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	d := Plugin{DirectiveBase: base, Module: ts[1].value}
	if len(ts) > 2 && ts[2].kind == tokString {
		d.Config = ts[2].value
	}
	p.appendDirective(d)
	p.add("W-PLUGIN-MIGRATION", diagnostic.Warning, ts[0].span)
}

// parseIncludeKeyword reads `include "path"`.
func (p *parser) parseIncludeKeyword(base DirectiveBase, ts []token) {
	if len(ts) != 2 || ts[1].kind != tokString {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	p.appendDirective(Include{DirectiveBase: base, Path: ts[1].value})
}

// parseTagKeyword reads `pushtag #tag` / `poptag #tag`.
func (p *parser) parseTagKeyword(base DirectiveBase, ts []token) {
	if len(ts) != 2 || !strings.HasPrefix(ts[1].text, "#") {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	p.appendDirective(TagDirective{DirectiveBase: base, Tag: strings.TrimPrefix(ts[1].text, "#")})
}

// parseDateDirective dispatches a dated directive (open/close/balance/...) to
// its dedicated parser. Each directive's arity and token-shape rules live
// with its parser so the dispatcher stays a pure lookup.
func (p *parser) parseDateDirective(keyword string, date Date, rest []token, dateSpan source.Span) {
	base := p.base(append([]token{{span: dateSpan}}, rest...))
	base.At.Start = dateSpan.Start
	if len(rest) == 0 {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, dateSpan)
		return
	}
	parser, ok := dateDirectiveParsers[keyword]
	if !ok {
		return
	}
	parser(p, base, date, rest)
}

// dateDirectiveParsers maps directive keywords to their parsers; the keys are
// exactly the set isDateDirective accepts.
var dateDirectiveParsers = map[string]func(*parser, DirectiveBase, Date, []token){
	"open":      (*parser).parseOpenDirective,
	"close":     (*parser).parseCloseDirective,
	"commodity": (*parser).parseCommodityDirective,
	"balance":   (*parser).parseBalanceDirective,
	"pad":       (*parser).parsePadDirective,
	"event":     (*parser).parseEventDirective,
	"query":     (*parser).parseQueryDirective,
	"price":     (*parser).parsePriceDirective,
	"document":  (*parser).parseDocumentDirective,
	"note":      (*parser).parseNoteDirective,
	"custom":    (*parser).parseCustomDirective,
}

func (p *parser) parseOpenDirective(base DirectiveBase, date Date, rest []token) {
	d := Open{DirectiveBase: base, Date: date, Account: rest[0].text}
	for _, t := range rest[1:] {
		if t.kind == tokString {
			d.Booking = t.value
		} else if t.text != "," {
			d.Currencies = append(d.Currencies, t.text)
		}
	}
	p.appendDirective(d)
}

func (p *parser) parseCloseDirective(base DirectiveBase, date Date, rest []token) {
	p.appendDirective(Close{DirectiveBase: base, Date: date, Account: rest[0].text})
}

func (p *parser) parseCommodityDirective(base DirectiveBase, date Date, rest []token) {
	p.appendDirective(Commodity{DirectiveBase: base, Date: date, Currency: rest[0].text})
}

// parseBalanceDirective reads "account amount [~ tolerance]"; the optional
// tolerance becomes a pointer so its absence stays distinguishable.
func (p *parser) parseBalanceDirective(base DirectiveBase, date Date, rest []token) {
	if len(rest) < 3 {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	d := Balance{DirectiveBase: base, Date: date, Account: rest[0].text}
	amount, next, ok := p.amount(rest, 1)
	if !ok {
		return
	}
	d.Amount = amount
	if next < len(rest) && rest[next].text == "~" && next+1 < len(rest) {
		n := p.number(rest[next+1])
		d.Tolerance = &n
	}
	p.appendDirective(d)
}

func (p *parser) parsePadDirective(base DirectiveBase, date Date, rest []token) {
	if len(rest) < 2 {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	p.appendDirective(Pad{DirectiveBase: base, Date: date, Account: rest[0].text, SourceAccount: rest[1].text})
}

// parseEventDirective requires two quoted strings (type and value).
func (p *parser) parseEventDirective(base DirectiveBase, date Date, rest []token) {
	if len(rest) < 2 || rest[0].kind != tokString || rest[1].kind != tokString {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	p.appendDirective(Event{DirectiveBase: base, Date: date, Type: rest[0].value, Value: rest[1].value})
}

// parseQueryDirective requires two quoted strings (name and query text).
func (p *parser) parseQueryDirective(base DirectiveBase, date Date, rest []token) {
	if len(rest) < 2 || rest[0].kind != tokString || rest[1].kind != tokString {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	p.appendDirective(Query{DirectiveBase: base, Date: date, Name: rest[0].value, Query: rest[1].value})
}

func (p *parser) parsePriceDirective(base DirectiveBase, date Date, rest []token) {
	if len(rest) < 3 {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	amount, _, ok := p.amount(rest, 1)
	if !ok {
		return
	}
	p.appendDirective(Price{DirectiveBase: base, Date: date, Currency: rest[0].text, Amount: amount})
}

// parseDocumentDirective collects filenames, tags, and links behind the
// account in whatever order they appear.
func (p *parser) parseDocumentDirective(base DirectiveBase, date Date, rest []token) {
	if len(rest) < 2 {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	d := Document{DirectiveBase: base, Date: date, Account: rest[0].text}
	for _, t := range rest[1:] {
		switch t.kind {
		case tokString:
			d.Filenames = append(d.Filenames, t.value)
		default:
			if strings.HasPrefix(t.text, "#") {
				d.Tags = append(d.Tags, strings.TrimPrefix(t.text, "#"))
			} else if strings.HasPrefix(t.text, "^") {
				d.Links = append(d.Links, strings.TrimPrefix(t.text, "^"))
			}
		}
	}
	p.appendDirective(d)
}

func (p *parser) parseNoteDirective(base DirectiveBase, date Date, rest []token) {
	if len(rest) < 2 || rest[1].kind != tokString {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	p.appendDirective(Note{DirectiveBase: base, Date: date, Account: rest[0].text, Comment: rest[1].value})
}

// parseCustomDirective reads typed custom values separated by commas; the
// type name may be quoted or bare, matching Beancount's grammar.
func (p *parser) parseCustomDirective(base DirectiveBase, date Date, rest []token) {
	if len(rest) < 2 {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
		return
	}
	typeName := rest[0].text
	if rest[0].kind == tokString {
		typeName = rest[0].value
	}
	d := Custom{DirectiveBase: base, Date: date, Type: typeName}
	for i := 1; i < len(rest); {
		if rest[i].text == "," {
			i++
			continue
		}
		v, next := p.value(rest, i)
		d.Values = append(d.Values, v)
		if next <= i {
			next = i + 1
		}
		i = next
	}
	p.appendDirective(d)
}

// parseTransaction parses a transaction header line: flag, payee and
// narration (quoted or bare), tags, and links; postings arrive as
// continuations.
func (p *parser) parseTransaction(date Date, ts []token) {
	base := p.base(ts)
	tx := Transaction{DirectiveBase: base, Date: date}
	idx := 1
	if idx < len(ts) && isFlag(ts[idx].text) {
		tx.Flag = ts[idx].text
		idx++
	}
	var stringsFound []string
	for ; idx < len(ts); idx++ {
		t := ts[idx]
		if t.kind == tokString {
			stringsFound = append(stringsFound, t.value)
		} else if strings.HasPrefix(t.text, "#") {
			tx.Tags = append(tx.Tags, strings.TrimPrefix(t.text, "#"))
		} else if strings.HasPrefix(t.text, "^") {
			tx.Links = append(tx.Links, strings.TrimPrefix(t.text, "^"))
		} else {
			p.add("E-PARSE-TOKEN", diagnostic.Error, t.span)
		}
	}
	if len(stringsFound) == 1 {
		tx.Narration = stringsFound[0]
	} else if len(stringsFound) >= 2 {
		tx.Payee, tx.Narration = stringsFound[0], stringsFound[1]
		if len(stringsFound) > 2 {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, ts[len(ts)-1].span)
		}
	}
	p.appendDirective(&tx)
	p.tx = &tx
	p.posting = nil
}

func isFlag(s string) bool {
	if s == "*" || s == "!" {
		return true
	}
	return len([]rune(s)) == 1 && unicode.IsLetter([]rune(s)[0])
}

// isDialectStart reports whether the top-level token stream opens a dialect
// line (ADR-0045). The shapes — amount-first, flag+amount, date+amount, or
// date+flag+amount — are all syntax errors in v3, so claiming them cannot
// misread any standard directive.
func isDialectStart(ts []token) bool {
	if len(ts) == 0 {
		return false
	}
	if looksDate(ts[0].text) {
		// A date opens a dialect line only when followed by an amount, or a
		// flag then an amount; every other dated shape is standard syntax.
		if len(ts) > 1 && looksAmountWord(ts[1].text) {
			return true
		}
		return len(ts) > 2 && (ts[1].text == "*" || ts[1].text == "!") && looksAmountWord(ts[2].text)
	}
	if looksAmountWord(ts[0].text) {
		return true
	}
	return (ts[0].text == "*" || ts[0].text == "!") && len(ts) > 1 && looksAmountWord(ts[1].text)
}

// looksAmountWord reports whether text plausibly opens an amount: an ASCII
// digit, optionally signed, or a leading decimal point followed by a digit.
// Full validation happens in parseDialectLine.
func looksAmountWord(text string) bool {
	if text == "" {
		return false
	}
	start := 0
	if text[0] == '-' || text[0] == '+' {
		start = 1
	}
	if start >= len(text) {
		return false
	}
	return isASCIIDigit(text[start]) || (text[start] == '.' && start+1 < len(text) && isASCIIDigit(text[start+1]))
}

// parseDialectLine parses one dialect shorthand line:
// [DATE] [FLAG] AMOUNT [CURRENCY] @source -> @destination ["payee"] [: narration] [#tag] [^link]
// Syntax problems are diagnosed here with E-DIALECT-* codes; endpoint and
// currency semantics are resolved later by the dialect pass.
func (p *parser) parseDialectLine(ln line, ts []token) {
	d := Dialect{DirectiveBase: p.base(ts), Flag: "*"}
	rest := p.dialectHead(ts, &d)
	if rest == nil {
		return
	}
	next, ok := p.dialectAmount(rest, &d)
	if !ok {
		return
	}
	source, afterSource, ok := p.dialectEndpoint(rest, next, "source")
	if !ok {
		return
	}
	d.SourceRef = source
	if afterSource >= len(rest) || rest[afterSource].kind != tokWord || rest[afterSource].text != "->" {
		span := rest[len(rest)-1].span
		if afterSource < len(rest) {
			span = rest[afterSource].span
		}
		p.add("E-DIALECT-ARROW", diagnostic.Error, span)
		return
	}
	dest, afterDest, ok := p.dialectEndpoint(rest, afterSource+1, "destination")
	if !ok {
		return
	}
	d.DestRef = dest
	if !p.dialectTail(rest, afterDest, &d) {
		return
	}
	if !d.HasDate {
		if p.dialectAnchor == nil {
			p.add("E-DIALECT-DATE", diagnostic.Error, d.At)
			return
		}
		d.Date, d.Anchored = *p.dialectAnchor, true
	}
	p.appendDirective(d)
}

// parseDialectLeg parses one indented leg of a dialect block: a positive
// amount, optional currency, and "@src -> @dst" endpoints. The transaction
// header owns date, flag, payee, narration, and tags; a leg only adds money
// movement. Legs never take payee/narration/tags of their own.
func (p *parser) parseDialectLeg(ln line, ts []token) {
	d := Dialect{DirectiveBase: p.base(ts), Flag: "*"}
	if p.posting != nil {
		p.add("E-DIALECT-LEG-ORDER", diagnostic.Error, d.At)
		return
	}
	if len(p.tx.Postings) > 0 {
		p.add("E-DIALECT-LEG-ORDER", diagnostic.Error, d.At)
		return
	}
	// The leg head is either a plain amount (with optional currency) or an
	// investment quantity + security + cost batch: "1,000 FUND_019305
	// {1.5010 CNY}", or the auto-quantity form with the quantity omitted:
	// "FUND_019305 {1.5010 CNY}". The investment shape replaces the amount
	// slot so the cash side can be derived or drive the quantity.
	idx := 0
	if isInvestmentHead(ts) {
		if !p.dialectSecurity(ts, &idx, &d) {
			return
		}
	} else {
		next, ok := p.dialectAmount(ts, &d)
		if !ok {
			return
		}
		idx = next
	}
	source, afterSource, ok := p.dialectEndpoint(ts, idx, "source")
	if !ok {
		return
	}
	d.SourceRef = source
	if afterSource >= len(ts) || ts[afterSource].kind != tokWord || ts[afterSource].text != "->" {
		span := ts[len(ts)-1].span
		if afterSource < len(ts) {
			span = ts[afterSource].span
		}
		p.add("E-DIALECT-ARROW", diagnostic.Error, span)
		return
	}
	dest, afterDest, ok := p.dialectEndpoint(ts, afterSource+1, "destination")
	if !ok {
		return
	}
	d.DestRef = dest
	cursor, ok := p.dialectLegTail(ts, afterDest, &d)
	if !ok {
		return
	}
	if cursor != len(ts) {
		p.add("E-DIALECT-SYNTAX", diagnostic.Error, ts[cursor].span)
		return
	}
	// A leg stays dateless; Expand pairs it with its header transaction.
	p.appendDirective(d)
}

// dialectLegTail parses the optional leg tail after the destination: a
// sell's gain endpoint ("-> @account", receiving the realized P&L as an
// elided posting) and the fee suffix ("手续费 AMOUNT CURRENCY @account").
func (p *parser) dialectLegTail(ts []token, idx int, d *Dialect) (int, bool) {
	cursor := idx
	if cursor+1 < len(ts) && ts[cursor].kind == tokWord && ts[cursor].text == "->" && ts[cursor+1].text == "@" {
		gain, afterGain, ok := p.dialectEndpoint(ts, cursor+1, "gain")
		if !ok {
			return cursor, false
		}
		d.GainRef = gain
		cursor = afterGain
	}
	if cursor < len(ts) && ts[cursor].kind == tokWord && ts[cursor].text == "手续费" {
		fee, afterFee, ok := p.dialectFee(ts, cursor)
		if !ok {
			return cursor, false
		}
		d.FeeAmount, d.FeeCurrency, d.FeeRef = fee.amount, fee.currency, fee.ref
		cursor = afterFee
	}
	return cursor, true
}

// dialectFee parses the fee suffix "手续费 AMOUNT CURRENCY @account" at
// ts[idx] (the 手续费 word).
func (p *parser) dialectFee(ts []token, idx int) (fee struct {
	amount   Number
	currency string
	ref      string
}, next int, ok bool) {
	amount := tryNumber(ts[idx+1])
	if amount.Raw == "" || (amount.Rat != nil && amount.Rat.Sign() < 0) {
		p.add("E-DIALECT-AMOUNT", diagnostic.Error, ts[idx+1].span)
		return fee, idx, false
	}
	fee.amount = amount
	cursor := idx + 2
	if cursor >= len(ts) || ts[cursor].kind != tokWord || !isCurrencyWord(ts[cursor].text) {
		p.addf("E-DIALECT-CURRENCY", diagnostic.Error, ts[idx+1].span, "fee needs an explicit currency")
		return fee, idx, false
	}
	fee.currency = ts[cursor].text
	ref, afterRef, ok := p.dialectEndpoint(ts, cursor+1, "fee")
	if !ok {
		return fee, idx, false
	}
	fee.ref = ref
	return fee, afterRef, true
}

// isInvestmentHead reports whether the leg's head is the investment shape;
// see investmentHeadShape for the recognized forms.
func isInvestmentHead(ts []token) bool {
	return investmentHeadShape(ts).start != -1
}

// investmentShape identifies the investment head forms. The position of the
// cost brace disambiguates them, so ticker symbols without digits (AAPL)
// never read as currencies:
//
//	QUANTITY SECURITY {COST}                 start 1 (buy, explicit qty)
//	AMOUNT CURRENCY SECURITY {COST}          start 2 (buy, auto qty)
//	AMOUNT CURRENCY QUANTITY SECURITY {COST} start 3 (sell, explicit cash)
//	SECURITY {COST}                          start 0 (underdetermined)
//
// start is the index of the security word; -1 means not an investment head.
// The sell form without leading cash (QUANTITY SECURITY {COST} @ PRICE ...)
// shares the first shape; the price after the cost batch marks it.
type investmentShape struct {
	start     int
	hasAmount bool
}

// investmentHeadShape recognizes the head of a dialect investment leg —
// "{cost}" alone, "QTY SEC", "AMT CUR QTY", or "AMT CUR" — and reports where
// the leg payload starts and whether an explicit amount was written.
func investmentHeadShape(ts []token) investmentShape {
	switch {
	case len(ts) >= 2 && ts[0].kind == tokWord && ts[1].text == "{":
		return investmentShape{start: 0}
	case quantitySecurityHead(ts):
		return investmentShape{start: 1}
	case amountCurrencyPrefix(ts) && len(ts) >= 4 && ts[2].kind == tokWord && ts[3].text == "{":
		return investmentShape{start: 2, hasAmount: true}
	case amountCurrencyPrefix(ts) && len(ts) >= 5 && looksAmountWord(ts[2].text) && ts[3].kind == tokWord && ts[4].text == "{":
		return investmentShape{start: 3, hasAmount: true}
	}
	return investmentShape{start: -1}
}

// quantitySecurityHead reports whether ts starts with "QUANTITY SECURITY {".
func quantitySecurityHead(ts []token) bool {
	return len(ts) >= 3 && looksAmountWord(ts[0].text) && ts[1].kind == tokWord && ts[2].text == "{"
}

// amountCurrencyPrefix reports whether ts starts with "AMOUNT CURRENCY".
func amountCurrencyPrefix(ts []token) bool {
	return len(ts) >= 2 && looksAmountWord(ts[0].text) && isCurrencyWord(ts[1].text)
}

// isCurrencyWord reports whether text is plausibly a currency symbol (an
// uppercase word that is not a security symbol with an embedded number).
func isCurrencyWord(text string) bool {
	return len(text) <= 8 && isAccountWord(text) && !strings.ContainsAny(text, "0123456789")
}

func isAccountWord(text string) bool {
	if text == "" {
		return false
	}
	return text[0] >= 'A' && text[0] <= 'Z'
}

// dialectSecurity parses the investment leg head. Shapes:
//   - "QUANTITY SECURITY {COST} [@ PRICE]"        buy or sell (explicit qty)
//   - "AMOUNT CURRENCY SECURITY {COST}"          buy, auto quantity
//   - "AMOUNT CURRENCY QUANTITY SECURITY {COST} [@ PRICE]"  sell, explicit cash
//   - "SECURITY {COST}"                          underdetermined; errors later
//
// The cost follows beancount's own posting grammar. A "@ NUMBER CURRENCY"
// after the cost batch is the sale price and marks a sell leg.
func (p *parser) dialectSecurity(ts []token, idx *int, d *Dialect) bool {
	shape := investmentHeadShape(ts)
	if shape.start < 0 {
		p.add("E-DIALECT-SECURITY", diagnostic.Error, ts[0].span)
		return false
	}
	if shape.hasAmount {
		d.Amount = tryNumber(ts[0])
		d.Currency = ts[1].text
	}
	switch shape.start {
	case 1:
		d.Quantity, d.HasQuantity = tryNumber(ts[0]), true
	case 3:
		d.Quantity, d.HasQuantity = tryNumber(ts[2]), true
	}
	d.Security = ts[shape.start].text
	cost, next := p.cost(ts, shape.start+1)
	d.Cost = &cost
	if next+1 < len(ts) && ts[next].kind == tokPunct && ts[next].text == "@" && looksAmountWord(ts[next+1].text) {
		// "@ NUMBER CURRENCY" sale price marks a sell leg. The currency is
		// required: this ledger runs two operating currencies.
		price := tryNumber(ts[next+1])
		if price.Raw == "" {
			p.add("E-DIALECT-SECURITY", diagnostic.Error, ts[next+1].span)
			return false
		}
		cursor := next + 2
		if cursor >= len(ts) || ts[cursor].kind != tokWord || !isCurrencyWord(ts[cursor].text) {
			p.addf("E-DIALECT-SECURITY", diagnostic.Error, ts[next+1].span, "sale price needs an explicit currency")
			return false
		}
		d.Price = &PriceSpec{At: ts[next].span, Amount: Amount{Number: price, Currency: ts[cursor].text, At: ts[next+1].span}}
		next = cursor + 1
	}
	*idx = next
	return true
}

// dialectHead consumes the optional date and flag and applies block
// anchoring state. It returns the remaining tokens, or nil on failure.
func (p *parser) dialectHead(ts []token, d *Dialect) []token {
	idx := 0
	if looksDate(ts[idx].text) {
		date := p.parseDate(ts[idx])
		d.Date, d.HasDate = date, true
		p.dialectAnchor = &date
		idx++
		if idx < len(ts) && (ts[idx].text == "*" || ts[idx].text == "!") {
			d.Flag = ts[idx].text
			idx++
		}
	} else if ts[idx].text == "*" || ts[idx].text == "!" {
		d.Flag = ts[idx].text
		idx++
	}
	if idx >= len(ts) {
		p.add("E-DIALECT-AMOUNT", diagnostic.Error, ts[len(ts)-1].span)
		return nil
	}
	return ts[idx:]
}

// dialectAmount validates and records the positive amount plus the optional
// currency word that may follow it. It returns the index of the next
// unconsumed token.
func (p *parser) dialectAmount(ts []token, d *Dialect) (int, bool) {
	amount := tryNumber(ts[0])
	if amount.Raw == "" || (amount.Rat != nil && amount.Rat.Sign() < 0) {
		p.add("E-DIALECT-AMOUNT", diagnostic.Error, ts[0].span)
		return 0, false
	}
	d.Amount = amount
	if len(ts) > 1 && ts[1].kind == tokWord && ts[1].text != "->" {
		d.Currency = ts[1].text
		return 2, true
	}
	return 1, true
}

// dialectEndpoint reads the "@name" endpoint at ts[idx]. It returns the
// endpoint text and the index after it.
func (p *parser) dialectEndpoint(ts []token, idx int, role string) (string, int, bool) {
	if idx >= len(ts) || ts[idx].kind != tokPunct || (ts[idx].text != "@" && ts[idx].text != "@@") {
		span := ts[len(ts)-1].span
		if idx < len(ts) {
			span = ts[idx].span
		}
		p.add("E-DIALECT-SYNTAX", diagnostic.Error, span)
		return "", idx, false
	}
	if ts[idx].text == "@@" || idx+1 >= len(ts) || ts[idx+1].kind != tokWord {
		p.add("E-DIALECT-SYNTAX", diagnostic.Error, ts[idx].span)
		return "", idx, false
	}
	return ts[idx+1].text, idx + 2, true
}

// dialectTail parses everything after the destination endpoint: an optional
// quoted payee, an optional ": narration" run, and #tag/^link words in any
// order after their first appearance.
func (p *parser) dialectTail(ts []token, idx int, d *Dialect) bool {
	narration := false
	var words []string
	for ; idx < len(ts); idx++ {
		t := ts[idx]
		switch {
		case t.kind == tokString && !narration:
			if d.Payee != "" {
				p.add("E-DIALECT-SYNTAX", diagnostic.Error, t.span)
				return false
			}
			d.Payee = t.value
		case t.text == ":" && !narration && len(words) == 0:
			narration = true
		case strings.HasPrefix(t.text, "#"):
			d.Tags = append(d.Tags, strings.TrimPrefix(t.text, "#"))
		case strings.HasPrefix(t.text, "^"):
			d.Links = append(d.Links, strings.TrimPrefix(t.text, "^"))
		case t.kind == tokString || t.kind == tokPunct:
			p.add("E-DIALECT-SYNTAX", diagnostic.Error, t.span)
			return false
		case !narration:
			// A bare word without a preceding ':' is not part of the grammar.
			p.add("E-DIALECT-SYNTAX", diagnostic.Error, t.span)
			return false
		default:
			words = append(words, t.text)
		}
	}
	if narration {
		d.Narration, d.HasNarration = strings.Join(words, " "), true
	}
	return true
}

// parsePosting parses one posting line: account [flag] [units] followed by
// cost, price, and assignment modifiers. It returns ok=false only when a
// modifier is syntactically incomplete; the caller then drops the line (its
// errors were already reported).
func (p *parser) parsePosting(ts []token) (Posting, bool) {
	if len(ts) == 0 {
		return Posting{}, false
	}
	base := p.base(ts)
	post := Posting{At: base.At, Raw: base.Raw, Account: ts[0].text}
	i := 1
	if i < len(ts) && isFlag(ts[i].text) {
		post.Flag = ts[i].text
		i++
	}
	if i < len(ts) && ts[i].text != "{" && ts[i].text != "@" && ts[i].text != "@@" {
		if amount, next, ok := p.amount(ts, i); ok {
			post.Units, i = &amount, next
		}
	}
	return p.parsePostingTail(post, ts, i)
}

// parsePostingTail consumes the trailing modifier run: {cost}, @/@@ price,
// and = / ~ amount assignments. Unknown tokens are reported and skipped so
// one bad modifier does not cascade.
func (p *parser) parsePostingTail(post Posting, ts []token, i int) (Posting, bool) {
	for i < len(ts) {
		switch ts[i].text {
		case "{":
			cost, next := p.cost(ts, i)
			post.Cost, i = &cost, next
		case "@", "@@":
			spec := PriceSpec{At: ts[i].span, Total: ts[i].text == "@@"}
			amount, next, ok := p.amount(ts, i+1)
			if !ok {
				return post, false
			}
			spec.Amount = amount
			post.Price, i = &spec, next
		default:
			i = p.skipPostingAssignment(ts, i)
		}
	}
	return post, true
}

// skipPostingAssignment consumes an "= amount" or "~ amount" assignment
// (retained only for source fidelity) or reports an unexpected token.
func (p *parser) skipPostingAssignment(ts []token, i int) int {
	if ts[i].text == "=" || ts[i].text == "~" {
		i++
		if _, next, ok := p.amount(ts, i); ok {
			return next
		}
		return i
	}
	p.add("E-PARSE-TOKEN", diagnostic.Error, ts[i].span)
	return i + 1
}

// cost parses a cost or cost-total spec — "{...}" or "{{...}}" — with
// amount, date, and label components.
func (p *parser) cost(ts []token, i int) (CostSpec, int) {
	spec := CostSpec{At: ts[i].span}
	start := i
	i++
	if i < len(ts) && ts[i].text == "{" {
		spec.Total = true
		i++
	}
	for i < len(ts) {
		if ts[i].text == "}" {
			i++
			if i < len(ts) && ts[i].text == "}" {
				i++
			}
			break
		}
		if ts[i].text == "," {
			i++
			continue
		}
		v, next := p.value(ts, i)
		spec.Components = append(spec.Components, v)
		if next <= i {
			next = i + 1
		}
		i = next
	}
	if i > start {
		end := ts[i-1].span.End
		spec.At = p.file.Span(ts[start].span.Start, end)
		spec.Raw = p.file.Text(spec.At)
	}
	return spec, i
}

// amount parses a number+currency pair starting at token i.
func (p *parser) amount(ts []token, i int) (Amount, int, bool) {
	if i >= len(ts) {
		return Amount{}, i, false
	}
	if ts[i].text == "*" {
		return Amount{At: ts[i].span}, i + 1, true
	}
	// Beancount's incomplete_amount grammar permits either side to be omitted:
	// `Account CURRENCY`, `Account 1.00`, or a completely bare posting. A
	// numeric token before a suffix delimiter is therefore a number-only amount,
	// while an ordinary word in that position is a currency-only amount. Keep
	// the distinction so evaluator inference can see which component is absent.
	if ts[i].kind == tokWord && (i+1 >= len(ts) || isPunctuation(ts[i+1].text)) {
		if number := tryNumber(ts[i]); number.Raw != "" {
			return Amount{Number: number, At: ts[i].span}, i + 1, true
		}
		return Amount{Currency: ts[i].text, At: ts[i].span}, i + 1, true
	}
	if i+1 >= len(ts) {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, ts[i].span)
		return Amount{}, i, false
	}
	n := p.number(ts[i])
	if n.Raw == "" || ts[i+1].kind == tokString || isPunctuation(ts[i+1].text) {
		p.add("E-PARSE-TOKEN", diagnostic.Error, ts[i].span)
		return Amount{}, i, false
	}
	return Amount{Number: n, Currency: ts[i+1].text, At: p.file.Span(ts[i].span.Start, ts[i+1].span.End)}, i + 2, true
}

func (p *parser) number(t token) Number {
	n := tryNumber(t)
	if n.Raw == "" {
		p.add("E-PARSE-TOKEN", diagnostic.Error, t.span)
	}
	return n
}

func tryNumber(t token) Number {
	raw := strings.ReplaceAll(t.text, ",", "")
	n := Number{Raw: raw, At: t.span}
	if _, ok := new(big.Rat).SetString(raw); !ok {
		n.Raw = ""
	} else {
		n.Rat, _ = new(big.Rat).SetString(raw)
	}
	return n
}

func (p *parser) parseMetadata(ts []token) (Metadata, bool) {
	if len(ts) < 2 {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, ts[0].span)
		return Metadata{}, false
	}
	key := strings.TrimSuffix(ts[0].text, ":")
	idx := 1
	if ts[0].text == key {
		if ts[1].text != ":" {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, ts[0].span)
			return Metadata{}, false
		}
		idx++
	}
	if idx >= len(ts) {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, ts[0].span)
		return Metadata{}, false
	}
	v, _ := p.value(ts, idx)
	return Metadata{Key: key, Value: v, Span: p.file.Span(ts[0].span.Start, ts[len(ts)-1].span.End)}, true
}

func isMetadataLine(ts []token) bool {
	if len(ts) < 2 {
		return false
	}
	return strings.HasSuffix(ts[0].text, ":") || ts[1].text == ":"
}

// value classifies one token as a typed metadata/custom value: lists (parsed
// recursively), strings, booleans, dates, tags, links, amounts (number
// followed by a non-delimiter currency word), numbers, accounts, and finally
// bare currencies. The order matters: date/tag/link/account shapes are
// checked before the number fallback so "2026-01-02" is a date, not a
// subtraction.
func (p *parser) value(ts []token, i int) (Value, int) {
	if i >= len(ts) {
		return Value{Kind: ValueInvalid}, i
	}
	t := ts[i]
	if t.text == "[" {
		return p.listValue(ts, i)
	}
	if t.kind == tokString {
		return Value{Kind: ValueString, Raw: t.text, String: t.value, At: t.span}, i + 1
	}
	if strings.EqualFold(t.text, "true") || strings.EqualFold(t.text, "false") {
		return Value{Kind: ValueBool, Raw: t.text, Bool: strings.EqualFold(t.text, "true"), At: t.span}, i + 1
	}
	if looksDate(t.text) {
		return Value{Kind: ValueDate, Raw: t.text, Date: p.parseDate(t), At: t.span}, i + 1
	}
	if strings.HasPrefix(t.text, "#") {
		return Value{Kind: ValueTag, Raw: t.text, String: strings.TrimPrefix(t.text, "#"), At: t.span}, i + 1
	}
	if strings.HasPrefix(t.text, "^") {
		return Value{Kind: ValueLink, Raw: t.text, String: strings.TrimPrefix(t.text, "^"), At: t.span}, i + 1
	}
	if n := tryNumber(t); n.Raw != "" {
		return p.numberValue(ts, i, n)
	}
	if isAccount(t.text) {
		return Value{Kind: ValueAccount, Raw: t.text, String: t.text, At: t.span}, i + 1
	}
	return Value{Kind: ValueCurrency, Raw: t.text, String: t.text, At: t.span}, i + 1
}

// listValue parses a [item, item, ...] list; a missing closing bracket
// simply ends the list at the line's end rather than failing.
func (p *parser) listValue(ts []token, i int) (Value, int) {
	v := Value{Kind: ValueList, At: ts[i].span}
	i++
	for i < len(ts) && ts[i].text != "]" {
		if ts[i].text == "," {
			i++
			continue
		}
		item, next := p.value(ts, i)
		v.List = append(v.List, item)
		if next <= i {
			next = i + 1
		}
		i = next
	}
	if i < len(ts) {
		v.At = p.file.Span(v.At.Start, ts[i].span.End)
		i++
	}
	return v, i
}

// numberValue classifies a numeric token: followed by a currency word it is
// an amount, otherwise a bare number.
func (p *parser) numberValue(ts []token, i int, n Number) (Value, int) {
	if i+1 < len(ts) && !isPunctuation(ts[i+1].text) {
		a := Amount{Number: n, Currency: ts[i+1].text, At: p.file.Span(ts[i].span.Start, ts[i+1].span.End)}
		return Value{Kind: ValueAmount, Raw: p.file.Text(a.At), Amount: a, At: a.At}, i + 2
	}
	return Value{Kind: ValueNumber, Raw: ts[i].text, Number: n, At: ts[i].span}, i + 1
}

func isAccount(s string) bool {
	if !strings.Contains(s, ":") {
		return false
	}
	first := s[0]
	return first >= 'A' && first <= 'Z'
}

// isPunctuation reports whether a token is one of the grammar's structural
// punctuation marks rather than a word or operator.
func isPunctuation(s string) bool {
	return s == "{" || s == "}" || s == "[" || s == "]" || s == "," || s == "@" || s == "@@" || s == "~" || s == "="
}

func looksDate(s string) bool {
	return len(s) == 10 && s[4] == '-' && s[7] == '-'
}

func (p *parser) parseDate(t token) Date {
	parts := strings.Split(t.text, "-")
	d := Date{Raw: t.text, Span: t.span}
	if len(parts) != 3 {
		p.add("E-PARSE-DATE", diagnostic.Error, t.span)
		return d
	}
	d.Year, _ = strconv.Atoi(parts[0])
	d.Month, _ = strconv.Atoi(parts[1])
	d.Day, _ = strconv.Atoi(parts[2])
	if !d.Valid() {
		p.add("E-PARSE-DATE", diagnostic.Error, t.span)
	}
	return d
}

func (p *parser) base(ts []token) DirectiveBase {
	if len(ts) == 0 {
		return DirectiveBase{}
	}
	start, end := ts[0].span.Start, ts[len(ts)-1].span.End
	return DirectiveBase{At: p.file.Span(start, end), Raw: p.file.Text(p.file.Span(start, end))}
}

func (p *parser) appendDirective(d Directive) {
	p.out.Directives = append(p.out.Directives, d)
	p.lastDirective = len(p.out.Directives) - 1
}

// attachMetadata attaches an orphan metadata line — one that follows a bare
// directive rather than a transaction or posting — to the most recently
// parsed directive, extending its span and raw text. Every directive type
// shares the "Meta []Metadata + embedded DirectiveBase" shape, so the value
// types are handled uniformly through one reflective write (a copy is made
// addressable, mutated, and stored back); *Transaction is stored by pointer
// and mutates in place.
func (p *parser) attachMetadata(meta Metadata) bool {
	if p.lastDirective < 0 || p.lastDirective >= len(p.out.Directives) {
		return false
	}
	d := p.out.Directives[p.lastDirective]
	if tx, ok := d.(*Transaction); ok {
		tx.Meta = append(tx.Meta, meta)
		extendDirective(&tx.DirectiveBase, meta.Span, p.file)
		return true
	}
	value := reflect.New(reflect.TypeOf(d)).Elem()
	value.Set(reflect.ValueOf(d))
	metaField := value.FieldByName("Meta")
	baseField := value.FieldByName("DirectiveBase")
	if !metaField.IsValid() || metaField.Kind() != reflect.Slice || !baseField.IsValid() {
		return false
	}
	metaField.Set(reflect.Append(metaField, reflect.ValueOf(meta)))
	extendDirective(baseField.Addr().Interface().(*DirectiveBase), meta.Span, p.file)
	p.out.Directives[p.lastDirective] = value.Interface().(Directive)
	return true
}

func extendDirective(base *DirectiveBase, span source.Span, file *source.SourceFile) {
	if base == nil || span.End <= base.At.End {
		return
	}
	base.At.End = span.End
	base.At.EndLine, base.At.EndColumn = span.EndLine, span.EndColumn
	base.Raw = file.Text(base.At)
}

func (p *parser) add(code string, sev diagnostic.Severity, span source.Span) {
	d := diagnostic.New(code, sev, span).WithPath(p.file.Path)
	p.bag.Add(d)
}

// addf is add with a formatted message.
func (p *parser) addf(code string, sev diagnostic.Severity, span source.Span, format string, args ...any) {
	d := diagnostic.New(code, sev, span, fmt.Sprintf(format, args...)).WithPath(p.file.Path)
	p.bag.Add(d)
}
