// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"orangecount/internal/ledger"
	"orangecount/internal/query"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	current := s.store.Current()
	if current == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/reports/"), "/")
	result, chartRoute, known, err := s.buildReport(r, current, name)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !known {
		http.NotFound(w, r)
		return
	}
	if format := strings.TrimSpace(r.URL.Query().Get("format")); strings.EqualFold(format, "csv") {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		if err := result.WriteCSV(w); err != nil {
			return
		}
		return
	} else if format != "" && !strings.EqualFold(format, "json") {
		writeAPIError(w, http.StatusBadRequest, "format must be json or csv")
		return
	}
	presented := report.Present(result)
	if chartRoute == "" {
		writeJSON(w, presented)
		return
	}
	evaluation := current.Evaluation()
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	valuation := strings.TrimSpace(r.URL.Query().Get("valuation"))
	if valuation == "" {
		valuation = "at-cost"
	}
	chart := report.ReportChart(evaluation, chartRoute, period, strings.TrimSpace(r.URL.Query().Get("currency")), valuation, strings.TrimSpace(r.URL.Query().Get("account")))
	// The statement trees additionally carry the per-measure, per-currency
	// chart set: one series per currency, no conversion. A single-currency
	// display needs a quote for every foreign posting, so a ledger without
	// price directives would blank those series with a "no conversion
	// quote" warning while the table below shows both currencies fine.
	charts := []report.PresentedChartSpec{}
	for _, spec := range report.ReportCharts(evaluation, chartRoute, period, valuation) {
		charts = append(charts, report.PresentChart(spec))
	}
	writeJSON(w, struct {
		query.Result
		Chart  report.PresentedChartSpec   `json:"chart"`
		Charts []report.PresentedChartSpec `json:"charts,omitempty"`
	}{Result: presented, Chart: report.PresentChart(chart), Charts: charts})
}

// buildReport is the shared semantic/report projection used by both the
// existing OrangeCount report endpoint and the private Fava-shaped adapter.
// Keeping one builder prevents the two surfaces from drifting in filtering,
// redaction, or report ownership. Known-but-failing parses are reported as
// (result, route, true, err); unknown names as (…, false, nil).
func (s *Server) buildReport(r *http.Request, current *snapshot.Snapshot, name string) (query.Result, string, bool, error) {
	result, chartRoute, err := reportForRequest(r, current, name)
	if err != nil {
		return query.Result{}, "", true, err
	}
	if !reportKnown(name) {
		return query.Result{}, "", false, nil
	}
	result = redactQueryPaths(result, current.Graph())
	filters, err := globalReportFilters(r)
	if err != nil {
		return query.Result{}, "", true, err
	}
	// The pivot applies the global filters itself while accumulating (its rows
	// are intervals, not postings), so the generic row filter must not re-run.
	if !strings.EqualFold(name, "pivot") {
		result = report.Filter(result, filters)
	}
	if strings.EqualFold(name, "journal") {
		result = report.FilterJournal(result, journalFiltersFromQuery(r))
	}
	return result, chartRoute, true, nil
}

// reportKnown reports whether name maps to a report this server serves.
func reportKnown(name string) bool {
	_, ok := lookupReportRoute(name)
	return ok
}

// reportRoute binds one report name to its builder and optional chart route.
// The builder receives the raw request (for its filter parameters), the
// evaluation, and the snapshot (for graph-backed reports).
type reportRoute struct {
	aliases []string
	chart   string
	build   func(*http.Request, ledger.Evaluation, *snapshot.Snapshot) (query.Result, string, error)
}

// reportRoutes is the single name table behind both reportKnown and
// reportForRequest, so the served set and the alias spellings cannot drift.
var reportRoutes = []reportRoute{
	{aliases: []string{"accounts", "account"}, chart: "accounts", build: func(r *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
		return accountReport(r, evaluation)
	}},
	{aliases: []string{"journal"}, build: func(r *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
		from, to, err := journalDateRange(r)
		if err != nil {
			return query.Result{}, "", err
		}
		return report.JournalBetween(evaluation, from, to), "", nil
	}},
	// The web trial-balance report needs explicit ancestors for Fava-style
	// hierarchy rendering. Keep report.TrialBalance flat for query-compatible
	// consumers and use its tree variant only at this presentation boundary.
	{aliases: []string{"trial-balance", "trial_balance", "trialbalance"}, chart: "trial-balance", build: func(_ *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
		return report.TrialBalanceTree(evaluation), "trial-balance", nil
	}},
	{aliases: []string{"balance-sheet", "balance_sheet", "balancesheet"}, chart: "balance-sheet", build: func(_ *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
		return report.BalanceSheet(evaluation), "balance-sheet", nil
	}},
	{aliases: []string{"income-statement", "income_statement", "incomestatement"}, chart: "income-statement", build: func(_ *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
		return report.IncomeStatement(evaluation), "income-statement", nil
	}},
	{aliases: []string{"holdings"}, build: func(r *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
		return holdingsReport(r, evaluation)
	}},
	{aliases: []string{"pivot"}, build: reportPivot},
	{aliases: []string{"prices", "price"}, build: func(_ *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
		return report.Prices(evaluation), "", nil
	}},
	{aliases: []string{"commodities", "commodity"}, build: func(_ *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
		return report.Commodities(evaluation), "", nil
	}},
	{aliases: []string{"events", "event"}, build: func(_ *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
		return report.Events(evaluation), "", nil
	}},
	{aliases: []string{"documents", "document"}, build: func(_ *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
		return report.Documents(evaluation), "", nil
	}},
	{aliases: []string{"statistics", "statistic", "stats"}, build: func(_ *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
		return report.Statistics(evaluation), "", nil
	}},
	{aliases: []string{"errors", "error"}, build: func(_ *http.Request, evaluation ledger.Evaluation, current *snapshot.Snapshot) (query.Result, string, error) {
		return report.ErrorsWithGraph(evaluation, current.Graph()), "", nil
	}},
}

