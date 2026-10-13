# G36 Cmd+N 原生新窗口验证

日期：2026-10-08（Asia/Singapore）。验证应用菜单 Cmd+N 会通过 MyGo 创建实际可访问的 Content 窗口。

## 场景

- 在真实账本 Content 窗口安装原生应用菜单，并将 New Window 回调连接到窗口创建函数。
- 聚焦账本窗口，经 AppKit application menu 执行 Cmd+N；Accessibility 树确认新窗口已显示“打开账本”主操作。
- 再次聚焦原账本窗口并执行 Cmd+N，确认第二个新窗口也被创建并可访问。

## 验证

```sh
./apps/goui/native-e2e.sh -run '^TestNativeNewWindowShortcutCreatesNativeWindow$' -count=5
make goui-check
make goui-native-e2e
```

Cmd+N 场景连续5次通过；`make goui-check` 和包含49项常规原生窗口场景的完整 `make goui-native-e2e` 通过。测试创建的窗口在结束时关闭并等待 workspace 释放。真人键盘/输入法与 VoiceOver 验收仍属于 G37。
