// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package quickentry

import (
	"fmt"
	"regexp"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/web/favaadapter"
)

// CompileRequest carries the per-batch inputs the compiler needs. The web
// layer assembles it from the JSON payload and the current snapshot.
type CompileRequest struct {
	// Text is the raw shorthand, one transaction per non-empty line.
	Text string
	// Date applies to every line in the batch (YYYY-MM-DD).
	Date string
	// Flag is "*" or "!"; empty defaults to "*".
	Flag string
	// OperatingCurrency is the ledger's operating_currency when exactly one
	// is configured. When empty and no per-line currency or template
	// currency applies, the line is rejected (no probabilistic default).
	OperatingCurrency string
	// Evaluation supplies the quick-entry profile directives.
	Evaluation *ledger.Evaluation
}

// LineResult is the compile outcome for one input line. A line is either
// fully compiled (Entry populated) or carries at least one Error.
type LineResult struct {
	Line      int    // 1-based line number in the input text
	Source    string // the original shorthand text
	Entry     *favaadapter.NewEntry
	Preview   string                  // canonical Beancount block, when compiled
	Duplicate bool                    // equivalent transaction exists in the ledger
	Errors    []LineError
}

// LineError explains why a line did not compile.
type LineError struct {
	Code    string
	Message string
}

// Compile parses and compiles a batch of quick-entry shorthand. Every
// non-empty line is processed independently; the result length matches the
// number of non-empty input lines. The caller (web layer) decides whether
// any error aborts the batch preview or commit.
func Compile(req CompileRequest) []LineResult {
	date := strings.TrimSpace(req.Date)
	if !compileDateRegex.MatchString(date) {
		// Surface a synthetic first-line error so the caller can show it
		// prominently; the batch cannot compile without a valid date.
		return []LineResult{{
			Line: 1, Source: strings.TrimSpace(req.Text),
			Errors: []LineError{{Code: "E-QUICK-DATE", Message: "invalid batch date (expected YYYY-MM-DD)"}},
		}}
	}
	txnDate := ledger.Date{
		Year: atoiOrZero(date[0:4]), Month: atoiOrZero(date[5:7]), Day: atoiOrZero(date[8:10]),
		Raw: date,
	}
	if !txnDate.Valid() {
		return []LineResult{{
			Line: 1, Source: strings.TrimSpace(req.Text),
			Errors: []LineError{{Code: "E-QUICK-DATE", Message: "invalid batch date (expected YYYY-MM-DD)"}},
		}}
	}
	flag := strings.TrimSpace(req.Flag)
	if flag == "" {
		flag = "*"
	}
	if flag != "*" && flag != "!" {
		return []LineResult{{
			Line: 1, Source: strings.TrimSpace(req.Text),
			Errors: []LineError{{Code: "E-QUICK-FLAG", Message: "invalid flag (expected * or !)"}},
		}}
	}

	profile := EffectiveProfile(req.Evaluation, txnDate)
	var results []LineResult
	for lineNo, raw := range splitLines(req.Text) {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		result := compileLine(line, lineNo, txnDate, flag, req.OperatingCurrency, profile)
		if len(result.Errors) == 0 && result.Entry != nil {
			preview, err := favaadapter.SerializeNewEntries([]favaadapter.NewEntry{*result.Entry})
			if err != nil {
				result.Errors = append(result.Errors, LineError{Code: "E-QUICK-SERIALIZE", Message: err.Error()})
				result.Preview = ""
			} else {
				result.Preview = preview
			}
		}
		results = append(results, result)
	}
	if len(results) == 0 {
		return []LineResult{{Line: 1, Source: "", Errors: []LineError{{Code: "E-QUICK-EMPTY", Message: "no quick-entry lines to compile"}}}}
	}
	return results
}

// DetectDuplicates flags transactions in the compiled batch that have an
// equivalent existing entry in the ledger. It is non-blocking per the
// grilling consensus: the caller surfaces a warning but may still publish.
func DetectDuplicates(results []LineResult, evaluation *ledger.Evaluation) {
	if evaluation == nil {
		return
	}
	for i := range results {
		r := &results[i]
		if r.Entry == nil || len(r.Errors) > 0 {
			continue
		}
		if hasEquivalentTransaction(r.Entry, evaluation.Entries) {
			r.Duplicate = true
		}
	}
}

