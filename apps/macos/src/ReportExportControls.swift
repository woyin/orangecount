// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.
import AppKit
import SwiftUI
import UniformTypeIdentifiers

struct ReportExportControls: View {
    @ObservedObject var store: LedgerStore
    @Environment(\.ledgerLanguage) private var language
    @StateObject private var exporter = ReportExportCoordinator()
    var body: some View {
        HStack {
            Menu(language.text("导出报表…", "Export report…")) {
                Button(language.text("保存分析文件…", "Save analysis…")) { exporter.save(store: store, language: language, format: "analysis") }
                Button(language.text("保存 CSV…", "Save CSV…")) { exporter.save(store: store, language: language, format: "csv") }
            }.disabled(exporter.saving || store.reportExportIdentity == nil)
            Text(store.reportExportIdentity?.capture_scope == "current-page"
                 ? language.text("导出当前页；CSV不含完整类型与上下文。", "Export current page; CSV omits full types and context.")
                 : language.text("导出完整投影，包含折叠及搜索隐藏记录；CSV不含完整上下文。", "Export the complete projection, including collapsed and search-hidden records; CSV omits full context."))
                .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
            if exporter.saving { ProgressView().controlSize(.small) }
            if let notice = exporter.notice {
                Text(notice).foregroundColor(exporter.failed ? .red : .secondary).textSelection(.enabled)
            }
        }.padding(.horizontal).padding(.vertical, 6).onDisappear { exporter.cancel() }
    }
}

@MainActor
final class ReportExportCoordinator: ObservableObject {
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
                let prepared = try await store.prepareReportExport(format: format)
                try Task.checkCancellation()
                let panel = NSSavePanel()
                panel.allowedContentTypes = format == "csv" ? [.commaSeparatedText] : [UTType(filenameExtension: "ocanalysis") ?? .data]
                panel.nameFieldStringValue = format == "csv" ? "report-result.csv" : "report-result.ocanalysis"
                panel.canCreateDirectories = true
                panel.isExtensionHidden = false
                panel.message = format == "csv"
                    ? language.text("保存已捕获的报表结果；CSV 不包含完整类型和上下文。", "Save the captured report result; CSV omits full types and context.")
                    : language.text("保存已捕获的报表结果及其类型和执行上下文。", "Save the captured report result with types and execution context.")
                let scope = prepared.identity.capture_scope == "current-page"
                    ? language.text("当前页", "Current page") : language.text("完整投影", "Complete projection")
                let statusNames = ["complete":language.text("完整估值", "Complete valuation"), "partial":language.text("部分估值", "Partial valuation"), "unavailable":language.text("估值不可用", "Valuation unavailable"), "empty":language.text("空结果", "Empty result"), "not-applicable":language.text("不适用估值", "Valuation not applicable")]
                panel.message = scope + " · " + language.text("记录数：", "Records: ") + String(prepared.identity.source_record_count) + " · "
                    + language.text("保存固定捕获：", "Save fixed capture: ")
                    + "\(prepared.identity.date_from.isEmpty ? "…" : prepared.identity.date_from) — \(prepared.identity.date_to.isEmpty ? "…" : prepared.identity.date_to)"
                    + " · " + prepared.identity.semantics_profile
                    + language.text("；估值状态：", "; valuation: ") + (statusNames[prepared.identity.completeness] ?? prepared.identity.completeness)
                activePanel = panel
                let response = await withCheckedContinuation { continuation in
                    panel.beginSheetModal(for: window) { continuation.resume(returning: $0) }
                }
                try Task.checkCancellation()
                guard response == .OK, let target = panel.url else {
                    notice = language.text("已取消保存。", "Save cancelled.")
                    return
                }
                try await store.saveReportExport(prepared, to: target)
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
