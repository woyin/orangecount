// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"errors"
	"net/http"
	"strings"

	"orangecount/internal/authoring"
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

func (s *Server) handleSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	current := s.store.Current()
	if current == nil || current.Graph() == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	graph := current.Graph()
	requested := strings.TrimSpace(r.URL.Query().Get("path"))
	if requested == "" {
		writeJSON(w, struct {
			Paths []string `json:"paths"`
		}{Paths: graph.DisplayPaths()})
		return
	}
	id, ok := graph.FileIDForDisplayPath(requested)
	if !ok {
		// Preserve exact graph-member lookup for callers that already hold an
		// internal path, while never reading an arbitrary filesystem path.
		id, ok = graph.ByPath[requested]
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	file := graph.File(id)
	if file == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}{Path: graph.DisplayPath(id), Content: string(file.Data)})
}

// handleEditor exposes only files that are members of the current include
// graph. Reads are GET-only; writes require an explicit POST and a snapshot
// precondition so an editor cannot silently overwrite a concurrent reload.
func (s *Server) handleEditor(w http.ResponseWriter, r *http.Request) {
	current := s.store.Current()
	if current == nil || current.Graph() == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	suffix := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/editor"), "/")
	switch {
	case r.Method == http.MethodGet && suffix == "":
		paths := current.Graph().DisplayPaths()
		writeJSON(w, struct {
			Paths      []string `json:"paths"`
			Entry      string   `json:"entry"`
			SnapshotID string   `json:"snapshot_id"`
		}{Paths: paths, Entry: current.Graph().DisplayPath(current.Graph().Entry), SnapshotID: current.ID})
	case r.Method == http.MethodGet && (suffix == "file" || suffix == "files"):
		s.handleEditorFile(w, r, current)
	case r.Method == http.MethodPost && suffix == "validate":
		s.handleEditorValidate(w, r, current)
	case r.Method == http.MethodPost && suffix == "save":
		s.handleEditorSave(w, r, current)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.NotFound(w, r)
	}
}

func (s *Server) handleEditorFile(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		writeAPIError(w, http.StatusBadRequest, "path is required")
		return
	}
	file, display, ok := graphFile(current.Graph(), path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, struct {
		Path       string `json:"path"`
		Content    string `json:"content"`
		SnapshotID string `json:"snapshot_id"`
	}{Path: display, Content: string(file.Data), SnapshotID: current.ID})
}

type editorContentRequest struct {
	Path             string `json:"path"`
	Content          string `json:"content"`
	ExpectedSnapshot string `json:"expected_snapshot_id"`
}

func (s *Server) handleEditorValidate(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	var request editorContentRequest
	if err := decodeJSONBody(w, r, &request, 4<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, _, ok := graphFile(current.Graph(), request.Path); !ok {
		http.NotFound(w, r)
		return
	}
	file, bag := ledger.ParseText(source.SafeDisplayPath(request.Path), []byte(request.Content))
	_ = file
	diagnostics := bag.All()
	writeJSON(w, struct {
		Valid       bool                 `json:"valid"`
		Diagnostics []diagnosticResponse `json:"diagnostics"`
	}{Valid: !bag.HasErrors(), Diagnostics: diagnosticsPayload(diagnostics, nil)})
}

func (s *Server) handleEditorSave(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	var request editorContentRequest
	if err := decodeJSONBody(w, r, &request, 8<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if request.ExpectedSnapshot != "" && request.ExpectedSnapshot != current.ID {
		writeAPIError(w, http.StatusConflict, "snapshot changed; reload before saving")
		return
	}
	_, display, ok := graphFile(current.Graph(), request.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	change, err := authoring.Replace(display, []byte(request.Content))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.authoring.Publish(current.ID, change)
	if err != nil {
		status := authoringStatus(err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		writeJSON(w, struct {
			Published   bool                 `json:"published"`
			Backup      string               `json:"backup,omitempty"`
			Diagnostics []diagnosticResponse `json:"diagnostics"`
		}{Backup: result.Backup, Diagnostics: diagnosticsPayload(result.Build.Diagnostics, current.Graph())})
		return
	}
	writeJSON(w, struct {
		Published   bool                 `json:"published"`
		SnapshotID  string               `json:"snapshot_id"`
		Backup      string               `json:"backup"`
		Diagnostics []diagnosticResponse `json:"diagnostics"`
	}{Published: true, SnapshotID: result.Build.Snapshot.ID, Backup: result.Backup, Diagnostics: diagnosticsPayload(result.Build.Diagnostics, result.Build.Snapshot.Graph())})
}

// handleDocumentUpload stores an uploaded attachment beneath a configured
// document root, in the subfolder chain of the target account, the way
// Fava's put_document endpoint lays out uploads. It is a reviewed write path:
// same-origin only, account validated against the evaluation, filename
// reduced to its basename, and no overwrite of existing files.
func graphFile(graph *source.Graph, displayPath string) (*source.SourceFile, string, bool) {
	if graph == nil {
		return nil, "", false
	}
	id, ok := graph.FileIDForDisplayPath(strings.TrimSpace(displayPath))
	if !ok {
		return nil, "", false
	}
	file := graph.File(id)
	if file == nil {
		return nil, "", false
	}
	return file, graph.DisplayPath(id), true
}

func authoringStatus(err error) int {
	switch {
	case errors.Is(err, authoring.ErrSnapshotChanged), errors.Is(err, authoring.ErrSourceChanged):
		return http.StatusConflict
	case errors.Is(err, authoring.ErrTargetMissing):
		return http.StatusNotFound
	case errors.Is(err, authoring.ErrValidation):
		return http.StatusUnprocessableEntity
	case errors.Is(err, authoring.ErrInvalidProposal):
		return http.StatusBadRequest
	case errors.Is(err, authoring.ErrIO):
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// safeImportName resolves a user-supplied import path: absolute paths are
// rejected, the file must live under the adapter's root, and the name must
// stay within ASCII word characters plus a short allowlist.
