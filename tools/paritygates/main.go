// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Command paritygates turns the six superset release gates in
// docs/parity-gates-2026-10-27.md into an executable judgement. It never
// reads a hand-ticked checklist: gate 1 derives from the extracted upstream
// inventory, gates 2..6 derive from the inventory plus the registrations in
// docs/compatibility/inventory-gate-config.json.
//
// Machine-readable JSON goes to stdout; the Chinese summary goes to stderr.
// Any failing gate exits nonzero so CI cannot mistake the report for approval.
package main

import (
	"encoding/json"
	"errors"
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
	"time"
)

const (
	schemaVersion        = 1
	defaultExpectedTotal = 366
	maxEntryBytes        = 4 << 20
)

// pluginKeywords decide whether an approved exclusion reason points at a user
// plugin or external extension. Only such exclusions may be counted.
var pluginKeywords = []string{"plugin", "插件", "extension", "扩展", "importer", "导入器", "beangulp"}

var (
	datePattern    = regexp.MustCompile("^[0-9]{4}-[0-9]{2}-[0-9]{2}$")
	versionPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)+$`)
	sha256Pattern  = regexp.MustCompile("^[0-9a-f]{64}$")
)

type inventory struct {
	SchemaVersion         int               `json:"schema_version"`
	Reference             map[string]any    `json:"reference"`
	BeanquerySourceSHA256 map[string]string `json:"beanquery_source_sha256"`
	BeancountLoaderSHA256 string            `json:"beancount_loader_sha256"`
	Capabilities          []capability      `json:"capabilities"`
}

type capability struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Kind         string `json:"kind"`
	GoLocation   any    `json:"go_location"`
	Cases        any    `json:"cases"`
	GoCase       any    `json:"go_case"`
	UpstreamCase any    `json:"upstream_case"`
	ApprovedBy   any    `json:"approved_by"`
	ApprovalDate any    `json:"approval_date"`
	Reason       any    `json:"reason"`
	VerifiedAt   any    `json:"verified_at"`
	Evidence     any    `json:"evidence"`
}

type failure struct {
	ID     string `json:"id"`
	Status string `json:"status,omitempty"`
	Reason string `json:"reason"`
}

type statusReport struct {
	SchemaVersion    int       `json:"schema_version"`
	Total            int       `json:"total"`
	Verified         int       `json:"verified"`
	ApprovedExcluded int       `json:"approved_excluded"`
	Pending          int       `json:"pending"`
	Failed           int       `json:"failed"`
	Failures         []failure `json:"failures"`
}

type gateReport struct {
	SchemaVersion int       `json:"schema_version"`
	Gate          int       `json:"gate"`
	Total         int       `json:"total"`
	Passed        bool      `json:"passed"`
	Basis         string    `json:"basis"`
	Note          string    `json:"note,omitempty"`
	Failures      []failure `json:"failures"`
}

type allReport struct {
	SchemaVersion int          `json:"schema_version"`
	Passed        bool         `json:"passed"`
	Gates         []gateReport `json:"gates"`
}

// runRecord is one committed record of an oracle invocation.
type runRecord struct {
	Command string         `json:"command"`
	RanAt   string         `json:"ran_at"`
	Output  map[string]any `json:"output"`
}

type gateConfig struct {
	SchemaVersion int          `json:"schema_version"`
	Gate3         *gate3Config `json:"gate3"`
	Gate4         *entryConfig `json:"gate4"`
	Gate5         *entryConfig `json:"gate5"`
}

type gate3Config struct {
	GenerativeDifferential *entryConfig `json:"generative_differential"`
	Promotion              *entryConfig `json:"promotion"`
}

// entryConfig registers one real repository entry point. The gate verifies the
// file exists, still contains the recorded markers, and is wired into the
// recorded pipeline; a registration cannot claim evidence by itself.
type entryConfig struct {
	Path            string   `json:"path"`
	Description     string   `json:"description"`
	Generative      bool     `json:"generative"`
	Differential    bool     `json:"differential"`
	Seed            *int     `json:"seed"`
	Fixtures        string   `json:"fixtures"`
	Markers         []string `json:"markers"`
	ReferencedIn    string   `json:"referenced_in"`
	ReferenceToken  string   `json:"reference_token"`
	Evidence        string   `json:"evidence"`
	EvidenceMarkers []string `json:"evidence_markers"`
	// RunRecord points at a committed record of one real invocation. The gate
	// reads it, requires the recorded command, and requires the named numeric
	// outputs to be positive. It never re-runs the oracle itself.
	RunRecord  string   `json:"run_record"`
	RunCommand string   `json:"run_command"`
	RunOutputs []string `json:"run_outputs"`
	// EvidenceTests names Go tests that carry a clause of the gate which the
	// private capture cannot observe, such as a work-budget cancellation or a
	// failed reload keeping the previous snapshot. Naming the test makes the
	// clause checkable instead of letting the criterion quietly disappear.
	EvidenceTests []string `json:"evidence_tests"`
	RerunStatus   string   `json:"rerun_status"`
	// InapplicableClauses records a criterion whose subject was removed from the
	// product. Retiring the subject is a product decision, so the gate stays
	// closed until the product owner records who approved it and when; otherwise
	// deleting the subject would silently delete the criterion.
	InapplicableClauses []inapplicableClause `json:"inapplicable_clauses"`
	// EvidenceRunRecords names additional committed invocations that carry a
	// clause the private capture does not cover on its own, such as the native
	// query flow exercised through the CLI. Each one is checked like the primary
	// record: the command must match and every named output must be positive.
	EvidenceRunRecords []runRecordSpec `json:"evidence_run_records"`
}

type runRecordSpec struct {
	RunRecord  string   `json:"run_record"`
	RunCommand string   `json:"run_command"`
	RunOutputs []string `json:"run_outputs"`
}

type inapplicableClause struct {
	Clause       string `json:"clause"`
	Reason       string `json:"reason"`
	ApprovedBy   string `json:"approved_by"`
	ApprovalDate string `json:"approval_date"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run keeps flag handling, gate selection and exit codes testable without
