<!--
Copyright 2026 OrangeCount contributors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
-->

# Fava 1.30.12 → OrangeCount contract map

Planning contract map. For every frontend request, a row records the Fava
source, the frontend call site, the request shape, the planned Go owner, the
semantic rule, response adaptation, tests, and provenance. This is the
self-check that "a reviewer can trace every in-scope page from a Fava source
module through its data contract to a planned OrangeCount owner" (ADR-0034).

References: `docs/fava-route-state-manifest.md` for canonical user-visible
coverage; `docs/fava-source-inventory.md` for source decisions;
`docs/fava-reference-lock.md` for pinning; and
`docs/fava-provenance-inventory.md` for attribution.

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
- Former open questions were resolved by ADR-0037, ADR-0038, and the accepted
  rendering-fidelity plan. Rows must name a concrete owner or explicitly
  excluded capability.
- `adopt` and `adapt` are target decisions, not proof that source is currently
  present. Current adoption requires a matching provenance row, notice,
  upstream hash, and route-gate evidence.

Wave 1 implementation note: the private `ledger_data`, `metadata`, and
`income_statement`/`balance_sheet`/`trial_balance` adapter paths now emit
snapshot envelopes. The staged frontend consumes the tree-report projection;
embedded cutover remains gated on the four-layer route evidence and approved
visual baselines.

## Route table (page → Fava route → adapter endpoint → Go owner)

| Page (frontend route) | Fava source module | Frontend call site | Adapter endpoint (Go) | Go owner | Provenance |
| --- | --- | --- | --- | --- | --- |
| Shell/bootstrap | `internal_api.get_ledger_data`, `_layout.html` `#ledger-data` | `get_ledger_data()` in `app.ts` | `ledger_data` | `internal/web` bootstrap DTO over `internal/snapshot` (`Store.Current().Evaluation()`) + `internal/report` | rewrite |
| Income Statement | `json_api.get_income_statement` | `get_income_statement` (tree_reports/index.ts) | `income_statement` | `internal/report.IncomeStatement` + `internal/report/charts.go` | adopt |
| Balance Sheet | `json_api.get_balance_sheet` | `get_balance_sheet` | `balance_sheet` | `internal/report.BalanceSheet` + charts | adopt |
| Trial Balance | `json_api.get_trial_balance` | `get_trial_balance` | `trial_balance` | `internal/report.TrialBalanceTree` + charts | adopt |
| Journal | `json_api.get_journal_page` + `_journal_table.html` | `get_journal_page` (journal/index.ts) | `journal_page` | `internal/report` transaction projection + complete FQL/time filtering + strictly escaped Go Fava-compatible HTML renderer (ADR-0037) | adapt |
| Account report | `json_api.get_account_report` | `get_account_report` (accounts/index.ts) | `account_report` | `internal/report.JournalBetween` scoped to account + `internal/report/charts.go` accountChart | adapt |
| Query | `json_api.get_query` + `core/query_shell.py` | `get_query` (query/index.ts, Query.svelte) | `query` | `internal/query.Evaluate` (BeanQuery semantics) | adopt/partial（查询图表已落地：恰两列 date+金额结果值嗅探后复用 LineChart + ▼/◀ 开关，`78d4233`；限制：str+Inventory 层级分支不可行（无 dtype）、BQL 编辑器仍为裸 textarea 依赖 H1） |
| Holdings | `json_api.get_query` with 4 predefined queries | `get_query` (holdings/index.ts) | `query` (holding query strings) | `internal/query` + `internal/report.Holdings*`; queries re-expressed; all four upstream aggregations incl. by_cost_currency covered (`2b8d370`) | adopt |
| Commodities | `json_api.get_commodities` | `get_commodities` (commodities/index.ts) | `commodities` | `internal/report.Prices` + price-map pairs | adopt |
| Documents | `json_api.get_documents` | `get_documents` (documents/index.ts) | `documents` | `internal/report.Documents` + `internal/source.DocumentRoots` containment; preview pane (`92ccb40`) + account-tree sidebar (`30ca6f3`) + move/rename dialog and private `move-document` route (`38f2618`) | adapt |
| Events | `json_api.get_events` | `get_events` (events/index.ts) | `events` | `internal/report.Events` | adopt |
| Statistics | `json_api.get_statistics` | `get_statistics` (statistics/index.ts) | `statistics` | `internal/report.Statistics` + `internal/query` for postings-per-account | adopt |
| Errors | `json_api.get_errors` | `get_errors` (app.ts, errors/index.ts) | `errors` | `internal/report.ErrorsWithGraph` + `internal/diagnostic` | adopt |
| Editor | `json_api.get_source` / `put_source` | `get_source`, `put_source` (editor/index.ts, Editor.svelte) | `source` / `source` (PUT) | `internal/web` `handleEditor` + `handleEditorSave` (atomic write, snapshot reload) | adopt |
| Import | `json_api.get_imports` / `get_extract` / `put_add_entries` | `get_imports`, `get_extract`, `save_entries` (import/index.ts, Import.svelte) | `imports` / `extract` / `add_entries` | `internal/web` `handleImport*` (native CSV/bean adapters; Python importers **excluded**) | adapt |
| Options | `json_api.get_options` | `get_options` (options/index.ts) | `options` | `internal/web` `handleOptions` for every applicable built-in Fava option; excluded capabilities are explicit deviations | adapt |
| Help | `application.help_page` | n/a (backend) | `help` | `internal/web` `handleHelp` + uploaded help content | adapt |
| Document attachment | `application.document` | `DocumentPreview`/`Table` | `document` (GET `/documents/`) | `internal/web` `handleDocument` + `internal/source.DocumentRoots` | adapt |
| Statement | `application.statement` | journal metadata links | `statement` | `internal/web` entry-hash→metadata path resolution with Document Root containment | adapt with security deviation if Fava is broader |
| Journal export | `application.download_journal` | `Export` modal | `download-journal` | deterministic Beancount rendering of filtered immutable-snapshot entries | rewrite |
| Query export | `application.download_query` | `QueryLinks` | `download-query.query_result.csv/.xlsx/.ods` | exact Go-native CSV/XLSX/ODS exporters over `internal/query.Result` | rewrite |
| `.jump` / sidebar links | `application.jump` | `SidebarLink` remote links | frontend URL handling | `internal/web` filter syncing | rewrite |

