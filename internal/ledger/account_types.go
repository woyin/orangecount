// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

package ledger

import "strings"

// AccountRootScheme selects the root-name vocabulary a ledger uses. It is a
// ledger-level switch because a root name is part of the ledger's convention,
// not of an individual account: mixing two localized vocabularies inside one
// file would make rival spellings of the same type with no way for a reader to
// tell which one the author meant.
type AccountRootScheme string

const (
	// SchemeGlobal is the upstream Beancount v3 vocabulary and the default:
	// only the five English root names are accepted.
	SchemeGlobal AccountRootScheme = "global"
	// SchemeZhHans adds the Simplified Chinese root names.
	SchemeZhHans AccountRootScheme = "zh-Hans"
	// SchemeZhHant adds the Traditional Chinese root names.
	SchemeZhHant AccountRootScheme = "zh-Hant"
	// SchemeJapanese adds the Japanese root names used by Japanese GAAP.
	SchemeJapanese AccountRootScheme = "ja"
	// SchemeKorean adds the Korean root names used by K-IFRS.
	SchemeKorean AccountRootScheme = "ko"
)

// AccountRootSchemeOption is the option key that selects the scheme.
const AccountRootSchemeOption = "account_root_scheme"

// localizedRoots maps a scheme to its five root names. The English names stay
// accepted under every scheme, so a localized ledger can adopt a vocabulary
// without renaming existing English-rooted accounts. Each set uses the standard
// statutory terms of its jurisdiction: Simplified and Traditional Chinese
// accounting, Japanese GAAP (資産/負債/純資産/収益/費用), and K-IFRS
// (자산/부채/자본/수익/비용).
var localizedRoots = map[AccountRootScheme]map[string]string{
	SchemeZhHans: {
		"Assets":      "资产",
		"Liabilities": "负债",
		"Equity":      "权益",
		"Income":      "收入",
		"Expenses":    "支出",
	},
	SchemeZhHant: {
		"Assets":      "資產",
		"Liabilities": "負債",
		"Equity":      "權益",
		"Income":      "收入",
		"Expenses":    "支出",
	},
	SchemeJapanese: {
		"Assets":      "資産",
		"Liabilities": "負債",
		"Equity":      "純資産",
		"Income":      "収益",
		"Expenses":    "費用",
	},
	SchemeKorean: {
		"Assets":      "자산",
		"Liabilities": "부채",
		"Equity":      "자본",
		"Income":      "수익",
		"Expenses":    "비용",
	},
}

// accountTypeKeys is the fixed canonical order of the five account types. The
// canonical keys stay English because they are internal identifiers, never
// user-facing labels; RootOf maps a key back to the configured display name.
var accountTypeKeys = [5]string{"Assets", "Liabilities", "Equity", "Income", "Expenses"}

// AccountTypes holds the root names in effect for an evaluation, together with
// the accepted vocabulary. Reports, queries, and charts classify accounts
// through TypeOf instead of comparing against hard-coded English literals, so a
// ledger that renames or localizes its roots keeps every report populated.
type AccountTypes struct {
	Assets      string
	Liabilities string
	Equity      string
	Income      string
	Expenses    string
	// Scheme selects which localized root names are also accepted.
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
// because a localized ledger may legitimately hold both "资产:钱包" and
// "Assets:Cash"; selecting only one spelling would drop the other subtree.
func (t AccountTypes) RootsOf(kind string) []string {
	primary := t.RootOf(kind)
	if primary == "" {
		return nil
	}
	roots := []string{primary}
	if localized, ok := localizedRoots[t.Scheme][kind]; ok && localized != primary {
		roots = append(roots, localized)
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

// resolveRootScheme maps an option value onto a scheme. The accepted aliases are
// deliberately generous because the value is typed by hand: the canonical BCP 47
// tags, the bare language codes, the common country-code shorthand, and the
// spelled-out names all name one vocabulary unambiguously. The second result
// reports whether the value was understood at all, so the evaluator can reject a
// typo instead of silently falling back.
func resolveRootScheme(value string) (AccountRootScheme, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "global", "en", "english":
		return SchemeGlobal, true
	case "zh-hans", "zh", "cn", "simplified", "simplified-chinese", "简体", "简体中文":
		return SchemeZhHans, true
	case "zh-hant", "zh-tw", "zh-hk", "tw", "hk", "traditional", "traditional-chinese", "繁體", "繁体", "繁體中文", "繁体中文":
		return SchemeZhHant, true
	case "ja", "jp", "japanese", "日本語", "日语", "日語":
		return SchemeJapanese, true
	case "ko", "kr", "korean", "한국어", "韩语", "韓語":
		return SchemeKorean, true
	}
	return SchemeGlobal, false
}

// RootScheme resolves the scheme selected by the account_root_scheme option. An
// absent value is the global scheme; the evaluator reports an unrecognized
// value as an option error so a typo never silently changes which roots a
// ledger accepts.
func (e Evaluation) RootScheme() AccountRootScheme {
	scheme, _ := resolveRootScheme(e.Options[AccountRootSchemeOption])
	return scheme
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
// spelling the ledger actually uses wins, so a localized ledger written in
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
// an ASCII uppercase lead byte, so a UTF-8 root is a dialect extension that a v3
// reader cannot load. Registering it keeps sourceexport's refusal to emit a
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

// activateLocalizedRoots registers the dialect extension for a non-global
// vocabulary. Upstream validates every option key and rejects
// account_root_scheme outright, so any ledger that selects one is not readable
// there regardless of which accounts it happens to use.
func (e *evaluator) activateLocalizedRoots(scheme AccountRootScheme) {
	if scheme == SchemeGlobal {
		return
	}
	if !contains(e.result.ActiveExtensions, "localized-account-roots") {
		e.result.ActiveExtensions = append(e.result.ActiveExtensions, "localized-account-roots")
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
// ambiguous -- a name_* override that collides with another type's localized
// alias does exactly that -- so the evaluator rejects the configuration instead
// of silently preferring one type.
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
