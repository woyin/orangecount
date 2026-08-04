# Fava 1.30.12 source inventory

Mandatory complete inventory of the Fava 1.30.12 reference (ADR-0034) before
any migration work. Every file below was read at the pinned commit; decisions
are `adopt` (copy/adapt the frontend unit), `adapt` (frontend unit adapted for
the Go adapter), `rewrite` (independent OrangeCount implementation with
equivalent observable behavior), or `exclude` (not adopted: Python runtime,
plugins, extensions, HTTP API, template backend pieces). No production
OrangeCount UI or Go semantic files were changed by this inventory.

Reference facts (see `docs/fava-reference-lock.md`):

- Commit `aa7538e8971252c9efc52c8a516a3a77d604553f` (`v1.30.12`, `deps & lint`)
- Mirror `$HOME/.orangecount/fava-1.30.12-reference` (external, read-only)
- Bank of facts: 502 tracked files; `frontend/` 249, `src/` 99, `tests/` 106,
  `contrib/` 14, `docs/` 9, `stubs/` 4.

## 1. Runtime architecture (context for decisions)

Fava 1.30.12 is a Flask WSGI app (`src/fava/application.py`) that serves
Jinja templates (`src/fava/templates/_layout.html`) containing:

- a full HTML page on the first load: `<script type="application/json"
  id="ledger-data">` bootstrap, `id="ledger-mtime"`, `id="translations"`,
  `id="page-title"`, CSS/JS asset links;
- partial (`?partial=true`) fragments for backend-rendered pages
  (`help`, `jump` redirect, document/statement downloads, extension pages).

The Svelte frontend (`frontend/src/app.ts`) mounts into the `article` element,
initializes the router (`frontend/src/router.ts`) over the 15 client-side
routes (`frontend/src/reports/routes.ts`) and calls the JSON API
(`src/fava/json_api.py`) mounted at `/&lt;bfile&gt;/api/...`. All JSON API
responses are wrapped as `{"data": ..., "mtime": "..."}`
(`json_success`); errors are `{"error": "..."}` with HTTP status. The
frontend validates every payload at runtime with hand-written validators
(`frontend/src/lib/validation.ts`, `frontend/src/api/validators.ts`).

The Python/Beancount ecosystem (`src/fava/core/...`, `fava.beans`,
`fava.ext`, `fava.plugins`, `beanquery`, `beangulp`) is the semantic backend
authority in Fava. OrangeCount replaces it with `internal/ledger`,
`internal/query`, `internal/report`, `internal/snapshot`, `internal/source`
(Beancount v3 authority, ADR-0026). None of it is adopted.

## 2. Backend routes (source: `src/fava/application.py` `_setup_routes`)

| Route | Handler | Frontend surface | Decision | OrangeCount owner |
| --- | --- | --- | --- | --- |
| `/` and `/<bfile>/` | `index` — redirect to default page | URL bootstrap | rewrite | `internal/web` server index + redirect; default page from fava-option equivalent |
| `/<bfile>/account/<name>/` | shell for client-side account page | account report | rewrite (route exists) | `internal/web/assets/app.js` account path `/account/<name>` + `internal/web/server.go` |
| `/<bfile>/document/` | `document` — download attachment (query `filename`) | document preview/links | rewrite (attachment containment) | `internal/web` `handleDocument` (`/documents/`) + `internal/source` `DocumentRoots` |
| `/<bfile>/statement/` | `statement` — download via `entry_hash`+`key` metadata | statement links in journal | excluded (editor/statement metadata flow not in scope v0; revisit) | `internal/web` question: needs entry-hash → path resolution from the Go snapshot |
| `/<bfile>/holdings/by_<key>/` | shell for client-side holdings variants | holdings | rewrite | `internal/web/assets/app.js` holdings variants over `/api/v1/reports/holdings` |
| `/<bfile>/<report_name>/` | shell for the 15 client-side reports | all report pages | rewrite | `internal/web` report handlers + uploaded frontend components |
| `/<bfile>/extension/<name>/<endpoint>` | extension endpoint | extension pages | **excluded** (ADR-0024, no plugin/extension ecosystem) | n/a |
| `/<bfile>/extension_js_module/<name>.js` | extension JS module | extensions | **excluded** | n/a |
| `/<bfile>/extension/<name>/` | extension report | extension pages | **excluded** | n/a |
| `/<bfile>/download-query/query_result.<fmt>` | query→CSV/XLSX/ODS | QueryLinks | rewrite | `internal/query` `Result.WriteCSV`; XLSX/ODS excluded unless later approved |
| `/<bfile>/download-journal/` | render filtered entries as Beancount | Export modal | rewrite | question: OrangeCount source export from snapshot (`internal/ledger` render) |
| `/<bfile>/help/<slug>` | markdown help pages | Help | adapt (rewrite translations; keep page set) | `internal/web` help endpoints + uploaded help component; zh-CN/en |
| `/jump` | redirect rewriting current filters | sidebar links | rewrite | not needed as a server route; frontend URL handling |
| `/<bfile>/api/*` | `src/fava/json_api.py` blueprint | all frontend data | **excluded as public API**; the shapes are re-expressed loopback-only (ADR-0033) | `internal/web` private Fava-shaped endpoints |

