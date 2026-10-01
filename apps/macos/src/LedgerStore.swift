// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import Combine
import Foundation

enum NavItem: String, CaseIterable, Identifiable {
    case balanceSheet = "balance_sheet"
    case incomeStatement = "income_statement"
    case journal
    case dashboard
    case account
    case holdings
    case prices

    var id: String { rawValue }
    var title: String {
        switch self {
        case .balanceSheet: return "资产负债表"
        case .incomeStatement: return "损益表"
        case .journal: return "日记账流水"
        case .dashboard: return "财务仪表盘"
        case .account: return "账户详情"
        case .holdings: return "持仓与收益"
        case .prices: return "商品与价格"
        }
    }
    var englishTitle: String {
        switch self {
        case .balanceSheet: return "Balance Sheet"
        case .incomeStatement: return "Income Statement"
        case .journal: return "Transactions"
        case .dashboard: return "Dashboard"
        case .account: return "Account details"
        case .holdings: return "Holdings and gains"
        case .prices: return "Commodities and prices"
        }
    }
    var icon: String {
        switch self {
        case .balanceSheet: return "chart.bar.doc.horizontal"
        case .incomeStatement: return "chart.line.uptrend.xyaxis"
        case .journal: return "list.bullet.rectangle"
        case .dashboard: return "chart.pie.fill"
        case .account: return "folder"
        case .holdings: return "chart.bar.xaxis"
        case .prices: return "chart.xyaxis.line"
        }
    }
}

// The real C adapter and controlled asynchronous test adapter share this seam.
protocol LedgerSessionClient: Sendable {
    func open(path: String, semantics: String, inputIdentity: String) async throws -> OpenLedgerResult
    func read(handle: UInt64, request: ViewRequest) async throws -> LedgerView
    func poll(handle: UInt64, after: UInt64) async throws -> PollResult
    func reload(handle: UInt64) async throws -> LedgerState
    func close(handle: UInt64) async
}

extension LedgerSessionClient {
    func open(path: String, semantics: String = "legacy") async throws -> OpenLedgerResult { try await open(path: path, semantics: semantics, inputIdentity: "") }
}

// One executor per window bounds outstanding C work. Cancellation is checked
// on entry, so superseded queued reads never start. A synchronous C call that
// already began may finish; LedgerStore still checks generation before publish.
actor NativeLedgerClient: LedgerSessionClient {
    func open(path: String, semantics: String, inputIdentity: String) async throws -> OpenLedgerResult {
        try Task.checkCancellation()
        return try BridgeClient.shared.openLedger(path: Self.resolveEntry(path), semantics: semantics, inputIdentity: inputIdentity)
    }
    func read(handle: UInt64, request: ViewRequest) async throws -> LedgerView {
        try Task.checkCancellation()
        return try BridgeClient.shared.readView(handle: handle, request: request)
    }
    func poll(handle: UInt64, after: UInt64) async throws -> PollResult {
        try Task.checkCancellation()
        return try BridgeClient.shared.pollState(handle: handle, after: after)
    }
    func reload(handle: UInt64) async throws -> LedgerState {
        try Task.checkCancellation()
        return try BridgeClient.shared.reloadLedger(handle: handle)
    }
    func close(handle: UInt64) async {
        // Cleanup must run even when its initiating task has been cancelled.
        try? BridgeClient.shared.closeLedger(handle: handle)
    }

    private static func resolveEntry(_ path: String) throws -> String {
        let normalized = URL(fileURLWithPath: (path as NSString).expandingTildeInPath).standardizedFileURL.resolvingSymlinksInPath().path
        var directory: ObjCBool = false
        if FileManager.default.fileExists(atPath: normalized, isDirectory: &directory), directory.boolValue {
            for name in ["main.bean", "index.bean", "ledger.bean"] {
                let candidate = (normalized as NSString).appendingPathComponent(name)
                if FileManager.default.fileExists(atPath: candidate) { return candidate }
            }
            let files = try FileManager.default.contentsOfDirectory(atPath: normalized).filter { $0.hasSuffix(".bean") }
            guard files.count == 1 else {
                throw BridgeError.remote("ambiguous-directory", "请选择具体的账本入口文件 / Select a ledger entry file")
            }
            return (normalized as NSString).appendingPathComponent(files[0])
        }
        return normalized
    }
}

enum LedgerPhase: Equatable {
    case idle, opening, loadingView, ready, reloading, stale, invalid, failed, closed
}

