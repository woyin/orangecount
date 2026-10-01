// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import Foundation

struct NativeClientFailure: Error { let message: String }

@main
struct NativeClientTests {
    @MainActor static func waitUntil(_ description: String, _ condition: () -> Bool) async throws {
        let deadline = Date().addingTimeInterval(10)
        while Date() < deadline {
            if condition() { return }
            try await Task.sleep(nanoseconds: 10_000_000)
        }
        throw NativeClientFailure(message: "Timed out: \(description)")
    }

    @MainActor static func verifyStoreRecovery(root: String) async throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent("orangecount-store-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        let entry = directory.appendingPathComponent("main.bean")
        let invalid = directory.appendingPathComponent("invalid.bean")
        let original = try Data(contentsOf: URL(fileURLWithPath: root + "/testdata/fixtures/native/cashflow.bean"))
        let syntaxError = try Data(contentsOf: URL(fileURLWithPath: root + "/testdata/fixtures/v3-parity/err-syntax.bean"))
        let balanceError = try Data(contentsOf: URL(fileURLWithPath: root + "/testdata/fixtures/v3-parity/err-unbalanced.bean"))
        try original.write(to: entry, options: .atomic)
        let store = LedgerStore(preferences: nil, pollNanoseconds: 10_000_000)
        defer { store.close() }
        store.start(path: entry.path, restoreRecent: false)
        try await waitUntil("initial real-client load") { store.phase == .ready }
        store.selectedNav = .journal
        store.updateFilters { $0.text = "Food"; $0.date_from = "2026-01-03"; $0.date_to = "2026-01-03" }
        try await waitUntil("filtered transaction") { store.journalPayload?.transactions.count == 1 }
        guard store.journalPayload?.transactions.first?.narration == "Food" else {
            throw NativeClientFailure(message: "Real store did not select the Food transaction")
        }
        let request = store.request
        let snapshotID = store.state?.snapshot_id
        try syntaxError.write(to: entry, options: .atomic)
        try await waitUntil("first error with retained data") { store.phase == .stale && (store.summary?.error_count ?? 0) > 0 }
        let firstCodes = Set(store.summary?.diagnostics?.map(\.code) ?? [])
        guard store.state?.snapshot_id == snapshotID, store.journalPayload?.transactions.first?.narration == "Food" else {
            throw NativeClientFailure(message: "A source error discarded the last valid filtered data")
        }
        try balanceError.write(to: entry, options: .atomic)
        try await waitUntil("distinct second error") {
            store.phase == .stale && Set(store.summary?.diagnostics?.map(\.code) ?? []) != firstCodes
        }
        try original.write(to: entry, options: .atomic)
        try await waitUntil("automatic recovery") { store.phase == .ready && store.summary?.error_count == 0 }
        guard store.request == request, store.journalPayload?.transactions.first?.narration == "Food" else {
            throw NativeClientFailure(message: "Recovery lost the same-ledger filters or data")
        }
        try syntaxError.write(to: invalid, options: .atomic)
        store.loadLedger(at: invalid.path)
        try await waitUntil("opening a different invalid ledger") { store.phase == .invalid }
        guard store.currentPath == invalid.path, store.state?.has_snapshot == false, store.journalPayload == nil else {
            throw NativeClientFailure(message: "A new invalid ledger retained the old ledger's data")
        }
        print("PASS: real store retains filtered data across two errors, recovers, and isolates a new invalid ledger")
    }

    @MainActor static func main() async {
        do {
            let root = CommandLine.arguments[1]
            let client = NativeLedgerClient()
            let opened = try await client.open(path: root + "/testdata/fixtures/native/cashflow.bean")
            let cancelled = Task { try await client.read(handle: opened.handle, request: ViewRequest(view: "dashboard")) }
            cancelled.cancel() // The main-actor task has not reached the client yet.
            var rejected = false
            do { _ = try await cancelled.value }
            catch is CancellationError { rejected = true }
            await client.close(handle: opened.handle)
            guard rejected else { throw NativeClientFailure(message: "A cancelled queued request still entered the C client") }
            print("PASS: cancelled requests stop before C execution")
            try await verifyStoreRecovery(root: root)
            try await verifyHoldingNavigation(root: root)
            try await verifyPriceNavigation(root: root)
        } catch {
            print("FAIL: \(error)")
            exit(1)
        }
    }

