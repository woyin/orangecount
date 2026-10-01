# Go 会计引擎与原生 macOS 实施及验收手册

**当前实施基线（2026-10-01）**：Q02已实现有界查询执行、快照上下文、会话取消及真实C/Swift typed契约；完整工作台未完成。全量Go/七包race、51会计/147查询、原生三套/app及最新私人Swift/C通过（双方2833指令、零差异），见[本批证据](evidence/q02-bounded-execution-2026-10-01.md)。会计v3-core-3、默认legacy及查询contract保持；下一步按31.2接后台客户端、Store/QueryView、错误位置和导出。Q01剩余语义、X01/X02及GUI/安装/发行继续保留。

首次阅读见[开发入口](go-beancount-native-roadmap.md)。当前下一批从第39节开始，以已有[查询能力矩阵](query-capability-matrix.md)补剩余签名/精度与预设；第33/34节已有标量和复合值实现，历史A02/A03步骤不再当作尚未修复的red。

执行时先读[当前进度](go-beancount-native-progress.md)。本文保留原始任务及设计步骤，标为“拟新增”的部分可能已有实现；以进度和当前代码判断剩余工作。现有严格门槛是 `make parity-full`，真实私有检查是 `make verify-private-ledger`；旧基线说明不作为当前通过规则。

初版：2026-09-30；更新：2026-10-01。状态：实施步骤与验收手册。依据：[设计](go-beancount-native-design-2026-09-30.md)与[开发计划](go-beancount-native-plan-2026-09-30.md)。

本手册是后续开发入口，不是实施完成报告。以下步骤保留原始复现与目标；部分 flag、Interface 和 fixture 已实现。已有入口包括显式 semantics、compare-semantics、协议 2、parity-full 和 verify-private-ledger；没有列为实际存在的测试名称仍只是建议。先读取实际进度，完成一个行为批次后更新证据，不将全文命令一次性运行。

持仓 Store 与价格校验 red 均已修复，第28/30节保留实施过程和条件。Q01默认来源与标量基础也已实现，最新源码和检查见[Q01记录](evidence/q01-typed-query-foundation-2026-10-01.md)，原生页面与性能见[A03记录](evidence/a03-native-prices-2026-10-01.md)。继续第39节，同时保留 A01/A02/A03 GUI 验收，不重做已完成接口。

## 1. B01 保存基线并建立失败信号

### 修改与检查位置

`internal/ledger/evaluator_test.go`、`internal/compat/`、`tools/reference/`、`apps/macos/tests/`。复用现有 fixture；下面列出的新 fixture 只用合成内容。

### 操作顺序

1. 记录 `git rev-parse HEAD`、`git status --short` 和工具链。保存实际工作区 diff 与相关 untracked 文件到仓库外临时目录，作为复查基线，不回滚现有修改。注意二进制 diff 不包含 untracked 文件内容，需要另行备份。
2. 记录本轮输入版本。对私人账本只在本地记录受保护哈希；不要把文件路径、内容或来源行写入仓库证据。
3. 先用当前代码运行第 2 节公开合成例子，记录 actual/expected。这个步骤必须能检出之前的缺陷，不能只验证代码不崩溃。
4. 将这些例子变成会在现有错误路径失败的 snapshot/ledger 测试，再进入修改。不要仅测试 allocator helper，因为问题发生在 options、日期、归约和插值的整条调用链。
5. 更新已有 `docs/go-beancount-native-progress.md`，分别记录新任务和继承证据。不要复制旧计划的所有“待办”状态。

现有命令：

```sh
git rev-parse HEAD
git status --short
go version
xcrun swiftc --version
xcrun --show-sdk-version
uv run --offline --project tools/reference python -c \
  'import beancount, beanquery; print(beancount.__version__, beanquery.__version__)'
go test -count=1 ./internal/ledger ./internal/compat
make mac-check
```

离线 oracle 不可用时按仓库 `uv.lock` 安装后重试，不升级包来使测试通过。确认输入/环境未变化后可复用已有广泛检查，B01 的新失败例子仍必须实际运行。

**完成条件**：知道当前源码包含哪些工作区修改；至少可重现 LIFO、全局 booking、STRICT/SIZE、容差四组失败；所有通过证据有明确适用范围。

## 2. G01–G05 共用的合成输入与数学期望

以下文本对应现有 `testdata/fixtures/semantics/lifo-implicit.bean` 的基础；不同方法创建独立 fixture，避免读取并改写另一 fixture。

```beancount
option "operating_currency" "USD"
2000-01-01 open Assets:Cash USD
2000-01-01 open Assets:Stock STK "LIFO"
2000-01-01 open Income:Gains USD

2000-01-02 * "first purchase"
  Assets:Stock 10 STK {10 USD}
  Assets:Cash -100 USD

2000-01-03 * "second purchase"
  Assets:Stock 20 STK {20 USD}
  Assets:Cash -400 USD

2000-01-04 * "sale"
  Assets:Stock -10 STK {} @ 30 USD
  Assets:Cash 300 USD
  Income:Gains
```

以下都是独立合成金额，不来自个人账本。成本日期省略是本例的关键，不能为了让 LIFO 测试通过而补写日期。

| fixture 变体 | v3/扩展预期 | 2026-09-30 旧路径实际 |
| --- | --- | --- |
| 显式 LIFO | 成本 200；经济收益 +100，Income:Gains 为 -100；剩余批次为 10@10 与 10@20，日期分别为 01-02/01-03 | Income:Gains -200 |
| 显式 LIFO，两笔写成本日期 | 同上 | -100；此例用于定位日期路径 |
| 显式 FIFO | 成本 100；收益 +200；剩余 20@20，日期 01-03 | 不据旧结果推断通过，实施时复测 |
| 删除账户方法，增加全局 LIFO | 与显式 LIFO 相同 | 0 |
| 默认方法，无全局声明 | v3 默认 STRICT，此卖出歧义，账本非法 | 合法，收益 0 |
| 显式 STRICT | 同上 | 合法，收益 0 |
| 显式 STRICT_WITH_SIZE | 选择数量为 10 的第一批；收益 +200，收益分录 -200 | 合法，收益 0 |
| 显式 AVERAGE | 均价 50/3；收益 +400/3，收益分录 -400/3；剩余成本 1000/3 | 主数学例子已正确，仍须补日期与其他场景 |
| 删除账户方法，增加全局 AVERAGE | 与显式 AVERAGE 相同，按 OC 扩展验收 | 0 |

AVERAGE 不能拿 Python 的成功结果做 oracle，Python 3.2.3 会拒绝该归约；采用 ADR-0042 和手算期望。

容差基础文本拟放入 `tolerance-multiplier.bean`：

```beancount
option "tolerance_multiplier" "1.0"
2000-01-01 open Assets:Cash USD
2000-01-01 open Expenses:Food USD
2000-01-02 * "rounding"
  Assets:Cash -1.00 USD
  Expenses:Food 1.01 USD
```

Beancount 3.2.3 接受此例，2026-09-30 旧实现报 `E-EVAL-UNBALANCED`；当前显式 v3 的正反例已修正。另建 multiplier 0.5 的反例，必须不能被新实现一并放宽为通过。

## 3. B02 显式语义配置与迁移

### 修改位置

`internal/ledger` 的 `EvalOptions`、`internal/snapshot` 的 `BuildOptions` 与身份、`cmd/orangecount`、`internal/desktop/session.go`、`internal/bridge/session.go`、Swift 模型/客户端/Store，以及 ADR-0049。

### 实施步骤

1. 核对 ADR-0049 与实施授权：先实现显式模式，保持当前默认。默认切换不作为此任务的隐式副作用。
2. 增加类型化 semantics 值和校验；空值按旧配置兼容，不接受未知字符串。实现版本也有确定名称，不能拿当前 Git dirty 状态当作会计版本。
3. `BuildWithOptions` 将配置传给 evaluator；快照身份纳入配置及会计版本。Store 保留最后有效快照时必须属于相同会话配置。
4. CLI 已新增 `check/query/serve/export` 的 `--semantics`；所有入口同样解析。方言展开后的标准交易使用同一配置，不能有专用隐式默认。
5. desktop 增加 `OpenWithOptions`，原 Open 保留 legacy 包装；同一会话配置不可变。bridge 增加拟议 `OC_OpenLedgerWithOptions`，Go 与 Swift 协议同时升级并真实链接验证。
6. native 提供当前配置标识、比较与显式应用。保存的偏好是本地用户选择；不在用户账本写未知 option。换配置执行完整会话切换，清空分页和旧请求代次。
7. 比较读取同一份源图：分别求值旧/新配置，输出有效性、完整批次、余额、关键收益差异。输出默认脱敏；原生本地差异视图仅在用户操作时显示必要详情。
8. 新版非法时保留旧会话供查看和修复，不将旧结果包装成 v3 有效快照。切换失败不泄漏新句柄。

### 必需验证

- 同源不同配置 SnapshotID 不同；同源同配置稳定；语言/搜索不改会计身份。
- legacy 结果与当前适用基线一致；v3 只在明确选择后执行。
- reload 保持配置；A legacy → B v3 的迟到请求不能更新 B。
- 首次非法、比较非法、关闭时尚在打开、双窗口不同配置均真实释放句柄。
- CLI、Web 与 native 的同配置结果一致；旧脚本未传参数时行为有清晰声明。

**完成条件**：配置可贯通全部会计入口、身份与显示；没有静默默认变化；ADR 状态及任何扩展效果准确记录。

## 4. G01 隐含成本日期和完整批次键

1. 在 acquisition 的归一化 Seam 补交易日期，仅在 v3 配置执行。明确归约缺失日期仍是通配，不能把买入与卖出用同一“空值补日期”逻辑处理。
2. 将归一化成本应用到 working inventory、最终 Position 与已归约分录；不要只修最终展示。保留 source AST/Span，不为了补日期重写文件文本。
3. 核对完整键相同批次的归并、部分归约和清仓后删除行为；不同日期/label 保持区分。
4. 复跑第 2 节隐含/显式日期 LIFO，以及同价不同日期、同日不同 label、指定日期卖出、历史持仓场景。
5. 给 AVERAGE 的 earliest date 补省略日期的例子，防止新均价批次继续没有日期。

拟新增测试名：`TestV3LIFOUsesImplicitCostDates`、`TestV3ReductionDateRemainsWildcard`、`TestV3LotIdentityIncludesDateAndLabel`。这些名称是开发建议，不是当前已有测试。

检查入口：`go test -count=1 ./internal/ledger ./internal/snapshot ./internal/compat`；命令覆盖实际 snapshot 路径。私有完整批次对照在 V02 做，不用仅检查 Position 数量验收本任务。

## 5. G02 全局 booking 和优先级

1. 将声明字符串与生效类型分开。检查固定 oracle 可接受的方法、大小写、非法方法和重复/include 行为。
2. 实现账户显式声明 → 全局声明 → profile 默认。v3 默认 STRICT；legacy 保留现状用于比较。
3. 所有方法通过同一生效值进入归约。普通 options 表和账户详情展示声明值/生效值的区别，不将空账户字符串当成“没有规则”。
4. 增加全局 LIFO、全局 AVERAGE、账户 FIFO 覆盖全局 LIFO、账户 AVERAGE 覆盖全局 FIFO、无声明默认 STRICT、非法方法反例。
5. HIFO 作为上游已有方法验证，修正文档中的误分类；不要把它放进 AVERAGE 扩展豁免。

拟新增测试：`TestV3BookingMethodPrecedence`，表驱动覆盖全局与账户组合。它必须检查归约分录、收益和剩余库存，不只检查 `AccountState.Booking` 字段。

检查入口：`go test -count=1 ./internal/ledger ./internal/compat ./cmd/orangecount`。

## 6. G03 STRICT、STRICT_WITH_SIZE 和失败归约

1. 把 bookPosting 返回状态改成可明确区分的成功/失败/无需处理，保持 Interface 小而完整。
2. STRICT 实现单匹配、多匹配全清、歧义和超卖。不能在 allocation 失败后原样进入售价插值路径。
3. STRICT_WITH_SIZE 先用 STRICT；歧义时选择与卖出量等量的批次。若多个等量匹配，按 oracle 最早成本日期选择；不要错误要求必须只有一个等量批次。
4. 对失败交易验证 working 与权威库存不发生部分变更；无效 snapshot 不发布。诊断保留源卖出及相关匹配位置，不能同一次失败重复报多条泛化库存错误。
5. NONE 的负库存与成本要求单独对照，不能被普通 STRICT 超卖修复连带改变。无法明确支持的混合长短仓模式给出准确诊断和范围说明。
6. 拆分 allocation 与 interpolation 的测试：卖出即使带 `@ 30 USD`，也不能把归约失败变成有效零收益。

