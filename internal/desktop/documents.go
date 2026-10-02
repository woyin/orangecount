// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
package desktop

import (
	"fmt"
	"orangecount/internal/documents"
	"path/filepath"
)

type AttachmentDTO struct {
	ID       string           `json:"id"`
	Filename string           `json:"filename"`
	Path     string           `json:"path"`
	Status   documents.Status `json:"status"`
}
type DocumentAction struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// References belong to the publication that presented them. A caller supplies
// only an ID, never an arbitrary path, and opening requires revalidation.
func (p *publication) projectAttachments(record *DirectiveDTO) error {
	if record.Kind != "document" {
		return nil
	}
	p.documentMu.Lock()
	defer p.documentMu.Unlock()
	if p.documentResolver == nil {
		graph := p.snapshot.Graph()
		resolver, err := documents.NewResolver([]string{filepath.Dir(graph.Path(graph.Entry))})
		if err != nil {
			return err
		}
		p.documentResolver = resolver
		p.documentReferences = make(map[string]documents.Reference)
	}
	for _, field := range record.Fields {
		if field.Name != "filename" {
			continue
		}
		name := field.Value.Value.(string)
		id := fmt.Sprintf("%s:%d", record.ID, len(record.Attachments))
		reference, found := p.documentReferences[id]
		if !found {
			reference = p.documentResolver.Resolve(record.Source.Path, name)
			p.documentReferences[id] = reference
		}
		record.Attachments = append(record.Attachments, AttachmentDTO{ID: id, Filename: name, Path: reference.Path, Status: reference.Status})
	}
	return nil
}

func (p *publication) authorizeDocument(id string) (*DocumentAction, error) {
	p.documentMu.Lock()
	defer p.documentMu.Unlock()
	reference, found := p.documentReferences[id]
	if !found || p.documentResolver == nil {
		return nil, ErrRequest
	}
	path, err := p.documentResolver.Revalidate(reference)
	if err != nil {
		return nil, err
	}
	return &DocumentAction{ID: id, Path: path}, nil
}
