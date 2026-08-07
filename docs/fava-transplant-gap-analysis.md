# Fava 移植版界面差距分析与 Bridge 计划（2026-08-05）

## 背景与方法

本文档记录**移植版 UI**（`ORANGECOUNT_TRANSPLANTED_UI=1`，工作区未提交状态）与
固定参照 Fava 1.30.12 之间的差距，并给出 bridge 实施计划。它是
[docs/fava-visual-gap-analysis.md](fava-visual-gap-analysis.md)（旧版手写 UI，
已停止投入）的移植版对应物；两份清单的根因不同，不可混用。

比对采用两条线：

1. **源码级清点**：固定镜像（`fava-reference-lock.md` 钉住的 `v1.30.12`
   检出，前端约 180 个单元）对照 `web/src/fava/`（33 个文件）与
   `web/provenance-manifest.json`（14 个 derived 单元），以及
   `internal/web/favaadapter/` 已接线端点。
2. **实测分层比对**：同一个私有生产账本（约 89 科目、约 4900 过账、多币种）
   双端并排运行；全部标准路由做加载态结构比对，shell 与 Income Statement /
   Balance Sheet / Trial Balance / Journal / Account Detail 六处做深度比对。
   英文为视觉基线（ADR-0028），light/dark 均过，zh-CN 仅结构性。

**脱敏纪律**：私有账本观察是临时的（ADR-0027）。本文档不含真实科目全名、
真实金额、真实事件内容、截图或浏览器转储；下文的规模数字和结构描述均已
抽象化。比对期间未对私有账本做任何写入。

## 结论先行

移植已覆盖"**shell + 树形报表 + Journal 行渲染 + 上游样式全集**"这条主线
（树表多货币列、折叠、Journal 条目徽章/展开等语义保留良好），但 Fava 的
**交互层整体缺位**：d3 图表套件、CodeMirror、全部模态、键盘快捷键、排序基建、
通知区、账户自动补全都未移植；六个次要路由降级为一个通用平表组件；数据层
缺 5 个 Fava 形状契约。首屏观感差距最大的三处是：图表（无坐标轴/图例/货币
圆点/层级钻取）、账户详情页过薄、次要路由平表化。

工作区现有约 2400 行未提交改动（Journal 徽章与展开、树表多货币 + Other 列、
图表与 journal 适配器增强），与 T1/H5 部分重叠，属于进行中的修复；本清单对
相应条目做了标注。

## 差距清单

图例：`根因层` = 前端组件缺失 / 适配器契约缺失 / 语义层 / 结构决策；
`WIP` = 已被工作区未提交改动部分覆盖。

### Critical（首屏观感与核心路由）

| # | Manifest | 差距 | Fava 行为 | 移植版现状 | 根因层 |
| --- | --- | --- | --- | --- | --- |
| T1 | R-IS/R-BS/R-TB | 图表系统为手写内联 SVG | d3 套件：坐标轴、刻度、日期标注、tooltip、图例、货币圆点选择器、图表类型切换（Stacked/Single Bars、Line/Area、Treemap/Sunburst/Icicle）、层级钻取 | 粗粒度 SVG；实测 IS/BS/TB/账户页均**无货币圆点**；TB 层级图已有 Treemap/Sunburst/Icicle 三视图切换（`b079d3b`/`b1247c8`）；条形/折线图已补坐标轴、nice 刻度、紧凑数值与日期标注（`d3e5216`）；图例已做成 Fava 式可点选货币/序列开关，点选即隐藏并自适应重算色阶、颜色稳定（`32cceb0`）；条形/折线图已加指针跟随 tooltip（条形按序列+期间、折线按最近期间列全部可见序列，`e8154b6`）；层级图 Treemap/Sunburst/Icicle 每个节点已做成指向该科目详情页的链接（`ab6ca07`）。至此 T1 列举的 Fava 行为均已具备，剩余仅为手写 SVG 与 d3 的实现层差异 | 前端组件缺失（已基本补齐） |
| T2 | R-ACCOUNT | 账户详情页过薄 | 标题含 `(Last entry: date)` 与科目层级面包屑；Balance/Changes 切换；Account Balance/Changes(monthly)/Balances(monthly) 三区块；账户图 + 货币圆点；Account Journal 带徽章 | 面包屑与 Last entry 指示器已实现（`88d90b9`/`eefbf83`），Journal 带 change 列（`ae4eedc`）；上游式 `?r=changes\|balances` 三区块切换 + 区间表已落地（`24ef4ca`：AccountIntervals 按账户子树聚合逐区间变动与累计余额，含安静区间补齐、时间过滤、月度/季度/年度）（限制：上游 IntervalTreeTable 的账户子树逐列形态与账户图货币圆点未实现；FQL 文本过滤不作用于区间聚合行） | 前端 + 适配器（up-to-date 状态契约与账户图仍缺） |
| T3 | G-SHELL/G-FILTERS | shell 控件缺口 | 导航含 `Go to account` 组合框、`+`（Add Entry）、`⬇`（Export）；Time/Account/FQL 为带建议下拉的 combobox，账户模糊自动补全、FQL 解析校验、`r` 重载与变更提示 | `Go to account`（侧栏）与 Time/Account/FQL 三个 AutocompleteInput combobox 已实现；`+` Add Entry 模态已落地（`8e28d53`）；变更提示已由 H7 落地，`r` 手动重载快捷键与 ⟳ 按钮已实现（`4792f73`）；FQL 解析校验与完整语义已落地（`785be13`：#tag/^link 精确匹配、key:"value"、并置 and/逗号 or/`-` 取反、all()/any()、金额比较；非法字符与解析错误在 API 边界 400 并于 shell 错误区展示）；⬇ 导出已落地（`2846847`） | 前端组件 + 适配器 |
| T4 | G-SHELL | 标准导航被 OC 原创项污染 | 导航严格等于 Fava 标准面 | D1/D2 已落地：OC 独有 `Accounts` 项移入标注的 OrangeCount 扩展区；顶栏原创 Language/Theme 下拉已移除，主题入口在 Options→Color scheme，locale 作为 fava option | 结构决策（见决策项 D1/D2，均已实现） |

### High

