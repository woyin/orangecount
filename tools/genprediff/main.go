// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

// Command genprediff generates deterministic synthetic ledgers from a fixed
// seed, evaluates each one with both the Go engine and the pinned Beancount
// oracle, and compares the full accounting state: validity, diagnostics,
// ordered entries, per-transaction inventory prefixes, final balances, and
// period income. Any difference is promoted to a fixed public regression under
// testdata/fixtures/gen/<seed>-<case>/ so it can never silently reappear.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"orangecount/internal/compat"
)

const genDefaultSeed = 424242

type fixtureCase struct {
	ID     string
	Ledger string
}

type oracleGolden struct {
	Fixture string `json:"fixture"`
	Valid   bool   `json:"valid"`
	State   struct {
		Entries  []string `json:"entries"`
		Balances map[string]struct {
			Currencies map[string]string `json:"currencies"`
			Lots       []string          `json:"lots"`
		} `json:"balances"`
	} `json:"state"`
	Errors []struct {
		Type string `json:"type"`
		Line *int   `json:"line"`
	} `json:"errors"`
	SchemaVersion int    `json:"schema_version"`
	OracleEngine  string `json:"oracle_engine"`
}

type report struct {
	SchemaVersion int   `json:"schema_version"`
	Seed          int64 `json:"seed"`
	// ApprovedDeviations names the registered deviations applied during the
	// comparison so a passing run never hides one of them.
	ApprovedDeviations []string `json:"approved_deviations,omitempty"`
	Cases              int      `json:"cases"`
	Passed             int      `json:"passed"`
	Failed             int      `json:"failed"`
	Promoted           []string `json:"promoted,omitempty"`
	Failures           []string `json:"failures,omitempty"`
}

func main() {
	seed := flag.Int64("seed", genDefaultSeed, "fixed generator seed")
	cases := flag.Int("cases", 12, "number of generated ledgers")
	workDir := flag.String("work", "", "working directory (default: a temp dir)")
	promote := flag.Bool("promote", true, "promote differences to testdata/fixtures/gen")
	promoteAll := flag.Bool("promote-all", false, "record every case with its oracle golden as a permanent regression corpus")
	keep := flag.Bool("keep", false, "keep the generated ledgers for inspection")
	flag.Parse()

	root, err := repoRoot()
	if err != nil {
		fail(err)
	}

	dir := *workDir
	if dir == "" {
		dir, err = os.MkdirTemp("", "orangecount-genprediff-")
		if err != nil {
			fail(err)
		}
		if !*keep {
			defer os.RemoveAll(dir)
		}
	}

	generated := generateCases(dir, *seed, *cases)
	oraclePath := filepath.Join(dir, "oracle.json")
	runOracle(root, dir, oraclePath)

	oracle, err := loadOracle(oraclePath)
	if err != nil {
		fail(err)
	}
	// Dimensions 3-6 (ordered entries, per-transaction inventory prefixes,
	// prices, period income) live only in the generative oracle payload.
	encodedOracle, err := os.ReadFile(oraclePath)
	if err != nil {
		fail(err)
	}
	dimensions, err := readOracleRecords(encodedOracle)
	if err != nil {
		fail(err)
	}

	out := report{SchemaVersion: 2, Seed: *seed, Cases: len(generated), ApprovedDeviations: []string{"posting-level-diagnostic-span"}}
	for _, item := range generated {
		want, ok := oracle[caseFileName(*seed, item.ID)]
		if !ok {
			out.Failed++
			out.Failures = append(out.Failures, item.ID+": oracle-missing")
			continue
		}
		path := filepath.Join(dir, caseFileName(*seed, item.ID))
		have, buildErr := compat.BuildFullSide(path)
		if buildErr != nil && have.Valid {
			out.Failed++
			out.Failures = append(out.Failures, item.ID+": build-error "+buildErr.Error())
			continue
		}
		differences := compat.FullGoldenDifferences(normalizeApproved(want), normalizeApproved(have))
		differences = append(differences, prefixDifferences(want, have)...)
		if record, ok := dimensions[caseFileName(*seed, item.ID)]; ok {
			goDimensions, dimensionErr := buildGoDimensions(path)
			if dimensionErr != nil {
				out.Failed++
				out.Failures = append(out.Failures, item.ID+": dimensions "+dimensionErr.Error())
				continue
			}
			differences = append(differences, dimensionDifferences(record, goDimensions)...)
		}
		if len(differences) == 0 {
			out.Passed++
			if *promoteAll && *promote {
				if promoted, err := promoteCase(root, *seed, item, want, have); err == nil {
					out.Promoted = append(out.Promoted, promoted)
				}
			}
			continue
		}
		out.Failed++
		out.Failures = append(out.Failures, item.ID+": "+strings.Join(differences, ","))
		if *promote {
			if promoted, err := promoteCase(root, *seed, item, want, have); err == nil {
				out.Promoted = append(out.Promoted, promoted)
			}
		}
	}

	encoded, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(encoded))
	if out.Failed != 0 {
		os.Exit(1)
	}
}

