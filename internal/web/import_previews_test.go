// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import "testing"

func TestImportPreviewStoreExpiresReplacesAndDiscardsPreviews(t *testing.T) {
	now := int64(100)
	previews := newImportPreviewStore()
	previews.nowUnix = func() int64 { return now }
	previews.Store("first", importPreview{Path: "first.bean", Content: "first"})
	if preview, ok := previews.Take("first"); !ok || preview.Path != "first.bean" || preview.expires != now+int64(importPreviewTTL.Seconds()) {
		t.Fatalf("stored preview=%+v ok=%v", preview, ok)
	}
	previews.Store("first", importPreview{Path: "first.bean", Content: "replacement"})
	if previews.len() != 1 {
		t.Fatalf("replacement grew store to %d", previews.len())
	}
	now += int64(importPreviewTTL.Seconds()) + 1
	if _, ok := previews.Take("first"); ok || previews.len() != 0 {
		t.Fatal("expired preview remained available")
	}
	previews.Store("second", importPreview{Path: "second.bean"})
	previews.Discard("second")
	if _, ok := previews.Take("second"); ok {
		t.Fatal("discarded preview remained available")
	}
}

func TestImportPreviewStoreSafelyHandlesNilAndEmptyKeys(t *testing.T) {
	var previews *importPreviewStore
	previews.Store("id", importPreview{})
	if preview, ok := previews.Take("id"); ok || preview != (importPreview{}) || previews.len() != 0 {
		t.Fatalf("nil store preview=%+v ok=%v len=%d", preview, ok, previews.len())
	}
	previews.Discard("id")
	actual := newImportPreviewStore()
	actual.Store("", importPreview{Path: "ignored"})
	if actual.len() != 0 {
		t.Fatal("empty preview ID was stored")
	}
}
