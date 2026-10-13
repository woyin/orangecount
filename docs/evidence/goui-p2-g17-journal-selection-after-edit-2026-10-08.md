# G17 流水选择跨无关源编辑恢复

日期：2026-10-08。对应旧版 GUI 场景10：选择一笔唯一交易后，在源文件前插入注释；选择应跟随同一交易，重复交易不得继承该选择。

Go UI 在流水表中选择交易并在下方显示分录详情，使用核心投影提供的稳定交易 ID。原生窗口回归发现 Model 收到新 SnapshotID 后会清空该 ID，所以即使交易内容没有改变，注释插入也会让详情消失。现保留当前页和已选 ID，让新流水页按新的交易 ID 重新解析选择。核心对重复交易生成 snapshot-bound ID；重复项不会匹配旧的唯一交易选择。

新增 `TestNativeJournalSelectionSurvivesUnrelatedEditButNotDuplicates`，在隔离临时账本和实际 MyGo/AppKit 窗口中：

1. 选择唯一的 `Unique dinner` 交易，确认 `Expenses:Dining` 分录详情可见。
2. 在源文件开头插入无关注释，等待 watcher 发布新 snapshot；同一交易的分录详情仍可见。
3. 将同一交易复制为重复项，等待新 snapshot；界面回到“选择一笔交易查看分录详情”，没有把旧选择套到任一重复行。

该场景连续5次通过，并在 race 模式下通过。`make goui-check` 通过；完整 `make goui-native-e2e` 通过，当前59项常规原生窗口场景，耗时46.874秒。测试没有读取或写入仓库 fixture、用户账本或偏好文件。

这补上了流水选择/详情恢复的真实 GUI 回归证据；G17/G38 的其余完整矩阵及 G37 的真人输入法/VoiceOver 验收仍保持未完成。