| # | Manifest | 差距 | Fava 行为 | 移植版现状 | 根因层 |
| --- | --- | --- | --- | --- | --- |
| H1 | R-EDITOR/R-QUERY | CodeMirror 未移植（上游 19 文件 + tree-sitter wasm） | 语法高亮、行号、折叠、补全、snippets、File/Edit 菜单、文件树 | 裸 textarea + Files listbox；无菜单；依赖集已入案（`4eef5a1`，见 H1-deps） | 前端组件缺失 |
| H2 | M-ADD/M-CONTEXT/M-EXPORT/M-DOCUMENT | 模态系统整体缺失（上游 9 文件） | Add Entry 表单、条目 Context（余额/位置）、Export/Download、文档上传 | M-EXPORT 已落地（`2846847`）：modals 目录建立，Export 模态（#export hash 驱动）+ download-journal 端点（按过滤切取源码的保源导出）；M-CONTEXT 已落地（`68ddcfa`：entry-context 路由 + 只读源码切片模态）；M-ADD 已落地（`8e28d53`：`#add-transaction` 模态，Transaction/Balance/Note 三型切换保留日期、postings 行增删、continue 复选框持久化；私有 `add-entries` POST 路由严格序列化校验 + 原子写入/备份/重新验证，失败回滚并保留诊断）；M-DOCUMENT 已落地（`339b63e`：上游式拖放上传模态——账户页标题 droptarget 触发，多文件 + 日期前缀改名输入、文档目录选择（来自 serve --document-root 配置根）、账户 datalist；私有 POST `document` 路由同源校验 + 账户/目录校验 + basename 净化 + 拒绝覆盖，上传落入 `<根>/<账户分层>/<文件名>` 并由 `/documents/` 路由回供）（限制：上游 uri-list 链接拖放 attach 流程与 entry hash 元数据插入未实现；droptarget 已补齐 journal 行与账户树单元格（`44e6476`，见 H2-droptarget-extend），与上游账户页标题/documents 账户表同构） | 前端 + 适配器 |
| H3 | G-KEYBOARD | 全局键盘快捷键缺失 | `g-*` 路由跳转、`t/f/a/d/s`、`?` 快捷键提示 | 已实现：`g-*` 路由跳转、`f t/f a/f f` 筛选快捷键、`?` 快捷键 tooltip（冒烟验证 19 条提示，含 `r` 重载）、`r` 手动重载（`4792f73`）；上游其余单键快捷键未登记为缺口 | 已完成 |
| H4 | R-HOLD-*/R-COMMODITIES/R-EVENTS/R-STATISTICS/R-DOCUMENTS | 六路由降级为通用平表 | Holdings 四子页签与成本分组；Commodities 价格折线图（ChartSwitcher+LineChart）+ base/quote 分组表；Events 按事件类型侧栏分组；Statistics 指令计数 + Postings-per-Account + 活动图；Documents 账户树 + 内嵌预览 | 复核修正（2026-08-07 冒烟）：六路由专用组件均已在案并按上游形态渲染——Holdings 六页签（上游四页签齐备，by_cost_currency 补于 `2b8d370`）+ 列名可读化（`HoldingsReport`）；Commodities 按 base/quote 分组价格表（`CommoditiesReport`+`PriceTable`）；Events 按类型分组 + `Event: <type>` 标题 + 可排序 Date/Description 表（冒烟双类型验证，默认 date desc）；Statistics 双区块（Postings-per-Account + Entries-per-Type 可排序）；Documents 表格 + 内嵌预览（`92ccb40`）+ 账户树侧栏（分层/计数/折叠/点击筛选，`30ca6f3`）+ 移动/改名模态（F2 或拖拽触发，`38f2618`）+ Update Activity 表（每账户最近条目/余额/上下文模态入口，`a66cf9d`）+ Events 散点图（无 d3 依赖 SVG，语义同上游，`4e88925`）+ Commodities 价格折线图（无 d3 依赖 SVG + 每商品对切换/显示开关，`d9b77d3`）。H4 全部落地，无契约级余项（上游 1.30.12 无每商品详情页，此前"每商品详情页"描述为复核失误，已更正） | 已完成 |
| H5 | R-JOURNAL | Journal 交互层不完整 | 全量条目类型徽章（含 Custom/B/Metadata/Postings）、排序与列菜单、点击条目→Context、URL 同步筛选、拖拽上传文档 | 核心徽章组与展开已现；点击条目→Context 已落地（`68ddcfa`：行尾 ⋮ 链接 + `#context-<hash>` 模态 + entry-context 私有路由，位置派生 entry_hash，只读源码切片；限制：before/after 余额与 CodeMirror 可编辑切片属 H1）；表头排序已落地（`f7d57ab`：Date/F/Payee-Narration 三列，同上游 `[列,向]` localStorage 持久化、默认 date desc、data-order 箭头与切换语义，复用已移植 Sorter）。拖拽上传文档已随 H2 M-DOCUMENT 落地（`339b63e`：账户页标题 droptarget；journal 行与账户树单元格的 droptarget 后经 H2-droptarget-extend 补齐，`44e6476`）。徽章覆盖复核已落地（`4bafc76`：适配器为全部指令类型投影条目元数据——此前仅 transaction——journal 行渲染条目与过账级 metadata-indicator 徽章（key[:2]，title `key: value`）、过账 flag 类（flag_to_type）、过账元数据 dl、linked/discovered 行类与 D/L 芯片（`d d`/`d l`，show-document 的子级，默认激活集与上游 default_journal_show 同构）；冒烟验证 au/re/cl 徽章、linked/discovered 行类、过账 dl 展开与 Metadata 芯片切换）。限制：B(budget) 芯片未加——OC 无上游 budget custom 指令形态、适配器不产生 budget 标记；custom 值按 dtype 渲染、balance diff_amount pending 展示、document→statement 元数据附件未实现 | 前端组件 |
| H6 | R-OPTIONS | Options 页不完整 | Color scheme（System/Dark/Light）单选组 + Fava options 表（带 help 链接）+ Beancount options 表 | 已实现：UtilityReport 含 Color scheme 单选组 + Fava options 表 + Beancount options 表；顶栏原创主题下拉已移除（D2） | 已完成（UtilityReport + `/__orangecount/fava/options` 契约） |
| H7 | M-NOTIFY | 通知区缺失 | 文件变更/保存结果 toast，带点击重载 | 已实现：notifications 模块早已在案（bootstrap/报告错误走 notify_err），本步补齐可感通知——文件变更 warning toast（点击再刷一次，5s 自动消失，冒烟验证文案与类名）与编辑器 Save 结果 toast（成功/拒绝/失败三态），`a5251c8` | 已完成 |

### Medium

