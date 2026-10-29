# OrangeCount 使用指南

这份文档面向**账本使用者**：你想校验一个 Beancount 账本、查一个数字、看一张报表、记一笔账、或者把账本交给别的工具。全部按"我想做什么"组织，每条命令都在这台机器上实测过。

如果你要改代码，看 [development.md](development.md)。想了解 Fava 界面移植和对齐范围，看 [fava-acceptance-matrix.md](fava-acceptance-matrix.md)。规划模块单独有一篇 [planning-user-guide.md](planning-user-guide.md)。

---

## 1. 它能做什么

| 能做 | 说明 |
| --- | --- |
| 校验账本 | 读入整个 include 图，报语法错和会计错，给中文修复建议 |
| 查询 | 用类 SQL（BeanQuery 子集）查分录、余额、价格 |
| 报表 | 资产负债表、损益表、试算平衡、持仓、商品、事件、统计 |
| 写入 | 网页工作台里新增/编辑条目、快速录入、导入，带备份和回滚 |
| 导出 | 导出纯 Beancount v3 源快照给 Fava 等工具；查询结果导出 XLSX/ODS |

**不能做**：不执行 Python plugin，不联网取价，不做税务申报，不替你决定账务处理。

核心设计：**你手写的 `.bean` 是唯一真相**。没有隐藏数据库，报表全部由源文件现算。

---

## 2. 安装

```sh
cd /Users/breestealth/Development/OrangeCount
make build
ln -sf "$PWD/bin/orangecount" ~/.local/bin/orangecount   # 可选，装成全局命令
```

只需要 Go，不需要 Python、Node 或容器。之后示例里的 `orangecount` 都可用 `bin/orangecount` 代替。

---

## 3. 五分钟上手

```sh
cd ~/Documents/FinanceBook
orangecount check --semantics v3 --locale zh-CN main.bean
```

没有输出、退出码 0，就是通过。

```sh
orangecount serve --semantics v3 --locale zh-CN --addr 127.0.0.1:5100 main.bean
```

浏览器打开 <http://127.0.0.1:5100>。**端口要换**：macOS 的 AirPlay 接收器占着默认的 5000。

---

## 4. 账本怎么组织

- **入口文件**：含 `include` 的那个 `.bean`。所有命令都传它，工具自己顺着 include 走。
- **备份**：网页写入会留 `<文件名>.orangecount.bak`。把这个模式加进 `.gitignore`，别提交。
- **只读保证**：`check`、`query`、`serve` 都不写账本文件。只有你在网页里保存、提交快速录入、导入时才写。

---

## 5. 校验

```sh
orangecount check --semantics v3 main.bean                    # 英文
orangecount check --semantics v3 --locale zh-CN main.bean      # 中文
orangecount check --semantics v3 --json main.bean              # 机器可读
```

**看退出码，不要找"0 errors"提示**。通过时一个字都不打印。

| 退出码 | 含义 |
| --- | --- |
| 0 | 无 error 级诊断 |
| 1 | 至少一条 error |
| 2 | 参数用法错误 |

每条诊断长这样：`路径:行:列: error 代码`，例如

```
main.bean:42:3: error E-EVAL-POSTING: posting references an account outside its open lifecycle
  -> Align the posting with the account lifecycle (help: diagnostics/E-EVAL-POSTING)
```

### 看某个错误码怎么修

```sh
orangecount help diagnostics/E-EVAL-POSTING
orangecount help --locale zh-CN diagnostics/E-EVAL-ACCOUNT
```

每个码都有：属于哪个阶段、什么含义、该查什么、安全步骤、改前改后示例。**先读指引再动手**，不要凭一行消息猜修法。

### 常见错误码

| 代码 | 含义 | 常见原因 |
| --- | --- | --- |
| `E-EVAL-ACCOUNT` | 根名不是已配置的账户类型，或账户名非法 | 写了 `Acme:X`；中文根名但没开 `account_root_scheme` |
| `E-EVAL-POSTING` | 记账日期在账户开户前或销户后 | 补记历史忘了补 `open` |
| `E-EVAL-UNBALANCED` | 交易不平衡 | 金额写错、漏一条腿 |
| `E-EVAL-INFER` | 省略的金额推不出来 | 两条腿都省略且币种不唯一 |
| `E-EVAL-BALANCE` | 余额断言失败 | 期初没对齐，或断言金额过时 |
| `E-EVAL-CURRENCY` | 账户不允许该币种 | `open` 时列了币种，记账用了别的 |
| `E-EVAL-INVENTORY` | 卖出的库存超过持有量 | 先买后卖顺序错，或成本批次不符 |
| `E-EVAL-PAD` / `E-EVAL-PAD-UNUSED` | pad 来源不可用 / pad 没被余额断言用到 | pad 和断言没配对 |
| `E-EVAL-OPTION` | 选项不支持、值非法、根名冲突 | 拼错 `name_*`，或两种类型同名 |
| `E-PARSE-DIRECTIVE` | 指令未知或格式错 | 关键字拼错，如 `opne` |
| `E-SOURCE-UTF8` | 文件不是合法 UTF-8 | 从 Windows 复制来的 GBK 文件 |

