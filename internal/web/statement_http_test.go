// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"orangecount/internal/source"
)

// statementFixture is one temporary ledger plus its document roots: two
// attachments inside the root and one outside it.
type statementFixture struct {
	server  *Server
	root    string
	outside string
}

func newStatementFixture(t *testing.T) statementFixture {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "documents")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"abs.txt", "sub/receipt.txt"} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte("statement body"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	outside := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "main.bean")
	ledgerText := strings.Join([]string{
		"2000-01-01 open Assets:Bank USD",
		"2000-01-01 open Equity:Opening USD",
		`2000-01-02 * "full path match"`,
		`  document: "` + filepath.Join(root, "abs.txt") + `"`,
		"  Assets:Bank 1.00 USD",
		"  Equity:Opening -1.00 USD",
		`2000-01-03 document Assets:Bank "` + filepath.Join(root, "abs.txt") + `"`,
		`2000-01-04 * "basename match"`,
		`  document: "receipt.txt"`,
		"  Assets:Bank 2.00 USD",
		"  Equity:Opening -2.00 USD",
		`2000-01-05 document Assets:Bank "sub/receipt.txt"`,
		`2000-01-06 * "invalid metadata"`,
		"  document: 42",
		"  Assets:Bank 3.00 USD",
		"  Equity:Opening -3.00 USD",
		`2000-01-07 * "missing document"`,
		`  document: "nowhere.txt"`,
		"  Assets:Bank 4.00 USD",
		"  Equity:Opening -4.00 USD",
		`2000-01-08 * "escaping document"`,
		`  document: "` + outside + `"`,
		"  Assets:Bank 5.00 USD",
		"  Equity:Opening -5.00 USD",
		`2000-01-09 document Assets:Bank "` + outside + `"`,
		"",
	}, "\n")
	if err := os.WriteFile(entry, []byte(ledgerText), 0o600); err != nil {
		t.Fatal(err)
	}
	built := snapshot.Build(entry)
	if built.Snapshot == nil {
		t.Fatalf("snapshot unavailable: %+v", built.Diagnostics)
	}
	roots, err := source.NewDocumentRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(Config{Store: snapshot.NewStore(built.Snapshot), DocumentRoots: roots, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	return statementFixture{server: server, root: root, outside: outside}
}

// hashOf returns the statement identity of the entry whose narration matches.
func (f statementFixture) hashOf(t *testing.T, narration string) string {
	t.Helper()
	for _, record := range f.server.store.Current().Evaluation().Entries {
		var transaction ledger.Transaction
		switch value := record.Directive.(type) {
		case ledger.Transaction:
			transaction = value
		case *ledger.Transaction:
			if value == nil {
				continue
			}
			transaction = *value
		default:
			continue
		}
		if transaction.Narration != narration {
			continue
		}
		return statementEntryHash(record)
	}
	t.Fatalf("no entry with narration %q", narration)
	return ""
}

func (f statementFixture) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	f.server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

func TestStatementRouteResolvesDocumentMetadata(t *testing.T) {
	fixture := newStatementFixture(t)
	statementURL := func(hash, key string) string {
		return "/__orangecount/fava/statement?entry_hash=" + url.QueryEscape(hash) + "&key=" + url.QueryEscape(key)
	}

	t.Run("full path match", func(t *testing.T) {
		recorder := fixture.get(t, statementURL(fixture.hashOf(t, "full path match"), "document"))
		if recorder.Code != http.StatusOK || recorder.Body.String() != "statement body" {
			t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
		}
		disposition := recorder.Header().Get("Content-Disposition")
		if !strings.HasPrefix(disposition, "inline; filename=") || !strings.Contains(disposition, "abs.txt") {
			t.Fatalf("disposition=%q", disposition)
		}
		if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("nosniff header missing: %q", recorder.Header().Get("X-Content-Type-Options"))
		}
		if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
			t.Fatalf("content-type=%q", contentType)
		}
	})

	t.Run("basename match on a shared account", func(t *testing.T) {
		recorder := fixture.get(t, statementURL(fixture.hashOf(t, "basename match"), "document"))
		if recorder.Code != http.StatusOK || recorder.Body.String() != "statement body" {
			t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Header().Get("Content-Disposition"), "receipt.txt") {
			t.Fatalf("disposition=%q", recorder.Header().Get("Content-Disposition"))
		}
	})

	t.Run("metadata is not a string", func(t *testing.T) {
		recorder := fixture.get(t, statementURL(fixture.hashOf(t, "invalid metadata"), "document"))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("missing metadata key", func(t *testing.T) {
		recorder := fixture.get(t, statementURL(fixture.hashOf(t, "invalid metadata"), "nosuchkey"))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("no matching document", func(t *testing.T) {
		recorder := fixture.get(t, statementURL(fixture.hashOf(t, "missing document"), "document"))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), fixture.root) {
			t.Fatalf("response leaks an absolute path: %s", recorder.Body.String())
		}
	})

	t.Run("document outside the roots", func(t *testing.T) {
		recorder := fixture.get(t, statementURL(fixture.hashOf(t, "escaping document"), "document"))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), fixture.outside) || recorder.Body.String() == "secret" {
			t.Fatalf("response leaks the outside file: %s", recorder.Body.String())
		}
	})

	t.Run("unknown entry hash", func(t *testing.T) {
		recorder := fixture.get(t, statementURL("deadbeef", "document"))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("missing parameters", func(t *testing.T) {
		recorder := fixture.get(t, "/__orangecount/fava/statement")
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})
}
