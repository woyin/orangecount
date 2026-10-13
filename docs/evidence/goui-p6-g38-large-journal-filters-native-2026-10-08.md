# G38 旧场景 9 大账本筛选与分页原生验证

日期：2026-10-08

扩展 `TestNativeJournalPagingProjectsOnlyOneHundredOf52500Transactions`，在真实 MyGo 窗口中使用 52,500 笔临时账本覆盖全量与筛选后的流水分页。合成交易按日期分为4段，每笔按固定规则选择 `Assets:Bank`/`Assets:Cash`，每10笔之一包含 `Needle`，因此结果计数可以独立核算。

- 全量模式保持 `Window=all` 且日期范围为空；总数为52,500，每页100条。第一页与第二页可见范围分别为1–100、101–200，首尾 narration 和 SnapshotID符合预期。
- 原本从第二页应用 2026-01-03 日期范围时，真实窗口请求仍带 `offset=100`。测试因此复现出新筛选没有回到第一页的问题；`applyReadingFilters` 现在会在流水页应用日期筛选时将 offset 重置为0。
- 修复后，同一账本日期筛选计数为13,125，选 `Assets:Bank` 后为3,281，搜索 `Needle` 后为656。原生 Accessibility 树也显示对应计数。
- 切换“旧到新”后，第一页 narration 从 `Needle Txn 13140` 到 `Needle Txn 15120`，第二页从 `Needle Txn 15140` 到 `Needle Txn 17120`。逐行检查日期、账户、交易编号和 posting，并跨页检查无重复 ID；所有请求仍使用同一 SnapshotID。

验证结果：该52,500笔原生场景连续5轮通过；定向 race 检查通过；`make goui-check` 通过；完整 `make goui-native-e2e` 通过（77.711秒）。此证据补齐旧场景9中大账本筛选与排序后分页的组合路径；G38 全功能等价矩阵、真人 IME/VoiceOver 与开发包完整 GUI 流程仍在进行。

```sh
./apps/goui/native-e2e.sh -run '^TestNativeJournalPagingProjectsOnlyOneHundredOf52500Transactions$' -count=5 -v
./apps/goui/native-e2e.sh -race -run '^TestNativeJournalPagingProjectsOnlyOneHundredOf52500Transactions$' -count=1 -v
make goui-check
make goui-native-e2e
```
