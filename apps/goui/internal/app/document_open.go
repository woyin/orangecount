// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package app

import (
	"path/filepath"
	"strings"
)

// RouteOpenFile sends an OS-opened ledger to an idle welcome window when one
// is available. A populated or busy window stays unchanged and gets a new
// window for the incoming document.
func RouteOpenFile(path string, active *Model, createWindow func() *Model) *Model {
	if path == "" {
		return active
	}
	if active == nil || !active.CanAcceptDocument() {
		if createWindow == nil {
			return active
		}
		active = createWindow()
	}
	if active != nil {
		active.OpenLedgerPath(path)
	}
	return active
}

// RouteOpenDocument sends supported document types to their owning window.
// Offline query captures can be viewed alongside an existing ledger without
// opening or changing that ledger's workspace.
func RouteOpenDocument(path string, active *Model, createWindow func() *Model) *Model {
	if strings.EqualFold(filepath.Ext(path), ".ocanalysis") {
		if path == "" {
			return active
		}
		if active == nil || !active.CanAcceptQueryAnalysis() {
			if createWindow == nil {
				return active
			}
			active = createWindow()
		}
		if active != nil {
			active.OpenQueryAnalysisPath(path)
		}
		return active
	}
	return RouteOpenFile(path, active, createWindow)
}

// RouteDroppedFiles applies the same document routing to a native window's
// file-drop event. The drop target is the preferred window for the first file.
func RouteDroppedFiles(paths []string, target *Model, createWindow func() *Model) *Model {
	active := target
	for _, path := range paths {
		active = RouteOpenDocument(path, active, createWindow)
	}
	return active
}
