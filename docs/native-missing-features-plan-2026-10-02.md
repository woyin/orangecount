# 原生缺少功能开发计划与实施步骤

状态：待实施。本文配套[详细设计](native-missing-features-design-2026-10-02.md)，基线0.1.1 / `278c6e4`。仅规划缺少功能，保留完整24任务范围；不将N04/R01/R02/R03发行验收移除。本文的拟新增文件、接口、测试与版本都是开发安排，尚无通过证据。

## 1. 执行规则与优先级

顺序为 **F01 → F02 → F03 → F04**。F01先交付资产负债表/损益的完整纵向流程，随后覆盖其他承诺报表；不一次新增所有DTO后再接界面。F04涉及既有直接写入路径，任何开发期间遇到覆盖源文件/逃逸目标的实际问题应立即提升为阻断，不等待排期末尾。

每个切片依次：公开固定输入和独立期望→必要的失败回归→最小实现→针对检查→真实C/Swift（若改协议）→原生实际操作→证据/说明→提交和对应开发包。自动测试已有覆盖时复用，不写镜像实现测试。原生编译串行，期间不修改Swift输入。

工作量采用相对级别：小=单一接口/文档及针对检查；中=一个后端加桥接切片；大=类型/会话/多页贯通。不是时间承诺。当前没有足够数据给出可靠工期，不用日历日期代替退出条件。

## 2. 开工核对（P00，小）

1. 核对git工作区并保留用户修改；读取0.1.1证据，不重跑所有旧门槛。
2. 将查询能力矩阵顶部更新为163场景与已交付工作台/导出/预设；旧段落标历史。
3. 建立 `docs/evidence/missing-features-status.md`：每个下面任务的代码状态、公开输入、检查、GUI、提交、包、外部缺口分栏记录。
4. 核对查询v1容器reader、report投影和源export当前Interface，确认本文拟议文件与命名适配当前代码；差异写设计修订，不在实现中静默改变合同。

退出：真实未实现格、可复用Module和代码身份确定，文档无“已有功能要重新开发”的安排。

## 3. F01：报表分析导出

| 编号 | 工作和拥有的文件范围 | 依赖 | 退出条件 | 量级 |
| --- | --- | --- | --- | --- |
| F01.1 | 公开报表v2样例、schema、独立reader；拟新增internal/analysis与tools/analysisverify版本分派 | P00 | 零行、多币种、partial、分页样例读回；恶意容器拒绝；查询v1兼容 | 大 |
| F01.2 | 树报表投影Adapter、publication投影身份/缓存、CaptureReportExport；desktop/session/projection及新report_export | F01.1 | 资产负债表/损益固定捕获与往返，无Double/源路径；身份淘汰及预算明确 | 大 |
| F01.3 | C入口、Bridge.h、Swift Models/Client/Store、树报表导出菜单和保存 | F01.2 | 真实C/Swift、保存/取消/覆盖/重载GUI；保存文件由reader验证 | 大 |
| F01.4 | 趋势、账户概览/历史Adapter与原生动作 | F01.3 | exact与缺价完整，曲线不从Double导出；scope/期间保持 | 中 |
| F01.5 | 交易/账户流水、价格目录/历史，页与行单位契约 | F01.4 | 多posting、重复价格、第二页准确；当前页不会标全集 | 大 |
| F01.6 | 持仓批次/历史/归约投影、record/set类型契约和原生导出 | F01.5 | nil/空label、AVERAGE exact、整池贡献/消费、未知收益完整读回 | 大 |
| F01.7 | 格式/预算/保存时序收尾，开发包 | F01.6 | 所有承诺section都有自动/真实原生证据与可读回文件 | 中 |

### F01.1 实施步骤

