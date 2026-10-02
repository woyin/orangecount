// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import Foundation

struct StateTestFailure: Error { let message: String }

actor ControlledClient: LedgerSessionClient, LedgerQueryClient, LedgerQueryExportClient {
    private var pending: [String: CheckedContinuation<OpenLedgerResult, Error>] = [:]
    private var paths: [UInt64: String] = [:]
    private var closed: Set<UInt64> = []
    private var polls: [UInt64: Int] = [:]
    private var revisions: [UInt64: UInt64] = [:]
    private var pollsAfterClose: [UInt64: Int] = [:]
    private var holdReads = false
    private var reads: [String: (UInt64, ViewRequest, CheckedContinuation<LedgerView, Error>)] = [:]

    func open(path: String, semantics: String, inputIdentity: String) async throws -> OpenLedgerResult {
        try await withCheckedThrowingContinuation { pending[path] = $0 }
    }

    func hasPending(_ path: String) -> Bool { pending[path] != nil }
    func isClosed(_ handle: UInt64) -> Bool { closed.contains(handle) }
    func pollCount(_ handle: UInt64) -> Int { polls[handle, default: 0] }
    func closedPollCount(_ handle: UInt64) -> Int { pollsAfterClose[handle, default: 0] }
    func suspendReads() { holdReads = true }
    func hasRead(_ view: String, currency: String? = nil, dateTo: String? = nil, groupBy: String? = nil, section: String? = nil, priceQuote: String? = nil) -> Bool {
        reads.values.contains { matches($0.1, view, currency, dateTo, groupBy, section) && (priceQuote == nil || $0.1.price_quote == priceQuote) }
    }
    func finishRead(_ view: String, currency: String? = nil, dateTo: String? = nil, groupBy: String? = nil, section: String? = nil, priceQuote: String? = nil) {
        guard let key = reads.first(where: { matches($0.value.1, view, currency, dateTo, groupBy, section) && (priceQuote == nil || $0.value.1.price_quote == priceQuote) })?.key,
              let (handle, request, continuation) = reads.removeValue(forKey: key) else { return }
        continuation.resume(returning: viewFor(handle: handle, request: request))
    }
    private func matches(_ request: ViewRequest, _ view: String, _ currency: String?, _ date: String?, _ grouping: String?, _ section: String?) -> Bool {
        request.view == view && (currency == nil || request.currency == currency) && (date == nil || request.date_to == date) && (grouping == nil || request.group_by == grouping) && (section == nil || request.holding_section == section)
    }

    func complete(_ path: String, handle: UInt64) {
        paths[handle] = path
        pending.removeValue(forKey: path)?.resume(returning: OpenLedgerResult(handle: handle, state: state(handle)))
    }

    private func state(_ handle: UInt64) -> LedgerState {
        let revision = revisions[handle, default: 1]
        return LedgerState(revision: revision, snapshot_id: revision == 1 ? "snapshot-\(handle)" : "snapshot-\(handle)-\(revision)", entry_path: paths[handle]!, valid: true, has_snapshot: true, reloading: false, build_count: revision, evaluation_revision: "legacy-1")
    }

    func read(handle: UInt64, request: ViewRequest) async throws -> LedgerView {
        if holdReads {
            return try await withCheckedThrowingContinuation { reads[UUID().uuidString] = (handle, request, $0) }
        }
        return viewFor(handle: handle, request: request)
    }

    private func viewFor(handle: UInt64, request: ViewRequest) -> LedgerView {
        let summary = LedgerSummary(valid: true, entry_path: paths[handle], title: paths[handle], operating_currencies: ["USD"], accounts: [], entries_count: 0, error_count: 0, errors: [], transactions_count: 0, diagnostics: [])
        return LedgerView(state: state(handle), request: request, summary: summary, tree: nil, journal: nil, trends: nil)
    }

    func poll(handle: UInt64, after: UInt64) async throws -> PollResult {
        polls[handle, default: 0] += 1
        if closed.contains(handle) { pollsAfterClose[handle, default: 0] += 1 }
        return PollResult(changed: false, state: state(handle))
    }

    private var queries: [UInt64: (UInt64, QueryRequest, CheckedContinuation<QueryResponse, Error>)] = [:]
    private var queryIDs: [UInt64] = []
    private var cancelledQueries: Set<UInt64> = []
    func executeQuery(handle: UInt64, request: QueryRequest) async throws -> QueryResponse {
        queryIDs.append(request.request_id)
        return try await withCheckedThrowingContinuation { queries[request.request_id] = (handle, request, $0) }
    }
    func cancelQuery(handle: UInt64, requestID: UInt64) async throws { cancelledQueries.insert(requestID) }
    func hasQuery(_ id: UInt64) -> Bool { queries[id] != nil }
    func submittedQueries() -> [UInt64] { queryIDs }
    func queryCancelled(_ id: UInt64) -> Bool { cancelledQueries.contains(id) }
    func finishQuery(_ id: UInt64, failed: Bool = false) throws {
        guard let (_, request, continuation) = queries.removeValue(forKey: id) else { return }
        let context: [String: Any] = ["snapshot_id": request.snapshot_id, "request_id": request.request_id,
            "text": request.text, "semantics_profile": "legacy", "evaluation_revision": "legacy-1",
            "fingerprint": String(repeating: "a", count: 64)]
        let execution: [String: Any] = ["result": ["contract": "beanquery-0.2.0-typed-v1",
            "columns": [["id": "c0", "name": "v", "type": "integer", "nullable": false]],
            "rows": [[ ["type": "integer", "value": "7"] ]]],
            "returned_rows": 1, "row_limit": 10000, "truncated": false]
        let error: [String: Any] = ["code": "unknown-function", "message": "Unknown function", "phase": "compile", "range": NSNull()]
        let executionValue: Any = failed ? NSNull() : execution
        let errorValue: Any = failed ? error : NSNull()
        let data = try JSONSerialization.data(withJSONObject: ["context": context,
            "execution": executionValue, "error": errorValue])
        continuation.resume(returning: try JSONDecoder().decode(QueryResponse.self, from: data))
    }

    func advanceSnapshot(_ handle: UInt64) { revisions[handle] = revisions[handle, default: 1] + 1 }

    private var pendingExport: (QueryExportRequest, CheckedContinuation<QueryExportCapture, Error>)?
    func captureQueryExport(handle: UInt64, request: QueryExportRequest) async throws -> QueryExportCapture {
        try await withCheckedThrowingContinuation { pendingExport = (request, $0) }
    }
    func hasExport() -> Bool { pendingExport != nil }
    func finishExport() {
        guard let (request, continuation) = pendingExport else { return }
        pendingExport = nil
        continuation.resume(returning: QueryExportCapture(context: QueryContext(snapshot_id: request.snapshot_id,
            semantics_profile: "legacy", evaluation_revision: "legacy-1", request_id: request.request_id,
            text: "SELECT 1", fingerprint: request.fingerprint), format: request.format, content: Data("v\n7\n".utf8)))
    }
    func reload(handle: UInt64) async throws -> LedgerState { state(handle) }
    func close(handle: UInt64) async { closed.insert(handle) }
}

