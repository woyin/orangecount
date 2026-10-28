// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Command favagates turns the five Fava 1:1 cutover gates in
// docs/fava-cutover-gate-2026-10-27.md into an executable judgement. It reads
// the route manifest, the approved-deviation registry, the embedded asset
// namespace, the frontend sources and the visual baseline tree. It never reads
// a hand-ticked checklist.
//
// Machine-readable JSON goes to stdout; the Chinese summary goes to stderr.
// Any failing gate exits nonzero.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const schemaVersion = 1

// evidenceLayers are the four route-gate layers a manifest row must cite.
var evidenceLayers = []string{"contract", "browser", "visual", "hardening"}

// layerReference is one parsed "layer:reference" pair from a manifest row.
type layerReference struct {
	Layer     string
	Reference string
}

// routePrefixes are the accepted manifest identifiers.
var routePrefixes = []string{"R-", "G-", "M-", "D-"}

var (
	datePattern = regexp.MustCompile(`\b[0-9]{4}-[0-9]{2}-[0-9]{2}\b`)
	statusCell  = regexp.MustCompile(`^\s*\|\s*Status\s*\|`)
	// layerToken captures one "layer:reference" pair. A reference ends at
	// whitespace or Markdown/CJK punctuation so a table cell can hold four.
	layerToken = regexp.MustCompile(`\b(contract|browser|visual|hardening):([^\s|,;，；。()（）\[\]]+)`)
	goTestName = regexp.MustCompile(`^(Test|Benchmark|Example)[A-Za-z0-9_]+$`)
	anchorWord = regexp.MustCompile(`[a-z0-9_]+`)
)

type failure struct {
	Subject string `json:"subject"`
	Reason  string `json:"reason"`
}

type checkReport struct {
	Gate     int       `json:"gate"`
	Title    string    `json:"title"`
	Passed   bool      `json:"passed"`
	Note     string    `json:"note,omitempty"`
	Failures []failure `json:"failures"`
}

type report struct {
	SchemaVersion int           `json:"schema_version"`
	Passed        bool          `json:"passed"`
	Checks        []checkReport `json:"checks"`
}

