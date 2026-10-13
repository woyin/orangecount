# G45 旧 Swift/C 验证能力迁移预审

日期：2026-10-08（Asia/Singapore）。这是基于当前 worktree 的迁移盘点，不是 G45 完成证明。G43 开发包与 GUI 等价门槛尚未通过，因此本记录不删除旧源码、测试、产物或入口，也不把 headless 测试标成原生 GUI 替代。

## 旧验证能力与当前替代

| 旧验证 | 当前 Go 替代及可复核入口 | 当前缺口与处置 |
|---|---|---|
| `ContractTests.swift`：桌面 DTO、报表、流水、query、directives、附件和来源字段通过 Swift/C bridge | 共享核心投影由 `internal/desktop` 和 `internal/query` 的测试直接验证；MyGo 页面响应见 `apps/goui/internal/app/model_test.go`（仪表盘、报表树、流水、查询、指令、诊断、价格、持仓等）；实际 `DesktopEngine` 生命周期和 query 错误见 `apps/goui/internal/workspace/controller_test.go` | Go UI 不再经过 C archive/JSON/Swift 解码边界，旧 wire protocol 专属的 malformed-payload 检查没有同构目标；其余业务断言仍需逐项核对。页面测试使用 MyGo headless Tester，不能替代真实窗口、VoiceOver 或 IME 验收。旧测试保留到 G43/G38 完成后再裁决。 |
| `StateTests.swift`：Swift actor 的 query/export 排队、取消、晚到结果和关闭 | `internal/desktop/query_test.go`；`apps/goui/internal/workspace/controller_test.go` 的 query/capture 取消、晚到结果丢弃、关闭与窗口隔离；`apps/goui/internal/workspace/lifecycle_soak_test.go` 的 20 次 workspace 打开/关闭和 60 次 reload | 语义已迁至 Go Controller/Session。workspace 测试不表示 MyGo 原生窗口 timer、sheet 或进程资源已覆盖；G40 原生 soak 未完成。 |
| `ExportTests.swift`：原子替换、source/symlink/hardlink/inode 保护、部分写入失败 | `apps/goui/internal/localfiles/save_darwin_test.go` 的冻结 bytes、目标身份保护、部分写失败和临提交校验；`apps/goui/internal/app/query_export_test.go`、`report_export_test.go` 的 capture 到保存闭环；归档读取见 `internal/desktop/query_analysis_test.go` | Go/Darwin adapter 有直接测试；跨窗口 NSSavePanel、VoiceOver 和完整真实 GUI 保存动作仍属于 G37/G38/G43。旧 Swift writer 暂不删除。 |
| `NativeClientTests.swift`：SwiftUI 导航和用户流程（query、预设、指令、附件 roots、保存、恢复、价格、持仓） | `apps/goui/internal/app/model_test.go` 的页面/导航渲染；`apps/goui/internal/workspace/controller_test.go` 的真实 DesktopEngine 状态；`apps/goui/internal/preferences/store_test.go`、`querypresets/store_test.go` 的重启/隔离迁移；Darwin bookmark、来源和保存 adapter 测试 | Go UI 对核心和 headless 页面有较广覆盖，但 Swift 测试驱动 `LedgerStore`，MyGo 测试驱动的是独立 Model/Tester；尚无完整原生窗口等价矩阵。G38 完成前不得删除旧端到端测试。 |
| `PrivateReadProbe.swift` / `apps/macos/private-read.sh`：外部账本的只读合同与准确期望值 | `apps/goui/private-read.sh` 和 `apps/goui/internal/app/private_read_test.go`；G41 harness 对同一 snapshot 的 core/workspace/UI/query/export 结果进行对照 | 新入口要求绝对外部路径且不记录账本内容。当前只用公开 fixture 验证 harness；真实私人账本对照未运行，旧 probe 在此之前仍是可撤回参照。 |
| `ClientLifecycleProbe.swift`、`SessionBenchmark.swift`、`ClientBurstBenchmark.swift`：旧客户端 session 周期、bridge/JSON 与 query burst 测量 | `apps/goui/internal/workspace/lifecycle_soak_test.go`；`apps/goui/internal/app/benchmark_test.go` 与 `apps/goui/benchmark.sh` 的核心打开、headless 页面和图表基准 | Go 直接调用核心，测量边界不同；已有结果不能作为原生 Metal、SwiftUI 同输入、输入到 present 或 10 分钟 RSS 对照。旧 probe 保留至 G39/G40 获得可比原生结果。 |

## Bridge 与入口保留判断

- `internal/bridge` 的导出 `OC_*` 函数由 `apps/macos/src/BridgeClient.swift` 调用；根 `Makefile` 的 `app`、`mac-test`、`mac-package` 仍构建旧 Swift/C 客户端。当前这些源码仍是可构建的旧应用和回退路径，不可随 MyGo 开发包一并删除。
- `apps/goui/third_party/mygo/internal/bridge` 是 vendor 的 `github.com/egoist/mygo/internal/bridge`，不等于仓库的 `internal/bridge`。MyGo 的 Darwin/Linux/Windows 窗口后端依赖 vendor 包；不能按同名路径误删。
- 默认 `make all` 仍构建仓库 Go 命令行程序，`make app` 仍是旧 macOS App，MyGo 入口是显式的 `make goui-*`。将默认 macOS 入口切换到 MyGo 属于 G46，需等 G43/G45 的回退、偏好和包验证成立后执行。

## 可执行退役顺序

1. 先完成 G37/G38，确认原生菜单、输入法、辅助功能、文件面板、导航、query/export 与长文本等行为；headless 测试不能满足此步。
2. 用同一公开与私人只读输入补齐 G41；私人期望文件留在仓库外，确认日志不泄露账本路径和值。
3. 完成 G39/G40 的 MyGo 原生进程/窗口性能与资源证据，再完成 G43 的仓库外解压启动和阅读、query、export、恢复闭环。
4. 之后才从 `apps/macos/tests` 按能力删除已有 Go+原生 GUI 替代且有源码身份/可恢复包的测试；Swift/C 文件只在旧应用不再是支持入口时另行评估。
5. `internal/bridge` 仅在没有旧 app/build/test/private-probe 引用、且用户回退路径已验证后才允许删除。G44 的签名发行是另一道正式发行门槛，不可由本地 development 包替代。

## 盘点边界

本次检查的是当前文件引用、测试入口和 Go 测试定义；没有运行旧 Swift/C suite，也没有真实 macOS GUI、私人账本或发布签名验收。G45 仍未通过，依赖 G43 的退役动作保持待定。
