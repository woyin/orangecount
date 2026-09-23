# OrangeCount macOS 原生桌面版架构方案与详细工程计划 (Native Desktop App Specification)

- **文档版本**：v2.0 (Industrial Detailed Specification)
- **文档状态**：Draft for Architecture Review & Execution Baseline
- **定位**：OrangeCount 原生 macOS 桌面客户端（纯 Native，零 WebView，零网络暴露，单进程内存直连）。

---

## 一、系统整体架构与内存模型

### 1. 进程内拓扑模型 (In-Process Topology)

桌面应用运行于单一 macOS App 进程空间中，不启动任何本地 HTTP 端口，不创建孤儿后台进程，不使用 WebKit 网页容器：

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                           OrangeCount.app 进程空间                           │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    macOS 原生展示层 (Swift 6 / SwiftUI)                │  │
│  │                                                                       │  │
│  │  • AppState / LedgerStore (主线程响应式状态机 @Observable / ObservableObject)│  │
│  │  • Views: NavigationSplitView, BalanceSheetTree, JournalTable, Charts │  │
│  │  • Native Components: NSOpenPanel, QuickLook Preview, Native Menus    │  │
│  └───────────────────────────────────┬───────────────────────────────────┘  │
│                                      │                                      │
│                  Swift C-Bridging 调用 (Zero-Copy / Direct Call)             │
│                                      ▼                                      │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                 C-ABI 接口契约层 (internal/bridge)                     │  │
│  │                                                                       │  │
│  │  • 头文件: liborangecount.h (由 cgo -buildmode=c-archive 生成)         │  │
│  │  • 函数导出: OC_LoadSnapshot, OC_GetTreeReport, OC_GetJournal 等       │  │
│  │  • 内存生命周期协调器: OC_FreeString                                     │  │
│  └───────────────────────────────────┬───────────────────────────────────┘  │
│                                      │                                      │
│                  Go Runtime & Internal Modules 内存直读                      │
│                                      ▼                                      │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    OrangeCount Go 会计核心引擎                         │  │
│  │                                                                       │  │
│  │  • internal/snapshot: Include 图解析、AST 构建、文件变更监听 (Watch)   │  │
│  │  • internal/ledger: Beancount v3 会计求值器 (Booked Positions, Lots)  │  │
│  │  • internal/report: 账户树递归汇总 (buildAccountTreeIndex), Journal   │  │
│  │  • internal/query: BQL 查询分析与执行                                  │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2. 跨语言内存生命周期契约 (Memory Safety Protocol)

Go 运行时具备垃圾回收（GC），Swift 具备自动引用计数（ARC），二者在 C-ABI 边界处遵循严格所有权规则：

1. **分配权归属**：所有由 Go 导出的数据指针（如 `*C.char`）均由 Go 运行时使用 `C.CString` 在 C-Heap 上显式分配；
2. **释放权归属**：Swift 在使用完 C 指针并将其解码为 Swift 原生内存结构后，**必须且只能**通过 `OC_FreeString(ptr)` 归还给 C-Heap 释放；
3. **Swift RAII 自动守卫**：Swift 端禁止裸调 C 指针，必须封装自动作用域守护函数：
   ```swift
   func withGoResponse<T>(_ call: () -> UnsafeMutablePointer<CChar>?, decode: (Data) throws -> T) throws -> T {
       guard let cPtr = call() else {
           throw LedgerError.nullPointerFromGo
       }
       defer { OC_FreeString(cPtr) }
       let data = Data(bytes: cPtr, count: strlen(cPtr))
       return try decode(data)
   }
   ```
4. **并发与主线程隔离**：所有跨语言 C 调用与 JSON 反序列化操作均置于 Swift 协作式后台任务（`Task.detached(priority: .userInitiated)`）中执行，完成后切换至 `@MainActor` 更新 UI 状态，确保 UI 线程 120 FPS 满帧不阻塞。

---

## 二、C-ABI 接口详细规约 (API Contract Specification)

所有 C 接口定义于 `internal/bridge/bridge.go`，并在 `liborangecount.h` 中暴露：

