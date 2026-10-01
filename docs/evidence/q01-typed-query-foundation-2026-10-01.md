# Q01：类型化查询基础与当前证据

本记录保留标量基础批次的输入、源码身份和检查。后续已有[复合值实现](q01-compound-query-2026-10-01.md)，当前源码不再全部匹配本节末尾旧SHA；不得用本记录覆盖最新状态。

日期：2026-10-01。当前dirty工作区为基线，未提交、创建PR或发布。Q01首批实现完成并经当前自动化验证，**完整Q01/Q02及M0–M4仍未完成**。本批无查询Swift DTO/C route/原生页面，不以既有原生测试推断查询GUI通过。

## 当前实现

`query.EvaluateTyped`、`ParseStandard`复用原解析器和执行链，缺省FROM为postings。编译在数据扫描前绑定SELECT/WHERE/GROUP/ORDER；ordered columns和ordinal ID加positional cells保留同名标题。结果contract为 `beanquery-0.2.0-typed-v1`。integer用十进制字符串，decimal用共享PresentedDecimal，NULL不含value，空字符串/false不省略，零行仍有schema。

当前支持account/number/currency/date/year/month/day/narration/flag/filename/lineno等已绑定标量列，abs/length/count/sum、直接用户meta/entry_meta/any_meta及基础分组/alias/排序。系统meta键、有效stack、compound、全部重载、完整NULL逻辑/动态比较仍待Q01后续。当前并不宣称所有已写代码的重载都有独立oracle验收；矩阵逐项记录。

LIMIT 0只编译返回空结果；decimal除零为NULL。日期词法在standard入口识别YYYY-MM-DD，避免被解释为减法；字符串比较/排序不强制转数值。typed排序可传递类型错误，NULL先于非NULL。GROUP使用规范化类型和值，并向上传递求值错误，不再吞错或把字符串2和数值2合并。

旧Evaluate/Parse及CLI/Web保留原适用行为，typed-json为显式新格式；旧map遇到重复标题明确报错，避免覆盖值。首次广泛检查发现旧EOF错误文案被改变，已将新的EOF检查限制在standard解析模式，旧错误回归通过。

pad生成posting的有效filename/lineno在固定oracle继承balance断言。本例来源第4行，而导航At仍为pad第3行。新增值类型PostingSourceMetadata保存该区别，避免可变指针别名；普通源posting使用自己的原始行。不能将所有生成posting统称NULL来源。

`internal/presentation`共用既有六位非终止显示策略；report公共PresentedDecimal/FormatDecimal接口保留，query不引入对report的循环依赖。精确值不改变，不使用Double处理财务数值。

## 当前实际验证

| 检查 | 结果与范围 |
| --- | --- |
| `go test -count=1 ./...` | 最后日期/字符串排序修正后全量通过，包含CLI/Web/import相关包 |
| query/compat/ledger/report/snapshot/desktop/bridge race | 当前7包 `-race -count=1` 通过 |
| `make fmt vet license parity-full` | 通过，schema2的46/46；AVERAGE/global booking另验；无运行时Go依赖 |
| `make query-parity` | 22/22，成功值、类型、位置、重数、排序和预期错误；离线只读golden |
| `make query-oracle-check` | 22场景由固定3.2.3/0.2.0重新生成后对照一致，不覆写golden |
| 故障注入 | 外部临时gold修改一格为123456789；真实 `tools/queryparity -golden` exit1、failed=1、typed-result-mismatch |
| 检测器负例 | 篡改值被检出；尾随第二JSON文档由red到拒绝；输入/manifest身份与完整case列表校验 |
| 公共补充来源/分组/排序 | pad第4行与oracle一致；string2/decimal2各计数1；叙述10、10、2、2按字符串排序；均有Go red→green |
| CLI | typed-json真实输出保留重复标题、30位整数在gate验收、空字符串/false、contract；旧CLI格式回归通过 |
| `make mac-test` | 真实Contract、受控State、真实NativeClient三套实际执行，覆盖现有账户/持仓/价格等页面 |
| `make app` | arm64、target13.0、Info.plist及ad-hoc签名检查通过；未启动应用或进行GUI验收 |

补充公共命令：

