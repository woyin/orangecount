# 上游能力能力级证据链（2026-10-27）

本文件记录 `docs/compatibility/upstream-inventory.json` 366 项能力的**能力级**验证方式与当前结论。它不替代 `docs/parity-gates-2026-10-27.md` 的发布门槛；门槛仍以该文件为准。

## 证据链结构

四项资产构成一条可复查链路，缺任一项都不能把条目记为 `verified`：

1. `tools/reference/triage_inventory.py` → `docs/compatibility/inventory-triage.json`
   逐项分诊为 `probe`（固定输入在 oracle 上直接求值）、`needs-fixture`（需要账本夹具）、`internal`（上游内部结构，无独立公开行为）。
2. `tools/reference/probe_inventory.py` → `testdata/golden/inventory/<id>.json`
   在固定 oracle（beancount 3.2.3 / beanquery 0.2.0）上真实求值，记录类型与值。求值失败如实记录 `status:"error"` 与错误类别，不写假值。
3. `internal/compat/inventory.go` + `internal/compat/inventory_test.go`
   用同一 SQL 文本在 Go 引擎执行并比较类型与行值，同时要求 `go_location` 指向真实产生该行为的语句。
4. `tools/reference/apply_inventory_verification.py`
   只把「golden 存在 + Go 对照通过 + go_location 非空」的条目写入 `verified`，其余保持原状态，且绝不写 `excluded`。

复现命令：

```sh
uv run --offline --project tools/reference python tools/reference/triage_inventory.py --check
uv run --offline --project tools/reference python tools/reference/probe_inventory.py --check
go test ./internal/compat -run TestInventoryGoldenCapabilities
```

## 探针口径

- 列与语法：`SELECT <expr> AS v FROM #<table>`，比对声明类型与全部行值；集合类列按声明类型比较，行值按集合多重集比较；Decimal 只比较数值量级，不比较书写精度。
- 函数与运算符：用类型签名构造最小合法输入。SQL 语法没有 dict/list/set/amount/position/inventory 字面量，因此这些操作数来自表中的真实列。
- 记账选项与指令：使用专用夹具（`fixture.bean`、`discrepancy.bean`、`directives.bean`、`options.bean`），记录 loader 结果与受该选项/指令影响的行值。
- `today()` 依赖当日日期，只比对声明类型；golden 记录 `declared_type_consistent` 说明该限制。
- 当上游声明类型与其实际返回值不一致（例如 `relativedelta - relativedelta` 声明 `date` 却返回 `relativedelta`），golden 记录该不一致，比对按实际值类型进行。

## id 列的哈希口径

`entries.id` 与 `postings.id` 复刻 `beancount.core.compare.hash_entry`：对该指令每个字段的 Python `str()` 拼写做 MD5；list 与 set 字段改为逐成员 MD5 后排序拼接；namedtuple 成员（Posting、Cost）递归处理。

与上游唯一的口径差异是 **源文件名归一化**：上游把 `meta['filename']` 的绝对路径直接纳入哈希，因此同一账本换目录会得到不同 id。本实现（以及 oracle 抓取）先把文件名换算为相对账本入口目录的路径再哈希，使 id 与检出位置无关。该归一化在 oracle 侧和 Go 侧一致应用，go_location 指向 `internal/query/directive_id.go`。

Go 侧需要与 Python `str()`/`repr()` 完全一致的拼写（Decimal、date、dict 顺序、Posting/Cost 元组、空 set 与 None 的区别），相关渲染集中在 `internal/query/directive_repr.go`。

## 当前结论

| 类别 | 总数 | verified | 未完成 |
| --- | --- | --- | --- |
| function | 116 | 116 | 0 |
| operator | 91 | 91 | 0 |
| column | 87 | 87 | 0 |
| accounting-option | 35 | 35 | 0 |
| syntax | 25 | 25 | 0 |
| accounting-directive | 12 | 12 | 0 |
| 合计 | 366 | 366 | 0 |

`syntax.Node` 与 `syntax.Ordering` 的验证方式：

