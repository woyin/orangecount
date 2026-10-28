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
	"strings"
	"testing"
)

// fixture builds one repository-shaped tree so each gate can be exercised in
// both directions without touching the real repository.
type fixture struct {
	root       string
	manifest   string
	deviations string
	assets     string
	baselines  string
	sources    []string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	root := t.TempDir()
	f := fixture{
		root:       root,
		manifest:   filepath.Join(root, "manifest.md"),
		deviations: filepath.Join(root, "deviations.md"),
		assets:     filepath.Join(root, "assets"),
		baselines:  filepath.Join(root, "baselines"),
		sources:    []string{filepath.Join(root, "internal"), filepath.Join(root, "web", "src")},
	}
	for _, dir := range []string{f.assets, f.sources[0], f.sources[1]} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Gate 1 also verifies that the engine and locale matrix the manifest
	// declares actually exists in the harness. The fixture supplies a minimal
	// one so the positive path is exercised; a focused test removes a piece and
	// expects the failure.
	f.write(t, filepath.Join(root, "web", "playwright.config.mjs"),
		`const crossBrowserProjects = [["webkit", "WebKit"], ["firefox", "Firefox"]];`+` testMatch: ["**/visual/fava-zh-structure.spec.mjs"];`)
	f.write(t, filepath.Join(root, "web", "tests", "visual", "fava-zh-structure.spec.mjs"), "// fixture")
	return f
}

// The manifest states which engines and locales the matrix covers. Gate 1 must
// read the harness, because that prose drifted ahead of the config once already.
func TestGateOneRequiresDeclaredEngineAndLocaleCoverage(t *testing.T) {
	f := newFixture(t)
	os.Remove(filepath.Join(f.root, "web", "tests", "visual", "fava-zh-structure.spec.mjs"))
	failures := checkControlledDimensions(paths{root: f.root})
	if len(failures) == 0 {
		t.Fatal("a missing Chinese structural spec was accepted")
	}
	joined := failures[0].Reason
	if !strings.Contains(joined, "简体中文结构巡检") {
		t.Fatalf("unexpected reason: %s", joined)
	}

	g := newFixture(t)
	g.write(t, filepath.Join(g.root, "web", "playwright.config.mjs"), `testMatch: ["**/visual/fava-zh-structure.spec.mjs"];`)
	if failures := checkControlledDimensions(paths{root: g.root}); len(failures) != 2 {
		t.Fatalf("a config without webkit/firefox yielded %d failures: %+v", len(failures), failures)
	}
}

func (f fixture) write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (f fixture) run(t *testing.T, gate string) (int, report, string) {
	t.Helper()
	args := []string{
		"-root", f.root,
		"-manifest", f.manifest,
		"-deviations", f.deviations,
		"-assets", f.assets,
		"-baselines", f.baselines,
		"-sources", strings.Join(f.sources, ","),
	}
	if gate != "" {
		args = append(args, "-gate", gate)
	}
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	var result report
	if stdout.Len() > 0 {
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("unparseable JSON (%v): %s", err, stdout.String())
		}
	}
	return code, result, stderr.String()
}

// evidence fixture writes the files a manifest row cites, so the accepted
// manifest resolves against real artefacts instead of bare strings.
func (f fixture) evidence(t *testing.T) {
	t.Helper()
	f.write(t, filepath.Join(f.root, "docs", "fava-contract-map.md"), "# Contract\n\n## ledger_data (GET)\n\ntext\n\n## shell surface\n")
	f.write(t, filepath.Join(f.root, "docs", "evidence", "root.md"), "# root hardening\n")
	f.write(t, filepath.Join(f.root, "docs", "evidence", "shell.md"), "# shell hardening\n")
	f.write(t, filepath.Join(f.sources[1], "root.spec.mjs"), "test(\"root loads\", () => {});\n")
	f.write(t, filepath.Join(f.sources[1], "shell.spec.mjs"), "test(\"shell loads\", () => {});\n")
	// Gate 1 accepts a row only when the visual bar it is judged against is
	// registered, so the fixture carries the functional-parity deviation the
	// real repository uses.
	f.write(t, f.deviations, "# Deviations\n\n### FD-0011 — Functional parity\n\n| Field | Value |\n| --- | --- |\n| Status | approved |\n| Approved evidence | approved by product owner 2026-10-28 |\n")
	// Baselines use the reference capture layout so gate 5 can attribute them.
	for _, cell := range []string{"desktop-light", "desktop-dark"} {
		for _, name := range []string{"root", "shell"} {
			f.write(t, filepath.Join(f.baselines, cell, "visual", "fava-reference.spec.mjs", name+".png"), "png\n")
		}
	}
}

