// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import AppKit
import SwiftUI
import UniformTypeIdentifiers

struct QueryView: View {
    @ObservedObject var store: LedgerStore
    @Environment(\.ledgerLanguage) private var language

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            if let catalogue = store.queryCatalogue, !catalogue.presets.isEmpty {
                HStack {
                    Text(language.text("查询预设", "Query presets")).font(LedgerFonts.font(.caption))
                    ForEach(catalogue.presets) { preset in
                        Button(language.text(preset.name_zh, preset.name_en)) { store.useQueryPreset(id: preset.id) }
                    }
                    if let period = catalogue.period {
                        Menu(language.text("按阅读期间…", "Use reading period…")) {
                            Text("\(period.date_from.isEmpty ? "…" : period.date_from) — \(period.date_to.isEmpty ? "…" : period.date_to)")
                            ForEach(period.presets) { preset in
                                Button(language.text(preset.name_zh, preset.name_en)) { store.useQueryPreset(id: preset.id, inPeriod: true) }
                            }
                        }
                    }
                }
            }
            savedQueries
            Text(language.text("查询完整快照；其他页面的日期、币种与估值筛选不自动应用。", "Queries use the whole snapshot; other pages' date, currency and valuation filters do not apply."))
                .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
            Text(language.text("balance 为按扫描及 WHERE 求值累计的库存；sum(balance) 合并累计库存。GROUP BY 支持别名与从1开始的列号。", "balance is running inventory evaluated in scan/WHERE order; sum(balance) merges those inventories. GROUP BY supports aliases and column ordinals starting at 1."))
                .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
            QueryTextEditor(text: $store.queryDraft,
                errorRange: store.queryResponse?.context.text == store.queryDraft ? store.queryResponse?.error?.range : nil,
                errorRequestID: store.queryResponse?.context.request_id,
                accessibilityLabel: language.text("查询文本", "Query text"))
                .frame(minHeight: 100, maxHeight: 160)
                .border(Color.secondary.opacity(0.4))
                .accessibilityLabel(language.text("查询文本", "Query text"))
            HStack {
                Button(language.text("执行", "Run")) { store.executeQuery() }
                    .keyboardShortcut(.return, modifiers: .command)
                    .disabled(store.state?.has_snapshot != true || store.state?.reloading == true || store.queryDraft.utf8.count > 64 * 1024)
                Button(language.text("取消", "Cancel")) { store.cancelQuery() }
                    .keyboardShortcut(.escape, modifiers: [])
                    .disabled(store.queryPhase != .running && store.queryPhase != .queued)
                if store.queryPhase == .running || store.queryPhase == .queued { ProgressView().controlSize(.small) }
                Text(status).font(LedgerFonts.font(.caption))
                Spacer()
                Text("\(store.queryDraft.utf8.count) / 65536 UTF-8 bytes").font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
            }
            QueryExportControls(store: store)
            if let message = store.queryErrorMessage {
                Text(message).foregroundColor(.red).textSelection(.enabled)
            }
            if let response = store.queryResponse, let execution = response.execution {
                context(response)
                Text(execution.truncated
                     ? language.text("已返回\(execution.returned_rows)行，结果已截断；这不是总匹配数。", "Returned \(execution.returned_rows) rows; truncated, not a total match count.")
                     : language.text("已返回\(execution.returned_rows)行。", "Returned \(execution.returned_rows) rows."))
                    .font(LedgerFonts.font(.caption))
                if execution.returned_rows == 0 {
                    Text(language.text("查询成功，结果为零行；列结构保留如下。", "Query succeeded with zero rows; the column schema remains below."))
                }
                Text(language.text("选择行后按⌘C复制类型化JSON，包含执行上下文与精确值。", "Select rows and press ⌘C to copy typed JSON with execution context and exact values."))
                    .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                NativeQueryTable(response: response, accessibilityLabel: language.text("查询结果", "Query results")).frame(minHeight: 150)
            } else {
                if let previous = store.lastSuccessfulQuery {
                    DisclosureGroup(language.text("上次成功执行（非本次结果）", "Previous successful execution (not this result)")) {
                        context(previous)
                        Text(language.text("返回行数：", "Returned rows: ") + String(previous.execution?.returned_rows ?? 0))
                    }.font(LedgerFonts.font(.caption))
                }
                Spacer()
            }
        }.padding(16)
    }

    @ViewBuilder private var savedQueries: some View {
        if let catalogue = store.queryCatalogue {
            DisclosureGroup(language.text("账本已保存查询（\(catalogue.page.total_count)）", "Saved ledger queries (\(catalogue.page.total_count))")) {
                VStack(alignment: .leading, spacing: 8) {
                    Text(language.text("选择只填入草稿，检查后点击执行。", "Selection fills the draft; review it before running."))
                        .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                    if catalogue.queries.isEmpty { Text(language.text("此页没有已保存查询。", "No saved queries on this page.")) }
                    ScrollView {
                        LazyVStack(alignment: .leading, spacing: 8) {
                            ForEach(catalogue.queries) { saved in
                                HStack(alignment: .top) {
                                    Button(saved.name.isEmpty ? language.text("未命名查询", "Unnamed query") : saved.name) {
                                        store.useSavedQuery(id: saved.id)
                                    }.buttonStyle(.link)
                                    Text(saved.date).font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                                    Spacer()
                                    SourceActions(source: saved.source)
                                }
                            }
                        }
                    }.frame(maxHeight: 130)
                    HStack {
                        Button(language.text("上一页", "Previous")) { store.queryCataloguePage(offset: max(0, catalogue.page.offset - catalogue.page.limit)) }
                            .disabled(catalogue.page.offset == 0)
                        Text("\(catalogue.page.offset + (catalogue.queries.isEmpty ? 0 : 1))–\(catalogue.page.offset + catalogue.queries.count) / \(catalogue.page.total_count)").font(LedgerFonts.font(.caption))
                        Button(language.text("下一页", "Next")) { store.queryCataloguePage(offset: catalogue.page.offset + catalogue.page.limit) }
                            .disabled(!catalogue.page.has_more)
                    }
                }.padding(.vertical, 6)
            }
        }
    }

    private var status: String {
        switch store.queryPhase {
        case .idle: return language.text("尚未执行", "Not executed")
        case .running: return language.text("正在执行", "Running")
        case .queued: return language.text("等待旧请求收尾，将执行最新请求", "Finishing previous work; latest request queued")
        case .succeeded: return language.text("执行成功", "Succeeded")
        case .failed: return language.text("执行失败", "Failed")
        case .cancelled: return language.text("已取消", "Cancelled")
        case .stale: return language.text("结果已过期，请重新执行", "Result is stale; run again")
        }
    }

    @ViewBuilder private func context(_ response: QueryResponse) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(response.context.semantics_profile + " · " + response.context.evaluation_revision + " · " + response.context.snapshot_id)
                .font(LedgerFonts.font(.caption, design: .monospaced)).textSelection(.enabled)
            if response.context.text != store.queryDraft {
                Text(language.text("草稿已修改；以下结果来自此前执行文本。", "Draft changed; the result belongs to the previously executed text."))
                    .foregroundColor(.orange)
            }
            if response.context.snapshot_id != store.state?.snapshot_id || store.queryPhase == .stale {
                Text(language.text("旧执行结果，不能视为当前快照结果。", "Previous execution; this is not the current snapshot result."))
                    .foregroundColor(.orange)
            }
            DisclosureGroup(language.text("查看已执行文本", "Executed text")) {
                Text(response.context.text).font(LedgerFonts.font(.caption, design: .monospaced)).textSelection(.enabled)
            }
        }
    }
}

