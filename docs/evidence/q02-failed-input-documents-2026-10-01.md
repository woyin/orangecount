# Q02 失败输入中新附件的保护

日期：2026-10-01；基线 `9e5d794`，接续保存工作流已知缺口。

BuildResult 新增独立 DocumentDeclaration 源证据集合，在 ParseGraph 后、方言展开与会计求值之前采集成功解析的 document 声明。即使语法/会计/include 失败，没有可发布 Snapshot，仍返回声明来源文件与原 filename；getter 返回独立副本。它不暴露失败 evaluation，不替代有效财务快照，不改变诊断或账户求值口径。读取失败的部分 graph 也从已捕获字节解析声明，不额外重读文件内容。

ExportProtection 合并本次构建的声明路径、可解析的 symlink 实际路径、最新源图与上次有效 publication 的保护集合。相对附件按声明来源目录解析。捕获内容保持原响应；原生保存器在保存前接收新保护集合并按既有规则拒绝覆盖。

## 当前验证

- desktop 公开测试先实际复现“new attachment in failed build unprotected”；实现后通过，失败输入没有发布财务快照。
- snapshot 公开 Build seam 覆盖语法错误、会计 balance 错误和 missing include：有效 document 声明保留，Snapshot 为 nil；修改 getter 返回数组不能修改源证据。
- 真实 C/Swift/Store：先准备成功 CSV，再向公开临时源加入新附件声明和语法错误。重载后 state.valid=false、旧 snapshot_id 保持，保护刷新包含新附件，保存到该附件返回 protectedTarget，附件字节不变。
- 新原生等待条件首次错误地期待 ready，导致超时；按照现有 Store 的 retained-data/stale 状态修正测试，未改变产品状态标签。
- 当前 Go 全量、snapshot/desktop/bridge/query 四包 race、147 查询对照、51 会计对照及独立 AVERAGE/global booking 扩展检查 exit0。
- 最终原生四套与 app 串行 exit0；fmt/vet/license exit0；arm64/target13/ad-hoc 开发包验证通过。

## 边界和接续

本批补齐成功解析的 document 声明；损坏到不能形成 document AST 的文本不会被猜测成附件路径。只读取构建已捕获的 source bytes，不能把未能读取的文件内容当作已知声明。

继续独立分析容器解包验证器、完整多币种/compound/截断/partial/unknown 案例、preset，以及刷新到 rename 间的并发保护与完整资源/GUI门槛。旧保存工作流证据中的“失败新输入附件未覆盖”为当时状态，本批将该项推进；其他缺口保留。没有本批新私人账本或实际 GUI 测试，前批公开 GUI 证据单独引用，不能据此关闭完整24任务。

## 当前公开源码身份

| 文件 | SHA256 |
| --- | --- |
| internal/snapshot/documents.go | 2dc372c6b0e379f62f31f823277ed845d659f9464b6ced3ffabb4ffff4a2e5d5 |
| internal/snapshot/documents_test.go | aea86a198f113e57ce804f283c48e1a3a2643759eaf5e6ab55d5de3eb554085c |
| internal/snapshot/snapshot.go | 24eb6f9ab5648dd1b034f24f0fc7d6e674650a0e5f9a2e2a3a92365c6c56a4e2 |
| internal/desktop/query_export.go | d271593bad2e0d4d03cc21e29b67943c0c10c361c5c568f5dd0c617c8bc9fc70 |
| internal/desktop/query_export_test.go | a11db97a9d136f20b58d7c66ac285618a072e1fbb1d971a65b0e2f017b456f5e |
| apps/macos/tests/NativeClientTests.swift | e6253c6c6006bd35a8a7429632cff40d51e1fbc850948e2dfc4eb2c985be839f |
