// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package source

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DocumentRoots is an immutable set of explicitly configured attachment roots.
// Resolved paths are real paths, so symlink escapes are rejected in addition to
// lexical traversal attempts.
type DocumentRoots struct{ roots []string }

// NewDocumentRoots resolves each candidate path to a real directory,
// rejecting symlinks that escape and non-directories.
func NewDocumentRoots(paths []string) (DocumentRoots, error) {
	roots := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return DocumentRoots{}, err
		}
		realPath, err := filepath.EvalSymlinks(absolute)
		if err != nil {
			return DocumentRoots{}, fmt.Errorf("document root %q: %w", path, err)
		}
		info, err := os.Stat(realPath)
		if err != nil {
			return DocumentRoots{}, fmt.Errorf("document root %q: %w", path, err)
		}
		if !info.IsDir() {
			return DocumentRoots{}, fmt.Errorf("document root %q is not a directory", path)
		}
		roots = append(roots, filepath.Clean(realPath))
	}
	return DocumentRoots{roots: roots}, nil
}

// Empty reports whether no document root is configured.
func (r DocumentRoots) Empty() bool { return len(r.roots) == 0 }

// Paths returns a copy of the configured root directories.
func (r DocumentRoots) Paths() []string { return append([]string(nil), r.roots...) }

// Resolve resolves a relative attachment name to an existing regular file.
// Absolute names, traversal, symlink escapes, and directories are rejected.
func (r DocumentRoots) Resolve(name string) (string, error) {
	if filepath.IsAbs(name) || name == "" {
		return "", fmt.Errorf("attachment path must be relative")
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("attachment path escapes document roots")
	}
	for _, root := range r.roots {
		candidate := filepath.Join(root, clean)
		realCandidate, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			continue
		}
		realCandidate = filepath.Clean(realCandidate)
		if !contained(root, realCandidate) {
			continue
		}
		info, err := os.Stat(realCandidate)
		if err == nil && info.Mode().IsRegular() {
			return realCandidate, nil
		}
	}
	return "", fmt.Errorf("attachment is not available beneath a configured document root")
}

func contained(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative))
}
