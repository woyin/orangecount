# G06 旧偏好端到端迁移验证

日期：2026-10-08。本记录先验证旧偏好读取、Go 迁移和普通 bookmark 解析的隔离端到端路径，随后通过明确 opt-in 的只读测试检查当前 `org.orangecount.desktop` production defaults 域；测试日志不输出账本路径、偏好值或 bookmark bytes。

## 验证内容

- 新增 Darwin 集成测试，在唯一的 `NSUserDefaults` suite 写入旧客户端使用的最近账本数组、`org.orangecount.semantics.<path>`、`org.orangecount.reading.<path>` 和 `org.orangecount.attachmentRoots.<path>`。
- 测试账本和附件目录都位于临时目录。附件记录由 `platform.CreateDirectoryBookmark` 生成真实普通 `NSURL` bookmark，并按旧 `Codable` JSON 格式存为 `NSData`。
- 通过 `OpenLegacyDefaults` 和 `Store.MigrateLegacy` 运行生产读取/迁移实现，检查最近路径、v3 profile、USD/market-value、bookmark generation 9 和原始 bookmark bytes；再调用 `ResolveDirectoryBookmark`，确认其仍解析到原临时目录。
- 迁移后再次读取旧值，确认 profile 和 roots 原样保留，并确认未知键仍存在。随后只在测试 suite 中把旧 profile 改成 `legacy`，再次调用迁移；Go 偏好仍保留首次导入的 `v3`，证明迁移标记生效。
- 测试使用规范化账本路径构造旧 keys，与旧 Swift 客户端保存路径的规范化行为保持一致。

## 验证命令

- `env GOWORK=off CGO_ENABLED=1 go test ./internal/preferences -run '^TestLegacyDefaultsMigrationEndToEndIsReadOnly$' -count=1 -v` 通过。
- `env GOWORK=off CGO_ENABLED=1 go test -race -count=1 ./internal/preferences ./internal/platform` 通过。
- `make goui-check` 通过，包含 Go UI 全包测试和 `CGO_ENABLED=0` 生产构建。
- `env GOWORK=off CGO_ENABLED=0 go vet ./...` 通过。
- `env GOWORK=off CGO_ENABLED=1 ORANGECOUNT_TEST_REAL_LEGACY_DEFAULTS=1 go test ./internal/preferences -run '^TestRealLegacyProductionDefaultsMigrationIsReadOnly$' -count=1 -v` 通过。当前 domain 有3条最近账本记录和3条可导入账本偏好；3条现存账本均验证了 profile/reading 导入，迁移前后最近路径、已知 profile、reading 与 bookmark 数据快照相同。当前3条记录没有旧附件目录配置。

## 尚未验收

生产 defaults 中没有附件 bookmark 可用于真实数据解析，因此 bookmark 数据迁移通过隔离 suite 内真实临时 `NSURL` bookmark 验证；没有读取账本内容。Go UI 首次启动后从迁移结果恢复附件策略的整套产品启动流程、G12迁移备份导出仍待后续验证。普通 bookmark 不提供 security-scoped access。G06 验收完成；G12 继续进行中。
