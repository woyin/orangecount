# Q01显式转换与标量Decimal上下文

日期：2026-10-01。以实施手册37节的当前失败继续，前次[文档交接](go-native-development-handoff-2026-10-01.md)保留原red和源码身份。本次实施了产品源码，未改个人账本、启动GUI、创建提交或发布；完整24任务仍未完成。

## 已实现及明确边界

- str/decimal/date/bool/int的已验标量与metadata重载；失败有限转换与NULL按固定规则处理。三参date接受integer/bool；NULL literal签名与nullable integer运行值分开，C整数越界明确unsupported。
- 统一隐式/显式Decimal转换表示，算术结果保留quantum与负零；`str(decimal(1.00+2))`返回3.00，date(2000,TRUE,1)返回2000-01-01。
- typed标量Decimal加/减/乘/除、unary minus、abs及SUM使用固定默认28位有效数字half-even上下文；SUM逐项按源顺序舍入，NULL不改累加器。构造Decimal及直接源值不经过这类运算舍入；Integer仍精确、不受28位限制。
- 精确ledger余额、成本与AVERAGE没有统一舍入，query上下文局限在typed查询Implementation；legacy Parse/Evaluate不变。query结果contract v1、原生协议2、默认legacy与v3-core-2均不变。
- 复合str、number(amount)的源表示、cost/库存运算与聚合上下文尚未完整实现；参考Decimal极端指数/上下文陷阱、非有限值、一般查询签名/分组alias、balance与sum(inventory)、预设和loader长尾仍缺。不能据本批宣布完整Q01。
- Q02没有路由/Swift查询页、查询预算与真实取消；X01/X02及GUI/安装/发行仍未完成。分离value/text的求值路径还需在Q02资源验证中检查深表达式计算成本，不用LIMIT冒充运行预算。

## Red→green及独立期望

| 切片 | 实际失败 | 修复后信号 |
| --- | --- | --- |
| 原注册算术str | Checked10/Failed1、arithmetic-decimal-string execution-error | 10场景green；3.00、1001、-0.00、11正确 |
| 组合Decimal与date bool | 前交接CLI组合返回3、日期拒绝；补12场景后date bool真实execution-error | 12场景green；布尔各参数位置与非法日期NULL匹配 |
| 算术上下文 | 第13场景typed-result-mismatch | half-even两种tie、carry、乘法与机器Decimal同时匹配 |
| 除法上下文 | 第14场景execution-error | 终止商保留quantum，非终止商28位、组合及除零匹配 |
| unary/abs | 第15场景execution-error | 上下文舍入、零符号及源精度匹配 |
| SUM上下文 | 第16场景execution-error | 逐项舍入、NULL、负零和精度匹配 |

另补八个正负签名/边界场景，共24场景；最终补SUM源序案例为25场景。该例通过公开账户/摘要regex构造四个查询值1E28、1、-1E28、0，固定oracle总和为0；精确累加最后才舍入会得到1。此例验证逐步舍入语义，不以实现重算期望。输入只有独立公开fixture。

当前九个corpus共116/116通过：default-source22、compound-values28、price-weight2、inventory-order1、metadata-source4、metadata-stack3、metadata-pad3、dynamic-values28、casts25。normal gate只读，只有casts显式regen，其他八组golden保留。

## 本次实际验证

| 检查 | 结果及适用范围 |
| --- | --- |
| `go test -count=1 ./...` | exit0；当前产品源码全包通过；执行时casts24场景 |
| 七包race：query/compat/ledger/report/snapshot/desktop/bridge | exit0；同一当前产品源码 |
| `make fmt vet license parity-full query-parity query-oracle-check` | exit0；50/50普通会计、独立AVERAGE；当时115 query |
| 最后补SUM顺序案例后query/compat/CLI及两个query gate | exit0；116/116及oracle重算核对通过；源码未再变，不重复无关原生/race |
| `make mac-test`后`make app`顺序执行 | exit0；ContractTests/StateTests/NativeClientTests真实执行，arm64开发包构建/验证 |
| CLI四列表达式 | exit0；3.00、2000-01-01、28位1/3、-0.00正确 |
| 外部临时golden故障注入 | 把组合Decimal期望3.00改3；真实queryparity exit1、Checked25/Failed1、typed-result-mismatch；临时文件自动清理，仓库期望不变 |
| 当前私人同源、重编PrivateReadProbe并链接本批archive | exit0；same_input_capture/native_checked/native_contract均true |

没有新GUI、性能或原生查询验收。客户端/应用门槛不能替代这些检查；过去截图失败也不能证明当前工具不可用。远端CI、旧系统/另一机器、公开签名及公证未执行。

