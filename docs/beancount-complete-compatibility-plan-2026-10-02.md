# 无插件 Beancount 完整兼容：设计与实施计划

本文件落实2026-10-02用户的新要求：不保留插件机制；源加载、查询生态和完整兼容验收必须完成。优先级高于此前F01–F04的限定范围和ADR-0046的查询子集限制。0.1.5保持已验开发基线，本文件不代表新能力已经实现。

## 1. 产品契约

目标是“无插件 Beancount v3 标准兼容引擎，加OrangeCount扩展”，不是未经限定的“兼容全部Python生态”。固定开发参考Beancount3.2.3、BeanQuery0.2.0；升级上游必须重新审计和差分。用户账本保持只读，Python只作开发oracle；产品不加载Python模块，不引入插件注册/发现/执行机制。

插件排除只允许排除用户plugin声明、第三方模块和其新增语义；不能把上游默认booking、插值、pad、验证等内部流水线工作也视为可删除插件。遇到plugin声明须有来源定位的明确诊断，禁止忽略后继续展示“有效账本”。保留识别该语法以报错，不等于保留插件机制。

查询目标包括上游BQL语法、内置Beancount数据表、列、函数及全部重载、元数据、类型/NULL、排序与聚合、内置账本查询命令，以及CLI/原生/Fava账本查询接入的一致行为。不是通用SQL，也不能将上游实际支持的SQL式构造拒绝为“不属于BQL”。第三方Python函数与外部数据库驱动不在无插件目标内；CSV等上游内置数据源在审计中单独登记，不能未经审计宣称不需要。

默认legacy不是标准兼容验收配置。首先在显式v3证明语义，之后设计新账本默认v3与已有账本显式配置迁移；未完成迁移证据前不得悄悄改变已有账本结果。

## 2. 当前事实与需要修订的旧结论

| 事实 | 本次要求下的处理 |
| --- | --- |
| 51会计/215查询、FinanceBook2836 entries零差异 | 保留回归基线，不算完整兼容证明 |
| LoadGraph只递归单个路径，不展开glob | 必须补全普通/递归glob及其状态监测 |
| 上游loader以队列处理源，并使用glob.glob(recursive=True)；匹配结果未在该处显式排序 | 必须独立调查同日顺序，不可直接用Go DFS+sort冒充上游顺序 |
| 重复include和posting保留metadata错误未对齐 | 纳入标准加载/校验，不再作为最终验收排除 |
| 新typed查询仅postings，其他表使用旧入口 | 所有标准表统一进入typed执行、导出与原生解码 |
| ADR-0046说上游无子查询并排除BQL PIVOT | 本地固定上游含SubqueryTable、EvalConstantSubquery1D、EvalPivot和AST.PivotBy；旧范围判断失效 |
| 字典/集合、非有限Decimal和部分动态重载明确unsupported | 新目标下是待实现缺口，不能用unsupported标签关闭任务 |
| export保守拒绝外部include/glob等 | 源加载兼容与源导出事务是两项能力；先补加载，再独立修订导出映射，不能一并放开写入 |

调查来源为tools/reference锁定安装的beancount/loader.py、beanquery/parser/ast.py、query_compile.py及官方测试源码。这里记录的是本轮阅读事实，尚未产生新的差分通过证据。

## 3. Module设计

### 3.1 Source capture

保留source.LoadGraph这一公开Interface，内部拥有include匹配、文件身份、加载顺序与诊断。Graph同时记录原include模式、匹配路径及匹配顺序、实际文件身份、未匹配模式与搜索目录证据；FileID是观察身份，不作为会计/查询标准排序依据。

递归glob必须审计`*`、`?`、`[]`、`**`、隐藏文件、字符转义、零匹配、目录匹配、symlink目录、循环、重复命中、绝对/相对模式和大小写。Pythonglob与filepath.Glob语义不同，不直接替换即可宣告兼容。目录扫描必须有取消/工作预算；有限正常输入与oracle一致，越限必须明确资源错误，不返回伪完整结果。

加载次序、metadata堆栈和同日entry排序分别建模：加载队列不直接等同于最终entry排序。重复文件不能重复求值；是否产生错误及其定位/类别由固定oracle决定。文件字节、include结果和来源身份一起参与输入指纹。

ChangeDetector使用文件身份/属性及glob搜索目录与匹配集合证据，覆盖新增匹配文件、新增递归目录、删除/移动、原零匹配变为命中以及构建期间变化。只监测已加载文件不能通过验收。原生重载、失败后保留旧快照、保护路径及私人同源capture统一使用此Module。

### 3.2 Query execution

沿用现有query的解析/编译/有界执行Interface；新增标准来源Adapter和必要AST节点，不维护第二套标准查询语义。先生成固定上游语法/表/列/重载清单，再分语义切片实施。postings、entries、accounts、prices以及审计发现的其他内置账本表具有独立schema，零行也完整声明类型。

标准复合值新增字典、集合、列表及动态联合类型的可读写契约；系统__tolerances__属于标准值，不能丢弃。非有限Decimal用明确tag和文本表示，不写JSON NaN，不伪装NULL。严格区分NULL、空集合、零与动态类型错误。旧typed-v1/v2冻结分析文件继续读取；改变结果或增加不可被旧reader理解的类型时升级查询契约并记录迁移。

