# Fava 视觉与交互差距分析（2026-08-03）

## 背景与方法

本文档记录了一次对 OrangeCount 内置 Web 界面（`http://127.0.0.1:54870`）与本地
Fava 实例（`http://127.0.0.1:5000`）逐页并排人工比对的结果，账本为使用者本人
的真实生产账本（约 89 个科目、约 4900 条过账记录，涉及 CNY/USD/AUD/HKD/JPY
等十余种货币及若干基金/股票商品）。

这与 [docs/fava-acceptance-matrix.md](fava-acceptance-matrix.md) 中记录的
2026-08-02 QA 结论（15 条路由全部 `pass`）使用的方法不同：那次验收用的是一个
小型 `core.bean` 夹具。本次比对表明，**很多结构性差异只有在真实的多币种、多
科目账本规模下才会显现**——小夹具下不触发的"货币不可用""图表整卡消失"等
状况，在真实账本上会在多个核心报表页反复出现，这很可能是"验收记录全线通过"
与用户主观感受"界面与 Fava 差异巨大"之间落差的主要原因。

本文档遵循与既有文档相同的脱敏纪律：不包含真实科目全名、真实金额、真实截图；
下文示例账户名、数值均为占位/泛化处理。

结论先行：现有验收矩阵证明的是 **ADR-0007 意义上的"工作流等价"**（能否完成
同样的探索/筛选/报表工作流），而用户现在要求的是**更细粒度的结构与交互对齐**。
下面按影响力从高到低列出具体差距，每条给出 Fava 现象、OrangeCount 现象、影响、
修复建议、涉及文件与工作量估计。部分条目已直接标注是否与既有 ADR 的取舍相关。

## 差距清单（按影响力排序）

### 1. 报表数据模型：货币"按行"表达 vs Fava"按列"表达 — Critical

- **Fava 现象**：资产负债表 / 损益表 / 试算表中，每个科目只出现一行；该科目
  持有的多种货币在同一行内以纵列（如 `CNY | USD | Other`）并排显示，报表始终
  展示科目的原始持有货币，不需要用户先选择一个"显示货币"。
- **OrangeCount 现象**：后端 `report.Accounts` / `BalanceSheet` /
  `IncomeStatement` / `TrialBalanceTree`（`internal/report/reports.go:190-522`）
  产出的 `query.Result` 是"每个科目 × 每种持有货币"各一行（列为
  `account, currency, own_balance, total_balance, ...`）。前端顶栏的
  USD/CNY 全局切换实际决定了整份报表用哪个货币取数；当某科目在该货币下没有
  自然余额且没有可用汇率报价时，图表与摘要直接判定为
  `unavailable-currency`（`internal/web/assets/app.js:570-576`），而不是像
  Fava 那样只是那一格没有数值、其余货币列照常显示。
- **影响**：这是本次比对中观感差异最大的单一原因。多货币账本下，OrangeCount
  的四个核心报表（资产负债表、损益表、试算表、科目列表）几乎每次切换货币都会
  触发 "Unavailable: no conversion quote"，或者同一科目在表格里重复出现多行
  （每种货币一行）。Fava 的账本浏览体验里从不会出现这种情况。
- **修复建议**：在展现层新增"按科目 pivot 货币列"的转换：以 `account` 分组，
  把 `currency` 展开为动态列（按账本中出现的货币排序，记账本位币优先），
  `own_balance` / `total_balance` 分别取该货币的值；科目未持有的货币格显示为
  空，而不是让整行/整卡"不可用"。全局货币开关的语义应改为"图表默认展示/汇总
  换算货币"，而非"唯一可查看货币"（见条目 5）。
- **涉及文件**：`internal/report/reports.go`、`internal/report/charts.go`、
  `internal/web/assets/app.js`（`renderTable` / `treeMetadata` / `mountTable`）。
- **工作量**：中—大（涉及表结构或前端 pivot 逻辑改造，需要补充多货币科目的
  测试用例）。
- **与既有 ADR 的关系**：ADR-0007 承诺的是"工作流等价"而非"UI 内部实现"，
  但"能否一眼看懂一个多币种科目的余额"本身就属于工作流的一部分，建议将其
  定性为工作流缺陷而非纯视觉细节。

### 2. Journal：按"过账行"平铺 vs 按"交易"分组 — Critical

- **Fava 现象**：每笔交易只占一个表头行（日期、勾选状态色块、
  `Payee · Narration`），点击行尾的过账计数徽章可展开缩进的过账明细；筛选
  工具栏是一组可点击切换的条目类型徽章（`Open / Close / Transaction / * / ! /
  x / Balance / Note / Document / D / L / Pad / Query / Custom / B /
  Metadata / Postings`）。
- **OrangeCount 现象**：`renderReport("journal")`
  （`internal/web/assets/app.js:939-953`）把 `/api/v1/reports/journal`
  返回的过账行逐行渲染成表格；一笔含多条过账的交易会展开成多行，且日期 /
  Payee / Narration 在每一行重复。筛选是自由文本框
  （Flag / Tag / Link / Payee / Narration）+ Apply/Reset 按钮，没有条目类型
  徽章，也没有交易级别的分组与展开。