### JSON API endpoints (`src/fava/json_api.py`) and their frontend consumers

The adapter must supply the same request/response shapes the uploaded
frontend expects. Complete endpoint list with exact wire contracts:

| Endpoint | Method+params | Response shape | Consumed by |
| --- | --- | --- | --- |
| `changed` | GET () | `bool` | `app.ts` poll (5s), mtime wiring |
| `errors` | GET () | `[{type, message, source{filename,lineno}|null}]` | `stores/index.ts` errors |
| `ledger_data` | GET () | `LedgerData` (see `get_ledger_data` / `LedgerData` dataclass, `internal_api.py`) | bootstrap in `_layout.html` `#ledger-data` |
| `payee_accounts` | GET (payee) | `string[]` | Transaction form |
| `query` | GET (query_string, account, filter, time) | `{t:"table",types:[{name,dtype}],rows:[[...]]}` or `{t:"string",contents}` or `{error}` | Query report, Holdings report, Statistics postings-per-account |
| `extract` | GET (filename, importer) | serialised entries | Import Extract |
| `context` | GET (entry_hash) | `{entry, balances_before, balances_after}` | Context modal |
| `source_slice` | GET (entry_hash) | `{slice, sha256sum}` | Context modal slice editor |
| `move` | PUT (account, new_name, filename) | string | Documents move |
| `payee_transaction` / `narration_transaction` | GET (payee/narration) | serialised Transaction or null | Transaction form autofill |
| `narrations` | GET | `string[]` | Transaction form suggestions |
| `source` | GET (filename) / PUT (file_path, source, sha256sum) | `SourceFile{file_path,sha256sum,source}` → updated sha256sum | Editor |
| `source_slice` PUT/DELETE | entry_hash, source, sha256sum | string | SliceEditor |
| `format_source` | PUT (source) | aligned string | editor align, CodeMirror `Control-d` |
| `document` | DELETE (filename) | string | Documents delete |
| `add_document` | PUT multipart | string | DocumentUpload |
| `attach_document` | PUT (filename, entry_hash) | string | journal context |
| `add_entries` | PUT (entries[]) | string | AddEntry modal, import save |
| `upload_import_file` | PUT multipart | string | ImportFileUpload |
| `journal` | GET () | serialised entries (filtered) | not used by current Svelte journal (uses `journal_page`) |
| `journal_page` | GET (page, order, account, filter, time, conversion, interval) | `{page,total_pages,journal:<html>`} | JournalTable (server-rendered HTML) |
| `events` | GET (filters) | serialised Events | Events report |
| `imports` | GET () | `FileImporters` list | Import report |
| `documents` | GET (filters) | serialised Documents | Documents report |
| `options` | GET () | `{fava_options:{k:v}, beancount_options:{k:v}}` | Options report |
| `commodities` | GET (filters) | `[{base,quote,prices:[[date,number]]}]` | Commodities report |
| `income_statement` / `balance_sheet` / `trial_balance` | GET (filters, conversion, interval) | `TreeReport{date_range,charts,trees}` | tree reports |
| `account_report` | GET (a, r, filters, conversion, interval) | `AccountReportJournal{charts,journal}` or `AccountReportTree{charts,interval_balances,budgets,dates}` | AccountReport |
| `statistics` | GET (filters) | `{all_balance_directives, balances:{account:Inventory}, entries_by_type}` | Statistics report |

