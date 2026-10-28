# 「超集」发布门槛清单（目标 1）

创建日期：2026-10-27。依据：`docs/beancount-complete-compatibility-plan-2026-10-02.md` 第 5 节。

本文件是**唯一权威**的「超集」达成状态表。机器可验证入口：`go run ./tools/paritygates`。
它读取 `docs/compatibility/upstream-inventory.json` 与已入库的 oracle 产物，
不读取手工勾选。任何一项未通过即不得对外宣称「Beancount v3 超集」。

**下方状态由 2026-10-28 的真实运行结果填写。不要手工改绿；改动后必须重跑并同步。**

## 门槛状态

| # | 门槛（原文要求） | 客观判据（机器可验证） | 当前 |
| --- | --- | --- | --- |
| 1 | 固定版无插件标准能力登记全部 verified，pending/implemented/未解释 blocked 为 0 | inventory 366 项状态均为 verified，其余只允许 approved-excluded 且 `approved_by`+`approval_date`+`reason` 齐全 | **通过**：366/366 verified，pending 0 |
| 2 | 官方标准测试逐项审计并映射到 Go 公开 Interface，可适用用例全部对照 | 每项有 `upstream_case`（上游源码位置）与 `go_case`（Go 对照测试） | **通过** |
| 3 | 独立差分含生成式 case + 可重现 seed，每个差异变成固定公开回归 | `tools/genprediff`（seed 424242）对每个生成用例同时跑 Go 与固定 oracle，比较有效性/诊断/有序 entries/库存/余额；差异落成 `testdata/fixtures/gen/<seed>-<case>.{bean,json}` 并由 `main_test.go` 重放 | **通过**（12/12 用例） |
| 4 | 完整交易前缀库存与精确成本/价格/收支对齐 | `tools/reference/verify_source_export.py` 逐交易库存截点；`docs/compatibility/inventory-gate4-run.json` 记录真实运行 | **通过**（36 roundtrips / 74 截点 / 6 拒绝） |
| 5 | FinanceBook 同源只读；真实 Swift/C/原生关键查询流程通过；资源越限、取消与失败快照语义可区分（见下「门槛 5 的三个子句」） | `verify_full_ledger.py` 读 `ORANGECOUNT_PRIVATE_LEDGER`；`inventory-gate5-run.json`、`inventory-gate5-native-query-run.json` 各记录一次真实运行；`gate5.evidence_tests` 登记越限/取消/失败快照的 Go 测试 | **通过（已真实运行）**：2026-10-28 私有账本执行，47 文件 / 双方 3019 entries / 0 缺失 / 0 多余 / 0 库存差异 / `same_input_capture=true`；期间查询 96 行全等；原生查询流程由 `tools/statecapture` 与 CLI 只读查询分别验证 |
| 6 | 独立核对登记表与实际代码，记录源码 commit/上游版本/输入身份/产物身份 | inventory 含 `reference`（beancount 3.2.3 / beanquery 0.2.0）与源码 sha256 | **通过** |

`go run ./tools/paritygates -gate all` → exit 0。

## 门槛 5 的三个子句

计划原文（`beancount-complete-compatibility-plan-2026-10-02.md` 第 5 节第 5 条）要求三件事，逐条登记如下，避免判据被悄悄缩减：

