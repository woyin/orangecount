# 路由门禁证据矩阵（2026-10-27）

本文件是 `docs/fava-route-state-manifest.md` 四层证据的逐行投影，用于把「缺什么」变成后续工单。
数据源只有 manifest 的 `Evidence (four-layer gate)` 列；判定入口是 `go run ./tools/favagates -gate 1`。

登记规则：

- `contract:` / `visual:` / `hardening:` 的路径必须存在于磁盘；Markdown 片段必须是真实标题锚点。
- `browser:` 若指向 `.spec.mjs`，片段必须是对应文件里的 `test(...)` 标题。
- `hardening:` 可写 Go 测试函数名，必须在 `*_test.go` 里存在。
- `pending` 表示证据确实不存在。它不会让门禁变绿，只记录缺口。

## 覆盖率

| 层 | 有证据 | 总数 | 覆盖率 |
| --- | --- | --- | --- |
| contract | 37 | 37 | 37/37 |
| browser | 19 | 37 | 19/37 |
| visual | 0 | 37 | 0/37 |
| hardening | 30 | 37 | 30/37 |
| 四层齐备 | 0 | 37 | 0/37 |

## 逐行证据

| ID | 状态 | contract | browser | visual | hardening |
| --- | --- | --- | --- | --- | --- |
| R-ROOT | legacy/prototype | 有 — `docs/fava-contract-map.md#ledger_data-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#root+and+core+reports+load+real+adapter+data` | 无（pending） | 有 — `TestServerQueryReportsAndEmbeddedUI` |
| R-IS | legacy/prototype | 有 — `docs/fava-contract-map.md#income_statement-balance_sheet-trial_balance-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#root+and+core+reports+load+real+adapter+data` | 无（pending） | 有 — `TestIncomeStatementChartsArePerCurrency` |
| R-BS | legacy | 有 — `docs/fava-contract-map.md#income_statement-balance_sheet-trial_balance-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#root+and+core+reports+load+real+adapter+data` | 无（pending） | 有 — `TestBalanceSheetBuildsExplicitAncestorTree` |
| R-TB | legacy | 有 — `docs/fava-contract-map.md#income_statement-balance_sheet-trial_balance-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#root+and+core+reports+load+real+adapter+data` | 无（pending） | 有 — `TestTrialBalanceTreeIncludesOnlyRealAncestors` |
| R-JOURNAL | legacy | 有 — `docs/fava-contract-map.md#journal_page-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#journal+groups+postings+and+query+executes` | 无（pending） | 有 — `TestProjectJournalHonorsFQLFilters` |
| R-ACCOUNT | legacy | 有 — `docs/fava-contract-map.md#account_report-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#account+detail+renders+sparse+multi-series+charts` | 无（pending） | 有 — `TestAccountIntervalsChangesAndBalancesCarryQuietMonths` |
| R-HOLD-ACCOUNT | legacy | 有 — `docs/fava-contract-map.md#query-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows` | 无（pending） | 有 — `TestHoldingsAggregateGroupsAndSumsWithinCostCurrency` |
| R-HOLD-CURRENCY | legacy | 有 — `docs/fava-contract-map.md#query-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows` | 无（pending） | 有 — `TestHoldingsAggregateGroupsAndSumsWithinCostCurrency` |
| R-HOLD-ROOT | legacy | 有 — `docs/fava-contract-map.md#query-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows` | 无（pending） | 有 — `TestHoldingsAggregateGroupsAndSumsWithinCostCurrency` |
| R-HOLD-COMMODITY | legacy | 有 — `docs/fava-contract-map.md#query-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows` | 无（pending） | 有 — `TestHoldingsAggregateGroupsAndSumsWithinCostCurrency` |
| R-COMMODITIES | legacy | 有 — `docs/fava-contract-map.md#query-get` | 无（pending） | 无（pending） | 有 — `TestCommoditiesListsDeclaredHeldCommoditiesWithoutPriceQuotes` |
| R-DOCUMENTS | legacy | 有 — `docs/fava-contract-map.md` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows` | 无（pending） | 有 — `TestDocumentUploadAndMoveLifecycle` |
| R-EVENTS | legacy | 有 — `docs/fava-contract-map.md` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows` | 无（pending） | 有 — `TestReportSetIncludesEveryCoreProjectionAndSafeGraphPaths` |
| R-STATISTICS | legacy | 有 — `docs/fava-contract-map.md#statistics-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#data-backed+utility+reports+render+their+adapter+rows` | 无（pending） | 有 — `TestStatisticsHasStableDirectiveCounts` |
| R-ERRORS | legacy | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 有 — `TestFavaDiagnosticsAndImportAdapters` |
| R-HELP | legacy | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 有 — `TestFavaAdapterHelpAndDiagnostics` |
| R-QUERY | legacy | 有 — `docs/fava-contract-map.md#query-get` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#journal+groups+postings+and+query+executes` | 无（pending） | 有 — `TestStandardFormsMatchBeanQueryOracle` |
| R-EDITOR | legacy | 有 — `docs/fava-contract-map.md#source-source-put-editor` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#editor+and+import+expose+reviewed+workflows` | 无（pending） | 有 — `TestEditorWriteWorkflowIsAtomicAndRevalidated` |
| R-IMPORT | legacy | 有 — `docs/fava-contract-map.md` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#editor+and+import+expose+reviewed+workflows` | 无（pending） | 有 — `TestFavaDiagnosticsAndImportAdapters` |
| R-OPTIONS | legacy | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 有 — `TestOptionsEndpointGetPostAndValidation` |
| G-SHELL | prototype | 有 — `docs/fava-ux-spec.md#shared-controls` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#narrow+menu+opens+the+transplanted+aside` | 无（pending） | 有 — `TestTransplantedUIAssetsAreTheOnlyServedSurface` |
| G-FILTERS | legacy/prototype | 有 — `docs/fava-ux-spec.md#shared-controls` | 有 — `web/tests/visual/fava-transplanted.spec.mjs#journal+groups+postings+and+query+executes` | 无（pending） | 有 — `TestReportFilterParsingAndOptionValidationCoverSupportedVariants` |
| G-THEME | prototype | 有 — `docs/fava-ux-spec.md#strict-visual-matrix` | 有 — `web/tests/unit/state.test.mjs#theme+and+locale+stores+accept+only+supported+values` | 无（pending） | 无（pending） |
| G-LOCALE | legacy/prototype | 有 — `docs/fava-ux-spec.md#strict-visual-matrix` | 无（pending） | 无（pending） | 有 — `TestTransplantedUIAssetsAreTheOnlyServedSurface` |
| G-KEYBOARD | planned | 有 — `docs/fava-ux-spec.md#keyboard-table` | 无（pending） | 无（pending） | 无（pending） |
| M-ADD | planned | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 有 — `TestFavaAddEntriesRoundTrip` |
| M-CONTEXT | planned | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 有 — `TestFavaAdapterDownloadJournalAndEntryContext` |
| M-EXPORT | planned | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 有 — `TestFavaAdapterDownloadJournalAndEntryContext` |
| M-DOCUMENT | planned | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 有 — `TestFavaAdapterDocumentMove` |
| M-NOTIFY | prototype/legacy | 有 — `docs/fava-ux-spec.md#shared-controls` | 无（pending） | 无（pending） | 无（pending） |
| D-STATEMENT | planned | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 无（pending） |
| D-DOCUMENT | legacy | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 有 — `TestFavaAdapterDocumentUpload` |
| D-JOURNAL | planned | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 有 — `TestFavaAdapterDownloadJournalAndEntryContext` |
| D-QUERY-CSV | legacy | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 有 — `TestTypedCSVPreservesOrdinalHeadersAndExactNumbers` |
| D-QUERY-XLSX | planned | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 无（pending） |
| D-QUERY-ODS | planned | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 无（pending） |
| D-PRINT | planned | 有 — `docs/fava-contract-map.md` | 无（pending） | 无（pending） | 无（pending） |

