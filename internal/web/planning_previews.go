package web

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type planningPreview struct {
	Content, Target, SnapshotID string
	ReplaceStart, ReplaceEnd    int
	expires                     int64
	seq                         int64 // insertion order; second-granular expires ties break on it
}
type planningPreviewStore struct {
	mu    sync.Mutex
	items map[string]planningPreview
	seq   int64
}

func newPlanningPreviewStore() *planningPreviewStore {
	return &planningPreviewStore{items: map[string]planningPreview{}}
}
func (s *planningPreviewStore) Store(p planningPreview) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().Unix()
	for k, v := range s.items {
		if v.expires <= now {
			delete(s.items, k)
		}
	}
	sum := sha256.Sum256([]byte(p.Content + p.Target + p.SnapshotID + time.Now().String()))
	id := hex.EncodeToString(sum[:])[:16]
	p.expires = now + 1800
	s.seq++
	p.seq = s.seq
	// At capacity evict the oldest insertion, matching the quick-entry
	// preview store's bounded-retention contract. expires has one-second
	// granularity, so insertion sequence is the tiebreaker.
	if _, exists := s.items[id]; !exists && len(s.items) >= 32 {
		var oldestID string
		var oldest int64
		for k, v := range s.items {
			if oldestID == "" || v.expires < oldest || (v.expires == oldest && v.seq < s.items[oldestID].seq) {
				oldestID, oldest = k, v.expires
			}
		}
		delete(s.items, oldestID)
	}
	s.items[id] = p
	return id
}
func (s *planningPreviewStore) Take(id string) (planningPreview, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.items[id]
	if !ok || p.expires <= time.Now().Unix() {
		delete(s.items, id)
		return planningPreview{}, false
	}
	delete(s.items, id)
	return p, true
}
