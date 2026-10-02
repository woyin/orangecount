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

    @MainActor static func verifyAttachmentRoots() async throws {
        let base = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let ledger = base.appendingPathComponent("ledger")
        let extra = base.appendingPathComponent("receipts")
        try FileManager.default.createDirectory(at: ledger, withIntermediateDirectories: true)
        try FileManager.default.createDirectory(at: extra, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: base) }
        let entry = ledger.appendingPathComponent("main.bean")
        let other = ledger.appendingPathComponent("other.bean")
        let receipt = extra.appendingPathComponent("receipt.txt")
        try "public receipt".write(to: receipt, atomically: true, encoding: .utf8)
        let text = "2024-01-01 open Assets:Cash\n2024-01-02 document Assets:Cash \"" + receipt.path + "\"\n"
        try text.write(to: entry, atomically: true, encoding: .utf8)
        try text.write(to: other, atomically: true, encoding: .utf8)
        let suite = "org.orangecount.tests.roots." + UUID().uuidString
        let defaults = UserDefaults(suiteName: suite)!
        defer { defaults.removePersistentDomain(forName: suite) }
        let first = LedgerStore(preferences: defaults, pollNanoseconds: 10_000_000)
        let second = LedgerStore(preferences: defaults, pollNanoseconds: 10_000_000)
        let unrelated = LedgerStore(preferences: defaults, pollNanoseconds: 10_000_000)
        defer { first.close(); second.close(); unrelated.close() }
        first.start(path: entry.path, restoreRecent: false)
        second.start(path: entry.path, restoreRecent: false)
        unrelated.start(path: other.path, restoreRecent: false)
        try await waitUntil("three directory policies ready") { first.attachmentRoots != nil && second.attachmentRoots != nil && unrelated.attachmentRoots != nil && first.phase == .ready && second.phase == .ready }
        let before = first.state!
        await first.updateAttachmentRoots([extra.path])
        try await waitUntil("same ledger root broadcast") { second.attachmentRoots?.additional_roots.count == 1 }
        guard unrelated.attachmentRoots?.additional_roots.isEmpty == true, first.state?.snapshot_id == before.snapshot_id, first.state?.build_count == before.build_count else { throw NativeClientFailure(message: "Root update changed accounting or another ledger") }
        first.selectedNav = .journal
        first.showDocuments()
        try await waitUntil("granted receipt appears") { first.visibleView?.directives?.records.first?.attachments.first?.status == "available" }
        let prefs = AttachmentRootPreferences(defaults: defaults)
        let restored = try prefs.restore(entry.path)
        guard restored.roots.count == 1, restored.invalid.isEmpty else { throw NativeClientFailure(message: "Bookmark did not restore") }
        await first.updateAttachmentRoots([])
        try await waitUntil("root revocation broadcast") { second.attachmentRoots?.additional_roots.isEmpty == true && first.visibleView?.directives?.records.first?.attachments.first?.status == "outside-root" }
        await first.updateAttachmentRoots([extra.path])
        let old = base.appendingPathComponent("old-receipts")
        try FileManager.default.moveItem(at: extra, to: old)
        try FileManager.default.createDirectory(at: extra, withIntermediateDirectories: true)
        let invalid = try prefs.restore(entry.path)
        guard invalid.roots.isEmpty, invalid.invalid.count == 1 else { throw NativeClientFailure(message: "Replaced directory silently retained authorization") }
        print("PASS: real root grant/revoke, bookmarks, replaced directory, same-ledger window broadcast and other-ledger/accounting isolation")
    }

    @MainActor static func verifyReportSave(root: String) async throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        let entry = directory.appendingPathComponent("main.bean")
        let text = try String(contentsOfFile: root + "/testdata/fixtures/native/cashflow.bean", encoding: .utf8)
        try text.write(to: entry, atomically: true, encoding: .utf8)
        let store = LedgerStore(preferences: nil, pollNanoseconds: 10_000_000)
        defer { store.close() }
        store.start(path: entry.path, restoreRecent: false)
        try await waitUntil("report capture ready") { store.phase == .ready && store.reportExportIdentity != nil }
        let prepared = try await store.prepareReportExport(format: "analysis")
        let first = directory.appendingPathComponent("before.ocanalysis")
        try await store.saveReportExport(prepared, to: first)
        let expected = try Data(contentsOf: first)
        let oldSnapshot = store.state?.snapshot_id
        try (text + "\n; public report reload\n").write(to: entry, atomically: true, encoding: .utf8)
        try await waitUntil("report source reloaded") { store.phase == .ready && store.state?.snapshot_id != oldSnapshot }
        let second = directory.appendingPathComponent("after.ocanalysis")
        try await store.saveReportExport(prepared, to: second)
        guard try Data(contentsOf: second) == expected else { throw NativeClientFailure(message: "Reload changed frozen report bytes") }
        do { try await store.saveReportExport(prepared, to: entry); throw NativeClientFailure(message: "Report overwrote ledger") }
        catch AnalysisFileExportError.protectedTarget { }
        store.close()
        do { try await store.saveReportExport(prepared, to: second); throw NativeClientFailure(message: "Closed store saved late report") }
        catch QueryExportError.changedExecution { }
        print("PASS: real report preparation/save freezes bytes across reload and protects sources/closed windows")
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
            try await verifyQueryClient(root: root)
            try await verifyQueryStore(root: root)
            try await verifySavedQueries(root: root)
            try await verifyQueryPresets(root: root)
            try await verifyDirectiveNavigation(root: root)
            try await verifyAttachmentRoots()
            try await verifyReportSave(root: root)
            try await verifyStoreRecovery(root: root)
            try await verifyHoldingNavigation(root: root)
            try await verifyPriceNavigation(root: root)
        } catch {
            print("FAIL: \(error)")
            exit(1)
        }
    }

    @MainActor static func verifySavedQueries(root: String) async throws {
        let store = LedgerStore(preferences: nil, pollNanoseconds: 60_000_000_000)
        defer { store.close() }
        store.loadLedger(at: root + "/testdata/fixtures/native/queries.bean", semantics: "v3")
        try await waitUntil("saved query ledger") { store.phase == .ready }
        store.selectedNav = .query
        store.updateFilters { $0.limit = 1 }
        try await waitUntil("saved query catalogue") { store.queryCatalogue?.page.limit == 1 }
        let included = store.queryCatalogue!.queries[0]
        guard store.useSavedQuery(id: included.id), store.queryDraft == "SELECT unknown(number) LIMIT 0",
              store.queryPhase == .idle, store.queryResponse == nil else {
            throw NativeClientFailure(message: "Selecting saved query executed it or discarded invalid text")
        }
        store.queryCataloguePage(offset: 1)
        try await waitUntil("first duplicate query") { store.queryCatalogue?.page.offset == 1 }
        let first = store.queryCatalogue!.queries[0]
        guard store.useSavedQuery(id: first.id), first.name == "同名", !store.useSavedQuery(id: included.id) else {
            throw NativeClientFailure(message: "Saved query identity crossed catalogue page")
        }
        store.executeQuery()
        try await waitUntil("saved query explicit execution") { store.queryPhase == .succeeded }
        guard store.queryResponse?.context.text == "SELECT account LIMIT 0", store.queryResponse?.execution?.returned_rows == 0 else {
            throw NativeClientFailure(message: "Saved query did not execute its exact draft")
        }
        store.queryCataloguePage(offset: 2)
        try await waitUntil("second duplicate query") { store.queryCatalogue?.page.offset == 2 }
        let second = store.queryCatalogue!.queries[0]
        guard second.name == first.name, second.id != first.id, store.useSavedQuery(id: second.id),
              store.queryDraft == "SELECT number LIMIT 1", store.queryDraftChanged,
              store.queryResponse?.context.text == "SELECT account LIMIT 0" else {
            throw NativeClientFailure(message: "Duplicate name selected wrong text or relabelled prior execution")
        }
        print("PASS: saved query sources, snapshot paging, duplicate names, draft-only selection and explicit execution")
    }

    @MainActor static func verifyQueryPresets(root: String) async throws {
        let store = LedgerStore(preferences: nil, pollNanoseconds: 60_000_000_000)
        defer { store.close() }
        store.loadLedger(at: root + "/testdata/fixtures/query/presets.bean", semantics: "v3")
        try await waitUntil("preset ledger") { store.phase == .ready }
        store.selectedNav = .query
        try await waitUntil("preset catalogue") { store.queryCatalogue?.presets.count == 4 }
        let presets = store.queryCatalogue!.presets
        guard !store.useQueryPreset(id: "unknown"), store.useQueryPreset(id: presets[0].id),
              store.queryDraft == presets[0].text, store.queryPhase == .idle, store.queryResponse == nil else {
            throw NativeClientFailure(message: "Preset selection executed or changed text")
        }
        store.executeQuery()
        try await waitUntil("preset explicit execution") { store.queryPhase == .succeeded }
        let executedID = store.queryResponse!.context.request_id
        guard store.queryResponse?.execution?.returned_rows == 2,
              store.useQueryPreset(id: presets[1].id), store.queryDraft == presets[1].text,
              store.queryResponse?.context.request_id == executedID,
              store.queryResponse?.context.text == presets[0].text, store.queryDraftChanged else {
            throw NativeClientFailure(message: "Preset changed captured execution")
        }
        store.executeQuery()
        try await waitUntil("expense preset explicit execution") { store.queryPhase == .succeeded && store.queryResponse?.context.text == presets[1].text }
        guard store.queryResponse?.execution?.returned_rows == 2 else { throw NativeClientFailure(message: "Expense months lost") }
        store.updateFilters { $0.window = "1m"; $0.date_from = ""; $0.date_to = "" }
        try await waitUntil("resolved preset period") { store.queryCatalogue?.period?.date_from == "2000-02-01" }
        let previous = store.queryResponse!.context.request_id
        guard store.useQueryPreset(id: presets[0].id, inPeriod: true),
              store.queryDraft.contains("date >= 2000-02-01 AND date <= 2000-02-21"),
              store.queryResponse?.context.request_id == previous else {
            throw NativeClientFailure(message: "Period selection executed or hid date bounds")
        }
        store.executeQuery()
        try await waitUntil("period preset execution") { store.queryPhase == .succeeded && store.queryResponse?.context.text == store.queryDraft }
        guard let periodExecution = store.queryResponse?.execution, periodExecution.returned_rows == 1,
              case .integer(let month) = periodExecution.result.rows[0][1].value, month == "2" else {
            throw NativeClientFailure(message: "Period preset did not filter to February")
        }
        print("PASS: native preset catalogue, draft-only selection, explicit monthly and captured period execution")
    }

    @MainActor static func verifyDirectiveNavigation(root: String) async throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent("orangecount-directives-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        for name in ["all.bean", "included.bean"] {
            try Data(contentsOf: URL(fileURLWithPath: root + "/testdata/fixtures/directives/" + name)).write(to: directory.appendingPathComponent(name))
        }
        let entry = directory.appendingPathComponent("all.bean")
        let store = LedgerStore(preferences: nil, pollNanoseconds: 60_000_000_000)
        defer { store.close() }
        store.loadLedger(at: entry.path, semantics: "v3")
        try await waitUntil("directive ledger") { store.phase == .ready }
        store.selectedNav = .journal
        store.updateFilters { $0.limit = 1 }
        try await waitUntil("transaction mode") { store.journalPayload?.total_count == 3 }
        store.selectJournalMode("all")
        try await waitUntil("all directive page") { store.directivesPayload?.page.total_count == 18 }
        store.updateFilters { $0.directive_kind = "note" }
        try await waitUntil("duplicate notes") { store.directivesPayload?.page.total_count == 2 }
        let firstID = store.directivesPayload!.records[0].id
        store.journalPage(1)
        try await waitUntil("second duplicate note") { store.directivesPayload?.page.offset == 1 }
        let secondID = store.directivesPayload!.records[0].id
        guard firstID != secondID else { throw NativeClientFailure(message: "Duplicate directive identities collapsed") }
        store.showAccount("Assets:Cash")
        try await waitUntil("account from directive mode") { store.accountDetail != nil }
        guard store.request.journal_mode == nil && store.request.directive_kind == nil else { throw NativeClientFailure(message: "Directive parameters leaked into account route") }
        store.navigateBack()
        try await waitUntil("directive Back") { store.directivesPayload?.page.offset == 1 }
        let oldSnapshot = store.state!.snapshot_id
        var changed = Data("; unrelated inserted line\n".utf8)
        changed.append(try Data(contentsOf: entry)); try changed.write(to: entry, options: .atomic)
        store.reloadSilently()
        try await waitUntil("directive reload") { store.state?.snapshot_id != oldSnapshot && store.directivesPayload?.page.offset == 0 }
        guard store.request.directive_kind == "note", store.directivesPayload?.records[0].id != firstID else { throw NativeClientFailure(message: "Reload restored duplicate by ordinal or lost kind filter") }
        store.selectJournalMode("config")
        try await waitUntil("configuration mode") { store.directivesPayload?.mode == "config" && store.directivesPayload?.page.total_count == 9 }
        guard store.request.directive_kind == nil, store.request.offset == 0 else { throw NativeClientFailure(message: "Mode switch retained stale kind or paging") }
        store.selectedNav = .query
        try await waitUntil("query after config") { store.queryCatalogue != nil }
        guard store.request.journal_mode == nil && store.request.directive_kind == nil else { throw NativeClientFailure(message: "Configuration parameters leaked into query") }
        print("PASS: native directive modes, duplicate pages, account Back, reload and route isolation")
    }

    @MainActor static func verifyQueryStore(root: String) async throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent("orangecount-query-store-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        let entry = directory.appendingPathComponent("main.bean")
        var input = try Data(contentsOf: URL(fileURLWithPath: root + "/testdata/fixtures/query/default-source.bean"))
        try input.write(to: entry, options: .atomic)
        let store = LedgerStore(preferences: nil, pollNanoseconds: 60_000_000_000)
        defer { store.close() }
        store.loadLedger(at: entry.path, semantics: "v3")
        try await waitUntil("real query store session") { store.phase == .ready }
        store.updateFilters { $0.currency = "USD"; $0.valuation = "market-value" }
        try await waitUntil("query origin analysis") { store.visibleView?.request.valuation == "market-value" }
        store.selectedNav = .query
        try await waitUntil("real query workbench route") { store.visibleView?.request.view == "query" }
        guard store.queryPhase == .idle, store.request.valuation == "at-cost" else {
            throw NativeClientFailure(message: "Workbench executed draft or retained a report-only valuation")
        }
        store.queryDraft = "SELECT number LIMIT 0"
        store.executeQuery()
        try await waitUntil("real zero-row query success") { store.queryPhase == .succeeded }
        guard store.queryResultIsCurrent, store.queryResponse?.execution?.returned_rows == 0,
              store.queryResponse?.execution?.result.columns.count == 1,
              store.queryResponse?.context.semantics_profile == "v3" else {
            throw NativeClientFailure(message: "Query Store lost zero-row schema or semantics")
        }
        store.queryDraft = "SELECT account LIMIT 1"
        let frozenCSV = try await store.captureQueryExport(format: "csv")
        guard String(data: frozenCSV.content, encoding: .utf8) == "number\n",
              frozenCSV.context.text == "SELECT number LIMIT 0" else {
            throw NativeClientFailure(message: "Export must use the successful execution, not the changed draft")
        }
        let frozenAnalysis = try await store.captureQueryExport(format: "analysis")
        guard frozenAnalysis.content.prefix(4) == Data([0x50,0x4b,0x03,0x04]) else {
            throw NativeClientFailure(message: "Async analysis export must preserve actual Go ZIP bytes")
        }
        let prepared = try await store.prepareQueryExport(format: "csv")
        store.queryDraft = "SELECT invalid_function(number) LIMIT 0"
        store.executeQuery()
        try await waitUntil("real query compile error") { store.queryPhase == .failed }
        do {
            _ = try await store.captureQueryExport(format: "csv")
            throw NativeClientFailure(message: "Store exported previous success as a failed execution")
        } catch is QueryExportError { }
        guard String(data: frozenCSV.content, encoding: .utf8) == "number\n" else {
            throw NativeClientFailure(message: "Later execution changed frozen export bytes")
        }
        guard !store.queryResultIsCurrent, store.queryResponse?.execution == nil,
              store.queryResponse?.error?.code == "unknown-function",
              store.lastSuccessfulQuery?.context.request_id == 1 else {
            throw NativeClientFailure(message: "Query Store mixed failure with old zero-row success")
        }
        var expression = "number"
        for _ in 0..<18 { expression = "(" + expression + " + 0.0)" }
        store.queryDraft = "SELECT str(" + expression + ") LIMIT 1"
        store.executeQuery()
        try await Task.sleep(nanoseconds: 20_000_000)
        store.cancelQuery()
        store.queryDraft = "SELECT account LIMIT 1"
        store.executeQuery()
        try await waitUntil("real query after cancellation") { store.queryPhase == .succeeded }
        guard store.queryResponse?.context.request_id == 4, store.queryResponse?.execution?.returned_rows == 1 else {
            throw NativeClientFailure(message: "Query Store published cancelled work over latest success")
        }
        let oldID = store.state?.snapshot_id
        input.append(Data("\n2000-01-03 price USD 1 EUR\n".utf8))
        try input.write(to: entry, options: .atomic)
        store.reloadSilently()
        try await waitUntil("real query snapshot change") { store.state?.snapshot_id != oldID && store.phase == .ready }
        let refreshedProtection = try await store.refreshExportProtection()
        guard refreshedProtection.protected_paths.contains(where: { $0.hasSuffix("main.bean") }),
              refreshedProtection.snapshot_id == store.state?.snapshot_id,
              String(data: frozenCSV.content, encoding: .utf8) == "number\n" else {
            throw NativeClientFailure(message: "Protection refresh must follow reload without replacing captured query bytes")
        }
        let exportTarget = directory.appendingPathComponent("captured.csv")
        try await store.saveQueryExport(prepared, to: exportTarget)
        guard try Data(contentsOf: exportTarget) == Data("number\n".utf8) else {
            throw NativeClientFailure(message: "Saving after reload replaced fixed analysis content")
        }
        let cancelledTarget = directory.appendingPathComponent("cancelled.csv")
        let cancelledSave = Task { @MainActor in try await store.saveQueryExport(prepared, to: cancelledTarget) }
        cancelledSave.cancel()
        do { try await cancelledSave.value; throw NativeClientFailure(message: "Cancelled save succeeded") }
        catch is CancellationError { }
        guard !FileManager.default.fileExists(atPath: cancelledTarget.path) else {
            throw NativeClientFailure(message: "Cancelled save created a target")
        }
        do {
            try await store.saveQueryExport(prepared, to: entry)
            throw NativeClientFailure(message: "Store saved an analysis over its source ledger")
        } catch AnalysisFileExportError.protectedTarget { }
        guard store.queryPhase == .stale, !store.queryResultIsCurrent,
              store.queryResponse?.context.snapshot_id == oldID else {
            throw NativeClientFailure(message: "Query Store relabelled previous result after real reload")
        }
        let validSnapshotID = store.state?.snapshot_id
        let newAttachment = directory.appendingPathComponent("new-attachment.csv")
        let attachmentContent = Data("public attachment".utf8)
        try attachmentContent.write(to: newAttachment)
        input.append(Data("\n2000-01-04 document Assets:Cash \"new-attachment.csv\"\nnot a valid directive\n".utf8))
        try input.write(to: entry, options: .atomic)
        store.reloadSilently()
        try await waitUntil("invalid reload with new attachment") { store.state?.valid == false && store.state?.reloading == false && store.phase == .stale }
        let failedProtection = try await store.refreshExportProtection()
        guard failedProtection.protected_paths.contains(where: { $0.hasSuffix("new-attachment.csv") }),
              store.state?.snapshot_id == validSnapshotID else {
            throw NativeClientFailure(message: "Failed input lost new attachment protection or replaced financial snapshot")
        }
        do {
            try await store.saveQueryExport(prepared, to: newAttachment)
            throw NativeClientFailure(message: "Export overwrote attachment declared in failed input")
        } catch AnalysisFileExportError.protectedTarget { }
        guard try Data(contentsOf: newAttachment) == attachmentContent else {
            throw NativeClientFailure(message: "Failed-input attachment content changed")
        }
        store.selectedNav = .balanceSheet
        try await waitUntil("analysis after workbench") { store.visibleView?.request.view == "balance_sheet" }
        guard store.request.valuation == "market-value", store.request.currency == "USD" else {
            throw NativeClientFailure(message: "Workbench overwrote analysis valuation preference")
        }
        print("PASS: real query route, zero rows, error, cancelled/latest work, stale snapshot and analysis preferences")
        print("PASS: failed reload protects newly declared attachment while retaining valid financial snapshot and captured export")
    }

    @MainActor static func verifyQueryClient(root: String) async throws {
        let client = NativeLedgerClient()
        let opened = try await client.open(path: root + "/testdata/fixtures/query/default-source.bean", semantics: "v3")
        let snapshotID = opened.state.snapshot_id
        do {
            let request = QueryRequest(snapshot_id: snapshotID, request_id: 1, text: "SELECT 1 AS v, 2 AS v LIMIT 1")
            let success = try await client.executeQuery(handle: opened.handle, request: request)
            guard success.execution?.result.rows.count == 1, success.error == nil else {
                throw NativeClientFailure(message: "Async query lost its typed result")
            }
            try await client.cancelQuery(handle: opened.handle, requestID: 2)
            let queued = try await client.executeQuery(handle: opened.handle, request: QueryRequest(snapshot_id: snapshotID, request_id: 2, text: "SELECT number"))
            guard queued.error?.code == "cancelled", queued.execution == nil else {
                throw NativeClientFailure(message: "Async control cancellation did not reach the session")
            }
            let next = try await client.executeQuery(handle: opened.handle, request: QueryRequest(snapshot_id: snapshotID, request_id: 3, text: "SELECT account LIMIT 0"))
            guard next.execution?.returned_rows == 0, next.error == nil else {
                throw NativeClientFailure(message: "Cancelled generation poisoned the next query")
            }
            var expression = "number"
            for _ in 0..<18 { expression = "(" + expression + " + 0.0)" }
            let heavyText = "SELECT str(" + expression + ") LIMIT 1"
            let baseline = try await client.executeQuery(handle: opened.handle, request: QueryRequest(snapshot_id: snapshotID, request_id: 4, text: heavyText))
            guard baseline.error?.code == "work-budget-exceeded" else {
                throw NativeClientFailure(message: "Cancellation test does not exercise bounded expensive evaluation")
            }
            let executing = Task {
                try await client.executeQuery(handle: opened.handle, request: QueryRequest(snapshot_id: snapshotID, request_id: 5, text: heavyText))
            }
            try await Task.sleep(nanoseconds: 20_000_000)
            try await client.cancelQuery(handle: opened.handle, requestID: 5)
            let stopped = try await executing.value
            guard stopped.error?.code == "cancelled", stopped.execution == nil else {
                throw NativeClientFailure(message: "Control cancellation queued behind expensive C execution")
            }
        } catch {
            await client.close(handle: opened.handle)
            throw error
        }
        await client.close(handle: opened.handle)
        print("PASS: async native query execution, cancellation control and subsequent generation")
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
