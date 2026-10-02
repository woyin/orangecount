# Go 会计引擎与原生 macOS 实施进度

**最新功能交付（2026-10-02）**：缺少功能F01–F04已实施，范围及当前证据见[状态](evidence/missing-features-status.md)。报表v2分析、附件根配置、215查询/typed-v2、安全源导出均可用；35公开源往返/71库存截点通过。下列2026-10-01章节为历史证据，不再作为待开发清单。完整N04/R01、远端CI、旧系统和正式发行继续保留。

**历史实施基线（2026-10-01）**：Q02已实现有界查询执行、快照上下文、会话取消及真实C/Swift typed契约；完整工作台未完成。全量Go/七包race、51会计/147查询、原生三套/app及最新私人Swift/C通过（双方2833指令、零差异），见[本批证据](evidence/q02-bounded-execution-2026-10-01.md)。会计v3-core-3、默认legacy及查询contract保持；后台客户端与Store首批见[新增证据](evidence/q02-query-store-2026-10-01.md)；[QueryView/导航首批](evidence/q02-query-view-2026-10-01.md)已接入；错误位置与[保存query目录](evidence/q02-saved-query-catalogue-2026-10-01.md)已推进；下一步补preset、固定执行导出与完整验收。Q01剩余语义、X01/X02及GUI/安装/发行继续保留。

前次文档修订新增[开发入口](go-beancount-native-roadmap.md)与[查询能力矩阵](query-capability-matrix.md)，完善设计第27节、计划第15节及实施第33节。当时只读私人Go/Python和query/价格具名检查通过，16个公开oracle探针核实签名；未实施Q01新类型或重跑原生/GUI。具体范围见[复验记录](evidence/go-native-design-refresh-2026-10-01.md)，下列产品任务状态不因文档交付自动升级。

更新：2026-10-01。基于用户授权的 [设计](go-beancount-native-design-2026-09-30.md)、[计划](go-beancount-native-plan-2026-09-30.md) 和 [实施手册](go-beancount-native-implementation-2026-09-30.md)。这是当前实施证据，不替代完整验收清单。

## 前批动态/NULL实现记录（本次最新状态见开头）

前轮文档交接曾统一入口并通过8个文件、82个本地链接、17段shell语法、24个任务ID及12项当时源码身份检查；该轮没有修改产品源码。随后本批实现复合查询值并重新执行产品门槛，最新源码/输入身份和结果见[动态/NULL证据](evidence/q01-dynamic-null-query-2026-10-01.md)。当前从实施手册第37节继续，再推进Q02/X01/X02，保留全部GUI/安装/发行未验项。

M1 核心和 M2 多币种投影已有实现与检查，完整里程碑验收仍未完成；M3 的 A01/A02/A03 已有账户、持仓与价格页面及自动化。价格请求校验、非有限图表坐标和旧平面图表解引用 red 已修复；最新全量Go/race、原生三套与应用构建通过。此前新页GUI尝试遇到系统捕获失败，人工验收仍缺。M4的Q01已实现标量/复合值、声明重载及来源/stack/pad metadata，以及动态/NULL/unary已验子集，共91个strict场景；下一批是声明签名/库存/精度/预设，随后Q02/X01/X02。默认仍为 `legacy`；显式 `v3` 的修正不等于已经完成默认迁移。没有创建提交、PR、发布，也没有改动个人账本。

文档交接时的 red 及源码身份见[交接基线](evidence/go-native-planning-baseline-2026-10-01.md)，该记录保留为历史。后续实施已接续手册第28节并完成 Store/UI 代码，最新证据见本文件末节；不要再按旧 red 重做已完成实现。

B01 保存了 HEAD `20a9a6e86a5b4e2d6bb51aa2d2ae19dcd2934ddd` 的工作区补丁、59 个修改/未跟踪文件和逐文件 SHA256；本地备份位于 `/var/folders/7_/k830hyb566b3386880csl8900000gn/T/orangecount-b01-deh9k1t0`。这些文件不含个人账本。工具链：Go 1.27.1 darwin/arm64、Swift 6.4、SDK 27.0；离线 reference 使用 Beancount 3.2.3 / BeanQuery 0.2.0。

公开的 `testdata/fixtures/semantics` 固定了省略/显式日期 FIFO/LIFO、全局 booking、STRICT/SIZE、AVERAGE 与容差案例。修正前的 CLI 对照确认 6 个失败：省略日期 LIFO、全局 LIFO、STRICT 歧义、STRICT_WITH_SIZE、全局 AVERAGE、multiplier=1。

## 任务状态

“部分完成”表示已有代码和针对性证据，尚未达到该任务全部门槛。后续任务按计划依赖继续。