The `mtime` wrapper is used by `set_mtime` (`stores/mtime.ts`) to drive
change polling and reload; the adapter must emit it for any endpoint the
uploaded frontend feeds into `fetch_and_handle_api_call`.

## 3. Frontend inventory with decisions

### 3.1 Build and tooling (`frontend/` root)

| File | Role | Decision |
| --- | --- | --- |
| `build.ts` | esbuild + esbuild-svelte bundle of `src/app.ts` → `src/fava/static`; splitting, wasm/woff loaders | adapt (reproduce for `web/` build emitting into `internal/web/assets/`) |
| `test.ts`, `setup.js` | node:test runner + svelte compile hook for tests | adapt (test harness in `web/`) |
| `package.json`, `package-lock.json`, `.npmrc` | npm manifest + lock (Node>=22.18) | adopt lock as provenance; dependency set reviewed in §5 |
| `tsconfig.json`, `eslint.config.js`, `prettier.config.cjs`, `stylelint.config.js`, `deno.json` (deno build task), `biome.json` (repo root used by pre-commit) | lint/format | adopt selectively for `web/` toolchain; not shipped |
| `sync-pre-commit.ts` | syncs eslint/prettier config | exclude (repo tooling detail) |
| `css/*.css` (13 files) | global styles: `base, charts, components, editor, fonts, grid, help, journal-table, layout, notifications, style, tree-table` | **adopt** (MIT licenses on fonts imported via @fontsource), with zh-CN font/layout adjustments allowed |
| `src/fava/static/favicon.ico` | only committed static asset | exclude from adoption; question: use own favicon or adopt under MIT |

### 3.2 App shell, router, state (`frontend/src` root + `sidebar` + `stores`)

| File | Role | Decision |
| --- | --- | --- |
| `app.ts` | entry: custom elements, change polling, router init, sidebar init, keyboard, theme | **adapt** (remove extension handling, tie to Go adapter bootstrap) |
| `router.ts` | SPA router: intercept clicks, history, partial reload, search-param sync, overlay close | **adapt** (base URL, no backend partial pages) |
| `helpers.ts` | `getUrlPath`, `urlFor*` builders | **adapt** |
| `i18n.ts` | `_()` from `#translations` script tag | **adapt** (Go-supplied catalogs en/zh-CN) |
| `keyboard-shortcuts.ts` | global `g x` sequences, tooltips, modKey | **adapt** |
| `notifications.ts`, `clipboard.ts`, `log.ts` | toasts, copyable, console logging | **adapt** (small) |
| `extensions.ts`, `extension-api.d.ts`, `svelte-custom-elements.ts`, `ambient.d.ts` | extensions + legacy custom elements | **exclude** (no extensions/plugin ecosystem; custom elements only needed by extensions) |
| `format.ts` | locale formatting, d3-format, date formats, incognito | **adapt** (incognito off / own redaction) |
| `AutocompleteInput.svelte` | ARIA combobox autocomplete | **adapt** (used by filters, forms) |
| `sidebar/` (13 files incl. `Header`, `AsideWithButton`, `AsideContents`, `FilterForm`, `SidebarLink`, `AccountSelector`, `AccountIndicator`, `AccountPageTitle`, `PageTitle`, `page-title.ts`, `index.ts`, `HeaderIcon`, `HeaderAndAside`) | shell header/sidebar/nav/filters/badges | **adapt** (no multi-ledger dropdown, no extensions list, no external `beancount://` by default) |
| `stores/index.ts` | `ledgerData` store + derived accounts/currencies/… | **adapt** (reticulate to adapter `ledger_data` payload) |
| `stores/url.ts` | URL-derived stores: conversion, interval, charts, synced params | **adapt** (own route base) |
| `stores/options.ts`, `stores/fava_options.ts` | ledger options + fava options stores | **adapt** (supported subset; plugin options excluded) |
| `stores/filters.ts` | time/account/fql filter stores | **adapt** |
| `stores/journal.ts`, `stores/query.ts`, `stores/editor.ts`, `stores/chart.ts`, `stores/color_scheme.ts`, `stores/mtime.ts`, `stores/format.ts`, `stores/accounts.ts` | localStorage-synced prefs + derived formatting/toggling | **adapt** (localStorage keys are Fava-derived; keep semantics, not the `fava-` prefix) |

### 3.3 `api/` and `lib/`

