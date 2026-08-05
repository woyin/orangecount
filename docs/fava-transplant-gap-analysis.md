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
| T1 | R-IS/R-BS/R-TB | 图表系统为手写内联 SVG | d3 套件：坐标轴、刻度、日期标注、tooltip、图例、货币圆点选择器、图表类型切换（Stacked/Single Bars、Line/Area、Treemap/Sunburst/Icicle）、层级钻取 | 粗粒度 SVG；实测 IS/BS/TB/账户页均**无货币圆点**、无图表类型切换（TB 的三种层级图完全缺失）；无 tooltip/刻度 | 前端组件缺失 |
| T2 | R-ACCOUNT | 账户详情页过薄 | 标题含 `(Last entry: date)` 与科目层级面包屑；Balance/Changes 切换；Account Balance/Changes(monthly)/Balances(monthly) 三区块；账户图 + 货币圆点；Account Journal 带徽章 | 仅余额树 + Journal；无 Last entry、无 Balance/Changes、无区间变化表、无账户图 | 前端 + 适配器（缺区间统计与 up-to-date 状态契约） |
| T3 | G-SHELL/G-FILTERS | shell 控件缺口 | 导航含 `Go to account` 组合框、`+`（Add Entry）、`⬇`（Export）；Time/Account/FQL 为带建议下拉的 combobox，账户模糊自动补全、FQL 解析校验、`r` 重载与变更提示 | 无 `Go to account`；`+` 为死链接；无 `⬇`；三个筛选为纯文本框，无补全、无 FQL 校验 | 前端组件 + 适配器（FQL 解析契约已有计划未落 UI） |
| T4 | G-SHELL | 标准导航被 OC 原创项污染 | 导航严格等于 Fava 标准面 | 导航含 OC 独有 `Accounts` 项；顶栏含 OC 原创 Language/Theme 下拉（Fava 的主题入口在 Options→Color scheme，locale 是 fava option） | 结构决策（违反移植计划不可妥协项 #6，见决策项 D1/D2） |

### High

| # | Manifest | 差距 | Fava 行为 | 移植版现状 | 根因层 |
| --- | --- | --- | --- | --- | --- |
| H1 | R-EDITOR/R-QUERY | CodeMirror 未移植（上游 19 文件 + tree-sitter wasm） | 语法高亮、行号、折叠、补全、snippets、File/Edit 菜单、文件树 | 裸 textarea + Files listbox；无菜单 | 前端组件缺失 |
| H2 | M-ADD/M-CONTEXT/M-EXPORT/M-DOCUMENT | 模态系统整体缺失（上游 9 文件） | Add Entry 表单、条目 Context（余额/位置）、Export/Download、文档上传 | 无 modals 目录；侧栏 `+` 死链 | 前端 + 适配器（缺 add_entries、entry context、export 契约） |
| H3 | G-KEYBOARD | 全局键盘快捷键缺失 | `g-*` 路由跳转、`t/f/a/d/s`、`?` 快捷键提示 | 无任何 keydown 监听 | 前端组件缺失 |
| H4 | R-HOLD-*/R-COMMODITIES/R-EVENTS/R-STATISTICS/R-DOCUMENTS | 六路由降级为通用平表 | Holdings 四子页签与成本分组；Commodities 商品列表 + 每商品页（元数据/精度/价格历史）；Events 按事件类型侧栏分组；Statistics 指令计数 + Postings-per-Account + 活动图；Documents 账户树 + 内嵌预览 | 统一 `GenericReport` 平表：Holdings 无子页签、列头为 snake_case 字段名；Commodities 渲染成价格明细表且实测空表；Events 无分组；Statistics 仅计数表；Documents 无预览 | 前端组件 + 适配器（专用契约未建） |
| H5 | R-JOURNAL | Journal 交互层不完整 | 全量条目类型徽章（含 Custom/B/Metadata/Postings）、排序与列菜单、点击条目→Context、URL 同步筛选、拖拽上传文档 | 核心徽章组与展开已现（**WIP**），其余未移植；Custom/B/Metadata/Postings 徽章覆盖待复核 | 前端组件缺失 |
| H6 | R-OPTIONS | Options 页不完整 | Color scheme（System/Dark/Light）单选组 + Fava options 表（带 help 链接）+ Beancount options 表 | 仅 Beancount options 表；主题入口被移到顶栏原创下拉 | 前端 + 适配器（fava_options 已有雏形未成表） |
| H7 | M-NOTIFY | 通知区缺失 | 文件变更/保存结果 toast，带点击重载 | 5s 轮询已对齐，但无可感通知 | 前端组件缺失 |

### Medium

