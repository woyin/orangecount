// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

// Package platform adapts user-visible system actions and platform-owned
// references such as ordinary macOS bookmarks.
package platform

import (
	"errors"
	"strings"
)

var ErrSourceEditorUnsupported = errors.New("opening source files is unavailable on this platform")
var ErrSourceEditorInvalid = errors.New("source editor must be an external macOS application")

type SourceEditor struct {
	BundleID string
	Name     string
}

// OpenSource opens a ledger source with its normal application unless OrangeCount
// itself owns that file association, in which case it uses an external editor.
func OpenSource(path string) error { return OpenSourceWithEditor(path, "") }

// OpenSourceWithEditor opens a source with bundleID when it is configured; an
// empty bundleID keeps the normal system association and OrangeCount fallback.
func OpenSourceWithEditor(path, bundleID string) error {
	if bundleID != "" {
		if err := ValidateSourceEditorBundleID(bundleID); err != nil {
			return err
		}
	}
	return openSource(path, bundleID)
}

// SourceEditorFromApplication validates an application bundle selected by the
// user and returns the stable identity persisted in preferences.
func SourceEditorFromApplication(path string) (SourceEditor, error) {
	return sourceEditorFromApplication(path)
}

// ValidateSourceEditorBundleID rejects malformed and OrangeCount bundle IDs.
func ValidateSourceEditorBundleID(bundleID string) error {
	if len(bundleID) < 3 || len(bundleID) > 255 {
		return ErrSourceEditorInvalid
	}
	lowerBundleID := strings.ToLower(bundleID)
	if lowerBundleID == "org.orangecount" || isOrangeCountHandler(bundleID, "") {
		return ErrSourceEditorInvalid
	}
	parts := strings.Split(bundleID, ".")
	if len(parts) < 2 {
		return ErrSourceEditorInvalid
	}
	for _, part := range parts {
		if part == "" {
			return ErrSourceEditorInvalid
		}
		for _, r := range part {
			if !('a' <= r && r <= 'z') && !('A' <= r && r <= 'Z') && !('0' <= r && r <= '9') && r != '-' {
				return ErrSourceEditorInvalid
			}
		}
	}
	return nil
}

func isOrangeCountHandler(bundleID, bundleName string) bool {
	if strings.HasPrefix(strings.ToLower(bundleID), "org.orangecount.") {
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(bundleName)), "orangecount")
}