| File | Role | Decision |
| --- | --- | --- |
| `api/index.ts` | typed endpoint wrappers (`get_*`, `put_*`) over `{data,mtime}` envelope | **rewrite into Go-adapter client** (same function names/shapes, different transport; keep the exact validators) |
| `api/validators.ts` | runtime validators for all payloads | **adapt** (keep as contract tests against Go adapter output) |
| `lib/validation.ts` (combinators), `lib/result.ts`, `lib/json.ts`, `lib/dom.ts`, `lib/fetch.ts`, `lib/errors.ts`, `lib/focus.ts`, `lib/fuzzy.ts`, `lib/equals.ts`, `lib/objects.ts`, `lib/regex.ts`, `lib/set.ts`, `lib/array.ts`, `lib/store.ts`, `lib/paths.ts`, `lib/tree.ts`, `lib/sources.ts`, `lib/account.ts`, `lib/interval.ts`, `lib/iso4217.ts` | pure TS helpers | **adapt** (or rewrite; small files; keep behavior; `iso4217.ts` is an ISO-4217 code set ~186 lines) |

### 3.4 Reports (`frontend/src/reports/`)

Route registry: `routes.ts` (15 routes) — all are frontend-rendered.

| Route | Components (files) | Data from API | Decision |
| --- | --- | --- | --- |
| `income_statement` | `tree_reports/IncomeStatement.svelte`, `index.ts` | `income_statement` | adopt |
| `balance_sheet` | `tree_reports/BalanceSheet.svelte`, `index.ts` | `balance_sheet` | adopt |
| `trial_balance` | `tree_reports/TrialBalance.svelte`, `index.ts` | `trial_balance` | adopt |
| `journal` | `journal/Journal.svelte`, `JournalTable.svelte`, `JournalFilters.svelte`, `JournalHeaders.svelte`, `click_handler.ts`, `sort.ts`, `index.ts` | `journal_page` (HTML!) | **adapt with the largest rewrite**: Fava renders journal rows server-side via `_journal_table.html` into HTML; OrangeCount must instead render rows client-side from a JSON contract (Go adapter `journal_page` equivalent returning structured rows), or re-implement the template renderer in Go. This is the single biggest adapter decision in the project. |
| `query` | `query/Query.svelte`, `QueryEditor.svelte`, `QueryBox.svelte`, `QueryTable.svelte`, `QueryLinks.svelte`, `ReadonlyQueryEditor.svelte`, `query_table.ts`, `index.ts` | `query` | adopt (Go BeanQuery backend, exact values; `query_table.ts` validators stay) |
| `editor` | `editor/Editor.svelte`, `EditorMenu.svelte`, `Sources.svelte`, `AppMenu*.svelte`, `Key.svelte`, `index.ts`, `stores.ts` | `source`, `errors`, `format_source`, `source_slice` | adopt (Go save/validate path: `internal/web` editor handlers already exist) |
| `import` | `import/Import.svelte`, `Extract.svelte`, `FileList.svelte`, `ImportFileUpload.svelte`, `index.ts` | `imports`, `extract`, `add_entries`, `move`, `document`, `upload_import_file` | adopt-shaped; **rewrite semantics**: OrangeCount uses native CSV/generic adapters, never Python importers (`internal/web` import handlers exist; `imports` payload must be re-expressed as file candidates) |
| `options` | `options/Options.svelte`, `OptionsTable.svelte`, `index.ts` | `options`, color scheme store | adopt (supported options; plugin-option rows excluded) |
| `holdings` | `holdings/Holdings.svelte`, `index.ts` (4 aggregation queries) | `query` | adopt (queries run against Go BeanQuery; `units/value` semantics per v3) |
| `commodities` | `commodities/Commodities.svelte`, `CommodityTable.svelte`, `index.ts` | `commodities` | adopt |
| `documents` | `documents/Documents.svelte`, `Table.svelte`, `Accounts.svelte` (move/drag), `DocumentPreview.svelte`, `stores.ts`, `index.ts` | `documents`, `move`, `add_document`, `document` | adapt (attachment root containment enforced in Go; preview editor kept) |
| `events` | `events/Events.svelte`, `EventTable.svelte`, `index.ts` | `events` | adopt |
| `statistics` | `statistics/Statistics.svelte`, `EntriesByType.svelte`, `UpdateActivity.svelte`, `index.ts` | `statistics`, `query` | adopt |
| `errors` | `errors/Errors.svelte`, `index.ts` | `errors` | adopt |
| `account` | `accounts/AccountReport.svelte`, `index.ts` | `account_report` | adopt (Go owner: `internal/report` accounts/journal + charts) |
| shared | `route.ts`, `route.svelte.ts`, `ReportLoadError.svelte` | router contracts | adapt |

