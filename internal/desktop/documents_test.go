// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop_test

import (
	"errors"
	"orangecount/internal/desktop"
	"orangecount/internal/documents"
	"orangecount/internal/ledger"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttachmentPresentationAndOpenRevalidation(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"all.bean", "included.bean"} {
		data, err := os.ReadFile("../../testdata/fixtures/directives/" + name)
		if err != nil {
			t.Fatal(err)
		}
		data = []byte(strings.ReplaceAll(string(data), "receipt.pdf", "receipt.txt"))
		if err = os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "receipt.txt")
	if err := os.WriteFile(target, []byte("public attachment"), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(filepath.Join(root, "all.bean"), desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	view, err := session.Read(desktop.Request{View: "journal", JournalMode: "all", DirectiveKind: "document"})
	if err != nil {
		t.Fatal(err)
	}
	items := view.Directives.Records[0].Attachments
	if len(items) != 1 || items[0].Status != documents.Available || items[0].Path != target {
		t.Fatalf("attachment not available: %+v", items)
	}
	action := desktop.Request{View: "journal", JournalMode: "all", SnapshotID: view.State.SnapshotID, DocumentID: items[0].ID}
	result, err := session.Read(action)
	if err != nil || result.DocumentAction == nil || result.DocumentAction.Path != target {
		t.Fatal("presented reference could not be authorized", err)
	}
	bad := action
	bad.SnapshotID = ""
	if _, err = session.Read(bad); !errors.Is(err, desktop.ErrRequest) {
		t.Fatal("unbound action accepted")
	}
	bad = action
	bad.DocumentID = "arbitrary-path"
	if _, err = session.Read(bad); !errors.Is(err, desktop.ErrRequest) {
		t.Fatal("unknown reference accepted")
	}
	bad = action
	bad.SnapshotID = "different"
	if _, err = session.Read(bad); !errors.Is(err, desktop.ErrStale) {
		t.Fatal("stale action accepted")
	}
	if err = os.WriteFile(target, []byte("changed public attachment"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = session.Read(action); !errors.Is(err, documents.ErrChanged) {
		t.Fatal("modified file authorized", err)
	}
	if err = os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if _, err = session.Read(action); !errors.Is(err, documents.ErrUnavailable) {
		t.Fatal("missing file authorized", err)
	}
}

func TestDirectiveRecursiveSearchAndDepthBoundary(t *testing.T) {
	session, err := desktop.OpenWithOptions("../../testdata/fixtures/directives/all.bean", desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, text := range []string{"1.25000000000000000001", "#tag", "^link"} {
		view, err := session.Read(desktop.Request{View: "journal", JournalMode: "all", DirectiveKind: "custom", Text: text})
		if err != nil || view.Directives.Page.TotalCount != 1 {
			t.Fatal("nested metadata not searchable", text, err)
		}
	}
	if _, err = session.Read(desktop.Request{View: "journal", JournalMode: "all", Text: strings.Repeat("x", 65537)}); !errors.Is(err, desktop.ErrRequest) {
		t.Fatal("oversize search accepted")
	}
	root := t.TempDir()
	data, err := os.ReadFile("../../testdata/fixtures/directives/all.bean")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data) + "\n2000-01-08 custom \"deep\" TRUE\n  value: " + strings.Repeat("[ ", 70) + "None" + strings.Repeat(" ]", 70) + "\n"
	if err = os.WriteFile(filepath.Join(root, "all.bean"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	included, err := os.ReadFile("../../testdata/fixtures/directives/included.bean")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "included.bean"), included, 0600); err != nil {
		t.Fatal(err)
	}
	deep, err := desktop.OpenWithOptions(filepath.Join(root, "all.bean"), desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer deep.Close()
	if view, readErr := deep.Read(desktop.Request{View: "journal", JournalMode: "all", DirectiveKind: "custom"}); !errors.Is(readErr, desktop.ErrResultBudget) {
		t.Fatalf("deep metadata not bounded: error=%v state=%+v summary=%+v directives=%+v", readErr, view.State, view.Summary, view.Directives)
	}
}

func TestAttachmentPageResourceBudget(t *testing.T) {
	root := t.TempDir()
	text := "2024-01-01 open Assets:Cash\n2024-01-02 document Assets:Cash" + strings.Repeat(" \"missing.txt\"", 501) + "\n"
	path := filepath.Join(root, "main.bean")
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	session, err := desktop.OpenWithOptions(path, desktop.OpenOptions{SemanticsProfile: ledger.V3})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	_, err = session.Read(desktop.Request{View: "journal", JournalMode: "all", DirectiveKind: "document"})
	if !errors.Is(err, desktop.ErrResultBudget) {
		t.Fatalf("expected attachment budget rejection, got %v", err)
	}
}
