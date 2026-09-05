// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orangecount/internal/snapshot"
)

// planningTestLedger is a small but complete planning ledger: a valid profile,
// one active plan, and a weekly recurring 25 CNY expense paid from the
// spendable account so pattern detection and match suggestions both fire.
const planningTestLedger = `2000-01-01 open Assets:Bank CNY
2000-01-01 open Expenses:Living:吃的 CNY
2000-01-01 open Equity:Opening CNY
2000-01-02 * "opening"
  Assets:Bank 100 CNY
  Equity:Opening -100 CNY
2000-01-02 custom "orangecount.planning-profile.v1" "primary"
  currency: CNY
  timezone: "Asia/Singapore"
  minimum_reserve: 10 CNY
  spendable_account: Assets:Bank
  recorded_through: 2000-01-02
2000-01-02 custom "orangecount.planned-flow.v1" "午餐计划"
  name: "午餐"
  expected_date: 2000-01-20
  amount: 25 CNY
  direction: outflow
  commitment: committed
  account: Assets:Bank
2000-01-05 * "超市" "午餐"
  Expenses:Living:吃的 25 CNY
  Assets:Bank -25 CNY
2000-01-12 * "超市" "午餐"
  Expenses:Living:吃的 25 CNY
  Assets:Bank -25 CNY
2000-01-19 * "超市" "午餐"
  Expenses:Living:吃的 25 CNY
  Assets:Bank -25 CNY
`

// newPlanningTestServer builds a server over a written ledger and returns the
// entry path plus a handle on the file's initial content.
func newPlanningTestServer(t *testing.T, ledgerText string) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(entry, []byte(ledgerText), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("build diagnostics=%+v", built.Diagnostics)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	return server, entry
}

func planningJSON(t *testing.T, server *Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("%s %s: %d %s", method, path, recorder.Code, recorder.Body.String())
	}
	// Adapter handlers wrap their payload in {data, mtime}; plain handlers do
	// not. Unwrap so assertions read one shape.
	if data, ok := payload["data"].(map[string]any); ok {
		return recorder.Code, data
	}
	return recorder.Code, payload
}

func TestPlanningReadReturnsSafeToSpend(t *testing.T) {
	server, _ := newPlanningTestServer(t, planningTestLedger)
	code, payload := planningJSON(t, server, http.MethodGet, "/__orangecount/fava/planning", "")
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%v", code, payload)
	}
	if payload["readiness_error"] != nil {
		t.Fatalf("unexpected readiness error: %v", payload["readiness_error"])
	}
	result, ok := payload["result"].(map[string]any)
	if !ok || result["currency"] != "CNY" {
		t.Fatalf("result=%v", payload["result"])
	}
	// 100 opened minus 75 spent leaves 25; the 10 reserve gives base headroom
	// 15, and the committed 25 CNY plan on 01-20 drives the low point to -10,
	// so the conservative answer is zero safe-to-spend with a 10 shortfall.
	if result["current_funds"] != "25" {
		t.Fatalf("current funds=%v", result["current_funds"])
	}
	if result["primary_safe_to_spend"] != "0" || result["funding_shortfall"] != "10" {
		t.Fatalf("safe-to-spend=%v shortfall=%v", result["primary_safe_to_spend"], result["funding_shortfall"])
	}
	profile, ok := payload["profile"].(map[string]any)
	if !ok || profile["currency"] != "CNY" {
		t.Fatalf("profile=%v", payload["profile"])
	}
	if plans, ok := profile["plans"].([]any); !ok || len(plans) != 1 {
		t.Fatalf("plans=%v", profile["plans"])
	}
}

