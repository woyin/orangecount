# G37/G38 原生 MyGo 窗口端到端检查

日期：2026-10-08（Asia/Singapore）。本记录覆盖新增的 opt-in macOS 原生窗口测试，不代表 G37 或 G38 完成。

## 测试隔离

- 测试和桥接代码使用 `darwin && orangecount_e2e` build tag；只有设置 `ORANGECOUNT_GOUI_NATIVE_E2E=1` 才启动 MyGo 原生应用循环。普通测试运行不弹窗，生产构建不包含这些测试钩子。
- 测试只读取仓库公开 `testdata/fixtures/native/cashflow.bean`；恢复场景在 `t.TempDir()` 创建和修复派生账本。
- 新增图表验收在 `t.TempDir()` 创建独立三个月账本，用真实 AppKit 原生窗口/Accessibility help 测试仪表盘现金流图的键盘选期与精确值。
- 新增筛选与导航验收在 `t.TempDir()` 创建独立三个月账本，通过原生日期文本框、币种下拉和页面导航控件验证 USD/日期请求、同一快照以及最终3月仪表盘的精确收入与净资产值。
- 原生操作通过 AppKit 窗口及其 Accessibility 和文本输入客户端执行。测试不启动第二个应用，也未请求 Accessibility 或屏幕录制权限。

## 覆盖行为

首轮29项原生场景覆盖：cashflow fixture 的 dashboard、资产负债表与流水导航；三个月临时账本的日期/币种筛选与财务概览、资产负债表、损益表、流水切换，并核对同一 SnapshotID 和3月精确收入30 USD/净资产60 USD；通过门控的真实 DesktopEngine 会话保持 balance-sheet Read 挂起，原生窗口仍能改日期和 USD 币种、应用筛选、导航到 dashboard，解除门控后最新请求覆盖队列且旧结果不发布；三个月临时账本的仪表盘现金流图 Home/右箭头/End 键盘选期，并从 macOS Accessibility help 核对逐月精确收入与支出；通过原生 Accessibility 选择 Salary 交易并检查精确分录；在105笔临时账本中原生翻到第2页、返回第1页并检查100/5条边界与快照身份；在52,500笔临时账本中原生检查前两页各投影100笔、总数/页码范围正确且保持同一SnapshotID；公开 `directives/all.bean` 的原生 all/config 页面检查生成关联、typed decimal/NULL/bool 以及2099日期条件下配置仍按源码顺序返回；使用公开cashflow账本连续创建/关闭20个真实Content窗口，逐轮检查workspace关闭状态并记录RSS/HeapAlloc；在两个真实窗口打开同一账本，授权临时附件目录后从另一个窗口原生移除，并确认两边策略撤销、附件重新分类为越权、旧附件令牌不触发系统打开；公开 `average-history.bean` 的 Legacy/V3 预览从原生页面显示差异，点击“应用 V3”后核对新epoch、同一输入身份、草稿清理、profile持久化和账本字节不变；空账本损益表的“当前期间没有活动”状态；v3 多币种余额分别保留 `1 USD` 和 `2 EUR`，切换 USD 估值后显示缺少本地报价；BQL 编辑器的 marked text 与 `é` 提交；包含 emoji 的 BQL 查询错误定位 UTF-8 token，并从 AppKit Native Input Client 检查 UTF-16 选区及提交替换；typed BQL 只有显式点击“执行查询”才执行，并检查重复列别名、NULL、空字符串、结果行数及 Accessibility 呈现；两个临时账本在同一个真实窗口间切换并验证命名预设隔离，账本文件改变后从原生“文件 → 重新读取账本”门控重载，确认旧 snapshot 的核心查询目录暂时隐藏、更新后新目录恢复且账本自定义预设仍保留；同一窗口显示 `main.bean:1:1` 语法诊断，修复临时账本后重新读取并验证路径/epoch 不变、交易数更新且 dashboard 返回 `30 USD`；验证外部入口文件写入非法内容时保留原快照、修复后不点击“重新读取”即可自动载入新快照并刷新 dashboard；验证被原子替换的 include 文件自动触发新快照和 dashboard 更新；通过 MyGo `OpenFiles` delegate 测试路径路由，首个账本进入欢迎窗口，已有工作区保留并为下一个账本创建新窗口；验证失效最近路径作为无有效快照的诊断工作区打开，之后打开其他账本时保留诊断窗口；来源编辑器菜单打开原生选择面板后取消，确认面板只附着到发起命令的窗口且两个窗口均无残留 sheet；从流水页打开原生报表 `NSSavePanel`，确认它属于触发窗口、可取消且不会留下文件或 sheet；保存面板显示期间关闭所属窗口，确认面板消失、另一窗口无孤立 sheet、workspace 关闭且没有文件落盘；从实际 BQL 查询结果打开查询 `NSSavePanel`，确认它附着到当前窗口、可取消且没有生成文件；另确认保留冻结 query capture 时关闭 Save Panel 所属窗口会销毁面板、释放 workspace response/capture、不给另一窗口留下孤立 sheet，也不落盘；Save Panel 打开前 gate 住 query capture 再关闭 owner 时，workspace 会等待 capture 调用结束并丢弃迟到结果。偏好备份菜单打开原生 `NSSavePanel`，验证发起窗口归属与取消；随后用隔离的测试保存适配器写出临时 JSON 备份并读回。当前有46项常规窗口场景，另有独立G40十分钟reload/RSS soak，见[G40记录](goui-p6-g40-2026-10-08.md)。

