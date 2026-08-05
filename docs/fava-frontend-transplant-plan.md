# Fava 1.30.12 rendering-fidelity execution plan

## Status and outcome

This is the authoritative execution plan for replacing OrangeCount's hand-built
Fava approximation with a selective transplant of the pinned Fava 1.30.12
frontend. The current clean-room Svelte shell is an architectural prototype,
not an accepted parity implementation, and must not receive further visual
polish.

The migration is complete only when the entire Fava standard surface is served
by the transplanted frontend, every route passes its four-layer route gate,
there are no unapproved visual differences, the legacy UI and migration flag
are deleted, and OrangeCount remains a Go-native, offline, loopback-only
application.

## Non-negotiable decisions

1. Fava 1.30.12 is the visual and interaction authority. Beancount v3 and the
   OrangeCount core remain the accounting, valuation, booking, and BeanQuery
   authority.
2. Visual work starts from selected Fava components, composition, CSS, fonts,
   assets, routing, state, and interactions. A clean-room rewrite is an
   exception, not the default.
3. Under controlled English Chromium conditions, every visual difference must
   either be removed or recorded as an Approved Fava deviation. An arbitrary
   screenshot-similarity percentage is not an acceptance rule.
4. English is the strict visual baseline. Simplified Chinese uses the same
   components, information architecture, control order, focus order, and
   responsive behavior, with language- and CJK-font-driven layout differences.
5. Chromium is the pinned visual authority. WebKit and Firefox are supported
   by behavior, accessibility, and serious-layout-regression checks rather than
   cross-browser pixel comparison.
6. The Fava standard navigation remains free of OrangeCount-exclusive pages.
   Such pages are frozen during migration and isolated under a clearly labeled
   OrangeCount extension area.
7. The frontend changes as little as possible. The private Go Fava-shaped
   adapter absorbs data-shape and transport differences without owning ledger
   semantics.
8. Every built-in user-visible Fava option that does not depend on an excluded
   capability is implemented with Fava-compatible interface effects and
   precedence. Unsupported excluded behavior is visible as an approved
   deviation, never silently ignored.
9. Python importers, plugin pages, user extensions, the Python/Beancount
   runtime, and Fava's public HTTP API remain excluded.
10. Editor, Import, Add Entry, source-slice, statement, and document operations
    are in scope through the Reviewed write workflow. Fava behavior never
    weakens OrangeCount's atomicity, backup, validation, snapshot, privacy, or
    document-root controls.
11. Fava's built-in budget behavior is implemented as a read-only presentation
    projection over `custom "budget"` entries; it never changes accounting
    semantics.
12. Query CSV, XLSX, and ODS export, filtered Journal Beancount export, and
    print behavior are in scope and implemented without Python.
13. Fava FQL and Fava time-filter syntax are complete UI contracts, distinct
    from BeanQuery and ledger semantics.
14. Journal preserves Fava's private HTML presentation contract: the Go
    adapter renders strictly escaped Fava-compatible Journal markup instead of
    replacing the page with a clean-room JSON/Svelte implementation.

The governing ADRs are ADR-0022 and ADR-0024 through ADR-0039. ADR-0023 is
superseded by ADR-0030; ADR-0004 and ADR-0007 describe earlier boundaries that
ADR-0022 superseded.

## Completion definition

The project may claim completion only when all of the following are true:

- A single route registry classifies every Fava 1.30.12 built-in route, modal,
  menu, keyboard path, loaded state, empty state, loading state, unavailable
  state, error state, and stale state as transplanted or explicitly excluded.
- Every in-scope route uses selected Fava-derived frontend units and only the
  private Go adapter plus the immutable OrangeCount snapshot.
- Every route has adapter-contract, behavior, visual/structural, and release
  evidence and has passed all four gates defined below.
- English passes the pinned Chromium matrix for desktop/narrow and light/dark.
  Simplified Chinese passes the structural, overflow, focus, keyboard, and
  responsive invariants.
- WebKit and Firefox complete the supported route flows without serious layout
  failure.
- The synthetic reference ledger demonstrates production-like density,
  multi-currency rows, missing valuation paths, long text, paging, lots,
  documents, events, diagnostics, editor failures, and import candidates.
- Reports, filters, queries, exports, and writes retain exact OrangeCount/v3
  results even where Fava's Python results differ.
