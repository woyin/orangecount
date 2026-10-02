//go:build !darwin && (!linux || !cgo)

// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package sourceexport

func platformAliasCheck(path string) error { return nil }
