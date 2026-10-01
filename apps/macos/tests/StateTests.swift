// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import Foundation

struct StateTestFailure: Error { let message: String }

actor ControlledClient: LedgerSessionClient {
    private var pending: [String: CheckedContinuation<OpenLedgerResult, Error>] = [:]
    private var paths: [UInt64: String] = [:]
    private var closed: Set<UInt64> = []
    private var polls: [UInt64: Int] = [:]
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
        LedgerState(revision: 1, snapshot_id: "snapshot-\(handle)", entry_path: paths[handle]!, valid: true, has_snapshot: true, reloading: false, build_count: 1)
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

    @MainActor static func main() async {
        do {
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