// terminating the test process.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("paritygates", flag.ContinueOnError)
	flags.SetOutput(stderr)
	inventoryPath := flags.String("inventory", filepath.Join("docs", "compatibility", "upstream-inventory.json"), "extracted upstream inventory JSON")
	configPath := flags.String("config", filepath.Join("docs", "compatibility", "inventory-gate-config.json"), "gate registration JSON")
	gateFlag := flags.String("gate", "", "gate number 2..6, or all; empty means gate 1")
	expectedTotal := flags.Int("expected-total", defaultExpectedTotal, "expected inventory capability count (ID set is an external contract)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	// Registered config paths are repository-relative. Resolve them against the
	// repository the inventory lives in, after flag parsing, so the gate works
	// from any working directory.
	gateRoot = repoRootFor(*inventoryPath)

	inv, err := loadInventory(*inventoryPath)
	if err != nil {
		fmt.Fprintf(stderr, "paritygates: 无法读取 inventory：%v\n", err)
		return 2
	}

	switch strings.ToLower(strings.TrimSpace(*gateFlag)) {
	case "", "1":
		report := checkStatus(inv, *expectedTotal)
		if err := writeJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "paritygates: 无法输出 JSON：%v\n", err)
			return 2
		}
		printStatusSummary(stderr, report)
		// A structural failure (wrong total, duplicate ID) blocks the gate even
		// when every present item carries complete evidence.
		if report.Verified+report.ApprovedExcluded != report.Total || len(report.Failures) > 0 {
			return 1
		}
		return 0
	case "2", "3", "4", "5", "6":
		number, _ := strconv.Atoi(*gateFlag)
		report := checkGate(inv, number, *configPath)
		if err := writeJSON(stdout, report); err != nil {
			fmt.Fprintf(stderr, "paritygates: 无法输出 JSON：%v\n", err)
			return 2
		}
		printGateSummary(stderr, report)
		if !report.Passed {
			return 1
		}
		return 0
	case "all":
		combined := allReport{SchemaVersion: schemaVersion, Passed: true, Gates: []gateReport{}}
		for number := 2; number <= 6; number++ {
			report := checkGate(inv, number, *configPath)
			combined.Gates = append(combined.Gates, report)
			if !report.Passed {
				combined.Passed = false
			}
		}
		if err := writeJSON(stdout, combined); err != nil {
			fmt.Fprintf(stderr, "paritygates: 无法输出 JSON：%v\n", err)
			return 2
		}
		for _, report := range combined.Gates {
			printGateSummary(stderr, report)
		}
		if !combined.Passed {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stderr, "paritygates: 未知 -gate 值 %q（可用 1..6 或 all）\n", *gateFlag)
	return 2
}

