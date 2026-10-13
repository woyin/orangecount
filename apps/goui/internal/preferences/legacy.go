// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package preferences

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"orangecount/apps/goui/internal/platform"
)

const (
	LegacyApplicationDomain        = "org.orangecount.desktop"
	LegacyRecentLedgersKey         = "org.orangecount.recent_ledgers"
	LegacyAttachmentRootsKeyPrefix = "org.orangecount.attachmentRoots."
	CurrentLegacyMigrationVersion  = 2
)

var ErrLegacyUnsupported = errors.New("legacy macOS preferences are unavailable on this platform")

// LegacyDefaults reads only known scalar values from the old macOS defaults
// domain. Implementations must be read-only; migration never writes to it.
type LegacyDefaults interface {
	StringArray(key string) ([]string, bool, error)
	String(key string) (string, bool, error)
	StringDictionary(key string) (map[string]string, bool, error)
	Data(key string) ([]byte, bool, error)
}

// MigrateLegacy imports the old app's recent paths, per-ledger semantics
// profile, reading controls, and attachment bookmark records exactly once.
// Unknown preference keys are left untouched in the old domain.
func (s *Store) MigrateLegacy(defaults LegacyDefaults) error {
	if defaults == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, err := s.loadLocked()
	if err != nil {
		return err
	}
	if doc.LegacyMigrationVersion >= CurrentLegacyMigrationVersion {
		return nil
	}
	paths, _, err := defaults.StringArray(LegacyRecentLedgersKey)
	if err != nil {
		return err
	}
	cleaned := make([]string, 0, min(5, len(paths)))
	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		path, err := normalizeLegacyPath(path)
		if err != nil || seen[path] {
			continue
		}
		seen[path] = true
		cleaned = append(cleaned, path)
		if len(cleaned) == 5 {
			break
		}
	}
	for _, path := range cleaned {
		identity, err := Identity(path)
		if err != nil {
			continue // Keep a missing recent path visible, but do not invent a ledger identity.
		}
		index := ledgerPreferenceIndex(doc.Ledgers, identity)
		created := index < 0
		if index < 0 {
			doc.Ledgers = append(doc.Ledgers, LedgerPreferences{
				Identity: identity, SemanticsProfile: "legacy", Locale: "zh-CN", TextScale: 1, Window: "all", Valuation: "at-cost", Columns: map[string][]string{},
			})
			index = len(doc.Ledgers) - 1
		}
		item := &doc.Ledgers[index]
		if created {
			if profile, ok, readErr := defaults.String("org.orangecount.semantics." + path); readErr != nil {
				return readErr
			} else if ok && (profile == "legacy" || profile == "v3") {
				item.SemanticsProfile = profile
			}
			if saved, ok, readErr := defaults.StringDictionary("org.orangecount.reading." + path); readErr != nil {
				return readErr
			} else if ok {
				if currency := strings.TrimSpace(saved["currency"]); validCurrency(currency) {
					item.Currency = currency
				}
				if valuation := saved["valuation"]; validValuation(valuation) {
					item.Valuation = valuation
				}
			}
		}
		if item.AttachmentRoots == nil {
			if canonicalPath, pathErr := canonicalLedgerPath(path); pathErr == nil {
				data, ok, readErr := defaults.Data(LegacyAttachmentRootsKeyPrefix + canonicalPath)
				if readErr != nil {
					return readErr
				}
				if ok {
					config, decodeErr := decodeLegacyAttachmentRoots(data)
					if decodeErr != nil {
						return decodeErr
					}
					item.AttachmentRootsGeneration = config.Generation
					item.AttachmentRoots = config.Directories
				}
			}
		}
	}
	doc.RecentLedgerPaths = prependMigratedRecent(doc.RecentLedgerPaths, cleaned)
	doc.LegacyMigrationVersion = CurrentLegacyMigrationVersion
	return s.writeLocked(doc)
}

type legacyAttachmentRootConfiguration struct {
	Generation  uint64                       `json:"generation"`
	Directories []platform.DirectoryBookmark `json:"directories"`
}

func decodeLegacyAttachmentRoots(data []byte) (legacyAttachmentRootConfiguration, error) {
	if len(data) == 0 || len(data) > maxPreferencesBytes {
		return legacyAttachmentRootConfiguration{}, ErrInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var config legacyAttachmentRootConfiguration
	if err := decoder.Decode(&config); err != nil {
		return legacyAttachmentRootConfiguration{}, err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return legacyAttachmentRootConfiguration{}, ErrInvalid
		}
		return legacyAttachmentRootConfiguration{}, err
	}
	if config.Generation == 0 || config.Directories == nil || len(config.Directories) > 32 {
		return legacyAttachmentRootConfiguration{}, ErrInvalid
	}
	seen := make(map[string]bool, len(config.Directories))
	for _, directory := range config.Directories {
		if directory.Validate() != nil || seen[directory.Path] {
			return legacyAttachmentRootConfiguration{}, ErrInvalid
		}
		seen[directory.Path] = true
	}
	return config, nil
}

func normalizeLegacyPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || strings.ContainsRune(path, '\x00') {
		return "", ErrInvalid
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		suffix := strings.TrimPrefix(path, "~/")
		if path != "~" {
			path = filepath.Join(home, suffix)
		} else {
			path = home
		}
	}
	if !filepath.IsAbs(path) {
		return "", ErrInvalid
	}
	absolute := filepath.Clean(path)
	if len(absolute) > 4096 {
		return "", ErrInvalid
	}
	return absolute, nil
}

func ledgerPreferenceIndex(items []LedgerPreferences, identity string) int {
	for i := range items {
		if items[i].Identity == identity {
			return i
		}
	}
	return -1
}

func validCurrency(currency string) bool {
	return len(currency) <= 32 && !strings.ContainsAny(currency, "\r\n\t ")
}

func validValuation(valuation string) bool {
	return valuation == "at-cost" || valuation == "market-value" || valuation == "units"
}

func prependMigratedRecent(current, imported []string) []string {
	seen := make(map[string]bool, len(current)+len(imported))
	paths := make([]string, 0, min(5, len(current)+len(imported)))
	for _, list := range [][]string{current, imported} {
		for _, path := range list {
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			paths = append(paths, path)
			if len(paths) == 5 {
				return paths
			}
		}
	}
	return paths
}
