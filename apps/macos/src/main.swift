import AppKit
import Foundation
import SwiftUI
import UniformTypeIdentifiers

private enum NativeLaunch {
    // The QA bundle stays isolated even if a UI tool relaunches it without
    // preserving the original process environment.
    static let isTestMode = ProcessInfo.processInfo.environment["ORANGECOUNT_TEST_MODE"] == "1"
        || Bundle.main.object(forInfoDictionaryKey: "OrangeCountTestMode") as? Bool == true
    static var colorScheme: ColorScheme? {
        guard isTestMode else { return nil }
        switch Bundle.main.object(forInfoDictionaryKey: "OrangeCountTestAppearance") as? String {
        case "light": return .light
        case "dark": return .dark
        default: return nil
        }
    }
}

struct LedgerCommands {
    let open: () -> Void
    let reload: () -> Void
    let language: LedgerLanguage
    let canReload: Bool
}
private struct LedgerCommandsKey: FocusedValueKey { typealias Value = LedgerCommands }
extension FocusedValues {
    var ledgerCommands: LedgerCommands? {
        get { self[LedgerCommandsKey.self] }
        set { self[LedgerCommandsKey.self] = newValue }
    }
}

@MainActor
struct ContentView: View {
    @StateObject private var store = LedgerStore(preferences: NativeLaunch.isTestMode ? nil : .standard)
    @StateObject private var journalState = JournalViewState()
    @StateObject private var treeState = AccountTreeState()
    @StateObject private var holdingsState = HoldingsViewState()
    private var language: LedgerLanguage { LedgerLanguage(rawValue: store.request.locale) ?? .chinese }

    var body: some View {
        NavigationSplitView {
            List(selection: $store.selectedNav) {
                Section(language.text("报表与流水", "Reports and transactions")) {
                    ForEach(NavItem.allCases.filter { $0 != .account }) { item in
                        NavigationLink(value: item) { Label(language.text(item.title, item.englishTitle), systemImage: item.icon) }
                    }
                }
                if !store.recentPaths.isEmpty {
                    Section(language.text("最近打开", "Recent ledgers")) {
                        ForEach(store.recentPaths, id: \.self) { path in
                            Button { store.loadLedger(at: path) } label: {
                                Label((path as NSString).lastPathComponent, systemImage: "clock.arrow.circlepath").lineLimit(1)
                            }.buttonStyle(.plain)
                        }
                    }
                }
            }.listStyle(.sidebar).navigationSplitViewColumnWidth(min: 190, ideal: 210)
        } detail: {
            VStack(alignment: .leading, spacing: 0) {
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(store.summary?.title?.isEmpty == false ? store.summary!.title! : "OrangeCount").font(LedgerFonts.font(.title3)).bold()
                        if !store.currentPath.isEmpty {
                            Text((store.currentPath as NSString).lastPathComponent).font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                                .help(store.currentPath)
                        }
                    }
                    Spacer()
                    if let summary = store.summary {
                        Button { store.showDiagnosticsSheet = true } label: {
                            Label(summary.valid ? language.text("校验通过", "Valid") : language.text("\(summary.error_count) 处错误", "\(summary.error_count) errors"), systemImage: summary.valid ? "checkmark.circle" : "exclamationmark.triangle")
                        }.tint(summary.valid ? .green : .orange)
                        .help(language.text("查看错误与警告", "View errors and warnings"))
                    }
                    Picker("Language", selection: $store.request.locale) { Text("中文").tag("zh-CN"); Text("English").tag("en") }.labelsHidden().frame(width: 95)
                    Button(action: store.reloadSilently) { Image(systemName: "arrow.clockwise") }
                        .disabled(store.state == nil).help(language.text("重新校验 (⌘R)", "Reload (⌘R)"))
                    Button(action: selectFile) { Label(language.text("打开…", "Open…"), systemImage: "folder.badge.plus") }.buttonStyle(.borderedProminent)
                }.padding(.horizontal, 20).padding(.vertical, 12).background(.ultraThinMaterial)
                Divider()
                if let error = store.errorMessage {
                    Label(error, systemImage: "exclamationmark.triangle.fill").font(LedgerFonts.font(.callout)).foregroundColor(.red).padding(10).textSelection(.enabled)
                }
                if store.phase == .stale {
                    Label(language.text("显示上次有效数据；请修复最新账本错误", "Showing the last valid data; repair the latest ledger errors"), systemImage: "exclamationmark.triangle").foregroundColor(.orange).padding(10)
                }
                if store.phase == .opening || store.phase == .loadingView || store.phase == .reloading {
                    ProgressView(language.text("正在读取账本…", "Reading ledger…")).controlSize(.small).padding(8)
                }
                if store.state?.has_snapshot == true {
                    if store.selectedNav != .query { ReadingControls(store: store) }
                    if let tree = store.treeReport, let status = tree.valuation_status, status.completeness == "partial" || status.completeness == "unavailable" {
                        Text(language.text("原币余额完整保留；缺少报价，已折算部分不代表完整总计。", "Native balances are retained. Missing quotes mean valued portions are not complete totals.")).foregroundColor(.orange).padding()
                    }
                    switch store.selectedNav {
                    case .balanceSheet, .incomeStatement:
                        AccountTreeView(rootNodes: store.treeReport?.root_nodes ?? [], operatingCurrencies: store.treeReport?.operating_currencies ?? [], reportType: $store.reportType, treeState: treeState, openAccount: store.showAccount)
                    case .journal:
                        JournalView(store: store, viewState: journalState)
                    case .dashboard:
                        DashboardView(trends: store.trendsPayload, summary: store.summary)
                    case .account:
                        AccountDetailView(store: store, viewState: journalState)
                    case .holdings:
                        HoldingsView(store: store, viewState: holdingsState)
                    case .prices:
                        PricesView(store: store)
                    case .query:
                        QueryView(store: store)
                    }
                } else if store.phase == .invalid {
                    VStack(spacing: 12) {
                        Text(language.text("账本尚无有效数据，请先查看并修复诊断。", "No valid snapshot yet. Review and repair the diagnostics."))
                        Button(language.text("查看诊断", "View diagnostics")) { store.showDiagnosticsSheet = true }
                    }.frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    VStack(spacing: 12) {
                        Image(systemName: "book.closed").font(LedgerFonts.font(.largeTitle))
                        Text(language.text("打开本地账本，查看报表与交易流水", "Open a local ledger to read reports and transactions"))
                        Button(language.text("打开账本…", "Open ledger…"), action: selectFile)
                    }.frame(maxWidth: .infinity, maxHeight: .infinity)
                }
            }
        }
        .environment(\.ledgerLanguage, language)
        .preferredColorScheme(NativeLaunch.colorScheme)
        .navigationTitle(store.currentPath.isEmpty ? "OrangeCount" : (store.currentPath as NSString).lastPathComponent)
        .focusedSceneValue(\.ledgerCommands, LedgerCommands(open: selectFile, reload: store.reloadSilently, language: language, canReload: store.state != nil))
        .onChange(of: store.currentPath) { _ in
            journalState.expandedTxIDs = []
            treeState.collapsedNodes = []
            treeState.selectedAccount = nil
            treeState.searchText = ""
            holdingsState.expanded = []
        }
        .frame(minWidth: 900, minHeight: 600)
        .font(LedgerFonts.font(.body))
        .sheet(isPresented: $store.showDiagnosticsSheet) {
            if let summary = store.summary { DiagnosticsSheetView(summary: summary, retry: store.reloadSilently).environment(\.ledgerLanguage, language) }
        }
        .onDrop(of: [UTType.fileURL], isTargeted: nil) { providers in
            guard let provider = providers.first else { return false }
            _ = provider.loadObject(ofClass: URL.self) { url, _ in
                if let url { DispatchQueue.main.async { store.loadLedger(at: url.path) } }
            }
            return true
        }
        .onAppear {
            let path = CommandLine.arguments.dropFirst().first { !$0.hasPrefix("-") }
            store.start(path: path, restoreRecent: !NativeLaunch.isTestMode)
        }
        .onOpenURL { store.loadLedger(at: $0.path) }
        .background(WindowCloseObserver(onClose: store.close))
    }

    private func selectFile() {
        let panel = NSOpenPanel()
        panel.allowsMultipleSelection = false
        panel.canChooseDirectories = true
        panel.canChooseFiles = true
        if panel.runModal() == .OK, let url = panel.url { store.loadLedger(at: url.path) }
    }
}

