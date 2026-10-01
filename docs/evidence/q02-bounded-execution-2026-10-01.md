# Q02有界执行、会话取消与C/Swift契约首批

日期：2026-10-01。按[实施31.2/39节](../go-beancount-native-implementation-2026-09-30.md)继续实际开发。Q01的147场景保留，本批不改golden、会计规则或个人账本，没有提交、GUI启动或发行。

## 本批实现

- 新公开`query.ExecuteTyped(ctx,text,evaluation,limits)`保留typed contract；旧`EvaluateTyped`/CLI/Web入口不强加交互预算。结果分开returned_rows/row_limit/truncated，不给未经证明的total_rows。
- UTF-8文本上限64KiB，最多10,000返回行，10,001行作为截断证据；排序/分组先执行，文本LIMIT与预算截断分开。丢弃的证明行不消耗返回字节预算，完整native响应上限16MiB，含context与C envelope。
- 默认最多2,000,000工作步；解析准备、源排序/扫描、WHERE、分组、聚合投影、结果排序和编码有实际检查。深Decimal表达式的重复求值也计入预算，不仅截最后显示行。每次执行regex cache最多64个pattern、每pattern最多4096字节；不会向旧全局无限cache加入交互pattern。
- 交互数值边界为输入coefficient文本4096字节、显式指数绝对值10000、计算值分子/分母各65536 bits；token最多4096、表达式深度最多64。计算结果在再次作为操作数前校验，阻止指数增长。已验数学值不因此舍入，超过资源边界失败。Context检查在操作之间，不能声明可抢占单次big.Rat/regexp/JSON或库排序内部动作。
- `desktop.Session.ExecuteQuery`捕获SnapshotID/profile/revision/文本及单调request_id，fingerprint涵盖财务视图/文本，排除投递代次。不会偷偷加入共享日期过滤。只用捕获的不可变publication；旧快照初始请求拒绝，无有效快照有no-snapshot。
- 每会话用gate串行计算；新代次取消旧正在计算/等待的代次，记住取消高水位，较晚到达的旧请求不能重新开始。会话超时5秒；有效publication变更及Close取消当前工作。实际应用层的一项current/一项latest-pending队列与Store时序尚未接入，不能仅凭Go gate称UI队列已完成。
- 新C入口`OC_ExecuteQuery`/`OC_CancelQuery`及BridgeClient真实调用，协议仍2。请求/响应nullable执行与错误分开；Swift保留ordered schema/positional cells与完整cost/date/label，拒绝未知cell、错误行宽/ID/类型/NULL形状/预算计数及缺必填nullable key。
- error.range已固定required-nullable UTF-8 byte schema，但当前始终null：精确语法/编译错误位置仍待实现，不将字段存在算完成。

## red与验证

| 切片 | 实际red | 当前验证 |
| --- | --- | --- |
| 新Execute/会话接口 | 测试编译时接口不存在；首次有一次无关unused import，立即清除，不把它计产品red | 10,002源行排序后返回10,000且truncated；LIMIT10不是预算截断；重复标题保留两列 |
| 深表达式预算 | 18层Decimal组合，Steps300仍成功，测试fail，原执行约1.08s | 新执行返回work-budget-exceeded；逐阶段预算测试在WHERE/group/project/sort实际失败 |
| 取消先于派发 | CancelQuery(1)后Execute(1)仍成功，queued cancellation测试fail | 记住代次后返回cancelled，后续代次可执行；迟到旧代次取消 |
| 返回字节的证明行 | 仅允许1行、按实际JSON长度预算时，被丢弃的第二行仍导致错误 | 返回JSON边界通过，证明行不消耗输出预算 |
| 中间值增长 | 1e10000相乘仍成功，numeric-budget测试fail | 计算值超过65536bits明确失败，不继续指数增长 |

