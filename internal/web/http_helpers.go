// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"orangecount/internal/diagnostic"
	"orangecount/internal/repairguidance"
	"orangecount/internal/source"
)

type diagnosticResponse struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Path     string `json:"path,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Message  string `json:"message"`
}

func diagnosticsPayload(values []diagnostic.Diagnostic, graph *source.Graph) []diagnosticResponse {
	result := make([]diagnosticResponse, 0, len(values))
	for _, value := range values {
		path := displayDiagnosticPath(value, graph)
		result = append(result, diagnosticResponse{Code: value.Code, Severity: string(value.Severity), Path: path, Line: value.Span.StartLine, Column: value.Span.StartColumn, Message: value.Message})
	}
	return result
}

// requireSameOrigin rejects cross-site requests that a browser would otherwise
// send for drive-by CSRF. The server is loopback-only, but a co-resident
// browser visiting a malicious site could still issue state-changing requests
// to it. Same-origin requests from the embedded app carry an Origin (or
// Referer) matching the loopback host; non-browser clients may omit both and
// are allowed through. When an Origin is present but does not match the request
// host, the request is rejected with 403.
func requireSameOrigin(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		writeAPIError(w, http.StatusForbidden, "cross-origin request rejected")
		return false
	}
	if strings.EqualFold(parsed.Host, r.Host) {
		return true
	}
	writeAPIError(w, http.StatusForbidden, "cross-origin request rejected")
	return false
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any, limit int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request body must contain one JSON object")
	}
	return nil
}

// handleImport serves the import pipeline: GET lists adapters and files,
// POST parses a source file into staged entries pending preview and commit.
func displayDiagnosticPath(value diagnostic.Diagnostic, graph *source.Graph) string {
	if graph != nil {
		if graph.File(value.Span.File) != nil {
			return graph.DisplayPath(value.Span.File)
		}
		if id, ok := graph.ByPath[value.Path]; ok {
			return graph.DisplayPath(id)
		}
	}
	return source.SafeDisplayPath(value.Path)
}

func diagnosticContextUnavailableReason(locale string) string {
	if locale == repairguidance.LocaleChinese {
		return "当前失败构建没有可安全展示的源文件上下文。请先修复 include 或文件读取问题，然后重新检查。"
	}
	return "Source context is unavailable for the failed build. Fix the include or file-read problem, then check again."
}

func helpTopicNotFoundMessage(locale string) string {
	if locale == repairguidance.LocaleChinese {
		return "找不到本地帮助主题。请检查诊断代码。"
	}
	return "local help topic not found; check the diagnostic code"
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	writeJSON(w, struct {
		Error string `json:"error"`
	}{Error: message})
}

func requestedLocale(r *http.Request) string {
	locale := r.URL.Query().Get("locale")
	if locale == "zh-CN" {
		return locale
	}
	return "en"
}