拟新增测试：`TestV3StrictRejectsAmbiguousReduction`、`TestV3StrictAllowsFullLiquidation`、`TestV3StrictWithSizeUsesOldestEqualSizedLot`、`TestFailedBookingCannotInferZeroGainOrPublish`。

检查入口同 G01，并增加相关诊断/repairguidance 测试。断言非法例子退出码非零，不仅匹配诊断文案。

## 7. G04 容差选项

1. 针对本轮 fixed oracle 建立 options 行为矩阵：新/旧 multiplier 名、默认值、负数/无效值、每币种默认、通配符、重复声明、显式 balance 容差、cost 推断。
2. 对原始数量的精度及成本/价格贡献按 oracle 公式计算，不沿用当前“所有字段最大小数位数”的推断假设。
3. 交易平衡容差与 balance 断言的容差分别实现/测试，保留来源与生效值。不要把 `inferred_tolerance_default` 自动用于它不控制的语义。
4. 第 2 节 multiplier 1.0 的正例和 0.5 的反例都必须正确；再加多币种、插值前后、单位/总价格、有理数和零数量场景。
5. OC 特有 `tolerance` 选项如继续支持，登记它相对 v3 的优先级与扩展效果，不借此掩盖标准选项的缺失。

检查入口：`go test -count=1 ./internal/ledger ./internal/compat ./internal/dialect ./cmd/orangecount`。

**完成条件**：选项不是仅可解析；交易有效性和断言分别有正反例，与固定 oracle 一致或精确标识扩展。

## 8. G05 AVERAGE 完整路径

1. 复用既有懒合并 Implementation，优先补配置/日期链路，不重新写另一套均价引擎。
2. 第 2 节固定例子断言精确 `50/3`、`-400/3`、`1000/3`。扩展测试在独立语料中运行，普通 v3 差分不能因上游拒绝而失败或全部豁免。
3. 加入买入 → 部分卖出 → 新买入 → 第二次卖出 → 全清 → 重新买入。逐步断言数量、成本、收益及 cost conservation。
4. 覆盖全局/账户优先级、隐含日期 earliest、显式成本拒绝、跨成本币种拒绝、零成本批次、单批次、超卖和 ordinary costless amounts。
5. 明确日期/label 限定匹配范围，并显示参与合并的批次；未匹配批次不能被静默改变。
6. 检查内部 M 腿在 holdings、平均成本序列和归约事件中的处理；重组腿不产生额外经济收益或现金流。
7. 真实 CLI snapshot 和 C/Swift 输出也验证至少一个部分卖出例子；不只跑 evaluator helper。
8. 验证未启用 AVERAGE 的账户在相同 profile 下不受该扩展影响。导出给 Python 的边界由 X02 另行验收，不提前承诺可逆。

M1 阶段检查上述核心归约、数学与已有端到端输出；第 6 项的历史持仓/均价末点及对应新原生页在 A02/M3 批次补齐。它们仍是必须验收项，不让尚未交付的页面反向阻止已经正确的核心成为后续开发基线。

检查入口：ledger/report/compat/dialect 的相关测试，`make mac-test`。任何数值比较均从独立数学预期生成，不能拿 holdings 的自身输出作为 expected。

## 9. V01 修复并扩展公开差分

### 文件责任

`tools/reference/generate_golden.py`、`internal/compat/parity.go`、`boundary.go`、`tools/parity/main.go`、`docs/compatibility-ledger.json`、公开 fixtures/golden、Makefile 和适用 CI。

### 步骤

1. 修复 Booking Enum 的显式序列化；将 cost date/label、元数据、posting flags 和价格结构纳入标准化。只保存自研比较格式，不复制 GPL booking Implementation。
2. 使用序列/多重集保存重复条目和分录。源顺序、已归约顺序和允许重排分别判定，不能集合去重后失去重数。
3. 用公开正反例核实 `@@` 总价归一化中的负数量；决定差异来自 harness 还是 engine。修正一侧并保留最小回归，不写通用 abs 豁免。
4. 金额数值按确切有理数/decimal 契约比较；非终止精度差异单独登记适用类型及 oracle 证据，不将“保留 12 位”变成通过规则。
5. diagnostic 建立错误类别映射，保留次数和来源关系；当前 posting/header 行号差异的允许范围明确。
6. 按设计第 8 节扩展 state/query/report。reports 加入独立截至日、期间和缺价样例；queries 不再只有 Cash narration 一条。
7. boundary 由配置清单精确匹配；ID 不存在、scope 不匹配或诊断未知都失败。修复 `query-subset` 硬编码与 JSON 不一致，但不注册一个无限宽豁免。
8. `tools/parity` 在未登记差异/缺失 golden 时非零退出；`differential-corpus.sh` 传播失败。自动 Jev 分流不参与 pass/fail，不发送私有数据。
9. 已有 `make parity-full`：默认只读取 committed golden，适合离线 Go CI；另设 pinned oracle 更新检查，生成到临时目录后对照，不能测试时自动覆写已提交 golden。
10. AVERAGE extension golden/数学测试有独立入口与 capability ID。普通 booking 不能匹配到该 ID。

先保存当前 18 fixture 的差分，再更新模型，最后审查每个新 golden 的变化理由。不要用更新所有 expected 的方式“修复”产品错误。

可立即运行、不会写入 tracked diff 文件的现有入口：

```sh
go run ./tools/parity \
  -fixtures testdata/fixtures/v3-parity \
  -golden testdata/golden/v3-parity
go test -count=1 ./internal/compat
```

当前主门槛为 `make parity-full`，未登记差异/缺失 golden 会返回非零；上述旧格式入口仅保留为 legacy 迁移基线。`make parity-report` 当前会重写 golden 并可能执行外部分流，不用它做只读基线验证。

**完成条件**：差分检出器有负向故障验证；遗漏日期、颠倒价格符号、删除重复分录、错误有效性都能使检查失败。

## 10. V02 私有账本与真实原生投影

1. 保留外部账本，只通过 `ORANGECOUNT_PRIVATE_LEDGER` 等本地参数选择，示例用 `/absolute/path/to/main.bean`。不要将用户真实路径写入 fixtures、CI 或文档命令。
2. 新私有检查入口同时使用 pinned Python、当前 Go snapshot 和真实 Swift/C 客户端，比较 M1 已声明范围。明确哪些部分只验证解码，哪些对照外部预期。
3. 检查账户×币种余额、完整成本批次日期/label、关键月/截至日的余额与收支、必要的已归约成本/收益。结果只输出数量、差异类别、合法性和退出码。
4. 替换现有私有 equivalence 脚本的“100% 等价”措辞、CLI 假称 native 验证及吞错行为。所有原始输出只在临时进程/文件，结束清理。
5. 比较前后输入哈希一致；个人账本合法改变后计数可变，不将 2,498/89/311 当永久 expected。最终 pass 由当前同输入对照决定。
6. AVERAGE 继续用公开合成 fixture，不能因用户账本没有 AVERAGE 就跳过验收。

现有脱敏有效性入口：

```sh
ORANGECOUNT_PRIVATE_LEDGER=/absolute/path/to/main.bean \
  sh tools/reference/differential.sh
```

此入口当前从源图一次捕获完整状态，对比 entries、inventory 和月度收支；使用当前源码的 Go capture。它报告 native_checked=false。实际 Swift/C 根汇总验证另用 `ORANGECOUNT_PRIVATE_LEDGER=/absolute/path/to/main.bean make verify-private-ledger`，并报告 native_checked=true。两者均不证明 GUI 或所有历史页面。

**完成条件**：完整批次不再忽略日期；关键报表和原生载荷有对应范围；临时文件清理，仓库无私人内容。只读验证不运行会编辑输入的 benchmark/soak 工具。

## 11. N01 多币种报表与可用性

1. 修改 `internal/report/native_statement.go`，分开完整原币树、目标币种已折算部分与 total completeness；移除 session 中不可用就清空全部节点的处理。
2. 新 DTO 按设计第 9 节表达 missing currency/price、覆盖计数、报价日期、空/零/部分/完整；Go、桥接、Models 和契约测试同批修改。
3. 新账本未有明确偏好时采用拟议原币默认；已有用户选择不被 reload 或页面切换覆盖。默认行为调整在原生偏差记录中说明。
4. 目标币种缺价时显示原币数据与缺报价说明；部分值明确“已折算部分”，完整净资产/收支总计为不可用。直接余额、子树汇总和导出保持相同口径。
5. 父节点未覆盖的非零净额不能变成完整总计；缺价子账户抵消形成合法零时，可显示父净额零，但保留子账户/表级缺价状态。合法零值仍显示零，不与 unavailable 共用显示。

公开 fixture：纯 USD、USD/EUR 无汇率、仅一条缺 quote、含证券成本、多本位币、无本位币、合法空账本。最少一例有可换算金额 100 USD 与不可换算金额 20 EUR；完整总计必须不可用，已折算部分仍可见。

检查：report/desktop/bridge 测试 → `make mac-test` → fixture GUI。私有账本只用作补充，不做截图 golden。

## 12. N02 原币趋势与历史估值

1. 将原币图表表示改为按 currency 分组的精确序列，不把多币种 monthly map 投入一条累计曲线。
2. 若提供 market 模式，按每个历史点的库存和当时已有报价估值；quote 日期不得晚于历史点。现有成本/units 模式仍保留独立标签。
3. 明确 month-end 和最后一个不完整月的日期；asset/liability carry forward 与期间 income/expense zero activity 区分。
4. 缺价点使用 nullable amount/结构化 status，不生成 0，也不跨缺口画连续线。Swift 可选币种或显示独立面板。
5. 审查退款/冲销、Equity、多期间、quiet month、future quote、无报价、全清重买和 M 腿的影响。

固定例子：月一净资产 700 USD；月二无活动仍为 700；月三支出 100，净资产 600。另加 EUR 系列独立展示，不与 USD 求和；历史点不能使用下月才出现的汇率。

检查：charts/report 固定值 → C/Swift exact labels → GUI 图例、tooltip、缺点和切换。

## 13. N03 协议、时序和关闭

该任务贯穿所有新增页面，不重新实现现有 Store。

1. 每次新增 view/DTO，先在真实 Swift/C ContractTests 验证解码及 exact 值，再修改 UI。
2. 请求身份包括 session/semantic profile/snapshot/normalized parameters；复用现有 generation，旧响应不可更新新导航。
3. 增加可控测试：旧 holdings 晚于新 account 返回；切币种/日期后旧趋势迟到；开 v3 后 legacy 请求迟到；查询执行时换账本/关闭。
4. 已开始的 C 调用允许正常收尾，结果丢弃并释放；排队取消不再进入 C。轮询/缓存/返回栈在关闭后释放。
5. 同账本错误保留旧有效数据并显示其快照/日期与诊断；另一非法账本不能借用前账本的数据。

检查：`make mac-test` 与 session/bridge targeted race。每个新响应字段有版本/必填/空集合测试，不用一堆 `try?` 吞掉协议错误。

## 14. N04 GUI 与性能

复用 [原生验收清单](native-macos-acceptance-checklist.md)，补新模式和历史缺口：

1. 欢迎页 → 打开 → 原币表 → 显式目标币种/缺价 → 流水来源 → 外部合法编辑 → 错误 A/B → 修复 → 退出。
2. Finder 打开、真实拖入、目录入口、失效最近文件、Cmd+O/R/F、账户方向键、Tab 和返回导航。
3. 记录实际 900×600 最小窗口；中英文、Light/Dark、长账户、多个币种。大字体必须确实改变渲染字号后再验收，不以无效果的 sizeCategory 设置算通过。
4. AX 标签应读出准确金额、币种、可用性与来源；图表不能重复累计值，也不以颜色作为唯一状态。
5. 两个独立账本/配置窗口；开关 20 次；关闭后无轮询。已有 10 分钟有界内存观察可作背景，不当作长期无泄漏证明。
6. 使用固定 52,500 笔合成账本测 cold/warm、四页和新增页面、过滤/分页、reload 各阶段。记录客户端与 GUI 计时的区别。

现有入口，按阶段顺序运行：

```sh
make mac-check
sh apps/macos/prepare-qa.sh
make mac-benchmark
```

已有 QA 隔离配置支持真正的字号倍率。大字场景可用 `ORANGECOUNT_QA_FONT_SCALE=1.5 ORANGECOUNT_QA_APPEARANCE=dark sh apps/macos/prepare-qa.sh` 准备，再实际启动和检查；脚本成功只证明 QA 包配置成功。正常产品包不能继承 QA 字号/外观标记。记录窗口外框与内容尺寸，避免把 SwiftUI 的 minHeight 当作实际窗口尺寸。

性能测量时停止并行构建。benchmark/soak 用合成账本和其副本，不传个人账本。GUI 捕获失败时记录具体缺口，继续客户端/状态/投影检查，但 GUI 项仍待验收。

## 15. R01–R03 构建与发行

### R01 开发产物

