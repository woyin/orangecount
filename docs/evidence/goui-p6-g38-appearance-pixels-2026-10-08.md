# G38 深浅色原生渲染像素验证

日期：2026-10-08（Asia/Singapore）。对应旧 GUI 场景 17 的 Light/Dark 像素颜色检查。

测试在真实 MyGo macOS Content 窗口中打开公开 `native/cashflow.bean`，窗口大小为 900×600。通过 `Window.CapturePage` 捕获 Go UI 当前绘制内容，分别设置 Light 与 Dark appearance；对每张 PNG 按相同位置抽取 48×32 网格，共 1536 个采样点。亮度按 Rec. 709 加权 RGB 计算，像素变化按 RGB 三通道绝对差之和至少 60 计入。

结果：平均亮度从 Light 的 241.2 降至 Dark 的 40.6；1536 个采样点中 1518 个发生明显变化。断言要求亮度至少相差 40，且至少 5% 采样点变化。本次实测约 98.8% 的采样点变化。外观恢复为测试启动前的应用 appearance。

同时修正查询结果复制原生测试的剪贴板同步判定：原测试仅等待通用 JSON 前缀，系统剪贴板中的旧查询 JSON 会使轮询提前结束。现在测试先写入唯一哨兵，并等待内容精确等于当前选中行的 typed JSON；读取主线程模型字段则改用同步 `RunOnMain`，消除测试自身的数据竞争。复制场景连续 5 次通过，和像素场景一起运行的 race 定向测试通过。

## 验证

- `./apps/goui/native-e2e.sh -run '^TestNativeAppearanceRendersDistinctLightAndDarkPixels$' -count=3 -v`：3 次通过。
- `./apps/goui/native-e2e.sh -run '^TestNativeQueryResultSortAndKeyboardSelectionKeepRowIdentity$' -count=5 -v`：5 次通过。
- `./apps/goui/native-e2e.sh -race -run '^TestNative(AppearanceRendersDistinctLightAndDarkPixels|QueryResultSortAndKeyboardSelectionKeepRowIdentity)$' -count=1 -v`：通过；像素样本为亮度 241.2→40.6、1518/1536 点变化。
- `make goui-native-e2e`：完整原生窗口 gate 通过，65.608 秒。
- `make goui-check`：通过。

该图像来自 MyGo Content 绘制缓冲区，不代表显示器色域/亮度校准或真人 VoiceOver 朗读。旧场景 17 其他尺寸的布局、完整辅助功能矩阵以及旧版 18 项 GUI 全矩阵仍未全部验收；G37/G38 继续进行中。
# 2026-10-08 重复运行稳定性补充

原像素场景在完整原生门禁偶发超时。单独运行 `TestNativeAppearanceRendersDistinctLightAndDarkPixels -count=5` 时，复现了第5轮暗色等待超时；带计数的探针在后续 `-count=10` 中又复现4轮。失败样本的截图 API均成功，`Theme.IsDark()`持续为真，但浅色阶段亮度已经是暗色的40.6；测试只检查 `Theme.IsDark()==false` 和非空截图，因此偶尔把上一帧的暗色像素错误保存为浅色基线，之后自然永远等不到比它更暗的帧。MyGo 的 `Theme.SetSource` 会安排 Content 主题更新，故这是测试等待条件不足，不是产品主题绘制回归。

现将浅色基线条件收紧为主题状态已是 Light 且实际 Content 像素平均亮度至少150，再采集 Dark 并验证相对差异；未修改产品主题逻辑。修正后的像素场景连续10次通过，定向 race 检查通过，包含该修正的完整 `make goui-native-e2e` 通过（78.646秒）。

```
./apps/goui/native-e2e.sh -run '^TestNativeAppearanceRendersDistinctLightAndDarkPixels$' -count=10 -v
./apps/goui/native-e2e.sh -race -run '^TestNativeAppearanceRendersDistinctLightAndDarkPixels$' -count=1 -v
make goui-native-e2e
```
