# G09 原生文件监视恢复记录

日期：2026-10-08（Asia/Singapore）。本记录只补充 G09 的原生文件监视恢复证据，不表示 G09 或 P1 已完成。

## 验收场景

- 使用 `t.TempDir()` 创建合成 `main.bean`，初始账本含1笔交易；只打开一个真实 MyGo Content 窗口。
- 从外部把文件写成语法错误，等待窗口进入 stale 状态并显示当前诊断；检查 `SnapshotID` 未变化，上一份有效响应仍含1笔交易。
- 外部修复文件并追加一笔交易，全程不点击“重新读取”；等待 watcher 发布新 `SnapshotID`，并检查窗口回到 ready、dashboard 响应含2笔交易且原生 Accessibility 内容显示 `30 USD`。
- 检查修复前后的活动路径与 workspace epoch 相同，证明这是当前窗口自动恢复，不是换窗或重新打开账本。
- 另以 `main.bean` include 一个合成 `activity.bean`，通过同目录临时文件 + `rename` 原子替换 include 内容并追加一笔交易；等待 watcher 载入新 `SnapshotID` 并刷新 `30 USD`，检查窗口路径/epoch 稳定，全程不触发手动刷新。
- 通过 opt-in 原生测试钩子把路径投递到 MyGo `OpenFiles` delegate，并使用与生产入口相同的 `RouteOpenFile` 决策：首个文件进入现有欢迎窗口；窗口已有账本后，再开的文件进入新窗口；原窗口路径、epoch、snapshot 与交易数保持不变。
- 从欢迎页点击一个已失效的最近路径，验证窗口打开无快照的 `PhaseInvalid` 工作区并显示 include 读取诊断，最近项仍在列表中；再投递一个有效文件打开事件，验证路由为它新建窗口，原诊断窗口保持原路径与 epoch。

## 结果

```sh
./apps/goui/native-e2e.sh -run '^TestNativeExternalFileWatchRecoversWithoutManualReload$' -count=5
./apps/goui/native-e2e.sh -run '^TestNativeIncludedFileWatchUpdatesWithoutManualReload$' -count=5
./apps/goui/native-e2e.sh -run '^TestNativeFinderOpenUsesWelcomeWindowAndPreservesPopulatedWindow$' -count=5
./apps/goui/native-e2e.sh -run '^TestNativeMissingRecentLedgerKeepsDiagnosticsWhenOpeningAnotherLedger$' -count=5
make goui-native-e2e
make goui-check
cd apps/goui && GOWORK=off CGO_ENABLED=0 go vet ./...
cd apps/goui && GOWORK=off CGO_ENABLED=1 go test -tags orangecount_e2e ./internal/app -run '^TestNativeExternalFileWatchRecoversWithoutManualReload$' -count=1
git diff --check
```

入口文件错误→修复、include 文件原子替换、MyGo `OpenFiles` delegate 窗口路由、失效最近项诊断后保留并行打开四个原生场景分别重复5次通过；完整24项原生窗口套件通过。`make goui-check`、`go vet` 与 `git diff --check` 通过。无 opt-in 测试按预期跳过且不启动原生窗口。

## 边界

测试只通过 MyGo delegate 钩子投递 OS-open 文件路径，并覆盖应用窗口路由；没有从 Finder 实际双击文档或验证 packaged file association。它还覆盖入口文件外部错误/修复、有效 include 文件原子替换，以及失效最近路径保留诊断和另一文档开窗。其他多窗口文件监视故障矩阵尚未通过原生 GUI 验收。测试只使用合成临时账本。
