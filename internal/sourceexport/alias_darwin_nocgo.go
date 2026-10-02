//go:build darwin && !cgo

// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package sourceexport

import "fmt"

func platformAliasCheck(path string) error {
	return fmt.Errorf("E-EXPORT-TARGET: replacement requires Finder alias check in macOS cgo build")
}
