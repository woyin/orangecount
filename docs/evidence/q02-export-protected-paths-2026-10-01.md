# Q02 固定捕获的本地目标保护信息

日期：2026-10-01；基线 `9e5d794`。接续客户端/Store 捕获动作。

有效 publication 新增源图与 document 声明的保护路径集合。附件相对声明文件所在目录解析，绝对名称保留；已有 symlink 的真实路径同时保留，缺失附件仍保护其声明目标。去重排序后与有效快照绑定，查询成功时与只读结果一起缓存。下一次查询/快照变化/关闭清除缓存。

CaptureQueryExport 新增 protected_paths，仅用于原生本地写入前检查，不加入 context/schema/cells/data.csv 四个 ZIP 成员。获取捕获时复制路径数组，随后内容序列化不依赖新快照或最新失败的 include 图。Swift 接收该字段，Bridge 要求非空、绝对路径、无重复；尚未实现文件写入或目标拒绝动作。

## 当前验证

- Go 公开临时账本/include/document 样例首次因缺少 ProtectedPaths 字段编译失败。接入后首个期望使用 `/var` 临时路径而源图已规范为 `/private/var`，测试失败；将预期目录按实际 symlink 规范化后通过，产品路径逻辑不因此改写。
- 实际成功 query 捕获包含主源、include 源、已有附件和缺失附件真实目录路径。
- 实际 ZIP 解包逐个检查，不包含临时目录字符串或 protected_paths 成员/字段。
- 真实 C/Swift 捕获包含公开 default-source 源保护路径；此前客户端/Store/迟到响应测试继续通过。
- 本批 Go 全量、desktop/bridge/query 三包 race、原生三套、fmt/vet/license exit0。
- 随后 app 构建 exit0，arm64/target13/ad-hoc 开发包验证通过；没有实际安装或旧系统运行验收。

## 接续与边界

保护集合只是保存器的输入，不能视为“已经禁止覆盖”。下一步实施 Swift 写入前规范化、目标 symlink/alias/hardlink 身份检查、同目录唯一临时文件及原子替换，并在替换前再次校验；再接保存面板、取消和错误状态。目录别名、保护文件替换/消失、保存途中 reload 与完整保护信息预算仍待验证。当前查询文本或结果字符串自身的来源路径隐私策略仍保留上一批缺口。

本批没有读取私人账本、重新跑会计/query oracle或启动 GUI；24任务保持未完成。
