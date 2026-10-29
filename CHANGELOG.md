# Changelog

All notable changes to OrangeCount are documented in this file.

## [0.3.11] - 2026-10-29

0.3.10 preserved the file it replaced, but the single backup slot was reused by
every later export: replacing the same target a second time moved the previous
export over the slot and destroyed the Finder alias kept there.

The tool now records what it publishes. `<target>.orangecount.published` holds the
SHA-256 of the bytes written there, so a target that still matches that record is
recognized as this tool's own previous output and is replaced directly. Only
content the tool did not write is preserved, and the backup slot is never
overwritten: when an earlier version already occupies it, the export stops and
names the path. Repeated exports therefore leave one record and at most one
backup, and a preserved alias survives them.

### Changed

- Replacing a file this tool wrote no longer creates a backup.
- An occupied backup slot stops the export instead of being overwritten.

### Fixed

- A second export to the same path destroyed the Finder alias that the first
  export had preserved.

## [0.3.10] - 2026-10-29

0.3.9 made export work from a build without cgo, but it still had to guess
whether a destination was safe to replace. A Finder alias is an ordinary file
whose alias record lives in an extended attribute, and only a cgo build can read
that attribute, so a build without cgo cannot recognize one. The guess that
shipped in 0.3.9 replaced an alias silently.

Replacement no longer guesses. The previous file is renamed to
`<target>.orangecount.bak` and the new file is linked into the freed name; if the
link fails, the backup is renamed back. Nothing inspects metadata, so an alias,
or anything else that occupied the name, survives. This is the same backup name
the web editor already uses for ledger writes.

Because the safety property no longer depends on a cgo-only primitive, the
package has one implementation. The cgo and build-tag variants, including the
macOS alias probe, are deleted, so a local build and a release binary run the
same code on every platform.

### Changed

- Replacing an existing single file leaves the previous file at
  `<target>.orangecount.bak` instead of deleting it.
- The backup slot is never overwritten when it is a symlink, a hardlink or not an
  ordinary file.

### Fixed

- A Finder alias at the export target is preserved instead of being replaced.

## [0.3.9] - 2026-10-29

Every binary the release workflow publishes is built with CGO_ENABLED=0, and
those binaries refused to export an include graph: they answered every export
with "exclusive tree publication requires the macOS or Linux cgo build". The
export feature therefore did not work on any shipped artifact.

### Fixed

- `orangecount export` now publishes an include tree from builds without cgo.
  The named target is refused when it already exists, then the staged tree is
  moved into place with `rename(2)`, which keeps publication all-or-nothing.
  The check and the rename are separate operations; that boundary is the same
  one `docs/source-export.md` already records for concurrent processes.
- macOS without cgo refused every single-file replacement because the Finder
  alias check needs cgo. It now refuses the targets the standard library can
  identify (symbolic links and non-regular files). A Finder alias is a regular
  file with an xattr, so it cannot be detected there; the gap is recorded in
  `docs/source-export.md` instead of being implied.
- `/api/v1/status` reported a hardcoded `0.1.0-dev` while the CLI reported the
  build version. The endpoint now reports the version the process runs.

### Quality

- `go test ./...` no longer needs the Python oracle: the genprediff end-to-end
  test skips when `uv` is absent, which is what made CI red on every push.
- `function.today.0` compared a frozen date and failed the day after capture.
  The generator now derives the dynamic marker from the query text, the
  comparator rejects an unrecognized marker instead of comparing literally, and
  a guard test pairs clock queries with their markers.

## [0.3.8] - 2026-10-28

This release publishes the generation of OrangeCount that was developed after
0.3.7 but never shipped. It changes the CLI surface: `check`, `query`,
`serve` and `export` now take `--semantics legacy|v3`, and `query` gains the
`typed-json`/`typed-csv` formats. Every flag 0.3.7 accepted is still accepted,
so a script written against 0.3.7 keeps running.

### Breaking

- A ledger whose account root is not one of the five configured account types
  is now rejected with `E-EVAL-ACCOUNT`; 0.3.7 accepted such an account and
  then silently omitted it from every report. A name such as `Acme:Cash`
  therefore changes from "loads with missing numbers" to "does not load". This
  matches upstream Beancount v3, which rejects the same name while parsing.
  Fix by moving the account under a configured root, declaring the root with
  the matching `name_*` option, or selecting a localized scheme.

### Added

- `--semantics legacy|v3` on `check`, `query`, `serve` and `export`, and the
  `compare-semantics` subcommand that reports every accounting difference
  between the two profiles (`--details` for the full list, `--fail-on-change`
  for use in a pipeline). The default profile stays `legacy`.
- Typed query formats `--format typed-json|typed-csv` over the declared
  BeanQuery subset, alongside the existing `json`/`csv` compatible formats.
  The two engines expose different column sets on `accounts`: the typed table
  has only `account`, the compatible one has `account`, `currency`,
  `balance` and `opened`.
- `option "account_root_scheme"` selects a localized root vocabulary.
  `zh-Hans` (资产/负债/权益/收入/支出), `zh-Hant` (資產/負債/權益/收入/支出),
  `ja` (資産/負債/純資産/収益/費用) and `ko` (자산/부채/자본/수익/비용) each make
  the localized names exact equivalents of the English roots, and the English
  names stay accepted under every scheme. A localized root outside `global` is
  a dialect extension, so `export` refuses such a ledger.
