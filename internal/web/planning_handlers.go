package web

import (
	"fmt"
	"net/http"
	"strings"

	"orangecount/internal/authoring"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

// handlePlanningPreview validates an exact generated planning custom directive
// before retaining it as a single-use reviewed-write token.
func (s *Server) handlePlanningPreview(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request struct {
		Content          string `json:"content"`
		Target           string `json:"target"`
		ExpectedSnapshot string `json:"expected_snapshot_id"`
	}
	if err := decodeJSONBody(w, r, &request, 1<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if request.ExpectedSnapshot != "" && request.ExpectedSnapshot != current.ID {
		writeAPIError(w, http.StatusConflict, "ledger changed; reload before previewing")
		return
	}
	if _, bag := ledger.ParseText("planning-preview.bean", []byte(request.Content)); bag.HasErrors() {
		writeJSON(w, map[string]any{"valid": false, "diagnostics": diagnosticsPayload(bag.All(), nil)})
		return
	}
	graph := current.Graph()
	if graph == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no source graph")
		return
	}
	target := strings.TrimSpace(request.Target)
	if target == "" {
		target = graph.DisplayPath(graph.Entry)
	}
	_, display, ok := graphFile(graph, target)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, fmt.Sprintf("target file %q is not in the ledger include graph", target))
		return
	}
	token := s.planningPreviews.Store(planningPreview{Content: strings.TrimSpace(request.Content), Target: display, SnapshotID: current.ID})
	writeJSON(w, map[string]any{"valid": true, "token": token, "target": display, "content": request.Content, "snapshot_id": current.ID})
}

func (s *Server) handlePlanningCommit(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request struct {
		Token            string `json:"token"`
		ExpectedSnapshot string `json:"expected_snapshot_id"`
	}
	if err := decodeJSONBody(w, r, &request, 64<<10); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	preview, ok := s.planningPreviews.Take(request.Token)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "planning preview not found, expired, or already committed")
		return
	}
	if preview.SnapshotID != current.ID || (request.ExpectedSnapshot != "" && request.ExpectedSnapshot != current.ID) {
		writeAPIError(w, http.StatusConflict, "ledger changed since preview; reload and re-preview")
		return
	}
	file, display, ok := graphFile(current.Graph(), preview.Target)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "target file is no longer in the ledger include graph")
		return
	}
	content := ""
	if preview.ReplaceStart >= 0 && preview.ReplaceEnd > preview.ReplaceStart && preview.ReplaceEnd <= len(file.Data) {
		content = string(file.Data[:preview.ReplaceStart]) + preview.Content + string(file.Data[preview.ReplaceEnd:])
	} else {
		content = strings.TrimRight(string(file.Data), "\n") + "\n\n" + preview.Content + "\n"
	}
	change, err := authoring.Replace(display, []byte(content))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.authoring.Publish(current.ID, change)
	if err != nil {
		status := authoringStatus(err)
		w.WriteHeader(status)
		writeJSON(w, map[string]any{"published": false, "backup": result.Backup, "diagnostics": diagnosticsPayload(result.Build.Diagnostics, current.Graph())})
		return
	}
	writeJSON(w, map[string]any{"published": true, "snapshot_id": result.Build.Snapshot.ID, "backup": result.Backup})
}
