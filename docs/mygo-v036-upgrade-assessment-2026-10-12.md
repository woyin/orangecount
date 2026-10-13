# MyGO v0.3.6 依赖升级评估（2026-10-12）

日期：2026-10-12。结论：**现在不要替换**。`apps/goui` 当前固定 `v0.2.16-0.20261006170038-869768d02d91` 加本地 fork。上游 `v0.3.6` 在同一版本线上把公开 API 改成“checked element value”，需要一次覆盖面很大的迁移。本轮只做只读评估，没有修改任何代码、`go.mod` 或 `third_party` 内容。

本文只回答 review 第 4 节第 5 行“MyGO `v0.3.6`：作为独立依赖升级候选，逐项处理选区、组合输入、系统面板与生命周期接口，补新证据”。它是评估，不是升级提交。

## 1. 方法与证据边界

| 项目 | 内容 |
| --- | --- |
| 基线 | `HEAD 1c2096d` 加本轮开始的未提交工作树 |
| 本地依赖 | `github.com/egoist/mygo` pseudo-version `v0.2.16-0.20261006170038-869768d02d91`，`replace => ./third_party/mygo` |
| 上游候选 | `v0.3.6`，tag 对象 `ca2668d1f4646be1c5c72e53bce0ed92c0958706`，发布时间 `2026-10-09T07:55:03Z` |
| 获取方式 | `go mod download github.com/egoist/mygo@v0.3.6` 成功（网络可用）。只读下载到 module cache，未写仓库 |
| fork 结构 | `apps/goui/third_party/mygo` 不是 git checkout，没有 `.git`。它是 module cache 中 pinned 版本的手工快照加本地补丁；差异用目录级对比得到 |
| 探测方式 | 把 `apps/goui` 复制到 `/tmp`，改副本的 `go.mod` 指向 `v0.3.6`，只在那里运行构建与测试。仓库工作树没有被探测改动影响 |
| 未做到 | 没有运行真实 macOS GUI、真人 IME、真人 VoiceOver、真机 DPI 验收。本轮的“测试证据”全部来自 `ui.NewTester` 无头路径、`go build`/`go vet` 与包级测试 |

`fix/review-evidence-2026-10-12/` 只用于核对 fork 基线的测试计数（`current-goui-tests.jsonl`：11 包、282 pass、4 skip）。该证据与本地 fork 基线一致，不是 `v0.3.6` 的验收。

## 2. 本地 fork 与 pinned base 的差异清单

`apps/goui/third_party/mygo` 相对 pinned base 有 14 个文件改动内容、6 个新增文件（含 `UPSTREAM.md` 与 5 个压缩包文件）。两类合计约 1,180 行，其中一半以上是 opt-in 测试钩子。

