# 安全会计源导出

`orangecount export` 将原账本转换为供外部 Beancount 工具读取的源快照。原账本仍是唯一编辑源；报表/查询分析文件不受本页的 AVERAGE 导出限制。运行不调用 Python 或 plugin。

```sh
# out 的父目录必须存在，并位于原账本入口目录之外。
mkdir -p /path/to/export-parent
bin/orangecount export -semantics v3 \
  -out /path/to/export-parent/book.bean /path/to/ledger/main.bean

# 有 include 的账本：输出到尚不存在的新目录。
bin/orangecount export -semantics v3 \
  -out /path/to/export-parent/book /path/to/ledger/main.bean
```

默认会计配置仍为 legacy。若 legacy 与 v3 的完整交易/库存数学不同，export 拒绝，先用 `compare-semantics` 评估迁移，再显式选择 v3；不会为导出偷偷切换账本配置。所有目标必须在入口目录外，含同目录的另一个文件。旧 fixpoint 测试保留转换内容，测试目标迁至不同目录以符合保护策略。

兼容门基于有效 booking，拒绝账户、全局默认或继承的 AVERAGE；拒绝所有 Python plugin、未登记 option、list/map metadata、数字表达式以及超过首批数字范围的源。首批 option 为 title、operating_currency、booking_method、inferred_tolerance_default、infer_tolerance_from_cost、tolerance_multiplier；数字仅有限十进制字面量，最多28个有效系数位且源拼写最多128字符。FIFO/LIFO、STRICT/SIZE、HIFO/NONE、总成本/总价格、pad、metadata 与容差只在上述子集内开放；这不是任意 Python 扩展兼容声明。

include 必须在入口目录树内，相对路径映射可保持；绝对 include、外部 include、重复 include、symlink include 和 glob 拒绝。Go 当前不加载 glob，不能声称冻结了其匹配结果。posting 的 filename/lineno 保留键显式拒绝，避免将 Go 尚未对齐的无效源导出为有效源。

document 必须为单个已有普通文件。导出将引用改为原文件的规范绝对路径；不复制附件，因此这是本机源快照而非独立可搬移的附件归档，且源快照会包含这些绝对路径。原生分析文件仍用其独立的脱敏格式。

`Prepare` 冻结字节、入口及源/附件保护；`Write` 在发布前再读源图核对输入身份，并在临时树重载 Go，逐交易的全部 posting、完整成本身份、账户存量、价格和 options 必须一致。未找到语义等价时，目标不变。`dialectize` 使用相同保护/事务，重新加载所选配置验证；export 的 AVERAGE/plugin 兼容门不限制逆转换。若逆转换会改变 posting 顺序等捕获含义，明确拒绝。

单文件可替换普通现有文件；symlink、hardlink、源文件与声明附件（包括保存的旧 inode）不得替换。临时文件完成 write/fsync/close；失败清理。多文件仅接受新目录，全部文件和目录同步、重载后一次发布，现有目录不逐个覆盖。

替换现有单文件不删除旧 inode：先把旧文件整体改名为 `<目标>.orangecount.bak`（与账本写入相同的备份约定），再把暂存文件排他 link 到目标名；若 link 失败则把备份改名还原。因此「目标处原来是什么」不需要识别——Finder alias、其他工具标注的文件或任何内容，都随旧 inode 完整保留在备份名下。这条设计取消了按元数据判断的必要性，也消除了不同构建之间的行为差异。

发布只用标准库，任何 `CGO_ENABLED` 设置、任何平台行为一致：`mkdir`/`link` 提供「创建或失败」，`rename(2)` 对空目录单步生效，因此整棵树仍是一次可见，不暴露部分树。与排他 rename 的唯一差别是存在一个极短窗口，其中目标目录已存在但为空；在该窗口内崩溃会留下空目录，下次导出把它报告为「已存在目标」而不是覆盖。

| 场景 | 行为 |
| --- | --- |
| 发布新的 include 树 | `Lstat` 拒绝已存在目标，`rename(2)` 整树一次就位 |
| 替换已存在的单文件 | 旧文件移到 `<目标>.orangecount.bak`，新文件排他 link 就位；失败则还原 |
| 目标为 symlink、hardlink 或非普通文件 | 拒绝，原名与内容不变 |
| 备份名已被 symlink 或非普通文件占用 | 拒绝，不覆盖该名 |
| 目标处是 Finder alias | 与普通文件相同：alias 文件整体移到备份名，文件内容与 alias 记录完整保留 |

目标身份检查不是对具有同等系统权限的恶意并发进程的完整安全隔离；未承诺断电后的目录条目持久性。

开发验收：`make source-export-oracle`，固定 Beancount3.2.3，公开输入；运行时无需该工具。具体本次证据见 [F04](evidence/f04-source-export-2026-10-02.md)。
