# 原生缺少功能详细设计

状态：设计提案，尚未实施。基线为源码 `278c6e4`、验收文档 `3e2a56a` 和开发包0.1.1。本文细化现有24任务中的 Q01/Q02/X02，不增加完整 SQL、Python 插件执行或账本编辑承诺。执行顺序见[开发计划](native-missing-features-plan-2026-10-02.md)。

## 1. 范围与当前事实

| 功能包 | 当前实现 | 本次补齐目标 |
| --- | --- | --- |
| F01 报表分析导出 | 查询 CSV、四成员 `.ocanalysis` 容器、独立查询 reader、原生原子保存已实现 | 固定捕获报表/分页及估值状态，生成、读取、原生保存贯通 |
| F02 附件目录配置 | 默认入口账本目录；附件五状态、打开及 Finder 已实现 | 显式额外目录、按账本保存、权限撤销、配置变更后的引用失效 |
| F03 BQL 已承诺长尾 | typed postings、预设、取消及预算已实现；12语料/163场景通过 | balance、sum(inventory)、分组/别名和已列重载长尾逐项对照或明确拒绝 |
| F04 会计源导出约束 | CLI export/dialectize 与公开往返测试存在 | 输出前兼容检查、AVERAGE 等修订语义拒绝、路径保护及文件树事务保存 |

全页面 GUI、键盘/VoiceOver、帧率/内存、安装循环属于后续验收；远端 CI、目标旧系统、Developer ID/公证属于发行，均不作为“缺少功能”的实现任务。各功能本身必要的真实原生操作仍是本次退出条件。

本次源码核对发现：CLI `writeConvertedFiles` 使用直接写文件，目录输出逐文件写入；不能由现有分析文件原子保存推断源导出也具有同样保护。查询矩阵的旧段落混有历史状态，不能直接当作未实现清单。

## 2. 共同约束

- 默认 legacy 不变；会计 profile/revision、查询 contract、分析格式版本分别标识。显示偏好和附件授权不修改会计快照。
- Go 产生精确结果与文件字节；Swift 管理用户授权、面板和文件保存。不得用格式化文字或 Double 重建金额。
- FinanceBook 始终只读；公开输入用于故障、覆盖、权限和文件替换测试。私人证据仅聚合。
- 新增接口是本文拟议 Interface，实施时同步 Go、C header、Swift Models/Client 和真实契约测试；不能在单侧偷加字段。新增required的投影/附件策略字段时，将原生协议2升级为3并让旧客户端明确报版本不匹配；查询typed contract与查询容器v1保持各自版本。
- 遵循现有16MiB结果上限。新预算失败必须有明确资源错误，不生成貌似完整的截断文件。
- 未完成捕获、取消、写失败、被拒绝的目标都不能显示“保存成功”。

## 3. F01：报表分析导出

### 3.1 用户流程

支持资产负债表、损益、趋势、账户详情、持仓、价格、交易日记账的已交付投影。诊断、迁移、全指令/配置、附件文件内容不在报表导出范围；查询保留原入口。

每个支持页面增加“导出”菜单，包含“保存分析文件”和“保存 CSV”。分页页面默认明确写“当前页”；完整结果入口仅在该投影已经实现单快照完整捕获时出现，不能拿分页循环充当完整结果。持仓批次/归约、价格历史、账户流水分别以当前子视图为单位，不合并未打开的其他子视图。

点击后先捕获成功响应和归一请求，再打开 NSSavePanel。面板展示数据截至日/期间、会计语义、估值币种、捕获范围和缺价/截断提示。用户更改过滤不改已捕获字节；同账本重载后仍可保存旧捕获，但须标明旧快照并刷新源/附件保护。切换账本或关闭窗口取消尚未提交的保存。成功只在原子写完成后显示。

### 3.2 Module 与 Interface

