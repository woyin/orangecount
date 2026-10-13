# G35 附件目录多窗口撤权原生验证

日期：2026-10-08（Asia/Singapore）。本记录补充真实 macOS 原生窗口中的共享目录撤权验证，不代表 G35 完成。

测试 `TestNativeAttachmentRootRevocationPropagatesAcrossWindows` 使用 `t.TempDir()` 创建入口账本和位于默认账本目录之外的 `receipt.pdf`，并创建真实 Darwin 普通目录 bookmark。两个独立 MyGo Content 窗口使用同一个隔离偏好 Store，打开同一账本；初始时两个 Session 都将附件标为“不在允许的附件目录中”。

测试通过窗口 A 的附件策略流程把 bookmark 加入账本策略，等待偏好 generation 与两个 Session 更新，并确认两个原生附件页都显示文件“可用”。随后在窗口 B 的附件目录管理页，通过 macOS Accessibility 点击“移除”。测试验证偏好中的 bookmark 已删除、两个 Session 的附加 roots 均为空；两个窗口重新读取附件声明后都显示越权状态。窗口 A 在撤权前保存的附件 DTO 再次提交时没有调用系统打开适配器。

这轮还修复了失效 bookmark 的重新授权路径：选择同一路径时，若旧记录被标记为失效，就用新创建的 bookmark 替换旧身份；有效同路径仍报告重复，默认账本目录仍不可添加。新增合并逻辑回归测试覆盖替换、重复、新增和默认目录四种情况。

本测试验证跨窗口撤权和迟到附件操作失效。添加 bookmark 通过真实 Darwin bookmark 创建及 Model 策略入口完成。后续 `TestNativeAttachmentRootOpenPanelIsOwnedAndCancellable` 已验证真实 `NSOpenPanel` 作为所属窗口 sheet 打开，并且取消后不改变 Session 或持久化策略；该场景连续5次通过。`TestNativeAttachmentRootSelectionPersistsAndAuthorizesDocument` 在真实 MyGo Content 窗口通过原生“添加目录…”动作，再从隔离的 picker adapter 返回临时目录；它使用真实 Darwin bookmark 创建/解析、workspace Configure 和临时偏好 Store，验证附件由 `OutsideRoot` 变为 `Available`。成功路径连续5次通过，但没有模拟 NSOpenPanel 内部的鼠标/Accessibility 选择动作。真实面板目录选中、宽范围目录确认、GUI 重启后的恢复及失效 bookmark 从原生面板重新授权仍待验收，因此 G35 继续进行中。

验证命令：

```sh
GOWORK=off CGO_ENABLED=0 go test ./internal/app \
  -run '^TestMergeAttachmentRootBookmarkReplacesOnlyInvalidDuplicates$' -count=1 -v
ORANGECOUNT_GOUI_NATIVE_E2E=1 GOWORK=off CGO_ENABLED=1 \
  go test -tags orangecount_e2e ./internal/app \
  -run '^TestNativeAttachmentRootRevocationPropagatesAcrossWindows$' -count=5 -v
make goui-native-e2e
make goui-check
GOWORK=off CGO_ENABLED=1 go test -race ./... -count=1
GOWORK=off CGO_ENABLED=0 go vet ./...
```

书签合并单测通过；定向原生场景连续 5 次通过；新增场景后完整常规原生窗口套件 25 项通过。`make goui-check`、race 和 vet 通过。独立十分钟 soak 不属于该常规套件。测试只使用临时账本、临时附件、临时偏好和真实原生窗口，不访问私人账本或用户偏好。

## 2026-10-08 原生目录面板取消补充

新增 `TestNativeAttachmentRootOpenPanelIsOwnedAndCancellable`，使用临时账本与隔离偏好，从附件目录管理页通过原生 Accessibility 点击“添加目录…”，确认真实 `NSOpenPanel` 可见且作为当前 Content 窗口的 sheet；取消后检查面板关闭、选择状态清除，且 workspace roots 和持久化 bookmark 仍为空。该场景连续5次通过；最新全量 gate 后常规原生窗口场景数为53项，另有1项独立十分钟 soak。这个结果覆盖面板呈现、所属窗口与取消语义，不代表成功选目录并保存授权已验收。

```sh
./apps/goui/native-e2e.sh -run '^TestNativeAttachmentRootOpenPanelIsOwnedAndCancellable$' -count=5 -v
make goui-native-e2e
make goui-check
```

测试使用临时账本和临时偏好；不访问私人账本或用户偏好。