- **影响**：Journal 是 Fava 中使用频率最高的页面之一。示例账本约 4900 行
  过账、约 99 页，平铺表格与 Fava 的分组视图在信息密度和可读性上差距非常
  直观，很可能是用户最容易察觉"这不像 Fava"的地方。
- **修复建议**：在前端按 `(date, 交易在文件中的锚点如 line/id)` 对过账分组，
  渲染"交易头行 + 可展开明细行"的结构；工具栏在自由 Flag 输入旁增加条目
  类型徽章组，点击切换等价于设置 `flag`/新增 `kind` 查询参数。优先做前端
  分组（复用现有 `/api/v1/reports/journal` 返回的过账数据），可以避免后端
  API 大改。
- **涉及文件**：`internal/web/assets/app.js`（`journalToolbar` /
  `renderReport` / `mountTable`）；如需要服务端预分组，涉及
  `internal/report`。
- **工作量**：中（纯前端分组可先落地）。

### 3. 科目详情页缺失（对应 Fava 的"账户页"）— High

- **Fava 现象**：点击任意科目名进入该科目专属页面，包含该科目余额走势图
  （`Balance` / `Changes` 切换）与专属的 Account Journal（同款条目类型徽章、
  且多一列运行结存 `Balance`）。
- **OrangeCount 现象**：科目链接实际跳转到 `?view=accounts&account=X`，落地
  页是通用的科目汇总列表（"Accounts"）按 `account` 参数过滤后的结果，并非
  专属的单科目页面；表格也没有运行结存列。
- **修复建议**：新增单科目详情视图（可复用 `accounts` 路由、在检测到精确
  匹配单一科目时切换渲染模式），渲染该科目专属余额图 + 过账明细（含运行
  结存列）。
- **涉及文件**：`internal/web/assets/app.js`（路由分发逻辑）；如需运行结存
  的服务端支持，涉及 `internal/report`。
- **工作量**：中。

### 4. 层级图表（Treemap/Sunburst/Icicle）在真实数据上"整卡消失" — High

- **Fava 现象**：试算表默认渲染 Treemap，顶部有可点击的货币圆点图例（按
  账本中出现的货币排列，默认选中记账本位币），切换 Sunburst / Icicle 正常
  渲染。
- **OrangeCount 现象**：`renderHierarchyTreemap` /
  `renderHierarchySunburst` / `renderHierarchyIcicle`
  （`internal/web/assets/app.js:702-763`）在 `hierarchyTotal(nodes)` 计算
  结果为 0 时直接返回空字符串，导致整个图表卡片消失且没有任何提示文案。由于
  全局货币开关默认是 `USD`，而示例账本中大多数科目净额以 `CNY` 计价，试算表
  的层级图表因此经常整卡消失。
- **修复建议**：图表默认货币应取账本 `operating_currency` 选项或余额规模最大
  的货币，而不是固定 `USD`；当所选货币下确实没有可视化数据时，应展示与
  `chartAvailabilityNote` 一致的提示文案，而不是让整张图表卡片静默消失。
- **涉及文件**：`internal/web/assets/app.js`（`renderHierarchyChart` 及其
  三个子函数、`reportState` 初始货币来源）、`internal/report/charts.go`。
- **工作量**：小—中。

### 5. 全局货币切换器的语义与 Fava 不同 — Medium（条目 1、4 的根因之一）

- **Fava 现象**：顶部没有"全局唯一显示货币"开关；每个报表 / 图表自行决定
  展示哪些货币列，或提供图表专属的货币圆点选择器。
- **OrangeCount 现象**：顶栏的 `USD`/`CNY` 是全站唯一的"当前显示货币"状态
  （`reportState` / `globalQuery()`），几乎所有报表表格、图表、CSV 导出都
  绑定这一个全局值。
- **修复建议**：将其重新定位为"默认换算 / 图表展示货币"的偏好项，报表主
  表格不再受它限制（见条目 1），仅用于需要单一汇总数字的场景（如 Overview
  卡片、图表默认取值）。
- **工作量**：建议与条目 1 合并实施，避免两次改动展现层。

### 6. Options 页信息密度远低于 Fava — Medium

- **Fava 现象**：提供 `System / Dark / Light` 主题切换，并展示一张从账本
  `option` 指令解析出的 Fava 配置项只读表（如
  `account-journal-include-children`、`fiscal-year-end`、`indent` 等）。
- **OrangeCount 现象**：仅 3 个可写字段（语言 / 货币 / 时间范围），没有主题
  切换（`index.html` 顶部硬编码 `color-scheme: dark` 及一整套暗色 CSS 变量），
  也没有展示账本 `option` 指令的只读表。