@MainActor
final class LedgerStore: ObservableObject {
    @Published private(set) var phase: LedgerPhase = .idle
    @Published private(set) var currentPath = ""
    @Published private(set) var snapshot: LedgerView?
    @Published private(set) var state: LedgerState?
    @Published private(set) var errorMessage: String?
    @Published private(set) var recentPaths: [String]
    @Published private(set) var semantics = "legacy"
    @Published var migrationPreview: SemanticsPreview?
    @Published var showDiagnosticsSheet = false
    @Published private(set) var navigationStack: [ViewRequest] = []
    @Published var request = ViewRequest() {
        didSet {
            if request.view != "journal" && request.view != "prices" { analysisValuation = request.valuation }
            if request != oldValue { loadCurrentView() }
        }
    }

    var selectedNav: NavItem {
        get { NavItem(rawValue: request.view) ?? .balanceSheet }
        set {
            guard newValue != .account else { return }
            guard newValue.rawValue != request.view else { return }
            navigationStack = []
            var next = request
            next.view = newValue.rawValue
            next.offset = 0
            next.snapshot_id = nil
            next.window = newValue == .journal ? "1m" : (newValue == .dashboard ? "1y" : "all")
            next.text = ""
            next.account = ""
            next.scope = nil
            Self.clearHoldingParameters(&next)
            Self.clearPriceParameters(&next)
            next.valuation = newValue == .journal || newValue == .prices ? "at-cost" : analysisValuation
            request = next
        }
    }
    var reportType: String {
        get { request.view }
        set { if newValue == "balance_sheet" || newValue == "income_statement" { selectedNav = NavItem(rawValue: newValue) ?? .balanceSheet } }
    }
    var summary: LedgerSummary? { snapshot?.summary }
    var treeReport: TreeReportPayload? { visibleView?.tree }
    var journalPayload: JournalPayload? { visibleView?.journal }
    var trendsPayload: HistoricalTrendsPayload? { visibleView?.trends }
    var accountDetail: AccountDetailPayload? { visibleView?.account_detail }
    var holdingsPayload: HoldingsPayload? { visibleView?.holdings }
    var pricesPayload: PricesPayload? { visibleView?.prices }
    var canNavigateBack: Bool { !navigationStack.isEmpty }
    var visibleView: LedgerView? { snapshot?.request == request ? snapshot : nil }

    private let client: any LedgerSessionClient
    private let preferences: UserDefaults?
    private let pollNanoseconds: UInt64
    private let recentsKey = "org.orangecount.recent_ledgers"
    private var preferencePath = ""
    private var analysisValuation = "at-cost"
    private var handle: UInt64?
    private var generation: UInt64 = 0
    private var readGeneration: UInt64 = 0
    private var started = false
    private var openTask: Task<Void, Never>?
    private var readTask: Task<Void, Never>?
    private var pollTask: Task<Void, Never>?
    private var reloadTask: Task<Void, Never>?
    private var migrationTask: Task<Void, Never>?

    init(client: any LedgerSessionClient = NativeLedgerClient(), preferences: UserDefaults? = .standard, pollNanoseconds: UInt64 = 250_000_000) {
        self.client = client
        self.preferences = preferences
        self.pollNanoseconds = pollNanoseconds
        recentPaths = preferences?.stringArray(forKey: recentsKey) ?? []
    }

    deinit {
        openTask?.cancel()
        readTask?.cancel()
        pollTask?.cancel()
        reloadTask?.cancel()
        migrationTask?.cancel()
        if let handle {
            let client = client
            Task { await client.close(handle: handle) }
        }
    }

    func start(path: String? = nil, restoreRecent: Bool = true) {
        guard !started else { return }
        started = true
        if let path { loadLedger(at: path); return }
        guard restoreRecent else { return }
        let paths = recentPaths
        let expected = generation
        openTask = Task { [weak self] in
            let candidate = await Task.detached {
                paths.first { FileManager.default.fileExists(atPath: $0) }
            }.value
            guard let self, !Task.isCancelled, self.generation == expected, let candidate else { return }
            self.loadLedger(at: candidate)
        }
    }

