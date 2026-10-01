import SwiftUI
import Charts

final class ChartHoverState: ObservableObject {
    @Published var month: String?
}

struct DashboardView: View {
    let trends: HistoricalTrendsPayload?
    let summary: LedgerSummary?
    @Environment(\.ledgerLanguage) private var language

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                HStack(spacing: 16) {
                    MetricCard(title: language.text("账户数（含已关闭）", "Accounts (including closed)"), value: "\(summary?.accounts.count ?? 0)", icon: "folder.fill", color: .blue)
                    MetricCard(title: language.text("交易笔数（全账本）", "Transactions (whole ledger)"), value: "\(summary?.transactions_count ?? 0)", icon: "doc.text.fill", color: .orange)
                }
                Text(language.text("净资产按月末或所选截至日计算，包含资产与负债；不同原币种分别展示。", "Net worth includes assets and liabilities at each month end or selected cutoff. Native currencies are shown separately."))
                    .font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                if let series = trends?.currency_series, !series.isEmpty {
                    ForEach(series) { CurrencyDashboardPanel(series: $0) }
                } else {
                    EmptyChartView(message: language.text("所选期间没有可绘制记录", "No chart records in the selected range"))
                }
            }.padding(20)
        }
    }
}

struct CurrencyDashboardPanel: View {
    let series: CurrencyHistorySeries
    @Environment(\.ledgerLanguage) private var language
    @StateObject private var hover = ChartHoverState()