G07 原生场景后来增加了 AppKit 鼠标点击现金流柱状图中点，并核对二月的精确收入/支出 Accessibility help；该扩展场景连续5次通过。实现期间发现并修复偏移布局命中计算混用窗口与元素局部坐标的问题，headless 折线图/柱状图偏移命中回归也通过。详见[G07图表命中验证](goui-p0-g07-2026-10-07.md)。

查询、导航和编辑器测试现等待目标 Accessibility 控件/文本实际到达后才继续，避免将异步 UI 发布误判为产品故障。曾出现的查询草稿、多币种文本及流水页码缺失来自测试没有等待原生树刷新；同步条件补齐后，新增流水详情/分页场景重复运行10次通过。报表保存面板归属/取消场景也重复运行10次通过。没有证据表明这些失败对应生产代码缺陷。

## 执行与结果

```sh
make goui-native-e2e
```

首轮二十九项原生窗口场景均通过；后续新增场景纳入完整套件后，当前有48项常规窗口场景。双窗口附件撤权、AVERAGE 语义切换、G28多账本重载、query Save Panel owner 关闭与挂起 capture owner 关闭场景各以 `-count=5` 连续运行5次通过。runner 使用 `GOWORK=off`、`CGO_ENABLED=1`、`darwin && orangecount_e2e` build tag 和 opt-in 环境变量；普通生产构建不包含测试钩子。图表键盘/Accessibility help和 Unicode 查询错误选区场景各重复10次通过；四项 G09 场景和两项 G10 筛选/导航场景分别重复5次通过；偏好备份面板取消/保存场景重复5次通过；流水详情/105笔分页、报表与查询面板取消及报表所属窗口关闭场景也分别重复10次通过；52,500笔 GUI 分页、G17长中文完整换行、G18全类指令和嵌套值换行场景各连续运行5次通过。G17原生详情 frame 为628×53点，位于900×568内容区内，完整精确金额可访问；G18原生嵌套值 frame 为177×153点，位于900×568内容区内，NULL、TRUE及精确decimal可访问。20窗口生命周期测试在同一测试进程中连续运行3轮（共60轮窗口/账本打开关闭）通过。另以 race 模式运行 Go UI 全量 package 测试并通过，当前 `make goui-check` 也通过。独立 `make goui-native-soak` 另行执行十分钟原生Reload测试，不计入常规套件场景数。

## 双窗口 Cmd+O/Cmd+R 路由补充

新增原生场景验证 Cmd+O 打开的 NSOpenPanel 只挂到当前 key window，取消后两个窗口均无遗留 sheet；Cmd+R 只增加当前账本 workspace operation 并完成重读，另一个窗口的 operation 不变。两项场景各以 `-count=5` 连续通过，随后完整 `make goui-native-e2e` 全套通过（48项常规窗口场景，33.069 秒），`make goui-check` 通过。详情见[G36 双窗口 Cmd+O/Cmd+R 原生路由验证](goui-p6-g36-native-open-reload-2026-10-08.md)。

偏好备份场景重复运行5次的命令：

```sh
./apps/goui/native-e2e.sh -run '^TestNativePreferencesBackupMenuPanelCancelAndSave$' -count=5
```

新图表场景单独复验命令：

```sh
./apps/goui/native-e2e.sh -run '^TestNativeDashboardCashflowChartKeyboardAndAccessibilityHelp$' -count=1
```

该场景另以 `-count=10` 连续运行10次，通过。

Unicode 查询错误原生选区场景单独复验命令：

