# G13 通用页面导航栈

日期：2026-10-08。

## 实现

- 每个 Go UI 窗口各自维护返回/前进页面栈，历史上限为64页；相邻重复页面不重复入栈。
- 内容标题栏提供中文/英文返回与前进按钮。新页面导航清除前进分支；账本路径切换清空两侧历史。
- 报表、流水、账户、商品与价格、持仓、指令、语义配置、查询和附件导航会按目标页重新发送当前阅读请求。已有日期、币种、估值、搜索和页面选择状态由窗口模型保留。
- 账户详情的“返回报表”复用同一个导航栈；诊断、附件目录和 QA 验证页也进入历史。
- Finder 打开/窗口拖入的离线分析捕获现在进入同一导航历史。往返恢复捕获或读取错误时保留离线状态，不会触发核心查询。

## 验证

- `TestAccountDetailNavigationReturnsToFilteredStatement` 使用公开 cashflow fixture 和真实 `DesktopEngine`，通过 `ui.Tester` 实际点击报表账户、返回报表、侧栏流水、返回、前进和新导航；确认筛选结果及账户选择仍保留，且新导航清除前进历史。
- `TestPageHistoryIsBoundedAndAvoidsAdjacentDuplicates` 与 `TestViewNavigationMapsAreBidirectional` 覆盖历史容量和页面/请求映射。
- `TestNavigationToolbarFitsMinimumWindowAtLargeTextScale` 在 English、900×600 和 1.5×系统文字比例下核对返回/前进按钮可见且位于内容边界内；这是 headless 布局断言，不替代真实显示器目视检查。
- `TestNativePageHistoryBackForwardAndBranchReplacement` 通过 AppKit 原生 Accessibility 按钮驱动真实 Content 窗口，覆盖报表/流水返回与前进、改开损益表后禁用前进；连续3次通过。
- G28 的双账本原生回归现先在第一本账本中建立双向导航历史，再打开第二本账本并断言返回/前进历史都已清空，且 Accessibility 树中的两个按钮均不接受 Press；与现有路径切换代码一起在 `TestNativeQueryCatalogueIsLedgerScopedAndReloadSafe` 中验证，相关原生场景连续3次通过，详见[G28多账本与重载原生窗口记录](goui-p4-g28-native-e2e-2026-10-08.md)。
- `TestSemanticsPreviewIsSnapshotBoundAndApplyPersistsOnlyAfterOpen` 使用受控的 `DesktopEngine` profile reopen：在请求已受理但 V3 会话仍挂起时，确认 Back/Forward history 已清空；通过 `ui.Tester` 按 Back 不会切到旧页面或向新会话发起读取。立即失败的 open intent 会保留已有 history，因为重置发生在 `Dispatch` 成功后。
- G13 当前仍待真实 VoiceOver 与真实显示器目视验收；profile reopen 的挂起导航路径现在有可重复的 headless/真实 DesktopEngine 回归。
- `TestOfflineAnalysisPageRestoresThroughBackForwardHistory` 与扩展后的 `TestNativeFinderOpenAndWindowDropReadOfflineAnalysis` 覆盖离线捕获/错误页历史恢复，并确认往返没有打开账本或执行查询；原生场景连续3次通过。
- `make goui-check` 通过，包括 Go UI 测试与 Darwin 无 CGO 构建。
- `make goui-native-e2e` 通过；既有完整原生窗口回归继续通过。
- 当前代码已生成隔离 development `.app`、arm64 ZIP 和 SHA-256；ZIP 校验通过。尝试用桌面截图通道查看打包窗口时，ScreenCaptureKit 返回 `-3811`，因此没有声称完成打包窗口的人工目视验收；原生窗口交互由上述 AppKit 测试验证。

此项仍处于进行中：本次验证覆盖渲染器交互、核心请求与原生窗口回归；真人 VoiceOver以及900×600实际窗口的导航按钮布局仍属于 G37/G38 后续实机验收。
