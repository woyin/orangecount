# OrangeCount web frontend

## Current state

`internal/web/assets/` still contains the legacy checked-in frontend consumed by the Go binary. `web/src/fava/` now contains the first Fava-derived shell transplant: pinned global CSS, fonts, header, mobile aside, page-title structure, and an OrangeCount adapter seam. Report routes remain staged-only until their adapter contracts and acceptance gates pass; do not mix the legacy and transplanted component trees.

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
`testdata/visual-baselines/`. Current prototype output is staged under
`web/staging/` and is not an accepted embedded asset.

## Route cutover rule

A route remains wholly legacy until the transplanted version passes contract/semantics, behavior/safety, visual/structural, and release-quality gates. A route never mixes legacy and transplanted component trees. After the final standard route passes, the legacy UI, prototype shell and migration flag are deleted.
