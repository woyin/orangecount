// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package web

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/planning"
	"orangecount/internal/snapshot"
	"orangecount/internal/web/favaadapter"
)

// handlePlanningReview returns evidence, not conclusions: the owner still has
// to confirm a recurring candidate or a plan-fulfilment match.
func (s *Server) handlePlanningReview(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	evaluation := current.Evaluation()
	profile := planning.EffectiveProfile(&evaluation)
	occurrences := planningOccurrences(evaluation, profile.Currency)
	patterns := planning.DetectPatterns(occurrences, 3)
	matches := planning.SuggestMatches(profile.Plans, occurrences, 7)
	fingerprint := planningReviewFingerprint(evaluation)
	completed, stale := planningCompletedReview(evaluation, fingerprint)
	writeJSON(w, favaadapter.NewEnvelope(struct {
		SnapshotID  string                   `json:"snapshot_id"`
		Fingerprint string                   `json:"fingerprint"`
		Completed   bool                     `json:"completed"`
		Stale       bool                     `json:"stale"`
		Patterns    []planningPatternView    `json:"patterns"`
		Matches     []planningMatchView      `json:"matches"`
		Occurrences []planningOccurrenceView `json:"occurrences"`
	}{SnapshotID: current.ID, Fingerprint: fingerprint, Completed: completed, Stale: stale, Patterns: planningPatternsResponse(patterns), Matches: planningMatchesResponse(matches), Occurrences: planningOccurrencesResponse(occurrences)}, current.BuiltAt))
}

func (s *Server) handlePlanningReviewPreview(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request struct {
		ExpectedSnapshot string `json:"expected_snapshot_id"`
		Matches          []struct {
			PlanID       string `json:"plan_id"`
			OccurrenceID string `json:"occurrence_id"`
		} `json:"matches"`
	}
	if err := decodeJSONBody(w, r, &request, 64<<10); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if request.ExpectedSnapshot != "" && request.ExpectedSnapshot != current.ID {
		writeAPIError(w, http.StatusConflict, "ledger changed; reload before completing review")
		return
	}
	evaluation := current.Evaluation()
	profile := planning.EffectiveProfile(&evaluation)
	occurrences := planningOccurrences(evaluation, profile.Currency)
	candidates := planning.SuggestMatches(profile.Plans, occurrences, 7)
	allowed := map[string]bool{}
	for _, match := range candidates {
		allowed[match.PlanID+"\x00"+match.OccurrenceID] = true
	}
	lines := []string{fmt.Sprintf("%s custom %q %q", planningToday(profile.Timezone).String(), "orangecount.cycle-review.v1", "completed"), fmt.Sprintf("  ledger_fingerprint: %q", planningReviewFingerprint(evaluation))}
	for _, match := range request.Matches {
		if !allowed[match.PlanID+"\x00"+match.OccurrenceID] {
			writeAPIError(w, http.StatusBadRequest, "review contains a match that is no longer suggested")
			return
		}
		lines = append(lines, fmt.Sprintf("  match: %q", match.PlanID+":"+match.OccurrenceID))
	}
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	target := graph.DisplayPath(graph.Entry)
	token := s.planningPreviews.Store(planningPreview{Content: strings.Join(lines, "\n"), Target: target, SnapshotID: current.ID})
	writeJSON(w, map[string]any{"valid": true, "token": token, "target": target, "content": strings.Join(lines, "\n"), "snapshot_id": current.ID})
}

