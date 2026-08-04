# Controlled Fava visual and browser harness

## Purpose

The harness establishes Fava 1.30.12 as an external visual authority and compares OrangeCount against it using only deterministic synthetic ledger data. It is development and CI tooling; it never participates in the embedded Go runtime.

The Playwright harness now has a controlled reference path and a separate
OrangeCount path. It has produced candidate-only Fava captures, but no capture
is an approved product baseline until the product owner reviews it.

## Current status (Prerequisite Phase 0 tooling complete; visual approval pending)

- The host remains outside the supported Node 22 range; reference capture runs Node 22 inside the OCI image.
- Fava 1.30.12 is isolated from the host Bison/Python toolchain and runs with no runtime network access.
- `tools/fixturegen` produces the committed dense synthetic fixture (87 accounts, 304 transaction directives, ten currencies/commodities) from fixed non-private inputs.
- The reference suite exercises all four English Chromium cells and currently captures five representative routes (20 candidate screenshots).
- No screenshots exist under `testdata/visual-baselines/`; candidate captures under `testdata/visual-candidates/fava-reference/` are not accepted evidence.
- `npm --prefix web run check:reference-output` verifies the candidate environment lock, fixture hash, and complete screenshot matrix without approving pixels.

A skipped reference suite can never satisfy a route gate. The representative
capture is intentionally not a claim that any OrangeCount route is accepted.

## Controlled OCI environment

The baseline generator pins:

- Fava `v1.30.12` at commit `aa7538e8971252c9efc52c8a516a3a77d604553f`;
- compatible Python, Beancount, Bison, and build libraries;
- Node 22 LTS and the repository's locked npm dependencies;
- Playwright 1.52.0 and the Debian Chromium executable recorded in the environment lock;
- Fira Sans, Fira Mono, Source Code Pro, and the selected CJK fallback;
- English locale, UTC, reduced motion, a deterministic scale factor, and fixed desktop/narrow viewports.

The container mounts only the synthetic fixture and an explicit candidate-output directory. It does not inspect an existing Fava server, private ledger, browser profile, screenshot, or session. Runtime networking is denied after dependencies are built.

Ordinary `make build`, `make test`, and the released OrangeCount binary require no container, Node, Python, browser, or network.

## Fixture tiers

### Compact fixture

The compact fixture drives exact loaded, empty, loading, unavailable, error, stale, validation, concurrency, rollback, and containment cases.

### Synthetic reference ledger

`tools/fixturegen` generates the dense deterministic ledger with 87 nested
accounts, ten currencies/commodities, multi-currency accounts, missing and
partially valued paths, Unicode labels, enough transactions for Journal
scrolling/grouping, flags, tags, links, metadata, lots, documents, events,
saved queries, editor errors, and import candidates.

It is generated from fixed non-private inputs and locked by the candidate
content hash. It must never be constructed by anonymizing a private ledger.

## Baseline matrix

Every in-scope route/state has English Chromium candidates for:

1. desktop/light;
2. desktop/dark;
3. narrow/light;
4. narrow/dark.

`docs/fava-route-state-manifest.md` is the source of truth for coverage. Simplified Chinese runs the same structural and behavior scenarios, checking component identity, information hierarchy, control/focus order, keyboard behavior, wrapping, overflow, table relationships, and responsive transitions without cross-language pixel comparison.

WebKit and Firefox run supported behavior, accessibility, and serious-layout-regression flows. Chromium alone is the strict visual authority.

## Baseline lifecycle

1. Start a new isolated Fava reference with the synthetic reference ledger.
2. Capture candidate Fava screenshots and route/state metadata in the controlled environment.
3. The OpenAI Codex `gpt-5.6-luna` visual agent analyzes structure, density, typography, themes, responsive behavior, and candidate diffs.
4. The coordinating Agent audits provenance, fixture hash, state setup, masks, and test evidence.
5. The user (product owner) explicitly approves or rejects each new baseline or Approved Fava deviation.
6. Only approved candidates replace committed baselines.
7. OrangeCount screenshots are compared to the approved Fava baselines; tests never update the expected images automatically.

The implementing agent cannot approve its own visual evidence.

## Difference rules

- Layout, dimensions, typography, color, spacing, density, control placement, table/chart composition, missing controls, and missing states are failures unless explicitly approved.
- Browser rasterization noise may use a narrow comparison rule established from stable repeated Fava captures.
- Masks are limited to named, truly nondeterministic regions. Broad masks and arbitrary global similarity thresholds are forbidden.
- Accounting-result differences are rendered through the same Fava composition and handled through the parity-authority rule.
- Every accepted difference is registered in `docs/fava-approved-deviations.md`.

## Four-layer route evidence

A route's harness output must link:

1. adapter/semantic contracts;
2. browser behavior and safety flows;
3. English visual comparisons and Chinese structural checks;
4. performance, accessibility, cross-browser, offline/CSP, provenance/license, and release checks.

A route cannot switch to the transplant until all four are present and passing.

## Target commands

Prerequisite Phase 0 will make these command families executable and deterministic:

```sh
# Install locked development dependencies under Node 22.
npm --prefix web ci
npm exec --prefix web playwright install chromium firefox webkit

# Build source assets and verify checked-in embedded output.
npm --prefix web run build
npm --prefix web run build:check
npm --prefix web test

# Generate Fava candidate baselines in the controlled OCI environment.
npm --prefix web run visual:reference
npm --prefix web run check:reference-output

# Compare OrangeCount with approved baselines and run browser flows.
npm --prefix web run visual:test
npm --prefix web run browser:test
```

The reference and candidate-completeness scripts are executable. `visual:test`
remains non-release evidence until the user approves committed baselines and
all route/state gates are implemented.

## Privacy rules

- Only deterministic synthetic ledger and document inputs may be mounted.
- No trace, video, DOM dump, response body, local absolute path, or attachment content from a private session is retained.
- Approved committed screenshots must be demonstrably synthetic.
- Existing local Fava/OrangeCount servers and browser profiles are never reused.
- Private-ledger release smoke runs locally and records only pass/fail outside repository artifacts.
