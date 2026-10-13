# G42/G43 当前源码开发包校验

日期：2026-10-08（Asia/Singapore）。从当前 MyGo 工作树重新生成 arm64 development `.app` 和 ZIP，并验证包内结构与构建元数据。

## 产物与校验

- 输出目录：`/tmp/orangecount-goui-final-20261008/`
- App：`OrangeCount Go UI Development.app`，版本 `0.0.0-goui-final`，bundle id `org.orangecount.desktop.goui.dev`。
- 目标：arm64；最低 macOS `13.0`；CGO 关闭；未签名、未公证。
- 依赖检查：无 Swift 运行时链接；包内包含 OrangeCount、Go toolchain、MyGo fork 与 resolved Go modules 的许可证材料。
- ZIP SHA-256：`1159494f25609854c1224686c52d80658e5c44962867759dd60221e389aa9377`。
- 交叉构建的 amd64 `.app` 使用相同版本/bundle/minimum/CGO/许可证门槛；完整包校验通过，ZIP SHA-256：`59f28452b38b31fa42b7be45b6ee374581c5bf633a947e634dc21bd9d86bb866`。输出目录：`/tmp/orangecount-goui-final-amd64-r2-20261008/`。

运行命令：

```sh
OUT_DIR=/tmp/orangecount-goui-final-20261008 VERSION=0.0.0-goui-final GOARCH=arm64 make goui-package
OUT_DIR=/tmp/orangecount-goui-final-20261008 VERSION=0.0.0-goui-final GOARCH=arm64 make goui-package-verify
```

打包脚本的 arm64 和 amd64 Mach-O/Info.plist 检查通过；完整包验证器分别核对 ZIP checksum、解压 bundle、架构、最低系统版本、CGO/Swift linkage、build-info 和模块许可证清单后通过。最新 `make goui-check` 也通过。

该新 ZIP 是包结构和构建校验结果。本次没有在此新 bundle 上重跑完整可见读账本/导出 GUI 循环；相同 workflow 的既有可见包验收见[G43 GUI 导出与读回记录](goui-p7-g43-gui-export-loop-2026-10-08.md)。G43 仍待同一当前包的 GUI 循环、全旧偏好迁移/dev-QA隔离矩阵与进程退出确认。

曾以 QA 模式从该 bundle 启动进程，但终端报告 `sandbox_extension_issue_file_to_process`（Operation not permitted），CUA 无法取得窗口，返回 ScreenCaptureKit `-3811`。该次没有截图或窗口状态证据，不计作 GUI 验收；只停止了本次启动的临时 bundle 进程，未关闭其他 OrangeCount 进程。

## 2026-10-08 最新工作树重打包

在 G20–G23 导出上下文、价格来源段显示及 G27 原生查询结果验证合入当前工作树后，用版本 `0.0.0-goui-dev-20261008` 重新生成两个架构的 development app 和 ZIP。两个 build-info 都指向仓库 HEAD `1c2096d435e07cc9b5a5957e0e974546ee1387a6`，并准确标记 `source_tree=dirty`、CGO 关闭、MyGo 固定版本、macOS 13.0 最低版本、unsigned/unnotarized。

- arm64：`/tmp/orangecount-goui-current-arm64-20261008/`；ZIP SHA-256 `ba6b094bbc54da8e17cbfad73b45c58f1d7d331138ba4517b7403575a9920446`。
- amd64：`/tmp/orangecount-goui-current-amd64-20261008/`；ZIP SHA-256 `4a8bfd8bddf73d68cfba545e1c5442407950f894e752ba87df7ca440d74e7e9c`。

两份产物均通过 `make goui-package-verify`：验证了 plist、ZIP hash、解压内容、目标架构、minimum、CGO/Swift linkage、build-info 和完整 module-license inventory。本次只证明当前源码能够生成结构完整的两个开发包，没有在这两份新 bundle 上重复完整可见 GUI 阅读/查询/导出/退出循环；G43 仍进行中。

`.github/workflows/goui.yml` 仍只存在于当前工作树，GitHub 默认分支没有该 workflow，远端只读查询返回 HTTP 404。当前没有提交或推送，因此没有可供引用的远端 Actions run；不记录为 CI 通过。G44 的 macOS 13 实机、另一台机器、Developer ID 和公证仍是独立发行门槛。

