# G38 include 删除、恢复与替换原生窗口证据

日期：2026-10-08。范围：旧 GUI 场景 14 的 include 文件变化和恢复。

原生 MyGo 窗口从入口 `main.bean` 加载 `activity.bean`。删除 include 后，窗口显示 `E-INCLUDE-READ` 并把位置指向入口文件的 include 指令 `main.bean:2:9`；上一个有效快照仍显示一笔交易。重新创建 include 文件会自动生成新快照并显示两笔交易。之后先写入新的 `replacement.bean`，再把入口切换到新路径；窗口自动加载新 include，最终显示两笔交易及 `50 USD`。整个过程始终使用同一入口路径和 workspace epoch。

验证结果：

- `./native-e2e.sh -run '^TestNativeIncludedFileRemovalRepairAndReplacement$' -count=5 -v`：5次通过。
- `./native-e2e.sh -race -run '^TestNativeIncludedFileRemovalRepairAndReplacement$' -count=1 -v`：通过。
- 临时目录中的公开格式 fixture，无私人账本内容。

验证覆盖最终发布的最新 include 图和旧快照保护；连续编辑期间不要求每个中间状态都发布。真人 VoiceOver、旧版 macOS 和完整18场景行为矩阵仍需独立验收。
