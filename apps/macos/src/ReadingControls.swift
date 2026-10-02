// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import AppKit
import SwiftUI

// QA-only font scaling exercises real point sizes without changing macOS or
// normal application preferences. At scale 1 the native semantic font stays.
enum LedgerFonts {
    static let scale: Double = {
        guard Bundle.main.object(forInfoDictionaryKey: "OrangeCountTestMode") as? Bool == true,
              let value = Bundle.main.object(forInfoDictionaryKey: "OrangeCountTestFontScale") as? NSNumber,
              [1.0, 1.25, 1.5].contains(value.doubleValue) else { return 1.0 }
        return value.doubleValue
    }()
    static func font(_ style: Font.TextStyle, design: Font.Design = .default) -> Font {
        guard scale != 1 else { return .system(style, design: design) }
        let native: NSFont.TextStyle
        switch style {
        case .largeTitle: native = .largeTitle
        case .title: native = .title1
        case .title2: native = .title2
        case .title3: native = .title3
        case .headline: native = .headline
        case .subheadline: native = .subheadline
        case .callout: native = .callout
        case .caption: native = .caption1
        case .caption2: native = .caption2
        case .footnote: native = .footnote
        default: native = .body
        }
        let points = NSFont.preferredFont(forTextStyle: native, options: [:]).pointSize
        return .system(size: points * scale, design: design)
    }
}

enum LedgerLanguage: String {
    case chinese = "zh-CN", english = "en"
    func text(_ chinese: String, _ english: String) -> String { self == .chinese ? chinese : english }
}

private struct LedgerLanguageKey: EnvironmentKey {
    static let defaultValue = LedgerLanguage.chinese
}
extension EnvironmentValues {
    var ledgerLanguage: LedgerLanguage {
        get { self[LedgerLanguageKey.self] }
        set { self[LedgerLanguageKey.self] = newValue }
    }
}

struct SourceActions: View {
    let source: SourceLocation
    @Environment(\.ledgerLanguage) private var language
    private var position: String { "\(source.path):\(source.line):\(source.column)" }
    var body: some View {
        HStack(spacing: 12) {
            Text("\((source.path as NSString).lastPathComponent):\(source.line)")
                .font(.caption.monospaced()).textSelection(.enabled).help(position)
                .id(position)
            Button(language.text("打开源文件", "Open source"), action: openSource)
                .help(language.text("打开文件；行号显示在左侧", "Opens the file; line number is shown on the left"))
            Button(language.text("在 Finder 中显示", "Show in Finder")) {
                NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: source.path)])
            }
            Button(language.text("复制位置", "Copy location")) {
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(position, forType: .string)
            }
        }
        .controlSize(.small)
    }

    private func openSource() {
        let workspace = NSWorkspace.shared
        let url = URL(fileURLWithPath: source.path)
        let preferred = workspace.urlForApplication(toOpen: url)
        let identifier = preferred.flatMap { Bundle(url: $0)?.bundleIdentifier } ?? ""
        // Finder may associate .bean with this viewer. Source navigation must
        // open an editor instead of creating another OrangeCount window.
        if preferred == nil || identifier == "org.orangecount.desktop" || identifier.hasPrefix("org.orangecount.desktop.") {
            if let editor = workspace.urlForApplication(withBundleIdentifier: "com.apple.TextEdit") {
                workspace.open([url], withApplicationAt: editor, configuration: NSWorkspace.OpenConfiguration())
            } else {
                workspace.activateFileViewerSelecting([url])
            }
        } else {
            workspace.open(url)
        }
    }
}

final class DateInputState: ObservableObject {
    @Published var from = ""
    @Published var to = ""
}

