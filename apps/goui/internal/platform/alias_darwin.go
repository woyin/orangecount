// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

//go:build darwin

package platform

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

func isFinderAlias(path string) (alias bool, err error) {
	if err := loadBookmarkFoundation(); err != nil {
		return false, fmt.Errorf("load Foundation: %w: %w", ErrAliasInspection, err)
	}
	withBookmarkPool(func() {
		url := objc.ID(objc.GetClass("NSURL")).Send(bookmarkSelector("fileURLWithPath:"), bookmarkString(path))
		if url == 0 {
			err = ErrAliasInspection
			return
		}
		key := bookmarkString("NSURLIsAliasFileKey")
		var value objc.ID
		var failure objc.ID
		read := objc.Send[bool](url, bookmarkSelector("getResourceValue:forKey:error:"), unsafe.Pointer(&value), key, unsafe.Pointer(&failure))
		if !read || value == 0 {
			err = fmt.Errorf("read NSURLIsAliasFileKey: %w", ErrAliasInspection)
			return
		}
		alias = objc.Send[bool](value, bookmarkSelector("boolValue"))
	})
	return alias, err
}
