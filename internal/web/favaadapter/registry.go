// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package favaadapter

// ContractRegistry declares every private adapter route that the transplanted
// frontend may call. It is the single place where an internal route name is
// bound to its Go domain owner, semantic authority, request parameters, and
// error shape, so that no code path can accidentally be documented or used as
// a public Fava-compatible API (ADR-0033).
//
// The registry deliberately carries no HTTP metadata (methods, path
// templates): route wiring belongs to internal/web and depends on the P2
// frontend build. This phase only declares intent so the contract layer is
// reviewable before any handler exists.

// RouteName is the internal identity of one private adapter endpoint. The
// string value is the internal identifier used in tests and future wiring; it
// is intentionally NOT a URL path and is never documented for clients.
type RouteName string

// Constants are the adapter routes needed by the transplanted frontend in
// dependency order (bootstrap first). Names are internal identifiers only.
const (
	RouteLedgerData    RouteName = "ledger-data"         // P3 bootstrap (contract row: ledger_data)
	RouteStatus        RouteName = "status"              // minimal health/snapshot state
	RouteErrors        RouteName = "errors"              // serialised diagnostics (contract row: errors)
	RouteMetadata1     RouteName = "metadata-projection" // P3 report/metadata projection (commodities/accounts preview)
	RouteJournal       RouteName = "journal"             // P4; declared now for registry completeness
	RouteTreeReport    RouteName = "tree-report"         // P4/P5; declared now
	RouteQueryShell    RouteName = "query"               // P7; declared now
	RouteEditorSource  RouteName = "source"              // P7; declared now
	RouteEditorSave    RouteName = "source-write"        // P7; declared now
	RouteImportPreview RouteName = "import-preview"      // P7; declared now
	RouteOptions       RouteName = "options"             // P7; declared now
	RouteDownload      RouteName = "download-journal"    // filtered entries as Beancount source
)

// ErrorShape enumerates the standardized adapter error shapes.
type ErrorShape string

const (
	// ErrorFull is the {"error": "message"} payload (AdapterError).
	ErrorFull ErrorShape = "full"
	// ErrorStatusCode is a status code plus the ErrorFull payload; used by
	// route wire adapters in P3 phase 2, never exposed as a public contract.
	ErrorStatusCode ErrorShape = "status+full"
)

// Authority names the semantic authority for a route. Framing each route with
// its authority makes it structurally impossible to treat the adapter as an
// accounting authority (ADR-0026: Fava decides UX, Beancount v3 decides
// ledger semantics).
type Authority string

const (
	// AuthorityLedger means the result is Beancount v3 ledger semantics.
	AuthorityLedger Authority = "ledger-v3"
	// AuthorityPresentation means the result is presentation only (locale,
	// projection, ordering) and never alters ledger meaning.
	AuthorityPresentation Authority = "presentation"
)

// RequestSpec documents the parameters a route accepts. Only strings and
// booleans are allowed in this phase; structured bodies arrive with phase 2
// wiring.
type RequestSpec struct {
	Params []string
	Body   bool
}

// Contract binds a route name to its owner, authority, error shape, and
// request spec.
type Contract struct {
	Route     RouteName
	Owner     string
	Authority Authority
	Errors    ErrorShape
	Request   RequestSpec
}

// Registry is the full private adapter route table. Any code that inspects
// these entries must treat them as internal facts; do not key public HTTP
// documentation on them.
var Registry = [...]Contract{
	{Route: RouteLedgerData, Owner: "internal/web/favaadapter: Bootstrap, internal/snapshot.Store, internal/report", Authority: AuthorityPresentation, Errors: ErrorFull, Request: RequestSpec{}},
	{Route: RouteStatus, Owner: "internal/web/favaadapter: Bootstrap + internal/snapshot.Store", Authority: AuthorityPresentation, Errors: ErrorFull, Request: RequestSpec{}},
	{Route: RouteErrors, Owner: "internal/report.ErrorsWithGraph + internal/diagnostic", Authority: AuthorityLedger, Errors: ErrorFull, Request: RequestSpec{}},
	{Route: RouteMetadata1, Owner: "internal/report: Prices/Accounts + internal/snapshot.Store", Authority: AuthorityPresentation, Errors: ErrorFull, Request: RequestSpec{Params: []string{"limit"}}},
	{Route: RouteJournal, Owner: "internal/report.JournalBetween + internal/report.FilterJournal", Authority: AuthorityLedger, Errors: ErrorFull, Request: RequestSpec{Params: []string{"from", "to", "account", "filter", "page", "order"}}},
	{Route: RouteTreeReport, Owner: "internal/report.BalanceSheet/IncomeStatement/TrialBalanceTree + internal/report/charts.go", Authority: AuthorityLedger, Errors: ErrorFull, Request: RequestSpec{Params: []string{"name", "conversion", "interval", "time"}}},
	{Route: RouteQueryShell, Owner: "internal/query.Evaluate", Authority: AuthorityLedger, Errors: ErrorFull, Request: RequestSpec{Params: []string{"query_string", "account", "filter", "time"}}},
	{Route: RouteEditorSource, Owner: "internal/web (editor read) + internal/source.Graph", Authority: AuthorityLedger, Errors: ErrorFull, Request: RequestSpec{Params: []string{"filename"}}},
	{Route: RouteEditorSave, Owner: "internal/web (editor write path: atomic replace + backup + revalidate)", Authority: AuthorityLedger, Errors: ErrorFull, Request: RequestSpec{Body: true}},
	{Route: RouteImportPreview, Owner: "internal/web (import preview: parse validate only; never publishes)", Authority: AuthorityLedger, Errors: ErrorFull, Request: RequestSpec{Body: true}},
	{Route: RouteOptions, Owner: "internal/web (handleOptions) + internal/snapshot.Evaluation.Options", Authority: AuthorityPresentation, Errors: ErrorFull, Request: RequestSpec{Body: true}},
	{Route: RouteDownload, Owner: "internal/web/favaadapter.ExportEntries + internal/source.Graph", Authority: AuthorityLedger, Errors: ErrorStatusCode, Request: RequestSpec{Params: []string{"from", "to", "account", "filter"}}},
}

// Lookup returns the contract for a route, or false if the route is not
// registered. Contract lookup is cheap and linear; the registry is small by
// design (an internal allowlist, not a public catalog).
func Lookup(route RouteName) (Contract, bool) {
	for _, contract := range Registry {
		if contract.Route == route {
			return contract, true
		}
	}
	return Contract{}, false
}