func TestPlanningReadSurfacesProblemsAndBadHorizon(t *testing.T) {
	server, _ := newPlanningTestServer(t, planningTestLedger)
	// A ledger without a profile reports setup trouble; the horizon parameter
	// is only parsed once the profile itself is sound.
	bare, _ := newPlanningTestServer(t, "2000-01-01 open Assets:Cash CNY\n")
	code, payload := planningJSON(t, bare, http.MethodGet, "/__orangecount/fava/planning", "")
	if code != http.StatusOK || payload["readiness_error"] != "Planning setup needs attention" {
		t.Fatalf("missing profile: status=%d payload=%v", code, payload)
	}
	code, payload = planningJSON(t, server, http.MethodGet, "/__orangecount/fava/planning?horizon_end=not-a-date", "")
	if code != http.StatusOK || !strings.Contains(fmt.Sprint(payload["readiness_error"]), "YYYY-MM-DD") {
		t.Fatalf("bad horizon: status=%d payload=%v", code, payload)
	}
	// A configured spendable account that the ledger does not declare fails
	// input resolution instead of silently ignoring the account.
	unknownAccount, _ := newPlanningTestServer(t, strings.Replace(planningTestLedger, "spendable_account: Assets:Bank", "spendable_account: Assets:Missing", 1))
	code, payload = planningJSON(t, unknownAccount, http.MethodGet, "/__orangecount/fava/planning", "")
	if code != http.StatusOK || !strings.Contains(fmt.Sprint(payload["readiness_error"]), "does not exist") {
		t.Fatalf("unknown account: status=%d payload=%v", code, payload)
	}
}

func TestPlanningScenarioEvaluatesDraftPurchase(t *testing.T) {
	server, _ := newPlanningTestServer(t, planningTestLedger)
	body := `{"name":"新手机","amount":"500","include_future_inflows":true,"release_adjustable":true}`
	code, payload := planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-scenario", body)
	if code != http.StatusOK {
		t.Fatalf("status=%d payload=%v", code, payload)
	}
	if payload["draft_is_persisted"] != false || payload["releases_adjustable"] != true {
		t.Fatalf("scenario flags=%v", payload)
	}
	if payload["result"] == nil {
		t.Fatalf("scenario result missing: %v", payload)
	}
}

func TestPlanningScenarioRejections(t *testing.T) {
	server, _ := newPlanningTestServer(t, planningTestLedger)
	cases := []struct {
		name    string
		body    string
		status  int
		message string
	}{
		{"bad amount", `{"amount":"zero"}`, http.StatusBadRequest, "purchase amount must be positive"},
		{"negative amount", `{"amount":"-3"}`, http.StatusBadRequest, "purchase amount must be positive"},
		{"bad horizon", `{"amount":"3","horizon_end":"soon"}`, http.StatusBadRequest, "YYYY-MM-DD"},
		{"bad draft date", `{"amount":"3","date":"soon"}`, http.StatusBadRequest, "purchase date"},
		{"draft beyond horizon", `{"amount":"3","date":"2999-01-01"}`, http.StatusBadRequest, "within the selected planning horizon"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			code, payload := planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-scenario", testCase.body)
			if code != testCase.status || !strings.Contains(fmt.Sprint(payload["error"]), testCase.message) {
				t.Fatalf("status=%d payload=%v", code, payload)
			}
		})
	}
	// A ledger without a valid profile is unprocessable for any scenario.
	empty, _ := newPlanningTestServer(t, "2000-01-01 open Assets:Cash CNY\n")
	code, payload := planningJSON(t, empty, http.MethodPost, "/__orangecount/fava/planning-scenario", `{"amount":"3"}`)
	if code != http.StatusUnprocessableEntity || !strings.Contains(fmt.Sprint(payload["error"]), "planning setup") {
		t.Fatalf("no-profile status=%d payload=%v", code, payload)
	}
}

