# G12 偏好备份导出

日期：2026-10-08。该功能补齐“可导出迁移后的偏好备份”；G12 仍需完成完整产品首次启动恢复验收。

## 行为与边界

- `preferences.Store.BackupSnapshot` 在 store 锁内读取并校验当前版本化文档，生成固定 JSON 快照和活动偏好文件保护路径。序列化内容包括 schema/migration marker、最近账本路径、每账本阅读与显示设置、附件目录普通 bookmark，以及来源编辑器等应用设置。
- Go UI 的“文件 → 导出偏好备份…”使用原生、可取消的 Save Panel。面板会说明备份包括本地路径和书签，不包含账本内容、查询草稿或查询结果。
- 用户选择位置后，由 `localfiles.Capture` 冻结字节并使用目录锚定、保护路径校验与原子替换写入；目标扩展名为 `.json`，新文件权限为 `0600`。活动偏好文件本身受到保护，不能被选作目标。
- 快照在显示保存面板之前生成；面板期间发生的设置变化只进入活动 store，不会改写本次备份内容。取消和关闭窗口都会取消待处理的保存。

## 验证

- `TestBackupSnapshotFreezesValidatedPreferenceDocument` 检查迁移 marker、最近路径、profile/reading/locale/列设置及固定快照行为。
- App 单测通过菜单命令注入保存位置并读回 JSON；期间将活动币种从 USD 改成 EUR，备份仍保存面板打开前的 USD 状态，活动 store 保留 EUR，导出文件权限为 `0600`。
- `TestPreferencesBackupCannotReplaceTheLiveStore` 确认选择活动 `preferences-v1.json` 会被拒绝且原文件字节不变。
- 原生 MyGo E2E 通过文件菜单打开真实 Save Panel，检查面板只归属发起窗口并可取消；随后通过同一命令保存临时 JSON，并确认界面报告成功。
- 当前轮 `make goui-check`、`go vet ./...`、Go UI 全量 `go test -race ./... -count=1` 通过。原生窗口场景也包含本功能的真实面板路径；完整套件与重复运行结果记录在任务台账。

## 限制

该备份包含用户明确选择导出的最近文件路径和附件目录书签，用户保存前可在面板查看导出范围。当前没有自动恢复按钮；从备份恢复及完整首次启动恢复仍属于后续交付验收。
