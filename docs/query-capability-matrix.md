# 查询能力与兼容矩阵

**当前实施基线（2026-10-01）**：Q02已实现有界查询执行、快照上下文、会话取消及真实C/Swift typed契约；完整工作台未完成。全量Go/七包race、51会计/147查询、原生三套/app及最新私人Swift/C通过（双方2833指令、零差异），见[本批证据](evidence/q02-bounded-execution-2026-10-01.md)。会计v3-core-3、默认legacy及查询contract保持；后台客户端与Store首批见[新增证据](evidence/q02-query-store-2026-10-01.md)；[QueryView/导航首批](evidence/q02-query-view-2026-10-01.md)已接入；错误位置与[保存query目录](evidence/q02-saved-query-catalogue-2026-10-01.md)已推进；下一步补preset、固定执行导出与完整验收。Q01剩余语义、X01/X02及GUI/安装/发行继续保留。

更新：2026-10-01。Oracle固定为Beancount 3.2.3 / BeanQuery 0.2.0，依赖身份来自 `tools/reference/uv.lock`。本文件是Q01/Q02实施契约；下列“oracle确认”只证明参考行为，**不表示Go已经实现**。前次16个探针见[历史文档复验](evidence/go-native-design-refresh-2026-10-01.md)，22个基础场景见[标量证据](evidence/q01-typed-query-foundation-2026-10-01.md)。当前11个公开语料集共147个strict场景通过，包含标量、复合值、metadata、转换及日期/聚合端点；见[Q01证据](evidence/q01-date-aggregate-price-2026-10-01.md)；完整Q01仍未完成。

