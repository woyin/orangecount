// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/quickentry"
	"orangecount/internal/snapshot"
	"orangecount/internal/web/favaadapter"
)

// quickBatchRecord remembers the last published quick-entry batch so the
// session can undo it while the snapshot is still current. It is intentionally
// minimal: one batch back, not a history system.
type quickBatchRecord struct {
	SnapshotID string
	TargetFile string
	EntryCount int
	// Serialized is the exact text appended to the file, so undo can remove
	// it by string replacement without re-parsing user intent.
	Serialized string
}

// quickLineResponse is the per-line JSON shape returned by preview.
type quickLineResponse struct {
	Line      int                   `json:"line"`
	Source    string                `json:"source"`
	Preview   string                `json:"preview,omitempty"`
	Duplicate bool                  `json:"duplicate"`
	Errors    []quickLineError      `json:"errors,omitempty"`
	Entry     *favaadapter.NewEntry `json:"entry,omitempty"`
}

type quickLineError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type quickPreviewResponse struct {
	Token      string                `json:"token"`
	Lines      []quickLineResponse   `json:"lines"`
	Target     string                `json:"target"`
	Problems   []quickProfileProblem `json:"problems,omitempty"`
	SnapshotID string                `json:"snapshot_id"`
	HasErrors  bool                  `json:"has_errors"`
}

type quickProfileProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Source  string `json:"source,omitempty"`
}

type quickCommitResponse struct {
	Published   bool                 `json:"published"`
	SnapshotID  string               `json:"snapshot_id,omitempty"`
	Backup      string               `json:"backup,omitempty"`
	Diagnostics []diagnosticResponse `json:"diagnostics,omitempty"`
	Error       string               `json:"error,omitempty"`
}

type quickUndoResponse struct {
	Undone      bool                 `json:"undone"`
	SnapshotID  string               `json:"snapshot_id,omitempty"`
	Diagnostics []diagnosticResponse `json:"diagnostics,omitempty"`
	Error       string               `json:"error,omitempty"`
}

// handleQuickPreview (POST) compiles a quick-entry line against the current
// snapshot and returns the would-be entries without writing anything.
func (s *Server) handleQuickPreview(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request struct {
		Text   string `json:"text"`
		Date   string `json:"date"`
		Flag   string `json:"flag"`
		Target string `json:"target"`
	}
	if err := decodeJSONBody(w, r, &request, 1<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	// Resolve target file: must already be in the include graph.
	target := strings.TrimSpace(request.Target)
	if target == "" {
		target = graph.DisplayPath(graph.Entry)
	}
	file, display, ok := graphFile(graph, target)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, fmt.Sprintf("target file %q is not in the ledger include graph", target))
		return
	}
	_ = file
	evaluation := current.Evaluation()
	evaluationPtr := &evaluation
	operatingCurrency := firstOperatingCurrency(evaluation.Options)
	results := quickentry.Compile(quickentry.CompileRequest{
		Text:              request.Text,
		Date:              request.Date,
		Flag:              request.Flag,
		OperatingCurrency: operatingCurrency,
		Evaluation:        evaluationPtr,
	})
	quickentry.DetectDuplicates(results, evaluationPtr)
	// Build per-line response and collect entries.
	var lines []quickLineResponse
	var entries []favaadapter.NewEntry
	hasErrors := false
	for _, res := range results {
		line := quickLineResponse{
			Line: res.Line, Source: res.Source, Preview: res.Preview, Duplicate: res.Duplicate,
		}
		for _, e := range res.Errors {
			line.Errors = append(line.Errors, quickLineError{Code: e.Code, Message: e.Message})
		}
		if len(res.Errors) > 0 {
			hasErrors = true
		} else if res.Entry != nil {
			entries = append(entries, *res.Entry)
			line.Entry = res.Entry
		}
		lines = append(lines, line)
	}
	if len(entries) == 0 || hasErrors {
		writeJSON(w, quickPreviewResponse{Lines: lines, Target: display, SnapshotID: current.ID, HasErrors: true})
		return
	}
	// Check for historical date warning: if the entry date predates the
	// target file's latest transaction date, it's informational only.
	token := s.quickPreviews.Store(entries, display)
	writeJSON(w, quickPreviewResponse{
		Token: token, Lines: lines, Target: display, SnapshotID: current.ID, HasErrors: hasErrors,
	})
}

