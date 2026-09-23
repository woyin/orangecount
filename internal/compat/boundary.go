// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package compat

// approvedBoundary maps a difference kind to an approved deviation id in
// docs/compatibility-ledger.json. An empty result means the difference is
// unregistered evidence and must be triaged (tools/reference/jev_triage.py).
func approvedBoundary(kind string) string {
	switch kind {
	case "query-unsupported":
		// The BeanQuery-shaped workbench intentionally covers a subset of
		// BeanQuery SQL; see ADR-0007 (workflows, not internals).
		return "query-subset"
	case "diagnostic-lines":
		// OrangeCount points errors directly to the offending posting line
		// rather than the transaction header line (docs/compatibility-ledger.json).
		return "posting-level-diagnostic-span"
	default:
		return ""
	}
}
