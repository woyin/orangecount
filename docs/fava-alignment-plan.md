# Fava interface alignment plan

OrangeCount will use the locally running Fava instance as an interaction and visual acceptance reference. It will remain a Go-native, local-first application; reference inspection must not copy private ledger data into the repository, fixtures, screenshots, logs, or documentation.

## Acceptance method

For each Fava route, record a redacted structural baseline: page shell, navigation, controls, empty/loading/error states, keyboard interactions, table/tree behavior, and local workflow outcome. OrangeCount passes a route only when browser-driven checks can complete the corresponding workflow with materially equivalent visible behavior.

## Delivery slices

1. **Application shell** — Fava-compatible dark visual system, responsive top bar, persistent left navigation, global Time/Account/text filters, currency switches, and URL-backed view state.
2. **Reports** — Income Statement, Balance Sheet, Trial Balance, Holdings, Commodities, Documents, Events, Statistics, and account pages with graphs, tree tables, drill-down, period/valuation controls, printing/export behavior, and empty states.
3. **Journal and query** — directive/flag toggles, account/date/tag/payee/narration filters, expandable transaction rows, pagination/virtualization, source links, saved/run query workflow, and CSV download.
4. **Editor** — source browsing, editable file buffers, syntax highlighting, diagnostics, validation preview, safe atomic write/backup, and refresh behavior.
5. **Import and options** — local import workflow, review/commit path, operational options, help surfaces, and settings persistence.
6. **Parity QA** — route-by-route browser comparison against Fava; visual screenshots at desktop and narrow widths; keyboard/accessibility checks; Go tests, race tests, license checks, and redacted FinanceBook regression.

## Constraints

- Fava 1.30.12 frontend code, styles, and assets may be selectively adapted under its MIT license; retain copyright and license text per derived file and record each imported unit in the third-party notice inventory. Do not copy non-Fava third-party code or assets without a separate license review.
- Never execute Python plugins as part of OrangeCount's accounting engine.
- Editor writes must use explicit user action, atomic replacement, a recoverable backup, and revalidation before publishing the new snapshot.
- Importers operate locally and must present changes for review before committing them to a ledger file.
