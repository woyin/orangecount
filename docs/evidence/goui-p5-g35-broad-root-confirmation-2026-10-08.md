# G35 大范围附件目录确认原生验收

日期：2026-10-08。验证 `附件目录` 页选择较大授权范围时，原生确认 sheet 的窗口归属、取消行为与允许后的书签/策略持久化。

## 隔离方式

- 每次测试都在 `t.TempDir()` 下创建账本、偏好文件和临时 home 目录，并将测试进程 `HOME` 临时指向该目录。
- picker adapter 只返回这个临时 home；没有选择或授权真实用户 home、`/Users`、卷根目录或其他真实用户文件。
- 测试分别取消确认和允许确认。允许路径创建普通 Darwin directory bookmark，只写入临时偏好 Store，并验证 bookmark 可解析回该临时目录。
- 原生确认由 MyGo `Dialog.Message` 创建真实 AppKit `NSAlert` sheet。仅在 `darwin && orangecount_e2e` 下增加的 MyGo test hook 遍历 sheet 的公开 `NSView.subviews`，按按钮标题调用 `performClick:`；生产构建不包含该 hook。它避免把 `_NSAlertPanel` 当成 `NSAlert` 调用不支持的 `buttons` selector。

## 结果

`TestNativeBroadAttachmentRootConfirmationCanBeCancelled` 验证真实 sheet 附属于当前 MyGo 窗口。点击“取消”后 sheet 关闭，`picking` 状态清除，workspace 附加根目录和偏好书签均为空。

`TestNativeBroadAttachmentRootConfirmationCanBeAccepted` 点击“允许”后验证 sheet 关闭，临时 home 出现在 workspace 策略与账本偏好中，生成的 Darwin bookmark 可解析回该路径。

两个场景组合以 `-count=5` 连续通过。随后 `make goui-check` 通过；完整 `make goui-native-e2e` 通过，当前含58项常规原生窗口场景，独立十分钟 soak 入口不计入此数，耗时39.256秒。`git diff --check` 通过。

这关闭了 G35/G38 的大范围目录确认原生验收缺口；开发/QA 身份间迁移隔离和 G38 其他等价矩阵仍未完成。
