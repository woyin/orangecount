// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package snapshot_test

import (
	"orangecount/internal/ledger"
	"orangecount/internal/snapshot"
	"os"
	"path/filepath"
	"testing"
)

func TestFailedBuildRetainsValidDocumentDeclarationsWithoutPublishingAccounts(t *testing.T) {
	for name, suffix := range map[string]string{
		"syntax":     "not a valid directive\n",
		"accounting": "2000-01-03 balance Assets:Cash 1 USD\n",
		"include":    "include \"missing.bean\"\n",
	} {
		t.Run(name, func(t *testing.T) {
			entry := filepath.Join(t.TempDir(), "main.bean")
			input := "2000-01-01 open Assets:Cash USD\n2000-01-02 document Assets:Cash \"receipt.csv\"\n" + suffix
			if err := os.WriteFile(entry, []byte(input), 0600); err != nil {
				t.Fatal(err)
			}
			built := snapshot.BuildWithOptions(entry, snapshot.BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
			if built.Snapshot != nil {
				t.Fatal("failed source published accounts")
			}
			declarations := built.DocumentDeclarations()
			realEntry, err := filepath.EvalSymlinks(entry)
			if err != nil {
				t.Fatal(err)
			}
			if len(declarations) != 1 || declarations[0].SourcePath != realEntry || declarations[0].Filename != "receipt.csv" {
				t.Fatalf("valid document declaration lost: %+v", declarations)
			}
			declarations[0].Filename = "changed"
			if built.DocumentDeclarations()[0].Filename != "receipt.csv" {
				t.Fatal("caller changed captured source evidence")
			}
		})
	}
}
