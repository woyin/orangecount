# Approved Fava deviations

This registry is the only place where OrangeCount may accept an observable difference from the pinned Fava 1.30.12 visual baseline or behavior. An implementing agent may propose a deviation and produce evidence, but only the user, acting as product owner, may approve it.

A deviation is valid when required by OrangeCount's accounting semantic authority, security, data integrity, privacy, or accessibility obligations, or when approved as a switchable ledger-owner presentation preference (see CONTEXT.md, "Approved Fava deviation"). Silent drift, unreviewed redesign, and presentation changes that cannot be switched back to parity are not valid.

## Status vocabulary

- `proposed` — evidence exists, but the difference still fails its route gate.
- `approved` — the user (product owner) accepted the bounded difference.
- `expired` — the approval no longer applies and the route fails until reviewed.
- `removed` — OrangeCount once differed but now matches Fava.

## Registry

### FD-0001 — Import file list and per-entry extract/review rely on Python importers

| Field | Value |
| --- | --- |
| Status | approved |
| Route and state | R-IMPORT (`import`) |
| Fava baseline | Import page "Importable Files" list (`FileList.svelte`) + per-entry "Extract" modal (`Extract.svelte`) driven by Python importers |
| OrangeCount behavior | Import is a single-file local-buffer workflow (upload/paste one file, pick Source path/Adapter/Target, Preview, Commit); no server-side import directory, no file list, no per-entry extract/review modal |
| Category | semantics |
| Reason | OrangeCount is a Go-native EagerLedger that approved excluding the Python importer ecosystem (decision D3). Upstream file attribution (identify/file_account/file_date) and per-entry extraction (extract into structured directives) live in user-supplied Python importer classes (beangulp, fava/core/ingest.py); the Go runtime cannot equivalently reproduce importer file recognition, entry extraction, or account inference, and commit writes appended text rather than per-entry add_entries. Reproducing importer intelligence in Go has no bounded recognition correctness. |
| Scope | import route: file list, per-entry extract/review modal, importer-driven auto-attribution; the local upload + generic beancount/csv adapter preview→commit flow is retained |
| Tests | M2-import-upload smoke (DataTransfer upload → local buffer load, path/adapter backfill, Preview diagnostics, Commit stays disabled until valid) |
| Owner | implementing agent (Francis Chen / OrangeCount maintainer) |
| Approver | user (product owner) only |
| Approved evidence | review reference and date: approved by product owner 2026-08-08 |
| Expiry condition | a Fava upgrade that changes the Import contract, or an OrangeCount decision to embed a supported importer runtime |
| Baseline impact | alternate expectation for the Import page |

### FD-0002 — Failed-balance diff_amount is not rendered

