# Fava 1.30.12 frontend transplant plan

## Decision summary

OrangeCount will deliver the complete built-in local Fava 1.30.12 experience
through a transplanted, selectively adapted Fava frontend and a private Go
adapter. This is an implementation plan, not a requirement to run Fava or
Python in the released product.

- Fava is the UX authority; Beancount v3 remains the accounting and BeanQuery
  semantic authority.
- Fava's MIT license permits selected code, style, and asset reuse only when
  copyright and license notices are retained and recorded.
- The released binary remains Go-served, embedded, offline, and loopback-only.
- The target excludes Fava's public HTTP API, Python/Beancount runtime,
  third-party plugin pages, user extensions, and Python importers.
- English is the visual reference. `zh-CN` keeps the same IA and interaction
  model, with legitimate translated-copy and CJK-font layout differences.
- Private-ledger observations are transient. All committed visual fixtures and
  evidence use a deterministic, sanitized ledger.

The authoritative decisions are ADR-0022 and ADR-0024 through ADR-0036.
ADR-0023 is superseded by ADR-0030.

## Definition of complete

The migration completes only when all the following are true.

1. Every Fava 1.30.12 built-in route, menu, modal, keyboard path, loading,
   empty, and error state is classified in the source inventory as adopted,
   replaced, or explicitly excluded by the decisions above.
2. Every adopted page is rendered by the transplanted frontend and receives
   data only through the Go adapter and immutable OrangeCount snapshot.
3. The English UI passes behavior, accessibility/responsiveness, and visual
   regression gates at desktop and narrow widths. `zh-CN` passes the first two
   gates and locale-aware visual review.
4. Reports and queries retain OrangeCount/Beancount-v3 results even when that
   differs from Fava's Python results.
5. Editor and import writes remain explicit, atomic, backed up, validated,
   and published through the existing Go snapshot mechanism.
6. `make fmt vet test race license build` passes, no network is needed at
   runtime, and `NOTICE` plus the provenance inventory pass review.
7. The feature flag is removed, the Fava UI becomes default, and no legacy UI
   code remains for an in-scope route.

## Target architecture

```text
Fava 1.30.12 reference mirror (external, read-only, pinned commit)
  ├─ complete source inventory + MIT/dependency review
  ├─ visual observer creates sanitized behavior/spec baselines
  └─ selected frontend source → attributed OrangeCount frontend workspace

OrangeCount binary
  transplanted Fava frontend
       │ private loopback requests only
       ▼
  internal/web Fava-shaped adapter
       │ maps contracts; no public compatibility promise
       ▼
  report / query / source / snapshot / ledger (Go, v3 semantic authority)
       │
       └─ editor and import use the existing atomic write + revalidation path
```

### Frontend boundary

`web/` becomes the maintained source workspace for the transplanted frontend.
Its build emits only static, local assets, which are copied into
`internal/web/assets/` and embedded by Go exactly as today. Node/package
manager tooling is development/build-only; it is neither needed nor shipped
at runtime. The current dependency-free `app.js` is retained only behind the
temporary UI flag until each route cutover is accepted.

No page may combine legacy OrangeCount DOM/components with transplanted Fava
components. A route is wholly legacy or wholly Fava-transplant while the flag
exists; shared Go write, snapshot, locale, and security services remain common.

### Go adapter boundary

Create a narrowly owned adapter layer under `internal/web/` (the exact package
layout is chosen after the inventory). It owns:

- the boot/session payload expected by the frontend;
- route-specific report, journal, account, commodity, document, event,
  statistics, query, editor, import, options, and error contract adapters;
- Fava-shaped status, validation, pagination, sort, and error payloads;
- URL/query decoding and conversion to typed Go report/query inputs;
- contract fixtures proving that a frontend request maps to the intended Go
  domain result.

It does **not** own accounting calculation, mutable ledger state, source-file
authorization, import parsing, or snapshot publication. Those remain in their
existing packages. The adapter is internal, loopback-only, versioned only for
this embedded client, and never documented as a public Fava API.

### Contract-map format

For every frontend request, the inventory creates a row with these fields:

