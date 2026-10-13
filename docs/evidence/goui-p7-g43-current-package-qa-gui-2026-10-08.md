# G43 当前源码包隔离 QA 界面验收

日期：2026-10-08（Asia/Singapore）。验证 G43 当前源码 arm64 development package 的可见启动、明确打开公开账本和退出。

## 包身份

- 原始 `.app`：`/tmp/orangecount-goui-g38-current-20261008/OrangeCount Go UI Development.app`。
- ZIP：`OrangeCount-Go-UI-Development-0.0.0-g38-current-arm64.zip`，SHA-256 `9231f8c85cec69e536921a620f33afd109724437c431cf840fc588107f1a425b`。
- 包已通过 `make goui-package-verify OUT_DIR=/tmp/orangecount-goui-g38-current-20261008 VERSION=0.0.0-g38-current GOARCH=arm64`；build-info 指向 source revision `1c2096d435e07cc9b5a5957e0e974546ee1387a6`，工作树标记为 dirty。
- 为绕开桌面上已运行的同 Bundle ID 应用，将包复制到一次性 QA 路径，仅更改副本 `Info.plist` 的 Bundle ID 与显示名称；应用 executable 保持原包字节。使用 `ORANGECOUNT_QA=1` 和唯一隔离 user-data 路径启动。

## 实际界面操作

隔离窗口显示“QA 模式：不会自动恢复最近账本”，欢迎页没有最近账本列表。通过“打开账本”进入原生 Open Panel，以“前往文件夹”定位仓库公开 fixture `testdata/fixtures/native/cashflow.bean` 并明确选择打开。读取成功后，真实窗口显示 3 个账户、2 笔交易和 5 条分录；财务概览显示净资产 `700 USD`、收入 `1000 USD`、支出 `300 USD`、净流量 `700 USD`，无解析诊断。Accessibility 树同时可见这些精确文本与相应报表行。

通过 Cmd+Q 关闭隔离窗口；CUA 返回该应用已退出，按唯一临时 executable 路径查询时也没有存活进程。桌面上此前运行的 development app 未被选择、关闭或改动；没有访问最近账本或执行写入。

## 范围

这补上了当前源码包的 QA 冷启动欢迎页和公开账本读取可见证据，并消除了先前 CUA 绑定到旧 Bundle ID 实例所造成的歧义。没有覆盖该包上的查询/导出、错误恢复、重启偏好恢复或旧偏好迁移；此前 r7 包的这些证据仍对应 r7，不向当前包借用。G43 继续进行中。

## 后续适用范围修订

随后实施 G38 持仓筛选修复，生成了 `0.0.0-g38-holdings-fix` r1 包。因此本文 GUI 操作仍准确对应上文列出的 `0.0.0-g38-current` ZIP/executable，但不再代表最新源码包。r1 的独立 package-verifier 结果见[G43 holdings-fix package 校验](goui-p7-g43-holdings-fix-package-2026-10-08.md)；r1 尚无可见 GUI 结果。
