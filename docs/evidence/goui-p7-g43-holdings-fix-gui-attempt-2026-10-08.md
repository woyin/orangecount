# G43 持仓筛选修复包隔离 GUI 验证尝试

日期：2026-10-08（Asia/Singapore）。对已通过 package verifier 的 `0.0.0-g38-holdings-fix` arm64 development 包尝试进行独立 bundle 可见 GUI 验收。

## 实际包与隔离启动

- 来源 `.app`：`/tmp/orangecount-goui-g38-holdings-fix-20261008/OrangeCount Go UI Development.app`。
- ZIP SHA-256：`7696c7fafb8bd4f5b2108b3aaa525ef304d9b3fb4b73b612580ec154780ab676`；本轮重新运行 `make goui-package-verify OUT_DIR=/tmp/orangecount-goui-g38-holdings-fix-20261008 VERSION=0.0.0-g38-holdings-fix GOARCH=arm64`，通过。
- `.app` executable SHA-256：`d60f336a1cec3dce7635c646bb8f7620bf67c79e56684374974992c086f26104`。隔离副本位于 `/tmp/orangecount-goui-g43-holdings-fix-visible-20261008/`，只更改副本 `Info.plist` 的 bundle ID 和显示名称；副本 executable 与来源包 SHA-256 相同。
- 使用唯一 bundle ID `org.orangecount.desktop.goui.g43.holdingsfixqa`、`ORANGECOUNT_QA=1` 与 owner-only `/tmp/orangecount-goui-g43-holdings-fix-visible-20261008/user-data` 启动。没有使用桌面上已有 development bundle 的身份或用户数据目录。
- build-info 对应 source revision `1c2096d435e07cc9b5a5957e0e974546ee1387a6`、dirty tree、Go 1.27.1、MyGo `v0.2.16-0.20261006170038-869768d02d91`、arm64、macOS 13.0 minimum、CGO disabled、unsigned/notarized false。

## 窗口观察与限制

CUA 首次绑定该 bundle 时读取到 OrangeCount Go UI 窗口、`打开账本` 按钮和“QA 模式：不会自动读取最近账本”，确认隔离冷启动没有自动打开账本。点击“打开账本”后，原生 Open Panel 可见；按 Cmd+Shift+G 后，CUA Accessibility 树显示路径定位对话框。尝试输入公开 fixture `testdata/fixtures/native/holdings-unknowns.bean` 并提交后，CUA 的状态读取返回 ScreenCaptureKit `SCStreamErrorDomain -3812`（invalid parameter）。对同一窗口的 AX 读取及重新绑定重复得到该错误，因此无法确认 panel 是否关闭、账本是否打开或持仓页面结果。

Shell 检查确认临时应用进程仍在运行，且隔离 user-data 目录没有偏好文件；这不足以证明账本已打开。随后只向本轮临时 QA executable 发送 SIGTERM 并确认进程退出。没有操作真实账本、用户偏好或已有 development 应用。

## 结论

本轮只确认当前 r1 包可启动并显示隔离 QA 欢迎页及原生文件选择面板。由于 ScreenCaptureKit 故障，持仓修复包的可见读账本/筛选流程没有完成，也不计为 GUI 通过；r1 的既有 package-verifier 证据仍有效。真实 MyGo 窗口中的持仓筛选行为由[G38 原生持仓验证](goui-p6-g38-holdings-filter-state-native-2026-10-08.md)与本轮完整 native E2E 覆盖，但不替代该 r1 包可见 GUI。G43 保持进行中。
