// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import Foundation

// A read-only, single-open private contract probe. Never prints paths, account
// names, amounts, narration or raw errors; this is not a benchmark/soak tool.
@main
struct PrivateReadProbe {
    struct Expected: Decodable {
        let input_identity: String
        let entry_count: Int
        let transaction_count: Int
        let account_count: Int
        let roots: [String: [String: String]]
        let holding_checks: [HoldingCheck]
        let price_catalog: [ExpectedCommodity]
        let price_checks: [PriceCheck]
    }
    struct HoldingCheck: Decodable {
        let date_to: String
        let groups: [ExpectedGroup]
    }
    struct ExpectedGroup: Decodable {
        let account: String
        let commodity: String
        let cost_currency: String?
        let quantity: String
        let cost_basis: String?
        let average_cost: String?
        let lots: [ExpectedLot]
    }
    struct ExpectedLot: Decodable, Equatable {
        let quantity: String
        let cost_number: String?
        let cost_date: String?
        let cost_label: String?
    }
    struct ExpectedCommodity: Decodable {
        let currency: String
        let quote_currencies: [String]
        let accounts: [String]
        let declaration_count: Int
        let price_count: Int
    }
    struct ExpectedSourcePrice: Decodable, Equatable {
        let date: String
        let rate: String
    }
    struct ExpectedSelectedPrice: Decodable {
        let date: String
        let rate: String
        let inverse: Bool
        let source_base: String
        let source_quote: String
        let age_days: Int
    }
    struct PriceCheck: Decodable {
        let base: String
        let quote: String
        let date_to: String
        let records: [ExpectedSourcePrice]
        let selected: ExpectedSelectedPrice?
    }
    static func main() {
        do { try run() }
        catch {
            print(#"{"native_contract":false}"#)
            exit(1)
        }
    }
    static func run() throws {
        guard CommandLine.arguments.count == 3 else { throw BridgeError.remote("probe", "Missing probe inputs") }
        let expected = try JSONDecoder().decode(Expected.self, from: Data(contentsOf: URL(fileURLWithPath: CommandLine.arguments[2])))
        let bridge = BridgeClient.shared
        let opened = try bridge.openLedger(path: CommandLine.arguments[1], semantics: "v3", inputIdentity: expected.input_identity)
        defer { try? bridge.closeLedger(handle: opened.handle) }
        var checkedRoots = 0
        for route in ["balance_sheet", "income_statement"] {
            let result = try bridge.readView(handle: opened.handle, request: ViewRequest(view: route, currency: "native", window: "all"))
            guard result.state.valid, result.state.semantics_profile == "v3", result.summary.entries_count == expected.entry_count, result.summary.transactions_count == expected.transaction_count, result.summary.accounts.count == expected.account_count else { throw BridgeError.remote("probe", "Counters differ") }
            for node in result.tree?.root_nodes ?? [] {
                guard let wanted = expected.roots[node.full_name] else { throw BridgeError.remote("probe", "Unexpected root") }
                for currency in Set(wanted.keys).union(node.subtree_amounts.keys) {
                    guard (wanted[currency] ?? "0") == (node.subtree_amounts[currency]?.exact ?? "0") else { throw BridgeError.remote("probe", "Exact tree amount differs") }
                }
                checkedRoots += 1
            }
        }
        let journal = try bridge.readView(handle: opened.handle, request: ViewRequest(view: "journal", window: "all", limit: 100))
        guard journal.journal?.total_count == expected.transaction_count, journal.journal?.transactions.count == min(100, expected.transaction_count), checkedRoots == expected.roots.count else { throw BridgeError.remote("probe", "Journal/tree coverage differs") }
        let state = try bridge.pollState(handle: opened.handle, after: opened.state.revision)
        guard !state.changed, state.state.build_count == 1 else { throw BridgeError.remote("probe", "Reading rebuilt the ledger") }
        for check in expected.holding_checks {
            try verifyHoldings(check, bridge: bridge, opened: opened)
        }
        try verifyPrices(expected, bridge: bridge, opened: opened)
        let afterHoldings = try bridge.pollState(handle: opened.handle, after: opened.state.revision)
        guard !afterHoldings.changed, afterHoldings.state.build_count == 1 else { throw BridgeError.remote("probe", "Holding reads rebuilt the ledger") }
        print("{\"native_contract\":true,\"entries\":\(expected.entry_count),\"transactions\":\(expected.transaction_count),\"checked_roots\":\(checkedRoots),\"holding_cutoffs\":\(expected.holding_checks.count),\"price_checks\":\(expected.price_checks.count),\"build_count\":1}")
    }

    static func verifyPrices(_ expected: Expected, bridge: BridgeClient, opened: OpenLedgerResult) throws {
        var commodities: [PriceCommodity] = []
        var request = ViewRequest(view: "prices", snapshot_id: opened.state.snapshot_id, limit: 100)
        while true {
            let view = try bridge.readView(handle: opened.handle, request: request)
            guard view.state.valid, view.state.snapshot_id == opened.state.snapshot_id, let payload = view.prices else { throw BridgeError.remote("probe", "Price context differs") }
            commodities.append(contentsOf: payload.commodities)
            if !payload.commodities_page.has_more { break }
            request.offset += payload.commodities_page.returned_count
        }
        guard commodities.count == expected.price_catalog.count else { throw BridgeError.remote("probe", "Price catalog count differs") }
        for wanted in expected.price_catalog {
            guard let item = commodities.first(where: { $0.currency == wanted.currency }), item.accounts == wanted.accounts, item.quote_currencies == wanted.quote_currencies, item.declarations.count == wanted.declaration_count, item.price_count == wanted.price_count else { throw BridgeError.remote("probe", "Price commodity use differs") }
        }
        for check in expected.price_checks {
            var rows: [ExpectedSourcePrice] = []
            var pair = ViewRequest(view: "prices", snapshot_id: opened.state.snapshot_id, date_to: check.date_to, price_base: check.base, price_quote: check.quote, limit: 100)
            while true {
                let response = try bridge.readView(handle: opened.handle, request: pair)
                guard response.state.snapshot_id == opened.state.snapshot_id, let payload = response.prices else { throw BridgeError.remote("probe", "Price history context differs") }
                if let desired = check.selected {
                    guard let chosen = payload.selected, chosen.date == desired.date, chosen.rate.exact == desired.rate, chosen.inverse == desired.inverse, chosen.source_base == desired.source_base, chosen.source_quote == desired.source_quote, chosen.age_days == desired.age_days else { throw BridgeError.remote("probe", "Price cutoff selection differs") }
                } else if payload.selected != nil || payload.selected_reason != .missingLocalQuote { throw BridgeError.remote("probe", "Missing price differs") }
                rows.append(contentsOf: payload.records.map { ExpectedSourcePrice(date: $0.date, rate: $0.rate.exact) })
                if !payload.records_page.has_more { break }
                pair.offset += payload.records_page.returned_count
            }
            var remaining = check.records
            for row in rows {
                guard let index = remaining.firstIndex(of: row) else { throw BridgeError.remote("probe", "Source price multiplicity differs") }
                remaining.remove(at: index)
            }
            guard remaining.isEmpty else { throw BridgeError.remote("probe", "Price history coverage differs") }
        }
    }

    static func verifyHoldings(_ expected: HoldingCheck, bridge: BridgeClient, opened: OpenLedgerResult) throws {
        var groups: [HoldingGroup] = []
        var request = ViewRequest(view: "holdings", currency: "native", snapshot_id: opened.state.snapshot_id, date_to: expected.date_to, group_by: "account", holding_section: "groups", limit: 100)
        while true {
            let response = try bridge.readView(handle: opened.handle, request: request)
            guard response.state.valid, response.state.snapshot_id == opened.state.snapshot_id, let payload = response.holdings else { throw BridgeError.remote("probe", "Holding context differs") }
            groups.append(contentsOf: payload.groups)
            if !payload.groups_page.has_more { break }
            request.offset += payload.groups_page.returned_count
        }
        guard groups.count == expected.groups.count else { throw BridgeError.remote("probe", "Holding group count differs") }
        for wanted in expected.groups {
            let matches = groups.filter { $0.account == wanted.account && $0.commodity == wanted.commodity && $0.cost_currency == wanted.cost_currency }
            guard matches.count == 1, let group = matches.first, group.quantity.exact == wanted.quantity, group.cost_basis?.exact == wanted.cost_basis, group.average_cost?.exact == wanted.average_cost else { throw BridgeError.remote("probe", "Holding math differs") }
            var lots = group.lots
            var offset = group.lots_page.returned_count
            var hasMore = group.lots_page.has_more
            while hasMore {
                var detail = request
                detail.holding_section = "lots"
                detail.group_id = group.id
                detail.offset = offset
                let response = try bridge.readView(handle: opened.handle, request: detail)
                guard response.state.snapshot_id == opened.state.snapshot_id, let page = response.holdings?.groups.first else { throw BridgeError.remote("probe", "Holding lot page differs") }
                lots.append(contentsOf: page.lots)
                offset += page.lots_page.returned_count
                hasMore = page.lots_page.has_more
            }
            var remaining = wanted.lots
            for lot in lots {
                let value = ExpectedLot(quantity: lot.quantity.exact, cost_number: lot.cost_number?.exact, cost_date: lot.cost_date, cost_label: lot.cost_label)
                guard let index = remaining.firstIndex(of: value) else { throw BridgeError.remote("probe", "Full holding lot identity differs") }
                remaining.remove(at: index)
            }
            guard remaining.isEmpty else { throw BridgeError.remote("probe", "Holding lot coverage differs") }
        }
    }
}
