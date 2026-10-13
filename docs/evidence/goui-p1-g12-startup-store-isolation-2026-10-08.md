# G12 应用启动偏好迁移与 QA 隔离

日期：2026-10-08。范围：验证正式应用启动使用的存储组装路径会迁移旧 defaults，QA 启动跳过旧域，并将偏好与命名查询预设放入指定的应用数据目录。

`cmd/orangecount-goui/main.go` 现调用 `openApplicationStores`；该函数把 user-data 路径、QA 身份和旧 defaults opener 明确作为输入。生产模式使用 `org.orangecount.desktop` 只读导入最近账本、profile 与 reading 设置，保存版本迁移标记并关闭 defaults reader。QA 模式不会调用 opener。两个 Store 使用独立的 `preferences-v1.json` 与 `query-presets-v1.json` 文件名。

测试使用公开 `cashflow.bean` fixture、fake legacy defaults 和 `t.TempDir()`：确认生产启动函数导入最近路径、v3 profile、USD/market-value 设置并写入迁移 marker；查询预设可单独保存；两个文件及其父目录权限受限。QA 测试令 opener 失败并计数，确认它从未被调用，且空偏好没有迁移 marker 或最近路径。测试没有读取或修改用户 defaults。

```sh
cd apps/goui
GOWORK=off go test ./cmd/orangecount-goui -count=1
```

该启动组装的定向测试通过。它覆盖与 `main()` 共用的 Store 创建/迁移函数，不等同于从安装包冷启动并验证 macOS `PathUserData`、首次运行和进程重启；G43 的实际包验收仍需完成。