- **修复建议**：
  1. 拆分出可覆盖的 CSS 自定义属性，新增浅色主题样式与
     `System / Dark / Light` 三态切换。
  2. 在 Options 页追加一张只读表，展示后端已经在读取的
     `current.Evaluation().Options`（参见 `internal/web/server.go` 中
     `handleOptions` 的 `GET` 分支，数据已经存在，只是前端未渲染成表）。
- **涉及文件**：`internal/web/assets/index.html`（CSS 变量化）、
  `internal/web/assets/app.js`（`renderOptions`）。
- **工作量**：主题切换为中等；选项只读表工作量小（数据源已存在）。

### 7. 页面标题重复 — Low（但每页都有，修复成本极低）

- **现象**：每个页面顶部固定的 `page-header`（`#page-title`）已经显示了页面
  名称，`renderReport` / `renderEditor` 等渲染函数又在 `#app` 内部重复渲染
  一次同名 `<h2>`（例如 `internal/web/assets/app.js:944` 的
  `` `<h2>${escapeHTML(t("journal"))}</h2>` ``）。Fava 只有一处标题
  （面包屑）。
- **修复建议**：移除各渲染函数中重复的 `<h2>`，标题只保留顶部
  `page-header`。
- **工作量**：小，但需要逐页面清理（Journal / Editor / Import / Options /
  Query / Help 等）。

### 8. 顶栏品牌与账本标题脱节 — Low

- **Fava 现象**：左上角面包屑显示账本 `option "title"` 指令中设置的账本
  名称 › 当前页面名，点击账本名回到首页。
- **OrangeCount 现象**：顶栏始终显示固定产品名 "OrangeCount"，未读取账本
  `option "title"` 的解析结果（该值其实已经被解析出来，可以在 Editor 源码
  高亮视图里看到对应的 `option` 行）。
- **修复建议**：让状态/选项接口把 `title` 选项一并暴露给前端，顶栏用
  "账本标题 › 当前页面"面包屑替换固定品牌字符串（保留跳转首页的链接）。
- **工作量**：小。

### 9. 导航项与 Fava 不完全一一对应 — Low（设计取舍而非缺陷）

- OrangeCount 比 Fava 多出 `Overview`、`Accounts`、`Prices`、`Source`、
  `Diagnostics` 几个导航项；Fava 没有独立的 Overview（默认落地页由
  `default-page` 选项决定，示例中是 Income Statement），`Prices` 并入
  `Commodities` 的一个分页签，`Errors` 只在存在诊断问题时才出现在导航中。
- **建议（可选）**：
  1. `Diagnostics` 参考 Fava 的 `Errors`，仅当计数 > 0 时才在导航中出现，
     而不是常驻显示 "0"。
  2. 评估是否把 `Prices` 并入 `Commodities` 的分页签，减少与 Fava 页面数量
     的一一对应困惑。
  3. `Overview` / `Accounts` / `Source` 属于 OrangeCount 的差异化增值页面，
     建议保留，但可以在视觉上做分组区分（例如单独一个"扩展"分组标题），
     避免用户误以为这是"翻译错的 Fava 页面"。
- **工作量**：小。

### 10. Editor 非实时高亮的静态文本域 — Low-Medium

- **Fava 现象**：CodeMirror 驱动的真代码编辑器，输入即时着色，并带
  File / Edit 菜单。
- **OrangeCount 现象**：`<textarea>` + 独立行号 `<pre>` + 一个
  `.syntax-preview` 高亮层；需要在真实交互中确认输入时高亮层是否实时刷新，
  当前观察到的是首次加载时渲染一次。
- **建议**：优先确认打字过程中高亮是否同步刷新；若预算有限，可先做到
  "停止输入后重新着色"的近似效果，完整的字符级实时双向绑定可作为后续投入
  产出比最低的一项放在最后处理。

## 建议的实施顺序

1. 条目 1 + 5（货币按列 pivot、全局货币开关语义修正）——影响面最广，是其余
   多个 "Unavailable" 类问题的根因。
2. 条目 2（Journal 交易分组 + 条目类型徽章）——使用频率最高的页面。
3. 条目 4（图表默认货币选择、"消失"降级为提示文案）——复用条目 1 的货币
   选择逻辑。
4. 条目 3（科目详情页）。
5. 条目 7 + 8（标题去重、面包屑）——成本很低，可以随手清理。
6. 条目 6（Options 信息表 + 主题切换）。
7. 条目 9（导航项取舍）、条目 10（编辑器实时高亮）。

## 对既有 QA 流程的建议

- 建议在 [docs/fava-acceptance-matrix.md](fava-acceptance-matrix.md) 使用的
  小型 `core.bean` 夹具之外，补充一个"结构相似但已脱敏"的多货币夹具（至少
  3 种货币、跨多层级科目、包含没有自然汇率报价的科目），纳入未来的验收回归，
  避免只有真实账本规模才会暴露的问题被小夹具掩盖。
- 每完成本文档中的一项修复，建议对照 Fava 对应页面重新手工过一遍，并在
  `fava-acceptance-matrix.md` 中补充一行"多货币场景"验收记录，而不是仅更新
  本文档，以保持两份文档的口径一致。
