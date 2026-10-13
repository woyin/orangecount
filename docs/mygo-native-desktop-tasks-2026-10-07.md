# OrangeCount MyGO Go ui 任务台账

日期：2026-10-08。实现已从 P0 骨架推进到工作区控制器、报表搜索与账户树初版、查询工作台/目录初版、偏好/bookmark 迁移、语义配置预览与应用初版、全指令阅读初版、附件目录策略/声明动作、账户详情、商品目录与价格历史、持仓与成本初版、查询 typed results/离线捕获阅读、安全文件保存及查询/报表保存首轮。当前 18 项已验证、24 项进行中、1 项已实现待验证、3 项未开始；其中 G01–G03、G05–G06、G08、G10–G11、G14、G17–G19、G26、G29–G33 已验证，P0 阶段仍未退出。尚未完成真实 IME/VoiceOver 与全功能 GUI 等价验收、原生进程长时性能/资源 soak，以及完整开发安装交付；G41已有生产最近账本的私有只读同源对照结果，但仍依赖未完成的 G38 全 GUI gate。状态以任务表为准，任务卡中的进度说明补充实现与证据，不另存第二份状态。证据见[P0 开工记录](evidence/goui-p0-2026-10-07.md)、[G05 面板与来源打开首轮记录](evidence/goui-p0-g05-2026-10-07.md)、[G06 旧偏好端到端迁移验证](evidence/goui-p0-g06-legacy-migration-2026-10-08.md)、[G07 图表键盘选择与精确值记录](evidence/goui-p0-g07-2026-10-07.md)、[G04 查询错误定位记录](evidence/goui-p0-g04-2026-10-08.md)、[P1 工作区/报表实现记录](evidence/goui-p1-2026-10-07.md)、[G10 原生筛选与导航记录](evidence/goui-p1-g10-filter-navigation-2026-10-08.md)、[G12 列布局偏好记录](evidence/goui-p1-g12-2026-10-07.md)、[G12诊断语言偏好记录](evidence/goui-p1-g12-diagnostic-locale-2026-10-07.md)、[G12偏好备份导出记录](evidence/goui-p1-g12-backup-2026-10-08.md)、[P2 指令阅读记录](evidence/goui-p2-2026-10-07.md)、[G13导航栈回归](evidence/goui-p2-g13-navigation-2026-10-08.md)、[G14 语义配置记录](evidence/goui-p2-g14-2026-10-07.md)、[G15 账户树记录](evidence/goui-p2-g15-2026-10-07.md)、[G15账户树与金额控件记录](evidence/goui-p2-g15-accounttree-2026-10-08.md)、[G15/G16账户树搜索与重载证据](evidence/goui-p2-g15-accounttree-reload-2026-10-08.md)、[G16报表搜索记录](evidence/goui-p2-g16-2026-10-07.md)、[G16自身余额与折算估值记录](evidence/goui-p2-g16-own-subtree-valuation-2026-10-08.md)、[G19外部来源编辑器设置](evidence/goui-p2-g19-source-editor-2026-10-08.md)、[G21账户详情记录](evidence/goui-p3-g21-2026-10-07.md)、[G22持仓与成本记录](evidence/goui-p3-g22-2026-10-07.md)、[G23商品与价格记录](evidence/goui-p3-g23-2026-10-07.md)、[G27类型化查询结果记录](evidence/goui-p4-g27-2026-10-07.md)、[G28查询目录记录](evidence/goui-p4-g28-2026-10-07.md)、[G28多账本与重载原生窗口验证](evidence/goui-p4-g28-native-e2e-2026-10-08.md)、[G29离线分析阅读记录](evidence/goui-p4-g29-2026-10-07.md)、[G30 固定捕获协调验证](evidence/goui-p5-g30-2026-10-07.md)、[G31 安全文件保存模块验证](evidence/goui-p5-g31-2026-10-07.md)、[G32 报表保存首轮记录](evidence/goui-p5-g32-2026-10-07.md)、[G33 查询保存首轮记录](evidence/goui-p5-g33-2026-10-07.md)、[G36 菜单与焦点命令记录](evidence/goui-p6-g36-2026-10-07.md)、[G36 每账本字号与快捷键验证](evidence/goui-p6-g36-text-scale-2026-10-08.md)、[G37/G38 原生 MyGo 窗口端到端检查](evidence/goui-p6-g37-g38-native-e2e-2026-10-08.md)、[G38连续错误与修复原生验证](evidence/goui-p6-g38-invalid-updates-native-2026-10-08.md)、[G38 include 删除与替换原生验证](evidence/goui-p6-g38-include-replacement-native-2026-10-08.md)、[G38 初次打开无效账本原生验证](evidence/goui-p6-g38-initial-invalid-native-2026-10-08.md)、[G38账户树方向键导航验证](evidence/goui-p6-g38-accounttree-keyboard-native-2026-10-08.md)、[G38双账本隔离验证](evidence/goui-p6-g38-multiwindow-isolation-2026-10-08.md)、[G21/G22/G23原生页面补充](evidence/goui-p6-g21-g23-native-pages-2026-10-08.md)、[G39 性能基准记录](evidence/goui-p6-g39-performance-2026-10-08.md)、[G40 workspace 生命周期 soak 首轮记录](evidence/goui-p6-g40-2026-10-08.md)、[G41 Go UI只读对照harness记录](evidence/goui-p6-g41-harness-2026-10-08.md)、[G42构建与包记录](evidence/goui-p7-g42-2026-10-07.md)、[G42开发包完整性校验](evidence/goui-p7-g42-package-verifier-2026-10-08.md)、[G42/G43开发包启动验证](evidence/goui-p7-g42-g43-2026-10-08.md)、[G42/G43当前源码包校验](evidence/goui-p7-g42-g43-current-package-2026-10-08.md)、[G43 r6 隔离开发 GUI 闭环](evidence/goui-p7-g43-r6-isolated-development-gui-2026-10-08.md)和[P4 查询工作台初版记录](evidence/goui-p4-2026-10-07.md)。

G43 后续验收记录：[持仓筛选修复包隔离 GUI 尝试](evidence/goui-p7-g43-holdings-fix-gui-attempt-2026-10-08.md)、[无效 Legacy 恢复与持仓筛选包可见验收](evidence/goui-p7-g43-semantics-recovery-visible-2026-10-08.md)、[修复包查询/报表导出与离线读回](evidence/goui-p7-g43-semantics-recovery-export-loop-2026-10-08.md)、[修复包开发/QA偏好隔离](evidence/goui-p7-g43-semantics-recovery-dev-qa-isolation-2026-10-08.md)。G14 自动化与原生证据：[初次 Legacy 无效恢复](evidence/goui-p2-g14-invalid-legacy-recovery-2026-10-08.md)。

G45 依赖 G43 的源码退役尚未开始；当前有一份仅用于排后续顺序的[旧 Swift/C 验证能力迁移预审](evidence/goui-p7-g45-retirement-preflight-2026-10-08.md)，不代表旧入口或 bridge 可以删除。

依据：[详细设计](mygo-native-desktop-design-2026-10-07.md)的R01–R15与[实施计划](mygo-native-desktop-plan-2026-10-07.md)的P0–P7。任务卡路径是责任范围；当前实现进度以本表和阶段证据为准，后续模块按依赖逐步建立。阶段证据写入`docs/evidence/goui-pN-<date>.md`。

## 1. 状态、依赖与责任

状态枚举：未开始、进行中、已实现待验证、已验证、外部条件未具备。只有相应完成证据和验收通过后才更新为已验证；阶段/正式发行还需满足计划gate。职责是实施时的文件责任范围，共享文件改动应由同一责任人协调，不回滚他人工作。