复用现有脚本，不再重复引入另一套 Package.swift/Xcode 工程。运行 `make mac-package`，检查 arm64、Mach-O/plist 最低版本、许可、版本、源码 revision/dirty、校验和和 ad hoc signature。解压到独立目录，使用公开 fixture 完整运行阅读/编辑错误恢复/退出循环。记录输入和实际环境。

### R02 远端 CI

在既有 `.github/workflows/macos.yml` 加入 V01 的可靠差分门槛，保持 CLI release 独立。只有在授权远端提交/执行后记录实际 run 链接、源码版本、工具链、日志和产物；不能由本地构建推断云端通过。CI 只用公开合成 fixture，oracle 固定版本。

### R03 系统与公开分发

在所声明系统/架构实机运行 R01 的循环；macOS 13 是编译目标而不是已有实机证据。若只具备本机测试，支持矩阵明确限定已验证环境。

公开分发需要可用 Developer ID、entitlements 检查、公证、staple/验证、实际下载/解压启动和目标机器验收。没有证书或测试机器时产出开发包及明确的待完成清单，不伪造签名、公证或公开发布状态。

## 16. A01 账户详情

1. 在 report/desktop 从当前 Evaluation 和 request 生成账户详情 DTO，包含 account、scope（direct/subtree）、生命周期、生效 booking、日期范围、原币/估值余额、趋势与分页流水。
2. Swift 新增账户详情视图，沿用 Store 客户端和 generation。树/流水账户链接进入，返回恢复上页筛选与展开。
3. 验证父账户直接余额为 10、子账户为 20 时 direct=10/subtree=30；search 只隐藏行，不改变 scope 汇总。
4. 关闭账户仍可查看过去；不存在账户明确报错，不返回另一账户结果。期间/截至日和 active profile 可见。

检查：report/desktop 固定值、真实 Swift、受控导航、GUI 进入/返回/重载。新增方法名/文件可采用 `account_detail.go`、`AccountDetailView.swift`，以最终契约为准。

## 17. A02 持仓、均价和收益

1. 复用当前 `ledger.ReplayBookedInventory` 与 `report.HoldingsView`，补截至日库存边缘；current inventory 不能直接过滤来充当历史。首批连续 AVERAGE 已有测试，不重写第二套计算。
2. 返回完整 lot key 和精确数量/成本/均价，按 account/commodity/cost_currency 分组。不跨商品求“总数量”，不跨成本币种求均价。
3. 建立归约事件的来源关系，区分实际卖出与内部 M 合并。已有 `@`/`@@` 销售价格与已归约成本才能形成可确定毛收益。
4. 净收益只有费用明确归属时展示；无售价、跨币种不可折算、交易中多个商品/费用无法归属时返回 unknown reason，不按摘要猜测。
5. Swift 提供分组、批次展开、截止日、口径、价格日期和来源。总成本/市值/收益沿用 completeness，不用 0 替代未知。
6. 固定例子覆盖 FIFO/LIFO/AVERAGE 逐时点、全部清仓、再买入、零成本、不同 label/日期、转账、无售价、费用、缺汇率和同日交易。
7. 均价图末点与 holdings 的同日同配置一致；该一致性只证明内部协调，另用手算/oracle 验证正确性。

检查：ledger/report/state 固定结果、real C/Swift、GUI 批次展开和快速 cutoff 切换。页面默认不宣称账户总体 ROI/IRR 或自动税务收益。

## 18. A03 价格历史

1. 从本地 price 指令生成 base/quote 列表、精确时间序列、来源、最新有效报价和 as-of 选择。
2. 商品来自 commodity 声明及持仓，不只来自 price map；无报价资产仍出现并显示缺价。
3. 原生列表/图表支持 quote 切换；不跨 quote 连一条线，不用未来数据。
4. 缺价、陈旧、同日多价、倒数/交叉报价规则按共享价格 Module 的已声明行为验证；不加入网络抓价。

检查：现有价格 fixture 扩展、Swift/C 解码、GUI。显示“价格来源与日期”，不把 price 指令当作发生交易。

## 19. Q01 BQL 能力分批完成

1. 更新已有[能力矩阵](query-capability-matrix.md)，包含语法、列/类型、函数、聚合/空值、错误与 oracle 版本。旧入口与typed入口分开记录；已有标量和默认来源检查保留，当前差距为复合类型、有效元数据及剩余声明签名。
2. 第一批补标准默认 FROM、`abs`、`length`、`meta`、`filename`、`lineno` 与来源预设。每格先固定 oracle positive/negative，再实现。`meta` 先验证直接字段、pushmeta/popmeta、覆盖/null、文件/include 作用范围和非法 pop；有效元数据需要按词法顺序传播，再进入日期排序，不能对排序后的 stack 指令猜测作用范围。
3. 第二批建立 typed amount/position/inventory 结果及序列化，随后补 `units/cost/weight/number/currency` 的声明范围。不要把同名函数/列都实现为 Decimal 的别名。
4. 检查 NULL、除零、空集合聚合、多个币种、GROUP BY、ORDER BY、重复行和 alias；未知功能必须给出准确错误。
5. 输出格式区分 machine exact value、type 与 display。保留现有 CSV 行为的适用范围，typed inventory 的导出不可隐式截成一种币种。
6. 全 SQL、JOIN/CTE/window 和任意扩展函数不在 M4。超出范围明确拒绝；pivot 继续使用既有报告路径。

检查：query targeted tests + pinned beanquery golden + CLI tests。每批完成后修改帮助/能力矩阵，不等到整页完成才暴露限制。

## 20. Q02 原生查询

1. 扩展 ReadView 的 query 请求和类型化结果，在 desktop 复用 Go query，不建立 Web HTTP 请求通道。
2. 请求绑定当前会话/快照/配置；已保存 query 指令作为只读选项。预设必须在 Q01 的实际能力范围内。
3. Swift 提供编辑/执行/错误位置/列类型/分页或结果上限/排序/CSV 保存。限制查询长度、结果行数和字节量，清楚区分计算和显示限制。
4. 后台执行，换账本/页面/关闭时丢弃迟到结果并释放资源。同步 C 调用的取消能力如未实现，应准确描述为取消显示和排队，不伪报中断执行。
5. profile 与 SnapshotID 变化清缓存，失败不能展示上一次查询结果为此次成功结果。

固定任务：按月支出、账户余额、来源文件/行号、NULL/错误、一万行结果上限、运行中换账本和关闭。检查真实 C/Swift 与受控乱序后再 GUI 验收。

## 21. X01 全指令浏览

1. 在既有 transaction index 旁增加指令索引，明确哪些是已求值日期指令、哪些是 source/config 指令。
2. 类型化 DTO 保留每类的自身字段，元数据/标签/链接/来源，不以一条 generic 金额行渲染所有内容。
3. UI 默认交易模式保持可用；提供全部指令模式和类型筛选。计数、排序与日期锚定清楚，无日期指令不使用今天。
4. 所有页绑定 SnapshotID。重复指令、源文件插行、热重载和类型切换时展开恢复不能串记录。

公开 fixture 要包含全部声明范围的类型及重复记录。check 的 diagnostics 页面继续服务其自身错误，不将其混成普通 Journal entry。

## 22. X02 附件与分析导出

1. document 指令可按账户/日期关联，显示存在/缺失状态，提供外部打开和 Finder 定位；源位置动作继续尊重默认编辑器。
2. 文档根规范化及 symlink 解析后检查允许范围。绝对路径、目录穿越、symlink 逃逸、失效文件和同名文件均有场景。
3. 使用保存面板选择 CSV 目标；从当前已捕获响应或固定快照导出，不在导出途中读入新快照。
4. 标明期间、截止日、币种、valuation、profile 和 completeness；partial 列不能命名为完整 total。精确数值不可被格式化显示替代。
5. 临时写入后原子替换，失败不破坏已有导出文件，不写入账本源图。
6. 单独验证纯 v3 源快照导出的 AVERAGE 边界；如果不能保持会计结果，明确拒绝或提供另行设计的已归约导出方案，不冒称无损。

检查：document/export targeted tests、Swift/C、GUI 保存/取消/错误。CSV 与源快照导出必须在 UI 和文档中分开命名。

## 23. 阶段检查与 D01 收尾

每个局部行为先跑相关包及真实契约。新增失败例子通过后，在里程碑集成时执行现有完整门槛：

```sh
make fmt vet test race license build mac-check
```

集成检查追加已有 `make parity-full`；oracle 再生成在独立更新步骤审阅，不在 gate 中覆盖 golden。修改 Web DTO/前端时增加对应 `web-check`；仅 Go 内部调整且既有 Web Go 测试覆盖时，不无条件重跑全部视觉采样。

阶段结束更新 README、development、compatibility 清单、原生 progress/acceptance/support。D01 对每个任务记录：源码与 profile、输入/oracle、实际命令、结果、产物、未验证项。旧证据只在源码/输入/环境仍适用时继承。

文档验收检查链接、任务 ID、依赖、协议字段、范围及状态一致性；不为文档创建与正文镜像的永久测试。

完成报告分别回答：代码实现了什么、自动化验证了什么、GUI 验收了什么、哪台机器能运行、是否公开签名发布。任何一个答案缺失，不用“100% 等价”代替。

## 24. 后续分析任务的固定验收场景

本节是可直接转为公开 fixture 的预期，不表示对应页面已经实现。已有 fixture 可复用；新增账本仅含合成数据。金额预期从下列数学关系产生，不从被测输出反推。

### 24.1 账户与截至日范围（A01）

建立父账户 Assets:Test 与子账户 Assets:Test:Child。第一日父账户增加 10 USD、子账户增加 20 USD，第三日子账户减少 5 USD，第五日子账户关闭；对手账户用于保持交易平衡。

| 请求 | 独立预期 |
| --- | --- |
| 截至第二日，direct | 10 USD |
| 截至第二日，subtree | 30 USD |
| 第三日至第三日的 subtree 期间变化 | -5 USD；截至日余额仍为 25 USD |
| 仅搜索父账户文字 | 展示可过滤，scope 原始汇总不变 |
| 第六日查看已关闭子账户 | 历史余额与生命周期可见，不能伪造清零 |
| 不存在账户 Assets:Testing | 明确目标错误，不能前缀匹配 Assets:Test |

核对期间流水与余额分别采用期间和截至日规则。进入详情→改筛选→返回→reload，恢复上页状态；输入导致目标消失时显示原因并可返回。

### 24.2 连续 AVERAGE 与经济事件（G05 / A02）

所有归约为显式 AVERAGE、无额外费用，商品 STK、成本/售价 USD，以下日期按先后递增：

| 操作 | 剩余数量 | 剩余成本 | 持仓均价 | 此次可确定毛收益 |
| --- | --- | --- | --- | --- |
| 买 10 × 10，再买 20 × 20 | 30 | 500 | 50/3 | 不产生出售收益 |
| 卖 10 × 30 | 20 | 1000/3 | 50/3 | 400/3 |
| 再买 10 × 40 | 30 | 2200/3 | 220/9 | 不产生出售收益 |
| 卖 12 × 50 | 18 | 440 | 220/9 | 920/3 |
| 卖完 18 × 25 | 0 | 0 | 不可用，无持仓 | 10 |
| 重新买 5 × 8 | 5 | 40 | 8 | 不产生出售收益 |

全清前累计买入成本为 900 USD，累计出售归约成本为 900 USD；三次毛收益合计 450 USD。内部 M 腿不增加出售次数、交易现金流或该收益总和。每个历史 cutoff 要看到当时持仓；全清日后的当前仓不能代替全清前历史。

另外各建一个无售价归约、转账、明确费用与多商品无法分摊费用的例子：毛收益只有已知售价/成本及同币种可确定时可用；无费用归属规则时净收益不可确定。零成本 lot 与无成本普通数量分别覆盖。均价图每个点先对数学预期，再对同 cutoff 持仓，不能只测两条错误路径相等。

### 24.3 历史报价与陈旧状态（N02 / A03）

复用 `testdata/fixtures/native/history.bean`：一月现金 900 USD 与 10 STK，尚无商品报价；二月 15 日 STK=20 USD；三月四日全部卖出，总资产 1100 USD；三月后报价 99 不影响已清仓资产。

一月末市值完整总计为 null、已估部分 900；二月 29 日为 1100、使用二月 15 日报价；三月末为 1100、持仓无需 STK 报价。成本序列一/二月为 1000、三月为 1100。无交易月份仍 carry forward；缺价点不置零，不跨段连线。

再加入同日两条同 pair 报价、倒数 pair、不同 quote 和未来价。列表展示来源与全部历史记录，选用规则按共享 PriceAt；没有 price 的 commodity/持仓仍出现在商品列表。陈旧显示 quote age，不改变有效性。

### 24.4 查询与全指令（Q01 / Q02 / X01）