- All Approved Fava deviations are recorded and explicitly approved by the
  user (the product owner). Implementing agents cannot approve their own baseline changes.
- `make fmt vet test race license build`, frontend unit/build checks, browser
  behavior tests, visual comparisons, accessibility checks, provenance checks,
  and deterministic artifact checks pass.
- No runtime network, Node, Python, container, or CDN dependency is introduced.
- The migration flag, legacy routes, legacy DOM, obsolete assets, and current
  prototype shell have been removed.

## Authority and boundaries

```text
Pinned Fava 1.30.12 reference mirror
  ├─ selected attributed frontend source
  ├─ controlled Fava visual baseline
  └─ observable behavior and interaction authority

Transplanted Fava frontend
  │ private loopback requests and HTML fragments
  ▼
Go Fava-shaped adapter
  ├─ Fava-compatible DTOs, errors, status, URL state, and Journal markup
  └─ no public API or accounting ownership
  ▼
report / query / source / snapshot / ledger
  ├─ Beancount v3 semantics and exact values
  └─ reviewed write workflow and active-snapshot publication
```

### Frontend source boundary

Selected upstream-derived units live under `web/src/fava/`, preserving useful
upstream path relationships and required MIT/OFL/BSD notices. OrangeCount-only
integration code lives under `web/src/orangecount/`. Every selected or adapted
unit has a provenance row with upstream path, pinned revision, upstream hash,
local path, modification summary, contract rows, license placement, and
verification evidence.

A Fava-derived component may be modified only for an adapter boundary, an
excluded capability, Simplified Chinese support, or an Approved Fava
deviation. OrangeCount integration code must not be scattered through the
upstream-derived tree merely for convenience. The full Fava checkout remains
outside the repository.

The current `web/src/fava/` clean-room shell files are prototype files despite
their directory name. Wave 1 replaces or relocates them; they are not evidence
that a Fava source unit has been transplanted.

### Go adapter boundary

The private adapter owns:

- the bootstrap/ledger-data envelope, `mtime`, changed, errors, locale, theme,
  options, route state, account details, commodity names, and precisions;
- complete FQL and Fava time-filter parsing into typed UI filter values;
- route-specific report, account, Journal, holdings, commodity, document,
  event, statistics, query, editor, import, options, help, and error contracts;
- Fava-compatible status, validation, pagination, sorting, cancellation,
  source-anchor, empty, unavailable, stale, and error shapes;
- strictly escaped Fava-compatible Journal and account-Journal markup;
- exact CSV/XLSX/ODS and Beancount export presentation;
- conversion from adapter requests to typed OrangeCount report/query/source
  inputs and adaptation of exact results back to frontend-required shapes.

It does not own accounting calculation, booking, mutable snapshot state,
source authorization, document-root policy, import parsing, file replacement,
or snapshot publication.

### Reviewed write boundary

All Editor, Source Slice, Add Entry, Import, Statement, and Document operations
must be explicit. Ledger publication requires preview or validation, atomic
replacement, a recoverable backup, re-evaluation, and successful publication
of a complete snapshot. A failed write or re-evaluation retains the previous
valid snapshot and a recoverable edit. Document operations additionally require
normalization, containment re-checks, confirmation, and recoverable handling of
cross-file partial failure.

## Execution model and role separation

Per ADR-0039, transplant work is executed by a single session model; the
earlier requirement for two distinct externally supplied models is retired.
Role separation survives as process discipline rather than model identity:

| Role | Ownership |
| --- | --- |
| Implementation | Go semantic projections, adapter contracts, FQL/time parsers, budgets, exporters, write pipeline, fixture semantics, unit/contract tests, build and license tooling; Fava source adaptation, CSS/fonts/assets, frontend composition and interaction, responsive behavior, baselines, visual diffs, visual fixes, zh-CN structural review |
| Coordinator | task decomposition, handoffs, integration, full verification, documentation, evidence synthesis, and reporting to the user |
| Final visual authority | user (product owner): approval of baseline updates and Approved Fava deviations |

The implementer must not certify Gate 3 visual parity for its own changes,
may not change ledger semantics while doing visual work, may not weaken
write/security behavior, and cannot approve its own baselines. The
coordinator resolves conflicts using the parity-authority rule. Concurrent
writer phases on the same files remain serialized; agents do not delegate
further and do not merge or commit independently.

## Reference and evidence system

### Controlled reference environment

A development-only OCI environment pins:

