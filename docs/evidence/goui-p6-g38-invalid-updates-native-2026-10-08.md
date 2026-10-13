# G38 有效账本连续错误与修复原生窗口证据

日期：2026-10-08。范围：旧 GUI 场景 12 的“有效账本改坏、诊断更新、修复后恢复”。

原生 MyGo 窗口从一个含一笔交易的有效临时账本开始。先写入未闭合字符串语法错误，再替换成不平衡交易错误。两次更新都保留原 `SnapshotID` 和一笔有效交易投影；诊断页从 `E-PARSE-STRING` 更新为 `E-EVAL-UNBALANCED`，不再保留前一个错误。随后写回有效内容并增加一笔交易，文件监视自动恢复到新快照；工作区路径和 epoch 不变，诊断页清空，仪表盘显示 `30 USD`。

验证结果：

- `./native-e2e.sh -run '^TestNativeRepeatedInvalidUpdatesReplaceDiagnosticsAndRecover$' -count=5 -v`：5次通过。
- `./native-e2e.sh -race -run '^TestNativeRepeatedInvalidUpdatesReplaceDiagnosticsAndRecover$' -count=1 -v`：通过。
- 所有数据只在临时目录中生成，没有读取私人账本。

此自动化结果覆盖状态、错误代码、保留的旧有效摘要及修复后的新投影。真人 VoiceOver、目标旧版 macOS 与完整18场景行为矩阵仍需独立验收。