| ID | 状态 | 当前证据 / 剩余门槛 |
| --- | --- | --- |
| B01 | 完成 | 外部工作区备份、工具链、6 个可复现失败；此前全量检查是继承证据，后续改动仍需当前全量 gate |
| B02 | 部分完成 | profile/revision 入快照身份，Store 拒绝会话内切换；CLI check/query/serve/export，C 协议 2 与 Swift 显式 open/reopen；同源比较和输入身份检查；GUI 预览/应用已验证，最新格式化详情仍待复看 |
| G01 | 部分完成 | acquisition 日期写入已归约分录及库存，完整键合并；隐含日期 LIFO 与不同日期/label 批次回归通过；完整键、空/非空 label 和日期约束已有回归，完整 oracle 通过；历史投影由 A02 验收 |
| G02 | 部分完成 | 账户 → 全局 → STRICT，未知方法拒绝；全局 LIFO/AVERAGE 和覆盖测试通过；公开全局/覆盖 oracle 与真实 Swift/C 收益检查通过；GUI 标识待验收 |
| G03 | 部分完成 | STRICT 歧义拒绝、SIZE 等量选择、失败 booking 不插值/不变库存；NONE signed lots、等量最早日期、full-clear 与失败原子性公开 oracle 通过 |
| G04 | 部分完成 | v3 multiplier、default、cost 开关、独立 balance 容差；multiplier 与混合精度回归通过；default、cost 开关、旧名称诊断和非法选项公开 oracle 通过；声明范围外选项不能由此推断 |
| G05 | 部分完成 | 全局显式 AVERAGE 生效并登记 average-booking；连续买卖、完全清仓、零成本和失败守恒以及真实 Swift/C 精确收益通过；均价图末点与 GUI 待验收 |
| V01 | 部分完成 | schema 2 完整差分、固定 3.2.3、50/50 golden、负向 gate 和 make parity-full 已实现；查询/报表扩展随 Q01/N02/A02 补充，远端 run 待 R02 |
| V02 | 当前范围检查通过 | 当前2831条输入同源完整 entries/inventory、月度收支、真实 Swift/C 根汇总、三个截至日持仓/完整批次和价格目录/48可用性场景通过；私人有价选用未覆盖，公开69价格检查补足；查询等继续扩展 |
| N01 | 实现与自动化检查通过，GUI 部分验收 | 默认各原币种；完整树保留、每节点 direct/subtree completeness、null complete_total、报价日期、直接/倒数 local quote、空与合法零区分；真实 Swift/C、偏好回归和公开 fixture 操作通过 |
| N02 | 实现与自动化检查通过，GUI 部分验收 | 分币种系列、逐月 inventory 聚合与截至日 quote、nullable point/segment；Go/真实 Swift/C 回归与历史图 GUI 通过；完整 N04 仍待验收 |
| N03 | 当前页面自动化检查通过，后续页面继续扩展 | 持仓与价格 Store 已贯通；price pair/cutoff/Back/迟到/关闭与per-ledger偏好，真实历史/分页/重载通过；后续查询等继续扩展 |
| N04 | 部分检查 | 原币/估值/历史缺口 GUI 与客户端基准已有；实际最小窗口、大字体、完整键盘/辅助功能和 GUI 性能仍未验收 |
| R01 | 待当前验证 | 开发包独立安装阅读循环 |
| R02 | 待实际执行 | 远端 CI run；不能由 YAML 代替 |
| R03 | 待外部环境验证 | 目标系统实机；公开分发还需证书/公证证据 |
| A01 | 实现与自动化检查通过，GUI 待验收 | direct/subtree、截至日余额/期间变化、生命周期/booking/来源、静默月历史、报价缺口、流水分页、树/流水跳转和返回代次；真实 GUI 未确认 |
| A02 | 已有 Store/UI 及自动化检查，GUI 待验收 | 持仓/批次/贡献/事件/均价页面已接入；真实及受控客户端、101贡献池、include 报价来源、同键重买来源与当前私人三个cutoff通过；重放性能缺口已定位并修正，客户端复测记录见末节；GUI/AX/窗口验证仍缺 |
| A03 | 已有实现与自动化检查，GUI 待验收 | 商品目录、源历史/重复/分页、独立as-of及quote age、直接/倒数/零/缺价、Models/Store/PricesView、持仓跳转；客户端p95约12–24ms；GUI捕获-3811，未验渲染 |
| Q01 | 已验范围通过，完整任务未完成 | 147/147 strict；报价表示、日期提取及聚合端点已验，精度故障注入有效；分组/alias/balance/预设、数值/复制/loader长尾仍缺 |
| Q02 | 待实施 | 原生查询与 CSV |
| X01 | 待实施 | 全指令与配置记录 |
| X02 | 待实施 | 附件与分析导出 |
| D01 | 进行中 | 本文件持续记录实际状态，最终支持/能力矩阵待更新 |

## M1 早期验证记录

- `go test ./internal/snapshot`：profile 身份、Store 固定配置以及已添加的 v3 回归通过。
- `go test ./internal/ledger`：既有 legacy 会计测试通过。
- `go test ./internal/desktop ./internal/bridge ./cmd/orangecount`：当前 profile 会话及 CLI 检查通过。
- `make mac-test`：协议 2、真实 Swift/C profile 与重载、既有 Store 代次和生命周期检查通过（成本/booking 后续改动尚需重新跑跨语言 gate）。
- 本节记录时没有 GUI、远端 CI、旧系统、签名或公证验收证据；后续 GUI 范围见下节。

## 2026-10-01 完整对照复验

本轮当前执行 `make parity-full`：46/46 公开 oracle 案例通过，两个具名 AVERAGE 扩展另用独立数学测试；未知/缺失/被修改 golden 的负向检查已有测试。

本轮当前执行 `make verify-private-ledger`：重新构建真实 C/Swift archive 与探针；47 个源文件、双方 2,830 条指令、0 错误；missing/extra entries 为 0，changed inventory accounts 为 0；月度 Expenses 72 行、Income 22 行均一致；same_input_capture/native_checked/native_contract 为 true。只保留聚合摘要，无个人内容入库，没有修改账本。

继承的本批实现验证：全量 Go、核心 race、fmt/vet/license、mac-test 已通过；本次私有入口重新执行 mac-test。后续 N01 等改动仍要跑相关 gate。本节当时尚无新增 GUI；后续当前批次的 GUI 证据见下节。远端 CI、旧系统、签名、公证仍未验收。

## N01 当前批次

已改变新/未配置账本的报表默认为各原币种；不改变会计 profile 默认。显式目标币种保留完整原币列，另展示折算总计或已折算部分/缺报价。报价仅使用截至日的直接本地币种对，找不到时允许非零倒数；不隐式多跳、联网或使用未来报价。成本/units 与币种偏好按账本保存，reload 与 reopen 保留，另一账本不继承。

该批针对性 `go test ./internal/report ./internal/desktop ./internal/bridge -count=1`、`make mac-test` 和 app 构建通过。公开 fixture 为 `testdata/fixtures/native/valuation.bean`。N01 当时未做 GUI；后续 GUI 与 Dashboard 的 N02 改进见下节，不由 N01 单独推断历史图正确。

## N02 与当前 GUI 批次

`StatementHistory` 一次遍历已归约 posting，按每个实际月末/最后截止日计算库存和当日可用报价。原币按币种分面板，选定目标币种后历史不可估值点返回 null，segment 防止跨缺口连线。成本、数量与明确目标币种的市值有独立标签；未选目标币种不能进入市值模式。