// lookupReportRoute resolves a request's report name (any alias, any case) to
// its route.
func lookupReportRoute(name string) (reportRoute, bool) {
	requested := strings.ToLower(name)
	for _, route := range reportRoutes {
		for _, alias := range route.aliases {
			if alias == requested {
				return route, true
			}
		}
	}
	return reportRoute{}, false
}

// reportForRequest computes the unfiltered result and chart route for one
// report name. A nil result with empty columns and nil error means the name
// was unknown.
func reportForRequest(r *http.Request, current *snapshot.Snapshot, name string) (query.Result, string, error) {
	route, ok := lookupReportRoute(name)
	if !ok {
		return query.Result{}, "", nil
	}
	return route.build(r, current.Evaluation(), current)
}

// reportPivot builds the Excel-style cross-tab from the request's row,
// column, value, and filter parameters.
func reportPivot(r *http.Request, evaluation ledger.Evaluation, _ *snapshot.Snapshot) (query.Result, string, error) {
	filters, err := globalReportFilters(r)
	if err != nil {
		return query.Result{}, "", err
	}
	spec := report.PivotSpec{
		Rows:    r.URL.Query().Get("rows"),
		Columns: r.URL.Query().Get("columns"),
		Values:  r.URL.Query().Get("values"),
		Account: r.URL.Query().Get("account"),
		Filters: filters,
	}
	return report.PivotTable(evaluation, spec), "", nil
}

// accountReport computes the accounts result. Fava's account page switches
// between the journal, per-interval changes, and per-interval balances with
// the `r` query parameter; the aggregate interval rows carry no entry columns,
// so they skip the generic row filters (time filters are applied inside).
func accountReport(r *http.Request, evaluation ledger.Evaluation) (query.Result, string, error) {
	view := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("r")))
	if view == "changes" || view == "balances" {
		filters, err := globalReportFilters(r)
		if err != nil {
			return query.Result{}, "", err
		}
		result := report.AccountIntervals(evaluation, strings.TrimSpace(r.URL.Query().Get("account")), view, strings.TrimSpace(r.URL.Query().Get("interval")), filters)
		return result, "accounts", nil
	}
	return report.Accounts(evaluation), "accounts", nil
}

// holdingsReport applies the optional as-of date, valuation, and aggregation
// selectors to the holdings result.
func holdingsReport(r *http.Request, evaluation ledger.Evaluation) (query.Result, string, error) {
	asOf, err := reportAsOfDate(r)
	if err != nil {
		return query.Result{}, "", err
	}
	valuation := strings.TrimSpace(r.URL.Query().Get("valuation"))
	if valuation == "" {
		valuation = "at-cost"
	}
	result := report.HoldingsAtCurrency(evaluation, asOf, valuation, strings.TrimSpace(r.URL.Query().Get("currency")))
	result = report.HoldingsAggregate(result, strings.TrimSpace(r.URL.Query().Get("aggregation")))
	return result, "", nil
}

// globalReportFilters parses the filter controls every report surface shares
// (semantic reports, the journal projection, and downloads). Parse errors are
// returned so the API edge can answer 400 instead of silently matching
// nothing; each dimension is delegated to a focused parser below.
func globalReportFilters(r *http.Request) (report.Filters, error) {
	text := strings.TrimSpace(r.URL.Query().Get("filter"))
	if text != "" {
		// Fava rejects an unparseable filter at the API edge; validating here
		// lets the shell show the parse error instead of silently dropping
		// entries with a different filter semantics.
		if _, err := report.ParseFQL(text); err != nil {
			return report.Filters{}, err
		}
	}
	period, err := periodFilter(r)
	if err != nil {
		return report.Filters{}, err
	}
	if valuation := strings.TrimSpace(r.URL.Query().Get("valuation")); valuation != "" && valuation != "at-cost" && valuation != "market-value" {
		return report.Filters{}, fmt.Errorf("invalid valuation filter")
	}
	prefix, begin, end, err := reportTimeFilter(strings.TrimSpace(r.URL.Query().Get("time")), time.Now())
	if err != nil {
		return report.Filters{}, err
	}
	return report.Filters{
		Account:    strings.TrimSpace(r.URL.Query().Get("account")),
		Text:       text,
		Period:     period,
		TimePrefix: prefix,
		TimeBegin:  begin,
		TimeEnd:    end,
	}, nil
}

