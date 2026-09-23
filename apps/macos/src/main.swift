import AppKit
import Foundation
import SwiftUI
import UniformTypeIdentifiers

enum NavItem: String, CaseIterable, Identifiable {
    case balanceSheet = "balance_sheet"
    case incomeStatement = "income_statement"
    case journal = "journal"
    case dashboard = "dashboard"

    var id: String { rawValue }
    var title: String {
        switch self {
        case .dashboard: return "财务仪表盘"
        case .balanceSheet: return "资产负债表"
        case .incomeStatement: return "损益表"
        case .journal: return "日记账流水"
        }
    }
    var icon: String {
        switch self {
        case .dashboard: return "chart.pie.fill"
        case .balanceSheet: return "chart.bar.doc.horizontal"
        case .incomeStatement: return "chart.line.uptrend.xyaxis"
        case .journal: return "list.bullet.rectangle"
        }
    }
}

class LedgerStore: ObservableObject {
    @Published var pingMessage: String = ""
    @Published var summary: LedgerSummary?
    @Published var currentPath: String = ""
    @Published var errorMessage: String?
    @Published var selectedNav: NavItem = .balanceSheet
    @Published var treeReport: TreeReportPayload?
    @Published var journalPayload: JournalPayload?
    @Published var trendsPayload: HistoricalTrendsPayload?
    @Published var recentPaths: [String] = []
    @Published var showDiagnosticsSheet: Bool = false
    @Published var reportType: String = "balance_sheet" {
        didSet {
            loadTreeReport()
        }
    }

    private var watchTimer: Timer?
    private var lastRevision: String = ""
    private let recentsKey = "org.orangecount.recent_ledgers"

    init() {
        pingMessage = BridgeClient.shared.ping()
        recentPaths = UserDefaults.standard.stringArray(forKey: recentsKey) ?? []

        let defaultFixture = locateDefaultFixture()
        if !defaultFixture.isEmpty {
            loadLedger(at: defaultFixture)
        }
        startWatcher()
    }

    private func startWatcher() {
        watchTimer = Timer.scheduledTimer(withTimeInterval: 0.8, repeats: true) { [weak self] _ in
            guard let self = self, !self.currentPath.isEmpty else { return }
            let rev = BridgeClient.shared.getLedgerRevision(path: self.currentPath)
            if !self.lastRevision.isEmpty && rev != self.lastRevision {
                self.lastRevision = rev
                self.reloadSilently()
            }
        }
    }

    func loadLedger(at rawPath: String) {
        let entryPath = resolveLedgerEntry(from: rawPath)
        currentPath = entryPath
        recordRecent(entryPath)
        lastRevision = BridgeClient.shared.getLedgerRevision(path: entryPath)

        Task.detached(priority: .userInitiated) {
            do {
                let s = try BridgeClient.shared.loadSnapshot(path: entryPath)
                let t = try BridgeClient.shared.getTreeReport(path: entryPath, reportType: "balance_sheet")
                let j = try BridgeClient.shared.getJournal(path: entryPath)
                let tr = try BridgeClient.shared.getHistoricalTrends(path: entryPath)
                await MainActor.run {
                    self.summary = s
                    self.treeReport = t
                    self.journalPayload = j
                    self.trendsPayload = tr
                    self.reportType = "balance_sheet"
                    self.errorMessage = nil
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                }
            }
        }
    }

    func reloadSilently() {
        guard !currentPath.isEmpty else { return }
        Task.detached(priority: .userInitiated) {
            do {
                let s = try BridgeClient.shared.loadSnapshot(path: self.currentPath)
                let t = try BridgeClient.shared.getTreeReport(path: self.currentPath, reportType: self.reportType)
                let j = try BridgeClient.shared.getJournal(path: self.currentPath)
                let tr = try BridgeClient.shared.getHistoricalTrends(path: self.currentPath)
                await MainActor.run {
                    withAnimation(.easeInOut(duration: 0.2)) {
                        self.summary = s
                        self.treeReport = t
                        self.journalPayload = j
                        self.trendsPayload = tr
                        self.errorMessage = nil
                    }
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                }
            }
        }
    }