当前全量 `go test -count=1 ./...`、相关 report/desktop/bridge/snapshot race、`make fmt vet license parity-full`、`make mac-test`、`make app` 均通过。新增真实 C/Swift 检查包括一月缺价、二月无交易却重新估值、三月清仓；最新私有入口复验仍无 entries/inventory/monthly 差异且 native_contract=true。

当前 GUI 在隔离 QA bundle、本机 macOS 27.0.1 / arm64 完成：Cmd+O 打开公开 valuation fixture、原币默认、零与空账户、截至日前 USD 部分折算、保留 USD/EUR 原币、partial 标签；history fixture 的成本与市值切换、二月/三月 1100 USD、明确一月缺价及线段断点；legacy→v3 比较/应用后显示 v3-core-1 并有效。只操作公开合成账本，QA 已正常退出，未关闭用户原有 OrangeCount 实例。

GUI 检查暴露迁移详情原先直接显示 AST JSON；已改为日期/摘要/来源行、成本及报价的账本描述，真实 Swift/C 呈现回归和 app 构建通过。此最新详情渲染仍需 GUI 复看。当前截图窗口不能代替精确 900×600、大字体、Dark/英文、拖入、完整键盘/AX、性能、双窗口新页面或独立安装循环；这些仍由 N04/R01 验收。

## N03 与性能补充记录

最新 `make mac-test` 已包含 AccountValuation 必填 nullable 字段及完整性不变量的负向解码；同页切换币种/截至日时先返回新请求、后返回旧请求，旧响应不得覆盖新状态。迁移详情也有真实 Swift/C 呈现回归。`make app` 构建包含 QA 专用真实字体倍率实现；字体放大尚未做实际 GUI 验收。

`make mac-benchmark` 的完整合成输入、环境与样本见[客户端基准记录](evidence/native-client-benchmark-2026-10-01.md)。52,500 笔交易、显式 v3，两种场景打开加首张表读取为 1,873 / 1,348 ms，四页读取 p95 为 2–63 ms。这是客户端调用链路，不能验收 GUI 首帧、交互 p95 或构建后的发布额外延迟。

本次文档整理核对当前接口与记录，补充设计/计划/实施场景；不重新执行未受文档变化影响的全量测试。前述测试与 GUI 均为同一持续开发任务的已有证据，文档更新本身不产生新的产品验收。

## A01 账户详情当前批次

新增 `report.AccountDetailView`、desktop 的 `account` view、typed `account_detail` 载荷与 `AccountDetailView.swift`。同时保留直接/子树余额；scope 只决定期间变化、历史与流水。日期区间首尾包含、期初存量保留；纯文本搜索仅过滤流水，不改变余额。账户目录可查看汇总；相似前缀不匹配，未知目标通过 C 返回 account-not-found。生命周期、有效 booking 和 open/close 来源原始行可见。

余额历史按各月真实末点重放已归约 posting，原币分系列，目标币种沿用 PriceAt 的截至日报价和 nullable/segment 缺口。关闭账户历史保留。账户树和展开流水的账户可进入详情；返回栈恢复原页面请求，树筛选/折叠状态继续由既有 State 保留。新账本及关闭清栈；请求代次拒绝返回后的迟到详情。分析口径独立保留，流水请求归一为 at-cost，避免把市值参数带入不支持的交易视图。

公开 `native/account-detail.bean` 与已有 history fixture 固定预期。已观察账户入口拒绝的 red、历史字段缺失的 red、导航方法尚未实现的 red，以及市值进入流水请求的 red；相应实现后 targeted desktop 和真实 C/Swift/Store 检查通过。当前全量 Go、相关 report/desktop/bridge race、fmt/vet/license、46/46 full parity、mac-test 通过；最新 `make app` 已在日期输入同步改动后重新执行并通过。

新增页客户端基准使用同一 52,500 笔输入，20 个样本，p95：混合币种 104 ms、USD 79 ms；完整记录见[账户详情客户端基准](evidence/native-account-client-benchmark-2026-10-01.md)。不证明 GUI 渲染或交互延迟。

本次 GUI 尝试在隔离 QA bundle（1.5 字号倍率、dark）完成欢迎页 AX 读取，但截图仅返回空白；公开 fixture 在 Open 面板被选中后 Open 仍 disabled，无法确认账本打开。Esc 与退出动作有效，随后进程检查确认 QA 已退出，原有实例保留。没有把启动配置、AX 选中或构建成功算作账户页、大字体或 Dark 的验收。A01/N04 的实际页面、返回/滚动、快速切换与热重载 GUI 仍待验证；私有账本的新账户投影补充亦待后续 V02。

## A02 首批代码与本次文档整理

工作区已有 `internal/ledger/inventory_replay.go`、`internal/report/native_holdings.go`、`native_average_history.go`、`native_reductions.go`，desktop 新增 holdings 入口。公开输入 `testdata/fixtures/native/average-history.bean` 覆盖买入→部分卖出→新买入→再次卖出→全清→重买。历史数量、成本、均价和三次收益用独立有理数预期；M 腿不增加经济事件，清仓后图表分段。

本次重新执行 `go test -count=1 ./internal/desktop -run '^TestNativeHoldings'`，两个测试通过。该结论只覆盖首批数学场景，不覆盖全部 A02 契约。当前尚无 Swift holdings DTO 或持仓页面；分组、估值/未实现差额、费用/转移/@@拆分等边缘、完整分页/预算、真实 Swift/C 和 GUI 仍需实施。此前全量 gate 发生在这批 Go 代码之前，不能继承为当前 A02 全量通过。

本次只读执行 `sh tools/reference/differential.sh`，使用用户授权的外部账本入口：47 个源文件，双方各 2,831 条指令、0 错误，missing/extra entries 和 changed inventory accounts 均为 0；月度 Expenses 73 行、Income 22 行一致，same_input_capture=true、native_checked=false。输入已较上段的 2,830 条发生变化；只记录聚合数，不保存源码、账户、金额或私人路径。当前验证没有新增 GUI 或真实 Swift/C 证据。

本次交付聚焦设计、计划与实施手册：补投资计算/收益归属、查询最小矩阵、当前 A02 剩余切片及可执行验收步骤；未修改产品源码，未创建提交或发布。下一开发入口为计划第 11 节及实施手册第 26 节。

