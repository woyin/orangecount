# A03 原生价格、独立对照与客户端证据

日期：2026-10-01。适用当前dirty工作区；A03实现和自动化已有证据，GUI尚未验收，完整M0–M4目标未完成。旧请求校验red保留在[文档交接基线](go-native-documents-handoff-2026-10-01.md)。

## 当前实现

Go从已求值历史建立商品目录，保留已清仓、无报价、仅声明及仅quote使用的商品；源price重数不去重。价格历史只列实际同方向指令，独立selected使用共享report.PriceAt的截至日直接优先/非零倒数规则。date_from只过滤源历史，不影响较早selected；同日最后源价、精确rate、age_days、源方向/位置均保留。

desktop请求拒绝价格页units/market-value及不适用参数，目录/历史分别分页，选中商品在目录预览之外仍完整返回，offset绑定快照，完整响应预算16MiB含C envelope。Swift增加typed价格DTO、required-nullable、page/pair/source方向等一致性解码；Store支持pair/cutoff/Back/分页/reload/关闭及分析偏好，PricesView包含目录、声明/账户使用、实际源曲线与精确列表、选用方向/年龄/来源；持仓报价可进入价格历史。

极大合法报价复现json Infinity，修正共用chartCoordinate为不可绘制时null；随后公开仪表盘回归复现旧flat字段解引用nil的panic，修正为保留完整nullable系列、避免平面图跨缺口。账户/仪表盘/均价的后续有效坐标使用新segment；数学complete_total及exact不因此消失。

## 实际自动化

| 命令 / 检查 | 结果 | 覆盖范围 |
| --- | --- | --- |
| `go test -count=1 ./internal/desktop -run '^TestNativePrices'` | 通过 | 8个具名价格检查：直接/倒数/零/缺价/未来价、目录、重数/include、分页/排序/reload、完整预算、极大值 |
| `TestNativeChartsRetainExactUnplottableHistoryWithoutJoiningAcrossGap` | red panic后green | 原生dashboard/account精确存量完整，坐标缺口与segment、旧flat不跨缺口 |
| `go test -count=1 ./...` | 通过 | 最后共用图表修正后全量Go，含CLI/Web等 |
| `go test -race -count=1 ./internal/ledger ./internal/report ./internal/snapshot ./internal/desktop ./internal/bridge` | 通过 | 当前核心/投影/会话/桥接race |
| `make fmt vet license parity-full` | 通过 | 无运行时Go依赖；schema2的46/46，另验具名AVERAGE/global booking |
| `make mac-test` | 三套实际执行通过 | 真实Contract、受控State、真实NativeClient；新价格契约/导航/pair与cutoff乱序/偏好reopen/关闭/分页与reload |
| `make app` | 通过 | 新PricesView编译、arm64/target13.0元数据和ad-hoc签名校验 |
| 公开独立探针 | 通过，故意篡改后exit1 | 外部临时副本20指令/2文件，69价格场景；人为改gold selected.rate得到native_contract=false |
| 当前私人同源对照 | exit0、native_contract=true | 完整Go/Python及真实Swift/C：47文件/2831指令、3持仓cutoff、48价格可用性场景 |

Swift负向价格载荷检查包含删除commodity/selected/value必填nullable字段、未知reason、已知与null矛盾、负age、错误source方向、未来记录、重复ID和page计数矛盾。公开同日10/11/11均能定位include/原始行，倒数selected不会制造源历史，直接零不是缺价。

私人命令为外部入口与当前已编译PrivateReadProbe环境参数加`sh tools/reference/differential.sh`；不在本记录保存入口路径。summary：双方entries=2831/errors=0/valid=true；missing_entries=0、extra_entries=0、changed_inventory_accounts=0；Expenses73行/Income22行equal=true；same_input_capture=true、schema_version=2、profile=v3、native_checked/native_contract=true、native_holding_cutoffs=3、native_price_checks=48。

此私人输入没有源price指令；新检查验证完整商品使用目录及缺报价状态。有价/倒数/零/未来价数学由公开69场景及具名Go/Swift检查证明，不能把48项私人缺价检查称为48项真实市价等价。Python与Swift临时gold不入库，私人源码/账户/币种/金额/日期/摘要/位置未记录，源文件未修改。

公开探针先暴露现有空Expenses查询rows=null导致TypeError；脚本按空集合处理后公开输入green。篡改测试实际执行新价格比较路径并失败，防止只检账本valid而漏检价格。

## 客户端基准

