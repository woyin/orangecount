# Fava 参考基线 vs OrangeCount 渲染：真实像素比对（2026-10-28）

## 结论

**当前「1:1」不成立。** 用同一确定性夹具（`testdata/fixtures/fava-visual`）、
同一 Playwright/Chromium、同一四格矩阵，把 Fava 1.30.12 的参考截图与
OrangeCount 的截图逐像素比对，**48 组中 0 组低于 1% 差异，中位数 44.7%**。

这不再是「文档声称 vs 门禁未跑」的问题，而是已测量的实际差距。

## 怎么产生的

1. 原生安装 Fava 1.30.12（`uv`，Docker 凭证助手不可用，见下），
   用 `testdata/fixtures/fava-visual/main.bean` 启动，四格（desktop/narrow × light/dark）
   抓 17 条路由 + 3 个模态 = 20 张/格，共 80 张，存 `testdata/visual-baselines/`。
2. OrangeCount 用同一夹具、同一四格矩阵渲染到 `testdata/visual-regression/`（48 张）。
3. `web/scripts/compare-to-reference.mjs` 逐像素比对（通道差 > 8 记为不同）。

## 实测数据（摘要）

| 单元格 | 路由 | 差异 |
| --- | --- | --- |
| desktop-light | account | 60.8% |
| desktop-light | trial-balance | 50.0% |
| desktop-light | balance-sheet | 45.8% |
| desktop-light | income-statement / root | 44.7% |
| desktop-light | commodities | 24.5% |
| desktop-light | editor | 22.2% |
| desktop-light | events | 21.5% |
| desktop-light | statistics | 17.5% |
| desktop-light | holdings | 17.5% |
| desktop-light | journal | 9.4% |
| desktop-light | documents | 7.3% |
| narrow-* | 7 条路由 | 页面总高不同（如 account 927 vs 1916 px），100% |
| narrow-* | options / query / errors / help / import / 模态 | 无对应 OrangeCount 快照 |

完整逐项数据见 `docs/evidence/visual-parity-2026-10-28.txt`。

## 这意味着什么

- `favagates` 门槛 5（视觉基线入库）通过，只证明「基线来自真实 Fava 且带环境锁」。
  它**不**证明 OrangeCount 与 Fava 一致。
- 37 条路由的 `status` 保持非 `accepted`：四层证据里 visual 层现在指向真实参考基线，
  但实际渲染并未对齐。把它们改标 `accepted` 会是伪造。

## 环境说明（诚实记录）

Docker daemon 的 credential helper 失效（`error getting credentials - err: signal: terminated`），
无法构建 `tools/fava-reference/Dockerfile` 的 OCI 参考镜像。
改用原生 `uv` 安装 Fava 1.30.12 + beancount 3.2.3，环境由
`environment-lock.json` 的 `environment_digest` 与 `uv_lock_sha256` 固定，
`favagates` 门槛 5 已扩展为接受这种可复现的原生环境（`capture_method` + 摘要 + 锁哈希）。

## 下一步（真正的工作）

1. 按差异从大到小逐条路由对齐：account（60.8%）、trial-balance（50%）、balance-sheet（45.8%）、
   income-statement/root（44.7%）。
2. narrow 三格先修页面总高差异（布局/字体/折叠行为）。
3. 为 options/query/errors/help/import 与三个模态补 OrangeCount 侧快照，才能比对。
4. 每修一条就重跑 `compare-to-reference.mjs`，把差异降到阈值内才把该行改 `accepted`。

## 结构级根因（2026-10-28 定位）

差异不是配色或字体——实测两者完全一致：

| 属性 | Fava 1.30.12 | OrangeCount |
| --- | --- | --- |
| 顶栏底色 | `rgb(0,101,163)` | 同 |
| 正文底色 | `rgb(242,242,242)` | 同 |
| 字体 | `"Fira Sans", sans-serif` 14px | 同 |
| 侧栏盒 | `0,52 160x748` | 同 |
| 主区盒 | `160,52 1120x748` | 同 |

差异全部来自**主区内容结构**（以 `/income_statement` 为例）：

| 位置 | Fava | OrangeCount |
| --- | --- | --- |
| y=73 | 控制条 `35px`：货币选择 + Stacked/Single Bars + At Cost | 无（并入图表卡） |
| y=115 | 图表 `1078x256` | 图表卡起始 `1078x420`（标题+图+图例合并） |
| y=371 | 图例行 `1078x21` | 无独立行 |
| y=396 | 表格区起始 | y=497（被 420px 图表卡下推 101px） |

具体缺口：

1. **缺少独立图表控制条**（Fava `ChartSwitcher` + 货币圆点选择器 + 估值选择器）。ADR-0041 把现代图表层设为默认并移除了 parity 回退，因此这条控制条整体不存在。
2. **图表高度 280px vs Fava 250px**（`ModernChart.svelte:71`）。
3. **标题/图例并入图表卡**，而 Fava 是三个独立块，导致后续内容整体下移。

