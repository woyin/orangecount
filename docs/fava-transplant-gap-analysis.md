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
| T2 | R-ACCOUNT | 账户详情页过薄 | 标题含 `(Last entry: date)` 与科目层级面包屑；Balance/Changes 切换；Account Balance/Changes(monthly)/Balances(monthly) 三区块；账户图 + 货币圆点；Account Journal 带徽章 | 面包屑与 Last entry 指示器已实现（`88d90b9`/`eefbf83`），Journal 带 change 列（`ae4eedc`）；仍无 Balance/Changes、无区间变化表、无账户图 | 前端 + 适配器（缺区间统计与 up-to-date 状态契约） |
| T3 | G-SHELL/G-FILTERS | shell 控件缺口 | 导航含 `Go to account` 组合框、`+`（Add Entry）、`⬇`（Export）；Time/Account/FQL 为带建议下拉的 combobox，账户模糊自动补全、FQL 解析校验、`r` 重载与变更提示 | `Go to account`（侧栏）与 Time/Account/FQL 三个 AutocompleteInput combobox 已实现；`+` 为死链接（Add Entry 模态属 S5）；变更提示已由 H7 落地，`r` 手动重载快捷键与 ⟳ 按钮已实现（`4792f73`）；FQL 解析校验与完整语义已落地（`785be13`：#tag/^link 精确匹配、key:"value"、并置 and/逗号 or/`-` 取反、all()/any()、金额比较；非法字符与解析错误在 API 边界 400 并于 shell 错误区展示）；⬇ 导出已落地（`2846847`）；余 `+`（Add Entry 模态属 S5） | 前端组件 + 适配器（余 `+` 属 S5） |
| T4 | G-SHELL | 标准导航被 OC 原创项污染 | 导航严格等于 Fava 标准面 | D1/D2 已落地：OC 独有 `Accounts` 项移入标注的 OrangeCount 扩展区；顶栏原创 Language/Theme 下拉已移除，主题入口在 Options→Color scheme，locale 作为 fava option | 结构决策（见决策项 D1/D2，均已实现） |

### High

| # | Manifest | 差距 | Fava 行为 | 移植版现状 | 根因层 |
| --- | --- | --- | --- | --- | --- |
| H1 | R-EDITOR/R-QUERY | CodeMirror 未移植（上游 19 文件 + tree-sitter wasm） | 语法高亮、行号、折叠、补全、snippets、File/Edit 菜单、文件树 | 裸 textarea + Files listbox；无菜单 | 前端组件缺失 |
| H2 | M-ADD/M-CONTEXT/M-EXPORT/M-DOCUMENT | 模态系统整体缺失（上游 9 文件） | Add Entry 表单、条目 Context（余额/位置）、Export/Download、文档上传 | M-EXPORT 已落地（`2846847`）：modals 目录建立，Export 模态（#export hash 驱动）+ download-journal 端点（按过滤切取源码的保源导出）；M-CONTEXT 已落地（`68ddcfa`：entry-context 路由 + 只读源码切片模态）；仍缺 Add Entry/Document 模态 | 前端 + 适配器（缺 add_entries、文档上传契约） |
| H3 | G-KEYBOARD | 全局键盘快捷键缺失 | `g-*` 路由跳转、`t/f/a/d/s`、`?` 快捷键提示 | 已实现：`g-*` 路由跳转、`f t/f a/f f` 筛选快捷键、`?` 快捷键 tooltip（冒烟验证 19 条提示，含 `r` 重载）、`r` 手动重载（`4792f73`）；上游其余单键快捷键未登记为缺口 | 已完成 |
| H4 | R-HOLD-*/R-COMMODITIES/R-EVENTS/R-STATISTICS/R-DOCUMENTS | 六路由降级为通用平表 | Holdings 四子页签与成本分组；Commodities 商品列表 + 每商品页（元数据/精度/价格历史）；Events 按事件类型侧栏分组；Statistics 指令计数 + Postings-per-Account + 活动图；Documents 账户树 + 内嵌预览 | 统一 `GenericReport` 平表：Holdings 无子页签、列头为 snake_case 字段名；Commodities 渲染成价格明细表且实测空表；Events 无分组；Statistics 仅计数表；Documents 无预览 | 前端组件 + 适配器（专用契约未建） |
| H5 | R-JOURNAL | Journal 交互层不完整 | 全量条目类型徽章（含 Custom/B/Metadata/Postings）、排序与列菜单、点击条目→Context、URL 同步筛选、拖拽上传文档 | 核心徽章组与展开已现；点击条目→Context 已落地（`68ddcfa`：行尾 ⋮ 链接 + `#context-<hash>` 模态 + entry-context 私有路由，位置派生 entry_hash，只读源码切片；限制：before/after 余额与 CodeMirror 可编辑切片属 H1）。余：排序与列菜单、拖拽上传文档；Custom/B/Metadata/Postings 徽章覆盖待复核 | 前端组件 + 适配器（余列菜单与文档上传契约） |
| H6 | R-OPTIONS | Options 页不完整 | Color scheme（System/Dark/Light）单选组 + Fava options 表（带 help 链接）+ Beancount options 表 | 已实现：UtilityReport 含 Color scheme 单选组 + Fava options 表 + Beancount options 表；顶栏原创主题下拉已移除（D2） | 已完成（UtilityReport + `/__orangecount/fava/options` 契约） |
| H7 | M-NOTIFY | 通知区缺失 | 文件变更/保存结果 toast，带点击重载 | 已实现：notifications 模块早已在案（bootstrap/报告错误走 notify_err），本步补齐可感通知——文件变更 warning toast（点击再刷一次，5s 自动消失，冒烟验证文案与类名）与编辑器 Save 结果 toast（成功/拒绝/失败三态），`a5251c8` | 已完成 |

