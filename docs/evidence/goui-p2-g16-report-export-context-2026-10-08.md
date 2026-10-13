# G16 报表导出上下文验证

日期：2026-10-08

## 问题

账本分析文件将树报表的 `valuation` 口径写成估值状态 `native` 或 `converted`，丢失了用户选择的 `at-cost` 或 `market-value`。报表导出身份和保存面板也没有显示目标币种/估值口径。

## 修复

- 树报表分析文件现在记录有效请求口径；未指定时明确规范化为 `at-cost`。
- `ReportExportIdentity` 暴露币种与估值字段，并纳入分析导出上下文。
- 保存面板按当前投影显示报表口径，例如“原币 / 按成本”或“USD / 按市价”；英文界面显示对应译文。
- 投影身份仍由快照与规范化请求生成，因此币种/估值不同的投影不会共用 identity。

## 验证

新增核心回归覆盖 Legacy/V3 两种 profile 下的原币默认成本和 USD 市价报表，检查身份字段、分析文件 normalized request、profile 与各投影 ID。另有 workspace→capture→Save Panel 的跨层回归，使用真实 `valuation.bean` 读取 USD 市价报表并确认最终面板仍显示 `USD / 按市价`。测试先复现旧行为：默认成本导出为 `native`，市价导出为 `converted`；修复后通过。Go UI 中英文保存面板文案测试通过。

- `go test ./internal/desktop -count=1`
- `GOWORK=off CGO_ENABLED=0 go test ./internal/app ./internal/preferences ./internal/i18n ./internal/workspace -count=1`
- `make goui-check`
- `make goui-native-e2e`：通过，dashboard、账户和持仓导出上下文修复后的原生 app 包测试耗时 62.886 秒。

G16 仍需完成更广的 options/profile 与完整 macOS GUI 验收；本记录只关闭报表导出身份中丢失估值口径的问题。