补充覆盖：非ASCII空白使文本恰好64KiB仍成功，超1字节失败；零行schema、失败不复用旧成功、fingerprint代次独立、reload旧snapshot拒绝；运行中的Context在第100次检查取消，证明并非仅首尾检测。Session测试重复发取消直到返回cancelled，未记录精确进入C内的阶段，因此不能据此称已完成真实C中途取消的所有阶段/重载/关闭验收。

当前命令均exit0：query/desktop/bridge/compat/CLI targeted；`go test -count=1 ./...`；七包race；`make fmt vet license parity-full query-parity query-oracle-check`（51会计/独立AVERAGE，147查询及十一组oracle）；随后顺序`make mac-test`、`make app`。真实ContractTests新增成功/失败/空行/标量/复合/NULL及取消先到C的正例；六类畸形成功载荷和缺error.range拒绝。StateTests与NativeClientTests原页面三套仍实际执行，但没有查询Store/UI时序用例。

最后重编PrivateReadProbe并链接当前archive，私人同源differential exit0：47文件、双方2833指令/0错误、零分录/库存差异、收支73/22一致，same_input_capture/native_checked/native_contract true、三个holding cutoff及48价格检查。只存聚合，无私人内容/路径明细/哈希/截图。没有新typed私人查询或查询GUI。

## 剩余交付

完整Q02未完成：NativeLedgerClient后台执行与独立取消、Store草稿/执行记录/最新待执行控制、切账本/profile/reload/close乱序、QueryView、保存query/预设列表、精确错误字节位置、固定结果CSV/原子保存与GUI。真实C中途取消/重载/关闭及原生10000/10001/16MiB完整边界、性能也要继续验收。Q01分组/alias/balance/sum(inventory)/预设和数值/复制/loader长尾，X01/X02、N04/R01/R02/R03/D01仍保留24任务目标。

下一步按31.2实现后台客户端与查询Store/UI，复用本批接口与typed schema；解析/编译Range按实际token位置实现，不用猜测子串当精确位置。不要重复已验typed模型或把当前后端切片关为完整M4。

## 当前公开源码身份

| 公开文件 | SHA256 |
| --- | --- |
| `internal/query/execution.go` | `b7d3925f93e397f1cde2de60c58edb400a6b6fac8c61efbaaf19642525bb4304` |
| `internal/query/execution_test.go` | `13b807bc6837bff45e7580733e1c9a99a18eaeab1c535042b039dfdab4f93d09` |
| `internal/query/typed.go` | `b3ce8090fa87c2f93ad6d7bd6a1742673a4a8754fee742e4ba473c6f75e97283` |
| `internal/query/query.go` | `18bf92ebc10bfb95f97ec21c6c5df76943cb70ad63bafda13f735b213c5839b6` |
| `internal/query/coercion.go` | `c66b410696638718b58c6735f8c81e89f941cee25cca9938df5b1847c410d1e6` |
| `internal/desktop/query.go` | `47b36aba3e2673e5a1353d10a6af3e5711af9ff7e737051ba32e182a9ce37fd6` |
| `internal/desktop/query_test.go` | `32654cfe6406444077bbe578fb42cf9cb3602c22c04ba2f7bd32e508e2b0158a` |
| `internal/desktop/session.go` | `2417d2bcaf59860186bc8b3e4da3e3083f2cf6a5ac899991630879b5dd94d7e7` |
| `internal/bridge/session.go` | `dcf513a410e7e40fa931815390440c89dfc49a433df0896c24fd54050bd0ff4a` |
| `apps/macos/src/Models.swift` | `dc22f9bb3a1059ae1dc8292432d3fd841efafe09c333d07cd2450a3fd7ef92d6` |
| `apps/macos/src/BridgeClient.swift` | `e1ed773ec791e3db486467a04353d7265ba4dc8ebc125c04fcd049c492f41479` |
| `apps/macos/tests/ContractTests.swift` | `330491bc3b309866c609aeb086f2807e0553d66481640228e4daf90b1c2ab479` |
