# G38 旧场景 8 成本、价格与分录来源原生验证

日期：2026-10-08

新增 `TestNativeJournalPostingCostsPricesAndSourceActionsKeepLocations`，在真实 MyGo 流水窗口分别打开公开 `cost-basis.bean` 和 `at-price.bean` fixture，通过原生 Accessibility 选择交易并检查详情：

- `Assets:Shares` 显示 `10 SH`、精确成本 `{2 USD}`，来源为 `cost-basis.bean:5:3`。
- `Assets:Cash` 显示 `100 USD`、单位价格 `@ 0.9 EUR`，来源为 `at-price.bean:6:3`。
- `Assets:Bank` 显示 `200 USD`、总价 `@@ 180 EUR`，来源为 `at-price.bean:9:3`。
- 三种分录的 `Journal` 投影均核对账户、单位、成本/价格类型及完整来源路径、行列；原生 Accessibility 树另核对可见详情和来源位置。
- Finder 来源按钮现在将动作名称与对应 `path:line:column` 一起作为 Accessibility 标签，避免多条分录都暴露同一个“显示”名称。测试激活该按钮并通过隔离回调确认接收到的路径与所选分录来源完全一致，不会实际打开 Finder。

验证结果：该三场景组合连续5轮通过；定向 race 检查通过；`make goui-check` 通过；完整 `make goui-native-e2e` 通过（77.688秒）。

这补充旧场景 8 对成本、`@`/`@@` 价格和来源位置的真实窗口核对，不代表 G38 全功能 GUI 等价矩阵完成；真人 IME/VoiceOver、其余旧版行为矩阵与打包应用全流程仍待验收。

```sh
./apps/goui/native-e2e.sh -run '^TestNativeJournalPostingCostsPricesAndSourceActionsKeepLocations$' -count=5 -v
./apps/goui/native-e2e.sh -race -run '^TestNativeJournalPostingCostsPricesAndSourceActionsKeepLocations$' -count=1 -v
make goui-check
make goui-native-e2e
```
