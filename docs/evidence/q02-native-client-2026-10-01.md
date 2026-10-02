# Q02 后台客户端与独立取消路径

日期：2026-10-01。基线提交：`9e5d794`。本批继续Q02，未完成整个工作台。

## 已实现

`NativeLedgerClient`实现独立`LedgerQueryClient`接口。执行沿用窗口actor的串行C入口；显式取消是nonisolated异步控制调用，在独立任务中调用已有C取消接口，避免排在被取消的执行后面。Swift Task取消处理器也直接通知Go取消；执行前后检查Task取消。Go实际中断仍发生在预算检查点，不宣称能抢占单次大数、排序或序列化库调用。

## 当前验证

- 新真实客户端测试先因缺少executeQuery/cancelQuery接口编译失败；同时发现并修正测试将非optional snapshot_id当optional绑定的错误。
- 真实Swift/C测试：typed查询成功、执行前取消、下一代成功零行均通过。
- 深18层表达式的无取消对照返回work-budget-exceeded；异步提交同一表达式，等待20ms后由客户端取消，返回cancelled且没有execution。调度等待不单独证明精确C进入时点。
- 临时移除取消方法的nonisolated修饰，让取消排回执行actor，实际测试exit1：Control cancellation queued behind expensive C execution。源码随后恢复，未保留该变异。

- 最终串行`make mac-test`与`make app`均exit0：真实ContractTests、StateTests、NativeClientTests通过；开发包arm64、target13、ad-hoc校验通过，未启动GUI。中间一次构建因编译期间补源码注释被Swift拒绝，保持源码不变后重新完成全部检查。
- 本批只修改Swift客户端/测试及文档；前批Go全量/race、51会计/147查询和私人对照为继承证据，本批没有重跑私人账本或将旧检查称作新检查。
- 修改文档链接、两个公开源码SHA及`git diff --check`通过。

## 未完成与接续

下一步接Store草稿、单调代次、current/latest-pending队列及成功/错误/过期状态；覆盖换账本、profile、reload和关闭后迟到响应，再接QueryView。错误字节位置、保存query/preset、固定成功结果CSV、完整跨语言资源边界及GUI验收继续保留。LedgerSessionClient既有受控报表测试不被默认为查询状态证据。

## 本批公开源码身份

| 文件 | SHA256 |
| --- | --- |
| `apps/macos/src/LedgerStore.swift` | `7963febdcb4e303dffab229418a3365b6b7c77befac612de849172118e907a7f` |
| `apps/macos/tests/NativeClientTests.swift` | `ccd0f1d098e30b3a7cee8b3fa157a4c163fbee84a4a84e6c6e8735155af98f9c` |
