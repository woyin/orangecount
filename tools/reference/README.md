# Optional Beancount v3 reference environment

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
script stores intermediate JSON only in a temporary directory, compares
normalized counts and diagnostic classes, and emits no ledger content. Known
intentional boundaries are recorded in
`docs/compatibility-ledger.json`; an empty difference is not a claim of full
Beancount or Fava parity.

The script refuses paths inside the repository and emits only redacted counts
and error type names. It does not copy the ledger or persist its contents.
The local `.reference/` directory is ignored by Git for optional owner config
and reports.
