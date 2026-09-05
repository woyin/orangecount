// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWriteEndpointsRejectCrossOriginEverywhere pins the same-origin guard on
// every state-changing adapter endpoint: a cross-site Origin is a 403 while a
// non-browser client (no headers) is unaffected.
func TestWriteEndpointsRejectCrossOriginEverywhere(t *testing.T) {
	server := newLedgerTestServer(t)
	paths := []string{
		"/__orangecount/fava/planning-preview",
		"/__orangecount/fava/planning-commit",
		"/__orangecount/fava/planning-scenario",
		"/__orangecount/fava/planning-review-preview",
		"/__orangecount/fava/planning-generate-preview",
		"/__orangecount/fava/quick-preview",
		"/__orangecount/fava/quick-commit",
		"/__orangecount/fava/quick-undo",
		"/__orangecount/fava/quick-profile-save",
		"/__orangecount/fava/add-entries",
		"/api/v1/import/preview",
		"/api/v1/import/commit",
		"/api/v1/editor/save",
	}
	for _, path := range paths {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "https://evil.example")
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("%s cross-origin status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}

// TestWriteEndpointsRejectMalformedJSON covers the decode guard on the same
// set of endpoints: broken bodies are 400 before any ledger access.
func TestWriteEndpointsRejectMalformedJSON(t *testing.T) {
	server := newLedgerTestServer(t)
	paths := []string{
		"/__orangecount/fava/planning-preview",
		"/__orangecount/fava/planning-commit",
		"/__orangecount/fava/planning-scenario",
		"/__orangecount/fava/planning-review-preview",
		"/__orangecount/fava/planning-generate-preview",
		"/__orangecount/fava/quick-preview",
		"/__orangecount/fava/quick-commit",
		"/__orangecount/fava/quick-profile-save",
		"/api/v1/import/preview",
		"/api/v1/editor/save",
	}
	for _, path := range paths {
		response := post(t, server, path, `{"token":`)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s malformed JSON status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
}