```sh
./apps/goui/native-e2e.sh -run '^TestNativeQueryErrorSelectsUnicodeRangeInInputClient$' -count=10
```

连续10次通过，测试会在 emoji 前缀之后检查 token 选区，然后提交替换文本。

流水详情/分页、报表与查询保存面板取消及报表所属窗口关闭场景分别连续运行10次通过：

```sh
./apps/goui/native-e2e.sh -run '^TestNativeJournal(SelectsTransactionAndShowsExactPostings|PaginationStaysOnSnapshot)$' -count=10
./apps/goui/native-e2e.sh -run '^TestNativeReportSavePanelIsOwnedAndCancellable$' -count=10
./apps/goui/native-e2e.sh -run '^TestNativeReportSavePanelClosesWithOwnerWindow$' -count=10
./apps/goui/native-e2e.sh -run '^TestNativeQuerySavePanelIsOwnedAndCancellable$' -count=10
```

未设置 `ORANGECOUNT_GOUI_NATIVE_E2E=1` 时，30个 `TestNative...` 测试入口（含常规29项及独立十分钟soak）均按预期跳过且不启动原生窗口；长时soak还要求 `ORANGECOUNT_GOUI_NATIVE_SOAK=1`。

G28 双账本隔离与重载场景单独复验命令：

```sh
./apps/goui/native-e2e.sh -run '^TestNativeQueryCatalogueIsLedgerScopedAndReloadSafe$' -count=5
```

连续5次通过。该场景对旧 snapshot 核心目录进行失效保护；按账本身份保存的显式自定义预设不会被重载误清除。

外部文件监视自动恢复场景单独复验命令：

```sh
./apps/goui/native-e2e.sh -run '^TestNativeExternalFileWatchRecoversWithoutManualReload$' -count=5
```

连续5次通过；未设置 opt-in 环境变量时，同一测试按预期跳过且不启动原生窗口。

include 文件原子替换场景单独复验命令：

```sh
./apps/goui/native-e2e.sh -run '^TestNativeIncludedFileWatchUpdatesWithoutManualReload$' -count=5
```

连续5次通过。

G10 日期/币种筛选与页面切换场景单独复验命令：

```sh
./apps/goui/native-e2e.sh -run '^TestNativeRapidDateCurrencyAndPageChangesStayOnSnapshot$' -count=5
./apps/goui/native-e2e.sh -run '^TestNativeUIRemainsResponsiveDuringPendingRead$' -count=5
```

两个场景各连续5次通过。筛选导航场景检查各页面请求、同一 SnapshotID 和3月趋势；挂起读取场景在余额表请求仍执行时继续改日期/币种并导航，解除门控后核对最终请求、快照与原生精确值。

MyGo 文件打开路由场景单独复验命令：

```sh
./apps/goui/native-e2e.sh -run '^TestNativeFinderOpenUsesWelcomeWindowAndPreservesPopulatedWindow$' -count=5
```

连续5次通过。

## 边界与未验收项

- 标记文本测试检查 AppKit input client 和应用草稿，不等同于真人拼音输入法的候选翻页、确认、取消、多窗口切换。图表测试读取原生 Accessibility help，但没有验证 VoiceOver 实际朗读。因此 G37 仍未验收。
- 本段检查完成时的原生窗口覆盖数为二十九项；之后的新增场景与当前总数见本记录后续补充。旧版18项行为的完整等价矩阵、附件授权/失效恢复、配置/全指令/全部报表和查询保存仍未全部通过。因此 G38 仍未验收。附件场景以 Darwin bookmark 和 Model 策略入口添加目录，尚未自动化真实目录选择面板的授权动作。
- 原生 `native-e2e` 自动化覆盖 Save Panel 的归属与取消、报表所属窗口关闭清理；偏好备份通过隔离测试保存适配器确认成功写出，但没有自动化远程 `NSSavePanel` 的最终确认按钮。2026-10-08 另以 `cua_repl` 对独立 QA-mode Go UI 执行可见保存操作，查询 CSV 与流水报表 CSV 均实际保存、由 CSV parser 读回并检查行数；细节与摘要 hash 见[G05 原生保存确认记录](goui-p0-g05-2026-10-07.md)。这一手工 GUI 证据不等同于自动化保存回归，G37/G38 仍未完成。
- 未运行私人账本，也未改变默认应用关联、生产包或发行设置。

## 2026-10-08 后续增补