| 文件 | 差异行数 | 用途 | 类别 |
| --- | ---: | --- | --- |
| `ui/editor.go` | 24 | `(*Element).SetTextSelection(start, end) bool`，半开 rune offset，聚焦编辑器并显示尾部 caret，IME 组合中返回 false | 选区 API |
| `ui/context.go` | 50 | `(*Context).SetTextScale(scale float32)`，叠加桌面辅助功能字号 | 字号 API |
| `ui/element.go` | 10 | `FontSize` 应用 `textScale` | 字号 API |
| `dialog.go` | 125 | `Dialog.SaveWithContext(ctx, opts)`、`ErrNilSaveContext`、`ErrSaveDialogCancellationUnsupported`、`toSaveDialogOptions` | 系统面板 |
| `internal/platform/platform.go` | 12 | `SaveDialogHandle`、`CancellableSaveDialogs` 能力接口 | 系统面板 |
| `internal/darwin/dialog.go` | 28 | `NSSavePanel` 的 `cancel:` 句柄与 completion 幂等 | 系统面板 |
| `internal/fake/fake.go` | 46 | fake backend 支持可取消 pane 与 hold 开关 | 系统面板测试 |
| `window.go` | 32 | `(*Window).clearListeners()`，最后一个事件后释放全部回调 | 生命周期 |
| `ui/window.go` | 5 | `detach()` 清除 idle timer 并报告 e2e host 生命周期 | 生命周期 |
| `internal/darwin/testhooks.go` | 104 | `TestPasteboardSnapshot/RestorePasteboard`、`TestAccessNodeWithID`、`TestAccessibilityWithIDs`、`TestHasSheet`、`TestClickSheetButton`、`TestOpenPanelVisible`、`TestSavePanelVisible`、`TestApplicationRunning`、`TestTerminateApplication`、`TestAccessibilityPress(_ID/_Allowed)`、`TestSurfaceLifecycle`、`TestCancelSheet`、`TestAccessNode.Help` | 测试钩子（未加 tag） |
| `e2e_testhooks_darwin.go`（新增） | 230 | `E2E*` 公开钩子：可访问性、点按、键盘、组合输入、输入客户端、剪贴板、面板、拖放、进程、surface 生命周期 | 测试钩子（`orangecount_e2e`） |
| `internal/darwin/e2e_testhooks.go`（新增） | 188 | 带 node ID 的可访问性、sheet、面板、进程、AXPress | 测试钩子（`orangecount_e2e`） |
| `internal/darwin/testhooks_accessibility_click_e2e_darwin.go`（新增） | 37 | 按 AX label/ID 点击 | 测试钩子（`orangecount_e2e`） |
| `ui/e2e_lifecycle_enabled.go` / `_disabled.go`（新增） | 51 / 6 | `ui.E2EHostLifecycle()` 与 no-op 版本 | 测试钩子（`orangecount_e2e`） |
| `mygo_test.go`、`ui/gpu_test.go`、`ui/preferences_test.go`、`ui/textarea_test.go` | 236 | fork 行为的回归测试 | fork 测试 |

fork 的快照还省略了 pinned 版本中的 `docs/`、`examples/`、`website/`、`plugins/`、`packages/`、`scripts/`、`.github/` 与 `AGENTS.md`。这些不参与编译。

### 2.1 分歧面比 review 记录的更大

review Standards 第 4 项记录了一个 API：`SetTextSelection` 的返回类型。实际 fork 提供了两个公开 API（`SetTextSelection`、`SetTextScale`）加一个可取消系统面板能力（`SaveWithContext`），并有约 500 行 opt-in 测试钩子。升级评估必须覆盖全部四项，不能只处理选区。

## 3. v0.3.6 相对 pinned base 的变化规模

`v0.3.6` 并非小版本修补。pin 是 `2026-10-06` 的 `v0.2.16` 预发布，`v0.3.6` 是 `2026-10-09` 的正式 tag，中间经过 `v0.2.17`、`v0.2.18`、`v0.3.0`—`v0.3.5`。

| 指标 | 数值 |
| --- | ---: |
| `diff -rq` 报告不同的文件 | 249 |
| `ui/` 下改动或新增的非测试 `.go` 文件 | 82（60 个改动 + 22 个新增） |
| `ui/api_gen.go` 生成文件行数 | 4,678 行（180 个 `Element` 值接收者方法） |
| `ui/` 新增文件 | `api_gen.go`、`frame.go`、`editor_input.go`、`editor_selection.go`、`editor_layout.go`、`editor_navigation.go`、`editor_history.go`、`ref.go`、`focus_binding.go`、`services.go`、`list_builder.go`、`data_drag.go`、`handles_debug.go`、`handles_release.go` 等 |
| `internal/darwin`、`internal/linux`、`internal/windows` 改动 | 各 10、10、9 处 |

核心变化是 **checked element value 模型**：

- 默认与开发构建中，widget 构造器返回 `ui.Element` 值（16 字节 checked handle），不是 `*ui.Element` 指针。
- 元素在**下一次构建 pass 前过期**，同一帧内的多次 pass 也会过期。查询过期元素在开发构建会 panic；`mygo_noinspector` 生产构建返回空结果或忽略写入。
- 持久身份改用 `ui.Handle`（`Bind`、`Focus`、`OnShortcut`、`Bounds`），或者 `FocusBind`/`FocusedValue`。
- 上游提供 `go tool mygo migrate-ui` codemod 与 `go tool mygo vet` 生命周期检查。
- `ui.Element` 有 `Valid()`（`ui/frame.go:118`），开发构建中过期元素会 panic，`mygo_noinspector` 生产构建返回空结果。`ui.Element{}` 是“缺席”零值。