func loadInventory(path string) (*inventory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var inv inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

func loadConfig(path string) (*gateConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("未找到登记文件 %s", path)
	}
	var cfg gateConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("登记文件 %s 不是合法 JSON：%v", path, err)
	}
	return &cfg, nil
}

func writeJSON(w io.Writer, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", encoded)
	return err
}

// checkStatus is gate 1: every capability is either verified with a Go
// location plus at least one case, or an approved exclusion that names the
// approving owner, the date, and the user-plugin/external-extension reason.
func checkStatus(inv *inventory, expectedTotal int) statusReport {
	report := statusReport{SchemaVersion: schemaVersion, Total: len(inv.Capabilities), Failures: []failure{}}
	if len(inv.Capabilities) != expectedTotal {
		report.Failures = append(report.Failures, failure{
			Reason: fmt.Sprintf("inventory 条目总数 %d，期望 %d（ID 集合是外部契约）", len(inv.Capabilities), expectedTotal),
		})
	}
	seen := map[string]bool{}
	for _, item := range inv.Capabilities {
		status := strings.TrimSpace(item.Status)
		reasons := []string{}
		switch status {
		case "verified":
			if text(item.GoLocation) == "" {
				reasons = append(reasons, "status=verified 但缺少非空 go_location")
			}
			if len(listText(item.Cases)) == 0 && text(item.GoCase) == "" {
				reasons = append(reasons, "status=verified 但缺少 cases 或 go_case")
			}
			// A verified claim needs the date it was verified. Without it an item
			// can be flipped to verified and never revisited, which is how four
			// capabilities once carried a status no one could date.
			if value := text(item.VerifiedAt); value == "" {
				reasons = append(reasons, "status=verified 但缺少 verified_at")
			} else if !validDate(value) {
				reasons = append(reasons, "verified_at 不是 YYYY-MM-DD 日期")
			}
		case "approved-excluded":
			if text(item.ApprovedBy) == "" {
				reasons = append(reasons, "status=approved-excluded 但缺少 approved_by")
			}
			if text(item.ApprovalDate) == "" {
				reasons = append(reasons, "status=approved-excluded 但缺少 approval_date")
			} else if !validDate(text(item.ApprovalDate)) {
				reasons = append(reasons, "approval_date 不是 YYYY-MM-DD 日期")
			}
			reason := text(item.Reason)
			if reason == "" {
				reasons = append(reasons, "status=approved-excluded 但缺少 reason")
			} else if !pointsAtPluginOrExtension(reason) {
				reasons = append(reasons, "reason 未指向用户插件或外部扩展")
			}
		case "pending":
			reasons = append(reasons, "status=pending：尚未验收")
		case "":
			reasons = append(reasons, "缺少 status")
		default:
			reasons = append(reasons, fmt.Sprintf("status=%s 不允许（只允许 verified 或 approved-excluded）", status))
		}
		if item.ID == "" {
			reasons = append(reasons, "缺少 id")
		} else if seen[item.ID] {
			reasons = append(reasons, "id 重复（ID 集合是外部契约）")
		}
		seen[item.ID] = true
		if len(reasons) == 0 {
			if status == "verified" {
				report.Verified++
			} else {
				report.ApprovedExcluded++
			}
			continue
		}
		if status == "pending" {
			report.Pending++
		}
		report.Failures = append(report.Failures, failure{ID: item.ID, Status: status, Reason: strings.Join(reasons, "；")})
	}
	report.Failed = report.Total - report.Verified - report.ApprovedExcluded - report.Pending
	return report
}

