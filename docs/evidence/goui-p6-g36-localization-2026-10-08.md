# G36 本地化与焦点更新

日期：2026-10-08。状态：实现与 Headless 验收通过；双窗口 Cmd+F 原生路由已补证，其他原生全窗口检查及 G37 的 IME/VoiceOver 场景仍待完成。

## 已完成

- en/zh-CN 文案覆盖 Go UI 页面、原生菜单、文件面板、确认面板、图表、分页、持仓、指令及查询结果类型标签；中文仍为默认界面语言。
- 账本偏好保存界面语言，当前窗口切换语言后刷新菜单和页面；所有读取请求携带所选语种。
- 动态账本账户、商品/币种、用户查询和预设名称、金额、文件名及来源路径保留原值。下拉框只翻译显式登记的应用选项，并把选择映射回原始状态值。
- Cmd+F 焦点矩阵覆盖资产负债表、损益表、流水、账户流水、持仓、指令和查询页面；所有用例在 900×600、1.5 字号下运行。
- 资源测试检查所有静态 `tr(...)` 英文条目存在且不含汉字、目录键唯一；另静态检查常用原生文字控件的中文字面量必须经翻译目录或已知本地化 helper。

## 验证

- `cd apps/goui && GOWORK=off go test ./...`：通过。
- `cd apps/goui && GOWORK=off go test -race ./internal/app`：通过。
- `cd apps/goui && GOWORK=off go vet ./...`：通过。
- `cd apps/goui && GOWORK=off go build ./cmd/orangecount-goui`：通过。
- 新增 Headless 测试：全页搜索焦点矩阵、英文语言持久化/请求语种、页面与导出动态文案、类型化查询标签，以及与翻译资源同名的动态账户值保真。

本记录的语言/缩放矩阵覆盖可重复的 Headless 与构建证据。双窗口 Cmd+F 原生菜单路由见[G36 双窗口 Cmd+F 原生验证](goui-p6-g36-native-find-2026-10-08.md)。真实系统输入法和 VoiceOver 不由这些测试代替；此前原生 QA 窗口捕获遇到 ScreenCaptureKit `SCStreamErrorDomain -3811`，详见[G36 首轮菜单与焦点记录](goui-p6-g36-2026-10-07.md)。


## 表格错误提示本地化补充

静态检查发现持仓表格和商品表格构建失败时直接拼接中文前缀，英语界面会混入中文。两处现在使用应用翻译目录，Go error 后缀保持原样；资源测试覆盖两个动态错误前缀。

验证通过：

```sh
cd apps/goui
env GOWORK=off CGO_ENABLED=0 go test ./internal/i18n ./internal/app -count=1
env GOWORK=off CGO_ENABLED=0 go vet ./...
cd ../..
make goui-check
OUT_DIR="$PWD/build/goui-package-check-2-2026-10-08" VERSION=0.0.0-check2 ./apps/goui/package.sh
```

打包命令成功生成临时版本 `.app`、ZIP 与 SHA-256；`Info.plist` 检查通过。该修补不替代 G36 尚未完成的原生菜单/深浅色/窗口尺寸矩阵。

## 图表无障碍文案英文回归

手动打开英文界面时发现，页面标题与金额已翻译，但图表的辅助功能名称/说明以及 exact-value 表格的“期间”列仍是中文。图表组件现在接受应用翻译函数：净资产折线图、现金流分组柱图的名称、键盘操作说明、选中期间说明、缺失状态及表格列标题均按当前语言呈现；账本期间、金额、币种和来源明细保持原值。没有 translator 的可复用图表 API 保留原有简体中文默认值。

i18n 资源测试现解析图表源文件，要求每个 `chartText(...)` 中的静态中文 UI 文案都有英文目录项。复核 dashboard 时还发现缺失净资产点把 exact 状态固定写成 `Unavailable`；现改用 `m.tr("不可用")`，Headless 回归确认中英文页面分别显示对应状态。

验证：

- `env GOWORK=off CGO_ENABLED=0 go test ./internal/charts ./internal/i18n ./internal/app`：通过。
- `make goui-check`：Go UI 全模块测试和独立构建通过。
- `make goui-native-e2e`：完整原生窗口 E2E 通过；现金流原生窗口测试核对折线图键盘选期与中点鼠标命中、现金流柱图键盘选期与鼠标命中，并切换英文确认图表 AX 名称为 `Grouped bar chart`、help 为英文且不含汉字、表头出现 `Period` 且不再出现 `期间`。
- `./apps/goui/native-e2e.sh -run '^TestNativeDashboardCashflowChartKeyboardAndAccessibilityHelp$' -count=5`：最新折线/柱图操作与语言切换场景连续5次通过。
- 从当前源码生成独立临时 development `.app` 并打开公开 `cashflow.bean`。实际 Accessibility 树显示 `Net Worth line chart`、`Grouped bar chart`、英文键盘帮助和 `Period` 表头；关闭该临时窗口后，原有手动 QA 窗口未受影响。

此记录生成时，Cmd+O/Cmd+R 原生路由仍待覆盖；后续已补双窗口焦点与面板归属回归，详见[G36 Cmd+O/Cmd+R 原生验证](goui-p6-g36-native-open-reload-2026-10-08.md)。G36 仍为“已实现待验证”：新建窗口菜单的原生创建、深浅色及 900×600/1.5 字号原生矩阵尚未全部覆盖；本补充只关闭发现的图表本地化缺陷，不代表 G36、G37 或 G38 验收完成。
