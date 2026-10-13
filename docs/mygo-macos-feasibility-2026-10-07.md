# MyGo 用于 OrangeCount macOS 的可行性研究

研究日期：2026-10-07。本次完成第一手文档/源码研究，以及临时目录中的接入与运行探针。

**后续路线决定（2026-10-07）**：用户在研究后明确选择直接使用 Go `ui`。执行依据已转为[详细设计](mygo-native-desktop-design-2026-10-07.md)、[实施计划](mygo-native-desktop-plan-2026-10-07.md)和[任务台账](mygo-native-desktop-tasks-2026-10-07.md)。下面保留研究时的建议与验证事实；WebView 路线不纳入当前迁移计划，研究探针不能记为新 Go ui 产品验收。

**结论：技术上可以采用 MyGo 重写桌面客户端，但现阶段不建议直接替换 SwiftUI 主版本。** Go 会计引擎和报告层可以继续复用；真正需要更换的是窗口、展示层及客户端状态管理。MyGo 的主要收益是统一 Go 桌面开发、减少 Swift/C ABI 和双语言 DTO 维护，并为跨平台提供路径。若优先复用现有 Web 工作台，WebView 路线更有实际收益；若继续追求当前 macOS 控件体验，`ui` 路线须先证明图表、中文输入和无障碍能力足够。

这项判断已有接入实测支持，但没有同账本性能对比、完整 GUI 验收或正式发行验证，不能宣称迁移会更快、更省内存或更可靠。

## OrangeCount 当前源码基线与可复用边界

研究开始时工作区干净，HEAD 为 `1c2096d435e07cc9b5a5957e0e974546ee1387a6`。本次只新增本文，没有修改应用源码、根 `go.mod` 或现有构建产物。