func validDate(value string) bool {
	if !datePattern.MatchString(value) {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func pointsAtPluginOrExtension(reason string) bool {
	lower := strings.ToLower(reason)
	for _, keyword := range pluginKeywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// checkGate dispatches gates 2..6.
func checkGate(inv *inventory, gate int, configPath string) gateReport {
	report := gateReport{SchemaVersion: schemaVersion, Gate: gate, Total: len(inv.Capabilities), Basis: "inventory", Failures: []failure{}}
	switch gate {
	case 2:
		report.Failures = checkCases(inv)
	case 3, 4, 5:
		cfg, err := loadConfig(configPath)
		if err != nil {
			report.Failures = append(report.Failures, failure{Status: "检查项", Reason: err.Error()})
			break
		}
		switch gate {
		case 3:
			report.Failures = checkGenerativeDifferential(cfg)
		case 4:
			report.Basis = "registration+recorded-run"
			report.Note = "校验登记入口、源码标记、管线引用，并读取已入库的真实运行记录（未包含输出判失败）；本工具本身不重跑前缀差分"
			report.Failures = checkRegisteredEntry("前缀库存差分入口", cfg.Gate4)
		case 5:
			report.Basis = "registration+recorded-run"
			report.Note = "校验登记入口、源码标记、管线引用与历史运行记录；本工具不重跑只读 capture"
			report.Failures = checkRegisteredEntry("只读 capture 入口", cfg.Gate5)
			if cfg.Gate5 != nil && cfg.Gate5.RerunStatus != "" {
				report.Note += "；" + cfg.Gate5.RerunStatus
			}
		}
	case 6:
		report.Failures = checkIdentity(inv)
	}
	report.Passed = len(report.Failures) == 0
	return report
}

// checkCases is gate 2: every capability names both the upstream test identity
// and the Go test identity that compares it.
func checkCases(inv *inventory) []failure {
	failures := []failure{}
	for _, item := range inv.Capabilities {
		reasons := []string{}
		if text(item.UpstreamCase) == "" {
			reasons = append(reasons, "缺少 upstream_case")
		}
		if text(item.GoCase) == "" {
			reasons = append(reasons, "缺少 go_case")
		}
		if len(reasons) == 0 {
			continue
		}
		failures = append(failures, failure{ID: item.ID, Status: strings.TrimSpace(item.Status), Reason: strings.Join(reasons, "；")})
	}
	return failures
}

// checkGenerativeDifferential is gate 3: a registered generative differential
// tool with a fixed seed, plus a registered failure-to-regression promotion.
func checkGenerativeDifferential(cfg *gateConfig) []failure {
	failures := []failure{}
	if cfg.Gate3 == nil {
		return []failure{{Status: "检查项", Reason: "登记文件缺少 gate3 段"}}
	}
	entry := cfg.Gate3.GenerativeDifferential
	if entry == nil || strings.TrimSpace(entry.Path) == "" {
		failures = append(failures, failure{Status: "检查项", Reason: "未登记生成式差分工具"})
	} else {
		source, err := readEntry(entry.Path)
		switch {
		case err != nil:
			failures = append(failures, failure{Status: "检查项", Reason: "登记的生成式差分工具不可读：" + err.Error()})
		default:
			if !entry.Generative {
				failures = append(failures, failure{Status: "检查项", Reason: "登记的工具不是生成式用例生成器（generative=false）"})
			}
			if !entry.Differential {
				failures = append(failures, failure{Status: "检查项", Reason: "登记的工具没有 oracle 差分阶段（differential=false）"})
			}
			if entry.Seed == nil {
				failures = append(failures, failure{Status: "检查项", Reason: "未登记固定 seed"})
			} else if *entry.Seed < 2 {
				failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("seed %d 不是可用的固定 seed", *entry.Seed)})
			} else if !containsSeed(source, *entry.Seed) {
				failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("固定 seed %d 未出现在工具源码中", *entry.Seed)})
			}
			failures = append(failures, missingMarkers(entry, source)...)
			// A registered pipeline reference must really mention the tool.
			if entry.ReferencedIn != "" {
				token := entry.ReferenceToken
				if token == "" {
					token = entry.Path
				}
				reference, err := readEntry(entry.ReferencedIn)
				switch {
				case err != nil:
					failures = append(failures, failure{Status: "检查项", Reason: "登记引用文件不可读：" + err.Error()})
				case !strings.Contains(reference, token):
					failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("%s 未被 %s 引用（缺少 %q）", entry.Path, entry.ReferencedIn, token)})
				}
			}
		}
	}
	promotion := cfg.Gate3.Promotion
	if promotion == nil || strings.TrimSpace(promotion.Path) == "" {
		failures = append(failures, failure{Status: "检查项", Reason: "未登记差异转固定回归机制"})
		return failures
	}
	info, err := os.Stat(resolveGatePath(promotion.Path))
	if err != nil {
		failures = append(failures, failure{Status: "检查项", Reason: "登记的回归目录不可读：" + err.Error()})
		return failures
	}
	if !info.IsDir() {
		failures = append(failures, failure{Status: "检查项", Reason: "登记的回归机制不是目录"})
		return failures
	}
	goldens := globNames(promotion.Path, ".json")
	if len(goldens) == 0 {
		failures = append(failures, failure{Status: "检查项", Reason: "登记目录中没有固定回归用例（.json）"})
	}
	if promotion.Fixtures != "" {
		fixtures := []string{}
		for _, dir := range splitList(promotion.Fixtures) {
			fixtures = append(fixtures, globNames(dir, ".bean")...)
		}
		if len(fixtures) == 0 {
			failures = append(failures, failure{Status: "检查项", Reason: "登记的回归输入目录中没有 .bean 用例"})
		} else if shared := sharedStems(goldens, fixtures); len(shared) == 0 {
			failures = append(failures, failure{Status: "检查项", Reason: "固定回归与输入用例没有同名配对"})
		}
	}
	if source, err := readEntry(promotion.Path); err == nil {
		failures = append(failures, missingMarkers(promotion, source)...)
	}
	return failures
}

