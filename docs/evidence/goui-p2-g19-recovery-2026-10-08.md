# G19 Go UI invalid/stale 修复恢复闭环

日期：2026-10-08（Asia/Singapore）。在真实 `DesktopEngine`、临时公开合成账本和 MyGo headless renderer 上验证同一工作区的错误诊断与修复恢复。

## 场景与结果

- 打开一笔交易的有效账本，读取财务概览并由 MyGo tester 确认页面显示原金额。
- 修改同一文件为语法错误，等待 watcher 发布 stale 状态，重新读取当前无效快照，确认 UI 显示诊断页、stale 文案和 `main.bean:1:1` 来源位置。
- 修复原文件，通过页面上的“重新读取”按钮触发真实 `workspace.IntentReload`；确认 workspace path/epoch 不变，再读取财务概览，MyGo tester 显示修复后的 30 USD 和两笔交易。
- 定向测试 `TestGoUIRendersCurrentInvalidDiagnosticsAndSameWindowRecovery` 通过（约 0.10s）；本轮 `make goui-check`、Go UI 全量 race 与 vet 通过。

每个独立 headless Tester 使用新的 Model，避免 MyGo `ListState` 在 Tester 间复用。这个证据覆盖核心 watcher/workspace 与 Go UI 的 headless 渲染/按钮闭环；尚不替代已打包应用中的真实 macOS 窗口、Launch Services、输入法或 VoiceOver 验收。

## 连续诊断修订与位置复制

随后扩展同一测试：在首个 parse error 已经显示并复制精确位置后，再通过真实 `DesktopEngine` 将同一文件改成另一处语法错误，等待 watcher 发布新修订并重新读取。MyGo tester 确认第二个行列位置替换了旧位置，复制结果采用核心当前诊断返回的完整路径、行和列。独立诊断页面测试同时确认“复制关联位置”使用关联来源本身的位置，而不是主诊断位置。

```sh
cd apps/goui
env GOWORK=off CGO_ENABLED=1 go test ./internal/app \
  -run '^(TestDiagnosticsPageDistinguishesStaleAndCopiesCurrentLocation|TestGoUIRendersCurrentInvalidDiagnosticsAndSameWindowRecovery)$' \
  -count=1
```

以上通过。随后扩展既有原生同窗口恢复场景：点击诊断来源行的“复制位置”，并从系统剪贴板核对核心投影的完整绝对路径、行和列。初版探针使用通用辅助技术名称；诊断列表中的多个来源按钮同名，原生测试可能命中错误行。现在复制、定位、打开动作的 Accessibility 标签会附上具体路径和行列，改善 VoiceOver 上下文并使测试目标唯一。全套原生窗口 gate 暴露切换页面时的间歇失败：AppKit Accessibility 树可能仍保留欢迎页内嵌诊断卡片中即将退出的按钮节点；AXPress 将点击记录在该节点上，随后节点被当前帧回收，鼠标点按当前页按钮则成功。测试现在采样 Accessibility 节点的 MyGo ID，并等待同一按钮连续8个50毫秒间隔样本保持唯一稳定，再通过当前诊断行的 AppKit `AXPress` 触发复制。最终 `./apps/goui/native-e2e.sh -run '^TestNativeInvalidDiagnosticsRecoverInTheSameWindow$' -count=10` 连续10次通过（20.94秒），并从系统剪贴板核对精确绝对路径、行和列。Headless Tester 的主/关联来源复制检查也通过。包含此稳定性修正后的完整 gate 结果另见本记录后续增补。

系统剪贴板的当前诊断精确复制已获稳定节点上的原生重复验证。Finder/source-editor 应用启动、面板中实际选择编辑器、真人 VoiceOver直接激活和其它多窗口 sheet 仍待原生验收。

## 稳定化后的完整 gate

完成 Accessibility 稳定节点等待后，整套 Go UI 检查和当前全部原生窗口场景通过：

```sh
git diff --check
make goui-check
make goui-native-e2e
```

`make goui-check` 完成 Go UI 测试与应用构建；`make goui-native-e2e` 以退出码 0 完成当前40个常规原生窗口场景。独立十分钟 reload soak 不属于该快速 gate。全量运行中的 G19 剪贴板复制不再超时；此前独立 `TestNativeInvalidDiagnosticsRecoverInTheSameWindow -count=10` 仍连续通过。

G25 新增原生 BQL 编辑菜单 Copy/Paste 测试后再次运行以上两个 gate，结果仍全部通过。该测试通过 AppKit 选择/拷贝/粘贴菜单处理含中文和 emoji 的查询，核对全文复制、选中替换及不隐式执行，连续5次通过。它先快照 general pasteboard 的所有项目、类型顺序与字节，并在清理时恢复；若遇到无法读取为稳定字节的延迟内容，则在更改剪贴板之前跳过。
