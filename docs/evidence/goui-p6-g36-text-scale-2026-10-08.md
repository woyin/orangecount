# G36 每账本文字大小与原生快捷键验证

日期：2026-10-08（Asia/Singapore）。补齐新 MyGo macOS 版本的文字大小菜单、布局响应及账本级偏好保存。

## 实现

- MyGo fork 新增 `ui.Context.SetTextScale`，在每帧统一缩放默认主题字号与显式 `FontSize`，并和桌面辅助功能文字缩放相乘。
- View 菜单支持放大、缩小和恢复默认，快捷键为 Cmd+Plus、Cmd+-、Cmd+0；倍率限于 100%、125%、150%。
- 当前账本独立保存倍率，打开另一本账本不继承前一本的字号；重新打开时还原该账本设置。
- MyGo API 的差异及上游版本记录在[本地补丁说明](../../apps/goui/patches/mygo-text-scale.md)和[`UPSTREAM.md`](../../apps/goui/third_party/mygo/UPSTREAM.md)。

## 验证

真实 AppKit 窗口打开临时账本，在 900×600 内容窗口中用 Accessibility 读取稳定文本行框。原生应用外观先切到 Light、再切到 Dark 并恢复进程原设置，资产负债表内容在两种外观下持续可访问；这不是像素颜色验收。菜单放大后从 100% 到 125%，原生 Cmd+Plus 再到 150%；文本行框高度逐级增加。Cmd+F 在 150% 下仍能聚焦账户搜索框。Cmd+- 将字号降回 125%，菜单恢复默认回到 100%；再提高到 125% 后执行 Cmd+0，行框与初始值相同。每次变化均等待模型、原生 Accessibility 布局与对应账本偏好写入完成，最后新建窗口重开账本确认字号仍为 100%。

```sh
go test ./internal/app ./internal/preferences ./internal/i18n -count=1
GOWORK=off go test ./ui -count=1
./apps/goui/native-e2e.sh -run '^TestNativeTextScaleMenuReflowsAndPersistsPerLedger$' -count=5 -v
make goui-check
make goui-native-e2e
```

应用、偏好及 MyGo UI 定向测试通过；定向原生场景连续5次通过。`make goui-check` 和完整 `make goui-native-e2e` 通过（64 个原生用例通过、3 个环境门控用例跳过）。Light/Dark 原生切换后的实际颜色对比、其他窗口尺寸完整视觉矩阵、真人输入法与 VoiceOver 验收仍由 G36/G37/G38 跟踪；这些本机自动化结果不替代人工系统交互验收。
