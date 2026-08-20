// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"orangecount/internal/snapshot"
	"orangecount/internal/web/favaadapter"
)

func (s *Server) handleDocumentUpload(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	roots := s.roots.Paths()
	if len(roots) == 0 {
		writeAPIError(w, http.StatusBadRequest, "No document root is configured (serve --document-root).")
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeAPIError(w, http.StatusBadRequest, "The upload could not be parsed.")
		return
	}
	account := strings.TrimSpace(r.FormValue("account"))
	if _, ok := current.Evaluation().Accounts[account]; !ok {
		writeAPIError(w, http.StatusBadRequest, fmt.Sprintf("Not a valid account: %q", account))
		return
	}
	folder, err := uploadFolder(roots, strings.TrimSpace(r.FormValue("folder")))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil || header == nil {
		writeAPIError(w, http.StatusBadRequest, "No file uploaded.")
		return
	}
	defer file.Close()
	// The basename reduction is Fava's separator-to-space sanitization made
	// strict: directory components can never survive the upload.
	name := filepath.Base(strings.TrimSpace(header.Filename))
	if !validDocumentName(name) {
		writeAPIError(w, http.StatusBadRequest, "Uploaded file is missing a filename.")
		return
	}
	// Defense in depth: the components are validated above, but the resolved
	// target must provably stay inside the chosen document root.
	target, err := documentTargetPath(folder, account, name)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Uploaded file is missing a filename.")
		return
	}
	if _, err := os.Stat(target); err == nil {
		writeAPIError(w, http.StatusConflict, "Target path already exists: "+name)
		return
	}
	if writeErr := storeUploadedFile(target, file); writeErr != nil {
		writeAPIError(w, writeErr.status, writeErr.message)
		return
	}
	relative := documentRelativePath(folder, target)
	writeJSON(w, favaadapter.NewEnvelope(struct {
		Filename string `json:"filename"`
		Message  string `json:"message"`
	}{Filename: relative, Message: "Uploaded to " + relative}, current.BuiltAt))
}

// httpError carries the status/message pair a failed filesystem write wants
// to answer with.
type httpError struct {
	status  int
	message string
}

func (e httpError) Error() string { return e.message }

// uploadFolder resolves the form's folder selector against the configured
// document roots. An empty selector stays on the primary root; an explicit
// selector must name one of the configured roots exactly.
func uploadFolder(roots []string, requested string) (string, error) {
	if requested == "" {
		return roots[0], nil
	}
	for _, candidate := range roots {
		if requested == candidate {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("Not a documents folder: %s.", requested)
}

// validDocumentName rejects basenames that are empty, traversal-shaped, or
// option-like (leading dash).
func validDocumentName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.HasPrefix(name, "-")
}

// documentTargetPath nests name under the account's colon-separated subfolder
// chain inside root. Defense in depth: the components are validated by the
// caller, but the resolved target must provably stay inside the root.
func documentTargetPath(root, account, name string) (string, error) {
	target := filepath.Join(root, filepath.Join(strings.Split(account, ":")...), name)
	if !pathWithin(root, filepath.Clean(target)) {
		return "", fmt.Errorf("target escapes document root")
	}
	return target, nil
}

// storeUploadedFile writes the upload into target without ever overwriting:
// the account folders are created, the file itself is opened O_EXCL, and any
// later failure removes the partial file again.
func storeUploadedFile(target string, source io.Reader) *httpError {
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return &httpError{status: http.StatusInternalServerError, message: "The account folder could not be created."}
	}
	created, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return &httpError{status: http.StatusConflict, message: "Target path already exists: " + filepath.Base(target)}
	}
	if _, err := io.Copy(created, source); err != nil {
		_ = created.Close()
		_ = os.Remove(target)
		return &httpError{status: http.StatusInternalServerError, message: "The document could not be written."}
	}
	if err := created.Close(); err != nil {
		_ = os.Remove(target)
		return &httpError{status: http.StatusInternalServerError, message: "The document could not be written."}
	}
	return nil
}

