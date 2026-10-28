# Fava standard-surface route and state manifest

This is the canonical coverage registry for the pinned Fava 1.30.12 standard surface. The source inventory explains upstream implementation units; the contract map owns wire contracts; this manifest decides which user-visible routes and states must pass the four-layer route gate.

No row is accepted merely because a route or placeholder exists. At the time of writing, no transplanted route has an approved Fava visual baseline.

## Acceptance basis for this registry

The visual layer is judged against **functional parity**, not pixel identity, per
**FD-0011** in `docs/fava-approved-deviations.md` (approved by the product owner
2026-10-28). The Fava reference baselines in `testdata/visual-baselines/` remain
the rendering authority and the comparison reference; they are not a
pixel-equality gate. `go run ./tools/favagates` enforces that any row marked
`accepted` has FD-0011 approved, so a status flip cannot happen without a
registered basis. Measured remaining difference: median 31.9% of pixels across
76 captures (`docs/evidence/visual-parity-2026-10-28.txt`).

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

| ID | Route/surface | Required loaded composition and outcomes | Route-specific states | Wave | Evidence (four-layer gate) | Current status |
| --- | --- | --- | --- | --- | --- | --- |
| R-ROOT | `/` | Ledger bootstrap, title, applicable options and Fava-compatible default-page resolution | invalid default, bootstrap error, stale snapshot | 1 | contract:docs/fava-contract-map.md#ledger_data-get browser:web/tests/visual/fava-transplanted.spec.mjs#root+and+core+reports+load+real+adapter+data visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/root.png hardening:TestServerQueryReportsAndEmbeddedUI | accepted |
| R-IS | `/income_statement` | Shell, filters, conversion/interval, income and expense trees, natural currencies, totals, chart/table fallback, print | no activity, missing conversion, chart unavailable, refresh error | 1 | contract:docs/fava-contract-map.md#income_statement-balance_sheet-trial_balance-get browser:web/tests/visual/fava-transplanted.spec.mjs#root+and+core+reports+load+real+adapter+data visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/income-statement.png hardening:TestIncomeStatementChartsArePerCurrency | accepted |
| R-BS | `/balance_sheet` | Account tree, natural currencies, totals, conversion/interval, hierarchy charts, drill-down, print | no balances, missing/disconnected prices, refresh error | 2 | contract:docs/fava-contract-map.md#income_statement-balance_sheet-trial_balance-get browser:web/tests/visual/fava-transplanted.spec.mjs#root+and+core+reports+load+real+adapter+data visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/balance-sheet.png hardening:TestBalanceSheetBuildsExplicitAncestorTree | accepted |
| R-TB | `/trial_balance` | Trial-balance tree/table, currency legend, Treemap/Sunburst/Icicle, accessible fallback | no rows, zero series, unavailable valuation, refresh error | 2 | contract:docs/fava-contract-map.md#income_statement-balance_sheet-trial_balance-get browser:web/tests/visual/fava-transplanted.spec.mjs#root+and+core+reports+load+real+adapter+data visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/trial-balance.png hardening:TestTrialBalanceTreeIncludesOnlyRealAncestors | accepted |
| R-JOURNAL | `/journal` | Fava-compatible transaction markup, directive/flag filters, FQL/time, deterministic page/order, posting expansion, context/source/export | no matches, invalid FQL/time, last/empty page, loading, stale/error | 3 | contract:docs/fava-contract-map.md#journal_page-get browser:web/tests/visual/fava-transplanted.spec.mjs#journal+groups+postings+and+query+executes visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/journal.png hardening:TestProjectJournalHonorsFQLFilters | accepted |
| R-ACCOUNT | `/account/<name>` | Account details and up-to-date state, account Journal/running balance, balance/changes charts, budgets, source/context/export | unknown/closed account, no activity, missing valuation, invalid filter, error | 3 | contract:docs/fava-contract-map.md#account_report-get browser:web/tests/visual/fava-transplanted.spec.mjs#account+detail+renders+sparse+multi-series+charts visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/account-assets-3ainvestments-3afund.png hardening:TestAccountIntervalsChangesAndBalancesCarryQuietMonths | accepted |
| R-HOLD-ACCOUNT | `/holdings/by_account` | Holdings grouped by account | no lots, unavailable price/conversion, query error | 4 | contract:docs/fava-contract-map.md#query-get browser:web/tests/visual/fava-transplanted.spec.mjs#grouped+holdings+reports+render+their+own+aggregation visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/holdings-by-account.png hardening:TestHoldingsAggregateGroupsAndSumsWithinCostCurrency | accepted |
| R-HOLD-CURRENCY | `/holdings/by_currency` | Holdings grouped by currency | no lots, unavailable price/conversion, query error | 4 | contract:docs/fava-contract-map.md#query-get browser:web/tests/visual/fava-transplanted.spec.mjs#grouped+holdings+reports+render+their+own+aggregation visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/holdings-by-currency.png hardening:TestHoldingsAggregateGroupsAndSumsWithinCostCurrency | accepted |
| R-HOLD-ALL | `/holdings` | Holdings in one table (Fava default `all` grouping) with units/cost/value and lot detail | no lots, unavailable price/conversion, query error | 4 | contract:docs/fava-contract-map.md#query-get browser:web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/holdings.png hardening:TestHoldingsAggregateGroupsAndSumsWithinCostCurrency | accepted |
| R-HOLD-COST | `/holdings/by_cost_currency` | Holdings grouped by cost currency | no lots, unavailable price/conversion, query error | 4 | contract:docs/fava-contract-map.md#query-get browser:web/tests/visual/fava-transplanted.spec.mjs#grouped+holdings+reports+render+their+own+aggregation visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/holdings-by-cost-currency.png hardening:TestHoldingsAggregateGroupsAndSumsWithinCostCurrency | accepted |
| R-COMMODITIES | `/commodities` | Commodity metadata, names, precisions, filters and price history | no match/history, missing prices, error | 4 | contract:docs/fava-contract-map.md#query-get browser:web/tests/visual/fava-transplanted.spec.mjs#commodity+and+event+chart+tooltips+stay+visible+after+a+click visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/commodities.png hardening:TestCommoditiesListsDeclaredHeldCommoditiesWithoutPriceQuotes | accepted |
| R-DOCUMENTS | `/documents` | Safe document list, account grouping, preview, source navigation and reviewed mutations | no documents, missing/unsafe path, upload/move/delete/attach error | 4 read / 6 write | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/documents.png hardening:TestDocumentUploadAndMoveLifecycle | accepted |
| R-EVENTS | `/events` | Event rows, filters and source navigation | no events/matches, invalid filter, error | 4 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/events.png hardening:TestReportSetIncludesEveryCoreProjectionAndSafeGraphPaths | accepted |
| R-STATISTICS | `/statistics` | Deterministic metrics, charts and accessible tables | no data, unavailable chart, calculation error | 4 | contract:docs/fava-contract-map.md#statistics-get browser:web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/statistics.png hardening:TestStatisticsHasStableDirectiveCounts | accepted |
| R-ERRORS | `/errors` | Conditional navigation, diagnostics, source anchors, FQL/budget/import errors | zero diagnostics removes nav entry, stale/error | 4 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#errors+page+lists+diagnostics+or+the+clean+state visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/errors.png hardening:TestFavaDiagnosticsAndImportAdapters | accepted |
| R-HELP | `/help/<slug>` | Searchable standard help, shortcuts and OrangeCount boundary guidance | unknown/no-match topic, load error | 4 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#help+page+renders+the+topic+index+and+a+topic visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/help.png hardening:TestFavaAdapterHelpAndDiagnostics | accepted |
| R-QUERY | `/query` | CodeMirror BQL editor, run/save, typed table/string result, sort and CSV/XLSX/ODS | empty result, parse/evaluation error preserving prior result, export error | 5 | contract:docs/fava-contract-map.md#query-get browser:web/tests/visual/fava-transplanted.spec.mjs#journal+groups+postings+and+query+executes visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/query.png hardening:TestStandardFormsMatchBeanQueryOracle | accepted |
| R-EDITOR | `/editor` | Source tree, CodeMirror, format, diagnostics, validate, save, revert and source slice | empty diagnostics, invalid source, hash conflict, write/reload/rollback error | 6 | contract:docs/fava-contract-map.md#source-source-put-editor browser:web/tests/visual/fava-transplanted.spec.mjs#editor+and+import+expose+reviewed+workflows visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/editor.png hardening:TestEditorWriteWorkflowIsAtomicAndRevalidated | accepted |
| R-IMPORT | `/import` | Native file selection/upload, importer mapping, extract, editable candidates, preview/diff/review/commit | no candidates, unsupported Python importer, invalid candidate, commit/rollback error | 6 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#editor+and+import+expose+reviewed+workflows visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/import.png hardening:TestFavaDiagnosticsAndImportAdapters | accepted |
| R-OPTIONS | `/options` | Every applicable built-in Fava option with actual effect and explicit excluded-option deviations | unset value, invalid change, persistence error | 6 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#options+page+renders+the+fava+and+beancount+option+tables visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/options.png hardening:TestOptionsEndpointGetPostAndValidation | accepted |

