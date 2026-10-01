# Q01 动态类型、NULL与逻辑求值证据

日期：2026-10-01。HEAD `20a9a6e86a5b4e2d6bb51aa2d2ae19dcd2934ddd`，保留现有dirty实现继续开发，无提交/PR/发布。[元数据证据](q01-metadata-query-2026-10-01.md)及更早记录保留历史身份；本记录给出当前源码与输入。

## 实现及独立依据

固定本地Beancount3.2.3/BeanQuery0.2.0开发oracle，独立实现Go，不复制/链接/打包参考执行代码。先通过公开probe确认实际行为，再按EvaluateTyped/compat/CLI已授权接口观察red/green。

- 两个object比较在扫描前拒绝，一侧有string/Decimal/date静态类型时绑定另一侧转换；Integer目标提升Decimal。无效有限数字/日期转换、缺键和显式NULL返回NULL。
- metadataDecimal/metadataAmount同时保存精确值与转换文本；10.00/2.00 USD文本精度保留，数值/group身份按规范值。系统Integer独立保留，source Number/AST不变。有限字符串支持Unicode十进制数字、指数/下划线；逗号/分数文本无效。
- NULL/TRUE/FALSE、IS [NOT] NULL、±、NOT及括号标题/优先级实现于标准入口。旧Parse/Evaluate及CLI旧json/csv保持旧适用行为。
- AND/OR接受已支持值的真值、按左右顺序短路；完整编译先于求值。NULL AND FALSE返回NULL，FALSE AND NULL返回false；NOT NULL返回true。WHERE使用同一真值，零Amount不选中。
- fixed reference的Inventory隐式真值抛NotImplementedError，Go明确unsupported-capability。混合str/Decimal object排序参考抛TypeError，Go报type-mismatch。bool/NULL直接等式与object负号按固定签名拒绝。
- 探针确认abs(int)无固定重载，修复当前typed误接受，LIMIT0也编译拒绝；abs(decimal)原有回归保留。

query结果contract仍beanquery-0.2.0-typed-v1；oracle golden schema2，8组注册语料，完整源图身份仍必需。会计默认legacy，v3-core-2及legacy-1不变；本轮没有改ledger/snapshot规则。

NaN/Infinity在参考转换中可产生非有限Decimal，但exact有理数无法表示。接口负例先观察错误地成为NULL，再改为unsupported-capability；这是已登记的明确能力边界，未称非有限Decimal结果等价。系统map等未表示对象继续明确拒绝。显式str/decimal/date函数仍unknown-function，本批隐式转换不等于对应函数已实现。

## 实际失败到通过

| 切片 | 首次观察 | 实现后 |
| --- | --- | --- |
| object/object比较 | 错误类别失配，旧coercion接受 | 编译type-mismatch |
| 静态目标转换 | string/decimal/date案例执行失败 | 静态签名转换后列/值一致 |
| unary/NOT | unsupported-expression及NOT绑定错误 | 负号、Any真值与比较优先级一致 |
| NULL/布尔literal | generator未注册NULL声明类型；Go未知列 | normalization注册后实际Go red，literal/schema一致 |
| AND/OR | 类型拒绝、NULL传播/短路错误 | 顺序规则和不读取被跳过系统对象通过 |
| IS NULL | 语法未支持 | 非nullable bool及缺键/NULL/空区分通过 |
| WHERE Amount | 零Amount仍被选中 | 同一真值过滤后精确行数一致 |
| 括号/+ | 语法失败、结果标题丢括号 | 标准parser保留括号，标题/优先级一致 |
| Amount文本 | 2.00/-0.00源精度行为失配 | 包装转换文本；按参考解析零为正零 |
| Inventory真值 | 空库存被自造为false | 实际求值明确拒绝 |
| Unicode Decimal | 合法Unicode字符串错误成为NULL | Nd规范化后精确10.5等值通过 |
| 非有限Decimal | NaN/Infinity静默NULL | 精确契约明确unsupported |
| abs(int) | 固定版不支持而Go接受 | 扫描前拒绝，LIMIT0反例通过 |

其余分组/系统Integer/非法函数等用例是在本轮增加覆盖，首次就通过，不称新修复。扩展public fixture时明确重新捕获输入及期望：加入精度Amount、Unicode/指数/非法数字文本、不同精度同值group，不为适应Go随意修改预期。

## 当前实际验证

