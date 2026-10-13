// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package platform

import (
	"errors"
	"testing"
)

func TestValidateResolvedDirectoryBookmark(t *testing.T) {
	expected := DirectoryBookmark{Path: "/tmp/attachments", Data: []byte{1}, Device: 9, Inode: 42}
	tests := []struct {
		name string
		got  resolvedDirectoryBookmark
		want error
	}{
		{name: "same directory", got: resolvedDirectoryBookmark{Path: expected.Path, Device: expected.Device, Inode: expected.Inode}},
		{name: "stale", got: resolvedDirectoryBookmark{Stale: true}, want: ErrBookmarkStale},
		{name: "moved directory", got: resolvedDirectoryBookmark{Path: "/tmp/renamed", Device: expected.Device, Inode: expected.Inode}, want: ErrBookmarkPathChanged},
		{name: "replaced directory", got: resolvedDirectoryBookmark{Path: expected.Path, Device: expected.Device, Inode: expected.Inode + 1}, want: ErrBookmarkIdentity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResolvedDirectoryBookmark(expected, tt.got)
			if !errors.Is(err, tt.want) {
				t.Fatalf("validation error = %v, want %v", err, tt.want)
			}
		})
	}
	if err := validateResolvedDirectoryBookmark(DirectoryBookmark{}, resolvedDirectoryBookmark{}); !errors.Is(err, ErrBookmarkInvalid) {
		t.Fatalf("invalid persisted bookmark error = %v, want %v", err, ErrBookmarkInvalid)
	}
}
