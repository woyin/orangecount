# G10 原生筛选、导航与挂起读取记录

日期：2026-10-08（Asia/Singapore）。本记录收录 G10 的 macOS 原生窗口验收证据；P1 和 G38 的其他验收仍未完成。

## 验收场景

`TestNativeRapidDateCurrencyAndPageChangesStayOnSnapshot` 在 `t.TempDir()` 创建三个月账本，含 USD 现金余额及收入交易。测试在真实 MyGo 窗口中选择 USD，依次应用 2026 年 1、2、3 月日期范围，并在财务概览、资产负债表、损益表和流水之间导航。每次操作都检查请求已完成、日期和页面正确、原始 `SnapshotID` 不变；流水按产品语义保留原币，只应用日期筛选。返回 3 月仪表盘后，响应 series 的当月收入为 `30 USD`、期末净资产为 `60 USD`，原生 Accessibility 树也显示这两个值。

`TestNativeUIRemainsResponsiveDuringPendingRead` 通过一个只门控首个 `balance_sheet` 调用的测试 Session，保持 Go UI workspace 真实 Controller 的读取挂起。测试在同一原生窗口中继续填写日期、选择 USD、应用筛选并导航到财务概览；阻塞期间 workspace 仍为 Busy，原有 dashboard 投影与 SnapshotID 仍可见。解除门控后，旧 balance-sheet 结果不覆盖新请求，Controller 执行最新的 dashboard/date/currency 请求，结果保留相同 SnapshotID 和3月趋势，Accessibility 树显示 `60 USD` 与 `30 USD`。测试 Session 基于真实 `DesktopEngine`；门控用于稳定地产生慢读时段，不用于衡量解析性能。

MyGo 的自绘下拉选项在原生 Accessibility 树中表现为静态文本，因此 opt-in 测试桥通过可访问元素的可见 bounds 发送真实点击。普通运行不会调用该测试方法。

## 结果

两个原生场景分别连续运行5次通过：

```sh
./apps/goui/native-e2e.sh -run '^TestNativeRapidDateCurrencyAndPageChangesStayOnSnapshot$' -count=5
./apps/goui/native-e2e.sh -run '^TestNativeUIRemainsResponsiveDuringPendingRead$' -count=5
```

包含两项场景的完整24项原生窗口套件通过：

```sh
make goui-native-e2e
```

另外通过：`make goui-check`、`GOWORK=off CGO_ENABLED=1 go test -race ./... -count=1`（从 `apps/goui` 运行）、`GOWORK=off CGO_ENABLED=0 go vet ./...`、无 opt-in 原生测试跳过检查，以及 `git diff --check`。

工作区单测覆盖一项执行中加一项最新待读、请求快照绑定、旧 ledger revision 拒绝、读取错误时保留已有投影。结合这些测试和两项真实窗口场景，G10 的队列、发布与交互验收通过。
