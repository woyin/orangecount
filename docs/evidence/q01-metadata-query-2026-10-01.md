# Q01 有效元数据与开发文档交接证据

本记录保留metadata批次的历史源码身份与63场景结果；当前动态/NULL批次已有91场景，见[最新证据](q01-dynamic-null-query-2026-10-01.md)。下列旧哈希不用于校验当前源码。
日期：2026-10-01。HEAD为 `20a9a6e86a5b4e2d6bb51aa2d2ae19dcd2934ddd`，在保留现有dirty工作区上继续；不是HEAD可重现的已提交发行。前批[复合值记录](q01-compound-query-2026-10-01.md)保留历史输入及源码身份，本记录单列当前状态。

## 实现与范围

查询系统filename/lineno分别读取posting/entry来源，系统行号为Integer而用户数字仍Decimal。any_meta仅在posting缺键时fallback，显式NULL、空字符串、false均保留。用户Amount已有前批实现，本轮保留对照。

v3在文件词法阶段将stack有效值写入交易副本，随后按会计顺序处理；每个include文件独立作用域，交易显式值覆盖stack。push/pop保留在源表示，不进入v3 loaded entries。非法pop与未配对字符串stack产生ParserError类诊断；不继续booking制造附加错误。共用AST、legacy不变的接口回归通过。v3 revision从core-1升为core-2，legacy-1及默认选择保持。

pad生成交易用户metadata来自pad，两posting来自触发balance；posting filename/lineno也来自balance，导航Span继续指向pad。Full黄金和query位置值同时核对，不能只验证来源数字而漏用户metadata。

普通交易的系统__tolerances__是字典对象，目前无map cell规范，entry/any明确unsupported-capability；posting缺失为NULL，pad交易也缺失。固定reference对metadata-source观测类型为NoneType/ImmutableDictWithDefault/ImmutableDictWithDefault，metadata-pad为三个NoneType。此项是已确认并明确拒绝的能力边界，不是成功字典结果等价。

query黄金升级schema 2，记录完整captured source graph的仓库相对路径及SHA。Go gate核对Build捕获字节，include子文件仅增加注释也拒绝身份变化。仅manifest声明source列规范化路径；typed结果contract仍v1，原生协议仍2。

## 独立期望与失败到通过

| 行为 | 实际red | 当前结果 |
| --- | --- | --- |
| 系统来源meta | metadata-source两个来源场景失配 | 系统fallback及Integer cell后4/4通过 |
| 词法stack | 查询无继承值，源配置泄漏loaded entries | 文件局部预处理及副本后通过；含child最终11 entries |
| 非法stack | 原先合法且无对应ParserError | 缺键pop/未配对字符串在预期行拒绝 |
| metadata NULL完整golden | NULL错误规范为字符串空值 | Full normalization保留null类型和值 |
| include身份 | 原格式不能绑定child字节 | schema 2捕获源图及故意改child注释的detector通过 |
| pad metadata | 用户metadata丢失，查询及Full字段失配 | entry/pad和posting/balance复制后通过 |
| 系统容差字典 | entry/any静默返回NULL | EvaluateTyped接口负例明确unsupported |
| 新诊断修复指引 | 全量Go三项coverage缺失报错 | 双语指导、manifest与真实触发fixture后全量通过 |

NULL/空/false的any_meta优先级在本轮新增oracle案例中原先就通过，属于新增覆盖，未冒称新修复。child加入后Full旧golden的9 entries已失去输入身份，显式从固定oracle重新生成11 entries；不为适配陈旧期望删除include。首次Full新案例golden尚未生成时的no-tests-to-run不计通过，随后实际执行red/green。

## 当前实际检查

- `go test -count=1 ./internal/query ./internal/compat ./internal/snapshot`：metadata范围通过。
- `go test -count=1 ./...`：首次发现缺修复指引；修正后重新执行全量通过，含CLI/Web/authoring/import调用方。
- `go test -race -count=1 ./internal/query ./internal/compat ./internal/ledger ./internal/report ./internal/snapshot ./internal/desktop ./internal/bridge`：七包通过；新增指导另跑repairguidance race。
- `make fmt vet license parity-full query-parity query-oracle-check`：通过，50/50普通会计schema2，AVERAGE显式/全局扩展另用具名数学测试，63/63query及固定reference只读复验。指导修正后fmt/vet/license再次通过。
- `make mac-test`：真实Contract、受控State、真实NativeClient实际执行；`make app`：arm64、target13.0、Info.plist及ad-hoc开发包。新指导属于bridge依赖，补齐后重新构建/执行，使用最终archive验证私人探针。