## 2026-10-08 启动存储组装更新后的重打包

将 production/QA 启动使用的存储初始化抽到 `cmd/orangecount-goui/openApplicationStores` 并接入 `main()` 后，重新构建当前 dirty tree 的 arm64/amd64 development bundle，版本 `0.0.0-goui-dev-20261008-r2`。build-info 指向 HEAD `1c2096d435e07cc9b5a5957e0e974546ee1387a6`，标记 tree dirty、Go 1.27.1、MyGo `v0.2.16-0.20261006170038-869768d02d91`、CGO 关闭、macOS 13.0 minimum、unsigned/unnotarized。

- arm64：`/tmp/orangecount-goui-current-arm64-20261008-r2/`，ZIP SHA-256 `d4a69ed3f2d75bd62b8a00d15220a6053f0459b8822aafaee9ebcc012bc32a68`。
- amd64：`/tmp/orangecount-goui-current-amd64-20261008-r2/`，ZIP SHA-256 `b83c4d688ca0ce431c71b2771a0c8daecf133947957e3c8cad75af4d6ab5af43`。

两份 ZIP 均通过 `make goui-package-verify`，包含解压内容、架构、plist、minimum、CGO/Swift linkage、build-info、hash 与 resolved module license inventory 校验。本轮没有从新 bundle 启动可见 GUI；完整包阅读/查询/导出/退出循环仍是 G43 的未完成验收。

## 2026-10-08 启动偏好错误显示后的当前包

欢迎页新增偏好目录启动错误显示后，以版本 `0.0.0-goui-dev-20261008-r3` 从当前 dirty tree 重生成 arm64/amd64 development bundle：

- arm64：`/tmp/orangecount-goui-current-arm64-20261008-r3/`；ZIP SHA-256 `1a7454bde98c32426e17c21386fcc722602359e591de24131f9053341ef05798`。
- amd64：`/tmp/orangecount-goui-current-amd64-20261008-r3/`；ZIP SHA-256 `1f214a3300dc560eac9791b5f8c29880ab18379ed80f4dfde7db7b8da1db8bcb`。

两包均通过 `make goui-package-verify`：校验 ZIP hash、解压 bundle、Mach-O 架构、最低 macOS 13.0、CGO/Swift linkage、build-info 与 resolved module license inventory。此轮没有启动打包 `.app`；真实原生窗口中注入错误后的 Accessibility 可见性通过了完整 `make goui-native-e2e`（69.393秒）。当前包的打包应用完整读账本/查询/导出/退出循环仍待 G43。

## 2026-10-08 G16 账户余额与折算显示后的当前包

修复报表账户树遗漏直属余额、子树总额和逐账户目标币种折算状态后，以版本 `0.0.0-goui-dev-20261008-r4` 从当前 dirty tree 重新生成 arm64/amd64 development bundle。build-info 指向源码 revision `1c2096d435e07cc9b5a5957e0e974546ee1387a6`，标记 dirty tree、Go `1.27.1`、MyGo `v0.2.16-0.20261006170038-869768d02d91`、CGO 关闭、macOS 13.0 minimum、unsigned/unnotarized。

- arm64：`/tmp/orangecount-goui-current-arm64-20261008-r4/`；ZIP SHA-256 `6604d9a95f33e4e352737583469d7f2d7836e8ec8b26b12ebca204cd6c7e5042`。
- amd64：`/tmp/orangecount-goui-current-amd64-20261008-r4/`；ZIP SHA-256 `589afe0acfac5f9c9dd8245595f27dcb0fcede75d0df8b1f51777917cd66540d9`。

两包均通过 `make goui-package-verify`，覆盖 ZIP/hash、解压 bundle、架构、plist/minimum、CGO/Swift linkage、build-info 和 resolved module-license inventory。新包未启动可见 GUI；G43 的当前源码包读账本/查询/导出/退出 GUI 循环仍待验收。G16 相关实现和测试见[G16 自身余额与折算估值显示记录](goui-p2-g16-own-subtree-valuation-2026-10-08.md)。

