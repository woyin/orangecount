// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import AppKit
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
    case query

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
        case .query: return "查询工作台"
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
        case .query: return "Query workbench"
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
        case .query: return "terminal"
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

protocol LedgerQueryClient: Sendable {
    func executeQuery(handle: UInt64, request: QueryRequest) async throws -> QueryResponse
    func cancelQuery(handle: UInt64, requestID: UInt64) async throws
}

protocol LedgerQueryExportClient: Sendable {
    func captureQueryExport(handle: UInt64, request: QueryExportRequest) async throws -> QueryExportCapture
}

protocol LedgerAttachmentRootsClient: Sendable {
 func attachmentRoots(handle: UInt64) async throws -> AttachmentRoots
 func configureAttachmentRoots(handle: UInt64, request: AttachmentRootsRequest) async throws -> AttachmentRoots
}

protocol LedgerReportExportClient: Sendable {
    func captureReportExport(handle: UInt64, request: ReportExportRequest) async throws -> ReportExportCapture
}

protocol LedgerExportProtectionClient: Sendable {
    func exportProtection(handle: UInt64) async throws -> ExportProtection
}

enum QueryExportError: Error {
    case unavailable, changedExecution, invalidFormat
}

struct PreparedQueryExport: Sendable {
    let context: QueryContext
    let format: String
    fileprivate let file: AnalysisFileExport
    fileprivate let generation: UInt64
    fileprivate let handle: UInt64
}