- The five `name_*` options now actually rename the root types that statements,
  charts and account-type queries classify against. Previously the values were
  stored and ignored, which let a renamed ledger load and then disappear from
  every report; an account whose root is not a configured type now reports
  `E-EVAL-ACCOUNT` instead.
- `E-EVAL-ACCOUNT` diagnostic with bilingual repair guidance.
- Project skills under `.agents/skills` (`orangecount-ledger`,
  `orangecount-authoring`, `orangecount-workbench`) and
  [docs/user-guide.md](docs/user-guide.md), a task-ordered manual.
- `--locale zh-CN` is accepted by `check`, `query`, `serve` and `help`.

### Changed

- Account names with UTF-8 components are accepted end to end, including the
  authoring and quick-entry write paths. A UTF-8 root name is accepted only
  under a localized scheme and is registered as the `utf8-account-root`
  dialect extension.
- A `custom` directive value such as `资产:钱包` is now typed as an account
  rather than a currency, so consumers that expect an account stop dropping it.
- The five `name_*` options join the export option policy. A colliding
  `name_*` value or a duplicate root spelling is `E-EVAL-OPTION` instead of a
  silent preference for one type.
- `option "account_root_scheme" "global"` is dropped from an exported snapshot
  because upstream rejects an unknown option key, and the reload fingerprint
  normalizes it out so the removal is not reported as an accounting change.
- Dialect transpilation now completes on a mixed real-world ledger:
  `orangecount dialectize` and `orangecount export` handle aggregated and
  per-leg postings, same-date entries and a display-only option.
- `render_commas` is accepted by the export gate as a proven display-only
  option, recorded in an explicit option policy table instead of a flat
  allow-list; unproven options stay refused.
- The source-export fingerprint compares accounting identity rather than leg
  layout: same-account legs merge by summing their units (ADR-0045 allows an
  aggregated line to split back into its per-leg components), and entries
  sharing a date compare in canonical content order because a conversion
  rewrites source line numbers. The cross-date entry sequence, the
  account/price/option sets and every posting value stay sensitive, so a
  genuine accounting change still fails `E-EXPORT-RELOAD`.
- Number-expression parsing merged into the directive dispatch parser while
  keeping single-literal amounts' source text in `Number.Raw`; `note`
  directives retain tags and links.
- `SELECT DISTINCT` now enforces a unique-key budget on both the typed and the
  legacy entry points. Dense `PIVOT BY` memoizes row and column identities;
  behavior is unchanged.
- The native desktop clients are retired; the released artifact is the Go
  binary with the transplanted web UI embedded.

### Fixed

- Made the query parity gate location-independent: regenerated the presets
  and source-order goldens with declared `source_columns` (paths normalize to
  fixture-relative form), and taught both golden normalizers to compare
  non-absolute filename metadata (ledger overrides such as
  `filename: "override.bean"`) as portable literals. The gate previously
  hard-coded the developer checkout path into golden data and failed 6 of 321
  checks on any other clone.
- Fixed a merged planning bug: expense-occurrence scanning now accepts both
  transaction entry forms, so the review page finds occurrences again.
- Planning preview retention now evicts deterministically (oldest insertion)
  with insertion-order tiebreaking, matching the quick-entry preview store's
  contract.
- An unconsumed pad directive is an error (`E-EVAL-PAD-UNUSED`), matching the
  upstream `PadError`; it was previously a warning, which let a ledger that
  Beancount rejects look valid here.

### Validated

- All 366 inventoried upstream capabilities are verified, enforced by six
  machine gates in `tools/paritygates` (`-gate all`), including a generative
  differential against the pinned Beancount v3 oracle.
- The Fava cutover is gated by five machine gates in `tools/favagates`: every
  standard route passes its four-layer gate, the transplanted UI is the only
  standard surface, the legacy UI is gone, no unapproved visual difference
  remains, and the visual baseline is committed (80 reference screenshots at
  four viewports).
- FD-0011 is registered as the approved deviation for the visual gap; the
  measured pixel difference is retained as evidence, not as a gate.
- FD-0010 approved in `docs/fava-approved-deviations.md` (strict week-date
  rejection instead of strptime rolling).
- The private reference ledger validates with zero diagnostics under both
  OrangeCount and Beancount v3 (differential harness: 2,708 entries, no error
  classes), and every built-in report route was walked against it in both
  web interfaces. The option-boundary note in the syntax coverage audit
  documents `inferred_tolerance_default`/`inferred_tolerance_multiplier` as
  recorded-but-inert.

### Quality

- Reduced the repository's worst cyclomatic complexity from 26 to 16
  (planning profile/timeline/generate handlers, report dispatch, number
  expression splitter), preserving behavior under the full test suite.
- Raised statement coverage from 89.0% to 92.3% with new branch tests across
  web handlers, planning, ledger parsing/evaluation, query errors, dialect
  round trips, authoring writes, reports/charts, and the benchmark tools.
- Removed 23 duplicate Claude Code skill symlinks and renamed `CLAUDE.md` to
  `AGENTS.md`.

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
