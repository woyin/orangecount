import Charts
import SwiftUI

struct DashboardView: View {
    let trends: HistoricalTrendsPayload?
    let summary: LedgerSummary?

    var latestNetWorth: Double {
        trends?.net_worth_points.last?.value ?? 0.0
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                // KPI Summary Row
                HStack(spacing: 16) {
                    MetricCard(
                        title: "当前净资产 (Net Worth)",
                        value: String(format: "%.2f %@", latestNetWorth, trends?.operating_currency ?? "USD"),
                        icon: "banknote.fill",
                        color: .green
                    )

                    MetricCard(
                        title: "有效账户总数",
                        value: "\(summary?.accounts.count ?? 0)",
                        icon: "folder.fill",
                        color: .blue
                    )

                    MetricCard(
                        title: "账本交易记录",
                        value: "\(summary?.entries_count ?? 0) 笔",
                        icon: "doc.text.fill",
                        color: .orange
                    )
                }

                // Net Worth Line Chart
                VStack(alignment: .leading, spacing: 10) {
                    HStack {
                        Label("资产净值历史趋势 (Net Worth Trend)", systemImage: "chart.line.uptrend.xyaxis")
                            .font(.headline)
                        Spacer()
                        Text("单位: \(trends?.operating_currency ?? "USD")")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }

                    if let points = trends?.net_worth_points, !points.isEmpty {
                        Chart(points) { pt in
                            AreaMark(
                                x: .value("日期", pt.date),
                                y: .value("净资产", pt.value)
                            )
                            .foregroundStyle(
                                LinearGradient(
                                    colors: [Color.accentColor.opacity(0.35), Color.accentColor.opacity(0.02)],
                                    startPoint: .top,
                                    endPoint: .bottom
                                )
                            )

                            LineMark(
                                x: .value("日期", pt.date),
                                y: .value("净资产", pt.value)
                            )
                            .foregroundStyle(Color.accentColor)
                            .lineStyle(StrokeStyle(lineWidth: 2.5))

                            PointMark(
                                x: .value("日期", pt.date),
                                y: .value("净资产", pt.value)
                            )
                            .foregroundStyle(Color.accentColor)
                        }
                        .frame(height: 220)
                        .padding(.vertical, 8)
                    } else {
                        EmptyChartView(message: "当前账本时间点数据不足以生成走势折线")
                    }
                }
                .padding()
                .background(RoundedRectangle(cornerRadius: 10).fill(.regularMaterial))

                // Monthly Cash Flows Bar Chart
                VStack(alignment: .leading, spacing: 10) {
                    HStack {
                        Label("月度收支走势 (Monthly Income & Expenses)", systemImage: "chart.bar.xaxis")
                            .font(.headline)
                        Spacer()
                        HStack(spacing: 12) {
                            HStack(spacing: 4) {
                                Circle().fill(Color.green).frame(width: 8, height: 8)
                                Text("收入").font(.caption).foregroundColor(.secondary)
                            }
                            HStack(spacing: 4) {
                                Circle().fill(Color.orange).frame(width: 8, height: 8)
                                Text("支出").font(.caption).foregroundColor(.secondary)
                            }
                        }
                    }

                    if let flows = trends?.monthly_cash_flows, !flows.isEmpty {
                        Chart {
                            ForEach(flows) { flow in
                                BarMark(
                                    x: .value("月份", flow.month),
                                    y: .value("金额", flow.income)
                                )
                                .foregroundStyle(Color.green.opacity(0.85))
                                .position(by: .value("类型", "收入"))

                                BarMark(
                                    x: .value("月份", flow.month),
                                    y: .value("金额", flow.expenses)
                                )
                                .foregroundStyle(Color.orange.opacity(0.85))
                                .position(by: .value("类型", "支出"))
                            }
                        }
                        .frame(height: 220)
                        .padding(.vertical, 8)
                    } else {
                        EmptyChartView(message: "当前账本无月度收支分录对比数据")
                    }
                }
                .padding()
                .background(RoundedRectangle(cornerRadius: 10).fill(.regularMaterial))
            }
            .padding(20)
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
            Image(systemName: icon)
                .font(.system(size: 26))
                .foregroundColor(color)
                .frame(width: 44, height: 44)
                .background(color.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: 8))

            VStack(alignment: .leading, spacing: 3) {
                Text(title)
                    .font(.caption)
                    .foregroundColor(.secondary)
                Text(value)
                    .font(.title3)
                    .bold()
            }
            Spacer()
        }
        .padding(14)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(RoundedRectangle(cornerRadius: 10).fill(.regularMaterial))
    }
}

struct EmptyChartView: View {
    let message: String

    var body: some View {
        HStack {
            Spacer()
            VStack(spacing: 8) {
                Image(systemName: "chart.xyaxis.line")
                    .font(.system(size: 32))
                    .foregroundColor(.secondary.opacity(0.5))
                Text(message)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            .padding(40)
            Spacer()
        }
    }
}