struct ReadingControls: View {
    @ObservedObject var store: LedgerStore
    @Environment(\.ledgerLanguage) private var language
    @StateObject private var dates = DateInputState()

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                if store.selectedNav != .balanceSheet {
                    Picker(language.text("期间", "Period"), selection: Binding(get: { store.request.window }, set: { value in
                        dates.from = ""; dates.to = ""
                        store.updateFilters { $0.window = value; $0.date_from = ""; $0.date_to = "" }
                    })) {
                        Text(language.text("最近一月", "Latest month")).tag("1m")
                        Text("1Y").tag("1y")
                        Text(language.text("全部", "All")).tag("all")
                    }.frame(maxWidth: 190)
                    TextField(language.text("起始 YYYY-MM-DD", "From YYYY-MM-DD"), text: $dates.from).frame(width: 150).onSubmit(apply)
                }
                TextField(language.text("截至 YYYY-MM-DD", "Through YYYY-MM-DD"), text: $dates.to).frame(width: 165).onSubmit(apply)
                Button(language.text("应用日期", "Apply dates"), action: apply)
                Spacer(minLength: 6)
            }.disabled(isConfiguration)
            HStack {
                Picker(language.text("会计语义", "Semantics"), selection: Binding(get: { store.semantics }, set: { store.switchSemantics(to: $0) })) {
                    Text("legacy").tag("legacy")
                    Text("v3").tag("v3")
                }.frame(maxWidth: 150)
                Text(store.state?.evaluation_revision ?? "").font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                if store.selectedNav != .journal && store.selectedNav != .prices {
                    Picker(language.text("口径", "Valuation"), selection: Binding(get: { store.request.valuation }, set: { value in store.updateFilters { $0.valuation = value } })) {
                        Text(language.text("成本", "At cost")).tag("at-cost")
                        Text(language.text("数量", "Units")).tag("units")
                        Text(language.text("市值", "Market value")).tag("market-value")
                            .disabled(store.request.currency.isEmpty || store.request.currency == "native")
                    }.frame(maxWidth: 155)
                    Picker(language.text("币种", "Currency"), selection: Binding(get: { store.request.currency }, set: { value in store.updateFilters { $0.currency = value } })) {
                        Text(defaultCurrencyLabel).tag("").disabled(store.request.valuation == "market-value")
                        if store.request.currency == "native" {
                            Text(language.text("各原币种", "Native currencies")).tag("native").disabled(store.request.valuation == "market-value")
                        }
                        ForEach(store.summary?.currencies ?? [], id: \.self) { Text($0).tag($0) }
                    }.frame(maxWidth: 150)
                }
                Spacer(minLength: 6)
                Text(scope).font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
            }
        }
        .textFieldStyle(.roundedBorder)
        .padding(.horizontal, 16).padding(.vertical, 10)
        .onAppear { dates.from = store.request.date_from; dates.to = store.request.date_to }
        .onChange(of: store.currentPath) { _ in dates.from = ""; dates.to = "" }
        .onChange(of: store.request.date_from) { dates.from = $0 }
        .onChange(of: store.request.date_to) { dates.to = $0 }
        .sheet(item: $store.migrationPreview) { preview in
            SemanticsPreviewSheet(preview: preview, apply: store.applySemanticsPreview, cancel: { store.migrationPreview = nil }).environment(\.ledgerLanguage, language)
        }
    }

    private var defaultCurrencyLabel: String {
        language.text("各原币种（默认）", "Native currencies (default)")
    }

    private var isConfiguration: Bool { store.selectedNav == .journal && store.request.journal_mode == "config" }

    private func apply() { store.updateFilters { $0.date_from = store.selectedNav == .balanceSheet ? "" : dates.from; $0.date_to = dates.to; $0.window = "all" } }
    private var scope: String {
        if isConfiguration { return language.text("配置指令无日期，按加载顺序显示", "Configuration directives are undated; shown in load order") }
        let start = store.treeReport?.date_from ?? store.journalPayload?.date_from ?? store.directivesPayload?.date_from ?? store.trendsPayload?.date_from ?? store.holdingsPayload?.date_from ?? store.pricesPayload?.date_from ?? ""
        let end = store.treeReport?.date_to ?? store.journalPayload?.date_to ?? store.directivesPayload?.date_to ?? store.trendsPayload?.date_to ?? store.holdingsPayload?.date_to ?? store.pricesPayload?.date_to ?? ""
        if store.selectedNav == .balanceSheet { return language.text("截至日余额：", "Balances through: ") + (end.isEmpty ? "—" : end) }
        if store.selectedNav == .prices { return language.text("源历史期间：", "Source history: ") + (start.isEmpty ? language.text("最早记录", "First record") : start) + " → " + (end.isEmpty ? "—" : end) }
        return language.text("发生期间（含首尾日）：", "Inclusive activity range: ") + (start.isEmpty ? language.text("最早记录", "First record") : start) + " → " + (end.isEmpty ? "—" : end)
    }
}

