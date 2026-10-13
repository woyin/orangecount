# G43 修复版开发/QA 偏好隔离复验

日期：2026-10-08（Asia/Singapore）。在同一 `0.0.0-g43-semantics-recovery` arm64 可执行文件上，分别运行隔离开发身份和隔离 QA 身份，检查欢迎页、最近账本、每账本偏好和重启读回。

## 包与身份

- 来源 `.app`：`/tmp/orangecount-goui-g43-semantics-recovery-20261008/OrangeCount Go UI Development.app`；ZIP SHA-256 `89c51ed0ba1de7d08aaebe905ed2cf43d2c0c4ae571a49aebba7936af4e2c3bd`。
- 两个隔离副本的 `Contents/MacOS/orangecount-goui` SHA-256 均为 `310f976ad5f1838637e588652c5d44e113037f468fbac518e9a83a9b9db3c17b`。为避免与机器上的其他 app 身份冲突，副本只改 `Info.plist` 中的 Bundle ID/显示名；开发副本 ID 为 `org.orangecount.desktop.goui.g43.isolation.dev`，QA 副本 ID 为 `org.orangecount.desktop.goui.g43.semanticsqa`。
- 开发身份显式设置 `ORANGECOUNT_ISOLATED_USER_DATA_DIR=/tmp/orangecount-goui-g43-semantics-isolation-20261008/dev-user-data`，未设置 `ORANGECOUNT_QA`。QA 身份设置 `ORANGECOUNT_QA=1`，使用 `/tmp/orangecount-goui-g43-semantics-recovery-visible-20261008/user-data`。两个 user-data 根目录均为 `0700`，两份 `preferences-v1.json` 均为 `0600`；路径和偏好文件分离。

## 可见开发身份流程

1. 全新开发隔离目录启动到欢迎页，页面显示“开发身份与现有客户端数据隔离”，没有自动恢复最近账本。
2. 通过应用 Open Panel 明确打开公开 `testdata/fixtures/native/cashflow.bean`。窗口显示“开发预览”、3 个账户、2 笔交易和 5 条分录；没有 QA 标识。
3. 将结束日期设为 `2026-01-02` 并应用。Cmd+Q 后进程表确认隔离 executable 已退出。
4. 用相同包副本和相同隔离 user-data 重启。欢迎页仅列出本轮的 `cashflow.bean`，没有自动打开账本；明确点选后，窗口恢复 `2026-01-02`，净资产 `1000 USD`、收入 `1000 USD`、支出 `0 USD`。再次 Cmd+Q 后进程表确认退出。

## 与 QA 数据对照

QA 隔离数据仅包含本轮公开的 `cashflow.bean` 与 `holdings-unknowns.bean` 最近项；保存的语义/日期偏好为 `v3` / `2000-01-02`。开发隔离数据仅包含 `cashflow.bean` 最近项；其语义/日期为 `legacy` / `2026-01-02`。读回的两个偏好文件分别保留各自的最近路径与值，没有串用另一身份的设置。

对应源码的启动 Store 测试已覆盖 production 迁移使用独立 fake legacy defaults、QA 不调用 legacy opener、显式隔离开发 override 不读取 legacy defaults、私有目录/符号链接约束。本次实际窗口验证覆盖同一包的开发隔离启动、用户明确打开、偏好写入、Cmd+Q、重启欢迎页和恢复读回；未读取或修改生产 UserDefaults。

通过终端直接启动开发 bundle executable 时 macOS 输出一次 `sandbox_extension_issue_file_to_process ... Operation not permitted`。应用仍显示原生 Open Panel 并成功读取公开 fixture，页面和重启偏好检查均完成；因此这里记录为非阻塞运行环境提示，不把本次标记为 LaunchServices 安装验收。生产 defaults 升级、清洁用户安装/迁移体验及完整身份矩阵仍未完成，G43 继续进行中。