- 拟新增 `internal/analysis`：分析文档校验、四成员容器读写、预算、CSV 一致性；不知道 Session 或文件路径。
- `internal/desktop/report_export.go`：固定 publication 捕获、投影 Adapter、保存保护路径。复用现有投影计算，不复制会计实现。
- 查询 v1 reader/writer 首先保持行为；提取共享 ZIP 处理时以已有恶意 ZIP/类型/CSV案例锁定行为，不重写查询执行。
- 拟新增 C `OC_CaptureReportExport`；Swift `ReportExportClient` 和现有 `AnalysisFileExport` 是原生 Seam。是否新增独立文件以维护局部性决定，禁止在 View 内拼接 ZIP。

拟议请求：

```json
{
  "snapshot_id": "opaque",
  "projection_id": "opaque",
  "format": "analysis",
  "capture_scope": "current-page"
}
```

`projection_id` 绑定 publication、归一请求、有效响应及分页，是不可猜测为任意文件路径的投影身份。Session 只保留最近16个可导出投影，总保留字节预算16MiB；超限淘汰最旧身份，捕获已淘汰身份返回 `expired-capture`，原生重新读取后重试。每个普通 Read 返回其可导出身份；查询不占用该缓存。

拟议响应：`context`、`format`、`content`（沿用现有 Data/base64桥接方式）、`protected_paths`。捕获时 publication 已改变返回 `stale-snapshot`；捕获后字节与会计结果不可变。保存保护独立调用现有 ExportProtection 刷新。

完整捕获由 Session 固定一次 publication 后在 Go 内执行一个完整投影，取消/预算覆盖遍历、排序和序列化。后续重载不会让处理中途改用新 publication。首批只开放非分页资产负债表、损益、趋势的 `complete-projection`；其他页面先支持 `current-page`。

### 3.3 分析格式

查询维持 `format_version=1`、`beanquery-0.2.0-typed-v1`；报表采用 `format_version=2`、`native-report-analysis-v1`，仍只有 context.json、schema.json、cells.ndjson、data.csv 四个成员。旧 reader 必须拒绝版本2；新入口按版本/contract分派。不得悄悄将查询 v1 改成另一种值域。

context 必填：producer、view、section、snapshot_id、semantics_profile、evaluation_revision、normalized_request、projection_id、capture_scope、returned_rows、truncated、completeness、page（无分页为null）、valuation_status（不适用为null）。normalized_request 去除入口绝对路径、附件路径、locale 等仅展示字段，期间用已解析真实日期，不保留相对日期表达式。

completeness 是捕获范围内的估值完整性（complete/partial/unavailable/empty/not-applicable）；truncated 和 page 独立表示捕获覆盖。`complete-projection` 可以估值 partial，`current-page` 也可以每行估值 complete，两者不可混为一个状态。

page 结构统一为 total_count、offset、limit、returned_count、has_more。同一行单位下禁止出现 returned_count≠returned_rows；交易展平采用下文明确的双单位校验。禁止 has_more 与计数矛盾、负数或 current-page 被标成完整结果。非分页导出不凭空制造 page。

schema 保留 ordinal column ID、标题、type、nullable。复用现有 integer/date/bool/string/decimal/amount/position/inventory/null 值；报表增加严格的 `record` 类型，仅用于固定结构的状态/归约详情，schema 必须声明 fields（ID、type、nullable），cell 为按 fields 顺序的 typed cells。record深度上限8，不支持任意对象执行或未知字段。动态币种集合优先用inventory，不能以显示文字代替结构。CSV 对 record/compound 使用规范 JSON；NULL 与空字符串的区别由 cells 保证。

估值状态记录缺失项目的账户、商品、原因、截至日及选中价格的币种/日期/exact。文件不带源绝对路径、附件副本或源账本。来源导航只保留快照内 opaque ID 与行列；离线文件无法访问原始来源时明确提示，不能把 ID 转成路径。账户名、交易叙述等仍是用户财务内容，导出说明须表达这是用户选择保存的数据。

### 3.4 各投影的固定映射

