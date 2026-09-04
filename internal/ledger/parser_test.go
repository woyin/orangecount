// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ledger

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"orangecount/internal/diagnostic"
	"orangecount/internal/source"
)

func TestParseCoreDirectivesAndTransaction(t *testing.T) {
	text := `option "title" "Demo"
plugin "example.plugin" "cfg"
include "other.bean"
pushtag #global
2000-01-01 open Assets:Cash USD, EUR
2000-01-01 commodity USD
2000-01-01 * "Payee" "Lunch" #food ^id
  Assets:Cash  -12.50 USD {12.50 USD} @ 1 USD
    source: "fixture"
  Expenses:Food  12.50 USD
2000-01-02 balance Assets:Cash 100 USD ~ 0.01 USD
2000-01-03 close Assets:Cash
2000-01-04 pad Assets:Cash Equity:Opening-Balances
2000-01-05 event "location" "home"
2000-01-06 query "q" "SELECT 1"
2000-01-07 price USD 1.2 EUR
2000-01-08 document Assets:Cash "receipt.pdf" #tag ^link
2000-01-09 note Assets:Cash "memo"
2000-01-10 custom "foo" 1 USD TRUE 2000-01-01
`
	f, bag := ParseText("fixture.bean", []byte(text))
	if bag.Len() == 0 {
		t.Fatalf("expected plugin migration warning at minimum")
	}
	if !bagHasCode(bag.All(), "W-PLUGIN-MIGRATION") {
		t.Fatalf("diagnostics=%+v", bag.All())
	}
	if len(f.Directives) != 16 {
		t.Fatalf("directives=%d %#v", len(f.Directives), f.Directives)
	}
	if got, ok := f.Directives[3].(TagDirective); !ok || got.Kind() != KindPushTag {
		t.Fatalf("pushtag=%#v", f.Directives[3])
	}
	txPtr, ok := f.Directives[6].(*Transaction)
	var tx Transaction
	if ok {
		tx = *txPtr
	}
	if !ok || len(tx.Postings) != 2 || tx.Payee != "Payee" || tx.Narration != "Lunch" || len(tx.Postings[0].Meta) != 1 {
		t.Fatalf("transaction=%#v", f.Directives[6])
	}
	event, ok := f.Directives[10].(Event)
	if !ok || event.Type != "location" || event.Value != "home" {
		t.Fatalf("event=%#v", f.Directives[10])
	}
	query, ok := f.Directives[11].(Query)
	if !ok || query.Name != "q" || query.Query != "SELECT 1" {
		t.Fatalf("query=%#v", f.Directives[11])
	}
}

func TestParseRecovery(t *testing.T) {
	text := `2000-01-01 open
2000-01-02 open Assets:Cash USD
not a directive
2000-01-03 close Assets:Cash
`
	f, bag := ParseText("bad.bean", []byte(text))
	if len(f.Directives) != 2 || bag.Len() < 2 {
		t.Fatalf("directives=%d diagnostics=%+v", len(f.Directives), bag.All())
	}
}

func TestParserAcceptsEmptyPayeeTransaction(t *testing.T) {
	file, diagnostics := ParseText("empty-payee.bean", []byte("2000-01-01 \"\" \"narration\" #tag\n"))
	if diagnostics.HasErrors() || len(file.Directives) != 1 {
		t.Fatalf("diagnostics=%+v directives=%d", diagnostics.All(), len(file.Directives))
	}
}

func TestParserAcceptsDateFlagWithoutWhitespace(t *testing.T) {
	file, diagnostics := ParseText("compact-flag.bean", []byte("2000-01-01* \"narration\"\n"))
	if diagnostics.HasErrors() || len(file.Directives) != 1 {
		t.Fatalf("diagnostics=%+v directives=%d", diagnostics.All(), len(file.Directives))
	}
}

func TestParserKeepsGroupedNumberAsOneAmount(t *testing.T) {
	file, diagnostics := ParseText("grouped-number.bean", []byte("2000-01-01 * \"narration\"\n  Assets:Cash 1,234.56 USD\n"))
	if diagnostics.HasErrors() || len(file.Directives) != 1 {
		t.Fatalf("diagnostics=%+v directives=%d", diagnostics.All(), len(file.Directives))
	}
	tx, ok := file.Directives[0].(*Transaction)
	if !ok || len(tx.Postings) != 1 || tx.Postings[0].Units == nil || tx.Postings[0].Units.Number.Raw != "1234.56" {
		t.Fatalf("transaction=%#v", file.Directives[0])
	}
}

