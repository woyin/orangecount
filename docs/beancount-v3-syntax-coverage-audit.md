# Beancount v3 Syntax Coverage Audit

## Verdict

OrangeCount is a Go-native reader and validator for **compatible Beancount v3
ledgers**. It still does not claim to be a complete v3 runtime: Python plugin
code is not executed and some v3 behavior is intentionally narrower than the
upstream implementation. At the syntax layer, the constructs identified in the
prior audit have now been implemented and covered by parser tests.

This audit uses the maintained upstream `v3` grammar as the language baseline,
not the older Vnext design notes. Upstream describes v3 as the maintained
branch and its parser grammar is the executable input-language authority:

- [Beancount v3 README](https://github.com/beancount/beancount/blob/v3/README.md)
- [Upstream v3 grammar](https://github.com/beancount/beancount/blob/v3/beancount/parser/grammar.y)

## Previously confirmed syntax gaps, now closed

| Construct | Change | Test |
| --- | --- | --- |
| `pushmeta key: value` / `popmeta key:` | `PushMeta`/`PopMeta` AST types plus parser support and source-preserved evaluation. | `TestParserAcceptsV3MetadataStackAndNoneValues` |
| Arithmetic number expressions, e.g. `1 + 2 USD`, `-(1 / 3) USD` | `parseNumberExpr` implements `+`, `-`, `*`, `/`, parentheses, unary signs, and exact rational evaluation for amounts, metadata values, custom values, and balance tolerances. | `TestParserEvaluatesNumberExpressions` |
| Metadata value `None` | `value()` returns `ValueNull` for `None`. | `TestParserAcceptsV3MetadataStackAndNoneValues` |
| Empty metadata value (`key:`) | `parseMetadata()` accepts an empty value as `ValueNull`. | `TestParserAcceptsV3MetadataStackAndNoneValues` |
| Transaction flag `#` | `isFlag()` accepts `#`. | `TestParserAcceptsHashFlagAndNoteTagsLinks` |
| `note` tags and links | `Note` retains `Tags` and `Links`. | `TestParserAcceptsHashFlagAndNoteTagsLinks` |

The expression parser was validated against the upstream parser on the same
files; upstream accepts the same constructs and evaluates the same arithmetic.

## Important distinction: parsing vs. evaluation

Even where OrangeCount parses a v3 directive, it does not promise equivalent
v3 execution. Plugin declarations are still parsed but intentionally not
executed; the evaluator emits a migration warning. Constructs that do not
affect accounting (such as `pushmeta`/`popmeta`) are preserved in the entry
stream without mutating account state.

The same boundary applies to options. Every `option "key" "value"` is parsed
and preserved verbatim in the evaluation's option map, but only
`operating_currency` (account/page valuation currencies) and `tolerance`
(default transaction-balancing tolerance) currently change evaluator
behavior. The five `name_assets`/`name_liabilities`/`name_equity`/
`name_income`/`name_expenses` options rename the root account types that
every statement, chart, and account-type query classifies against. An account
whose root is none of the five configured names receives `E-EVAL-ACCOUNT`
instead of being accepted and then dropped from reports. Options take effect
for the whole file regardless of where the `option` line sits, which is more
lenient than the upstream textual-order requirement and accepts a strict
superset of the ledgers upstream accepts here. A non-ASCII root name
is accepted as a dialect extension (`utf8-account-root`), and `export`
refuses such a ledger because a v3 reader cannot load it.

`option "account_root_scheme"` selects a built-in localized root vocabulary.
Each value makes five localized names exact equivalents of Assets, Liabilities,
Equity, Income, and Expenses:

| Value | Assets | Liabilities | Equity | Income | Expenses |
| --- | --- | --- | --- | --- | --- |
| `global` (default) | Assets | Liabilities | Equity | Income | Expenses |
| `zh-Hans` | 资产 | 负债 | 权益 | 收入 | 支出 |
| `zh-Hant` | 資產 | 負債 | 權益 | 收入 | 支出 |
| `ja` | 資産 | 負債 | 純資産 | 収益 | 費用 |
| `ko` | 자산 | 부채 | 자본 | 수익 | 비용 |

The Japanese set uses the statutory terms of Japanese GAAP and the Korean set
uses the K-IFRS terms. The English vocabulary stays usable under every scheme,
so an existing English-rooted ledger can adopt one without renaming accounts.
`global` rejects a localized root with `E-EVAL-ACCOUNT` instead of guessing at
its type, and a localized root from a different scheme does not leak into the
selected one.

The option value accepts the canonical BCP 47 tag plus common spellings (`cn`,
`tw`, `jp`, `kr`, the English names, and the native names); an unrecognized
value is `E-EVAL-OPTION` rather than a silent fallback.

The scheme is a dialect extension (`localized-account-roots`): `export` refuses
a localized ledger, and drops the `account_root_scheme` line when the value is
`global`, because upstream rejects unknown option keys.
`render_commas` is honored as a display option by both web
interfaces: displayed amounts group thousands while exact ledger values and
machine exports stay untouched. Currency-specific tolerance options
(`inferred_tolerance_default`, `inferred_tolerance_multiplier`) are recorded
but not yet applied to balance-assertion tolerance; the private reference
ledger (2,385 transactions, 80 balance assertions) validates with zero
diagnostics under both OrangeCount and Beancount v3, so the boundary has not
affected a real ledger. Revisit if a ledger relies on inferred per-currency
tolerance floors instead of assertion precision.

## Test evidence and scope

`go test ./...` passes. Parser tests in `internal/ledger/parser_test.go` cover
the constructs above. The compatibility fixture remains the curated
`beancount-v3-core` corpus; it documents the supported core rather than
claiming full upstream behavioral equivalence.

## Recommended compatibility wording

Keep the current wording: **“reader and validator for compatible Beancount v3
ledgers”**. Do not claim “Beancount v3 syntax superset” or “100% Beancount v3
parser” until a comprehensive differential suite against the upstream v3
parser covers every grammar production and every accounting behavior.