func (f fixture) acceptedManifest() string {
	return "# Manifest\n\n" +
		"| ID | Route | Evidence | Wave | Current status |\n" +
		"| --- | --- | --- | --- | --- |\n" +
		"| R-ROOT | `/` | contract:docs/fava-contract-map.md#ledger_data-get browser:web/src/root.spec.mjs#root+loads visual:baselines/desktop-light/visual/fava-reference.spec.mjs/root.png hardening:docs/evidence/root.md | 1 | accepted |\n" +
		"| G-SHELL | shell | contract:docs/fava-contract-map.md#shell-surface browser:web/src/shell.spec.mjs#shell+loads visual:baselines/desktop-dark/visual/fava-reference.spec.mjs/shell.png hardening:docs/evidence/shell.md | 1 | accepted |\n"
}

func TestGateOneRequiresAcceptedStatusAndFourLayers(t *testing.T) {
	f := newFixture(t)
	f.evidence(t)
	f.write(t, f.manifest, f.acceptedManifest())
	if code, result, stderr := f.run(t, "1"); code != 0 || !result.Passed {
		t.Fatalf("complete manifest failed: code=%d stderr=%s", code, stderr)
	}

	cases := []struct {
		name   string
		row    string
		reason string
	}{
		{"not-accepted", "| R-ROOT | `/` | contract:a browser:b visual:c hardening:d | 1 | gated |", "要求 accepted"},
		{"missing-visual", "| R-ROOT | `/` | contract:a browser:b hardening:d | 1 | accepted |", "visual"},
		{"missing-contract", "| R-ROOT | `/` | browser:b visual:c hardening:d | 1 | accepted |", "contract"},
		{"missing-browser", "| R-ROOT | `/` | contract:a visual:c hardening:d | 1 | accepted |", "browser"},
		{"missing-hardening", "| R-ROOT | `/` | contract:a browser:b visual:c | 1 | accepted |", "hardening"},
		{"fabricated-contract-path", "| R-ROOT | `/` | contract:docs/does-not-exist.md browser:web/src/root.spec.mjs#root+loads visual:baselines/desktop-light/visual/fava-reference.spec.mjs/root.png hardening:docs/evidence/root.md | 1 | accepted |", "不存在"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "manifest.md")
			f.write(t, path, "| ID | Route | Evidence | Wave | Current status |\n| --- | --- | --- | --- | --- |\n"+tc.row+"\n")
			fixture := f
			fixture.manifest = path
			code, result, _ := fixture.run(t, "1")
			if code != 1 || result.Passed {
				t.Fatalf("incomplete row accepted: code=%d", code)
			}
			if !strings.Contains(result.Checks[0].Failures[0].Reason, tc.reason) {
				t.Fatalf("missing reason %q in %+v", tc.reason, result.Checks[0].Failures)
			}
		})
	}

	t.Run("empty-manifest", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "manifest.md")
		f.write(t, path, "# no rows\n")
		fixture := f
		fixture.manifest = path
		if code, result, _ := fixture.run(t, "1"); code != 1 || result.Passed {
			t.Fatalf("manifest without routes accepted: code=%d", code)
		}
	})

	t.Run("missing-file", func(t *testing.T) {
		fixture := f
		fixture.manifest = filepath.Join(t.TempDir(), "absent.md")
		if code, result, _ := fixture.run(t, "1"); code != 1 || result.Passed {
			t.Fatalf("missing manifest accepted: code=%d", code)
		}
	})
}

