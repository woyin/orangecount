# Go UI G39 性能基准首轮记录

日期：2026-10-08（Asia/Singapore）。本记录包括 headless Go UI 与核心层样本，不等同 macOS 原生窗口交互验收。

## 固定条件

- 机器：Apple M4，10 逻辑 CPU；macOS 27.0.1（build 26A434），Darwin arm64。
- 工具链：Go 1.27.1，`CGO_ENABLED=0`、`GOWORK=off`，默认优化编译并使用生产构建的 `-trimpath -ldflags=-buildid=`。
- MyGo：`v0.2.16-0.20261006170038-869768d02d91`，对应修订 `869768d02d91`。
- 输入：`tools/perfbench` 固定 seed 生成的合成账本；基础普通交易 52,500 笔，另含固定的 lot 买卖记录。语义 profile 为 v3，SHA-256：`a1aab9cbffeb1047d32a3a33bd6c98ce10150d7971104d72760b58147903f195`。未读取私人账本。
- 生产页面使用 1180×780 DIP；图表基准为 900×600 DIP，device scale 1。交易页读取100行；图表保留10,000原始点。
- 运行入口：`apps/goui/benchmark.sh`。原始输出位于本机忽略目录 `build/benchmarks/g39-final/`；脚本会在输出目录保存 metadata 与原始 Go benchmark 结果。

## 观测结果

以下图表帧统计每项100帧、重复10轮；区间是十轮各自分位数的最小值到最大值。核心查询读取50次；账本打开10次。

| 场景 | 结果 |
|---|---|
| Welcome headless 帧 | p95 19.1–26.7 µs；p99 19.3–30.1 µs；每轮100帧 |
| 52,500 基础交易账本的交易页 headless 帧 | p95 112.5–127.0 µs；p99 119.2–167.2 µs；每轮100帧；视图仅呈现100行分页 |
| 10,000 点折线、分组柱与精确表格 headless 帧 | p95 4.08–5.28 ms；p99 4.21–5.88 ms；每轮100帧。每帧约6.48 MB、1,439次分配；单轮最大值17.82 ms |
| 核心首次打开 | 10次样本；p95/p99 1.902 s；范围1.784–1.902 s；Go benchmark累计分配约3.81 GB/op（不是RSS） |
| 核心交易页读取100行 | 50次样本；p95 34.79 ms、p99 42.06 ms；范围23.01–42.06 ms；约27.85 MB/op（不是RSS） |

用相同机器和输入的首个图表探针作方向性比较：精确表格原先为每个期间建一个map；改为每个序列建一个label索引后，单次样本的headless帧由5.42 ms降为4.05 ms，分配由9.65 MB/21,372次降为6.49 MB/1,440次。首轮与最终轮的样本数不同，此比较用于说明优化方向，不作为性能gate。

### 原生窗口分页与开账本延迟（初步）

- 在macOS 27.0.1 / Apple M4 / Go 1.27.1上，通过 `./apps/goui/native-e2e.sh -run '^TestNativeJournalPagingProjectsOnlyOneHundredOf52500Transactions$' -count=5 -v` 重复5次。每次在Go测试进程已运行的前提下生成并写入一个52,500笔、两账户的合成账本；计时从账本文件写完后开始，因此不含生成/写盘，也不含进程冷启动。
- 从创建窗口/打开账本到仪表盘内容出现在原生Accessibility树：0.882–1.007 s；从财务概览点击“流水”到第1页请求完成且“显示 1–100”进入Accessibility树：46.8–67.0 ms；翻到第2页并看到“显示 101–200”：49.2–70.6 ms。每次均同时核对52,500总数、100行页投影和相同SnapshotID。
- 这是5次真实AppKit窗口端到端观测值的范围，不是p95/p99，也不是进程冷启动、屏幕绘制到present、滚动帧或稳定性能门槛；AX树轮询属于计时终点的一部分。它仅补足52,500笔账本打开后分页交互的初步时延证据。

### 原生开发包空闲进程抽样

- 2026-10-08 03:08:03–03:08:33（Asia/Singapore），由实际打包的 `org.orangecount.desktop.goui.dev` arm64 应用打开公开 `cashflow.bean` 后静置 31 秒；PID 60327 指向 `/tmp/orangecount-goui-dev-delivery-20261008/OrangeCount Go UI Development.app`，采样期间未发送交互。
- `top -l 31 -s 1 -pid 60327 -stats pid,cpu,state` 共采得31个每秒 CPU 样本：最小 0.0%、均值 0.28%、最大 1.8%；`ps` 同步采样 RSS 最小 106.2 MiB、均值 107.5 MiB、最大 114.1 MiB。31个进程状态样本中30次为 sleeping、1次为 running。
- 数据只代表本机 macOS 27 上、单个公开小账本、单窗口、短时间静置的开发包。CPU 百分比是 top 的采样值，不包含用户交互输入到 present 的时延；RSS 范围也不足以证明长期平台期或与 SwiftUI 的同输入比值。

### 52,500 笔账本隔离 QA 进程抽样

- 2026-10-08 03:47:32–03:48:02（Asia/Singapore），由 Launch Services 启动仓库外隔离的 `OrangeCount Go UI Perf QA.app`，bundle ID 为 `org.orangecount.desktop.goui.perf.20261008`，PID 57274；账本为 `tools/perfbench` 固定 seed 生成的 52,500 笔公开合成输入，SHA-256 与上方固定基准相同：`a1aab9cbffeb1047d32a3a33bd6c98ce10150d7971104d72760b58147903f195`。采样前已完成首次打开并静置，采样期间没有发送交互。
- `top -l 31 -s 1 -pid 57274 -stats pid,cpu,state` 得到31个CPU样本：均值0.20%、最大1.6%；31次状态均为 sleeping。同步的31个 `ps` RSS样本均为698,544 KiB（682.2 MiB）。
- 该短样本期间主机可用内存约96–100 MiB、压缩内存约8.6 GiB，系统内存压力较高；结果只描述当前机器和隔离 QA 构建，不能外推为生产包阈值或长时平台期。
- 文件系统活动未能采集：本机 `fs_usage` 明确要求 root 权限，本次没有提升权限。因此这组 CPU/RSS 数据不能证明静置时无文件内容读取；冷启动、首屏、交互/滚动帧、取消反馈及 SwiftUI 同输入对照也仍缺少原生结果。

## 结论与缺口

- headless painter、虚拟化交易表格、核心开账本和分页读取已有可复跑入口；当前样本可用于观察 CPU 侧布局/软件绘制与核心投影时间。
- headless `ui.Tester` 在内存软件渲染。其 p95/p99 不能代替 Metal/GPU、原生窗口输入到present时延或刷新率；Welcome 帧不是进程冷启动测量。
- 现在已有公开小账本与52,500笔合成账本两组原生进程 idle CPU/RSS 短样本，以及上述5次真实窗口开账本/翻页到Accessibility可见的时延范围；仍未检查空闲期间文件内容读取，也未采集屏幕绘制到present/滚动帧、取消反馈、长时间RSS平台期或同输入SwiftUI baseline。新增的20次workspace open/close与60次reload是G40的Controller/Session测试，不能替代20个原生窗口或10分钟进程soak。因此 R13 与 G39 仍未通过，任务状态保留“进行中”。