func TestParserKeepsIncompletePostingAmountSides(t *testing.T) {
	text := `2000-01-01 * "narration"
  Assets:Cash USD {} @ 1 USD
  Assets:Shares 1 {} @ 2 USD
  Equity:Opening -100 USD
`
	file, diagnostics := ParseText("incomplete-amount.bean", []byte(text))
	if diagnostics.HasErrors() || len(file.Directives) != 1 {
		t.Fatalf("directives=%#v diagnostics=%+v", file.Directives, diagnostics.All())
	}
	tx, ok := file.Directives[0].(*Transaction)
	if !ok || len(tx.Postings) != 3 {
		t.Fatalf("transaction=%#v", file.Directives[0])
	}
	if units := tx.Postings[0].Units; units == nil || units.Number.Raw != "" || units.Currency != "USD" {
		t.Fatalf("currency-only units=%#v", units)
	}
	if units := tx.Postings[1].Units; units == nil || units.Number.Raw != "1" || units.Currency != "" {
		t.Fatalf("number-only units=%#v", units)
	}
}

func TestParseCommentsAndEscapedStrings(t *testing.T) {
	f, bag := ParseText("comments.bean", []byte("; header\noption \"title\" \"a; b\\n\" ; inline\n"))
	if bag.Len() != 0 || len(f.Comments) != 2 {
		t.Fatalf("comments=%+v diagnostics=%+v", f.Comments, bag.All())
	}
	option, ok := f.Directives[0].(Option)
	if !ok || option.Value != "a; b\n" {
		t.Fatalf("option=%#v", f.Directives[0])
	}
}

func TestDirectiveMetadata(t *testing.T) {
	f, bag := ParseText("meta.bean", []byte("2000-01-01 open Assets:Cash USD\n  owner: \"cash\"\n"))
	if bag.Len() != 0 || len(f.Directives) != 1 {
		t.Fatalf("directives=%#v diagnostics=%+v", f.Directives, bag.All())
	}
	open, ok := f.Directives[0].(Open)
	if !ok || len(open.Meta) != 1 || open.Meta[0].Key != "owner" || open.Span().EndLine != 2 {
		t.Fatalf("open=%#v", f.Directives[0])
	}
}

