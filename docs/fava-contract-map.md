<!--
Copyright 2026 OrangeCount contributors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
-->

# Fava 1.30.12 → OrangeCount contract map

Initial contract map (P0). For every frontend request, a row records the Fava
source, the frontend call site, the request shape, the planned Go owner, the
semantic rule, response adaptation, tests, and provenance. This is the
self-check that "a reviewer can trace every in-scope page from a Fava source
module through its data contract to a planned OrangeCount owner" (ADR-0034).

References: `docs/fava-source-inventory.md` for decisions; `docs/fava-reference-lock.md`
for pinning; `docs/fava-provenance-inventory.md` for attribution.

## Conventions

- "Frontend call site" = symbol in `frontend/src/api/index.ts` (or a store/route
  loader) that invokes the endpoint.
- "Go owner" = the OrangeCount function that supplies the data. `internal/web`
  handlers are the adapter boundary; they delegate to `internal/report`,
  `internal/query`, `internal/snapshot`, `internal/ledger`, `internal/source`.
- "Response adaptation" = the exact JSON shape the embedded frontend requires
  (matching the Fava validators in `frontend/src/api/validators.ts`).
- `mtime` wrapper: every response is wrapped `{"data": ..., "mtime": "..."}`
  unless otherwise noted; the adapter owns mtime provenance.
- Rows marked **[open]** are unresolved decisions (see source-inventory §8).

## Route table (page → Fava route → adapter endpoint → Go owner)

| Page (frontend route) | Fava source module | Frontend call site | Adapter endpoint (Go) | Go owner | Provenance |
| --- | --- | --- | --- | --- | --- |
| Shell/bootstrap | `internal_api.get_ledger_data`, `_layout.html` `#ledger-data` | `get_ledger_data()` in `app.ts` | `ledger_data` | `internal/web` bootstrap DTO over `internal/snapshot` (`Store.Current().Evaluation()`) + `internal/report` | rewrite |
| Income Statement | `json_api.get_income_statement` | `get_income_statement` (tree_reports/index.ts) | `income_statement` | `internal/report.IncomeStatement` + `internal/report/charts.go` | adopt |
| Balance Sheet | `json_api.get_balance_sheet` | `get_balance_sheet` | `balance_sheet` | `internal/report.BalanceSheet` + charts | adopt |
| Trial Balance | `json_api.get_trial_balance` | `get_trial_balance` | `trial_balance` | `internal/report.TrialBalanceTree` + charts | adopt |
| Journal | `json_api.get_journal_page` + `_journal_table.html` | `get_journal_page` (journal/index.ts) | `journal_page` **[open Q1]** | `internal/report.JournalBetween` + `internal/report.FilterJournal`; Go serializer for rows | adapt (largest rewrite) |
| Account report | `json_api.get_account_report` | `get_account_report` (accounts/index.ts) | `account_report` | `internal/report.JournalBetween` scoped to account + `internal/report/charts.go` accountChart | adapt |
| Query | `json_api.get_query` + `core/query_shell.py` | `get_query` (query/index.ts, Query.svelte) | `query` | `internal/query.Evaluate` (BeanQuery semantics) | adopt |
| Holdings | `json_api.get_query` with 4 predefined queries | `get_query` (holdings/index.ts) | `query` (holding query strings) | `internal/query` + `internal/report.Holdings*`; queries re-expressed | adopt |
| Commodities | `json_api.get_commodities` | `get_commodities` (commodities/index.ts) | `commodities` | `internal/report.Prices` + price-map pairs | adopt |
| Documents | `json_api.get_documents` | `get_documents` (documents/index.ts) | `documents` | `internal/report.Documents` + `internal/source.DocumentRoots` containment | adapt |
| Events | `json_api.get_events` | `get_events` (events/index.ts) | `events` | `internal/report.Events` | adopt |
| Statistics | `json_api.get_statistics` | `get_statistics` (statistics/index.ts) | `statistics` | `internal/report.Statistics` + `internal/query` for postings-per-account | adopt |
| Errors | `json_api.get_errors` | `get_errors` (app.ts, errors/index.ts) | `errors` | `internal/report.ErrorsWithGraph` + `internal/diagnostic` | adopt |
| Editor | `json_api.get_source` / `put_source` | `get_source`, `put_source` (editor/index.ts, Editor.svelte) | `source` / `source` (PUT) | `internal/web` `handleEditor` + `handleEditorSave` (atomic write, snapshot reload) | adopt |
| Import | `json_api.get_imports` / `get_extract` / `put_add_entries` | `get_imports`, `get_extract`, `save_entries` (import/index.ts, Import.svelte) | `imports` / `extract` / `add_entries` | `internal/web` `handleImport*` (native CSV/bean adapters; Python importers **excluded**) | adapt |
| Options | `json_api.get_options` | `get_options` (options/index.ts) | `options` | `internal/web` `handleOptions` (supported subset) | adapt |
| Help | `application.help_page` | n/a (backend) | `help` | `internal/web` `handleHelp` + uploaded help content | adapt |
| Document attachment | `application.document` | `DocumentPreview`/`Table` | `document` (GET `/documents/`) | `internal/web` `handleDocument` + `internal/source.DocumentRoots` | adapt |
| Statement | `application.statement` **[open Q5]** | journal metadata links | `statement` | `internal/web` question: entry-hash→path resolution | open |
| Journal export | `application.download_journal` **[open Q5]** | `Export` modal | `download-journal` | `internal/ledger` render of snapshot entries | open |
| Query export | `application.download_query` | `QueryLinks` | `download-query.query_result.csv` (xlsx/ods open) | `internal/query.Result.WriteCSV` | open (CSV now) |
| `.jump` / sidebar links | `application.jump` | `SidebarLink` remote links | frontend URL handling | `internal/web` filter syncing | rewrite |

