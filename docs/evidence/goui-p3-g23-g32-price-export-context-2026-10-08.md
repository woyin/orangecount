# G23/G32 价格报表导出上下文补充

日期：2026-10-08

## 问题与修复

价格历史分析文件原本已保存 `price_base` 与 `price_quote`，但 `ReportExportIdentity` 没有暴露报价对，保存面板看不出正在导出哪组价格，默认文件名也会让不同报价对落到相同名称。现在身份包含规范化的 base/quote；中英文保存面板显示报价对；默认文件名将 base/quote 加入 section 和日期范围之后，并替换路径分隔符。

## 验证

- `TestPriceReportIdentityRetainsSelectedQuotePair` 使用真实 `prices.bean` 捕获 STK/USD 报价历史，检查身份和分析文件都保留 `STK/USD`。
- `TestEnglishPresentationTranslatesApplicationLabelsAndPreservesLedgerValues` 检查中英文报价对文案，并确认 STK/USD 与 EUR/USD 默认文件名不同。
- `go test ./...`、`go vet ./...`、`make goui-check`、`git diff --check` 与 Go 格式检查通过。
- `make goui-native-e2e`：通过，原生 app 包测试耗时 62.958 秒。

这项修复不替代可见保存面板上的人工 VoiceOver/视觉验收；G23 与 G32 仍按任务卡保留人工 macOS 验收项。
