# G12 启动偏好存储错误可见性

日期：2026-10-08。范围：当 macOS 未能提供应用 user-data 路径时，把启动错误传入 Go UI Model，并在欢迎页显示；同时让已有偏好读取错误在尚未打开账本时可见。

`main()` 仍记录 `mygo.App.Path(mygo.PathUserData)` 错误，但现在也把错误交给每个新窗口的 `Model.SetPreferenceStartupError`。欢迎页用危险色显示错误；文案根据界面语言本地化，操作系统错误细节保留原样。用户仍可继续选择并阅读账本。

定向 `ui.Tester` 测试分别验证简体中文原文和 English 错误文案；新增原生窗口 E2E 通过 macOS Accessibility 确认欢迎页可读到错误内容，定向和完整原生 E2E 均通过。完整门禁耗时 69.393 秒。验证命令：

```sh
make goui-check
cd apps/goui
GOWORK=off go vet ./...
GOWORK=off go test -race ./internal/app ./internal/i18n ./cmd/orangecount-goui
```

上述检查通过。随后从更新后的 dirty tree 生成 arm64/amd64 development bundles，并分别通过 `make goui-package-verify`。arm64 ZIP SHA-256 为 `1a7454bde98c32426e17c21386fcc722602359e591de24131f9053341ef05798`；amd64 为 `1f214a3300dc560eac9791b5f8c29880ab18379ed80f4dfde7db7b8da1db8bcb`。两包标记 unsigned/unnotarized，最低 macOS 13.0、CGO 关闭、无 Swift runtime linkage。

本轮没有启动打包 `.app`，也没有故意破坏实际 macOS user-data 目录；因此系统 `PathUserData` 失败后的当前 bundle 冷启动路径仍需 G43 验收。原生测试覆盖的是注入失败后的真实窗口与 Accessibility 输出，不替代 G37/G38 的真人 IME/VoiceOver 与旧版全 GUI 等价验收。
