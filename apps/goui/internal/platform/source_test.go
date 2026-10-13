// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package platform

import "testing"

func TestOrangeCountDefaultHandlerDetection(t *testing.T) {
	for _, tc := range []struct {
		id, name string
		want     bool
	}{
		{id: "org.orangecount.desktop", name: "OrangeCount", want: true},
		{id: "org.orangecount.desktop.qa", name: "OrangeCount QA", want: true},
		{id: "org.orangecount.goui.qa", name: "OrangeCount Go UI QA", want: true},
		{name: "OrangeCount Go UI Development", want: true},
		{id: "com.apple.TextEdit", name: "TextEdit", want: false},
		{id: "com.example.editor", name: "Code Editor", want: false},
	} {
		if got := isOrangeCountHandler(tc.id, tc.name); got != tc.want {
			t.Errorf("isOrangeCountHandler(%q, %q) = %t, want %t", tc.id, tc.name, got, tc.want)
		}
	}
}

func TestConfiguredSourceEditorRequiresExternalBundleIdentifier(t *testing.T) {
	for _, bundleID := range []string{"com.apple.TextEdit", "com.example.code-editor"} {
		if err := ValidateSourceEditorBundleID(bundleID); err != nil {
			t.Errorf("valid external editor bundle ID %q: %v", bundleID, err)
		}
	}
	for _, bundleID := range []string{"", "org.orangecount", "org.orangecount.goui", "com.example/path", "com..editor", "single", "com.example editor"} {
		if err := ValidateSourceEditorBundleID(bundleID); err == nil {
			t.Errorf("unsafe source editor bundle ID %q was accepted", bundleID)
		}
	}
}