---

## 6. 会计语义：legacy 与 v3

**这一条最容易踩。** `check`、`query`、`serve`、`export` 都有 `--semantics` 参数，**默认是 `legacy`**。

| 配置 | 适用 |
| --- | --- |
| `legacy` | 早期未迁移账本 |
| `v3` | Beancount v3 语义，**推荐** |

同一个账本在两种配置下数字可能不同。差异来自 booking 标注、pad 语义、容差推断等。别猜，直接比：

```sh
orangecount compare-semantics main.bean                     # 汇总
orangecount compare-semantics --details main.bean            # 逐条明细
orangecount compare-semantics --fail-on-change main.bean     # 有差异退出 1，可放进脚本
```

**习惯**：所有命令都显式写 `--semantics v3`。忘了写会静默用 legacy。

---

## 7. 查询

```sh
orangecount query --semantics v3 main.bean "SELECT account, sum(number) AS total FROM postings GROUP BY account ORDER BY total DESC"
```

### 两种引擎，列不一样

| `--format` | 引擎 | `accounts` 表有哪些列 |
| --- | --- | --- |
| `json` / `csv` | 兼容查询 | `account`, `currency`, `balance`, `opened` |
| `typed-json` / `typed-csv` | 类型化 BeanQuery 子集 | 只有 `account` |

类型化引擎的 `postings` 表有：`account`、`currency`、`number`、`balance`、`date`、`year`、`month`、`day`、`payee`、`narration`、`flag`、`tags`、`links`、`cost`、`weight`、`price`、`id`、`type`。

`--json` 和 `--csv` 是**兼容格式**的别名，不等于 typed。

### 实测可用的查询

```sql
-- 各账户余额排行
SELECT account, sum(number) AS total FROM postings GROUP BY account ORDER BY total DESC

-- 各支出账户合计（正则过滤；没有 LIKE）
SELECT account, sum(number) AS total FROM postings WHERE account ~ '^Expenses' GROUP BY account ORDER BY total DESC

-- 按月支出
SELECT year, month, sum(number) AS total FROM postings WHERE account ~ '^Expenses' GROUP BY year, month ORDER BY year, month

-- 每个账户每个币种的余额（要 --format csv，走兼容引擎）
SELECT account, currency, sum(balance) AS balance FROM accounts GROUP BY account, currency ORDER BY account

-- 某段时间的流水
SELECT date, narration, account, number, currency FROM postings WHERE date >= '2026-01-01' ORDER BY date

-- 计数
SELECT count(*) AS n FROM postings
```

### 三条硬规则

1. **不同币种不能相加**。库存类数值跨币种求和没有意义。要按 `currency` 过滤，或只查一个账户。
2. **没有 `LIKE`**。用 `~` 做正则匹配。
3. **金额是精确十进制**。不要当浮点处理，也不要自己重排小数位。

### 导出查询结果到表格

```sh
orangecount query --format csv --semantics v3 main.bean "..." > out.csv
```

XLSX/ODS 只能从工作台的 HTTP 接口拿（见下一节）。

---

## 8. 报表与工作台

### 启动

```sh
orangecount serve --semantics v3 --locale zh-CN --addr 127.0.0.1:5100 main.bean
```

| 参数 | 作用 |
| --- | --- |
| `--addr` | 只接受回环地址；写 `0.0.0.0` 会被拒绝 |
| `--document-root` | 把票据目录指到账本目录之外 |
| `--locale` | `en` 或 `zh-CN` |
| `--sensitive-logs` | 只在本地排障时用，可能把账本内容打进日志 |

先确认端口空闲：`lsof -iTCP:5100 -sTCP:LISTEN -n -P`，没输出就是空的。

### 页面

```text
/                        损益表（首页）
/income_statement        损益表    /balance_sheet   资产负债表
/trial_balance           试算平衡  /journal         日记账
/query                   查询      /holdings        持仓

/holdings/by_account  /holdings/by_currency  /holdings/by_cost_currency
/holdings/by_root_account  /holdings/by_commodity

/commodities  /documents  /events  /statistics
/editor       /import     /options /help
/source       /diagnostics /errors /quick-profile /planning
```

