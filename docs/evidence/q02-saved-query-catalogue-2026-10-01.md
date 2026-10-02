# Q02 已保存query目录与草稿选择

日期：2026-10-01。基线`9e5d794`；接续查询工作台和错误位置切片。

## 当前实现

query读取响应新增query_catalogue：捕获snapshot_id、逐条id/date/name/text/source与分页。来自有效Evaluation的query指令，包括include来源；按实际entry序号与快照产生独立ID，不按名称去重，不预先编译或执行已保存文本。默认100条，最高500条，完整读取响应有16MiB保护。页尾及空目录返回真实分页，不假称所有条目均已显示。

Swift严格检查行数/分页、ID唯一性和来源；Bridge检查目录与读取快照一致。原生QueryView浏览目录、来源和分页，选择只写入草稿；Store只允许使用当前可见页且匹配当前快照的ID。旧响应/页面ID不改写当前草稿。执行仍需独立用户动作，并绑定填入后的精确文本。

本批是已有账本query指令的只读浏览，不写账本或新建query指令；编辑器authoring不在当前分析版范围。已保存文本可能不受当前typed能力支持，浏览仍保留它，实际执行按当前能力报错。

## 本批验证

- Go公开接口先因缺少目录字段编译失败；实现后目录的include源、重复名称、独立ID、分页及reload旧快照拒绝通过。
- Swift真实契约先因缺少query_catalogue模型编译失败；随后真实C读取验证来源/分页，错误行数载荷被拒绝。
- NativeClientTests检查目录只填草稿、包含unknown函数的文本不自动执行、分页后旧ID拒绝、同名不同文本分别选择、显式执行及草稿修改不重标记旧成功。
- 固定Beancount3.2.3只读加载公开fixture：6条指令，0错误。未对私人内容进行目录探测或落盘。
- 全量Go、desktop/bridge/query/compat四包race、fmt/vet/license exit0；原生三套与app串行exit0，arm64/target13/ad-hoc开发包通过。

147查询对照及51会计为前批证据，本批未改query/evaluator语义，也未重跑这些oracle或私人账本；不是本批新检查。实际GUI目录展开、来源打开、分页选择与16MiB完整边界尚未验收。

## 接续

继续固定成功执行结果的CSV/分析导出、已验preset及完整原生资源/时序边界，随后实际GUI和安装/发行验收。保存query目录已接入不代表完整Q02/X01/X02或24任务完成。

## 当前公开源码身份

| 文件 | SHA256 |
| --- | --- |
| `internal/desktop/query_catalogue.go` | `83d7748a31b79d348801fc35b6d68f4659ea8669512ebb62831087185cc53b15` |
| `internal/desktop/query_catalogue_test.go` | `552640b626621d37c73738d9263c4ce3657af3021a3ee738fbd5678c99e036f1` |
| `internal/desktop/session.go` | `9baf78515411b15398c0d9508946f3bf7a201cee6b809e2f6fe1d6121149fd0f` |
| `testdata/fixtures/native/queries.bean` | `54a10ca19b3f3e8edc73a062b36771ab8a6070a8de3c9304a13a841ada306c44` |
| `testdata/fixtures/native/queries-included.bean` | `1bd72b6e255a326f7176874f981fd6625f2299674aa602a5eb0d941ca20df69a` |
| `apps/macos/src/Models.swift` | `da45c6a48fbd2ba9ad416a10e8846920670eaed0979bbad60e1d63fb1fd3d931` |
| `apps/macos/src/BridgeClient.swift` | `139737a84e5bbb7ddc89cc1060575ccbcb17d1f881cd60b997d8d8771878e7d8` |
| `apps/macos/src/LedgerStore.swift` | `9d22c3e9eeb0d162aa30491f9b177623372ec6ccfe4af5330b77a7d0e193df11` |
| `apps/macos/src/QueryView.swift` | `0a381e983c5bae627f68d6f43bb4a5d40c4a953ce5ae0e62fc5a9b384b08e710` |
| `apps/macos/tests/ContractTests.swift` | `5a5ede0b1054390a06426399e22a0c50afb64a1ab76628f98fce39b2bb801f4c` |
| `apps/macos/tests/NativeClientTests.swift` | `e00347b878f1d528916e93397dcc5c0c72ddc071489c3b63a4105f652b41181e` |
