// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Command perfbench generates a deterministic large ledger, times OrangeCount
// loading it, and, when a Python Beancount installation is available, times
// the same ledger under Beancount for an honest same-machine comparison.
package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"orangecount/internal/snapshot"
)

func main() {
	ledgerPath := flag.String("ledger", "", "existing ledger to benchmark instead of generating one")
	transactions := flag.Int("transactions", 50000, "two-leg transactions in the generated ledger")
	skipCompare := flag.Bool("skip-compare", false, "skip the optional Beancount comparison")
	pythonBin := flag.String("python", "python3", "Python interpreter used for the Beancount comparison")
	outPath := flag.String("out", "", "keep the generated ledger at this path instead of a temp dir")
	flag.Parse()

	entry, cleanup, err := prepareLedger(*ledgerPath, *transactions, *outPath)
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	ocCold, ocWarm := runOrangeCount(entry)
	fmt.Printf("OrangeCount  %s  cold=%s  warm=%s\n",
		runtime.Version(), ocCold.Round(time.Millisecond), ocWarm.Round(time.Millisecond))

	if *skipCompare {
		fmt.Println("Beancount    skipped (-skip-compare)")
		return
	}
	version, ok := detectBeancount(*pythonBin)
	if !ok {
		fmt.Printf("Beancount    skipped (no importable beancount for %s; try -python)\n", *pythonBin)
		return
	}
	bcCold, bcWarm, err := runBeancount(*pythonBin, entry)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("Beancount    %s  cold=%s  warm=%s\n", version, bcCold.Round(time.Millisecond), bcWarm.Round(time.Millisecond))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "perfbench:", err)
	os.Exit(1)
}

// prepareLedger returns the ledger path to benchmark and a cleanup func. With
// -ledger set the caller's file is used untouched; otherwise a deterministic
// ledger is generated (fixed seed) so results are reproducible everywhere.
func prepareLedger(existing string, transactions int, out string) (string, func(), error) {
	if existing != "" {
		return existing, func() {}, nil
	}
	data := generateLedger(transactions)
	dir := os.TempDir()
	if out != "" {
		if err := os.WriteFile(out, data, 0o600); err != nil {
			return "", nil, err
		}
		return out, func() {}, nil
	}
	entry := filepath.Join(dir, fmt.Sprintf("perfbench-%d.bean", time.Now().UnixNano()))
	if err := os.WriteFile(entry, data, 0o600); err != nil {
		return "", nil, err
	}
	return entry, func() { _ = os.Remove(entry) }, nil
}

