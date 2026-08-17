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
	Preview   string // canonical Beancount block, when compiled
	Duplicate bool   // equivalent transaction exists in the ledger
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

// parseTemplateRest folds a template invocation's trailing tokens into
// amount, currency, payee alias, narration, tags, and links; tokens are
// consumed by shape, so an unknown word is a syntax error.
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
// explicitForm holds the parsed pieces of the explicit shorthand form
// "AMOUNT [CURRENCY] @source -> @destination [: narration] [#tag] [^link]".
type explicitForm struct {
	amount, currency, sourceAlias, destAlias, narration string
	tags, links                                         []string
}

// compileExplicitForm handles the explicit delimited form. Parsing splits
// into the left side (before ->), the right side (after), and endpoint
// resolution, so each stage reports its own error codes.
func compileExplicitForm(line string, lineNo int, tokens []token, txnDate ledger.Date, flag, operatingCurrency string, profile Profile) LineResult {
	result := LineResult{Line: lineNo, Source: line}
	form, err := parseExplicitForm(tokens)
	if err != nil {
		result.Errors = append(result.Errors, *err)
		return result
	}
	if currencyErr := validateExplicitCurrency(&form, operatingCurrency); currencyErr != nil {
		result.Errors = append(result.Errors, *currencyErr)
		return result
	}
	sourceAccount, sourceErr := resolveEndpoint(form.sourceAlias, profile, "E-QUICK-SOURCE", "source")
	if sourceErr != nil {
		result.Errors = append(result.Errors, *sourceErr)
		return result
	}
	destAccount, destErr := resolveEndpoint(form.destAlias, profile, "E-QUICK-DEST", "destination")
	if destErr != nil {
		result.Errors = append(result.Errors, *destErr)
		return result
	}
	result.Entry = buildEntry(txnDate, flag, "", form.narration, sourceAccount, destAccount, form.amount, form.currency, form.tags, form.links)
	return result
}

// parseExplicitForm splits the token stream around the -> arrow and parses
// both sides.
func parseExplicitForm(tokens []token) (explicitForm, *LineError) {
	var form explicitForm
	arrowIdx := -1
	for i, t := range tokens {
		if t.kind == tokArrow {
			arrowIdx = i
			break
		}
	}
	if arrowIdx < 0 {
		return form, &LineError{Code: "E-QUICK-ARROW", Message: "explicit form requires -> between source and destination"}
	}
	if err := parseExplicitLeft(tokens[:arrowIdx], &form); err != nil {
		return form, err
	}
	if err := parseExplicitRight(tokens[arrowIdx+1:], &form); err != nil {
		return form, err
	}
	if form.amount == "" {
		return form, &LineError{Code: "E-QUICK-AMOUNT", Message: "explicit form requires an amount"}
	}
	if !compileAmountRegex.MatchString(form.amount) {
		return form, &LineError{Code: "E-QUICK-AMOUNT", Message: "amount must be a positive decimal number"}
	}
	if form.sourceAlias == "" {
		return form, &LineError{Code: "E-QUICK-SOURCE", Message: "explicit form requires a source account (@alias)"}
	}
	if form.destAlias == "" {
		return form, &LineError{Code: "E-QUICK-DEST", Message: "explicit form requires a destination account (@alias)"}
	}
	return form, nil
}

// parseExplicitLeft reads "AMOUNT [CURRENCY] [@sourceAlias]".
func parseExplicitLeft(left []token, form *explicitForm) *LineError {
	for _, t := range left {
		switch t.kind {
		case tokWord:
			if form.amount == "" {
				form.amount = t.text
			} else if form.currency == "" {
				form.currency = t.text
			} else {
				return &LineError{Code: "E-QUICK-LEFT-TOKENS", Message: "too many tokens before ->"}
			}
		case tokAt:
			if form.sourceAlias != "" {
				return &LineError{Code: "E-QUICK-SOURCE", Message: "only one source account allowed"}
			}
			form.sourceAlias = t.text[1:]
		default:
			return &LineError{Code: "E-QUICK-LEFT-TOKENS", Message: "unexpected token before ->"}
		}
	}
	return nil
}