@main
struct StateTests {
    @MainActor static func waitUntil(_ condition: () async -> Bool) async throws {
        for _ in 0..<400 {
            if await condition() { return }
            try await Task.sleep(nanoseconds: 5_000_000)
        }
        throw StateTestFailure(message: "Timed out waiting for store state")
    }

    @MainActor static func verifyQueryState() async throws {
        let client = ControlledClient()
        let store = LedgerStore(client: client, preferences: nil, pollNanoseconds: 60_000_000_000)
        defer { store.close() }
        store.loadLedger(at: "query-A")
        try await waitUntil { await client.hasPending("query-A") }
        await client.complete("query-A", handle: 20)
        try await waitUntil { store.phase == .ready }
        store.selectedNav = .query
        try await waitUntil { store.visibleView?.request.view == "query" }
        guard store.queryPhase == .idle else { throw StateTestFailure(message: "Query navigation executed the draft automatically") }
        store.queryDraft = "SELECT 1"
        store.executeQuery()
        try await waitUntil { await client.hasQuery(1) }
        store.queryDraft = "SELECT 2"
        store.executeQuery()
        store.queryDraft = "SELECT 3"
        store.executeQuery()
        try await waitUntil { await client.queryCancelled(1) }
        guard store.queryPhase == .queued else { throw StateTestFailure(message: "Latest query was not queued") }
        try await client.finishQuery(1)
        try await waitUntil { await client.hasQuery(3) }
        guard await client.submittedQueries() == [1, 3], store.queryResponse == nil else {
            throw StateTestFailure(message: "Intermediate query executed or obsolete success published")
        }
        try await client.finishQuery(3)
        try await waitUntil { store.queryPhase == .succeeded }
        guard store.queryResponse?.context.text == "SELECT 3", store.lastSuccessfulQuery?.context.request_id == 3 else {
            throw StateTestFailure(message: "Successful query lost its captured draft identity")
        }
        store.queryDraft = "SELECT 4"
        guard store.queryDraftChanged else { throw StateTestFailure(message: "Editing draft changed captured execution") }
        store.executeQuery()
        try await waitUntil { await client.hasQuery(4) }
        store.cancelQuery()
        try await client.finishQuery(4)
        try await Task.sleep(nanoseconds: 20_000_000)
        guard store.queryPhase == .cancelled, store.queryResponse == nil, store.lastSuccessfulQuery?.context.request_id == 3 else {
            throw StateTestFailure(message: "Late cancelled result replaced execution state")
        }
        store.executeQuery()
        try await waitUntil { await client.hasQuery(5) }
        try await client.finishQuery(5, failed: true)
        try await waitUntil { store.queryPhase == .failed }
        guard store.queryResponse?.execution == nil, store.queryResponse?.error?.code == "unknown-function" else {
            throw StateTestFailure(message: "Query error masqueraded as previous successful result")
        }
        store.executeQuery()
        try await waitUntil { await client.hasQuery(6) }
        store.loadLedger(at: "query-B")
        try await waitUntil { await client.hasPending("query-B") }
        await client.complete("query-B", handle: 21)
        try await client.finishQuery(6)
        try await waitUntil { store.phase == .ready }
        guard store.queryResponse == nil, store.lastSuccessfulQuery == nil else {
            throw StateTestFailure(message: "Query result crossed ledger boundary")
        }
        store.executeQuery()
        try await waitUntil { await client.hasQuery(7) }
        store.reloadSilently()
        try await waitUntil { await client.queryCancelled(7) }
        try await client.finishQuery(7)
        try await Task.sleep(nanoseconds: 20_000_000)
        guard store.queryPhase == .stale, store.queryResponse == nil else {
            throw StateTestFailure(message: "Reload accepted an in-flight query result")
        }
        store.executeQuery()
        try await waitUntil { await client.hasQuery(8) }
        try await client.finishQuery(8)
        try await waitUntil { store.queryResultIsCurrent }
        await client.advanceSnapshot(21)
        store.reloadSilently()
        try await waitUntil { store.state?.snapshot_id == "snapshot-21-2" }
        guard store.queryPhase == .stale, !store.queryResultIsCurrent,
              store.queryResponse?.context.snapshot_id == "snapshot-21" else {
            throw StateTestFailure(message: "A new snapshot relabelled old query success")
        }
        store.executeQuery()
        try await waitUntil { await client.hasQuery(9) }
        store.selectedNav = .balanceSheet
        try await client.finishQuery(9)
        try await Task.sleep(nanoseconds: 20_000_000)
        guard store.queryPhase == .cancelled, store.queryResponse == nil else {
            throw StateTestFailure(message: "Leaving workbench accepted an in-flight result")
        }
        store.selectedNav = .query
        store.executeQuery()
        try await waitUntil { await client.hasQuery(10) }
        store.close()
        try await client.finishQuery(10)
        try await Task.sleep(nanoseconds: 20_000_000)
        guard store.queryPhase == .idle, store.queryResponse == nil, store.lastSuccessfulQuery == nil else {
            throw StateTestFailure(message: "Closed store accepted a late query result")
        }
        print("PASS: query reload cancels active work, snapshot changes mark prior results stale, and close rejects late responses")
        print("PASS: query current/latest-pending, captured draft, cancellation, failure and ledger isolation")
    }

