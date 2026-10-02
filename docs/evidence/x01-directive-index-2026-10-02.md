# X01 全指令与配置索引首批（2026-10-02）

新增有效 Snapshot 的只读 `DirectiveCatalogue`。源索引在 dialect 展开前捕获配置与 source kind；日期记录直接使用成功 Evaluation 的有效记录和顺序。没有修改 parser、metadata stack 或 booking，也没有从 failed build 的半解析配置拼接有效报表。

## 当前公开验证

公开 `testdata/fixtures/directives/all.bean` / included.bean 覆盖解析器 20 种 source kind：option/plugin/include、tag/meta 栈、open/close/commodity/balance/pad/event/query/price/document/note/custom/transaction 与独立 dialect 行。插件只展示配置，保留既有 warning，未执行 Python。

- configuration 保留词法次序，在 include 位置进入子文件；重复 include 声明两条都显示，子文件与 evaluator 一样只加载一次；配置没有伪造日期。
- dated 保留有效日期排序、typed directive、有效交易 metadata/tag/link 与完整数字；pad 合成交易有 Generated、SourceKind=pad、OriginID 指向 pad 声明。独立 dialect 编译交易保留 SourceKind=dialect 与生成关联。
- unique source ID 由路径/kind/raw 内容构造，插入无关行后保持。完全相同的重复 note 保留重数及不同 ID，并绑定当前 snapshot，避免跨版本按下标恢复错误记录。生成 ID 绑定 snapshot，不假装具有稳定内容身份。
- catalogue getter 深复制 typed AST；实际修改 exact big.Rat、custom amount、transaction posting、metadata、tags、叙述和配置，不改变随后读取的 Snapshot。
- 实际失败重载（新 title + 非法指令）只更新诊断；旧 snapshot/configuration/dated 保留，未混入 unpublished title。

实际 red：公开 catalogue 与记录类型不存在，新增索引后 targeted green。全量 Go、snapshot/desktop/bridge/query/documents race、fmt/vet/license、163 查询差分、51 会计差分与两项 AVERAGE 扩展测试退出 0。新增深复制测试 targeted race 退出 0。四套 mac-test 已通过，最终 app 结果后续补充。

## 尚未完成

这是源/有效记录索引，还没有 journal transaction/all/config 的分页 DTO、C/Swift 模型或原生模式；现有交易流水计数与页面保持原口径。配置过滤、模式切换、分页、真实 GUI、边界预算与大型账本性能仍需验收。当前公共 getter 返回完整复制，后续分页必须在只读索引上先选页再复制，不能为了显示少量记录每次复制全部 AST。

独立 dialect 行关联已验，block dialect 的多腿来源图尚未完整实现；include glob 与额外加载规则不由此索引新增支持。完整每类 JSON payload/metadata union 必须独立定义，不能让 Swift 解析 RawText。私人 FinanceBook 没有本批重新读取；旧会计/原生私人证据是继承证据。

最后串行 mac-test/app 全部退出 0：四套原生测试与 arm64/target 13.0/ad-hoc 开发 bundle 检查通过。最新 app 包含此 Snapshot 索引与期间预设代码；没有在用户介入后的 QA 页面重新启动、退出或覆盖其正在使用的实例。
