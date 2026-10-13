# G36 双窗口 Cmd+O/Cmd+R 原生路由验证

日期：2026-10-08（Asia/Singapore）。验证打开与重读快捷键只作用于当前焦点账本窗口。

## 场景

- 创建两个真实 MyGo Content 窗口，各自打开独立临时账本，并将应用菜单回调映射到对应模型。
- 先后聚焦两个窗口，经 AppKit application menu 执行 Cmd+O；原生 Open Panel 只挂在当前窗口，取消后两个窗口均无遗留 sheet，模型选择状态也恢复。
- 先后聚焦两个窗口，经 AppKit application menu 执行 Cmd+R；当前工作区完成重新读取，另一窗口的 operation 序号保持不变。

## 验证

```sh
./apps/goui/native-e2e.sh -run '^TestNative(OpenLedger|Reload)ShortcutTargetsTheFocusedWindow$' -count=5
```

两项场景各连续5次通过。MyGo 的原生 E2E 钩子仅在 `darwin && orangecount_e2e` 构建启用，不进入生产构建。Cmd+N 的原生窗口创建回调、真实输入法冲突、VoiceOver，以及原生深浅色与窗口尺寸矩阵仍未由本记录验收。
