# A02 原生持仓、会话与重放性能证据

日期：2026-10-01。适用范围：当前 dirty 工作区的 A02 实现和本次检查；不证明完整 M0–M4、GUI 或公开分发完成。此前 Store red 见[文档交接基线](go-native-planning-baseline-2026-10-01.md)，该 red 已修复。

## 实际实现

- 持仓侧栏及账户入口，groups/events/history 主页、完整批次与合并贡献详情、精确金额、未知原因、来源、均价图分段及精确伴随列表。
- Store 的 selector/分页/排序绑定当前可见快照，Back 恢复请求，日期/分组/账户/快照改变后清旧 selector，离开持仓不泄漏参数；复用单一原生会话。
- 重放使用一次调用内的完整精确键索引，代替按每个 posting 扫描整个账户批次。清零时删除键，同键重新买入建立新位置和来源；保留无成本/零成本、nullable label/date、signed 库存及 M 顺序。未新增跨快照缓存，未改 booking 默认或财务精度。
- 私有探针扩展独立 Python 历史库存预期：第一、居中、最后日期去重后选择截至日；比较各数学域数量/成本/均价及完整批次，实际 Swift/C 分页读取。期望文件只在仓库外临时目录内存在并自动清理。

## 当前自动化与实际客户端

| 检查 | 当前结果与范围 |
| --- | --- |
| `go test -count=1 ./...` | 全量通过；包含完整重放与32个合法 pinned oracle 最终库存 |
| `go test -race -count=1 ./internal/ledger ./internal/report ./internal/snapshot ./internal/desktop ./internal/bridge` | 通过 |
| `make fmt vet license parity-full` | 通过，full oracle 46/46，具名 AVERAGE 扩展单独通过 |
| `make mac-test` | ContractTests、StateTests、NativeClientTests 全部实际执行通过 |
| `make app` | arm64/macOS13编译目标、plist、ad hoc bundle 检查通过；不是旧系统实机或公开签名证明 |
| 当前私人 `make verify-private-ledger` | 47文件、双方2831指令、零错误/分录差异/库存差异；Expenses73行、Income22行一致；same_input_capture/native_checked/native_contract=true，native_holding_cutoffs=3 |

新增具体检查：101个 AVERAGE 原贡献批次的100项预览和最后1项；include 报价 FileID 到正确源文件/起止范围的真实解码；清仓后同完整成本键再买入的新建立来源；受控 lots→account→Back、cutoff 改变及迟到响应；真实历史/merge 分页、详情降序和 reload 清过期选择器。个人账本没有 AVERAGE，AVERAGE 仍靠合成数学样例验收。

## 客户端性能与诊断

同机 Mac16,10 / Apple M4、24GiB、arm64；macOS27.0.1（26A434）；Go1.27.1、Swift6.4、SDK27.0。性能测量与其他 agent 构建分开执行；用户原有应用未停止。

输入为固定公开生成器：单文件52500交易、2000买入/500归约、1500剩余成本批次。持仓请求固定 Assets:Broker:AAPL，1组/500事件/1系列/168历史点；groups 预览100 lots、100 events、最近100 points，详情请求读取相应主集合。每项20样本；p95取排序第19个，表中为脚本取整值。

| 输入 / 请求 | 优化前客户端 p95 ms | 优化后客户端 p95 ms | 优化后 JSON bytes |
| --- | --- | --- | --- |
| 混合 / groups | 426 | 64 | 243413 |
| 混合 / events | 432 | 60 | 157185 |
| 混合 / history | 404 | 57 | 30395 |
| 混合 / lots | 440 | 62 | 59304 |
| 混合 / history points | 429 | 61 | 30003 |
| 混合 / consumed | 413 | 57 | 3365 |
| USD / groups | 301 | 67 | 242795 |
| USD / events | 299 | 60 | 156767 |
| USD / history | 297 | 65 | 30377 |
| USD / lots | 298 | 65 | 59086 |
| USD / history points | 298 | 65 | 29985 |
| USD / consumed | 293 | 63 | 3343 |

载荷字节数及上述记录总数前后相同，性能改善不依靠删页、减量或降低精度。结果低于200ms客户端预算；GUI首帧/交互尚未测量，不能据此认定 GUI 门槛通过。

原始完整样本：[优化前](a02-holdings-client-before-2026-10-01.txt)、[优化后](a02-holdings-client-after-2026-10-01.txt)。含 Go/C投影与JSON、Swift复制/解码、全部读取样本和其余现有页面，不只保存p95。输入 SHA256：

