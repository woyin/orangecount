// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package ledger

import "strings"

// AccountTypes holds the five root account names in effect for an evaluation.
// The names are configurable through the name_assets, name_liabilities,
// name_equity, name_income, and name_expenses options, matching the upstream
// account-type configuration surface. Reports, queries, and charts classify
// accounts through TypeOf instead of comparing against hard-coded English
// literals, so a ledger that renames its roots keeps every report populated.
type AccountTypes struct {
	Assets      string
	Liabilities string
	Equity      string
	Income      string
	Expenses    string
}

// accountTypeKeys is the fixed canonical order of the five account types. The
// canonical keys stay English because they are internal identifiers, never
// user-facing labels; RootOf maps a key back to the configured display name.
var accountTypeKeys = [5]string{"Assets", "Liabilities", "Equity", "Income", "Expenses"}

// DefaultAccountTypes returns the upstream default root account names.
func DefaultAccountTypes() AccountTypes {
	return AccountTypes{
		Assets:      "Assets",
		Liabilities: "Liabilities",
		Equity:      "Equity",
		Income:      "Income",
		Expenses:    "Expenses",
	}
}

// RootOf returns the configured root name for a canonical account-type key.
// An unknown key returns an empty string.
func (t AccountTypes) RootOf(kind string) string {
	switch kind {
	case "Assets":
		return t.Assets
	case "Liabilities":
		return t.Liabilities
	case "Equity":
		return t.Equity
	case "Income":
		return t.Income
	case "Expenses":
		return t.Expenses
	}
	return ""
}

// TypeOf classifies an account by its root component and returns the canonical
// key. An account whose root is not one of the five configured types returns an
// empty string; callers must treat that as unclassified rather than guessing,
// because silently guessing would move balances between statements.
func (t AccountTypes) TypeOf(account string) string {
	root := account
	if at := strings.IndexByte(root, ':'); at >= 0 {
		root = root[:at]
	}
	if root == "" {
		return ""
	}
	for _, kind := range accountTypeKeys {
		if root == t.RootOf(kind) {
			return kind
		}
	}
	return ""
}

// Keys returns the canonical account-type keys in their stable order.
func (t AccountTypes) Keys() []string {
	return []string{"Assets", "Liabilities", "Equity", "Income", "Expenses"}
}

// AccountTypes returns the root account names configured by the name_* options.
// Values that are not syntactically valid root names are ignored here; the
// evaluator reports them as option errors so a typo never silently renames a
// statement.
func (e Evaluation) AccountTypes() AccountTypes {
	types := DefaultAccountTypes()
	options := [5]struct {
		key  string
		dest *string
	}{
		{"name_assets", &types.Assets},
		{"name_liabilities", &types.Liabilities},
		{"name_equity", &types.Equity},
		{"name_income", &types.Income},
		{"name_expenses", &types.Expenses},
	}
	for _, option := range options {
		value, ok := e.Options[option.key]
		if !ok || !ValidRootName(value) {
			continue
		}
		*option.dest = value
	}
	return types
}

// activateUTF8AccountTypes marks an extension activation when a configured root
// name contains a non-ASCII rune. Upstream accepts UTF-8 in the components
// after the root, but its root-name production only admits an ASCII uppercase
// lead byte, so a UTF-8 root is a dialect extension that a v3 reader cannot
// load. Registering it keeps sourceexport's refusal to emit a non-portable
// snapshot accurate instead of silently writing a file no v3 tool can read.
func (e *evaluator) activateUTF8AccountTypes(root string) {
	for _, r := range root {
		if r > 0x7F {
			if !contains(e.result.ActiveExtensions, "utf8-account-root") {
				e.result.ActiveExtensions = append(e.result.ActiveExtensions, "utf8-account-root")
			}
			return
		}
	}
}

// isAccountNameOption reports whether key is one of the five options that
// rename a root account type.
func isAccountNameOption(key string) bool {
	switch key {
	case "name_assets", "name_liabilities", "name_equity", "name_income", "name_expenses":
		return true
	}
	return false
}

// accountNamesDistinct reports whether the five configured root names are
// mutually distinct. Two types sharing one root name would make classification
// ambiguous, so the evaluator rejects that configuration.
func (e *evaluator) accountNamesDistinct() bool {
	types := e.result.AccountTypes()
	seen := make(map[string]bool, 5)
	for _, kind := range types.Keys() {
		root := types.RootOf(kind)
		if root == "" {
			return false
		}
		if seen[root] {
			return false
		}
		seen[root] = true
	}
	return true
}

// ValidRootName reports whether name can be a root account name, matching the
// upstream ACCOUNTTYPE production: an uppercase letter or a UTF-8 lead byte,
// followed by letters, digits, or hyphens. A root name never contains a
// separator.
func ValidRootName(name string) bool {
	if name == "" || strings.Contains(name, ":") {
		return false
	}
	first := true
	for _, r := range name {
		utf8Only := r > 0x7F
		if first {
			first = false
			if utf8Only {
				continue
			}
			if r < 'A' || r > 'Z' {
				return false
			}
			continue
		}
		if utf8Only {
			continue
		}
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
}