// paths holds every location the gates read. Tests point these at fixtures so
// the checks can be exercised in both directions.
type paths struct {
	root       string
	manifest   string
	deviations string
	assets     string
	baselines  string
	sources    []string
	transplant string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("favagates", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	manifest := flags.String("manifest", filepath.Join("docs", "fava-route-state-manifest.md"), "route and state manifest")
	deviations := flags.String("deviations", filepath.Join("docs", "fava-approved-deviations.md"), "approved deviation registry")
	assets := flags.String("assets", filepath.Join("internal", "web", "assets"), "embedded asset directory")
	baselines := flags.String("baselines", filepath.Join("testdata", "visual-baselines"), "visual baseline directory")
	sources := flags.String("sources", "internal,cmd,web/src", "comma-separated source trees to scan")
	transplant := flags.String("token", "ORANGECOUNT_TRANSPLANTED_UI", "transplant switch environment variable")
	gate := flags.String("gate", "", "gate 1..5, or all; empty means all")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	p := paths{
		root:       *root,
		manifest:   resolve(*root, *manifest),
		deviations: resolve(*root, *deviations),
		assets:     resolve(*root, *assets),
		baselines:  resolve(*root, *baselines),
		transplant: *transplant,
	}
	for _, dir := range strings.Split(*sources, ",") {
		if dir = strings.TrimSpace(dir); dir != "" {
			p.sources = append(p.sources, resolve(*root, dir))
		}
	}

	selected := []int{1, 2, 3, 4, 5}
	if strings.TrimSpace(*gate) != "" && strings.ToLower(strings.TrimSpace(*gate)) != "all" {
		number, err := strconv.Atoi(strings.TrimSpace(*gate))
		if err != nil || number < 1 || number > 5 {
			fmt.Fprintf(stderr, "favagates: 未知 -gate 值 %q（可用 1..5 或 all）\n", *gate)
			return 2
		}
		selected = []int{number}
	}

	result := report{SchemaVersion: schemaVersion, Passed: true, Checks: []checkReport{}}
	for _, number := range selected {
		check := runCheck(number, p)
		result.Checks = append(result.Checks, check)
		if !check.Passed {
			result.Passed = false
		}
	}

	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "favagates: 无法输出 JSON：%v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "%s\n", encoded)
	for _, check := range result.Checks {
		printCheck(stderr, check)
	}
	if !result.Passed {
		return 1
	}
	return 0
}

func resolve(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func runCheck(number int, p paths) checkReport {
	check := checkReport{Gate: number, Failures: []failure{}}
	switch number {
	case 1:
		check.Title = "每条标准路由四层门禁通过"
		check.Failures = checkRouteManifest(p, goTestIndex(p.root))
		check.Failures = append(check.Failures, checkControlledDimensions(p)...)
	case 2:
		check.Title = "移植 UI 为唯一标准面"
		check.Failures = checkTransplantToken(p)
	case 3:
		check.Title = "legacy UI 与原型 shell 删除"
		check.Failures = checkLegacyAssets(p)
	case 4:
		check.Title = "未批准视觉差异清零"
		check.Failures = checkDeviations(p)
	case 5:
		check.Title = "视觉基线入库"
		check.Failures, check.Note = checkVisualBaselines(p)
	default:
		check.Title = "未知门禁"
		check.Failures = append(check.Failures, failure{Subject: "gate", Reason: "未实现的门禁编号"})
	}
	check.Passed = len(check.Failures) == 0
	return check
}

// visualBaselineMatchesRoute reports why a row's visual baseline does not
// belong to the route it cites, or "" when they correspond.
//
// The manifest path and the captured baseline name are two spellings of the
// same upstream route: `/holdings/by_cost_currency` is captured as
// `holdings-by-cost-currency.png`. A row that cites a different slug points at
// another route's pixels, so the layer would pass on a file that proves
// nothing about this route.
func visualBaselineMatchesRoute(pathCell string, references []layerReference) string {
	route := strings.Trim(strings.TrimSpace(pathCell), "`*_ ")
	if route == "" {
		return "路由路径为空，无法核对视觉基线归属"
	}
	// G/M/D rows describe a surface in prose ("Header, Sidebar, PageTitle") and
	// legitimately share a representative baseline, so only an actual route
	// path is judged. The caller filters on the R- prefix as well.
	if !strings.HasPrefix(route, "/") {
		return ""
	}
	want := routeSlug(route)
	for _, reference := range references {
		if reference.Layer != "visual" {
			continue
		}
		value := strings.Trim(strings.TrimSpace(reference.Reference), "*_`")
		if strings.HasPrefix(strings.ToLower(value), "pending") || !strings.HasSuffix(value, ".png") {
			return ""
		}
		slug := strings.TrimSuffix(filepath.Base(value), ".png")
		if want != "" && slug != want {
			return fmt.Sprintf("visual 层引用了别的路由的截图：%s 对应 %s.png，实际引用 %s.png", route, want, slug)
		}
	}
	return ""
}

// routeSlug maps a manifest route path to the baseline name the capture uses.
// The capture slug lowercases, turns every run of non-alphanumerics into one
// dash and trims the ends, which is why `/income_statement` becomes
// `income-statement` and `/holdings/by_cost_currency` becomes
// `holdings-by-cost-currency`.
func routeSlug(route string) string {
	cleaned := strings.Trim(route, "/")
	if cleaned == "" {
		return "root"
	}
	// A parameterised route is captured through one representative value.
	switch {
	case cleaned == "account/<name>":
		return "account-assets-3ainvestments-3afund"
	case strings.HasPrefix(cleaned, "help/"):
		return "help"
	}
	var b strings.Builder
	previousDash := false
	for _, r := range strings.ToLower(cleaned) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			previousDash = false
			continue
		}
		if !previousDash && b.Len() > 0 {
			b.WriteByte('-')
			previousDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// checkRouteManifest is gate 1. Every R-/G-/M-/D- row must be accepted and must
// cite all four evidence layers with a parseable reference.
func checkRouteManifest(p paths, tests map[string]bool) []failure {
	data, err := os.ReadFile(p.manifest)
	if err != nil {
		return []failure{{Subject: p.manifest, Reason: "无法读取 route manifest：" + err.Error()}}
	}
	failures := []failure{}
	// A row may only be accepted when the visual bar it is judged against is
	// itself registered. FD-0011 is the approved functional-parity deviation:
	// without it in the registry, "accepted" would silently mean pixel parity
	// that nobody measured.
	functionalParity := deviationApproved(p.deviations, "FD-0011")
	rows := 0
	for number, line := range strings.Split(string(data), "\n") {
		cells, ok := tableRow(line)
		if !ok || len(cells) < 2 {
			continue
		}
		id := cells[0]
		if !hasRoutePrefix(id) {
			continue
		}
		rows++
		reasons := []string{}
		status := strings.TrimSpace(cells[len(cells)-1])
		if status != "accepted" {
			reasons = append(reasons, fmt.Sprintf("状态为 %q，要求 accepted", status))
		} else if !functionalParity {
			reasons = append(reasons, "已标 accepted，但视觉基准未登记：需要 FD-0011（功能一致）在偏差表中为 approved")
		}
		// Every layer must cite a reference that resolves on disk. A declared
		// "pending" is an honest gap: it fails the layer instead of passing it.
		// Each row carries one evidence cell holding all four refs.
		references := parseLayerReferences(line)
		// A route row's visual baseline must name that route's own capture.
		// Checking only that some PNG exists let a row cite another route's
		// screenshot, which is evidence of nothing. The check is limited to R rows
		// whose second cell is a route path: G/M/D rows describe a surface
		// ("Header, Sidebar, PageTitle") and legitimately share a representative
		// baseline, so a path-to-slug rule would mis-fire on every one of them.
		if strings.HasPrefix(id, "R-") && strings.HasPrefix(strings.TrimSpace(cells[1]), "`/") {
			if reason := visualBaselineMatchesRoute(cells[1], references); reason != "" {
				reasons = append(reasons, reason)
			}
		}
		for _, layer := range evidenceLayers {
			found := false
			why := "未登记 " + layer + ": 标记"
			for _, reference := range references {
				if reference.Layer != layer {
					continue
				}
				found = true
				if reason := resolveReference(p.root, reference.Reference, tests); reason != "" {
					why = reason
					continue
				}
				why = ""
				break
			}
			if found && why == "" {
				continue
			}
			if !found {
				why = "缺少 " + layer + ": 标记"
			}
			reasons = append(reasons, layer+" 层不可用："+why)
		}
		if len(reasons) > 0 {
			failures = append(failures, failure{Subject: fmt.Sprintf("%s（第 %d 行）", id, number+1), Reason: strings.Join(reasons, "；")})
		}
	}
	if rows == 0 {
		failures = append(failures, failure{Subject: p.manifest, Reason: "manifest 中没有 R-/G-/M-/D- 路由行"})
	}
	return failures
}

// parseLayerReferences extracts every registered layer reference from a
// manifest row. Later duplicates of the same layer are kept so one honest
// reference can satisfy a layer that also lists a pending gap.
func parseLayerReferences(line string) []layerReference {
	references := []layerReference{}
	for _, match := range layerToken.FindAllStringSubmatch(line, -1) {
		references = append(references, layerReference{Layer: match[1], Reference: match[2]})
	}
	return references
}

// resolveReference reports why a reference is not usable evidence, or "" when
// it resolves. It only trusts artefacts that exist on disk: a path, a path with
// a verifiable fragment, a Playwright test title, or a Go test function name.
func resolveReference(root, raw string, tests map[string]bool) string {
	value := strings.Trim(strings.TrimSpace(raw), "*_`")
	if value == "" {
		return "证据引用为空"
	}
	if strings.HasPrefix(strings.ToLower(value), "pending") {
		// A note after "pending-" is kept in the reason so the operator sees
		// the specific gap instead of a generic pending marker.
		if note := strings.TrimPrefix(value[len("pending"):], "-"); note != "" {
			return "标记为 pending（证据尚未产出）：" + strings.ReplaceAll(note, "-", " ")
		}
		return "标记为 pending（证据尚未产出）"
	}
	if goTestName.MatchString(value) {
		if tests == nil {
			return "无法建立 Go 测试索引"
		}
		if tests[value] {
			return ""
		}
		return fmt.Sprintf("在 *_test.go 中找不到 Go 测试 %s", value)
	}
	path, fragment := value, ""
	if index := strings.Index(value, "#"); index >= 0 {
		path, fragment = value[:index], value[index+1:]
	}
	if strings.TrimSpace(path) == "" {
		return "证据引用缺少路径"
	}
	info, err := os.Stat(resolve(root, path))
	if err != nil {
		return fmt.Sprintf("路径 %s 不存在", path)
	}
	if fragment == "" {
		return ""
	}
	if info.IsDir() {
		return fmt.Sprintf("目录 %s 不能带片段", path)
	}
	data, err := os.ReadFile(resolve(root, path))
	if err != nil {
		return fmt.Sprintf("无法读取 %s", path)
	}
	switch filepath.Ext(path) {
	case ".mjs", ".js", ".ts", ".tsx":
		title := strings.ReplaceAll(fragment, "+", " ")
		for _, quote := range []string{"\"", "'", "`"} {
			if strings.Contains(string(data), "test("+quote+title+quote) {
				return ""
			}
		}
		return fmt.Sprintf("%s 中找不到用例 %q", path, title)
	case ".md":
		if !markdownAnchorExists(string(data), fragment) {
			return fmt.Sprintf("%s 中找不到标题锚点 #%s", path, fragment)
		}
		return ""
	}
	content := strings.ToLower(string(data))
	for _, word := range anchorWord.FindAllString(strings.ToLower(fragment), -1) {
		if !strings.Contains(content, word) {
			return fmt.Sprintf("%s 中找不到片段 %q", path, fragment)
		}
	}
	return ""
}

// markdownAnchorExists matches a fragment against the document headings after
// normalization, so a registered anchor must name a heading that is really
// present instead of any incidental word.
func markdownAnchorExists(document, fragment string) bool {
	want := normalizeAnchor(fragment)
	if want == "" {
		return false
	}
	for _, line := range strings.Split(document, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		if normalizeAnchor(strings.TrimLeft(trimmed, "# ")) == want {
			return true
		}
	}
	return false
}

// normalizeAnchor lowercases and collapses everything that is not a letter,
// digit, underscore or hyphen into single hyphen separators.
func normalizeAnchor(value string) string {
	lower := strings.ToLower(value)
	var builder strings.Builder
	lastSeparator := true
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			builder.WriteRune(r)
			lastSeparator = false
		default:
			if !lastSeparator {
				builder.WriteByte('-')
				lastSeparator = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

// goTestIndex collects every Go test function name once per run.
func goTestIndex(root string) map[string]bool {
	tests := map[string]bool{}
	_ = filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "dist", "staging", ".venv", "__pycache__", ".pi", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return nil
		}
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "func ") {
				continue
			}
			rest := strings.TrimPrefix(trimmed, "func ")
			if index := strings.Index(rest, "("); index > 0 {
				tests[rest[:index]] = true
			}
		}
		return nil
	})
	return tests
}

