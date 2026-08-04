# Sanitized Fava visual fixture

This fixture is synthetic and deterministic. It is not derived from a private
ledger. It exists for route, state, keyboard, document-root, editor-diagnostic,
and import-preview checks.

| Fixture area | Intended coverage |
| --- | --- |
| `main.bean`, `accounts.bean`, `activity.bean` | English shell data, nested accounts, multi-currency natural balances, a priced commodity, an unpriced currency, transactions, directives, events, saved query metadata, and source links |
| `documents/` | Safe in-root document attachments only |
| `editor/invalid.bean` | Deterministic editor diagnostics without entering the main include graph |
| `import/import-candidate.csv` | Local import preview input; it is not included by the ledger |
| `saved-queries/` | Synthetic saved-query editor content |

The entry ledger is valid. The editor file is intentionally invalid and the
import file is intentionally separate so each workflow can exercise its own
state without changing the evaluated snapshot.

## Offline checks

```sh
go run ./cmd/orangecount check testdata/fixtures/fava-visual/main.bean
go test ./internal/compat -run FavaVisualFixture
```

A local server may be started for manual checks only with the entry ledger and
`documents/` as its explicit document root. Browser and visual procedures are
defined in `docs/fava-visual-harness.md`; no private or pre-existing ledger
session is an acceptable input.
