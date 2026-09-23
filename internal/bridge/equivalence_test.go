// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orangecount/internal/report"
)

type goldenFixture struct {
	Fixture  string `json:"fixture"`
	Valid    bool   `json:"valid"`
	Balances map[string]struct {
		Currencies map[string]string `json:"currencies"`
		Lots       []string          `json:"lots"`
	} `json:"balances"`
}

func loadGoldenFile(name string) (goldenFixture, error) {
	fixtureGolden := filepath.Join(testFixturePath("..", "..", "golden", "v3-parity"), strings.TrimSuffix(name, ".bean")+".golden.json")
	var g goldenFixture
	data, err := os.ReadFile(fixtureGolden)
	if err != nil {
		return g, err
	}
	err = json.Unmarshal(data, &g)
	return g, err
}

// TestEquivalenceL1AndL2_ReportsAndTreeInvariants rigorously verifies that
// native desktop tree reports match official Beancount v3 ground truth balances
// and satisfy all 4 tree aggregation invariants.
func TestEquivalenceL1AndL2_ReportsAndTreeInvariants(t *testing.T) {
	matches, err := filepath.Glob(testFixturePath("*.bean"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("no parity fixtures found: %v", err)
	}

	for _, fixturePath := range matches {
		fileName := filepath.Base(fixturePath)
		t.Run(fileName, func(t *testing.T) {
			golden, err := loadGoldenFile(fileName)
			if err != nil {
				t.Fatalf("failed to read golden file for %s: %v", fileName, err)
			}

			// If golden says invalid, verify Native snapshot rejects it (L4 Lifecycle Equivalence)
			if !golden.Valid {
				snap := loadSnapshot(fixturePath)
				if snap.Valid {
					t.Fatalf("fixture %s is invalid in Beancount, but Native snapshot reported valid", fileName)
				}
				if snap.ErrorCount == 0 && len(snap.Errors) == 0 {
					t.Fatalf("fixture %s is invalid, but snapshot produced 0 error diagnostics", fileName)
				}
				return
			}

			// For valid fixtures: verify L1 balances & L2 tree invariants
			// Balance Sheet covers Assets, Liabilities, Equity
			bsTree := loadTreeReport(fixturePath, "balance_sheet")
			if bsTree.Error != "" {
				t.Fatalf("unexpected tree report error for %s: %s", fileName, bsTree.Error)
			}

			// Income Statement covers Income, Expenses
			isTree := loadTreeReport(fixturePath, "income_statement")
			if isTree.Error != "" {
				t.Fatalf("unexpected income statement error for %s: %s", fileName, isTree.Error)
			}

			flatDirect := map[string]map[string]string{}
			for _, root := range bsTree.RootNodes {
				verifyTreeInvariants(t, root, flatDirect)
			}
			for _, root := range isTree.RootNodes {
				verifyTreeInvariants(t, root, flatDirect)
			}

			// L1 Equivalence: compare direct balances with golden
			for acct, gBalance := range golden.Balances {
				if len(gBalance.Currencies) == 0 {
					continue
				}
				direct, exists := flatDirect[acct]
				if !exists && len(gBalance.Currencies) > 0 {
					// Check if account has only lots
					t.Fatalf("[%s] account %s has golden balances %v, but was missing in native tree direct balances", fileName, acct, gBalance.Currencies)
				}
				for cur, wantAmt := range gBalance.Currencies {
					gotAmt := direct[cur]
					if canonicalNum(gotAmt) != canonicalNum(wantAmt) {
						t.Fatalf("[%s] balance mismatch on %s %s: got %s, want %s", fileName, acct, cur, gotAmt, wantAmt)
					}
				}
			}
		})
	}
}

// verifyTreeInvariants recursively checks:
// 1. Depth invariant: child.depth == parent.depth + 1
// 2. Leaf invariant: node.is_leaf == (len(node.children) == 0)
// 3. Namespace invariant: child.full_name starts with parent.full_name + ":"
// 4. Aggregation invariant: subtree_balance == direct_balance + sum(children.subtree_balance)
func verifyTreeInvariants(t *testing.T, node report.AccountTreeNode, flatDirect map[string]map[string]string) {
	if len(node.DirectBalances) > 0 {
		flatDirect[node.FullName] = node.DirectBalances
	}

	if node.IsLeaf != (len(node.Children) == 0) {
		t.Fatalf("leaf invariant violated on %s: is_leaf=%v, len(children)=%d", node.FullName, node.IsLeaf, len(node.Children))
	}

	// Calculate expected subtree total: own direct + children subtree sums
	expectedSubtree := map[string]*big.Rat{}
	for cur, sVal := range node.DirectBalances {
		r, ok := new(big.Rat).SetString(sVal)
		if ok {
			expectedSubtree[cur] = r
		}
	}

	for _, child := range node.Children {
		// Depth invariant
		if child.Depth != node.Depth+1 {
			t.Fatalf("depth invariant violated on %s -> %s: parent depth=%d, child depth=%d", node.FullName, child.FullName, node.Depth, child.Depth)
		}
		// Namespace invariant
		if !strings.HasPrefix(child.FullName, node.FullName+":") {
			t.Fatalf("namespace invariant violated on child %s of parent %s", child.FullName, node.FullName)
		}

		for cur, sVal := range child.SubtreeBalances {
			r, ok := new(big.Rat).SetString(sVal)
			if ok {
				prev, exists := expectedSubtree[cur]
				if !exists {
					expectedSubtree[cur] = r
				} else {
					expectedSubtree[cur] = new(big.Rat).Add(prev, r)
				}
			}
		}

		verifyTreeInvariants(t, child, flatDirect)
	}

	// Verify Aggregation Invariant against node.SubtreeBalances
	for cur, expRat := range expectedSubtree {
		if expRat.Sign() == 0 {
			continue
		}
		actualStr, ok := node.SubtreeBalances[cur]
		if !ok {
			t.Fatalf("aggregation invariant violated on %s: missing currency %s in subtree_balances (expected %s)", node.FullName, cur, expRat.String())
		}
		actRat, ok := new(big.Rat).SetString(actualStr)
		if !ok || actRat.Cmp(expRat) != 0 {
			t.Fatalf("aggregation invariant violated on %s %s: subtree has %s, children+direct sum to %s", node.FullName, cur, actualStr, expRat.String())
		}
	}
}

func canonicalNum(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "0"
	}
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	if s == "-0" || s == "" {
		return "0"
	}
	return s
}