| Field | Required content |
| --- | --- |
| Fava source | tag, path, symbol, and frontend call site |
| Route/state | URL, relevant query state, and UI state entered |
| Request shape | method, parameters, body, and cancellation/loading behavior |
| Go owner | snapshot/report/query/source/editor/import function used |
| Semantic rule | v3 behavior, valuation and unavailable-data policy |
| Response adaptation | exact frontend-required fields, types, ordering, errors |
| Tests | contract fixture, browser flow, visual state, accessibility checks |
| Provenance | copied/adapted/rewritten decision and license notice location |

## Reference, license, and privacy workflow

### Reference lock

Work package P0 resolves `v1.30.12` to its immutable upstream commit and
creates a committed lock record containing tag, commit, retrieval date,
upstream URL, Fava license hash, and frontend dependency lock hashes. The full
checkout lives outside this repository in an ignored read-only location.

### Provenance

Create a committed provenance inventory. For each Fava-derived file or asset,
record upstream path and revision, OrangeCount path, whether it was copied or
adapted, local modifications, copyright holder, MIT notice placement, and
dependency-license result. Update `NOTICE` and the license check so a derived
file cannot be added without inventory evidence.

### Visual evidence

Only a visual-capable operator may perform Fava observation and approve visual
output. That operator uses the local Fava instance transiently, then writes a
sanitized specification. Private screenshots, DOM dumps, API responses, paths,
account names, values, and raw interaction recordings are never written to the
repository or passed to implementation agents.

The committed regression corpus contains a synthetic multi-currency ledger
with: nested accounts, commodities, missing conversion paths, directives,
documents under a sandbox root, editor errors, import candidates, saved query
states, and enough transactions for pagination. It must contain no private
ledger material.

## Page-family migration order

The ordering eliminates shared risks first and addresses the currently observed
high-impact gaps before secondary surfaces.

| Wave | Page family | Why this order | Principal acceptance |
| --- | --- | --- | --- |
| 0 | Reference, inventory, licenses, fixture, visual harness | Required inputs for all later work | P0 and P1 evidence accepted |
| 1 | Build, shell, navigation, URL/state, locale, theme | Shared visual and routing foundation | Fava standard navigation/default page; desktop+narrow shell |
| 2 | Journal and account detail | Highest-frequency workflow; transaction grouping | Transaction header/detail expansion, directive badges, running balance |
| 3 | Balance sheet, income statement, trial balance, accounts | Solves multi-currency and hierarchy root cause | Account-by-row / currency-by-column, drill-down, non-disappearing charts |
| 4 | Holdings, commodities/prices, documents, events, statistics | Reuses reports, filters, tables, chart controls | Empty/unavailable states and source/document safeguards |
| 5 | Query and saved queries | Contract-heavy but isolated semantic owner | Run/save/export/error preservation and typed sort |
| 6 | Editor, import, options, help, errors | Write safety and configuration boundaries | Atomic save/rollback, reviewed import, Fava-shaped options/help |
| 7 | Cross-route hardening and cutover | Eliminates fallback and protects release | Full matrix, visual/a11y/perf/license release gate |

## Work packages

Each package has one owner. Agents without visual capability must consume the
published sanitized spec and must not self-certify visual acceptance.

### P0 — Fava reference lock and full source inventory

**Owner:** research/architecture agent. **Depends on:** none. **May not edit:**
OrangeCount production UI or Go behavior.

1. Obtain an external read-only checkout at the locked `v1.30.12` commit.
2. Read all frontend, backend route, state, build, test, style, and dependency
   areas; record the inventory rather than copying the checkout.
3. Produce `docs/fava-source-inventory.md`, route and module tables, a list of
   all Fava frontend requests, and an adoption/rewrite/exclusion decision for
   every file.
4. Produce the initial contract map and a dependency license inventory.
5. Add the reference lock and provenance-template documents.

**Done when:** an independent reviewer can trace every in-scope page from a
Fava source module through its data contract to a planned OrangeCount owner.

### P1 — Sanitized fixture and visual-spec baseline

**Owner:** visual-capable agent/operator only. **Depends on:** P0 route list.
**May not do:** persist private ledger evidence.