## Shared / cross-cutting contracts

### `ledger_data` bootstrap (P3 first milestone)

| Field | Fava source | Go owner | Notes |
| --- | --- | --- | --- |
| `accounts` | `attributes.accounts` | `internal/report` account set | ranked string[] |
| `account_details` | `core/accounts.py` | **[open Q4]** close date/last entry/uptodate | `AccountData` serialization |
| `base_url` | `url_for("index")` | `internal/web` | route base for the embedded app |
| `currencies` / `currency_names` / `precisions` | `attributes.currencies`, `core/commodities.py` | `internal/ledger` Commodity directives → **[open Q7]** | precisions from `precision` meta |
| `errors` | `get_errors()` | `internal/diagnostic` → `errors` | `[{type,message,source}]` |
| `fava_options` | `fava_options` dataclass | `internal/web` supported subset | 27 fields; exclude plugin-only |
| `options` | `_get_options()` | `internal/ledger` `Evaluation.Options` | title, filename, include, operating_currency, name_* |
| `payees`/`tags`/`links`/`years`/`narrations` | `attributes.*` | `internal/report`/`internal/query` | string[] |
| `user_queries` | `all_entries_by_type.Query` | `internal/ledger` Query directives | `[{name, query_string}]` |
| `sidebar_links` | `core/misc.py` | `internal/web` | custom `fava-sidebar-link` subset |
| `upcoming_events_count` | `core/misc.py` | `internal/report` Events | int |
| `extensions` / `other_ledgers` / `incognito` / `have_excel` | extensions / multi-ledger / incognito / excel | `internal/web` — emit empty/empty/false/false | extensions excluded |
| envelope `mtime` | `json_success` | `internal/web` snapshot mtime | from `Store.Current()` |

### `query` response (`{t:"table",types:[{name,dtype}],rows}`)

| Fava dtype | Go producer | Frontend validator (`query_table.ts`) | Notes |
| --- | --- | --- | --- |
| `bool` / `int` / `str` / `date` / `set` | `internal/query` row values | `boolean`/`number`/`string`/`Date`/`string[]` | |
| `Decimal` | `internal/query` exact decimal (string) | `number` (optional) | exact value preserved in machine export |
| `Amount` | `internal/query` `{number,currency}` | `Amount.validator` | |
| `Inventory` | `internal/report` projection | `Inventory.validator` → `{currency:number}` | multi-currency pivot target |
| `Position` | `internal/query` `{units,cost}` | `Position.validator` | |

### Tree report (`SerialisedTreeNode`)

| Field | Fava source | Go owner |
| --- | --- | --- |
| `account`, `balance`, `balance_children`, `children`, `has_txns`, `cost`, `cost_children` | `core/tree.py` `serialise` | `internal/report` account tree projection (natural currency pivot) |

## Per-request contract rows (P0 detail for the first slice)

### `ledger_data` (GET)

| Field | Required content |
| --- | --- |
| Fava source | `src/fava/internal_api.py:get_ledger_data`; `_layout.html` `#ledger-data` |
| Route/state | any page load; bootstrap |
| Request shape | GET, no params; full-page load only |
| Go owner | `internal/web` bootstrap adapter over `internal/snapshot.Store.Current()` + `internal/report` |
| Semantic rule | v3 authority; no plugin execution; snapshot may be stale but never partial |
| Response adaptation | exact `LedgerData` validator fields; `{data, mtime}` envelope |
| Tests | `internal/web` contract fixture; `web` frontend `initialiseLedgerData` corpus |
| Provenance | rewrite (bootstrap is a new OrangeCount DTO) |

### `income_statement` / `balance_sheet` / `trial_balance` (GET)