func planningReviewFingerprint(evaluation ledger.Evaluation) string {
	hash := sha256.New()
	for _, entry := range evaluation.Entries {
		if custom, ok := entry.Directive.(ledger.Custom); ok && custom.Type == "orangecount.cycle-review.v1" {
			continue
		}
		_, _ = hash.Write([]byte(entry.Directive.RawText()))
		_, _ = hash.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
func planningCompletedReview(evaluation ledger.Evaluation, fingerprint string) (bool, bool) {
	for index := len(evaluation.Entries) - 1; index >= 0; index-- {
		custom, ok := evaluation.Entries[index].Directive.(ledger.Custom)
		if !ok || custom.Type != "orangecount.cycle-review.v1" {
			continue
		}
		for _, meta := range custom.Meta {
			if meta.Key == "ledger_fingerprint" && meta.Value.Kind == ledger.ValueString {
				return true, meta.Value.String != fingerprint
			}
		}
		return true, true
	}
	return false, false
}

type planningOccurrenceView struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Account string `json:"account"`
	Name    string `json:"name"`
	Amount  string `json:"amount"`
}
type planningPatternView struct {
	Key           string                   `json:"key"`
	Name          string                   `json:"name"`
	Account       string                   `json:"account"`
	MinimumAmount string                   `json:"minimum_amount"`
	MaximumAmount string                   `json:"maximum_amount"`
	CadenceDays   int                      `json:"cadence_days"`
	Occurrences   []planningOccurrenceView `json:"occurrences"`
}
type planningMatchView struct {
	PlanID       string `json:"plan_id"`
	OccurrenceID string `json:"occurrence_id"`
	SameName     bool   `json:"same_name"`
	SameAccount  bool   `json:"same_account"`
	SameAmount   bool   `json:"same_amount"`
	DateDistance int    `json:"date_distance"`
}

func planningOccurrences(evaluation ledger.Evaluation, currency string) []planning.Occurrence {
	var occurrences []planning.Occurrence
	for entryIndex, entry := range evaluation.Entries {
		transaction, ok := entry.Directive.(ledger.Transaction)
		if !ok {
			continue
		}
		name := strings.TrimSpace(strings.TrimSpace(transaction.Payee) + " " + strings.TrimSpace(transaction.Narration))
		for postingIndex, posting := range transaction.Postings {
			if posting.Units == nil || posting.Units.Currency != currency || posting.Units.Number.Rat == nil || posting.Units.Number.Rat.Sign() <= 0 || !strings.HasPrefix(posting.Account, "Expenses:") {
				continue
			}
			account := ""
			for _, counterpart := range transaction.Postings {
				if counterpart.Units != nil && counterpart.Units.Currency == currency && counterpart.Units.Number.Rat != nil && counterpart.Units.Number.Rat.Sign() < 0 && (strings.HasPrefix(counterpart.Account, "Assets:") || strings.HasPrefix(counterpart.Account, "Liabilities:")) {
					account = counterpart.Account
					break
				}
			}
			if account == "" {
				continue
			}
			occurrences = append(occurrences, planning.Occurrence{ID: fmt.Sprintf("%d-%d", entryIndex, postingIndex), Date: transaction.Date, Account: account, Name: name, Amount: ledger.DecimalFromNumber(posting.Units.Number)})
		}
	}
	return occurrences
}

func planningOccurrencesResponse(values []planning.Occurrence) []planningOccurrenceView {
	result := make([]planningOccurrenceView, 0, len(values))
	for _, value := range values {
		result = append(result, planningOccurrenceView{value.ID, value.Date.String(), value.Account, value.Name, value.Amount.String()})
	}
	return result
}
func planningPatternsResponse(values []planning.CashFlowPattern) []planningPatternView {
	result := make([]planningPatternView, 0, len(values))
	for _, value := range values {
		result = append(result, planningPatternView{Key: value.Key, Name: value.Name, Account: value.Account, MinimumAmount: value.MinimumAmount.String(), MaximumAmount: value.MaximumAmount.String(), CadenceDays: value.CadenceDays, Occurrences: planningOccurrencesResponse(value.Occurrences)})
	}
	return result
}
func planningMatchesResponse(values []planning.MatchSuggestion) []planningMatchView {
	result := make([]planningMatchView, 0, len(values))
	for _, value := range values {
		result = append(result, planningMatchView{PlanID: value.PlanID, OccurrenceID: value.OccurrenceID, SameName: value.SameName, SameAccount: value.SameAccount, SameAmount: value.SameAmount, DateDistance: value.DateDistance})
	}
	return result
}
