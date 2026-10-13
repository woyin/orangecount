# G43 原生 Cmd+Q 退出进程验证

日期：2026-10-08

## 验证范围

新增 `TestNativeCmdQExitsApplicationProcess`，在隔离的原生 E2E 子进程中使用 OrangeCount 实际 `NewApplicationMenu` 创建应用菜单和原生窗口，再通过 AppKit 的菜单 `performKeyEquivalent:` 注入 Cmd+Q。子进程只有在菜单处理了快捷键、MyGo `App.Run` 返回、`TestMain` 正常返回且进程以 0 退出后才算通过；未处理快捷键时的兜底退出会返回失败，父测试设置 15 秒超时。

## 结果

- 定向 `./apps/goui/native-e2e.sh -run '^TestNativeCmdQExitsApplicationProcess$' -count=5 -v -timeout=1m`：连续 5 次通过。
- 定向 `./apps/goui/native-e2e.sh -race -run '^TestNativeCmdQExitsApplicationProcess$' -count=1 -v -timeout=2m`：通过。
- `make goui-check`：通过。
- `make goui-native-e2e`：通过；当前源码包含 dashboard、账户、持仓、价格导出上下文、来源段表格、G22 空状态回归及 G27 原生排序/键盘导航/类型化复制回归，原生 app 包测试耗时 66.217 秒。新增查询用例验证原生 Accessibility 表头操作、900×600/150% 字号下的 Home/End/Up/Down 导航、数值升序、排序后所选行身份保持，以及 typed JSON 剪贴板内容和测试后 pasteboard 恢复。

这证明原生菜单快捷键到 MyGo run loop 的自动化退出路径能够结束子进程。先前打包 `.app` 可见界面测试中“Cmd+Q 后仍显示运行”的观察尚未在当前源码新包上重跑，本记录不替代该交付验收；G43 仍进行中。