G29 新增 `TestNativeFinderOpenAndWindowDropReadOfflineAnalysis`，覆盖 `.ocanalysis` 经 MyGo application delegate 打开及 Finder 风格原生窗口拖放。该场景连续运行3次、并纳入后续 `make goui-native-e2e` 全套；全套现有30项常规原生窗口场景，另有1项独立十分钟 soak 入口。该补充只验证应用事件处理与窗口拖放，没有更改默认应用关联，也没有进行 Launch Services 双击验收。

G40 又新增 `TestNativeQueryExecutionOwnerCloseWaitsForLateSessionCall`：底层真实 Desktop Session 的查询调用被测试 gate 挂起时关闭 owner window，检查 workspace 等待该调用、关闭后丢弃迟到结果且不弹保存面板/写文件。该场景 `-count=5` 连续通过，并纳入后续 `make goui-native-e2e`；全套现有31项常规原生窗口场景，另有1项独立十分钟 soak 入口。

随后新增 `TestNativeQueryRunningAndSavePanelOwnersCloseConcurrently`，在两个窗口分别挂起真实 query execution、持有冻结 capture 并显示 query Save Panel，再连续关闭两个 owner。该场景 `-count=5` 通过；之后重跑的 `make goui-native-e2e` 全套包含32项常规窗口场景，另有1项独立十分钟 soak 入口。

G40 后续新增 `TestNativeWindowIdleAndCloseReleaseFrameTimers`：窗口打开公开账本并静置后，检查 Go UI idle timer、AppKit 帧 timer 与运行中 display link 已停止；关闭后 host、surface、display link 和 timer 数回到基线。该场景 `-count=5` 通过；当前 `make goui-native-e2e` 有33项常规窗口场景，另有1项独立十分钟 soak 入口。

本轮尝试读取当前运行中的 QA app 窗口以补充可见手工验收，但 macOS ScreenCaptureKit 无法启动 capture stream（报错 `Failed to start stream due to audio/video capture failure`）。此尝试不计为 G37/G38 手工 GUI、IME 或 VoiceOver 证据；相应项继续待验。

G14 后续新增 `TestNativeStaleSemanticsPreviewCannotApplyAfterInputChanges`，对临时账本修改、刷新期间旧操作隐藏、旧 InputIdentity 拒绝和 Legacy 会话保留进行真实窗口验证，连续5次通过；当时共有34项常规窗口场景。随后 G17 新增长中文换行和窄窗口布局验收，使数量达到35项；G18 再增加嵌套中文换行与typed-value保真验收，使数量达到36项常规窗口场景和1项独立十分钟 soak。

G19 初版系统剪贴板探针使用通用辅助技术名称；多条诊断来源动作同名，原生测试可能命中错误按钮。来源复制、定位、打开动作现将路径和行列加入 Accessibility 标签，既给 VoiceOver 更具体的上下文，也让原生定位唯一。全套原生 gate 后续发现页面切换期间 AX 树可能暂时保留欢迎页内嵌诊断卡片的退出节点；原生测试现在等路径唯一的复制按钮节点 ID 连续8个50毫秒间隔样本稳定后，再通过 AppKit `AXPress` 激活，并核对系统剪贴板中的精确绝对路径、行和列。`TestNativeInvalidDiagnosticsRecoverInTheSameWindow -count=10` 稳定性复验连续通过；稳定化后的全量 gate 正在重跑，结果会在 G19 恢复证据后续增补。G19 的剪贴板动作有稳定节点原生重复证据；Finder/source-editor 应用启动、编辑器面板选择 bundle、真人 VoiceOver朗读和辅助技术实际操作仍未验收，因此 G19/G37 继续进行中。

G36 新增 `TestNativeFindShortcutTargetsTheFocusedLedgerWindow`：两个真实账本窗口依次聚焦并通过 AppKit Cmd+F key-equivalent 路由查找，原生 Accessibility focused 状态分别落到当前窗口的流水搜索框；单项连续5次通过。加入该场景后，`make goui-native-e2e` 的常规窗口场景数从36增至37，完整套件通过。该场景只验证菜单 key-equivalent 和焦点路由，不替代物理键盘/IME/VoiceOver验收；详见[G36 双窗口 Cmd+F 原生验证](goui-p6-g36-native-find-2026-10-08.md)。

## 2026-10-08 后续完整 gate

G19 Accessibility 稳定节点等待及 G25 原生 BQL Edit 菜单测试加入后，`make goui-check` 与 `make goui-native-e2e` 均通过，当前原生套件为40个常规窗口场景和1个独立十分钟 soak 入口。新增 G25 场景对中文/emoji 查询执行实际 AppKit Select All、Copy、Paste 菜单，验证系统 pasteboard 内容、选中替换、不运行查询；单项连续5次通过，并精确恢复测试前 general pasteboard 的 item/type/byte 内容。测试会在无法保存 promise type 时不改动并跳过。

