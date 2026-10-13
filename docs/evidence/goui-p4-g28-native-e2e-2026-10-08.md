# G28 多账本与重载原生窗口验证

日期：2026-10-08（Asia/Singapore）。本记录补充 G28 的 macOS 原生窗口验收；不代表 G28 或 P4 完成。

测试在 `t.TempDir()` 写入两份独立账本，并用隔离的 `query-presets-v1.json` 分别预置账本专属查询。真实 MyGo Content 窗口依次打开账本 A、账本 B、再打开账本 A。Accessibility 树确认查询目录和命名预设随账本切换：A 只显示 A 的内容，B 只显示 B 的内容，重新打开 A 后恢复 A 的预设。

随后测试替换 A 中的 BQL 查询名称，通过真实“文件 → 重新读取账本”菜单触发重载，并在 Session.Reload 门控期间检查页面：旧 snapshot 的核心查询目录已隐藏，界面显示“查询目录正在读取或已过期”；账本身份绑定的自定义预设仍可见，这是有意保留的用户数据。释放重载后，workspace 发布新的 SnapshotID，原生窗口显示更新后的 BQL 项目，旧目录项目消失，A 的自定义预设仍保留。

定向场景连续5次通过，完整原生窗口套件也通过：

```sh
./apps/goui/native-e2e.sh -run '^TestNativeQueryCatalogueIsLedgerScopedAndReloadSafe$' -count=5
make goui-native-e2e
```

G28 的核心/Headless 目录搜索、排序、分页和预设存储边界仍由[G28首轮记录](goui-p4-g28-2026-10-07.md)覆盖。键盘/VoiceOver、真人输入法和旧版18项完整等价矩阵尚未完成；本记录只证明双账本隔离和重载期间的原生目录失效保护。

2026-10-08 后续回归在账本 A 中先打开 Balance Sheet、Journal 并返回，使返回和前进两侧均有历史，再打开账本 B，断言两侧历史都清空，并等待新的 Accessibility 树稳定后确认两个按钮都不接受 Press。该扩展与 `TestNativePageHistoryBackForwardAndBranchReplacement` 一起通过 `-count=3`；随后 `make goui-check` 与完整 `make goui-native-e2e` 通过。详见[G13 通用页面导航栈记录](goui-p2-g13-navigation-2026-10-08.md)。
