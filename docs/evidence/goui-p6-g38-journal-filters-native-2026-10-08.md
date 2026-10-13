# G38 旧场景 9 流水筛选与账户树清除验证

日期：2026-10-08

## 真实窗口结果

新增 `TestNativeJournalDateAccountSearchAndSortKeepSnapshotIdentity`，在真实 MyGo Content 窗口读取临时生成的 105 笔公开测试账本，通过原生 Accessibility 控件依次操作日期、账户、搜索和排序：

- 日期范围限定到 2026-01-02 后为 35 笔。
- 账户限定到 `Assets:Bank` 后为 17 笔。
- 搜索 `Needle` 后为 9 笔，并将排序切换为“旧到新”。
- 首尾交易 narration 为 `Needle Txn 007` 和 `Needle Txn 103`；日期、账户、搜索与排序期间 SnapshotID 不变。
- 界面可访问性树显示 9 笔匹配交易及搜索字段。最终 AX 检查前显式请求一帧重绘，避免读取旧的原生可访问性快照。

新增 `TestNativeAccountTreeClearSearchRestoresCollapsedTree`，先折叠两棵根账户，再搜索深层账户以展开匹配路径，最后通过 AXPress 清除搜索；确认搜索文本和草稿均为空、展开状态恢复为折叠、显示行数回到根节点数。

## 诊断与验证

完整原生门禁首次复现账户树测试失败。缩小后的真实窗口用例证明产品状态逻辑正确，失败来自测试点击反馈不充分：坐标点击 hook 只报告找到有效 AX 界限并发送了鼠标事件，无法确认按钮回调已执行。现新增 `pressNativeAccessibilityEventually`，检查 MyGo 原生 press 的 found、allowed、performed 三个结果；账户树回归测试和原场景均已改用该方式，没有修改产品业务逻辑。

验证结果：

- 流水筛选场景 `-count=5` 通过。
- 账户树清除搜索场景 `-count=5` 通过。
- 流水、账户树清除及账户树 reload 三场景的 race 检查通过。
- `make goui-check` 通过。
- 修正后的 `make goui-native-e2e` 全套通过，耗时 71.123 秒。

这只补齐旧场景 9 中日期、账户、文字、排序及原生窗口筛选结果的一组组合路径，并修复账户树原生测试的反馈判定；旧版 18 项完整 GUI 等价矩阵、真人 IME/VoiceOver 和打包应用完整安装验收仍由 G37/G38/G43 跟踪。