// periodFilter validates the interval selector. "all" and the empty string
// mean unfiltered and normalize to an empty Period.
func periodFilter(r *http.Request) (string, error) {
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" || period == "all" {
		return "", nil
	}
	switch period {
	case "month", "quarter", "year":
		return period, nil
	default:
		return "", fmt.Errorf("invalid period filter")
	}
}

// reportTimeFilter resolves Fava's `time` vocabulary into either a date
// prefix or a half-open YYYY-MM-01 begin/end range — the two shapes
// report.Filters can express. Relative words ("year", "month") anchor at
// now; explicit YYYY, YYYY-MM, and YYYY-Qn forms are validated strictly so a
// typo becomes a 400 rather than a filter that silently matches nothing.
func reportTimeFilter(raw string, now time.Time) (prefix, begin, end string, err error) {
	switch raw {
	case "", "all":
		return "", "", "", nil
	case "year":
		return now.Format("2006"), "", "", nil
	case "month":
		return now.Format("2006-01"), "", "", nil
	}
	if isNumericYear(raw) {
		return raw, "", "", nil
	}
	if begin, end, ok := quarterTimeRange(raw); ok {
		return "", begin, end, nil
	}
	if _, parseErr := time.Parse("2006-01", raw); parseErr == nil {
		return raw, "", "", nil
	}
	return "", "", "", fmt.Errorf("invalid time filter")
}

// isNumericYear reports whether raw is exactly four ASCII digits.
func isNumericYear(raw string) bool {
	if len(raw) != 4 {
		return false
	}
	_, err := strconv.Atoi(raw)
	return err == nil
}

// quarterTimeRange parses Fava's "2025-Q2" syntax into the half-open month
// range a prefix cannot express. ok is false for anything else, including a
// syntactically valid quarter out of range.
func quarterTimeRange(raw string) (begin, end string, ok bool) {
	if len(raw) != 7 || raw[4] != '-' || (raw[5] != 'Q' && raw[5] != 'q') {
		return "", "", false
	}
	year, yearErr := strconv.Atoi(raw[:4])
	quarter, quarterErr := strconv.Atoi(raw[6:])
	if yearErr != nil || quarterErr != nil || quarter < 1 || quarter > 4 {
		return "", "", false
	}
	beginMonth := (quarter-1)*3 + 1
	endYear, endMonth := year, beginMonth+3
	if endMonth > 12 {
		endYear++
		endMonth = 1
	}
	return fmt.Sprintf("%04d-%02d-01", year, beginMonth), fmt.Sprintf("%04d-%02d-01", endYear, endMonth), true
}

func journalDateRange(r *http.Request) (*ledger.Date, *ledger.Date, error) {
	from, err := parseISODate(r.URL.Query().Get("from"))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid from date: %w", err)
	}
	to, err := parseISODate(r.URL.Query().Get("to"))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid to date: %w", err)
	}
	if from != nil && to != nil && from.Raw > to.Raw {
		return nil, nil, fmt.Errorf("from date must not be after to date")
	}
	return from, to, nil
}

func parseISODate(raw string) (*ledger.Date, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("expected YYYY-MM-DD")
	}
	year, month, day := parsed.Date()
	if year <= 0 {
		return nil, fmt.Errorf("expected a positive year")
	}
	return &ledger.Date{Year: year, Month: int(month), Day: day, Raw: raw}, nil
}

func reportAsOfDate(r *http.Request) (string, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("as_of"))
	if raw == "" {
		return "", nil
	}
	if _, err := parseISODate(raw); err != nil {
		return "", fmt.Errorf("invalid as-of date: %w", err)
	}
	return raw, nil
}

// handleQuery (GET) runs a read-only query against the current snapshot and
// returns rows in JSON or CSV per the format parameter.
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	text := strings.TrimSpace(r.URL.Query().Get("q"))
	if text == "" {
		writeAPIError(w, http.StatusBadRequest, "query parameter q is required")
		return
	}
	current := s.store.Current()
	if current == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	result, err := query.Evaluate(text, current.Evaluation())
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	result = redactQueryPaths(result, current.Graph())
	if strings.EqualFold(r.URL.Query().Get("format"), "csv") {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		if err := result.WriteCSV(w); err != nil {
			return
		}
		return
	}
	if format := strings.TrimSpace(r.URL.Query().Get("format")); format != "" && !strings.EqualFold(format, "json") {
		writeAPIError(w, http.StatusBadRequest, "format must be json or csv")
		return
	}
	writeJSON(w, result)
}

func redactQueryPaths(result query.Result, graph *source.Graph) query.Result {
	for _, row := range result.Rows {
		for _, column := range []string{"file", "path"} {
			value, ok := row[column].(string)
			if !ok || value == "" {
				continue
			}
			if graph != nil {
				if id, found := graph.ByPath[value]; found {
					row[column] = graph.DisplayPath(id)
					continue
				}
			}
			row[column] = source.SafeDisplayPath(value)
		}
	}
	return result
}