文档检查通过：4 个文件的本地链接、代码块闭合、行末空白、24 个顶层任务及编号；`git diff --check` 通过。已明确 M1 核心与 M3/G05 均价页面的验收归属，避免反向依赖，完整目标未缩减。

## A02 后端、分页与真实 Swift/C 当前批次

本轮继续实现而非仅修订文档。公开新增 `holdings-intraday.bean`、`holdings-trades.bean`、`holdings-identities.bean` 和 `holdings-unknowns.bean`，均为合成输入。复现同日清仓重买仍连旧曲线的 red 后，改为交易级清仓分段、日末合并呈现；M 交易内部重组不造成假清仓。完整 lot ID 只采用精确成本键。

重放保留实际消费 lot 的原建立来源，归约事件与消费 lot 的来源分别可追溯。FIFO/LIFO 多腿 @@ 原始总价只计算一次；明确归属费用得到 gross100/fees5/net95，未归属 Expense 和多个归约保持净收益未知。转移、无售价、事件后才出现的汇率、未核对的现金和负售价各有明确 reason；重复内容的不同源交易保留独立事件。

`group_by=account|commodity|cost_currency` 已接入；跨账户聚合仍按商品/成本币种分开，保留每个 lot 的账户与精确键。成本和市值各自有 AccountValuation，分别使用截至日本地 quote；目标成本缺汇率时可以有市值，但未实现差额为 null。普通无成本和零成本保持不同；NONE 开空保留 signed 库存并不伪报多头出售收益。新增 report 测试读取46个固定外部 golden 中的32个合法案例，将重放后的全账户完整库存与 Python 3.2.3 预期直接对照，通过；非法案例仍由完整核心 gate 检查。

desktop 新增 `holding_section=groups|events|history|lots`，group_id/event_id 选择器必须绑定 SnapshotID；分组、事件、系列、批次和历史点均有各自 page 元数据。默认100项预览，均价预览保留最近100点及实际 offset；选择器可以读取剩余批次/历史点。完整数量/成本/收益不由页内元素反算。source event 的101批次案例验证预览100、下一页1和 reload 后旧ID/旧快照拒绝；超大合成摘要验证16 MiB响应预算及结构化错误。预算包括 state/summary，并为C envelope保留字节。

Swift Models 已有 holdings typed 载荷、枚举和 required-nullable wrapper，编码也保留显式 null；真实 C/Swift 验证18/440/220/9、920/3收益、lot selector、零/无成本、nil/空label、转移/缺报价/负价 reason，以及75 EUR市值、25 EUR成本、50 EUR差额。删除必填 nullable key、未知 inventory kind、矛盾成本字段会拒绝解码。没有将此解码证据当作页面渲染验收。

当前已执行通过：全量 `go test -count=1 ./...`；ledger/report/snapshot/desktop/bridge race（后续预算与分页改动又复跑 desktop/bridge）；`make fmt vet license`；`make parity-full` 46/46；`make mac-test`；`make app`。最后的预算、源事件选择器、跨成本币种/快照回归之后重新执行全量 Go、desktop/bridge race、mac-test 和 app，均通过。未运行新的 GUI 或性能基准；私人账本证据沿用上个文档批次，不冒充本轮新投影验证。

收尾又验证明细 Sort 作用于选中的 lots/history points；无目标账户的 direct 范围被拒绝。相应 red 后修正并重新执行全量 Go、desktop/bridge race、mac-test、app、vet/license 和 diff 检查，通过；desktop 目前有12个具名 holdings 检查。文档互链/代码块/空白/私人路径检查通过。没有仍在运行的构建或测试会话。

下一步：补 AVERAGE 归约参与的原批次与重组来源展示（当前事件只显示实际消费后的成本 lot）；提供 quote 来源文件定位；接入 HoldingsView 和 Store 的分组/截止日/选择器导航及时序；再做新页 GUI、性能和 V02 扩展。A02/M3 尚未完成；A03、Q01/Q02、X01/X02、完整N04/R01和R02/R03仍按原计划推进。

## 本次详细文档交接复验

本次完善设计、计划与实施手册：同步当前 A02 合并贡献契约、原生页面结构、A03–X02 目标数据结构及逐步实施切片；保留24个顶层任务和全部验收门槛。文件里区分实际实现、当前检查、继承证据及目标行为，不将文档完成写为产品完成。

实际新验证为两个 Go 包的 holdings/replay targeted 检查通过；`make mac-test` 中真实 ContractTests 通过，StateTests 因 NavItem.holdings / showHoldingLots / showHoldingEventLots 缺失而编译失败，NativeClientTests 未执行。现有失败测试保留，下一动作是实现 Store 接口并运行全部原生检查，再接 HoldingsView。

私有只读同源对照当前再次通过：47个文件、双方2831条指令、零错误、零分录及库存差异，月度Expenses73行/Income22行一致；native_checked=false。当前输入未执行私人真实 C/Swift，新持仓 GUI/性能也未验证。详细命令、环境及12个公开源码/fixture身份见[交接基线](evidence/go-native-planning-baseline-2026-10-01.md)。本轮产品源码及私人账本均未修改。

## A02 Store/UI、当前私人持仓与重放性能

已完成 NavItem/Store 持仓入口、snapshot selector、两类事件批次、Back/过滤/重载清理，新增 HoldingsView 及账户跳转。页面包含分组/持仓/估值完整性、消费与整池来源、未知毛/净收益原因、均价分段和 exact 列表；仍由单一 Go 会话计算。现有 StateTests 编译 red 已转为执行通过，真实 NativeClientTests 覆盖历史/贡献分页、降序和 reload 清过期选择器。101贡献池、include报价源范围及同键清仓重买来源回归通过。

本次当前全量 Go、ledger/report/snapshot/desktop/bridge race、fmt/vet/license、46/46 full parity及具名AVERAGE、原生三套检查、最新app构建均通过。当前私人 make verify-private-ledger 同源捕获47文件/2831指令、零错误/分录/库存差异、Expenses73行/Income22行一致；native_checked/native_contract=true、native_holding_cutoffs=3，比较完整数量/成本/均价/批次并实际读取 Swift/C 分页。只有脱敏摘要入库，源文件未改。

