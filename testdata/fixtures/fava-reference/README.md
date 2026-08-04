# Dense synthetic Fava reference fixture

This directory is generated exclusively from fixed non-private inputs by

tools/fixturegen. It is safe to mount into the isolated Fava 1.30.12
reference container and is never populated from an owner ledger.

- accounts: 87
- generated transaction directives: 304
- currencies/commodities: USD, EUR, JPY, GBP, CAD, CHF, AUD, SHARES, BTC, GOLD
- route states: reports, journal grouping, holdings/lots, documents, events,
  saved query, source metadata, editor diagnostics, and CSV import preview

Regenerate byte-identically with:

    go run ./tools/fixturegen -output testdata/fixtures/fava-reference

The editor and import files are intentionally outside the include graph. The
ledger entry itself is valid and has no network or private-file dependency.