同一变化还改变了输入时序：表格表头点击在 **view 函数返回之后**才应用（新增 `afterInput` 机制）。pinned 版本在同一 pass 内应用。

## 4. 逐项兼容表：当前 `apps/goui` 依赖的 fork API

全模块（含测试文件）引用 49 个非 E2E 的 `mygo.*` 符号、47 个非 E2E 的 `ui.*` 符号，另有 26 个 `mygo.E2E*` 符号与 2 个 `ui.E2E*` 符号。按依赖 fork 补丁的四个面分组如下。

图例：**破坏** = `v0.3.6` 无等价公开 API，需要适配；**改名/改型** = API 存在但签名不同；**兼容** = 行为等价。

### 4.1 选区（选区与错误定位）

| 依赖 API | 位置 | v0.3.6 状态 | 适配工作 | 需要的证据 |
| --- | --- | --- | --- | --- |
| `(*ui.Element).SetTextSelection(start, end) bool` | `internal/widgets/queryeditor/offset.go:38`（唯一调用点） | **破坏**：`v0.3.6` 有 `Element.SetTextSelection(int, int) Element`，只设选区，**不聚焦**，没有防组合输入的守卫，返回 `Element` | `SelectByteRange` 改为 `ui.Element` 参数，用零值判断替代 `nil`；显式补 `editor.Focus()`；补 `editor.Composing()` 判断；返回值函数改为“成功返回 true” | `internal/widgets/queryeditor` 单测（已存在 `TestSelectByteRangeSelectsTheCoreDiagnosticToken`、`TestSelectByteRangeLeavesEditorUnchangedForInvalidBytes`）；`internal/app` 的 `TestQueryPageLocatesUnicodeParserRangeInEditor`、`TestQueryErrorNavigationDoesNotMoveMarkedInputComposition` |
| `*ui.Element` 作为编辑器参数 | `internal/app/query.go:219` `locateQueryError(editor *ui.Element, ...)` | **改名/改型**：`*ui.Element` 不再是 `ui.TextArea` 的返回类型 | 参数改为 `ui.Element` 值 | 同上 |
| `ui.Local(e, key, init)` | `internal/charts/hit.go:20` | **改名/改型**：`v0.3.6` 只接受 `ui.Element` 值 | 参数类型跟随上游 | `internal/charts` 全包测试 |
| `ui.TextArea` / `ui.TextInput` 返回值 | `internal/app/*`、`internal/widgets/*` | **改名/改型**：返回 `ui.Element` 值 | 所有接收变量、结构字段、函数返回值改为值类型 | 全包 `go build` + `go vet` |

上游确实提供了 `Element.Valid()`（`ui/frame.go:118`），但没有“防组合输入的选区设置”：`SetTextSelection` 无条件改写 `editorSelection.anchor/caret`，不检查 `compose`。本轮实测的关键行为差异（在临时副本上用 `ui.NewTester` 探测，两边都跑同一探针）：

| 探针 | fork（现状） | v0.3.6 |
| --- | --- | --- |
| 先点按钮再 `SetTextSelection`，然后输入 | 选中区间被替换：`SELECT 'x', account LIMIT 0` | **选区不生效**：`SELECT 'x', missing LIMIT 0`（未聚焦） |
| 同上但额外调用 `Focus()` | 同左 | 替换成功：`SELECT 'x', account LIMIT 0` |
| 直接对已聚焦编辑器设选区 | 替换成功 | 同左 |
| 组合输入期间 `SetTextSelection` | 保持组合，选区不变，`Composing()` 仍为 true | 选区被设置，`Composing()` 仍为 true，组合文本未被破坏 |