// tableRow splits one Markdown table row into trimmed cells. Header and
// separator rows are skipped by the caller's prefix test.
func tableRow(line string) ([]string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
		return nil, false
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(trimmed, "|"), "|"), "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells, true
}

// checkControlledDimensions verifies the browser engines and locale matrix the
// manifest claims exist. That claim is prose, so without a check it drifts ahead
// of the harness: an earlier revision stated WebKit, Firefox and a Simplified
// Chinese sweep as fact while the config produced Chromium cells only, and
// nothing failed.
func checkControlledDimensions(p paths) []failure {
	failures := []failure{}
	config := filepath.Join(p.root, "web", "playwright.config.mjs")
	data, err := os.ReadFile(config)
	if err != nil {
		return []failure{{Subject: relative(p.root, config), Reason: "无法读取 Playwright 配置：" + err.Error()}}
	}
	configText := string(data)
	for _, browser := range []string{"webkit", "firefox"} {
		if !strings.Contains(configText, "\""+browser+"\"") {
			failures = append(failures, failure{Subject: relative(p.root, config), Reason: fmt.Sprintf("manifest 声明了 %s 流，但配置没有声明该浏览器项目", browser)})
		}
	}
	spec := filepath.Join(p.root, "web", "tests", "visual", "fava-zh-structure.spec.mjs")
	if _, err := os.Stat(spec); err != nil {
		failures = append(failures, failure{Subject: relative(p.root, spec), Reason: "manifest 声明了简体中文结构巡检，但该 spec 不存在"})
	} else if !strings.Contains(configText, "fava-zh-structure.spec.mjs") {
		failures = append(failures, failure{Subject: relative(p.root, config), Reason: "简体中文结构 spec 存在，但没有被 testMatch 选中"})
	}
	return failures
}