    func switchSemantics(to profile: String) {
        guard ["legacy", "v3"].contains(profile), profile != semantics, !currentPath.isEmpty else { return }
        guard let handle else { return }
        migrationTask?.cancel()
        let expected = generation
        let revision = state?.revision
        let requested = ViewRequest(view: "migration", locale: request.locale, snapshot_id: state?.snapshot_id, migration_details: true)
        migrationTask = Task { [weak self, client] in
            do {
                let response = try await client.read(handle: handle, request: requested)
                guard let self, self.generation == expected, self.state?.revision == revision, !Task.isCancelled, let comparison = response.migration else { return }
                self.migrationPreview = SemanticsPreview(profile: profile, comparison: comparison)
            } catch {
                guard let self, self.generation == expected, !Task.isCancelled else { return }
                self.errorMessage = (error as? BridgeError)?.message(locale: self.request.locale) ?? error.localizedDescription
            }
        }
    }

    func applySemanticsPreview() {
        guard let preview = migrationPreview else { return }
        migrationPreview = nil
        loadLedger(at: currentPath, semantics: preview.profile, inputIdentity: preview.comparison.input_identity)
    }

    func loadLedger(at path: String, semantics selected: String? = nil, inputIdentity: String = "") {
        let profile = selected ?? preferences?.string(forKey: "org.orangecount.semantics." + path) ?? "legacy"
        guard ["legacy", "v3"].contains(profile) else { return }
        semantics = profile
        generation &+= 1
        readGeneration &+= 1
        let expected = generation
        cancelTasks()
        migrationPreview = nil
        if let previous = handle { Task { [client] in await client.close(handle: previous) } }
        handle = nil
        navigationStack = []
        // Restore explicit reading preferences for this ledger only.
        preferencePath = path
        let saved = preferences?.dictionary(forKey: readingPreferenceKey(path)) as? [String: String]
        let valuation = ["at-cost", "units", "market-value"].contains(saved?["valuation"] ?? "") ? saved!["valuation"]! : "at-cost"
        analysisValuation = valuation
        let landing = selectedNav == .account ? NavItem.balanceSheet : selectedNav
        request = ViewRequest(view: landing.rawValue, valuation: landing == .journal || landing == .prices ? "at-cost" : valuation, currency: saved?["currency"] ?? "", locale: request.locale, window: landing == .journal ? "1m" : (landing == .dashboard ? "1y" : "all"))
        snapshot = nil
        state = nil
        currentPath = path
        errorMessage = nil
        showDiagnosticsSheet = false
        phase = .opening
        openTask = Task { [weak self, client] in
            do {
                let opened = try await client.open(path: path, semantics: profile, inputIdentity: inputIdentity)
                guard let self, self.generation == expected, !Task.isCancelled else {
                    await client.close(handle: opened.handle)
                    return
                }
                self.handle = opened.handle
                self.state = opened.state
                self.currentPath = opened.state.entry_path
                self.preferences?.set(profile, forKey: "org.orangecount.semantics." + path)
                self.preferences?.set(profile, forKey: "org.orangecount.semantics." + opened.state.entry_path)
                self.loadCurrentView()
                self.startPolling(handle: opened.handle, generation: expected)
            } catch {
                guard let self, self.generation == expected, !Task.isCancelled else { return }
                self.errorMessage = (error as? BridgeError)?.message(locale: self.request.locale) ?? error.localizedDescription
                self.phase = .failed
            }
        }
    }

    private func loadCurrentView() {
        guard let handle else { return }
        readGeneration &+= 1
        let expectedRead = readGeneration
        let expected = generation
        let requested = request
        readTask?.cancel()
        if visibleView == nil { phase = .loadingView }
        readTask = Task { [weak self, client] in
            do {
                let response = try await client.read(handle: handle, request: requested)
                guard let self, self.generation == expected, self.readGeneration == expectedRead,
                      self.request == requested, !Task.isCancelled else { return }
                if response.state.revision < (self.state?.revision ?? 0) {
                    self.loadCurrentView()
                    return
                }
                self.snapshot = response
                self.state = response.state
                self.errorMessage = nil
                self.phase = Self.phase(for: response.state)
                if response.state.valid { self.recordRecent(response.state.entry_path) }
            } catch {
                guard let self, self.generation == expected, self.readGeneration == expectedRead, !Task.isCancelled else { return }
                if case BridgeError.remote("snapshot-changed", _) = error {
                    self.updateFilters { _ in }
                    return
                }
                self.errorMessage = (error as? BridgeError)?.message(locale: self.request.locale) ?? error.localizedDescription
                self.phase = .failed
            }
        }
    }

