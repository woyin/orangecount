import AppKit
import SwiftUI

struct DiagnosticsSheetView: View {
    let summary: LedgerSummary
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack {
                Label("账本校验诊断 (Diagnostics)", systemImage: "exclamationmark.octagon.fill")
                    .font(.title2)
                    .bold()
                    .foregroundColor(.red)
                Spacer()
                Button("关闭") {
                    dismiss()
                }
                .keyboardShortcut(.cancelAction)
            }

            Text("当前账本存在 \(summary.error_count) 处严重错误，OrangeCount 严格遵循 Beancount 语义，在错误修复前拒绝不一致的数据流：")
                .font(.subheadline)
                .foregroundColor(.secondary)

            Divider()

            ScrollView {
                VStack(alignment: .leading, spacing: 10) {
                    ForEach(Array(summary.errors.enumerated()), id: \.offset) { idx, err in
                        HStack(alignment: .top, spacing: 12) {
                            Text("\(idx + 1)")
                                .font(.caption)
                                .bold()
                                .foregroundColor(.white)
                                .frame(width: 22, height: 22)
                                .background(Circle().fill(Color.red.opacity(0.85)))

                            VStack(alignment: .leading, spacing: 4) {
                                Text(err)
                                    .font(.system(.body, design: .monospaced))
                                    .foregroundColor(.primary)
                            }
                            Spacer()
                            Button(action: {
                                NSPasteboard.general.clearContents()
                                NSPasteboard.general.setString(err, forType: .string)
                            }) {
                                Image(systemName: "doc.on.doc")
                                    .font(.caption)
                            }
                            .buttonStyle(.plain)
                            .help("复制错误信息")
                        }
                        .padding(10)
                        .background(RoundedRectangle(cornerRadius: 6).fill(Color.red.opacity(0.08)))
                    }
                }
            }
        }
        .padding(24)
        .frame(minWidth: 580, minHeight: 380)
    }
}