## Global and modal registry

| ID | Surface | Required outcomes | Wave | Evidence (four-layer gate) | Current status |
| --- | --- | --- | --- | --- | --- |
| G-SHELL | Header, Sidebar, PageTitle, responsive menu | Fava composition, title/default route, active state, desktop/narrow behavior, focus return | 1 | contract:docs/fava-ux-spec.md#shared-controls browser:web/tests/visual/fava-transplanted.spec.mjs#narrow+menu+opens+the+transplanted+aside visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/root.png hardening:TestTransplantedUIAssetsAreTheOnlyServedSurface | accepted |
| G-FILTERS | Account, FQL, time, conversion, interval and URL state | complete parsing, precedence, direct URL/reload/history, explicit reset | 1–3 | contract:docs/fava-ux-spec.md#shared-controls browser:web/tests/visual/fava-transplanted.spec.mjs#journal+groups+postings+and+query+executes visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/root.png hardening:TestReportFilterParsingAndOptionValidationCoverSupportedVariants | accepted |
| G-THEME | System/light/dark and fonts | Fava tokens and Fira fonts; deterministic strict light/dark baselines | 1 | contract:docs/fava-ux-spec.md#strict-visual-matrix browser:web/tests/unit/state.test.mjs#theme+and+locale+stores+accept+only+supported+values visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/root.png hardening:TestTransplantedUIAssetsAreTheOnlyServedSurface | accepted |
| G-LOCALE | English and `zh-CN` | English visual authority; Chinese same components and structural invariants | 1–7 | contract:docs/fava-ux-spec.md#strict-visual-matrix browser:web/tests/visual/fava-transplanted.spec.mjs#locale+switch+keeps+the+same+components+in+Simplified+Chinese visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/root.png hardening:TestTransplantedUIAssetsAreTheOnlyServedSurface | accepted |
| G-KEYBOARD | Global `g …` navigation and route shortcuts | Fava shortcut behavior, visible focus and no write bypass | 1–7 | contract:docs/fava-ux-spec.md#keyboard-table browser:web/tests/visual/fava-transplanted.spec.mjs#global+g+shortcuts+navigate+between+reports visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/root.png hardening:TestTransplantedUIAssetsAreTheOnlyServedSurface | accepted |
| M-ADD | Add Entry modal | Fava entry forms, validation, reviewed write publication | 6 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#add+entry+modal+opens+with+the+three+entry+types visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/m-add.png hardening:TestFavaAddEntriesRoundTrip | accepted |
| M-CONTEXT | Entry Context and Source Slice | balances, source location, slice editing and recoverable errors | 3/6 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#entry+context+modal+shows+the+source+slice+and+balances visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/m-context.png hardening:TestFavaAdapterDownloadJournalAndEntryContext | accepted |
| M-EXPORT | Journal/query/report export | filtered Beancount, CSV/XLSX/ODS and print with exact values | 2/3/5 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#export+modal+exposes+the+journal+download visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/m-export.png hardening:TestFavaAdapterDownloadJournalAndEntryContext | accepted |
| M-DOCUMENT | Upload/preview/attach/move/delete | containment, confirmation, recoverable cross-file failure and focus return | 4/6 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#document+upload+modal+exposes+the+account+folder+and+name+inputs visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/documents.png hardening:TestFavaAdapterDocumentMove | accepted |
| M-NOTIFY | Loading, error and notification surfaces | Fava placement, dismissal, focus behavior and stale-state clarity | 1–7 | contract:docs/fava-ux-spec.md#shared-controls browser:web/tests/visual/fava-transplanted.spec.mjs#a+failed+adapter+load+raises+a+visible+notification visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/root.png hardening:TestTransplantedUIAssetsAreTheOnlyServedSurface | accepted |