// hasEquivalentTransaction returns true when the ledger already contains a
// transaction with the same date, flag, payee, narration, and postings.
func hasEquivalentTransaction(entry *favaadapter.NewEntry, entries []ledger.EntryRecord) bool {
	for _, record := range entries {
		var tx ledger.Transaction
		switch value := record.Directive.(type) {
		case ledger.Transaction:
			tx = value
		case *ledger.Transaction:
			tx = *value
		default:
			continue
		}
		if record.Date.Raw != entry.Date {
			continue
		}
		if tx.Flag != entry.Flag {
			continue
		}
		if strings.TrimSpace(tx.Payee) != strings.TrimSpace(entry.Payee) {
			continue
		}
		if strings.TrimSpace(tx.Narration) != strings.TrimSpace(entry.Narration) {
			continue
		}
		if len(tx.Postings) != len(entry.Postings) {
			continue
		}
		match := true
		for i, p := range entry.Postings {
			if tx.Postings[i].Account != p.Account {
				match = false
				break
			}
			if tx.Postings[i].Units == nil {
				match = false
				break
			}
			if tx.Postings[i].Units.Number.Raw != p.Amount {
				match = false
				break
			}
			if tx.Postings[i].Units.Currency != p.Currency {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// compileLine handles one line of shorthand, dispatching between the compact
// template-invocation form and the explicit delimited form.
func compileLine(line string, lineNo int, txnDate ledger.Date, flag, operatingCurrency string, profile Profile) LineResult {
	result := LineResult{Line: lineNo, Source: line}
	tokens := tokenize(line)
	if len(tokens) == 0 {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-EMPTY-LINE", Message: "empty line"})
		return result
	}
	// Template invocation: first token exactly matches a defined template name.
	if tmpl, ok := findTemplate(profile, tokens[0].text); ok {
		return compileTemplateInvocation(line, lineNo, tmpl, tokens[1:], txnDate, flag, operatingCurrency, profile)
	}
	// Explicit form: AMOUNT [CURRENCY] @source -> @destination [: narration] [#tag] [^link]
	return compileExplicitForm(line, lineNo, tokens, txnDate, flag, operatingCurrency, profile)
}

// compileTemplateInvocation handles the compact form like "午餐 28 @微信".
// The template supplies defaults; tokens override amount, currency, payee,
// and may supply the unfilled posting alias.
func compileTemplateInvocation(line string, lineNo int, tmpl TemplateRule, rest []token, txnDate ledger.Date, flag, operatingCurrency string, profile Profile) LineResult {
	result := LineResult{Line: lineNo, Source: line}
	amount, currency, payeeAlias, narrationOverride, extraTags, extraLinks, err := parseTemplateRest(rest)
	if err != nil {
		result.Errors = append(result.Errors, *err)
		return result
	}
	if amount == "" {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-AMOUNT", Message: "template invocation requires an amount"})
		return result
	}
	if !compileAmountRegex.MatchString(amount) {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-AMOUNT", Message: "amount must be a positive decimal number"})
		return result
	}
	resolvedCurrency := currency
	if resolvedCurrency == "" {
		resolvedCurrency = tmpl.Currency
	}
	if resolvedCurrency == "" {
		resolvedCurrency = operatingCurrency
	}
	if resolvedCurrency == "" {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-CURRENCY", Message: "no currency supplied and none inferable from template or ledger"})
		return result
	}
	if !compileCurrencyRegex.MatchString(resolvedCurrency) {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-CURRENCY", Message: "invalid currency"})
		return result
	}
	sourceAccount := resolveAccount(tmpl.Source, profile)
	destAccount := resolveAccount(tmpl.Destination, profile)
	if payeeAlias != "" {
		if acc := resolveAccount(payeeAlias, profile); acc != "" {
			if tmpl.Source == "" {
				sourceAccount = acc
			} else if tmpl.Destination == "" {
				destAccount = acc
			}
		}
	}
	if sourceAccount == "" {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-SOURCE", Message: "source account could not be resolved for the template"})
		return result
	}
	if destAccount == "" {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-DEST", Message: "destination account could not be resolved for the template"})
		return result
	}
	narration := tmpl.Narration
	if narrationOverride != "" {
		narration = narrationOverride
	}
	tags := append([]string{}, tmpl.Tags...)
	tags = append(tags, extraTags...)
	links := append([]string{}, tmpl.Links...)
	links = append(links, extraLinks...)
	result.Entry = buildEntry(txnDate, flag, tmpl.Payee, narration, sourceAccount, destAccount, amount, resolvedCurrency, tags, links)
	return result
}

func parseTemplateRest(rest []token) (amount, currency, payeeAlias, narration string, tags, links []string, err *LineError) {
	for _, t := range rest {
		switch {
		case t.kind == tokHash:
			tags = append(tags, t.text[1:])
		case t.kind == tokLink:
			links = append(links, t.text[1:])
		case t.kind == tokColon:
			narration = strings.TrimSpace(t.text[1:])
		case t.kind == tokAt:
			payeeAlias = t.text[1:]
		case t.kind == tokWord && compileAmountRegex.MatchString(t.text):
			if amount == "" {
				amount = t.text
			} else if currency == "" {
				currency = t.text
			}
		case t.kind == tokWord && compileCurrencyRegex.MatchString(t.text):
			if currency == "" {
				currency = t.text
			} else {
				return "", "", "", "", nil, nil, &LineError{Code: "E-QUICK-TEMPLATE-TOKEN", Message: "unexpected token in template invocation"}
			}
		default:
			return "", "", "", "", nil, nil, &LineError{Code: "E-QUICK-TEMPLATE-TOKEN", Message: "unexpected token in template invocation"}
		}
	}
	return amount, currency, payeeAlias, narration, tags, links, nil
}

// compileExplicitForm handles "28 CNY @微信 -> @餐饮 : 工作午餐 #tag".
func compileExplicitForm(line string, lineNo int, tokens []token, txnDate ledger.Date, flag, operatingCurrency string, profile Profile) LineResult {
	result := LineResult{Line: lineNo, Source: line}
	var amount, currency, sourceAlias, destAlias, narration string
	var tags, links []string
	arrowIdx := -1
	for i, t := range tokens {
		if t.kind == tokArrow {
			arrowIdx = i
			break
		}
	}
	if arrowIdx < 0 {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-ARROW", Message: "explicit form requires -> between source and destination"})
		return result
	}
	// Left side: amount [currency] [@sourceAlias]
	left := tokens[:arrowIdx]
	right := tokens[arrowIdx+1:]
	for _, t := range left {
		switch t.kind {
		case tokWord:
			if amount == "" {
				amount = t.text
			} else if currency == "" {
				currency = t.text
			} else {
				result.Errors = append(result.Errors, LineError{Code: "E-QUICK-LEFT-TOKENS", Message: "too many tokens before ->"})
				return result
			}
		case tokAt:
			if sourceAlias != "" {
				result.Errors = append(result.Errors, LineError{Code: "E-QUICK-SOURCE", Message: "only one source account allowed"})
				return result
			}
			sourceAlias = t.text[1:]
		default:
			result.Errors = append(result.Errors, LineError{Code: "E-QUICK-LEFT-TOKENS", Message: "unexpected token before ->"})
			return result
		}
	}
	if amount == "" {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-AMOUNT", Message: "explicit form requires an amount"})
		return result
	}
	if !compileAmountRegex.MatchString(amount) {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-AMOUNT", Message: "amount must be a positive decimal number"})
		return result
	}
	if currency == "" {
		currency = operatingCurrency
	}
	if currency == "" {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-CURRENCY", Message: "no currency supplied and none inferable"})
		return result
	}
	if !compileCurrencyRegex.MatchString(currency) {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-CURRENCY", Message: "invalid currency"})
		return result
	}
	for _, t := range right {
		switch t.kind {
		case tokAt:
			if destAlias != "" {
				result.Errors = append(result.Errors, LineError{Code: "E-QUICK-DEST", Message: "only one destination account allowed"})
				return result
			}
			destAlias = t.text[1:]
		case tokColon:
			narration = strings.TrimSpace(t.text[1:])
		case tokHash:
			tags = append(tags, t.text[1:])
		case tokLink:
			links = append(links, t.text[1:])
		default:
			result.Errors = append(result.Errors, LineError{Code: "E-QUICK-RIGHT-TOKENS", Message: "unexpected token after ->"})
			return result
		}
	}
	if sourceAlias == "" {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-SOURCE", Message: "explicit form requires a source account (@alias)"})
		return result
	}
	if destAlias == "" {
		result.Errors = append(result.Errors, LineError{Code: "E-QUICK-DEST", Message: "explicit form requires a destination account (@alias)"})
		return result
	}
	sourceAccount := resolveAccount(sourceAlias, profile)
	if sourceAccount == "" {
		if isValidAccount(sourceAlias) {
			sourceAccount = sourceAlias
		} else {
			result.Errors = append(result.Errors, LineError{Code: "E-QUICK-SOURCE", Message: fmt.Sprintf("unknown source account alias %q", sourceAlias)})
			return result
		}
	}
	destAccount := resolveAccount(destAlias, profile)
	if destAccount == "" {
		if isValidAccount(destAlias) {
			destAccount = destAlias
		} else {
			result.Errors = append(result.Errors, LineError{Code: "E-QUICK-DEST", Message: fmt.Sprintf("unknown destination account alias %q", destAlias)})
			return result
		}
	}
	result.Entry = buildEntry(txnDate, flag, "", narration, sourceAccount, destAccount, amount, currency, tags, links)
	return result
}

func buildEntry(txnDate ledger.Date, flag, payee, narration, source, dest, amount, currency string, tags, links []string) *favaadapter.NewEntry {
	// The source posting carries a negative amount (value flows out) and the
	// destination posting carries the positive amount (value flows in). Both
	// are always written explicitly per "Explicit quick-entry output".
	return &favaadapter.NewEntry{
		Type:      "transaction",
		Date:      txnDate.Raw,
		Flag:      flag,
		Payee:     payee,
		Narration: narration,
		Tags:      tags,
		Links:     links,
		Postings: []favaadapter.NewPosting{
			{Account: source, Amount: "-" + amount, Currency: currency},
			{Account: dest, Amount: amount, Currency: currency},
		},
	}
}

func resolveAccount(name string, profile Profile) string {
	if name == "" {
		return ""
	}
	if isValidAccount(name) {
		return name
	}
	for _, a := range profile.Accounts {
		if a.Alias == name {
			return a.Account
		}
	}
	return ""
}

func findTemplate(profile Profile, name string) (TemplateRule, bool) {
	for _, t := range profile.Templates {
		if t.Name == name {
			return t, true
		}
	}
	return TemplateRule{}, false
}

func splitLines(text string) []string {
	return strings.Split(text, "\n")
}

func atoiOrZero(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

var (
	compileDateRegex     = regexp.MustCompile(`\A\d{4}-\d{2}-\d{2}\z`)
	compileAmountRegex   = regexp.MustCompile(`\A\d+(\.\d+)?\z`)
	compileCurrencyRegex = regexp.MustCompile(`\A[A-Z][A-Z0-9'._\-]{0,23}\z`)
)

// ---- tokenizer ----

type tokenKind int

const (
	tokWord tokenKind = iota
	tokAt
	tokArrow
	tokColon
	tokHash
	tokLink
)

type token struct {
	kind tokenKind
	text string
}

func tokenize(line string) []token {
	var tokens []token
	i := 0
	runes := []rune(line)
	for i < len(runes) {
		// Skip whitespace.
		if runes[i] == ' ' || runes[i] == '\t' {
			i++
			continue
		}
		// Single-char sigils.
		switch runes[i] {
		case '@':
			start := i
			i++
			for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
				i++
			}
			tokens = append(tokens, token{kind: tokAt, text: string(runes[start:i])})
			continue
		case '#':
			start := i
			i++
			for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
				i++
			}
			tokens = append(tokens, token{kind: tokHash, text: string(runes[start:i])})
			continue
		case '^':
			start := i
			i++
			for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
				i++
			}
			tokens = append(tokens, token{kind: tokLink, text: string(runes[start:i])})
			continue
		case ':':
			// Colon starts the narration; it runs until the next tag or
			// link sigil (so inline "#tag" after the narration still
			// parses as a tag rather than becoming narration text).
			tail := runes[i+1:]
			cutoff := len(tail)
			for j, r := range tail {
				if r == '#' || r == '^' {
					cutoff = j
					break
				}
			}
			narration := strings.TrimSpace(string(tail[:cutoff]))
			tokens = append(tokens, token{kind: tokColon, text: ":" + narration})
			i = i + 1 + cutoff
			continue
		}
		// Arrow.
		if i+1 < len(runes) && runes[i] == '-' && runes[i+1] == '>' {
			tokens = append(tokens, token{kind: tokArrow, text: "->"})
			i += 2
			continue
		}
		// Generic word.
		start := i
		for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
			i++
		}
		tokens = append(tokens, token{kind: tokWord, text: string(runes[start:i])})
	}
	return tokens
}
