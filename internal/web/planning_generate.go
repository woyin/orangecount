// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package web

import (
	"net/http"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/planning"
	"orangecount/internal/snapshot"
)

// planningGenerateRequest is the JSON body of the form-to-directive preview.
type planningGenerateRequest struct {
	Kind             string `json:"kind"`
	Target           string `json:"target"`
	ExpectedSnapshot string `json:"expected_snapshot_id"`
	Profile          struct {
		ID                    string   `json:"id"`
		Currency              string   `json:"currency"`
		Timezone              string   `json:"timezone"`
		MinimumReserve        string   `json:"minimum_reserve"`
		RecordedThrough       string   `json:"recorded_through"`
		SpendableAccounts     []string `json:"spendable_accounts"`
		ShortTermDebtAccounts []string `json:"short_term_debt_accounts"`
	} `json:"profile"`
	Plan struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Date       string `json:"date"`
		Amount     string `json:"amount"`
		Direction  string `json:"direction"`
		Commitment string `json:"commitment"`
		Account    string `json:"account"`
		Revision   int    `json:"revision"`
		Status     string `json:"status"`
	} `json:"plan"`
}

// planningGenerated is one previewed planning directive: its frozen source
// text plus the source span it replaces in place (negative when appending).
type planningGenerated struct {
	content       string
	replaceStart  int
	replaceEnd    int
	replaceTarget string
}

// planningFormError is a request-level rejection with its HTTP status.
type planningFormError struct {
	status  int
	message string
}

// handlePlanningGeneratePreview converts form values into the frozen source
// schema. The generated directive then takes the same single-use reviewed
// write path as every other planning change.
func (s *Server) handlePlanningGeneratePreview(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request planningGenerateRequest
	if err := decodeJSONBody(w, r, &request, 64<<10); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if request.ExpectedSnapshot != "" && request.ExpectedSnapshot != current.ID {
		writeAPIError(w, http.StatusConflict, "ledger changed; reload before previewing")
		return
	}
	evaluation := current.Evaluation()
	profile := planning.EffectiveProfile(&evaluation)
	today := planningToday(profile.Timezone)
	var generated planningGenerated
	var err *planningFormError
	switch request.Kind {
	case "profile":
		generated, err = s.generateProfileForm(request, evaluation, profile, current, today)
	case "plan":
		generated, err = s.generatePlanForm(request, evaluation, profile, today)
	default:
		err = &planningFormError{status: http.StatusBadRequest, message: "unknown planning form kind"}
	}
	if err != nil {
		writeAPIError(w, err.status, err.message)
		return
	}
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	target := strings.TrimSpace(request.Target)
	if generated.replaceTarget != "" {
		target = generated.replaceTarget
	}
	if target == "" {
		target = graph.DisplayPath(graph.Entry)
	}
	_, display, ok := graphFile(graph, target)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "target is not in the ledger include graph")
		return
	}
	token := s.planningPreviews.Store(planningPreview{Content: generated.content, Target: display, SnapshotID: current.ID, ReplaceStart: generated.replaceStart, ReplaceEnd: generated.replaceEnd})
	writeJSON(w, map[string]any{"valid": true, "token": token, "target": display, "content": generated.content, "snapshot_id": current.ID})
}

// generateProfileForm serializes a profile upsert. An existing profile's
// directive is replaced in place; the first definition wins as the replace
// target because a valid ledger never holds two.
func (s *Server) generateProfileForm(request planningGenerateRequest, evaluation ledger.Evaluation, profile planning.Profile, current *snapshot.Snapshot, today ledger.Date) (planningGenerated, *planningFormError) {
	generated := planningGenerated{replaceStart: -1, replaceEnd: -1}
	if profile.ID != "" {
		for _, entry := range evaluation.Entries {
			custom, ok := entry.Directive.(ledger.Custom)
			if !ok || custom.Type != "orangecount.planning-profile.v1" {
				continue
			}
			generated.replaceStart, generated.replaceEnd = custom.Span().Start, custom.Span().End
			generated.replaceTarget = current.Graph().DisplayPath(custom.Span().File)
			break
		}
	}
	reserve, reserveErr := ledger.ParseDecimal(request.Profile.MinimumReserve)
	if reserveErr != nil {
		return generated, &planningFormError{http.StatusBadRequest, "minimum reserve must be a valid amount"}
	}
	if request.Profile.RecordedThrough == "" {
		return generated, &planningFormError{http.StatusBadRequest, "recorded-through is required"}
	}
	recorded, dateErr := planningHorizon(request.Profile.RecordedThrough, ledger.Date{Year: 1, Month: 1, Day: 1, Raw: "0001-01-01"})
	if dateErr != nil {
		return generated, &planningFormError{http.StatusBadRequest, "recorded-through: " + dateErr.Error()}
	}
	content, err := planning.SerializeProfileDefinition(planning.ProfileDefinition{
		Date: today, ID: request.Profile.ID, Currency: request.Profile.Currency, Timezone: request.Profile.Timezone,
		MinimumReserve:    reserve,
		SpendableAccounts: trimPlanningValues(request.Profile.SpendableAccounts), ShortTermDebtAccounts: trimPlanningValues(request.Profile.ShortTermDebtAccounts),
		RecordedThrough: recorded,
	})
	if err != nil {
		return generated, &planningFormError{http.StatusBadRequest, err.Error()}
	}
	generated.content = content
	return generated, nil
}

// generatePlanForm serializes one plan revision; it requires an existing,
// problem-free profile because every plan inherits the profile's currency.
func (s *Server) generatePlanForm(request planningGenerateRequest, evaluation ledger.Evaluation, profile planning.Profile, today ledger.Date) (planningGenerated, *planningFormError) {
	generated := planningGenerated{replaceStart: -1, replaceEnd: -1}
	if profile.ID == "" || len(profile.Problems) > 0 {
		return generated, &planningFormError{http.StatusUnprocessableEntity, "create a valid planning profile before adding plans"}
	}
	amount, amountErr := ledger.ParseDecimal(request.Plan.Amount)
	if amountErr != nil {
		return generated, &planningFormError{http.StatusBadRequest, "plan amount must be valid"}
	}
	if request.Plan.Date == "" {
		return generated, &planningFormError{http.StatusBadRequest, "plan date is required"}
	}
	date, dateErr := planningHorizon(request.Plan.Date, ledger.Date{Year: 1, Month: 1, Day: 1, Raw: "0001-01-01"})
	if dateErr != nil {
		return generated, &planningFormError{http.StatusBadRequest, "plan date: " + dateErr.Error()}
	}
	revision := request.Plan.Revision
	if revision == 0 {
		revision = 1
	}
	content, err := planning.SerializePlanRevision(planning.PlanRevisionDefinition{
		Date: today, Revision: revision, Status: request.Plan.Status,
		Plan: planning.Plan{
			ID: request.Plan.ID, Name: request.Plan.Name, Date: date, Amount: amount, Currency: profile.Currency,
			Direction: planning.Direction(request.Plan.Direction), Commitment: planning.Commitment(request.Plan.Commitment), Account: request.Plan.Account,
		},
	})
	if err != nil {
		return generated, &planningFormError{http.StatusBadRequest, err.Error()}
	}
	generated.content = content
	return generated, nil
}

func trimPlanningValues(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}
