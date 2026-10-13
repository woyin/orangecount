# G38 双账本原生窗口隔离与重复关闭验证

日期：2026-10-08（Asia/Singapore）。验证两个真实 MyGo 窗口各自打开不同账本、应用不同报表账户筛选，并在一个窗口关闭和反复重开后保持另一个窗口稳定。

## 场景与结果

- 第一窗口读取 `native/account-detail.bean` 临时副本，第二窗口读取 `native/cashflow.bean` 临时副本；两者发布不同的 SnapshotID。
- 两个窗口分别进入资产负债表。第一窗口筛选 `Assets:Test:Child`，第二窗口筛选 `Assets:Bank`。模型中的筛选值、原生 Accessibility 树的账户行都保持窗口私有，没有显示另一账本的行。
- 关闭第一窗口后，第二窗口仍处于 ready，保留原路径、snapshot、资产负债表页面、`assets:bank` 筛选和 Accessibility 账户行。
- 随后20轮创建并关闭读取第一账本的临时原生窗口。每轮等待 dashboard 真正就绪后关闭，确认 workspace 进入 closed、响应/capture 被清空、注册窗口数回到基线；固定的第二窗口始终保持原 snapshot 和筛选。

所有输入都是仓库公开 fixture 的临时副本；测试不访问用户账本，也不写用户偏好。

## 验证

```sh
./apps/goui/native-e2e.sh -run '^TestNativeLedgerWindowsKeepReportFiltersIsolatedAcrossCloseCycles$' -count=3 -v
./apps/goui/native-e2e.sh -run '^TestNativeLedgerWindowsKeepReportFiltersIsolatedAcrossCloseCycles$' -race -count=1 -v
make goui-native-e2e
```

包含20轮开关的定向场景连续3次通过，race 通过。完整 native gate 通过：64个原生测试用例通过，3个环境门控用例跳过。该场景补齐旧验收场景16的双账本筛选与关闭隔离路径；进程空闲文件读取归因仍由 G39/G40 单独跟踪。