1. Build the sanitized multi-currency fixture and document roots.
2. Exercise every Fava route/state in English at agreed desktop and narrow
   viewports; record abstract control/state behavior and keyboard paths.
3. Generate committed sanitized screenshots and visual-regression masks only
   where browser rendering is intentionally variable.
4. Write `docs/fava-ux-spec.md` as page/state tables, not prose impressions.
5. Establish browser tests that can run against Fava and OrangeCount using the
   same fixture without storing Fava private data.

**Done when:** a non-visual agent can implement a route from the spec and a
visual operator can reproduce an objective pass/fail comparison.

### P2 — Build pipeline, attribution guard, and shell transplant

**Owner:** frontend/platform agent. **Depends on:** P0, P1 shell spec.
**Owns:** `web/`, generated-asset integration, `NOTICE`, license tooling, shell
routes; coordinate with existing dirty changes rather than reverting them.

1. Import only P0-approved Fava frontend build/config/source units with MIT
   notices and provenance entries.
2. Add reproducible build target that emits static assets into the existing Go
   embed directory; update `Makefile` so normal Go release builds consume
   checked-in assets and need no runtime Node/CDN.
3. Implement the temporary UI flag and route isolation.
4. Transplant app shell, sidebar, top controls, standard navigation, default
   page, global URL state, locale/theme, responsive menu, focus handling, and
   common loading/error components.
5. Add artifact, source-header, dependency, and license checks.

**Done when:** Fava UI can boot behind the flag with no page-specific legacy
DOM, and shell visual/a11y browser tests pass at both viewports.

### P3 — Adapter foundation and bootstrap contract

**Owner:** Go web agent. **Depends on:** P0, P2 request map. **Owns:**
`internal/web/` adapter only; do not alter ledger semantics.

1. Define typed adapter DTOs and request/response fixture helpers.
2. Implement session/bootstrap, locale/options, route-state, diagnostics,
   source-link, snapshot-status, and standardized error contracts.
3. Map existing `/api/v1/*` data to private Fava-shaped routes without exposing
   a public compatibility commitment.
4. Preserve loopback-only routing, content security, attachment containment,
   redaction, and stale-snapshot behavior.

**Done when:** shell and one read-only fixture page run entirely through
contract-tested adapter endpoints.

### P4 — Journal and account-detail vertical slice

**Owner:** one frontend agent plus one Go report agent, with non-overlapping
ownership. **Depends on:** P1–P3.

- **Go report owner:** expose transaction identity, directive kind, postings,
  source anchors, filtering, and account running-balance data; add tests for
  multi-posting grouping and deterministic ordering.
- **Frontend owner:** transplant Journal, directive/flag badges, transaction
  expansion, account page, account graph controls, pagination, and URL state.
- **Visual operator:** accepts only against P1's journal/account states.

**Done when:** transaction grouping replaces posting-row flattening; account
links open a dedicated account page with graph and running balance; keyboard,
source navigation, narrow layout, and CSV behavior pass.

### P5 — Core report and currency-presentation vertical slice

**Owner:** report-model agent and frontend report agent. **Depends on:** P3.

1. Introduce a presentation projection that groups rows by account and pivots
   natural holdings into dynamic currency columns. It must not alter evaluator
   inventory or exact decimals.
2. Redefine any global currency value as a default valuation/chart preference,
   never as a filter that hides natural-currency table values.
3. Transplant balance sheet, income statement, trial balance, account tree,
   hierarchy charts, currency legends, drill-down, export/print and empty
   states.
4. Default a chart currency from operating currency; render a localized
   unavailable-data card rather than silently dropping a chart.

**Done when:** the synthetic multi-currency fixture demonstrates one account
row with multiple natural currency columns, usable hierarchy charts, and
stable v3 exact values.

### P6 — Secondary read-only pages

**Owner:** page-family agents, one family per agent. **Depends on:** P3 and
shared table/chart primitives from P2/P5.

| Agent ownership | Pages and required work |
| --- | --- |
| Holdings | lots, cost/market valuation, prices, unavailable pricing, sort/export |
| Commodities | commodity overview and the Fava-equivalent prices view/tab, metadata and filters |
| Documents/events | list/filter/source link, document-root enforcement, event navigation |
| Statistics | deterministic metrics, chart controls, accessible table fallback |
| Errors/help | Fava-shaped conditional errors surface, searchable help and shortcuts |