1. 首先固定context/schema的v2规范，特别区分valuation completeness、capture scope、truncated，以及交易page_unit/row_unit。
2. 制作公开资产负债表完整/部分估值、空损益、第二页交易的四成员样例；样例独立指定exact预期，不用生产writer生成“期望”。
3. 实现bounded reader和严格schema/type校验，复用查询v1成熟逻辑时保留其负例；record/set仅按v2接受。
4. tools/analysisverify新增版本分派和只读摘要输出，不执行query/ledger。格式unknown、duplicate、CRC、ZIP解压预算、extra JSON、伪page、非法NULL均固定反例。

### F01.2–F01.3 实施步骤

1. 将成功投影与归一请求绑定projection_id，固定Session缓存预算与淘汰行为；失败读取不产生可导出身份。
2. 树节点递归展平但保留parent、depth、direct/subtree inventory和两套valuation；原符号/真实零/unknown保持。
3. Capture先验证snapshot和projection身份，再从缓存冻结值输出；完整捕获只允许已实现的非分页section，并受取消/行字节预算约束。
4. 通过response→ZIP→reader一致性后增加C入口和required字段校验。测试不存在/已淘汰projection、旧snapshot、已关闭Session、乱序响应。
5. Store保存状态为idle/capturing/choosing/saving/succeeded/failed/cancelled；准备字节完成才打开sheet；应用generation变更拒绝迟到结果。
6. 复用AnalysisFileExport和ExportProtection，覆盖保护包括错误重载新增附件；实际保存后独立读取产物，不能只测试按钮触发。

### F01.4–F01.7 实施步骤

1. 每次新增一个section映射，固定列ID、nullable与顺序；更新格式例子/reader和真实C契约。
2. 账户流水与Journal按posting展平，source_record_count保存交易数量；page仍按交易计数。测试一交易3postings及跨页重复头。
3. 持仓采用完整成本键和独立归约record；零成本、nil/空label、未实现收益字段不得通过空字符串或0兜底。
4. 受控测试捕获中重载/切换/取消/关闭，确保旧捕获内容与新保护范围互不混淆；公开GUI至少每种保存入口成功一次，公共保存故障路径集中复用。
5. 完成F01后形成一次对应源码的开发包与功能说明，再进入F02；查询v1既有文件仍可读。

## 4. F02：附件目录配置

| 编号 | 工作和文件范围 | 依赖 | 退出条件 | 量级 |
| --- | --- | --- | --- | --- |
| F02.1 | documents策略身份、Session根配置和policy_revision；现documents/desktop | P00，按顺序在F01后 | 原子更新、旧策略引用拒绝、会计快照不变 | 中 |
| F02.2 | C配置入口、DTO/Models/Client/Store、原生bookmark偏好 | F02.1 | 根只来自显式选择；重启恢复与失效授权可区分 | 大 |
| F02.3 | 目录设置sheet、同账本窗口广播、刷新附件及Open/Finder动作核对 | F02.2 | 授权→打开→撤销→拒绝闭环，双窗口与两账本隔离 | 大 |

实施步骤：

1. 把根策略与publication会计结果分离，确定锁顺序：Session配置锁→document策略锁；文件系统耗时核验避免长期持有全Session写锁。读页持有策略快照，更新后不混用新旧revision。
2. roots按目录canonical身份去重；配置无效整次拒绝。新增policy_revision到DTO/动作，缓存键不能仅为record:index。
3. 新建原生AttachmentRootPreferences Module，负责ledger规范身份、bookmark、stale状态和窗口通知；View不直接读写UserDefaults或构造授权路径。
4. NSOpenPanel只选择目录；增加/移除成功后更新Go并保存偏好。任一阶段失败按可恢复状态提示，不显示已经生效。持久化失败需回滚活动配置或明确“本次有效但未保存”，禁止静默分歧。
5. 多窗口各自Session应用同一已保存配置；广播带配置代次，晚到旧通知不能复活撤销目录。账本移位后按新入口身份重新授权，暂不自动继承旧根。
6. 测失效bookmark、原路径新目录、权限失败、symlink逃逸、Finder alias拒绝、文件替换刷新；Open/Finder都必须验证snapshot+policy+generation。
7. GUI用独立公开账本和账本外receipt验证，避免修改FinanceBook或用户原生实例。

