// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

//go:build darwin

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

const (
	bookmarkResolutionWithoutUI       = 1 << 8
	bookmarkResolutionWithoutMounting = 1 << 9
)

var (
	bookmarkFoundationOnce sync.Once
	bookmarkFoundationErr  error
)

func createDirectoryBookmark(path string) (bookmark DirectoryBookmark, err error) {
	canonical, device, inode, err := inspectDirectory(path)
	if err != nil {
		return DirectoryBookmark{}, err
	}
	if err := loadBookmarkFoundation(); err != nil {
		return DirectoryBookmark{}, err
	}
	withBookmarkPool(func() {
		pathObject := bookmarkString(canonical)
		url := objc.ID(objc.GetClass("NSURL")).Send(bookmarkSelector("fileURLWithPath:isDirectory:"), pathObject, true)
		if url == 0 {
			err = fmt.Errorf("create directory URL: %w", ErrBookmarkInvalid)
			return
		}
		data := objc.Send[objc.ID](url, bookmarkSelector("bookmarkDataWithOptions:includingResourceValuesForKeys:relativeToURL:error:"), uintptr(0), objc.ID(0), objc.ID(0), objc.ID(0))
		if data == 0 {
			err = fmt.Errorf("create ordinary directory bookmark: %w", ErrBookmarkInvalid)
			return
		}
		size := objc.Send[uintptr](data, bookmarkSelector("length"))
		bytes := objc.Send[unsafe.Pointer](data, bookmarkSelector("bytes"))
		if size == 0 || size > maxDirectoryBookmarkBytes || bytes == nil {
			err = ErrBookmarkInvalid
			return
		}
		bookmark = DirectoryBookmark{
			Path: canonical, Data: append([]byte(nil), unsafe.Slice((*byte)(bytes), int(size))...),
			Device: device, Inode: inode,
		}
	})
	if err != nil {
		return DirectoryBookmark{}, err
	}
	return bookmark, nil
}

func resolveDirectoryBookmark(bookmark DirectoryBookmark) (path string, err error) {
	if err := validateDirectoryBookmarkRecord(bookmark); err != nil {
		return "", err
	}
	if err := loadBookmarkFoundation(); err != nil {
		return "", err
	}
	actual := resolvedDirectoryBookmark{}
	withBookmarkPool(func() {
		data := objc.ID(objc.GetClass("NSData")).Send(bookmarkSelector("dataWithBytes:length:"), unsafe.Pointer(unsafe.SliceData(bookmark.Data)), uintptr(len(bookmark.Data)))
		runtime.KeepAlive(bookmark.Data)
		if data == 0 {
			err = ErrBookmarkInvalid
			return
		}
		var stale bool
		url := objc.Send[objc.ID](objc.ID(objc.GetClass("NSURL")), bookmarkSelector("URLByResolvingBookmarkData:options:relativeToURL:bookmarkDataIsStale:error:"), data, uintptr(bookmarkResolutionWithoutUI|bookmarkResolutionWithoutMounting), objc.ID(0), unsafe.Pointer(&stale), objc.ID(0))
		if stale {
			actual.Stale = true
			return
		}
		if url == 0 {
			err = ErrBookmarkInvalid
			return
		}
		pathObject := objc.Send[objc.ID](url, bookmarkSelector("path"))
		if pathObject == 0 {
			err = ErrBookmarkInvalid
			return
		}
		resolvedPath := objc.Send[string](pathObject, bookmarkSelector("UTF8String"))
		actual.Path, actual.Device, actual.Inode, err = inspectDirectory(resolvedPath)
	})
	if err != nil {
		return "", err
	}
	if err := validateResolvedDirectoryBookmark(bookmark, actual); err != nil {
		return "", err
	}
	return actual.Path, nil
}

func inspectDirectory(path string) (string, uint64, uint64, error) {
	if path == "" {
		return "", 0, 0, ErrBookmarkInvalid
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", 0, 0, err
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", 0, 0, err
	}
	info, err := os.Lstat(canonical)
	if err != nil {
		return "", 0, 0, err
	}
	if !info.IsDir() {
		return "", 0, 0, fmt.Errorf("bookmark target is not a directory: %w", ErrBookmarkInvalid)
	}
	identity, ok := info.Sys().(*syscall.Stat_t)
	if !ok || identity.Dev == 0 || identity.Ino == 0 {
		return "", 0, 0, ErrBookmarkInvalid
	}
	return filepath.Clean(canonical), uint64(identity.Dev), uint64(identity.Ino), nil
}

func loadBookmarkFoundation() error {
	bookmarkFoundationOnce.Do(func() {
		_, bookmarkFoundationErr = purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_GLOBAL|purego.RTLD_NOW)
	})
	return bookmarkFoundationErr
}

func withBookmarkPool(fn func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(bookmarkSelector("alloc")).Send(bookmarkSelector("init"))
	defer pool.Send(bookmarkSelector("drain"))
	fn()
}

func bookmarkString(value string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(bookmarkSelector("stringWithUTF8String:"), value+"\x00")
}

func bookmarkSelector(name string) objc.SEL { return objc.RegisterName(name) }
