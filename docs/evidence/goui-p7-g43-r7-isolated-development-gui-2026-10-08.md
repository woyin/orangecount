# G43 r7 隔离开发包可见 GUI 与重启闭环

日期：2026-10-08（Asia/Singapore）。在当前源码 arm64 r7 development 包上完成真实 macOS GUI 读账本、查询、导出、错误恢复、偏好重启与退出检查。

## 实际包与隔离身份

- 来源包：`/tmp/orangecount-goui-current-arm64-20261008-r7/OrangeCount Go UI Development.app`。
- 源 ZIP：`OrangeCount-Go-UI-Development-0.0.0-goui-dev-20261008-r7-arm64.zip`，SHA-256 `7c960e378683ea20eeda900a59a1c7493387a16bfa37cb968551859591d9584b`；来源包此前已通过完整 `make goui-package-verify`。
- 为隔离当前桌面上已有的同名应用，将 `.app` 复制到 `/tmp/orangecount-goui-r7-visible-20261008/`，只在临时副本 `Info.plist` 中更改 bundle ID/name 为 `org.orangecount.desktop.goui.dev.r7.isolated` / `OrangeCount Go UI Development r7 Isolated`。测试运行的是包内原始 executable。
- 两次进程都通过 `ORANGECOUNT_ISOLATED_USER_DATA_DIR=/tmp/orangecount-goui-r7-visible-20261008/user-data` 启动。user-data 根目录为 `0700`，`preferences-v1.json` 为 `0600`；重启前后最近列表只含本轮打开的公开 `cashflow.bean` 和临时 `main.bean`。首次欢迎页没有自动打开最近账本，重启后也需明确点击公开账本才打开。
- 启动 shell 曾打印 `sandbox_extension_issue_file_to_process ... Operation not permitted`，但 CUA 随后成功绑定该 bundle 的真实窗口，完成下述界面操作；两次 Cmd+Q 后启动该包的 shell session 均以状态码 `0` 结束。

## 同一进程内的 GUI 操作

1. 欢迎页通过原生 Open Panel 打开公开 `testdata/fixtures/native/cashflow.bean`。仪表盘显示 3 个账户、2 笔交易、5 条分录，初始净资产 `700 USD`、收入 `1000 USD`、支出 `300 USD`。
2. 在查询页输入并明确执行 `SELECT 1 AS value, 'packaged' AS source LIMIT 1`。原生结果表显示 integer `1` 和 string `packaged`。用 Save Panel 保存 query CSV 与查询 `.ocanalysis`，再通过应用 Open Panel 打开该分析文件。界面明确显示离线捕获且不会重跑原查询，结果仍为 `1 / packaged`。
3. 在流水页看到 2 笔交易、当前页 4 条分录；经 Save Panel 保存流水 CSV 和 `.ocanalysis`。
4. 在资产负债表设置结束日期 `2026-01-02` 并应用。界面显示 `Assets 1000 USD`，排除 1 月 3 日支出；保存余额 CSV 和 `.ocanalysis`。
5. 将公开 fixture 复制到隔离 recovery 目录，权限 `0600`，然后从窗口打开。把该副本替换为公开 `testdata/fixtures/v3-parity/err-syntax.bean` 内容后，真实窗口显示 `E-PARSE-STRING`、`main.bean:1:40` 和“当前显示上次有效快照”，同时保留 3/2/5 统计及 700 USD 仪表盘。恢复原有效字节后，同一窗口自动显示“账本已读取”、同样的有效数值且无诊断。

## 落盘校验与重启

所有输出都由应用的原生 Save Panel 写入 `/tmp/orangecount-goui-r7-visible-20261008/exports/`。用 CSV parser 读回 query 为 1 行、journal 为 4 条 posting、balance sheet 为 2 个账户行；query 的确切值为 `1, packaged`。三份 `.ocanalysis` 均通过 ZIP 完整性检查，包含 `context.json`、`schema.json`、`cells.ndjson` 与 `data.csv`：query 为 format 1 / `query` / captured-result；journal 为 format 2 / `journal` / current-page / 11 列；balance sheet 为 format 2 / `balance_sheet` / complete-projection / 12 列。查询 capture 也由 GUI 离线打开并核对未重新执行。

第一个进程 Cmd+Q 退出后，以相同 bundle 和相同隔离 user-data override 再启动。重启后的欢迎页显示本轮两个测试最近项但没有自动打开；显式选择公开 cashflow fixture 后显示 3/2/5 统计，并恢复之前设置的 `2026-01-02` 结束日期。第二次 Cmd+Q 后 shell session 也以状态码 `0` 结束。

## 导出文件 SHA-256

| 文件 | SHA-256 |
|---|---|
| `cashflow-query.csv` | `eb006fb2ed731e6fb83eb3a903007dbe2028ff2598c5fad5a6bf737bc599cd9b` |
| `cashflow-query.ocanalysis` | `723755f8e0162ed225aad5230f1016181200dc7f758ef7844163c20f14b4aa7c` |
| `cashflow-transactions--2026-01-03.csv` | `ae18a127880b608339fb5446d18b89dafc287923a0f89ffa7be48b6f94be1638` |
| `cashflow-transactions--2026-01-03.ocanalysis` | `572a4831fca85d36c1a09d9e91d59775d54105dd8db01179c5f0b89a42f5801b` |
| `cashflow-tree--2026-01-02.csv` | `c1b4d14f4c2e337045bb2859cb0dfd75b64937a5088f9f1c72674e36e32b2fa0` |
| `cashflow-tree--2026-01-02.ocanalysis` | `dd3400ad7e422423283be4fb634ac59fc609cdc14edf569c2fadd78949532652` |

这轮验证补齐 r7 包的 GUI 导出、离线捕获重读、同窗口错误恢复、隔离偏好重启与进程退出证据。它不替代干净 macOS 用户下的完整旧偏好迁移/dev-QA 隔离矩阵、G37 真人 IME/VoiceOver、其他机器/macOS 13 运行、远端 CI 或正式签名/公证；G43 仍进行中。完成后关闭了本轮 r7 进程，准备移除的仅是上述明确创建的 `/tmp/orangecount-goui-r7-visible-20261008/` 测试副本与输出。