// handleQuickCommit (POST) compiles and appends a quick entry, rebuilding the
// snapshot; on validation failure nothing is written and the error context
// comes back for the form.
func (s *Server) handleQuickCommit(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request struct {
		Token            string `json:"token"`
		ExpectedSnapshot string `json:"expected_snapshot_id"`
	}
	if err := decodeJSONBody(w, r, &request, 1<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	preview, ok := s.quickPreviews.Take(request.Token)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "preview not found, expired, or already committed")
		return
	}
	if request.ExpectedSnapshot != "" && request.ExpectedSnapshot != current.ID {
		writeAPIError(w, http.StatusConflict, "ledger changed since preview; reload and re-preview")
		return
	}
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	file, display, ok := graphFile(graph, preview.Target)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "target file is no longer in the ledger include graph")
		return
	}
	serialized, err := favaadapter.SerializeNewEntries(preview.Entries)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	content := strings.TrimRight(string(file.Data), "\n") + "\n\n" + serialized + "\n"
	result, backup, err := s.replaceGraphFile(current, file.Path, display, []byte(content))
	if err != nil {
		status := http.StatusUnprocessableEntity
		if result.Err != nil {
			status = http.StatusInternalServerError
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		writeJSON(w, quickCommitResponse{
			Backup: backup, Diagnostics: diagnosticsPayload(result.Diagnostics, current.Graph()),
		})
		return
	}
	// Record the batch as undoable while its resulting snapshot is current.
	s.quickLastBatch = &quickBatchRecord{
		SnapshotID: result.Snapshot.ID,
		TargetFile: file.Path,
		EntryCount: len(preview.Entries),
		Serialized: serialized,
	}
	writeJSON(w, quickCommitResponse{
		Published: true, SnapshotID: result.Snapshot.ID, Backup: backup,
	})
}

// handleQuickUndo (POST) reverts the last quick-entry batch by restoring the
// files it touched; a post-undo validation failure rolls the restore back.
func (s *Server) handleQuickUndo(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	s.writeMu.Lock()
	batch := s.quickLastBatch
	s.writeMu.Unlock()
	if batch == nil {
		writeAPIError(w, http.StatusNotFound, "no quick-entry batch to undo")
		return
	}
	if current.ID != batch.SnapshotID {
		// Snapshot has moved on; undo is disabled to avoid clobbering later edits.
		writeAPIError(w, http.StatusConflict, "ledger changed since the batch was published; correct manually in the editor")
		return
	}
	old, err := os.ReadFile(batch.TargetFile)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, fmt.Sprintf("cannot read target file: %v", err))
		return
	}
	// Remove the serialized block by exact string match. The append format is
	// "\n\n" + serialized + "\n", so we remove that exact suffix.
	removed := strings.TrimRight(string(old), "\n")
	suffix := "\n\n" + batch.Serialized
	if !strings.HasSuffix(removed, suffix) {
		writeAPIError(w, http.StatusConflict, "target file no longer ends with the quick-entry batch; correct manually")
		return
	}
	restored := strings.TrimSuffix(removed, suffix) + "\n"
	info, err := os.Stat(batch.TargetFile)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}
	backup := batch.TargetFile + ".orangecount.bak"
	if err := os.WriteFile(backup, old, info.Mode().Perm()); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := atomicWrite(batch.TargetFile, []byte(restored), info.Mode().Perm()); err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := s.store.Reload(current.EntryPath, snapshot.BuildOptions{})
	if result.Snapshot == nil {
		// Restore the previous content on validation failure.
		_ = atomicWrite(batch.TargetFile, old, info.Mode().Perm())
		_ = s.store.Reload(current.EntryPath, snapshot.BuildOptions{})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnprocessableEntity)
		writeJSON(w, quickUndoResponse{
			Diagnostics: diagnosticsPayload(result.Diagnostics, current.Graph()),
			Error:       "validation failed after undo; file restored",
		})
		return
	}
	s.quickLastBatch = nil
	writeJSON(w, quickUndoResponse{Undone: true, SnapshotID: result.Snapshot.ID})
}

