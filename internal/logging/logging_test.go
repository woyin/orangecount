// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoggerRedactsByDefaultAndAllowsSensitiveOptIn(t *testing.T) {
	var out bytes.Buffer
	logger := New(&out, Options{})
	if err := logger.Event("reload", map[string]any{"account": "Assets:Cash", "snapshot_id": "abc"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Assets:Cash") || !strings.Contains(out.String(), "[redacted]") {
		t.Fatalf("log=%q", out.String())
	}
	out.Reset()
	logger = New(&out, Options{Sensitive: true})
	if err := logger.Event("reload", map[string]any{"account": "Assets:Cash"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Assets:Cash") || !strings.Contains(out.String(), `"sensitive":true`) {
		t.Fatalf("sensitive log=%q", out.String())
	}
}
