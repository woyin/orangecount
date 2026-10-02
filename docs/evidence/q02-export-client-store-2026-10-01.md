# Q02 原生导出客户端与 Store 捕获动作

日期：2026-10-01，基线 `9e5d794`，接续会话/C/Swift 固定导出契约。

NativeLedgerClient 新增独立 LedgerQueryExportClient seam，在窗口 actor 上执行 Go 内容捕获，进入和返回时检查 Task 取消。Store.captureQueryExport 只接受当前成功执行及 csv/analysis 两种格式，使用已执行响应身份而非编辑器草稿；异步返回后检查窗口 generation/handle、当前快照及请求/指纹/格式。返回固定字节后，调用方可以在保存面板期间持有该内容，不需要重读账本。

采用单独导出协议，既有只实现查询执行/取消的客户端不会被赋予导出能力。当前方法返回捕获或明确错误，不发布“保存成功”，也不写文件。

## 本批证据

- 真实 NativeClientTests 首次编译失败，指出 Store 捕获接口和错误类型缺失；接入后原生三套测试 exit0。
- 实际零行 query 成功后修改草稿；导出的 CSV 仍是原执行 number 列标题，context.text 仍为原查询，analysis 是实际 Go ZIP 字节。
- 后续 unknown-function 失败时 Store 拒绝导出此前成功；已捕获 CSV 保持原内容。
- 受控客户端暂停捕获返回，分别触发新查询成功、快照发布变化、关闭窗口；三种迟到捕获均被 Store 的 changedExecution 拒绝。
- 加入受控覆盖后原生三套再次 exit0；fmt/vet/license exit0。Go 产品未在本批修改，Go 全量/相关 race 使用上一批证据，没有重新宣称本批执行。
- 随后 app 构建 exit0；arm64、target 13、ad-hoc 开发包验证通过，未进行实际 GUI 或旧系统验收。

## 接续

尚未增加 QueryView 导出按钮、保存面板、唯一临时文件与原子替换、账本源图/附件/alias/hardlink 目标保护、保存错误呈现或实际 GUI 验收。多次同时捕获的完整预算、捕获取消时序和路径隐私/完整容器案例仍需验证。不能把捕获返回称为文件已保存，完整24任务保持未完成。
