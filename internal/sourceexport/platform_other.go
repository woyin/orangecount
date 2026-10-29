// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
//go:build (!darwin && !linux) || !cgo

package sourceexport

import (
	"fmt"
	"os"
	"path/filepath"
)

// The cgo builds publish through the directory descriptor with
// renameatx_np(RENAME_EXCL) and linkat, which combine "create or fail" with a
// single visible step. Go 1.22 has no portable equivalent, so these builds
// reproduce both properties from the standard library:
//
//   - mkdir(2) creates the name or fails with EEXIST, which is the no-replace
//     guarantee;
//   - rename(2) onto an empty directory is permitted by POSIX, so the staged
//     tree still becomes visible in one step and no partial tree is exposed.
//
// The residual difference is a brief window in which an empty destination
// directory exists. A crash inside that window leaves an empty directory, which
// the next export reports as an existing target instead of overwriting it.

// publishParentPath confirms the held directory descriptor still names the path
// recorded for it, so a concurrent rename of the parent cannot redirect the
// publication. The cgo builds operate from the descriptor and need no such
// check.
func publishParentPath(parent *os.File) (string, error) {
	name := parent.Name()
	if name == "" {
		return "", fmt.Errorf("E-EXPORT-TARGET: staging parent has no path")
	}
	held, err := parent.Stat()
	if err != nil {
		return "", err
	}
	current, err := os.Stat(name)
	if err != nil {
		return "", err
	}
	if !held.IsDir() || !os.SameFile(held, current) {
		return "", fmt.Errorf("E-EXPORT-TARGET: staging parent changed before publication")
	}
	return name, nil
}

func publishNewTree(parent *os.File, from, to string) error {
	dir, err := publishParentPath(parent)
	if err != nil {
		return err
	}
	target := filepath.Join(dir, to)
	// Refuse an existing target: rename(2) would otherwise replace an ordinary
	// file or an empty directory with the staged tree. This is also why the
	// caller must not pre-create the destination.
	if _, statErr := os.Lstat(target); statErr == nil {
		return &os.LinkError{Op: "rename", Old: from, New: to, Err: os.ErrExist}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	// rename(2) publishes the whole staged tree in one step: either every file
	// appears or nothing changes. The check above and this rename are not one
	// atomic operation, which matches the documented boundary: an equal-privilege
	// process racing this call is outside the isolation claim.
	return os.Rename(filepath.Join(dir, from), target)
}

func checkAlias(path string) error { return platformAliasCheck(path) }

func publishFile(parent *os.File, from, to string, exclusive bool) error {
	dir, err := publishParentPath(parent)
	if err != nil {
		return err
	}
	a, b := filepath.Join(dir, from), filepath.Join(dir, to)
	if exclusive {
		return os.Link(a, b)
	}
	return os.Rename(a, b)
}

func pathForDirectory(parent *os.File, initial string) (string, error) {
	if path, err := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", parent.Fd())); err == nil {
		return path, nil
	}
	info, err := parent.Stat()
	if err != nil {
		return "", err
	}
	candidates, _ := os.ReadDir(filepath.Dir(initial))
	for _, candidate := range candidates {
		path := filepath.Join(filepath.Dir(initial), candidate.Name())
		current, err := os.Stat(path)
		if err == nil && os.SameFile(info, current) {
			return path, nil
		}
	}
	return "", fmt.Errorf("cannot locate staging parent")
}