| 页面/section | 一行的含义 | 必须保存的列/事实 |
| --- | --- | --- |
| 资产负债表/损益 | 一个账户树节点 | account、parent、depth、explicit、direct_inventory、subtree_inventory、两类valued_portion/complete_total及valuation状态；原有符号不做展示反转 |
| 趋势净资产 | 一个日期/币种点 | date、currency、exact amount、可用性/缺价；不导出图表Double |
| 趋势现金流 | 一个月/币种 | income、expenses、net exact amount及可用性；保留真实零 |
| 账户概览/历史 | 一个汇总指标或日期点 | section、account、scope、cutoff/period、exact库存或金额、valuation；账户流水用下一行规则 |
| 交易日记账/账户流水 | 一条posting | transaction_id、posting_index、date、flag、payee、narration、tags/links规范结构、account、position、price、source_id；交易分页page在context，returned_rows按posting数另设 source_record_count |
| 持仓当前批次 | 一个成本批次 | account、commodity、position（完整成本日期/nil或空label）、value、quotes、unknown原因、cutoff；不把未知收益写0 |
| 持仓历史/归约详情 | 一个检查点或归约事件 | section、稳定事件/贡献ID、日期、exact units/cost/proceeds/gain、来源ID、整池贡献与实际消费分别记录；record字段按section固定 |
| 价格目录/历史 | 一个商品或一个源价格 | section、base、quote、date、exact quote、direct/inverse、可用性、source_id；同日不同记录不去重 |

tags/links使用固定record内的编号字符串字段会导致不稳定schema，因此实施阶段为其增加专用 `set<string>` 类型：数组去重排序，NULL与空集合不同，CSV为JSON数组。此类型只在报表v2开放；查询集合支持另行登记，不能顺带声称已实现。

交易导出分页明确区分源记录数与扁平结果行数：page.returned_count对应交易数，context.returned_rows对应posting行数；因此交易section不得套用一般returned_count等式，而使用schema声明 `row_unit=posting`、`page_unit=transaction`，校验 source_record_count 与交易ID重数。其他section row/page单位相同。

### 3.5 校验与资源限制

限制压缩文件16MiB、单成员解压32MiB、总解压64MiB；最多4成员，拒绝重复/未知名字、目录、symlink、坏CRC、非UTF-8、额外JSON值、未知版本与type。最多10000输出行/4096列、record深度8、单字段64KiB；超限整次失败。ZIP预算在写入过程中计数，不能靠高度压缩绕过解压预算。

reader恢复具体类型并重建canonical CSV逐字节对照。校验exact分数、nullable、估值总额约束、page单位、snapshot/profile身份，零行保留schema/context。损坏文件返回可定位错误，不自动执行账本或查询。

### 3.6 错误与恢复合同

| 错误 | 触发与恢复 |
| --- | --- |
| invalid-request | 未知format/view/scope或schema矛盾；停止捕获，修正请求 |
| stale-snapshot | 捕获请求的快照已不是当前快照；重读当前页面，不自动拿新数据替换用户旧选择 |
| expired-capture | projection已被缓存淘汰；提示重新读取并捕获 |
| resource-limit | 行、字段、结构、ZIP或运算预算超限；整次失败，允许用户缩小范围 |
| cancelled / closed | 用户取消或窗口关闭；不显示保存成功，清理临时产物 |
| protected-target / filesystem | 源/附件覆盖或写入失败；保留旧文件，允许重新选择目标 |

错误code跨Go/C/Swift稳定，说明文本中英本地化。独立reader对损坏容器给出成员/字段位置，不把文件内容整段放进错误日志。

### 3.7 验收

公开样例覆盖空报表、多币种、0、NULL、nil/空label、非终止精确分数、缺一部分/全部价格、当前页和完整投影。源响应→文档→ZIP→reader逐字段一致；CSV独立解析核对。真实C/Swift验证格式、身份和预算；GUI完成保存/取消/覆盖/重载/切账本，并实际用reader读出用户保存的文件。

## 4. F02：附件目录配置

### 4.1 用户体验与持久化

附件入口增加“允许的附件目录…”。入口账本目录作为只读默认项；用户通过系统目录面板添加额外目录，可删除额外项。显示目录、可访问状态和“需重新授权”；禁止自动授权整个Home、include目录或父目录。显式选择Home等大目录时显示覆盖范围确认，授权本身不改变include读取规则。

