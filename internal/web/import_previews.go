// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"sync"
	"time"
)

// importPreview is a reviewed-but-not-yet-published ledger addition.
// It is intentionally kept outside Server so the server's HTTP lifecycle
// does not also own preview retention and synchronization policy.
type importPreview struct {
	Path    string
	Content string
	expires int64
}

// importPreviewTTL bounds how long a preview may wait before being committed.
// Previews are anonymous server-side state; an abandoned preview must not
// accumulate forever in memory.
const importPreviewTTL = 30 * time.Minute

// maxImportPreviews caps the number of previews retained simultaneously. A
// client that repeatedly previews hash-distinct content cannot grow server
// memory without bound.
const maxImportPreviews = 32

func (p importPreview) live(unix int64) bool { return p.expires > unix }

// importPreviewStore owns the concurrent, bounded, expiring preview cache.
// Its interface is deliberately limited to the reviewed-write lifecycle:
// store a preview, retrieve a live preview, or discard a published preview.
type importPreviewStore struct {
	mu      sync.Mutex
	items   map[string]importPreview
	nowUnix func() int64
}

func newImportPreviewStore() *importPreviewStore {
	return &importPreviewStore{items: make(map[string]importPreview), nowUnix: func() int64 { return time.Now().Unix() }}
}

func (s *importPreviewStore) Store(id string, preview importPreview) {
	if s == nil || id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for existingID, existing := range s.items {
		if !existing.live(now) {
			delete(s.items, existingID)
		}
	}
	preview.expires = now + int64(importPreviewTTL/time.Second)
	if _, replacesExisting := s.items[id]; !replacesExisting && len(s.items) >= maxImportPreviews {
		var oldestID string
		var oldest int64
		for existingID, existing := range s.items {
			if oldestID == "" || existing.expires < oldest {
				oldestID, oldest = existingID, existing.expires
			}
		}
		delete(s.items, oldestID)
	}
	s.items[id] = preview
}

func (s *importPreviewStore) Take(id string) (importPreview, bool) {
	if s == nil || id == "" {
		return importPreview{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	preview, ok := s.items[id]
	if !ok || !preview.live(s.now()) {
		delete(s.items, id)
		return importPreview{}, false
	}
	return preview, true
}

func (s *importPreviewStore) Discard(id string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, id)
}

func (s *importPreviewStore) len() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items)
}

func (s *importPreviewStore) now() int64 {
	if s.nowUnix == nil {
		return time.Now().Unix()
	}
	return s.nowUnix()
}
