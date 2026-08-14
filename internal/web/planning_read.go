// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package web

import (
	"fmt"
	"net/http"
	"time"

	"orangecount/internal/ledger"
	"orangecount/internal/planning"
	"orangecount/internal/snapshot"
	"orangecount/internal/web/favaadapter"
)

// handlePlanning exposes a deliberately small, stable view model rather than
// leaking Go's exported-field JSON names into the browser contract.
func (s *Server) handlePlanning(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	evaluation := current.Evaluation()
	profile := planning.EffectiveProfile(&evaluation)
	response := planningResponse{SnapshotID: current.ID, Profile: planningProfileResponse(profile)}
	if len(profile.Problems) > 0 {
		response.ReadinessError = "Planning setup needs attention"
		writeJSON(w, favaadapter.NewEnvelope(response, current.BuiltAt))
		return
	}
	today := planningToday(profile.Timezone)
	end, err := planningHorizon(r.URL.Query().Get("horizon_end"), today)
	if err != nil {
		response.ReadinessError = err.Error()
		writeJSON(w, favaadapter.NewEnvelope(response, current.BuiltAt))
		return
	}
	input, err := planning.BuildInput(&evaluation, profile, today, end)
	if err != nil {
		response.ReadinessError = err.Error()
		writeJSON(w, favaadapter.NewEnvelope(response, current.BuiltAt))
		return
	}
	result, err := planning.Calculate(input)
	if err != nil {
		response.ReadinessError = err.Error()
		writeJSON(w, favaadapter.NewEnvelope(response, current.BuiltAt))
		return
	}
	response.Result = planningResultResponse(result)
	writeJSON(w, favaadapter.NewEnvelope(response, current.BuiltAt))
}

// handlePlanningScenario evaluates an unpersisted proposed purchase.  It is a
// POST because it carries an owner-entered amount, but it only reads the
// published snapshot and never writes a ledger directive.
func (s *Server) handlePlanningScenario(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request struct {
		HorizonEnd           string `json:"horizon_end"`
		Name                 string `json:"name"`
		Amount               string `json:"amount"`
		Date                 string `json:"date"`
		IncludeFutureInflows bool   `json:"include_future_inflows"`
		ReleaseAdjustable    bool   `json:"release_adjustable"`
	}
	if err := decodeJSONBody(w, r, &request, 64<<10); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	evaluation := current.Evaluation()
	profile := planning.EffectiveProfile(&evaluation)
	if len(profile.Problems) > 0 {
		writeAPIError(w, http.StatusUnprocessableEntity, "planning setup needs attention")
		return
	}
	today := planningToday(profile.Timezone)
	end, err := planningHorizon(request.HorizonEnd, today)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	draftDate, err := planningHorizon(request.Date, today)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "purchase date: "+err.Error())
		return
	}
	if draftDate.String() > end.String() {
		writeAPIError(w, http.StatusBadRequest, "purchase date must be within the selected planning horizon")
		return
	}
	amount, err := ledger.ParseDecimal(request.Amount)
	if err != nil || amount.Sign() <= 0 {
		writeAPIError(w, http.StatusBadRequest, "purchase amount must be positive")
		return
	}
	input, err := planning.BuildInput(&evaluation, profile, today, end)
	if err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if request.Name == "" {
		request.Name = "Proposed purchase"
	}
	draft := planning.Plan{ID: "scenario-purchase", Name: request.Name, Date: draftDate, Amount: amount, Currency: profile.Currency, Direction: planning.Outflow, Commitment: planning.Committed}
	result, err := planning.EvaluateScenario(input, planning.ScenarioOptions{IncludeFutureInflows: request.IncludeFutureInflows, ReleaseAdjustable: request.ReleaseAdjustable, Draft: &draft})
	if err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, struct {
		SnapshotID            string              `json:"snapshot_id"`
		Result                *planningResultView `json:"result"`
		IncludesFutureInflows bool                `json:"includes_future_inflows"`
		ReleasesAdjustable    bool                `json:"releases_adjustable"`
		DraftIsPersisted      bool                `json:"draft_is_persisted"`
	}{SnapshotID: current.ID, Result: planningResultResponse(result.Result), IncludesFutureInflows: result.IncludesFutureInflows, ReleasesAdjustable: result.ReleasesAdjustable, DraftIsPersisted: false})
}

type planningResponse struct {
	SnapshotID     string              `json:"snapshot_id"`
	Profile        planningProfileView `json:"profile"`
	ReadinessError string              `json:"readiness_error,omitempty"`
	Result         *planningResultView `json:"result,omitempty"`
}