结论：`v0.3.6` 能表达 fork 的选区行为，但**必须由应用补两步**（`Focus()` 与 `Composing()` 判断）。上游不会替应用保留“组合输入中不移动选区”的约定。少了这两步，现有测试 `TestQueryPageLocatesUnicodeParserRangeInEditor` 会直接失败。

### 4.2 组合输入（IME）

| 依赖 API | 位置 | v0.3.6 状态 | 适配工作 | 需要的证据 |
| --- | --- | --- | --- | --- |
| `(*ui.Element).Composing() bool` | `internal/app/query.go:45,50` | **兼容**：`Element.Composing() bool` 存在 | 接收者类型改值 | 现有 `TestQueryEditorBlocksShortcutDuringComposition` 类用例、`TestQueryErrorNavigationDoesNotMoveMarkedInputComposition` |
| 组合输入期间的快捷键拦截语义 | `internal/app/query.go` 的 `m.executeQuery(editor.Composing())` | **需复验**：`v0.3.6` 重写了编辑器输入路径（`editor_input.go`、`textbuffer_input.go`、`TextInputClient`），组合输入的处理位置不同 | 通常不需要改调用点，但要跑通用例 | 无头组合探针 + 真人 IME 验收 |

本轮探测只覆盖无头 `ui.Tester.Compose`。真实 macOS 输入法候选窗、marked text 与撤销事务在 `v0.3.6` 中被重写（新增单一 undo transaction for preedit），本轮**没有**验证。这是必须补齐的证据。

### 4.3 系统面板（保存面板与取消）

| 依赖 API | 位置 | v0.3.6 状态 | 适配工作 | 需要的证据 |
| --- | --- | --- | --- | --- |
| `mygo.Dialog.SaveWithContext(ctx, opts) (string, error)` | `internal/app/preferences_backup.go:66`、`internal/app/query_export.go:284,319` | **破坏**：`v0.3.6` 只有 `Dialog.Save(opts)`；`CancellableSaveDialogs`、`SaveDialogHandle`、`ErrSaveDialogCancellationUnsupported`、`ErrNilSaveContext` 全部不存在 | 三个调用点都要改写。若保留取消语义，必须重新实现可取消面板（fork 的做法是在 `internal/darwin` 里拿 `NSSavePanel` 句柄并 `cancel:`），或者退回无取消并接受“关窗时面板仍可见”的回归 | 现有 `TestQuerySaveCoordinatorCancelsWhenLeavingQueryPage`、`TestReportSaveCoordinatorCancellationDoesNotWrite`、`preferences_backup_test.go` 的取消用例；原生 `TestNativeQuerySavePanelClosesWithOwnerWindow`、`TestNativeQueryRunningAndSavePanelOwnersCloseConcurrently` |
| `mygo.FileFilter`、`mygo.SaveDialogOptions` | 同上 | **兼容** | 无 | — |
| fork 的 `toSaveDialogOptions` 收敛 | fork `dialog.go:271` | **不适用**：上游没有导出同名 helper，应用也不调用它 | 无 | — |
| `model.querySaveDialog` / `preferenceBackupSaveDialog` 注入点 | `internal/app/model.go:154,158` | **兼容**：是应用自己的函数字段，与 API 无关 | 无 | 现有导出测试 |

`v0.3.6` 的 `dialog.go` 与 pinned base 逐字节相同；125 行差异只来自 fork。丢失可取消面板后，`TestNativeQuerySavePanelClosesWithOwnerWindow` 与 `TestNativeQueryRunningAndSavePanelOwnersCloseConcurrently` 会失去依据：这两个场景依赖“owner 窗口关闭时 sheet 立即消失”。这是 spec 级回归，不是编译级回归。

### 4.4 生命周期（窗口监听器与 idle timer）

