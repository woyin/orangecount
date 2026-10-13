// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package widgets

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// SourceLocation is an immutable source position from the core projection.
type SourceLocation struct {
	Path   string
	Line   int
	Column int
}

// SourceActions supplies the app-owned actions for a source row. Open and
// Reveal stay in the app because opening a source must validate its current
// workspace/snapshot identity and Finder is a platform action.
type SourceActions struct {
	CopyLabel    string
	RevealLabel  string
	OpenLabel    string
	OpenDisabled bool
	Copy         func()
	Reveal       func()
	Open         func()
}

// SourceRow presents one stable location and its available source actions.
// Empty paths render a placeholder and never produce an actionable link.
func SourceRow(c *ui.Context, location SourceLocation, actions SourceActions) *ui.Element {
	if location.Path == "" {
		return ui.Text(c, "—")
	}
	row := ui.Row(c).Gap(8).AlignItems(ui.Center)
	row.Children(func() {
		label := SourceText(location)
		ui.Text(c, label).Label(label).Tooltip(label).SingleLine().TextColor(c.Theme().TextMuted).Grow(1)
		if actions.Copy != nil && actions.CopyLabel != "" {
			if ui.Button(c, actions.CopyLabel).Label(actions.CopyLabel + " · " + label).Clicked() {
				actions.Copy()
			}
		}
		if actions.Reveal != nil && actions.RevealLabel != "" {
			if ui.Button(c, actions.RevealLabel).Label(actions.RevealLabel + " · " + label).Clicked() {
				actions.Reveal()
			}
		}
		if actions.Open != nil && actions.OpenLabel != "" {
			if ui.Button(c, actions.OpenLabel).Label(actions.OpenLabel + " · " + label).Disabled(actions.OpenDisabled).Clicked() {
				actions.Open()
			}
		}
	})
	return row
}

// SourceText formats a path as path[:line[:column]] without inventing a
// location for line-zero entries.
func SourceText(location SourceLocation) string {
	if location.Path == "" {
		return "—"
	}
	if location.Line <= 0 {
		return location.Path
	}
	if location.Column > 0 {
		return fmt.Sprintf("%s:%d:%d", location.Path, location.Line, location.Column)
	}
	return fmt.Sprintf("%s:%d", location.Path, location.Line)
}
