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

// handlePlanningGeneratePreview converts form values into the frozen source
// schema. The generated directive then takes the same single-use reviewed
// write path as every other planning change.
func (s *Server) handlePlanningGeneratePreview(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request struct {
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
	var content string
	var err error
	replaceStart, replaceEnd := -1, -1
	replaceTarget := ""
	switch request.Kind {
	case "profile":
		if profile.ID != "" {
			for _, entry := range evaluation.Entries {
				if custom, ok := entry.Directive.(ledger.Custom); ok && custom.Type == "orangecount.planning-profile.v1" {
					replaceStart, replaceEnd = custom.Span().Start, custom.Span().End
					replaceTarget = current.Graph().DisplayPath(custom.Span().File)
					break
				}
			}
		}
		reserve, reserveErr := ledger.ParseDecimal(request.Profile.MinimumReserve)
		if reserveErr != nil {
			writeAPIError(w, http.StatusBadRequest, "minimum reserve must be a valid amount")
			return
		}
		if request.Profile.RecordedThrough == "" {
			writeAPIError(w, http.StatusBadRequest, "recorded-through is required")
			return
		}
		recorded, dateErr := planningHorizon(request.Profile.RecordedThrough, ledger.Date{Year: 1, Month: 1, Day: 1, Raw: "0001-01-01"})
		if dateErr != nil {
			writeAPIError(w, http.StatusBadRequest, "recorded-through: "+dateErr.Error())
			return
		}
		content, err = planning.SerializeProfileDefinition(planning.ProfileDefinition{Date: today, ID: request.Profile.ID, Currency: request.Profile.Currency, Timezone: request.Profile.Timezone, MinimumReserve: reserve, SpendableAccounts: trimPlanningValues(request.Profile.SpendableAccounts), ShortTermDebtAccounts: trimPlanningValues(request.Profile.ShortTermDebtAccounts), RecordedThrough: recorded})
	case "plan":
		if profile.ID == "" || len(profile.Problems) > 0 {
			writeAPIError(w, http.StatusUnprocessableEntity, "create a valid planning profile before adding plans")
			return
		}
		amount, amountErr := ledger.ParseDecimal(request.Plan.Amount)
		if amountErr != nil {
			writeAPIError(w, http.StatusBadRequest, "plan amount must be valid")
			return
		}
		if request.Plan.Date == "" {
			writeAPIError(w, http.StatusBadRequest, "plan date is required")
			return
		}
		date, dateErr := planningHorizon(request.Plan.Date, ledger.Date{Year: 1, Month: 1, Day: 1, Raw: "0001-01-01"})
		if dateErr != nil {
			writeAPIError(w, http.StatusBadRequest, "plan date: "+dateErr.Error())
			return
		}
		revision := request.Plan.Revision
		if revision == 0 {
			revision = 1
		}
		content, err = planning.SerializePlanRevision(planning.PlanRevisionDefinition{Date: today, Revision: revision, Status: request.Plan.Status, Plan: planning.Plan{ID: request.Plan.ID, Name: request.Plan.Name, Date: date, Amount: amount, Currency: profile.Currency, Direction: planning.Direction(request.Plan.Direction), Commitment: planning.Commitment(request.Plan.Commitment), Account: request.Plan.Account}})
	default:
		writeAPIError(w, http.StatusBadRequest, "unknown planning form kind")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	target := strings.TrimSpace(request.Target)
	if replaceTarget != "" {
		target = replaceTarget
	}
	if target == "" {
		target = graph.DisplayPath(graph.Entry)
	}
	_, display, ok := graphFile(graph, target)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "target is not in the ledger include graph")
		return
	}
	token := s.planningPreviews.Store(planningPreview{Content: content, Target: display, SnapshotID: current.ID, ReplaceStart: replaceStart, ReplaceEnd: replaceEnd})
	writeJSON(w, map[string]any{"valid": true, "token": token, "target": display, "content": content, "snapshot_id": current.ID})
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