func TestParseGraph(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	child := filepath.Join(dir, "child.bean")
	if err := os.WriteFile(entry, []byte("include \"child.bean\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(child, []byte("option \"title\" \"ok\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	graph, err := source.LoadGraph(entry)
	if err != nil {
		t.Fatal(err)
	}
	parsed, bag := ParseGraph(graph)
	if len(parsed) != 2 || bag.Len() != 0 {
		t.Fatalf("parsed=%d diagnostics=%+v", len(parsed), bag.All())
	}
}

func TestParseRejectsInvalidUTF8(t *testing.T) {
	_, bag := ParseText("invalid.bean", []byte{0xff, '\n'})
	if !bagHasCode(bag.All(), "E-SOURCE-UTF8") {
		t.Fatalf("diagnostics=%+v", bag.All())
	}
}

func TestParserPreservesTypedCustomValuesAndDirectiveMetadata(t *testing.T) {
	text := `option "title" "Demo"
  key: "option"
plugin "example.plugin"
  key: "plugin"
include "child.bean"
  key: "include"
pushtag #tag
  key: "tag"
2000-01-01 open Assets:Cash USD
  key: "open"
2000-01-01 close Assets:Cash
  key: "close"
2000-01-01 commodity USD
  key: "commodity"
2000-01-01 balance Assets:Cash 1 USD
  key: "balance"
2000-01-01 pad Assets:Cash Equity:Opening
  key: "pad"
2000-01-01 event "kind" "value"
  key: "event"
2000-01-01 query "name" "SELECT 1"
  key: "query"
2000-01-01 price USD 1 EUR
  key: "price"
2000-01-01 document Assets:Cash "receipt.pdf"
  key: "document"
2000-01-01 note Assets:Cash "memo"
  key: "note"
2000-01-01 custom "all" "text" 1 TRUE 2000-01-01 Assets:Cash USD #tag ^link [1, "two"]
  key: "custom"
2000-01-01 * "transaction"
  Assets:Cash 1 USD
    key: "posting"
  Equity:Opening -1 USD
`
	file, diagnostics := ParseText("metadata-all.bean", []byte(text))
	if diagnostics.HasErrors() || len(file.Directives) != 16 {
		t.Fatalf("diagnostics=%+v directives=%d", diagnostics.All(), len(file.Directives))
	}
	for _, directive := range file.Directives[:15] {
		switch value := directive.(type) {
		case Option:
			if len(value.Meta) != 1 {
				t.Errorf("option metadata=%+v", value.Meta)
			}
		case Plugin:
			if len(value.Meta) != 1 {
				t.Errorf("plugin metadata=%+v", value.Meta)
			}
		case Include:
			if len(value.Meta) != 1 {
				t.Errorf("include metadata=%+v", value.Meta)
			}
		case TagDirective:
			if len(value.Meta) != 1 {
				t.Errorf("tag metadata=%+v", value.Meta)
			}
		case Open:
			if len(value.Meta) != 1 {
				t.Errorf("open metadata=%+v", value.Meta)
			}
		case Close:
			if len(value.Meta) != 1 {
				t.Errorf("close metadata=%+v", value.Meta)
			}
		case Commodity:
			if len(value.Meta) != 1 {
				t.Errorf("commodity metadata=%+v", value.Meta)
			}
		case Balance:
			if len(value.Meta) != 1 {
				t.Errorf("balance metadata=%+v", value.Meta)
			}
		case Pad:
			if len(value.Meta) != 1 {
				t.Errorf("pad metadata=%+v", value.Meta)
			}
		case Event:
			if len(value.Meta) != 1 {
				t.Errorf("event metadata=%+v", value.Meta)
			}
		case Query:
			if len(value.Meta) != 1 {
				t.Errorf("query metadata=%+v", value.Meta)
			}
		case Price:
			if len(value.Meta) != 1 {
				t.Errorf("price metadata=%+v", value.Meta)
			}
		case Document:
			if len(value.Meta) != 1 {
				t.Errorf("document metadata=%+v", value.Meta)
			}
		case Note:
			if len(value.Meta) != 1 {
				t.Errorf("note metadata=%+v", value.Meta)
			}
		case Custom:
			if len(value.Meta) != 1 || len(value.Values) != 8 || value.Values[7].Kind != ValueList {
				t.Errorf("custom=%+v", value)
			}
		}
	}
	transaction, ok := file.Directives[15].(*Transaction)
	if !ok || len(transaction.Postings) != 2 || len(transaction.Postings[0].Meta) != 1 {
		t.Fatalf("transaction=%+v", file.Directives[15])
	}
}

func TestParserRecoversAcrossMalformedGrammarAndPreservesValidNeighbors(t *testing.T) {
	text := `option title "missing quotes"
plugin "module" 9
include child.bean
pushtag tag
2000-02-30 open Assets:Bad USD
2000-01-01 balance Assets:Cash not-a-number USD
2000-01-01 pad Assets:Cash
2000-01-01 event kind "value"
2000-01-01 query "name" query
2000-01-01 price USD no-number EUR
2000-01-01 note Assets:Cash missing
2000-01-01 * "one" "two" "three" unexpected
  Assets:Cash 1 USD {{ 2 USD, 2000-01-01, "lot" }} @@ 3 USD = 1 USD ~ 0.1 USD trailing
    too-deep 1 USD
  metadata without colon
  key:
2000-01-02 * "survives"
  Assets:Cash 1 USD
  Equity:Opening -1 USD
`
	file, diagnostics := ParseText("grammar-edges.bean", []byte(text))
	if !diagnostics.HasErrors() {
		t.Fatal("malformed grammar produced no diagnostics")
	}
	if len(file.Directives) == 0 {
		t.Fatal("parser discarded all directives after recovery")
	}
	last, ok := file.Directives[len(file.Directives)-1].(*Transaction)
	if !ok || last.Narration != "survives" || len(last.Postings) != 2 {
		t.Fatalf("recovery transaction=%#v", file.Directives[len(file.Directives)-1])
	}
	for _, code := range []string{"E-PARSE-EXPECTED", "E-PARSE-DATE", "E-PARSE-TOKEN"} {
		if !bagHasCode(diagnostics.All(), code) {
			t.Errorf("diagnostics missing %s: %+v", code, diagnostics.All())
		}
	}
}

func TestParserValueAndGraphEdgeContracts(t *testing.T) {
	file := source.NewSourceFile(7, "values.bean", []byte("[1, true, 2000-01-01, Assets:Cash, USD, #tag, ^link]\nkey: \"value\"\n"))
	p := &parser{file: file, bag: new(diagnostic.Bag)}
	listTokens := tokenize(file, splitLines(file)[0])
	value, next := p.value(listTokens, 0)
	if value.Kind != ValueList || next != len(listTokens) || len(value.List) != 7 {
		t.Fatalf("list value=%+v next=%d tokens=%d", value, next, len(listTokens))
	}
	if got := []ValueKind{value.List[0].Kind, value.List[1].Kind, value.List[2].Kind, value.List[3].Kind, value.List[4].Kind, value.List[5].Kind, value.List[6].Kind}; !reflect.DeepEqual(got, []ValueKind{ValueNumber, ValueBool, ValueDate, ValueAccount, ValueCurrency, ValueTag, ValueLink}) {
		t.Fatalf("value kinds=%v", got)
	}
	metaTokens := tokenize(file, splitLines(file)[1])
	metadata, ok := p.parseMetadata(metaTokens)
	if !ok || metadata.Key != "key" || metadata.Value.Kind != ValueString {
		t.Fatalf("metadata=%+v ok=%v", metadata, ok)
	}
	empty := tokenize(file, splitLines(file)[1])[:1]
	blankMeta, blankOK := p.parseMetadata(empty)
	if !blankOK || blankMeta.Key != "key" || blankMeta.Value.Kind != ValueNull {
		t.Fatalf("empty metadata=%+v ok=%v", blankMeta, blankOK)
	}
	if _, ok := p.parseMetadata(empty[:0]); ok {
		t.Fatal("empty metadata token list accepted")
	}

	if parsed, bag := ParseGraph(nil); len(parsed) != 0 || bag.Len() != 0 {
		t.Fatalf("nil graph parsed=%v diagnostics=%+v", parsed, bag.All())
	}
	graph := &source.Graph{
		Order: []source.FileID{7, 99},
		Files: map[source.FileID]*source.SourceFile{7: file},
		Diagnostics: []source.GraphIssue{
			{Code: "W-GRAPH-WARNING", Path: "display.bean", Related: []source.Span{{File: 7}}},
			{Code: "E-GRAPH-ERROR", SourcePath: "source.bean"},
		},
	}
	parsed, bag := ParseGraph(graph)
	if len(parsed) != 1 || !bagHasCode(bag.All(), "W-GRAPH-WARNING") || !bagHasCode(bag.All(), "E-GRAPH-ERROR") {
		t.Fatalf("graph parsed=%d diagnostics=%+v", len(parsed), bag.All())
	}
}

func TestParserAcceptsV3MetadataStackAndNoneValues(t *testing.T) {
	text := `pushmeta owner: "alice"
pushmeta note: None
2000-01-01 open Assets:Cash USD
  empty:
  none: None
popmeta note:
popmeta owner:
`
	file, diagnostics := ParseText("meta-stack.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics=%+v", diagnostics.All())
	}
	if len(file.Directives) != 5 {
		t.Fatalf("directives=%d %#v", len(file.Directives), file.Directives)
	}
	push, ok := file.Directives[0].(PushMeta)
	if !ok || push.Key != "owner" || push.Value.Kind != ValueString || push.Value.String != "alice" {
		t.Fatalf("pushmeta=%#v", file.Directives[0])
	}
	pushNone, ok := file.Directives[1].(PushMeta)
	if !ok || pushNone.Key != "note" || pushNone.Value.Kind != ValueNull {
		t.Fatalf("pushmeta none=%#v", file.Directives[1])
	}
	open, ok := file.Directives[2].(Open)
	if !ok || len(open.Meta) != 2 || open.Meta[0].Value.Kind != ValueNull || open.Meta[1].Value.Kind != ValueNull {
		t.Fatalf("open metadata=%#v", file.Directives[2])
	}
	popNote, ok := file.Directives[3].(PopMeta)
	if !ok || popNote.Key != "note" {
		t.Fatalf("popmeta note=%#v", file.Directives[3])
	}
	popOwner, ok := file.Directives[4].(PopMeta)
	if !ok || popOwner.Key != "owner" {
		t.Fatalf("popmeta owner=%#v", file.Directives[4])
	}
}

func TestParserAcceptsHashFlagAndNoteTagsLinks(t *testing.T) {
	text := `2000-01-01 # "hash flag"
  Assets:Cash 1 USD
  Equity:Opening -1 USD
2000-01-02 note Assets:Cash "memo" #tag ^link
`
	file, diagnostics := ParseText("hash-flag.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics=%+v", diagnostics.All())
	}
	if len(file.Directives) != 2 {
		t.Fatalf("directives=%d %#v", len(file.Directives), file.Directives)
	}
	tx, ok := file.Directives[0].(*Transaction)
	if !ok || tx.Flag != "#" || tx.Narration != "hash flag" {
		t.Fatalf("transaction=%#v", file.Directives[0])
	}
	note, ok := file.Directives[1].(Note)
	if !ok || len(note.Tags) != 1 || note.Tags[0] != "tag" || len(note.Links) != 1 || note.Links[0] != "link" {
		t.Fatalf("note=%#v", file.Directives[1])
	}
}

