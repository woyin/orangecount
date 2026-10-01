// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import Foundation
import OrangeCountBridge

@main
struct SessionBenchmark {
    static func main() throws {
        let client = BridgeClient.shared
        let started = Date()
        let profile = CommandLine.arguments.count > 2 ? CommandLine.arguments[2] : "v3"
        let opened = try client.openLedger(path: CommandLine.arguments[1], semantics: profile)
        print("semantics_profile=\(profile) evaluation_revision=\(opened.state.evaluation_revision)")
        let openTime = Date().timeIntervalSince(started)
        defer { try? client.closeLedger(handle: opened.handle) }
        var samples: [String: [Double]] = [:]
        var projectionSamples: [String: [Double]] = [:]
        var decodeSamples: [String: [Double]] = [:]
        var byteCounts: [String: Int] = [:]
        var holdingGroupID: String?
        var holdingEventID: String?
        for iteration in 0..<20 {
            for view in ["balance_sheet", "income_statement", "journal", "dashboard", "account", "holdings_groups", "holdings_events", "holdings_history", "holdings_lots", "holdings_history_points", "holdings_consumed", "prices_catalog", "prices_history"] {
                var selected = ViewRequest(view: view, currency: view == "dashboard" || view == "journal" ? "" : "native", account: view == "account" ? "Assets:Broker" : "", scope: view == "account" ? "subtree" : nil)
                if view.hasPrefix("holdings_") {
                    selected.view = "holdings"
                    selected.account = "Assets:Broker:AAPL"
                    selected.holding_section = String(view.dropFirst("holdings_".count))
                    if view == "holdings_lots" || view == "holdings_history_points" {
                        guard let holdingGroupID else { throw BridgeError.remote("benchmark", "Synthetic holdings have no group") }
                        selected.holding_section = view == "holdings_lots" ? "lots" : "history"
                        selected.group_id = holdingGroupID
                        selected.snapshot_id = opened.state.snapshot_id
                    } else if view == "holdings_consumed" {
                        guard let holdingEventID else { throw BridgeError.remote("benchmark", "Synthetic holdings have no reduction") }
                        selected.holding_section = "lots"
                        selected.event_id = holdingEventID
                        selected.lot_kind = "consumed"
                        selected.snapshot_id = opened.state.snapshot_id
                    }
                }
                if view.hasPrefix("prices_") {
                    selected.view = "prices"
                    selected.price_section = view == "prices_catalog" ? "commodities" : "history"
                    if view == "prices_history" { selected.price_base = "AAPL"; selected.price_quote = "USD" }
                }
                let request = String(decoding: try JSONEncoder().encode(selected), as: UTF8.self)
                let start = Date()
                guard let pointer = request.withCString({ OC_ReadView(opened.handle, UnsafeMutablePointer(mutating: $0)) }) else { throw BridgeError.nullPointerFromGo }
                let projected = Date()
                let bytes = Data(bytes: pointer, count: strlen(pointer))
                OC_FreeString(pointer)
                let response = try BridgeClient.decodeResponse(bytes, as: LedgerView.self)
                byteCounts[view] = bytes.count
                projectionSamples[view, default: []].append(projected.timeIntervalSince(start) * 1000)
                decodeSamples[view, default: []].append(Date().timeIntervalSince(projected) * 1000)
                guard response.state.valid else { throw BridgeError.remote("benchmark", "Synthetic fixture is invalid") }
                if view == "holdings_groups" {
                    holdingGroupID = response.holdings?.groups.first?.id
                    holdingEventID = response.holdings?.events.first?.id
                    if iteration == 0, let payload = response.holdings {
                        print("holdings groups=\(payload.groups_page.total_count) events=\(payload.events_page.total_count) series=\(payload.history_page.total_count) lots=\(payload.groups.first?.lots_page.total_count ?? 0) history_points=\(payload.average_history.first?.points_page.total_count ?? 0)")
                    }
                }
                if iteration == 0, let prices = response.prices {
                    print("prices section=\(prices.section.rawValue) commodities=\(prices.commodities_page.total_count) source_records=\(prices.records_page.total_count) returned=\(prices.records.count)")
                }
                samples[view, default: []].append(Date().timeIntervalSince(start) * 1000)
                if iteration == 0 && view == "balance_sheet" {
                    print("open_and_first_tree_ms=\(Int((openTime + Date().timeIntervalSince(start)) * 1000))")
                }
                if iteration == 0, let trends = response.trends {
                    print("dashboard net_worth=\(trends.net_worth_availability ?? "missing") currency_series=\(trends.currency_series?.count ?? 0) points=\(trends.currency_series?.reduce(0) { $0 + $1.net_worth_points.count } ?? 0) cash_flow=\(trends.cash_flow_availability ?? "missing") months=\(trends.currency_series?.reduce(0) { $0 + $1.monthly_cash_flows.count } ?? 0)")
                }
            }
        }
        for view in samples.keys.sorted() {
            let ordered = samples[view]!.sorted()
            print("\(view) json_bytes=\(byteCounts[view]!) go_projection_and_json_p95_ms=\(Int(projectionSamples[view]!.sorted()[18])) swift_copy_decode_p95_ms=\(Int(decodeSamples[view]!.sorted()[18]))")
            print("\(view) samples=\(ordered.count) first_ms=\(Int(samples[view]![0])) p95_ms=\(Int(ordered[18]))")
            print("\(view) read_samples_ms=\(samples[view]!)")
            print("\(view) projection_samples_ms=\(projectionSamples[view]!) decode_samples_ms=\(decodeSamples[view]!)")
        }
        let filteredRequests: [(String, ViewRequest)] = [
            ("journal_text", ViewRequest(view: "journal", text: "payee 25")),
            ("journal_account", ViewRequest(view: "journal", account: "Expenses")),
            ("journal_period", ViewRequest(view: "journal", date_from: "2010-01-01", date_to: "2010-03-31")),
            ("journal_combined", ViewRequest(view: "journal", date_from: "2010-01-01", date_to: "2010-12-31", account: "Expenses", text: "payee 25", sort: "ascending")),
            ("journal_second_page", ViewRequest(view: "journal", snapshot_id: opened.state.snapshot_id, offset: 100, limit: 100)),
            ("income_period", ViewRequest(view: "income_statement", currency: "native", date_from: "2010-01-01", date_to: "2010-03-31")),
            ("dashboard_period", ViewRequest(view: "dashboard", date_from: "2010-01-01", date_to: "2010-03-31"))
        ]
        for (name, request) in filteredRequests {
            var timings: [Double] = []
            var count: Int?
            for _ in 0..<20 {
                let start = Date()
                let response = try client.readView(handle: opened.handle, request: request)
                timings.append(Date().timeIntervalSince(start) * 1000)
                guard response.state.snapshot_id == opened.state.snapshot_id, response.state.build_count == 1 else {
                    throw BridgeError.remote("benchmark", "A filtered read changed or rebuilt the snapshot")
                }
                count = response.journal?.total_count
            }
            print("filter=\(name) samples=20 p95_ms=\(Int(timings.sorted()[18])) matching_transactions=\(count.map(String.init) ?? "n/a")")
            print("filter=\(name) read_samples_ms=\(timings)")
        }
        let pollStart = Date()
        for _ in 0..<1000 { _ = try client.pollState(handle: opened.handle, after: opened.state.revision) }
        print("poll_mean_ms=\(Date().timeIntervalSince(pollStart))")
        let state = try client.pollState(handle: opened.handle, after: 0).state
        guard state.build_count == 1 else { throw BridgeError.remote("benchmark", "Reads rebuilt the ledger") }
        print("build_count=\(state.build_count)")
        var reloads: [Double] = []
        for _ in 0..<5 {
            let start = Date()
            let refreshed = try client.reloadLedger(handle: opened.handle)
            guard refreshed.valid else { throw BridgeError.remote("benchmark", "Manual reload invalidated the fixture") }
            reloads.append(Date().timeIntervalSince(start) * 1000)
        }
        print("manual_reload_samples_ms=\(reloads.map { Int($0) })")
    }
}
