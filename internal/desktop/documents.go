// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop

import (
	"errors"
	"fmt"
	"orangecount/internal/documents"
	"path/filepath"
	"sync"
)

type AttachmentDTO struct {
	PolicyRevision uint64           `json:"policy_revision"`
	ID             string           `json:"id"`
	Filename       string           `json:"filename"`
	Path           string           `json:"path"`
	Status         documents.Status `json:"status"`
}
type DocumentAction struct {
	PolicyRevision uint64 `json:"policy_revision"`
	ID             string `json:"id"`
	Path           string `json:"path"`
}

// References belong to the publication that presented them. A caller supplies
// only an ID, never an arbitrary path, and opening requires revalidation.
func (p *publication) projectAttachments(record *DirectiveDTO) error {
	if record.Kind != "document" {
		return nil
	}

	for _, field := range record.Fields {
		if field.Name != "filename" {
			continue
		}
		name := field.Value.Value.(string)
		id := fmt.Sprintf("%s:%s:%d", p.snapshot.ID, record.ID, len(record.Attachments))
		reference, found := p.documents.references[id]
		if !found {
			if len(p.documents.references) >= 8192 {
				p.documents.references = map[string]documents.Reference{}
			}
			reference = p.documents.resolver.Resolve(record.Source.Path, name)
			p.documents.references[id] = reference
		}
		record.Attachments = append(record.Attachments, AttachmentDTO{PolicyRevision: p.documents.revision, ID: id, Filename: name, Path: reference.Path, Status: reference.Status})
	}
	return nil
}

func (p *publication) authorizeDocument(id string, revision uint64) (*DocumentAction, error) {

	p.documents.mu.Lock()
	defer p.documents.mu.Unlock()
	if revision != p.documents.revision {
		return nil, ErrAttachmentPolicy
	}
	reference, found := p.documents.references[id]
	if !found || p.documents.resolver == nil {
		return nil, ErrRequest
	}
	path, err := p.documents.resolver.Revalidate(reference)
	if err != nil {
		return nil, err
	}
	return &DocumentAction{PolicyRevision: p.documents.revision, ID: id, Path: path}, nil
}

var ErrAttachmentPolicy = errors.New("attachment policy changed; refresh the reference")

type AttachmentRootsRequest struct {
	PolicyRevision  uint64   `json:"policy_revision"`
	AdditionalRoots []string `json:"additional_roots"`
}
type AttachmentRoots struct {
	PolicyRevision  uint64   `json:"policy_revision"`
	DefaultRoot     string   `json:"default_root"`
	AdditionalRoots []string `json:"additional_roots"`
}
type attachmentPolicy struct {
	mu          sync.Mutex
	revision    uint64
	defaultRoot string
	additional  []string
	resolver    *documents.Resolver
	references  map[string]documents.Reference
}

func newAttachmentPolicy(root string) (*attachmentPolicy, error) {
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	resolver, err := documents.NewResolver([]string{canonical})
	if err != nil {
		return nil, err
	}
	return &attachmentPolicy{revision: 1, defaultRoot: canonical, additional: []string{}, resolver: resolver, references: map[string]documents.Reference{}}, nil
}
func (p *attachmentPolicy) dto() AttachmentRoots {
	return AttachmentRoots{p.revision, p.defaultRoot, append([]string{}, p.additional...)}
}
func (s *Session) AttachmentRoots() (AttachmentRoots, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return AttachmentRoots{}, ErrClosed
	}
	s.documents.mu.Lock()
	defer s.documents.mu.Unlock()
	return s.documents.dto(), nil
}

// Validate the entire proposed policy before exchanging it. Root changes never
// rebuild the accounting publication. Compare-and-swap prevents late updates
// from restoring a policy that another operation has already revoked.
func (s *Session) ConfigureAttachmentRoots(request AttachmentRootsRequest) (AttachmentRoots, error) {
	if len(request.AdditionalRoots) > 32 {
		return AttachmentRoots{}, documents.ErrInvalidRoots
	}
	roots := []string{s.documents.defaultRoot}
	extra := []string{}
	seen := map[string]bool{roots[0]: true}
	for _, root := range request.AdditionalRoots {
		if !filepath.IsAbs(root) {
			return AttachmentRoots{}, documents.ErrInvalidRoots
		}
		canonical, err := filepath.EvalSymlinks(filepath.Clean(root))
		if err != nil {
			return AttachmentRoots{}, documents.ErrInvalidRoots
		}
		if !seen[canonical] {
			seen[canonical] = true
			roots = append(roots, canonical)
			extra = append(extra, canonical)
		}
	}
	resolver, err := documents.NewResolver(roots)
	if err != nil {
		return AttachmentRoots{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return AttachmentRoots{}, ErrClosed
	}
	p := s.documents
	p.mu.Lock()
	defer p.mu.Unlock()
	if request.PolicyRevision != p.revision {
		return AttachmentRoots{}, ErrAttachmentPolicy
	}
	p.revision++
	p.additional = extra
	p.resolver = resolver
	p.references = map[string]documents.Reference{}
	return p.dto(), nil
}
