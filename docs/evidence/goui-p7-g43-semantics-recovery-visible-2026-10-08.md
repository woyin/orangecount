# G43 修复版包的语义恢复与持仓筛选可见验收

日期：2026-10-08（Asia/Singapore）。在包含初次 Legacy 无效恢复修复的最新 arm64 development 包中，以隔离 QA bundle 实际操作了语义配置和持仓筛选。

## 包与隔离

- 来源 `.app`：`/tmp/orangecount-goui-g43-semantics-recovery-20261008/OrangeCount Go UI Development.app`。
- 版本：`0.0.0-g43-semantics-recovery`；bundle ID `org.orangecount.desktop.goui.dev`；arm64；最低 macOS 13.0；CGO disabled；未签名/未公证。
- ZIP SHA-256：`89c51ed0ba1de7d08aaebe905ed2cf43d2c0c4ae571a49aebba7936af4e2c3bd`。
- app executable SHA-256：`310f976ad5f1838637e588652c5d44e113037f468fbac518e9a83a9b9db3c17b`。
- `make goui-package-verify OUT_DIR=/tmp/orangecount-goui-g43-semantics-recovery-20261008 VERSION=0.0.0-g43-semantics-recovery GOARCH=arm64` 通过。build-info 对应 source revision `1c2096d435e07cc9b5a5957e0e974546ee1387a6`、dirty tree、Go 1.27.1、MyGo `v0.2.16-0.20261006170038-869768d02d91`。
- 可见测试使用 `/tmp/orangecount-goui-g43-semantics-recovery-visible-20261008/` 下的一次性 bundle；只改副本 `Info.plist` bundle ID/name，副本 executable 与来源包 SHA-256 相同。通过 `ORANGECOUNT_QA=1` 和独立 owner-only user-data 目录启动，并用 LaunchServices 打开公开 fixture。QA 数据只有该 fixture 的 recent 项、V3 profile 和 `2000-01-02` 截止偏好；没有读写生产偏好或账本。

## 可见窗口操作

1. Legacy 初次读取显示 `E-EVAL-INVENTORY`。打开“语义配置”后，窗口显示 Legacy `14 条分录 · 1 条错误 · 有效：false`，V3 `14 条分录 · 0 条错误 · 有效：true`，并显示“应用 V3”。
2. 点击“应用 V3”后，窗口显示 V3 当前配置及非空快照；同一公开账本成功读取，没有修改源文件。
3. 进入“持仓与成本”，将结束日期设为 `2000-01-02` 并应用。表格显示 `10 STK / 100 USD` 与 `3 STK / 60 EUR` 两组成本基础。
4. 通过“持仓分组”原生 selector 选择“按成本币种”。应用前，窗口显示“请应用持仓筛选以读取新结果”，并保留“应用持仓范围”按钮。点击后读取新结果，USD/EUR 成本仍分开，没有合并成 `160`。
5. 从同一隔离偏好目录再次启动并明确打开公开 fixture；Accessibility 显示账本已读取且无诊断，日期偏好为 `2000-01-02`。隔离 `preferences-v1.json` 为 `0600`，其中只记录该 fixture 的 profile/date。

退出时点击应用菜单“退出 OrangeCount”；CUA 的即时应用清单短暂保留运行标记，但随后按该临时 bundle 的 executable 完整路径检查已无进程。没有向该进程发送信号。

本轮通过 LaunchServices 直接打开 fixture，没有重复操作原生 Open Panel；也没有重复当前版本的查询/CSV/`.ocanalysis` 导出循环。先前 r7 包的导出证据不转移为本版本结果，G43 仍待当前包完整 GUI 读写矩阵、旧偏好迁移/隔离矩阵及其他未完条件验收。

本次语义恢复的原生自动化及回归证据见[G14 初次 Legacy 无效恢复](goui-p2-g14-invalid-legacy-recovery-2026-10-08.md)；持仓筛选自动化证据见[G38 持仓筛选状态与成本币种原生验证](goui-p6-g38-holdings-filter-state-native-2026-10-08.md)。