    func loadTreeReport() {
        guard !currentPath.isEmpty else { return }
        Task.detached(priority: .userInitiated) {
            do {
                let t = try BridgeClient.shared.getTreeReport(path: self.currentPath, reportType: self.reportType)
                await MainActor.run {
                    self.treeReport = t
                    self.errorMessage = nil
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                }
            }
        }
    }

    private func recordRecent(_ path: String) {
        var recents = recentPaths.filter { $0 != path }
        recents.insert(path, at: 0)
        if recents.count > 5 {
            recents = Array(recents.prefix(5))
        }
        recentPaths = recents
        UserDefaults.standard.set(recents, forKey: recentsKey)
    }

    private func resolveLedgerEntry(from path: String) -> String {
        var isDir: ObjCBool = false
        if FileManager.default.fileExists(atPath: path, isDirectory: &isDir), isDir.boolValue {
            let candidates = ["main.bean", "index.bean", "ledger.bean"]
            for c in candidates {
                let candidatePath = (path as NSString).appendingPathComponent(c)
                if FileManager.default.fileExists(atPath: candidatePath) {
                    return candidatePath
                }
            }
            if let contents = try? FileManager.default.contentsOfDirectory(atPath: path) {
                let beans = contents.filter { $0.hasSuffix(".bean") }
                if beans.count == 1 {
                    return (path as NSString).appendingPathComponent(beans[0])
                }
            }
        }
        return path
    }

    private func locateDefaultFixture() -> String {
        let cwd = FileManager.default.currentDirectoryPath
        let candidates = [
            "\(cwd)/testdata/fixtures/v3-parity/txn-basic.bean",
            "\(cwd)/../../testdata/fixtures/v3-parity/txn-basic.bean"
        ]
        for p in candidates {
            if FileManager.default.fileExists(atPath: p) {
                return (p as NSString).standardizingPath
            }
        }
        return ""
    }
}

struct ContentView: View {
    @StateObject private var store = LedgerStore()

