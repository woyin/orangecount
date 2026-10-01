// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0.

import Foundation

struct TestFailure: Error, CustomStringConvertible {
    let description: String
}

func expect(_ condition: @autoclosure () -> Bool, _ message: String) throws {
    if !condition() { throw TestFailure(description: message) }
}

@main
struct ContractTests {
    static func main() {
        do { try run() }
        catch {
            print("FAIL: \(error)")
            exit(1)
        }
    }

    static func run() throws {
        do {
            let incompatible = Data(#"{"protocol_version":999,"data":"incompatible payload"}"#.utf8)
            let _: LedgerView = try BridgeClient.decodeResponse(incompatible, as: LedgerView.self)
            throw TestFailure(description: "Incompatible protocol must be rejected before payload decoding")
        } catch BridgeError.protocolMismatch(let version) {
            try expect(version == 999, "Protocol mismatch must retain the received version")
        }
        print("PASS: protocol mismatch is distinguishable from malformed ledger data")
        for raw in [
            #"{"protocol_version":2,"data":{"mode":"converted","target_currency":"USD","as_of":"2000-01-02","completeness":"partial","covered_item_count":1,"missing_item_count":1,"missing":[{"commodity":"EUR","reason":"missing-local-quote"}],"quotes":[],"valued_portion":{"exact":"100","display":"100","approximate":false}}}"#,
            #"{"protocol_version":2,"data":{"mode":"converted","target_currency":"USD","as_of":"2000-01-02","completeness":"partial","covered_item_count":1,"missing_item_count":1,"missing":[{"commodity":"EUR","reason":"missing-local-quote"}],"quotes":[],"valued_portion":{"exact":"100","display":"100","approximate":false},"complete_total":{"exact":"100","display":"100","approximate":false}}}"#
        ] {
            do {
                let _: AccountValuation = try BridgeClient.decodeResponse(Data(raw.utf8), as: AccountValuation.self)
                throw TestFailure(description: "Missing null field or partial complete total must be rejected")
            } catch BridgeError.jsonDecodeError { }
        }
        print("PASS: required nullable fields and valuation invariants reject malformed payloads")

        let root = CommandLine.arguments[1]
        let bridge = BridgeClient.shared
        let priceSession = try bridge.openLedger(path: root + "/testdata/fixtures/native/prices.bean", semantics: "v3")
        defer { try? bridge.closeLedger(handle: priceSession.handle) }
        let priceView = try bridge.readView(handle: priceSession.handle, request: ViewRequest(view: "prices", date_from: "2000-01-05", date_to: "2000-01-08", price_base: "STK", price_quote: "USD"))
        try expect(priceView.prices?.selected?.rate.exact == "11" && priceView.prices?.records.isEmpty == true && priceView.prices?.selected?.age_days == 4, "Price history dates must not discard earlier as-of quotes through real Swift/C")
        let priceCatalog = try bridge.readView(handle: priceSession.handle, request: ViewRequest(view: "prices", date_to: "2000-01-08")).prices!
        try expect(priceCatalog.commodities.count == 10 && priceCatalog.commodities.first { $0.currency == "STK" }?.price_count == 8 && priceCatalog.selected_reason == .unspecifiedPair, "Source commodity catalog retains cleared, unquoted, declaration-only and quote-only use")
        let priceHistory = try bridge.readView(handle: priceSession.handle, request: ViewRequest(view: "prices", date_to: "2000-01-08", price_base: "STK", price_quote: "USD")).prices!
        try expect(priceHistory.records.count == 3 && priceHistory.records[0].source.path.hasSuffix("prices-included.bean") && priceHistory.records[0].source.line == 2 && (priceHistory.records[0].source.end_column ?? 0) > priceHistory.records[0].source.column, "Every actual source price and full include location must survive")
        let inversePrice = try bridge.readView(handle: priceSession.handle, request: ViewRequest(view: "prices", currency: "USD", date_to: "2000-01-08", price_base: "STK", price_quote: "EUR")).prices!
        try expect(inversePrice.records.isEmpty && inversePrice.selected?.rate.exact == "0.5" && inversePrice.selected?.inverse == true && inversePrice.selected?.source_base == "EUR" && inversePrice.selected?.source_quote == "STK", "Selected inverse is exact and distinct from source history")
        let zeroPrice = try bridge.readView(handle: priceSession.handle, request: ViewRequest(view: "prices", date_to: "2000-01-08", price_base: "ZERO", price_quote: "USD")).prices!
        try expect(zeroPrice.selected?.rate.exact == "0" && zeroPrice.records.first?.value == 0 && zeroPrice.selected_reason == .known, "Direct zero quote must not become missing")
        for (base, quote, cutoff) in [("STK", "GBP", "2000-01-08"), ("STK", "AUD", "2000-01-08"), ("STK", "USD", "2000-01-03"), ("UNQUOTED", "USD", "2000-01-08")] {
            let missingPrice = try bridge.readView(handle: priceSession.handle, request: ViewRequest(view: "prices", date_to: cutoff, price_base: base, price_quote: quote)).prices!
            try expect(missingPrice.selected == nil && missingPrice.selected_reason == .missingLocalQuote && missingPrice.commodity?.currency == base, "Zero inverse, implicit hops, future-only and unquoted must remain explicit missing quotes")
        }
        do {
            _ = try bridge.readView(handle: priceSession.handle, request: ViewRequest(view: "prices", price_base: "UNKNOWN"))
            throw TestFailure(description: "Unknown price commodity must reject")
        } catch BridgeError.remote(let code, _) { try expect(code == "commodity-not-found", "Unknown commodity is a target error, not missing price") }
        let priceObject = try JSONSerialization.jsonObject(with: JSONEncoder().encode(priceHistory)) as! [String: Any]
        for mutation in 0..<10 {
            var damaged = priceObject
            switch mutation {
            case 0: damaged.removeValue(forKey: "selected")
            case 1: damaged.removeValue(forKey: "commodity")
            case 2: damaged["selected_reason"] = "unknown-reason"
            case 3: damaged["selected"] = NSNull()
            case 4:
                var selection = damaged["selected"] as! [String: Any]; selection["age_days"] = -1; damaged["selected"] = selection
            case 5:
                var selection = damaged["selected"] as! [String: Any]; selection["source_base"] = "EUR"; damaged["selected"] = selection
            case 6:
                var rows = damaged["records"] as! [[String: Any]]; rows[0].removeValue(forKey: "value"); damaged["records"] = rows
            case 7:
                var rows = damaged["records"] as! [[String: Any]]; rows[0]["date"] = "2000-01-20"; damaged["records"] = rows
            case 8:
                var rows = damaged["records"] as! [[String: Any]]; rows[1]["id"] = rows[0]["id"]; damaged["records"] = rows
            default:
                var page = damaged["records_page"] as! [String: Any]; page["returned_count"] = 2; damaged["records_page"] = page
            }
            let envelope = try JSONSerialization.data(withJSONObject: ["protocol_version": 2, "data": damaged])
            do {
                let _: PricesPayload = try BridgeClient.decodeResponse(envelope, as: PricesPayload.self)
                throw TestFailure(description: "Malformed price contract accepted mutation \(mutation)")
            } catch BridgeError.jsonDecodeError { }
        }
        print("PASS: price catalog/history, exact cutoff selection, inverse/zero/missing and strict nullable contracts cross real Swift/C")
        let holdingSession = try bridge.openLedger(path: root + "/testdata/fixtures/native/average-history.bean", semantics: "v3")
        defer { try? bridge.closeLedger(handle: holdingSession.handle) }
        let holdings = try bridge.readView(handle: holdingSession.handle, request: ViewRequest(view: "holdings", date_to: "2000-01-06", account: "Assets:Stock")).holdings!
        try expect(holdings.groups.first?.quantity.exact == "18" && holdings.groups.first?.cost_basis?.exact == "440" && holdings.groups.first?.average_cost?.exact == "220/9", "Historical holdings must preserve exact rational cost through real Swift/C")
        try expect(holdings.events.last?.gross_gain?.exact == "920/3" && holdings.events.last?.net_gain?.exact == "920/3", "Only actual reductions produce exact realized gain")
        try expect(holdings.events.first?.merge_lots.count == 2 && holdings.events.first?.merge_lots.first?.origin == .posting && holdings.events.first?.lots.first?.origin == .averageMerge, "AVERAGE must expose whole pre-merge pool separately from the consumed merged lot")
        let mergedPage = try bridge.readView(handle: holdingSession.handle, request: ViewRequest(view: "holdings", snapshot_id: holdingSession.state.snapshot_id, date_to: "2000-01-06", account: "Assets:Stock", holding_section: "lots", event_id: holdings.events[0].id, lot_kind: "merged", offset: 1, limit: 1)).holdings!
        try expect(mergedPage.events[0].merge_lots_page.total_count == 2 && mergedPage.events[0].merge_lots.count == 1 && mergedPage.events[0].cost_basis?.exact == "500/3", "Contribution pagination must never become additional consumed cost")
        let selectedLots = try bridge.readView(handle: holdingSession.handle, request: ViewRequest(view: "holdings", snapshot_id: holdingSession.state.snapshot_id, date_to: "2000-01-06", account: "Assets:Stock", holding_section: "lots", group_id: holdings.groups[0].id, limit: 1)).holdings!
        try expect(selectedLots.groups[0].lots_page.total_count == 1 && selectedLots.groups[0].cost_basis?.exact == "440", "Lot selector must stay snapshot-bound without changing full group cost")
        let encodedHolding = try JSONEncoder().encode(holdings)
        let validHolding = try JSONSerialization.jsonObject(with: encodedHolding) as! [String: Any]
        for mutation in 0..<3 {
            var broken = validHolding
            var groups = broken["groups"] as! [[String: Any]]
            if mutation == 0 { groups[0].removeValue(forKey: "unrealized_gain") }
            if mutation == 1 { groups[0]["inventory_kind"] = "unsupported-kind" }
            if mutation == 2 { groups[0]["cost_currency"] = NSNull() }
            broken["groups"] = groups
            let envelope = try JSONSerialization.data(withJSONObject: ["protocol_version": 2, "data": broken])
            do {
                let _: HoldingsPayload = try BridgeClient.decodeResponse(envelope, as: HoldingsPayload.self)
                throw TestFailure(description: "Required nullable keys, inventory kind and cost invariants must reject malformed holdings")
            } catch BridgeError.jsonDecodeError { }
        }
        print("PASS: historical holdings, source lots, exact AVERAGE gains and malformed contracts cross real Swift/C")
        let identitySession = try bridge.openLedger(path: root + "/testdata/fixtures/native/holdings-identities.bean", semantics: "v3")
        defer { try? bridge.closeLedger(handle: identitySession.handle) }
        let identityHoldings = try bridge.readView(handle: identitySession.handle, request: ViewRequest(view: "holdings")).holdings!
        let zeroCost = identityHoldings.groups.first { $0.account == "Assets:Zero" }!
        let plainPosition = identityHoldings.groups.first { $0.account == "Assets:Plain" }!
        let labeledLots = identityHoldings.groups.first { $0.account == "Assets:Tagged" }!.lots
        try expect(zeroCost.average_cost?.exact == "0" && plainPosition.average_cost == nil && labeledLots.contains { $0.cost_label == nil } && labeledLots.contains { $0.cost_label == "" }, "Zero, absent cost and absent/empty labels must remain distinct through Swift/C")
        let unknownSession = try bridge.openLedger(path: root + "/testdata/fixtures/native/holdings-unknowns.bean", semantics: "v3")
        defer { try? bridge.closeLedger(handle: unknownSession.handle) }
        let unknownEvents = try bridge.readView(handle: unknownSession.handle, request: ViewRequest(view: "holdings", account: "Assets:Stock")).holdings!.events
        try expect(unknownEvents.map(\.gross_reason) == [.transferOrExchange, .missingSalePrice, .missingSalePrice, .missingProceedsQuote, .known, .negativeSalePrice], "Transfers, duplicate events, event-date quotes and negative declared prices must retain typed unknown reasons")
        let quotedCost = try bridge.readView(handle: unknownSession.handle, request: ViewRequest(view: "holdings", valuation: "market-value", currency: "EUR", date_to: "2000-01-06", account: "Assets:Stock")).holdings!.groups[0]
        try expect(quotedCost.market_valuation.complete_total?.exact == "75" && quotedCost.cost_valuation.complete_total?.exact == "25" && quotedCost.unrealized_gain?.exact == "50" && quotedCost.cost_valuation.quotes.first?.inverse == true, "Market and historical cost use their own dated local quotes in the selected currency")
        print("PASS: zero/unknown cost, nullable labels, source reduction reasons and separate cost quotes cross real Swift/C")
        let includeSession = try bridge.openLedger(path: root + "/testdata/fixtures/native/holdings-include.bean", semantics: "v3")
        defer { try? bridge.closeLedger(handle: includeSession.handle) }
        let includeHoldings = try bridge.readView(handle: includeSession.handle, request: ViewRequest(view: "holdings", valuation: "market-value", currency: "USD", account: "Assets:Stock")).holdings!
        let includeGroup = includeHoldings.groups[0]
        let includeQuote = includeGroup.market_valuation.quotes[0]
        let quoteSource = includeHoldings.source(for: includeQuote)
        try expect(includeGroup.unrealized_gain?.exact == "10" && quoteSource?.path.hasSuffix("holdings-include-prices.bean") == true && quoteSource?.line == 2 && quoteSource?.end_line == 2 && (quoteSource?.end_column ?? 0) > (quoteSource?.column ?? 0), "Quote FileID must resolve to the included source file and its full range")
        print("PASS: holding quotes preserve included-file provenance and source ranges across real Swift/C")
        let profiled = try bridge.openLedger(path: root + "/testdata/fixtures/semantics/lifo-explicit.bean", semantics: "v3")
        try expect(profiled.state.semantics_profile == "v3" && !profiled.state.evaluation_revision.isEmpty, "Explicit semantics must cross the real C ABI")
        let retained = try bridge.reloadLedger(handle: profiled.handle)
        try expect(retained.snapshot_id == profiled.state.snapshot_id && retained.semantics_profile == "v3", "Reload must preserve semantics")
        try bridge.closeLedger(handle: profiled.handle)
        for (name, gain) in [("lifo-implicit", "-100"), ("global-lifo", "-100"), ("strict-size", "-200"), ("average-explicit", "-400/3"), ("global-average", "-400/3")] {
            let session = try bridge.openLedger(path: root + "/testdata/fixtures/semantics/" + name + ".bean", semantics: "v3")
            let response = try bridge.readView(handle: session.handle, request: ViewRequest(view: "journal", window: "all"))
            let realized = response.journal?.transactions.flatMap { $0.postings }.first { $0.account == "Income:Gains" }
            try expect(realized?.amount?.exact == gain, "Explicit v3 booking/gain must cross the real Swift/C ABI: " + name)
            if name.contains("average") { try expect(session.state.active_extensions.contains("average-booking"), "AVERAGE must be explicit in the session identity") }
            try bridge.closeLedger(handle: session.handle)
        }
        print("PASS: v3 implicit-date LIFO, global booking, SIZE and AVERAGE exact gains")
        let migrationSession = try bridge.openLedger(path: root + "/testdata/fixtures/semantics/lifo-implicit.bean")
        defer { try? bridge.closeLedger(handle: migrationSession.handle) }
        let migration = try bridge.readView(handle: migrationSession.handle, request: ViewRequest(view: "migration", migration_details: true)).migration!
        let entryChange = migration.details!.first { $0.dimension == "entries" }!
        let presentation = entryChange.presentation(locale: "zh-CN")
        try expect(!presentation.key.contains("Rat") && !presentation.key.contains("Span") && !presentation.key.contains("{\""), "Migration detail title must expose ledger concepts rather than serialized implementation objects")
        try expect(presentation.detail?.contains("Assets:") == true && presentation.detail?.contains("USD") == true, "Local migration details must retain useful booked posting amounts")

        func read(_ path: String, _ view: String, currency: String = "") throws -> LedgerView {
            let session = try bridge.openLedger(path: path)
            defer { try? bridge.closeLedger(handle: session.handle) }
            return try bridge.readView(handle: session.handle, request: ViewRequest(view: view, currency: currency))
        }
        let empty = root + "/testdata/fixtures/native/empty.bean"
        let tree = try read(empty, "income_statement").tree!
        try expect(tree.root_nodes.isEmpty, "An account-only ledger must have an empty income statement")
        let journal = try read(empty, "journal").journal!
        try expect(journal.transactions.isEmpty && journal.total_count == 0,
                   "An account-only ledger must have an empty journal")
        print("PASS: empty reports decode through the real C ABI")

        let flows = try read(root + "/testdata/fixtures/native/cashflow.bean", "dashboard").trends!
        try expect(flows.monthly_cash_flows.count == 1, "Expected one monthly cash flow")
        let january = flows.monthly_cash_flows[0]
        try expect(january.income == 1000 && january.expenses == 300 && january.net == 700,
                   "Salary 1000 and spending 300 must display 1000/300/700; got \(january.income)/\(january.expenses)/\(january.net)")
        print("PASS: income, expenses and net preserve their economic meaning")

        let multiView = try read(root + "/testdata/fixtures/v3-parity/options.bean", "balance_sheet")
        let multi = multiView.summary
        try expect(multiView.tree?.currency == "" && multiView.tree?.root_nodes.first?.subtree_amounts["EUR"]?.exact == "2", "Unconfigured tables must retain all native currencies")
        let converted = try read(root + "/testdata/fixtures/v3-parity/options.bean", "balance_sheet", currency: "USD").tree!
        let partial = converted.root_nodes.first!.subtree_valuation!
        try expect(converted.root_nodes.first?.subtree_amounts["EUR"]?.exact == "2", "Missing quotes must not remove native balances")
        try expect(converted.valuation_status?.completeness == "partial" && partial.completeness == "partial" && partial.complete_total == nil && partial.valued_portion?.exact == "1", "Partial values cannot masquerade as complete totals")
        try expect(partial.missing.map { $0.commodity } == ["EUR"], "Missing currencies must be explicit")
        let nativeTree = try read(root + "/testdata/fixtures/v3-parity/options.bean", "balance_sheet", currency: "native").tree!
        try expect(nativeTree.root_nodes.first?.subtree_amounts["USD"]?.exact == "1" && nativeTree.root_nodes.first?.subtree_amounts["EUR"]?.exact == "2", "Explicit native-currency selection must retain both exact balances")
        try expect(multi.operating_currencies == ["USD", "EUR"], "Operating currencies must be separate tokens")
        let multiTrends = try read(root + "/testdata/fixtures/v3-parity/options.bean", "dashboard").trends!
        try expect(multiTrends.operating_currency.isEmpty && multiTrends.currency_series?.count == 2, "Native dashboard must preserve separate USD and EUR series")
        let pricedSession = try bridge.openLedger(path: root + "/testdata/fixtures/native/valuation.bean", semantics: "v3")
        defer { try? bridge.closeLedger(handle: pricedSession.handle) }
        let priced = try bridge.readView(handle: pricedSession.handle, request: ViewRequest(view: "balance_sheet", currency: "USD", date_to: "2000-01-04")).tree!
        let pricedAssets = priced.root_nodes.first!
        try expect(pricedAssets.subtree_valuation?.complete_total?.exact == "140" && priced.valuation_status?.completeness == "complete", "The cutoff quote EUR/USD=2 must value 100 USD + 20 EUR as 140, without future or JPY quotes")
        try expect(pricedAssets.subtree_valuation?.quotes.first?.date == "2000-01-03" && pricedAssets.subtree_valuation?.quotes.first?.rate.exact == "2", "Used quote date and exact rate must cross the real ABI")
        let beforeQuote = try bridge.readView(handle: pricedSession.handle, request: ViewRequest(view: "balance_sheet", currency: "USD", date_to: "2000-01-02")).tree!
        try expect(beforeQuote.root_nodes.first?.subtree_valuation?.complete_total == nil && beforeQuote.root_nodes.first?.subtree_valuation?.valued_portion?.exact == "100", "A later quote cannot fill an earlier cutoff")
        let zero = pricedAssets.children.first { $0.name == "Zero" }!
        let inactive = pricedAssets.children.first { $0.name == "Empty" }!
        try expect(zero.direct_amounts["USD"]?.exact == "0" && zero.direct_valuation?.complete_total?.exact == "0" && inactive.direct_valuation?.completeness == "empty", "Active zero balances are distinguishable from empty accounts")
        print("PASS: native and converted balances, null partial totals and as-of quotes cross the real C ABI")
        print("PASS: multiple operating currencies remain distinct")
        let convertedTrends = try read(root + "/testdata/fixtures/v3-parity/options.bean", "dashboard", currency: "USD").trends!
        try expect(convertedTrends.currency_series?.first?.net_worth_points.first?.valuation.complete_total == nil && convertedTrends.net_worth_points.isEmpty,
                   "Missing EUR/USD quotes must retain a null point, not a trustworthy zero net worth")
        let byCurrency = Dictionary(uniqueKeysWithValues: multiTrends.currency_series!.map { ($0.currency, $0.net_worth_points.first!.valuation.complete_total!.exact) })
        try expect(byCurrency == ["USD": "1", "EUR": "2"], "Native trends cannot add currencies")
        print("PASS: unavailable conversion cannot masquerade as zero")
        let historySession = try bridge.openLedger(path: root + "/testdata/fixtures/native/history.bean", semantics: "v3")
        defer { try? bridge.closeLedger(handle: historySession.handle) }
        let history = try bridge.readView(handle: historySession.handle, request: ViewRequest(view: "dashboard", valuation: "market-value", currency: "USD", date_to: "2000-03-31")).trends!
        let historical = history.currency_series!.first!.net_worth_points
        try expect(historical.count == 3 && historical[0].value == nil && historical[0].valuation.complete_total == nil && historical[0].valuation.valued_portion?.exact == "900", "Missing January quote must remain a gap, with only cash valued")
        try expect(historical[1].valuation.complete_total?.exact == "1100" && historical[1].valuation.quotes.first?.date == "2000-02-15", "Quiet February must revalue its own held inventory")
        try expect(historical[2].valuation.complete_total?.exact == "1100" && historical[2].valuation.quotes.isEmpty && historical[1].segment != historical[0].segment, "Liquidated stock is zero; no line may bridge January's gap")
        print("PASS: as-of inventory, quote dates and nullable historical gaps cross the real C ABI")
        let stockHistory = try bridge.readView(handle: historySession.handle, request: ViewRequest(view: "account", valuation: "market-value", currency: "USD", date_to: "2000-03-31", account: "Assets:Stock", scope: "direct")).account_detail!.history.first!.points
        try expect(stockHistory.count == 3 && stockHistory[0].value == nil && stockHistory[1].valuation.complete_total?.exact == "200" && stockHistory[1].valuation.quotes.first?.date == "2000-02-15" && stockHistory[2].valuation.complete_total?.exact == "0", "Account history must share dated quote/gap rules while excluding cash in another account")


        let lots = try read(root + "/testdata/fixtures/v3-parity/cost-basis.bean", "journal").journal!
        try expect(lots.transactions.first?.postings.first?.cost != nil,
                   "A costed purchase must retain its cost annotation")
        print("PASS: journal retains cost annotations")
        let shares = lots.transactions.flatMap { $0.postings }.first { $0.account == "Assets:Shares" }!
        try expect(shares.cost?.components.first?.number?.exact == "2" && shares.cost?.components.first?.currency == "USD",
                   "Cost components must retain exact cost and currency")
        let prices = try read(root + "/testdata/fixtures/v3-parity/at-price.bean", "journal").journal!.transactions.flatMap { $0.postings }.compactMap { $0.price }
        try expect(prices.contains { $0.total && $0.number.exact == "180" && $0.currency == "EUR" } && prices.contains { !$0.total && $0.number.exact == "0.9" },
                   "Unit and total prices must remain distinguishable")
        print("PASS: exact costs and unit/total prices retain their structure")

        let temporary = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: temporary, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: temporary) }
        let directions = temporary.appendingPathComponent("directions.bean")
        let original = try String(contentsOfFile: root + "/testdata/fixtures/native/cashflow.bean", encoding: .utf8)
        try (original + """

        2026-01-04 * "Refund"
          Expenses:Food -50 USD
          Assets:Bank 50 USD
        2026-01-05 * "Salary reversal"
          Income:Salary 100 USD
          Assets:Bank -100 USD

        """).write(to: directions, atomically: true, encoding: .utf8)
        let reversed = try read(directions.path, "dashboard").trends!
        try expect(reversed.income_amount.exact == "900" && reversed.expenses_amount.exact == "250" && reversed.net_amount.exact == "650", "Refunds and reversals must keep their economic direction")
        let withoutCurrency = temporary.appendingPathComponent("native.bean")
        try original.replacingOccurrences(of: "option \"operating_currency\" \"USD\"\n", with: "").write(to: withoutCurrency, atomically: true, encoding: .utf8)
        let unselected = try read(withoutCurrency.path, "dashboard").trends!
        try expect(unselected.currency_series?.count == 1 && unselected.currency_series?.first?.currency == "USD" && unselected.currency_series?.first?.mode == "native", "Native series derive their explicit currency from postings, without requiring operating currency")
        print("PASS: refunds, reversals and no-operating-currency behavior")