// TestGateOneResolvesEveryReferenceOnDisk pins the anti-fabrication rule: a
// row that names a missing path, a missing Markdown anchor, a missing Playwright
// title, or a missing Go test must fail the specific layer.
func TestGateOneResolvesEveryReferenceOnDisk(t *testing.T) {
	f := newFixture(t)
	f.evidence(t)
	f.write(t, filepath.Join(f.sources[0], "evidence_test.go"), "package internal\n\nfunc TestRootContract(t *testing.T) {}\n")

	row := func(evidence string) string {
		return "| ID | Route | Evidence | Wave | Current status |\n| --- | --- | --- | --- | --- |\n| R-ROOT | `/` | " + evidence + " | 1 | accepted |\n"
	}
	bundled := "contract:docs/fava-contract-map.md#ledger_data-get visual:baselines/desktop-light/visual/fava-reference.spec.mjs/root.png"

	t.Run("go-test-name-resolves", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "manifest.md")
		f.write(t, path, row(bundled+" browser:web/src/root.spec.mjs#root+loads hardening:TestRootContract"))
		fixture := f
		fixture.manifest = path
		if code, result, stderr := fixture.run(t, "1"); code != 0 || !result.Passed {
			t.Fatalf("registered Go test rejected: code=%d stderr=%s", code, stderr)
		}
	})

	cases := []struct {
		name     string
		evidence string
		layer    string
		reason   string
	}{
		{"missing-path", bundled + " browser:web/src/root.spec.mjs#root+loads hardening:docs/evidence/absent.md", "hardening", "不存在"},
		{"missing-anchor", "contract:docs/fava-contract-map.md#absent-anchor visual:baselines/desktop-light/visual/fava-reference.spec.mjs/root.png browser:web/src/root.spec.mjs#root+loads hardening:docs/evidence/root.md", "contract", "标题锚点"},
		{"missing-playwright-title", "contract:docs/fava-contract-map.md#ledger_data-get visual:baselines/desktop-light/visual/fava-reference.spec.mjs/root.png browser:web/src/root.spec.mjs#absent+case hardening:docs/evidence/root.md", "browser", "找不到用例"},
		{"missing-go-test", bundled + " browser:web/src/root.spec.mjs#root+loads hardening:TestNotInTheRepository", "hardening", "找不到 Go 测试"},
		{"pending-layer", bundled + " browser:pending hardening:docs/evidence/root.md", "browser", "pending"},
		{"empty-reference", bundled + " browser: hardening:docs/evidence/root.md", "browser", "缺少 browser"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "manifest.md")
			f.write(t, path, row(tc.evidence))
			fixture := f
			fixture.manifest = path
			code, result, _ := fixture.run(t, "1")
			if code != 1 || result.Passed {
				t.Fatalf("unresolvable evidence accepted: code=%d", code)
			}
			if !strings.Contains(result.Checks[0].Failures[0].Reason, tc.layer) || !strings.Contains(result.Checks[0].Failures[0].Reason, tc.reason) {
				t.Fatalf("unexpected failure for %s: %+v", tc.name, result.Checks[0].Failures)
			}
		})
	}
}