该 gate 证明自动化覆盖在本机继续稳定；真人 IME 候选操作、VoiceOver 实际朗读/激活、完整旧客户端 GUI 等价矩阵、真实 NSOpenPanel 选择应用及 Launch Services 首次启动仍未完成，G37/G38 和 G19 状态不变。

随后 G26 新增 `TestNativeQueryCancelButtonCancelsCurrentExecution`，以真实 `DesktopEngine` Session 挂起当前 query generation，经原生 Accessibility 按下“取消查询”，并验证该 session 收到相同 request ID，最终返回 typed `cancelled`。该场景 `-count=5` 通过；取消中 UI 显示独立进度并阻止重复取消。新增后的 `git diff --check`、`make goui-check` 与 `make goui-native-e2e` 均通过，常规原生窗口场景数增至41，十分钟 soak 仍独立执行。

随后 G26 新增 `TestNativeReloadDuringQueryRejectsLateResult`。测试以真实 `DesktopEngine` 和 MyGo 原生窗口运行查询，通过原生“文件 → 重新读取账本”菜单切换到更新的 snapshot；旧查询在新 snapshot 发布后才从 gate 释放，最终确认迟到结果没有重新填入 `QueryResponse`。该场景 `-count=5` 连续通过；加入后的 `git diff --check`、`make goui-check` 和 `make goui-native-e2e` 均通过。当前原生套件为42项常规窗口场景，另有1项独立十分钟 soak 入口。G26 的 profile 切换、重新打开和无窗口异步展示仍待验收。

之后 G26 增加 `TestNativeProfileReplacementDuringQueryRejectsLateResult`：在 Legacy 查询挂起期间打开真实语义配置页并点击“应用 V3”，等新 profile 会话及 snapshot 进入就绪态后才释放旧查询，确认新会话不接收迟到结果。Profile重开与File菜单reload两个迟到结果场景各连续5次通过；随后 `make goui-check`、完整 `make goui-native-e2e` 与 `git diff --check` 通过。当前套件为43项常规窗口场景，另有1项独立十分钟 soak。G26的无窗口异步展示和完整旧版 GUI 等价矩阵，以及 G37/G38 的实机验收仍未完成。

新增 `TestNativeQueryCompletionWhileAwayFromQueryPageIsRestored`：原生执行真实 BQL 后切到财务概览，确认独立查询 runner 继续运行；查询在该页完成后回到查询页，仍能看到原 request/snapshot 上下文和 typed 结果行。该场景连续5次通过。此次全量 `make goui-check` 与 `make goui-native-e2e` 通过，常规原生窗口场景增至44项，十分钟 soak 独立计数。结合原有 draft 保留、request generation、切账本/reload拒绝、profile重开拒绝、执行/取消/结果显示与无共享日期过滤项证据，G26 的任务验收条件现已满足，任务台账标记为已验证；这不替代 G38 的旧客户端全 GUI 等价矩阵，也不代表 G37/G38 用户验收完成。

## 2026-10-08 导航历史补充

G13 页面历史回归现覆盖离线分析返回/前进及读错恢复，并将账本切换后的导航重置加入 G28 双账本原生测试。打开第二本账本后，测试确认 Back/Forward 历史都为空，且等待新 Accessibility 树刷新后确认两个控件均不可按；此项与独立页面历史场景连续3次通过。随后 `make goui-check` 和完整 `make goui-native-e2e` 通过。更新前原生测试文件声明47个 `TestNative...` 函数：46个常规场景和1个独立十分钟 soak 入口。打包应用可见 GUI 的查询/导出仍待 G43 验收；此前启动独立包时，macOS 拒绝 osascript 辅助功能访问，故未把它计为包内交互证据。

## 2026-10-08 Cmd+N 新窗口补充

新增 `TestNativeNewWindowShortcutCreatesNativeWindow`，通过 AppKit Cmd+N 连续创建两个可访问的原生 Content 窗口。该场景以 `-count=5` 通过；随后 `make goui-check` 与完整 `make goui-native-e2e` 通过（49项常规原生场景，33.311 秒）。当前原生测试文件有50个 `TestNative...` 函数：49个常规窗口场景和1个 opt-in 十分钟 soak 入口。

## 2026-10-08 账户、持仓与商品/价格页面补充