private struct QueryExportControls: View {
    @ObservedObject var store: LedgerStore
    @Environment(\.ledgerLanguage) private var language
    @StateObject private var exporter = QueryExportCoordinator()

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Button(language.text("保存分析文件…", "Save analysis…")) { exporter.save(store: store, language: language, format: "analysis") }
                Button(language.text("保存 CSV…", "Save CSV…")) { exporter.save(store: store, language: language, format: "csv") }
                if exporter.saving { ProgressView().controlSize(.small) }
                Text(language.text("CSV 不包含完整类型和执行上下文。", "CSV does not include full types or execution context."))
                    .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
            }.disabled(exporter.saving || !store.queryResultIsCurrent)
            if let notice = exporter.notice {
                Text(notice).font(LedgerFonts.font(.caption)).foregroundColor(exporter.failed ? .red : .secondary)
                    .textSelection(.enabled)
            }
        }.onDisappear {
            exporter.cancel()
        }
    }

}

@MainActor
private final class QueryExportCoordinator: ObservableObject {
    @Published var saving = false
    @Published var notice: String?
    @Published var failed = false
    private var saveTask: Task<Void, Never>?
    private var activePanel: NSSavePanel?

    func cancel() {
        saveTask?.cancel()
        activePanel?.cancel(nil)
    }