第一批 oracle 包含省略 FROM 的默认 postings、abs(-3)、length 空/非空字符串、filename/lineno、直接 meta 与有效 stack 元数据。独立覆盖未知函数、未知列、非法语法、NULL、零除和空集合聚合；将查询文本、类型、值、顺序、重数和错误分别写入 golden。

第二批包含同一交易的 USD 金额与 STK costed position，验证 units/cost/weight/number/currency 返回类型及完整库存，不把函数名称相近当成结果类型相同。原生表格可展示 exact 和类型，compound 单元格可展开；CSV 不丢币种或 lot key。

全部指令 fixture 每类至少一条，并含完全重复内容、无日期配置、include、push/pop、plugin 声明。transaction 模式与 all/config 模式计数分别核对；源记录和求值生成记录显式区分，不重复冒充用户输入。分页后 reload/插行/切类型不会沿用旧 ID 展开另一条记录。

### 24.5 附件与固定投影导出（X02）

使用临时公开目录：合法相对文件、允许根内绝对文件、失效文件、`../` 越界、symlink 逃逸、同名不同目录各一例。路径解析依据 document 所在源文件及明确配置的文档根，不猜工作目录。打开前检查文件当前状态，拒绝项保留声明来源。

选择导出目标后再触发 reload，保存内容必须仍属于捕获的快照；比较 CSV exact、币种、类型、完整性与该响应。模拟不可写目标和写入失败，原目标内容不变；取消不写文件。尝试导出覆盖源文件/附件时明确拒绝。

AVERAGE 源导出单独执行 pinned Python 加载与完整结果对照，不能因为分析 CSV 成功而宣称会计 round-trip 成功。无法保持结果时拒绝该路径并给出具体扩展原因。

## 25. 开工及收尾操作顺序

1. 读取进度、当前任务设计节、相邻 ADR 和涉及文件；保存或核对 B01 工作区备份。记录当前输入/profile/工具链，识别可继承与已失效证据。
2. 选择一个用户可见纵向切片；在既定 snapshot/report/desktop/CLI 或真实 C/Swift 接口固定独立预期，先复现失败。不先写全部未来页面或内部 mock 框架。
3. 最小实现 Go 投影或核心行为；targeted 检查通过后同步协议/Models，再写 UI。新字段的 required/null/empty/未知枚举及精确字符串规则一起检查。
4. Store 测导航/日期/币种/账本切换乱序与关闭；真实 C/Swift 测实际序列化。二者不能互相替代。
5. 公开 fixture GUI 检查精确文本、状态和操作，再做该任务必要性能测量。私有账本只做脱敏补充，不作为公开快照或截图。
6. 里程碑运行集成 gate；仅在失败、新改动或未解决问题下追加检查。更新任务进度和能力说明，写明下一个未完成动作。

文档本身收尾仅需链接/任务/依赖/契约一致性与 `git diff --check`；不需要重跑未受文档修改影响的会计、UI 或 benchmark 全量测试。实际开发时按第 23 节的对应门槛执行。

## 26. A02 从当前代码继续的实施清单

### 26.1 开工位置与现有检查

当前文件可直接作为起点：

| 位置 | 已有行为 | 本批需要补齐 |
| --- | --- | --- |
| `internal/ledger/inventory_replay.go` | 完整键、实际消费批次来源、signed库存；32个合法 oracle 最终库存通过 | 新出现的重放/来源缺口；不重写现有重放 |
| `internal/report/native_holdings.go` | account/scope/分组、成本/均价、成本与市值估值/差额 | 原生来源定位与完整性汇总展示 |
| `internal/report/native_average_history.go` | 交易后检查点、真实清仓分段、同日日末合并、cutoff 末点 | 新原生曲线及 quiet/过滤/分页实际展示 |
| `internal/report/native_reductions.go` | @@一次计价、毛/净收益/未知原因、消费及 AVERAGE 整池贡献来源 | 界面区分实际消费与整池，补大贡献池分页等缺口 |
| `internal/desktop/holdings_test.go` | 连续 AVERAGE、FIFO/LIFO、nullable键、费用/价格/分组/分页/预算/旧快照 | 新出现的具体未覆盖行为，不重写既有检查 |
| `internal/desktop/holdings.go`、session、bridge | 四section分页、snapshot selector、16 MiB完整响应预算 | 新页交互链路及性能 |
| `apps/macos/src/Models.swift`、Store/tests、HoldingsView | holdings/merge typed、受控时序、真实客户端、原生页已实现并检查 | GUI/键盘/AX及后续新增场景，不重写 Store/UI |

以下为现有公开检查；原生命令在 Store 接入前曾失败，当前实现后通过：

```sh
go test -count=1 ./internal/desktop ./internal/report -run 'TestNativeHoldings|TestHoldingsReplay'
make mac-test
```

原文档交接时 Go 和 ContractTests 通过、StateTests 编译失败、NativeClientTests 未执行；后续三套均实际执行通过。先检查实际测试文件，缺少的场景逐个固定 red 信号；新增测试名由行为决定，不因名称存在就认为验收完成。

### 26.2 公开边缘场景及独立期望

| 场景 | 合成输入/数学预期 | 必需检查 |
| --- | --- | --- |
| FIFO 与 LIFO 历史 | 10×10、20×20，卖10×30：FIFO 剩20、成本400；LIFO 剩20、成本300 | 买入日、卖前、卖后；不能把最终 lot 过滤为卖前库存 |
| 完整 label 键 | 同成本/同日期两 lot：label 缺失、label 空字符串 | 归约匹配/展开身份分别存在；不按 display 生成键 |
| 零成本与无成本 | 5 STK 成本0 USD、7 STK 无成本，用合法显式对手分录建立 | 零成本基础/均价是0；无成本是null；成本0不可被跳过 |
| @@ 拆为两批次 | 买2×10、3×20；FIFO 卖4 STK `@@ 100 USD` | 消费成本60、gross40、剩1×20；仅一个源出售事件、售价100只计一次 |
| 重复交易 | 两条源交易摘要/数量/报价完全相同，但有不同来源位置 | 两个独立事件；不以内容哈希去重 |
| 同日清仓后重买 | 日内买10×10→卖完→买5×8 | 日末5/40/8；交易事件完整；新持仓段不继承已清仓段；无 M 假清仓点 |
| 已知费用 | 买10×10，卖10×20；cash195、expense5，显式归属该账户 | gross100、fees5、net95；归属标记无会计副作用 |
| 未归属费用 | 与上例同数值，删除费用归属 metadata | gross可用；net null，unattributed-expenses；不能根据账户名猜测 |
| 多商品交易费用 | 两个经济出售共用一笔费用，即使给出账户名 | 不自动分摊；ambiguous-trade-fees；各可确定 gross 仍可见 |
| 事件日缺汇率 | 成本USD、售价EUR，仅事件后一天提供EUR/USD | 事件收益未知；不能用后一天或 cutoff 的汇率 |
| 同币种市值 | 剩10 STK、成本100 USD，cutoff price=15 USD | quantity10、basis100、market150、unrealized50，quote date≤cutoff |
| 目标成本汇率缺失 | 已知USD成本和STK/EUR报价，但无USD/EUR | EUR市值可用，EUR成本/未实现差额null；原USD成本保留 |
| 转移/无售价 | 同商品在资产账户间移入移出；另一例无明确price | 不从 Income 或摘要推断出售；对应未知 reason |
| NONE 开空仓 | 新增负数量且不存在可消费的同键多头 lot | consumed_quantity=0；非普通已实现出售；signed库存保留 |
| 分页/未知参数 | 大量groups/events；未知group_by、错误scope、过期snapshot | 真实返回页/count/truncated正确；非法组合明确报错，不静默忽略 |

label、无成本、转移和 NONE 的账本是否合法先由固定 oracle/声明扩展判定，不能给非法输入强行建立“有效”预期。附属 cash/equity/income 分录采用能明确平衡的合成输入。@@ 精确总价来自源 PriceSpec，不能从已舍入的 PerUnit 重算。

费用例子的建议源写法如下，金额为合成值。metadata 放在费用 posting 下，`trade_fee_for` 是产品分析约定：

```beancount
2000-01-04 * "sale with attributed fee"
  Assets:Stock -10 STK {} @ 20 USD
  Assets:Cash 195 USD
  Expenses:TradeFees 5 USD
    trade_fee_for: "Assets:Stock"
  Income:Gains
```

前序需有 10 STK×10 USD 的合法库存及账户 open。Income:Gains 的插值为 -100 USD，经济毛收益100、净收益95；不能因最终 Income 分录为 -100 就忽略现金扣除的5费用。额外普通消费或其他结算分录会使归因不确定，需建立负例。

### 26.3 数据契约与跨语言实施

1. 固定 `group_by` 及响应层级，列出 groups/events/lots/history 哪些是分页、哪些随所选组返回。全部合法请求参数有实际作用；当前尚未使用的参数不能被宣称已支持。
2. lot ID 使用完整精确键；projection/record ID 绑定快照；源交易分组保留拆分与重复记录。序列化字段保留必填 null，空集合编码为 `[]`，不用 null 代替无记录。
3. Go 计算原成本、目标成本、市值和未实现差额；使用共享 PriceAt，来源/截至日/缺项状态一起返回。持仓/平均成本历史/收益事件来自同一 Evaluation 和请求上下文。
4. Swift typed 模型先用真实 C 库载荷验证 `50/3`、`220/9`、零、null、label空/缺失、quote/source 及 reason。删掉必填 key、未知枚举、矛盾 completeness、改协议版本均应拒绝解码；不要使用 `try?` 补默认。
5. 扩展 Store 受控客户端：旧 cutoff 晚到、改变 group_by、holdings→account→Back、切账本和关闭。检查旧响应不改变新页数据、标题、页码或错误。
6. 加入原生持仓入口，再做分组列表、批次展开、来源、账户跳转、均价图及 exact 列表、收益未知原因。保留已有分析偏好，切 journal 时继续归一为该页支持的口径。

新增 UI 文件建议 `apps/macos/src/HoldingsView.swift`，命名可以调整。不得为持仓再建立一套独立 LedgerStore、HTTP 服务器或 Swift 金额计算。若兼容 JSON 只增加可选投影字段，可保持协议2；修改必填/字段含义属于不兼容变化，须同步版本及拒绝检查。

### 26.4 当前检查、集成和人工验收

局部开发按受影响包验证，不一次性并行运行共享构建目录的多份 Swift 构建：

```sh
go test -count=1 ./internal/ledger ./internal/report ./internal/desktop ./internal/bridge
go test -race -count=1 ./internal/report ./internal/desktop ./internal/bridge
make mac-test
make app
```

上述是待实施批次的检查命令，本次文档整理未运行这组完整检查。A02 集成时再运行第 23 节完整 gate 与 `make parity-full`。benchmark 增加实际持仓请求与响应样本，不能引用 A01 耗时代替新页性能。

GUI 使用公开连续 AVERAGE 输入：卖前→部分卖出→再次买卖→全清→重买逐日切换，核对精确文本/段、展开批次、收益原因、分组/币种/来源、返回与热重载；再检验最小窗口、大字、键盘/AX 及双窗口。截图/文件面板工具不可用时记录具体阻断，不填 GUI 通过。私有 V02 另扩展当前持仓/选定历史 cutoff 的独立预期与真实 Swift/C 聚合；不上传私人截图或输出账户金额。

## 27. 历史文档交接与当前入口

三个主文档共同规定行为、顺序、修改位置和独立预期，progress只记录实际实施证据。开始开发时先读progress与计划第15节，再从本手册第37节继续。第11节A02与第28/30/33节是早期批次步骤，已有实现不重做。文档中“目标”“拟议”“待实施”的字段和命令不能提前算作已实现。

以下为早期文档交接记录，StateTests编译失败已修复：当时holdings/replay targeted Go、公开真实 C/Swift ContractTests，以及外部账本同源 Go/Python 完整对照通过；47个文件、2831条指令、零错误/分录差异/库存差异，Expenses73行与Income22行一致，native_checked=false。原red保留在基线证据，不作为当前开工失败。最新原生三套及私人真实Swift/C范围见Q01记录。

文档完成检查包括：三文件互链、24 个顶层任务 ID 及依赖、来源引用、Markdown 代码块、当前状态、私人内容隔离、`git diff --check`。本次不改产品源码，不创建提交、PR 或发布；后续实现按文档中的任务门槛执行。

## 28. 当前 A02 Store/UI 的逐步实施

本节步骤已形成实现及自动化证据，保留为行为与接续依据。当前缺口是 GUI 渲染/键盘/AX 验收；性能复测与私人检查结果见 progress 最新记录，不能从步骤文字推断通过。

### 28.1 接续已存在的 red

