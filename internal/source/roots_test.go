// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package source

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestDocumentRootsValidateConfigurationAndReturnDefensivePaths(t *testing.T) {
	if roots, err := NewDocumentRoots([]string{"", "  "}); err != nil || !roots.Empty() || len(roots.Paths()) != 0 {
		t.Fatalf("empty roots=%+v err=%v paths=%v", roots, err, roots.Paths())
	}
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := NewDocumentRoots([]string{missing}); err == nil {
		t.Fatal("missing root was accepted")
	}
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewDocumentRoots([]string{file}); err == nil {
		t.Fatal("file root was accepted")
	}

	root := t.TempDir()
	roots, err := NewDocumentRoots([]string{root, root})
	if err != nil {
		t.Fatal(err)
	}
	paths := roots.Paths()
	if len(paths) != 2 || !reflect.DeepEqual(paths, roots.Paths()) {
		t.Fatalf("paths=%v", paths)
	}
	paths[0] = "changed"
	if roots.Paths()[0] == "changed" {
		t.Fatal("Paths leaked internal slice")
	}
	for _, name := range []string{"", ".", "dir/../..", "directory"} {
		if _, err := roots.Resolve(name); err == nil {
			t.Errorf("expected rejection for %q", name)
		}
	}
}

func TestContainedRequiresARealDescendant(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "root")
	cases := []struct {
		path string
		want bool
	}{
		{root, true},
		{filepath.Join(root, "child"), true},
		{filepath.Join(string(filepath.Separator), "tmp", "root-other"), false},
		{filepath.Join(string(filepath.Separator), "tmp"), false},
	}
	for _, tc := range cases {
		if got := contained(root, tc.path); got != tc.want {
			t.Errorf("contained(%q, %q)=%v, want %v", root, tc.path, got, tc.want)
		}
	}
}
