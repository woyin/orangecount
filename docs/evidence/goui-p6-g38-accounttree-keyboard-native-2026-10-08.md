# G38 账户树原生方向键导航证据

日期：2026-10-08。范围：旧 GUI 场景 15 的资产负债表账户树键盘导航。

原生 MyGo 窗口打开公开 `account-detail.bean` 的临时副本，切换到资产负债表并折叠全部分支。实际点击 `Assets` 行后，Right Arrow 展开父账户，再次按右键选择首个子账户；继续按右键展开 `Assets:Test`。Left Arrow 依次折叠该分支、返回父账户、折叠父账户。每一步核对可见行数、展开状态和选中账户；最终状态仍选中 `Assets`，根行保持可见。

原生树行与分类摘要存在相同账户标签，因此测试先按稳定 AX 节点身份找到并点击 `AXRow`，确保键盘事件作用于真实行。

验证结果：

- `./apps/goui/native-e2e.sh -run '^TestNativeAccountTreeArrowKeysNavigateAndExpandHierarchy$' -count=5 -v`：5次通过。
- `./apps/goui/native-e2e.sh -race -run '^TestNativeAccountTreeArrowKeysNavigateAndExpandHierarchy$' -count=1 -v`：通过。
- `make goui-check`：通过。
- `make goui-native-e2e`：通过，62个原生用例通过，3个条件门控用例跳过。
- 测试只读取公开 fixture，并在临时目录创建副本。

这补足了账户树在真实窗口中的方向键行为；真人 VoiceOver、真人输入法和完整18场景等价矩阵仍未验收。