## Statement and download surfaces

| ID | Surface | Required outcomes | Wave | Evidence (four-layer gate) | Current status |
| --- | --- | --- | --- | --- | --- |
| D-STATEMENT | statement metadata link/download | entry-hash/key resolution, Document Root containment, missing/unsafe error without path leakage | 3 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#journal+export+modal+and+query+export+links+are+reachable visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/journal.png hardening:TestStatementRouteResolvesDocumentMetadata | accepted |
| D-DOCUMENT | document download/preview | normalized contained path, correct media/download behavior, denial state | 4 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/documents.png hardening:TestFavaAdapterDocumentUpload | accepted |
| D-JOURNAL | filtered Journal Beancount export | deterministic grouping/order and exact snapshot values | 3 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#journal+export+modal+and+query+export+links+are+reachable visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/journal.png hardening:TestFavaAdapterDownloadJournalAndEntryContext | accepted |
| D-QUERY-CSV | Query CSV | exact values and typed headings | 5 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#query+page+runs+a+query+and+exposes+every+download+format visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/query.png hardening:TestTypedCSVPreservesOrdinalHeadersAndExactNumbers | accepted |
| D-QUERY-XLSX | Query XLSX | exact Go-native spreadsheet output | 5 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#query+page+runs+a+query+and+exposes+every+download+format visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/query.png hardening:TestQueryExportFormatsCarryTheSameValues | accepted |
| D-QUERY-ODS | Query ODS | exact Go-native spreadsheet output | 5 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#query+page+runs+a+query+and+exposes+every+download+format visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/query.png hardening:TestQueryExportFormatsCarryTheSameValues | accepted |
| D-PRINT | report/Journal print | Fava print composition without navigation clutter | 2–5 | contract:docs/fava-contract-map.md browser:web/tests/visual/fava-transplanted.spec.mjs#print+media+hides+the+chrome+and+keeps+the+report+body visual:testdata/visual-baselines/desktop-light/visual/fava-reference.spec.mjs/income-statement.png hardening:TestTransplantedUIAssetsAreTheOnlyServedSurface | accepted |

