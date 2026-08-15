// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"net/http"
	"strings"

	"orangecount/internal/diagnostic"
	"orangecount/internal/query"
	"orangecount/internal/repairguidance"
	"orangecount/internal/report"
	"orangecount/internal/snapshot"
	"orangecount/internal/web/favaadapter"
)

// The private Fava-shaped adapter lives under /__orangecount/fava/. Its
// surface is a flat set of sub-resources; handleFavaAdapter owns the shared
// preconditions (method allowlist, snapshot availability) and then dispatches
// each sub-resource to one small handler through favaAdapterRoutes. Keeping
// the handlers separate keeps each endpoint independently testable and the
// dispatch table the single place to audit the surface.

// favaAdapterWritePaths are the only sub-resources that accept POST; every
// other route is read-only GET. All of them are reviewed write paths
// (ledger append, attachment upload/move, quick-entry) guarded by
// requireSameOrigin inside their handler.
var favaAdapterWritePaths = map[string]bool{
	"add-entries":        true,
	"document":           true,
	"move-document":      true,
	"quick-preview":      true,
	"quick-commit":       true,
	"quick-undo":         true,
	"quick-profile":      true,
	"quick-profile-save": true,
}

// favaHandler is the signature every adapter sub-resource handler shares.
// current is guaranteed non-nil and is the snapshot the response envelope
// should be stamped with.
type favaHandler func(s *Server, w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot)

// favaAdapterRoutes maps one adapter sub-resource to its handler. The three
// statement trees share one handler because they only differ in projection
// name.
var favaAdapterRoutes = map[string]favaHandler{
	"changed":            (*Server).favaChanged,
	"ledger_data":        (*Server).favaLedgerData,
	"metadata":           (*Server).favaMetadata,
	"options":            (*Server).favaOptions,
	"help":               (*Server).favaHelp,
	"diagnostics":        (*Server).favaDiagnostics,
	"editor":             (*Server).favaEditor,
	"import":             (*Server).favaImport,
	"source":             (*Server).favaSource,
	"journal":            (*Server).favaJournal,
	"download-journal":   (*Server).favaDownloadJournal,
	"entry-context":      (*Server).favaEntryContext,
	"add-entries":        (*Server).favaAddEntries,
	"quick-preview":      (*Server).handleQuickPreview,
	"quick-commit":       (*Server).handleQuickCommit,
	"quick-undo":         (*Server).handleQuickUndo,
	"quick-profile":      (*Server).handleQuickProfile,
	"quick-profile-save": (*Server).handleQuickProfileSave,
	"document":           (*Server).handleDocumentUpload,
	"move-document":      (*Server).handleDocumentMove,
	"income_statement":   (*Server).favaTreeReport,
	"balance_sheet":      (*Server).favaTreeReport,
	"trial_balance":      (*Server).favaTreeReport,
}

