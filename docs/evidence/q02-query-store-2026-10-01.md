# Q02 Store查询状态与有界排队

日期：2026-10-01。基线提交`9e5d794`；接续后台客户端切片，完整Q02尚未完成。

## 当前实现

Store通过LedgerQueryClient执行查询。请求ID在窗口内单调增长，捕获会话handle、窗口代次、快照、profile、会计修订和执行文本。只保留一条active与一条最新pending，连续提交替换pending并通知active取消；即使受控客户端忽略取消并迟到返回，也不启动被替换的中间请求或发布旧结果。

草稿、当前响应、最近成功记录、错误信息与执行阶段分别保存。当前响应失败时execution为空，不借用历史成功；编辑草稿不更改已执行文本。queryResultIsCurrent核对成功阶段及快照/profile/revision；历史成功必须单独标识，尚未接导出入口。

显式reload先取消正在执行/排队的查询；新快照或profile/revision变化使已执行结果stale。换账本和关闭清空查询记录，并丢弃迟到结果。deinit取消查询Task，实际C释放与Go检查点中断沿用上批客户端。

## 行为检查

1. 首个Store测试因缺少queryDraft/executeQuery及状态接口编译失败；测试构造中的Any三元类型错误同时修正。
2. 受控客户端连续提交1/2/3：仅1和3到达执行接口，1迟到成功不发布；3成功绑定其原始文本。
3. 编辑草稿、取消后迟到成功、失败不冒充成功、换账本清理记录分别检查。
4. 受控reload取消active；新快照保留旧结果身份但标为stale；close拒绝迟到响应。
5. 真实Swift/C Store：成功零行仍有列schema及v3身份；未知函数LIMIT0仍编译失败；取消耗时查询后最新查询成功；修改公开临时输入并reload后旧结果stale且快照身份不改写。

受控测试证明发布/排队时序，不单独证明C中断；实际客户端取消范围另见[客户端证据](q02-native-client-2026-10-01.md)。本批未使用私人账本、未运行GUI，也不重做上批Go会计/查询对照。

## 本批验证结果

最终串行`make mac-test`及`make app`均exit0。真实ContractTests、受控StateTests、NativeClientTests全部通过；开发应用arm64、deployment target13、ad-hoc包校验通过。未启动GUI，旧系统运行兼容性仍须实机证明。修改文档的本地链接、三项公开源码SHA及`git diff --check`通过。

## 接续与限制

下一步QueryView及原生导航接入，增加页面离开行为与受控profile/窗口deallocation覆盖。错误字节范围、保存query/预设、固定执行CSV、完整跨语言资源边界与GUI仍未交付。仅Store接口存在或本批检查通过不能关闭Q02/N03/N04。

## 公开源码身份

| 文件 | SHA256 |
| --- | --- |
| `apps/macos/src/LedgerStore.swift` | `f86b45f091f0619359530bd102fe993edb20311a18306f34e8cc3b8d52783c7e` |
| `apps/macos/tests/StateTests.swift` | `f60bfe0e5f92ab7b47ec6eb8a53b2c1baa509365a7dbb9b5db979bf84188b69b` |
| `apps/macos/tests/NativeClientTests.swift` | `ba7611264469a396e68f5b5e4688acf39bfbac6ae2c583797fbfabaf1a44dd6a` |
