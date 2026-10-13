# G03 欢迎页最近账本显式打开原生验收

日期：2026-10-08。范围：在真实 macOS MyGo Content 窗口中验证最近账本不会自动恢复，并且可由用户明确选择打开。

测试 `TestNativeWelcomeRecentLedgerOpensOnlyAfterSelection` 使用非 QA 的应用 Model、公开 `testdata/fixtures/native/cashflow.bean` 和 `t.TempDir()` 中的隔离偏好文件。偏好文件预先记录该账本后，真实欢迎窗口仍显示“欢迎使用 OrangeCount”和不自动读取最近账本说明；workspace/controller 与 Model 均没有活动账本。测试随后通过 AppKit Accessibility 按下最近账本按钮，等待同一公开 fixture 成功打开，确认两笔交易以及预期现金流值可见。

```sh
./apps/goui/native-e2e.sh -run '^TestNativeWelcomeRecentLedgerOpensOnlyAfterSelection$' -count=5
```

定向场景连续5次通过；包含该场景的完整 `make goui-native-e2e` 套件通过（63.926秒）。测试只使用临时偏好和公开 fixture，不读取生产 defaults 或私人账本。

本证据覆盖非 QA 应用界面分支，但偏好 Store 由测试注入，未启动生产 bundle 或读取用户默认值。它不代表安装包首次启动/重启恢复、真人 VoiceOver 或完整 macOS GUI 等价矩阵已验收；G38 与 G43 仍跟踪这些项目。
