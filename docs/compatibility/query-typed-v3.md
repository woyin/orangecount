# Typed query v3 返回与保存契约

参考版本：BeanQuery 0.2.0。当前返回标识为 `beanquery-0.2.0-typed-v3`；旧 v1/v2 查询分析文件继续可读，不能在旧标识下引入新类型。分析 ZIP 的 format_version 仍为 1，query contract 独立升级。

## 值与 schema

列仍按位置及独立 ID 标识，保留重复标题。每个 cell 带实际 type。NULL 不带 value；空字符串、空列表、空集合和空字典都有 value，不等同 NULL。

既有 string/date/integer/bool/decimal/amount/position/inventory 保持原表示。新增：

| 类型 | value 表示 |
| --- | --- |
| dictionary | 按 UTF-8 字节顺序排列的 `{key, value: Cell}` 数组 |
| list | 按原顺序排列的 Cell 数组 |
| set&lt;string&gt; | 排序、去重的字符串数组 |
| open、close、transaction | 有序键值数组，包含当前受支持的结构字段；字段值仍为 Cell |

采用键值数组是为了保留 Python 中可区分、而 Swift String 视为 Unicode 等价的不同字典键。读取时拒绝重复键、乱序、未知 cell 类型、缺少必填结构字段、错误 NULL 形状及非规范集合。

结构字段范围：open 的 meta/date/account/currencies/booking；close 的 meta/date/account；transaction 的 meta/date/flag/payee/narration/tags/links/accounts。此为当前实现的结构字段，**尚不代表所有 Transaction 原始内容已无损表达**，posting 子结构及相关访问/转换仍需后续封口。

上游 `other_accounts` 的声明 dtype 是 set，实际值为排序 list。本契约保留这一差异：列为 set&lt;string&gt; 时允许 list cell，不能把实际值静默改为 set。普通集合仍输出 set cell。

## 取数与源语义

标准表绑定在编译期完成，不靠第一行猜类型。已支持 9 个新增标准表及 postings 的多数标准字段；wildcard 根据上游静态列序展开，零行仍保留 schema。entries wildcard 所需的标准 id/hash 尚未实现，不能用自定义 hash 或略去该列替代。

Posting meta 与交易 meta 分别保留。交易系统 `__tolerances__` 使用 v3 核心在 booking/interpolation 前根据源 postings 推导的结果，包括空字典。公开案例验证 1.00 USD 的容差为 0.005。NULL metadata 的标准源关键字为 NULL；显式空 payee 与缺省 payee 分别返回空字符串和 NULL。源解析含义变化使会计 revision 更新为 v3-core-5。

Price cell 是上游 book 后的单位价格。总价 @@ 先除以绝对数量，按固定参考 Decimal context 保留精度；金融核心的精确权重不因显示字段改变。源导出的财务指纹忽略衍生的 Tolerances 字段，继续检查声明 metadata、全部 postings、prices、accounts 与中间库存，避免仅查询系统 metadata 的有无导致 legacy/v3 可导出的账务被误拒绝。

## 捕获及预算

查询结果与 CSV/分析 ZIP 使用相同不可变捕获。递归编码沿用执行取消和 step 预算，最大嵌套 32 层、单个集合/字典/列表 10000 项；字典键及集合成员最多 65536 字节。整体返回仍受 10000 行及 16 MiB 限制。CSV 对集合/结构输出规范 JSON；类型及 NULL 信息由伴随分析文件保存。

## 标准形式：DISTINCT、HAVING、派生表与 PIVOT BY

四种标准形式已按固定 oracle 的观测实现，期望值来自本仓库内的 [oracle 观测](../evidence/query-oracle-2026-10-12.json)（评审期原始捕获在 `fix/review-evidence-2026-10-12/query-comparison.json`，不随仓库发布）：

| 形式 | 语义 | 代码位置 |
| --- | --- | --- |
| `SELECT DISTINCT ...` | 按 typed 身份去重。相等 Decimal 的不同书写、重复 NULL 都折叠为一行；NULL 与空字符串、相同数值的 integer 与 decimal 保持不同行。去重在 projection 之后、ORDER BY/LIMIT 之前执行 | `distinctResult` |
| `GROUP BY ... HAVING <aggregate predicate>` | 只在 GROUP BY 之后解析（上游语法要求），过滤整个分组；必须包含聚合，不接受 projection 别名；非聚合 HAVING 报 `invalid-group`。HAVING 中的裸源列按上游的扫描上下文行（最后一条源记录）求值，是分组级判定，不是逐组过滤 | `applyHaving`/`havingContext` |
| `SELECT ... FROM (SELECT ... WHERE ...)` | 派生表先执行内层 select，外层读取其输出列名与声明类型；支持嵌套、别名、`SELECT *`、聚合与 DISTINCT，引用不存在的列报 `unknown-column`；`SELECT *` 只跳过自身字段校验，WHERE/GROUP BY/HAVING/ORDER BY 仍校验 | `executeTypedPlan`/`typedDerivedTable`/`prepareDerivedQuery` |
| `GROUP BY ... PIVOT BY <col>, <col>` | 第一个 pivot 列保留为行键，第二个展开为每值一列，其余 projection 列作为 measure 在每个 pivot 值下重复；缺失组合为 NULL，measure 列声明 nullable；行键与 pivot 值按 typed 顺序升序；全部投影列都是 pivot 键时只保留行键列 | `pivotTypedPlan` |

全部四条路径都经过 `evaluateRowsBounded` 的 step 预算与取消检查；PIVOT 的 reshape 与 DISTINCT 的去重也各自计数。PIVOT 的稠密矩阵在分配前受 cell 预算约束（`QueryPivotCellLimit`，可由 `ExecutionLimits.PivotCells` 收紧），pivot 后的列宽超过行上限时报 `result-budget-exceeded`，不返回被截断的“完整”结果。CLI typed-json/typed-csv 与 `desktop.Session` 的分析捕获使用同一条管道，因此 `SELECT DISTINCT` 的列、HAVING 的过滤、派生表的列类型以及 PIVOT 的 `row/column` 标题都以 typed schema 保存并可回读。

旧 `Parse`/`Evaluate` 入口同样接受这四种形式（HAVING/PIVOT 由同一 parser 提供），但投影列名仍以文本契约输出，不声明类型。

### 未封口的具体项

以下差异已定位，未在本批修复，不能读作“已等价”：

- `IN` 子查询与 FROM 期间修饰。
- `PIVOT BY` 的表命令形式（PRINT/JOURNAL/BALANCES 等）。
- HAVING 裸源列的上下文行依赖上游的循环残留值；本项目已按观测复现，但这不是有意的 SQL 语义，上游重构会改变它。

## 仍需完成

非有限 Decimal、完整 Any 的转换/运算、全部结构访问、标准 entry id/hash、完整函数/操作符重载、`IN` 子查询与FROM期间修饰、`PIVOT BY` 的表命令形式（PRINT/JOURNAL/BALANCES等）、命令和生态等仍未封口；源 metadata 保留字段的异常类型覆盖也仍需审计。因此此文档是已实现返回契约，不能作为完整 Beancount/BeanQuery 兼容声明。