- `go test -count=1 ./internal/query ./internal/compat ./cmd/orangecount`及针对性dynamic gate：通过。
- `go test -count=1 ./...`：所有Go包通过；abs补充后重新全量通过。
- query/compat/report/snapshot/desktop/bridge六包race通过；最终abs补充后query/compat再跑race通过。后四包未受最后abs签名代码路径影响，不冒称重新执行它们。
- `make fmt vet license parity-full query-parity query-oracle-check`：最终通过。50/50普通会计、单列AVERAGE具名数学测试；91/91查询，8组固定oracle只读复验，无新增第三方Go运行时依赖。
- `make mac-test`实际Contract/State/NativeClient三套及`make app`通过；abs补充后再次执行，最终C archive/app包含当前源码。arm64、deployment13.0、Info.plist与ad-hoc开发包，不是旧系统或公开分发验收。
- 真实CLI标准查询在公开输入返回单行text、exact11及IS NULL true，WHERE排除零Amount；不是只调用内部函数。
- 外部临时dynamic gold将ordered-null-boolean-logic第一格NULL改false，真实工具 `-golden` exit1、checked28/failed1、该case typed-result-mismatch；未改入库gold故障注入。输入/manifest/sourcegraph验证继续保留。

| Corpus | 场景数 |
| --- | --- |
| default-source | 22 |
| compound-values | 28 |
| price-weight | 2 |
| inventory-order | 1 |
| metadata-source | 4 |
| metadata-stack | 3 |
| metadata-pad | 3 |
| dynamic-values | 28 |
| 合计 | 91 |

## 私人只读范围

最终mac-test完成后复用其最新archive重新编译PrivateReadProbe，再通过offline differential读取已授权外部账本；没有重复整套原生gate作为私人入口前置。结果只记录聚合计数，不保存源数据/路径列表/私人hash。

当前47文件、双方2831指令、0错误/valid=true；完整missing/extra entries及changed inventory accounts均0；Expenses73行/Income22行一致。same_input_capture/native_checked/native_contract=true，3持仓cutoff、48价格可用性检查。没有新的typed query私人验收；私人没有AVERAGE与源price，有价算法及均价扩展由公开输入证明。

## 当前交接及未验范围

设计第29节、实施第37节和计划第17节收敛当前规则；matrix/roadmap/progress及reference README同步91场景。源码/输入身份见下表，早期哈希保留历史。文档检查覆盖10个文件、99个本地链接/锚点、20段shell语法、24个唯一任务及依赖、14项当前公开文件SHA，全部通过；git diff --check通过。

下一具体切片为显式转换函数，再完成声明日期/聚合、balance/sum(inventory)、一般Decimal上下文和可执行预设。loader长尾继续按手册36.3；新查到的显式转换和min/max/first/last参考签名只属于primary probe，尚未实现。未知组合、全NULL/alias/聚合合法性不由91场景泛化。

没有查询C路由/Swift DTO/QueryView、预算/取消/性能或新GUI。Q02/X01/X02、完整N04/R01、R02远端run、R03实机/DeveloperID/公证均保持原目标。旧GUI捕获错误仅历史，未断言当前阻塞；既有用户应用及私人源文件保留。完整开发目标未完成。

## 公开文件身份

| 文件 | SHA256 |
| --- | --- |
| `internal/query/query.go` | `1458a89899695c726a0cc381ab258319230bff248d1f59567ecbaa1416eb7329` |
| `internal/query/typed.go` | `2ec4f0b03cb47ef21ea33bcaeff30b8480e4c03d6b73e04df95042929ec79e40` |
| `internal/query/typed_test.go` | `6ba287eba63b3807702afcff3eec71e85b7de785aa0fdeb12fe1dcbdebd2c417` |
| `internal/query/coercion.go` | `99e28d0459f61cbea30e5ec304784bf9a88fcb8d477607d869917f98bcadbe4b` |
| `internal/query/compound.go` | `da8228570a17b7485203ce39548080c73ee45c02c8678402aab5fc00a53053a5` |
| `internal/compat/query.go` | `ef7fd4af40586952d7b52ba604ce8f3a9b2c9cf6243ebc1fd9a23a81c7d35a54` |
| `internal/compat/query_test.go` | `46ca24decaeea9adeaa6e9ae2000696f493b023c552c9be6718c111d17364c37` |
| `cmd/orangecount/query.go` | `7308c14ef722162e854fd2a2356877272d81c7502e8fe759794d1b5a106e0dd6` |
| `cmd/orangecount/query_typed_test.go` | `2b71d1057cf1edf3361079ab465aee27f6a96c0f4b08d20183d8106d03d42c54` |
| `tools/queryparity/main.go` | `3d51e6422bc0aff96a2933d55845996cb3fb412d31b19f8eeb991062cb139616` |
| `tools/reference/generate_query_golden.py` | `ded78fe47c05f0b700d6f6bc1d99e4449b5cecaf0cab871c85380d033164e8a6` |
| `testdata/fixtures/query/dynamic-values.bean` | `a7136be9a2979dc6f9c31a1baaf8a6c59b9817260359b1f931f74828eb323f8b` |
| `testdata/fixtures/query/dynamic-values.queries.json` | `21a32cba18eb2eeefe9693546586ea3c911e721f974dea9f1dbaead3111ab0fb` |
| `testdata/golden/query-v1/dynamic-values.json` | `167515dd31ad9c60536f2b021e4b609e1374d696f1965cdae8af21f1ab814a02` |
