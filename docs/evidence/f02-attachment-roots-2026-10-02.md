# F02 附件目录策略实施

实现Session独立目录策略，默认根不可移除，最多32额外canonical目录；原子更新与policy_revision比较交换，旧策略动作拒绝。引用缓存8192项，更新清空；会计快照/BuildCount不变。协议4新增读取/配置入口，附件与动作带策略身份。

原生书签按规范账本入口保存，恢复核对bookmark stale与目录device/inode，目录替换需重新选择。当前非沙盒开发版使用普通bookmark；未声称已有security-scoped权限。设置页支持显式目录选择、移除、附件观察刷新；同账本窗口广播带持久配置代次，不同账本隔离。持久化失败明确显示本次有效但未保存。迟到目录选择与附件Open/Finder检查策略和窗口代次。Finder alias在原生启动前拒绝，Go仅核验普通路径且不会解析Finder alias。

Go全量通过；documents/desktop/bridge race通过；fmt/vet/license通过。真实C/Swift检查验证grant/revoke、书签恢复、目录替换、同账本双窗口与不同账本隔离、会计快照不变。原生四套最终测试通过。App首次编译失败来自SDK State宏，改用既有StateObject模式后构建通过；随后补辅助功能按钮与alias防护，最终源码包构建另记。

公开GUI独立rootsqa：外部receipt初始outside-root，Open/Finder禁用；显式选择receipts目录后available，两动作启用；实际Open后TextEdit显示公开测试内容。切换外部应用后计算机观测返回SCStream -3812；未将失败观测记通过。另一独立rootsrevokeqa完成相同授权后移除，随后outside-root且两按钮禁用。目录移除原List未独立暴露按钮，改用ScrollView/VStack；完整N04仍保留。

使用输入仅build/f02-public公开临时目录，FinanceBook及用户原生实例未修改。旧F01私人对照继承，F02未改会计求值；当前版本私人archive对照尚未重新执行。

最终源码4d49e1a、0.1.3 build-info working_tree=clean；最终App编译、签名/arm64/目标13资源检查通过。开发ZIP仓库外解压校验与build-info比对通过。SHA-256：`ef1f5243dde11cafa332c074c224d970bc0c24ecc86670b5f11d389b39a58d26`。最终真实NativeClientTests再次运行通过。

后续回归修订：失效的额外根不再阻断默认根内附件；新增公开回归及race通过。选择系统根、用户Home、/Users或/Volumes时增加覆盖范围确认；该确认未执行实际宽泛授权GUI验收。