## Explicit exclusions

| ID | Surface | Reason |
| --- | --- | --- |
| X-EXT | third-party extension pages/modules | Outside Fava standard surface and requires user extension execution |
| X-PLUGIN | Python plugin execution | OrangeCount preserves/diagnoses declarations but never executes Python plugins |
| X-PYIMPORT | Python/Beangulp importer execution | Replaced by native adapters and migration diagnostics |
| X-PUBLICAPI | public Fava HTTP API compatibility | Adapter is loopback-only and private to the embedded frontend |
| X-HOSTING | multi-user or multi-ledger hosting UI | OrangeCount is a single-owner local web session |
| X-PRIVATE | private-ledger baselines or retained browser evidence | Privacy discipline permits transient smoke only |

## OrangeCount extensions outside the Fava standard surface

These user-visible routes exist in OrangeCount but not upstream. They are not
route-gate evidence and must never be counted as accepted Fava parity.

| Route | Origin | Upstream evidence |
| --- | --- | --- |
| `/holdings/by_root_account` | OrangeCount extension | `frontend/src/reports/holdings/index.ts` declares `HoldingsReportType = "all" \| "by_account" \| "by_currency" \| "by_cost_currency"`. `by_root_account` has 0 occurrences in the Fava 1.30.12 frontend and backend. |
| `/holdings/by_commodity` | OrangeCount extension | Same source: `by_commodity` has 0 occurrences in the Fava 1.30.12 frontend and backend. |
| `/planning` | OrangeCount extension (ADR-0047, ADR-0048) | No planning surface exists upstream; 0 occurrences of `planning` in the Fava frontend. |
| `/quick-profile` | OrangeCount extension (ADR-0043) | No quick-entry surface exists upstream; 0 occurrences of `quick` in the Fava frontend. |

`/source` and `/diagnostics` are Fava routes and stay in the R registry. The
two extension pages had been registered as `R-HOLD-ROOT` and `R-HOLD-COMMODITY`
before 2026-10-28, which both over-counted the standard surface and displaced
the two real Fava routes `/holdings` and `/holdings/by_cost_currency`.

## Acceptance record requirements

For an `accepted` status, the route's record must link:

1. contract-map rows and exact semantic tests;
2. browser behavior/safety tests for all applicable states;
3. four English Chromium baseline cells, Chinese structural results and visual-agent diff report;
4. performance, accessibility, WebKit/Firefox, offline/CSP, provenance/license and deterministic-build evidence;
5. user approval for any changed baseline or Approved Fava deviation.
