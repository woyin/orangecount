// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"orangecount/apps/goui/internal/preferences"
	"orangecount/apps/goui/internal/querypresets"
)

type legacyDefaultsOpener func(string) (preferences.LegacyDefaults, error)
type startupLogger func(string, ...any)

func configureIsolatedUserDataPath(setPath func(mygo.PathName, string), path string) (bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return false, nil
	}
	if setPath == nil || !filepath.IsAbs(path) {
		return false, errors.New("isolated user-data directory must be an absolute path")
	}
	path = filepath.Clean(path)
	if err := os.MkdirAll(path, 0o700); err != nil {
		return false, errors.New("cannot prepare isolated user-data directory")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return false, errors.New("isolated user-data directory must be a private directory")
	}
	setPath(mygo.PathUserData, path)
	return true, nil
}

func openApplicationStores(
	userData string,
	skipLegacyMigration bool,
	openLegacy legacyDefaultsOpener,
	logf startupLogger,
) (*preferences.Store, *querypresets.Store) {
	if userData == "" {
		return nil, nil
	}

	preferenceStore := preferences.NewStore(filepath.Join(userData, "preferences-v1.json"))
	queryPresetStore := querypresets.NewStore(filepath.Join(userData, "query-presets-v1.json"))
	if !skipLegacyMigration && openLegacy != nil {
		legacy, err := openLegacy(preferences.LegacyApplicationDomain)
		if err == nil {
			if err := preferenceStore.MigrateLegacy(legacy); err != nil {
				logStartup(logf, "cannot migrate recognized legacy preferences: %v", err)
			}
			if closer, ok := legacy.(interface{ Close() }); ok {
				closer.Close()
			}
		} else if !errors.Is(err, preferences.ErrLegacyUnsupported) {
			logStartup(logf, "cannot read legacy preferences: %v", err)
		}
	}
	if _, err := preferenceStore.Load(); err != nil {
		logStartup(logf, "cannot read app preferences: %v", err)
	}
	return preferenceStore, queryPresetStore
}

func logStartup(logf startupLogger, format string, args ...any) {
	if logf != nil {
		logf(format, args...)
	}
}