| 依赖 API | 位置 | v0.3.6 状态 | 适配工作 | 需要的证据 |
| --- | --- | --- | --- | --- |
| `(*mygo.Window).clearListeners()` | fork `window.go` 内部（应用不直接调用） | **破坏但可旁路**：`v0.3.6` 与后续 `v0.3.7`、`v0.4.0`—`v0.4.5` 都没有 `clearListeners`，关窗后仍保留全部回调 | 无编译改动；这是一个**未修复的内存保留**。应用在 `cmd/orangecount-goui/main.go` 注册的 `OnFocus`、`OnFileDrop`、`OnClose`、`OnClosed` 闭包会在窗口关闭后继续被 slice 引用 | 复验 `docs/evidence/goui-p6-g40-2026-10-08.md` 的监听器释放结论；若无 fork 补丁，该结论在 `v0.3.6` 上不成立 |
| fork `detach()` 清 idle timer | fork `ui/window.go` 内部 | **破坏但可旁路**：`v0.3.6` 的 `detach()` 仍只 `Stop()`，不清 `idleTimer`、不清 `idleArmed`/`idleAt` | 无编译改动；需要新证据判定实际影响（timer 已 Stop，泄漏面是 host 与 timer 对象的可见保留） | `TestDetachStopsAndForgetsIdleTimer`（fork 单测）；原生 `TestNativeWindowIdleAndCloseReleaseFrameTimers` |
| `ui.E2EHostLifecycle()`、`ui.E2EHostLifecycleStats` | `internal/app/native_e2e_darwin_test.go:316,357,364,376` | **破坏**：只存在于 fork 的 `orangecount_e2e` 构建 | 需要重建等价钩子，或改写该原生断言 | 原生 idle/frame timer 场景 |

### 4.5 其余破坏面：测试钩子与元素类型

本轮用 codemod 后仍剩下的编译错误只有一条（`offset.go`），但 **E2E 构建** 与 **无头测试** 还有独立的破坏面。

| 面 | 依赖 | v0.3.6 状态 | 适配工作 |
| --- | --- | --- | --- |
| `mygo.E2E*`（26 个符号、385 个调用点） | `internal/app/native_e2e_darwin_test.go` | **破坏**：`v0.3.6` 只有 `internal/darwin` 的 `Test*` 内部钩子（`TestAccessibility`、`TestPerformKeyEquivalent`、`TestEndSheet`、`TestKey`、`TestCompose`、`TestDropFiles` 等），没有 fork 的 `E2E*` 公开层，也没有 `TestAccessNodeWithID`、`TestPasteboardSnapshot/RestorePasteboard`、`TestHasSheet`、`TestClickSheetButton`、`TestSavePanelVisible`、`TestApplicationRunning`、`TestTerminateApplication`、`TestAccessibilityPress*`、`TestSurfaceLifecycle`、`TestCancelSheet` | 需要重建一层 `E2E*` 包装（约 27 个函数、500 行），或改写 385 个调用点 |
| `ui.E2EHostLifecycle`（2 个符号） | 同上 | **破坏** | 重建 fork 的 `orangecount_e2e` 钩子 |
| `*ui.Element` 作为结构字段与函数返回 | `internal/app/localization.go:19`（`localizedSelect.element`）、19 个函数签名 | **改名/改型** | codemod 覆盖大部分；`localizedSelect` 需要人工复核（上游 codemod 明确提示“review stored Element fields; use Handle”） |
| `ui.OutlineState[T]` | 账户树 | **兼容**：`v0.3.6` 仍是泛型类型 | 无 |
| `ui.Table`、`ui.ListState`、`ui.SortOrder`、`ui.TableColumn`、`ui.TableLayout` | `internal/widgets`、`internal/charts` | **兼容**（签名相同） | 无 |
| `ui.NewTester` 与 26 个 `Tester` 方法 | 全部无头测试 | **兼容**：`Tester.Find`、`SetPreferences`、`Compose`、`Type`、`Key`、`Click` 等都在 | 无 |
| 其余 `mygo.*`（`App`、`Window`、`Menu`、`Clipboard`、`Shell`、`RunOnMain`、`Theme` 等 49 个非 E2E 符号） | `cmd/orangecount-goui`、`internal/app` | **兼容** | 无 |

## 5. 实测结果

探测在 `/tmp` 副本中进行，仓库工作树未改动。

### 5.1 直接替换（未适配）