## 5. F03：查询长尾

### F03.0 固定调查清单（小）

为每项登记参考版本、公开输入、Go当前表现和是否实际缺失。已有正确行为只补必要负例/证据。矩阵排除项不能由本轮自动升级为支持项。

| 编号 | 待核对/实现格 | 主要文件 | 先固定的输入与失败 | 退出条件 |
| --- | --- | --- | --- | --- |
| F03.1 | balance Inventory与源/过滤顺序 | query.go、typed.go、compound.go | 两账户、WHERE删除前序行、ORDER反序、LIMIT0/1、同日include | 输出时点/累计范围与oracle匹配，完整成本键及预算 |
| F03.2 | sum(inventory) | typed.go、compound.go | 空/NULL、相抵、成本date/label、多币种、嵌套聚合 | 类型、NULL、键与精度逐格通过，错误不吞 |
| F03.3 | GROUP合法性、alias/ordinal | typed.go、解析/执行相关文件 | 非group投影、duplicate alias、非法ordinal、空输入 | 扫描前拒绝，错误位置/类别明确，重复标题不丢 |
| F03.4 | 已承诺聚合/转换重载 | typed.go、casts/coercion、compound | Inventory极值、动态混合、NULL/零行、错误类型 | 各重载有公开正负例或明确排除，无泛化宣称 |
| F03.5 | 有限Decimal极端指数与陷阱 | decimal_text/context与execution预算 | 巨大指数、取消、precision/tie、非终止exact | 有限声明范围明确、越限可终止，不修改会计数学 |
| F03.6 | metadata/include源顺序 | ledger求值/snapshot/query，按实际原因最小改 | 重复include、保留键覆盖、同日跨文件、stack显式覆盖 | oracle支持范围匹配，legacy/源AST隔离 |
| F03.7 | 矩阵/帮助/native集成 | matrix、QueryView帮助、公开预设 | 新类型解码、错误位置、cancel、CSV往返 | 已支持目录与实际实现一致，所有旧语料通过 |

F03.1–F03.6均为中到大，必须按独立语义切片执行，不一次重写query engine。每格记录函数输入/输出type、nullable、column title、位置cells、重数/顺序、错误类别与byte range。

执行方法：

1. 通过锁定离线工具生成并审阅参考期望；调查不是修改golden让Go变绿。
2. 参考顺序已确定后，在同一query扫描/编译Module实现；不从Report或Swift显示结果反推。
3. 共享ledger/snapshot若改动，执行会计51基线、AVERAGE与legacy隔离；查询变化保持全部163旧场景，新增场景真实计数。
4. 语义结果发生改变时判断query contract是否需要升级，记录迁移；新增支持而不改变旧签名可保留合同。会计revision不用于遮盖查询变化。
5. 最终四预设仍实际执行；不是仅检查生成文本。新版native解码、取消/错误位置及CSV保持。

## 6. F04：会计源导出

| 编号 | 工作和文件范围 | 依赖 | 退出条件 | 量级 |
| --- | --- | --- | --- | --- |
| F04.1 | 有效booking/扩展兼容清单与写前检查；convert、dialect及拟sourceexport | P00，按顺序在F03后 | 全局/继承AVERAGE及未证实扩展明确拒绝，目标未改 | 中 |
| F04.2 | 冻结Plan、源/附件保护、include相对路径验证 | F04.1 | ../、源覆盖、附件覆盖、源内输出全部拒绝 | 中 |
| F04.3 | 单文件原子写、多文件临时树+新目录发布 | F04.2 | 第N写失败/rename失败不留半套产物，旧目标不坏 | 大 |
| F04.4 | Go与锁定Python往返，CLI帮助及边界说明 | F04.3 | 可表示样例的完整库存/关键期间一致，fixpoint保留 | 中 |

实施步骤：