func TestPlanningReviewDetectsPatternsAndSuggestsMatches(t *testing.T) {
	server, _ := newPlanningTestServer(t, planningTestLedger)
	code, payload := planningJSON(t, server, http.MethodGet, "/__orangecount/fava/planning-review", "")
	if code != http.StatusOK {
		t.Fatalf("status=%d payload=%v", code, payload)
	}
	occurrences, ok := payload["occurrences"].([]any)
	if !ok || len(occurrences) != 3 {
		t.Fatalf("occurrences=%v", payload["occurrences"])
	}
	if payload["completed"] != false {
		t.Fatalf("fresh ledger must not be reviewed: %v", payload["completed"])
	}
	patterns, ok := payload["patterns"].([]any)
	if !ok || len(patterns) == 0 {
		t.Fatalf("patterns=%v", payload["patterns"])
	}
	matches, _ := payload["matches"].([]any)
	if len(matches) != 1 {
		t.Fatalf("only the 2000-01-19 occurrence is inside the 7-day window: %v", payload["matches"])
	}
	match, _ := matches[0].(map[string]any)

	// Completing the review with the suggested match previews the marker; a
	// match that is not suggested is rejected instead of guessed.
	body := fmt.Sprintf(`{"matches":[{"plan_id":%q,"occurrence_id":%q}]}`, match["plan_id"], match["occurrence_id"])
	code, payload = planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-review-preview", body)
	if code != http.StatusOK || payload["valid"] != true {
		t.Fatalf("review preview status=%d payload=%v", code, payload)
	}
	if content, _ := payload["content"].(string); !strings.Contains(content, "match:") {
		t.Fatalf("preview content=%v", payload["content"])
	}
	code, payload = planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-review-preview", `{"matches":[{"plan_id":"午餐计划","occurrence_id":"0-0"}]}`)
	if code != http.StatusBadRequest || !strings.Contains(fmt.Sprint(payload["error"]), "no longer suggested") {
		t.Fatalf("unsuggested match status=%d payload=%v", code, payload)
	}
}

func TestPlanningReviewRecognizesCompletedMarker(t *testing.T) {
	marked := planningTestLedger + `2000-01-21 custom "orangecount.cycle-review.v1" "completed"
  ledger_fingerprint: "stale-fingerprint"
`
	server, _ := newPlanningTestServer(t, marked)
	code, payload := planningJSON(t, server, http.MethodGet, "/__orangecount/fava/planning-review", "")
	if code != http.StatusOK || payload["completed"] != true || payload["stale"] != true {
		t.Fatalf("status=%d completed=%v stale=%v", code, payload["completed"], payload["stale"])
	}
}

func TestPlanningGenerateProfileReplacesExistingDirective(t *testing.T) {
	server, entry := newPlanningTestServer(t, planningTestLedger)
	body := `{"kind":"profile","target":"main.bean","profile":{"id":"primary","currency":"CNY","timezone":"Asia/Singapore","minimum_reserve":"12","recorded_through":"2000-01-02","spendable_accounts":[" Assets:Bank "],"short_term_debt_accounts":["","Assets:Bank"]}}`
	code, payload := planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-generate-preview", body)
	if code != http.StatusOK || payload["valid"] != true {
		t.Fatalf("status=%d payload=%v", code, payload)
	}
	token, _ := payload["token"].(string)
	if token == "" {
		t.Fatalf("token missing: %v", payload)
	}
	// The trimmed account list dropped the empty entry.
	if content, _ := payload["content"].(string); strings.Count(content, "spendable_account") != 1 {
		t.Fatalf("generated content=%v", payload["content"])
	}
	commitBody := fmt.Sprintf(`{"token":%q,"expected_snapshot_id":%q}`, token, payload["snapshot_id"])
	code, payload = planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-commit", commitBody)
	if code != http.StatusOK || payload["published"] != true {
		t.Fatalf("commit status=%d payload=%v", code, payload)
	}
	saved, err := os.ReadFile(entry)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(saved), "planning-profile.v1"); got != 1 {
		t.Fatalf("profile must be replaced in place, found %d directives", got)
	}
	if !strings.Contains(string(saved), "minimum_reserve: 12 CNY") {
		t.Fatalf("reserve not updated: %s", saved)
	}
}

func TestPlanningGeneratePlanAppendsRevision(t *testing.T) {
	server, entry := newPlanningTestServer(t, planningTestLedger)
	body := `{"kind":"plan","target":"main.bean","plan":{"id":"话费","name":"话费","date":"2000-01-25","amount":"30","direction":"outflow","commitment":"adjustable","account":"Assets:Bank","status":"active"}}`
	code, payload := planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-generate-preview", body)
	if code != http.StatusOK || payload["valid"] != true {
		t.Fatalf("status=%d payload=%v", code, payload)
	}
	token, _ := payload["token"].(string)
	code, payload = planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-commit", fmt.Sprintf(`{"token":%q,"expected_snapshot_id":%q}`, token, payload["snapshot_id"]))
	if code != http.StatusOK || payload["published"] != true {
		t.Fatalf("commit status=%d payload=%v", code, payload)
	}
	saved, err := os.ReadFile(entry)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), `planned-flow.v1" "话费"`) {
		t.Fatalf("plan not appended: %s", saved)
	}
}