- Fava tag and commit;
- Python, Beancount, Bison, and system-library versions;
- Node 22 LTS, Playwright, Chromium, and browser revision;
- Fira Sans, Fira Mono, Source Code Pro, and CJK fallback versions;
- English locale, UTC, reduced motion, deterministic scale factor, and fixed
  desktop/narrow viewports.

It mounts only the synthetic reference ledger and an explicit output directory.
It never enters the release binary. Ordinary Go build/test commands require no
container, Python, Node, or browser.

### Synthetic fixtures

Two non-private fixture tiers are required:

1. **Compact fixture** — precise contract, empty, error, unavailable, stale,
   validation, rollback, and security states.
2. **Synthetic reference ledger** — deterministic production-like density,
   targeting 80–100 nested accounts, 6–10 currencies/commodities, multi-
   currency accounts, missing and disconnected price paths, long Unicode
   labels, enough transactions for paging, all directive/flag families, tags,
   links, metadata, lots, documents, events, saved queries, editor errors, and
   import candidates.

A fixed generator input and committed content hash make the dense ledger
reproducible. It must not be derived by anonymizing a private ledger. A private
reference ledger remains transient local smoke evidence only.

### Baselines

Approved Fava screenshots generated solely from the synthetic reference ledger
are committed under the designated baseline directory. Candidate updates never
overwrite approved images automatically. Masks are permitted only for named,
truly nondeterministic regions; global tolerance cannot hide layout, color,
spacing, typography, density, or missing-state differences.

The strict English matrix is:

- desktop/light;
- desktop/dark;
- narrow/light;
- narrow/dark.

Simplified Chinese reuses the route/state manifest and verifies component
identity, information hierarchy, table relationships, control and focus order,
keyboard behavior, wrapping, overflow, and responsive transitions without
cross-language pixel comparison.

### Approved deviations

`docs/fava-approved-deviations.md` is the sole registry. Each entry records an
ID, route/state, Fava behavior, OrangeCount behavior, permitted reason,
category, affected regions, tests, owner, approver, permanence/expiry, and
baseline impact. Only semantic authority, security, data integrity, privacy,
or accessibility can justify a deviation. Product preference, convenience,
modernization, or subjective improvement cannot.

## Route scope summary

`docs/fava-route-state-manifest.md` is the sole canonical coverage and status
registry. Prerequisite Phase 0 makes it machine-checkable and complete. The
following is only a page-family summary; nested modal, menu, download,
keyboard, and state entries must be changed in the canonical manifest first:

| Surface | Required scope |
| --- | --- |
| `/` | Fava-compatible default-page redirect/landing and ledger title |
| `/income_statement` | tree report, intervals, conversion, charts, budgets where visible |
| `/balance_sheet` | tree report, natural currencies, conversion, charts |
| `/trial_balance` | tree report, currency legend, hierarchy modes, fallback |
| `/journal` | FQL/time filtering, pagination, sorting, directive types, expansion, context, export |
| `/account/<name>` | account details, up-to-date state, charts, budgets, running balance, account Journal |
| `/holdings/by_<key>` | all built-in grouping variants, cost/value/units and unavailable valuation |
| `/commodities` | metadata, names, precisions, prices, history and filters |
| `/documents` | list, preview, upload, attach, move, delete, safe source/document links |
| `/events` | filters, source navigation, empty/error states |
| `/statistics` | deterministic metrics, charts and accessible tables |
| `/query` | editor, run, saved state, errors, typed results, CSV/XLSX/ODS |
| `/editor` | sources, CodeMirror, diagnostics, format, validate, save, revert, source slice |
| `/import` | native adapters, upload/select, extract, edit, preview, diff, review, commit |
| `/options` | all applicable built-in Fava options and explicit excluded-option deviations |
| `/errors` | conditional navigation, diagnostics, source anchors and budget/FQL/import errors |
| `/help/<slug>` | standard help and shortcuts, localized for OrangeCount boundaries |
| Statements/documents | safe metadata statement links, downloads, uploads and containment errors |
| Global/modals | navigation, filters, keyboard shortcuts, Add Entry, Context, Export, notifications |

Third-party extension routes, plugin pages, multi-ledger hosting, public Fava
API compatibility, and Python importer execution remain excluded.

## Four-layer route gate

A route cannot switch from legacy to transplant until all four layers pass.

### Gate 1 — Contract and semantics