| ID | 阶段 | 任务 | 依赖 | 责任 | 状态 |
|---|---|---|---|---|---|
| G01 | P0 | [冻结迁移行为与同输入基线](#g01) | — | 工程与验收 | 已验证 |
| G02 | P0 | [建立独立Go桌面module与固定依赖](#g02) | G01 | 构建 | 已验证 |
| G03 | P0 | [可运行Go ui骨架与隔离QA入口](#g03) | G02 | 应用骨架 | 已验证 |
| G04 | P0 | [公开文本选区与错误定位缺口](#g04) | G03 | 编辑器与上游适配 | 进行中 |
| G05 | P0 | [macOS面板、取消与来源编辑器](#g05) | G03 | 系统集成 | 已验证 |
| G06 | P0 | [普通bookmark与旧偏好读取验证](#g06) | G05 | 平台与偏好 | 已验证 |
| G07 | P0 | [财务绘图路径前置验证](#g07) | G03 | 图表 | 进行中 |
| G08 | P1 | [工作区状态机与资源所有权](#g08) | G03 | 工作区 | 已验证 |
| G09 | P1 | [打开、轮询、重载与双窗口](#g09) | G08,G05 | 工作区与窗口 | 进行中 |
| G10 | P1 | [有界读取与快照发布](#g10) | G09 | 工作区 | 已验证 |
| G11 | P1 | [查询及捕获取消快速路径](#g11) | G08,G09 | 工作区 | 已验证 |
| G12 | P1 | [版本化偏好与迁移基础](#g12) | G06,G08 | 偏好 | 进行中 |
| G13 | P2 | [界面壳、导航与阅读控制](#g13) | G10,G12 | 界面 | 进行中 |
| G14 | P2 | [显式profile预览与应用](#g14) | G13 | 工作区与界面 | 已验证 |
| G15 | P2 | [稳定表格、账户树和精确值控件](#g15) | G13 | 基础界面 | 进行中 |
| G16 | P2 | [资产负债表与损益表](#g16) | G15,G14 | 报表界面 | 进行中 |
| G17 | P2 | [分页流水与交易详情](#g17) | G15 | 流水界面 | 已验证 |
| G18 | P2 | [全指令和配置阅读](#g18) | G17 | 流水界面 | 已验证 |
| G19 | P2 | [诊断、错误恢复与来源动作](#g19) | G09,G05,G13 | 诊断界面 | 已验证 |
| G20 | P3 | [仪表盘与现金流/净资产](#g20) | G24,G16 | 分析界面 | 进行中 |
| G21 | P3 | [账户详情与返回导航](#g21) | G24,G16,G17 | 分析界面 | 进行中 |
| G22 | P3 | [持仓、收益、事件与lot详情](#g22) | G24,G15,G14 | 分析界面 | 进行中 |
| G23 | P3 | [商品目录与价格历史](#g23) | G24,G15 | 分析界面 | 进行中 |
| G24 | P3 | [完整可复用财务图表Module](#g24) | G07,G15 | 图表 | 进行中 |
| G25 | P4 | [生产查询编辑器交互](#g25) | G04,G13 | 查询界面 | 进行中 |
| G26 | P4 | [查询工作区与执行上下文](#g26) | G25,G11 | 查询界面与工作区 | 已验证 |
| G27 | P4 | [类型化结果表格、复合详情与复制](#g27) | G26,G15 | 查询结果界面 | 进行中 |
| G28 | P4 | [保存query目录与预设](#g28) | G26 | 查询界面 | 进行中 |
| G29 | P4 | [离线分析文件阅读](#g29) | G27 | 查询捕获阅读 | 已验证 |
| G30 | P5 | [统一固定捕获协调](#g30) | G10,G11 | 工作区导出 | 已验证 |
| G31 | P5 | [Go安全文件保存Module](#g31) | G30,G05 | 本地文件 | 已验证 |
| G32 | P5 | [全部适用报表导出](#g32) | G31,G16,G17,G20,G21,G22,G23 | 报表导出界面 | 已验证 |
| G33 | P5 | [查询成功结果保存](#g33) | G31,G27,G28 | 查询导出界面 | 已验证 |
| G34 | P5 | [附件声明到Open/Finder闭环](#g34) | G18,G05 | 附件界面 | 进行中 |
| G35 | P5 | [附件目录管理、恢复与共享撤销](#g35) | G34,G06,G12 | 附件与偏好 | 进行中 |
| G36 | P6 | [全页面文案、快捷键与命令焦点](#g36) | G19,G20,G21,G22,G23,G28,G29,G32,G33,G35 | 应用与界面整合 | 已实现待验证 |
| G37 | P6 | [真实输入法和VoiceOver验收](#g37) | G36,G04 | 系统交互验收 | 进行中 |
| G38 | P6 | [新Go ui全功能GUI与等价矩阵](#g38) | G36 | 产品验收 | 进行中 |
| G39 | P6 | [性能基准、帧与空闲核验](#g39) | G36,G01 | 性能验收 | 进行中 |
| G40 | P6 | [资源释放与持续重载观察](#g40) | G38,G08 | 生命周期验收 | 进行中 |
| G41 | P6 | [当前核心与新UI私人只读对照](#g41) | G38 | 等价验收 | 进行中 |
| G42 | P7 | [生产构建、许可证与独立CI](#g42) | G02,G36 | 构建发行 | 进行中 |
| G43 | P7 | [完整开发包与安装交付](#g43) | G37,G38,G39,G40,G41,G42 | 交付 | 进行中 |
| G44 | P7 | [正式目标系统与签名发行门槛](#g44) | G43 | 发行验收 | 未开始 |
| G45 | P7 | [旧Swift/C验证能力迁移和退役清单](#g45) | G43 | 迁移收尾 | 未开始 |
| G46 | P7 | [默认入口接管、迁移与可撤回交付](#g46) | G45 | 产品接管 | 未开始 |

## 2. 可执行任务卡

每张卡给出修改位置、实施内容、完成证据与设计要求。除G44所需外部发行条件外，先完成可独立进行的代码与验证；不要为了后续缺设备/证书跳过当前任务。任务中“通过”均是未来完成条件，不是本轮已执行结果。

### P0 — 工程起点与高成本验证

<a id="g01"></a>

#### G01 — 冻结迁移行为与同输入基线

**依赖：** —。**对应要求：** R02,R06,R13,R15。**职责：** 工程与验收。

**修改位置：** docs/evidence/goui-p0-<date>.md；公开fixture/既有apps/macos验收脚本。

**实施：** 记录真实HEAD/dirty diff、MyGO固定修订、工具链；将旧18项GUI场景及新增查询/附件/导出能力映射到新R01–R15；复制公开fixture而非私人数据；取得适用SwiftUI baseline。

**完成证据：** 保存输入hash、机器、prodflags和当前观察；每条历史证据标识来源与适用范围；缺失baseline有具体原因，不借用旧数字。

<a id="g02"></a>

#### G02 — 建立独立Go桌面module与固定依赖

**依赖：** G01。**对应要求：** R01,R14。**职责：** 构建。

**修改位置：** apps/goui/go.mod、go.sum；apps/goui/build.sh、test.sh；Makefile。

**实施：** module orangecount/apps/goui，replace核心到../..；固定MyGOcommit对应pseudo-version/Origin.Hash；GOWORK=off；新增基础goui-build/test；根不加go.work。

**完成证据：** 独立module可导入internal/desktop，CGO_ENABLED=0生产build通过；核心go.mod/go.sum策略和root license未被改变；module/hash记录可追溯。

**依赖升级评估：** `v0.3.6` 候选的只读评估见[MyGO v0.3.6 依赖升级评估](mygo-v036-upgrade-assessment-2026-10-12.md)。结论是当前不升级：`v0.3.6` 没有可取消保存面板、监听器释放与 E2E 可观察面，升级需要继续维护 fork。该评估属于本卡依赖范围的后续处理项，不代表 G02 状态变化。

<a id="g03"></a>

#### G03 — 可运行Go ui骨架与隔离QA入口

**依赖：** G02。**对应要求：** R01,R08,R11。**职责：** 应用骨架。

**修改位置：** apps/goui/cmd/orangecount-goui/main.go；internal/app；prepare-qa.sh；tests。

**实施：** main运行App.Run，建立Content窗口、欢迎页、虚拟Table验证页；开发/QA身份与userData隔离；ui.NewTester入口和真实GUI启动/退出路径。

**完成证据：** 真实Content窗口显示并退出，Page=nil；运行路径不创建WebView/Bind/服务器；QA重新启动仍不读取用户最近账本；C archive/Swift不参与生产链接。

2026-10-08 原生欢迎页补充：真实 MyGo Content 窗口预置隔离的最近账本后仍停留欢迎页，controller 与 Model 均无活动账本；通过 Accessibility 明确选择后才打开公开 fixture。定向连续5次通过，详见[G03欢迎页原生验收](evidence/goui-p0-g03-native-welcome-2026-10-08.md)。此证据未覆盖生产身份重启恢复或真人 VoiceOver。

<a id="g04"></a>

#### G04 — 公开文本选区与错误定位缺口

**依赖：** G03。**对应要求：** R07,R12。**职责：** 编辑器与上游适配。

**修改位置：** apps/goui/internal/widgets/queryeditor；上游patch manifest/最小fork（必要时）。

**实施：** 验证固定ui.TextArea选区能力；补公开选区设置/滚动可见Interface；UTF-8 byte→rune转换；明确forkcommit/许可证/退回上游条件，禁止访问私有editor。

**完成证据：** 真实短查询错误定位准确；中文/emoji/CRLF/组合字符/尾部range测试通过；选区设置不破坏undo、caret或marked text；有patch身份。

**当前进度：** 固定MyGo commit的本地fork已增加公开 `SetTextSelection`，rune范围会显示尾部caret；组合输入期间拒绝移动选区。fork保留MIT与commit provenance，MyGo `ui`全套测试及UTF-8字节→rune选择、undo、marked text、滚动测试通过。真实应用查询页已通过真实核心错误响应的Unicode定位测试；新增 macOS 原生窗口测试在 AppKit Native Input Client 中核对 emoji 前缀后的 UTF-16错误选区，并提交文本替换目标token，连续10次通过；组合期间保留文本的应用路径也有headless测试。当前源码隔离 QA 包又补做屏幕上错误 token 选区和替换的可见验证；真人输入法候选窗和 caret 的独立视觉验收仍待完成，故本任务仍进行中。证据见[G04记录](evidence/goui-p0-g04-2026-10-08.md)、[G04可见选区原生验证](evidence/goui-p0-g04-visible-selection-native-2026-10-08.md)及[G37/G38原生窗口记录](evidence/goui-p6-g37-g38-native-e2e-2026-10-08.md)。

2026-10-08 可见选区补充：当前源码隔离 QA 包对公开 cashflow fixture 执行带 emoji 的只读查询，真实窗口将 parser 错误的 `missing` 精确高亮；替换为 `account` 后只在用户再次执行时返回成功的0行类型化结果。隔离包启动不复现常驻旧 QA bundle 的未知偏好字段提示。系统输入源与快捷键仅作只读检查；CUA 按键仍未生成拼音 marked text/候选窗，故不计作 G37 真人 IME 通过。详见[G04可见选区原生验证](evidence/goui-p0-g04-visible-selection-native-2026-10-08.md)。

<a id="g05"></a>

#### G05 — macOS面板、取消与来源编辑器

**依赖：** G03。**对应要求：** R08,R10,R12。**职责：** 系统集成。

**修改位置：** apps/goui/internal/platform/source_darwin.go；apps/goui/third_party/mygo/dialog.go、internal/darwin/dialog.go；apps/goui/internal/app 的来源动作。

**实施：** 核实MyGO Dialog父窗口/可取消能力；不足时实现小型纯Go Darwin adapter；来源默认关联若为任一OrangeCount身份改用外部编辑器/Finder。

**完成证据：** 开两个窗口后面板归属正确，关窗没有孤立sheet；cancel有实际行为；bean关联自身时来源动作不会递归开新应用；无Swift/cgo/外部osascript运行依赖。

**当前进度：** `Dialog.SaveWithContext` 已提供带父窗口的可取消 Save Panel，Darwin 实现通过 main-thread cancel 消除孤立 sheet；查询/报表面板已接入。新增 Darwin `platform.OpenSource` 检查默认应用 bundle；OrangeCount 生产/QA/Go UI association 会改用 TextEdit，失败时显示 Finder；普通外部默认应用仍由 NSWorkspace 打开。账本/诊断/指令/流水/账户/商品与持仓来源处已加“打开源文件”，动作校验当前 revision/epoch/operation/snapshot。classifier 与迟到动作测试及全量构建通过。本机 LaunchServices 对公开 `.bean` fixture 的默认处理器是 OrangeCount QA bundle；从独立 QA 模式 Go UI 打开来源后，界面报告成功，TextEdit 实际显示该 fixture，没有递归启动 OrangeCount。原生 MyGo 窗口测试验证来源选择面板、报表/查询保存面板的窗口归属与取消，并验证关闭报表面板所属窗口后无孤立 sheet。另通过可见的 NSSavePanel 用户操作分别确认查询 CSV 和流水报表 CSV 保存到专用 `/tmp` 目录；界面显示成功，CSV 解析检查分别得到1条结果行和4条分录。Go UI 生产构建保持 `CGO_ENABLED=0`，未增加 Swift/cgo/osascript 运行依赖。证据见[G05面板与来源路由实机验收记录](evidence/goui-p0-g05-2026-10-07.md)。

<a id="g06"></a>

#### G06 — 普通bookmark与旧偏好读取验证

**依赖：** G05。**对应要求：** R09,R11。**职责：** 平台与偏好。

**修改位置：** apps/goui/internal/platform/bookmarks.go、bookmarks_darwin.go；apps/goui/internal/preferences/legacy_darwin.go。

**实施：** 用Darwin纯Go适配读取旧UserDefaults与普通NSURL bookmark；保持withoutUI/withoutMounting及stale/canonical/device/inode校验；定义只读迁移记录。

**完成证据：** 公开临时目录bookmark可保存/解析；删除/替换/失效不自动授权；旧域读取不改旧偏好；测试说明普通bookmark不等于security-scoped。

**当前进度：** 旧 macOS defaults 目前只读迁移最近账本、每账本语义 profile、币种、估值和符合旧 Codable 结构的附件 roots；验证后写入 Go 独立偏好文件并记录第二版一次性迁移标记。旧 UserDefaults 不被修改，QA 身份不读取旧域；新账本默认值已恢复为旧客户端的 `legacy`。purego/Objective-C reader 通过独立 UserDefaults suite 运行时测试，覆盖 NSData 读取；新增端到端 Darwin 测试从隔离 suite 读取旧格式 defaults，经 Go Store 迁移后解析真实临时目录 bookmark，并确认未知键保留、迁移幂等和旧值未被迁移过程改写。Darwin bookmark adapter 的普通 NSURL bookmark 创建/解析、withoutUI/withoutMounting、stale/canonical/device/inode 校验及临时目录往返、移动/替换、删除和损坏数据测试通过。适配器不请求 security scope。Go UI 已用 roots bookmark 恢复 Session policy；opt-in 只读测试在用户真实 `org.orangecount.desktop` defaults 域迁移出3条最近路径和3份账本偏好，核验3条现存记录，迁移前后最近路径、profile、reading 与对应 bookmark keys 均未变化；当前域内3条最近记录没有附件目录配置。隔离 suite 的真实临时 bookmark 端到端测试覆盖旧 JSON/NSData 格式及解析。G06 已验证；应用首次启动恢复属于 G12/G43 后续验收。证据见[G06旧偏好端到端迁移验证](evidence/goui-p0-g06-legacy-migration-2026-10-08.md)。

<a id="g07"></a>

#### G07 — 财务绘图路径前置验证

**依赖：** G03。**对应要求：** R02,R12,R13。**职责：** 图表。

**修改位置：** apps/goui/internal/charts验证入口；tests公开series。

**实施：** 用MyGO Painter绘折线和收入/支出分组柱；正负/单点/全零/缺值分段；鼠标/键盘point命中、exact表、theme/DPI。

**完成证据：** 真实窗口和headless都能查看cashflow图；零值与缺失分开、断线不跨segment；1×/2×及负轴正确；确认相邻标准Table可承载精确可访问数据。

**当前进度：** 折线图和分组柱状图已支持鼠标选点、Tab聚焦、左右箭头逐期切换及Home/End首尾跳转；提示与辅助技术描述包含当前期间精确文本。修复了图表位于窗口偏移位置时命中坐标错用窗口原点的缺陷；headless 回归覆盖偏移布局下折线与柱状图的中点选取、零值/缺失/负轴、精确表格及 1×/2×浅色/深色渲染。三个月临时账本原生窗口测试核对现金流柱图和净资产折线图的键盘选期、鼠标中点和逐月 exact help；完整 `make goui-native-e2e` 通过，新增原生场景连续5次通过。真人 VoiceOver朗读和真实显示器下深浅色/缩放视觉验收仍未完成，故G07继续进行中。证据见[G07记录](evidence/goui-p0-g07-2026-10-07.md)、[G36图表英文无障碍回归](evidence/goui-p6-g36-localization-2026-10-08.md)及[G37/G38原生窗口记录](evidence/goui-p6-g37-g38-native-e2e-2026-10-08.md)。

### P1 — 工作区完整生命周期

<a id="g08"></a>

#### G08 — 工作区状态机与资源所有权

**依赖：** G03。**对应要求：** R03,R04,R13。**职责：** 工作区。

**修改位置：** apps/goui/internal/workspace/state.go、intent.go、runner.go、engine.go。

**实施：** 实现Dispatch/State/Close与内部Engine adapter；main拥有已发布State；runner拥有Session/工作资源；epoch/op/windowID校验；Closing/Closed及清理责任。

**完成证据：** 可控adapter重现晚到Open、乱序read、关窗后Update丢弃；返回Session即使UI回调不执行也关闭；共享slice/map无并发写，相关race通过。

<a id="g09"></a>

#### G09 — 打开、轮询、重载与双窗口

**依赖：** G08,G05。**对应要求：** R03,R04,R08。**职责：** 工作区与窗口。

**修改位置：** apps/goui/internal/workspace/open.go、poll.go；internal/app/windows.go。

**实施：** 接真实desktop.OpenWithOptions、Poll/Reload/Close；每窗口独立Session；Open与Reload不阻塞main；欢迎/Invalid/Failed/Stale状态与最新诊断。

**完成证据：** cashflow载入、有效→错误A→错误B→恢复；初次invalid无别本残留；两个账本filter/profile隔离；关一窗另一窗正常；poll停止。

**当前进度：** `DesktopEngine` headless 测试已覆盖错误A→错误B→恢复、首次非法账本诊断、失败的新Open保留旧工作区、双窗口不同profile/filters以及单窗关闭；原生 MyGo 窗口测试验证入口文件修复时保留旧快照、修复后无需手动刷新自动恢复、include 文件原子替换后刷新仪表盘、MyGo `OpenFiles` 回调把首个账本送入欢迎窗口并为第二个账本新建窗口，以及已失效的最近账本进入可读诊断状态后继续打开其他账本时保留原窗口。四项G09窗口场景分别连续5次通过。欢迎页的最近账本可显式打开，偏好记录和最近列表自动化通过；真实 Finder 启动/文件关联及完整GUI矩阵仍需实机验收。详见[G09原生文件监视恢复记录](evidence/goui-p1-g09-native-watch-2026-10-08.md)。

<a id="g10"></a>

#### G10 — 有界读取与快照发布

**依赖：** G09。**对应要求：** R02,R03,R05,R13。**职责：** 工作区。

**修改位置：** apps/goui/internal/workspace/read.go、result.go。

**实施：** 一个执行中+一个最新待读；对normalized request/epoch/op/snapshot/revision校验；只读发布现有ViewResponse；展示索引不修改core结果。

**完成证据：** 连续改变日期/币种/页面时队列有界；晚到旧revision不覆盖新结果；read error正确映射；读取期间UI可交互；实际Session集成与race通过。

**当前进度：** Controller 已绑定规范化 Request、workspace epoch/operation、Session generation、SnapshotID 与 ledger revision；旧revision映射为 stale，不能覆盖当前结果，读取错误会显示错误而不丢弃旧投影。工作区测试验证一项执行中加一项最新待读、旧 revision 拒绝、错误保留当前结果；真实 `DesktopEngine` 集成与 race 通过。两项原生窗口场景分别验证三个月日期/币种/页面序列，以及在 balance-sheet Read 被门控挂起时继续修改日期和币种、应用筛选并导航到 dashboard；阻塞期间旧 dashboard 投影仍显示，解除后只发布最新请求且 SnapshotID 不变。两项场景分别重复5次通过。详见[G10 原生筛选与导航记录](evidence/goui-p1-g10-filter-navigation-2026-10-08.md)。

<a id="g11"></a>

#### G11 — 查询及捕获取消快速路径

**依赖：** G08,G09。**对应要求：** R03,R07,R10。**职责：** 工作区。

**修改位置：** apps/goui/internal/workspace/query.go、capture.go。

**实施：** 独立query runner，直接传context+CancelQuery；requestID单调；报表CancelReportExport不排在capture之后；Open/close/profile变更取消旧工作。

**完成证据：** 可控长任务下cancel可送达而不死锁；旧执行不改新状态；核心budget/取消结果正确处理；overflow不复用ID；计数与相关race通过。

**实际验收：** 每窗口查询runner独立于报表读取队列；request ID单调递增并在溢出时拒绝复用。可控任务测试覆盖读阻塞时直接取消、后发查询取代前序结果、账本切换/关窗取消及迟到结果丢弃；真实`DesktopEngine`测试覆盖typed重复列成功结果和带UTF-8 range的查询错误。报表捕获使用独立CaptureID和直接`CancelReportExport`路径，核心测试确认执行前取消会被记住。race与命令见[P1证据](evidence/goui-p1-2026-10-07.md)。

<a id="g12"></a>

#### G12 — 版本化偏好与迁移基础

**依赖：** G06,G08。**对应要求：** R11,R14。**职责：** 偏好。

**修改位置：** apps/goui/internal/preferences/schema.go、store.go、migration.go。

**实施：** userData版本化JSON、0600/0700、原子持久化及串行writer；明确ledger identity、列schema和旧键映射；保留旧偏好备份/marker，不存查询草稿/私人结果。

**完成证据：** load/update/restart/malformed schema/failed write有确定行为；未知旧键不猜；QA/dev隔离；两窗口并发写不丢项；相关race通过。

**当前进度：** `preferences-v1.json` 已增加最多5个明确可重开的账本路径和版本化旧 defaults 迁移标记；按规范化 ledger identity 保存偏好，最近路径只出现在最近列表字段。旧 profile/reading values 仅对不存在的新记录导入，已支持从旧 defaults 读取附件 roots 的 NSData/JSON、保留普通 bookmark 及 generation；现有 Go 偏好和显式清空的roots不会被旧值覆盖，未知键忽略。账本偏好现保存稳定表格ID的列顺序与宽度；报表、流水、持仓/价格、账户详情及分录/精确值表格在打开账本时恢复，动态查询结果按typed schema摘要隔离。诊断语言可按账本选择简体中文或英文，全部 Go UI 核心读取请求均使用该偏好；该控件只本地化核心诊断文本，不代表完整 UI 文案已双语。更新串行化并限制每账本最多32张表、每表128列；race测试覆盖保存/重启和多账本隔离。文件仍执行原子写入及0600/0700权限。新增“文件 → 导出偏好备份”，在原生 Save Panel 中说明备份范围，冻结版本化偏好快照并使用安全原子写入；备份文件权限为0600，拒绝覆盖活动偏好文件。单元和原生窗口测试覆盖迁移marker/最近账本/显示设置、保存前变更仍保留原快照、取消与成功保存、面板归属及实时偏好未被覆盖；全量race测试通过。2026-10-08 将正式启动路径中的 preferences/query-preset Store 组装提取为 `openApplicationStores`，它现在由 `main()` 调用；临时目录测试确认生产模式迁移最近路径/profile/reading并关闭 defaults reader，QA 模式完全不调用 legacy opener，同时核对两个独立偏好文件及私有目录权限。详见[G12启动Store隔离测试](evidence/goui-p1-g12-startup-store-isolation-2026-10-08.md)。旧生产 defaults 域迁移已通过只读验收；完整产品首次启动恢复仍需在 G43 验收，故G12继续进行中。证据见[G12列布局记录](evidence/goui-p1-g12-2026-10-07.md)、[G12诊断语言记录](evidence/goui-p1-g12-diagnostic-locale-2026-10-07.md)和[G12偏好备份导出记录](evidence/goui-p1-g12-backup-2026-10-08.md)。

2026-10-08 补齐 `PathUserData` 获取失败时的错误传递与欢迎页显示，并通过 `ui.Tester` 验证中英文文案；原生窗口 E2E 确认错误进入 macOS Accessibility 树，完整原生门禁通过（69.393秒）。现有偏好读取错误也会显示在未打开账本的欢迎页。系统路径失败后的真实打包 GUI 仍需 G43 验收。详见[G12启动错误显示](evidence/goui-p1-g12-startup-error-2026-10-08.md)。

### P2 — 报表与阅读主循环

<a id="g13"></a>

#### G13 — 界面壳、导航与阅读控制

**依赖：** G10,G12。**对应要求：** R05,R06,R08,R11,R12。**职责：** 界面。

**修改位置：** apps/goui/internal/app/model.go、navigation.go、commands.go；阅读控制与各页面模块。

**实施：** 侧栏/标题/状态/导航栈；日期period/currency/valuation；原币/缺价/旧snapshot提示；900×600与字体scale；稳定Key与操作Intent。

**完成证据：** 控制值与实际Request一致；报表/流水/query的控制含义正确；失效日期有校验；导航返回保留允许的局部状态；中文/英文不裁掉关键动作。

**当前进度：** 每个窗口已增加有界返回/前进历史（最多64页），新导航清空前进分支，受理新账本或profile reopen后立即清空历史；如果open intent本身被拒绝则保留旧历史。各页保留已有筛选、搜索和选择状态，返回到核心页面会使用当前读取控制重新请求。账户详情返回报表复用同一栈；诊断、附件目录和QA验证页也支持返回。Finder 打开/拖入的离线分析及错误页也进入历史，往返保留捕获状态且不读取账本或重跑查询。真实 `DesktopEngine` + `ui.Tester` 回归覆盖账户详情、筛选报表、流水返回/前进及新导航分支；AppKit原生Accessibility窗口回归覆盖普通页与离线捕获的返回/前进及分支替换，相关场景连续3次通过；双账本原生回归确认换账本会清空返回和前进历史；挂起 profile reopen 的真实 DesktopEngine 测试确认旧页面历史不能在新会话打开期间导航或发起读取。headless布局验证了英文返回/前进按钮在900×600、1.5×系统文字比例下未越界。`make goui-check` 与完整 `make goui-native-e2e` 通过。真人 VoiceOver及900×600真实显示器目视检查尚未验收，G13仍进行中。详见[G13导航栈回归](evidence/goui-p2-g13-navigation-2026-10-08.md)。

<a id="g14"></a>

#### G14 — 显式profile预览与应用

**依赖：** G13。**对应要求：** R02,R03,R05。**职责：** 工作区与界面。

**修改位置：** apps/goui/internal/workspace/controller.go；apps/goui/internal/app/semantics.go、model.go；Go UI 工作区测试。

**实施：** 复用migration view/InputIdentity；相同输入legacy/v3比较、明确apply；应用建新workspace epoch和Session，默认profile延续当前core决定。

**完成证据：** 预览后输入改变拒绝apply；从旧profile不残留数据/查询/capture；选择不修改账本；公开AVERAGE/migration场景通过。

**当前进度：** 语义配置页以 Legacy/V3 摘要和差异计数呈现只读预览；请求绑定当前 profile snapshot，应用前以核心 `InputIdentity` 再校验账本源图，成功后重开 Session 并持久化 profile，失败保留旧会话。工作区测试覆盖同输入应用、变更输入拒绝、report capture 取消及 query capture 迟到结果丢弃；profile reopen 受理后立即清空页面导航历史，挂起期间按返回不会把旧视图读取送入新会话；profile 替换清除旧响应和 capture 状态。真实 MyGo 原生窗口通过 Accessibility 验证公开 AVERAGE fixture 的预览、显式应用 V3、epoch/profile/snapshot 更新、旧查询草稿清理及账本字节不变；另在 watcher 发现源文件变化并 gate 新预览时确认旧操作隐藏，刷新后重放旧 `InputIdentity` 仍被拒绝且 Legacy 会话保持不变。相关原生场景连续5次通过；query capture 替换测试普通 `-count=10` 和 race 通过。上述任务验收条件已满足，G14 已验证。真人 VoiceOver及 profile/capture 并发的广泛 GUI 等价矩阵仍由 G37/G38 验收。证据见[G14首轮记录](evidence/goui-p2-g14-2026-10-07.md)及[G14原生窗口验证](evidence/goui-p2-g14-native-average-2026-10-08.md)。

2026-10-08 G14 补充：无有效 SnapshotID 的初次 Legacy 错误态也可以显示当前 migration preview，并通过输入身份安全应用有效 V3；旧 revision 仍拒绝。真实 MyGo 场景连续3次、headless race `-count=5`、最新完整 native gate 通过；当前 arm64 package 的可见界面也核验了 Legacy error→V3 apply→holdings 分币种筛选。G14 继续保持已验证；profile/capture 并发宽矩阵仍由 G37/G38 跟踪。详见[G14初次 Legacy 无效恢复](evidence/goui-p2-g14-invalid-legacy-recovery-2026-10-08.md)和[G43修复版包可见验收](evidence/goui-p7-g43-semantics-recovery-visible-2026-10-08.md)。

<a id="g15"></a>

#### G15 — 稳定表格、账户树和精确值控件

**依赖：** G13。**对应要求：** R02,R05,R06,R12。**职责：** 基础界面。

**修改位置：** apps/goui/internal/widgets/table.go、accounttree.go、amount.go、source.go；apps/goui/internal/app/model.go。

**实施：** ui.Table/OutlineTable封装选择、稳定Key、列布局和精确显示；多币种列、长中文、缺价/NULL；表格排序使用业务/typed比较，不按格式化金额字符串排序。

**完成证据：** 重排/过滤后selection指向同ID；重复标题列使用独立ID；树祖先/键盘展开正确；exact复制/label不经过float；headless与相关实际控件检查通过。

**当前进度：** 资产负债表和损益表已用 MyGo `OutlineTable` 展示可折叠账户层级，行身份和辅助功能标签用完整账户名；可全部展开/折叠，金额直接使用核心精确投影并保留币种列。headless测试覆盖树的可见行身份、可访问标签和展开/折叠。新增`internal/widgets.AccountTree`集中处理稳定FullName键、搜索祖先保留、原样使用核心汇总、未过滤币种列和完整树展开；页面已改用该模块。深层搜索会临时展开命中链路并在清除时恢复折叠；同账本新快照保留仍有效的展开键和所选账户，切换上下文清空旧状态；原生场景连续5次通过并通过 race 检查，详见[G15/G16账户树搜索与重载证据](evidence/goui-p2-g15-accounttree-reload-2026-10-08.md)。新增`AmountCell`统一近似显示、exact辅助名称/提示与右键精确复制，报表树/账户余额/持仓金额已接入。新增`SourceRow`统一来源路径、位置与复制/Finder/打开动作，诊断及关联诊断已接入；外部动作仍由应用处理，打开沿用快照身份校验。新增`StableTable[T]`校验稳定行/列ID并用 typed comparator 排序，已接入商品目录及持仓分组；数量/金额比较采用核心 exact rational，重排后选择保持同一行。全量Go UI测试、Darwin构建、race和vet通过。VoiceOver、真实键盘/排序操作、长中文及多币种列实机验收仍待完成，G15继续进行中。证据见[G15首轮记录](evidence/goui-p2-g15-2026-10-07.md)和[G15账户树与金额控件记录](evidence/goui-p2-g15-accounttree-2026-10-08.md)。

<a id="g16"></a>

#### G16 — 资产负债表与损益表

**依赖：** G15,G14。**对应要求：** R02,R05,R06。**职责：** 报表界面。

**修改位置：** apps/goui/internal/views/statements.go。

**实施：** 消费TreeReportPayload，搜索/全部展开折叠、原币/估值状态、账户跳转与导出identity呈现；不重算父汇总。

**完成证据：** cashflow截至日1000/当前700、期间300、options多币种/缺价/empty固定期望；搜索或折叠不改变总计；实际页面可操作。

**当前进度：** 资产负债表和损益表已支持全展开/折叠及按完整账户路径过滤；过滤只裁剪可见节点并保留祖先，币种列来自未过滤投影，父级金额直接用核心`SubtreeAmounts`且不会重算。核心顶层分类汇总固定显示，不受搜索或折叠影响；展开状态按报表类型和账本路径隔离，同账本重载保留有效状态。headless真实Session测试确认cashflow中的账户筛选、无命中时汇总仍在、清除还原、父级exact金额不变及切换报表类型时重新展开。新原生场景实际搜索深层账户、选择行、插入注释触发同账本自动重载并确认搜索/展开/选择保留，连续5次通过。新增options/empty UI测试覆盖USD转换缺少EUR报价提示、USD/EUR原币分列、2000-01-03截至日及空损益表；`native-multi`/缺价/empty内部状态码改为可读中英文提示。更多options/profile组合、报表捕获和导出身份联动及完整macOS页面验收仍待完成。证据见[G16记录](evidence/goui-p2-g16-2026-10-07.md)和[G15/G16账户树搜索与重载证据](evidence/goui-p2-g15-accounttree-reload-2026-10-08.md)。

2026-10-08 报表导出上下文补充：修正树报表分析文件把 `valuation` 写成 `native/converted`、丢失用户成本/市价选择的问题；有效请求口径及币种加入 `ReportExportIdentity`，保存面板显示原币/成本或目标币种/估值方法。核心回归覆盖 Legacy/V3 下的默认 `at-cost` 与 USD `market-value`，核对 identity、analysis normalized request、profile 与投影ID；中英文面板文案、desktop/app/workspace 测试、`make goui-check` 与完整 `make goui-native-e2e` 通过。G16 更广 options/profile 组合及 GUI 矩阵仍待验收。详见[G16报表导出上下文证据](evidence/goui-p2-g16-report-export-context-2026-10-08.md)。

2026-10-08 直属余额与折算值补充：对齐旧 SwiftUI 账户树信息层级，分别显示直属余额与子树汇总；从核心投影读取每账户折算总计，缺报价时区分已折算部分并列出缺少币种，日期和精确报价放入提示。Legacy/V3 headless 回归覆盖原币分列、直属/子树/分类合计同时呈现、完整/部分估值的中英文文案，以及 holdings include 的 at-cost/market-value；原生 Accessibility 回归覆盖栏标题、缺价/完整折算状态与带日期的精确 EUR/USD 报价提示。定向 headless 与 native 用例、`make goui-check`、全量 race/vet 和完整 native E2E 通过。详见[G16自身余额与折算估值显示](evidence/goui-p2-g16-own-subtree-valuation-2026-10-08.md)。

<a id="g17"></a>

#### G17 — 分页流水与交易详情

**依赖：** G15。**对应要求：** R02,R03,R05,R06。**职责：** 流水界面。

**修改位置：** apps/goui/internal/views/journal.go；internal/widgets/transaction.go。

**实施：** search/account/date/sort/offset/limit，最多100/页；可变行高posting/cost/@/@@/metadata，稳定交易展开；使用core总数和分页。

**完成证据：** 52,500交易页数/记录数正确；翻页不混snapshot；前插注释后唯一交易展开恢复、重复交易不继承；长中文与小数完整显示。

**当前进度：** 页面已接入核心快照绑定流水读取，支持全文/账户/日期筛选、排序、100条分页、总匹配数和翻页；详情直接显示核心精确分录、成本日期、`@`/`@@`价格、typed metadata、交易/分录来源及 Finder 动作。流水身份按交易内容跨无关行移动保持，重复交易身份限制在本快照。核心52,500笔合成账本测试检查首页/第二页总数、排序、边界和单次构建；真实 `holdings-trades.bean` 测试检查成本、总价、单位价、metadata 与源位置，MyGo headless 页面检查这些字段实际呈现。原生窗口测试通过 Accessibility 选择 Salary 交易并读取 exact postings，在105笔临时账本中实际翻页/返回并核对100/5条边界和同一snapshot（重复10次通过）；新增52,500笔临时账本原生测试核对首页/第二页各100条、总数、页码范围和同一SnapshotID，并在5次窗口测量中记录打开到财务概览就绪0.882–1.007秒、翻到前两页并达到Accessibility页码46.8–70.6毫秒（详见G39；不含进程冷启动）。900×600原生窗口中的长中文交易说明现完整换行，实际详情frame为628×53点且未超出900×568内容区，完整精确金额 `123456789.123456789 USD` 出现在Accessibility树中；场景 `-count=5`、`make goui-check` 和该轮包含36项常规窗口场景的完整 `make goui-native-e2e` 通过。G32流水导出现包含交易级/分录级 typed metadata 列；嵌套 map/list 保留类型化 exact 值并排除 metadata 来源路径，公开 `metadata-source.bean` 的 CSV/analysis 捕获测试通过。新增真实 MyGo 窗口场景验证选择唯一交易后，源文件前插入无关注释仍保留同一交易的分录详情；加入内容完全相同的重复交易后不继承旧选择。该场景连续5次通过并在 race 模式下通过，详见[G17流水选择跨源编辑恢复](evidence/goui-p2-g17-journal-selection-after-edit-2026-10-08.md)。键盘/VoiceOver实机使用、真实 Finder 动作、进程冷启动、滚动/屏幕呈现帧和更长时段性能仍属于G37/G38/G39验收范围。证据见[G17分页流水记录](evidence/goui-p2-g17-2026-10-07.md)、[G37/G38原生窗口检查](evidence/goui-p6-g37-g38-native-e2e-2026-10-08.md)和[G39性能记录](evidence/goui-p6-g39-performance-2026-10-08.md)。

<a id="g18"></a>

#### G18 — 全指令和配置阅读

**依赖：** G17。**对应要求：** R02,R05,R06。**职责：** 流水界面。

**修改位置：** apps/goui/internal/app/directives.go；internal/widgets/metadata.go。

**实施：** journal mode切换transactions/all/config；typed metadata/containers、tag/link前缀、block/dialect关联与generated posting来源；附件栏占位只显示实际DTO。

**完成证据：** 配置明确不按日期过滤；指令分页/筛选、来源和嵌套值正确；scalar/NULL/集合不过度字符串化；公开directives fixtures真实core/GUI通过。

**当前进度：** 新增全部指令/配置页，读取现有 `DirectivesPayload`，按模式、类型、文本与排序分页；配置按源码顺序且不显示日期/币种/估值控制。字段、metadata、嵌套 list/map、NULL、tag/link 前缀、交易 posting 精确金额/成本/价格与来源、生成记录的 origin ID 都有专用呈现；document 行只消费 core 返回的附件 DTO。`testdata/fixtures/directives/all.bean` 真实核心响应已在 headless UI 检查标记/metadata/方言生成关联；原生窗口测试再核对 all/config 模式、精确嵌套 decimal/NULL/bool/生成 lineage，并将日期筛选设到2099验证配置请求仍无日期范围且保留记录（连续5次通过）。配置日期范围不进入请求，搜索长度在 UI 层受 64 KiB 限制。修正核心 `PresentedDecimal` 在嵌套 list/map 中被格式化为 Go 结构体文本的问题后，真实 MyGo 900×600窗口中长嵌套中文值完整换行为177×153点，NULL、TRUE和精确小数保持原类型；场景连续5次通过。公开 fixture 的全部 supported dated/config 记录类型均经真实 core 响应进入 headless UI 呈现检查；定向 core/headless 测试、`make goui-check` 与该轮包含36项常规场景的 `make goui-native-e2e` 通过。键盘/VoiceOver实机使用属于G37仍待验范围。证据见[P2指令阅读记录](evidence/goui-p2-2026-10-07.md)与[G37/G38原生窗口记录](evidence/goui-p6-g37-g38-native-e2e-2026-10-08.md)。

<a id="g19"></a>

#### G19 — 诊断、错误恢复与来源动作

**依赖：** G09,G05,G13。**对应要求：** R04,R06,R08,R12。**职责：** 诊断界面。

**修改位置：** apps/goui/internal/views/diagnostics.go；internal/widgets/source.go。

**实施：** 展示最新error/warning/code/location/related，重试/复制位置/外部编辑器/Finder；主界面Stale提示与sheet内容同revision。

**完成证据：** invalid首次与Stale reload有区别；A/B连续错误诊断更新；source对应正确位置且不递归；clipboard内容准确；日志不含源正文。

**当前进度：** 欢迎页保留诊断摘要，侧栏增加独立诊断页；页面明确区分 invalid 与 stale 快照，显示最新消息、code、来源与关联来源，并提供复制位置、Finder 定位和外部编辑器打开。主/关联来源复用 `widgets.SourceRow`，打开回调会二次核验当前 revision/epoch/operation/snapshot；同一 workspace 的损坏账本→当前诊断→修复→重新读取闭环已有真实 `DesktopEngine` + MyGo 页面测试。诊断页面现在也以真实 `DesktopEngine` 连续读取 A/B 两次不同无效修订，验证第二次来源行号覆盖旧值，复制位置返回当前核心投影的精确绝对路径/行/列；关联来源按钮独立核对其自身路径/行/列。为避免多条诊断的同名动作被辅助技术误判，来源复制/定位/打开动作的 Accessibility 标签现包含具体路径和行列。原生 MyGo 测试通过当前诊断行的 AppKit `AXPress` 触发“复制位置”，并从 macOS 系统剪贴板核对完整绝对路径、行和列；切换页面时 Accessibility 树会暂留首页退出节点，测试现等待唯一 MyGo 节点 ID 连续8次稳定后再按下，恢复/复制场景连续10次通过，随后 `make goui-check` 和当前完整 `make goui-native-e2e` 均通过。应用级外部来源编辑器设置仅保存所选 `.app` 的 bundle ID，并经当前偏好路由来源打开、拒绝 OrangeCount 自身；bundle 解析、跨账本更新、取消保留、恢复系统默认和 Model 路由测试通过。真实原生菜单已验证选择器打开、取消和恢复系统默认；真实 macOS Open Panel 已选择 `/System/Applications/TextEdit.app`，页面显示设置成功且 QA 偏好中记录 `com.apple.TextEdit`；opt-in 测试验证 TextEdit 路由以及 Script Editor 未运行时首次启动。新 Store/Model 重建测试验证持久化 bundle ID 路由，TextEdit→Finder 回退顺序也有 Darwin 测试；应用侧来源/诊断路径无直接日志调用，入口日志仅处理偏好启动错误和 App.Run 错误。G19列出的完成证据均已验证。打包应用完整重启流程仍归入G43，真人 VoiceOver和其它多窗口 sheet 仍归入G37/G38验收。证据见[G19诊断页面记录](evidence/goui-p2-g19-2026-10-07.md)、[G19修复恢复闭环](evidence/goui-p2-g19-recovery-2026-10-08.md)和[G19外部来源编辑器设置](evidence/goui-p2-g19-source-editor-2026-10-08.md)。

### P3 — 图表与资产分析

<a id="g20"></a>

#### G20 — 仪表盘与现金流/净资产

**依赖：** G24,G16。**对应要求：** R02,R05,R06,R12。**职责：** 分析界面。

**修改位置：** apps/goui/internal/views/dashboard.go。

**实施：** 接HistoricalTrendsPayload、收入/支出/结余/净资产、currency series及期间；消费charts线/分组柱与exact详情。

**完成证据：** cashflow700、收入1000/支出300；单日净资产携期初、静默月份不造flow；缺价提示及tooltip exact/quote；实际键盘与hover通过。

**当前进度：** 仪表盘已消费核心 `HistoricalTrendsPayload`，按币种分开展示净资产折线、月度收支分组柱和旁边的 exact 表格；财务数值始终来自核心 projection，未用绘图 float 回显账务金额。headless 真实 Session 测试覆盖现金流收入 1000、支出 300、多币种分组、缺少 EUR 报价时的不可绘点，以及静默月份不虚构现金流；图表包覆盖鼠标悬停、键盘选择及10,000点绘图抽样/原始点命中。趋势响应现在保留报价源路径，净资产提示显示实际报价的汇率/日期/方向/来源行，缺价点提示商品与原因；真实 Session + UI 悬停测试已验证这条数据链。日期矩阵覆盖单日闭区间、余额结转、月末/月初、静默日和反向区间。另有独立“财务概览”导航页，欢迎页仍保留概览摘要。availability enum 已映射为可读中英文提示。2026-10-08 补齐仪表盘导出上下文规范化：原生币种在导出身份/分析文件中存为空币种，默认估值存为 `at-cost`，核心捕获测试通过；dashboard、账户和持仓导出上下文修复后的完整原生门禁耗时62.886秒并通过。真实窗口 VoiceOver/键盘/hover验收仍待完成。图表记录见[G24图表记录](evidence/goui-p3-g24-2026-10-07.md)和[G20仪表盘记录](evidence/goui-p3-g20-2026-10-07.md)。

<a id="g21"></a>

#### G21 — 账户详情与返回导航

**依赖：** G24,G16,G17。**对应要求：** R02,R05,R06。**职责：** 分析界面。

**修改位置：** apps/goui/internal/views/account.go；workspace导航Intent。

**实施：** 消费direct/subtree、lifecycle/booking/sources/history及账户流水；目录账户保持无直接开户资料；保存返回状态。

**完成证据：** 公开account-detail含目录/direct/subtree/无活动月份；全币种历史和来源正确；返回原页面filter/按稳定账户路径恢复selection；实际曲线通过。

**当前进度：** 资产负债表/损益表账户行可打开独立详情页；消费核心账户 DTO 显示 direct/subtree余额、期间变化、开户/关闭/booking、生命周期来源与 Finder 动作、按币种的 exact 历史曲线/表格及关联流水筛选和分页。估值完整度和缺少本地报价原因映射为可读文案；账户 DTO 附带报价源文件映射，详情页显示实际报价日期/精确汇率并可定位 Finder 或打开源文件。账户流水默认选中首笔交易，并显示交易级与分录级 metadata/source 和来源动作。真实临时账本测试分别覆盖 EUR 无报价与 STK/USD 报价来源动作；扩展的 `account-detail.bean` 页面测试检查交易/分录 metadata/source；目录账户与无活动月份测试确认不伪造直接开户资料或 1–0 交易行。返回报表时保留原报告响应、本地搜索和按稳定账户路径识别的选中行，只在快照仍相同时恢复；新快照清空旧选择。Headless `DesktopEngine` 往返测试检查账户操作、同一快照、筛选与选择恢复；新增原生窗口场景经 AX 聚焦账户行、使用独立详情动作、核对生命周期/历史/来源并返回同一快照，连续3次通过。2026-10-08 修正账户详情报表默认 `at-cost` 导出上下文，测试确认 identity 与分析文件均写有效估值；完整原生门禁通过（62.886秒）。证据见[G21账户详情记录](evidence/goui-p3-g21-2026-10-07.md)、[G21/G22报表导出上下文补充](evidence/goui-p3-g21-g22-report-export-context-2026-10-08.md)及[G21/G22/G23原生页面补充](evidence/goui-p6-g21-g23-native-pages-2026-10-08.md)。真人 VoiceOver及完整GUI等价矩阵仍待完成。

<a id="g22"></a>

#### G22 — 持仓、收益、事件与lot详情

**依赖：** G24,G15,G14。**对应要求：** R02,R05,R06。**职责：** 分析界面。

**修改位置：** apps/goui/internal/views/holdings.go；widgets/lot.go。

**实施：** group/event/history/lots模式，AVERAGE/cost currency/consumed/merged细节与均价曲线，截日与account scope保持coreRequest约束。

**完成证据：** 公开average-history/holding identities等exact案例通过；不同成本币种不相加；零/负/缺价/极值处理完整；切换详情不串ID，GUI可用。

**当前进度：** 新增原生持仓与成本页，可按账户/子树、grouping、日期、币种和估值读取快照；显示 exact 数量/成本基数/均价/未实现损益，缺项状态和原因不再暴露内部枚举；已实现事件展示收入、费用、总/净损益原因和来源；AVERAGE 历史显示 exact 曲线/表格；当前 lot、已消耗 lot 和合并前来源可分页查看并定位原始文件。真实 `average-history.bean` 页面测试覆盖 5 STK / 40 USD / 8 USD、缺报价、首次卖出 `400/3`、消耗 lot、两条合并前贡献和含清仓/回购间断的 7 点均价历史；`holdings-trades.bean` 测试覆盖已明确归属的 5 USD 费用、不明归属费用不产生净损益；`history.bean` 测试覆盖 100 USD 未实现收益和所用报价 Finder 来源；`holdings-identities.bean` 与101批次临时账本覆盖无成本库存、空头和第101个批次的页面翻页；`holdings-unknowns.bean` 页面测试确认 USD 成本基数 100 与 EUR 成本基数 60 分组显示、不跨币种合并。真实临时账本还验证 EUR 成本按本地 EUR/USD 转为 60 USD、STK 通过反向本地报价显示 6 USD，并拒绝通过 GLD/EUR 与 EUR/USD 猜测多跳市值。请求身份测试覆盖账户 scope、grouping、过滤、排序、分页、快照以及 group/event ID 分离。2026-10-08 修正持仓报表默认原币 `at-cost` 和显式 USD `market-value` 导出上下文，identity 与分析文件均反映投影有效口径；完整原生门禁通过（62.886秒）。新增原生场景按 `Assets:Stock` 筛选，逐页检查 exact 当前持仓、7点 AVERAGE 历史和批次详情，连续3次通过。自动化证据见[G22持仓与成本记录](evidence/goui-p3-g22-2026-10-07.md)、[G21/G22报表导出上下文补充](evidence/goui-p3-g21-g22-report-export-context-2026-10-08.md)及[G21/G22/G23原生页面补充](evidence/goui-p6-g21-g23-native-pages-2026-10-08.md)。目录账户与空历史的自动化页面矩阵已通过；更多反向/跨币种转换边界、真人VoiceOver及完整GUI/DPI验收仍待完成。

2026-10-08 新增 `TestHoldingsPageKeepsDirectoryAndEmptyAverageHistoryHonest`：目录账户 direct 范围不包含子账户持仓，非 AVERAGE 账户显示明确空历史；`make goui-check` 与完整原生门禁通过（60.199秒）。详见[G22持仓与成本记录](evidence/goui-p3-g22-2026-10-07.md)。

<a id="g23"></a>

#### G23 — 商品目录与价格历史

**依赖：** G24,G15。**对应要求：** R02,R05,R06。**职责：** 分析界面。

**修改位置：** apps/goui/internal/views/prices.go。

**实施：** 商品/pair、history/commodities、sort和cutoff quote、direct/inverse/zero/missing/source multiplicity；价格曲线按source_order与segment。

**完成证据：** 公开价格边界与来源数通过；不可画value保留exact并断线；pair/cutoff/source操作准确；同日期不乱序。

**当前进度：** 新增原生“商品与价格”页，可切换商品目录与报价历史，按商品、报价币种、日期和方向读取固定快照；目录展示账户、报价币种、价格数和声明数，支持在完整核心目录分页前按商品/账户/报价币种过滤，选择商品后可查看每条声明来源及 Finder 动作；历史显示截止日 direct/inverse 报价、来源位置、exact 曲线/表格、选择来源 Finder 动作和分页。目录与历史分页偏移分开。核心 source segment 现传到图表，连续区段不会被折线连接；不可绘制精确报价仍留在 exact 表，价格表新增 1 起始“来源段”列。真实 `prices.bean` 测试覆盖未报价/仅声明商品、目录过滤后清除搜索、跨 include 同日多报价的 source order、历史 exact 值、截止报价、声明/报价来源动作和分页隔离；核心 205 商品测试搜索到第 200 项之后且只返回过滤后的总数。临时巨大报价/零价 UI 测试确认 exact 值保留、零值可画、来源段正确变化。新增原生窗口场景通过目录账户筛选、AX 选择 STK 行及报价下拉选 USD，核对 `2000-01-20` 的 `99 USD` 和来源，连续3次通过；AX 重复标签的节点身份由仅测试编译的钩子消歧。2026-10-08 价格历史导出身份新增规范化报价对，保存面板显示 pair，默认文件名区分不同 pair；根测试、vet、`make goui-check` 与完整原生门禁通过。来源段表格回归及最新完整原生门禁也通过（60.199秒）。自动化证据见[G23商品与价格记录](evidence/goui-p3-g23-2026-10-07.md)、[G23/G32价格报表导出上下文补充](evidence/goui-p3-g23-g32-price-export-context-2026-10-08.md)及[G21/G22/G23原生页面补充](evidence/goui-p6-g21-g23-native-pages-2026-10-08.md)。真人VoiceOver/DPI和完整GUI矩阵仍待完成。

<a id="g24"></a>

#### G24 — 完整可复用财务图表Module

**依赖：** G07,G15。**对应要求：** R02,R12,R13。**职责：** 图表。

**修改位置：** apps/goui/internal/charts/model.go、layout.go、hit.go、render.go、tests。

**实施：** 包内 Layout/Hit/Render 接口；日期/数值域、zero/negative/legend、segment、min/max像素桶抽样；hover/键盘/精确标准Table/AX。布局当前按每帧 viewport 重算；只有性能基准证明需要时才增加带明确数据版本和 viewport key 的私有缓存。

**完成证据：** 纯布局独立期望、10,000点抽样保持极值与断点；命中原始点而非仅抽样；窗口尺寸/scale变化使用当前 viewport 重算，主题颜色使用当前 Theme；真实 drawing 与 AX 验证入口可用。

**当前进度：** 折线、分组柱和 exact 标准表格已支持按鼠标横坐标命中原始期间、绘制选择指示并展示 exact 值；分组柱增加与系列颜色一致的收入/支出图例。键盘可聚焦、左右键切换、Home/End跳转，当前 exact 文本也进入辅助技术描述。10,000点绘图按当前宽度分配像素桶预算，折线逐有效段保留首尾/每桶极值；折线区间同时尊重缺失断点和核心显式 source segment。柱图逐series保留期间边界/每桶极值并保持至少0.5 DIP宽；鼠标、键盘命中和精确表格仍指向完整原始序列。headless 覆盖稀疏月份、负/零/缺失、显式segment、图例、1×/2×、纯布局、viewport变化、命中、悬停/键盘和10,000点未抽样点命中。包内 Layout/Hit/Render 已分离，外部入口保持精简；当前无布局缓存，每次使用最新 viewport 和 Theme。VoiceOver与实机字体/DPI/绘图验收尚未完成。详见[G24图表记录](evidence/goui-p3-g24-2026-10-07.md)。

### P4 — 查询与捕获阅读

<a id="g25"></a>

#### G25 — 生产查询编辑器交互

**依赖：** G04,G13。**对应要求：** R07,R12。**职责：** 查询界面。

**修改位置：** apps/goui/internal/widgets/queryeditor.go；internal/views/query_editor.go。

**实施：** 固定TextArea与选区adapter，64KiB UTF-8限制，Cmd+Enter/按钮显式执行，组合输入不提交；undo/redo/clipboard/error range/focus。

**完成证据：** 中文拼音/emoji/组合字符/CRLF、draft与executed text不同位置拒绝定位；headless compose和实际输入分别记录；undo和菜单编辑角色正常。

**当前进度：** 新增原生 TextArea、显式执行按钮、Cmd+Enter、64 KiB UTF-8 上限、组合输入期间阻止快捷键提交，以及基于 UTF-8 parser range 的选区导航；执行文本与草稿分离，改草稿会保留旧执行。Headless UI 覆盖 emoji 前缀错误定位与替换、快捷键执行、草稿不自动运行、undo/marked-text 的 MyGo 编辑器回归和超限拒绝。真实 macOS 窗口测试通过 AppKit“编辑→撤销/重做”菜单完成字符往返，并通过“全选/拷贝/粘贴”菜单将中文/emoji BQL 精确复制后替换选中草稿，均确认不会自动运行；Undo/Redo、Copy/Paste 各连续5次通过。原生测试保存并恢复 general pasteboard 的项目、类型和字节，无法安全保存延迟提供的内容时不触碰剪贴板。真人输入法候选操作和 VoiceOver 仍待验，完整结果表交互由 G27 继续实现。新增项后 `make goui-check` 与包含40个常规原生窗口场景的 `make goui-native-e2e` 通过。证据见[P4 记录](evidence/goui-p4-2026-10-07.md)。

<a id="g26"></a>

#### G26 — 查询工作区与执行上下文

**依赖：** G25,G11。**对应要求：** R02,R03,R07。**职责：** 查询界面与工作区。

**修改位置：** apps/goui/internal/views/query.go；workspace/query状态。

**实施：** draft/running/success/error/stale独立状态；快照/profile/fingerprint/执行文本与预算计数显示；cancel、连续Run和保留上次成功结果。

**完成证据：** 改草稿不改成功result；reload/profile/reopen后旧query标识/拒绝结果；长query cancel有效；不会应用其它页面共享日期filter。

**当前进度：** 每账本快照上显式执行查询，显示执行文本、快照ID、profile、evaluation revision、指纹、返回行数、文本字节/工作步预算；保留上次执行供恢复，workspace runner直接取消并在账本生命周期变化时清除旧结果。真实 DesktopEngine fixture 查询已从查询页点击运行并显示重复列结果，Command-Enter 快捷路径通过 headless 测试。新增查询取消进度文案：工作区进入 `QueryCancelling` 时显示“正在取消…”，并禁用重复取消按钮；headless状态测试通过。原生窗口 E2E 将真实 `DesktopEngine.ExecuteQuery` 挂起在取消入口，经 Accessibility 点击“取消查询”，核对只取消当前 request ID，最终收到 typed `cancelled` 响应；另覆盖 File 菜单重载及应用 V3 profile 重开会话，两者都等新 snapshot/session 就绪后才释放旧查询，并确认迟到响应不复活；还验证离开查询页期间查询仍完成并在返回后显示。取消、重载、profile重开和离页完成场景分别连续5次通过。最新完整 `make goui-check` 与44项常规窗口场景的 `make goui-native-e2e` 通过。G26验收项已满足并标记已验证；跨功能旧版 GUI 等价矩阵仍由 G38验收。

<a id="g27"></a>

#### G27 — 类型化结果表格、复合详情与复制

**依赖：** G26,G15。**对应要求：** R02,R07,R12。**职责：** 查询结果界面。

**修改位置：** apps/goui/internal/app/query.go、model.go、model_test.go；后续可按交互与格式化边界拆成widgets。

**实施：** ordered columns+positional cells、重复title、NULL/empty Inventory、Amount/Position/Inventory/recursive metadata；动态schema隔离列布局；按coretyped值比较。

**完成证据：** 标量/复合/NULL/重复标题/零行真实query输入通过；typed JSON复制保持context/column order/full cost；10,000行选择/滚动有界；整数不失真。

**当前进度：** 查询表已给稳定列 ID 和固定可滚动视口；typed schema变化会清除旧排序，并按schema摘要仅恢复匹配的列布局，执行ID变化会重置选择，行身份由执行ID和原始行号组成。选中详情精确展示 decimal/amount/position/inventory，并递归展开 typed metadata；详情最多显示199项再折叠。复制行输出包含 QueryContext、contract、按序 columns 和原始 typed cells，数值仍以字符串保持大整数精度。可排序列调用核心 `query.CompareCells`，覆盖整数/decimal、字符串/日期、bool、amount、position、inventory；不支持比较的集合类型不可排序。排序切换保留原结果行选择，复制仍取该行。测试现覆盖重复标题、NULL/空字符串、持仓成本、typed JSON 复制、10,000行虚拟首屏/鼠标滚动到末端/Home/End、排序后选中复制及真实 DesktopEngine 零行结果schema。新增真实 macOS 原生窗口回归：在 900×600 窗口和 150% 字号下执行真实现金流查询，Accessibility 聚焦结果行，Home/End 选择首尾；按 `number` 表头排序后核对数值升序且原选中 Salary 行跟随到新位置，再用 Up/Down 检查排序后的选择与详情，并点击复制按钮核对 macOS 剪贴板 JSON 与原始 context/schema/cells 完全一致；测试会精确恢复原 pasteboard。定向用例连续5次通过，完整 `make goui-native-e2e` 通过（66.217秒）。VoiceOver、表格外完整键盘链路和不同显示器/DPI 实机验收仍待完成，故G27继续进行中。证据见[G27类型化查询结果记录](evidence/goui-p4-g27-2026-10-07.md)。

<a id="g28"></a>

#### G28 — 保存query目录与预设

**依赖：** G26。**对应要求：** R06,R07。**职责：** 查询界面。

**修改位置：** internal/desktop/session.go、query_catalogue.go、journal.go；apps/goui/internal/app/query.go、model.go、model_test.go、internal/querypresets/。

**实施：** 消费现有QueryCatalogue；目录按日期/名称排序并在核心分页前搜索日期、名称和BQL；支持显式日期范围生成模板；用户明确命名后将每账本BQL写入独立安全存储。选择所有模板仅填draft，显式Run；snapshot变化更新目录/内置预设，不把保存预设误当成自动执行。

**完成证据：** 四预设/账本saved query真实core一致；目录搜索在分页前过滤并保持稳定身份；日期/名称升降序、按期间生成和命名预设保存/重开/删除均有测试；选择不触发Execute；新目录不冒充旧snapshot；切换账本不保留别本条目；真实macOS键盘/VoiceOver验收。

**当前进度：** 查询页显示核心 `QueryCatalogue` 的四个内置模板和当前账本保存的 BQL，带来源位置、分页、日期/名称排序及在分页前按日期、名称和 BQL 搜索；自定义日期范围可生成期间模板。用户显式命名保存的预设写入独立 `query-presets-v1.json`，以账本 identity 隔离，支持重开和确认后删除；该文件具备schema校验、0600/0700、原子写入和有界存储。所有预设选择只填草稿，不运行查询。目录请求绑定当前 snapshot，并排除其它报表共享筛选；目录响应与当前 snapshot 不同，或 workspace 正在重载/非有效状态时都会隐藏选择入口。新增原生窗口测试实际打开并切换两个临时账本，确认各自命名预设不串账；重载门控期间旧 snapshot 的核心查询目录不再显示，重载后新 snapshot 的更新目录恢复，账本自定义预设仍按账本保留；另在切换账本前建立双向页面历史，确认换账本会清空两侧导航历史，且 Accessibility 的返回/前进控件均不可按。G28 场景此前连续5次通过；加入导航历史断言后，相关原生场景连续3次及完整原生 E2E 均通过。根核心查询目录测试及Go UI store/headless测试通过，证据见[G28查询目录记录](evidence/goui-p4-g28-2026-10-07.md)和[G28多账本与重载原生窗口验证](evidence/goui-p4-g28-native-e2e-2026-10-08.md)。键盘/VoiceOver仍待验收，故 G28 继续进行中。

<a id="g29"></a>

#### G29 — 离线分析文件阅读

**依赖：** G27。**对应要求：** R02,R06,R07,R08。**职责：** 查询捕获阅读。

**修改位置：** apps/goui/internal/app/query_analysis.go、document_open.go、model.go、query.go、model_test.go、native_e2e_darwin_test.go、cmd/orangecount-goui/main.go；apps/goui/qa/Info.plist、package.sh、third_party/mygo/e2e_testhooks_darwin.go；internal/desktop/query_analysis_test.go。

**实施：** 用ReadQueryAnalysis读取现有archive；统一typed展示；识别capture context、版本/预算/checksum；打开捕获不修改活跃ledgerSession。

**完成证据：** 当前core支持的v1/v2/typed-v3通过；损坏/超限/未知版本拒绝；不自动执行原query；复制与详情精确；明确离线捕获身份。

**当前进度：** 查询页增加原生文件面板读取 `.ocanalysis`；文件大小先限于核心查询字节预算，之后委托 `ReadQueryAnalysis` 校验 archive、checksum、格式与typed cells。读取在后台完成，成功后以“离线分析捕获”身份显示producer/format/contract/snapshot/profile/completeness；不会打开或更改活动ledger session，也不会执行捕获查询。Finder 打开/窗口拖入的捕获会进入页面历史；返回和前进可恢复离线结果或读取错误，不触发账本读取或查询执行。用户可显式把原查询填入草稿，仍要单独执行；可关闭捕获回到活动查询。Headless真实v3捕获测试覆盖位置全成本详情、腐坏/超限拒绝、填草稿不执行、关闭后活动查询仍原样；还检查数量、成本、日期和标签，并把复制行的QueryContext、typed contract、列schema及cells与捕获记录逐字节比对。核心测试另覆盖v1/v2/v3 scalar contracts和未知contract/未来format拒绝。macOS 原生 E2E 通过 MyGo application delegate 打开 `.ocanalysis`，并通过窗口 Finder 风格拖放打开第二份捕获；确认已有捕获保留在原窗口、第二份进入新窗口，两个 workspace 均没有账本路径；损坏 archive 从同一delegate打开时在隔离窗口显示读取错误。扩展的原生场景还通过 Accessibility 返回/前进恢复成功捕获与损坏文件错误，连续3次通过。QA 与开发包 Info.plist 已注册 `org.orangecount.query-analysis` UTI 和 Viewer 文档类型，QA 使用 Alternate rank、开发包使用 Default rank。整套 `make goui-check`、`make goui-native-e2e` 通过；QA plist 与实际生成的开发 `.app` plist 均通过 `plutil -lint`，临时目录内的 app/zip/checksum 打包成功。真实 Launch Services 双击、真实旧版本文件语料、完整键盘/VoiceOver及其他 macOS 版本仍未验收；G29验收标准均已满足，状态为已验证。证据见[G29离线分析阅读记录](evidence/goui-p4-g29-2026-10-07.md)与[G29 Finder/拖放路由记录](evidence/goui-p4-g29-native-open-drop-2026-10-08.md)。

### P5 — 附件与安全导出

<a id="g30"></a>

#### G30 — 统一固定捕获协调

**依赖：** G10,G11。**对应要求：** R03,R10。**职责：** 工作区导出。

**修改位置：** apps/goui/internal/workspace/export.go；localfiles/capture.go。

**实施：** 报表ProjectionID/scope和query requestID/fingerprint分别核验；捕获前成功结果仍当前；复制content/context/protection身份，限制一个保存流程。

**完成证据：** capture后reload/后续query不换字节；expired/projection mismatch不显示保存成功；取消先到capture和窗口关闭案例可控通过。

**当前进度：** 查询捕获按成功 QueryResponse 的 snapshot/requestID/fingerprint/format 核验，报表捕获按当前 ReportExportIdentity 的 projectionID/snapshot/scope 核验；workspace 不允许两类 capture 并行，成功 capture 都复制冻结 bytes 与来源保护身份。查询与报表共用单一 active export slot、取消入口、Save Panel 后的保护刷新和 `saveCapturedExport` 原子文件写入。取消、窗口/账本生命周期和旧 projection 拒绝有可控覆盖。真实 DesktopEngine query/report fixtures 验证 capture-after-reload 保持原 bytes，并刷新后保护新 source。定向测试和本轮全量 race/build/vet/root/license/diff 检查通过，G30 已验证。证据见[G30记录](evidence/goui-p5-g30-2026-10-07.md)、[G32记录](evidence/goui-p5-g32-2026-10-07.md)及[G33记录](evidence/goui-p5-g33-2026-10-07.md)。

<a id="g31"></a>

#### G31 — Go安全文件保存Module

**依赖：** G30,G05。**对应要求：** R10,R13。**职责：** 本地文件。

**修改位置：** apps/goui/internal/localfiles/save_darwin.go、protection.go、tests；platform/alias_darwin.go。

**实施：** 迁移Swift保护算法：规范父目录、os.Root、独占临时0600、完整write/fsync/close、重新验证path/inode/alias/parent、Root.Rename；失败清理；取消与commit顺序明确。

**完成证据：** 真实source/hardlink/symlink/alias/parent rename、目标替换、write failure/overwrite/cancel通过；失败保留源和既有目标；替换无半写；不以Root本身当完整安全证明。

**当前进度：** `localfiles.Capture` 冻结字节并保留源/附件路径与文件身份；Darwin 实现以 `os.Root` 锚定目录，在同目录创建独占0600临时文件，完整写入、fsync、关闭后再次检查父目录身份、目标类型、Finder alias 和保护身份，再经 `Root.Rename` 原子提交。真实 macOS 测试覆盖新建/覆盖、source/hardlink、移动源后的身份保留、缺失保护路径、symlink、Finder alias、提交前目标替换、父目录替换、部分写入EIO、临界取消和临时文件清理。全量 race、goui-check、vet、核心测试、license 和 diff 检查通过。G30/G32/G33 已接入；G34/G35 的附件 Open/Finder 与目录共享撤销代码分别在各任务卡记录，macOS 可见界面验收仍待完成。边界和逐项证据见[G31记录](evidence/goui-p5-g31-2026-10-07.md)。

<a id="g32"></a>

#### G32 — 全部适用报表导出

**依赖：** G31,G16,G17,G20,G21,G22,G23。**对应要求：** R02,R06,R10。**职责：** 报表导出界面。

**修改位置：** apps/goui/internal/views/export_controls.go；workspace/report_export.go。

**实施：** 按实际ReportExportIdentity开放CSV/analysis；面板说明current-page/complete-projection、日期/profile/completeness；reload刷新保护不换内容。

**完成证据：** 树/趋势完整投影及流水/账户/持仓/价格当前core支持section逐一验证；搜索折叠不缩小complete范围；不支持的指令模式不伪造export identity。

**当前进度：** 通用报表保存工具条只在当前响应带有效 `ReportExportIdentity` 时显示，已接入 dashboard、balance/income、journal、account、prices 的 commodities/history 与 holdings 的 groups/events/history/lots；面板明确 scope、日期、profile、completeness 和截断，CSV/分析归档均走 G30/G31 安全协调。流水导出包含交易级与分录级 typed metadata；`list<record>` schema 的 `value_json` 保留 exact 值并递归剔除 metadata 来源位置。2026-10-08 补齐价格历史导出的规范化报价对，面板展示 pair，默认文件名含报价对，避免不同 pair 使用相同建议名。测试覆盖 11 类页面/section 的 UI 动作、CSV/analysis 实际保存与读回、metadata/null/false/exact/source-path 用例、取消不落盘、树搜索不改变完整导出身份、directive 页不显示虚构导出，以及保存面板期间 reload 后保护新 source。G32 自动化验收通过；报表原生保存面板的双窗口归属、取消和关窗清理以及查询面板归属/取消已由 G37/G38 覆盖，两类面板实际确认保存、VoiceOver、用户视觉验收和完整 macOS 交互矩阵仍待验收。证据见[G32记录](evidence/goui-p5-g32-2026-10-07.md)、[G23/G32价格报表导出上下文补充](evidence/goui-p3-g23-g32-price-export-context-2026-10-08.md)、[G32 metadata 导出补齐](evidence/goui-p5-g32-metadata-2026-10-08.md)和[G37/G38原生窗口检查](evidence/goui-p6-g37-g38-native-e2e-2026-10-08.md)。

<a id="g33"></a>

#### G33 — 查询成功结果保存

**依赖：** G31,G27,G28。**对应要求：** R02,R07,R10。**职责：** 查询导出界面。

**修改位置：** apps/goui/internal/views/query_export.go；workspace/query_export.go。

**实施：** 当前成功query捕获保存；CSV与ocanalysis说明差异；父窗口面板、cancel、失败notice；保存期间重载保护更新。

**完成证据：** draft变化不改变content；陈旧query不能新准备capture；capture后reload仍保存原值且保护新源；query失败、关窗/离页、覆盖保护正确。

**当前进度：** 查询工作台已提供当前成功结果的 CSV 与 `.ocanalysis` 原生保存面板；执行前固定成功结果，冻结字节与typed context，面板期间取消或 reload，提交前刷新来源保护。测试验证 draft 修改不换内容、最新查询失败后旧成功结果不能继续保存、离页/窗口关闭取消、CSV/分析文件读回、cancel不落盘及reload新增来源保护；MyGo cancellable Save Panel单测通过。G33 已验证；真实用户操作和完整 macOS GUI 矩阵由 G37/G38 验收。证据见[G33记录](evidence/goui-p5-g33-2026-10-07.md)。

<a id="g34"></a>

#### G34 — 附件声明到Open/Finder闭环

**依赖：** G18,G05。**对应要求：** R03,R06,R09。**职责：** 附件界面。

**修改位置：** apps/goui/internal/views/attachments.go；workspace/documents.go。

**实施：** 显示coreAttachmentDTO状态；动作发送ID/snapshot/policy，Session重验证后主线程再次核epoch，再系统打开/显示。

**完成证据：** available/missing/outside-root/permissions及身份替换覆盖；列表path不能绕过授权；晚到动作不打开旧账本；实际Open/Finder通过。

**当前进度：** 附件页已读取核心 `document` 指令 DTO，分页显示文件名与可用/缺失/越权/不可读状态，不展示解析后的绝对路径；只有可用项能请求打开或 Finder 显示。动作把 ID、snapshot 和 policy revision 发回当前 Session 重验证，窗口再核对工作区 epoch、当前账本/快照/策略后才调用系统 Shell。headless DesktopEngine 测试覆盖实际打开、Finder adapter 调用及策略变化后拒绝迟到动作；核心 documents 测试覆盖缺失、越权、文件身份替换等路径。真实 Finder/默认应用弹出与权限受限情形仍待 macOS GUI 验收。

<a id="g35"></a>

#### G35 — 附件目录管理、恢复与共享撤销

**依赖：** G34,G06,G12。**对应要求：** R09,R11。**职责：** 附件与偏好。

**修改位置：** apps/goui/internal/views/attachment_roots.go；preferences/roots.go；workspace/root_policy.go。

**实施：** 普通bookmark准备→Configure→持久化，失败标会话有效；按ledger identity通知同应用窗口refresh policy；旧偏好迁移/失效项重新选择。

**完成证据：** 两窗口roots撤销立即失效，filters/Session仍隔离；stale bookmark/目录inode替换拒绝；默认root不扩成include授权；restart/dev/QA迁移不相互覆盖。

**当前进度：** 偏好层按账本保存并恢复普通 bookmark 记录与 generation；旧 UserDefaults 可迁移旧 roots，清空列表会持久化为空值，防止旧目录复活。Go UI 已提供目录选择、宽范围确认、移除及失效标记；有效记录先解析并以 CAS 更新本窗口 Session，再持久化。启动恢复在首次报表读取前应用；同一进程其它窗口收到版本事件后更新自己的 Session policy 并刷新当前视图。默认账本目录不可重复添加或移除。失效书签可通过重新选择同一路径替换；有效重复项仍拒绝。Store 通知、workspace CAS、恢复顺序、bookmark 身份、Model 操作与竞态已有 headless/race 覆盖；原生窗口测试覆盖双窗口撤权、真实 `NSOpenPanel` 归属/取消、picker 结果返回后的授权持久化。新增两个真实 MyGo 窗口场景：以新 preferences Store 和新 Model 模拟启动后从磁盘恢复 bookmark，并验证策略应用后附件可用；将已授权目录移走并在原路径新建目录后，页面显示身份失效，重新选择该路径会生成新 bookmark、替换旧记录并恢复附件授权。两项各连续5次通过，并在 race 模式通过。另在隔离 Manual QA `.app` 中用真实 `NSOpenPanel` 选择临时附件目录，原生页面显示 bookmark 路径并把 `receipt.pdf` 标为可用；退出并重新启动该 `.app`、重新打开临时账本后，附件仍为可用。随后移除测试目录授权，并从 Manual QA 偏好中清理仅由本次测试创建的 ledger identity/recent path 和临时文件。新增两个真实 MyGo 窗口场景，通过 AppKit NSAlert 验证宽范围根目录确认 sheet 的归属、取消无副作用，以及允许后仅将隔离临时 home 写入当前 Session/临时偏好并成功解析 bookmark；两项各连续5次通过。开发包和 QA 身份间的偏好迁移/隔离完整验收仍未完成，G35 继续进行中。证据见[G35 多窗口撤权与原生目录面板验证](evidence/goui-p5-g35-multiwindow-2026-10-08.md)和[G35 大范围附件目录确认](evidence/goui-p5-g35-broad-root-confirmation-2026-10-08.md)。

### P6 — 全面验收与性能

<a id="g36"></a>

#### G36 — 全页面文案、快捷键与命令焦点

**依赖：** G19,G20,G21,G22,G23,G28,G29,G32,G33,G35。**对应要求：** R06,R08,R11,R12。**职责：** 应用与界面整合。

**修改位置：** apps/goui/internal/i18n；app/menus.go、commands.go；全部views。

**实施：** en/zh-CN统一资源；Cmd+O/R/F/Enter/C、Undo/Redo、Tab/方向键、菜单与窗口焦点；theme/fontscale/minsize；source/clipboard规则一致。

**完成证据：** 所有菜单作用于当前窗口；文本编辑与全局快捷键不抢IME；900×600及1.5字号无不可达动作；中英文/深浅色金额与请求语义一致。

**当前进度：** MyGo 原生应用菜单已加入打开账本、新建窗口、重新读取、查找及标准编辑命令；窗口菜单回调通过焦点窗口注册表路由。en/zh-CN 文案资源已经接入全部页面、图表、原生文件/确认面板与菜单；动态账本账户、商品名称、查询文本、金额和来源路径保持原值。Headless 测试覆盖菜单路由、界面语言持久化、英文资源覆盖、导出说明、查询类型标签，以及七类搜索页在900×600、1.5字号下的焦点。双窗口原生场景先后聚焦两个账本窗口并通过 AppKit Cmd+F key-equivalent 路由查找；Cmd+O 打开面板只附属于当前窗口且取消后无孤立面板；Cmd+R 只重新读取当前窗口账本；Cmd+N 原生创建新的可访问 Content 窗口。四项场景各连续5次通过。`make goui-check` 与当前完整原生窗口套件均通过。原生现金流测试核对折线/柱图选期和 exact help，并切换英文检查辅助功能名称、说明和表头；独立 development 包的实际 Accessibility 树也显示英文图表名称、键盘帮助和 `Period`。静态目录测试已覆盖 chartText 文案；缺失净资产点的 exact 状态亦修正为随语言切换。原生输入法冲突、真人键盘事件和 VoiceOver 仍由 G37 验收；深浅色与 900×600/1.5 原生矩阵尚未全部覆盖，状态仍为已实现待验证。本轮修复了持仓/商品表格错误提示和图表本地化缺漏；动态错误后缀、中英图表状态及原生回归通过。详见[G36 首轮菜单与焦点记录](evidence/goui-p6-g36-2026-10-07.md)、[G36 本地化与焦点更新](evidence/goui-p6-g36-localization-2026-10-08.md)、[G36 双窗口 Cmd+F 原生验证](evidence/goui-p6-g36-native-find-2026-10-08.md)、[G36 双窗口 Cmd+N 原生验证](evidence/goui-p6-g36-native-new-window-2026-10-08.md)和[G36 双窗口 Cmd+O/Cmd+R 原生验证](evidence/goui-p6-g36-native-open-reload-2026-10-08.md)。

2026-10-08 补充：新增每账本字号偏好和 MyGo `SetTextScale` API；View 菜单与 Cmd+Plus/Cmd+-/Cmd+0 在真实窗口中改变布局，900×600 下 150% 仍可聚焦账户搜索框，重开账本恢复保存值。定向 native 场景连续5次通过，`make goui-check` 与整套原生 gate 通过。Light/Dark 外观切换后的原生 Accessibility 保持已覆盖；颜色像素对比、其他窗口尺寸和完整视觉矩阵仍待覆盖，故 G36 维持“已实现待验证”。详见[G36 每账本字号与快捷键验证](evidence/goui-p6-g36-text-scale-2026-10-08.md)。

<a id="g37"></a>

#### G37 — 真实输入法和VoiceOver验收

**依赖：** G36,G04。**对应要求：** R07,R12。**职责：** 系统交互验收。

**修改位置：** apps/goui/tests/gui-ime、gui-ax场景；docs/evidence。

**实施：** 系统拼音候选翻页/Enter/Esc/混输、切窗/对话框；VoiceOver编辑/表格/树/诊断/图表精确替代/异步announcement。

**完成证据：** 每个场景记录实际操作与期望；测试hook与真人系统IME分开；AX可达性无阻断；失败有复现/修复与相关复验。

**当前进度：** 新增仅在 `darwin && orangecount_e2e` 下编译的真实 MyGo 窗口测试；AppKit Native Input Client 验证 BQL 编辑器中的 marked text 及 `é` 提交、emoji 前缀后 parser token 的 UTF-16错误选区与替换，Accessibility 验证控件、查询结果和错误诊断可达。完整原生窗口 gate 最新通过（63.926秒）；覆盖导航、真实现金流图键盘选期与精确 Accessibility help、日期/币种筛选与多页面切换、挂起读取期间继续编辑和导航、流水交易选择/分页（包含52,500笔账本的首页/第二页各100条及同一SnapshotID），唯一交易注释插入后恢复详情、重复交易不继承选择、账户树深层搜索及重载后恢复展开和选择、真实方向键父子导航与展开/折叠、900×600原生窗口中的长中文详情换行与精确小数呈现、嵌套指令长中文换行及NULL/bool/exact decimal保真、all/config 指令读取和日期范围隔离、20轮真实原生窗口创建/关闭与 workspace 状态清理、多窗口附件目录撤权及旧附件令牌失效、附件目录面板归属/取消、宽范围根目录确认取消与允许、隔离 picker 结果驱动的书签授权闭环、新 Model/Store 的 bookmark 恢复和失效 bookmark 重选、Legacy/V3 AVERAGE 原生预览和显式应用、多币种/缺失报价、空账本报表、查询执行及结果表 Home/End/Up/Down、数值排序和类型化复制、双账本预设隔离及重载期间旧查询目录隐藏、入口及 include 文件自动监视恢复、MyGo 账本文件打开 delegate 窗口路由、离线分析文件打开和原生窗口 Finder 风格拖放、查询取消终态Accessibility按钮稳定消失、查询执行挂起时关窗并丢弃迟到结果、双窗口并发关闭查询与保存面板、欢迎页最近账本须显式选择才打开、失效最近项诊断及后续文件打开、来源编辑器面板取消、报表/查询保存面板取消、偏好备份面板归属/取消与保存及报表/查询保存面板 owner 关闭清理、双窗口 Cmd+O/Open Panel 归属、Cmd+N 原生新窗和 Cmd+R 重读路由；查询/导航测试等待 Accessibility 树中的目标状态就绪。全套原生测试通过；图表、Unicode错误选区、双窗口 Find/Open/Reload 菜单、四项G09场景、两项G10筛选/导航场景、流水详情/分页、G18指令、G39测时、G28多账本重载、附件撤权、两个 picker 场景及 bookmark 恢复/失效重选场景、AVERAGE profile切换、离线分析打开/拖放、query面板关闭、capture挂起关窗和20窗口循环均按任务记录重复通过。没有覆盖真人拼音候选翻页/Enter/Esc、多窗口输入法切换或 VoiceOver 朗读，故G37仍进行中。详见[G37/G38 原生 MyGo 窗口端到端检查](evidence/goui-p6-g37-g38-native-e2e-2026-10-08.md)。

2026-10-08 补充：新增原生窗口 AppKit key-equivalent 检查 Cmd+Plus、Cmd+-、Cmd+0，并验证字号对应的 Accessibility frame 变化及偏好还原。当前完整 gate 为63个原生用例通过、3个条件门控用例跳过。真人拼音候选导航、多窗口输入法切换与 VoiceOver 朗读仍待验收，G37 保持进行中。

<a id="g38"></a>

#### G38 — 新Go ui全功能GUI与等价矩阵

**依赖：** G36。**对应要求：** R02,R03,R04,R05,R06,R08,R09,R10,R15。**职责：** 产品验收。

**修改位置：** apps/goui/tests/gui_scenarios；docs/evidence/goui-p6-<date>.md。

**实施：** 迁移旧18场景并扩展query/附件/导出/全指令；使用独立QA bundle和公开fixture，分实际UI与core/headless证据。

**完成证据：** 所有适用R要求有明确结果；金额/计数/source/上下文吻合独立期望；没有用截图存在或进程存活替代实际操作。

**当前进度：** 最新完整原生窗口 gate 通过（81.641秒；5个 opt-in 场景跳过），新增无效 Legacy→V3恢复原生场景；覆盖 dashboard、现金流图真实窗口键盘选期与 Accessibility help精确值、Unicode查询错误选区和替换、日期/币种筛选与财务概览/资产负债表/损益表/流水切换、挂起读取期间仍可修改日期/币种与导航、流水交易选择/分页（105笔及52,500笔账本），唯一交易注释插入后恢复详情且重复交易不继承选择、账户树深层搜索及重载后恢复展开和选择、真实方向键父子导航与展开/折叠、900×600窗口长中文换行和精确金额、嵌套指令列表的中文换行与typed值、all/config指令读取与配置日期隔离、20轮真实原生窗口创建/关闭和workspace关闭状态检查、双账本不同报表筛选与窗口隔离、关闭后20轮重开（详见G38双账本隔离证据）；两窗口共享附件目录授权撤销与旧附件令牌失效、附件目录原生 Open Panel 所属窗口与取消无副作用、隔离 picker 结果驱动的真实 bookmark/Session policy/偏好持久化闭环、新 Model/Store 启动后的 bookmark 恢复与失效 bookmark 原路径重选闭环、AVERAGE Legacy/V3 原生预览与profile应用、空账本损益表、多币种余额与缺失报价、BQL 查询及 900×600/150% 字号下的结果表键盘导航/排序/类型化复制、双账本预设隔离及重载期间隐藏旧 snapshot 查询目录、首次打开无效账本隔离及精确诊断、连续语法与不平衡错误诊断更新、旧快照保留及修复恢复、include 删除和重建后自动恢复、切换 include 目标后显示新快照、MyGo 文件打开 delegate 将账本送入欢迎窗口/新窗口并保留已有工作区、离线分析文件 delegate 打开及窗口 Finder 风格拖放、查询取消终态Accessibility按钮稳定消失、查询执行挂起时关窗并丢弃迟到结果、双窗口并发关闭查询与保存面板、失效最近项诊断和后续文件打开、来源编辑器面板归属/取消、宽范围根目录确认取消与允许、报表保存面板双窗口归属/取消及关窗清理、查询保存面板归属/取消与 owner 关闭清理、偏好备份菜单的面板归属/取消与备份保存、双窗口 Cmd+O 面板归属、Cmd+N 原生新窗与 Cmd+R 重读路由。测试均使用公开 fixture 或临时文件；`make goui-native-e2e` 全套通过，附件撤权、Open Panel 取消与隔离 picker 结果授权场景、AVERAGE profile 应用、G28多账本重载和query Save Panel owner 关闭及挂起 capture 关窗场景各另连续5次通过，图表、Unicode错误选区、四项G09场景、两项G10筛选/导航场景和偏好备份面板取消/保存场景各重复5次或10次，流水详情/105笔分页、报表/查询面板取消和报表关窗场景各重复10次，52,500笔分页、G17长中文、G18全类指令及嵌套值场景各连续5次通过，离线分析 delegate 打开与窗口拖放场景连续3次通过，query execution owner close 与双窗口 query/Save Panel 并发关闭场景各连续5次通过，20窗口关闭循环连续运行3轮通过。旧版18项行为等价矩阵、语义配置与全指令完整可用性仍未全部跑完；G38继续进行中。详见[G14 AVERAGE 原生窗口验证](evidence/goui-p2-g14-native-average-2026-10-08.md)、[G35 多窗口撤权与目录面板验证](evidence/goui-p5-g35-multiwindow-2026-10-08.md)、[G37/G38 原生 MyGo 窗口端到端检查](evidence/goui-p6-g37-g38-native-e2e-2026-10-08.md)、[G38连续错误与修复原生验证](evidence/goui-p6-g38-invalid-updates-native-2026-10-08.md)、[G38 include 删除与替换原生验证](evidence/goui-p6-g38-include-replacement-native-2026-10-08.md)、[G38 初次打开无效账本原生验证](evidence/goui-p6-g38-initial-invalid-native-2026-10-08.md)、[G38账户树方向键导航原生验证](evidence/goui-p6-g38-accounttree-keyboard-native-2026-10-08.md)、[G38双账本隔离验证](evidence/goui-p6-g38-multiwindow-isolation-2026-10-08.md)及[G29 Finder/拖放路由记录](evidence/goui-p4-g29-native-open-drop-2026-10-08.md)。

2026-10-08 页面原生验收补充：增加账户详情往返、持仓账户筛选/平均成本/批次、商品声明/原生 USD 报价选择三个真实页面场景；这些场景各连续3次通过，并纳入最新完整 native gate。按账户行选择后现在显示独立 AX 详情动作，报价菜单项按节点身份选择，避免表格中同名 USD 标签造成误点。随后增加真实附件目录 NSOpenPanel 所属窗口/取消无副作用和隔离 picker 返回路径驱动的授权成功场景，各连续5次通过；再新增新 Model/Store bookmark 恢复与失效 bookmark 同路径重选场景，两项组合连续5轮并通过 race。另在隔离 Manual QA `.app` 中通过原生面板实际选择临时附件目录，附件显示“可用”；退出并重启该 QA 包、重新打开账本后仍为“可用”，随后移除授权并清理本次测试偏好和临时文件。本轮完整原生 gate 的61个 TestNative 用例全部通过、3个环境门控用例跳过，账户树深层搜索及重载后恢复展开和选择、真实方向键父子导航与展开/折叠的场景另连续5次通过，连续错误诊断与自动修复场景重复5次并通过 race，include 删除、恢复与目标替换场景也重复5次并通过 race；真实 VoiceOver、真人 IME、旧版18项完整等价矩阵、development 包完整 GUI 读写循环和正式签名发行仍未完成，因此 G35/G37/G38/G43 状态保持进行中。完整结果及 arm64/amd64 开发包 SHA-256 见[G21/G22/G23原生页面补充](evidence/goui-p6-g21-g23-native-pages-2026-10-08.md)和[G35附件面板补充](evidence/goui-p5-g35-multiwindow-2026-10-08.md)。

2026-10-08 新增旧场景13首次无效打开验证：新窗口无有效快照，原生诊断显示 `E-PARSE-STRING` 与 `invalid.bean:1:40`，相邻已打开账本窗口的 snapshot/epoch/金额不变；定向连续5次和 race 通过。更新后 `make goui-check` 与完整 `make goui-native-e2e` 通过（61个 TestNative 通过，3个环境门控场景跳过）。详见[G38初次打开无效账本原生验证](evidence/goui-p6-g38-initial-invalid-native-2026-10-08.md)。

2026-10-08 新增旧场景15账户树键盘验证：真实点击稳定 AX 行后，方向键完成父子选择与分支展开/折叠；定向连续5次和 race 通过。更新后 `make goui-check` 与完整 `make goui-native-e2e` 通过（62个 TestNative 通过，3个环境门控场景跳过）。详见[G38账户树方向键导航原生验证](evidence/goui-p6-g38-accounttree-keyboard-native-2026-10-08.md)。

2026-10-08 新增旧场景8成本与价格注释原生验证：在真实流水详情中核对 `{2 USD}` 成本、`@ 0.9 EUR` 单位价格、`@@ 180 EUR` 总价格及三处准确来源行列；分别激活来源按钮并确认它路由到所选 fixture 路径。来源按钮的 Accessibility 标签现包含对应位置，便于辅助技术识别重复分录动作。三种场景组合连续5轮及定向 race 通过，`make goui-check` 与完整 `make goui-native-e2e` 均通过（77.688秒）。测试回调验证路径路由，不模拟 Finder 窗口；G38 其余等价矩阵仍进行中。详见[G38成本、价格与来源原生验证](evidence/goui-p6-g38-journal-cost-price-source-native-2026-10-08.md)。

2026-10-08 扩展旧场景9到同一份52,500笔真实窗口账本：全量页每页100笔；随后日期/账户/文字筛选计数依次为13,125、3,281、656，升序后检查两页的逐行交易编号、账户、日期、跨页唯一ID和同一SnapshotID。该组合先发现从第二页应用日期筛选会错误保留 `offset=100`；现已在流水页应用阅读筛选时重置 offset。定向场景连续5轮和 race 通过，`make goui-check` 与完整 `make goui-native-e2e` 通过（77.711秒）。详见[G38大账本筛选与分页原生验证](evidence/goui-p6-g38-large-journal-filters-native-2026-10-08.md)。

2026-10-08 新增旧场景3–5原生期望值核对：公开 cashflow fixture 的 dashboard 为收入1000、支出300、净额700 USD；1月2日余额截止后 Assets 为1000 USD；只看1月3日时损益为收入0、支出300、净额−300 USD，净资产仍为700 USD。场景连续5轮和 race 通过。与此同时修正深浅色像素测试：此前只确认 MyGo 主题状态，未确认 Content 已绘出浅色帧，偶发把旧暗色帧当作基线；现在以实际亮度阈值确认浅色像素后再比较暗色，连续10轮和 race 通过。最新 `make goui-check` 与完整 `make goui-native-e2e` 通过（78.646秒）。详见[现金流 cutoff/期间核对](evidence/goui-p6-g38-cashflow-cutoff-native-2026-10-08.md)及[深浅色像素稳定性补充](evidence/goui-p6-g38-appearance-pixels-2026-10-08.md)。

2026-10-08 G43 当前源码包结构复验：为本轮 dirty tree 生成仓库外 arm64 development `.app`、ZIP 与 SHA-256；`make goui-package-verify` 核对 plist、Mach-O架构/minOS、CGO/Swift linkage、ZIP内容、build-info 和依赖许可证清单通过。版本 `0.0.0-g38-current`，SHA-256 `9231f8c85cec69e536921a620f33afd109724437c431cf840fc588107f1a425b`，source revision `1c2096d435e07cc9b5a5957e0e974546ee1387a6`，build-info 记为 dirty。可见启动尝试时，CUA 按相同 bundle ID 连接到了此前已运行的另一份 development app，因此没有把该窗口记作新包 GUI 验收；没有选择或打开最近列表中的账本。新包的实际可见欢迎页/读账本/完整流程仍待 G43 验收。

2026-10-08 当前源码包隔离 QA GUI 补充：复制上述已校验 arm64 `.app` 到一次性目录，只更改副本 Bundle ID/名称，使用 QA 和唯一 user-data 身份启动包内原 executable。真实窗口显示 QA 首屏不自动恢复最近账本；经原生 Open Panel 明确打开公开 `cashflow.bean` 后显示 3 个账户、2 笔交易、5 条分录，以及 `700 USD` 净资产、`1000/300 USD` 收入/支出和 `700 USD` 净流量。Cmd+Q 后 CUA 报告该隔离应用已退出，且唯一 executable 路径无存活进程。该记录修正了先前“包 GUI 尚未验证”的状态，但只覆盖 QA 冷启动与读取；不借用 r7 的查询/导出/重启证据，也未覆盖旧偏好迁移。详见[G43 当前源码包隔离 QA GUI 验收](evidence/goui-p7-g43-current-package-qa-gui-2026-10-08.md)。

2026-10-08 新增旧场景17字号验证：测试菜单与 Cmd+Plus/Cmd+-/Cmd+0 的真实 AppKit key-equivalent，确认 900×600 原生窗口中行高随 100%/125%/150% 增长、搜索焦点可达、偏好按账本保存并在重新打开后恢复。定向连续5次通过；更新后 `make goui-check` 与完整 `make goui-native-e2e` 通过（64个 TestNative 通过，3个环境门控场景跳过）。旧场景17 Light/Dark 切换后的像素颜色、其他尺寸和完整辅助功能矩阵，以及旧场景16/18尚未闭环的外部条件仍由 G37/G38/G43/G44 跟踪。详见[G36 每账本字号与快捷键验证](evidence/goui-p6-g36-text-scale-2026-10-08.md)。

2026-10-08 新增旧场景16双账本过滤隔离验证：两个原生窗口分别打开不同公开 fixture 副本，独立筛选账户行；关闭第一窗口后第二窗口仍保留原路径、snapshot、资产负债表与筛选。继续20轮创建/读取/关闭临时账本窗口，每轮确认 workspace 释放且稳定窗口未变；场景连续3次及 race 通过。当前完整 gate 为64个原生用例通过、3个环境门控用例跳过。详见[G38双账本隔离验证](evidence/goui-p6-g38-multiwindow-isolation-2026-10-08.md)。

2026-10-08 补充旧场景17深浅色像素验证：900×600 真实 MyGo Content 窗口中，Light/Dark 的平均绘制亮度为241.2/40.6，1536个均匀采样点有1518个 RGB 差异超过阈值；场景定向运行3次通过。完整 `make goui-native-e2e` 最新通过（65.608秒），`make goui-check` 通过。该结果补上此前记录的像素颜色缺口，但不代表显示器色域校准、其他尺寸完整布局或真人 VoiceOver。复验还发现查询复制测试会因读取已有剪贴板内容而提前结束，现使用剪贴板哨兵并等待当前 typed JSON 精确结果，查询复制场景连续5次及 appearance/query 两场景 race 通过。证据见[G38 深浅色原生渲染像素验证](evidence/goui-p6-g38-appearance-pixels-2026-10-08.md)。

2026-10-08 新增真实窗口指令类型筛选、搜索、升序和分页场景：用临时公开账本生成103条备注（101条匹配、2条不匹配），实际操作类型菜单、搜索框及分页按钮；确认筛选前103条、搜索后101条、100+1分页边界、来源行号5–105和snapshot身份在搜索/翻页往返中稳定。该场景定向原生测试及race均通过；随后 `make goui-check` 与完整 `make goui-native-e2e` 通过（67.905秒）。这补充G18/G38的一条交互路径，不替代旧版18场景全量等价及 IME/VoiceOver 验收。详见[G38 指令筛选、搜索与分页原生验证](evidence/goui-p6-g38-directive-search-pagination-native-2026-10-08.md)。

2026-10-08 补充旧场景9流水筛选组合：在105笔临时账本的真实窗口中操作日期、账户、文字搜索和升序控件，匹配数依次为35、17、9，首尾交易身份正确且SnapshotID稳定；筛选场景连续5次及三场景race检查通过。同步缩小并修正账户树清除搜索原生回归：原坐标点击hook只报告命中AX界限、不能证明回调执行，现改用核实performed结果的AXPress helper；账户树最小清除场景连续5次及包含reload/选中的原场景通过。`make goui-check`与完整`make goui-native-e2e`通过（71.123秒）。完整旧18场景等价矩阵和真人 IME/VoiceOver仍未完成，G38/G37保持进行中。详见[G38流水日期/账户/搜索/排序和账户树清除验证](evidence/goui-p6-g38-journal-filters-native-2026-10-08.md)。

2026-10-08 G38/G14 最新复验：初次 Legacy 无效账本的 profile 预览门控已修复。G14 invalid-input MyGo 场景连续3次通过，`GOWORK=off go test -race ./internal/app -run '^TestSemantics' -count=5` 与 `make goui-check` 通过；完整 native gate 通过（81.641秒）。旧版18项完整等价矩阵、真人 IME/VoiceOver及5个 opt-in 环境场景仍未验收。详见[G14恢复记录](evidence/goui-p2-g14-invalid-legacy-recovery-2026-10-08.md)与[G37/G38全量原生结果](evidence/goui-p6-g37-g38-native-e2e-2026-10-08.md)。

<a id="g39"></a>

#### G39 — 性能基准、帧与空闲核验

**依赖：** G36,G01。**对应要求：** R01,R13。**职责：** 性能验收。

**修改位置：** apps/goui/tests/benchmarks；benchmark.sh；MYGO_FRAME_STATS测量adapter。

**实施：** 同机同输入prodflags比较欢迎/52,500加载、操作/滚动/10k chart/cancel/RSS/idle；区分engine/view-layout/paint/end-to-end时间。

**完成证据：** 设计第12节门槛以当前真实样本判断；报告p95/p99/样本数和范围；FRAME_STATS不冒充输入到present延迟；无持续idle重建/文件内容读取。

**当前进度：** 新增`benchmark.sh`与Go UI基准，分别计时核心打开/100行读取、Welcome/52,500账本交易页 headless 帧和10,000点图表；记录固定输入SHA、机器、Go/MyGo、profile、窗口与构建参数。Mac mini M4首轮样本显示核心打开p95约1.90s、读取100行p95约34.8ms；headless 10k图表p95约4.08–5.28ms。精确表格将每期间map改为按序列索引后，分配从约9.65MB/21,372次降至6.48MB/1,439次。新增5次52,500笔临时账本真实AppKit窗口计时：测试进程已运行时，从文件写完至仪表盘进入Accessibility树0.882–1.007秒，翻到前两页至AX页码更新46.8–70.6毫秒；这只是窗口/读取端到端样本，不含冷启动或屏幕present时延。原生短样本还包括公开小账本开发包（31秒，CPU均值0.28%、最大1.8%，RSS均值107.5 MiB）与隔离QA包打开52,500笔账本后（31个每秒样本，CPU均值0.20%、最大1.6%，RSS稳定682.2 MiB）；后者采样期间状态均为sleeping。`fs_usage`因需root未运行，因此不能证明空闲期间无文件读取；冷启动、真实滚动/绘制帧、取消反馈、长时RSS平台期和SwiftUI同输入对照也仍待测。headless绘制不代表Metal输入到present时延，G39继续进行中。证据见[G39记录](evidence/goui-p6-g39-performance-2026-10-08.md)。

<a id="g40"></a>

#### G40 — 资源释放与持续重载观察

**依赖：** G38,G08。**对应要求：** R03,R04,R13。**职责：** 生命周期验收。

**修改位置：** apps/goui/tests/lifecycle、process_soak；workspace计数探针。

**实施：** 20次窗口/账本切换、60次reload/10分钟、query/capture/面板并行关闭；观察session/runner/timer/listener数量和RSS平台期。

**完成证据：** 后台session/poll/query/panel计数回零；无无限任务/RSS线性增长；不可取消Open/Build完成后也释放；必要race与实际process证据。

**当前进度：** 新增真实 `DesktopEngine` 生命周期 soak：使用公开 `cashflow.bean` fixture、短间隔 watcher 连续打开/关闭20个独立 workspace，每次都等待 `Controller.Done()` 并确认最终状态不保留响应/捕获；另在单 workspace 内顺序执行60次 reload，逐次确认仍处于同一有效 snapshot，最后等待关闭完成。原生 E2E 新增20轮真实 Content 窗口/账本打开关闭，每轮检查Accessibility仪表盘、等待 workspace 完全关闭并确认无已发布 response/query/capture、MyGo注册窗口数恢复基线；每轮GC后读取当前测试进程RSS与Go HeapAlloc。首次缓存升温后，同一进程额外40轮RSS变化分别约+336 KiB、+48 KiB，HeapAlloc约1.4–1.85 MiB；后半段增长阈值检查通过。另新增独立 `make goui-native-soak`：真实窗口每10秒 reload 一次、共60次、持续10分钟并按分钟采集RSS/HeapAlloc；本机样本预热后第2–10分钟RSS下降7,664 KiB、HeapAlloc增加34,416 bytes，Dashboard SnapshotID/交易数始终一致。新增原生 query Save Panel 关闭测试，确认冻结捕获随 owner window 关闭释放；另在 Save Panel 打开前挂起 query capture 并关闭 owner，确认 workspace 等待操作结束后丢弃结果；新增原生测试还将 ExecuteQuery 挂起后关闭 owner，确认 workspace 等待底层调用并丢弃迟到响应；前三个单项关闭场景各连续5次通过；双窗口中一个查询执行挂起、另一个冻结捕获并显示 Save Panel 时同时关闭两个 owner 的场景也连续5次通过。MyGo `windowHandler.Closed()` 现会在 `OnClosed` 回调执行后清空 Window/Page 所有事件监听器，避免闭包继续持有已关闭 model/workspace；`TestClosedWindowReleasesEventListeners` 连续5次通过。新增 `ui.E2EHostLifecycle` 与原生 Surface 计数钩子（仅 opt-in E2E 编译），实测空闲窗口活动 idle/frame timer 和运行中 display link 为零，关闭后 host/surface/link/timer 数回到基线；原生计数场景连续5次通过，`TestDetachStopsAndForgetsIdleTimer` 连续5次通过。上述改动后 `make goui-check`、完整 `make goui-native-e2e` 与 MyGo 可运行的22个包测试通过。完整 MyGo `./...` 仍被 vendored checkout 缺少 `plugins/fetch`、`packages/cli/index.d.ts` 和 `node_modules/mygo-cli` 阻断。诊断发现首版循环把 `t.Cleanup` 闭包留到测试结束造成假增长；改为局部defer并清空引用后，后续窗口循环稳定。持续空闲期间的账本文件内容读取仍待验，G40继续进行中。详见[G40 workspace及原生窗口生命周期记录](evidence/goui-p6-g40-2026-10-08.md)。

2026-10-08 字号集成复验：在 MyGo `Context.SetTextScale` 改动后重新执行 `make goui-native-soak`，60次真实窗口 reload 和十分钟 snapshot 校验通过；第2–10分钟 RSS 变化 −9,408 KiB，HeapAlloc 增长37,392 bytes，RSS范围112,624–122,832 KiB。持续空闲期间系统级账本文件读取归因仍未有 `fs_usage` 证据，因此 G40 继续进行中；完整样本见[G40生命周期记录](evidence/goui-p6-g40-2026-10-08.md)。

本轮新增 `TestWatchLoadsSourceGraphOnceWhileIdle`，以计数 loader 验证稳定账本在10个5ms watcher interval 内仅执行一次完整 graph load，定向 `-count=20` 通过；随后根 `go test ./...`、snapshot race、`go vet ./...`、`make goui-check` 和完整 `make goui-native-e2e` 均通过。这个单测证明 watcher 不重复调用图加载器，不替代操作系统层的进程文件读取归因。持续空闲期间的账本内容读取仍待 `fs_usage` 级验收；G40继续进行中。

<a id="g41"></a>

#### G41 — 当前核心与新UI私人只读对照

**依赖：** G38。**对应要求：** R02,R06,R15。**职责：** 等价验收。

**修改位置：** apps/goui/tests/private_read；goui-private-read；tools/reference既有入口。

**实施：** 公开输入先通过；按外部只读账本入口跑同源Go/reference与新workspace/ui-ready投影，记录实际snapshot/profile和核心功能结果。

**完成证据：** 金额/库存/计数/截止与query/export相关投影一致，私人内容不入仓库；新Go ui链路单列，旧Swift/C结果不能替代。输入缺失时具体记录而不伪造通过。

**当前进度：** 新增 `make goui-private-read` opt-in harness；要求绝对外部账本路径及显式 legacy/v3 profile，分别打开核心 Session 与 Go UI `DesktopEngine`，校验同一 snapshot/profile，并对dashboard、资产负债表、损益表、流水、商品/价格、持仓、全部指令、附件、语义配置和查询目录逐页比较完整 `ViewResponse` 后由MyGo headless UI实际渲染标题。固定 BQL 的 typed query response 完整匹配且结果列呈现，query CSV、`.ocanalysis`、流水报表 CSV 捕获字节逐一相同。对比只规范化空 `ActiveExtensions` 的nil/empty表示；日志仅输出视图/导出ID、格式、字节数、profile、snapshot单向摘要和节点数，不输出路径、文本、金额或诊断。公开 `cashflow.bean`（v3）测试覆盖10页和三种导出；另新增仅 opt-in 的 Darwin wrapper，从生产 defaults 最近列表选择一份可读账本，运行 G41 完整只读对照。本机 `legacy` profile 实测10页、typed query、query CSV、`.ocanalysis` 和 journal report CSV 全部核心/Go UI 匹配，导出逐字节一致；日志不含路径或账目内容。没有修改账本或用户偏好。G41 的私人输入验收已通过，但按任务依赖仍等待未完成的 G38 全 GUI 等价矩阵。证据见[G41生产最近账本只读验收](evidence/goui-p6-g41-harness-2026-10-08.md)。

### P7 — 打包与接管

<a id="g42"></a>

#### G42 — 生产构建、许可证与独立CI

**依赖：** G02,G36。**对应要求：** R01,R14。**职责：** 构建发行。

**修改位置：** apps/goui/build.sh、test.sh、package.sh、resources/licenses；Makefile；.github/workflows/goui.yml。

**实施：** 固定prodflags/CGO_ENABLED=0、Mach-O/plist最低13匹配、依赖/fork/fonts/icons许可证manifest；root/nested gates分开，CI记录实际工具链。

**完成证据：** 完整desktop测试+相关race、核心冻结gate和prod build通过；无C archive/Swift inputs；bundle许可证与实际依赖匹配；远端CI只按真实run记结果。

**当前进度：** `build.sh` 固定 GOOS=darwin、CGO_ENABLED=0、trimpath 和空 Go build ID；新增开发 `.app`/ZIP/sha256 打包脚本，检查 Mach-O 与 `Info.plist` 最低 macOS 13.0一致，写入来源 revision/dirty、Go/MyGo版本和无签名标记。包内会按 resolved module graph 收集每个依赖的许可证，包含Go工具链/Apache NOTICE；当前未捆绑第三方字体或图标。新增 macOS 15 独立 Go UI workflow，包含核心冻结gate、UI race/vet、生产构建及临时 development package。当前本机 arm64 包与交叉构建的 amd64 包均经校验（Mach-O minos、plist、许可证目录和ZIP hash通过）；新增 verify-package.sh 与 make goui-package-verify 检查 ZIP 哈希、解压 bundle、Mach-O 架构与 minimum、CGO和Swift linkage、build-info，以及许可证清单与resolved module graph的一致性，且CI已接入该校验；远端workflow尚无run结果，未完成G36前也不声明G42验收结束。证据见[G42构建与包记录](evidence/goui-p7-g42-2026-10-07.md)、[G42开发包完整性校验](evidence/goui-p7-g42-package-verifier-2026-10-08.md)。

2026-10-08 补充：在加入 MyGo 原生字号 API 与场景16/17回归后，重新构建当前 dirty tree 的 arm64 与 amd64 development 包；两者均经 `make goui-package-verify` 校验。ZIP SHA-256 分别为 arm64 `1159494f25609854c1224686c52d80658e5c44962867759dd60221e389aa9377`、amd64 `59f28452b38b31fa42b7be45b6ee374581c5bf633a947e634dc21bd9d86bb866`。本轮只校验包结构，未在这份新包上重跑可见 GUI 全循环。`.github/workflows/goui.yml` 不在 GitHub 默认分支，远端查询返回 404；当前没有推送，因此没有远端 CI run。本地用 actionlint v1.7.7 校验 workflow 通过，Ruby YAML 解析也通过。详见[G42/G43当前源码包校验](evidence/goui-p7-g42-g43-current-package-2026-10-08.md)。

<a id="g43"></a>

#### G43 — 完整开发包与安装交付

**依赖：** G37,G38,G39,G40,G41,G42。**对应要求：** R06,R11,R14,R15。**职责：** 交付。

**修改位置：** dist/goui；apps/goui/package.sh、prepare-qa.sh；开发版使用说明。

**实施：** 生成App/ZIP/SHA-256/build-info；仓库外临时目录解压，欢迎→打开→错误→恢复→query/export→退出；旧偏好迁移和dev/QA隔离。

**完成证据：** 实际产物可启动并完成阅读循环；开发等级/目标系统限制明确；提供文件hash、当前源码身份与使用说明；旧用户App未覆盖。

**当前进度：** 已在仓库外临时目录生成独立 arm64 development `.app`、ZIP 与 SHA-256；解压后核对 bundle identity、最低 macOS 版本、构建信息与许可证目录，并从该实际打包产物打开公开 `cashflow.bean`，窗口显示 3 个账户、2 笔交易和 5 条分录。另从同一打包进程分别打开语法错误账本和有效公开账本，实际观察到解析诊断及有效账本页面；这验证的是分开的文件打开事件。现在新增真实 `DesktopEngine` + MyGo headless UI 的同 workspace 错误→诊断→页面 reload→修复报表闭环，验证旧窗口/epoch延续和更新金额渲染；它不替代打包 app 的真实 GUI 流程。新增[Go UI开发版使用说明](../apps/goui/README.md)，记录构建/测试/QA/打包/校验命令、dev/QA数据身份与未签名限制；根 README 已更新为实际开发状态并指向开发指南。包含最新查询取消界面的 development `.app`/ZIP/SHA 也已在 `dist/goui/` 生成，plist、最低系统版本、build-info、ZIP和checksum通过核对；G13离线导航修复后又生成 `/tmp/orangecount-goui-offline-nav/` arm64 development 包，ZIP SHA-256 `c0e8676ee8cb9def16201f83baf34112225386cd8ae51468e0631a6fc5c4b4a6`，plist、ZIP结构和checksum通过核对。此前 ScreenCaptureKit/辅助功能受限的旧构建尝试不计入 GUI 证据；2026-10-08 已在本机新 arm64 development bundle 中完成真实界面读账本、运行 BQL、保存并读回查询 CSV、保存流水报表 CSV 与报表分析包，并通过应用原生打开面板重新载入查询分析文件，界面确认捕获结果未重新执行；同一打包进程另在隔离临时账本上验证语法错误时保留有效快照、恢复公开内容后自动回到无诊断有效状态。Cmd+Q 后窗口返回欢迎页，但应用清单仍显示运行中，故本轮不把进程退出记为已验证。新包版本 `0.0.0-g43`，ZIP SHA-256 `307770defb1956b0128a1d29b2b94aeafd7d73f068f9fa2e874b0c813c08f03c`，当前 `make goui-package-verify OUT_DIR=/tmp/orangecount-goui-g43-gui-20261008 VERSION=0.0.0-g43 GOARCH=arm64` 通过。旧偏好迁移和 dev/QA 隔离完整矩阵、生命周期全项及 G37/G38 IME、VoiceOver、旧版全 GUI 等价矩阵仍未完成，G43 继续进行中。详见[G43开发包实际 GUI 导出与读回闭环](evidence/goui-p7-g43-gui-export-loop-2026-10-08.md)、[G42/G43开发包启动验证](evidence/goui-p7-g42-g43-2026-10-08.md)和[G19修复恢复闭环](evidence/goui-p2-g19-recovery-2026-10-08.md)。

2026-10-08 Cmd+Q 退出补充：新增原生子进程 E2E，使用 OrangeCount 应用菜单通过 AppKit Cmd+Q key-equivalent 退出；测试要求 `App.Run` 返回后 helper 的 `TestMain` 正常返回，子进程以0状态退出。定向连续5次及 race 检查通过；随后 `make goui-check` 与完整 `make goui-native-e2e` 通过。当前源码加入 G27 原生查询排序、键盘导航、类型化复制及 900×600/150% 字号窗口回归后，完整 `make goui-native-e2e` 再次通过（66.217秒），其中 Cmd+Q 子进程场景一并通过。先前可见打包 `.app` 仍运行的观察尚未在当前源码新包上重跑，因此自动化证据不替代 G43 最新包 GUI 退出验收。详见[Cmd+Q 原生进程退出验证](evidence/goui-p7-g43-cmdq-process-exit-2026-10-08.md)。

2026-10-08 当前源码包补充：以 `0.0.0-goui-dev-20261008` 分别生成 arm64 和 amd64 `.app`/ZIP；两包均通过完整 `make goui-package-verify`，核对架构、macOS 13.0 minimum、CGO/Swift linkage、build-info、ZIP 内容与 module-license inventory。arm64 ZIP SHA-256 为 `ba6b094bbc54da8e17cbfad73b45c58f1d7d331138ba4517b7403575a9920446`，amd64 为 `4a8bfd8bddf73d68cfba545e1c5442407950f894e752ba87df7ca440d74e7e9c`；产物保存在 `/tmp/orangecount-goui-current-arm64-20261008/` 和 `/tmp/orangecount-goui-current-amd64-20261008/`。此次没有在新包上重跑可见 GUI 闭环，G43 仍进行中。详见[G42/G43当前源码包校验](evidence/goui-p7-g42-g43-current-package-2026-10-08.md)。


2026-10-08 启动 Store 组装变更后再次从当前 dirty tree 打包，版本 `0.0.0-goui-dev-20261008-r2`：arm64 ZIP SHA-256 `d4a69ed3f2d75bd62b8a00d15220a6053f0459b8822aafaee9ebcc012bc32a68`，amd64 `b83c4d688ca0ce431c71b2771a0c8daecf133947957e3c8cad75af4d6ab5af43`。两包均经完整 package verifier 校验；本轮未启动 bundle 做可见 GUI 循环，G43 继续进行中。详见[G42/G43当前源码包校验](evidence/goui-p7-g42-g43-current-package-2026-10-08.md)。

2026-10-08 启动偏好失败可见性更新后重新生成版本 `0.0.0-goui-dev-20261008-r3` 的当前 dirty tree development packages：arm64 ZIP SHA-256 `1a7454bde98c32426e17c21386fcc722602359e591de24131f9053341ef05798`，amd64 `1f214a3300dc560eac9791b5f8c29880ab18379ed80f4dfde7db7b8da1db8bcb`。两包的 plist、ZIP、架构、macOS 13.0 minimum、CGO/Swift linkage、build-info 与模块许可证清单均经 package verifier 检查通过；没有从包启动可见 GUI，G43 继续进行中。界面错误路径测试见[G12启动错误显示](evidence/goui-p1-g12-startup-error-2026-10-08.md)。

2026-10-08 G16 折算余额显示更新后生成版本 `0.0.0-goui-dev-20261008-r4` 的当前 dirty tree arm64/amd64 development packages：arm64 ZIP SHA-256 `6604d9a95f33e4e352737583469d7f2d7836e8ec8b26b12ebca204cd6c7e5042`，amd64 `589afe0acfac5f9c9dd8245595f27dcb0fcede75d0df8b1f51777917cd66540d9`。两包均通过 package verifier 的 plist、ZIP 内容/hash、架构、macOS 13.0 minimum、CGO/Swift linkage、build-info 与 resolved module license inventory 检查。未从新 bundle 启动可见 GUI，G43 仍需当前包的 GUI 闭环。详见[G16自身余额与折算估值显示](goui-p2-g16-own-subtree-valuation-2026-10-08.md)。

2026-10-08 G43 r6 补测：在当前 dirty tree 的 arm64 包隔离副本中运行开发身份，使用 owner-only 的独立 user-data 路径且未启用 QA。可见 GUI 完成公开 cashflow fixture 阅读、typed BQL 结果、query CSV/`.ocanalysis` 保存及离线读回、balance sheet 与 journal CSV/报表归档保存、同窗口解析错误与自动恢复；Cmd+Q 对应进程退出状态为 0。arm64/amd64 ZIP 均重新通过 package verifier。隔离副本只更改 `Info.plist` 中的 bundle id/name，executable 来自原 r6 包；一份额外全新 user-data 冷启动的 CUA 观察被 ScreenCaptureKit `-3811` 阻断，故重启、旧偏好迁移和 dev/QA 完整隔离矩阵仍待验收。详见[G43 r6 隔离开发包 GUI 闭环](evidence/goui-p7-g43-r6-isolated-development-gui-2026-10-08.md)和[G42/G43 当前源码开发包校验](evidence/goui-p7-g42-g43-current-package-2026-10-08.md)。

2026-10-08 G43 r7 打包补充：隔离 user-data override 现在拒绝最终目录符号链接，race 测试确认拒绝时不改共享目录或目标权限；定向 race、`go vet` 与 `make goui-check` 通过。新 arm64/amd64 r7 packages 均通过完整 verifier，ZIP SHA-256 分别为 `7c960e378683ea20eeda900a59a1c7493387a16bfa37cb968551859591d9584b` 和 `84c9d03497bdccd1547180941039c0c5f11022c150dd0aea84ffc568f378788a`。该窄改动后的 r7 包尚未重跑可见 GUI，r6 GUI 记录不冒充 r7 实测；G43 继续进行中。详见[G42/G43 当前源码开发包校验](evidence/goui-p7-g42-g43-current-package-2026-10-08.md)。

2026-10-08 G43 r7 可见 GUI 补充：在上述 r7 arm64 包的隔离副本中完成了实际窗口读账本、BQL 查询、CSV/`.ocanalysis` 导出与离线读回、流水和资产负债表导出、同窗口语法错误→恢复，以及 Cmd+Q→同一隔离 user-data 重启→进程退出码 0。重启欢迎页不自动打开最近账本，偏好恢复日期并需用户明确选择账本；测试数据仅写入 owner-only 临时目录。此证据关闭了“当前 r7 包 GUI 导出/重启/退出循环”缺口；旧偏好迁移和 dev/QA 隔离完整矩阵仍未覆盖，因此 G43 继续进行中。详见[G43 r7 隔离开发包 GUI 与重启闭环](evidence/goui-p7-g43-r7-isolated-development-gui-2026-10-08.md)。


2026-10-08 G43 holdings-fix 包 GUI 尝试：重新校验 `0.0.0-g38-holdings-fix` arm64 ZIP 通过，并以隔离 bundle 启动包内字节完全相同的 executable。CUA 可见 QA 欢迎页和原生 Open Panel；提交公开 holdings fixture 路径后，窗口 AX/截图读取持续遇到 ScreenCaptureKit `-3812`，无法确认账本是否打开或筛选结果，因此不计为 r1 GUI 通过。仅清理本轮临时进程；用户数据目录未产生偏好文件。G43 仍需要在窗口通道恢复后完成该包的读账本/筛选可见流程。详见[G43 持仓筛选修复包隔离 GUI 尝试](evidence/goui-p7-g43-holdings-fix-gui-attempt-2026-10-08.md)。

2026-10-08 G43 当前源码包补充：重新生成并校验版本 `0.0.0-g43-semantics-recovery` 的 arm64 包（ZIP SHA-256 `89c51ed0ba1de7d08aaebe905ed2cf43d2c0c4ae571a49aebba7936af4e2c3bd`）。隔离 bundle 的可见 GUI 确认无效 Legacy→V3 恢复、持仓截止 `10 STK / 100 USD` 与 `3 STK / 60 EUR`、按成本币种草稿保留 Apply 控件并提交成功；同一隔离偏好重启后恢复 V3 与截止日期。应用菜单退出后，CUA 应用清单短暂滞后，但唯一临时 executable 路径检查已确认进程退出。该记录本轮未覆盖原生 Open Panel、query/CSV/analysis export 和完整重启矩阵；先前 r7 包证据不转移到此版本，G43 仍进行中。详见[G43修复版包 GUI 证据](evidence/goui-p7-g43-semantics-recovery-visible-2026-10-08.md)。

2026-10-08 G43 修复版当前包导出补充：同一 `0.0.0-g43-semantics-recovery` arm64 包在隔离 QA bundle 中打开公开 `cashflow.bean`，保存并核验查询 CSV 与 typed `.ocanalysis`，再由查询页原生打开离线捕获，确认显示原查询/结果且没有重新执行。流水页保存 4 posting 行 CSV 与 format-2 journal `.ocanalysis`，归档含 11 列 schema、4 typed rows、2 source records。资产负债表和损益表分别显示 700 USD、收入 −1000 USD/支出 300 USD，并经原生保存面板导出 CSV，分别校验 2 行与 4 行。当前包另可见验证 `Assets:Bank` 账户详情、USD商品/无本地报价的价格历史、5条全部指令，以及附件空页/默认根目录；真实附件目录面板取消后无额外授权。查询离线 reader 使用 `ReadQueryAnalysis`，不接受 journal report archive；交叉尝试明确报 `invalid query analysis container`，不计作报表 reader 验收。Cmd+Q 后进程表确认无 OrangeCount 进程；包 verifier 重跑通过。旧偏好迁移、dev/QA 隔离全矩阵、G37/G38旧版等价及 IME/VoiceOver、G44签名公证仍未完成，G43 继续进行中。详见[G43修复版包导出与离线读回](evidence/goui-p7-g43-semantics-recovery-export-loop-2026-10-08.md)。

2026-10-08 G43 修复版开发/QA 隔离补充：同一当前 arm64 package executable 在不同 bundle ID、不同 owner-only user-data 目录下分别运行开发和 QA 身份。开发窗口没有 QA 标识；明确打开公开 fixture 后保存日期偏好，Cmd+Q/重启后欢迎页只列本轮最近项、不自动读账本，明确重开后恢复日期。QA 偏好仍保有另一份账本的 V3/截止日期；两份 JSON 为 `0600`、父目录 `0700`，最近列表及值没有交叉。对应 startup store fake-defaults/QA bypass/isolated-development tests 已存在；本次没有读取或改写生产 UserDefaults。开发包直接 executable 启动时记录到一条非阻塞 sandbox-extension 提示，故这不是 LaunchServices 安装或干净用户迁移验收。G43 仍待 production legacy upgrade、完整 dev/QA 矩阵、G37/G38等价与 IME/VoiceOver、G44签名公证。详见[G43修复版开发与QA偏好隔离复验](evidence/goui-p7-g43-semantics-recovery-dev-qa-isolation-2026-10-08.md)。

<a id="g44"></a>

#### G44 — 正式目标系统与签名发行门槛

**依赖：** G43。**对应要求：** R12,R14。**职责：** 发行验收。

**修改位置：** 正式bundle/安装证据；签名公证配置与使用说明。

**实施：** macOS13真实目标、另一台arm64机器安装；Developer ID hardened runtime/timestamp/notary/staple；Intel/universal仅另有范围时做。

**完成证据：** 实际目标运行与签名、公证、解压/升级证据齐备后才称正式发行；证书/设备缺失记录具体外部条件。local dev不因此伪装成正式包。

<a id="g45"></a>

#### G45 — 旧Swift/C验证能力迁移和退役清单

**依赖：** G43。**对应要求：** R15。**职责：** 迁移收尾。

**修改位置：** apps/macos/tests与internal/bridge/tools的用途映射；新tests；docs退役记录。

**实施：** 逐项映射Contract/State/Export/NativeClient/private probe到新Go接口+ui/GUI；识别bridge其他用途；有新替代后才删除对应冗余代码。

**完成证据：** 清单中每项删除有可执行替代/同场景新证据；有独立用途的旧bridge仍保留；旧可用标签/产物/源码hash可撤回；core行为未变。

**预审记录：** 当前已盘点 Swift Contract/State/Export/NativeClient/private-read 与 lifecycle/benchmark 测试在 Go 核心、workspace、MyGo headless 页面和保存适配器中的对应位置，并确认仓库 `internal/bridge` 仍服务旧 `apps/macos` 构建路径。G43 尚未完成，故本项表格状态保持“未开始”；没有删除旧测试、Swift/C 源码、bridge 或旧构建入口。详见[G45 迁移预审](evidence/goui-p7-g45-retirement-preflight-2026-10-08.md)。

<a id="g46"></a>

#### G46 — 默认入口接管、迁移与可撤回交付

**依赖：** G45。**对应要求：** R11,R14,R15。**职责：** 产品接管。

**修改位置：** Makefile默认mac入口；README/roadmap/support；稳定bundle身份/偏好；最终包。

**实施：** 接管macOS开发主版本，固定稳定身份与versioned偏好迁移；产出最终App/ZIP和撤回说明；formal等级还需G44，保留尚未验系统范围。

**完成证据：** 实际新入口从解压包正常运行；旧偏好/文件关联有迁移与撤回验证；完整本地功能gate通过。若G44未完成，最终标签与说明仍为development。

## 3. 每次实施的交接信息

交接记录包含：完成/部分实现的任务ID、当前core/desktop/MyGO/fork修订与dirty差异、已实际运行的针对性检查、未执行门槛与具体原因、App/ZIP/hash、下一可执行任务。变更只覆盖当前任务需要的文件；任务ID不因拆批改变。

不得把上轮MyGO原生Counter或WebView探针记为G03/G07/G38已验证；不得把旧SwiftUI gui/race/私有对照标成新客户端结果。开始实施时重新确认这些历史证据的源码/输入/环境身份。

## 4. 本地完成与正式发行的区别

G01–G43及G45–G46满足相应本地验收后，可以交付完整development等级的Go ui主版本。G44独立控制正式目标系统/另一机器/签名公证；未完成时整个正式发行范围保留未完成状态，不能以本地包替代。Intel/universal、Windows/Linux、自动更新、原生写账和App Store沙盒是后续范围，当前没有暗含任务。

2026-10-08 G38 持仓筛选补充：真实窗口核验 V3 账本截点下 `10 STK / 100 USD` 与 `3 STK / 60 EUR` 成本基础保持分离；fixture 中缺少直接 STK/USD 本地报价时，市价估值明确不可用且不猜 cross-rate。复现并修复筛选身份变化后旧响应提前返回、连 Apply 控件一起隐藏的问题；当前待应用状态仍保留控件并提示读取新结果。该 native 场景 race 连续5次通过，`make goui-check` 通过；本轮 `make goui-native-e2e` 有多项焦点/面板场景超时，已中止且不记为通过，最先失败的既有 holdings 场景单独重跑通过。详见[G38持仓筛选状态与成本币种原生验证](evidence/goui-p6-g38-holdings-filter-state-native-2026-10-08.md)。

2026-10-09 G38 旧场景15补验：opt-in 原生测试将临时 `.bean` 来源交给未运行的系统 Script Editor，确认目标进程由 Launch Services 启动，并在测试清理中退出；后续进程核查未发现残留的 OrangeCount 或 Script Editor。此项补足来源编辑器首次启动覆盖，真人 IME/VoiceOver及完整 18 项等价矩阵仍未完成。详见[G38 来源编辑器首次启动补验](evidence/goui-p6-g38-source-editor-2026-10-09.md)。

2026-10-09 G43 进程状态纠正：全进程检查发现两个来自临时 GUI 包、已空闲约18小时的 orphan 应用，CPU分别为0.0%和0.1–0.2%，采样均停在AppKit事件循环，且没有打开账本/偏好文件；已按PID发送SIGTERM并核实退出。这解释了“没有测试进程”与“仍有 GUI 应用进程”之间的区别。检查及清理详见[G43临时GUI进程审计](evidence/goui-p7-g43-process-audit-2026-10-09.md)。

2026-10-08 G43 同步包：持仓筛选修复后重新产出 `0.0.0-g38-holdings-fix` arm64 development `.app`/ZIP，完整 `make goui-package-verify` 通过，ZIP SHA-256 `7696c7fafb8bd4f5b2108b3aaa525ef304d9b3fb4b73b612580ec154780ab676`。该 r1 包尚未重做可见 GUI 闭环；先前隔离 QA GUI 记录对应修复前的 `0.0.0-g38-current` 包，不转嫁为 r1 GUI 证据。详见[G43 holdings-fix package 校验](evidence/goui-p7-g43-holdings-fix-package-2026-10-08.md)。

2026-10-08 G38 native helper 稳定性复验：发现并修复测试辅助函数把“AX action 允许”误当作“动作已执行”的问题；现在按节点身份调用并核验 AXPress performed，文本焦点也等待 AX 状态确认。持仓成本币种用例现通过实际原生 selector 菜单选择 `cost_currency`，定向 race `-count=5` 全通过；`make goui-check`、`git diff --check` 和完整 `./apps/goui/native-e2e.sh -v -count=1 -timeout=3m` 通过（78.210秒）。5 个 opt-in 场景跳过：十分钟 soak、两种外部编辑器启动、私人账本对照两项。Finder/拖放的 sandbox extension 警告未导致用例失败。旧版18项完整等价矩阵及真人 IME/VoiceOver仍未完成，G37/G38 保持进行中。详见[G38持仓筛选状态与成本币种原生验证](evidence/goui-p6-g38-holdings-filter-state-native-2026-10-08.md)与[G37/G38原生 MyGo 窗口端到端检查](evidence/goui-p6-g37-g38-native-e2e-2026-10-08.md)。

随后把仍有的4处 AXRow 直接 press 调用统一到按节点身份执行并确认 performed 的 helper。场景8来源注释、场景10唯一交易展开恢复/重复交易清理、长中文流水行及查询结果行选择组成的 race 矩阵 `-count=5` 全通过；`make goui-check`、`git diff --check` 与 gofmt 检查通过。该测试辅助清理后没有重复全量 gate，相关4个场景已在 race 模式定向复验。

继续补旧场景7：将多币种报表测试由直接改 `filterCurrency` 改为真实展开“原币”selector、选择 USD、确认待应用值及原币请求仍在，再点击“应用筛选”验证 USD request 与 EUR 缺价提示。USD 估值 selector 与持仓成本币种 selector 组合 race `-count=5` 全通过，`make goui-check`、`git diff --check` 和 gofmt 检查通过。此 test-only 修改后未重跑全量 native gate；最新完整 gate 78.210 秒证据指向此前源码，当前 selector 变更由上述定向测试验证。

同一验收继续将按日期汇率窗口场景中的直接 Model 日期/币种赋值改成真实 USD selector、AX 日期输入和“应用筛选”按钮；`TestNativeStatementShowsDatedConvertedAccountValues` race `-count=5` 通过，验证 2000-01-03 EUR/USD 来源和 2000-01-02 缺价解释。`make goui-check`、格式/空白检查通过；该 test-only 变更后全量 native gate 未重跑。

之后增加 USD draft 渲染等待，避免日期 AX 输入先于 selector 状态更新；调整 query reload 测试，检查旧 query 清空与新 snapshot 发布，不再依赖瞬时 Busy 帧。reload 场景 race `-count=10` 全通过。随后 `make goui-check` 和完整 `./apps/goui/native-e2e.sh -v -count=1 -timeout=3m` 通过（88.461秒）；5 个 opt-in 场景跳过（十分钟 soak、两种外部编辑器及两项私人账本对照）。Finder/拖放的 sandbox extension 警告未导致用例失败。旧版18项完整 GUI 等价矩阵、真人 IME/VoiceOver和 opt-in 外部环境仍未完成，G37/G38 保持进行中。此结果取代前述“test-only 修改后未跑全量 gate”的临时状态。