    var body: some View {
        NavigationSplitView {
            List(selection: $store.selectedNav) {
                Section("报表分析") {
                    NavigationLink(value: NavItem.balanceSheet) {
                        Label(NavItem.balanceSheet.title, systemImage: NavItem.balanceSheet.icon)
                    }
                    NavigationLink(value: NavItem.incomeStatement) {
                        Label(NavItem.incomeStatement.title, systemImage: NavItem.incomeStatement.icon)
                    }
                }

                Section("交易明细") {
                    NavigationLink(value: NavItem.journal) {
                        Label(NavItem.journal.title, systemImage: NavItem.journal.icon)
                    }
                    NavigationLink(value: NavItem.dashboard) {
                        Label(NavItem.dashboard.title, systemImage: NavItem.dashboard.icon)
                    }
                }

                if !store.recentPaths.isEmpty {
                    Section("最近打开") {
                        ForEach(store.recentPaths, id: \.self) { p in
                            Button(action: { store.loadLedger(at: p) }) {
                                HStack {
                                    Image(systemName: "clock.arrow.circlepath")
                                        .foregroundColor(.secondary)
                                    Text((p as NSString).lastPathComponent)
                                        .font(.caption)
                                        .lineLimit(1)
                                }
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }

                Section("系统连接") {
                    HStack {
                        Circle()
                            .fill(.green)
                            .frame(width: 8, height: 8)
                        Text("Go Core 内存直连正常")
                            .font(.caption)
                    }
                    Text(store.pingMessage)
                        .font(.system(size: 10, design: .monospaced))
                        .foregroundColor(.secondary)
                }
            }
            .listStyle(.sidebar)
            .navigationSplitViewColumnWidth(min: 200, ideal: 230)
        } detail: {
            VStack(alignment: .leading, spacing: 0) {
                // Top Global Bar
                HStack {
                    VStack(alignment: .leading, spacing: 3) {
                        HStack(spacing: 8) {
                            Text(store.summary?.title?.isEmpty == false ? store.summary!.title! : "OrangeCount 原生 macOS 桌面版")
                                .font(.title3)
                                .bold()
                            if let s = store.summary {
                                Button(action: {
                                    if s.error_count > 0 {
                                        store.showDiagnosticsSheet = true
                                    }
                                }) {
                                    HStack(spacing: 4) {
                                        Circle()
                                            .fill(s.valid ? Color.green : Color.red)
                                            .frame(width: 6, height: 6)
                                        Text(s.valid ? "VALID" : "\(s.error_count) 处错误")
                                            .font(.system(size: 10, weight: .bold))
                                    }
                                    .padding(.horizontal, 6)
                                    .padding(.vertical, 3)
                                    .background(Capsule().fill(s.valid ? Color.green.opacity(0.15) : Color.red.opacity(0.15)))
                                    .foregroundColor(s.valid ? .green : .red)
                                }
                                .buttonStyle(.plain)
                                .help(s.valid ? "账本校验完全通过" : "点击查看错误诊断详情")
                            }
                        }
                        Text("入口：\(store.currentPath.isEmpty ? "未选择" : store.currentPath)")
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .lineLimit(1)
                            .truncationMode(.middle)
                    }
                    Spacer()

                    Button(action: { store.reloadSilently() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                    .buttonStyle(.bordered)
                    .help("重新加载当前账本 (Cmd+R)")
                    .keyboardShortcut("r", modifiers: .command)

                    Button(action: selectFile) {
                        Label("打开...", systemImage: "folder.badge.plus")
                    }
                    .buttonStyle(.borderedProminent)
                    .help("打开账本文件或目录 (Cmd+O)")
                    .keyboardShortcut("o", modifiers: .command)
                }
                .padding(.horizontal, 20)
                .padding(.vertical, 12)
                .background(.ultraThinMaterial)

                Divider()

                if let err = store.errorMessage {
                    HStack {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .foregroundColor(.red)
                        Text("错误：\(err)")
                            .font(.caption)
                            .foregroundColor(.red)
                        Spacer()
                    }
                    .padding(10)
                    .background(Color.red.opacity(0.1))
                }

                // Main Content View based on Route
                switch store.selectedNav {
                case .balanceSheet:
                    AccountTreeView(
                        rootNodes: store.treeReport?.root_nodes ?? [],
                        operatingCurrencies: store.treeReport?.operating_currencies ?? [],
                        reportType: $store.reportType
                    )
                case .incomeStatement:
                    AccountTreeView(
                        rootNodes: store.treeReport?.root_nodes ?? [],
                        operatingCurrencies: store.treeReport?.operating_currencies ?? [],
                        reportType: $store.reportType
                    )
                case .journal:
                    JournalView(transactions: store.journalPayload?.transactions ?? [])
                case .dashboard:
                    DashboardView(trends: store.trendsPayload, summary: store.summary)
                }
            }
        }
        .frame(minWidth: 880, minHeight: 580)
        .sheet(isPresented: $store.showDiagnosticsSheet) {
            if let s = store.summary {
                DiagnosticsSheetView(summary: s)
            }
        }
        .onDrop(of: [UTType.fileURL], isTargeted: nil) { providers in
            guard let provider = providers.first else { return false }
            _ = provider.loadObject(ofClass: URL.self) { url, _ in
                if let url = url {
                    DispatchQueue.main.async {
                        store.loadLedger(at: url.path)
                    }
                }
            }
            return true
        }
        .onChange(of: store.selectedNav) { _, newNav in
            if newNav == .balanceSheet && store.reportType != "balance_sheet" {
                store.reportType = "balance_sheet"
            } else if newNav == .incomeStatement && store.reportType != "income_statement" {
                store.reportType = "income_statement"
            }
        }
    }

    private func selectFile() {
        let panel = NSOpenPanel()
        panel.allowsMultipleSelection = false
        panel.canChooseDirectories = true
        panel.canChooseFiles = true
        panel.allowedContentTypes = []
        if panel.runModal() == .OK, let url = panel.url {
            store.loadLedger(at: url.path)
        }
    }
}

@main
struct OrangeCountNativeApp: App {
    var body: some Scene {
        WindowGroup {
            ContentView()
        }
        .windowStyle(.titleBar)
        .windowToolbarStyle(.unified)
    }
}