func TestParserEvaluatesNumberExpressions(t *testing.T) {
	text := `2000-01-01 * "expr"
  Assets:Cash 1 + 2 USD
  Equity:Opening -3 USD
2000-01-02 balance Assets:Cash 1 + 2 USD ~ 1 / 2 USD
2000-01-03 custom "expr" 1 + 2 USD (1 + 2) * 3
2000-01-04 custom "expr" -(1 / 3) USD
`
	file, diagnostics := ParseText("expr.bean", []byte(text))
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics=%+v", diagnostics.All())
	}
	if len(file.Directives) != 4 {
		t.Fatalf("directives=%d %#v", len(file.Directives), file.Directives)
	}
	tx, ok := file.Directives[0].(*Transaction)
	if !ok || len(tx.Postings) != 2 {
		t.Fatalf("transaction=%#v", file.Directives[0])
	}
	if units := tx.Postings[0].Units; units == nil || units.Number.Raw != "3" || units.Currency != "USD" {
		t.Fatalf("posting units=%#v", units)
	}
	balance, ok := file.Directives[1].(Balance)
	if !ok || balance.Amount.Number.Raw != "3" || balance.Tolerance == nil || balance.Tolerance.Raw != "0.5" {
		t.Fatalf("balance=%#v", file.Directives[1])
	}
	custom, ok := file.Directives[2].(Custom)
	if !ok || len(custom.Values) != 2 || custom.Values[0].Kind != ValueAmount || custom.Values[0].Amount.Number.Raw != "3" || custom.Values[1].Kind != ValueNumber || custom.Values[1].Number.Raw != "9" {
		t.Fatalf("custom=%#v", file.Directives[2])
	}
	neg, ok := file.Directives[3].(Custom)
	if !ok || len(neg.Values) != 1 || neg.Values[0].Kind != ValueAmount || neg.Values[0].Amount.Number.Raw != "-1/3" {
		t.Fatalf("negative custom=%#v", file.Directives[3])
	}
}

