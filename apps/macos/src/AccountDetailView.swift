// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import SwiftUI
import Charts

struct AccountDetailView: View {
    @ObservedObject var store: LedgerStore
    @ObservedObject var viewState: JournalViewState
    @Environment(\.ledgerLanguage) private var language

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Button(action: store.navigateBack) { Label(language.text("返回", "Back"), systemImage: "chevron.left") }
                    .disabled(!store.canNavigateBack)
                    .keyboardShortcut("[", modifiers: .command)
                let components = store.request.account.split(separator: ":").map(String.init)
                ForEach(Array(components.enumerated()), id: \.offset) { index, name in
                    let path = components.prefix(index + 1).joined(separator: ":")
                    Button(name) { store.showAccount(path) }.disabled(path == store.request.account)
                    if index < components.count - 1 { Text(":").foregroundColor(.secondary) }
                }
                Spacer()
                Button(language.text("查看持仓", "View holdings")) { store.showHoldings(store.request.account) }
                Picker(language.text("账户范围", "Account scope"), selection: Binding(get: { store.request.scope ?? "subtree" }, set: { scope in store.updateFilters { $0.scope = scope } })) {
                    Text(language.text("直属", "Direct")).tag("direct")
                    Text(language.text("整个子树", "Subtree")).tag("subtree")
                }.pickerStyle(.segmented).frame(maxWidth: 230)
            }.padding().background(.ultraThinMaterial)
            Divider()
            if let detail = store.accountDetail {
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 18) {
                        Text(detail.account).font(LedgerFonts.font(.title2)).bold().textSelection(.enabled)
                        HStack(spacing: 16) {
                            Text(lifecycle(detail))
                            if !detail.booking.isEmpty { Text("Booking: " + detail.booking) }
                            if !detail.currencies.isEmpty { Text(detail.currencies.joined(separator: ", ")) }
                            Spacer()
                        }.font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                        Text(language.text("截至日：", "Through: ") + detail.date_to + " · " + (store.state?.semantics_profile ?? "") + " / " + (store.state?.evaluation_revision ?? ""))
                            .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                        ForEach(detail.sources) { item in
                            VStack(alignment: .leading, spacing: 4) {
                                Text((item.kind == "open" ? language.text("开户：", "Opened: ") : language.text("关闭：", "Closed: ")) + item.date).font(LedgerFonts.font(.caption))
                                SourceActions(source: item.source)
                            }
                        }
                        HStack(alignment: .top, spacing: 24) {
                            balance(language.text("直属余额", "Direct balance"), detail.direct_amounts, detail.direct_valuation)
                            balance(language.text("子树余额", "Subtree balance"), detail.subtree_amounts, detail.subtree_valuation)
                            balance(language.text("所选范围期间变化", "Period change for selected scope"), detail.period_change, detail.period_valuation)
                        }
                        Text(language.text("期间仅限制变化量和流水；截至日余额包括期初存量。", "The period limits changes and transactions; balances include opening inventory."))
                            .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                        Divider()
                        Text(language.text("余额趋势", "Balance history")).font(LedgerFonts.font(.headline))
                        if detail.history.isEmpty {
                            Text(language.text("截至日无账户活动", "No account activity through this date")).foregroundColor(.secondary)
                        }
                        ForEach(detail.history) { series in
                            VStack(alignment: .leading, spacing: 6) {
                                Text(series.currency + " · " + (series.mode == "native" ? language.text("原币", "Native") : language.text("折算", "Converted"))).bold()
                                Chart {
                                    ForEach(series.points) { point in
                                        if let value = point.value {
                                            LineMark(x: .value("Date", point.date), y: .value("Balance", value), series: .value("Segment", point.segment))
                                            PointMark(x: .value("Date", point.date), y: .value("Balance", value))
                                        }
                                    }
                                }.frame(height: 150).accessibilityHidden(true)
                                ForEach(series.points) { point in
                                    HStack {
                                        Text(point.date)
                                        Spacer()
                                        if let total = point.valuation.complete_total {
                                            Text(total.label + " " + series.currency).help(total.exact)
                                        } else {
                                            Text(language.text("缺报价：", "Missing quotes: ") + point.valuation.missing.map { $0.commodity }.joined(separator: ", ")).foregroundColor(.orange)
                                        }
                                    }.font(LedgerFonts.font(.caption, design: .monospaced))
                                }
                            }
                        }
                        Divider()
                        TextField(language.text("筛选账户流水（不改变余额）", "Filter account transactions (balances unchanged)"), text: Binding(get: { store.request.text }, set: { text in store.updateFilters { $0.text = text } }))
                            .textFieldStyle(.roundedBorder)
                        if let page = store.journalPayload {
                            HStack {
                                Text(language.text("共 \(page.total_count) 笔交易", "\(page.total_count) transactions"))
                                Spacer()
                                Button(language.text("上一页", "Previous")) { store.journalPage(page.offset - page.limit) }.disabled(page.offset == 0)
                                Button(language.text("下一页", "Next")) { store.journalPage(page.offset + page.limit) }.disabled(!page.has_more)
                            }.font(LedgerFonts.font(.caption))
                            if page.transactions.isEmpty { Text(language.text("无匹配交易", "No matching transactions")).foregroundColor(.secondary) }
                            ForEach(page.transactions) { tx in TransactionCardView(tx: tx, viewState: viewState, openAccount: store.showAccount) }
                        }
                    }.padding(20)
                }
            } else { Spacer() }
        }
    }

    private func balance(_ title: String, _ amounts: [String: DisplayAmount], _ valuation: AccountValuation) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title).font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
            BalanceColumnView(balances: amounts, valuation: valuation)
        }.frame(maxWidth: .infinity, alignment: .leading)
    }

    private func lifecycle(_ detail: AccountDetailPayload) -> String {
        switch detail.lifecycle {
        case "closed": return language.text("已关闭；历史保留", "Closed; history retained")
        case "not-opened": return language.text("截至日尚未开户", "Not yet opened at cutoff")
        case "implicit": return language.text("账户目录", "Account directory")
        default: return language.text("已开户", "Open")
        }
    }
}
