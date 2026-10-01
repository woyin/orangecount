import AppKit
import SwiftUI

struct DiagnosticsSheetView: View {
    let summary: LedgerSummary
    let retry: () -> Void
    @Environment(\.dismiss) private var dismiss
    @Environment(\.ledgerLanguage) private var language

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack {
                Label(language.text("账本诊断", "Ledger diagnostics"), systemImage: "exclamationmark.triangle").font(LedgerFonts.font(.title2))
                Spacer()
                Button(language.text("重新校验", "Revalidate"), action: retry)
                Button(language.text("关闭", "Close")) { dismiss() }.keyboardShortcut(.cancelAction)
            }
            Text(language.text("\(summary.error_count) 处错误；报表仅使用有效快照。", "\(summary.error_count) errors; reports use valid snapshots only."))
                .foregroundColor(.secondary)
            Divider()
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 16) {
                    ForEach(Array((summary.diagnostics ?? []).enumerated()), id: \.offset) { _, item in
                        VStack(alignment: .leading, spacing: 8) {
                            Label("\(item.code) · " + (item.severity == "error" ? language.text("错误", "Error") : language.text("警告", "Warning")), systemImage: item.severity == "error" ? "xmark.octagon" : "exclamationmark.triangle")
                                .foregroundColor(item.severity == "error" ? .red : .orange).font(LedgerFonts.font(.headline))
                            Text(item.message).textSelection(.enabled)
                            if !item.source.path.isEmpty { SourceActions(source: item.source) }
                            ForEach(Array(item.related.enumerated()), id: \.offset) { _, related in
                                Text(related.message).font(LedgerFonts.font(.caption))
                                if !related.source.path.isEmpty { SourceActions(source: related.source) }
                            }
                            if let guide = item.guide {
                                DisclosureGroup(language.text("解释与修复建议", "Explanation and repair guidance")) {
                                    VStack(alignment: .leading, spacing: 6) {
                                        Text(guide.what).bold()
                                        Text(guide.why)
                                        ForEach(guide.inspect, id: \.self) { Text("• " + $0) }
                                        ForEach(guide.safe_steps, id: \.self) { Text("• " + $0) }
                                        if !guide.example.before.isEmpty { Text(guide.example.before + "\n↓\n" + guide.example.after).font(.body.monospaced()).textSelection(.enabled) }
                                        Text(guide.revalidate).foregroundColor(.secondary)
                                    }.padding(.vertical, 6)
                                }
                            }
                        }.padding().frame(maxWidth: .infinity, alignment: .leading).background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8))
                        // A replacement diagnostic must also replace selectable
                        // text's native accessibility value, not just its pixels.
                        .id([item.code, item.severity, item.message, item.source.path,
                             String(item.source.line), String(item.source.column), language.rawValue])
                    }
                    if summary.diagnostics?.isEmpty != false { Text(language.text("没有诊断信息", "No diagnostics")) }
                }
            }
        }.padding(24).frame(minWidth: 720, minHeight: 440)
    }
}
