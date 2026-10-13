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
	"testing"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

func TestSourceEditorFromApplicationReadsSystemEditorBundle(t *testing.T) {
	const path = "/System/Applications/TextEdit.app"
	if _, err := os.Stat(path); err != nil {
		t.Skipf("system TextEdit bundle is unavailable: %v", err)
	}
	editor, err := SourceEditorFromApplication(path)
	if err != nil {
		t.Fatal(err)
	}
	if editor.BundleID != "com.apple.TextEdit" || editor.Name == "" {
		t.Fatalf("TextEdit identity = %+v, want com.apple.TextEdit with a display name", editor)
	}
}

func TestSourceEditorFromApplicationRejectsOrangeCountBundles(t *testing.T) {
	cases := []struct {
		name     string
		bundleID string
		appName  string
	}{
		{name: "production identity", bundleID: "org.orangecount.desktop", appName: "Ledger"},
		{name: "development identity", bundleID: "org.orangecount.desktop.goui.dev", appName: "OrangeCount Go UI"},
		{name: "QA name", bundleID: "com.example.editor", appName: "OrangeCount QA"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			appPath := filepath.Join(t.TempDir(), "Candidate.app")
			if err := os.MkdirAll(filepath.Join(appPath, "Contents"), 0700); err != nil {
				t.Fatal(err)
			}
			info := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>%s</string>
<key>CFBundleName</key><string>%s</string>
<key>CFBundlePackageType</key><string>APPL</string>
</dict></plist>
`, tc.bundleID, tc.appName)
			if err := os.WriteFile(filepath.Join(appPath, "Contents", "Info.plist"), []byte(info), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := SourceEditorFromApplication(appPath); !errors.Is(err, ErrSourceEditorInvalid) {
				t.Fatalf("self-associated bundle %q (%q) was accepted: err=%v", tc.bundleID, tc.appName, err)
			}
		})
	}
}

func TestOpenWithTextEditOrFinderFallbackOrder(t *testing.T) {
	for _, tc := range []struct {
		name       string
		textEditOK bool
		finderOK   bool
		wantCalls  string
		wantError  bool
	}{
		{name: "TextEdit succeeds", textEditOK: true, finderOK: true, wantCalls: "TextEdit"},
		{name: "Finder handles TextEdit failure", finderOK: true, wantCalls: "TextEdit,Finder"},
		{name: "both handlers fail", wantCalls: "TextEdit,Finder", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			err := openWithTextEditOrFinder(func() bool {
				calls = append(calls, "TextEdit")
				return tc.textEditOK
			}, func() bool {
				calls = append(calls, "Finder")
				return tc.finderOK
			})
			if got := strings.Join(calls, ","); got != tc.wantCalls {
				t.Fatalf("handler order = %q, want %q", got, tc.wantCalls)
			}
			if tc.wantError != errors.Is(err, ErrSourceEditorUnavailable) {
				t.Fatalf("fallback error = %v, want unavailable=%t", err, tc.wantError)
			}
		})
	}
}

func TestSelectSourceInFinderWithAppKit(t *testing.T) {
	if os.Getenv("ORANGECOUNT_GOUI_FINDER_E2E") != "1" {
		t.Skip("set ORANGECOUNT_GOUI_FINDER_E2E=1 to select a temporary source in Finder")
	}
	if _, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_GLOBAL|purego.RTLD_NOW); err != nil {
		t.Fatalf("load AppKit for Finder integration test: %v", err)
	}
	path := filepath.Join(t.TempDir(), "finder-selection.bean")
	if err := os.WriteFile(path, []byte("temporary source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadBookmarkFoundation(); err != nil {
		t.Fatal(err)
	}
	selected := false
	withBookmarkPool(func() {
		workspace := objc.ID(objc.GetClass("NSWorkspace")).Send(bookmarkSelector("sharedWorkspace"))
		selected = selectSourceInFinder(workspace, path)
	})
	if !selected {
		t.Fatal("NSWorkspace did not report selecting the temporary source in Finder")
	}
}
