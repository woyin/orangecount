# Fava 1:1 切换门槛清单（目标 2）

创建日期：2026-10-27。依据：`docs/fava-frontend-transplant-plan.md` Wave 7 done-when 与
`docs/fava-route-state-manifest.md`。

本文件是**唯一权威**的「1:1 复刻」达成状态表。机器可验证入口：`go run ./tools/favagates`。
它读取 route manifest、approved-deviations、provenance manifest 与仓库实际文件，
不读取手工勾选。

**下方状态由 2026-10-28 的真实运行结果填写。不要手工改绿；改动后必须重跑并同步。**

## 门槛状态

| # | 门槛 | 客观判据 | 当前 |
| --- | --- | --- | --- |
| 1 | 每条标准路由四层门禁通过 | manifest 中所有 R-/G-/M-/D- 行为 `accepted`，四层证据可解析且路径真实存在；`accepted` 行必须在偏差表中登记 FD-0011（功能一致为验收基准） | **通过**：37/37 accepted |
| 2 | 移植 UI 为唯一标准面 | `ORANGECOUNT_TRANSPLANTED_UI` 零命中 | **通过** |
| 3 | legacy UI 与原型 shell 删除 | `internal/web/assets/{app.js,app.css,index.html}` 不存在 | **通过** |
| 4 | 未批准视觉差异清零 | 无 `proposed` 条目；approved 均有批准日期 | **通过** |
| 5 | 视觉基线入库 | 四格齐全、每张 PNG 来自 Fava 参考捕获、基线根部有环境锁 | **通过**（80 张，17 路由 + 3 模态 × 四格） |

`go run ./tools/favagates` → exit 0。

## 验收基准：功能一致，不是像素 1:1

产品 owner 于 2026-10-28 决定：**不要求像素级 1:1，功能保持一致即可**。
该决定登记为 **FD-0011**（`docs/fava-approved-deviations.md`），类别是 presentation preference。

这带来两个后果：

1. **视觉基线仍然是渲染权威**：`testdata/visual-baselines/` 保存真实 Fava 1.30.12 的四格参考截图，
   用于回归与差异度量；它不再是像素相等门禁。
2. **门禁保护登记本身**：`favagates` 门槛 1 现在要求「若某行标 `accepted`，
   则 FD-0011 必须在偏差表中为 `approved` 且带批准日期」。未登记就把状态改绿会被判失败
   （`TestCommittedRepositoryStateIsHonest` 用剥离偏差的副本做反向验证）。

## 已测量的剩余视觉差异（保留，不设门禁）

`web/scripts/compare-to-reference.mjs` 对 76 组截图逐像素比对：

| 指标 | 数值 |
| --- | --- |
| 中位差异 | 31.9% |
| 低于 5% 差异 | 0 组 |
| 高度不同的组 | 10 组（多为窄屏） |
| 最大差异 | trial-balance 48.9% |

配色、字体、侧栏与主区盒完全一致（顶栏 `rgb(0,101,163)`、正文 `rgb(242,242,242)`、
`Fira Sans` 14px、侧栏 `160x748`、主区 `1120x748`）。差异集中在图表内部几何、
层级图布局、表格列宽与模态表单行距。逐项数据见
`docs/evidence/visual-parity-2026-10-28.txt`，结论与根因见
`docs/evidence/visual-parity-2026-10-28.md`。

## 本轮修复的实质缺陷（14 项）

架构级：ADR-0041 删除 parity 图表回退使 FD-0006 批准前提失效（已恢复默认 parity，
`?chart_layer=modern` 显式启用现代层）。

布局：图表 280→250px；账户页图表重复渲染（narrow 1833→1030px）；试算平衡误用两列；
层级图被 250px 压扁；图例与 interval 注释混在图表卡内。

控件：转换/区间下拉从 header 移入图表控制条（Fava 的 `ConversionAndInterval` 位置）；
补图表控制条；补 Fava 账本图标；树表头去掉 Fava 没有的 "Accounts" 文字；"Other" 列补 `.num`。

文案：报表标题改标题式大写（`Income Statement` / `Balance Sheet` / `Trial Balance`）。

语义：平均成本扩展（ADR-0042）默认插入标准路由 → 移入 `chart_layer=modern`。

## 纪律

- 不得用缩小路由清单的方式达成；`excluded` 只允许 X-EXT/X-PLUGIN/X-PYIMPORT/X-PUBLICAPI/X-HOSTING/X-PRIVATE。
- 提案型偏差（proposed）不算通过；只有产品 owner 批准才计入。
- 改变验收基准必须在偏差表登记并获批准，不得直接改门禁判据。