私人摘要仅保留聚合：47文件；双方entries2831/errors0/valid=true；missing/extra entries与changed inventory accounts均0；Expenses73行/Income22行一致；native_holding_cutoffs=3、native_price_checks=48。没有保存私人源内容、账户、金额、日期明细、路径列表、哈希或截图。没有新的typed查询私人对照；48项可用性检查不代表48条源报价，AVERAGE与有价估值继续用公开输入。

私人实际使用与native构建相同工具链，重编三个公开Swift文件，再给differential传外部入口及探针环境变量；用户账本只读。以下为不含私人路径的复现形式：

```sh
set -eu
. apps/macos/toolchain.sh
cast_probe_dir="$PWD/build/macos-tests"
"$OC_SWIFT" -O -parse-as-library -swift-version 6 -target "$OC_TARGET" \
  -sdk "$SDKROOT" -I "$cast_probe_dir" -L "$cast_probe_dir" -lorangecount \
  apps/macos/src/Models.swift apps/macos/src/BridgeClient.swift \
  apps/macos/tests/PrivateReadProbe.swift -o "$cast_probe_dir/PrivateReadProbe"
# 运行前显式设置已授权的ORANGECOUNT_PRIVATE_LEDGER外部入口。
ORANGECOUNT_NATIVE_PROBE="$cast_probe_dir/PrivateReadProbe" UV_OFFLINE=1 \
  sh tools/reference/differential.sh
```

## 下一入口及公开身份

设计30节记录本批范围，计划18节包1已推进；当前下一入口是实施38节的复合转换/上下文与剩余签名，随后Q02/X01/X02，所有原生与发行门槛保留。旧red和旧SHA保持历史，不改写为当前成功。

| 公开文件 | SHA256 |
| --- | --- |
| `internal/query/query.go` | `441cd76aa5a0eec869b4b0ccc07774880d87ebc085823517fcb806306b5e999d` |
| `internal/query/typed.go` | `df915dc88ab21debf5fa2d6a77777476ac70cfe5aa0e7e58d0cf14635c4d1de2` |
| `internal/query/coercion.go` | `2d7811c4b8c835396a42136274b704cef14904f52f21187d50496c7558c02708` |
| `internal/query/casts.go` | `f46ab690ae2bd7026f77b93181e64273b8b4a062ace3cfdb5ef367924d33b1f9` |
| `internal/query/decimal_text.go` | `a11bea6d7a867cee8e93ff1b3aeb27313a24bf24ef29a651fca4455e821482f7` |
| `internal/query/compound.go` | `da8228570a17b7485203ce39548080c73ee45c02c8678402aab5fc00a53053a5` |
| `internal/compat/query.go` | `95bfb5586a8f09ce5674aab11ff60f38990037c36420ce96ced14e437f08b108` |
| `internal/compat/query_test.go` | `5dd6ed7442eb9702c8aa2efe03643274070f5b1b9444c674c7adac725e6f9971` |
| `tools/reference/generate_query_golden.py` | `09b14c1b54731ab30b30fc3cfd9b29cdfa63ca8801ae9d2dd54e926dce9edca2` |
| `tools/reference/uv.lock` | `60acbc24073541a373b032e1e8c9a21949c8b354cbc1f77b6f0553609f8931ec` |
| `testdata/fixtures/query/casts.bean` | `c0e689ac4a2b72052306673849d7dfdc2283c8766e4351fc446f2c939d6a38f0` |
| `testdata/fixtures/query/casts.queries.json` | `c836d3391e266901883da0e23af168e8329975bf715454f74d8072f9f42b1194` |
| `testdata/fixtures/query/dynamic-values.bean` | `a7136be9a2979dc6f9c31a1baaf8a6c59b9817260359b1f931f74828eb323f8b` |
| `testdata/golden/query-v1/casts.json` | `f179a463bbbc058837bb60a63a3403106b5f8112e58144b806ce4a8234ac2ae8` |
| `apps/macos/tests/PrivateReadProbe.swift` | `3eecf7160a1a238539da82aa823975ddb3d22fc99a667cd2e34ac1af6e351ab6` |

## 文档一致性检查

本批实际检查7个文档、91个本地链接/锚点、16段shell语法、24个唯一顶层任务ID及依赖、15项公开源码/输入SHA256，全部通过；代码块、换行和空白检查通过，`git diff --check`退出0。旧证据身份保留历史，本批身份单列。实施38节是当前入口，不再按37节的旧101/red重复开发。
