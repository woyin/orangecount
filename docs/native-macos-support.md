**原生 macOS 开发产物支持与验证范围**

构建架构：Apple Silicon / arm64。`make app` 显式将 Swift、cgo/C 链接与 Info.plist 的 deployment target 统一为 macOS 13.0；`check-bundle.sh` 比对最终 Mach-O 与 plist，而非仅检查声明。

macOS 13.0 是编译目标，尚无该系统的真实运行证据。Intel / universal 暂不支持。2026-10-01 当前本机环境是 Apple M4 / Mac16,10、macOS 27.0.1 (26A434)、Go 1.27.1、Swift 6.4、macOS 27 SDK。旧 macOS 27.0 (26A428) 记录的核心查看、错误恢复、分页与 20 次窗口生命周期属于继承证据；最新投影、部分 GUI 和未验项见[当前进度](go-beancount-native-progress.md)，不能将编译成功等同于旧系统兼容。

CI 使用明确的 `macos-15` arm64 runner 标签，不使用随时间迁移的 latest；Go 版本固定为 1.27.1，Swift 与 SDK 由该 runner 的选定 Xcode 提供，作业记录其实际版本。标签架构来源：[GitHub hosted runners](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)。CI 文件已经加入仓库；远端运行结果须以实际工作流为准，本地不能宣称通过远端 CI。

`make mac-package` 将已验证的 app 打为 `dist/macos/OrangeCount-0.1.0-arm64-development.zip`，同时生成 SHA-256 校验文件。app 包含 LICENSE、NOTICE 与 build-info.txt（版本、源码修订、工作区是否有修改、工具链、SDK、目标系统）。支持 `ORANGECOUNT_VERSION`；本地开发默认 0.1.0。

这是 ad hoc 签名的开发包，尚未执行 Developer ID 签名、公证或另一台目标机器安装验收。对外分发应在这些证据齐全后单独产出分发包。

正常启动不读取仓库 fixture。命令行测试启动设置 `ORANGECOUNT_TEST_MODE=1` 并指定脱敏账本；GUI 自动化使用 `prepare-qa.sh` 生成的独立 QA bundle。两者都隔离用户最近账本偏好。无账本启动展示欢迎页。

当前本机开发交付为[0.1.5](native-development-release-0.1.5.md)，F01–F04功能及App/CLI包已验；上文目标系统与正式发行限制继续适用。