// TestEquivalenceL1_Journal verifies transaction and posting integrity.
func TestEquivalenceL1_Journal(t *testing.T) {
	fixture := testFixturePath("txn-basic.bean")
	journal := loadJournal(fixture, "")
	if journal.Error != "" {
		t.Fatalf("journal error: %s", journal.Error)
	}
	if journal.TotalCount != 1 {
		t.Fatalf("expected 1 transaction, got %d", journal.TotalCount)
	}
	tx := journal.Transactions[0]
	if len(tx.Postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(tx.Postings))
	}

	// Verify posting balances sum to zero for simple single-currency transactions
	sum := new(big.Rat)
	for _, p := range tx.Postings {
		r, ok := new(big.Rat).SetString(p.UnitsNumber)
		if !ok {
			t.Fatalf("invalid units number: %s", p.UnitsNumber)
		}
		sum = sum.Add(sum, r)
	}
	if sum.Sign() != 0 {
		t.Fatalf("transaction postings do not balance to zero: sum = %s", sum.String())
	}
}

// TestEquivalenceL3_HistoricalTrends verifies that trend time-series
// compute successfully for valid ledgers.
func TestEquivalenceL3_HistoricalTrends(t *testing.T) {
	fixture := testFixturePath("txn-basic.bean")
	trends := loadHistoricalTrends(fixture)
	if trends.Error != "" {
		t.Fatalf("trends error: %s", trends.Error)
	}
	if len(trends.NetWorthPoints) == 0 {
		t.Fatal("expected at least 1 net worth point")
	}
	latest := trends.NetWorthPoints[len(trends.NetWorthPoints)-1]
	if latest.Value <= 0 {
		t.Fatalf("expected positive net worth for txn-basic, got %f", latest.Value)
	}
}
