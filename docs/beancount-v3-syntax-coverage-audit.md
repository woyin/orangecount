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
