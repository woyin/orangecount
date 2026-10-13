// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

//go:build darwin

package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

var (
	ErrSourcePathInvalid       = errors.New("source path is not an existing regular file")
	ErrSourceEditorUnavailable = errors.New("could not open the source in an external editor")
)

func openSource(path, editorBundleID string) (err error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return ErrSourcePathInvalid
	}
	if editorBundleID != "" {
		if err := ValidateSourceEditorBundleID(editorBundleID); err != nil {
			return err
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect source file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return ErrSourcePathInvalid
	}
	if err := loadBookmarkFoundation(); err != nil {
		return fmt.Errorf("load Foundation: %w", err)
	}
	withBookmarkPool(func() {
		fileURL := objc.ID(objc.GetClass("NSURL")).Send(bookmarkSelector("fileURLWithPath:"), bookmarkString(path))
		if fileURL == 0 {
			err = ErrSourcePathInvalid
			return
		}
		workspace := objc.ID(objc.GetClass("NSWorkspace")).Send(bookmarkSelector("sharedWorkspace"))
		urls := objc.ID(objc.GetClass("NSArray")).Send(bookmarkSelector("arrayWithObject:"), fileURL)
		if editorBundleID != "" {
			applicationURL := workspace.Send(bookmarkSelector("URLForApplicationWithBundleIdentifier:"), bookmarkString(editorBundleID))
			if applicationURL == 0 {
				err = fmt.Errorf("find configured source editor %q: %w", editorBundleID, ErrSourceEditorUnavailable)
				return
			}
			bundle := objc.ID(objc.GetClass("NSBundle")).Send(bookmarkSelector("bundleWithURL:"), applicationURL)
			name := ""
			if bundle != 0 {
				name = sourceNSString(bundle.Send(bookmarkSelector("objectForInfoDictionaryKey:"), bookmarkString("CFBundleDisplayName")))
				if name == "" {
					name = sourceNSString(bundle.Send(bookmarkSelector("objectForInfoDictionaryKey:"), bookmarkString("CFBundleName")))
				}
			}
			if isOrangeCountHandler(editorBundleID, name) {
				err = ErrSourceEditorInvalid
				return
			}
			opened := objc.Send[bool](workspace, bookmarkSelector("openURLs:withAppBundleIdentifier:options:additionalEventParamDescriptor:launchIdentifiers:"), urls, bookmarkString(editorBundleID), uintptr(0), objc.ID(0), unsafe.Pointer(nil))
			if !opened {
				err = fmt.Errorf("open source with configured editor %q: %w", editorBundleID, ErrSourceEditorUnavailable)
			}
			return
		}
		applicationURL := workspace.Send(bookmarkSelector("URLForApplicationToOpenURL:"), fileURL)
		if applicationURL == 0 {
			if !objc.Send[bool](workspace, bookmarkSelector("openURL:"), fileURL) {
				err = fmt.Errorf("open source file: %w", ErrSourceEditorUnavailable)
			}
			return
		}
		bundle := objc.ID(objc.GetClass("NSBundle")).Send(bookmarkSelector("bundleWithURL:"), applicationURL)
		bundleID := ""
		bundleName := ""
		if bundle != 0 {
			bundleID = sourceNSString(bundle.Send(bookmarkSelector("bundleIdentifier")))
			bundleName = sourceNSString(bundle.Send(bookmarkSelector("objectForInfoDictionaryKey:"), bookmarkString("CFBundleName")))
		}
		if !isOrangeCountHandler(bundleID, bundleName) {
			if !objc.Send[bool](workspace, bookmarkSelector("openURL:"), fileURL) {
				err = fmt.Errorf("open source with its default application: %w", ErrSourceEditorUnavailable)
			}
			return
		}

		// Finder is a safe fallback if TextEdit is unavailable; it never routes
		// the file back through the OrangeCount association.
		err = openWithTextEditOrFinder(
			func() bool {
				return objc.Send[bool](workspace, bookmarkSelector("openURLs:withAppBundleIdentifier:options:additionalEventParamDescriptor:launchIdentifiers:"), urls, bookmarkString("com.apple.TextEdit"), uintptr(0), objc.ID(0), unsafe.Pointer(nil))
			},
			func() bool {
				return selectSourceInFinder(workspace, path)
			},
		)
	})
	return err
}

func selectSourceInFinder(workspace objc.ID, path string) bool {
	// activateFileViewerSelectingURLs: returns void. selectFile:... returns a
	// BOOL so the fallback can distinguish a successful Finder selection.
	return objc.Send[bool](workspace, bookmarkSelector("selectFile:inFileViewerRootedAtPath:"), bookmarkString(path), bookmarkString(""))
}

func openWithTextEditOrFinder(openWithTextEdit, revealInFinder func() bool) error {
	if openWithTextEdit() {
		return nil
	}
	if revealInFinder() {
		return nil
	}
	return fmt.Errorf("open source in TextEdit or Finder: %w", ErrSourceEditorUnavailable)
}

func sourceEditorFromApplication(path string) (editor SourceEditor, err error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path || !strings.EqualFold(filepath.Ext(path), ".app") {
		return SourceEditor{}, ErrSourceEditorInvalid
	}
	info, err := os.Stat(path)
	if err != nil {
		return SourceEditor{}, fmt.Errorf("inspect source editor application: %w", err)
	}
	if !info.IsDir() {
		return SourceEditor{}, ErrSourceEditorInvalid
	}
	if err := loadBookmarkFoundation(); err != nil {
		return SourceEditor{}, fmt.Errorf("load Foundation: %w", err)
	}
	withBookmarkPool(func() {
		fileURL := objc.ID(objc.GetClass("NSURL")).Send(bookmarkSelector("fileURLWithPath:"), bookmarkString(path))
		bundle := objc.ID(objc.GetClass("NSBundle")).Send(bookmarkSelector("bundleWithURL:"), fileURL)
		if bundle == 0 {
			err = ErrSourceEditorInvalid
			return
		}
		bundleID := sourceNSString(bundle.Send(bookmarkSelector("bundleIdentifier")))
		name := sourceNSString(bundle.Send(bookmarkSelector("objectForInfoDictionaryKey:"), bookmarkString("CFBundleDisplayName")))
		if name == "" {
			name = sourceNSString(bundle.Send(bookmarkSelector("objectForInfoDictionaryKey:"), bookmarkString("CFBundleName")))
		}
		if ValidateSourceEditorBundleID(bundleID) != nil || isOrangeCountHandler(bundleID, name) {
			err = ErrSourceEditorInvalid
			return
		}
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		editor = SourceEditor{BundleID: bundleID, Name: name}
	})
	return editor, err
}

func sourceNSString(value objc.ID) string {
	if value == 0 {
		return ""
	}
	return objc.Send[string](value, bookmarkSelector("UTF8String"))
}
