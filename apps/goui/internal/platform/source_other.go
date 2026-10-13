// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

//go:build !darwin

package platform

func openSource(string, string) error { return ErrSourceEditorUnsupported }

func sourceEditorFromApplication(string) (SourceEditor, error) {
	return SourceEditor{}, ErrSourceEditorUnsupported
}
