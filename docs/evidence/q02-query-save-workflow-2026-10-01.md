# Q02 保护刷新、原生保存工作流及首批实际 GUI

日期：2026-10-01，基线 `9e5d794`；接续原子保存器。全部 GUI 输入为公开 fixture 或其 build 目录一次性副本，没有使用私人账本。

## 实现

ExportProtection 独立读取当前有效 publication 的源/附件保护路径，并合并最新尝试的 include 源图，包含加载失败的新源文件。它不重建导出内容；重载中、采集期间 revision 变化和关闭会话时拒绝。新增 C、Swift 严格路径响应、NativeLedgerClient 和 Store 刷新接口。

Store.prepareQueryExport 捕获成功执行内容及当时的文件身份，形成 PreparedQueryExport。saveQueryExport 校验窗口 generation/handle，刷新保护路径并与旧路径和 inode 身份合并，再在后台调用原子保存器。捕获之后的查询/草稿/快照变化不改写文件内容；更换或关闭账本取消已登记的写任务，父 Task 取消也传递给 writer。写入前和 rename 前检查 Task 取消。

QueryView 分别提供“保存分析文件”和“保存 CSV”，仅当前成功结果可发起。先准备内容，再打开 NSSavePanel；面板明确两种格式的区别。取消面板不调用保存；离开页面取消任务/面板。成功提示只在原子写入返回后出现；源/附件拒绝与不支持目标有明确错误提示。状态由 ObservableObject 保存协调器管理。

## 本批程序验证

- Go 新保护刷新测试先因接口缺失编译失败；实际有效新增 include、随后失败的 include 图均进入保护集合，原捕获 CSV/快照身份不变；关闭后拒绝。
- 真实 Store 保存测试先因缺少 prepare/save 接口编译失败；接入后准备零行 CSV，执行其他 query 再重载，写出仍为原 number 标题；主源覆盖拒绝；预先取消的保存没有创建文件。
- Go 全量、desktop/bridge/query 三包 race exit0；原生四套 exit0，包含真实保护刷新和保存动作。fmt/vet/license exit0。
- 初次 app 编译遇到 SDK 的同名 State 宏缺少 SwiftUIMacros 插件；显式 State 泛型试验仍失败，改用既有 ObservableObject/StateObject 模式后 app 通过。

## 实际 GUI 验证与产物

使用 prepare-qa.sh 隔离 bundle/偏好及显式公开 default-source.bean；原运行用户应用保持原样。进入 query 不自动执行，保存按钮禁用；执行以下公开查询后按钮启用，AX 结果分别保留重复标题、精确值、NULL 与空字符串：

```sql
SELECT 1 AS v, 2 AS v, decimal('1.00000000000000000001') AS precise,
       NULL AS n, '' AS blank LIMIT 1
```

1. 实际 CSV 面板显示类型/上下文省略说明；点击 Cancel，页面显示“已取消保存”，默认目录没有新增 query-result.csv。
2. CSV 保存到 build/qa-export，页面显示实际成功文件名。Python 标准 csv 解码产物与已知列/行逐项一致。
3. 分析面板最初显示重复扩展名 query-result.ocanalysis.ocanalysis。取消、关闭 QA、把 allowedContentTypes 配置移到 nameFieldStringValue 之前、重建并重启后，实际面板显示单个扩展名，保存成功。
4. 实际分析 ZIP 恰好四个固定成员；context version/scope/行数、重复标题独立 ID、Decimal exact、typed NULL/空字符串均正确；内含 CSV 与独立 CSV 字节相等；没有 protected_paths 字段或重复扩展名产物。
5. 将公开账本复制为 build/qa-export/protected-ledger.csv，显式在 QA 加载，以该一次性副本作为 CSV 目标。经过系统 Replace 确认后，应用显示“不能覆盖账本源文件或声明附件”；副本与公开 fixture 字节相等，目录无临时文件。
6. 本轮隔离 QA 已退出。没有保存私人内容或截取用户原应用窗口。

| 公开 GUI 产物 | SHA256 |
| --- | --- |
| build/qa-export/query-result.csv | 44311eeb25264a8e3fb115cf93993ddc244af82e355eae2fa1bf00d51e6a0f37 |
| build/qa-export/query-result.ocanalysis | fc9b35a85fbb08b4fec05de79bd1a67bd3b4f871d125023f23f4c84477512cbd |

产物位于忽略的 build 目录，不作为提交 fixture；哈希只标识本次公开捕获。最终 app/QA 包构建与签名验证 exit0，arm64/target13/ad-hoc；不是其他机器/旧系统验收。

## 未验范围与接续

继续完整容器解包验证器及多币种/compound/截断/partial/unknown 案例、query preset、附件状态/外部打开、全指令页面，以及各窗口/字体/键盘/AX/性能/安装发行门槛。GUI 当前只验证默认语言、默认字号、legacy 的公开标量查询；v3 保存、附件目标、真实 GUI 保存期间 reload/关闭、多个同时保存仍需扩展。

失败新输入目前保护 include 源图及上次有效附件；失败输入中新声明的附件尚未建立完整保护集合。Go 捕获与 Swift 文件身份初始化之间的替换、刷新保护后至最终 rename 的并发 source 变更、保护元数据完整预算、Finder alias 和 close/fsync/rename 失败注入仍保留。没有通过某个 green 测试推断这些边界已完成。

会计51/query147与私人账本属于此前证据，本批未重跑。完整 Q02/X02 和24任务未关闭。

## 本批源码身份

| 文件 | SHA256 |
| --- | --- |
| internal/desktop/query_export.go | 35ed9eee8a84bfc27f235b205bc469cdeab2ae3100c2c35c58de2bf3fe2c277a |
| internal/desktop/query_export_test.go | b25dedeed8efadb9de112eabcb08a04a5ed739d19baa6db06c5e93910d00b889 |
| internal/bridge/session.go | 5da82b09dca9a9794a4bdb6ac3df5a4c4b875e5e47f93f64581c575e11c3f2b3 |
| apps/macos/src/AnalysisFileExport.swift | 9a986701eb04056435b3dd7563553ca07e1f9d4f8dd7cd1902c9381c6f8648cd |
| apps/macos/src/QueryView.swift | f5b729237e5b4a6b910ac2d806451d19dc9928d0907eee8cf80181fd03607e32 |
| apps/macos/src/LedgerStore.swift | 469be2dbfe3b26946086b1c93acaf3d32440445eca206089e88ba934297b2785 |
| apps/macos/tests/NativeClientTests.swift | 0ea9dba680da41d33cc8224156ac8aefcfbd3fe79e6cabfe8b63bbb3f5899cf7 |
