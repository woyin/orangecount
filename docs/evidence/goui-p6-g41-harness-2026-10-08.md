# G41 Go UI 私人只读对照 harness 首轮记录

日期：2026-10-08（Asia/Singapore）。本次建立并使用**公开 fixture**验证 opt-in harness；没有读取用户私人账本，不能视为 G41 私人验收通过。

## Harness 行为

- 新命令 `make goui-private-read` 要求 `ORANGECOUNT_PRIVATE_LEDGER` 是绝对外部账本文件路径，可选 `ORANGECOUNT_PRIVATE_PROFILE=legacy|v3`（默认 legacy）。路径缺失或相对时只显示通用提示，不回显路径。
- `TestPrivateLedgerUIReadOnlyProjection` 同时打开核心 `desktop.Session` 与 Go UI 的真实 `workspace.DesktopEngine`，校验 snapshot/profile 一致；使用同一 snapshot 请求并完整比较核心与 workspace 的 `ViewResponse`。空 `ActiveExtensions` 的 nil 与空 slice 规范为同一语义。
- 对每个 workspace 响应创建独立 Model/Tester，渲染十个视图：dashboard、balance sheet、income statement、journal、prices、holdings、directives、attachments、migration、query catalogue。匹配页面标题，不输出实际私有文本。
- 成功日志只包括 page ID、profile、snapshot ID 的 12位单向摘要、UI节点数和 `read_only=true`；不打印账本路径、金额、交易文本、查询文本、附件位置或诊断。没有持久化偏好store，也没有保存/写入动作。

## 公开 fixture 验证

实际运行：

```sh
ORANGECOUNT_PRIVATE_LEDGER=/Users/breestealth/Development/OrangeCount/testdata/fixtures/native/cashflow.bean \
ORANGECOUNT_PRIVATE_PROFILE=v3 \
make goui-private-read
```

结果：十个页面均报告 `core_workspace=match`、`ui_heading=present`；snapshot 摘要 `3df7981ed40a`；profile 为 `v3`；测试通过。首轮运行暴露了 workspace 自动绑定 snapshot 和 nil/empty `ActiveExtensions` 表示差异，harness 已显式绑定同一 snapshot 并仅规范化空 slice；随后发现 MyGo ListState 不能在多个独立 Tester 间复用，现每页创建新的 Model/Tester。

## 未覆盖与后续 gate

本轮扩展后已覆盖 query execution/result 和导出字节的跨路径比较，但目前只运行公开 fixture，没有运行真实私人账本；G38 原生全功能 GUI 门槛也未通过。G41 仍进行中，需先满足 G38，再在本地只读运行绝对外部账本；私人内容不得进入仓库或测试日志。

## 2026-10-08 harness 扩展

在同一公开 `cashflow.bean`（v3）输入上，harness 额外执行账本相关的只读 BQL `SELECT account, number FROM postings LIMIT 10`，比较核心与 workspace 的完整 typed query response，并用 MyGo tester 确认结果列已渲染；随后逐字节比较 query CSV、`.ocanalysis` 和流水报表 CSV 的捕获内容。三种导出均匹配，字节数分别为 87、1210、2988。成功日志只记录导出类型、格式、匹配状态和大小，不打印内容。整套 opt-in 测试通过（约 0.24s）；这仍是公开 fixture 的 harness 校验，不构成私人账本或原生 GUI 验收。

## 2026-10-08 生产最近账本只读验收

新增 Darwin-only opt-in wrapper：仅当 `ORANGECOUNT_TEST_PRIVATE_RECENT_LEDGER=1` 时，从旧生产 defaults 中选择第一份可读最近账本，并调用既有隐私安全对照 harness。默认测试不会读取该账本；这次没有把路径传到命令行或测试日志，也没有写入账本、Go 偏好或查询预设。

```sh
cd apps/goui
env GOWORK=off CGO_ENABLED=0 ORANGECOUNT_TEST_PRIVATE_RECENT_LEDGER=1 \
  go test ./internal/app \
  -run '^TestPrivateLedgerUIReadOnlyProjectionFromProductionRecent$' -count=1 -v
```

本机实测通过，选择的配置为 `legacy`。10个页面的核心/Go UI 完整响应匹配且标题可渲染；typed query result 匹配，query CSV（441 bytes）、`.ocanalysis`（1362 bytes）及 journal report CSV（184589 bytes）逐字节匹配。最终 harness 日志不包含源路径、账目值、查询文本或诊断。使用的是本地 production defaults 最近账本，仅进行只读打开和内存投影；没有写账本或持久化用户数据。

G41 私人输入门槛已满足；G41 按台账仍等待其前置任务 G38 的完整 GUI 等价矩阵通过，因此当前整体状态继续为进行中。