1. 从当前export代码固定真实行为，新增写前兼容拒绝及路径反例；原接口不能被测试假象证明已有安全保存。
2. 从已求值有效booking判断AVERAGE，不用字符串匹配；同样登记容差、插值/总成本等修订影响。先保守拒绝，再以独立往返证据逐类开放。
3. Prepare先产完整输出Plan与源/附件保护，尚未触碰目标；Go临时重新加载验证后才发布。
4. 单文件支持普通文件覆盖但保护源/附件；多文件首批只接受新目录，不对现有目录做逐文件替换。
5. 源入口目录外include明确拒绝；拒绝诊断定位原声明。目标文件树不得包含源绝对路径映射或目录逃逸。
6. 使用公开fault writer/真实临时目录注入失败，检查临时文件/树清理和旧目标；私人账本只读验证不写源导出。
7. 锁定Python仅作为开发工具，产物运行时不依赖Python。导出不兼容时分析文件仍可用于保存结果。

## 7. 检查命令与升级规则

这些是后续实施门槛，不是本次设计文档的运行结果。

| 变化类型 | 首先执行 | 冻结切片的必要扩大检查 |
| --- | --- | --- |
| F01格式/desktop | `go test -count=1 ./internal/analysis ./internal/desktop`（analysis创建后）及analysisverify公开样例 | 查询reader旧案例、desktop/bridge相关race；原生协议变动执行mac-test再app |
| F02目录/会话 | documents/desktop相关Go测试和race | 真实C/Swift配置及动作；公开原生授权循环 |
| F03查询 | query/compat/CLI针对检查、`make query-parity query-oracle-check` | ledger数值/源序变化扩大Go全量、会计差分、相关race及私人同源 |
| F04源导出 | dialect/CLI/sourceexport创建后的测试 | 锁定Python公开往返、legacy隔离及保护/事务故障 |
| 一个可交付版本冻结 | `go test -count=1 ./...`、相关race、`make fmt vet license parity-full query-parity`、`make mac-check` | 当前native archive私人只读探针；实际GUI；package解压校验及证据 |

不在每次文档或label修改后重复整个门槛。原生编译、测试、私人probe、benchmark不得并发共享build目录。编译输入冻结后才启动编译；新的代码变化只重跑受影响检查和最终必要门槛。

## 8. 交付和完成定义

每个F功能包交付：实现源码提交、功能使用说明、公开输入/格式样例、检查与真实原生记录、对应开发ZIP及SHA-256。包内build-info记录源码revision和工作区状态；文档提交可独立于包源码。

| 功能包 | 产品完成标准 |
| --- | --- |
| F01 | 所有承诺页面/section能捕获、保存并独立读回；partial/unknown/page/exact未丢；取消/失败保护成立 |
| F02 | 根配置可显式授权与恢复/撤销，窗口/账本隔离正确，旧引用失效，会计结果不变 |
| F03 | 本文冻结各格已验或明确排除；所有旧查询仍通过；native/CLI帮助不超出真实能力 |
| F04 | 不可表示的会计语义写前拒绝；允许子集往返一致；输出保护和单/多文件事务成立 |

完成四包之后，才进入N04/R01完整交互、辅助功能、性能与安装清单；R02/R03需要真实远端、机器和证书证据。四包完成也不代表24任务全部结束。

## 9. 开发时必须先解决的调查门槛

- F01开始编码前：确认报表v2的record/set schema和交易页/行计数规范，公开reader样例先成立。
- F02开始编码前：确认bookmark方案在当前非沙盒开发包和未来沙盒构建的分支，不编造系统授权已存在。
- F03.1开始编码前：锁定oracle真实balance扫描顺序，这是待调查事实，不是需要用户决定的产品偏好。
- F04开放任一修订语义前：提供独立会计往返证据，否则维持明确拒绝。

这些都可由开发者在当前环境调查；无需为常规实现选择反复请求用户确认。只有新增原范围之外的查询来源、账本编辑、alias自动解析或多文件覆盖产品承诺时，才需另立设计范围。
