# Fava standard-surface route and state manifest

This is the canonical coverage registry for the pinned Fava 1.30.12 standard surface. The source inventory explains upstream implementation units; the contract map owns wire contracts; this manifest decides which user-visible routes and states must pass the four-layer route gate.

No row is accepted merely because a route or placeholder exists. At the time of writing, no transplanted route has an approved Fava visual baseline.

## Controlled dimensions

Every adopted route's required English loaded state is exercised in all four Chromium cells:

- `desktop-light`
- `desktop-dark`
- `narrow-light`
- `narrow-dark`

Each route also covers its applicable `empty`, `loading`, `unavailable`, `error`, and `stale` states. Shared state rendering may use representative visual baselines, but route-specific states require their own evidence. Simplified Chinese repeats the structural/behavior manifest; WebKit and Firefox repeat supported behavior and serious-layout checks.

## Status vocabulary

- `legacy` — available only in the old OrangeCount UI; not parity evidence.
- `prototype` — scaffold or placeholder; not parity evidence.
- `planned` — inventory and accepted scope exist, implementation not accepted.
- `gated` — transplant implementation exists but one or more route gates fail.
- `accepted` — all four route gates passed and the user approved any baseline/deviation change.
- `excluded` — outside the Fava standard surface with an explicit governing decision.

## Route registry

| ID | Route/surface | Required loaded composition and outcomes | Route-specific states | Wave | Current status |
| --- | --- | --- | --- | --- | --- |
| R-ROOT | `/` | Ledger bootstrap, title, applicable options and Fava-compatible default-page resolution | invalid default, bootstrap error, stale snapshot | 1 | legacy/prototype |
| R-IS | `/income_statement` | Shell, filters, conversion/interval, income and expense trees, natural currencies, totals, chart/table fallback, print | no activity, missing conversion, chart unavailable, refresh error | 1 | legacy/prototype |
| R-BS | `/balance_sheet` | Account tree, natural currencies, totals, conversion/interval, hierarchy charts, drill-down, print | no balances, missing/disconnected prices, refresh error | 2 | legacy |
| R-TB | `/trial_balance` | Trial-balance tree/table, currency legend, Treemap/Sunburst/Icicle, accessible fallback | no rows, zero series, unavailable valuation, refresh error | 2 | legacy |
| R-JOURNAL | `/journal` | Fava-compatible transaction markup, directive/flag filters, FQL/time, deterministic page/order, posting expansion, context/source/export | no matches, invalid FQL/time, last/empty page, loading, stale/error | 3 | legacy |
| R-ACCOUNT | `/account/<name>` | Account details and up-to-date state, account Journal/running balance, balance/changes charts, budgets, source/context/export | unknown/closed account, no activity, missing valuation, invalid filter, error | 3 | legacy |
| R-HOLD-ACCOUNT | `/holdings/by_account` | Holdings grouped by account with units/cost/value and lot detail | no lots, unavailable price/conversion, query error | 4 | legacy |
| R-HOLD-CURRENCY | `/holdings/by_currency` | Holdings grouped by currency | no lots, unavailable price/conversion, query error | 4 | legacy |
| R-HOLD-ROOT | `/holdings/by_root_account` | Holdings grouped by root account | no lots, unavailable price/conversion, query error | 4 | legacy |
| R-HOLD-COMMODITY | `/holdings/by_commodity` | Holdings grouped by commodity | no lots, unavailable price/conversion, query error | 4 | legacy |
| R-COMMODITIES | `/commodities` | Commodity metadata, names, precisions, filters and price history | no match/history, missing prices, error | 4 | legacy |
| R-DOCUMENTS | `/documents` | Safe document list, account grouping, preview, source navigation and reviewed mutations | no documents, missing/unsafe path, upload/move/delete/attach error | 4 read / 6 write | legacy |
| R-EVENTS | `/events` | Event rows, filters and source navigation | no events/matches, invalid filter, error | 4 | legacy |
| R-STATISTICS | `/statistics` | Deterministic metrics, charts and accessible tables | no data, unavailable chart, calculation error | 4 | legacy |
| R-ERRORS | `/errors` | Conditional navigation, diagnostics, source anchors, FQL/budget/import errors | zero diagnostics removes nav entry, stale/error | 4 | legacy |
| R-HELP | `/help/<slug>` | Searchable standard help, shortcuts and OrangeCount boundary guidance | unknown/no-match topic, load error | 4 | legacy |
| R-QUERY | `/query` | CodeMirror BQL editor, run/save, typed table/string result, sort and CSV/XLSX/ODS | empty result, parse/evaluation error preserving prior result, export error | 5 | legacy |
| R-EDITOR | `/editor` | Source tree, CodeMirror, format, diagnostics, validate, save, revert and source slice | empty diagnostics, invalid source, hash conflict, write/reload/rollback error | 6 | legacy |
| R-IMPORT | `/import` | Native file selection/upload, importer mapping, extract, editable candidates, preview/diff/review/commit | no candidates, unsupported Python importer, invalid candidate, commit/rollback error | 6 | legacy |
| R-OPTIONS | `/options` | Every applicable built-in Fava option with actual effect and explicit excluded-option deviations | unset value, invalid change, persistence error | 6 | legacy |