- 现有 macOS 是 SwiftUI + AppKit + Swift Charts，通过 Go `c-archive` 在同一进程读取账本，无 HTTP 监听端口；构建目标为 macOS 13 / arm64。事实来源：[README](../README.md#native-macos-reader)、[构建脚本](../apps/macos/build.sh)、[工具链设置](../apps/macos/toolchain.sh)。支持声明与实际旧系统验证是不同事项，见[支持范围](native-macos-support.md)。
- 17 个 Swift 产品文件共 5,419 行，其中 Models 1,413 行、LedgerStore 1,073 行、BridgeClient 237 行。切换到 MyGo 可能删除一部分 ABI/Swift DTO，但不能删除其中承载的语义校验、请求代次、取消、快照绑定和关闭释放规则。[客户端](../apps/macos/src/BridgeClient.swift)、[Store](../apps/macos/src/LedgerStore.swift)、[模型](../apps/macos/src/Models.swift)。行数是当前规模观察，不是工期估算。
- 可以直接复用 `internal/ledger`、`snapshot`、`report`、`query`、`analysis` 和 `desktop.Session`。Session 已负责只读会话、快照、查询取消及报表投影；MyGo 需要新的每窗口 adapter，而非重写会计引擎。[会话](../internal/desktop/session.go)、[查询](../internal/desktop/query.go)。
- 现有 macOS 包含资产负债表、损益、流水、趋势、账户、持仓、价格和查询工作台；查询编辑/结果分别用 NSTextView/NSTableView，趋势用 Swift Charts。[窗口与导航](../apps/macos/src/main.swift)、[查询控件](../apps/macos/src/QueryView.swift)、[图表](../apps/macos/src/DashboardView.swift)。这些能力需要功能等价迁移，不能由框架的 Counter 示例替代。
- 保存操作已有捕获字节、源路径/inode保护、目录 descriptor 锚定和原子替换；附件已有 roots、文件身份复核、Finder动作及普通 bookmark恢复。MyGo 的保存面板只解决选择路径，不自动替代这些规则。[导出写入](../apps/macos/src/AnalysisFileExport.swift)、[目录偏好](../apps/macos/src/AttachmentRootPreferences.swift)。现有 bookmark 不是 security-scoped 授权，不能把未来沙盒需求描述成当前已实现能力。
- 根 Go module 当前 `go 1.22`、无第三方依赖。试点宜放在独立 desktop module，例如 module path `orangecount/apps/mygo`，依赖/replace 核心 module；本次独立 module 已成功导入内部包，能够隔离 MyGo 的 Go 1.27.1 与第三方依赖，不必为研究更改整个核心依赖策略。[根 go.mod](../go.mod)。

## 上游证据基线

- 固定源码：[`869768d02d9154d5b2f9f3a6ebb39767f8f96983`](https://github.com/egoist/mygo/commit/869768d02d9154d5b2f9f3a6ebb39767f8f96983)，提交时间 2026-10-07 01:00:38 +0800，内容为原生 UI 内存占用与移除视图释放修复。本次下载目录为 `/tmp/orangecount-mygo-research-20261007`。
- 最新正式发布：[`v0.2.15`](https://github.com/egoist/mygo/releases/tag/v0.2.15)，2026-10-06 11:55:34 UTC，tag 指向 `da84fc54dfc532704f60d5a066c0d1025d4dd282`。研究 HEAD 仍自报 `Version = "0.2.15"`，但包含发布后的改动；试点应锁 commit，不能仅记录版本字符串。[Version 源码](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/mygo.go#L67)
- [GitHub 仓库元数据](https://api.github.com/repos/egoist/mygo)记录创建时间为 2026-09-26 15:21:37 UTC，距研究日约 11 天。频繁发布说明正在维护，不能据此推断已获得长期生产验证。
- [许可证](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/LICENSE)为 MIT；[go.mod](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/go.mod)要求 Go 1.27.1，直接依赖 purego、go-text/typesetting、x/image。MyGo 的无 cgo 是通过运行时动态调用系统 API 实现，并不意味着没有平台桥或第三方依赖。[构建实现](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/cmd/mygo/proc.go#L53)

**明确差异：最低 macOS 版本不能照抄 MyGo 文档。** MyGo 的[发行文档](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/docs/distribution.md#info-plist-and-entitlements)和打包配置默认写 `12.0`；但 [Go 1.27 官方发行说明](https://go.dev/doc/go1.27#darwin)明确要求 macOS 13 Ventura 或更新版本。因此在当前工具链要求下，应按 macOS 13+ 评估和声明支持，plist 的数值不能降低 Go runtime 的支持下限。

## 两条 UI 路线的实际含义

| 路线 | 源码确认的行为 | 对迁移的含义 |
|---|---|---|
| WebView | macOS 使用系统 WKWebView，网页经生成的 TypeScript 客户端调用 Go，也能由自定义 URL scheme 直接接收 Go HTTP handler 的响应 | 可复用 Web 页面和 HTTP API；仍存在 WebKit 的网页进程及 Web 技术边界 |
| `ui` | Go 构建每一帧的元素树，MyGo 自己布局和绘制；macOS 使用自定义 NSView、Metal及 CPU 局部重绘、Core Text，并自行桥接输入法与无障碍 | 可以直接调用 Go 业务层，消除 Swift/Go ABI；现有 SwiftUI 控件、Charts及其状态管理需要重写 |

依据：[ui 包定义](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/ui/doc.go)、[macOS surface](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/darwin/surface.go#L134)、[Metal renderer 选择](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/ui/gpu_darwin.go)、[渲染说明](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/docs/ui/rendering.md)。

这里的 native UI 是无网页的本地自绘 UI，并不是 Go 驱动 SwiftUI 或一组 NSTableView、NSOutlineView、NSTextField。窗口、系统菜单和文件对话框确实调用 AppKit；窗口内部的大多数 `ui` 控件由框架绘制。同一应用可混合两种窗口；[Content 窗口无 Page，且忽略 URL](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/docs/ui/windows.md)，因此不能假设现有 Web 图表可以直接嵌入 `ui` 元素树。

## 与账本应用有关的能力核实

| 能力 | 当前固定源码的事实 | 尚需实测的边界 |
|---|---|---|
| 交易表格 | `ui.Table` 有可见行虚拟化、可变行高、多选、键盘选择、排序状态、列宽调整/拖动、横向滚动和行内重命名；排序数据本身由应用处理 | 实际账本复杂单元格、长中文摘要、选择状态和大数据量下的滚动表现；不等同于 NSTableView 的成熟度。[表格源码](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/ui/table.go#L111) |
| 账户树 | 有 `Tree`、虚拟化 `Outline` 和 `OutlineTable`，支持折叠、键盘展开、层级和选择；账户余额/排序等领域规则仍由应用实现 | 实际账户深度、过滤后的定位和切换账本时的展开状态。[Outline 源码](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/ui/outline.go) |
| Charts | 本次检查 `ui`、`docs/ui` 和 examples 未发现现成 Chart/Plot 控件；具备 Painter、Path、文本、填充和事件基础 | `ui` 路线须另做坐标轴、刻度、图例、提示、交互与无障碍，或引入/实现图表组件。不能把绘图 API 当成 Swift Charts 的替代品。[绘图 API](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/docs/ui/drawing.md) |
| 中文输入 | `MyGoSurfaceView` 实现 NSTextInputClient，处理 UTF-16/rune转换、marked text、替换范围和候选位置；有 `ni`→`你`、组合期间 Enter/Escape 不误提交/关闭的 e2e 测试 | 上游测试通过原生 test hooks 调组合接口，不能替代完整系统拼音输入器的候选翻页、拼音混输、表格编辑和不同输入法验收。[输入桥](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/darwin/surface_input.go)、[测试及钩子](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/e2e/e2e_test.go#L2474)、[Apple 协议](https://developer.apple.com/documentation/appkit/nstextinputclient) |
| 无障碍 | 自建角色、标签、值、状态、表格/树行与操作树，通过 NSAccessibilityElement 桥接；有读取、按钮操作、编辑及虚拟列表测试 | 实际 VoiceOver 对复杂交易表格、错误提示和自绘 Charts 的浏览；应用仍须正确提供 Label/Role/Announce。[macOS AX 实现](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/darwin/access.go)、[使用说明](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/docs/ui/accessibility.md) |
| 多窗口/菜单/快捷键 | 窗口注册、父子窗口、关闭取消、系统菜单、原生上下文菜单及快捷键均有实现；`ui.Content` 可供多个窗口，各自保存元素状态 | 每窗口账本/请求生命周期、共享业务状态、关闭后回调和菜单焦点归属仍由应用设计。[窗口生命周期说明](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/docs/architecture.md#windows-lifecycle-and-quitting) |
| 打开/保存/Finder | NSOpenPanel、NSSavePanel、NSWorkspace 打开/显示文件、移入废纸篓、文件拖放和 `application:openURLs:` 均有实现；Info.plist 文件关联配合 OnOpenFile | 不自动提供 NSDocument 的文档生命周期。源码未发现 security-scoped bookmark API；若未来需要沙盒持久目录授权须单独设计。[面板](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/darwin/dialog.go#L96)、[Finder API](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/darwin/system.go#L102)、[文件打开事件](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/app.go#L461) |
| 签名/公证/打包 | CLI 有 .app、DMG、arm64/amd64 universal、codesign hardened runtime+timestamp、notarytool submit与stapler调用；正式签名和DMG须在 macOS 做 | 必须用 OrangeCount 的证书/entitlements/真实发行包验证；默认 ad hoc 不代表可面向其他用户发行。[签名实现](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/cmd/mygo/bundle.go#L189)、[公证实现](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/cmd/mygo/dmg.go)、[Apple 要求](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution) |
| 自动更新 | 实现 HTTPS下载、Ed25519验签、整包/差量更新、失败回退、替换 .app，以及 Web/Go `ui` 更新窗口插件 | 是 MyGo 自己的更新器，并不是集成 Sparkle。安装目录必须可写；源码在 app 旁建临时目录并用 rename 替换，无需自动推断具备管理员提权更新。[Updater](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/updater.go#L125)、[签名](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/update/update.go#L82)、[更新器说明](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/docs/plugins/updater.md) |

上游存在测试与三平台 CI 配置，[CI 源码](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/.github/workflows/ci.yml)会执行构建和测试；这属于继承的验证设计证据，不等同于本次观察到这些任务通过，也不等同于 OrangeCount 的验收。

## HTTP 和 IPC 对接约束

**HTTP handler 复用在接口上可行，且无须监听端口。** `Protocol.Handle("mygo", handler)` 覆盖默认前端服务；scheme request 在线程外构建 `*http.Request` 后直接调用 `handler.ServeHTTP`，支持 streamed response 和 Flush。macOS 使用 WKURLSchemeHandler 交付资源。依据：[protocol.go](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/protocol.go#L32)、[Darwin scheme bridge](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/darwin/scheme.go)、[Apple 协议](https://developer.apple.com/documentation/webkit/wkurlschemehandler)。

本次已实际复用 OrangeCount handler 的读取路由（结果见下节）。写入接口的 Origin/Referer、CSP、绝对 URL、WASM资源和流式请求仍须在完整工作台中验证；现有 `requireSameOrigin` 按 Host 判断，custom scheme 的实际 Origin 不能直接继承浏览器 HTTP 测试结论。[现有来源检查](../internal/web/http_helpers.go)。MyGo 设置 `RequestURI = URL.RequestURI()`，`RemoteAddr = "mygo:" + windowID`；注册 scheme 的页面默认获准调用绑定服务，不能把原本的 loopback HTTP 信任边界原样照搬。注册表是全局 scheme→handler，窗口 ID 通过 RemoteAddr 提供；多账本窗口须明确分配会话，不能将一个全局 Server 当成每窗口隔离。[请求构造](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/protocol.go#L149)、[页面信任判断](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/window.go#L974)

**TypeScript IPC 不保证 64 位整数无损。** Go `int64`/`uint64` 被生成器归为 TS `number`，结果使用普通 JSON 数字；未发现 bigint 映射或自动安全整数保护。若直接暴露超过 `2^53-1` 的 revision/request_id/原始整数，存在精度风险；这不是已经观察到 OrangeCount 金额失真。现有 `PresentedDecimal.exact` 是字符串，本次实测仍保持 `123.45`，生成的 State.revision 则是 TS number。结构字段 `json:",string"` 会生成 TS `string`，也可以设计显式字符串 DTO；不要直接把所有内部结构绑定给页面。原生 `ui` 直接调用 Go，不引入这一 JS 数字边界。[数字类型分类](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/tsgen/types.go#L54)、[生成规则及 string tag](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/tsgen/generate.go#L274)、[JSON 编码](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/ipc.go#L373)

## 线程、生命周期与明确限制

`mygo` 的 init 锁定主 goroutine，`App.Run` 必须从 main 调用。窗口事件和 `ui` view 在主线程同步执行，慢任务需转 goroutine；bound service调用与HTTP handler由独立 goroutine执行。`ui.Context`/Element只在该帧有效，后台结果通过异步 `Window.Update` 交还主线程，不能直接并发修改界面读取的状态。重排/删除列表需要稳定 Key。需要运行中的原生 API 在 main 的 `App.Run` 之前调用会 panic；这些不是“全Go”就能省掉的限制。[线程公开约定](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/mygo.go#L45)、[view 规则](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/docs/ui/views.md)、[Update 实现](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/content.go#L141)

对 macOS 所需核心功能，本次检查没有发现只是公开空壳的表格、输入法、无障碍或文件面板。不过“跨平台 API”不保证每个选项处处实现：[平台差异表](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/docs/architecture.md#platform-differences)明确记录 Linux 的 content protection/click-through 等被忽略，Dock和traffic lights仅macOS；macOS [UnregisterURLScheme 明确返回 unsupported](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/darwin/deeplink.go#L31)，通知要求 app bundle，直接 `go run` 不能据此验收通知。[通知支持检查](https://github.com/egoist/mygo/blob/869768d02d9154d5b2f9f3a6ebb39767f8f96983/internal/darwin/notify.go#L66)

近期维护记录仍包含框架基础行为的修复：[v0.2.13](https://github.com/egoist/mygo/releases/tag/v0.2.13)修正 macOS resize/fullscreen 的 KVO stack overflow及组合输入的 Enter/Escape；其 `Context.OpenURL` 签名变更又在 [v0.2.14](https://github.com/egoist/mygo/releases/tag/v0.2.14)撤回；研究 HEAD修复原生 UI内存/视图释放。这是锁版本、控制试点范围的实际依据，不是据项目年龄直接判定不能使用。

## 本次实测及其范围

环境：macOS 27.0.1 / arm64，Go 1.27.1。上游使用上述固定 commit；本项目使用上述 clean HEAD。探针位于 `/tmp/orangecount-mygo-integration-20261007`，以独立 module 的 `replace` 引用本项目与上游，所有 Go 命令设置 `CGO_ENABLED=0`。输入是临时生成的公开示例账本，一笔 `123.45 USD` 交易；没有使用私人账本。

| 当前执行 | 观察结果 | 能证明的范围 |
|---|---|---|
| 上游 `go test ./ui ./examples/counter-native` | 两包通过，exit 0 | 当前源码的 ui/headless 与 Counter 测试；没有运行上游全量平台 e2e |
| 上游 `go build -o …/counter-native ./examples/counter-native` | exit 0 | 当前 macOS 上无 cgo 原生示例可构建；未运行 Metal 原生窗口验收 |
| 独立 module `go run .` | exit 0：有效账本、balance_sheet保留exact、查询成功、生成TS client 13,671 bytes、headless UI金额文本存在、关闭后ErrClosed | 现有 Go业务包可直接用于MyGo；不包含整个 LedgerStore 时序/导出/附件迁移 |
| 同一探针的现有Handler请求 | `/`、`/app.js`、`/api/v1/status`、`/api/v1/reports/balance-sheet` 在 `mygo://localhost` GET请求下均200 | 现有只读路由和入口资源兼容这种请求构造；未完整操作工作台 |
| 独立 module `go run . -runtime` | 实际隐藏WKWebView窗口，两个真实fetch均200，IPC读取State返回valid=true，正常退出 | 自定义协议在真实WKWebView中可读取现有API，并可调用绑定Go服务；不只是接口推断 |

实际运行输出：

```text
MyGO=0.2.15 engine_valid=true exact_decimal=true query=true native_headless=true typescript_bytes=13671 handler_routes=4 no_listener=true
WKWebView_runtime={"codes":[200,200],"scheme":"mygo:","valid":true}
```

探针没有调用 `Server.Serve` 或 `net.Listen`，而是把 `Server.Handler()` 注册给 `mygo.Protocol.Handle`。App 使用隐藏窗口及 prohibited activation policy，userData/cache 指向临时目录，运行设置25秒超时并主动退出。这是代码路径与实际请求证据；没有另做系统端口枚举或所有外部网络行为审计。

探针 `main.go` SHA-256：`6fabf2180480626e09fe4d3f576f3b83c231fdcdd558cff70972269fd955164f`；`go.mod` SHA-256：`09efd67c3931d68508e9c3de594fc0c6aeae2bcc511e5cf9cd75f689347817a4`。临时目录不作为永久研究附件；清理后需按本文的模块/路由/公开输入重新建立探针。

没有重跑现有 SwiftUI 的 `make mac-check` 或已有私人对照，因为产品代码未变；历史原生通过记录只作为功能基线，见[0.1.6交付证据](native-development-release-0.1.6.md)，不作为本次迁移成功的证据。没有性能对比、真人中文输入/VoiceOver、旧macOS/Intel、完整多窗口或签名公证验收。

## 迁移选择与建议

| 选择 | 主要收益 | 成本与适用判断 |
|---|---|---|
| 继续 SwiftUI macOS 主版本 | 保留当前 AppKit/Charts和已实现工作流；继续完成未验项 | 若目标是更好的macOS体验，这是当前建议；不会自动得到跨平台客户端 |
| MyGo WebView桌面客户端 | 复用已有 TS/Svelte/Web工作台和Go Handler；减少重复页面建设；免监听端口已实测 | 跨平台与Web功能统一时最值得试点。仍须做多窗口、读写范围、来源策略与桌面文件动作adapter；WebKit子进程要计入性能和架构判断 |
| MyGo `ui`重写 | Go直接调用Session，移除Swift/C ABI，统一桌面语言，保留无WebView路线 | 用户界面、状态机、图表和平台细节需要迁移；图表组件及成熟度是目前主要代价。只有试点收益明确后才建议替换主版本 |

**建议下一步是一个可丢弃的并行试点，并保留原生交付基线。** 固定本次上游commit，使用独立 module，复用会计引擎；不要顺带迁移会计语义、存储或作者写入规则。如果方向是纯Go/无WebView，试点应优先挑战 `ui` 的复杂表格、账户树和一张真实财务图表；如果方向是减少双端重复开发、统一Web工作台，则优先WebView。两条路线不需要同时全面实现。

| 阶段 | 可审查产物与通过条件 |
|---|---|
| 1：桌面接入 | 打开公开账本；一个窗口一个Session；资产负债表/分页流水精确值一致；双窗口不同账本不会串数据；关闭后取消任务并释放Session |
| 2：高成本交互 | `ui`试点完成中文查询编辑、动态查询表格、多选复制、虚拟账户树和财务图表交互/AX；WebView试点完成实际现有工作台、资源/CSP、来源检查和系统打开/保存/Finder动作。两者都验证重载错误保留旧快照、迟到响应拒绝及导出保护 |
| 3：同输入比较与打包 | 同机器同账本对比现有SwiftUI的首次可用时间、全部进程空闲RSS、交易滚动响应、查询取消及连续开关窗口；检查真实Mach-O最低系统、旧系统/目标架构，并完成签名公证/安装试验 |
| 4：迁移决策 | 功能和数据正确性无回退，中文/VoiceOver通过，内存和响应无未解释退化，或明确得到可接受的跨平台/维护收益，才将替换列入实施计划 |

此处是后续试点建议，尚未执行阶段1–4的完整验收，也不提供没有工程分解依据的工期承诺。
