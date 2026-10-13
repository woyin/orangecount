# G14 Legacy/V3 语义预览与切换原生窗口验证

日期：2026-10-08（Asia/Singapore）。本记录为 G14 语义预览与切换验收证据，不代表 P2 阶段完成。

`TestNativeAverageSemanticsPreviewAndApplyUsesCurrentInput` 将公开 `testdata/fixtures/native/average-history.bean` 复制到临时目录，按 Legacy 打开真实原生窗口。通过 Accessibility 导航到“语义配置”，确认 UI 显示 Legacy/V3 预览及差异；核心响应与当前快照绑定、`InputIdentity` 非空、没有请求逐条详情，AVERAGE fixture 至少存在一项配置差异。

测试在旧会话中设置查询草稿，再由 Accessibility 点击“应用 V3”。完成后检查 Session epoch 已更新、profile 为 V3、快照 ID 与同一输入的 V3 预览一致、重新发布的迁移响应仍绑定原输入身份、旧查询草稿清空、该账本的 profile 偏好已保存；临时账本文件逐字节未变。

验证命令：

```sh
ORANGECOUNT_GOUI_NATIVE_E2E=1 GOWORK=off CGO_ENABLED=1 \
  go test -tags orangecount_e2e ./internal/app \
  -run '^TestNativeAverageSemanticsPreviewAndApplyUsesCurrentInput$' -count=5 -v
```

真实原生窗口场景连续 5 次通过。测试使用临时账本与隔离偏好，不访问私人数据。外部修改预览后旧身份拒绝另见下节；真人 VoiceOver 与跨页面/面板交互由 G37/G38 的系统验收矩阵覆盖。

## 2026-10-08 原生预览失效与旧身份拒绝补充

新增 `TestNativeStaleSemanticsPreviewCannotApplyAfterInputChanges`：真实窗口先读取临时 `average-history.bean` 并显示 Legacy/V3 预览；测试向临时源文件追加注释，等待自动 watcher 发布新 snapshot，并用 gate 暂停新 migration read。原预览操作此时从 Accessibility 树消失，显式调用旧 profile 操作也因工作区忙碌而不切换会话。释放 gate 后，窗口显示绑定新 snapshot 且 `InputIdentity` 已变化的预览；重放旧身份仍显示“账本快照已变化”并保留 Legacy profile、当前 snapshot 与 epoch。场景连续 5 次通过。

这补上了真实 UI 的输入变化/旧预览失效路径；应用期间 query/report capture 的实时取消场景及真人 VoiceOver 仍待验收，G14继续进行中。

## 2026-10-08 profile 替换期间的 query capture 隔离补充

新增 workspace 测试 `TestProfileReplacementDropsLateQueryCapture`：先执行成功查询并挂起其冻结导出，再以 V3 替换 Legacy workspace。新 profile 打开期间立即清空旧 capture ID/state；新会话就绪后才释放旧 Session 的捕获结果，并确认晚到 bytes 不会发布或污染新 workspace。普通 `-count=10` 与定向 race 测试通过。

这个验证证明 profile 替换能使旧 capture 结果失效并丢弃，不代表底层有界序列化操作被中断；report/query capture profile replacement 的真实窗口并发组合仍列在 G38 GUI 等价矩阵中。