| query corpus | 场景数 | 范围 |
| --- | --- | --- |
| default-source | 22 | 保留前批标量 |
| compound-values | 28 | 保留完整复合值 |
| price-weight | 2 | 保留标准报价weight |
| inventory-order | 1 | 保留内部插入次序反例 |
| metadata-source | 4 | 系统scope、NULL/空/false |
| metadata-stack | 3 | 词法嵌套、Amount、child独立scope和来源 |
| metadata-pad | 3 | 生成用户metadata/来源/不存在容差 |
| 合计 | 63 | 严格列/值/重数/顺序及错误对照 |

## 只读私人验证

mac-test完成后复用本轮最终C archive，用本机工具链重新编译PrivateReadProbe，再执行offline differential同源检查。未重复整套mac-test作为私人入口前置。环境变量为已授权外部ledger和本地probe；不复制源文件到仓库。

47文件，双方2831指令，0错误/valid=true；missing/extra entries与changed inventory accounts均0。Expenses73行/Income22行一致；same_input_capture/native_checked/native_contract=true；3个持仓cutoff、48项价格可用性检查。没有新的typed query私人对照。账本无AVERAGE及源price；扩展数学和有价行为由公开语料独立证明，48项不能当48条真实市价。

## 文档产物、下一步及限制

设计新增第28节数据层、词法/生成来源/输入身份规则；计划第16节分8包明确责任及退出信号；手册第36节给出下一动态比较切片、loader长尾、签名/精度/预设、检查和交接命令。24顶层任务与M0–M4范围保留，matrix/roadmap/progress同步入口及63场景。旧工程日估算保留为预算参考，无工时采样不宣称精确剩余工期。当前文档检查覆盖10文件、93个本地链接/锚点、19段shell语法、24个唯一任务ID及依赖、34项公开文件SHA；全部通过，git diff --check通过。

下一项是动态object比较，之后完整NULL、unary/date/aggregate、balance/sum(inventory)、一般Decimal精度及可执行预设。重复include/metadata、reserved键覆盖、同日跨文件顺序尚未验完整规则。非字符串未闭合stack的上游异常长尾未登记普通兼容golden。

没有Q02查询C路由/Swift DTO/工作台、查询预算/取消/性能验收，也没有X01/X02或新GUI。旧捕获失败保留历史，不据此断言当前环境不可用。N04/R01完整窗口/字体/键盘/AX/安装循环，R02远端run、R03目标实机/DeveloperID/公证仍缺。没有修改私人账本、关闭用户应用、提交、PR或发布；完整开发目标继续。

## 本轮公开文件身份

仅记录公开源码/固定输入，不记录私人路径列表、源内容或私人哈希。