func TestPlanningGenerateRejections(t *testing.T) {
	server, _ := newPlanningTestServer(t, planningTestLedger)
	cases := []struct {
		name    string
		body    string
		status  int
		message string
	}{
		{"unknown kind", `{"kind":"mystery"}`, http.StatusBadRequest, "unknown planning form kind"},
		{"bad reserve", `{"kind":"profile","profile":{"minimum_reserve":"free"}}`, http.StatusBadRequest, "valid amount"},
		{"missing recorded-through", `{"kind":"profile","profile":{"minimum_reserve":"1"}}`, http.StatusBadRequest, "recorded-through is required"},
		{"bad recorded-through", `{"kind":"profile","profile":{"minimum_reserve":"1","recorded_through":"yesterday"}}`, http.StatusBadRequest, "recorded-through"},
		{"bad plan amount", `{"kind":"plan","plan":{"amount":"cheap"}}`, http.StatusBadRequest, "must be valid"},
		{"missing plan date", `{"kind":"plan","plan":{"amount":"1"}}`, http.StatusBadRequest, "plan date is required"},
		{"bad plan date", `{"kind":"plan","plan":{"amount":"1","date":"someday"}}`, http.StatusBadRequest, "plan date"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			code, payload := planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-generate-preview", testCase.body)
			if code != testCase.status || !strings.Contains(fmt.Sprint(payload["error"]), testCase.message) {
				t.Fatalf("status=%d payload=%v", code, payload)
			}
		})
	}
	// Two rejections need a ledger with no planning profile at all: plan
	// generation requires one, and a valid profile body cannot be redirected
	// to an out-of-graph target because an existing profile would override it.
	bare, _ := newPlanningTestServer(t, "2000-01-01 open Assets:Bank CNY\n")
	code, payload := planningJSON(t, bare, http.MethodPost, "/__orangecount/fava/planning-generate-preview", `{"kind":"plan","plan":{"id":"x","amount":"1","date":"2000-01-25"}}`)
	if code != http.StatusUnprocessableEntity || !strings.Contains(fmt.Sprint(payload["error"]), "before adding plans") {
		t.Fatalf("plan without profile status=%d payload=%v", code, payload)
	}
	body := `{"kind":"profile","target":"nope.bean","profile":{"id":"primary","currency":"CNY","timezone":"Asia/Singapore","minimum_reserve":"1","recorded_through":"2000-01-02","spendable_accounts":["Assets:Bank"]}}`
	code, payload = planningJSON(t, bare, http.MethodPost, "/__orangecount/fava/planning-generate-preview", body)
	if code != http.StatusBadRequest || !strings.Contains(fmt.Sprint(payload["error"]), "include graph") {
		t.Fatalf("unknown target status=%d payload=%v", code, payload)
	}
}

func TestPlanningPreviewAndCommitRejections(t *testing.T) {
	server, _ := newPlanningTestServer(t, planningTestLedger)
	// A preview carrying an unparsable directive is reported invalid.
	code, payload := planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-preview", `{"content":"this is not beancount"}`)
	if code != http.StatusOK || payload["valid"] != false {
		t.Fatalf("invalid preview status=%d payload=%v", code, payload)
	}
	// Unknown and stale tokens never publish.
	code, payload = planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-commit", `{"token":"bogus"}`)
	if code != http.StatusNotFound {
		t.Fatalf("bogus token status=%d payload=%v", code, payload)
	}
	code, payload = planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-preview", `{"content":"2000-01-02 * \"x\"\n  Assets:Bank -1 CNY\n  Expenses:Living:吃的 1 CNY\n"}`)
	if code != http.StatusOK || payload["valid"] != true {
		t.Fatalf("valid preview status=%d payload=%v", code, payload)
	}
	code, payload = planningJSON(t, server, http.MethodPost, "/__orangecount/fava/planning-commit", fmt.Sprintf(`{"token":%q,"expected_snapshot_id":"stale"}`, payload["token"]))
	if code != http.StatusConflict {
		t.Fatalf("stale snapshot status=%d payload=%v", code, payload)
	}
}