```sh
go test -count=1 ./internal/query -run 'TestTypedQuery|TestLegacyQueryRejectsDuplicate'
go test -count=1 ./internal/compat -run TestQueryOracle
go test -race -count=1 ./internal/query ./internal/compat ./internal/ledger \
  ./internal/report ./internal/snapshot ./internal/desktop ./internal/bridge
go run ./cmd/orangecount query --format typed-json --semantics v3 \
  testdata/fixtures/query/default-source.bean \
  'SELECT account, sum(number) AS total GROUP BY account ORDER BY total DESC'
```

核心私人只读对照已通过：47文件、双方2831指令/0错误/valid=true、完整分录missing/extra=0、inventory差异账户0；Expenses73行/Income22行一致，same_input_capture=true、profile=v3、schema2、native_checked=false。使用外部入口环境变量及offline differential脚本；不保存私人源内容或明细。

随后复用本批mac-test生成的最新C archive，重新编译PrivateReadProbe，不重复整套mac-test。带该探针执行新的同源私人对照，仍为上述47/2831及零差异汇总，native_checked/native_contract=true、native_holding_cutoffs=3、native_price_checks=48。该探针验证现有根汇总、持仓与价格可用性，不执行新的typed query，也不验GUI。前批已记录私人输入无源price，有价数学继续由公开价格语料证明；不能将48项称为48个真实报价。

原有GUI、客户端性能与支持矩阵为[前批A03证据](a03-native-prices-2026-10-01.md)，本批没有新增查询UI、CPU大输入预算或GUI性能证据。实际旧系统/另一机器、远端run和签名公证仍缺。

## 后续实施

先完成position/amount/inventory及units/cost/weight列/number/currency/sum重载，再核对stack/include词法传播、系统meta键、NULL/动态类型和预设。所有新签名继续扩展独立oracle，不能直接加宽豁免。Q02再接desktop/bridge/Swift/Store/工作台、上下文和预算；X01/X02及GUI/安装/分发保持完整原范围。

当前工具只以public query fixture为输入，无私人账本参数。生成器是明确更新工具，gate和oracle-check只读；source_columns仅按manifest声明将真实来源规范化为公共语料根相对路径，不删除普通字符串内容。

配套：[查询矩阵](../query-capability-matrix.md)、[进度](../go-beancount-native-progress.md)、[实施手册](../go-beancount-native-implementation-2026-09-30.md)。

## 当前公开源码/输入身份

HEAD仍为 `20a9a6e86a5b4e2d6bb51aa2d2ae19dcd2934ddd` 加实际工作区，以下不含私人输入。

| 文件 | SHA256 |
| --- | --- |
| `internal/query/query.go` | `17825ce99540e8414fed3cf7324096152ab7256d3774d3774f74c7bdf6eec0c8` |
| `internal/query/typed.go` | `ff4505e0a7da0d49049f9a6383713f6eb22c22e5d9a26dc57661ec854ce7d30e` |
| `internal/query/typed_test.go` | `553befb5d218d69f6e7678451c73e31d545f868f4a731a4c3d002370dc7d0df0` |
| `internal/compat/query.go` | `431bbc851ed88009166fdfe5c651831aa201efed3c2fb07bc6d9cddc80c7cbaa` |
| `cmd/orangecount/query.go` | `7308c14ef722162e854fd2a2356877272d81c7502e8fe759794d1b5a106e0dd6` |
| `internal/ledger/ast.go` | `0e03924e37c19022429c87f9171180bdd48434e5065f1eb7c8a728a210be991e` |
| `internal/ledger/evaluator.go` | `0799ad3f0e64a592165827752d18f83645b688e3d4812f776aee58f26dcacd1e` |
| `internal/presentation/decimal.go` | `aa79039c5b9bf161d6bd38e17a68b7cf7b1f19706d98344d78c7f5dec88ba8be` |
| `tools/reference/generate_query_golden.py` | `79343c0d0a50e3b183258a4a6c996dfbd1e3acd1cd85d9945deba28991fcd920` |
| `testdata/fixtures/query/default-source.bean` | `36be0578f294d4bd6769a479231d28b6fe532b518dc96c01cf6cbe9660ef515d` |
| `testdata/fixtures/query/default-source.queries.json` | `2a52c102b66c6f2153c6b6bbbada2640126416b8a58f535eddd6a91b73723f3d` |
| `testdata/golden/query-v1/default-source.json` | `c5a6d9bd67d43abbbbedfb1fd2e375c9a921276469efc0a09fefe788c77e7109` |
