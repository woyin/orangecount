# G04 查询错误选区的可见窗口验证

日期：2026-10-08。以当前源码构建唯一 bundle ID 的 QA 测试包，user-data 指向 owner-only 临时目录；使用公开 `testdata/fixtures/native/cashflow.bean`，未保存查询或改写账本。测试包和临时偏好目录已在验证后清理。

在 MyGo 查询页输入并执行 `SELECT '💰', missing LIMIT 0`，页面显示 `unknown-column` 和 UTF-8 字节范围 `[15,22)`。点击“定位到查询错误”后，真实窗口截图中 `missing` 精确显示为文本选区。直接键入 `account` 仅替换该 token，查询草稿没有自动运行；显式再次执行后得到有效的零行类型化结果。这补足了原有 AppKit Native Input Client 选区数据之外的屏幕可见性证据。

系统输入法方面只做了只读检查：macOS 显示输入源为 ABC 与 Squirrel - Simplified，输入源快捷键为 F18（上一个）和 Control-Option-Space（下一个）。CUA 逐键输入仍只向编辑器提交 ASCII `ni`，未生成 marked text 或候选窗；因此这不是真人输入法验收，G37 的候选翻页、Enter/Esc、多窗口切换和 VoiceOver 朗读继续未完成。没有修改系统输入源配置或快捷键设置。

另观察到常驻旧 QA bundle 读取偏好时提示未知字段 `source_editor_bundle_id`；当前源码的偏好模型包含该字段，使用全新隔离 user-data 启动的当前源码包未复现此错误。该现象属于旧可执行文件与较新偏好格式不匹配，不据此改写生产偏好或归因于当前源码。