52,500笔合成客户端测量暴露持仓293–440ms的p95缺口；profile确认重放线性批次比较主导，已用一次调用内的完整精确键索引替代，未加跨快照缓存/改财务精度。相同输入/响应大小/计数复测p95为57–67ms；Go单请求均值375→57ms、临时分配473→59MB。完整前后原始样本、源码身份与命令见[A02证据](evidence/a02-native-holdings-2026-10-01.md)。

GUI尝试受ScreenCaptureKit -3811音视频捕获失败阻断，没有新持仓截图/AX/窗口/字体/键盘证明。隔离QA进程已清理，用户原应用未关闭。A02实现与自动化已前进，GUI及G05曲线实际展示仍待验收；A03/Q01/Q02/X01/X02、N04/R01完整循环、R02/R03环境证据仍需继续，完整目标未完成。下一独立开发入口为实施手册第29节A03，保留全部未验收项。

## 本次详细设计、计划与实施步骤交付

按用户的文档请求完善三份主文档，修正持仓尚未实现、价格完全未开工、PriceAt所在Module等过时信息。补设计第24–26节价格完整Interface、查询typed/执行身份/预算、附件状态及version1分析文件；计划第13/14节给出当前顺序与分批退出条件；实施第30–32节给出已有代码接续、公开数学预期、跨语言/Store/UI顺序、检查与最终交接。24个顶层任务和M0–M4范围保留。本批未修改产品源码或私人账本。

本次发现并保留当前A03 red：已有Go价格目录和as-of测试通过，但prices请求校验接受units，三项targeted整体失败。Swift价格模型/Store/UI仍未接入，不能沿用前批A02全量green宣称当前门槛通过。后续第一步是实施手册30.1修复该校验，再补独立边缘及真实C/Swift。

本次私人同源Go/Python差分重新通过：47文件、双方2831指令、0错误、零分录与完整库存差异、Expenses73行/Income22行一致；same_input_capture=true，native_checked=false。本次未重跑私人Swift/C、GUI或性能；保留前批A02证据及所有未完成验收。具体命令、公开源码身份和限制见[文档交接证据](evidence/go-native-documents-handoff-2026-10-01.md)。文档本地链接/锚点、代码块、任务数和空白检查通过；产品完整目标尚未完成。

## A03 价格页面及当前端到端自动化

本轮继续完整开发目标，已修prices/units校验red，新增8个具名价格检查，覆盖直接/倒数/零/缺价/未来价、目录历史商品、源重数/include、分页/排序/reload、选中商品完整上下文和16MiB预算。极大合法报价先复现Infinity；共享坐标改null后又复现旧平面仪表盘解引用panic，已通过独立公开回归修复，保留exact/complete_total并按缺口分段。

Swift新增prices typed载荷及严格required-nullable/page/pair不变量；Store/NavItem支持价格、Back、分页、quote独立于分析currency、neutral估值不覆盖偏好、跨页参数清理、reopen和pair/cutoff乱序/关闭。PricesView和持仓跳转已接入，源曲线标明当前页、精确列表及源动作同时提供。全量Go、核心5包race、fmt/vet/license、46/46 full parity及具名AVERAGE、mac-test三套实际运行、最新app构建均通过。

独立Python临时gold扩展商品目录/源价格重数/各cutoff选用；公开输入69价格检查通过，故意篡改selected rate实际使native_contract=false/exit1；空Expenses查询的对照脚本TypeError亦修复。当前私人同源47文件/2831指令，零分录/库存差异、Expenses73行/Income22行一致，真实Swift/C根汇总、3持仓cutoff及48价格可用性检查通过。私人没有源price，有价算法由公开输入验证，不能据私人缺价证据泛化为真实市价等价。私人输入未改，只记录汇总。

同一52500交易/4000报价公开输入，20样本价格目录/历史客户端p95为12–24ms；不是GUI响应延迟。最新QA getApp仍失败ScreenCaptureKit -3811，无法验页面截图/AX；只清理本轮QA进程，原应用保留。完整原始样本、源码身份与命令见[A03证据](evidence/a03-native-prices-2026-10-01.md)。

下一独立入口为实施手册第31节Q01/Q02，随后X01/X02，保留全部A01/A02/A03/N04/R01 GUI/安装以及R02/R03外部验收要求。A03实现与自动化已交付，不等于M3全部验收；完整24任务目标仍在推进。

## Q01 类型化查询基础当前批次

新增ParseStandard/EvaluateTyped及显式CLI typed-json，复用执行链；有序schema/ordinal列ID/位置cell保留重复标题，integer字符串、exact decimal、NULL与空字符串/false分开。普通来源使用posting行，pad有效来源继承balance；导航Span保持原行为。共用presentation精确/显示策略，默认会计配置仍为legacy。

本批已观察并修复默认入口缺失、重复标题/空字符串、空数据非法WHERE被隐藏、LIMIT0、除零、聚合类型、metadata范围、GROUP吞错/类型碰撞、日期误算成1997和数字字符串比较/排序red。旧EOF文案的广泛回归也已修复；旧map现在明确拒绝重复标题。

当前全量Go、7包race、fmt/vet/license、46/46会计差分、22/22只读query golden及pinned oracle-check、故障注入、原生三套和app构建通过。私人core-only同源47文件/2831指令无差异；随后最新C archive重新编译探针的真实Swift/C补充也通过，3持仓cutoff、48价格可用性场景，不执行新的typed query。本批没有查询C/Swift DTO或GUI，无新的查询性能/预算验收。细节见[Q01当前证据](evidence/q01-typed-query-foundation-2026-10-01.md)。

完整Q01仍缺compound、系统/有效stack metadata、全部NULL/动态类型、函数重载和预设；随后推进Q02、X01/X02及完整GUI/安装/发行验收。已有工作区、私人源文件和用户原应用保留，没有提交、远端run或发布。

## Q01 复合值、库存及标准报价weight当前批次

已接续第34节，实现Position/Amount/Inventory与完整nullable成本、units/cost/number/currency、sum(position/amount)、库存归约、金额meta和三类ORDER BY。内部排序次序与财务身份/规范化输出分开，Span不参与分组；复合直接比较按固定oracle拒绝。无成本报价weight采用归一单价的28位half-even；源总价、旧weight和精确会计行为保留。

