// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
)

// The statement route mirrors Fava's /statement/ endpoint: a journal entry
// carries a document path in its metadata, and the browser asks for that file
// by (entry hash, metadata key) instead of by filename. Resolving the pair
// here — rather than linking the raw path — keeps absolute filesystem paths
// out of URLs and lets every response stay inside the configured document
// roots.

// statementError carries the status/message pair a failed statement lookup
// answers with. Messages never contain a path: the response body must not
// leak the ledger's absolute location.
type statementError struct {
	status  int
	message string
}

func (e statementError) Error() string { return e.message }

// favaStatement implements GET statement?entry_hash=…&key=…: it resolves the
// metadata value to a document beneath a configured document root and serves
// the file inline, the way Fava's send_file_inline does.
func (s *Server) favaStatement(w http.ResponseWriter, r *http.Request, _ *snapshot.Snapshot) {
	entryHash := strings.TrimSpace(r.URL.Query().Get("entry_hash"))
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if entryHash == "" || key == "" {
		writeAPIError(w, http.StatusBadRequest, "entry_hash and key are required")
		return
	}
	current := s.store.Current()
	if current == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "no valid snapshot")
		return
	}
	path, name, err := s.statementPath(current, entryHash, key)
	if err != nil {
		var failure statementError
		if errors.As(err, &failure) {
			writeAPIError(w, failure.status, failure.message)
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "The statement could not be resolved.")
		return
	}
	writeStatementFile(w, r, path, name)
}

// statementPath is Fava's statement_path: locate the entry by hash, read the
// string metadata value, then match it against the ledger's document
// directives — first by full path, then by basename on a shared account.
func (s *Server) statementPath(current *snapshot.Snapshot, entryHash, key string) (string, string, error) {
	evaluation := current.Evaluation()
	documents := statementDocuments(evaluation)
	for _, record := range evaluation.Entries {
		if statementEntryHash(record) != entryHash {
			continue
		}
		value, ok := statementMetadataValue(record.Directive, key)
		if !ok {
			return "", "", statementError{status: http.StatusBadRequest, message: "The entry has no string metadata under that key."}
		}
		accounts := statementEntryAccounts(record.Directive)
		// Fava resolves the metadata value against the entry file's directory;
		// an absolute value stays absolute (pathlib replaces on an absolute
		// right operand, while filepath.Join would concatenate).
		fullPath := filepath.Clean(value)
		if !filepath.IsAbs(value) {
			fullPath = filepath.Clean(filepath.Join(filepath.Dir(record.File), value))
		}
		for _, document := range documents {
			// Beancount stores a document's filename resolved against the file
			// that declares it, so the same resolution applies here before the
			// full-path comparison.
			documentPath := document.filename
			if !filepath.IsAbs(documentPath) && document.file != "" {
				documentPath = filepath.Clean(filepath.Join(filepath.Dir(document.file), documentPath))
			}
			matched := filepath.Clean(documentPath) == fullPath
			if !matched && accounts[document.account] && filepath.Base(documentPath) == value {
				matched = true
			}
			if !matched {
				continue
			}
			path, err := s.statementDocumentFile(documentPath, document.filename)
			if err != nil {
				return "", "", err
			}
			return path, filepath.Base(path), nil
		}
		return "", "", statementError{status: http.StatusNotFound, message: "No matching document was found for this statement."}
	}
	return "", "", statementError{status: http.StatusNotFound, message: "The entry was not found."}
}

// statementDocument is one document directive together with the file that
// declares it; a document's relative path resolves against its own file.
type statementDocument struct {
	filename string
	account  string
	file     string
}

// statementDocuments lists every document directive in the published entry
// stream, in ledger order.
func statementDocuments(evaluation ledger.Evaluation) []statementDocument {
	documents := make([]statementDocument, 0)
	for _, record := range evaluation.Entries {
		var document ledger.Document
		switch value := record.Directive.(type) {
		case ledger.Document:
			document = value
		case *ledger.Document:
			if value == nil {
				continue
			}
			document = *value
		default:
			continue
		}
		for _, filename := range document.Filenames {
			documents = append(documents, statementDocument{filename: filename, account: document.Account, file: record.File})
		}
	}
	return documents
}

// statementEntryHash mirrors the position-derived entry identity the journal
// and context modal publish, so a statement link built from a journal row
// resolves against the same entry stream.
func statementEntryHash(record ledger.EntryRecord) string {
	digest := sha256.New()
	digest.Write([]byte(record.File))
	digest.Write([]byte{0})
	digest.Write([]byte(record.Span.String()))
	return hex.EncodeToString(digest.Sum(nil))
}

