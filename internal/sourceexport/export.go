// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

// Package sourceexport freezes compatible source bytes and publishes them as
// one filesystem transaction. It never executes Python or ledger plugins.
package sourceexport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
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
		if e.Options["booking_method"] == "AVERAGE" || len(e.ActiveExtensions) > 0 {
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
var allowedOptions = map[string]bool{"title": true, "operating_currency": true, "booking_method": true, "inferred_tolerance_default": true, "infer_tolerance_from_cost": true, "tolerance_multiplier": true}

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
				if !allowedOptions[v.Key] {
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
			for _, key := range []string{"At", "Span", "Raw", "File", "SourceMetadata"} {
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
	b, err = json.Marshal(v)
	return string(b), err
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
