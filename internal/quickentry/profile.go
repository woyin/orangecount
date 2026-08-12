// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package quickentry compiles the transient quick-entry shorthand into
// canonical Beancount transactions. The package is UI-independent: it reads a
// ledger Evaluation, resolves the effective quick-entry profile for a given
// transaction date, parses shorthand text, and returns NewEntry values the
// Add Entry path already serializes. Publication, atomic write, backup,
// revalidation, and snapshot management stay in the web layer.
package quickentry

import (
	"fmt"
	"sort"
	"strings"

	"orangecount/internal/ledger"
)

// SchemaVersion is the public quick-entry profile schema version. It appears
// in the Beancount custom directive type name (for example
// "orangecount.quick-account.v1") and is fixed by ADR-0043. An incompatible
// representation requires a new suffix; .v1 meaning is frozen.
const SchemaVersion = "v1"

// The public custom directive type names. These are part of the source-ledger
// contract: persisted user ledgers depend on the strings staying stable.
const (
	accountCustomType  = "orangecount.quick-account." + SchemaVersion
	templateCustomType = "orangecount.quick-template." + SchemaVersion
)

// AccountRule maps an alias (like "微信") to a fully qualified Beancount
// account. It is the effective resolution of one quick-account custom
// directive as of a given transaction date.
type AccountRule struct {
	Alias   string
	Account string
	// SourceDate is the directive date that made this rule effective.
	SourceDate ledger.Date
}

// TemplateRule is the effective resolution of one quick-template custom
// directive. A template may prefill either or both posting roles plus
// descriptive fields, but never amount or date (those are always supplied at
// capture time per the grilling consensus).
type TemplateRule struct {
	Name        string
	Source      string
	Destination string
	Currency    string
	Payee       string
	Narration   string
	Tags        []string
	Links       []string
	// SourceDate is the directive date that made this rule effective.
	SourceDate ledger.Date
}

// Profile is the set of effective rules for one transaction date. The same
// transaction date always resolves to the same Profile for the same ledger;
// this is the determinism contract fixed during grilling.
type Profile struct {
	Accounts  []AccountRule
	Templates []TemplateRule
	// Problems are non-blocking configuration issues (unsupported versions,
	// same-day competing definitions, malformed typed values). They disable
	// only the affected rule and never invalidate the accounting ledger.
	Problems []ProfileProblem
}

// ProfileProblem describes a non-accounting issue in a syntactically valid
// quick-entry-profile custom directive. See CONTEXT.md:
// "Quick-entry profile diagnostic".
type ProfileProblem struct {
	Code    string
	Message string
	Source  string
}

// EffectiveProfile reads the dated quick-entry custom directives from an
// Evaluation and returns the rules effective as of the given transaction
// date. Rules dated after the transaction date do not apply (historical
// capture uses historically correct mappings).
func EffectiveProfile(evaluation *ledger.Evaluation, txnDate ledger.Date) Profile {
	var profile Profile
	if evaluation == nil {
		return profile
	}
	var accountCandidates []accountCandidate
	var templateCandidates []templateCandidate

	for _, entry := range evaluation.Entries {
		custom, ok := entry.Directive.(ledger.Custom)
		if !ok {
			continue
		}
		if !custom.Date.Valid() || dateIsAfter(custom.Date, txnDate) {
			continue
		}
		switch custom.Type {
		case accountCustomType:
			rule, problem, ok := parseAccountCustom(custom)
			if problem != nil {
				profile.Problems = append(profile.Problems, *problem)
			}
			if ok {
				accountCandidates = append(accountCandidates, accountCandidate{rule: rule, order: len(accountCandidates)})
			}
		case templateCustomType:
			rule, problem, ok := parseTemplateCustom(custom)
			if problem != nil {
				profile.Problems = append(profile.Problems, *problem)
			}
			if ok {
				templateCandidates = append(templateCandidates, templateCandidate{rule: rule, order: len(templateCandidates)})
			}
		default:
			if strings.HasPrefix(custom.Type, "orangecount.quick-account.") ||
				strings.HasPrefix(custom.Type, "orangecount.quick-template.") {
				profile.Problems = append(profile.Problems, ProfileProblem{
					Code:    "W-QUICK-SCHEMA-UNSUPPORTED",
					Message: fmt.Sprintf("unsupported quick-entry schema %q; compile ignored", custom.Type),
					Source:  custom.Type,
				})
			}
		}
	}

	profile.Accounts = resolveAccountRules(accountCandidates, &profile.Problems)
	profile.Templates = resolveTemplateRules(templateCandidates, &profile.Problems)
	return profile
}

type accountCandidate struct {
	rule  AccountRule
	order int
}

type templateCandidate struct {
	rule  TemplateRule
	order int
}

// parseAccountCustom parses one orangecount.quick-account.v1 directive.
// Expected form:
//
//	2026-01-01 custom "orangecount.quick-account.v1" "微信" Assets:WeChat
func parseAccountCustom(custom ledger.Custom) (AccountRule, *ProfileProblem, bool) {
	if len(custom.Values) < 2 {
		return AccountRule{}, &ProfileProblem{
			Code:    "W-QUICK-ACCOUNT-ARITY",
			Message: "quick-account rule needs an alias and an account",
			Source:  custom.Type,
		}, false
	}
	aliasValue := custom.Values[0]
	accountValue := custom.Values[1]
	if aliasValue.Kind != ledger.ValueString || strings.TrimSpace(aliasValue.String) == "" {
		return AccountRule{}, &ProfileProblem{
			Code:    "W-QUICK-ACCOUNT-ALIAS",
			Message: "quick-account alias must be a non-empty string",
			Source:  custom.Type,
		}, false
	}
	account := accountString(accountValue)
	if account == "" {
		return AccountRule{}, &ProfileProblem{
			Code:    "W-QUICK-ACCOUNT-ACCOUNT",
			Message: "quick-account value must be a valid account",
			Source:  custom.Type,
		}, false
	}
	return AccountRule{
		Alias:      aliasValue.String,
		Account:    account,
		SourceDate: custom.Date,
	}, nil, true
}

