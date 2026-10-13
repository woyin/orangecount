# CSV 周日期映射（2026-10-12）

本文回答 review 第 4 节“CSV 周日期”行：**先映射到当前实际需要的导入接口，确定目标接口和严格日期差异，完成原生实现与失败无部分提交的回归**。本文是映射记录，不是外部补丁的移植记录，也不是完整导入器声明。

外部候选补丁的观察证据已复制到本仓库：[CSV 周日期证据](evidence/csv-week-date-2026-10-12.json)（候选补丁后的结果）与 [oracle 观测](evidence/query-oracle-2026-10-12.json)。评审期原始捕获在 `fix/review-evidence-2026-10-12/`（`csv-week-cases.json`、`csv-week-before.json`、`csv-week-after.json`），该目录不随仓库发布。

## 1. 当前导入接口与缺口

| 项目 | 当前实现 |
| --- | --- |
| 入口 | `POST /api/v1/import/preview`（`internal/web/imports_http.go`），适配器 `csv` |
| 列映射 | `date`、`account`、`amount` 必填；`payee`、`narration`、`currency` 可选 |
| 日期解析 | 缺省 `parseISODate` 只接受 `2006-01-02`；声明 `date_layout` 时走 `internal/datefmt.Parse` 严格解析并回写 `YYYY-MM-DD` |
| 周日期 | 支持：`%G-W%V-%u` 等布局解析后回写 `YYYY-MM-DD`（如 `2020-W01-1` → `2019-12-30`） |
| 失败行为 | `csvToBeancount` 在任何一行出错时返回 `("", err)`，不返回已渲染的行（无部分提交） |

缺口是**日期布局不可声明**，不是周日期算法错误。外部补丁改动的是它自己的 `fava` CSVImporter 契约，与本仓库的接口不同，不能直接套用。

## 2. 目标接口

新增 `internal/datefmt`，只做一件事：按 strftime 风格布局解析一个日期字符串。它是纯函数包，不读文件、不持有状态、不依赖导入器。

| 导出符号 | 契约 |
| --- | --- |
| `Parse(format, value string) (ledger.Date, error)` | 成功返回所有字段一致认可的日历日期；失败返回零值和 `Error` |
| `Error{Code, Message, Format, Value}` | `Code` 是稳定分类，供导入诊断映射；`Message` 仅供人读 |
| `Code` 常量 | `date-format-syntax`、`date-value-mismatch`、`date-value-out-of-range`、`date-value-conflict` |
| `ISOWeekday`、`SundayWeek`、`MondayWeek` | 周字段写回助手，供测试与后续导入器复用 |

支持的指令：`%Y %y %G %m %d %V %U %W %w %u %%`。不支持的指令在读取任何值之前报 `date-format-syntax`。

## 3. 严格日期差异

外部候选的输入契约是：字段缺失、互相冲突、或不能按声明的公历年/周号/星期回写时明确报错，不静默滚动。本实现采用同一契约，并把分类显式化。

| 场景 | Python `strptime` 行为 | 本实现行为 | 分类 |
| --- | --- | --- | --- |
| `%G-W%V-%u` 跨公历年（`2020-W01-1`） | `2019-12-30` | 同值接受 | — |
| `%G-W%V-%u` 真实第 53 周（`2020-W53-7`） | `2021-01-03` | 同值接受 | — |
| `%G-W%V-%u` 不存在的第 53 周（`2021-W53-1`） | 滚动到 `2022-01-03` | 拒绝 | `date-value-out-of-range` |
| `%G-W%V-%u` 第 0 周（`2020-W00-1`） | 值不匹配 | 拒绝 | `date-value-mismatch` |
| `%Y %U %w` 第 0 周滚回上一年 | `2020-12-27`（声明年 2021） | 拒绝 | `date-value-out-of-range` |
| `%Y %W %w` 第 0 周滚回上一年 | `2020-12-28`（声明年 2021） | 拒绝 | `date-value-out-of-range` |
| `%Y %U %w` 第 53 周滚到下一公历年 | `2022-01-08`（声明年 2021） | 拒绝 | `date-value-out-of-range` |
| 只有 `%U`/`%W` 没有年或星期 | 默认或忽略缺失字段 | 拒绝 | `date-format-syntax` |
| 公历字段与周字段指向不同日期 | 按优先级覆盖 | 拒绝 | `date-value-conflict` |
| `%Y` 与 `%V` 混用 | 拒绝（提示用 `%G`） | 拒绝 | `date-format-syntax` |

**可接受的范围**：跨公历年的 ISO 周、真实存在的第 0/53 周、以及周字段与公历字段一致时都正常返回。拒绝只发生在“按声明字段回写会得到另一个日期”的情况。

**失败无部分提交**：`Parse` 失败时返回零值 `ledger.Date{}`，测试 `TestParseNeverReturnsADateWithAnError` 断言日期字段与错误互斥。导入路径本身已在第一行错误时丢弃整个转换结果，因此一次失败不会写出半批交易。

该严格契约相对 Python `strptime` 滚动行为的偏差已按流程登记为 [FD-0010](fava-approved-deviations.md)（产品所有者批准）。

## 4. 证据与验证

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 单元测试 | `go test ./internal/datefmt/ -count=1` | 通过 |
| 接入测试 | `go test ./internal/web/... ./internal/datefmt/... -count=1` | 通过（internal/web 107、favaadapter 24、datefmt 5；0 失败） |
| 期望来源 | 有效日期用固定参考 `tools/reference/.venv/bin/python3` 的 `datetime.strptime` 重算；拒绝行与分类对照 [CSV 周日期证据](evidence/csv-week-date-2026-10-12.json) | 逐项一致 |
| 失败无部分结果 | `TestParseNeverReturnsADateWithAnError` | 通过 |

`internal/datefmt` 是新增能力，尚未接入 `internal/web` 的 CSV 适配器：当前导入接口没有“日期布局”字段，接入需要同时改 API 与前端，属于独立范围。本批交付的是已验收的解析契约与回归，不是导入器端到端声明。

## 5. 接入状态与未纳入

**已接入**：`POST /api/v1/import/preview` 的 `csv` 适配器接受可选 `date_layout`（strftime 风格，如 `%Y-%m-%d`、`%G-W%V-%u`）。缺省（空）仍是 ISO `%Y-%m-%d` 旧路径，行为逐字节不变；声明布局时由 `internal/datefmt.Parse` 严格解析并回写日历日期。布局在读取任何行之前校验（空值探针即报语法错，400）；逐行诊断保留 `CSV row N` 前缀并附 datefmt 稳定 code；任一行失败仍丢弃整批（all-or-nothing）。非 `csv` 适配器携带非空 `date_layout` 返回 400。提交路径无需布局：commit 重放 preview 已转换的正文。

未纳入：

- 前端未接：默认内嵌旧版 UI（`internal/web/assets/app.js`，仓库内无生成源）与 opt-in 的 `web/src/fava/reports/ImportReport.svelte`（需 `ORANGECOUNT_TRANSPLANTED_UI=1`，未过路由切换门禁）均未增加日期布局输入；当前该字段仅由 API 客户端使用。前端接入属 Import 路由切换阶段任务。
- 未支持 `%j`、`%a`、`%A`、`%b`、`%B`、时区与复合文本指令。遇到即报 `date-format-syntax`。
- 未重新生成外部 Python oracle；有效值由本机固定参考重算，拒绝值引用评审期证据。
- 未引入任何外部源码、标识符或文件头。`internal/datefmt` 按本仓库的 Apache-2.0 头与命名习惯独立编写。