| 文件 | SHA256 |
| --- | --- |
| `internal/query/query.go` | `e2d8a2954af3a9b2ee97fe56950b1c9be685bdb40b02710d6538ce88d36646bf` |
| `internal/query/typed.go` | `10eed013988674578c94cab62f41b374381b4328e78f411ad90e07766af4ee36` |
| `internal/query/typed_test.go` | `445158249e730c912690403250f2902705ac1096ca23149c10df8002481749f8` |
| `internal/ledger/metadata.go` | `c2acb3b5fbcc191a01e32e0d53dced4ab1905f7b60cc6b8cf9eff8fe3faa5342` |
| `internal/ledger/evaluator.go` | `a734496bcbd061e981b8be6623e80aab0846947aa48cb4495616f7e9f8254037` |
| `internal/ledger/semantics.go` | `7aa990730c339413cbcc110f0d980100f5a660467d420faff5a02cf51268218f` |
| `internal/compat/query.go` | `5beeffd91485535cba3c1dba964b68de6d3a6ed2d962e7707d1df1e5a9219d3e` |
| `internal/compat/query_test.go` | `512e007788d4fc7831462ec5fb1a96d31c397c0a14eba8d05001a0c7765ae6aa` |
| `internal/compat/full.go` | `aaa3b0905b58dcef600f2660567fd68d22de3753dc72a748cda7671d235a5496` |
| `internal/snapshot/metadata_v3_test.go` | `9985d26ce5a13eadf246ffcde331baca808013698a2466230312a5bdb222a9d0` |
| `internal/diagnostic/diagnostic.go` | `a1fc6cfb0dc2d85c6a4254eb65a730e412176ca4cf18f932c13c101b52da55d9` |
| `internal/repairguidance/repairguidance.go` | `524070db5368a632a9ba201947631bb68e01abdbfff845c9da7a7740874407f6` |
| `internal/repairguidance/fixture_manifest_test.go` | `d81a88f279e0660b83825e9ae9d346288537b5695663b294056ebba528c46915` |
| `internal/repairguidance/diagnostic_fixtures_test.go` | `9e6607a700ab8aff25a3b0459643c097aa14150991ff5023d96400ea931231bf` |
| `tools/reference/generate_query_golden.py` | `aa4ca1e292ad50b88b69a758473b3b9c6647aaad0a5230c6d6ae0d6bedd1af41` |
| `tools/reference/generate_full_golden.py` | `9d97a8ce828a3cc0482108def1bc7689ed2d321fd0665b43312bd24842744c09` |
| `testdata/fixtures/query/metadata-source.bean` | `74a389d978ffbc5fc49887920ab3d621e68f16db9ee4865e0e9ca83a4e90814b` |
| `testdata/fixtures/query/metadata-source.queries.json` | `7aa32cceac47caa4640e8aa68bfd263b3f59a9106900d73b0897118112cbbe3e` |
| `testdata/golden/query-v1/metadata-source.json` | `97687379aeda3f4ffc8b2399d0d456b2de242716bb262368734425997f6f20ad` |
| `testdata/fixtures/query/metadata-stack.bean` | `79e45de1b371109825357bca5ea3d92de76349fd384e7b756b6ae42f7dd0e86d` |
| `testdata/fixtures/query/metadata-stack.queries.json` | `a711337b04204148908996802cbecb0fa25bdbeb8d9d72dfd1b4762bdf4d2f05` |
| `testdata/golden/query-v1/metadata-stack.json` | `94ca521bd67d22051ec5738a8871697180aefbb4c2b945fe0fad75e9b72aec28` |
| `testdata/fixtures/query/metadata-pad.bean` | `3f689afddce02faa7ba77680f5064cb8eeaca20511ec9500e9a5e8b4ce606f76` |
| `testdata/fixtures/query/metadata-pad.queries.json` | `bb14cb35b0854b41d5a5d99f85c948bfb92a40ed247450cea73dcd57e914c3da` |
| `testdata/golden/query-v1/metadata-pad.json` | `de3b73aa0bcb55eb96c9cfbeb601901005b629b8218716b9ac3f21ee745581e9` |
| `testdata/fixtures/query/metadata-stack-child.bean` | `08d6e6e217ea0bece8f2a4bb1ce053f65cf25c40d6821e623717f426233669eb` |
| `testdata/fixtures/semantics/metadata-stack.bean` | `d12434c5bb359fce10da806c795c6ba6030acd6f371906c8089b096fd950deb3` |
| `testdata/golden/v3-full/metadata-stack.golden.json` | `e71978e7799c3fda4a37cdc097217a544a9cfd568718b2b4c9ab7cc902ac15bb` |
| `testdata/fixtures/semantics/metadata-pad.bean` | `284adcfe9fe26909fa42d549f7f3ac35fb5f17e92974a5ac4011f8cedb94639e` |
| `testdata/golden/v3-full/metadata-pad.golden.json` | `f7fca8cad0cb8e5ed14419e4330fc0917dc3c037ec35398f04169a09d4bd964c` |
| `testdata/fixtures/semantics/metadata-pop-absent.bean` | `d09926623f0db5395901547a6783daedd8610e2c66f6fc0d83acd489da27a794` |
| `testdata/golden/v3-full/metadata-pop-absent.golden.json` | `d1beb8ae9c54f6b3ebf1900e65bfc361e9519c10a09e072f8cb937b1628b634b` |
| `testdata/fixtures/semantics/metadata-stack-unbalanced.bean` | `b8ad58e83616842eee36da775f7e5982e80e8f9ae0c06136a2547ff716bfae73` |
| `testdata/golden/v3-full/metadata-stack-unbalanced.golden.json` | `f1b47b612be29a65bfccf0d1d4147ebac83825e62ac1baed2c8dcc4437ed38f1` |