### 3.5 Charts (`frontend/src/charts/`)

| File | Role | Decision |
| --- | --- | --- |
| `index.ts`, `context.ts`, `helpers.ts`, `tooltip.ts`, `query-charts.ts` | parsing/context/axes/tooltip | adopt |
| `bar.ts`, `line.ts`, `hierarchy.ts`, `scatterplot.ts` | chart model classes + validators | adopt (hierarchy/line take the v3-pivoted currency projections) |
| `Chart.svelte`, `ChartSwitcher.svelte`, `ChartLegend.svelte`, `BarChart.svelte`, `LineChart.svelte`, `ScatterPlot.svelte`, `HierarchyContainer.svelte`, `Treemap.svelte`, `Sunburst.svelte`, `Icicle.svelte`, `Axis.svelte`, `ModeSwitch.svelte`, `SelectCombobox.svelte`, `ConversionAndInterval.svelte` | rendering | adopt |

### 3.6 Tree tables and entry forms

| File | Role | Decision |
| --- | --- | --- |
| `tree-table/*` (TreeTable, TreeTableNode, AccountCell, AccountCellHeader, Diff, IntervalTreeTable, IntervalTreeTableNode, helpers.ts, tree-table-custom-element.ts) | account trees, interval trees, expand/collapse, not-shown logic | adopt (tree-table-custom-element.ts used only by legacy extensions → exclude) |
| `entries/index.ts`, `amount.ts`, `cost.ts`, `metadata.ts`, `position.ts` | serialised entry models + validators | adopt (contract tests) |
| `entry-forms/*` (Entry, Transaction, Posting, Balance, Note, AccountInput, EntryMetadata, AddMetadataButton) | AddEntry/import editing forms | adopt (submission goes to Go `add_entries`-equivalent that validates/inserts via existing editor save path) |

### 3.7 Codemirror (`frontend/src/codemirror/`)

| File | Role | Decision |
| --- | --- | --- |
| `base-extensions.ts`, `dom.ts`, `ruler.ts`, `editor-transactions.ts`, `types.ts` | CM shell | adopt |
| `beancount-language.ts`, `beancount-highlight.ts`, `beancount-indent.ts`, `beancount-fold.ts`, `beancount-format.ts`, `beancount-autocomplete.ts`, `beancount-snippets.ts`, `tree-sitter-parser.ts`, `tree-sitter-beancount.wasm`, `beancount.ts` | Beancount language support (tree-sitter WASM, Lezer bridging) | **adopt** — WASM binary is a pre-built parser from `yagebu/tree-sitter-beancount` committed upstream; record its upstream origin in provenance |
| `bql.ts`, `bql-language.ts`, `bql-grammar.ts`, `bql-highlight.ts`, `bql-autocomplete.ts`, `bql-stream-parser.ts` | BQL/query language support | adopt (grammar JSON generated by `contrib/scripts.py`; keep generated file + note) |

### 3.8 Modals and editor primitives

| File | Role | Decision |
| --- | --- | --- |
| `modals/ModalBase.svelte`, `Modals.svelte`, `AddEntry.svelte`, `Context.svelte`, `EntryContextBalances.svelte`, `EntryContextLocation.svelte`, `Export.svelte`, `DocumentUpload.svelte`, `document-upload.ts` | dialogs (add transaction, entry context, export, document upload) | adapt (AddEntry/context need `add_entries`/`context`/`source_slice` Go equivalents; Export uses `download-journal`) |
| `editor/DeleteButton.svelte`, `SaveButton.svelte`, `SliceEditor.svelte`, `DocumentPreviewEditor.svelte` | save UI + slice editor | adapt |
| `sort/index.ts`, `sort/SortHeader.svelte`, `sort/sortable-table.ts` | typed sorting | adopt |

## 4. Fava backend modules (all `exclude` for adoption; read for semantics)

