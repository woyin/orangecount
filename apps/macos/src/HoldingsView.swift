// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import SwiftUI
import Charts

@MainActor
final class HoldingsViewState: ObservableObject {
    @Published var expanded: Set<String> = []
}

struct HoldingsView: View {
    @ObservedObject var store: LedgerStore
    @ObservedObject var viewState: HoldingsViewState
    @Environment(\.ledgerLanguage) private var language

    private var isDetail: Bool { store.request.group_id != nil || store.request.event_id != nil }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            controls
            Divider()
            if let payload = store.holdingsPayload {
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 18) {
                        context(payload)
                        switch payload.section {
                        case .groups:
                            pager(payload.groups_page)
                            if payload.groups.isEmpty { empty(language.text("截至日没有持仓", "No holdings at cutoff")) }
                            ForEach(payload.groups) { group in groupCard(group, payload) }
                        case .events:
                            pager(payload.events_page)
                            if payload.events.isEmpty { empty(language.text("所选期间没有归约事件", "No reductions in this period")) }
                            ForEach(payload.events) { event in eventCard(event, payload) }
                        case .lots:
                            lotDetail(payload)
                        case .history:
                            if store.request.group_id == nil { pager(payload.history_page) }
                            if payload.average_history.isEmpty { empty(language.text("没有可确定的成本历史", "No known cost history")) }
                            ForEach(payload.average_history) { series in historyCard(series) }
                        }
                    }.padding(20)
                }
            } else { Spacer() }
        }
        .onChange(of: store.state?.snapshot_id) { _ in viewState.expanded = [] }
    }

    private var controls: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Button(action: store.navigateBack) { Label(language.text("返回", "Back"), systemImage: "chevron.left") }
                    .disabled(!store.canNavigateBack).keyboardShortcut("[", modifiers: .command)
                Picker(language.text("账户", "Account"), selection: Binding(get: { store.request.account }, set: { account in
                    store.updateFilters { $0.account = account; $0.scope = account.isEmpty ? nil : "subtree" }
                })) {
                    Text(language.text("全部资产与负债", "All assets and liabilities")).tag("")
                    ForEach(accounts, id: \.self) { Text($0).tag($0) }
                }.frame(maxWidth: .infinity)
                if !store.request.account.isEmpty {
                    Button { store.showAccount(store.request.account) } label: { Image(systemName: "folder") }
                        .help(language.text("查看账户详情", "Open account details"))
                }
            }
            HStack {
                Picker(language.text("分组", "Group by"), selection: Binding(get: { store.request.group_by ?? "account" }, set: { value in store.updateFilters { $0.group_by = value } })) {
                    Text(language.text("账户", "Account")).tag("account")
                    Text(language.text("商品", "Commodity")).tag("commodity")
                    Text(language.text("成本币种", "Cost currency")).tag("cost_currency")
                }.frame(maxWidth: 220)
                if !store.request.account.isEmpty {
                    Picker(language.text("范围", "Scope"), selection: Binding(get: { store.request.scope ?? "subtree" }, set: { value in store.updateFilters { $0.scope = value } })) {
                        Text(language.text("直属", "Direct")).tag("direct")
                        Text(language.text("子树", "Subtree")).tag("subtree")
                    }.frame(maxWidth: 165)
                }
                Spacer(minLength: 0)
                Picker(language.text("排序", "Sort"), selection: Binding(get: { store.request.sort == "descending" ? "descending" : "ascending" }, set: store.setHoldingSort)) {
                    Text(language.text("升序", "Ascending")).tag("ascending")
                    Text(language.text("降序", "Descending")).tag("descending")
                }.frame(maxWidth: 170)
            }
            if !isDetail {
                Picker(language.text("持仓内容", "Holdings section"), selection: Binding(get: { store.request.holding_section ?? "groups" }, set: { value in
                    if let section = HoldingSection(rawValue: value) { store.setHoldingSection(section) }
                })) {
                    Text(language.text("截至日持仓", "Holdings at cutoff")).tag("groups")
                    Text(language.text("归约与收益", "Reductions and gains")).tag("events")
                    Text(language.text("平均成本历史", "Average cost history")).tag("history")
                }.pickerStyle(.segmented)
            }
        }.padding(12).background(.ultraThinMaterial)
    }

    private var accounts: [String] {
        Array(Set((store.summary?.accounts ?? []) + (store.request.account.isEmpty ? [] : [store.request.account]))).sorted()
    }

    private func context(_ payload: HoldingsPayload) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(language.text("截至日：", "Through: ") + payload.date_to + " · " + (store.state?.semantics_profile ?? "") + " / " + (store.state?.evaluation_revision ?? ""))
            Text(language.text("当前口径：", "Selected valuation: ") + valuationName(payload.valuation) + " · " + (payload.currency.isEmpty ? language.text("各成本原币种", "Native cost currencies") : payload.currency))
            Text(language.text("持仓包括期初存量；期间只限制归约事件与成本历史。不同商品的数量、不同成本币种的均价分别阅读。", "Holdings include opening inventory; the period limits reductions and cost history. Read quantities and average costs separately by commodity and cost currency."))
        }.font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
    }

    private func groupCard(_ group: HoldingGroup, _ payload: HoldingsPayload) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text(group.commodity).font(LedgerFonts.font(.title3)).bold()
                Text(inventoryName(group.inventory_kind)).foregroundColor(.secondary)
                Spacer()
                if group.cost_currency != nil {
                    Button(language.text("成本历史", "Cost history")) { store.showHoldingHistory(groupID: group.id) }
                }
                Button(language.text("全部批次", "All lots")) { store.showHoldingLots(groupID: group.id) }
            }
            accountLinks(group.accounts)
            if payload.valuation == "units" {
                amountRow(language.text("当前口径：数量", "Selected valuation: units"), group.quantity, group.commodity)
            } else if payload.valuation == "market-value" {
                amountRow(language.text("当前口径：市值", "Selected valuation: market value"), group.market_valuation.complete_total, group.market_valuation.target_currency)
            } else {
                amountRow(language.text("当前口径：成本", "Selected valuation: cost"), payload.currency.isEmpty ? group.cost_basis : group.cost_valuation.complete_total, payload.currency.isEmpty ? group.cost_currency ?? "" : group.cost_valuation.target_currency)
            }
            amountRow(language.text("数量", "Quantity"), group.quantity, group.commodity)
            amountRow(language.text("原成本基础", "Original cost basis"), group.cost_basis, group.cost_currency ?? "")
            amountRow(group.inventory_kind == .mixed ? language.text("有符号成本比率", "Signed cost ratio") : language.text("平均成本", "Average cost"), group.average_cost, group.cost_currency ?? "")
            valuation(language.text("目标币种成本", "Cost in valuation currency"), group.cost_valuation, payload)
            valuation(language.text("市值", "Market value"), group.market_valuation, payload)
            amountRow(language.text("未实现差额", "Unrealized difference"), group.unrealized_gain, group.market_valuation.target_currency)
            if group.unrealized_gain == nil { reason(group.gain_reason) }
            DisclosureGroup(isExpanded: expanded("group:" + group.id)) {
                lotRows(group.lots)
                preview(group.lots_page)
            } label: { Text(language.text("批次预览", "Lot preview") + " (\(group.lots_page.total_count))") }
        }.padding(14).background(.regularMaterial, in: RoundedRectangle(cornerRadius: 10))
    }

    private func eventCard(_ event: ReductionEvent, _ payload: HoldingsPayload) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(event.date + " · " + event.commodity + " · " + eventName(event.kind)).font(LedgerFonts.font(.headline))
            if !event.narration.isEmpty { Text(event.narration).textSelection(.enabled) }
            accountLinks([event.account])
            amountRow(language.text("减少数量", "Reduction quantity"), event.quantity, event.commodity)
            amountRow(language.text("实际消费数量", "Consumed quantity"), event.consumed_quantity, event.commodity)
            amountRow(language.text("消费成本", "Consumed cost"), event.cost_basis, event.cost_currency ?? "")
            amountRow(language.text("声明售价", "Declared proceeds"), event.proceeds, event.price_currency)
            if let quote = event.conversion {
                amountRow(language.text("成本币种售价", "Proceeds in cost currency"), event.proceeds_at_cost, event.cost_currency ?? "")
                quoteRow(quote, payload)
            }
            amountRow(language.text("毛收益", "Gross gain"), event.gross_gain, event.cost_currency ?? "")
            if event.gross_gain == nil { reason(event.gross_reason) }
            amountRow(language.text("归属费用", "Attributed fees"), event.fees, event.cost_currency ?? "")
            amountRow(language.text("净收益", "Net gain"), event.net_gain, event.cost_currency ?? "")
            if event.net_gain == nil { reason(event.net_reason) }
            SourceActions(source: event.source)
            DisclosureGroup(isExpanded: expanded("event:" + event.id)) {
                lotRows(event.lots)
                preview(event.lots_page)
                Button(language.text("查看全部消费批次", "View all consumed lots")) { store.showHoldingEventLots(eventID: event.id) }
            } label: { Text(language.text("实际消费批次", "Consumed lots") + " (\(event.lots_page.total_count))") }
            if event.merge_lots_page.total_count > 0 {
                DisclosureGroup(isExpanded: expanded("merge:" + event.id)) {
                    Text(language.text("合并前整池用于解释均价来源，不是额外出售成本。", "The pre-merge pool explains average cost; it is not additional sale cost.")).foregroundColor(.secondary)
                    lotRows(event.merge_lots)
                    preview(event.merge_lots_page)
                    Button(language.text("查看全部合并贡献", "View all merge contributions")) { store.showHoldingEventLots(eventID: event.id, merged: true) }
                } label: { Text(language.text("AVERAGE 合并前整池", "AVERAGE pre-merge pool") + " (\(event.merge_lots_page.total_count))") }
            }
        }.padding(14).background(.regularMaterial, in: RoundedRectangle(cornerRadius: 10))
    }

    @ViewBuilder private func lotDetail(_ payload: HoldingsPayload) -> some View {
        if let group = payload.groups.first {
            Text(group.commodity + " · " + language.text("完整批次", "All lots")).font(LedgerFonts.font(.headline))
            accountLinks(group.accounts)
            amountRow(language.text("完整组成本", "Full group cost"), group.cost_basis, group.cost_currency ?? "")
            pager(group.lots_page)
            lotRows(group.lots)
        } else if let event = payload.events.first {
            let merged = store.request.lot_kind == "merged"
            Text(event.date + " · " + event.commodity).font(LedgerFonts.font(.headline))
            accountLinks([event.account])
            amountRow(language.text("本次实际消费成本", "Actual consumed cost"), event.cost_basis, event.cost_currency ?? "")
            SourceActions(source: event.source)
            Text(merged ? language.text("AVERAGE 合并前整池；不是额外出售成本", "AVERAGE pre-merge pool; not additional sale cost") : language.text("本次实际消费批次", "Consumed lots for this reduction")).font(LedgerFonts.font(.headline))
            pager(merged ? event.merge_lots_page : event.lots_page)
            lotRows(merged ? event.merge_lots : event.lots)
        } else { empty(language.text("当前快照没有所选批次", "Selected lots are unavailable in this snapshot")) }
    }

    private func lotRows(_ lots: [HoldingLot]) -> some View {
        ForEach(lots) { lot in
            VStack(alignment: .leading, spacing: 5) {
                Divider()
                accountLinks([lot.account])
                amountRow(language.text("数量", "Quantity"), lot.quantity, lot.commodity)
                amountRow(language.text("单位成本", "Unit cost"), lot.cost_number, lot.cost_currency ?? "")
                amountRow(language.text("成本基础", "Cost basis"), lot.cost_basis, lot.cost_currency ?? "")
                Text(language.text("成本日期：", "Cost date: ") + (lot.cost_date ?? language.text("无成本日期", "No cost date")))
                Text(language.text("标签：", "Label: ") + (lot.cost_label.map { $0.isEmpty ? language.text("空字符串", "Empty string") : $0 } ?? language.text("无标签", "Absent")))
                Text(lot.origin == .averageMerge ? language.text("来源：AVERAGE 重组建立", "Origin: AVERAGE reorganization") : language.text("来源：原分录建立", "Origin: posting establishment")).foregroundColor(.secondary)
                SourceActions(source: lot.source)
            }.font(LedgerFonts.font(.caption)).textSelection(.enabled)
        }
    }

    private func historyCard(_ series: AverageCostSeries) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text(series.commodity + " / " + series.cost_currency).font(LedgerFonts.font(.headline))
                Spacer()
                if !isDetail { Button(language.text("全部历史", "All history")) { store.showHoldingHistory(groupID: series.id) } }
            }
            accountLinks(series.accounts)
            if isDetail { pager(series.points_page) } else { preview(series.points_page) }
            Chart {
                ForEach(Array(series.points.enumerated()), id: \.offset) { _, point in
                    if let value = point.value {
                        LineMark(x: .value("Date", point.date), y: .value("Average cost", value), series: .value("Holding segment", point.segment))
                        PointMark(x: .value("Date", point.date), y: .value("Average cost", value))
                    }
                }
            }.frame(height: 180).accessibilityHidden(true)
            ForEach(Array(series.points.enumerated()), id: \.offset) { _, point in
                VStack(alignment: .leading, spacing: 4) {
                    Divider()
                    Text(point.date).bold()
                    amountRow(language.text("数量", "Quantity"), point.quantity, series.commodity)
                    amountRow(language.text("成本基础", "Cost basis"), point.cost_basis, series.cost_currency)
                    if let average = point.average_cost { amountRow(language.text("平均成本", "Average cost"), average, series.cost_currency) }
                    else { Text(language.text("已清仓；均价不可用，曲线断段", "Cleared; average cost unavailable, line segment ends")).foregroundColor(.secondary) }
                }.font(LedgerFonts.font(.caption))
            }
        }.padding(14).background(.regularMaterial, in: RoundedRectangle(cornerRadius: 10))
    }

    private func amountRow(_ title: String, _ value: DisplayAmount?, _ currency: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            HStack(alignment: .firstTextBaseline) {
                Text(title).foregroundColor(.secondary)
                Spacer(minLength: 12)
                if let value { Text(value.label + (currency.isEmpty ? "" : " " + currency)).font(LedgerFonts.font(.body, design: .monospaced)).textSelection(.enabled).help(value.exact) }
                else { Text(language.text("不可用", "Unavailable")).foregroundColor(.secondary) }
            }
            if let value, value.approximate {
                Text(language.text("精确值：", "Exact: ") + value.exact + (currency.isEmpty ? "" : " " + currency))
                    .font(LedgerFonts.font(.caption, design: .monospaced)).textSelection(.enabled)
            }
        }
    }

    private func valuation(_ title: String, _ value: AccountValuation, _ payload: HoldingsPayload) -> some View {
        VStack(alignment: .leading, spacing: 5) {
            amountRow(title, value.complete_total, value.target_currency)
            if value.complete_total == nil, let portion = value.valued_portion {
                amountRow(language.text("已估部分（非完整总计）", "Valued portion (not a complete total)"), portion, value.target_currency)
            }
            if !value.missing.isEmpty {
                Text(language.text("缺项：", "Missing: ") + value.missing.map { $0.commodity + " · " + missingName($0.reason) }.joined(separator: "; "))
                    .font(LedgerFonts.font(.caption)).foregroundColor(.orange)
            }
            ForEach(Array(value.quotes.enumerated()), id: \.offset) { _, quote in quoteRow(quote, payload) }
        }
    }

    private func quoteRow(_ quote: ValuationQuote, _ payload: HoldingsPayload) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(quote.base + "/" + quote.quote + " " + quote.rate.label + " @ " + quote.date + (quote.inverse ? language.text("（倒数报价）", " (inverse quote)") : ""))
                .help(quote.rate.exact).textSelection(.enabled)
            if let source = payload.source(for: quote) { SourceActions(source: source) }
            Button(language.text("查看价格历史", "View price history")) { store.showPrices(base: quote.base, quote: quote.quote) }
        }.font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
    }

    private func accountLinks(_ accounts: [String]) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            ForEach(accounts, id: \.self) { account in Button(account) { store.showAccount(account) }.buttonStyle(.link).lineLimit(2) }
        }
    }

    private func pager(_ page: ProjectionPage) -> some View {
        HStack {
            Text(pageLabel(page)).textSelection(.enabled)
            Spacer()
            Button(language.text("上一页", "Previous")) { store.holdingsPage(max(0, page.offset - page.limit), page: page) }.disabled(page.offset == 0)
            Button(language.text("下一页", "Next")) { store.holdingsPage(page.offset + page.limit, page: page) }.disabled(!page.has_more)
        }.font(LedgerFonts.font(.caption))
    }

    private func preview(_ page: ProjectionPage) -> some View {
        Text(pageLabel(page) + (page.truncated ? language.text(" · 预览未包含全部记录", " · Preview does not contain all records") : ""))
            .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
    }

    private func pageLabel(_ page: ProjectionPage) -> String {
        if page.returned_count == 0 { return language.text("共 \(page.total_count) 条；当前页为空", "\(page.total_count) records; this page is empty") }
        return language.text("第 \(page.offset + 1)–\(page.offset + page.returned_count) 条，共 \(page.total_count) 条", "\(page.offset + 1)–\(page.offset + page.returned_count) of \(page.total_count) records")
    }

    private func expanded(_ id: String) -> Binding<Bool> {
        Binding(get: { viewState.expanded.contains(id) }, set: { value in
            if value { viewState.expanded.insert(id) } else { viewState.expanded.remove(id) }
        })
    }

    private func empty(_ message: String) -> some View { Text(message).foregroundColor(.secondary).padding(.vertical, 20) }
    private func reason(_ value: HoldingReason) -> some View { Text(reasonName(value)).font(LedgerFonts.font(.caption)).foregroundColor(.orange) }

    private func inventoryName(_ kind: HoldingInventoryKind) -> String {
        switch kind {
        case .long: return language.text("多头持仓", "Long inventory")
        case .short: return language.text("空头持仓", "Short inventory")
        case .mixed: return language.text("混合长短仓", "Mixed long and short inventory")
        case .plain: return language.text("无成本数量", "Units without cost")
        }
    }
    private func valuationName(_ value: String) -> String {
        switch value {
        case "units": return language.text("数量", "Units")
        case "market-value": return language.text("市值", "Market value")
        default: return language.text("成本", "At cost")
        }
    }
    private func eventName(_ kind: ReductionKind) -> String {
        switch kind {
        case .sale: return language.text("可确定出售", "Known sale")
        case .reduction: return language.text("归约", "Reduction")
        case .transferOrExchange: return language.text("转移或交换", "Transfer or exchange")
        case .shortOrMixed: return language.text("开空或混合归约", "Short or mixed reduction")
        }
    }
    private func missingName(_ value: String) -> String {
        switch value {
        case "missing-cost": return language.text("缺成本", "Missing cost")
        case "unspecified-quote": return language.text("未选择报价币种", "Quote currency unspecified")
        case "missing-local-quote", "missing-price": return language.text("缺本地报价", "Missing local quote")
        default: return language.text("估值不可用", "Valuation unavailable")
        }
    }
    private func reasonName(_ value: HoldingReason) -> String {
        switch value {
        case .known: return ""
        case .missingCost: return language.text("成本未知，无法确定差额。", "Cost is unknown; the difference is unavailable.")
        case .mixedCostCurrencies: return language.text("成本涉及多个币种，无法唯一确定。", "Multiple cost currencies prevent a unique result.")
        case .missingSalePrice: return language.text("没有明确售价，无法确定出售收益。", "No declared sale price; sale gain is unavailable.")
        case .missingProceedsQuote: return language.text("事件日缺少售价换算报价。", "The event date has no proceeds conversion quote.")
        case .negativeSalePrice: return language.text("源售价为负；保留原值，不推断普通出售收益。", "Declared proceeds are negative; ordinary sale gain is unavailable.")
        case .transferOrExchange: return language.text("可能为资产转移或交换，无法确认出售。", "Possible asset transfer or exchange; a sale cannot be confirmed.")
        case .shortPositionOpening: return language.text("开空或混合开仓，不能按消费多头成本计算。", "Short or mixed opening cannot use long-lot consumption gain.")
        case .unattributedExpenses: return language.text("存在未明确归属的费用，净收益未知。", "Expenses are not explicitly attributed; net gain is unknown.")
        case .ambiguousTradeFees: return language.text("多个归约共用费用，未自动分摊。", "Multiple reductions share fees; no automatic allocation.")
        case .missingFeeOrSettlementQuote: return language.text("事件日缺费用或结算换算报价。", "The event date lacks a fee or settlement quote.")
        case .proceedsNotReconciled: return language.text("结算与声明售价、费用不一致，净收益未知。", "Settlement does not reconcile with proceeds and fees; net gain is unknown.")
        case .unsupportedTradeCounterpart: return language.text("其他分录使交易归因不确定。", "Other postings prevent a reliable trade attribution.")
        case .missingPrice: return language.text("缺少截至日商品报价。", "No commodity quote through the cutoff.")
        case .missingCostQuote: return language.text("目标币种成本不可用，未实现差额未知。", "Cost in the target currency is unavailable; unrealized difference is unknown.")
        }
    }
}
