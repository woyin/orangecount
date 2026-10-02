// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

// Package documents resolves local attachment references without changing
// ledger validity. It does not launch applications or read attachment contents.
package documents

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Status string

const (
	Available       Status = "available"
	Missing         Status = "missing"
	OutsideRoot     Status = "outside-root"
	UnsupportedKind Status = "unsupported-kind"
	Unreadable      Status = "unreadable"
)

var (
	ErrInvalidRoots = errors.New("attachment roots must be existing absolute directories")
	ErrChanged      = errors.New("attachment identity changed; refresh the reference")
	ErrUnavailable  = errors.New("attachment is unavailable for opening")
)

// Resolver owns a fixed set of explicitly allowed canonical directory roots.
// Construct a new resolver when the user's allowed-root configuration changes.
type Resolver struct {
	roots      []string
	identities []os.FileInfo
}

// Reference retains its declaration and observed identity for revalidation.
// Public fields are presentation only and cannot authorize a different target.
type Reference struct {
	Status       Status
	Path         string
	source       string
	declared     string
	observedPath string
	info         os.FileInfo
	owner        *Resolver
}

func NewResolver(roots []string) (*Resolver, error) {
	if len(roots) == 0 {
		return nil, ErrInvalidRoots
	}
	resolver := &Resolver{}
	for _, root := range roots {
		if !filepath.IsAbs(root) {
			return nil, ErrInvalidRoots
		}
		canonical, err := filepath.EvalSymlinks(filepath.Clean(root))
		if err != nil {
			return nil, ErrInvalidRoots
		}
		info, err := os.Stat(canonical)
		if err != nil || !info.IsDir() {
			return nil, ErrInvalidRoots
		}
		resolver.roots = append(resolver.roots, canonical)
		resolver.identities = append(resolver.identities, info)
	}
	return resolver, nil
}

func (r *Resolver) Resolve(sourcePath, declaredPath string) Reference {
	reference := Reference{Status: Unreadable, source: sourcePath, declared: declaredPath, owner: r}
	if r == nil || !filepath.IsAbs(sourcePath) || declaredPath == "" || strings.ContainsRune(declaredPath, 0) {
		return reference
	}
	for i, root := range r.roots {
		info, err := os.Stat(root)
		if err != nil || !os.SameFile(info, r.identities[i]) {
			return reference
		}
	}
	path := declaredPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(sourcePath), path)
	}
	path = filepath.Clean(path)
	canonical, err := resolveMissingPath(path)
	if err != nil {
		return reference
	}
	reference.Path = canonical
	reference.observedPath = canonical
	if !r.contains(canonical) {
		reference.Status = OutsideRoot
		return reference
	}
	info, err := os.Stat(canonical)
	if errors.Is(err, os.ErrNotExist) {
		reference.Status = Missing
		return reference
	}
	if err != nil {
		return reference
	}
	if !info.Mode().IsRegular() {
		reference.Status = UnsupportedKind
		return reference
	}
	file, err := os.Open(canonical)
	if err != nil {
		return reference
	}
	opened, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil || closeErr != nil || !os.SameFile(info, opened) {
		return reference
	}
	reference.Status = Available
	reference.info = opened
	return reference
}

// Revalidate is required immediately before handing the returned canonical path
// to a default application. It detects observed path/inode/content-metadata
// changes; callers still must handle changes after this check as an OS race.
func (r *Resolver) Revalidate(reference Reference) (string, error) {
	if r == nil || reference.owner != r || reference.info == nil || reference.Status != Available {
		return "", ErrUnavailable
	}
	current := r.Resolve(reference.source, reference.declared)
	if current.Status != Available {
		return "", ErrUnavailable
	}
	if current.observedPath != reference.observedPath || !os.SameFile(current.info, reference.info) || current.info.Size() != reference.info.Size() || !current.info.ModTime().Equal(reference.info.ModTime()) {
		return "", ErrChanged
	}
	return current.observedPath, nil
}

func (r *Resolver) contains(path string) bool {
	for _, root := range r.roots {
		relative, err := filepath.Rel(root, path)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
			return true
		}
	}
	return false
}

// Even a missing target must resolve existing ancestors: a dangling child of
// a symlink to an outside directory is outside-root, rather than merely missing.
func resolveMissingPath(path string) (string, error) { return resolveMissingPathDepth(path, 256) }
func resolveMissingPathDepth(path string, remaining int) (string, error) {
	if remaining == 0 {
		return "", ErrUnavailable
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err == nil {
		return canonical, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	info, statErr := os.Lstat(path)
	if statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		return resolveMissingPathDepth(filepath.Clean(target), remaining-1)
	}
	parent := filepath.Dir(path)
	if parent == path {
		return "", err
	}
	canonical, err = resolveMissingPathDepth(parent, remaining-1)
	if err != nil {
		return "", err
	}
	return filepath.Join(canonical, filepath.Base(path)), nil
}
