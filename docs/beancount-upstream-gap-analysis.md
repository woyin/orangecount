# Beancount 上游生态缺陷与未完成工作分析（2026-08）

## 背景与方法

- **调研日期**：2026-08（上游数据快照截至 2026-08-14：各仓库 open issues、
  tags、changelog、设计文档均以当日抓取为准）。
- **方法**：只引用一手来源——GitHub issue/PR（经 GitHub API 逐条抓取原文）、
  仓库内 CHANGES/TODO 原文、官方文档（beancount.github.io、beancount vnext
  设计文档）、Fava 官方 changelog 与 tag 比对（`compare/v1.30.12...v1.30.14`）。
  博客/Reddit/HN 仅用于定位线索，不作为论据。无法溯源到一手文本的结论一律
  不写（例如检索引擎摘要声称 beancount TODO 含 load-cache/负库存条目，但直接
  抓取的 master TODO 全文关键词检索无命中，已弃用该说法）。
- **与 OrangeCount 的关联读法**：OC 以 Beancount v3 为会计语义契约、Fava
  1.30.12 为 UI 工作流参照，只服务 valid ledger（有 error 诊断即拒绝服务），
  不执行 Python 插件（FD-0001/D3 已批准排除），预算已推迟（ADR-0017）。因此
  每个问题点标注三类判读：
  - **避免型**：上游踩过的坑，OC 应在实现层显式规避或自查；
  - **机会型**：上游多年未解决的空白，可转化为 OC 路线图机会（注意与
    Beancount 兼容契约的偏差需按 CLAUDE.md 登记 `docs/fava-approved-deviations.md`
    或走 ADR）；
  - **观察型**：与 OC 现状相关但暂无行动必要，保持跟踪。
- 风格对齐 [docs/fava-transplant-gap-analysis.md](fava-transplant-gap-analysis.md)。
- **抽查复核**：落库前对 5 条最承重的引用（beancount #1040/#614、fava
  #2336/#1151、beanprice #141）做了独立检索复核，issue 标题与编号均与
  GitHub 官方页面吻合。

## 结论先行

上游生态的系统性弱点集中在五处：**(1)** booking/interpolation 语义长尾缺陷
（2015 年起的 `abs()` 价格插值 bug、FIFO 多币种成本错配、同日交易排序敏感）；
**(2)** 大账本加载性能是作者本人在设计文档中承认的痛点（"6 秒 vs 期望半秒内"），
C++/Rust 重写多年悬置，且默认开启的 pickle 缓存存在已报告未修复的 RCE 风险；
**(3)** 查询层（BeanQuery）与 SQL 语义的差距长期未收敛（NULL、UNION、窗口
聚合、AVERAGE/PRICE 函数缺失），且无迁移文档；**(4)** 导入生态（beangulp）
发布停滞（一年无 release）、与 Fava 的集成接缝处持续破损；**(5)** 预算、
账户间 lot 转账、价格源可靠性、文档-交易关联等"人人都要、多年未做"的能力
仍停在 issue/设计文档阶段。对 OC 而言，最大的三个可执行机会是：转账
（transfer）一等支持、价格源健壮性、以及把"valid-only + Go 常驻"变成性能
与正确性双重卖点；预算与 Python importer 相关机会与现有 ADR 冲突，须先走
ADR 流程。

---

## 1. beancount v3（github.com/beancount/beancount）

### 1.1 v2→v3 迁移与弃用时间线

