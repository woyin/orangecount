# Q02 原生原子文件保存器首批

日期：2026-10-01，基线 `9e5d794`，接续保护路径捕获。

新增独立 Foundation/Darwin 模块 AnalysisFileExport。初始化固定内容和保护路径，并保留当时存在的保护文件 device/inode。保存目标先规范化父目录，拒绝保护路径/保护 inode、非普通文件和 Finder alias 文件；最终文件 symlink 不允许作为目标。普通分析文件覆盖通过同目录唯一临时文件、完整写入、fsync、关闭、重新校验父目录身份/原选择 symlink 解析/目标保护，再 renameat 替换完成。临时创建和替换锚定打开的目录 descriptor；失败清理临时文件。

当前模块不依赖 Store，不重新执行 query，也未接保存面板。它不向账本写入语义更改；普通目标的新文件权限为 0600。保存器只使用传入的保护集合，不自动获取重载后新加入的源图。

## 当前验证

- 独立公开文件保存 seam 首次因缺少 AnalysisFileExport/错误类型编译失败；实现后通过。Darwin.stat 在 Swift 名称解析中与 stat 结构冲突导致编译失败，改为 fstatat 后正常。
- 普通已有目标实际替换，读取产物与捕获字节完全一致。
- 源文件、最终 symlink、hardlink 均拒绝，源内容保持不变。
- 源移动后保留 inode 继续拒绝其 hardlink；缺失附件声明路径也拒绝新建覆盖。
- 失效目标目录返回错误；检查无残留临时文件。
- 独立 ExportTests 进程通过 RLIMIT_FSIZE=1 和临时忽略 SIGXFSZ，实际临时文件部分写入后触发 EFBIG；返回准确错误，已有目标完整保留，临时文件清理，之后恢复进程 limit/signal。没有对用户进程或私人文件施加该设置。
- 新 ExportTests 已加入 mac-test 门槛；首轮含保存器的原生四套 exit0。补充部分写失败后独立测试 exit0；fmt/vet/license exit0。
- 最终含部分写失败的原生四套再次 exit0；随后 app 构建 exit0，arm64/target13/ad-hoc 开发包验证通过，未进行实际 GUI/旧系统验收。

## 接续和限制

继续保存面板、捕获/保护身份固定时机、当前账本源图变化的保护更新、覆盖确认/取消和错误状态。Finder alias 拒绝分支、目录 symlink 在写入中变化、覆盖 rename 失败、close/fsync 失败、实际 GUI 与文件保存途中 reload 尚未注入验证。校验和 rename 是独立系统调用，不声称防御恶意并发文件系统修改；掉电后目录持久性亦未验证。目标 inode 在 Go 捕获与保存器初始化之间的变化还需跨层覆盖。

本批没有 Go 产品修改，Go 全量/race沿用前批；没有新私人账本、oracle、实际 GUI 或安装证据。完整24任务未完成。
