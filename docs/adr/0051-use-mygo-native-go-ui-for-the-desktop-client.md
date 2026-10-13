---
status: accepted
date: 2026-10-07
---

# 用 MyGO 的 Go ui 重写桌面客户端

用户在可行性研究后明确选择 MyGO 的 Go `ui` 路线。桌面客户端采用无 WebView 的 Go 自绘界面，直接复用现有 Go 会计、查询、报告和 `desktop.Session`；迁移目标是减少 Swift/C ABI 与双语言展示模型的维护，并统一桌面开发语言。macOS 是首个验收平台，现有 SwiftUI 客户端在迁移期间保留为行为与对比基线，新客户端完成等价验收后接管 macOS 主版本。

这一选择接受了自绘控件、财务图表和 macOS 适配的实现成本。MyGO 的窗口/表格/输入法能力不是现有 SwiftUI、Charts、导出保护或附件授权的自动替代品；这些行为按[设计](../mygo-native-desktop-design-2026-10-07.md)、[计划](../mygo-native-desktop-plan-2026-10-07.md)与[任务](../mygo-native-desktop-tasks-2026-10-07.md)逐项迁移。MyGO 固定源码修订、独立桌面 Go module 以及前置的图表/编辑器验证用于控制上游变化与迁移成本。

本决策替代旧[SwiftUI 桌面方案](../native-desktop-app-plan.md)作为下一代桌面技术路线；现有会计兼容契约、显式语义配置与源账本只读范围继续作为本次迁移的约束。Web 工作台和 CLI 的设计决策独立适用。本文记录路线决策，不表示新客户端已实现或通过验收。
