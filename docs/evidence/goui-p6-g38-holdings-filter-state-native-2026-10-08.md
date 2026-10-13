# G38 持仓筛选未应用状态与多币种成本原生验证

日期：2026-10-08（Asia/Singapore）。使用真实 MyGo macOS Content 窗口与公开 `native/holdings-unknowns.bean`，验证 V3 持仓在日期截止下保留成本币种边界，并修复筛选尚未应用时控制器消失的问题。

原先 `holdingsPage` 在读取/过滤响应之前先检查完整请求身份。当用户修改分组、范围等会立即更新 Model 的筛选项时，旧响应不再匹配，页面提前返回；这同时隐藏了“应用持仓范围”按钮，用户无法提交新筛选。现先渲染页面切换、持仓筛选和排序控件，再检查数据响应；有旧响应但筛选身份已变化时明确提示“请应用持仓筛选以读取新结果”。旧数据仍不呈现为新筛选结果。

测试将账本截点设为 `2000-01-02`，核对原生窗口中的核心投影仍在同一 SnapshotID，并包含 USD 成本 `100` 对应 `10 STK`、EUR 成本 `60` 对应 `3 STK`。该 fixture 有 STK→EUR 与 EUR→USD 报价但没有直接 STK→USD 报价；核心投影和窗口都说明 USD 市价估值缺少本地报价，没有虚构 `150 USD`。测试通过原生 AXPress 打开分组选择器；确认“按成本币种”选项出现在窗口 Accessibility 树后，按当前节点 ID 选择它。Accessibility 树随后确认 Apply 控件仍存在且显示待应用提示，再用真实原生 AXPress 按钮提交请求。新响应再次核对 `10 STK / 100 USD` 和 `3 STK / 60 EUR`，并确认界面没有把两种成本相加成 `160`。

## 验证

- `./apps/goui/native-e2e.sh -run '^TestNativeHoldingsKeepCostCurrencyGroupsExact$' -count=1 -v`：通过。
- `./apps/goui/native-e2e.sh -race -run '^TestNativeHoldingsKeepCostCurrencyGroupsExact$' -count=5 -v`：5 次通过。
- `make goui-check`：通过。
- `make goui-native-e2e`：本轮未通过。全量进程运行约 204 秒后，输出多项需要当前 key window、菜单操作或系统面板的原生用例各自超时；其中有 `TestNativeHoldingsNavigatesGroupsAverageHistoryAndLots`、查询撤销/复制粘贴、文本缩放、偏好备份面板、来源编辑器面板及双窗口快捷键场景。为及时结束长时间 UI 等待，向本轮自启动的 `app.test` 发送 SIGQUIT 并记录栈；这些失败不计为通过。随后单独重跑最先失败的持仓导航场景，通过。检查到桌面前台进程是此前运行的 `orangecount-goui`；E2E `TestMain` 现调用 MyGo `App.Focus()` 激活自启动测试进程，但单独运行查询 Undo 场景仍超时，故旧实例前台不足以解释所有失败。没有关闭或更改旧实例；全量 UI 门禁的焦点/菜单失败原因仍未确定，不能据此声明 G38 整套原生门禁通过。

G38 仍在进行中：旧版18项完整等价矩阵尚未完成；真人 IME 与 VoiceOver由 G37 跟踪。

## 2026-10-08 原生 AX 操作与选择器稳定性复验

调查全量 gate 的焦点/菜单超时后，发现 Darwin 测试辅助函数此前只检查 AX action 是否允许，没有确认 MyGo 实际执行了该动作；MyGo 正在更新 Accessibility tree 时会拒绝动作。测试 hook 现按当前 MyGo 节点 ID 执行 AXPress，并报告是否实际执行；helper 重试直到成功，文本焦点也要等到 AX 状态确认。持仓成本币种测试通过原生选择器菜单项本身修改筛选值，并等待选项实际出现后按节点身份选择，不再直接修改 Model 字段。

- `./apps/goui/native-e2e.sh -race -run '^TestNativeHoldingsKeepCostCurrencyGroupsExact$' -count=5 -v -timeout=90s`：5 次通过。
- `make goui-check && git diff --check`：通过。
- `./apps/goui/native-e2e.sh -v -count=1 -timeout=3m`：完整 gate 通过，`apps/goui/internal/app` 用时 78.210 秒；5 个 opt-in 场景按条件跳过（十分钟 soak、两种外部编辑器启动、私人账本对照两项）。Finder/拖放场景有一条 sandbox extension 警告，但测试通过。

本次完整 gate 结果取代上文首次失败的 gate 记录。旧版18项完整等价矩阵、真人 IME/VoiceOver，以及 opt-in 私人账本和外部编辑器条件仍未验收，G37/G38 继续进行中。

同轮继续清理 4 处流水/查询 AXRow 选择调用：改用按 role/label 重新获取当前节点、核验 AXPress 已执行的 helper，覆盖旧场景8来源注释、旧场景10选择恢复/重复交易、长中文分录及查询结果行选择。4 个场景组合 race 连续5轮通过；之后 `make goui-check`、`git diff --check` 与 gofmt 检查通过。完整 native gate 在这次测试辅助代码清理之前已通过；该清理涉及的4个场景随后在 race 模式全部通过。

继续复核旧场景7时，将多币种 balance sheet 的估值币种步骤从直接写入 `Model.filterCurrency` 改为真实原生 selector 操作：展开“原币”、点选菜单中的 USD，确认 draft 已变但请求仍保持原币种，再通过“应用筛选”发出 USD 请求并核对 EUR 缺价解释。新的可复用 selector helper 同时承载持仓成本币种场景。两项场景组合 race 连续5轮通过；`make goui-check`、`git diff --check` 与 gofmt 检查通过。完整 native gate 的 78.210 秒结果早于这次 test-only selector 替换，本次以两个场景定向五轮验证。

再将 `TestNativeStatementShowsDatedConvertedAccountValues` 中的两次直接 Model 日期/币种修改改为真实 USD selector、Accessibility 日期字段与“应用筛选”操作；确认 2000-01-03 报价来源和 2000-01-02 的缺价部分估值。该场景 race 连续5次通过，`make goui-check` 与格式/空白检查通过。此次 test-only 修改后仍未重跑全量 native gate。

## 2026-10-08 完整 gate 最终复验

场景 7 的真实币种选择与待应用状态、日期汇率控件的顺序等待、以及 query reload 的稳定等待条件全部加入后，完整 `./apps/goui/native-e2e.sh -v -count=1 -timeout=3m` 通过，`apps/goui/internal/app` 用时 88.461 秒。包含新/改场景的目标 race 分别连续5或10次通过；`make goui-check`、`git diff --check`、gofmt 和空白检查通过。5 个 opt-in 场景跳过：十分钟 soak、两种外部编辑器启动和两项私人账本对照。Finder/拖放用例的一条 sandbox extension 警告后仍通过。该结果取代本文件先前“完整 gate 尚未重跑”的说明；旧版18项完整等价矩阵及真人 IME/VoiceOver仍待完成。
