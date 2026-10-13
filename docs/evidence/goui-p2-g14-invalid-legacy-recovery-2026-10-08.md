# G14 初次 Legacy 评估无效时切换到 V3 的恢复路径

日期：2026-10-08（Asia/Singapore）。修复语义配置页在初次 Legacy 评估无效、因此没有有效 SnapshotID 时无法展示已生成迁移预览的问题。

## 复现与原因

真实 QA 包打开公开 `native/holdings-unknowns.bean` 后，Legacy 报 `E-EVAL-INVENTORY`，账本没有有效快照。打开“语义配置”页时 workspace 已返回 migration response；该响应的输入身份非空，V3 摘要为 14 条分录、0 条错误且 `Valid=true`。原 `migrationResponseMatchesCurrent` 仍要求 SnapshotID 非空，因而把预览判旧，页面持续显示“正在生成预览”，不提供“应用 V3”。

修复后，当当前状态没有有效快照时，仅在 migration request/response 的 SnapshotID 都为空、响应 ledger revision 与当前 revision 相同、entry path 与当前工作区一致且输入身份非空时接受预览。`applySemanticsProfile` 对 `PhaseInvalid` 原本允许显式重开；核心重新打开时仍核对 `InputIdentity`，输入变化会拒绝切换。增加较旧 invalid-ledger revision 的拒绝测试，避免以同路径旧预览覆盖新输入。

## 验证

- 新增 `TestSemanticsPageCanSwitchAwayFromInvalidLegacyEvaluation`，用实际 DesktopEngine 打开该 fixture、读取 migration response、检查 V3 动作并点击；修复前测试稳定失败并显示空 SnapshotID、非空 migration identity、V3 有效，修复后成功进入同一文件的有效 V3 Session。
- `GOWORK=off go test -race ./internal/app -run '^TestSemantics' -count=5`：通过。
- 新增 `TestNativeInvalidLegacyLedgerCanApplyV3Recovery`，真实 MyGo Content 窗口的诊断→预览→Accessibility“应用 V3”→有效 Session 场景连续3次通过。
- `make goui-check`：通过。
- `./apps/goui/native-e2e.sh -v -count=1 -timeout=3m`：完整 gate 通过，app 测试 81.641 秒；5 个 opt-in 场景跳过（十分钟 soak、两种外部编辑器及两项私人账本对照）。Finder/拖放有一条 sandbox extension warning，但测试通过。

修复和最新源码包中的可见操作见[G43 语义恢复与持仓筛选包验收](goui-p7-g43-semantics-recovery-visible-2026-10-08.md)。G14 的 profile/capture 并发广泛 GUI 矩阵、真人 VoiceOver 仍由 G37/G38 跟踪。
