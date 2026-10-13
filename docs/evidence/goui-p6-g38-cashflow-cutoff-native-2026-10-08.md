# G38 旧场景 3–5 现金流与报表日期原生验证

日期：2026-10-08

新增 `TestNativeCashflowBalanceCutoffAndPeriodTotalsMatchFixture`，通过真实 MyGo 窗口读取公开 `cashflow.bean`，使用原生导航、日期输入和应用筛选控件验证固定期望值：

- 默认 dashboard 的收入、支出和净额分别为 `1000 USD`、`300 USD`、`700 USD`。
- 资产负债表截止 2026-01-02 时，Assets 汇总为 `1000 USD`，排除 1 月 3 日支出。
- 仅看 2026-01-03 时，损益表 Income 无 USD 余额、Expenses 为 `300 USD`；回到 dashboard 后收入为0、支出为300、当期净额为−300，而净资产仍为 `700 USD`。
- 三个页面请求始终使用打开时的 SnapshotID；Accessibility 树确认 cutoff 金额和期间精确值可见。

定向原生场景连续5次通过，race 检查通过。完整 `make goui-check` 与 `make goui-native-e2e` 均通过；本轮全量原生门禁耗时78.646秒。

这补足旧清单场景3–5中 dashboard 金额、资产负债表 cutoff 与期间损益/净资产携带检查；不代表全部旧版18项 GUI 等价矩阵完成。

```sh
./apps/goui/native-e2e.sh -run '^TestNativeCashflowBalanceCutoffAndPeriodTotalsMatchFixture$' -count=5 -v
./apps/goui/native-e2e.sh -race -run '^TestNativeCashflowBalanceCutoffAndPeriodTotalsMatchFixture$' -count=1 -v
make goui-check
make goui-native-e2e
```