// parseTemplateCustom parses one orangecount.quick-template.v1 directive.
// Expected form:
//
//	2026-01-01 custom "orangecount.quick-template.v1" "午餐"
//	  destination: "Expenses:Food"
//	  currency: "CNY"
func parseTemplateCustom(custom ledger.Custom) (TemplateRule, *ProfileProblem, bool) {
	if len(custom.Values) < 1 || custom.Values[0].Kind != ledger.ValueString ||
		strings.TrimSpace(custom.Values[0].String) == "" {
		return TemplateRule{}, &ProfileProblem{
			Code:    "W-QUICK-TEMPLATE-NAME",
			Message: "quick-template name must be a non-empty string",
			Source:  custom.Type,
		}, false
	}
	rule := TemplateRule{
		Name:       custom.Values[0].String,
		SourceDate: custom.Date,
	}
	for _, meta := range custom.Meta {
		switch meta.Key {
		case "source":
			rule.Source = strings.TrimSpace(meta.Value.String)
		case "destination":
			rule.Destination = strings.TrimSpace(meta.Value.String)
		case "currency":
			rule.Currency = strings.TrimSpace(meta.Value.String)
		case "payee":
			rule.Payee = strings.TrimSpace(meta.Value.String)
		case "narration":
			rule.Narration = strings.TrimSpace(meta.Value.String)
		case "tags":
			if t := strings.TrimSpace(meta.Value.String); t != "" {
				rule.Tags = splitSpaceList(t)
			}
		case "links":
			if l := strings.TrimSpace(meta.Value.String); l != "" {
				rule.Links = splitSpaceList(l)
			}
		}
	}
	return rule, nil, true
}

func resolveAccountRules(candidates []accountCandidate, problems *[]ProfileProblem) []AccountRule {
	byAlias := make(map[string][]accountCandidate)
	for _, c := range candidates {
		byAlias[c.rule.Alias] = append(byAlias[c.rule.Alias], c)
	}
	var aliases []string
	for alias := range byAlias {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	var result []AccountRule
	for _, alias := range aliases {
		group := byAlias[alias]
		winner, ambiguous := latestAccount(group)
		if ambiguous {
			*problems = append(*problems, ProfileProblem{
				Code:    "W-QUICK-ACCOUNT-AMBIGUOUS",
				Message: fmt.Sprintf("multiple quick-account definitions for %q on the same effective date; rule disabled", alias),
				Source:  alias,
			})
			continue
		}
		result = append(result, winner)
	}
	return result
}

func resolveTemplateRules(candidates []templateCandidate, problems *[]ProfileProblem) []TemplateRule {
	byName := make(map[string][]templateCandidate)
	for _, c := range candidates {
		byName[c.rule.Name] = append(byName[c.rule.Name], c)
	}
	var names []string
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	var result []TemplateRule
	for _, name := range names {
		group := byName[name]
		winner, ambiguous := latestTemplate(group)
		if ambiguous {
			*problems = append(*problems, ProfileProblem{
				Code:    "W-QUICK-TEMPLATE-AMBIGUOUS",
				Message: fmt.Sprintf("multiple quick-template definitions for %q on the same effective date; rule disabled", name),
				Source:  name,
			})
			continue
		}
		result = append(result, winner)
	}
	return result
}

func latestAccount(group []accountCandidate) (AccountRule, bool) {
	if len(group) == 0 {
		return AccountRule{}, false
	}
	best := group[0]
	for _, c := range group[1:] {
		cmp := compareDate(c.rule.SourceDate, best.rule.SourceDate)
		if cmp > 0 {
			best = c
		} else if cmp == 0 {
			return AccountRule{}, true
		}
	}
	return best.rule, false
}

func latestTemplate(group []templateCandidate) (TemplateRule, bool) {
	if len(group) == 0 {
		return TemplateRule{}, false
	}
	best := group[0]
	for _, c := range group[1:] {
		cmp := compareDate(c.rule.SourceDate, best.rule.SourceDate)
		if cmp > 0 {
			best = c
		} else if cmp == 0 {
			return TemplateRule{}, true
		}
	}
	return best.rule, false
}

func compareDate(a, b ledger.Date) int {
	switch {
	case a.Year != b.Year:
		return a.Year - b.Year
	case a.Month != b.Month:
		return a.Month - b.Month
	default:
		return a.Day - b.Day
	}
}

func dateIsAfter(a, reference ledger.Date) bool {
	return compareDate(a, reference) > 0
}

func accountString(v ledger.Value) string {
	switch v.Kind {
	case ledger.ValueAccount:
		return v.String
	case ledger.ValueString:
		s := strings.TrimSpace(v.String)
		if isValidAccount(s) {
			return s
		}
	}
	return ""
}

func isValidAccount(s string) bool {
	if !strings.Contains(s, ":") {
		return false
	}
	for _, part := range strings.Split(s, ":") {
		if part == "" {
			return false
		}
		first := part[0]
		if first < 'A' || first > 'Z' {
			return false
		}
	}
	return true
}

func splitSpaceList(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t'
	})
	var result []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}