func hasRoutePrefix(id string) bool {
	for _, prefix := range routePrefixes {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return false
}

// checkTransplantToken is gate 2. The transplant switch must be gone from all
// scanned source trees; documentation is out of scope.
func checkTransplantToken(p paths) []failure {
	failures := []failure{}
	for _, tree := range p.sources {
		info, err := os.Stat(tree)
		if err != nil {
			failures = append(failures, failure{Subject: tree, Reason: "源码目录不可读：" + err.Error()})
			continue
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(tree, func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" || entry.Name() == "dist" || entry.Name() == "staging" {
					return filepath.SkipDir
				}
				return nil
			}
			if !isSourceFile(name) {
				return nil
			}
			data, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			for index, line := range strings.Split(string(data), "\n") {
				if strings.Contains(line, p.transplant) {
					failures = append(failures, failure{
						Subject: fmt.Sprintf("%s:%d", relative(p.root, name), index+1),
						Reason:  fmt.Sprintf("仍引用 %s", p.transplant),
					})
				}
			}
			return nil
		})
		if err != nil {
			failures = append(failures, failure{Subject: tree, Reason: "扫描失败：" + err.Error()})
		}
	}
	if len(p.sources) == 0 {
		failures = append(failures, failure{Subject: "-sources", Reason: "未给出要扫描的源码目录"})
	}
	return failures
}