| Field | Value |
| --- | --- |
| Status | approved |
| Route and state | R-JOURNAL (`journal`) |
| Fava baseline | balance rows where the assertion fails render the expected amount with a `pending` class and an extra change column showing `diff_amount` (beancount's actual-minus-expected difference) |
| OrangeCount behavior | balance rows always render the amount and no difference column; a failed assertion never reaches the runtime |
| Category | semantics |
| Reason | OrangeCount's accounting semantic authority serves only valid ledgers: `serve` (cmd/orangecount/main.go:181) and the editor/commit write paths reject any snapshot carrying error diagnostics. `diff_amount` exists only on a failed balance assertion, so it can never appear in a served ledger; implementing its rendering would be an unreachable code path. |
| Scope | journal balance rows; the adapter projects no diff field and the journal renders no pending/difference column |
| Tests | the evaluator emits E-EVAL-BALANCE on a failed assertion (evaluator_test coverage); the runtime never serves such a ledger |
| Owner | implementing agent (Francis Chen / OrangeCount maintainer) |
| Approver | user (product owner) only |
| Approved evidence | review reference and date: approved by product owner 2026-08-08 |
| Expiry condition | if OrangeCount ever serves invalid ledgers (e.g. a diagnostics mode), diff_amount rendering becomes reachable and should be re-reviewed |
| Baseline impact | alternate expectation for the journal balance row |

### FD-0003 — Journal budget (B) chip

| Field | Value |
| --- | --- |
| Status | approved |
| Route and state | R-JOURNAL (`journal`) |
| Fava baseline | the "B" journal filter chip shows/hides `custom "budget"` entries (beancount budget directives) |
| OrangeCount behavior | the B chip is absent from the journal filter set |
| Category | semantics |
| Reason | Beancount parser/interpolation of `custom "budget"` directives plus a budget model underlies the chip; ADR-0017 defers budgeting until it has an explicit model, deliberately not introducing a private budget syntax in the first release. |
| Scope | journal filters; the budget module and account-page budget charts also stay deferred |
| Tests | none (chip absent) |
| Owner | implementing agent (Francis Chen / OrangeCount maintainer) |
| Approver | user (product owner) only |
| Approved evidence | review reference and date: approved by product owner 2026-08-08 |
| Expiry condition | a future budget model decision (ADR-0017 revisit) |
| Baseline impact | alternate expectation for the journal filter set |

### FD-0004 — serve refuses ledgers with error diagnostics

| Field | Value |
| --- | --- |
| Status | approved |
| Route and state | G-SHELL / R-ERRORS (`errors`) |
| Fava baseline | Fava serves a ledger even with error diagnostics and shows the full set on /errors |
| OrangeCount behavior | `serve` (cmd/orangecount/main.go:181) and the editor/commit write paths reject a snapshot carrying error diagnostics; /errors can only show warnings |
| Category | semantics |
| Reason | OrangeCount's accounting semantic authority serves only valid ledgers so reports are computed over a consistent, balanced state; a ledger with error diagnostics is not served. This is the same valid-only constraint that makes FD-0002's diff_amount unreachable. |
| Scope | serve startup, editor/commit write paths, /errors page |
| Tests | serve exits on error diagnostics; /errors renders warnings only |
| Owner | implementing agent (Francis Chen / OrangeCount maintainer) |
| Approver | user (product owner) only |
| Approved evidence | review reference and date: approved by product owner 2026-08-08 |
| Expiry condition | if OrangeCount ever introduces a diagnostics/error-serving mode, re-review |
| Baseline impact | alternate expectation for the /errors page |

### FD-0005 — File-change reload keeps a warning toast

| Field | Value |
| --- | --- |
| Status | approved |
| Route and state | G-SHELL (notifications) |
| Fava baseline | auto-reload defaults to silently reloading without a toast |
| OrangeCount behavior | a file change reloads and also shows a warning toast (click to reload again, auto-dismiss after 5s) so the change is perceptible |
| Category | semantics |
| Reason | A perceptible reload notice guards against surprise data changes; the toast is the user-visible signal that the served snapshot changed. |
| Scope | file-change notification on any served route |
| Tests | H7 smoke (warning toast text + class, click-to-reload, auto-dismiss) |
| Owner | implementing agent (Francis Chen / OrangeCount maintainer) |
| Approver | user (product owner) only |
| Approved evidence | review reference and date: approved by product owner 2026-08-08 |
| Expiry condition | if upstream auto-reload behavior changes or a silent-reload preference is requested |
| Baseline impact | alternate expectation for the reload notification |

### FD-0006 — Modern chart layer (switchable owner presentation preference) [removed]

| Field | Value |
| --- | --- |
| Status | approved |
| Removal | Removed 2026-08-09: ADR-0041 promoted the modern time-series presentation to the standard-route default and removed the parity fallback, so it is no longer a deviation. Hierarchy charts still render through the parity ReportChart (out of scope for the modern layer). |
| Route and state | R-IS / R-BS / R-ACCOUNT (chart regions of income_statement, balance_sheet, account) |
| Fava baseline | `ReportChart.svelte` hand-written SVG bar/line time-series charts, the parity default |
| OrangeCount behavior | an alternative chart presentation enabled by `?chart_layer=modern` (default off, parity remains the standard-route default). It renders the same income-statement, balance-sheet, and account time-series with d3-backed scales/axes/stack-offsets/tick-formatting, per-pixel responsive redrawing, a crosshair with linked series highlighting, and an HCL ordinal palette so >4 currencies do not collide. |
| Category | presentation preference |
| Reason | ledger-owner readability preference; the modern layer must be switchable back to parity and is never the default for a standard route |
| Scope | IS bar chart + BS/account line chart only; hierarchy (treemap/sunburst/icicle), Commodities line, Events scatter, and Statistics activity charts are out of scope (first phase) |
| Semantic boundaries | the modern layer consumes the same adapter data contract (PresentedChartSpec); it performs no secondary aggregation, no re-valuation, no derived accounting; diverging stack and currency toggling are visual placement only; tooltip values are the DisplayAmount; no new accounting concept is introduced |
| Tests | `?chart_layer=modern` smoke on multi-currency real ledger (16 currencies): IS bar chart renders stacked (136 rects, shared period band) ↔ single (side-by-side, half bandwidth); BS/account line chart renders line ↔ area (5 area-fills); responsive per-pixel redraw 1063↔300 px with x-axis tick density adapting (13↔3); crosshair + hover-card + linked series dimming; HCL palette yields 5 distinct series colors (no 4-color recycle); parity restored on toggle-back; `?chart_layer=` empty = parity default; full chain npm test 32/32 / build:embedded / make check / go test ./internal/... (10 pkgs) / make build all green |
| Owner | implementing agent (Francis Chen / OrangeCount maintainer) |
| Approver | user (product owner) only |
| Approved evidence | approved by product owner 2026-08-09 after grilling consensus (D1–D7) and implementation with the browser smoke above |
| Expiry condition | if the modern layer becomes the standard-route default, loses its switch-back, or changes ledger semantics; or on a Fava upgrade that redefines the chart baseline |
| Baseline impact | alternate presentation under `?chart_layer=modern` only; default route unchanged |

No other deviations are currently approved. Existing differences in the prototype and legacy UI are migration gaps, not approved deviations.

## Entry template

```markdown
### FD-0001 — Short name

| Field | Value |
| --- | --- |
| Status | proposed / approved / expired / removed |
| Route and state | route-manifest identifier |
| Fava baseline | baseline path and bounded region |
| OrangeCount behavior | precise observable difference |
| Category | semantics / security / data integrity / privacy / accessibility / presentation preference |
| Reason | why matching Fava would violate the named obligation |
| Scope | exact themes, viewports, locales, controls, and states covered |
| Tests | contract, browser, visual, and safety evidence |
| Owner | implementing agent or maintainer |
| Approver | user (product owner) only |
| Approved evidence | review reference and date |
| Expiry condition | event that requires re-review, including a Fava upgrade |
| Baseline impact | masks, alternate expectation, or none |
```

## Rules

1. Baseline candidates are never promoted merely because OrangeCount changed.
2. Masks must be limited to the entry's exact nondeterministic region; a deviation is not permission for a global visual threshold.
3. A semantic deviation must preserve visible access to exact OrangeCount/v3 results rather than hiding the conflict.
4. A security, integrity, or privacy deviation must keep the Fava information hierarchy wherever the protected behavior allows.
5. Accessibility changes should preserve ordinary visual composition and document keyboard/focus differences.
6. Upgrading the pinned Fava version expires every approval unless the entry explicitly proves that its reason and scope remain unchanged.
