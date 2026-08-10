# Changelog

All notable changes to OrangeCount are documented in this file.

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
