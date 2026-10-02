# 完整兼容实施证据（2026-10-02）

本文件记录 ADR-0050 的实际实现与证据，不替代完整兼容验收。固定参考版本为 Beancount 3.2.3、BeanQuery 0.2.0，排除第三方 Python 插件执行。

## 本次源加载切片

已实现普通和递归 include glob，保留 Python glob 的文件系统枚举顺序而非人为排序；隐藏文件、字符集、Unicode、相对及绝对路径均有公开对照。新增取消、扫描工作量及深度限制，越限阻止有效快照发布。重复 include 与循环显式报错。v3 源声明按上游队列顺序收集，日期和指令优先级相同后按源 lineno 稳定排序；legacy 保留原递归排序。v3 会计 revision 更新为 v3-core-4。

搜索目录和缺失目录祖先纳入变更监测，新增匹配文件与新增递归目录可触发重载。文件替换即使大小与修改时间不变，设备号/inode 变化仍触发重载。插件声明是阻断错误，解析 AST 只用于定位，没有动态插件加载机制。导入接口先拒绝 include/plugin，保持 HTTP 400，避免插件解析错误绕过导入策略。Posting 的保留或重复 metadata 键显式报错。

`tools/reference/audit_compatibility.py` 生成上游能力登记表，记录源码摘要及 366 个签名/字段/AST/选项/指令条目。所有条目仍标记 pending：登记到一个能力并不意味着 Go 已实现，更不意味着通过验收。

## 当前验证

- `UV_OFFLINE=1 uv run --project tools/reference python tools/reference/verify_source_loading.py`：20 个 glob 模式与 6 个加载语义案例，对照同一文件系统及 Python loader，通过。
- `go test ./...`：通过。
- `go test -race ./internal/source ./internal/snapshot ./internal/ledger ./internal/web`：通过。
- `go vet ./...`：通过。
- `make parity-full query-parity`：51 个账务样例、215 个查询样例，零差异。
- `make mac-test`：四组真实原生测试通过（Export、Contract、State、NativeClient）。
- FinanceBook 只读同源捕获：47 文件、双方 2836 条 entries，零错误、零 entries/库存差异；Expenses 73 行、Income 22 行一致；真实 Swift/C 契约通过，3 个持仓截止日及 48 个价格检查通过。仅提交计数，不提交私人内容。

## 尚未封口

S01–S03 还需要 symlink 别名/目录循环、更多 metadata 非数字覆盖、构建期间新增/删除竞争、实际原生 glob 重载的对照证据；不能由上面的有限案例推断完整源兼容。C00 仍需补齐诊断规则与全部 grammar 组合，关联逐项 Go 实现与证据。

查询当前 typed 执行仍只支持 postings，Q10–Q14 未完成：全部标准表、字典/集合/结构化值、非有限 Decimal、全函数和运算符重载、子查询、HAVING、DISTINCT、PIVOT 及命令生态均需继续按设计实施。V01/V02 与默认 v3 迁移 M01 也未完成。现阶段不宣称完整兼容或 Beancount 的超集。

Glob 源加载不自动放开 glob 源导出；导出映射及原子发布另行验收，保持现有保守拒绝。