按规范入口路径区分账本配置，同一账本多窗口共享已保存目录；更改广播到其他窗口使旧附件引用失效。现有会计/阅读偏好保持兼容。用Foundation bookmark保存目录身份（沙盒构建需security-scoped授权；当前开发包不假装已开启沙盒），不能只存一个路径并在之后悄悄授权该路径上的新目录。

重启解析bookmark，stale或目录身份改变标记需重新授权，不自动扩大根。目录暂时不可访问时保留配置项但不把它传给Go；默认根仍可使用。显式移除最后额外目录不会影响默认根。

### 4.2 Session配置及引用身份

拟议 `ConfigureDocumentRoots(expected_revision, roots)` 和C对应入口；roots只有原生授权解析得到的绝对目录（最多32个额外根），Go再canonicalize、验证目录身份、去重，不信任Swift字符串即可授权。更新失败保留旧配置，返回invalid-roots或stale-policy。

附件策略属于Session，并从publication会计数据中分离；以 `policy_revision` 单调编号管理。账本重载不恢复已经撤销的根。DTO增加 `policy_revision`；动作请求同时带snapshot_id、document_id、policy_revision，旧快照或旧策略都拒绝。

更新在document锁内原子替换Resolver并清空引用缓存；publication读附件时获取策略快照，整个页面使用同一revision。既有 `document_id=record:index` 不能跨策略单独授权，三元身份为(snapshot,policy,id)。请求已经通过核验但系统打开尚未提交时，Swift再次验证window generation和policy_revision；策略变更取消未交接动作。系统已收到打开请求后无法承诺撤回已启动应用。

权限恢复/文件替换通过“刷新附件”清空观察引用并更新policy revision，不重算会计快照；不用要求用户改变账本内容。拒绝失效动作后显示“附件已变化，请刷新”。

### 4.3 文件路径约束

普通symlink按真实路径检查根；Finder alias先明确标成unsupported-kind，不把别名文件本体交给默认应用以绕过根。需要支持alias时另开明确设计，不隐式加入本包。保留POSIX元数据身份校验；不宣称NSWorkspace系统交接后的TOCTOU完全消除。Open/Finder均走同一授权动作，不能只保护Open。

保存保护仍收集所有账本声明附件，包括根外、缺失、错误重载新增声明；不能因为目录未授权就允许导出覆盖其路径。

### 4.4 验收

两账本隔离、同账本双窗口同步、重启bookmark恢复、根添加/撤销、目录替换/移动、失效bookmark、权限变化、symlink跨根、alias拒绝、刷新后身份更新。确保根更新不增加会计build_count、不改变snapshot/profile或余额；真实Swift/C和GUI实际验证根外收据授权后可打开、撤销后不能打开。

## 5. F03：查询长尾

### 5.1 范围冻结

本包完成现有矩阵已经承诺的postings来源语义，不实现JOIN/CTE/window、任意Python函数、完整SQL。entries/accounts/prices仍走既有旧入口；typed新来源不因本包自动加入。list/map metadata、系统__tolerances__字典、非有限Decimal暂时继续明确unsupported，避免引入未设计类型契约。

必须处理的格：balance Inventory列；sum(inventory)；显式GROUP BY合法性；GROUP/ORDER alias与ordinal及歧义；min/max/first/last已承诺Inventory/动态混合重载；现有日期/转换重载的边界；有限Decimal极端指数/陷阱与预算；重复include、保留metadata键和同日跨文件posting顺序。

“处理”不等于每个参考行为都新增支持：固定版不存在的签名必须拒绝；原范围外的重载明确列排除。已有group/alias代码先对照真实结果，正确则补证据，禁止为了制造开发任务重新实现。

### 5.2 balance与库存聚合

`balance`是参考源表定义的累计Inventory，不复用旧每账户显示字符串。实施入口先用锁定BeanQuery执行跨账户、同日乱序、include、WHERE、ORDER、LIMIT公开案例，确定累积是在过滤前还是过滤后、是否跨账户、输出时点及初始状态。该顺序是oracle调查门槛，当前不能凭直觉写死；结果记录到矩阵后再写实现。

