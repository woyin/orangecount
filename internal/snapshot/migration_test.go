// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package snapshot

import (
	"encoding/json"
	"orangecount/internal/source"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationUsesOneLoadedGraphAndRedactsByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.bean")
	data, err := os.ReadFile("../../testdata/fixtures/semantics/lifo-implicit.bean")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	graph, err := source.LoadGraph(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("invalid on disk\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result := CompareSemantics(graph, false)
	if !result.Legacy.Valid || !result.V3.Valid || result.Legacy.SnapshotID == result.V3.SnapshotID {
		t.Fatalf("did not compare loaded input: %+v", result)
	}
	if result.ChangeCounts["balances"] == 0 || result.ChangeCounts["lots"] == 0 || result.ChangeCounts["entries"] == 0 {
		t.Fatalf("missing dimensions: %+v", result.ChangeCounts)
	}
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), "Assets:Stock") || strings.Contains(string(encoded), path) || strings.Contains(string(encoded), "purchase") {
		t.Fatal("default comparison leaks private details")
	}
	detailed := CompareSemantics(graph, true)
	if len(detailed.Details) == 0 || detailed.InputIdentity != result.InputIdentity {
		t.Fatal("local details lost input identity")
	}
}
