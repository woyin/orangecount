# G42 development package verifier

日期：2026-10-08（Asia/Singapore）。新增可在本机和独立 macOS CI 重复运行的 `.app`/ZIP 完整性校验。

## 校验内容

- 校验 ZIP SHA-256，并解压确认 archive 含有预期可运行的 development `.app`。
- 检查 bundle ID、版本和 macOS 13.0 minimum；Mach-O 架构及 deployment target 必须与 plist 一致。
- 检查 Go build metadata 为 `GOOS=darwin`、目标架构和 `CGO_ENABLED=0`，并确认无 Swift runtime linkage。
- 检查 build-info 的架构/版本/开发包签名状态，以及所有 manifest 引用的许可证文件。
- 将包内 module/license 映射与当前 `apps/goui/go.mod` 的 resolved module graph 对照，并比较原始 `.app` 与 ZIP 中的 plist、可执行文件和许可证清单。

CI 的 package job 现在调用 `apps/goui/verify-package.sh`；本机可用 `make goui-package-verify OUT_DIR=... VERSION=... GOARCH=...` 验证已生成的包。

## 验证

- 对现有 `dist/goui` 包运行 `make goui-package-verify OUT_DIR=dist/goui VERSION=0.0.0-dev GOARCH=arm64`：通过。
- 从当前源码生成新临时 arm64 `.app`/ZIP 后运行 verifier：通过，SHA-256 匹配，最低系统版本为 macOS 13.0，module/license 清单与当前 module graph 一致。
- 从当前源码交叉生成 amd64 `.app`/ZIP 后运行 verifier：通过。验证器将 Go 的 `amd64` 名称映射到 Mach-O 工具使用的 `x86_64`，并核对二进制 metadata 仍标记 `GOARCH=amd64`。
- 对新包副本追加一个字节后运行 verifier：按预期因 SHA-256 不匹配而失败。
- `sh -n apps/goui/verify-package.sh`：通过。

此验证提升了包完整性和依赖许可证闭环证据；没有启动包内查询/导出 GUI，也不替代真实安装、签名、公证或远端 Actions 运行。