- mixed：`c33b0e61e5c9a1d5cf58d78bc78dbf3e6b1a730ca9aac474919a315af891817c`
- USD：`3f4cfb702e845a5b863d80922b50719436b42ff3c35538756e95aa59aab7a69a`

单请求定位命令：`go test ./internal/desktop -run '^$' -bench '^BenchmarkNativeHoldingsProjection$' -benchtime=5x`。固定公开文件，不接受私人路径覆盖。优化前加 `-cpuprofile` 的 profile 表明 ReplayBookedInventory 1.63s累计样本，averageInventoryHistory 0.04s、JSON 0.02s；精确键线性比较是主因。二次验证用同一请求及原始客户端场景，不修改200ms目标。

```text
before: BenchmarkNativeHoldingsProjection-10    	       5	 374855542 ns/op	473099956 B/op	21638425 allocs/op
after:  BenchmarkNativeHoldingsProjection-10    	       5	  56966975 ns/op	59096947 B/op	 2204608 allocs/op
```

Go benchmark 是5次读取均值及分配量，不是 p95。profile 数据和临时二进制保留在 build 诊断目录，无产品调试日志。微基准与完整客户端各有独立结果。

## GUI 与尚未完成范围

准备隔离 QA bundle 并尝试读取原生状态时，Computer Use 返回 ScreenCaptureKit -3811（音视频捕获失败），getApp/AX/键盘退出均受影响。没有取得新持仓页面截图或可靠 AX 状态，不记录页面渲染/键盘/窗口/字号验收通过。隔离 QA 精确进程随后终止并核实不存在；用户原 OrangeCount 实例未关闭。

A02 的 GUI/AX/900×600/大字体及双窗口仍待验证；G05 均价曲线实际展示同样未验收。A03、Q01/Q02、X01/X02 仍需实施；R01独立安装、R02远端执行、R03目标系统/公开签名仍缺。独立开发可以继续，完整目标未完成。

## 相关源码身份

HEAD仍为 `20a9a6e86a5b4e2d6bb51aa2d2ae19dcd2934ddd` 加既有和本次工作区。下列hash界定证据适用范围，未包含私人输入。

| 文件 | SHA256 |
| --- | --- |
| `internal/ledger/inventory_replay.go` | `c2c473f57ae7a58f4c9327dd88feab70d7cefbdf6dbb32adce66291342b0f535` |
| `internal/desktop/holdings_test.go` | `2280b49d13a5df3bb030f65f4ee4b024c235ebda599344bd30583c4319018197` |
| `internal/desktop/holdings_benchmark_test.go` | `4ba677d53d5d599a8411ab07cbef89845dcaca8a2037de738ae7fc54db4a1290` |
| `apps/macos/src/LedgerStore.swift` | `47abe5954fa57c2eb45a04f22860e399e36dd478ee187bfc17fa80ad42da37bb` |
| `apps/macos/src/HoldingsView.swift` | `434819c7be4bd685342342873a6392347f550976b6d561a3c5cb58b9662d04bb` |
| `apps/macos/src/main.swift` | `7750d883df314c5f3a48fab046baa61236b7ad81f7ea82e93182a25778b51374` |
| `apps/macos/src/Models.swift` | `67ea57bdd1a33ce18a2dcbd3a4eb9a0ba47c9fad886f6225950b135c4f42ae8a` |
| `apps/macos/src/ReadingControls.swift` | `075b43fcb9f8e14d9fd6e3adc80df95d2d00cce7fb16e46078d5b1f5b4cb5fd7` |
| `apps/macos/src/AccountDetailView.swift` | `93e4da843ea71c753249bb420d539116390b711cac79ac80921db117ad083435` |
| `apps/macos/tests/NativeClientTests.swift` | `fd68c211961471c21632d3e6dba79e87bd8bb7612a14d30f591ec90b4df68345` |
| `apps/macos/tests/ContractTests.swift` | `ab924ea595f1d8d7d06837e26d0f1949c6507286035a90a8c32fa5c2264326b3` |
| `apps/macos/tests/PrivateReadProbe.swift` | `9d1b9368733f6f7d1c42976e83f2fbadc28a17e6af917317e035b441f6f5cd2c` |
| `tools/reference/verify_full_ledger.py` | `79167102e037a098b4ca35e8c39ea9fa6cede38f054d0e1c516fc8b6c5f01b11` |
| `apps/macos/tests/SessionBenchmark.swift` | `4efa279b43c0f698d73c79885553a6a845ada2e97ab59ee3482f33ff26a814ae` |