struct SemanticsPreviewSheet: View {
    let preview: SemanticsPreview
    let apply: () -> Void
    let cancel: () -> Void
    @Environment(\.ledgerLanguage) private var language
    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text(language.text("切换会计语义前的比较", "Accounting semantics comparison")).font(LedgerFonts.font(.title2))
            Text(language.text("两种配置使用同一份账本输入。切换将重新打开会话。", "Both profiles use the same ledger input. Switching reopens the session."))
            HStack(spacing: 30) {
                profile(preview.comparison.legacy)
                profile(preview.comparison.v3)
            }
            ForEach(preview.comparison.change_counts.keys.sorted(), id: \.self) { dimension in
                HStack {
                    Text(dimensionLabel(dimension))
                    Spacer()
                    Text("\(preview.comparison.change_counts[dimension] ?? 0)")
                }
            }
            if !preview.comparison.legacy.valid || !preview.comparison.v3.valid {
                Text(language.text("无效配置的库存和分录是诊断用的部分结果。请先查看错误。", "Invalid profiles expose partial diagnostic state. Review the errors first.")).foregroundColor(.orange)
            }
            if let details = preview.comparison.details, !details.isEmpty {
                DisclosureGroup(language.text("本地详细变化", "Local change details")) {
                    ScrollView {
                        LazyVStack(alignment: .leading, spacing: 12) {
                            ForEach(details) { change in
                                let presentation = change.presentation(locale: language.rawValue)
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(dimensionLabel(change.dimension)).font(LedgerFonts.font(.headline))
                                    Text(presentation.key).font(.caption.monospaced()).textSelection(.enabled)
                                    Text("legacy: " + presentation.before).textSelection(.enabled)
                                    Text("v3: " + presentation.after).textSelection(.enabled)
                                    if let detail = presentation.detail, !detail.isEmpty { Text(detail).font(.caption.monospaced()).textSelection(.enabled) }
                                }
                            }
                        }
                    }.frame(maxHeight: 240)
                }
            }
            HStack {
                Button(language.text("取消", "Cancel"), action: cancel).keyboardShortcut(.cancelAction)
                Spacer()
                Button(language.text("使用 ", "Use ") + preview.profile, action: apply).keyboardShortcut(.defaultAction)
            }
        }.padding(24).frame(minWidth: 460)
    }
    private func profile(_ value: ProfileSummary) -> some View {
        VStack(alignment: .leading) {
            Text(value.semantics_profile).font(LedgerFonts.font(.headline))
            Text(value.valid ? language.text("有效", "Valid") : language.text("无效", "Invalid"))
            Text(language.text("错误：", "Errors: ") + "\(value.error_count)")
            if !value.active_extensions.isEmpty { Text(value.active_extensions.joined(separator: ", ")) }
        }
    }
    private func dimensionLabel(_ value: String) -> String {
        switch value {
        case "accounts": return language.text("账户规则变化", "Account rule changes")
        case "balances": return language.text("余额变化", "Balance changes")
        case "lots": return language.text("成本批次变化", "Cost lot changes")
        case "entries": return language.text("分录变化", "Entry changes")
        case "diagnostics": return language.text("诊断变化", "Diagnostic changes")
        case "validity": return language.text("有效性变化", "Validity changes")
        default: return value
        }
    }
}
