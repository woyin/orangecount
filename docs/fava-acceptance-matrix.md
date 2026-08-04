# Historical Fava-alignment acceptance matrix

> **Superseded acceptance evidence.** This document records the 2026-08-02
> clean-room legacy-UI QA and explains why that approach produced false
> confidence. It does not satisfy the rendering-fidelity route gates and must
> not be updated as the current acceptance authority. Canonical coverage lives
> in `docs/fava-route-state-manifest.md`; current requirements live in
> `docs/fava-ux-spec.md` and `docs/fava-frontend-transplant-plan.md`.

At the time of this historical run, no Fava implementation code was copied and
no screenshots were retained. The accepted strategy now selectively adapts
Fava-derived frontend source and commits approved screenshots generated only
from the synthetic reference ledger.

## Shared shell

| Area | Reference behavior to verify | OrangeCount acceptance criteria |
| --- | --- | --- |
| Top bar | Dark blue application bar with page title, period/time controls, and global search/filter affordances | Persistent responsive top bar; current view and active period are visible; controls remain keyboard reachable |
| Navigation | Persistent left navigation on wide screens; collapses or becomes a menu on narrow screens | Routes are reachable from navigation and direct URLs; active route has an accessible state; layout works at desktop and narrow widths |
| Global filters | Time/period, account, and text filters affect the current report or journal | Filter values are URL-backed, survive reload/navigation, and visibly change results; reset restores the unfiltered state |
| Currency/valuation | Report-specific currency and valuation controls where supported | Controls are explicit, deterministic, and do not mutate ledger semantics |
| States | Loading, empty, invalid/error, and stale-snapshot states | Each state has localized, actionable copy; failed reloads do not erase the last valid snapshot |

## Routes and workflows

The route names below follow the Fava naming convention. Exact labels may be
localized, but the URL state must remain stable and bookmarkable.

| Route | Primary controls | Acceptance workflow |
| --- | --- | --- |
| `/` | Period summary, account/text filters | Open overview, change a global filter, refresh, and retain the selected state |
| `/journal` | Date from/to, account, text, tags/links, payee/narration, flags, pagination or virtual scrolling | Filter to one day, combine filters, clear them, expand a transaction, and follow its source location |
| `/balance_sheet` | Period, currency/valuation, account tree | Expand/collapse account tree, drill into an account, and export/print the visible report |
| `/income_statement` | Period, currency/valuation, account tree | Change period and verify deterministic totals and drill-down behavior |
| `/trial_balance` | Period, currency, account tree | Show debit/credit or balance columns, sort headers, and open account details |
| `/holdings` | As-of date, currency/valuation, account filters | View lots/cost basis, sort columns, and handle an empty holdings state |
| `/commodities` | Commodity/currency filter, date range | View prices/commodity metadata and show a localized empty state |
| `/documents` | Account/date/tag filters | List document links and enforce configured-root containment |
| `/events` | Date range, type/text filters | Filter events and open the related source location |
| `/statistics` | Period and account filters | Show deterministic counts/totals and a responsive chart/table fallback |
| `/query` | Query editor, run/save controls, result format/export | Run a query, sort result headers, preserve exact CSV/query values, and show parse errors without losing the last result |
| `/editor` | File tree, editable buffer, diagnostics, validate/save/revert | Edit a file, preview diagnostics, explicitly save with atomic backup, revalidate, and retain the previous snapshot on failure |
| `/import` | Local source selection, importer/options, review/commit controls | Preview proposed postings, require explicit review/commit, write only to the selected ledger, and revalidate |
| `/options` | Display/runtime options and persistence controls | Show supported options, validate changes, and keep unsupported/plugin options clearly separated |
| `/help` | Searchable help and keyboard shortcuts | Open help from the shell, search it, and return to the prior route without losing URL state |
| Account detail | Account/date/valuation controls, journal and graph/tree views | Open from a report or direct URL, retain global filters, and provide source-linked drill-down |

## Cross-route interaction checks

- Every rendered table has keyboard-focusable sortable headers with
  `aria-sort`, typed date/decimal/text comparison, and stable tie ordering.
- Every chart has an accessible tabular fallback and does not require a remote
  asset or runtime network request.
- Every write operation is explicit, atomic, recoverable, and followed by
  parse/evaluate validation before a new snapshot is published.
- English (`en`) and Simplified Chinese (`zh-CN`) cover shell controls,
  report labels, diagnostics, empty states, editor/import actions, and help.