## Global and modal registry

| ID | Surface | Required outcomes | Wave | Current status |
| --- | --- | --- | --- | --- |
| G-SHELL | Header, Sidebar, PageTitle, responsive menu | Fava composition, title/default route, active state, desktop/narrow behavior, focus return | 1 | prototype |
| G-FILTERS | Account, FQL, time, conversion, interval and URL state | complete parsing, precedence, direct URL/reload/history, explicit reset | 1–3 | legacy/prototype |
| G-THEME | System/light/dark and fonts | Fava tokens and Fira fonts; deterministic strict light/dark baselines | 1 | prototype |
| G-LOCALE | English and `zh-CN` | English visual authority; Chinese same components and structural invariants | 1–7 | legacy/prototype |
| G-KEYBOARD | Global `g …` navigation and route shortcuts | Fava shortcut behavior, visible focus and no write bypass | 1–7 | planned |
| M-ADD | Add Entry modal | Fava entry forms, validation, reviewed write publication | 6 | planned |
| M-CONTEXT | Entry Context and Source Slice | balances, source location, slice editing and recoverable errors | 3/6 | planned |
| M-EXPORT | Journal/query/report export | filtered Beancount, CSV/XLSX/ODS and print with exact values | 2/3/5 | planned |
| M-DOCUMENT | Upload/preview/attach/move/delete | containment, confirmation, recoverable cross-file failure and focus return | 4/6 | planned |
| M-NOTIFY | Loading, error and notification surfaces | Fava placement, dismissal, focus behavior and stale-state clarity | 1–7 | prototype/legacy |

## Statement and download surfaces

| ID | Surface | Required outcomes | Wave | Current status |
| --- | --- | --- | --- | --- |
| D-STATEMENT | statement metadata link/download | entry-hash/key resolution, Document Root containment, missing/unsafe error without path leakage | 3 | planned |
| D-DOCUMENT | document download/preview | normalized contained path, correct media/download behavior, denial state | 4 | legacy |
| D-JOURNAL | filtered Journal Beancount export | deterministic grouping/order and exact snapshot values | 3 | planned |
| D-QUERY-CSV | Query CSV | exact values and typed headings | 5 | legacy |
| D-QUERY-XLSX | Query XLSX | exact Go-native spreadsheet output | 5 | planned |
| D-QUERY-ODS | Query ODS | exact Go-native spreadsheet output | 5 | planned |
| D-PRINT | report/Journal print | Fava print composition without navigation clutter | 2–5 | planned |

## Explicit exclusions

| ID | Surface | Reason |
| --- | --- | --- |
| X-EXT | third-party extension pages/modules | Outside Fava standard surface and requires user extension execution |
| X-PLUGIN | Python plugin execution | OrangeCount preserves/diagnoses declarations but never executes Python plugins |
| X-PYIMPORT | Python/Beangulp importer execution | Replaced by native adapters and migration diagnostics |
| X-PUBLICAPI | public Fava HTTP API compatibility | Adapter is loopback-only and private to the embedded frontend |
| X-HOSTING | multi-user or multi-ledger hosting UI | OrangeCount is a single-owner local web session |
| X-PRIVATE | private-ledger baselines or retained browser evidence | Privacy discipline permits transient smoke only |

## Acceptance record requirements

For an `accepted` status, the route's record must link:

1. contract-map rows and exact semantic tests;
2. browser behavior/safety tests for all applicable states;
3. four English Chromium baseline cells, Chinese structural results and visual-agent diff report;
4. performance, accessibility, WebKit/Firefox, offline/CSP, provenance/license and deterministic-build evidence;
5. user approval for any changed baseline or Approved Fava deviation.