## Complete frontend request registry

Every request below requires a detailed fixture before its owning route can
pass Gate 1. `planned` means the decision and owner are closed but the contract
fixture has not yet been implemented.

| Request | Fava source / frontend call site | Method and request shape | Frontend-required response | Go owner | Required tests | Provenance/status |
| --- | --- | --- | --- | --- | --- | --- |
| `changed` | `json_api.changed`; `app.ts` poll | GET, none | wrapped `bool` plus `mtime` | `internal/snapshot` via adapter | mtime/change contract + reload browser flow | rewrite/planned |
| `errors` | `json_api.get_errors`; app/errors store | GET, none | `[{type,message,source|null}]` | `internal/diagnostic`, `internal/report` | diagnostic/source contract + conditional-nav flow | adapt/planned |
| `ledger_data` | `internal_api.get_ledger_data`; bootstrap | GET, none | validator-complete `LedgerData` | adapter over snapshot/report/ledger | full fixture validator + bootstrap flow | rewrite/partial shell slice |
| `payee_accounts` | `json_api.get_payee_accounts`; transaction form | GET `payee` | `string[]` | report attribute ranking | exact ordering + autocomplete flow | adapt/planned |
| `payee_transaction` | `json_api.get_payee_transaction`; transaction form | GET `payee` | serialized Transaction or `null` | report transaction projection | fixture match + form-fill flow | adapt/planned |
| `narration_transaction` | `json_api.get_narration_transaction`; transaction form | GET `narration` | serialized Transaction or `null` | report transaction projection | fixture match + form-fill flow | adapt/planned |
| `narrations` | `json_api.get_narrations`; transaction form | GET, none | ranked `string[]` | report attributes | ordering + autocomplete flow | adapt/planned |
| `query` | `json_api.get_query`; Query/Holdings/Statistics | GET `query_string,account,filter,time` | table/string/error union plus `mtime` | `internal/query` + adapter | dtype/exact-value/error contract + prior-result flow | adopt frontend/rewrite backend |
| `context` | `json_api.get_context`; Context modal | GET `entry_hash` | entry plus balances before/after | report/context projection | hash lookup + modal flow | adapt/partial（OC 私有路由 `entry-context` 合并 context+source slice 只读投影；balances before/after 未实现） |
| `source_slice` read | `json_api.get_source_slice`; SliceEditor | GET `entry_hash` | `{slice,sha256sum}` | source service | hash/source contract + context flow | adapt/partial（读路径并入 `entry-context`；SliceEditor 可编辑形态仍待 H1） |
| `source_slice` write/delete | `put/delete_source_slice`; SliceEditor | PUT/DELETE `entry_hash,source,sha256sum` | updated hash/status string | Reviewed write workflow | conflict/invalid/write/reload/rollback tests | adapt/planned |
| `source` read | `json_api.get_source`; Editor | GET `filename` | `SourceFile{file_path,sha256sum,source}` | source/editor service | containment/hash contract + editor load | adopt frontend/rewrite backend |
| `source` write | `json_api.put_source`; Editor | PUT `file_path,source,sha256sum` | updated hash | Reviewed write workflow | validation/conflict/atomicity/rollback/snapshot tests | adopt frontend/rewrite backend |
| `format_source` | `json_api.put_format_source`; Editor | PUT `source` | formatted source string | Go formatter boundary | deterministic formatting + keyboard flow | adapt/planned |
| `journal` | `json_api.get_journal`; legacy consumers | GET filters | serialized entries | report transaction projection | directive coverage + ordering | adapt/partial（全指令类型 + 条目/过账元数据投影已落地，`4bafc76`；分页与 balance diff_amount 未实现） |
| `journal_page` | `json_api.get_journal_page`; Journal | GET `page,order,account,filter,time,conversion,interval` | `{page,total_pages,journal:html}` | report + FQL/time + strict Go HTML template | escaping/grouping/paging contract + browser/visual flow | adapt/planned (ADR-0037) |
| `income_statement` | tree report loader | GET filters/conversion/interval | tree-report DTO | `internal/report` + charts | exact projection + route flow | adopt frontend/rewrite backend |
| `balance_sheet` | tree report loader | GET filters/conversion/interval | tree-report DTO | `internal/report` + charts | exact projection + route flow | adopt frontend/rewrite backend |
| `trial_balance` | tree report loader | GET filters/conversion/interval | tree-report DTO | `internal/report` + charts | exact projection + route flow | adopt frontend/rewrite backend |
| `account_report` | account loader | GET `a,r,filters,conversion,interval` | Journal HTML or tree/chart/budget DTO | report/account/budget + strict HTML template | running balance/details/budget contract + route flow | adapt/partial（`r=changes|balances` 区间表已以 OC table-report（AccountIntervals）落地，`24ef4ca`；上游 interval_balances 子树/budget DTO 未实现） |
| `events` | `json_api.get_events`; Events | GET filters | serialized Events | `internal/report` | ordering/filter/source contract + route flow | adopt/partial（`report.Events` + 按类型分组表格已接线并冒烟验证；散点图以无 d3 依赖 SVG 落地，`4e88925`） |
| `statistics` | `json_api.get_statistics`; Statistics | GET filters | balances/directives/entry counts | report/query + favaadapter.UpdateActivity | deterministic metric contract + route flow | adopt/partial（复合载荷 entries_by_type + postings_per_account + update_activity 三区块已落地并冒烟，`a66cf9d`；上游 all_balance_directives/balances 库存与 uptodate_status 指示未实现） |
| `commodities` | `json_api.get_commodities`; Commodities | GET filters | base/quote price series | report price projection | precision/history/unavailable contract + route flow | adopt/partial（base/quote 分组价格表 + 每商品对折线图与切换器已落地并冒烟，`d9b77d3`；line/area 模式切换与跨导航图表选择持久化未实现） |
| `options` | `json_api.get_options`; Options | GET, none | Fava and Beancount option maps | ledger options + adapter | every applicable option and precedence + route flow | adapt/planned |
| `imports` | `json_api.get_imports`; Import | GET, none | native file/importer candidates | native import service | candidate/status contract + route flow | adapt/planned |
| `extract` | `json_api.get_extract`; Import | GET `filename,importer` | serialized candidate entries | native import adapters | valid/invalid/unsupported-importer contract | adapt/planned |
| `upload_import_file` | `json_api.put_upload_import_file`; Import | PUT multipart | local candidate identifier/status | native import service | containment/type/size/error + browser flow | adapt/planned |
| `add_entries` | `json_api.put_add_entries`; AddEntry/Import | PUT serialized entries | status/new hash | Reviewed write workflow | validation/atomicity/rollback/snapshot + modal flow | adapt/partial（OC 私有路由 `add-entries` 已接线：严格序列化校验 + 原子写入/备份/重新验证，AddEntryModal 走该路由；Import 流程的 put_add_entries 仍未实现） |
| `documents` | `json_api.get_documents`; Documents | GET filters | serialized Documents | report + Document Roots | grouping/filter/unsafe-state contract + route flow | adapt/partial（行选中 + `/documents/` 内联预览已落地，`92ccb40`；账户树侧栏与 move/rename 写路径未实现） |
| `document` download | `application.document`; preview/table | GET normalized filename | contained file/download response | web/source roots | containment/missing/content headers + browser flow | rewrite/planned |
| `document` delete | `json_api.delete_document`; Documents | DELETE `filename` | status | Reviewed document workflow | containment/confirm/partial failure + flow | adapt/planned |
| `add_document` | `json_api.put_add_document`; DocumentUpload | PUT multipart | status/path identifier | Reviewed document workflow | containment/type/partial failure + flow | adapt/partial（OC 私有 POST `document` 路由已接线：同源校验 + 账户/目录校验 + basename 净化 + 拒绝覆盖，DocumentUploadModal 走该路由，`339b63e`；entry hash 附件元数据与 uri-list 链接拖放未实现） |
| `attach_document` | `json_api.put_attach_document`; Context | PUT `filename,entry_hash` | status/new hash | document + Reviewed write workflow | cross-file rollback/snapshot + flow | adapt/planned |
| `move` | `json_api.put_move`; Documents | PUT `account,new_name,filename` | status | document + Reviewed write workflow | containment/collision/rollback + flow | adapt/planned |
| `statement` | `application.statement`; Journal metadata | GET `entry_hash,key` | contained file/download or safe error | context metadata + source roots | hash/key/containment/no-path-leak + flow | rewrite/planned |
| `download-journal` | `application.download_journal`; Export modal | GET active filters | deterministic Beancount file | report filter + ledger renderer | exact order/content + download flow | rewrite/planned |
| `download-query` | `application.download_query`; QueryLinks | GET query/format | CSV/XLSX/ODS file | query result exporters | exact values/deterministic artifact + downloads | rewrite/planned |
| `help` | `application.help_page`; Help route | GET `slug` | localized help document | web help catalog | known/unknown/search + route flow | adapt/planned |