| Module | Contents read | OrangeCount semantic owner |
| --- | --- | --- |
| `src/fava/core/__init__.py` | `FavaLedger`, `FilteredLedger`, pagination (1000/page), interval balances, account journal, context, statement path | `internal/snapshot`, `internal/report`, `internal/web` (pagination/interval semantics re-expressed in Go) |
| `core/tree.py` | account tree, cap/transfer for balance sheet, net profit | `internal/report` (TrialBalanceTree/BalanceSheet project hierarchy) |
| `core/charts.py`, `internal_api.py` ChartApi | balances/bar line-chart data, net worth | `internal/report/charts.go` |
| `core/conversion.py` | at_cost/at_value/units/currency conversions | `internal/report` valuation (v3 price map) |
| `core/filters.py` | FQL syntax lexer/parser (`#tag`, `^link`, `payee:`, `any()`, `all()`, amounts, ranges) | question: OrangeCount `internal/report` `Filter`/`FilterJournal` implements subset; full FQL parity needs a Go parser (open) |
| `core/fava_options.py` | 27 fava-options parsed from `custom "fava-option"` | `internal/web` `handleOptions` (supported subset, ADR decision in plan) |
| `core/accounts.py` | account details: close date, uptodate status, balance string | question: `account_details`/uptodate indicator data not yet produced by Go — adapter additions needed |
| `core/attributes.py` | ranked accounts/currencies/payees/tags/links/years/narrations | `internal/report` + `internal/web` (partial; ranking util is small) |
| `core/ingest.py`, `beangulp` | import extraction (Python importers) | **excluded**; OrangeCount native CSV/bean adapters (ADR plan P7) |
| `core/query_shell.py`, `core/query.py`, `beanquery` | BQL shell + serialised table columns (`COLUMNS` dtype map) | `internal/query` |
| `core/budgets.py` | budget custom entries | question: OrangeCount has no budget model; tree-table/account budget overlays excluded until a model exists |
| `core/documents.py` | document path containment | `internal/source/roots.go` |
| `core/file.py` | source read/write, sha256 concurrency guards, entry slice editing | `internal/web` editor + `internal/source` |
| `core/commodities.py` | commodity names/precisions | question: `precisions`/`currency_names` in `ledger_data` — Go must compute from Commodity directives |
| `core/misc.py` | sidebar links, upcoming events, align | `internal/web` options subset |
| `core/group_entries.py`, `core/inventory.py`, `core/number.py`, `core/watcher.py`, `core/module_base.py`, `core/extensions.py` | grouping, inventories, decimal formatting, file watcher, extensions | `internal/ledger`, `internal/report`, `internal/snapshot` watcher; extensions excluded |
| `util/date.py` (576 lines) | time-filter parsing: `2015`, `2016-Q1`, `fy2018`, relative `year-1`, dateranges, fiscal year end | question: OrangeCount implements `time` filter as year/month prefix + from/to; full Fava time syntax incl. ranges/fiscal-year needs Go port decisions |
| `util/excel.py`, `util/ranking.py`, `util/sets.py`, `util/__init__.py` | xlsx/ods export, decay ranking, sets, misc | excel export excluded unless approved; ranking adapt |
| `serialisation.py`, `beans/*` (abc, account, create, flags, funcs, helpers, ingest, load, prices, protocols, str, types, `__init__`) | Beancount data model wrappers, hash_entry, price map, position strings | `internal/ledger` (v3 semantics) |
| `plugins/*` | `link_documents`, `tag_discovered_documents` | **excluded** (plugins never executed; OrangeCount diagnoses) |
| `ext/*` (auto_commit, portfolio_list, fava_ext_test) | example extensions | **excluded** |
| `help/*.md` (9 pages) | user docs | adapt as localized help content (rewrite to OrangeCount behavior; budgets/extensions/import pages re-scoped) |
| `templates/*` (`_layout.html`, `_journal_table.html`, `_query_table.html`, `beancount_file`, `help.html`, macros) | server templates | **excluded as runtime**; `_journal_table.html` and `_query_table.html` inform the Go serializer/CSV contracts |
| `translations/*.po` | 17 locales | exclude (en + zh-CN catalogs re-expressed in Go/TS; note `zh` exists upstream) |
| `cli.py`, `application.py`, `context.py`, `_ctx_globals_class.py`, `helpers.py`, `template_filters.py`, `json_api.py` (889 lines), `internal_api.py` (221) | WSGI server + API | exclude application code; contracts per §2 |