func isSourceFile(name string) bool {
	switch filepath.Ext(name) {
	case ".go", ".ts", ".tsx", ".js", ".mjs", ".svelte", ".css", ".html":
		return true
	}
	return false
}

// checkLegacyAssets is gate 3. The legacy checked-in bundle must be gone and no
// source may still reference it.
func checkLegacyAssets(p paths) []failure {
	failures := []failure{}
	for _, name := range []string{"app.js", "app.css", "index.html"} {
		path := filepath.Join(p.assets, name)
		if _, err := os.Stat(path); err == nil {
			failures = append(failures, failure{Subject: relative(p.root, path), Reason: "legacy 资产仍然存在"})
		} else if !os.IsNotExist(err) {
			failures = append(failures, failure{Subject: relative(p.root, path), Reason: "无法检查：" + err.Error()})
		}
	}
	// Only production sources can keep a dead asset reference alive. Test files
	// legitimately assert that the legacy assets are gone, so they are excluded
	// here; gate 2 still requires the transplant token to be absent everywhere.
	legacyReferences := []string{"assets/app.js", "assets/app.css", "assets/index.html"}
	for _, tree := range p.sources {
		if info, err := os.Stat(tree); err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(tree, func(name string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !isSourceFile(name) || strings.HasSuffix(name, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(name)
			if err != nil {
				return nil
			}
			for index, line := range strings.Split(string(data), "\n") {
				for _, reference := range legacyReferences {
					if strings.Contains(line, reference) {
						failures = append(failures, failure{
							Subject: fmt.Sprintf("%s:%d", relative(p.root, name), index+1),
							Reason:  "仍引用 legacy 资产 " + reference,
						})
					}
				}
			}
			return nil
		})
	}
	return failures
}

// deviation is one registry entry parsed from the Markdown document.
type deviation struct {
	ID       string
	Status   string
	Evidence string
}

// checkDeviations is gate 4. No proposal may remain, and every approved entry
// must record an approval date in its evidence field.
func checkDeviations(p paths) []failure {
	data, err := os.ReadFile(p.deviations)
	if err != nil {
		return []failure{{Subject: p.deviations, Reason: "无法读取偏差登记表：" + err.Error()}}
	}
	failures := []failure{}
	entries := []deviation{}
	current := -1
	fence := false
	for number, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			entries = append(entries, deviation{ID: strings.TrimPrefix(trimmed, "### ")})
			current = len(entries) - 1
			continue
		}
		if statusCell.MatchString(line) {
			cells, ok := tableRow(line)
			if ok && len(cells) >= 2 {
				value := strings.TrimSpace(cells[1])
				if strings.HasPrefix(strings.ToLower(value), "proposed") {
					failures = append(failures, failure{
						Subject: fmt.Sprintf("%s（第 %d 行）", deviationID(entries, current), number+1),
						Reason:  "仍为 proposed 状态，未获产品 owner 批准",
					})
				}
				if current >= 0 && entries[current].Status == "" {
					entries[current].Status = value
				}
			}
			continue
		}
		if current >= 0 && strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, "Approved evidence") {
			cells, ok := tableRow(line)
			if ok && len(cells) >= 2 && entries[current].Evidence == "" {
				entries[current].Evidence = strings.TrimSpace(cells[1])
			}
		}
	}
	for _, entry := range entries {
		if entry.Status != "approved" {
			continue
		}
		if entry.Evidence == "" {
			failures = append(failures, failure{Subject: entry.ID, Reason: "approved 条目缺少 Approved evidence 字段"})
			continue
		}
		if !datePattern.MatchString(entry.Evidence) {
			failures = append(failures, failure{Subject: entry.ID, Reason: "Approved evidence 中没有批准日期"})
		}
	}
	if len(entries) == 0 {
		failures = append(failures, failure{Subject: p.deviations, Reason: "登记表中没有 FD- 条目"})
	}
	return failures
}