## Shared / cross-cutting contracts

### `ledger_data` bootstrap (golden-slice prerequisite)

| Field | Fava source | Go owner | Notes |
| --- | --- | --- | --- |
| `accounts` | `attributes.accounts` | `internal/report` account set | ranked string[] |
| `account_details` | `core/accounts.py` | `internal/report` close date, last entry and up-to-date projection | `AccountData` serialization |
| `base_url` | `url_for("index")` | `internal/web` | route base for the embedded app |
| `currencies` / `currency_names` / `precisions` | `attributes.currencies`, `core/commodities.py` | v3-compatible Commodity directives and exact display rules | deterministic names and precisions |
| `errors` | `get_errors()` | `internal/diagnostic` → `errors` | `[{type,message,source}]` |
| `fava_options` | `fava_options` dataclass | `internal/web` all applicable built-in options | excluded capability fields remain explicit approved deviations |
| `options` | `_get_options()` | `internal/ledger` `Evaluation.Options` | title, filename, include, operating_currency, name_* |
| `payees`/`tags`/`links`/`years`/`narrations` | `attributes.*` | `internal/report`/`internal/query` | string[] |
| `user_queries` | `all_entries_by_type.Query` | `internal/ledger` Query directives | `[{name, query_string}]` |
| `sidebar_links` | `core/misc.py` | `internal/web` | custom `fava-sidebar-link` subset |
| `upcoming_events_count` | `core/misc.py` | `internal/report` Events | int |
| `extensions` / `other_ledgers` / `incognito` / `have_excel` | extensions / multi-ledger / incognito / excel | `internal/web` — emit empty/empty/false/true once XLSX/ODS are available | excluded surfaces remain empty; export capability is truthful |
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