        let invalid = try read(root + "/testdata/fixtures/v3-parity/err-syntax.bean", "balance_sheet").summary
        try expect(!invalid.valid && invalid.diagnostics?.first?.code == "E-PARSE-STRING",
                   "Invalid ledgers must retain structured diagnostic codes")
        try expect(invalid.diagnostics?.first?.source.line == 1 && invalid.diagnostics?.first?.guide != nil,
                   "Diagnostics must include a source location and bundled repair guidance")
        let basic = try read(root + "/testdata/fixtures/v3-parity/txn-basic.bean", "balance_sheet").summary
        try expect(basic.transactions_count == 1 && basic.entries_count == 4,
                   "Transaction count must not include other directives")
        print("PASS: diagnostics and counters retain their meaning")
        try expect(january.net_amount?.exact == "700", "Chart labels must retain the exact net amount")
        print("PASS: chart labels have exact amounts independent of floating-point coordinates")

        let opened = try bridge.openLedger(path: root + "/testdata/fixtures/native/cashflow.bean")
        defer { try? bridge.closeLedger(handle: opened.handle) }
        let sessionJournal = try bridge.readView(handle: opened.handle, request: ViewRequest(view: "journal"))
        try expect(sessionJournal.journal?.total_count == 2, "Session journal must contain both transactions")
        let polled = try bridge.pollState(handle: opened.handle, after: opened.state.revision)
        try expect(!polled.changed && polled.state.build_count == 1, "Polling and reading must not rebuild")
        try bridge.closeLedger(handle: opened.handle)
        do {
            _ = try bridge.pollState(handle: opened.handle, after: 0)
            throw TestFailure(description: "A closed handle must be rejected")
        } catch BridgeError.remote(let code, _) {
            try expect(code == "session-closed", "Closed handles need a typed error")
        }
        print("PASS: session ownership and shared snapshots cross the real C ABI")

