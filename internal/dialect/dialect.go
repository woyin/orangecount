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
		out.Directives, edits, diagnostics = expandFile(index, file, fileID, out, edits, diagnostics)
		expanded[fileID] = out
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].Span.Start < edits[j].Span.Start })
	return expanded, edits, diagnostics
}

// expandFile rewrites one parsed file's dialect directives (standalone lines
// and blocks) into compiled transactions, appending to out.Directives.
func expandFile(index *index, file *ledger.File, fileID source.FileID, out *ledger.File, edits []Edit, diagnostics []diagnostic.Diagnostic) ([]ledger.Directive, []Edit, []diagnostic.Diagnostic) {
	out.Directives = make([]ledger.Directive, 0, len(file.Directives))
	var pendingHeader *ledger.Transaction
	var pendingLegs []ledger.Dialect
	flush := func() {
		if pendingHeader == nil && len(pendingLegs) == 0 {
			return
		}
		if pendingHeader == nil || len(pendingLegs) == 0 {
			// A posting-less header with no legs (invalid v3, evaluator will
			// flag it) or a stray dateless leg: keep them in the tree.
			if pendingHeader != nil {
				out.Directives = append(out.Directives, pendingHeader)
			}
			for _, leg := range pendingLegs {
				out.Directives = append(out.Directives, leg)
			}
			pendingHeader, pendingLegs = nil, nil
			return
		}
		appendCompiled(out, index, fileID, pendingHeader, pendingLegs, &edits, &diagnostics)
		pendingHeader, pendingLegs = nil, nil
	}
	for _, directive := range file.Directives {
		if txn, ok := directive.(*ledger.Transaction); ok && len(txn.Postings) == 0 {
			flush()
			pendingHeader = txn
			continue
		}
		if leg, ok := directive.(ledger.Dialect); ok && !leg.HasDate && pendingHeader != nil {
			pendingLegs = append(pendingLegs, leg)
			continue
		}
		flush()
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
	flush()
	return out.Directives, edits, diagnostics
}

// appendCompiled compiles a pending block and appends either the compiled
// transaction (plus its edit) or the raw header and legs when compilation
// failed, leaving the error diagnostics to block publication.
func appendCompiled(out *ledger.File, index *index, fileID source.FileID, header *ledger.Transaction, legs []ledger.Dialect, edits *[]Edit, diagnostics *[]diagnostic.Diagnostic) {
	txn, edit, diags := compileBlock(index, header, legs)
	*diagnostics = append(*diagnostics, diags...)
	if txn != nil {
		out.Directives = append(out.Directives, txn)
		edit.FileID = fileID
		*edits = append(*edits, edit)
		return
	}
	// Keep the failing block in the tree so source views can navigate; the
	// error diagnostics block publication.
	out.Directives = append(out.Directives, header)
	for _, leg := range legs {
		out.Directives = append(out.Directives, leg)
	}
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

// compileBlock compiles a dialect block: a transaction header (date, flag,
// payee, narration, tags) followed by one or more dateless legs. Each leg
// becomes two postings; the header owns the transaction-level fields. The
// compiled transaction and a single edit spanning the whole block are
// returned, or nil plus E-DIALECT-* diagnostics when any leg fails.
func compileBlock(idx *index, header *ledger.Transaction, legs []ledger.Dialect) (*ledger.Transaction, Edit, []diagnostic.Diagnostic) {
	var diags []diagnostic.Diagnostic
	fail := func(code, message string) (*ledger.Transaction, Edit, []diagnostic.Diagnostic) {
		return nil, Edit{}, append(diags, diagnostic.New(code, diagnostic.Error, header.At, message))
	}
	if !header.Date.Valid() {
		return fail("E-DIALECT-DATE", "dialect block header has no valid date")
	}
	for _, leg := range legs {
		if len(leg.Meta) > 0 {
			return fail("E-DIALECT-LEG-META", "dialect block leg metadata is unsupported")
		}
	}
	txn := &ledger.Transaction{
		DirectiveBase: header.DirectiveBase,
		Date:          header.Date,
		Flag:          flagOrDefault(header.Flag),
		Payee:         header.Payee,
		Narration:     header.Narration,
		Tags:          header.Tags,
		Links:         header.Links,
		Meta:          header.Meta,
	}
	if txn.Narration == "" {
		txn.Narration = "消费"
	}
	for _, leg := range legs {
		currency := legCurrency(idx, leg)
		if currency == "" {
			return fail("E-DIALECT-CURRENCY", "no currency given and the ledger has no single operating_currency")
		}
		srcAccount, srcErr := resolveEndpoint(idx, leg.SourceRef, header.Date)
		if srcErr != nil {
			srcErr.Span = leg.At
			return nil, Edit{}, append(diags, *srcErr)
		}
		dstAccount, dstErr := resolveEndpoint(idx, leg.DestRef, header.Date)
		if dstErr != nil {
			dstErr.Span = leg.At
			return nil, Edit{}, append(diags, *dstErr)
		}
		if leg.Price != nil {
			postings, errDiags := compileSellLeg(idx, header, leg, currency, srcAccount, dstAccount)
			if errDiags != nil {
				return nil, Edit{}, append(diags, errDiags...)
			}
			txn.Postings = append(txn.Postings, postings...)
			continue
		}
		if leg.Security != "" {
			postings, errDiags := compileBuyLeg(idx, header, leg, currency, srcAccount, dstAccount)
			if errDiags != nil {
				return nil, Edit{}, append(diags, errDiags...)
			}
			txn.Postings = append(txn.Postings, postings...)
			continue
		}
		txn.Postings = append(txn.Postings,
			ledger.Posting{
				At:      leg.At,
				Raw:     leg.Raw,
				Account: srcAccount,
				Units:   &ledger.Amount{Number: negateNumber(leg.Amount), Currency: currency, At: leg.At},
			},
			ledger.Posting{
				At:      leg.At,
				Raw:     leg.Raw,
				Account: dstAccount,
				Units:   &ledger.Amount{Number: leg.Amount, Currency: currency, At: leg.At},
			},
		)
	}
	txn.Postings = groupPostings(txn.Postings)
	span := source.Span{Start: header.At.Start, End: legs[len(legs)-1].At.End}
	return txn, Edit{Span: span, Text: SerializeTransaction(txn)}, diags
}

// compileBuyLeg builds the postings for one buy leg. With a fee suffix and
// no explicit amount the cash posting stays elided so the residual absorbs
// quantity × unit cost plus fee, matching how the ledger records stock
// buys. Otherwise the source pays the explicit amount or the derived
// quantity × unit cost (a bonus-share source pays from an income account).
// The auto-quantity form leaves the securities quantity empty for the
// evaluator to infer from the explicit cash amount.
func compileBuyLeg(idx *index, header *ledger.Transaction, leg ledger.Dialect, currency, srcAccount, dstAccount string) ([]ledger.Posting, []diagnostic.Diagnostic) {
	if leg.FeeAmount.Raw != "" && leg.Amount.Raw == "" {
		postings := []ledger.Posting{
			{At: leg.At, Raw: leg.Raw, Account: srcAccount},
			{At: leg.At, Raw: leg.Raw, Account: dstAccount, Units: &ledger.Amount{Number: leg.Quantity, Currency: leg.Security, At: leg.At}, Cost: leg.Cost},
		}
		return appendFeePosting(postings, idx, leg, header.Date)
	}
	if !leg.HasQuantity {
		if leg.Amount.Raw == "" {
			return nil, []diagnostic.Diagnostic{diagnostic.New("E-DIALECT-SECURITY", diagnostic.Error, leg.At, "auto-quantity investment leg needs an explicit cash amount")}
		}
		return []ledger.Posting{
			{At: leg.At, Raw: leg.Raw, Account: srcAccount, Units: &ledger.Amount{Number: negateNumber(leg.Amount), Currency: currency, At: leg.At}},
			{At: leg.At, Raw: leg.Raw, Account: dstAccount, Units: &ledger.Amount{Currency: leg.Security, At: leg.At}, Cost: leg.Cost},
		}, nil
	}
	sourceAmount, ok := investmentCashAmount(leg)
	if !ok {
		return nil, []diagnostic.Diagnostic{diagnostic.New("E-DIALECT-SECURITY", diagnostic.Error, leg.At, "investment leg needs an explicit amount or a unit cost to derive the cash side")}
	}
	postings := []ledger.Posting{
		{At: leg.At, Raw: leg.Raw, Account: srcAccount, Units: &ledger.Amount{Number: negateNumber(sourceAmount), Currency: currency, At: leg.At}},
		{At: leg.At, Raw: leg.Raw, Account: dstAccount, Units: &ledger.Amount{Number: leg.Quantity, Currency: leg.Security, At: leg.At}, Cost: leg.Cost},
	}
	return appendFeePosting(postings, idx, leg, header.Date)
}

// appendFeePosting appends the leg's resolved fee expense posting, or
// returns a diagnostic when a suffix is present but unresolved.
func appendFeePosting(postings []ledger.Posting, idx *index, leg ledger.Dialect, date ledger.Date) ([]ledger.Posting, []diagnostic.Diagnostic) {
	fee, ok := legFeePosting(idx, leg, date)
	if !ok {
		return nil, []diagnostic.Diagnostic{diagnostic.New("E-DIALECT-ACCOUNT", diagnostic.Error, leg.At, "fee endpoint %q is unknown", leg.FeeRef)}
	}
	if fee != nil {
		postings = append(postings, *fee)
	}
	return postings, nil
}

// compileSellLeg builds the postings for one sell leg: the securities
// reduction at the sale price (source endpoint), the cash side (explicit or
// elided to absorb the residual), the elided gain posting, and the fee.
func compileSellLeg(idx *index, header *ledger.Transaction, leg ledger.Dialect, currency, srcAccount, dstAccount string) ([]ledger.Posting, []diagnostic.Diagnostic) {
	if !leg.HasQuantity {
		return nil, []diagnostic.Diagnostic{diagnostic.New("E-DIALECT-SECURITY", diagnostic.Error, leg.At, "sell leg needs an explicit quantity")}
	}
	if leg.Amount.Raw == "" && leg.GainRef == "" && leg.FeeAmount.Raw == "" {
		return nil, []diagnostic.Diagnostic{diagnostic.New("E-DIALECT-SECURITY", diagnostic.Error, leg.At, "sell leg needs an explicit amount, a gain endpoint, or a fee to balance")}
	}
	if leg.Amount.Raw == "" && leg.GainRef != "" {
		return nil, []diagnostic.Diagnostic{diagnostic.New("E-DIALECT-SECURITY", diagnostic.Error, leg.At, "sell leg with a gain endpoint needs an explicit cash amount")}
	}
	var postings []ledger.Posting
	if leg.Amount.Raw != "" {
		postings = append(postings, ledger.Posting{At: leg.At, Raw: leg.Raw, Account: dstAccount, Units: &ledger.Amount{Number: leg.Amount, Currency: currency, At: leg.At}})
	} else {
		postings = append(postings, ledger.Posting{At: leg.At, Raw: leg.Raw, Account: dstAccount})
	}
	postings = append(postings, ledger.Posting{
		At:      leg.At,
		Raw:     leg.Raw,
		Account: srcAccount,
		Units:   &ledger.Amount{Number: negateNumber(leg.Quantity), Currency: leg.Security, At: leg.At},
		Cost:    leg.Cost,
		Price:   leg.Price,
	})
	if leg.GainRef != "" {
		gainAccount, gainErr := resolveEndpoint(idx, leg.GainRef, header.Date)
		if gainErr != nil {
			gainErr.Span = leg.At
			return nil, []diagnostic.Diagnostic{*gainErr}
		}
		postings = append(postings, ledger.Posting{At: leg.At, Raw: leg.Raw, Account: gainAccount})
	}
	fee, ok := legFeePosting(idx, leg, header.Date)
	if !ok {
		return nil, []diagnostic.Diagnostic{diagnostic.New("E-DIALECT-ACCOUNT", diagnostic.Error, leg.At, "fee endpoint %q is unknown", leg.FeeRef)}
	}
	if fee != nil {
		postings = append(postings, *fee)
	}
	return postings, nil
}

// legFeePosting resolves and builds the explicit fee expense posting from a
// leg's fee suffix. It returns nil with ok=true when no suffix is present,
// and ok=false when a suffix is present but the endpoint does not resolve.
func legFeePosting(idx *index, leg ledger.Dialect, date ledger.Date) (*ledger.Posting, bool) {
	if leg.FeeAmount.Raw == "" {
		return nil, true
	}
	account, err := resolveEndpoint(idx, leg.FeeRef, date)
	if err != nil {
		return nil, false
	}
	return &ledger.Posting{
		At:      leg.At,
		Raw:     leg.Raw,
		Account: account,
		Units:   &ledger.Amount{Number: leg.FeeAmount, Currency: leg.FeeCurrency, At: leg.At},
	}, true
}

// investmentCashAmount returns the cash side of an investment leg: the
// explicit amount when written, otherwise quantity × unit cost. The unit
// cost is the amount component of the cost batch; a cost without an amount
// cannot drive auto-calc.
func investmentCashAmount(leg ledger.Dialect) (ledger.Number, bool) {
	if leg.Amount.Raw != "" {
		return leg.Amount, true
	}
	if leg.Cost == nil || leg.Quantity.Rat == nil {
		return ledger.Number{}, false
	}
	for _, value := range leg.Cost.Components {
		if value.Kind != ledger.ValueAmount {
			continue
		}
		qty := ledger.NewDecimal(leg.Quantity.Rat)
		unit := ledger.DecimalFromNumber(value.Amount.Number)
		product := qty.Mul(unit)
		return ledger.Number{Raw: product.String(), Rat: product.Rat()}, true
	}
	return ledger.Number{}, false
}

// legCurrency resolves a leg's cash currency: explicit currency first, then
// an investment cost's amount currency (so a stock bought in CNY carries
// CNY even with two operating currencies), then the sale price and fee
// currencies (an elided-cash sell carries its currency only there), then
// the single operating currency.
func legCurrency(idx *index, leg ledger.Dialect) string {
	if leg.Currency != "" {
		return leg.Currency
	}
	if leg.Cost != nil {
		for _, value := range leg.Cost.Components {
			if value.Kind == ledger.ValueAmount && value.Amount.Currency != "" {
				return value.Amount.Currency
			}
		}
	}
	if leg.Price != nil && leg.Price.Amount.Currency != "" {
		return leg.Price.Amount.Currency
	}
	if leg.FeeCurrency != "" {
		return leg.FeeCurrency
	}
	return idx.opCurrency
}

// groupPostings orders postings so same-account entries stay adjacent,
// matching how a human writes a multi-leg block — without summing them:
// each leg is its own record (two 49900 gifts are two records), and
// merging them into one number would destroy that granularity.
func groupPostings(postings []ledger.Posting) []ledger.Posting {
	ordered := make([]ledger.Posting, 0, len(postings))
	used := make([]bool, len(postings))
	for i := range postings {
		if used[i] {
			continue
		}
		ordered = append(ordered, postings[i])
		used[i] = true
		for j := i + 1; j < len(postings); j++ {
			if !used[j] && postings[j].Account == postings[i].Account {
				ordered = append(ordered, postings[j])
				used[j] = true
			}
		}
	}
	return ordered
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