    func save(store: LedgerStore, language: LedgerLanguage, format: String) {
        guard !saving, let window = NSApp.keyWindow else { return }
        saving = true
        notice = nil
        failed = false
        saveTask = Task { @MainActor in
            defer { saving = false; activePanel = nil; saveTask = nil }
            do {
                let prepared = try await store.prepareQueryExport(format: format)
                try Task.checkCancellation()
                let panel = NSSavePanel()
                panel.allowedContentTypes = format == "csv" ? [.commaSeparatedText] : [UTType(filenameExtension: "ocanalysis") ?? .data]
                panel.nameFieldStringValue = format == "csv" ? "query-result.csv" : "query-result.ocanalysis"
                panel.canCreateDirectories = true
                panel.isExtensionHidden = false
                panel.message = format == "csv"
                    ? language.text("保存已捕获的查询结果；CSV 不包含完整类型和上下文。", "Save the captured query result; CSV omits full types and context.")
                    : language.text("保存已捕获的查询结果及其类型和执行上下文。", "Save the captured query result with types and execution context.")
                activePanel = panel
                let response = await withCheckedContinuation { continuation in
                    panel.beginSheetModal(for: window) { continuation.resume(returning: $0) }
                }
                try Task.checkCancellation()
                guard response == .OK, let target = panel.url else {
                    notice = language.text("已取消保存。", "Save cancelled.")
                    return
                }
                try await store.saveQueryExport(prepared, to: target)
                notice = language.text("已保存捕获结果：", "Saved captured result: ") + target.lastPathComponent
            } catch is CancellationError {
                notice = language.text("已取消保存。", "Save cancelled.")
            } catch AnalysisFileExportError.protectedTarget {
                failed = true
                notice = language.text("不能覆盖账本源文件或声明附件，请选择其他目标。", "Choose another destination; ledger sources and declared attachments cannot be overwritten.")
            } catch AnalysisFileExportError.unsupportedTarget {
                failed = true
                notice = language.text("此目标不支持保存，请选择普通文件。", "Choose a regular file destination.")
            } catch {
                failed = true
                notice = language.text("保存失败：", "Save failed: ") + error.localizedDescription
            }
        }
    }
}

private struct CopiedQueryRows: Encodable, Sendable {
    let contract: String
    let context: QueryContext
    let columns: [QueryColumn]
    let rows: [[QueryCell]]
}

@MainActor
private final class QueryTable: NSTableView {
    var copySelection: (() -> Void)?
    override func keyDown(with event: NSEvent) {
        if event.modifierFlags.intersection(.deviceIndependentFlagsMask) == .command,
           event.charactersIgnoringModifiers?.lowercased() == "c" {
            copySelection?()
        } else { super.keyDown(with: event) }
    }
}

private struct NativeQueryTable: NSViewRepresentable {
    let response: QueryResponse
    let accessibilityLabel: String

    func makeCoordinator() -> Coordinator { Coordinator() }
    func makeNSView(context: Context) -> NSScrollView {
        let scroll = NSScrollView()
        let table = QueryTable()
        table.dataSource = context.coordinator
        table.delegate = context.coordinator
        table.allowsMultipleSelection = true
        table.usesAlternatingRowBackgroundColors = true
        table.columnAutoresizingStyle = .noColumnAutoresizing
        table.setAccessibilityLabel(accessibilityLabel)
        table.copySelection = { [weak table, weak coordinator = context.coordinator] in
            guard let table else { return }
            coordinator?.copy(rows: table.selectedRowIndexes)
        }
        scroll.hasVerticalScroller = true
        scroll.hasHorizontalScroller = true
        scroll.documentView = table
        return scroll
    }
    func updateNSView(_ scroll: NSScrollView, context: Context) {
        guard let table = scroll.documentView as? QueryTable, let execution = response.execution else { return }
        if context.coordinator.response?.context.request_id == response.context.request_id,
           context.coordinator.response?.context.fingerprint == response.context.fingerprint { return }
        context.coordinator.response = response
        context.coordinator.textRows = execution.result.rows.map { $0.map(Self.text) }
        for column in table.tableColumns { table.removeTableColumn(column) }
        for column in execution.result.columns {
            let native = NSTableColumn(identifier: NSUserInterfaceItemIdentifier(column.id))
            native.title = column.name
            native.headerToolTip = column.type + (column.nullable ? " / NULL" : "")
            native.width = 220
            native.minWidth = 80
            table.addTableColumn(native)
        }
        table.deselectAll(nil)
        table.reloadData()
    }