实现将累计状态放在query扫描上下文，保持已booked posting的完整成本键、nil/空label、成本日期；查询局部结果不改变ledger库存。预算/取消逐posting检查。排序不得先改变参考累积次序。

sum(inventory)逐行合并每个Inventory的完整position键，NULL按固定版规则处理，空库存不等于NULL。沿用inventoryAccumulator的现有成本归约/精度规则；按oracle确定源序上下文，不能用聚合显示字符串解析库存或盲目相加累计余额。嵌套聚合仍按编译契约拒绝。

### 5.3 编译与类型

显式group校验必须先于扫描，空账本/LIMIT0也拒绝非法普通投影。alias解析记录具体位置；同名输出列仍保留ordinal ID，在需要唯一alias的位置返回ambiguous-column。ordinal合法范围与GROUP/ORDER位置分别验证，不默认套用其他SQL语义。

每一函数按输入type/nullable/返回type登记重载；动态object运行时转换、NULL、混合类型错误必须匹配已声明规则，不把错误变NULL。Inventory/Position比较或排序沿用完整键，禁止以str结果替代。

有限Decimal指数输入在创建巨型幂之前受预算控制；明确可接受范围及resource-limit错误。不要修改ledger.Decimal会计精度，也不要将非终止AVERAGE值强制变为28位小数。query context与财务exact值分别比较。

### 5.4 差分与产品说明

复用tools/reference锁定Beancount3.2.3/BeanQuery0.2.0，新增公开fixture和manifest；oracle生成期望仅用于审阅，gate不得重写。每格具备正例、NULL/空集、非法类型、错误类别、列标题/类型、位置值、重数、排序及资源行为。

矩阵每行记录implemented/oracle-verified/explicitly-unsupported/pending和具体证据，首页更新为163基线，旧147数字标历史。Native查询帮助来自已支持目录，不宣称未验函数可用；预设文本仍由Go生成且显式执行。

## 6. F04：会计源导出约束

### 6.1 产品语义

沿用既有 `orangecount export`：输出供外部工具使用的纯Beancount快照，原账本是唯一编辑源。分析文件不属于会计源；本包不新增原生账本编辑器或假设native已经有源导出按钮。

导出成功必须满足声明的可表示子集：语法可加载、booking/完整库存/金额/关键期间含义保持。无法证实的修订语义在写任何文件前明确拒绝，列出诊断位置和原因。首批对有效AVERAGE booking（包括全局默认/账户继承）拒绝源导出；不能只搜源文本“AVERAGE”。其他v3修订逐条分类，未证明可保持的类也拒绝，不能因为账本当前余额一致就允许。

Python plugin声明和外部工具依赖可能改变账本含义；未纳入声明子集时拒绝或明确要求用户先迁移，不能自动执行plugin。

### 6.2 Module与流程

把CLI编排中的计划与写入拆成拟议 `internal/sourceexport` Module：`Prepare(graph, parsed, evaluation, targetPolicy) -> Plan` 返回全部冻结字节、相对路径、入口、受保护源/附件及兼容诊断；`Write(plan,target)`负责事务。dialect.ExportText继续是语法转换Adapter，不承担目标写入。

处理顺序：源捕获/有效求值→有效booking/扩展兼容检查→dialect转换→验证相对路径/include映射→在临时输出树重新加载Go检查→一次性发布。任何前置失败不改目标。锁定Python往返作为开发验收，不成为Go产品运行依赖。

单文件使用同目录临时文件、完整write/fsync/close、目标保护复核、atomic rename。多文件仅接受不存在的目标目录：同父临时目录完成全部写入和自检，再原子rename；现有目录明确拒绝。跨平台不声称多文件覆盖事务，后续需要覆盖时另设计版本目录切换。

