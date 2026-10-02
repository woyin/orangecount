# Q02 查询分析容器读取首批验证（2026-10-02）

本批新增公开 `desktop.ReadQueryAnalysis` 与独立命令 `go run ./tools/analysisverify FILE.ocanalysis`。读取不打开账本、不执行查询；命令仅输出版本、视图、行列数和截断摘要。严格接受当前生产者的四个成员与 canonical CSV；不承诺第三方 CSV 格式变体。

## 实现与实际验证

- ZIP 只允许 context.json、schema.json、cells.ndjson、data.csv，各恰好一次；限制压缩容器、单成员和合计展开大小，拒绝非普通文件，读取校验 CRC 与 UTF-8。
- JSON 拒绝重复键、未知字段、缺失必需字段、非 nullable 字段的 null、尾随数据与超过 64 层的嵌套。
- 校验查询契约、上下文身份、指纹、返回行数、截断元数据、列 ID 和位置类型。恢复 Decimal/Amount/Position/Inventory 的具体 Go 类型；完整成本日期与 nil/空标签保持区分。
- 精确数字有长度与语法限制，日期必须有效，CSV 必须与恢复值的 canonical 投影逐字节一致。
- 公开真实查询验证重复列标题、精确小数、NULL/空字符串、零行 schema、库存完整成本信息。
- 八类实际篡改验证拒绝：额外路径成员、重复成员、缺失成员、错误 CSV、行数不符、schema null、非法 integer、重复 JSON 键。

实际 red：读取函数不存在导致编译失败；实现后首次测试暴露将 semantics profile 与 evaluation revision 混用，修正为 legacy/v3 后通过。未更改生产者协议。

当前执行退出码均为 0：针对读取测试、`go test -count=1 ./...`、`go test -race -count=1 ./internal/desktop`、desktop/analysisverify vet、fmt、license、diff check。

独立命令读取上一批 GUI 的公开 `build/qa-export/query-result.ocanalysis` 实际产物：format_version=1、view=query、rows=1、columns=5、truncated=false。本批没有重新运行 GUI；此前原生与私人账本证据为继承证据，未进行新的私人读取或验证。

## 尚未验收

多币种、非终止比值、空库存、nil/空成本标签与截断的完整样本网格；真实预算边界/压缩炸弹/损坏 CRC 的专项案例；报表 partial/unknown 与分页容器；第三方兼容读取及资源性能。字段显示值仅保持生产者内容，当前没有独立验证显示舍入策略。此切片不代表整个 X02 或 24 项任务完成。
