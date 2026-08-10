# Changelog

All notable changes to OrangeCount are documented in this file.

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