// deviationApproved reports whether the registry holds an approved entry whose
// ID starts with the given prefix and carries a dated approval.
func deviationApproved(path, prefix string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	id := ""
	fence := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			id = strings.TrimPrefix(trimmed, "### ")
			continue
		}
		if id == "" || !strings.HasPrefix(id, prefix) {
			continue
		}
		if statusCell.MatchString(line) {
			cells, ok := tableRow(line)
			if ok && len(cells) >= 2 && strings.EqualFold(strings.TrimSpace(cells[1]), "approved") {
				return true
			}
		}
	}
	return false
}

func deviationID(entries []deviation, index int) string {
	if index < 0 || index >= len(entries) {
		return "(未归属条目)"
	}
	id := entries[index].ID
	if cut := strings.Index(id, " —"); cut > 0 {
		return id[:cut]
	}
	return id
}

// baselineLock is the recorded authority of a visual baseline tree.
type baselineLock struct {
	SchemaVersion int    `json:"schema_version"`
	ImageID       string `json:"image_id"`
	// Native captures cannot record an OCI image. They must instead pin the
	// interpreter environment by digest plus the exact dependency lock, which is
	// the same guarantee by a different route: the baseline names the precise
	// Fava build and its resolved dependencies.
	CaptureMethod     string `json:"capture_method"`
	EnvironmentDigest string `json:"environment_digest"`
	UVLockSHA256      string `json:"uv_lock_sha256"`
	// Producers records which capture script produced how many PNGs. Playwright
	// embeds the spec path in the tree, but the modal capture writes into the
	// reference spec's directory, so the directory alone cannot attribute it.
	// The lock states the split and the gate verifies the total, which turns an
	// unverifiable claim into a checked one.
	Producers map[string]int `json:"producers"`
	Fava      struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
	} `json:"fava"`
}

// referenceSpecSegment identifies captures produced by the isolated Fava
// reference suite. Playwright records the producing spec in the snapshot path,
// so a self-capture of OrangeCount can never masquerade as a Fava baseline.
const referenceSpecSegment = "fava-reference.spec.mjs"

