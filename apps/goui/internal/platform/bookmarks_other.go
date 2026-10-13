// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

//go:build !darwin

package platform

func createDirectoryBookmark(string) (DirectoryBookmark, error) {
	return DirectoryBookmark{}, ErrBookmarkUnsupported
}

func resolveDirectoryBookmark(DirectoryBookmark) (string, error) {
	return "", ErrBookmarkUnsupported
}
