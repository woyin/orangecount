// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

//go:build !darwin

package preferences

func OpenLegacyDefaults(string) (LegacyDefaults, error) { return nil, ErrLegacyUnsupported }
