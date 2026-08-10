// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGeneratesFixtureAndReportsFlagErrors(t *testing.T) {
	output := filepath.Join(t.TempDir(), "fixture")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--output", output}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "fixturegen: wrote") || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"--unknown"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