    func updateFilters(_ change: (inout ViewRequest) -> Void) {
        var next = request
        change(&next)
        next.offset = 0
        next.snapshot_id = nil
        Self.clearHoldingSelection(&next)
        let readingChanged = next.currency != request.currency || next.valuation != request.valuation
        request = next
        if readingChanged && !currentPath.isEmpty {
            let saved = ["currency": next.currency, "valuation": next.view == "journal" || next.view == "prices" ? analysisValuation : next.valuation]
            preferences?.set(saved, forKey: readingPreferenceKey(currentPath))
            if !preferencePath.isEmpty { preferences?.set(saved, forKey: readingPreferenceKey(preferencePath)) }
        }
    }

    private func readingPreferenceKey(_ path: String) -> String {
        "org.orangecount.reading." + URL(fileURLWithPath: (path as NSString).expandingTildeInPath).standardizedFileURL.path
    }

    func journalPage(_ offset: Int) {
        guard let current = visibleView, let journal = current.journal else { return }
        var next = request
        next.offset = max(0, offset)
        next.limit = journal.limit
        next.snapshot_id = current.state.snapshot_id
        request = next
    }

    func showAccount(_ account: String) {
        guard handle != nil, !account.isEmpty else { return }
        guard request.view != "account" || request.account != account else { return }
        navigationStack.append(request)
        var next = request
        next.view = "account"
        next.account = account
        next.scope = "subtree"
        next.valuation = analysisValuation
        next.text = ""
        next.offset = 0
        next.snapshot_id = nil
        Self.clearHoldingParameters(&next)
        Self.clearPriceParameters(&next)
        request = next
    }

    func showHoldings(_ account: String) {
        guard handle != nil, !account.isEmpty else { return }
        navigationStack.append(request)
        var next = request
        let scope = next.account == account ? next.scope : nil
        Self.clearHoldingParameters(&next)
        Self.clearPriceParameters(&next)
        next.view = "holdings"
        next.account = account
        next.scope = scope ?? "subtree"
        next.group_by = "account"
        next.holding_section = "groups"
        next.valuation = analysisValuation
        next.text = ""
        next.offset = 0
        next.snapshot_id = nil
        next.sort = "ascending"
        request = next
    }

    func showHoldingLots(groupID: String) {
        showHoldingSelection(section: .lots, groupID: groupID)
    }

    func showHoldingEventLots(eventID: String, merged: Bool = false) {
        showHoldingSelection(section: .lots, eventID: eventID, lotKind: merged ? "merged" : "consumed")
    }

    func showHoldingHistory(groupID: String) {
        showHoldingSelection(section: .history, groupID: groupID)
    }

    private func showHoldingSelection(section: HoldingSection, groupID: String? = nil, eventID: String? = nil, lotKind: String? = nil) {
        guard handle != nil, request.view == "holdings", let current = visibleView,
              !(groupID ?? eventID ?? "").isEmpty else { return }
        navigationStack.append(request)
        var next = request
        next.holding_section = section.rawValue
        next.group_id = groupID
        next.event_id = eventID
        next.lot_kind = lotKind
        next.offset = 0
        next.snapshot_id = current.state.snapshot_id
        request = next
    }

    func setHoldingSection(_ section: HoldingSection) {
        guard request.view == "holdings", section != .lots else { return }
        updateFilters {
            $0.holding_section = section.rawValue
            $0.group_id = nil
            $0.event_id = nil
            $0.lot_kind = nil
        }
    }

    func holdingsPage(_ offset: Int, page: ProjectionPage) {
        guard request.view == "holdings", let current = visibleView else { return }
        var next = request
        next.offset = max(0, offset)
        next.limit = page.limit
        next.snapshot_id = current.state.snapshot_id
        request = next
    }

    func setHoldingSort(_ sort: String) {
        guard request.view == "holdings", ["ascending", "descending"].contains(sort) else { return }
        var next = request
        next.sort = sort
        next.offset = 0
        next.snapshot_id = next.group_id != nil || next.event_id != nil ? visibleView?.state.snapshot_id ?? next.snapshot_id : nil
        request = next
    }

    func showPrices(base: String = "", quote: String = "") {
        guard handle != nil, !base.isEmpty || quote.isEmpty else { return }
        guard request.view != "prices" || (request.price_base ?? "") != base || (request.price_quote ?? "") != quote else { return }
        navigationStack.append(request)
        var next = request
        Self.clearHoldingParameters(&next)
        Self.clearPriceParameters(&next)
        next.view = "prices"
        next.account = ""
        next.scope = nil
        next.text = ""
        next.valuation = "at-cost"
        next.offset = 0
        next.snapshot_id = nil
        next.sort = "ascending"
        next.price_section = base.isEmpty ? "commodities" : "history"
        next.price_base = base.isEmpty ? nil : base
        next.price_quote = quote.isEmpty ? nil : quote
        request = next
    }

