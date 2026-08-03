// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"math/big"
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
		start := i
		if r == '"' {
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
			out = append(out, token{kind: tokString, text: raw, value: value, span: f.Span(ln.start+start, ln.start+i)})
			if !closed {
				out = append(out, token{kind: tokPunct, text: "<unterminated>", span: f.Span(ln.start+i, ln.start+len(text))})
			}
			continue
		}
		if strings.ContainsRune("{}[](),@~=*", r) && !(r == ',' && i > 0 && i+1 < len(text) && isASCIIDigit(text[i-1]) && isASCIIDigit(text[i+1])) {
			// @@ is one operator; the cost delimiters are represented by two
			// adjacent braces and interpreted by the posting parser.
			if text[i] == '@' && i+1 < len(text) && text[i+1] == '@' {
				i += 2
				out = append(out, token{kind: tokPunct, text: "@@", span: f.Span(ln.start+start, ln.start+i)})
				continue
			}
			i++
			out = append(out, token{kind: tokPunct, text: text[start:i], span: f.Span(ln.start+start, ln.start+i)})
			continue
		}
		for i < len(text) {
			r, size = utf8.DecodeRuneInString(text[i:])
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
			// Invalid UTF-8 is diagnosed by Parse, but tokenization must still
			// make progress over every byte so malformed input cannot hang.
			if size <= 0 {
				size = 1
			}
			i += size
		}
		out = append(out, token{kind: tokWord, text: text[start:i], value: text[start:i], span: f.Span(ln.start+start, ln.start+i)})
	}
	return out
}

func isASCIIDigit(value byte) bool { return value >= '0' && value <= '9' }

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

func (p *parser) parseDirective(ln line, ts []token) {
	if len(ts) == 0 {
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

func (p *parser) parseKeyword(ts []token) {
	if len(ts) == 0 {
		return
	}
	base := p.base(ts)
	switch strings.ToLower(ts[0].text) {
	case "option":
		if len(ts) != 3 || ts[1].kind != tokString || ts[2].kind != tokString {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		p.appendDirective(Option{DirectiveBase: base, Key: ts[1].value, Value: ts[2].value})
	case "plugin":
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
	case "include":
		if len(ts) != 2 || ts[1].kind != tokString {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		p.appendDirective(Include{DirectiveBase: base, Path: ts[1].value})
	case "pushtag", "poptag":
		if len(ts) != 2 || !strings.HasPrefix(ts[1].text, "#") {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		p.appendDirective(TagDirective{DirectiveBase: base, Tag: strings.TrimPrefix(ts[1].text, "#")})
	}
}

func (p *parser) parseDateDirective(keyword string, date Date, rest []token, dateSpan source.Span) {
	base := p.base(append([]token{{span: dateSpan}}, rest...))
	if len(rest) == 0 {
		p.add("E-PARSE-EXPECTED", diagnostic.Error, dateSpan)
		return
	}
	base.At.Start = dateSpan.Start
	switch keyword {
	case "open":
		d := Open{DirectiveBase: base, Date: date}
		if len(rest) < 1 {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		d.Account = rest[0].text
		for _, t := range rest[1:] {
			if t.kind == tokString {
				d.Booking = t.value
			} else if t.text != "," {
				d.Currencies = append(d.Currencies, t.text)
			}
		}
		p.appendDirective(d)
	case "close":
		if len(rest) < 1 {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		p.appendDirective(Close{DirectiveBase: base, Date: date, Account: rest[0].text})
	case "commodity":
		if len(rest) < 1 {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		p.appendDirective(Commodity{DirectiveBase: base, Date: date, Currency: rest[0].text})
	case "balance":
		d := Balance{DirectiveBase: base, Date: date}
		if len(rest) < 3 {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		d.Account = rest[0].text
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
	case "pad":
		if len(rest) < 2 {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		p.appendDirective(Pad{DirectiveBase: base, Date: date, Account: rest[0].text, SourceAccount: rest[1].text})
	case "event":
		if len(rest) < 2 || rest[0].kind != tokString || rest[1].kind != tokString {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		p.appendDirective(Event{DirectiveBase: base, Date: date, Type: rest[0].value, Value: rest[1].value})
	case "query":
		if len(rest) < 2 || rest[0].kind != tokString || rest[1].kind != tokString {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		p.appendDirective(Query{DirectiveBase: base, Date: date, Name: rest[0].value, Query: rest[1].value})
	case "price":
		if len(rest) < 3 {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		amount, _, ok := p.amount(rest, 1)
		if !ok {
			return
		}
		p.appendDirective(Price{DirectiveBase: base, Date: date, Currency: rest[0].text, Amount: amount})
	case "document":
		d := Document{DirectiveBase: base, Date: date}
		if len(rest) < 2 {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		d.Account = rest[0].text
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
	case "note":
		if len(rest) < 2 || rest[1].kind != tokString {
			p.add("E-PARSE-EXPECTED", diagnostic.Error, base.At)
			return
		}
		p.appendDirective(Note{DirectiveBase: base, Date: date, Account: rest[0].text, Comment: rest[1].value})
	case "custom":
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
}

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
	for i < len(ts) {
		switch ts[i].text {
		case "{":
			cost, next := p.cost(ts, i)
			post.Cost, i = &cost, next
		case "@", "@@":
			spec := PriceSpec{At: ts[i].span, Total: ts[i].text == "@@"}
			if amount, next, ok := p.amount(ts, i+1); ok {
				spec.Amount = amount
				post.Price, i = &spec, next
			} else {
				return post, false
			}
		default:
			if ts[i].text == "=" || ts[i].text == "~" {
				i++
				if _, next, ok := p.amount(ts, i); ok {
					i = next
				}
			} else {
				p.add("E-PARSE-TOKEN", diagnostic.Error, ts[i].span)
				i++
			}
		}
	}
	return post, true
}

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

func (p *parser) value(ts []token, i int) (Value, int) {
	if i >= len(ts) {
		return Value{Kind: ValueInvalid}, i
	}
	t := ts[i]
	if t.text == "[" {
		v := Value{Kind: ValueList, At: t.span}
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
		if i+1 < len(ts) && !isPunctuation(ts[i+1].text) {
			a := Amount{Number: n, Currency: ts[i+1].text, At: p.file.Span(t.span.Start, ts[i+1].span.End)}
			return Value{Kind: ValueAmount, Raw: p.file.Text(a.At), Amount: a, At: a.At}, i + 2
		}
		return Value{Kind: ValueNumber, Raw: t.text, Number: n, At: t.span}, i + 1
	}
	if isAccount(t.text) {
		return Value{Kind: ValueAccount, Raw: t.text, String: t.text, At: t.span}, i + 1
	}
	return Value{Kind: ValueCurrency, Raw: t.text, String: t.text, At: t.span}, i + 1
}

func isAccount(s string) bool {
	if !strings.Contains(s, ":") {
		return false
	}
	first := s[0]
	return first >= 'A' && first <= 'Z'
}

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

func (p *parser) attachMetadata(meta Metadata) bool {
	if p.lastDirective < 0 || p.lastDirective >= len(p.out.Directives) {
		return false
	}
	d := p.out.Directives[p.lastDirective]
	switch x := d.(type) {
	case Option:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Plugin:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Include:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case TagDirective:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Open:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Close:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Commodity:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Balance:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Pad:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Event:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Query:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Price:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Document:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Note:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case Custom:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
		p.out.Directives[p.lastDirective] = x
	case *Transaction:
		x.Meta = append(x.Meta, meta)
		extendDirective(&x.DirectiveBase, meta.Span, p.file)
	default:
		return false
	}
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
