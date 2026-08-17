// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package web

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"orangecount/internal/web/favaadapter"
)

// quickPreview is a compiled-but-not-yet-published quick-entry batch. The
// server holds it briefly so the commit step can validate that the client is
// publishing exactly what was previewed, without re-interpreting free text.
type quickPreview struct {
	Token   string
	Entries []favaadapter.NewEntry
	Target  string
	expires int64
}

// quickPreviewTTL bounds how long a quick-entry preview may wait before
// commit. Like import previews, abandoned quick previews must not accumulate.
const quickPreviewTTL = 30 * time.Minute

// maxQuickPreviews caps the number of simultaneous quick-entry previews, the
// same bound the import preview store uses.
const maxQuickPreviews = 32

type quickPreviewStore struct {
	mu      sync.Mutex
	items   map[string]quickPreview
	nowUnix func() int64
}

func newQuickPreviewStore() *quickPreviewStore {
	return &quickPreviewStore{items: make(map[string]quickPreview), nowUnix: func() int64 { return time.Now().Unix() }}
}

// Store stages a preview batch and returns its single-use token; expired
// batches drop on write and, at capacity, the oldest is evicted.
func (s *quickPreviewStore) Store(entries []favaadapter.NewEntry, target string) string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.nowUnix()
	for id, existing := range s.items {
		if existing.expires <= now {
			delete(s.items, id)
		}
	}
	// Single-use token derived from the entries and the current time so two
	// identical batches still get distinct tokens.
	hasher := sha256.New()
	hasher.Write([]byte(time.Now().Format(time.RFC3339Nano)))
	for _, e := range entries {
		hasher.Write([]byte(e.Date))
		hasher.Write([]byte(e.Narration))
		for _, p := range e.Postings {
			hasher.Write([]byte(p.Account))
			hasher.Write([]byte(p.Amount))
			hasher.Write([]byte(p.Currency))
		}
	}
	token := hex.EncodeToString(hasher.Sum(nil))[:16]
	// Evict the oldest if at capacity and this is a new token.
	if _, exists := s.items[token]; !exists && len(s.items) >= maxQuickPreviews {
		var oldestID string
		var oldest int64
		for id, existing := range s.items {
			if oldestID == "" || existing.expires < oldest {
				oldestID, oldest = id, existing.expires
			}
		}
		delete(s.items, oldestID)
	}
	s.items[token] = quickPreview{
		Token:   token,
		Entries: entries,
		Target:  target,
		expires: now + int64(quickPreviewTTL/time.Second),
	}
	return token
}

// Take consumes a single-use preview token. A replayed commit gets false,
// which is how the duplicate-guard contract is enforced: one preview, one
// commit, no retry duplication.
func (s *quickPreviewStore) Take(token string) (quickPreview, bool) {
	if s == nil || token == "" {
		return quickPreview{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	preview, ok := s.items[token]
	if !ok || preview.expires <= s.now() {
		delete(s.items, token)
		return quickPreview{}, false
	}
	delete(s.items, token)
	return preview, true
}

func (s *quickPreviewStore) len() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items)
}

func (s *quickPreviewStore) now() int64 {
	if s.nowUnix != nil {
		return s.nowUnix()
	}
	return time.Now().Unix()
}