    @MainActor static func verifyPriceNavigation(root: String) async throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent("orangecount-price-store-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        for name in ["prices.bean", "prices-included.bean"] {
            try Data(contentsOf: URL(fileURLWithPath: root + "/testdata/fixtures/native/" + name)).write(to: directory.appendingPathComponent(name), options: .atomic)
        }
        let entry = directory.appendingPathComponent("prices.bean")
        let store = LedgerStore(preferences: nil, pollNanoseconds: 60_000_000_000)
        defer { store.close() }
        store.loadLedger(at: entry.path, semantics: "v3")
        try await waitUntil("price session") { store.phase == .ready }
        store.updateFilters { $0.currency = "USD"; $0.valuation = "market-value"; $0.date_to = "2000-01-08" }
        store.selectedNav = .prices
        try await waitUntil("price catalog") { store.pricesPayload?.commodities_page.total_count == 10 }
        guard store.request.valuation == "at-cost", store.request.currency == "USD" else { throw NativeClientFailure(message: "Price catalog overwrote analysis preference") }
        store.showPrices(base: "STK", quote: "USD")
        store.updateFilters { $0.limit = 1 }
        try await waitUntil("first real price page") { store.pricesPayload?.records.first?.rate.exact == "10" }
        guard let page = store.pricesPayload?.records_page else { throw NativeClientFailure(message: "Missing source price page") }
        store.pricePage(2, page: page)
        try await waitUntil("third duplicate source price") { store.pricesPayload?.records.first?.source.line == 18 }
        guard store.pricesPayload?.selected?.rate.exact == "11", store.request.snapshot_id == store.state?.snapshot_id else { throw NativeClientFailure(message: "Price page lost full selection or snapshot identity") }
        let old = store.state?.snapshot_id
        var changed = try Data(contentsOf: entry)
        changed.append(Data("\n2000-01-08 price STK 12 USD\n".utf8))
        try changed.write(to: entry, options: .atomic)
        store.reloadSilently()
        try await waitUntil("new source price selection") { store.state?.snapshot_id != old && store.pricesPayload?.selected?.rate.exact == "12" }
        guard store.request.offset == 0, store.request.snapshot_id == nil, store.request.price_base == "STK" else { throw NativeClientFailure(message: "Price reload failed to clear stale paging or retain meaningful pair") }
        store.updateFilters { $0.price_quote = "EUR" }
        try await waitUntil("real inverse price selection") { store.pricesPayload?.selected?.inverse == true && store.pricesPayload?.selected?.rate.exact == "0.5" }
        store.navigateBack()
        try await waitUntil("catalog restored") { store.pricesPayload?.section == .commodities }
        store.selectedNav = .balanceSheet
        try await waitUntil("analysis restored") { store.visibleView?.request.view == "balance_sheet" }
        guard store.request.valuation == "market-value", store.request.currency == "USD", store.request.price_base == nil, store.request.price_quote == nil else { throw NativeClientFailure(message: "Price page erased market preference or leaked pair") }
        print("PASS: real price catalogue, duplicate pages, inverse, reload and analysis preference restoration")
    }

    @MainActor static func verifyHoldingNavigation(root: String) async throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent("orangecount-holding-store-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        let entry = directory.appendingPathComponent("main.bean")
        let original = try Data(contentsOf: URL(fileURLWithPath: root + "/testdata/fixtures/native/average-history.bean"))
        try original.write(to: entry, options: .atomic)
        let store = LedgerStore(preferences: nil, pollNanoseconds: 60_000_000_000)
        defer { store.close() }
        store.loadLedger(at: entry.path, semantics: "v3")
        try await waitUntil("real holding session") { store.phase == .ready }
        store.showHoldings("Assets:Stock")
        store.updateFilters { $0.date_to = "2000-01-06"; $0.limit = 1 }
        try await waitUntil("real holding projection") { store.holdingsPayload?.groups.first?.average_cost?.exact == "220/9" }
        let origin = store.request
        guard let group = store.holdingsPayload?.groups.first, let event = store.holdingsPayload?.events.first else {
            throw NativeClientFailure(message: "Holding root must expose stock group and reductions")
        }
        store.showHoldingHistory(groupID: group.id)
        try await waitUntil("selected history") { store.holdingsPayload?.section == .history && store.holdingsPayload?.average_history.first?.points.count == 1 }
        guard let pointsPage = store.holdingsPayload?.average_history.first?.points_page else { throw NativeClientFailure(message: "Missing history page") }
        store.holdingsPage(4, page: pointsPage)
        try await waitUntil("exact history point page") { store.holdingsPayload?.average_history.first?.points.first?.average_cost?.exact == "220/9" && store.request.offset == 4 }
        store.setHoldingSort("descending")
        try await waitUntil("sorted detail keeps selector") { store.holdingsPayload?.average_history.first?.points.first?.date == "2000-01-06" }
        guard store.request.group_id == group.id, store.request.snapshot_id == store.state?.snapshot_id else {
            throw NativeClientFailure(message: "Sorting selected history lost its snapshot selector")
        }
        store.navigateBack()
        try await waitUntil("holding origin restored") { store.holdingsPayload?.section == .groups }
        guard store.request == origin else { throw NativeClientFailure(message: "History Back lost original grouping/cutoff/page settings") }
        store.showHoldingEventLots(eventID: event.id, merged: true)
        try await waitUntil("real merge pool detail") { store.holdingsPayload?.events.first?.merge_lots_page.total_count == 2 && store.holdingsPayload?.section == .lots }
        guard let mergePage = store.holdingsPayload?.events.first?.merge_lots_page else { throw NativeClientFailure(message: "Missing merge page") }
        store.holdingsPage(1, page: mergePage)
        try await waitUntil("second merge contribution") { store.holdingsPayload?.events.first?.merge_lots_page.offset == 1 }
        guard store.holdingsPayload?.events.first?.merge_lots.count == 1, store.holdingsPayload?.events.first?.cost_basis?.exact == "500/3" else {
            throw NativeClientFailure(message: "Merge pagination changed consumed cost or exceeded its requested page")
        }
        let oldSnapshot = store.state?.snapshot_id
        var changed = original
        changed.append(Data("\n2000-01-09 price STK 9 USD\n".utf8))
        try changed.write(to: entry, options: .atomic)
        store.reloadSilently()
        try await waitUntil("reload drops stale merge selector") { store.state?.snapshot_id != oldSnapshot && store.holdingsPayload?.section == .groups }
        guard store.request.group_id == nil, store.request.event_id == nil, store.request.lot_kind == nil, store.request.offset == 0 else {
            throw NativeClientFailure(message: "Reload retained a stale merge selector or page")
        }
        store.navigateBack()
        try await waitUntil("Back after real snapshot change") { store.holdingsPayload != nil }
        guard store.request.event_id == nil, store.request.group_id == nil else { throw NativeClientFailure(message: "Back restored an expired holding selector") }
        print("PASS: real holding navigation, history/merge pagination, detail sort and reload retain exact values and clear stale selectors")
    }
}