| # | Manifest | 差距 | 说明 |
| --- | --- | --- | --- |
| M1 | R-QUERY | Query 页不完整 | 无保存查询、无结果排序、无查询图表；BQL 编辑器为裸 textarea（依赖 H1） |
| M2 | R-IMPORT | Import 为 OC 原创表单 | Source path/Adapter/Target 表单 vs Fava 的文件列表 + 上传 + extract/review 流程（Python importer 排除已是批准偏差，但 UI 流程应对齐） |
| M3 | R-HELP | Help 无页面索引 | Fava 有 Index/Beancount Syntax/Budgets/Conversion/Extensions/Features 等子页 |
| M4 | 跨路由 | 排序基建缺失 | `SortHeader`/`sortable-table` 未移植，次要路由表头不可排序 |
| M5 | 跨路由 | 三份 CSS 未移植 | `editor.css`、`help.css`、`notifications.css`（main 仅引入 9/11） |
| M6 | G-FILTERS | URL 状态对齐未逐项验证 | `state.mjs` 单一 reducer 替代上游 stores/*，直链/历史/重载行为需逐路由核对 |

### Low

| # | Manifest | 差距 | 说明 |
| --- | --- | --- | --- |
| L1 | R-ACCOUNT | 账户页标题无层级面包屑 | Fava：`Assets › Bank › <科目>`；移植版为完整科目名 |
| L2 | G-LOCALE | i18n 为静态 en/zh-CN 字典 | 上游为 gettext 目录；用户可见行为等价，建议登记为实现性偏差（D4） |

### 适配器契约缺口（数据层）

已接线：`changed`、`ledger_data`、`metadata`、`options`、`help`、
`diagnostics`、`editor`(读)、`import`(adapters/files/content)、`journal`、
三大树报表、泛型 `reports/*`、BeanQuery。

尚未支撑的 Fava 数据契约：

- `add_entries`（模态新增条目 POST）
- document upload（multipart）
- export/download（账本文件导出、filtered Beancount 导出）
- entry context（单条目余额/位置）
- query shell 元数据（补全、保存查询）
- editor 保存与 import commit 目前走 `/api/v1/*` 非 fava 适配器契约，
  形态是否并入 fava-shaped 适配器需在 Wave 6 前评估。

## Bridge 计划

排序策略（已确认）：差距映射到波次；跨路由共享根因作为共享基础提前；
其余严格按波次推进，波次内按差距优先级排序（ADR-0032 深度优先纪律不变）。

### 任务 0 — WIP 收尾（先决）

将当前 ~2400 行未提交改动按差距归属拆分提交（Journal 徽章/展开、树表多
货币 + Other 列、图表与 journal 适配器），提交后复核 T1/H5 并勾销已修复
项。**在完成前不再叠加新的界面改动**，保证"一次修复对一条差距"可追溯。

### 优先级 1 — Phase 0 补做（共享基础，先于一切路由工作）

| 项 | 内容 | 完成判据 |
| --- | --- | --- |
| P0-1 | 稠密多币种合成参照账本 + 确定性生成器与内容哈希 | 80–100 嵌套科目、6–10 货币、含无报价/断链汇率路径、长 Unicode 标签、分页级条目量、全指令/旗标族、lots、documents、events、诊断与 import 候选 |
| P0-2 | OCI 参照环境用同一账本生成 Fava 四格批准基线 | `testdata/visual-baselines/` 四格（desktop/narrow × light/dark）经产品 owner 批准 |
| P0-3 | 每条可复现差距在合成账本上落回归证据 | 差距条目与测试/截图证据一一挂钩，私有观察不再承担回归职责 |

### 优先级 2 — 跨路由共享基础（可与 Wave 1/2 并行）

| 序 | 内容 | 覆盖差距 |
| --- | --- | --- |
| S1 | 移植上游 d3 图表套件（含货币圆点图例、图表类型切换、tooltip、层级钻取） | T1（T2 账户图亦依赖） |
| S2 | FQL 完整解析落 UI + 账户模糊自动补全 + 筛选 combobox 化 + 重载提示 | T3、M6 |
| S3 | 排序基建 + 通知区 | M4、H7 |
| S4 | CodeMirror 资产移植（beancount + BQL，tree-sitter wasm 已在参照锁哈希内） | H1、M1、F2 |
| S5 | 模态系统 + 五个缺失适配器契约 | H2 |
| S6 | 键盘快捷键 | H3 |

### 波次映射

| 波次 | 本清单工作项 |
| --- | --- |
| Wave 1 收尾 | T3/T4（shell 与导航保真，含 D1/D2 决策落地）、H6（Options Color scheme + fava options 表）、M5 样式补齐 |
| Wave 2 | T1 验收（BS/TB 图表货币圆点、Treemap/Sunburst/Icicle）、L1 |
| Wave 3 | T2（账户详情完整化）、H5 收尾、M-CONTEXT/M-EXPORT 模态、账户 Journal change 列 |
| Wave 4 | H4（Holdings/Commodities/Events/Statistics/Documents/Errors 专用组件）、M3 |
| Wave 5 | M1（Query 完整形态，依赖 S4） |
| Wave 6 | Editor/Import/AddEntry 写入路径（依赖 S4/S5）、M2 |
| Wave 7 | M6 全路由 URL 状态核对、L2 偏差登记、清理 |

### 需产品 owner 拍板的决策项

| # | 问题 | 建议 |
| --- | --- | --- |
| D1 | OC 独有 `Accounts` 导航项混在标准导航 | 移入明确标注的 OrangeCount 扩展区（符合移植计划不可妥协项 #6），不登记为偏差 |
| D2 | 顶栏 Language/Theme 下拉 | 恢复 Fava 入口：主题进 Options→Color scheme，locale 作为 fava option；顶栏原创下拉移除。若坚持保留，需登记 Approved Fava deviation |
| D3 | Import 页 OC 原创表单 | 对齐 Fava 流程（文件列表 + 上传 + extract/review）；Python importer 排除维持既有批准偏差 |
| D4 | i18n 静态字典 vs gettext | 登记为实现性 Approved Fava deviation（无用户可见差异） |

## 与 QA 流程的口径

- 每修复一条差距：在合成参照账本上双端复比 + 更新
  [docs/fava-route-state-manifest.md](fava-route-state-manifest.md) 对应行
  状态；私有账本仅做临时冒烟，不留证据。
- 本清单是修复工作输入，不是验收权威；验收仍以四层路由门禁与
  manifest 为准。
- 旧版 UI 差距清单（fava-visual-gap-analysis.md）已标注范围，不再更新。
