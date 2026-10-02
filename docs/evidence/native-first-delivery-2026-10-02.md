# 原生首版交付验收（2026-10-02）

## 当前检查

本批修复全指令 payload 的捕获日期与原生日期说明，配置日期为空且控件禁用，递归元数据 tag/link 保留 #/^ 前缀。针对 Go 捕获日期新增检查通过。

- `go test -count=1 ./...`：通过。
- desktop/snapshot/bridge/documents race：通过。
- fmt/vet/license：通过；无 runtime Go 依赖。
- 会计差分：51/51，AVERAGE 扩展及相关 snapshot 检查通过。
- 查询差分：163/163。
- mac-check：原生四套、app 与 bundle 检查通过（窗口生命周期补丁前）。
- 当前 FinanceBook 只读重新对照：47 文件，双方 2833 指令，零错误、分录与库存零差异；Expenses 73/Income 22 行一致；native_checked/native_contract true，3 截至日持仓及48价格检查通过。私人内容未写入仓库。

## 当前实际 GUI

独立 delivery QA 使用公开 all.bean：交易3、全部指令18；期间2000-01-01至2000-01-07正确；高精度元数据、NULL/空字符串/空list、#tag/^link显示正确；配置9条按加载序、日期禁用且说明无日期。账户库存预设仅填草稿，显式执行成功2行，导出按钮可用。CUA 两次瞬时 AXError.failure，刷新完整树后确认动作已成功；不将错误当成验收成功。

通过文件选择切换私人账本后出现空页面，阻断首版。窗口会话清理此前绑定 ContentView.onDisappear；已改为所属 NSWindow.willCloseNotification。首个编译因 Swift6 非隔离 deinit 不能访问非 Sendable observer token 失败，改为 NSObject selector observer 后 app 构建与 bundle 检查通过。复测文件选择后保留了读取状态，但私人 GUI 仍未通过；不能宣称已解决整条打开路径。

## 未完成范围

附件原生列表/打开、block dialect 完整关联、全指令完整资源边界、查询和报表容器剩余范围、完整GUI/性能/辅助功能、远端CI/旧系统实机/签名公证仍保留。开发包不能称为正式发行版。

## 开发产物与阻断（最终本批状态）

最终 app 构建 exit0；package.sh exit0；ZIP 解压后原始签名校验通过。为避免触碰用户实例，解压副本仅修改 QA bundle 身份和 test mode 并重新 ad hoc 签名，实际启动公开账本成功。此方法验证同一执行文件和 archive 解压，不能代替未修改 bundle 的安装验收。

私人 GUI 文件选择后的读取仍未完成，process sample 显示后台线程停在系统 open 调用、主线程运行事件循环；原因未确定。解压 QA 文件选择器中也观察到选中文件但 Open 禁用（私人及公开路径）。没有据此断言 TCC、权限或文件提供器为根因，没有更改系统权限，也没有修改用户账本。FinanceBook 的终端启动真实 C/Swift 对照通过是当前独立证据，不能代替 Finder 启动/文件选择 GUI 验收。

结论：提供可启动开发产物并提交代码，但交付计划 C/D 尚未全部满足；完整24项任务没有完成。下一个必须解决的门槛是文件选择/系统 open 阻断，之后补齐完整 GUI 阅读及安装循环。