### HTTP 接口

只读接口，可直接用 curl 或脚本调：

```text
GET /api/v1/status
GET /api/v1/diagnostics
GET /api/v1/source
GET /api/v1/options
GET /api/v1/reports/balance-sheet?format=json
GET /api/v1/reports/income-statement?format=json
GET /api/v1/reports/trial-balance?format=json
GET /api/v1/query?q=<查询>&format=json|csv|xlsx|ods
```

报表名也接受下划线写法（`balance_sheet`）。查询接口支持直接下表格：

```sh
curl -sG 'http://127.0.0.1:5100/api/v1/query' \
  --data-urlencode "q=SELECT account, sum(number) AS total FROM postings WHERE account ~ '^Expenses' GROUP BY account" \
  --data-urlencode format=xlsx -o expenses.xlsx
```

### 一个重要限制

报表来自**启动时加载的快照**加上之后每次写入。你在工作台外面直接改了 `.bean`，界面不会自动看到，需要让它重载。判断依据是 `/api/v1/status`。

---

## 9. 写入

**没有命令行写入命令。** 两条路径，安全模型相同。

### 路径 A：工作台里的受控流程

| 做什么 | 在哪 |
| --- | --- |
| 新增一条标准指令 | 右上 `+` → Add Entry |
| 快速记几笔两腿交易 | `+` → Quick 标签 |
| 带语法高亮改源文件 | `/editor` |
| 导入文件并核对差异 | `/import` |
| 上传或移动票据 | `/documents` |

快速录入有两套严格语法，不做自然语言理解：

```text
# 显式形态
28 CNY @微信 -> @餐饮 : 工作午餐 #报销

# 模板形态
午餐 28 @微信
```

生成的两条腿都带明确金额和币种，符号按资金流向，不写入任何私有元数据。别名解析不唯一时**报错，不猜**。先预览再提交，提交是一次原子追加，也是本次会话可撤销的最后一步。

### 路径 B：直接改源文件

适合表单表达不了的内容：`metadata`、`custom` 指令、批量调整。

```sh
orangecount check --semantics v3 main.bean    # 1. 先确认基线是干净的
cp main.bean main.bean.manual.bak             # 2. 自己的后悔药
# 3. 改文件，保持周围格式
orangecount check --semantics v3 main.bean    # 4. 改完必须还是干净的
```

规则：

- 保持原文件风格：同样的日期写法、两空格缩进、同样的元数据顺序。
- 新交易放到日期相近的邻居旁边。求值按日期排序，位置不影响会计，但影响人看。
- 一次只改一处，不要顺手格式化无关行。
- 别把备份提交进仓库。

### 写入安全模型

每次写入都做：全量重新求值 → 失败就从备份回滚并返回诊断 → 成功才发布新快照。

### 不要做的事

- 为了消除账户错误去改 `option`。通常错的是账户。
- 因为余额断言失败就删交易。要修断言或修分录，并先确认哪个才是错的。
- 在编辑器里有未保存内容时从外部改文件。
- 没先 `check` 就写入。否则你分不清是自己改坏了还是本来就坏。

---

## 10. 账户根名与多语言

账本的五个一级类型默认是 `Assets` / `Liabilities` / `Equity` / `Income` / `Expenses`。报表、图表、账户类型查询全部按这五个分类。

### 用中文（或其他语言）根名

在账本里加一行：

```beancount
option "account_root_scheme" "zh-Hans"
```

| 值 | Assets | Liabilities | Equity | Income | Expenses |
| --- | --- | --- | --- | --- | --- |
| `global`（默认） | Assets | Liabilities | Equity | Income | Expenses |
| `zh-Hans` | 资产 | 负债 | 权益 | 收入 | 支出 |
| `zh-Hant` | 資產 | 負債 | 權益 | 收入 | 支出 |
| `ja` | 資産 | 負債 | 純資産 | 収益 | 費用 |
| `ko` | 자산 | 부채 | 자본 | 수익 | 비용 |

之后可以这样写：

```beancount
2020-01-01 open 资产:钱包:微信 CNY
2020-01-01 open 收入:工资 CNY
```

要点：

- **英文五种在任何方案下都照常可用**，所以老账本不用改名就能启用。
- `global` 下写中文根名会报 `E-EVAL-ACCOUNT`，并提示需要这个 option。
- 同一个账本里两种写法可以混用，报表两边都会出现。
- 选项值接受常见写法：`cn`、`tw`、`jp`、`kr`、`简体中文`、`繁體中文`、`日本語`、`한국어` 等。写错的值得报 `E-EVAL-OPTION`，不会静默回退。

