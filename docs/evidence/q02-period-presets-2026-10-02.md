# Q02 显式期间预设（2026-10-02）

四个内置预设现在提供“整个快照”与“按阅读期间”两条草稿入口。默认预设文本保持不变；期间入口由当前发布快照解析 request.window/date_from/date_to，返回 concrete date_from/date_to 和已经生成的完整 BQL。它只填草稿，不改已有执行、不改共享筛选、不自动执行。

## 契约

- query catalogue 新增 nullable period，含 date_from/date_to/presets；无有效日期边界时不提供。日期来自同一发布快照，最近一月使用现有 dateRange 规则，而非 UI 当前系统月份。
- `query.PresetsForPeriod` 仅处理四个固定模板，不重写用户编辑文本；边界校验为固定 YYYY-MM-DD、合法日期、顺序正确且至少一端存在。
- 日期首尾包含；现有 WHERE 使用 AND，无 WHERE 增加 WHERE，均置于 GROUP/ORDER 前；不会把月收支的 account 条件覆盖。
- native 菜单显示具体日期范围。选择捕获的期间文本，编辑器里直接可见 date >= / <=；既有成功执行身份和文本保持不变。
- Swift decoder 校验日期、顺序、边界非空、与完整预设相同的有序 ID、文本预算及名称。旧无 presets/period 的 payload 仍可解码为空目录，协议仍为 2。

## 当前实际验证

- red：缺少 PresetsForPeriod，新增公开 API 后 targeted green。
- 固定 oracle 新增 12 个期间案例（四预设 × 双边界/仅起始/仅截至），生产生成文本与审阅输入一致性测试。12 corpus / 163 查询差分通过。
- CLI typed-json 实际执行 12 个期间查询；按月收支仅返回一月或二月对应行。
- desktop 从 1m 捕获 2000-02-01 至 2000-02-21，实际执行期间预设，返回二月一行；原请求日期字段保持空值，表示没有隐式改写请求。
- 真实 Swift/C 验证整个快照预设选择不执行、显式执行两个月；选择 1m 期间只改可见草稿且保留前次 request_id；显式执行后得到二月一行。
- malformed Swift/C 负例拒绝非法日期、逆序、无边界、ID 不符及超预算文本。
- 首次 native build 的测试代码把 QueryCell 当成可比较 QueryValue 导致编译失败；修正为匹配 Cell.value，未改变产品实现。

本批最终 Go 全量、query/desktop/bridge/documents race、fmt/vet/license、oracle-check、163 查询差分均退出 0。四套 mac-test 已通过；app 最终构建结果见后续补充。没有重新读私人账本、没有在用户已介入的 QA 页继续操作；期间菜单 GUI 尚未验收。它不能代替完整 Q01/Q02、报表容器、附件页面、全指令浏览或发行验收。

最终 app 构建退出 0：arm64、部署目标 13.0、开发 ad-hoc 签名与 bundle 检查通过。没有声称旧系统运行或正式发行签名已验证。