// handleFavaAdapter serves the private Fava-shaped API: method gating, the
// no-snapshot guard, then sub-resource dispatch. Unknown sub-resources fall
// through to the reports/ namespace or 404.
func (s *Server) handleFavaAdapter(w http.ResponseWriter, r *http.Request) {
	resource := strings.Trim(strings.TrimPrefix(r.URL.Path, "/__orangecount/fava/"), "/")
	if r.Method != http.MethodGet && !(r.Method == http.MethodPost && favaAdapterWritePaths[resource]) {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	current := s.store.Current()
	if current == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	if handler, ok := favaAdapterRoutes[resource]; ok {
		handler(s, w, r, current)
		return
	}
	if name := strings.TrimPrefix(resource, "reports/"); name != resource {
		s.favaReport(name, w, r, current)
		return
	}
	http.NotFound(w, r)
}

// favaChanged implements Fava's changed endpoint: a lightweight mtime
// comparison so the shell can poll for external ledger edits.
func (s *Server) favaChanged(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	previous := strings.TrimSpace(r.URL.Query().Get("mtime"))
	currentMtime := favaadapter.NewEnvelope(nil, current.BuiltAt).Mtime
	writeJSON(w, favaadapter.NewEnvelope(previous != "" && previous != currentMtime, current.BuiltAt))
}

// favaLedgerData returns the full bootstrap projection the transplanted Fava
// shell needs on first load.
func (s *Server) favaLedgerData(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	projection := favaadapter.BootstrapProjection(favaadapter.BootstrapOptions{Snapshot: current, BaseURL: "/", DocumentRoots: s.roots.Paths()})
	writeJSON(w, favaadapter.NewEnvelope(projection, current.BuiltAt))
}

// favaMetadata returns option-style key/value projections for the shell.
func (s *Server) favaMetadata(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	root := strings.TrimSpace(r.URL.Query().Get("root"))
	projection := favaadapter.MetadataProjectionOptions(favaadapter.MetadataOptions{Evaluation: current.Evaluation(), Root: root})
	writeJSON(w, favaadapter.NewEnvelope(projection, current.BuiltAt))
}

// favaOptions merges evaluation options with locally overridden ones; the
// fava_options half is static presentation configuration.
func (s *Server) favaOptions(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	options := map[string]string{}
	for key, value := range current.Evaluation().Options {
		options[key] = value
	}
	s.optionsMu.RLock()
	for key, value := range s.options {
		options[key] = value
	}
	s.optionsMu.RUnlock()
	writeJSON(w, favaadapter.NewEnvelope(struct {
		Options     map[string]string `json:"options"`
		FavaOptions map[string]string `json:"fava_options"`
	}{Options: options, FavaOptions: map[string]string{"locale": "en", "theme": "system"}}, current.BuiltAt))
}

// favaHelp serves the help index or one localized diagnostics topic.
func (s *Server) favaHelp(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if topic := strings.TrimSpace(r.URL.Query().Get("topic")); topic != "" {
		code := strings.TrimPrefix(topic, "diagnostics/")
		guide, ok := repairguidance.Lookup(code, requestedLocale(r))
		if !ok || topic != guide.Topic {
			writeAPIError(w, http.StatusNotFound, helpTopicNotFoundMessage(requestedLocale(r)))
			return
		}
		writeJSON(w, favaadapter.NewEnvelope(guide, current.BuiltAt))
		return
	}
	writeJSON(w, favaadapter.NewEnvelope(struct {
		Sections []map[string]string `json:"sections"`
	}{Sections: helpSections(requestedLocale(r))}, current.BuiltAt))
}

// favaDiagnostics returns localized diagnostics with display paths.
func (s *Server) favaDiagnostics(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	locale := requestedLocale(r)
	graph := current.Graph()
	values := s.store.Diagnostics()
	localized := make([]diagnostic.Diagnostic, len(values))
	for i, value := range values {
		localized[i] = diagnostic.Localize(value, locale)
		localized[i].Path = displayDiagnosticPath(localized[i], graph)
	}
	writeJSON(w, favaadapter.NewEnvelope(localized, current.BuiltAt))
}

// favaEditor serves the include-graph file list or one file's content.
func (s *Server) favaEditor(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	requested := strings.TrimSpace(r.URL.Query().Get("path"))
	if requested == "" {
		writeJSON(w, favaadapter.NewEnvelope(struct {
			Paths      []string `json:"paths"`
			Entry      string   `json:"entry"`
			SnapshotID string   `json:"snapshot_id"`
		}{Paths: graph.DisplayPaths(), Entry: graph.DisplayPath(graph.Entry), SnapshotID: current.ID}, current.BuiltAt))
		return
	}
	file, display, ok := graphFile(graph, requested)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, favaadapter.NewEnvelope(struct {
		Path       string `json:"path"`
		Content    string `json:"content"`
		SnapshotID string `json:"snapshot_id"`
	}{Path: display, Content: string(file.Data), SnapshotID: current.ID}, current.BuiltAt))
}

// favaImport lists importable sources and known adapters.
func (s *Server) favaImport(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	if r.URL.Query().Get("kind") == "adapters" {
		writeJSON(w, favaadapter.NewEnvelope(struct {
			Adapters []map[string]any `json:"adapters"`
		}{Adapters: []map[string]any{
			{"id": "beancount", "label": "Beancount source", "extensions": []string{".bean", ".beancount"}},
			{"id": "csv", "label": "Generic CSV", "extensions": []string{".csv"}},
		}}, current.BuiltAt))
		return
	}
	writeJSON(w, favaadapter.NewEnvelope(struct {
		Paths      []string `json:"paths"`
		Entry      string   `json:"entry"`
		SnapshotID string   `json:"snapshot_id"`
	}{Paths: graph.DisplayPaths(), Entry: graph.DisplayPath(graph.Entry), SnapshotID: current.ID}, current.BuiltAt))
}

