// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

// Package sourceexport freezes compatible source bytes and publishes them as
// one filesystem transaction. It never executes Python or ledger plugins.
package sourceexport

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"orangecount/internal/diagnostic"
	"orangecount/internal/dialect"
	"orangecount/internal/documents"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

type Options struct {
	Build   snapshot.BuildOptions
	Reverse bool
}
type output struct {
	path string
	data []byte
}
type protected struct {
	path string
	info os.FileInfo
}

// Plan owns immutable rendered bytes; source changes invalidate publication.
type Plan struct {
	graph       *source.Graph
	root, entry string
	files       []output
	protected   []protected
	options     Options
	fingerprint string
}

// Prepare performs all compatibility checks without touching a target.
func Prepare(entry string, options Options) (*Plan, error) {
	graph, err := source.LoadGraph(entry)
	if err != nil {
		return nil, err
	}
	parsed, bag := ledger.ParseGraph(graph)
	if bag.HasErrors() {
		return nil, diagnosticsError(bag.All())
	}
	selected := snapshot.BuildGraphWithOptions(graph, options.Build)
	if selected.Snapshot == nil {
		return nil, diagnosticsError(selected.Diagnostics)
	}
	var standardFingerprint string
	if !options.Reverse {
		standard := snapshot.BuildGraphWithOptions(graph, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
		if standard.Snapshot == nil {
			return nil, diagnosticsError(standard.Diagnostics)
		}
		e := standard.Snapshot.Evaluation()
		for _, a := range e.Accounts {
			if a.Booking == "AVERAGE" {
				return nil, averageError(graph, parsed, a.Name)
			}
		}
		if e.Options["booking_method"] == "AVERAGE" {
			return nil, averageError(graph, parsed, "")
		}
		// A UTF-8 root account name is a dialect extension: the v3 reader only
		// admits an ASCII lead byte in the root, so exporting it would produce a
		// file no v3 tool can load. Refusing is the only portable answer.
		if slices.Contains(e.ActiveExtensions, "utf8-account-root") {
			return nil, fmt.Errorf("E-EXPORT-COMPAT: a non-ASCII root account name is not representable in Beancount v3; rename the root with an ASCII name_* option value")
		}
		if len(e.ActiveExtensions) > 0 {
			return nil, averageError(graph, parsed, "")
		}
		if err = compatible(graph, parsed); err != nil {
			return nil, err
		}
		standardFingerprint, err = financialFingerprint(e)
		if err != nil {
			return nil, err
		}
	}
	p := &Plan{graph: graph, root: filepath.Dir(graph.Path(graph.Entry)), entry: filepath.Base(graph.Path(graph.Entry)), options: options}
	p.fingerprint, err = financialFingerprint(selected.Snapshot.Evaluation())
	if err != nil {
		return nil, err
	}
	if !options.Reverse && p.fingerprint != standardFingerprint {
		return nil, fmt.Errorf("E-EXPORT-COMPAT: selected accounting differs from Beancount v3; use -semantics v3 after migration")
	}
	outputs := map[source.FileID][]byte{}
	if options.Reverse {
		for id, f := range parsed {
			edits, _ := dialect.Dialectize(f)
			outputs[id] = dialect.ApplyEdits(f.Source.Data, edits)
		}
	} else {
		var diags []diagnostic.Diagnostic
		outputs, diags = dialect.ExportText(graph, parsed)
		for _, d := range diags {
			if d.Severity == diagnostic.Error {
				return nil, diagnosticsError(diags)
			}
		}
	}
	seen := map[source.FileID]bool{graph.Entry: true}
	for _, id := range graph.Order {
		rel, err := filepath.Rel(p.root, graph.Path(id))
		if err != nil || !insideRelative(rel) {
			return nil, fmt.Errorf("E-EXPORT-INCLUDE: include outside entry directory: %s", graph.Path(id))
		}
		data := outputs[id]
		if !options.Reverse {
			data, err = rewriteDocuments(data, graph.Path(id))
			if err != nil {
				return nil, err
			}
		}
		p.files = append(p.files, output{rel, append([]byte(nil), data...)})
		info, err := os.Stat(graph.Path(id))
		if err != nil {
			return nil, err
		}
		p.protected = append(p.protected, protected{graph.Path(id), info})
		for _, edge := range graph.Edges[id] {
			childRel, relErr := filepath.Rel(p.root, graph.Path(edge.To))
			if relErr != nil || !insideRelative(childRel) {
				return nil, fmt.Errorf("E-EXPORT-INCLUDE: include outside entry directory at %s:%d", graph.Path(id), edge.Span.StartLine)
			}
			spelled := filepath.Clean(filepath.Join(filepath.Dir(graph.Path(id)), edge.Literal))
			if !filepath.IsAbs(edge.Literal) && !strings.ContainsAny(edge.Literal, "*?[") && spelled != graph.Path(edge.To) {
				return nil, fmt.Errorf("E-EXPORT-INCLUDE: symlink include requires remapping at %s:%d", graph.Path(id), edge.Span.StartLine)
			}

			if strings.ContainsAny(edge.Literal, "*?[") {
				return nil, fmt.Errorf("E-EXPORT-INCLUDE: glob include is outside the export subset at %s:%d", graph.Path(id), edge.Span.StartLine)
			}
			if filepath.IsAbs(edge.Literal) {
				return nil, fmt.Errorf("E-EXPORT-INCLUDE: absolute include at %s:%d", graph.Path(id), edge.Span.StartLine)
			}
			if seen[edge.To] {
				return nil, fmt.Errorf("E-EXPORT-INCLUDE: duplicate include at %s:%d", graph.Path(id), edge.Span.StartLine)
			}
			seen[edge.To] = true
		}
	}
	for _, doc := range selected.DocumentDeclarations() {
		name := doc.Filename
		if !filepath.IsAbs(name) {
			name = filepath.Join(filepath.Dir(doc.SourcePath), name)
		}
		name, err = filepath.Abs(name)
		if err != nil {
			return nil, err
		}
		info, _ := os.Stat(name)
		p.protected = append(p.protected, protected{name, info})
	}
	return p, nil
}
func diagnosticsError(ds []diagnostic.Diagnostic) error {
	for _, d := range ds {
		if d.Severity == diagnostic.Error {
			return fmt.Errorf("%s %s at %s", d.Code, d.Message, d.Span)
		}
	}
	return fmt.Errorf("E-EXPORT-INVALID: invalid ledger")
}
func insideRelative(rel string) bool {
	return rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

var plainNumber = regexp.MustCompile(`^[+-]?(?:[0-9][0-9,]*(?:\.[0-9]*)?|\.[0-9]+)$`)

// optionPolicy classifies every option the export gate understands. An option
// is validated-semantic when OrangeCount reads it during evaluation, so its
// value is already reflected in the fingerprints; display-only when no Go code
// path reads it and it only affects the upstream renderer's number formatting;
// anything absent is refused as unproven (and a plugin directive is always
// refused independently).
var optionPolicy = map[string]string{
	"title":                      "validated-semantic",
	"operating_currency":         "validated-semantic",
	"booking_method":             "validated-semantic",
	"inferred_tolerance_default": "validated-semantic",
	"infer_tolerance_from_cost":  "validated-semantic",
	"tolerance_multiplier":       "validated-semantic",
	// The name_* options rename the root accounts that every statement and
	// chart classifies against, so the evaluator reads them and the fingerprints
	// reflect them. Upstream validates every value as a root account name, which
	// ValidRootName mirrors, so a value accepted here is accepted there too.
	"name_assets":      "validated-semantic",
	"name_liabilities": "validated-semantic",
	"name_equity":      "validated-semantic",
	"name_income":      "validated-semantic",
	"name_expenses":    "validated-semantic",
	// render_commas has no reader in this codebase (grep "render_commas" over
	// internal/ and cmd/ finds no Go consumer); upstream Beancount stores it in
	// the display context and uses it only for number formatting. It therefore
	// cannot change an evaluated value, an exported byte, or reload semantics,
	// so export accepts it and the fingerprints prove the rest.
	"render_commas": "display-only",
}

func allowedOption(key string) bool {
	class, ok := optionPolicy[key]
	return ok && (class == "validated-semantic" || class == "display-only")
}

func compatible(g *source.Graph, parsed map[source.FileID]*ledger.File) error {
	for _, id := range g.Order {
		for _, d := range parsed[id].Directives {
			fail := func(reason string) error {
				return fmt.Errorf("E-EXPORT-COMPAT: %s at %s:%d", reason, g.Path(id), d.Span().StartLine)
			}
			switch v := d.(type) {
			case ledger.Document:
				if len(v.Filenames) != 1 {
					return fail("multiple document filenames are not Beancount-compatible")
				}
				name := v.Filenames[0]
				if !filepath.IsAbs(name) {
					name = filepath.Join(filepath.Dir(g.Path(id)), name)
				}
				info, err := os.Stat(name)
				if err != nil || !info.Mode().IsRegular() {
					return fail("document must reference an existing ordinary file")
				}

			case ledger.Plugin:
				return fail("Python plugin is not executed or certified")
			case ledger.Option:
				if !allowedOption(v.Key) {
					return fail("unproven option " + v.Key)
				}
			}
			if err := checkValues(reflect.ValueOf(d)); err != nil {
				return fail(err.Error())
			}
			var txn *ledger.Transaction
			switch v := d.(type) {
			case ledger.Transaction:
				txn = &v
			case *ledger.Transaction:
				txn = v
			}
			if txn != nil {
				for _, posting := range txn.Postings {
					for _, m := range posting.Meta {
						if m.Key == "filename" || m.Key == "lineno" {
							return fail("reserved posting metadata " + m.Key)
						}
					}
				}
			}
		}
	}
	return nil
}

// Check only AST value shapes, stopping before rational internals.
func checkValues(v reflect.Value) error {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		return checkValues(v.Elem())
	}
	if v.Type() == reflect.TypeOf(ledger.Number{}) {
		n := v.Interface().(ledger.Number)
		if n.Rat != nil && (!plainNumber.MatchString(n.Raw) || !boundedSourceNumber(n.Raw)) {
			return fmt.Errorf("unproven numeric expression %q", n.Raw)
		}
		return nil
	}
	if v.Type() == reflect.TypeOf(ledger.Value{}) {
		x := v.Interface().(ledger.Value)
		if x.Kind == ledger.ValueList || x.Kind == ledger.ValueMap || x.Kind == ledger.ValueInvalid {
			return fmt.Errorf("unsupported value shape %s", x.Kind)
		}
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).CanInterface() {
				if err := checkValues(v.Field(i)); err != nil {
					return err
				}
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if err := checkValues(v.Index(i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// Preserve every evaluated transaction/posting (including intermediate lots),
// account lifecycle/full final inventory and prices, but exclude provenance.
// canonicalAccounting reduces a transaction to its accounting identity so two
// layouts of the same records compare equal in the fingerprint. Postings that
// share an account, currency, flag, cost and price are merged by summing their
// units (the dialect may split one aggregated line into its per-leg components,
// and ADR-0045 requires that to be accepted), and the merged postings are then
// sorted into a stable order. Nothing is dropped: a posting whose account,
// currency or annotation differs stays its own entry, so a real change of what
// was recorded still changes the fingerprint.
func canonicalAccounting(txn ledger.Transaction) ledger.Transaction {
	if len(txn.Postings) < 2 {
		return txn
	}
	type merged struct {
		key     string
		posting ledger.Posting
		total   *big.Rat
	}
	byKey := map[string]*merged{}
	var order []*merged
	passthrough := false
	for _, p := range txn.Postings {
		if p.Units == nil || p.Units.Number.Rat == nil {
			// Elided or non-numeric legs carry no value to merge; keep the
			// transaction untouched so the comparison stays exact.
			passthrough = true
			break
		}
		key := postingMergeKey(p)
		if existing, ok := byKey[key]; ok {
			existing.total.Add(existing.total, p.Units.Number.Rat)
			continue
		}
		cp := p
		units := *p.Units
		total := new(big.Rat).Set(p.Units.Number.Rat)
		units.Number = ledger.Number{Rat: total, Raw: p.Units.Number.Raw}
		cp.Units = &units
		entry := &merged{key: key, posting: cp, total: total}
		byKey[key] = entry
		order = append(order, entry)
	}
	if passthrough {
		return txn
	}
	if len(order) == len(txn.Postings) {
		// Nothing merged: only normalize the order.
		ordered := append([]ledger.Posting(nil), txn.Postings...)
		sort.SliceStable(ordered, func(i, j int) bool {
			return postingSortKey(ordered[i]) < postingSortKey(ordered[j])
		})
		txn.Postings = ordered
		return txn
	}
	postings := make([]ledger.Posting, 0, len(order))
	for _, m := range order {
		p := m.posting
		units := *p.Units
		units.Number = ledger.Number{Rat: new(big.Rat).Set(m.total), Raw: p.Units.Number.Raw}
		p.Units = &units
		postings = append(postings, p)
	}
	sort.SliceStable(postings, func(i, j int) bool {
		return postingSortKey(postings[i]) < postingSortKey(postings[j])
	})
	txn.Postings = postings
	return txn
}

// postingMergeKey identifies postings that an aggregation may combine: the same
// account, flag, currency and cost/price annotation. The numeric value is
// deliberately excluded because summing it is the point of the merge.
func postingMergeKey(p ledger.Posting) string {
	var b strings.Builder
	b.WriteString(p.Account)
	b.WriteByte(0)
	b.WriteString(p.Flag)
	b.WriteByte(0)
	if p.Units != nil {
		b.WriteString(p.Units.Currency)
	}
	b.WriteByte(0)
	if p.Cost != nil {
		b.WriteString(p.Cost.Raw)
	}
	b.WriteByte(0)
	if p.Price != nil {
		b.WriteString(p.Price.Amount.Currency)
		if p.Price.Total {
			b.WriteByte('T')
		}
	}
	return b.String()
}

func postingSortKey(p ledger.Posting) string {
	var b strings.Builder
	b.WriteString(p.Account)
	b.WriteByte(0)
	if p.Units != nil {
		b.WriteString(p.Units.Currency)
		b.WriteByte(0)
		b.WriteString(p.Units.Number.String())
	}
	b.WriteByte(0)
	b.WriteString(p.Flag)
	if p.Cost != nil {
		b.WriteByte(1)
	}
	if p.Price != nil {
		b.WriteByte(2)
	}
	return b.String()
}

func financialFingerprint(e ledger.Evaluation) (string, error) {
	for i, entry := range e.Entries {
		switch doc := entry.Directive.(type) {
		case ledger.Document:
			doc.Filenames = append([]string(nil), doc.Filenames...)
			for j, name := range doc.Filenames {
				if !filepath.IsAbs(name) {
					name = filepath.Join(filepath.Dir(entry.File), name)
				}
				canonical, err := documents.CanonicalPath(name)
				if err != nil {
					return "", err
				}
				doc.Filenames[j] = canonical
			}
			e.Entries[i].Directive = doc
		}
	}
	for name, account := range e.Accounts {
		account.Booking = ""
		e.Accounts[name] = account
	}
	// A dialect rewrite may change how one transaction's legs are recorded and
	// ordered without changing its accounting: ADR-0045 records that an
	// aggregated line may split back into its exact per-leg components, and the
	// dialect serializer emits block legs in canonical order. Normalize each
	// transaction to its accounting identity (same-account legs merged, then
	// sorted) before marshalling, so the fingerprint compares what the ledger
	// records rather than how the legs happen to be laid out. The entry sequence
	// stays order-sensitive and a different total or account set is still caught.
	for i, entry := range e.Entries {
		switch txn := entry.Directive.(type) {
		case ledger.Transaction:
			e.Entries[i].Directive = canonicalAccounting(txn)
		case *ledger.Transaction:
			if txn != nil {
				normalized := canonicalAccounting(*txn)
				e.Entries[i].Directive = &normalized
			}
		}
	}
	b, err := json.Marshal(struct {
		Accounts map[string]ledger.AccountState
		Prices   map[string][]ledger.PriceQuote
		Entries  []ledger.EntryRecord
		Options  map[string]string
	}{e.Accounts, e.Prices, e.Entries, e.Options})
	if err != nil {
		return "", err
	}
	var v any
	if err = json.Unmarshal(b, &v); err != nil {
		return "", err
	}
	var clean func(any)
	clean = func(x any) {
		switch y := x.(type) {
		case map[string]any:
			// Tolerances is derived query metadata; legacy/v3 export acceptance
			// compares actual postings/inventories and declared user metadata.
			for _, key := range []string{"At", "Span", "Raw", "File", "SourceMetadata", "Tolerances"} {
				delete(y, key)
			}
			for key, z := range y {
				if list, ok := z.([]any); ok && len(list) == 0 {
					y[key] = nil
				} else {
					clean(z)
				}
			}
		case []any:
			for _, z := range y {
				clean(z)
			}
		}
	}
	clean(v)
	// Entries of one date are ordered by source line number in the evaluator,
	// and a conversion rewrites lines, so the same accounting can present its
	// same-day entries in a different sequence. Sort each date group by the
	// cleaned content so the comparison sees the records rather than their
	// rewritten line order; the cross-date sequence is left as evaluated, so a
	// real reordering across dates still changes the fingerprint.
	canonicalizeCleanedEntries(v)
	b, err = json.Marshal(v)
	return string(b), err
}

// canonicalizeCleanedEntries sorts the cleaned entry records inside each date
// group. It runs on the cleaned JSON values, where source spans are already
// removed, so both sides of a comparison produce the same key.
func canonicalizeCleanedEntries(v any) {
	root, ok := v.(map[string]any)
	if !ok {
		return
	}
	entries, ok := root["Entries"].([]any)
	if !ok {
		return
	}
	dateKey := func(entry any) string {
		m, ok := entry.(map[string]any)
		if !ok {
			return ""
		}
		d, ok := m["Date"].(map[string]any)
		if !ok {
			return ""
		}
		return fmt.Sprintf("%v-%v-%v", d["Year"], d["Month"], d["Day"])
	}
	start := 0
	for start < len(entries) {
		end := start + 1
		for end < len(entries) && dateKey(entries[end]) == dateKey(entries[start]) {
			end++
		}
		if end-start > 1 {
			block := entries[start:end]
			sort.SliceStable(block, func(i, j int) bool {
				left, _ := json.Marshal(block[i])
				right, _ := json.Marshal(block[j])
				return string(left) < string(right)
			})
		}
		start = end
	}
}

var quotedSource = regexp.MustCompile(`"(?:\\.|[^"\\])*"`)

// Resolve document references to the original files; attachments are not copied.
func rewriteDocuments(data []byte, path string) ([]byte, error) {
	file, bag := ledger.ParseText(path, data)
	if bag.HasErrors() {
		return nil, diagnosticsError(bag.All())
	}
	var edits []dialect.Edit
	for _, d := range file.Directives {
		doc, ok := d.(ledger.Document)
		if !ok {
			continue
		}
		if len(doc.Filenames) != 1 {
			return nil, fmt.Errorf("E-EXPORT-COMPAT: multiple document filenames")
		}
		name := doc.Filenames[0]
		if !filepath.IsAbs(name) {
			name = filepath.Join(filepath.Dir(path), name)
		}
		canonical, err := documents.CanonicalPath(name)
		if err != nil {
			return nil, err
		}
		raw := string(data[doc.At.Start:doc.At.End])
		found := false
		for _, match := range quotedSource.FindAllStringIndex(raw, -1) {
			decoded, err := strconv.Unquote(raw[match[0]:match[1]])
			if err == nil && decoded == doc.Filenames[0] {
				span := doc.At
				span.Start += match[0]
				span.End = doc.At.Start + match[1]
				edits = append(edits, dialect.Edit{Span: span, Text: strconv.Quote(canonical)})
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("E-EXPORT-COMPAT: cannot preserve document reference at %s:%d", path, doc.At.StartLine)
		}
	}
	return dialect.ApplyEdits(data, edits), nil
}

func averageError(g *source.Graph, parsed map[source.FileID]*ledger.File, account string) error {
	for _, id := range g.Order {
		for _, d := range parsed[id].Directives {
			switch v := d.(type) {
			case ledger.Open:
				if account != "" && v.Account == account {
					return fmt.Errorf("E-EXPORT-COMPAT: effective AVERAGE for %s at %s:%d is not representable", account, g.Path(id), d.Span().StartLine)
				}
			case ledger.Option:
				if account == "" && v.Key == "booking_method" && v.Value == "AVERAGE" {
					return fmt.Errorf("E-EXPORT-COMPAT: global AVERAGE at %s:%d is not representable", g.Path(id), d.Span().StartLine)
				}
			}
		}
	}
	return fmt.Errorf("E-EXPORT-COMPAT: AVERAGE or unproven accounting extension is not representable")
}

func boundedSourceNumber(raw string) bool {
	digits := strings.NewReplacer("+", "", "-", "", ".", "", ",", "").Replace(raw)
	return len(raw) <= 128 && len(strings.TrimLeft(digits, "0")) <= 28
}