func TestParserRejectsLeadingWordsBeforeNumberExpressions(t *testing.T) {
	text := `2000-01-01 * "bad amount"
  Assets:Cash Extra 1 USD
  Equity:Opening -1 USD
`
	_, diagnostics := ParseText("leading-word.bean", []byte(text))
	if !diagnostics.HasErrors() {
		t.Fatalf("leading non-number word before an amount expression produced no diagnostics: %+v", diagnostics.All())
	}
	if !bagHasCode(diagnostics.All(), "E-PARSE-TOKEN") {
		t.Fatalf("expected E-PARSE-TOKEN diagnostic, got %+v", diagnostics.All())
	}
}

func TestParserSplitsJuxtaposedNumberValuesWithoutOperator(t *testing.T) {
	file, diagnostics := ParseText("custom-values.bean", []byte(`2000-01-01 custom "expr" 1 2 USD
`))
	if diagnostics.HasErrors() {
		t.Fatalf("diagnostics=%+v", diagnostics.All())
	}
	custom, ok := file.Directives[0].(Custom)
	if !ok || len(custom.Values) != 2 || custom.Values[0].Kind != ValueNumber || custom.Values[0].Number.Raw != "1" || custom.Values[1].Kind != ValueAmount || custom.Values[1].Amount.Number.Raw != "2" || custom.Values[1].Amount.Currency != "USD" {
		t.Fatalf("custom=%#v", file.Directives[0])
	}
}

