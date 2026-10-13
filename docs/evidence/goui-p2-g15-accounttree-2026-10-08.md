# G15 可复用账户树模块首轮记录

日期：2026-10-08。此记录完成账户树数据/交互投影的一部分；G15 继续进行中。

## 变更

- 新增 `apps/goui/internal/widgets.AccountTree`，以核心 `AccountTreeNode` 投影建立完整路径稳定键、可见根节点、可见子节点索引和节点查询。
- 搜索会保留匹配节点的祖先，只裁剪可见子节点；节点和父级金额直接使用核心 DTO，模块不重算财务汇总。币种列从未过滤投影计算，搜索不会丢失仍属该报表的列。
- 全展开使用原始完整树，因此筛选期间展开后清除筛选仍可见完整展开；全折叠清空 MyGo `OutlineState`。
- `statementPage` 已从分散的过滤、索引和展开函数切换到该模块；树的键仍是完整账户路径。
- 新增 `widgets.AmountCell` 与精确文本格式化：可读显示使用核心提供的 Display/Approximate，辅助名称和提示使用 Exact，右键复制走 MyGo clipboard 并复制 exact+currency，不从 float 重建数值。资产负债表/损益表金额格和汇总、账户余额文字、持仓 exact 金额已接入。
- 新增 `widgets.SourceRow` 统一展示来源路径/行/列及复制、Finder 显示、打开动作；应用层仍负责剪贴板、Finder 和校验账本/快照身份后打开。诊断页与关联诊断已切换到该控件，其他页面的来源格式也复用 `SourceText`。
- 新增 `widgets.StableTable[T]`：校验唯一且显式的行/列 ID、重复标题可用不同 ID、排序必须有业务比较器，并把 MyGo 虚拟表格的选择、焦点和滚动锚点绑定到稳定行 ID。商品目录用数值/词典序比较器；持仓分组数量、成本、均价和损益按 exact rational 排序，排序后仍选中同一商品/分组。
- `widgets.CompareAmounts` 和 `CompareOptionalAmounts` 以核心 canonical exact 值调用 `ledger.ParseDecimal`，不读取近似显示或转成浮点；不可用金额在升序下排在有效金额后。

## 验证

- `TestAccountTreeFilterKeepsAncestorsAndCoreTotals` 检查稳定路径、祖先保留、金额精确值未变、隐藏兄弟节点和未过滤币种列。
- `TestAccountTreeNoMatchAndExpansionUseStableFullPaths` 检查空结果与完整树展开/折叠。
- `TestAmountCellKeepsExactValueAccessibleWithoutChangingDisplay` 检查近似显示、exact accessible name、右键菜单与复制内容。
- `TestSourceTextKeepsStableLocationFormat`、`TestSourceRowActionsAndDisabledOpen` 和 `TestSourceRowEmptyLocationHasNoActions` 覆盖来源格式、动作、禁用打开和空路径占位。
- `TestStableTableSortsTypedValuesAndKeepsStableIDs`、`TestStableTableRejectsAmbiguousIdentityAndSchema` 和 `TestStableTableSortHeaderUsesPersistentTypedSort` 覆盖数值排序、重复标题/列 ID规则、稳定选中和原生表头排序交互；商品目录 headless 测试覆盖排序后选择保持。
- `TestCompareAmountsUsesExactRationals` 覆盖非终止分数、负数和缺失金额排序。
- 资产负债表 headless Session 测试继续检查搜索结果、无匹配时摘要保留、清除搜索恢复全部行以及树状态行为。
- `make goui-check` 通过：Go UI 全模块测试与 Darwin arm64 构建通过。
- `cd apps/goui && GOWORK=off CGO_ENABLED=1 go test -race ./internal/widgets ./internal/app` 通过。

## 未完成

VoiceOver、真实键盘/排序操作、长中文及多币种列实机验收仍待完成；所以 G15 未关闭。