// parseExplicitRight reads "[@destinationAlias] [: narration] [#tag] [^link]".
func parseExplicitRight(right []token, form *explicitForm) *LineError {
	for _, t := range right {
		switch t.kind {
		case tokAt:
			if form.destAlias != "" {
				return &LineError{Code: "E-QUICK-DEST", Message: "only one destination account allowed"}
			}
			form.destAlias = t.text[1:]
		case tokColon:
			form.narration = strings.TrimSpace(t.text[1:])
		case tokHash:
			form.tags = append(form.tags, t.text[1:])
		case tokLink:
			form.links = append(form.links, t.text[1:])
		default:
			return &LineError{Code: "E-QUICK-RIGHT-TOKENS", Message: "unexpected token after ->"}
		}
	}
	return nil
}

// validateExplicitCurrency fills the currency default chain (explicit, then
// operating currency) and validates the result.
func validateExplicitCurrency(form *explicitForm, operatingCurrency string) *LineError {
	if form.currency == "" {
		form.currency = operatingCurrency
	}
	if form.currency == "" {
		return &LineError{Code: "E-QUICK-CURRENCY", Message: "no currency supplied and none inferable"}
	}
	if !compileCurrencyRegex.MatchString(form.currency) {
		return &LineError{Code: "E-QUICK-CURRENCY", Message: "invalid currency"}
	}
	return nil
}

// resolveEndpoint resolves an @alias through the profile; a full account
// name is accepted directly so power users can target unregistered accounts.
func resolveEndpoint(alias string, profile Profile, code, role string) (string, *LineError) {
	if account := resolveAccount(alias, profile); account != "" {
		return account, nil
	}
	if isValidAccount(alias) {
		return alias, nil
	}
	return "", &LineError{Code: code, Message: fmt.Sprintf("unknown %s account alias %q", role, alias)}
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

// tokenize splits one shorthand line into sigil-prefixed and word tokens.
// The sigil cases (@ # ^) share the "sigil plus non-space run" shape and
// delegate to tokenizeSigil; the colon sigil has its own rule because the
// narration runs until the next tag/link sigil rather than to the next space.
func tokenize(line string) []token {
	var tokens []token
	runes := []rune(line)
	for i := 0; i < len(runes); {
		if runes[i] == ' ' || runes[i] == '\t' {
			i++
			continue
		}
		switch runes[i] {
		case '@', '#', '^':
			var t token
			t, i = tokenizeSigil(runes, i, sigilKind(runes[i]))
			tokens = append(tokens, t)
		case ':':
			i = tokenizeNarration(runes, i, &tokens)
		default:
			i = tokenizeWord(runes, i, &tokens)
		}
	}
	return tokens
}

// sigilKind maps a sigil rune to its token kind.
func sigilKind(r rune) tokenKind {
	switch r {
	case '@':
		return tokAt
	case '#':
		return tokHash
	default: // '^'
		return tokLink
	}
}

// tokenizeSigil consumes a sigil character followed by its non-whitespace
// run (e.g. "@微信" or "#food").
func tokenizeSigil(runes []rune, i int, kind tokenKind) (token, int) {
	start := i
	i++
	for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
		i++
	}
	return token{kind: kind, text: string(runes[start:i])}, i
}

// tokenizeNarration consumes ": narration..." up to the next # or ^ sigil
// (so inline "#tag" after the narration still parses as a tag rather than
// becoming narration text).
func tokenizeNarration(runes []rune, i int, tokens *[]token) int {
	tail := runes[i+1:]
	cutoff := len(tail)
	for j, r := range tail {
		if r == '#' || r == '^' {
			cutoff = j
			break
		}
	}
	narration := strings.TrimSpace(string(tail[:cutoff]))
	*tokens = append(*tokens, token{kind: tokColon, text: ":" + narration})
	return i + 1 + cutoff
}

// tokenizeWord consumes the -> arrow or a plain non-whitespace word.
func tokenizeWord(runes []rune, i int, tokens *[]token) int {
	if i+1 < len(runes) && runes[i] == '-' && runes[i+1] == '>' {
		*tokens = append(*tokens, token{kind: tokArrow, text: "->"})
		return i + 2
	}
	start := i
	for i < len(runes) && runes[i] != ' ' && runes[i] != '\t' {
		i++
	}
	*tokens = append(*tokens, token{kind: tokWord, text: string(runes[start:i])})
	return i
}
