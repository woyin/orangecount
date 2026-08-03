// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package logging writes local structured JSON events with conservative
// redaction by default.
package logging

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Options struct {
	Sensitive bool
}

type Logger struct {
	mu        sync.Mutex
	out       io.Writer
	sensitive bool
}

func New(out io.Writer, options Options) *Logger {
	return &Logger{out: out, sensitive: options.Sensitive}
}

func (l *Logger) Event(name string, fields map[string]any) error {
	if l == nil || l.out == nil {
		return nil
	}
	record := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"event":     name,
		"sensitive": l.sensitive,
	}
	for key, value := range fields {
		if l.sensitive || allowedField(key) {
			record[key] = value
		} else {
			record[key] = "[redacted]"
		}
	}
	enc := json.NewEncoder(l.out)
	enc.SetEscapeHTML(false)
	l.mu.Lock()
	defer l.mu.Unlock()
	return enc.Encode(record)
}

func allowedField(key string) bool {
	key = strings.ToLower(key)
	if strings.Contains(key, "path") || strings.Contains(key, "account") || strings.Contains(key, "amount") || strings.Contains(key, "query") || strings.Contains(key, "metadata") || strings.Contains(key, "narration") || strings.Contains(key, "transaction") {
		return false
	}
	if strings.Contains(filepath.Base(key), "raw") || strings.Contains(filepath.Base(key), "text") {
		return false
	}
	switch key {
	case "version", "snapshot_id", "file_count", "duration_ms", "code", "line", "column", "error_type", "published", "valid", "reload", "port":
		return true
	default:
		return false
	}
}
