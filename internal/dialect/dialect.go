// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package dialect compiles OrangeCount dialect shorthand lines (ADR-0045)
// into ordinary Beancount v3 transactions. The expansion runs between
// parsing and evaluation, so every downstream consumer — reports, queries,
// validation — behaves exactly as on a standard ledger. The package also
// renders the text edits that turn a dialect source into a pure v3 export
// and the reverse dialectize filter.
package dialect

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/source"
)

// quickAccountCustomType mirrors the public source-ledger contract fixed by
// ADR-0043; the constant is duplicated here so the dialect layer stays
// independent of the web-facing quickentry package.
const quickAccountCustomType = "orangecount.quick-account.v1"

// Edit is one source-text replacement: the span of a dialect line (or, for
// dialectize, a transaction) and the text that replaces it. Consumers apply
// edits to the original file bytes; everything outside edited spans stays
// byte-identical.
type Edit struct {
	FileID source.FileID
	Span   source.Span
	Text   string
}

// Expand replaces every ledger.Dialect directive in parsed with a compiled
// ledger.Transaction at the same position. It returns the rewritten parsed
// map, the text edits that produce the equivalent pure-v3 source, and the
// E-DIALECT-* diagnostics. Semantic failures (endpoint resolution, currency
// default) are errors and leave the original Dialect directive in place;
// callers gate snapshot publication on the returned diagnostics.
func Expand(graph *source.Graph, parsed map[source.FileID]*ledger.File) (map[source.FileID]*ledger.File, []Edit, []diagnostic.Diagnostic) {
	index := buildIndex(graph, parsed)
	var edits []Edit
	var diagnostics []diagnostic.Diagnostic
	expanded := make(map[source.FileID]*ledger.File, len(parsed))
	for fileID, file := range parsed {
		if file == nil || !hasDialect(file) {
			expanded[fileID] = file
			continue
		}
		out := &ledger.File{Source: file.Source, Comments: file.Comments}
		out.Directives = make([]ledger.Directive, 0, len(file.Directives))
		for _, directive := range file.Directives {
			d, ok := directive.(ledger.Dialect)
			if !ok {
				out.Directives = append(out.Directives, directive)
				continue
			}
			txn, edit, diags := compileDialect(index, d)
			diagnostics = append(diagnostics, diags...)
			if txn != nil {
				out.Directives = append(out.Directives, txn)
				edit.FileID = fileID
				edits = append(edits, edit)
				continue
			}
			// Keep the failing line in the tree so source views can still
			// navigate to it; the error diagnostic blocks publication.
			out.Directives = append(out.Directives, d)
		}
		expanded[fileID] = out
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].Span.Start < edits[j].Span.Start })
	return expanded, edits, diagnostics
}

func hasDialect(file *ledger.File) bool {
	for _, directive := range file.Directives {
		if _, ok := directive.(ledger.Dialect); ok {
			return true
		}
	}
	return false
}

// index is the whole-ledger context endpoint and currency resolution needs:
// opened accounts, their tail segments, date-effective quick-account aliases,
// and the single operating_currency declaration.
type index struct {
	accounts   map[string]bool
	tails      map[string][]string
	aliases    []aliasRule
	opCurrency string
}

type aliasRule struct {
	date    ledger.Date
	alias   string
	account string
}

func buildIndex(graph *source.Graph, parsed map[source.FileID]*ledger.File) *index {
	idx := &index{accounts: map[string]bool{}, tails: map[string][]string{}}
	var order []source.FileID
	if graph != nil {
		order = graph.Order
	}
	opCurrencyCount := 0
	for _, fileID := range order {
		file := parsed[fileID]
		if file == nil {
			continue
		}
		for _, directive := range file.Directives {
			switch d := directive.(type) {
			case ledger.Open:
				idx.accounts[d.Account] = true
				tail := accountTail(d.Account)
				idx.tails[tail] = append(idx.tails[tail], d.Account)
			case ledger.Option:
				if strings.EqualFold(d.Key, "operating_currency") {
					opCurrencyCount++
					idx.opCurrency = d.Value
				}
			case ledger.Custom:
				if d.Type != quickAccountCustomType || !d.Date.Valid() {
					continue
				}
				if alias, account, ok := parseQuickAccountCustom(d); ok {
					idx.aliases = append(idx.aliases, aliasRule{date: d.Date, alias: alias, account: account})
				}
			}
		}
	}
	if opCurrencyCount != 1 {
		idx.opCurrency = ""
	}
	return idx
}