// goTestIndex lists every Go test function declared under internal/ so a gate
// can prove that a named evidence test still exists instead of trusting a
// document to remember it.
func goTestIndex() (map[string]bool, error) {
	index := map[string]bool{}
	pattern := regexp.MustCompile(`^func (Test[A-Za-z0-9_]+)\s*\(`)
	// Resolve against the repository root, not the process working directory:
	// the test suite runs this from tools/paritygates, where ./internal does not
	// exist, and a cwd-relative walk silently reported "no tests".
	err := filepath.WalkDir(resolveGatePath("internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, line := range strings.Split(string(data), "\n") {
			if match := pattern.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
				index[match[1]] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return index, nil
}

func missingMarkers(entry *entryConfig, source string) []failure {
	failures := []failure{}
	for _, marker := range entry.Markers {
		if marker == "" {
			continue
		}
		if !strings.Contains(source, marker) {
			failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("%s 缺少标记 %q", entry.Path, marker)})
		}
	}
	return failures
}

// checkRegisteredEntry is gates 4 and 5: the registered entry point exists,
// still contains the recorded markers, and is referenced by the recorded
// pipeline.
func checkRegisteredEntry(label string, entry *entryConfig) []failure {
	if entry == nil || strings.TrimSpace(entry.Path) == "" {
		return []failure{{Status: "检查项", Reason: "未登记" + label}}
	}
	source, err := readEntry(entry.Path)
	if err != nil {
		return []failure{{Status: "检查项", Reason: label + "不可读：" + err.Error()}}
	}
	failures := missingMarkers(entry, source)
	if entry.Evidence != "" {
		record, err := readEntry(entry.Evidence)
		switch {
		case err != nil:
			failures = append(failures, failure{Status: "检查项", Reason: "登记的运行记录不可读：" + err.Error()})
		default:
			for _, marker := range entry.EvidenceMarkers {
				if marker != "" && !strings.Contains(record, marker) {
					failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("%s 未记录运行结果 %q", entry.Evidence, marker)})
				}
			}
		}
	}
	if entry.RunRecord != "" {
		failures = append(failures, checkRunRecord(entry)...)
	}
	for _, clause := range entry.InapplicableClauses {
		switch {
		case strings.TrimSpace(clause.ApprovedBy) == "" || strings.TrimSpace(clause.ApprovalDate) == "":
			failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("子句「%s」列为不适用（%s），但没有产品 owner 的 approved_by 与 approval_date", clause.Clause, clause.Reason)})
		case !validDate(clause.ApprovalDate):
			failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("子句「%s」的 approval_date 不是 YYYY-MM-DD 日期", clause.Clause)})
		}
	}
	for _, extra := range entry.EvidenceRunRecords {
		probe := &entryConfig{Path: entry.Path, RunRecord: extra.RunRecord, RunCommand: extra.RunCommand, RunOutputs: extra.RunOutputs}
		failures = append(failures, checkRunRecord(probe)...)
	}
	if len(entry.EvidenceTests) > 0 {
		index, err := goTestIndex()
		if err != nil {
			failures = append(failures, failure{Status: "检查项", Reason: "无法建立 Go 测试索引：" + err.Error()})
		} else {
			for _, name := range entry.EvidenceTests {
				if !index[name] {
					failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("登记的越限/取消/失败快照证据测试 %s 在仓库中不存在", name)})
				}
			}
		}
	}
	if entry.ReferencedIn != "" {
		token := entry.ReferenceToken
		if token == "" {
			token = entry.Path
		}
		reference, err := readEntry(entry.ReferencedIn)
		switch {
		case err != nil:
			failures = append(failures, failure{Status: "检查项", Reason: "登记引用文件不可读：" + err.Error()})
		case !strings.Contains(reference, token):
			failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("%s 未被 %s 引用（缺少 %q）", entry.Path, entry.ReferencedIn, token)})
		}
	}
	return failures
}

