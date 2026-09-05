# Changelog

All notable changes to OrangeCount are documented in this file.

## [Unreleased]

### Quality

- Reduced the repository's worst cyclomatic complexity from 26 to 16
  (planning profile/timeline/generate handlers, report dispatch, number
  expression splitter), preserving behavior under the full test suite.
- Raised statement coverage from 89.0% to 92.3% with new branch tests across
  web handlers, planning, ledger parsing/evaluation, query errors, dialect
  round trips, authoring writes, reports/charts, and the benchmark tools.
- Fixed a merged planning bug: expense-occurrence scanning now accepts both
  transaction entry forms, so the review page finds occurrences again.
- Planning preview retention now evicts deterministically (oldest insertion)
  with insertion-order tiebreaking, matching the quick-entry preview store's
  contract.

### Added

- Financial planning superset feature (merged from `feature/financial-planning`):
  ledger-embedded planning profiles via versioned `custom` directives
  (ADR-0047), liquidity planning with safe-to-spend scenarios and timeline
  (ADR-0048), planning adapter routes with reviewed writes published through
  the centralized authoring writer, and a transplanted-UI Planning report
  page.
- The ledger's `render_commas` option now groups thousands in every displayed
  amount of the built-in web interface (tables, journal postings, running
  balances, pivot cells, chart tooltips). Exact ledger values, sorting, CSV
  exports, and query results are unaffected.

### Changed

- Number-expression parsing merged into the directive dispatch parser while
  keeping single-literal amounts' source text in `Number.Raw`; `note`
  directives retain tags and links.

### Validated

- The private reference ledger validates with zero diagnostics under both
  OrangeCount and Beancount v3 (differential harness: 2,708 entries, no error
  classes), and every built-in report route was walked against it in both
  web interfaces. The option-boundary note in the syntax coverage audit
  documents `inferred_tolerance_default`/`inferred_tolerance_multiplier` as
  recorded-but-inert.

## [0.1.3] - 2026-08-10

### Changed

- Refactored the local server's import-preview retention and port-inspection
  boundaries without changing the user-facing accounting or UI behavior.
- Made the fixture generator's command-line transport independently testable.

### Quality

- Expanded unit and HTTP-contract coverage across the ledger evaluator,
  parser, reports, Fava adapter, CLI, and reviewed write workflows. The full
  Go suite now reports at least 90% statement coverage.

## [0.1.2] - 2026-08-10

### Added

- Per-account Beancount `booking "AVERAGE"` support with exact weighted-average
  cost basis, inventory diagnostics, and average-cost reporting.
- Average-cost columns across holdings views and an account-page cost evolution
  chart.

### Changed

- `orangecount serve` now defaults to `http://127.0.0.1:5000`.
- When the requested port is already in use, the CLI identifies its listener
  and asks for confirmation before stopping it.

### Fixed

- Account charts now render sparse series correctly when commodities begin in
  different periods.
