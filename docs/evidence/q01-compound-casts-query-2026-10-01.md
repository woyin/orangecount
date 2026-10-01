# Q01复合转换、金额表示与库存计算

日期：2026-10-01。从实施38节继续；上批[标量cast证据](q01-casts-query-2026-10-01.md)保留历史。当前产品源码经过以下实际验证，完整24任务仍未完成。未改个人账本、创建提交、启动GUI或发布。

## 本批行为

- str(amount/position/inventory)已支持；完整成本/date/label、空/零成本、空库存、固定Position排序与原始标签文本分别保持。格式结果与typed财务cell分开，标签中的双引号按固定str规则显示，不把其显示文本冒充可重新解析的账本源码。
- number(amount)保留Decimal表示；数量/单位成本精度、成本乘法、weight有成本分支、units/cost(inventory)及inventory累加使用已验规则。Text字段不进入财务JSON、lotKey或规范group身份。
- 混合普通/聚合投影在未显式GROUP BY时自动按普通投影分组，避免只取首行、把全部库存混在一起；legacy入口分组不变。完整group/alias合法性仍未全部验收。
- ledger Number.DecimalText及Cost.DecimalText保存归约表示；Decimal私有表示跟随精确运算，只有能精确表示原值时保留。Raw、Rat、比较、JSON财务身份和容差输入不改变；完整/拆分FIFO与可精确表示的插值、总成本转单价已有公开例子。
- typed有限Decimal成本乘法/库存累加应用固定28位上下文；精确非终止AVERAGE成本保留有理数扩展，不把它当量子0的整数舍入。账户余额、成本基准和AVERAGE数学仍精确。

本批仍未交付：无成本报价weight的完整源表示、全部插值/总成本高精度边界、极端指数/上下文陷阱、其余日期/聚合重载与group/alias、balance/sum(inventory)、预设、loader长尾；Q02/X01/X02及GUI/安装/远端/目标机器/签名公证。源码支持某函数不等于全部重载通过。Decimal快照深复制的表示保留及深表达式成本也需继续按相关接口验收，不能由本批推断所有表示路径已完整。

## 独立期望与实际失败

| 切片 | 原实际失败 | 修复/覆盖 |
| --- | --- | --- |
| casts复合str | execution-error，之后发现自动分组typed-result-mismatch | 正负USD分别成组，库存2/-2，不错误相抵成空 |
| 完整成本与空库存 | 新公开32组compound-values均匹配 | 成本/date/label、金额str、固定库存顺序/归约/空值与普通自动分组 |
| precision-acquisition | execution-error | 2.00 STK、10.00 USD及20.0000 USD；amount提取精度保留 |
| precision-reduction-and-inferred-leg | typed-result-mismatch | -1.00 STK、-10.0000 USD与可表示插值-2.00 USD |
| precision-inventory-addition | typed-result-mismatch | 余仓1.00 STK及10.0000 USD，规范数学键不改变 |
| split-lot-quantity-precision | typed-result-mismatch | 两个已归约腿分别-2.0/-1.00，完整日期/成本精度 |
| 高精度cost/weight、总成本单位精度 | 补充独立参考覆盖 | source position保持原数量，查询cost/weight按28位；1.00总成本/2单位得到0.50 |

最初混合复合投影还触发了错误使用不存在helper的编译失败，立即改为检查实际select表达式；此编译失败不作为oracle产品red。总成本表示的有效性保护来自代码检查，补充公开覆盖后通过，不宣称该覆盖先red。

新增compound-casts固定公开语料，含include源图，共6场景；compound-values增加4场景至32；casts增加1场景至26。当前十组共127/127通过：22+32+2+1+4+3+3+28+26+6。其他通过golden保留，只有相关三组显式regen。

## 当前实际检查

| 命令/操作 | 结果及范围 |
| --- | --- |
| query/compat/CLI及ledger/snapshot targeted | exit0，先验证各实际red及升级影响 |
| `go test -count=1 ./...` | exit0，当前产品源码全量 |
| 七包race：query/compat/ledger/report/snapshot/desktop/bridge | exit0 |
| `make fmt vet license parity-full query-parity query-oracle-check` | exit0；50/50普通会计，AVERAGE独立数学；127/127 query及固定oracle重算 |
| `make mac-test`后`make app` | 顺序exit0，ContractTests/StateTests/NativeClientTests真实执行；arm64开发包构建验证 |
| 精度故障注入 | 临时gold把number(units(position))的期望2.00改2；queryparity真实exit1、Checked6/Failed1、typed-result-mismatch；仓库期望未改 |
| 重编PrivateReadProbe、链接本批archive后私人同源differential | exit0，same_input_capture/native_checked/native_contract均true |

