# G21/G22/G23 原生页面验收补充

日期：2026-10-08（Asia/Singapore）。本记录补充账户详情、持仓和商品/价格页面的真实 MyGo Content 窗口自动化证据，不替代真人 VoiceOver 或完整产品验收。

## 原生交互与结果

- `TestNativeAccountDetailShowsCoreHistoryAndReturnsToStatement` 使用公开 `account-detail.bean` 和原生 `AXRow` 聚焦选择账户，通过独立的“查看账户详情”按钮打开页面；检查关闭状态、关闭日期、FIFO、历史余额、交易/分录来源及返回后相同报表快照。此场景也覆盖原生 AX 树未呈现 OutlineTable 内嵌按钮的问题：选择账户行后显示有名称的独立辅助功能动作。
- `TestNativeHoldingsNavigatesGroupsAverageHistoryAndLots` 使用公开 `average-history.bean`，通过账户字段筛选 `Assets:Stock`，验证快照上的 `5 STK / 40 USD / 8 USD`，再经原生页面按钮进入 7 点 AVERAGE 历史和当前批次详情。
- `TestNativePricesNavigatesCommodityCatalogAndQuoteHistory` 使用公开 `prices.bean`，通过商品目录筛选验证未报价商品及声明来源，再按 `Assets:Stock` 筛选并聚焦 `STK` 行；打开原生报价币种选择器，按稳定节点 ID 点击重复标签中的 USD 选项，检查 `STK/USD` 截止报价 `99 USD`、日期 `2000-01-20` 和来源文件。

原生检查发现两个辅助功能细节并已修正：余额树现在提供明确的 `账户余额` 名称和选中行详情按钮；商品报价下拉框使用不与旁边说明文字重复的名称“选择报价币种”，中英文目录均有对应文案。仅 `darwin && orangecount_e2e` 编译的节点 ID 点击钩子用于在菜单项与表格存在同名标签时精确选择，不进入生产包。

## 当前验证

- 三个新增原生场景连续运行三次通过：

  ```sh
  ./apps/goui/native-e2e.sh -run '^TestNative(AccountDetailShowsCoreHistoryAndReturnsToStatement|HoldingsNavigatesGroupsAverageHistoryAndLots|PricesNavigatesCommodityCatalogAndQuoteHistory)$' -count=3
  ```

- 最新完整 `make goui-native-e2e` 通过。原生测试文件包含53个 `TestNative...` 入口：52项常规窗口场景和1项独立十分钟 soak。
- 最新 `make goui-check` 通过；根 `make fmt vet test race license build` 通过；Go UI `GOWORK=off go test -race -count=1 ./...` 与 `GOWORK=off CGO_ENABLED=0 go vet ./...` 通过。
- 从当时源码生成 arm64/amd64 unsigned development `.app`/ZIP；两包均通过 `verify-package.sh` 的 ZIP checksum、Mach-O 架构、macOS 13.0 minimum、build metadata、无 Swift/CGO linkage 和 module/license graph 校验。

  | 架构 | ZIP SHA-256 |
  |---|---|
  | arm64 | `3c8ddfc914e59db7af843bc85fbc365f4b559b27ed1a86db333c81af5b8eff0c` |
  | amd64 | `8de0966268e3a0274be67bc42a346468f304221d9cd11db5bac8571a76dc712d` |

## 验收边界

三个页面仍需要真人 VoiceOver/键盘检查及 G38 旧版 18 项行为矩阵中的未覆盖场景。报价选择场景使用原生下拉选项；核心数据显示、来源及截止值来自真实 `DesktopEngine` 响应。独立开发包的完整 GUI 阅读/查询/导出循环、macOS TCC 辅助功能流程、远端 CI、签名/公证和默认入口切换仍未验收。G21、G22、G23、G37、G38、G42、G43 状态不因本补充单独变更。
