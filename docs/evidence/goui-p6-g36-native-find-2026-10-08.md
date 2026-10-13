# G36 双窗口 Cmd+F 原生路由验证

日期：2026-10-08（Asia/Singapore）。验证 Go UI 原生菜单的 Find key-equivalent 根据当前 key window 路由到对应账本模型。

## 场景

- 创建两个临时账本、两个真实 MyGo Content 窗口及其独立 workspace；两个窗口都进入流水页并显示原生“搜索流水”输入框。
- 聚焦第一个窗口，经 AppKit application menu 执行 Cmd+F key-equivalent；原生 Accessibility 树确认第一个窗口的搜索框进入 focused 状态，第二窗口模型没有遗留 Find 请求。
- 聚焦第二个窗口并重复；焦点落到第二个窗口，命令没有继续发送给第一个窗口。

测试钩子仅在 `darwin && orangecount_e2e` 构建中暴露原生 Accessibility focused 状态和 AppKit `performKeyEquivalent`。这验证应用菜单的 key-equivalent 路由及原生焦点结果，不等同于物理键盘事件、中文输入法候选交互或真人 VoiceOver 验收。

## 验证

```sh
./apps/goui/native-e2e.sh -run '^TestNativeFindShortcutTargetsTheFocusedLedgerWindow$' -count=5
make goui-check
make goui-native-e2e
```

单项场景连续运行5次通过；`make goui-check` 通过；完整 native E2E 的37项常规窗口场景通过。此次新增不修改生产构建中的 MyGo API，也不改变任何用户账本或偏好。

此记录生成时，Cmd+O/Cmd+R 原生路由尚未覆盖；后续双窗口回归已验证打开面板归属和重读命令路由，详见[G36 Cmd+O/Cmd+R 原生验证](goui-p6-g36-native-open-reload-2026-10-08.md)。新建窗口菜单的原生创建、深浅色和原生 900×600/1.5 界面矩阵仍待覆盖。输入法与 VoiceOver 属于 G37。
