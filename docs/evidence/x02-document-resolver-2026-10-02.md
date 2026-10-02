# X02 附件路径模块首批（2026-10-02）

新增 `internal/documents`，独立于 ledger booking。调用者必须提供现存绝对目录的允许根；构造时规范化 symlink 并持有固定目录范围。相对声明基于 sourcePath 所在目录，不基于 cwd。模块不打开默认应用、不读取附件内容、不改账本合法性。

## 公开实际测试

- 相对/绝对文件、根内 symlink：available，重新校验返回规范路径。
- 缺失文件：missing；目录：unsupported-kind；绝对根外路径、根外 symlink、根外目录 symlink 下缺失子文件：outside-root。
- 指向根外不存在文件的最终 dangling symlink 曾实际 red，被误判 missing；实现显式追踪最终 symlink，修复后 outside-root。
- symlink loop：unreadable；路径解析递归限 256，避免恶意不存在路径与链无限递归。
- 文件 inode 被替换、symlink 改向、原地写不同内容长度、文件删除：Revalidate 拒绝，返回可恢复 ErrChanged/ErrUnavailable。
- 无根、相对根、文件充当根、缺失根：构造拒绝；只有显式增加根才能放行根外文件。
- 根路径前缀相似的 sibling 不算根内；另一 Resolver 的引用不能作为授权；改公开 presentation Path 不改变私有捕获目标。

实际 red：新包没有实现；实现后 targeted green。随后 dangling symlink 专项 red→green。全量 Go、documents race/vet、fmt/license/diff 检查退出 0；不需要新的会计 oracle，因为未改变求值且模块尚未接入会话。

## 限制与后续

这只是独立 resolver，不是完整附件功能。尚须快照绑定的账户/日期/声明来源 DTO、显式 roots 配置、C/Swift 契约、列表、打开/Finder 动作及 GUI。没有把“目录可解析”当成会计源 include 授权。

身份检查使用 canonical path、os.SameFile、大小与修改时间。没有内容哈希，不能检测保留 inode/长度/mtime 的刻意改写；检查之后与系统默认应用打开之间仍有文件系统 race。Finder alias 的额外目标解析、权限撤销/不可读专项案例与真实系统打开也未验收。用户控制页面后隔离 QA 保留；此切片没有 GUI 输入或私人账本读取。

后续公开权限案例实际执行且未跳过：chmod 000 的普通附件被判 unreadable；先 available 后撤销权限，Revalidate 返回 ErrUnavailable。独立 permission 测试与 documents race 退出 0。其余权限/系统交互边界仍未泛化为完成。