        let pagedSession = try bridge.openLedger(path: root + "/testdata/fixtures/native/cashflow.bean")
        defer { try? bridge.closeLedger(handle: pagedSession.handle) }
        let page = try bridge.readView(handle: pagedSession.handle, request: ViewRequest(view: "journal", limit: 1))
        try expect(page.journal?.transactions.count == 1 && page.journal?.total_count == 2 && page.journal?.has_more == true,
                   "The first journal page must be bounded and retain the full matching count")
        let filtered = try bridge.readView(handle: pagedSession.handle,
                                          request: ViewRequest(view: "journal", date_from: "2026-01-03", date_to: "2026-01-03", account: "Expenses", text: "FOOD"))
        try expect(filtered.journal?.total_count == 1 && filtered.journal?.transactions.first?.narration == "Food",
                   "Inclusive dates, account subtree and plain case-insensitive search must compose")
        print("PASS: filtered snapshot-bound journal pages decode through the C ABI")

        let accountSession = try bridge.openLedger(path: root + "/testdata/fixtures/native/account-detail.bean", semantics: "v3")
        defer { try? bridge.closeLedger(handle: accountSession.handle) }
        let accountView = try bridge.readView(handle: accountSession.handle, request: ViewRequest(view: "account", date_from: "2000-01-03", date_to: "2000-02-29", account: "Assets:Test", scope: "subtree", limit: 1))
        let detail = accountView.account_detail!
        try expect(detail.account == "Assets:Test" && detail.booking == "LIFO" && detail.closed == nil, "Account identity, effective global booking and nullable lifecycle must cross C/Swift")
        try expect(detail.direct_amounts["USD"]?.exact == "10" && detail.subtree_amounts["USD"]?.exact == "25" && detail.period_change["USD"]?.exact == "-5", "Account period must preserve opening inventory and both scopes")
        try expect(detail.history.first?.points.map { $0.valuation.complete_total?.exact } == ["25", "25"], "Account history must carry through quiet months")
        try expect(accountView.journal?.total_count == 1 && accountView.journal?.transactions.count == 1 && detail.sources.first?.source.line == 4, "Account journal and lifecycle source must be bounded and traceable")
        let childDetail = try bridge.readView(handle: accountSession.handle, request: ViewRequest(view: "account", account: "Assets:Test:Child", scope: "direct")).account_detail!
        try expect(childDetail.lifecycle == "closed" && childDetail.booking == "FIFO" && childDetail.direct_amounts["USD"]?.exact == "15", "Closing retains history and account booking overrides global")
        do {
            _ = try bridge.readView(handle: accountSession.handle, request: ViewRequest(view: "account", account: "Assets:Testi", scope: "subtree"))
            throw TestFailure(description: "Unknown account must fail")
        } catch BridgeError.remote(let code, _) {
            try expect(code == "account-not-found", "Unknown account must have a typed, actionable error")
        }
        var malformed = try JSONSerialization.jsonObject(with: JSONEncoder().encode(detail)) as! [String: Any]
        malformed.removeValue(forKey: "closed")
        do {
            let _: AccountDetailPayload = try BridgeClient.decodeResponse(JSONSerialization.data(withJSONObject: malformed), as: AccountDetailPayload.self)
            throw TestFailure(description: "Missing nullable account field must fail")
        } catch BridgeError.jsonDecodeError { }
        print("PASS: account scopes, quiet history, lifecycle, source, bounded journal and unknown target cross C/Swift")
        let incomeOnly = try bridge.readView(handle: pagedSession.handle, request: ViewRequest(view: "dashboard", date_from: "2026-01-02", date_to: "2026-01-02")).trends!
        let expenseOnly = try bridge.readView(handle: pagedSession.handle, request: ViewRequest(view: "dashboard", date_from: "2026-01-03", date_to: "2026-01-03")).trends!
        try expect(incomeOnly.income_amount.exact == "1000" && incomeOnly.expenses_amount.exact == "0" && incomeOnly.net_amount.exact == "1000", "An income-only period has no invented expenses")
        try expect(expenseOnly.income_amount.exact == "0" && expenseOnly.expenses_amount.exact == "300" && expenseOnly.net_amount.exact == "-300", "An expense-only period has no invented income")
        print("PASS: inclusive one-sided periods preserve exact cash-flow totals")

