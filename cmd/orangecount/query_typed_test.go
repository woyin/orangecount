// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"orangecount/internal/query"
)

func TestRunQueryTypedJSONPreservesColumnPositions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQuery([]string{"--format", "typed-json", "--semantics", "v3", "../../testdata/fixtures/query/default-source.bean", "SELECT 1 AS v, 2 AS v, number = 0 AS zero, '' AS blank LIMIT 1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	var result query.TypedResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Contract != "beanquery-0.2.0-typed-v1" || len(result.Columns) != 4 || result.Columns[0].ID == result.Columns[1].ID || result.Rows[0][0].Value != "1" || result.Rows[0][1].Value != "2" || result.Rows[0][2].Value != false || result.Rows[0][3].Value != "" {
		t.Fatalf("result=%s", stdout.String())
	}
}