私人仅保留聚合：47文件、双方entries2831/errors0/valid=true；missing/extra entries及changed inventory accounts均0；Expenses73行/Income22行一致；三个holding cutoff及48项价格可用性检查。未记录个人内容、账户/金额/日期明细、路径列表、哈希或截图。没有新增typed查询私人对照，当前输入不能验收AVERAGE与所有报价场景。

没有本批GUI、性能基准、独立安装、远端run、目标系统/另一机器或签名公证。原生自动化不能替代查询工作台或全页GUI验收。旧性能/原生证据只在对应输入/源码仍适用时继承。

## 后续入口及公开身份

当前继续实施39节：先补尚缺的报价表示和相关数值/复制路径，再补其余Q01最小签名、group/alias、balance/预设；稳定typed形状可继续Q02协议与工作台，X01/X02、N04/R01/D01/R02/R03仍保持完整范围。

| 公开文件 | SHA256 |
| --- | --- |
| `internal/ledger/ast.go` | `4ca2b8a2993a30725e5e876c6b1a261eb24d1d20847e0e5e4497d119d78aac6b` |
| `internal/ledger/decimal.go` | `7067ab225a2c4470a18f9e366fa0716532717133eaa04f085818cc546080c294` |
| `internal/ledger/evaluator.go` | `aa1805d72ebbcc5f28d76a80135449de58b54768b950c374df9c4df47681abfb` |
| `internal/ledger/tolerance.go` | `f63fe33342cb416d9e0af12511129fb491d8725b3e84c99763a4e7924882a869` |
| `internal/query/query.go` | `1a375a02b6d4d14511c602c75241a8a611fccc16188c184f65914dfa6f298e9d` |
| `internal/query/typed.go` | `1fd75a41c1eea1b8664a0ced3b6fe0e3f1c9d4592fde6a60bb4db8b98db8a1d6` |
| `internal/query/coercion.go` | `1eb7020e82a91ccb98577912cd09fa76cbbb9bd77f58a7834cbf626e9e5af965` |
| `internal/query/compound.go` | `33368eec05b5f49cd531712eb52327a8920881a9042f1134996646f1de97480f` |
| `internal/query/decimal_text.go` | `a11bea6d7a867cee8e93ff1b3aeb27313a24bf24ef29a651fca4455e821482f7` |
| `internal/compat/query.go` | `cd44fdd5ed4586bcc9a32b10dbeb1bc73f5fd8f35a1b600343a70991005afc39` |
| `internal/compat/query_test.go` | `54580f7b223eeabc2c3eb593c11fb5244f6395d0d26913757d762404bbf183c0` |
| `tools/reference/generate_query_golden.py` | `5456bd5f15e4ce1e96cfbdaba408c52e14ce375c9568004ff27c2b85086bdba3` |
| `tools/reference/uv.lock` | `60acbc24073541a373b032e1e8c9a21949c8b354cbc1f77b6f0553609f8931ec` |
| `testdata/fixtures/query/compound-casts.bean` | `11815badcb4947c23cb59232f51044ca587219b330b56e1f272b6bf371505314` |
| `testdata/fixtures/query/compound-casts.queries.json` | `73c71aac39df3928d0968bd46c93c83a18bc9aa5c1ced5f5202e7fc786dd9c3b` |
| `testdata/golden/query-v1/compound-casts.json` | `255ecbc780b33539624eb4d24b4d325b1f14aeedf670d2192bcad3a13b57a6e0` |
| `testdata/fixtures/query/compound-values.queries.json` | `a3301fff50889aa51641ac5669fa4dec80ccf2afb5b8766c5108140501dc936c` |
| `testdata/golden/query-v1/compound-values.json` | `ba95a83e84dd279487bcceabb9e5966aed3333f375fd9df92c58666e66eb58c3` |
| `testdata/fixtures/query/casts.queries.json` | `0d4d4212f6d83740038b787dc4a1428d2065631ff4da9d475078ce282398b3f8` |
| `testdata/golden/query-v1/casts.json` | `951f128c594db413fb4692d0f12e81f181dd119ccb9357758f8be28e08f02996` |

## 文档交接检查

本批七份主入口/状态/证据文档检查exit0：94个本地链接及存在的锚点、28个成对代码块、15个shell块的`sh -n`、24个唯一任务及引用依赖、20项当前公开SHA256全部通过；尾空白/末尾换行及`git diff --check`通过。已同步当前127计数与实施39节入口，旧116/91等开工记录保留为历史。随后仅追加本检查记录，未修改产品源码或个人账本。