        let directory = root + "/testdata/fixtures/v3-parity"
        let names = try FileManager.default.contentsOfDirectory(atPath: directory).filter { $0.hasSuffix(".bean") }.sorted()
        for name in names {
            let session = try bridge.openLedger(path: directory + "/" + name)
            defer { try? bridge.closeLedger(handle: session.handle) }
            for route in ["balance_sheet", "income_statement", "journal", "dashboard"] {
                let response = try bridge.readView(handle: session.handle, request: ViewRequest(view: route))
                try expect(response.state.has_snapshot == response.summary.valid,
                           "Initial snapshot and validity must agree for fixture \(name)")
            }
        }
        print("PASS: all \(names.count) parity fixtures decode on all four native routes")
        try verifyQueries(root: root, bridge: bridge)
    }
    static func verifyQueries(root: String, bridge: BridgeClient) throws {
        let session = try bridge.openLedger(path: root + "/testdata/fixtures/query/default-source.bean", semantics: "v3")
        defer { try? bridge.closeLedger(handle: session.handle) }
        let first = try bridge.executeQuery(handle: session.handle, request: QueryRequest(snapshot_id: session.state.snapshot_id, request_id: 1,
            text: "SELECT 1 AS v, 2 AS v, '', NULL, TRUE, date, number, weight, position, meta('pmeta') LIMIT 1"))
        let execution = first.execution!
        try expect(first.error == nil && first.context.evaluation_revision == session.state.evaluation_revision && execution.returned_rows == 1 && !execution.truncated,
                   "Query must carry its captured accounting identity and exact row count")
        let cells = execution.result.rows[0]
        guard case .integer("1") = cells[0].value, case .integer("2") = cells[1].value,
              case .text("") = cells[2].value, case .null = cells[3].value,
              case .boolean(true) = cells[4].value, case .decimal(let number) = cells[6].value,
              case .position(let position) = cells[8].value else { throw TestFailure(description: "Typed query cells lost their actual values") }
        try expect(number.exact == "-2" && position.cost == nil && execution.result.columns[0].name == execution.result.columns[1].name && execution.result.columns[0].id != execution.result.columns[1].id,
                   "Duplicate titles, decimal and nullable position cost survive real C")
        var malformed = try JSONSerialization.jsonObject(with: JSONEncoder().encode(first)) as! [String: Any]
        for fault in ["row-width", "duplicate-id", "cell-type", "null-value", "missing-nullable", "row-count"] {
            var altered = malformed
            if fault == "missing-nullable" { altered.removeValue(forKey: "error") }
            else {
                var data = altered["execution"] as! [String: Any]
                var result = data["result"] as! [String: Any]
                var columns = result["columns"] as! [[String: Any]]
                var rows = result["rows"] as! [[[String: Any]]]
                switch fault {
                case "row-width": rows[0].removeLast()
                case "duplicate-id": columns[1]["id"] = columns[0]["id"]
                case "cell-type": rows[0][0]["value"] = true
                case "null-value": rows[0][3]["value"] = ""
                default: data["returned_rows"] = 0
                }
                result["columns"] = columns; result["rows"] = rows; data["result"] = result; altered["execution"] = data
            }
            let raw: [String: Any] = ["protocol_version": 2, "data": altered]
            do {
                let _: QueryResponse = try BridgeClient.decodeResponse(JSONSerialization.data(withJSONObject: raw), as: QueryResponse.self)
                throw TestFailure(description: "Malformed query \(fault) accepted")
            } catch BridgeError.jsonDecodeError { }
        }
        let failed = try bridge.executeQuery(handle: session.handle, request: QueryRequest(snapshot_id: session.state.snapshot_id, request_id: 2, text: "SELECT unknown(number) LIMIT 0"))
        try expect(failed.execution == nil && failed.error?.code == "unknown-function" && failed.error?.range == nil && failed.context.request_id == 2,
                   "A failed query is separate from its previous success, even on LIMIT 0")
        malformed = try JSONSerialization.jsonObject(with: JSONEncoder().encode(failed)) as! [String: Any]
        var failure = malformed["error"] as! [String: Any]; failure.removeValue(forKey: "range"); malformed["error"] = failure
        do {
            let _: QueryResponse = try BridgeClient.decodeResponse(JSONSerialization.data(withJSONObject: ["protocol_version": 2, "data": malformed]), as: QueryResponse.self)
            throw TestFailure(description: "Missing nullable query error range accepted")
        } catch BridgeError.jsonDecodeError { }
        try bridge.cancelQuery(handle: session.handle, requestID: 3)
        let cancelled = try bridge.executeQuery(handle: session.handle, request: QueryRequest(snapshot_id: session.state.snapshot_id, request_id: 3, text: "SELECT number"))
        try expect(cancelled.execution == nil && cancelled.error?.code == "cancelled", "Cancellation before C dispatch must be remembered")
        let compound = try bridge.openLedger(path: root + "/testdata/fixtures/query/compound-casts.bean", semantics: "v3")
        defer { try? bridge.closeLedger(handle: compound.handle) }
        let inventory = try bridge.executeQuery(handle: compound.handle, request: QueryRequest(snapshot_id: compound.state.snapshot_id, request_id: 1, text: "SELECT sum(position) WHERE account = 'Assets:Precision'"))
        guard case .inventory(let positions) = inventory.execution!.result.rows[0][0].value else { throw TestFailure(description: "Inventory was flattened into text") }
        try expect(positions.count == 1 && positions[0].units.number.exact == "1" && positions[0].cost?.number.exact == "10" && positions[0].cost?.label == "a\"b", "Full costs and raw labels survive query C/Swift")
        let empty = try bridge.executeQuery(handle: compound.handle, request: QueryRequest(snapshot_id: compound.state.snapshot_id, request_id: 2, text: "SELECT account, position LIMIT 0"))
        try expect(empty.execution?.result.columns.count == 2 && empty.execution?.result.rows.isEmpty == true && empty.execution?.returned_rows == 0,
                   "Successful zero rows retain their declared schema")
        print("PASS: query snapshot identity, duplicate titles, typed scalar/compound/NULL, zero rows, failed executions and queued cancellation cross real C/Swift")
        print("PASS: malformed query widths, column IDs, cell types, NULL shape, nullable fields and row metadata are rejected")
    }

}
