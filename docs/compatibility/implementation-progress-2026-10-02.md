# 完整兼容实施证据（2026-10-02）

本文件记录 ADR-0050 的实际实现与证据，不替代完整兼容验收。固定参考版本为 Beancount 3.2.3、BeanQuery 0.2.0，排除第三方 Python 插件执行。

## 本次源加载切片

已实现普通和递归 include glob，保留 Python glob 的文件系统枚举顺序而非人为排序；隐藏文件、字符集、Unicode、相对及绝对路径均有公开对照。新增取消、扫描工作量及深度限制，越限阻止有效快照发布。重复 include 与循环显式报错。v3 源声明按上游队列顺序收集，日期和指令优先级相同后按源 lineno 稳定排序；legacy 保留原递归排序。v3 会计 revision 更新为 v3-core-4。

搜索目录和缺失目录祖先纳入变更监测，新增匹配文件与新增递归目录可触发重载。文件替换即使大小与修改时间不变，设备号/inode 变化仍触发重载。插件声明是阻断错误，解析 AST 只用于定位，没有动态插件加载机制。导入接口先拒绝 include/plugin，保持 HTTP 400，避免插件解析错误绕过导入策略。Posting 的保留或重复 metadata 键显式报错。

`tools/reference/audit_compatibility.py` 生成上游能力登记表，记录源码摘要及 366 个签名/字段/AST/选项/指令条目。**2026-10-02 当时**所有条目均标记 pending：登记到一个能力并不意味着 Go 已实现，更不意味着通过验收。

> 当前状态（2026-10-28）：inventory 366 项全部 verified，由 `go run ./tools/paritygates` 与 `tools/favagates` 校验；门槛判据与证据见 [parity-gates-2026-10-27.md](../parity-gates-2026-10-27.md)。本文保留 2026-10-02 的原始结论与日期，避免日期记录被改写成后来状态。

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

源切片冻结时 typed 执行仍只支持 postings。随后 Q10/Q11 的实施见后文；Q10–Q14 整体仍未完成：全部标准表、字典/集合/结构化值、非有限 Decimal、全函数和运算符重载、子查询、HAVING、DISTINCT、PIVOT 及命令生态均需继续按设计实施。V01/V02 与默认 v3 迁移 M01 也未完成。现阶段不宣称完整兼容或 Beancount 的超集。

Glob 源加载不自动放开 glob 源导出；导出映射及原子发布另行验收，保持现有保守拒绝。

## Q10 首批标量表切片

已实现 typed 表级编译上下文及取数接口，支持 entries、transactions、accounts、commodities、balances、notes、events、documents、prices 的标量字段查询。列名在读行之前绑定到所选表，非法列在零行或 LIMIT 0 时也拒绝。标准 `#table` 与双引号源名可执行。prices/balances 的 amount 使用既有精确金额 wire；document 文件名与上游一致解析为绝对路径。AST 记录显式 payee 的存在，避免空字符串变为 NULL，标准源导出也保持该区别。

此处只交付表的标量部分，**Q10 未封口**：id/hash、结构化 open/close、meta、集合、transaction entry 等字段依赖后续类型与函数切片。全标准表字段与 wildcard 仍需补齐，不把未实现字段归为契约外。

当前增量验证：33 个新独立 Python query goldens 加入原 215 个案例，共 248 个零差异；51 个账务差分通过。Go 全量、vet、query/desktop/bridge/dialect race 通过；四组原生测试通过，新增标准表及 Amount 返回值真实 Swift/C 检查通过；旧 query goldens `--check` 全部通过；源导出 35 个 roundtrip、71 个库存前缀及 6 个拒绝通过。FinanceBook 使用本次新 C archive/Swift probe 再次只读验证，仍为 47 文件、2836 entries、零账务/库存差异，原生契约及价格/持仓检查通过。

## Q10/Q11 结构化值与返回格式切片

Go、原生解码/查询显示和查询分析文件读取已增加递归字典、列表、字符串集合及 open/close/transaction 的当前结构字段。query contract 升级为 typed-v3；旧 v1/v2 分析仍可读取。Unicode 字典键以有序键值数组保留，重复/乱序/错误 NULL/深层嵌套/缺少结构字段均有 Go 和真实 Swift 解码反例。Posting meta、交易 meta 与系统 __tolerances__ 保持分离。源 NULL 与显式空 payee 得以保留；v3 revision 为 v3-core-5。

标准 wildcard 用声明列序展开，包含零行 schema；entries 的 id/hash 尚缺，因此该表 wildcard 仍不能算完成。新 price 和 cost 字段有独立 oracle，其中 @@ 的 price 修正为上游单位价格及 Decimal 精度。当前公开查询案例为 284 个，零差异；Q10/Q11 仍为部分实施，不能从这些案例推断全语言兼容。具体 wire 及剩余限制见 [query-typed-v3.md](query-typed-v3.md)。

本切片冻结验证：Go 全量、vet、query/desktop/bridge/ledger race、51 账务差分、284 查询差分均通过；17 个 query corpora 的 Python `--check` 全部通过。原生四组测试通过，包含递归结构值及 Unicode 键保留/拒绝反例。源导出增至 36 个 roundtrip、74 个库存前缀、6 个拒绝；源加载 20 个模式及 6 个语义对照仍通过。FinanceBook 在 v3-core-5/typed-v3 的最新真实 C/Swift probe 下，仍为 47 文件、双方 2836 条记录，零账务/库存差异；Expenses 73 行、Income 22 行一致，3 个持仓截止日、48 个价格检查通过。

已从干净 `13f00ee` 完成 0.1.6 全量 SwiftUI App 编译、签名和 App/CLI 打包；两包仓库外解压验收通过。GUI 连接本次报错，未计为实际点击验收。版本、校验值及完整限制见[0.1.6 交付](../native-development-release-0.1.6.md)。