// checkRunRecord verifies one committed invocation record. A registration may
// not claim a passing run that leaves no artifact with the expected outputs.
func checkRunRecord(entry *entryConfig) []failure {
	data, err := os.ReadFile(resolveGatePath(entry.RunRecord))
	if err != nil {
		return []failure{{Status: "检查项", Reason: "登记的运行记录不可读：" + err.Error()}}
	}
	var record runRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return []failure{{Status: "检查项", Reason: "运行记录不是合法 JSON：" + err.Error()}}
	}
	failures := []failure{}
	if !validDate(record.RanAt) {
		failures = append(failures, failure{Status: "检查项", Reason: "运行记录缺少 YYYY-MM-DD 的 ran_at"})
	}
	if entry.RunCommand != "" && !strings.Contains(record.Command, entry.RunCommand) {
		failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("运行记录的命令 %q 不包含 %q", record.Command, entry.RunCommand)})
	}
	for _, key := range entry.RunOutputs {
		value, ok := record.Output[key]
		if !ok {
			failures = append(failures, failure{Status: "检查项", Reason: "运行记录缺少输出 " + key})
			continue
		}
		number, ok := value.(float64)
		if !ok || number <= 0 {
			failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("运行记录的 %s 不是正数：%v", key, value)})
		}
	}
	return failures
}

// checkIdentity is gate 6: the inventory records the pinned reference versions
// and the source digests that identify the audited upstream code.
func checkIdentity(inv *inventory) []failure {
	failures := []failure{}
	if inv.SchemaVersion != schemaVersion {
		failures = append(failures, failure{Status: "检查项", Reason: fmt.Sprintf("inventory schema_version=%d，期望 %d", inv.SchemaVersion, schemaVersion)})
	}
	for _, name := range []string{"beancount", "beanquery"} {
		value := text(inv.Reference[name])
		if value == "" {
			failures = append(failures, failure{Status: "检查项", Reason: "inventory 缺少 reference." + name})
		} else if !versionPattern.MatchString(value) {
			failures = append(failures, failure{Status: "检查项", Reason: "reference." + name + " 不是版本号：" + value})
		}
	}
	if len(inv.BeanquerySourceSHA256) == 0 {
		failures = append(failures, failure{Status: "检查项", Reason: "inventory 缺少 beanquery_source_sha256"})
	} else {
		names := make([]string, 0, len(inv.BeanquerySourceSHA256))
		for name := range inv.BeanquerySourceSHA256 {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if !sha256Pattern.MatchString(inv.BeanquerySourceSHA256[name]) {
				failures = append(failures, failure{Status: "检查项", Reason: "beanquery_source_sha256." + name + " 不是 sha256"})
				break
			}
		}
	}
	if !sha256Pattern.MatchString(inv.BeancountLoaderSHA256) {
		failures = append(failures, failure{Status: "检查项", Reason: "inventory 缺少合法的 beancount_loader_sha256"})
	}
	return failures
}

func containsSeed(source string, seed int) bool {
	pattern := regexp.MustCompile("(^|[^0-9])" + strconv.Itoa(seed) + "([^0-9]|$)")
	return pattern.MatchString(source)
}

// readEntry reads a registered file, or every source file under a registered
// directory, with a bounded size so a registration cannot hang the gate.
// gateRoot is the repository directory that registered config paths resolve
// against. It is derived from the inventory path so the gate is not tied to the
// process working directory.
var gateRoot = "."

// repoRootFor derives the repository root from an inventory path that lives at
// <root>/docs/compatibility/upstream-inventory.json.
func repoRootFor(inventoryPath string) string {
	slashed := filepath.ToSlash(inventoryPath)
	if index := strings.Index(slashed, "/docs/compatibility/"); index >= 0 {
		return filepath.FromSlash(slashed[:index])
	}
	if index := strings.Index(slashed, "docs/compatibility/"); index >= 0 {
		root := strings.TrimSuffix(slashed[:index], "/")
		if root == "" {
			return "."
		}
		return filepath.FromSlash(root)
	}
	return "."
}