## Detailed high-impact contract rows

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
| Provenance | adopt (frontend components) + rewrite (Go tree/chart projection); Wave 1 staged slice implemented |

### `journal_page` (GET)

| Field | Required content |
| --- | --- |
| Fava source | `json_api.get_journal_page`; `_journal_table.html` (server HTML) |
| Route/state | journal; `page`, `order`, `account`, `filter`, `time`, `conversion`, `interval` |
| Request shape | GET; Fava returns HTML string `journal` + `total_pages` |
| Go owner | typed `internal/report` transaction projection plus complete FQL/time execution and a strictly escaped presentation template |
| Semantic rule | transaction grouping, not posting flattening; deterministic order; v3 exact values; HTML has no semantic ownership |
| Response adaptation | `{page,total_pages,journal:html}` with Fava-compatible markup (ADR-0037) |
| Tests | journal grouping test; browser flow; keyboard |
| Provenance | adapt/rewrite — the HTML renderer is not adopted as-is |

### `account_report` (GET)

| Field | Required content |
| --- | --- |
| Fava source | `json_api.get_account_report`; `core/__init__.account_journal` |
| Route/state | `/account/<name>/`; `r` in {journal, changes, balances}, filters, conversion, interval |
| Request shape | GET with `a`, `r`, filters |
| Go owner | `internal/report` journal scoped by account + `charts.go` accountChart; interval balances（`account_intervals.go` AccountIntervals 已实现 changes/balances 区间表） |
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
| Go owner | `internal/report.Statistics` + `internal/query` postings-per-account + `favaadapter.UpdateActivity` |
| Semantic rule | deterministic counts/totals; exact values; update activity in journal order (last entry per Assets/Liabilities account wins) |
| Response adaptation | composite `{entries_by_type, postings_per_account, update_activity:[{account, last_entry_date, entry_hash, balances}]}`; upstream `all_balance_directives`/`balances` inventory omitted (balances are ledger balances) |
| Tests | statistics deterministic test; UpdateActivity unit test; browser accessible table fallback |
| Provenance | adopt (frontend) + rewrite (Go) |

## Provenance guard mapping

Every row's provenance column links to the corresponding `docs/fava-provenance-inventory.md`
row. Prerequisite Phase 0 adds a check that fails if a selected or Fava-influenced `web/` or
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