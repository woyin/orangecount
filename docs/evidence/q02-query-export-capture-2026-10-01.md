# Q02 会话固定导出与 C/Swift 内容契约

日期：2026-10-01，基线 `9e5d794`。接续 CSV 与分析容器内容生成。

Session 保留一份当前成功执行的独立只读副本，复制列、行、库存、成本及 nullable 日期/标签，不与返回给调用方的响应共享可变结构。下一次有效查询启动、快照发布变化和关闭会话清除缓存；旧成功不作为后续失败执行的结果。最终发布在锁内校验请求代次、取消高水位和快照，防止被更新请求替代的结果重新成为导出对象。

`CaptureQueryExport` 校验 snapshot/request/fingerprint 和 csv/analysis 格式，返回冻结字节及原执行上下文。格式生成仍由 Go 完成；Swift 不从显示字符串重建数值。单次内容输出限制 16 MiB，超限返回现有结果预算错误；C 响应的 base64 编码有额外传输体积，16 MiB 是解码文件内容上限。捕获只在成功对象仍为当前对象时发起，已捕获字节独立于后续 Session 状态，供后续保存面板使用。

新增 `OC_CaptureQueryExport` 和 Swift request/capture 模型、BridgeClient 方法；Swift 检查返回身份、格式和解码内容预算。协议仍为 2，不改变既有 query contract。

## 当前验证

- desktop 公开测试先因缺少请求/捕获接口编译失败；实现后通过。
- 实际带成本库存执行，调用方修改列标题、库存成本数值、标签和单元格后，导出仍匹配修改前 CSV。
- 错误 fingerprint 拒绝；后续实际 unknown-function 失败之后，旧代次不能再次捕获，已捕获内容保持一致。
- 真实 C/Swift：CSV 重复标题和执行文本、ZIP 文件头及 base64/Data 内容传输通过；失败后旧导出请求收到 invalid-request。
- 当前全量 Go、desktop/bridge/query 三包 race、原生三套测试、fmt/vet/license exit0；加强嵌套成本变更测试后 desktop race 重跑 exit0。
- 随后 app 构建 exit0，开发包 arm64、deployment target 13、ad-hoc 签名检查通过；不是旧系统运行或安装验收。

## 保留范围

尚未接 NativeLedgerClient/Store 动作与保存面板。内容捕获接口不写目标文件；原子替换、源图/附件与 alias/hardlink 目标保护、保存取消、实际 reload/close 期间保存、多次同时捕获预算及 GUI 验收仍缺。路径隐私、独立容器解包验证器、多币种/截断等完整格式实例沿用上一批未验清单。没有本批新私人账本或会计/query oracle证据，完整 Q02/X02 和24任务不关闭。