`make mac-benchmark`在全部构建和探针结束后执行，未启动QA应用；用户原有应用保持运行。环境沿用Mac16,10 / Apple M4 /24GiB、macOS27.0.1(26A434)、Go1.27.1、Swift6.4、SDK27.0。公开生成器的52500交易、4000条同日AAPL/USD报价，和A02相同输入；每请求20个样本，p95取排序第19项并取整。

| 场景/请求 | 客户端p95 ms | Go投影/JSON p95 ms | Swift复制/解码 p95 ms | JSON bytes |
| --- | --- | --- | --- | --- |
| 混合 / 目录 | 19 | 19 | 0 | 2721 |
| 混合 / 历史 | 24 | 23 | 1 | 40460 |
| USD / 目录 | 12 | 12 | 0 | 2109 |
| USD / 历史 | 14 | 14 | 0 | 39644 |

目录4/2商品，源历史total4000、首页100；BuildCount保持1。打开加首张树约1890/1544ms，为客户端调用，不是GUI首帧。0ms为取整，不代表零成本。完整所有样本/既有页面/重载见[原始记录](a03-prices-client-2026-10-01.txt)。输入SHA256：mixed=`c33b0e61e5c9a1d5cf58d78bc78dbf3e6b1a730ca9aac474919a315af891817c`，USD=`3f4cfb702e845a5b863d80922b50719436b42ff3c35538756e95aa59aab7a69a`。

## GUI与剩余要求

最新独立QA bundle已由当前app复制/ad-hoc签名。CUA getApp再次失败：ScreenCaptureKit SCStreamErrorDomain -3811，无法取得页面截图/AX。未声称价格页显示、最小窗口、大字体、键盘/AX或GUI性能通过。随后只对该QA精确可执行路径发送SIGTERM并确认消失，用户原OrangeCount进程保留。

A01/A02/A03完整GUI、N04/R01阅读和独立安装循环仍缺。Q01/Q02/X01/X02继续按实施手册第31/32节开发；远端CI、旧系统/另一机器、Developer ID和公证仍需要真实条件。未创建提交、PR、远端run或发布。

## 相关公开代码身份

HEAD仍为`20a9a6e86a5b4e2d6bb51aa2d2ae19dcd2934ddd`加已有工作区；以下为当前文件SHA256，不包含私人输入。

| 文件 | SHA256 |
| --- | --- |
| `internal/report/native_prices.go` | `8655c4c6023b2e789228ec70c6032d42aca452cc4bbf0b016cefd4bb97095034` |
| `internal/report/native_history.go` | `5e2b4337b26524ff239ccb756892cb9703f388b335fa88a9d0239adb54df2211` |
| `internal/report/account_detail.go` | `cb7fb18aa545b73309f47961b0c0c4ee27c5136091daef222deadb8f36588eb0` |
| `internal/report/native_average_history.go` | `8f8b1dd60426451d37396237cfd1becd955a66611924b733608b04a1cf118528` |
| `internal/desktop/prices.go` | `fbc3407cc9572ffb24554f7a96003ee1941bcf50532f7a44295a8ffabb944059` |
| `internal/desktop/prices_test.go` | `1553dc13e6463ae00cea3bb64a1300f77528fd017f2487b145ab93226e635fcf` |
| `internal/desktop/journal.go` | `3ebad8d1c5baed35b032ada500fa8c9a29586368aff66ac46407d927437c452a` |
| `internal/desktop/projection.go` | `f92deb919e9b24a3bc53f695f731472f9f1560c6882496e77795d059abd9ca63` |
| `apps/macos/src/Models.swift` | `4c18bb43b18328fc74da3333219ae29d894c7ffcf96ee33c4d5eaac8bab8cfa3` |
| `apps/macos/src/LedgerStore.swift` | `4a01a0b9966a68acdf1ffdaf48e50115b40ba64a63f1d63de4a328c92265a4dc` |
| `apps/macos/src/PricesView.swift` | `b7a5daff883f7c7cb40f3fe2b761bda876843ced9d17f049116cb569c6f207d1` |
| `apps/macos/tests/PrivateReadProbe.swift` | `3eecf7160a1a238539da82aa823975ddb3d22fc99a667cd2e34ac1af6e351ab6` |
| `tools/reference/verify_full_ledger.py` | `d7638788cc47c53f17686932e94c574c1cd09bbfedfcb29a8c6c9ce34dd5cf24` |
| `apps/macos/tests/SessionBenchmark.swift` | `c15053a87ebc217c423961b65e2fd8822ffa3e73a8a59e030d721709304c6000` |
