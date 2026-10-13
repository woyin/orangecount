// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

// Package localfiles saves immutable local captures without overwriting source
// files or partially replacing an existing destination.
package localfiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const MaxCaptureBytes = 16 << 20

var (
	ErrInvalidCapture    = errors.New("invalid file capture")
	ErrInvalidTarget     = errors.New("invalid save target")
	ErrProtectedTarget   = errors.New("save target is a protected source")
	ErrUnsupportedTarget = errors.New("save target is not a regular file")
	ErrParentChanged     = errors.New("save parent directory changed")
	ErrCancelled         = errors.New("file save cancelled")
	ErrFilesystem        = errors.New("file save filesystem operation failed")
)

// Capture freezes bytes and the source paths that must not be replaced. The
// stored FileInfo values retain filesystem identity even if a source is moved.
// Callers receive copies so the captured bytes cannot be changed after capture.
type Capture struct {
	content    []byte
	protected  []string
	identities []os.FileInfo
}

// NewCapture freezes content and captures identities for all existing protected
// paths. Missing paths remain protected by name, so they cannot be created as
// an export destination later.
func NewCapture(content []byte, protectedPaths []string) (*Capture, error) {
	if len(content) > MaxCaptureBytes || len(protectedPaths) == 0 {
		return nil, ErrInvalidCapture
	}

	paths := make([]string, 0, len(protectedPaths))
	for _, path := range protectedPaths {
		if path == "" || !filepath.IsAbs(path) {
			return nil, ErrInvalidCapture
		}
		clean := filepath.Clean(path)
		paths = append(paths, clean, canonicalProtectedPath(clean))
	}
	slices.Sort(paths)
	paths = slices.Compact(paths)
	if len(paths) == 0 {
		return nil, ErrInvalidCapture
	}

	identities := make([]os.FileInfo, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if err == nil {
			identities = append(identities, info)
			continue
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return nil, fmt.Errorf("inspect protected path %q: %w: %w", path, ErrFilesystem, err)
	}
	return &Capture{
		content:    append([]byte(nil), content...),
		protected:  paths,
		identities: identities,
	}, nil
}

func canonicalProtectedPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved)
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err == nil {
		return filepath.Join(parent, filepath.Base(path))
	}
	return path
}

// WithProtectedPaths returns a new capture with additional current protection
// paths while retaining identities from the original capture. This supports a
// protection refresh after the save panel without changing the captured bytes.
func (c *Capture) WithProtectedPaths(paths []string) (*Capture, error) {
	if c == nil {
		return nil, ErrInvalidCapture
	}
	merged := append(append([]string(nil), c.protected...), paths...)
	refreshed, err := NewCapture(c.content, merged)
	if err != nil {
		return nil, err
	}
	refreshed.identities = append(refreshed.identities, c.identities...)
	return refreshed, nil
}

// Content returns a defensive copy of the frozen bytes.
func (c *Capture) Content() []byte {
	if c == nil {
		return nil
	}
	return append([]byte(nil), c.content...)
}

// ProtectedPaths returns a defensive copy of the captured protection paths.
func (c *Capture) ProtectedPaths() []string {
	if c == nil {
		return nil
	}
	return append([]string(nil), c.protected...)
}

func (c *Capture) protects(targetPath string, targetInfo os.FileInfo) (bool, error) {
	for _, path := range c.protected {
		if filepath.Clean(path) == filepath.Clean(targetPath) {
			return true, nil
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil && filepath.Clean(resolved) == filepath.Clean(targetPath) {
			return true, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("resolve protected path %q: %w: %w", path, ErrFilesystem, err)
		}
		current, err := os.Stat(path)
		if err == nil && targetInfo != nil && os.SameFile(current, targetInfo) {
			return true, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("inspect protected path %q: %w: %w", path, ErrFilesystem, err)
		}
	}
	if targetInfo != nil {
		for _, identity := range c.identities {
			if os.SameFile(identity, targetInfo) {
				return true, nil
			}
		}
	}
	return false, nil
}