| # | Manifest | 差距 | 说明 |
| --- | --- | --- | --- |
| M1 | R-QUERY | Query 页不完整 | 保存查询已落地（`1abfaf0`）：账本 `query` 指令投影为侧栏 Query 项子菜单（同名截断规则同上游），页内点选即回显并重跑；结果排序冒烟确认 GenericReport 列排序可用（含数值列方向切换）——原"无结果排序"判断有误；查询图表已落地（`78d4233`，见 M1-query-chart）；余项仅 BQL 编辑器为裸 textarea（依赖 H1） |
| M2 | R-IMPORT | Import 为 OC 原创表单 | 上传块已对齐（`ea13c07`，见 M2-import-upload）：上游 ImportFileUpload 形态的文件选择器落地（本地读入导入缓冲区、按扩展名推断 adapter）；余项：服务端 import 目录文件列表与逐条目 extract/review 弹窗（上游依赖 Python importer 生态，OC 无 import 目录，待方案决策）；Source path/Adapter/Target + 粘贴缓冲区保留为 OC 扩展 |
| M3 | R-HELP | ~~Help 无页面索引~~（已完成，`f1a01f0`） | `/help` 现渲染子页索引，`/help/<id>` 渲染单节页面 + 返回链接；Options 页标题链接 `/help/options`（限制：子页集合为 OC 自有 8 节，非上游 Index/Syntax/Budgets 全集） |
| M4 | 跨路由 | ~~排序基建缺失~~（已完成，`62de047`） | `sort/index.ts`（Sorter/SortColumn 契约，同上游点击语义，无 d3 依赖）+ `SortHeader.svelte`（legacy 模式，含 aria-sort 与箭头提示）已落地；events/commodities/documents/statistics/options 表头可排序，冒烟验证方向切换与列切换重排（限制：holdings 与上游一致不可排序；Query 结果排序经 GenericReport 实际可用，`1abfaf0` 冒烟确认） |
| M5 | 跨路由 | ~~三份 CSS 未移植~~（已完成） | `editor.css`、`help.css`、`notifications.css` 均已移植，main.ts 引入 11/11 |
| M6 | G-FILTERS | ~~URL 状态对齐未逐项验证~~（已核对，`d68e5e9`/`36fb31b`） | 全路由深链、筛选回显、history back/forward、reload 逐项冒烟通过；核对发现并修复三处：`time=2025-Q2` 季度语法被拒（新增 Filters.TimeBegin/End 半开区间）、`/editor?path=` 与 `/query?query_string=` 未回显（限制：diagnostics 仍为 JSON 兜底视图；Query 页图表彼时仍属 M1，后经 M1-query-chart 落地，保存查询与排序已落地） |

### Low

| # | Manifest | 差距 | 说明 |
| --- | --- | --- | --- |
| L1 | R-ACCOUNT | ~~账户页标题无层级面包屑~~（已完成，`88d90b9`；Last entry 指示器 `eefbf83`） | 标题现为祖先面包屑 + Last entry 指示器；指示器跟随当前过滤，无条目上下文链接（限制） |
| L2 | G-LOCALE | i18n 为静态 en/zh-CN 字典 | 上游为 gettext 目录；用户可见行为等价，建议登记为实现性偏差（D4） |
| L3 | R-HOLDINGS | ~~Holdings 页签集合与上游不一致~~（已完成，`2b8d370`） | 上游 by_cost_currency 已补齐（后端 HoldingsAggregate 新增 cost_currency 分组，前端页签/路由/文案/CSV 全链路），OC 扩展页签 by_root_account/by_commodity 保留为实现性偏差。限制：by_cost_currency 组内 units 跨不同持仓货币直接相加为单一数值（上游按货币逐行展示库存），book_value 仍按成本货币单一性规则输出 |
| L4 | R-ERRORS | serve 拒绝加载含 error 级诊断的账本（偏差登记） | Fava 带错服务并在 /errors 展示全部诊断；OC `serve` 在 main.go:181 检测到 error 即退出，/errors 页面只能展示 warning（`b902d7d`）。登记为实现性偏差：若要对齐 Fava，需 owner 批准放宽启动门禁 |
| L5 | M-NOTIFY | 文件变更提示与自动重载并存（偏差登记） | 上游 auto-reload 默认开启时静默重载、不弹 toast；本实现重载与 warning toast 并存以保证可感知（`a5251c8`）。登记为实现性偏差：若要对齐上游静默行为，移除变更 toast 即可 |

### 适配器契约缺口（数据层）

已接线：`changed`、`ledger_data`、`metadata`、`options`、`help`、
`diagnostics`、`editor`(读)、`import`(adapters/files/content)、`journal`、
三大树报表、泛型 `reports/*`、BeanQuery、`download-journal`
（过滤后条目按源码 span 切取的 Beancount 导出，`2846847`）、
`entry-context`（位置派生 entry_hash → 条目投影 + 只读源码切片，`68ddcfa`）、
`add-entries`（新增条目 POST：严格序列化校验 + 原子写入/备份/重新验证，`8e28d53`）、
`document`（文档上传 POST，`339b63e`）、`move-document`（文档移动/改名
POST，`38f2618`）。

尚未支撑的 Fava 数据契约：

- entry context 的 before/after 余额（只读切片已接线）
- query shell 补全元数据（保存查询已经 user_queries 落地）
- editor 保存与 import commit 目前走 `/api/v1/*` 非 fava 适配器契约，
  形态是否并入 fava-shaped 适配器需在 Wave 6 前评估。

## Bridge 计划

排序策略（已确认）：差距映射到波次；跨路由共享根因作为共享基础提前；
其余严格按波次推进，波次内按差距优先级排序（ADR-0032 深度优先纪律不变）。

### 任务 0 — WIP 收尾（先决）

将当前 ~2400 行未提交改动按差距归属拆分提交（Journal 徽章/展开、树表多
货币 + Other 列、图表与 journal 适配器），提交后复核 T1/H5 并勾销已修复
项。**在完成前不再叠加新的界面改动**，保证"一次修复对一条差距"可追溯。