| # | 问题 | 一手来源与摘录 | 影响面 | 判读 |
| --- | --- | --- | --- | --- |
| 1.1a | **v3 是"拆包"而非重写；v2 冻结政策靠口头约定** | [CHANGES（仓库原文，2024-06-16）](https://raw.githubusercontent.com/beancount/beancount/master/CHANGES)："Created branch 'v3' and released 3.0.0 … v2 is now even less subject to freeze exceptions than before … The default reply to patches to v2 will be 'move that to v3.'"；同条目明确 "v3: a new branch, to contain the state of the master branch MINUS the C++ bits"（C++ 重写移入 `cpp` 分支）。 | 生态分裂为 beancount/beangulp/beanquery/beanprice/beangrow/beancount2ledger 多包，用户需自行拼装 | 观察型（OC 只锚定 v3 语义，无 v2 包袱是既定优势） |
| 1.1b | **工具移除无平替迁移期** | 同上 CHANGES（2020-07-05）："Removed bean-report and all associated codes. Use bean-query instead (or Fava). Removed bean-web and bean-bake (for v3)." | 报表脚本用户被迫迁移 BQL | 观察型 |
| 1.1c | **导入 API 断裂（beancount.ingest → beangulp）** | [beangulp importer.py](https://github.com/beancount/beangulp/blob/master/beangulp/importer.py)：`ImporterProtocol` 标注为 old/deprecated；方法更名 `file_account()`→`account()`、`extract(file_memo, existing_entries)`→`extract(filepath, existing)`（`_FileMemo` 缓存对象→纯路径）；`bean-identify/bean-extract/bean-file` 三个 CLI 移除，改为 `beangulp.Ingest` 脚本（[设计文档](https://beancount.github.io/docs/beangulp_design_doc_for_new_repo/)）。 | 所有第三方 importer 需改写；大量社区 importer 长期停留在 v2 形态 | 观察型（OC 已排除 Python importer，FD-0001） |
| 1.1d | **Fava 1.30.13（2026-05-19）正式移除 Beancount v2 支持**，双栈期约 2 年 | [Fava changelog](https://beancount.github.io/fava/changelog.html) v1.30.13："This version drops support for Beancount version 2 … this mainly means you will have to switch importers to use beangulp." | v2 用户被硬切 | 观察型 |
| 1.1e | **v3 切换造成真实功能回退与回归** | [Fava changelog v1.30](https://beancount.github.io/fava/changelog.html)："Due to changes related to duplicate detection, duplicate detection is not automatically done by Fava here but can still be manually specified as hooks"（v3 导入路径下 Fava 不再自动查重）；[beancount #982](https://github.com/beancount/beancount/issues/982)（open）："Another regression I noticed when switching from v2 to v3"——`bean-format` 对带括号 balance 断言的列对齐在 v3 错位。 | 迁移用户体验 | 避免型：OC 的导入内建确定性查重（见 §4）；OC 的"保源导出/重写"路径必须保持无损（对照 §1.2g print_entries 有损） |

### 1.2 解析器 / booking 语义问题

| # | 问题 | 一手来源与摘录 | 影响面 | 判读 |
| --- | --- | --- | --- | --- |
| 1.2a | **价格插值使用 `abs()`：半个符号空间直接产生不平衡交易**（2015 年引入，至今 open） | [beancount #1040](https://github.com/beancount/beancount/issues/1040)（2026-08-01，open，3.2.3）："Interpolated price uses abs(), so half the sign space yields an unbalanced transaction"；`booking_full.py:1003 new_price_number = abs(weight / units.number)`；复现：`100.00 USD @` + `120.00 CAD` → 填 `@ 1.2 CAD` 后报 "Transaction does not balance: (240.000 CAD)"。作者考证 bug 自 8888b4c6b（2015）。 | 缺 `@` 价格的负数腿交易（卖空、退款） | 避免型/机会型：OC 的 interpolation 实现应显式拒绝或按符号求解；这是"比上游更正确的错误信息"的低成本差异点 |
| 1.2b | **tolerance 推断忽略仍在 MISSING 状态的币种** | [beancount #1004](https://github.com/beancount/beancount/issues/1004)（2026-01，open）："Interpolated currencies are missed when inferring tolerances … can still be `MISSING`; Beancount then applies a fallback tolerance … and only later reports the resulting imbalance." | 不完整交易的容差判定 | 避免型：OC 校验管线中 tolerance 计算应在 interpolation 完成后做，避免"先容差后报错"的错位诊断 |
| 1.2c | **合法 FIFO 交易被判 "Too many missing numbers"（P1，2020 年至今）** | [beancount #598](https://github.com/beancount/beancount/issues/598)（P1/booking，open since 2020-12-28）；[beancount #324](https://github.com/beancount/beancount/issues/324)（P2，2018）："When matching against a single lot, infer the currency … This ought to work, there's a single posting in the ante-inventory" | lot 转移/减少交易的可用性（详见 §5b） | 机会型：booking 推断能力是上游 P1 长期空白 |
| 1.2d | **多币种成本错配：FIFO 把商品本身填进成本币种** | [beancount #481](https://github.com/beancount/beancount/issues/481)（open，2018）："Booking FIFO errorneously fills in the commodity cost currency with the commodity itself" | 多币种 cost 的库存核算 | 避免型：OC 多币种成本路径需要针对性测试用例 |
| 1.2e | **同日交易顺序敏感（self-reduction / 中转账户）** | [beancount #602](https://github.com/beancount/beancount/issues/602)（P1，2021 至今）"Handle self-reductions"；[beancount #820](https://github.com/beancount/beancount/issues/820)（2024，open）：staking/unstaking 经中转账户时 "the unstaking transaction on the main account is processed before the one on the staking account. When the latter transaction is processed, there is no matching lot"，用户只能手工调序或改日期；vnext 设计文档对 self-reductions 仅标注 "Revisit." | 任何同日先减后增的账本（转账、staking） | 机会型：OC booking 可实现"同日批次内先 augment 后 reduce"或等价启发式（需登记与上游语义的偏差） |
| 1.2f | **零值自动腿被静默剪枝** | [beancount #1026](https://github.com/beancount/beancount/issues/1026)（PR，open，fixes #962）："a zero-valued auto-posting could be pruned before account-reference validation saw it, so an unknown placeholder account could fail to produce a validation error." | 校验正确性（未知账户漏报） | 避免型：OC 的 valid-only 前提使"校验漏报"成为高危类别，应保持显式零值腿 |
| 1.2g | **print_entries 有损：pad 被展开导致往返失真** | [beancount #1044](https://github.com/beancount/beancount/issues/1044)（2026-08-08，open）："print_entries destroys padding … There is an actual transaction inserted, so that the balance get unused, `bean-check` reports 'Unused Pad entry'." | 任何"读入→改写→输出"的 transformer 工作流 | 避免型：OC 的 download-journal/保源导出坚持按源码 span 切取（现行设计已如此），勿走 AST 重打印路径 |
| 1.2h | **数值工具违反文档契约** | [beancount #1043](https://github.com/beancount/beancount/issues/1043)（2026-08-08，open）：`round_to()` 文档称 "round down" 但 `int()` 向零截断，负数金额不向下取整（`round_to(-0.125, 0.01) → -0.12`，应为 -0.13）。 | 退款/贷记金额取整漂移 | 观察型：OC 的 Decimal 取整策略需在单测中锁定方向语义 |
| 1.2i | **账户名 i18n 限制多年未决** | [beancount PR #1022](https://github.com/beancount/beancount/pull/1022)（open since 2026-03，未合并）自述修复 #733/#398/#161/#377："Chinese characters cannot be used at the start of account names, because only upper case letters are allowed"。 | 中文等非大小写文字用户 | 机会型：OC 解析器若放宽为 Unicode 账户名（UAX#31 风格），属于对契约的超集扩展，需登记偏差 |
| 1.2j | **bean-format 对齐边角缺陷** | [beancount #982](https://github.com/beancount/beancount/issues/982)（open）：括号 balance 表达式 `-w/-W` 下错位。 | 格式化工具 | 观察型 |

### 1.3 性能

| # | 问题 | 一手来源与摘录 | 影响面 | 判读 |
| --- | --- | --- | --- | --- |
| 1.3a | **加载慢是作者自认的核心痛点；"instant" 目标 6 年未达成** | [Beancount Vnext 设计文档（官方 docs，Martin Blais）](https://beancount.github.io/docs/beancount_v3/)："The current state of Beancount is that development has been static for a while now"；"I really do want that 'instant' feeling … that it runs in *well under half a second*"（对比其本人账本 6 秒）；附录："my own file went from 4s -> 0.3ms for the parsing stage of the largest file"（C++ 原型，未进入发布分支）。 | 大账本用户、每次命令行调用 | 避免型/机会型：Go 常驻 serve + 增量加载是 OC 的结构性优势，应在路线图中作为可宣传差异点 |
| 1.3b | **重写路线多次改道，社区实现悬置** | [CHANGES 2024-06-16](https://raw.githubusercontent.com/beancount/beancount/master/CHANGES)：C++ 工作移入 `cpp` 分支，"If things do move to Rust, I will probably salvage bits and pieces"；[beancount #829 "Next Version / Rewrite"](https://github.com/beancount/beancount/issues/829)（pinned，2024-06 起 open，74 评论，指向两份详细设计文档）；[PR #869 "feat: add a rust parser"](https://github.com/beancount/beancount/pull/869)（2024-11 起 open 未合并）。 | 生态性能前景不确定 | 观察型 |
| 1.3c | **默认开启的 pickle 加载缓存存在已报告未修复的 RCE（CWE-502）** | [beancount PR #1034](https://github.com/beancount/beancount/pull/1034)（2026-06，open）："The default pickle cache is stored alongside the .beancount file … `pickle.load()` is called without any integrity verification … executes arbitrary Python code the next time any beancount command processes that file. The cache is enabled by default."（在 3.2.3 动态验证，CVSS 7.8）。vnext 文档亦将 "removal of the pickle cache" 列为目标。 | 共享目录/协作仓库场景的全部 CLI 用户 | 避免型：OC 如做加载缓存，必须（a）不放在账本同目录、（b）带完整性校验或用非执行型格式、（c）失效判定先于反序列化 |
| 1.3d | **Python 进程模型本身的开销** | 同 1.3a 设计文档（parser→Python 回调成本、protobuf 输出构想 "processing from other languages (e.g. Go) will have first-class support"——未落地）。 | 每次查询/校验冷启动 | 避免型（OC 天然规避） |

---

## 2. BeanQuery（github.com/beancount/beanquery）

背景：Fava 1.30 起查询由 beanquery 提供，官方 changelog 明示语义不兼容：
"Beancount query support is now provided by the beanquery package, which has
some minor differences in syntax, the provided columns and functions … For
extensions using Fava's query_shell directly, this will lead to breakage"
（[Fava changelog v1.30](https://beancount.github.io/fava/changelog.html)）。

| # | 问题 | 一手来源与摘录 | 影响面 | 判读 |
| --- | --- | --- | --- | --- |
| 2a | **NULL 处理架构悬而未决（2022 年 idea 至今）** | [beanquery #76](https://github.com/beancount/beanquery/issues/76)（label: idea，2022-04，open）："Alternative way to handle NULLs … Instead than bubbling up NULLs through the BQL call stack, we can simply raise a dedicate exception"；[#127](https://github.com/beancount/beanquery/issues/127)（2023，open）："Make the division operator handle division by zero returning `NULL` as in SQL and eventually remove the `SAFE_DIV()` function." | 除零、缺值列的所有查询 | 机会型：OC 查询层应一次性定义 NULL 语义（三值逻辑）并写进契约文档，避免上游的两套机制并存 |
| 2b | **聚合/函数覆盖缺口** | [#177](https://github.com/beancount/beanquery/issues/177)（2024，open）"Implement `AVERAGE(inventory)`"（附 blais 原 TODO 设想）；[#187](https://github.com/beancount/beanquery/issues/187)（2024，open）"Add a `PRICE(currency, costcurrency, date)` function"；[#189](https://github.com/beancount/beanquery/issues/189)（2024，open）：`balance` 按逐 posting 报告，"This happens because each posting is considered in isolation"，需要窗口式聚合而无窗口函数；[#85](https://github.com/beancount/beanquery/issues/85)（2022，open）：Amount 算术运算不支持。 | 报表类查询表达力 | 机会型：OC 查询契约可声明函数超集（AVERAGE/PRICE/窗口 balance），标注为扩展并列 deviation |
| 2c | **无 UNION / GROUPING SETS** | [PR #281](https://github.com/beancount/beanquery/pull/281)（2026-05，open 未合并）"Implement UNION / UNION ALL"，并说明是 #265（GROUP BY GROUPING SETS/ROLLUP）的前置；[PR #282](https://github.com/beancount/beanquery/pull/282)（2026-05，open）：新增 `custom` 表以支持查询 custom 指令——**当前 custom 指令（含 budget 形态）不可查询**。 | 合并报表、预算类查询 | 机会型（custom 表对预算联动尤其关键，但受 ADR-0017 约束） |
| 2d | **日期函数语义债务** | [#152](https://github.com/beancount/beanquery/issues/152)（2023，open）："The one argument form of the `parse_date()` … could interpret different rows differently. It should be deprecated"；[#239](https://github.com/beancount/beanquery/issues/239)（2025，open）"Deprecations in 0.3.0"：将弃用 `year/month/day` 列与函数、双引号字符串、`run_query()` 模块等。 | 依赖旧函数的查询稳定性 | 观察型：OC 的 FQL/查询子集应避开被弃用面 |
| 2e | **输出与文档可用性** | [#285](https://github.com/beancount/beanquery/issues/285)（2026-06，open）：`select distinct accounts` CLI 输出 "inconsistent with 0-4 accounts per line"（set 列渲染混乱，CSV 同样）；[#284](https://github.com/beancount/beanquery/issues/284)：COALESCE 可用但 `.help targets` 不列出；[#191](https://github.com/beancount/beanquery/issues/191)（2024，open，documentation）："Are there any migration instructions anywhere? … trying to convert my code from beancount 2 over to beanquery is right now more a trial and error"（含 `value(position, #"2020-12-31")` → `value(position, date("2020-12-31"))`、返回行不再为 named tuple）；[#278](https://github.com/beancount/beanquery/issues/278)：社区交互手册请求挂 README。 | 嵌入式/脚本用户、新手 | 避免型/机会型：OC 对 BeanQuery 兼容契约应自带"支持子集 + 差异清单"文档（上游缺迁移文档的教训） |
| 2f | **零和列消失** | [#271](https://github.com/beancount/beanquery/issues/271)（2026-03，open）："Column disappears in query output when sum of a position for all rows is ZERO"。 | 对账/核验查询 | 避免型：OC 查询输出对空 Inventory 应保留列（我们的 GenericReport 已按列契约渲染，需加此用例） |

---

## 3. Fava 1.30.x（github.com/beancount/fava）

### 3.1 固定 1.30.12 相对上游的已知落后点

- 最新版本线：changelog 只记录到 **v1.30.13（2026-05-19）**：drop Beancount 2
  支持、翻译迁至 Weblate、新增韩语、移除 `unrealized` fava-option（改用
  Beancount 选项 `account_unrealized_gains`）（[changelog](https://beancount.github.io/fava/changelog.html)）。
- `v1.30.12...v1.30.14` tag 比对共 **43 个 commit**（[compare](https://github.com/beancount/fava/compare/v1.30.12...v1.30.14)）：
  抽样审读 + 对整段比对文本检索 `fix:`/`fix(` 无命中，主题集中于——
  `drop Beancount 2 support`、Weblate/i18n 迁移与翻译批量更新、`deps & lint`、
  makefile/构建调整。即：**1.30.13/1.30.14 没有落入 changelog 的功能性 bugfix
  条目，且 patch 版本本身不逐条记 changelog**（置信度：主题分布=高；
  "无功能修复"=中高，基于 commit message 全文检索而非逐 diff 审读）。
- 1.30.14 仍有真实缺陷在案：[#2336](https://github.com/beancount/fava/issues/2336)（2026-08-12，对 v1.30.14 提出）："Holdings report uses future prices for market value when date filter is applied"——holdings 页 BQL 的 `value(...)` 未传过滤区间终点。
- **判读**：避免型——OC 的 holdings/估值路径必须显式将 time filter 终点传入
  价格查询（与 #2336 同类语义自查）；观察型——升级到 1.30.13+ 的收益主要是
  v2 移除与 i18n 基建，对 OC 参照面影响小。

### 3.2 open issues 中的用户痛点

| # | 问题 | 一手来源与摘录 | 影响面 | 判读 |
| --- | --- | --- | --- | --- |
| 3.2a | **对账工作流缺一键 flag 翻转（2020 至今，help wanted）** | [fava #1151](https://github.com/beancount/fava/issues/1151)（2020-08，open）："Support quickly flipping transaction/posting flag between '*' and '!' … would allow quickly reconciling the ledger against a bank statement, like it's possible e.g. in GnuCash." | 月度对账效率 | 机会型：journal 行内 flag 切换写回（OC 已有 add-entries/原子写管线，增量成本低） |
| 3.2b | **导入流程接缝破损（beangulp 集成）** | [#2271](https://github.com/beancount/fava/issues/2271)（2026-05，open）：官方 beangulp 示例 `import.py` 在 Fava 内 `ModuleNotFoundError`（`run_path` 不把配置目录加入 `sys.path`）；[#2179](https://github.com/beancount/fava/issues/2179)（2026-01，bug，open）："beangulp-style hook detection is very fragile"（靠类型注解字符串探测 hook 签名，callable 对象/非字符串注解即失效）；[#1585](https://github.com/beancount/fava/issues/1585)（2023，help wanted）：导入含 price 指令的条目被前端校验拒绝；[#1607](https://github.com/beancount/fava/issues/1607)（2023，bug）：`Decimal` 元数据导入触发 500（`ValueError: Unexpected value: '114'`）；[#980](https://github.com/beancount/fava/issues/980)（2019，bug）：balance 元数据导入错渲染；[#1188](https://github.com/beancount/fava/issues/1188)（2020）：auto_accounts 插件产生的账户被 "Should be one of the declared accounts" 拒绝。 | 导入是 Fava 高频路径，长期带伤 | 避免型：OC 导入缓冲区的类型化元数据序列化 + 声明式 adapter 应在单测覆盖 Decimal/Amount/price 指令等边角；Python importer 本体的等价复刻受 FD-0001 约束 |
| 3.2c | **扩展系统易碎点** | [#2321](https://github.com/beancount/fava/pull/2321)（2026-07，open PR）："an exception in an extension endpoint bubbled up to Flask's default HTML 500 page, making it hard for extension UIs to show a friendly message"；v1.30 changelog：beanquery 切换对使用 query_shell 的扩展造成 breakage。 | 扩展生态 | 观察型（OC 无 Python 扩展面，但 API 错误契约应统一 JSON） |
| 3.2d | **性能路径的实验未落地** | [#2189](https://github.com/beancount/fava/pull/2189)（draft，2026-01，yagebu 本人）："draft: add support for uromyces … to load files (instead of the corresponding Beancount functionality)"——用 Rust 解析器替换 Python loader 的试验仍处草稿。 | 大账本 Fava 用户 | 观察型：印证 §1.3 性能瓶颈向 UI 层传导 |
| 3.2e | **图表/可视化诉求多年悬置** | [#1171](https://github.com/beancount/fava/issues/1171)（2020，9 👍，help wanted，open）："add average (and maybe other quantiles) to bar charts"；实现 PR [#2314](https://github.com/beancount/fava/pull/2314)（2026-07，open 未合并，对应 #2148 的损益表区间均值）；[#2299](https://github.com/beancount/fava/pull/2299)（open）：`0.01 USD` 式价格显示的自适应基数改进；[#2301](https://github.com/beancount/fava/pull/2301)（open）：账户树层级视觉强化。 | 图表可读性 | 机会型：bar chart 均值线、自适应价格显示可作为 OC 图表扩展（登记偏差后超越参照） |
| 3.2f | **移动端体验** | [#979](https://github.com/beancount/fava/issues/979)（2019，platform:mobile，help wanted，open）："The current document upload implementation depends on dropping a file on a journal entry … I am not sure how to do it on a mobile platform."；[#2161](https://github.com/beancount/fava/issues/2161)（2025-12，platform:mobile + bug，open）：iOS 16.7 Safari UI 错位（截图在案）。 | 手机记账/查账 | 机会型（中长期）：见 §5f |

---

## 4. 导入生态（beangulp / importer 框架）

| # | 问题 | 一手来源与摘录 | 影响面 | 判读 |
| --- | --- | --- | --- | --- |
| 4a | **发布停滞：一年无 release，安装文档不含 beangulp** | [beangulp #203](https://github.com/beancount/beangulp/issues/203)（2026-01，open，3 👍）："`pip install beangulp` pulls the most recent release that is now a year old (v0.2.0) … The beancount installation doc … has no mention of beangulp. Are new beangulp releases planned?" | 新用户上手、bugfix 分发 | 观察型（上游维护节奏事实）；机会型（OC 自带导入不依赖该包） |
| 4b | **命名约定与 Fava 不一致且无文档** | [beangulp #195](https://github.com/beancount/beangulp/issues/195)（2025-12，open）：Fava `ingest.py` 期望模块级变量 `CONFIG`/`HOOKS`，而 beangulp 官方示例用 `importers`/`hooks`——"This is confusing and not documented anywhere." | Fava 导入用户必踩 | 避免型：OC 的导入配置必须是声明式（非 Python 模块级变量），从根上消除此类接缝 |
| 4c | **官方示例在 Fava 中跑不通** | fava [#2271](https://github.com/beancount/fava/issues/2271)（见 3.2b）：beangulp 示例目录结构在 Fava `run_path` 下 `ModuleNotFoundError`。 | 同上 | 同上 |
| 4d | **标准 importer 覆盖不全** | [beangulp #77](https://github.com/beancount/beangulp/issues/77)（2021-05，open）："there is no standard OFX importer (there is one in the examples but there is not one in the library)"；[#199](https://github.com/beancount/beangulp/pull/199)（2026-01，open PR，fixes #196）：CSVReader 无法处理可变行数的头/尾，用户被迫临时文件 hack。 | OFX 用户、非规整 CSV | 机会型：OC 的 Go 原生 CSV/QFX 解析器若覆盖常见银行格式即形成差异（注意 FD-0001 排除的是 Python importer 生态，非导入能力本身） |
| 4e | **平台边角：Windows 编码崩溃** | [beangulp #206](https://github.com/beancount/beangulp/issues/206)（2026-03，open）：`beangulp extract -o` 在 Windows 上对日文 payee 抛 `UnicodeEncodeError`（`click.File('w')` 用 cp1252）。 | Windows 用户 | 避免型：OC 全链路强制 UTF-8 读写 |
| 4f | **工作流语义缺口：多 importer/退出码/缓存** | [#182](https://github.com/beancount/beangulp/pull/182)（2025-07，open PR）：identify 无匹配时无退出码，且 "beangulp does not permit multiple importers to be able to import a given file"（无法做 fallback importer）；[#181](https://github.com/beancount/beangulp/pull/181)（2025-07，open PR）：`beangulp.cache` 已弃用但 `simple_cache` "not released yet"，提议干脆改用第三方 cachier。 | 导入自动化脚本 | 机会型：OC 导入管线内建 fallback adapter、确定性退出码与结果分类 |
| 4g | **Fava v3 路径下自动查重被移除** | [Fava changelog v1.30](https://beancount.github.io/fava/changelog.html)（见 1.1e）。 | 重复交易风险 | 避免型/机会型：OC 的导入 commit 前强制相似度查重（上游从"自动"退到"手动 hooks"） |

**维护状态小结**：beangulp issues 持续有新报告（2025-07 至 2026-07 活跃），
但 release/merge 节奏慢（4a/4f 的 PR 悬置 6-12 个月）；Fava 侧集成代码
（`fava/core/ingest.py`）与 beangulp 示例的契约漂移是反复出问题的接缝。

---

## 5. 多年未落地的规划能力

| 能力 | 上游现状（一手证据） | 影响面 | OC 判读 |
| --- | --- | --- | --- |
| **a. 预算（budgeting）** | beancount core 的 open issue 检索（"budget"）无直接特性跟踪（仅 #131 DisplayContext 提及预算邮件线程、#258 marker directive 旁及）；vnext 设计文档将 "budgeting/balance inequalities" 列为设想的核心特性，未实现；Fava 的预算是 2016 年 UI 级实现：changelog v0.3.0（2016-03）"Simple budgeting functionality in the Account view. See help pages on how to use budgets. [#294]"，v1.1 "bar charts on account pages now also show budgets"，v1.4 "Budgets are now accumulated over all children"——基于 `custom "budget"` 指令，无 core 语义。 | 记账软件最高频诉求之一 | **机会型，但与 ADR-0017 冲突**：任何预算排期须先修订/重开 ADR-0017；上游十年未标准化说明语义设计空间仍开放（beanquery #282 的 custom 表缺失也卡住了预算查询） |
| **b. 账户间转账（lot transfer / inter-account transfers）** | [beancount #614](https://github.com/beancount/beancount/issues/614)（blais 本人开，2021-01，P2/booking，12 👍，open）："Add syntax for automatically transferring lots at cost (crypto users) … This type of transfer is a recurring question on the mailling-list"；[#564](https://github.com/beancount/beancount/issues/564)（2020-10，16 👍，open）："Feature request for v3: Lot transfers … It is very tedious to specify the lots manually, but it seems that at the moment it is the only way."；配套语义坑见 §1.2c/#598、#324、#820；仓库 TODO 的 "Settlement Dates (Split & Merge)" 章节（proposal-settlement：两半交易跨账户合并）规划多年未动；第三方插件路径亦破损（[#732](https://github.com/beancount/beancount/issues/732)）。 | 加密货币/多券商/多钱包用户；`#transfer` tag 只是约定无语义 | **机会型（高价值）**：OC 可在语法（如 `{}` 增强腿推断）或导入期检测（同日对冲腿识别）上做出上游六年未做的能力；语法级扩展需登记契约偏差 |
| **c. 价格获取（beanprice）** | 仓库 46 个 open issues、最后 push 2026-02-07（[repo API](https://api.github.com/repos/beancount/beanprice)）；[#64](https://github.com/beancount/beanprice/issues/64)（2021-10，open）"OANDA source no longer works"；[#141](https://github.com/beancount/beanprice/issues/141)（2026-07，open）：Yahoo 源硬编码 `exchange: "NYSE"`，对 LSE 标的返回 2019 年陈旧价，"corrupted a real personal-finance net worth calculation before being caught"；[#123](https://github.com/beancount/beanprice/issues/123)（2025-12，open）：东京时区请求美股拿到 T-1 收盘价（`16:00 local` 假设）；[#142](https://github.com/beancount/beanprice/pull/142)（2026-07，open）：天天基金 API 迁移 404；新源 PR 悬置（[#129 FT 源](https://github.com/beancount/beanprice/pull/129) 11 评论未合并、[#140 UniRateAPI](https://github.com/beancount/beanprice/pull/140)）；跨仓修复割裂（beancount [#1013](https://github.com/beancount/beancount/pull/1013) 只修 beanprice#134 一半）。 | 估值可信度（静默错误最危险） | **机会型（高价值）**：OC 内建价格模块应做到：多源冗余 + 交叉校验、陈旧度/偏离度告警、按市场时区取收盘而非本机时区、失败显式化——直击上游"静默错误"痛点 |
| **d. 文档管理（document↔交易关联）** | beancount TODO（master 原文）"Move Much of Core to Plugins → Document Directives" 整节未竟："Documents found in parent directories don't end up creating a directive … this is probably not what we want"；"Document finding from files should not create documents that have been explicitly specified in the ledger. Avoid duplication! **This is an important fix to make**"；以及 document↔transaction 用 `^link` 关联、"doc:" metadata 转换等设想全部停留在 idea。 | 票据/对账单管理 | 观察型→机会型：OC 已有上传/移动/预览（H2/H4 移植），下一步"文档-交易自动关联（金额+日期匹配建议 link）"是上游只想未做的点 |
| **e. 多账本 / 合并** | beancount open issues 检索 "consolidate"/"multiple ledgers" 零命中——合并能力甚至不在 core 跟踪范围内；include 为单根模型；Fava 仅提供多文件切换（changelog v0.3.0："Support for switching between multiple beancount files. [#213]"）；vnext 文档的 "better includes" 未落地；TODO 的 settlement 章节涉及"两半交易合并"。（置信度：中——基于缺席证据+设计文档） | 家庭/多实体合并报表 | 观察型：OC 的单账本定位与上游一致；如未来做多账本，属全新增量而非移植 |
| **f. 移动端** | 生态无官方移动客户端项目（置信度：低，基于"无对应仓库/issue"的缺席证据）；Fava 侧仅响应式 Web，且 platform:mobile 标签下长期积压（#979 2019 年上传不可用、#2161 2025 年 iOS Safari 布局 bug，均 open，见 3.2f）。 | 移动记账 | 观察型（中长期机会）：Go 单二进制 + PWA/自托管轻客户端是 OC 结构性可行的路线；无上游参照物，超出 Fava-parity 范畴，需产品决策 |

---

## 优化点优先级表

评级维度：用户价值 × 实现成本（高/中/低粗分）。**ADR 冲突列**标出与 OC
现有决策相抵、需先走 ADR/偏差登记流程的项（ADR-0017 预算推迟；FD-0001/D3
排除 Python importer；FD-0002/FD-0004 valid-only 服务约束；CLAUDE.md 的
Fava-parity 偏差登记纪律）。

| 优先级 | 优化点 | 上游依据 | 用户价值 | 成本 | 类型 | ADR 冲突 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 常驻加载性能与增量重载作为卖点（大账本亚秒响应；安全缓存设计） | vnext 文档 6s/"half a second"；#1034 pickle RCE；#869/#829 重写悬置 | 高 | 中 | 避免型 | 无（缓存不得复制 pickle 模式） |
| 2 | 账户间 lot 转账：`{}` 增强腿推断 / 导入期转账检测 | #614（P2，12👍）、#564（16👍）、#598（P1）、#820 | 高 | 中 | 机会型 | 语法扩展需登记 Beancount 契约偏差 |
| 3 | 价格模块：多源+交叉校验+陈旧度/时区告警 | beanprice #141/#123/#64/#142；PR #129/#140 悬置 | 高 | 中 | 机会型 | 无 |
| 4 | 导入管线内建确定性查重 + 类型化元数据/边角单测 | Fava v1.30 changelog（自动查重移除）；fava #1607/#1585/#980 | 高 | 中 | 避免型 | Python importer 等价复刻仍受 FD-0001/D3 约束（本项不含） |
| 5 | 对账工作流：journal 行内 flag `*`/`!` 快速翻转写回 | fava #1151（2020 至今，help wanted） | 中 | 低 | 机会型 | 无 |
| 6 | 查询契约文档化 + NULL/除零语义一次性定义；补 AVERAGE/PRICE/窗口 balance 扩展 | beanquery #76/#127/#177/#187/#189；#191 迁移文档缺失 | 中 | 中 | 机会型 | 函数超集需登记查询层偏差 |
| 7 | 估值/holdings 的 time-filter 终点语义自查（防"未来价格"） | fava #2336（1.30.14 在案缺陷） | 中 | 低 | 避免型 | 无 |
| 8 | Unicode 账户名（UAX#31 风格超集） | beancount PR #1022（修复 #733/#398/#161/#377，悬置） | 中 | 低 | 机会型 | 契约超集需登记偏差 |
| 9 | 图表增强：bar chart 均值/分位线、自适应价格显示 | fava #1171（9👍）/#2314/#2299 | 中 | 低 | 机会型 | 超 Fava 1.30.12 参照面，登记偏差 |
| 10 | 预算能力（custom directive 查询 + 预算报表） | vnext 设想；Fava v0.3.0 #294；beanquery #282 | 高 | 高 | 机会型 | **冲突 ADR-0017：须先修订 ADR 再排期** |
| 11 | 带错服务/诊断分级（errors 页服务全部诊断） | fava 全量 /errors 行为 vs OC FD-0004 | 中 | 中 | 避免型 | **冲突 FD-0004/FD-0002（valid-only）：需 ADR 才能重开** |
| 12 | 移动端可用性（PWA/轻客户端、移动上传入口） | fava #979/#2161（platform:mobile） | 中 | 高 | 观察型 | 超出 Fava-parity 范畴，需产品 ADR |
| 13 | Python importer 兼容层（任意重开讨论） | §4 全节（beangulp 拆分之痛） | 低（对 OC 用户群） | 高 | 观察型 | **冲突 FD-0001/D3（已批准排除）：重开须 ADR** |

## 附:方法与置信度备注

1. 所有 issue 状态（open/closed、日期、👍 数）以 2026-08 GitHub API 抓取为
   准；引文为 issue 原文节选，未做改写。
2. "Fava 1.30.13/1.30.14 无功能性修复"结论置信度中高：基于 changelog 官方
   条目 + `v1.30.12...v1.30.14` 全部 43 个 commit 的 message 全文检索
   （`fix:` 无命中）与抽样审读，未逐 diff 审查。
3. 多账本合并、移动端"无官方项目"两处为缺席证据型结论，已标注中/低置信度。
4. 上游 issue 的开放状态会变化；若本文用于排期，建议在立项时复核对应 issue
   链接的最新状态（尤其 beancount #1040/#1034 与 beanprice #141）。

---

## 附：OrangeCount 本方基线盘点（2026-08-15，与上文对照用）

调研同时对本方实现做了盘点，作为对照上游缺口时的"本方事实"：

| 维度 | 本方现状 |
| --- | --- |
| 运行时 | Go 单二进制、离线、无 Python 依赖（上游 §1.3 的启动/加载开销不存在） |
| 服务模型 | valid-only 快照（FD-0004），结构性排除上游"带错账本照样出报表"的不一致 |
| Booking | NONE / 精确归约（≈STRICT）/ FIFO / LIFO / HIFO / AVERAGE 全实现（ADR-0042） |
| 指令集 | v3 核心 17 种指令全覆盖（含 pad、custom、query、document、note） |
| BeanQuery | `sum/count/min/max/first/last/avg`、`year/month/day`、`has_tag/has_link` |
| 已知空白 | 预算（ADR-0017）、Python 导入生态（FD-0001）、插件执行（intentional-boundary） |

本方待办确认项（与上文机会型条目呼应）：

1. **BeanQuery 函数面窄于上游**：`internal/query/query.go` 的
   `functionExpr.eval` 遇到未实现函数（如 `length/abs/root/currency/
   number/units/cost/weight/position/meta/filename/lineno`）直接报
   `unsupported function`——对应 §2b/§6 的函数超集机会。
2. **HIFO booking 为本方扩展拼写**：上游 v3 文档方法为
   STRICT/FIFO/LIFO/AVERAGE/NONE（`isOrderedBooking` 额外接受 hifo），
   是否保留应登记契约偏差或移除。
3. **interpolation 符号语义**：本方 `inferElision`/价格推断需对照
   §1.2a（abs() 符号 bug）补符号方向测试用例。
4. **holdings 估值时间语义**：本方 `marketHoldingValue` 已按 as-of 过滤
   价格（fixture 覆盖），对应 §7/fava #2336 的"未来价格"自查已满足，
   但应补 time-filter 端点联调用例防回归。