    private static func encoded<T: Encodable>(_ value: T) -> String {
        let encoder = JSONEncoder(); encoder.outputFormatting = [.sortedKeys]
        return (try? encoder.encode(value)).map { String(decoding: $0, as: UTF8.self) } ?? "Encoding error"
    }
    private static func text(_ cell: QueryCell) -> String {
        switch cell.value {
        case .null: return "NULL"
        case .text(let text): return text.isEmpty ? "\"\"" : text
        case .integer(let integer): return integer
        case .boolean(let value): return value ? "true" : "false"
        case .decimal(let value): return value.exact
        case .amount(let value): return value.number.exact + " " + value.currency
        case .position(let value): return encoded(value)
        case .inventory(let value): return encoded(value)
        case .dictionary(let value), .record(let value): return encoded(value)
        case .list(let value): return encoded(value)
        case .stringSet(let value): return encoded(value)
        }
    }

    @MainActor final class Coordinator: NSObject, NSTableViewDataSource, NSTableViewDelegate {
        var response: QueryResponse?
        var textRows: [[String]] = []
        private var copyTask: Task<Void, Never>?
        deinit { copyTask?.cancel() }
        func numberOfRows(in tableView: NSTableView) -> Int { textRows.count }
        func tableView(_ tableView: NSTableView, viewFor tableColumn: NSTableColumn?, row: Int) -> NSView? {
            guard let tableColumn, let column = tableView.tableColumns.firstIndex(of: tableColumn),
                  textRows.indices.contains(row), textRows[row].indices.contains(column) else { return nil }
            let text = textRows[row][column]
            let field = NSTextField(labelWithString: text)
            field.lineBreakMode = .byTruncatingTail
            field.font = NSFont.monospacedSystemFont(ofSize: NSFont.systemFontSize * LedgerFonts.scale, weight: .regular)
            field.toolTip = text
            field.setAccessibilityLabel(tableColumn.title + ": " + text)
            return field
        }
        func copy(rows: IndexSet) {
            guard let response, let result = response.execution?.result, !rows.isEmpty else { return }
            let selected = rows.compactMap { result.rows.indices.contains($0) ? result.rows[$0] : nil }
            let data = CopiedQueryRows(contract: result.contract, context: response.context, columns: result.columns, rows: selected)
            copyTask?.cancel()
            copyTask = Task {
                let bytes = await Task.detached {
                    let encoder = JSONEncoder(); encoder.outputFormatting = [.sortedKeys]
                    return try? encoder.encode(data)
                }.value
                guard !Task.isCancelled, let bytes else { return }
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(String(decoding: bytes, as: UTF8.self), forType: .string)
            }
        }
    }
}

// NSTextView uses UTF-16 selections; protocol offsets are UTF-8 bytes.
private struct QueryTextEditor: NSViewRepresentable {
    @Binding var text: String
    let errorRange: QueryByteRange?
    let errorRequestID: UInt64?
    let accessibilityLabel: String

    func makeCoordinator() -> Coordinator { Coordinator(text: $text) }
    func makeNSView(context: Context) -> NSScrollView {
        let scroll = NSScrollView()
        let editor = NSTextView()
        editor.isRichText = false
        editor.isAutomaticQuoteSubstitutionEnabled = false
        editor.isAutomaticDashSubstitutionEnabled = false
        editor.isAutomaticTextReplacementEnabled = false
        editor.isAutomaticSpellingCorrectionEnabled = false
        editor.isVerticallyResizable = true
        editor.isHorizontallyResizable = false
        editor.autoresizingMask = [.width]
        editor.textContainer?.widthTracksTextView = true
        editor.delegate = context.coordinator
        editor.textContainerInset = NSSize(width: 6, height: 6)
        scroll.hasVerticalScroller = true
        scroll.documentView = editor
        return scroll
    }
    func updateNSView(_ scroll: NSScrollView, context: Context) {
        guard let editor = scroll.documentView as? NSTextView else { return }
        context.coordinator.text = $text
        if editor.string != text { editor.string = text }
        editor.font = NSFont.monospacedSystemFont(ofSize: NSFont.systemFontSize * LedgerFonts.scale, weight: .regular)
        editor.setAccessibilityLabel(accessibilityLabel)
        if let selection = errorRange?.nativeRange(in: text),
           context.coordinator.selectedErrorRequest != errorRequestID {
            editor.setSelectedRange(selection)
            editor.scrollRangeToVisible(selection)
            context.coordinator.selectedErrorRequest = errorRequestID
        } else if errorRange == nil {
            context.coordinator.selectedErrorRequest = nil
        }
    }
    @MainActor final class Coordinator: NSObject, NSTextViewDelegate {
        var text: Binding<String>
        var selectedErrorRequest: UInt64?
        init(text: Binding<String>) { self.text = text }
        func textDidChange(_ notification: Notification) {
            guard let editor = notification.object as? NSTextView else { return }
            text.wrappedValue = editor.string
        }
    }
}