func TestGateTwoFindsTransplantTokenInProductionSources(t *testing.T) {
	f := newFixture(t)
	f.write(t, filepath.Join(f.sources[0], "web", "assets_http.go"), "package web\n\nfunc uiAsset() string {\n\tif os.Getenv(\"ORANGECOUNT_TRANSPLANTED_UI\") == \"1\" {\n\t\treturn \"assets/transplanted/\"\n\t}\n\treturn \"assets/\"\n}\n")
	code, result, stderr := f.run(t, "2")
	if code != 1 || result.Passed {
		t.Fatalf("token reference accepted: code=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(result.Checks[0].Failures[0].Reason, "ORANGECOUNT_TRANSPLANTED_UI") {
		t.Fatalf("unexpected failure: %+v", result.Checks[0].Failures)
	}

	if err := os.Remove(filepath.Join(f.sources[0], "web", "assets_http.go")); err != nil {
		t.Fatal(err)
	}
	f.write(t, filepath.Join(f.sources[1], "app.ts"), "export const transplanted = true;\n")
	if code, result, _ := f.run(t, "2"); code != 0 || !result.Passed {
		t.Fatalf("clean sources failed: code=%d", code)
	}

	// Documentation is out of scope for this gate.
	f.write(t, filepath.Join(f.root, "docs", "note.md"), "ORANGECOUNT_TRANSPLANTED_UI is documented here.\n")
	if code, result, _ := f.run(t, "2"); code != 0 || !result.Passed {
		t.Fatalf("documentation was scanned: code=%d", code)
	}
}

func TestGateThreeRequiresLegacyAssetsRemoved(t *testing.T) {
	f := newFixture(t)
	for _, name := range []string{"app.js", "app.css", "index.html"} {
		f.write(t, filepath.Join(f.assets, name), "legacy\n")
	}
	code, result, _ := f.run(t, "3")
	if code != 1 || len(result.Checks[0].Failures) != 3 {
		t.Fatalf("legacy assets accepted: code=%d failures=%+v", code, result.Checks[0].Failures)
	}

	for _, name := range []string{"app.js", "app.css", "index.html"} {
		if err := os.Remove(filepath.Join(f.assets, name)); err != nil {
			t.Fatal(err)
		}
	}
	f.write(t, filepath.Join(f.assets, "transplanted", "app.js"), "bundle\n")
	if code, result, _ := f.run(t, "3"); code != 0 || !result.Passed {
		t.Fatalf("clean asset tree failed: code=%d", code)
	}

	f.write(t, filepath.Join(f.sources[0], "assets.go"), "var path = \"assets/app.js\"\n")
	code, result, _ = f.run(t, "3")
	if code != 1 || !strings.Contains(result.Checks[0].Failures[0].Reason, "assets/app.js") {
		t.Fatalf("dead asset reference accepted: code=%d failures=%+v", code, result.Checks[0].Failures)
	}
}

const cleanDeviations = "# Approved Fava deviations\n\n" +
	"### FD-0001 — Approved item\n\n" +
	"| Field | Value |\n| --- | --- |\n" +
	"| Status | approved |\n" +
	"| Approved evidence | approved by product owner 2026-08-08 |\n" +
	"\n### FD-0011 — Functional parity\n\n" +
	"| Field | Value |\n| --- | --- |\n" +
	"| Status | approved |\n" +
	"| Approved evidence | approved by product owner 2026-10-28 |\n" +
	"\n```\n" +
	"### FD-0001 — Short name\n| Status | proposed / approved / expired / removed |\n| Approved evidence | review reference and date |\n```\n"

func TestGateFourRejectsProposalsAndUndatedApprovals(t *testing.T) {
	f := newFixture(t)
	f.write(t, f.deviations, cleanDeviations)
	if code, result, stderr := f.run(t, "4"); code != 0 || !result.Passed {
		t.Fatalf("clean registry failed (documented template was misread): code=%d stderr=%s", code, stderr)
	}

	proposed := "# Registry\n\n### FD-0002 — Proposed item\n\n| Field | Value |\n| --- | --- |\n| Status | proposed |\n| Approved evidence | none yet |\n"
	f.write(t, f.deviations, proposed)
	code, result, _ := f.run(t, "4")
	if code != 1 || !strings.Contains(result.Checks[0].Failures[0].Reason, "proposed") {
		t.Fatalf("proposal accepted: code=%d failures=%+v", code, result.Checks[0].Failures)
	}

	undated := "# Registry\n\n### FD-0003 — Approved without date\n\n| Field | Value |\n| --- | --- |\n| Status | approved |\n| Approved evidence | approved under native-desktop-app-plan.md |\n"
	f.write(t, f.deviations, undated)
	code, result, _ = f.run(t, "4")
	if code != 1 || !strings.Contains(result.Checks[0].Failures[0].Reason, "批准日期") {
		t.Fatalf("undated approval accepted: code=%d failures=%+v", code, result.Checks[0].Failures)
	}

	missingEvidence := "# Registry\n\n### FD-0004 — Approved without evidence row\n\n| Field | Value |\n| --- | --- |\n| Status | approved |\n"
	f.write(t, f.deviations, missingEvidence)
	code, result, _ = f.run(t, "4")
	if code != 1 || !strings.Contains(result.Checks[0].Failures[0].Reason, "Approved evidence") {
		t.Fatalf("missing evidence row accepted: code=%d failures=%+v", code, result.Checks[0].Failures)
	}
}

const referenceLockJSON = `{
  "schema_version": 1,
  "image_id": "sha256:64152901c794d93da74b199b321740cc875962539872edf24d9a7c0760db85f2",
  "fava": {"version": "1.30.12", "commit": "aa7538e8971252c9efc52c8a516a3a77d604553f"}
}
`

// writeReferenceBaselines lays out captures the way Playwright's
// snapshotPathTemplate does: <cell>/visual/<spec>/<name>.png.
func (f fixture) writeReferenceBaselines(t *testing.T, spec string, names ...string) {
	t.Helper()
	for _, cell := range []string{"desktop-light", "desktop-dark", "narrow-light", "narrow-dark"} {
		for _, name := range names {
			f.write(t, filepath.Join(f.baselines, cell, "visual", spec, name+".png"), "png\n")
		}
	}
}

func TestGateFiveRequiresFourBaselineCells(t *testing.T) {
	f := newFixture(t)
	code, result, _ := f.run(t, "5")
	if code != 1 || !strings.Contains(result.Checks[0].Failures[0].Reason, "不存在") {
		t.Fatalf("missing baseline tree accepted: code=%d failures=%+v", code, result.Checks[0].Failures)
	}

	f.writeReferenceBaselines(t, "fava-reference.spec.mjs", "journal")
	f.write(t, filepath.Join(f.baselines, "environment-lock.json"), referenceLockJSON)
	if code, result, _ := f.run(t, "5"); code != 0 || !result.Passed {
		t.Fatalf("complete reference matrix failed: code=%d failures=%+v", code, result.Checks[0].Failures)
	}

	// One missing cell is a gap even when every other cell has baselines.
	partial := newFixture(t)
	partial.writeReferenceBaselines(t, "fava-reference.spec.mjs", "journal")
	partial.write(t, filepath.Join(partial.baselines, "environment-lock.json"), referenceLockJSON)
	if err := os.RemoveAll(filepath.Join(partial.baselines, "narrow-dark")); err != nil {
		t.Fatal(err)
	}
	code, result, _ = partial.run(t, "5")
	if code != 1 || !strings.Contains(result.Checks[0].Failures[0].Subject, "narrow-dark") {
		t.Fatalf("incomplete matrix accepted: code=%d failures=%+v", code, result.Checks[0].Failures)
	}
}

// Gate 5 must reject a tree that only compares OrangeCount against itself: a
// self-capture is not the Fava visual authority.
func TestGateFiveRejectsSelfCapturedBaselines(t *testing.T) {
	f := newFixture(t)
	f.writeReferenceBaselines(t, "fava-transplanted.spec.mjs", "journal")
	f.write(t, filepath.Join(f.baselines, "environment-lock.json"), referenceLockJSON)
	code, result, _ := f.run(t, "5")
	if code != 1 || result.Passed {
		t.Fatalf("self-captured baselines accepted: code=%d", code)
	}
	joined := ""
	for _, item := range result.Checks[0].Failures {
		joined += item.Reason + " "
	}
	// A mismatched producer segment now fails because its script cannot be
	// confirmed as an isolated reference capture, which is the same rejection
	// stated in terms the gate can check from disk.
	if !strings.Contains(joined, "采集脚本不存在，无法确认来源") {
		t.Fatalf("missing provenance failure: %s", joined)
	}
}

// A non-reference producer is only acceptable when the lock declares it and the
// script keeps the isolation guard; either half alone must fail the gate.
func TestGateFiveRequiresDeclaredGuardedSecondaryProducer(t *testing.T) {
	f := newFixture(t)
	f.writeReferenceBaselines(t, "capture-modals.mjs", "m-add")
	// The script keeps the guard, so this case isolates the undeclared-segment
	// rule rather than failing on the guard first.
	f.write(t, filepath.Join(f.root, "web", "scripts", "capture-modals.mjs"),
		"const safeReference = Boolean(process.env.FAVA_BASE_URL);")
	f.write(t, filepath.Join(f.baselines, "environment-lock.json"), referenceLockJSON)
	code, result, _ := f.run(t, "5")
	if code != 1 || result.Passed {
		t.Fatalf("an undeclared producer segment was accepted: code=%d", code)
	}
	joined := ""
	for _, item := range result.Checks[0].Failures {
		joined += item.Reason + " "
	}
	if !strings.Contains(joined, "未声明该生产者") {
		t.Fatalf("missing undeclared-producer failure: %s", joined)
	}
}

// Gate 5 must require the recorded reference environment; a tree without the
// lock cannot prove which Fava build produced the pixels.
func TestGateFiveRequiresEnvironmentLock(t *testing.T) {
	f := newFixture(t)
	f.writeReferenceBaselines(t, "fava-reference.spec.mjs", "journal")
	code, result, _ := f.run(t, "5")
	if code != 1 || !strings.Contains(result.Checks[0].Failures[0].Reason, "环境锁") {
		t.Fatalf("missing lock accepted: code=%d failures=%+v", code, result.Checks[0].Failures)
	}

	// A lock that names a different Fava build is also a failure.
	f.write(t, filepath.Join(f.baselines, "environment-lock.json"), strings.ReplaceAll(referenceLockJSON, "1.30.12", "1.30.11"))
	code, result, _ = f.run(t, "5")
	if code != 1 || !strings.Contains(result.Checks[0].Failures[0].Reason, "1.30.12") {
		t.Fatalf("wrong reference version accepted: code=%d failures=%+v", code, result.Checks[0].Failures)
	}
}

func TestUnknownGateAndCombinedRun(t *testing.T) {
	f := newFixture(t)
	if code, _, stderr := f.run(t, "9"); code != 2 || !strings.Contains(stderr, "未知 -gate") {
		t.Fatalf("unknown gate accepted: code=%d stderr=%s", code, stderr)
	}
	f.evidence(t)
	f.write(t, f.manifest, f.acceptedManifest())
	f.write(t, f.deviations, cleanDeviations)
	f.writeReferenceBaselines(t, "fava-reference.spec.mjs", "root", "shell")
	f.write(t, filepath.Join(f.baselines, "environment-lock.json"), referenceLockJSON)
	code, result, stderr := f.run(t, "")
	if code != 0 || !result.Passed || len(result.Checks) != 5 {
		t.Fatalf("clean tree failed: code=%d stderr=%s", code, stderr)
	}
}

// The real repository must currently fail, and the failure list must name the
// concrete blocker. This pins the expected unfinished state instead of
// demanding a green run.
// The committed repository must pass its cutover gates, and a copy without the
// registered functional-parity deviation must fail. The second half is the
// anti-tamper control: flipping manifest rows to "accepted" without approving a
// visual bar has to be rejected.
func TestCommittedRepositoryStateIsHonest(t *testing.T) {
	root := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(root, "docs", "fava-route-state-manifest.md")); err != nil {
		t.Skip("repository documents not present")
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("committed repository should pass the cutover gates: code=%d stderr=%s", code, stderr.String())
	}
	var result report
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("unparseable JSON: %v", err)
	}
	if !result.Passed {
		t.Fatal("cutover gates reported failure on a passing tree")
	}

	// Point -deviations at a stripped copy and confirm gate 1 rejects the same
	// rows: accepting a route without an approved visual bar must not pass.
	deviations := filepath.Join(root, "docs", "fava-approved-deviations.md")
	body, err := os.ReadFile(deviations)
	if err != nil {
		t.Skipf("read deviations: %v", err)
	}
	// Target FD-0011 specifically: its status line is the one directly above the
	// "all standard routes" scope row.
	stripped := strings.Replace(
		string(body),
		"| Status | approved |\n| Route and state | all standard routes",
		"| Status | proposed |\n| Route and state | all standard routes",
		1,
	)
	if stripped == string(body) {
		t.Fatal("FD-0011 entry not found to strip")
	}
	strippedPath := filepath.Join(t.TempDir(), "deviations.md")
	if err := os.WriteFile(strippedPath, []byte(stripped), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-root", root, "-deviations", strippedPath, "-gate", "1"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "FD-0011") {
		t.Fatalf("accepted rows without an approved visual bar were not rejected: code=%d stderr=%s", code, stderr.String())
	}
}