    @MainActor static func verifyLateQueryExports() async throws {
        for transition in ["query", "snapshot", "close"] {
            let client = ControlledClient()
            let store = LedgerStore(client: client, preferences: nil, pollNanoseconds: 60_000_000_000)
            store.loadLedger(at: "export")
            try await waitUntil { await client.hasPending("export") }
            await client.complete("export", handle: 40)
            try await waitUntil { store.phase == .ready }
            store.queryDraft = "SELECT 1"
            store.executeQuery()
            try await waitUntil { await client.hasQuery(1) }
            try await client.finishQuery(1)
            try await waitUntil { store.queryResultIsCurrent }
            let capture = Task { @MainActor in
                do { _ = try await store.captureQueryExport(format: "csv"); return false }
                catch QueryExportError.changedExecution { return true }
                catch { return false }
            }
            try await waitUntil { await client.hasExport() }
            switch transition {
            case "query":
                store.executeQuery()
                try await waitUntil { await client.hasQuery(2) }
                try await client.finishQuery(2)
                try await waitUntil { store.queryResultIsCurrent }
            case "snapshot":
                await client.advanceSnapshot(40)
                store.reloadSilently()
                try await waitUntil { store.state?.snapshot_id == "snapshot-40-2" }
            default: store.close()
            }
            await client.finishExport()
            guard await capture.value else { throw StateTestFailure(message: "Late export accepted after \(transition)") }
            store.close()
        }
        print("PASS: query export capture rejects late bytes after new execution, snapshot change and window close")
    }