**Done when:** each family passes its P1 state table, contract fixtures, narrow
layout, keyboard checks, and its existing Go security/semantic tests.

### P7 — Query, editor, import, and options

**Owner:** separate agents by write boundary. **Depends on:** P2/P3 and the
existing Go write services.

- **Query:** transplant query editor/results/saved-query UX; adapt to Go
  BeanQuery; retain exact CSV values and preserve prior result on error.
- **Editor:** transplant file tree, buffer, syntax/editor commands and
  diagnostics UX; call existing Go validate/save/revert paths only; require
  atomic replacement, backup, revalidation, and previous-snapshot retention.
- **Import:** transplant Fava-shaped selection/preview/review/commit UX;
  connect native CSV/Beancount adapters; never execute Python Fava importer
  configuration; surface migration guidance when such config is encountered.
- **Options:** transplant standard layout/theme/system controls and read-only
  ledger option display; distinguish supported local options from excluded
  plugin options.

**Done when:** browser tests prove failed editor/import writes do not publish a
new snapshot and successful writes are explicit, recoverable, and localized.

### P8 — Final integration, removal, and release gate

**Owner:** integration/visual operator. **Depends on:** P0–P7.

1. Run every documented Fava standard route and state against the sanitized
   fixture in English and `zh-CN`.
2. Run visual diffs only for approved deterministic English states; triage
   structural regressions, not font rasterization noise.
3. Run private-ledger smoke validation locally without retaining evidence.
4. Remove legacy route implementations and feature flag, update navigation and
   help, and verify no dead legacy asset is embedded.
5. Run release commands and provenance/license audits.

## Required test layers

| Layer | Owner | Gate |
| --- | --- | --- |
| Go unit/differential | ledger/report/query owners | v3 semantics, exact values, projection invariants |
| Adapter contract | Go web owner | Fava frontend shape and error/status behavior |
| Frontend unit | frontend owners | reducers/state, formatting, routes, keyboard semantics |
| Browser flow | page owner | task completion, URL persistence, responsive layout, write safety |
| Visual regression | visual-capable operator | English structural hierarchy and visual density |
| Accessibility | page owner + visual operator | focus order, labels, keyboard, table/chart fallback |
| License/provenance | platform owner | MIT notices, inventory completeness, dependency review |

## Handoff rules for non-visual agents

Every task issued to a non-visual agent must include: the route/state identifier
from the UX spec; requested source and target files; adapter contract rows;
fixture/test command; acceptance selectors/assertions; and provenance rule. It
must not include private screenshots, DOM dumps, ledger data, or a request to
make an unverified “visual match.” The visual operator alone creates baselines,
reviews diffs, and records the result.

## Risk controls

| Risk | Control |
| --- | --- |
| Upstream drift | freeze Fava 1.30.12 and commit lock; upgrades are separate projects |
| Accidental whole-app port | P0 adoption decisions, private adapter boundary, no Python runtime dependency |
| MIT attribution loss | per-file provenance inventory, NOTICE update, CI license guard |
| Private-ledger leakage | transient-only observation and sanitized committed fixture |
| Semantic regression while matching UI | v3 authority, exact-value tests, adapter-only projection |
| Visual work delegated to blind agents | P1/P8 visual-capable ownership and objective spec artifacts |
| Broken writes during parallel UI migration | shared Go atomic write/snapshot path and browser rollback tests |
| Existing dirty worktree conflicts | task owners inspect status first, never revert unrelated changes, and coordinate file ownership |

## First delegation sequence

Issue only these tasks initially, in order:

1. **P0 source inventory** — no production edits.
2. **P1 visual baseline** — sanitized artifacts only; run in parallel after P0
   supplies the route list.
3. **P2 frontend/platform** and **P3 Go adapter** — begin after P0/P1 approve
   the shell and bootstrap contracts.
4. **P4 Journal/account** and **P5 core reports** — parallel after P2/P3.

Do not delegate secondary pages, editor/import, or legacy removal until P4/P5
have established shared routing, table, chart, request, and provenance patterns.