新增三项原生页面场景并各以 `-count=3` 重复通过：账户详情 AX 行选择/同快照返回、`Assets:Stock` 持仓筛选并读取 AVERAGE 历史与批次、按账户搜索商品后从原生报价菜单选择 USD 并核对 STK 截止价。商品菜单中的 USD 与后方表格单元格重名；测试钩子现可按当前 MyGo accessibility node ID 点击，确保选择实际菜单项。该钩子仅在 `darwin && orangecount_e2e` 下编译。实现还为账户余额树增加原生可访问名称及独立账户详情按钮，并为报价下拉框增加不重复的中英文辅助名称。

最新完整 `make goui-native-e2e` 通过，耗时36.172秒；原生测试文件共有53个入口，即52项常规窗口场景和1项独立十分钟 soak。`make goui-check`、根 `make fmt vet test race license build`、Go UI `go test -race`/`go vet` 与 arm64/amd64 development 包校验也通过。新增三页原生场景、package SHA-256 和未完成的实机验收边界详见[G21/G22/G23 页面补充记录](goui-p6-g21-g23-native-pages-2026-10-08.md)。

## 2026-10-08 附件目录原生面板补充

新增 `TestNativeAttachmentRootOpenPanelIsOwnedAndCancellable`。测试用隔离临时账本和偏好打开附件目录管理页，通过 MyGo Accessibility 点击“添加目录…”，等待真实 AppKit `NSOpenPanel` 出现并确认它附属于当前 Content 窗口；再取消面板，验证它关闭、`picking` 状态清除、workspace 附加 roots 为空且偏好未保存 bookmark。该场景连续5次通过。

新增 `TestNativeAttachmentRootSelectionPersistsAndAuthorizesDocument`。测试仍运行在真实 MyGo Content 窗口，实际通过原生 Accessibility 点击“添加目录…”；为了稳定验证 picker 返回后的产品流程，测试注入隔离 adapter 返回临时附件目录。之后执行真实 Darwin bookmark 创建/重新解析、workspace policy 更新和临时偏好持久化，检查附件从 `OutsideRoot` 转为 `Available`，并确认原生附件页显示“可用”。该场景连续5次通过。它不证明真实 `NSOpenPanel` 内部的鼠标/Accessibility 目录选择动作；那一项仍待验。

加入两个附件场景后，完整 `make goui-native-e2e` 通过（38.728秒），`make goui-check`、`GOWORK=off go test -race -count=1 ./...` 与 `GOWORK=off CGO_ENABLED=0 go vet ./...` 通过；授权成功的 native E2E 场景单独以 race 模式运行也通过。原生测试文件共有55个入口，即54项常规窗口场景和1项独立十分钟 soak。

两个场景分别证明目录选择器的呈现/窗口归属/取消语义，以及接收选择路径后的授权与持久化流程；还没有真实选择器内部的目录选中、宽范围确认、重启后的书签恢复，也不代替真人 IME/VoiceOver、旧版18项等价矩阵或打包 GUI 验收。对应 G35、G37、G38、G43 仍进行中。

```sh
./apps/goui/native-e2e.sh -run '^TestNativeAttachmentRootOpenPanelIsOwnedAndCancellable$' -count=5 -v
./apps/goui/native-e2e.sh -run '^TestNativeAttachmentRootSelectionPersistsAndAuthorizesDocument$' -count=5 -v
make goui-native-e2e
make goui-check
```


## 2026-10-08 附件 bookmark 恢复与失效重选补充

新增 `TestNativeAttachmentRootRestoresBeforeDocumentRead` 与 `TestNativeInvalidAttachmentBookmarkCanBeReselected`。前者用全新的偏好 Store、Model 和原生窗口验证磁盘 bookmark 在账本打开时恢复为 Session 附件策略，附件读取可用且恢复不改写偏好；后者模拟原目录被替换，验证失效标记、同路径重新授权、新 bookmark 持久化和附件状态恢复。两个原生窗口场景组合运行5轮均通过，race detector 定向运行通过。最新完整 `make goui-native-e2e` 通过（56项常规窗口场景，37.165秒）；`make goui-check` 与 `git diff --check` 通过。native 测试入口共57个：56项常规窗口场景和1项十分钟 soak。

fresh Model/Store 测试仍不等同于进程重启；另一个隔离 Manual QA `.app` 已通过实际 UI 面板选择、退出重启和账本重开验证其 bookmark 恢复，临时授权及相应 QA preference/recent path 已清理。该结果不等同于 G43 development bundle 的完整首启/读写验收。宽范围根目录确认、真人输入法候选交互和 VoiceOver、旧版18场景全量等价矩阵及 development 包完整 GUI 流程仍未完成。宽目录确认自动化探针曾因 `_NSAlertPanel` 不支持测试代码调用的 `buttons` selector 而使测试进程退出；相关探针和实验测试已移除，没有用户账本或系统偏好写入。G35、G37、G38、G43 状态仍为进行中。