新增成功路径复验：

```sh
./apps/goui/native-e2e.sh -run '^TestNativeAttachmentRootSelectionPersistsAndAuthorizesDocument$' -count=5 -v
```

该场景核对 Dialog 选项仍将当前窗口作为 Parent、启用目录选择并使用预期标题；返回路径后检查规范化 Session root、Store 中的 bookmark 可解析、附件状态变化及 Accessibility 页面状态。测试入口现有55个：54项常规窗口场景和1项独立十分钟 soak。完整原生 gate 与 `make goui-check` 已在加入该场景后通过。

额外验证：`GOWORK=off go test -race -count=1 ./...` 和 `GOWORK=off CGO_ENABLED=0 go vet ./...` 通过；授权成功的 native E2E 场景单独以 race 模式运行也通过。`git diff --check` 通过。


## 2026-10-08 启动恢复与失效 bookmark 重选补充

新增 `TestNativeAttachmentRootRestoresBeforeDocumentRead`：先把真实 Darwin bookmark 写入隔离偏好文件，再以全新的 preferences Store、Model 和 MyGo Content 窗口打开账本；验证启动恢复把附加目录加进 Session policy，附件状态为 `Available`，且恢复过程没有改写 generation 或 bookmark。新增 `TestNativeInvalidAttachmentBookmarkCanBeReselected`：移动已 bookmark 的目录并在同一规范路径新建目录，验证新 Model 将旧文件身份标为失效、未授权附件；从附件目录管理页重新选择同一路径后，新 bookmark 替换旧身份、偏好 generation 前进，附件恢复为可用。两项原生场景分别连续5次通过，组合运行5轮通过；带 race detector 的定向原生场景通过。

完整 `make goui-native-e2e` 通过（56项常规窗口场景，37.165秒），`make goui-check` 和 `git diff --check` 通过。native 测试文件目前有57个 `TestNative...` 入口：56项常规场景与1项独立十分钟 soak。fresh Model/Store 测试模拟应用重新初始化，但没有启动打包 app 新进程；真实 `NSOpenPanel` 内的目录选中和宽范围目录确认仍未验收。另一次仅用于测试的 `NSAlert` 按钮探针错误地向 `_NSAlertPanel` 发送 `buttons` selector 并导致 E2E 进程异常退出；探针与对应实验用例已删除，未修改生产界面、用户偏好或用户文件，因此宽范围确认没有被标为通过。

```sh
./apps/goui/native-e2e.sh -run '^TestNative(AttachmentRootRestoresBeforeDocumentRead|InvalidAttachmentBookmarkCanBeReselected)$' -count=5 -v
ORANGECOUNT_GOUI_NATIVE_E2E=1 GOWORK=off CGO_ENABLED=1 go test -race -tags orangecount_e2e -run '^TestNative(AttachmentRootRestoresBeforeDocumentRead|InvalidAttachmentBookmarkCanBeReselected)$' -count=1 ./internal/app
make goui-native-e2e
make goui-check
```


## 2026-10-08 Manual QA 包真实面板选择与进程重启

通过 `cua_repl` 操作隔离的 `OrangeCount Go UI Manual QA.app`（bundle ID `org.orangecount.desktop.goui.manualqa`），创建 `/tmp/orangecount-g35-cua.ONB6DY` 下的临时账本和 `attachments/receipt.pdf`。在原生 UI 打开“附件目录”页，启动真实 `NSOpenPanel`，用 macOS“前往文件夹”输入临时附件目录并按“打开”；界面显示规范化后的 `/private/tmp/.../attachments` bookmark。随后打开引用该 receipt 的临时账本，原生附件页显示 `receipt.pdf` 状态为“可用”。退出该 QA `.app` 并从同一 bundle 路径重新启动，重新打开欢迎页中的该临时账本后，附件仍显示为“可用”，证实该 QA 包的磁盘偏好在真实进程重启后恢复。

验收后通过 UI 移除临时附件授权、退出 QA 进程，并从 Manual QA 偏好文件中只删除本次测试创建的 ledger identity 和 recent path；临时账本/附件目录亦已删除，复核确认对应记录与目录均不存在。该手工 UI 结果补足了真实面板选择与该 QA bundle 的进程重启恢复。它不等同于 G43 development bundle 的首次启动/完整读写循环，也不验证开发/QA 身份迁移矩阵。宽范围根目录确认尚未执行；没有选择或授权用户主目录、`/Users` 或系统根目录。