1. 读取 `StateTests.swift` 的 holdings-navigation 段和 `LedgerStore.swift`；现有 `ControlledClient` 已支持按 view/cutoff/group_by/section 控制返回顺序。不要再建另一套客户端。
2. 扩充 NavItem 的 holdings 及全部 title/icon switch。Store 增加可见 holdings 投影访问器，保持 `visibleView` 的完整请求匹配条件。
3. 实现 showHoldingLots / showHoldingEventLots，必要时加 showHoldingHistory：只从当前可见响应进入，入栈完整原请求，offset=0、snapshot_id=该响应快照，分别固定 group_id 或 event_id；merged 详情显式设置 lot_kind。
4. 导航到 account/journal 等其他路由时清理全部 holding 参数。进入账户页保留共享阅读偏好，不能把分组或事件 ID 带入 account 请求。
5. 日期、账户、币种或分组变化时清除 selector/lot_kind/旧 snapshot 与页码；selected lots 回到 groups，单组历史回到 history 集合。详情的翻页/排序走专门动作，保留合法 selector 和快照，不调用会清 selector 的普通过滤路径。
6. Back 恢复原请求并应用当前语言/币种/分析估值偏好；快照已变时清过期选择器，回到相应主集合。reload/poll 的快照变化采用同一归一化规则；换账本/关闭清栈、取消排队请求、丢弃迟到结果。
7. 运行 `make mac-test`，检查三套程序全部实际执行；只修到编译通过不算完成。补 reload 后旧 ID、快速分组切换、切账本/关闭的具体缺口，再运行受影响检查。

当前测试的关键顺序是：root→lots（挂起）→account（先成功）→lots迟到→Back lots→Back root→merged lots（挂起）→改 cutoff（先成功）→旧详情迟到。最终可见请求、金额、页码和错误必须属于最新请求。

### 28.2 原生阅读页面

| 切片 | 修改位置与动作 | 实际通过信号 |
| --- | --- | --- |
| 入口 | 新 HoldingsView、main.swift 路由及 AccountDetailView 跳转 | 全局/账户入口使用同一 Store；未知或空账户不会读错目标 |
| 分组列表 | 共享读取控件、group_by、scope、groups 主分页；数量/成本/均价/市值/差额 | native、converted、partial、zero、plain/short/mixed 均有准确文本；不跨商品汇总数量 |
| 批次与来源 | lots 预览、完整详情与页面按钮、nullable label/date、origin、source | 预览之外批次可读取；无 label 与空 label 区分；报价 source_files 映射实际来源 |
| 归约事件 | events 主分页、gross/fees/net/原因、消费与 merge 两种详情 | 整池500与消费500/3不重复；转移/无售价/费用未知不显示为零收益 |
| 均价 | history 系列/选组点分页、Charts 按 segment、exact 伴随列表 | 清仓为 null 且断段；重买新段；同 cutoff 末点及手算值正确 |
| 操作恢复 | Back、分页/排序、筛选、reload、双窗口、错误/修复 | 不拼接跨快照页面，不恢复同下标的另一记录；关闭不新增请求 |

UI 可以拆小的 typed 子视图，以避免单个巨大 SwiftUI body；字段格式化复用现有 LedgerFonts 和金额模型。持仓不在 Swift 用 Double 做财务计算，报价/收益 unknown 枚举提供中英文说明及 AX 文本。无 payload 的 loading 不生成空仓或零收益。

来源补测：至少一个 include 文件报价，其 FileID 经 source_files 还原到正确位置；起止行列不指向入口文件。大合并池补测至少101个贡献批次，主金额完整、预览100、下一页1；这与已有101个普通消费批次检查是两件事。反复均价来源只承诺本轮重组的直接贡献，不伪称递归买入追踪。

### 28.3 集成退出条件

targeted Go/真实 Swift/Store 通过后运行第23节集成 gate 和 parity-full，再做公开 GUI。benchmark 增加 holdings/groups/events/history/详情请求，记录 payload 和所有样本；查询/价格尚未实施时不把它们算到已测试页面。

V02 对新增投影扩展独立的当前与选定历史 cutoff 对照，并在当前私人输入实际运行 Swift/C；只记录聚合成功/差异类别。本次 Go/Python 零差异不替代该项。G05 的原生曲线末点和 M 排除在这里完成验收，之前核心通过仍保留自己的范围。

## 29. A03–X02 的后续实施批次

这些步骤沿用设计第21–23节及本手册第18–22节，不新增顶层任务。每批把数据、C/Swift 模型、Store 和最小 UI 贯通；字段名称可调整，语义与失败例子不能省略。

| 批次 | 主要文件 / 顺序 | 独立验收与失败信号 |
| --- | --- | --- |
| A03.1 目录/选价 | `report.PriceAt` → `native_prices.go` → desktop 请求校验；已实现并有当前green，保留回归 | 无价商品可见；直接优先、倒数次选、同日最后源记录、未来价不使用；非法pair/参数报错 |
| A03.2 原生价格 | Models/ContractTests → Store 时序 → 建议 PricesView → 持仓报价跳转 | pair/cutoff 快速切换不串价；同日多个记录均可定位；quote 与分析币种偏好分开 |
| Q01.1 基础兼容 | reference 公开 query golden → `internal/query` 默认FROM/函数/meta → CLI tests | 每个函数/类型有正反例；meta 作用域按词法，未知函数/列不能返回零 |
| Q01.2 类型 | query 内值模型/列推断 → desktop Adapter → typed 模型；保留旧 CLI/Web Adapter | 重复列标题不丢值；多币种 inventory、完整成本键、NULL 与空集合正确 |
| Q01.3 聚合预设 | parser/eval 的 grouping/sorting/alias → 固定收入/支出/余额/来源预设 | oracle 类型/值/重数/顺序/错误全比较；无宽泛 query-subset 豁免 |
| Q02.1 工作台 | query 请求/结果 → 真实 C/Swift → Store Execute/取消展示 → 建议 QueryView | 编辑文本与已执行结果身份明确；错误不复用旧成功；换快照/关闭后迟到不发布 |
| Q02.2 预算/保存 | 文本/行/字节预算 → 边界样例 → fixed-result 导出入口 | 64KiB文本、10000行、16MiB目标边界实际测；截断不假报总数；进入C的取消能力准确记录 |
| X01.1 索引/DTO | 对照 `ledger/ast.go` 和 source Graph → publication 索引 → 各 kind payload | 各 AST 指令至少一例；生成与源记录区分；配置无日期，重复记录保持重数 |
| X01.2 原生浏览 | Models/真实契约 → Journal 模式/配置列表 → Store 分页/展开 | 热重载插行、重复内容、模式切换不串记录；原交易总数保持原口径 |
| X02.1 附件 | 建议独立 document 路径解析 Module → desktop 引用 → 外部打开动作 | 根内/缺失/越界/symlink 分开；打开前重新校验；不修改源文件 |
| X02.2 分析保存 | 先固定版本化文件规范/示例 → Go 精确序列化 → native 保存面板/原子写 | 空/partial/unknown/compound/分页上下文往返；取消与写失败不破坏旧目标 |
| X02.3 源导出边界 | 既有 convert/export → pinned Python round-trip / AVERAGE 明确拒绝 | 普通 v3 源结果可对照；AVERAGE 无法保持时失败，不因CSV成功声称无损 |

A03 复用本地报价，不另建网络价格客户端；Q01 保持 Go 无运行时依赖；Q02 复用原生会话，不通过本机 HTTP 调 Web；X01 不在 Swift 再解析源码；X02 不让通用文件写入进入会计求值路径。

每个新增页完成后补 N03/N04/V02/D01 的该页部分。最终 D01 逐项核对24个任务与里程碑退出条件，R01 本机独立包循环完成后再执行有条件的 R02/R03；目标系统/发布条件缺失时准确保留限制。

## 30. A03 原失败与逐步实施记录

### 30.1 当前代码与首个动作

初版已有 `internal/report/native_prices.go`、`internal/desktop/prices.go`、`prices_test.go`，session 请求和bridge错误；当时Swift尚未接入。初版下列命令exit 1：prices接受units。后续已拒绝该参数，接入Models/Store/PricesView与持仓跳转，8个价格及1个共用图表检查、真实原生三套通过。本节保留red及实施步骤，最新剩余GUI见A03证据；保留回归，不重做旧接口。

```sh
go test -count=1 ./internal/desktop -run '^TestNativePrices'
```

先在 `internal/desktop/journal.go` 的统一校验中把价格 valuation 限为空值/at-cost，再运行同一命令。不能把 units 正规化为 at-cost 以掩盖错误请求；Swift 新页负责主动生成合法请求。未知商品保留 target error，已知但无报价返回成功缺价。

### 30.2 固定输入与独立预期

复用已有公开 `native/prices.bean` 及 include，不复制私人数据。首批已验证目录与选价用例继续保留，以下表格的未覆盖格子先成为失败/边缘测试再实现；预期来自公开源 price 数值，不由 production helper 生成。

| 场景 | 请求/操作 | 固定预期 |
| --- | --- | --- |
| 全目录 | to=2000-01-08 | 10商品；STK有8条相关源报价、1个历史账户、1条声明和4个pair币种；目录包含已清仓、UNQUOTED、DECLAREDONLY及仅作为quote的JPY |
| 起始日不影响选价 | STK/USD，from=01-05，to=01-08 | 源历史0条；选用01-04直接11，age=4；较新的USD/STK与01-20未来99均不能覆盖 |
| 源报价重数 | STK/USD，from=01-04，to=01-04 | 三条10、11、11；10来源include，两个11来自不同源记录；全部ID不同，最后源11选中 |
| 倒数 | STK/EUR，to=01-08 | 源STK/EUR历史0；selected=1/2，inverse=true，source_pair=EUR/STK，date=01-06、age=2 |
| 零价 | ZERO/USD 和 STK/GBP，to=01-08 | 前者selected=0且有效坐标0；后者只有零倒数，selected=null/missing-local-quote |
| 时间缺价 | STK/USD，to=01-03 | future/direct/inverse都不可用，selected=null，商品仍可选 |
| 不隐式多跳 | STK/AUD 或 UNQUOTED/USD，to=01-08 | 已知商品但无可用本地pair；成功缺价，不是commodity-not-found |
| 窗口币种独立 | currency=EUR、price_quote=USD | STK/USD仍选11；切quote不改变分析币种偏好 |
| 分页/排序 | 三条源历史limit=1，下一页offset=1并绑定snapshot；再降序 | 页依次保留10/11/11，降序按反向源顺序；selected不受页码影响，page.total=3 |
| reload | 在仓库外公开副本追加price，再reload | 旧snapshot分页拒绝；新ID来自新snapshot；Back/重载不复用旧页面明细 |
| 参数与预算 | 未知商品/section、无base的quote、501limit、跨页无snapshot、超16MiB完整响应 | 分别结构化错误，无半截JSON；预算涵盖state/summary及C envelope |

日期列省略的年份/月均为2000-01。另建空合法账本和超过100个商品/500条报价的合成输入，验证空数组不是null、辅助预览截断明确、选中商品位于目录预览之外仍返回完整上下文。极大精确报价不能产生 JSON Infinity；exact仍保留，coordinate=null并断线。

### 30.3 Go → C → Swift 契约

1. 在 Models 增加 price 请求字段及 optional prices 载荷；新增 typed section、reason、Commodity、Record、Selected、Page。commodity/selected/value 使用 required-nullable，与现有 decoder 规范一致。
2. 在真实 ContractTests 覆盖表内直接/倒数/零价/缺价/include源；负向删除nullable key、未知reason、selected与pair矛盾、负age、非法page和非有限坐标。旧协议2可加 optional page 载荷；改变既有 required 字段语义时另升协议，不借optional规避严格解码。
3. 在 NativeClientTests 从真实 C Read 读取目录→pair→分页→切cutoff→reload；source 保留实际文件起止范围。bridge错误本地化包含commodity-not-found，不把它渲染为缺报价。
4. `make mac-test` 必须使 Contract/State/NativeClient 三套均实际执行；前一套green不替代后一套编译或执行。

### 30.4 Store 与 UI

1. 给 NavItem 加 prices；进入时 valuation=at-cost，但 analysisValuation 不受此中性请求更新。loadLedger/selectedNav/Back/request.didSet 等所有相关路径都要覆盖，而非只修一个按钮。
2. 增加 showPrices(base,quote)、showPriceCatalog、pricePage；离开 prices 清三类price参数，进入清holding selectors。保留窗口currency/locale，pair从price_quote读取。
3. 受控 StateTests：pairA迟到于pairB；新cutoff先回、旧cutoff后回；holding→price→Back；price→account；market分析偏好进入/离开prices仍保留；分页后reload/关闭不接受旧结果。
4. 新建 `PricesView.swift` 并接主路由：商品目录可分页，pair选择使用完整选中commodity的quote集合；图表与精确列表同时可读；显示selected方向、源日期、age、缺价原因和来源动作。图表注明当前分页范围，不以当前页冒充完整历史。
5. Shared ReadingControls 对 prices 保留日期/语言/profile，隐藏不适用的分析 valuation/currency 控件；从持仓报价跳入并有返回；inverse选用可单独进入原始方向历史。
6. GUI 用公开fixture检查实际目录、Picker、重复价/倒数、来源、日期、返回和热重载；捕获仍失败时记录具体工具错误，继续独立任务，不把该格标为通过。

