# G32 流水 metadata 导出补齐

日期：2026-10-08（Asia/Singapore）。补齐交易流水 CSV 与 `.ocanalysis` 捕获中此前被忽略的交易级、分录级 typed metadata。

## 导出表示

- 流水 report schema 增加 `transaction_metadata` 与 `posting_metadata` 两列，二者均为 `list<record>`，每项包含 `key`、`type` 和 `value_json`。
- `value_json` 保留每项 metadata 的原始类型和值；decimal/amount 继续使用 exact 字段。list/map 递归序列化。
- metadata 来源位置会包含绝对路径，因此导出表示明确移除 `DirectiveMetadata.Source`；不会把源文件路径复制到 CSV 或分析归档。

## 验证

- `metadata-source.bean` 端到端捕获测试验证交易继承值与 posting 覆盖值、NULL、false、CSV 列和 `.ocanalysis` typed cells。
- 内部序列化测试验证嵌套 map 中的 amount 保留 `1.2300` exact 与 USD，同时递归移除 metadata 来源位置。
- `env GOWORK=off go test ./...`、`go vet ./...`、Go UI 全量测试/构建/vet、核心 desktop race、Go UI 全量 race、`make license` 与 `make goui-private-read` 均通过；`git diff --check` 和 `gofmt` 检查通过。

这项验证补足 G17/G32 的自动化导出表示；不替代原生 GUI、VoiceOver、真实窗口操作和导出面板实机验收。