// One executor per window bounds outstanding C work. Cancellation is checked
// on entry, so superseded queued reads never start. A synchronous C call that
// already began may finish; LedgerStore still checks generation before publish.
// Queries additionally use Go budget checkpoints and an independent cancel path.
actor NativeLedgerClient: LedgerSessionClient, LedgerQueryClient, LedgerQueryExportClient, LedgerExportProtectionClient, LedgerReportExportClient, LedgerAttachmentRootsClient {
 func attachmentRoots(handle: UInt64) async throws -> AttachmentRoots {
 try Task.checkCancellation(); return try BridgeClient.shared.attachmentRoots(handle: handle)
 }
 func configureAttachmentRoots(handle: UInt64, request: AttachmentRootsRequest) async throws -> AttachmentRoots {
 try Task.checkCancellation(); return try BridgeClient.shared.configureAttachmentRoots(handle: handle, request: request)
 }
    func open(path: String, semantics: String, inputIdentity: String) async throws -> OpenLedgerResult {
        try Task.checkCancellation()
        return try BridgeClient.shared.openLedger(path: Self.resolveEntry(path), semantics: semantics, inputIdentity: inputIdentity)
    }
    func read(handle: UInt64, request: ViewRequest) async throws -> LedgerView {
        try Task.checkCancellation()
        return try BridgeClient.shared.readView(handle: handle, request: request)
    }
    func executeQuery(handle: UInt64, request: QueryRequest) async throws -> QueryResponse {
        try Task.checkCancellation()
        return try await withTaskCancellationHandler {
            let response = try BridgeClient.shared.executeQuery(handle: handle, request: request)
            try Task.checkCancellation()
            return response
        } onCancel: {
            // Runs outside actor isolation, including while its synchronous C
            // execution is active. Go observes cancellation at budget checkpoints.
            try? BridgeClient.shared.cancelQuery(handle: handle, requestID: request.request_id)
        }
    }

    nonisolated func cancelQuery(handle: UInt64, requestID: UInt64) async throws {
        // A control call must not queue behind the C execution it interrupts.
        // Detached work also keeps this small synchronous bridge call off UI.
        try await Task.detached {
            try BridgeClient.shared.cancelQuery(handle: handle, requestID: requestID)
        }.value
    }

    func captureQueryExport(handle: UInt64, request: QueryExportRequest) async throws -> QueryExportCapture {
        try Task.checkCancellation()
        let capture = try BridgeClient.shared.captureQueryExport(handle: handle, request: request)
        try Task.checkCancellation()
        return capture
    }

    func captureReportExport(handle: UInt64, request: ReportExportRequest) async throws -> ReportExportCapture {
        try Task.checkCancellation()
        return try await withTaskCancellationHandler {
            do {
                let capture = try BridgeClient.shared.captureReportExport(handle: handle, request: request)
                try Task.checkCancellation()
                return capture
            } catch {
                if Task.isCancelled { throw CancellationError() }
                throw error
            }
        } onCancel: {
            try? BridgeClient.shared.cancelReportExport(handle: handle, captureID: request.capture_id)
        }
    }

    func exportProtection(handle: UInt64) async throws -> ExportProtection {
        try Task.checkCancellation()
        let protection = try BridgeClient.shared.exportProtection(handle: handle)
        try Task.checkCancellation()
        return protection
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

enum QueryPhase: Equatable {
    case idle, running, queued, succeeded, failed, cancelled, stale
}

enum LedgerPhase: Equatable {
    case idle, opening, loadingView, ready, reloading, stale, invalid, failed, closed
}

@MainActor
final class LedgerStore: ObservableObject {
    @Published private(set) var phase: LedgerPhase = .idle
    @Published private(set) var currentPath = ""
    @Published private(set) var snapshot: LedgerView?
    @Published private(set) var state: LedgerState? {
        didSet {
            if let previous = oldValue, let state,
               previous.snapshot_id != state.snapshot_id || previous.semantics_profile != state.semantics_profile
                || previous.evaluation_revision != state.evaluation_revision {
                invalidateQuery(keepPrevious: true)
            }
        }
    }
    @Published var queryDraft = "SELECT date, account, number LIMIT 100"
    @Published private(set) var queryPhase: QueryPhase = .idle
    @Published private(set) var queryResponse: QueryResponse?
    @Published private(set) var lastSuccessfulQuery: QueryResponse?
    @Published private(set) var queryErrorMessage: String?
    @Published private(set) var errorMessage: String?
    @Published private(set) var recentPaths: [String]
    @Published private(set) var semantics = "legacy"
    @Published var migrationPreview: SemanticsPreview?
    @Published var showDiagnosticsSheet = false
    @Published private(set) var navigationStack: [ViewRequest] = []
    @Published var request = ViewRequest() {
        didSet {
            if request.view != "journal" && request.view != "prices" && request.view != "query" { analysisValuation = request.valuation }
            if oldValue.view == "query", request.view != "query", activeQuery != nil || pendingQuery != nil { cancelQuery() }
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
            next.journal_mode = nil; next.directive_kind = nil
            next.offset = 0
            next.snapshot_id = nil
            next.window = newValue == .journal ? "1m" : (newValue == .dashboard ? "1y" : "all")
            next.text = ""
            next.account = ""
            next.scope = nil
            Self.clearHoldingParameters(&next)
            Self.clearPriceParameters(&next)
            next.valuation = newValue == .journal || newValue == .prices || newValue == .query ? "at-cost" : analysisValuation
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
    var directivesPayload: DirectivesPayload? { visibleView?.directives }
    var trendsPayload: HistoricalTrendsPayload? { visibleView?.trends }
    var accountDetail: AccountDetailPayload? { visibleView?.account_detail }
    var holdingsPayload: HoldingsPayload? { visibleView?.holdings }
    var pricesPayload: PricesPayload? { visibleView?.prices }
    var queryCatalogue: QueryCatalogue? { visibleView?.query_catalogue }
    var canNavigateBack: Bool { !navigationStack.isEmpty }
    var visibleView: LedgerView? { snapshot?.request == request ? snapshot : nil }

    private struct QueryJob: Sendable {
        let handle: UInt64
        let generation: UInt64
        let semantics: String
        let evaluationRevision: String
        let request: QueryRequest
    }
    private let queryClient: (any LedgerQueryClient)?
    private var querySequence: UInt64 = 0
    private var latestQueryID: UInt64?
    private var activeQuery: QueryJob?
    private var pendingQuery: QueryJob?
    private var queryTask: Task<Void, Never>?
    @Published private(set) var attachmentRoots: AttachmentRoots?
    @Published private(set) var attachmentRootNotice: String?
    @Published private(set) var attachmentRootsBusy = false
    @Published var showAttachmentRootsSheet = false
    private let rootPreferences: AttachmentRootPreferences
    private var rootObserver: AnyCancellable?
    private var rootConfigurationGeneration: UInt64 = 0
    private var rootOperation: UInt64 = 0
    private let client: any LedgerSessionClient
    private let preferences: UserDefaults?
    private let pollNanoseconds: UInt64
    private let recentsKey = "org.orangecount.recent_ledgers"
    private var preferencePath = ""
    private var analysisValuation = "at-cost"
    private var handle: UInt64?
    private var exportWriters: [UUID: Task<Void, Error>] = [:]
    private var generation: UInt64 = 0 {
        didSet { if generation != oldValue { for writer in exportWriters.values { writer.cancel() } } }
    }
    private var readGeneration: UInt64 = 0
    private var started = false
    private var openTask: Task<Void, Never>?
    private var readTask: Task<Void, Never>?
    private var pollTask: Task<Void, Never>?
    private var reloadTask: Task<Void, Never>?
    private var migrationTask: Task<Void, Never>?

    init(client: any LedgerSessionClient = NativeLedgerClient(), preferences: UserDefaults? = .standard, pollNanoseconds: UInt64 = 250_000_000) {
        self.client = client
        self.queryClient = client as? any LedgerQueryClient
        self.preferences = preferences
 self.rootPreferences = AttachmentRootPreferences(defaults: preferences)
        self.pollNanoseconds = pollNanoseconds
        recentPaths = preferences?.stringArray(forKey: recentsKey) ?? []
        rootObserver = NotificationCenter.default.publisher(for: AttachmentRootPreferences.changed).sink { [weak self] note in
            guard let ledger = note.object as? String else { return }
            Task { @MainActor [weak self] in
                guard let self, AttachmentRootPreferences.ledgerIdentity(self.currentPath) == ledger else { return }
                await self.restoreAttachmentRoots()
            }
        }
    }

    deinit {
        for writer in exportWriters.values { writer.cancel() }
        queryTask?.cancel()
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

    var reportExportIdentity: ReportExportIdentity? {
        guard let view = visibleView, view.state.snapshot_id == state?.snapshot_id else { return nil }
        return view.report_export
    }

    func prepareReportExport(format: String) async throws -> PreparedReportExport {
        guard format == "analysis" || format == "csv" else { throw QueryExportError.invalidFormat }
        guard let identity = reportExportIdentity, let handle,
              let exportClient = client as? any LedgerReportExportClient else { throw QueryExportError.unavailable }
        let expectedGeneration = generation
        let capture = try await exportClient.captureReportExport(handle: handle,
            request: ReportExportRequest(projection_id: identity.projection_id, snapshot_id: identity.snapshot_id,
                format: format, capture_scope: identity.capture_scope))
        try Task.checkCancellation()
        guard generation == expectedGeneration, self.handle == handle,
              reportExportIdentity == identity, capture.identity == identity else { throw QueryExportError.changedExecution }
        let file = try AnalysisFileExport(content: capture.content, protectedPaths: capture.protected_paths)
        return PreparedReportExport(identity: identity, format: format, file: file,
            generation: expectedGeneration, handle: handle)
    }

    func saveReportExport(_ prepared: PreparedReportExport, to target: URL) async throws {
        try await saveCapturedFile(prepared.file, generation: prepared.generation, handle: prepared.handle, to: target)
    }

    var queryDraftChanged: Bool {
        queryResponse.map { $0.context.text != queryDraft } ?? false
    }
    var queryResultIsCurrent: Bool {
        queryPhase == .succeeded && queryResponse?.context.snapshot_id == state?.snapshot_id
            && queryResponse?.context.semantics_profile == semantics
            && queryResponse?.context.evaluation_revision == state?.evaluation_revision
    }

    // Capture before presenting a save panel. The returned bytes can then be
    // saved independently of subsequent editor, query, or snapshot changes.
    func captureQueryExport(format: String) async throws -> QueryExportCapture {
        guard format == "csv" || format == "analysis" else { throw QueryExportError.invalidFormat }
        guard queryResultIsCurrent, let response = queryResponse, response.execution != nil,
              let handle, let exportClient = client as? any LedgerQueryExportClient else {
            throw QueryExportError.unavailable
        }
        let expectedGeneration = generation
        let identity = response.context
        let capture = try await exportClient.captureQueryExport(handle: handle,
            request: QueryExportRequest(snapshot_id: identity.snapshot_id, request_id: identity.request_id,
                fingerprint: identity.fingerprint, format: format))
        try Task.checkCancellation()
        guard generation == expectedGeneration, self.handle == handle, queryResultIsCurrent,
              queryResponse?.context.request_id == identity.request_id,
              queryResponse?.context.fingerprint == identity.fingerprint,
              capture.context.snapshot_id == identity.snapshot_id,
              capture.context.request_id == identity.request_id,
              capture.context.fingerprint == identity.fingerprint, capture.format == format else {
            throw QueryExportError.changedExecution
        }
        return capture
    }

    func refreshExportProtection() async throws -> ExportProtection {
        guard let handle, let protectionClient = client as? any LedgerExportProtectionClient else {
            throw QueryExportError.unavailable
        }
        let expectedGeneration = generation
        let protection = try await protectionClient.exportProtection(handle: handle)
        try Task.checkCancellation()
        guard generation == expectedGeneration, self.handle == handle else { throw QueryExportError.changedExecution }
        return protection
    }

    func prepareQueryExport(format: String) async throws -> PreparedQueryExport {
        let expectedGeneration = generation
        guard let handle else { throw QueryExportError.unavailable }
        let capture = try await captureQueryExport(format: format)
        let file = try AnalysisFileExport(content: capture.content, protectedPaths: capture.protected_paths)
        guard generation == expectedGeneration, self.handle == handle else { throw QueryExportError.changedExecution }
        return PreparedQueryExport(context: capture.context, format: format, file: file,
            generation: expectedGeneration, handle: handle)
    }

    func saveQueryExport(_ prepared: PreparedQueryExport, to target: URL) async throws {
        try await saveCapturedFile(prepared.file, generation: prepared.generation, handle: prepared.handle, to: target)
    }

    private func saveCapturedFile(_ capturedFile: AnalysisFileExport, generation expectedGeneration: UInt64,
                                  handle expectedHandle: UInt64, to target: URL) async throws {
        guard generation == expectedGeneration, handle == expectedHandle else { throw QueryExportError.changedExecution }
        let protection = try await refreshExportProtection()
        try Task.checkCancellation()
        guard generation == expectedGeneration, handle == expectedHandle else { throw QueryExportError.changedExecution }
        let file = try capturedFile.addingProtectedPaths(protection.protected_paths)
        let writer = Task.detached { try file.save(to: target) }
        let writerID = UUID()
        exportWriters[writerID] = writer
        defer { exportWriters.removeValue(forKey: writerID) }
        try await withTaskCancellationHandler { try await writer.value } onCancel: { writer.cancel() }
    }

    @discardableResult
    func useSavedQuery(id: String) -> Bool {
        guard let catalogue = queryCatalogue, catalogue.snapshot_id == state?.snapshot_id,
              let saved = catalogue.queries.first(where: { $0.id == id }) else { return false }
        queryDraft = saved.text
        return true
    }

    @discardableResult
    func useQueryPreset(id: String, inPeriod: Bool = false) -> Bool {
        guard let catalogue = queryCatalogue, catalogue.snapshot_id == state?.snapshot_id,
              let preset = (inPeriod ? (catalogue.period?.presets ?? []) : catalogue.presets).first(where: { $0.id == id }) else { return false }
        queryDraft = preset.text
        return true
    }

    func queryCataloguePage(offset: Int) {
        guard request.view == "query", let catalogue = queryCatalogue,
              catalogue.snapshot_id == state?.snapshot_id, offset >= 0 else { return }
        var next = request
        next.offset = offset
        next.limit = catalogue.page.limit
        next.snapshot_id = catalogue.snapshot_id
        request = next
    }

    func executeQuery() {
        queryResponse = nil
        queryErrorMessage = nil
        guard let handle, let state, state.has_snapshot, !state.reloading,
              queryClient != nil, querySequence < UInt64.max else {
            latestQueryID = nil
            pendingQuery = nil
            cancelActiveQuery()
            queryPhase = .failed
            queryErrorMessage = request.locale == "zh-CN" ? "尚无可执行查询的有效会话。" : "No queryable ledger session is available."
            return
        }
        querySequence += 1
        let job = QueryJob(handle: handle, generation: generation, semantics: semantics,
            evaluationRevision: state.evaluation_revision, request: QueryRequest(snapshot_id: state.snapshot_id, request_id: querySequence, text: queryDraft))
        latestQueryID = querySequence
        if activeQuery != nil {
            pendingQuery = job
            queryPhase = .queued
            cancelActiveQuery()
        } else {
            launchQuery(job)
        }
    }

    func cancelQuery() {
        latestQueryID = nil
        pendingQuery = nil
        cancelActiveQuery()
        queryResponse = nil
        queryErrorMessage = nil
        queryPhase = .cancelled
    }

    private func cancelActiveQuery() {
        queryTask?.cancel()
        if let job = activeQuery, let queryClient {
            Task { try? await queryClient.cancelQuery(handle: job.handle, requestID: job.request.request_id) }
        }
    }

    private func invalidateQuery(keepPrevious: Bool) {
        let hadQuery = activeQuery != nil || pendingQuery != nil || queryResponse != nil || lastSuccessfulQuery != nil
        latestQueryID = nil
        pendingQuery = nil
        cancelActiveQuery()
        queryErrorMessage = nil
        if !keepPrevious { queryResponse = nil; lastSuccessfulQuery = nil }
        queryPhase = keepPrevious && hadQuery ? .stale : .idle
    }

    private func launchQuery(_ job: QueryJob) {
        guard let queryClient else { return }
        activeQuery = job
        queryPhase = .running
        queryTask = Task { [weak self] in
            let outcome: Result<QueryResponse, Error>
            do { outcome = .success(try await queryClient.executeQuery(handle: job.handle, request: job.request)) }
            catch { outcome = .failure(error) }
            self?.finishQuery(job, outcome: outcome)
        }
    }

    private func finishQuery(_ job: QueryJob, outcome: Result<QueryResponse, Error>) {
        guard activeQuery?.request.request_id == job.request.request_id else { return }
        activeQuery = nil
        queryTask = nil
        if latestQueryID == job.request.request_id, generation == job.generation,
           handle == job.handle, state?.snapshot_id == job.request.snapshot_id {
            switch outcome {
            case .success(let response):
                guard response.context.request_id == job.request.request_id,
                      response.context.snapshot_id == job.request.snapshot_id,
                      response.context.text == job.request.text,
                      response.context.semantics_profile == job.semantics,
                      response.context.evaluation_revision == job.evaluationRevision else {
                    queryPhase = .failed
                    queryErrorMessage = "Query response does not match the captured execution."
                    return
                }
                queryResponse = response
                if response.execution != nil {
                    lastSuccessfulQuery = response
                    queryPhase = .succeeded
                } else {
                    queryErrorMessage = response.error?.message
                    queryPhase = response.error?.code == "cancelled" ? .cancelled : .failed
                }
            case .failure(let error):
                queryResponse = nil
                queryErrorMessage = (error as? BridgeError)?.message(locale: request.locale) ?? error.localizedDescription
                queryPhase = error is CancellationError ? .cancelled : .failed
            }
        }
        if let pending = pendingQuery {
            pendingQuery = nil
            if pending.generation == generation, pending.handle == handle,
               pending.request.snapshot_id == state?.snapshot_id, latestQueryID == pending.request.request_id {
                launchQuery(pending)
            }
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
        request = ViewRequest(view: landing.rawValue, valuation: landing == .journal || landing == .prices || landing == .query ? "at-cost" : valuation, currency: saved?["currency"] ?? "", locale: request.locale, window: landing == .journal ? "1m" : (landing == .dashboard ? "1y" : "all"))
        snapshot = nil
        state = nil
        currentPath = path
 attachmentRoots = nil; attachmentRootNotice = nil; attachmentRootsBusy = false; rootConfigurationGeneration = 0; rootOperation &+= 1
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
                await self.restoreAttachmentRoots()
                guard self.generation == expected, !Task.isCancelled else { return }
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

    private func restoreAttachmentRoots() async {
        guard let handle, let rootsClient = client as? any LedgerAttachmentRootsClient else { return }
        let expected = generation
        rootOperation &+= 1
        let operation = rootOperation
        do {
            let restored = try rootPreferences.restore(currentPath)
            if attachmentRoots != nil && restored.generation <= rootConfigurationGeneration { return }
            let current = try await rootsClient.attachmentRoots(handle: handle)
            guard generation == expected, self.handle == handle, rootOperation == operation else { return }
            let updated = try await rootsClient.configureAttachmentRoots(handle: handle,
                request: AttachmentRootsRequest(policy_revision: current.policy_revision, additional_roots: restored.roots))
            guard generation == expected, self.handle == handle, rootOperation == operation else { return }
            attachmentRoots = updated
            rootConfigurationGeneration = restored.generation
            attachmentRootNotice = restored.invalid.isEmpty ? nil : "部分目录需要重新选择授权 / Some directories require renewed selection: " + restored.invalid.joined(separator: ", ")
            loadCurrentView()
        } catch {
            guard generation == expected, self.handle == handle, rootOperation == operation else { return }
            attachmentRootNotice = "附件目录恢复失败 / Attachment directory restore failed: " + error.localizedDescription
        }
    }

    func updateAttachmentRoots(_ roots: [String], persist: Bool = true, expectedRevision: UInt64? = nil) async {
        guard !attachmentRootsBusy, let handle, let current = attachmentRoots,
              let rootsClient = client as? any LedgerAttachmentRootsClient else { return }
        guard expectedRevision == nil || expectedRevision == current.policy_revision else { attachmentRootNotice = "目录策略已变化，请重新选择 / Directory policy changed; select again"; return }
        let expected = generation
        rootOperation &+= 1
        let operation = rootOperation
        attachmentRootsBusy = true
        defer { if generation == expected { attachmentRootsBusy = false } }
        do {
            let prepared = persist ? try rootPreferences.prepare(roots, ledger: currentPath) : nil
            let updated = try await rootsClient.configureAttachmentRoots(handle: handle,
                request: AttachmentRootsRequest(policy_revision: current.policy_revision, additional_roots: prepared?.directories.map(\.path) ?? roots))
            guard generation == expected, self.handle == handle, rootOperation == operation else { return }
            attachmentRoots = updated
            attachmentRootNotice = nil
            if let prepared {
                rootConfigurationGeneration = prepared.generation
                do { try rootPreferences.save(prepared, ledger: currentPath) }
                catch { attachmentRootNotice = "目录本次有效，但未保存；其他窗口未同步 / Directories are active for this session but were not saved: " + error.localizedDescription }
            }
            loadCurrentView()
        } catch {
            guard generation == expected, self.handle == handle, rootOperation == operation else { return }
            attachmentRootNotice = "目录配置未更新 / Directory configuration was not updated: " + error.localizedDescription
        }
    }

    func showDocuments() {
        updateFilters { $0.journal_mode = "all"; $0.directive_kind = "document"; $0.window = "all"; $0.date_from = ""; $0.date_to = ""; $0.text = ""; $0.account = "" }
    }

    func openDocument(_ attachment: Attachment, reveal: Bool) {
        guard !attachmentRootsBusy, attachment.status == "available", attachment.policy_revision == attachmentRoots?.policy_revision, let handle, let identity = state?.snapshot_id else { return }
        let expected = generation
        let expectedRead = readGeneration
        let actionRequest = ViewRequest(view: "journal", snapshot_id: identity, journal_mode: "all", document_id: attachment.id, document_policy_revision: attachment.policy_revision)
        Task { [weak self, client] in
            do {
                let response = try await client.read(handle: handle, request: actionRequest)
                guard let self, self.generation == expected, self.readGeneration == expectedRead,
                      self.state?.snapshot_id == identity, let action = response.document_action,
                      action.id == attachment.id, action.path == attachment.path,
                      !self.attachmentRootsBusy, action.policy_revision == self.attachmentRoots?.policy_revision else { return }
                let url = URL(fileURLWithPath: action.path)
                guard (try url.resourceValues(forKeys: [.isAliasFileKey])).isAliasFile != true else {
 self.errorMessage = "不支持 Finder 替身附件 / Finder alias attachments are unsupported"; return
 }
                if reveal { NSWorkspace.shared.activateFileViewerSelecting([url]) }
                else if !NSWorkspace.shared.open(url) { self.errorMessage = "无法打开附件 / Unable to open attachment" }
            } catch {
                guard let self, self.generation == expected, self.readGeneration == expectedRead else { return }
                self.errorMessage = "附件已变化或不可访问，请重新加载账本 / Attachment changed or unavailable; reload the ledger"
            }
        }
    }

    func selectJournalMode(_ mode: String) {
        guard request.view == "journal", ["transactions", "all", "config"].contains(mode) else { return }
        updateFilters { $0.journal_mode = mode; $0.directive_kind = nil; $0.text = ""; $0.account = ""; $0.sort = "" }
    }

    func journalPage(_ offset: Int) {
        guard let current = visibleView, let limit = current.journal?.limit ?? current.directives?.page.limit else { return }
        var next = request
        next.offset = max(0, offset)
        next.limit = limit
        next.snapshot_id = current.state.snapshot_id
        request = next
    }

    func showAccount(_ account: String) {
        guard handle != nil, !account.isEmpty else { return }
        guard request.view != "account" || request.account != account else { return }
        navigationStack.append(request)
        var next = request
        next.view = "account"
            next.journal_mode = nil; next.directive_kind = nil
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
            next.journal_mode = nil; next.directive_kind = nil
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
            next.journal_mode = nil; next.directive_kind = nil
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
        if activeQuery != nil || pendingQuery != nil { invalidateQuery(keepPrevious: true) }
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
        invalidateQuery(keepPrevious: false)
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

struct PreparedReportExport: Sendable {
    let identity: ReportExportIdentity
    let format: String
    fileprivate let file: AnalysisFileExport
    fileprivate let generation: UInt64
    fileprivate let handle: UInt64
}
