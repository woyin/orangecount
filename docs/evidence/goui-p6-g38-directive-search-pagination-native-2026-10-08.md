# G38 指令筛选、搜索与分页原生验证

日期：2026-10-08。范围：真实 macOS MyGo Content 窗口中的指令类型筛选、文本搜索、排序和分页；输入为测试专用临时公开账本，不读取或修改用户账本。

测试生成一条 `open` 指令、两条不匹配搜索词的 `note` 和101条匹配的 `note`。通过 Accessibility 控件打开“全部指令”、在“全部类型”菜单选择“备注”、选择“正序”，并在“搜索指令”输入 `needle-` 后按“筛选”。类型筛选结果为103条，第一页100条；来源行号3–102。应用搜索后结果减少为101条，第一页100条对应来源行5–104，下一页仅一条且来源行号为105，返回上一页后重新从行5开始。搜索和翻页前后均保留同一 snapshot ID，查询请求的类型、文本、排序和 offset 与实际控件操作一致。

验证结果：

- `./native-e2e.sh -run '^TestNativeDirectiveKindSearchAndPaginationKeepSourceIdentity$' -count=1 -v`：通过。
- 同一用例加 `-race`：通过。
- `make goui-check`：通过，Go UI 全部包检查及应用构建通过。
- `make goui-native-e2e`：通过，完整原生 UI 测试套件用时67.905秒。

本记录只证明该指令场景的窗口交互及源身份保持；不代表旧版18场景全量等价、中文 IME/VoiceOver 或生产账本升级验收已完成。G38仍为进行中。
