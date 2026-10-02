# Q02 捕获查询的分析容器内容生成

日期：2026-10-01。基线 `9e5d794`，接续 typed CSV。

新增 `QueryResponse.WriteAnalysis`，只接受成功执行响应，生成格式版本 1 的 `.ocanalysis` ZIP，成员为 `context.json`、`schema.json`、`cells.ndjson`、`data.csv`。上下文记录快照、语义 profile/revision、原始执行身份与文本、typed contract、行数、行上限、截断和 captured-result 范围；query 没有页面，page 为 null。complete 表示该捕获执行未被资源行上限截断，不表示整个账本或所有查询匹配的完整投影。

列序和重复标题保留；typed 行逐行 JSON、零行时 cells 文件零字节。CSV 使用已有精确 writer。接口不读取 Session、源图、附件或诊断，之后执行失败不会替换传入的成功响应。本接口接收 Go 内存响应，尚无 Swift 回传解码或 Session 导出缓存。

## 当前验证

- 公开 desktop seam 的首个测试因缺少 WriteAnalysis/QueryAnalysisContext 编译失败；实现后通过。
- 实际查询→ZIP 解包验证恰好四个成员、捕获文本/快照/行数、schema 原样、typed 行原样、重复标题与高精度 CSV。
- 零行实际查询保留 schema 和 CSV 标题，没有生成数据行。
- 实际失败查询导出被拒绝且没有写出字节；关闭 writer 错误传播，之后仍能独立导出此前成功响应。
- 当前全量 Go、desktop/query 两包 race、fmt/vet/license exit 0。

## 尚未完成

原生固定捕获导出入口、独立 CSV 操作、保存面板、原子文件替换、源图/附件目标保护仍待实现。调用方需要冻结响应；当前流式内容 writer 不保护目标文件，也不防止调用方改写传入结构。不得将本接口当成完整 X02 验收。

容器没有加入单独的来源路径成员，但用户查询文本或字符串单元格可能自带路径；来源列的导出隐私策略及绝对来源路径排除仍需明确验证。多币种/完整 compound、截断、partial/unknown 的容器实例和独立解包验证器尚需补齐。原生和私人账本未在本批重新验证。