// quickProfileRule is the JSON shape for one effective rule in the profile listing.
type quickProfileRule struct {
	Type      string   `json:"type"`
	Name      string   `json:"name"`
	Account   string   `json:"account,omitempty"`
	Source    string   `json:"source,omitempty"`
	Dest      string   `json:"destination,omitempty"`
	Currency  string   `json:"currency,omitempty"`
	Payee     string   `json:"payee,omitempty"`
	Narration string   `json:"narration,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Links     []string `json:"links,omitempty"`
}

func (s *Server) handleQuickProfile(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	// GET: list effective rules as of today.
	today := ledger.Date{}
	if t := current.BuiltAt; !t.IsZero() {
		today = ledger.Date{Year: t.Year(), Month: int(t.Month()), Day: t.Day(), Raw: t.Format("2006-01-02")}
	}
	evalCopy := current.Evaluation()
	profile := quickentry.EffectiveProfile(&evalCopy, today)
	var rules []quickProfileRule
	for _, a := range profile.Accounts {
		rules = append(rules, quickProfileRule{Type: "account", Name: a.Alias, Account: a.Account})
	}
	for _, t := range profile.Templates {
		rule := quickProfileRule{Type: "template", Name: t.Name, Source: t.Source, Dest: t.Destination, Currency: t.Currency, Payee: t.Payee, Narration: t.Narration, Tags: t.Tags, Links: t.Links}
		rules = append(rules, rule)
	}
	var problems []quickProfileProblem
	for _, p := range profile.Problems {
		problems = append(problems, quickProfileProblem{Code: p.Code, Message: p.Message, Source: p.Source})
	}
	writeJSON(w, map[string]any{"rules": rules, "problems": problems})
}

// handleQuickProfileSave creates or supersedes a quick-entry-profile directive
// through the reviewed write path. It emits a dated custom directive and
// appends it to the entry file, the same way Add Entry publishes entries.
func (s *Server) handleQuickProfileSave(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request struct {
		Date      string `json:"date"`
		RuleType  string `json:"type"`
		Name      string `json:"name"`
		Account   string `json:"account"`
		Source    string `json:"source"`
		Dest      string `json:"destination"`
		Currency  string `json:"currency"`
		Payee     string `json:"payee"`
		Narration string `json:"narration"`
	}
	if err := decodeJSONBody(w, r, &request, 1<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	serialized, err := serializeQuickProfileDirective(request.Date, request.RuleType, request.Name, request.Account, request.Source, request.Dest, request.Currency, request.Payee, request.Narration)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	file, display, ok := graphFile(graph, graph.DisplayPath(graph.Entry))
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "entry file unavailable")
		return
	}
	content := strings.TrimRight(string(file.Data), "\n") + "\n\n" + serialized + "\n"
	result, backup, err := s.replaceGraphFile(current, file.Path, display, []byte(content))
	if err != nil {
		status := http.StatusUnprocessableEntity
		if result.Err != nil {
			status = http.StatusInternalServerError
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		writeJSON(w, quickCommitResponse{Backup: backup, Diagnostics: diagnosticsPayload(result.Diagnostics, current.Graph())})
		return
	}
	writeJSON(w, quickCommitResponse{Published: true, SnapshotID: result.Snapshot.ID, Backup: backup})
}

// serializeQuickProfileDirective renders one quick-profile rule as a ledger
// directive line, quoting free-text fields and validating the date format.
func serializeQuickProfileDirective(date, ruleType, name, account, source, dest, currency, payee, narration string) (string, error) {
	if !quickProfileDateRegex.MatchString(date) {
		return "", fmt.Errorf("invalid date %q", date)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	switch ruleType {
	case "account":
		account = strings.TrimSpace(account)
		if account == "" {
			return "", fmt.Errorf("account is required for an account rule")
		}
		if !quickProfileAccountRegex.MatchString(account) {
			return "", fmt.Errorf("invalid account %q", account)
		}
		return fmt.Sprintf("%s custom \"orangecount.quick-account.v1\" \"%s\" %s", date, escapeBeancountString(name), account), nil
	case "template":
		var lines []string
		head := fmt.Sprintf("%s custom \"orangecount.quick-template.v1\" \"%s\"", date, escapeBeancountString(name))
		lines = append(lines, head)
		if source = strings.TrimSpace(source); source != "" {
			lines = append(lines, fmt.Sprintf("  source: \"%s\"", escapeBeancountString(source)))
		}
		if dest = strings.TrimSpace(dest); dest != "" {
			lines = append(lines, fmt.Sprintf("  destination: \"%s\"", escapeBeancountString(dest)))
		}
		if currency = strings.TrimSpace(currency); currency != "" {
			lines = append(lines, fmt.Sprintf("  currency: \"%s\"", escapeBeancountString(currency)))
		}
		if payee = strings.TrimSpace(payee); payee != "" {
			lines = append(lines, fmt.Sprintf("  payee: \"%s\"", escapeBeancountString(payee)))
		}
		if narration = strings.TrimSpace(narration); narration != "" {
			lines = append(lines, fmt.Sprintf("  narration: \"%s\"", escapeBeancountString(narration)))
		}
		return strings.Join(lines, "\n"), nil
	default:
		return "", fmt.Errorf("unknown rule type %q (expected account or template)", ruleType)
	}
}

func escapeBeancountString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

func firstOperatingCurrency(options map[string]string) string {
	for _, c := range strings.Split(options["operating_currency"], ",") {
		if c = strings.TrimSpace(c); c != "" {
			return c
		}
	}
	return ""
}

var (
	quickProfileDateRegex    = regexp.MustCompile(`\A\d{4}-\d{2}-\d{2}\z`)
	quickProfileAccountRegex = regexp.MustCompile(`\A[A-Z][A-Za-z0-9\-]*(?::[A-Z][A-Za-z0-9\-]*)+\z`)
)
