# OrangeCount 0.1.6 开发交付

源码为 `13f00ee`，App 与 CLI 从干净工作区构建。本版交付完整兼容计划的首批实现，**整份计划尚未完成，不能宣称 Beancount/BeanQuery 完整兼容或超集**。完整状态见[实施证据](compatibility/implementation-progress-2026-10-02.md)及[typed-v3 契约](compatibility/query-typed-v3.md)。

## 使用

解压 App ZIP，打开 OrangeCount.app，选择账本入口。现有默认语义仍为 legacy；验证修订的 booking/AVERAGE 及标准查询行为时，在“会计语义”查看 v3 迁移比较并应用，不修改源账本。默认 v3 迁移 M01 将在完整兼容验收后实施。

新增可用内容：普通/递归 include glob、匹配目录变更监测、重复/循环及预算错误；插件声明阻断诊断；标准表首批字段、递归 metadata、集合及当前结构字段查询；静态 wildcard 列序；系统容差字典；查询分析 typed-v3，并继续读取旧 v1/v2 文件。总价 @@ 的 price 列正确返回单位报价。账本源导出继续拒绝 glob 等未验写入范围。

CLI 示例：

```sh
./orangecount query -semantics v3 -format typed-json /path/main.bean 'SELECT date,description FROM #events'
./orangecount query -semantics v3 -format typed-json /path/main.bean 'SELECT tags,meta,entry LIMIT 5'
```

## 产物

| 产物 | SHA-256 |
| --- | --- |
| `dist/macos/OrangeCount-0.1.6-arm64-development.zip` | `4f3ecffc60e7261e7df152014e6bf0323a64de970736336b16b0acc7a1d238fe` |
| `dist/cli/OrangeCount-CLI-0.1.6-darwin-arm64.tar.gz` | `85cfd641e2bee99bbd7c1a425f9f60ea71ff3d2106c056ad3ba7b96dbcd4f421` |

仓库内为 `build/OrangeCount.app` 和 `bin/orangecount`。App 与 CLI 的实际 Mach-O deployment target 均检查为 13.0，arm64；仅本机 macOS 27.0.1 有当前运行证据。App 为 ad hoc 签名，尚未公证。CLI 打包时先 source `apps/macos/toolchain.sh` 再构建，避免本机 SDK 默认使 CLI 最低目标误变为 27。

## 当前证据与限制

Go 全量、vet、相关 race、51 个账务差分及 284 个查询差分通过。原生四组测试通过，包括真实 Swift/C 的结构化值及恶意递归值拒绝。固定 Python query goldens 全量 check 通过；源导出 36 个往返、74 个逐交易库存截点、6 个拒绝通过；源加载 20 个 glob 模式、6 个语义案例通过。FinanceBook 只读同源对照为 47 文件、双方 2836 entries，零 entries/库存差异，期间及真实原生价格/持仓检查通过。

两包均在仓库外临时目录解压检查。App 包结构/签名/许可证/build-info 一致；CLI 字节比对、版本、目标系统、typed-v3 查询及真实公开源 export/check 通过。

本次 GUI 自动化两次报 `Sky Computer Use native pipe startup failed`，因此未把 0.1.6 的实际界面点击/显示验收记为通过。全量 SwiftUI App 编译、真实 C/Swift 契约和 Store 行为已通过；0.1.5 的 GUI 记录是历史证据。

仍需开发/验收：源 symlink/metadata/竞争边界，标准 id/hash 和完整结构访问，非有限 Decimal/完整 Any，全函数与运算符重载，子查询/HAVING/DISTINCT/PIVOT，命令/查询生态，完整逐项兼容登记及生成式证明，最终默认 v3 迁移。N04/R01 全面 GUI、旧系统实机、另一台机器安装、远端 CI 和正式签名发行仍保留。