// parseQuickAccountCustom extracts ("alias", account) from the typed values
// of an orangecount.quick-account.v1 directive, mirroring the ADR-0043
// contract: a string alias followed by an account value.
func parseQuickAccountCustom(custom ledger.Custom) (string, string, bool) {
	if len(custom.Values) < 2 {
		return "", "", false
	}
	aliasValue, accountValue := custom.Values[0], custom.Values[1]
	if aliasValue.Kind != ledger.ValueString || aliasValue.String == "" {
		return "", "", false
	}
	account := ""
	// An unquoted account token types as ValueAccount; a quoted one as
	// ValueString. Both spell the account name in String.
	switch accountValue.Kind {
	case ledger.ValueAccount, ledger.ValueString:
		account = accountValue.String
	}
	if account == "" {
		return "", "", false
	}
	return aliasValue.String, account, true
}

func errAt(code, message string) *diagnostic.Diagnostic {
	d := diagnostic.New(code, diagnostic.Error, source.Span{}, message)
	return &d
}

func accountTail(account string) string {
	if pos := strings.LastIndex(account, ":"); pos >= 0 {
		return account[pos+1:]
	}
	return account
}

// accountNameValid mirrors the Beancount account shape: colon-separated
// segments, each starting with an uppercase letter.
func accountNameValid(name string) bool {
	// Mirror the ledger parser's isAccount rule (parser.go): any
	// colon-separated name whose first segment starts with an uppercase
	// ASCII letter. Segments may be CJK or otherwise non-ASCII, as in
	// Assets:Wallet:微信, which upstream beancount accepts too.
	if !strings.Contains(name, ":") {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

// resolveEndpoint applies the three-level contract from ADR-0045: exact full
// name, then date-effective declared alias, then a unique tail-segment match.
// Ambiguity and misses are errors listing the candidates; nothing is guessed.
func resolveEndpoint(idx *index, ref string, date ledger.Date) (string, *diagnostic.Diagnostic) {
	if accountNameValid(ref) {
		return ref, nil
	}
	if account, ok := resolveAlias(idx, ref, date); ok {
		return account, nil
	}
	if candidates := idx.tails[ref]; len(candidates) == 1 {
		return candidates[0], nil
	} else if len(candidates) > 1 {
		sorted := append([]string(nil), candidates...)
		sort.Strings(sorted)
		return "", errAt("E-DIALECT-AMBIGUOUS", fmt.Sprintf("endpoint %q matches several accounts: %s; use the full name or declare an alias", ref, strings.Join(sorted, ", ")))
	}
	return "", errAt("E-DIALECT-ACCOUNT", fmt.Sprintf("unknown endpoint %q; use a full account name, a declared alias, or a unique account tail", ref))
}

// resolveAlias returns the account for an alias effective at the transaction
// date. Later definitions supersede earlier ones; same-day competing
// definitions disable the alias (ADR-0043) and resolution falls through.
func resolveAlias(idx *index, alias string, date ledger.Date) (string, bool) {
	bestDate := ""
	bestAccount := ""
	competing := false
	for _, rule := range idx.aliases {
		if rule.alias != alias || !rule.date.Valid() {
			continue
		}
		if rule.date.Year > date.Year || (rule.date.Year == date.Year && (rule.date.Month > date.Month || (rule.date.Month == date.Month && rule.date.Day > date.Day))) {
			continue
		}
		key := fmt.Sprintf("%04d-%02d-%02d", rule.date.Year, rule.date.Month, rule.date.Day)
		if key > bestDate {
			bestDate, bestAccount, competing = key, rule.account, false
		} else if key == bestDate && rule.account != bestAccount {
			competing = true
		}
	}
	if bestAccount == "" || competing {
		return "", false
	}
	return bestAccount, true
}

// compileDialect turns one parsed dialect line into a Transaction plus its
// export edit. Failures return diagnostics anchored at the line's span.
func compileDialect(idx *index, d ledger.Dialect) (*ledger.Transaction, Edit, []diagnostic.Diagnostic) {
	var diags []diagnostic.Diagnostic
	fail := func(code, message string) (*ledger.Transaction, Edit, []diagnostic.Diagnostic) {
		return nil, Edit{}, append(diags, diagnostic.New(code, diagnostic.Error, d.At, message))
	}
	if !d.Date.Valid() {
		return fail("E-DIALECT-DATE", "dialect line has no valid date and no anchor")
	}
	currency := d.Currency
	if currency == "" {
		currency = idx.opCurrency
		if currency == "" {
			return fail("E-DIALECT-CURRENCY", "no currency given and the ledger has no single operating_currency")
		}
	}
	srcAccount, srcErr := resolveEndpoint(idx, d.SourceRef, d.Date)
	if srcErr != nil {
		srcErr.Span = d.At
		return nil, Edit{}, append(diags, *srcErr)
	}
	dstAccount, dstErr := resolveEndpoint(idx, d.DestRef, d.Date)
	if dstErr != nil {
		dstErr.Span = d.At
		return nil, Edit{}, append(diags, *dstErr)
	}
	narration := d.Narration
	if !d.HasNarration {
		narration = "消费"
	}
	txn := &ledger.Transaction{
		DirectiveBase: ledger.DirectiveBase{At: d.At, Raw: d.Raw},
		Date:          d.Date,
		Flag:          d.Flag,
		Payee:         d.Payee,
		Narration:     narration,
		Tags:          d.Tags,
		Links:         d.Links,
	}
	if d.Flag == "" {
		d.Flag = "*"
	}
	amount := d.Amount
	txn.Postings = []ledger.Posting{
		{
			At:      d.At,
			Raw:     d.Raw,
			Account: srcAccount,
			Units:   &ledger.Amount{Number: negateNumber(amount), Currency: currency, At: d.At},
		},
		{
			At:      d.At,
			Raw:     d.Raw,
			Account: dstAccount,
			Units:   &ledger.Amount{Number: amount, Currency: currency, At: d.At},
		},
	}
	return txn, Edit{Span: d.At, Text: SerializeTransaction(txn)}, diags
}

// negateNumber returns the negated copy of a parsed number, normalizing any
// explicit plus sign so "-+5" can never appear in generated source.
func negateNumber(n ledger.Number) ledger.Number {
	out := n
	raw := strings.TrimPrefix(n.Raw, "+")
	out.Raw = "-" + raw
	if n.Rat != nil {
		out.Rat = new(big.Rat).Neg(n.Rat)
	}
	return out
}

// ExportText renders the pure-Beancount-v3 source for every file in the
// graph: dialect lines are replaced by their compiled blocks and every other
// byte is preserved. The returned diagnostics gate the export the same way
// snapshot publication is gated.
func ExportText(graph *source.Graph, parsed map[source.FileID]*ledger.File) (map[source.FileID][]byte, []diagnostic.Diagnostic) {
	expanded, edits, diagnostics := Expand(graph, parsed)
	output := make(map[source.FileID][]byte, len(expanded))
	for fileID, file := range expanded {
		if file == nil || file.Source == nil {
			continue
		}
		output[fileID] = file.Source.Data
	}
	byFile := make(map[source.FileID][]Edit)
	for _, edit := range edits {
		byFile[edit.FileID] = append(byFile[edit.FileID], edit)
	}
	for fileID, fileEdits := range byFile {
		data := output[fileID]
		if data == nil {
			continue
		}
		output[fileID] = ApplyEdits(data, fileEdits)
	}
	return output, diagnostics
}