- Adapter requests, DTOs/HTML, status, errors, cancellation, ordering, URL
  parameters, and Fava validators agree.
- Exact OrangeCount/v3 values, natural-currency invariants, FQL/time parsing,
  option precedence, budget projection, exports, and source anchors pass Go
  tests.
- Missing conversions remain localized to converted summaries/charts and never
  erase natural holdings.

### Gate 2 — Behavior and safety

- Direct navigation, reload, history, filters, sorting, pagination, expansion,
  keyboard shortcuts, focus return, empty/loading/error/stale recovery, and
  responsive interactions pass browser flows.
- Query errors retain the prior successful result.
- Failed writes retain the previous snapshot and recoverable source state.
- Document and statement operations enforce containment and do not reveal local
  paths.

### Gate 3 — Visual and structural fidelity

- The route passes all four English Chromium baseline cells for required
  states with no unapproved difference.
- Simplified Chinese passes the structural/localization invariants.
- Visual work supplies a diff report; the user (product owner) approves any
  baseline or deviation change.
- The implementer cannot self-certify this gate.

### Gate 4 — Release quality

- Dense-fixture performance is no worse than the agreed Fava-relative budget;
  the initial target is at most 1.2× the same-environment Fava measurement,
  replaced by measured per-flow limits in Prerequisite Phase 0.
- Offline/CSP, accessibility, deterministic build, dependency size,
  license/provenance, Go race, and cross-browser checks pass.
- The route contains no mixed legacy/transplant DOM and no undocumented source
  derivative.

## Prerequisite Phase 0 and seven-wave implementation sequence

Phase 0 is a readiness prerequisite, not one of the seven implementation waves.
The seven waves are depth-first. Shared foundations may support later work, but
no wave may create broad placeholder pages and call that route progress.

### Prerequisite Phase 0 — Reference, fixture, decisions, and tooling

**Implementation:** create deterministic dense
fixture generator and hashes; build route/state manifest; close contract rows;
prepare deterministic export/build/license checks; define measured performance
flows.

**Visual work:** build the OCI Fava/Playwright reference environment; adopt
pinned fonts; capture and review the four English baseline cells; define
Chinese structural invariants and permitted mask rules.

**Coordinator:** reconcile source inventory, UX spec, provenance, reference
lock, contract map, and deviation registry; verify privacy and model ownership.

**Done when:** Fava and
OrangeCount can load the same synthetic ledger; reference tests are not
skipped; route/state manifest and baseline matrix are complete; all inventory
open questions are resolved or registered as approved exclusions.

### Wave 1 — Shell plus Income Statement golden vertical slice

**Code work:** implement full bootstrap, changed/mtime, errors, account detail
metadata, commodity names/precisions, applicable options, FQL/time foundations,
and Income Statement/tree/chart contracts over OrangeCount semantics.

**Visual work:** replace the prototype with selected Fava shell, router, stores,
sidebar, header, filters, theme, fonts, CSS, common components, Income
Statement, tree table, conversion/interval controls, and chart components.

**Done when:** `/`, the entire shell, and `/income_statement` display real dense
fixture data and pass all four route gates. No page-specific placeholder counts
as completion.

### Wave 2 — Core tree reports

Implement and accept `/balance_sheet` and `/trial_balance`, natural-currency
columns, account trees, conversion and interval behavior, fiscal periods,
Treemap/Sunburst/Icicle, legends, drill-down, accessible table fallbacks,
printing, and unavailable/error states.

**Done when:** all three core tree reports share stable upstream-derived
primitives and separately pass the four-layer gate with exact v3 values.

### Wave 3 — Journal and Account Detail

**Code work:** implement Fava-compatible Journal HTML templates with strict
escaping, complete directive/flag models, FQL/time execution, deterministic
pagination/sort, account running balances, account details/up-to-date status,
budget projection, source/context contracts, filtered Beancount export, and
safe statement resolution.

**Visual work:** adapt Fava Journal, filters, headers, interaction handlers,
account page, charts, context/export modals, badges, expansion, source links,
and dense/narrow behavior with minimal frontend change.

**Done when:** grouped transactions, postings, all directive kinds, budgets,
paging, keyboard behavior, account drill-down, running balance, statements,
exports, errors, and four English baseline cells pass.

### Wave 4 — Secondary read-only and document surfaces