| 命令 | 结果 |
| --- | --- |
| `CGO_ENABLED=0 go build ./...`（`v0.3.6`，未适配） | 失败：12 个类型错误，分布在 `internal/widgets`（4）、`internal/charts`（7）、`internal/widgets/queryeditor`（1） |
| `go vet ./...`（未适配） | 同样 12 个错误 |
| 错误种类 | 11 个是 `ui.Element` 值 vs `*ui.Element`；1 个是 `SetTextSelection` 返回类型 |

注意：直接替换仍能通过**依赖解析**。`v0.3.6` 的 `go.mod` 与 fork 完全相同（`purego v0.11.1`、`go-text/typesetting v0.3.5`、`golang.org/x/image v0.46.0`、`go 1.27.1`）。升级不需要新的依赖，也不需要改根 `go.mod`。

### 5.2 最小适配后（`go tool mygo migrate-ui -write` + 3 个适配点）

适配改动共 14 个文件、约 40 行（不含适配器）：

| 改动 | 说明 |
| --- | --- |
| codemod | 12 个文件：`*ui.Element` → `ui.Element`、`nil` 判断 → 零值判断、`ui.Local` 参数随上游 |
| `offset.go` | 显式 `Focus()` 与 `Composing()` 判断（见 4.1） |
| `model.go` | `SetTextScale` → 适配器（`c.Theme()` 拷贝后改 `FontSize`，再 `SetTheme`） |
| `preferences_backup.go`、`query_export.go` | `SaveWithContext` → 适配器（`ctx.Err()` 检查 + `Dialog.Save`） |

适配后 `CGO_ENABLED=0 go build ./...` 与 `go vet ./...` 通过。

### 5.3 适配后的测试对比

同一套测试、同一台机器、`CGO_ENABLED=0`、`-count=1`。基线树与探测树都复制到 `/tmp`，并软链同一份 `testdata`，因此两者面对相同的 fixture 与相同的路径假设。

| 树 | pass | fail | skip | 失败包 |
| --- | ---: | ---: | ---: | --- |
| 本地 fork 基线（同源副本） | 275 | 3 | 4 | `internal/app` |
| `v0.3.6` 最小适配 | 273 | 5 | 4 | `internal/app` |

基线树的 3 个失败是 `/tmp` 路径假设造成的噪声，不是产品缺陷：

- `TestWelcomeOffersExplicitlyReopenableRecentLedger`
- `TestPricesPageRendersCatalogAndExactSourceOrderedHistory`
- `TestHoldingsPageRevealsActualMarketQuoteSource`

`v0.3.6` 树多出的 2 个失败是真实行为差异：

- **`TestQueryPageLocatesUnicodeParserRangeInEditor`**：适配时如果只改类型、不补 `Focus()`/`Composing()`，选区不生效。补上后通过（本轮已验证）。
- **`TestQueryTableSortsTypedIntegersAndKeepsSelectedRowForCopy`**：`v0.3.6` 用 `afterInput` 把表头排序推迟到 view 之后，应用的同一帧“重排 + 按行号重映射选中项”失效，选中项停在错误行。需要在 view 开头按 `queryAppliedSort` 顺序先把选中行映射回旧索引、再映射到新顺序。本轮已用最小补丁验证该算法可行（同一测试通过）。
- 其余包（`charts`、`i18n`、`localfiles`、`platform`、`preferences`、`querypresets`、`widgets`、`widgets/queryeditor`、`workspace`）在适配后全部通过。

这些数字只说明“可达”，不说明“等价”。同一份数字不能替代 GUI、IME 与 VoiceOver 验收。

## 6. 需要的适配工作（汇总）

