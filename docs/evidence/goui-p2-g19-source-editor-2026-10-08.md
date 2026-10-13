# G19 外部来源编辑器设置

日期：2026-10-08（Asia/Singapore）。本轮补齐用户选择外部来源编辑器、持久化及来源动作路由，并区分代码级重建验证与实际应用进程重启验收。

## 行为

- 文件菜单新增“选择来源编辑器…”和“使用系统默认应用打开来源”。应用选择面板选择 `.app` bundle；Darwin 适配器读取 bundle ID 与显示名，并拒绝无效应用及 OrangeCount 自身。
- 应用级偏好只保存稳定 bundle ID，不保存所选应用路径。偏好跨重启读取，也不会被 per-ledger 阅读设置更新覆盖。QA 与生产仍由各自 userData 文件隔离。
- 来源动作执行时读取当前偏好。存在自定义编辑器时，Darwin `NSWorkspace` 按 bundle ID 打开；找不到应用时明确报错，不静默改用其他应用。路由前重新检查应用 bundle，阻止 OrangeCount 身份递归打开自身。
- 取消选择保留原设置；清除设置后恢复系统文件关联。若系统关联仍是 OrangeCount，既有 TextEdit/Finder 安全回退继续生效。

## 验证

TDD 用例覆盖偏好默认值、bundle ID 持久化、跨账本设置更新、取消和恢复系统默认、非法/OrangeCount bundle ID 拒绝，以及 Model 来源动作读取存储的 bundle ID。菜单中英文标签测试通过。Darwin 平台测试从 `/System/Applications/TextEdit.app` 解析出 `com.apple.TextEdit`，仅读取 bundle 信息，没有启动 TextEdit。新增隔离原生 MyGo 测试，实际激活“文件”菜单，打开来源编辑器原生面板并取消，确认原 bundle ID 保留；随后通过原生菜单恢复系统默认并确认偏好已清除。测试偏好写入 `t.TempDir()`。

```sh
cd apps/goui
env GOWORK=off CGO_ENABLED=0 go test ./internal/app ./internal/preferences ./internal/platform ./internal/i18n \
  -run '^(TestSourceEditorSelectionPersistsRoutesSourcesAndRestoresSystemDefault|TestApplicationMenuRoutesCommandsAndExposesMacShortcuts|TestApplicationMenuLocalizesForFocusedLedger|TestSourceEditorPreferencePersistsAndRejectsUnsafeBundleIDs|TestConfiguredSourceEditorRequiresExternalBundleIdentifier)$' \
  -count=1
env GOWORK=off CGO_ENABLED=0 go test ./internal/platform \
  -run '^TestSourceEditorFromApplicationReadsSystemEditorBundle$' -count=1
env GOWORK=off CGO_ENABLED=1 go test -race -count=1 ./...
env GOWORK=off CGO_ENABLED=0 go vet ./...
cd ../..
make goui-check
env GOWORK=off CGO_ENABLED=1 ORANGECOUNT_GOUI_NATIVE_E2E=1 \
  go test -tags orangecount_e2e ./internal/app \
  -run '^TestNativeSourceEditorMenuOpensPickerCancelsAndRestoresDefault$' -count=1 -v
env GOWORK=off CGO_ENABLED=1 ORANGECOUNT_GOUI_NATIVE_E2E=1 \
  go test -tags orangecount_e2e ./internal/app -count=1
```

以上均通过；完整 app suite 中的五项真实窗口场景全部通过。`make goui-check` 在当前 Darwin 主机构建生产 Go UI，并运行全套 Go UI 测试；嵌套 module 全包 race 测试与 `go vet` 也重新通过。应用偏好仍通过已有原子写入和 0600/0700 权限策略保存。

另有 opt-in 外部编辑器路由测试，将临时 `.bean` 来源通过 Darwin `NSWorkspace` 交给真实 `/System/Applications/TextEdit.app`：

```sh
ORANGECOUNT_GOUI_SOURCE_EDITOR_E2E=1 ./apps/goui/native-e2e.sh \
  -run '^TestNativeConfiguredSourceEditorRoutesFileToTextEdit$' -count=1 -v
```

测试通过，`OpenSourceWithEditor` 返回成功。运行日志显示 TextEdit 在请求前已经运行，所以本次证据是系统把打开请求交给现存 TextEdit；没有声称验证了首次启动。测试发现 TextEdit 原先未运行时才会尝试结束由测试启动的进程；本次现存实例保持不变。

## 新进程模型与自关联回退测试

新 Store 和新 Model 的路由测试验证偏好文件重开后仍读取原有 bundle ID，并把来源请求交给该编辑器。它覆盖数据持久化及模型重建路径，不等同于重新启动打包应用后观察 UI。

Darwin 回退顺序测试覆盖 TextEdit 成功时不调用 Finder、TextEdit 失败时调用 Finder，以及两者均失败时返回不可用错误。此前 G05 真实 QA 包来源测试已确认 Launch Services 默认关联指向 OrangeCount 时会把公开 `.bean` 文件交给 TextEdit，避免递归打开 OrangeCount。

检查本机 macOS SDK 的 `NSWorkspace.h` 后发现 `activateFileViewerSelectingURLs:` 返回 `void`，原实现却把其 ABI 按 `BOOL` 读取，可能误报回退失败。现在使用 SDK 声明返回 `BOOL` 的 `selectFile:inFileViewerRootedAtPath:`，并新增 opt-in AppKit 集成测试实际选择临时来源文件；Finder 返回选择成功。该测试直接验证系统选择 API，不模拟 UI 回调。

