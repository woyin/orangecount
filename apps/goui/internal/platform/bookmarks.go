// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

// Package platform adapts user-visible system actions and platform-owned
// references such as ordinary macOS bookmarks.
package platform

import (
	"errors"
	"path/filepath"
)

const maxDirectoryBookmarkBytes = 1 << 20

var (
	ErrBookmarkUnsupported = errors.New("directory bookmarks are unavailable on this platform")
	ErrBookmarkInvalid     = errors.New("directory bookmark is invalid")
	ErrBookmarkStale       = errors.New("directory bookmark is stale")
	ErrBookmarkPathChanged = errors.New("directory bookmark resolves to a different path")
	ErrBookmarkIdentity    = errors.New("directory bookmark resolves to a different filesystem object")
	ErrAliasInspection     = errors.New("Finder alias could not be inspected")
)

// IsFinderAlias reports whether path is a Finder alias. It also returns true
// for symbolic links, so callers may use it as a conservative final check.
func IsFinderAlias(path string) (bool, error) { return isFinderAlias(path) }

// DirectoryBookmark records an ordinary bookmark and the identity of the
// directory selected when it was created. It does not grant security-scoped
// access. Call ResolveDirectoryBookmark before using it and handle failures as
// a request for the user to select the directory again.
type DirectoryBookmark struct {
	Path   string `json:"path"`
	Data   []byte `json:"bookmark"`
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`
}

// Validate checks only the serialized record shape. Resolution remains
// mandatory before a caller uses the directory.
func (b DirectoryBookmark) Validate() error { return validateDirectoryBookmarkRecord(b) }

// CreateDirectoryBookmark creates an ordinary bookmark for an existing
// directory and captures its canonical path and device/inode identity.
func CreateDirectoryBookmark(path string) (DirectoryBookmark, error) {
	return createDirectoryBookmark(path)
}

// ResolveDirectoryBookmark resolves a stored bookmark without UI or volume
// mounting, then rejects stale data, path changes, and filesystem replacement.
func ResolveDirectoryBookmark(bookmark DirectoryBookmark) (string, error) {
	return resolveDirectoryBookmark(bookmark)
}

type resolvedDirectoryBookmark struct {
	Path   string
	Device uint64
	Inode  uint64
	Stale  bool
}

func validateResolvedDirectoryBookmark(expected DirectoryBookmark, actual resolvedDirectoryBookmark) error {
	if err := validateDirectoryBookmarkRecord(expected); err != nil {
		return err
	}
	if actual.Stale {
		return ErrBookmarkStale
	}
	if actual.Path == "" || !filepath.IsAbs(actual.Path) || filepath.Clean(actual.Path) != actual.Path || actual.Path != expected.Path {
		return ErrBookmarkPathChanged
	}
	if actual.Device != expected.Device || actual.Inode != expected.Inode {
		return ErrBookmarkIdentity
	}
	return nil
}

func validateDirectoryBookmarkRecord(bookmark DirectoryBookmark) error {
	if bookmark.Path == "" || !filepath.IsAbs(bookmark.Path) || filepath.Clean(bookmark.Path) != bookmark.Path || len(bookmark.Data) == 0 || len(bookmark.Data) > 1<<20 || bookmark.Device == 0 || bookmark.Inode == 0 {
		return ErrBookmarkInvalid
	}
	return nil
}