// documentRelativePath renders target as the slash-separated path relative
// to its document root for the response envelope.
func documentRelativePath(root, target string) string {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		relative = filepath.Base(target)
	}
	return filepath.ToSlash(relative)
}

// handleDocumentMove moves an existing attachment into the subfolder chain of
// another account, optionally renaming it, the way Fava's move_document
// endpoint works. The file must already live beneath a configured document
// root; the target stays inside that same root and never overwrites.
func (s *Server) handleDocumentMove(w http.ResponseWriter, r *http.Request, current *snapshot.Snapshot) {
	if !requireSameOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeAPIError(w, http.StatusMethodNotAllowed, "Only POST is supported.")
		return
	}
	if s.roots.Empty() {
		writeAPIError(w, http.StatusBadRequest, "No document root is configured (serve --document-root).")
		return
	}
	var request struct {
		Filename string `json:"filename"`
		Account  string `json:"account"`
		NewName  string `json:"new_name"`
	}
	if err := decodeJSONBody(w, r, &request, 1<<16); err != nil {
		writeAPIError(w, http.StatusBadRequest, "The request body must be a JSON object.")
		return
	}
	account := strings.TrimSpace(request.Account)
	if _, ok := current.Evaluation().Accounts[account]; !ok {
		writeAPIError(w, http.StatusBadRequest, fmt.Sprintf("Not a valid account: %q", account))
		return
	}
	sourcePath, err := s.roots.Resolve(strings.TrimSpace(request.Filename))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "The document could not be found beneath a configured document root.")
		return
	}
	root := documentRootFor(s.roots.Paths(), sourcePath)
	if root == "" {
		writeAPIError(w, http.StatusInternalServerError, "The document root for this attachment could not be determined.")
		return
	}
	name := filepath.Base(strings.TrimSpace(request.NewName))
	if !validDocumentName(name) {
		name = filepath.Base(sourcePath)
	}
	// Defense in depth: account and name are validated above, but the resolved
	// target must provably stay inside the source's document root.
	target, targetErr := documentTargetPath(root, account, name)
	if targetErr != nil || filepath.Clean(target) == sourcePath {
		if targetErr != nil {
			writeAPIError(w, http.StatusBadRequest, "The new filename is not valid.")
			return
		}
		writeJSON(w, favaadapter.NewEnvelope(struct {
			Filename string `json:"filename"`
			Message  string `json:"message"`
		}{Filename: documentRelativePath(root, target), Message: "Document unchanged."}, current.BuiltAt))
		return
	}
	if _, err := os.Stat(target); err == nil {
		writeAPIError(w, http.StatusConflict, "Target path already exists: "+name)
		return
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "The account folder could not be created.")
		return
	}
	if err := os.Rename(sourcePath, target); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "The document could not be moved.")
		return
	}
	relative := documentRelativePath(root, target)
	writeJSON(w, favaadapter.NewEnvelope(struct {
		Filename string `json:"filename"`
		Message  string `json:"message"`
	}{Filename: relative, Message: "Moved to " + relative}, current.BuiltAt))
}

// documentRootFor finds the configured root that actually contains the
// attachment. The relative != "." requirement keeps the root directories
// themselves from being treated as movable documents.
func documentRootFor(roots []string, sourcePath string) string {
	for _, candidate := range roots {
		if relative, relErr := filepath.Rel(candidate, sourcePath); relErr == nil && pathWithin(candidate, sourcePath) && relative != "." {
			return candidate
		}
	}
	return ""
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative))
}

func (s *Server) handleDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	encoded := strings.TrimPrefix(r.URL.Path, "/documents/")
	if encoded == "" {
		// Bare "/documents/" is the Documents UI route, not an attachment
		// request: Fava links documents with a trailing slash, so serving the
		// shell here keeps that URL bookmarkable and refreshable.
		s.handleIndex(w, r)
		return
	}
	name, err := url.PathUnescape(encoded)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	path, err := s.roots.Resolve(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}