// statementMetadataValue reads one metadata key, requiring a string value the
// way Fava does: any other data type is an invalid statement reference.
func statementMetadataValue(directive ledger.Directive, key string) (string, bool) {
	for _, meta := range directiveMetadata(directive) {
		if meta.Key != key {
			continue
		}
		if meta.Value.Kind != ledger.ValueString {
			return "", false
		}
		return meta.Value.String, true
	}
	return "", false
}

// directiveMetadata returns a directive's trailing metadata. Every directive
// embeds DirectiveBase, so a single reflective field lookup covers the whole
// AST without a per-kind type switch.
func directiveMetadata(directive ledger.Directive) []ledger.Metadata {
	value := reflect.ValueOf(directive)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	field := value.FieldByName("Meta")
	if !field.IsValid() || field.Type() != reflect.TypeOf([]ledger.Metadata(nil)) {
		return nil
	}
	values, _ := field.Interface().([]ledger.Metadata)
	return values
}

// statementEntryAccounts is Fava's get_entry_accounts: the accounts an entry
// names, which decide whether a basename-only document match is acceptable.
func statementEntryAccounts(directive ledger.Directive) map[string]bool {
	accounts := map[string]bool{}
	switch value := directive.(type) {
	case ledger.Transaction:
		for _, posting := range value.Postings {
			accounts[posting.Account] = true
		}
		return accounts
	case *ledger.Transaction:
		if value != nil {
			for _, posting := range value.Postings {
				accounts[posting.Account] = true
			}
		}
		return accounts
	case ledger.Custom:
		for _, item := range value.Values {
			if item.Kind == ledger.ValueAccount {
				accounts[item.String] = true
			}
		}
		return accounts
	case *ledger.Custom:
		if value != nil {
			for _, item := range value.Values {
				if item.Kind == ledger.ValueAccount {
					accounts[item.String] = true
				}
			}
		}
		return accounts
	case ledger.Pad:
		accounts[value.Account] = true
		accounts[value.SourceAccount] = true
		return accounts
	case *ledger.Pad:
		if value != nil {
			accounts[value.Account] = true
			accounts[value.SourceAccount] = true
		}
		return accounts
	}
	if account := statementAccountField(directive); account != "" {
		accounts[account] = true
	}
	return accounts
}

// statementAccountField reads the single Account field every non-transaction
// directive that names an account carries.
func statementAccountField(directive ledger.Directive) string {
	value := reflect.ValueOf(directive)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}
	field := value.FieldByName("Account")
	if !field.IsValid() || field.Kind() != reflect.String {
		return ""
	}
	return field.String()
}

// statementDocumentFile turns a matched document path into a real file
// beneath a configured document root. The resolved document path is tried
// first, then the raw filename against the document roots — that second form
// is how attachments stored inside a documents/ folder are named.
func (s *Server) statementDocumentFile(resolvedPath, raw string) (string, error) {
	escaped := false
	resolved, outOfBounds := s.statementCandidate(resolvedPath)
	if resolved != "" {
		return resolved, nil
	}
	escaped = escaped || outOfBounds
	if !filepath.IsAbs(raw) {
		if path, err := s.roots.Resolve(raw); err == nil {
			return path, nil
		}
	}
	if escaped {
		return "", statementError{status: http.StatusForbidden, message: "The statement document is outside the configured document roots."}
	}
	return "", statementError{status: http.StatusNotFound, message: "The statement document is not available."}
}

// statementCandidate inspects one candidate path. It returns the resolved
// regular file when it exists inside a document root, or outOfBounds when the
// path exists but escapes every configured root.
func (s *Server) statementCandidate(path string) (resolved string, outOfBounds bool) {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(realPath)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	for _, root := range s.roots.Paths() {
		if pathWithin(root, filepath.Clean(realPath)) {
			return filepath.Clean(realPath), false
		}
	}
	return "", true
}

// writeStatementFile serves the resolved statement inline with Fava's headers:
// the browser opens the file instead of saving it, the basename is offered as
// the filename, and content sniffing is disabled.
func writeStatementFile(w http.ResponseWriter, r *http.Request, path, name string) {
	contentType := mime.TypeByExtension(filepath.Ext(name))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", sanitizeStatementName(name)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, path)
}

// sanitizeStatementName keeps a quoted Content-Disposition value well formed:
// backslashes and double quotes would otherwise break the header.
func sanitizeStatementName(name string) string {
	return strings.NewReplacer("\\", "_", "\"", "_").Replace(name)
}
