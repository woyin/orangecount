// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

// Package preferences stores non-sensitive per-ledger reading controls.
// It deliberately excludes ledger contents, queries, and result projections.
package preferences

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"orangecount/apps/goui/internal/platform"
)

const SchemaVersion = 1

const maxPreferencesBytes = 64 << 20

var (
	ErrPath        = errors.New("preferences path is empty")
	ErrSchema      = errors.New("unsupported preferences schema")
	ErrInvalid     = errors.New("invalid preferences document")
	ErrInvalidDate = errors.New("date must use YYYY-MM-DD and start must not be after end")
)

type LedgerPreferences struct {
	Identity                  string                        `json:"identity"`
	SemanticsProfile          string                        `json:"semantics_profile"`
	Locale                    string                        `json:"locale,omitempty"`
	TextScale                 float32                       `json:"text_scale,omitempty"`
	DateFrom                  string                        `json:"date_from,omitempty"`
	DateTo                    string                        `json:"date_to,omitempty"`
	Window                    string                        `json:"window"`
	Currency                  string                        `json:"currency,omitempty"`
	Valuation                 string                        `json:"valuation"`
	Columns                   map[string][]string           `json:"columns,omitempty"`
	ColumnWidths              map[string]map[string]float32 `json:"column_widths,omitempty"`
	AttachmentRootsGeneration uint64                        `json:"attachment_roots_generation,omitempty"`
	AttachmentRoots           []platform.DirectoryBookmark  `json:"attachment_roots"`
}

type Document struct {
	SchemaVersion          int                 `json:"schema_version"`
	Ledgers                []LedgerPreferences `json:"ledgers"`
	RecentLedgerPaths      []string            `json:"recent_ledger_paths,omitempty"`
	SourceEditorBundleID   string              `json:"source_editor_bundle_id,omitempty"`
	LegacyMigrationVersion int                 `json:"legacy_migration_version,omitempty"`
}

// BackupSnapshot is a validated, immutable-by-convention export of the
// preference document at one point in time. ProtectedPaths must be passed to
// the safe file writer so an export cannot replace the live preference file.
type BackupSnapshot struct {
	Content        []byte
	ProtectedPaths []string
}

type Store struct {
	mu           sync.Mutex
	path         string
	writeFile    func(string, []byte) error
	listenersMu  sync.Mutex
	listeners    map[uint64]attachmentRootsListener
	nextListener uint64
}

type AttachmentRootsChange struct {
	Identity   string
	Origin     string
	Generation uint64
	Roots      []platform.DirectoryBookmark
}

type attachmentRootsListener struct {
	owner string
	fn    func(AttachmentRootsChange)
}

func NewStore(path string) *Store {
	return &Store{path: path, writeFile: atomicWrite, listeners: make(map[uint64]attachmentRootsListener)}
}

// Identity returns a stable, non-reversible key for an existing ledger path.
// The full path is not written to the preference file.
func Identity(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", ErrPath
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve ledger identity: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("ledger identity requires an existing regular file")
	}
	sum := sha256.Sum256([]byte(filepath.Clean(resolved)))
	return hex.EncodeToString(sum[:]), nil
}

func (s *Store) Load() (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Store) ForLedger(path string) (LedgerPreferences, error) {
	identity, err := Identity(path)
	if err != nil {
		return LedgerPreferences{}, err
	}
	doc, err := s.Load()
	if err != nil {
		return LedgerPreferences{}, err
	}
	for _, item := range doc.Ledgers {
		if item.Identity == identity {
			return cloneLedger(item), nil
		}
	}
	return LedgerPreferences{Identity: identity, SemanticsProfile: "legacy", Locale: "zh-CN", TextScale: 1, Window: "all", Valuation: "at-cost", Columns: map[string][]string{}}, nil
}

