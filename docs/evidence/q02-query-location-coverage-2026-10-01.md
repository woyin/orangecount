# Q02 操作符、来源、别名与预算错误范围

日期：2026-10-01。接续[错误位置首批](q02-query-locations-2026-10-01.md)。

二元/一元操作符保存实际token范围；FROM保存源token范围；歧义别名引用使用引用处范围。递归错误仍优先保留最内层范围。运行时非法正则保留原有syntax分类并定位实际匹配操作符；预算错误不借用外层函数/操作符的范围。

## 本批验证

- 新操作符例子先因range:null失败，新增11个位置例子全部通过，包括嵌套字段、非法日期、FROM、别名及运行时正则。
- 手写别名引用预期最初少一个字节，核对后修正为[31,32)。曾用除零作为错误探针，但当前查询返回NULL；未改变该行为，运行时错误改用实际非法正则。
- 深表达式预算反例先暴露外层函数被错误附加范围；修正后普通检查通过。首次race因5秒deadline先于2M steps而失败，随后使用公开ExecuteTyped的300 steps检查work-budget范围为空，原生会话使用快速numeric-budget检查范围为空；没有增加生产timeout或放宽预算。
- 最终四包race（query/desktop/bridge/compat）exit0；全量Go、fmt/vet/license、147查询及11语料oracle身份检查本批通过。额外预算测试随后通过race；产品源码未再改变。
- 原生三套与app串行exit0，arm64/target13/ad-hoc开发包通过。未启动GUI、未运行私人账本或会计oracle，这些范围不冒充新验收。

完整查询语义、保存query/preset、固定执行CSV/分析导出、跨语言资源边界与GUI仍未完成。下一步接保存query目录和导出；完整24任务继续。

## 当前公开源码身份

| 文件 | SHA256 |
| --- | --- |
| `internal/query/query.go` | `851b880e91741158625ea33783105489462d05413bd86d325c3e97904cf160d7` |
| `internal/query/typed.go` | `18ac51b147af09a5aac03dae86d7004f9dea8e1a368dc32a6b2d153fbfd15403` |
| `internal/query/location.go` | `e6243b14bb80ac00ebfcf8044acbaa6d4a2060eda7ca432c5df3705520b188be` |
| `internal/query/execution_test.go` | `03b331651e7fe20d0cbf5cf0cf8695537e29839105f60ef9b7ab3cb51cf57a4a` |
| `internal/desktop/query_test.go` | `5f214873c9a5e0bdb85c53e7b4638879ed409c5a4d3a727f70fbb0f6a700ba0a` |
