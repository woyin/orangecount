// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

//go:build darwin

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOrdinaryDirectoryBookmarkRoundTripAndIdentityChecks(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "selected")
	if err := os.Mkdir(selected, 0o700); err != nil {
		t.Fatal(err)
	}
	bookmark, err := CreateDirectoryBookmark(selected)
	if err != nil {
		t.Fatal(err)
	}
	if bookmark.Path == selected || len(bookmark.Data) == 0 || bookmark.Device == 0 || bookmark.Inode == 0 {
		t.Fatalf("bookmark did not capture canonical path and filesystem identity: %+v", bookmark)
	}
	got, err := ResolveDirectoryBookmark(bookmark)
	if err != nil || got != bookmark.Path {
		t.Fatalf("ResolveDirectoryBookmark = %q, %v; want %q", got, err, bookmark.Path)
	}

	moved := filepath.Join(root, "moved")
	if err := os.Rename(selected, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(selected, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveDirectoryBookmark(bookmark); err == nil {
		t.Fatal("bookmark for a moved/replaced directory was accepted")
	}
	if !errors.Is(validateResolvedDirectoryBookmark(bookmark, resolvedDirectoryBookmark{Path: bookmark.Path, Device: bookmark.Device + 1, Inode: bookmark.Inode}), ErrBookmarkIdentity) {
		t.Fatal("device identity changes were not rejected")
	}

	if err := os.RemoveAll(selected); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(moved); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveDirectoryBookmark(bookmark); err == nil {
		t.Fatal("bookmark for a deleted directory was accepted")
	}
}

func TestDirectoryBookmarkRejectsInvalidData(t *testing.T) {
	root := t.TempDir()
	bookmark, err := CreateDirectoryBookmark(root)
	if err != nil {
		t.Fatal(err)
	}
	bookmark.Data = []byte{0x01, 0x02, 0x03}
	if _, err := ResolveDirectoryBookmark(bookmark); !errors.Is(err, ErrBookmarkInvalid) {
		t.Fatalf("invalid bookmark data error = %v, want %v", err, ErrBookmarkInvalid)
	}
}
