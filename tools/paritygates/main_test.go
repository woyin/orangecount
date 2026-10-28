// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const loaderSHA = "91ec104a758402cbc2d3b0638fec47054954fc3fbe8411a53436bbfffd39bd82"

func baseInventory(caps ...map[string]any) map[string]any {
	list := make([]any, 0, len(caps))
	for _, cap := range caps {
		list = append(list, cap)
	}
	return map[string]any{
		"schema_version":          1,
		"reference":               map[string]any{"beancount": "3.2.3", "beanquery": "0.2.0"},
		"beanquery_source_sha256": map[string]any{"query_compile.py": loaderSHA},
		"beancount_loader_sha256": loaderSHA,
		"capabilities":            list,
	}
}

func verified(id string) map[string]any {
	return map[string]any{
		"id": id, "status": "verified", "kind": "column",
		"go_location": "internal/query/columns.go",
		"cases":       []any{"TestAccountsColumn"},
	}
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func runGate(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// Gate 1 must accept only complete evidence and must reject every partial
// status shape, so the gate cannot be turned green by relabelling.
func TestGateOneRequiresCompleteEvidence(t *testing.T) {
	dir := t.TempDir()
	inventory := filepath.Join(dir, "inventory.json")
	pass := baseInventory(
		verified("table.accounts.account"),
		map[string]any{
			"id": "option.plugin_processing", "status": "approved-excluded",
			"approved_by": "product owner", "approval_date": "2026-10-27",
			"reason": "用户插件 Python plugin 执行不在标准范围内",
		},
	)
	writeJSONFile(t, inventory, pass)
	code, stdout, stderr := runGate(t, "-inventory", inventory, "-expected-total", "2")
	if code != 0 {
		t.Fatalf("complete evidence failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	var report statusReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unparseable JSON: %v", err)
	}
	if report.Total != 2 || report.Verified != 1 || report.ApprovedExcluded != 1 || report.Failed != 0 {
		t.Fatalf("unexpected counts: %+v", report)
	}

	cases := []struct {
		name   string
		cap    map[string]any
		reason string
	}{
		{"verified-without-location", map[string]any{"id": "a", "status": "verified", "cases": []any{"T"}}, "go_location"},
		{"verified-without-cases", map[string]any{"id": "a", "status": "verified", "go_location": "x.go"}, "cases 或 go_case"},
		{"excluded-without-approver", map[string]any{"id": "a", "status": "approved-excluded", "approval_date": "2026-10-27", "reason": "plugin"}, "approved_by"},
		{"excluded-without-date", map[string]any{"id": "a", "status": "approved-excluded", "approved_by": "owner", "reason": "plugin"}, "approval_date"},
		{"excluded-with-bad-date", map[string]any{"id": "a", "status": "approved-excluded", "approved_by": "owner", "approval_date": "27/10/2026", "reason": "plugin"}, "YYYY-MM-DD"},
		{"excluded-without-reason", map[string]any{"id": "a", "status": "approved-excluded", "approved_by": "owner", "approval_date": "2026-10-27"}, "reason"},
		{"excluded-for-non-plugin", map[string]any{"id": "a", "status": "approved-excluded", "approved_by": "owner", "approval_date": "2026-10-27", "reason": "hard to implement"}, "插件或外部扩展"},
		{"pending", map[string]any{"id": "a", "status": "pending"}, "pending"},
		{"implemented", map[string]any{"id": "a", "status": "implemented"}, "implemented"},
		{"blocked", map[string]any{"id": "a", "status": "blocked"}, "blocked"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "inventory.json")
			writeJSONFile(t, path, baseInventory(tc.cap))
			code, stdout, stderr := runGate(t, "-inventory", path, "-expected-total", "1")
			if code != 1 {
				t.Fatalf("expected failure exit, got code=%d stdout=%s stderr=%s", code, stdout, stderr)
			}
			if !strings.Contains(stdout, tc.reason) {
				t.Fatalf("missing reason %q in %s", tc.reason, stdout)
			}
		})
	}
}

// Gate 1 also rejects an inventory whose item count changed, because the 366
// IDs are an external contract.
func TestGateOneRejectsChangedTotal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	writeJSONFile(t, path, baseInventory(verified("a")))
	code, stdout, _ := runGate(t, "-inventory", path, "-expected-total", "2")
	if code != 1 || !strings.Contains(stdout, "期望 2") {
		t.Fatalf("changed total accepted: code=%d stdout=%s", code, stdout)
	}
}