    @MainActor static func main() async {
        do {
            try await verifyQueryState()
            try await verifyLateQueryExports()
            let client = ControlledClient()
            let store = LedgerStore(client: client, preferences: nil, pollNanoseconds: 60_000_000_000)
            store.loadLedger(at: "A")
            try await waitUntil { await client.hasPending("A") }
            store.loadLedger(at: "B")
            try await waitUntil { await client.hasPending("B") }
            await client.complete("B", handle: 2)
            try await waitUntil { store.summary?.title == "B" }
            await client.complete("A", handle: 1)
            try await waitUntil { await client.isClosed(1) }
            guard store.currentPath == "B", store.summary?.title == "B" else {
                throw StateTestFailure(message: "A late response replaced ledger B")
            }
            await client.suspendReads()
            store.selectedNav = .journal
            try await waitUntil { await client.hasRead("journal") }
            store.selectedNav = .dashboard
            try await waitUntil { await client.hasRead("dashboard") }
            await client.finishRead("dashboard")
            try await waitUntil { store.visibleView?.request.view == "dashboard" }
            await client.finishRead("journal")
            try await Task.sleep(nanoseconds: 20_000_000)
            guard store.selectedNav == .dashboard, store.visibleView?.request.view == "dashboard" else {
                throw StateTestFailure(message: "A late report response replaced the selected dashboard")
            }
            print("PASS: navigation and late view responses preserve the latest request")
            store.close()
            try await waitUntil { await client.isClosed(2) }
            guard store.phase == .closed else { throw StateTestFailure(message: "Store failed to close") }
            print("PASS: late open results cannot replace the current ledger; all handles close")

            let abandonedClient = ControlledClient()
            let abandoned = LedgerStore(client: abandonedClient, preferences: nil)
            abandoned.loadLedger(at: "abandoned")
            try await waitUntil { await abandonedClient.hasPending("abandoned") }
            abandoned.close()
            await abandonedClient.complete("abandoned", handle: 3)
            try await waitUntil { await abandonedClient.isClosed(3) }
            guard abandoned.snapshot == nil, abandoned.phase == .closed else {
                throw StateTestFailure(message: "A closed window accepted an in-flight open")
            }
            print("PASS: closing during an open releases its eventual handle")

            let windows = ControlledClient()
            var left: LedgerStore? = LedgerStore(client: windows, preferences: nil)
            let right = LedgerStore(client: windows, preferences: nil)
            left!.loadLedger(at: "left")
            right.loadLedger(at: "right")
            try await waitUntil {
                let leftReady = await windows.hasPending("left")
                let rightReady = await windows.hasPending("right")
                return leftReady && rightReady
            }
            await windows.complete("left", handle: 4)
            await windows.complete("right", handle: 5)
            try await waitUntil { left?.phase == .ready && right.phase == .ready }
            left!.updateFilters { $0.text = "left-only" }
            guard right.request.text.isEmpty && right.currentPath == "right" else { throw StateTestFailure(message: "Window filters leaked into another session") }
            left = nil
            try await waitUntil { await windows.isClosed(4) }
            guard right.phase == .ready && right.currentPath == "right" else { throw StateTestFailure(message: "Releasing one window affected the other") }
            right.updateFilters { $0.text = "keep this search"; $0.date_from = "2026-01-03"; $0.date_to = "2026-01-03" }
            let unchanged = right.request
            right.selectedNav = right.selectedNav
            guard right.request == unchanged else { throw StateTestFailure(message: "Selecting the current page cleared its filters") }
            right.close()
            try await waitUntil { await windows.isClosed(5) }
            print("PASS: independent windows and store deallocation release their sessions")

            let pollingClient = ControlledClient()
            let pollingWindow = LedgerStore(client: pollingClient, preferences: nil, pollNanoseconds: 20_000_000)
            pollingWindow.loadLedger(at: "polling-window")
            try await waitUntil { await pollingClient.hasPending("polling-window") }
            await pollingClient.complete("polling-window", handle: 6)
            try await waitUntil { await pollingClient.pollCount(6) >= 2 }
            // Let the completed poll return before closing between intervals.
            try await Task.sleep(nanoseconds: 5_000_000)
            pollingWindow.close()
            try await waitUntil { await pollingClient.isClosed(6) }
            // Keep the Store alive, as a window host may do after dismissal.
            // Closing must stop polling independently of deallocation.
            try await Task.sleep(nanoseconds: 60_000_000)
            guard await pollingClient.closedPollCount(6) == 0,
                  pollingWindow.phase == .closed, pollingWindow.snapshot == nil else {
                throw StateTestFailure(message: "A closed window continued polling its released session")
            }
            print("PASS: a retained closed window stops polling its released session")

            let suite = "org.orangecount.tests.reading." + UUID().uuidString
            let preferences = UserDefaults(suiteName: suite)!
            defer { preferences.removePersistentDomain(forName: suite) }
            let readerClient = ControlledClient()
            let reader = LedgerStore(client: readerClient, preferences: preferences, pollNanoseconds: 60_000_000_000)
            reader.loadLedger(at: "reading-A")
            try await waitUntil { await readerClient.hasPending("reading-A") }
            await readerClient.complete("reading-A", handle: 7)
            try await waitUntil { reader.phase == .ready }
            reader.updateFilters { $0.currency = "EUR"; $0.valuation = "units" }
            reader.reloadSilently()
            try await waitUntil { reader.phase == .ready }
            guard reader.request.currency == "EUR", reader.request.valuation == "units" else {
                throw StateTestFailure(message: "Reload lost explicit reading preferences")
            }
            reader.loadLedger(at: "reading-B")
            try await waitUntil { await readerClient.hasPending("reading-B") }
            await readerClient.complete("reading-B", handle: 8)
            try await waitUntil { reader.phase == .ready }
            guard reader.request.currency == "", reader.request.valuation == "at-cost" else {
                throw StateTestFailure(message: "Another ledger inherited reading preferences")
            }
            reader.loadLedger(at: "reading-A")
            try await waitUntil { await readerClient.hasPending("reading-A") }
            await readerClient.complete("reading-A", handle: 9)
            try await waitUntil { reader.phase == .ready }
            guard reader.request.currency == "EUR", reader.request.valuation == "units" else {
                throw StateTestFailure(message: "Reopening a ledger lost its explicit currency and valuation")
            }
            reader.selectedNav = .prices
            reader.updateFilters { $0.currency = "GBP" }
            reader.loadLedger(at: "reading-A")
            try await waitUntil { await readerClient.hasPending("reading-A") }
            await readerClient.complete("reading-A", handle: 15)
            try await waitUntil { reader.phase == .ready }
            guard reader.request.view == "prices", reader.request.valuation == "at-cost" else { throw StateTestFailure(message: "Reload/reopen landed on prices with invalid analysis valuation") }
            reader.selectedNav = .balanceSheet
            guard reader.request.valuation == "units", reader.request.currency == "GBP" else { throw StateTestFailure(message: "Price preferences saved neutral at-cost instead of the real analysis mode") }
            reader.close()
            try await waitUntil { await readerClient.isClosed(9) }
            try await waitUntil { await readerClient.isClosed(15) }
            print("PASS: currency and valuation preferences persist per ledger across reload and reopen")

            let lateClient = ControlledClient()
            let lateStore = LedgerStore(client: lateClient, preferences: nil, pollNanoseconds: 60_000_000_000)
            lateStore.selectedNav = .dashboard
            lateStore.loadLedger(at: "late-trends")
            try await waitUntil { await lateClient.hasPending("late-trends") }
            await lateClient.complete("late-trends", handle: 10)
            try await waitUntil { lateStore.phase == .ready }
            await lateClient.suspendReads()
            lateStore.updateFilters { $0.currency = "USD"; $0.date_to = "2000-01-31" }
            try await waitUntil { await lateClient.hasRead("dashboard", currency: "USD") }
            lateStore.updateFilters { $0.currency = "EUR"; $0.date_to = "2000-02-29" }
            try await waitUntil { await lateClient.hasRead("dashboard", currency: "EUR") }
            await lateClient.finishRead("dashboard", currency: "EUR")
            try await waitUntil { lateStore.visibleView?.request.currency == "EUR" }
            await lateClient.finishRead("dashboard", currency: "USD")
            try await Task.sleep(nanoseconds: 20_000_000)
            guard lateStore.visibleView?.request.currency == "EUR", lateStore.visibleView?.request.date_to == "2000-02-29" else {
                throw StateTestFailure(message: "Late currency/cutoff response replaced the current trend request")
            }
            lateStore.close()
            try await waitUntil { await lateClient.isClosed(10) }
            print("PASS: late same-view currency and cutoff responses cannot replace current trends")

            let accountClient = ControlledClient()
            let accountStore = LedgerStore(client: accountClient, preferences: nil, pollNanoseconds: 60_000_000_000)
            accountStore.loadLedger(at: "accounts")
            try await waitUntil { await accountClient.hasPending("accounts") }
            await accountClient.complete("accounts", handle: 11)
            try await waitUntil { accountStore.phase == .ready }
            accountStore.updateFilters { $0.text = "keep search"; $0.date_to = "2000-01-03" }
            try await waitUntil { accountStore.phase == .ready }
            let origin = accountStore.request
            await accountClient.suspendReads()
            accountStore.showAccount("Assets:Test")
            try await waitUntil { await accountClient.hasRead("account") }
            guard accountStore.request.account == "Assets:Test", accountStore.request.scope == "subtree", accountStore.request.date_to == origin.date_to, accountStore.request.text.isEmpty else {
                throw StateTestFailure(message: "Account navigation lost shared dates or reused text as an accounting filter")
            }
            accountStore.navigateBack()
            try await waitUntil { await accountClient.hasRead("balance_sheet") }
            await accountClient.finishRead("balance_sheet")
            try await waitUntil { accountStore.visibleView != nil }
            await accountClient.finishRead("account")
            try await Task.sleep(nanoseconds: 20_000_000)
            guard accountStore.request == origin, !accountStore.canNavigateBack, accountStore.visibleView?.request == origin else {
                throw StateTestFailure(message: "Back failed to restore the origin or accepted late account response")
            }
            accountStore.close()
            try await waitUntil { await accountClient.isClosed(11) }
            print("PASS: account navigation restores origin filters and rejects detail arriving after Back")

            let modeClient = ControlledClient()
            let modeStore = LedgerStore(client: modeClient, preferences: nil, pollNanoseconds: 60_000_000_000)
            modeStore.loadLedger(at: "market-navigation")
            try await waitUntil { await modeClient.hasPending("market-navigation") }
            await modeClient.complete("market-navigation", handle: 12)
            try await waitUntil { modeStore.phase == .ready }
            modeStore.updateFilters { $0.currency = "USD"; $0.valuation = "market-value" }
            modeStore.selectedNav = .journal
            guard modeStore.request.valuation == "at-cost" else {
                throw StateTestFailure(message: "Transactions request retained an unsupported market valuation")
            }
            modeStore.showAccount("Assets:Stock")
            guard modeStore.request.valuation == "market-value" else { throw StateTestFailure(message: "Account lost the selected analysis valuation") }
            modeStore.navigateBack()
            guard modeStore.request.view == "journal", modeStore.request.valuation == "at-cost" else { throw StateTestFailure(message: "Back restored an invalid market journal request") }
            modeStore.selectedNav = .balanceSheet
            guard modeStore.request.valuation == "market-value", modeStore.request.currency == "USD" else { throw StateTestFailure(message: "Journal navigation erased the analysis preference") }
            modeStore.close()
            try await waitUntil { await modeClient.isClosed(12) }
            print("PASS: account and transaction navigation preserve analysis preferences while normalizing journal requests")

            let holdingClient = ControlledClient()
            let holdingStore = LedgerStore(client: holdingClient, preferences: nil, pollNanoseconds: 60_000_000_000)
            holdingStore.loadLedger(at: "holdings-navigation")
            try await waitUntil { await holdingClient.hasPending("holdings-navigation") }
            await holdingClient.complete("holdings-navigation", handle: 13)
            try await waitUntil { holdingStore.phase == .ready }
            holdingStore.selectedNav = .holdings
            holdingStore.updateFilters { $0.group_by = "commodity"; $0.date_to = "2000-01-06" }
            try await waitUntil { holdingStore.visibleView?.request.view == "holdings" }
            let holdingOrigin = holdingStore.request
            await holdingClient.suspendReads()
            holdingStore.showHoldingLots(groupID: "stock-domain")
            try await waitUntil { await holdingClient.hasRead("holdings", section: "lots") }
            guard holdingStore.request.snapshot_id == holdingStore.state?.snapshot_id, holdingStore.request.group_id == "stock-domain" else { throw StateTestFailure(message: "Lot detail selector must bind to the visible snapshot") }
            holdingStore.showAccount("Assets:Stock")
            try await waitUntil { await holdingClient.hasRead("account") }
            guard holdingStore.request.group_by == nil, holdingStore.request.holding_section == nil, holdingStore.request.group_id == nil, holdingStore.request.event_id == nil, holdingStore.request.lot_kind == nil else { throw StateTestFailure(message: "Holding selectors leaked into account requests") }
            await holdingClient.finishRead("account")
            try await waitUntil { holdingStore.visibleView?.request.view == "account" }
            await holdingClient.finishRead("holdings", section: "lots")
            try await Task.sleep(nanoseconds: 20_000_000)
            guard holdingStore.visibleView?.request.view == "account" else { throw StateTestFailure(message: "Late holding detail replaced the account page") }
            holdingStore.navigateBack()
            try await waitUntil { await holdingClient.hasRead("holdings", section: "lots") }
            await holdingClient.finishRead("holdings", section: "lots")
            try await waitUntil { holdingStore.visibleView != nil }
            holdingStore.navigateBack()
            guard holdingStore.request == holdingOrigin else { throw StateTestFailure(message: "Back failed to restore holdings grouping and cutoff") }
            try await waitUntil { await holdingClient.hasRead("holdings", dateTo: "2000-01-06", groupBy: "commodity") }
            await holdingClient.finishRead("holdings", dateTo: "2000-01-06", groupBy: "commodity")
            try await waitUntil { holdingStore.visibleView != nil }
            holdingStore.showHoldingEventLots(eventID: "source-sale", merged: true)
            try await waitUntil { await holdingClient.hasRead("holdings", section: "lots") }
            holdingStore.updateFilters { $0.date_to = "2000-01-07" }
            guard holdingStore.request.holding_section == "groups", holdingStore.request.group_id == nil, holdingStore.request.event_id == nil, holdingStore.request.lot_kind == nil, holdingStore.request.snapshot_id == nil else { throw StateTestFailure(message: "Changed cutoff retained a stale source selector") }
            try await waitUntil { await holdingClient.hasRead("holdings", dateTo: "2000-01-07") }
            await holdingClient.finishRead("holdings", dateTo: "2000-01-07")
            try await waitUntil { holdingStore.visibleView != nil }
            await holdingClient.finishRead("holdings", dateTo: "2000-01-06", section: "lots")
            try await Task.sleep(nanoseconds: 20_000_000)
            guard holdingStore.visibleView?.request.date_to == "2000-01-07" else { throw StateTestFailure(message: "Late contribution page replaced the new cutoff") }
            holdingStore.selectedNav = .journal
            guard holdingStore.request.group_by == nil, holdingStore.request.holding_section == nil else { throw StateTestFailure(message: "Holding state leaked into another route") }
            holdingStore.close()
            try await waitUntil { await holdingClient.isClosed(13) }
            print("PASS: holding selectors, Back, cutoff changes and late responses preserve coherent navigation")
            let priceClient = ControlledClient()
            let priceStore = LedgerStore(client: priceClient, preferences: nil, pollNanoseconds: 60_000_000_000)
            priceStore.loadLedger(at: "prices")
            try await waitUntil { await priceClient.hasPending("prices") }
            await priceClient.complete("prices", handle: 14)
            try await waitUntil { priceStore.phase == .ready }
            priceStore.showHoldings("Assets:Stock")
            priceStore.updateFilters { $0.currency = "USD"; $0.valuation = "market-value"; $0.date_to = "2000-01-08"; $0.group_by = "commodity" }
            try await waitUntil { priceStore.visibleView != nil }
            let priceHoldingOrigin = priceStore.request
            await priceClient.suspendReads()
            priceStore.showPrices(base: "STK", quote: "USD")
            try await waitUntil { await priceClient.hasRead("prices", priceQuote: "USD") }
            guard priceStore.selectedNav == .prices, priceStore.request.valuation == "at-cost", priceStore.request.currency == "USD", priceStore.request.account.isEmpty, priceStore.request.group_by == nil else { throw StateTestFailure(message: "Price route must be neutral and clear holding parameters") }
            priceStore.updateFilters { $0.price_quote = "EUR" }
            try await waitUntil { await priceClient.hasRead("prices", priceQuote: "EUR") }
            await priceClient.finishRead("prices", priceQuote: "EUR")
            try await waitUntil { priceStore.visibleView?.request.price_quote == "EUR" }
            await priceClient.finishRead("prices", priceQuote: "USD")
            try await Task.sleep(nanoseconds: 20_000_000)
            guard priceStore.visibleView?.request.price_quote == "EUR", priceStore.request.currency == "USD" else { throw StateTestFailure(message: "Late price pair or quote change corrupted the current selection/preferences") }
            priceStore.updateFilters { $0.date_to = "2000-01-06" }
            try await waitUntil { await priceClient.hasRead("prices", dateTo: "2000-01-06") }
            priceStore.updateFilters { $0.date_to = "2000-01-07" }
            try await waitUntil { await priceClient.hasRead("prices", dateTo: "2000-01-07") }
            await priceClient.finishRead("prices", dateTo: "2000-01-07")
            try await waitUntil { priceStore.visibleView?.request.date_to == "2000-01-07" }
            await priceClient.finishRead("prices", dateTo: "2000-01-06")
            try await Task.sleep(nanoseconds: 20_000_000)
            guard priceStore.visibleView?.request.date_to == "2000-01-07" else { throw StateTestFailure(message: "Late price cutoff replaced current quote selection") }
            priceStore.navigateBack()
            guard priceStore.request == priceHoldingOrigin else { throw StateTestFailure(message: "Price Back erased market analysis preference or original holding context") }
            try await waitUntil { await priceClient.hasRead("holdings") }
            await priceClient.finishRead("holdings")
            try await waitUntil { priceStore.visibleView != nil }
            priceStore.showPrices(base: "STK", quote: "USD")
            try await waitUntil { await priceClient.hasRead("prices", priceQuote: "USD") }
            priceStore.showAccount("Assets:Stock")
            guard priceStore.request.price_section == nil, priceStore.request.price_base == nil, priceStore.request.price_quote == nil, priceStore.request.valuation == "market-value" else { throw StateTestFailure(message: "Price parameters leaked into account navigation") }
            try await waitUntil { await priceClient.hasRead("account") }
            await priceClient.finishRead("account")
            try await waitUntil { priceStore.visibleView?.request.view == "account" }
            await priceClient.finishRead("prices", priceQuote: "USD")
            try await Task.sleep(nanoseconds: 20_000_000)
            guard priceStore.visibleView?.request.view == "account" else { throw StateTestFailure(message: "Late prices replaced the account page") }
            priceStore.navigateBack()
            guard priceStore.request.view == "prices", priceStore.request.valuation == "at-cost" else { throw StateTestFailure(message: "Back restored an unsupported market price request") }
            try await waitUntil { await priceClient.hasRead("prices", priceQuote: "USD") }
            priceStore.close()
            await priceClient.finishRead("prices", priceQuote: "USD")
            try await waitUntil { await priceClient.isClosed(14) }
            guard priceStore.phase == .closed, priceStore.snapshot == nil else { throw StateTestFailure(message: "Closing prices accepted an in-flight response") }
            print("PASS: price pairs, Back, analysis preferences, route cleanup and late responses preserve coherent navigation")


        } catch {
            print("FAIL: \(error)")
            exit(1)
        }
    }
}