> **进度（2026-08-05 晚）**：任务 0 已完成——WIP 拆分为两个提交
> （`265f351` 文档、`2c6590a` 移植实现）。随后又落地：D1（Accounts 移入
> 标注扩展区，`bbd232b`）、D2（主题/语言入口移入 Options，`bbd232b`）、
> D4（确认无用户可见差异、无需登记）、M5（三份页面样式表补齐，12/12，
> `201f9e9`）、L1（账户标题改为可点击的层级面包屑，`25dc9ea`）、
> S6 基础（移植 keyboard-shortcuts 并绑定上游 g 系导航快捷键与 ? 提示，
> `8e4105d`）、H7 基础（移植 notifications/errors 帮助函数，适配失败
> 弹 Fava 式错误提示，`056d787`）、S2 第一步（移植 AutocompleteInput
> 与 fuzzy 帮助函数，Header 账户过滤改为模糊建议组合框，`052a128`）、
> S2 完成（Time/FQL 过滤同样改为组合框：年份、#tag/^link/payee 建议，
> `1f919fb`）、S6 日志过滤（journal 过滤芯片绑定上游 s/t/m/p 快捷键与
> 超类级联，`c78baa1`）、T2 第一步（账户页修复日志解析、渲染余额图表、
> 遵循 time/filter 参数，`4d50b49`）、S3（通用报表表格列可排序：点击
> 循环 升序→降序→还原，数值列按金额比较、文本列按 localeCompare，
> 带 aria-sort 与 ▲/▼ 指示，浏览器冒烟验证三态，`029b687`）、
> T3 第一步（侧栏移植 Go to account 模糊组合框，`g a` 快捷键聚焦、
> 选中即跳转并清空，冒烟验证，`6dc8d38`）、H6（Options 页对齐上游：
> ModeSwitch 式配色方案按钮、Fava options 表带 help 链接、两表按键
> 排序、zh-CN 词条补齐，冒烟验证主题/语言切换，`4249322`）、
> H4 第一步（Holdings 四聚合页签落地：后端新增 HoldingsAggregate
> 按 account/currency/root/commodity 分组求和 units 与 book value，
> 不跨成本货币合并；前端 HoldingsReport 提供上游 headerline 式页签与
> 可读列名、CSV 导出带 aggregation 参数，含单测与四页签冒烟，
> `3c9fef1`。页签集合保留 OC 扩展的 by_root_account/by_commodity、
> 缺上游 by_cost_currency，已登记为实现性偏差 L3）、
> H4 Events（事件页按 type 分组渲染，组内按日期倒序，空态文案，
> 冒烟验证唯一 event 指令，`696bf81`）、
> H4 Statistics 第一步（后端新增 PostingsPerAccount 与 statistics
> 专用载荷 entries_by_type + postings_per_account；前端两节：
> Postings per Account 表按数量倒序、Entries per Type 带 Total 脚注，
> 冒烟验证合计 420，`f6474a5`。上游 Update Activity 节依赖缺失的
> account_details 契约（last_entry/uptodate_status/balances，即 T2
> 缺口），标题 Query 链接依赖上游 postings BQL，本步均未实现；
> 载荷暂不应用全局 time/filter，作为限制记录）、
> H4 Documents（文档页改为 Date/Account/Name 表格：basename 去日期
> 前缀、按日期倒序、行 title 保留完整路径，冒烟验证 3 条文档，
> `4bb03e3`。上游三栏布局的账户树与文档预览依赖文档文件服务端点，
> 本步未实现，作为限制记录）、
> 账户 Journal change 列（journal 适配器在账户过滤下为每条 transaction
> 计算账户内过账的按货币求和 change；账户页表头第三列由 Price 换为
> Change 并渲染该值，全局 journal 保持 Price，含适配器单测与双端冒烟，
> `ae4eedc`）、
> H4 Commodities（商品页按 base/quote 对分组渲染价格表，组内按日期
> 倒序，冒烟验证 9 个货币对，`132ad60`。上游每对的 d3 折线图与
> ChartSwitcher 属 S1 范围，本步未实现，作为限制记录）、
> H4 Errors（错误页改为 File/Line/Error 三列：默认按文件倒序、三态
> 可排序、severity 行样式、代码前缀、源文件链接、"No errors." 空态，
> 警告夹具与干净夹具双端冒烟，`b902d7d`。发现 serve 拒绝加载含 error
> 级诊断的账本（main.go:181），故 /errors 实际只能展示 warning 级
> 诊断——与 Fava 带错服务行为不一致，登记为实现性偏差 L4；上游消息内
> 账户名自动链接本步未实现，作为限制记录）、
> L1（账户页标题改为祖先面包屑：每段显示 leaf 并链接到对应层级账户，
> title 保留全名，冒烟验证三级账户渲染与点击导航，`88d90b9`）、
> T2 标题部分（面包屑后追加 "(Last entry: date)" 指示器：取已加载
> journal 最新条目日期，上游小字号样式，冒烟验证 balance 指令日期，
> `eefbf83`。上游取全局 account_details 且链接到条目上下文；本实现
> 跟随当前 time/filter 过滤且无上下文链接，作为限制记录）、
> S1 Icicle（层级图增加 Treemap/Icicle 视图切换：Icicle 按层级逐行
> 铺排矩形、宽度按值占比、聚合行与其子行同框可见，切换按钮采用上游
> ChartSwitcher 的居中弱化样式，冒烟验证 trial_balance 双视图切换
> （Treemap 86 叶块 / Icicle 98 块三层），`b079d3b`。上游第三视图
> Sunburst 与 d3 钻取交互属 S1 剩余范围，作为限制记录）、
> S1 Sunburst（层级图补齐第三视图：极坐标分区，根账户居内环、逐层
> 成环，扇区角度按值占比，满圆扇区留发丝缺口防退化，冒烟验证
> trial_balance 三视图循环切换（Sunburst 98 扇区），`b1247c8`。
> 上游 d3 的 tooltip、钻取缩放与货币圆点仍属 S1 剩余范围，作为限制
> 记录）、
> 清单复核（逐行比对在案实现：H3 快捷键（含 `?` tooltip，冒烟 18 条）、
> H6 Options 三表、M5 三份 CSS、T3 Go-to-account 与三个筛选 combobox、
> T4/D1/D2 导航与主题入口均已落地，相应现状列与 D1/D2 标记更新，
> `e631330`/本次；未改任何代码）、
> H7（轮询发现文件变更时除照旧重载外，另弹 warning toast
> "File change detected. Click to reload."（点击再刷一次、5s 自动消失、
> 跟随 locale），编辑器 Save 成功/拒绝/失败分别弹 info/warning/error
> toast；冒烟以临时夹具触发 watcher 与保存双路径验证，`a5251c8`。
> 上游 auto-reload 默认为静默重载、toast 仅在关闭 auto-reload 时出现；
> 本实现选择重载与提示并存以保证可感知，登记为实现性偏差 L5）、
> M3（`/help` 渲染 8 节子页索引，`/help/<id>` 渲染单节页面 +
> "‹ Help" 返回链接；router/state/App/ReportOutlet/UtilityReport
> 贯通 helpPage，server.go 补 options 节并使 Options 页标题链接
> `/help/options`；冒烟验证索引链接全集、editor 子页与 SPA 内点击
> 跳转，`f1a01f0`。子页集合为 OC 自有 8 节，非上游 Index/Syntax/
> Budgets 全集，作为限制记录）、
> M4（新增 `sort/index.ts` 与 `SortHeader.svelte`（legacy 模式、
> aria-sort + 箭头提示），events/commodities/documents/statistics/
> options 的表头全部可排序；events 与 commodities 按上游拆为每组一张
> 独立排序表，documents 保留账户列，冒烟验证方向切换、列切换与实际
> 重排，`62de047`。Query 结果排序仍属 M1 范围，作为限制记录）、
> M6（全路由 URL 状态逐项冒烟：深链 income/journal/account/holdings/
> query/editor/import/options/help/source/errors/diagnostics、time 与
> query 参数回显、history back/forward、reload 均通过；核对暴露三处
> 缺口并修复——`time=2025-Q2` 季度语法新增 Filters.TimeBegin/End
> 半开区间支持（`d68e5e9`，含单元与集成测试），`/editor?path=`、
> `/query?query_string=` 深链回显（`36fb31b`）。diagnostics 仍为 JSON
> 兜底视图，作为限制记录）、
> T1-axes（条形/折线图补坐标轴、nice 刻度、紧凑数值标注与稀疏日期
> 刻度；层级三视图几何不变；冒烟验证 Income/Balance/Trial 均渲染
> 网格与刻度，`d3e5216`。悬停 tooltip、货币圆点选择器、层级钻取
> 仍缺，作为限制记录）、
> T1-legend（图例由静态色块改为 Fava 式可点选货币/序列按钮，点选
> 即在图中隐藏该序列并自适应重算 y 轴范围，再点恢复；颜色按原序列
> 位置稳定不重排，隐藏态加删除线与灰度。冒烟于 Balance Sheet 7 序列
> 折线图逐项验证——隐藏 EUR 后 7→6 条路径且 USD/GBP 颜色保持不变，
> `32cceb0`。上游为 d3 + store，本实现为手写 toggle，行为等价；
> 层级钻取仍缺，作为限制记录）、
> T1-tooltip（条形/折线图新增指针跟随 tooltip：条形悬停显示「序列 ·
> 期间: 金额 货币」，折线按指针 x 位置二分定位最近期间并逐行列出全部
> 可见序列取值；移出即隐藏。层级图保留原生 `<title>`。冒烟于 Income
> Statement 条形图与 Balance Sheet 7 序列折线图分别验证出现/内容与
> 隐藏，`e8154b6`。上游为 d3 浮层，本实现为单一定位 div，行为等价）、
> T1-drilldown（层级图「钻取」按 Fava 语义实现为逐节点账户链接：
> Treemap/Sunburst/Icicle 每个节点包一层 `<a href=/account/…>`，点击
> 进入该科目详情页，与 tree-table 账户链接约定一致。冒烟验证三视图
> 均有链接（86/98/98）且点击 Treemap 磁贴成功跳至 Equity:Opening-
> Balances 账户页，`ab6ca07`。T1 列举的 Fava 行为至此全部具备）、
> T3-reload（Header 新增 ⟳ 手动重载按钮并绑定 `r` 快捷键，点击/按键
> 均触发 bootstrap 重新拉取账本；因 OC 已在文件变更时自动重载（L5），
> 该按钮不做 has_changes 门控而常驻，登记为实现性偏差。冒烟验证按钮
> data-key=r、按 `r` 触发重载且页面保持完好、在可编辑元素内 `r` 被
> 正确抑制、`?` 提示增至 19 条并含 `r`，`4792f73`）。
> T3-FQL（新增 Fava 式过滤查询 `internal/report/fql.go`：#tag/^link 精确
> 匹配、裸/引号串对 narration/payee/comment 做大小写不敏感正则、
> key:"value" 可达元数据与行内列名、并置为 and、逗号为 or、`-` 取反、
> all()/any() 量化 postings、比较符按金额绝对值匹配；journal 条目按完整
> 语义匹配，表行按行粒度匹配——派生行无法表达 tags/postings，作为限制
> 登记；非法字符与解析错误在 API 边界返回 400，shell 错误区展示原文，
> 冒烟验证 #evidence 过滤出 2 条 document、`payee:` 报错展示、带 filter
> 的 IS 返回 200，`785be13`）。
> S5-export（侧栏 Import 行新增 ⬇ secondary 链接 → #export hash 模态 →
> `/__orangecount/fava/download-journal` 下载过滤后条目为 Beancount 源码：
> ExportEntries 按条目源码 span 切取原文，保源导出；journal 过滤谓词重构
> 为 journalFilterState 与 journal 共用；模态改写自上游
> modals/Export.svelte（provenance 登记，上游 hash 按重建计算已注明）。
> 冒烟验证 ⬇ 打开模态且下载链接携带当前 filter、背景关闭清除 hash 页面
> 完好、curl 验证 #evidence 导出恰为两条 document 源码行，`2846847`）。
> M1-saved（账本 `query` 指令经 ledger_data 的 user_queries 投影至前端：
> adapter-client 边界补齐字段、shell state 接纳、侧栏 Query 项下渲染子
> 菜单（上游同名截断规则），点选在页内即回显编辑器并重跑——
> QueryReport 新增路由 query_string 反应式同步。冒烟验证子菜单渲染
> saved-overview、点选后 URL/编辑器/188 行结果齐备、GenericReport 列
> 排序含数值列方向切换；原 M1"无结果排序"判断经冒烟证伪，`1abfaf0`）。
> H5-context（journal 条目投影新增位置派生 entry_hash（file+span 的
> sha256）；行尾 ⋮ 链接打开 `#context-<hash>` 模态，模态经新私有路由
> entry-context 解析条目投影与只读源码切片（entrySourceBlock 复用），
> 未知 hash 在模态内展示适配器错误。改写自上游 modals/Context.svelte
> （provenance 登记，上游 hash 按重建计算已注明）；before/after 余额与
> CodeMirror 可编辑切片属 H1，作为限制记录。冒烟验证 400 条 ⋮ 链接、
> close 指令切片逐字回显、交易切片含 postings、Escape/背景关闭清除
> hash 页面完好、未知 hash 报错，`68ddcfa`）。
> H5-sort（journal 表头 Date/F/Payee-Narration 变为排序按钮：复用已移
> 植 Sorter（DateColumn/StringColumn），`[列,向]` 元组持久化于
> localStorage journal-sort-order 且取值校验同上游，默认 date desc，
> data-order 箭头沿用已移植 base.css，切列回 asc、同列翻向的点击循环
> 与上游一致；账户页 journal 同享排序。冒烟验证默认序、narration 升/
> 降、flag 列、reload 后持久化保持、账户页无回归，`f7d57ab`）。
> H2-add（侧栏 `+` 打开 `#add-transaction` 模态：Transaction/Balance/
> Note 三型切换保留日期，交易表单含 flag/payee/narration/tags/links 与
> postings 行增删，balance/note 表单按型收敛；提交走新私有 POST 路由
> `add-entries`：SerializeNewEntries 严格校验日期/账户/货币/金额/标签
> 后序列化，复用 replaceGraphFile 原子写入 + .bak 备份 + 全图重新验证，
> 失败回滚并返回诊断（422）；continue 复选框持久化同上游
> add-entry-continue，保存后保留日期重置表单。冒烟在临时夹具副本验证
> 交易/备注追加、journal 即时刷新、非法账户 400 内联报错、重新验证
> 失败回滚还原，`8e28d53`）。
> H2-document（上游式文档上传模态：账户页标题为 `.droptarget`（拖入时
> `.dragover` 虚线高亮），放下文件即开模态并预填账户；多文件 + 逐文件
> 改名输入（无日期前缀自动补 `YYYY-MM-DD `，同上游）、文档目录下拉
> （serve --document-root 配置根，经 bootstrap document_roots 下发）、
> 账户 datalist；提交走新私有 POST 路由 `document`：同源校验、账户存在
> 校验、目录白名单、basename 净化 + root 内包含复核、O_EXCL 拒绝覆盖，
> 上传落入 `<根>/<账户分层>/<文件名>` 并由既有 `/documents/` 路由回供。
> 冒烟在临时夹具副本验证 curl 上传/回供/重复 409 与浏览器拖放→改名
> 预填→提交 toast→文件落盘、非法账户内联报错、Escape 关闭，`339b63e`。
> 限制：uri-list 链接拖放 attach 与 entry hash 元数据插入未实现）。
> T2-intervals（账户详情页上游式三区块切换：Balance / Changes (interval)
> / Balances (interval)，`?r=changes|balances` 走新报告函数
> AccountIntervals——按账户子树聚合逐区间变动与累计余额，月度/季度/
> 年度区间、安静区间补齐、时间过滤；`r` 加入 router QUERY_KEYS 白名单
> 后切换在 history 导航间保持；顺带修复 AccountReport 重复渲染
> balance.chart 的既有问题。冒烟验证 changes/balances/默认视图、季度
> 标签切换、time=2024/2025 过滤与 journal 无回归，`24ef4ca`）。
> H5-badges（journal 徽章覆盖复核收尾：适配器 `projectJournalEntry` 为
> balance/open/close/note/document/pad/query/custom 全部投影条目元数据
> ——此前仅 transaction；JournalReport 行渲染条目与过账级
> metadata-indicator 徽章（key[:2]，title `key: value`，CSS 早已在案）、
> 过账 flag 类（flag_to_type：*→cleared、!→pending、其余非空→other）、
> 过账元数据 dl、linked/discovered 行类与 D/L 芯片（`d d`/`d l`，为
> show-document 子级，Document 芯片连带翻转；默认激活集补入
> show-discovered/show-linked，与上游 default_journal_show 同构）。
> 冒烟验证元数据徽章（au/re/cl 及 title）、linked/discovered 行类、
> 过账 dl 随指示器点击展开、Metadata 芯片切换显示，`4bafc76`。限制：
> B(budget) 芯片未加（OC 无上游 budget custom 形态、适配器不产生该
> 标记）；custom 值 dtype 渲染、balance diff_amount pending、document
> →statement 元数据附件未实现）。
> H4-review（冒烟复核六路由专用组件：Holdings 五页签、Commodities
> base/quote 分组、Events 类型分组、Statistics 双区块、Documents 表格
> 均在案；H4 行现状列改写为已验证形态并列出契约级余项——散点图/
> 商品详情页/文档预览/活动图/成本分组筛选。Events 冒烟以双类型夹具
> 验证分组标题、date desc 默认序与排序表头）。
> H4-documents-preview（Documents 行点击选中（selected/hover 高亮）+
> 预览窗格落地：pdf `<object>`、图片 `<img>`、纯文本（csv/json/qfx/
> txt/xml）fetch 后 `<pre>` 只读、html/htm 沙箱 iframe、其余按上游文案
> 提示未实现；URL 走既有 `/documents/<分段编码名>`。改写自上游
> DocumentPreview.svelte（provenance 登记，哈希直接对 pinned 参照计算）；
> 上游纯文本用 CodeMirror 只读编辑器，属 H1，登记为限制。账户树侧栏
> （stratifyAccounts）与移动/改名模态（move_document 写路径）未实现，
> 登记为限制。冒烟验证 txt 文本回显、svg 图片加载、zip 未实现提示与
> 选中行高亮，`92ccb40`）。
> H4-holdings-cost-currency（Holdings 补上游第四页签 by_cost_currency：
> 后端 HoldingsAggregate 新增 cost_currency 分组（cost_currency/units/
> book_value，沿用成本货币单一性规则），既有 holdings 适配器路由按
> aggregation 参数透传，前端 router/ReportOutlet/HoldingsReport 页签与
> en/zh 文案接入，CSV 导出同步覆盖；冒烟验证页签激活态、单组 USD 行与
> CSV 内容。限制：组内 units 跨持仓货币直接相加（上游按货币分行展示
> 库存），登记于 L3，`2b8d370`）。
> H4-documents-account-tree（Documents 页补齐上游账户树侧栏：按文档
> 账户 stratify 出层级树（隐式中间账户自动插入）、每账户计数徽标、
> ▾/▸ 折叠、点击叶子按子树前缀筛选表格（再点清空），选中行高亮；
> 布局改为上游 1fr 2fr（带预览时 1fr 2fr 3fr）三栏。改写自上游
> Documents.svelte/Accounts.svelte（provenance 登记，哈希对 pinned 参照
> 计算）。移动/改名模态与拖拽移动依赖 move_document 写契约，未实现，
> 登记为限制。冒烟验证树渲染（含隐式 Assets 根）、Cash 子树筛选
> 3→2 行、折叠隐藏子节点、清空还原、预览三栏，`30ca6f3`）。
> H4-documents-move（Documents 移动/改名闭环：私有 move-document 写路由
> （同源校验 + 账户校验 + basename 净化 + 目标根内防穿越 + 拒绝覆盖，
> 登记入 ContractRegistry），前端 DocumentMoveModal（F2 选中行或表格行
> 文件名拖拽到账户树节点触发，datalist 账户输入 + 新名输入 + 错误回显），
> 移动后按上游 router.reload 语义重取 documents。冒烟验证 F2→填写→提交
> 文件实移（documents/doc-002.txt → Assets/Receivables/Client01/
> moved-002.txt）、模态关闭与错误路径回显（根布局错误时 400 文案入框）。
> 限制：移动只动物理文件不改账本 document 指令（同上游，指令路径随之
> 失效）；拖拽触发路径与 F2 同源共用模态，拖拽本身未单独冒烟，`38f2618`）。
> H4-statistics-activity（Statistics 补上游 Update Activity 区块：后端
> favaadapter.UpdateActivity 按 journal 顺序求每 Assets/Liabilities 账户
> 最近条目（日期 + entryHash）与当前余额，statistics 复合载荷新增
> update_activity；前端 StatisticsReport 新增第三区块——账户列链到账户
> 页、日期列链到 `#context-<hash>` 模态、余额列按货币多行（UnsortedColumn，
> 同上游），en/zh 文案 updateActivity/updateActivityLastEntry 新键（避开
> 既有 lastEntry 冒号形态与 zh balance=余额断言歧义）。Go 单测覆盖账户
> 过滤/排序/最近条目语义/余额。冒烟验证 62 行渲染、账户与 context 链接、
> 点击日期打开上下文模态并显示对应 document 指令。限制：上游
> uptodate_status/AccountIndicator 指示列未移植（依赖 fava option 驱动的
> account_details 契约）；余额取账本余额而非上游估值库存，`a66cf9d`）。
> H4-chart-scope（对照 pinned 参照复核并收敛剩余图表缺口：上游 1.30.12
> Commodities 无每商品详情页——此前 H4 行"每商品详情页"描述为复核失误，
> 已更正为仅 ChartSwitcher+LineChart；Events 余项为 ChartSwitcher+
> ScatterPlot。两者与 M1 查询图表共享同一图表基建：上游 charts/ 目录
> 24 文件（Chart/Axis/ChartSwitcher/ChartLegend/HierarchyContainer/
> ModeSwitch/SelectCombobox/conversion 控件 + tooltip/context/helpers/
> bar/line/scatterplot/hierarchy/query-charts），Svelte 5 语法（$props/
> $derived/@attach）+ d3 模块（d3-array/axis/quadtree/scale/color，
> line/bar 另需 d3-shape/time-format 系）；OC web/ 目前刻意保持最小依赖
> （字体 + esbuild + svelte，无任何图表库）。移植该基建需先批准引入 d3
> 依赖，再做 Svelte 5→legacy 改写与适配器图表数据契约（commodities
> charts/scatterplot/query chart），登记为 H4/M1 唯一剩余图表工作项。
> 注：H4-events-scatter 落地后确认壳内无依赖 SVG 惯例可等价承载图表
> 语义，剩余图表项不必引入 d3；随后 H4-commodities-line 亦按该惯例落地
> （`d9b77d3`），H4 图表项全部完成，仅剩 M1 查询图表）。
> H4-events-scatter（Events 散点图落地：采用壳内既有无依赖 SVG 图表
> 惯例（ReportChart 同风格），不引入 d3——新增 charts/ScatterPlot.svelte
> 复刻上游语义：时间×事件类型 point scale（padding 1、首类型居底）、
> 每事件一点、按类型着色（HSL 等亮度环近似上游 HCL 轮）、指针最近点
> tooltip（描述 + 日期）、未来日期降饱和、稀疏日期/类型轴刻度；
> EventsReport 顶部加上游 Chart.svelte/ChartSwitcher 形态的 ▼/◀ 显示
> 开关与图表标签行（数据直接取表格载荷的 date/type/value，无需后端
> 改动）。provenance 登记 original（语义镜像、非代码派生）。冒烟以
> 双类型六事件夹具验证六点、两类型轴标、tooltip 文案（Berlin
> 2023-02-10）、开关隐藏/还原与 ◀ 字符形变。限制：无 quadtree——最近
> 点为线性扫描（事件量级无碍）；ChartSwitcher 简化为单图标签 + 本地
> 开关态，未接 URL charts=false 参数、c/C 快捷键与 lastActiveChartName
> 持久化；色板为 HSL 近似而非 d3-color HCL，`4e88925`）。
> H4-commodities-line（Commodities 折线图落地，H4 收尾：新增
> charts/LineChart.svelte（无依赖 SVG：时间线性 x、上游式 padExtent
> 值域不强制含零、nice ticks、最近点 tooltip `1 BASE = x QUOTE` + 日期）；
> CommoditiesReport 顶部加上游 ChartSwitcher 形态——▼/◀ 显示开关 +
> 每 base/quote 对标签按钮行（点击切换活动图，选中态高亮，分隔线同
> 上游），图表数据取表格载荷 date/amount（display 解析，支持分数），
> 无需后端改动。provenance 登记 original。冒烟验证 9 商品对按钮与
> 折线渲染、EUR/USD 切换选中态、tooltip 文案（1 EUR = 1.1 USD
> 2025-06-28）；限制：显示开关在 commodities 页未单独冒烟（与 events
> 页同构代码，后者已验证）；line/area 模式切换（lineChartMode）与
> lastActiveChartName 跨导航持久化未实现，`d9b77d3`）。
> M1-query-chart（Query 页查询图表落地，M1 图表项收尾：上游
> getQueryChart 仅对"恰两列"结果出图——str+Inventory→层级图、
> date+Inventory→折线图；OC 查询结果无 dtype/Inventory 元数据，
> QueryReport 改为值嗅探：第一列全为 ISO 日期字符串且第二列可解析为
> 数值（PresentedDecimal/number/数值字符串/同形数组求和）时，复用
> H4 的 charts/LineChart.svelte 于结果表上方出图，配同款 ▼/◀ 显示
> 开关，tooltip 取默认 `日期: display`。冒烟验证默认 account/balance
> 查询无图（188 行表）、`SELECT date, amount FROM prices ORDER BY
> date` 出 9 点折线（两次独立会话）；限制：str+金额层级分支不可行
> （无 Inventory dtype，登记为实现性偏差）；图表开关与 tooltip 在
> query 页未逐项冒烟（agent-browser daemon 会话中期挂起，同构代码
> 已在 events/commodities 页验证）；无 ChartSwitcher 图表名标签、
> 无 charts=false URL 参数，`78d4233`）。
> M2-import-upload（Import 页上传块对齐上游 ImportFileUpload：新增
> "Upload files for import" 表单——文件选择器 + Upload 按钮，选中文件
> 经 FileReader 本地读入导入缓冲区，source path 取文件名、adapter 按
> 扩展名推断（.csv→csv，其余 beancount），状态行回执加载结果；
> 预览/提交既有流程不动。冒烟以 DataTransfer 注入 upload-test.bean
> 验证缓冲区装载、路径/adapter 回填与状态文案，随后 Preview 正常返回
> 诊断且 Commit 保持禁用。限制：上游为服务端 import 目录暂存 +
> importer 识别 + 文件列表 + 逐条目 Extract 弹窗，OC 无 import 目录
> 与 Python importer 生态，文件列表/extract 弹窗登记为待方案决策
> 余项；单文件读取（上游支持多文件并发上传）；Import 页文案仍为
> 硬编码英文（既有状态），`ea13c07`）。
> H2-droptarget-extend（文档拖放目标补齐至上游全集：journal 行复刻
> 上游 JournalTable ondragenter——dragenter 时将行内 description 单元
> 标记为 droptarget，账户取行内首个 account 链接（交易即首个过账
> 账户）；账户树 AccountCell 对应物（TreeTableNode 账户单元格）静态
> 挂载 droptarget + data-account-name。至此 droptarget 覆盖与上游四处
> （账户页标题/documents 账户表/journal 行/树单元格）同构。冒烟以
> 合成 Files-dragenter 验证 journal 行标记（Assets:Investments:Index05）
> 与树单元格高亮（Assets）；drop→模态复用已验证上传路径。限制：
> 上游 dragenter 同时写入 data-entry-date/data-entry-hash 供 uri-list
> attach，OC 未实现该分支，`44e6476`）。
> H1-deps（H1 依赖集入案：@codemirror/{autocomplete,commands,language,
> lint,search,state,view}@^6、@lezer/{common,highlight}@^1、
> web-tree-sitter@^0.26，与 pinned fava 1.30.12 frontend package.json
> 同 semver 区间；全链路验证通过（依赖暂未被代码引用）。此为 S4 波次
> 前置步，编辑器/查询编辑器的 CodeMirror 集成随后分步落地，`4eef5a1`）。
> T1/H5 的 WIP 覆盖部分仍待稠密夹具复比勾销。