拒绝输出覆盖源文件、附件、别名/symlink/hardlink；相对路径不得逃逸目标。源图存在入口目录外include时，要么显式拒绝并定位，要么采用已设计的稳定重映射并同步改写include；本批选择拒绝，绝不沿用 `filepath.Rel` 得到 `../` 后直接写。目标在源图内部、输出会被include glob重新捕获时拒绝。

### 6.3 往返与验收

公开单/多文件、dialect、pad、metadata、完整成本日期/标签、FIFO/LIFO、全局/账户AVERAGE、容差/总成本修订、plugin及外部include案例。允许样例通过Go重新加载并与锁定Python比较完整entries/库存、余额、收支期间；按时间检查关键中间库存，避免仅末余额掩盖消费差异。

被拒绝样例确认目标不存在或原内容不变。注入第N文件写失败、fsync/rename失败、目录失效、symlink/hardlink替换，确认旧目标完整、临时树清理。现有export/dialectize fixpoint及legacy行为保留；新拒绝边界在CLI帮助和文档明确标出。

## 7. 跨功能状态与风险

| 场景 | 必须表现 |
| --- | --- |
| 捕获完成后同账本重载失败 | 保留旧捕获字节，保存前包含失败新输入的保护路径，显示捕获时间/身份 |
| 切账本/关闭窗口 | 不接收晚到捕获，不提交未完成保存，不误释放其他窗口会话 |
| 撤销附件根后打开旧引用 | policy revision不匹配拒绝；默认根和会计快照不受影响 |
| 缺价报表导出 | complete_total为NULL、valued_portion保留，缺失原因可读回 |
| 当前交易页有多条posting | 分页交易数和导出posting行数分别保存，禁止错标全集 |
| query语义修订 | 变更前后oracle和旧Adapter分别验证，必要时升级query contract；不冒用会计revision |
| AVERAGE账本请求源导出 | 写前明确拒绝；查询/报表分析文件仍可保存exact扩展值 |

发布风险按实际证据记录。设计存在不等于功能完成，Passing自动化不等于GUI已验，开发包不等于正式发行。

## F01 实施修订：嵌套集合与投影上下文

本节记录本轮实现对第3节拟议格式的具体化；查询v1不变。

- 报表v2增加 `list<record>`：schema.fields是每个元素固定record的字段；cell.value是由位置typed cells数组组成的数组。声明、历史、批次和归约贡献列表保持原顺序。仍受record深度8、每个列表10000项及文件预算约束，不支持任意Python对象。
- context必填 `valuation_details`，按稳定记录键保存完整ValuationStatus（包括缺价与使用的报价），避免仅保存行上的计数而丢失原因。可选 `supplement` 包含columns/cells，用于当前价格选择、批次所属组和归约事件等非主列表信息；由同一typed schema校验，零行也保留。
- 树报表与趋势保留平面列；账户概览、商品/价格、持仓组/事件使用根据已声明Go DTO类型生成的record列。不是根据第一行推断；浮点坐标和绝对路径在Adapter排除，exact、nullable、嵌套page完整保留。持仓选定组的历史导出主行是point，lots主行是lot，所属信息在supplement；未选择组的历史主行是series，含其points_page。
- 交易日记账按posting展平，原posting信息用record列；units_number转为PresentedDecimal，交易ID/index/date/tag/link独立列。账户概览导出包含已交付history，账户关联流水通过日记账交易入口另存，不把两种捕获范围混成一份完整流水。
- compound/record CSV的规范JSON按声明类型字段顺序输出；Decimal字段顺序为display/exact/approximate，record/set/list用已声明位置/集合次序。JSON对象输入字段顺序不决定读取后输出。公开样例在testdata/fixtures/analysis-v2。
- 原生协议升级3；菜单明确区分完整投影与当前页，面板显示捕获期间、会计语义、记录数和估值状态。源/附件保护仍在最终保存前刷新。

报表捕获增加独立capture_id与OC_CancelReportExport控制；序列化遍历/实际输出检查Go Context，原生取消不排队在捕获后。提前取消记录有32项上限，活跃捕获最多16，关闭取消活跃工作。接口调用的尝试身份与可重复使用的projection_id分开。