    func pricePage(_ offset: Int, page: ProjectionPage) {
        guard request.view == "prices", let current = visibleView else { return }
        var next = request
        next.offset = max(0, offset)
        next.limit = page.limit
        next.snapshot_id = current.state.snapshot_id
        request = next
    }

    private static func clearPriceParameters(_ request: inout ViewRequest) {
        request.price_section = nil
        request.price_base = nil
        request.price_quote = nil
    }

    private static func clearHoldingSelection(_ request: inout ViewRequest) {
        guard request.view == "holdings" else { return }
        if request.holding_section == "lots" { request.holding_section = "groups" }
        request.group_id = nil
        request.event_id = nil
        request.lot_kind = nil
    }

    private static func clearHoldingParameters(_ request: inout ViewRequest) {
        request.group_by = nil
        request.holding_section = nil
        request.group_id = nil
        request.event_id = nil
        request.lot_kind = nil
    }

    func navigateBack() {
        guard var previous = navigationStack.popLast() else { return }
        previous.locale = request.locale
        previous.currency = request.currency
        previous.valuation = previous.view == "journal" || previous.view == "prices" ? "at-cost" : analysisValuation
        if let identity = previous.snapshot_id, identity != state?.snapshot_id {
            previous.offset = 0
            previous.snapshot_id = nil
            Self.clearHoldingSelection(&previous)
        }
        request = previous
    }

    func reloadSilently() {
        guard let handle else { return }
        let expected = generation
        reloadTask?.cancel()
        phase = .reloading
        reloadTask = Task { [weak self, client] in
            do {
                let state = try await client.reload(handle: handle)
                guard let self, self.generation == expected, !Task.isCancelled else { return }
                let changedSnapshot = self.state?.snapshot_id != state.snapshot_id
                if state.revision >= (self.state?.revision ?? 0) { self.state = state }
                if changedSnapshot && self.request.snapshot_id != nil {
                    self.updateFilters { _ in }
                    return
                }
                self.loadCurrentView()
            } catch {
                guard let self, self.generation == expected, !Task.isCancelled else { return }
                self.errorMessage = (error as? BridgeError)?.message(locale: self.request.locale) ?? error.localizedDescription
                self.phase = .failed
            }
        }
    }

    private func startPolling(handle: UInt64, generation expected: UInt64) {
        pollTask?.cancel()
        let delay = pollNanoseconds
        pollTask = Task { [weak self, client] in
            while !Task.isCancelled {
                do {
                    try await Task.sleep(nanoseconds: delay)
                    guard let revision = self?.state?.revision else { return }
                    let result = try await client.poll(handle: handle, after: revision)
                    guard let self, self.generation == expected, !Task.isCancelled else { return }
                    guard result.changed, result.state.revision >= (self.state?.revision ?? 0) else { continue }
                    let changedSnapshot = self.state?.snapshot_id != result.state.snapshot_id
                    self.state = result.state
                    if changedSnapshot && self.request.snapshot_id != nil {
                        self.updateFilters { _ in }
                        continue
                    }
                    if result.state.reloading { self.phase = .reloading }
                    else { self.loadCurrentView() }
                } catch is CancellationError { return }
                catch {
                    guard let self, self.generation == expected, !Task.isCancelled else { return }
                    self.errorMessage = (error as? BridgeError)?.message(locale: self.request.locale) ?? error.localizedDescription
                    self.phase = .failed
                    return
                }
            }
        }
    }

    func close() {
        generation &+= 1
        readGeneration &+= 1
        cancelTasks()
        if let previous = handle { Task { [client] in await client.close(handle: previous) } }
        handle = nil
        snapshot = nil
        state = nil
        navigationStack = []
        phase = .closed
    }

    private func cancelTasks() {
        openTask?.cancel()
        readTask?.cancel()
        pollTask?.cancel()
        reloadTask?.cancel()
        migrationTask?.cancel()
    }

    private func recordRecent(_ path: String) {
        let updated = Array(([path] + recentPaths.filter { $0 != path }).prefix(5))
        guard updated != recentPaths else { return }
        recentPaths = updated
        preferences?.set(updated, forKey: recentsKey)
    }

    private static func phase(for state: LedgerState) -> LedgerPhase {
        if state.reloading { return .reloading }
        if state.valid { return .ready }
        return state.has_snapshot ? .stale : .invalid
    }
}
