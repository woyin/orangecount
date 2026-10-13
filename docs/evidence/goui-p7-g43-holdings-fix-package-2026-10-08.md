# G43 持仓筛选修复后的 development package 校验

日期：2026-10-08（Asia/Singapore）。为避免交付包落后于 G38 持仓筛选状态修复，基于当前 dirty tree 重新打包并执行完整结构校验。

- 输出目录：`/tmp/orangecount-goui-g38-holdings-fix-20261008/`。
- 版本：`0.0.0-g38-holdings-fix`，架构：arm64。
- ZIP SHA-256：`7696c7fafb8bd4f5b2108b3aaa525ef304d9b3fb4b73b612580ec154780ab676`。
- build-info：source revision `1c2096d435e07cc9b5a5957e0e974546ee1387a6`，tree `dirty`，Go `go1.27.1 darwin/arm64`，MyGo `v0.2.16-0.20261006170038-869768d02d91`，CGO disabled，min macOS 13.0，unsigned/notarized false。
- `make goui-package-verify OUT_DIR=/tmp/orangecount-goui-g38-holdings-fix-20261008 VERSION=0.0.0-g38-holdings-fix GOARCH=arm64`：通过，核对 ZIP/hash、plist、架构与 minOS、CGO/Swift linkage、build-info 和依赖许可证清单。

这个 r1 包尚未重复可见 GUI 验收。前一条隔离 QA GUI 记录对应在 holdings filter fix 之前生成的 `0.0.0-g38-current` 包；它仍是该包的有效读取证据，但不代表本 r1 executable 的 GUI 结果。本轮 native suite 的多个窗口/菜单用例在当前机器上失败或超时，r1 的可见启动/读账本/操作闭环暂不宣称通过。
