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
}
type planningPreviewStore struct {
	mu    sync.Mutex
	items map[string]planningPreview
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
	if len(s.items) >= 32 {
		for k := range s.items {
			delete(s.items, k)
			break
		}
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