四个公开query corpus共53场景通过，新增31个独立期望；按切片观察red→green。成本键四字段篡改测试及真实CLI外部gold故障注入均检出；缺失注册golden会失败。全量Go、相关七包race、fmt/vet/license、46/46会计差分、53/53query和固定oracle复验、原生三套及app构建全部实际通过。复用最新C archive重编译私人探针后，同源47文件/2831指令、零分录与库存差异、支出73行/收入22行一致、native_contract=true、3个持仓cutoff/48项价格可用性检查；不执行新typed query或GUI。

完整源码身份、检查范围及边界见[本批证据](evidence/q01-compound-query-2026-10-01.md)。下一批从手册第35节系统与有效元数据开始，随后完整NULL/动态类型、日期/其他聚合、balance/sum(inventory)、数值精度和预设。Q02、X01/X02及GUI/安装/发行保持完整原范围，目标继续。

## Q01 有效metadata与本轮文档交接

实现系统filename/lineno及Integer cell、显式NULL/空/false优先级、文件独立词法stack、求值副本与AST/legacy隔离，以及pad交易/两个posting的用户metadata。非法stack阻止有效快照，诊断和双语修复指引有实际触发fixture。普通交易容差字典明确unsupported，避免把存在但未表示的对象假作NULL。v3 revision升为v3-core-2；默认仍legacy。

本轮七组query corpus共63场景、50个普通会计完整golden与独立AVERAGE通过；query golden schema 2绑定完整include源图，子文件只改注释的负向检查实际失败。全量Go首次发现新诊断缺指引，补齐后重新全量通过；七包race及fmt/vet/license通过。真实Contract/State/NativeClient三套与app构建通过，没有query DTO/UI或新GUI验收。

复用本轮新C archive重编私人探针，47文件/双方2831指令、零错误、零分录与库存差异、Expenses73行/Income22行一致，same_input_capture/native_checked/native_contract=true，3持仓cutoff/48价格可用性检查。账本只读；不验证新typed查询，也不把无源price账本的可用性检查当真实报价数学证据。

三份主文档新增设计第28节、计划第16节、实施第36节；统一当前入口、能力矩阵和默认/版本，给出文件责任、独立期望、red→green及升级门槛。下一项是metadata动态比较与完整NULL，再继续剩余签名/精度/预设、Q02、X01/X02，保留所有GUI/安装/发行门槛。完整命令、当前公开源码身份与未验范围见[本轮证据](evidence/q01-metadata-query-2026-10-01.md)；前批哈希保留历史。

## Q01 动态绑定、NULL及逻辑当前批次

继续36.2节，已按pinned规则拒绝两个object直接比较；一侧静态类型时转换object到string/finite Decimal/date，Integer目标提升Decimal。源精度文本与exact/group身份分开；加入NULL/布尔字面量、IS判定、unary、括号标题、NOT优先级、左右短路和动态WHERE真值。库存隐式真值明确拒绝；NaN/Infinity不能表示为有理数，也明确拒绝而不伪作NULL。探针还确认abs(int)无固定签名，实际LIMIT0 red后修正旧typed误接受。

新增dynamic-values28场景，当前八组共91/91 strict query通过；外部临时gold将NULL改false时真实CLI exit1/failed1。全量Go、六包race、fmt/vet/license、50/50会计及独立AVERAGE、原生三套/app通过；abs补充后全量及query/compat race、所有gate与原生/app再执行。当前私人检查和版本身份见[动态/NULL证据](evidence/q01-dynamic-null-query-2026-10-01.md)。

没有Q02原生查询/预算/取消或新GUI。下一具体入口为实施第37节显式转换及剩余声明签名，再补balance/sum(inventory)、精度/预设；loader长尾保留36.3。Q02/X01/X02及所有未验GUI/安装/远端/实机/签名公证仍是完整目标范围，不标完成。

## Q01 显式转换与标量Decimal上下文当前批次

从手册37节的已知red继续，先修复算术精度/负零、组合decimal及date bool；按独立oracle/red逐步增加加减乘、除法、unary/abs和SUM上下文。`decimal_text.go`统一标量28位half-even及quantum，财务identity与源表示分开；构造/Integer/legacy及ledger/AVERAGE不统一舍入。源码、实际red→green、命令和身份见[本批证据](evidence/q01-casts-query-2026-10-01.md)。

casts最终25场景、九组共116/116通过，独立源序例区分逐步SUM与最后一次舍入。外部临时gold故障注入3.00→3时真实CLI exit1/checked25/failed1。当前产品源码全量Go、七包race、fmt/vet/license、50普通会计与独立AVERAGE、原生三套及app通过；最后只补公开SUM场景后query/compat/CLI与全部query gate重跑。最新私人捕获47文件/2831双方/0差异，真实Swift/C、三个cutoff与48可用性检查通过；没有新typed私人查询或GUI。

当前入口38节：复合str/number(amount)表示及成本/库存上下文，再补剩余声明签名、balance/sum(inventory)、预设和loader长尾。Q02、X01/X02及所有未验GUI/安装/远端/实机/签名公证仍保持完整范围，目标未完成。

## Q01复合转换与金额表示当前批次

从38节继续，已修复str复合值及混合普通/聚合的自动分组；金额提取保留精度，成本乘法/库存累加采用已验查询上下文。Number/Cost表示与Decimal内部精度分开保留，不改变Raw、Rat、比较或财务JSON；FIFO拆分、插值、总成本单位精度和高精度查询成本有独立公开覆盖。当前十组127/127、全量Go/七包race、会计50/50及AVERAGE、原生三套/app和当前私人真实Swift/C通过。外部gold注入2.00→2时CLI实际退出1，私人47文件/2831指令/0分录与库存差异、收支73/22一致，三个cutoff/48可用性通过。详细命令与20项公开身份见[本批记录](evidence/q01-compound-casts-query-2026-10-01.md)。

没有新GUI或原生查询；报价表示、其他数值/复制及查询签名/group/balance长尾仍缺，继续39节及Q02/X01/X02，完整24任务未完成。

## Q01报价、日期与聚合端点；v3-core-3