```c
#ifdef __cplusplus
extern "C" {
#endif

// 验证连通性
extern char* OC_Ping(void);

// 加载指定路径的账本快照摘要（元数据、币种、账户列表、有效性、错误）
extern char* OC_LoadSnapshot(char* cPath);

// 获取树状报表（Balance Sheet / Income Statement）
// cPath: 账本入口绝对路径
// cReportType: "balance_sheet" 或 "income_statement"
extern char* OC_GetTreeReport(char* cPath, char* cReportType);

// 获取交易日记账（Journal）流式列表
// cPath: 账本入口绝对路径
// cFilterJSON: 过滤选项 JSON 字符串 (如 {"account": "", "query": "", "from": "", "to": ""})
extern char* OC_GetJournal(char* cPath, char* cFilterJSON);

// 获取历史资产净值与收支时间序列（用于 Swift Charts 原生渲染）
// cPath: 账本入口绝对路径
extern char* OC_GetHistoricalTrends(char* cPath);

// 查询账本修改版本号（基于文件图 mtime 校验，用于毫秒级热重载探针）
// cPath: 账本入口绝对路径
extern char* OC_GetLedgerRevision(char* cPath);

// 释放由 Go 运行时通过 C.CString 分配的堆内存
extern void OC_FreeString(char* ptr);

#ifdef __cplusplus
}
#endif
```

---

## 三、跨语言数据交换协议 (Data Schemas)

所有从 Go 返回给 Swift 的数据均为 UTF-8 编码的标准 JSON 字符串。

### 1. 账本快照元信息与状态 (`OC_LoadSnapshot` 响应)
```json
{
  "valid": true,
  "entry_path": "/Users/name/ledger/main.bean",
  "title": "My Personal Ledger",
  "operating_currencies": ["USD", "CNY"],
  "accounts_count": 48,
  "entries_count": 1250,
  "error_count": 0,
  "diagnostics": [
    {
      "code": "E-EVAL-BALANCE",
      "severity": "error",
      "message": "balance assertion failed: expected 100 USD != actual 90 USD",
      "file": "/Users/name/ledger/accounts.bean",
      "line": 42
    }
  ]
}
```

### 2. 递归账户树报表 (`OC_GetTreeReport` 响应)
直接由 `internal/report:buildAccountTreeIndex` 递归折叠生成：
```json
{
  "report_type": "balance_sheet",
  "operating_currency": "USD",
  "root_nodes": [
    {
      "name": "Assets",
      "full_name": "Assets",
      "depth": 0,
      "is_leaf": false,
      "explicit": true,
      "direct_balances": {},
      "subtree_balances": {
        "USD": "125000.50",
        "CNY": "35000.00"
      },
      "children": [
        {
          "name": "Cash",
          "full_name": "Assets:Cash",
          "depth": 1,
          "is_leaf": false,
          "explicit": true,
          "direct_balances": {},
          "subtree_balances": { "USD": "15000.00" },
          "children": [
            {
              "name": "Checking",
              "full_name": "Assets:Cash:Checking",
              "depth": 2,
              "is_leaf": true,
              "explicit": true,
              "direct_balances": { "USD": "15000.00" },
              "subtree_balances": { "USD": "15000.00" },
              "children": []
            }
          ]
        }
      ]
    }
  ]
}
```

### 3. 流水日记账 (`OC_GetJournal` 响应)
```json
{
  "total_count": 1250,
  "transactions": [
    {
      "id": "tx_20260102_001",
      "date": "2026-01-02",
      "flag": "*",
      "payee": "Apple Store",
      "narration": "MacBook Pro M4 Max",
      "tags": ["hardware", "work"],
      "links": ["order-9988"],
      "postings": [
        {
          "account": "Expenses:Equipment:Tech",
          "units_number": "2499.00",
          "units_currency": "USD",
          "cost": null,
          "price": null
        },
        {
          "account": "Liabilities:CreditCard:AppleCard",
          "units_number": "-2499.00",
          "units_currency": "USD",
          "cost": null,
          "price": null
        }
      ]
    }
  ]
}
```

### 4. 财务趋势与时间序列 (`OC_GetHistoricalTrends` 响应)
```json
{
  "net_worth_points": [
    { "date": "2025-12-31", "value": 118000.00, "currency": "USD" },
    { "date": "2026-01-31", "value": 122500.00, "currency": "USD" }
  ],
  "monthly_cash_flows": [
    {
      "month": "2026-01",
      "income": 8500.00,
      "expenses": 4200.00,
      "net": 4300.00,
      "currency": "USD"
    }
  ]
}
```

---

## 四、macOS 原生 UI 架构与组件规范

纯 SwiftUI 原生设计，无 HTML/CSS。

