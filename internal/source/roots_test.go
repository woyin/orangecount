// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package source

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDocumentRootsContainment(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "receipt.pdf"), []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.pdf"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	roots, err := NewDocumentRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	resolved, err := roots.Resolve("receipt.pdf")
	if err != nil || resolved != filepath.Join(resolvedRoot, "receipt.pdf") {
		t.Fatalf("resolved=%q err=%v", resolved, err)
	}
	for _, name := range []string{"../secret.pdf", "/etc/passwd", "missing.pdf"} {
		if _, err := roots.Resolve(name); err == nil {
			t.Fatalf("expected rejection for %q", name)
		}
	}
	if runtime.GOOS != "windows" {
		if err := os.Symlink(filepath.Join(outside, "secret.pdf"), filepath.Join(root, "escape.pdf")); err == nil {
			if _, err := roots.Resolve("escape.pdf"); err == nil {
				t.Fatal("symlink escape was accepted")
			}
		}
	}
}
