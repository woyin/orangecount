// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package localfiles

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestNewCaptureFreezesBytesAndRejectsInvalidProtection(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.bean")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	content := []byte("captured output")
	capture, err := NewCapture(content, []string{source})
	if err != nil {
		t.Fatal(err)
	}
	content[0] = 'X'
	got := capture.Content()
	got[0] = 'Y'
	if string(capture.Content()) != "captured output" {
		t.Fatal("capture content changed through caller-owned byte slices")
	}

	for _, paths := range [][]string{nil, {"relative.bean"}} {
		if _, err := NewCapture([]byte("data"), paths); !errors.Is(err, ErrInvalidCapture) {
			t.Fatalf("NewCapture(%q) error = %v, want %v", paths, err, ErrInvalidCapture)
		}
	}
	if _, err := NewCapture(make([]byte, MaxCaptureBytes+1), []string{source}); !errors.Is(err, ErrInvalidCapture) {
		t.Fatalf("oversized capture error = %v, want %v", err, ErrInvalidCapture)
	}
}

func TestWithProtectedPathsRetainsOriginalIdentityAndCopiesPaths(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.bean")
	refreshedSource := filepath.Join(root, "refreshed.bean")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(refreshedSource, []byte("new source"), 0o600); err != nil {
		t.Fatal(err)
	}
	capture, err := NewCapture([]byte("export"), []string{source})
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := capture.WithProtectedPaths([]string{refreshedSource})
	if err != nil {
		t.Fatal(err)
	}
	paths := refreshed.ProtectedPaths()
	want := append([]string(nil), paths...)
	paths[0] = "/changed"
	if got := refreshed.ProtectedPaths(); !slices.Equal(got, want) {
		t.Fatalf("protected paths are not defensive copies: %v", got)
	}
	if !slices.Contains(want, source) || !slices.Contains(want, refreshedSource) {
		t.Fatalf("protection refresh lost a path: %v", want)
	}
	if string(refreshed.Content()) != "export" {
		t.Fatal("protection refresh changed captured bytes")
	}
}
