// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package gen

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateIsDeterministic(t *testing.T) {
	first, second := filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "second")
	if _, err := Generate(first, DefaultConfig); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(second, DefaultConfig); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.bean", "accounts.bean", "activity.bean", "directives.bean", "generator-lock.json", "import/import-candidate.csv"} {
		left, err := os.ReadFile(filepath.Join(first, name))
		if err != nil {
			t.Fatal(err)
		}
		right, err := os.ReadFile(filepath.Join(second, name))
		if err != nil {
			t.Fatal(err)
		}
		if got, want := sha256.Sum256(left), sha256.Sum256(right); got != want {
			t.Fatalf("%s differs between deterministic runs", name)
		}
	}
}