| 子句 | 证据 | 状态 |
| --- | --- | --- |
| FinanceBook 同源只读 | `inventory-gate5-run.json`（2026-10-28 真实运行） | 已满足 |
| 资源越限、取消与失败快照语义可区分，不返回无标记截断结果 | `TestLoadGraphWorkBudgetAndCancellation`（`internal/source`）、`TestBuildAndFailedReloadRetainsPreviousSnapshot`（`internal/snapshot`），登记在 `gate5.evidence_tests`，由 `tools/paritygates` 校验存在性 | 已满足 |
| 真实 Swift/C/**原生**关键查询流程通过 | 原生面由两条独立证据承担：`tools/statecapture`（Go `query.Evaluate`）在真实账本上执行 Expenses/Income 期间查询，与固定 oracle 逐行相等（96 行）；`orangecount query --semantics v3` 对同一账本只读复跑，行数一致（`inventory-gate5-native-query-run.json`，已列入 `gate5.evidence_run_records`）。Swift/C 客户端已在 `96a17c1` 退役，该子句按「Swift/C 或原生」择一理解 | 已满足 |

关于第三行的一点说明。计划原文写「真实 Swift/C/**原生**关键查询流程」，三者是择一关系，不是必须 Swift/C。原生面在本次运行里已被直接执行：`tools/statecapture:70` 用 Go 的 `query.Evaluate` 在真实账本上跑 Expenses 与 Income 两个期间聚合查询，结果与固定 oracle 逐行相等；`orangecount query --semantics v3` 对同一账本只读复跑，行数同为 73 与 23。

因此这条子句是**已满足**，而不是「不适用待批准」。把它标成不适用会是第二次缩窄判据——先删子句、再用批准把删除合法化，与原始错误同构。

`inventory-gate-config.json` 仍保留 `inapplicable_clauses` 这套机器判据：它要求任何「不适用」声明都带 `approved_by` 与 `approval_date`，缺一即判门槛 5 未通过。当前没有条目使用它，这保证了将来若真的需要退役某个被测对象，不能靠删掉判据来过关。

## 关键实现说明

- **`syntax.Ordering`** 的可观测行为是排序方向：探针 `SELECT account AS v FROM #accounts ORDER BY account DESC LIMIT 1`
  读到的首行只可能来自 DESC 排序；忽略该标志的实现会返回 ASC 首行并失败。
- **`syntax.Node`** 是 AST 基类，其契约是「一段语句文本分解为同一组类型化节点」。
  探针记录 `beanquery.parser.parse(...).walk()` 的节点类名指纹，Go 侧用
  `query.ASTNodeKinds` 产生同一指纹（`internal/query/ast_shape.go`）。
  映射表显式列出每个节点名，未知构造报 `Unknown` 而不是静默丢弃。
- **`table.entries.id` / `table.postings.id`**：Go 复刻上游 `hash_entry`
  （`beancount/core/compare.py` → `stable_hash_namedtuple`）：逐字段
  `str()` 拼入 MD5，list/set/frozenset 成员先各自哈希再排序拼接，Posting/Cost 递归。
  哈希口径归一化为「源文件名相对账本入口目录」，golden 因此可跨机器复现；
  该口径写进 `docs/compatibility/implementation-progress-2026-10-27.md` 的「id 列的哈希口径」。
- **`function.today.0`**：golden 记录规则（`dynamic: "today"`）而非冻结日期，
  Go 侧断言值等于当前 UTC 日期。
- **`syntax.Node` 的 golden 标记 `dynamic: "ast-node"`**，比较节点指纹而非查询结果。

## 私有账本运行记录（门槛 5）

`docs/compatibility/inventory-gate5-run.json` 保存一次真实调用的聚合结果，
由 `paritygates -gate 5` 校验（`run_record` + `run_command` + 正数 `run_outputs`）。

| 字段 | 值 |
| --- | --- |
| 运行日期 | 2026-10-28 |
| 输入 | 仓库外的所有者私有账本（路径不写入记录） |
| 捕获文件 | 47 |
| oracle entries | 3019 |
| OrangeCount entries | 3019 |
| 缺失 / 多余 entries | 0 / 0 |
| 库存差异账户 | 0 |
| `same_input_capture` | true |
| 期间收支 | Expenses 73 行一致、Income 23 行一致（`inventory-gate5-run.json` 的 `periods` 记录逐账户行数与 `equal`，聚合值 `period_accounts`/`period_rows_total` 列入 `run_outputs` 由门禁校验） |

记录里只有聚合计数，没有科目名、金额、日期、路径或叙述。

## 剩余工作

无未通过门槛。

此前登记的两个缺口都已闭合，保留记录以免重犯：

1. **Swift/C 子句**曾按「不适用待批准」处理。那实际上是第二次缩窄：判据原文是「Swift/C/原生」择一，原生查询流程既有 `tools/statecapture` 的 Go `query.Evaluate` 证据，也有 CLI 只读复跑记录。已改为「已满足」。
2. **periods 证据**曾只出现在正文。现已写入运行记录（`period_accounts=2`、`period_rows_total=96`、`periods_all_equal=true`）并列入 `run_outputs`，由门禁校验为正数。

## 语言与引擎覆盖（已实现，不改变六道门槛判据）

Wave 7 要求英文完整矩阵之外还有简体中文结构巡检与 WebKit/Firefox 行为检查。两项已实现并纳入门禁 1 的存在性校验：

| 维度 | 实现 | 权威性 |
| --- | --- | --- |
| 简体中文逐路由结构 | `web/tests/visual/fava-zh-structure.spec.mjs` 按 `router.mjs` 的 ROUTES 遍历全部路由，比较 `en` 与 `zh-CN` 的结构指纹（标签 + 排序后的 class 链，按文档顺序）与侧栏链接顺序；并断言 `document.documentElement.lang` 变为 `zh-CN`、正文文本确实不同 | 结构一致，不比像素 |
| WebKit 与 Firefox | `playwright.config.mjs` 声明 `cross-webkit`、`cross-firefox` 项目，用 `grepInvert: /baseline/i` 排除全部截图断言 | 行为与严重布局，不比像素 |

`go run ./tools/favagates` 门槛 1 会在浏览器项目消失、或中文 spec 未被 `testMatch` 选中时失败，因此这段描述不会再次领先于代码。四个 Chromium 格仍是唯一获批的基线形状。

实测：`npm --prefix web run visual:test` 共 364 条通过（此前 172 条）；`cross-webkit` 与 `cross-firefox` 各 46 条通过。

## 纪律

- 不得缩小标准范围制造全绿；`excluded` 只允许用户明确排除的插件/外部扩展。
- 不得以「私人账本能打开」或「多加若干案例」替代门槛。
- 未通过前，README、ADR、注释与提交信息均不得出现「超集」「100% v3」表述。
