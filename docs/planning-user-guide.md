# OrangeCount Planning 使用指南

这篇文章面向第一次接触 OrangeCount Planning 的用户，从“它解决什么问题”开始，逐步带你完成：初始化规划、配置安全池、添加计划现金流、读懂“安全可支配金额”、试算大额支出，以及进行周期复盘。

如果你只想先跑通一遍，可以跳到[“最快上手流程”](#最快上手流程)。想深入理解背后的账本表示和计算规则，再看后面的章节。

## 1. Planning 解决什么问题

Planning 是 OrangeCount 里的一个**保守可支配资金规划**模块。它回答两个问题：

1. 在下一笔稳定收入到来前，我现在手里有多少钱是“安全可花”的？
2. 如果我想购买一件大额商品，这次试算会不会违反我已知的支出计划或最低保留金？

它不替你做决定，也不自动给你推荐“该买还是不该买”。它把所有假设明确列出来，由你来确认。

官方概念见：

- [financial-planning-design-plan.md](financial-planning-design-plan.md)：产品设计、计算规则、决策记录。
- [ADR-0044](adr/0044-embed-planning-assumptions-in-the-ledger.md)：规划数据如何放进账本。
- [ADR-0045](adr/0045-build-liquidity-planning-before-category-budgets.md)：为什么先做流动性规划，不做分类预算。

## 2. 核心概念

### 2.1 规划配置（Planning Profile）

一个规划必须有且只有一个“规划配置”。配置声明：

| 字段 | 含义 | 说明 |
| --- | --- | --- |
| `currency` | 规划币种 | 只使用这一个币种计算；其他币种可见但不算进去。 |
| `timezone` | 规划时区 | “今天”按这个时区判断，默认初始化建议用机器时区。 |
| `minimum_reserve` | 最低保留金 | 你要求始终留在池子里、不能视为可花的钱。 |
| `spendable_account` | 安全池账户 | 参与计算的资金账户，按你的明确选择进入。 |
| `short_term_debt_account` | 短期债务账户 | 需先扣除的债务（如信用卡、短期借款）。 |
| `recorded_through` | 记账截至日期 | 你自认为真实活动已记到账本的日期。 |

**原则：不猜。** 如果配置不完整，页面只显示 checklist，不给你一个“看似可用”的数字。

### 2.2 计划现金流（Planned Flow）

一个计划现金流最少包含：

- 名称；
- 预计日期；
- 规划币种金额；
- 方向：`inflow`（流入）或 `outflow`（流出）；
- 如果是流出：`committed`（已承诺）或 `adjustable`（可调整）。

可选地还可以指定预期账户 `account`，方便后续匹配实际流水。

**计划不是交易。** 它只是“我确认会在某个日期发生一笔资金进出”的假设，不会修改你的账本余额。

### 2.3 安全可支配金额（Primary Safe-to-Spend）

安全可支配金额是一个日期序列上的**最低点**，不是期末余额，也不是平均值：

```text
某日 headroom
= 当前安全池合计
- 当前短期债务合计
- 截至该日所有已承诺流出
- 截至该日所有可调整流出
- 最低保留金
```

安全可支配金额 = 整个时间线上最低的那个 `headroom`。

两条保守规则：

- 未来收入**不会**提高主结果，只会在时间线上“可见但忽略”。
- 同日发生流出时，先扣流出，后看流入。

如果最低点是负的，页面会把安全可支配金额显示为 **0**，并同时显示绝对资金缺口、缺口最早发生日期和当时的假设。

### 2.4 复盘（Cycle Review）

复盘阶段会从账本真实流水里给出两类“证据”：

- **计划-实际匹配建议**：把计划中的流出和账本里同名的实际消费配对，仅供你确认，不会自动完成计划。
- **周期性候选**：根据本地、确定性的历史模式，告诉你某个描述和金额看起来像固定周期支出。你可以选择把它“用作计划草稿”，再自己确认。

任何一个候选都不直接成为计划。你确认、预览，并通过受保护写入后才生效。

## 3. 最快上手流程

假设你的账本已经在 `ledger/main.bean`，里面至少有一个资产账户和权益账户。打开本地 Web：

```sh
./bin/orangecount serve --addr 127.0.0.1:5000 ledger/main.bean
```

浏览器进入 `http://127.0.0.1:5000/planning`。

第 1 步：**填写规划配置**

页面处于 “Planning setup required” 时，会显示：

- Currency：例如 `CNY`；
- Timezone：例如 `Asia/Singapore`；
- Minimum reserve：你想始终保留的金额；
- Spendable accounts：用逗号分隔的安全池账户；
- Short-term debt accounts：用逗号分隔的短期债务账户（可选）；
- Recorded through：你确认已记账到哪天。

填好后点 **Preview**，预览生成的 `custom` 指令，确认后点 **Commit**。

第 2 步：**添加计划现金流**

在 “Add a planned flow” 区域填写：

- Plan ID：稳定 ID，例如 `rent-2026-09`；
- Name：显示名称，例如 `房租`；
- Amount：金额；
- Date：预计日期；
- Direction：`outflow` / `inflow`；
- Commitment：只有流出需要选 `committed` / `adjustable`；
- Account：可选的来源/去向账户。

点 **Preview**，查看将要追加到账本的 Beancount 指令，点 **Commit** 生效。

第 3 步：**看结果**

页面头部会显示“Safe to spend”，旁边是时间线表格。每一行都是一天或一个计划日期的 `headroom`。

第 4 步：**试算大额支出**

在 “Test a large purchase” 中填写名称、金额、日期，勾选是否需要“计入未来流入”或“释放可调整流出”，点 **Evaluate**。结果不会写入账本，只是告诉你如果这么做，试算最低点会是什么样子。

第 5 步：**周期复盘**

点 “Inspect review evidence”：

- 查看计划-实际匹配建议；
- 勾选你确认的匹配；
- 查看周期性候选；
- 点 “Use as a plan draft” 会把候选带入新增计划表单；
- 点 “Preview review completion”，确认后点 “Save reviewed completion”。

复盘完成记录会保存到账本；如果之后账本内容变了，该记录会被标记为 stale，需要重新确认。

## 4. Planning 在账本里长什么样

所有规划配置都使用标准的 Beancount `custom` 指令，因此不会影响会计余额。

### 4.1 规划配置

```beancount
2026-08-12 custom "orangecount.planning-profile.v1" "primary"
  currency: "CNY"
  timezone: "Asia/Singapore"
  minimum_reserve: 20000 CNY
  spendable_account: Assets:CMB:Checking
  spendable_account: Assets:WeChat
  short_term_debt_account: Liabilities:CMB:CreditCard
  recorded_through: 2026-08-12
```

注意：

- `currency` 是字符串，`minimum_reserve` 是金额；
- `spendable_account` 可以重复出现；
- `recorded_through` 写的是“你记账记到哪天”，不是写这条自定义指令的日期。

### 4.2 新增一个流出计划

```beancount
2026-08-12 custom "orangecount.planned-flow.v1" "rent-2026-09"
  revision: 1
  name: "房租"
  direction: "outflow"
  expected_date: 2026-09-01
  amount: 5000 CNY
  commitment: "committed"
  status: "active"
```

### 4.3 新增一个流入计划

```beancount
2026-08-12 custom "orangecount.planned-flow.v1" "salary-2026-09"
  revision: 1
  name: "工资"
  direction: "inflow"
  expected_date: 2026-09-10
  amount: 30000 CNY
  status: "active"
```

### 4.4 修订、取消、履行

计划采用**只追加修订**：不要修改旧指令，而是追加更高 `revision` 的新指令。

取消：

```beancount
2026-08-13 custom "orangecount.planned-flow.v1" "rent-2026-09"
  revision: 2
  status: "cancelled"
```

标记已履行：

```beancount
2026-08-13 custom "orangecount.planned-flow.v1" "rent-2026-09"
  revision: 2
  status: "fulfilled"
```

改期：保留 `status: "active"`，只改 `expected_date` 等字段：

```beancount
2026-08-13 custom "orangecount.planned-flow.v1" "rent-2026-09"
  revision: 2
  name: "房租"
  direction: "outflow"
  expected_date: 2026-09-05
  amount: 5000 CNY
  commitment: "committed"
  status: "active"
```

界面上的 “Resolve an active plan” 会自动帮你生成这些修订指令，你不需要手写。

### 4.5 复盘完成记录

```beancount
2026-08-13 custom "orangecount.cycle-review.v1" "completed"
  ledger_fingerprint: "……"
  match: "rent-2026-09:12-0"
```

这是“确认事实”的快照。后续来源变化会导致复盘记录变 stale，但不会把历史抹掉。

## 5. 大额支出试算的规则

试算永远不会写账本。你提交的是：

- `horizon_end`：规划终点；
- `name`：拟购名称；
- `amount`：金额；
- `date`：拟购日期；
- `include_future_inflows`：是否把未来流入也视为可支付来源；
- `release_adjustable`：是否假定可调整流出可以被释放。

默认（两个选项都不勾）是最保守的：只用现有资金、保留所有计划。勾选选项后，页面会明确指出你放松了哪一条假设。

示例：

- 不勾任何选项：结果等于“现状 + 这个购买在日期上扣一次金额”后的最低点；
- 勾 `include_future_inflows`：结果可能变好，因为它把未来流入也加进计算；
- 勾 `release_adjustable`：结果可能变好，因为可调整流出不再全部占用名额。

## 6. 周期复盘怎么理解

复盘页提供的都是**供你确认的证据**：

| 类型 | 含义 | 你需要做什么 |
| --- | --- | --- |
| 计划-实际匹配建议 | 某个计划和某笔账本消费在名称、账户、金额、日期窗口上“像同一件事” | 人工判断是否真的是同一笔，勾选后加入完成记录 |
| 周期性候选 | 本地历史中重复出现的 name/account 组合，间隔固定、金额有上下界 | 决定是否把它作为下一个周期计划草稿 |
| 完成/过期状态 | 之前的复盘确认是否仍和当前账本一致 | stale 时重新确认 |

这些匹配不是自动完成。未处理的过期流出会继续占用资金名额，直到你显式标记履行、取消或改期。

## 7. 常见误区

误区 1：**“我记了账，Planning 就会自动知道我要付什么。”**

不会。安全池账户和计划都要你显式配置。记现金流 ≠ 自动变成规划。

误区 2：**“未来工资会提高安全可支配金额。”**

主结果不会。未来流入默认不计入主结果；只有试算时显式勾选才纳入。

误区 3：**“A 账户没钱，但安全池里别的账户有钱，所以没法买。”**

聚合视角下，只要整个安全池在时间线的每个点上都够，就还有 headroom；账户之间的转账是账户流动性问题，不是聚合缺口问题。

误区 4：**“我把计划取消了，历史就消失了。”**

不会。取消是通过追加更高 revision 实现的，旧版本和旧计划记录仍保留在账本。

误区 5：**“试算通过就等于可以买。”**

不等于。Planning 只告诉你约束下还剩多少 headroom、会不会出现资金缺口，不做消费建议。

## 8. 当前边界

当前版本不包含：

- 银行/微信/支付宝自动抓取；
- 自动把实际支出匹配成计划（只给建议）；
- 自动创建计划或自动改期；
- 跨币种换算；
- 分类预算；
- 通知、提醒或自动转账；
- 概率预测或外部 AI 评分。

如果你需要机构文件的获取思路，可参考：
[china-statement-export-research.md](china-statement-export-research.md)。

## 9. 一句话总结

Planning 是一个“你确认假设，它给你解释与试算”的工具：先选安全池和债务，再列出你确认的未来现金计划，最后看时间线上的最低 headroom 以及一笔大额购买会不会让低点跌破保留金或变成缺口。