// UpdateLedger serializes read-modify-write so changes from separate windows
// in this application process cannot overwrite each other.
func (s *Store) UpdateLedger(path string, update func(*LedgerPreferences) error) error {
	identity, err := Identity(path)
	if err != nil {
		return err
	}
	if update == nil {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.loadLocked()
	if err != nil {
		return err
	}
	index := -1
	for i := range doc.Ledgers {
		if doc.Ledgers[i].Identity == identity {
			index = i
			break
		}
	}
	item := LedgerPreferences{Identity: identity, SemanticsProfile: "legacy", Locale: "zh-CN", TextScale: 1, Window: "all", Valuation: "at-cost", Columns: map[string][]string{}}
	if index >= 0 {
		item = cloneLedger(doc.Ledgers[index])
	}
	if err := update(&item); err != nil {
		return err
	}
	item.Identity = identity
	if err := validateLedger(item); err != nil {
		return err
	}
	if index >= 0 {
		doc.Ledgers[index] = item
	} else {
		if len(doc.Ledgers) >= 256 {
			doc.Ledgers = append(doc.Ledgers[1:], item)
		} else {
			doc.Ledgers = append(doc.Ledgers, item)
		}
	}
	sort.Slice(doc.Ledgers, func(i, j int) bool { return doc.Ledgers[i].Identity < doc.Ledgers[j].Identity })
	return s.writeLocked(doc)
}

// RecentLedgerPaths returns a copy of paths the user may explicitly reopen.
// Paths are stored only in this list; per-ledger preference records remain
// keyed by a one-way identity hash.
func (s *Store) RecentLedgerPaths() ([]string, error) {
	doc, err := s.Load()
	if err != nil {
		return nil, err
	}
	return append([]string(nil), doc.RecentLedgerPaths...), nil
}

// SourceEditorBundleID returns the optional application-wide editor selected
// for opening ledger source files. An empty value keeps the system default.
func (s *Store) SourceEditorBundleID() (string, error) {
	doc, err := s.Load()
	if err != nil {
		return "", err
	}
	return doc.SourceEditorBundleID, nil
}

// BackupSnapshot captures the current validated preference document without
// including query drafts, ledger contents, or rendered results.
func (s *Store) BackupSnapshot() (BackupSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.loadLocked()
	if err != nil {
		return BackupSnapshot{}, err
	}
	content, err := marshalDocument(doc)
	if err != nil {
		return BackupSnapshot{}, err
	}
	protectedPath, err := filepath.Abs(s.path)
	if err != nil {
		return BackupSnapshot{}, err
	}
	return BackupSnapshot{Content: content, ProtectedPaths: []string{filepath.Clean(protectedPath)}}, nil
}

// SetSourceEditorBundleID stores an application-wide editor choice without
// recording the selected application's filesystem path.
func (s *Store) SetSourceEditorBundleID(bundleID string) error {
	if bundleID != "" && platform.ValidateSourceEditorBundleID(bundleID) != nil {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.loadLocked()
	if err != nil {
		return err
	}
	doc.SourceEditorBundleID = bundleID
	return s.writeLocked(doc)
}

// RecordRecentLedger moves a successfully opened ledger to the front of the
// bounded recent list. A ledger's contents and query data are never stored.
func (s *Store) RecordRecentLedger(path string) error {
	canonical, err := canonicalLedgerPath(path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.loadLocked()
	if err != nil {
		return err
	}
	doc.RecentLedgerPaths = prependRecent(doc.RecentLedgerPaths, canonical)
	return s.writeLocked(doc)
}

// AttachmentRoots returns a deep copy of ordinary bookmark records for the
// ledger. Callers must resolve each bookmark before using its directory.
func (s *Store) AttachmentRoots(path string) ([]platform.DirectoryBookmark, uint64, error) {
	item, err := s.ForLedger(path)
	if err != nil {
		return nil, 0, err
	}
	return cloneBookmarks(item.AttachmentRoots), item.AttachmentRootsGeneration, nil
}

// UpdateAttachmentRoots stores a complete roots configuration after the
// caller has successfully applied it to the active Session. The generation is
// incremented atomically with the configuration and returned to the caller.
func (s *Store) UpdateAttachmentRoots(path string, roots []platform.DirectoryBookmark, origin string) (uint64, error) {
	identity, err := Identity(path)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	doc, err := s.loadLocked()
	if err != nil {
		s.mu.Unlock()
		return 0, err
	}
	index := ledgerPreferenceIndex(doc.Ledgers, identity)
	item := LedgerPreferences{Identity: identity, SemanticsProfile: "legacy", Locale: "zh-CN", TextScale: 1, Window: "all", Valuation: "at-cost", Columns: map[string][]string{}}
	if index >= 0 {
		item = cloneLedger(doc.Ledgers[index])
	}
	if item.AttachmentRootsGeneration == ^uint64(0) {
		s.mu.Unlock()
		return 0, ErrInvalid
	}
	item.AttachmentRootsGeneration++
	item.AttachmentRoots = cloneBookmarks(roots)
	if err := validateLedger(item); err != nil {
		s.mu.Unlock()
		return 0, err
	}
	if index >= 0 {
		doc.Ledgers[index] = item
	} else {
		doc.Ledgers = append(doc.Ledgers, item)
	}
	sort.Slice(doc.Ledgers, func(i, j int) bool { return doc.Ledgers[i].Identity < doc.Ledgers[j].Identity })
	if err := s.writeLocked(doc); err != nil {
		s.mu.Unlock()
		return 0, err
	}
	generation := item.AttachmentRootsGeneration
	s.mu.Unlock()
	s.publishAttachmentRoots(AttachmentRootsChange{Identity: identity, Origin: origin, Generation: generation, Roots: cloneBookmarks(roots)})
	return generation, nil
}

// SubscribeAttachmentRoots observes policy records written through this Store
// by another in-process window. The owner tag suppresses the originating
// window's own notification. The returned function removes the subscription.
func (s *Store) SubscribeAttachmentRoots(owner string, fn func(AttachmentRootsChange)) func() {
	if fn == nil {
		return func() {}
	}
	s.listenersMu.Lock()
	s.nextListener++
	id := s.nextListener
	s.listeners[id] = attachmentRootsListener{owner: owner, fn: fn}
	s.listenersMu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			s.listenersMu.Lock()
			delete(s.listeners, id)
			s.listenersMu.Unlock()
		})
	}
}