### 1. 窗口布局与响应式分栏
- **窗口骨架**：基于 `NavigationSplitView(sidebar:detail:)`。
- **窗口规格**：
  - 默认打开尺寸：`1280 × 800`；
  - 最小支持尺寸：`900 × 580`；
  - 侧边栏宽度：最小 200pt，理想 240pt，最大 320pt；
- **材质与主题**：
  - 侧边栏采用 macOS 原生 `.ultraThinMaterial` 磨砂毛玻璃材质；
  - 适配系统级 Light / Dark 模式平滑切换；
  - 数字与金额列统一应用 `.font(.system(.body, design: .monospaced))` 保证排版对齐。

### 2. 核心视图层级拆解 (Component Hierarchy)

```text
OrangeCountNativeApp (@main)
└── WindowGroup
    └── RootView
        ├── ErrorBannerSheet (发生致命语法/平衡错误时弹出)
        └── NavigationSplitView
            ├── SidebarView
            │   ├── SystemStatusBadge (Go Core 内存直连健康度)
            │   ├── FileLocationHeader (当前账本路径、重载触发器)
            │   └── NavigationList
            │       ├── Item: 仪表盘 (DashboardView)
            │       ├── Item: 资产负债表 (BalanceSheetView)
            │       ├── Item: 损益表 (IncomeStatementView)
            │       └── Item: 日记账流水 (JournalView)
            └── DetailContainerView (根据当前路由动态呈现)
                ├── DashboardView (Net Worth 折线图 + 月度柱状图 + KPI 指标卡)
                ├── AccountTreeTableView (树状列表、折叠/展开展开器、多币种并列列)
                └── JournalVirtualListView (LazyVStack 交易卡片、分录明细展开、即时过滤)
```

---

## 五、分阶段实施路线图 (Sprint Milestones & Execution)

### 阶段一（Sprint 1）：数据桥深化与“资产负债 / 损益”原生折叠树
- **周期预估**：2 天
- **子任务细分**：
  - **Task 1.1 (Go)**：在 `internal/bridge/bridge.go` 中封装 `accountTreeIndex` 转换为 JSON 的结构体；实现 `OC_GetTreeReport(cPath, cReportType)`，输出 `report_type`, `root_nodes` 完整嵌套树；
  - **Task 1.2 (Go 单元测试)**：编写 `internal/bridge/bridge_test.go`，针对 `v3-parity/directives-mix.bean` 验证生成的树结构深度、父子关系和多币种汇总是否正确；
  - **Task 1.3 (Swift)**：在 `apps/macos/src/Models.swift` 中定义 `AccountTreeNode`（遵循 `Identifiable`, `Codable`），定义 `TreeReportPayload`；
  - **Task 1.4 (Swift)**：构建 `AccountTreeView.swift`：
    - 树形展现：使用递归组件或原生 `OutlineGroup`；
    - 节点展开/折叠状态管理（支持一键“全部展开” / “全部折叠”）；
    - 账户名搜索即时高亮与父链自动展开；
    - 多币种金额并列排版（右对齐、等宽字体、负数红字展示）；
  - **Task 1.5 (集成验证)**：通过 `make app` 编译并在 Native App 中打开 `testdata/fixtures/v3-parity/` 下的所有语料，验证资产负债表与损益表完全展示正确。
- **阶段一验收标准**：
  1. `make app` 零警告通过编译；
  2. 能够在 Native App 窗口中随意展开/折叠 Assets、Liabilities、Equity、Income、Expenses 节点；
  3. 各节点多币种余额与 Beancount 计算结果分毫不差。

---

### 阶段二（Sprint 2）：原生流水日记账（Journal）与即时检索
- **周期预估**：2 天
- **子任务细分**：
  - **Task 2.1 (Go)**：在 `internal/bridge/bridge.go` 中解析 `cFilterJSON`，调用 `internal/report:JournalBetween`，实现 `OC_GetJournal(cPath, cFilterJSON)`；
  - **Task 2.2 (Go 单元测试)**：在 `bridge_test.go` 中增加过滤参数测试（日期区间、账户过滤），验证分录金额完整性；
  - **Task 2.3 (Swift)**：在 Swift 端定义 `JournalTransaction`、`JournalPosting` 模型；
  - **Task 2.4 (Swift)**：构建 `JournalView.swift`：
    - 采用 `LazyVStack` 虚拟滚动容器实现交易流；
    - 单笔交易呈现：日期徽章、Flag（`*`/`!`）胶囊、Payee、Narration、Tags（`#` 标签气泡）、Links（`^` 链接气泡）；
    - 点击交易卡片支持平滑展开分录列表（展示具体各个 Posting 的借贷账户、借贷金额与折算汇率）；
    - 顶部过滤搜索栏：支持按账户名、标签、Payee 关键字的毫秒级输入过滤；
  - **Task 2.5 (集成验证)**：打开万级分录语料，验证上下连续惯性滑动无卡顿，搜索输入即时响应。