func TestGateTwoRequiresBothCaseIdentities(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.json")
	both := verified("a")
	both["upstream_case"] = "beanquery.query_compile_test.TestX"
	both["go_case"] = "TestX"
	writeJSONFile(t, path, baseInventory(both))
	if code, stdout, stderr := runGate(t, "-inventory", path, "-gate", "2"); code != 0 {
		t.Fatalf("complete cases failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}

	missing := verified("a")
	missing["upstream_case"] = "beanquery.query_compile_test.TestX"
	writeJSONFile(t, path, baseInventory(missing))
	code, stdout, _ := runGate(t, "-inventory", path, "-gate", "2")
	if code != 1 || !strings.Contains(stdout, "缺少 go_case") {
		t.Fatalf("missing go_case accepted: code=%d stdout=%s", code, stdout)
	}
}

// gate3Fixture builds a generative differential tool, a promotion directory and
// a matching fixture set so the positive case exercises the real checks.
func gate3Fixture(t *testing.T, seed, registeredSeed int, differential bool) string {
	t.Helper()
	dir := t.TempDir()
	tool := filepath.Join(dir, "gendiff", "main.go")
	if err := os.MkdirAll(filepath.Dir(tool), 0o755); err != nil {
		t.Fatal(err)
	}
	source := "package main\n\nimport \"math/rand\"\n\nfunc main() {\n\trng := rand.New(rand.NewSource(" + itoa(seed) + "))\n\t_ = rng\n}\n"
	if differential {
		source += "// oracle differential: compares each generated case\n"
	}
	if err := os.WriteFile(tool, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	goldens := filepath.Join(dir, "golden")
	fixtures := filepath.Join(dir, "fixtures")
	for _, name := range []string{goldens, fixtures} {
		if err := os.MkdirAll(name, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(goldens, "case-1.golden.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixtures, "case-1.bean"), []byte("2000-01-01 open Assets:Cash\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := map[string]any{
		"schema_version": 1,
		"gate3": map[string]any{
			"generative_differential": map[string]any{
				"path": tool, "generative": true, "differential": differential,
				"seed": registeredSeed, "markers": []any{"rand.NewSource"},
			},
			"promotion": map[string]any{"path": goldens, "fixtures": fixtures},
		},
	}
	configPath := filepath.Join(dir, "config.json")
	writeJSONFile(t, configPath, config)
	return configPath
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

func TestGateThreeChecksSeedAndDifferential(t *testing.T) {
	inventory := filepath.Join(t.TempDir(), "inventory.json")
	writeJSONFile(t, inventory, baseInventory(verified("a")))

	if code, stdout, stderr := runGate(t, "-inventory", inventory, "-config", gate3Fixture(t, 42, 42, true), "-gate", "3"); code != 0 {
		t.Fatalf("complete gate3 registration failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	code, stdout, _ := runGate(t, "-inventory", inventory, "-config", gate3Fixture(t, 42, 42, false), "-gate", "3")
	if code != 1 || !strings.Contains(stdout, "differential=false") {
		t.Fatalf("missing differential stage accepted: code=%d stdout=%s", code, stdout)
	}
	code, stdout, _ = runGate(t, "-inventory", inventory, "-config", gate3Fixture(t, 42, 1234, true), "-gate", "3")
	if code != 1 || !strings.Contains(stdout, "固定 seed 1234") {
		t.Fatalf("unrecorded seed accepted: code=%d stdout=%s", code, stdout)
	}
	code, stdout, _ = runGate(t, "-inventory", inventory, "-config", filepath.Join(t.TempDir(), "absent.json"), "-gate", "3")
	if code != 1 || !strings.Contains(stdout, "未找到登记文件") {
		t.Fatalf("missing config accepted: code=%d stdout=%s", code, stdout)
	}

	// A claimed pipeline reference must really mention the tool.
	wired := filepath.Join(t.TempDir(), "Makefile")
	if err := os.WriteFile(wired, []byte("genprediff:\n\tgo run ./tools/genprediff\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wireTree := t.TempDir()
	toolFile := filepath.Join(wireTree, "genprediff", "main.go")
	if err := os.MkdirAll(filepath.Dir(toolFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(toolFile, []byte("package main\n\nimport \"math/rand\"\n\nfunc main() { _ = rand.New(rand.NewSource(42)) }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wireGoldens := filepath.Join(wireTree, "golden")
	wireFixtures := filepath.Join(wireTree, "fixtures")
	for _, dir := range []string{wireGoldens, wireFixtures} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(wireGoldens, "case-1.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wireFixtures, "case-1.bean"), []byte("2000-01-01 open Assets:Cash\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wireConfig := func(token string) string {
		cfg := map[string]any{
			"schema_version": 1,
			"gate3": map[string]any{
				"generative_differential": map[string]any{
					"path": toolFile, "generative": true, "differential": true,
					"seed": 42, "referenced_in": wired, "reference_token": token,
				},
				"promotion": map[string]any{"path": wireGoldens, "fixtures": wireFixtures},
			},
		}
		path := filepath.Join(t.TempDir(), "config.json")
		writeJSONFile(t, path, cfg)
		return path
	}
	if code, stdout, stderr := runGate(t, "-inventory", inventory, "-config", wireConfig("genprediff"), "-gate", "3"); code != 0 {
		t.Fatalf("wired gate3 registration failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if code, stdout, _ := runGate(t, "-inventory", inventory, "-config", wireConfig("absent-target"), "-gate", "3"); code != 1 || !strings.Contains(stdout, "未被") {
		t.Fatalf("false wiring claim accepted: code=%d stdout=%s", code, stdout)
	}
}

func TestGateFourAndFiveRequireRegisteredEntryPoints(t *testing.T) {
	dir := t.TempDir()
	inventory := filepath.Join(dir, "inventory.json")
	writeJSONFile(t, inventory, baseInventory(verified("a")))
	entry := filepath.Join(dir, "capture.py")
	if err := os.WriteFile(entry, []byte("# read-only capture\nprint('x')\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	pipeline := filepath.Join(dir, "Makefile")
	if err := os.WriteFile(pipeline, []byte("run:\n\tpython capture.py\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, gate := range []string{"4", "5"} {
		key := "gate" + gate
		config := map[string]any{
			"schema_version": 1,
			key: map[string]any{
				"path": entry, "markers": []any{"read-only capture"},
				"referenced_in": pipeline, "reference_token": "capture.py",
			},
		}
		configPath := filepath.Join(dir, "config-"+gate+".json")
		writeJSONFile(t, configPath, config)
		if code, stdout, stderr := runGate(t, "-inventory", inventory, "-config", configPath, "-gate", gate); code != 0 {
			t.Fatalf("gate %s rejected a wired entry: code=%d stdout=%s stderr=%s", gate, code, stdout, stderr)
		}

		unwired := map[string]any{
			"schema_version": 1,
			key: map[string]any{
				"path": entry, "markers": []any{"read-only capture"},
				"referenced_in": pipeline, "reference_token": "other.py",
			},
		}
		writeJSONFile(t, configPath, unwired)
		code, stdout, _ := runGate(t, "-inventory", inventory, "-config", configPath, "-gate", gate)
		if code != 1 || !strings.Contains(stdout, "未被") {
			t.Fatalf("gate %s accepted an unwired entry: code=%d stdout=%s", gate, code, stdout)
		}

		missing := map[string]any{"schema_version": 1}
		writeJSONFile(t, configPath, missing)
		code, stdout, _ = runGate(t, "-inventory", inventory, "-config", configPath, "-gate", gate)
		if code != 1 || !strings.Contains(stdout, "未登记") {
			t.Fatalf("gate %s accepted a missing registration: code=%d stdout=%s", gate, code, stdout)
		}
	}
}

// A registered run record is the only accepted substitute for re-running the
// oracle. A missing, incomplete, failed, or falsified record must fail.
func TestGateRunRecordMustShowPositiveOutputs(t *testing.T) {
	dir := t.TempDir()
	inventory := filepath.Join(dir, "inventory.json")
	writeJSONFile(t, inventory, baseInventory(verified("a")))
	entry := filepath.Join(dir, "oracle.py")
	if err := os.WriteFile(entry, []byte("# prefix differential\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(dir, "run.json")
	config := filepath.Join(dir, "config.json")
	writeConfig := func() {
		writeJSONFile(t, config, map[string]any{
			"schema_version": 1,
			"gate4": map[string]any{
				"path": entry, "markers": []any{"prefix differential"},
				"run_record": record, "run_command": "oracle.py",
				"run_outputs": []any{"accepted_roundtrips"},
			},
		})
	}
	writeConfig()

	cases := []struct {
		name   string
		record any
		reason string
	}{
		{"positive", map[string]any{"command": "python oracle.py", "ran_at": "2026-10-27", "output": map[string]any{"accepted_roundtrips": 36}}, ""},
		{"zero-output", map[string]any{"command": "python oracle.py", "ran_at": "2026-10-27", "output": map[string]any{"accepted_roundtrips": 0}}, "不是正数"},
		{"missing-output", map[string]any{"command": "python oracle.py", "ran_at": "2026-10-27", "output": map[string]any{}}, "缺少输出"},
		{"wrong-command", map[string]any{"command": "echo done", "ran_at": "2026-10-27", "output": map[string]any{"accepted_roundtrips": 36}}, "不包含"},
		{"undated", map[string]any{"command": "python oracle.py", "output": map[string]any{"accepted_roundtrips": 36}}, "ran_at"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writeJSONFile(t, record, tc.record)
			code, stdout, _ := runGate(t, "-inventory", inventory, "-config", config, "-gate", "4")
			if tc.reason == "" {
				if code != 0 {
					t.Fatalf("complete run record rejected: code=%d stdout=%s", code, stdout)
				}
				return
			}
			if code != 1 || !strings.Contains(stdout, tc.reason) {
				t.Fatalf("bad run record accepted: code=%d stdout=%s", code, stdout)
			}
		})
	}

	if err := os.Remove(record); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := runGate(t, "-inventory", inventory, "-config", config, "-gate", "4")
	if code != 1 || !strings.Contains(stdout, "运行记录不可读") {
		t.Fatalf("missing run record accepted: code=%d stdout=%s", code, stdout)
	}
}

func TestGateSixRequiresIdentityFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.json")
	writeJSONFile(t, path, baseInventory(verified("a")))
	if code, stdout, stderr := runGate(t, "-inventory", path, "-gate", "6"); code != 0 {
		t.Fatalf("complete identity failed: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}

	cases := []struct {
		name   string
		mutate func(map[string]any)
		reason string
	}{
		{"no-reference", func(inv map[string]any) { delete(inv, "reference") }, "reference.beancount"},
		{"no-version", func(inv map[string]any) { inv["reference"] = map[string]any{"beancount": "latest"} }, "不是版本号"},
		{"no-source-sha", func(inv map[string]any) { delete(inv, "beanquery_source_sha256") }, "beanquery_source_sha256"},
		{"bad-source-sha", func(inv map[string]any) { inv["beanquery_source_sha256"] = map[string]any{"a.py": "xyz"} }, "不是 sha256"},
		{"no-loader-sha", func(inv map[string]any) { inv["beancount_loader_sha256"] = "" }, "beancount_loader_sha256"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inv := baseInventory(verified("a"))
			tc.mutate(inv)
			writeJSONFile(t, path, inv)
			code, stdout, _ := runGate(t, "-inventory", path, "-gate", "6")
			if code != 1 || !strings.Contains(stdout, tc.reason) {
				t.Fatalf("identity gap accepted: code=%d stdout=%s", code, stdout)
			}
		})
	}
}

func TestUnknownGateAndUnreadableInventory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.json")
	writeJSONFile(t, path, baseInventory(verified("a")))
	if code, _, stderr := runGate(t, "-inventory", path, "-gate", "9"); code != 2 || !strings.Contains(stderr, "未知 -gate") {
		t.Fatalf("unknown gate accepted: code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := runGate(t, "-inventory", filepath.Join(t.TempDir(), "absent.json")); code != 2 || !strings.Contains(stderr, "无法读取 inventory") {
		t.Fatalf("unreadable inventory accepted: code=%d stderr=%s", code, stderr)
	}
}

// The committed inventory must be reported coherently: 366 entries, a verdict
// that matches the observed failures, and one concrete reason per failure. This
// pins internal consistency rather than a specific colour, so the gate can turn
// green when the work is really done without the test having to change.
func TestCommittedInventoryReportsCoherently(t *testing.T) {
	inventory := filepath.Join("..", "..", "docs", "compatibility", "upstream-inventory.json")
	config := filepath.Join("..", "..", "docs", "compatibility", "inventory-gate-config.json")
	if _, err := os.Stat(inventory); err != nil {
		t.Skip("committed inventory not present")
	}
	code, stdout, stderr := runGate(t, "-inventory", inventory)
	if code != 0 && code != 1 {
		t.Fatalf("gate 1 must exit 0 or 1, got %d (stderr=%s)", code, stderr)
	}
	var report statusReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("unparseable JSON: %v", err)
	}
	if report.Total != 366 {
		t.Fatalf("committed inventory must hold 366 capabilities, got %d", report.Total)
	}
	if code == 0 {
		if len(report.Failures) != 0 || report.Verified != 366 {
			t.Fatalf("a passing gate must verify all 366 items: verified=%d failures=%d", report.Verified, len(report.Failures))
		}
		if !strings.Contains(stderr, "通过") {
			t.Fatalf("summary missing pass verdict: %s", stderr)
		}
	} else {
		if len(report.Failures) == 0 {
			t.Fatalf("a failing gate must list at least one reason")
		}
		if !strings.Contains(stderr, "未通过") {
			t.Fatalf("summary missing fail verdict: %s", stderr)
		}
	}
	// -gate all must not disagree with gate 1 about the inventory itself.
	combined, combinedOut, _ := runGate(t, "-inventory", inventory, "-config", config, "-gate", "all")
	if (code == 0) != (combined == 0) {
		t.Fatalf("gate 1 and -gate all disagree: gate1=%d all=%d (%s)", code, combined, strings.TrimSpace(combinedOut[:min(len(combinedOut), 120)]))
	}
}
