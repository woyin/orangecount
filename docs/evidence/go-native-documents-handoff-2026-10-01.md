# 详细设计、计划与实施手册的当前交接证据

日期：2026-10-01。本次交付为三份主文档的完善和只读复验；没有修改产品源码、私人账本、创建提交、触发远端或发布。

## 实际结果

| 项目 | 当前验证 | 范围与限制 |
| --- | --- | --- |
| 价格targeted | `go test -count=1 ./internal/desktop -run '^TestNativePrices'`：exit 1 | 目录和日期选价两个具名测试通过；参数测试失败，prices错误接受valuation=units |
| 价格具名明细 | 同范围加`-v`，显示2 PASS / 1 FAIL | 不声明整个desktop或Go全量green；失败后的参数格尚未执行完 |
| 私人只读差分 | 外部入口参数加`sh tools/reference/differential.sh`：exit 0 | 当前同源捕获，完整分录/库存及月度查询比较；本次native_checked=false |
| 文档检查 | 本地链接/锚点、代码块、24任务、空白及`git diff --check` | 文档一致性检查，不代表实现或GUI验收 |

私人摘要仅保存：captured_files=47；双方entries=2831、errors=0、valid=true；missing_entries=0、extra_entries=0、changed_inventory_accounts=0；Expenses rows=73、Income rows=22，双方equal=true；same_input_capture=true；schema_version=2；semantics_profile=v3；native_checked=false。没有保存私人源文件、账户、金额、币种对、日期范围、摘要、路径或截图。

## 当前交接点

已有A02 Store/UI及其历史green保留。已有A03 Go目录、源历史与选价，以及commodity-not-found桥接错误；Swift价格DTO/Store/UI尚未接入。后续先执行实施手册第30.1节，使当前units请求校验red转green，再补第30.2节独立边缘和分页，然后贯通真实C/Swift与原生页面。Q01/Q02/X01/X02按第31/32节实施。

本次不运行全量Go/race/parity、mac-test、app、GUI或benchmark；这些门槛在后续产品实现后重新执行。A02当前私人三个截至日持仓与客户端57–67ms p95属于[A02证据](a02-native-holdings-2026-10-01.md)，不能与本次native_checked=false混成一次验收。GUI捕获-3811、完整N04/R01、远端CI、旧系统/其他机器、Developer ID及公证仍有缺口。

HEAD为`20a9a6e86a5b4e2d6bb51aa2d2ae19dcd2934ddd`，含大量已有未提交修改。后续开发必须保留实际工作区；只checkout HEAD不能重现本次实现。相关公开代码/输入身份如下；这些哈希不包含私人账本。

| 文件 | SHA256 |
| --- | --- |
| `internal/report/native_prices.go` | `8655c4c6023b2e789228ec70c6032d42aca452cc4bbf0b016cefd4bb97095034` |
| `internal/desktop/prices.go` | `fbc3407cc9572ffb24554f7a96003ee1941bcf50532f7a44295a8ffabb944059` |
| `internal/desktop/prices_test.go` | `32f523c4c6ac8e68cc77c4a7f52ac3b3be3ee1a40bd15687c7a7f766d03d3c34` |
| `internal/desktop/journal.go` | `6de70832bf05af3988c7ec951b2b328d29162d39e71c5b18c7a9880e3e5881d7` |
| `internal/desktop/session.go` | `d9aa75865cddd83e4bbf6d25bdf56ea474b50647d90bf1dc77071efcd19ef4f7` |
| `internal/bridge/session.go` | `f9ded962f0fd47e236bb57c1b16b07d6fc075725f27e34a7ea39e9fb00c96d22` |
| `testdata/fixtures/native/prices.bean` | `9151756cb8bb417ed0bb483057d0968b28e6df624c8b86160ca4aa84cc0210e9` |
| `testdata/fixtures/native/prices-included.bean` | `2f5ce02fb54bca9cf91805732e55a35dde197e932e7fe22028e8caa2ecac4109` |

配套：[设计](../go-beancount-native-design-2026-09-30.md)、[计划](../go-beancount-native-plan-2026-09-30.md)、[实施手册](../go-beancount-native-implementation-2026-09-30.md)、[持续进度](../go-beancount-native-progress.md)。上述文件规定目标与实施顺序，本记录只证明实际执行的范围。