### Medium

| # | Manifest | 差距 | 说明 |
| --- | --- | --- | --- |
| M1 | R-QUERY | Query 页不完整 | 保存查询已落地（`1abfaf0`）：账本 `query` 指令投影为侧栏 Query 项子菜单（同名截断规则同上游），页内点选即回显并重跑；结果排序冒烟确认 GenericReport 列排序可用（含数值列方向切换）——原"无结果排序"判断有误；仍无查询图表；BQL 编辑器为裸 textarea（依赖 H1） |
| M2 | R-IMPORT | Import 为 OC 原创表单 | Source path/Adapter/Target 表单 vs Fava 的文件列表 + 上传 + extract/review 流程（Python importer 排除已是批准偏差，但 UI 流程应对齐） |
| M3 | R-HELP | ~~Help 无页面索引~~（已完成，`f1a01f0`） | `/help` 现渲染子页索引，`/help/<id>` 渲染单节页面 + 返回链接；Options 页标题链接 `/help/options`（限制：子页集合为 OC 自有 8 节，非上游 Index/Syntax/Budgets 全集） |
| M4 | 跨路由 | ~~排序基建缺失~~（已完成，`62de047`） | `sort/index.ts`（Sorter/SortColumn 契约，同上游点击语义，无 d3 依赖）+ `SortHeader.svelte`（legacy 模式，含 aria-sort 与箭头提示）已落地；events/commodities/documents/statistics/options 表头可排序，冒烟验证方向切换与列切换重排（限制：holdings 与上游一致不可排序；Query 结果排序经 GenericReport 实际可用，`1abfaf0` 冒烟确认） |
| M5 | 跨路由 | ~~三份 CSS 未移植~~（已完成） | `editor.css`、`help.css`、`notifications.css` 均已移植，main.ts 引入 11/11 |
| M6 | G-FILTERS | ~~URL 状态对齐未逐项验证~~（已核对，`d68e5e9`/`36fb31b`） | 全路由深链、筛选回显、history back/forward、reload 逐项冒烟通过；核对发现并修复三处：`time=2025-Q2` 季度语法被拒（新增 Filters.TimeBegin/End 半开区间）、`/editor?path=` 与 `/query?query_string=` 未回显（限制：diagnostics 仍为 JSON 兜底视图；Query 页图表仍属 M1，保存查询与排序已落地） |

### Low

| # | Manifest | 差距 | 说明 |
| --- | --- | --- | --- |
| L1 | R-ACCOUNT | ~~账户页标题无层级面包屑~~（已完成，`88d90b9`；Last entry 指示器 `eefbf83`） | 标题现为祖先面包屑 + Last entry 指示器；指示器跟随当前过滤，无条目上下文链接（限制） |
| L2 | G-LOCALE | i18n 为静态 en/zh-CN 字典 | 上游为 gettext 目录；用户可见行为等价，建议登记为实现性偏差（D4） |
| L3 | R-HOLDINGS | Holdings 页签集合与上游不一致（偏差登记） | 保留 OC 扩展页签 by_root_account/by_commodity，缺上游 by_cost_currency（适配器无 lot 成本货币维度，`3c9fef1`）。登记为实现性偏差：不补齐 by_cost_currency，除非后续引入成本货币分组数据 |
| L4 | R-ERRORS | serve 拒绝加载含 error 级诊断的账本（偏差登记） | Fava 带错服务并在 /errors 展示全部诊断；OC `serve` 在 main.go:181 检测到 error 即退出，/errors 页面只能展示 warning（`b902d7d`）。登记为实现性偏差：若要对齐 Fava，需 owner 批准放宽启动门禁 |
| L5 | M-NOTIFY | 文件变更提示与自动重载并存（偏差登记） | 上游 auto-reload 默认开启时静默重载、不弹 toast；本实现重载与 warning toast 并存以保证可感知（`a5251c8`）。登记为实现性偏差：若要对齐上游静默行为，移除变更 toast 即可 |

### 适配器契约缺口（数据层）

已接线：`changed`、`ledger_data`、`metadata`、`options`、`help`、
`diagnostics`、`editor`(读)、`import`(adapters/files/content)、`journal`、
三大树报表、泛型 `reports/*`、BeanQuery、`download-journal`
（过滤后条目按源码 span 切取的 Beancount 导出，`2846847`）、
`entry-context`（位置派生 entry_hash → 条目投影 + 只读源码切片，`68ddcfa`）。

尚未支撑的 Fava 数据契约：

- `add_entries`（模态新增条目 POST）
- document upload（multipart）
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