// checkVisualBaselines is gate 5. The four English Chromium cells must exist,
// each must hold at least one PNG, and every PNG must come from the Fava
// reference capture with a recorded environment lock. A baseline that compares
// OrangeCount against itself proves nothing, so it is rejected.
func checkVisualBaselines(p paths) ([]failure, string) {
	cells := []string{"desktop-light", "desktop-dark", "narrow-light", "narrow-dark"}
	info, err := os.Stat(p.baselines)
	if err != nil {
		return []failure{{Subject: relative(p.root, p.baselines), Reason: "视觉基线目录不存在"}}, ""
	}
	if !info.IsDir() {
		return []failure{{Subject: relative(p.root, p.baselines), Reason: "视觉基线路径不是目录"}}, ""
	}
	failures := []failure{}
	// The note records what was observed even when the gate fails, so a reader
	// can see that the cell layout requirement is separate from provenance.
	counts := []string{}
	for _, cell := range cells {
		// Playwright writes snapshots under the project directory using the
		// spec path, so the cell must be searched recursively. A cell with one
		// PNG at any depth satisfies the existence requirement; approving the
		// pixels is a separate, human decision.
		count := 0
		err := filepath.WalkDir(filepath.Join(p.baselines, cell), func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(name), ".png") {
				count++
			}
			return nil
		})
		if err != nil {
			failures = append(failures, failure{Subject: cell, Reason: "无法读取该格的基线目录：" + err.Error()})
			continue
		}
		if count == 0 {
			failures = append(failures, failure{Subject: cell, Reason: "缺少该格的 .png 基线（目录内无 png）"})
		}
		counts = append(counts, fmt.Sprintf("%s=%d", cell, count))
	}
	// Provenance: the PNGs must be Fava reference captures, not OrangeCount
	// screenshots of itself. Playwright's snapshotPathTemplate embeds the
	// producing spec path, so this is checkable from the tree alone.
	producers := map[string]int{}
	_ = filepath.WalkDir(p.baselines, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(name), ".png") {
			return nil
		}
		segments := strings.Split(filepath.ToSlash(relative(p.root, name)), "/")
		producer := "(未知来源)"
		// A producer segment names the capture script: Playwright writes
		// <spec>.spec.mjs, and the standalone modal capture writes its own
		// <script>.mjs directory so its files are not attributed to the suite.
		for _, segment := range segments {
			if strings.HasSuffix(segment, ".mjs") {
				producer = segment
			}
		}
		producers[producer]++
		return nil
	})
	names := make([]string, 0, len(producers))
	for producer := range producers {
		names = append(names, producer)
	}
	sort.Strings(names)
	// A producer other than the reference suite is only acceptable when the
	// environment lock declares it and its script carries the same isolation
	// guard, because such a script screenshots a live server and could
	// otherwise capture OrangeCount itself and file it as a Fava reference.
	for _, producer := range names {
		if producer == referenceSpecSegment {
			continue
		}
		if reason := secondaryProducerReason(p.root, producer); reason != "" {
			failures = append(failures, failure{
				Subject: relative(p.root, p.baselines) + "/<cell>/visual/" + producer,
				Reason:  fmt.Sprintf("%d 张基线由 %s 生成，%s", producers[producer], producer, reason),
			})
		}
	}
	// The tree must also record the pinned reference environment.
	lockPath := filepath.Join(p.baselines, "environment-lock.json")
	note := fmt.Sprintf("四格 PNG 数：%s；来源：%s", strings.Join(counts, "、"), producerNote(producers))
	data, err := os.ReadFile(lockPath)
	if err != nil {
		failures = append(failures, failure{Subject: relative(p.root, lockPath), Reason: "缺少视觉基线环境锁（未记录 Fava 版本、commit 与镜像身份）"})
		return failures, note
	}
	var lock baselineLock
	if err := json.Unmarshal(data, &lock); err != nil {
		failures = append(failures, failure{Subject: relative(p.root, lockPath), Reason: "环境锁不是合法 JSON"})
		return failures, note
	}
	if lock.Fava.Version != "1.30.12" {
		failures = append(failures, failure{Subject: relative(p.root, lockPath), Reason: "环境锁的 Fava 版本不是 1.30.12"})
	}
	if lock.Fava.Commit != "aa7538e8971252c9efc52c8a516a3a77d604553f" {
		failures = append(failures, failure{Subject: relative(p.root, lockPath), Reason: "环境锁的 Fava commit 与固定参考不一致"})
	}
	if !lockPinsEnvironment(lock) {
		failures = append(failures, failure{Subject: relative(p.root, lockPath), Reason: "环境锁既没有 OCI 镜像身份，也没有可复现的原生环境（capture_method + environment_digest + uv_lock_sha256）"})
	}
	// The declared split is compared per producer, not by total. Summing let a
	// lock claim every baseline for one script and still pass, which made the
	// attribution unverifiable. Each producer's directory segment is checked on
	// its own, so a file in the wrong segment is caught.
	// A secondary producer must be declared even when the lock carries no
	// producers map at all, otherwise its files are silently credited to
	// whatever segment name they sit in.
	if len(lock.Producers) == 0 {
		for producer, count := range producers {
			if producer == referenceSpecSegment {
				continue
			}
			failures = append(failures, failure{
				Subject: relative(p.root, lockPath),
				Reason:  fmt.Sprintf("树里有 %d 张基线属于 %s，环境锁未声明该生产者", count, producer),
			})
		}
	}
	if len(lock.Producers) > 0 {
		for producer, count := range lock.Producers {
			if actual := producers[producer]; actual != count {
				failures = append(failures, failure{
					Subject: relative(p.root, lockPath),
					Reason:  fmt.Sprintf("环境锁声明 %s 生成 %d 张，树里该段实际有 %d 张", producer, count, actual),
				})
			}
		}
		for producer, count := range producers {
			if _, declared := lock.Producers[producer]; !declared {
				failures = append(failures, failure{
					Subject: relative(p.root, lockPath),
					Reason:  fmt.Sprintf("树里有 %d 张基线属于 %s，环境锁未声明该生产者", count, producer),
				})
			}
		}
		note = fmt.Sprintf("四格 PNG 数：%s；来源：%s", strings.Join(counts, "、"), producerNote(lock.Producers))
	}
	return failures, note
}

