# Go/native设计、计划与步骤修订：本次复验

日期：2026-10-01。工作区HEAD为 `20a9a6e86a5b4e2d6bb51aa2d2ae19dcd2934ddd`，已有大量dirty/untracked实现。本次只修改文档，未修改产品源码、私人源文件、启动应用、创建提交或执行远端发布。三份主文档及新矩阵是后续开发规格，不是实现报告。

## 本次执行与适用范围

| 检查 | 结果 | 不能由此推断 |
| --- | --- | --- |
| 当前代码检查 | query为标题map结果；FROM必需；Limit>0才截断；函数主要在逐行求值；GROUP BY有忽略error路径 | 未执行的新typed契约不算实现 |
| query/desktop具名用例（命令见下） | 两包exit0 | 仅匹配的用例；非query/desktop全量，更非全量Go或GUI |
| 私人入口环境变量 + `UV_OFFLINE=1 sh tools/reference/differential.sh` | exit0；same_input_capture=true | 本次native_checked=false，未重跑私人Swift/C |
| 固定oracle公开探针 | 16场景实际执行；10成功、6预期CompilationError | 只证明这些参考行为，未运行新的Go类型实现 |
| 文档校验 | 本地链接、代码块、任务数与空白检查 | 非产品验收 |

私人摘要：captured_files=47；双方entries=2831/errors=0/valid=true；missing_entries=0、extra_entries=0、changed_inventory_accounts=0；Expenses rows=73、Income rows=22，双方equal=true；schema_version=2、semantics_profile=v3、same_input_capture=true、native_checked=false。没有保存个人源内容、账户、金额、币种、日期、摘要、路径或截图。本次账本输入没有AVERAGE及源price指令，有价与均价扩展用独立公开输入验收。

具名检查的实际命令：

```sh
go test -count=1 ./internal/query ./internal/desktop \
  -run 'TestNativePrices|TestNativeCharts|TestQueryParsingAndEvaluationRejectInvalidExpressions'
```

此前全量Go/race、46/46公开差分、真实C/Swift、应用及客户端性能属于[A03实施证据](a03-native-prices-2026-10-01.md)，本次文档修订未重复这些门槛。早期[交接red](go-native-documents-handoff-2026-10-01.md)保留历史身份，当前不能据此重做已修复价格校验。

## 公开探针复现

以下只包含合成文本，在仓库根执行；使用锁文件和本地离线环境，不调用私人账本。列类型使用Column.datatype，不能用DB-API的type_code哈希。

```sh
uv run --offline --project tools/reference python - <<'PY'
import json
import beancount, beanquery
from beancount import loader
from beanquery import query, types
source = '''2000-01-01 open Assets:Cash USD
2000-01-01 open Expenses:Food USD
2000-01-02 * "public test"
  txmeta: "foo"
  Assets:Cash -2 USD
    pmeta: "bar"
  Expenses:Food 2 USD
'''
entries, errors, options = loader.load_string(source)
assert not errors
cases = [
    ('default', 'SELECT account, number'),
    ('abs-length-source', 'SELECT abs(number), length(account), filename, lineno'),
    ('metadata', "SELECT meta('txmeta'), meta('pmeta'), entry_meta('txmeta'), any_meta('txmeta') LIMIT 1"),
    ('duplicate-title', 'SELECT 1 AS v, 2 AS v LIMIT 1'),
    ('limit-zero', 'SELECT number LIMIT 0'),
    ('empty-aggregate', "SELECT count(*), sum(number) WHERE account = 'NO'"),
    ('division-zero', 'SELECT number / 0 LIMIT 1'),
    ('bad-type-empty', "SELECT abs(account) WHERE account = 'NO'"),
    ('source-as-function', 'SELECT filename(), lineno()'),
    ('units-column', 'SELECT units'),
    ('cost-column', 'SELECT cost'),
    ('typed-position', 'SELECT position, units(position), cost(position), weight LIMIT 1'),
    ('typed-amount', 'SELECT number(units(position)), currency(units(position)) LIMIT 1'),
    ('typed-inventory', 'SELECT sum(position), units(sum(position)), cost(sum(position))'),
    ('weight-function', 'SELECT weight(position)'),
    ('avg', 'SELECT avg(number)'),
]
print(json.dumps({'beancount': beancount.__version__, 'beanquery': beanquery.__version__}))
for name, text in cases:
    try:
        columns, rows = query.run_query(entries, options, text)
        print(json.dumps({'id': name, 'query': text,
            'columns': [{'name': c.name, 'type': types.name(c.datatype)} for c in columns],
            'rows': [[repr(cell) for cell in row] for row in rows]}, ensure_ascii=False))
    except Exception as error:
        print(json.dumps({'id': name, 'query': text,
            'error': type(error).__name__, 'message': str(error)}, ensure_ascii=False))
PY
```

这个探针显示参考行为，并非strict gate：捕获错误后继续输出，用例类别和值需要逐项核对。后续实施必须加入类型化golden和断言，不把此脚本的exit0当作所有输入成功。

## 本次实际观察

| ID | 结果摘要 |
| --- | --- |
| default | str账户、decimal数量；两行-2/+2 |
| abs-length-source | decimal绝对值2；int长度11/13；filename为公开字符串源；int lineno为posting行5/7 |
| metadata | 四个object列；第一行依次NULL、bar、foo、foo |
| duplicate-title | int列v、v；值1、2都保留 |
| limit-zero / empty-aggregate | 成功零行，仍有列类型；count为int、sum为decimal |
| division-zero | decimal列的值为NULL |
| bad-type-empty | CompilationError：abs(str)不匹配，空WHERE不能免除编译 |
| source-as-function | CompilationError：filename()不匹配 |
| units-column / cost-column | CompilationError：postings未知列 |
| typed-position | position、amount、amount、amount；首行均表示-2 USD，无成本 |
| typed-amount | decimal -2、str USD |
| typed-inventory | 三个inventory列，均为空；不返回单币种0 |
| weight-function / avg | CompilationError：weight(position)、avg(decimal)不匹配 |

完整有成本、stack/include、NULL排序、GROUP/alias与运行预算本次未验证。已写入矩阵/实施步骤，下一批逐项建立失败信号。

## 文档变更与后续入口

本次新增[文档导航](../go-beancount-native-roadmap.md)、[查询矩阵](../query-capability-matrix.md)，修正设计中来源与元数据签名和历史red状态，补查询编译/Adapter契约、当前剩余估算及下一次开发的第一批步骤。设计/计划/手册沿用原文件名和24个顶层任务，避免断开既有引用。

下一次从实施手册第33节开始。未完成GUI、N04/R01完整阅读与独立安装循环、远端CI、其他机器/旧系统、Developer ID和公证仍保留；本次未申请或代替这些外部条件。

## 适用的公开文件身份

| 文件 | SHA256 |
| --- | --- |
| `internal/query/query.go` | `218fc307e972a036c3f436c37d353f5d27e4c446bf330b9ee33a5ca4cb2ad6f0` |
| `internal/desktop/prices_test.go` | `1553dc13e6463ae00cea3bb64a1300f77528fd017f2487b145ab93226e635fcf` |
| `tools/reference/verify_full_ledger.py` | `d7638788cc47c53f17686932e94c574c1cd09bbfedfcb29a8c6c9ce34dd5cf24` |
| `tools/reference/uv.lock` | `60acbc24073541a373b032e1e8c9a21949c8b354cbc1f77b6f0553609f8931ec` |
