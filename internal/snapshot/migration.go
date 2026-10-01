// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package snapshot

import (
	"encoding/json"
	"orangecount/internal/diagnostic"
	"orangecount/internal/ledger"
	"orangecount/internal/source"
	"sort"
	"strings"
)

type ProfileSummary struct {
	Profile    ledger.SemanticsProfile `json:"semantics_profile"`
	Revision   string                  `json:"evaluation_revision"`
	SnapshotID string                  `json:"snapshot_id"`
	Valid      bool                    `json:"valid"`
	EntryCount int                     `json:"entry_count"`
	ErrorCount int                     `json:"error_count"`
	ErrorCodes map[string]int          `json:"error_codes"`
	Extensions []string                `json:"active_extensions"`
}

type MigrationChange struct {
	Dimension string `json:"dimension"`
	Key       string `json:"key"`
	Before    any    `json:"before"`
	After     any    `json:"after"`
}

// MigrationComparison defaults to counts only. Detailed account names, exact
// amounts and entries are opt-in for local review and must never enter logs.
type MigrationComparison struct {
	InputIdentity string            `json:"input_identity"`
	Legacy        ProfileSummary    `json:"legacy"`
	V3            ProfileSummary    `json:"v3"`
	ChangeCounts  map[string]int    `json:"change_counts"`
	Details       []MigrationChange `json:"details,omitempty"`
}

// InputIdentity binds a migration preview to the exact source graph bytes.
func InputIdentity(graph *source.Graph) string { return snapshotID(graph) }

func CompareSemantics(graph *source.Graph, localDetails bool) MigrationComparison {
	old := BuildGraphWithOptions(graph, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.Legacy}})
	new := BuildGraphWithOptions(graph, BuildOptions{Evaluation: ledger.EvalOptions{SemanticsProfile: ledger.V3}})
	result := MigrationComparison{InputIdentity: snapshotID(graph), Legacy: profileSummary(old, ledger.Legacy), V3: profileSummary(new, ledger.V3), ChangeCounts: map[string]int{}}
	if result.Legacy.Valid != result.V3.Valid {
		result.ChangeCounts["validity"] = 1
	}
	for _, dimension := range []string{"accounts", "balances", "lots", "entries", "diagnostics"} {
		before, after := comparisonValues(old, dimension), comparisonValues(new, dimension)
		keys := map[string]bool{}
		for key := range before {
			keys[key] = true
		}
		for key := range after {
			keys[key] = true
		}
		ordered := make([]string, 0, len(keys))
		for key := range keys {
			ordered = append(ordered, key)
		}
		sort.Strings(ordered)
		for _, key := range ordered {
			left, right := before[key], after[key]
			x, _ := json.Marshal(left)
			y, _ := json.Marshal(right)
			if string(x) == string(y) {
				continue
			}
			result.ChangeCounts[dimension]++
			if localDetails {
				result.Details = append(result.Details, MigrationChange{dimension, key, left, right})
			}
		}
	}
	return result
}

func profileSummary(result BuildResult, profile ledger.SemanticsProfile) ProfileSummary {
	value := ProfileSummary{Profile: profile, Revision: profile.EvaluationRevision(), Valid: result.Snapshot != nil, ErrorCodes: map[string]int{}, Extensions: []string{}}
	if result.Snapshot != nil {
		value.SnapshotID = result.Snapshot.ID
	}
	if result.evaluation != nil {
		value.EntryCount = len(result.evaluation.Entries)
		value.Extensions = append([]string{}, result.evaluation.ActiveExtensions...)
	}
	for _, d := range result.Diagnostics {
		if d.Severity == diagnostic.Error {
			value.ErrorCount++
			value.ErrorCodes[d.Code]++
		}
	}
	return value
}

func comparisonValues(result BuildResult, dimension string) map[string]any {
	values := map[string]any{}
	if dimension == "diagnostics" {
		for _, d := range result.Diagnostics {
			encoded, _ := json.Marshal(d)
			key := string(encoded)
			count, _ := values[key].(int)
			values[key] = count + 1
		}
		return values
	}
	e := result.evaluation
	if e == nil {
		return values
	}
	if dimension == "entries" {
		// A multiset preserves repeated directives/postings. Both sides were
		// parsed afresh from the same graph, so source locations are identical.
		for _, entry := range e.Entries {
			encoded, _ := json.Marshal(entry.Directive)
			key := string(encoded)
			count, _ := values[key].(int)
			values[key] = count + 1
		}
		return values
	}
	for name, account := range e.Accounts {
		switch dimension {
		case "accounts":
			values[name] = struct {
				Opened     string
				Closed     *ledger.Date
				Booking    string
				Currencies []string
			}{account.Opened.Raw, account.Closed, account.Booking, account.Currencies}
		case "balances":
			for cur, amount := range account.Balances {
				if !amount.IsZero() {
					values[name+"\x00"+cur] = amount.String()
				}
			}
		case "lots":
			for _, lot := range account.Positions {
				date, label, number, quote, labelKind := "", "", "", "", "absent"
				if lot.Cost != nil {
					if lot.Cost.LabelPresent {
						labelKind = "present"
					}
					number, quote, label = lot.Cost.Number.String(), lot.Cost.Currency, lot.Cost.Label
					if lot.Cost.Date != nil {
						date = lot.Cost.Date.Raw
					}
				}
				key, _ := json.Marshal([]string{name, lot.Currency, number, quote, date, label, labelKind})
				current := ledger.Zero()
				if previous, ok := values[string(key)].(string); ok {
					current, _ = ledger.ParseDecimal(previous)
				}
				values[string(key)] = current.Add(lot.Units).String()
			}
		}
	}
	// A zero lot is absent inventory, not an additional identity.
	if dimension == "lots" {
		for key, value := range values {
			if strings.TrimSpace(value.(string)) == "0" {
				delete(values, key)
			}
		}
	}
	return values
}