type planningProfileView struct {
	Currency        string                `json:"currency"`
	Timezone        string                `json:"timezone"`
	RecordedThrough string                `json:"recorded_through"`
	Problems        []planningProblemView `json:"problems"`
	Plans           []planningPlanView    `json:"plans"`
}

type planningPlanView struct {
	ID         string `json:"id"`
	Revision   int    `json:"revision"`
	Name       string `json:"name"`
	Date       string `json:"date"`
	Amount     string `json:"amount"`
	Direction  string `json:"direction"`
	Commitment string `json:"commitment"`
	Account    string `json:"account"`
}

type planningProblemView struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type planningResultView struct {
	Currency           string              `json:"currency"`
	CurrentFunds       string              `json:"current_funds"`
	CurrentDebt        string              `json:"current_short_term_debt"`
	MinimumReserve     string              `json:"minimum_reserve"`
	CurrentPlanning    bool                `json:"current_planning"`
	PrimarySafeToSpend string              `json:"primary_safe_to_spend"`
	FundingShortfall   string              `json:"funding_shortfall"`
	LiquidityLowPoint  planningPointView   `json:"liquidity_low_point"`
	Timeline           []planningPointView `json:"timeline"`
}

type planningPointView struct {
	Date              string `json:"date"`
	CommittedOutflow  string `json:"committed_outflow"`
	AdjustableOutflow string `json:"adjustable_outflow"`
	IgnoredInflows    string `json:"ignored_inflows"`
	Headroom          string `json:"headroom"`
}

func planningProfileResponse(profile planning.Profile) planningProfileView {
	view := planningProfileView{Currency: profile.Currency, Timezone: profile.Timezone, RecordedThrough: profile.RecordedThrough.String()}
	for _, problem := range profile.Problems {
		view.Problems = append(view.Problems, planningProblemView{Code: problem.Code, Message: problem.Message})
	}
	for _, plan := range profile.Plans {
		view.Plans = append(view.Plans, planningPlanView{ID: plan.ID, Revision: plan.Revision, Name: plan.Name, Date: plan.Date.String(), Amount: plan.Amount.String(), Direction: string(plan.Direction), Commitment: string(plan.Commitment), Account: plan.Account})
	}
	return view
}

func planningResultResponse(result planning.Result) *planningResultView {
	view := &planningResultView{Currency: result.Currency, CurrentFunds: result.CurrentFunds.String(), CurrentDebt: result.CurrentShortTermDebt.String(), MinimumReserve: result.MinimumReserve.String(), CurrentPlanning: result.CurrentPlanning, PrimarySafeToSpend: result.PrimarySafeToSpend.String(), FundingShortfall: result.FundingShortfall.String(), LiquidityLowPoint: planningPointResponse(result.LiquidityLowPoint)}
	for _, point := range result.Timeline {
		view.Timeline = append(view.Timeline, planningPointResponse(point))
	}
	return view
}

func planningPointResponse(point planning.Point) planningPointView {
	return planningPointView{Date: point.Date.String(), CommittedOutflow: point.CommittedOutflow.String(), AdjustableOutflow: point.AdjustableOutflow.String(), IgnoredInflows: point.IgnoredInflows.String(), Headroom: point.Headroom.String()}
}

func planningToday(timezone string) ledger.Date {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.Local
	}
	now := time.Now().In(location)
	return ledger.Date{Year: now.Year(), Month: int(now.Month()), Day: now.Day(), Raw: now.Format("2006-01-02")}
}

func planningHorizon(raw string, today ledger.Date) (ledger.Date, error) {
	if raw == "" {
		return today, nil
	}
	// Dates have a compact parser in the ledger package only through directives;
	// this endpoint accepts ISO dates and validates their components locally.
	if len(raw) != 10 || raw[4] != '-' || raw[7] != '-' {
		return ledger.Date{}, fmt.Errorf("planning horizon must be YYYY-MM-DD")
	}
	date := ledger.Date{Year: atoiPlanning(raw[0:4]), Month: atoiPlanning(raw[5:7]), Day: atoiPlanning(raw[8:10]), Raw: raw}
	if !date.Valid() || date.String() != raw {
		return ledger.Date{}, fmt.Errorf("planning horizon must be a valid date")
	}
	if date.Year < today.Year || (date.Year == today.Year && (date.Month < today.Month || (date.Month == today.Month && date.Day < today.Day))) {
		return ledger.Date{}, fmt.Errorf("planning horizon ends before today")
	}
	return date, nil
}

func atoiPlanning(raw string) int {
	value := 0
	for _, r := range raw {
		if r < '0' || r > '9' {
			return 0
		}
		value = value*10 + int(r-'0')
	}
	return value
}
