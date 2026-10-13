# G21/G22 报表导出上下文补充

日期：2026-10-08

## 问题与修复

账户详情和持仓投影在请求未指定估值时会实际采用 `at-cost`，但报表导出身份与分析文件仍保存空的 `valuation`。导出上下文现在读取账户/持仓投影的有效估值；持仓币种也读取投影规范化结果，使 `native` 导出为空币种。明确选择 USD 市价估值仍保存为 `USD` / `market-value`。

## 回归验证

新增 `TestAccountAndHoldingsReportAnalysisUseEffectiveCurrencyAndValuation`，分别捕获真实 `account-detail.bean` 与 `average-history.bean` 投影，核对 `ReportExportIdentity` 和分析文件 `NormalizedRequest`。测试先复现账户与持仓默认估值为空；修复后原生币种默认成本及 USD 市价两种持仓场景均通过。

- `go test ./internal/desktop -run '^(TestDashboardReportAnalysisNormalizesNativeCurrencyAndDefaultValuation|TestAccountAndHoldingsReportAnalysisUseEffectiveCurrencyAndValuation)$' -count=1`
- `go test ./internal/desktop -count=1`
- `make goui-check`
- `make goui-native-e2e`：通过，原生 app 包测试耗时 62.886 秒。

本记录只覆盖导出上下文缺陷；账户/持仓页面的真人 VoiceOver 与完整 GUI 验收仍未完成。
