# Go / 原生开发文档交接基线

日期：2026-10-01。用途：为[设计](../go-beancount-native-design-2026-09-30.md)、[计划](../go-beancount-native-plan-2026-09-30.md)及[实施手册](../go-beancount-native-implementation-2026-09-30.md)提供本次只读复验结果。不是里程碑完成报告。

## 源码与环境

- HEAD：`20a9a6e86a5b4e2d6bb51aa2d2ae19dcd2934ddd`，包含已有大量 tracked/untracked 工作区修改；只检出 HEAD 无法重现当前行为。
- 当前环境：macOS 27.0.1（26A434）、arm64；Go 1.27.1；Swift 6.4 / driver 1.168.6；SDK 27.0。
- 产品编译目标沿用 macOS 13 / arm64；本次没有验证 macOS 13 实机、另一机器、Developer ID 或公证。
- oracle：固定 Beancount 3.2.3；本次私有对照 profile=v3。默认 profile 仍为 legacy。
- 本轮只修改文档；现有产品源码及先写的失败测试保留，没有创建提交或发布。

## 本次实际命令与结果

| 检查 | 实际命令 | 结果及范围 |
| --- | --- | --- |
| 公开持仓/库存 | `go test -count=1 ./internal/desktop ./internal/report -run 'TestNativeHoldings\|TestHoldingsReplay'` | exit 0；两个包通过，涵盖当前具名 targeted 检查，不等于全量 Go/race |
| 原生链路 | `make mac-test` | exit 2；真实 ContractTests 全部完成并通过，StateTests 编译失败，后续 NativeClientTests 未执行 |
| 私有同源差分 | `ORANGECOUNT_PRIVATE_LEDGER=/absolute/path/to/main.bean sh tools/reference/differential.sh` | exit 0；完整 entries/inventory 与月度查询一致，native_checked=false |

StateTests 的具体缺口：`NavItem` 没有 holdings，`LedgerStore` 没有 showHoldingLots / showHoldingEventLots。测试已表达快照绑定、跨页返回、切 cutoff 清 selector 和迟到响应要求，Implementation 尚未跟上；后续从 A02.5 补齐，不删除或跳过这组测试来制造 green。

ContractTests 已通过的本批相关行为包括历史库存、精确 AVERAGE 收益、实际消费与合并贡献来源、合并贡献分页、nullable/full-key/未知原因，以及字段和协议负向检查。它们不是 Store 时序或 GUI 证据。

私有结果只保留聚合：captured_files=47；双方 entries=2831、errors=0、valid=true；missing_entries=0、extra_entries=0、changed_inventory_accounts=0；Expenses 73 行和 Income 22 行 equal=true；same_input_capture=true、schema_version=2、native_checked=false。没有修改账本，没有记录路径、账户、金额、摘要、来源行或截图。

## 继承证据与限制

以前全量 Go/race/parity、原生全套、app 构建和部分 GUI 的结果见[实施进度](../go-beancount-native-progress.md)。当前有新增契约/测试且原生门槛明确失败，旧的 mac-test 成功不能覆盖这次失败。先前私有真实 Swift/C 为另一批已捕获输入，不与本次计数合并。

本次未执行全量 Go/race/parity、make app、GUI、benchmark、私有真实 Swift/C 或远端 CI。文档整理不改变会计行为，因此没有将这些未执行门槛自动记为通过。后续实现修改相应模块后，按实施手册重新运行受影响 gate。

## 后续第一个开发动作

阅读实施手册 26.3 及 28 节，补 NavItem/Store 的持仓入口、详情选择器、返回和过滤归一化，使当前 StateTests 从编译失败变为实际执行通过；随后接入 HoldingsView、补新场景和集成验收，再继续 A03–X02。任何代码变更都会使下列相应文件身份成为继承记录。

## 相关源码与公开输入 SHA256

文件身份仅用于判断证据是否仍适用，不包含私人输入。

| 文件 | SHA256 |
| --- | --- |
| `internal/ledger/inventory_replay.go` | `efc766d460d9772092be28a62e90666dd868c8b05b24546c178f439bc3c86611` |
| `internal/report/native_holdings.go` | `eb448ba1b73314195d032fce1f1956c07128d38cdca477b9a29b75529ae9c673` |
| `internal/report/native_reductions.go` | `0617723cdd8f23e6d41acf493ab642b63bf2743afe4034ce0fbf0e8ecbd6d612` |
| `internal/report/native_average_history.go` | `a7d203a661b299f4de16da521fcab829777a7fc7634f187e0f46e49f94fa777c` |
| `internal/desktop/holdings.go` | `ac0b1ed43253ade171538d19db62abdb9820eaaf9a827bc847285748321101b7` |
| `internal/desktop/holdings_test.go` | `f6ce0304b2b0aebd965cf0612abf63aacc64cc85a166f777c94a47f98f5823d6` |
| `internal/desktop/journal.go` | `8470baaaffb3d693bf2f4fad14d84e2e53e60a7028d3caeb6f07d3ca7c00bd3f` |
| `apps/macos/src/Models.swift` | `67ea57bdd1a33ce18a2dcbd3a4eb9a0ba47c9fad886f6225950b135c4f42ae8a` |
| `apps/macos/src/LedgerStore.swift` | `157440e04ff27d2167e393af051e8063b695635a54f9015f93909b75d4b65a29` |
| `apps/macos/tests/StateTests.swift` | `d546dc53d650f4b9ee4e182c7451e5c1420f69bbe29ab7f412d1bc48f828d20d` |
| `apps/macos/tests/ContractTests.swift` | `ec1cc6c8fae8ad9ac8f9c5b6d7347901fe5be7194126c3fb1c7dd07912a3bd09` |
| `testdata/fixtures/native/average-history.bean` | `7b30ff7f1744e4a83f16bae9feeced8d1e297a2f2264e36ed9e2aa1b2001122d` |