标准签名以本地锁定版本的公开执行和[BeanQuery v0.2.0源码](https://github.com/beancount/beanquery/tree/v0.2.0)为准；[官方项目说明](https://github.com/beancount/beanquery)提供定位。在线主分支不替代锁定版本。

## 当前代码与最小交付范围

“代码核对”表示检查了当前实现，尚未建立新的严格query golden。已有query测试和旧parity保持为继承基础，不推导完整类型兼容。

| 能力/签名 | 固定oracle行为 | 当前Go与后续动作 | 任务 |
| --- | --- | --- | --- |
| `SELECT account, number` | 省略FROM读取postings；str、decimal | 新ParseStandard/EvaluateTyped及CLI typed-json已支持，golden通过；旧Parse仍要求FROM | Q01.1 |
| `abs(decimal)` | decimal，-2返回2 | typed已支持；decimal重载通过；固定版无integer重载，现typed在扫描前拒绝，LIMIT0反例通过 | Q01.1 |
| `length(str)` | int；本例账户长度11/13 | typed已支持；账户长度与餐😀的两字符golden通过 | Q01.1 |
| `filename`、`lineno` | str、int列；posting原始行5/7 | typed列已支持；普通posting行5/7；pad继承balance第4行另验，导航At仍保留pad行 | Q01.1 |
| `filename()`、`lineno()` | 编译错误，不是零参函数 | typed明确拒绝，unknown-function与固定oracle错误通过 | Q01.1 |
| `meta(str)` | object；读取posting，缺键NULL | 用户标量/Amount、系统filename/lineno及pad有效来源有strict场景；list/map未支持 | Q01.1 |
| `entry_meta(str)` | object；读取所属交易 | 用户交易标量/Amount、系统来源和有效stack已验；普通交易__tolerances__字典明确unsupported | Q01.1 |
| `any_meta(str)` | object；posting优先，缺键读取交易 | posting优先，仅缺键fallback；显式NULL、空字符串、false均停止fallback，strict通过 | Q01.1 |
| pushmeta/popmeta/include | 按词法顺序，文件独立stack，交易显式值优先；非法stack为ParserError | v3求值副本已实现；嵌套/Amount/NULL/日期乱序/单次include及非法stack已验；重复include等未验 | Q01.1 |
| pad有效metadata | entry来自pad，posting来自balance；系统来源scope分别保留 | v3已实现，两类golden通过；导航Span不变 | Q01.1 |
| 系统 `__tolerances__` | 普通entry为字典，posting/pad缺失 | entry/any普通字典明确unsupported；posting/pad返回NULL；primary probe与接口负例已验，未声称字典结果等价 | Q01.1 |
| `position` | position列，保留数量和可选成本 | typed从已booked数据构造；完整日期/label、无成本/零成本和FIFO消费strict通过 | Q01.2 |
| `units(position)` | amount，含币种 | typed标准重载已通过；旧units Decimal列保持原适用范围 | Q01.2 |
| `cost(position)` | amount；无成本时保留原units | typed标准重载已通过，零成本不视为缺成本；旧cost文本列保留 | Q01.2 |
| `weight` | amount列 | typed保留报价币种及源表示；直接/总价/负数量、零数量/报价、高精度和非终止单价strict通过；旧Decimal列保留 | Q01.2 |
| `weight(position)` | 编译错误，锁定版无该重载 | 明确拒绝，不能为满足函数列表而虚构标准 | Q01.2 |
| `number(amount)`、`currency(amount)` | decimal、str | typed实现并有strict正例及空输入非法类型反例 | Q01.2 |
| `sum(position)` | inventory；本例相抵后空库存 | typed完整键、多币种、相抵空库存、分组和零行strict通过，Span不进入身份 | Q01.2 |
| `sum(amount)` | inventory，无成本position | typed实现；units/cost/weight聚合strict通过 | Q01.2 |
| `units(inventory)`、`cost(inventory)` | inventory；本例保留空库存 | typed已实现，全部币种归约、110 USD成本与空数组有strict证据 | Q01.2 |
| 复合类型直接比较 | amount/position/inventory比较均编译失败 | typed在扫描前type-mismatch，避免格式化字符串伪比较；三类strict反例通过 | Q01.3 |
| 复合类型ORDER BY | Amount按币种/数值；Position与Inventory有参考稳定次序 | typed三类strict通过；库存源插入次序与canonical输出次序分别保留，特殊tie反例通过 | Q01.3 |
| `units`、`cost`裸列 | pinned postings表报未知列 | 当前是旧扩展；新标准入口与旧Adapter明确区分 | Q01.2 |
| `SELECT 1 AS v, 2 AS v` | 同名两列，各保留值1、2；int | typed ordinal ID和位置cell已实现；旧map明确拒绝重复；真实CLI及golden通过 | Q01.2 |
| `LIMIT 0` | 成功零行，列类型仍存在 | typed已支持并保留零行schema，golden通过；旧入口保持兼容 | Q01.3 |
| `number / 0` | 成功NULL，列类型decimal | typed nullable decimal/NULL已实现，golden通过；旧入口除零错误保留 | Q01.3 |
| 空集`count(*)`/`sum(number)` | 本例成功零行，不生成零值行 | typed返回空行及int/decimal schema；golden通过，不补零值行 | Q01.3 |
| `abs(str)`，即使WHERE无行 | 编译错误 | typed在扫描前报type-mismatch；非法WHERE函数/regex类型也有空结果反例 | Q01.3 |
| `avg(decimal)` | 锁定版本报编译错误 | 旧入口扩展保留；typed明确拒绝，golden通过；不称标准avg支持 | Q01.3 |
| 动态object运算 | 两object无比较签名；一侧有静态类型时转换另一侧；int目标提升Decimal | string/finite Decimal/date隐式转换及NULL结果、metadata精度文本已验；显式cast函数另列规划 | Q01.3 |
| unary/括号 | 正号省略于标题；负号仅numeric；NOT接受Any并低于比较优先级 | ±、NOT、括号标题与编译期非法object负号已验 | Q01.3 |
| NULL/布尔字面量与IS NULL | NULL独立声明类型，IS [NOT] NULL返回bool；NULL/bool直接等式无签名 | 字面量、判定、LIMIT0编译拒绝已验；不套用其他SQL规则 | Q01.3 |
| AND/OR与WHERE真值 | Any truth，左侧顺序短路；NULL AND FALSE为NULL，FALSE AND NULL为false | 核实组合与编译不短路、动态Amount/metadata WHERE、库存真值明确unsupported | Q01.3 |
| 动态GROUP/ORDER | 数字不同源精度合组；str与Decimal混排参考抛TypeError | canonical财务值与文本转换分开，group及排序错误strict通过 | Q01.3 |
| GROUP/ORDER/alias/日期其余 | 尚未完整对照 | 已有基础及本批动态/NULL范围；全部重载/alias和日期函数仍需逐格验收 | Q01.3 |
| 显式`str/decimal/date/bool/int` | 固定cast语料含NULL、失败转换、Unicode数字、日期与整数截断 | 已验标量/metadata重载通过；casts34场景（含日期提取）及compound-casts6场景；已验复合str通过，完整Any重载另验 | Q01.3 |
| `str`与Decimal算术组合 | `str(1.00+2)`为`3.00`；负零、指数及metadata转换各有规则 | 已修复；组合Decimal/算术、负零及科学记数法strict通过 | Q01.3 |
| typed标量Decimal上下文 | 默认28位有效数字half-even；unary/abs及每项SUM按context计算 | 加减乘除、两类tie/carry、非终止商、NULL和SUM源序strict通过；复合及极端指数/陷阱另验 | Q01.3 |
| `date(integer/bool,integer/bool,integer/bool)` | `date(2000,TRUE,1)`为2000-01-01 | 已实现并strict通过；每位置TRUE/FALSE、nullable integer、无签名NULL literal和C整数越界分别验 | Q01.3 |
| `str(position/amount/inventory)` | 完整成本、原始标签、空库存及稳定Position次序 | 已验普通/精度、拆分FIFO、总成本、空/零库存strict通过；报价及表示长尾另验 | Q01.3 |
| amount→number及成本/库存上下文 | 源表示、有限Decimal成本乘法与逐项库存累加 | 已验2.00/10.00→20.0000、高精度cost/weight与原position不舍入；AVERAGE精确扩展分开 | Q01.2/3 |
| 普通/聚合自动分组 | 没有显式group时按普通投影分组 | mixed str(position)/sum与account/sum/count strict通过；显式group合法性仍需完整核对 | Q01.3 |
| `first/last/min/max(Any)` | FIRST跳过前导NULL，LAST包含末尾NULL，极值跳NULL、同值保留首次 | 已验数值/日期/string/integer/bool/Amount/Position/metadata及NULL、分组/零行/非法嵌套；Inventory与动态混合长尾未全验 | Q01.3 |
| `year/month/day(date)` | integer；运行时nullable date传播NULL，NULL literal/object无签名 | 已验边界、组合聚合、NULL、wrong arity与零行非法类型；其他日期函数不由此推断 | Q01.3 |
| `balance` | 本次未执行验证 | 当前每账户字符串；锁定源表使用Inventory，执行/过滤时序须另验 | Q01.2/3 |

## 类型与协议

列结构保留ordinal生成的ID、标题、声明类型和nullable；每行是按列位置排列的cell数组。integer、decimal、amount、position、inventory、date、bool、string、null分别编码。金额复用PresentedDecimal；整数使用十进制字符串。object列声明动态类型，单元格按实际值带type；零行不能靠第一行推断列类型。

meta返回的object首版支持可从账本值域无损映射的标量与金额。对list/map/其他对象，在新增规范化结构和往返测试前明确拒绝，不能调用fmt转为任意字符串。标签/链接集合规范化排序，不把它们当具有源顺序的数组；具体支持签名仍需golden。

标准typed入口和既有CLI/Web输出之间设明确Adapter。保留旧行为的调用方不无声改变金额类型；旧标题map遇到重复标题必须报无法表示或采用公开的新格式，不丢弃值。会计semantics与query contract分别标识，不能推断v3 profile必然启用新query schema。Go入口为EvaluateTyped/ParseStandard，CLI显式typed-json已可执行；查询执行C/Swift协议已实现，原生工作台与导出尚未接入。

## 每一行何时可以标为已实现

每个重载必须有独立公开输入、锁定oracle期望、Go通过记录及负例。记录列声明类型、NULL、位置值、行重数、排序与错误类别；类型对象用Column.name/datatype读取，不把DB-API的type_code当类型名称。查询gate只读已审阅golden，不自动重写期望。

先验parser和编译，再验扫描、分组、投影、排序及LIMIT；空输入仍执行签名校验。不要用所有值的显示文本组成分组键，不吞掉表达式错误。查询函数范围与内置报表能力分开，报表已有市值投影不证明BQL的convert/value已实现。

## 明确不纳入M4

JOIN、CTE、window、任意Python函数和Python对象执行不纳入本轮。所有未列入已验重载的函数均拒绝或标为规划；convert/value、复杂集合重载和完整SQL能力不因Go报表存在而自动加入承诺。

## 当前实际入口与下一批

```sh
make query-parity
make query-oracle-check
go run ./cmd/orangecount query --format typed-json --semantics v3 \
  testdata/fixtures/query/default-source.bean \
  'SELECT account, sum(number) AS total GROUP BY account ORDER BY total DESC'
```

当前typed来源仅postings；entries/accounts/prices通过旧入口继续可用，新入口给出unsupported-capability。新JSON带contract/columns/rows；Q02执行入口已提供快照上下文与资源预算，原生结果页面首批已接入，GUI验收与导出未完成。integer除法产生的float不在当前已声明cell类型内，明确拒绝；财务decimal保持精确值，非终止算术与oracle精度差异仍须独立登记/验收。

本批报价表示、已验日期/聚合签名通过；下一批补分组/alias/ordinal、balance/sum(inventory)、可执行预设及其他表示/数值长尾。重复include/metadata、保留键覆盖及同日跨文件排序另核实；见实施手册第39节。不能据147场景或existing native页面检查关闭Q01/Q02。typed标量算术/SUM已有28位上下文，复合成本/库存的已验范围已通过，其他表示路径及极端指数仍待独立确定，不能将该范围扩成全局数值豁免。

当前golden为schema 2，sources_sha256绑定完整捕获源图；十一个注册corpus均必需。子文件仅增加注释也会触发身份失败。类型化结果contract仍是v1，与会计v3-core-3及原生协议2分别记录。

非有限Decimal转换（NaN/Infinity等）目前明确unsupported-capability，不能伪装成NULL。固定reference可产生非有限Decimal，本批接口负例登记能力边界；有限Decimal仍保留exact值，输入数字可包含Unicode十进制数字/指数，下划线按固定转换规则处理，逗号/分数文本无效。已有标量Decimal运算上下文；已验有限复合上下文通过；没有全部表示路径、极端指数上下文或全类型转换兼容声明。

## Q02当前执行接口

ExecuteTyped与Session.ExecuteQuery/CancelQuery、OC_ExecuteQuery/OC_CancelQuery及BridgeClient已实现；资源边界、真实失败与C/Swift解码见[证据](evidence/q02-bounded-execution-2026-10-01.md)。后台客户端及Store首批已接入，见[Store证据](evidence/q02-query-store-2026-10-01.md)；QueryView及导航首批已接入，导出与GUI验收未完成；range字段required-nullable；语法/函数/字段首批已提供真实token位置，见[证据](evidence/q02-query-locations-2026-10-01.md)，其余位置链与GUI未完整验收。此前“尚未提供预算”的描述属于首批前状态，不作为当前源码判断。

### 2026-10-02 当前补充

首批四预设（按月收入、按月支出、账户库存余额、分录来源）已通过固定 oracle 与 CLI，并接入 native 草稿选择；12 corpus / 151 查询全部通过。它们读取完整快照，收支保留记账符号和多币种库存，不含隐式日期/换汇。查询 CSV/分析保存及独立容器读取已落地；最新范围与未验项见[证据](evidence/q02-analysis-grid-presets-2026-10-02.md)。前文“预设尚未实现”保留历史背景，不代表当前这四项状态；其他查询长尾与完整验收仍待完成。

期间预设已支持四个模板的合法双边界及单边界：选择只把具体日期写入草稿，最近一月由捕获快照解析。当前 12 corpus / 163 查询通过；见[证据](evidence/q02-period-presets-2026-10-02.md)。完整 Q01/Q02 与期间菜单 GUI 未关闭。
