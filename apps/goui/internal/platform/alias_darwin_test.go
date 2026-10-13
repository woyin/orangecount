// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

//go:build darwin

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

func TestIsFinderAliasRecognizesRealAliasAndOrdinaryFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "target-alias")
	writeTestFinderAlias(t, target, alias)

	got, err := IsFinderAlias(alias)
	if err != nil || !got {
		t.Fatalf("IsFinderAlias(%q) = %t, %v; want true", alias, got, err)
	}
	got, err = IsFinderAlias(target)
	if err != nil || got {
		t.Fatalf("IsFinderAlias(%q) = %t, %v; want false", target, got, err)
	}
	link := filepath.Join(root, "target-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	got, err = IsFinderAlias(link)
	if err != nil || !got {
		t.Fatalf("IsFinderAlias(%q) = %t, %v; want true", link, got, err)
	}
}

func writeTestFinderAlias(t *testing.T, targetPath, aliasPath string) {
	t.Helper()
	if err := loadBookmarkFoundation(); err != nil {
		t.Fatal(err)
	}
	problem := ""
	withBookmarkPool(func() {
		nsURL := objc.ID(objc.GetClass("NSURL"))
		targetURL := nsURL.Send(bookmarkSelector("fileURLWithPath:"), bookmarkString(targetPath))
		if targetURL == 0 {
			problem = "target URL was nil"
			return
		}
		data := objc.Send[objc.ID](targetURL, bookmarkSelector("bookmarkDataWithOptions:includingResourceValuesForKeys:relativeToURL:error:"), uintptr(1<<10), objc.ID(0), objc.ID(0), objc.ID(0))
		if data == 0 {
			problem = "bookmark data was nil"
			return
		}
		aliasURL := nsURL.Send(bookmarkSelector("fileURLWithPath:"), bookmarkString(aliasPath))
		if aliasURL == 0 {
			problem = "alias URL was nil"
			return
		}
		var failure objc.ID
		ok := objc.Send[bool](nsURL, bookmarkSelector("writeBookmarkData:toURL:options:error:"), data, aliasURL, uintptr(0), unsafe.Pointer(&failure))
		if !ok {
			message := "writeBookmarkData returned false"
			if failure != 0 {
				description := objc.Send[objc.ID](failure, bookmarkSelector("localizedDescription"))
				if description != 0 {
					message = objc.Send[string](description, bookmarkSelector("UTF8String"))
				}
			}
			problem = message
		}
	})
	if problem != "" {
		t.Fatal(fmt.Errorf("could not create Finder alias fixture: %s", problem))
	}
}
