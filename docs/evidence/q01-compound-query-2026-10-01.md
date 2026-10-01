# Q01：复合查询值、库存与报价 weight

本记录保留复合值批次的历史源码身份及53场景结果；随后metadata批次继续推进，当前63场景及完整源图身份见[最新证据](q01-metadata-query-2026-10-01.md)。不要用下列旧哈希校验当前文件。
日期：2026-10-01。基线为实际dirty工作区，HEAD仍为`20a9a6e86a5b4e2d6bb51aa2d2ae19dcd2934ddd`。本批继续完整24任务目标，未提交、创建PR、运行远端CI或发布。Q01有所推进，完整Q01/Q02/M0–M4仍未完成。

## 实际实现

新增`internal/query/compound.go`。Position从已经booked的posting构造，包含Amount units与nullable完整成本；无成本、零成本、成本日期、缺失/空label分别保留。金额仍用精确Decimal，来源Span不进入财务值、分组或库存键。

typed入口增加`position`、Amount类型的`weight`、`units(position)`、`cost(position)`、`number(amount)`、`currency(amount)`、`sum(position)`、`sum(amount)`、`units(inventory)`和`cost(inventory)`。用户Amount metadata可由meta/entry_meta/any_meta投影。Inventory合并相同完整键，保持多币种，消除相抵零批次；空库存编码为`[]`，不是NULL。原CLI/Web的json/csv继续使用旧入口。

Amount/Position/Inventory的ORDER BY有独立oracle场景。库存内部保留插入次序，用于固定参考版本的稳定Position排序；分组身份和输出规范化另外按完整财务键排序。特殊反例确认：直接用规范化输出顺序替代内部次序，会改变两个库存的ORDER BY结果，已修复。固定oracle不接受这些复合类型的直接比较运算，typed编译阶段明确type-mismatch，不能经fmt字符串比较返回伪结果。

价格权重另外遵循已归一单价：新增`ledger.BookedPriceWeight`只用于typed无成本报价weight，按28位有效数字half-even计算。公开3单位`@@ 10`权重为`9.999999999999999999999999999`；负向3单位`@@ 11`为`-11`。源总价、原PostingWeight、余额、成本和AVERAGE保持原精确表示。这是具体报价字段契约，绝非对所有差分做28位舍入。高精度成本乘法、一般算术、SUM的参考Decimal上下文与扩展精确有理数关系仍需后续独立确定，不能据这两个报价案例宣称全部数值兼容。

## Oracle与失败信号

固定Beancount3.2.3/BeanQuery0.2.0，生成器只接受注册public corpus名称；无私人账本输入参数。正常gate和oracle-check只读，生成另由显式开发操作执行并审阅。

| 公开语料 | 场景数 | 当前范围 |
| --- | --- | --- |
| default-source | 22 | 继承标量基础，当前重新检查 |
| compound-values | 28 | 完整Position、Amount重载、库存、分组/排序、零行、错误和金额meta |
| price-weight | 2 | 正/负非终止单价与标准weight/number/currency |
| inventory-order | 1 | 内部插入顺序与规范化输出排序不能混用 |
| 合计 | 53 | 值、类型、列位置、行重数/顺序和预期错误全部比较 |

实际red→green包括未知Position列、未注册复合值normalization、units/cost/number/currency/weight缺失、库存聚合/归约、Amount metadata、复合类型错误比较、排序、非终止报价weight，以及库存排序次序反例。缺失已注册golden也会失败。测试中一次使用os.IsNotExist检查wrapped error不正确，已改errors.Is；这属于测试断言修正，不记为产品缺陷。

成本number/currency/date/label逐字段篡改的detector测试均检出；真实CLI把外部临时gold的首个成本日期改为1900-01-01，exit1、checked28/failed1、position-full-cost/typed-result-mismatch。没有修改入库gold来执行故障注入。

## 当前实际验证

- `go test -count=1 ./...`：全量通过，包括旧CLI、Web、import/authoring等调用方。
- `go test -race -count=1 ./internal/query ./internal/compat ./internal/ledger ./internal/report ./internal/snapshot ./internal/desktop ./internal/bridge`：七包全部通过。
- `make fmt vet license parity-full query-parity query-oracle-check`：通过；46/46会计schema2、独立AVERAGE/global booking、53/53查询及固定oracle只读复验，无第三方运行时Go依赖。
- `make mac-test`：真实Contract、受控State、真实NativeClient三套均实际执行并通过；覆盖既有账户/持仓/价格，不覆盖查询新DTO。
- `make app`：当前arm64/deployment13.0/Info.plist/ad-hoc构建通过，没有启动GUI或进行安装验收。
- 真实CLI的公开`units(sum(position))`/`cost(sum(position))`得到15 STK/110 USD，保留inventory类型、完整schema和精确数值。