// lockPinsEnvironment reports whether the baseline records a reproducible
// reference environment. An OCI image id is the container route; a native
// capture instead names its environment digest and the exact uv lock, which
// resolves the same Fava version against the same dependency set.
func lockPinsEnvironment(lock baselineLock) bool {
	if strings.HasPrefix(lock.ImageID, "sha256:") && len(lock.ImageID) == len("sha256:")+64 {
		return true
	}
	if strings.TrimSpace(lock.CaptureMethod) == "" {
		return false
	}
	if !strings.HasPrefix(lock.EnvironmentDigest, "sha256:") || len(lock.EnvironmentDigest) != len("sha256:")+64 {
		return false
	}
	if len(lock.UVLockSHA256) != 64 {
		return false
	}
	for _, r := range lock.UVLockSHA256 {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

// producerNote summarizes which specs produced the baseline PNGs.
// secondaryProducerReason reports why a non-reference producer segment is not
// acceptable evidence, or "" when its script exists and keeps the isolation
// guard that stops it capturing OrangeCount's own server.
func secondaryProducerReason(root, producer string) string {
	if !strings.HasSuffix(producer, ".mjs") {
		return "不是可识别的采集脚本段（自我对照不能作为视觉权威）"
	}
	script := filepath.Join(root, "web", "scripts", producer)
	data, err := os.ReadFile(script)
	if err != nil {
		return "采集脚本不存在，无法确认来源"
	}
	if !strings.Contains(string(data), "safeReference") {
		return "采集脚本缺少 safeReference 隔离护栏，可能把 OrangeCount 自身当成参考"
	}
	return ""
}

func producerNote(producers map[string]int) string {
	names := make([]string, 0, len(producers))
	for producer := range producers {
		names = append(names, producer)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s×%d", name, producers[name]))
	}
	if len(parts) == 0 {
		return "无 PNG"
	}
	return strings.Join(parts, "、")
}

func relative(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

func printCheck(w io.Writer, check checkReport) {
	verdict := "通过"
	if !check.Passed {
		verdict = "未通过"
	}
	fmt.Fprintf(w, "Fava 切换门槛 %d（%s）：%s\n", check.Gate, check.Title, verdict)
	if check.Note != "" {
		fmt.Fprintf(w, "  观测：%s\n", check.Note)
	}
	if len(check.Failures) == 0 {
		fmt.Fprintf(w, "  失败明细：无\n")
		return
	}
	limit := len(check.Failures)
	if limit > 10 {
		limit = 10
	}
	fmt.Fprintf(w, "  失败明细（共 %d 条，前 %d 条）：\n", len(check.Failures), limit)
	for _, item := range check.Failures[:limit] {
		fmt.Fprintf(w, "    - %s：%s\n", item.Subject, item.Reason)
	}
}