| Field | Required content |
| --- | --- |
| Fava source | `json_api.get_income_statement` etc.; `core/tree.py`; `core/charts.py` |
| Route/state | tree report pages; `account`, `filter`, `time`, `conversion`, `interval` |
| Request shape | GET with those query params |
| Go owner | `internal/report.IncomeStatement/BalanceSheet/TrialBalanceTree` + `internal/report/charts.go` |
| Semantic rule | v3 exact balances; natural-currency columns; unavailable conversion is a labelled card, never a dropped chart |
| Response adaptation | `{date_range, charts, trees: SerialisedTreeNode[]}`; charts validated by `chart_validator` |
| Tests | `internal/report` exact-value tests; adapter contract fixture; browser `web/tests/fava-routes.spec.ts` |
| Provenance | adopt (frontend components) + rewrite (Go tree/chart projection) |

### `journal_page` (GET) — [open Q1]

| Field | Required content |
| --- | --- |
| Fava source | `json_api.get_journal_page`; `_journal_table.html` (server HTML) |
| Route/state | journal; `page`, `order`, `account`, `filter`, `time`, `conversion`, `interval` |
| Request shape | GET; Fava returns HTML string `journal` + `total_pages` |
| Go owner | `internal/report.JournalBetween` + `internal/report.FilterJournal`; **Go must produce structured rows** (Q1) |
| Semantic rule | transaction grouping, not posting flattening; deterministic order; v3 values |
| Response adaptation | either `{page,total_pages,journal:html}` (port Jinja→Go template) or structured JSON rendered in Svelte (recommended) |
| Tests | journal grouping test; browser flow; keyboard |
| Provenance | adapt/rewrite — the HTML renderer is not adopted as-is |

### `account_report` (GET) — [open Q1, Q4]

| Field | Required content |
| --- | --- |
| Fava source | `json_api.get_account_report`; `core/__init__.account_journal` |
| Route/state | `/account/<name>/`; `r` in {journal, changes, balances}, filters, conversion, interval |
| Request shape | GET with `a`, `r`, filters |
| Go owner | `internal/report` journal scoped by account + `charts.go` accountChart; interval balances |
| Semantic rule | running balance per account; v3 exact |
| Response adaptation | `AccountReportJournal{charts,journal}` or `AccountReportTree{charts,interval_balances,budgets,dates}` |
| Tests | account running-balance test; browser |
| Provenance | adapt (frontend) + rewrite (Go balances) |

### `query` (GET)

| Field | Required content |
| --- | --- |
| Fava source | `json_api.get_query`; `core/query_shell.py` |
| Route/state | query, holdings, statistics; `query_string`, filters |
| Request shape | GET with `query_string`, `account`, `filter`, `time` |
| Go owner | `internal/query.Evaluate` (BeanQuery semantics) |
| Semantic rule | v3 result authority; preserve exact CSV values; preserve prior result on error |
| Response adaptation | `{t:"table"|"string", ...}` validated by `query_validator` |
| Tests | `internal/query` exact-value tests; query error-preservation browser test |
| Provenance | adopt (frontend) + rewrite (Go query) |

### `source` / `source` (PUT, editor)

| Field | Required content |
| --- | --- |
| Fava source | `json_api.get_source` / `put_source`; `core/file.py` |
| Route/state | editor; `file_path`, `line` |
| Request shape | GET(filename) → `SourceFile{file_path,sha256sum,source}`; PUT(file_path, source, sha256sum) → updated sha256sum |
| Go owner | `internal/web` `handleEditor`/`handleEditorSave` (atomic write, backup, revalidate, snapshot) |
| Semantic rule | explicit, atomic, backed up, revalidated; failed writes never publish |
| Response adaptation | `SourceFile` shape; editor diagnostics via `errors` |
| Tests | editor save/rollback tests; browser |
| Provenance | adopt (frontend) + rewrite (Go save path already exists) |

### `statistics` (GET)

| Field | Required content |
| --- | --- |
| Fava source | `json_api.get_statistics`; `core/accounts.all_balance_directives` |
| Route/state | statistics; filters |
| Request shape | GET with filters |
| Go owner | `internal/report.Statistics` + `internal/query` postings-per-account |
| Semantic rule | deterministic counts/totals; exact values |
| Response adaptation | `{all_balance_directives, balances:{account:Inventory}, entries_by_type}` |
| Tests | statistics deterministic test; browser accessible table fallback |
| Provenance | adopt (frontend) + rewrite (Go) |

## Provenance guard mapping

Every row's provenance column links to the corresponding `docs/fava-provenance-inventory.md`
row. The P2 build will add a CI check that fails if a `web/` or
`internal/web/assets/` file lacks a provenance row; the contract-map rows
document the data contract that each adopted frontend file depends on, so a
new frontend file cannot be added without both a contract row and a
provenance row.

## Verification

- All rows above were derived by reading the pinned Fava source (see
  `docs/fava-source-inventory.md` §1–§2 for the exact endpoint/source mapping).
- No private-ledger data was accessed; the contract facts are public upstream
  only.
- `git diff --check` passes on the OrangeCount worktree (see completion report).