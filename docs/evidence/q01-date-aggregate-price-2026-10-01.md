# Q01日期、聚合端点与报价精度；v3零数量总报价修正

日期：2026-10-01。沿[实施39节](../go-beancount-native-implementation-2026-09-30.md)继续实际开发；[前批复合转换证据](q01-compound-casts-query-2026-10-01.md)保留历史。完整24任务目标未完成，没有提交、GUI启动或发布，个人账本只读。

## 实现与边界

1. **报价表示**：typed `weight`的无成本`@/@@`分支保留源数量/报价的Decimal表示。`@@`先用绝对数量作上下文除法，再乘带符号数量；两个操作应用锁定28位half-even规则。`str(weight)`与`str(number(weight))`分别保留金额/Decimal文本。重复小数、正负数量、零数量、零报价、直接/总报价及高精度有公开参考。源精度文本不可用时沿用BookedPriceWeight的数值fallback，不宣称其源表示与标准Decimal等价；极端指数尚未验收。
2. **聚合端点/极值**：typed新增`first/last/min/max`。FIRST跳过前导NULL，LAST包含末尾NULL，MIN/MAX跳过NULL；相等数值保留首次表示，不根据规范财务值重建文本。独立参考覆盖Decimal、string/date/integer/bool、Amount/Position、动态metadata及NULL、普通/聚合自动分组、空集schema、零行wrong arity与nested aggregate。Inventory及动态混合类型的所有比较组合未全验，不能称Any完整兼容。
3. **日期提取**：typed `year(date)/month(date)/day(date)`返回integer；运行时nullable date传播NULL。日期边界、组合聚合、空输入非法integer/object/NULL literal与wrong arity有参考。其他日期函数没有因此加入声明范围。
4. **零数量总报价核心修正**：v3把`0.00 STK @@ 1.00 USD`的booked单位报价归零，财务balancing和公开PostingWeight均为零；保留原始Total/Amount与源AST。`@@0.00`亦通过。之前误计原始总金额导致合法交易无效，故v3会计修订升至`v3-core-3`；快照/会话现有身份规则自动纳入此修订。默认和`legacy-1`未变，同一例legacy保持旧无效结果。查询contract仍为`beanquery-0.2.0-typed-v1`、原生协议仍2。

快照Build/Evaluation/Parsed公开入口已检查源总价/AST不被原位替换；当前查询golden也实际通过snapshot副本。Decimal私有表示被内部深复制清除的代码路径仍需在其具体使用接口验收；没有因未观察到当前posting查询差异便宣布全部复制路径完成。其他插值/总成本网格、loader长尾、分组/aggregate位置/alias/ordinal、balance/sum(inventory)、预设及Q02/X01/X02仍缺。

## 独立期望与真实red

| 输入/门槛 | 实现前实际结果 | 实现后 |
| --- | --- | --- |
| price-weight/quote-weight-text | Checked3/Failed1，typed-result-mismatch | 3/3；6.00000、1.00、0.0000、-6.000及重复小数表示保持 |
| aggregates/scalar-extrema-and-endpoints | Checked1/Failed1，execution-error | 初始1/1；随后扩展独立参考到11/11 |
| casts/date-components | Checked27/Failed1，execution-error | 日期提取green后边界/负例扩至casts34/34 |
| zero-total-price.bean | `make parity-full` exit2，底层exit1；51中失败1，validity/diagnostics/entries/balances不同；含此输入query gate build失败exit2 | 51/51；零数量booked quote为0、库存/有效性与参考一致 |

高精度报价、其他类型/NULL/非法签名等在相关基本切片green后补独立参考覆盖；不宣称每个补充例都先red。负报价的固定oracle探针报告ParserError/ValidationError，本批没有虚构接受负报价的合法fixture。

## 当前验证

- `go test -count=1 ./...` exit0。
- 七包race（query/compat/ledger/report/snapshot/desktop/bridge）exit0。
- `make fmt vet license parity-full query-parity query-oracle-check` exit0：普通会计51/51，独立AVERAGE/global booking回归通过，查询147/147、十一组锁定oracle一致。
- `make mac-test`后`make app`顺序exit0；ContractTests/StateTests/NativeClientTests实际执行，arm64/ad-hoc开发包验证；最低编译目标13不代表旧系统实机通过。
- 最后仅补公开price-weight高精度/零总价行后，query/compat/CLI与query-parity/query-oracle-check再次exit0；产品源码未再改动。当前147 = 22+32+3+1+4+3+3+28+34+6+11。
- 临时外部aggregate golden把MIN所选文本1.0改1，真实CLI exit1，Checked11/Failed1、typed-result-mismatch；临时文件删除，仓库期望不改。
- PrivateReadProbe重新编译、链接本批mac-test archive，随后私人同源differential exit0；same_input_capture/native_checked/native_contract全部true。

