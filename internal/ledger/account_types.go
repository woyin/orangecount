// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package ledger

import "strings"

// AccountRootScheme selects the accepted root-name vocabulary. It is a
// ledger-level switch because a root name is part of a ledger's convention,
// not of an individual account: mixing two vocabularies inside one file would
// make "资产:Cash" and "Assets:Cash" rival spellings of the same type with no
// way for a reader to tell which one the author meant.
type AccountRootScheme string

const (
	// SchemeGlobal is the upstream Beancount v3 vocabulary and the default:
	// only the five English root names are accepted.
	SchemeGlobal AccountRootScheme = "global"
	// SchemeCN additionally accepts the five Chinese root names as exact
	// equivalents of the English ones. Both vocabularies stay usable in the
	// same ledger, so a Chinese ledger can adopt the scheme without renaming
	// existing English-rooted accounts.
	SchemeCN AccountRootScheme = "cn"
)

// AccountRootSchemeOption is the option key that selects the scheme.
const AccountRootSchemeOption = "account_root_scheme"

// cnRootNames maps each canonical account type to its Chinese equivalent.
var cnRootNames = map[string]string{
	"Assets":      "资产",
	"Liabilities": "负债",
	"Equity":      "权益",
	"Income":      "收入",
	"Expenses":    "支出",
}

// accountTypeKeys is the fixed canonical order of the five account types. The
// canonical keys stay English because they are internal identifiers, never
// user-facing labels; RootOf maps a key back to the configured display name.
var accountTypeKeys = [5]string{"Assets", "Liabilities", "Equity", "Income", "Expenses"}

// AccountTypes holds the root names in effect for an evaluation, together with
// the accepted vocabulary. Reports, queries, and charts classify accounts
// through TypeOf instead of comparing against hard-coded English literals, so
// a ledger that renames or translates its roots keeps every report populated.
type AccountTypes struct {
	Assets      string
	Liabilities string
	Equity      string
	Income      string
	Expenses    string
	// Scheme selects whether the Chinese root names are also accepted.
	Scheme AccountRootScheme
}

// DefaultAccountTypes returns the upstream default root account names under the
// global scheme.
func DefaultAccountTypes() AccountTypes {
	return AccountTypes{
		Assets:      "Assets",
		Liabilities: "Liabilities",
		Equity:      "Equity",
		Income:      "Income",
		Expenses:    "Expenses",
		Scheme:      SchemeGlobal,
	}
}

// RootOf returns the configured primary root name for a canonical account-type
// key. An unknown key returns an empty string.
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

// RootsOf returns every root spelling accepted for a canonical type, primary
// first. Tree builders need the whole list rather than just the primary name,
// because a Chinese-scheme ledger may legitimately hold both "资产:钱包" and
// "Assets:Cash"; selecting only one spelling would drop the other subtree.
func (t AccountTypes) RootsOf(kind string) []string {
	primary := t.RootOf(kind)
	if primary == "" {
		return nil
	}
	roots := []string{primary}
	if t.Scheme == SchemeCN {
		if alias := cnRootNames[kind]; alias != "" && alias != primary {
			roots = append(roots, alias)
		}
	}
	return roots
}

// AllRoots returns every accepted root spelling across all five types.
func (t AccountTypes) AllRoots() []string {
	roots := make([]string, 0, 10)
	for _, kind := range accountTypeKeys {
		roots = append(roots, t.RootsOf(kind)...)
	}
	return roots
}

// TypeOf classifies an account by its root component and returns the canonical
// key. An account whose root is not an accepted spelling for any configured
// type returns an empty string; callers must treat that as unclassified rather
// than guessing, because guessing would move balances between statements.
func (t AccountTypes) TypeOf(account string) string {
	root := rootComponent(account)
	if root == "" {
		return ""
	}
	for _, kind := range accountTypeKeys {
		for _, accepted := range t.RootsOf(kind) {
			if root == accepted {
				return kind
			}
		}
	}
	return ""
}

// Keys returns the canonical account-type keys in their stable order.
func (t AccountTypes) Keys() []string {
	return []string{"Assets", "Liabilities", "Equity", "Income", "Expenses"}
}

// rootComponent returns the first component of an account name.
func rootComponent(account string) string {
	if at := strings.IndexByte(account, ':'); at >= 0 {
		return account[:at]
	}
	return account
}

// RootScheme resolves the scheme selected by the account_root_scheme option.
// An absent or unrecognized value falls back to the global scheme; the
// evaluator reports an unrecognized value as an option error so a typo never
// silently changes which roots a ledger accepts.
func (e Evaluation) RootScheme() AccountRootScheme {
	switch strings.ToLower(strings.TrimSpace(e.Options[AccountRootSchemeOption])) {
	case string(SchemeCN):
		return SchemeCN
	default:
		return SchemeGlobal
	}
}

// AccountTypes returns the root account names configured by the name_* options
// together with the vocabulary selected by account_root_scheme. Values that are
// not syntactically valid root names are ignored here; the evaluator reports
// them as option errors so a typo never silently renames a statement.
func (e Evaluation) AccountTypes() AccountTypes {
	types := DefaultAccountTypes()
	types.Scheme = e.RootScheme()
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

// DisplayRoot returns the root spelling to label a canonical type with. The
// spelling the ledger actually uses wins, so a Chinese-scheme ledger written in
// English keeps English labels and vice versa; the primary name is the fallback
// when neither spelling appears.
func (e Evaluation) DisplayRoot(kind string) string {
	types := e.AccountTypes()
	for _, root := range types.RootsOf(kind) {
		if e.RootInUse(root) {
			return root
		}
	}
	return types.RootOf(kind)
}

// RootInUse reports whether any account sits under root.
func (e Evaluation) RootInUse(root string) bool {
	if root == "" {
		return false
	}
	for name := range e.Accounts {
		if name == root || strings.HasPrefix(name, root+":") {
			return true
		}
	}
	return false
}

// activateUTF8AccountTypes marks an extension activation when a configured root
// name contains a non-ASCII rune. Upstream's root-name production only admits
// an ASCII uppercase lead byte, so a UTF-8 root is a dialect extension that a
// v3 reader cannot load. Registering it keeps sourceexport's refusal to emit a
// non-portable snapshot accurate instead of silently writing a file no v3 tool
// can read.
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

// accountNamesDistinct reports whether the accepted root names are mutually
// distinct. Two types sharing one root spelling would make classification
// ambiguous -- under the Chinese scheme that includes a name_* override that
// collides with another type's Chinese alias -- so the evaluator rejects the
// configuration instead of silently preferring one type.
func (e *evaluator) accountNamesDistinct() bool {
	types := e.result.AccountTypes()
	seen := make(map[string]bool, 10)
	for _, kind := range types.Keys() {
		roots := types.RootsOf(kind)
		if len(roots) == 0 || roots[0] == "" {
			return false
		}
		for _, root := range roots {
			if seen[root] {
				return false
			}
			seen[root] = true
		}
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