修法（按影响排序）：

1. 在 `TreeReport.svelte` 的 `.report-charts` 之上补 Fava 式控制条（`ChartSwitcher` + 货币圆点 + 估值），高度 35px。
2. `ModernChart` 高度改 250，图例拆出为独立行（21px）。
3. 重跑 `compare-to-reference.mjs`，预期 income-statement 从 44.7% 降到 10% 以内。

## 账户页：多渲染了两个区块（narrow 927 vs 1833 px）

同一 narrow 视口（520×800）下，主区结构实测：

| Fava 1.30.12 | 高度 | OrangeCount | 高度 |
| --- | --- | --- | --- |
| 筛选行 | 40 | headerline | 45 |
| 图表 | 257 | headerline sections | 24 |
| 图例行 | 24 | chart-card（余额图） | 355 |
| 账户日记账表 | 378 | average-cost-current | 52 |
| | | chart-card（成本图） | 355 |
| | | headerline | 45 |
| | | chart-card（第二余额图） | 355 |
| | | table-scroll | 73 |
| | | journal-chips | 127 |
| | | journal 列表 | 118 |

**根因**：OrangeCount 在账户页额外渲染了「平均成本演化」区块与两张附加图表卡。
Fava 1.30.12 的账户页只有一个图表 + 图例 + 账户日记账。
平均成本是 OrangeCount 的扩展能力（ADR-0042），按偏差规则应以可选方式呈现，
而不是默认插进标准路由。

修法：把 average-cost 区块与附加图表卡改为 `?chart_layer=modern` 或独立开关下才显示，
默认路径只保留 Fava 的单图表 + 图例 + 日记账。预计 narrow 账户页从 1833px 降到约 930px。

## 已完成的修正（本次）

1. **恢复 parity 图表为默认**（`ChartView.svelte`）：ADR-0041 曾把现代图表设为默认并删除回退，
   这使 FD-0006 的批准前提（可切回 parity）失效。现在默认 `ReportChart`，
   `?chart_layer=modern` 显式启用现代层。
   效果：中位差异 **44.7% → 32.4%**。
2. **图表高度 280 → 250**（`ModernChart.svelte`），对齐 Fava 的 250px。
3. 补齐 5 条路由 + 2 个模态的 OrangeCount 快照，使比对从 48 组扩到 76 组。
4. 新增 8 条浏览器用例覆盖 errors/help/options/query/locale/keyboard/export/print。

## 最新测量（修复后）

| 指标 | 起始 | 现在 |
| --- | --- | --- |
| 比对组数 | 48 | **76** |
| 中位像素差 | 44.7% | **31.2%** |
| 低于 5% 差异 | 0 | **0** |
| 页面总高不同的组 | 20（多为窄屏） | **24** |

已修：

1. 恢复 parity 图表为默认（ADR-0041 曾删除回退，使 FD-0006 的批准前提失效）。
2. 图表高度 280 → 250，对齐 Fava。
3. 账户页重复渲染图表（`AccountReport` 与 `GenericReport` 各渲染一次）→ 去重，
   narrow 账户页从 1833px 降到 1030px（Fava 927px）。
4. 平均成本区块改为 `chart_layer=modern` 下才显示（它是 OrangeCount 扩展 ADR-0042）。
5. 比对口径修正：宽度不同才判全不等；高度不同只比重叠区并单独报告高度差，
   避免把 19px 的高度差误报成 100% 全不等。

仍未对齐的主要项：

| 项 | 现状 |
| --- | --- |
| 图表控制条（Fava `ChartSwitcher` + 货币圆点 + 估值） | 缺失，约 35px 高度差 |
| narrow 各路由页面总高 | 与 Fava 差 20–100px（布局/折叠细节） |
| 模态截图比对 | `m-add`/`m-export` 差异 28–88%，模态内容与位置不同 |
| account 桌面 | 36.6%（桌面） |

## 最终测量（2026-10-28 收口）

```
76 组比对（17 路由 + 2 模态 × 四格）
0 组低于 1% 像素差
0 组低于 5%
中位差异 32.9%
24 组存在页面高度差
```

### 本轮已修的实质缺陷

