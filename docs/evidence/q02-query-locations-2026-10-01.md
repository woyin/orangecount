# Q02 查询错误位置与原生选择首批

日期：2026-10-01；提交基线`9e5d794`，接续查询工作台首批。

## 当前实现与边界

词法token保存原始UTF-8 start/end字节偏移。standard parser的ParseError保留失败token范围，EOF为[len,len)；已消费的非法日期/操作符/参数使用原token，不错误定位到后一token。legacy Parse错误文字与语义保持原行为。

函数名与字段节点保留源token范围；compileTyped递归错误保留最内层已知范围，所以嵌套错误不会被外层覆盖，也不会搜索字符串中的同名内容。TypedError新增可选range与实际phase，通过desktop QueryFailure传到现有required-nullable协议；不变更财务schema或会计profile。

Swift转换UTF-8字节范围为NSTextView所需UTF-16 NSRange，并拒绝切入UTF-8字符中间的范围。QueryView改用原生纯文本NSTextView，禁用智能引号/连字符等替换；仅当草稿仍等于捕获执行文本时选择对应错误位置，编辑后的草稿不套用旧范围。

本批范围是语法失败、函数/字段节点及其签名错误的定位首批。二元/一元操作符、FROM源、别名歧义及某些全局/运行时错误还没有覆盖完整定位链；没有可靠位置的错误继续range:null，不猜测。实际视觉选择、键盘焦点及编辑器窗口布局仍须GUI验收，不能用NSRange转换测试代替。

## 验证

- Go公开会话测试先失败：unknown-function缺range；手写emoji前缀预期最初少算一个字节，核对后修正为[26,36)。实现无字符串搜索。
- 六个实际会话例子通过：emoji与字符串里的同名函数、中文后语法token、EOF、嵌套missing字段、year(string)签名、非法日期；均核对实际字节范围。
- Swift先因缺少nativeRange编译失败；随后真实C/Swift将[26,36)转为UTF-16 [24,34)，并拒绝emoji中间的[9,12)。nullable range缺键仍拒绝，取消错误仍可为null。
- `go test -count=1 ./...`、query/desktop/bridge/compat四包race、`make fmt vet license query-parity query-oracle-check`均exit0；147/147查询、11语料oracle身份通过。本批额外三个位置例子随后经targeted会话测试通过，产品源码未再改变。
- 最终串行`make mac-test`、`make app` exit0：真实契约/状态/客户端通过，arm64/target13/ad-hoc开发包通过。未启动GUI、未跑私人对照或会计oracle；这些继承上批证据。

## 接续

继续操作符/源/别名等完整错误位置覆盖，再接保存query浏览、已验preset、固定执行CSV/分析导出及真实预算边界；最终执行公开GUI和安装/发行门槛。完整Q01/Q02及24任务不关闭。

## 公开源码身份

| 文件 | SHA256 |
| --- | --- |
| `internal/query/query.go` | `d7053a3372c110a64cf860274b475cddc2c4d7a2d2c7b24eb815eb120a71357d` |
| `internal/query/typed.go` | `fff118457282168dab0d8db21e703946238fcfe10403856f0f33a261648ada3c` |
| `internal/query/execution.go` | `81a4c56fcdd73032a39bad1528145a5809507d091ee6fca32d4f9dcb1cec11e8` |
| `internal/query/location.go` | `2a17bc1590c62156ad0073495a85e2d4badd1549fd8586153fd3fa0a6724aaf0` |
| `internal/desktop/query.go` | `05fcb235b2c234d954a9b7a1425d41804964a6f20de3ce6b018db741bdd4cab5` |
| `internal/desktop/query_test.go` | `2e3f824762ea1a42dc9b164e6c2f3b68f239d92bd5b9988d8bb398f658571752` |
| `apps/macos/src/Models.swift` | `08101d369b82b9ab9ada0e6eb666d934e24bbb4d3600b2961e250e91a29fc683` |
| `apps/macos/src/QueryView.swift` | `3a2db700b17d17bdd5eb3939856f2f0bae8d820fdcea2a76ebe005698fc8b271` |
| `apps/macos/tests/ContractTests.swift` | `8089a9fabd1e28a68591bec790c1dfedeb0bd96e5fa8ac3d61f9ad9965f94dc2` |