// favaSource serves source file contents by display path.
func (s *Server) favaSource(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	requested := strings.TrimSpace(r.URL.Query().Get("path"))
	if requested == "" {
		writeJSON(w, favaadapter.NewEnvelope(struct {
			Paths []string `json:"paths"`
		}{Paths: graph.DisplayPaths()}, current.BuiltAt))
		return
	}
	id, ok := graph.FileIDForDisplayPath(requested)
	if !ok {
		id, ok = graph.ByPath[requested]
	}
	if !ok || graph.File(id) == nil {
		http.NotFound(w, r)
		return
	}
	file := graph.File(id)
	writeJSON(w, favaadapter.NewEnvelope(struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}{Path: graph.DisplayPath(id), Content: string(file.Data)}, current.BuiltAt))
}

// favaJournal projects the filtered journal for the shell's journal view.
func (s *Server) favaJournal(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	filters, err := globalReportFilters(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	projection := favaadapter.ProjectJournal(current.Evaluation(), current.Graph(), filters, journalFiltersFromQuery(r))
	writeJSON(w, favaadapter.NewEnvelope(projection, current.BuiltAt))
}

// journalFiltersFromQuery reads the directive-level journal filter controls
// shared by the journal and download endpoints.
func journalFiltersFromQuery(r *http.Request) report.JournalFilters {
	return report.JournalFilters{
		Flag:      r.URL.Query().Get("flag"),
		Tag:       r.URL.Query().Get("tag"),
		Link:      r.URL.Query().Get("link"),
		Payee:     r.URL.Query().Get("payee"),
		Narration: r.URL.Query().Get("narration"),
		Kind:      r.URL.Query().Get("kind"),
	}
}

// favaDownloadJournal streams the filtered journal as Beancount source. The
// download is a file, not an adapter envelope: the browser saves the filtered
// entries the way Fava does.
func (s *Server) favaDownloadJournal(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	filters, err := globalReportFilters(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	text := favaadapter.ExportEntries(current.Evaluation(), graph, filters, journalFiltersFromQuery(r))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="journal.bean"`)
	if _, err := w.Write([]byte(text)); err != nil {
		return
	}
}

// favaEntryContext returns the source context around one entry hash.
func (s *Server) favaEntryContext(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	entryHash := strings.TrimSpace(r.URL.Query().Get("entry_hash"))
	if entryHash == "" {
		writeAPIError(w, http.StatusBadRequest, "missing entry_hash")
		return
	}
	context, ok := favaadapter.ProjectEntryContext(current.Evaluation(), graph, entryHash)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "entry not found")
		return
	}
	writeJSON(w, favaadapter.NewEnvelope(context, current.BuiltAt))
}

// favaAddEntries appends serialized entries to the entry file and revalidates
// the whole graph before publishing a new snapshot.
func (s *Server) favaAddEntries(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	var request struct {
		Entries []favaadapter.NewEntry `json:"entries"`
	}
	if err := decodeJSONBody(w, r, &request, 1<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	serialized, err := favaadapter.SerializeNewEntries(request.Entries)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	file, display, ok := graphFile(graph, graph.DisplayPath(graph.Entry))
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "entry file unavailable")
		return
	}
	// New entries land at the end of the entry file, separated by one blank
	// line, the way Fava appends entries it inserts.
	content := strings.TrimRight(string(file.Data), "\n") + "\n\n" + serialized + "\n"
	result, backup, err := s.replaceGraphFile(current, file.Path, display, []byte(content))
	if err != nil {
		status := http.StatusUnprocessableEntity
		if result.Err != nil {
			status = http.StatusInternalServerError
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		writeJSON(w, struct {
			Published   bool                 `json:"published"`
			Backup      string               `json:"backup,omitempty"`
			Diagnostics []diagnosticResponse `json:"diagnostics"`
		}{Backup: backup, Diagnostics: diagnosticsPayload(result.Diagnostics, current.Graph())})
		return
	}
	writeJSON(w, struct {
		Published  bool   `json:"published"`
		SnapshotID string `json:"snapshot_id"`
		Backup     string `json:"backup"`
	}{Published: true, SnapshotID: result.Snapshot.ID, Backup: backup})
}

// favaTreeReport projects one of the three statement trees
// (income_statement / balance_sheet / trial_balance).
func (s *Server) favaTreeReport(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	resource := strings.Trim(strings.TrimPrefix(r.URL.Path, "/__orangecount/fava/"), "/")
	filters, err := globalReportFilters(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	projection := favaadapter.ProjectTreeReport(
		current.Evaluation(),
		resource,
		filters,
		strings.TrimSpace(r.URL.Query().Get("period")),
		strings.TrimSpace(r.URL.Query().Get("currency")),
		treeReportValuation(r),
	)
	writeJSON(w, favaadapter.NewEnvelope(projection, current.BuiltAt))
}

// treeReportValuation maps the shell's conversion selector vocabulary onto
// the valuation names the report layer understands. "units" must stay
// distinct from "at-cost", otherwise choosing it still converts lots and the
// control looks broken. An explicit valuation query parameter wins.
func treeReportValuation(r *http.Request) string {
	valuation := strings.TrimSpace(r.URL.Query().Get("valuation"))
	if valuation != "" {
		return valuation
	}
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("conversion"))) {
	case "market_value":
		return "market-value"
	case "units":
		return "units"
	default:
		return "at-cost"
	}
}

// favaReport routes one report page: the query workbench, the statistics
// composition, or a semantic report with its chart payload.
func (s *Server) favaReport(name string, w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	name = strings.Trim(name, "/")
	if name == "query" {
		s.favaReportQuery(w, r, current)
		return
	}
	if name == "statistics" {
		s.favaReportStatistics(w, r, current)
		return
	}
	s.favaSemanticReport(name, w, r, current)
}

// favaReportQuery evaluates the query workbench's query_string (or q).
func (s *Server) favaReportQuery(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	text := strings.TrimSpace(r.URL.Query().Get("query_string"))
	if text == "" {
		text = strings.TrimSpace(r.URL.Query().Get("q"))
	}
	if text == "" {
		writeAPIError(w, http.StatusBadRequest, "query_string is required")
		return
	}
	result, err := query.Evaluate(text, current.Evaluation())
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	result = redactQueryPaths(result, current.Graph())
	writeJSON(w, favaadapter.NewEnvelope(report.Present(result), current.BuiltAt))
}

// favaReportStatistics composes the datasets Fava's statistics page needs;
// the flat directive-count table alone cannot feed it.
func (s *Server) favaReportStatistics(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	evaluation := current.Evaluation()
	entriesByType := map[string]int{}
	for _, row := range report.Statistics(evaluation).Rows {
		entriesByType[row["directive"].(string)] = row["count"].(int)
	}
	payload := struct {
		EntriesByType      map[string]int                  `json:"entries_by_type"`
		PostingsPerAccount query.Result                    `json:"postings_per_account"`
		UpdateActivity     []favaadapter.UpdateActivityRow `json:"update_activity"`
	}{EntriesByType: entriesByType, PostingsPerAccount: report.Present(report.PostingsPerAccount(evaluation)), UpdateActivity: favaadapter.UpdateActivity(evaluation)}
	writeJSON(w, favaadapter.NewEnvelope(payload, current.BuiltAt))
}

// favaSemanticReport serves a named semantic report together with its chart
// payload when the report has a chart route.
func (s *Server) favaSemanticReport(name string, w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	result, chartRoute, known, err := s.buildReport(r, current, name)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !known {
		http.NotFound(w, r)
		return
	}
	presented := report.Present(result)
	payload := struct {
		query.Result
		Chart            *report.PresentedChartSpec `json:"chart,omitempty"`
		AverageCostChart *report.PresentedChartSpec `json:"average_cost_chart,omitempty"`
	}{Result: presented}
	if chartRoute != "" {
		chart := s.reportChartPayload(chartRoute, r, current)
		payload.Chart = &chart
		if extra := s.averageCostChartPayload(chartRoute, r, current); extra != nil {
			payload.AverageCostChart = extra
		}
	}
	writeJSON(w, favaadapter.NewEnvelope(payload, current.BuiltAt))
}

// reportChartPayload builds the primary chart for a chart-carrying report.
func (s *Server) reportChartPayload(chartRoute string, r *http.Request, current *snapshot.Snapshot) report.PresentedChartSpec {
	valuation := strings.TrimSpace(r.URL.Query().Get("valuation"))
	if valuation == "" {
		valuation = "at-cost"
	}
	return report.PresentChart(report.ReportChart(current.Evaluation(), chartRoute, r.URL.Query().Get("period"), r.URL.Query().Get("currency"), valuation, r.URL.Query().Get("account")))
}

// averageCostChartPayload adds the per-account average-cost drilldown chart
// on the accounts report when an account is selected.
func (s *Server) averageCostChartPayload(chartRoute string, r *http.Request, current *snapshot.Snapshot) *report.PresentedChartSpec {
	if chartRoute != "accounts" || strings.TrimSpace(r.URL.Query().Get("account")) == "" {
		return nil
	}
	chart := report.PresentChart(report.AccountAverageCostChart(current.Evaluation(), r.URL.Query().Get("period"), r.URL.Query().Get("account")))
	return &chart
}
