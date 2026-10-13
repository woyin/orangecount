# G15/G16 账户树搜索与同账本重载证据

日期：2026-10-08。范围：旧 GUI 场景 11 的账户树搜索、展开、选择与重载。此记录只证明自动化 UI 和公开 fixture 行为，不代表全量 GUI、VoiceOver 或旧版18场景验收完成。

## 行为

- 深层账户搜索会临时展开命中路径，搜索仍只筛选账户行，核心汇总和币种列保持原投影。
- 清除搜索恢复搜索前的展开状态。搜索期间显式点“全部展开”或“全部折叠”会更新清除搜索后恢复的状态。
- 展开状态按报表类型和账本路径维护；同一账本产生新快照时，仍存在的账户展开键和所选完整账户名会保留，不存在的分支键会清理。
- 切换报表类型或账本仍会初始化新的树上下文并清除旧选择。

## 验证

- Headless 回归覆盖深层搜索的临时展开、清除后恢复原折叠、搜索时显式全部展开，以及同账本新快照后保留折叠和账户选择。
- `./native-e2e.sh -run '^TestNativeAccountTreeSearchSelectionSurvivesReloadAndRestoresCollapse$' -count=5 -v`：5次通过。原生 MyGo 窗口使用公开 `account-detail.bean` 的临时副本；实际搜索并选择 `Assets:Test:Child`，在源文件前插入注释触发自动重载，核对同一账户仍可见且被选中，清除搜索后回到原折叠状态。
- `./native-e2e.sh -race -run '^TestNativeAccountTreeSearchSelectionSurvivesReloadAndRestoresCollapse$' -count=1 -v`：通过；测试通过 `mygo.RunOnMain` 同步读写窗口模型状态。
- `make goui-check`：通过。
- `make goui-native-e2e`：完整原生窗口套件通过。

原生场景没有把模拟的模型折叠动作记成用户实际点击；全部展开/折叠控件本身由 Headless UI 测试验证。真人键盘树导航、VoiceOver，以及旧版18场景的完整 GUI 等价矩阵仍未验收。