| 序号 | 工作 | 规模 | 风险 |
| ---: | --- | --- | --- |
| A1 | `ui.Element` 值迁移：19 个函数签名、1 个结构字段、`ui.Local` 调用点、`nil` 判断改零值 | codemod + 人工复核，约 14 文件 | 中：`localizedSelect` 的跨帧保存需要改为 `ui.Handle` 或重构 |
| A2 | 选区适配：`SelectByteRange` 补 `Focus()` 与 `Composing()` | 1 文件、约 6 行 | 中：行为依赖两个显式调用，容易在后续重构中丢失 |
| A3 | 跨帧字号适配：`ui.Context.SetTextScale` 等价物 | 1 文件、约 10 行适配器 | 中：`SetTheme` 与桌面辅助功能字号的叠加顺序需要验证 |
| A4 | 保存面板取消：重建 `SaveWithContext` 与 `CancellableSaveDialogs` | `dialog.go`、`internal/platform`、`internal/darwin`、`internal/fake`，约 180 行 | 高：这是原生行为，重新实现等于再次维护一份 fork |
| A5 | 生命周期补丁：`clearListeners` 与 `detach()` idle timer 清理 | `window.go`、`ui/window.go`，约 40 行 | 中：没有上游 API，只能继续 fork 或接受保留 |
| A6 | E2E 钩子层：27 个 `mygo.E2E*` / `ui.E2E*` | 约 500 行 + 385 个调用点 | 高：钩子依赖上游 `internal/darwin` 私有状态（`accessElementOf`、`node.ID`、`idleArmed`），上游重构会反复破坏 |
| A7 | 排序时序适配：`queryAppliedSort` 重映射 | `internal/app/query.go`，约 15 行 | 中：需要在真实表格交互中复验 |
| A8 | 组合输入复验：`Composing()` 语义与 undo 事务 | 无代码改动 | 高：无头覆盖不足，必须真人 IME |

A4 与 A6 是决定性问题。它们不是迁移成本，而是“fork 必须继续存在”的证据：`v0.3.6` 没有可取消保存面板，也没有 OrangeCount 原生验收依赖的 E2E 可观察面。升级后这两块工作等于把 fork 重写一遍。

## 7. 验收计划（按顺序）

升级只有在每一项都有证据后才能提交。下表是建议顺序；前四项可以合并执行，后三项必须单独记录。

| 步骤 | 命令或动作 | 通过条件 |
| --- | --- | --- |
| E1 单元 | `GOWORK=off CGO_ENABLED=0 go test ./... -count=1`（`apps/goui`） | 11 包通过；失败数不高于 fork 基线 |
| E2 静态 | `GOWORK=off CGO_ENABLED=0 go vet ./...` | 通过，且 `go tool mygo vet` 生命周期检查无新增告警 |
| E3 race | `GOWORK=off go test -race -count=1 ./...` | 通过 |
| E4 无头定向 | `-run 'TestQueryPageLocatesUnicodeParserRangeInEditor\|TestQueryErrorNavigationDoesNotMoveMarkedInputComposition\|TestQueryTableSortsTypedIntegersAndKeepsSelectedRowForCopy\|TestTextScale'` | 全部通过；这些是 A2、A3、A7 的判定用例 |
| E5 GUI | `make goui-native-e2e` | 全部常规场景通过；监听器/idle timer 计数不高于 fork 基线 |
| E6 IME | 真人输入法（拼音候选）手工场景 | 组合期间错误定位不移动选区；组合结束后可继续输入；无候选窗丢失 |
| E7 VoiceOver | 真人 VoiceOver 朗读关键页面与表格行 | 与 fork 基线一致；不因 `Element` 值迁移丢失可访问性节点 |

E6、E7 是本轮明确未覆盖的项，也是 `mygo-native-desktop-tasks-2026-10-07.md` 中 G37“真实输入法和 VoiceOver 验收”仍未完成的项。升级会改动编辑器输入路径与构建时长，因此 E6、E7 必须在升级后重跑，不能引用现有 fork 结论。

## 8. 建议

**现在不升级。** 推荐状态：继续使用当前 fork，直到下列条件同时满足。

建议升级的触发条件（满足任意一条后再启动）：

1. 上游发布包含可取消保存面板（或等价能力），并且具备应用可用的事件监听器释放与 idle timer 清理。
2. 上游回归测试面覆盖 OrangeCount 依赖的原生验收：AX 节点身份、sheet 探测、剪贴板快照恢复、进程与 surface 生命周期。
3. `apps/goui` 的 `SetTextScale` 与 `SetTextSelection` 组合输入约定被上游采纳，或有明确的上游替代方案。