func (s *Store) publishAttachmentRoots(change AttachmentRootsChange) {
	s.listenersMu.Lock()
	listeners := make([]attachmentRootsListener, 0, len(s.listeners))
	for _, listener := range s.listeners {
		if listener.owner != change.Origin {
			listeners = append(listeners, listener)
		}
	}
	s.listenersMu.Unlock()
	for _, listener := range listeners {
		listener := listener
		event := change
		event.Roots = cloneBookmarks(change.Roots)
		go listener.fn(event)
	}
}

func (s *Store) loadLocked() (Document, error) {
	if s.path == "" {
		return Document{}, ErrPath
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Document{}, err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return Document{}, err
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Document{SchemaVersion: SchemaVersion, Ledgers: []LedgerPreferences{}}, nil
	}
	if err != nil {
		return Document{}, err
	}
	if len(data) > maxPreferencesBytes {
		return Document{}, ErrInvalid
	}
	if err := os.Chmod(s.path, 0o600); err != nil {
		return Document{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var doc Document
	if err := decoder.Decode(&doc); err != nil {
		return Document{}, fmt.Errorf("decode preferences: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return Document{}, fmt.Errorf("decode preferences: trailing data")
		}
		return Document{}, fmt.Errorf("decode preferences: %w", err)
	}
	if doc.SchemaVersion != SchemaVersion {
		return Document{}, ErrSchema
	}
	if doc.Ledgers == nil {
		doc.Ledgers = []LedgerPreferences{}
	}
	if doc.RecentLedgerPaths == nil {
		doc.RecentLedgerPaths = []string{}
	}
	if len(doc.Ledgers) > 256 {
		return Document{}, ErrInvalid
	}
	if doc.LegacyMigrationVersion < 0 || doc.LegacyMigrationVersion > CurrentLegacyMigrationVersion || len(doc.RecentLedgerPaths) > 5 {
		return Document{}, ErrInvalid
	}
	if doc.SourceEditorBundleID != "" && platform.ValidateSourceEditorBundleID(doc.SourceEditorBundleID) != nil {
		return Document{}, ErrInvalid
	}
	recent := make(map[string]bool, len(doc.RecentLedgerPaths))
	for _, path := range doc.RecentLedgerPaths {
		if path == "" || len(path) > 4096 || !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsRune(path, '\x00') || recent[path] {
			return Document{}, ErrInvalid
		}
		recent[path] = true
	}
	seen := make(map[string]bool, len(doc.Ledgers))
	for _, item := range doc.Ledgers {
		if err := validateLedger(item); err != nil || seen[item.Identity] {
			return Document{}, ErrInvalid
		}
		seen[item.Identity] = true
	}
	sort.Slice(doc.Ledgers, func(i, j int) bool { return doc.Ledgers[i].Identity < doc.Ledgers[j].Identity })
	return doc, nil
}

func (s *Store) writeLocked(doc Document) error {
	data, err := marshalDocument(doc)
	if err != nil {
		return err
	}
	return s.writeFile(s.path, data)
}

func marshalDocument(doc Document) ([]byte, error) {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func canonicalLedgerPath(path string) (string, error) {
	if _, err := Identity(path); err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func prependRecent(paths []string, path string) []string {
	out := make([]string, 0, min(5, len(paths)+1))
	out = append(out, path)
	for _, previous := range paths {
		if previous == path {
			continue
		}
		out = append(out, previous)
		if len(out) == 5 {
			break
		}
	}
	return out
}

func validateLedger(item LedgerPreferences) error {
	if len(item.Identity) != sha256.Size*2 {
		return ErrInvalid
	}
	if _, err := hex.DecodeString(item.Identity); err != nil {
		return ErrInvalid
	}
	if item.SemanticsProfile != "legacy" && item.SemanticsProfile != "v3" {
		return ErrInvalid
	}
	if item.Locale != "" && item.Locale != "zh-CN" && item.Locale != "en" {
		return ErrInvalid
	}
	if item.TextScale != 0 && item.TextScale != 1 && item.TextScale != 1.25 && item.TextScale != 1.5 {
		return ErrInvalid
	}
	if item.Window != "all" && item.Window != "1m" && item.Window != "1y" {
		return ErrInvalid
	}
	if item.Valuation != "at-cost" && item.Valuation != "market-value" && item.Valuation != "units" {
		return ErrInvalid
	}
	if !validCurrency(item.Currency) {
		return ErrInvalid
	}
	if !validValuation(item.Valuation) {
		return ErrInvalid
	}
	if _, _, err := validateDateRange(item.DateFrom, item.DateTo); err != nil {
		return err
	}
	if len(item.Columns) > 32 {
		return ErrInvalid
	}
	if len(item.ColumnWidths) > 32 {
		return ErrInvalid
	}
	if len(item.AttachmentRoots) > 32 {
		return ErrInvalid
	}
	seenRoots := map[string]bool{}
	for _, root := range item.AttachmentRoots {
		if root.Validate() != nil || seenRoots[root.Path] {
			return ErrInvalid
		}
		seenRoots[root.Path] = true
	}
	for table, columns := range item.Columns {
		if table == "" || len(table) > 64 || len(columns) > 128 {
			return ErrInvalid
		}
		seen := map[string]bool{}
		for _, column := range columns {
			if column == "" || len(column) > 64 || seen[column] {
				return ErrInvalid
			}
			seen[column] = true
		}
	}
	for table, widths := range item.ColumnWidths {
		if table == "" || len(table) > 64 || len(widths) > 128 {
			return ErrInvalid
		}
		for column, width := range widths {
			if column == "" || len(column) > 64 || math.IsNaN(float64(width)) || math.IsInf(float64(width), 0) || width < 16 || width > 10000 {
				return ErrInvalid
			}
		}
	}
	return nil
}

func validateDateRange(from, to string) (string, string, error) {
	for _, value := range []string{from, to} {
		if value == "" {
			continue
		}
		date, err := time.Parse(time.DateOnly, value)
		if err != nil || date.Format(time.DateOnly) != value {
			return "", "", ErrInvalidDate
		}
	}
	if from != "" && to != "" && from > to {
		return "", "", ErrInvalidDate
	}
	return from, to, nil
}

func cloneLedger(item LedgerPreferences) LedgerPreferences {
	if item.Columns != nil {
		columns := make(map[string][]string, len(item.Columns))
		for key, values := range item.Columns {
			columns[key] = append([]string(nil), values...)
		}
		item.Columns = columns
	}
	if item.ColumnWidths != nil {
		widths := make(map[string]map[string]float32, len(item.ColumnWidths))
		for table, values := range item.ColumnWidths {
			widths[table] = make(map[string]float32, len(values))
			for column, width := range values {
				widths[table][column] = width
			}
		}
		item.ColumnWidths = widths
	}
	item.AttachmentRoots = cloneBookmarks(item.AttachmentRoots)
	return item
}

func cloneBookmarks(roots []platform.DirectoryBookmark) []platform.DirectoryBookmark {
	if roots == nil {
		return nil
	}
	clone := make([]platform.DirectoryBookmark, len(roots))
	for i, root := range roots {
		root.Data = append([]byte(nil), root.Data...)
		clone[i] = root
	}
	return clone
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
	tmp, err := os.CreateTemp(dir, ".preferences-*.tmp")
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