私人只保留聚合：47文件；Go/Python双方2831指令、errors0/valid=true；missing/extra entries与changed inventory accounts均0；Expenses73行/Income22行一致；三个holding cutoff与48价格可用性检查。没有个人账户、金额/日期明细、路径列表、哈希或截图入库。此私人检查不包含新增typed查询、零总报价公开反例或AVERAGE的私人验收。

无本批GUI、性能、独立安装、远端run、旧系统/其他机器、Developer ID/公证证据；这些门槛继续保留，不由自动化或单个账本相等推断完成。

## 下一步及公开身份

继续实施39节剩余分组/aggregate位置/alias/ordinal、balance/sum(inventory)时序与预设；在稳定typed schema上按31.2节推进Q02实际执行预算/Context取消、协议和原生工作台。复制/数值/loader长尾按最小实际输入验收；X01/X02及N04/R01/R02/R03/D01完整范围保留。不要重复已验日期和聚合切片，也不关闭完整Q01。

| 公开文件 | SHA256 |
| --- | --- |
| `internal/ledger/price_normalization.go` | `e68d405d95b7c9f59526edf4dddeb5834223ea9256916e895ebf2b588d810144` |
| `internal/ledger/evaluator.go` | `17f19eba7b77cc6961cfc1942c74a3ae1c2d4cc8864dfb18a0df7fb07cc2b983` |
| `internal/ledger/semantics.go` | `46bba397f19c9d7054d2ea5b59fc2ef9fb7b457b27b422b1da97fd33425aed12` |
| `internal/query/query.go` | `85cc68bb1e879451870f09c4dad025aab4075cfa36b1a7851c3fa200a9eaa972` |
| `internal/query/typed.go` | `4931865ca48cba2f0d458eed0b7a141e93426305d751f61f7e5da6fb6e016902` |
| `internal/query/compound.go` | `d301e814befcd114fe1b38add52bdc140a712cddb06c6547bb18f1b3627f4449` |
| `internal/query/casts.go` | `3ec97ee0905c32b5942d8e835c12d01c57f70acf2c5de4d001d6c2419a541a15` |
| `internal/query/aggregates.go` | `94399d2fbee0008ab1379e5a3ffad22eabe92bf4bf4559e8fa7ae65ef2394bb9` |
| `internal/snapshot/price_v3_test.go` | `4b588f3f6ea219f8ea01dd18ebbaa6ab549ff2b68f58f821b0c8b9cf3693e1eb` |
| `internal/compat/query.go` | `a987cba9e40be5b38f81b697390a68f611ac5738163fafdb8d62409f1df587d1` |
| `internal/compat/query_test.go` | `a62d69d04755cae43c69c636f35c72840a7917a77e694b4cd20628f67d6d7207` |
| `tools/reference/generate_query_golden.py` | `347721d8499303540d227b3b0cdfc6fe6c9583593654e9bbf0774f040bb711bd` |
| `tools/reference/uv.lock` | `60acbc24073541a373b032e1e8c9a21949c8b354cbc1f77b6f0553609f8931ec` |
| `testdata/fixtures/semantics/zero-total-price.bean` | `ec10d341bbc5ec6c05d5fa4505e97177438e7786b51c183d7b3f4e11ecc0ff1d` |
| `testdata/golden/v3-full/zero-total-price.golden.json` | `3641bf32ec0325ecf71f063ef003f60031bf0f9ea974cc825cc874178fd56756` |
| `testdata/fixtures/query/price-weight.bean` | `d06800bb7fb7c4f1a569e932f5cd3bea99280702966c70c588178d53914b9cb6` |
| `testdata/fixtures/query/price-weight.queries.json` | `2217120340702667f5d0a0e842633053d626a17ed1d4da0a8024734a0dc19e65` |
| `testdata/golden/query-v1/price-weight.json` | `17583291a5b4bf5738391078d7c50e15b4be8e3aa94506a5a99a0cf5da29ba76` |
| `testdata/fixtures/query/aggregates.bean` | `cfc8210832a1aa1b59e46213225a8b8680b70df84707be4b8ef836aea9240915` |
| `testdata/fixtures/query/aggregates.queries.json` | `b2d6d8a38521addf4c1a653afda00ef59320b7f68214c31da571abd5aee76c5e` |
| `testdata/golden/query-v1/aggregates.json` | `3fd5ef4fb252c719e438b041075358f0eaf021f6f3f72317d24105e5f1ef6c4a` |
| `testdata/fixtures/query/casts.queries.json` | `53ee09a0fcaa69b3443df5c11e297305b37eb4b0fc773374749d1a3b73046335` |
| `testdata/golden/query-v1/casts.json` | `c4a7a47fa070efc5db19e0afa1a7dc2161f42b143f8f57ba1f3e361fb4314904` |

## 文档检查

本批八份设计/计划/实施/导航/进度/矩阵/ADR/证据文档检查exit0：100个本地链接/锚点、28个成对代码块、15个shell块的`sh -n`、24个唯一任务及依赖、23项当前公开SHA256均通过；尾空白/末尾换行通过。仅追加本检查记录，未再修改产品源码或个人输入。