### 优先级 1 — Phase 0 补做（共享基础，先于一切路由工作）

> **现状修正（2026-08-05 复核）**：稠密合成账本与参照捕获的基础设施
> **已经存在**——`tools/fixturegen` 生成已提交的稠密夹具（87 科目、304
> 交易、10 货币/商品，`testdata/fixtures/fava-reference`，含
> generator-lock.json 内容哈希），OCI 参照环境已产出 5 条代表路由 × 4 格
> 的候选截图（`testdata/visual-candidates/fava-reference/`）。剩余工作是
> 批准与覆盖，不是从零建造。

| 项 | 内容 | 完成判据 |
| --- | --- | --- |
| P0-1 | 稠密夹具缺口复核 | 核对 fixturegen 输出是否覆盖全部差距复现所需状态（无报价货币、断链汇率、分页量、全指令族）；缺口以固定输入扩展生成器，而不是手工改 .bean |
| P0-2 | 候选截图扩面 + 产品 owner 批准 | 候选捕获从 5 条代表路由扩到全部在案路由/状态；`testdata/visual-baselines/` 四格经产品 owner 批准后落地 |
| P0-3 | 每条可复现差距在合成账本上落回归证据 | 差距条目与测试/截图证据一一挂钩，私有观察不再承担回归职责 |