## 2026-10-08 真实附件面板与 QA 包重启补充

在隔离 Manual QA `.app` 中通过原生 Accessibility UI 打开真实 `NSOpenPanel`，从“前往文件夹”选择临时附件目录；UI 显示已保存目录。打开引用临时 `receipt.pdf` 的账本后附件显示“可用”。退出并重新启动同一 QA bundle，重新打开账本后附件仍为“可用”。完成后移除授权、清除本次测试写入的 QA ledger preference/recent path，并删除临时账本和附件文件。该手工结果证明真实面板选择和 QA bundle 的进程重启恢复，不代表 G43 development bundle 的安装/首启/查询/导出验收；宽范围目录确认、真人 IME、VoiceOver、旧版18场景完整等价矩阵仍未完成。

## 2026-10-08 输入法实机补查

只读检查系统键盘页面确认已配置 `ABC` 与 `Squirrel - Simplified` 输入源；Input Sources 快捷键为 `Control+Option+Space`。在实际 development bundle 查询编辑器中尝试系统切换快捷键并逐键输入 `nihao`，编辑器仍得到普通拉丁文本，没有出现可观察的 marked text 或候选确认，因此不计为真人 IME 通过。测试查询草稿已恢复为 `SELECT 1 AS v LIMIT 1`，系统设置页恢复到原先的 General 页面；未改动输入源配置。截图通道在本轮返回 ScreenCaptureKit `-3812`，所以没有视觉候选窗证据。候选翻页、Enter/Esc、混输及切窗/面板场景仍待真人操作验收。

同日对实际 G43 development bundle 做了补充复验：通过查询页焦点编辑器逐键输入 `nihao`，再按空格和 Return；Accessibility 值只显示 `nihao`（空格后仍为拉丁文本），没有确认到中文候选。之后已将草稿恢复为 `SELECT 1 AS v LIMIT 1`。本次读取 System Settings 与窗口截图分别被 ScreenCaptureKit `-3812` / `-10005` 阻断，未改变系统输入源设置；该次尝试也不计作真人 IME 通过。

## 2026-10-08 查询结果表原生键盘与类型化复制补充

新增 `TestNativeQueryResultSortAndKeyboardSelectionKeepRowIdentity`：在 900×600、150% 字号的真实 cashflow 窗口执行 BQL，聚焦结果行后通过 Home/End 和排序后的 Up/Down 操作检查键盘选择；用原生 Accessibility 按 `number` 表头后验证数值升序且 Salary 行身份保持；点击“复制类型化结果行”后逐字节比较 general pasteboard JSON 与执行上下文、列 schema 和原始 typed cells。测试进入时保存剪贴板，退出时恢复并再次核对原始项目/类型/字节。定向用例连续5次通过，包含该场景的 `make goui-check`、Go UI `go vet`、app/i18n race 测试、根模块全量测试及完整 `make goui-native-e2e` 均通过；当时的原生 gate 为65个常规原生用例通过、3个条件门控场景跳过，耗时66.217秒。该覆盖不替代真人 VoiceOver、旧版全 GUI 等价矩阵或 DPI/视觉验收，G27、G37、G38仍进行中。

## 2026-10-08 欢迎页最近账本与原生取消状态补充

新增 `TestNativeWelcomeRecentLedgerOpensOnlyAfterSelection`：使用非 QA 应用 Model 与隔离偏好文件预置公开 cashflow fixture 后，真实欢迎窗口仍无活动账本；通过 Accessibility 明确选择后才打开并显示两笔交易及现金流值。该场景连续5次通过，随后完整 `make goui-native-e2e` 通过（63.926秒）。详情见[G03欢迎页原生验收](goui-p0-g03-native-welcome-2026-10-08.md)。

完整套件曾在 `TestNativeQueryCancelButtonCancelsCurrentExecution` 观察到取消结束后单份 Accessibility 快照仍短暂列出“取消查询”按钮。定向 `-count=100` 复现2次；带诊断采样的复现显示 controller 与 Model 的 QueryBusy/QueryCancelling 均已清零，旧 Accessibility 节点下一次 20ms 采样即消失。测试现要求连续3次 Accessibility 采样确认按钮消失，避免把单帧旧快照误判为产品状态；修正后同一场景 `-count=100` 通过，完整原生套件也通过。未改生产查询状态逻辑。