struct LedgerMenu: Commands {
    @FocusedValue(\.ledgerCommands) private var actions
    var body: some Commands {
        CommandGroup(after: .newItem) {
            Button(actions?.language.text("打开账本…", "Open ledger…") ?? "打开账本… / Open ledger…") { actions?.open() }.keyboardShortcut("o", modifiers: .command)
            Button(actions?.language.text("重新校验", "Reload ledger") ?? "重新校验 / Reload ledger") { actions?.reload() }.keyboardShortcut("r", modifiers: .command).disabled(actions?.canReload != true)
        }
    }
}

@main
struct OrangeCountNativeApp: App {
    var body: some Scene {
        WindowGroup { ContentView() }
            .windowStyle(.titleBar).windowToolbarStyle(.unified)
            .commands { LedgerMenu() }
    }
}

// A view can disappear while the window stays open (for example during a
// file-panel transition). Release its ledger only when its owning window closes.
@MainActor
private struct WindowCloseObserver: NSViewRepresentable {
    let onClose: () -> Void
    func makeNSView(context: Context) -> WindowCloseView { WindowCloseView(onClose: onClose) }
    func updateNSView(_ nsView: WindowCloseView, context: Context) { nsView.onClose = onClose }
}

@MainActor
private final class WindowCloseView: NSView {
    var onClose: () -> Void
    init(onClose: @escaping () -> Void) { self.onClose = onClose; super.init(frame: .zero) }
    required init?(coder: NSCoder) { fatalError("init(coder:) is unavailable") }
    override func viewDidMoveToWindow() {
        super.viewDidMoveToWindow()
        NotificationCenter.default.removeObserver(self)
        guard let window else { return }
        NotificationCenter.default.addObserver(self, selector: #selector(windowClosing), name: NSWindow.willCloseNotification, object: window)
    }
    @objc private func windowClosing(_ notification: Notification) { onClose() }
    deinit { NotificationCenter.default.removeObserver(self) }
}
