// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
//go:build (!darwin && !linux) || !cgo

package sourceexport

import (
	"fmt"
	"os"
	"path/filepath"
)

// No portable Go 1.22 exclusive directory rename exists. Refuse instead of
// claiming a no-replace transaction that could overwrite a raced target.
func publishNewTree(parent *os.File, from, to string) error {
	return fmt.Errorf("E-EXPORT-TARGET: exclusive tree publication requires the macOS or Linux cgo build")
}
func checkAlias(path string) error { return platformAliasCheck(path) }

func publishFile(parent *os.File, from, to string, exclusive bool) error {
	a, b := filepath.Join(parent.Name(), from), filepath.Join(parent.Name(), to)
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
