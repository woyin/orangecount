# Ship a Beancount v3 superset dialect with bidirectional compilation

OrangeCount will accept a dialect superset of Beancount v3 in ledger source files: every valid v3 file stays valid, and additional dialect lines offer a terse two-posting shorthand. Dialect lines are compiled, never rewritten in place: the dialect file is the canonical source of truth, and a separate `export` command emits a pure v3 snapshot for external tools. This is an experiment on `feature/dialect-superset`; main keeps the strict v3 contract until the experiment graduates.

## Grammar and detection

A dialect line is:

```
[DATE] [!] AMOUNT [CURRENCY] @source -> @destination ["payee"] [: narration] [#tag] [^link]...
```

- Detection is a natural syntactic gap: a date whose second token is a number, or a top-level line beginning with a number, is invalid v3 today. No valid or standard-parseable line can be misread as dialect.
- The arrow carries value flow: a positive amount moves from the left account to the right account; the compiler emits the signs.
- Account endpoints resolve through three levels: full account name, then declared `orangecount.quick-account.v1` aliases (date-effective per ADR-0043), then a unique tail-segment match across all opened accounts. Ambiguity or no match is a per-line error listing candidates; the compiler never guesses.
- An omitted currency uses the ledger's single `operating_currency`, else errors. An omitted narration defaults to "消费". A quoted string after the arrow segment is the payee. The flag may be `*` (default) or `!`.
- A missing date uses block anchoring: the nearest preceding dialect line that carries a date, within the same file. Anchoring never reads "today" and never inherits from non-dialect directives, so every compile of the same source yields the same ledger. No anchor is an error.
- Template invocation stays out of the dialect (template expansion depends on ledger state and would break reproducible compilation).

## Compilation and export

- In-process builds compile dialect lines into ordinary transactions during snapshot construction, so all reports, queries, and validation behave exactly as on a standard ledger (E3).
- `orangecount export source.bean -o out.bean` writes a pure v3 snapshot for Fava, bean-query, and future tools (E1). Export output is a disposable artifact: it must never be hand-edited, and changes go only to the dialect source. A synchronized on-disk export file (E2) is rejected — it would create a second truth and a sync war.
- Dialect line failures are first-class `E-DIALECT-*` diagnostics pointing at the exact line. A ledger whose dialect lines fail to compile does not publish a snapshot (FD-0004 semantics).

## Reverse conversion (dialectize) is a filter, not a translation

`orangecount dialectize v3.bean -o dialect.bean` converts a v3 ledger into the dialect by filtering: a transaction becomes a dialect line only when it is exactly two postings, opposite signed amounts in one currency, no cost, no price, no metadata, flag `*` or `!`, and a non-empty narration (an empty narration would be re-materialized as the "消费" default and mutate the record). Everything else is preserved byte-for-byte in standard syntax. The dialect is a shorthand for the common shape, not a second full language; multi-leg, cost, and metadata transactions never enter it.

Two property tests lock round-trip safety mechanically: for any v3 ledger, `dialectize` then `export` must build a snapshot with identical account balances to the original; and re-running the round trip must reach a fixpoint (the second export is byte-stable). Layout of converted lines is rewritten, which forfeits git blame for those lines; accepted for the experiment.

## Consequences

The canonical ledger is now readable only by OrangeCount; ecosystem tools consume exports. The compatibility contract gains a second surface: OrangeCount must remain a correct v3 reader (unchanged) and a correct dialect compiler (new). The superset keeps every existing write path valid: editor saves, quick entry, and standard syntax inside dialect files all continue to work unchanged.
