# G38 初次打开无效账本的原生窗口证据

日期：2026-10-08。范围：旧 GUI 场景 13“初次打开语法错误账本”。

测试先在一个 MyGo 窗口读取公开 `cashflow.bean`，再在新窗口首次打开公开 `err-syntax.bean` 的临时副本。新窗口进入 `PhaseInvalid`，没有有效快照或 `SnapshotID`；诊断页显示 `E-PARSE-STRING`、“字符串未结束”和 `invalid.bean:1:40`。新窗口没有显示原窗口的 `700 USD`，原窗口的路径、epoch、snapshot 和两笔交易均保持不变。

验证结果：

- `./apps/goui/native-e2e.sh -run '^TestNativeInitialInvalidLedgerShowsDiagnosticsWithoutPreviousSnapshot$' -count=5 -v`：5次通过。
- `./apps/goui/native-e2e.sh -race -run '^TestNativeInitialInvalidLedgerShowsDiagnosticsWithoutPreviousSnapshot$' -count=1 -v`：通过。
- `make goui-check`：通过。
- `make goui-native-e2e`：通过，61个原生用例通过，3个条件门控用例跳过。
- 所有错误账本内容只写入临时目录；输入来自仓库公开 fixture。

这验证了首次无效打开、精确诊断位置及窗口间快照隔离。真人 VoiceOver、真人输入法及完整18场景等价矩阵仍未验收。