| # | 缺陷 | 影响 |
| --- | --- | --- |
| 1 | **ADR-0041 删除 parity 图表回退**，使 FD-0006 的批准前提（可切回 parity）失效 | 中位差异 44.7% → 32.4% |
| 2 | 图表高度 280px vs Fava 250px | 后续内容整体下移 |
| 3 | **账户页图表重复渲染**（`AccountReport` 与 `GenericReport` 各渲染一次） | narrow 账户页 1833px → 1030px（Fava 927px） |
| 4 | 平均成本区块（ADR-0042 扩展）默认插入标准路由 | 改为 `chart_layer=modern` 下才显示 |
| 5 | 缺少图表控制条（货币图例 + 模式切换 + 转换/区间） | 已补 `.chart-controls` |
| 6 | 图例与 interval/valuation 注释混在图表卡内 | 拆为图表卡下方的独立行 |

### 仍未对齐

| 项 | 现状 | 修法 |
| --- | --- | --- |
| income_statement / root | 31–64% | 图表内部几何（柱宽、轴刻度、padding）与 Fava `BarChart.svelte` 不一致 |
| trial_balance | 50% | 层级图（Treemap/Sunburst/Icicle）的布局算法差异 |
| account | 42–65% | 账户页仍有额外 headerline 与区块顺序差异 |
| narrow 各路由 | 高度差 20–100px | 折叠/换行细节 |
| 模态（m-add/m-export） | 27–88% | 模态表单布局与 Fava `entry-forms` 不同 |

### 结论

**「Go 近似 1:1 复刻 Fava」在当前状态下不成立**，且这不是文档口径问题：
用真实 Fava 1.30.12 参考截图逐像素比对，中位差异 32.9%，没有任何一组在 5% 以内。

要达成该说法，剩余工作是逐路由的图表几何与布局对齐，工作量在数十个前端切片量级。
在此之前，`docs/fava-route-state-manifest.md` 的 37 行保持非 `accepted` 是正确状态。

## 收口状态（2026-10-28 末）

```
76 组比对（17 路由 + 2 模态 × 四格）
0 组低于 1% 像素差
0 组低于 5%
中位差异 31.9%（起始 44.7%）
12 组存在页面高度差（起始 24）
```

### 本轮修掉的 8 个实质缺陷

| # | 缺陷 | 证据 |
| --- | --- | --- |
| 1 | ADR-0041 删除 parity 图表回退，使 FD-0006 批准前提失效 | `ChartView.svelte` 恢复默认 `ReportChart` |
| 2 | 图表高度 280 vs Fava 250 | `ModernChart`/`ReportChart` 均改 250 |
| 3 | 账户页图表重复渲染（`AccountReport` + `GenericReport` 各一次） | 去重后 narrow 账户页 1833 → 1030px（Fava 927） |
| 4 | 平均成本扩展区块（ADR-0042）默认插入标准路由 | 改为 `chart_layer=modern` 下才显示 |
| 5 | 缺少图表控制条（货币图例 + 转换/区间） | `TreeReport` 新增 `.chart-controls` |
| 6 | 转换/区间下拉错放在 header（Fava 在图表控制条） | 从 `Header` 移入 `TreeReport`，narrow header 129 → 102px 对齐 Fava |
| 7 | header 缺 Fava 账本图标 | 补 24×25 SVG |
| 8 | 报告标题句式大写（`Income statement`）vs Fava 标题式（`Income Statement`） | `translations.ts` 修正三个报表名 |
| 9 | 图例与 interval/valuation 注释混在图表卡内 | 拆为图表卡下方独立行，并删掉 Fava 没有的 meta 行 |

### 剩余差异的定位

- **图表内部几何**（试算平衡 49%、账户 40%、损益 32%）：Fava `BarChart.svelte` 的柱宽、y 轴 tick 数与 padding 与我们的实现不同。
- **层级图**（试算平衡）：Treemap/Sunburst/Icicle 的布局算法差异。
- **模态表单**（m-add 26%）：条目表单的行距与分节与 Fava `entry-forms` 不同。
- **其余路由 5–25%**：多为表格列宽与字体渲染的亚像素差。

这些都是逐路由的前端对齐工作，不是单一根因。中位 31.9% 到「1:1」之间还差数十个切片。

## 补充修正（同日，图表宽度与树表布局）

| # | 缺陷 | 效果 |
| --- | --- | --- |
| 10 | 试算平衡树表头显示 "Accounts" 文字（Fava 的 AccountCellHeader 没有文字） | 表头对齐 |
| 11 | "Other" 列缺 `.num` 类（Fava 是数值列） | 列宽对齐 |
| 12 | 层级图被 250px 时间序列高度压扁（Fava 为 min(width/2.5, 400)，sunburst 500） | 试算平衡图表尺寸对齐 |
| 13 | 试算平衡用两列布局，Fava 的 TrialBalance.svelte 不带 column 包装 | 高度差组数 12 → 10 |
| 14 | 图表宽度试过改为 100% 反而变差（31.9% → 40.5%），已回退到 `min(100%, 52rem)` | 记录实测，避免再次误改 |

当前：76 组，中位 **31.9%**，0 组低于 5%，10 组有高度差。
