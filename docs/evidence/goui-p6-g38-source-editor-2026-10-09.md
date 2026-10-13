# G38 旧场景 15 来源编辑器首次启动补验

日期：2026-10-09。范围：补验来源文件交给一个尚未运行的外部编辑器时，Launch Services 能启动目标应用；不代表 G38 完整等价矩阵完成。

执行：

```sh
ORANGECOUNT_GOUI_SOURCE_EDITOR_E2E=1 ./apps/goui/native-e2e.sh \
  -run '^TestNativeConfiguredSourceEditorLaunchesScriptEditor$' -count=1 -v -timeout=3m
```

结果：通过，耗时约 1.6 秒。原生测试使用 `t.TempDir()` 中权限为 `0600` 的临时 `source.bean`，将它路由到 `/System/Applications/Utilities/Script Editor.app`。请求前确认目标应用未运行；请求成功后确认 Script Editor 已启动。测试清理关闭它自己启动的实例，后续进程检查未发现 OrangeCount 或 Script Editor 测试进程。

该结果补足旧 GUI 场景 15 的“配置外部编辑器首次启动”自动化覆盖。它没有验证用户亲自通过设置面板选择应用、真人 VoiceOver/IME，也不表示编辑器内的文件内容已由辅助功能读取确认。旧版 18 项整体等价矩阵仍未完成。
