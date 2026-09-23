# Beancount 兼容性验证计划（Go 复刻完成度）

- **目标**：验证 OrangeCount 是否达到"Go 版 beancount"的会计语义契约
  （ADR-0005/0008：Beancount v3 为 oracle；CLAUDE.md：UI 对齐 Fava 另行验证）。
- **现状基线**（2026-09）：`tools/reference/differential.sh` 仅对比 entry 数量与
  error 类名；ADR-0008 承诺的六个差分维度中约 1.5 维已落地；无 CI 化公开语料差分。
- **原则**：
  1. **Beancount v3 是唯一 pass/fail oracle**（经 uv 运行，GPL 源码不进仓库）；
  2. **jev-latest 只是分流/决策辅助**，其输出永不直接决定 CI 通过与否；
  3. 送往任何外部 API 的数据仅限**红acted 差分摘要**（计数、错误码、归一化差异
     类型），不含账本内容——与 ADR-0009 隐私约束及 `differential.sh` 现有脱敏
     策略一致。

## Phase 0 — 环境与密钥（半天）

- [ ] 确认 `TYPESAFE_API_KEY` 落位到本仓库（当前 `.env` 不存在于本仓库根目录）。
      `.env` 已在 `.gitignore` 中，不提交。
- [ ] 连通性冒烟：`curl` 调 `POST /v1/systemone`，`model` 固定 pin `jev-1.13.0`
      （`jev-latest` 是会漂移的别名；脚本中记录实际返回的 model 版本以便复现）。
- [ ] `uv sync --project tools/reference` 确认 beancount v3 oracle 可用。

## Phase 1 — 黄金差分语料（2–3 天）

现状 `testdata/fixtures` 只有 6 个核心 fixture，覆盖面远小于 beancount v3 语义面。

- [ ] 按特性矩阵扩充脱敏语料，每格至少 1 正 1 反（error）用例：
  - 指令全集：open/close/txn/balance/pad/commodity/event/note/document/query/custom；
  - booking：STRICT/FIFO/LIFO/AVERAGE、成本 basis、同日多笔排序敏感场景；
  - 插值：单腿插值、`abs()` 价格插值（上游历史 bug 区）、多币种；
  - options/include 循环与缺失、Unicode 账户、日期/数字方言；
  - diagnostics：每类 beancount error 至少一个触发用例。
- [ ] 用 oracle 生成**黄金快照**：对每个 fixture 由 beancount v3 导出归一化 JSON
  （entries dump、errors 类名+位置、余额表、bean-query 结果），提交进
  `testdata/golden/`（只含脱敏数据）。
- [ ] 新增 `tools/reference/differential-corpus.sh`：无需私有账本，遍历
  `testdata/fixtures`，双引擎各跑一遍 → 产出差分记录。CI 可跑（缓存 uv）。

## Phase 2 — 六维差分深化（3–5 天，对齐 ADR-0008 承诺）

每维产出一类**红acted 差分记录**（结构化 JSON，可计数、无账本文本）：

| 维度 | OC 侧 | Oracle 侧 | 对比方式 |
| --- | --- | --- | --- |
| M1 parsing/entries | `snapshot` entries dump | `beancount.loader` + printer | 归一化 entry 序列 diff |
| M2 diagnostics | error code+severity+span | error `__class__`+位置 | 映射表 diff（映射表进 compatibility-ledger） |
| M3 accounting state | 各账户各币种终态余额 | `bean-report balances` / API | 余额表 diff |
| M4 queries | `internal/query` 结果行 | `bean-query` 结果行 | 行集归一化 diff |
| M5 reports | balance/journal 渲染 | Fava/beancount 渲染 | 文本归一化 diff |
| M6 有效性判定 | exit status | errors 非空 | 布尔一致 |

- [ ] 每维一个 Go 测试入口（挂在 `internal/compat` 下，`-tags reference` 门控），
      产出 `differences.json`：`{fixture, dimension, kind, summary}` 列表。
- [ ] 通过 `docs/compatibility-ledger.json` 的 `approved_differences` 过滤已知
      边界（plugins-not-executed 等），剩余为**净差分**。

## Phase 3 — Jev 分流层（1–2 天）

新增 `tools/reference/jev-triage.py`（或 Go 小工具），消费净差分记录：

- **state**：单条差分记录 + 其所属维度 + fixture 特性标签（红acted）。
- **typed questions**（一次请求并行问完）：
  - `classification`(choice)：`real-semantic-divergence` / `approved-boundary-miss`
    / `display-only` / `formatting-noise` / `oracle-version-artifact`；
  - `blocks_parity`(noul)：是否阻碍"会计语义等价"判定；
  - `fix_priority`(score)：0–100 修复优先级。
- **阈值策略**：`blocks_parity` ≥ 0.9 → 自动登记 issue；0.5–0.9 → 人工复核队列；
  ≤ 0.5 → 记录后忽略。所有判定连同概率落盘 `differences-triaged.json`。
- **红线**：Jev 结论只影响**人工处理顺序与分诊**；CI 的 pass/fail 仍由
  Phase 2 的确定性 diff 决定。

## Phase 4 — TDD 辅助增强（与 Phase 2/3 并行）

Jev 不能写测试（无文本生成），在红-绿-重构环中承担**决策点**：

1. **红灯前**：新特性实现前，把"计划 fixture + 特性标签"作为 state 问
   `must_match_oracle`(noul) 与 `boundary_type`(choice)——高置信"必须对齐"
   → 先写 red 差分测试；高置信"intentional boundary" → 先登记
   `fava-approved-deviations.md` / compatibility-ledger，避免白写测试。
2. **失败后**：差分失败记录进 Phase 3 分流，决定修哪个、以何顺序（score 排序）。
3. **测试本身**仍由编码 agent 依据黄金快照撰写；黄金快照才是断言来源。

## Phase 5 — 验收记分卡与 CI（1 天）

- [ ] `make parity-report`：跑全语料六维差分，产出记分卡
      （每维 pass rate、净差分数、已登记边界数），落 `docs/` 或 CI artifact。
- [ ] **验收标准**：
  - M1–M6 在非边界维度 100% 一致；
  - 所有剩余差异均已在 `compatibility-ledger.json` 登记且经 product-owner 批准
    （CLAUDE.md 约束）；
  - 记分卡由确定性 diff 生成，Jev 分流记录仅作附件证据。
- [ ] CI：差分语料测试每 PR 必跑；Jev 分流为手动/夜间任务（密钥不出本机、
  不进 CI secrets 之外的环境）。

## 里程碑与工作量

| 阶段 | 内容 | 预估 |
| --- | --- | --- |
| P0 | 密钥落位 + oracle 冒烟 | 0.5 天 |
| P1 | 语料扩充 + 黄金快照 + corpus 脚本 | 2–3 天 |
| P2 | 六维差分深化 | 3–5 天 |
| P3 | Jev 分流层 | 1–2 天 |
| P4 | TDD 决策点接入 | 并行 |
| P5 | 记分卡 + CI | 1 天 |
