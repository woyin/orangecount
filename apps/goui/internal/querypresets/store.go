// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

// Package querypresets stores only query text the user explicitly saves as a
// named preset. Drafts and executed results are never written here.
package querypresets

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"orangecount/apps/goui/internal/preferences"
	"orangecount/internal/query"
)

const SchemaVersion = 1

const (
	maxStoreBytes = 16 << 20
	maxLedgers    = 256
	maxPresets    = 64
	maxNameRunes  = 80
	maxNameBytes  = 320
)

var (
	ErrPath          = errors.New("query preset store path is empty")
	ErrSchema        = errors.New("unsupported query preset schema")
	ErrInvalid       = errors.New("invalid query preset document or value")
	ErrNotFound      = errors.New("query preset not found")
	ErrDuplicateName = errors.New("query preset name already exists")
	ErrLimit         = errors.New("query preset storage limit reached")
)

type Preset struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Text string `json:"text"`
}

type Ledger struct {
	Identity string   `json:"identity"`
	Presets  []Preset `json:"presets"`
}

type Document struct {
	SchemaVersion int      `json:"schema_version"`
	Ledgers       []Ledger `json:"ledgers"`
}

type Store struct {
	mu        sync.Mutex
	path      string
	writeFile func(string, []byte) error
}

func NewStore(path string) *Store {
	return &Store{path: path, writeFile: atomicWrite}
}

// LoadLedger returns saved presets for an existing ledger without exposing
// its path in the persisted document.
func (s *Store) LoadLedger(path string) ([]Preset, error) {
	identity, err := preferences.Identity(path)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	for _, ledger := range doc.Ledgers {
		if ledger.Identity == identity {
			return clonePresets(ledger.Presets), nil
		}
	}
	return []Preset{}, nil
}

// Save adds a named preset for the ledger. Names are unique per ledger,
// case-insensitively; saving a duplicate never silently replaces user data.
func (s *Store) Save(path, name, text string) ([]Preset, error) {
	identity, err := preferences.Identity(path)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if !validName(name) || !validText(text) {
		return nil, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	index := ledgerIndex(doc.Ledgers, identity)
	if index >= 0 {
		for _, existing := range doc.Ledgers[index].Presets {
			if strings.EqualFold(existing.Name, name) {
				return nil, ErrDuplicateName
			}
		}
		if len(doc.Ledgers[index].Presets) >= maxPresets {
			return nil, ErrLimit
		}
	} else if len(doc.Ledgers) >= maxLedgers {
		return nil, ErrLimit
	}
	newPreset, err := newPreset(name, text)
	if err != nil {
		return nil, err
	}
	if index < 0 {
		doc.Ledgers = append(doc.Ledgers, Ledger{Identity: identity, Presets: []Preset{newPreset}})
	} else {
		doc.Ledgers[index].Presets = append(doc.Ledgers[index].Presets, newPreset)
	}
	sort.Slice(doc.Ledgers, func(i, j int) bool { return doc.Ledgers[i].Identity < doc.Ledgers[j].Identity })
	if err := s.writeLocked(doc); err != nil {
		return nil, err
	}
	index = ledgerIndex(doc.Ledgers, identity)
	return clonePresets(doc.Ledgers[index].Presets), nil
}

// Delete removes one explicitly saved preset. Unknown IDs leave the store
// unchanged.
func (s *Store) Delete(path, id string) ([]Preset, error) {
	identity, err := preferences.Identity(path)
	if err != nil {
		return nil, err
	}
	if !validID(id) {
		return nil, ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	ledgerAt := ledgerIndex(doc.Ledgers, identity)
	if ledgerAt < 0 {
		return nil, ErrNotFound
	}
	presets := doc.Ledgers[ledgerAt].Presets
	presetAt := -1
	for i := range presets {
		if presets[i].ID == id {
			presetAt = i
			break
		}
	}
	if presetAt < 0 {
		return nil, ErrNotFound
	}
	presets = append(presets[:presetAt], presets[presetAt+1:]...)
	if len(presets) == 0 {
		doc.Ledgers = append(doc.Ledgers[:ledgerAt], doc.Ledgers[ledgerAt+1:]...)
	} else {
		doc.Ledgers[ledgerAt].Presets = presets
	}
	if err := s.writeLocked(doc); err != nil {
		return nil, err
	}
	return clonePresets(presets), nil
}

func (s *Store) loadLocked() (Document, error) {
	if s.path == "" {
		return Document{}, ErrPath
	}
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Document{SchemaVersion: SchemaVersion, Ledgers: []Ledger{}}, nil
	}
	if err != nil {
		return Document{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Document{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxStoreBytes {
		return Document{}, ErrInvalid
	}
	data, err := io.ReadAll(io.LimitReader(file, maxStoreBytes+1))
	if err != nil {
		return Document{}, err
	}
	if len(data) > maxStoreBytes {
		return Document{}, ErrInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var doc Document
	if err := decoder.Decode(&doc); err != nil {
		return Document{}, fmt.Errorf("decode query preset store: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Document{}, ErrInvalid
	}
	if doc.SchemaVersion != SchemaVersion {
		return Document{}, ErrSchema
	}
	if err := validateDocument(doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func (s *Store) writeLocked(doc Document) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxStoreBytes {
		return ErrLimit
	}
	if s.writeFile == nil {
		return ErrPath
	}
	return s.writeFile(s.path, data)
}

func validateDocument(doc Document) error {
	if len(doc.Ledgers) > maxLedgers {
		return ErrLimit
	}
	seenLedgers := make(map[string]bool, len(doc.Ledgers))
	for _, ledger := range doc.Ledgers {
		if !validIdentity(ledger.Identity) || seenLedgers[ledger.Identity] || len(ledger.Presets) == 0 || len(ledger.Presets) > maxPresets {
			return ErrInvalid
		}
		seenLedgers[ledger.Identity] = true
		seenIDs := make(map[string]bool, len(ledger.Presets))
		for index, preset := range ledger.Presets {
			duplicateName := false
			for _, earlier := range ledger.Presets[:index] {
				if strings.EqualFold(earlier.Name, preset.Name) {
					duplicateName = true
					break
				}
			}
			if !validID(preset.ID) || seenIDs[preset.ID] || !validName(preset.Name) || duplicateName || !validText(preset.Text) {
				return ErrInvalid
			}
			seenIDs[preset.ID] = true
		}
	}
	return nil
}

func validIdentity(identity string) bool {
	if len(identity) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(identity)
	return err == nil
}

func validID(id string) bool {
	if len(id) != 32 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func validName(name string) bool {
	return name != "" && name == strings.TrimSpace(name) && utf8.ValidString(name) && !strings.ContainsRune(name, '\x00') && len(name) <= maxNameBytes && utf8.RuneCountInString(name) <= maxNameRunes
}

func validText(text string) bool {
	return text != "" && strings.TrimSpace(text) != "" && utf8.ValidString(text) && !strings.ContainsRune(text, '\x00') && len(text) <= query.QueryTextLimit
}

func ledgerIndex(ledgers []Ledger, identity string) int {
	for i := range ledgers {
		if ledgers[i].Identity == identity {
			return i
		}
	}
	return -1
}

func clonePresets(presets []Preset) []Preset {
	return append([]Preset(nil), presets...)
}

func newPreset(name, text string) (Preset, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return Preset{}, err
	}
	return Preset{ID: hex.EncodeToString(id[:]), Name: name, Text: text}, nil
}

func atomicWrite(path string, data []byte) error {
	if path == "" {
		return ErrPath
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".query-presets-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