// generateLedger builds the reference shape: dated two-leg transactions across
// a fixed account set in mixed currencies, unique-cost lot purchases, and
// matching reductions. Buys are transactions/25 and sells buys/4, mirroring
// the 50000/2000/500 ledger used for the published numbers.
func generateLedger(transactions int) []byte {
	rng := rand.New(rand.NewSource(42))
	accounts := []string{
		"Assets:Cash", "Assets:Bank:Checking", "Assets:Broker:Shares",
		"Expenses:Food:Groceries", "Expenses:Travel", "Income:Salary", "Equity:Opening",
	}
	commodities := []string{"USD", "EUR", "CNY"}
	var lines []string
	lines = append(lines, `option "operating_currency" "USD"`)
	for _, account := range accounts {
		lines = append(lines, fmt.Sprintf("2000-01-01 open %s USD,EUR,CNY", account))
	}
	lines = append(lines, "2000-01-01 open Assets:Broker:AAPL AAPL",
		"2000-01-01 commodity AAPL")
	for i := 1; i <= 4000; i++ {
		lines = append(lines, fmt.Sprintf("2000-01-01 price AAPL %d.00 USD", 100+i%50))
	}
	for i := 0; i < transactions; i++ {
		date := fmt.Sprintf("2010-%02d-%02d", i/100%12+1, i/100%28+1)
		amount := fmt.Sprintf("%d.%02d", 1+rng.Intn(5000), rng.Intn(100))
		currency := commodities[rng.Intn(len(commodities))]
		source, target := pickTwo(rng, accounts)
		lines = append(lines,
			fmt.Sprintf("%s * \"payee %d\" \"narration %d\"", date, i, i),
			fmt.Sprintf("  %s -%s %s", source, amount, currency),
			fmt.Sprintf("  %s %s %s", target, amount, currency))
	}
	buys := transactions / 25
	for i := 0; i < buys; i++ {
		cost := 100 + i
		lines = append(lines,
			fmt.Sprintf("2015-%02d-%02d * \"buy %d\" \"lot churn\"", i%12+1, i%28+1, i),
			fmt.Sprintf("  Assets:Broker:AAPL 10 AAPL {%d.00 USD}", cost),
			fmt.Sprintf("  Assets:Broker:Shares -%d.00 USD", 10*cost))
	}
	for i := 0; i < buys/4; i++ {
		cost := 100 + i
		lines = append(lines,
			fmt.Sprintf("2020-%02d-%02d * \"sell %d\" \"reduce\"", i%12+1, i%28+1, i),
			fmt.Sprintf("  Assets:Broker:AAPL -10 AAPL {%d.00 USD}", cost),
			fmt.Sprintf("  Assets:Broker:Shares %d.00 USD", 10*cost))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func pickTwo(rng *rand.Rand, accounts []string) (string, string) {
	source := rng.Intn(len(accounts))
	target := rng.Intn(len(accounts) - 1)
	if target >= source {
		target++
	}
	return accounts[source], accounts[target]
}

// runOrangeCount times one cold build plus the best of two warm rebuilds of
// the full pipeline (read, parse, evaluate, publish).
func runOrangeCount(entry string) (cold, warm time.Duration) {
	start := time.Now()
	first := snapshot.Build(entry)
	cold = time.Since(start)
	if first.Snapshot == nil {
		fatal(fmt.Errorf("generated ledger failed to build: %v", first.Diagnostics))
	}
	warm = time.Hour
	for i := 0; i < 2; i++ {
		start = time.Now()
		if result := snapshot.Build(entry); result.Snapshot == nil {
			fatal(fmt.Errorf("warm rebuild failed: %v", result.Diagnostics))
		}
		if elapsed := time.Since(start); elapsed < warm {
			warm = elapsed
		}
	}
	return cold, warm
}

// detectBeancount reports whether the interpreter can import Beancount, plus
// its version string for the report.
func detectBeancount(pythonBin string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, pythonBin, "-c", "import beancount; print(beancount.__version__)").Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// beancountScript times three loads in one interpreter: the first includes
// interpreter and module import warm-up (cold), the later ones reuse it
// (warm); it prints "cold_seconds warm_seconds" on the last line.
const beancountScript = `
import time
from beancount import loader
cold = warm = None
for i in range(3):
    t = time.perf_counter()
    entries, errors, _ = loader.load_file(ENTRY)
    dt = time.perf_counter() - t
    if errors:
        raise SystemExit("ledger has %d errors" % len(errors))
    if i == 0:
        cold = dt
    else:
        warm = dt if warm is None else min(warm, dt)
print("%f %f" % (cold, warm))
`

// runBeancount executes the timing script against the same ledger file and
// parses its "cold warm" line.
func runBeancount(pythonBin, entry string) (cold, warm time.Duration, err error) {
	script := strings.ReplaceAll(beancountScript, "ENTRY", fmt.Sprintf("%q", entry))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, pythonBin, "-c", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("beancount run failed: %v: %s", err, strings.TrimSpace(string(output)))
	}
	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("unexpected beancount output: %q", string(output))
	}
	var coldSeconds, warmSeconds float64
	if _, err = fmt.Sscanf(fields[len(fields)-2]+" "+fields[len(fields)-1], "%f %f", &coldSeconds, &warmSeconds); err != nil {
		return 0, 0, err
	}
	return time.Duration(coldSeconds * float64(time.Second)), time.Duration(warmSeconds * float64(time.Second)), nil
}