func TestParserInternalRecoveryContracts(t *testing.T) {
	if parsed, diagnostics := Parse(nil); parsed.Source != nil || diagnostics.Len() != 0 {
		t.Fatalf("nil source parse=%+v diagnostics=%+v", parsed, diagnostics.All())
	}
	file := source.NewSourceFile(9, "internal-branches.bean", []byte("unterminated \"string\n  Assets:Cash 1 USD\n"))
	p := &parser{file: file, bag: new(diagnostic.Bag), out: &File{Source: file}, lastDirective: -1}
	lines := splitLines(file)
	if tokens := tokenize(file, lines[0]); len(tokens) < 2 || tokens[len(tokens)-1].text != "<unterminated>" {
		t.Fatalf("unterminated tokens=%+v", tokens)
	}
	p.parseContinuation(lines[1], tokenize(file, lines[1]))
	if !bagHasCode(p.bag.All(), "E-PARSE-EXPECTED") {
		t.Fatalf("orphan continuation diagnostics=%+v", p.bag.All())
	}
	p.tx = &Transaction{}
	p.posting = &Posting{}
	p.parseContinuation(line{indent: 4}, tokenize(file, lines[1]))
	// Exercise incomplete amounts, punctuation separators, and an invalid
	// numeric amount directly at the parser boundary. These are the recovery
	// cases that keep a later top-level directive parseable.
	for _, raw := range []string{"USD", "1", "bad \"quoted\"", "1 @"} {
		probe := source.NewSourceFile(10, "amount.bean", []byte(raw))
		probeParser := &parser{file: probe, bag: new(diagnostic.Bag), out: &File{Source: probe}, lastDirective: -1}
		tokens := tokenize(probe, splitLines(probe)[0])
		_, _, _ = probeParser.amount(tokens, 0)
	}
	for _, raw := range []string{"2000-01", "bad", "2000-02-30"} {
		probe := source.NewSourceFile(11, "date.bean", []byte(raw))
		probeParser := &parser{file: probe, bag: new(diagnostic.Bag), out: &File{Source: probe}, lastDirective: -1}
		probeParser.parseDate(tokenize(probe, splitLines(probe)[0])[0])
	}
	if base := p.base(nil); base.At.Valid() || base.Raw != "" || len(base.Meta) != 0 {
		t.Fatalf("empty base=%+v", base)
	}
	extendDirective(nil, source.Span{}, file)
	base := DirectiveBase{At: source.Span{End: 10}}
	extendDirective(&base, source.Span{End: 5}, file)
	if base.At.End != 10 {
		t.Fatal("shorter metadata span extended directive")
	}
	for _, raw := range []string{"key", "key :", "key value"} {
		probe := source.NewSourceFile(12, "metadata.bean", []byte(raw))
		probeParser := &parser{file: probe, bag: new(diagnostic.Bag), out: &File{Source: probe}, lastDirective: -1}
		_, _ = probeParser.parseMetadata(tokenize(probe, splitLines(probe)[0]))
	}
}

func bagHasCode(ds []diagnostic.Diagnostic, code string) bool {
	for _, d := range ds {
		if d.Code == code {
			return true
		}
	}
	return false
}