### 30.5 当前批次收尾

先运行 targeted report/desktop/bridge 及 mac-test；实现稳定后执行全量Go、相关race、fmt/vet/license、parity-full、app构建。这些是后续要运行的门槛，本次没有据旧A02证据将其全部记为通过。

价格客户端基准沿用固定52,500交易输入并显式加入大量重复/同日报价，目录和历史各20个样本，保存payload/page计数及完整原始样本；在全部构建结束后测量。客户端p95与GUI首帧分别记录。V02新价格探针先在公开输入证明gold/缺价检查能失败，再使用私人临时捕获；仅保存商品/价格数量和差异类别，不记录币种对、日期、金额或文件路径。

## 31. Q01/Q02 可执行步骤

### 31.1 Oracle 与类型先行

1. 阅读 `internal/query/query.go` 当前 map结果与CSV Adapter，以及 parser/eval tests；按设计第21/25/27节更新已有[能力矩阵](query-capability-matrix.md)，现有功能标为“实现但未对照”或“固定oracle通过”，不一次宣称全部兼容。
2. 复用已有 `testdata/fixtures/query/`、`testdata/golden/query-v1/`、`tools/reference/generate_query_golden.py`及`tools/queryparity`。已有22个场景；新增复合类型和stack输入时明确扩展public manifest覆盖列表。生成只用于审阅更新，gate与oracle-check不能覆盖期望。
3. 语料至少包含：普通无成本/零成本、FIFO完整键、两币种同账户、相同标题两列、重复行、有效元数据栈/include、未按日期排列的源配置、空输入、NULL、错误函数/列/类型、除零、GROUP/ORDER/alias。记录列类型、位置值、重数、排序与错误类别。
4. typed值模型在query内部替换浅层map结果传递；将旧CLI/Web映射集中Adapter，保持已声明旧行为或给出明确重复标题错误。不要在desktop重新推断原结果已经丢失的类型。
5. 每个函数的重载分别对照：例如units(持仓)与units(库存)不能用同一个返回Decimal的假实现。meta词法传播要先于日期排序，在source/snapshot合法层完成，再由query读取有效值。
6. 将收入/支出/余额/来源预设保存为公开可执行查询文本，逐个与pinned oracle对照。UI选择期间时文本明确显示日期谓词。

### 31.2 查询工作台与预算

1. 按设计第25节固定请求和响应schema，再添加desktop route与错误位置；一次Read只使用捕获快照，旧profile/快照请求拒绝。
2. Models/真实ContractTests先green，随后Store草稿/执行记录与单一后台队列；最后实现QueryView。编辑草稿、成功零行、失败、旧成功、截断为不同状态。
3. 对64KiB UTF-8文本做边界及非ASCII测试；对10,000/10,001行验证returned/truncated；超完整16MiB返回错误。真实执行记录必须覆盖WHERE、GROUP BY和ORDER BY的大输入，不能只限制最后显示行数。
4. 受控时序验证执行A→B乱序、执行中切账本/profile、reload与关闭；真实C入口验证同步执行实际取消范围。状态测试只能证明不显示旧结果，不能证明底层计算已中断。
5. 导出固定成功执行结果，由X02序列化；不能将尚未执行的草稿或失败后的旧成功误称为此次结果。
6. 收尾检查query/CLI/Web影响包、严格query golden、真实原生三套、共享核心全量gate，再验公开GUI执行/选择/复制/取消/保存。进度逐条写出矩阵已通过签名与尚未支持类型。

## 32. X01/X02 与最终交接步骤

### 32.1 全指令及配置

1. 对照 `internal/ledger/ast.go`、parser支持类型与source Graph逐项列清单；新增公开all-directives fixture，各类型至少一条并含重复、pad生成和include/元数据栈。
2. 在发布有效快照时建立只读索引：日期指令按有效排序，配置按实际词法加载顺序；源/生成关联独立。非法新输入只更新诊断，不把半解析的新配置与旧金额混成同一有效页面。
3. desktop给每类指令 typed payload，metadata/tag/link/source不丢；通过真实Swift契约后增加Journal的transaction/all/config模式。已有交易计数和分页口径保持单独字段。
4. 验证相同内容两条源记录、插行、模式切换、过滤及reload分页；跨快照展开不能仅按数组下标恢复。来源路径只用于本地产品显示，私人证据不保存这些载荷。

### 32.2 附件与固定分析文件

1. 独立document路径Module接受声明路径/允许根，返回设计第26节状态；语义不放入ledger booking。公开临时目录覆盖绝对/相对/不存在/越界/目录/symlink，打开前模拟文件被替换。
2. 增加附件DTO、真实解码与状态动作，再加系统外部打开/Finder定位；打开前重新解析，失效引用仍可定位原声明。
3. 先制作设计第26节格式的公开version1样例与解包验证器，再实现Go文件内容生成。复用typed cells，不从Swift显示字符串恢复金额。样例至少一个零行和一个多币种/NULL/partial案例。
4. 通过规范与往返后增加native保存面板和同目录临时写/原子替换。写到受控失败writer/失效目录、覆盖失败、取消和导出途中reload，分别验证旧目标不坏、内容固定、临时文件清理。
5. 完整分析容器与独立CSV分开标注；保存成功必须对应实际写入产物。会计源convert/export用pinned Python round-trip另验，AVERAGE不可保持时明确拒绝，不由CSV green推断无损。

### 32.3 最终验收与开发记录

完整执行第23节门槛及新增query/document/export检查；以公开合成输入完成各页GUI、精确最小窗口/字号/键盘/AX、双窗口/热重载/错误恢复。生成独立开发包，在仓库外解压后打开公开账本，修改→错误→修复→关闭→再打开；确认所有进程/轮询释放。原用户应用和私人源文件保留。

D01核对计划24任务，不按最近一次green覆盖后来的改动。每条记录实现状态、当前/继承检查、具体GUI证据、适用机器及外部缺口。R02/R03只记录真实远端/机器/签名结果；缺条件时最终交付明确为本机开发包，不将未验目标系统写为已支持。

## 33. Q01基础批次的原实施顺序

本节为基础实现前的文档记录：当时仅修订文档、建立[查询矩阵](query-capability-matrix.md)并执行[公开探针](evidence/go-native-design-refresh-2026-10-01.md)，未修改产品源码；从默认FROM开工。以下顺序保留作回归与未完成项核对，不再作为当前待实现状态。

上述为本节初始文档交付状态。后续基础批次已实现并通过当前检查，见[Q01证据](evidence/q01-typed-query-foundation-2026-10-01.md)；已有query fixtures/golden、生成器、只读gate和typed CLI，不再将它们列为建议不存在的路径。

### 33.1 开工与一个纵向切片

1. 阅读当前进度、设计第25/27节、计划第15节；记录实际dirty工作区和涉及文件身份。保留用户修改，不clean/reset，不修改私人源文件。
2. 运行当前query基线 `go test -count=1 ./internal/query ./cmd/orangecount`。本次只运行了一个既有query错误用例，该命令完整query/CLI结果须在实施时实际获取。
3. 建议新增公开 `testdata/fixtures/query/default-source.bean`，含两posting、元数据及明确原始行；参考本次探针，不复制私人数据。先建立 `SELECT account, number` 缺省FROM的外部期望，必须观察当前“requires FROM”失败。
4. 新oracle生成器和strict gate路径在第31.1节基础上定稿。生成器记录包版本、输入SHA、query文本、列name/datatype、位置值及错误类别；gate默认只读golden。对golden故意改一格，必须非零退出，防止只检查valid。
5. 只实现默认postings的解析切片；显式FROM postings、未知表、WHERE/GROUP/ORDER/LIMIT及含隐式alias的语法都有回归。现有 `SELECT account postings` 拒绝用例可能受语法变化影响，先查oracle再决定测试期望，不能删测试来使green。
6. 通过针对性检查后记录产物、失败到通过的证据和未验项，进入类型骨架。建议路径是计划，未创建前不得宣称已存在。

### 33.2 类型骨架及迁移

1. 在query Module新增结构化Value/Column/Result；Result有序列和位置cell、compile schema，零行schema存在。第一批覆盖integer、decimal、string、bool、date、NULL及重复标题；dynamic metadata与compound重载随后补齐。
2. 类型检查先于扫描，错误传递不能被groupRows吞掉；无行时未知函数/非法类型仍失败。LIMIT 0和division-zero先用本次独立期望建立red再修，不在旧入口偷偷改变行为。
3. 用 `rg -n 'query\.Evaluate|query\.Result|WriteCSV' internal cmd` 盘点report、Web、import及CLI。明确哪些仍使用旧契约，哪些采用typed；新Adapter不把compound或重复标题静默压平。
4. 编写位置投影、typed往返和旧Adapter实用回归；新标准签名与旧avg/units/cost列扩展分别登记。CLI新增契约/格式名称、帮助、未知值和退出码统一验证后再记录真实命令。
5. Go/C/Swift采用同一判别联合；decimal/amount/position复用精确表示，inventory全币种/完整键；Swift未知type、required-nullable缺失、列数不符、cell与schema不匹配均拒绝。

### 33.3 来源、metadata和compound

1. filename/lineno列及abs/length已实现，公开行5/7指向posting。pad有效来源继承balance断言，第4行期望已有oracle与Go检查；无真实来源的其他生成记录另验nullable，不能把pad来源改为NULL。
2. 分别验meta、entry_meta、any_meta；补posting与交易同键、显式NULL、不同标量/Amount、缺键及零行类型。object column不从第一行推断声明类型。
3. stack fixture包含未按日期排序的交易、push覆盖、pop、include独立文件、重复include及非法stack。用锁定loader确认作用范围；在snapshot/ledger的正确阶段实现有效传播，并保护解析AST和legacy基线。
4. Amount/Position/Inventory按矩阵逐重载推进：units、cost、weight列、number、currency、sum。覆盖无成本/零成本、同价不同日期/label、多个成本币种、相抵空库存、重复posting。不实现锁定版拒绝的weight(position)来冒充标准。
5. GROUP/ORDER/alias/日期/NULL行为补strict golden；按月收支/账户余额/来源预设必须实际运行。完成后继续第31.2节Q02与第32节X01/X02。

### 33.4 验证升级条件

parser/类型切片先跑query和CLI针对性检查；旧Adapter改动增加report/Web/import相关包；stack传播或共享数值改动执行全量Go、核心race及 `make fmt vet license parity-full`。新DTO/Store在真实 `make mac-test` 三套实际执行后构建 `make app`；不并行运行共享build目录的原生测试、私人探针和benchmark。

修改影响私人求值或原生投影时，再运行现有 `make verify-private-ledger`（外部入口环境变量按本地配置提供）；只保留脱敏汇总。文档改动本身无需重跑全量产品门槛。阶段收尾依第23节扩展到GUI/安装，准确记录取消实际中断范围、查询聚合/排序预算与未完成的系统验收。

## 34. 从当前Q01基础继续

本节的Position/Amount/Inventory、units/cost/number/currency/sum(position/amount)、金额meta和排序已在后续批次落地，四个corpus共53场景通过，见[复合值证据](evidence/q01-compound-query-2026-10-01.md)。下文保留原实施顺序及文件责任供回归核对；当前入口是第36节。

当前代码入口为 `internal/query/typed.go`，共享执行链在 `query.go`，精确呈现在 `internal/presentation`；CLI显式 `--format typed-json`。public corpus在 `testdata/fixtures/query/`、`testdata/golden/query-v1/`，固定生成器为 `tools/reference/generate_query_golden.py`，Go gate为 `tools/queryparity` / `internal/compat/query.go`。已经有22场景和3个补充输入，不重新建立默认FROM或重做已修复red。

1. **下一切片：Position列**。用公开FIFO完整成本/date/label、无成本与零成本、两个币种建立独立oracle；先验证当前unknown-column失败，再在typed source schema和值模型添加Position/Amount。来源Span与财务键分开，不能把Span纳入group/inventory身份。
2. **units/cost/weight及聚合**。先units(position)返回Amount，再cost(position)、weight列、number/currency(amount)、sum(position)和units/cost(inventory)。每个重载完成后扩展生成器cell normalization、schema、公开golden和gate；inventory空值必须是空列表，多币种/完整键不降成一个小数。
3. **对象及元数据**。用户金额meta复用Amount cell；核实系统filename/lineno/meta和posting/entry/any的覆盖/NULL关系。stack/include先用独立词法输入确认，再修正确snapshot/ledger阶段，并复跑core及private。当前直接标量读取不证明有效stack已经实现。
4. **其余标准签名与NULL**。补unary、布尔/NULL literal、日期函数、动态比较、NULL逻辑/排序、min/max/first/last、全部声明重载及预设。不要把当前支持代码未经oracle的组合列为全部通过。float/int除法和非终止decimal的精度契约独立确定，不把Double引入财务计算。
5. **Q02接入**。上述模型与能力稳定后按第31.2节添加请求/快照上下文、C/Swift DTO、Store队列、工作台、64KiB文本/10,000行/16MiB预算和实际取消范围；再执行GUI，后续X01/X02和发布范围保持原计划。