无成本@/@@的weight表示、first/last/min/max已验范围及year/month/day已实施。公开零数量@@发现原总报价误计weight导致无效，已修正booked零单价和balancing；保留AST/源总价，显式v3修订core-3、legacy不变。当前51会计/147查询（十一组）、全量Go/七包race、原生三套/app和当前私人真实Swift/C通过；高精度price fixture最后补充后query/compat/CLI及全部query gate重跑通过。MIN文本1.0→1的临时gold故障注入实际exit1；私人47文件/2831双方、零分录/库存差异、收支73/22、三个cutoff/48价格检查通过。具体red/命令/23项公开身份与未验边界见[本批证据](evidence/q01-date-aggregate-price-2026-10-01.md)。

继续39节分组/aggregate位置/alias/ordinal、balance/sum(inventory)/预设及数值复制长尾；可以按31.2推进Q02预算/取消与原生协议。没有新GUI/安装/发行证据，完整24任务不关闭。

## Q02有界执行与C/Swift契约首批

已实现ExecuteTyped资源/行/字节预算、实际Context检查、快照上下文及fingerprint、单调请求代次与取消高水位、会话串行gate、C执行/取消接口及Swift严格typed decoder。真实red修复与当前完整检查见[证据](evidence/q02-bounded-execution-2026-10-01.md)。最新私人双方2833条指令相等，旧2831记录为此前捕获。完整Q02仍缺后台客户端/Store/QueryView、精确错误range、保存query/预设及CSV/GUI；不关闭24任务。

## Q02后台客户端接入（2026-10-01）

NativeLedgerClient已接typed执行和独立取消控制路径，真实客户端成功/预先取消/后续代次及耗时表达式取消检查通过；将取消排回执行actor的临时变异被测试拒绝。详见[证据](evidence/q02-native-client-2026-10-01.md)。Store、QueryView、错误位置与导出尚未交付，完整Q02不关闭。

## Q02 Store查询状态首批（2026-10-01）

已接窗口单调请求、捕获执行身份、current/latest-pending、草稿与成功/失败/取消/过期记录，以及reload/换账本/关闭保护。受控队列与真实C/Swift Store新增检查见[证据](evidence/q02-query-store-2026-10-01.md)。尚无QueryView/导航、错误位置或导出；完整Q02继续未完成。

## Q02 原生查询页面与导航首批（2026-10-01）

QueryView、原生结果表及query摘要读取路由已接入，进入不自动执行、离开取消和分析偏好恢复有当前受控/真实C证据；当前全量Go、相关race、原生三套与app通过，见[证据](evidence/q02-query-view-2026-10-01.md)。错误位置、保存query/preset、CSV及GUI/完整资源边界继续未验收，完整Q02不关闭。

## Q02 错误位置首批（2026-10-01）

standard token/函数/字段错误范围已沿Go→真实C/Swift→NSTextView接入，Unicode转换及不分割字符的契约通过。当前六位置例子、147查询、Go全量/相关race、原生三套/app通过，见[证据](evidence/q02-query-locations-2026-10-01.md)。操作符/源/别名及其他全局错误完整覆盖、实际GUI选择仍未验收，完整任务不关闭。

## Q02 操作符/来源/别名错误位置及预算范围（2026-10-01）

已补实际操作符、FROM与别名引用位置；11位置例子与公开低steps/原生numeric预算空范围检查通过。初次race的deadline竞速已改为确定性预算测试，最终四包race、原生三套/app通过，见[证据](evidence/q02-query-location-coverage-2026-10-01.md)。继续保存query/preset与固定执行导出，实际GUI及完整任务尚未完成。

## Q02 保存query目录与草稿选择（2026-10-01）

账本query指令目录、include来源、重复名称独立ID、快照分页及草稿选择已贯通Go→真实C/Swift→Store/QueryView；不会自动执行。当前Go全量/相关race与原生三套/app通过，见[证据](evidence/q02-saved-query-catalogue-2026-10-01.md)。继续preset与固定执行导出；实际GUI、完整资源边界及完整任务未关闭。

## Q02 类型化 CSV 与命令行入口（2026-10-01）

已实现 TypedResult CSV writer 与 `--format typed-csv`，保留列序、重复标题、精确数值和完整库存成本；公开测试、当前 Go 全量及 query/CLI race 通过，见[证据](evidence/q02-typed-csv-2026-10-01.md)。CSV 空字段不能区分 NULL 与空字符串；原生固定执行导出、原子保存、GUI 验收及完整任务仍未完成。

## Q02 分析容器内容生成（2026-10-01）

新增成功 QueryResponse 的 `.ocanalysis` 四成员 ZIP 内容生成；实际解包验证上下文、重复列名、高精度、NULL/空字符串和零行，失败执行拒绝及 writer 失败传播通过。当前 Go 全量、desktop/query race、fmt/vet/license exit0，见[证据](evidence/q02-analysis-container-2026-10-01.md)。原生固定捕获入口、独立验证器、多币种/截断等容器样例、来源路径策略及原子保存仍待完成，完整任务保持未关闭。

## Q02 会话固定导出与 C/Swift 契约（2026-10-01）

当前成功执行独立缓存与按身份捕获字节已实现，调用方修改嵌套库存不会改变导出；错误身份/后续失败的旧代次拒绝。真实 C/Swift CSV/ZIP 传输和当前 Go/相关 race、原生三套通过，见[证据](evidence/q02-query-export-capture-2026-10-01.md)。继续客户端/Store/保存面板、原子替换和目标保护；完整验收尚未完成。

## Q02 原生导出客户端与 Store（2026-10-01）

客户端异步捕获及 Store 成功执行导出动作已接入；草稿不替代执行身份，后续失败不能导出旧成功，异步期间的新请求/快照变化/关闭窗口拒绝迟到内容。真实和受控检查、原生三套、fmt/vet/license通过，见[证据](evidence/q02-export-client-store-2026-10-01.md)。继续保存面板/原子替换/目标保护及 GUI，完整任务仍未完成。

## Q02 固定捕获的保护路径（2026-10-01）

有效快照的源图/document 保护路径已随捕获响应传到 Swift，与导出内容分离；公开 include/缺失附件与解包检查通过。本批 Go 全量、相关 race、原生三套、fmt/vet/license通过，见[证据](evidence/q02-export-protected-paths-2026-10-01.md)。目标拒绝与原子写入尚未实现，继续原生保存器和面板，完整任务不关闭。

