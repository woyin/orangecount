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
    @ObservedObject var store: LedgerStore
    @ObservedObject var viewState: JournalViewState
    @Environment(\.ledgerLanguage) private var language
    @FocusState private var searchFocused: Bool

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                TextField(language.text("搜索收款人、摘要、账户、标签、链接…", "Search payee, narration, account, tags, links…"), text: Binding(get: { store.request.text }, set: { value in store.updateFilters { $0.text = value } }))
                    .focused($searchFocused).textFieldStyle(.roundedBorder)
                Button(action: { searchFocused = true }) { Image(systemName: "magnifyingglass") }
                    .keyboardShortcut("f", modifiers: .command)
                    .help(language.text("查找流水", "Find transactions"))
                Picker(language.text("账户", "Account"), selection: Binding(get: { store.request.account }, set: { value in store.updateFilters { $0.account = value } })) {
                    Text(language.text("全部账户", "All accounts")).tag("")
                    ForEach(store.summary?.accounts ?? [], id: \.self) { Text($0).tag($0) }
                }.frame(maxWidth: 240)
                Picker(language.text("排序", "Order"), selection: Binding(get: { store.request.sort }, set: { value in store.updateFilters { $0.sort = value } })) {
                    Text(language.text("最新优先", "Newest first")).tag("")
                    Text(language.text("最早优先", "Oldest first")).tag("ascending")
                }.frame(maxWidth: 160)
            }.padding().background(.ultraThinMaterial)
            Divider()
            if let page = store.journalPayload {
                HStack {
                    Text(language.text("共 \(page.total_count) 笔；本页 \(page.transactions.count) 笔", "\(page.total_count) matching; \(page.transactions.count) on this page"))
                    Spacer()
                    Button(language.text("上一页", "Previous")) { store.journalPage(page.offset - page.limit) }.disabled(page.offset == 0)
                    Button(language.text("下一页", "Next")) { store.journalPage(page.offset + page.limit) }.disabled(!page.has_more)
                }.font(LedgerFonts.font(.caption)).padding(.horizontal).padding(.vertical, 8)
                if page.transactions.isEmpty {
                    Text(language.text("无匹配交易流水", "No matching transactions")).foregroundColor(.secondary).frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    ScrollView {
                        LazyVStack(spacing: 8) {
                            ForEach(page.transactions) { tx in TransactionCardView(tx: tx, viewState: viewState, openAccount: store.showAccount) }
                        }.padding()
                    }.id(page.offset)
                }
            } else { Spacer() }
        }
    }
}

struct TransactionCardView: View {
    @Environment(\.ledgerLanguage) private var language
    let tx: JournalTransaction
    @ObservedObject var viewState: JournalViewState
    var openAccount: (String) -> Void = { _ in }

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
                        .font(LedgerFonts.font(.caption, design: .monospaced))
                        .bold()
                        .padding(.horizontal, 6)
                        .padding(.vertical, 3)
                        .background(RoundedRectangle(cornerRadius: 4).fill(Color.accentColor.opacity(0.12)))
                        .foregroundColor(.accentColor)

                    // Flag
                    Text(tx.flag.isEmpty ? "*" : tx.flag)
                        .font(LedgerFonts.font(.caption, design: .monospaced))
                        .bold()
                        .foregroundColor(tx.flag == "!" ? .orange : .secondary)

                    // Payee & Narration
                    VStack(alignment: .leading, spacing: 2) {
                        if !tx.payee.isEmpty {
                            Text(tx.payee)
                                .font(LedgerFonts.font(.subheadline))
                                .bold()
                        }
                        Text(tx.narration.isEmpty ? language.text("（无摘要）", "(No narration)") : tx.narration)
                            .font(LedgerFonts.font(.body))
                            .foregroundColor(tx.payee.isEmpty ? .primary : .secondary)
                    }

                    Spacer()

                    // Tags & Links badges
                    HStack(spacing: 4) {
                        ForEach(tx.tags, id: \.self) { tag in
                            Text("#\(tag)")
                                .font(.system(size: 10))
                                .padding(.horizontal, 5)
                                .padding(.vertical, 2)
                                .background(Capsule().fill(Color.blue.opacity(0.1)))
                                .foregroundColor(.blue)
                        }
                    }

                    // Postings Count / Quick Preview
                    Text(language.text("\(tx.postings.count) 条分录", "\(tx.postings.count) postings"))
                        .font(LedgerFonts.font(.caption2))
                        .foregroundColor(.secondary)

                    Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                        .font(LedgerFonts.font(.caption2))
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

                VStack(alignment: .leading, spacing: 6) {
                    SourceActions(source: tx.source).padding(.horizontal, 14)
                    if !tx.links.isEmpty { Text(tx.links.map { "^" + $0 }.joined(separator: "  ")).font(LedgerFonts.font(.caption)).padding(.horizontal, 14).textSelection(.enabled) }
                    ForEach(Array(tx.postings.enumerated()), id: \.offset) { _, p in
                        HStack {
                            Button(p.account) { openAccount(p.account) }
                                .buttonStyle(.plain)
                                .font(LedgerFonts.font(.caption, design: .monospaced))
                                .help(language.text("查看账户详情", "View account details"))
                            Spacer()
                            if !p.units_number.isEmpty {
                                let isNeg = p.units_number.hasPrefix("-")
                                HStack(spacing: 4) {
                                    Text(p.amount?.label ?? p.units_number)
                                        .help(p.amount?.exact ?? p.units_number)
                                        .font(LedgerFonts.font(.caption, design: .monospaced))
                                        .bold()
                                        .foregroundColor(isNeg ? .red : .primary)
                                    Text(p.units_currency)
                                        .font(.system(size: 10, design: .monospaced))
                                        .foregroundColor(.secondary)
                                }
                            } else {
                                Text(language.text("（自动平衡）", "(Balanced automatically)"))
                                    .font(.system(size: 10))
                                    .foregroundColor(.secondary)
                            }
                        }
                        .padding(.horizontal, 14)
                        .padding(.vertical, 3)
                        if let cost = p.cost {
                            Text(cost.display).font(.caption.monospaced()).padding(.horizontal, 14)
                        }
                        if let price = p.price {
                            Text((price.total ? "@@ " : "@ ") + price.number.label + " " + price.currency).font(.caption.monospaced()).padding(.horizontal, 14).help(price.number.exact)
                        }
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
