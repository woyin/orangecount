# OrangeCount 0.1.5 本机开发交付

本批F01–F04已实施，源码5f4fc4c；App与CLI来自clean工作区。完整证据见[实施状态](evidence/missing-features-status.md)。Apple Silicon/arm64，macOS编译目标13；当前本机27.0.1验证。ad hoc签名，未公证。

解压macOS开发ZIP，打开OrangeCount.app，再通过“打开…”选择账本入口。默认语义仍为legacy；使用修订的booking/AVERAGE时，在“会计语义”中查看v3迁移比较后应用，不修改源账本。

本批可用：

- 资产负债表、损益、趋势、账户、流水、价格和持仓分析捕获/保存，报表格式v2；查询分析格式1，typed-v2 reader兼容旧typed-v1冻结文件。
- 附件额外目录设置、书签恢复、同账本窗口策略同步、撤销与旧引用拒绝。默认根不可删除；过宽目录选择有覆盖范围确认。
- balance Inventory、sum(inventory)、分组/ordinal/alias长尾及有界数值执行，215公开查询通过。非法GROUP/UTF-8错误可定位，并修复实际GUI辅助功能递归崩溃。
- CLI安全源导出：完整兼容检查、源/附件保护、单文件原子替换、多文件新目录一次发布。有效AVERAGE不能导出为纯Beancount源；其报表/查询分析仍可保存。详见[source-export](source-export.md)。

App ZIP：`dist/macos/OrangeCount-0.1.5-arm64-development.zip`，SHA-256 `3369cfd6faa3463ca01dc05bd24df9400e43fc696ebe8211d3b5d1f52bd470fc`。

CLI包：`dist/cli/OrangeCount-CLI-0.1.5-darwin-arm64.tar.gz`，SHA-256 `d525defb0ca2ebfa99163f6ccc958a3f2a6d85329c73283085ca892a88b4255b`。解压后使用`./orangecount`，仓库内则为`bin/orangecount`。

两包均在仓库外解压检查。App完成bundle/signature/arm64/deployment资源检查，build-info比对；CLI字节比对并实际完成公开dialect源export与v3 check。最新FinanceBook只读同源对照2836 entries、0库存/entries/期间差异，真实Swift/C通过。

完整N04/R01、另一台机器安装、旧系统实机、远端CI、Developer ID与公证尚未验收。这些不由本批功能完成自动关闭；本批明确排除的查询/源导出扩展范围见矩阵与source-export文档。
