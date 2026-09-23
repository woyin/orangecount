import SwiftUI

final class JournalViewState: ObservableObject {
    @Published var searchText: String = ""
    @Published var expandedTxIDs: Set<String> = []

    func isExpanded(_ id: String) -> Bool {
        return expandedTxIDs.contains(id)
    }

    func toggleExpanded(_ id: String) {
        if expandedTxIDs.contains(id) {
            expandedTxIDs.remove(id)
        } else {
            expandedTxIDs.insert(id)
        }
    }
}

struct JournalView: View {
    let transactions: [JournalTransaction]
    @StateObject private var viewState = JournalViewState()

    var filteredTransactions: [JournalTransaction] {
        if viewState.searchText.isEmpty {
            return transactions
        }
        let q = viewState.searchText.lowercased()
        return transactions.filter { tx in
            tx.narration.lowercased().contains(q) ||
            tx.payee.lowercased().contains(q) ||
            tx.date.contains(q) ||
            tx.tags.contains(where: { $0.lowercased().contains(q) }) ||
            tx.links.contains(where: { $0.lowercased().contains(q) }) ||
            tx.postings.contains(where: { $0.account.lowercased().contains(q) })
        }
    }

    var body: some View {
        VStack(spacing: 0) {
            // Filter Bar
            HStack {
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("搜索交易 (收款人 / 摘要 / 账户 / #标签 / 日期)...", text: $viewState.searchText)
                        .textFieldStyle(.plain)
                    if !viewState.searchText.isEmpty {
                        Button(action: { viewState.searchText = "" }) {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(.secondary)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(8)
                .background(RoundedRectangle(cornerRadius: 8).fill(.regularMaterial))

                Spacer()

                Text("共 \(filteredTransactions.count) 笔流水")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .padding()
            .background(.ultraThinMaterial)

            Divider()

            // Transactions Stream
            if filteredTransactions.isEmpty {
                VStack(spacing: 12) {
                    Spacer()
                    Image(systemName: "tray")
                        .font(.system(size: 36))
                        .foregroundColor(.secondary)
                    Text("无匹配交易流水")
                        .foregroundColor(.secondary)
                    Spacer()
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                ScrollView {
                    LazyVStack(spacing: 8) {
                        ForEach(filteredTransactions) { tx in
                            TransactionCardView(tx: tx, viewState: viewState)
                        }
                    }
                    .padding()
                }
            }
        }
    }
}

struct TransactionCardView: View {
    let tx: JournalTransaction
    @ObservedObject var viewState: JournalViewState

    var isExpanded: Bool {
        viewState.isExpanded(tx.id)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Summary Header
            Button(action: {
                withAnimation(.easeInOut(duration: 0.15)) {
                    viewState.toggleExpanded(tx.id)
                }
            }) {
                HStack(spacing: 12) {
                    // Date Badge
                    Text(tx.date)
                        .font(.system(.caption, design: .monospaced))
                        .bold()
                        .padding(.horizontal, 6)
                        .padding(.vertical, 3)
                        .background(RoundedRectangle(cornerRadius: 4).fill(Color.accentColor.opacity(0.12)))
                        .foregroundColor(.accentColor)

                    // Flag
                    Text(tx.flag.isEmpty ? "*" : tx.flag)
                        .font(.system(.caption, design: .monospaced))
                        .bold()
                        .foregroundColor(tx.flag == "!" ? .orange : .secondary)

                    // Payee & Narration
                    VStack(alignment: .leading, spacing: 2) {
                        if !tx.payee.isEmpty {
                            Text(tx.payee)
                                .font(.subheadline)
                                .bold()
                        }
                        Text(tx.narration.isEmpty ? "(无摘要)" : tx.narration)
                            .font(.body)
                            .foregroundColor(tx.payee.isEmpty ? .primary : .secondary)
                    }

                    Spacer()

                    // Tags & Links badges
                    HStack(spacing: 4) {
                        ForEach(tx.tags.prefix(3), id: \.self) { tag in
                            Text("#\(tag)")
                                .font(.system(size: 10))
                                .padding(.horizontal, 5)
                                .padding(.vertical, 2)
                                .background(Capsule().fill(Color.blue.opacity(0.1)))
                                .foregroundColor(.blue)
                        }
                    }

                    // Postings Count / Quick Preview
                    Text("\(tx.postings.count) 条分录")
                        .font(.caption2)
                        .foregroundColor(.secondary)

                    Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }
                .padding(10)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)

            // Postings Expansion (Native Details Table)
            if isExpanded {
                Divider()
                    .padding(.horizontal, 8)

                VStack(spacing: 4) {
                    ForEach(Array(tx.postings.enumerated()), id: \.offset) { _, p in
                        HStack {
                            Text(p.account)
                                .font(.system(.caption, design: .monospaced))
                                .foregroundColor(.primary)
                            Spacer()
                            if !p.units_number.isEmpty {
                                let isNeg = p.units_number.hasPrefix("-")
                                HStack(spacing: 4) {
                                    Text(p.units_number)
                                        .font(.system(.caption, design: .monospaced))
                                        .bold()
                                        .foregroundColor(isNeg ? .red : .primary)
                                    Text(p.units_currency)
                                        .font(.system(size: 10, design: .monospaced))
                                        .foregroundColor(.secondary)
                                }
                            } else {
                                Text("(自动平衡)")
                                    .font(.system(size: 10))
                                    .foregroundColor(.secondary)
                            }
                        }
                        .padding(.horizontal, 14)
                        .padding(.vertical, 3)
                    }
                }
                .padding(.vertical, 6)
                .background(Color(NSColor.controlBackgroundColor).opacity(0.5))
            }
        }
        .background(RoundedRectangle(cornerRadius: 8).fill(.regularMaterial))
        .overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.secondary.opacity(0.15), lineWidth: 1))
    }
}