- **阶段二验收标准**：
  1. 流水列表虚拟滚动保持 120 FPS 满帧；
  2. 搜索过滤响应延迟 < 50ms。

---

### 阶段三（Sprint 3）：原生金融图表（Swift Charts）与自动热重载
- **周期预估**：2 天
- **子任务细分**：
  - **Task 3.1 (Go)**：在 `internal/bridge/bridge.go` 中实现历史净资产计算与时间序列采样，实现 `OC_GetHistoricalTrends(cPath)`；
  - **Task 3.2 (Go)**：实现 `OC_GetLedgerRevision(cPath)`，提取账本包含图所有文件的最高 mtime 与文件签名哈希；
  - **Task 3.3 (Swift)**：构建 `DashboardView.swift`：
    - 导入原生 `Charts` 框架；
    - 净资产走势：绘制折线图与面积渐变（支持鼠标 Hover 十字星标刻度尺探针）；
    - 月度现金流：绘制双色柱状图（绿色代表 Income，橙色代表 Expense）；
  - **Task 3.4 (Swift 热重载)**：在 `LedgerStore` 中建立 500ms 定时轮询探针：
    - 比对 `OC_GetLedgerRevision`，当检测到文件修改（在外部编辑器保存）时，自动触发重载并在当前视图做平滑渐变更新，**严格保留用户的展开节点与滚动位置**；
  - **Task 3.5 (集成验证)**：在 VS Code 中修改账本的一笔金额并保存，Native 窗口在 0.5s 内自动重绘图表与余额。
- **阶段三验收标准**：
  1. Swift Charts 渲染平滑无撕裂，支持暗黑模式；
  2. 外部保存文件后，App 自动无感热重载，无需用户手动刷新。

---

### 阶段四（Sprint 4）：系统级深度集成、诊断体验与正式打包
- **周期预估**：1~2 天
- **子任务细分**：
  - **Task 4.1 (交互增强)**：
    - 支持直接将 `.bean` 文件或包含 `main.bean` 的文件夹拖入 App 窗口直接打开；
    - 菜单栏接入系统级“最近打开的文件”历史记录；
    - 全局快捷键绑定：`Cmd+O`（打开文件）、`Cmd+R`（手动重载）、`Cmd+F`（聚焦过滤框）；
  - **Task 4.2 (诊断与错误呈现)**：
    - 当账本解析遇到 `E-EVAL-BALANCE`、`E-PARSE-*` 时，状态栏徽章切换为红色；
    - 点击弹出原生诊断面板，列出精确文件相对路径、行号、错误代码与修复建议，支持一键点击在系统默认编辑器中打开该文件定位；
  - **Task 4.3 (构建与交付固化)**：
    - 完善 `apps/macos/build.sh` 与 `Makefile`：支持生成通用架构二进制（支持 Apple Silicon 与 Intel）；
    - 产出标准的 `build/OrangeCount.app`。
- **阶段四验收标准**：
  1. 拖放文件、快捷键、文件历史均符合 macOS 一等公民软件规范；
  2. `make app` 成为 CI 和日常本地编译的唯一标准入口。

---

## 六、开发准则与不可触碰的红线

1. **绝对单向依赖**：Go Core 严禁引入任何 UI 或 AppKit 逻辑；Swift 端严禁自行编写任何复式记账余额折算或借贷平衡算法；
2. **零网络原则**：绝不监听任何 TCP 端口，所有交互必须走 Cgo 内存直调；
3. **零 Xcode.app 强依赖**：所有编译脚本必须在仅安装 `CommandLineTools` 的环境下 100% 成功执行；
4. **阶段交付纪律**：必须严格按照 Sprint 1 -> Sprint 2 -> Sprint 3 -> Sprint 4 推进，每一阶段完成后进行实际可运行的 App 冒烟验证。