Migrate Holdings variants, Commodities/Prices, Documents, Events, Statistics,
Errors, and Help. Implement cost/value/units, lots, price/unavailable states,
commodity metadata/precision, deterministic statistics, document preview and
source navigation, conditional Errors navigation, and localized help.

Document upload/attach/move/delete controls may be rendered here but cannot be
accepted until their Reviewed write workflow passes Wave 6 safety gates.

**Done when:** every read-only state, missing-price/conversion state, attachment
denial, empty/error state, and route-specific keyboard path passes four gates.

### Wave 5 — Query and exports

Adapt Fava Query and CodeMirror/BQL components to OrangeCount BeanQuery.
Implement saved state, typed sorting, table/string/error results, prior-result
retention, CSV/XLSX/ODS, and deterministic exact-value export.

**Done when:** query semantics remain OrangeCount/v3-authoritative while the
entire Fava Query workflow and export surface pass all four route gates.

### Wave 6 — Reviewed authoring

Migrate Editor, Source Slice, Add Entry, Import, Options, and mutating Document
operations. Adopt Fava CodeMirror and Tree-sitter assets. Implement all
applicable Fava options, native CSV/Beancount import adapters, Python-importer
migration diagnostics, preview/review/commit, concurrency hashes, atomic
replacement, backup, revalidation, rollback, and snapshot publication.

**Done when:** success, invalid input, concurrent change, write failure,
revalidation failure, rollback, and stale-snapshot browser flows all pass; no
implicit write exists; every visual state passes its matrix.

### Wave 7 — Cross-route hardening and final cutover

Run the complete English matrix, Chinese structural suite, WebKit/Firefox
flows, keyboard/accessibility checks, dense-fixture performance, offline/CSP,
private-ledger transient smoke, license/provenance audit, deterministic build,
Go race, and dead-asset scan. Resolve or obtain product-owner approval for all
remaining deviations.

**Done when:** every route has four gate records; the transplanted UI is the
only standard surface; legacy UI, prototype shell, feature flag, dead assets,
and legacy navigation are deleted.

## Per-route handoff packet

Every implementation task must provide:

- route/state manifest IDs and current wave;
- exact upstream Fava paths and target local paths;
- contract-map rows and semantic owner;
- fixture input/hash and required state setup;
- model role and exclusive file ownership;
- test commands and selectors/assertions;
- required English baseline cells and Chinese invariants;
- provenance/license requirements;
- Approved Fava deviations already applicable;
- required report format back to the coordinator: changed files, commands,
  screenshots/diffs, residual risks, and unapproved differences.

A task lacking this packet is not ready for delegation.

## Verification command families

The final exact commands are established in Prerequisite Phase 0. At minimum the release
gate includes:

```sh
make fmt
make vet
make test
make race
make license
make build
npm --prefix web ci
npm --prefix web test
npm --prefix web run build:check
npm --prefix web run visual:test
```

Additional commands must cover route-manifest completeness, deterministic
fixture/export/build hashes, provenance headers and manifests, accessibility,
WebKit/Firefox flows, visual diffs, dead legacy assets, offline requests, and
performance budgets.

## Principal risks and controls

| Risk | Control |
| --- | --- |
| Another clean-room approximation | Upstream-derived source boundary, provenance hashes, minimal patches, visual-work ownership |
| Baselines that compare OrangeCount to itself | OCI Fava reference generated from the same synthetic ledger |
| Small-fixture false confidence | Dense deterministic synthetic reference ledger plus compact state fixtures |
| Visual agent changes semantics | Contract-first handoff and OrangeCount/v3 authority |
| Implementer self-certifies visuals | Independent visual evidence and product-owner approval |
| Baseline laundering | Candidate-only updates, deviation registry, no arbitrary global threshold |
| Private data leakage | Synthetic committed evidence only; private smoke remains transient |
| Python/plugin creep | Native adapters and explicit migration diagnostics |
| Unsafe Fava write behavior | Reviewed write workflow and Approved Fava security deviations |
| Journal drift from a rewrite | Go-rendered Fava-compatible private HTML contract |
| Unsupported filter/time behavior | Complete Go-native FQL and time-filter contracts |
| License loss | Per-file provenance, NOTICE, dependency and asset license gates |
| Shared-worktree agent conflicts | Serialized writer ownership and coordinator-only integration |
| Performance regression | Same-environment Fava-relative dense-fixture budgets |
