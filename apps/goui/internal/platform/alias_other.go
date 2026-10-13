// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

//go:build !darwin

package platform

func isFinderAlias(string) (bool, error) { return false, nil }