### 自定义根名

如果不想用内置词汇表，用 `name_*` 逐个改名：

```beancount
option "name_assets"      "Actifs"
option "name_liabilities" "Dettes"
option "name_equity"      "Capitaux"
option "name_income"      "Revenus"
option "name_expenses"    "Depenses"
```

改了以后旧的英文根名就不再生效。两个类型改成同名会报 `E-EVAL-OPTION`。

### 一个限制

启用本地化方案后**不能导出为纯 v3**——上游既拒绝这个 option 键，也只认 ASCII 根名。要从导出，账本必须用 `global`。

---

## 11. 导出

### 导出纯 v3 源快照

给 Fava、`beancount` 命令行等外部工具用：

```sh
mkdir -p /path/outside/the/ledger
orangecount export --semantics v3 -out /path/outside/the/ledger/book.bean main.bean
```

- 输出目录必须在账本入口目录**之外**。单文件可替换已有普通文件；有 include 的账本要输出到一个尚不存在的目录。
- 替换单文件时旧文件保留为 `<文件名>.orangecount.bak`，不会被删除。这样目标处无论是普通文件还是 Finder 替身，原内容都还在。
- 拒绝无法在 v3 表达的账本：AVERAGE booking、Python plugin、未登记 option、list/map 元数据、本地化根名方案。
- 发布前会把输出重新加载一遍并比对会计结果，所以拒绝时目标文件保持原样。
- `option "account_root_scheme" "global"` 这一行会从快照里删掉，因为上游不认这个键。

完整规则见 [source-export.md](source-export.md)。

### 把方言简写还原成标准 v3

```sh
orangecount dialectize -out /path/outside/the/ledger/book.bean main.bean
```

同样要求 `-out`，同样的保护。

---

## 12. 故障排查

| 现象 | 原因与处理 |
| --- | --- |
| `check` 无输出但退出码 1 | 有 error 级诊断但被 locale 或输出吞了？用 `--json` 看全量 |
| 报表数字和命令行不一致 | `serve` 没加 `--semantics v3`，用的是 legacy |
| 中文账户报 `E-EVAL-ACCOUNT` | 没加 `option "account_root_scheme"`，或值为 `global` |
| 查询过滤 `^Assets` 查不到东西 | 账本用了本地化根名，得过滤 `资产`/`資產`/`資産`/`자산` |
| `accounts.currency` 报 unknown column | 用了 typed 格式；改用 `--format csv` |
| `LIKE` 报语法错 | typed 引擎不支持，改 `~` |
| 端口起不来 | 5000 被 AirPlay 占用，换 5100 以上 |
| `serve` 拒绝启动，说 non-loopback | `--addr` 只能写回环地址 |
| 导出一句话失败 | 输出目录在账本目录内，或账本用了本地化根名/插件 |
| 网页写入后账本多出 `.bak` | 正常行为，加进 `.gitignore` |

---

## 13. 命令速查

```sh
# 校验
orangecount check --semantics v3 [--locale zh-CN] [--json] main.bean

# 修复指引
orangecount help [--locale zh-CN] diagnostics/<CODE>

# 查询
orangecount query --semantics v3 [--format json|csv|typed-json|typed-csv] main.bean "<SQL>"

# 语义对比
orangecount compare-semantics [--details] [--fail-on-change] main.bean

# 工作台
orangecount serve --semantics v3 [--locale zh-CN] [--addr 127.0.0.1:5100] \
  [--document-root DIR] [--sensitive-logs] main.bean

# 导出纯 v3 快照 / 还原方言
orangecount export --semantics v3 -out <目标> main.bean
orangecount dialectize -out <目标> main.bean
```

---

## 14. 给它加技能（Agent 用）

仓库内置三个项目技能，供 pi 之类的编码助手自动调用：

| 技能 | 干什么 | 触发例子 |
| --- | --- | --- |
| `orangecount-ledger` | 校验、诊断解读、查询、语义对比 | "帮我检查这个账本"、"这个报错怎么修"、"上个月支出多少" |
| `orangecount-authoring` | 安全写入：新增/编辑条目、快速录入、导入 | "记一笔 28 元午饭"、"改一下这笔的金额" |
| `orangecount-workbench` | 启动工作台、报表接口、XLSX/ODS、导出 v3 | "打开工作台"、"导出成表格"、"导出给 Fava" |

放在 `.agents/skills/`，pi 启动时自动发现。手动调用：`/skill:orangecount-ledger`。