## Q02 原子文件保存器首批（2026-10-01）

独立原生保存器已实现固定字节、保护路径/inode 检查、同目录临时写/fsync/关闭和 renameat 替换；源/symlink/hardlink/源移动/缺失附件拒绝与实际部分写失败保留旧文件、临时清理通过，见[证据](evidence/q02-atomic-file-export-2026-10-01.md)。保存面板及跨层保护刷新、并发变化/失败与 GUI 仍待验收，完整任务不关闭。

## Q02 保存工作流与首批实际 GUI（2026-10-01）

保护刷新、prepare/save、后台写入取消及独立 CSV/分析保存面板已接入。当前 Go 全量/相关 race、原生四套/app通过；隔离 QA 实际验证保存/取消、产物精确值和 typed NULL/空字符串、重复扩展名修复、公开源副本覆盖拒绝，见[证据](evidence/q02-query-save-workflow-2026-10-01.md)。继续失败输入新附件保护、容器验证器/完整案例、preset、完整资源/GUI及24任务其他门槛，不关闭任务。

## Q02 失败输入中新附件保护（2026-10-01）

构建过程独立保留已解析 document 源声明，失败时也合并到导出保护；真实 Store 保存验证拒绝新附件覆盖且保留旧有效财务快照。当前 Go 全量/四包 race、147 查询/51 会计对照、原生四套/app及静态检查通过，见[证据](evidence/q02-failed-input-documents-2026-10-01.md)。继续容器验证器与完整格式案例、preset、并发保护/资源/GUI以及其余24任务门槛；完整目标未完成。

### 2026-10-02：查询分析容器读取首批

新增有界、严格 JSON 的独立查询容器读取器与 analysisverify 命令；公开精确值恢复、零行和八类篡改测试通过。全量 Go、desktop race、vet/fmt/license 通过；命令验证此前 GUI 的公开保存产物。见[证据](evidence/q02-analysis-reader-2026-10-02.md)。继续多币种/比值/空库存/成本 nullable/截断与预算异常网格，再推进报表容器、保存并发、预设、X01/X02 和完整原生发行验收；没有缩小 24 项范围。

### 2026-10-02：分析样本网格与首批查询预设

容器补齐多币种/空库存/成本标签/零成本/rational/实际截断及压缩预算、CRC、symlink、UTF-8 故障；修复大小写字段别名和误导 Decimal 显示验证。四个单一来源预设贯通 Go 目录、CLI、Swift/C Store 与原生界面；151/151 固定查询差分通过。见[证据](evidence/q02-analysis-grid-presets-2026-10-02.md)。期间预设、报表容器与其他 24 项未验收范围继续保留。

### 2026-10-02：附件解析与打开前身份检查首批

独立 documents Resolver 已实现显式目录根、声明文件相对路径、symlink containment、五种状态与重新校验。公开替换/改向/修改/删除、额外根与 dangling symlink 逃逸测试通过；全量 Go、documents race/vet 通过。见[证据](evidence/x02-document-resolver-2026-10-02.md)。DTO/原生列表/打开/Finder 和权限、alias、系统 race 验收仍待完成，不把 resolver 完成当成 X02 完成。

### 2026-10-02：显式期间预设

同一快照解析阅读期间，完整日期条件显式写入草稿；四预设分别覆盖双边界/单边界，共新增 12 对照案例，当前 163/163 查询差分通过。CLI、desktop 及真实 Swift/C Store 验证捕获期间、选择不执行、保留旧身份与显式运行；非法日期/逆序/ID/文本预算拒绝。见[证据](evidence/q02-period-presets-2026-10-02.md)。期间菜单 GUI 未验收；用户介入的隔离 QA 保留不操作。

### 2026-10-02：全指令与配置只读索引

有效 Snapshot 建立源 kind/配置词法序与日期有效记录索引，公开 fixture 覆盖 20 source kinds，pad/独立 dialect 生成关联、重复身份、插行、深复制和错误重载隔离通过。当前 163 查询/51 会计及 AVERAGE 扩展、全量 Go/五包 race、四套 native 测试通过。见[证据](evidence/x01-directive-index-2026-10-02.md)。下一阶段实现先分页再复制的 typed DTO、journal all/config 原生模式及 block dialect 来源关联；X01 没有标完成。

### 2026-10-02：首版交付收敛及当前阻断

全指令 typed DTO、原生交易/all/config 模式、生成来源、分页与精确元数据已经接通；本轮修复捕获日期和标签显示，当前51会计/163查询、Go全量/四包race、原生四套与app检查通过，FinanceBook只读对照及真实Swift/C通过。窗口清理改为实际window close，最终app/开发ZIP生成并解压公开输入启动。私人GUI文件选择读取阻塞系统open，解压QA还观察到Open禁用；根因尚未确认，C/D完整验收未满足。见[本批证据](evidence/native-first-delivery-2026-10-02.md)与[使用说明](native-first-delivery-2026-10-02.md)。不能将开发产物、测试和私人GUI验收混为一谈。

本批后续复核更正：final QA 的 FinanceBook 原路径读取已完成，实际核心6页、账户详情和计数查询通过，关闭窗口验证通过。系统open等待并非持续阻断，但首次等待根因及完整安装/性能仍未关闭。代码提交74272ff，开发ZIP与SHA256已生成；见同一证据的“后续复核”。用户介入delivery QA后停止操作该实例。

## 2026-10-02：附件原生流程及全指令边界

已接通 document DTO、日记账附件入口、五种状态、打开前重新核验、默认应用打开及 Finder。公开实际操作已验证 TextEdit 内容、Finder 选中和不可用状态禁用。block DSL 生成来源、嵌套检索及深度/文本/附件数量限制已补齐。文件选择使用异步 sheet，公开账本实际重载通过。

具体证据与尚未完成边界见 [验收记录](evidence/native-attachments-and-directives-2026-10-02.md)。下一项仍为查询能力矩阵与报表分析容器，随后完整 GUI/辅助功能/安装验收；不将这批计为全项目完成。