## 5. Frontend dependency license inventory (`frontend/package-lock.json`, pinned)

npm `lockfileVersion 3`; license fields aggregated:

| License field | Count (of 372 packages) | Representative direct deps |
| --- | --- | --- |
| MIT | 294 | svelte, d3-array/axis/color/format/hierarchy/quadtree/scale/selection/shape/time/time-format, @codemirror/* (7), @lezer/common+highlight, @fontsource/fira-*, @ungap/custom-elements, typescript (+ types), prettier* |
| ISC | 31 | transitive toolchain |
| Apache-2.0 | 16 | eslint/*, typescript-eslint, @tsconfig/strictest |
| BSD-2-Clause | 15 | transitive |
| BSD-3-Clause | 5 | web-tree-sitter (Primary), jsdom deps |
| BlueOak-1.0.0 | 4 | transitive |
| MIT-0 | 4 | transitive |
| OFL-1.1 | 3 | @fontsource/* font assets |
| CC0-1.0 | 1 | transitive |
| Python-2.0 | 1 | transitive |
| (none) | 1 | `svg-tags@1.0.0` (dev-only transitive) |

Web-tree-sitter is BSD-3-Clause; the Beancount grammar WASM
(`tree-sitter-beancount.wasm`, from `yagebu/tree-sitter-beancount` v0.0.3)
needs its own license record in provenance before adoption. Python backend
deps (`uv.lock`, incl. Flask, beancount, beanquery, beangulp, Babel,
simplejson, cheroot, markdown2, ply, watchfiles) are reference-only, not
adopted.

## 6. Tests and build inventory (all `exclude` from product; repurpose? )

| Area | Files | Notes |
| --- | --- | --- |
| Python tests | `tests/*.py` (31 test modules) + `tests/__snapshots__/` and `tests/data/*.beancount` | snapshot JSON files are **contract gold**: `test_json_api-*.json`, `test_internal_api-test_get_ledger_data.json` define every API shape. Decide (open): commit sanitized re-expressions of these shapes as adapter contract fixtures; never copy the private/demo ledgers wholesale without review (they are Fava's own demo fixtures, MIT-licensed — acceptable as *reference*, must be re-expressed, not vendored). |
| Frontend tests | `frontend/test/*.test.ts` (21 files) + `helpers.ts`/`dom.ts` | adopt as the test corpus for `web/` (they load Python snapshot JSON — adjust the fixture loader to the Go adapter fixtures) |
| Build | `Makefile` (uv + npm orchestration), `_build_backend.py` | reference only |

## 7. Risks and strict first implementation slice

Risks (see also plan's risk table):

1. **Journal is server-rendered HTML in Fava** (`journal_page` returns HTML;
   `account_report` journal is HTML; `_journal_table.html` does the render).
   A naive "copy the HTML" keeps Python/Jinja semantics inside Go. Mitigation:
   define a structured journal contract in Go (rows with directive kind, flag,
   date, payee/narration, tags/links, metadata, postings, change/balance,
   source anchor) and port the visible structure to a Svelte template. This is
   a rewrite of presentation, not semantics.
2. **FQL filter syntax is a Python PLY grammar** (tags, links, payee/narration
   keys, `any(...)`, `all(...)`, amount comparisons). Go has a subset today
   (`internal/report.Filter` text match + `FilterJournal`). Full parity needs a
   Go FQL parser. First slice: text/search subset; document the gap.
3. **Time filter syntax** (`fy2018`, `2016-Q1`, relative `year-1`, ranges) is
   Python (`util/date.py`, 576 lines). Go parses year/month prefix + from/to.
   First slice: keep current Go time model; mark range/fiscal-year parity open.
4. **Charts require a weight/price map at as-of dates** — `linechart`,
   `interval_totals`, net worth, hierarchy trees use `FavaPriceMap` up to a
   date. Go `charts.go` already computes chart series with price-map lookups;
   parity concerns are the multi-currency "unavailable" behavior already
   documented in `fava-visual-gap-analysis.md` (pivot currencies per column).
5. **`ledger_data` bootstrap payload** is the frontend's source of truth
   (accounts, account_details, currencies, currency_names, precisions, fava
   options, options, years, payees, tags, links, user_queries, sidebar_links,
   upcoming_events_count, extension list, other_ledgers, incognito,
   have_excel). The Go adapter must produce this field-for-field or the
   frontend validators (`ledgerDataValidator`) fail. This is the P3 bootstrap
   contract and the first milestone.
6. **Dependency surface**: adopting charts/editor pulls in d3, @codemirror,
   web-tree-sitter WASM, @fontsource fonts. Bundle sizes and CSP must be
   re-audited; offline/no-CDN is already satisfied by esbuild bundling.
7. **Statement/download exports** (`statement/`, `download-journal`,
   `download-query.query_result.xlsx/ods`) depend on entry-hash→file render
   and Excel libs; first slice: CSV only, mark xlsx/ods/statement excluded
   pending decisions.
8. **Account `uptodate-indicator`** (green/yellow/red circles) requires
   close-date + last-entry + balance-check logic not yet present in Go —
   open question for account/statistics parity.

Strict first slice (P3+P4, after P0/P1/P2 foundation):

- (a) Go adapter bootstrap contract: `ledger_data` (full validator-compatible
  payload), `changed`, `errors`, `source`, `options`, static assets; all
  loopback-only.
- (b) Shell transplant: `_layout.html`-equivalent shell in `internal/web`
  serving `index.html` with `#ledger-data`/`#ledger-mtime`/`#translations`
  script tags; router + sidebar + header + filters + theme + keyboard.
- (c) One read-only vertical slice: Income Statement / Balance Sheet tree
  tables + hierarchy charts over `account_report`/tree endpoints, using the
  existing `internal/report` + `internal/report/charts.go`, wired through
  adapter endpoints that return `{data, mtime}` envelope with validated
  `SerialisedTreeNode` shapes.
- (d) Journal as a structured (not HTML) contract behind the `journal_page`
  shape, rendered client-side; accept only the subset of flags/directives that
  Go produces today.
- Defer: editor save, import, query, documents move/upload, options writes,
  help content, xlsx/ods exports, full FQL/time parser, uptodate indicators,
  budgets.

## 8. Open questions (blocking later packages, not P0)

| # | Question | Where it blocks |
| --- | --- | --- |
| Q1 | Should `journal_page`/`account_report.journal` remain server-rendered HTML (Fava behavior) or become a structured JSON contract rendered in Svelte? Recommended: structured JSON (keeps Go the only serializer). | P4 journal/account slice |
| Q2 | Does OrangeCount port the FQL filter grammar (tags/links/payee/amount/`any`/`all`/ranges) to Go, or keep the current text-filter subset? | P4 journal; global filter parity |
| Q3 | Full Fava time-filter syntax (quarters, fiscal year end, relative `year-1`, ranges) in Go? | all period-dependent reports |
| Q4 | `account_details` + uptodate indicators (close date, last entry, balance check status): add to Go report layer? | account page, statistics, tree tables |
| Q5 | `statement/` and `download-journal/` exports and xlsx/ods query exports: in scope? (CSV now, others later?) | P4/P6/P7 exports |
| Q6 | Budget module (custom budget entries): exclude entirely, or implement a v3-compatible subset? | tree-table budget overlays, account balances |
| Q7 | `precisions`/`currency_names` from Commodity directives in Go `ledger_data` — confirm source of truth and format (already needed for Q1's bootstrap). | P3 bootstrap |
| Q8 | Fava demo fixtures in `tests/data/` and `tests/__snapshots__/`: reuse as sanitized contract fixtures (MIT) or re-express only shapes? | P3 contract fixtures |
| Q9 | zh-CN: Fava ships upstream `zh` and `zh_Hant_TW` catalogs (PO). OrangeCount's existing catalog is in `web/src/translations.ts`. Adopt Fava's zh strings (MIT, re-expressed) or keep OrangeCount's? | P2 shell locale |
| Q10 | `favicon.ico` and `HeaderIcon` logo: adopt under MIT or replace? | P2 shell |

## 9. Verification performed

- Mirror cloned, detached and pinned: HEAD == peeled tag commit
  `aa7538e8971252c9efc52c8a516a3a77d604553f`; `.git` made read-only.
- Every file in §3–§6 read or structurally inspected (manifests, headers,
  exports, route tables, tests). No private-ledger data was accessed, captured,
  or persisted; all facts above come from the public upstream checkout only.
- `git diff --check` on the OrangeCount worktree reports no whitespace errors
  from these documentation-only changes (see completion report).