```sh
cd apps/goui
env GOWORK=off CGO_ENABLED=1 ORANGECOUNT_GOUI_FINDER_E2E=1 go test ./internal/platform -run '^TestSelectSourceInFinderWithAppKit$' -count=1 -v
```

测试通过。TextEdit→Finder 的回退顺序仍由独立平台策略测试覆盖；本机 TextEdit 可用，因此没有通过真实生产 self-association 场景强制令 TextEdit 失败。

对 `apps/goui/internal/app`、`internal/platform`、`internal/workspace` 和 `internal/diagnostics` 做日志调用审查：这些来源/诊断路径没有直接写日志的调用。应用入口的直接日志限于启动偏好定位、读取/迁移错误和 `App.Run` 返回错误；源码读取/parse 诊断由当前 workspace 状态呈现在 UI，不经这些入口日志。此项是静态代码审查结论，不是全系统日志捕获审计。

```sh
cd apps/goui
env GOWORK=off CGO_ENABLED=0 go test ./internal/app -run '^TestSourceEditorPreferenceRoutesAfterModelRestart$' -count=20
env GOWORK=off CGO_ENABLED=1 go test ./internal/platform -run '^TestOpenWithTextEditOrFinderFallbackOrder$' -count=20
```

两项测试均通过；完整回归 `make goui-check`、`make goui-native-e2e`、`env GOWORK=off CGO_ENABLED=1 go test -race ./internal/app ./internal/platform` 与 `env GOWORK=off CGO_ENABLED=0 go vet ./...` 均通过。

## 跨任务剩余验收

G19 任务卡列出的完成证据均已有对应验证：invalid/stale 区分、连续诊断位置更新、来源位置与自关联安全、系统剪贴板精确值，以及来源路径不记录账本正文。原生面板已实际选择 TextEdit.app；新 Store/Model 测试覆盖偏好持久化后的重新路由，Script Editor 场景覆盖配置编辑器首次启动；Finder BOOL 选择 API也已用临时文件实测。G19 因此记录为已验证。

打包应用的完整重启 UI 流程仍由 G43 开发包验收覆盖，真人 VoiceOver及其它多窗口 sheet 属于 G37/G38。TextEdit 可用，因此没有在真实 self-association 场景中强制令 TextEdit 失败；TextEdit→Finder 分支由策略测试和 Finder 系统 API 集成测试分别覆盖。

本轮补充 `TestSourceEditorFromApplicationRejectsOrangeCountBundles`，用隔离临时 `.app` bundle 验证 production bundle ID、development bundle ID 和仅以 `CFBundleName` 标识的 OrangeCount/QA app 均被来源编辑器选择器拒绝；系统 TextEdit 正常识别测试和三个拒绝用例通过：

```sh
cd apps/goui
env GOWORK=off CGO_ENABLED=1 go test ./internal/platform \
  -run '^TestSourceEditorFromApplication' -count=1 -v
```

这增强了自关联拒绝的自动化证据，不替代从原生选择面板实际选取 `.app`、首次启动或 Finder 回退验收。

2026-10-08 又新增 opt-in 原生窗口场景，使用 macOS 自带 Script Editor 验证首次启动：若它已经运行，测试会跳过并保留现有应用和文档；本机测试开始时它未运行，`NSWorkspace` 收到临时 `.bean` 的定向打开请求后启动了 Script Editor。测试清理只终止本次启动的实例。场景通过：

```sh
ORANGECOUNT_GOUI_SOURCE_EDITOR_E2E=1 ./apps/goui/native-e2e.sh \
  -run '^TestNativeConfiguredSourceEditorLaunchesScriptEditor$' -count=1 -v
```

这验证配置 bundle ID 对应应用的首次启动和打开请求，不替代从原生面板实际选择 `.app`、重启恢复、Finder 回退或 VoiceOver 验收。

## 原生面板实际选择 — 2026-10-08

用当前源码构建一个临时 QA bundle，在应用的“文件 → 选择来源编辑器…”菜单中打开真实 macOS Open Panel。通过面板的“前往文件夹”定位 `/System/Applications`，在列表中选中 `TextEdit.app` 并按“选择”。面板关闭后，欢迎页显示“来源编辑器已设为 TextEdit。”；只读检查 QA 偏好文件得到 `source_editor_bundle_id=com.apple.TextEdit`。

该 bundle 使用独立的 macOS bundle ID，但 MyGo 的 `PathUserData` 按 `App.Name` 创建，因此它与既有 Go UI QA 包共用 `OrangeCount Go UI QA` 偏好目录；没有改写生产 app 的偏好域或生产 userData。实际 UI 选取及 QA 偏好持久化已验证，重启后的新进程恢复尚未验证。本节不依赖之前已删除的实验性 AppKit 选择钩子。

随后尝试重新启动临时 bundle 检查新进程恢复时，`cua_repl` 返回 ScreenCaptureKit audio/video capture failure。应用 inventory 一度报告进程正在运行，但没有拿到其窗口 Accessibility 状态，因此不把重启恢复记为通过。当前 QA 偏好中来源编辑器值为 `com.apple.TextEdit`；生产偏好未触及。