// resolveGatePath maps a registered repository-relative path onto gateRoot.
func resolveGatePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(gateRoot, path)
}

func readEntry(path string) (string, error) {
	path = resolveGatePath(path)
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		if len(data) > maxEntryBytes {
			return "", errors.New("登记文件超过大小上限")
		}
		return string(data), nil
	}
	var builder strings.Builder
	err = filepath.WalkDir(path, func(name string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || builder.Len() > maxEntryBytes {
			return nil
		}
		switch filepath.Ext(name) {
		case ".go", ".py", ".sh", ".mjs", ".js", ".json", ".md":
			data, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			builder.Write(data)
			builder.WriteString("\n")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}

func globNames(dir, suffix string) []string {
	matches, err := filepath.Glob(filepath.Join(resolveGatePath(dir), "*"+suffix))
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		name := strings.TrimSuffix(filepath.Base(match), suffix)
		name = strings.TrimSuffix(name, ".golden")
		names = append(names, name)
	}
	return names
}

func splitList(value string) []string {
	parts := []string{}
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func sharedStems(left, right []string) []string {
	seen := map[string]bool{}
	for _, name := range left {
		seen[name] = true
	}
	shared := []string{}
	for _, name := range right {
		if seen[name] {
			shared = append(shared, name)
		}
	}
	sort.Strings(shared)
	return shared
}

func text(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if len(typed) == 0 {
			return ""
		}
		encoded, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(encoded)
	case []any:
		return strings.Join(listText(value), "; ")
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func listText(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := []string{}
		for _, item := range typed {
			if entry := text(item); entry != "" {
				out = append(out, entry)
			}
		}
		return out
	case []string:
		out := []string{}
		for _, item := range typed {
			if item = strings.TrimSpace(item); item != "" {
				out = append(out, item)
			}
		}
		return out
	case string:
		if typed = strings.TrimSpace(typed); typed != "" {
			return []string{typed}
		}
	}
	return nil
}

func printStatusSummary(w io.Writer, report statusReport) {
	verdict := "通过"
	if report.Total == 0 || report.Verified+report.ApprovedExcluded != report.Total || len(report.Failures) > 0 {
		verdict = "未通过"
	}
	fmt.Fprintf(w, "超集门槛 1（固定版无插件标准能力登记）：%s\n", verdict)
	fmt.Fprintf(w, "  条目总数 %d；verified %d；approved-excluded %d；pending %d；其他失败 %d\n",
		report.Total, report.Verified, report.ApprovedExcluded, report.Pending, report.Failed)
	printFailureGroups(w, "  失败明细", report.Failures)
}

func printGateSummary(w io.Writer, report gateReport) {
	verdict := "通过"
	if !report.Passed {
		verdict = "未通过"
	}
	fmt.Fprintf(w, "超集门槛 %d：%s（条目总数 %d，依据 %s）\n", report.Gate, verdict, report.Total, report.Basis)
	if report.Note != "" {
		fmt.Fprintf(w, "  说明：%s\n", report.Note)
	}
	printFailureGroups(w, "  失败明细", report.Failures)
}

// printFailureGroups lists the first 10 failures of each status class so the
// summary stays readable while the JSON keeps the full list.
func printFailureGroups(w io.Writer, label string, failures []failure) {
	if len(failures) == 0 {
		fmt.Fprintf(w, "%s：无\n", label)
		return
	}
	groups := map[string][]failure{}
	order := []string{}
	for _, item := range failures {
		key := item.Status
		if key == "" {
			key = "结构/身份"
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], item)
	}
	sort.Strings(order)
	fmt.Fprintf(w, "%s（共 %d 条）：\n", label, len(failures))
	for _, key := range order {
		list := groups[key]
		limit := len(list)
		if limit > 10 {
			limit = 10
		}
		fmt.Fprintf(w, "    [%s] %d 条，前 %d 条：\n", key, len(list), limit)
		for _, item := range list[:limit] {
			id := item.ID
			if id == "" {
				id = "(inventory)"
			}
			fmt.Fprintf(w, "      - %s：%s\n", id, item.Reason)
		}
	}
}