当前可直接运行：`make query-parity query-oracle-check`、query/compat/CLI针对性测试。后续新增多fixture时扩展生成器及gate的明确public manifest列表，不接收私人账本作为golden输入。生成只在人工审阅更新时执行，正常gate不修改期望。跨共享核心/呈现/协议的改动升级到全量、race、parity-full、mac-test/app及适用的同源私人检查；不重复与改动无关的旧benchmark。

### 34.1 下一切片的文件责任与交接门槛

本表是后续实施清单，没有将建议新增的fixture或compound类型记为已实现。同一开发者可顺序承担全部责任；不要求并行agent。

| 顺序 | 负责文件/范围 | 实施动作 | 必须可审阅的结果 |
| --- | --- | --- | --- |
| 1 | `testdata/fixtures/query/`、reference生成器 | 新建公开复合值输入及manifest；用pinned oracle固定Position列的类型、值、成本日期/label与NULL | 独立期望包含无成本、零成本、同价不同日期/label；来源路径只在声明列规范化 |
| 2 | `internal/compat/query.go`、query golden及`tools/queryparity` | 扩展Amount/Cost/Position/Inventory规范化，严格比较类型、完整键、重数和顺序 | 篡改一个成本日期或label时真实gate非零退出；正常检查不写golden |
| 3 | `internal/query/typed.go`及共享执行链 | 先使Position的外部接口测试失败，再添加编译schema、值与cell；复用同一执行链 | 零行有schema；非法参数即使空输入也失败；分组不含Span；精确值不经过浮点数 |
| 4 | query/CLI测试与旧调用方 | 验证typed-json真实载荷，保留旧json/csv回归；每完成一个重载再扩展覆盖 | 重复标题不丢值；未支持能力报准确错误；不静默把复合值压成单币种数字 |
| 5 | 查询矩阵、progress及本计划 | 标明本批实际通过的签名、当前输入/源码身份和未验项 | targeted及strict oracle结果可复验；没有原生DTO/UI时不写跨语言或GUI已通过 |

每次交接写出下一条尚未通过的具体行为，不仅写“继续Q01”。若fixture揭示新的账务语义问题，先归属G/V任务并给出最小复现；若只涉及查询值表示，留在Q01。预算、取消、保存面板与实际UI验收在Q02实施，不能用当前CLI读取速度代替。

## 35. 从复合查询值继续完成Q01

本节步骤1/2的已验子集已在本轮实施，见[元数据证据](evidence/q01-metadata-query-2026-10-01.md)。本阶段的63个场景继续保留，当时生成器与Go gate注册7个public输入；当前91场景和8组以第37节为准，schema 2还绑定完整include源图；缺文件或子文件字节变化即失败。重复include等未验子项移入第36节。源总价和精确会计值仍保留，typed报价weight单独使用已归一单价及28位half-even。

1. **系统元数据与NULL优先级**：从`lookupMetadata`、标准source rows及实际booked posting核对；新增公开posting/交易同键、缺键、显式NULL、filename/lineno等系统来源案例，用固定oracle确认读取及覆盖规则。Amount值已实现，不重做；不能把不支持的系统对象错误包装成“缺键NULL”。逐例产生red，再在合适数据层实现。
2. **词法有效传播**：检查parser/source/snapshot已有pushmeta/popmeta/include处理，不凭旧状态标签推断是否已经支持。公开输入含未按日期排序的交易、同键嵌套覆盖、pop、include文件作用域、重复include及非法stack。先核实锁定loader的实际规则，再修求值副本；有效来源和全指令配置列表共用规则，不改共享AST或用户源。涉及核心时执行全量、相关race、parity-full及同源私人对照。
3. **NULL与动态对象**：补unary、bool/NULL literal、AND/OR传播、metadata动态比较、排序与非法类型。列schema在无行时仍来自编译，object cell保留真实类型。固定版拒绝的复合直接比较保持拒绝；ORDER BY的比较规则与财务身份不能混用。
4. **剩余签名与预设**：逐个补日期函数、声明聚合、balance/sum(inventory)等矩阵项。balance的更新时点、WHERE过滤、跨账户累积和重复选择先用oracle固定，不能复用旧每账户显示字符串。收入/支出/余额/来源预设提交可执行查询文本，实际逐条对照。
5. **数值精度契约**：高精度成本乘法、一般decimal算术及聚合与固定Decimal上下文分别建独立输入；明确兼容字段与精确扩展的关系，保留AVERAGE守恒，不以报价weight的已通过案例扩大为全局舍入。integer除法对应float仍需先明确cell/产品契约，不能转Double后假称财务精确。
6. **Q02及后续**：Q01声明范围检查收敛后，按第31.2节添加同快照请求、错误位置、C/Swift DTO、Store队列、QueryView、真实预算/取消与GUI，再执行第32节X01/X02。现有原生三套通过只证明现有页面，不能提前验收这些新路由。

每个切片更新能力矩阵、CLI帮助及progress的当前源码/输入身份和未验项；保留已通过63场景。旧GUI捕获错误是历史证据，实际进入N04/R01时重新检查环境，不因等待GUI/远端证书停掉上述独立实施。

## 36. NULL、动态类型批次步骤与剩余Q01

设计第28节、计划第16节及[metadata证据](evidence/q01-metadata-query-2026-10-01.md)是基线。第35节系统来源/stack/pad已验子集保留，不再重做。本节36.2的动态推断、NULL/逻辑及unary已验子集在后续批次落地，见[最新证据](evidence/q01-dynamic-null-query-2026-10-01.md)。下文保留该批步骤；当前入口第37节。两object直接比较由固定版拒绝，一侧静态类型则允许已核实的显式语义转换，不能用通用coercion代替签名绑定。

### 36.1 开工及测试接口

使用当前dirty工作区，记录参与文件身份；不要checkout HEAD代替现有实现。已授权测试接口为snapshot Build/BuildGraph、query EvaluateTyped、公开compat gate，以及后续真实C/Swift/Store。通过这些接口观察行为，不测试私有函数形状。

```sh
go test -count=1 ./internal/query ./internal/compat ./internal/snapshot ./cmd/orangecount
make query-parity query-oracle-check parity-full
go run ./cmd/orangecount query --format typed-json --semantics v3 \
  testdata/fixtures/query/metadata-stack.bean \
  "SELECT narration, entry_meta('scope') AS scope"
```

本阶段开工时基线为63查询、50普通会计案例和单列AVERAGE扩展；当前从第37节91查询基线继续。不要将上述账本替换成私人输入来生成期望。

### 36.2 首个纵向切片：动态metadata比较

1. 阅读 `internal/query/typed.go` 的binary编译、object cell、lookupMetadata、compareTypedValues，以及 `query.go` 的比较/分组/排序；列出仍进入旧coercion的位置和旧调用方依赖，先不重构执行链。
2. 新公开fixture包含数字字符串、Decimal、false、空字符串、NULL、缺键和Amount。每个query先在锁定BeanQuery执行，记录声明类型、实际值、排序或错误类别，不把SQL常识当作期望。
3. 选择一个oracle与Go实际不同的比较用例写入manifest，运行gate观察red。已经一致的用例作为覆盖保留，另选未满足行为；不人为改期望制造red。
4. 编译决定重载，object按运行值实际类型处理；NULL传播和类型错误采用独立已核实规则，不以字符串转数字掩盖不一致。用户Decimal与系统Integer分别保留。
5. 观察WHERE/GROUP/ORDER中相同行为，零结果不能吞签名/类型错误；分组用规范类型值，排序不修改财务身份。只实现当前失败所需规则。
6. 一个切片green后再扩展unary、bool/NULL literal、AND/OR与NULL排序；逐例核对oracle，保留旧标量、compound、metadata语料，不留永久skip。

### 36.3 Loader长尾：先定位再修订

| 未验行为 | 最小输入 / 观察 | 修改位置及边界 |
| --- | --- | --- |
| 重复include | 同一公开子文件被包含两次，比较加载错误类、行0与entries重数 | Graph含边；v3校验与legacy隔离。不能靠visited静默消掉错误 |
| 重复metadata | 同交易或posting两次同键；跨scope同键另例 | 显式值校验，错误类/头行按oracle；any_meta优先级不等于重复键许可 |
| 保留filename/lineno | 普通、pad、显式用户覆盖分别查询列与meta | 查询来源/导航分开；逐例核实，不能全局改Span |
| 同日跨文件顺序 | include置于父交易前/后、同日期不同源行 | 核对loader排序键、posting和booking结果；影响会计时归属G/V并更新revision |
| 非字符串未平衡stack | 数字、Amount、NULL push后不pop | 记录上游诊断或异常；Go安全拒绝，不把上游崩溃虚构成ParserError golden |
| 系统容差map | 普通entry存在、posting/pad缺失、用户同键 | 当前明确unsupported；支持前先定义map cell及往返，不对象字符串化 |

这些是未验项，不是50/63门槛的豁免。失败应在最早正确层报告，扩展双语修复指引；非法快照不能替换会话最后有效快照。

### 36.4 签名、balance、精度及预设

逐一核对日期函数与min/max/first/last等聚合签名，pinned不接受的重载保持拒绝。typed balance不能复用旧每账户字符串：先确认累积对象、更新时点、WHERE前后和跨账户边界，再决定Inventory运行值。sum(inventory)先确认固定签名和空值规则，不从sum(position)推导。

高精度输入分别检查成本乘法、一般Decimal算术、聚合与非终止结果，每个兼容字段明确Decimal上下文。精确会计及AVERAGE守恒不可通过统一舍入破坏。Integer除法若排除Float则帮助明确拒绝；若纳入则先定义独立Float cell，不让浮点金额参与财务计算。

预设提交可执行文本：收入、支出、账户余额、来源检索，注明列与范围。每条在公开输入对照oracle并经CLI执行；帮助只列已验签名。之后继续第31.2节Q02、第32节X01/X02，不把旧入口可用查询无验证复制到标准入口。

### 36.5 文件责任、验证及交接

同一开发者按下表顺序执行，无需并行agent；按具体失败扩大修改范围。

| 文件责任 | 验证信号 |
| --- | --- |
| query fixture/manifest、固定生成器、golden | 版本、完整源图SHA、列位置/类型、重数/排序/错误；显式regen审阅 |
| query typed/compound、共享执行链、CLI | 最小red→green、零行签名、动态类型、旧json/csv及调用方回归 |
| ledger/snapshot/source、diagnostic/repairguidance | 完整oracle、有效/非法快照、AST/legacy隔离、双语指导及触发fixture |
| 后续desktop/bridge/Swift Models/Store/UI | 同批契约、真实三套原生检查、预算/取消/乱序和GUI |
| 主文档/matrix/progress/evidence | 只升级有证据的状态，明确下一行为、身份及未验范围 |

跨核心改动执行下列门槛；原生测试与app顺序执行，私人探针使用最新archive，不与benchmark争用build目录：

```sh
go test -count=1 ./...
go test -race -count=1 ./internal/query ./internal/compat ./internal/ledger \
  ./internal/report ./internal/snapshot ./internal/desktop ./internal/bridge
make fmt vet license parity-full query-parity query-oracle-check
make mac-test
make app
```

私人检查使用 `make verify-private-ledger`，须显式设置 `ORANGECOUNT_PRIVATE_LEDGER` 为用户已授权的外部入口；未设置会退出2。先核对 `apps/macos/private-read.sh` 当前路径/参数。只提交计数和差分类别摘要，私人源数据、路径列表与哈希不入库。该命令会再跑mac-test；若本批已完成原生门槛，可像当前证据所述只重编PrivateReadProbe后执行differential，准确记录archive及探针身份。

文档本地链接/锚点、代码块、shell语法、24任务ID及依赖必须一致。旧证据哈希保留历史，新记录单列当前身份；文档修订不触发无关产品复测。完整目标仍包括Q02/X01/X02、未验GUI/安装与R02/R03，不能标记目标完成。

## 37. 标量转换实施记录：原red与验证步骤

**本批结果**：本节标量cast与Decimal上下文已实现，116场景及适用门槛通过，见[cast证据](evidence/q01-casts-query-2026-10-01.md)。本节下文保留开工时的失败和步骤；当前继续第38节。