## 当前缺口工单

- visual 全缺（37 条）：`testdata/visual-baselines/` 现有 48 张 PNG 全部位于 `visual/fava-transplanted.spec.mjs/` 下，是 OrangeCount 自身截图；该目录按 `docs/fava-visual-harness.md` 应存 Fava 参考基线，自截图不能当作视觉权威。因此 37 条 visual 仍为 pending，直到 Fava 参考捕获入库并带 `environment-lock.json`。
- 基线来源现状（2026-10-27 核对）：`testdata/visual-baselines/` 有 48 张 PNG（12 路由 × 4 格），全部由 `fava-transplanted.spec.mjs` 生成；`testdata/visual-candidates/fava-reference/screenshots/` 只有 20 张 Fava 参考候选（5 路由 × 4 格），且 `npm --prefix web run check:reference-output` 报 `fixture content hash does not match the captured lock`。基线根部没有 `environment-lock.json`。
- browser 缺 18 条：`R-COMMODITIES`, `R-ERRORS`, `R-HELP`, `R-OPTIONS`, `G-LOCALE`, `G-KEYBOARD`, `M-ADD`, `M-CONTEXT`, `M-EXPORT`, `M-DOCUMENT`, `M-NOTIFY`, `D-STATEMENT`, `D-DOCUMENT`, `D-JOURNAL`, `D-QUERY-CSV`, `D-QUERY-XLSX`, `D-QUERY-ODS`, `D-PRINT`。这些路由/弹窗没有针对移植 shell 的 Playwright 用例。
- hardening 缺 7 条：`G-THEME`, `G-KEYBOARD`, `M-NOTIFY`, `D-STATEMENT`, `D-QUERY-XLSX`, `D-QUERY-ODS`, `D-PRINT`。
- contract 全部 37 条已登记真实路径与锚点。

真实功能缺口（不是文档缺口，需实现后才能登记）：

- `/commodities` 未挂载 `CommoditiesReport`，价格折线图与 tooltip 不渲染（W10）。
- statement 下载入口不存在（`server.go` 只有 `/documents/`）。
- Query XLSX / ODS 导出在后端不存在（只有 CSV 与 typed CSV）。
- 打印（print）只有 CSS `@media print`，无独立打印组合与浏览器用例。
- 全局 `g …` 键盘导航无实现、无用例。
- Add Entry / Context / Export / Document / notification 五个弹窗无 Playwright 用例。

状态列全部不是 `accepted`，因此 `favagates -gate 1` 仍整体失败。这份矩阵不改变任何状态。
