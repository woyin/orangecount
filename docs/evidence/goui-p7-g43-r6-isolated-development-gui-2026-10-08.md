# G43 r6 隔离开发包 GUI 闭环

日期：2026-10-08（Asia/Singapore）。本记录补充当前 dirty tree r6 development package 的可见 GUI 证据。它补足了同包读取、查询、导出与 Cmd+Q 进程退出检查，但不关闭 G43。

## 构建产物

- 源码 revision：`1c2096d435e07cc9b5a5957e0e974546ee1387a6`；build-info 标记 `source_tree=dirty`。
- MyGo：`v0.2.16-0.20261006170038-869768d02d91`；Go `1.27.1`；CGO 关闭；最低 macOS `13.0`；未签名、未公证。
- arm64 ZIP：`/tmp/orangecount-goui-current-arm64-20261008-r6/OrangeCount-Go-UI-Development-0.0.0-goui-dev-20261008-r6-arm64.zip`，SHA-256 `5e30771c01e84d19e7cc8a779187439f041c313caad2579cea6814ae7ad7cf94`。
- amd64 ZIP：`/tmp/orangecount-goui-current-amd64-20261008-r6/OrangeCount-Go-UI-Development-0.0.0-goui-dev-20261008-r6-amd64.zip`，SHA-256 `538f2b0dfe5651cf093f875ef738e1a49143e543580da4dc2423750d0c66ecca`。
- 两份产物重新运行 `make goui-package-verify` 均通过，验证 ZIP、plist、架构、macOS minimum、CGO/Swift linkage、build-info 与 module-license inventory。

为避免与桌面上已运行的正式 development 和 QA 身份冲突，将 arm64 包的 `.app` 复制到 `/tmp/orangecount-goui-g43-r6-dev/`，只在副本 `Info.plist` 中更改 bundle identifier 与显示名称。应用可执行文件仍来自 r6 包。通过 `ORANGECOUNT_ISOLATED_USER_DATA_DIR=/tmp/orangecount-goui-g43-r6-dev/user-data` 启动，目录权限为 `0700`；未设置 `ORANGECOUNT_QA`，窗口显示普通“开发预览”身份。没有打开其他 OrangeCount 应用或用户账本。

## 可见读写流程

1. 首次窗口为欢迎页，没有自动打开账本。使用应用原生文件面板打开仓库公开的 `testdata/fixtures/native/cashflow.bean`；仪表盘显示 3 个账户、2 笔交易、5 条分录，净资产变化 `700 USD`、收入 `1000 USD`、支出 `300 USD`。
2. 在查询工作台明确执行 `SELECT 1 AS value, 'packaged' AS source LIMIT 1`。窗口显示一行 typed result：integer `1`、string `packaged`。通过原生保存面板写出 query CSV 与 query `.ocanalysis`。
3. 再通过应用的原生打开面板载入该 query `.ocanalysis`。窗口显示“已载入离线捕获；文件中的查询不会自动执行”，并显示同一条 typed result；没有再次运行查询。
4. 在同一账本窗口保存资产负债表 CSV 和报表 `.ocanalysis`，并保存流水 CSV。输出均写入隔离的 `/tmp/orangecount-goui-g43-r6-dev/`。
5. CSV 文件行数：query 2 行、balance sheet 3 行、journal 5 行（含 header）。query 与 balance-sheet `.ocanalysis` 均通过 `unzip -t`，压缩条目完整。

## 同窗口错误与恢复

将公开 `cashflow.bean` 复制为隔离目录中的 `recovery.bean` 并从应用打开。随后只覆盖该临时副本为公开语法错误 fixture `testdata/fixtures/v3-parity/err-syntax.bean` 的内容。窗口自动报告 `E-PARSE-STRING`（第 1 行第 40 列），显示保留上次有效快照的提示，原有 3/2/5 统计与仪表盘仍可见。将公开有效 fixture 内容复制回临时文件后，窗口自动回到“账本已读取”、3/2/5 统计和无解析诊断状态。仓库 fixture 与任何用户账本均未修改。

## 退出与限制

在该 r6 包进程中发送原生 Cmd+Q 后，窗口回到欢迎页；对应直接启动该 bundle executable 的终端 session 随后以状态码 `0` 结束。欢迎页的最近路径列表与隔离 preferences 文件中的两条测试路径不一致；由于 CUA 可能自动重启 bundle，未将该欢迎页当作隔离证明。仅检查默认 Go UI Development 数据目录的文件元数据，没有发现本次运行时写入；没有读取该目录内容，也没有打开列表中的账本。另一份全新私有 user-data 目录的冷启动尝试中，应用进程出现在系统 app inventory，但 CUA 返回 ScreenCaptureKit `-3811`，所以没有把该次窗口状态或重启路径计为通过。隔离 preferences 文件只用于确认本轮显式打开的公开 fixture 与临时 recovery 副本，不能替代完整旧偏好迁移/dev-QA 矩阵。

仍未完成：干净 macOS 用户下的首次安装和重启验收、旧版偏好迁移/dev-QA 隔离矩阵、G37/G38 的 IME、VoiceOver 与旧客户端完整 GUI 等价验收，以及 G39/G40 所需的完整长期性能与生命周期覆盖。此记录不构成签名发行或默认入口切换验收。