- **Ordering**（`ASC=0`、`DESC=1` 的 IntEnum）唯一的可观测效果是排序方向。
  探针执行 `SELECT account AS v FROM #accounts ORDER BY account DESC LIMIT 1`，
  其首行只可能来自降序；忽略该标志的实现会返回升序首行并失败。
- **Node**（AST 基类）的契约是「一段语句文本分解为同一组类型化节点」。
  探针记录 `beanquery.parser.parse(...).walk()` 得到的节点类名指纹
  （`['Column','Column','OrderBy','Select','Table','Target']`），
  Go 侧 `query.ASTNodeKinds`（`internal/query/ast_shape.go`）产生同一指纹；
  映射表逐项显式列出节点名，未知构造报 `Unknown` 而不是静默丢弃。
  golden 标记 `dynamic: "ast-node"`，比较节点指纹而非查询结果。

这两项都不属于「上游内部结构」，因此不需要 `excluded` 例外记录。

## 曾登记为「未完成」的 2 项：已用可观测探针解决

这两项先前按 `internal` 分诊为 `pending`，理由是「没有独立于其它语法节点的公开行为，无法构造能区分它的查询」。上文的探针设计推翻了这个前提：两者都有可观测契约，因此不需要 `excluded` 例外记录，也不需要收窄采集范围。

| 条目 | 状态 | 可观测契约 | 探针 |
| --- | --- | --- | --- |
| `syntax.Node` | verified (2026-10-28) | 一段语句文本分解出的类型化节点集合 | golden `dynamic: "ast-node"`，比较 `beanquery` 的 `walk()` 类名指纹与 Go 侧 `query.ASTNodeKinds` |
| `syntax.Ordering` | verified (2026-10-28) | `ORDER BY ... DESC` 只可能返回降序首行 | `SELECT account AS v FROM #accounts ORDER BY account DESC LIMIT 1` |

两项的 `verified_at` 已按实际校验日期记录。

### 防伪验证

`verified` 不是改状态得来：把实现改坏，golden 必须变红。三项变异均实测复现：

| 变异 | 结果 |
| --- | --- |
| `ASTNodeKinds` 提前返回只含 `Select` | `row-mismatch want [Column Column OrderBy Select Table Target] got [Select]` |
| 从指纹中删掉 `OrderBy` 节点 | `row-mismatch want [...] got [Column Column Select Table Target]` |
| `directiveID` 不再把字段混入摘要 | `table.entries.id` 与 `table.postings.id` 双双 `row-mismatch` |

同时 `uv run --offline --project tools/reference python tools/reference/probe_inventory.py --check` 报告 `inventory goldens match the oracle: 366 items`，说明 golden 本身与固定上游 oracle 一致。

## 本轮补齐的行为

实现这些能力时发现并修复的真实差距（均有对应 golden 对照）：

- 查询语法补齐 `BETWEEN`、`IN`、`NOT IN`、`%`、`!~`、`?~`、字符串 `+`、日期 `±` 整数、`interval()` 与日历区间运算；整数除法按上游返回 Decimal。
- 新增 50 余个标量函数：账户路径函数、日期函数、字符串函数、`getprice`/`convert`/`value`、`possign`/`filter_currency`/`only`、`open_meta`/`commodity_meta` 等。
- `balances.discrepancy`：评估器把断言观测到的差额记回该 `Balance` 指令，列不再恒为 NULL。
- `pad` 合成交易的 narration 按断言金额精度渲染（`60` → `60.00`）。
- `entries.id` / `postings.id`：新增 Go 侧 `hash_entry` 复刻与路径归一化口径。
- 保留 `abs`/`cost`/`currency`/`units`/`meta`/`entry_meta`/`any_meta` 的既有专用分支，避免新实现改变已审核的取值语义。

## 尚未覆盖

- 先前 pending 的 `syntax.Node` 与 `syntax.Ordering` 已用可观测探针补齐（见上文「曾登记为未完成的 2 项」），inventory 无 pending 项。
- 本证据链只覆盖已登记在 inventory 的能力；它不证明完整兼容，也不证明「Beancount 超集」。完整门槛仍需 `docs/parity-gates-2026-10-27.md` 的全部 6 项通过。