本批mac-test完成后，复用其最新C archive重编译PrivateReadProbe，再执行offline differential同源检查；没有重复整套mac-test。47个文件，双方2831指令、零错误/valid=true、missing/extra entries与changed inventory accounts均0；Expenses73行/Income22行一致，same_input_capture/native_checked/native_contract=true，3个持仓截至日、48项价格可用性检查。未执行新typed query，未检查查询GUI。私人无AVERAGE及源price，48项不能称为48条真实报价；扩展与有价数学由公开语料独立证明。只保存汇总，没有写入或改动私人源图。

## 剩余与交接

下一批核对系统来源metadata、显式NULL优先级及有效pushmeta/popmeta/include传播，之后补完整NULL/动态类型、unary/日期/聚合签名、balance/sum(inventory)、精度与可执行预设。当前所有已写代码组合不等于全部oracle已验签名。

没有查询C route、Swift DTO或QueryView，Q02的快照上下文、取消/队列、文本/行/字节预算与GUI均未交付。没有本批查询性能数字，不借旧原生客户端基准证明新查询的分组/排序成本。X01/X02、N04/R01和R02/R03保持原范围。

设计、计划和实施手册以本记录及当前能力矩阵为新交接基线；[标量基础证据](q01-typed-query-foundation-2026-10-01.md)的旧哈希保留历史身份，不替换为本批源码。

## 本批公开源码与输入身份

仅包含公开文件；不记录私人源图哈希或明细。

| 文件 | SHA256 |
| --- | --- |
| `internal/query/query.go` | `53f1cb08f9602a0a36d640ad4b86fa158f4a0bad561cc40fc89008f52f6a4c26` |
| `internal/query/typed.go` | `7a0e051a05fdeda8a04b157062ffe555a7d48ebfbe93ba582f4981f8247108c6` |
| `internal/query/compound.go` | `da8228570a17b7485203ce39548080c73ee45c02c8678402aab5fc00a53053a5` |
| `internal/query/compound_test.go` | `e017e1c49483395e804f6002480ea7ef8d2042bdc77dd625501f01247b02171b` |
| `internal/compat/query.go` | `d157c323582b48c6ec571a2a78b3b8bf0ff8637a469449db3c6437c16d2c01b1` |
| `internal/compat/query_test.go` | `d82927f9415df02d5a6e2099279eb0fd577eb4a3e611b4a039d1b068744777f0` |
| `tools/queryparity/main.go` | `3d51e6422bc0aff96a2933d55845996cb3fb412d31b19f8eeb991062cb139616` |
| `tools/reference/generate_query_golden.py` | `61e0710f751304c8dda870d0020cce356ca0815bcf58ef0b986eafe6e635e88e` |
| `internal/ledger/price_normalization.go` | `6e26757ca9a2528295aa70795b3c0e456b2ffed6c0e7872729954a34c4c21d98` |
| `testdata/fixtures/query/compound-values.bean` | `4681f4762bc2fe7cc64229491bacfc553624e6e4aa132bc4e6fde9538d718b50` |
| `testdata/fixtures/query/compound-values.queries.json` | `4d0adae0bec02c01134d65f598014c3573bdd7ef83712d41219a38b9627802b7` |
| `testdata/golden/query-v1/compound-values.json` | `9b21c6e28a94c91c33386ef4ffa420eee1ba2d17c309c66f8fc8cd7c855e9043` |
| `testdata/fixtures/query/price-weight.bean` | `4cbb3d5456ff955addbc932078ac5b12f234c13dcf7e3464fc66579679ef6128` |
| `testdata/fixtures/query/price-weight.queries.json` | `db140b65270c610abb3a47286e7b42efcb2616af6b3fa604f44fa72c7acc5c77` |
| `testdata/golden/query-v1/price-weight.json` | `128fecb7fb875c82fa5dd09d4b917df7904f6794a63a6b33129c23198d99a3d3` |
| `testdata/fixtures/query/inventory-order.bean` | `ff88d0ddda8ad4d3e42172032cff5cb3811a4012544a0e5c22d7fe3f2b18117d` |
| `testdata/fixtures/query/inventory-order.queries.json` | `89874abfea1741de7bf350b3aedf809104ea9c41e06d4b6902eeb0341b6f3b13` |
| `testdata/golden/query-v1/inventory-order.json` | `b22151d94bd9c1475c8a0b07ff5c0a4f9d62b1ba92ce705fadb6c6daa34f3d22` |
| `testdata/golden/query-v1/default-source.json` | `c5a6d9bd67d43abbbbedfb1fd2e375c9a921276469efc0a09fefe788c77e7109` |