本节开工依据为[本次交接证据](evidence/go-native-development-handoff-2026-10-01.md)及设计第30节。九个注册public corpus共101场景，100匹配、1失败；oracle golden自身全部复验通过。`internal/query/casts.go`和cast基础已存在。第36.2已验object/NULL子集继续保留，36.3 loader长尾及36.4数值上下文尚未完成。下面步骤保留开工过程；前次只写文档，本次已实施并修复该批标量差异。

### 37.1 保存当前输入与实际失败

保留当前dirty工作区；记录公开文件SHA和HEAD，不能checkout HEAD替代工作区。无需再次执行所有历史基线步骤，先运行当前失败的公开入口：

```sh
go test -count=1 ./internal/compat -run '^TestQueryOracleGateChecksExplicitCasts$'
make query-parity
make query-oracle-check
```

前两条在开工时失败，修复后应通过；原ID为`casts/arithmetic-decimal-string`、类别execution-error；第三条当前通过，说明期望与固定oracle一致。分别运行并记录退出码，不能用前一条失败导致后续命令未执行，再称全部检查通过。normal gate只读golden。

```sh
go run ./cmd/orangecount query --format typed-json --semantics v3 \
  testdata/fixtures/query/casts.bean \
  'SELECT str(decimal(1.00 + 2)) LIMIT 1'
go run ./cmd/orangecount query --format typed-json --semantics v3 \
  testdata/fixtures/query/casts.bean \
  'SELECT date(2000, TRUE, 1) LIMIT 1'
```

开工时第一条成功但返回`3`，第二条拒绝；现已分别正确返回`3.00`和2000-01-01，且已进入新增cast严格场景。开工101计数不包含它们，当前总数116。

### 37.2 将差异变成独立可重复的验收输入

1. 阅读`testdata/fixtures/query/casts.queries.json`、对应golden、`typed.go`、`coercion.go`与`casts.go`最小相关路径。以EvaluateTyped/公开compat gate观察行为，不按私有helper结构写镜像测试。
2. 在cast manifest加入组合Decimal转换与date bool每个参数位置；另加wrong arity、NULL literal与nullable integer、日期边界、非法字符串及零行签名反例。先运行锁定oracle，确认实际类型/NULL/值/异常，不凭Python或SQL常识决定期望。
3. 对Amount/Position/Inventory转字符串及算术源精度另设最小输入。普通值、零成本、空库存、日期/label、源顺序与canonical顺序分别验收；不要把全部对象交给fmt字符串化。
4. 只显式生成本corpus并审阅fixture、manifest、源图SHA、列位置/类型、行重数与错误类；不修改其他通过golden。

```sh
uv run --offline --project tools/reference python \
  tools/reference/generate_query_golden.py --corpus casts
go test -count=1 ./internal/compat -run '^TestQueryOracleGateChecksExplicitCasts$'
```

这是该批实施命令；前次文档整理没有生成golden，本次只对casts显式生成并审阅。先证实新增案例真实失败，再实现；若某例已一致，保留覆盖并选择另一个实际失败，不改期望制造red。

### 37.3 实施精度传播及参数绑定

| 文件责任 | 具体工作 | 必须保持的行为 |
| --- | --- | --- |
| `coercion.go` | 统一隐式/显式有限Decimal转换与表示信息；处理metadata、字符串、bool | 财务身份与源表示分开；无效文本NULL；非有限明确拒绝 |
| `typed.go` | literal/字段/转换/算术传播表示；清除computed Decimal最简字符串成功兜底 | 空输入仍编译；group按规范数值；旧执行链影响有回归 |
| `casts.go` | 三参date接受固定版确认的integer/bool；逐参数读取，NULL与越界分开 | 不对bool做Decimal断言；不将编译错误伪装NULL |
| `compound.go`及typed转换 | 若纳入复合str，按固定格式输出，保留原typed结构 | 完整批次键、空与零、排序规则不改变 |
| fixture/manifest/golden/matrix | 新增正负例、记录能力边界与源码身份 | 注册corpus不可缺；不放宽比较或删除失败 |

加/减/乘在未舍入条件下推导Decimal指数，负零与科学记数法用独立oracle校验。除法、高精度与SUM先明确参考上下文，再实现；不能将现有财务Decimal统一降为28位或转Double。无法表示的组合显式报能力错误，不返回错误文本作为成功结果。复用一套转换规则，不在cast和operator各自维护不同实现。

### 37.4 验证与升级

```sh
go test -count=1 ./internal/query ./internal/compat ./cmd/orangecount
make query-parity
make query-oracle-check
```

通过条件：原有101场景无失败，新增日期/组合转换正负例全部匹配；正确值之外还检查类型、NULL和列标题。更新数字时计入真实manifest案例数，不预先写某个将来的总数。

共享执行链、数值语义或ledger/source改变时按36.5升级全量Go、相关race、fmt/vet/license、公开会计与查询门槛；新核心行为还检查legacy隔离、revision和只读私人同源。原生契约/客户端改变则依次执行`make mac-test`和`make app`，私人探针必须链接最新archive。文档修改自身只需要链接/锚点、代码块、shell语法、任务号/依赖及证据身份检查，不用无关GUI测试替代文档验证。

### 37.5 完成当前切片后继续的步骤

1. **Q01剩余签名**：日期及min/max/first/last等逐重载固定正负例；nested aggregate、group legality、alias位置和零行编译先对照。固定版无abs(integer)重载，继续拒绝。
2. **balance与sum(inventory)**：确认累积对象、WHERE前后、跨账户与重复posting顺序；完整成本/date/label及清零重买独立输入。不能从旧每账户字符串或sum(position)推导实现。
3. **精度与预设**：分别固定查询一般算术、成本乘法与SUM上下文；运行收入、支出、账户余额、来源检索预设。旧CLI/Web Adapter、标准typed协议与会计profile分别记录。
4. **Q02**：按31.2节串联desktop/bridge→Swift Models/Client→Store→QueryView；固定快照、文本/行/字节预算、真实取消、错误字节位置、typed表及CSV。受控时序和真实C/Swift后再做GUI，不能用现有价格页测试代替查询验收。
5. **X01/X02**：按第32节完成全指令/配置索引、附件路径/失效状态及固定分析导出；分页与导出绑定快照，失败/取消不毁旧文件。
6. **N04/R01/D01及R02/R03**：全页窗口/字体/键盘/AX、独立安装阅读/错误/恢复/退出循环；远端run、旧系统/另一机器、签名公证按实际条件完成。历史截图失败只记录发生过的错误，当前GUI工具可用性需要届时验证。

每批证据记录实际命令、退出码、公开源身份、继承范围和下一具体行为。上述全部仍属24任务目标，不能因当前转换修复、单个账本一致或文件交付便关闭整个项目。

## 38. 复合转换/上下文实施步骤（已验范围见本批记录）

开工基线：query/compat/CLI、全部116 strict场景及oracle核对通过；当前源码全量/七包race、50会计与AVERAGE、原生三套/app、私人真实Swift/C亦有本批证据。标量cast和算术不重做；metadata/source与legacy隔离继续保持。完整Q01/Q02、X01/X02及原生/发行剩余门槛未关闭。

### 38.1 下一个纵向切片

1. 用公开casts输入核实`str(position)`、`str(weight)`、`str(sum(position))`，固定类型、空库存、顺序及文本。此前oracle首行1 USD、1 USD、(2 USD)已有探针，仍先放进manifest并观察当前Go实际red，不把历史探针exit0当作本次Go通过。
2. 同一行为green后扩展公开cost输入：省略/显式日期、nil/空label、无成本/零成本、不同币种与源数量精度。补`str(number(units(position)))`，确认amount提取Decimal时保留参考表示，而财务group/key不受源精度影响。
3. 修改责任落在`compound.go`的运行值/转换与typed绑定、`decimal_text.go`的query上下文、`query.go`的posting投影；规范表示不得混入lotKey、canonicalInventory或PresentedDecimal的exact身份。Source Span仍用于导航，与字符串结果身份分开。
4. cost(position)、weight有成本分支、units/cost(inventory)、sum(amount/position/inventory)分别观察固定版本的乘法/累加上下文。不要因标量28位通过便全局调用舍入；原始ledger、余额和AVERAGE不被改写。
5. 任一新兼容字段改变时审阅普通公开oracle、完整批次和现有Compound gate；新增期望只更新相关注册corpus，旧通过语料全部保留。非终止精确扩展值与标准Decimal上下文分开声明。

### 38.2 后续签名、库存和预设

按36.4及37.5逐重载补日期、min/max/first/last、group/alias合法性，先编译后扫描；balance明确累积/WHERE/跨账户时序，sum(inventory)独立验收。查询极端指数/上下文陷阱仍要定义或明确拒绝，不将已验有限标量子集宣称为任意Decimal支持。可执行收支/余额/来源预设经公开oracle与CLI验证，矩阵和帮助只列已验签名。

### 38.3 原生及完成门槛

稳定typed值后继续31.2节Q02，新增真实快照执行、预算、取消、错误位置和CSV；检查深表达式的重复求值成本，在读取/扫描/聚合/排序中执行实际Context预算，不把丢弃迟到响应称为中断计算。随后32节X01/X02，按计划18节包4–8完成原生全指令、附件/导出及N04/R01/D01/R02/R03。不同页面与发行门槛各自实际验收，不缩小24任务范围。

新query切片先运行query/compat/CLI与两个query gate。影响共享求值/数值时升级37.4/36.5的全量、race、会计、legacy及适用私人检查；原生协议/页面同步三套Swift/C、app与GUI。每次交接记录确切仍缺的签名、输入和实际结果，源码身份变化后旧证据保留历史。

## 39. 当前入口：完成剩余查询契约并接入原生工作台

38节已验复合转换、金额提取表示、完整/拆分FIFO表示、成本/库存上下文及自动分组已有127/127严格场景和本批完整门槛，见[证据](evidence/q01-compound-casts-query-2026-10-01.md)。旧38节开工116计数保留历史；不重复已修复的str或库存精度。

1. **报价与表示长尾**：公开fixture核实无成本@/@@ weight的str/number表示，包含2.00单位报价、1.00总报价/2单位、负数量/报价、零数量与高精度。核对Number.DecimalText、Cost.DecimalText与Decimal内部表示在快照复制中的身份及精度，源AST仍不被修改。其他插值/总成本不在精确表示网格时先对照实际参考，不通过显示舍入偷偷改变金额；涉及会计行为时明确profile/revision与扩展取舍。
2. **剩余最小查询能力**：日期函数与min/max/first/last逐重载核实；显式group的非法投影、aggregate位置、嵌套、alias/ordinal与零行校验逐项固定。balance确认全局/账户累积、WHERE与返回时点，sum(inventory)单独验收。收入、支出、余额、来源预设经公开oracle和CLI实际执行。
3. **资源与协议**：保持现有typed columns/cells，按31.2节实现Q02快照执行、预算、Context取消、错误位置、后台队列与CSV；检查深表达式重复求值及Decimal表示运算成本。稳定基础上可以先做协议与受控Store，但不能用丢弃响应或文本LIMIT假作真正中断计算。
4. **原生页面和记录**：Go/bridge→Models/Client→Store→QueryView与真实C/Swift时序；随后32节X01全指令/配置、X02附件与固定分析导出。所有分页/导出绑定当前快照，保存取消和写失败不毁旧文件。
5. **退出及发行**：按计划固定24任务，继续N03/N04/V02/D01随新增页验证；全页GUI/字体/窗口/键盘/AX、独立包安装阅读/错误/恢复/退出及R02/R03真实环境证据保留。不能将查询子集通过当作整个Q01或M4完成。

每个纵向切片先独立期望/真实red，再最小实现和targeted；共享ledger/query表示改变升级全量、race、公开差分、legacy/AST及私人同源；原生改变顺序mac-test/app与GUI。证据记录实际输入、源码、退出码和未验项，旧哈希保持历史。当前十一组golden均必需，缺compound-casts或aggregates也必须失败。

39节首批报价、日期/聚合切片已实际完成：见[当前证据](evidence/q01-date-aggregate-price-2026-10-01.md)，147查询/51会计通过，零数量总报价会计修订为core-3。继续第2项的分组/alias/ordinal、balance/sum(inventory)与预设；第1项复制/数值长尾保留待验，不重做已验quote str或year/month/day及聚合端点。资源与协议按第3项及31.2推进。

### 31.2当前接续点（Q02首批后）

后端有界执行与真实C/Swift模型已落地，见[证据](evidence/q02-bounded-execution-2026-10-01.md)。下一步将execute/cancel接入NativeLedgerClient，取消必须能绕过阻塞的同步C执行；实现Store单调代次、草稿/成功/失败/旧结果、current/latest-pending与切账本/reload/关闭时序，再交付QueryView。error.range当前始终null，须按实际token位置补齐；CSV冻结成功执行并原子保存。原生完整资源边界、实际中途取消及GUI仍须验收。
