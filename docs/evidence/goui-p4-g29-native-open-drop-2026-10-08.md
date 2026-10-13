# G29 Finder 打开与原生拖放路由

日期：2026-10-08。本记录覆盖 `.ocanalysis` 的应用事件路由、窗口拖放和 macOS bundle 声明；不表示 G29 或 Go UI 迁移完成。

## 实现

- MyGo `App.OnOpenFile` 事件按扩展名分流：`.ocanalysis` 交给离线分析 reader，其余文件继续沿用账本打开路由。扩展名比较不区分大小写。
- 原生窗口通过 `Window.OnFileDrop` 接收 Finder 拖放，并使用同一文件路由。账本仍遵守“空欢迎窗口复用、已有账本另开窗口”的规则；离线捕获可叠加到当前活动账本窗口，不读取或改变其 workspace。已有离线捕获仍在读取或已显示时，再打开另一份捕获会创建独立窗口。
- 离线文件后台读取期间标记 loading，后到的读取结果不会覆盖较新的打开请求；reader 继续限制普通文件、查询字节预算并校验 archive 内容。导入失败显示在查询页，不启动账本读取或捕获查询。
- QA 和开发 bundle 声明导出的 `org.orangecount.query-analysis` UTI、`.ocanalysis` 扩展与 Viewer 文档角色。QA handler rank 为 `Alternate`；开发包 handler rank 为 `Default`。

## 验证

- `TestNativeFinderOpenAndWindowDropReadOfflineAnalysis` 连续运行 3 次通过。首份捕获通过 MyGo application delegate 打开并进入现有欢迎窗口的查询页；第二份通过 MyGo/AppKit 文件拖放进入新窗口。断言两个窗口均没有活动账本路径，第一份捕获仍保留在原窗口，第二份显示自己的捕获身份；随后从同一 delegate 打开损坏 archive，应用在隔离窗口显示读取错误且不打开账本。扩展后还通过原生 Accessibility 返回/前进离线捕获页，确认成功捕获与损坏文件错误均可恢复，workspace 始终没有活动账本路径。
- `make goui-check` 通过。
- `make goui-native-e2e` 通过，完整原生 E2E 用时约 21 秒。
- `GOWORK=off CGO_ENABLED=1 go test -race ./... -count=1`、`GOWORK=off CGO_ENABLED=0 go vet ./...`（均在 `apps/goui`）和 `make license` 通过。
- `plutil -lint apps/goui/qa/Info.plist`、`sh -n apps/goui/package.sh apps/goui/prepare-qa.sh` 与 `git diff --check` 通过。
- `package.sh` 在 `/tmp` 临时输出目录成功生成开发 `.app`、zip 与 SHA-256 文件；生成包内的 `Contents/Info.plist` 通过 `plutil -lint`。

## 未完成

原生事件回调与窗口拖放已由 macOS E2E 覆盖，但尚未从 Launch Services 实际双击文件启动 bundle；真实历史版本文件语料、完整键盘/VoiceOver 和其他 macOS 版本仍待验收。以上自动化合成文件验证不能代替这些系统级验收。
