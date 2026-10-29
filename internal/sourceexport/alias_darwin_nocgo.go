//go:build darwin && !cgo

// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package sourceexport

import (
	"fmt"
	"os"
)

// platformAliasCheck refuses targets this build can prove are unsafe.
//
// Reading Finder metadata needs cgo, so a macOS build without cgo cannot detect
// a Finder alias. It still refuses symbolic links and anything that is not an
// ordinary file, which is every unsafe target the standard library can see;
// the residual gap (a Finder alias is a regular file with a Finder xattr) is
// recorded in docs/source-export.md rather than implied.
func platformAliasCheck(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("E-EXPORT-TARGET: unreadable target: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("E-EXPORT-TARGET: refusing to replace a symbolic link")
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("E-EXPORT-TARGET: refusing to replace a non-regular file")
	}
	return nil
}