### 优先级 2 — 跨路由共享基础（可与 Wave 1/2 并行）

| 序 | 内容 | 覆盖差距 |
| --- | --- | --- |
| S1 | 移植上游 d3 图表套件（含货币圆点图例、图表类型切换、tooltip、层级钻取） | T1（T2 账户图亦依赖） |
| S2 | FQL 完整解析落 UI + 账户模糊自动补全 + 筛选 combobox 化 + 重载提示 | T3、M6 |
| S3 | 排序基建 + 通知区（已完成：M4 `62de047`、H7 `a5251c8`） | M4、H7 |
| S4 | CodeMirror 资产移植（beancount + BQL，tree-sitter wasm 已在参照锁哈希内） | H1、M1、F2 |
| S5 | 模态系统 + 五个缺失适配器契约 | H2 |
| S6 | 键盘快捷键 | H3 |

### 波次映射

| 波次 | 本清单工作项 |
| --- | --- |
| Wave 1 收尾 | T3/T4（shell 与导航保真，含 D1/D2 决策落地）、H6（已完成）、M5（已完成，11/11 CSS） |
| Wave 2 | T1 验收（BS/TB 图表货币圆点、Treemap/Sunburst/Icicle；三视图已完成，`b079d3b`/`b1247c8`）、L1（已完成，`88d90b9`） |
| Wave 3 | T2（账户详情完整化）、H5 收尾、M-CONTEXT/M-EXPORT 模态、账户 Journal change 列（已完成，`ae4eedc`） |
| Wave 4 | H4（Holdings/Events/Statistics/Documents/Commodities/Errors 专用组件已完成）、M3（已完成，`f1a01f0`） |
| Wave 5 | M1（Query 完整形态，依赖 S4） |
| Wave 6 | Editor/Import/AddEntry 写入路径（依赖 S4/S5）、M2 |
| Wave 7 | M6 全路由 URL 状态核对（已完成，`d68e5e9`/`36fb31b`）、L2 偏差登记（holdings 页签集合已登记 L3，`cf397c4`）、清理 |

### 需产品 owner 拍板的决策项

| # | 问题 | 建议 |
| --- | --- | --- |
| D1 | OC 独有 `Accounts` 导航项混在标准导航 | **已落地**：移入标注的 OrangeCount 扩展区，无需登记偏差 |
| D2 | 顶栏 Language/Theme 下拉 | **已落地**：顶栏原创下拉已移除，主题进 Options→Color scheme，locale 作为 fava option |
| D3 | Import 页 OC 原创表单 | 对齐 Fava 流程（文件列表 + 上传 + extract/review）；Python importer 排除维持既有批准偏差 |
| D4 | i18n 静态字典 vs gettext | 无用户可见差异，不构成偏差、无需登记（实现自由度）；仅需在 zh-CN 结构检查中持续验证覆盖完整性 |

## 与 QA 流程的口径

- 每修复一条差距：在合成参照账本上双端复比 + 更新
  [docs/fava-route-state-manifest.md](fava-route-state-manifest.md) 对应行
  状态；私有账本仅做临时冒烟，不留证据。
- 本清单是修复工作输入，不是验收权威；验收仍以四层路由门禁与
  manifest 为准。
- 旧版 UI 差距清单（fava-visual-gap-analysis.md）已标注范围，不再更新。