## 2026-10-08 G43 r6 隔离开发 GUI 实测

在加入显式 `ORANGECOUNT_ISOLATED_USER_DATA_DIR` 后，以 `0.0.0-goui-dev-20261008-r6` 重新构建当前 dirty tree 的 arm64/amd64 packages。arm64 ZIP SHA-256 `5e30771c01e84d19e7cc8a779187439f041c313caad2579cea6814ae7ad7cf94`；amd64 `538f2b0dfe5651cf093f875ef738e1a49143e543580da4dc2423750d0c66ecca`。两包重新通过 `make goui-package-verify`。

同一 arm64 包的 executable 在隔离副本身份下完成了开发模式 UI 读账本、查询、CSV/报表归档保存、离线查询捕获读回、同窗口语法错误与恢复；Cmd+Q 后对应 shell-launched 进程状态码为 0。完整步骤和限制见[G43 r6 隔离开发包 GUI 闭环](goui-p7-g43-r6-isolated-development-gui-2026-10-08.md)。一份额外全新 user-data 目录的冷启动因 CUA/ScreenCaptureKit `-3811` 无法读取窗口，不计为重启验收；旧偏好迁移及 dev/QA 完整隔离矩阵仍未通过，G43 继续进行中。

## 2026-10-08 G43 r7 隔离目录符号链接校验更新

`configureIsolatedUserDataPath` 现在对最终 user-data 路径使用 `Lstat`，拒绝目录本身是符号链接的 override；定向测试确认共享权限目录和 symlink target 的权限均不会被校验流程改写。`GOWORK=off go test -race ./cmd/orangecount-goui -count=1`、`GOWORK=off go vet ./cmd/orangecount-goui` 与 `make goui-check` 通过。基于该源码生成的 arm64/amd64 r7 ZIP 均通过完整 `make goui-package-verify`：

- arm64 ZIP SHA-256：`7c960e378683ea20eeda900a59a1c7493387a16bfa37cb968551859591d9584b`。
- amd64 ZIP SHA-256：`84c9d03497bdccd1547180941039c0c5f11022c150dd0aea84ffc568f378788a`。

此窄改动后的 r7 包尚未重跑可见 GUI；可见 GUI 证据仍是 r6 包中的 owner-only 普通目录场景。G43 仍待重启、旧偏好迁移/dev-QA 完整矩阵和 G37/G38 验收。

### 2026-10-08 r7 可见 development 与 QA 隔离补测

随后在 r7 arm64 development package 的隔离副本上完成 GUI 导出、错误恢复、重启与退出，见[G43 r7 development GUI 记录](goui-p7-g43-r7-isolated-development-gui-2026-10-08.md)。再从当前 dirty source 生成 `qaBuild=true` 临时 arm64 bundle，在独立 bundle ID/私有 user-data 下完成 QA 首启、公开账本读取和偏好落盘/读取；QA 的 PID 确认重启、最近项归属和 GUI 优雅退出仍待复验，见[G43 QA 身份与隔离矩阵](goui-p7-g43-qa-isolation-matrix-2026-10-08.md)。干净用户安装下完整 legacy production preference 升级、G37/G38 与其余依赖也仍未完成，G43 继续进行中。

### 2026-10-08 r7 包隔离 GUI 复验

上面“尚未重跑可见 GUI”的记录是打包时的状态，之后已在 r7 arm64 ZIP 对应 executable 的隔离 bundle 副本上完成可见 GUI 闭环。实测覆盖读账本、BQL 查询、CSV 与 `.ocanalysis` 保存/读回、流水和资产负债表导出、同窗口语法错误恢复、Cmd+Q、同一隔离 user-data 重启和再次退出；两次启动的 shell session 均以状态码 `0` 结束。详细步骤、导出校验和 hash 见[G43 r7 隔离开发包 GUI 与重启闭环](goui-p7-g43-r7-isolated-development-gui-2026-10-08.md)。

这次只关闭当前 r7 bundle 的可见 GUI 循环缺口。旧偏好迁移及 development/QA 完整隔离矩阵、G37/G38 验收、远端 CI 结果仍未完成；G43 继续进行中。
