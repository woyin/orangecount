// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

import (
	"sort"
	"strings"

	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

// Bootstrap is the typed bootstrap payload for the transplanted frontend's
// shell. It is an independent Go mapping of the Fava "ledger_data" contract
// (docs/fava-contract-map.md row "ledger_data"; frontend validator
// ledgerDataValidator in frontend/src/api/validators.ts).
//
// Field-by-field contract notes:
//   - Accounts, Currencies, Payees, Tags, Links, Years: sorted, deduplicated
//     string slices. The frontend readers expect arrays; no map ordering may
//     reach the wire.
//   - Options and FavaOptions: maps of display options. FavaOptions is the
//     OPT-IN subset, never a full passthrough.
//   - Errors: serialized diagnostics (redacted paths only).
//   - BaseURL: the loopback route base the embedded shell is served at; the
//     adapter handler supplies it (default "/").
//   - Extensions, OtherLedgers, Incognito, HaveExcel: emitted as empty/false
//     because extensions, multi-ledger, incognito, and Excel export are out of
//     scope (ADR-0024, ADR-0033). The frontend validator accepts these as
//     empty/false.
type Bootstrap struct {
	// Accounts is the sorted, deduplicated account list.
	Accounts []string `json:"accounts"`
	// Currencies is the sorted, deduplicated currency list.
	Currencies []string `json:"currencies"`
	// Payees is the sorted, deduplicated payee list.
	Payees []string `json:"payees"`
	// Tags is the sorted, deduplicated tag list.
	Tags []string `json:"tags"`
	// Links is the sorted, deduplicated link list.
	Links []string `json:"links"`
	// Years is the sorted, deduplicated year list.
	Years []string `json:"years"`
	// Options is the display options map (title, operating_currency, ...).
	Options map[string]string `json:"options"`
	// FavaOptions is the opt-in Fava option subset.
	FavaOptions map[string]string `json:"fava_options"`
	// Errors are redacted diagnostics.
	Errors []AdapterDiagnostic `json:"errors"`
	// BaseURL is the loopback base the embedded shell is served at.
	BaseURL string `json:"base_url"`
	// Extensions is always empty (excluded per ADR-0024).
	Extensions []string `json:"extensions"`
	// OtherLedgers is always empty (single-ledger product).
	OtherLedgers []string `json:"other_ledgers"`
	// Incognito is always false (no incognito mode projection).
	Incognito bool `json:"incognito"`
	// HaveExcel is always false (Excel export out of scope).
	HaveExcel bool `json:"have_excel"`
	// SnapshotID is the immutable snapshot identifier (redaction-safe, not a
	// path). Exposed for the shell header only.
	SnapshotID string `json:"snapshot_id"`
	// Valid reports whether the current snapshot is valid.
	Valid bool `json:"valid"`
}

// AdapterDiagnostic is a redacted, display-safe diagnostic. The frontend
// errors validator expects {type, message, source{filename,lineno}|null};
// this is the independent Go mapping (contract row: errors).
type AdapterDiagnostic struct {
	// Type echoes the diagnostic code (e.g. "E-CUSTOM").
	Type string `json:"type"`
	// Message is the diagnostic message.
	Message string `json:"message"`
	// Source is the display-safe source location, or nil.
	Source *AdapterSource `json:"source"`
}

// AdapterSource is a display-safe source location. Filename is a
// source.Graph.DisplayPath identifier, never an absolute filesystem path.
type AdapterSource struct {
	Filename string `json:"filename"`
	Lineno   int    `json:"lineno"`
}

// BootstrapOptions are the projection inputs. The adapter handler supplies
// the current snapshot (may be nil) and the base URL.
type BootstrapOptions struct {
	Snapshot *snapshot.Snapshot
	BaseURL  string
}

// BootstrapProjection builds a Bootstrap from the current snapshot. It is
// pure: it never mutates the snapshot or evaluation, never reads source
// files, and emits only display-safe values. A nil snapshot yields a valid
// but empty bootstrap (valid=false), which is the correct "no valid snapshot"
// state.
func BootstrapProjection(opts BootstrapOptions) Bootstrap {
	proj := Bootstrap{
		BaseURL:      opts.BaseURL,
		Extensions:   []string{},
		OtherLedgers: []string{},
		FavaOptions:  map[string]string{},
		Options:      map[string]string{},
	}
	if opts.Snapshot == nil {
		return proj
	}
	proj.SnapshotID = opts.Snapshot.ID
	proj.Valid = opts.Snapshot.Valid()
	evaluation := opts.Snapshot.Evaluation()
	graph := opts.Snapshot.Graph()

	// Accounts, currencies, payees, tags, links, years.
	accounts := make([]string, 0, len(evaluation.Accounts))
	for name := range evaluation.Accounts {
		accounts = append(accounts, name)
	}
	sort.Strings(accounts)
	proj.Accounts = accounts

	currencies := collectCurrencies(evaluation)
	proj.Currencies = currencies

	payees, tags, links, years := collectDimensions(evaluation)
	proj.Payees = payees
	proj.Tags = tags
	proj.Links = links
	proj.Years = years

	// Options: copy the evaluated options map (already a copy from the
	// snapshot), then apply the opt-in Fava option subset.
	for key, value := range evaluation.Options {
		if key == "" {
			continue
		}
		proj.Options[key] = value
	}
	proj.FavaOptions = projectFavaOptions(optionsDefault(evaluation.Options))

	// Errors: redacted diagnostics.
	proj.Errors = projectDiagnostics(opts.Snapshot.Diagnostics(), graph)
	return proj
}

// optionsDefault returns the subset of evaluated options that the frontend
// reads for shell display (title, operating_currency, name_*). Unknown keys
// are ignored. This is a presentation projection, not a semantic one.
func optionsDefault(options map[string]string) map[string]string {
	subset := map[string]string{}
	for _, key := range []string{"title", "operating_currency", "name_assets", "name_liabilities", "name_equity", "name_income", "name_expenses"} {
		if value, ok := options[key]; ok {
			subset[key] = value
		}
	}
	return subset
}

// projectFavaOptions maps the opt-in fava-option subset (SupportedFavaOptions)
// onto the values present in the options map. Only allowlisted keys are
// projected; an unknown key is never emitted.
func projectFavaOptions(options map[string]string) map[string]string {
	result := map[string]string{}
	for _, option := range SupportedFavaOptions {
		if value, ok := options[option.Key]; ok && value != "" {
			result[option.Key] = value
		}
	}
	return result
}

// collectCurrencies returns the sorted, deduplicated currency set derived from
// evaluated account balances and price-map base/quote currencies. No
// commodity directive or private data is read.
func collectCurrencies(evaluation ledger.Evaluation) []string {
	set := map[string]bool{}
	for _, state := range evaluation.Accounts {
		for currency := range state.Balances {
			if currency != "" {
				set[currency] = true
			}
		}
		for _, position := range state.Positions {
			if position.Currency != "" {
				set[position.Currency] = true
			}
			if position.Cost != nil && position.Cost.Currency != "" {
				set[position.Cost.Currency] = true
			}
		}
	}
	for base, quotes := range evaluation.Prices {
		if base != "" {
			set[base] = true
		}
		for _, quote := range quotes {
			if quote.Currency != "" {
				set[quote.Currency] = true
			}
		}
	}
	return sortedKeys(set)
}

// collectDimensions returns the sorted, deduplicated payee, tag, link, and
// year lists derived from evaluated entries. It inspects only directive
// fields, never raw source text.
func collectDimensions(evaluation ledger.Evaluation) (payees, tags, links, years []string) {
	payeeSet := map[string]bool{}
	tagSet := map[string]bool{}
	linkSet := map[string]bool{}
	yearSet := map[string]bool{}
	for _, entry := range evaluation.Entries {
		if entry.Date.Raw != "" {
			if len(entry.Date.Raw) >= 4 {
				yearSet[entry.Date.Raw[:4]] = true
			}
		}
		switch d := entry.Directive.(type) {
		case ledger.Transaction:
			if d.Payee != "" {
				payeeSet[d.Payee] = true
			}
			for _, tag := range d.Tags {
				if tag != "" {
					tagSet[tag] = true
				}
			}
			for _, link := range d.Links {
				if link != "" {
					linkSet[link] = true
				}
			}
		case ledger.Document:
			for _, tag := range d.Tags {
				if tag != "" {
					tagSet[tag] = true
				}
			}
			for _, link := range d.Links {
				if link != "" {
					linkSet[link] = true
				}
			}
		}
	}
	return sortedKeys(payeeSet), sortedKeys(tagSet), sortedKeys(linkSet), sortedKeys(yearSet)
}

// projectDiagnostics maps diagnostics to display-safe AdapterDiagnostic
// values. Paths are resolved through the graph's DisplayPath when available;
// otherwise source.SafeDisplayPath strips absolute context. A synthetic source
// is emitted only when a line number is present.
func projectDiagnostics(diagnostics []diagnostic.Diagnostic, graph *source.Graph) []AdapterDiagnostic {
	if len(diagnostics) == 0 {
		return nil
	}
	result := make([]AdapterDiagnostic, 0, len(diagnostics))
	for _, diag := range diagnostics {
		adapter := AdapterDiagnostic{Type: diag.Code, Message: diag.Message}
		if diag.Span.StartLine > 0 {
			filename := source.SafeDisplayPath(diag.Path)
			if graph != nil {
				if graph.File(diag.Span.File) != nil {
					filename = graph.DisplayPath(diag.Span.File)
				} else if id, ok := graph.ByPath[diag.Path]; ok {
					filename = graph.DisplayPath(id)
				}
			}
			adapter.Source = &AdapterSource{Filename: filename, Lineno: diag.Span.StartLine}
		}
		result = append(result, adapter)
	}
	return result
}

// sortedKeys returns the sorted non-empty keys of a string set.
func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// normalizeBaseURL trims trailing slashes and returns a path-safe base. It
// never returns an absolute URL or a scheme; the adapter serves loopback only.
func normalizeBaseURL(base string) string {
	base = strings.TrimSpace(base)
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		return "/"
	}
	return base
}
