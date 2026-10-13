# G16 自身余额与折算估值显示

日期：2026-10-08（Asia/Singapore）。修复资产负债表/损益表账户树未呈现 MyGo 核心投影中直属余额和逐账户折算估值的问题。

旧 SwiftUI 页面会分别展示账户直属余额、子树汇总和折算值；Go UI 此前只呈现子树的原币金额，虽然核心 DTO 已提供直属金额、直属估值及子树估值字段。现在账户树分开显示直属余额与子树总额，并在可用时展示目标币种折算总计；报价不完整时展示已折算部分和缺少报价的币种，报价日期与精确汇率可从值的提示中查看。原币列仍使用核心投影，页面不自行重算父级汇总。

headless Session 回归覆盖 Legacy 和 V3 两种语义：验证 EUR/USD 原币分列、直属与子树金额同时呈现、2000-01-03 截止时的 140 USD 完整折算、2000-01-02 时的 100 USD 已折算部分及缺失 EUR 报价；同时检查直属余额、子树总额和固定分类总计均出现在 UI 文本树中。相同用例检查完整和部分估值的中英文文案，并验证 holdings include 在 at-cost 下为 20 USD、market-value 下为 2 STK 与 30 USD。原生 MyGo Accessibility 回归覆盖栏标题、完整和部分折算状态、缺价提示，以及 `EUR/USD 2 @ 2000-01-03` 精确报价提示。

主要实现位于 `apps/goui/internal/app/model.go` 和 `statement_amounts.go`，文案位于 `internal/i18n/translate.go`；覆盖位于 `model_test.go` 与 `native_e2e_darwin_test.go`。

验证结果：

- 针对性 headless 回归通过：
  `GOWORK=off go test ./internal/app ./internal/i18n -run '^(TestStatementPageShowsOwnSubtreeAndConvertedBalancesForBothProfiles|TestStatementPageRendersExactAccountTreeFromCore|TestStatementPageExplainsNativeMissingQuoteAndEmptyStates|TestStatementSearchFiltersRowsWithoutChangingCoreTotals|TestTextTranslatesApplicationMessagePrefixesOnly)$' -count=1`
- 针对性原生窗口回归通过：
  `./apps/goui/native-e2e.sh -run '^(TestNativeMultiCurrencyBalancesKeepCurrenciesSeparate|TestNativeStatementShowsDatedConvertedAccountValues)$' -count=1 -v`
- 扩展后的 Legacy/V3、中英文 headless 估值回归通过：
  `GOWORK=off go test ./internal/app -run '^TestStatementPageShowsOwnSubtreeAndConvertedBalancesForBothProfiles$' -count=1`
- 新增的原生 quote Help 检查单项连续 5 次通过：
  `./apps/goui/native-e2e.sh -run '^TestNativeStatementShowsDatedConvertedAccountValues$' -count=5 -v`
- 全量 `make goui-check`、`GOWORK=off go vet ./...`、`GOWORK=off go test -race ./...` 与 `make goui-native-e2e` 均通过；完整原生套件用时 76.147 秒。
- 基于当前 dirty tree 生成的 arm64/amd64 开发包均通过 `make goui-package-verify`。包仍为 unsigned/unnotarized；本轮没有从新 bundle 启动可见 GUI，G43 的最新包 GUI 流程仍未完成。

G16 继续进行中：完整 options/profile 组合、报表捕获/导出 identity 联动、VoiceOver 与真实 macOS 页面等价矩阵尚未全部验收。