// A route row must cite its own capture. The gate used to accept any existing
// PNG, so a row could point at another route's screenshot and still pass.
func TestGateOneRejectsAnotherRoutesBaseline(t *testing.T) {
	for route, want := range map[string]string{
		"/":                          "root",
		"/income_statement":          "income-statement",
		"/holdings/by_cost_currency": "holdings-by-cost-currency",
		"/account/<name>":            "account-assets-3ainvestments-3afund",
		"/help/<slug>":               "help",
	} {
		if got := routeSlug(route); got != want {
			t.Errorf("routeSlug(%q)=%q want %q", route, got, want)
		}
	}
	wrong := []layerReference{{Layer: "visual", Reference: "testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/holdings.png"}}
	if reason := visualBaselineMatchesRoute("`/holdings/by_cost_currency`", wrong); reason == "" {
		t.Fatal("a row citing another route's baseline must be rejected")
	}
	matching := []layerReference{{Layer: "visual", Reference: "testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/holdings-by-cost-currency.png"}}
	if reason := visualBaselineMatchesRoute("`/holdings/by_cost_currency`", matching); reason != "" {
		t.Fatalf("the matching baseline was rejected: %s", reason)
	}
	// G/M/D rows describe a surface rather than a path, so the rule must not fire.
	if reason := visualBaselineMatchesRoute("Header, Sidebar, PageTitle, responsive menu", wrong); reason != "" {
		t.Fatalf("a surface row was wrongly judged: %s", reason)
	}
}
