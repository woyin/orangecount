// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import SwiftUI
import Charts

struct PricesView: View {
    @ObservedObject var store: LedgerStore
    @Environment(\.ledgerLanguage) private var language

    var body: some View {
        VStack(spacing: 0) {
            controls
            Divider()
            if let payload = store.pricesPayload {
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 16) {
                        if payload.section == .commodities {
                            Text(language.text("商品来自完整账本历史；目录不表示截至日仍有持仓。", "Commodities come from the full ledger history; this catalogue does not imply current holdings."))
                                .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                            if payload.commodities.isEmpty { Text(language.text("此账本尚无商品记录。", "This ledger has no commodity records.")) }
                            ForEach(payload.commodities) { commodity in commodityRow(commodity, selectable: true) }
                            pager(payload.commodities_page)
                        } else {
                            if let commodity = payload.commodity { commodityRow(commodity, selectable: false) }
                            quotePicker(payload)
                            selectedQuote(payload)
                            Text(language.text("以下只列实际同方向 price 指令；倒数选用不会添加虚构历史。", "The history below contains actual price directives in this direction. Inverse selection does not add source records."))
                                .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                            if payload.records.isEmpty {
                                Text(payload.quote.isEmpty ? language.text("请选择报价币种。", "Select a quote currency.") : language.text("当前期间没有此方向的源报价；截至日选用可能仍有值。", "No source quotes in this direction during the period. The cutoff selection may still be available."))
                            } else {
                                priceChart(payload)
                                ForEach(payload.records) { record in
                                    VStack(alignment: .leading, spacing: 6) {
                                        HStack {
                                            Text(record.date).font(LedgerFonts.font(.body, design: .monospaced))
                                            Spacer()
                                            Text(record.rate.label + " " + record.quote + " / " + record.base)
                                                .font(LedgerFonts.font(.body, design: .monospaced)).help(record.rate.exact)
                                        }.textSelection(.enabled)
                                        Text(language.text("精确值：", "Exact: ") + record.rate.exact).font(LedgerFonts.font(.caption, design: .monospaced)).textSelection(.enabled)
                                        if record.value == nil { Text(language.text("数值超出图表坐标范围，保留精确值并断开曲线。", "Outside the chart coordinate range; the exact value is retained and the line is broken.")).foregroundColor(.secondary) }
                                        SourceActions(source: record.source)
                                    }.padding(12).background(.quaternary.opacity(0.3)).cornerRadius(8)
                                }
                            }
                            pager(payload.records_page)
                        }
                    }.padding(16)
                }
            } else {
                Spacer()
                Text(language.text("等待价格数据。", "Waiting for price data.")).foregroundColor(.secondary)
                Spacer()
            }
        }
    }

    private var controls: some View {
        HStack {
            Button { store.navigateBack() } label: { Label(language.text("返回", "Back"), systemImage: "chevron.left") }
                .disabled(!store.canNavigateBack).keyboardShortcut("[", modifiers: .command)
            if !(store.request.price_base ?? "").isEmpty {
                Button(language.text("商品目录", "Commodity catalogue")) { store.showPrices() }
            }
            Spacer()
            Picker(language.text("排序", "Sort"), selection: Binding(get: { store.request.sort.isEmpty ? "ascending" : store.request.sort }, set: { sort in
                store.updateFilters { $0.sort = sort }
            })) {
                Text(language.text("升序", "Ascending")).tag("ascending")
                Text(language.text("降序", "Descending")).tag("descending")
            }.frame(maxWidth: 180)
        }.padding(.horizontal, 16).padding(.vertical, 10)
    }

    private func commodityRow(_ commodity: PriceCommodity, selectable: Bool) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                if selectable {
                    Button(commodity.currency) { store.showPrices(base: commodity.currency) }.buttonStyle(.link)
                        .font(LedgerFonts.font(.headline))
                } else { Text(commodity.currency).font(LedgerFonts.font(.headline)) }
                Spacer()
                Text(language.text("源报价 \(commodity.price_count) 条", "\(commodity.price_count) source quotes"))
                    .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
            }
            Text(commodity.quote_currencies.isEmpty ? language.text("没有本地报价币种对。", "No local source pairs.") : language.text("本地币种对：", "Local pairs: ") + commodity.quote_currencies.joined(separator: ", "))
                .font(LedgerFonts.font(.caption)).foregroundColor(.secondary).textSelection(.enabled)
            if !commodity.accounts.isEmpty {
                DisclosureGroup(language.text("历史账户使用（\(commodity.accounts.count)）", "Historical account use (\(commodity.accounts.count))")) {
                    ForEach(commodity.accounts, id: \.self) { account in
                        Button(account) { store.showAccount(account) }.buttonStyle(.link)
                    }
                }
            }
            if !commodity.declarations.isEmpty {
                DisclosureGroup(language.text("商品声明（\(commodity.declarations.count)）", "Commodity declarations (\(commodity.declarations.count))")) {
                    ForEach(Array(commodity.declarations.enumerated()), id: \.offset) { _, declaration in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(declaration.date).textSelection(.enabled)
                            SourceActions(source: declaration.source)
                        }.padding(.vertical, 4)
                    }
                }
            }
        }.id(commodity.id).padding(12).background(.quaternary.opacity(0.3)).cornerRadius(8)
    }

    private func quotePicker(_ payload: PricesPayload) -> some View {
        let current = payload.quote
        var choices = payload.commodity?.quote_currencies ?? []
        if !current.isEmpty && !choices.contains(current) { choices.append(current); choices.sort() }
        return HStack {
            Text(payload.base).font(LedgerFonts.font(.headline))
            Picker(language.text("报价币种", "Quote currency"), selection: Binding(get: { store.request.price_quote ?? "" }, set: { quote in
                store.updateFilters { $0.price_quote = quote.isEmpty ? nil : quote }
            })) {
                Text(language.text("尚未选择", "Not selected")).tag("")
                ForEach(choices, id: \.self) { Text($0).tag($0) }
            }.frame(maxWidth: 220)
            Text(language.text("选择只影响此币种对，不改变分析币种。", "This selection affects the pair and preserves the analysis currency."))
                .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
        }
    }

    @ViewBuilder private func selectedQuote(_ payload: PricesPayload) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(language.text("截至日选用", "Quote selected at cutoff")).font(LedgerFonts.font(.headline))
            if let selected = payload.selected {
                Text(selected.base + "/" + selected.quote + " " + selected.rate.label)
                    .font(LedgerFonts.font(.title3, design: .monospaced)).help(selected.rate.exact).textSelection(.enabled)
                Text(language.text("精确值：", "Exact: ") + selected.rate.exact).font(LedgerFonts.font(.caption, design: .monospaced)).textSelection(.enabled)
                Text(language.text("源日期 \(selected.date) · 距截至日 \(selected.age_days) 天", "Source date \(selected.date) · \(selected.age_days) days before cutoff"))
                    .font(LedgerFonts.font(.caption)).textSelection(.enabled)
                if selected.inverse {
                    Text(language.text("使用原始 \(selected.source_base)/\(selected.source_quote) 的倒数。", "Uses the inverse of source \(selected.source_base)/\(selected.source_quote)."))
                    Button(language.text("查看原始方向历史", "View source direction history")) { store.showPrices(base: selected.source_base, quote: selected.source_quote) }
                }
                SourceActions(source: selected.source)
                Text(language.text("报价年龄是阅读信息，未设置自动失效阈值。", "Quote age is informational; no automatic expiry threshold is configured."))
                    .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
            } else {
                Text(payload.selected_reason == .unspecifiedPair ? language.text("尚未选择完整币种对。", "No complete pair selected.") : language.text("截至日没有可用的本地直接或非零倒数报价。", "No usable local direct or nonzero inverse quote through the cutoff."))
                    .foregroundColor(.secondary)
            }
        }.padding(12).frame(maxWidth: .infinity, alignment: .leading).background(.thinMaterial).cornerRadius(8)
    }

    private func priceChart(_ payload: PricesPayload) -> some View {
        let points = payload.records.sorted { $0.date == $1.date ? $0.source_order < $1.source_order : $0.date < $1.date }
        return VStack(alignment: .leading, spacing: 8) {
            Text(language.text("当前页源报价曲线", "Source quotes on this page")).font(LedgerFonts.font(.headline))
            Text(language.text("仅显示本页 \(points.count) 个源记录，完整精确值见下方列表。", "Shows \(points.count) source records on this page; exact values are listed below."))
                .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
            Chart {
                ForEach(points) { record in
                    if let value = record.value {
                        LineMark(x: .value("Date", record.date), y: .value("Rate", value), series: .value("Segment", record.segment))
                            .foregroundStyle(Color.accentColor)
                        PointMark(x: .value("Date", record.date), y: .value("Rate", value)).foregroundStyle(Color.accentColor)
                    }
                }
            }.frame(height: 220)
                .accessibilityLabel(language.text("\(payload.base)/\(payload.quote) 价格曲线，精确值见列表", "\(payload.base)/\(payload.quote) price chart; exact values are in the list"))
        }
    }

    private func pager(_ page: ProjectionPage) -> some View {
        HStack {
            Text(page.returned_count == 0 ? language.text("共 \(page.total_count) 条；当前页为空", "\(page.total_count) records; current page is empty") : language.text("第 \(page.offset + 1)–\(page.offset + page.returned_count) 条，共 \(page.total_count) 条", "\(page.offset + 1)–\(page.offset + page.returned_count) of \(page.total_count) records"))
                .font(LedgerFonts.font(.caption)).textSelection(.enabled)
            Spacer()
            Button(language.text("上一页", "Previous")) { store.pricePage(max(0, page.offset - page.limit), page: page) }.disabled(page.offset == 0)
            Button(language.text("下一页", "Next")) { store.pricePage(page.offset + page.limit, page: page) }.disabled(!page.has_more)
        }
    }
}