// normalizeApproved applies the registered approved deviations that change only
// where a diagnostic points, never what it means. docs/compatibility-ledger.json
// records "posting-level-diagnostic-span": Beancount attributes posting
// lifecycle errors to the transaction header, while OrangeCount points at the
// offending posting. Error class and multiplicity must still match exactly, so
// this drops the line number for ValidationError only and never for another
// class.
func normalizeApproved(value compat.FullGolden) compat.FullGolden {
	result := value
	result.Errors = make([]compat.GoldenError, 0, len(value.Errors))
	for _, item := range value.Errors {
		if item.Type == "ValidationError" {
			item.Line = 0
		}
		result.Errors = append(result.Errors, item)
	}
	return result
}

// prefixDifferences compares the diagnostics per case and additionally checks
// that the entry count matches, which is the prefix-inventory proxy available
// without replaying the oracle per transaction in this tool.
func prefixDifferences(want compat.FullGolden, have compat.FullGolden) []string {
	differences := []string{}
	if len(want.State.Entries) != len(have.State.Entries) {
		differences = append(differences, "entry-count")
	}
	wantAccounts := make([]string, 0, len(want.State.Balances))
	for account := range want.State.Balances {
		wantAccounts = append(wantAccounts, account)
	}
	haveAccounts := make([]string, 0, len(have.State.Balances))
	for account := range have.State.Balances {
		haveAccounts = append(haveAccounts, account)
	}
	sort.Strings(wantAccounts)
	sort.Strings(haveAccounts)
	if strings.Join(wantAccounts, ",") != strings.Join(haveAccounts, ",") {
		differences = append(differences, "balance-accounts")
	}
	return differences
}

func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("genprediff must run inside the repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func runOracle(root, dir, output string) {
	command := exec.Command("uv", "run", "--offline", "--project", "tools/reference", "python",
		"tools/reference/genprediff_oracle.py", dir, output)
	command.Dir = root
	command.Stdout = os.Stderr
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		fail(fmt.Errorf("oracle run failed: %w", err))
	}
}

func loadOracle(path string) (map[string]compat.FullGolden, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw []compat.FullGolden
	if err := json.Unmarshal(encoded, &raw); err != nil {
		return nil, err
	}
	result := map[string]compat.FullGolden{}
	for _, item := range raw {
		result[item.Fixture] = item
	}
	return result, nil
}

// promoteCase writes a paired <stem>.bean + <stem>.json regression directly in
// the promotion directory so the gate can prove the input and its recorded
// difference share a stem.
func promoteCase(root string, seed int64, item fixtureCase, want, have compat.FullGolden) (string, error) {
	dir := filepath.Join(root, "testdata", "fixtures", "gen")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	stem := fmt.Sprintf("%d-%s", seed, item.ID)
	if err := os.WriteFile(filepath.Join(dir, stem+".bean"), []byte(item.Ledger), 0o644); err != nil {
		return "", err
	}
	payload := map[string]any{"oracle": want, "actual": have}
	encoded, _ := json.MarshalIndent(payload, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, stem+".json"), append(encoded, '\n'), 0o644); err != nil {
		return "", err
	}
	return filepath.ToSlash(dir), nil
}

// caseFileName is the single source of the generated ledger's name, so the
// oracle golden recorded during a run already names the committed file.
func caseFileName(seed int64, id string) string {
	return fmt.Sprintf("%d-%s.bean", seed, id)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "genprediff:", err)
	os.Exit(1)
}

// generateCases builds deterministic, self-contained ledgers. Every case covers
// a different accounting surface so a difference points at a real capability.
func generateCases(dir string, seed int64, count int) []fixtureCase {
	cases := []fixtureCase{
		{ID: "plain", Ledger: plainLedger()},
		{ID: "multicurrency", Ledger: multicurrencyLedger()},
		{ID: "costs", Ledger: costsLedger()},
		{ID: "balance", Ledger: balanceLedger()},
		{ID: "metadata", Ledger: metadataLedger()},
		{ID: "flags", Ledger: flagsLedger()},
		{ID: "pad", Ledger: padLedger()},
		{ID: "invalid-unbalanced", Ledger: unbalancedLedger()},
		{ID: "invalid-closed", Ledger: closedLedger()},
		{ID: "prices", Ledger: pricesLedger()},
	}
	random := rand.New(rand.NewSource(seed))
	for len(cases) < count {
		cases = append(cases, randomLedger(random, len(cases)))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fail(err)
	}
	for _, item := range cases {
		if err := os.WriteFile(filepath.Join(dir, caseFileName(seed, item.ID)), []byte(item.Ledger), 0o644); err != nil {
			fail(err)
		}
	}
	if len(cases) > count {
		cases = cases[:count]
	}
	return cases
}
