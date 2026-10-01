# Go会计引擎与原生macOS：当前文档交接证据

日期：2026-10-01。用途：供[设计](../go-beancount-native-design-2026-09-30.md)、[计划](../go-beancount-native-plan-2026-09-30.md)与[实施手册](../go-beancount-native-implementation-2026-09-30.md)后续开发使用。本次仅修改文档，保留已有产品源码和未提交修改；未改私人源文件、创建提交、启动GUI或发布。HEAD为`20a9a6e86a5b4e2d6bb51aa2d2ae19dcd2934ddd`，当前工作区不能仅由HEAD重现。

## 本次实际检查

| 命令 | 退出结果 | 适用范围 |
| --- | --- | --- |
| `go test -count=1 ./internal/query ./internal/desktop` | 0，两包通过 | 不包含compat、不等于全量Go/race或查询gate通过 |
| `make query-parity query-oracle-check` | 2；query-parity报告checked=101/failed=1，子进程exit1 | make在首项失败后停止，后面的oracle-check未执行 |
| 单独`make query-oracle-check` | 0；九个corpus的101期望均与锁定oracle一致 | 不代表Go结果通过 |
| `make parity-full` | 0；schema2、50/50普通会计案例；具名AVERAGE/global booking测试通过 | AVERAGE两个extension另列，不以Python失败作为数学期望 |
| 外部入口环境变量、`UV_OFFLINE=1 sh tools/reference/differential.sh` | 0 | 本次Go/Python同源对照，native_checked=false |
| 公开cast oracle/CLI探针 | 见下一节 | 两个新增差异尚未纳入101个注册场景 |

固定版本为Beancount3.2.3/BeanQuery0.2.0，依赖身份来自uv.lock。本次没有执行全量Go、race、真实Swift/C、app构建、GUI、性能、远端CI、旧系统/其他机器、签名或公证。此前这些通过记录属于[动态/NULL批次](q01-dynamic-null-query-2026-10-01.md)的继承证据，不能覆盖当前cast改动。

本次私人摘要只保留计数与差分类别：captured_files=47；双方entries=2831/errors=0/valid=true；missing_entries=0、extra_entries=0、changed_inventory_accounts=0；Expenses rows=73、Income rows=22，双方equal=true；same_input_capture=true、schema_version=2、semantics_profile=v3、native_checked=false。没有保存私人源内容、账户、金额、来源路径列表、哈希或截图。该检查不能验收AVERAGE、所有查询重载或原生GUI。

## 当前失败及公开参考行为

| 公开输入 | 固定期望 | 当前Go观察 |
| --- | --- | --- |
| `casts/arithmetic-decimal-string` | 四列`3.00`、`1001`、`-0.00`、`11` | 注册gate：execution-error |
| `SELECT str(decimal(1.00 + 2)) LIMIT 1` | string `3.00` | CLI exit0、string `3`，静默精度损失 |
| `SELECT date(2000, TRUE, 1) LIMIT 1` | date 2000-01-01 | CLI exit1，requires integers |
| `SELECT str(position), str(weight), str(sum(position))` | 首行`1 USD`、`1 USD`、`(2 USD)` | 只执行oracle探针，本次未验证Go兼容，不算已实现 |

cast语料十个场景中九个匹配、一个失败；此前八组共91场景本次未报告差异。101是当前注册数，不包含表内两个额外差异，也不由此推导整个Q01完成。

公开探针实际复现命令：

```sh
uv run --offline --project tools/reference python - <<'PYPROBE'
from beancount import loader
from beanquery import query
entries, errors, options = loader.load_file('testdata/fixtures/query/casts.bean')
assert not errors
for sql in [
    "SELECT date(2000, TRUE, 1) LIMIT 1",
    "SELECT str(1.00 + 2), str(decimal('1.0e3') + 1), str(decimal('-0.00') * 2), str(meta('padded') + 1) LIMIT 1",
    "SELECT str(decimal(1.00 + 2)) LIMIT 1",
    "SELECT str(position), str(weight), str(sum(position))",
]:
    try:
        columns, rows = query.run_query(entries, options, sql)
        print({'query': sql, 'columns': [c.name for c in columns], 'rows': repr(rows[:1])})
    except Exception as error:
        print({'query': sql, 'error': type(error).__name__})
PYPROBE
```

这是公开探针，捕获异常后继续；脚本exit0不能证明每个查询成功。CLI复现及后续strict化步骤见实施37.1–37.4。本次没有重新生成golden或删除失败语料。

## 文档交付与适用源码身份

三份主文档新增当前基线说明：设计30节固定精确值/数值上下文/表示信息的职责；计划18节划分剩余执行批次；实施37节提供当前真实失败、独立期望、文件责任及升级验证。roadmap、matrix与progress同步，24个顶层任务和完整退出门槛保留。本次文档检查结果在下节记录；文档交付不表示产品目标完成。

以下只包含公开源码和公开输入，旧证据哈希不改写。

| 文件 | SHA256 |
| --- | --- |
| `internal/query/query.go` | `441cd76aa5a0eec869b4b0ccc07774880d87ebc085823517fcb806306b5e999d` |
| `internal/query/typed.go` | `6a3e6a69583dc862ae3ea76b4a12a3c51716d0261a3c457e78b8d6beceeca745` |
| `internal/query/coercion.go` | `3f685ebbdabee122cece0b3086eb3619ef5c2b32fcb0aff0923ce9f1675ff4c0` |
| `internal/query/casts.go` | `7a0b6faaed74d562c9a71bc2e5d7e665daa03478bc9d82ec510150ea84b94cff` |
| `internal/query/compound.go` | `da8228570a17b7485203ce39548080c73ee45c02c8678402aab5fc00a53053a5` |
| `internal/compat/query.go` | `95bfb5586a8f09ce5674aab11ff60f38990037c36420ce96ced14e437f08b108` |
| `tools/reference/generate_query_golden.py` | `09b14c1b54731ab30b30fc3cfd9b29cdfa63ca8801ae9d2dd54e926dce9edca2` |
| `tools/reference/uv.lock` | `60acbc24073541a373b032e1e8c9a21949c8b354cbc1f77b6f0553609f8931ec` |
| `testdata/fixtures/query/casts.bean` | `c0e689ac4a2b72052306673849d7dfdc2283c8766e4351fc446f2c939d6a38f0` |
| `testdata/fixtures/query/casts.queries.json` | `17bd9a5ac0efcc3052934ac0604884891491d68e1d1cac3200db12b4a563c939` |
| `testdata/fixtures/query/dynamic-values.bean` | `a7136be9a2979dc6f9c31a1baaf8a6c59b9817260359b1f931f74828eb323f8b` |
| `testdata/golden/query-v1/casts.json` | `5b072b5ba7d98b03ddd238c4cb1c6453795f717f51ef59748442a2024d087bbb` |

## 文档检查结果

本次实际检查7个交付文档：91个本地链接及其锚点、16段shell代码的`sh -n`语法、24个唯一顶层任务ID与依赖引用、12项公开源码/输入SHA256，均通过；代码块配对、末尾换行和行尾空白检查通过。`git diff --check`限定本批文档退出0；未跟踪文档的内容由上述独立检查覆盖。检查不执行实施手册中的未来修改命令，也不构成产品验收。
