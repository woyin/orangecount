# Optional Beancount v3 reference environment

The environment is pinned to Beancount **3.2.3** and BeanQuery **0.2.0**.
The primary accounting gate is `make parity-full`. It evaluates explicit
`v3` semantics against schema-2 golden files and exits nonzero for missing
goldens, unknown schema/oracle identity, or any unapproved difference. Cost
dates, nullable labels, posting flags/order, user metadata and duplicate
entries are retained. Exact numbers are never rounded to 12 digits. Only
the two named posting/header diagnostic location exceptions are allowed;
error class and multiplicity must still match.

`make golden-full-regen` uses the pinned external oracle on public fixtures
only. AVERAGE reduction fixtures are listed individually as extensions;
their independent exact arithmetic tests run with `make parity-full`.
The old `parity-diff` remains a legacy migration baseline and does not verify
the full accounting state.

This directory is development-only. The released OrangeCount binary does not
depend on Python, `uv`, or Beancount. Beancount is used as an external oracle
only, and its GPL-licensed source is never copied, linked, or bundled here.

To validate an owner-controlled ledger without putting it in this repository:

```sh
ORANGECOUNT_PRIVATE_LEDGER=/absolute/path/to/main.bean \
  ./tools/reference/check-ledger.sh
```

For a redacted OrangeCount-vs-Beancount v3 check, use
`tools/reference/differential.sh` with the same environment variable. The
script captures one immutable source graph, evaluates explicit v3 semantics
and feeds the same captured bytes to the pinned oracle. It compares complete
inventories, ordered postings, metadata, flags and exact monthly expense/income
rows, and exits nonzero for any difference. It emits only aggregated counts.
Source copies and sensitive JSON are removed with their temporary directory.
Known
intentional boundaries are recorded in
`docs/compatibility-ledger.json`; an empty difference is not a claim of full
Beancount or Fava parity.

For the real Swift/C read contract on the same input:

```sh
ORANGECOUNT_PRIVATE_LEDGER=/absolute/path/to/main.bean make verify-private-ledger
```

This opens the ledger once, checks exact native-currency root totals against
the oracle, validates counters/journal bounds and closes the session. It does
not change the ledger, benchmark it, or claim GUI acceptance. The default
core-only check reports `native_checked: false` explicitly.

The script refuses paths inside the repository and emits only redacted counts
and error type names. It does not copy the ledger or persist its contents.
The local `.reference/` directory is ignored by Git for optional owner config
and reports.

## Public typed query oracle

`make query-parity` checks the reviewed query-v1 golden in Go only. It fails
on any typed result or error-category mismatch, missing coverage, changed
input/manifest/include-graph identity, malformed schema or extra JSON document. Source
paths are normalized only for the manifest's explicit source columns.

`make query-oracle-check` executes the fixed Beancount 3.2.3 / BeanQuery 0.2.0
oracle and compares without modifying the golden. `make query-golden-regen`
is an explicit regeneration operation; review input, query and type/value
changes before accepting it. The generator accepts no private ledger path.
All eight registered public corpora are required (91 cases): default-source,
compound-values, price-weight, inventory-order, metadata-source, metadata-stack,
metadata-pad and dynamic-values. Query golden schema 2 records SHA256 for every captured source
file, including includes; the typed result contract remains version 1. A missing golden fails the
gate. Inventory values normalize their financial identity order, while query
row order and duplicates remain exact; nullable cost date/label are retained.
`--corpus` selects a registered public name in the oracle generator; no arbitrary
ledger input is accepted. `--golden` in the Go tool selects a single reviewed
file for inspection or external temporary fault injection.
Current Q01 scope and evidence: [query matrix](../../docs/query-capability-matrix.md)
and [dynamic/NULL verification](../../docs/evidence/q01-dynamic-null-query-2026-10-01.md).
