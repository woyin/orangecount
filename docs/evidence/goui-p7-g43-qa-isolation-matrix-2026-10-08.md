# G43 临时 QA bundle 可见启动与偏好隔离补测

日期：2026-10-08（Asia/Singapore）。本记录验证当前 dirty working tree 构建的 MyGo QA 身份在独立 bundle 和私有 user-data 下运行；它不是正式 QA 发行包。

## 构建与隔离

- 从 `apps/goui` 运行 `GOWORK=off CGO_ENABLED=0 go build -trimpath -ldflags='-buildid= -X main.qaBuild=true'`，产物为 arm64 Mach-O。来源 HEAD 为 `1c2096d435e07cc9b5a5957e0e974546ee1387a6`，工作树含未提交改动。
- 临时 bundle 位于 `/tmp/orangecount-goui-qa-r7-isolated-20261008/OrangeCount Go UI QA Isolated.app`。使用 QA 的 `Info.plist`，仅把 bundle ID/name 改为 `org.orangecount.desktop.goui.qa.r7.isolated` / `OrangeCount Go UI QA Isolated`，避免与机器上现存 QA app 身份冲突。运行的是当前源码构建出的原始 executable。
- executable SHA-256：`66b1e9c63111937fff496cf8876fb7804932e987792eab28976b891773796753`。
- 每次都显式传入 `ORANGECOUNT_ISOLATED_USER_DATA_DIR=/tmp/orangecount-goui-qa-r7-isolated-20261008/user-data`。目录权限为 `0700`；偏好文件 `preferences-v1.json` 为 `0600`。QA 启动不读取旧 production defaults；该隔离运行的迁移标记为 `0`。

## 可见 QA 流程

1. 全新偏好目录启动到欢迎页，Accessibility 明确显示“QA 模式：不会自动恢复最近账本”，未自动打开账本。
2. 通过原生 Open Panel 打开公开 `testdata/fixtures/native/cashflow.bean`。窗口显示 3 个账户、2 笔交易、5 条分录；净资产 `700 USD`、收入 `1000 USD`、支出 `300 USD`。
3. 设置并应用结束日期 `2026-01-02` 后，窗口显示收入 `1000 USD`、支出 `0 USD`、净流量 `1000 USD`。隔离偏好文件只记录这一条公开 fixture 最近项和一份账本偏好，其中 `date_to` 为 `2026-01-02`；不含 migration marker。
4. 后续 CUA 调用得到该 QA 身份的欢迎页，只显示本轮公开 fixture；明确选择最近项后，结束日期为 `2026-01-02`，页面显示净资产 `1000 USD`、收入 `1000 USD`、支出 `0 USD`。但直接进程检查当时仍发现这个临时 executable 的 PID 存活，因此这只能算偏好文件/界面读回，不能算 PID 已确认的新进程重启。
5. 尝试 Cmd+Q 后启动 shell session 返回状态码 `0`，但精确命令路径对应的 PID `91212` 仍存活。为清理本轮实例，向该唯一临时 bundle PID 发 SIGTERM，随后 `ps` 确认进程消失。这个清理不构成 GUI 优雅退出验收；其他 OrangeCount Dev/QA 进程没有被关闭或修改。

这个 QA 可见循环与[G43 r7 development bundle 隔离 GUI 记录](goui-p7-g43-r7-isolated-development-gui-2026-10-08.md)互补：development 与 QA 分别使用不同 bundle ID、显式 owner-only user-data 目录；两者的首屏、最近项选择和每账本偏好恢复均有实际窗口证据。启动层的生产旧偏好迁移、QA 禁止 legacy reader、隔离 override 权限等另由[启动存储定向测试](../../apps/goui/cmd/orangecount-goui/stores_test.go)与[G06 只读迁移记录](goui-p0-g06-legacy-migration-2026-10-08.md)覆盖；本轮未把生产用户 defaults 改写为测试数据。

## 观察限制与剩余验收

一次 Cmd+Q 后，旧 CUA handle 返回了两条不在隔离偏好文件中的历史路径。直接进程检查同时确认临时 executable 仍在运行；这两条路径既不在隔离 JSON 中，也无法确定来自被测 bundle 还是 CUA 窗口绑定到的同机其他 QA 窗口。因此这次 AX 树不用于判断偏好隔离。迁移跳过由 `qaBuild`/隔离启动代码和定向测试证明，实际窗口里的历史项归属仍需在能按 PID 绑定窗口的条件下复验。

本记录确认了 QA bundle 的本机可见首启、明确打开和偏好落盘/读取；PID 确认的重启、QA 最近项归属和 GUI 优雅退出仍待复验。它也不替代干净用户安装下的完整升级/旧偏好迁移体验，更不关闭 G37 真人输入法/VoiceOver、G38 完整 GUI 等价矩阵、G39 性能 soak、G40 生命周期 soak、G41 全范围核心对照或 G42 远端 CI；G43 仍进行中。