如果必须现在升级，建议顺序与最小范围：

| 顺序 | 内容 | 说明 |
| ---: | --- | --- |
| 1 | A1 + A2 + A3 + A7 | 这四项只改 `apps/goui`，不改依赖，可先做“依赖升级到 `v0.3.6`，其余 fork 补丁保留在 fork 分支上”的过渡 |
| 2 | A5 + A6 | 在 `apps/goui/third_party` 之外维护补丁层，避免把 385 个调用点与 500 行钩子改写一遍 |
| 3 | A4 | 只有在上游提供可取消面板后才移除；否则继续保留 fork 的 `dialog.go` 补丁 |
| 4 | E1—E7 | E6 与 E7 不通过则回退 |

在过渡状态下，`go.mod` 的 `replace` 仍然存在，但指向的 fork 只保留 3 个补丁（可取消面板、监听器清理、E2E 钩子）。这是本次评估认为最现实的目标形态，但它不减少“fork 必须继续存在”的事实。

不建议把 `v0.3.6` 与 `apps/goui` 的真实功能迁移放在同一批改动里。选区的 `Focus()`/`Composing()` 适配、跨帧字号适配、排序时序适配各自都有测试退化风险，混合提交会掩盖退化来源。

## 9. 本轮未覆盖

- 没有运行 `make goui-native-e2e`，没有打开真实窗口。`v0.3.6` 的 385 个 E2E 调用点与 27 个钩子符号是静态比对结果，不是运行结果。
- 没有真人 IME、VoiceOver、DPI 与深浅色验收。
- 没有评估 `v0.3.7`—`v0.4.5`。抽查 `v0.4.5` 的结果是：`SetTextScale` 仍不存在，`clearListeners` 仍不存在，可取消保存面板仍不存在，E2E 钩子仍不存在，`SetTextSelection` 仍是 `Element` 值返回。因此更高版本不改变本评估的结论。
- 没有评估 `v0.3.6` 新增能力（`Handle`、`FocusBind`、`FocusedValue`、`c.OnShortcut`、`Element.OnClick`、`data_drag`、`transfer` 包、`clipboard.go`）能否替代 fork 用途。这些是潜在收益，但需要单独设计。注意 `ListState` 在 `v0.3.6` 内嵌了 `Handle`，可作为 A1 中 `localizedSelect` 的迁移目标。

## 10. 证据与复现命令

```sh
# 只读获取上游源码
cd "$(mktemp -d)" && printf 'module probe\n\ngo 1.27.1\n' > go.mod
go mod download github.com/egoist/mygo@v0.3.6

# 列出可用版本（网络可用）
go list -m -versions github.com/egoist/mygo

# fork 与 pinned base 的差异（pinned base 在 module cache 中）
M="$(go env GOMODCACHE)/github.com/egoist/mygo@v0.2.16-0.20261006170038-869768d02d91"
diff -rq "$M" apps/goui/third_party/mygo

# 第 3 节表格的计数（对照 v0.3.6 的 ui/ 目录）
V="$(go env GOMODCACHE)/github.com/egoist/mygo@v0.3.6"
diff -rq "$M/ui" "$V/ui" | grep -E '\.go' | wc -l                              # 全部 .go：141
diff -rq "$M/ui" "$V/ui" | grep -E '\.go' | grep -v '_test.go' | wc -l         # 非测试：82

# 未适配的直接替换（在临时副本中）
rsync -a --exclude build --exclude orangecount-goui \
  apps/goui/ /tmp/goui-probe/
cd /tmp/goui-probe && go mod edit -require=github.com/egoist/mygo@v0.3.6 \
  && go mod edit -dropreplace=github.com/egoist/mygo
CGO_ENABLED=0 go build ./...

# 最小适配
go run github.com/egoist/mygo/cmd/mygo@v0.3.6 migrate-ui -write .
```

本轮使用的实际探测目录：`/tmp/octest`（`v0.3.6`）、`/tmp/octest-base`（fork 基线）。两者都未写入仓库。
