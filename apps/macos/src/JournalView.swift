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

    private var directiveKinds: [String] {
        store.request.journal_mode == "config" ? ["option", "plugin", "include", "pushtag", "poptag", "pushmeta", "popmeta"] : ["open", "close", "commodity", "balance", "pad", "event", "query", "price", "document", "note", "custom", "transaction"]
    }

    var body: some View {
        VStack(spacing: 0) {
            VStack {
            HStack {
                Picker(language.text("记录", "Records"), selection: Binding(get: { store.request.journal_mode ?? "transactions" }, set: store.selectJournalMode)) {
                    Text(language.text("交易", "Transactions")).tag("transactions")
                    Text(language.text("全部指令", "All directives")).tag("all")
                    Text(language.text("配置", "Configuration")).tag("config")
                }.frame(maxWidth: 200)
                if store.request.journal_mode == "all" || store.request.journal_mode == "config" {
                    Picker(language.text("类型", "Kind"), selection: Binding(get: { store.request.directive_kind ?? "" }, set: { value in store.updateFilters { $0.directive_kind = value.isEmpty ? nil : value } })) {
                        Text(language.text("全部类型", "All kinds")).tag("")
                        ForEach(directiveKinds, id: \.self) { kind in Text(directiveKindName(kind, language: language)).tag(kind) }
                    }.frame(maxWidth: 160)
                }
                Button(language.text("附件", "Attachments"), action: store.showDocuments)
                Spacer()
            }
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
                    Text(store.request.journal_mode == "config" ? language.text("加载顺序", "Load order") : language.text("最新优先", "Newest first")).tag("")
                    Text(language.text("最早优先", "Oldest first")).tag("ascending")
                    if store.request.journal_mode == "config" { Text(language.text("逆序", "Reverse order")).tag("descending") }
                }.frame(maxWidth: 160)
            }
            }.padding().background(.ultraThinMaterial)
            Divider()
            if let page = store.directivesPayload {
                VStack(alignment: .leading) {
                    if page.mode == "config" { Text(language.text("配置按源加载顺序展示；日期筛选不适用。", "Configuration follows source load order; date filters do not apply.")) }
                    HStack {
                        Text(language.text("共 \(page.page.total_count) 条；本页 \(page.records.count) 条", "\(page.page.total_count) matching records; \(page.records.count) on this page"))
                        Spacer()
                        Button(language.text("上一页", "Previous")) { store.journalPage(page.page.offset - page.page.limit) }.disabled(page.page.offset == 0)
                        Button(language.text("下一页", "Next")) { store.journalPage(page.page.offset + page.page.limit) }.disabled(!page.page.has_more)
                    }
                }.font(LedgerFonts.font(.caption)).padding(8)
                if page.records.isEmpty {
                    Text(language.text("无匹配记录", "No matching records")).foregroundColor(.secondary).frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    ScrollView {
                        LazyVStack(alignment: .leading, spacing: 8) {
                            ForEach(page.records) { record in DirectiveCard(record: record, mode: page.mode, viewState: viewState, openAttachment: store.openDocument) }
                        }.padding()
                    }.id("\(page.mode):\(page.page.offset)")
                }
            } else if let page = store.journalPayload {
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

private func directiveKindName(_ kind: String, language: LedgerLanguage) -> String {
    let names = ["option":"选项", "plugin":"插件声明", "include":"包含文件", "pushtag":"压入标签", "poptag":"弹出标签", "pushmeta":"压入元数据", "popmeta":"弹出元数据", "open":"开户", "close":"关闭账户", "commodity":"商品声明", "balance":"余额断言", "pad":"余额补齐", "event":"事件", "query":"保存查询", "price":"价格", "document":"附件声明", "note":"备注", "custom":"自定义记录", "transaction":"交易", "dialect":"简写交易"]
    return language.text(names[kind] ?? kind, kind)
}

private struct DirectiveCard: View {
    let record: DirectiveRecord
    let mode: String
    @ObservedObject var viewState: JournalViewState
    @Environment(\.ledgerLanguage) private var language
    let openAttachment: (Attachment, Bool) -> Void
    private var expansionID: String { "directive:\(mode):\(record.id)" }
    private func fieldName(_ name: String) -> String {
        let names = ["key":"键", "value":"值", "module":"模块", "config":"配置", "path":"路径", "tag":"标签", "account":"账户", "currency":"币种", "booking":"批次选择方法", "amount":"金额", "tolerance":"容差", "source_account":"补齐来源账户", "event_type":"事件类型", "name":"名称", "query":"查询", "filename":"文件", "comment":"备注", "custom_type":"记录类型", "flag":"标记", "payee":"收款人", "narration":"摘要"]
        return language.text(names[name] ?? name, name)
    }
    private func attachmentStatus(_ status: String) -> String {
        let names = ["available":"可用", "missing":"文件不存在", "outside-root":"超出账本目录", "unsupported-kind":"不是普通文件", "unreadable":"无法读取"]
        let english = ["available":"Available", "missing":"Missing file", "outside-root":"Outside ledger folder", "unsupported-kind":"Not a regular file", "unreadable":"Unreadable"]
        return language.text(names[status] ?? status, english[status] ?? status)
    }
    var body: some View {
        DisclosureGroup(isExpanded: Binding(get: { viewState.isExpanded(expansionID) }, set: { value in if value != viewState.isExpanded(expansionID) { viewState.toggleExpanded(expansionID) } })) {
            VStack(alignment: .leading, spacing: 6) {
                SourceActions(source: record.source)
                if record.generated {
                    Text(language.text("由源声明生成：", "Generated from source declaration: ") + directiveKindName(record.source_kind, language: language)).foregroundColor(.secondary)
                }
                ForEach(Array(record.fields.enumerated()), id: \.offset) { _, field in
                    HStack(alignment: .top) { Text(fieldName(field.name) + ":").foregroundColor(.secondary); Text(field.value.exactText).textSelection(.enabled) }
                }
                ForEach(record.attachments) { attachment in
                    HStack {
                        Text(attachment.filename).textSelection(.enabled)
                        Text(attachmentStatus(attachment.status)).foregroundColor(.secondary)
                        Button(language.text("打开附件", "Open attachment")) { openAttachment(attachment, false) }.disabled(attachment.status != "available")
                        Button(language.text("在 Finder 中显示附件", "Reveal attachment in Finder")) { openAttachment(attachment, true) }.disabled(attachment.status != "available")
                    }
                }
                if !record.tags.isEmpty { Text(record.tags.map { "#" + $0 }.joined(separator: "  ")).textSelection(.enabled) }
                if !record.links.isEmpty { Text(record.links.map { "^" + $0 }.joined(separator: "  ")).textSelection(.enabled) }
                ForEach(Array(record.metadata.enumerated()), id: \.offset) { _, metadata in
                    HStack(alignment: .top) { Text(metadata.key + ": " + metadata.value.exactText).textSelection(.enabled); Spacer(); SourceActions(source: metadata.source) }
                }
                ForEach(Array(record.postings.enumerated()), id: \.offset) { _, item in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(item.posting.account + "  " + (item.posting.amount?.exact ?? item.posting.units_number) + " " + item.posting.units_currency).textSelection(.enabled)
                        if let cost = item.posting.cost { Text(cost.display).textSelection(.enabled) }
                        if let price = item.posting.price { Text((price.total ? "@@ " : "@ ") + price.number.exact + " " + price.currency).textSelection(.enabled) }
                        SourceActions(source: item.posting.source)
                        ForEach(Array(item.metadata.enumerated()), id: \.offset) { _, metadata in Text(metadata.key + ": " + metadata.value.exactText).textSelection(.enabled) }
                    }.padding(.leading, 12)
                }
            }.font(LedgerFonts.font(.caption, design: .monospaced)).padding(8)
        } label: {
            HStack {
                Text(record.date ?? language.text("无日期配置", "Undated configuration")).font(LedgerFonts.font(.caption, design: .monospaced))
                Text(directiveKindName(record.kind, language: language)).bold()
                if record.generated { Text(language.text("生成", "Generated")).foregroundColor(.secondary) }
                Text(record.fields.first(where: { $0.name == "narration" || $0.name == "comment" || $0.name == "name" })?.value.exactText ?? record.fields.first?.value.exactText ?? "").lineLimit(1)
                Spacer()
            }
        }.padding(10).background(RoundedRectangle(cornerRadius: 8).fill(.regularMaterial))
    }
}