- Browser/API tests use sanitized fixtures only. Private reference validation
  emits aggregate counts/error classes and never stores source data or
  screenshots.

## Historical evidence policy

This run recorded only structural outcomes. The current policy additionally
requires approved Fava synthetic-ledger screenshots in the strict English
matrix. Private screenshots, ledger values, source paths, narrations, account
names, and raw private browser dumps remain forbidden.

## Sanitized parity status (2026-08-02)

The following checks were run with `agent-browser` against the local Fava
reference and a temporary OrangeCount server built from `core.bean`. Only
structural observations are recorded here; no reference ledger text, values,
paths, screenshots, or browser dumps were retained.

| Route/workflow | Desktop | Narrow (520px) | Verified behavior | Deliberate non-parity or boundary |
| --- | --- | --- | --- | --- |
| Shared shell | pass | pass | 52px blue top bar, 160px sidebar, global time/account/text controls, currency and locale switches, active navigation, responsive menu | OrangeCount uses a Go-native shell and does not copy Fava implementation code |
| `/` | pass | pass | Overview status cards, filter state retained in URL | No Fava plugin panels |
| `/journal` | pass | pass | Inclusive date range, account/text/tag/link/payee/narration/flag filters, sortable headers, expandable tags/links, source/file links, CSV export, functional page controls for large result sets | Virtualized rendering is not needed for the sanitized corpus; page controls remain the deterministic fallback |
| `/balance_sheet`, `/income_statement`, `/trial_balance` | pass | pass | Period/valuation controls, typed sort, account tree indentation, account drill-down links, CSV export, chart/table fallback, keyboard chart selection | Valuation uses exact local price maps; unsupported currencies remain explicitly unvalued |
| `/holdings` | pass | pass | As-of date, cost/market valuation, presentation currency, exact value columns, explicit unavailable-price/unavailable-currency statuses, empty state and report controls | Historical sold-lot reconstruction is bounded by the immutable evaluator position snapshot; no external market provider is contacted |
| `/commodities`, `/events`, `/documents` | pass | pass | Local price-map/events/document tables, safe document roots, source links, empty states | No remote asset, external market provider, or plugin-backed commodity views; local price directives are the supported manual path |
| `/statistics` | pass | pass | Deterministic directive counts, keyboard-selectable inline SVG chart with table fallback, controls | Chart is a compact native SVG with no remote dependency |
| `/query` | pass | pass | Query editor, run/save, typed sorting, exact CSV link, parse errors preserve prior result | BeanQuery surface is intentionally Go-native |
| `/editor` | pass | pass | Include-graph file tree, editable buffer with line gutter/token coloring, Ctrl+S/Ctrl+Enter, validate, atomic save/backup/revalidate, source-linked diagnostics | Tokenization is intentionally local and dependency-free |
| `/import` | pass | pass | Local file selection, Beancount or generic CSV adapter, mapping controls, parser diagnostics, diff summary, review rows, explicit commit/revalidate to selected graph file | No Python plugin execution; adapter extension remains local Go code |
| `/options` | pass | pass | Locale/currency/time validation and local persistence | Unsupported runtime/plugin options are rejected |
| `/help` | pass | pass | Searchable local help and keyboard guidance | Content is OrangeCount-native |

Go/API coverage includes sanitized route, filter, editor rollback, import
commit, options, help, and accessibility-marker tests. The release gate also
requires `make fmt vet test race license build` and a redacted private-ledger
differential summary; neither emits source paths or ledger values.

## Chart semantics

Report JSON keeps `columns` and `rows` stable and adds an optional `chart`
object for account, balance-sheet, and income-statement views. Chart series
are report-defined (balance-sheet assets/liabilities/equity/net worth;
income/expenses/net profit; selected-account running balance), carry an
explicit currency, valuation, and month/quarter/year interval, and preserve
exact decimal values alongside display values. `period=all` uses monthly
intervals for the chart while leaving table filtering unchanged. Statistics
continues to use a directive-count composition chart. No chart reads an
arbitrary numeric table column or contacts an external price provider.

Final QA run: all 15 routes above loaded at 1280px and 520px with zero
rendered error panels; narrow routes had no horizontal overflow and the menu
button opened the hidden sidebar. Sort, filter, source-link, chart-keyboard,
editor-shortcut, and CSV-adapter preview interactions were exercised in the
same sanitized session.