    private func amount(_ value: DisplayAmount?) -> String {
        guard let value else { return "—" }
        return value.label + " " + series.currency
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack {
                Text(series.currency).font(LedgerFonts.font(.title2)).bold()
                Text(series.mode == "native" ? language.text("原币", "Native") : language.text("折算", "Converted")).foregroundColor(.secondary)
                Text(valuationLabel).foregroundColor(.secondary)
            }
            LazyVGrid(columns: [GridItem(.adaptive(minimum: 210))], spacing: 12) {
                MetricCard(title: language.text("截至日净资产", "Net worth through end date"), value: amount(series.net_worth_points.last?.valuation.complete_total), icon: "banknote.fill", color: .green)
                MetricCard(title: language.text("期间收入", "Period income"), value: amount(series.income_total), icon: "arrow.down.left", color: .green)
                MetricCard(title: language.text("期间支出", "Period expenses"), value: amount(series.expenses_total), icon: "arrow.up.right", color: .orange)
                MetricCard(title: language.text("期间结余", "Period net"), value: amount(series.net_total), icon: "equal", color: .blue)
            }
            VStack(alignment: .leading, spacing: 10) {
                Label(language.text("净资产趋势", "Net worth trend"), systemImage: "chart.line.uptrend.xyaxis").font(LedgerFonts.font(.headline))
                if series.net_worth_points.contains(where: { $0.value != nil }) {
                    Chart(series.net_worth_points) { point in
                        if let value = point.value, let exact = point.valuation.complete_total {
                            LineMark(x: .value(language.text("日期", "Date"), point.date), y: .value(language.text("净资产", "Net worth"), value), series: .value("Segment", point.segment))
                                .foregroundStyle(Color.accentColor).accessibilityHidden(true)
                            PointMark(x: .value(language.text("日期", "Date"), point.date), y: .value(language.text("净资产", "Net worth"), value))
                                .foregroundStyle(Color.accentColor)
                                .accessibilityLabel(point.date).accessibilityValue(exact.exact + " " + series.currency)
                        }
                    }.frame(height: 210).chartOverlay { proxy in hoverOverlay(proxy) }
                } else { EmptyChartView(message: language.text("没有可完整估值的净资产点", "No completely valued net worth points")) }
                if let point = series.net_worth_points.first(where: { $0.date == hover.month }) ?? series.net_worth_points.last {
                    Text(point.date + " · " + amount(point.valuation.complete_total)).font(.caption.monospaced())
                        .help(quoteDetails(point.valuation))
                }
                ForEach(series.net_worth_points.filter { $0.valuation.complete_total == nil }) { point in
                    Text(point.date + " · " + language.text("缺报价：", "Missing quotes: ") + point.valuation.missing.map { $0.commodity }.joined(separator: ", "))
                        .font(LedgerFonts.font(.caption)).foregroundColor(.orange)
                }
                if series.net_worth_availability == "partial" {
                    Text(language.text("缺价日期保留为空值，曲线不跨缺口连接。", "Missing dates remain unavailable; lines do not bridge gaps."))
                        .font(LedgerFonts.font(.caption)).foregroundColor(.orange)
                }
            }.padding().background(.regularMaterial, in: RoundedRectangle(cornerRadius: 10))
            VStack(alignment: .leading, spacing: 10) {
                Label(language.text("月度收入与支出", "Monthly income and expenses"), systemImage: "chart.bar.xaxis").font(LedgerFonts.font(.headline))
                if series.monthly_cash_flows.contains(where: { $0.income_value != nil || $0.expenses_value != nil }) {
                    Chart {
                        ForEach(series.monthly_cash_flows) { flow in
                            if let value = flow.income_value, let exact = flow.income.complete_total {
                                BarMark(x: .value(language.text("月份", "Month"), flow.month), y: .value(language.text("金额", "Amount"), value))
                                    .foregroundStyle(by: .value(language.text("类型", "Type"), language.text("收入", "Income"))).position(by: .value(language.text("类型", "Type"), language.text("收入", "Income")))
                                    .accessibilityLabel(flow.month + " " + language.text("收入", "Income")).accessibilityValue(exact.exact + " " + series.currency)
                            }
                            if let value = flow.expenses_value, let exact = flow.expenses.complete_total {
                                BarMark(x: .value(language.text("月份", "Month"), flow.month), y: .value(language.text("金额", "Amount"), value))
                                    .foregroundStyle(by: .value(language.text("类型", "Type"), language.text("支出", "Expenses"))).position(by: .value(language.text("类型", "Type"), language.text("支出", "Expenses")))
                                    .accessibilityLabel(flow.month + " " + language.text("支出", "Expenses")).accessibilityValue(exact.exact + " " + series.currency)
                            }
                        }
                    }.chartForegroundStyleScale([language.text("收入", "Income"): Color.green, language.text("支出", "Expenses"): Color.orange])
                        .frame(height: 210).chartOverlay { proxy in hoverOverlay(proxy) }
                    if let flow = series.monthly_cash_flows.first(where: { $0.month == hover.month }) ?? series.monthly_cash_flows.last {
                        Text(flow.month + " · " + language.text("收入 ", "Income ") + amount(flow.income.complete_total) + " · " + language.text("支出 ", "Expenses ") + amount(flow.expenses.complete_total) + " · " + language.text("结余 ", "Net ") + amount(flow.net.complete_total))
                            .font(.caption.monospaced()).textSelection(.enabled)
                    }
                } else { EmptyChartView(message: language.text("期间没有可绘制收支", "No drawable income or expenses in the period")) }
                ForEach(series.monthly_cash_flows.filter { $0.income.complete_total == nil || $0.expenses.complete_total == nil }) { flow in
                    Text(flow.month + " · " + language.text("部分收支缺少报价；期间总计不可用。", "Some flows lack quotes; period totals are unavailable."))
                        .font(LedgerFonts.font(.caption)).foregroundColor(.orange)
                }
            }.padding().background(.regularMaterial, in: RoundedRectangle(cornerRadius: 10))
        }
    }

    private var valuationLabel: String {
        switch series.valuation {
        case "units": return language.text("数量", "Units")
        case "market-value": return language.text("市值", "Market value")
        default: return language.text("成本", "At cost")
        }
    }
    private func quoteDetails(_ valuation: AccountValuation) -> String {
        (valuation.complete_total?.exact ?? "—") + "\n" + valuation.quotes.map { $0.base + "/" + $0.quote + " " + $0.rate.exact + " @ " + $0.date }.joined(separator: "\n")
    }
    private func hoverOverlay(_ proxy: ChartProxy) -> some View {
        GeometryReader { geometry in
            Rectangle().fill(Color.clear).contentShape(Rectangle()).onContinuousHover { phase in
                switch phase {
                case .active(let location):
                    let x = location.x - geometry[proxy.plotAreaFrame].origin.x
                    hover.month = proxy.value(atX: x, as: String.self)
                case .ended: hover.month = nil
                }
            }
        }
    }
}

struct MetricCard: View {
    let title: String
    let value: String
    let icon: String
    let color: Color
    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: icon).font(LedgerFonts.font(.title2)).foregroundColor(color).frame(width: 36, height: 36)
            VStack(alignment: .leading, spacing: 3) {
                Text(title).font(LedgerFonts.font(.caption)).foregroundColor(.secondary)
                // Recreate selectable text so its native accessibility value
                // follows the displayed amount after a reload.
                Text(value).font(LedgerFonts.font(.title3)).bold().textSelection(.enabled).id(value)
            }
            Spacer(minLength: 0)
        }.padding(14).frame(maxWidth: .infinity, alignment: .leading).background(.regularMaterial, in: RoundedRectangle(cornerRadius: 10))
    }
}

struct EmptyChartView: View {
    let message: String
    var body: some View {
        VStack(spacing: 10) {
            Image(systemName: "chart.xyaxis.line").font(LedgerFonts.font(.largeTitle))
            Text(message).font(LedgerFonts.font(.callout))
        }.foregroundColor(.secondary).padding(36).frame(maxWidth: .infinity)
    }
}