## 2026-10-08 Holdings selector 与完整 gate 稳定性补充

定位到 Darwin E2E helper 曾在 Accessibility action “允许”时就返回，没有核对 MyGo 是否实际执行；Accessibility tree 更新期间动作会被拒绝。辅助函数现以当前节点 ID 调用 AXPress，确认 performed 后才算成功，文本输入焦点等待 AX focused 状态；成本币种持仓测试也改为真实打开 selector、观察菜单选项并按节点 ID 选择。该场景 `-race -count=5` 连续通过；随后 `make goui-check && git diff --check` 通过，完整 `./apps/goui/native-e2e.sh -v -count=1 -timeout=3m` 通过（78.210秒）。5 个 opt-in 场景跳过：十分钟 soak、两种外部编辑器启动和两项私人账本对照；Finder/拖放用例虽有 sandbox extension 警告，仍通过。G37/G38 的真人 IME/VoiceOver、旧版18项完整等价矩阵和以上 opt-in 环境条件仍未验收。

之后将剩余4处 AXRow 直接 press 调用改为同一执行确认 helper。场景8成本/价格来源、场景10插入注释后恢复及重复交易清理、长中文流水行和查询结果选择合并运行5轮 race 全通过；随后 `make goui-check`、`git diff --check`、gofmt 检查通过。此测试辅助代码清理没有重跑完整 native gate；相关4个场景已做对应重复验证。

继续复核旧场景7时，将多币种 balance sheet 测试从直接写入 `filterCurrency` 改为真实打开“原币”selector 并选择 USD，验证待应用 draft、应用后的 request currency 和缺价说明；该场景与持仓成本币种 selector 合并 race `-count=5` 全通过，随后 `make goui-check`、`git diff --check`、gofmt 检查通过。此前78.210秒完整 native gate 在此次 test-only 修改之前运行，本次没有重跑全量套件。

之后将按日期汇率窗口验证中的直接 Model 日期/币种赋值改为真实 USD selector、Accessibility 日期字段填入及 Apply 按钮操作。`TestNativeStatementShowsDatedConvertedAccountValues` race 连续5次通过，确认 Jan 3 使用的 EUR/USD 报价来源及 Jan 2 缺失报价状态；`make goui-check`、格式/空白检查通过。全量 native gate 未在这项 test-only 修改后重跑。

## 2026-10-08 场景7 selector 与全量 gate 最新结果

随后增加 USD selector 变更后等待 AX 中的 USD draft 渲染，再填写日期；这避免日期写入与 selector 状态更新争用。也将 reload/query 场景的断言从瞬时 `Busy` 状态改为旧 query 清空与新 snapshot 发布两个持久结果，并加入超时状态诊断；该场景 race `-count=10` 通过。最终 `make goui-check` 通过，完整 `./apps/goui/native-e2e.sh -v -count=1 -timeout=3m` 通过（88.461秒）。5 个 opt-in 场景按条件跳过：十分钟 soak、两个外部编辑器、两项私人账本对照。Finder/拖放 sandbox extension 警告未导致失败。G37/G38 真人 IME/VoiceOver与旧版18项完整等价矩阵仍进行中。

## 2026-10-08 初次 Legacy 无效恢复与最新完整 gate

新增 `TestNativeInvalidLegacyLedgerCanApplyV3Recovery`：真实窗口打开默认 Legacy 下无有效快照的公开 holdings fixture，进入语义配置，核对 V3 有效预览和“应用 V3”动作，再验证同一 ledger path 以 V3 建立有效 snapshot。该场景连续3次通过。headless regression 另验证 invalid ledger 的旧 revision 预览被拒绝。

`GOWORK=off go test -race ./internal/app -run '^TestSemantics' -count=5`、`make goui-check` 通过；随后 `./apps/goui/native-e2e.sh -v -count=1 -timeout=3m` 完整通过，用时 81.641 秒。5 个 opt-in 场景按设计跳过（十分钟 soak、两种外部编辑器及两项私人账本对照）；Finder/拖放一条 sandbox extension warning 未导致失败。该结果覆盖此前 88.461 秒 gate 之后加入的 invalid-profile 原生场景。旧版18项完整等价矩阵、真人 IME/VoiceOver 与 opt-in 外部条件仍未验收。G43 最新源码包的可见 profile/holdings 操作见[G43 语义恢复与持仓筛选包验收](goui-p7-g43-semantics-recovery-visible-2026-10-08.md)。
