# OrangeCount web frontend

## Current state

`internal/web/assets/transplanted/` holds the only UI the Go server serves: `index.html`, `app.js`, `app.css` and `manifest.json`. The directory is committed so `make build` and `make test` work without Node, and no environment flag or build tag selects a different asset tree. `web/src/fava/` contains the Fava-derived shell, report, Journal, Query, Editor, Import, and adapter surfaces; route-specific acceptance is still tracked separately.

The authoritative migration plan is [`../docs/fava-frontend-transplant-plan.md`](../docs/fava-frontend-transplant-plan.md).

## Target source boundary

```text
web/src/fava/         selected Fava 1.30.12-derived frontend units
web/src/orangecount/  OrangeCount adapter client, security integration and localization
```

Selected Fava files retain required notices and have upstream path/revision/hash rows in `docs/fava-provenance-inventory.md`. The complete Fava checkout remains outside the repository.

The source build emits deterministic static assets that are copied into `internal/web/assets/` and embedded by Go. Node is required only for development and source-asset verification; the released application remains an offline Go binary with no runtime Node, Python, container, browser, font CDN or network dependency.

## Required development models

Herdr coordinates two user-selected implementation agents whose work returns to the current coordinating Agent:

- code agent: Pi; model: DeepSeek V4 Flash supplied by WoYin, selected as `WoYin/clinepass/cline-pass/deepseek-v4-flash`;
- visual: OpenAI Codex `gpt-5.6-luna`.

The code model owns Go contracts, parsers, semantic projections, exporters, write safety and non-visual tests. The visual model owns Fava-derived frontend adaptation, styles, fonts/assets, responsive behavior, browser baselines and visual fixes. Writer ownership is serialized; neither model commits or integrates independently.

Both configurations have been verified through Herdr. The visual agent starts as Codex `gpt-5.6-luna`; the code agent starts with Herdr `kind=pi`, and its Pi footer reports `(WoYin) clinepass/cline-pass/deepseek-v4-flash` with high thinking. Each task must still verify its live identity. The separate `deepseek/deepseek-v4-flash` catalog entry is not interchangeable; neither provider nor model may be silently substituted.

## Development and Phase 0 commands

Under a supported Node 22 environment, the static checks are:

```sh
npm --prefix web ci
npm --prefix web run build
npm --prefix web run build:check
npm --prefix web test
npm --prefix web run check:phase0
```

The deterministic dense reference fixture is generated without private-ledger
input:

```sh
go run ./tools/fixturegen -output testdata/fixtures/fava-reference
npm --prefix web run visual:reference
npm --prefix web run check:reference-output
```

Reference screenshots are candidate-only files under
`testdata/visual-candidates/fava-reference/`; they never update
`testdata/visual-baselines/`. `npm --prefix web run build:embedded` builds the shell into
`web/staging/` and copies it into `internal/web/assets/transplanted/`; that
command is the only way to refresh the embedded UI, and re-running it on
unchanged sources produces byte-identical files.

## Route cutover rule

A route counts as complete only when the transplanted version passes contract/semantics, behavior/safety, visual/structural, and release-quality gates. A route never mixes component trees. The legacy UI, the prototype shell and the migration flag are deleted; `go run ./tools/favagates` verifies that they stay deleted.