子查询、IN子查询、HAVING、DISTINCT、PIVOT BY、标准FROM期间修饰、PRINT/JOURNAL/BALANCES等命令逐项以实际上游清单为准；出现CREATE/INSERT等语句须审计其影响范围和账本调用可达性，不能凭名称认作任意SQL或源账本编辑。

### 3.3 Compatibility evidence

建立机器可读兼容登记表：每项有upstream_symbol、版本/源码身份、输入类别、输出类型、NULL与错误规则、顺序约束、Go实现位置、独立案例ID、状态。状态仅pending/implemented/verified/blocked/excluded；excluded只准用户明确排除的插件/外部扩展，必须有原因，不能收缩标准范围制造全绿。

参考工具与Go读同一冻结输入，比较有效性、诊断类别/位置、ordered entries与metadata、每笔交易后的全部账户完整库存、最终余额、价格、期间收支，以及查询schema/标题/ordinal/type/nullable/每格值/行重数/顺序/错误/CSV与分析往返。精确数值不以display或浮点比较。

## 4. 开发顺序与可审查交付

| ID | 工作 | 主要文件 | 退出门槛 |
| --- | --- | --- | --- |
| C00 | 固定上游能力清单与差距登记，修订错误ADR | tools/reference、docs、compat登记表 | 语法、内置表/列/函数/错误规则逐项有上游来源；无未登记标准能力 |
| S01 | glob/递归glob正常加载及错误诊断 | internal/source、diagnostic、公开fixtures | 每类模式有正负oracle、相对/绝对路径与非ASCII案例；加载顺序已调查 |
| S02 | include重复/循环/同日顺序/metadata来源 | source、ledger、snapshot | ordered entries/错误/完整成本中间库存差分；旧51/215保持 |
| S03 | glob新增/删除/构建竞争与监测 | source、snapshot、desktop、Swift/C测试 | 原零匹配恢复、递归新增目录与匹配文件触发重载；失败不发布，无漏保护 |
| Q10 | 全部标准账本表与typed schema | query、desktop、Models、CLI | 原生/CLI所有表零行及实际行逐格对齐；旧冻结文件可读 |
| Q11 | 字典/集合/动态类型/非有限Decimal | query、analysis reader/writer、Swift解码 | 标准类型及系统metadata正负例，预算/取消，CSV/分析无损往返 |
| Q12 | 函数/运算符/聚合完整重载 | query、兼容登记表 | 上游注册签名全部覆盖，非法签名即使零行也拒绝；标题/精度/NULL一致 |
| Q13 | 子查询/HAVING/DISTINCT/PIVOT及剩余语法 | query parser/compiler/execution | 官方能力清单全部正负/组合差分；不以独立报表替代标准BQL |
| Q14 | 命令、保存query、CLI/原生/Fava账本查询集成 | CLI、desktop、Swift、现有web adapter | 标准query声明及命令实际执行，取消/错误定位/分页与导出一贯 |
| V01 | 上游回归、生成式/变形/模糊检查 | compat、reference、公开fixtures | 标准项全部verified；无未解释差异；新反例固定入库 |
| V02 | 私人账本同源复验与最终兼容审计 | 已有只读harness、证据文档 | FinanceBook全量/期间/查询/Swift-C通过；不以单账本替代V01 |
| M01 | v3默认与已有配置迁移 | snapshot/desktop/Swift偏好、文档 | 已有用户配置保留；新账本行为明确；legacy隔离与迁移实际验收 |

每项按“公开失败例→最小实现→独立oracle→相关回归→提交”交付，不能积累数个功能只有最终一次可用结果。先C00，随后S01→S02→S03；查询按Q10→Q11→Q12→Q13→Q14；V01贯穿各切片，V02/M01在标准兼容封口前。没有证据的时间估计不写成完成日期。

源语义改动需评估会计revision，查询变化需评估query contract；两者不能相互替代。每次共享求值/源序变化执行51会计、215查询、Go全量、相关race、真实Swift/C；原生编译串行。私人账本只读，不用于生成公开golden。S系列完成后再单独开放glob源导出及其冻结/目标保护，不弱化0.1.5写入约束。

## 5. “完整证明”的发布门槛

这里的完整证明指可审计的固定版本标准兼容验收，不是有限测试对无限输入的数学完备证明。用户要求不能降级为“私人账本能打开”或“多加若干案例”。

1. 固定版无插件标准能力登记全部verified，pending/implemented/未解释blocked为0；排除项只限明确插件/外部扩展。
2. 官方标准测试逐项审计并映射到Go公开Interface；可适用用例全部对照，未移植项须证明仅属排除的插件机制或上游自身非契约内部结构。使用官方素材先核对许可证，不复制Python产品实现。
3. 独立差分包含有效/无效、零输入/重复输入/缺失、顺序与类型组合；有生成式case与可重现seed，并把每个差异变成固定公开回归。
4. 完整交易前缀库存与精确成本/价格/收支对齐，不只看最终余额；所有查询类型/声明/值/错误及捕获reader往返通过。
5. FinanceBook同源只读与真实Swift/C/原生关键查询流程通过；资源越限、取消与失败快照语义可区分，不返回无标记截断结果。
6. 独立核对登记表与实际代码，没有把旧子集排除项遗漏到登记表之外；记录源码commit、上游版本、工具链、输入身份与产物身份。

通过这些门槛后，才能发布“固定版本、无插件标准范围的Beancount兼容＋扩展”结论。完整N04/R01、跨系统与正式签名仍是产品发行门槛，不能与会计/查询兼容证明混为一谈。
